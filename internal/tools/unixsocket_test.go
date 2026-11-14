package tools

import (
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// socketDir returns a short directory, as the path of a unix socket is limited in length.
func socketDir(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "ps")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	return dir
}

func exists(path string) bool {
	_, err := os.Lstat(path)

	return err == nil
}

// A regular file or directory at the address used to be deleted.
func TestPrepareSocketLeavesFilesAlone(t *testing.T) {
	dir := socketDir(t)

	file := filepath.Join(dir, "important.txt")
	if err := os.WriteFile(file, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := PrepareSocket(file); err == nil || !strings.Contains(err.Error(), "not a socket") {
		t.Errorf("a regular file: error %v", err)
	}
	if b, err := os.ReadFile(file); err != nil || string(b) != "data" {
		t.Errorf("the regular file was changed: %q, %v", b, err)
	}

	emptyDir := filepath.Join(dir, "adir")
	if err := os.Mkdir(emptyDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := PrepareSocket(emptyDir); err == nil || !strings.Contains(err.Error(), "not a socket") {
		t.Errorf("a directory: error %v", err)
	}
	if !exists(emptyDir) {
		t.Error("the directory was removed")
	}

	// control: nothing there is fine
	if err := PrepareSocket(filepath.Join(dir, "missing.sock")); err != nil {
		t.Errorf("a path which does not exist: %v", err)
	}
}

// A socket file left behind by a process which is no longer running is removed.
func TestPrepareSocketRemovesStaleSocket(t *testing.T) {
	sock := filepath.Join(socketDir(t), "stale.sock")

	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	// closing the listener leaves the file behind, as when a process is killed
	ln.(*net.UnixListener).SetUnlinkOnClose(false)
	_ = ln.Close()

	if !exists(sock) {
		t.Fatal("the test socket file is not there")
	}
	if err := PrepareSocket(sock); err != nil {
		t.Fatalf("a stale socket: %v", err)
	}
	if exists(sock) {
		t.Error("the stale socket file was not removed")
	}
}

// A socket which is in use is left alone, and the connection used to check is closed.
func TestPrepareSocketKeepsActiveSocketAndClosesProbe(t *testing.T) {
	sock := filepath.Join(socketDir(t), "live.sock")

	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	result := make(chan error, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			result <- err
			return
		}
		defer c.Close()

		// the probe is closed by the client, which is seen here as the end of the stream
		_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
		_, err = io.ReadAll(c)
		result <- err
	}()

	if err := PrepareSocket(sock); err == nil || !strings.Contains(err.Error(), "already in use") {
		t.Fatalf("an active socket: error %v", err)
	}
	if !exists(sock) {
		t.Error("the active socket file was removed")
	}

	select {
	case err := <-result:
		if err != nil {
			t.Errorf("the connection used to check was not closed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("the connection used to check was not closed")
	}
}

// A socket which cannot be connected to (not allowed) is not known to be stale, so it is not removed.
// It used to be removed, which would take a socket which is in use from whoever is using it.
func TestPrepareSocketKeepsSocketItCannotCheck(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root is allowed to connect to any socket")
	}

	sock := filepath.Join(socketDir(t), "private.sock")

	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	if err := os.Chmod(sock, 0); err != nil {
		t.Fatal(err)
	}

	if err := PrepareSocket(sock); err == nil || !strings.Contains(err.Error(), "unable to tell") {
		t.Errorf("a socket which cannot be connected to: error %v", err)
	}
	if !exists(sock) {
		t.Error("the socket was removed")
	}
}
