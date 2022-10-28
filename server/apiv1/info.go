package apiv1

import (
	"encoding/json"
	"net/http"
	"os"
	"runtime"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/storage"
	"github.com/axllent/mailpit/utils/updater"
)

type appVersion struct {
	Version       string
	LatestVersion string
	Database      string
	DatabaseSize  int64
	Messages      int
	Memory        uint64
}

// AppInfo returns some basic details about the running app, and latest release.
func AppInfo(w http.ResponseWriter, r *http.Request) {

	info := appVersion{}
	info.Version = config.Version

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	info.Memory = m.Sys - m.HeapReleased

	latest, _, _, err := updater.GithubLatest(config.Repo, config.RepoBinaryName)
	if err == nil {
		info.LatestVersion = latest
	}

	info.Database = config.DataFile

	db, err := os.Stat(info.Database)
	if err == nil {
		info.DatabaseSize = db.Size()
	}

	info.Messages = storage.CountTotal()

	bytes, _ := json.Marshal(info)

	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(bytes)
}
