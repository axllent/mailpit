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
	"strconv"
	"strings"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/axllent/mailpit/server/apiv1"
)

// httpClient bounds each remote request so a slow or hostile --http endpoint
// cannot hang the dump indefinitely.
var httpClient = &http.Client{Timeout: time.Minute}

// maxSummarySize caps the bytes read from the remote messages-summary endpoint
// to prevent a hostile server from exhausting memory via an unbounded response.
const maxSummarySize = 20 * 1024 * 1024 // 20 MiB

// pageSize is the per-request limit when paging through the remote messages
// summary endpoint.
const pageSize = 10000

var (
	linkRe = regexp.MustCompile(`(?i)^https?:\/\/`)

	// idRe matches a valid Mailpit message ID (alphanumeric or dash, 8–60 chars).
	idRe = regexp.MustCompile(`^[a-zA-Z0-9-]{8,60}$`)

	outDir string

	// Base URL of mailpit instance
	base string

	// URL is the base URL of a remove Mailpit instance
	URL string

	dumpIDs = make(map[string]struct {
		Timestamp time.Time
	})
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

		start := 0
		var total uint64
		for {
			data, err := fetchSummaryPage(start)
			if err != nil {
				return err
			}

			if start == 0 {
				total = data.Total
			}

			for _, m := range data.Messages {
				dumpIDs[m.ID] = struct {
					Timestamp time.Time
				}{Timestamp: m.Created}
			}

			logger.Log().Debugf("Fetched messages summary page start=%d size=%d (%d/%d)", start, len(data.Messages), len(dumpIDs), total)

			// stop on empty page to guard against stale/inconsistent Total
			if len(data.Messages) == 0 {
				break
			}

			if uint64(len(dumpIDs)) >= total {
				break
			}

			start += pageSize
		}

	} else {
		// make sure the database isn't pruned while open
		config.MaxMessages = 0

		// local database
		if err := storage.InitDB(); err != nil {
			return err
		}

		logger.Log().Debugf("Fetching messages summary from %s", config.Database)

		start := 0
		for {
			page, err := storage.List(start, 0, pageSize)
			if err != nil {
				return err
			}

			for _, m := range page {
				dumpIDs[m.ID] = struct {
					Timestamp time.Time
				}{Timestamp: m.Created}
			}

			if len(page) < pageSize {
				break
			}

			start += pageSize
		}
	}

	if len(dumpIDs) == 0 {
		return errors.New("no messages found")
	}

	return nil
}

// fetchSummaryPage fetches a single page of the remote messages summary,
// starting at the given offset.
func fetchSummaryPage(start int) (*apiv1.MessagesSummary, error) {
	url := base + "api/v1/messages?limit=" + strconv.Itoa(pageSize) + "&start=" + strconv.Itoa(start)
	res, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("error fetching messages summary: HTTP " + res.Status)
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, maxSummarySize+1))
	if err != nil {
		return nil, err
	}

	if int64(len(body)) > maxSummarySize {
		return nil, errors.New("messages summary exceeds size cap")
	}

	var data apiv1.MessagesSummary
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return &data, nil
}

func saveMessages() error {
	for id, m := range dumpIDs {
		if !idRe.MatchString(id) {
			logger.Log().Errorf("skipping message with invalid ID: %q", id)
			continue
		}

		out := filepath.Join(outDir, id+".eml")

		// skip if message exists
		if tools.IsFile(out) {
			continue
		}

		var b []byte

		limit := int64(config.MaxMessageSize) * 1024 * 1024

		if base != "" {
			res, err := httpClient.Get(base + "api/v1/message/" + id + "/raw")

			if err != nil {
				logger.Log().Errorf("error fetching message %s: %s", id, err.Error())
				continue
			}

			if res.StatusCode != http.StatusOK {
				res.Body.Close()
				logger.Log().Errorf("error fetching message %s: HTTP %d", id, res.StatusCode)
				continue
			}

			if config.MaxMessageSize > 0 {
				b, err = io.ReadAll(io.LimitReader(res.Body, limit+1))
				res.Body.Close()

				if err != nil {
					logger.Log().Errorf("error fetching message %s: %s", id, err.Error())
					continue
				}

				if int64(len(b)) > limit {
					logger.Log().Warnf("message %s exceeds %d MiB size cap, skipping", id, config.MaxMessageSize)
					continue
				}
			} else {
				b, err = io.ReadAll(res.Body)
				res.Body.Close()

				if err != nil {
					logger.Log().Errorf("error fetching message %s: %s", id, err.Error())
					continue
				}
			}
		} else {
			var err error
			b, err = storage.GetMessageRaw(id)
			if err != nil {
				logger.Log().Errorf("error fetching message %s: %s", id, err.Error())
				continue
			}

			if config.MaxMessageSize > 0 && int64(len(b)) > limit {
				logger.Log().Warnf("message %s exceeds %d MiB size cap, skipping", id, config.MaxMessageSize)
				continue
			}
		}

		if err := os.WriteFile(out, b, 0644); /* #nosec */ err != nil {
			logger.Log().Errorf("error writing message %s: %s", id, err.Error())
			continue
		}

		_ = os.Chtimes(out, m.Timestamp, m.Timestamp)

		logger.Log().Debugf("Saved message %s to %s", id, out)
	}

	return nil
}
