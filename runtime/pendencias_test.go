package runtime

import (
	"strings"
	"testing"
)

// Pending items (GEP 0009, em teste): a person placed among the assignees
// gets a pending item that only they see and can complete; whoever assigns
// gets none for themselves; removing a person removes their open items;
// deleting the record deletes its items.
func TestPendencias(t *testing.T) {
	_, ana := loadApp(t, "testdata/pendencias/app.ge")
	people := map[string]*client{"ana": ana}
	ids := map[string]float64{}
	for i, n := range []string{"ana", "bia", "caio"} {
		c := ana
		if i > 0 {
			_, c = ana.fresh(t)
			people[n] = c
		}
		me := c.expect("POST", "/cadastro", map[string]any{"nome": n, "email": n + "@x.com", "senha": "senha-" + n + "-123"}, 201)
		c.csrf = csrfFromCookie(t, c)
		ids[n] = me["id"].(float64)
	}
	bia, caio := people["bia"], people["caio"]
	ch := ana.expect("POST", "/_ge/api/chamados", map[string]any{"titulo": "Impressora", "responsaveis": []any{ids["bia"], ids["ana"]}}, 201)
	path := "/_ge/api/chamados/" + itoa(int(ch["id"].(float64)))

	list := func(c *client) []map[string]any {
		_, _, raw := c.do("GET", "/_ge/api/pendencias", nil)
		return decodeList(t, raw)
	}
	mine := list(bia)
	if len(mine) != 1 || !strings.Contains(mine[0]["motivo"].(string), "Impressora") {
		t.Fatalf("bia deveria ter uma pendência do chamado: %v", mine)
	}
	if n := len(list(ana)); n != 0 {
		t.Fatalf("quem atribui não recebe pendência para si: %d", n)
	}
	pid := itoa(int(mine[0]["id"].(float64)))
	ana.expect("GET", "/_ge/api/pendencias/"+pid, nil, 404) // someone else's
	done := bia.expect("POST", "/_ge/api/pendencias/"+pid+"/concluir", nil, 200)
	if done["estado"] != "concluida" {
		t.Fatalf("concluir: %v", done)
	}

	// caio in, bia out: caio gets one; bia's completed item stays
	ana.expect("PATCH", path, map[string]any{"responsaveis": []any{ids["caio"]}}, 200)
	if n := len(list(caio)); n != 1 {
		t.Fatalf("caio deveria ter uma pendência: %d", n)
	}
	if n := len(list(bia)); n != 1 {
		t.Fatalf("a pendência concluída de bia deveria ficar: %d", n)
	}
	// removing caio removes his open item
	ana.expect("PATCH", path, map[string]any{"responsaveis": []any{}}, 200)
	if n := len(list(caio)); n != 0 {
		t.Fatalf("quem sai perde a pendência aberta: %d", n)
	}
	// deleting the record deletes its items
	ana.expect("DELETE", path, nil, 204)
	if n := len(list(bia)); n != 0 {
		t.Fatalf("excluir o chamado remove as pendências: %d", n)
	}
}
