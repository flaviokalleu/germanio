package servidor

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Copies (docs/gep/0029-copias.md, em teste): `copiar` in the access rules
// of a data lets those people copy a record they can see. The copy is a new
// record created by whoever copies — with every rule of creating one (where
// it may be created, `quem cria vira owner`, the visibility ceilings) — that
// takes the plain values of the original, has a copy of its repository and
// remembers the original in copiado_de_id. It is never more visible than
// the original.

var visibilityOrder = map[string]int{"private": 0, "internal": 1, "public": 2}

// copyRecord creates the copy of origin asked by atual; body may change
// the values of the copy (a new name, where it goes, a lower visibility).
func (a *intentAPI) copyRecord(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual map[string]any, e *ast.Entity, origin, body map[string]any, deny func(map[string]any)) {
	if !a.in.Can(ctx, atual, e, "copiar", origin) {
		deny(origin)
		return
	}
	// Copying a repository is reading it: whoever cannot download the code
	// cannot copy it.
	if e.Repository && !a.in.Can(ctx, atual, e, "baixar_codigo", origin) {
		deny(origin)
		return
	}
	body = a.inwardBody(e, body)
	if f := fileIn(e, body); f != "" {
		a.fail(w, 400, fmt.Sprintf("%s é um arquivo: envie-o para o endereço da cópia depois de criá-la", f))
		return
	}
	data := copiedValues(e, origin)
	for k, v := range a.writable(atual, e, body, map[string]any{}) {
		data[k] = v
	}
	data["copiado_de_id"] = origin["id"]
	if vis := e.Visibility; vis != "" {
		if _, asked := body[vis]; !asked {
			// the copy starts as visible as the original, never more than
			// where it is created; asking for more is refused by guards
			data[vis] = a.lowestVisibility(ctx, e, data, toStr(origin[vis]))
		}
	}
	if err := a.namesIn(ctx, atual, e, data, nil); err != nil {
		a.failErr(w, r, err)
		return
	}
	if !a.canCreate(ctx, atual, e, data) {
		if pe := a.hiddenParent(ctx, atual, e, data); pe != nil {
			a.fail(w, 404, a.msg("404", pe))
			return
		}
		a.fail(w, 403, a.msg("403", e))
		return
	}
	if err := a.frozenFor(ctx, "criar", e, data, data); err != nil {
		a.failErr(w, r, err)
		return
	}
	a.create(w, r, ctx, atual, e, data, body, origin)
}

// copiedValues: the plain values of the original that a copy takes. What
// belongs to the original's place (its parents, its number, its address),
// what the runtime keeps (state, counts, the repository path), files,
// secrets and references to other data are not copied.
func copiedValues(e *ast.Entity, origin map[string]any) map[string]any {
	skip := map[string]bool{e.HierarchyField: true, e.ReadOnlyWhen: true}
	for field := range e.Parents {
		skip[field] = true
	}
	for _, owner := range e.OwnerFields {
		skip[owner] = true
	}
	if e.Address != nil {
		skip[e.Address.Field] = true
	}
	out := map[string]any{}
	for _, f := range e.Model.Fields {
		key := strings.ToLower(f.Name)
		if skip[key] || f.System || f.Hidden || f.IsSecret() || isFileField(f) || f.NumberedBy != "" || f.Reference != "" {
			continue
		}
		if f.Type == ast.FieldLista && f.ListOf != "texto" && f.ListOf != "numero" {
			continue
		}
		if v, ok := origin[key]; ok && v != nil {
			out[key] = v
		}
	}
	return out
}

// lowestVisibility: want, lowered to the visibility of every parent that
// bounds the record's (não pode ser mais visível que o grupo).
func (a *intentAPI) lowestVisibility(ctx *interp.Context, e *ast.Entity, data map[string]any, want string) string {
	if _, ok := visibilityOrder[want]; !ok {
		want = "private"
	}
	for _, field := range e.CeilingFields {
		if data[field] == nil {
			continue
		}
		pe := a.app.Entities[e.Parents[field]]
		res, _ := a.in.Op(ctx, pe.Singular, "buscar", data[field])
		if parent, _ := res.(map[string]any); parent != nil {
			if pv := toStr(parent[pe.Visibility]); visibilityOrder[pv] < visibilityOrder[want] {
				want = pv
			}
		}
	}
	return want
}

// copyCeiling: a copy is never more visible than its original, when it is
// created and whenever it is edited later.
func (a *intentAPI) copyCeiling(ctx *interp.Context, e *ast.Entity, data map[string]any) error {
	if !e.Copies || e.Visibility == "" || data["copiado_de_id"] == nil || data[e.Visibility] == nil {
		return nil
	}
	res, _ := a.in.Op(ctx, e.Singular, "buscar", data["copiado_de_id"])
	origin, _ := res.(map[string]any)
	if origin == nil {
		return nil
	}
	mine, theirs := toStr(data[e.Visibility]), toStr(origin[e.Visibility])
	if visibilityOrder[mine] <= visibilityOrder[theirs] {
		return nil
	}
	msg := fmt.Sprintf("A visibilidade não pode ser maior que a do original (%s)", theirs)
	if a.app.Messages == "en" {
		msg = fmt.Sprintf("Visibility level %s is not allowed since the fork source has a more restrictive visibility", mine)
	}
	return &interp.RuntimeError{Status: 400, Message: msg, Payload: map[string]any{e.Visibility: []any{msg}}}
}

// forgetOrigin: copies of a record being deleted stay, without the link.
func (a *intentAPI) forgetOrigin(ctx *interp.Context, e *ast.Entity, id any) error {
	if !e.Copies {
		return nil
	}
	_, err := a.dbOf(ctx).Executar(fmt.Sprintf(`UPDATE %s SET copiado_de_id = NULL WHERE copiado_de_id = %s`, quoteIdent(strings.ToLower(e.Model.Name)), a.s.ph(1)), id)
	return err
}

// originOut shows the original of a copy to whoever may see it; to anyone
// else the copy does not say where it came from.
func originOut(ctx *interp.Context, in *interp.Interpreter, atual map[string]any, e *ast.Entity, row, out map[string]any) {
	if !e.Copies || row["copiado_de_id"] == nil {
		return
	}
	res, _ := in.Op(ctx, e.Singular, "buscar", row["copiado_de_id"])
	origin, _ := res.(map[string]any)
	if origin == nil || !in.Can(ctx, atual, e, "ver", origin) {
		delete(out, "copiado_de_id")
		return
	}
	out["copiado_de"] = serializeFor(nil, nil, nil, e, origin, false)
}
