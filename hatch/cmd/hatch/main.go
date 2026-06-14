// Command hatch is the CLI entry point.
package main

import (
	"fmt"
	"os"

	"github.com/fioenix/overclaud/hatch/internal/cli"
)

func main() {
	if err := cli.NewRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "hatch: "+err.Error())
		os.Exit(1)
	}
}
