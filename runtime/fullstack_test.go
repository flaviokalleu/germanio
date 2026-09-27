package runtime

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// client wraps an httptest server with a cookie jar and JSON helpers.
type client struct {
	t    *testing.T
	base string
	http *http.Client
	csrf string
	// Header is added to every request (for example PRIVATE-TOKEN).
	Header http.Header
}

func loadApp(t *testing.T, file string) (*App, *client) {
	t.Helper()
	t.Setenv("GERMANIO_SQLITE", filepath.Join(t.TempDir(), "app.db"))
	t.Setenv("GERMANIO_BCRYPT_RAPIDO", "1")
	t.Setenv("GERMANIO_GIT_RAIZ", filepath.Join(t.TempDir(), "repos"))
	app, err := Carregar(file, "0")
	if err != nil {
		t.Fatalf("Carregar(%s): %v", file, err)
	}
	srv := httptest.NewServer(app.Handler)
	t.Cleanup(func() { srv.Close(); app.Fechar() })
	jar, _ := cookiejar.New(nil)
	return app, &client{t: t, base: srv.URL, http: &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, Header: http.Header{}}
}

func (c *client) do(method, path string, body any) (int, map[string]any, string) {
	c.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, c.base+path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.csrf != "" {
		req.Header.Set("X-CSRF-Token", c.csrf)
	}
	for k, v := range c.Header {
		req.Header[k] = v
	}
	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]any
	json.Unmarshal(raw, &out)
	return resp.StatusCode, out, string(raw)
}

func (c *client) expect(method, path string, body any, status int) map[string]any {
	c.t.Helper()
	got, out, raw := c.do(method, path, body)
	if got != status {
		c.t.Fatalf("%s %s: status %d, esperado %d; corpo %s", method, path, got, status, raw)
	}
	return out
}

func TestFullstackRoutesAndCapabilities(t *testing.T) {
	_, c := loadApp(t, "testdata/fullstack/app.ge")

	c.expect("POST", "/api/contas", map[string]any{"login": "ada", "senha": "curta"}, 400)
	out := c.expect("POST", "/api/contas", map[string]any{"login": "ada", "senha": "segredo123"}, 201)
	if out["login"] != "ada" || out["senha_hash"] != nil {
		t.Fatalf("resposta de criação inesperada: %v", out)
	}
	// declared `unico` → field-level validation error (GitLab/Rails format)
	dup := c.expect("POST", "/api/contas", map[string]any{"login": "ada", "senha": "segredo123"}, 400)
	if msg, _ := dup["message"].(map[string]any); msg == nil || msg["login"].([]any)[0] != "has already been taken" {
		t.Fatalf("erro de unicidade inesperado: %v", dup)
	}

	// internal models have no automatic REST endpoint
	c.expect("GET", "/api/conta", nil, 404)

	c.expect("POST", "/api/notas", map[string]any{"title": "x"}, 401)
	c.expect("POST", "/api/sessao", map[string]any{"login": "ada", "senha": "errada!!"}, 401)
	sess := c.expect("POST", "/api/sessao", map[string]any{"login": "ada", "senha": "segredo123"}, 200)

	// Session cookie present but no CSRF token → refused
	c.expect("POST", "/api/notas", map[string]any{"title": "sem csrf"}, 403)
	c.csrf = sess["csrf"].(string)

	n1 := c.expect("POST", "/api/notas", map[string]any{"title": "primeira"}, 201)
	n2 := c.expect("POST", "/api/notas", map[string]any{"title": "segunda"}, 201)
	if n1["iid"].(float64) != 1 || n2["iid"].(float64) != 2 {
		t.Fatalf("iids sequenciais esperados 1 e 2, obtidos %v %v", n1["iid"], n2["iid"])
	}
	got := c.expect("GET", "/api/notas/2", nil, 200)
	meta := got["meta"].(map[string]any)
	if got["title"] != "segunda" || got["state"] != "opened" || meta["dono"] != "ada" || len(meta["tags"].([]any)) != 2 {
		t.Fatalf("nota inesperada: %v", got)
	}
	c.expect("GET", "/api/notas/99", nil, 404)
	upd := c.expect("PUT", "/api/notas/1", map[string]any{"state": "closed"}, 200)
	if upd["state"] != "closed" || upd["title"] != "primeira" {
		t.Fatalf("atualização parcial falhou: %v", upd)
	}
	list := c.expect("GET", "/api/notas?state=opened", nil, 200)
	if list["total"].(float64) != 1 || list["primeiro"] != "segunda" {
		t.Fatalf("filtro por state falhou: %v", list)
	}
	all := c.expect("GET", "/api/notas", nil, 200)
	if titles := all["titulos"].([]any); len(titles) != 2 || titles[0] != "segunda" {
		t.Fatalf("ordenação -iid falhou: %v", all)
	}

	if r := c.expect("GET", "/api/classificar/50", nil, 200); r["a"] != "grande!" {
		t.Fatalf("senao/indentação: %v", r)
	}
	if r := c.expect("GET", "/api/classificar/5", nil, 200); r["a"] != "pequeno!" {
		t.Fatalf("senao/indentação: %v", r)
	}

	// Undefined variable is a 500 with position in the log, never a silent value.
	code, body, _ := c.do("GET", "/api/quebrado", nil)
	if code != 500 || body["message"] != "500 Internal Server Error" {
		t.Fatalf("erro de variável indefinida: %d %v", code, body)
	}
	if r := c.expect("GET", "/api/capturado", nil, 200); r["capturado"] != "chá" {
		t.Fatalf("tentar/erro: %v", r)
	}
}

func TestParseErrorsHavePositions(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "ruim.ge")
	src := "sistema x\nrotas\n  rota GET \"/a\"\n    definir x = {a: 1, a: 2}\n"
	if err := writeFile(file, src); err != nil {
		t.Fatal(err)
	}
	_, err := parseFG(file)
	if err == nil || !strings.Contains(err.Error(), "ruim.ge:4") || !strings.Contains(err.Error(), "repetida") {
		t.Fatalf("esperado erro posicionado de chave repetida, obtido %v", err)
	}
}
