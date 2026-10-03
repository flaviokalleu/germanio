package e2e

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"testing"
)

// baixar fetches a URL with the person's token: status and bytes.
func baixar(t *testing.T, who *api, path string) (int, http.Header, []byte) {
	t.Helper()
	req, _ := http.NewRequest("GET", who.base+path, nil)
	if who.token != "" {
		req.Header.Set("Authorization", "Bearer "+who.token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header, b
}

// RP-10: baixar o código de uma revisão como zip ou tar.gz, para quem pode
// baixar código; quem não vê o projeto recebe 404.
func TestBaixarCodigo(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	eve := signup(t, base, "eve")
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Lib", "path": "lib", "initialize_with_readme": true}, 201)
	pid := id(p)
	ada.must("PUT", "/api/v4/projects/"+pid+"/repository/files/src/main.go", map[string]any{"branch": "main", "content": "package main", "commit_message": "código"}, 200)

	code, h, body := baixar(t, ada, "/api/v4/projects/"+pid+"/repository/archive.zip?sha=main")
	if code != 200 || h.Get("Content-Type") != "application/zip" || !strings.Contains(h.Get("Content-Disposition"), `filename="lib-main.zip"`) {
		t.Fatalf("zip: %d %v %s", code, h, body)
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
	if files["lib-main/src/main.go"] != "package main" || !strings.HasPrefix(files["lib-main/README.md"], "# Lib") {
		t.Fatalf("conteúdo do zip: %v", files)
	}

	code, h, body = baixar(t, ada, "/api/v4/projects/"+pid+"/repository/archive.tar.gz")
	if code != 200 || h.Get("Content-Type") != "application/gzip" {
		t.Fatalf("tar.gz: %d %v", code, h)
	}
	gz, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	names := map[string]bool{}
	for {
		hd, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names[hd.Name] = true
	}
	if !names["lib-main/src/main.go"] || !names["lib-main/README.md"] {
		t.Fatalf("conteúdo do tar.gz: %v", names)
	}

	if code, _, _ := baixar(t, ada, "/api/v4/projects/"+pid+"/repository/archive.zip?sha=nao-existe"); code != 404 {
		t.Fatalf("revisão inexistente: %d", code)
	}
	if code, _, _ := baixar(t, ada, "/api/v4/projects/"+pid+"/repository/archive.zip?sha=--output=x"); code != 400 {
		t.Fatalf("revisão com forma de opção: %d", code)
	}
	if code, _, _ := baixar(t, eve, "/api/v4/projects/"+pid+"/repository/archive.zip"); code != 404 {
		t.Fatalf("projeto privado baixado por quem não é membro: %d", code)
	}
	if code, _, _ := baixar(t, &api{t: t, base: base}, "/api/v4/projects/"+pid+"/repository/archive.zip"); code != 404 && code != 401 {
		t.Fatalf("projeto privado baixado por visitante: %d", code)
	}
}
