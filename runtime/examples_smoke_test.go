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
		"../examples/blog.ge", "../examples/crm.ge", "../examples/ecommerce.ge",
		"../examples/cadastro-simples.ge", "../examples/english-mode.ge", "../examples/prompt-saas.ge",
		"../examples/site-germanio/inicio.ge", "../examples/evoticket/inicio.ge", "../examples/whaticket/inicio.ge", "../demo/plano/inicio.ge", "../demo/organizado/inicio.ge",
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
