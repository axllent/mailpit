package storage

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/utils/logger"
)

var (
	testTextEmail []byte
	testMimeEmail []byte
	testRuns      = 100
)

func TestTextEmailInserts(t *testing.T) {
	setup()
	defer Close()

	t.Log("Testing text email storage")

	start := time.Now()

	assertEqualStats(t, 0, 0)

	for i := 0; i < testRuns; i++ {
		if _, err := Store(testTextEmail); err != nil {
			t.Log("error ", err)
			t.Fail()
		}
	}

	assertEqual(t, CountTotal(), testRuns, "Incorrect number of text emails stored")

	t.Logf("Inserted %d text emails in %s", testRuns, time.Since(start))

	assertEqualStats(t, testRuns, testRuns)

	delStart := time.Now()
	if err := DeleteAllMessages(); err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	assertEqual(t, CountTotal(), 0, "incorrect number of text emails deleted")

	t.Logf("deleted %d text emails in %s", testRuns, time.Since(delStart))

	assertEqualStats(t, 0, 0)
}

func TestMimeEmailInserts(t *testing.T) {
	setup()
	defer Close()

	t.Log("Testing mime email storage")

	start := time.Now()

	assertEqualStats(t, 0, 0)

	for i := 0; i < testRuns; i++ {
		if _, err := Store(testMimeEmail); err != nil {
			t.Log("error ", err)
			t.Fail()
		}
	}

	assertEqual(t, CountTotal(), testRuns, "Incorrect number of mime emails stored")

	t.Logf("Inserted %d text emails in %s", testRuns, time.Since(start))

	assertEqualStats(t, testRuns, testRuns)

	delStart := time.Now()
	if err := DeleteAllMessages(); err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	assertEqual(t, CountTotal(), 0, "incorrect number of mime emails deleted")

	t.Logf("Deleted %d mime emails in %s", testRuns, time.Since(delStart))

	assertEqualStats(t, 0, 0)
}

func TestRetrieveMimeEmail(t *testing.T) {
	setup()
	defer Close()

	t.Log("Testing mime email retrieval")

	id, err := Store(testMimeEmail)
	if err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	msg, err := GetMessage(id)
	if err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	assertEqual(t, msg.From.Name, "Sender Smith", "\"From\" name does not match")
	assertEqual(t, msg.From.Address, "sender2@example.com", "\"From\" address does not match")
	assertEqual(t, msg.Subject, "inline + attachment", "subject does not match")
	assertEqual(t, len(msg.To), 1, "incorrect number of recipients")
	assertEqual(t, msg.To[0].Name, "Recipient Ross", "\"To\" name does not match")
	assertEqual(t, msg.To[0].Address, "recipient2@example.com", "\"To\" address does not match")
	assertEqual(t, len(msg.Attachments), 1, "incorrect number of attachments")
	assertEqual(t, msg.Attachments[0].FileName, "Sample PDF.pdf", "attachment filename does not match")
	assertEqual(t, len(msg.Inline), 1, "incorrect number of inline attachments")
	assertEqual(t, msg.Inline[0].FileName, "inline-image.jpg", "inline attachment filename does not match")

	attachmentData, err := GetAttachmentPart(id, msg.Attachments[0].PartID)
	if err != nil {
		t.Log("error ", err)
		t.Fail()
	}
	assertEqual(t, len(attachmentData.Content), msg.Attachments[0].Size, "attachment size does not match")

	inlineData, err := GetAttachmentPart(id, msg.Inline[0].PartID)
	if err != nil {
		t.Log("error ", err)
		t.Fail()
	}
	assertEqual(t, len(inlineData.Content), msg.Inline[0].Size, "inline attachment size does not match")
}

func BenchmarkImportText(b *testing.B) {
	setup()
	defer Close()

	for i := 0; i < b.N; i++ {
		if _, err := Store(testTextEmail); err != nil {
			b.Log("error ", err)
			b.Fail()
		}
	}
}

func BenchmarkImportMime(b *testing.B) {
	setup()
	defer Close()

	for i := 0; i < b.N; i++ {
		if _, err := Store(testMimeEmail); err != nil {
			b.Log("error ", err)
			b.Fail()
		}
	}

}

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
