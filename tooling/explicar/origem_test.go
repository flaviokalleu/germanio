package explicar

import (
	"strings"
	"testing"

	"github.com/flaviokalleu/germanio/compiler/lexer"
	"github.com/flaviokalleu/germanio/compiler/parser"
)

func explain(t *testing.T, src, name string) string {
	t.Helper()
	toks, err := lexer.New(src).Tokenize()
	if err != nil {
		t.Fatal(err)
	}
	p := parser.New(toks)
	p.File = "app.ge"
	prog, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	if err := parser.ResolveIntent(prog); err != nil {
		t.Fatal(err)
	}
	out, err := Entidade(prog, name)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

const cabecalho = "crie sistema x\n\ntenha usuarios\n\ncada usuario tem\n    email obrigatório e único\n    senha\n\ntenha login\n\n"

// ge explain mostra, para cada fato, a frase plana equivalente e a origem
// (arquivo, linha e caminho do bloco).
func TestExplainMostraOrigem(t *testing.T) {
	out := explain(t, cabecalho+"chamados\n    tem\n        titulo\n    começa aberto\n    pode\n        fechar\n    acesso\n        usuario\n            ver\n", "chamado")
	for _, want := range []string{
		"chamado tem titulo — app.ge:13 (chamados › tem)",
		"chamado começa aberto — app.ge:14 (chamados › começa aberto)",
		"chamado pode fechar — app.ge:16 (chamados › pode)",
		"usuario pode ver chamados — app.ge:19 (chamados › acesso › usuario)",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("faltou %q:\n%s", want, out)
		}
	}
	// A frase plana mostra a mesma frase, sem caminho de bloco.
	flat := explain(t, cabecalho+"tenha chamados\n\ncada chamado tem\n    titulo\n\nchamado começa aberto\nchamado pode fechar\nusuario pode ver chamados\n", "chamado")
	for _, want := range []string{"chamado começa aberto — app.ge:16\n", "usuario pode ver chamados — app.ge:18\n"} {
		if !strings.Contains(flat, want) {
			t.Fatalf("faltou %q:\n%s", want, flat)
		}
	}
}
