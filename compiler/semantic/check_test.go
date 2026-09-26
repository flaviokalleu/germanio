package semantic

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidPrograms(t *testing.T) {
	for _, src := range []string{
		`mostre "Olá"`,
		"nome = pergunte \"Qual seu nome?\"\nmostre \"Olá {nome}\"",
		"mut contador = 0\ncontador += 1\nmostre contador",
		"idade: inteiro = 30\nse idade >= 18\n  mostre \"Adulto\"\nsenao\n  mostre \"Menor\"",
		"dobro(x) = x * 2\nmostre dobro(3)",
		"somar(a, b)\n  a + b\nmostre somar(2, 3)",
		"f(n: inteiro) -> inteiro\n  se n == 0\n    retorne 1\n  retorne n * f(n - 1)\nmostre f(4)",
		"mut total = 0\npara item em [1, 2, 3]\n  total += item\nmostre total",
		"mut nome: texto? = nulo\nnome = \"Flavio\"\nmostre nome != nulo",
		"valores: [inteiro] = []\nmostre quantidade(valores)",
		"valores: [texto?] = [nulo, \"A\"]\nmostre valores[0] == nulo",
		"valores: [texto?] = [nulo]\nmostre valores[0] == nulo",
		"base: [texto] = [\"A\"]\nopcionais: [texto?] = base\nmostre opcionais[0]",
		"f(x: texto?) = x == nulo\nmostre f(nulo)",
		"saudar(nome: texto?) -> texto\n  se nome != nulo\n    retorne nome + \"!\"\n  retorne \"sem nome\"\nmostre saudar(\"Ada\")",
		"f(nome: texto?) -> texto\n  se nome == nulo\n    retorne \"vazio\"\n  retorne nome + \"!\"\nmostre f(\"Ada\")",
		"nome: texto? = \"Ada\"\nse nao (nulo == nome)\n  mostre nome + \"!\"",
		"nome: texto? = \"Ada\"\nse nulo == nome\n  mostre \"vazio\"\nsenao\n  mostre nome + \"!\"",
		"const PI = 3.14\nmostre PI",
		"_ = 42",
		"crie botão escrito \"Comprar\" azul\n  borda arredondada 16px",
		`mostre numero("5") + 2`,
		"mut i = 0\nenquanto i < 3\n  i += 1\n  se i == 2\n    continue\n  mostre i",
	} {
		t.Run(src, func(t *testing.T) {
			if _, err := CheckSource("ok.ge", src); err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestInvalidPrograms(t *testing.T) {
	cases := []struct{ source, code string }{
		{`mostre "5" + 2`, "GE2004"},
		{"nome = \"Flavio\"\nse nome\n  mostre nome", "GE2004"},
		{"x = 1\nx = 2\nmostre x", "GE2002"},
		{"mut x = 1\nx = \"x\"\nmostre x", "GE2004"},
		{"x = 1", "GE2003"},
		{`moste "Olá"`, "GE1001"},
		{"mostre faltante", "GE2001"},
		{"x: inteiro = \"1\"\nmostre x", "GE2004"},
		{"x = nulo\nmostre x", "GE2004"},
		{"x: texto = nulo\nmostre x", "GE2004"},
		{"valores = [nulo]\nmostre valores", "GE2004"},
		{"f(x) = x\nmostre f(nulo)", "GE2004"},
		{"mostre nulo == nulo", "GE2004"},
		{"mostre 1 == nulo", "GE2004"},
		{"valores: [texto] = [nulo, \"A\"]\nmostre valores", "GE2004"},
		{"x: inteiro? = nulo\nmostre x + 1", "GE2004"},
		{"nome: texto? = \"Ada\"\nse nome != nulo\n  mostre nome + \"!\"\nmostre nome + \"!\"", "GE2004"},
		{"mut nome: texto? = \"Ada\"\nse nome != nulo\n  mostre nome + \"!\"", "GE2004"},
		{"nome: texto? = \"Ada\"\nse nome == nulo\n  mostre nome + \"!\"", "GE2004"},
		{"nome: texto? = \"Ada\"\nse nome != nulo\n  mostre nome + \"!\"\nsenao\n  mostre nome + \"!\"", "GE2004"},
		{"mostre [1, \"2\"]", "GE2004"},
		{"mostre [1][\"0\"]", "GE2004"},
		{"mostre 10 / 0", "GE2005"},
		{"mostre nao 1", "GE2004"},
		{"mostre verdadeiro e 1", "GE2004"},
		{"dobro(x) = x * 2\nmostre dobro(\"2\")", "GE2004"},
		{"dobro(x) = x * 2\nmostre dobro()", "GE2004"},
		{"f(x) = 1\nmostre f(2)", "GE2003"},
		{"f(x,x) = x\nmostre f(1,2)", "GE2001"},
		{"f()\n  retorne 1\n  mostre 2\nmostre f()", "GE2007"},
		{"f(x: bool) -> inteiro\n  se x\n    retorne 1\nmostre f(verdadeiro)", "GE2004"},
		{"retorne 1", "GE1001"},
		{"pare", "GE1001"},
		{"continue", "GE1001"},
		{"mut x = 1\nse verdadeiro\n  mut x = 2\nmostre x", "GE2001"},
		{"crie botão azul\n  borda arredondada 45deg", "GE4102"},
		{"f() = 1\nf() = 2", "GE2001"},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			_, err := CheckSource("bad.ge", tc.source)
			if err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("want %s, got %v", tc.code, err)
			}
			for _, part := range []string{"bad.ge:", "Por quê:", "Como corrigir:", "Exemplo:"} {
				if !strings.Contains(err.Error(), part) {
					t.Errorf("missing %s: %v", part, err)
				}
			}
		})
	}
}

func TestLocalModules(t *testing.T) {
	dir := t.TempDir()
	write := func(name, source string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("math.ge", "dobro(x) = x * 2\n")
	write("main.ge", "usa \"math.ge\"\nmostre math.dobro(4)\n")
	m, err := Load(filepath.Join(dir, "main.ge"))
	if err != nil {
		t.Fatal(err)
	}
	if err = Check(m); err != nil {
		t.Fatal(err)
	}
	write("main.ge", "usa \"math.ge\"\nmostre 1\n")
	m, err = Load(filepath.Join(dir, "main.ge"))
	if err != nil {
		t.Fatal(err)
	}
	if err = Check(m); err == nil || !strings.Contains(err.Error(), "GE2003") {
		t.Fatal(err)
	}
	write("math.ge", "usa \"main.ge\"\n")
	_, err = Load(filepath.Join(dir, "main.ge"))
	if err == nil || !strings.Contains(err.Error(), "circular") {
		t.Fatal(err)
	}
}
func TestImportCannotEscapeProject(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "project")
	os.Mkdir(inside, 0700)
	os.WriteFile(filepath.Join(root, "secret.ge"), []byte("mostre 1"), 0600)
	os.WriteFile(filepath.Join(inside, "main.ge"), []byte("usa \"../secret.ge\""), 0600)
	_, err := Load(filepath.Join(inside, "main.ge"))
	if err == nil || !strings.Contains(err.Error(), "GE3001") {
		t.Fatal(err)
	}
}
