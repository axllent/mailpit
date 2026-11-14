package snakeoil

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/axllent/mailpit/internal/logger"
)

func TestMain(m *testing.M) {
	logger.NoLogging = true

	code := m.Run()

	for _, pair := range Certificates() {
		_ = os.Remove(pair.Public)
		_ = os.Remove(pair.Private)
	}

	os.Exit(code)
}

func readCert(t *testing.T, path string) *x509.Certificate {
	t.Helper()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(b)
	if block == nil {
		t.Fatal("not PEM")
	}
	c, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}

	return c
}

func TestParse(t *testing.T) {
	// exploit: values without the prefix used to panic (slice out of range)
	for _, in := range []string{"", "san", "sans", "localhost", "SANS:localhost"} {
		if _, _, err := parse(in); err == nil {
			t.Errorf("parse(%q) did not fail", in)
		}
	}
	for _, in := range []string{"sans:", "sans: , ,"} {
		if _, _, err := parse(in); err == nil {
			t.Errorf("parse(%q) did not fail", in)
		}
	}

	// control
	d, k, err := parse("sans: Localhost, a.example ,LOCALHOST")
	if err != nil || k == "" || !slices.Equal(d, []string{"localhost", "a.example"}) {
		t.Fatalf("got %v %q %v", d, k, err)
	}
	if _, k2, _ := parse("sans:a.example,localhost"); k2 == k {
		t.Error("a different order should not share a key")
	}
}

func FuzzParse(f *testing.F) {
	for _, s := range []string{"sans:localhost", "sans:a,B , a", "sans:", "", "san", "sans:\x00"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		d, k, err := parse(s)
		if err == nil && (len(d) == 0 || k == "") {
			t.Fatal("empty result without error")
		}
		if err != nil && (len(d) != 0 || k != "") {
			t.Fatal("result with an error")
		}
		for _, x := range d {
			if x == "" || x != strings.ToLower(strings.TrimSpace(x)) {
				t.Fatalf("bad domain %q", x)
			}
		}
	})
}

func TestSplitSANs(t *testing.T) {
	names, ips := splitSANs([]string{"localhost", "127.0.0.1", "::1", "[::2]", "1.2.3.4.example", "300.1.1.1"})
	if !slices.Equal(names, []string{"localhost", "1.2.3.4.example", "300.1.1.1"}) {
		t.Errorf("names: %v", names)
	}
	if len(ips) != 3 || ips[0].String() != "127.0.0.1" || ips[1].String() != "::1" || ips[2].String() != "::2" {
		t.Errorf("ips: %v", ips)
	}
}

func TestGeneratedCertificate(t *testing.T) {
	str := "sans:localhost,127.0.0.1,::1"

	pub, priv := Public(str), Private(str)
	if pub == "" || priv == "" {
		t.Fatal("no certificate generated")
	}
	if pub2 := Public(str); pub2 != pub {
		t.Error("the certificate is not reused for the same domains")
	}

	c := readCert(t, pub)

	// control: names verify
	if err := c.VerifyHostname("localhost"); err != nil {
		t.Errorf("localhost: %v", err)
	}
	// exploit: IP addresses used to fail, as they were in the DNS names
	for _, ip := range []string{"127.0.0.1", "::1"} {
		if err := c.VerifyHostname(ip); err != nil {
			t.Errorf("%s: %v", ip, err)
		}
	}
	if err := c.VerifyHostname("other.example"); err == nil {
		t.Error("an unlisted name verified")
	}

	if c.SerialNumber.Sign() <= 0 {
		t.Errorf("serial number %v is not positive", c.SerialNumber)
	}

	if _, err := tls.LoadX509KeyPair(pub, priv); err != nil {
		t.Errorf("certificate and key do not match: %v", err)
	}

	st, err := os.Stat(priv)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Errorf("private key mode %v", st.Mode().Perm())
	}
	if len(Certificates()) == 0 {
		t.Error("certificates not registered for removal")
	}
}

// Regenerated certificates used to all have serial number 0, so identical issuer and serial.
func TestSerialNumbersDiffer(t *testing.T) {
	var serials []string
	for range 2 {
		priv, pub, err := generate([]string{"localhost"})
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(priv)
		defer os.Remove(pub)
		serials = append(serials, readCert(t, pub).SerialNumber.String())
	}
	if serials[0] == serials[1] {
		t.Fatalf("two certificates share serial %s", serials[0])
	}
}

func TestInvalidInputGivesEmptyPath(t *testing.T) {
	for _, in := range []string{"", "sans:", "sans:badé.example"} {
		if got := Public(in); got != "" {
			t.Errorf("Public(%q) = %q", in, got)
		}
		if got := Private(in); got != "" {
			t.Errorf("Private(%q) = %q", in, got)
		}
	}
}

// A failure after the first file is written must not leave it behind.
func TestNoFilesLeftOnFailure(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)

	// control: success leaves exactly the two files
	priv, pub, err := generate([]string{"localhost"})
	if err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 2 {
		t.Fatalf("%d files after success", len(entries))
	}
	_ = os.Remove(priv)
	_ = os.Remove(pub)

	// exploit: the second file cannot be created, after the private key was written
	calls := 0
	createTemp = func(d, pattern string) (*os.File, error) {
		calls++
		if calls == 2 {
			return nil, errors.New("disk full")
		}
		return os.CreateTemp(d, pattern)
	}
	defer func() { createTemp = os.CreateTemp }()

	if priv, pub, err := generate([]string{"localhost"}); err == nil || priv != "" || pub != "" {
		t.Fatalf("expected an error and no paths, got %q %q %v", priv, pub, err)
	}
	createTemp = os.CreateTemp

	// a file that cannot be written is removed
	t.Setenv("TMPDIR", dir+"/missing")
	if _, err := writeTemp("x-*", nil); err == nil {
		t.Fatal("expected an error")
	}
	t.Setenv("TMPDIR", dir)
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("%d files left", len(entries))
	}
}
