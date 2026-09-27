package servidor

import (
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
