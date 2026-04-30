// Command c7search is a CLI for the Context7 documentation API. See
// `c7search --help` for usage.
package main

import (
	"os"

	"github.com/kevin-burns/c7search/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
