package xui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

type panelSettings struct {
	Type          string `json:"type"`
	BaseURL       string `json:"base_url"`
	APICredential string `json:"credential_file"`
}

// ConfiguredPanelInbounds reloads the protected installer settings on each
// observation. An installation without panel settings can still enroll.
func ConfiguredPanelInbounds(path string) RouteEvidenceSource {
	if path == "" {
		return nil
	}
	return func(ctx context.Context) ([]PanelInbound, error) {
		client, err := configuredPanelClient(path)
		if err != nil {
			return nil, err
		}
		return client.SelectedBridgeInbounds(ctx)
	}
}

func configuredPanelClient(path string) (*Client, error) {
	settings, err := readProtectedPanelFile(path)
	if err != nil {
		return nil, errors.New("protected panel settings unavailable")
	}
	var config panelSettings
	decoder := json.NewDecoder(bytes.NewReader(settings))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&config) != nil || decoder.Decode(new(json.RawMessage)) != io.EOF {
		return nil, errors.New("invalid panel settings")
	}
	switch config.Type {
	case "3x-ui":
	case "remnawave":
		return nil, errors.New("Remnawave inbound adapter unavailable")
	default:
		return nil, errors.New("unknown panel type")
	}
	if !filepath.IsAbs(config.APICredential) {
		return nil, errors.New("API credential path must be absolute")
	}
	credential, err := readProtectedPanelFile(config.APICredential)
	if err != nil {
		return nil, errors.New("protected API credential unavailable")
	}
	return newClientFromHeader(config.BaseURL, credential)
}

// Inspect the opened file so an atomic replacement cannot invalidate the
// permissions check. Refuse symlinks and avoid blocking on special files.
func readProtectedPanelFile(path string) ([]byte, error) {
	const limit = 16 << 10
	unavailable := errors.New("protected file unavailable")
	if !filepath.IsAbs(path) {
		return nil, unavailable
	}
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, unavailable
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return nil, unavailable
	}
	contents, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || len(contents) > limit {
		return nil, unavailable
	}
	return contents, nil
}
