package tools

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path"
	"regexp"
	"strconv"
	"time"
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

// PrepareSocket makes the address available for a new socket. It returns an error if there is an
// active socket there, if something cannot be told to be a stale socket, or if there is a file
// or directory which is not a socket (which is not removed). A stale socket file, which is left
// behind when Mailpit stops without removing it, is removed.
func PrepareSocket(address string) error {
	address = path.Clean(address)

	info, err := os.Lstat(address)
	if os.IsNotExist(err) {
		// does not exist, OK
		return nil
	}
	if err != nil {
		return fmt.Errorf("unable to check %s: %w", address, err)
	}

	if info.Mode().IsRegular() || info.IsDir() {
		return fmt.Errorf("%s exists and is not a socket", address)
	}

	conn, err := net.DialTimeout("unix", address, time.Second)
	if err == nil {
		_ = conn.Close()

		// socket is listening
		return fmt.Errorf("socket already in use: %s", address)
	}

	// A socket which is busy or not allowed to be used is not known to be stale, so it is left alone.
	var netErr net.Error
	if errors.Is(err, os.ErrPermission) || (errors.As(err, &netErr) && netErr.Timeout()) {
		return fmt.Errorf("unable to tell if socket %s is in use: %w", address, err)
	}

	return os.Remove(address)
}
