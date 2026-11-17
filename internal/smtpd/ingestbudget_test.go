package smtpd

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/internal/tools"
)

func smallBudget(t *testing.T, size int64, wait time.Duration) *tools.Budget {
	t.Helper()

	origBudget, origWait := ingestBudget, ingestWait
	ingestBudget, ingestWait = tools.NewBudget(size), wait
	t.Cleanup(func() { ingestBudget, ingestWait = origBudget, origWait })

	return ingestBudget
}

func budgetMessage(extra int) string {
	return "From: a@b.test\r\nTo: c@d.test\r\nSubject: budget\r\n\r\n" + strings.Repeat("x", extra) + "\r\n"
}

// The budget is free again: the whole of it can be taken at once.
func budgetIsFree(t *testing.T, b *tools.Budget, size int64) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	release, err := b.Acquire(ctx, size)
	if err != nil {
		t.Fatalf("the budget is still held: %v", err)
	}
	release()
}

// A message waits for its turn when the budget is full, and is stored once there is room.
func TestIngestWaitsForItsTurn(t *testing.T) {
	strictRFC(t, false)
	b := smallBudget(t, 1000, 10*time.Second)

	hold, err := b.Acquire(context.Background(), 1000)
	if err != nil {
		t.Fatal(err)
	}

	before := storage.CountTotal()
	type result struct {
		id  string
		err error
	}
	done := make(chan result, 1)
	go func() { id, err := save(t, budgetMessage(100)); done <- result{id, err} }()

	select {
	case r := <-done:
		t.Fatalf("stored while the budget was full: %q, %v", r.id, r.err)
	case <-time.After(300 * time.Millisecond):
	}
	if storage.CountTotal() != before {
		t.Error("the message was stored while it waited")
	}

	hold()
	select {
	case r := <-done:
		if r.err != nil || r.id == "" {
			t.Fatalf("%q, %v", r.id, r.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("not stored once there was room")
	}
	if storage.CountTotal() != before+1 {
		t.Error("the message was not stored")
	}
	budgetIsFree(t, b, 1000)
}

// A sender which waits too long is told to try again later (451), and the message is not stored. It
// would otherwise wait for as long as the others take, which can be longer than a client waits.
func TestIngestToldToTryAgainLater(t *testing.T) {
	strictRFC(t, false)
	b := smallBudget(t, 1000, 200*time.Millisecond)

	hold, _ := b.Acquire(context.Background(), 1000)
	defer hold()

	before := storage.CountTotal()
	start := time.Now()
	id, err := save(t, budgetMessage(100))
	if err == nil || !strings.HasPrefix(err.Error(), "451 4.3.2 ") || id != "" {
		t.Fatalf("id %q, error %v", id, err)
	}
	if d := time.Since(start); d < 150*time.Millisecond || d > 3*time.Second {
		t.Errorf("waited %v", d)
	}
	if storage.CountTotal() != before {
		t.Error("the message was stored")
	}
	// it is sent to the client as it is, and as a temporary failure
	if !checkErrFormatRE.MatchString(err.Error()) {
		t.Errorf("%q is not sent as an SMTP response", err.Error())
	}

	// in a session, the client gets the 451
	conn := newConn(t, &Server{MsgIDHandler: SaveToDatabase})
	defer conn.Close()
	cmdCode(t, conn, "EHLO host.example.com", "250")
	cmdCode(t, conn, "MAIL FROM:<a@b.test>", "250")
	cmdCode(t, conn, "RCPT TO:<c@d.test>", "250")
	cmdCode(t, conn, "DATA", "354")
	if resp := cmdCode(t, conn, "From: a@b.test\r\nTo: c@d.test\r\nSubject: x\r\n\r\nbody\r\n.", "451"); !strings.Contains(resp, "4.3.2") {
		t.Errorf("response %q", resp)
	}
}

// A message which is larger than the budget is stored, on its own, and not refused.
func TestIngestLargerThanTheBudgetIsStored(t *testing.T) {
	strictRFC(t, false)
	b := smallBudget(t, 100, 5*time.Second)

	id, err := save(t, budgetMessage(5000))
	if err != nil || id == "" {
		t.Fatalf("%q, %v", id, err)
	}
	budgetIsFree(t, b, 100)

	// while it is held on its own, others wait, and they go once it is done
	hold, _ := b.Acquire(context.Background(), 5000)
	done := make(chan error, 1)
	go func() { _, err := save(t, budgetMessage(10)); done <- err }()
	select {
	case <-done:
		t.Fatal("stored while a larger message had all of it")
	case <-time.After(300 * time.Millisecond):
	}
	hold()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

// Control: a small message is not held up when there is room, and the budget is given back.
func TestIngestSmallMessageWithRoomDoesNotWait(t *testing.T) {
	strictRFC(t, false)
	b := smallBudget(t, 1000, 5*time.Second)

	hold, _ := b.Acquire(context.Background(), 400)
	defer hold()

	start := time.Now()
	id, err := save(t, budgetMessage(50))
	if err != nil || id == "" {
		t.Fatalf("%q, %v", id, err)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("took %v", d)
	}

	// only what was held is held: the message gave back what it took
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	release, err := b.Acquire(ctx, 600)
	if err != nil {
		t.Fatalf("the message did not give back its share: %v", err)
	}
	release()
}

// A message which is refused before it is stored (headers which cannot be read) does not take any of it.
func TestIngestRefusedMessageTakesNoBudget(t *testing.T) {
	strictRFC(t, false)
	b := smallBudget(t, 1000, 5*time.Second)

	if _, err := save(t, "From: a@b.test\r\nthis is not a header\r\n\r\nbody"); err == nil {
		t.Fatal("expected an error")
	}
	budgetIsFree(t, b, 1000)
}
