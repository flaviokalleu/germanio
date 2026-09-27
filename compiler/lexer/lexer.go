package lexer

import (
	"fmt"
	"github.com/flaviokalleu/germanio/compiler/diagnostics"
	"strconv"
	"strings"
	"unicode"

	"github.com/flaviokalleu/germanio/compiler/idiomas"
)

// TokenType identifies the kind of token.
type TokenType int

const (
	// Special
	TokenEOF TokenType = iota
	TokenNewline
	TokenIndent

	// Literals
	TokenIdentifier
	TokenString
	TokenNumber

	// Punctuation
	TokenColon
	TokenDot
	TokenComma
	TokenEquals
	TokenPlus
	TokenMinus
	TokenStar
	TokenSlash
	TokenLParen
	TokenRParen
	TokenLBrace
	TokenRBrace

	// Block keywords
	TokenSistema
	TokenDados
	TokenTelas
	TokenAcoes
	TokenEventos
	TokenIntegracoes
	TokenTema
	TokenLogica
	TokenBanco
	TokenAutenticacao
	TokenConfig
	TokenTabelaTop
	TokenQuandoTop

	// Import
	TokenImportar
	TokenDe

	// Screen keywords
	TokenTela
	TokenTitulo
	TokenLista
	TokenMostrar
	TokenBotao
	TokenFormulario
	TokenChat
	TokenEntrada
	TokenTextoKW
	TokenBusca
	TokenDashboard
	TokenGrafico
	TokenTabela
	TokenCampo
	TokenSelecionar
	TokenAreaTexto

	// Type keywords
	TokenTexto
	TokenNumero
	TokenData
	TokenBooleano
	TokenEmail
	TokenTelefone
	TokenImagem
	TokenArquivo
	TokenUpload
	TokenLink
	TokenStatus
	TokenDinheiro
	TokenSenha
	TokenTextoLongo
	TokenEnum

	// Event keywords
	TokenQuando
	TokenClicar
	TokenCriar
	TokenAtualizar
	TokenDeletar
	TokenEnviar
	TokenReceber
	TokenBuscar
	TokenOnde
	TokenResponder
	TokenSucesso
	TokenCom

	// Logic keywords
	TokenSe
	TokenSenao
	TokenIgual
	TokenMaior
	TokenMenor
	TokenE
	TokenOu
	TokenEntao
	TokenValidar
	TokenCalcular
	TokenDefinir
	TokenRetornar
	TokenMudar
	TokenPara
	TokenParaCada
	TokenFuncao
	TokenTentar
	TokenErro

	// Auth keywords
	TokenLogin
	TokenRegistro
	TokenUsuario
	TokenPermissao
	TokenRequer
	TokenAdmin
	TokenPublico

	// Integration keywords
	TokenWhatsapp
	TokenSessao
	TokenQRCode
	TokenPresenca
	TokenStreaming
	TokenFilaJobs
	TokenMensagem
	TokenNotificar
	TokenEmailInteg
	TokenCron
	TokenCada
	TokenHora
	TokenMinuto
	TokenChamar
	TokenApi
	TokenWebhook
	TokenPagamento

	// Relationship keywords
	TokenPertenceA
	TokenTemMuitos
	TokenMuitosParaMuitos

	// Theme keywords
	TokenCor
	TokenIcone
	TokenEscuro

	// Modifier keywords
	TokenObrigatorio
	TokenUnico
	TokenPadrao
	TokenIndice
	TokenSoftDelete

	// New logic/scripting tokens
	TokenPausar
	TokenContinuar
	TokenParar
	TokenEnquanto
	TokenRepetir
	TokenVezes
	TokenNulo
	TokenVerdadeiro
	TokenFalso
	TokenNao
	TokenEm
	TokenAte
	TokenDiferente
	TokenMaiorIgual
	TokenMenorIgual
	TokenMaiorQue
	TokenMenorQue
	TokenEqualEqual
	TokenLBracket
	TokenRBracket

	// New field type tokens
	TokenCPF
	TokenCEP
	TokenEstrelas
	TokenHoraTipo
	TokenDataHora
	TokenPercentual
	TokenTags
	TokenURLTipo
	TokenMoeda
	TokenQuestion
	TokenArrow
	TokenPlusAssign
	TokenMinusAssign
	TokenModulo
)

// Token represents a single lexical token.
type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
	Indent int
}

func (t Token) String() string {
	return fmt.Sprintf("Token(%d, %q, line=%d, col=%d, indent=%d)", t.Type, t.Value, t.Line, t.Column, t.Indent)
}

var keywords = map[string]TokenType{
	// ===================== PORTUGUÊS =====================
	"sistema": TokenSistema, "app": TokenSistema, "aplicacao": TokenSistema,
	"dados": TokenDados, "modelos": TokenDados, "telas": TokenTelas,
	"acoes": TokenAcoes, "eventos": TokenEventos, "integracoes": TokenIntegracoes,
	"tema": TokenTema, "logica": TokenLogica, "banco": TokenBanco,
	"autenticacao": TokenAutenticacao, "config": TokenConfig,
	"importar": TokenImportar, "de": TokenDe,
	// Tela
	"tela": TokenTela, "titulo": TokenTitulo, "lista": TokenLista,
	"mostrar": TokenMostrar, "botao": TokenBotao, "formulario": TokenFormulario,
	"chat":    TokenChat,
	"entrada": TokenEntrada, "busca": TokenBusca, "dashboard": TokenDashboard,
	"grafico": TokenGrafico, "tabela": TokenTabela, "campo": TokenCampo,
	"selecionar": TokenSelecionar, "area_texto": TokenAreaTexto,
	// Tipos
	"texto": TokenTexto, "numero": TokenNumero, "data": TokenData,
	"booleano": TokenBooleano, "email": TokenEmail, "telefone": TokenTelefone,
	"imagem": TokenImagem, "arquivo": TokenArquivo, "upload": TokenUpload,
	"link": TokenLink, "status": TokenStatus, "dinheiro": TokenDinheiro,
	"senha": TokenSenha, "texto_longo": TokenTextoLongo, "enum": TokenEnum,
	// Eventos
	"quando": TokenQuando, "clicar": TokenClicar, "criar": TokenCriar,
	"atualizar": TokenAtualizar, "deletar": TokenDeletar, "enviar": TokenEnviar,
	"receber": TokenReceber, "buscar": TokenBuscar,
	"onde": TokenOnde, "responder": TokenResponder, "sucesso": TokenSucesso,
	"com": TokenCom,
	// Lógica
	"se": TokenSe, "senao": TokenSenao, "igual": TokenIgual,
	"maior": TokenMaior, "menor": TokenMenor, "e": TokenE, "ou": TokenOu,
	"entao": TokenEntao, "validar": TokenValidar, "calcular": TokenCalcular,
	"definir": TokenDefinir, "retornar": TokenRetornar, "mudar": TokenMudar,
	"para": TokenPara, "para_cada": TokenParaCada, "funcao": TokenFuncao,
	"tentar": TokenTentar, "erro": TokenErro,
	// Auth
	"login": TokenLogin, "registro": TokenRegistro, "usuario": TokenUsuario,
	"permissao": TokenPermissao, "requer": TokenRequer, "admin": TokenAdmin,
	"publico": TokenPublico,
	// Integrações
	"whatsapp": TokenWhatsapp, "sessao": TokenSessao, "qr_code": TokenQRCode,
	"presenca": TokenPresenca, "streaming": TokenStreaming, "fila_jobs": TokenFilaJobs,
	"mensagem": TokenMensagem, "notificar": TokenNotificar,
	"cron": TokenCron, "cada": TokenCada, "hora": TokenHora, "minuto": TokenMinuto,
	"chamar": TokenChamar, "api": TokenApi, "webhook": TokenWebhook,
	"pagamento": TokenPagamento,
	// Relacionamentos
	"pertence_a": TokenPertenceA, "tem_muitos": TokenTemMuitos,
	"muitos_para_muitos": TokenMuitosParaMuitos,
	// Tema
	"cor": TokenCor, "icone": TokenIcone, "escuro": TokenEscuro,
	// Modificadores
	"obrigatorio": TokenObrigatorio, "unico": TokenUnico, "padrao": TokenPadrao,
	"indice": TokenIndice, "soft_delete": TokenSoftDelete,
	// Scripting PT
	"pausar": TokenPausar, "continuar": TokenContinuar, "parar": TokenParar,
	"enquanto": TokenEnquanto, "repetir": TokenRepetir, "vezes": TokenVezes,
	"nulo": TokenNulo, "nada": TokenNulo,
	"verdadeiro": TokenVerdadeiro, "falso": TokenFalso,
	"nao": TokenNao, "em": TokenEm, "ate": TokenAte,
	// Tipos extras
	"cpf": TokenCPF, "cnpj": TokenCPF,
	"cep":        TokenCEP,
	"estrelas":   TokenEstrelas,
	"data_hora":  TokenDataHora,
	"percentual": TokenPercentual, "porcentagem": TokenPercentual,
	"tags": TokenTags, "etiquetas": TokenTags,
	"moeda": TokenMoeda,

	// ===================== ENGLISH =====================
	"system": TokenSistema, "models": TokenDados, "screens": TokenTelas,
	"actions": TokenAcoes, "events": TokenEventos, "integrations": TokenIntegracoes,
	"theme": TokenTema, "logic": TokenLogica, "database": TokenBanco, "db": TokenBanco,
	"auth": TokenAutenticacao, "authentication": TokenAutenticacao,
	"import": TokenImportar, "from": TokenDe,
	// Screen
	"screen": TokenTela, "title": TokenTitulo, "list": TokenLista,
	"show": TokenMostrar, "button": TokenBotao, "form": TokenFormulario,
	"input": TokenEntrada, "search": TokenBusca, "chart": TokenGrafico,
	"table": TokenTabela, "field": TokenCampo, "select": TokenSelecionar,
	"textarea": TokenAreaTexto,
	// Types
	"text": TokenTexto, "number": TokenNumero, "date": TokenData,
	"boolean": TokenBooleano, "phone": TokenTelefone, "image": TokenImagem,
	"file": TokenArquivo, "money": TokenDinheiro, "password": TokenSenha,
	"currency": TokenDinheiro, "long_text": TokenTextoLongo,
	// Events
	"when": TokenQuando, "click": TokenClicar, "create": TokenCriar,
	"update": TokenAtualizar, "delete": TokenDeletar, "send": TokenEnviar,
	"qrcode": TokenQRCode,
	// Logic
	"if": TokenSe, "else": TokenSenao, "equals": TokenIgual, "equal": TokenIgual,
	"greater": TokenMaior, "less": TokenMenor, "and": TokenE, "or": TokenOu,
	"then": TokenEntao, "validate": TokenValidar, "compute": TokenCalcular,
	"set": TokenDefinir, "return": TokenRetornar, "change": TokenMudar,
	"to": TokenPara, "for_each": TokenParaCada, "function": TokenFuncao,
	"try": TokenTentar, "error": TokenErro,
	// Auth
	"register": TokenRegistro, "user": TokenUsuario, "permission": TokenPermissao,
	"requires": TokenRequer, "public": TokenPublico,
	// Integrations
	"message": TokenMensagem, "notify": TokenNotificar,
	"every": TokenCada, "hour": TokenHora, "minute": TokenMinuto,
	"call": TokenChamar, "payment": TokenPagamento,
	// Relationships
	"belongs_to": TokenPertenceA, "has_many": TokenTemMuitos,
	"many_to_many": TokenMuitosParaMuitos,
	// Theme
	"color": TokenCor, "icon": TokenIcone, "dark": TokenEscuro,
	// Modifiers
	"required": TokenObrigatorio, "unique": TokenUnico, "default": TokenPadrao,
	"index": TokenIndice,
	// Scripting EN
	"pause": TokenPausar, "continue": TokenContinuar, "break": TokenParar, "stop": TokenParar,
	"while": TokenEnquanto, "repeat": TokenRepetir, "times": TokenVezes,
	"null": TokenNulo, "nothing": TokenNulo,
	"true": TokenVerdadeiro, "false": TokenFalso,
	"not": TokenNao, "in": TokenEm, "until": TokenAte,
	"for": TokenPara, "each": TokenCada, "print": TokenMostrar,
	// Extra types EN
	"rating": TokenEstrelas, "stars": TokenEstrelas,
	"zipcode": TokenCEP, "postal_code": TokenCEP,
	"datetime":   TokenDataHora,
	"percentage": TokenPercentual, "percent": TokenPercentual,
	"chips":         TokenTags,
	"currency_type": TokenMoeda,

	// Colors
	"azul": TokenIdentifier, "verde": TokenIdentifier, "vermelho": TokenIdentifier,
	"blue": TokenIdentifier, "green": TokenIdentifier, "red": TokenIdentifier,
}

// Lexer tokenizes Germanio source code.
type Lexer struct {
	germanio bool
	filename string
	source   []rune
	pos      int
	line     int
	col      int
	tokens   []Token
}

// NewGermanio reuses scanning mechanics without legacy keyword translation.
func NewGermanio(filename, source string) *Lexer {
	l := New(source)
	l.germanio, l.filename = true, filename
	return l
}

// New creates a new Lexer for the given source code.
func New(source string) *Lexer {
	return &Lexer{
		source: []rune(source),
		pos:    0,
		line:   1,
		col:    1,
	}
}

// Tokenize processes the entire source and returns all tokens.
func (l *Lexer) Tokenize() ([]Token, error) {
	for l.pos < len(l.source) {
		if err := l.scanToken(); err != nil {
			if l.germanio {
				return nil, &diagnostics.Diagnostic{Code: "GE1001", Position: diagnostics.Position{File: l.filename, Line: l.line, Column: l.col}, Source: string(l.source), Message: "Código não reconhecido", Reason: err.Error(), Fix: "Confira caracteres, números e aspas duplas. Use mostre para mostrar um valor.", Example: `mostre "Olá"`}
			}
			return nil, err
		}
	}
	l.tokens = append(l.tokens, Token{Type: TokenEOF, Line: l.line, Column: l.col})
	return l.tokens, nil
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.source) {
		return 0
	}
	return l.source[l.pos]
}

func (l *Lexer) advance() rune {
	ch := l.source[l.pos]
	l.pos++
	l.col++
	return ch
}

func (l *Lexer) scanToken() error {
	ch := l.peek()
	if l.germanio {
		if ch == '?' || ch == '%' {
			tt := TokenQuestion
			if ch == '%' {
				tt = TokenModulo
			}
			l.tokens = append(l.tokens, Token{Type: tt, Value: string(ch), Line: l.line, Column: l.col})
			l.advance()
			return nil
		}
		if l.pos+1 < len(l.source) {
			pair := string(l.source[l.pos : l.pos+2])
			if tt, ok := map[string]TokenType{"->": TokenArrow, "+=": TokenPlusAssign, "-=": TokenMinusAssign}[pair]; ok {
				l.tokens = append(l.tokens, Token{Type: tt, Value: pair, Line: l.line, Column: l.col})
				l.advance()
				l.advance()
				return nil
			}
		}
	}

	// Handle newlines
	if ch == '\n' {
		l.tokens = append(l.tokens, Token{Type: TokenNewline, Value: "\n", Line: l.line, Column: l.col})
		l.advance()
		l.line++
		l.col = 1

		// Count indentation on the new line
		indent := 0
		for l.pos < len(l.source) && (l.peek() == ' ' || l.peek() == '\t') {
			if l.peek() == '\t' {
				indent += 2
			} else {
				indent++
			}
			l.advance()
		}
		if l.pos < len(l.source) && l.peek() != '\n' && l.peek() != '\r' {
			l.tokens = append(l.tokens, Token{Type: TokenIndent, Value: fmt.Sprintf("%d", indent), Line: l.line, Column: 1, Indent: indent})
		}
		return nil
	}

	// Skip carriage return
	if ch == '\r' {
		l.advance()
		return nil
	}

	// Skip spaces (mid-line)
	if ch == ' ' || ch == '\t' {
		l.advance()
		return nil
	}

	// Skip comments (// or #)
	if ch == '/' && l.pos+1 < len(l.source) && l.source[l.pos+1] == '/' {
		for l.pos < len(l.source) && l.peek() != '\n' {
			l.advance()
		}
		return nil
	}
	if ch == '#' {
		for l.pos < len(l.source) && l.peek() != '\n' {
			l.advance()
		}
		return nil
	}

	// Colon
	if ch == ':' {
		l.tokens = append(l.tokens, Token{Type: TokenColon, Value: ":", Line: l.line, Column: l.col})
		l.advance()
		return nil
	}

	// Dot
	if ch == '.' {
		l.tokens = append(l.tokens, Token{Type: TokenDot, Value: ".", Line: l.line, Column: l.col})
		l.advance()
		return nil
	}

	// Comma
	if ch == ',' {
		l.tokens = append(l.tokens, Token{Type: TokenComma, Value: ",", Line: l.line, Column: l.col})
		l.advance()
		return nil
	}

	// Operators
	if ch == '=' {
		col := l.col
		l.advance()
		if l.pos < len(l.source) && l.peek() == '=' {
			l.advance()
			l.tokens = append(l.tokens, Token{Type: TokenEqualEqual, Value: "==", Line: l.line, Column: col})
		} else {
			l.tokens = append(l.tokens, Token{Type: TokenEquals, Value: "=", Line: l.line, Column: col})
		}
		return nil
	}
	if ch == '!' {
		col := l.col
		l.advance()
		if l.pos < len(l.source) && l.peek() == '=' {
			l.advance()
			l.tokens = append(l.tokens, Token{Type: TokenDiferente, Value: "!=", Line: l.line, Column: col})
		} else {
			l.tokens = append(l.tokens, Token{Type: TokenNao, Value: "!", Line: l.line, Column: col})
		}
		return nil
	}
	if ch == '<' {
		col := l.col
		l.advance()
		if l.pos < len(l.source) && l.peek() == '=' {
			l.advance()
			l.tokens = append(l.tokens, Token{Type: TokenMenorIgual, Value: "<=", Line: l.line, Column: col})
		} else if l.pos < len(l.source) && l.peek() == '>' {
			l.advance()
			l.tokens = append(l.tokens, Token{Type: TokenDiferente, Value: "<>", Line: l.line, Column: col})
		} else {
			l.tokens = append(l.tokens, Token{Type: TokenMenorQue, Value: "<", Line: l.line, Column: col})
		}
		return nil
	}
	if ch == '>' {
		col := l.col
		l.advance()
		if l.pos < len(l.source) && l.peek() == '=' {
			l.advance()
			l.tokens = append(l.tokens, Token{Type: TokenMaiorIgual, Value: ">=", Line: l.line, Column: col})
		} else {
			l.tokens = append(l.tokens, Token{Type: TokenMaiorQue, Value: ">", Line: l.line, Column: col})
		}
		return nil
	}
	if ch == '+' {
		l.tokens = append(l.tokens, Token{Type: TokenPlus, Value: "+", Line: l.line, Column: l.col})
		l.advance()
		return nil
	}
	if ch == '-' {
		l.tokens = append(l.tokens, Token{Type: TokenMinus, Value: "-", Line: l.line, Column: l.col})
		l.advance()
		return nil
	}
	if ch == '*' {
		l.tokens = append(l.tokens, Token{Type: TokenStar, Value: "*", Line: l.line, Column: l.col})
		l.advance()
		return nil
	}
	if ch == '/' {
		l.tokens = append(l.tokens, Token{Type: TokenSlash, Value: "/", Line: l.line, Column: l.col})
		l.advance()
		return nil
	}
	if ch == '(' {
		l.tokens = append(l.tokens, Token{Type: TokenLParen, Value: "(", Line: l.line, Column: l.col})
		l.advance()
		return nil
	}
	if ch == ')' {
		l.tokens = append(l.tokens, Token{Type: TokenRParen, Value: ")", Line: l.line, Column: l.col})
		l.advance()
		return nil
	}
	if ch == '[' {
		l.tokens = append(l.tokens, Token{Type: TokenLBracket, Value: "[", Line: l.line, Column: l.col})
		l.advance()
		return nil
	}
	if ch == ']' {
		l.tokens = append(l.tokens, Token{Type: TokenRBracket, Value: "]", Line: l.line, Column: l.col})
		l.advance()
		return nil
	}
	if ch == '{' {
		l.tokens = append(l.tokens, Token{Type: TokenLBrace, Value: "{", Line: l.line, Column: l.col})
		l.advance()
		return nil
	}
	if ch == '}' {
		l.tokens = append(l.tokens, Token{Type: TokenRBrace, Value: "}", Line: l.line, Column: l.col})
		l.advance()
		return nil
	}

	// String literal
	if ch == '"' {
		return l.scanString()
	}

	// Number
	if unicode.IsDigit(ch) {
		return l.scanNumber()
	}

	// Identifier or keyword (supports all Unicode scripts including CJK, Devanagari, Arabic, etc.)
	if unicode.IsLetter(ch) || ch == '_' || unicode.IsMark(ch) {
		return l.scanIdentifier()
	}

	// Skip unknown unicode marks (combining characters at unexpected positions)
	if unicode.IsMark(ch) || unicode.Is(unicode.Mn, ch) {
		l.advance()
		return nil
	}

	return fmt.Errorf("unexpected character %q at line %d, column %d", string(ch), l.line, l.col)
}

func (l *Lexer) scanString() error {
	startLine := l.line
	startCol := l.col

	// Multiline string support with """
	if l.pos+2 < len(l.source) && l.source[l.pos] == '"' && l.source[l.pos+1] == '"' && l.source[l.pos+2] == '"' {
		l.pos += 3
		l.col += 3
		var buf strings.Builder
		for l.pos < len(l.source) {
			if l.pos+2 < len(l.source) && l.source[l.pos] == '"' && l.source[l.pos+1] == '"' && l.source[l.pos+2] == '"' {
				break
			}
			ch := l.source[l.pos]
			if ch == '\n' {
				l.line++
				l.col = 1
				l.pos++
				buf.WriteByte('\n')
				continue
			}
			buf.WriteRune(ch)
			l.pos++
			l.col++
		}
		if l.pos >= len(l.source) {
			return fmt.Errorf("unterminated multiline string at line %d, column %d", startLine, startCol)
		}
		l.pos += 3
		l.col += 3
		l.tokens = append(l.tokens, Token{Type: TokenString, Value: buf.String(), Line: startLine, Column: startCol})
		return nil
	}

	l.advance() // skip opening quote
	var buf strings.Builder
	for l.pos < len(l.source) && l.peek() != '"' {
		if l.peek() == '\n' {
			return fmt.Errorf("unterminated string at line %d, column %d", startLine, startCol)
		}
		if l.peek() == '\\' {
			l.advance()
			if l.pos >= len(l.source) {
				return fmt.Errorf("unterminated escape at line %d", l.line)
			}
			switch l.peek() {
			case 'n':
				buf.WriteByte('\n')
			case 't':
				buf.WriteByte('\t')
			case '"':
				buf.WriteByte('"')
			case '\\':
				buf.WriteByte('\\')
			default:
				if l.germanio {
					return fmt.Errorf("escape desconhecido: use \\n, \\t, \\\\ ou \\\"")
				}
				buf.WriteRune(l.peek())
			}
			l.advance()
			continue
		}
		buf.WriteRune(l.advance())
	}
	if l.pos >= len(l.source) {
		return fmt.Errorf("unterminated string at line %d, column %d", startLine, startCol)
	}
	l.advance() // skip closing quote
	l.tokens = append(l.tokens, Token{Type: TokenString, Value: buf.String(), Line: startLine, Column: startCol})
	return nil
}

func (l *Lexer) scanNumber() error {
	startCol := l.col
	var buf strings.Builder
	for l.pos < len(l.source) && (unicode.IsDigit(l.peek()) || l.peek() == '.') {
		buf.WriteRune(l.advance())
	}
	if l.germanio {
		if _, err := strconv.ParseFloat(buf.String(), 64); err != nil {
			return fmt.Errorf("número inválido %q", buf.String())
		}
	}
	l.tokens = append(l.tokens, Token{Type: TokenNumber, Value: buf.String(), Line: l.line, Column: startCol})
	return nil
}

func (l *Lexer) scanIdentifier() error {
	startCol := l.col
	var buf strings.Builder
	for l.pos < len(l.source) && (unicode.IsLetter(l.peek()) || unicode.IsDigit(l.peek()) || l.peek() == '_' || unicode.IsMark(l.peek())) {
		buf.WriteRune(l.advance())
	}
	word := buf.String()
	if l.germanio {
		l.tokens = append(l.tokens, Token{Type: TokenIdentifier, Value: word, Line: l.line, Column: startCol})
		return nil
	}
	lower := strings.ToLower(word)

	// Check native keywords first (PT + EN)
	if tt, ok := keywords[lower]; ok {
		l.tokens = append(l.tokens, Token{Type: tt, Value: lower, Line: l.line, Column: startCol})
		return nil
	}

	// Check multilingual translations → canonical PT keyword → token
	if canonical, ok := idiomas.Translations[lower]; ok && canonical != "_skip" {
		if tt, ok := keywords[canonical]; ok {
			l.tokens = append(l.tokens, Token{Type: tt, Value: canonical, Line: l.line, Column: startCol})
			return nil
		}
	}
	// Also try the original word (handles CJK characters that may not lowercase)
	if canonical, ok := idiomas.Translations[word]; ok && canonical != "_skip" {
		if tt, ok := keywords[canonical]; ok {
			l.tokens = append(l.tokens, Token{Type: tt, Value: canonical, Line: l.line, Column: startCol})
			return nil
		}
	}

	l.tokens = append(l.tokens, Token{Type: TokenIdentifier, Value: word, Line: l.line, Column: startCol})
	return nil
}

// IsBlockKeyword returns true if the token type is a top-level block keyword.
func IsBlockKeyword(tt TokenType) bool {
	switch tt {
	case TokenSistema, TokenDados, TokenTelas, TokenAcoes, TokenEventos, TokenIntegracoes,
		TokenTema, TokenLogica, TokenBanco, TokenAutenticacao, TokenConfig,
		TokenTabela, TokenQuando:
		return true
	}
	return false
}

// IsTypeKeyword returns true if the token is a data type.
func IsTypeKeyword(tt TokenType) bool {
	switch tt {
	case TokenTexto, TokenNumero, TokenData, TokenBooleano, TokenEmail,
		TokenTelefone, TokenImagem, TokenArquivo, TokenUpload, TokenLink,
		TokenStatus, TokenDinheiro, TokenSenha, TokenTextoLongo, TokenEnum,
		TokenCPF, TokenCEP, TokenEstrelas, TokenDataHora, TokenPercentual, TokenTags, TokenMoeda:
		return true
	}
	return false
}
