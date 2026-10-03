package banco

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/runtime/cofre"
)

// Secrets at rest (docs/gep/0049-segredos-guardados.md, em teste). A sealed
// field (ast.Field.Sealed: a hidden text people write, or a credential a
// capability keeps) is encrypted by the Banco on every write and opened on
// every read that returns whole records to the runtime (Filtrar,
// BuscarRegistro and what is built on them). The legacy readers (Listar,
// Buscar…) never return sealed columns at all. The database therefore only
// ever holds ciphertext — as long as the server has a lasting key; without
// one (development) values stay in clear until a key is set, and the next
// start seals them (SelarLegados).
//
// A sealed column cannot be compared by the database (each ciphertext is
// different, even for the same text), and comparing it would reveal it to
// whoever filters: filters, search, ordering and grouping on it are refused
// with ErrSegredo.

// ErrSegredo: an operation that would compare or expose a sealed field.
type ErrSegredo struct{ Modelo, Campo, Uso string }

func (e *ErrSegredo) Error() string {
	return fmt.Sprintf("o campo '%s' de '%s' é oculto e fica cifrado no banco: não pode ser usado para %s (o banco não compara valores cifrados, e a comparação revelaria o valor). Use outro campo", e.Campo, e.Modelo, e.Uso)
}

// UsarCofre makes every later write seal and every read open with c. A nil
// Cofre keeps values in clear (no lasting key: development only).
func (b *Banco) UsarCofre(c *cofre.Cofre) { b.Cofre = c }

// sealedOf lists the sealed columns of a table.
func (b *Banco) sealedOf(modelo string) map[string]bool { return b.sealedCols[modelo] }

// indexSealed records the sealed columns of each table (once, at Abrir).
func (b *Banco) indexSealed() {
	b.sealedCols = map[string]map[string]bool{}
	for name, m := range b.Models {
		for _, f := range m.Fields {
			if f.Sealed {
				if b.sealedCols[name] == nil {
					b.sealedCols[name] = map[string]bool{}
				}
				b.sealedCols[name][strings.ToLower(f.Name)] = true
			}
		}
	}
}

// TemSegredos reports whether some model has a sealed field.
func TemSegredos(models []*ast.Model) bool {
	for _, m := range models {
		for _, f := range m.Fields {
			if f.Sealed {
				return true
			}
		}
	}
	return false
}

func purpose(modelo, col string) string { return modelo + "." + col }

// sealArg turns the value about to be written in column col of modelo into
// what the database keeps.
func (b *Banco) sealArg(modelo, col string, v any) (any, error) {
	v = normalizeArg(v)
	if b.Cofre == nil || v == nil || !b.sealedOf(modelo)[strings.ToLower(col)] {
		return v, nil
	}
	s, ok := v.(string)
	if !ok {
		s = fmt.Sprint(v)
	}
	if s == "" {
		return s, nil // nothing to hide; keeps "no value" distinguishable from a value
	}
	return b.Cofre.Selar(purpose(modelo, strings.ToLower(col)), s)
}

var unreadable sync.Map // table.column already reported as unreadable

// openRows replaces the sealed values of rows by their clear text. A value
// that does not open (a key that is gone, an altered row) becomes nil and is
// reported once, without its content.
func (b *Banco) openRows(modelo string, rows []map[string]any) {
	cols := b.sealedOf(modelo)
	if len(cols) == 0 {
		return
	}
	for _, row := range rows {
		for col := range cols {
			s, ok := row[col].(string)
			if !ok || !cofre.Selado(s) {
				continue
			}
			if b.Cofre == nil {
				row[col] = nil
				b.reportUnreadable(modelo, col, cofre.ErrChaveDesconhecida)
				continue
			}
			plain, err := b.Cofre.Abrir(purpose(modelo, col), s)
			if err != nil {
				row[col] = nil
				b.reportUnreadable(modelo, col, err)
				continue
			}
			row[col] = plain
		}
	}
}

func (b *Banco) reportUnreadable(modelo, col string, err error) {
	if _, seen := unreadable.LoadOrStore(purpose(modelo, col), true); !seen {
		fmt.Printf("[germanio] segredos: um valor de %s.%s não abre (%v); ele é tratado como vazio. Confira GERMANIO_SEGREDO e GERMANIO_SEGREDO_ANTERIOR\n", modelo, col, err)
	}
}

// dropSealed removes sealed columns from rows of the legacy readers, which
// hand records to old routes and exports as they are.
func (b *Banco) dropSealed(modelo string, rows []map[string]any) []map[string]any {
	for col := range b.sealedOf(modelo) {
		for _, row := range rows {
			delete(row, col)
		}
	}
	return rows
}

// RelatorioSegredos says what SelarLegados did.
type RelatorioSegredos struct {
	Campos     int // sealed columns in the program
	Cifrados   int // values in clear sealed now
	Recifrados int // values sealed with a previous key, sealed again with the current one
	EmClaro    int // values left in clear (no key)
}

// lote: how many rows SelarLegados rewrites per transaction.
const lote = 500

// SelarLegados brings every sealed column to the current key: values in
// clear (written before the field was sealed, or while the server had no
// key) are sealed, and values sealed with a previous key
// (GERMANIO_SEGREDO_ANTERIOR) are sealed again with the current one. It is
// idempotent and safe while other servers write: each row is rewritten only
// if it still holds the value read (compare and swap), in batches, and
// readers accept both forms meanwhile. A value sealed with a key this server
// does not have stops the start with the reason; without a key, sealed
// values stop the start too (they could not be read), and values in clear
// are only counted.
func (b *Banco) SelarLegados() (RelatorioSegredos, error) {
	var rel RelatorioSegredos
	var unknown []string
	for _, modelo := range sortedKeys(b.Models) {
		cols := b.sealedOf(modelo)
		for _, col := range sortedKeys(cols) {
			rel.Campos++
			if b.Cofre == nil {
				var sealed, clear int
				b.DB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s LIKE %s", q(modelo), q(col), b.ph(1)), cofre.Prefixo+"%").Scan(&sealed)
				if sealed > 0 {
					return rel, fmt.Errorf("o banco tem %d valor(es) cifrado(s) em %s.%s, mas este servidor não tem GERMANIO_SEGREDO.\nPor quê: sem a chave, eles não podem ser lidos (webhooks e conexões com outros sistemas parariam).\nComo corrigir: defina GERMANIO_SEGREDO com a mesma chave usada quando eles foram guardados", sealed, modelo, col)
				}
				b.DB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s IS NOT NULL AND %s <> ''", q(modelo), q(col), q(col))).Scan(&clear)
				rel.EmClaro += clear
				continue
			}
			n, re, bad, err := b.sealColumn(modelo, col)
			rel.Cifrados += n
			rel.Recifrados += re
			if err != nil {
				return rel, err
			}
			if bad > 0 {
				unknown = append(unknown, fmt.Sprintf("%s.%s (%d)", modelo, col, bad))
			}
		}
	}
	if len(unknown) > 0 {
		return rel, fmt.Errorf("o banco tem segredos cifrados com uma chave que este servidor não tem: %s.\nPor quê: a chave de quando eles foram guardados não é a GERMANIO_SEGREDO atual nem uma das GERMANIO_SEGREDO_ANTERIOR.\nComo corrigir: ao trocar a chave, defina GERMANIO_SEGREDO com a nova e GERMANIO_SEGREDO_ANTERIOR com a antiga; na partida tudo é cifrado de novo com a nova, e depois a antiga pode sair", strings.Join(unknown, ", "))
	}
	if rel.Cifrados > 0 && (b.Driver == "sqlite" || b.Driver == "") {
		// the clear text overwritten may survive in free pages and in the
		// write-ahead log: rebuild the file once so it is really gone
		b.DB.Exec("VACUUM")
		b.DB.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	}
	return rel, nil
}

// sealColumn rewrites the values of one column not yet sealed with the
// current key; it returns how many were in clear, how many were sealed again
// and how many have an unknown key.
func (b *Banco) sealColumn(modelo, col string) (clear, again, unknown int, err error) {
	var last int64
	pending := fmt.Sprintf("SELECT %s, %s FROM %s WHERE %s IS NOT NULL AND %s <> '' AND %s NOT LIKE %s AND %s > %s ORDER BY %s LIMIT %d",
		q("id"), q(col), q(modelo), q(col), q(col), q(col), b.ph(1), q("id"), b.ph(2), q("id"), lote)
	for {
		rows, err := b.DB.Query(pending, b.Cofre.PrefixoAtual()+"%", last)
		if err != nil {
			return clear, again, unknown, err
		}
		type item struct {
			id  int64
			old string
		}
		var batch []item
		for rows.Next() {
			var it item
			var v any
			if err := rows.Scan(&it.id, &v); err != nil {
				rows.Close()
				return clear, again, unknown, err
			}
			switch x := v.(type) {
			case string:
				it.old = x
			case []byte:
				it.old = string(x)
			default:
				it.old = fmt.Sprint(x)
			}
			batch = append(batch, it)
		}
		rows.Close()
		if len(batch) == 0 {
			return clear, again, unknown, nil
		}
		err = b.EmTransacao(func(tx *Banco) error {
			for _, it := range batch {
				if cofre.Selado(it.old) && !b.Cofre.Conhece(it.old) {
					unknown++
					continue
				}
				plain, err := b.Cofre.Abrir(purpose(modelo, col), it.old)
				if err != nil {
					unknown++ // altered or moved: never overwritten, reported
					continue
				}
				sealed, err := b.Cofre.Selar(purpose(modelo, col), plain)
				if err != nil {
					return err
				}
				res, err := tx.x().Exec(fmt.Sprintf("UPDATE %s SET %s = %s WHERE %s = %s AND %s = %s", q(modelo), q(col), b.ph(1), q("id"), b.ph(2), q(col), b.ph(3)), sealed, it.id, it.old)
				if err != nil {
					return err
				}
				if n, _ := res.RowsAffected(); n == 1 {
					if cofre.Selado(it.old) {
						again++
					} else {
						clear++
					}
				}
			}
			return nil
		})
		if err != nil {
			return clear, again, unknown, err
		}
		last = batch[len(batch)-1].id
		if len(batch) < lote {
			return clear, again, unknown, nil
		}
	}
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
