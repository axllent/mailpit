package storage

import (
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/logger"
	"github.com/axllent/mailpit/server/websockets"
	"github.com/jhillyerd/enmime"
	"github.com/k3a/html2text"
	"github.com/ostafen/clover/v2"
)

// Return a header field as a []*mail.Address, or "null" is not found/empty
func addressToSlice(env *enmime.Envelope, key string) []*mail.Address {
	data, _ := env.AddressList(key)

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
	h := strings.TrimSpace(html2text.HTML2Text(env.HTML))
	if h != "" {
		b.WriteString(h + " ")
	} else {
		b.WriteString(env.Text + " ")
	}
	// add attachment filenames
	for _, a := range env.Attachments {
		b.WriteString(a.FileName + " ")
	}

	d := b.String()

	// remove/replace new lines
	re := regexp.MustCompile(`(\r?\n|\t|>|<|"|:|\,|;)`)
	d = re.ReplaceAllString(d, " ")
	// remove duplicate whitespace and trim
	d = strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(d)), " "))

	return d
}

// Auto-prune runs every 5 minutes to automatically delete oldest messages
// if total is greater than the threshold
func pruneCron() {
	for {
		// time.Sleep(5 * 60 * time.Second)
		time.Sleep(60 * time.Second)
		mailboxes, err := db.ListCollections()
		if err != nil {
			logger.Log().Errorf("[db] %s", err)
			continue
		}

		for _, m := range mailboxes {
			total, _ := db.Count(clover.NewQuery(m))
			if total > config.MaxMessages {
				limit := total - config.MaxMessages
				if limit > 5000 {
					limit = 5000
				}
				start := time.Now()
				if err := db.Delete(clover.NewQuery(m).
					Sort(clover.SortOption{Field: "Created", Direction: 1}).
					Limit(limit)); err != nil {
					logger.Log().Warnf("Error pruning %s: %s", m, err.Error())
					continue
				}
				elapsed := time.Since(start)
				logger.Log().Infof("Pruned %d messages from %s in %s", limit, m, elapsed)
				if !strings.HasSuffix(m, "_data") {
					websockets.Broadcast("prune", nil)
				}
			}
		}
	}
}
