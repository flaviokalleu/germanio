package servidor

import (
	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/runtime/banco"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// PreFiltroVisibilidade (on by default) lets the differential test compare
// the narrowed list with the full scan.
var PreFiltroVisibilidade = true

// visibilityPrefilter returns groups of filters (at least one must hold)
// that contain every record of e atual may see: a superset built from the
// ways a record can become visible — through a parent atual can reach, its
// own visibility or membership, ownership, or a restriction that lists
// people. Can still decides each row, so the answer never changes; only the
// rows read do (docs/research/performance/AUDITORIA.md, G85). ok=false: the
// shape is not covered and the list reads everything, as before.
func (a *intentAPI) visibilityPrefilter(ctx *interp.Context, atual map[string]any, e *ast.Entity) ([]map[string]any, bool) {
	app := a.app
	if !PreFiltroVisibilidade || (atual != nil && a.in.IsAdmin(atual)) {
		return nil, false
	}
	if e.HierarchyField != "" || e.Singular == app.MemberModel || e.Singular == app.LoginEntity || len(e.Hooks["antes_ver"].GetBody()) > 0 {
		return nil, false
	}
	var groups []map[string]any
	// through each parent with members: the parent must be reachable
	var membered []string
	for field, target := range e.Parents {
		pe := app.Entities[target]
		if target == app.LoginEntity || pe == nil || !a.in.HasMembersChain(pe) {
			continue
		}
		membered = append(membered, field)
		ids, ok := a.reachable(ctx, atual, pe)
		if !ok {
			return nil, false
		}
		groups = append(groups, map[string]any{field + "__em": ids})
	}
	if e.Visibility != "" {
		groups = append(groups, map[string]any{e.Visibility: "public"})
		if atual != nil {
			groups = append(groups, map[string]any{e.Visibility: "internal"})
		}
		if e.HasMembers && atual != nil {
			ids, err := a.memberOf(ctx, atual, e)
			if err != nil {
				return nil, false
			}
			groups = append(groups, map[string]any{"id__em": ids})
		}
	} else {
		if len(membered) == 0 {
			return nil, false // nothing to narrow by
		}
		// with no parent that has members, other rules decide
		none := map[string]any{}
		for _, f := range membered {
			none[f] = nil
		}
		groups = append(groups, none)
	}
	if atual != nil {
		for _, f := range e.OwnerFields {
			groups = append(groups, map[string]any{f: atual["id"]})
		}
	}
	for _, rs := range e.Restrictions {
		groups = append(groups, map[string]any{rs.Flag: true})
	}
	return groups, true
}

// reachable lists the ids of pe atual may reach (cached for the request).
func (a *intentAPI) reachable(ctx *interp.Context, atual map[string]any, pe *ast.Entity) ([]any, bool) {
	key := "__alcancaveis_" + pe.Singular
	if ctx != nil && ctx.Values != nil {
		if ids, ok := ctx.Values[key].([]any); ok {
			return ids, true
		}
	}
	db := a.dbOf(ctx)
	ids := []any{}
	for page := 1; ; page++ {
		rows, _, err := db.Filtrar(pe.Singular, banco.Consulta{Limite: 500, Pagina: page, Ordenar: "id"})
		if err != nil {
			return nil, false
		}
		for _, row := range rows {
			if a.in.MayReach(ctx, atual, pe, row) {
				ids = append(ids, row["id"])
			}
		}
		if len(rows) < 500 {
			break
		}
	}
	if ctx != nil {
		if ctx.Values == nil {
			ctx.Values = map[string]any{}
		}
		ctx.Values[key] = ids
	}
	return ids, true
}

// memberOf lists the records of e atual is a member of.
func (a *intentAPI) memberOf(ctx *interp.Context, atual map[string]any, e *ast.Entity) ([]any, error) {
	rows, _, err := a.dbOf(ctx).Filtrar(a.app.MemberModel, banco.Consulta{Filtros: map[string]any{"recurso": e.Singular, "pessoa_id": atual["id"]}, Limite: 100000})
	if err != nil {
		return nil, err
	}
	ids := make([]any, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r["recurso_id"])
	}
	return ids, nil
}
