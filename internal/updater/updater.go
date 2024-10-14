// package Updater checks and downloads new versions
package updater

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/semver"
)

var (
	// AllowPrereleases defines whether pre-releases may be included
	AllowPrereleases = false

	// temporary directory
	tempDir string
)

// Releases struct for Github releases json
type Releases []struct {
	Name       string `json:"name"`       // release name
	Tag        string `json:"tag_name"`   // release tag
	Prerelease bool   `json:"prerelease"` // Github pre-release
	Assets     []struct {
		BrowserDownloadURL string `json:"browser_download_url"`
		ID                 int64  `json:"id"`
		Name               string `json:"name"`
		Size               int64  `json:"size"`
	} `json:"assets"`
}

// Release struct contains the file data for downloadable release
type Release struct {
	Name string
	Tag  string
	URL  string
	Size int64
}

// GithubLatest fetches the latest release info & returns release tag, filename & download url
func GithubLatest(repo, name string) (string, string, string, error) {
	releaseURL := fmt.Sprintf("https://api.github.com/repos/%s/releases", repo)

	timeout := time.Duration(5 * time.Second)

	client := http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequest("GET", releaseURL, nil)
	if err != nil {
		return "", "", "", err
	}

	req.Header.Set("User-Agent", "Mailpit/"+config.Version)

	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return "", "", "", err
	}

	linkOS := runtime.GOOS
	linkArch := runtime.GOARCH
	linkExt := ".tar.gz"
	if linkOS == "windows" {
		// Windows uses .zip instead
		linkExt = ".zip"
	}

	var allReleases = []Release{}

	var releases Releases

	if err := json.Unmarshal(body, &releases); err != nil {
		return "", "", "", err
	}

	archiveName := fmt.Sprintf("%s-%s-%s%s", name, linkOS, linkArch, linkExt)

	// loop through releases
	for _, r := range releases {
		if !semver.IsValid(r.Tag) {
			// Invalid semversion, skip
			continue
		}

		if !AllowPrereleases && (semver.Prerelease(r.Tag) != "" || r.Prerelease) {
			// we don't accept AllowPrereleases, skip
			continue
		}

		for _, a := range r.Assets {
			if a.Name == archiveName {
				thisRelease := Release{a.Name, r.Tag, a.BrowserDownloadURL, a.Size}
				allReleases = append(allReleases, thisRelease)
				break
			}
		}
	}

	if len(allReleases) == 0 {
		// no releases with suitable assets found
		return "", "", "", fmt.Errorf("No binary releases found")
	}

	var latestRelease = Release{}

	for _, r := range allReleases {
		// detect the latest release
		if semver.Compare(r.Tag, latestRelease.Tag) == 1 {
			latestRelease = r
		}
	}

	return latestRelease.Tag, latestRelease.Name, latestRelease.URL, nil
}

// GreaterThan compares the current version to a different version
// returning < 1 not upgradeable
func GreaterThan(toVer, fromVer string) bool {
	return semver.Compare(toVer, fromVer) == 1
}

// GithubUpdate the running binary with the latest release binary from Github
func GithubUpdate(repo, appName, currentVersion string) (string, error) {
	ver, filename, downloadURL, err := GithubLatest(repo, appName)

	if err != nil {
		return "", err
	}

	if ver == currentVersion {
		return "", fmt.Errorf("No new release found")
	}

	if semver.Compare(ver, currentVersion) < 1 {
		return "", fmt.Errorf("No newer releases found (latest %s)", ver)
	}

	tmpDir := getTempDir()

	// outFile can be a tar.gz or a zip, depending on architecture
	outFile := filepath.Join(tmpDir, filename)

	if err := downloadToFile(downloadURL, outFile); err != nil {
		return "", err
	}

	newExec := filepath.Join(tmpDir, "mailpit")

	if runtime.GOOS == "windows" {
		if _, err := Unzip(outFile, tmpDir); err != nil {
			return "", err
		}
		newExec = filepath.Join(tmpDir, "mailpit.exe")
	} else {
		if err := TarGZExtract(outFile, tmpDir); err != nil {
			return "", err
		}
	}

	if runtime.GOOS != "windows" {
		err := os.Chmod(newExec, 0755) // #nosec
		if err != nil {
			return "", err
		}
	}

	// ensure the new binary is executable (mainly for inconsistent darwin builds)
	/* #nosec G204 */
	cmd := exec.Command(newExec, "-h")
	if err := cmd.Run(); err != nil {
		return "", err
	}

	// get the running binary
	oldExec, err := os.Executable()
	if err != nil {
		return "", err
	}

	if err = replaceFile(oldExec, newExec); err != nil {
		return "", err
	}

	return ver, nil
}

// DownloadToFile downloads a URL to a file
func downloadToFile(url, fileName string) error {
	// Get the data
	resp, err := http.Get(url) // #nosec
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath.Clean(fileName))
	if err != nil {
		return err
	}

	defer func() {
		if err := out.Close(); err != nil {
			logger.Log().Errorf("error closing file: %s", err.Error())
		}
	}()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)

	return err
}

// ReplaceFile replaces one file with another.
// Running files cannot be overwritten, so it has to be moved
// and the new binary saved to the original path. This requires
// read & write permissions to both the original file and directory.
// Note, on Windows it is not possible to delete a running program,
// so the old exe is renamed and moved to os.TempDir()
func replaceFile(dst, src string) error {
	// open the source file for reading
	source, err := os.Open(filepath.Clean(src))
	if err != nil {
		return err
	}

	// destination directory eg: /usr/local/bin
	dstDir := filepath.Dir(dst)
	// binary filename
	binaryFilename := filepath.Base(dst)
	// old binary tmp name
	dstOld := fmt.Sprintf("%s.old", binaryFilename)
	// new binary tmp name
	dstNew := fmt.Sprintf("%s.new", binaryFilename)
	// absolute path of new tmp file
	newTmpAbs := filepath.Join(dstDir, dstNew)
	// absolute path of old tmp file
	oldTmpAbs := filepath.Join(dstDir, dstOld)

	// get src permissions
	fi, _ := os.Stat(dst)
	srcPerms := fi.Mode().Perm()

	// create the new file
	tmpNew, err := os.OpenFile(filepath.Clean(newTmpAbs), os.O_CREATE|os.O_RDWR, srcPerms) // #nosec
	if err != nil {
		return err
	}

	// copy new binary to <binary>.new
	if _, err := io.Copy(tmpNew, source); err != nil {
		return err
	}

	// close immediately else Windows has a fit
	if err := tmpNew.Close(); err != nil {
		return err
	}

	if err := source.Close(); err != nil {
		return err
	}

	// rename the current executable to <binary>.old
	if err := os.Rename(dst, oldTmpAbs); err != nil {
		return err
	}

	// rename the <binary>.new to current executable
	if err := os.Rename(newTmpAbs, dst); err != nil {
		return err
	}

	// delete the old binary
	if runtime.GOOS == "windows" {
		tmpDir := os.TempDir()
		delFile := filepath.Join(tmpDir, filepath.Base(oldTmpAbs))
		if err := os.Rename(oldTmpAbs, delFile); err != nil {
			return err
		}
	} else {
		if err := os.Remove(oldTmpAbs); err != nil {
			return err
		}
	}

	// remove the src file
	return os.Remove(src)
}

// GetTempDir will create & return a temporary directory if one has not been specified
func getTempDir() string {
	if tempDir == "" {
		randBytes := make([]byte, 6)
		if _, err := rand.Read(randBytes); err != nil {
			panic(err)
		}
		tempDir = filepath.Join(os.TempDir(), "updater-"+hex.EncodeToString(randBytes))
	}
	if err := mkDirIfNotExists(tempDir); err != nil {
		// need a better way to exit
		logger.Log().Errorf("error: %s", err.Error())
		os.Exit(2)
	}

	return tempDir
}

// MkDirIfNotExists will create a directory if it doesn't exist
func mkDirIfNotExists(path string) error {
	if !isDir(path) {
		return os.MkdirAll(path, os.ModePerm) // #nosec
	}

	return nil
}

// IsDir returns if a path is a directory
func isDir(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) || !info.IsDir() {
		return false
	}

	return true
}
