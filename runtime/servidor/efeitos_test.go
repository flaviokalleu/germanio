package servidor

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/runtime/banco"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// The dispatch of external effects around the request transaction (G86).

func effectServer(t *testing.T) *Servidor {
	t.Helper()
	t.Setenv("GERMANIO_SQLITE", filepath.Join(t.TempDir(), "t.db"))
	db, err := banco.Abrir(&ast.DatabaseConfig{Driver: "sqlite"}, "t", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Fechar() })
	for _, q := range []string{
		`CREATE TABLE pai (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE filho (id INTEGER PRIMARY KEY, pai_id INTEGER REFERENCES pai(id) DEFERRABLE INITIALLY DEFERRED)`,
	} {
		if _, err := db.DB.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	return &Servidor{DB: db}
}

// the commit itself fails (a deferred foreign key is checked only then): the
// effect recorded during the change never runs.
func TestEfeitoNaoRodaQuandoOCommitFalha(t *testing.T) {
	s := effectServer(t)
	ran := 0
	w := httptest.NewRecorder()
	s.transactional(w, httptest.NewRequest("POST", "/", nil), func(w http.ResponseWriter, r *http.Request) {
		ctx := newContext(w, r)
		if _, err := ctx.DB.Executar(`INSERT INTO filho (pai_id) VALUES (42)`); err != nil {
			t.Fatal(err) // deferred: accepted until the commit
		}
		ctx.Effects.Add(interp.Effect{Kind: "teste", Run: func() error { ran++; return nil }})
		w.WriteHeader(http.StatusCreated)
	})
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("um commit que falha responde 500, veio %d", w.Code)
	}
	if ran != 0 {
		t.Fatal("o efeito rodou apesar de o commit falhar")
	}
	var n int
	s.DB.DB.QueryRow(`SELECT COUNT(*) FROM filho`).Scan(&n)
	if n != 0 {
		t.Fatalf("filhos: %d", n)
	}
}

// effects run after the commit, in order, outside the transaction (the lock
// is free: an effect can write), and a failing one neither undoes the change
// nor stops the others.
func TestEfeitosEmOrdemForaDaTransacao(t *testing.T) {
	s := effectServer(t)
	var order []string
	w := httptest.NewRecorder()
	s.transactional(w, httptest.NewRequest("POST", "/", nil), func(w http.ResponseWriter, r *http.Request) {
		ctx := newContext(w, r)
		ctx.DB.Executar(`INSERT INTO pai (id) VALUES (1)`)
		ctx.Effects.Add(interp.Effect{Kind: "um", Run: func() error {
			order = append(order, "um")
			// outside the transaction: the change is visible and the lock is free
			var n int
			s.DB.DB.QueryRow(`SELECT COUNT(*) FROM pai`).Scan(&n)
			if n != 1 {
				t.Errorf("o efeito rodou antes do commit (pais=%d)", n)
			}
			_, err := s.DB.DB.Exec(`INSERT INTO pai (id) VALUES (2)`)
			return err
		}})
		ctx.Effects.Add(interp.Effect{Kind: "dois", Run: func() error { order = append(order, "dois"); return errors.New("fora do ar") }})
		ctx.Effects.Add(interp.Effect{Kind: "tres", Run: func() error { order = append(order, "tres"); panic("quebrou") }})
		ctx.Effects.Add(interp.Effect{Kind: "quatro", Run: func() error { order = append(order, "quatro"); return nil }})
		w.WriteHeader(http.StatusCreated)
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("status %d", w.Code)
	}
	if got := len(order); got != 4 || order[0] != "um" || order[3] != "quatro" {
		t.Fatalf("ordem dos efeitos: %v", order)
	}
	var n int
	s.DB.DB.QueryRow(`SELECT COUNT(*) FROM pai`).Scan(&n)
	if n != 2 {
		t.Fatalf("a mudança e a escrita do efeito deviam ficar: pais=%d", n)
	}
}

func TestGlobMatch(t *testing.T) {
	cases := []struct {
		p, s string
		ok   bool
	}{
		{"main", "main", true}, {"main", "mains", false}, {"release/*", "release/1.0", true},
		{"release/*", "release", false}, {"*-stable", "2-stable", true}, {"*", "qualquer/coisa", true},
		{"a*b*c", "a-x-b-y-c", true}, {"a*b*c", "a-x-c", false},
	}
	for _, c := range cases {
		if globMatch(c.p, c.s) != c.ok {
			t.Errorf("globMatch(%q, %q) != %v", c.p, c.s, c.ok)
		}
	}
}

// Backpressure (GEP 0020): a viewer that never reads does not block the
// announcement nor grow memory — its one-slot signal is just full.
func TestAvisoNaoBloqueiaComClienteParado(t *testing.T) {
	h := &liveHub{a: &intentAPI{app: &ast.App{Entities: map[string]*ast.Entity{}}}, watchers: map[*watcher]bool{}}
	stalled := &watcher{depends: map[string]bool{"x": true}, signal: make(chan struct{}, 1)}
	active := &watcher{depends: map[string]bool{"x": true}, signal: make(chan struct{}, 1)}
	h.add(stalled)
	h.add(active)
	done := make(chan struct{})
	go func() {
		for i := 0; i < 10000; i++ {
			h.publish(change{model: "x"})
			select { // the active viewer keeps reading
			case <-active.signal:
			default:
			}
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("um cliente parado bloqueou os avisos")
	}
	if len(stalled.signal) != 1 {
		t.Fatalf("o cliente parado acumulou avisos: %d", len(stalled.signal))
	}
}
