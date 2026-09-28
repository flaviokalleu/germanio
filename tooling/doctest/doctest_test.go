// Package doctest checks that the .ge code published in the documentation is
// real: every ```ge block in the public documents must compile, either as a
// complete application, as a fragment of one (compiled under a system
// declaration), or as a program of the strict core.
package doctest

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/flaviokalleu/germanio/compiler/parser"
	"github.com/flaviokalleu/germanio/runtime"
)

var block = regexp.MustCompile("(?s)```ge\n(.*?)```")

func publicDocs(t *testing.T) []string {
	files := []string{
		"../../README.md",
		"../../README.pt-BR.md",
		"../../CONTRIBUTING.md",
		"../../docs/getting-started.md",
		"../../docs/language-tour.md",
		"../../docs/comparisons.md",
		"../../docs/ARCHITECTURE.md",
		"../../examples/README.md",
	}
	more, err := filepath.Glob("../../examples/*/README.md")
	if err != nil {
		t.Fatal(err)
	}
	return append(files, more...)
}

func TestPublishedCodeCompiles(t *testing.T) {
	dir := t.TempDir()
	count := 0
	for _, doc := range publicDocs(t) {
		src, err := os.ReadFile(doc)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			t.Fatal(err)
		}
		for i, m := range block.FindAllStringSubmatch(string(src), -1) {
			code := m[1]
			count++
			if _, err := parser.ParseGermanio(doc, code); err == nil {
				continue // a program of the strict core
			}
			if !strings.HasPrefix(strings.TrimSpace(code), "crie sistema") {
				code = "crie sistema Exemplo\n\n" + code
			}
			file := filepath.Join(dir, "bloco.ge")
			if err := os.WriteFile(file, []byte(code), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := runtime.Compilar(file); err != nil {
				t.Errorf("%s, bloco %d não compila: %v\n%s", doc, i+1, err, m[1])
			}
		}
	}
	t.Logf("%d blocos ge verificados", count)
	if count < 10 {
		t.Fatalf("só %d blocos ge encontrados: os documentos mudaram de lugar?", count)
	}
}
