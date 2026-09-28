package servidor

import (
	"strconv"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Lists named by a field (`issue tem labels por nome`): people and
// integrations write and read names; the record keeps references. Names are
// looked up among the items of the same parent (labels of the project), and
// a missing one is created when the person may create it there.

// sharedParent is the parent field that list items and the record both have.
func (a *intentAPI) sharedParent(e *ast.Entity, f *ast.Field) string {
	target := a.app.Entities[f.ListOf]
	if target == nil {
		return ""
	}
	for field, t := range target.Parents {
		if e.Parents[field] == t {
			return field
		}
	}
	return ""
}

func listItems(v any) []any {
	switch x := v.(type) {
	case []any:
		return x
	case string:
		var out []any
		for _, p := range strings.Split(x, ",") {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		return out
	case nil:
		return nil
	}
	return []any{v}
}

// namesIn replaces names by references in data (row: the record being edited).
func (a *intentAPI) namesIn(ctx *interp.Context, atual map[string]any, e *ast.Entity, data, row map[string]any) error {
	for _, f := range e.Model.Fields {
		key := strings.ToLower(f.Name)
		v, given := data[key]
		if f.ByName == "" || !given || v == nil {
			continue
		}
		target := a.app.Entities[f.ListOf]
		pf := a.sharedParent(e, f)
		var pv any
		if pf != "" {
			if pv = data[pf]; pv == nil && row != nil {
				pv = row[pf]
			}
		}
		out := []any{}
		for _, it := range listItems(v) {
			if n, ok := it.(float64); ok {
				out = append(out, n) // already a reference (page forms)
				continue
			}
			name := strings.TrimSpace(toStr(it))
			if name == "" {
				continue
			}
			filters := map[string]any{f.ByName: name}
			if pf != "" {
				filters[pf] = pv
			}
			res, err := a.in.Op(ctx, target.Singular, "encontrar", filters)
			if err != nil {
				return err
			}
			if found, _ := res.(map[string]any); found != nil {
				out = append(out, found["id"])
				continue
			}
			if !a.canCreate(ctx, atual, target, filters) {
				msg := map[string]string{"pt": "não existe: ", "en": "does not exist: "}[a.app.Messages] + name
				return &interp.RuntimeError{Status: 400, Message: msg, Payload: map[string]any{key: []any{msg}}}
			}
			created, err := a.in.Op(ctx, target.Singular, "criar", filters)
			if err != nil {
				return err
			}
			out = append(out, created.(map[string]any)["id"])
		}
		data[key] = out
	}
	return nil
}

// namesOut shows the names of the items of named lists.
func namesOut(ctx *interp.Context, in *interp.Interpreter, e *ast.Entity, out map[string]any) {
	for _, f := range e.Model.Fields {
		key := strings.ToLower(f.Name)
		if f.ByName == "" || out[key] == nil {
			continue
		}
		names := []any{}
		for _, it := range listItems(out[key]) {
			res, err := in.Op(ctx, f.ListOf, "buscar", it)
			if row, _ := res.(map[string]any); err == nil && row != nil {
				names = append(names, row[f.ByName])
			}
		}
		out[key] = names
	}
}

// nameFilter turns ?labels=bug into the references that carry that name
// (inside the parent when the list is nested). ok=false: nothing matches.
func (a *intentAPI) nameFilter(ctx *interp.Context, e *ast.Entity, f *ast.Field, value string, scope map[string]any) ([]any, bool) {
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return []any{`"` + value + `"`}, true
	}
	filters := map[string]any{f.ByName: value}
	if pf := a.sharedParent(e, f); pf != "" && scope[pf] != nil {
		filters[pf] = scope[pf]
	}
	res, err := a.in.Op(ctx, f.ListOf, "filtrar", filters, map[string]any{"limite": 500})
	if err != nil {
		return nil, false
	}
	var ids []any
	for _, it := range res.([]any) {
		ids = append(ids, `"`+toStr(it.(map[string]any)["id"])+`"`)
	}
	return ids, len(ids) > 0
}
