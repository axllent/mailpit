package auth

import (
	"errors"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const validSHA512Crypt = "$6$saltsalt$qFmFH.bQmmtXzyBY0s9v7Oicd2z4XSIecDzlB5KiA2/jctKu9YterLp8wwnSq.qc.eoxqOmSuNp2xS0ktL3nh/"

func TestLoad(t *testing.T) {
	bc, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		in      string
		wantErr bool
		noCreds bool
	}{
		{"control: plain", "alice:secret", false, false},
		{"control: bcrypt", "alice:" + string(bc), false, false},
		{"control: valid sha512crypt", "alice:" + validSHA512Crypt, false, false},
		{"control: several entries", "alice:a bob:b\ncarol:c", false, false},
		{"empty", "", false, true},
		{"whitespace only", " \n\t ", false, true},
		{"space after colon leaves a stray entry", "alice: secret", true, false},
		{"entry without colon", "alicesecret", true, false},
		{"valid entry plus typo", "alice:secret bob", true, false},
		{"malformed bcrypt hash", "alice:$2y$bad", true, false},
		{"sha256crypt with rounds and no hash", "alice:$5$rounds=1$", true, false},
		{"sha512crypt with invalid rounds", "alice:$6$rounds=abc$salt$hash", true, false},
		{"line over the 64 KiB scanner limit", "alice:" + strings.Repeat("a", 70000), true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := load(tt.in)

			if tt.noCreds {
				if !errors.Is(err, ErrNoCredentials) || c != nil {
					t.Fatalf("want ErrNoCredentials, got c=%v err=%v", c, err)
				}
				return
			}

			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && c != nil {
				t.Fatal("credentials returned alongside an error")
			}

			// errors must never echo credentials back, they may contain passwords
			if err != nil && strings.Contains(err.Error(), "secret") {
				t.Fatalf("error leaks credential text: %v", err)
			}
		})
	}
}

func TestSettersLeaveGlobalsUntouchedOnError(t *testing.T) {
	UICredentials = nil
	t.Cleanup(func() { UICredentials = nil })

	if err := SetUIAuth("alice:secret"); err != nil {
		t.Fatal(err)
	}
	good := UICredentials

	if err := SetUIAuth("alice:$2y$bad"); err == nil {
		t.Fatal("expected an error")
	}
	if UICredentials != good {
		t.Fatal("a failed Set replaced the previous credentials")
	}

	if err := SetUIAuth(""); !errors.Is(err, ErrNoCredentials) {
		t.Fatalf("err = %v, want ErrNoCredentials", err)
	}
}

func TestMatch(t *testing.T) {
	c, err := load("alice:secret bob:" + validSHA512Crypt)
	if err != nil {
		t.Fatal(err)
	}

	long := strings.Repeat("a", maxCredentialLength+1)
	atLimit := strings.Repeat("a", maxCredentialLength)

	tests := []struct {
		name, user, pass string
		want             bool
	}{
		{"control: valid plain", "alice", "secret", true},
		{"wrong password", "alice", "nope", false},
		{"unknown user", "mallory", "secret", false},
		{"empty password", "alice", "", false},
		{"over-long password", "alice", long, false},
		{"over-long username", long, "secret", false},
		{"at the length limit is still checked", "alice", atLimit, false},
		{"control: sha512crypt hash cannot match a wrong password", "bob", "wrong", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := c.Match(tt.user, tt.pass); got != tt.want {
				t.Fatalf("Match = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchNilReceiver(t *testing.T) {
	var c *Credentials
	if c.Match("alice", "secret") {
		t.Fatal("nil credentials must never match")
	}
}

// A valid long password must be accepted at the limit, but anything longer must be rejected
// quickly, rather than being fed to an expensive hash.
func TestMatchLongPasswordIsCheap(t *testing.T) {
	c, err := load("alice:" + validSHA512Crypt)
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	c.Match("alice", strings.Repeat("a", 1<<17)) // takes about a minute without the bound
	if d := time.Since(start); d > time.Second {
		t.Fatalf("over-long password took %v", d)
	}
}

// Match must recover if the underlying library panics on an entry that passed load,
// so a future library bug cannot crash the SMTP or POP3 server.
func TestMatchRecoversFromPanic(t *testing.T) {
	// bypass load's validation to simulate an entry the validator did not anticipate
	f, err := loadUnvalidated("alice:$5$rounds=1$")
	if err != nil {
		t.Fatal(err)
	}

	if f.Match("alice", "x") {
		t.Fatal("panicking entry must not match")
	}
}

func TestCredentialsFromString(t *testing.T) {
	got := credentialsFromString("  a:b\tc:d\r\n\ne:f ")
	if strings.Join(got, ",") != "a:b,c:d,e:f" {
		t.Fatalf("got %v", got)
	}
}
