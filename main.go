package main

import (
	"os"
	"path/filepath"
	"strings"

	legacy "github.com/flaviokalleu/germanio/cli"
	"github.com/flaviokalleu/germanio/tooling/gecli"
)

func main() {
	// Existing installers build the root entry point as germanio/germanio.exe.
	if strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe") == "germanio" {
		legacy.Run(os.Args)
		return
	}
	os.Exit(gecli.Run(os.Args, os.Stdin, os.Stdout, os.Stderr))
}
