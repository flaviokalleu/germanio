package runtime

import (
	"bytes"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// Aggregates (GEP 0047, em teste) and pairs (GEP 0048, em teste) in a
// domain with no Git and no GitLab: accounts show their balance (the sum of
// their entries) and how many entries they have; a number counts only the
// entries the viewer may see; zeroing the balance is one change, even when
// asked many times at once; two accounts are linked once, in any order, and
// never to themselves.

func account(t *testing.T, c *client, path string) map[string]any {
	t.Helper()
	return c.expect("GET", path, nil, 200)
}

func numbersOf(t *testing.T, m map[string]any, saldo, total, conferidos float64) {
	t.Helper()
	if m["saldo"] != saldo || m["total_de_lancamentos"] != total || m["total_de_lancamentos_conferidos"] != conferidos {
		t.Fatalf("números da conta: saldo %v, total %v, conferidos %v (esperado %v %v %v): %v",
			m["saldo"], m["total_de_lancamentos"], m["total_de_lancamentos_conferidos"], saldo, total, conferidos, m)
	}
}

func TestAgregados(t *testing.T) {
	t.Setenv("GERMANIO_ADMIN_SENHA", "admin-senha-longa-1")
	app, anon := loadApp(t, "testdata/agregados/app.ge")
	ana := signIn(t, anon.base, "Ana", "ana@x.com")
	bia := signIn(t, anon.base, "Bia", "bia@x.com")
	cid := signIn(t, anon.base, "Cid", "cid@x.com")
	anaID := ana.expect("GET", "/_ge/eu", nil, 200)["id"]
	biaID := bia.expect("GET", "/_ge/eu", nil, 200)["id"]
	cidID := cid.expect("GET", "/_ge/eu", nil, 200)["id"]
	ana.expect("POST", "/_ge/api/contas", map[string]any{"nome": "Caixa"}, 201)
	ana.expect("POST", "/_ge/api/contas/1/membros", map[string]any{"pessoa_id": biaID, "papel": "leitor"}, 201)
	ana.expect("POST", "/_ge/api/contas/1/membros", map[string]any{"pessoa_id": cidID, "papel": "tesoureiro"}, 201)

	// an account with nothing yet shows zeros, not empty values
	numbersOf(t, account(t, ana, "/_ge/api/contas/1"), 0, 0, 0)
	ana.expect("POST", "/_ge/api/contas/1/lancamentos", map[string]any{"valor": 100}, 201)
	ana.expect("POST", "/_ge/api/contas/1/lancamentos", map[string]any{"valor": 50.5}, 201)
	ana.expect("POST", "/_ge/api/contas/1/lancamentos/2/conferir", nil, 200)
	ana.expect("POST", "/_ge/api/contas/1/lancamentos", map[string]any{"valor": 1000, "sigiloso": true}, 201)

	// the author sees the confidential entry; a reader does not count it
	numbersOf(t, account(t, ana, "/_ge/api/contas/1"), 1150.5, 3, 1)
	numbersOf(t, account(t, bia, "/_ge/api/contas/1"), 150.5, 2, 1)
	if _, _, l := bia.do("GET", "/_ge/api/contas/1/lancamentos", nil); strings.Contains(l, `"valor":1000`) {
		t.Fatalf("a lista também não mostra o lançamento sigiloso: %s", l)
	}
	// a list fills the numbers of every record of the page
	_, _, raw := bia.do("GET", "/_ge/api/contas", nil)
	if !strings.Contains(raw, `"saldo":150.5`) || !strings.Contains(raw, `"total_de_lancamentos":2`) {
		t.Fatalf("a lista de contas mostra os números de quem vê: %s", raw)
	}
	// an administrator counts everything
	_, root := anon.fresh(t)
	root.expect("POST", "/entrar", map[string]any{"login": "root@x.local", "senha": "admin-senha-longa-1"}, 200)
	root.csrf = csrfFromCookie(t, root)
	numbersOf(t, account(t, root, "/_ge/api/contas/1"), 1150.5, 3, 1)
	// a number is never written: sent values are ignored
	if m := ana.expect("PUT", "/_ge/api/contas/1", map[string]any{"saldo": 7, "nome": "Caixa geral"}, 200); m["saldo"] != 1150.5 || m["nome"] != "Caixa geral" {
		t.Fatalf("o saldo vem dos lançamentos, nunca da entrada: %v", m)
	}
	// the page of the account shows the numbers too, for its viewer
	if code, _, page := bia.do("GET", "/contas/1", nil); code != 200 || !strings.Contains(page, "Saldo") || !strings.Contains(page, "150.5") || strings.Contains(page, "1150.5") {
		t.Fatalf("página da conta: %d %s", code, page)
	}

	// an open list of accounts follows its numbers: a new entry announces
	// the account it belongs to (GEP 0020)
	live, code := watch(t, bia, "/contas")
	if code != 200 {
		t.Fatalf("assinatura da página: %d", code)
	}
	ana.expect("POST", "/_ge/api/contas/1/lancamentos", map[string]any{"valor": 0.5}, 201)
	expectChange(t, live, true, "um lançamento novo muda o saldo da conta")
	ana.expect("DELETE", "/_ge/api/contas/1/lancamentos/4", nil, 204)
	expectChange(t, live, true, "um lançamento excluído muda o saldo da conta")

	// zeroing: a reader may not create entries, so may not zero
	bia.expect("POST", "/_ge/api/contas/1/zerar_saldo", nil, 403)
	// a treasurer who does not see every entry would learn their sum: refused
	if _, body, raw := cid.do("POST", "/_ge/api/contas/1/zerar_saldo", nil); !strings.Contains(raw, "não vê") {
		t.Fatalf("zerar sem ver todos os lançamentos: %v %s", body, raw)
	}
	numbersOf(t, account(t, ana, "/_ge/api/contas/1"), 1150.5, 3, 1)
	if m := ana.expect("POST", "/_ge/api/contas/1/zerar_saldo", nil, 200); m["saldo"] != float64(0) || m["total_de_lancamentos"] != float64(4) {
		t.Fatalf("zerar registra o lançamento que desconta tudo: %v", m)
	}
	if n := rowsWhere(t, app, `SELECT COUNT(*) FROM lancamento WHERE valor = -1150.5 AND autor_id = `+itoa(int(anaID.(float64)))); n != 1 {
		t.Fatalf("o desconto é um lançamento de quem zerou: %d", n)
	}
	// zero again: nothing to subtract, nothing recorded
	if m := ana.expect("POST", "/_ge/api/contas/1/zerar_saldo", nil, 200); m["total_de_lancamentos"] != float64(4) {
		t.Fatalf("zerar um saldo zerado não registra nada: %v", m)
	}

	// many resets at once: exactly one subtracts, the sum ends at zero
	for i := 0; i < 3; i++ {
		ana.expect("POST", "/_ge/api/contas/1/lancamentos", map[string]any{"valor": 10}, 201)
	}
	statuses := concurrently(t, ana, 8, "POST", "/_ge/api/contas/1/zerar_saldo")
	for _, s := range statuses {
		if s != 200 {
			t.Fatalf("zerar ao mesmo tempo: %v", statuses)
		}
	}
	numbersOf(t, account(t, ana, "/_ge/api/contas/1"), 0, 8, 1)
	if n := rowsWhere(t, app, `SELECT COUNT(*) FROM lancamento WHERE valor = -30`); n != 1 {
		t.Fatalf("oito pedidos ao mesmo tempo descontam uma vez só: %d descontos de -30", n)
	}

	// the number follows a record that moves out (it is not stored anywhere)
	ana.expect("DELETE", "/_ge/api/contas/1/lancamentos/1", nil, 204)
	numbersOf(t, account(t, ana, "/_ge/api/contas/1"), -100, 7, 1)
}

func TestParUnico(t *testing.T) {
	app, anon := loadApp(t, "testdata/agregados/app.ge")
	ana := signIn(t, anon.base, "Ana", "ana@x.com")
	for _, nome := range []string{"Caixa", "Banco", "Cofre"} {
		ana.expect("POST", "/_ge/api/contas", map[string]any{"nome": nome}, 201)
	}
	ana.expect("POST", "/_ge/api/contas/1/vinculos", map[string]any{"vinculada_id": 2}, 201)
	// the same pair again, and in the other order: one link already exists
	for _, path := range []string{"/_ge/api/contas/1/vinculos", "/_ge/api/contas/2/vinculos"} {
		other := 2
		if strings.HasPrefix(path, "/_ge/api/contas/2") {
			other = 1
		}
		dup := ana.expect("POST", path, map[string]any{"vinculada_id": other}, 409)
		if m, _ := dup["message"].(map[string]any); m == nil || m["vinculada"].([]any)[0] != "esse par já existe (em qualquer ordem)" {
			t.Fatalf("par repetido: %v", dup)
		}
	}
	// never to itself
	self := ana.expect("POST", "/_ge/api/contas/1/vinculos", map[string]any{"vinculada_id": 1}, 400)
	if m, _ := self["message"].(map[string]any); m == nil || m["vinculada"].([]any)[0] != "não pode ligar um registro a ele mesmo" {
		t.Fatalf("vínculo consigo mesmo: %v", self)
	}
	// editing cannot sneak in a repeated pair or a self link either
	v := ana.expect("POST", "/_ge/api/contas/1/vinculos", map[string]any{"vinculada_id": 3}, 201)
	vid := itoa(int(v["id"].(float64)))
	ana.expect("PATCH", "/_ge/api/contas/1/vinculos/"+vid, map[string]any{"vinculada_id": 2}, 409)
	ana.expect("PATCH", "/_ge/api/contas/1/vinculos/"+vid, map[string]any{"vinculada_id": 1}, 400)
	if n := rowsWhere(t, app, `SELECT COUNT(*) FROM vinculo`); n != 2 {
		t.Fatalf("nada repetido foi gravado: %d vínculos", n)
	}
	// the database keeps the pair too: a write that skipped the check fails
	if _, err := app.DB.DB.Exec(`INSERT INTO vinculo (conta_id, vinculada_id) VALUES (2, 1)`); err == nil {
		t.Fatal("o banco aceitou o mesmo par na ordem inversa")
	}
	if _, err := app.DB.DB.Exec(`INSERT INTO vinculo (conta_id, vinculada_id) VALUES (2, 3)`); err != nil {
		t.Fatalf("outro par é aceito pelo banco: %v", err)
	}
}

func rowsWhere(t *testing.T, app *App, query string) int {
	t.Helper()
	var n int
	if err := app.DB.DB.QueryRow(query).Scan(&n); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return n
}

// concurrently sends n identical requests at once with c's session and
// returns their statuses (the test client fails the test on the first error,
// which cannot happen outside the test's goroutine).
func concurrently(t *testing.T, c *client, n int, method, path string) []int {
	t.Helper()
	var wg sync.WaitGroup
	var mu sync.Mutex
	var out []int
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			req, _ := http.NewRequest(method, c.base+path, bytes.NewReader([]byte("{}")))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-CSRF-Token", c.csrf)
			resp, err := c.http.Do(req)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				out = append(out, -1)
				return
			}
			resp.Body.Close()
			out = append(out, resp.StatusCode)
		}()
	}
	close(start)
	wg.Wait()
	return out
}
