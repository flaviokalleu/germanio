package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/flaviokalleu/germanio/runtime"
)

type pessoa struct {
	t    *testing.T
	base string
	c    *http.Client
	csrf string
}

func (p *pessoa) do(method, path string, body any) (int, map[string]any) {
	p.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, p.base+path, rd)
	req.Header.Set("Content-Type", "application/json")
	if p.csrf != "" {
		req.Header.Set("X-CSRF-Token", p.csrf)
	}
	resp, err := p.c.Do(req)
	if err != nil {
		p.t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out map[string]any
	json.Unmarshal(raw, &out)
	return resp.StatusCode, out
}

func (p *pessoa) must(method, path string, body any, status int) map[string]any {
	p.t.Helper()
	code, out := p.do(method, path, body)
	if code != status {
		p.t.Fatalf("%s %s = %d, esperado %d: %v", method, path, code, status, out)
	}
	return out
}

var csrfRe = regexp.MustCompile(`name="_csrf" value="([^"]*)"`)

// entrar signs in with the session cookie and picks the CSRF token from a page.
func entrar(t *testing.T, base, email, senha string) *pessoa {
	jar, _ := cookiejar.New(nil)
	p := &pessoa{t: t, base: base, c: &http.Client{Jar: jar}}
	p.must("POST", "/entrar", map[string]any{"email": email, "senha": senha}, 200)
	resp, err := p.c.Get(base + "/pedidos")
	if err != nil {
		t.Fatal(err)
	}
	page, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	m := csrfRe.FindSubmatch(page)
	if m == nil {
		t.Fatalf("página sem CSRF:\n%s", page)
	}
	p.csrf = string(m[1])
	return p
}

func carregar(t *testing.T, app string) (*runtime.App, error) {
	t.Setenv("GERMANIO_SQLITE", filepath.Join(t.TempDir(), "loja.db"))
	t.Setenv("GERMANIO_BCRYPT_RAPIDO", "1")
	t.Setenv("ADMIN_EMAIL", "admin@loja.local")
	t.Setenv("ADMIN_SENHA", "senha-do-admin")
	return runtime.Carregar(app, "0")
}

// germanio init: backend/ e frontend/ importados inteiros e funcionando.
func TestInitBackendFrontend(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "loja")
	cmdInit(dir)
	for _, f := range []string{"app.ge", "backend/pessoas.ge", "backend/produtos.ge", "backend/pedidos.ge", "frontend/loja.ge", ".env.exemplo", ".gitignore"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Fatalf("faltou %s", f)
		}
	}
	app, err := carregar(t, filepath.Join(dir, "app.ge"))
	if err != nil {
		t.Fatalf("Carregar: %v", err)
	}
	srv := httptest.NewServer(app.Handler)
	defer func() { srv.Close(); app.Fechar() }()

	admin := entrar(t, srv.URL, "admin@loja.local", "senha-do-admin")
	admin.must("POST", "/_ge/api/produtos", map[string]any{"nome": "Caneca", "preco": 30, "estoque": 2}, 201)

	anon := &pessoa{t: t, base: srv.URL, c: http.DefaultClient}
	anon.must("POST", "/cadastro", map[string]any{"nome": "Ana", "email": "ana@example.com", "senha": "senha-da-ana"}, 201)
	anon.must("POST", "/cadastro", map[string]any{"nome": "Bia", "email": "bia@example.com", "senha": "senha-da-bia"}, 201)
	if code, _ := anon.do("GET", "/_ge/api/produtos", nil); code != 200 {
		t.Fatalf("todos podem ver produtos: %d", code)
	}
	ana := entrar(t, srv.URL, "ana@example.com", "senha-da-ana")
	ana.must("POST", "/_ge/api/produtos", map[string]any{"nome": "X", "preco": 1}, 403)
	if out := ana.must("POST", "/_ge/api/pedidos", map[string]any{"produto_id": 1, "quantidade": 5}, 400); !strings.Contains(strings.ToLower(jsonText(out)), "estoque insuficiente") {
		t.Fatalf("regra de estoque: %v", out)
	}
	p := ana.must("POST", "/_ge/api/pedidos", map[string]any{"produto_id": 1}, 201)
	if p["estado"] != "aberto" || p["quantidade"] != float64(1) {
		t.Fatalf("pedido criado: %v", p)
	}
	if p = ana.must("POST", "/_ge/api/pedidos/1/pagar", nil, 200); p["estado"] != "pago" {
		t.Fatalf("pagar: %v", p)
	}
	bia := entrar(t, srv.URL, "bia@example.com", "senha-da-bia")
	bia.must("GET", "/_ge/api/pedidos/1", nil, 404)
}

func jsonText(v any) string { b, _ := json.Marshal(v); return string(b) }

// Cada pasta mantém seu papel.
func TestPastasMantemSeuPapel(t *testing.T) {
	for _, c := range []struct{ file, src, want string }{
		{"frontend/cupons.ge", "tenha cupons\n", "vai em backend/"},
		{"frontend/regras.ge", "somente administrador pode excluir produtos\n", "vai em backend/"},
		{"backend/tela.ge", "crie página Cupons\n    mostre produtos\n", "vão em frontend/"},
		{"integracoes/externo.ge", "tenha cupons\n", "integracoes/ só traduz"},
		{"backend/formato.ge", "traduza arquivos de execução com ler\n", "vai em integracoes/"},
		{"integracoes/externo.ge", "somente administrador pode excluir produtos\n", "integracoes/ só traduz"},
	} {
		dir := filepath.Join(t.TempDir(), "loja")
		cmdInit(dir)
		os.MkdirAll(filepath.Dir(filepath.Join(dir, c.file)), 0755)
		os.WriteFile(filepath.Join(dir, c.file), []byte(c.src), 0644)
		if strings.HasPrefix(c.file, "integracoes/") {
			app, _ := os.ReadFile(filepath.Join(dir, "app.ge"))
			os.WriteFile(filepath.Join(dir, "app.ge"), append(app, []byte("importar \"integracoes\"\n")...), 0644)
		}
		_, err := carregar(t, filepath.Join(dir, "app.ge"))
		if err == nil || !strings.Contains(err.Error(), c.want) || !strings.Contains(err.Error(), c.file) {
			t.Fatalf("%s: erro esperado com %q, veio %v", c.file, c.want, err)
		}
	}
}

// O frontend diz o que usa do backend, e o que ele importa precisa existir.
func TestFrontendImportaDoBackend(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{"crie página Pessoas\n    mostre usuarios\n", "importar usuarios do backend"},
		{"importar cupons do backend\n", "backend não tem cupons (tem: pedidos, produtos, usuarios)"},
		{"importar usuarios do servidor\n", "não existe a pasta servidor/"},
		{"importar usuarios do backend\n\ncrie página Pessoas\n    mostre usuarios\n", ""},
	} {
		dir := filepath.Join(t.TempDir(), "loja")
		cmdInit(dir)
		os.WriteFile(filepath.Join(dir, "frontend", "pessoas.ge"), []byte(c.src), 0644)
		app, err := carregar(t, filepath.Join(dir, "app.ge"))
		if c.want == "" {
			if err != nil {
				t.Fatalf("%q: %v", c.src, err)
			}
			app.Fechar()
			continue
		}
		if err == nil || !strings.Contains(err.Error(), c.want) || !strings.Contains(err.Error(), "frontend/pessoas.ge") {
			t.Fatalf("%q: erro esperado com %q, veio %v", c.src, c.want, err)
		}
	}
}
