package lexer

import (
	"testing"
)

func TestBasicKeywords(t *testing.T) {
	l := New("sistema meuapp")
	tokens, err := l.Tokenize()
	if err != nil {
		t.Fatal(err)
	}
	if len(tokens) < 3 {
		t.Fatalf("expected at least 3 tokens, got %d", len(tokens))
	}
	if tokens[0].Type != TokenSistema {
		t.Errorf("expected TokenSistema, got %d", tokens[0].Type)
	}
	if tokens[1].Type != TokenIdentifier || tokens[1].Value != "meuapp" {
		t.Errorf("expected identifier 'meuapp', got type=%d value=%q", tokens[1].Type, tokens[1].Value)
	}
	if tokens[2].Type != TokenEOF {
		t.Errorf("expected TokenEOF, got %d", tokens[2].Type)
	}
}

func TestAllBlockKeywords(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"sistema", TokenSistema},
		{"dados", TokenDados},
		{"telas", TokenTelas},
		{"acoes", TokenAcoes},
		{"eventos", TokenEventos},
		{"integracoes", TokenIntegracoes},
		{"tema", TokenTema},
		{"logica", TokenLogica},
		{"banco", TokenBanco},
		{"autenticacao", TokenAutenticacao},
		{"config", TokenConfig},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := New(tt.input)
			tokens, err := l.Tokenize()
			if err != nil {
				t.Fatal(err)
			}
			if tokens[0].Type != tt.expected {
				t.Errorf("keyword %q: expected token type %d, got %d", tt.input, tt.expected, tokens[0].Type)
			}
		})
	}
}

func TestBilingualKeywords(t *testing.T) {
	pairs := []struct {
		pt, en   string
		expected TokenType
	}{
		{"sistema", "system", TokenSistema},
		{"dados", "models", TokenDados},
		{"telas", "screens", TokenTelas},
		{"eventos", "events", TokenEventos},
		{"tema", "theme", TokenTema},
		{"logica", "logic", TokenLogica},
		{"banco", "database", TokenBanco},
		{"autenticacao", "auth", TokenAutenticacao},
		{"importar", "import", TokenImportar},
		{"de", "from", TokenDe},
		{"se", "if", TokenSe},
		{"senao", "else", TokenSenao},
		{"funcao", "function", TokenFuncao},
		{"definir", "set", TokenDefinir},
		{"retornar", "return", TokenRetornar},
		{"verdadeiro", "true", TokenVerdadeiro},
		{"falso", "false", TokenFalso},
		{"nulo", "null", TokenNulo},
		{"pertence_a", "belongs_to", TokenPertenceA},
		{"tem_muitos", "has_many", TokenTemMuitos},
		{"muitos_para_muitos", "many_to_many", TokenMuitosParaMuitos},
	}

	for _, p := range pairs {
		t.Run(p.pt+"="+p.en, func(t *testing.T) {
			lPT := New(p.pt)
			tokPT, err := lPT.Tokenize()
			if err != nil {
				t.Fatal(err)
			}
			lEN := New(p.en)
			tokEN, err := lEN.Tokenize()
			if err != nil {
				t.Fatal(err)
			}
			if tokPT[0].Type != p.expected {
				t.Errorf("PT keyword %q: expected type %d, got %d", p.pt, p.expected, tokPT[0].Type)
			}
			if tokEN[0].Type != p.expected {
				t.Errorf("EN keyword %q: expected type %d, got %d", p.en, p.expected, tokEN[0].Type)
			}
		})
	}
}

func TestFieldTypes(t *testing.T) {
	types := []struct {
		input    string
		expected TokenType
	}{
		{"texto", TokenTexto},
		{"numero", TokenNumero},
		{"dinheiro", TokenDinheiro},
		{"email", TokenEmail},
		{"telefone", TokenTelefone},
		{"imagem", TokenImagem},
		{"arquivo", TokenArquivo},
		{"upload", TokenUpload},
		{"link", TokenLink},
		{"status", TokenStatus},
		{"senha", TokenSenha},
		{"data", TokenData},
		{"booleano", TokenBooleano},
		{"texto_longo", TokenTextoLongo},
		{"enum", TokenEnum},
	}

	for _, tt := range types {
		t.Run(tt.input, func(t *testing.T) {
			l := New(tt.input)
			tokens, err := l.Tokenize()
			if err != nil {
				t.Fatal(err)
			}
			if tokens[0].Type != tt.expected {
				t.Errorf("field type %q: expected %d, got %d", tt.input, tt.expected, tokens[0].Type)
			}
		})
	}
}

func TestRealtimeKeywords(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"chat", TokenChat},
		{"sessao", TokenSessao},
		{"qrcode", TokenQRCode},
		{"presenca", TokenPresenca},
		{"fila_jobs", TokenFilaJobs},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := New(tt.input)
			tokens, err := l.Tokenize()
			if err != nil {
				t.Fatal(err)
			}
			if tokens[0].Type != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, tokens[0].Type)
			}
		})
	}
}

func TestStringsAndNumbers(t *testing.T) {
	l := New(`"hello world" 42 3.14`)
	tokens, err := l.Tokenize()
	if err != nil {
		t.Fatal(err)
	}

	if tokens[0].Type != TokenString || tokens[0].Value != "hello world" {
		t.Errorf("expected string 'hello world', got type=%d value=%q", tokens[0].Type, tokens[0].Value)
	}
	if tokens[1].Type != TokenNumber || tokens[1].Value != "42" {
		t.Errorf("expected number '42', got type=%d value=%q", tokens[1].Type, tokens[1].Value)
	}
	if tokens[2].Type != TokenNumber || tokens[2].Value != "3.14" {
		t.Errorf("expected number '3.14', got type=%d value=%q", tokens[2].Type, tokens[2].Value)
	}
}

func TestStringEscapes(t *testing.T) {
	l := New(`"hello\nworld" "tab\there" "quote\"inside"`)
	tokens, err := l.Tokenize()
	if err != nil {
		t.Fatal(err)
	}

	if tokens[0].Value != "hello\nworld" {
		t.Errorf("expected newline escape, got %q", tokens[0].Value)
	}
	if tokens[1].Value != "tab\there" {
		t.Errorf("expected tab escape, got %q", tokens[1].Value)
	}
	if tokens[2].Value != `quote"inside` {
		t.Errorf("expected quote escape, got %q", tokens[2].Value)
	}
}

func TestOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
		value    string
	}{
		{"==", TokenEqualEqual, "=="},
		{"!=", TokenDiferente, "!="},
		{">", TokenMaiorQue, ">"},
		{"<", TokenMenorQue, "<"},
		{">=", TokenMaiorIgual, ">="},
		{"<=", TokenMenorIgual, "<="},
		{"+", TokenPlus, "+"},
		{"-", TokenMinus, "-"},
		{"*", TokenStar, "*"},
		{"/", TokenSlash, "/"},
		{"=", TokenEquals, "="},
		{"(", TokenLParen, "("},
		{")", TokenRParen, ")"},
		{"[", TokenLBracket, "["},
		{"]", TokenRBracket, "]"},
		{":", TokenColon, ":"},
		{".", TokenDot, "."},
		{",", TokenComma, ","},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := New(tt.input)
			tokens, err := l.Tokenize()
			if err != nil {
				t.Fatal(err)
			}
			if tokens[0].Type != tt.expected {
				t.Errorf("operator %q: expected type %d, got %d", tt.input, tt.expected, tokens[0].Type)
			}
			if tokens[0].Value != tt.value {
				t.Errorf("operator %q: expected value %q, got %q", tt.input, tt.value, tokens[0].Value)
			}
		})
	}
}

func TestComments(t *testing.T) {
	// Hash comment
	l := New("sistema # this is a comment\nmeuapp")
	tokens, err := l.Tokenize()
	if err != nil {
		t.Fatal(err)
	}
	// Should have: TokenSistema, TokenNewline, TokenIndent (0), TokenIdentifier("meuapp"), TokenEOF
	found := false
	for _, tok := range tokens {
		if tok.Type == TokenIdentifier && tok.Value == "meuapp" {
			found = true
			break
		}
	}
	if !found {
		t.Error("comment should not consume next line; 'meuapp' not found")
	}

	// Double-slash comment
	l2 := New("sistema // another comment\nmeuapp")
	tokens2, err := l2.Tokenize()
	if err != nil {
		t.Fatal(err)
	}
	found2 := false
	for _, tok := range tokens2 {
		if tok.Type == TokenIdentifier && tok.Value == "meuapp" {
			found2 = true
			break
		}
	}
	if !found2 {
		t.Error("// comment should not consume next line; 'meuapp' not found")
	}
}

func TestIndentationTracking(t *testing.T) {
	source := "sistema\n  nome\n    campo"
	l := New(source)
	tokens, err := l.Tokenize()
	if err != nil {
		t.Fatal(err)
	}

	// Find indent tokens and check their indent levels
	var indents []int
	for _, tok := range tokens {
		if tok.Type == TokenIndent {
			indents = append(indents, tok.Indent)
		}
	}
	if len(indents) < 2 {
		t.Fatalf("expected at least 2 indent tokens, got %d", len(indents))
	}
	if indents[0] != 2 {
		t.Errorf("first indent: expected 2, got %d", indents[0])
	}
	if indents[1] != 4 {
		t.Errorf("second indent: expected 4, got %d", indents[1])
	}
}

func TestImportKeywords(t *testing.T) {
	l := New(`importar dados de "models.ge"`)
	tokens, err := l.Tokenize()
	if err != nil {
		t.Fatal(err)
	}

	if tokens[0].Type != TokenImportar {
		t.Errorf("expected TokenImportar, got %d", tokens[0].Type)
	}
	if tokens[1].Type != TokenDados {
		t.Errorf("expected TokenDados, got %d", tokens[1].Type)
	}
	if tokens[2].Type != TokenDe {
		t.Errorf("expected TokenDe, got %d", tokens[2].Type)
	}
	if tokens[3].Type != TokenString || tokens[3].Value != "models.ge" {
		t.Errorf("expected string 'models.ge', got type=%d value=%q", tokens[3].Type, tokens[3].Value)
	}
}

func TestIdentifiers(t *testing.T) {
	l := New("meuCampo meu_campo campo123")
	tokens, err := l.Tokenize()
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"meuCampo", "meu_campo", "campo123"}
	for i, exp := range expected {
		if tokens[i].Type != TokenIdentifier {
			t.Errorf("token %d: expected TokenIdentifier, got %d", i, tokens[i].Type)
		}
		if tokens[i].Value != exp {
			t.Errorf("token %d: expected %q, got %q", i, exp, tokens[i].Value)
		}
	}
}

func TestIsBlockKeyword(t *testing.T) {
	blockTypes := []TokenType{
		TokenSistema, TokenDados, TokenTelas, TokenAcoes, TokenEventos,
		TokenIntegracoes, TokenTema, TokenLogica, TokenBanco, TokenAutenticacao, TokenConfig,
	}
	for _, tt := range blockTypes {
		if !IsBlockKeyword(tt) {
			t.Errorf("expected IsBlockKeyword(%d) to be true", tt)
		}
	}

	nonBlock := []TokenType{TokenIdentifier, TokenString, TokenNumber, TokenSe, TokenFuncao}
	for _, tt := range nonBlock {
		if IsBlockKeyword(tt) {
			t.Errorf("expected IsBlockKeyword(%d) to be false", tt)
		}
	}
}

func TestIsTypeKeyword(t *testing.T) {
	typeTokens := []TokenType{
		TokenTexto, TokenNumero, TokenData, TokenBooleano, TokenEmail,
		TokenTelefone, TokenImagem, TokenArquivo, TokenUpload, TokenLink,
		TokenStatus, TokenDinheiro, TokenSenha, TokenTextoLongo, TokenEnum,
	}
	for _, tt := range typeTokens {
		if !IsTypeKeyword(tt) {
			t.Errorf("expected IsTypeKeyword(%d) to be true", tt)
		}
	}
	if IsTypeKeyword(TokenIdentifier) {
		t.Error("expected IsTypeKeyword(TokenIdentifier) to be false")
	}
}

func TestUnterminatedString(t *testing.T) {
	l := New(`"unterminated string`)
	_, err := l.Tokenize()
	if err == nil {
		t.Error("expected error for unterminated string, got nil")
	}
}

func TestLineAndColumnTracking(t *testing.T) {
	l := New("sistema\nmeuapp")
	tokens, err := l.Tokenize()
	if err != nil {
		t.Fatal(err)
	}
	// "sistema" is on line 1
	if tokens[0].Line != 1 {
		t.Errorf("sistema: expected line 1, got %d", tokens[0].Line)
	}
	// "meuapp" is on line 2
	for _, tok := range tokens {
		if tok.Type == TokenIdentifier && tok.Value == "meuapp" {
			if tok.Line != 2 {
				t.Errorf("meuapp: expected line 2, got %d", tok.Line)
			}
			break
		}
	}
}
