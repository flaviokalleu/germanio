// Package baseline is the same clientes application written directly in Go
// (net/http + database/sql + the same pure-Go SQLite driver and settings the
// Germanio runtime uses). It exists only to measure what the Germanio
// abstraction costs; it is not a product and has no features beyond what the
// benchmarks compare: a paginated list with a total, reading one record,
// creating one with the same validations, and an HTML page with a table.
package baseline

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type cliente struct {
	ID        int64   `json:"id"`
	Nome      string  `json:"nome"`
	Email     string  `json:"email"`
	Telefone  *string `json:"telefone"`
	Cidade    *string `json:"cidade"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

const columns = "id, nome, email, telefone, cidade, created_at, updated_at"

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

var page = template.Must(template.New("p").Parse(`<!doctype html><html lang="pt-BR"><head><meta charset="utf-8"><title>Clientes</title></head><body><main><h1>Clientes</h1><table><thead><tr><th>Nome</th><th>Email</th><th>Telefone</th><th>Cidade</th></tr></thead><tbody>{{range .}}<tr><td>{{.Nome}}</td><td>{{.Email}}</td><td>{{with .Telefone}}{{.}}{{end}}</td><td>{{with .Cidade}}{{.}}{{end}}</td></tr>{{end}}</tbody></table></main></body></html>`))

// New opens (and creates) the database at path and returns the handler and a
// function that closes the database.
func New(path string) (http.Handler, func() error, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_txlock=immediate")
	if err != nil {
		return nil, nil, err
	}
	// the same pool settings as runtime/banco, so the comparison measures the
	// abstraction and not a different configuration
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS clientes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		nome TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		telefone TEXT,
		cidade TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL)`); err != nil {
		db.Close()
		return nil, nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /clientes", func(w http.ResponseWriter, r *http.Request) { list(db, w, r) })
	mux.HandleFunc("GET /clientes/{id}", func(w http.ResponseWriter, r *http.Request) { get(db, w, r) })
	mux.HandleFunc("POST /clientes", func(w http.ResponseWriter, r *http.Request) { create(db, w, r) })
	mux.HandleFunc("GET /pagina", func(w http.ResponseWriter, r *http.Request) { html(db, w, r) })
	return mux, db.Close, nil
}

func scan(rows interface{ Scan(...any) error }) (cliente, error) {
	var c cliente
	err := rows.Scan(&c.ID, &c.Nome, &c.Email, &c.Telefone, &c.Cidade, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func pageOf(db *sql.DB, r *http.Request) ([]cliente, int, error) {
	pg, _ := strconv.Atoi(r.URL.Query().Get("page"))
	per, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if pg < 1 {
		pg = 1
	}
	if per < 1 || per > 100 {
		per = 20
	}
	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM clientes").Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := db.Query("SELECT "+columns+" FROM clientes ORDER BY id DESC LIMIT ? OFFSET ?", per, (pg-1)*per)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]cliente, 0, per)
	for rows.Next() {
		c, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	return out, total, rows.Err()
}

func list(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	items, total, err := pageOf(db, r)
	if err != nil {
		http.Error(w, "erro", http.StatusInternalServerError)
		return
	}
	w.Header().Set("X-Total", strconv.Itoa(total))
	writeJSON(w, http.StatusOK, items)
}

func get(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	c, err := scan(db.QueryRow("SELECT "+columns+" FROM clientes WHERE id = ?", r.PathValue("id")))
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "não encontrado"})
		return
	} else if err != nil {
		http.Error(w, "erro", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func create(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	var in struct {
		Nome, Email      string
		Telefone, Cidade *string
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "JSON inválido"})
		return
	}
	in.Nome, in.Email = strings.TrimSpace(in.Nome), strings.ToLower(strings.TrimSpace(in.Email))
	problems := map[string][]string{}
	if in.Nome == "" {
		problems["nome"] = append(problems["nome"], "é obrigatório")
	}
	if in.Email == "" {
		problems["email"] = append(problems["email"], "é obrigatório")
	} else if !emailRe.MatchString(in.Email) {
		problems["email"] = append(problems["email"], "é inválido")
	} else {
		var n int
		if err := db.QueryRow("SELECT COUNT(*) FROM clientes WHERE email = ?", in.Email).Scan(&n); err != nil {
			http.Error(w, "erro", http.StatusInternalServerError)
			return
		}
		if n > 0 {
			problems["email"] = append(problems["email"], "já está em uso")
		}
	}
	if len(problems) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"message": problems})
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Exec("INSERT INTO clientes (nome, email, telefone, cidade, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		in.Nome, in.Email, in.Telefone, in.Cidade, now, now)
	if err != nil {
		http.Error(w, "erro", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, cliente{ID: id, Nome: in.Nome, Email: in.Email, Telefone: in.Telefone, Cidade: in.Cidade, CreatedAt: now, UpdatedAt: now})
}

func html(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	items, _, err := pageOf(db, r)
	if err != nil {
		http.Error(w, "erro", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	page.Execute(w, items)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
