package bench

import (
	"os"
	"testing"

	"github.com/flaviokalleu/germanio/runtime"
)

// carregar loads an application without its start-up messages, so the
// benchmark output stays parseable by benchstat.
func carregar(b *testing.B, file string) *runtime.App {
	b.Helper()
	stdout := os.Stdout
	null, err := os.Open(os.DevNull)
	if err == nil {
		os.Stdout = null
		defer func() { os.Stdout = stdout; null.Close() }()
	}
	app, err := runtime.Carregar(file, "0")
	if err != nil {
		b.Fatal(err)
	}
	return app
}
