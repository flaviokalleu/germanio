package servidor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/runtime/banco"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Moving a record to another parent (docs/gep/0034-mudar-de-lugar.md, em
// teste): `issue pode mudar de projeto`. The record keeps what is its own
// (comments, files, history) and takes its place in the destination as if
// it had been created there:
//   - whoever moves must be allowed to change it where it is (editar) and to
//     create one in the destination;
//   - nothing moves out of, or into, something read-only;
//   - a number per parent (numero por projeto) becomes the destination's
//     next one;
//   - references to things of the old parent (its milestone, its labels) do
//     not follow: a list named by a field keeps the names the destination
//     also has, anything else is emptied.
// Everything happens in the request's transaction.

func (a *intentAPI) move(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual map[string]any, e *ast.Entity, row, body map[string]any, deny func(map[string]any)) {
	// who may change it where it is (mudar is a synonym of editar in the rules)
	if !a.in.Can(ctx, atual, e, "editar", row) {
		deny(row)
		return
	}
	field := e.MoveField
	pe := a.app.Entities[e.Parents[field]]
	dest := body[field]
	if dest == nil || toStr(dest) == "" {
		msg := map[string]string{"pt": "diga para onde: " + field, "en": "can't be blank"}[a.app.Messages]
		a.fail(w, 400, map[string]any{a.ext(field): []any{msg}})
		return
	}
	to := a.find(ctx, pe, toStr(dest), nil)
	if to == nil || !a.in.Can(ctx, atual, pe, "ver", to) {
		a.fail(w, 404, a.msg("404", pe))
		return
	}
	if toStr(to["id"]) == toStr(row[field]) {
		msg := map[string]string{
			"pt": fmt.Sprintf("%s já está em %s", e.Label, titleOf(pe, to)),
			"en": fmt.Sprintf("Cannot move %s to %s it originates from", strings.ToLower(e.Label), strings.ToLower(pe.Label)),
		}[a.app.Messages]
		a.fail(w, 400, msg)
		return
	}
	data := map[string]any{field: to["id"]}
	if err := a.referencesAfterMove(ctx, e, pe, row, to, data); err != nil {
		a.failErr(w, r, err)
		return
	}
	merged := map[string]any{}
	for k, v := range row {
		merged[k] = v
	}
	for k, v := range data {
		merged[k] = v
	}
	delete(merged, "id") // the destination decides as for a new record
	if !a.canCreate(ctx, atual, e, merged) {
		deny(nil)
		return
	}
	if err := a.frozenFor(ctx, "criar", e, merged, merged); err != nil {
		a.failErr(w, r, err)
		return
	}
	merged["id"] = row["id"]
	if err := a.guards(ctx, atual, e, merged, row); err != nil {
		a.failErr(w, r, err)
		return
	}
	// The place and its numbers change together (a number is unique only
	// inside its place); the number never changes by editing, only a move
	// gives a new one. The ordinary update then checks the rest.
	db := a.dbOf(ctx)
	place := map[string]any{field: to["id"]}
	for _, f := range e.Model.Fields {
		if f.NumberedBy != field {
			continue
		}
		key := strings.ToLower(f.Name)
		n, err := db.Sequencia(interp.SequenceKey(e.Model, key, to["id"]))
		if err != nil {
			a.failErr(w, r, err)
			return
		}
		place[key] = n
	}
	if _, err := db.AtualizarOnde(e.Singular, banco.Consulta{Filtros: map[string]any{"id": row["id"]}}, place); err != nil {
		a.failErr(w, r, err)
		return
	}
	updated, err := a.in.Op(ctx, e.Singular, "atualizar", row["id"], data)
	if err != nil {
		a.failErr(w, r, err)
		return
	}
	urow := updated.(map[string]any)
	if e.MinRole != "" && e.InheritVia == field {
		if err := a.keepsHolderAfter(ctx, e, urow); err != nil {
			a.failErr(w, r, err)
			return
		}
	}
	if err := a.history(ctx, atual, e, "mudar", row, urow); err != nil {
		a.failErr(w, r, err)
		return
	}
	a.emit(ctx, e, "mudar", urow, atual)
	a.json(w, 200, serializeFor(ctx, a.in, atual, e, urow, false), nil)
}

// referencesAfterMove fills data with the references of row that point at
// things of the old parent: emptied, or — for a list named by a field
// (labels por nome) — the items of the destination with the same names.
func (a *intentAPI) referencesAfterMove(ctx *interp.Context, e, pe *ast.Entity, row, to map[string]any, data map[string]any) error {
	inside := func(target string) (*ast.Entity, string) {
		te := a.app.Entities[target]
		if te == nil || te == pe || target == a.app.LoginEntity {
			return nil, ""
		}
		return te, parentFieldOf(te, pe.Singular)
	}
	for _, f := range e.Model.Fields {
		key := strings.ToLower(f.Name)
		if key == e.MoveField || f.System {
			continue
		}
		switch {
		case f.Type == ast.FieldLista && f.ListOf != "":
			te, pf := inside(f.ListOf)
			if te == nil || pf == "" {
				continue
			}
			kept := []any{}
			if f.ByName != "" {
				for _, id := range orderedIDs(row[key]) {
					res, _ := a.in.Op(ctx, te.Singular, "buscar", id)
					item, _ := res.(map[string]any)
					if item == nil {
						continue
					}
					found, err := a.in.Op(ctx, te.Singular, "encontrar", map[string]any{f.ByName: item[f.ByName], pf: to["id"]})
					if err != nil {
						return err
					}
					if same, _ := found.(map[string]any); same != nil {
						kept = append(kept, same["id"])
					}
				}
			}
			data[key] = kept
		case f.Reference != "" && row[key] != nil:
			if te, pf := inside(f.Reference); te != nil && pf != "" {
				data[key] = nil
			}
		}
	}
	return nil
}

// orderedIDs reads a list of references in its own order.
func orderedIDs(v any) []any {
	var list []any
	switch x := v.(type) {
	case []any:
		list = x
	case string:
		json.Unmarshal([]byte(x), &list)
	}
	var out []any
	for _, it := range list {
		if n := int64(asNumber(it)); n > 0 {
			out = append(out, n)
		}
	}
	return out
}
