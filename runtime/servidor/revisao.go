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

// ---------- merging when the executions pass (GEP 0027, em teste) ----------

// latestRun returns the newest execution of the record with the repository
// for the current version of the source branch (nil when there is none:
// nothing will run for that version).
func (a *intentAPI) latestRun(ctx *interp.Context, e *ast.Entity, row map[string]any) map[string]any {
	repo, parent := a.reviewRepo(ctx, e, row)
	run := a.app.Entities[e.Review.Runs]
	if parent == nil || run == nil {
		return nil
	}
	source := fmt.Sprint(row[e.Review.Source])
	head, err := a.s.Git.Resolve(repo, "refs/heads/"+source)
	if err != nil {
		return nil
	}
	res, err := a.in.Op(ctx, run.Singular, "filtrar", map[string]any{run.Execution.OwnerField: parent["id"], "branch": source, "versao": head}, map[string]any{"ordenar": "-id", "limite": 1})
	if err != nil {
		return nil
	}
	if list, _ := res.([]any); len(list) > 0 {
		m, _ := list[0].(map[string]any)
		return m
	}
	return nil
}

// scheduleMerge: "merge when the executions pass". Everything that must hold
// for a merge is checked now; when the latest execution of the source is
// still running, the record waits (and is returned); when it already passed,
// or nothing runs for that version, nil means "merge now"; a failed or
// canceled execution refuses, because the person asked to merge only if it
// passes.
func (a *intentAPI) scheduleMerge(ctx *interp.Context, atual map[string]any, e *ast.Entity, row map[string]any) (map[string]any, error) {
	if err := a.mergeReady(ctx, atual, e, row); err != nil {
		return nil, err
	}
	run := a.latestRun(ctx, e, row)
	switch st := toStr(run["estado"]); st {
	case "", stSuccess:
		return nil, nil
	case stFailed, stCanceled:
		msg := fmt.Sprintf("A última execução de %s %s: corrija e envie o código de novo, ou mescle sem esperar.", row[e.Review.Source], map[string]string{stFailed: "falhou", stCanceled: "foi cancelada"}[st])
		if a.app.Messages == "en" {
			msg = "405 Method Not Allowed: the latest pipeline did not succeed"
		}
		return nil, &interp.RuntimeError{Status: 405, Message: msg}
	}
	res, err := a.in.Op(ctx, e.Singular, "atualizar", row["id"], map[string]any{"mesclar_quando_passar": true, "mesclagem_agendada_por_id": atual["id"]})
	if err != nil {
		return nil, err
	}
	return res.(map[string]any), nil
}

// unscheduleMerge stops waiting.
func (a *intentAPI) unscheduleMerge(ctx *interp.Context, e *ast.Entity, row map[string]any) (map[string]any, error) {
	res, err := a.in.Op(ctx, e.Singular, "atualizar", row["id"], map[string]any{"mesclar_quando_passar": false, "mesclagem_agendada_por_id": nil})
	if err != nil {
		return nil, err
	}
	return res.(map[string]any), nil
}

// runFinished merges the records waiting for this execution: when it is the
// latest one of their source branch and it passed, each is merged as the
// person who asked, with every rule checked again (permissions, approvals,
// conflicts, protected branches); if anything refuses, or the execution
// failed or was canceled, the record stops waiting and stays open.
func (a *intentAPI) runFinished(ctx *interp.Context, run *ast.Entity, runID any, state string) {
	waits := false
	for _, e := range a.app.Entities {
		waits = waits || e.Review != nil && e.Review.Runs == run.Singular
	}
	if !waits {
		return // nothing can wait for these executions
	}
	res, _ := a.in.Op(ctx, run.Singular, "buscar", runID)
	runRow, _ := res.(map[string]any)
	if runRow == nil {
		return
	}
	for _, n := range a.app.Order {
		e := a.app.Entities[n]
		if e.Review == nil || e.Review.Runs != run.Singular {
			continue
		}
		res, err := a.in.Op(ctx, e.Singular, "filtrar", map[string]any{
			e.Review.RepoVia: runRow[run.Execution.OwnerField], e.Review.Source: runRow["branch"],
			"mesclar_quando_passar": true, e.StateField: e.Initial,
		}, map[string]any{"ordenar": "id", "limite": 100})
		if err != nil {
			continue
		}
		list, _ := res.([]any)
		for _, it := range list {
			row, _ := it.(map[string]any)
			if latest := a.latestRun(ctx, e, row); latest == nil || fmt.Sprint(latest["id"]) != fmt.Sprint(runID) {
				continue // an older execution: wait for the newest one
			}
			if err := a.mergeWaiting(ctx, e, row, state); err != nil {
				fmt.Printf("[germanio] %s %v não foi mesclado: %v\n", e.Singular, row["id"], err)
				a.unscheduleMerge(ctx, e, row)
			}
		}
	}
}

func (a *intentAPI) mergeWaiting(ctx *interp.Context, e *ast.Entity, row map[string]any, state string) error {
	if state != stSuccess {
		return fmt.Errorf("a execução terminou como %s", state)
	}
	res, _ := a.in.Op(ctx, a.app.LoginEntity, "buscar", row["mesclagem_agendada_por_id"])
	atual, _ := res.(map[string]any)
	if atual == nil {
		return fmt.Errorf("quem pediu a mesclagem não existe mais")
	}
	tr := e.Transitions["mesclar"]
	if tr == nil || !a.in.Can(ctx, atual, e, "mesclar", row) {
		return fmt.Errorf("quem pediu a mesclagem não pode mais mesclar")
	}
	if err := a.frozenFor(ctx, "acao", e, row, nil); err != nil {
		return err // e.g. the project became read-only meanwhile
	}
	if err := a.merge(ctx, atual, e, row, truthy(row["juntar_commits"])); err != nil {
		return err
	}
	_, err := a.transition(ctx, atual, e, tr, a.find(ctx, e, fmt.Sprint(row["id"]), nil), nil)
	return err
}
