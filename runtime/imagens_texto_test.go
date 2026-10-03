package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Images in formatted text (UP-01).
func TestImagensNoTexto(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GERMANIO_ARQUIVOS", root)
	_, c := loadApp(t, "testdata/imagens_texto/app.ge")
	ana := signIn(t, c.base, "Ana", "ana@x.com")
	bia := signIn(t, c.base, "Bia", "bia@x.com")
	ana.expect("POST", "/_ge/api/issues", map[string]any{"titulo": "Tela quebrada"}, 201)

	// Bia can comment on it, so she may add an image to its texts
	code, _, raw := sendRaw(t, bia, "POST", "/_ge/api/issues/1/imagens?nome=tela.png", png)
	if code != 201 {
		t.Fatalf("enviar imagem: %d %s", code, raw)
	}
	var out map[string]string
	json.Unmarshal([]byte(raw), &out)
	if !strings.HasPrefix(out["markdown"], "![tela.png](/_ge/api/issues/1/imagens/") {
		t.Fatalf("markdown: %v", out)
	}
	code, h, body := sendRaw(t, ana, "GET", out["url"], nil)
	if code != 200 || h.Get("Content-Type") != "image/png" || body != string(png) || h.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("baixar a imagem: %d %v", code, h)
	}
	// only images: an SVG (a page in disguise) is refused
	if code, _, _ := sendRaw(t, ana, "POST", "/_ge/api/issues/1/imagens?nome=x.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"/>`)); code != 400 {
		t.Fatalf("SVG aceito como imagem: %d", code)
	}
	// a visitor is not told anything
	_, anon := c.fresh(t)
	if code, _, _ := sendRaw(t, anon, "GET", out["url"], nil); code < 400 {
		t.Fatalf("visitante baixou a imagem: %d", code)
	}
	// a confidential issue: its image is hidden from who cannot see it
	ana.expect("POST", "/_ge/api/issues", map[string]any{"titulo": "Segredo", "confidencial": true}, 201)
	_, _, raw = sendRaw(t, ana, "POST", "/_ge/api/issues/2/imagens?nome=s.png", png)
	json.Unmarshal([]byte(raw), &out)
	if code, _, _ := sendRaw(t, bia, "GET", out["url"], nil); code != 404 {
		t.Fatalf("imagem de issue confidencial para quem não a vê: %d", code)
	}
	// the page says where images go only to whoever may send one
	_, _, page := ana.do("GET", "/issues/1", nil)
	if !strings.Contains(page, `name="ge-imagens" content="/_ge/api/issues/1/imagens"`) || !strings.Contains(page, `data-formatado`) {
		t.Fatalf("a página não aceita imagens no texto:\n%s", page)
	}
	if _, _, page := anon.do("GET", "/issues/1", nil); strings.Contains(page, "ge-imagens") {
		t.Fatal("um visitante recebeu o destino das imagens")
	}
	// deleting the record deletes its images
	files, _ := filepath.Glob(filepath.Join(root, "imagens", "*"))
	before := len(files)
	ana.expect("DELETE", "/_ge/api/issues/2", nil, 204)
	files, _ = filepath.Glob(filepath.Join(root, "imagens", "*"))
	if len(files) != before-1 {
		t.Fatalf("a imagem de uma issue excluída ficou no disco: %d → %d", before, len(files))
	}
	if _, err := os.Stat(filepath.Join(root, ".recebendo")); err == nil {
		if left, _ := filepath.Glob(filepath.Join(root, ".recebendo", "*")); len(left) > 0 {
			t.Fatalf("sobrou arquivo temporário: %v", left)
		}
	}
}
