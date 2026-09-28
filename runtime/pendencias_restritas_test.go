package runtime

import (
	"strings"
	"testing"
)

// A pending item never tells its owner about a record they cannot see.
func TestPendenciaNaoRevelaRegistroRestrito(t *testing.T) {
	_, c := loadApp(t, "testdata/pendencias_restritas/app.ge")
	ana := signIn(t, c.base, "Ana", "ana@x.com")
	bia := signIn(t, c.base, "Bia", "bia@x.com")
	me := bia.expect("GET", "/_ge/eu", nil, 200)
	ana.expect("POST", "/_ge/api/chamados", map[string]any{"titulo": "Demissão do diretor", "confidencial": true, "responsaveis": []any{me["id"]}}, 201)
	_, _, raw := bia.do("GET", "/_ge/api/pendencias", nil)
	if strings.Contains(raw, "Demissão") {
		t.Fatalf("a pendência revelou o título de um chamado que Bia não pode ver: %s", raw)
	}
	if !strings.Contains(raw, `"motivo":"Responsaveis: Chamado"`) {
		t.Fatalf("Bia continua sabendo que foi posta como responsável: %s", raw)
	}
	// a record she can see keeps its title
	ana.expect("POST", "/_ge/api/chamados", map[string]any{"titulo": "Impressora", "responsaveis": []any{me["id"]}}, 201)
	if _, _, raw := bia.do("GET", "/_ge/api/pendencias", nil); !strings.Contains(raw, "Impressora") {
		t.Fatalf("o título de um chamado visível deveria aparecer: %s", raw)
	}
}
