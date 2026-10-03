package parser

import (
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
)

// A sum that never goes below zero (docs/gep/0050-soma-nunca-negativa.md, em
// teste): the time spent on an issue, the stock of a product (the sum of its
// movements), the balance of an account without overdraft. Read-then-write
// in an adapter or a hook ("read the total, refuse if the entry subtracts
// more") races: two subtractions at once both pass. Declared, the rule is
// checked by the core in the change's transaction, with the record locked:
//
//	issues
//	    regras
//	        não pode ficar com tempo gasto negativo
//
// flat: `issue não pode ficar com tempo gasto negativo`.

const floorExample = "issue não pode ficar com tempo gasto negativo (ou, no bloco do dado: regras › não pode ficar com tempo gasto negativo)"

// floorLine reads `<dado> não pode ficar com <número> negativo`; i is the
// position of "não".
func (p *Parser) floorLine(head dline, i int) error {
	w := wordsOf(head.toks)
	subject, _ := phrase(w[:i])
	rest := w[i+4:]
	n := len(rest)
	if subject == "" || n < 2 || !strings.HasPrefix(rest[n-1], "negativ") {
		return p.teach(head.toks[0], "\""+lineText(head)+"\" não é uma regra conhecida",
			"não pode ficar com … negativo diz que uma soma do registro (um indicador) nunca fica abaixo de zero",
			"escreva: "+floorExample, "")
	}
	name, _ := phrase(rest[:n-1])
	p.intent().Floors = append(p.intent().Floors, &ast.FloorDecl{Entity: subject, Name: name, Pos: p.at(head.toks[0])})
	return nil
}

// floors resolves each rule to a sum the data shows (GEP 0047).
func (r *resolver) floors(in *ast.Intent) error {
	for _, d := range in.Floors {
		e, err := r.entity(d.Entity, d.Pos)
		if err != nil {
			return err
		}
		var sums []string
		var ag *ast.Aggregate
		for _, x := range e.Aggregates {
			if x.Field != "" {
				sums = append(sums, strings.ReplaceAll(x.Name, "_", " "))
			}
			if x.Name == d.Name {
				ag = x
			}
		}
		label := strings.ReplaceAll(d.Name, "_", " ")
		if ag == nil {
			hint := "declare antes a soma, por exemplo: indicadores › soma da duracao dos tempos gastos como tempo gasto"
			if len(sums) > 0 {
				hint = "use uma das somas de " + e.Plural + ": " + strings.Join(sums, ", ")
			}
			return r.errAt(d.Pos, "%s não pode ficar com %s negativo: %s não mostra um número chamado %s.\nPor quê: a regra vale para uma soma do registro (um indicador).\nComo corrigir: %s", e.Singular, label, e.Plural, label, hint)
		}
		if ag.Field == "" {
			return r.errAt(d.Pos, "%s não pode ficar com %s negativo: %s é uma contagem, e uma contagem nunca é negativa.\nComo corrigir: use a regra numa soma (soma do <campo> dos <dados>)", e.Singular, label, label)
		}
		if ag.State != "" {
			return r.errAt(d.Pos, "%s não pode ficar com %s negativo: a soma conta só os registros %s, e um registro que muda de estado mudaria a soma sem ser criado nem editado.\nComo corrigir: use a regra numa soma sem estado", e.Singular, label, ag.State)
		}
		ag.NonNegative = true
	}
	return nil
}
