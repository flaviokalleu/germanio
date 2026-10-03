package runtime

import (
	"net/url"
	"regexp"
	"strings"
	"testing"
)

// Data shown by state (GEP 0023, em teste).
func TestPorEstado(t *testing.T) {
	_, c := loadApp(t, "testdata/por_estado/app.ge")
	ana := signIn(t, c.base, "Ana", "ana@x.com")
	bia := signIn(t, c.base, "Bia", "bia@x.com")
	ana.expect("POST", "/_ge/api/quadros", map[string]any{"nome": "Produto"}, 201)
	bid := bia.expect("GET", "/_ge/eu", nil, 200)["id"]
	ana.expect("POST", "/_ge/api/quadros/1/membros", map[string]any{"pessoa_id": bid, "papel": "leitor"}, 201)
	ana.expect("POST", "/_ge/api/quadros/1/cartoes", map[string]any{"titulo": "Login"}, 201)

	_, _, page := ana.do("GET", "/quadros/1", nil)
	cols := regexp.MustCompile(`class="coluna" data-estado="(\w+)"`).FindAllStringSubmatch(page, -1)
	var names []string
	for _, m := range cols {
		names = append(names, m[1])
	}
	if strings.Join(names, ",") != "pendente,comecado,concluido" {
		t.Fatalf("colunas pelos estados, na ordem: %v\n%s", names, page)
	}
	// the gestor (editor or above) may start or finish a pending card, not reopen it
	if !strings.Contains(page, `/quadros/1/cartoes/1/acao/comecar`) || !strings.Contains(page, `/quadros/1/cartoes/1/acao/concluir`) || strings.Contains(page, `acao/reabrir`) {
		t.Fatalf("movimentos de um cartão pendente:\n%s", page)
	}
	// a reader sees the board but no move
	if _, _, p := bia.do("GET", "/quadros/1", nil); !strings.Contains(p, "Login") || strings.Contains(p, "/cartoes/1/acao/") || strings.Contains(p, `draggable="true"`) {
		t.Fatalf("o leitor vê o cartão e nenhum movimento:\n%s", p)
	}
	// moving by the button is the transition
	form := url.Values{"_csrf": {ana.csrf}}
	resp, err := ana.http.PostForm(c.base+"/quadros/1/cartoes/1/acao/concluir", form)
	if err != nil || resp.StatusCode >= 400 {
		t.Fatalf("mover pelo botão: %v %v", err, resp)
	}
	resp.Body.Close()
	if got := ana.expect("GET", "/_ge/api/quadros/1/cartoes/1", nil, 200)["estado"]; got != "concluido" {
		t.Fatalf("o cartão foi para concluido: %v", got)
	}
	if _, _, p := ana.do("GET", "/quadros/1", nil); !strings.Contains(p, `acao/reabrir`) {
		t.Fatalf("um cartão concluído pode ser reaberto:\n%s", p)
	}
	// a forged move by the reader is refused by the server
	bia.do("POST", "/_ge/api/quadros/1/cartoes/1/reabrir", nil)
	if got := ana.expect("GET", "/_ge/api/quadros/1/cartoes/1", nil, 200)["estado"]; got != "concluido" {
		t.Fatalf("o leitor moveu o cartão: %v", got)
	}
}
