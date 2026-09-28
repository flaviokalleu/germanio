package runtime

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Files of a record (GEP 0014, em teste).

func sendRaw(t *testing.T, c *client, method, path string, body []byte) (int, http.Header, string) {
	t.Helper()
	req, _ := http.NewRequest(method, c.base+path, bytes.NewReader(body))
	if c.csrf != "" {
		req.Header.Set("X-CSRF-Token", c.csrf)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header, string(b)
}

func storedFiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	filepath.WalkDir(filepath.Join(root, "chamado"), func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			out = append(out, p)
		}
		return nil
	})
	return out
}

var png = append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 64)...)

func TestArquivos(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GERMANIO_ARQUIVOS", root)
	t.Setenv("GERMANIO_ARQUIVO_MAX_MB", "1")
	_, c := loadApp(t, "testdata/arquivos/app.ge")
	ana := signIn(t, c.base, "Ana", "ana@x.com")
	bia := signIn(t, c.base, "Bia", "bia@x.com")
	ana.expect("POST", "/_ge/api/chamados", map[string]any{"titulo": "Nota fiscal"}, 201)

	// JSON cannot set a file (the place on disk never comes from the client)
	refused := ana.expect("PUT", "/_ge/api/chamados/1", map[string]any{"anexo": `{"nome":"x","chave":"../../../etc/passwd"}`}, 400)
	if msg, _ := refused["message"].(string); !strings.Contains(msg, "é um arquivo") {
		t.Fatalf("o erro explica como enviar o arquivo: %v", refused)
	}
	if got := ana.expect("GET", "/_ge/api/chamados/1", nil, 200); got["anexo"] != nil {
		t.Fatalf("um campo de arquivo foi escrito por JSON: %v", got["anexo"])
	}

	// upload, then the record shows name, size and type — never the place
	code, _, raw := sendRaw(t, ana, "PUT", "/_ge/api/chamados/1/anexo?nome=../../nota.pdf", []byte("%PDF-1.4 conteúdo"))
	if code != 200 {
		t.Fatalf("enviar anexo: %d %s", code, raw)
	}
	got := ana.expect("GET", "/_ge/api/chamados/1", nil, 200)
	meta, _ := got["anexo"].(map[string]any)
	if meta["nome"] != "nota.pdf" || meta["tamanho"] != float64(len("%PDF-1.4 conteúdo")) || meta["chave"] != nil {
		t.Fatalf("o registro mostra nome e tamanho, sem caminho nem chave: %v", got["anexo"])
	}
	// download: attachment, nosniff, the same bytes
	code, h, body := sendRaw(t, bia, "GET", "/_ge/api/chamados/1/anexo", nil)
	if code != 200 || body != "%PDF-1.4 conteúdo" || !strings.HasPrefix(h.Get("Content-Disposition"), "attachment") || h.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("baixar anexo: %d %v %q", code, h, body)
	}
	// only the author edits: Bia cannot replace or remove it
	if code, _, _ := sendRaw(t, bia, "PUT", "/_ge/api/chamados/1/anexo", []byte("troca")); code != 403 {
		t.Fatalf("quem não edita não envia arquivo: %d", code)
	}
	if code, _, _ := sendRaw(t, bia, "DELETE", "/_ge/api/chamados/1/anexo", nil); code != 403 {
		t.Fatalf("quem não edita não remove arquivo: %d", code)
	}

	// an uploaded page is never served as the application's page
	sendRaw(t, ana, "PUT", "/_ge/api/chamados/1/anexo?nome=x.html", []byte("<html><script>alert(1)</script></html>"))
	_, h, _ = sendRaw(t, bia, "GET", "/_ge/api/chamados/1/anexo", nil)
	if h.Get("Content-Type") != "application/octet-stream" || !strings.HasPrefix(h.Get("Content-Disposition"), "attachment") || !strings.Contains(h.Get("Content-Security-Policy"), "sandbox") {
		t.Fatalf("HTML enviado é baixado, nunca exibido: %v", h)
	}
	if n := len(storedFiles(t, root)); n != 1 {
		t.Fatalf("trocar o arquivo apaga o antigo: %d arquivos", n)
	}

	// an image field accepts only images, shown inline
	if code, _, _ := sendRaw(t, ana, "PUT", "/_ge/api/chamados/1/foto", []byte("<svg onload=alert(1)>")); code != 400 {
		t.Fatalf("foto que não é imagem: %d", code)
	}
	if code, _, raw := sendRaw(t, ana, "PUT", "/_ge/api/chamados/1/foto?nome=f.png", png); code != 200 {
		t.Fatalf("foto: %d %s", code, raw)
	}
	if _, h, _ := sendRaw(t, bia, "GET", "/_ge/api/chamados/1/foto", nil); h.Get("Content-Type") != "image/png" || !strings.HasPrefix(h.Get("Content-Disposition"), "inline") {
		t.Fatalf("imagem é exibida: %v", h)
	}

	// the limit, streamed: over 1 MB is refused and nothing stays
	big := bytes.Repeat([]byte("x"), 1<<20+10)
	if code, _, _ := sendRaw(t, ana, "PUT", "/_ge/api/chamados/1/anexo", big); code != 413 {
		t.Fatalf("arquivo acima do limite: %d", code)
	}
	if left, _ := filepath.Glob(filepath.Join(root, ".recebendo", "*")); len(left) != 0 {
		t.Fatalf("sobrou arquivo temporário: %v", left)
	}

	// an undone change keeps the old file and discards the new one: the
	// edit rule refuses changes to a record called "trava"
	ana.expect("POST", "/_ge/api/chamados", map[string]any{"titulo": "trava"}, 201)
	before := len(storedFiles(t, root))
	if code, _, _ := sendRaw(t, ana, "PUT", "/_ge/api/chamados/2/anexo?nome=novo.pdf", []byte("novo")); code != 400 {
		t.Fatalf("a regra de edição recusa: %d", code)
	}
	if n := len(storedFiles(t, root)); n != before {
		t.Fatalf("uma mudança desfeita deixou um arquivo: %d → %d", before, n)
	}
	if left, _ := filepath.Glob(filepath.Join(root, ".recebendo", "*")); len(left) != 0 {
		t.Fatalf("sobrou arquivo temporário: %v", left)
	}
	if got := ana.expect("GET", "/_ge/api/chamados/2", nil, 200); got["anexo"] != nil {
		t.Fatalf("o registro guardou um arquivo recusado: %v", got["anexo"])
	}

	// a confidential record's file is hidden from others
	ana.expect("POST", "/_ge/api/chamados", map[string]any{"titulo": "Salários", "confidencial": true}, 201)
	sendRaw(t, ana, "PUT", "/_ge/api/chamados/3/anexo?nome=folha.pdf", []byte("folha"))
	if code, _, _ := sendRaw(t, bia, "GET", "/_ge/api/chamados/3/anexo", nil); code != 404 {
		t.Fatalf("anexo de chamado confidencial: %d", code)
	}

	// deleting the record deletes its files
	ana.expect("DELETE", "/_ge/api/chamados/1", nil, 204)
	for _, f := range storedFiles(t, root) {
		if b, _ := os.ReadFile(f); strings.Contains(string(b), "script") || bytes.Equal(b, png) {
			t.Fatalf("o arquivo de um registro excluído ficou no disco: %s", f)
		}
	}
}

// The record's page shows its files and, to whoever may edit, a form that
// sends one (multipart, CSRF first, streamed to the record's address).
func TestArquivosNaPagina(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GERMANIO_ARQUIVOS", root)
	_, c := loadApp(t, "testdata/arquivos/app.ge")
	ana := signIn(t, c.base, "Ana", "ana@x.com")
	bia := signIn(t, c.base, "Bia", "bia@x.com")
	ana.expect("POST", "/_ge/api/chamados", map[string]any{"titulo": "Contrato"}, 201)
	_, _, page := sendRaw(t, ana, "GET", "/chamados/1", nil)
	if !strings.Contains(page, `enctype="multipart/form-data"`) || !strings.Contains(page, `action="/chamados/1/arquivo/anexo"`) {
		t.Fatalf("a página de quem edita tem o formulário de envio:\n%s", page)
	}
	send := func(who *client, csrf string) (int, string) {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		mw.WriteField("_csrf", csrf)
		fw, _ := mw.CreateFormFile("arquivo", "contrato.pdf")
		fw.Write([]byte("%PDF contrato"))
		mw.Close()
		req, _ := http.NewRequest("POST", c.base+"/chamados/1/arquivo/anexo", &buf)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		resp, err := who.http.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode, resp.Header.Get("Location")
	}
	if code, _ := send(ana, "errado"); code != 403 {
		t.Fatalf("sem o token CSRF certo: %d", code)
	}
	if code, loc := send(ana, ana.csrf); code != 303 || !strings.Contains(loc, "ok=") {
		t.Fatalf("enviar pela página: %d %s", code, loc)
	}
	if code, loc := send(bia, bia.csrf); code != 303 || !strings.Contains(loc, "erro=") {
		t.Fatalf("quem não edita recebe o erro: %d %s", code, loc)
	}
	_, _, page = sendRaw(t, bia, "GET", "/chamados/1", nil)
	if !strings.Contains(page, `href="/_ge/api/chamados/1/anexo"`) || !strings.Contains(page, "contrato.pdf") || strings.Contains(page, `action="/chamados/1/arquivo/anexo"`) {
		t.Fatalf("quem vê tem o link; quem não edita não tem o formulário:\n%s", page)
	}
}
