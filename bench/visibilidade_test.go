package bench

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// BenchmarkVisibilidade reproduces the case the performance audit measured
// (docs/research/performance/AUDITORIA.md): records whose visibility depends
// on their parent (issues of private projects), a person who is a member of
// one project out of 50, and a list of 20 issues out of 50 000.
func BenchmarkVisibilidade(b *testing.B) {
	const projects, perProject = 50, 1000
	dir := b.TempDir()
	b.Setenv("GERMANIO_SQLITE", filepath.Join(dir, "v.db"))
	b.Setenv("GERMANIO_BCRYPT_RAPIDO", "1")
	app := carregar(b, "testdata/visibilidade.ge")
	defer app.Fechar()
	h := app.Handler

	// a person, through the real sign-up (session cookie)
	req := httptest.NewRequest("POST", "/cadastro", strings.NewReader(`{"nome":"Ana","email":"ana@x.com","senha":"senha-forte-1"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		b.Fatalf("cadastro: %d %s", w.Code, w.Body.String())
	}
	var ana map[string]any
	json.Unmarshal(w.Body.Bytes(), &ana)
	cookies := w.Result().Cookies()

	// the data, straight into the database
	tx, err := app.DB.DB.Begin()
	if err != nil {
		b.Fatal(err)
	}
	for p := 1; p <= projects; p++ {
		if _, err := tx.Exec(`INSERT INTO projeto (id, nome, visibilidade) VALUES (?, ?, 'private')`, p, fmt.Sprint("projeto ", p)); err != nil {
			b.Fatal(err)
		}
		for i := 0; i < perProject; i++ {
			if _, err := tx.Exec(`INSERT INTO issue (titulo, projeto_id) VALUES (?, ?)`, fmt.Sprint("issue ", i), p); err != nil {
				b.Fatal(err)
			}
		}
	}
	if _, err := tx.Exec(`INSERT INTO membro (recurso, recurso_id, pessoa_id, papel) VALUES ('projeto', 1, ?, 'guest')`, ana["id"]); err != nil {
		b.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		b.Fatal(err)
	}

	list := func() []any {
		r := httptest.NewRequest("GET", "/_ge/api/issues?page=1", nil)
		for _, c := range cookies {
			r.AddCookie(c)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			b.Fatalf("lista: %d %s", w.Code, w.Body.String())
		}
		var items []any
		json.Unmarshal(w.Body.Bytes(), &items)
		return items
	}
	if items := list(); len(items) != 20 {
		b.Fatalf("esperadas 20 issues visíveis, vieram %d", len(items))
	}
	b.ReportAllocs()
	for b.Loop() {
		list()
	}
}
