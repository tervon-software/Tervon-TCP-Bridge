package xui

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// XrayRuntimeSource reads the live Xray projection and checks backend TCP
// reachability from the configured Decapsulator's network namespace.
type XrayRuntimeSource struct {
	ProbePath      string
	OwnerUnit      string
	ExecutableName string
	BackendAlias   string
	Decapsulator   DockerPolicyConfirmer
}

func (s XrayRuntimeSource) Evidence(ctx context.Context) ([]EffectiveInbound, error) {
	if !filepath.IsAbs(s.ProbePath) || s.OwnerUnit == "" || s.ExecutableName == "" || s.BackendAlias == "" {
		return nil, errors.New("live Xray settings incomplete")
	}
	ctx, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()

	pid, err := s.containerPID(ctx)
	if err != nil {
		return nil, err
	}
	output, err := exec.CommandContext(ctx, "python3", "-EsB", s.ProbePath,
		s.OwnerUnit, s.ExecutableName, pid, s.BackendAlias).Output()
	if err != nil {
		return nil, errors.New("live Xray or Decapsulator backend evidence unavailable")
	}
	evidence, err := decodeLiveXrayEvidence(output)
	if err != nil {
		return nil, err
	}
	current, err := s.containerPID(ctx)
	if err != nil || current != pid {
		return nil, errors.New("Decapsulator changed during Xray observation")
	}
	return evidence, nil
}

func (s XrayRuntimeSource) containerPID(ctx context.Context) (string, error) {
	if err := s.Decapsulator.verifyIdentity(ctx); err != nil {
		return "", err
	}
	raw, err := dockerCommand(ctx, "inspect", "--format", "{{.State.Pid}}", s.Decapsulator.ContainerID)
	pid := strings.TrimSpace(string(raw))
	n, parseErr := strconv.Atoi(pid)
	if err != nil || parseErr != nil || n <= 1 {
		return "", errors.New("Decapsulator process unavailable")
	}
	return strconv.Itoa(n), nil
}

func decodeLiveXrayEvidence(output []byte) ([]EffectiveInbound, error) {
	var reply struct {
		Address  string
		Inbounds []struct {
			Port     int
			Protocol string
			Network  string
		}
	}
	invalid := errors.New("invalid live Xray projection")
	if err := json.Unmarshal(output, &reply); err != nil || net.ParseIP(reply.Address) == nil ||
		reply.Inbounds == nil || len(reply.Inbounds) > 4096 {
		return nil, invalid
	}
	out := make([]EffectiveInbound, 0, len(reply.Inbounds))
	seen := make(map[int]bool, len(reply.Inbounds))
	for _, inbound := range reply.Inbounds {
		if inbound.Port < 1 || inbound.Port > 65535 || seen[inbound.Port] ||
			!bridgeEligibleProtocol(inbound.Protocol) || !bridgeEligibleTCPNetwork(inbound.Network) {
			return nil, invalid
		}
		seen[inbound.Port] = true
		port := strconv.Itoa(inbound.Port)
		out = append(out, EffectiveInbound{
			Port: inbound.Port, Protocol: inbound.Protocol, Network: canonicalInboundNetwork(inbound.Network),
			Service: port, Address: net.JoinHostPort(reply.Address, port),
		})
	}
	return out, nil
}
