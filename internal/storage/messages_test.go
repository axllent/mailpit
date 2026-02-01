package storage

import (
	"os"
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
)

func TestTextEmailInserts(t *testing.T) {
	setup("")
	defer Close()

	t.Log("Testing text email storage")

	start := time.Now()

	for i := 0; i < testRuns; i++ {
		if _, err := Store(&testTextEmail, nil); err != nil {
			t.Log("error ", err)
			t.Fail()
		}
	}

	assertEqual(t, CountTotal(), uint64(testRuns), "Incorrect number of text emails stored")

	t.Logf("Inserted %d text emails in %s", testRuns, time.Since(start))

	delStart := time.Now()
	if err := DeleteAllMessages(); err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	assertEqual(t, CountTotal(), uint64(0), "incorrect number of text emails deleted")

	t.Logf("deleted %d text emails in %s", testRuns, time.Since(delStart))

	assertEqualStats(t, 0, 0)
}

func TestMimeEmailInserts(t *testing.T) {
	for _, tenantID := range []string{"", "MyServer 3", "host.example.com"} {
		tenantID = config.DBTenantID(tenantID)

		setup(tenantID)

		if tenantID == "" {
			t.Log("Testing mime email storage")
		} else {
			t.Logf("Testing mime email storage (tenant %s)", tenantID)
		}

		start := time.Now()

		for i := 0; i < testRuns; i++ {
			if _, err := Store(&testMimeEmail, nil); err != nil {
				t.Log("error ", err)
				t.Fail()
			}
		}

		assertEqual(t, CountTotal(), uint64(testRuns), "Incorrect number of mime emails stored")

		t.Logf("Inserted %d text emails in %s", testRuns, time.Since(start))

		delStart := time.Now()
		if err := DeleteAllMessages(); err != nil {
			t.Log("error ", err)
			t.Fail()
		}

		assertEqual(t, CountTotal(), uint64(0), "incorrect number of mime emails deleted")

		t.Logf("Deleted %d mime emails in %s", testRuns, time.Since(delStart))

		Close()
	}
}

func TestRetrieveMimeEmail(t *testing.T) {
	compressionLevels := []int{0, 1, 2, 3}

	for _, compressionLevel := range compressionLevels {
		t.Logf("Testing compression level: %d", compressionLevel)
		for _, tenantID := range []string{"", "MyServer 3", "host.example.com"} {
			tenantID = config.DBTenantID(tenantID)
			config.Compression = compressionLevel
			setup(tenantID)

			if tenantID == "" {
				t.Log("Testing mime email retrieval")
			} else {
				t.Logf("Testing mime email retrieval (tenant %s)", tenantID)
			}

			id, err := Store(&testMimeEmail, nil)
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
			assertEqual(t, uint64(len(attachmentData.Content)), msg.Attachments[0].Size, "attachment size does not match")

			inlineData, err := GetAttachmentPart(id, msg.Inline[0].PartID)
			if err != nil {
				t.Log("error ", err)
				t.Fail()
			}
			assertEqual(t, uint64(len(inlineData.Content)), msg.Inline[0].Size, "inline attachment size does not match")

			Close()
		}
	}

	// reset compression
	config.Compression = 1
}

func TestMessageSummary(t *testing.T) {
	for _, tenantID := range []string{"", "MyServer 3", "host.example.com"} {
		tenantID = config.DBTenantID(tenantID)

		setup(tenantID)

		if tenantID == "" {
			t.Log("Testing message summary")
		} else {
			t.Logf("Testing message summary (tenant %s)", tenantID)
		}

		if _, err := Store(&testMimeEmail, nil); err != nil {
			t.Log("error ", err)
			t.Fail()
		}

		summaries, err := List(0, 0, 1)
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

		Close()
	}
}

func BenchmarkImportText(b *testing.B) {
	setup("")
	defer Close()

	for i := 0; i < b.N; i++ {
		if _, err := Store(&testTextEmail, nil); err != nil {
			b.Log("error ", err)
			b.Fail()
		}
	}
}

func BenchmarkImportMime(b *testing.B) {
	setup("")
	defer Close()

	for i := 0; i < b.N; i++ {
		if _, err := Store(&testMimeEmail, nil); err != nil {
			b.Log("error ", err)
			b.Fail()
		}
	}

}

func TestInlineImageContentIdHandling(t *testing.T) {
	setup("")
	defer Close()
	t.Log("Testing inline content handling")
	// Test case: Proper inline image with Content-Disposition: inline
	inlineAttachment, err := os.ReadFile("testdata/inline-attachment.eml")
	if err != nil {
		t.Fatalf("Failed to read test email: %v", err)
	}
	storedMessage, err := Store(&inlineAttachment, nil)
	if err != nil {
		t.Fatal("Failed to store test case 1:", err)
	}

	msg, err := GetMessage(storedMessage)
	if err != nil {
		t.Fatal("Failed to retrieve test case 1:", err)
	}
	// Assert
	if len(msg.Inline) != 1 {
		t.Errorf("Test case 1: Expected 1 inline attachment, got %d", len(msg.Inline))
	}
	if len(msg.Attachments) != 0 {
		t.Errorf("Test case 1: Expected 0 regular attachments, got %d", len(msg.Attachments))
	}
	if msg.Inline[0].ContentID != "test1@example.com" {
		t.Errorf("Test case 1: Expected ContentID 'test1@example.com', got '%s'", msg.Inline[0].ContentID)
	}
}

func TestRegularAttachmentHandling(t *testing.T) {
	setup("")
	defer Close()
	t.Log("Testing regular attachment handling")
	// Test case: Regular attachment without Content-ID
	regularAttachment, err := os.ReadFile("testdata/regular-attachment.eml")
	if err != nil {
		t.Fatalf("Failed to read test email: %v", err)
	}
	storedMessage, err := Store(&regularAttachment, nil)
	if err != nil {
		t.Fatal("Failed to store test case 3:", err)
	}
	msg, err := GetMessage(storedMessage)
	if err != nil {
		t.Fatal("Failed to retrieve test case 3:", err)
	}
	// Assert
	if len(msg.Inline) != 0 {
		t.Errorf("Test case 3: Expected 0 inline attachments, got %d", len(msg.Inline))
	}
	if len(msg.Attachments) != 1 {
		t.Errorf("Test case 3: Expected 1 regular attachment, got %d", len(msg.Attachments))
	}
	if msg.Attachments[0].ContentID != "" {
		t.Errorf("Test case 3: Expected empty ContentID, got '%s'", msg.Attachments[0].ContentID)
	}

	// Checksum tests
	assertEqual(t, msg.Attachments[0].Checksums.MD5, "b04930eb1ba0c62066adfa87e5d262c4", "Attachment MD5 checksum does not match")
	assertEqual(t, msg.Attachments[0].Checksums.SHA1, "15605d6a2fca44e966209d1701f16ecf816df880", "Attachment SHA1 checksum does not match")
	assertEqual(t, msg.Attachments[0].Checksums.SHA256, "92c4ccff376003381bd9054d3da7b32a3c5661905b55e3b0728c17aba6d223ec", "Attachment SHA256 checksum does not match")
}

func TestMixedAttachmentHandling(t *testing.T) {
	setup("")
	defer Close()
	t.Log("Testing mixed attachment handling")
	// Mixed scenario with both inline and regular attachment
	mixedAttachment, err := os.ReadFile("testdata/mixed-attachment.eml")
	if err != nil {
		t.Fatalf("Failed to read test email: %v", err)
	}
	storedMessage, err := Store(&mixedAttachment, nil)
	if err != nil {
		t.Fatal("Failed to store test case 4:", err)
	}
	msg, err := GetMessage(storedMessage)
	if err != nil {
		t.Fatal("Failed to retrieve test case 4:", err)
	}
	// Assert: Should have 1 inline (with ContentID) and 1 attachment (without ContentID)
	if len(msg.Inline) != 1 {
		t.Errorf("Test case 4: Expected 1 inline attachment, got %d", len(msg.Inline))
	}
	if len(msg.Attachments) != 1 {
		t.Errorf("Test case 4: Expected 1 regular attachment, got %d", len(msg.Attachments))
	}
	if msg.Inline[0].ContentID != "inline@example.com" {
		t.Errorf("Test case 4: Expected inline ContentID 'inline@example.com', got '%s'", msg.Inline[0].ContentID)
	}
	if msg.Attachments[0].ContentID != "" {
		t.Errorf("Test case 4: Expected attachment ContentID to be empty, got '%s'", msg.Attachments[0].ContentID)
	}
}
