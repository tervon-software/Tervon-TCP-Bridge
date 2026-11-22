package xui

import (
    "context"
    "crypto/x509"
    "encoding/pem"
    "testing"
    "time"
)

func TestBridgeIssuerRejectsInvalidCSRAndCAWithoutSubscriptionChecks(t *testing.T) {
    now := time.Now().UTC().Truncate(time.Second)
    csr,caPEM,caKey := testCertificateInputs(t,now)
    cases := []struct {name string; csr,ca []byte}{
        {"missing CSR",nil,caPEM},
        {"malformed CSR",[]byte("bad"),caPEM},
        {"missing CA",csr,nil},
        {"malformed CA",csr,[]byte("bad")},
    }
    for _,tc:=range cases {
        t.Run(tc.name,func(t *testing.T){
            issued,err := IssueBridgeClientCertificate(context.Background(),tc.csr,tc.ca,caKey,now)
            if err == nil || len(issued.PEM)!=0 || issued.Fingerprint!="" {
                t.Fatal("signed malformed certificate request")
            }
        })
    }
    cancelled,cancel := context.WithCancel(context.Background())
    cancel()
    if _,err:=IssueBridgeClientCertificate(cancelled,csr,caPEM,caKey,now);err==nil {
        t.Fatal("cancelled request accepted")
    }
}

func TestBridgeIssuerSignsExistingP256CSRWithoutVPNSubscription(t *testing.T) {
    now := time.Now().UTC().Truncate(time.Second)
    csr,caPEM,caKey := testCertificateInputs(t,now)
    issued,err := IssueBridgeClientCertificate(context.Background(),csr,caPEM,caKey,now)
    if err != nil { t.Fatal(err) }
    block,rest := pem.Decode(issued.PEM)
    if block == nil || block.Type!="CERTIFICATE" || len(rest)!=0 {t.Fatal("invalid leaf PEM")}
    leaf,err:=x509.ParseCertificate(block.Bytes)
    if err!=nil {t.Fatal(err)}
    if !leaf.NotAfter.After(now) || leaf.NotAfter.After(now.Add(24*time.Hour)){
        t.Fatal("wrong leaf validity")
    }
    if len(leaf.ExtKeyUsage)!=1 || leaf.ExtKeyUsage[0]!=x509.ExtKeyUsageClientAuth {
        t.Fatal("leaf not clientAuth only")
    }
    if issued.Fingerprint=="" {t.Fatal("missing Bridge ACL fingerprint")}
}
