package xui

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SystemdPolicyConfirmer binds policy changes to an installed native core.
type SystemdPolicyConfirmer struct {
	Unit       string
	Binary     string
	SHA256     string
	ConfigPath string
}

var bridgeUnitPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*\.service$`)
var invocationPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

type nativeProcess struct {
	pid, invocation string
}

func (s SystemdPolicyConfirmer) process(ctx context.Context) (nativeProcess, error) {
	var process nativeProcess
	if !bridgeUnitPattern.MatchString(s.Unit) || !policyDigestPattern.MatchString(s.SHA256) ||
		!filepath.IsAbs(s.Binary) || !filepath.IsAbs(s.ConfigPath) {
		return process, errors.New("native Bridge identity not configured")
	}
	out, err := exec.CommandContext(ctx, "systemctl", "show", "-p", "MainPID", "-p", "InvocationID",
		"-p", "ActiveState", "--", s.Unit).Output()
	if err != nil {
		return process, errors.New("native Bridge service unavailable")
	}
	fields := make(map[string]string)
	for _, line := range strings.Split(string(out), "\n") {
		if key, val, ok := strings.Cut(line, "="); ok {
			fields[key] = val
		}
	}
	pid, err := strconv.Atoi(fields["MainPID"])
	if err != nil || pid <= 1 || fields["ActiveState"] != "active" || !invocationPattern.MatchString(fields["InvocationID"]) {
		return process, errors.New("native Bridge process unavailable")
	}
	proc := filepath.Join("/proc", strconv.Itoa(pid))
	exe := filepath.Join(proc, "exe")
	actual, err := os.Readlink(exe)
	if err != nil || actual != s.Binary {
		return process, errors.New("native Bridge executable mismatch")
	}
	cmdline, err := os.ReadFile(filepath.Join(proc, "cmdline"))
	expected := s.Binary + "\x00-config\x00" + s.ConfigPath + "\x00"
	if err != nil || string(cmdline) != expected {
		return process, errors.New("native Bridge configuration path mismatch")
	}
	file, err := os.Open(exe)
	if err != nil {
		return process, errors.New("cannot verify native Bridge executable")
	}
	defer file.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil || hex.EncodeToString(digest.Sum(nil)) != s.SHA256 {
		return process, errors.New("native Bridge executable checksum mismatch")
	}
	return nativeProcess{strconv.Itoa(pid), fields["InvocationID"]}, nil
}

func (s SystemdPolicyConfirmer) PID(ctx context.Context) (string, error) {
	process, err := s.process(ctx)
	return process.pid, err
}

func journalPolicyACK(output []byte, after time.Time, digest string) bool {
	for _, line := range bytes.Split(output, []byte("\n")) {
		var entry struct {
			Time    string `json:"__REALTIME_TIMESTAMP"`
			Message string `json:"MESSAGE"`
		}
		if json.Unmarshal(line, &entry) != nil {
			continue
		}
		usec, err := strconv.ParseInt(entry.Time, 10, 64)
		if err != nil {
			continue
		}
		stamped := time.UnixMicro(usec).UTC().Format(time.RFC3339Nano) + " " + entry.Message
		if policyACKInLogs([]byte(stamped), after, digest) {
			return true
		}
	}
	return false
}

func (s SystemdPolicyConfirmer) Confirm(ctx context.Context, digest string) error {
	if ctx == nil || !policyDigestPattern.MatchString(digest) {
		return errors.New("invalid policy confirmation request")
	}
	process, err := s.process(ctx)
	if err != nil {
		return err
	}
	after := time.Now().UTC()
	if err := exec.CommandContext(ctx, "systemctl", "kill", "--kill-who=main", "--signal=HUP", "--", s.Unit).Run(); err != nil {
		return errors.New("cannot signal native Bridge service")
	}

	ticker := time.NewTicker(125 * time.Millisecond)
	defer ticker.Stop()
	since := fmt.Sprintf("@%d", after.Unix())
	for {
		out, err := exec.CommandContext(ctx, "journalctl", "--no-pager", "--output=json", "--since", since,
			"_SYSTEMD_UNIT="+s.Unit, "_SYSTEMD_INVOCATION_ID="+process.invocation, "_PID="+process.pid).Output()
		if err != nil {
			return errors.New("native Bridge journal unavailable")
		}
		if journalPolicyACK(out, after, digest) {
			current, err := s.process(ctx)
			if err != nil || current != process {
				return errors.New("native Bridge changed during policy confirmation")
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return errors.New("native Bridge policy acknowledgment unavailable")
		case <-ticker.C:
		}
	}
}
