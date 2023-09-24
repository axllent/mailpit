package storage

import (
	"fmt"
	"os"
	"testing"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/utils/logger"
)

var (
	testTextEmail []byte
	testMimeEmail []byte
	testRuns      = 100
)

func setup() {
	logger.NoLogging = true
	config.MaxMessages = 0
	config.DataFile = ""

	if err := InitDB(); err != nil {
		panic(err)
	}

	var err error

	testTextEmail, err = os.ReadFile("testdata/plain-text.eml")
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
	if total != s.Total {
		t.Fatalf("Incorrect total mailbox stats: \"%d\" != \"%d\"", total, s.Total)
	}

	if unread != s.Unread {
		t.Fatalf("Incorrect unread mailbox stats: \"%d\" != \"%d\"", unread, s.Unread)
	}
}
