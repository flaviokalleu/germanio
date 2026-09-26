package lexer

import (
	"strings"
	"testing"
)

func TestGermanioTokensAndLegacyIsolation(t *testing.T) {
	tokens, err := NewGermanio("app.ge", "mut contador: inteiro = 0\ncontador += 1\nsomar(a) -> inteiro\n  a\ntelefone: texto? = nulo\n").Tokenize()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[TokenType]bool{}
	for _, tok := range tokens {
		seen[tok.Type] = true
		if tok.Line <= 0 || tok.Column <= 0 {
			t.Fatalf("missing position: %v", tok)
		}
	}
	for _, tt := range []TokenType{TokenPlusAssign, TokenArrow, TokenQuestion, TokenIndent} {
		if !seen[tt] {
			t.Errorf("missing token %v", tt)
		}
	}
	modern, _ := NewGermanio("a.ge", "set Print sistema").Tokenize()
	if modern[0].Type != TokenIdentifier || modern[1].Value != "Print" {
		t.Fatal("legacy translations leaked")
	}
	old, _ := New("set x = 1").Tokenize()
	if old[0].Type != TokenDefinir {
		t.Fatal("legacy broken")
	}
}
func TestGermanioLexicalErrors(t *testing.T) {
	for _, src := range []string{`mostre "ab`, "mostre 1.2.3", `mostre "\q"`, "mostre @"} {
		t.Run(src, func(t *testing.T) {
			_, err := NewGermanio("bad.ge", src).Tokenize()
			if err == nil || !strings.Contains(err.Error(), "GE1001") {
				t.Fatalf("expected diagnostic, got %v", err)
			}
		})
	}
}
