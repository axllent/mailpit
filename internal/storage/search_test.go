package storage

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"

	"github.com/axllent/mailpit/config"
	"github.com/jhillyerd/enmime"
)

func TestSearch(t *testing.T) {
	for _, tenantID := range []string{"", "MyServer 3", "host.example.com"} {
		tenantID = config.DBTenantID(tenantID)

		setup(tenantID)

		if tenantID == "" {
			t.Log("Testing search")
		} else {
			t.Logf("Testing search (tenant %s)", tenantID)
		}

		for i := 0; i < testRuns; i++ {
			msg := enmime.Builder().
				From(fmt.Sprintf("From %d", i), fmt.Sprintf("from-%d@example.com", i)).
				CC(fmt.Sprintf("CC %d", i), fmt.Sprintf("cc-%d@example.com", i)).
				CC(fmt.Sprintf("CC2 %d", i), fmt.Sprintf("cc2-%d@example.com", i)).
				Subject(fmt.Sprintf("Subject line %d end", i)).
				Text([]byte(fmt.Sprintf("This is the email body %d <jdsauk;dwqmdqw;>.", i))).
				To(fmt.Sprintf("To %d", i), fmt.Sprintf("to-%d@example.com", i)).
				To(fmt.Sprintf("To2 %d", i), fmt.Sprintf("to2-%d@example.com", i)).
				ReplyTo(fmt.Sprintf("Reply To %d", i), fmt.Sprintf("reply-to-%d@example.com", i))

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

			if _, err := Store(&bufBytes); err != nil {
				t.Log("error ", err)
				t.Fail()
			}
		}

		for i := 1; i < 51; i++ {
			// search a random something that will return a single result
			uniqueSearches := []string{
				fmt.Sprintf("from-%d@example.com", i),
				fmt.Sprintf("from:from-%d@example.com", i),
				fmt.Sprintf("to-%d@example.com", i),
				fmt.Sprintf("to:to-%d@example.com", i),
				fmt.Sprintf("to2-%d@example.com", i),
				fmt.Sprintf("to:to2-%d@example.com", i),
				fmt.Sprintf("cc-%d@example.com", i),
				fmt.Sprintf("cc:cc-%d@example.com", i),
				fmt.Sprintf("cc2-%d@example.com", i),
				fmt.Sprintf("cc:cc2-%d@example.com", i),
				fmt.Sprintf("reply-to-%d@example.com", i),
				fmt.Sprintf("reply-to:\"reply-to-%d@example.com\"", i),
				fmt.Sprintf("\"Subject line %d end\"", i),
				fmt.Sprintf("subject:\"Subject line %d end\"", i),
				fmt.Sprintf("\"the email body %d jdsauk dwqmdqw\"", i),
			}
			searchIdx := rand.Intn(len(uniqueSearches))

			search := uniqueSearches[searchIdx]

			summaries, _, err := Search(search, "", 0, 0, 100)
			if err != nil {
				t.Log("error ", err)
				t.Fail()
			}

			assertEqual(t, len(summaries), 1, "search result expected")

			assertEqual(t, summaries[0].From.Name, fmt.Sprintf("From %d", i), "\"From\" name does not match")
			assertEqual(t, summaries[0].From.Address, fmt.Sprintf("from-%d@example.com", i), "\"From\" address does not match")
			assertEqual(t, summaries[0].To[0].Name, fmt.Sprintf("To %d", i), "\"To\" name does not match")
			assertEqual(t, summaries[0].To[0].Address, fmt.Sprintf("to-%d@example.com", i), "\"To\" address does not match")
			assertEqual(t, summaries[0].Subject, fmt.Sprintf("Subject line %d end", i), "\"Subject\" does not match")
		}

		// search something that will return 200 results
		summaries, _, err := Search("This is the email body", "", 0, 0, testRuns)
		if err != nil {
			t.Log("error ", err)
			t.Fail()
		}
		assertEqual(t, len(summaries), testRuns, "search results expected")

		Close()
	}
}

func TestSearchDelete100(t *testing.T) {
	for _, tenantID := range []string{"", "MyServer 3", "host.example.com"} {
		tenantID = config.DBTenantID(tenantID)

		setup(tenantID)

		if tenantID == "" {
			t.Log("Testing search delete of 100 messages")
		} else {
			t.Logf("Testing search delete of 100 messages (tenant %s)", tenantID)
		}

		for i := 0; i < 100; i++ {
			if _, err := Store(&testTextEmail); err != nil {
				t.Log("error ", err)
				t.Fail()
			}
			if _, err := Store(&testMimeEmail); err != nil {
				t.Log("error ", err)
				t.Fail()
			}
		}

		_, total, err := Search("from:sender@example.com", "", 0, 0, 100)
		if err != nil {
			t.Log("error ", err)
			t.Fail()
		}

		assertEqual(t, total, 100, "100 search results expected")

		if err := DeleteSearch("from:sender@example.com", ""); err != nil {
			t.Log("error ", err)
			t.Fail()
		}

		_, total, err = Search("from:sender@example.com", "", 0, 0, 100)
		if err != nil {
			t.Log("error ", err)
			t.Fail()
		}

		assertEqual(t, total, 0, "0 search results expected")

		Close()
	}
}

func TestSearchDelete1100(t *testing.T) {
	setup("")
	defer Close()

	t.Log("Testing search delete of 1100 messages")
	for i := 0; i < 1100; i++ {
		if _, err := Store(&testTextEmail); err != nil {
			t.Log("error ", err)
			t.Fail()
		}
	}

	_, total, err := Search("from:sender@example.com", "", 0, 0, 100)
	if err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	assertEqual(t, total, 1100, "100 search results expected")

	if err := DeleteSearch("from:sender@example.com", ""); err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	_, total, err = Search("from:sender@example.com", "", 0, 0, 100)
	if err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	assertEqual(t, total, 0, "0 search results expected")
}

func TestEscPercentChar(t *testing.T) {
	tests := map[string]string{}
	tests["this is a test"] = "this is a test"
	tests["this is% a test"] = "this is%% a test"
	tests["this is%% a test"] = "this is%%%% a test"
	tests["this is%%% a test"] = "this is%%%%%% a test"
	tests["%this is% a test"] = "%%this is%% a test"
	tests["Ä"] = "Ä"
	tests["Ä%"] = "Ä%%"

	for search, expected := range tests {
		res := escPercentChar(search)
		assertEqual(t, res, expected, "no match")
	}
}

func TestSizeToBytes(t *testing.T) {
	tests := map[string]int64{}
	tests["1m"] = 1048576
	tests["1mb"] = 1048576
	tests["1 M"] = 1048576
	tests["1 MB"] = 1048576
	tests["1k"] = 1024
	tests["1kb"] = 1024
	tests["1 K"] = 1024
	tests["1 kB"] = 1024
	tests["1.5M"] = 1572864
	tests["1234567890"] = 1234567890
	tests["invalid"] = 0
	tests["1.2.3"] = 0
	tests["1.2.3M"] = 0

	for search, expected := range tests {
		res := sizeToBytes(search)
		assertEqual(t, res, expected, "size does not match")
	}
}
