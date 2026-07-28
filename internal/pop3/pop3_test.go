package pop3

import (
	"bytes"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/auth"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/pop3client"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/jhillyerd/enmime/v2"
)

var (
	testingPort int
)

func TestPOP3(t *testing.T) {
	t.Log("Testing POP3 server")
	setup()
	defer storage.Close()

	// connect with bad password
	t.Log("Testing invalid login")
	if _, err := connectBadAuth(); err == nil {
		t.Error("invalid login gained access")
		return
	}

	t.Log("Testing valid login")
	c, err := connectAuth()
	if err != nil {
		t.Error(err.Error())
		return
	}

	count, size, err := c.Stat()
	if err != nil {
		t.Error(err.Error())
		return
	}

	assertEqual(t, count, 0, "incorrect message count")
	assertEqual(t, size, 0, "incorrect size")

	// quit else we get old data
	if err := c.Quit(); err != nil {
		t.Error(err.Error())
		return
	}

	t.Log("Inserting 50 messages")

	insertEmailData(t) // insert 50 messages

	c, err = connectAuth()
	if err != nil {
		t.Error(err.Error())
		return
	}

	count, _, err = c.Stat()
	if err != nil {
		t.Error(err.Error())
		return
	}

	assertEqual(t, count, 50, "incorrect message count")

	t.Log("Fetching 20 messages")

	for i := 1; i <= 20; i++ {
		_, err := c.Retr(i)
		if err != nil {
			t.Error(err.Error())
			return
		}
	}

	t.Log("Checking UIDL with multiple arguments")

	_, err = c.Cmd("UIDL", false, 1, 2, 3)
	if err == nil {
		t.Error("UIDL with multiple arguments should return an error")
		return
	}

	t.Log("Checking UIDL without a message id")

	messageIDs, err := c.Uidl(0)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if len(messageIDs) != 50 {
		assertEqual(t, len(messageIDs), 50, "incorrect UIDL message count")
	}

	t.Log("Checking UIDL with a message ID")

	messageIDs, err = c.Uidl(50)
	if err != nil {
		t.Error(err.Error())
		return
	}

	assertEqual(t, len(messageIDs), 1, "incorrect UIDL message count")

	t.Log("Checking UIDL with an invalid message ID")

	if _, err := c.Uidl(51); err == nil {
		t.Errorf("UIDL 51 should return an error")
		return
	}

	t.Log("Deleting 25 messages")

	for i := 1; i <= 25; i++ {
		if err := c.Dele(i); err != nil {
			t.Error(err.Error())
			return
		}
	}

	// messages get deleted after a QUIT
	if err := c.Quit(); err != nil {
		t.Error(err.Error())
		return
	}

	// allow for background delete when using rqlite driver
	time.Sleep(time.Millisecond * 200)

	c, err = connectAuth()
	if err != nil {
		t.Error(err.Error())
		return
	}

	t.Log("Fetching message count")

	count, _, err = c.Stat()
	if err != nil {
		t.Error(err.Error())
		return
	}

	assertEqual(t, count, 25, "incorrect message count")

	// messages get deleted after a QUIT
	if err := c.Quit(); err != nil {
		t.Error(err.Error())
		return
	}

	c, err = connectAuth()
	if err != nil {
		t.Error(err.Error())
		return
	}

	t.Log("Deleting 25 messages")

	for i := 1; i <= 25; i++ {
		if err := c.Dele(i); err != nil {
			t.Error(err.Error())
			return
		}
	}

	t.Log("Undeleting messages")

	if err := c.Rset(); err != nil {
		t.Error(err.Error())
		return
	}

	if err := c.Quit(); err != nil {
		t.Error(err.Error())
		return
	}

	c, err = connectAuth()
	if err != nil {
		t.Error(err.Error())
		return
	}

	count, _, err = c.Stat()
	if err != nil {
		t.Error(err.Error())
		return
	}

	assertEqual(t, count, 25, "incorrect message count")

	if err := c.Quit(); err != nil {
		t.Error(err.Error())
		return
	}
}

func TestAuthentication(t *testing.T) {
	// commands only allowed after authentication
	authCommands := make(map[string]bool)
	authCommands["STAT"] = false
	authCommands["LIST"] = true
	authCommands["NOOP"] = false
	authCommands["RSET"] = false
	authCommands["RETR 1"] = true

	t.Log("Testing authenticated commands while not logged in")
	setup()
	defer storage.Close()

	insertEmailData(t) // insert 50 messages

	// non-authenticated connection
	c, err := connect()
	if err != nil {
		t.Error(err.Error())
		return
	}

	for cmd, multi := range authCommands {
		if _, err := c.Cmd(cmd, multi); err == nil {
			t.Errorf("%s should require authentication", cmd)
			return
		}

		if _, err := c.Cmd(strings.ToLower(cmd), multi); err == nil {
			t.Errorf("%s should require authentication", cmd)
			return
		}
	}

	if err := c.Quit(); err != nil {
		t.Error(err.Error())
		return
	}

	t.Log("Testing authenticated commands while logged in")

	// authenticated connection
	c, err = connectAuth()
	if err != nil {
		t.Error(err.Error())
		return
	}

	for cmd, multi := range authCommands {
		if _, err := c.Cmd(cmd, multi); err != nil {
			t.Errorf("%s should work when authenticated", cmd)
			return
		}

		if _, err := c.Cmd(strings.ToLower(cmd), multi); err != nil {
			t.Errorf("%s should work when authenticated", cmd)
			return
		}
	}

	if err := c.Quit(); err != nil {
		t.Error(err.Error())
		return
	}
}

func TestBruteForceDisconnect(t *testing.T) {
	t.Log("Testing brute force login protection")
	setup()
	defer storage.Close()

	c, err := connect()
	if err != nil {
		t.Fatal(err)
	}

	for i := 1; i <= 5; i++ {
		if err := c.Send("USER username"); err != nil {
			t.Fatalf("attempt %d: USER send failed: %s", i, err)
		}
		if _, err := c.ReadOne(); err != nil {
			t.Fatalf("attempt %d: USER read failed: %s", i, err)
		}

		if err := c.Send("PASS wrongpassword"); err != nil {
			t.Fatalf("attempt %d: PASS send failed: %s", i, err)
		}
		_, passErr := c.ReadOne()
		if passErr == nil {
			t.Fatalf("attempt %d: PASS should have returned -ERR", i)
		}
	}

	// The 5th failure should have disconnected us.
	// Any further command should fail with a read error.
	if err := c.Send("USER username"); err != nil {
		// Write failed - connection already closed, that's fine
		return
	}
	if _, err := c.ReadOne(); err == nil {
		t.Error("connection should have been closed after 5 failed logins")
	}
}

func TestPOP3DotStuffing(t *testing.T) {
	// Test that byte-stuffing is correctly applied to TOP and RETR commands
	// RFC 1939 requires that lines beginning with "." be escaped as ".."
	t.Log("Testing POP3 dot-stuffing for TOP and RETR commands")
	setup()
	defer storage.Close()

	// Create a message with content that has lines starting with dots
	textBody := []byte("Line 1\n.This line starts with a dot\n..This line starts with two dots\nNormal line")
	msg := enmime.Builder().
		From("Test", "test@example.com").
		Subject("Dot-stuffing test").
		Text(textBody).
		To("Test", "test@example.com")

	env, err := msg.Build()
	if err != nil {
		t.Fatal("error building message:", err)
	}

	buf := new(bytes.Buffer)
	if err := env.Encode(buf); err != nil {
		t.Fatal("error encoding message:", err)
	}

	bufBytes := buf.Bytes()
	_, err = storage.Store(&bufBytes, nil)
	if err != nil {
		t.Fatal("error storing message:", err)
	}

	// Connect and authenticate
	c, err := connectAuth()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Quit()

	// Test RETR command with dot-stuffing
	t.Log("Testing RETR with dot-stuffing")
	retrivedMsg, err := c.Retr(1)
	if err != nil {
		t.Fatal("RETR failed:", err)
	}

	retrivedBody, err := io.ReadAll(retrivedMsg.Body)
	if err != nil {
		t.Fatal("error reading RETR body:", err)
	}

	bodyStr := string(retrivedBody)
	// The dot-stuffing should be transparent - the body should contain the original text
	if !strings.Contains(bodyStr, ".This line starts with a dot") {
		t.Errorf("RETR did not properly handle dot-stuffing: %s", bodyStr)
	}

	// Test TOP command with dot-stuffing
	t.Log("Testing TOP with dot-stuffing")
	topMsg, err := c.Top(1, 10)
	if err != nil {
		t.Fatal("TOP failed:", err)
	}

	topBody, err := io.ReadAll(topMsg.Body)
	if err != nil {
		t.Fatal("error reading TOP body:", err)
	}

	topBodyStr := string(topBody)
	// The dot-stuffing should be transparent - the body should contain the original text
	if !strings.Contains(topBodyStr, ".This line starts with a dot") {
		t.Errorf("TOP did not properly handle dot-stuffing: %s", topBodyStr)
	}

	if !strings.Contains(topBodyStr, "..This line starts with two dots") {
		t.Errorf("TOP did not properly handle double-dot-stuffing: %s", topBodyStr)
	}
}

func setup() {
	if err := auth.SetPOP3Auth("username:password"); err != nil {
		panic(err)
	}
	logger.NoLogging = true
	config.MaxMessages = 0
	config.Database = os.Getenv("MP_DATABASE")
	var foundPort bool
	for !foundPort {
		testingPort = randRange(1111, 2000)
		if portFree(testingPort) {
			foundPort = true
		}
	}

	config.POP3Listen = fmt.Sprintf("localhost:%d", testingPort)

	if err := storage.InitDB(); err != nil {
		panic(err)
	}

	if err := storage.DeleteAllMessages(); err != nil {
		panic(err)
	}

	go Run()

	time.Sleep(time.Second)
}

// connect and authenticate
func connectAuth() (*pop3client.Conn, error) {
	c, err := connect()
	if err != nil {
		return c, err
	}

	err = c.Auth("username", "password")

	return c, err
}

// connect and authenticate
func connectBadAuth() (*pop3client.Conn, error) {
	c, err := connect()
	if err != nil {
		return c, err
	}

	err = c.Auth("username", "notPassword")

	return c, err
}

// connect but do not authenticate
func connect() (*pop3client.Conn, error) {
	p := pop3client.New(pop3client.Opt{
		Host:       "localhost",
		Port:       testingPort,
		TLSEnabled: false,
	})

	c, err := p.NewConn()
	if err != nil {
		return c, err
	}

	return c, err
}

func portFree(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return false
	}

	if err := ln.Close(); err != nil {
		panic(err)
	}

	return true
}

func randRange(min, max int) int {
	return rand.IntN(max-min) + min
}

func insertEmailData(t *testing.T) {
	for i := range 50 {
		msg := enmime.Builder().
			From(fmt.Sprintf("From %d", i), fmt.Sprintf("from-%d@example.com", i)).
			Subject(fmt.Sprintf("Subject line %d end", i)).
			Text(fmt.Appendf(nil, "This is the email body %d <jdsauk;dwqmdqw;>.", i)).
			To(fmt.Sprintf("To %d", i), fmt.Sprintf("to-%d@example.com", i))

		env, err := msg.Build()
		if err != nil {
			t.Log("error ", err)
			t.Fail()
		}

		buf := new(bytes.Buffer)

		if err := env.Encode(buf); err != nil {
			t.Log("error ", err)
			t.Fail()
		}

		bufBytes := buf.Bytes()

		id, err := storage.Store(&bufBytes, nil)
		if err != nil {
			t.Log("error ", err)
			t.Fail()
		}

		if _, err := storage.SetMessageTags(id, []string{fmt.Sprintf("Test tag %03d", i)}); err != nil {
			t.Log("error ", err)
			t.Fail()
		}
	}
}

func assertEqual(t *testing.T, a any, b any, message string) {
	if a == b {
		return
	}
	message = fmt.Sprintf("%s: \"%v\" != \"%v\"", message, a, b)
	t.Fatal(message)
}
