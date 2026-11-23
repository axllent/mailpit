// Command gen writes Mailpit's OpenAPI specification. Run it from the repository root: `go run ./internal/openapi/gen`.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/axllent/mailpit/internal/openapi"
)

func main() {
	// run from the module root, or give the output file
	out := filepath.Join("server", "ui", "api", "v1", "openapi.json")
	if len(os.Args) > 1 {
		out = os.Args[1]
	}

	b, err := openapi.Generate()
	if err != nil {
		fmt.Fprintln(os.Stderr, "openapi:", err)
		os.Exit(1)
	}

	if err := os.WriteFile(out, b, 0o644); err != nil { //nolint:gosec // a public file
		fmt.Fprintln(os.Stderr, "openapi:", err)
		os.Exit(1)
	}

	fmt.Println("wrote", out)
}
