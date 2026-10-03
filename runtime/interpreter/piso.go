package interpreter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/runtime/banco"
)

// Sums that never go below zero (docs/gep/0050-soma-nunca-negativa.md, em
// teste). A write of a record counted by such a sum — created, edited,
// deleted — runs in a transaction (the request's, or one of its own) that
// first locks the record(s) whose sum it changes, then writes, then adds up
// the whole sum (every record, not only those the person sees: the rule is
// about the real total) and refuses the change when it is below zero. Two
// subtractions at once are serialised by the lock: the second one sees the
// first and is refused. SQLite needs no row lock (its transactions take the
// write lock up front); PostgreSQL and MySQL use SELECT … FOR UPDATE, taken
// before the write so the lock on the parent never has to be upgraded.

// floor: one sum of owner that the records of a model count.
type floor struct {
	owner *ast.Entity
	ag    *ast.Aggregate
}

// floorsOf lists the never-negative sums that records of model count.
func (interp *Interpreter) floorsOf(model string) []floor {
	if interp.App == nil {
		return nil
	}
	var out []floor
	for _, n := range interp.App.Order {
		e := interp.App.Entities[n]
		for _, ag := range e.Aggregates {
			if ag.NonNegative && strings.EqualFold(ag.Of, model) {
				out = append(out, floor{owner: e, ag: ag})
			}
		}
	}
	return out
}

// floorParents: the records whose sums rows (before and after a change)
// count, per floor, sorted so locks are always taken in the same order.
func floorParents(fl floor, rows ...map[string]any) []int64 {
	seen := map[int64]bool{}
	var ids []int64
	for _, row := range rows {
		if row == nil || row[fl.ag.Via] == nil {
			continue
		}
		id, ok := tryNumber(row[fl.ag.Via])
		if !ok || seen[int64(id)] {
			continue
		}
		seen[int64(id)] = true
		ids = append(ids, int64(id))
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// lockFloors locks the records whose sums a change of rows touches.
func lockFloors(db *banco.Banco, floors []floor, rows ...map[string]any) error {
	for _, fl := range floors {
		for _, id := range floorParents(fl, rows...) {
			if err := db.Travar(strings.ToLower(fl.owner.Model.Name), id); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkFloors refuses (400, on the summed field) a change that left one of
// the sums below zero.
func (interp *Interpreter) checkFloors(c *Call, db *banco.Banco, model string, floors []floor, rows ...map[string]any) {
	for _, fl := range floors {
		for _, id := range floorParents(fl, rows...) {
			if removing(c.Ctx(), fl.owner.Singular, id) {
				continue
			}
			res, err := db.Agregar(model, banco.Consulta{Filtros: map[string]any{fl.ag.Via: id}}, nil, fl.ag.Field)
			if err != nil {
				panic(c.Fail(500, "%s: %v", c.Name, err))
			}
			total := 0.0
			if len(res) > 0 {
				total = toNumber(res[0]["_total"])
			}
			if total >= 0 {
				continue
			}
			label := strings.ReplaceAll(fl.ag.Name, "_", " ")
			msg := fmt.Sprintf("would make the %s of the %s negative", label, fl.owner.Singular)
			if interp.Lang() == "pt" {
				msg = fmt.Sprintf("deixaria %s de %s negativo; o registro não pode ficar com %s negativo", label, strings.ReplaceAll(fl.owner.Singular, "_", " "), label)
			}
			errs := fieldErrors{}
			errs.add(strings.ToLower(fl.ag.Field), msg)
			panic(&RuntimeError{Status: 400, Message: errs.sentence(), Payload: errs.payload(), Pos: c.Pos})
		}
	}
}

// guardFloors runs write so that no never-negative sum ends below zero: in
// a transaction (joining the current one), with the parents locked first.
// Without such sums it just runs write.
func (interp *Interpreter) guardFloors(c *Call, db *banco.Banco, model string, before, data map[string]any, write func(db *banco.Banco) (map[string]any, error)) (map[string]any, error) {
	floors := interp.floorsOf(model)
	if len(floors) == 0 {
		return write(db)
	}
	var row map[string]any
	err := db.EmTransacao(func(tx *banco.Banco) error {
		if err := lockFloors(tx, floors, before, data); err != nil {
			return err
		}
		var err error
		if row, err = write(tx); err != nil {
			return err
		}
		interp.checkFloors(c, tx, model, floors, before, row)
		return nil
	})
	return row, err
}

const removingKey = "germanio_removendo"

// MarcarRemovendo records that the record entity:id is being deleted with
// everything that belongs to it: its sums no longer matter, so deleting its
// records one by one is never refused for a total that goes away with it.
func MarcarRemovendo(ctx *Context, entity string, id any) {
	if ctx == nil {
		return
	}
	if ctx.Values == nil {
		ctx.Values = map[string]any{}
	}
	m, _ := ctx.Values[removingKey].(map[string]bool)
	if m == nil {
		m = map[string]bool{}
		ctx.Values[removingKey] = m
	}
	n, _ := tryNumber(id)
	m[fmt.Sprintf("%s:%d", entity, int64(n))] = true
}

func removing(ctx *Context, entity string, id int64) bool {
	if ctx == nil || ctx.Values == nil {
		return false
	}
	m, _ := ctx.Values[removingKey].(map[string]bool)
	return m[fmt.Sprintf("%s:%d", entity, id)]
}
