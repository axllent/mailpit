package storage

import (
	"fmt"
	"os"
	"testing"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
)

var (
	testTextEmail []byte
	testTagEmail  []byte
	testMimeEmail []byte
	testRuns      = 100
)

func setup() {
	logger.NoLogging = true
	config.MaxMessages = 0
	config.Database = os.Getenv("MP_DATABASE")

	if err := InitDB(); err != nil {
		panic(err)
	}

	var err error

	// ensure DB is empty
	if err := DeleteAllMessages(); err != nil {
		panic(err)
	}

	testTextEmail, err = os.ReadFile("testdata/plain-text.eml")
	if err != nil {
		panic(err)
	}

	testTagEmail, err = os.ReadFile("testdata/tags.eml")
	if err != nil {
		panic(err)
	}

	testMimeEmail, err = os.ReadFile("testdata/mime-attachment.eml")
	if err != nil {
		panic(err)
	}
}

func assertEqual(t *testing.T, a interface{}, b interface{}, message string) {
	if a == b {
		return
	}
	message = fmt.Sprintf("%s: \"%v\" != \"%v\"", message, a, b)
	t.Fatal(message)
}

func assertEqualStats(t *testing.T, total int, unread int) {
	s := StatsGet()
	if float64(total) != s.Total {
		t.Fatalf("Incorrect total mailbox stats: \"%v\" != \"%v\"", total, s.Total)
	}

	if float64(unread) != s.Unread {
		t.Fatalf("Incorrect unread mailbox stats: \"%v\" != \"%v\"", unread, s.Unread)
	}
}
