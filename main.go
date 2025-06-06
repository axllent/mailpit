// Package main is the entrypoint
package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/axllent/mailpit/cmd"
	sendmail "github.com/axllent/mailpit/sendmail/cmd"
)

func main() {
	// if the command executable contains "send" in the name (eg: sendmail), then run the sendmail command
	if strings.Contains(strings.ToLower(filepath.Base(os.Args[0])), "send") {
		sendmail.Run()
	} else {
		// else run mailpit
		cmd.Execute()
	}
}
