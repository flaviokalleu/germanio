package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/diagnostics"
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
	case "mensagens":
		return len(w) == 3 && w[1] == "em"
	case "vocabulario":
		return true
	case "traduza":
		return len(w) >= 4 && w[len(w)-2] == "com"
	case "enderecos":
		return len(w) >= 2 && w[1] == "reservados"
	case "escopo":
		return len(w) >= 4
	case "quem":
		return len(w) >= 5 && w[1] == "cria"
	case "cada":
		return len(w) >= 3 && w[len(w)-1] == "tem"
	case "login":
		return len(w) >= 2
	case "ao":
		return len(w) == 2 && w[1] == "iniciar"
	case "antes":
		return len(w) >= 4 && w[1] == "de"
	case "renomeie", "descarte":
		return len(w) >= 4
	case "quando":
		return len(w) >= 3 && !legacyTrigger[w[1]]
	}
	if len(w) >= 4 && w[len(w)-1] == "final" && w[len(w)-2] == "e" {
		return true
	}
	if len(w) >= 5 && w[len(w)-2] == "somente" && w[len(w)-1] == "leitura" && w[len(w)-3] == "e" {
		return true
	}
	for i, x := range w[1:] {
		if x == "tem" || x == "pode" || x == "podem" || x == "pertence" || x == "herda" || x == "comeca" || x == "recebe" || x == "executa" || x == "executam" || x == "precisa" || x == "gera" {
			return true
		}
		if (x == "usa" || x == "usam") && len(w) >= 5 && (contains(w, "do") || contains(w, "da") || contains(w, "dos") || contains(w, "das")) {
			return true // pipelines usam as variaveis do projeto (GEP 0015)
		}
		if x == "guarda" && i+2 == len(w)-1 && w[i+2] == "historico" {
			return true // issue guarda histórico (GEP 0011)
		}
	}
	return false
}

var legacyTrigger = map[string]bool{"receber": true, "receive": true, "chamar": true, "call": true, "clicar": true, "click": true}

// parseIntentLine parses one intent phrase (with its block, if any).
func (p *Parser) parseIntentLine() error {
	lines := p.blockLinesWithHeader()
	if _, err := p.layoutTree(lines, ""); err != nil {
		return err
	}
	return p.intentFrom(lines[0], lines[1:])
}

// intentFrom interprets one flat phrase (header + indented lines). Data
// blocks reduce their sections to calls of this function.
func (p *Parser) intentFrom(head dline, body []dline) error {
	w := wordsOf(head.toks)
	in := p.intent()
	pos := p.at(head.toks[0])
	switch w[0] {
	case "renomeie":
		// renomeie nome de clientes para nome_completo
		k, d := -1, -1
		for i, x := range w {
			if x == "de" && d < 0 {
				d = i
			}
			if x == "para" {
				k = i
			}
		}
		if d < 2 || k < d+2 || k+1 >= len(w) {
			return p.teach(head.toks[0], "\""+lineText(head)+"\" não diz o que renomear", "um rename diz o campo antigo, o dado e o nome novo", "escreva: renomeie nome de clientes para nome_completo (ou, no bloco do dado: renomeie nome para nome_completo)", "")
		}
		from, _ := phrase(w[1:d])
		entity, _ := phrase(w[d+1 : k])
		to, _ := phrase(w[k+1:])
		if from == "" || entity == "" || to == "" {
			return p.teach(head.toks[0], "\""+lineText(head)+"\" não diz o que renomear", "um rename diz o campo antigo, o dado e o nome novo", "escreva: renomeie nome de clientes para nome_completo", "")
		}
		in.Renames = append(in.Renames, &ast.RenameDecl{Entity: entity, From: from, To: to, Pos: pos})
		return nil
	case "descarte":
		// descarte telefone de clientes
		d := -1
		for i, x := range w {
			if x == "de" {
				d = i
			}
		}
		if d < 2 || d+1 >= len(w) {
			return p.teach(head.toks[0], "\""+lineText(head)+"\" não diz o que descartar", "descarte diz qual campo foi removido de propósito e de qual dado", "escreva: descarte telefone de clientes (ou, no bloco do dado: descarte telefone)", "")
		}
		from, _ := phrase(w[1:d])
		entity, _ := phrase(w[d+1:])
		if from == "" || entity == "" {
			return p.teach(head.toks[0], "\""+lineText(head)+"\" não diz o que descartar", "descarte diz qual campo foi removido de propósito e de qual dado", "escreva: descarte telefone de clientes", "")
		}
		in.Discards = append(in.Discards, &ast.RenameDecl{Entity: entity, From: from, Pos: pos})
		return nil
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
		if len(head.toks) == 1 {
			return p.eachItem(head, body, p.parsePermita)
		}
		return p.parsePermita(head, body)
	case "disponibilize":
		if len(w) >= 2 && w[1] == "para" {
			// disponibilize para integração + itens `dados [como "nome"]`
			for _, l := range body {
				synthetic := dline{toks: append(append([]lexer.Token{head.toks[0]}, l.toks...), head.toks[1:]...)}
				if err := p.parseDisponibilize(synthetic); err != nil {
					return err
				}
			}
			return nil
		}
		return p.parseDisponibilize(head)
	case "integracao":
		// integração em "/api/v1"
		if len(head.toks) != 3 || head.toks[2].Type != lexer.TokenString {
			return p.errorf(head.toks[0], `use: integração em "/api/v1"`)
		}
		in.IntegrationPrefix = strings.TrimRight(head.toks[2].Value, "/")
		return nil
	case "enderecos":
		// endereços reservados + one name per line (or comma-separated)
		lines := append([][]lexer.Token{head.toks[2:]}, tokenLines(body)...)
		for _, l := range lines {
			for _, t := range l {
				if t.Type == lexer.TokenComma {
					continue
				}
				v := t.Value
				if t.Type != lexer.TokenString {
					v = t.Name()
				}
				in.ReservedAddresses = append(in.ReservedAddresses, strings.ToLower(v))
			}
		}
		return nil
	case "traduza":
		// traduza arquivos de execução com ler_formato (adapters, integracoes/)
		k := len(w) - 2
		if len(w) < 4 || w[k] != "com" {
			return p.errorf(head.toks[0], "use: traduza <arquivos de execução | variáveis das etapas> com <função>")
		}
		point, _ := phrase(w[1:k])
		if in.Translators == nil {
			in.Translators = map[string]string{}
		}
		in.Translators[point] = head.toks[len(head.toks)-1].Name()
		return nil
	case "login":
		return p.parseLoginConfig(head)
	case "quem":
		// quem cria grupo vira owner
		k := len(w) - 2
		if w[k] != "vira" {
			return p.errorf(head.toks[0], "use: quem cria <dado> vira <papel>")
		}
		target, _ := phrase(w[2:k])
		in.Creators = append(in.Creators, &ast.CreatorRole{Entity: target, Role: w[k+1], Pos: pos})
		return nil
	case "escopo":
		// escopo "read_api" permite ler
		if len(head.toks) < 4 || head.toks[1].Type != lexer.TokenString || w[2] != "permite" {
			return p.errorf(head.toks[0], `use: escopo "nome" permite tudo|ler|escrever|baixar código|enviar código`)
		}
		if in.Login == nil {
			in.Login = &ast.LoginDecl{Pos: pos}
		}
		if in.Login.Scopes == nil {
			in.Login.Scopes = map[string][]string{}
		}
		name := head.toks[1].Value
		for _, it := range splitItems(head.toks[3:]) {
			perm := strings.Join(it, "_")
			switch perm {
			case "tudo", "ler", "escrever", "baixar_codigo", "enviar_codigo":
				in.Login.Scopes[name] = append(in.Login.Scopes[name], perm)
			default:
				return p.errorf(head.toks[0], "permissão de escopo desconhecida %q (use tudo, ler, escrever, baixar código, enviar código)", perm)
			}
		}
		return nil
	case "vocabulario":
		// vocabulário da integração + linhas `nome é "externo"`
		if in.Vocabulary == nil {
			in.Vocabulary = map[string]string{}
		}
		for _, l := range body {
			t := l.toks
			if len(t) != 3 || t[1].Type != lexer.TokenE || t[2].Type != lexer.TokenString {
				return p.errorf(t[0], `use: <nome> é "nome externo"`)
			}
			k := strings.ToLower(foldWord(t[0].Name()))
			if old, ok := in.Vocabulary[k]; ok && old != t[2].Value {
				return p.teach(t[0], fmt.Sprintf("%s já é traduzido como %q (linha %d)", k, old, in.VocabularyPos[k].Line), "um nome tem uma só tradução na integração; a segunda substituiria a primeira em silêncio", fmt.Sprintf("fique com uma: %s é %q", k, old), "")
			}
			in.Vocabulary[k] = t[2].Value
			if in.VocabularyPos == nil {
				in.VocabularyPos = map[string]diagnostics.Position{}
			}
			in.VocabularyPos[k] = p.at(t[0])
		}
		return nil
	case "mensagens":
		if len(w) != 3 || w[1] != "em" {
			break // a data called mensagens (mensagens tem …), not the language of messages
		}
		switch w[2] {
		case "ingles", "english":
			in.Messages = "en"
		case "portugues":
			in.Messages = "pt"
		default:
			return p.errorf(head.toks[2], "use: mensagens em português ou mensagens em inglês")
		}
		return nil
	case "ao":
		stmts, err := p.statementsIn(body)
		in.Init = append(in.Init, stmts...)
		return err
	case "quando", "antes":
		before := w[0] == "antes"
		rest := w[1:]
		if before {
			rest = w[2:]
		}
		verb := rest[0]
		targetWords := rest[1:]
		// "enviar código para projeto" → verb enviar_codigo
		if (verb == "enviar" || verb == "baixar") && len(targetWords) > 0 && targetWords[0] == "codigo" {
			verb, targetWords = verb+"_codigo", targetWords[1:]
			if len(targetWords) > 0 && (targetWords[0] == "para" || targetWords[0] == "em") {
				targetWords = targetWords[1:]
			}
		}
		target, _ := phrase(targetWords)
		stmts, err := p.statementsIn(body)
		if err != nil {
			return err
		}
		if len(stmts) == 0 {
			return p.errorf(head.toks[0], "quando %s %s precisa de um bloco com o que deve acontecer", verb, target)
		}
		in.Hooks = append(in.Hooks, &ast.Hook{Before: before, Verb: verb, Target: target, Body: stmts, Pos: pos})
		return nil
	case "somente":
		return p.parsePode(head, body, true)
	}
	// projeto arquivado é somente leitura
	if len(w) >= 5 && w[len(w)-2] == "somente" && w[len(w)-1] == "leitura" && w[len(w)-3] == "e" {
		subject, _ := phrase(w[:len(w)-4])
		in.ReadOnly = append(in.ReadOnly, &ast.VisibilityRule{Entity: subject, Flag: w[len(w)-4], Pos: pos})
		return nil
	}
	// merge request mesclado é final
	if len(w) >= 4 && w[len(w)-1] == "final" && w[len(w)-2] == "e" {
		subject, _ := phrase(w[:len(w)-3])
		in.Finals = append(in.Finals, &ast.StateDecl{Entity: subject, Initial: w[len(w)-3], Pos: pos})
		return nil
	}
	// X não pode ser mais visível que [o] Y
	for i := 0; i+5 < len(w); i++ {
		if w[i] == "nao" && w[i+1] == "pode" && w[i+2] == "ser" && w[i+3] == "mais" && strings.HasPrefix(w[i+4], "visive") && w[i+5] == "que" {
			subject, _ := phrase(w[:i])
			parent, _ := phrase(w[i+6:])
			in.Ceilings = append(in.Ceilings, &ast.VisibilityCeiling{Entity: subject, Parent: parent, Pos: pos})
			return nil
		}
	}
	// X [condição] pode ser vista por ...
	for i := 0; i+3 < len(w)+1; i++ {
		if i+3 < len(w) && (w[i] == "pode" || w[i] == "podem") && w[i+1] == "ser" && strings.HasPrefix(w[i+2], "vist") && w[i+3] == "por" {
			rule := &ast.VisibilityRule{Entity: strings.Join(w[:i], " "), Pos: pos}
			for _, it := range splitItems(head.toks[i+4:]) {
				rule.Who = append(rule.Who, strings.Join(it, "_"))
			}
			for _, l := range body {
				for _, it := range splitItems(l.toks) {
					rule.Who = append(rule.Who, strings.Join(it, "_"))
				}
			}
			in.Visibility = append(in.Visibility, rule)
			return nil
		}
	}
	if n := len(w); n >= 3 && w[n-2] == "guarda" && w[n-1] == "historico" {
		// issue guarda histórico (GEP 0011, em teste)
		subject, _ := phrase(w[:n-2])
		if subject == "" {
			return p.teach(head.toks[0], "\""+lineText(head)+"\" não diz de qual dado", "guarda histórico vem depois do dado", "escreva, por exemplo: issue guarda histórico (ou, no bloco do dado: guarda histórico)", "")
		}
		p.intent().History = append(p.intent().History, &ast.HistoryDecl{Entity: subject, Pos: pos})
		return nil
	}
	for i, x := range w {
		switch x {
		case "gera":
			// issue gera pendência para responsaveis (GEP 0009, em teste)
			subject, _ := phrase(w[:i])
			if subject == "" || i+3 >= len(w) || w[i+1] != "pendencia" || w[i+2] != "para" {
				return p.teach(head.toks[0], "\""+lineText(head)+"\" não é uma frase conhecida", "gera só aparece em: <dado> gera pendência para <pessoas>", "escreva, por exemplo: issue gera pendência para responsaveis", "")
			}
			var fields []string
			for _, f := range w[i+3:] {
				if f != "e" && f != "," {
					fields = append(fields, f)
				}
			}
			p.intent().PendingItems = append(p.intent().PendingItems, &ast.PendingRule{Entity: subject, Fields: fields, Pos: pos})
			return nil
		case "precisa":
			// todo grupo precisa ter pelo menos um owner
			k := len(w) - 1
			subject, _ := phrase(w[:i])
			subject = strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(subject, "todo_"), "toda_"), "cada_")
			if subject == "" || k <= i+4 || strings.Join(w[i+1:k], " ") != "ter pelo menos um" && strings.Join(w[i+1:k], " ") != "ter pelo menos uma" {
				return p.errorf(head.toks[0], "use: todo <dado> precisa ter pelo menos um <papel>")
			}
			in.MinRoles = append(in.MinRoles, &ast.CreatorRole{Entity: subject, Role: w[k], Pos: pos})
			return nil
		case "executam":
			// runners executam jobs: records with a credential run steps elsewhere
			who, _ := phrase(w[:i])
			steps, _ := phrase(w[i+1:])
			if who == "" || steps == "" {
				return p.errorf(head.toks[0], "use: <quem> executam <etapas>, por exemplo runners executam jobs")
			}
			in.RemoteExecutors = append(in.RemoteExecutors, &ast.RemoteExecutorDecl{Executor: who, Steps: steps, Pos: pos})
			return nil
		case "usa", "usam":
			// pipelines usam as variaveis do projeto (GEP 0015, em teste)
			runs, _ := phrase(w[:i])
			rest := w[i+1:]
			j := -1
			for k, x := range rest {
				if x == "do" || x == "da" || x == "dos" || x == "das" {
					j = k
				}
			}
			var data, owner string
			if j > 0 {
				data, _ = phrase(rest[:j])
				owner, _ = phrase(rest[j+1:])
			}
			if runs == "" || data == "" || owner == "" {
				return p.teach(head.toks[0], "\""+lineText(head)+"\" não diz quais variáveis nem de quem", "as execuções usam as variáveis de quem as executa", "escreva: pipelines usam as variaveis do projeto", "")
			}
			in.RunVariables = append(in.RunVariables, &ast.RunVariablesDecl{Runs: runs, Data: data, Owner: owner, Pos: pos})
			return nil
		case "executa":
			// projeto executa pipelines a cada envio de código conforme "pipeline.yml"
			owner, _ := phrase(w[:i])
			k := i + 1
			for k < len(w) && w[k] != "a" {
				k++
			}
			runs, _ := phrase(w[i+1 : k])
			file := ""
			for _, t := range head.toks {
				if t.Type == lexer.TokenString {
					file = t.Value
				}
			}
			if file == "" || !strings.Contains(strings.Join(w, " "), "envio de codigo") {
				return p.errorf(head.toks[0], `use: <dado> executa <execuções> a cada envio de código conforme "arquivo"`)
			}
			in.Executions = append(in.Executions, &ast.ExecutionDecl{Owner: owner, Entity: runs, File: file, Pos: pos})
			return nil
		case "recebe":
			subject, _ := phrase(w[:i])
			// webhook recebe eventos do projeto + tipos de evento
			if i+1 < len(w) && w[i+1] == "eventos" {
				owner, _ := phrase(w[i+2:])
				sub := &ast.SubscriptionDecl{Subscriber: subject, Owner: owner, Pos: pos}
				for _, l := range body {
					kw := wordsOf(l.toks)
					if len(kw) >= 2 && kw[0] == "enviar" && kw[1] == "codigo" {
						sub.Kinds = append(sub.Kinds, "enviar_codigo")
						continue
					}
					k, _ := phrase(kw)
					sub.Kinds = append(sub.Kinds, k)
				}
				if len(sub.Kinds) == 0 {
					return p.errorf(head.toks[0], "liste os eventos abaixo, por exemplo: enviar código, issues")
				}
				in.Subscriptions = append(in.Subscriptions, sub)
				return nil
			}
			// merge request recebe aprovações
			if i != len(w)-2 || !strings.HasPrefix(w[i+1], "aprovac") {
				return p.errorf(head.toks[0], "use: <dado> recebe aprovações")
			}
			in.Approvals = append(in.Approvals, subject)
			return nil
		case "comeca":
			// issue começa aberta
			if i == 0 || i != len(w)-2 {
				return p.errorf(head.toks[0], "use: <dado> começa <estado>, por exemplo pedido começa aberto")
			}
			subject, _ := phrase(w[:i])
			in.States = append(in.States, &ast.StateDecl{Entity: subject, Initial: w[i+1], Pos: pos})
			return nil
		case "pode", "podem":
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
						continue
					}
					return p.teach(k.toks[0], "\""+lineText(k)+"\" não é algo que mostre aceite",
						"abaixo de mostre só cabe quantos registros aparecem por página", "escreva, por exemplo: 20 por página", "")
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
			case pageSections[bw[0]] && len(bw) == 1:
				last, err := p.pageSection(pg, body, i)
				if err != nil {
					return err
				}
				i = last
			case len(bw) == 3 && bw[1] == "por" && bw[2] == "pagina":
				return p.teach(body[i].toks[0], "\""+lineText(body[i])+"\" está no nível da página",
					"quantos registros aparecem por página é uma propriedade do que a página mostra",
					"escreva a linha recuada abaixo de mostre:\n    mostre <dados>\n        "+lineText(body[i]), "")
			default:
				return p.teach(body[i].toks[0], "\""+lineText(body[i])+"\" não é uma parte de página",
					"uma página diz o que mostra e o que permite fazer", "use mostre <dados> e permita <ações>", "")
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
	// tenha busca geral em projetos, issues e merge requests
	if w := wordsOf(head.toks); len(w) >= 5 && w[1] == "busca" && w[2] == "geral" && w[3] == "em" {
		for _, it := range splitItems(head.toks[4:]) {
			name, _ := phrase(it)
			in.GlobalSearch = append(in.GlobalSearch, name)
		}
		return nil
	}
	// tenha administrador inicial "root"
	if w := wordsOf(head.toks); len(w) >= 3 && w[1] == "administrador" && w[2] == "inicial" {
		if len(head.toks) != 4 || head.toks[3].Type != lexer.TokenString {
			return p.errorf(head.toks[0], `use: tenha administrador inicial "nome de login"`)
		}
		in.InitialAdmin = head.toks[3].Value
		return nil
	}
	// tenha avisos por e-mail (GEP 0013, em teste): read whole, the "e" of
	// e-mail is not the conjunction of a list
	if w := wordsOf(head.toks); len(w) >= 4 && w[1] == "avisos" && w[2] == "por" {
		switch strings.Join(w[3:], "") {
		case "e-mail", "email":
			in.EmailNotices = true
			in.EmailNoticesPos = p.at(head.toks[0])
			return nil
		}
	}
	items := splitItems(head.toks[1:])
	// tenha papeis + indented list of roles
	if len(items) == 1 && len(items[0]) == 1 && (items[0][0] == "papeis" || items[0][0] == "papel") && len(body) > 0 {
		wrapped := append([]dline{{indent: body[0].indent - 1, toks: head.toks[1:2]}}, body...)
		return p.parseTenha(dline{toks: head.toks[:1]}, wrapped)
	}
	// tenha login / tenha cadastro take nothing else on the line: "tenha login
	// com email e senha" would otherwise declare data named login_com_email
	// and senha (G84).
	for _, it := range items {
		if (it[0] == "login" || it[0] == "cadastro") && len(it) > 1 {
			return p.teach(head.toks[0], "\""+lineText(head)+"\" mistura a declaração do "+it[0]+" com outras palavras",
				"tenha "+it[0]+" é uma frase completa; o resto da linha viraria nomes de dados",
				"escreva tenha "+it[0]+" sozinho, e diga como se entra em outra linha: login usa email", "")
		}
	}
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
					// "guest 10" sets the level explicitly; otherwise levels
					// grow by 10 in declaration order.
					if n, err := strconv.Atoi(it[len(it)-1]); err == nil && len(it) > 1 {
						if n <= level {
							return p.errorf(k.toks[0], "papéis vão do menor para o maior: %d deve ser maior que %d", n, level)
						}
						level = n
						it = it[:len(it)-1]
					} else {
						level += 10
					}
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
		case name == "presenca":
			// tenha presença (GEP 0021, em teste)
			in.Presence = true
		case name == "recuperacao_de_senha":
			// tenha recuperação de senha (GEP 0008, em teste)
			if in.Login == nil {
				in.Login = &ast.LoginDecl{Pos: pos}
			}
			in.Login.Recovery = true
		case it[0] == "papeis" || it[0] == "papel":
			level := 10 * len(in.Roles)
			for _, r := range it[1:] {
				level += 10
				in.Roles = append(in.Roles, &ast.Role{Name: r, Level: level, Pos: pos})
			}
		default:
			if !validName(name) {
				return p.teach(positions[k], fmt.Sprintf("%q não pode ser o nome de um dado", strings.ReplaceAll(name, "_", " ")), "o nome de um dado tem só letras, números e espaços (lista separada por vírgula ou \"e\")", "escreva, por exemplo: tenha clientes e pedidos", "")
			}
			in.Entities = append(in.Entities, &ast.EntityDecl{Name: name, Pos: pos})
		}
	}
	return nil
}

// validName: a data name is letters, digits and underscores (the spaces of
// a multi-word name), starting with a letter.
func validName(name string) bool {
	for i, r := range name {
		letter := r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r > 127
		if !letter && (i == 0 || r < '0' || r > '9') {
			return false
		}
	}
	return name != ""
}

func (p *Parser) parsePode(head dline, body []dline, only bool) error {
	w := wordsOf(head.toks)
	in := p.intent()
	i := 0
	if only {
		i = 1
	}
	k := i
	for k < len(w) && w[k] != "pode" && w[k] != "podem" {
		k++
	}
	if k == len(w) || k == i {
		return p.errorf(head.toks[0], "use: <papel> pode <ação> <dados>")
	}
	// repositório do projeto pode começar com "README.md" contendo "# {nome}"
	if !only && k+1 < len(w) && w[i] == "repositorio" && w[k+1] == "comecar" {
		var strs []string
		for _, t := range head.toks {
			if t.Type == lexer.TokenString {
				strs = append(strs, t.Value)
			}
		}
		entity, _ := phrase(w[i+1 : k])
		if entity == "" || len(strs) != 2 {
			return p.errorf(head.toks[0], `use: repositório do <dado> pode começar com "arquivo" contendo "texto"`)
		}
		in.InitialFiles = append(in.InitialFiles, &ast.InitialFileDecl{Entity: entity, Path: strs[0], Content: strs[1], Pos: p.at(head.toks[0])})
		return nil
	}
	// runner pode pertencer a projeto: an optional parent
	if !only && k+1 < len(w) && w[k+1] == "pertencer" {
		subject, _ := phrase(w[i:k])
		rest := w[k+2:]
		if len(rest) > 0 && rest[0] == "a" {
			rest = rest[1:]
		}
		to, _ := phrase(rest)
		if to == "" {
			return p.errorf(head.toks[0], "use: %s pode pertencer a <dado>", subject)
		}
		in.Relations = append(in.Relations, &ast.RelationDecl{Kind: "pertence", From: subject, To: to, Optional: true, Pos: p.at(head.toks[0])})
		return nil
	}
	// "autor ou planner pode" grants the same actions to each role.
	var roles []string
	cur := []string{}
	for _, x := range w[i:k] {
		if x == "ou" {
			if len(cur) > 0 {
				r, _ := phrase(cur)
				roles = append(roles, r)
			}
			cur = nil
			continue
		}
		cur = append(cur, x)
	}
	if len(cur) > 0 {
		r, _ := phrase(cur)
		roles = append(roles, r)
	}
	for _, role := range roles {
		if err := p.grantsFor(in, role, only, head, body, k, w); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) grantsFor(in *ast.Intent, role string, only bool, head dline, body []dline, k int, w []string) error {
	add := func(ws []string, tok lexer.Token) error {
		if len(ws) < 1 {
			return p.errorf(tok, "falta a ação depois de pode")
		}
		verb := ws[0]
		rest := ws[1:]
		if (verb == "enviar" || verb == "baixar") && len(rest) > 0 && rest[0] == "codigo" {
			verb, rest = verb+"_codigo", rest[1:]
			if len(rest) > 0 && (rest[0] == "para" || rest[0] == "em") {
				rest = rest[1:]
			}
		}
		target, own := phrase(rest)
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
		if b != "e" && b != "," {
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
		if x == "para" && end == len(w) {
			end = k
		}
		if x == "como" && k < end {
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

// eachItem applies fn to every top-level line of body as if it followed
// the header word (permita + indented list).
func (p *Parser) eachItem(head dline, body []dline, fn func(dline, []dline) error) error {
	for i := 0; i < len(body); i++ {
		kids := children(body, i)
		synthetic := dline{toks: append([]lexer.Token{head.toks[0]}, body[i].toks...)}
		if err := fn(synthetic, kids); err != nil {
			return err
		}
		i += len(kids)
	}
	return nil
}

func contains(list []string, x string) bool {
	for _, v := range list {
		if v == x {
			return true
		}
	}
	return false
}
