package parser

import (
	"strconv"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/lexer"
)

// dline is one source line of a block: its indentation, its tokens (without
// NEWLINE/INDENT) and the token range it occupies in p.tokens.
type dline struct {
	indent   int
	toks     []lexer.Token
	from, to int
}

// blockLines collects the lines that belong to the block whose header was
// just consumed: everything until the next non-blank line that starts at
// column 1. Tokens left on the header line become line 0 with indent -1.
func (p *Parser) blockLines() []dline {
	var lines []dline
	var cur *dline
	indent := -1
	start := p.pos
	for !p.isAtEnd() {
		tok := p.current()
		switch tok.Type {
		case lexer.TokenNewline:
			if cur != nil {
				cur.to = p.pos
				lines = append(lines, *cur)
				cur = nil
			}
			indent = 0
			p.advance()
			start = p.pos
			continue
		case lexer.TokenIndent:
			indent = tok.Indent
			p.advance()
			continue
		}
		if cur == nil {
			if indent == 0 {
				// Column 1: the block is over. Leave the token for Parse.
				return lines
			}
			cur = &dline{indent: indent, from: start}
		}
		cur.toks = append(cur.toks, tok)
		p.advance()
	}
	if cur != nil {
		cur.to = p.pos
		lines = append(lines, *cur)
	}
	return lines
}

// children returns the lines nested under lines[i] (strictly deeper).
func children(lines []dline, i int) []dline {
	var out []dline
	for j := i + 1; j < len(lines) && lines[j].indent > lines[i].indent; j++ {
		out = append(out, lines[j])
	}
	return out
}

// directChildren returns indexes of the lines one level below lines[i].
func directChildren(lines []dline, i int) []int {
	var out []int
	level := -1
	for j := i + 1; j < len(lines) && lines[j].indent > lines[i].indent; j++ {
		if level == -1 {
			level = lines[j].indent
		}
		if lines[j].indent == level {
			out = append(out, j)
		}
	}
	return out
}

// subParser parses statements that live in a range of tokens (bodies of
// actions and startup blocks) with the ordinary statement grammar.
func (p *Parser) subParser(from, to int) *Parser {
	toks := make([]lexer.Token, 0, to-from+3)
	toks = append(toks, lexer.Token{Type: lexer.TokenNewline})
	toks = append(toks, p.tokens[from:to]...)
	toks = append(toks, lexer.Token{Type: lexer.TokenNewline}, lexer.Token{Type: lexer.TokenEOF})
	sp := New(toks)
	sp.File = p.File
	return sp
}

// statementsIn parses lines as a statement block.
func (p *Parser) statementsIn(lines []dline) ([]*ast.Statement, error) {
	if len(lines) == 0 {
		return nil, nil
	}
	sp := p.subParser(lines[0].from, lines[len(lines)-1].to)
	return sp.parseBlock(lines[0].indent - 1)
}

// ==================== dados ====================

// parseDados parses a dados block. Models sit at the block's first
// indentation; their members are the deeper lines.
func (p *Parser) parseDados() error {
	p.advance() // consume 'dados'
	lines := p.blockLines()
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		kids := children(lines, i)
		m, err := p.modelFromLines(l.toks, kids)
		if err != nil {
			return err
		}
		p.program.Models = append(p.program.Models, m)
		i += len(kids)
	}
	return nil
}

// parseDirectTable parses "tabela Nome" at the top level.
func (p *Parser) parseDirectTable() error {
	p.advance() // consume 'tabela'
	lines := p.blockLines()
	if len(lines) == 0 {
		return p.errorf(p.current(), "tabela exige um nome")
	}
	m, err := p.modelFromLines(lines[0].toks, lines[1:])
	if err != nil {
		return err
	}
	p.program.Models = append(p.program.Models, m)
	return nil
}

func (p *Parser) modelFromLines(head []lexer.Token, members []dline) (*ast.Model, error) {
	if len(head) == 0 || (!p.isNameToken(head[0]) && !lexer.IsBlockKeyword(head[0].Type)) {
		return nil, p.errorf(head[0], "nome de modelo esperado")
	}
	m := &ast.Model{Name: head[0].Name(), Pos: p.at(head[0])}
	for _, t := range head[1:] {
		switch {
		case t.Type == lexer.TokenSoftDelete:
			m.SoftDelete = true
		case t.Name() == "interno" || t.Name() == "internal":
			m.Internal = true
		default:
			return nil, p.errorf(t, "modificador de modelo desconhecido %q (use interno ou soft_delete)", t.Name())
		}
	}
	for _, l := range members {
		if err := p.modelMember(m, l.toks); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func (p *Parser) modelMember(m *ast.Model, t []lexer.Token) error {
	first := t[0]
	word := func(i int) string {
		if i < len(t) {
			return t[i].Name()
		}
		return ""
	}
	switch {
	case first.Type == lexer.TokenTemMuitos:
		if len(t) < 2 {
			return p.errorf(first, "tem_muitos exige um modelo")
		}
		m.HasMany = append(m.HasMany, t[1].Name())
		return nil
	case first.Type == lexer.TokenMuitosParaMuitos:
		if len(t) < 2 {
			return p.errorf(first, "muitos_para_muitos exige um modelo")
		}
		m.ManyToMany = append(m.ManyToMany, t[1].Name())
		return nil
	case (first.Type == lexer.TokenUnico || first.Type == lexer.TokenIndice) && len(t) > 1 && t[1].Type == lexer.TokenLParen:
		var group []string
		for _, x := range t[2:] {
			if x.Type == lexer.TokenComma {
				continue
			}
			if x.Type == lexer.TokenRParen {
				break
			}
			group = append(group, x.Name())
		}
		if len(group) < 2 {
			return p.errorf(first, "%s(...) exige pelo menos dois campos", first.Value)
		}
		if first.Type == lexer.TokenUnico {
			m.UniqueTogether = append(m.UniqueTogether, group)
		} else {
			m.IndexTogether = append(m.IndexTogether, group)
		}
		return nil
	case first.Type == lexer.TokenPertenceA || (word(0) == "pertence" && word(1) == "a"):
		// pertence_a modelo [opcional] [como nome] → nome_id inteiro indexado
		i := 1
		if first.Type != lexer.TokenPertenceA {
			i = 2
		}
		if i >= len(t) {
			return p.errorf(first, "pertence_a exige um modelo")
		}
		ref := t[i].Name()
		f := &ast.Field{Name: ref + "_id", Type: ast.FieldInteiro, Reference: ref, Index: true, Required: true, Pos: p.at(first)}
		for i++; i < len(t); i++ {
			switch t[i].Name() {
			case "opcional":
				f.Required = false
			case "como":
				if i+1 >= len(t) {
					return p.errorf(t[i], "como exige um nome")
				}
				f.Name = t[i+1].Name() + "_id"
				i++
			default:
				return p.errorf(t[i], "depois de pertence_a use opcional ou como <nome>")
			}
		}
		m.Fields = append(m.Fields, f)
		return nil
	case word(0) == "expira" && len(t) == 4 && t[1].Type == lexer.TokenEm && t[2].Type == lexer.TokenNumber && (word(3) == "dias" || word(3) == "dia"):
		n, _ := strconv.Atoi(t[2].Value)
		m.ExpiresDays = n
		m.Fields = append(m.Fields, &ast.Field{Name: "expira_em", Type: ast.FieldData, Pos: p.at(first)})
		return nil
	case word(0) == "revogavel" && len(t) == 1:
		m.Revocable = true
		m.Fields = append(m.Fields, &ast.Field{Name: "revogado", Type: ast.FieldBooleano, HasDefault: true, DefaultValue: false, Pos: p.at(first)})
		return nil
	}
	f, err := p.fieldFromTokens(t)
	if err != nil {
		return err
	}
	m.Fields = append(m.Fields, f)
	return nil
}

// Type inference for fields declared without a type (documented table).
// The table is part of the language definition (docs/INTENCAO.md).
func inferType(name string) ast.FieldType {
	n := strings.ToLower(foldWord(name))
	switch n {
	case "email", "e_mail":
		return ast.FieldEmail
	case "senha", "password":
		return ast.FieldSenha
	case "telefone", "celular", "phone":
		return ast.FieldTelefone
	case "foto", "imagem", "avatar", "logo", "image", "photo":
		return ast.FieldImagem
	case "arquivo", "anexo", "file", "attachment":
		return ast.FieldArquivo
	case "descricao", "description", "conteudo", "observacoes", "bio", "body", "texto", "mensagem", "comentario":
		return ast.FieldTextoLongo
	case "visibilidade", "visibility":
		return ast.FieldVisibilidade
	case "url", "site", "endereco_web", "link":
		return ast.FieldURL
	case "preco", "valor", "price", "total":
		return ast.FieldDinheiro
	case "quantidade", "estoque", "idade", "posicao", "numero":
		return ast.FieldInteiro
	case "admin", "ativo", "active", "bloqueado", "arquivado", "archived", "publicado", "confidencial", "confidential":
		return ast.FieldBooleano
	}
	if strings.HasPrefix(n, "pode_") || strings.HasPrefix(n, "is_") || strings.HasPrefix(n, "tem_") || strings.HasPrefix(n, "can_") {
		return ast.FieldBooleano
	}
	return ast.FieldTexto
}

// fieldFromTokens parses: nome [:] [tipo] [modificadores] [= padrão].
func (p *Parser) fieldFromTokens(t []lexer.Token) (*ast.Field, error) {
	name := t[0]
	if !p.isNameToken(name) && !lexer.IsBlockKeyword(name.Type) {
		return nil, p.errorf(name, "nome de campo esperado, encontrado %q", name.Value)
	}
	// Names are accent-folded so "descrição" and "descricao" are the same field.
	f := &ast.Field{Name: foldWord(name.Name()), Label: strings.ReplaceAll(name.Name(), "_", " "), Pos: p.at(name)}
	i := 1
	if i < len(t) && t[i].Type == lexer.TokenColon {
		i++
	}
	// lista de texto / lista de usuarios
	if i+2 < len(t) && foldWord(strings.ToLower(t[i].Name())) == "lista" && t[i+1].Type == lexer.TokenDe {
		f.Type = ast.FieldLista
		f.ListOf = strings.ToLower(foldWord(t[i+2].Name()))
		i += 3
	}
	if i < len(t) && f.Type == "" {
		if ft, err := tokenToFieldType(t[i]); err == nil && t[i].Type != lexer.TokenUnico && t[i].Type != lexer.TokenIndice {
			f.Type = ft
			i++
			if ft == ast.FieldEnum && i < len(t) && t[i].Type == lexer.TokenLParen {
				for i++; i < len(t) && t[i].Type != lexer.TokenRParen; i++ {
					if t[i].Type != lexer.TokenComma {
						f.EnumValues = append(f.EnumValues, t[i].Name())
					}
				}
				i++
			}
		} else if n := foldWord(strings.ToLower(t[i].Name())); n == "segredo" || n == "secreto" || n == "secreta" {
			f.Type = ast.FieldSegredo
			i++
		}
	}
	num := func(k int) (*float64, error) {
		if k >= len(t) || t[k].Type != lexer.TokenNumber {
			return nil, p.errorf(t[k-1], "%s exige um número", t[k-1].Name())
		}
		v, _ := strconv.ParseFloat(t[k].Value, 64)
		return &v, nil
	}
	str := func(k int) (string, error) {
		if k >= len(t) || t[k].Type != lexer.TokenString {
			return "", p.errorf(t[k-1], "%s exige um texto entre aspas", t[k-1].Name())
		}
		return t[k].Value, nil
	}
	for ; i < len(t); i++ {
		x := t[i]
		if x.Type == lexer.TokenE || x.Type == lexer.TokenComma {
			continue // "obrigatório e único"
		}
		if strings.ToLower(x.Name()) == "sem" {
			if i+1 < len(t) && strings.HasPrefix(foldWord(strings.ToLower(t[i+1].Name())), "cripto") {
				return nil, p.errorf(x, "senhas nunca são guardadas sem criptografia; remova \"sem criptografia\"")
			}
			return nil, p.errorf(x, "\"sem\" só é usado em \"sem criptografia\", que não é permitido")
		}
		if foldWord(strings.ToLower(x.Name())) == "comeca" && i+2 < len(t) && strings.ToLower(t[i+1].Name()) == "com" {
			t = append(append(append([]lexer.Token{}, t[:i]...), lexer.Token{Type: lexer.TokenEquals, Value: "=", Line: x.Line, Column: x.Column}), t[i+2:]...)
			x = t[i]
		}
		switch {
		case x.Type == lexer.TokenObrigatorio:
			f.Required = true
		case x.Type == lexer.TokenUnico:
			f.Unique = true
		case x.Type == lexer.TokenIndice:
			f.Index = true
		case x.Type == lexer.TokenAte:
			// titulo até 255 → no máximo 255 caracteres
			var err error
			if f.Max, err = num(i + 1); err != nil {
				return nil, err
			}
			i++
		case x.Type == lexer.TokenPertenceA:
			if i+1 >= len(t) {
				return nil, p.errorf(x, "pertence_a exige um modelo")
			}
			f.Reference = t[i+1].Name()
			i++
		case x.Type == lexer.TokenEquals || x.Type == lexer.TokenPadrao:
			if i+1 >= len(t) {
				return nil, p.errorf(x, "valor padrão ausente em %s", f.Name)
			}
			if x.Type == lexer.TokenPadrao && t[i+1].Type == lexer.TokenColon {
				i++
			}
			v := t[i+1]
			switch v.Type {
			case lexer.TokenNumber:
				n, _ := strconv.ParseFloat(v.Value, 64)
				f.DefaultValue = n
				if f.Type == "" {
					if strings.Contains(v.Value, ".") {
						f.Type = ast.FieldNumero
					} else {
						f.Type = ast.FieldInteiro
					}
				}
			case lexer.TokenString:
				f.DefaultValue = v.Value
			case lexer.TokenVerdadeiro, lexer.TokenFalso, lexer.TokenNao:
				f.DefaultValue = v.Type == lexer.TokenVerdadeiro
				if f.Type == "" {
					f.Type = ast.FieldBooleano
				}
			case lexer.TokenNulo:
				f.DefaultValue = nil
			default:
				return nil, p.errorf(v, "valor padrão deve ser número, texto, verdadeiro, falso ou nulo")
			}
			f.HasDefault = true
			f.Default = "" // runtime applies typed defaults
			i++
		default:
			var err error
			switch x.Name() {
			case "protegida", "protegido":
				f.Protected = true
				if f.Type == "" {
					f.Type = ast.FieldSenha
				}
			case "oculto", "oculta":
				f.Hidden = true
			case "privado", "privada":
				f.Private = true
			case "formatado", "formatada":
				f.Formatted = true
				if f.Type == "" || f.Type == ast.FieldTexto {
					f.Type = ast.FieldTextoLongo
				}
			case "imutavel":
				f.Immutable = true
			case "por":
				// numero por projeto → 1, 2, 3… dentro de cada projeto
				if i+1 >= len(t) {
					return nil, p.errorf(x, "use: numero por <dado pai>")
				}
				f.NumberedBy = strings.ToLower(foldWord(t[i+1].Name()))
				if f.Type == "" {
					f.Type = ast.FieldInteiro
				}
				f.Immutable = true
				i++
			case "numerado":
				// numerado por projeto → 1, 2, 3… dentro de cada projeto
				if i+2 >= len(t) || strings.ToLower(t[i+1].Name()) != "por" {
					return nil, p.errorf(x, "use: numerado por <dado pai>")
				}
				f.NumberedBy = strings.ToLower(foldWord(t[i+2].Name()))
				if f.Type == "" {
					f.Type = ast.FieldInteiro
				}
				f.Immutable = true
				i += 2
			case "opcional":
				f.Required = false
			case "min":
				f.Min, err = num(i + 1)
				i++
			case "max":
				f.Max, err = num(i + 1)
				i++
			case "formato":
				f.Format, err = str(i + 1)
				i++
			case "prefixo":
				f.Prefix, err = str(i + 1)
				i++
			case "valida":
				if i+1 >= len(t) || !p.isNameToken(t[i+1]) {
					return nil, p.errorf(x, "valida exige o nome de uma função")
				}
				f.Validator = t[i+1].Name()
				i++
			default:
				return nil, p.errorf(x, "modificador desconhecido %q no campo %s; use obrigatorio, unico, indice, opcional, protegida, oculto, imutavel, min, max, formato, valida, prefixo, pertence_a ou = valor", x.Name(), f.Name)
			}
			if err != nil {
				return nil, err
			}
		}
	}
	if f.Type == "" {
		f.Type, f.TypeInferred = inferType(f.Name), true
	}
	// Things are active until someone turns them off.
	if f.Type == ast.FieldBooleano && !f.HasDefault {
		switch strings.ToLower(f.Name) {
		case "ativo", "ativa", "active", "habilitado", "habilitada", "enabled":
			f.HasDefault, f.DefaultValue = true, true
		}
	}
	if f.Type == ast.FieldSenha {
		f.Protected = true
	}
	return f, nil
}
