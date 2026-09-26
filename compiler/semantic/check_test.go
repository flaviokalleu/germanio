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
		"variavel contador = 0\ncontador += 1\nmostre contador",
		"variavel nome: texto? = nulo\nnome = \"Ana\"\nmostre nome",
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
		"variavel nome: texto? = \"Ada\"\nse nome != nulo\n  mostre nome + \"!\"",
		"variavel nome: texto? = nulo\nse nome == nulo\n  mostre \"ausente\"\nsenao\n  mostre nome + \"!\"",
		"variavel nome: texto? = \"Ada\"\nse nome != nulo e quantidade(nome) > 0\n  mostre nome + \"!\"",
		"variavel nome: texto? = \"Ada\"\nse nome != nulo\n  mostre nome + \"!\"\n  nome = nulo\nmostre nome == nulo",
		"variavel nome: texto? = \"Ada\"\nse nome != nulo\n  nome = nome + \"!\"\n  mostre nome == nulo",
		"nome: texto? = \"Ada\"\nse nulo == nome\n  mostre \"vazio\"\nsenao\n  mostre nome + \"!\"",
		"nome: texto? = \"Ada\"\nse nome != nulo e quantidade(nome) > 0\n  mostre nome + \"!\"",
		"nome: texto? = \"Ada\"\nse nome == nulo ou quantidade(nome) > 0\n  mostre \"ok\"\nsenao\n  mostre nome + \"!\"",
		"nome: texto? = \"Ada\"\nse nome != nulo e nome != nulo\n  mostre nome + \"!\"",
		"nome: texto? = \"Ada\"\nse nome == nulo ou nome == nulo\n  mostre \"vazio\"\nsenao\n  mostre nome + \"!\"",
		"nome: texto? = \"Ada\"\nenquanto nome != nulo e falso\n  mostre nome + \"!\"\nmostre nome",
		"nome: texto? = \"Ada\"\nse nome != nulo e quantidade(nome) > 0\n  mostre nome + \"!\"\nsenao\n  mostre \"vazio\"",
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
		{"variavel contador = 0\nvariavel contador = 1\nmostre contador", "GE2001"},
		{"variavel nome = 0\nnome = \"x\"\nmostre nome", "GE2004"},
		{"variavel = 1\nmostre variavel", "GE1001"},
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
		{"variavel nome: texto? = \"Ada\"\nse nome != nulo\n  mostre nome + \"!\"\nmostre nome + \"!\"", "GE2004"},
		{"variavel nome: texto? = \"Ada\"\nse nome != nulo\n  nome = nulo\n  mostre nome + \"!\"", "GE2004"},
		{"variavel nome: texto? = \"Ada\"\nse nome != nulo\n  mostre nome + \"!\"\n  nome = nulo\n  mostre nome + \"!\"", "GE2004"},
		{"variavel nome: texto? = \"Ada\"\nse nome != nulo\n  se verdadeiro\n    nome = nulo\n  mostre nome + \"!\"", "GE2004"},
		{"variavel nome: texto? = \"Ada\"\nenquanto nome != nulo\n  nome = nulo\n  mostre nome + \"!\"", "GE2004"},
		{"nome: texto? = \"Ada\"\nse nome == nulo\n  mostre nome + \"!\"", "GE2004"},
		{"nome: texto? = \"Ada\"\nse nome != nulo\n  mostre nome + \"!\"\nsenao\n  mostre nome + \"!\"", "GE2004"},
		{"nome: texto? = nulo\nse nome != nulo ou quantidade(nome) > 0\n  mostre \"erro\"", "GE2004"},
		{"nome: texto? = nulo\nse nome == nulo e quantidade(nome) > 0\n  mostre \"erro\"", "GE2004"},
		{"nome: texto? = nulo\nse nome != nulo ou verdadeiro\n  mostre nome + \"!\"", "GE2004"},
		{"mut nome: texto? = \"Ada\"\nse nome != nulo e quantidade(nome) > 0\n  nome = nulo\n  mostre nome + \"!\"", "GE2004"},
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

func TestOptionalDiagnosticPointsToOperand(t *testing.T) {
	src := "nome: texto? = nulo\nmostre nome + \"!\""
	_, err := CheckSource("opcional.ge", src)
	if err == nil {
		t.Fatal("expected optional operand error")
	}
	for _, part := range []string{"GE2004", "opcional.ge:2:8", "pode ser nulo", "se valor != nulo", "Exemplo:\nvalor: texto?"} {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("missing %q: %v", part, err)
		}
	}
}

func TestMutableGuardExplainsInvalidation(t *testing.T) {
	src := "variavel nome: texto? = \"Ada\"\nse nome != nulo\n  nome = nulo\n  mostre nome + \"!\""
	_, err := CheckSource("ramo.ge", src)
	if err == nil || !strings.Contains(err.Error(), "ramo.ge:4:10") || !strings.Contains(err.Error(), "evite reatribuir") {
		t.Fatalf("expected an explanation at the unsafe read: %v", err)
	}
}

func TestMutableOptionalJoins(t *testing.T) {
	for _, source := range []string{
		"variavel nome: texto? = nulo\nse verdadeiro\n  nome = \"Ada\"\nsenao\n  nome = \"Bia\"\nmostre nome + \"!\"",
		"variavel nome: texto? = nulo\nnome = \"Ada\"\nmostre nome + \"!\"",
		"f(flag: bool) -> texto\n  variavel nome: texto? = nulo\n  se flag\n    nome = \"Ada\"\n  senao\n    retorne \"fora\"\n  retorne nome + \"!\"\nmostre f(verdadeiro)",
	} {
		if _, err := CheckSource("join.ge", source); err != nil {
			t.Errorf("expected valid program: %v", err)
		}
	}
	for _, source := range []string{
		"variavel nome: texto? = nulo\nse verdadeiro\n  nome = \"Ada\"\nmostre nome + \"!\"",
		"variavel nome: texto? = \"Ada\"\nse verdadeiro\n  nome = nulo\nmostre nome + \"!\"",
		"variavel nome: texto? = nulo\nse verdadeiro\n  nome = \"Ada\"\nsenao\n  nome = nulo\nmostre nome + \"!\"",
		"variavel nome: texto? = \"Ada\"\nvariavel i = 0\nenquanto i < 1\n  i += 1\n  nome = nulo\nmostre nome + \"!\"",
		"variavel nome: texto? = \"Ada\"\nse nome != nulo\n  se verdadeiro\n    nome = nulo\n  mostre nome + \"!\"",
	} {
		if _, err := CheckSource("join.ge", source); err == nil || !strings.Contains(err.Error(), "GE2004") {
			t.Errorf("expected optional diagnostic, got %v", err)
		}
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
func TestPrivateModuleFunction(t *testing.T) {
	dir := t.TempDir()
	write := func(name, src string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("math.ge", "privado dobrar(x: inteiro) -> inteiro\n  x * 2\npublico(x: inteiro) = dobrar(x)")
	write("main.ge", "usa \"math.ge\"\nmostre math.publico(3)")
	m, err := Load(filepath.Join(dir, "main.ge"))
	if err != nil {
		t.Fatal(err)
	}
	if err := Check(m); err != nil {
		t.Fatal(err)
	}
	write("main.ge", "usa \"math.ge\"\nmostre math.dobrar(3)")
	m, err = Load(filepath.Join(dir, "main.ge"))
	if err != nil {
		t.Fatal(err)
	}
	if err = Check(m); err == nil || !strings.Contains(err.Error(), "GE3002") || !strings.Contains(err.Error(), "main.ge:2:8") || !strings.Contains(err.Error(), "Exemplo:") {
		t.Fatalf("expected educational private access diagnostic: %v", err)
	}
}
func TestGenericCalls(t *testing.T) {
	for _, source := range []string{
		"identidade<T>(valor: T) -> T\n  valor\nmostre identidade(1)\nmostre identidade(\"Ada\")",
		"primeiro<T>(valores: [T]) -> T\n  valores[0]\nmostre primeiro([1, 2])\nmostre primeiro([\"A\"])",
		"igual<T>(a: T, b: T) -> bool\n  a == b\nmostre igual(1, 2)",
		"id<T>(x: T) = x\nrepetir<U>(v: U) -> U\n  id(v)\nmostre repetir(\"sim\")",
	} {
		if _, err := CheckSource("generico.ge", source); err != nil {
			t.Errorf("valid generic: %v", err)
		}
	}
	for _, source := range []string{
		"igual<T>(a: T, b: T) = a == b\nmostre igual(1, \"um\")",
		"id<T>(x: T) = x\nmostre id(nulo)",
		"id<T>(x: T) = x\nmostre id([])",
		"f<T>(x: inteiro) -> T\n  retorne x\nmostre f(1)",
		"f<T>(x: T) -> T\n  x + 1\nmostre f(2)",
	} {
		if _, err := CheckSource("generico.ge", source); err == nil {
			t.Errorf("invalid generic accepted: %s", source)
		}
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

func TestImportFailuresPointToImport(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "pasta.ge"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ imported, message string }{
		{`"faltando.ge"`, "Import não encontrado"},
		{`"../segredo.ge"`, "Import fora do projeto"},
		{`"pasta.ge"`, "Import precisa de arquivo"},
	} {
		t.Run(tc.imported, func(t *testing.T) {
			path := filepath.Join(dir, "inicio.ge")
			if err := os.WriteFile(path, []byte("# título\nusa "+tc.imported+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			_, err := Load(path)
			if err == nil {
				t.Fatal("expected import error")
			}
			for _, part := range []string{"GE3001", "inicio.ge:2:1", tc.message, "Por quê:", "Como corrigir:", "Exemplo:"} {
				if !strings.Contains(err.Error(), part) {
					t.Errorf("missing %q in %v", part, err)
				}
			}
		})
	}
}
func TestImportCyclePointsToClosingImport(t *testing.T) {
	dir := t.TempDir()
	for name, source := range map[string]string{
		"a.ge": "usa \"b.ge\"\nmostre 1\n",
		"b.ge": "usa \"c.ge\"\nmostre 2\n",
		"c.ge": "# ciclo\nusa \"a.ge\"\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	_, err := Load(filepath.Join(dir, "a.ge"))
	if err == nil {
		t.Fatal("expected import cycle error")
	}
	for _, part := range []string{"GE3001", "c.ge:2:1", "Dependência circular", "Por quê:", "Como corrigir:", "Exemplo:"} {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("missing %q in %v", part, err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "b.ge"), []byte("usa \"a.ge\""), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = Load(filepath.Join(dir, "a.ge"))
	if err == nil || !strings.Contains(err.Error(), "b.ge:1:1") {
		t.Fatalf("expected two-file cycle at b.ge import: %v", err)
	}
}
