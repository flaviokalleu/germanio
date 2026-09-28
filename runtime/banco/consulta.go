package banco

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ErrNaoEncontrado is returned when a lookup by id or filter finds no row.
var ErrNaoEncontrado = errors.New("registro não encontrado")

// ErrConflito wraps unique-constraint violations so callers can answer 409.
var ErrConflito = errors.New("conflito de unicidade")

// ErrCampo reports a filter, order or data key that is not a column.
type ErrCampo struct{ Modelo, Campo string }

func (e *ErrCampo) Error() string {
	return fmt.Sprintf("o modelo '%s' não tem o campo '%s'", e.Modelo, e.Campo)
}

// Consulta describes a parameterised query over one model. Values are never
// interpolated into SQL; field names are checked against the model.
//
// Filter keys are "campo" (igual) or "campo__op" where op is one of
// diferente, maior, menor, maior_igual, menor_igual, contem, comeca_com, em,
// nao_em. A nil value with "campo" means IS NULL; with "campo__diferente",
// IS NOT NULL.
type Consulta struct {
	Filtros map[string]any
	Ordenar string // campo; prefixo "-" = decrescente
	Limite  int
	Pagina  int
	// Busca looks for a substring in BuscaCampos (or every text field).
	Busca       string
	BuscaCampos []string
}

// columns returns the set of queryable columns of a model.
func (b *Banco) columns(modelo string) (map[string]bool, error) {
	model, ok := b.Models[modelo]
	if !ok {
		return nil, fmt.Errorf("modelo '%s' não encontrado", modelo)
	}
	cols := map[string]bool{"id": true, "criado_em": true, "atualizado_em": true}
	for _, f := range model.Fields {
		cols[strings.ToLower(f.Name)] = true
	}
	if model.SoftDelete {
		cols["deletado_em"] = true
	}
	return cols, nil
}

func (b *Banco) where(modelo string, c Consulta) (string, []any, error) {
	cols, err := b.columns(modelo)
	if err != nil {
		return "", nil, err
	}
	model := b.Models[modelo]
	var where []string
	var args []any
	n := 1
	if model.SoftDelete {
		where = append(where, q("deletado_em")+" IS NULL")
	}
	keys := make([]string, 0, len(c.Filtros))
	for k := range c.Filtros {
		keys = append(keys, k)
	}
	sort.Strings(keys) // deterministic SQL for identical filters
	for _, key := range keys {
		val := normalizeArg(c.Filtros[key])
		field, op := key, "igual"
		if i := strings.Index(key, "__"); i > 0 {
			field, op = key[:i], key[i+2:]
		}
		field = strings.ToLower(field)
		if !cols[field] {
			return "", nil, &ErrCampo{modelo, field}
		}
		col := q(field)
		switch op {
		case "igual":
			if val == nil {
				where = append(where, col+" IS NULL")
				continue
			}
			where = append(where, col+" = "+b.ph(n))
		case "diferente":
			if val == nil {
				where = append(where, col+" IS NOT NULL")
				continue
			}
			where = append(where, "("+col+" IS NULL OR "+col+" <> "+b.ph(n)+")")
		case "maior":
			where = append(where, col+" > "+b.ph(n))
		case "menor":
			where = append(where, col+" < "+b.ph(n))
		case "maior_igual":
			where = append(where, col+" >= "+b.ph(n))
		case "menor_igual":
			where = append(where, col+" <= "+b.ph(n))
		case "contem":
			where = append(where, "LOWER("+col+") LIKE "+b.ph(n)+" ESCAPE '\\'")
			val = "%" + escapeLike(strings.ToLower(fmt.Sprint(val))) + "%"
		case "contem_algum":
			// any of the values (a list field containing any of the given items)
			list, ok := c.Filtros[key].([]any)
			if !ok {
				return "", nil, fmt.Errorf("o filtro '%s' exige uma lista", key)
			}
			if len(list) == 0 {
				where = append(where, "1 = 0")
				continue
			}
			var ors []string
			for _, item := range list {
				ors = append(ors, "LOWER("+col+") LIKE "+b.ph(n)+" ESCAPE '\\'")
				args = append(args, "%"+escapeLike(strings.ToLower(fmt.Sprint(item)))+"%")
				n++
			}
			where = append(where, "("+strings.Join(ors, " OR ")+")")
			continue
		case "comeca_com":
			where = append(where, col+" LIKE "+b.ph(n)+" ESCAPE '\\'")
			val = escapeLike(fmt.Sprint(val)) + "%"
		case "em", "nao_em":
			list, ok := c.Filtros[key].([]any)
			if !ok {
				return "", nil, fmt.Errorf("o filtro '%s' exige uma lista", key)
			}
			if len(list) == 0 {
				if op == "em" {
					where = append(where, "1 = 0")
				}
				continue
			}
			phs := make([]string, len(list))
			for i, item := range list {
				phs[i] = b.ph(n)
				args = append(args, normalizeArg(item))
				n++
			}
			not := ""
			if op == "nao_em" {
				not = "NOT "
			}
			where = append(where, col+" "+not+"IN ("+strings.Join(phs, ", ")+")")
			continue
		default:
			return "", nil, fmt.Errorf("operador de filtro desconhecido '%s' em '%s'", op, key)
		}
		args = append(args, val)
		n++
	}
	if c.Busca != "" {
		fields := c.BuscaCampos
		if len(fields) == 0 {
			for _, f := range model.Fields {
				if f.Type.SQLType() == "TEXT" {
					fields = append(fields, strings.ToLower(f.Name))
				}
			}
		}
		var conds []string
		for _, f := range fields {
			f = strings.ToLower(f)
			if !cols[f] {
				return "", nil, &ErrCampo{modelo, f}
			}
			conds = append(conds, "LOWER("+q(f)+") LIKE "+b.ph(n)+" ESCAPE '\\'")
			args = append(args, "%"+escapeLike(strings.ToLower(c.Busca))+"%")
			n++
		}
		if len(conds) > 0 {
			where = append(where, "("+strings.Join(conds, " OR ")+")")
		}
	}
	if len(where) == 0 {
		return "", args, nil
	}
	return " WHERE " + strings.Join(where, " AND "), args, nil
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// Filtrar runs a Consulta and returns the page of rows plus the total count.
func (b *Banco) Filtrar(modelo string, c Consulta) ([]map[string]any, int64, error) {
	whereSQL, args, err := b.where(modelo, c)
	if err != nil {
		return nil, 0, err
	}
	var total int64
	if err := b.x().QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s%s", q(modelo), whereSQL), args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	order, dir := "id", "ASC"
	if c.Ordenar != "" {
		order = strings.ToLower(c.Ordenar)
		if strings.HasPrefix(order, "-") {
			order, dir = order[1:], "DESC"
		}
		cols, _ := b.columns(modelo)
		if !cols[order] {
			return nil, 0, &ErrCampo{modelo, order}
		}
	}
	query := fmt.Sprintf("SELECT * FROM %s%s ORDER BY %s %s", q(modelo), whereSQL, q(order), dir)
	if order != "id" {
		query += ", " + q("id") + " " + dir
	}
	if c.Limite > 0 {
		pagina := c.Pagina
		if pagina < 1 {
			pagina = 1
		}
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", c.Limite, (pagina-1)*c.Limite)
	}
	rows, err := b.x().Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	list, err := scanRowsRaw(rows)
	if err == nil {
		b.tipar(modelo, list)
	}
	return list, total, err
}

// tipar converts driver values back to the declared field types: booleans
// stored as 0/1 become booleans and timestamps become RFC 3339 text.
func (b *Banco) tipar(modelo string, rows []map[string]any) {
	model := b.Models[modelo]
	kinds := map[string]string{"criado_em": "tempo", "atualizado_em": "tempo", "deletado_em": "tempo"}
	for _, f := range model.Fields {
		switch f.Type {
		case "booleano":
			kinds[strings.ToLower(f.Name)] = "bool"
		case "data_hora":
			kinds[strings.ToLower(f.Name)] = "tempo"
		case "lista":
			if f.ListOf == "texto" {
				kinds[strings.ToLower(f.Name)] = "lista_texto"
			} else {
				kinds[strings.ToLower(f.Name)] = "lista_num"
			}
		}
	}
	for _, row := range rows {
		for col, kind := range kinds {
			v, ok := row[col]
			if !ok || v == nil {
				continue
			}
			switch kind {
			case "bool":
				switch x := v.(type) {
				case int64:
					row[col] = x != 0
				case float64:
					row[col] = x != 0
				case string:
					row[col] = x == "1" || x == "true"
				}
			case "lista_texto", "lista_num":
				var items []string
				if str, ok := v.(string); ok {
					json.Unmarshal([]byte(str), &items)
				}
				list := make([]any, 0, len(items))
				for _, it := range items {
					if kind == "lista_num" {
						if n, err := strconv.ParseFloat(it, 64); err == nil {
							list = append(list, n)
							continue
						}
					}
					list = append(list, it)
				}
				row[col] = list
			case "tempo":
				if s, ok := v.(string); ok {
					if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
						row[col] = t.UTC().Format(time.RFC3339)
					}
				}
			}
		}
	}
}

// ContarFiltro counts rows matching filters.
func (b *Banco) ContarFiltro(modelo string, c Consulta) (int64, error) {
	whereSQL, args, err := b.where(modelo, c)
	if err != nil {
		return 0, err
	}
	var total int64
	err = b.x().QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s%s", q(modelo), whereSQL), args...).Scan(&total)
	return total, err
}

// BuscarRegistro returns the full row (including every column) by id.
func (b *Banco) BuscarRegistro(modelo string, id int64) (map[string]any, error) {
	list, _, err := b.Filtrar(modelo, Consulta{Filtros: map[string]any{"id": id}, Limite: 1})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, ErrNaoEncontrado
	}
	return list[0], nil
}

// dataColumns validates the keys of a data map against the model fields.
func (b *Banco) dataColumns(modelo string, dados map[string]any) ([]string, error) {
	model := b.Models[modelo]
	fields := map[string]bool{}
	for _, f := range model.Fields {
		fields[strings.ToLower(f.Name)] = true
	}
	keys := make([]string, 0, len(dados))
	for k := range dados {
		lk := strings.ToLower(k)
		if !fields[lk] {
			return nil, &ErrCampo{modelo, k}
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, nil
}

// CriarMapa inserts a row from a map. Unknown keys are an error, not ignored.
func (b *Banco) CriarMapa(modelo string, dados map[string]any) (map[string]any, error) {
	if _, ok := b.Models[modelo]; !ok {
		return nil, fmt.Errorf("modelo '%s' não encontrado", modelo)
	}
	keys, err := b.dataColumns(modelo, dados)
	if err != nil {
		return nil, err
	}
	if err := b.Validar(modelo, dados); err != nil {
		return nil, err
	}
	cols := make([]string, len(keys))
	phs := make([]string, len(keys))
	vals := make([]any, len(keys))
	for i, k := range keys {
		cols[i] = q(strings.ToLower(k))
		phs[i] = b.ph(i + 1)
		vals[i] = normalizeArg(dados[k])
	}
	var query string
	if len(cols) == 0 {
		query = fmt.Sprintf("INSERT INTO %s DEFAULT VALUES", q(modelo))
	} else {
		query = fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", q(modelo), strings.Join(cols, ", "), strings.Join(phs, ", "))
	}
	var id int64
	if b.Driver == "postgres" || b.Driver == "postgresql" {
		err = b.x().QueryRow(query+" RETURNING "+q("id"), vals...).Scan(&id)
	} else {
		var res sql.Result
		res, err = b.x().Exec(query, vals...)
		if err == nil {
			id, err = res.LastInsertId()
		}
	}
	if err != nil {
		return nil, classify(err)
	}
	return b.BuscarRegistro(modelo, id)
}

// AtualizarMapa changes only the given keys (partial update).
func (b *Banco) AtualizarMapa(modelo string, id int64, dados map[string]any) (map[string]any, error) {
	if _, ok := b.Models[modelo]; !ok {
		return nil, fmt.Errorf("modelo '%s' não encontrado", modelo)
	}
	keys, err := b.dataColumns(modelo, dados)
	if err != nil {
		return nil, err
	}
	if err := b.ValidarParcial(modelo, dados); err != nil {
		return nil, err
	}
	sets := make([]string, 0, len(keys)+1)
	vals := make([]any, 0, len(keys)+1)
	for i, k := range keys {
		sets = append(sets, q(strings.ToLower(k))+" = "+b.ph(i+1))
		vals = append(vals, normalizeArg(dados[k]))
	}
	sets = append(sets, q("atualizado_em")+" = CURRENT_TIMESTAMP")
	vals = append(vals, id)
	res, err := b.x().Exec(fmt.Sprintf("UPDATE %s SET %s WHERE %s = %s", q(modelo), strings.Join(sets, ", "), q("id"), b.ph(len(keys)+1)), vals...)
	if err != nil {
		return nil, classify(err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrNaoEncontrado
	}
	return b.BuscarRegistro(modelo, id)
}

// DeletarFiltro removes every row matching filters and returns how many.
// An empty filter is refused so a typo never wipes a table.
func (b *Banco) DeletarFiltro(modelo string, c Consulta) (int64, error) {
	if len(c.Filtros) == 0 {
		return 0, fmt.Errorf("apagar sem filtro não é permitido; use um filtro explícito")
	}
	whereSQL, args, err := b.where(modelo, c)
	if err != nil {
		return 0, err
	}
	var res sql.Result
	if b.Models[modelo].SoftDelete {
		res, err = b.x().Exec(fmt.Sprintf("UPDATE %s SET %s = CURRENT_TIMESTAMP%s", q(modelo), q("deletado_em"), whereSQL), args...)
	} else {
		res, err = b.x().Exec(fmt.Sprintf("DELETE FROM %s%s", q(modelo), whereSQL), args...)
	}
	if err != nil {
		return 0, classify(err)
	}
	return res.RowsAffected()
}

// ValidarParcial applies field rules only to keys present in dados.
func (b *Banco) ValidarParcial(modelo string, dados map[string]any) error {
	return b.validar(modelo, dados, true)
}

// Sequencia atomically increments and returns the counter named chave,
// starting at 1. It backs per-scope numbering such as #1, #2 per project.
func (b *Banco) Sequencia(chave string) (v int64, err error) {
	if _, err := b.x().Exec(`CREATE TABLE IF NOT EXISTS ` + q("_germanio_sequencias") + ` (` + q("chave") + ` VARCHAR(255) PRIMARY KEY, ` + q("valor") + ` BIGINT NOT NULL)`); err != nil {
		return 0, err
	}
	var tx executor = b.tx
	if b.tx == nil {
		t, e := b.DB.Begin()
		if e != nil {
			return 0, e
		}
		defer t.Rollback()
		defer func() {
			if err == nil {
				err = t.Commit()
			}
		}()
		tx = t
	}
	// Upsert then read inside one transaction; SQLite serialises writers and
	// PostgreSQL/MySQL lock the row for the rest of the transaction.
	var upsert string
	switch b.Driver {
	case "mysql":
		upsert = "INSERT INTO " + q("_germanio_sequencias") + " (" + q("chave") + ", " + q("valor") + ") VALUES (?, 1) ON DUPLICATE KEY UPDATE " + q("valor") + " = " + q("valor") + " + 1"
	default:
		upsert = "INSERT INTO " + q("_germanio_sequencias") + " (" + q("chave") + ", " + q("valor") + ") VALUES (" + b.ph(1) + ", 1) ON CONFLICT (" + q("chave") + ") DO UPDATE SET " + q("valor") + " = " + q("_germanio_sequencias") + "." + q("valor") + " + 1"
	}
	if _, err := tx.Exec(upsert, chave); err != nil {
		return 0, err
	}
	if err = tx.QueryRow("SELECT "+q("valor")+" FROM "+q("_germanio_sequencias")+" WHERE "+q("chave")+" = "+b.ph(1), chave).Scan(&v); err != nil {
		return 0, err
	}
	return v, nil
}

// executor is what data operations run on: the pool or the current transaction.
type executor interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// Travar locks a row until the current transaction ends, so invariants that
// read before writing (the last owner) cannot race. SQLite needs nothing:
// its transactions take the write lock up front (_txlock=immediate).
func (b *Banco) Travar(modelo string, id any) error {
	if b.tx == nil || b.Driver == "sqlite" || b.Driver == "" {
		return nil
	}
	var x any
	err := b.tx.QueryRow(fmt.Sprintf("SELECT %s FROM %s WHERE %s = %s FOR UPDATE", q("id"), q(modelo), q("id"), b.ph(1)), normalizeArg(id)).Scan(&x)
	if err == sql.ErrNoRows {
		return nil
	}
	return err
}

// Executar runs a statement on the current transaction (or the pool).
func (b *Banco) Executar(query string, args ...any) (sql.Result, error) {
	return b.x().Exec(query, args...)
}

func (b *Banco) x() executor {
	if b.tx != nil {
		return b.tx
	}
	return b.DB
}

// EmTransacao runs fn with a Banco whose operations share one transaction:
// all of them are kept if fn succeeds, none if it fails or panics. Nested
// calls join the outer transaction.
func (b *Banco) EmTransacao(fn func(tx *Banco) error) (err error) {
	if b.tx != nil {
		return fn(b)
	}
	t, err := b.DB.Begin()
	if err != nil {
		return err
	}
	inner := *b
	inner.tx = t
	defer func() {
		if r := recover(); r != nil {
			t.Rollback()
			panic(r)
		}
		if err != nil {
			t.Rollback()
			return
		}
		err = t.Commit()
	}()
	return fn(&inner)
}

func classify(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate") {
		return fmt.Errorf("%w: %s", ErrConflito, err.Error())
	}
	return err
}

// normalizeArg converts interpreter values into driver arguments.
func normalizeArg(v any) any {
	switch x := v.(type) {
	case float64:
		if x == float64(int64(x)) {
			return int64(x)
		}
		return x
	case bool:
		return x
	case map[string]any, []any:
		return fmt.Sprint(x)
	}
	return v
}

// scanRowsRaw converts rows into maps with portable value types: integers as
// int64, text as string, times as RFC 3339 strings. Unlike scanRows it keeps
// every column: code in .ge decides what to expose.
func scanRowsRaw(rows *sql.Rows) ([]map[string]any, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	results := []map[string]any{}
	for rows.Next() {
		values := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(columns))
		for i, col := range columns {
			switch v := values[i].(type) {
			case []byte:
				row[col] = string(v)
			case time.Time:
				row[col] = v.UTC().Format(time.RFC3339)
			default:
				row[col] = v
			}
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

// AtualizarOnde updates rows matching filters and returns how many changed.
// With filters that include the current state it is an atomic claim
// (estado = pendente → executando happens once, whoever asks first).
func (b *Banco) AtualizarOnde(modelo string, c Consulta, dados map[string]any) (int64, error) {
	if len(c.Filtros) == 0 {
		return 0, fmt.Errorf("atualizar sem filtro não é permitido")
	}
	whereSQL, args, err := b.where(modelo, c)
	if err != nil {
		return 0, err
	}
	keys, err := b.dataColumns(modelo, dados)
	if err != nil {
		return 0, err
	}
	sets := make([]string, 0, len(keys)+1)
	vals := make([]any, 0, len(keys)+len(args))
	n := len(args) + 1
	for _, k := range keys {
		sets = append(sets, q(strings.ToLower(k))+" = "+b.ph(n))
		vals = append(vals, normalizeArg(dados[k]))
		n++
	}
	sets = append(sets, q("atualizado_em")+" = CURRENT_TIMESTAMP")
	// Placeholders: WHERE args come first for PostgreSQL numbering.
	query := fmt.Sprintf("UPDATE %s SET %s%s", q(modelo), strings.Join(sets, ", "), whereSQL)
	all := append(append([]any{}, args...), vals...)
	if b.Driver != "postgres" && b.Driver != "postgresql" {
		// '?' placeholders are positional in textual order: SET values first.
		all = append(append([]any{}, vals...), args...)
	}
	res, err := b.x().Exec(query, all...)
	if err != nil {
		return 0, classify(err)
	}
	return res.RowsAffected()
}
