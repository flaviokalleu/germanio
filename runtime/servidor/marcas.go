package servidor

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Marks (docs/gep/0030-estrelas-topicos-avatar.md, em teste): `projetos
// recebe estrelas` lets each signed-in person who sees a record mark it once
// (marcar) and take the mark back (desmarcar). The record keeps how many
// marks it has in the field named after them, changed in the same
// transaction as the mark, so lists never count. Who marked what is state
// of the application per person: it is never shown to anyone else, and it
// goes away with the record and with the person.

const marksTable = "_germanio_marcas"

// markLimit bounds how many marks of one person a list reads at once.
const markLimit = 10000

func (a *intentAPI) setupMarks() {
	any := false
	for _, n := range a.app.Order {
		any = any || a.app.Entities[n].Marks != ""
	}
	if !any {
		return
	}
	a.s.DB.DB.Exec(`CREATE TABLE IF NOT EXISTS ` + marksTable + ` (pessoa_id INTEGER NOT NULL, recurso VARCHAR(255) NOT NULL, recurso_id INTEGER NOT NULL, UNIQUE (pessoa_id, recurso, recurso_id))`)
	a.s.DB.DB.Exec(`CREATE INDEX IF NOT EXISTS idx_germanio_marcas_recurso ON ` + marksTable + ` (recurso, recurso_id)`)
}

// markAction performs marcar or desmarcar for atual. Nothing changes when
// the mark is already as asked: the answer is 304 (not modified).
func (a *intentAPI) markAction(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual map[string]any, e *ast.Entity, row map[string]any, verb string) {
	if atual == nil {
		a.fail(w, 401, a.msg("401", e))
		return
	}
	db := a.dbOf(ctx)
	var q string
	if verb == "marcar" {
		switch a.s.DB.Driver {
		case "postgres", "postgresql":
			q = `INSERT INTO %s (pessoa_id, recurso, recurso_id) VALUES (%s, %s, %s) ON CONFLICT DO NOTHING`
		case "mysql":
			q = `INSERT IGNORE INTO %s (pessoa_id, recurso, recurso_id) VALUES (%s, %s, %s)`
		default:
			q = `INSERT OR IGNORE INTO %s (pessoa_id, recurso, recurso_id) VALUES (%s, %s, %s)`
		}
	} else {
		q = `DELETE FROM %s WHERE pessoa_id = %s AND recurso = %s AND recurso_id = %s`
	}
	res, err := db.Executar(fmt.Sprintf(q, marksTable, a.s.ph(1), a.s.ph(2), a.s.ph(3)), atual["id"], e.Singular, row["id"])
	if err != nil {
		a.failErr(w, r, err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	step := "+ 1"
	if verb == "desmarcar" {
		step = "- 1"
	}
	count := quoteIdent(e.Marks)
	if _, err := db.Executar(fmt.Sprintf(`UPDATE %s SET %s = COALESCE(%s, 0) %s WHERE id = %s`, quoteIdent(strings.ToLower(e.Model.Name)), count, count, step, a.s.ph(1)), row["id"]); err != nil {
		a.failErr(w, r, err)
		return
	}
	a.json(w, 200, serializeFor(ctx, a.in, atual, e, a.find(ctx, e, fmt.Sprint(row["id"]), nil), false), nil)
}

// isMarked: atual marked the record (pages: read outside any change).
func (a *intentAPI) isMarked(atual map[string]any, e *ast.Entity, id any) bool {
	if atual == nil || id == nil {
		return false
	}
	var n int
	a.s.DB.DB.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE pessoa_id = %s AND recurso = %s AND recurso_id = %s`, marksTable, a.s.ph(1), a.s.ph(2), a.s.ph(3)), atual["id"], e.Singular, id).Scan(&n)
	return n > 0
}

// markedBy: the records of e atual marked (none for someone not signed in).
func (a *intentAPI) markedBy(atual map[string]any, e *ast.Entity) []any {
	ids := []any{}
	if atual == nil {
		return ids
	}
	rows, err := a.s.DB.DB.Query(fmt.Sprintf(`SELECT recurso_id FROM %s WHERE pessoa_id = %s AND recurso = %s ORDER BY recurso_id DESC LIMIT %d`, marksTable, a.s.ph(1), a.s.ph(2), markLimit), atual["id"], e.Singular)
	if err != nil {
		return ids
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			ids = append(ids, float64(id))
		}
	}
	return ids
}

// dropMarks: the marks of a record being deleted go with it.
func (a *intentAPI) dropMarks(ctx *interp.Context, e *ast.Entity, id any) error {
	if e.Marks == "" {
		return nil
	}
	_, err := a.dbOf(ctx).Executar(fmt.Sprintf(`DELETE FROM %s WHERE recurso = %s AND recurso_id = %s`, marksTable, a.s.ph(1), a.s.ph(2)), e.Singular, id)
	return err
}

// personMarks: the marks of a person being deleted go with them, and the
// records they marked count one less.
func (a *intentAPI) personMarks(ctx *interp.Context, person map[string]any) error {
	db := a.dbOf(ctx)
	any := false
	for _, n := range a.app.Order {
		e := a.app.Entities[n]
		if e.Marks == "" {
			continue
		}
		any = true
		count := quoteIdent(e.Marks)
		q := fmt.Sprintf(`UPDATE %s SET %s = COALESCE(%s, 0) - 1 WHERE id IN (SELECT recurso_id FROM %s WHERE pessoa_id = %s AND recurso = %s)`,
			quoteIdent(strings.ToLower(e.Model.Name)), count, count, marksTable, a.s.ph(1), a.s.ph(2))
		if _, err := db.Executar(q, person["id"], e.Singular); err != nil {
			return err
		}
	}
	if !any {
		return nil
	}
	_, err := db.Executar(fmt.Sprintf(`DELETE FROM %s WHERE pessoa_id = %s`, marksTable, a.s.ph(1)), person["id"])
	return err
}
