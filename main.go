package main

import (
	"os"
	"path/filepath"

	"github.com/axllent/mailpit/cmd"
	sendmail "github.com/axllent/mailpit/sendmail/cmd"
)

func main() {
	exec, err := os.Executable()
	if err != nil {
		panic(err)
	}

	// running directly
	if filepath.Base(exec) == filepath.Base(os.Args[0]) {
		cmd.Execute()
	} else {
		// symlinked
		sendmail.Run()
	}
}
