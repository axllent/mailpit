package pop3

import (
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/auth"
	"github.com/axllent/mailpit/internal/storage"
)

func TestPOP3Segments(t *testing.T) {
	setup()
	defer storage.Close()

	if err := auth.SetPOP3Auth("alpha:password beta:password"); err != nil {
		t.Fatal(err)
	}

	config.SegmentDomain = "mail.example.com"
	defer func() { config.SegmentDomain = "" }()

	for seg, total := range map[string]int{"alpha": 2, "beta": 1, "": 1} {
		for range total {
			b := []byte("From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: For " + seg + "\r\n\r\nHello\r\n")
			if _, err := storage.Store(seg, &b, nil); err != nil {
				t.Fatal(err)
			}
		}
	}

	stat := func(user string) int {
		c, err := connect()
		if err != nil {
			t.Fatal(err)
		}
		if err := c.Auth(user, "password"); err != nil {
			t.Fatal(err)
		}
		count, _, err := c.Stat()
		if err != nil {
			t.Fatal(err)
		}
		_ = c.Quit()
		return count
	}

	assertEqual(t, stat("alpha"), 2, "incorrect message count for alpha")
	assertEqual(t, stat("beta"), 1, "incorrect message count for beta")

	// deleting as alpha only deletes in alpha
	c, err := connect()
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Auth("alpha", "password"); err != nil {
		t.Fatal(err)
	}
	if err := c.Dele(1, 2); err != nil {
		t.Fatal(err)
	}
	if err := c.Quit(); err != nil {
		t.Fatal(err)
	}

	// the deletion happens when the connection closes
	time.Sleep(100 * time.Millisecond)

	assertEqual(t, storage.CountTotal("alpha"), uint64(0), "alpha messages not deleted")
	assertEqual(t, storage.CountTotal("beta"), uint64(1), "beta messages deleted by alpha")
	assertEqual(t, storage.CountTotal(""), uint64(1), "default messages deleted by alpha")
}
