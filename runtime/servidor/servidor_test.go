package servidor

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flaviokalleu/germanio/compiler/ast"
)

func TestHandlePaginaServesCustomRootPage(t *testing.T) {
	s := Novo(&ast.Program{
		System: &ast.System{Name: "teste"},
		Pages: []*ast.CustomPage{{
			Path:    "/",
			Title:   "Portal",
			Content: "<!DOCTYPE html><html><body>portal oficial</body></html>",
		}},
	}, nil, "0")

	r := httptest.NewRequest("GET", "http://example.test/", nil)
	w := httptest.NewRecorder()
	s.handlePagina(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), "portal oficial") {
		t.Fatalf("root body did not contain custom page: %q", w.Body.String())
	}
	if got := w.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("Content-Type = %q, want HTML", got)
	}
}

func TestHandlePaginaServesDeclarativeBlocks(t *testing.T) {
	s := Novo(&ast.Program{
		System: &ast.System{Name: "teste"},
		Pages: []*ast.CustomPage{{
			Path:  "/declarativo",
			Title: "Germanio Declarativo",
			Blocks: []ast.PageUIBlock{
				&ast.PageNavbar{Brand: "Germanio", Links: []*ast.PageNavLink{{Label: "Docs", URL: "/docs"}}},
				&ast.PageHero{Title: "Diga o que quer construir.", Badge: "V1"},
			},
		}},
	}, nil, "0")

	r := httptest.NewRequest("GET", "http://example.test/declarativo", nil)
	w := httptest.NewRecorder()
	s.handlePagina(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Germanio Declarativo") || !strings.Contains(body, "Diga o que quer construir.") || !strings.Contains(body, ">Docs<") {
		t.Fatalf("declarative page body missing rendered blocks: %q", body)
	}
}

func TestIPRealSoConfiaEmProxiesDeclarados(t *testing.T) {
	req := func(remote string, h map[string]string) *http.Request {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = remote
		for k, v := range h {
			r.Header.Set(k, v)
		}
		return r
	}
	spoof := map[string]string{"X-Forwarded-For": "1.2.3.4", "CF-Connecting-IP": "5.6.7.8", "X-Real-IP": "9.9.9.9"}
	// Sem proxies declarados, cabeçalhos são ignorados.
	proxiesNets = nil
	proxiesOnce.Do(func() {})
	if ip := getRealIP(req("203.0.113.7:5000", spoof)); ip != "203.0.113.7" {
		t.Fatalf("cabeçalho forjado aceito: %s", ip)
	}
	proxiesNets = parseProxies("10.0.0.0/8, 127.0.0.1")
	defer func() { proxiesNets = nil }()
	// Conexão direta (não é proxy): cabeçalhos ignorados.
	if ip := getRealIP(req("203.0.113.7:5000", spoof)); ip != "203.0.113.7" {
		t.Fatalf("cliente direto escolheu o IP: %s", ip)
	}
	// Via proxy: o cliente é o último endereço que não é proxy.
	if ip := getRealIP(req("10.0.0.2:80", map[string]string{"X-Forwarded-For": "6.6.6.6, 198.51.100.9, 10.0.0.5"})); ip != "198.51.100.9" {
		t.Fatalf("XFF via proxy: %s", ip)
	}
	if ip := getRealIP(req("127.0.0.1:80", map[string]string{"X-Real-IP": "198.51.100.10"})); ip != "198.51.100.10" {
		t.Fatalf("X-Real-IP via proxy: %s", ip)
	}
	if ip := getRealIP(req("10.0.0.2:80", map[string]string{"X-Forwarded-For": "lixo"})); ip != "10.0.0.2" {
		t.Fatalf("XFF inválido: %s", ip)
	}
}
