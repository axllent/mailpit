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
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/tools"
)

var (
	keysMu sync.Mutex
	keys   = make(map[string]KeyPair)
)

// KeyPair holds the public and private key paths for a self-signed certificate.
type KeyPair struct {
	Public  string
	Private string
}

// Certificates returns all configured self-signed certificates in use,
// used for file deletion on exit.
func Certificates() map[string]KeyPair {
	keysMu.Lock()
	defer keysMu.Unlock()

	copied := make(map[string]KeyPair, len(keys))
	for k, v := range keys {
		copied[k] = v
	}

	return copied
}

// Public returns the path to a generated PEM-encoded certificate, or an empty string on error.
func Public(str string) string {
	pair, err := keyPair(str)
	if err != nil {
		logger.Log().Errorf("[tls] failed to generate public certificate: %v", err)
		return ""
	}

	return pair.Public
}

// Private returns the path to a generated PEM-encoded RSA private key, or an empty string on error.
func Private(str string) string {
	pair, err := keyPair(str)
	if err != nil {
		logger.Log().Errorf("[tls] failed to generate private key: %v", err)
		return ""
	}

	return pair.Private
}

// keyPair returns the key pair for the "sans:" string, generating it the first time.
func keyPair(str string) (KeyPair, error) {
	domains, key, err := parse(str)
	if err != nil {
		return KeyPair{}, err
	}

	keysMu.Lock()
	defer keysMu.Unlock()

	if pair, ok := keys[key]; ok {
		return pair, nil
	}

	private, public, err := generate(domains)
	if err != nil {
		return KeyPair{}, err
	}

	keys[key] = KeyPair{Public: public, Private: private}

	return keys[key], nil
}

// Parse takes the original string input, removes the "sans:" prefix,
// splits the result into individual domains, and returns a slice of unique domains,
// along with a unique key that is a comma-separated list of these domains.
func parse(str string) ([]string, string, error) {
	str, ok := strings.CutPrefix(str, "sans:")
	if !ok {
		return nil, "", errors.New(`value must start with "sans:"`)
	}

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

// splitSANs separates IP addresses from DNS names, as they are different certificate fields.
func splitSANs(domains []string) ([]string, []net.IP) {
	var names []string
	var ips []net.IP

	for _, d := range domains {
		// an IPv6 address may be given in brackets
		if ip := net.ParseIP(strings.Trim(d, "[]")); ip != nil {
			ips = append(ips, ip)
		} else {
			names = append(names, d)
		}
	}

	return names, ips
}

// Generate a new self-signed certificate and return the private & public key paths.
func generate(domains []string) (string, string, error) {
	logger.Log().Infof("[tls] generating temp self-signed certificate for: %s", strings.Join(domains, ","))
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return "", "", err
	}

	keyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		},
	)

	// a random positive serial number (RFC 5280), so regenerated certificates never share one
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
	if err != nil {
		return "", "", err
	}
	serial.Add(serial, big.NewInt(1))

	names, ips := splitSANs(domains)

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   domains[0],
			Organization: []string{"Mailpit self-signed certificate"},
		},
		DNSNames:              names,
		IPAddresses:           ips,
		SignatureAlgorithm:    x509.SHA256WithRSA,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement | x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return "", "", err
	}

	certPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: derBytes,
		},
	)

	privPath, err := writeTemp(".mailpit-*-private.pem", keyPEM)
	if err != nil {
		return "", "", err
	}

	pubPath, err := writeTemp(".mailpit-*-public.pem", certPEM)
	if err != nil {
		// do not leave the private key behind without its certificate
		_ = os.Remove(privPath)
		return "", "", err
	}

	return privPath, pubPath, nil
}

// createTemp is a variable so that the tests can make it fail.
var createTemp = os.CreateTemp

// writeTemp writes the data to a new temporary file (mode 0600) and returns its path. The file is
// removed again if it cannot be written.
func writeTemp(pattern string, data []byte) (string, error) {
	f, err := createTemp("", pattern)
	if err != nil {
		return "", err
	}

	_, werr := f.Write(data)
	cerr := f.Close()

	if err := errors.Join(werr, cerr); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}

	return f.Name(), nil
}
