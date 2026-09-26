package germanio

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flaviokalleu/germanio/compiler/semantic"
)

func run(t *testing.T, source, input string) (string, error) {
	t.Helper()
	m, err := semantic.CheckSource("test.ge", source)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	err = (&Engine{Input: strings.NewReader(input), Output: &out, MaxSteps: 10000}).Run(m)
	return out.String(), err
}
func TestExecution(t *testing.T) {
	for _, tc := range []struct{ source, input, want string }{
		{`mostre "Olá mundo"`, "", "Olá mundo\n"},
		{"nome = pergunte \"Qual seu nome?\"\nmostre \"Olá {nome}\"", "Flavio\n", "Qual seu nome? Olá Flavio\n"},
		{"dobro(x) = x * 2\nmostre dobro(21)", "", "42\n"},
		{"somar(a, b)\n  a + b\nmostre somar(2, 3)", "", "5\n"},
		{"mut total = 0\npara item em [1, 2, 3]\n  total += item\nmostre total", "", "6\n"},
		{"variavel total = 0\npara item em [1, 2, 3]\n  total += item\nmostre total", "", "6\n"},
		{"mut i = 0\nenquanto i < 3\n  i += 1\n  se i == 2\n    continue\n  mostre i", "", "1\n3\n"},
		{"mut i = 0\nenquanto verdadeiro\n  i += 1\n  se i == 2\n    pare\nmostre i", "", "2\n"},
		{"f(n: inteiro) -> inteiro\n  se n == 0\n    retorne 1\n  retorne n * f(n - 1)\nmostre f(5)", "", "120\n"},
		{`mostre numero("5") + 2`, "", "7\n"},
		{`mostre quantidade("Olá 🐱")`, "", "5\n"},
		{"x: texto? = nulo\nmostre x == nulo", "", "verdadeiro\n"},
		{"nomes: [texto?] = [nulo, \"Ada\"]\nmostre nomes[0] == nulo\nmostre nomes[1]", "", "verdadeiro\nAda\n"},
		{"saudar(nome: texto?) -> texto\n  se nome != nulo\n    retorne nome + \"!\"\n  retorne \"sem nome\"\nmostre saudar(\"Ada\")\nmostre saudar(nulo)", "", "Ada!\nsem nome\n"},
		{"nome: texto? = \"Ada\"\nse nome == nulo\n  mostre \"vazio\"\nsenao\n  mostre nome + \"!\"", "", "Ada!\n"},
		{"nome: texto? = \"Ada\"\nse nome != nulo e quantidade(nome) > 0\n  mostre nome + \"!\"", "", "Ada!\n"},
		{"nome: texto? = nulo\nse nome != nulo e quantidade(nome) > 0\n  mostre \"inacessível\"\nsenao\n  mostre \"vazio\"", "", "vazio\n"},
		{"nome: texto? = \"Ada\"\nse nome == nulo ou quantidade(nome) > 0\n  mostre \"ok\"", "", "ok\n"},
		{"nome: texto? = nulo\nse nome == nulo ou quantidade(nome) > 0\n  mostre \"ok\"", "", "ok\n"},
		{"valores = [10, 20]\nmostre valores[1]", "", "20\n"},
		{`mostre "verdadeiro"`, "", "verdadeiro\n"},
		{`mostre "["`, "", "[\n"},
		{"zero = 0\nmostre falso e (1 / zero > 0)", "", "falso\n"},
		{"mostre 9007199254740993 > 9007199254740992", "", "verdadeiro\n"},
	} {
		t.Run(tc.source, func(t *testing.T) {
			got, err := run(t, tc.source, tc.input)
			if err != nil || got != tc.want {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}
func TestRuntimeDiagnostics(t *testing.T) {
	for _, tc := range []struct{ source, code string }{
		{"zero = 0\nmostre 1 / zero", "GE2005"},
		{"mostre 9223372036854775807 + 1", "GE2005"},
		{"mostre [1][4]", "GE2006"},
		{"mostre [1][-1]", "GE2006"},
		{`mostre numero("abc")`, "GE2004"},
		{`mostre numero("NaN")`, "GE2005"},
		{`mostre inteiro(1.2)`, "GE2005"},
		{"enquanto verdadeiro\n  mostre 1", "GE9001"},
		{"f() = f()\nmostre f()", "GE9001"},
	} {
		t.Run(tc.source, func(t *testing.T) {
			_, err := run(t, tc.source, "")
			if err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("want %s, got %v", tc.code, err)
			}
		})
	}
}
func TestNaturalUIEscapes(t *testing.T) {
	got, err := run(t, "crie botão escrito \"<script>alert(1)</script>\" azul\n  borda arredondada 12px", "")
	if err != nil || strings.Contains(got, "<script>") || !strings.Contains(got, "&lt;script&gt;") || !strings.Contains(got, "12px") {
		t.Fatalf("%s %v", got, err)
	}
}
func TestModuleFunctionsDoNotRunImportedScripts(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "math.ge"), []byte("mostre \"não executar\"\ndobro(x) = x * 2\n"), 0600)
	os.WriteFile(filepath.Join(dir, "main.ge"), []byte("usa \"math.ge\"\nmostre math.dobro(4)\n"), 0600)
	m, err := semantic.Load(filepath.Join(dir, "main.ge"))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err = (&Engine{Output: &out}).Run(m)
	if err != nil || out.String() != "8\n" {
		t.Fatalf("%s %v", out.String(), err)
	}
}
