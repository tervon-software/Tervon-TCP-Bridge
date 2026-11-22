package xui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var dockerIDPattern = regexp.MustCompile("^[0-9a-f]{12,64}$")
var digestPattern = regexp.MustCompile("^[0-9a-f]{64}$")

// DockerACLConfirmer uses only the already running, exact Android ingress
// container. It never restarts a process or touches a Linux RU/FI Bridge.
// The single persistent config bind mount is required to be read-only
// inside the container. The root-host registrar writes it atomically.
type DockerACLConfirmer struct {
	ContainerID string
	HostConfigDirectory string
}

type dockerInspectState struct {
	Config struct {
		Image string `json:"Image"`
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
	State struct{Running bool `json:"Running"`} `json:"State"`
	Mounts []struct {
		Type string `json:"Type"`
		Source string `json:"Source"`
		Destination string `json:"Destination"`
		RW bool `json:"RW"`
	} `json:"Mounts"`
}

func dockerCommand(ctx context.Context, args ...string) ([]byte,error) {
	c:=exec.CommandContext(ctx,"docker",args...)
	out,err:=c.CombinedOutput()
	if err!=nil{return nil,fmt.Errorf("docker %s failed",args[0])}
	if len(out)>2<<20{return nil,errors.New("docker log output too large")}
	return out,nil
}

func (d DockerACLConfirmer) validateTarget(ctx context.Context) error {
	if !dockerIDPattern.MatchString(d.ContainerID) {
		return errors.New("invalid Android ingress container ID")
	}
	dir:=d.HostConfigDirectory
	if !filepath.IsAbs(dir) || filepath.Clean(dir)!=dir || dir=="/"{
		return errors.New("invalid expected Android ingress config directory")
	}
	out,err:=dockerCommand(ctx,"inspect",d.ContainerID)
	if err!=nil{return err}
	var states []dockerInspectState
	if json.Unmarshal(out,&states)!=nil || len(states)!=1 ||
		states[0].Config.Image!="tervon-bridge-android-ingress:m6-runtime-acl" ||
		states[0].Config.Labels["org.tervon.bridge.hot_acl"]!="m6-v1" ||
		!states[0].State.Running {
		return errors.New("unexpected Android ingress container")
	}
	for _,m:=range states[0].Mounts {
		if m.Type=="bind" && m.Source==dir && m.Destination=="/config" && !m.RW{
			return nil
		}
	}
	return errors.New("Android ingress configuration mount mismatch")
}

// Confirm sends SIGHUP, then checks the exact digest emitted AFTER an
// in-process ACL swap. Docker signaling success alone is never sufficient.
// No operator secret, CA private key or VPN subscription is sent via Docker.
func (d DockerACLConfirmer) Confirm(ctx context.Context,digest string) error {
	if !digestPattern.MatchString(digest){return errors.New("invalid ACL generation digest")}
	if err:=d.validateTarget(ctx);err!=nil{return err}
	start:=time.Now().Add(-30*time.Millisecond).UTC().Format(time.RFC3339Nano)
	if _,err:=dockerCommand(ctx,"kill","--signal=HUP",d.ContainerID);err!=nil{return err}
	ticker:=time.NewTicker(125*time.Millisecond)
	defer ticker.Stop()
	pattern:="M6_ACL_RELOAD=PASS digest="+digest
	for {
		out,err:=dockerCommand(ctx,"logs","--since",start,d.ContainerID)
		if err!=nil{return err}
		for _,line:=range strings.Split(string(out),"\n"){
			if strings.Contains(line,pattern){return nil}
		}
		select {
		case <-ctx.Done():
			return errors.New("Android ingress ACL reload acknowledgment timed out")
		case <-ticker.C:
		}
	}
}
