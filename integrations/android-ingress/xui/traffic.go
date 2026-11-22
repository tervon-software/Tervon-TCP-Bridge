package xui

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// GetUsageByEmail reads per-client counters from the documented 3X-UI
// /panel/api/clients/traffic/{email} endpoint. The row ID is a traffic-row
// ID and must not be compared with ClientRecord.ID.
//
// Missing counters are a hard error: missing must never become zero usage.
// This method is not an Android authentication or enrollment endpoint.
func validateEmail(email string) error {
	if email == "" || strings.IndexFunc(email, func(r rune) bool {
		return r < 0x20 || r == 0x7f || strings.ContainsRune("/\\?#", r)
	}) >= 0 {
		return errors.New("invalid client identifier")
	}
	return nil
}

func (c *Client) GetUsageByEmail(ctx context.Context, email string) (UsageSnapshot, error) {
	if err := validateEmail(email); err != nil {
		return UsageSnapshot{}, err
	}
	var raw json.RawMessage
	if err := c.get(ctx, "traffic/"+email, &raw); err != nil {
		return UsageSnapshot{}, err
	}
	var counters struct {
		Email *string `json:"email"`
		Up    *int64  `json:"up"`
		Down  *int64  `json:"down"`
	}
	if err := json.Unmarshal(raw, &counters); err != nil ||
		counters.Email == nil || *counters.Email != email ||
		counters.Up == nil || counters.Down == nil {
		return UsageSnapshot{}, errors.New("missing or mismatched client traffic counters")
	}
	return UsageSnapshot{Up: *counters.Up, Down: *counters.Down}, nil
}

// CheckEligibilityByEmail reads one current client and, for a limited quota,
// the matching traffic counters before checking subscription status.
// This is only an entitlement precondition; issuing a certificate additionally
// requires independent enrollment authentication, device quota and ACL checks.
func (c *Client) CheckEligibilityByEmail(ctx context.Context, email string, now time.Time) error {
	record, err := c.GetByEmail(ctx, email)
	if err != nil {
		return err
	}
	if record.TotalGB == 0 {
		return CheckSubscriptionEligibility(record, nil, now)
	}
	// Even for malformed or disabled records, don't grant anything: the
	// policy below rejects them. Request traffic only when still eligible
	// for a quota check.
	if err := CheckSubscriptionEligibility(record, &UsageSnapshot{}, now); err != nil {
		return err
	}
	usage, err := c.GetUsageByEmail(ctx, record.Email)
	if err != nil {
		return err
	}
	return CheckSubscriptionEligibility(record, &usage, now)
}
