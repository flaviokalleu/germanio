package parser

import (
	"strings"
	"testing"
)

// `X pode mudar de Y` (GEP 0034): Y is a parent of X; block and flat forms
// say the same.
func TestMudarDeLugar(t *testing.T) {
	base := "crie sistema x\n\ntenha filas, chamados, setores\n\nfilas\n    tem\n        nome\n        chamados\n\nsetores\n    tem\n        nome\n\n"
	for _, form := range []string{
		"chamados\n    tem\n        titulo\n    pode\n        mudar de fila\n",
		"chamado tem titulo\nchamado pode mudar de fila\n",
	} {
		prog := parse(t, base+form)
		if err := ResolveIntent(prog); err != nil {
			t.Fatalf("%q: %v", form, err)
		}
		if f := prog.App.Entities["chamado"].MoveField; f != "fila_id" {
			t.Fatalf("%q: MoveField = %q", form, f)
		}
	}
	for form, want := range map[string]string{
		"chamados\n    tem\n        titulo\n    pode\n        mudar de setor\n":   "chamado não pertence a setor",
		"chamados\n    tem\n        titulo\n    pode\n        mudar de planeta\n": `não conheço "planeta"`,
		"chamados\n    tem\n        titulo\n    pode\n        mudar\n":            "pode mudar de quê",
	} {
		prog := parse(t, base+form)
		if err := ResolveIntent(prog); err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("%q: esperado %q, veio %v", form, want, err)
		}
	}
}
