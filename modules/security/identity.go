package security

import (
	"crypto/x509"
	"errors"
	"fmt"
)

// IdentityGate acts as the protocol-layer KYC interceptor for the P2P transport
type IdentityGate struct {
	RegulatoryRootCAs *x509.CertPool
}

func NewIdentityGate(rootCA *x509.Certificate) *IdentityGate {
	pool := x509.NewCertPool()
	pool.AddCert(rootCA)
	return &IdentityGate{
		RegulatoryRootCAs: pool,
	}
}

// VerifyPeerCertificate matches the tls.Config signature for seamless mTLS integration
func (g *IdentityGate) VerifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	if len(rawCerts) == 0 {
		return errors.New("SECURITY_FAULT: Anonymous connections strictly forbidden")
	}

	cert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		return fmt.Errorf("SECURITY_FAULT: Malformed X.509 certificate: %v", err)
	}

	opts := x509.VerifyOptions{
		Roots:     g.RegulatoryRootCAs,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	if _, err := cert.Verify(opts); err != nil {
		return fmt.Errorf("JURISDICTION_VIOLATION: Peer holds unauthorized or counterfeit credentials: %v", err)
	}

	return nil
}
