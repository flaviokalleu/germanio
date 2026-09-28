package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

// Uma criação e tudo o que seus hooks fazem ficam juntos ou não ficam.
func TestCriacaoETudoJuntoOuNada(t *testing.T) {
	app, c := loadApp(t, "testdata/intencao/transacao.ge")
	c.expect("POST", "/_ge/api/lojas", map[string]any{"nome": "Centro"}, 201)
	count := func(table string) int {
		var n int
		if err := app.DB.DB.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	c.expect("POST", "/_ge/api/lojas/1/pedidos", map[string]any{"cliente": "recusado"}, 400)
	if p, i, a, tk := count("pedido"), count("item"), count("aviso"), count("_germanio_tarefas"); p+i+a+tk != 0 {
		t.Fatalf("recusado deixou restos: pedidos=%d itens=%d avisos=%d tarefas=%d", p, i, a, tk)
	}

	ok := c.expect("POST", "/_ge/api/lojas/1/pedidos", map[string]any{"cliente": "Ana"}, 201)
	if ok["numero"] != float64(1) {
		t.Fatalf("a numeração da tentativa desfeita não pode ser consumida: %v", ok["numero"])
	}
	if p, i, a := count("pedido"), count("item"), count("aviso"); p != 1 || i != 1 || a != 1 {
		t.Fatalf("pedidos=%d itens=%d avisos=%d", p, i, a)
	}
}

// O que foi feito fora do banco (repositório no disco) também é desfeito.
func TestCriacaoDesfeitaRemoveRepositorio(t *testing.T) {
	_, c := loadApp(t, "testdata/intencao/transacao.ge")
	c.expect("POST", "/_ge/api/lojas", map[string]any{"nome": "falha"}, 400)
	var repos []string
	filepath.WalkDir(os.Getenv("GERMANIO_GIT_RAIZ"), func(p string, d os.DirEntry, err error) error {
		if err == nil && d.IsDir() && filepath.Ext(p) == ".git" {
			repos = append(repos, p)
		}
		return nil
	})
	if len(repos) != 0 {
		t.Fatalf("repositório de criação desfeita ficou no disco: %v", repos)
	}
	c.expect("POST", "/_ge/api/lojas", map[string]any{"nome": "Centro"}, 201)
	c.expect("GET", "/_ge/api/lojas/1/repositorio/branches", nil, 200)
}
