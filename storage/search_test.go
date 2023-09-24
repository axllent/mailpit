package storage

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"

	"github.com/jhillyerd/enmime"
)

func TestSearch(t *testing.T) {
	setup()
	defer Close()

	t.Log("Testing search")
	for i := 0; i < testRuns; i++ {
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

		if _, err := Store(buf.Bytes()); err != nil {
			t.Log("error ", err)
			t.Fail()
		}
	}

	for i := 1; i < 51; i++ {
		// search a random something that will return a single result
		searchIdx := rand.Intn(4) + 1
		var search string
		switch searchIdx {
		case 1:
			search = fmt.Sprintf("from-%d@example.com", i)
		case 2:
			search = fmt.Sprintf("to-%d@example.com", i)
		case 3:
			search = fmt.Sprintf("\"Subject line %d end\"", i)
		default:
			search = fmt.Sprintf("\"the email body %d jdsauk dwqmdqw\"", i)
		}

		summaries, _, err := Search(search, 0, 100)
		if err != nil {
			t.Log("error ", err)
			t.Fail()
		}

		assertEqual(t, len(summaries), 1, "1 search result expected")

		assertEqual(t, summaries[0].From.Name, fmt.Sprintf("From %d", i), "\"From\" name does not match")
		assertEqual(t, summaries[0].From.Address, fmt.Sprintf("from-%d@example.com", i), "\"From\" address does not match")
		assertEqual(t, summaries[0].To[0].Name, fmt.Sprintf("To %d", i), "\"To\" name does not match")
		assertEqual(t, summaries[0].To[0].Address, fmt.Sprintf("to-%d@example.com", i), "\"To\" address does not match")
		assertEqual(t, summaries[0].Subject, fmt.Sprintf("Subject line %d end", i), "\"Subject\" does not match")
	}

	// search something that will return 200 results
	summaries, _, err := Search("This is the email body", 0, testRuns)
	if err != nil {
		t.Log("error ", err)
		t.Fail()
	}
	assertEqual(t, len(summaries), testRuns, "search results expected")
}

func TestSearchDelete100(t *testing.T) {
	setup()
	defer Close()

	t.Log("Testing search delete of 100 messages")
	for i := 0; i < 100; i++ {
		if _, err := Store(testTextEmail); err != nil {
			t.Log("error ", err)
			t.Fail()
		}
		if _, err := Store(testMimeEmail); err != nil {
			t.Log("error ", err)
			t.Fail()
		}
	}

	_, total, err := Search("from:sender@example.com", 0, 100)
	if err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	assertEqual(t, total, 100, "100 search results expected")

	if err := DeleteSearch("from:sender@example.com"); err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	_, total, err = Search("from:sender@example.com", 0, 100)
	if err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	assertEqual(t, total, 0, "0 search results expected")
}

func TestSearchDelete1100(t *testing.T) {
	setup()
	defer Close()

	t.Log("Testing search delete of 1100 messages")
	for i := 0; i < 1100; i++ {
		if _, err := Store(testTextEmail); err != nil {
			t.Log("error ", err)
			t.Fail()
		}
	}

	_, total, err := Search("from:sender@example.com", 0, 100)
	if err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	assertEqual(t, total, 1100, "100 search results expected")

	if err := DeleteSearch("from:sender@example.com"); err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	_, total, err = Search("from:sender@example.com", 0, 100)
	if err != nil {
		t.Log("error ", err)
		t.Fail()
	}

	assertEqual(t, total, 0, "0 search results expected")
}
