package parser

import (
	"strings"
	"testing"
)

// The people's avisos_por_email (GEP 0013) is a yes/no choice.
func TestAvisosPorEmailEscolha(t *testing.T) {
	base := "crie sistema x\n\nusuarios\n    tem\n        email obrigatório e único\n        senha\n        CAMPO\n\ntenha login\ntenha avisos por e-mail\n\nchamados\n    tem\n        titulo\n        responsaveis\n    pendência para\n        responsaveis\n"
	prog := parse(t, strings.Replace(base, "CAMPO", "avisos_por_email começa com verdadeiro", 1))
	if err := ResolveIntent(prog); err != nil || prog.App.EmailChoiceField != "avisos_por_email" {
		t.Fatalf("a escolha da pessoa: %v %q", err, prog.App.EmailChoiceField)
	}
	prog = parse(t, strings.Replace(base, "CAMPO", "avisos_por_email", 1))
	if err := ResolveIntent(prog); err == nil || !strings.Contains(err.Error(), "avisos_por_email começa com verdadeiro") {
		t.Fatalf("esperado erro educativo, veio %v", err)
	}
}
