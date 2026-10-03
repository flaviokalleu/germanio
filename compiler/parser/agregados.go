package parser

import (
	"sort"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/lexer"
)

// Aggregates (docs/gep/0047-agregados.md, em teste): a record shows numbers
// about what belongs to it, the same family as the indicators of a page
// (GEP 0012):
//
//	milestones
//	    indicadores
//	        total de issues abertas
//	        soma do peso das issues como peso total
//
// flat: `milestone mostra soma do peso das issues como peso total`.
//
// Pairs (docs/gep/0048-pares.md, em teste): `ligacoes › regras › única por
// par de issues` (flat: `ligacao é única por par de issues`) — a record that
// links two records of one data never links one to itself, and each pair,
// in any order, is linked once.

const aggregateExample = "total de issues abertas, ou soma do peso das issues como peso total"

// connector: the little words between the parts of a phrase (do peso das issues).
func connector(w string) bool {
	return w == "de" || w == "do" || w == "da" || w == "dos" || w == "das"
}

// aggregateLine reads `<dado> mostra (total de … | soma do … dos …) [como nome]`;
// i is the position of "mostra".
func (p *Parser) aggregateLine(head dline, i int) error {
	w := wordsOf(head.toks)
	subject, _ := phrase(w[:i])
	toks, rest := head.toks[i+1:], w[i+1:]
	for len(rest) > 0 && (rest[0] == "a" || rest[0] == "o") { // mostra a soma, mostra o total
		toks, rest = toks[1:], rest[1:]
	}
	if subject == "" {
		return p.teach(head.toks[0], "\""+lineText(head)+"\" não diz de qual dado é o número",
			"mostra vem depois do dado: o registro mostra um número sobre o que pertence a ele",
			"escreva, por exemplo: milestone mostra soma do peso das issues (ou, no bloco do dado: indicadores › soma do peso das issues)", "")
	}
	for _, t := range toks {
		if t.Type == lexer.TokenString {
			return p.teach(t, "\""+t.Value+"\" é um rótulo, mas um número do registro tem um nome",
				"o número aparece no registro com um nome, como os outros campos; o rótulo das páginas vem desse nome",
				"dê o nome com como: "+strings.Join(rest[:len(rest)-1], " ")+" como "+strings.ToLower(t.Value), "")
		}
	}
	d := &ast.AggregateDecl{Entity: subject, Pos: p.at(head.toks[0])}
	for k := len(rest) - 1; k >= 0; k-- {
		if rest[k] != "como" {
			continue
		}
		d.Name, _ = phrase(rest[k+1:])
		if d.Name == "" {
			return p.teach(toks[k], "falta o nome depois de como", "como dá o nome com que o número aparece no registro", "escreva, por exemplo: soma do peso das issues como peso total", "")
		}
		toks, rest = toks[:k], rest[:k]
		break
	}
	switch {
	case len(rest) >= 3 && rest[0] == "total" && rest[1] == "de":
		d.Words = rest[2:]
	case len(rest) >= 4 && rest[0] == "soma" && connector(rest[1]):
		d.Sum, d.Words = true, rest[2:]
	default:
		return p.teach(head.toks[0], "\""+lineText(dline{toks: toks})+"\" não é um indicador",
			"um indicador do registro conta (total de) ou soma (soma de) os registros que pertencem a ele",
			"escreva: "+aggregateExample, "")
	}
	d.Line = lineText(dline{toks: toks})
	p.intent().Aggregates = append(p.intent().Aggregates, d)
	return nil
}

// pairAt: the position of única/único in `… única por par de <dado>`, or -1.
func pairAt(w []string) int {
	for i := 0; i+4 < len(w); i++ {
		if (w[i] == "unica" || w[i] == "unico") && w[i+1] == "por" && w[i+2] == "par" && connector(w[i+3]) {
			return i
		}
	}
	return -1
}

// pairLine reads `[cada] <dado> [é] única por par de <dado>`.
func (p *Parser) pairLine(head dline, k int) error {
	w := wordsOf(head.toks)
	sub := w[:k]
	if n := len(sub); n > 0 && sub[n-1] == "e" { // é
		sub = sub[:n-1]
	}
	if len(sub) > 0 && (sub[0] == "cada" || sub[0] == "toda" || sub[0] == "todo") {
		sub = sub[1:]
	}
	subject, _ := phrase(sub)
	of, _ := phrase(w[k+4:])
	if subject == "" || of == "" {
		return p.teach(head.toks[0], "\""+lineText(head)+"\" não diz qual dado liga qual",
			"única por par diz que um registro liga dois registros de um dado, nunca um a ele mesmo, e cada par uma vez só",
			"escreva: ligacao é única por par de issues (ou, no bloco do dado: regras › única por par de issues)", "")
	}
	p.intent().Pairs = append(p.intent().Pairs, &ast.PairDecl{Entity: subject, Of: of, Pos: p.at(head.toks[0])})
	return nil
}

// numericField: a field whose values can be added up.
func numericField(f *ast.Field) bool {
	switch f.Type {
	case ast.FieldNumero, ast.FieldInteiro, ast.FieldDinheiro, ast.FieldPercentual, ast.FieldEstrelas:
		return true
	}
	return false
}

// viaField: the field of c pointing at e (the owner's own reference first:
// issue_id), or "" when c does not belong to e; ambiguous lists every
// candidate when there is no owner reference among several.
func viaField(c, e *ast.Entity) (string, []string) {
	var fields []string
	for f, t := range c.Parents {
		if t == e.Singular {
			fields = append(fields, f)
		}
	}
	sort.Strings(fields)
	if c.Parents[e.Singular+"_id"] == e.Singular {
		return e.Singular + "_id", nil
	}
	if len(fields) == 1 {
		return fields[0], nil
	}
	return "", fields
}

// childOf resolves `<dado> [estado]` to data that belongs to e.
func (r *resolver) childOf(e *ast.Entity, words []string, d *ast.AggregateDecl) (*ast.Entity, string, string, error) {
	what := "total de"
	if d.Sum {
		what = "soma"
	}
	name, _ := phrase(words)
	c, state := r.byName[name], ""
	if c == nil && len(words) >= 2 {
		if pc := r.byName[strings.Join(words[:len(words)-1], "_")]; pc != nil {
			word := words[len(words)-1]
			states := statesOf(pc)
			for _, st := range states {
				if word == st || word == st+"s" {
					c, state = pc, st
				}
			}
			if c == nil {
				if len(states) == 0 {
					return nil, "", "", r.errAt(d.Pos, "%s: %s não tem estados, então %q não diz nada.\nComo corrigir: tire %q da linha", d.Line, pc.Plural, word, word)
				}
				return nil, "", "", r.errAt(d.Pos, "%s: %q não é um estado de %s (estados: %s)", d.Line, word, pc.Plural, strings.Join(states, ", "))
			}
		}
	}
	if c == nil {
		_, err := r.entity(name, d.Pos)
		return nil, "", "", err
	}
	via, many := viaField(c, e)
	if via == "" {
		if len(many) > 1 {
			return nil, "", "", r.errAt(d.Pos, "%s: %s aponta para %s por mais de um campo (%s), e não sei por qual contar.\nComo corrigir: declare em %s › tem a linha %s, para que um deles seja o dono", d.Line, c.Plural, e.Singular, strings.Join(many, ", "), e.Plural, c.Plural)
		}
		return nil, "", "", r.errAt(d.Pos, "%s: %s não pertence a %s.\nPor quê: um registro só mostra números sobre o que pertence a ele (%s %s %s mostra os de cada %s).\nComo corrigir: declare em %s › tem a linha %s, ou em %s › pertence a a linha %s", d.Line, c.Plural, e.Singular, e.Singular, what, c.Plural, e.Singular, e.Plural, c.Plural, c.Plural, e.Singular)
	}
	if c.Singular == r.app.MemberModel || c.ViewThrough != nil {
		return nil, "", "", r.errAt(d.Pos, "%s: %s é mantido pelo sistema e não é somado nem contado num registro", d.Line, c.Plural)
	}
	return c, via, state, nil
}

// aggregates resolves the numbers of each record (GEP 0047).
func (r *resolver) aggregates(in *ast.Intent) error {
	for _, d := range in.Aggregates {
		e, err := r.entity(d.Entity, d.Pos)
		if err != nil {
			return err
		}
		ag := &ast.Aggregate{Pos: d.Pos}
		var c *ast.Entity
		if !d.Sum {
			if c, ag.Via, ag.State, err = r.childOf(e, d.Words, d); err != nil {
				return err
			}
		} else {
			// soma do <campo> dos <dado>: the field may hold connectors
			// itself (tempo de resposta), so the first split that names a
			// data belonging to e and one of its fields wins
			var firstErr error
			for k := 1; k < len(d.Words)-1 && c == nil; k++ {
				if !connector(d.Words[k]) {
					continue
				}
				field, _ := phrase(d.Words[:k])
				cc, via, st, err := r.childOf(e, d.Words[k+1:], d)
				if err != nil {
					if firstErr == nil {
						firstErr = err
					}
					continue
				}
				f := fieldByNameAST(cc.Model, field)
				if f == nil {
					var nums []string
					for _, x := range cc.Model.Fields {
						if numericField(x) && !x.Private && !x.Hidden && !x.IsSecret() {
							nums = append(nums, x.Name)
						}
					}
					hint := "declare o campo como número (ex.: " + field + " inteiro)"
					if len(nums) > 0 {
						hint = "some um destes: " + strings.Join(nums, ", ")
					}
					firstErr = r.errAt(d.Pos, "%s: %s não tem o campo %q.\nComo corrigir: %s", d.Line, cc.Plural, field, hint)
					continue
				}
				if !numericField(f) {
					return r.errAt(d.Pos, "%s: %s é %s, e só números se somam.\nComo corrigir: declare %s com um tipo de número (inteiro, numero ou dinheiro), ou conte com: total de %s", d.Line, f.Name, f.Type, f.Name, cc.Plural)
				}
				if f.Private || f.Hidden || f.IsSecret() {
					return r.errAt(d.Pos, "%s: %s não aparece para todos que veem %s, e a soma revelaria os valores.\nComo corrigir: some um campo que todos que veem o registro podem ver", d.Line, f.Name, cc.Plural)
				}
				c, ag.Via, ag.State, ag.Field = cc, via, st, f.Name
			}
			if c == nil {
				if firstErr == nil {
					firstErr = r.errAt(d.Pos, "%s: não entendi qual campo de qual dado somar.\nComo corrigir: escreva soma do <campo> dos <dados>, por exemplo: soma do peso das issues", d.Line)
				}
				return firstErr
			}
		}
		ag.Of = c.Singular
		ag.Name = d.Name
		if ag.Name == "" {
			ag.Name, _ = phrase(strings.Fields(strings.ToLower(foldWord(d.Line))))
		}
		ag.Label = titleCase(ag.Name)
		if d.Name == "" {
			ag.Label = strings.ToUpper(d.Line[:1]) + d.Line[1:]
		}
		if fieldByNameAST(e.Model, ag.Name) != nil || ag.Name == "id" || ag.Name == "created_at" || ag.Name == "updated_at" || ag.Name == e.Marks {
			return r.errAt(d.Pos, "%s: %s já tem um campo %s, e o número teria o mesmo nome.\nComo corrigir: dê outro nome com como, por exemplo: %s como %s_total", d.Line, e.Plural, ag.Name, d.Line, ag.Name)
		}
		dup := false
		for _, old := range e.Aggregates {
			if old.Name == ag.Name {
				if old.Of == ag.Of && old.Field == ag.Field && old.State == ag.State {
					dup = true // the same fact twice changes nothing
					break
				}
				return r.errAt(d.Pos, "%s: %s já mostra um número chamado %s (%s).\nComo corrigir: dê outro nome com como", d.Line, e.Plural, ag.Name, where(old.Pos))
			}
		}
		if !dup {
			e.Aggregates = append(e.Aggregates, ag)
		}
	}
	return nil
}

// resets resolves `pode zerar <nome>` (GEP 0047): the sum is brought back to
// zero by one more record that subtracts it, so it must be a sum whose
// records can be created with only that value.
func (r *resolver) resets(grants []*ast.Grant) error {
	for _, g := range grants {
		e := r.byName[g.Role]
		var ag *ast.Aggregate
		for _, x := range e.Aggregates {
			if x.Name == g.Target {
				ag = x
			}
		}
		if ag == nil {
			var names []string
			for _, x := range e.Aggregates {
				if x.Field != "" {
					names = append(names, x.Name)
				}
			}
			hint := "declare antes a soma, por exemplo: indicadores › soma da duracao dos tempos gastos como tempo gasto"
			if len(names) > 0 {
				hint = "zere uma das somas de " + e.Plural + ": " + strings.Join(names, ", ")
			}
			if g.Target == "" {
				return r.errAt(g.Pos, "%s pode zerar o quê? Diga qual soma.\nComo corrigir: %s", e.Singular, hint)
			}
			return r.errAt(g.Pos, "%s pode zerar %s: %s não mostra um número chamado %s.\nComo corrigir: %s", e.Singular, strings.ReplaceAll(g.Target, "_", " "), e.Plural, g.Target, hint)
		}
		if ag.Field == "" {
			return r.errAt(g.Pos, "%s pode zerar %s: %s é uma contagem, e uma contagem não se zera.\nPor quê: zerar registra um valor que desconta a soma; para contar menos, exclua os registros", e.Singular, strings.ReplaceAll(g.Target, "_", " "), ag.Name)
		}
		if ag.State != "" {
			return r.errAt(g.Pos, "%s pode zerar %s: a soma conta só os registros %s, e o registro que desconta nasceria em outro estado.\nComo corrigir: zere uma soma sem estado", e.Singular, ag.Name, ag.State)
		}
		c := r.app.Entities[ag.Of]
		for _, f := range c.Model.Fields {
			key := strings.ToLower(f.Name)
			if key == strings.ToLower(ag.Field) {
				if f.Min != nil && *f.Min >= 0 {
					return r.errAt(g.Pos, "%s pode zerar %s: zerar registra um %s com %s negativo, mas %s tem min %v.\nComo corrigir: tire o mínimo de %s, ou não zere esta soma", e.Singular, ag.Name, c.Singular, f.Name, f.Name, *f.Min, f.Name)
				}
				continue
			}
			if !f.Required || f.System || f.HasDefault || f.NumberedBy != "" || key == ag.Via || contains(c.OwnerFields, key) || f.Type == ast.FieldSegredo {
				continue
			}
			return r.errAt(g.Pos, "%s pode zerar %s: zerar registra um %s só com %s, mas %s é obrigatório.\nComo corrigir: tire obrigatório de %s, ou dê a ele um valor inicial (começa com …)", e.Singular, ag.Name, c.Singular, ag.Field, f.Name, f.Name)
		}
		verb := "zerar_" + ag.Name
		if e.Transitions[verb] != nil || e.Hooks[verb] != nil {
			return r.errAt(g.Pos, "%s já tem uma ação %s; zerar %s teria o mesmo nome", e.Singular, verb, ag.Name)
		}
		ag.Reset = true
	}
	return nil
}

// pairs resolves `única por par de <dado>` (GEP 0048): the record has
// exactly two references to that data.
func (r *resolver) pairs(in *ast.Intent) error {
	for _, d := range in.Pairs {
		e, err := r.entity(d.Entity, d.Pos)
		if err != nil {
			return err
		}
		of, err := r.entity(d.Of, d.Pos)
		if err != nil {
			return err
		}
		var fields []string
		for f, t := range e.Parents {
			if t == of.Singular {
				fields = append(fields, f)
			}
		}
		sort.Strings(fields)
		if len(fields) != 2 {
			return r.errAt(d.Pos, "%s é única por par de %s, mas aponta para %s por %d campo(s)%s.\nPor quê: um par são dois %s diferentes; o registro precisa nomear os dois.\nComo corrigir: %s pertence a um %s (%s › tem › %s) e a outro com outro nome (%s › pertence a › %s como relacionada)",
				e.Singular, of.Plural, of.Singular, len(fields), listed(fields), of.Plural, e.Plural, of.Singular, of.Plural, e.Plural, e.Plural, of.Singular)
		}
		owner, other := fields[0], fields[1]
		if other == of.Singular+"_id" {
			owner, other = other, owner
		}
		p := &ast.Pair{Of: of.Singular, Owner: owner, Other: other}
		if e.Pair != nil {
			if *e.Pair == *p {
				continue
			}
			return r.errAt(d.Pos, "%s já é único por par de %s", e.Singular, e.Pair.Of)
		}
		e.Pair = p
		e.Model.Pairs = append(e.Model.Pairs, [2]string{owner, other})
	}
	return nil
}

func listed(fields []string) string {
	if len(fields) == 0 {
		return ""
	}
	return " (" + strings.Join(fields, ", ") + ")"
}
