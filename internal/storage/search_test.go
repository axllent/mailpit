package storage

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"net/mail"
	"slices"
	"testing"

	"github.com/axllent/mailpit/config"
	"github.com/jhillyerd/enmime/v2"
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

		for i := range testRuns {
			msg := enmime.Builder().
				From(fmt.Sprintf("From %d", i), fmt.Sprintf("from-%d@example.com", i)).
				CC(fmt.Sprintf("CC %d", i), fmt.Sprintf("cc-%d@example.com", i)).
				CC(fmt.Sprintf("CC2 %d", i), fmt.Sprintf("cc2-%d@example.com", i)).
				Subject(fmt.Sprintf("Subject line %d end", i)).
				Text(fmt.Appendf(nil, "This is the email body %d <jdsauk;dwqmdqw;>.", i)).
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

			if _, err := Store(&bufBytes, nil); err != nil {
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
			searchIdx := rand.IntN(len(uniqueSearches))

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

		for range 100 {
			if _, err := Store(&testTextEmail, nil); err != nil {
				t.Log("error ", err)
				t.Fail()
			}
			if _, err := Store(&testMimeEmail, nil); err != nil {
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
	for range 1100 {
		if _, err := Store(&testTextEmail, nil); err != nil {
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

func TestSearchLikeWildcards(t *testing.T) {
	setup("")
	defer Close()

	t.Log("Testing search with SQL LIKE wildcard characters")

	// The first message of each pair has a literal `_`, `%` or `\` where the
	// second has another character, so a search for the first must not match the second.
	type addrs struct{ from, to, cc, bcc, replyTo string }

	messages := []struct {
		addrs     addrs
		subject   string
		body      string
		messageID string
	}{
		{addrs{"from_a", "to_a", "cc_a", "blind_a", "reply_a"}, "Save 50% today", "ref abc_def", "<id_1@example.com>"},
		{addrs{"fromxa", "toxa", "ccxa", "blindxa", "replyxa"}, "Save 50 on everything today", "ref abcXdef", "<idx1@example.com>"},
		{addrs{"other1", "other1", "other1", "other1", "other1"}, `dir a\b`, "body", "<other1@example.com>"},
		{addrs{"other2", "other2", "other2", "other2", "other2"}, "dir ab", "body", "<other2@example.com>"},
	}

	for _, m := range messages {
		env, err := enmime.Builder().
			From("", m.addrs.from+"@example.com").
			To("", m.addrs.to+"@example.com").
			CC("", m.addrs.cc+"@example.com").
			Header("Bcc", m.addrs.bcc+"@example.com").
			ReplyTo("", m.addrs.replyTo+"@example.com").
			Subject(m.subject).
			Header("Message-Id", m.messageID).
			Text([]byte(m.body)).
			Build()
		if err != nil {
			t.Fatal(err)
		}

		buf := new(bytes.Buffer)
		if err := env.Encode(buf); err != nil {
			t.Fatal(err)
		}

		b := buf.Bytes()
		if _, err := Store(&b, nil); err != nil {
			t.Fatal(err)
		}
	}

	searches := []struct {
		search  string
		subject string
	}{
		{"from:from_a", "Save 50% today"},
		{"to:to_a", "Save 50% today"},
		{"cc:cc_a", "Save 50% today"},
		{"bcc:blind_a", "Save 50% today"},
		{"reply-to:reply_a", "Save 50% today"},
		{"addressed:from_a", "Save 50% today"},
		{"addressed:to_a", "Save 50% today"},
		{"addressed:cc_a", "Save 50% today"},
		{"addressed:blind_a", "Save 50% today"},
		{"addressed:reply_a", "Save 50% today"},
		{"message-id:id_1", "Save 50% today"},
		{`subject:"50% today"`, "Save 50% today"},
		{"abc_def", "Save 50% today"},
		{`"50% today"`, "Save 50% today"},
		{`subject:a\b`, `dir a\b`},
	}

	for _, s := range searches {
		// the search must match only the literal message
		summaries, _, err := Search(s.search, "", 0, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		if len(summaries) != 1 || summaries[0].Subject != s.subject {
			t.Errorf("search %s: expected [%s], got %q", s.search, s.subject, subjects(summaries))
		}

		// the negated search must exclude only the literal message
		summaries, _, err = Search("-"+s.search, "", 0, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		if len(summaries) != len(messages)-1 || slices.Contains(subjects(summaries), s.subject) {
			t.Errorf("search -%s: expected all but [%s], got %q", s.search, s.subject, subjects(summaries))
		}
	}

	// deleting by search must only delete the literal match
	if err := DeleteSearch("to:to_a", ""); err != nil {
		t.Fatal(err)
	}
	if n := CountTotal(); n != uint64(len(messages)-1) {
		t.Errorf("DeleteSearch to:to_a: expected %d messages left, got %d", len(messages)-1, n)
	}
}

func TestSearchJSONEscapedAddresses(t *testing.T) {
	setup("")
	defer Close()

	t.Log("Testing address searches for characters escaped in the stored JSON")

	// Addresses are searched in the message metadata JSON, where `&` is stored
	// as `\u0026` and `\` as `\\`. The second message of each pair differs
	// only by that character, so a search for the first must not match it.
	type addrs struct{ from, to, cc, bcc, replyTo string }

	messages := []struct {
		addrs   addrs
		subject string
	}{
		{addrs{"From & Co", "To & Co", "Cc & Co", "Bcc & Co", "Reply & Co"}, "ampersand"},
		{addrs{"From and Co", "To and Co", "Cc and Co", "Bcc and Co", "Reply and Co"}, "no ampersand"},
		{addrs{`From a\b`, `To a\b`, `Cc a\b`, `Bcc a\b`, `Reply a\b`}, "backslash"},
		{addrs{"From ab", "To ab", "Cc ab", "Bcc ab", "Reply ab"}, "no backslash"},
	}

	for i, m := range messages {
		env, err := enmime.Builder().
			From(m.addrs.from, fmt.Sprintf("from%d@example.com", i)).
			To(m.addrs.to, fmt.Sprintf("to%d@example.com", i)).
			CC(m.addrs.cc, fmt.Sprintf("cc%d@example.com", i)).
			Header("Bcc", (&mail.Address{Name: m.addrs.bcc, Address: fmt.Sprintf("bcc%d@example.com", i)}).String()).
			ReplyTo(m.addrs.replyTo, fmt.Sprintf("reply%d@example.com", i)).
			Subject(m.subject).
			Text([]byte("body")).
			Build()
		if err != nil {
			t.Fatal(err)
		}

		buf := new(bytes.Buffer)
		if err := env.Encode(buf); err != nil {
			t.Fatal(err)
		}

		b := buf.Bytes()
		if _, err := Store(&b, nil); err != nil {
			t.Fatal(err)
		}
	}

	searches := []struct {
		search  string
		subject string
	}{
		{`from:"from & co"`, "ampersand"},
		{`to:"to & co"`, "ampersand"},
		{`cc:"cc & co"`, "ampersand"},
		{`bcc:"bcc & co"`, "ampersand"},
		{`reply-to:"reply & co"`, "ampersand"},
		{`addressed:"from & co"`, "ampersand"},
		{`addressed:"to & co"`, "ampersand"},
		{`addressed:"cc & co"`, "ampersand"},
		{`addressed:"bcc & co"`, "ampersand"},
		{`addressed:"reply & co"`, "ampersand"},
		{`from:a\b`, "backslash"},
		{`to:a\b`, "backslash"},
		{`cc:a\b`, "backslash"},
		{`bcc:a\b`, "backslash"},
		{`reply-to:a\b`, "backslash"},
		{`addressed:a\b`, "backslash"},
	}

	for _, s := range searches {
		summaries, _, err := Search(s.search, "", 0, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		if len(summaries) != 1 || summaries[0].Subject != s.subject {
			t.Errorf("search %s: expected [%s], got %q", s.search, s.subject, subjects(summaries))
		}

		summaries, _, err = Search("-"+s.search, "", 0, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		if len(summaries) != len(messages)-1 || slices.Contains(subjects(summaries), s.subject) {
			t.Errorf("search -%s: expected all but [%s], got %q", s.search, s.subject, subjects(summaries))
		}
	}

	// deleting by a negated search must keep the matching message
	if err := DeleteSearch(`-from:"from & co"`, ""); err != nil {
		t.Fatal(err)
	}
	if n := CountTotal(); n != 1 {
		t.Errorf(`DeleteSearch -from:"from & co": expected 1 message left, got %d`, n)
	}
}

func subjects(summaries []MessageSummary) []string {
	s := []string{}
	for _, m := range summaries {
		s = append(s, m.Subject)
	}
	return s
}

func TestEscLikeChars(t *testing.T) {
	tests := map[string]string{}
	tests["this is a test"] = "this is a test"
	tests["this is% a test"] = `this is\% a test`
	tests["this is%% a test"] = `this is\%\% a test`
	tests["this is%%% a test"] = `this is\%\%\% a test`
	tests["%this is% a test"] = `\%this is\% a test`
	tests["john_doe"] = `john\_doe`
	tests[`a\b`] = `a\\b`
	tests[`a\%_b`] = `a\\\%\_b`
	tests["Ä"] = "Ä"
	tests["Ä%"] = `Ä\%`

	for search, expected := range tests {
		res := escLikeChars(search)
		assertEqual(t, res, expected, "no match")
	}
}

func TestEscJSONChars(t *testing.T) {
	tests := map[string]string{}
	tests["john doe"] = "john doe"
	tests["marks & spencer"] = `marks \u0026 spencer`
	tests[`a\b`] = `a\\b`
	tests["john_doe%"] = "john_doe%"
	tests["Ä"] = "Ä"

	for search, expected := range tests {
		res := escJSONChars(search)
		assertEqual(t, res, expected, "no match")
	}
}

func TestSizeToBytes(t *testing.T) {
	tests := map[string]uint64{}
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
