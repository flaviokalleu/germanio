package banco

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
)

// Migrations never disguise a destructive change as an inferred rename
// (G93). Germanio renames a field only when the program says so
// (renomeie a para b); when a column with data disappears from the
// declaration while a new field appears, it stops and asks, because it
// cannot prove the two hold the same information.

// colInfo is what the migration knows about a column of the real table.
type colInfo struct {
	family  string // text, integer, real, date, datetime, boolean, other
	hasData bool
}

// MigrationError explains a migration Germanio refuses to guess.
type MigrationError struct {
	Table   string
	Message string
}

func (e *MigrationError) Error() string { return e.Message }

// systemColumns are kept by the runtime even when not declared.
var systemColumns = map[string]bool{"id": true, "criado_em": true, "atualizado_em": true, "deletado_em": true, "role": true}

func typeFamily(sqlType string) string {
	t := strings.ToUpper(sqlType)
	switch {
	case strings.Contains(t, "CHAR"), strings.Contains(t, "TEXT"), strings.Contains(t, "CLOB"):
		return "text"
	case strings.Contains(t, "BOOL"):
		return "boolean"
	case strings.Contains(t, "INT"):
		return "integer"
	case strings.Contains(t, "REAL"), strings.Contains(t, "FLOA"), strings.Contains(t, "DOUB"), strings.Contains(t, "NUMERIC"), strings.Contains(t, "DECIMAL"):
		return "real"
	case t == "DATE":
		return "date"
	case strings.Contains(t, "TIME"):
		return "datetime"
	}
	return "other"
}

// inspect reads the columns of table, their type family and whether each one
// holds any value. ok=false: the table does not exist yet.
func (b *Banco) inspect(table string) (map[string]colInfo, bool, error) {
	rows, err := b.DB.Query(fmt.Sprintf("SELECT * FROM %s LIMIT 0", q(table)))
	if err != nil {
		return nil, false, nil // no table: nothing to migrate
	}
	types, err := rows.ColumnTypes()
	rows.Close()
	if err != nil {
		return nil, false, err
	}
	out := map[string]colInfo{}
	for _, ct := range types {
		name := strings.ToLower(ct.Name())
		info := colInfo{family: typeFamily(ct.DatabaseTypeName())}
		var one int
		if err := b.DB.QueryRow(fmt.Sprintf("SELECT 1 FROM %s WHERE %s IS NOT NULL LIMIT 1", q(table), q(ct.Name()))).Scan(&one); err == nil {
			info.hasData = true
		}
		out[name] = info
	}
	return out, true, nil
}

// renameStep renames From to To; ReplaceEmpty first removes an empty To
// column (created by an earlier start that did not know the rename yet).
type renameStep struct {
	From, To     string
	ReplaceEmpty bool
}

// migrationPlan is what a start would do to one table.
type migrationPlan struct {
	Renames []renameStep
	Orphans []string // columns with data that are no longer declared (warned)
}

// planMigration decides, without touching the database, how the table
// becomes the model. It refuses what it cannot prove safe.
func planMigration(model *ast.Model, cols map[string]colInfo, driverType func(*ast.Field) string) (migrationPlan, error) {
	var plan migrationPlan
	table := strings.ToLower(model.Name)
	declared := map[string]*ast.Field{}
	for _, f := range model.Fields {
		declared[strings.ToLower(f.Name)] = f
	}
	renamed := map[string]bool{}
	for _, rn := range model.Renames {
		from, to := strings.ToLower(rn.From), strings.ToLower(rn.To)
		old, hasFrom := cols[from]
		cur, hasTo := cols[to]
		switch {
		case !hasFrom:
			continue // applied before, or nothing to rename (a new database)
		case hasTo && cur.hasData:
			return plan, &MigrationError{table, fmt.Sprintf("%s: renomeie %s para %s, mas %s já existe e tem dados. O rename não sobrescreve dados: decida o que fazer com os valores de %s antes", table, from, to, to, to)}
		}
		if f := declared[to]; f != nil && old.hasData && old.family != "other" && old.family != typeFamily(driverType(f)) {
			return plan, &MigrationError{table, fmt.Sprintf("%s: renomear %s para %s e mudar o tipo (de %s para %s) ao mesmo tempo não é seguro. Faça em duas etapas: primeiro o rename com o tipo antigo, depois a mudança de tipo", table, from, to, old.family, typeFamily(driverType(f)))}
		}
		plan.Renames = append(plan.Renames, renameStep{From: from, To: to, ReplaceEmpty: hasTo})
		renamed[from], renamed[to] = true, true
	}
	discarded := map[string]bool{}
	for _, d := range model.Discarded {
		discarded[strings.ToLower(d)] = true
	}
	// columns with data that the program no longer declares, and declared
	// fields the table does not have yet
	var orphans, missing []string
	for name, info := range cols {
		if declared[name] == nil && !systemColumns[name] && !renamed[name] && !discarded[name] && info.hasData {
			orphans = append(orphans, name)
		}
	}
	for name := range declared {
		if _, ok := cols[name]; !ok && !renamed[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(orphans)
	sort.Strings(missing)
	if len(orphans) > 0 && len(missing) > 0 {
		// A field with data disappeared and another appeared: maybe a rename,
		// maybe not. Germanio does not decide by the names alone.
		old, novo := orphans[0], bestMatch(orphans[0], missing)
		return plan, &MigrationError{table, fmt.Sprintf(`%s: parece que %q pode ter sido renomeado para %q.

Germanio não pode provar que os dois campos representam a mesma informação.

Para preservar os dados, declare no bloco de %s:

    renomeie %s para %s

Se %s foi removido de propósito (os dados antigos ficam guardados, fora da aplicação), declare:

    descarte %s`, table, old, novo, table, old, novo, old, old)}
	}
	plan.Orphans = orphans
	return plan, nil
}

// bestMatch picks, among candidates, the name closest to old (only to order
// the suggestion; the decision is the programmer's).
func bestMatch(old string, candidates []string) string {
	best, score := candidates[0], -1
	for _, c := range candidates {
		s := commonPrefix(old, c)
		if strings.Contains(c, old) || strings.Contains(old, c) {
			s += 10
		}
		if s > score {
			best, score = c, s
		}
	}
	return best
}

func commonPrefix(a, b string) int {
	n := 0
	for n < len(a) && n < len(b) && a[n] == b[n] {
		n++
	}
	return n
}

// sqlTypeFor is the column type of a field for this driver.
func (b *Banco) sqlTypeFor(f *ast.Field) string {
	t := f.Type.SQLType()
	if b.Driver == "mysql" && t == "TEXT" {
		t = "VARCHAR(500)"
	}
	return t
}

// applyRenames runs the renames of a plan in one transaction: all of them
// happen, or none.
func (b *Banco) applyRenames(table string, steps []renameStep) error {
	if len(steps) == 0 {
		return nil
	}
	tx, err := b.DB.Begin()
	if err != nil {
		return err
	}
	for _, st := range steps {
		if st.ReplaceEmpty {
			for _, idx := range []string{"uq_" + table + "_" + st.To, "idx_" + table + "_" + st.To} {
				if _, err := tx.Exec("DROP INDEX IF EXISTS " + q(idx)); err != nil {
					tx.Rollback()
					return err
				}
			}
			if _, err := tx.Exec(fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", q(table), q(st.To))); err != nil {
				tx.Rollback()
				return fmt.Errorf("renomeie %s para %s: não consegui trocar a coluna vazia %s: %w", st.From, st.To, st.To, err)
			}
		}
		if _, err := tx.Exec(fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", q(table), q(st.From), q(st.To))); err != nil {
			tx.Rollback()
			return fmt.Errorf("renomeie %s para %s: %w", st.From, st.To, err)
		}
	}
	return tx.Commit()
}

// Verificar checks, without changing anything, what starting the program
// would do to an existing database: the renames it would apply and the
// migrations it would refuse (ge check).
func Verificar(config *ast.DatabaseConfig, appName string, models []*ast.Model, exists func(string) bool) ([]string, error) {
	if config == nil {
		config = ast.DefaultDatabase()
	}
	driver, dsn := buildDSN(config, appName)
	if driver == "sqlite" && !exists(strings.SplitN(dsn, "?", 2)[0]) {
		return nil, nil // no database yet: nothing to check
	}
	db, err := sql.Open(driver, dsn)
	if err != nil || db.Ping() != nil {
		return nil, nil // not reachable from here: the start will tell
	}
	defer db.Close()
	b := &Banco{DB: db, Driver: config.Driver, Models: map[string]*ast.Model{}}
	var notes []string
	for _, m := range models {
		cols, ok, err := b.inspect(strings.ToLower(m.Name))
		if err != nil || !ok {
			continue
		}
		plan, err := planMigration(m, cols, b.sqlTypeFor)
		if err != nil {
			return notes, err
		}
		for _, st := range plan.Renames {
			notes = append(notes, fmt.Sprintf("%s: a próxima partida renomeia %s para %s, preservando os dados", strings.ToLower(m.Name), st.From, st.To))
		}
	}
	return notes, nil
}
