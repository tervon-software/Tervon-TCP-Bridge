package xui

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const policyRegistrarFixture = `{
  "mode":"server", "listen":"0.0.0.0:8443",
  "routes":{"vpn":[{"Name":"backend","Address":"127.0.0.1:33000"}]},
  "authorized":{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa":["vpn"]}
}`

func TestFullPolicyConfigDigestMatchesCoreContract(t *testing.T) {
	got, err := FullPolicyConfigDigest([]byte(policyRegistrarFixture))
	if err != nil {
		t.Fatal(err)
	}
	want, err := PolicyGenerationDigest(
		map[string][]PolicyBackend{"vpn": {{Name: "backend", Address: "127.0.0.1:33000"}}},
		map[string][]string{strings.Repeat("a", 64): {"vpn"}},
	)
	if err != nil || got != want {
		t.Fatalf("policy digest mismatch: %v", err)
	}
	for _, bad := range []string{
		`{"mode":"client","routes":{"vpn":[{"Address":"127.0.0.1:33000"}]},"authorized":{"a":["vpn"]}}`,
		`{"mode":"server","authorized":{"a":["vpn"]}}`,
		`{"mode":"server","routes":{"vpn":[{"Address":"broken"}]},"authorized":{"a":["vpn"]}}`,
	} {
		if _, err := FullPolicyConfigDigest([]byte(bad)); err == nil {
			t.Fatal("invalid policy accepted")
		}
	}
}

func TestV102EnrollmentConfirmsFullPolicyAndRollsBack(t *testing.T) {
	for _, reject := range []bool{false, true} {
		t.Run(map[bool]string{false: "accept", true: "reject"}[reject], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "server.json")
			before := []byte(policyRegistrarFixture)
			if err := os.WriteFile(path, before, 0600); err != nil {
				t.Fatal(err)
			}
			var calls int
			a := &HotACLActivator{
				Path: path, Service: "vpn", Digest: FullPolicyConfigDigest,
				Confirm: func(_ context.Context, expected string) error {
					calls++
					current, err := os.ReadFile(path)
					if err != nil {
						return err
					}
					got, err := FullPolicyConfigDigest(current)
					if err != nil || got != expected {
						return errors.New("wrong full policy generation")
					}
					if reject && calls == 1 {
						return errors.New("runtime ACK refused")
					}
					return nil
				},
			}
			err := a.Activate(context.Background(), strings.Repeat("b", 64))
			after, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if reject {
				if err == nil || calls != 2 || !bytes.Equal(after, before) {
					t.Fatalf("rollback failed: %v, calls=%d", err, calls)
				}
			} else {
				if err != nil || calls != 1 || !bytes.Contains(after, []byte(strings.Repeat("b", 64))) {
					t.Fatalf("confirmation failed: %v, calls=%d", err, calls)
				}
			}
		})
	}
}
