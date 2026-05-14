// Package dump is used to export all messages from mailpit into a directory
package dump

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/axllent/mailpit/server/apiv1"
)

// httpClient bounds each remote request so a slow or hostile --http endpoint
// cannot hang the dump indefinitely. Body size is independently capped by
// maxRawSize / maxSummarySize via io.LimitReader.
var httpClient = &http.Client{Timeout: time.Minute}

// maxRawSize caps the bytes read per remote message to prevent a hostile
// server from exhausting local disk via an unbounded response body.
const maxRawSize = 50 * 1024 * 1024 // 50 MiB

// maxSummarySize caps the bytes read from the remote messages-summary endpoint
// to prevent a hostile server from exhausting memory via an unbounded response.
const maxSummarySize = 1000 * 1024 * 1024 // 1000 MiB

var (
	linkRe = regexp.MustCompile(`(?i)^https?:\/\/`)

	// idRe matches a valid Mailpit message ID (alphanumeric or dash, 8–60 chars).
	idRe = regexp.MustCompile(`^[a-zA-Z0-9-]{8,60}$`)

	outDir string

	// Base URL of mailpit instance
	base string

	// URL is the base URL of a remove Mailpit instance
	URL string

	summary = []storage.MessageSummary{}
)

// Sync will sync all messages from the specified database or API to the specified output directory
func Sync(d string) error {

	outDir = filepath.Clean(d)

	if URL != "" {
		if !linkRe.MatchString(URL) {
			return errors.New("invalid URL")
		}

		base = strings.TrimRight(URL, "/") + "/"
	}

	if base == "" && config.Database == "" {
		return errors.New("no database or API URL specified")
	}

	if !tools.IsDir(outDir) {
		if err := os.MkdirAll(outDir, 0755); /* #nosec */ err != nil {
			return err
		}
	}

	if err := loadIDs(); err != nil {
		return err
	}

	if err := saveMessages(); err != nil {
		return err
	}

	return nil
}

// LoadIDs will load all message IDs from the specified database or API
func loadIDs() error {
	if base != "" {
		// remote
		logger.Log().Debugf("Fetching messages summary from %s", base)
		res, err := httpClient.Get(base + "api/v1/messages?limit=0")

		if err != nil {
			return err
		}

		if res.StatusCode != http.StatusOK {
			res.Body.Close()
			return errors.New("error fetching messages summary: HTTP " + res.Status)
		}

		body, err := io.ReadAll(io.LimitReader(res.Body, maxSummarySize+1))

		if err != nil {
			return err
		}

		if int64(len(body)) > maxSummarySize {
			return errors.New("messages summary exceeds size cap")
		}

		var data apiv1.MessagesSummary
		if err := json.Unmarshal(body, &data); err != nil {
			return err
		}

		summary = data.Messages

	} else {
		// make sure the database isn't pruned while open
		config.MaxMessages = 0

		var err error
		// local database
		if err = storage.InitDB(); err != nil {
			return err
		}

		logger.Log().Debugf("Fetching messages summary from %s", config.Database)

		summary, err = storage.List(0, 0, 0)
		if err != nil {
			return err
		}
	}

	if len(summary) == 0 {
		return errors.New("no messages found")
	}

	return nil
}

func saveMessages() error {
	for _, m := range summary {
		if !idRe.MatchString(m.ID) {
			logger.Log().Errorf("skipping message with invalid ID: %q", m.ID)
			continue
		}

		out := filepath.Join(outDir, m.ID+".eml")

		// skip if message exists
		if tools.IsFile(out) {
			continue
		}

		var b []byte

		if base != "" {
			res, err := httpClient.Get(base + "api/v1/message/" + m.ID + "/raw")

			if err != nil {
				logger.Log().Errorf("error fetching message %s: %s", m.ID, err.Error())
				continue
			}

			if res.StatusCode != http.StatusOK {
				res.Body.Close()
				logger.Log().Errorf("error fetching message %s: HTTP %d", m.ID, res.StatusCode)
				continue
			}

			b, err = io.ReadAll(io.LimitReader(res.Body, maxRawSize+1))
			res.Body.Close()

			if err != nil {
				logger.Log().Errorf("error fetching message %s: %s", m.ID, err.Error())
				continue
			}

			if len(b) > maxRawSize {
				logger.Log().Errorf("message %s exceeds size cap (%d bytes), skipping", m.ID, maxRawSize)
				continue
			}
		} else {
			var err error
			b, err = storage.GetMessageRaw(m.ID)
			if err != nil {
				logger.Log().Errorf("error fetching message %s: %s", m.ID, err.Error())
				continue
			}
		}

		if err := os.WriteFile(out, b, 0644); /* #nosec */ err != nil {
			logger.Log().Errorf("error writing message %s: %s", m.ID, err.Error())
			continue
		}

		_ = os.Chtimes(out, m.Created, m.Created)

		logger.Log().Debugf("Saved message %s to %s", m.ID, out)
	}

	return nil
}
