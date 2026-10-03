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
