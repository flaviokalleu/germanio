package gecli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
func TestNoUnsupportedCommandsClaimSuccess(t *testing.T) {
	for _, name := range []string{"build", "fuzz", "ai", "testar", "gpu", "wasm"} {
		if code, _, _ := invoke(name); code == 0 {
			t.Errorf("%s claimed success", name)
		}
	}
}
