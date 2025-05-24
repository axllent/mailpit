package storage

import (
	"net/mail"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/axllent/mailpit/internal/html2text"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/jhillyerd/enmime/v2"
)

var (
	// for stats to prevent import cycle
	mu sync.RWMutex
	// StatsDeleted for counting the number of messages deleted
	StatsDeleted uint64
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

	return data
}

// Generate the search text based on some header fields (to, from, subject etc)
// and either the stripped HTML body (if exists) or text body
func createSearchText(env *enmime.Envelope) string {
	var b strings.Builder

	b.WriteString(env.GetHeader("From") + " ")
	b.WriteString(env.GetHeader("Subject") + " ")
	b.WriteString(env.GetHeader("To") + " ")
	b.WriteString(env.GetHeader("Cc") + " ")
	b.WriteString(env.GetHeader("Bcc") + " ")
	b.WriteString(env.GetHeader("Reply-To") + " ")
	b.WriteString(env.GetHeader("Return-Path") + " ")

	h := html2text.Strip(env.HTML, true)
	if h != "" {
		b.WriteString(h + " ")
	} else {
		b.WriteString(env.Text + " ")
	}
	// add attachment filenames
	for _, a := range env.Attachments {
		b.WriteString(a.FileName + " ")
	}

	d := cleanString(b.String())

	return d
}

// CleanString removes unwanted characters from stored search text and search queries
func cleanString(str string) string {
	// replace \uFEFF with space, see https://github.com/golang/go/issues/42274#issuecomment-1017258184
	str = strings.ReplaceAll(str, string('\uFEFF'), " ")

	// remove/replace new lines
	re := regexp.MustCompile(`(\r?\n|\t|>|<|"|\,|;|\(|\))`)
	str = re.ReplaceAllString(str, " ")

	// remove duplicate whitespace and trim
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(str)), " "))
}

// LogMessagesDeleted logs the number of messages deleted
func logMessagesDeleted(n int) {
	mu.Lock()
	StatsDeleted = StatsDeleted + uint64(n)
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

// Convert `%` to `%%` for SQL searches
func escPercentChar(s string) string {
	return strings.ReplaceAll(s, "%", "%%")
}
