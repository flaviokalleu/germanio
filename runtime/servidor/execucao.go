package servidor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/runtime/banco"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
	"gopkg.in/yaml.v3"
)

// Execution capability: `X executa Ys a cada envio de código conforme "f"`.
// A run (pipeline) is created for each push from the file at that commit;
// its steps (jobs) are grouped in stages and executed in isolated
// directories by the local executor. The mechanism knows nothing about the
// application: it works for any record that owns a repository.

const (
	stCreated  = "criado"
	stPending  = "pendente"
	stRunning  = "executando"
	stSuccess  = "sucesso"
	stFailed   = "falhou"
	stCanceled = "cancelado"
	stSkipped  = "ignorado"
	stManual   = "manual"
)

type executor struct {
	a       *intentAPI
	mode    string // "local" or "docker"
	slots   chan struct{}
	mu      sync.Mutex
	cancels map[int64]context.CancelFunc
	runMu   sync.Map // run id → *sync.Mutex (serialises stage advancement)
	stop    chan struct{}
}

// stepSpec is one job from the configuration file.
type stepSpec struct {
	Name, Stage, When, Image string
	Script, After            []string
	AllowFailure             bool
	Order                    int
}

var reservedKeys = map[string]bool{"stages": true, "variables": true, "image": true, "default": true, "include": true, "workflow": true,
	"before_script": true, "after_script": true, "services": true, "cache": true}

func lines(v any) []string {
	switch x := v.(type) {
	case string:
		return []string{x}
	case []any:
		var out []string
		for _, it := range x {
			if s, ok := it.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// parseRunFile reads the stages/jobs format: `stages:` plus one mapping per
// job with script, stage, when, allow_failure, image, before/after_script.
func parseRunFile(data []byte) ([]stepSpec, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("configuração inválida: %v", err)
	}
	stages := []string{"build", "test", "deploy"}
	if s := lines(doc["stages"]); len(s) > 0 {
		stages = s
	}
	order := map[string]int{".pre": -1, ".post": len(stages)}
	for i, s := range stages {
		order[s] = i
	}
	var before, after []string
	image, _ := doc["image"].(string)
	if d, ok := doc["default"].(map[string]any); ok {
		before, after = lines(d["before_script"]), lines(d["after_script"])
		if im, ok := d["image"].(string); ok {
			image = im
		}
	}
	if b := lines(doc["before_script"]); b != nil {
		before = b
	}
	if a := lines(doc["after_script"]); a != nil {
		after = a
	}
	var specs []stepSpec
	names := make([]string, 0, len(doc))
	for k := range doc {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, name := range names {
		if reservedKeys[name] || strings.HasPrefix(name, ".") {
			continue
		}
		job, ok := doc[name].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("configuração inválida: %s deve ser um mapa", name)
		}
		script := lines(job["script"])
		if len(script) == 0 {
			return nil, fmt.Errorf("configuração inválida: o job %s não tem script", name)
		}
		stage, _ := job["stage"].(string)
		if stage == "" {
			stage = "test"
		}
		idx, ok := order[stage]
		if !ok {
			return nil, fmt.Errorf("configuração inválida: o job %s usa a etapa %q, que não está em stages", name, stage)
		}
		when, _ := job["when"].(string)
		if when == "" {
			when = "on_success"
		}
		sp := stepSpec{Name: name, Stage: stage, When: when, Order: idx + 1, Image: image}
		if im, ok := job["image"].(string); ok {
			sp.Image = im
		}
		jb := before
		if b := lines(job["before_script"]); b != nil {
			jb = b
		}
		sp.Script = append(append([]string{}, jb...), script...)
		sp.After = after
		if a := lines(job["after_script"]); a != nil {
			sp.After = a
		}
		sp.AllowFailure, _ = job["allow_failure"].(bool)
		if when == "manual" {
			if _, set := job["allow_failure"]; !set {
				sp.AllowFailure = true
			}
		}
		specs = append(specs, sp)
	}
	if len(specs) == 0 {
		return nil, fmt.Errorf("configuração inválida: nenhum job definido")
	}
	sort.SliceStable(specs, func(i, j int) bool { return specs[i].Order < specs[j].Order })
	return specs, nil
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

// createRun reads the file at sha and creates the run and its steps.
// found=false means the repository has no configuration file there.
func (a *intentAPI) createRun(ctx *interp.Context, atual map[string]any, run *ast.Entity, owner map[string]any, branch, sha string) (map[string]any, bool, error) {
	x := run.Execution
	repo, _ := owner["repositorio"].(string)
	blob, err := a.s.Git.ReadFile(repo, sha, x.File, 1<<20)
	if err != nil || blob == nil {
		return nil, false, nil
	}
	specs, perr := parseRunFile(blob.Content)
	data := map[string]any{x.OwnerField: owner["id"], "branch": branch, "versao": sha}
	if atual != nil {
		data[a.app.LoginEntity+"_id"] = atual["id"]
	}
	if perr != nil {
		data["estado"], data["erro_configuracao"] = stFailed, perr.Error()
	}
	res, err := a.in.Op(ctx, run.Singular, "criar", data)
	if err != nil {
		return nil, true, err
	}
	row := res.(map[string]any)
	if perr != nil {
		return row, true, nil
	}
	step := a.app.Entities[x.Step]
	for _, sp := range specs {
		body, _ := json.Marshal(map[string]any{"script": sp.Script, "after": sp.After})
		if _, err := a.in.Op(ctx, step.Singular, "criar", map[string]any{
			step.Execution.RunField: row["id"], "nome": sp.Name, "etapa": sp.Stage, "ordem": float64(sp.Order),
			"script": string(body), "quando": sp.When, "permitir_falha": sp.AllowFailure, "imagem": sp.Image,
		}); err != nil {
			return row, true, err
		}
	}
	a.advance(ctx, run, row["id"])
	fresh, _ := a.in.Op(ctx, run.Singular, "buscar", row["id"])
	if m, ok := fresh.(map[string]any); ok {
		row = m
	}
	return row, true, nil
}

// startRuns creates runs for every branch updated by a push.
func (a *intentAPI) startRuns(ctx *interp.Context, atual map[string]any, owner *ast.Entity, ownerRow map[string]any, updates []any) {
	for _, n := range a.app.Order {
		run := a.app.Entities[n]
		if run.Execution == nil || run.Execution.Role != "run" || run.Execution.Owner != owner.Singular {
			continue
		}
		for _, it := range updates {
			u, _ := it.(map[string]any)
			branch, _ := u["branch"].(string)
			if branch == "" || u["tipo"] == "excluir" {
				continue
			}
			if _, _, err := a.createRun(ctx, atual, run, ownerRow, branch, fmt.Sprint(u["depois"])); err != nil {
				fmt.Printf("[germanio] execução para %s: %v\n", branch, err)
			}
		}
	}
}

func (a *intentAPI) steps(ctx *interp.Context, run *ast.Entity, runID any) []map[string]any {
	step := a.app.Entities[run.Execution.Step]
	res, err := a.in.Op(ctx, step.Singular, "filtrar", map[string]any{step.Execution.RunField: runID}, map[string]any{"ordenar": "id", "limite": 1000})
	if err != nil {
		return nil
	}
	var out []map[string]any
	for _, it := range res.([]any) {
		m := it.(map[string]any)
		if b, _ := m["repetido"].(bool); !b {
			out = append(out, m)
		}
	}
	return out
}

func (a *intentAPI) runLock(id any) *sync.Mutex {
	if a.exec == nil {
		return &sync.Mutex{}
	}
	m, _ := a.exec.runMu.LoadOrStore(fmt.Sprint(id), &sync.Mutex{})
	return m.(*sync.Mutex)
}

// advance moves a run forward: the next stage becomes pending when the
// current one succeeded; a failure skips what is left; the run's state
// summarises its steps.
func (a *intentAPI) advance(ctx *interp.Context, run *ast.Entity, runID any) {
	lock := a.runLock(runID)
	lock.Lock()
	defer lock.Unlock()
	step := a.app.Entities[run.Execution.Step]
	jobs := a.steps(ctx, run, runID)
	byOrder := map[int][]map[string]any{}
	var orders []int
	for _, j := range jobs {
		o := int(asNumber(j["ordem"]))
		if _, ok := byOrder[o]; !ok {
			orders = append(orders, o)
		}
		byOrder[o] = append(byOrder[o], j)
	}
	sort.Ints(orders)
	set := func(j map[string]any, st string) {
		a.in.Op(ctx, step.Singular, "atualizar", j["id"], map[string]any{"estado": st})
		j["estado"] = st
	}
	finish := func(st string) {
		res, _ := a.in.Op(ctx, run.Singular, "buscar", runID)
		row, _ := res.(map[string]any)
		change := map[string]any{"estado": st, "terminado_em": now()}
		if row != nil {
			if started := toStr(row["iniciado_em"]); started != "" {
				if t0, err := time.Parse(time.RFC3339, started); err == nil {
					change["duracao"] = time.Since(t0).Seconds()
				}
			}
		}
		a.in.Op(ctx, run.Singular, "atualizar", runID, change)
	}
	for i, o := range orders {
		stage := byOrder[o]
		busy, fresh, failed, canceled := false, false, false, false
		for _, j := range stage {
			switch toStr(j["estado"]) {
			case stPending, stRunning:
				busy = true
			case stCreated:
				fresh = true
			case stFailed:
				if b, _ := j["permitir_falha"].(bool); !b {
					failed = true
				}
			case stCanceled:
				canceled = true
			}
		}
		if fresh {
			for _, j := range stage {
				if toStr(j["estado"]) == stCreated {
					if toStr(j["quando"]) == "manual" {
						set(j, stManual)
					} else {
						set(j, stPending)
					}
				}
			}
			// Manual steps wait for someone to start them; they do not block.
			for _, j := range stage {
				if toStr(j["estado"]) == stPending {
					busy = true
				}
			}
		}
		if busy {
			a.in.Op(ctx, run.Singular, "atualizar", runID, map[string]any{"estado": stRunning})
			return
		}
		if failed || canceled {
			for _, later := range orders[i+1:] {
				for _, j := range byOrder[later] {
					if st := toStr(j["estado"]); st == stCreated || st == stManual {
						set(j, stSkipped)
					}
				}
			}
			if canceled && !failed {
				finish(stCanceled)
			} else {
				finish(stFailed)
			}
			return
		}
	}
	finish(stSuccess)
}

// ---------- executor ----------

func (a *intentAPI) startExecutor() {
	mode := os.Getenv("GERMANIO_EXECUTOR")
	if mode != "local" && mode != "docker" {
		fmt.Println("[germanio] Execuções: nenhum executor local ativo (defina GERMANIO_EXECUTOR=local ou docker para executar jobs neste servidor).")
		return
	}
	fmt.Printf("[germanio] Execuções: executor %s ativo. Atenção: jobs rodam com as permissões deste servidor.\n", mode)
	x := &executor{a: a, mode: mode, slots: make(chan struct{}, 2), cancels: map[int64]context.CancelFunc{}, stop: make(chan struct{})}
	a.exec = x
	a.s.onClose = append(a.s.onClose, func() { close(x.stop) })
	go x.loop()
}

func (x *executor) loop() {
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-x.stop:
			return
		case <-tick.C:
			x.claimAll()
		}
	}
}

func (x *executor) claimAll() {
	a := x.a
	for _, n := range a.app.Order {
		step := a.app.Entities[n]
		if step.Execution == nil || step.Execution.Role != "step" {
			continue
		}
		for {
			select {
			case x.slots <- struct{}{}:
			default:
				return
			}
			ctx := &interp.Context{}
			res, err := a.in.Op(ctx, step.Singular, "encontrar", map[string]any{"estado": stPending}, map[string]any{"ordenar": "id"})
			job, _ := res.(map[string]any)
			if err != nil || job == nil {
				<-x.slots
				break
			}
			n, err := a.s.DB.AtualizarOnde(step.Singular, banco.Consulta{Filtros: map[string]any{"id": job["id"], "estado": stPending}}, map[string]any{"estado": stRunning, "iniciado_em": now()})
			if err != nil || n != 1 {
				<-x.slots
				continue
			}
			go func(step *ast.Entity, job map[string]any) {
				defer func() { <-x.slots }()
				x.run(step, job)
			}(step, job)
		}
	}
}

// logBuffer collects output and saves it periodically.
type logBuffer struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (l *logBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.buf.Len() < 4<<20 {
		l.buf.Write(p)
	}
	return len(p), nil
}

func (l *logBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}

func (x *executor) run(step *ast.Entity, job map[string]any) {
	a := x.a
	ctx := &interp.Context{}
	run := a.app.Entities[step.Execution.Run]
	id := int64(asNumber(job["id"]))
	res, _ := a.in.Op(ctx, run.Singular, "buscar", job[step.Execution.RunField])
	runRow, _ := res.(map[string]any)
	if runRow == nil {
		return
	}
	if toStr(runRow["iniciado_em"]) == "" {
		a.in.Op(ctx, run.Singular, "atualizar", runRow["id"], map[string]any{"iniciado_em": now()})
	}
	owner := a.app.Entities[run.Execution.Owner]
	res, _ = a.in.Op(ctx, owner.Singular, "buscar", runRow[run.Execution.OwnerField])
	ownerRow, _ := res.(map[string]any)
	var spec struct{ Script, After []string }
	json.Unmarshal([]byte(toStr(job["script"])), &spec)

	timeout := time.Hour
	if d, err := time.ParseDuration(os.Getenv("GERMANIO_JOB_TIMEOUT")); err == nil {
		timeout = d
	}
	jctx, cancel := context.WithTimeout(context.Background(), timeout)
	x.mu.Lock()
	x.cancels[id] = cancel
	x.mu.Unlock()
	defer func() {
		cancel()
		x.mu.Lock()
		delete(x.cancels, id)
		x.mu.Unlock()
	}()

	log := &logBuffer{}
	done := make(chan struct{})
	go func() { // stream the log while the job runs
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				a.in.Op(ctx, step.Singular, "atualizar", id, map[string]any{"log": log.String()})
			}
		}
	}()
	status := x.execute(jctx, log, ownerRow, runRow, job, spec.Script, spec.After)
	close(done)
	if jctx.Err() == context.Canceled {
		status = stCanceled
	}
	change := map[string]any{"estado": status, "terminado_em": now(), "log": log.String()}
	if t0, err := time.Parse(time.RFC3339, toStr(job["iniciado_em"])); err == nil {
		change["duracao"] = time.Since(t0).Seconds()
	}
	// A cancel may have already set the state; do not overwrite it.
	current, _ := a.in.Op(ctx, step.Singular, "buscar", id)
	if m, ok := current.(map[string]any); ok && toStr(m["estado"]) == stCanceled {
		change["estado"] = stCanceled
	}
	a.in.Op(ctx, step.Singular, "atualizar", id, change)
	a.advance(ctx, run, runRow["id"])
}

// execute clones the commit into a private directory and runs each script
// line with sh (no shell expansion of application data happens here: the
// lines come from the repository's own file).
func (x *executor) execute(ctx context.Context, log *logBuffer, owner, run, job map[string]any, script, after []string) string {
	dir, err := os.MkdirTemp("", "germanio-job-*")
	if err != nil {
		fmt.Fprintf(log, "ERRO: %v\n", err)
		return stFailed
	}
	defer os.RemoveAll(dir)
	repoPath, err := x.a.s.Git.Path(toStr(owner["repositorio"]))
	if err != nil {
		fmt.Fprintf(log, "ERRO: %v\n", err)
		return stFailed
	}
	work := filepath.Join(dir, "projeto")
	sha := toStr(run["versao"])
	fmt.Fprintf(log, "Preparando o código em %s (%s)\n", toStr(run["branch"]), sha[:min(8, len(sha))])
	for _, args := range [][]string{{"git", "clone", "--quiet", "--no-hardlinks", repoPath, work}, {"git", "-C", work, "checkout", "--quiet", "--detach", sha}} {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + dir, "GIT_TERMINAL_PROMPT=0"}
		cmd.Stdout, cmd.Stderr = log, log
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(log, "ERRO ao preparar o código: %v\n", err)
			return stFailed
		}
	}
	env := []string{"PATH=" + os.Getenv("PATH"), "HOME=" + dir, "CI=true", "GERMANIO=true",
		"CI_COMMIT_SHA=" + sha, "CI_COMMIT_REF_NAME=" + toStr(run["branch"]), "CI_JOB_NAME=" + toStr(job["nome"]),
		"CI_JOB_STAGE=" + toStr(job["etapa"]), "CI_PIPELINE_ID=" + fmt.Sprint(run["id"]), "CI_JOB_ID=" + fmt.Sprint(job["id"]),
		"CI_PROJECT_DIR=" + work}
	runLines := func(list []string) bool {
		for _, line := range list {
			fmt.Fprintf(log, "$ %s\n", line)
			var cmd *exec.Cmd
			if x.mode == "docker" && toStr(job["imagem"]) != "" {
				cmd = exec.CommandContext(ctx, "docker", "run", "--rm", "--network", "none", "-v", work+":/builds/projeto", "-w", "/builds/projeto", toStr(job["imagem"]), "sh", "-c", line)
			} else {
				cmd = exec.CommandContext(ctx, "sh", "-c", line)
			}
			cmd.Dir, cmd.Env = work, env
			cmd.Stdout, cmd.Stderr = log, log
			if err := cmd.Run(); err != nil {
				if ctx.Err() == context.DeadlineExceeded {
					fmt.Fprintln(log, "ERRO: tempo limite excedido")
				} else if ctx.Err() == nil {
					fmt.Fprintf(log, "ERRO: o comando terminou com %v\n", err)
				}
				return false
			}
		}
		return true
	}
	ok := runLines(script)
	if len(after) > 0 && ctx.Err() == nil {
		runLines(after)
	}
	if ok {
		fmt.Fprintln(log, "Job concluído com sucesso")
		return stSuccess
	}
	fmt.Fprintln(log, "Job falhou")
	return stFailed
}

// ---------- actions ----------

// executionAction implements cancelar, repetir and executar for runs and steps.
func (a *intentAPI) executionAction(ctx *interp.Context, atual map[string]any, e *ast.Entity, row map[string]any, verb string) (map[string]any, error) {
	x := e.Execution
	en := a.app.Messages == "en"
	refuse := func(pt, eng string) error {
		if en {
			return &interp.RuntimeError{Status: 400, Message: eng}
		}
		return &interp.RuntimeError{Status: 400, Message: pt}
	}
	state := toStr(row["estado"])
	if x.Role == "step" {
		run := a.app.Entities[x.Run]
		switch verb {
		case "cancelar":
			if state != stPending && state != stRunning && state != stCreated && state != stManual {
				return nil, refuse("Este job não pode ser cancelado", "Job is not cancelable")
			}
			a.in.Op(ctx, e.Singular, "atualizar", row["id"], map[string]any{"estado": stCanceled, "terminado_em": now()})
			a.killStep(row["id"])
			a.advance(ctx, run, row[x.RunField])
		case "repetir":
			if state != stFailed && state != stCanceled && state != stSuccess {
				return nil, refuse("Só jobs terminados podem ser repetidos", "Job is not retryable")
			}
			a.in.Op(ctx, e.Singular, "atualizar", row["id"], map[string]any{"repetido": true})
			res, err := a.in.Op(ctx, e.Singular, "criar", map[string]any{x.RunField: row[x.RunField], "nome": row["nome"], "etapa": row["etapa"],
				"ordem": row["ordem"], "script": row["script"], "quando": "on_success", "permitir_falha": row["permitir_falha"], "imagem": row["imagem"]})
			if err != nil {
				return nil, err
			}
			a.reopenRun(ctx, run, row[x.RunField])
			a.advance(ctx, run, row[x.RunField])
			fresh, _ := a.in.Op(ctx, e.Singular, "buscar", res.(map[string]any)["id"])
			return fresh.(map[string]any), nil
		case "executar":
			if state != stManual {
				return nil, refuse("Só jobs manuais podem ser executados", "Unplayable Job")
			}
			a.in.Op(ctx, e.Singular, "atualizar", row["id"], map[string]any{"estado": stPending})
			a.reopenRun(ctx, run, row[x.RunField])
		}
		fresh, _ := a.in.Op(ctx, e.Singular, "buscar", row["id"])
		return fresh.(map[string]any), nil
	}
	// runs
	step := a.app.Entities[x.Step]
	switch verb {
	case "cancelar":
		for _, j := range a.steps(ctx, e, row["id"]) {
			switch toStr(j["estado"]) {
			case stCreated, stPending, stRunning, stManual:
				a.in.Op(ctx, step.Singular, "atualizar", j["id"], map[string]any{"estado": stCanceled})
				a.killStep(j["id"])
			}
		}
		a.advance(ctx, e, row["id"])
	case "repetir":
		for _, j := range a.steps(ctx, e, row["id"]) {
			if st := toStr(j["estado"]); st == stFailed || st == stCanceled {
				a.executionAction(ctx, atual, step, j, "repetir")
			}
		}
	}
	fresh, _ := a.in.Op(ctx, e.Singular, "buscar", row["id"])
	return fresh.(map[string]any), nil
}

func (a *intentAPI) reopenRun(ctx *interp.Context, run *ast.Entity, id any) {
	a.in.Op(ctx, run.Singular, "atualizar", id, map[string]any{"estado": stRunning, "terminado_em": nil})
}

func (a *intentAPI) killStep(id any) {
	if a.exec == nil {
		return
	}
	a.exec.mu.Lock()
	defer a.exec.mu.Unlock()
	if cancel, ok := a.exec.cancels[int64(asNumber(id))]; ok {
		cancel()
	}
}

// stepLog answers the log of a step as plain text.
func (a *intentAPI) stepLog(w http.ResponseWriter, row map[string]any) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, toStr(row["log"]))
}
