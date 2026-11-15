package storage

import (
	"fmt"
	"os"
	"path/filepath"
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

// testDatabase is the SQLite database the tests use when MP_DATABASE is not set. Without it every test
// would have InitDB create a temporary database in the system's temp directory, which nothing removed.
var testDatabase string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "storage-test")
	if err != nil {
		panic(err)
	}
	testDatabase = filepath.Join(dir, "mailpit.db")

	code := m.Run()

	Close()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func setup(tenantID string) {
	// the background goroutines of the previous test read the configuration and the database
	stopBackground()

	if !logger.NoLogging {
		logger.NoLogging = true
	}
	config.MaxMessages = 0
	config.Database = os.Getenv("MP_DATABASE")
	if config.Database == "" {
		config.Database = testDatabase
	}
	config.TenantID = config.DBTenantID(tenantID)

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

func assertEqual(t *testing.T, a any, b any, message string) {
	if a == b {
		return
	}
	message = fmt.Sprintf("%s: \"%v\" != \"%v\"", message, a, b)
	t.Fatal(message)
}

func assertEqualStats(t *testing.T, total int, unread int) {
	s := StatsGet()
	if uint64(total) != s.Total {
		t.Fatalf("Incorrect total mailbox stats: \"%v\" != \"%v\"", total, s.Total)
	}

	if uint64(unread) != s.Unread {
		t.Fatalf("Incorrect unread mailbox stats: \"%v\" != \"%v\"", unread, s.Unread)
	}
}
