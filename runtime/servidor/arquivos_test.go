package servidor

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

// An SVG sent by someone can carry a script: it is served as a download,
// inside a sandbox, never sniffed.
func TestUploadSVGNaoExecuta(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "x.svg"), []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), 0o644)
	w := httptest.NewRecorder()
	http.StripPrefix("/uploads/", uploadsServer(dir)).ServeHTTP(w, httptest.NewRequest("GET", "/uploads/x.svg", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	if csp := w.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "sandbox") {
		t.Errorf("sem sandbox: %q", csp)
	}
	if w.Header().Get("Content-Disposition") != "attachment" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("cabeçalhos: %v", w.Header())
	}
}
