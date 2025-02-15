package tools

import (
	"os"
	"path/filepath"
)

// IsFile returns whether a file exists and is readable
func IsFile(path string) bool {
	f, err := os.Open(filepath.Clean(path))
	defer f.Close()
	return err == nil
}

// IsDir returns whether a path is a directory
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil || os.IsNotExist(err) || !info.IsDir() {
		return false
	}

	return true
}
