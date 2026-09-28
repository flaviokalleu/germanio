package runtime

import (
	"regexp"
	"strings"
	"testing"
)

// Indicators (GEP 0012, em teste): a number counts only what the viewer
// could list; an administrator counts everything.

func numbers(t *testing.T, c *client, path string) map[string]string {
	t.Helper()
	code, _, raw := c.do("GET", path, nil)
	if code != 200 {
		t.Fatalf("GET %s: %d %s", path, code, raw)
	}
	out := map[string]string{}
	for _, m := range regexp.MustCompile(`<span class="valor">(\d+)</span><span class="rotulo">([^<]+)</span>`).FindAllStringSubmatch(raw, -1) {
		out[m[2]] = m[1]
	}
	return out
}

func TestIndicadores(t *testing.T) {
	t.Setenv("GERMANIO_ADMIN_SENHA", "admin-senha-longa-1")
	_, c := loadApp(t, "testdata/indicadores/app.ge")
	ana := signIn(t, c.base, "Ana", "ana@x.com")
	bia := signIn(t, c.base, "Bia", "bia@x.com")
	ana.expect("POST", "/_ge/api/projetos", map[string]any{"nome": "Loja"}, 201)
	ana.expect("POST", "/_ge/api/projetos/1/issues", map[string]any{"titulo": "Um"}, 201)
	ana.expect("POST", "/_ge/api/projetos/1/issues", map[string]any{"titulo": "Dois"}, 201)
	ana.expect("POST", "/_ge/api/projetos/1/issues/2/fechar", nil, 200)
	ana.expect("POST", "/_ge/api/projetos/1/issues", map[string]any{"titulo": "Segredo", "confidencial": true}, 201)

	// the list itself must not show a confidential record (it did, before
	// restrictions made visibility record-dependent)
	for _, path := range []string{"/_ge/api/projetos/1/issues", "/_ge/api/issues"} {
		if _, _, raw := bia.do("GET", path, nil); strings.Contains(raw, "Segredo") {
			t.Fatalf("%s mostrou uma issue confidencial a quem não pode vê-la: %s", path, raw)
		}
	}
	got := numbers(t, bia, "/painel")
	if got["Total de issues abertas"] != "1" || got["Resolvidas"] != "1" || got["Total de projetos"] != "1" {
		t.Fatalf("Bia não vê a issue confidencial, então não a conta: %v", got)
	}
	if _, shown := got["Total de usuarios"]; shown {
		t.Fatalf("usuários não são visíveis para Bia: o indicador não aparece: %v", got)
	}
	if got := numbers(t, ana, "/painel"); got["Total de issues abertas"] != "2" {
		t.Fatalf("Ana vê a sua issue confidencial: %v", got)
	}
	// an administrator counts everything
	_, root := c.fresh(t)
	root.expect("POST", "/entrar", map[string]any{"login": "root@x.local", "senha": "admin-senha-longa-1"}, 200)
	if got := numbers(t, root, "/painel"); got["Total de issues abertas"] != "2" || got["Total de usuarios"] != "3" {
		t.Fatalf("o administrador conta tudo: %v", got)
	}
	// indicators above a list
	if got := numbers(t, ana, "/projetos"); got["Total de projetos"] != "1" {
		t.Fatalf("indicador na página da lista: %v", got)
	}
	// a dashboard has nothing to post to, and a visitor is sent to sign in
	if code, _, _ := ana.do("POST", "/painel", map[string]any{}); code != 404 {
		t.Fatalf("POST no painel: %d", code)
	}
	_, anon := c.fresh(t)
	if code, _, raw := anon.do("GET", "/painel", nil); code != 303 || strings.Contains(raw, "valor") {
		t.Fatalf("visitante no painel: %d %s", code, raw)
	}
}
