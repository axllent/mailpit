package storage

import (
	"net/mail"
	"testing"
)

func TestNormalizeAddress(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// dot-atom local-parts: no quoting needed
		{"plain address unchanged", "user@example.com", "user@example.com"},
		{"dot-atom unchanged", "first.last@example.com", "first.last@example.com"},
		{"atext specials unchanged", "user+tag@example.com", "user+tag@example.com"},
		// non-dot-atom characters: quoting required
		{"space requires quoting", "odd user@example.com", `"odd user"@example.com`},
		{"embedded @ requires quoting", "a@b@example.com", `"a@b"@example.com`},
		// escaping within quoted local-parts
		{"double-quote escaped", "a\"b@example.com", `"a\"b"@example.com`},
		{"backslash escaped", "a\\b@example.com", `"a\\b"@example.com`},
		{"backslash and quote escaped", "a\\\"b@example.com", `"a\\\"b"@example.com`},
		// dot-atom structural violations: quoting required
		{"leading dot requires quoting", ".user@example.com", `".user"@example.com`},
		{"trailing dot requires quoting", "user.@example.com", `"user."@example.com`},
		{"consecutive dots requires quoting", "user..name@example.com", `"user..name"@example.com`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &mail.Address{Address: tt.input}
			normalizeAddress(a)
			if a.Address != tt.expected {
				t.Errorf("normalizeAddress(%q) = %q, want %q", tt.input, a.Address, tt.expected)
			}
		})
	}
}

func TestNormalizeAddressWithName(t *testing.T) {
	a := &mail.Address{Name: "Bob", Address: "odd user@example.com"}
	normalizeAddress(a)
	if a.Address != `"odd user"@example.com` {
		t.Errorf("got %q, want %q", a.Address, `"odd user"@example.com`)
	}
	if a.Name != "Bob" {
		t.Errorf("Name was modified: got %q, want %q", a.Name, "Bob")
	}
}
