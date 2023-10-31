package apiv1

import (
	"encoding/json"
	"net/http"
	"os"
	"runtime"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/internal/updater"
)

// Response includes the current and latest Mailpit version, database info, and memory usage
//
// swagger:model AppInformation
type appInformation struct {
	// Current Mailpit version
	Version string
	// Latest Mailpit version
	LatestVersion string
	// Database path
	Database string
	// Database size in bytes
	DatabaseSize int64
	// Total number of messages in the database
	Messages int
	// Current memory usage in bytes
	Memory uint64
}

// AppInfo returns some basic details about the running app, and latest release.
func AppInfo(w http.ResponseWriter, _ *http.Request) {
	// swagger:route GET /api/v1/info application AppInformation
	//
	// # Get application information
	//
	// Returns basic runtime information, message totals and latest release version.
	//
	//	Produces:
	//	- application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//		200: InfoResponse
	//		default: ErrorResponse
	info := appInformation{}
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
