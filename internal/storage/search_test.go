package storage

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"net/mail"
	"slices"
	"strings"
	"testing"
	"time"

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

// storeUnicodeTestMessage stores a message using name in every address field
func storeUnicodeTestMessage(t *testing.T, i int, name, subject string) {
	t.Helper()

	env, err := enmime.Builder().
		From(name, fmt.Sprintf("from%d@example.com", i)).
		To(name, fmt.Sprintf("to%d@example.com", i)).
		CC(name, fmt.Sprintf("cc%d@example.com", i)).
		Header("Bcc", (&mail.Address{Name: name, Address: fmt.Sprintf("bcc%d@example.com", i)}).String()).
		ReplyTo(name, fmt.Sprintf("reply%d@example.com", i)).
		Subject(subject).
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

// forEachTestTenant runs fn as a subtest against an empty database, once without a
// tenant and once for each of two tenant IDs
func forEachTestTenant(t *testing.T, fn func(t *testing.T)) {
	for _, tenantID := range []string{"", "MyServer 3", "host.example.com"} {
		name := tenantID
		if name == "" {
			name = "no tenant"
		}

		t.Run(name, func(t *testing.T) {
			setup(config.DBTenantID(tenantID))
			defer Close()

			fn(t)
		})
	}
}

func TestMetaSearchTextUnicodeCase(t *testing.T) {
	forEachTestTenant(t, testMetaSearchTextUnicodeCase)
}

func testMetaSearchTextUnicodeCase(t *testing.T) {
	t.Log("Testing field searches are case-insensitive for non-ASCII characters")

	// SQLite's LIKE only folds ASCII case, so field searches match against a
	// case-folded copy of the fields. Each message is matched by terms in any case,
	// whether the stored value is capitalised or lowercase. The ASCII message is a
	// control for the case-insensitivity LIKE already provided.
	messages := []struct {
		name    string
		subject string
		terms   []string
	}{
		{"Иван Петров", "Привет Мир", []string{"Иван", "иван", "ИВАН ПЕТРОВ", "иВАН пЕТРОВ"}},
		{"ärger öl", "äpfel", []string{"ärger", "Ärger", "ÄRGER ÖL"}},
		{"Jörg Straße", "Οδυσσεύς", []string{"STRASSE", "strasse", "Straße", "JÖRG STRAẞE"}},
		{"John Smith", "Hello World", []string{"john", "JOHN SMITH"}},
	}

	for i, m := range messages {
		storeUnicodeTestMessage(t, i, m.name, m.subject)
	}

	assertSearch := func(search string, expected []string) {
		t.Helper()
		summaries, _, err := Search(search, "", 0, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		got := subjects(summaries)
		slices.Sort(got)
		slices.Sort(expected)
		if !slices.Equal(got, expected) {
			t.Errorf("search %s: expected %q, got %q", search, expected, got)
		}
	}

	allSubjects := []string{}
	for _, m := range messages {
		allSubjects = append(allSubjects, m.subject)
	}

	for _, m := range messages {
		others := slices.DeleteFunc(slices.Clone(allSubjects), func(s string) bool { return s == m.subject })

		for _, prefix := range []string{"from:", "to:", "cc:", "bcc:", "reply-to:", "addressed:"} {
			for _, term := range m.terms {
				search := prefix + `"` + term + `"`
				assertSearch(search, []string{m.subject})
				assertSearch("-"+search, others)
			}
		}

		for _, term := range []string{m.subject, strings.ToLower(m.subject), strings.ToUpper(m.subject)} {
			search := `subject:"` + term + `"`
			assertSearch(search, []string{m.subject})
			assertSearch("-"+search, others)
		}
	}

	// control: a term that matches nothing still matches nothing
	assertSearch(`from:"ÖLIVER"`, []string{})
	assertSearch(`-from:"ÖLIVER"`, allSubjects)
}

func TestMetaSearchTextBackfill(t *testing.T) {
	forEachTestTenant(t, testMetaSearchTextBackfill)
}

func testMetaSearchTextBackfill(t *testing.T) {
	t.Log("Testing MetaSearchText is populated for existing messages")

	storeUnicodeTestMessage(t, 0, "Ärger John", "Äpfel")
	storeUnicodeTestMessage(t, 1, "Öl Jane", "Öl")

	// simulate messages stored before the MetaSearchText column existed
	if _, err := db.Exec(`UPDATE ` + tenant("mailbox") + ` SET MetaSearchText = NULL`); err != nil {
		t.Fatal(err)
	}

	assertCount := func(search string, expected int) {
		t.Helper()
		summaries, _, err := Search(search, "", 0, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		if len(summaries) != expected {
			t.Errorf("search %s: expected %d results, got %q", search, expected, subjects(summaries))
		}
	}

	pendingCount := func() int {
		t.Helper()
		var pending float64 // use float64 for rqlite compatibility
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + tenant("mailbox") + ` WHERE MetaSearchText IS NULL`).Scan(&pending); err != nil {
			t.Fatal(err)
		}
		return int(pending)
	}

	// before backfill, field searches fall back to the original metadata: ASCII is
	// case-insensitive, non-ASCII is not, and the unpopulated messages are not
	// dropped from negated searches
	assertCount(`from:"JOHN"`, 1)
	assertCount(`from:"ÄRGER"`, 0)
	assertCount(`subject:"ÄPFEL"`, 0)
	assertCount(`-from:"nobody"`, 2)

	// metadata that is valid JSON but cannot be unmarshalled into Metadata, so the
	// backfill must skip it rather than stop or select it forever (Search already
	// omits such messages from results)
	if _, err := db.Exec(`UPDATE `+tenant("mailbox")+` SET Metadata = '{"To":"invalid"}' WHERE Subject = ?`, "Öl"); err != nil {
		t.Fatal(err)
	}

	if err := backfillMetaSearchText(); err != nil {
		t.Fatal(err)
	}

	assertCount(`from:"JOHN"`, 1)
	assertCount(`from:"ÄRGER"`, 1)
	assertCount(`from:"ärger"`, 1)
	assertCount(`subject:"ÄPFEL"`, 1)
	assertCount(`-from:"ÄRGER"`, 0)
	assertCount(`-from:"nobody"`, 1)

	if n := pendingCount(); n != 1 {
		t.Errorf("expected 1 message without MetaSearchText, got %d", n)
	}

	// a second run terminates and leaves the unreadable message unpopulated
	if err := backfillMetaSearchText(); err != nil {
		t.Fatal(err)
	}
	if n := pendingCount(); n != 1 {
		t.Errorf("expected 1 message without MetaSearchText, got %d", n)
	}
}

func TestMetaSearchTextReindex(t *testing.T) {
	forEachTestTenant(t, testMetaSearchTextReindex)
}

func testMetaSearchTextReindex(t *testing.T) {
	t.Log("Testing reindexing regenerates MetaSearchText")

	storeUnicodeTestMessage(t, 0, "Ärger Öl", "Äpfel")

	// replace the stored value with one that no longer matches the message
	if _, err := db.Exec(`UPDATE ` + tenant("mailbox") + ` SET MetaSearchText = '{"From":{"Name":"stale","Address":"stale@example.com"},"Subject":"stale"}'`); err != nil {
		t.Fatal(err)
	}

	assertCount := func(search string, expected int) {
		t.Helper()
		summaries, _, err := Search(search, "", 0, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		if len(summaries) != expected {
			t.Errorf("search %s: expected %d results, got %q", search, expected, subjects(summaries))
		}
	}

	// searches use the stale value until the message is reindexed
	assertCount(`from:"stale"`, 1)
	assertCount(`from:"ÄRGER"`, 0)

	ReindexAll()

	assertCount(`from:"stale"`, 0)
	assertCount(`subject:"stale"`, 0)
	assertCount(`from:"ÄRGER"`, 1)
	assertCount(`from:"ärger öl"`, 1)
	assertCount(`subject:"ÄPFEL"`, 1)
}

func TestSearchTextCaseFolding(t *testing.T) {
	forEachTestTenant(t, testSearchTextCaseFolding)
}

func testSearchTextCaseFolding(t *testing.T) {
	t.Log("Testing general searches use case folding")

	storeUnicodeTestMessage(t, 0, "Jörg", "Straße Οδυσσεύς")
	// control: same shape, no characters affected by case folding
	storeUnicodeTestMessage(t, 1, "John", "Street Odysseus")

	assertCount := func(search string, expected int) {
		t.Helper()
		summaries, _, err := Search(search, "", 0, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		if len(summaries) != expected {
			t.Errorf("search %s: expected %d results, got %q", search, expected, subjects(summaries))
		}
	}

	for _, term := range []string{"straße", "STRASSE", "strasse", "STRAẞE", "οδυσσεύς", "ΟΔΥΣΣΕΎΣ", "οδυσσεύσ"} {
		assertCount(term, 1)
		assertCount("-"+term, 1)
	}

	assertCount("street", 1)
	assertCount("ODYSSEUS", 1)
	assertCount("strassen", 0)
}

func TestRefoldSearchText(t *testing.T) {
	forEachTestTenant(t, testRefoldSearchText)
}

func testRefoldSearchText(t *testing.T) {
	t.Log("Testing existing search text is case-folded")

	storeUnicodeTestMessage(t, 0, "Jörg", "Straße")
	storeUnicodeTestMessage(t, 1, "John", "Street")

	// simulate search text stored before case folding, when it was only lowercased
	const oldText = "jörg straße body"
	if _, err := db.Exec(`UPDATE `+tenant("mailbox")+` SET SearchText = ? WHERE Subject = ?`, oldText, "Straße"); err != nil {
		t.Fatal(err)
	}
	if err := SettingPut(searchTextFoldedSetting, ""); err != nil {
		t.Fatal(err)
	}

	searchText := func(subject string) string {
		t.Helper()
		var s string
		if err := db.QueryRow(`SELECT SearchText FROM `+tenant("mailbox")+` WHERE Subject = ?`, subject).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}

	assertCount := func(search string, expected int) {
		t.Helper()
		summaries, _, err := Search(search, "", 0, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		if len(summaries) != expected {
			t.Errorf("search %s: expected %d results, got %q", search, expected, subjects(summaries))
		}
	}

	// search terms are folded, so the unfolded text is not matched until refolded
	assertCount("STRASSE", 0)

	controlBefore := searchText("Street")

	if err := refoldSearchText(); err != nil {
		t.Fatal(err)
	}

	if got := searchText("Straße"); got != "jörg strasse body" {
		t.Errorf("expected refolded search text %q, got %q", "jörg strasse body", got)
	}
	if got := searchText("Street"); got != controlBefore {
		t.Errorf("expected control search text to be unchanged, got %q", got)
	}
	if SettingGet(searchTextFoldedSetting) != "1" {
		t.Errorf("expected %s setting to be set", searchTextFoldedSetting)
	}

	assertCount("STRASSE", 1)
	assertCount("straße", 1)

	// once complete, it does not run again
	if _, err := db.Exec(`UPDATE `+tenant("mailbox")+` SET SearchText = ? WHERE Subject = ?`, oldText, "Straße"); err != nil {
		t.Fatal(err)
	}
	if err := refoldSearchText(); err != nil {
		t.Fatal(err)
	}
	if got := searchText("Straße"); got != oldText {
		t.Errorf("expected search text to be left alone after completion, got %q", got)
	}
}

func TestSearchNegatedSizeAndDate(t *testing.T) {
	setup("")
	defer Close()

	useMessageDates := config.UseMessageDates
	config.UseMessageDates = true
	defer func() { config.UseMessageDates = useMessageDates }()

	t.Log("Testing negated size and date searches")

	// the second message is dated exactly at midnight, the boundary of the date searches below
	dates := []time.Time{
		time.Date(2025, 12, 31, 12, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	for i, date := range dates {
		env, err := enmime.Builder().
			From("", "sender@example.com").
			To("", "recipient@example.com").
			Subject(fmt.Sprintf("message %d", i)).
			Date(date).
			Text([]byte(strings.Repeat("x", (i+1)*100))).
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

	// use the size of the middle message, the boundary of the size searches below
	all, _, err := Search("", "", 0, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	size := uint64(0)
	for _, m := range all {
		if m.Subject == "message 1" {
			size = m.Size
		}
	}
	if size == 0 {
		t.Fatal("message 1 not found")
	}

	tests := []struct {
		search   string
		timezone string
		want     []string
	}{
		{fmt.Sprintf("larger:%d", size), "UTC", []string{"message 1", "message 2"}},
		{fmt.Sprintf("-larger:%d", size), "UTC", []string{"message 0"}},
		{fmt.Sprintf("smaller:%d", size), "UTC", []string{"message 0"}},
		{fmt.Sprintf("-smaller:%d", size), "UTC", []string{"message 1", "message 2"}},
		{"after:2026-01-01", "UTC", []string{"message 1", "message 2"}},
		{"-after:2026-01-01", "UTC", []string{"message 0"}},
		{"before:2026-01-01", "UTC", []string{"message 0"}},
		{"-before:2026-01-01", "UTC", []string{"message 1", "message 2"}},
		// adjacent ranges tile: each message is in exactly one, none are dropped or counted twice
		{"after:2025-12-31 before:2026-01-01", "UTC", []string{"message 0"}},
		{"after:2026-01-01 before:2026-01-02", "UTC", []string{"message 1", "message 2"}},
		// combined with a size filter, the control (message 0 is in the date range but too small)
		{fmt.Sprintf("larger:%d before:2026-01-02", size), "UTC", []string{"message 1", "message 2"}},
		// midnight in New York is 05:00 UTC
		{"after:2026-01-01", "America/New_York", []string{"message 2"}},
		{"-after:2026-01-01", "America/New_York", []string{"message 0", "message 1"}},
		{"before:2026-01-01", "America/New_York", []string{"message 0", "message 1"}},
		{"-before:2026-01-01", "America/New_York", []string{"message 2"}},
		// midnight in Tokyo is 15:00 UTC the day before
		{"after:2026-01-01", "Asia/Tokyo", []string{"message 1", "message 2"}},
		{"-after:2026-01-01", "Asia/Tokyo", []string{"message 0"}},
		{"before:2026-01-01", "Asia/Tokyo", []string{"message 0"}},
		{"-before:2026-01-01", "Asia/Tokyo", []string{"message 1", "message 2"}},
	}

	for _, test := range tests {
		if _, err := time.LoadLocation(test.timezone); err != nil {
			t.Fatalf("timezone %s not available: %s", test.timezone, err)
		}

		summaries, _, err := Search(test.search, test.timezone, 0, 0, 100)
		if err != nil {
			t.Fatal(err)
		}

		got := subjects(summaries)
		slices.Sort(got)
		if !slices.Equal(got, test.want) {
			t.Errorf("%s (%s) returned %v, expected %v", test.search, test.timezone, got, test.want)
		}
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
