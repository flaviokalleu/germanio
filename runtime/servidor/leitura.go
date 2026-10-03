package servidor

import (
	"fmt"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Reading (docs/gep/0022-leitura.md, em teste): opening a container's page
// reads every record in it for that person; the container shows, to each
// viewer, how many records arrived since (not counting the viewer's own).
// The marks are application state per person, never domain data.

const readTable = "_germanio_leitura"

// readChildren: the data read inside records of e.
func (a *intentAPI) readChildren(e *ast.Entity) []*ast.Entity {
	var out []*ast.Entity
	for _, n := range a.app.Order {
		if c := a.app.Entities[n]; c.ReadParent == e.Singular {
			out = append(out, c)
		}
	}
	return out
}

func (a *intentAPI) setupReading() {
	any := false
	for _, n := range a.app.Order {
		any = any || a.app.Entities[n].ReadParent != ""
	}
	if !any {
		return
	}
	a.s.DB.DB.Exec(`CREATE TABLE IF NOT EXISTS ` + readTable + ` (pessoa_id INTEGER NOT NULL, recurso TEXT NOT NULL, recurso_id INTEGER NOT NULL, ate INTEGER NOT NULL, UNIQUE (pessoa_id, recurso, recurso_id))`)
	a.in.Decorate = a.decorateUnread
}

// markRead: everything in the record of e is read for the person now.
func (a *intentAPI) markRead(person any, e *ast.Entity, id any) {
	if person == nil || id == nil {
		return
	}
	for _, c := range a.readChildren(e) {
		var last int64
		a.s.DB.DB.QueryRow(fmt.Sprintf(`SELECT COALESCE(MAX(id), 0) FROM %s WHERE %s = %s`, quoteIdent(c.Singular), quoteIdent(c.ReadField), a.s.ph(1)), id).Scan(&last)
		key := c.Singular + ":" + e.Singular
		var before int64 = -1
		a.s.DB.DB.QueryRow(fmt.Sprintf(`SELECT ate FROM %s WHERE pessoa_id = %s AND recurso = %s AND recurso_id = %s`, readTable, a.s.ph(1), a.s.ph(2), a.s.ph(3)), person, key, id).Scan(&before)
		if before == last {
			continue
		}
		a.s.DB.DB.Exec(fmt.Sprintf(`DELETE FROM %s WHERE pessoa_id = %s AND recurso = %s AND recurso_id = %s`, readTable, a.s.ph(1), a.s.ph(2), a.s.ph(3)), person, key, id)
		a.s.DB.DB.Exec(fmt.Sprintf(`INSERT INTO %s (pessoa_id, recurso, recurso_id, ate) VALUES (%s, %s, %s, %s)`, readTable, a.s.ph(1), a.s.ph(2), a.s.ph(3), a.s.ph(4)), person, key, id, last)
		// the count changed: pages showing the container follow it
		if res, _ := a.in.Op(&interp.Context{}, e.Singular, "buscar", id); res != nil {
			row := res.(map[string]any)
			a.live.publish(change{model: e.Singular, before: row, after: row})
		}
	}
}

// unread: records of the container after the person's last reading, not
// written by the person.
func (a *intentAPI) unread(person any, e *ast.Entity, id any) int {
	total := 0
	for _, c := range a.readChildren(e) {
		key := c.Singular + ":" + e.Singular
		var ate int64
		a.s.DB.DB.QueryRow(fmt.Sprintf(`SELECT ate FROM %s WHERE pessoa_id = %s AND recurso = %s AND recurso_id = %s`, readTable, a.s.ph(1), a.s.ph(2), a.s.ph(3)), person, key, id).Scan(&ate)
		q := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s = %s AND id > %s`, quoteIdent(c.Singular), quoteIdent(c.ReadField), a.s.ph(1), a.s.ph(2))
		args := []any{id, ate}
		if len(c.OwnerFields) > 0 {
			q += fmt.Sprintf(` AND (%s IS NULL OR %s <> %s)`, quoteIdent(c.OwnerFields[0]), quoteIdent(c.OwnerFields[0]), a.s.ph(3))
			args = append(args, person)
		}
		var n int
		a.s.DB.DB.QueryRow(q, args...).Scan(&n)
		total += n
	}
	return total
}

func (a *intentAPI) decorateUnread(atual map[string]any, e *ast.Entity, row, out map[string]any) {
	if atual == nil || row["id"] == nil || len(a.readChildren(e)) == 0 {
		return
	}
	out["nao_lidas"] = a.unread(atual["id"], e, row["id"])
}

func quoteIdent(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }
