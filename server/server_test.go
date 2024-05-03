package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/server/apiv1"
	"github.com/jhillyerd/enmime"
)

var (
	putDataStruct struct {
		Read bool
		IDs  []string
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
		t.Errorf(err.Error())
	}

	// check count of empty database
	assertStatsEqual(t, ts.URL+"/api/v1/messages", 0, 0)

	// insert 100
	t.Log("Insert 100 messages")
	insertEmailData(t)
	assertStatsEqual(t, ts.URL+"/api/v1/messages", 100, 100)

	m, err = fetchMessages(ts.URL + "/api/v1/messages")
	if err != nil {
		t.Errorf(err.Error())
	}

	// read first 10 messages
	t.Log("Read first 10 messages including raw & headers")
	for idx, msg := range m.Messages {
		if idx == 10 {
			break
		}

		if _, err := clientGet(ts.URL + "/api/v1/message/" + msg.ID); err != nil {
			t.Errorf(err.Error())
		}

		// get RAW
		if _, err := clientGet(ts.URL + "/api/v1/message/" + msg.ID + "/raw"); err != nil {
			t.Errorf(err.Error())
		}

		// get headers
		if _, err := clientGet(ts.URL + "/api/v1/message/" + msg.ID + "/headers"); err != nil {
			t.Errorf(err.Error())
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
		t.Errorf(err.Error())
	}

	// check count of empty database
	assertStatsEqual(t, ts.URL+"/api/v1/messages", 0, 0)

	// insert 100
	t.Log("Insert 100 messages")
	insertEmailData(t)
	assertStatsEqual(t, ts.URL+"/api/v1/messages", 100, 100)

	m, err = fetchMessages(ts.URL + "/api/v1/messages")
	if err != nil {
		t.Errorf(err.Error())
	}

	// read first 10 IDs
	t.Log("Get first 10 IDs")
	putIDS := []string{}
	for idx, msg := range m.Messages {
		if idx == 10 {
			break
		}

		// store for later
		putIDS = append(putIDS, msg.ID)
	}
	assertStatsEqual(t, ts.URL+"/api/v1/messages", 100, 100)

	// mark first 10 as unread
	t.Log("Mark first 10 as read")
	putData := putDataStruct
	putData.Read = true
	putData.IDs = putIDS
	j, err := json.Marshal(putData)
	if err != nil {
		t.Errorf(err.Error())
	}
	_, err = clientPut(ts.URL+"/api/v1/messages", string(j))
	if err != nil {
		t.Errorf(err.Error())
	}
	assertStatsEqual(t, ts.URL+"/api/v1/messages", 90, 100)

	// mark first 10 as read
	t.Log("Mark first 10 as unread")
	putData.Read = false
	j, err = json.Marshal(putData)
	if err != nil {
		t.Errorf(err.Error())
	}
	_, err = clientPut(ts.URL+"/api/v1/messages", string(j))
	if err != nil {
		t.Errorf(err.Error())
	}
	assertStatsEqual(t, ts.URL+"/api/v1/messages", 100, 100)

	// mark all as read
	putData.Read = true
	putData.IDs = []string{}
	j, err = json.Marshal(putData)
	if err != nil {
		t.Errorf(err.Error())
	}

	t.Log("Mark all read")
	_, err = clientPut(ts.URL+"/api/v1/messages", string(j))
	if err != nil {
		t.Errorf(err.Error())
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

	resp := apiv1.SendMessageConfirmation{}

	if err := json.Unmarshal(b, &resp); err != nil {
		t.Errorf(err.Error())
		return
	}

	t.Logf("Fetching response for message %s", resp.ID)
	msg, err := fetchMessage(ts.URL + "/api/v1/message/" + resp.ID)
	if err != nil {
		t.Errorf(err.Error())
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

	attachmentBytes, err := clientGet(ts.URL + "/api/v1/message/" + resp.ID + "/part/" + msg.Attachments[0].PartID)
	if err != nil {
		t.Errorf(err.Error())
	}
	assertEqual(t, `This is a plain text attachment`, string(attachmentBytes), "wrong Attachment content")
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
		t.Errorf(err.Error())
		return
	}

	if err := json.Unmarshal(data, &m); err != nil {
		t.Errorf(err.Error())
		return
	}

	assertEqual(t, float64(unread), m.Unread, "wrong unread count")
	assertEqual(t, float64(total), m.Total, "wrong total count")
}

func assertSearchEqual(t *testing.T, uri, query string, count int) {
	t.Logf("Test search: %s", query)
	m := apiv1.MessagesSummary{}

	limit := fmt.Sprintf("%d", count)

	data, err := clientGet(uri + "?query=" + url.QueryEscape(query) + "&limit=" + limit)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	if err := json.Unmarshal(data, &m); err != nil {
		t.Errorf(err.Error())
		return
	}

	assertEqual(t, float64(count), m.MessagesCount, "wrong search results count")
}

func insertEmailData(t *testing.T) {
	for i := 0; i < 100; i++ {
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

		if err := storage.SetMessageTags(id, []string{fmt.Sprintf("Test tag %03d", i)}); err != nil {
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

	defer resp.Body.Close()

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

	defer resp.Body.Close()

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

	defer resp.Body.Close()

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

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned status %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)

	return data, err
}

func assertEqual(t *testing.T, a interface{}, b interface{}, message string) {
	if a == b {
		return
	}
	message = fmt.Sprintf("%s: \"%v\" != \"%v\"", message, a, b)
	t.Fatal(message)
}
