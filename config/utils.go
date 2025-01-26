package config

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/axllent/mailpit/internal/tools"
)

// IsFile returns whether a file exists and is readable
func isFile(path string) bool {
	f, err := os.Open(filepath.Clean(path))
	defer f.Close()
	return err == nil
}

// IsDir returns whether a path is a directory
func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil || os.IsNotExist(err) || !info.IsDir() {
		return false
	}

	return true
}

func isValidURL(s string) bool {
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return false
	}

	return strings.HasPrefix(u.Scheme, "http")
}

// DBTenantID converts a tenant ID to a DB-friendly value if set
func DBTenantID(s string) string {
	s = tools.Normalize(s)
	if s != "" {
		re := regexp.MustCompile(`[^a-zA-Z0-9\_]`)
		s = re.ReplaceAllString(s, "_")
		if !strings.HasSuffix(s, "_") {
			s = s + "_"
		}
	}

	return s
}
