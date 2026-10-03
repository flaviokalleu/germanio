package parser

import (
	"sort"
	"strconv"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/diagnostics"
)

// Approval minimums (GEP 0026, em teste):
//
//	merge requests precisam de 2 aprovações para mesclar
//	merge requests › regras › precisa de 2 aprovações para mesclar
//
// The words after precisa(m) mention approvals.
func isApprovalMinimum(rest []string) bool {
	for _, x := range rest {
		if strings.HasPrefix(x, "aprovac") {
			return true
		}
	}
	return false
}

var numberWords = map[string]int{"um": 1, "uma": 1, "dois": 2, "duas": 2, "tres": 3, "quatro": 4, "cinco": 5, "seis": 6, "sete": 7, "oito": 8, "nove": 9, "dez": 10}

const approvalExample = "merge requests precisam de 2 aprovações para mesclar (no bloco do dado: regras › precisa de 2 aprovações para mesclar)"

// approvalMinimum parses `[de|ter] [pelo menos] N aprovações para <verbo>`.
func (p *Parser) approvalMinimum(head dline, subject string, rest []string, pos diagnostics.Position) error {
	at := head.toks[0]
	k := 0
	if k < len(rest) && (rest[k] == "de" || rest[k] == "ter") {
		k++
	}
	if k+1 < len(rest) && rest[k] == "pelo" && rest[k+1] == "menos" {
		k += 2
	}
	if k >= len(rest) {
		return p.teach(at, "\""+lineText(head)+"\" não diz quantas aprovações", "o número de aprovações vem antes de \"aprovações\"", "escreva: "+approvalExample, "")
	}
	n, ok := numberWords[rest[k]]
	if !ok {
		v, err := strconv.Atoi(rest[k])
		if err != nil {
			return p.teach(at, "\""+rest[k]+"\" não é um número de aprovações", "o mínimo é um número inteiro, escrito com algarismos (2) ou por extenso (duas)", "escreva: "+approvalExample, "")
		}
		n = v
	}
	if n < 1 || n > 100 {
		return p.teach(at, "o mínimo de aprovações precisa estar entre 1 e 100", "sem aprovação nenhuma, a frase não é necessária; mais de 100 pessoas não é uma regra de revisão", "escreva: "+approvalExample, "")
	}
	k++
	if k >= len(rest) || !strings.HasPrefix(rest[k], "aprovac") {
		return p.teach(at, "\""+lineText(head)+"\" não é uma frase conhecida", "depois do número vem \"aprovações\"", "escreva: "+approvalExample, "")
	}
	k++
	if k+1 >= len(rest) || rest[k] != "para" {
		return p.teach(at, "\""+lineText(head)+"\" não diz para quê as aprovações são necessárias", "as aprovações liberam uma ação do dado, que vem depois de \"para\"", "escreva: "+approvalExample, "")
	}
	verb, _ := phrase(rest[k+1:])
	if subject == "" {
		return p.teach(at, "\""+lineText(head)+"\" não diz de qual dado", "a regra vem depois do dado", "escreva: "+approvalExample, "")
	}
	p.intent().ApprovalMinimums = append(p.intent().ApprovalMinimums, &ast.ApprovalMinimum{Entity: subject, Count: n, Verb: verb, Pos: pos})
	return nil
}

// approvalMinimums: the data receives approvals and the verb is one of its
// actions; two different minimums for the same action are an error.
func (r *resolver) approvalMinimums(in *ast.Intent) error {
	for _, m := range in.ApprovalMinimums {
		e, err := r.entity(m.Entity, m.Pos)
		if err != nil {
			return err
		}
		if !e.Approvals {
			return r.errAt(m.Pos, "%s precisam de aprovações para %s, mas não recebem aprovações. Declare também: %s recebe aprovações (no bloco do dado: recebe aprovações)", e.Plural, m.Verb, e.Singular)
		}
		if e.Transitions[m.Verb] == nil {
			var verbs []string
			for v := range e.Transitions {
				verbs = append(verbs, v)
			}
			sort.Strings(verbs)
			return r.errAt(m.Pos, "%s precisam de aprovações para %s, mas %s não é uma ação de %s (ações: %s). As aprovações liberam uma ação declarada em \"pode\"", e.Plural, m.Verb, m.Verb, e.Plural, strings.Join(verbs, ", "))
		}
		if e.ApprovalsNeeded == nil {
			e.ApprovalsNeeded = map[string]int{}
		}
		if old, ok := e.ApprovalsNeeded[m.Verb]; ok && old != m.Count {
			return r.errAt(m.Pos, "%s precisam de %d aprovações para %s em um lugar e de %d em outro; declare um só número", e.Plural, old, m.Verb, m.Count)
		}
		e.ApprovalsNeeded[m.Verb] = m.Count
	}
	return nil
}
