package storage

import (
	"testing"
	"time"
)

func TestTextEmailInserts(t *testing.T) {
	setup()
	defer Close()

	t.Log("Testing text email storage")

	start := time.Now()

	for i := 0; i < testRuns; i++ {
		if _, err := Store(&testTextEmail); err != nil {
			t.Log("error ", err)
			t.Fail()
		}
	}

	assertEqual(t, CountTotal(), float64(testRuns), "Incorrect number of text emails stored")

	t.Logf("Inserted %d text emails in %s", testRuns, time.Since(start))

	delStart := time.Now()
	if err := DeleteAllMessages(); err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	assertEqual(t, CountTotal(), float64(0), "incorrect number of text emails deleted")

	t.Logf("deleted %d text emails in %s", testRuns, time.Since(delStart))

	assertEqualStats(t, 0, 0)
}

func TestMimeEmailInserts(t *testing.T) {
	setup()
	defer Close()

	t.Log("Testing mime email storage")

	start := time.Now()

	for i := 0; i < testRuns; i++ {
		if _, err := Store(&testMimeEmail); err != nil {
			t.Log("error ", err)
			t.Fail()
		}
	}

	assertEqual(t, CountTotal(), float64(testRuns), "Incorrect number of mime emails stored")

	t.Logf("Inserted %d text emails in %s", testRuns, time.Since(start))

	delStart := time.Now()
	if err := DeleteAllMessages(); err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	assertEqual(t, CountTotal(), float64(0), "incorrect number of mime emails deleted")

	t.Logf("Deleted %d mime emails in %s", testRuns, time.Since(delStart))
}

func TestRetrieveMimeEmail(t *testing.T) {
	setup()
	defer Close()

	t.Log("Testing mime email retrieval")

	id, err := Store(&testMimeEmail)
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
	assertEqual(t, float64(len(attachmentData.Content)), msg.Attachments[0].Size, "attachment size does not match")

	inlineData, err := GetAttachmentPart(id, msg.Inline[0].PartID)
	if err != nil {
		t.Log("error ", err)
		t.Fail()
	}
	assertEqual(t, float64(len(inlineData.Content)), msg.Inline[0].Size, "inline attachment size does not match")
}

func TestMessageSummary(t *testing.T) {
	setup()
	defer Close()

	t.Log("Testing message summary")

	if _, err := Store(&testMimeEmail); err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	summaries, err := List(0, 1)
	if err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	assertEqual(t, len(summaries), 1, "Expected 1 result")

	msg := summaries[0]

	assertEqual(t, msg.From.Name, "Sender Smith", "\"From\" name does not match")
	assertEqual(t, msg.From.Address, "sender2@example.com", "\"From\" address does not match")
	assertEqual(t, msg.Subject, "inline + attachment", "subject does not match")
	assertEqual(t, len(msg.To), 1, "incorrect number of recipients")
	assertEqual(t, msg.To[0].Name, "Recipient Ross", "\"To\" name does not match")
	assertEqual(t, msg.To[0].Address, "recipient2@example.com", "\"To\" address does not match")
	assertEqual(t, msg.Snippet, "Message with inline image and attachment:", "\"Snippet\" does does not match")
	assertEqual(t, msg.Attachments, 1, "Expected 1 attachment")
	assertEqual(t, msg.MessageID, "33af2ac1-c33d-9738-35e3-a6daf90bbd89@gmail.com", "\"MessageID\" does not match")
}

func BenchmarkImportText(b *testing.B) {
	setup()
	defer Close()

	for i := 0; i < b.N; i++ {
		if _, err := Store(&testTextEmail); err != nil {
			b.Log("error ", err)
			b.Fail()
		}
	}
}

func BenchmarkImportMime(b *testing.B) {
	setup()
	defer Close()

	for i := 0; i < b.N; i++ {
		if _, err := Store(&testMimeEmail); err != nil {
			b.Log("error ", err)
			b.Fail()
		}
	}

}
