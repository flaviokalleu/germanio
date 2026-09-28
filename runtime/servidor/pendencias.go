package servidor

import (
	"encoding/json"
	"fmt"

	"github.com/flaviokalleu/germanio/compiler/ast"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Pending items (docs/gep/0009-pendencias.md, em teste): a person placed in
// a pending field of a record (the assignees of an issue) receives a pending
// item for it; a person removed from the field loses the open ones; deleting
// the record deletes its pending items. Whoever makes the change does not
// get one for themselves.

func peopleIn(row map[string]any, field string) map[int64]bool {
	out := map[int64]bool{}
	if row == nil {
		return out
	}
	switch v := row[field].(type) {
	case nil:
	case []any:
		for _, it := range v {
			if n := int64(asNumber(it)); n > 0 {
				out[n] = true
			}
		}
	case string:
		var list []any
		if json.Unmarshal([]byte(v), &list) == nil {
			for _, it := range list {
				if n := int64(asNumber(it)); n > 0 {
					out[n] = true
				}
			}
		} else if n := int64(asNumber(v)); n > 0 {
			out[n] = true
		}
	default:
		if n := int64(asNumber(v)); n > 0 {
			out[n] = true
		}
	}
	return out
}

func (a *intentAPI) pending(ctx *interp.Context, atual map[string]any, e *ast.Entity, before, after map[string]any) error {
	pe := a.app.PendingEntity
	if len(e.PendingFields) == 0 || pe == "" || after == nil {
		return nil
	}
	self := int64(0)
	if atual != nil {
		self = int64(asNumber(atual["id"]))
	}
	for _, f := range e.PendingFields {
		old, now := peopleIn(before, f), peopleIn(after, f)
		for id := range now {
			if old[id] || id == self {
				continue
			}
			motivo := fmt.Sprintf("%s: %s %s", label(f), e.Label, titleOf(e, after))
			if _, err := a.in.Op(ctx, pe, "criar", map[string]any{"dono_id": id, "recurso": e.Singular, "recurso_id": after["id"], "motivo": motivo}); err != nil {
				return err
			}
		}
		for id := range old {
			if now[id] {
				continue
			}
			if err := a.dropPending(ctx, e, after["id"], map[string]any{"dono_id": id, "estado": "aberta"}); err != nil {
				return err
			}
		}
	}
	return nil
}

// dropPending deletes the pending items of a record (optionally narrowed).
func (a *intentAPI) dropPending(ctx *interp.Context, e *ast.Entity, id any, extra map[string]any) error {
	pe := a.app.PendingEntity
	if pe == "" || len(e.PendingFields) == 0 {
		return nil
	}
	filters := map[string]any{"recurso": e.Singular, "recurso_id": id}
	for k, v := range extra {
		filters[k] = v
	}
	for {
		rows, err := a.in.Op(ctx, pe, "filtrar", filters, map[string]any{"limite": 500, "ordenar": "id"})
		if err != nil {
			return err
		}
		list := rows.([]any)
		if len(list) == 0 {
			return nil
		}
		for _, it := range list {
			if _, err := a.in.Op(ctx, pe, "deletar", it.(map[string]any)["id"]); err != nil {
				return err
			}
		}
	}
}
