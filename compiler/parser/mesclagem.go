package parser

import (
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
)

// mergeOptions gives records that propose changes between two branches the
// fields of how they are merged (GEP 0027, em teste; no syntax):
//
//   - juntar_commits (the record): one commit with all the changes;
//   - forma_de_mesclar (the record with the repository): mesclagem,
//     semi_linear or linear;
//   - mesclar_quando_passar and who asked for it (the record), when the
//     record with the repository has executions: the merge waits for the
//     latest execution of the source branch.
func mergeOptions(app *ast.App) {
	for _, n := range app.Order {
		e := app.Entities[n]
		if e.Review == nil || e.Review.Target == "" {
			continue
		}
		owner := app.Entities[e.Parents[e.Review.RepoVia]]
		addField(e, &ast.Field{Name: "juntar_commits", Type: ast.FieldBooleano, HasDefault: true, DefaultValue: false, Label: "juntar commits"})
		addField(e, &ast.Field{Name: "base_mesclagem", Type: ast.FieldTexto, System: true, Hidden: true})
		addField(owner, &ast.Field{Name: "forma_de_mesclar", Type: ast.FieldEnum, EnumValues: ast.MergeMethods, HasDefault: true, DefaultValue: ast.MergeMethods[0], Default: ast.MergeMethods[0], Label: "forma de mesclar"})
		for _, rn := range app.Order {
			run := app.Entities[rn]
			if run.Execution != nil && run.Execution.Role == "run" && run.Execution.Owner == owner.Singular {
				e.Review.Runs = run.Singular
				addField(e, &ast.Field{Name: "mesclar_quando_passar", Type: ast.FieldBooleano, HasDefault: true, DefaultValue: false, System: true})
				addField(e, &ast.Field{Name: "mesclagem_agendada_por_id", Type: ast.FieldInteiro, Reference: app.LoginEntity, System: true})
				break
			}
		}
	}
}

// addField adds f unless the data already declares a field with its name
// (a declaration always wins).
func addField(e *ast.Entity, f *ast.Field) {
	for _, have := range e.Model.Fields {
		if strings.EqualFold(have.Name, f.Name) {
			return
		}
	}
	e.Model.Fields = append(e.Model.Fields, f)
}
