package servidor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// History (docs/gep/0011-historico.md, em teste): every change of a data
// that keeps history writes one activity, in the change's transaction —
// who did what to which record, never the values. Who reads an activity is
// decided by the record it describes (interp.Can, ast.ViewThrough).

func (a *intentAPI) history(ctx *interp.Context, atual map[string]any, e *ast.Entity, action string, before, after map[string]any) error {
	ae := a.app.ActivityEntity
	if ae == "" || !e.History {
		return nil
	}
	row := after
	if row == nil {
		row = before
	}
	if row == nil || row["id"] == nil {
		return nil
	}
	data := map[string]any{"acao": action, "recurso": e.Singular, "recurso_id": row["id"], "resumo": titleOf(e, row)}
	if action == "editar" {
		changed := changedFields(e, before, after)
		if len(changed) == 0 {
			return nil // nothing changed: nothing happened
		}
		data["campos"] = strings.Join(changed, ", ")
	}
	var fields []string
	for f := range e.Parents {
		fields = append(fields, f)
	}
	// the record's place (a project), not a person it names (its author)
	sort.Slice(fields, func(i, j int) bool {
		pi, pj := e.Parents[fields[i]] == a.app.LoginEntity, e.Parents[fields[j]] == a.app.LoginEntity
		if pi != pj {
			return pj
		}
		return fields[i] < fields[j]
	})
	for _, f := range fields {
		if row[f] != nil {
			data["dentro"], data["dentro_id"] = e.Parents[f], row[f]
			break
		}
	}
	if atual != nil {
		data["autor_id"] = atual["id"]
	}
	if action == "excluir" {
		// once the record is gone its parent decides who reads its history:
		// no title of it may remain
		data["resumo"] = ""
		db := ctx.DB
		if db == nil {
			db = a.s.DB
		}
		if _, err := db.Executar(fmt.Sprintf(`UPDATE "%s" SET resumo = '' WHERE recurso = %s AND recurso_id = %s`, strings.ToLower(ae), a.s.ph(1), a.s.ph(2)), e.Singular, row["id"]); err != nil {
			return err
		}
	}
	_, err := a.in.Op(ctx, ae, "criar", data)
	return err
}

// changedFields are the names of the fields an edit changed (not system,
// hidden or password fields: their names say nothing a person needs).
func changedFields(e *ast.Entity, before, after map[string]any) []string {
	var out []string
	for _, f := range e.Model.Fields {
		if f.System || f.Hidden || f.Type == ast.FieldSenha {
			continue
		}
		name := strings.ToLower(f.Name)
		key := name
		if f.Reference != "" && !strings.HasSuffix(key, "_id") {
			key += "_id"
		}
		if fmt.Sprint(before[key]) != fmt.Sprint(after[key]) {
			out = append(out, name)
		}
	}
	return out
}
