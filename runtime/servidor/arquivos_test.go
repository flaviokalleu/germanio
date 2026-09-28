package servidor

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// Served folders (/assets/, /uploads/) hand out files, never listings: a
// directory listing of /uploads/ would let anyone enumerate what other people
// sent. Hidden files (.env, .git) are never served either.
func TestPastaServidaNaoListaNemEntregaOcultos(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "foto.png"), []byte("png"), 0o644)
	os.WriteFile(filepath.Join(dir, ".env"), []byte("SEGREDO=1"), 0o644)
	os.Mkdir(filepath.Join(dir, "pessoa1"), 0o755)
	os.WriteFile(filepath.Join(dir, "pessoa1", "doc.pdf"), []byte("pdf"), 0o644)
	h := http.StripPrefix("/uploads/", fileServer(dir))
	for path, want := range map[string]int{
		"/uploads/":                http.StatusNotFound,
		"/uploads/pessoa1/":        http.StatusNotFound,
		"/uploads/pessoa1":         http.StatusNotFound,
		"/uploads/.env":            http.StatusNotFound,
		"/uploads/foto.png":        http.StatusOK,
		"/uploads/pessoa1/doc.pdf": http.StatusOK,
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != want {
			t.Errorf("%s: status %d, esperado %d (%q)", path, w.Code, want, w.Body.String())
		}
	}
}
