package bench

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// BenchmarkEscritaConcorrente: many people creating records at the same time
// on SQLite (one writer at a time). Every request must succeed: waiting is
// acceptable, an error is not.
func BenchmarkEscritaConcorrente(b *testing.B) {
	b.Setenv("GERMANIO_SQLITE", filepath.Join(b.TempDir(), "w.db"))
	app := carregar(b, "testdata/clientes.ge")
	defer app.Fechar()
	var failed atomic.Int64
	b.SetParallelism(2) // 2 × GOMAXPROCS writers
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			body := `{"nome":"Cliente","email":"c` + strings.ReplaceAll(randomID(), "-", "") + `@x.com"}`
			r := httptest.NewRequest("POST", "/_ge/api/clientes", strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			app.Handler.ServeHTTP(w, r)
			if w.Code != http.StatusCreated {
				failed.Add(1)
			}
		}
	})
	if n := failed.Load(); n > 0 {
		b.Errorf("%d escritas falharam", n)
	}
}

var counter atomic.Int64

func randomID() string {
	return strings.TrimSpace(strings.Repeat(" ", 0)) + itoa(counter.Add(1))
}

func itoa(n int64) string {
	const digits = "0123456789"
	if n == 0 {
		return "0"
	}
	var s []byte
	for n > 0 {
		s = append([]byte{digits[n%10]}, s...)
		n /= 10
	}
	return string(s)
}
