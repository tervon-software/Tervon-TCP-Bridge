package xui

import (
	"encoding/json"
	"errors"
)

// FullPolicyConfigDigest produces the SAME combined Routes+ACL generation
// digest as the v1.0.2 core, without modifying any route or ACL entry.
// It is used for both the forward ACK and the rollback ACK.
func FullPolicyConfigDigest(serverJSON []byte) (string, error) {
	raw, err := decodeServerConfig(serverJSON)
	if err != nil {
		return "", err
	}
	var mode string
	if err := json.Unmarshal(raw["mode"], &mode); err != nil || mode != "server" {
		return "", errors.New("invalid Bridge server mode")
	}
	var routes map[string][]PolicyBackend
	if err := json.Unmarshal(raw["routes"], &routes); err != nil || routes == nil {
		return "", errors.New("invalid Bridge policy routes")
	}
	var authorized map[string][]string
	if err := json.Unmarshal(raw["authorized"], &authorized); err != nil || len(authorized) == 0 {
		return "", errors.New("invalid Bridge policy ACL")
	}
	return PolicyGenerationDigest(routes, authorized)
}
