package banco

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/flaviokalleu/germanio/compiler/ast"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

// Banco wraps the database connection and model metadata.
type Banco struct {
	DB     *sql.DB
	Models map[string]*ast.Model
	Driver string      // "sqlite", "mysql", "postgres"
	Rules  []*ast.Rule // user-defined validation rules
}

// Abrir creates the database and tables from model definitions.
func Abrir(config *ast.DatabaseConfig, appName string, models []*ast.Model) (*Banco, error) {
	if config == nil {
		config = ast.DefaultDatabase()
	}

	driver, dsn := buildDSN(config, appName)

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar (%s): %w", config.Driver, err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("erro ao conectar ao banco %s: %w", config.Driver, err)
	}

	// Connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	fmt.Printf("[germanio] Banco: %s\n", config.Driver)

	b := &Banco{
		DB:     db,
		Models: make(map[string]*ast.Model),
		Driver: config.Driver,
	}

	for _, m := range models {
		b.Models[strings.ToLower(m.Name)] = m
		if err := b.criarTabela(m); err != nil {
			return nil, fmt.Errorf("erro ao criar tabela '%s': %w", m.Name, err)
		}
		fmt.Printf("[germanio] Tabela: %s (%d campos)\n", m.Name, len(m.Fields))
	}

	// Create join tables for many-to-many relationships
	for _, m := range models {
		for _, related := range m.ManyToMany {
			relLower := strings.ToLower(related)
			mLower := strings.ToLower(m.Name)
			// Sort names alphabetically for consistent table naming
			name1, name2 := mLower, relLower
			if name1 > name2 {
				name1, name2 = name2, name1
			}
			joinTable := name1 + "_" + name2
			// Check if already created
			if _, exists := b.Models[joinTable]; exists {
				continue
			}
			joinSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
				%s INTEGER REFERENCES %s(%s) ON DELETE CASCADE,
				%s INTEGER REFERENCES %s(%s) ON DELETE CASCADE,
				PRIMARY KEY (%s, %s)
			)`, q(joinTable),
				q(mLower+"_id"), q(mLower), q("id"),
				q(relLower+"_id"), q(relLower), q("id"),
				q(mLower+"_id"), q(relLower+"_id"))
			if _, err := b.DB.Exec(joinSQL); err != nil {
				fmt.Printf("[germanio] AVISO: erro ao criar tabela join '%s': %s\n", joinTable, err)
			} else {
				fmt.Printf("[germanio] Tabela join: %s\n", joinTable)
			}
		}
	}

	return b, nil
}

func buildDSN(config *ast.DatabaseConfig, appName string) (driver string, dsn string) {
	switch strings.ToLower(config.Driver) {
	case "mysql":
		host := config.Host
		if host == "" {
			host = "localhost"
		}
		port := config.Port
		if port == "" {
			port = "3306"
		}
		dbName := config.Name
		if dbName == "" {
			dbName = appName
		}
		user := config.User
		if user == "" {
			user = "root"
		}
		// user:password@tcp(host:port)/dbname?parseTime=true
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4",
			user, config.Password, host, port, dbName)
		return "mysql", dsn

	case "postgres", "postgresql":
		host := config.Host
		if host == "" {
			host = "localhost"
		}
		port := config.Port
		if port == "" {
			port = "5432"
		}
		dbName := config.Name
		if dbName == "" {
			dbName = appName
		}
		user := config.User
		if user == "" {
			user = "postgres"
		}
		sslmode := "disable"
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, config.Password, dbName, sslmode)
		return "postgres", dsn

	default: // sqlite
		dbName := config.Name
		if dbName == "" {
			dbName = appName + ".db"
		}
		// GERMANIO_SQLITE overrides the file (tests, deploys, ":memory:"-like temp files).
		if override := os.Getenv("GERMANIO_SQLITE"); override != "" {
			dbName = override
		}
		// busy_timeout: concurrent writers (requests, task workers) wait instead of failing.
		return "sqlite", dbName + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	}
}

// q quotes a SQL identifier.
func q(name string) string {
	return `"` + name + `"`
}

// placeholder returns the correct placeholder for the driver.
func (b *Banco) ph(n int) string {
	if b.Driver == "postgres" || b.Driver == "postgresql" {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

// placeholders returns N placeholders for the driver.
func (b *Banco) placeholders(count int) []string {
	phs := make([]string, count)
	for i := range phs {
		phs[i] = b.ph(i + 1)
	}
	return phs
}

func (b *Banco) criarTabela(model *ast.Model) error {
	name := strings.ToLower(model.Name)

	autoInc := "AUTOINCREMENT"
	if b.Driver == "mysql" {
		autoInc = "AUTO_INCREMENT"
	}
	if b.Driver == "postgres" || b.Driver == "postgresql" {
		autoInc = "" // use SERIAL instead
	}

	var cols []string
	if b.Driver == "postgres" || b.Driver == "postgresql" {
		cols = append(cols, q("id")+" SERIAL PRIMARY KEY")
	} else {
		cols = append(cols, q("id")+" INTEGER PRIMARY KEY "+autoInc)
	}

	for _, f := range model.Fields {
		sqlType := f.Type.SQLType()
		// Adjust types per driver
		if b.Driver == "mysql" {
			if sqlType == "TEXT" {
				sqlType = "VARCHAR(500)"
			}
		}

		col := fmt.Sprintf("%s %s", q(strings.ToLower(f.Name)), sqlType)
		if f.Required {
			col += " NOT NULL"
		}
		if f.Unique {
			col += " UNIQUE"
		}
		if f.Default != "" {
			col += fmt.Sprintf(" DEFAULT '%s'", f.Default)
		}
		if f.Reference != "" {
			col += fmt.Sprintf(" REFERENCES %s(%s)", q(strings.ToLower(f.Reference)), q("id"))
		}
		cols = append(cols, col)
	}

	tsDefault := "CURRENT_TIMESTAMP"
	tsType := "DATETIME"
	if b.Driver == "postgres" || b.Driver == "postgresql" {
		tsType = "TIMESTAMP"
	}

	hasCriadoEm := false
	hasAtualizadoEm := false
	for _, f := range model.Fields {
		switch strings.ToLower(f.Name) {
		case "criado_em":
			hasCriadoEm = true
		case "atualizado_em":
			hasAtualizadoEm = true
		}
	}

	if !hasCriadoEm {
		cols = append(cols, fmt.Sprintf("%s %s DEFAULT %s", q("criado_em"), tsType, tsDefault))
	}
	if !hasAtualizadoEm {
		cols = append(cols, fmt.Sprintf("%s %s DEFAULT %s", q("atualizado_em"), tsType, tsDefault))
	}

	// Soft delete column
	if model.SoftDelete {
		cols = append(cols, fmt.Sprintf("%s %s DEFAULT NULL", q("deletado_em"), tsType))
	}

	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n  %s\n)", q(name), strings.Join(cols, ",\n  "))
	if _, err := b.DB.Exec(query); err != nil {
		return err
	}

	// After CREATE TABLE IF NOT EXISTS, add missing columns
	for _, f := range model.Fields {
		fname := strings.ToLower(f.Name)
		sqlType := f.Type.SQLType()
		if b.Driver == "mysql" && sqlType == "TEXT" {
			sqlType = "VARCHAR(500)"
		}
		alterSQL := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", q(name), q(fname), sqlType)
		if f.Default != "" {
			alterSQL += fmt.Sprintf(" DEFAULT '%s'", f.Default)
		}
		b.DB.Exec(alterSQL) // ignore error if column already exists
	}

	// Composite constraints: unico(a, b) and indice(a, b)
	for _, group := range model.UniqueTogether {
		if err := b.compositeIndex(name, group, true); err != nil {
			return err
		}
	}
	for _, group := range model.IndexTogether {
		if err := b.compositeIndex(name, group, false); err != nil {
			return err
		}
	}

	// Create indexes for fields with Index: true
	for _, f := range model.Fields {
		if f.Index {
			fname := strings.ToLower(f.Name)
			idxQuery := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s(%s)",
				q(fmt.Sprintf("idx_%s_%s", name, fname)), q(name), q(fname))
			if _, err := b.DB.Exec(idxQuery); err != nil {
				return fmt.Errorf("erro ao criar índice para '%s.%s': %w", name, fname, err)
			}
		}
	}

	return nil
}

func (b *Banco) compositeIndex(table string, group []string, unique bool) error {
	cols := make([]string, len(group))
	for i, c := range group {
		cols[i] = q(strings.ToLower(c))
	}
	kind, prefix := "INDEX", "idx"
	if unique {
		kind, prefix = "UNIQUE INDEX", "uniq"
	}
	name := fmt.Sprintf("%s_%s_%s", prefix, table, strings.ToLower(strings.Join(group, "_")))
	stmt := fmt.Sprintf("CREATE %s IF NOT EXISTS %s ON %s(%s)", kind, q(name), q(table), strings.Join(cols, ", "))
	if _, err := b.DB.Exec(stmt); err != nil {
		return fmt.Errorf("erro ao criar %s %s: %w", strings.ToLower(kind), name, err)
	}
	return nil
}

// Listar returns all rows from a model table.
// ListarParams holds query parameters for listing.
type ListarParams struct {
	Pagina  int
	Limite  int
	Ordenar string
	Ordem   string // asc, desc
	Busca   string
	Filtros map[string]string
}

func (b *Banco) Listar(modelo string, params *ListarParams) ([]map[string]any, int64, error) {
	if _, ok := b.Models[modelo]; !ok {
		return nil, 0, fmt.Errorf("modelo '%s' não encontrado", modelo)
	}

	// Defaults
	if params == nil {
		params = &ListarParams{}
	}
	if params.Limite <= 0 {
		params.Limite = 100
	}
	if params.Pagina <= 0 {
		params.Pagina = 1
	}
	if params.Ordem == "" {
		params.Ordem = "DESC"
	}
	if params.Ordenar == "" {
		params.Ordenar = "id"
	}

	var where []string
	var args []any
	n := 1

	// Soft delete: exclude deleted records
	model := b.Models[modelo]
	if model.SoftDelete {
		where = append(where, q("deletado_em")+" IS NULL")
	}

	// Filters
	for _, f := range model.Fields {
		fname := strings.ToLower(f.Name)
		if val, ok := params.Filtros[fname]; ok && val != "" {
			where = append(where, fmt.Sprintf("%s = %s", q(fname), b.ph(n)))
			args = append(args, val)
			n++
		}
	}

	// Search (across all text fields)
	if params.Busca != "" {
		var searchConds []string
		for _, f := range model.Fields {
			if f.Type.SQLType() == "TEXT" {
				searchConds = append(searchConds, fmt.Sprintf("%s LIKE %s", q(strings.ToLower(f.Name)), b.ph(n)))
				args = append(args, "%"+params.Busca+"%")
				n++
			}
		}
		if len(searchConds) > 0 {
			where = append(where, "("+strings.Join(searchConds, " OR ")+")")
		}
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = " WHERE " + strings.Join(where, " AND ")
	}

	// Count total
	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", q(modelo), whereSQL)
	b.DB.QueryRow(countQuery, args...).Scan(&total)

	// Order + Pagination
	offset := (params.Pagina - 1) * params.Limite
	query := fmt.Sprintf("SELECT * FROM %s%s ORDER BY %s %s LIMIT %d OFFSET %d",
		q(modelo), whereSQL, q(params.Ordenar), params.Ordem, params.Limite, offset)

	rows, err := b.DB.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	results, err := scanRows(rows)
	return results, total, err
}

// Buscar returns a single row by ID.
func (b *Banco) Buscar(modelo string, id int64) (map[string]any, error) {
	if _, ok := b.Models[modelo]; !ok {
		return nil, fmt.Errorf("modelo '%s' não encontrado", modelo)
	}

	rows, err := b.DB.Query(fmt.Sprintf("SELECT * FROM %s WHERE %s = %s", q(modelo), q("id"), b.ph(1)), id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("registro %d não encontrado", id)
	}
	return results[0], nil
}

// Criar inserts a new row.
func (b *Banco) Criar(modelo string, dados json.RawMessage) (map[string]any, error) {
	model, ok := b.Models[modelo]
	if !ok {
		return nil, fmt.Errorf("modelo '%s' não encontrado", modelo)
	}

	var input map[string]any
	if err := json.Unmarshal(dados, &input); err != nil {
		return nil, fmt.Errorf("dados inválidos: %w", err)
	}

	if err := b.Validar(modelo, input); err != nil {
		return nil, err
	}

	var cols []string
	var phs []string
	var vals []any
	n := 1

	for _, f := range model.Fields {
		fname := strings.ToLower(f.Name)
		if v, exists := input[fname]; exists {
			cols = append(cols, q(fname))
			phs = append(phs, b.ph(n))
			vals = append(vals, v)
			n++
		}
	}

	if len(cols) == 0 {
		return nil, fmt.Errorf("nenhum campo fornecido")
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		q(modelo), strings.Join(cols, ", "), strings.Join(phs, ", "))

	if b.Driver == "postgres" || b.Driver == "postgresql" {
		query += " RETURNING " + q("id")
		var id int64
		err := b.DB.QueryRow(query, vals...).Scan(&id)
		if err != nil {
			return nil, err
		}
		return b.Buscar(modelo, id)
	}

	result, err := b.DB.Exec(query, vals...)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return b.Buscar(modelo, id)
}

// Atualizar updates a row by ID.
func (b *Banco) Atualizar(modelo string, id int64, dados json.RawMessage) (map[string]any, error) {
	model, ok := b.Models[modelo]
	if !ok {
		return nil, fmt.Errorf("modelo '%s' não encontrado", modelo)
	}

	var input map[string]any
	if err := json.Unmarshal(dados, &input); err != nil {
		return nil, fmt.Errorf("dados inválidos: %w", err)
	}

	if err := b.Validar(modelo, input); err != nil {
		return nil, err
	}

	var sets []string
	var vals []any
	n := 1

	for _, f := range model.Fields {
		fname := strings.ToLower(f.Name)
		if v, exists := input[fname]; exists {
			sets = append(sets, q(fname)+" = "+b.ph(n))
			vals = append(vals, v)
			n++
		}
	}

	if len(sets) == 0 {
		return nil, fmt.Errorf("nenhum campo para atualizar")
	}

	sets = append(sets, q("atualizado_em")+" = CURRENT_TIMESTAMP")
	vals = append(vals, id)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = %s",
		q(modelo), strings.Join(sets, ", "), q("id"), b.ph(n))
	_, err := b.DB.Exec(query, vals...)
	if err != nil {
		return nil, err
	}

	return b.Buscar(modelo, id)
}

// Deletar removes a row by ID (or soft-deletes if model has SoftDelete).
func (b *Banco) Deletar(modelo string, id int64) error {
	model, ok := b.Models[modelo]
	if !ok {
		return fmt.Errorf("modelo '%s' não encontrado", modelo)
	}
	if model.SoftDelete {
		_, err := b.DB.Exec(fmt.Sprintf("UPDATE %s SET %s = CURRENT_TIMESTAMP WHERE %s = %s",
			q(modelo), q("deletado_em"), q("id"), b.ph(1)), id)
		return err
	}
	_, err := b.DB.Exec(fmt.Sprintf("DELETE FROM %s WHERE %s = %s", q(modelo), q("id"), b.ph(1)), id)
	return err
}

// Restaurar restores a soft-deleted row by ID.
func (b *Banco) Restaurar(modelo string, id int64) (map[string]any, error) {
	model, ok := b.Models[modelo]
	if !ok {
		return nil, fmt.Errorf("modelo '%s' não encontrado", modelo)
	}
	if !model.SoftDelete {
		return nil, fmt.Errorf("modelo '%s' não suporta soft delete", modelo)
	}
	_, err := b.DB.Exec(fmt.Sprintf("UPDATE %s SET %s = NULL WHERE %s = %s",
		q(modelo), q("deletado_em"), q("id"), b.ph(1)), id)
	if err != nil {
		return nil, err
	}
	return b.Buscar(modelo, id)
}

// Fechar closes the database connection.
func (b *Banco) Fechar() {
	b.DB.Close()
}

// Contar returns the count of rows in a table.
func (b *Banco) Contar(modelo string) (int64, error) {
	var count int64
	model := b.Models[modelo]
	whereSQL := ""
	if model != nil && model.SoftDelete {
		whereSQL = " WHERE " + q("deletado_em") + " IS NULL"
	}
	err := b.DB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s%s", q(modelo), whereSQL)).Scan(&count)
	return count, err
}

// BuscarRelacionados returns related records for a model via foreign key.
func (b *Banco) BuscarRelacionados(modelo string, id int64, relacao string) ([]map[string]any, error) {
	relLower := strings.ToLower(relacao)
	modelLower := strings.ToLower(modelo)

	// Check if it's a has_many (FK on related table)
	if relModel, ok := b.Models[relLower]; ok {
		for _, f := range relModel.Fields {
			if f.Reference == modelo || strings.ToLower(f.Reference) == modelLower {
				query := fmt.Sprintf("SELECT * FROM %s WHERE %s = %s ORDER BY %s DESC",
					q(relLower), q(strings.ToLower(f.Name)), b.ph(1), q("id"))
				rows, err := b.DB.Query(query, id)
				if err != nil {
					return nil, err
				}
				defer rows.Close()
				return scanRows(rows)
			}
		}
	}

	// Check if it's a many_to_many (join table)
	name1, name2 := modelLower, relLower
	if name1 > name2 {
		name1, name2 = name2, name1
	}
	joinTable := name1 + "_" + name2

	query := fmt.Sprintf("SELECT r.* FROM %s r INNER JOIN %s j ON r.%s = j.%s WHERE j.%s = %s",
		q(relLower), q(joinTable), q("id"), q(relLower+"_id"), q(modelLower+"_id"), b.ph(1))
	rows, err := b.DB.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

// ContarPorStatus returns counts grouped by the status field for a model.
func (b *Banco) ContarPorStatus(modelo string) (map[string]int64, error) {
	model, ok := b.Models[modelo]
	if !ok {
		return nil, fmt.Errorf("modelo '%s' não encontrado", modelo)
	}

	// Find the status field
	statusField := ""
	for _, f := range model.Fields {
		if f.Type == "status" {
			statusField = strings.ToLower(f.Name)
			break
		}
	}
	if statusField == "" {
		return nil, nil // no status field
	}

	whereSQL := ""
	if model.SoftDelete {
		whereSQL = " WHERE " + q("deletado_em") + " IS NULL"
	}

	query := fmt.Sprintf("SELECT COALESCE(%s, 'sem_status'), COUNT(*) FROM %s%s GROUP BY %s",
		q(statusField), q(modelo), whereSQL, q(statusField))
	rows, err := b.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		result[status] = count
	}
	return result, nil
}

// ListarTodos returns all rows (for export), ignoring soft-deleted records.
func (b *Banco) ListarTodos(modelo string) ([]map[string]any, error) {
	model, ok := b.Models[modelo]
	if !ok {
		return nil, fmt.Errorf("modelo '%s' não encontrado", modelo)
	}

	whereSQL := ""
	if model.SoftDelete {
		whereSQL = " WHERE " + q("deletado_em") + " IS NULL"
	}

	query := fmt.Sprintf("SELECT * FROM %s%s ORDER BY %s DESC", q(modelo), whereSQL, q("id"))
	rows, err := b.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRows(rows)
}

// Validar checks field constraints.
func (b *Banco) Validar(modelo string, dados map[string]any) error {
	return b.validar(modelo, dados, false)
}

// validar checks field types and rules. In partial mode (updates) only the
// keys present in dados are checked, so absent required fields are kept.
func (b *Banco) validar(modelo string, dados map[string]any, parcial bool) error {
	model, ok := b.Models[modelo]
	if !ok {
		return fmt.Errorf("modelo '%s' não encontrado", modelo)
	}

	for _, f := range model.Fields {
		fname := strings.ToLower(f.Name)
		val, exists := dados[fname]
		if parcial && !exists {
			continue
		}

		if f.Required && (!exists || val == nil || val == "") {
			return fmt.Errorf("campo '%s' é obrigatório", f.Name)
		}

		if !exists || val == nil {
			continue
		}

		strVal := fmt.Sprintf("%v", val)

		if f.Type == ast.FieldEmail && strVal != "" {
			if !strings.Contains(strVal, "@") || !strings.Contains(strVal, ".") {
				return fmt.Errorf("email inválido no campo '%s'", f.Name)
			}
		}

		if f.Type == ast.FieldTelefone && strVal != "" {
			clean := strings.Map(func(r rune) rune {
				if r >= '0' && r <= '9' || r == '+' {
					return r
				}
				return -1
			}, strVal)
			if len(clean) < 7 {
				return fmt.Errorf("telefone inválido no campo '%s'", f.Name)
			}
		}
	}

	// Check user-defined validation rules
	for _, rule := range b.Rules {
		if rule.Action != "validar" {
			continue
		}
		fname := strings.ToLower(rule.Field)
		val, exists := dados[fname]
		if parcial && !exists {
			continue
		}

		// Check if this rule applies to a field in this model
		fieldInModel := false
		for _, f := range model.Fields {
			if strings.ToLower(f.Name) == fname {
				fieldInModel = true
				break
			}
		}
		if !fieldInModel {
			continue
		}

		strVal := ""
		if val != nil {
			strVal = fmt.Sprintf("%v", val)
		}
		numVal := float64(0)
		if val != nil {
			if n, err := strconv.ParseFloat(strVal, 64); err == nil {
				numVal = n
			}
		}

		switch rule.Operator {
		case "obrigatorio", "required":
			if !exists || val == nil || strVal == "" {
				return fmt.Errorf("campo '%s' é obrigatório", rule.Field)
			}
		case "igual", "equals", "equal":
			if rule.Value == "@" {
				// Special: must contain @
				if !strings.Contains(strVal, "@") {
					return fmt.Errorf("campo '%s' deve conter '@'", rule.Field)
				}
			} else if strVal != rule.Value {
				return fmt.Errorf("campo '%s' deve ser igual a '%s'", rule.Field, rule.Value)
			}
		case "diferente", "not_equal":
			if strVal == rule.Value {
				return fmt.Errorf("campo '%s' não pode ser '%s'", rule.Field, rule.Value)
			}
		case "maior", "greater", ">":
			ruleNum, _ := strconv.ParseFloat(rule.Value, 64)
			if numVal <= ruleNum {
				return fmt.Errorf("campo '%s' deve ser maior que %s", rule.Field, rule.Value)
			}
		case "menor", "less", "<":
			ruleNum, _ := strconv.ParseFloat(rule.Value, 64)
			if numVal >= ruleNum {
				return fmt.Errorf("campo '%s' deve ser menor que %s", rule.Field, rule.Value)
			}
		case "maior_igual", ">=":
			ruleNum, _ := strconv.ParseFloat(rule.Value, 64)
			if numVal < ruleNum {
				return fmt.Errorf("campo '%s' deve ser maior ou igual a %s", rule.Field, rule.Value)
			}
		case "menor_igual", "<=":
			ruleNum, _ := strconv.ParseFloat(rule.Value, 64)
			if numVal > ruleNum {
				return fmt.Errorf("campo '%s' deve ser menor ou igual a %s", rule.Field, rule.Value)
			}
		case "minimo", "min", "min_length":
			ruleNum, _ := strconv.ParseFloat(rule.Value, 64)
			if float64(len(strVal)) < ruleNum {
				return fmt.Errorf("campo '%s' deve ter no mínimo %s caracteres", rule.Field, rule.Value)
			}
		case "maximo", "max", "max_length":
			ruleNum, _ := strconv.ParseFloat(rule.Value, 64)
			if float64(len(strVal)) > ruleNum {
				return fmt.Errorf("campo '%s' deve ter no máximo %s caracteres", rule.Field, rule.Value)
			}
		case "contem", "contains":
			if !strings.Contains(strVal, rule.Value) {
				return fmt.Errorf("campo '%s' deve conter '%s'", rule.Field, rule.Value)
			}
		}
	}

	return nil
}

// scanRows converts sql.Rows into a slice of maps.
func scanRows(rows *sql.Rows) ([]map[string]any, error) {
	columns, _ := rows.Columns()
	var results []map[string]any

	for rows.Next() {
		values := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := map[string]any{}
		for i, col := range columns {
			// Mascarar campos sensíveis de senha em leituras de API
			if col == "senha" || col == "password" {
				continue
			}
			row[col] = values[i]
		}
		results = append(results, row)
	}

	return results, nil
}
