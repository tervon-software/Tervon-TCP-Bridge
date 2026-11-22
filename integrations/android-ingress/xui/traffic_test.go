package xui

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestTrafficLookupAndEligibility(t *testing.T) {
	now := time.UnixMilli(1800000000000)
	cases := []struct {
		name, traffic string
		quota int64
		wantErr bool
		wantTraffic bool
	}{
		{"within_quota", `{"email":"Diag","up":10,"down":89}`, 100, false, true},
		{"quota_exhausted", `{"email":"Diag","up":10,"down":90}`, 100, true, true},
		{"missing_up", `{"email":"Diag","down":0}`, 100, true, true},
		{"missing_down", `{"email":"Diag","up":0}`, 100, true, true},
		{"wrong_email", `{"email":"Other","up":0,"down":0}`, 100, true, true},
		{"empty_traffic", `null`, 100, true, true},
		{"malformed_traffic", `{"email":"Diag","up":"0","down":0}`, 100, true, true},
		{"unlimited_quota_skips_usage", `null`, 0, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var trafficCalled atomic.Bool
			s := server(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer test-secret" || r.Method != http.MethodGet {
					t.Errorf("missing expected authorization or wrong method")
					w.WriteHeader(403); return
				}
				switch r.URL.Path {
				case "/private-panel/panel/api/clients/get/Diag":
					_, _ = fmt.Fprintf(w, `{"success":true,"obj":{"client":{"id":10,"email":"Diag","enable":true,"expiryTime":%d,"totalGB":%d}}}`, now.Add(time.Hour).UnixMilli(), tc.quota)
				case "/private-panel/panel/api/clients/traffic/Diag":
					trafficCalled.Store(true)
					_, _ = fmt.Fprintf(w, `{"success":true,"obj":%s}`, tc.traffic)
				default:
					t.Errorf("unexpected path: %s", r.URL.Path); http.NotFound(w,r)
				}
			})
			c := adapter(t,s)
			err := c.CheckEligibilityByEmail(context.Background(), "Diag",now)
			if (err != nil) != tc.wantErr { t.Fatalf("err %v; wantErr %v",err,tc.wantErr) }
			if trafficCalled.Load() != tc.wantTraffic { t.Fatalf("traffic called %v, want %v",trafficCalled.Load(),tc.wantTraffic) }
		})
	}
}

func TestTrafficLookupFailClosedOnPanelStatus(t *testing.T) {
	s := server(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})
	c := adapter(t,s)
	if _,err := c.GetUsageByEmail(context.Background(),"Diag"); err == nil {
		t.Fatal("missing traffic must be denied")
	}
}

func TestTrafficLookupRejectsInvalidIdentifier(t *testing.T) {
	c,err := NewClient("https://example.org/private-panel","secret")
	if err != nil { t.Fatal(err) }
	for _,name := range []string{"","../admin","a/b","a?b", "a#b","a\\b","a\nb"} {
		_,err := c.GetUsageByEmail(context.Background(),name)
		if err == nil || !strings.Contains(err.Error(),"invalid client identifier") {
			t.Errorf("invalid identifier %q accepted or caused network request: %v",name,err)
		}
	}
}
