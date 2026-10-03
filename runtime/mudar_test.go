package runtime

import (
	"strings"
	"testing"
)

// A record moves to another parent (GEP 0034, em teste): `chamado pode mudar
// de fila` — in a support desk, no Git and no GitLab. Whoever moves must be
// able to change it where it is and create one in the destination; it gets
// the destination's next number; labels follow by name; read-only places
// neither give nor take.
func TestMudarDeLugar(t *testing.T) {
	_, anon := loadApp(t, "testdata/mudar/app.ge")
	ana := signIn(t, anon.base, "Ana", "ana@x.com")
	bia := signIn(t, anon.base, "Bia", "bia@x.com")
	biaID := bia.expect("GET", "/_ge/eu", nil, 200)["id"]
	ana.expect("POST", "/_ge/api/filas", map[string]any{"nome": "Suporte"}, 201)    // 1
	ana.expect("POST", "/_ge/api/filas", map[string]any{"nome": "Financeiro"}, 201) // 2
	bia.expect("POST", "/_ge/api/filas", map[string]any{"nome": "Da Bia"}, 201)     // 3
	ana.expect("POST", "/_ge/api/filas/1/membros", map[string]any{"pessoa_id": biaID, "papel": "atendente"}, 201)
	ana.expect("POST", "/_ge/api/filas/1/etiquetas", map[string]any{"nome": "urgente"}, 201)
	ana.expect("POST", "/_ge/api/filas/1/etiquetas", map[string]any{"nome": "lento"}, 201)
	ana.expect("POST", "/_ge/api/filas/2/etiquetas", map[string]any{"nome": "urgente"}, 201)
	ana.expect("POST", "/_ge/api/filas/1/chamados", map[string]any{"titulo": "Impressora", "etiquetas": "urgente,lento"}, 201)
	ana.expect("POST", "/_ge/api/filas/1/chamados", map[string]any{"titulo": "Rede"}, 201)
	ana.expect("POST", "/_ge/api/filas/2/chamados", map[string]any{"titulo": "Boleto"}, 201)

	moved := ana.expect("POST", "/_ge/api/filas/1/chamados/1/mudar", map[string]any{"fila_id": 2}, 200)
	if moved["numero"] != float64(2) || moved["fila_id"] != float64(2) {
		t.Fatalf("o chamado ganha o próximo número da fila de destino: %v", moved)
	}
	if l, _ := moved["etiquetas"].([]any); len(l) != 1 || l[0] != "urgente" {
		t.Fatalf("as etiquetas seguem pelo nome, só as que o destino tem: %v", moved["etiquetas"])
	}
	if got := ana.expect("GET", "/_ge/api/filas/2/chamados/2", nil, 200); got["titulo"] != "Impressora" {
		t.Fatalf("o chamado está na fila de destino: %v", got)
	}
	ana.expect("GET", "/_ge/api/filas/1/chamados/1", nil, 404)
	// the next one created at the destination does not repeat the number
	if c := ana.expect("POST", "/_ge/api/filas/2/chamados", map[string]any{"titulo": "Nota"}, 201); c["numero"] != float64(3) {
		t.Fatalf("a numeração do destino continua: %v", c)
	}
	// the history says it moved, and where it is now
	code, _, raw := ana.do("GET", "/_ge/api/atividades?acao=mudar", nil)
	if code != 200 || !strings.Contains(raw, `"acao":"mudar"`) || !strings.Contains(raw, `"dentro_id":2`) {
		t.Fatalf("histórico da mudança: %d %s", code, raw)
	}

	// Bia sees the queue but may not change Ana's ticket there
	bia.expect("POST", "/_ge/api/filas/1/chamados/2/mudar", map[string]any{"fila_id": 3}, 403)
	// Ana may not create in Bia's queue: for her it does not exist
	ana.expect("POST", "/_ge/api/filas/1/chamados/2/mudar", map[string]any{"fila_id": 3}, 404)
	ana.expect("POST", "/_ge/api/filas/1/chamados/2/mudar", map[string]any{"fila_id": 1}, 400)
	ana.expect("POST", "/_ge/api/filas/1/chamados/2/mudar", nil, 400)
	// a read-only queue neither receives nor lets go
	ana.expect("PUT", "/_ge/api/filas/2", map[string]any{"arquivada": true}, 200)
	ana.expect("POST", "/_ge/api/filas/1/chamados/2/mudar", map[string]any{"fila_id": 2}, 403)
	ana.expect("POST", "/_ge/api/filas/2/chamados/2/mudar", map[string]any{"fila_id": 1}, 403)
	if got := ana.expect("GET", "/_ge/api/filas/1/chamados/2", nil, 200); got["titulo"] != "Rede" {
		t.Fatalf("nada mudou com as recusas: %v", got)
	}
	// the number is never changed by editing
	ana.expect("PUT", "/_ge/api/filas/1/chamados/2", map[string]any{"numero": 9}, 400)
}
