package runtime

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/flaviokalleu/germanio/runtime/banco"
)

// Listas com visibilidade por registro: contagem e páginas certas mesmo com
// muito mais registros do que cabem em um lote.
func TestListaVisivelSemTeto(t *testing.T) {
	app, c := loadApp(t, "testdata/intencao/pastas.ge")
	c.expect("POST", "/cadastro", map[string]any{"nome": "Ana", "email": "ana@x.com", "senha": "senha-da-ana"}, 201)
	err := app.DB.EmTransacao(func(tx *banco.Banco) error {
		for i := 1; i <= 1200; i++ {
			vis := "public"
			if i%2 == 0 {
				vis = "private"
			}
			row, err := tx.CriarMapa("pasta", map[string]any{"nome": fmt.Sprintf("p%d", i), "visibilidade": vis})
			if err != nil {
				return err
			}
			if i%2 == 0 && i <= 10 { // Ana lê 5 pastas privadas
				if _, err := tx.CriarMapa("membro", map[string]any{"recurso": "pasta", "recurso_id": row["id"], "pessoa_id": 1, "papel": "leitor"}); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	total := func(hc *http.Client, query string) (int, int) {
		resp, err := hc.Get(c.base + "/_ge/api/pastas" + query)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var list []any
		json.NewDecoder(resp.Body).Decode(&list)
		if resp.StatusCode != 200 {
			t.Fatalf("GET %s = %d", query, resp.StatusCode)
		}
		n, _ := strconv.Atoi(resp.Header.Get("X-Total"))
		return n, len(list)
	}
	if n, got := total(http.DefaultClient, "?per_page=100&page=6"); n != 600 || got != 100 {
		t.Fatalf("anônimo: total=%d página=%d (esperado 600 e 100)", n, got)
	}
	c.expect("POST", "/entrar", map[string]any{"email": "ana@x.com", "senha": "senha-da-ana"}, 200)
	if n, got := total(c.http, "?per_page=100&page=7"); n != 605 || got != 5 {
		t.Fatalf("Ana: total=%d última página=%d (esperado 605 e 5)", n, got)
	}
	if n, got := total(c.http, "?per_page=100&page=8"); n != 605 || got != 0 {
		t.Fatalf("Ana depois do fim: total=%d página=%d", n, got)
	}
}
