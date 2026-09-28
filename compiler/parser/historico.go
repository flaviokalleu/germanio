package parser

import (
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/lexer"
)

// historySource is the data the history is (GEP 0011, em teste). Ordinary
// Germanio: ge explain atividades shows it. Who sees an activity is decided
// by the record it describes (ast.ViewThrough), never written here.
const historySource = `atividades
    tem
        acao
        recurso
        recurso_id inteiro
        resumo
        campos
        dentro
        dentro_id inteiro
        autor
    acesso
        {{pessoa}}
            ver
    permita
        filtrar por acao, recurso, recurso_id, dentro, dentro_id e autor
`

// historyFields are what the runtime writes; a program that declares its
// own atividades must have them.
var historyFields = []string{"acao", "recurso", "recurso_id", "resumo", "campos", "dentro", "dentro_id", "autor_id"}

// withHistoryData adds the history data when a data keeps history and the
// program does not declare atividades itself.
func withHistoryData(in *ast.Intent) error {
	if len(in.History) == 0 {
		return nil
	}
	for _, d := range in.Entities {
		if d.Name == "atividades" || d.Singular == "atividade" {
			return nil // declared by the application: its own definition wins
		}
	}
	toks, err := lexer.New(strings.ReplaceAll(historySource, "{{pessoa}}", loginData(in))).Tokenize()
	if err != nil {
		return err
	}
	p := New(toks)
	p.File = "<histórico (GEP 0011)>"
	prog, err := p.Parse()
	if err != nil {
		return err
	}
	if prog.Intent != nil {
		*in = *ast.MergeIntent(in, prog.Intent)
	}
	return nil
}

// loginData is the data people log in with (the one with a password).
func loginData(in *ast.Intent) string {
	person := "usuario"
	for _, b := range in.FieldBlocks {
		for _, line := range b.Lines {
			if w := wordsOf(line); len(w) > 0 && (w[0] == "senha" || w[0] == "password") {
				person = b.Entity
			}
		}
	}
	return person
}

// history marks the data that keep history and wires the history data.
func (r *resolver) history(in *ast.Intent, app *ast.App) error {
	if len(in.History) == 0 {
		return nil
	}
	for _, h := range in.History {
		e, err := r.entity(h.Entity, h.Pos)
		if err != nil {
			return err
		}
		e.History = true
	}
	ae := r.byName["atividades"]
	if ae == nil {
		return nil
	}
	have := map[string]bool{}
	for _, f := range ae.Model.Fields {
		have[strings.ToLower(f.Name)] = true
	}
	var missing []string
	for _, f := range historyFields {
		if !have[f] {
			missing = append(missing, strings.TrimSuffix(f, "_id"))
		}
	}
	if len(missing) > 0 {
		return r.errAt(in.History[0].Pos, "o histórico é guardado em atividades, mas atividades não tem %s. Declare esses campos, ou tire a declaração de atividades para o Germanio escrevê-la", strings.Join(missing, ", "))
	}
	ae.ViewThrough = &ast.ViewThrough{Kind: "recurso", ID: "recurso_id", ParentKind: "dentro", ParentID: "dentro_id", Author: "autor_id"}
	app.ActivityEntity = ae.Singular
	return nil
}
