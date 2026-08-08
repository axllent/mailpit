// Package main is the entrypoint
package main

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"

	"github.com/axllent/mailpit/cmd"
	"github.com/axllent/mailpit/internal/licenses"
	sendmail "github.com/axllent/mailpit/sendmail/cmd"
)

//go:embed LICENSE
var mailpitLicense string

func main() {
	licenses.Mailpit = mailpitLicense

	// if the command executable contains "send" in the name (eg: sendmail), then run the sendmail command
	if strings.Contains(strings.ToLower(filepath.Base(os.Args[0])), "send") {
		sendmail.Run()
	} else {
		// else run mailpit
		cmd.Execute()
	}
}
