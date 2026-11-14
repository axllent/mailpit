package storage

import (
	"encoding/json"
	"net/mail"
	"os"
	"strings"
	"sync"

	"github.com/axllent/mailpit/internal/html2text"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/mimeparser"
	"github.com/axllent/mailpit/internal/tools"
	enmimehtml2text "github.com/inbucket/html2text"
	"github.com/jhillyerd/enmime/v2"
	"golang.org/x/text/cases"
)

var (
	// for stats to prevent import cycle
	mu sync.RWMutex
	// StatsDeleted for counting the number of messages deleted
	StatsDeleted uint64

	// envelopeParser is a shared enmime parser instance used for message parsing
	envelopeParser = mimeparser.New()

	// storeParser is used when storing new messages. The plain text version of a HTML-only
	// message is not stored, so the (expensive) HTML to text down-conversion is skipped.
	storeParser = mimeparser.New(enmime.DisableTextConversion(true))
)

// AddTempFile adds a file to the slice of files to delete on exit
func AddTempFile(s string) {
	temporaryFiles = append(temporaryFiles, s)
}

// DeleteTempFiles will delete files added via AddTempFiles
func deleteTempFiles() {
	for _, f := range temporaryFiles {
		if err := os.Remove(f); err == nil {
			logger.Log().Debugf("removed temporary file: %s", f)
		}
	}
}

// Return a header field as a []*mail.Address, or "null" is not found/empty
func addressToSlice(env *enmime.Envelope, key string) []*mail.Address {
	data, err := env.AddressList(key)
	if err != nil || data == nil {
		return []*mail.Address{}
	}

	for _, a := range data {
		normalizeAddress(a)
	}

	return data
}

// normalizeAddress re-quotes the local-part of an email address when it
// contains characters not valid in a dot-atom (RFC 5321 §4.1.2).
// Go's mail.Address stores the decoded local-part, stripping quotes during
// parsing; this restores them so the Address field is a valid addr-spec.
//
// Rather than reimplementing RFC quoting rules, this delegates to
// mail.Address.String() which already handles quoting and escaping
// correctly, then extracts the addr-spec from the angle brackets.
func normalizeAddress(a *mail.Address) {
	s := a.String()
	start := strings.LastIndex(s, "<")
	end := strings.LastIndex(s, ">")
	if start >= 0 && end > start {
		a.Address = s[start+1 : end]
	}
}

// searchTextAndSnippet returns the search text and snippet of a message,
// stripping the HTML body (if any) only once for both
func searchTextAndSnippet(env *enmime.Envelope) (string, string) {
	if strings.TrimSpace(env.HTML) != "" {
		if plain, withLinks, err := html2text.StripTrimmedAndFull(env.HTML); err == nil {
			return createSearchText(env, withLinks), tools.SnippetFromStripped(plain)
		}
	}

	return createSearchText(env, ""), tools.CreateSnippet(envelopeText(env), env.HTML)
}

// envelopeText returns the plain text body of a message. Messages parsed with
// storeParser have no plain text body for HTML-only messages, so the HTML is
// down-converted here (as enmime does) for the rare cases it is needed.
func envelopeText(env *enmime.Envelope) string {
	if env.Text != "" || env.HTML == "" {
		return env.Text
	}

	text, err := enmimehtml2text.FromString(env.HTML)
	if err != nil {
		return ""
	}

	return text
}

// Generate the search text based on some header fields (to, from, subject etc)
// and either the stripped HTML body (if exists) or text body
func createSearchText(env *enmime.Envelope, strippedHTML string) string {
	var b strings.Builder

	_, _ = b.WriteString(env.GetHeader("From") + " ")
	_, _ = b.WriteString(env.GetHeader("Subject") + " ")
	_, _ = b.WriteString(env.GetHeader("To") + " ")
	_, _ = b.WriteString(env.GetHeader("Cc") + " ")
	_, _ = b.WriteString(env.GetHeader("Bcc") + " ")
	_, _ = b.WriteString(env.GetHeader("Reply-To") + " ")
	_, _ = b.WriteString(env.GetHeader("Return-Path") + " ")

	if strippedHTML != "" {
		_, _ = b.WriteString(strippedHTML + " ")
	} else {
		_, _ = b.WriteString(envelopeText(env) + " ")
	}
	// add attachment filenames
	for _, a := range env.Attachments {
		_, _ = b.WriteString(a.FileName + " ")
	}

	d := cleanString(b.String())

	return d
}

// CleanString removes unwanted characters from stored search text and search queries
func cleanString(str string) string {
	// replace \uFEFF with space, see https://github.com/golang/go/issues/42274#issuecomment-1017258184
	str = strings.ReplaceAll(str, string('\uFEFF'), " ")

	// remove/replace new lines, tabs and some punctuation
	str = strings.Map(func(r rune) rune {
		switch r {
		case '\n', '\t', '>', '<', '"', ',', ';', '(', ')':
			return ' '
		}
		return r
	}, str)

	// remove duplicate whitespace and trim
	return foldCase(strings.Join(strings.Fields(strings.TrimSpace(str)), " "))
}

// FoldCase normalises the case of stored search text and search terms so they can be
// compared case-insensitively, as SQLite's LIKE only folds ASCII case. Unicode case
// folding also matches characters which lowercasing alone does not (eg: `ß` & `SS`,
// Greek final sigma `ς` & `Σ`). The string is lowercased first so that Turkish `İ`
// matches `i` (case folding alone turns it into `i` + a combining dot).
//
// Stored text must be normalised with this function too, see refoldSearchText for
// existing messages.
func foldCase(s string) string {
	// a cases.Caser is stateful and must not be shared between goroutines
	return cases.Fold().String(strings.ToLower(s))
}

// metaSearchText is a case-folded copy (see foldCase) of the message Metadata plus the
// subject, stored as JSON in the mailbox MetaSearchText column for field searches (eg:
// `from:`). SQLite's LIKE only folds ASCII case, so the values are folded in Go (as is
// the search term) to make field searches case-insensitive for non-ASCII characters too.
// Username is not currently searchable, but is included so the column mirrors Metadata.
type metaSearchText struct {
	From     *mail.Address
	To       []*mail.Address
	Cc       []*mail.Address
	Bcc      []*mail.Address
	ReplyTo  []*mail.Address
	Username string
	Subject  string
}

// createMetaSearchText returns the case-folded MetaSearchText JSON for a message
func createMetaSearchText(meta Metadata, subject string) (string, error) {
	foldAddress := func(a *mail.Address) *mail.Address {
		if a == nil {
			return nil
		}
		return &mail.Address{Name: foldCase(a.Name), Address: foldCase(a.Address)}
	}

	foldAddresses := func(addresses []*mail.Address) []*mail.Address {
		if addresses == nil {
			return nil
		}
		l := make([]*mail.Address, len(addresses))
		for i, a := range addresses {
			l[i] = foldAddress(a)
		}
		return l
	}

	b, err := json.Marshal(metaSearchText{
		From:     foldAddress(meta.From),
		To:       foldAddresses(meta.To),
		Cc:       foldAddresses(meta.Cc),
		Bcc:      foldAddresses(meta.Bcc),
		ReplyTo:  foldAddresses(meta.ReplyTo),
		Username: foldCase(meta.Username),
		Subject:  foldCase(subject),
	})
	if err != nil {
		return "", err
	}

	return string(b), nil
}

// LogMessagesDeleted logs the number of messages deleted
func logMessagesDeleted(n int) {
	mu.Lock()
	StatsDeleted = StatsDeleted + tools.SafeUint64(n)
	mu.Unlock()
}

// IsFile returns whether a path is a file
func isFile(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) || !info.Mode().IsRegular() {
		return false
	}

	return true
}

// likeEscaper escapes the SQL LIKE wildcards `%` and `_` (and the escape character
// itself) so they match literally. Queries must use `LIKE ? ESCAPE '\'`.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// escLikeChars escapes a search term for use in a SQL LIKE pattern
func escLikeChars(s string) string {
	return likeEscaper.Replace(s)
}

// escJSONChars encodes a search term the way encoding/json stores it in the message
// metadata (eg: `&` as `\u0026` and `\` as `\\`), so it can be matched against
// the To, From, Cc, Bcc and ReplyTo JSON.
func escJSONChars(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return s
	}

	// remove the surrounding quotes
	return string(b[1 : len(b)-1])
}
