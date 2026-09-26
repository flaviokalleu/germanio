package parser

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/diagnostics"
	"github.com/flaviokalleu/germanio/compiler/lexer"
)

type geLine struct {
	tokens []lexer.Token
	indent int
}
type geParser struct {
	lines   []geLine
	at      int
	inTest  bool
	program *ast.Program
}

// ParseGermanio is a strict entry point. It never skips unknown statements.
// Legacy Parse remains available for .fg, including its multilingual grammar.
func ParseGermanio(filename, source string) (*ast.Program, error) {
	tokens, err := lexer.NewGermanio(filename, source).Tokenize()
	if err != nil {
		return nil, err
	}
	g := &geParser{program: &ast.Program{Filename: filename, Source: source, Domain: "shared"}}
	var line []lexer.Token
	flush := func() {
		if len(line) > 0 {
			g.lines = append(g.lines, geLine{tokens: line, indent: line[0].Column - 1})
			line = nil
		}
	}
	for _, t := range tokens {
		if t.Type == lexer.TokenNewline || t.Type == lexer.TokenEOF {
			flush()
		} else if t.Type != lexer.TokenIndent {
			line = append(line, t)
		}
	}
	for i, s := range strings.Split(source, "\n") {
		prefix := s[:len(s)-len(strings.TrimLeft(s, " \t"))]
		if strings.Contains(prefix, "\t") {
			return nil, g.fail(lexer.Token{Line: i + 1, Column: 1}, "GE1002", "Tabulação não permitida", "Tabulações variam entre editores.", "Use dois espaços por nível.")
		}
	}
	g.program.Scripts, err = g.block(0, true)
	return g.program, err
}

func (g *geParser) pos(t lexer.Token) diagnostics.Position {
	return diagnostics.Position{File: g.program.Filename, Line: t.Line, Column: t.Column}
}
func (g *geParser) fail(t lexer.Token, code, msg, why, fix string) error {
	return &diagnostics.Diagnostic{Code: code, Position: g.pos(t), Source: g.program.Source, Message: msg, Reason: why, Fix: fix, Example: `mostre "Olá"`}
}
func (g *geParser) syntax(t lexer.Token, msg string) error {
	return g.fail(t, "GE1001", msg, "A instrução não corresponde à gramática Germanio.", "Confira a palavra, o valor e os delimitadores; use mostre para mostrar um valor.")
}
func (g *geParser) body(indent int) ([]*ast.Statement, error) {
	if g.at >= len(g.lines) || g.lines[g.at].indent != indent+2 {
		t := lexer.Token{Line: 1, Column: 1}
		if g.at > 0 {
			t = g.lines[g.at-1].tokens[0]
		}
		return nil, g.fail(t, "GE1002", "Bloco ausente ou desalinhado", "Todo bloco precisa de pelo menos uma instrução.", "Indente o corpo com mais dois espaços.")
	}
	return g.block(indent+2, false)
}
func (g *geParser) block(indent int, top bool) ([]*ast.Statement, error) {
	if indent > 512 {
		return nil, g.fail(g.lines[g.at].tokens[0], "GE1002", "Muitos blocos aninhados", "O limite de 256 níveis foi excedido.", "Divida o código em funções menores.")
	}
	var out []*ast.Statement
	for g.at < len(g.lines) {
		l := g.lines[g.at]
		if l.indent < indent {
			break
		}
		if l.indent != indent {
			return nil, g.fail(l.tokens[0], "GE1002", "Indentação inesperada", "O nível não pertence a um bloco aberto.", "Use dois espaços por nível e alinhe instruções irmãs.")
		}
		g.at++
		s, err := g.statement(l, top)
		if err != nil {
			return nil, err
		}
		if s != nil {
			out = append(out, s)
		}
	}
	return out, nil
}
func (g *geParser) statement(l geLine, top bool) (*ast.Statement, error) {
	t := l.tokens
	first := t[0]
	s := &ast.Statement{Pos: g.pos(first)}
	expr := func(ts []lexer.Token) (*ast.Expression, error) { return g.expression(ts) }
	var err error
	keyword := first.Value
	if first.Type != lexer.TokenIdentifier {
		keyword = ""
	}
	switch keyword {
	case "teste":
		if !top || g.inTest || len(t) != 2 || t[1].Type != lexer.TokenString || t[1].Value == "" {
			return nil, g.syntax(first, "Use teste \"nome\" no topo do arquivo, seguido de um bloco")
		}
		g.inTest = true
		body, e := g.body(l.indent)
		g.inTest = false
		if e != nil {
			return nil, e
		}
		g.program.Tests = append(g.program.Tests, &ast.TestDecl{Pos: g.pos(first), Name: t[1].Value, Body: body})
		return nil, nil
	case "espera":
		if !g.inTest {
			return nil, g.syntax(first, "espera pertence ao corpo de teste \"nome\"")
		}
		s.Type = "expect"
		s.Expect, err = expr(t[1:])
		return s, err
	case "usa":
		if !top {
			return nil, g.syntax(first, "Importações pertencem ao topo do arquivo")
		}
		if len(t) != 2 || t[1].Type != lexer.TokenString {
			return nil, g.syntax(first, `Use usa "matematica.ge"; módulos std remotos ainda não são suportados`)
		}
		g.program.Imports = append(g.program.Imports, &ast.Import{Path: t[1].Value, Pos: g.pos(first)})
		return nil, nil
	case "mostre", "retorne":
		s.Expr, err = expr(t[1:])
		s.Type = "expr"
		if first.Value == "mostre" {
			s.Type = "print"
			s.Print = s.Expr
		} else {
			s.Type = "return"
			s.Return = s.Expr
		}
		return s, err
	case "pare", "continue":
		if len(t) != 1 {
			return nil, g.syntax(first, "Comando sem argumentos")
		}
		s.Type = "break"
		if first.Value == "continue" {
			s.Type = "continue"
		}
		return s, nil
	case "se", "enquanto":
		cond, e := expr(t[1:])
		if e != nil {
			return nil, e
		}
		body, e := g.body(l.indent)
		if e != nil {
			return nil, e
		}
		if first.Value == "enquanto" {
			s.Type = "while"
			s.While = &ast.WhileStmt{Condition: *cond, Body: body}
			return s, nil
		}
		s.Type = "if"
		s.If = &ast.IfStmt{Condition: *cond, Body: body}
		if g.at < len(g.lines) && g.lines[g.at].indent == l.indent && g.lines[g.at].tokens[0].Value == "senao" {
			other := g.lines[g.at]
			g.at++
			if len(other.tokens) != 1 {
				return nil, g.syntax(other.tokens[0], "Use senao seguido de um bloco")
			}
			s.If.Else, err = g.body(l.indent)
		}
		return s, err
	case "para":
		if len(t) < 4 || t[1].Type != lexer.TokenIdentifier || t[2].Value != "em" {
			return nil, g.syntax(first, "Use para item em lista")
		}
		coll, e := expr(t[3:])
		if e != nil {
			return nil, e
		}
		body, e := g.body(l.indent)
		s.Type = "for_each"
		s.ForEach = &ast.ForEachStmt{VarName: t[1].Value, Collection: *coll, Body: body}
		return s, e
	case "crie":
		s.Type = "ui"
		s.UI, err = g.ui(t, l.indent)
		return s, err
	}
	private := keyword == "privado"
	if private {
		if !top || len(t) < 3 || t[1].Type != lexer.TokenIdentifier || t[2].Value != "(" && t[2].Value != "<" {
			return nil, g.syntax(first, "Use privado antes de uma função no topo do módulo")
		}
		t = t[1:]
	}
	// Type parameters appear only on declarations, immediately before (.
	open := 1
	var typeParams []string
	if len(t) > 2 && t[1].Value == "<" {
		open = 2
		for open < len(t) && t[open].Value != ">" {
			if t[open].Type != lexer.TokenIdentifier || reservedTypeName(t[open].Value) {
				return nil, g.syntax(t[open], "Parâmetro de tipo precisa de um nome como T")
			}
			for _, previous := range typeParams {
				if previous == t[open].Value {
					return nil, g.syntax(t[open], "Parâmetro de tipo duplicado: "+previous)
				}
			}
			typeParams = append(typeParams, t[open].Value)
			open++
			if open < len(t) && t[open].Value == "," {
				open++
				if open < len(t) && t[open].Value == ">" {
					return nil, g.syntax(t[open], "Remova a vírgula final de <T>")
				}
			} else if open < len(t) && t[open].Value != ">" {
				return nil, g.syntax(t[open], "Separe parâmetros de tipo por vírgulas")
			}
		}
		if len(typeParams) == 0 || open >= len(t) || open+1 >= len(t) || t[open+1].Value != "(" {
			return nil, g.syntax(first, "Use funcao<T>(valor: T) = valor")
		}
		open++
	}
	// A function declaration is distinguished from a call by =, -> or a body.
	if len(t) > open+1 && t[0].Type == lexer.TokenIdentifier && t[open].Value == "(" {
		close := -1
		depth := 0
		for i, x := range t {
			if x.Type == lexer.TokenLParen {
				depth++
			}
			if x.Type == lexer.TokenRParen {
				depth--
				if depth == 0 {
					close = i
					break
				}
			}
		}
		if close >= 0 && (close < len(t)-1 && (t[close+1].Value == "=" || t[close+1].Value == "->") || close == len(t)-1 && g.at < len(g.lines) && g.lines[g.at].indent > l.indent) {
			if !top {
				return nil, g.syntax(first, "Funções devem ser declaradas no topo do módulo")
			}
			fn := &ast.FuncDecl{Name: t[0].Value, Pos: g.pos(first), Private: private, TypeParams: typeParams}
			args := t[open+1 : close]
			for len(args) > 0 {
				if args[0].Type != lexer.TokenIdentifier {
					return nil, g.syntax(args[0], "Parâmetro precisa de um nome")
				}
				fn.Params = append(fn.Params, args[0].Value)
				args = args[1:]
				typ := ""
				if len(args) > 0 && args[0].Value == ":" {
					args = args[1:]
					n := 0
					for n < len(args) && args[n].Value != "," {
						n++
					}
					typ, err = g.annotation(args[:n], fn.TypeParams)
					if err != nil {
						return nil, err
					}
					args = args[n:]
				}
				fn.ParamTypes = append(fn.ParamTypes, typ)
				if len(args) > 0 {
					if args[0].Value != "," || len(args) == 1 {
						return nil, g.syntax(args[0], "Separe parâmetros por vírgulas")
					}
					args = args[1:]
				}
			}
			tail := t[close+1:]
			if len(tail) > 0 && tail[0].Value == "->" {
				n := 1
				for n < len(tail) && tail[n].Value != "=" {
					n++
				}
				fn.ResultType, err = g.annotation(tail[1:n], fn.TypeParams)
				if err != nil {
					return nil, err
				}
				tail = tail[n:]
			}
			if len(tail) > 0 {
				if tail[0].Value != "=" {
					return nil, g.syntax(tail[0], "Esperado = ou corpo indentado")
				}
				x, e := expr(tail[1:])
				if e != nil {
					return nil, e
				}
				fn.Body = []*ast.Statement{{Type: "return", Return: x, Pos: x.Pos}}
			} else {
				fn.Body, err = g.body(l.indent)
				if err != nil {
					return nil, err
				}
				last := fn.Body[len(fn.Body)-1]
				if last.Type == "expr" {
					last.Type = "return"
					last.Return = last.Expr
				}
			}
			g.program.Functions = append(g.program.Functions, fn)
			return nil, nil
		}
	}
	if private {
		return nil, g.syntax(first, "Use privado apenas antes da declaração de uma função")
	}
	mutable, constant := false, false
	if first.Value == "mut" || first.Value == "variavel" || first.Value == "const" {
		mutable = first.Value == "mut" || first.Value == "variavel"
		constant = first.Value == "const"
		t = t[1:]
		if len(t) == 0 {
			return nil, g.syntax(first, "Falta nome e valor")
		}
	}
	if len(t) > 1 && t[0].Type == lexer.TokenIdentifier && (t[1].Value == "=" || t[1].Value == ":" || t[1].Value == "+=" || t[1].Value == "-=") {
		name := t[0].Value
		rest := t[1:]
		annotation := ""
		if rest[0].Value == ":" {
			n := 1
			for n < len(rest) && rest[n].Value != "=" {
				n++
			}
			annotation, err = g.annotation(rest[1:n], nil)
			if err != nil {
				return nil, err
			}
			rest = rest[n:]
		}
		if len(rest) < 2 {
			return nil, g.syntax(first, "Variáveis exigem valor inicial")
		}
		if rest[0].Value != "=" && rest[0].Value != "+=" && rest[0].Value != "-=" {
			return nil, g.syntax(rest[0], "Esperado =")
		}
		x, e := expr(rest[1:])
		if e != nil {
			return nil, e
		}
		if rest[0].Value != "=" {
			if mutable || constant || annotation != "" {
				return nil, g.syntax(first, "Declare com = antes de usar += ou -=")
			}
			s.Type = "assign"
			s.Assign = &ast.Assignment{Target: name, Value: ast.Expression{Type: "binary", Operator: string(rest[0].Value[0]), Left: &ast.Expression{Type: "variable", Name: name, Pos: g.pos(t[0])}, Right: x, Pos: g.pos(rest[0])}}
			return s, nil
		}
		if constant && !top {
			return nil, g.syntax(first, "const pertence ao topo do módulo")
		}
		s.Type = "bind"
		s.VarDecl = &ast.VarDecl{Name: name, Value: *x, Mutable: mutable, Constant: constant, Annotation: annotation}
		return s, nil
	}
	if mutable || constant {
		return nil, g.syntax(first, "Use variavel nome = valor ou const NOME = valor")
	}
	if len(t) > 1 && first.Type == lexer.TokenIdentifier && t[1].Type == lexer.TokenString && (first.Value == "moste" || first.Value == "mostra" || first.Value == "mostr" || first.Value == "mostrar") {
		return nil, g.fail(first, "GE1001", "Você quis dizer mostre?", first.Value+" não é o comando de saída Germanio.", `Use mostre "Olá". O compilador não altera seu código silenciosamente.`)
	}
	s.Type = "expr"
	s.Expr, err = expr(t)
	return s, err
}
func reservedTypeName(name string) bool {
	switch name {
	case "texto", "inteiro", "decimal", "bool", "nulo", "privado", "variavel", "mut", "const":
		return true
	}
	return false
}
func (g *geParser) annotation(ts []lexer.Token, typeParams []string) (string, error) {
	if len(ts) == 0 {
		return "", g.syntax(lexer.Token{Line: 1, Column: 1}, "Tipo ausente")
	}
	var b strings.Builder
	for _, t := range ts {
		b.WriteString(t.Value)
	}
	s := b.String()
	base := strings.TrimSuffix(s, "?")
	for strings.HasPrefix(base, "[") && strings.HasSuffix(base, "]") {
		base = base[1 : len(base)-1]
		base = strings.TrimSuffix(base, "?")
	}
	switch base {
	case "texto", "inteiro", "decimal", "bool":
		return s, nil
	}
	for _, name := range typeParams {
		if base == name {
			return s, nil
		}
	}
	return "", g.syntax(ts[0], "Tipo ainda não suportado: "+s+". Use texto, inteiro, decimal, bool, [T] ou T?")
}

type geExpr struct {
	g     *geParser
	ts    []lexer.Token
	at    int
	depth int
}

func (g *geParser) expression(ts []lexer.Token) (*ast.Expression, error) {
	p := &geExpr{g: g, ts: ts}
	x, err := p.parse(0)
	if err == nil && p.at < len(ts) {
		err = g.syntax(ts[p.at], "Trecho inesperado: "+ts[p.at].Value)
	}
	return x, err
}
func (p *geExpr) tok() lexer.Token {
	if p.at < len(p.ts) {
		return p.ts[p.at]
	}
	if len(p.ts) > 0 {
		t := p.ts[len(p.ts)-1]
		t.Value = "fim da linha"
		t.Column++
		return t
	}
	return lexer.Token{Line: 1, Column: 1, Value: "fim da linha"}
}
func (p *geExpr) accept(s string) bool {
	if p.at < len(p.ts) && p.ts[p.at].Type != lexer.TokenString && p.ts[p.at].Value == s {
		p.at++
		return true
	}
	return false
}

var gePrecedence = map[string]int{"ou": 1, "e": 2, "==": 3, "!=": 3, "<": 4, ">": 4, "<=": 4, ">=": 4, "+": 5, "-": 5, "*": 6, "/": 6, "%": 6}

func (p *geExpr) parse(min int) (*ast.Expression, error) {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > 512 {
		return nil, p.g.syntax(p.tok(), "Expressão muito aninhada; divida em variáveis intermediárias")
	}
	t := p.tok()
	if p.at >= len(p.ts) {
		return nil, p.g.syntax(t, "Expressão ausente")
	}
	p.at++
	x := &ast.Expression{Pos: p.g.pos(t)}
	var err error
	switch {
	case t.Type == lexer.TokenString:
		x, err = p.g.stringExpr(t)
	case t.Value == "-" || t.Value == "nao":
		if t.Value == "-" && p.at < len(p.ts) && p.ts[p.at].Type == lexer.TokenNumber && p.ts[p.at].Value == "9223372036854775808" {
			p.at++
			x.Type = "literal"
			x.Value = int64(-9223372036854775808)
			break
		}
		x.Type = "unary"
		x.Operator = t.Value
		x.Right, err = p.parse(7)
	case t.Value == "(":
		x, err = p.parse(0)
		if err == nil && !p.accept(")") {
			err = p.g.syntax(p.tok(), "Falta fechar )")
		}
	case t.Value == "[":
		x.Type = "list"
		if !p.accept("]") {
			for {
				el, e := p.parse(0)
				if e != nil {
					return nil, e
				}
				x.Elements = append(x.Elements, el)
				if p.accept("]") {
					break
				}
				if !p.accept(",") {
					return nil, p.g.syntax(p.tok(), "Esperado , ou ]")
				}
			}
		}
	case t.Type == lexer.TokenNumber:
		x.Type = "literal"
		if strings.Contains(t.Value, ".") {
			x.Value, err = strconv.ParseFloat(t.Value, 64)
		} else {
			x.Value, err = strconv.ParseInt(t.Value, 10, 64)
		}
		if err != nil {
			return nil, p.g.fail(t, "GE2005", "Número fora dos limites", err.Error(), "Use um inteiro de 64 bits ou um decimal finito.")
		}
	case t.Value == "verdadeiro" || t.Value == "falso":
		x.Type = "literal"
		x.Value = t.Value == "verdadeiro"
	case t.Value == "nulo":
		x.Type = "literal"
	case t.Value == "pergunte":
		x.Type = "call"
		x.Name = "pergunte"
		a, e := p.parse(0)
		err = e
		x.Args = []*ast.Expression{a}
	case t.Type == lexer.TokenIdentifier:
		x.Type = "variable"
		x.Name = t.Value
		if p.accept(".") {
			field := p.tok()
			if field.Type != lexer.TokenIdentifier {
				return nil, p.g.syntax(field, "Esperado nome exportado do módulo")
			}
			p.at++
			x.Name += "." + field.Value
		}
		if p.accept("(") {
			x.Type = "call"
			if !p.accept(")") {
				for {
					a, e := p.parse(0)
					if e != nil {
						return nil, e
					}
					x.Args = append(x.Args, a)
					if p.accept(")") {
						break
					}
					if !p.accept(",") {
						return nil, p.g.syntax(p.tok(), "Esperado , ou )")
					}
				}
			}
		}
	default:
		return nil, p.g.syntax(t, "Expressão desconhecida: "+t.Value)
	}
	if err != nil {
		return nil, err
	}
	for p.accept("[") {
		idx, e := p.parse(0)
		if e != nil {
			return nil, e
		}
		if !p.accept("]") {
			return nil, p.g.syntax(p.tok(), "Falta fechar ]")
		}
		x = &ast.Expression{Type: "index", Left: x, Index: idx, Pos: x.Pos}
	}
	for p.at < len(p.ts) {
		op := p.tok()
		if op.Type == lexer.TokenString {
			break
		}
		prec := gePrecedence[op.Value]
		if prec == 0 || prec < min {
			break
		}
		p.at++
		right, e := p.parse(prec + 1)
		if e != nil {
			return nil, e
		}
		x = &ast.Expression{Type: "binary", Left: x, Right: right, Operator: op.Value, Pos: p.g.pos(op)}
	}
	return x, nil
}
func (g *geParser) stringExpr(t lexer.Token) (*ast.Expression, error) {
	x := &ast.Expression{Type: "literal", Value: t.Value, Pos: g.pos(t)}
	if !strings.Contains(t.Value, "{") {
		return x, nil
	}
	x.Type = "interpolate"
	s := t.Value
	for len(s) > 0 {
		start := strings.Index(s, "{")
		if start < 0 {
			x.Elements = append(x.Elements, &ast.Expression{Type: "literal", Value: s, Pos: x.Pos})
			break
		}
		x.Elements = append(x.Elements, &ast.Expression{Type: "literal", Value: s[:start], Pos: x.Pos})
		end := strings.Index(s[start+1:], "}")
		if end < 0 {
			return nil, g.syntax(t, "Falta fechar } na interpolação")
		}
		name := s[start+1 : start+1+end]
		if name == "" {
			return nil, g.syntax(t, "Interpolação exige um nome")
		}
		for i, r := range name {
			if !(unicode.IsLetter(r) || r == '_' || i > 0 && unicode.IsDigit(r)) {
				return nil, g.syntax(t, "Use um nome simples em {nome}")
			}
		}
		x.Elements = append(x.Elements, &ast.Expression{Type: "variable", Name: name, Pos: x.Pos})
		s = s[start+end+2:]
	}
	return x, nil
}
func (g *geParser) ui(ts []lexer.Token, indent int) (*ast.UIElement, error) {
	t := ts[1:]
	if len(t) > 0 && t[0].Value == "um" {
		t = t[1:]
	}
	if len(t) == 0 {
		return nil, g.syntax(ts[0], "Use crie botão azul")
	}
	u := &ast.UIElement{Kind: t[0].Value, Text: "Botão", Color: "azul", Radius: "8px"}
	t = t[1:]
	if u.Kind != "botão" && u.Kind != "navbar" {
		return nil, g.syntax(ts[0], "Nesta fase use crie botão ou crie navbar")
	}
	if u.Kind == "navbar" {
		u.Text = "Navegação"
		u.Color = "preto"
	}
	for len(t) > 0 {
		if t[0].Value == "escrito" && len(t) > 1 && t[1].Type == lexer.TokenString {
			u.Text = t[1].Value
			t = t[2:]
			continue
		}
		c := t[0].Value
		if c == "escuro" {
			c = "preto"
		}
		if _, ok := ast.ColorName[c]; !ok {
			return nil, g.syntax(t[0], "Cor desconhecida")
		}
		u.Color = c
		t = t[1:]
	}
	for g.at < len(g.lines) && g.lines[g.at].indent > indent {
		l := g.lines[g.at]
		if l.indent != indent+2 {
			return nil, g.syntax(l.tokens[0], "Propriedades usam dois espaços")
		}
		g.at++
		t = l.tokens
		if len(t) == 4 && t[0].Value == "borda" && t[1].Value == "arredondada" && t[2].Type == lexer.TokenNumber {
			unit := t[3].Value
			if unit != "px" && unit != "rem" {
				return nil, g.fail(t[3], "GE4102", "Unidade incorreta", fmt.Sprintf("%s não é uma unidade de comprimento suportada para bordas.", unit), "Use borda arredondada 16px ou 1rem.")
			}
			u.Radius = t[2].Value + unit
		} else {
			return nil, g.syntax(t[0], "Propriedade UI ainda não suportada; use borda arredondada 16px")
		}
	}
	return u, nil
}
