package parser

import (
	"strconv"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/lexer"
)

// The intent layer: lines that describe an application in human terms.
// Every phrase has one meaning; names are resolved in ResolveIntent.

func (p *Parser) intent() *ast.Intent {
	if p.program.Intent == nil {
		p.program.Intent = &ast.Intent{}
	}
	return p.program.Intent
}

// Filler words dropped from targets: "sair do grupo" → grupo.
var fillers = map[string]bool{"o": true, "a": true, "os": true, "as": true, "um": true, "uma": true,
	"do": true, "da": true, "dos": true, "das": true, "de": true, "no": true, "na": true, "nos": true, "nas": true}

var possessives = map[string]bool{"seu": true, "sua": true, "seus": true, "suas": true}

func wordsOf(t []lexer.Token) []string {
	out := make([]string, len(t))
	for i, x := range t {
		out[i] = strings.ToLower(foldWord(x.Name()))
	}
	return out
}

func foldWord(s string) string {
	return strings.NewReplacer("á", "a", "à", "a", "â", "a", "ã", "a", "é", "e", "ê", "e", "í", "i", "ó", "o", "ô", "o", "õ", "o", "ú", "u", "ç", "c").Replace(s)
}

// phrase joins target words into one name: "tokens de acesso" → tokens_de_acesso.
// Leading fillers are dropped; a leading possessive sets own.
func phrase(words []string) (name string, own bool) {
	i := 0
	for i < len(words) && (fillers[words[i]] || possessives[words[i]]) {
		if possessives[words[i]] {
			own = true
		}
		i++
	}
	return strings.Join(words[i:], "_"), own
}

// isIntentLine decides whether the column-1 line starting at the current
// token is an intent phrase. It never consumes tokens.
func (p *Parser) isIntentLine() bool {
	tok := p.current()
	if tok.Column != 1 {
		return false
	}
	var line []lexer.Token
	for i := p.pos; i < len(p.tokens) && p.tokens[i].Type != lexer.TokenNewline && p.tokens[i].Type != lexer.TokenEOF; i++ {
		if p.tokens[i].Type != lexer.TokenIndent {
			line = append(line, p.tokens[i])
		}
	}
	w := wordsOf(line)
	if len(w) == 0 {
		return false
	}
	switch w[0] {
	case "crie", "tenha", "permita", "disponibilize", "integracao", "somente":
		return true
	case "cada":
		return len(w) >= 3 && w[len(w)-1] == "tem"
	case "login":
		return len(w) >= 2
	case "ao":
		return len(w) == 2 && w[1] == "iniciar"
	case "quando":
		return len(w) >= 3 && !legacyTrigger[w[1]]
	}
	for _, x := range w[1:] {
		if x == "tem" || x == "pode" || x == "pertence" || x == "herda" {
			return true
		}
	}
	return false
}

var legacyTrigger = map[string]bool{"receber": true, "receive": true, "chamar": true, "call": true, "clicar": true, "click": true}

// parseIntentLine parses one intent phrase (with its block, if any).
func (p *Parser) parseIntentLine() error {
	lines := p.blockLinesWithHeader()
	head := lines[0]
	body := lines[1:]
	w := wordsOf(head.toks)
	in := p.intent()
	pos := p.at(head.toks[0])
	switch w[0] {
	case "crie":
		return p.parseCrie(head, body)
	case "tenha":
		return p.parseTenha(head, body)
	case "cada":
		// cada cliente tem
		name, _ := phrase(w[1 : len(w)-1])
		in.FieldBlocks = append(in.FieldBlocks, &ast.FieldsDecl{Entity: name, Lines: tokenLines(body), Pos: pos})
		return nil
	case "permita":
		return p.parsePermita(head, body)
	case "disponibilize":
		return p.parseDisponibilize(head)
	case "integracao":
		// integração em "/api/v4"
		if len(head.toks) != 3 || head.toks[2].Type != lexer.TokenString {
			return p.errorf(head.toks[0], `use: integração em "/api/v4"`)
		}
		in.IntegrationPrefix = strings.TrimRight(head.toks[2].Value, "/")
		return nil
	case "login":
		return p.parseLoginConfig(head)
	case "ao":
		stmts, err := p.statementsIn(body)
		in.Init = append(in.Init, stmts...)
		return err
	case "quando":
		verb := w[1]
		target, _ := phrase(w[2:])
		stmts, err := p.statementsIn(body)
		if err != nil {
			return err
		}
		if len(stmts) == 0 {
			return p.errorf(head.toks[0], "quando %s %s precisa de um bloco com o que deve acontecer", verb, target)
		}
		in.Hooks = append(in.Hooks, &ast.Hook{Verb: verb, Target: target, Body: stmts, Pos: pos})
		return nil
	case "somente":
		return p.parsePode(head, body, true)
	}
	for i, x := range w {
		switch x {
		case "pode":
			return p.parsePode(head, body, false)
		case "tem":
			subject, _ := phrase(w[:i])
			if i == len(w)-1 {
				in.FieldBlocks = append(in.FieldBlocks, &ast.FieldsDecl{Entity: subject, Lines: tokenLines(body), Pos: pos})
				return nil
			}
			// cliente tem pedidos / grupo tem membros com papel / grupo tem subgrupos
			in.FieldBlocks = append(in.FieldBlocks, &ast.FieldsDecl{Entity: subject, Lines: [][]lexer.Token{head.toks[i+1:]}, Pos: pos})
			if len(body) > 0 {
				return p.errorf(body[0].toks[0], "linha inesperada depois de %q", strings.Join(w, " "))
			}
			return nil
		case "pertence":
			// pedido pertence a cliente [opcional] [como dono]
			subject, _ := phrase(w[:i])
			rest := w[i+1:]
			if len(rest) > 0 && rest[0] == "a" {
				rest = rest[1:]
			}
			r := &ast.RelationDecl{Kind: "pertence", From: subject, Pos: pos}
			var target []string
			for k := 0; k < len(rest); k++ {
				switch rest[k] {
				case "opcional":
					r.Optional = true
				case "como":
					if k+1 < len(rest) {
						r.As = rest[k+1]
						k++
					}
				default:
					target = append(target, rest[k])
				}
			}
			r.To, _ = phrase(target)
			if r.To == "" {
				return p.errorf(head.toks[0], "use: %s pertence a <entidade>", subject)
			}
			in.Relations = append(in.Relations, r)
			return nil
		case "herda":
			// projeto herda membros do grupo
			subject, _ := phrase(w[:i])
			if i+1 >= len(w) || w[i+1] != "membros" {
				return p.errorf(head.toks[0], "use: %s herda membros do <entidade>", subject)
			}
			from, _ := phrase(w[i+2:])
			in.Memberships = append(in.Memberships, &ast.MembershipDecl{Entity: subject, InheritFrom: from, Pos: pos})
			return nil
		}
	}
	return p.errorf(head.toks[0], "frase não reconhecida: %q", strings.Join(w, " "))
}

func tokenLines(lines []dline) [][]lexer.Token {
	out := make([][]lexer.Token, 0, len(lines))
	for _, l := range lines {
		out = append(out, l.toks)
	}
	return out
}

// blockLinesWithHeader returns the header line (tokens up to the first
// newline) followed by the indented block lines.
func (p *Parser) blockLinesWithHeader() []dline {
	head := dline{indent: 0, from: p.pos}
	for !p.isAtEnd() && p.current().Type != lexer.TokenNewline {
		head.toks = append(head.toks, p.advance())
	}
	head.to = p.pos
	return append([]dline{head}, p.blockLines()...)
}

func (p *Parser) parseCrie(head dline, body []dline) error {
	w := wordsOf(head.toks)
	if len(w) < 3 {
		return p.errorf(head.toks[0], "use: crie sistema <Nome> ou crie página <Nome>")
	}
	switch w[1] {
	case "sistema", "app", "aplicacao":
		p.program.System = &ast.System{Name: head.toks[2].Name()}
		return nil
	case "pagina":
		pg := &ast.PageDecl{Pos: p.at(head.toks[0])}
		if w[2] == "para" && len(w) >= 5 && w[3] == "gerenciar" {
			pg.Manage, _ = phrase(w[4:])
			pg.Name = head.toks[4].Name()
			pg.Show = pg.Manage
			pg.Permits = []string{"pesquisar", "criar", "editar", "excluir"}
		} else {
			parts := make([]string, 0, len(head.toks)-2)
			for _, t := range head.toks[2:] {
				parts = append(parts, t.Name())
			}
			pg.Name = strings.Join(parts, " ")
		}
		for i := 0; i < len(body); i++ {
			bw := wordsOf(body[i].toks)
			switch {
			case bw[0] == "mostre":
				pg.Show, _ = phrase(bw[1:])
				for _, k := range children(body, i) {
					kw := wordsOf(k.toks)
					if len(kw) == 3 && kw[1] == "por" && kw[2] == "pagina" {
						pg.PerPage, _ = strconv.Atoi(kw[0])
					}
				}
				i += len(children(body, i))
			case bw[0] == "permita":
				verbs := bw[1:]
				for _, k := range children(body, i) {
					verbs = append(verbs, wordsOf(k.toks)...)
				}
				for _, v := range verbs {
					if v != "e" && v != "," {
						pg.Permits = append(pg.Permits, v)
					}
				}
				i += len(children(body, i))
			default:
				return p.errorf(body[i].toks[0], "na página use mostre <dados> e permita <ações>")
			}
		}
		p.intent().Pages = append(p.intent().Pages, pg)
		return nil
	}
	return p.errorf(head.toks[1], "use: crie sistema <Nome> ou crie página <Nome>")
}

func splitItems(t []lexer.Token) [][]string {
	var items [][]string
	var cur []string
	for _, x := range t {
		if x.Type == lexer.TokenComma || x.Type == lexer.TokenE {
			if len(cur) > 0 {
				items = append(items, cur)
			}
			cur = nil
			continue
		}
		cur = append(cur, strings.ToLower(foldWord(x.Name())))
	}
	if len(cur) > 0 {
		items = append(items, cur)
	}
	return items
}

func (p *Parser) parseTenha(head dline, body []dline) error {
	in := p.intent()
	items := splitItems(head.toks[1:])
	positions := []lexer.Token{}
	for range items {
		positions = append(positions, head.toks[0])
	}
	for i := 0; i < len(body); i++ {
		bw := wordsOf(body[i].toks)
		// tenha papeis + nested role names
		if len(bw) == 1 && (bw[0] == "papeis" || bw[0] == "papel") {
			level := 0
			for _, k := range children(body, i) {
				for _, it := range splitItems(k.toks) {
					level += 10
					in.Roles = append(in.Roles, &ast.Role{Name: strings.Join(it, "_"), Level: level, Pos: p.at(k.toks[0])})
				}
			}
			i += len(children(body, i))
			continue
		}
		for _, it := range splitItems(body[i].toks) {
			items = append(items, it)
			positions = append(positions, body[i].toks[0])
		}
	}
	for k, it := range items {
		name := strings.Join(it, "_")
		pos := p.at(positions[k])
		switch {
		case name == "login":
			if in.Login == nil {
				in.Login = &ast.LoginDecl{Pos: pos}
			}
		case name == "cadastro":
			if in.Login == nil {
				in.Login = &ast.LoginDecl{Pos: pos}
			}
			in.Login.Signup = true
		case it[0] == "papeis" || it[0] == "papel":
			level := 10 * len(in.Roles)
			for _, r := range it[1:] {
				level += 10
				in.Roles = append(in.Roles, &ast.Role{Name: r, Level: level, Pos: pos})
			}
		default:
			in.Entities = append(in.Entities, &ast.EntityDecl{Name: name, Pos: pos})
		}
	}
	return nil
}

func (p *Parser) parsePode(head dline, body []dline, only bool) error {
	w := wordsOf(head.toks)
	in := p.intent()
	i := 0
	if only {
		i = 1
	}
	k := i
	for k < len(w) && w[k] != "pode" {
		k++
	}
	if k == len(w) || k == i {
		return p.errorf(head.toks[0], "use: <papel> pode <ação> <dados>")
	}
	role, _ := phrase(w[i:k])
	add := func(ws []string, tok lexer.Token) error {
		if len(ws) < 1 {
			return p.errorf(tok, "falta a ação depois de pode")
		}
		verb := ws[0]
		target, own := phrase(ws[1:])
		in.Grants = append(in.Grants, &ast.Grant{Role: role, Only: only, Verb: verb, Target: target, Own: own, Pos: p.at(tok)})
		return nil
	}
	if k+1 < len(w) {
		if err := add(w[k+1:], head.toks[0]); err != nil {
			return err
		}
	}
	for _, l := range body {
		if err := add(wordsOf(l.toks), l.toks[0]); err != nil {
			return err
		}
	}
	if k+1 == len(w) && len(body) == 0 {
		return p.errorf(head.toks[0], "%s pode… o quê? Liste as ações abaixo, indentadas", role)
	}
	return nil
}

var commonVerbs = map[string]bool{"ver": true, "listar": true, "mostrar": true, "criar": true, "cadastrar": true, "adicionar": true,
	"editar": true, "alterar": true, "atualizar": true, "excluir": true, "remover": true, "apagar": true, "pesquisar": true, "buscar": true, "filtrar": true}

func (p *Parser) parsePermita(head dline, body []dline) error {
	w := wordsOf(head.toks)
	in := p.intent()
	i := 1
	var verbs []string
	for i < len(w) && (commonVerbs[w[i]] || w[i] == "e" || w[i] == ",") {
		if commonVerbs[w[i]] {
			verbs = append(verbs, w[i])
		}
		i++
	}
	// comma tokens are not words; rebuild using tokens to catch "cadastrar, editar"
	if len(verbs) == 0 {
		if len(w) < 3 {
			return p.errorf(head.toks[0], "use: permita <ação> <dados>, por exemplo permita criar projetos")
		}
		verbs = []string{w[1]}
		i = 2
	}
	rest := w[i:]
	var by []string
	for k, x := range rest {
		if x == "por" {
			by = rest[k+1:]
			rest = rest[:k]
			break
		}
	}
	target, _ := phrase(rest)
	for _, l := range body {
		for _, it := range splitItems(l.toks) {
			by = append(by, strings.Join(it, "_"))
		}
	}
	var fields []string
	for _, b := range by {
		if b != "e" {
			fields = append(fields, b)
		}
	}
	for _, v := range verbs {
		in.Permits = append(in.Permits, &ast.Permit{Verb: v, Target: target, By: fields, Pos: p.at(head.toks[0])})
	}
	return nil
}

func (p *Parser) parseDisponibilize(head dline) error {
	w := wordsOf(head.toks)
	in := p.intent()
	as := ""
	end := len(w)
	for k, x := range w {
		if x == "para" {
			end = k
		}
		if x == "como" && k+1 < len(head.toks) && head.toks[k+1].Type == lexer.TokenString {
			as = head.toks[k+1].Value
		}
	}
	if end == len(w) {
		return p.errorf(head.toks[0], `use: disponibilize <dados> para integração [como "nome"]`)
	}
	for _, it := range splitItems(head.toks[1:end]) {
		name, _ := phrase(it)
		in.Integrations = append(in.Integrations, &ast.Integration{Target: name, As: as, Pos: p.at(head.toks[0])})
	}
	return nil
}

func (p *Parser) parseLoginConfig(head dline) error {
	w := wordsOf(head.toks)
	in := p.intent()
	if in.Login == nil {
		in.Login = &ast.LoginDecl{Pos: p.at(head.toks[0])}
	}
	l := in.Login
	num := func(k int) int {
		if k < len(head.toks) && head.toks[k].Type == lexer.TokenNumber {
			n, _ := strconv.Atoi(head.toks[k].Value)
			return n
		}
		return -1
	}
	switch {
	case w[1] == "usa":
		l.Fields = nil
		for _, it := range splitItems(head.toks[2:]) {
			l.Fields = append(l.Fields, strings.Join(it, "_"))
		}
		return nil
	case w[1] == "aceita" && len(w) >= 3 && w[2] == "oauth":
		// login aceita oauth por 2 horas
		n := num(4)
		if n <= 0 || len(w) < 6 {
			return p.errorf(head.toks[0], "use: login aceita oauth por 2 horas")
		}
		l.OAuthSeconds = n * unitSeconds(w[5])
		return nil
	case w[1] == "aceita":
		// login aceita tokens de acesso [no cabeçalho "PRIVATE-TOKEN"]
		end := len(w)
		for k, x := range w {
			if x == "no" && k+1 < len(w) && w[k+1] == "cabecalho" {
				end = k
				if k+2 < len(head.toks) && head.toks[k+2].Type == lexer.TokenString {
					l.TokenHeader = head.toks[k+2].Value
				}
			}
		}
		l.TokenEntity, _ = phrase(w[2:end])
		return nil
	case w[1] == "bloqueia":
		// login bloqueia após 10 tentativas por 10 minutos
		l.LockAttempts = num(3)
		l.LockMinutes = num(6)
		if l.LockAttempts <= 0 || l.LockMinutes <= 0 {
			return p.errorf(head.toks[0], "use: login bloqueia após 10 tentativas por 10 minutos")
		}
		return nil
	case w[1] == "exige" && len(head.toks) == 4:
		// login exige state "active"
		l.ActiveField = head.toks[2].Name()
		switch head.toks[3].Type {
		case lexer.TokenString:
			l.ActiveValue = head.toks[3].Value
		case lexer.TokenVerdadeiro, lexer.TokenFalso:
			l.ActiveValue = head.toks[3].Type == lexer.TokenVerdadeiro
		default:
			return p.errorf(head.toks[3], "login exige <campo> <valor>")
		}
		return nil
	}
	return p.errorf(head.toks[0], "login: use usa, aceita, bloqueia ou exige")
}

func unitSeconds(u string) int {
	switch strings.TrimSuffix(u, "s") {
	case "minuto":
		return 60
	case "hora":
		return 3600
	case "dia":
		return 86400
	}
	return 1
}
