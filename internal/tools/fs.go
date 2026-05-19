package tools

import (
	"os"
)

// IsFile returns whether a path exists and is a regular file.
// Symlinks are deliberately rejected to prevent following links to
// arbitrary files outside the intended location.
func IsFile(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

// IsDir returns whether a path is a directory
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}

	return true
}
