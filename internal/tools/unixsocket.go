package tools

import (
	"fmt"
	"io/fs"
	"net"
	"os"
	"path"
	"regexp"
	"strconv"
)

// UnixSocket returns a path and a FileMode if the address is in
// the format of unix:<path>:<permission>
func UnixSocket(address string) (string, fs.FileMode, bool) {
	re := regexp.MustCompile(`^unix:(.*):(\d\d\d\d?)$`)

	var f fs.FileMode

	if !re.MatchString(address) {
		return "", f, false
	}

	m := re.FindAllStringSubmatch(address, 1)

	modeVal, err := strconv.ParseUint(m[0][2], 8, 32)

	if err != nil {
		return "", f, false
	}

	return path.Clean(m[0][1]), fs.FileMode(modeVal), true
}

// PrepareSocket returns an error if an active socket file already exists
func PrepareSocket(address string) error {
	address = path.Clean(address)
	if _, err := os.Stat(address); os.IsNotExist(err) {
		// does not exist, OK
		return nil
	}

	if _, err := net.Dial("unix", address); err == nil {
		// socket is listening
		return fmt.Errorf("socket already in use: %s", address)
	}

	return os.Remove(address)
}
