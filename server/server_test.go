package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/server/apiv1"
	"github.com/axllent/mailpit/storage"
	"github.com/axllent/mailpit/utils/logger"
	"github.com/jhillyerd/enmime"
)

var (
	putDataStruct struct {
		Read bool     `json:"read"`
		IDs  []string `json:"ids"`
	}
)

func Test_APIv1(t *testing.T) {
	setup()
	defer storage.Close()

	r := defaultRoutes()

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

	// store this for later tests

	m, err = fetchMessages(ts.URL + "/api/v1/messages")
	if err != nil {
		t.Errorf(err.Error())
	}

	// read first 10
	t.Log("Read first 10 messages including raw & headers")
	putIDS := []string{}
	for indx, msg := range m.Messages {
		if indx == 10 {
			break
		}

		if _, err := clientGet(ts.URL + "/api/v1/message/" + msg.ID); err != nil {
			t.Errorf(err.Error())
		}

		// test RAW
		if _, err := clientGet(ts.URL + "/api/v1/message/" + msg.ID + "/raw"); err != nil {
			t.Errorf(err.Error())
		}

		// test headers
		if _, err := clientGet(ts.URL + "/api/v1/message/" + msg.ID + "/headers"); err != nil {
			t.Errorf(err.Error())
		}

		// store for later
		putIDS = append(putIDS, msg.ID)
	}
	assertStatsEqual(t, ts.URL+"/api/v1/messages", 90, 100)

	// mark first 10 as unread
	t.Log("Mark first 10 as unread")
	putData := putDataStruct
	putData.IDs = putIDS
	j, err := json.Marshal(putData)
	if err != nil {
		t.Errorf(err.Error())
	}
	_, err = clientPut(ts.URL+"/api/v1/messages", string(j))
	if err != nil {
		t.Errorf(err.Error())
	}
	assertStatsEqual(t, ts.URL+"/api/v1/messages", 100, 100)

	// mark first 10 as read
	t.Log("Mark first 10 as read")
	putData.Read = true
	j, err = json.Marshal(putData)
	if err != nil {
		t.Errorf(err.Error())
	}
	_, err = clientPut(ts.URL+"/api/v1/messages", string(j))
	if err != nil {
		t.Errorf(err.Error())
	}
	assertStatsEqual(t, ts.URL+"/api/v1/messages", 90, 100)

	// search
	assertSearchEqual(t, ts.URL+"/api/v1/search", "from-1@example.com", 1)
	assertSearchEqual(t, ts.URL+"/api/v1/search", "to:from-1@example.com", 0)
	assertSearchEqual(t, ts.URL+"/api/v1/search", "from:@example.com", 100)
	assertSearchEqual(t, ts.URL+"/api/v1/search", "subject:\"Subject line\"", 100)
	assertSearchEqual(t, ts.URL+"/api/v1/search", "subject:\"Subject line 17 end\"", 1)
	assertSearchEqual(t, ts.URL+"/api/v1/search", "!thisdoesnotexist", 100)
	assertSearchEqual(t, ts.URL+"/api/v1/search", "-thisdoesnotexist", 100)
	assertSearchEqual(t, ts.URL+"/api/v1/search", "thisdoesnotexist", 0)

	// delete first 10
	t.Log("Delete first 10")
	_, err = clientDelete(ts.URL+"/api/v1/messages", string(j))
	if err != nil {
		t.Errorf(err.Error())
	}
	assertStatsEqual(t, ts.URL+"/api/v1/messages", 90, 90)

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
	assertStatsEqual(t, ts.URL+"/api/v1/messages", 0, 90)

	// delete all
	t.Log("Delete all messages")
	_, err = clientDelete(ts.URL+"/api/v1/messages", "{}")
	if err != nil {
		t.Errorf("Expected nil, received %s", err.Error())
	}
	assertStatsEqual(t, ts.URL+"/api/v1/messages", 0, 0)
}

func setup() {
	logger.NoLogging = true
	config.MaxMessages = 0
	config.DataFile = ""

	if err := storage.InitDB(); err != nil {
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

	assertEqual(t, unread, m.Unread, "wrong unread count")
	assertEqual(t, total, m.Total, "wrong total count")
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

	assertEqual(t, count, m.MessagesCount, "wrong search results count")
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

		if _, err := storage.Store(buf.Bytes()); err != nil {
			t.Log("error ", err)
			t.Fail()
		}
	}

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

	data, err := ioutil.ReadAll(resp.Body)

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

	data, err := ioutil.ReadAll(resp.Body)

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

	data, err := ioutil.ReadAll(resp.Body)

	return data, err
}

func assertEqual(t *testing.T, a interface{}, b interface{}, message string) {
	if a == b {
		return
	}
	message = fmt.Sprintf("%s: \"%v\" != \"%v\"", message, a, b)
	t.Fatal(message)
}
