// Package xui is a read-only 3X-UI client API adapter; it does not issue credentials.
package xui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const maxResponseBytes = 2 << 20

// ClientRecord contains fields confirmed on the deployed 3X-UI API.
// UUID and SubID are sensitive and must never be logged or sent to clients.
type ClientRecord struct {
	ID         int64  `json:"id"`
	Email      string `json:"email"`
	SubID      string `json:"subId"`
	UUID       string `json:"uuid"`
	LimitHWID  int64  `json:"limitHwid"`
	TotalGB    int64  `json:"totalGB"`
	ExpiryTime int64  `json:"expiryTime"`
	Enable     bool   `json:"enable"`
}
type envelope struct {
	Success bool            `json:"success"`
	Obj     json.RawMessage `json:"obj"`
}

// Client uses HTTPS, GET, and a server-only bearer credential.
type Client struct {
	base       *url.URL
	token      string
	httpClient *http.Client
}

// NewClientFromHeader consumes the existing root-protected curl header file.
func NewClientFromHeader(baseURL, headerPath string) (*Client, error) {
	b, err := os.ReadFile(headerPath)
	if err != nil {
		return nil, fmt.Errorf("read API credential: %w", err)
	}
	const prefix = "Authorization: Bearer "
	header := strings.TrimSpace(string(b))
	if !strings.HasPrefix(header, prefix) {
		return nil, errors.New("API credential format invalid")
	}
	token := strings.TrimPrefix(header, prefix)
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return nil, errors.New("API credential format invalid")
	}
	return NewClient(baseURL, token)
}

// baseURL ends at the secret panel base prefix, before /panel/api.
// The caller must keep it and the token outside GitHub and Android.
func NewClient(baseURL, token string) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("panel base URL must be HTTPS without userinfo, query or fragment")
	}
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return nil, errors.New("empty or invalid API credential")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath = ""
	return &Client{base: u, token: token, httpClient: &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}}, nil
}

func (c *Client) get(ctx context.Context, suffix string, output any) error {
	u := *c.base
	u.Path = c.base.Path + "/panel/api/clients/" + suffix
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return errors.New("construct panel request")
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	res, err := c.httpClient.Do(req)
	if err != nil {
		return errors.New("panel request failed") // Do not leak token/base path.
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("panel returned HTTP %d", res.StatusCode)
	}
	var reply envelope
	if err := json.NewDecoder(io.LimitReader(res.Body, maxResponseBytes+1)).Decode(&reply); err != nil {
		return errors.New("invalid panel JSON response")
	}
	if !reply.Success || len(reply.Obj) == 0 || string(reply.Obj) == "null" {
		return errors.New("panel returned unsuccessful or empty client data")
	}
	if err := json.Unmarshal(reply.Obj, output); err != nil {
		return errors.New("unrecognized panel client structure")
	}
	return nil
}

// List reads all current clients. It is not an enrollment authorization check.
func (c *Client) List(ctx context.Context) ([]ClientRecord, error) {
	var clients []ClientRecord
	if err := c.get(ctx, "list", &clients); err != nil {
		return nil, err
	}
	if clients == nil {
		return nil, errors.New("client list was not an array")
	}
	return clients, nil
}

// GetByEmail reads a nested obj.client record, without authenticating the caller.
func (c *Client) GetByEmail(ctx context.Context, email string) (ClientRecord, error) {
	if email == "" || strings.ContainsAny(email, "/\\?#\x00\r\n") {
		return ClientRecord{}, errors.New("invalid client identifier")
	}
	var payload struct {
		Client *ClientRecord `json:"client"`
	}
	if err := c.get(ctx, "get/"+email, &payload); err != nil {
		return ClientRecord{}, err
	}
	if payload.Client == nil {
		return ClientRecord{}, errors.New("missing client record")
	}
	if payload.Client.Email != email {
		return ClientRecord{}, errors.New("client identifier mismatch")
	}
	return *payload.Client, nil
}
