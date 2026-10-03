package runtime

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// A sum that never goes below zero (GEP 0050, em teste) in a domain with no
// Git and no GitLab: the stock of a product is the sum of its movements and
// never goes negative. The rule holds for creating, editing and deleting
// movements, and under concurrent requests (each subtraction sees the ones
// before it, with the product locked); deleting the product itself is not
// refused.

func postAll(t *testing.T, c *client, n int, path string, body any) []int {
	t.Helper()
	raw, _ := json.Marshal(body)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var out []int
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			req, _ := http.NewRequest("POST", c.base+path, bytes.NewReader(raw))
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

func stockOf(t *testing.T, c *client, id string) float64 {
	t.Helper()
	m := c.expect("GET", "/_ge/api/produtos/"+id, nil, 200)
	v, _ := m["estoque"].(float64)
	return v
}

func TestSomaNuncaNegativa(t *testing.T) {
	app, anon := loadApp(t, "testdata/estoque/app.ge")
	ana := signIn(t, anon.base, "Ana", "ana@x.com")
	ana.expect("POST", "/_ge/api/produtos", map[string]any{"nome": "Parafuso"}, 201)
	mov := "/_ge/api/produtos/1/movimentos"

	ana.expect("POST", mov, map[string]any{"quantidade": 10, "motivo": "compra"}, 201)
	ana.expect("POST", mov, map[string]any{"quantidade": -4, "motivo": "venda"}, 201)
	// taking more than there is: refused on the field, with the reason
	refused := ana.expect("POST", mov, map[string]any{"quantidade": -7}, 400)
	if m, _ := refused["message"].(map[string]any); m == nil || !strings.Contains(toStrT(m["quantidade"].([]any)[0]), "estoque") {
		t.Fatalf("recusa: %v", refused)
	}
	if s := stockOf(t, ana, "1"); s != 6 {
		t.Fatalf("estoque depois da recusa: %v", s)
	}
	// editing and deleting follow the same rule
	ana.expect("PATCH", mov+"/1", map[string]any{"quantidade": 3}, 400) // 3 - 4 < 0
	ana.expect("DELETE", mov+"/1", nil, 400)                            // only -4 would remain
	ana.expect("PATCH", mov+"/1", map[string]any{"quantidade": 9}, 200)
	if s := stockOf(t, ana, "1"); s != 5 {
		t.Fatalf("estoque depois de editar: %v", s)
	}

	// eight subtractions at once with five in stock: exactly five pass
	statuses := postAll(t, ana, 8, mov, map[string]any{"quantidade": -1})
	ok, no := 0, 0
	for _, s := range statuses {
		switch s {
		case 201:
			ok++
		case 400:
			no++
		default:
			t.Fatalf("subtrações ao mesmo tempo: %v", statuses)
		}
	}
	if ok != 5 || no != 3 {
		t.Fatalf("com 5 em estoque, 8 saídas de 1 ao mesmo tempo: %d passaram, %d recusadas (%v)", ok, no, statuses)
	}
	if s := stockOf(t, ana, "1"); s != 0 {
		t.Fatalf("estoque final: %v", s)
	}
	var sum int
	app.DB.DB.QueryRow(`SELECT COALESCE(SUM(quantidade), 0) FROM movimento WHERE produto_id = 1`).Scan(&sum)
	if sum != 0 {
		t.Fatalf("a soma no banco ficou %d", sum)
	}

	// deleting the product takes its movements, whatever their order
	ana.expect("POST", mov, map[string]any{"quantidade": 2}, 201)
	ana.expect("DELETE", "/_ge/api/produtos/1", nil, 204)
	if n := rowsWhere(t, app, `SELECT COUNT(*) FROM movimento`); n != 0 {
		t.Fatalf("os movimentos saem com o produto: %d", n)
	}
}
