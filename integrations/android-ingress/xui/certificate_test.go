package xui

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"math/big"
	"testing"
	"time"
)

func testCertificateInputs(t *testing.T, now time.Time) ([]byte, []byte, *ecdsa.PrivateKey) {
	t.Helper()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil { t.Fatal(err) }
	caTemplate := &x509.Certificate{
		SerialNumber:big.NewInt(1), Subject:pkix.Name{CommonName:"Test client CA"},
		NotBefore:now.Add(-time.Hour),NotAfter:now.Add(48*time.Hour),
		KeyUsage:x509.KeyUsageCertSign, BasicConstraintsValid:true, IsCA:true,
	}
	caDER,err := x509.CreateCertificate(rand.Reader,caTemplate,caTemplate,&caKey.PublicKey,caKey)
	if err != nil { t.Fatal(err) }
	deviceKey,err:=ecdsa.GenerateKey(elliptic.P256(),rand.Reader)
	if err!=nil {t.Fatal(err)}
	csrDER,err:=x509.CreateCertificateRequest(rand.Reader,&x509.CertificateRequest{
		Subject:pkix.Name{CommonName:"Android device"},
	},deviceKey)
	if err!=nil {t.Fatal(err)}
	return pem.EncodeToMemory(&pem.Block{Type:"CERTIFICATE REQUEST",Bytes:csrDER}),
		pem.EncodeToMemory(&pem.Block{Type:"CERTIFICATE",Bytes:caDER}),caKey
}

func TestSignDeviceCSR(t *testing.T) {
	now:=time.Now().UTC().Truncate(time.Second)
	csr,caPEM,caKey:=testCertificateInputs(t,now)
	issued,err:=SignDeviceCSR(csr,caPEM,caKey,now,12*time.Hour)
	if err!=nil {t.Fatal(err)}
	block,_:=pem.Decode(issued.PEM)
	if block==nil {t.Fatal("missing certificate PEM")}
	leaf,err:=x509.ParseCertificate(block.Bytes)
	if err!=nil {t.Fatal(err)}
	caBlock,_:=pem.Decode(caPEM)
	ca,_:=x509.ParseCertificate(caBlock.Bytes)
	pool:=x509.NewCertPool()
	pool.AddCert(ca)
	if _,err=leaf.Verify(x509.VerifyOptions{
		Roots:pool,CurrentTime:now,
		KeyUsages:[]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	});err!=nil {t.Fatal(err)}
	if leaf.IsCA || len(leaf.ExtKeyUsage)!=1 || leaf.ExtKeyUsage[0]!=x509.ExtKeyUsageClientAuth {
		t.Fatal("certificate granted wrong purpose")
	}
	sum:=sha256.Sum256(block.Bytes)
	if issued.Fingerprint!=hex.EncodeToString(sum[:]) {t.Fatal("incorrect SHA256 fingerprint")}
	if leaf.NotAfter.After(now.Add(12*time.Hour)) {t.Fatal("certificate exceeds lifetime")}
}

func TestSignDeviceCSRRejectsBadProofAndCA(t *testing.T) {
	now:=time.Now().UTC().Truncate(time.Second)
	csr,caPEM,caKey:=testCertificateInputs(t,now)
	otherCSR,otherCA,otherKey:=testCertificateInputs(t,now)
	_ = otherCSR
	for _,tc:=range []struct{
		name string
		csr,ca []byte
		key *ecdsa.PrivateKey
		lifetime time.Duration
	}{
		{"empty CSR",nil,caPEM,caKey,time.Hour},
		{"garbled CSR",[]byte("no PEM"),caPEM,caKey,time.Hour},
		{"wrong CA private key",csr,caPEM,otherKey,time.Hour},
		{"wrong CA data",csr,[]byte("bad CA"),caKey,time.Hour},
		{"expired CA",csr,otherCA,caKey,time.Hour},
		{"zero lifetime",csr,caPEM,caKey,0},
		{"lifetime >24h",csr,caPEM,caKey,25*time.Hour},
	}{
		t.Run(tc.name,func(t *testing.T){
			if _,err:=SignDeviceCSR(tc.csr,tc.ca,tc.key,now,tc.lifetime);err==nil {
				t.Fatal("unauthorized or malformed CSR was signed")
			}
		})
	}
}
