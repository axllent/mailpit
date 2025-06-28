// Package dump is used to export all messages from mailpit into a directory
package dump

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/axllent/mailpit/server/apiv1"
)

var (
	linkRe = regexp.MustCompile(`(?i)^https?:\/\/`)

	outDir string

	// Base URL of mailpit instance
	base string

	// URL is the base URL of a remove Mailpit instance
	URL string

	summary = []storage.MessageSummary{}
)

// Sync will sync all messages from the specified database or API to the specified output directory
func Sync(d string) error {

	outDir = path.Clean(d)

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
		res, err := http.Get(base + "api/v1/messages?limit=0")

		if err != nil {
			return err
		}

		body, err := io.ReadAll(res.Body)

		if err != nil {
			return err
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
		out := path.Join(outDir, m.ID+".eml")

		// skip if message exists
		if tools.IsFile(out) {
			continue
		}

		var b []byte

		if base != "" {
			res, err := http.Get(base + "api/v1/message/" + m.ID + "/raw")

			if err != nil {
				logger.Log().Errorf("error fetching message %s: %s", m.ID, err.Error())
				continue
			}

			b, err = io.ReadAll(res.Body)

			if err != nil {
				logger.Log().Errorf("error fetching message %s: %s", m.ID, err.Error())
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
