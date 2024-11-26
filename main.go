package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/axllent/mailpit/cmd"
	sendmail "github.com/axllent/mailpit/sendmail/cmd"
)

func main() {
	exec, err := os.Executable()
	if err != nil {
		panic(err)
	}

	// running directly
	if normalize(filepath.Base(exec)) == normalize(filepath.Base(os.Args[0])) ||
		!strings.Contains(filepath.Base(os.Args[0]), "sendmail") {
		cmd.Execute()
	} else {
		// symlinked as "*sendmail*"
		sendmail.Run()
	}
}

// Normalize returns a lowercase string stripped of the file extension (if exists).
// Used for detecting Windows commands which ignores letter casing and `.exe`.
// eg: "MaIlpIT.Exe" returns "mailpit"
func normalize(s string) string {
	s = strings.ToLower(s)

	return strings.TrimSuffix(s, filepath.Ext(s))
}
