package login

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

func ReadCert(privateKeyData io.Reader) (*rsa.PrivateKey, error) {
	pemBytes, err := io.ReadAll(privateKeyData)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key data: %w", err)
	}

	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("invalid private key data")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("could not parse RSA private key: %w", err)
	}

	return privateKey, nil
}

func ReadTestCert(t *testing.T) (*rsa.PrivateKey, error) {
	const data = `-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----`
	reader := strings.NewReader(data)
	return ReadCert(reader)
}
