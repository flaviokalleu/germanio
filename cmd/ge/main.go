package main

import (
	"github.com/flaviokalleu/germanio/tooling/gecli"
	"os"
)

func main() { os.Exit(gecli.Run(os.Args, os.Stdin, os.Stdout, os.Stderr)) }
