package parser

import (
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/lexer"
)

// pendingSource is the data every pending item is (GEP 0009, em teste). It
// is ordinary Germanio: the same rules, the same explanations (ge explain
// pendencias), nothing hidden in Go.
const pendingSource = `pendencias
    tem
        motivo
        recurso
        recurso_id inteiro
        dono
    começa aberta
    pode
        concluir
    acesso
        {{pessoa}}
            ver seus
            concluir seus
            excluir seus
`

// withPendingData adds the pending items data when the program asks for
// pending items and does not declare it itself.
func withPendingData(in *ast.Intent) error {
	if len(in.PendingItems) == 0 {
		return nil
	}
	for _, d := range in.Entities {
		if d.Name == "pendencias" || d.Singular == "pendencia" {
			return nil // declared by the application: its own definition wins
		}
	}
	// the actor is the data people log in with (the one with a password)
	person := "usuario"
	for _, b := range in.FieldBlocks {
		for _, line := range b.Lines {
			if w := wordsOf(line); len(w) > 0 && (w[0] == "senha" || w[0] == "password") {
				person = b.Entity
			}
		}
	}
	toks, err := lexer.New(strings.ReplaceAll(pendingSource, "{{pessoa}}", person)).Tokenize()
	if err != nil {
		return err
	}
	p := New(toks)
	p.File = "<pendências (GEP 0009)>"
	prog, err := p.Parse()
	if err != nil {
		return err
	}
	if prog.Intent == nil {
		return nil
	}
	merged := ast.MergeIntent(in, prog.Intent)
	*in = *merged
	return nil
}
