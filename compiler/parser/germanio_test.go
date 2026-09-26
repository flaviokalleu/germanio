package parser

import (
	"reflect"
	"strings"
	"testing"
)

func TestGermanioBoundsAndSuggestion(t *testing.T) {
	if _, err := ParseGermanio("min.ge", "mostre -9223372036854775808"); err != nil {
		t.Fatal(err)
	}
	_, err := ParseGermanio("typo.ge", `moste "Olá"`)
	if err == nil || !strings.Contains(err.Error(), "Você quis dizer mostre?") {
		t.Fatal(err)
	}
	_, err = ParseGermanio("deep.ge", "mostre "+strings.Repeat("(", 600)+"1"+strings.Repeat(")", 600))
	if err == nil || !strings.Contains(err.Error(), "GE1001") {
		t.Fatal(err)
	}
}

func TestGermanioAST(t *testing.T) {
	source := "somar(a: inteiro, b: inteiro) -> inteiro\n  a + b\nmut total = 0\npara item em [1, 2]\n  total += item\nmostre somar(total, 4)\n"
	p, err := ParseGermanio("a.ge", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Functions) != 1 || len(p.Scripts) != 3 || p.Functions[0].Body[0].Type != "return" {
		t.Fatalf("wrong AST: %+v", p)
	}
	if p.Scripts[0].Pos.Line != 3 || !p.Scripts[0].VarDecl.Mutable {
		t.Fatal("missing metadata")
	}
	again, err := ParseGermanio("a.ge", source)
	if err != nil || !reflect.DeepEqual(p, again) {
		t.Fatal("non deterministic parse")
	}
}
func TestMutableSpellingPreservesAST(t *testing.T) {
	for _, keyword := range []string{"mut", "variavel"} {
		program, err := ParseGermanio("a.ge", keyword+" contador = 0\ncontador += 1\nmostre contador")
		if err != nil {
			t.Fatal(err)
		}
		if len(program.Scripts) != 3 || !program.Scripts[0].VarDecl.Mutable {
			t.Fatalf("%s: expected mutable declaration", keyword)
		}
	}
}
func TestGermanioParserInvalid(t *testing.T) {
	for _, src := range []string{"se verdadeiro\nmostre 1", "  mostre 1", "se verdadeiro\n\tmostre 1", "mostre (1 + 2", "mostre [1 2]", "mostre [1,]", "somar(a,) = a", "mostre 1 lixo", "idade: banana = 1", "mut", "variavel", "senao\n  mostre 1", `mostre "Olá {nome"`, "mostre 9223372036854775808"} {
		t.Run(src, func(t *testing.T) {
			if _, err := ParseGermanio("bad.ge", src); err == nil {
				t.Fatal("invalid source accepted")
			}
		})
	}
}
func TestGermanioStringPunctuation(t *testing.T) {
	for _, src := range []string{`mostre "("`, `mostre ")"`, `mostre "["`, `mostre "verdadeiro"`, `mostre "nulo"`, `mostre "pergunte"`, `mostre texto(")")`} {
		if _, err := ParseGermanio("a.ge", src); err != nil {
			t.Fatalf("%s: %v", src, err)
		}
	}
}
func FuzzGermanioParser(f *testing.F) {
	for _, s := range []string{"mostre 1", "dobro(x) = x * 2", "mostre [1, 2]", "se verdadeiro\n  mostre 1"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > 16384 {
			t.Skip()
		}
		_, _ = ParseGermanio("fuzz.ge", s)
	})
}
