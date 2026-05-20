package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/auth"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/server/apiv1"
	jose "github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/jhillyerd/enmime/v2"
	"golang.org/x/crypto/bcrypt"
)

var (
	putDataStruct struct {
		Read bool
		IDs  []string
	}

	// Shared test message structure for consistency
	testSendMessage = map[string]any{
		"From": map[string]string{
			"Email": "test@example.com",
		},
		"To": []map[string]string{
			{"Email": "recipient@example.com"},
		},
		"Subject": "Test",
		"Text":    "Test message",
	}
)

func TestAPIv1Messages(t *testing.T) {
	setup()
	defer storage.Close()

	r := apiRoutes()

	ts := httptest.NewServer(r)
	defer ts.Close()

	m, err := fetchMessages(ts.URL + "/api/v1/messages")
	if err != nil {
		t.Error(err.Error())
	}

	// check count of empty database
	assertStatsEqual(t, ts.URL+"/api/v1/messages", 0, 0)

	// insert 100
	t.Log("Insert 100 messages")
	insertEmailData(t)
	assertStatsEqual(t, ts.URL+"/api/v1/messages", 100, 100)

	m, err = fetchMessages(ts.URL + "/api/v1/messages")
	if err != nil {
		t.Error(err.Error())
	}

	// read first 10 messages
	t.Log("Read first 10 messages including raw & headers")
	for idx, msg := range m.Messages {
		if idx == 10 {
			break
		}

		if _, err := clientGet(ts.URL + "/api/v1/message/" + msg.ID); err != nil {
			t.Error(err.Error())
		}

		// get RAW
		if _, err := clientGet(ts.URL + "/api/v1/message/" + msg.ID + "/raw"); err != nil {
			t.Error(err.Error())
		}

		// get headers
		if _, err := clientGet(ts.URL + "/api/v1/message/" + msg.ID + "/headers"); err != nil {
			t.Error(err.Error())
		}
	}

	// 10 should be marked as read
	assertStatsEqual(t, ts.URL+"/api/v1/messages", 90, 100)

	// delete all
	t.Log("Delete all messages")
	_, err = clientDelete(ts.URL+"/api/v1/messages", "{}")
	if err != nil {
		t.Errorf("Expected nil, received %s", err.Error())
	}
	assertStatsEqual(t, ts.URL+"/api/v1/messages", 0, 0)
}

func TestAPIv1ToggleReadStatus(t *testing.T) {
	setup()
	defer storage.Close()

	r := apiRoutes()

	ts := httptest.NewServer(r)
	defer ts.Close()

	m, err := fetchMessages(ts.URL + "/api/v1/messages")
	if err != nil {
		t.Error(err.Error())
	}

	// check count of empty database
	assertStatsEqual(t, ts.URL+"/api/v1/messages", 0, 0)

	// insert 100
	t.Log("Insert 100 messages")
	insertEmailData(t)
	assertStatsEqual(t, ts.URL+"/api/v1/messages", 100, 100)

	m, err = fetchMessages(ts.URL + "/api/v1/messages")
	if err != nil {
		t.Error(err.Error())
	}

	// read first 10 IDs
	t.Log("Get first 10 IDs")
	putIDs := []string{}
	for idx, msg := range m.Messages {
		if idx == 10 {
			break
		}

		// store for later
		putIDs = append(putIDs, msg.ID)
	}
	assertStatsEqual(t, ts.URL+"/api/v1/messages", 100, 100)

	// mark first 10 as unread
	t.Log("Mark first 10 as read")
	putData := putDataStruct
	putData.Read = true
	putData.IDs = putIDs
	j, err := json.Marshal(putData)
	if err != nil {
		t.Error(err.Error())
	}
	_, err = clientPut(ts.URL+"/api/v1/messages", string(j))
	if err != nil {
		t.Error(err.Error())
	}
	assertStatsEqual(t, ts.URL+"/api/v1/messages", 90, 100)

	// mark first 10 as read
	t.Log("Mark first 10 as unread")
	putData.Read = false
	j, err = json.Marshal(putData)
	if err != nil {
		t.Error(err.Error())
	}
	_, err = clientPut(ts.URL+"/api/v1/messages", string(j))
	if err != nil {
		t.Error(err.Error())
	}
	assertStatsEqual(t, ts.URL+"/api/v1/messages", 100, 100)

	// mark all as read
	putData.Read = true
	putData.IDs = []string{}
	j, err = json.Marshal(putData)
	if err != nil {
		t.Error(err.Error())
	}

	t.Log("Mark all read")
	_, err = clientPut(ts.URL+"/api/v1/messages", string(j))
	if err != nil {
		t.Error(err.Error())
	}
	assertStatsEqual(t, ts.URL+"/api/v1/messages", 0, 100)
}

func TestAPIv1Search(t *testing.T) {
	setup()
	defer storage.Close()

	r := apiRoutes()

	ts := httptest.NewServer(r)
	defer ts.Close()

	// insert 100
	t.Log("Insert 100 messages & tag")
	insertEmailData(t)
	assertStatsEqual(t, ts.URL+"/api/v1/messages", 100, 100)

	// search
	assertSearchEqual(t, ts.URL+"/api/v1/search", "from-1@example.com", 1)
	assertSearchEqual(t, ts.URL+"/api/v1/search", "from:from-1@example.com", 1)
	assertSearchEqual(t, ts.URL+"/api/v1/search", "-from:from-1@example.com", 99)
	assertSearchEqual(t, ts.URL+"/api/v1/search", "-FROM:FROM-1@EXAMPLE.COM", 99)
	assertSearchEqual(t, ts.URL+"/api/v1/search", "to:from-1@example.com", 0)
	assertSearchEqual(t, ts.URL+"/api/v1/search", "from:@example.com", 100)
	assertSearchEqual(t, ts.URL+"/api/v1/search", "subject:\"Subject line\"", 100)
	assertSearchEqual(t, ts.URL+"/api/v1/search", "subject:\"SUBJECT LINE 17 END\"", 1)
	assertSearchEqual(t, ts.URL+"/api/v1/search", "!thisdoesnotexist", 100)
	assertSearchEqual(t, ts.URL+"/api/v1/search", "-ThisDoesNotExist", 100)
	assertSearchEqual(t, ts.URL+"/api/v1/search", "thisdoesnotexist", 0)
	assertSearchEqual(t, ts.URL+"/api/v1/search", "tag:\"Test tag 065\"", 1)
	assertSearchEqual(t, ts.URL+"/api/v1/search", "tag:\"TEST TAG 065\"", 1)
	assertSearchEqual(t, ts.URL+"/api/v1/search", "!tag:\"Test tag 023\"", 99)
}

func TestAPIv1Send(t *testing.T) {
	setup()
	defer storage.Close()

	r := apiRoutes()

	ts := httptest.NewServer(r)
	defer ts.Close()

	jsonData := `{
		"From": {
		  "Email": "john@example.com",
		  "Name": "John Doe"
		},
		"To": [
		  {
			"Email": "jane@example.com",
			"Name": "Jane Doe"
		  }
		],
		"Cc": [
		  {
			"Email": "manager1@example.com",
			"Name": "Manager 1"
		  },
		  {
			"Email": "manager2@example.com",
			"Name": "Manager 2"
		  }
		],
		"Bcc": ["jack@example.com"],
		"Headers": {
			"X-IP": "1.2.3.4"
		},
		"Subject": "Mailpit message via the HTTP API",
		"Text": "This is the text body",
		"HTML": "<p style=\"font-family: arial\">Mailpit is <b>awesome</b>!</p>",
		"Attachments": [
		  {
			"Content": "VGhpcyBpcyBhIHBsYWluIHRleHQgYXR0YWNobWVudA==",
			"Filename": "Attached File.txt"
		  },
		  {
			"Content": "iVBORw0KGgoAAAANSUhEUgAAAEEAAAA8CAMAAAAOlSdoAAAACXBIWXMAAAHrAAAB6wGM2bZBAAAAS1BMVEVHcEwRfnUkZ2gAt4UsSF8At4UtSV4At4YsSV4At4YsSV8At4YsSV4At4YsSV4sSV4At4YsSV4At4YtSV4At4YsSV4At4YtSV8At4YsUWYNAAAAGHRSTlMAAwoXGiktRE5dbnd7kpOlr7zJ0d3h8PD8PCSRAAACWUlEQVR42pXT4ZaqIBSG4W9rhqQYocG+/ys9Y0Z0Br+x3j8zaxUPewFh65K+7yrIMeIY4MT3wPfEJCidKXEMnLaVkxDiELiMz4WEOAZSFghxBIypCOlKiAMgXfIqTnBgSm8CIQ6BImxEUxEckClVQiHGj4Ba4AQHikAIClwTE9KtIghAhUJwoLkmLnCiAHJLRKgIMsEtVUKbBUIwoAg2C4QgQBE6l4VCnApBgSKYLLApCnCa0+96AEMW2BQcmC+Pr3nfp7o5Exy49gIADcIqUELGfeA+bp93LmAJp8QJoEcN3C7NY3sbVANixMyI0nku20/n5/ZRf3KI2k6JEDWQtxcbdGuAqu3TAXG+/799Oyyas1B1MnMiA+XyxHp9q0PUKGPiRAau1fZbLRZV09wZcT8/gHk8QQAxXn8VgaDqcUmU6O/r28nbVwXAqca2mRNtPAF5+zoP2MeN9Fy4NgC6RfcbgE7XITBRYTtOE3U3C2DVff7pk+PkUxgAbvtnPXJaD6DxulMLwOhPS/M3MQkgg1ZFrIXnmfaZoOfpKiFgzeZD/WuKqQEGrfJYkyWf6vlG3xUgTuscnkNkQsb599q124kdpMUjCa/XARHs1gZymVtGt3wLkiFv8rUgTxitYCex5EVGec0Y9VmoDTFBSQte2TfXGXlf7hbdaUM9Sk7fisEN9qfBBTK+FZcvM9fQSdkl2vj4W2oX/bRogO3XasiNH7R0eW7fgRM834ImTg+Lg6BEnx4vz81rhr+MYPBBQg1v8GndEOrthxaCTxNAOut8WKLGZQl+MPz88Q9tAO/hVuSeqQAAAABJRU5ErkJggg==",
			"Filename": "logo.png",
			"ContentID": "inline-cid",
			"ContentType": "overridden/type"
		  }
		],
		"ReplyTo": [
		  {
			"Email": "secretary@example.com",
			"Name": "Secretary"
		  }
		],
		"Tags": [
		  "Tag 1",
		  "Tag 2"
		]
	  }`

	t.Log("Sending message via HTTP API")
	b, err := clientPost(ts.URL+"/api/v1/send", jsonData)
	if err != nil {
		t.Errorf("Expected nil, received %s", err.Error())
	}

	resp := struct {
		ID string
	}{}

	if err := json.Unmarshal(b, &resp); err != nil {
		t.Error(err.Error())
		return
	}

	t.Logf("Fetching response for message %s", resp.ID)
	msg, err := fetchMessage(ts.URL + "/api/v1/message/" + resp.ID)
	if err != nil {
		t.Error(err.Error())
	}

	t.Logf("Testing response for message %s", resp.ID)
	assertEqual(t, `Mailpit message via the HTTP API`, msg.Subject, "wrong subject")
	assertEqual(t, `This is the text body`, msg.Text, "wrong text")
	assertEqual(t, `<p style="font-family: arial">Mailpit is <b>awesome</b>!</p>`, msg.HTML, "wrong HTML")
	assertEqual(t, `"John Doe" <john@example.com>`, msg.From.String(), "wrong HTML")
	assertEqual(t, 1, len(msg.To), "wrong To count")
	assertEqual(t, `"Jane Doe" <jane@example.com>`, msg.To[0].String(), "wrong To address")
	assertEqual(t, 2, len(msg.Cc), "wrong Cc count")
	assertEqual(t, `"Manager 1" <manager1@example.com>`, msg.Cc[0].String(), "wrong Cc address")
	assertEqual(t, `"Manager 2" <manager2@example.com>`, msg.Cc[1].String(), "wrong Cc address")
	assertEqual(t, 1, len(msg.Bcc), "wrong Bcc count")
	assertEqual(t, `<jack@example.com>`, msg.Bcc[0].String(), "wrong Bcc address")
	assertEqual(t, 1, len(msg.ReplyTo), "wrong Reply-To count")
	assertEqual(t, `"Secretary" <secretary@example.com>`, msg.ReplyTo[0].String(), "wrong Reply-To address")
	assertEqual(t, 2, len(msg.Tags), "wrong Tags count")
	assertEqual(t, `Tag 1,Tag 2`, strings.Join(msg.Tags, ","), "wrong Tags")
	assertEqual(t, 1, len(msg.Attachments), "wrong Attachment count")
	assertEqual(t, `Attached File.txt`, msg.Attachments[0].FileName, "wrong Attachment name")
	assertEqual(t, `text/plain`, msg.Attachments[0].ContentType, "wrong Content-Type")
	assertEqual(t, 1, len(msg.Inline), "wrong inline Attachment count")
	assertEqual(t, `logo.png`, msg.Inline[0].FileName, "wrong Attachment name")
	assertEqual(t, `overridden/type`, msg.Inline[0].ContentType, "wrong Content-Type")

	attachmentBytes, err := clientGet(ts.URL + "/api/v1/message/" + resp.ID + "/part/" + msg.Attachments[0].PartID)
	if err != nil {
		t.Error(err.Error())
	}
	assertEqual(t, `This is a plain text attachment`, string(attachmentBytes), "wrong Attachment content")
}

func TestAPIv1SendMaxMessageSize(t *testing.T) {
	setup()
	defer storage.Close()

	r := apiRoutes()

	ts := httptest.NewServer(r)
	defer ts.Close()

	original := config.MaxMessageSize
	defer func() { config.MaxMessageSize = original }()

	config.MaxMessageSize = 1 // 1 MiB cap for the test

	bigText := strings.Repeat("X", 2*1024*1024)
	oversized := fmt.Sprintf(`{
		"From": {"Email": "a@example.com"},
		"To": [{"Email": "b@example.com"}],
		"Subject": "oversize",
		"Text": %q
	}`, bigText)

	t.Log("Sending oversize message via HTTP API (expect 413)")
	req, err := http.NewRequest("POST", ts.URL+"/api/v1/send", strings.NewReader(oversized))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unexpected transport error: %s", err)
	}
	_ = resp.Body.Close()
	assertEqual(t, http.StatusRequestEntityTooLarge, resp.StatusCode, "expected 413 for oversize body")

	t.Log("Sending normal-sized message via HTTP API (expect 200)")
	jsonData, _ := json.Marshal(testSendMessage)
	if _, err := clientPost(ts.URL+"/api/v1/send", string(jsonData)); err != nil {
		t.Errorf("expected success for in-bound payload, got: %s", err)
	}

	t.Log("Setting MaxMessageSize=0 (unlimited), oversize should now succeed")
	config.MaxMessageSize = 0
	if _, err := clientPost(ts.URL+"/api/v1/send", oversized); err != nil {
		t.Errorf("expected success when MaxMessageSize=0, got: %s", err)
	}
}

func TestSendAPIAuthMiddleware(t *testing.T) {
	setup()
	defer storage.Close()

	// Test 1: Send API with accept-any enabled (should bypass all auth)
	t.Run("SendAPIAuthAcceptAny", func(t *testing.T) {
		// Set up UI auth and enable accept-any for send API
		originalSendAPIAuthAcceptAny := config.SendAPIAuthAcceptAny
		originalUICredentials := auth.UICredentials
		defer func() {
			config.SendAPIAuthAcceptAny = originalSendAPIAuthAcceptAny
			auth.UICredentials = originalUICredentials
		}()

		// Enable accept-any for send API
		config.SendAPIAuthAcceptAny = true

		// Set up UI auth that would normally block requests
		testHash, _ := bcrypt.GenerateFromPassword([]byte("testpass"), bcrypt.DefaultCost)
		if err := auth.SetUIAuth("testuser:" + string(testHash)); err != nil {
			t.Fatalf("Failed to set UI auth: %s", err.Error())
		}

		r := apiRoutes()
		ts := httptest.NewServer(r)
		defer ts.Close()

		// Should succeed without any auth headers
		jsonData, _ := json.Marshal(testSendMessage)
		_, err := clientPost(ts.URL+"/api/v1/send", string(jsonData))
		if err != nil {
			t.Errorf("Expected send to succeed with accept-any, got error: %s", err.Error())
		}
	})

	// Test 2: Send API with dedicated credentials
	t.Run("SendAPIWithDedicatedCredentials", func(t *testing.T) {
		originalSendAPIAuthAcceptAny := config.SendAPIAuthAcceptAny
		originalUICredentials := auth.UICredentials
		originalSendAPICredentials := auth.SendAPICredentials
		defer func() {
			config.SendAPIAuthAcceptAny = originalSendAPIAuthAcceptAny
			auth.UICredentials = originalUICredentials
			auth.SendAPICredentials = originalSendAPICredentials
		}()

		config.SendAPIAuthAcceptAny = false

		// Set up UI auth
		uiHash, _ := bcrypt.GenerateFromPassword([]byte("uipass"), bcrypt.DefaultCost)
		if err := auth.SetUIAuth("uiuser:" + string(uiHash)); err != nil {
			t.Fatalf("Failed to set UI auth: %s", err.Error())
		}

		// Set up dedicated Send API auth
		sendHash, _ := bcrypt.GenerateFromPassword([]byte("sendpass"), bcrypt.DefaultCost)
		if err := auth.SetSendAPIAuth("senduser:" + string(sendHash)); err != nil {
			t.Fatalf("Failed to set Send API auth: %s", err.Error())
		}

		r := apiRoutes()
		ts := httptest.NewServer(r)
		defer ts.Close()

		jsonData, _ := json.Marshal(testSendMessage)

		// Should succeed with correct Send API credentials
		_, err := clientPostWithAuth(ts.URL+"/api/v1/send", string(jsonData), "senduser", "sendpass")
		if err != nil {
			t.Errorf("Expected send to succeed with correct Send API credentials, got error: %s", err.Error())
		}

		// Should fail with wrong Send API credentials
		_, err = clientPostWithAuth(ts.URL+"/api/v1/send", string(jsonData), "senduser", "wrongpass")
		if err == nil {
			t.Error("Expected send to fail with wrong Send API credentials")
		}

		// Should fail with UI credentials when Send API credentials are set
		_, err = clientPostWithAuth(ts.URL+"/api/v1/send", string(jsonData), "uiuser", "uipass")
		if err == nil {
			t.Error("Expected send to fail with UI credentials when Send API credentials are required")
		}
	})

	// Test 3: Send API fallback to UI auth when no Send API auth is configured
	t.Run("SendAPIFallbackToUIAuth", func(t *testing.T) {
		originalSendAPIAuthAcceptAny := config.SendAPIAuthAcceptAny
		originalUICredentials := auth.UICredentials
		originalSendAPICredentials := auth.SendAPICredentials
		defer func() {
			config.SendAPIAuthAcceptAny = originalSendAPIAuthAcceptAny
			auth.UICredentials = originalUICredentials
			auth.SendAPICredentials = originalSendAPICredentials
		}()

		config.SendAPIAuthAcceptAny = false
		auth.SendAPICredentials = nil

		// Set up only UI auth
		uiHash, _ := bcrypt.GenerateFromPassword([]byte("uipass"), bcrypt.DefaultCost)
		if err := auth.SetUIAuth("uiuser:" + string(uiHash)); err != nil {
			t.Fatalf("Failed to set UI auth: %s", err.Error())
		}

		r := apiRoutes()
		ts := httptest.NewServer(r)
		defer ts.Close()

		jsonData, _ := json.Marshal(testSendMessage)

		// Should succeed with UI credentials when no Send API auth is configured
		_, err := clientPostWithAuth(ts.URL+"/api/v1/send", string(jsonData), "uiuser", "uipass")
		if err != nil {
			t.Errorf("Expected send to succeed with UI credentials when no Send API auth configured, got error: %s", err.Error())
		}

		// Should fail without any credentials
		_, err = clientPost(ts.URL+"/api/v1/send", string(jsonData))
		if err == nil {
			t.Error("Expected send to fail without credentials when UI auth is required")
		}
	})

	// Test 4: Regular API endpoints should not be affected by Send API auth settings
	t.Run("RegularAPINotAffectedBySendAPIAuth", func(t *testing.T) {
		originalSendAPIAuthAcceptAny := config.SendAPIAuthAcceptAny
		originalUICredentials := auth.UICredentials
		originalSendAPICredentials := auth.SendAPICredentials
		defer func() {
			config.SendAPIAuthAcceptAny = originalSendAPIAuthAcceptAny
			auth.UICredentials = originalUICredentials
			auth.SendAPICredentials = originalSendAPICredentials
		}()

		// Set up UI auth and Send API auth
		uiHash, _ := bcrypt.GenerateFromPassword([]byte("uipass"), bcrypt.DefaultCost)
		if err := auth.SetUIAuth("uiuser:" + string(uiHash)); err != nil {
			t.Fatalf("Failed to set UI auth: %s", err.Error())
		}

		sendHash, _ := bcrypt.GenerateFromPassword([]byte("sendpass"), bcrypt.DefaultCost)
		if err := auth.SetSendAPIAuth("senduser:" + string(sendHash)); err != nil {
			t.Fatalf("Failed to set Send API auth: %s", err.Error())
		}

		r := apiRoutes()
		ts := httptest.NewServer(r)
		defer ts.Close()

		// Regular API endpoint should require UI credentials, not Send API credentials
		_, err := clientGetWithAuth(ts.URL+"/api/v1/messages", "uiuser", "uipass")
		if err != nil {
			t.Errorf("Expected regular API to work with UI credentials, got error: %s", err.Error())
		}

		// Regular API endpoint should fail with Send API credentials
		_, err = clientGetWithAuth(ts.URL+"/api/v1/messages", "senduser", "sendpass")
		if err == nil {
			t.Error("Expected regular API to fail with Send API credentials")
		}
	})
}

func setup() {
	logger.NoLogging = true
	config.MaxMessages = 0
	config.Database = os.Getenv("MP_DATABASE")

	if err := storage.InitDB(); err != nil {
		panic(err)
	}

	if err := storage.DeleteAllMessages(); err != nil {
		panic(err)
	}
}

func assertStatsEqual(t *testing.T, uri string, unread, total int) {
	m := apiv1.MessagesSummary{}

	data, err := clientGet(uri)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if err := json.Unmarshal(data, &m); err != nil {
		t.Error(err.Error())
		return
	}

	assertEqual(t, uint64(unread), m.Unread, "wrong unread count")
	assertEqual(t, uint64(total), m.Total, "wrong total count")
}

func assertSearchEqual(t *testing.T, uri, query string, count int) {
	t.Logf("Test search: %s", query)
	m := apiv1.MessagesSummary{}

	limit := fmt.Sprintf("%d", count)

	data, err := clientGet(uri + "?query=" + url.QueryEscape(query) + "&limit=" + limit)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if err := json.Unmarshal(data, &m); err != nil {
		t.Error(err.Error())
		return
	}

	assertEqual(t, uint64(count), m.MessagesCount, "wrong search results count")
}

func insertEmailData(t *testing.T) {
	for i := range 100 {
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

func fetchMessage(url string) (storage.Message, error) {
	m := storage.Message{}

	data, err := clientGet(url)
	if err != nil {
		return m, err
	}

	if err := json.Unmarshal(data, &m); err != nil {
		return m, err
	}

	return m, nil
}

func fetchMessages(url string) (apiv1.MessagesSummary, error) {
	m := apiv1.MessagesSummary{}

	data, err := clientGet(url)
	if err != nil {
		return m, err
	}

	if err := json.Unmarshal(data, &m); err != nil {
		return m, err
	}

	return m, nil
}

func clientGet(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned status %d", url, resp.StatusCode)
	}

	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)

	return data, err
}

func clientDelete(url, body string) ([]byte, error) {
	client := new(http.Client)

	b := strings.NewReader(body)
	req, err := http.NewRequest("DELETE", url, b)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned status %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)

	return data, err
}

func clientPut(url, body string) ([]byte, error) {
	client := new(http.Client)

	b := strings.NewReader(body)
	req, err := http.NewRequest("PUT", url, b)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned status %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)

	return data, err
}

func clientPost(url, body string) ([]byte, error) {
	client := new(http.Client)

	b := strings.NewReader(body)
	req, err := http.NewRequest("POST", url, b)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned status %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)

	return data, err
}

func clientPostWithAuth(url, body, username, password string) ([]byte, error) {
	client := new(http.Client)

	b := strings.NewReader(body)
	req, err := http.NewRequest("POST", url, b)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned status %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)

	return data, err
}

func clientGetWithAuth(url, username, password string) ([]byte, error) {
	client := new(http.Client)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned status %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)

	return data, err
}

func assertEqual(t *testing.T, a any, b any, message string) {
	if a == b {
		return
	}
	message = fmt.Sprintf("%s: \"%v\" != \"%v\"", message, a, b)
	t.Fatal(message)
}

// clientGetWithBearer issues an authenticated GET using a Bearer JWT.
func clientGetWithBearer(url, token string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return http.DefaultClient.Do(req)
}

// clientGetRaw returns the raw *http.Response so tests can assert on
// status code and headers. The caller must close the body.
func clientGetRaw(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}

// testIdP is a minimal OIDC provider used for integration tests of
// the server's auth middleware. It mirrors the helper in
// internal/auth/oidc_test.go (kept separate to avoid build-tag plumbing).
type testIdP struct {
	t      *testing.T
	server *httptest.Server
	signer jose.Signer
	priv   *rsa.PrivateKey
}

func newTestIdP(t *testing.T) *testIdP {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa key: %v", err)
	}
	kid := "test-key-1"
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: priv},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", kid),
	)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	idp := &testIdP{t: t, signer: signer, priv: priv}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                idp.URL(),
			"jwks_uri":                              idp.URL() + "/jwks",
			"authorization_endpoint":                idp.URL() + "/authorize",
			"token_endpoint":                        idp.URL() + "/token",
			"id_token_signing_alg_values_supported": []string{"RS256"},
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		jwk := jose.JSONWebKey{Key: &priv.PublicKey, KeyID: kid, Algorithm: "RS256", Use: "sig"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}})
	})
	idp.server = httptest.NewServer(mux)
	return idp
}

func (i *testIdP) URL() string { return i.server.URL }
func (i *testIdP) Close()      { i.server.Close() }

func (i *testIdP) issue(clientID, sub string, exp time.Time) string {
	i.t.Helper()
	claims := map[string]any{
		"iss": i.URL(),
		"aud": clientID,
		"sub": sub,
		"exp": exp.Unix(),
		"iat": time.Now().Unix(),
	}
	tok, err := jwt.Signed(i.signer).Claims(claims).Serialize()
	if err != nil {
		i.t.Fatalf("sign jwt: %v", err)
	}
	return tok
}

func TestUIAuthOIDC(t *testing.T) {
	setup()
	defer storage.Close()

	idp := newTestIdP(t)
	defer idp.Close()

	origUI := auth.UICredentials
	origVerifier := auth.OIDCVerifier
	defer func() {
		auth.UICredentials = origUI
		auth.OIDCVerifier = origVerifier
	}()

	auth.UICredentials = nil // OIDC-only mode for this test
	if err := auth.InitOIDC(context.Background(), idp.URL(), "mailpit"); err != nil {
		t.Fatalf("init oidc: %v", err)
	}

	ts := httptest.NewServer(apiRoutes())
	defer ts.Close()

	t.Run("NoAuth_Returns401_WithOIDCHeader", func(t *testing.T) {
		resp, err := clientGetRaw(ts.URL + "/api/v1/messages")
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		assertEqual(t, http.StatusUnauthorized, resp.StatusCode, "expected 401")
		assertEqual(t, "oidc", resp.Header.Get("X-Mp-Auth-Required"), "expected X-Mp-Auth-Required header")
		// Basic challenge must NOT be present (no htpasswd configured).
		assertEqual(t, "", resp.Header.Get("WWW-Authenticate"), "Basic challenge unexpected")
	})

	t.Run("ValidBearer_Returns200", func(t *testing.T) {
		tok := idp.issue("mailpit", "alice", time.Now().Add(time.Hour))
		resp, err := clientGetWithBearer(ts.URL+"/api/v1/messages", tok)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		assertEqual(t, http.StatusOK, resp.StatusCode, "expected 200 with valid Bearer")
	})

	t.Run("BearerInQueryParam_Returns200", func(t *testing.T) {
		tok := idp.issue("mailpit", "alice", time.Now().Add(time.Hour))
		resp, err := clientGetRaw(ts.URL + "/api/v1/messages?access_token=" + tok)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		assertEqual(t, http.StatusOK, resp.StatusCode, "expected 200 with ?access_token=")
	})

	t.Run("ExpiredBearer_Returns401", func(t *testing.T) {
		tok := idp.issue("mailpit", "alice", time.Now().Add(-time.Hour))
		resp, err := clientGetWithBearer(ts.URL+"/api/v1/messages", tok)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		assertEqual(t, http.StatusUnauthorized, resp.StatusCode, "expected 401 for expired token")
	})

	t.Run("TamperedBearer_Returns401", func(t *testing.T) {
		tok := idp.issue("mailpit", "alice", time.Now().Add(time.Hour))
		// Flip a char in the middle of the signature segment so we are
		// always changing real signature bytes (the last base64url char
		// may encode unused padding bits and produce identical bytes).
		dot := strings.LastIndex(tok, ".")
		sigStart := dot + 1
		mid := sigStart + (len(tok)-sigStart)/2
		swap := byte('A')
		if tok[mid] == 'A' {
			swap = 'B'
		}
		tampered := tok[:mid] + string(swap) + tok[mid+1:]
		resp, err := clientGetWithBearer(ts.URL+"/api/v1/messages", tampered)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		assertEqual(t, http.StatusUnauthorized, resp.StatusCode, "expected 401 for tampered token")
	})

	t.Run("SPAShell_ServesWithoutAuth_WhenOIDCEnabled", func(t *testing.T) {
		// The SPA shell must load without an auth challenge so the SPA
		// can run the OIDC redirect itself. Otherwise the browser pops
		// up its native Basic Auth dialog and the SPA never boots.
		for _, path := range []string{"/", "/search", "/view/abc", "/dist/app.js", "/favicon.svg"} {
			resp, err := clientGetRaw(ts.URL + path)
			if err != nil {
				t.Fatalf("get %s: %v", path, err)
			}
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusUnauthorized {
				t.Errorf("%s: expected SPA shell to load without auth, got 401 (WWW-Authenticate=%q)",
					path, resp.Header.Get("Www-Authenticate"))
			}
			if got := resp.Header.Get("Www-Authenticate"); got != "" {
				t.Errorf("%s: SPA shell must not return WWW-Authenticate, got %q", path, got)
			}
		}
	})

}

func TestIsSPAShellRequest(t *testing.T) {
	// Webroot is "/" in tests by default.
	cases := []struct {
		method string
		path   string
		want   bool
	}{
		{"GET", "/", true},
		{"GET", "/search", true},
		{"GET", "/auth/callback", true},
		{"GET", "/view/abc", true},
		{"GET", "/view/abc123XYZ", true},
		{"GET", "/dist/app.js", true},
		{"GET", "/dist/app.css", true},
		{"GET", "/favicon.ico", true},
		{"GET", "/favicon.svg", true},
		{"GET", "/mailpit.svg", true},
		{"GET", "/notification.png", true},
		// Must remain gated:
		{"GET", "/view/abc.html", false},
		{"GET", "/view/abc.txt", false},
		{"GET", "/view/latest", false},
		{"GET", "/api/v1/messages", false},
		{"GET", "/api/v1/webui", false},
		{"GET", "/api/events", false},
		{"GET", "/proxy", false},
		// HEAD also qualifies (browsers + curl -I).
		{"HEAD", "/", true},
		{"HEAD", "/dist/app.js", true},
		// Non-GET/HEAD methods never qualify.
		{"POST", "/", false},
		{"PUT", "/dist/app.js", false},
	}
	for _, tc := range cases {
		r, err := http.NewRequest(tc.method, tc.path, nil)
		if err != nil {
			t.Fatalf("NewRequest: %v", err)
		}
		if got := isSPAShellRequest(r); got != tc.want {
			t.Errorf("isSPAShellRequest(%s %s) = %v, want %v", tc.method, tc.path, got, tc.want)
		}
	}
}

func TestUIAuthOIDCAndBasicCoexist(t *testing.T) {
	setup()
	defer storage.Close()

	idp := newTestIdP(t)
	defer idp.Close()

	origUI := auth.UICredentials
	origVerifier := auth.OIDCVerifier
	defer func() {
		auth.UICredentials = origUI
		auth.OIDCVerifier = origVerifier
	}()

	// Configure BOTH OIDC and Basic Auth.
	testHash, _ := bcrypt.GenerateFromPassword([]byte("testpass"), bcrypt.DefaultCost)
	if err := auth.SetUIAuth("testuser:" + string(testHash)); err != nil {
		t.Fatalf("set ui auth: %v", err)
	}
	if err := auth.InitOIDC(context.Background(), idp.URL(), "mailpit"); err != nil {
		t.Fatalf("init oidc: %v", err)
	}

	ts := httptest.NewServer(apiRoutes())
	defer ts.Close()

	t.Run("ValidBearer_Returns200", func(t *testing.T) {
		tok := idp.issue("mailpit", "alice", time.Now().Add(time.Hour))
		resp, err := clientGetWithBearer(ts.URL+"/api/v1/messages", tok)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		assertEqual(t, http.StatusOK, resp.StatusCode, "expected 200 with Bearer")
	})

	t.Run("ValidBasic_Returns200", func(t *testing.T) {
		if _, err := clientGetWithAuth(ts.URL+"/api/v1/messages", "testuser", "testpass"); err != nil {
			t.Fatalf("expected 200 with Basic, got %v", err)
		}
	})

	t.Run("NoAuth_401_OIDCHintOnly_NoBasicChallenge", func(t *testing.T) {
		// When OIDC is enabled the server must NOT advertise a Basic
		// challenge, even if htpasswd is also configured — otherwise
		// the browser pops its native dialog on any SPA-side 401.
		// Basic Auth still works for clients that proactively send it.
		resp, err := clientGetRaw(ts.URL + "/api/v1/messages")
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		assertEqual(t, http.StatusUnauthorized, resp.StatusCode, "expected 401")
		assertEqual(t, "oidc", resp.Header.Get("X-Mp-Auth-Required"), "expected OIDC hint")
		assertEqual(t, "", resp.Header.Get("WWW-Authenticate"), "Basic challenge must be suppressed when OIDC is enabled")
	})
}

func TestUIAuthOIDCDisabled_BasicStillWorks(t *testing.T) {
	setup()
	defer storage.Close()

	origUI := auth.UICredentials
	origVerifier := auth.OIDCVerifier
	defer func() {
		auth.UICredentials = origUI
		auth.OIDCVerifier = origVerifier
	}()

	auth.OIDCVerifier = nil
	testHash, _ := bcrypt.GenerateFromPassword([]byte("testpass"), bcrypt.DefaultCost)
	if err := auth.SetUIAuth("testuser:" + string(testHash)); err != nil {
		t.Fatalf("set ui auth: %v", err)
	}

	ts := httptest.NewServer(apiRoutes())
	defer ts.Close()

	resp, err := clientGetRaw(ts.URL + "/api/v1/messages")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	assertEqual(t, http.StatusUnauthorized, resp.StatusCode, "expected 401")
	// No OIDC, so no OIDC hint header.
	assertEqual(t, "", resp.Header.Get("X-Mp-Auth-Required"), "X-Mp-Auth-Required must not be set when OIDC is disabled")
	if resp.Header.Get("WWW-Authenticate") == "" {
		t.Fatalf("expected WWW-Authenticate Basic challenge")
	}

	if _, err := clientGetWithAuth(ts.URL+"/api/v1/messages", "testuser", "testpass"); err != nil {
		t.Fatalf("expected 200 with valid Basic creds, got %v", err)
	}
}

func TestUIAuthBothNil_AllowsAnonymous(t *testing.T) {
	setup()
	defer storage.Close()

	origUI := auth.UICredentials
	origVerifier := auth.OIDCVerifier
	defer func() {
		auth.UICredentials = origUI
		auth.OIDCVerifier = origVerifier
	}()

	auth.UICredentials = nil
	auth.OIDCVerifier = nil

	ts := httptest.NewServer(apiRoutes())
	defer ts.Close()

	resp, err := clientGetRaw(ts.URL + "/api/v1/messages")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	assertEqual(t, http.StatusOK, resp.StatusCode, "expected 200 with no auth configured")
}
