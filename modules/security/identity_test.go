package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"
)

func generateTestCert(isCA bool, parent *x509.Certificate, parentKey *rsa.PrivateKey) (*x509.Certificate, []byte, *rsa.PrivateKey) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"Sovereign State Authority"}},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  isCA,
	}

	if parent == nil {
		parent = template
		parentKey = key
	}

	derBytes, _ := x509.CreateCertificate(rand.Reader, template, parent, &key.PublicKey, parentKey)
	cert, _ := x509.ParseCertificate(derBytes)
	return cert, derBytes, key
}

func TestIdentityGate(t *testing.T) {
	// 1. Generate Root Authority
	rootCert, _, rootKey := generateTestCert(true, nil, nil)
	gate := NewIdentityGate(rootCert)

	// 2. Generate Valid Node Cert (Signed by Root)
	_, validNodeDER, _ := generateTestCert(false, rootCert, rootKey)

	// 3. Generate Rogue Node Cert (Self-Signed / Different Root)
	_, rogueNodeDER, _ := generateTestCert(false, nil, nil)

	// TEST 1: Reject Anonymous
	if err := gate.VerifyPeerCertificate(nil, nil); err == nil {
		t.Fatal("SECURITY FAULT: Identity gate accepted anonymous connection")
	}

	// TEST 2: Reject Unauthorized/Rogue
	if err := gate.VerifyPeerCertificate([][]byte{rogueNodeDER}, nil); err == nil {
		t.Fatal("SECURITY FAULT: Identity gate accepted unauthorized counterfeit certificate")
	}

	// TEST 3: Accept Authorized
	if err := gate.VerifyPeerCertificate([][]byte{validNodeDER}, nil); err != nil {
		t.Fatalf("FAULT: Identity gate rejected valid institutional certificate: %v", err)
	}
}
