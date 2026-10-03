package runtime

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"testing"
)

// raw returns the status, the Content-Disposition and the bytes of path.
func (c *client) raw(path string) (int, string, []byte) {
	c.t.Helper()
	resp, err := c.http.Get(c.base + path)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header.Get("Content-Disposition"), b
}

// Downloading the code of a revision as one file (a book written in Git,
// no GitLab): who may download code gets the zip or the tar.gz with the
// files under one folder; anyone else gets "not found".
func TestBaixarCodigoComoArquivo(t *testing.T) {
	_, c := loadApp(t, "testdata/baixar_codigo/app.ge")
	ana := signIn(t, c.base, "Ana", "ana@x.com")
	bia := signIn(t, c.base, "Bia", "bia@x.com")
	ana.expect("POST", "/_ge/api/livros", map[string]any{"nome": "contos"}, 201)
	ana.expect("PUT", "/_ge/api/livros/1/repositorio/arquivos/cap1.md", map[string]any{"conteudo": "# Capítulo 1", "branch": "main"}, 200)
	ana.expect("PUT", "/_ge/api/livros/1/repositorio/arquivos/notas/fim.md", map[string]any{"conteudo": "fim", "branch": "main"}, 200)

	code, disp, body := ana.raw("/_ge/api/livros/1/repositorio/baixar.zip?sha=main")
	if code != 200 || disp != `attachment; filename="contos-main.zip"` {
		t.Fatalf("zip: %d %q %s", code, disp, body)
	}
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, f := range zr.File {
		if !f.FileInfo().IsDir() {
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			rc.Close()
			files[f.Name] = string(b)
		}
	}
	if len(files) != 2 || files["contos-main/cap1.md"] != "# Capítulo 1" || files["contos-main/notas/fim.md"] != "fim" {
		t.Fatalf("conteúdo do zip: %v", files)
	}

	code, _, body = ana.raw("/_ge/api/livros/1/repositorio/baixar")
	if code != 200 {
		t.Fatalf("tar.gz: %d %s", code, body)
	}
	gz, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	got := map[string]string{}
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if h.Typeflag == tar.TypeReg {
			b, _ := io.ReadAll(tr)
			got[h.Name] = string(b)
		}
	}
	if got["contos-main/cap1.md"] != "# Capítulo 1" || got["contos-main/notas/fim.md"] != "fim" {
		t.Fatalf("conteúdo do tar.gz: %v", got)
	}

	if code, _, _ := bia.raw("/_ge/api/livros/1/repositorio/baixar.zip"); code != 404 {
		t.Fatalf("quem não pode ver baixou: %d", code)
	}
	if code, _, _ := ana.raw("/_ge/api/livros/1/repositorio/baixar.zip?sha=nada"); code != 404 {
		t.Fatalf("revisão inexistente: %d", code)
	}
	if code, _, _ := ana.raw("/_ge/api/livros/1/repositorio/baixar.rar"); code != 404 {
		t.Fatalf("formato desconhecido: %d", code)
	}
}
