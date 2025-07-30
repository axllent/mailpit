// Package snakeoil provides functionality to generate a temporary self-signed certificates
// for testing purposes. It generates a public and private key pair, stores them in the
// OS's temporary directory, returning the paths to these files.
package snakeoil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/tools"
)

var keys = make(map[string]KeyPair)

// KeyPair holds the public and private key paths for a self-signed certificate.
type KeyPair struct {
	Public  string
	Private string
}

// Certificates returns all configured self-signed certificates in use,
// used for file deletion on exit.
func Certificates() map[string]KeyPair {
	return keys
}

// Public returns the path to a generated PEM-encoded RSA public key.
func Public(str string) string {
	domains, key, err := parse(str)
	if err != nil {
		logger.Log().Errorf("[tls] failed to parse domains: %v", err)
		return ""
	}

	if pair, ok := keys[key]; ok {
		return pair.Public
	}

	private, public, err := generate(domains)
	if err != nil {
		logger.Log().Errorf("[tls] failed to generate public certificate: %v", err)
		return ""
	}

	keys[key] = KeyPair{
		Public:  public,
		Private: private,
	}

	return public
}

// Private returns the path to a generated PEM-encoded RSA private key.
func Private(str string) string {
	domains, key, err := parse(str)
	if err != nil {
		logger.Log().Errorf("[tls] failed to parse domains: %v", err)
		return ""
	}

	if pair, ok := keys[key]; ok {
		return pair.Private
	}

	private, public, err := generate(domains)
	if err != nil {
		logger.Log().Errorf("[tls] failed to generate public certificate: %v", err)
		return ""
	}

	keys[key] = KeyPair{
		Public:  public,
		Private: private,
	}

	return private
}

// Parse takes the original string input, removes the "sans:" prefix,
// splits the result into individual domains, and returns a slice of unique domains,
// along with a unique key that is a comma-separated list of these domains.
func parse(str string) ([]string, string, error) {
	// remove "sans:" prefix
	str = str[5:]
	var domains []string
	// split the string by commas and trim whitespace
	for domain := range strings.SplitSeq(str, ",") {
		domain = strings.ToLower(strings.TrimSpace(domain))
		if domain != "" && !tools.InArray(domain, domains) {
			domains = append(domains, domain)
		}
	}

	if len(domains) == 0 {
		return domains, "", errors.New("no valid domains provided")
	}

	// generate sha256 hash of the domains to create a unique key
	hasher := sha256.New()
	hasher.Write([]byte(strings.Join(domains, ",")))
	key := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

	return domains, key, nil
}

// Generate a new self-signed certificate and return a public & private key paths.
func generate(domains []string) (string, string, error) {
	logger.Log().Infof("[tls] generating temp self-signed certificate for: %s", strings.Join(domains, ","))
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return "", "", err
	}

	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	// PEM encoding of private key
	keyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: keyBytes,
		},
	)

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	// create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(0),
		Subject: pkix.Name{
			CommonName:   domains[0],
			Organization: []string{"Mailpit self-signed certificate"},
		},
		DNSNames:              domains,
		SignatureAlgorithm:    x509.SHA256WithRSA,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement | x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}

	// create certificate using template
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return "", "", err

	}

	// PEM encoding of certificate
	certPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: derBytes,
		},
	)

	// Store the paths to the generated keys
	priv, err := os.CreateTemp("", ".mailpit-*-private.pem")
	if err != nil {
		return "", "", err
	}

	if _, err := priv.Write(keyPEM); err != nil {
		return "", "", err
	}

	if err := priv.Close(); err != nil {
		return "", "", err
	}

	pub, err := os.CreateTemp("", ".mailpit-*-public.pem")
	if err != nil {
		return "", "", err
	}

	if _, err := pub.Write(certPem); err != nil {
		return "", "", err
	}

	if err := pub.Close(); err != nil {
		return "", "", err
	}

	return priv.Name(), pub.Name(), nil
}
