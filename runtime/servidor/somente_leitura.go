package servidor

import (
	"fmt"

	"github.com/flaviokalleu/germanio/compiler/ast"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Read-only while a condition holds (`projeto arquivado é somente leitura`):
// the record changes only to lift the condition, and nothing that belongs to
// it is created, edited, deleted or moved to another state; its repository
// receives no code. Deleting the record itself stays possible.

// lockedAncestor returns the first frozen record among rec's parents
// (self=true also looks at rec itself).
func (a *intentAPI) lockedAncestor(ctx *interp.Context, e *ast.Entity, rec map[string]any, self bool, depth int) (*ast.Entity, map[string]any) {
	if e == nil || rec == nil || depth > 16 {
		return nil, nil
	}
	if self && e.ReadOnlyWhen != "" && truthy(rec[e.ReadOnlyWhen]) {
		return e, rec
	}
	if e.Singular == a.app.MemberModel {
		if target := a.app.Entities[toStr(rec["recurso"])]; target != nil {
			res, _ := a.in.Op(ctx, target.Singular, "buscar", rec["recurso_id"])
			if parent, _ := res.(map[string]any); parent != nil {
				return a.lockedAncestor(ctx, target, parent, true, depth+1)
			}
		}
		return nil, nil
	}
	for field, target := range e.Parents {
		if rec[field] == nil || target == a.app.LoginEntity {
			continue
		}
		pe := a.app.Entities[target]
		res, _ := a.in.Op(ctx, pe.Singular, "buscar", rec[field])
		if parent, _ := res.(map[string]any); parent != nil {
			if le, lr := a.lockedAncestor(ctx, pe, parent, true, depth+1); le != nil {
				return le, lr
			}
		}
	}
	return nil, nil
}

func (a *intentAPI) readOnlyError(e *ast.Entity) error {
	msg := map[string]string{
		"pt": fmt.Sprintf("%s está %s: é somente leitura", e.Label, e.ReadOnlyWhen),
		"en": fmt.Sprintf("%s is read-only while %s", e.Label, a.ext(e.ReadOnlyWhen)),
	}[a.app.Messages]
	return &interp.RuntimeError{Status: 403, Message: msg}
}

// frozenFor: nil when the operation may go on.
//   - criar: data of the new record (its parents decide);
//   - editar: row being edited and the changes (lifting the condition is allowed);
//   - excluir: the row (its parents decide; the frozen record itself may go);
//   - acao: the row (itself or its parents).
func (a *intentAPI) frozenFor(ctx *interp.Context, op string, e *ast.Entity, row, data map[string]any) error {
	switch op {
	case "criar", "excluir":
		if le, _ := a.lockedAncestor(ctx, e, row, false, 0); le != nil {
			return a.readOnlyError(le)
		}
	case "editar":
		if e.ReadOnlyWhen != "" && truthy(row[e.ReadOnlyWhen]) {
			for k := range data {
				if k != e.ReadOnlyWhen {
					return a.readOnlyError(e)
				}
			}
			return nil
		}
		if le, _ := a.lockedAncestor(ctx, e, row, false, 0); le != nil {
			return a.readOnlyError(le)
		}
	case "acao":
		if le, _ := a.lockedAncestor(ctx, e, row, true, 0); le != nil {
			return a.readOnlyError(le)
		}
	}
	return nil
}
