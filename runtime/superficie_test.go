package runtime

import (
	"strings"
	"testing"
)

// An adapter asks the application itself (GEP 0033, em teste): the call runs
// as the person of the request, with the same rules, visibility and counts —
// in a domain with no Git, no CI and no GitLab.
func TestAdaptadorPedeAAplicacao(t *testing.T) {
	_, anon := loadApp(t, "testdata/superficie/app.ge")
	ana := signIn(t, anon.base, "Ana", "ana@x.com")
	bia := signIn(t, anon.base, "Bia", "bia@x.com")
	ana.expect("POST", "/_ge/api/estantes", map[string]any{"nome": "Clássicos"}, 201)
	ana.expect("POST", "/_ge/api/estantes/1/livros", map[string]any{"titulo": "Dom Casmurro"}, 201)
	ana.expect("POST", "/_ge/api/estantes/1/livros", map[string]any{"titulo": "Iracema", "raro": true}, 201)

	// a write, as the person: the author may, someone else may not
	if got := ana.expect("POST", "/catalogo/estantes/1/livros/1/renomear", map[string]any{"novo": "Memórias"}, 200); got["title"] != "Memórias" {
		t.Fatalf("a resposta vem com os nomes da integração: %v", got)
	}
	bia.expect("POST", "/catalogo/estantes/1/livros/1/renomear", map[string]any{"novo": "Roubado"}, 403)
	anon.expect("POST", "/catalogo/estantes/1/livros/1/renomear", map[string]any{"novo": "Roubado"}, 404) // the shelf is not visible to visitors
	bia.expect("POST", "/catalogo/estantes/1/livros/2/renomear", map[string]any{"novo": "Roubado"}, 404)
	if b := ana.expect("GET", "/_ge/api/estantes/1/livros/1", nil, 200); b["titulo"] != "Memórias" {
		t.Fatalf("só a autora renomeia: %v", b)
	}
	link := *ana // a link planted on another site: the session cookie, no CSRF proof
	link.csrf = ""
	link.expect("GET", "/catalogo/estantes/1/livros/1/renomear_por_link", nil, 403)
	// a refused change leaves nothing behind
	if code, _, raw := ana.do("GET", "/_ge/api/estantes/1/livros", nil); code != 200 || strings.Contains(raw, "Roubado") || strings.Contains(raw, "Por link") {
		t.Fatalf("nada de quem não podia: %s", raw)
	}

	// reads and counts see only what the person sees
	if r := ana.expect("GET", "/catalogo/estantes/1/resumo", nil, 200); r["nome"] != "Clássicos" || r["livros"] != "2" || r["visiveis"] != float64(2) || r["disponiveis"] != float64(2) {
		t.Fatalf("resumo da autora: %v", r)
	}
	if r := bia.expect("GET", "/catalogo/estantes/1/resumo", nil, 200); r["livros"] != "1" || r["visiveis"] != float64(1) {
		t.Fatalf("o livro raro não conta para quem não pode vê-lo: %v", r)
	}
	anon.expect("GET", "/catalogo/estantes/1/resumo", nil, 404)

	// the adapter never leaves the integration, never reaches a declared
	// route (no loops) and counts only real states
	if code, _, raw := ana.do("GET", "/catalogo/fora", nil); code != 500 {
		t.Fatalf("caminho fora da integração: %d %s", code, raw)
	}
	if code, _, raw := ana.do("GET", "/catalogo/estado", nil); code != 500 {
		t.Fatalf("estado inexistente: %d %s", code, raw)
	}
	if r := ana.expect("GET", "/catalogo/eco", nil, 200); r["estado"] != float64(508) {
		t.Fatalf("uma rota declarada não é alcançada por superficie.pedir: %v", r)
	}
}
