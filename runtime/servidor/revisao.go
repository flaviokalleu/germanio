package servidor

import (
	"fmt"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// ---------- approval minimums (GEP 0026, em teste) ----------

// approvals counts, for verb, the approvals required and the approvals the
// record has. Only distinct people count, and never the record's owner (the
// author does not approve their own proposal).
func approvals(e *ast.Entity, verb string, row map[string]any) (need, have int) {
	need = e.ApprovalsNeeded[verb]
	if need == 0 || row == nil {
		return need, 0
	}
	owners := map[string]bool{}
	for _, f := range e.OwnerFields {
		if v := row[f]; v != nil {
			owners[fmt.Sprint(v)] = true
		}
	}
	seen := map[string]bool{}
	list, _ := row["aprovacoes"].([]any)
	for _, v := range list {
		k := fmt.Sprint(v)
		if !owners[k] && !seen[k] {
			seen[k] = true
			have++
		}
	}
	return need, have
}

// approvalGate refuses verb while the record lacks approvals.
func (a *intentAPI) approvalGate(e *ast.Entity, verb string, row map[string]any) error {
	need, have := approvals(e, verb, row)
	if have >= need {
		return nil
	}
	if a.app.Messages == "en" {
		return &interp.RuntimeError{Status: 405, Message: fmt.Sprintf("%d more approval(s) required", need-have)}
	}
	plural := func(n int, one, many string) string {
		if n == 1 {
			return one
		}
		return many
	}
	return &interp.RuntimeError{Status: 405, Message: fmt.Sprintf(
		"%s precisa de %d %s para %s e tem %d. %s %d %s de alguém que não seja o autor: peça a quem pode aprovar que aprove.",
		e.Label, need, plural(need, "aprovação", "aprovações"), verb, have, plural(need-have, "Falta", "Faltam"), need-have, plural(need-have, "aprovação", "aprovações"))}
}

// approvalStatus adds, to a record shown to someone, how many approvals
// its actions need and how many are still missing (the largest minimum).
func approvalStatus(e *ast.Entity, row, out map[string]any) {
	need, have := 0, 0
	for verb := range e.ApprovalsNeeded {
		n, h := approvals(e, verb, row)
		if n > need {
			need, have = n, h
		}
	}
	if need == 0 {
		return
	}
	out["aprovacoes_necessarias"] = need
	out["aprovacoes_faltando"] = max(0, need-have)
}

// conflictError refuses a merge whose branches conflict, naming the files.
func (a *intentAPI) conflictError(files []string) error {
	if a.app.Messages == "en" {
		return &interp.RuntimeError{Status: 406, Message: "Branch cannot be merged"}
	}
	return &interp.RuntimeError{Status: 406, Message: "Há conflitos entre as branches: " + strings.Join(files, ", ")}
}

func toStrings(v any) []string {
	list, _ := v.([]any)
	out := make([]string, 0, len(list))
	for _, it := range list {
		out = append(out, fmt.Sprint(it))
	}
	return out
}
