package xui

import "testing"

func TestDecodeLiveXrayEvidence(t *testing.T) {
	data := []byte(`{"Address":"2001:db8::1","Inbounds":[{"Port":55535,"Protocol":"vless","Network":"splithttp"}]}`)
	got, err := decodeLiveXrayEvidence(data)
	want := EffectiveInbound{Port: 55535, Protocol: "vless", Network: "xhttp", Service: "55535", Address: "[2001:db8::1]:55535"}
	if err != nil || len(got) != 1 || got[0] != want {
		t.Fatalf("got %+v, %v", got, err)
	}
	for _, input := range []string{
		`{"Address":"host.example","Inbounds":[]}`,
		`{"Address":"192.0.2.1","Inbounds":[{"Port":0,"Protocol":"vless","Network":"tcp"}]}`,
		`{"Address":"192.0.2.1","Inbounds":[{"Port":123,"Protocol":"vless","Network":"quic"}]}`,
		`{"Address":"192.0.2.1","Inbounds":[{"Port":123,"Protocol":"vless","Network":"tcp"},{"Port":123,"Protocol":"trojan","Network":"tcp"}]}`,
	} {
		if _, err := decodeLiveXrayEvidence([]byte(input)); err == nil {
			t.Fatal("accepted invalid evidence")
		}
	}
}
