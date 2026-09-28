package runtime

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A program that does not compile never replaces the running one, and data
// folders are not scanned.
func TestRecargaSoComProgramaValido(t *testing.T) {
	dir := t.TempDir()
	app := filepath.Join(dir, "app.ge")
	os.WriteFile(app, []byte("crie sistema Teste\n\ntenha notas\n"), 0o644)
	if err := reloadable(app); err != nil {
		t.Fatalf("programa válido recusado: %v", err)
	}
	os.WriteFile(app, []byte("crie sistema Teste\n\nnotas\n    tme\n        titulo\n"), 0o644)
	if err := reloadable(app); err == nil {
		t.Fatal("um programa com erro seria recarregado")
	}

	os.MkdirAll(filepath.Join(dir, "repositorios", "x"), 0o755)
	os.WriteFile(filepath.Join(dir, "repositorios", "x", "outro.ge"), []byte("x"), 0o644)
	stamps := map[string]time.Time{}
	scanChanged(dir, stamps)
	if _, ok := stamps[filepath.Join(dir, "repositorios", "x", "outro.ge")]; ok {
		t.Fatal("a varredura entrou numa pasta de dados")
	}
	t.Setenv("GERMANIO_PRODUCAO", "1")
	if hotReloadOn() {
		t.Fatal("recarga ligada em produção")
	}
}
