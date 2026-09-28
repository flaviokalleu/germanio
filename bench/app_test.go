package bench

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/flaviokalleu/germanio/bench/baseline"
	"github.com/flaviokalleu/germanio/runtime"
)

// target is one implementation of the clientes application behind the same
// four operations. Requests go straight to the handler (no network), so the
// numbers show the cost of each implementation, not of the loopback.
type target struct {
	name                      string
	h                         http.Handler
	list, one, create, pageOf string
}

var seeded = 1000

func germanio(b *testing.B) target {
	dir := b.TempDir()
	b.Setenv("GERMANIO_SQLITE", filepath.Join(dir, "app.db"))
	b.Setenv("GERMANIO_BCRYPT_RAPIDO", "1")
	app, err := runtime.Carregar("testdata/clientes.ge", "0")
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { app.Fechar() })
	return target{"germanio", app.Handler, "/_ge/api/clientes", "/_ge/api/clientes/", "/_ge/api/clientes", "/clientes"}
}

func golang(b *testing.B) target {
	h, closeDB, err := baseline.New(filepath.Join(b.TempDir(), "go.db"))
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { closeDB() })
	return target{"go", h, "/clientes", "/clientes/", "/clientes", "/pagina"}
}

func (t target) do(b *testing.B, method, path, body string, want int) {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	t.h.ServeHTTP(w, req)
	if w.Code != want {
		b.Fatalf("%s %s %s: status %d, esperado %d: %s", t.name, method, path, w.Code, want, w.Body.String())
	}
}

var serial atomic.Int64

func cliente() string {
	n := serial.Add(1)
	return fmt.Sprintf(`{"nome":"Cliente %d","email":"cliente%d@exemplo.com","telefone":"8199990%04d","cidade":"Recife"}`, n, n, n%10000)
}

func seed(b *testing.B, t target) {
	for range seeded {
		t.do(b, "POST", t.create, cliente(), http.StatusCreated)
	}
}

// BenchmarkAplicacao compares the same operation on Germanio and on Go direct
// with the same data. The ratio germanio/go is the cost of the abstraction.
func BenchmarkAplicacao(b *testing.B) {
	for _, make := range []func(*testing.B) target{germanio, golang} {
		t := make(b)
		seed(b, t)
		b.Run("listar/"+t.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				t.do(b, "GET", t.list+"?page=3", "", http.StatusOK)
			}
		})
		b.Run("obter/"+t.name, func(b *testing.B) {
			b.ReportAllocs()
			i := 0
			for b.Loop() {
				i++
				t.do(b, "GET", fmt.Sprint(t.one, i%seeded+1), "", http.StatusOK)
			}
		})
		b.Run("criar/"+t.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				t.do(b, "POST", t.create, cliente(), http.StatusCreated)
			}
		})
		b.Run("pagina/"+t.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				t.do(b, "GET", t.pageOf, "", http.StatusOK)
			}
		})
	}
}
