package xui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// InboundSnapshot is a redacted projection of one 3x-ui inbound. It contains no
// client credentials, user identifiers, or VPN payload.
type InboundSnapshot struct {
	ID       int64
	Port     int
	Protocol string
	Enabled  bool
	Network  string
	Security string
}

// ListInbounds reads the panel's slim inbound projection. It does not change
// Xray, Bridge routes, or any client record. Callers must separately validate
// the live Xray configuration before allowing a backend.
func (c *Client) ListInbounds(ctx context.Context) ([]InboundSnapshot, error) {
	if c == nil || c.base == nil || c.httpClient == nil {
		return nil, errors.New("panel client unavailable")
	}
	u := *c.base
	u.Path = c.base.Path + "/panel/api/inbounds/list/slim"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.New("construct inbound request")
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.New("inbound API unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("inbound API HTTP %d", res.StatusCode)
	}
	var reply envelope
	dec := json.NewDecoder(io.LimitReader(res.Body, maxResponseBytes+1))
	if err := dec.Decode(&reply); err != nil {
		return nil, errors.New("invalid inbound API JSON")
	}
	if !reply.Success || len(reply.Obj) == 0 || string(reply.Obj) == "null" {
		return nil, errors.New("inbound API rejected request")
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(reply.Obj, &entries); err != nil || entries == nil {
		return nil, errors.New("inbound API list malformed")
	}
	if len(entries) > 4096 {
		return nil, errors.New("inbound API list too large")
	}
	out := make([]InboundSnapshot, 0, len(entries))
	seen := make(map[int64]bool, len(entries))
	for _, entry := range entries {
		var raw struct {
			ID             *int64          `json:"id"`
			Port           json.RawMessage `json:"port"`
			Protocol       string          `json:"protocol"`
			Enable         *bool           `json:"enable"`
			StreamSettings json.RawMessage `json:"streamSettings"`
		}
		if err := json.Unmarshal(entry, &raw); err != nil || raw.ID == nil || *raw.ID <= 0 || raw.Enable == nil {
			return nil, errors.New("invalid inbound identity or enable flag")
		}
		if seen[*raw.ID] {
			return nil, errors.New("duplicate inbound identity")
		}
		seen[*raw.ID] = true
		var port int
		if err := json.Unmarshal(raw.Port, &port); err != nil || port < 1 || port > 65535 {
			return nil, errors.New("invalid inbound port")
		}
		if strings.TrimSpace(raw.Protocol) == "" || len(raw.Protocol) > 64 {
			return nil, errors.New("invalid inbound protocol")
		}
		settings := raw.StreamSettings
		if len(settings) == 0 || string(settings) == "null" {
			return nil, errors.New("inbound stream settings missing")
		}
		if settings[0] == '"' {
			var encoded string
			if err := json.Unmarshal(settings, &encoded); err != nil {
				return nil, errors.New("inbound stream settings malformed")
			}
			settings = []byte(encoded)
		}
		var transport struct {
			Network  string `json:"network"`
			Security string `json:"security"`
		}
		if err := json.Unmarshal(settings, &transport); err != nil {
			return nil, errors.New("inbound stream settings malformed")
		}
		if transport.Network == "" {
			transport.Network = "tcp"
		}
		if transport.Security == "" {
			transport.Security = "none"
		}
		if len(transport.Network) > 64 || len(transport.Security) > 64 {
			return nil, errors.New("inbound transport invalid")
		}
		out = append(out, InboundSnapshot{
			ID: *raw.ID, Port: port, Protocol: strings.ToLower(raw.Protocol),
			Enabled: *raw.Enable, Network: strings.ToLower(transport.Network),
			Security: strings.ToLower(transport.Security),
		})
	}
	return out, nil
}

// InboundPortKey is intentionally a plain validated port, not a backend
// address; the route publisher must derive the address from trusted settings.
func (i InboundSnapshot) InboundPortKey() string { return strconv.Itoa(i.Port) }

// SelectedBridgeInbounds adapts the existing read-only 3x-ui API directly
// to the server-side route coordinator. It never changes VPN or Android state.
func (c *Client) SelectedBridgeInbounds(ctx context.Context) ([]PanelInbound,error) {
 entries,err:=c.ListInbounds(ctx)
 if err!=nil{return nil,err}
 out:=make([]PanelInbound,0,len(entries))
 for _,in:=range entries{
  out=append(out,PanelInbound{Port:in.Port,Protocol:in.Protocol,Network:in.Network,Enabled:in.Enabled})
 }
 return FilterBridgePanelInbounds(out),nil
}
