// The historical entry point preserves Flang CLI command names and behavior.
package main

import (
	"github.com/flaviokalleu/germanio/cli"
	"os"
)

func main() { cli.Run(os.Args) }
