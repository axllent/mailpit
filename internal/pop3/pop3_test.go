package pop3

import (
	"bytes"
	"fmt"
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
	"github.com/jhillyerd/enmime"
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
	c, err := connectBadAuth()
	if err == nil {
		t.Error("invalid login gained access")
		return
	}

	t.Log("Testing valid login")
	c, err = connectAuth()
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	count, size, err := c.Stat()
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	assertEqual(t, count, 0, "incorrect message count")
	assertEqual(t, size, 0, "incorrect size")

	// quit else we get old data
	if err := c.Quit(); err != nil {
		t.Errorf(err.Error())
		return
	}

	t.Log("Inserting 50 messages")

	insertEmailData(t) // insert 50 messages

	c, err = connectAuth()
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	count, _, err = c.Stat()
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	assertEqual(t, count, 50, "incorrect message count")

	t.Log("Fetching 20 messages")

	for i := 1; i <= 20; i++ {
		_, err := c.Retr(i)
		if err != nil {
			t.Errorf(err.Error())
			return
		}
	}

	t.Log("Deleting 25 messages")

	for i := 1; i <= 25; i++ {
		if err := c.Dele(i); err != nil {
			t.Errorf(err.Error())
			return
		}
	}

	// messages get deleted after a QUIT
	if err := c.Quit(); err != nil {
		t.Errorf(err.Error())
		return
	}

	c, err = connectAuth()
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	t.Log("Fetching message count")

	count, _, err = c.Stat()
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	assertEqual(t, count, 25, "incorrect message count")

	// messages get deleted after a QUIT
	if err := c.Quit(); err != nil {
		t.Errorf(err.Error())
		return
	}

	c, err = connectAuth()
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	t.Log("Deleting 25 messages")

	for i := 1; i <= 25; i++ {
		if err := c.Dele(i); err != nil {
			t.Errorf(err.Error())
			return
		}
	}

	t.Log("Undeleting messages")

	if err := c.Rset(); err != nil {
		t.Errorf(err.Error())
		return
	}

	if err := c.Quit(); err != nil {
		t.Errorf(err.Error())
		return
	}

	c, err = connectAuth()
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	count, _, err = c.Stat()
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	assertEqual(t, count, 25, "incorrect message count")

	if err := c.Quit(); err != nil {
		t.Errorf(err.Error())
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
		t.Errorf(err.Error())
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
		t.Errorf(err.Error())
		return
	}

	t.Log("Testing authenticated commands while logged in")

	// authenticated connection
	c, err = connectAuth()
	if err != nil {
		t.Errorf(err.Error())
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
		t.Errorf(err.Error())
		return
	}
}

func setup() {
	auth.SetPOP3Auth("username:password")
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
	for i := 0; i < 50; i++ {
		msg := enmime.Builder().
			From(fmt.Sprintf("From %d", i), fmt.Sprintf("from-%d@example.com", i)).
			Subject(fmt.Sprintf("Subject line %d end", i)).
			Text([]byte(fmt.Sprintf("This is the email body %d <jdsauk;dwqmdqw;>.", i))).
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

		id, err := storage.Store(&bufBytes)
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

func assertEqual(t *testing.T, a interface{}, b interface{}, message string) {
	if a == b {
		return
	}
	message = fmt.Sprintf("%s: \"%v\" != \"%v\"", message, a, b)
	t.Fatal(message)
}
