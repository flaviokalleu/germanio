package gecli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	germanioRuntime "github.com/flaviokalleu/germanio/runtime"
	"github.com/flaviokalleu/germanio/runtime/banco"
)

func invoke(args ...string) (int, string, string) {
	var out, errs bytes.Buffer
	code := Run(append([]string{"ge"}, args...), strings.NewReader("Flavio\n"), &out, &errs)
	return code, out.String(), errs.String()
}
func TestCLILifecycle(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "program")
	if code, _, err := invoke("novo", dir); code != 0 {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "inicio.ge")
	if code, out, err := invoke("rodar", file); code != 0 || out != "Qual seu nome? Olá Flavio\n" {
		t.Fatalf("%s %s", out, err)
	}
	if code, out, err := invoke("check", file); code != 0 || strings.Contains(out, "Qual seu nome?") {
		t.Fatalf("check executed input: %s %s", out, err)
	}
	if code, _, _ := invoke("novo", dir); code == 0 {
		t.Fatal("overwrote existing directory")
	}
	if code, out, err := invoke("explicar", "GE2004"); code != 0 || !strings.Contains(out, "Tipos incompatíveis") {
		t.Fatalf("%s %s", out, err)
	}
}
func TestCLIInvalidAndFormatting(t *testing.T) {
	file := filepath.Join(t.TempDir(), "a.ge")
	os.WriteFile(file, []byte("x=1\nmostre x+2"), 0600)
	before, _ := os.ReadFile(file)
	if code, _, _ := invoke("fmt", file, "--check"); code != 1 {
		t.Fatal("expected format diff")
	}
	after, _ := os.ReadFile(file)
	if !bytes.Equal(before, after) {
		t.Fatal("--check wrote file")
	}
	if code, _, err := invoke("fmt", file); code != 0 {
		t.Fatal(err)
	}
	if code, _, err := invoke("fmt", file, "--check"); code != 0 {
		t.Fatal(err)
	}
	if code, _, _ := invoke("check", file, "--future"); code == 0 {
		t.Fatal("silently accepted flag")
	}
	os.WriteFile(file, []byte("mostre \"5\" + 2"), 0600)
	if code, _, err := invoke("rodar", file); code != 1 || !strings.Contains(err, "GE2004") {
		t.Fatal(err)
	}
}
func TestInitCreatesContextualProjectAndCanExplainIt(t *testing.T) {
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)

	if code, out, err := invoke("init", "MeuCRM", "--yes"); code != 0 {
		t.Fatalf("init failed: %s %s", out, err)
	}
	if _, err := os.Stat(filepath.Join(project, ".germanio", "project.json")); err != nil {
		t.Fatalf("project manifest was not created: %v", err)
	}
	if code, out, err := invoke("init", "entidade", "Cliente", "--yes"); code != 0 {
		t.Fatalf("entity init failed: %s %s", out, err)
	}
	if code, out, err := invoke("init", "pagina", "clientes", "--yes"); code != 0 {
		t.Fatalf("page init failed: %s %s", out, err)
	}
	if code, out, err := invoke("graph"); code != 0 || !strings.Contains(out, "Cliente") {
		t.Fatalf("graph failed: %d %s %s", code, out, err)
	}
	if code, out, err := invoke("explain", "pagina", "clientes"); code != 0 || !strings.Contains(out, "Página:") {
		t.Fatalf("explain failed: %d %s %s", code, out, err)
	}
}

func TestInitFlagsExportFromAndModes(t *testing.T) {
	project := t.TempDir()
	oldWD, _ := os.Getwd()
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)
	if code, out, err := invoke("init", "--dry-run", "--yes", "--export", "spec.geinit"); code != 0 || !strings.Contains(out, "dry-run") || err != "" {
		t.Fatalf("dry-run: %d %s %s", code, out, err)
	}
	if _, err := os.Stat(filepath.Join(project, "spec.geinit")); !os.IsNotExist(err) {
		t.Fatal("dry-run wrote export")
	}
	if code, _, err := invoke("init", "--yes", "--export", "spec.geinit"); code != 0 {
		t.Fatalf("export: %s", err)
	}
	if _, err := os.Stat(filepath.Join(project, "spec.geinit")); err != nil {
		t.Fatal(err)
	}
	other := t.TempDir()
	old := project
	_ = old
	if err := os.Chdir(other); err != nil {
		t.Fatal(err)
	}
	if code, out, err := invoke("init", "--from", filepath.Join(project, "spec.geinit")); code != 0 || !strings.Contains(out, "importado") {
		t.Fatalf("from: %d %s %s", code, out, err)
	}
}

func TestContextualInitCommandsReportCapabilityState(t *testing.T) {
	project := t.TempDir()
	oldWD, _ := os.Getwd()
	defer os.Chdir(oldWD)
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"api", "componente", "auth", "pagamento", "chat", "busca", "upload", "notificacao"} {
		code, out, err := invoke("init", name)
		if code != 0 || !strings.Contains(out, "CAP_") || err != "" {
			t.Fatalf("%s: %d %s %s", name, code, out, err)
		}
	}
}

func TestNoUnsupportedCommandsClaimSuccess(t *testing.T) {
	for _, name := range []string{"build", "fuzz", "ai", "gpu", "wasm"} {
		if code, _, _ := invoke(name); code == 0 {
			t.Errorf("%s claimed success", name)
		}
	}
}
func TestCLINativeTests(t *testing.T) {
	dir := t.TempDir()
	write := func(name, source string) string {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	write("matematica.ge", "soma(a: inteiro, b: inteiro) = a + b\nnunca() = 99\n")
	path := write("matematica_teste.ge", "usa \"matematica.ge\"\nteste \"soma\"\n  espera matematica.soma(2, 2) == 4\n")
	if code, out, err := invoke("testar", dir); code != 0 || !strings.Contains(out, "1 teste(s) passaram") {
		t.Fatalf("directory test: %d %s %s", code, out, err)
	}
	if code, out, err := invoke("testar", path); code != 0 || !strings.Contains(out, "ok ") {
		t.Fatalf("file test: %d %s %s", code, out, err)
	}
	if code, out, err := invoke("testar", dir, "--coverage"); code != 0 || !strings.Contains(out, "Cobertura de instruções: 2/3 (66.7%)") || !strings.Contains(out, "Sem cobertura: "+filepath.Join(dir, "matematica.ge")+":2:") {
		t.Fatalf("expected coverage: %d %s %s", code, out, err)
	}
	if code, _, _ := invoke("testar", dir, "--race"); code == 0 {
		t.Fatal("claimed to support race detection")
	}
	write("matematica_teste.ge", "teste \"errado\"\n  espera 1 == 2\n")
	if code, _, err := invoke("testar", dir); code != 1 || !strings.Contains(err, "GE2008") {
		t.Fatalf("expected failed assertion: %d %s", code, err)
	}
	if code, _, err := invoke("testar", filepath.Join(dir, "matematica.ge")); code != 1 || !strings.Contains(err, "Nenhum teste") {
		t.Fatalf("expected no-tests diagnostic: %d %s", code, err)
	}
}

// `ge new` creates an application in the intent syntax that passes its own
// check and the formatter; `run` and `test` are the English names of rodar and
// testar.
func TestCLINewApplication(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "loja")
	if code, _, err := invoke("new", dir); code != 0 {
		t.Fatal(err)
	}
	app := filepath.Join(dir, "app.ge")
	if code, out, err := invoke("check", app); code != 0 || !strings.Contains(out, "verificado") {
		t.Fatalf("o projeto novo não passa no check: %s %s", out, err)
	}
	if code, out, err := invoke("fmt", dir, "--check"); code != 0 {
		t.Fatalf("o projeto novo não está na forma canônica: %s %s", out, err)
	}
	if code, _, _ := invoke("new", dir); code == 0 {
		t.Fatal("new sobrescreveu uma pasta existente")
	}
	if code, out, err := invoke("test", filepath.Join("..", "..", "examples", "germanio")); code != 0 {
		t.Fatalf("ge test: %s %s", out, err)
	}
}

// ge check reads the existing database without changing it: a probable
// rename stops with the educational message; the declared rename is announced
// (G93).
func TestCheckMigracaoDoBancoExistente(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "clientes.db")
	t.Setenv("GERMANIO_SQLITE", db)
	app := filepath.Join(dir, "app.ge")
	write := func(src string) {
		if err := os.WriteFile(app, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	head := "crie sistema Clientes\n\ntenha clientes\n\n"
	write(head + "cada cliente tem\n    nome\n")
	prog, err := germanioRuntime.Compilar(app)
	if err != nil {
		t.Fatal(err)
	}
	b, err := banco.Abrir(prog.Database, prog.System.Name, prog.Models)
	if err != nil {
		t.Fatal(err)
	}
	b.CriarMapa("cliente", map[string]any{"nome": "Ana"})
	b.Fechar()

	write(head + "cada cliente tem\n    nome_completo\n")
	code, out, errs := invoke("check", app)
	if code == 0 || !strings.Contains(errs, "renomeie nome para nome_completo") || !strings.Contains(errs, "não pode provar") {
		t.Fatalf("o check deveria recusar o rename provável: %d %s %s", code, out, errs)
	}
	write(head + "cada cliente tem\n    nome_completo\n\nrenomeie nome de clientes para nome_completo\n")
	code, out, errs = invoke("check", app)
	if code != 0 || !strings.Contains(out, "renomeia nome para nome_completo") {
		t.Fatalf("o check deveria anunciar o rename: %d %s %s", code, out, errs)
	}
	// nothing changed: the old column still holds the data
	b, err = banco.Abrir(prog.Database, prog.System.Name, prog.Models)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Fechar()
	var nome string
	if err := b.DB.QueryRow(`SELECT nome FROM cliente`).Scan(&nome); err != nil || nome != "Ana" {
		t.Fatalf("o check mudou o banco: %q %v", nome, err)
	}
	b.Fechar()
	// the start with the new program applies the rename, keeping the data
	prog, err = germanioRuntime.Compilar(app)
	if err != nil {
		t.Fatal(err)
	}
	b, err = banco.Abrir(prog.Database, prog.System.Name, prog.Models)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.DB.QueryRow(`SELECT nome_completo FROM cliente`).Scan(&nome); err != nil || nome != "Ana" {
		t.Fatalf("o rename declarado não preservou os dados: %q %v", nome, err)
	}
}
