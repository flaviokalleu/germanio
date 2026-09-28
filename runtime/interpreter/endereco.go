package interpreter

import (
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/runtime/banco"
)

// Addresses: `endereço dentro do grupo pai ou do criador`. The address of a
// record is its container's address + "/" + its own segment, computed on
// every write (never accepted from input), unique across every addressed
// entity and — when people contain records — the people's names. When a
// container's address changes, the records inside follow.

func (interp *Interpreter) entityOf(m *ast.Model) *ast.Entity {
	if interp.App == nil || m == nil {
		return nil
	}
	return interp.App.Entities[strings.ToLower(m.Name)]
}

// personName is the login field that names people in addresses (username).
func (interp *Interpreter) personName() string {
	if l := interp.App.Login; l != nil && len(l.Fields) > 0 {
		return strings.ToLower(l.Fields[0])
	}
	return ""
}

// peopleInNamespace: some address is "dentro do criador" (or similar).
func (interp *Interpreter) peopleInNamespace() bool {
	for _, e := range interp.App.Entities {
		if e.Address != nil {
			for _, r := range e.Address.Within {
				if r.Entity == interp.App.LoginEntity {
					return true
				}
			}
		}
	}
	return false
}

func (interp *Interpreter) addressOf(entity string, row map[string]any) string {
	if e := interp.App.Entities[entity]; e != nil && e.Address != nil {
		return toString(row[e.Address.Field])
	}
	if entity == interp.App.LoginEntity {
		return toString(row[interp.personName()])
	}
	return ""
}

// addressTaken reports whether full is already somebody's address.
func (interp *Interpreter) addressTaken(full, self string, id int64) bool {
	for name, e := range interp.App.Entities {
		if e.Address == nil {
			continue
		}
		f := map[string]any{e.Address.Field: full}
		if name == self && id > 0 {
			f["id__diferente"] = id
		}
		if n, err := interp.DB.ContarFiltro(name, banco.Consulta{Filtros: f}); err == nil && n > 0 {
			return true
		}
	}
	if pn := interp.personName(); pn != "" && self != interp.App.LoginEntity && interp.peopleInNamespace() {
		if n, err := interp.DB.ContarFiltro(interp.App.LoginEntity, banco.Consulta{Filtros: map[string]any{pn: full}}); err == nil && n > 0 {
			return true
		}
	}
	return false
}

// dropAddress removes the computed field from input before validation.
func (interp *Interpreter) dropAddress(m *ast.Model, out map[string]any) {
	if e := interp.entityOf(m); e != nil && e.Address != nil {
		delete(out, e.Address.Field)
	}
}

// address sets out[endereco] when the segment or a container is written,
// and keeps people's names out of addresses already in use.
func (interp *Interpreter) address(m *ast.Model, out map[string]any, create bool, id int64, errs fieldErrors) {
	e := interp.entityOf(m)
	if e == nil {
		return
	}
	if e.Singular == interp.App.LoginEntity && interp.peopleInNamespace() {
		pn := interp.personName()
		if v, ok := out[pn]; ok && v != nil && interp.addressTaken(toString(v), e.Singular, id) {
			errs.add(pn, "has already been taken")
		}
	}
	a := e.Address
	if a == nil {
		return
	}
	relevant := create
	if _, ok := out[a.Segment]; ok {
		relevant = true
	}
	for _, r := range a.Within {
		if _, ok := out[r.Field]; ok {
			relevant = true
		}
	}
	if !relevant {
		return
	}
	var cur map[string]any
	if !create {
		cur, _ = interp.DB.BuscarRegistro(e.Singular, id)
	}
	val := func(k string) any {
		if v, ok := out[k]; ok {
			return v
		}
		if cur != nil {
			return cur[k]
		}
		return nil
	}
	seg := toString(val(a.Segment))
	if seg == "" || len(errs[a.Segment]) > 0 {
		return // required/invalid segment is reported by its own rules
	}
	prefix := ""
	for _, r := range a.Within {
		v := val(r.Field)
		if v == nil || toString(v) == "" {
			continue
		}
		row, _ := interp.DB.BuscarRegistro(r.Entity, int64(toNumber(v)))
		if row == nil {
			return // "must exist" comes from the reference check
		}
		prefix = interp.addressOf(r.Entity, row)
		break
	}
	full := seg
	if prefix != "" {
		full = prefix + "/" + seg
	}
	if interp.addressTaken(full, e.Singular, id) {
		errs.add(a.Segment, "has already been taken")
		return
	}
	out[a.Field] = full
}

// followAddress updates the addresses inside a record whose address (or,
// for people, name) may have changed.
func (interp *Interpreter) followAddress(m *ast.Model, id int64, before, after map[string]any) {
	e := interp.entityOf(m)
	if e == nil || before == nil || after == nil {
		return
	}
	if interp.addressOf(e.Singular, before) == interp.addressOf(e.Singular, after) {
		return
	}
	for name, d := range interp.App.Entities {
		if d.Address == nil {
			continue
		}
		for _, r := range d.Address.Within {
			if r.Entity != e.Singular {
				continue
			}
			rows, _, err := interp.DB.Filtrar(name, banco.Consulta{Filtros: map[string]any{r.Field: id}})
			if err != nil {
				continue
			}
			dm := interp.modelAST(name)
			for _, row := range rows {
				rid := int64(toNumber(row["id"]))
				out := map[string]any{d.Address.Segment: row[d.Address.Segment]}
				errs := fieldErrors{}
				interp.address(dm, out, false, rid, errs)
				if len(errs) > 0 || out[d.Address.Field] == nil {
					continue
				}
				if updated, err := interp.DB.AtualizarMapa(name, rid, map[string]any{d.Address.Field: out[d.Address.Field]}); err == nil {
					interp.followAddress(dm, rid, row, updated)
				}
			}
		}
	}
}
