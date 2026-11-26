package xui

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestJournalPolicyACKRequiresFreshExactDigest(t *testing.T) {
	digest := strings.Repeat("a", 64)
	after := time.Unix(1700000000, 0).UTC()
	message := "M6_POLICY_RELOAD=PASS digest=" + digest
	for _, tc := range []struct {
		name  string
		at    time.Time
		entry string
		ok    bool
	}{
		{"fresh", after.Add(time.Millisecond), message, true},
		{"with Go prefix", after.Add(time.Millisecond), "2026/11/26 12:00:01 " + message, true},
		{"stale", after.Add(-time.Millisecond), message, false},
		{"suffix", after, message + " extra", false},
		{"different digest", after, message[:len(message)-1] + "b", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			row, err := json.Marshal(map[string]string{
				"__REALTIME_TIMESTAMP": strconv.FormatInt(tc.at.UnixMicro(), 10),
				"MESSAGE":              tc.entry,
			})
			if err != nil {
				t.Fatal(err)
			}
			if journalPolicyACK(row, after, digest) != tc.ok {
				t.Fatal("unexpected policy ACK result")
			}
		})
	}
}
