package servidor

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/runtime/banco"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Aggregates (docs/gep/0047-agregados.md, em teste): a record shows numbers
// about the records that belong to it — how many there are, or the sum of a
// numeric field — under `indicadores` of its data block. Like the
// indicators of a page (GEP 0012), a number counts only what the viewer may
// see. The database computes it: one grouped query per number for a whole
// page of records, never loading the records counted. Visibility is decided
// once per distinct combination of the columns it depends on (parents,
// owners, flags and lists of a restriction), not once per record; only data
// whose visibility depends on the record's own identity (members of its
// own, a hierarchy, `antes de ver`) is read record by record, in batches.

func (a *intentAPI) setupAggregates() {
	for _, n := range a.app.Order {
		if len(a.app.Entities[n].Aggregates) > 0 {
			a.in.Aggregates = a.decorateAggregates
			return
		}
	}
}

// decorateAggregates fills the numbers of one record being shown that a
// list has not filled already (fillAggregates).
func (a *intentAPI) decorateAggregates(ctx *interp.Context, atual map[string]any, e *ast.Entity, row, out map[string]any) {
	if row == nil || row["id"] == nil {
		return
	}
	var missing []*ast.Aggregate
	for _, ag := range e.Aggregates {
		if _, ok := out[ag.Name]; !ok {
			missing = append(missing, ag)
		}
	}
	if len(missing) == 0 {
		return
	}
	vals := a.aggregateValues(ctx, atual, missing, []any{row["id"]})
	for _, ag := range missing {
		out[ag.Name] = vals[ag.Name][idKey(row["id"])]
	}
}

// fillAggregates puts the numbers of a page of records into them: one query
// per number, whatever the number of records.
func (a *intentAPI) fillAggregates(ctx *interp.Context, atual map[string]any, e *ast.Entity, rows []map[string]any) {
	if len(e.Aggregates) == 0 || len(rows) == 0 {
		return
	}
	ids := make([]any, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row["id"])
	}
	vals := a.aggregateValues(ctx, atual, e.Aggregates, ids)
	for _, row := range rows {
		for _, ag := range e.Aggregates {
			row[ag.Name] = vals[ag.Name][idKey(row["id"])]
		}
	}
}

func idKey(v any) string { return fmt.Sprint(int64(asNumber(v))) }

// aggregateValues: each number for each record id (0 when nothing counts).
func (a *intentAPI) aggregateValues(ctx *interp.Context, atual map[string]any, ags []*ast.Aggregate, ids []any) map[string]map[string]float64 {
	if ctx == nil {
		ctx = &interp.Context{}
	}
	if !interp.ReadCacheActive(ctx) {
		interp.BeginReadCache(ctx) // a parent decides many groups: read it once
		defer interp.EndReadCache(ctx)
	}
	admin := a.in.IsAdmin(atual)
	out := map[string]map[string]float64{}
	for _, ag := range ags {
		vals := map[string]float64{}
		for _, id := range ids {
			vals[idKey(id)] = 0
		}
		out[ag.Name] = vals
		c := a.app.Entities[ag.Of]
		if c == nil {
			continue
		}
		filters := map[string]any{ag.Via + "__em": ids}
		if ag.State != "" && c.StateField != "" {
			filters[c.StateField] = ag.State
		}
		cols := []string{ag.Via}
		if !admin {
			vis, ok := visibilityColumns(a.app, c)
			if !ok {
				a.scanAggregate(ctx, atual, c, ag, filters, vals)
				continue
			}
			cols = mergeColumns(cols, vis)
		}
		groups, err := a.dbOf(ctx).Agregar(c.Singular, banco.Consulta{Filtros: filters}, cols, ag.Field)
		if err != nil {
			fmt.Printf("[germanio] indicador %s de %s: %v\n", ag.Name, c.Plural, err)
			continue
		}
		for _, g := range groups {
			if !admin {
				rec := make(map[string]any, len(cols))
				for _, col := range cols {
					rec[col] = g[col]
				}
				if !a.in.Can(ctx, atual, c, "ver", rec) {
					continue
				}
			}
			vals[idKey(g[ag.Via])] += asNumber(g["_total"])
		}
	}
	return out
}

// visibilityColumns: the columns that decide whether a record of c is
// visible; ok=false when the record's own identity matters (it has members
// of its own, sits in a hierarchy, or `antes de ver` looks at it).
func visibilityColumns(app *ast.App, c *ast.Entity) ([]string, bool) {
	if c.HasMembers || c.HierarchyField != "" || c.ViewThrough != nil || c.Singular == app.MemberModel || c.Singular == app.LoginEntity || len(c.Hooks["antes_ver"].GetBody()) > 0 {
		return nil, false
	}
	var cols []string
	for f := range c.Parents {
		cols = append(cols, f)
	}
	cols = append(cols, c.OwnerFields...)
	if c.Visibility != "" {
		cols = append(cols, c.Visibility)
	}
	for _, rs := range c.Restrictions {
		cols = append(cols, rs.Flag)
		cols = append(cols, rs.Owners...)
		cols = append(cols, rs.Lists...)
	}
	return mergeColumns(nil, cols), true
}

func mergeColumns(base, more []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range append(append([]string{}, base...), more...) {
		c = strings.ToLower(c)
		if c != "" && !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	sort.Strings(out[min(len(base), len(out)):])
	return out
}

// scanAggregate: records whose visibility depends on themselves are read in
// batches and checked one by one (only the columns that exist; never all at
// once in memory).
func (a *intentAPI) scanAggregate(ctx *interp.Context, atual map[string]any, c *ast.Entity, ag *ast.Aggregate, filters map[string]any, vals map[string]float64) {
	for batch := 1; ; batch++ {
		res, err := a.in.Op(ctx, c.Singular, "filtrar", filters, map[string]any{"limite": 500, "pagina": batch, "ordenar": "id"})
		if err != nil {
			return
		}
		rows := res.([]any)
		for _, it := range rows {
			row := it.(map[string]any)
			if !a.in.Can(ctx, atual, c, "ver", row) {
				continue
			}
			if ag.Field == "" {
				vals[idKey(row[ag.Via])]++
			} else {
				vals[idKey(row[ag.Via])] += asNumber(row[strings.ToLower(ag.Field)])
			}
		}
		if len(rows) < 500 {
			return
		}
	}
}

// resettable: the aggregate an action zerar_<nome> brings back to zero.
func resettable(e *ast.Entity, verb string) *ast.Aggregate {
	if !strings.HasPrefix(verb, "zerar_") {
		return nil
	}
	for _, ag := range e.Aggregates {
		if ag.Reset && "zerar_"+ag.Name == verb {
			return ag
		}
	}
	return nil
}

// resetAggregate performs `pode zerar <nome>`: one more record that
// subtracts the whole sum, created as the person (with the rules, hooks,
// history and events of creating it), in the request's transaction. The
// record is locked first, so two resets at once never both subtract: the
// second one finds zero and records nothing. Records are never deleted: the
// sum keeps its history.
func (a *intentAPI) resetAggregate(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual map[string]any, e *ast.Entity, row map[string]any, ag *ast.Aggregate, deny func(map[string]any)) {
	c := a.app.Entities[ag.Of]
	db := a.dbOf(ctx)
	if err := db.Travar(strings.ToLower(e.Model.Name), row["id"]); err != nil {
		a.failErr(w, r, err)
		return
	}
	data := a.writable(atual, c, map[string]any{}, map[string]any{ag.Via: row["id"]})
	if !a.canCreate(ctx, atual, c, data) {
		deny(row)
		return
	}
	if err := a.frozenFor(ctx, "criar", c, data, data); err != nil {
		a.failErr(w, r, err)
		return
	}
	whole, err := db.Agregar(c.Singular, banco.Consulta{Filtros: map[string]any{ag.Via: row["id"]}}, nil, ag.Field)
	if err != nil {
		a.failErr(w, r, err)
		return
	}
	all, err := db.Agregar(c.Singular, banco.Consulta{Filtros: map[string]any{ag.Via: row["id"]}}, nil, "")
	if err != nil {
		a.failErr(w, r, err)
		return
	}
	// zeroing records the person cannot see would reveal how much they add up to
	seen := a.aggregateValues(ctx, atual, []*ast.Aggregate{{Name: "_vistos", Of: ag.Of, Via: ag.Via}}, []any{row["id"]})
	if len(all) == 0 || seen["_vistos"][idKey(row["id"])] != asNumber(all[0]["_total"]) {
		msg := map[string]string{
			"pt": fmt.Sprintf("%s tem %s que você não vê; zerar %s revelaria quanto eles somam", e.Label, c.Plural, strings.ReplaceAll(ag.Name, "_", " ")),
			"en": "403 Forbidden",
		}[a.app.Messages]
		a.fail(w, 403, msg)
		return
	}
	total := 0.0
	if len(whole) > 0 {
		total = asNumber(whole[0]["_total"])
	}
	if total != 0 {
		value := any(-total)
		if f := fieldNamed(c, ag.Field); f != nil && f.Type == ast.FieldInteiro {
			value = int64(math.Round(-total))
		}
		data[strings.ToLower(ag.Field)] = value
		held := &heldResponse{header: http.Header{}}
		a.create(held, r, ctx, atual, c, data, map[string]any{}, nil)
		if held.status >= 400 {
			for k, v := range held.header {
				w.Header()[k] = v
			}
			w.WriteHeader(held.status)
			w.Write(held.body.Bytes())
			return
		}
	}
	a.json(w, 200, serializeFor(ctx, a.in, atual, e, a.find(ctx, e, fmt.Sprint(row["id"]), nil), false), nil)
}

func fieldNamed(e *ast.Entity, name string) *ast.Field {
	for _, f := range e.Model.Fields {
		if strings.EqualFold(f.Name, name) {
			return f
		}
	}
	return nil
}

// aggregateParents: the records whose numbers a change of a record of model
// shows (before and after: a record that moved changes both); live pages
// showing them follow (GEP 0020).
func (a *intentAPI) aggregateParents(model string, before, after map[string]any) []change {
	var out []change
	seen := map[string]bool{}
	for _, n := range a.app.Order {
		pe := a.app.Entities[n]
		for _, ag := range pe.Aggregates {
			if ag.Of != model {
				continue
			}
			for _, row := range []map[string]any{before, after} {
				if row == nil || row[ag.Via] == nil {
					continue
				}
				key := pe.Singular + ":" + idKey(row[ag.Via])
				if seen[key] {
					continue
				}
				seen[key] = true
				if res, _ := a.in.Op(&interp.Context{}, pe.Singular, "buscar", row[ag.Via]); res != nil {
					p := res.(map[string]any)
					out = append(out, change{model: pe.Singular, before: p, after: p})
				}
			}
		}
	}
	return out
}

// aggregateDetails: the numbers of a record on its page, after its fields.
func aggregateDetails(e *ast.Entity, row map[string]any) []detailItem {
	var out []detailItem
	for _, ag := range e.Aggregates {
		if v, ok := row[ag.Name]; ok {
			out = append(out, detailItem{Label: ag.Label, Value: displayFor(ast.FieldNumero, v)})
		}
	}
	return out
}
