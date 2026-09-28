package runtime

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

// TestExamplesLoadAndServe guards backward compatibility: every example app
// must still load (parse, migrate, run startup scripts) and serve its UI.
func TestExamplesLoadAndServe(t *testing.T) {
	examples := []string{
		"../examples/legacy/blog.ge", "../examples/legacy/crm.ge", "../examples/legacy/ecommerce.ge",
		"../examples/legacy/cadastro-simples.ge", "../examples/legacy/english-mode.ge", "../examples/legacy/prompt-saas.ge",
		"../examples/legacy/whaticket.ge", "../examples/legacy/evoticket/inicio.ge", "../examples/legacy/whaticket/inicio.ge",
		"../examples/site-germanio/inicio.ge", "../demo/plano/inicio.ge", "../demo/organizado/inicio.ge",
	}
	for _, ex := range examples {
		t.Run(filepath.Base(filepath.Dir(ex))+"/"+filepath.Base(ex), func(t *testing.T) {
			t.Setenv("GERMANIO_SQLITE", filepath.Join(t.TempDir(), "x.db"))
			app, err := Carregar(ex, "0")
			if err != nil {
				t.Fatalf("Carregar: %v", err)
			}
			defer app.Fechar()
			for _, path := range []string{"/", "/health"} {
				rec := httptest.NewRecorder()
				app.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
				if rec.Code != http.StatusOK {
					t.Fatalf("GET %s = %d", path, rec.Code)
				}
			}
			for _, l := range app.Interpreter.GetLogs(false) {
				if len(l) > 4 && l[:4] == "ERRO" {
					t.Errorf("erro em script de inicialização: %s", l)
				}
			}
		})
	}
}

// TestIntentExamplesLoadAndServe does the same for the examples written in the
// intent syntax. Their pages may redirect (to the first page, or to sign in
// when the data needs a login), so / is followed until a page answers.
func TestIntentExamplesLoadAndServe(t *testing.T) {
	examples := []string{
		"todo", "crud", "authentication", "rest-api", "blog", "crm", "ecommerce",
	}
	for _, ex := range examples {
		t.Run(ex, func(t *testing.T) {
			t.Setenv("GERMANIO_SQLITE", filepath.Join(t.TempDir(), "x.db"))
			app, err := Carregar(filepath.Join("..", "examples", ex, "app.ge"), "0")
			if err != nil {
				t.Fatalf("Carregar: %v", err)
			}
			defer app.Fechar()
			for _, start := range []string{"/", "/health"} {
				path := start
				for hops := 0; ; hops++ {
					rec := httptest.NewRecorder()
					app.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
					if rec.Code == http.StatusSeeOther || rec.Code == http.StatusFound {
						if hops == 3 {
							t.Fatalf("GET %s: redirecionamentos demais", start)
						}
						path = rec.Header().Get("Location")
						continue
					}
					if rec.Code != http.StatusOK {
						t.Fatalf("GET %s (via %s) = %d", path, start, rec.Code)
					}
					break
				}
			}
		})
	}
}
