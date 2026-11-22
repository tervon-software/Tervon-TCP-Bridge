package xui

import (
	"errors"
	"math"
	"time"
)

// UsageSnapshot is provided by an authoritative 3X-UI traffic endpoint.
// The caller must not manufacture a zero-usage snapshot when traffic is unknown.
type UsageSnapshot struct {
	Up   int64
	Down int64
}

// CheckSubscriptionEligibility evaluates only the panel's client subscription
// state. It does NOT authenticate an Android caller, authorize a device,
// validate inbound attachment, or authorize certificate issuance.
//
// In the current 3X-UI API, 0 for ExpiryTime or TotalGB denotes unlimited;
// a negative expiry waits for first-use activation and is rejected here.
// When traffic is limited, a real usage snapshot is mandatory.
func CheckSubscriptionEligibility(c ClientRecord, usage *UsageSnapshot, now time.Time) error {
	if c.ID <= 0 || c.Email == "" {
		return errors.New("missing client identity")
	}
	if !c.Enable {
		return errors.New("client disabled")
	}
	if c.ExpiryTime < 0 {
		return errors.New("client expiry pending activation")
	}
	if c.ExpiryTime > 0 && now.UnixMilli() >= c.ExpiryTime {
		return errors.New("client expired")
	}
	if c.TotalGB < 0 || c.LimitHWID < 0 {
		return errors.New("invalid subscription limits")
	}
	if c.TotalGB == 0 {
		return nil
	}
	if usage == nil {
		return errors.New("usage unavailable for limited subscription")
	}
	if usage.Up < 0 || usage.Down < 0 || usage.Up > math.MaxInt64-usage.Down {
		return errors.New("invalid usage counters")
	}
	if usage.Up+usage.Down >= c.TotalGB {
		return errors.New("traffic quota exhausted")
	}
	return nil
}
