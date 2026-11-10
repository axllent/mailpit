package smtpd

import (
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/segment"
	"github.com/axllent/mailpit/internal/storage"
)

// sendSegmentMessage sends a message over an SMTP session, authenticating with
// AUTH PLAIN if a username is given.
func sendSegmentMessage(t *testing.T, username, subject string) {
	t.Helper()

	server := &Server{
		MsgIDHandler: mailHandler,
		AuthHandler:  authHandlerAny,
		AuthMechs:    map[string]bool{"PLAIN": true},
	}

	conn := newConn(t, server)
	defer func() { _ = conn.Close() }()

	cmdCode(t, conn, "EHLO host.example.com", "250")

	if username != "" {
		cmdCode(t, conn, "AUTH PLAIN", "334")
		cmdCode(t, conn, base64.StdEncoding.EncodeToString([]byte("\x00"+username+"\x00password")), "235")
	}

	cmdCode(t, conn, "MAIL FROM:<sender@example.com>", "250")
	cmdCode(t, conn, "RCPT TO:<recipient@example.com>", "250")
	cmdCode(t, conn, "DATA", "354")
	cmdCode(t, conn, fmt.Sprintf("From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: %s\r\n\r\nHello\r\n.", subject), "250")
	cmdCode(t, conn, "QUIT", "221")
}

func TestSMTPAuthUsernameSegment(t *testing.T) {
	logger.NoLogging = true
	config.MaxMessages = 0
	config.Database = os.Getenv("MP_DATABASE")
	config.SegmentDomain = "mail.example.com"
	defer func() { config.SegmentDomain = "" }()

	if err := storage.InitDB(); err != nil {
		t.Fatal(err)
	}
	defer storage.Close()

	if err := storage.DeleteAllMessages(segment.All); err != nil {
		t.Fatal(err)
	}

	sendSegmentMessage(t, "Alpha.Project", "For alpha")
	sendSegmentMessage(t, "beta", "For beta")
	sendSegmentMessage(t, "beta", "For beta again")
	sendSegmentMessage(t, "", "Unauthenticated")

	tests := map[string]uint64{
		"alpha-project": 1,
		"beta":          2,
		segment.Default: 1,
		"alpha.project": 0,
		"unknown":       0,
		segment.All:     4,
	}

	for seg, expected := range tests {
		if got := storage.CountTotal(seg); got != expected {
			t.Errorf("segment %q has %d messages, expected %d", seg, got, expected)
		}
	}

	list, err := storage.List("alpha-project", 0, 0, 10)
	if err != nil || len(list) != 1 {
		t.Fatal("expected one message in segment alpha-project")
	}
	if list[0].Subject != "For alpha" || list[0].Username != "Alpha.Project" {
		t.Errorf("unexpected message in segment alpha-project: %q by %q", list[0].Subject, list[0].Username)
	}
}

func TestSMTPAuthUsernameSegmentDisabled(t *testing.T) {
	logger.NoLogging = true
	config.MaxMessages = 0
	config.Database = os.Getenv("MP_DATABASE")

	if err := storage.InitDB(); err != nil {
		t.Fatal(err)
	}
	defer storage.Close()

	if err := storage.DeleteAllMessages(segment.Default); err != nil {
		t.Fatal(err)
	}

	sendSegmentMessage(t, "alpha", "For alpha")

	// stored in the default segment when segments are disabled
	config.SegmentDomain = "mail.example.com"
	defer func() { config.SegmentDomain = "" }()

	if got := storage.CountTotal(segment.Default); got != 1 {
		t.Errorf("expected the message in the default segment, got %d", got)
	}
	if got := storage.CountTotal("alpha"); got != 0 {
		t.Errorf("expected no message in segment alpha, got %d", got)
	}
}

func TestAuthHandlerSegment(t *testing.T) {
	config.SegmentDomain = "mail.example.com"
	config.SegmentPassword = "shared"
	defer func() {
		config.SegmentDomain = ""
		config.SegmentPassword = ""
	}()

	tests := []struct {
		user, pass string
		ok         bool
	}{
		{"alpha", "shared", true},
		{"alpha", "wrong", false},
		{"Alpha", "shared", false},
		{"alpha.project", "shared", false},
		{"", "shared", false},
	}

	for _, tc := range tests {
		got, err := authHandlerSegment(&net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1025}, "PLAIN", []byte(tc.user), []byte(tc.pass), nil)
		if err != nil || got != tc.ok {
			t.Errorf("authHandlerSegment(%q, %q) = %v, expected %v", tc.user, tc.pass, got, tc.ok)
		}
	}
}
