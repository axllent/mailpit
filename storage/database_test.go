package storage

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
)

var (
	testTextEmail []byte
	testMimeEmail []byte
)

func TestTextEmailInserts(t *testing.T) {
	setup()

	start := time.Now()
	for i := 0; i < 1000; i++ {
		if _, err := Store(DefaultMailbox, testTextEmail); err != nil {
			t.Log("error ", err)
			t.Fail()
		}
	}

	count, err := Count(DefaultMailbox)
	if err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	assertEqual(t, count, 1000, "incorrect number of text emails stored")

	t.Logf("inserted 1,000 text emails in %s\n", time.Since(start))

	delStart := time.Now()
	if err := DeleteAllMessages(DefaultMailbox); err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	count, err = Count(DefaultMailbox)
	if err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	assertEqual(t, count, 0, "incorrect number of text emails deleted")

	t.Logf("deleted 1,000 text emails in %s\n", time.Since(delStart))

	db.Close()
}

func TestMimeEmailInserts(t *testing.T) {
	setup()

	start := time.Now()
	for i := 0; i < 1000; i++ {
		if _, err := Store(DefaultMailbox, testMimeEmail); err != nil {
			t.Log("error ", err)
			t.Fail()
		}
	}

	count, err := Count(DefaultMailbox)
	if err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	assertEqual(t, count, 1000, "incorrect number of mime emails stored")

	t.Logf("inserted 1,000 emails with mime attachments in %s\n", time.Since(start))

	delStart := time.Now()
	if err := DeleteAllMessages(DefaultMailbox); err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	count, err = Count(DefaultMailbox)
	if err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	assertEqual(t, count, 0, "incorrect number of mime emails deleted")

	t.Logf("deleted 1,000 mime emails in %s\n", time.Since(delStart))

	db.Close()
}

func TestRetrieveMimeEmail(t *testing.T) {
	setup()

	id, err := Store(DefaultMailbox, testMimeEmail)
	if err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	msg, err := GetMessage(DefaultMailbox, id)
	if err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	assertEqual(t, msg.From.Name, "Sender Smith", "\"From\" name does not match")
	assertEqual(t, msg.From.Address, "sender@example.com", "\"From\" address does not match")
	assertEqual(t, msg.Subject, "inline + attachment", "subject does not match")
	assertEqual(t, len(msg.To), 1, "incorrect number of recipients")
	assertEqual(t, msg.To[0].Name, "Recipient Ross", "\"To\" name does not match")
	assertEqual(t, msg.To[0].Address, "recipient@example.com", "\"To\" address does not match")
	assertEqual(t, len(msg.Attachments), 1, "incorrect number of attachments")
	assertEqual(t, msg.Attachments[0].FileName, "Sample PDF.pdf", "attachment filename does not match")
	assertEqual(t, len(msg.Inline), 1, "incorrect number of inline attachments")
	assertEqual(t, msg.Inline[0].FileName, "inline-image.jpg", "inline attachment filename does not match")
	attachmentData, err := GetAttachmentPart(DefaultMailbox, id, msg.Attachments[0].PartID)
	assertEqual(t, len(attachmentData.Content), msg.Attachments[0].Size, "attachment size does not match")
	inlineData, err := GetAttachmentPart(DefaultMailbox, id, msg.Inline[0].PartID)
	assertEqual(t, len(inlineData.Content), msg.Inline[0].Size, "inline attachment size does not match")
}

func BenchmarkImportText(b *testing.B) {
	setup()

	for i := 0; i < b.N; i++ {
		if _, err := Store(DefaultMailbox, testTextEmail); err != nil {
			b.Log("error ", err)
			b.Fail()
		}
	}

	db.Close()
}

func BenchmarkImportMime(b *testing.B) {
	setup()

	for i := 0; i < b.N; i++ {
		if _, err := Store(DefaultMailbox, testMimeEmail); err != nil {
			b.Log("error ", err)
			b.Fail()
		}
	}
	db.Close()
}

func setup() {
	config.NoLogging = true
	config.MaxMessages = 0
	if err := InitDB(); err != nil {
		panic(err)
	}

	var err error

	testTextEmail, err = ioutil.ReadFile("testdata/plain-text.eml")
	if err != nil {
		panic(err)
	}

	testMimeEmail, err = ioutil.ReadFile("testdata/mime-attachment.eml")
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
