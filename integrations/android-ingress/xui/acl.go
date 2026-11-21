package xui

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
)

// AddDeviceToACL produces a complete staged server JSON config with exactly
// one additional device fingerprint authorized for an existing service.
// This function does NOT write disk, reload Bridge, or modify running Xray.
// The owner must atomically install and activate the staged config separately.
func AddDeviceToACL(serverJSON []byte, leafFingerprint, service string) ([]byte, error) {
    return AddDeviceToACLServices(serverJSON, leafFingerprint, []string{service})
}

// AddDeviceToACLServices only adds authorized services when every requested
// service already has a configured server-side backend.
func AddDeviceToACLServices(serverJSON []byte, leafFingerprint string, services []string) ([]byte, error) {
	raw, err := decodeServerConfig(serverJSON)
	if err != nil {
		return nil, err
	}
	fp, err := hex.DecodeString(leafFingerprint)
	if err != nil || len(fp) != 32 || hex.EncodeToString(fp) != leafFingerprint {
		return nil, errors.New("invalid SHA256 leaf fingerprint")
	}
	var mode string
	if err := json.Unmarshal(raw["mode"], &mode); err != nil || mode != "server" {
		return nil, errors.New("ACL update requires Bridge server configuration")
	}
	var routes map[string]json.RawMessage
	if err := json.Unmarshal(raw["routes"], &routes); err != nil {
		return nil, errors.New("service route unavailable")
	}
    if len(services) == 0 || len(services) > 256 {
        return nil, errors.New("invalid Bridge service list")
    }
    seen := make(map[string]bool)
    for _, name := range services {
        if name == "" || seen[name] {
            return nil, errors.New("duplicate or empty Bridge service")
        }
        seen[name] = true
        var backends []json.RawMessage
        if err := json.Unmarshal(routes[name], &backends); err != nil || len(backends) == 0 {
            return nil, errors.New("service route unavailable")
        }
    }
	var acl map[string][]string
	if err := json.Unmarshal(raw["authorized"], &acl); err != nil || acl == nil {
		return nil, errors.New("missing authorized ACL")
	}
	if _, exists := acl[leafFingerprint]; exists {
		return nil, errors.New("fingerprint already present")
	}
	acl[leafFingerprint] = append([]string(nil), services...)
	updated, err := json.Marshal(acl)
	if err != nil {
		return nil, errors.New("cannot marshal ACL")
	}
	raw["authorized"] = updated
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return nil, errors.New("cannot marshal staged server configuration")
	}
	return append(out, '\n'), nil
}

func decodeServerConfig(data []byte) (map[string]json.RawMessage, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	var raw map[string]json.RawMessage
	if err := d.Decode(&raw); err != nil || raw == nil {
		return nil, errors.New("invalid server configuration")
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return nil, errors.New("multiple or invalid server config documents")
	}
	return raw, nil
}
