package runtime

import (
	"strings"
	"testing"
)

// Reading (GEP 0022, em teste).
func TestLeitura(t *testing.T) {
	_, c := loadApp(t, "testdata/leitura/app.ge")
	ana := signIn(t, c.base, "Ana", "ana@x.com")
	bia := signIn(t, c.base, "Bia", "bia@x.com")
	ana.expect("POST", "/_ge/api/canais", map[string]any{"nome": "geral"}, 201)
	unread := func(who *client) any { return who.expect("GET", "/_ge/api/canais/1", nil, 200)["nao_lidas"] }

	list, _ := watch(t, bia, "/canais")
	ana.expect("POST", "/_ge/api/canais/1/mensagens", map[string]any{"texto": "um"}, 201)
	ana.expect("POST", "/_ge/api/canais/1/mensagens", map[string]any{"texto": "dois"}, 201)
	expectChange(t, list, true, "a lista de canais segue a contagem")
	if unread(bia) != float64(2) {
		t.Fatalf("Bia tem 2 não lidas: %v", unread(bia))
	}
	if unread(ana) != float64(0) {
		t.Fatalf("o que Ana escreveu não conta para ela: %v", unread(ana))
	}
	if _, _, html := bia.do("GET", "/canais", nil); !strings.Contains(html, "2 não lidas") {
		t.Fatalf("a lista mostra a contagem:\n%s", html)
	}
	// opening the channel reads it, and only for Bia
	bia.do("GET", "/canais/1", nil)
	if unread(bia) != float64(0) {
		t.Fatalf("abrir o canal lê tudo: %v", unread(bia))
	}
	expectChange(t, list, true, "ler muda a contagem")
	bia.expect("POST", "/_ge/api/canais/1/mensagens", map[string]any{"texto": "três"}, 201)
	if unread(ana) != float64(1) || unread(bia) != float64(0) {
		t.Fatalf("marcas por pessoa: ana=%v bia=%v", unread(ana), unread(bia))
	}
}
