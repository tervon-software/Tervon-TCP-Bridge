package xui

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestSubscriptionEligibility(t *testing.T) {
	now := time.UnixMilli(1800000000000)
	base := ClientRecord{
		ID: 12, Email: "test-client", Enable: true,
		ExpiryTime: now.Add(time.Hour).UnixMilli(), TotalGB: 100,
	}
	validUsage := &UsageSnapshot{Up: 40, Down: 59}
	testCases := []struct {
		name string
		mutate func(*ClientRecord)
		usage *UsageSnapshot
		wantError string
	}{
		{"active below quota", nil, validUsage, ""},
		{"missing client id", func(c *ClientRecord){ c.ID = 0 }, validUsage, "identity"},
		{"missing client email", func(c *ClientRecord){ c.Email = "" }, validUsage, "identity"},
		{"disabled", func(c *ClientRecord){ c.Enable = false }, validUsage, "disabled"},
		{"expired", func(c *ClientRecord){ c.ExpiryTime = now.UnixMilli() }, validUsage, "expired"},
		{"pending first-use activation", func(c *ClientRecord){ c.ExpiryTime = -1 }, validUsage, "activation"},
		{"invalid quota", func(c *ClientRecord){ c.TotalGB = -1 }, validUsage, "limits"},
		{"invalid device cap", func(c *ClientRecord){ c.LimitHWID = -1 }, validUsage, "limits"},
		{"missing usage", nil, nil, "unavailable"},
		{"quota exactly reached", nil, &UsageSnapshot{Up: 40, Down: 60}, "exhausted"},
		{"quota exceeded", nil, &UsageSnapshot{Up: 41, Down: 60}, "exhausted"},
		{"negative traffic", nil, &UsageSnapshot{Up: -1, Down: 0}, "invalid usage"},
		{"counter overflow", nil, &UsageSnapshot{Up: math.MaxInt64, Down: 1}, "invalid usage"},
		{"unlimited traffic and expiry", func(c *ClientRecord){ c.TotalGB = 0; c.ExpiryTime = 0 }, nil, ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := base
			if tc.mutate != nil { tc.mutate(&c) }
			err := CheckSubscriptionEligibility(c, tc.usage, now)
			if tc.wantError == "" {
				if err != nil { t.Fatalf("unexpected failure: %v", err) }
			} else if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("error = %v; want %s", err, tc.wantError)
			}
		})
	}
}

func TestSubscriptionEligibilityDoesNotAuthorizeDevice(t *testing.T) {
	// An eligible subscription is necessary but never sufficient to issue mTLS.
	// Callers MUST enforce device limits and prove possession of a separate
	// enrollment credential before issuing a certificate.
	c := ClientRecord{ID: 1, Email: "test-client", Enable: true, LimitHWID: 1}
	if err := CheckSubscriptionEligibility(c, nil, time.Now()); err != nil {
		t.Fatal(err)
	}
}
