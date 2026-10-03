package parser

import (
	"strconv"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/lexer"
)

// Hierarchical, contextual syntax (docs/INTENCAO.md › Sintaxe hierárquica):
// what is below belongs to the context above. A data block
//
//	projetos
//	    acesso
//	        developer
//	            enviar código
//
// is reduced, through a closed table of sections, to the flat phrases the
// intent layer already specifies ("developer pode enviar código para
// projetos"). Both forms therefore produce the same facts by construction;
// every fact made inside a block carries its hierarchical path in its
// position, so ge explain can show where it came from.

// node is one line of a block and the lines nested under it.
type node struct {
	line     dline
	children []*node
}

// layoutTree builds the tree of a block from its lines (line 0 is the
// header at column 1). Indentation follows the level stack of the normative
// layout: a deeper line opens a child level; a shallower line must return to
// a level that is open, otherwise it has no parent.
func (p *Parser) layoutTree(lines []dline, context string) (*node, error) {
	root := &node{line: lines[0]}
	type level struct {
		indent int
		n      *node
	}
	stack := []level{{-1, root}}
	for _, l := range lines[1:] {
		var open []string // the levels this line could land on
		for _, s := range stack[1:] {
			open = append(open, strconv.Itoa(s.indent)+" = "+lineText(s.n.line))
		}
		closed := false
		for len(stack) > 1 && l.indent < stack[len(stack)-1].indent {
			stack = stack[:len(stack)-1]
			closed = true
		}
		top := stack[len(stack)-1]
		switch {
		case closed && l.indent != top.indent:
			// after closing levels the line must land on an open level
			return nil, p.teach(l.toks[0],
				"a linha \""+lineText(l)+"\" não tem um nível aberto acima dela",
				"cada nível abre embaixo da linha de cima; este recuo ("+strconv.Itoa(l.indent)+") não corresponde a nenhum bloco aberto ("+strings.Join(open, ", ")+")",
				"recue a linha para o mesmo recuo de um bloco aberto ou para dentro dele", context)
		case l.indent == top.indent:
			// sibling of the top: its parent is one level below in the stack
			parent := stack[len(stack)-2].n
			n := &node{line: l}
			parent.children = append(parent.children, n)
			stack[len(stack)-1] = level{l.indent, n}
		case l.indent > top.indent:
			n := &node{line: l}
			top.n.children = append(top.n.children, n)
			stack = append(stack, level{l.indent, n})
		default:
			return nil, p.teach(l.toks[0],
				"a linha \""+lineText(l)+"\" não tem um nível aberto acima dela",
				"cada nível abre embaixo da linha de cima; este recuo ("+strconv.Itoa(l.indent)+") não corresponde a nenhum bloco aberto ("+strings.Join(open, ", ")+")",
				"recue a linha para o mesmo recuo de um bloco aberto ou para dentro dele", context)
		}
	}
	return root, nil
}

func lineText(l dline) string {
	var parts []string
	for _, t := range l.toks {
		if t.Type == lexer.TokenString {
			parts = append(parts, "\""+t.Value+"\"")
			continue
		}
		parts = append(parts, t.Name())
	}
	return strings.Join(parts, " ")
}

// teach builds an educational error: what happened, where (file:line and the
// hierarchical path), why, and how to fix.
func (p *Parser) teach(tok lexer.Token, what, why, fix, context string) error {
	msg := what
	if context != "" {
		msg += "\nOnde: " + context
	}
	if why != "" {
		msg += "\nPor quê: " + why
	}
	if fix != "" {
		msg += "\nComo corrigir: " + fix
	}
	return p.errorf(tok, "%s", msg)
}

// leaf: in a list section each line is one item. A line indented under an
// item has no meaning there; it used to be dropped or read as a sibling
// (G67). No line is ever ignored.
func (p *Parser) leaf(item *node, section, path string) error {
	if len(item.children) == 0 {
		return nil
	}
	c := item.children[0]
	return p.teach(c.line.toks[0],
		"\""+lineText(c.line)+"\" está recuada abaixo de \""+lineText(item.line)+"\", mas um item de "+section+" não tem itens dentro dele",
		"em "+section+", cada linha é um item; uma linha abaixo de um item não teria significado",
		"alinhe \""+lineText(c.line)+"\" com os outros itens de "+section+", ou junte as duas na mesma linha", path)
}

// flattenLines returns every line under n, in order (children before siblings).
func flattenLines(n *node) []dline {
	var out []dline
	for _, c := range n.children {
		out = append(out, c.line)
		out = append(out, flattenLines(c)...)
	}
	return out
}

// Words that start the legacy blocks: a data block header is never one of them.
var legacyBlockWords = map[string]bool{
	"sistema": true, "app": true, "aplicacao": true, "dados": true, "modelos": true, "telas": true, "tela": true,
	"acoes": true, "eventos": true, "integracoes": true, "tema": true, "logica": true, "banco": true,
	"autenticacao": true, "config": true, "rotas": true, "paginas": true, "sidebar": true, "menu": true,
	"importar": true, "whatsapp": true, "email": true, "cron": true, "tabela": true, "funcao": true,
	"teste": true, "se": true, "para": true, "enquanto": true, "mostrar": true, "definir": true,
}

// sections are the aspects a data block may contain (folded words).
var sections = []string{"tem", "pertence a", "comeca", "pode", "regras", "acesso", "permita",
	"integracao", "quando", "antes de", "recebe", "executa", "executam", "repositorio", "singular", "pendencia para",
	"renomeie", "descarte", "guarda historico", "guarda leitura", "usam", "usa", "espelham", "espelha"}

func sectionOf(w []string) string {
	if len(w) == 0 {
		return ""
	}
	switch w[0] {
	case "tem", "comeca", "pode", "regras", "acesso", "permita", "integracao", "quando", "recebe", "executa", "executam", "repositorio", "singular", "renomeie", "descarte", "usam", "usa", "espelham", "espelha":
		return w[0]
	case "pertence":
		return "pertence a"
	case "pendencia":
		if len(w) > 1 && w[1] == "para" {
			return "pendencia para"
		}
	case "guarda":
		if len(w) > 1 && w[1] == "historico" {
			return "guarda historico"
		}
		if len(w) > 1 && w[1] == "leitura" {
			return "guarda leitura"
		}
	case "antes":
		if len(w) > 1 && w[1] == "de" {
			return "antes de"
		}
	}
	return ""
}

// isDataBlock: a column-1 line made only of words (the data's name) whose
// first nested line starts with a section. `página Nome` (a name, not a
// quoted path) is a page block.
func (p *Parser) isDataBlock() (page bool, ok bool) {
	tok := p.current()
	if tok.Column != 1 {
		return false, false
	}
	var head []lexer.Token
	i := p.pos
	for ; i < len(p.tokens) && p.tokens[i].Type != lexer.TokenNewline && p.tokens[i].Type != lexer.TokenEOF; i++ {
		if p.tokens[i].Type != lexer.TokenIndent {
			head = append(head, p.tokens[i])
		}
	}
	if len(head) == 0 {
		return false, false
	}
	w := wordsOf(head)
	if (w[0] == "pagina" || w[0] == "page") && len(head) >= 2 && head[1].Type != lexer.TokenString {
		return true, true
	}
	if legacyBlockWords[w[0]] {
		return false, false
	}
	for _, t := range head {
		if t.Type == lexer.TokenString || t.Type == lexer.TokenNumber || !isWord(t.Name()) {
			return false, false
		}
	}
	// the first nested line
	indent := 0
	var child []lexer.Token
	for ; i < len(p.tokens) && p.tokens[i].Type != lexer.TokenEOF; i++ {
		t := p.tokens[i]
		switch t.Type {
		case lexer.TokenNewline:
			if len(child) > 0 {
				i = len(p.tokens)
			}
			continue
		case lexer.TokenIndent:
			indent = t.Indent
			continue
		}
		if len(child) == 0 && indent == 0 {
			return false, false // the next line is at column 1: no block
		}
		child = append(child, t)
	}
	return false, indent > 0 && sectionOf(wordsOf(child)) != ""
}

func isWord(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= 0xC0) {
			return false
		}
	}
	return true
}

// synth makes intent tokens for words at the position of at.
func synth(at lexer.Token, words ...string) []lexer.Token {
	out := make([]lexer.Token, 0, len(words))
	for _, w := range words {
		out = append(out, lexer.Token{Type: lexer.TokenIdentifier, Value: w, Line: at.Line, Column: at.Column})
	}
	return out
}

func join(parts ...[]lexer.Token) []lexer.Token {
	var out []lexer.Token
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// withContext runs fn with the hierarchical path recorded in positions.
func (p *Parser) withContext(path string, fn func() error) error {
	saved := p.context
	p.context = path
	defer func() { p.context = saved }()
	return fn()
}

// parseDataBlock reads `nome` + sections and reduces each section to the
// flat phrases of the table in docs/INTENCAO.md.
func (p *Parser) parseDataBlock() error {
	lines := p.blockLinesWithHeader()
	name := lineText(lines[0])
	root, err := p.layoutTree(lines, name)
	if err != nil {
		return err
	}
	nameToks := lines[0].toks
	before := len(p.intent().Entities)
	if err := p.withContext(name, func() error {
		return p.intentFrom(dline{toks: join(synth(nameToks[0], "tenha"), nameToks)}, nil)
	}); err != nil {
		return err
	}
	for _, d := range p.intent().Entities[before:] {
		d.Implicit = true
	}
	for _, sec := range root.children {
		if err := p.dataSection(name, nameToks, sec); err != nil {
			return err
		}
	}
	return nil
}

// placed returns the data's name tokens positioned at a section line, so
// facts point to the line that says them, not to the block header.
func placed(subject []lexer.Token, at lexer.Token) []lexer.Token {
	out := make([]lexer.Token, len(subject))
	for i, t := range subject {
		t.Line, t.Column = at.Line, at.Column
		out[i] = t
	}
	return out
}

func (p *Parser) dataSection(name string, header []lexer.Token, sec *node) error {
	toks := sec.line.toks
	subject := placed(header, toks[0])
	w := wordsOf(toks)
	kind := sectionOf(w)
	path := name + " › " + strings.Join(strings.Fields(lineText(sec.line)), " ")
	at := toks[0]
	flat := func(head []lexer.Token, body []dline) error {
		return p.withContext(path, func() error { return p.intentFrom(dline{toks: head}, body) })
	}
	switch kind {
	case "":
		return p.teach(at, "\""+lineText(sec.line)+"\" não é uma seção de "+name,
			"dentro de um dado, cada linha do primeiro nível diz de que aspecto se trata",
			"use uma destas seções: "+strings.Join(displaySections(), ", ")+suggest(w[0]), name)
	case "tem", "comeca", "recebe", "executa", "executam", "usam", "usa", "espelham", "espelha":
		if kind == "tem" && len(toks) == 1 && len(sec.children) == 0 {
			return p.teach(at, "a seção tem está vazia", "tem lista campos, relações e pessoas do dado", "escreva um item por linha, recuado abaixo de tem", name)
		}
		if kind == "tem" {
			for _, c := range sec.children {
				if err := p.leaf(c, "tem", path); err != nil {
					return err
				}
			}
		}
		return flat(join(subject, toks), flattenLines(sec))
	case "pode":
		if len(toks) == 1 && len(sec.children) == 0 {
			return p.teach(at, "a seção pode está vazia", "pode lista ações e condições do dado", "escreva uma ação por linha, recuada abaixo de pode (ex.: fechar, reabrir, ser confidencial)", name)
		}
		for _, c := range sec.children {
			if err := p.leaf(c, "pode", path); err != nil {
				return err
			}
		}
		return flat(join(subject, toks), flattenLines(sec))
	case "pertence a":
		if len(w) > 2 {
			return flat(join(subject, toks), nil)
		}
		for _, c := range sec.children {
			if err := p.leaf(c, "pertence a", path); err != nil {
				return err
			}
			if err := p.withContext(path+" › "+lineText(c.line), func() error {
				return p.intentFrom(dline{toks: join(subject, synth(c.line.toks[0], "pertence", "a"), c.line.toks)}, nil)
			}); err != nil {
				return err
			}
		}
		return nil
	case "renomeie":
		// renomeie nome para nome_completo → renomeie nome de <dado> para nome_completo
		k := -1
		for i, x := range w {
			if x == "para" {
				k = i
			}
		}
		if k < 2 || k+1 >= len(w) || len(sec.children) > 0 {
			return p.teach(at, "\""+lineText(sec.line)+"\" não diz o que renomear", "renomeie diz o nome antigo e o novo do campo", "escreva: renomeie nome para nome_completo", name)
		}
		return flat(join(toks[:k], synth(at, "de"), subject, toks[k:]), nil)
	case "descarte":
		if len(toks) < 2 || len(sec.children) > 0 {
			return p.teach(at, "\""+lineText(sec.line)+"\" não diz qual campo", "descarte diz qual campo foi removido de propósito", "escreva: descarte telefone", name)
		}
		return flat(join(toks, synth(at, "de"), subject), nil)
	case "guarda historico", "guarda leitura":
		// guarda histórico / guarda leitura → issue guarda histórico
		if len(toks) != 2 || len(sec.children) > 0 {
			return p.teach(at, "\""+lineText(sec.line)+"\" não é uma seção conhecida", lineText(sec.line)+" fica sozinho na linha, no bloco do dado", "escreva: "+lineText(dline{toks: toks}), name)
		}
		return flat(join(subject, toks), nil)
	case "pendencia para":
		// pendência para › responsaveis → issue gera pendência para responsaveis
		for _, c := range sec.children {
			if err := p.leaf(c, "pendência para", path); err != nil {
				return err
			}
			if err := p.withContext(path+" › "+lineText(c.line), func() error {
				return p.intentFrom(dline{toks: join(subject, synth(c.line.toks[0], "gera", "pendencia", "para"), c.line.toks)}, nil)
			}); err != nil {
				return err
			}
		}
		return nil
	case "regras":
		for _, r := range sec.children {
			if err := p.rule(name, subject, path, r); err != nil {
				return err
			}
		}
		return nil
	case "acesso":
		for _, actor := range sec.children {
			if err := p.access(name, subject, path, actor); err != nil {
				return err
			}
		}
		return nil
	case "permita":
		for _, c := range sec.children {
			cw := wordsOf(c.line.toks)
			ct := c.line.toks
			k := 1
			for k < len(cw) && cw[k] != "por" {
				k++
			}
			head := join(synth(ct[0], "permita"), ct[:1], subject, ct[1:k], ct[k:])
			if err := p.withContext(path+" › "+lineText(c.line), func() error {
				return p.intentFrom(dline{toks: head}, flattenLines(c))
			}); err != nil {
				return err
			}
		}
		return nil
	case "integracao":
		head := join(synth(at, "disponibilize"), subject, synth(at, "para", "integracao"))
		for _, c := range sec.children {
			cw := wordsOf(c.line.toks)
			if len(cw) == 2 && cw[0] == "nome" && c.line.toks[1].Type == lexer.TokenString {
				head = append(head, synth(c.line.toks[0], "como")[0], c.line.toks[1])
				continue
			}
			return p.teach(c.line.toks[0], "\""+lineText(c.line)+"\" não é uma configuração de integração",
				"integração aceita o nome externo do dado", "escreva: nome \"projects\"", path)
		}
		return flat(head, nil)
	case "quando", "antes de":
		return flat(join(toks, subject), flattenLines(sec))
	case "singular":
		// singular token de acesso → cada token de acesso tem (the form that names one)
		if len(toks) < 2 {
			return p.teach(at, "falta a forma no singular", "singular diz como se chama um só registro quando o plural admite duas leituras (tokens → token ou tokem)", "escreva: singular "+strings.TrimSuffix(name, "s"), name)
		}
		// the declared form names the data, whatever the inference says
		// (branches protegidas → branch protegida, not branche protegida)
		plural, _ := phrase(wordsOf(subject))
		sing, _ := phrase(wordsOf(toks[1:]))
		for _, d := range p.intent().Entities {
			if d.Name == plural {
				d.Singular = sing
			}
		}
		return flat(join(synth(at, "cada"), toks[1:], synth(at, "tem")), nil)
	case "repositorio":
		// repositório pode começar com "x" contendo "y" → repositório do <dado> pode …
		return flat(join(toks[:1], synth(at, "do"), subject, toks[1:]), nil)
	}
	return nil
}

// rule reduces one line of `regras`.
func (p *Parser) rule(_ string, header []lexer.Token, path string, r *node) error {
	t := r.line.toks
	subject := placed(header, t[0])
	w := wordsOf(t)
	at := t[0]
	var head []lexer.Token
	switch {
	case len(w) >= 3 && w[0] == "quem" && w[1] == "cria" && w[2] == "vira":
		// quem cria vira owner → quem cria <dado> vira owner
		head = join(t[:2], subject, t[2:])
	case len(w) >= 5 && w[0] == "precisa" && (w[1] == "de" || w[1] == "ter") && w[2] == "pelo" && w[3] == "menos":
		// precisa de pelo menos um owner → todo <dado> precisa ter pelo menos um owner
		head = join(synth(at, "todo"), subject, synth(at, "precisa", "ter"), t[2:])
	default:
		head = join(subject, t)
	}
	return p.withContext(path+" › "+lineText(r.line), func() error {
		return p.intentFrom(dline{toks: head}, flattenLines(r))
	})
}

// access reduces `acesso` › actor › actions to `<actor> pode <action> <dado>`.
func (p *Parser) access(name string, header []lexer.Token, path string, actor *node) error {
	at := actor.line.toks[0]
	actorPath := path + " › " + lineText(actor.line)
	if len(actor.children) == 0 {
		return p.teach(at, "\""+lineText(actor.line)+"\" não diz o que pode fazer",
			"em acesso, cada pessoa ou papel lista, recuadas abaixo, as ações que pode fazer em "+name,
			"escreva as ações abaixo dele, por exemplo:\n    "+lineText(actor.line)+"\n        ver", path)
	}
	var body []dline
	for _, a := range actor.children {
		if err := p.leaf(a, "acesso › "+lineText(actor.line), actorPath); err != nil {
			return err
		}
		aw := wordsOf(a.line.toks)
		k := 1
		if len(aw) > 1 && (aw[0] == "enviar" || aw[0] == "baixar") && aw[1] == "codigo" {
			k = 2
		}
		if k < len(aw) && (aw[k] == "para" || aw[k] == "em") {
			k++
		}
		for k < len(aw) && (possessives[aw[k]] || fillers[aw[k]]) {
			k++
		}
		rest := aw[k:]
		toks := a.line.toks
		subject := placed(header, a.line.toks[0])
		switch {
		case len(rest) == 0:
			toks = join(toks, subject) // no target: the data itself
		case strings.Join(rest, "_") == "branch_padrao":
			toks = join(toks, synth(a.line.toks[0], "dos"), subject)
		}
		body = append(body, dline{indent: a.line.indent, toks: toks, from: a.line.from, to: a.line.to})
	}
	before := len(p.intent().Grants)
	err := p.withContext(actorPath, func() error {
		return p.intentFrom(dline{toks: join(actor.line.toks, synth(at, "pode"))}, body)
	})
	for _, g := range p.intent().Grants[before:] {
		g.Context = name
	}
	return err
}

func displaySections() []string {
	return []string{"tem", "pertence a", "começa", "pode", "regras", "acesso", "permita", "integração", "quando", "antes de", "recebe", "executa", "executam", "repositório", "singular", "espelham"}
}

// suggest proposes the closest section for a misspelled word.
func suggest(word string) string {
	best, dist := "", 3
	for _, s := range sections {
		if d := editDistance(word, strings.Fields(s)[0]); d < dist {
			best, dist = s, d
		}
	}
	if best == "" {
		return ""
	}
	return " (você quis dizer \"" + best + "\"?)"
}

func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}

// parsePageBlock: `página Nome` + body is `crie página Nome` + body.
func (p *Parser) parsePageBlock() error {
	lines := p.blockLinesWithHeader()
	name := lineText(lines[0])
	if _, err := p.layoutTree(lines, name); err != nil {
		return err
	}
	head := lines[0]
	head.toks = join(synth(head.toks[0], "crie"), head.toks)
	return p.withContext(name, func() error { return p.parseCrie(head, lines[1:]) })
}
