package runtime

import (
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// The pages in a real browser: dragging a card, another tab following it
// without reloading, and recovering what was missed after going offline.
// Runs when GERMANIO_NAVEGADOR (a Chromium executable) and
// GERMANIO_NAVEGADOR_MODULOS (a node_modules with playwright-core) are set.
func TestNavegador(t *testing.T) {
	exe, mods := os.Getenv("GERMANIO_NAVEGADOR"), os.Getenv("GERMANIO_NAVEGADOR_MODULOS")
	if exe == "" || mods == "" {
		t.Skip("GERMANIO_NAVEGADOR e GERMANIO_NAVEGADOR_MODULOS não definidos")
	}
	t.Setenv("GERMANIO_SQLITE", filepath.Join(t.TempDir(), "app.db"))
	t.Setenv("GERMANIO_BCRYPT_RAPIDO", "1")
	t.Setenv("GERMANIO_ARQUIVOS", t.TempDir())
	app, err := Carregar("testdata/por_estado/app.ge", "0")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(app.Handler)
	defer func() { srv.Close(); app.Fechar() }()
	script, _ := filepath.Abs("testdata/navegador/quadro.js")
	cmd := exec.Command("node", script, srv.URL, exe)
	cmd.Env = append(os.Environ(), "NODE_PATH="+mods)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("navegador: %v\n%s", err, out)
	}
}

// The forms of actions that ask something (merge, copy, move) in a real
// browser: they work by clicking, and axe-core finds no serious or critical
// violation on the pages that carry them. Same requirements as TestNavegador.
func TestNavegadorAcoes(t *testing.T) {
	exe, mods := os.Getenv("GERMANIO_NAVEGADOR"), os.Getenv("GERMANIO_NAVEGADOR_MODULOS")
	if exe == "" || mods == "" {
		t.Skip("GERMANIO_NAVEGADOR e GERMANIO_NAVEGADOR_MODULOS não definidos")
	}
	_, anon := loadApp(t, "testdata/paginas_acoes/app.ge")
	ana := signIn(t, anon.base, "Ana", "ana@x.com")
	bia := signIn(t, anon.base, "Bia", "bia@x.com")
	ana.expect("POST", "/_ge/api/livros", map[string]any{"nome": "atlas"}, 201)
	ana.expect("POST", "/_ge/api/livros", map[string]any{"nome": "mapa"}, 201)
	ana.expect("POST", "/_ge/api/livros/1/membros", map[string]any{"pessoa_id": bia.expect("GET", "/_ge/eu", nil, 200)["id"], "papel": "revisor"}, 201)
	put := func(branch, path, content string) {
		ana.expect("PUT", "/_ge/api/livros/1/repositorio/arquivos/"+path, map[string]any{"branch": branch, "conteudo": content, "mensagem": path}, 200)
	}
	put("main", "compilar.yml", "etapas:\n  testar:\n    comandos: [\"true\"]\n")
	ana.expect("POST", "/_ge/api/livros/1/repositorio/branches", map[string]any{"nome": "rev1", "origem": "main"}, 201)
	put("rev1", "cap2.txt", "dois\n")
	bia.expect("POST", "/_ge/api/livros/1/revisoes", map[string]any{"titulo": "Cap 2", "origem": "rev1", "destino": "main"}, 201)
	ana.expect("POST", "/_ge/api/livros/1/revisoes/1/aprovar", nil, 200)
	ana.expect("POST", "/_ge/api/livros/1/tarefas", map[string]any{"titulo": "Revisar"}, 201)
	script, _ := filepath.Abs("testdata/navegador/acoes.js")
	cmd := exec.Command("node", script, anon.base, exe)
	cmd.Env = append(os.Environ(), "NODE_PATH="+mods)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("navegador: %v\n%s", err, out)
	}
}
