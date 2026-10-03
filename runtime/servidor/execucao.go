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
	Artifacts                []string // files the step keeps (artefatos), for executors that collect them
	AllowFailure             bool
	Order                    int
	// flow (execucao_regras.go)
	Only, Except []string
	OnlySet      bool
	Rules        []stepRule
	Needs        []need
	NeedsSet     bool
	From         []string // recebe_artefatos_de
	FromSet      bool
	Expire       time.Duration // artefatos_expiram_em (0: kept)
}

// stepPlan is what a created step keeps of its spec (the step's script field).
type stepPlan struct {
	Script    []string `json:"script"`
	After     []string `json:"after"`
	Artifacts []string `json:"artifacts"`
	Needs     []string `json:"needs,omitempty"`
	NeedsSet  bool     `json:"needs_set,omitempty"`
	From      []string `json:"from,omitempty"`
	FromSet   bool     `json:"from_set,omitempty"`
	Expire    float64  `json:"expire,omitempty"` // seconds
}

func planOf(job map[string]any) stepPlan {
	var p stepPlan
	json.Unmarshal([]byte(toStr(job["script"])), &p)
	return p
}

// When a step runs, in the native format.
const (
	whenAuto   = "automatico" // after the previous stages succeed
	whenManual = "manual"     // when someone starts it
	whenAlways = "sempre"     // even after failures
)

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

// readRunFile turns the execution file into steps. The file is YAML; an
// adapter may translate an external format (`traduza arquivos de execução
// com f`) into the native one:
//
//	estagios: [construir, testar]
//	etapas:
//	  compilar:
//	    estagio: construir
//	    comandos: [make]          # or one line
//	    depois: [make clean]      # always runs after the commands
//	    quando: automatico | manual | sempre
//	    pode_falhar: sim | não
//	    imagem: golang:1.23       # container image (docker executor)
//	    artefatos: [saida/]       # files kept after success
//	    artefatos_expiram_em: 7 dias   # then deleted (default: kept)
//	    precisa: [preparar, {etapa: lint, opcional: sim}]  # starts when these end, not the whole stage
//	    recebe_artefatos_de: [preparar]  # default: the steps it waits for
//	    somente_em: [main, release/*, branch padrão]  # branches where it exists
//	    exceto_em: [rascunho/*]
//	    regras:                   # the first that matches the branch decides
//	      - em: [main]
//	        quando: manual        # automatico | manual | sempre | nunca
//	        pode_falhar: não
func (a *intentAPI) readRunFile(data []byte) ([]stepSpec, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("configuração inválida: %v", err)
	}
	if fn := a.app.Translators["arquivos_de_execucao"]; fn != "" {
		out, err := a.in.RunFunction(fn, []any{geValue(doc)}, &interp.Context{})
		if err != nil {
			return nil, fmt.Errorf("configuração inválida: %s", interp.Friendly(err))
		}
		m, ok := out.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("configuração inválida: %s não devolveu {estagios, etapas}", fn)
		}
		doc = m
	}
	return parseNativeRun(doc)
}

// geValue converts decoded YAML into the values .ge works with (numbers are
// float64, maps have string keys).
func geValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[k] = geValue(val)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[fmt.Sprint(k)] = geValue(val)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, val := range x {
			out[i] = geValue(val)
		}
		return out
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case uint64:
		return float64(x)
	}
	return v
}

// parseNativeRun reads the native format (see readRunFile).
func parseNativeRun(doc map[string]any) ([]stepSpec, error) {
	stages := lines(doc["estagios"])
	if len(stages) == 0 {
		stages = []string{"padrao"}
	}
	order := map[string]int{}
	for i, s := range stages {
		order[s] = i
	}
	steps, _ := doc["etapas"].(map[string]any)
	names := make([]string, 0, len(steps))
	for k := range steps {
		names = append(names, k)
	}
	sort.Strings(names)
	var specs []stepSpec
	for _, name := range names {
		job, ok := steps[name].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("configuração inválida: a etapa %s deve ser um mapa", name)
		}
		commands := lines(job["comandos"])
		if len(commands) == 0 {
			return nil, fmt.Errorf("configuração inválida: a etapa %s não tem comandos", name)
		}
		stage, _ := job["estagio"].(string)
		if stage == "" {
			stage = stages[0]
		}
		idx, ok := order[stage]
		if !ok {
			return nil, fmt.Errorf("configuração inválida: a etapa %s usa o estágio %q, que não está em estagios", name, stage)
		}
		when, _ := job["quando"].(string)
		switch when {
		case "":
			when = whenAuto
		case whenAuto, whenManual, whenAlways:
		default:
			return nil, fmt.Errorf("configuração inválida: a etapa %s tem quando %q (use automatico, manual ou sempre)", name, when)
		}
		sp := stepSpec{Name: name, Stage: stage, When: when, Order: idx + 1, Script: commands, After: lines(job["depois"]), Artifacts: lines(job["artefatos"])}
		sp.Image, _ = job["imagem"].(string)
		if v, ok := job["pode_falhar"]; ok && v != nil {
			b, valid := yes(v)
			if !valid {
				return nil, fmt.Errorf("configuração inválida: a etapa %s tem pode_falhar %v (use sim ou não)", name, v)
			}
			sp.AllowFailure = b
		}
		if err := parseFlow(&sp, job); err != nil {
			return nil, err
		}
		specs = append(specs, sp)
	}
	if len(specs) == 0 {
		return nil, fmt.Errorf("configuração inválida: nenhuma etapa definida")
	}
	if err := checkGraph(specs); err != nil {
		return nil, err
	}
	sort.SliceStable(specs, func(i, j int) bool { return specs[i].Order < specs[j].Order })
	return specs, nil
}

// stepVariables: the environment of a step. The core only says it runs in
// CI; an adapter may add the names an external format expects
// (`traduza variáveis das etapas com f`).
func (a *intentAPI) stepVariables(ctx *interp.Context, work map[string]any) map[string]string {
	out := map[string]string{"CI": "true"}
	defer func() {
		// the owner's variables (GEP 0015) never replace the execution's names
		for _, it := range asList(work["variaveis"]) {
			v, _ := it.(map[string]any)
			if name := toStr(v["nome"]); name != "" && validEnvName(name) {
				if _, taken := out[name]; !taken {
					out[name] = toStr(v["valor"])
				}
			}
		}
	}()
	if fn := a.app.Translators["variaveis_das_etapas"]; fn != "" {
		if res, err := a.in.RunFunction(fn, []any{work}, ctx); err == nil {
			if m, ok := res.(map[string]any); ok {
				for k, v := range m {
					out[k] = toStr(v)
				}
			}
		} else {
			fmt.Printf("[germanio] %s: %s\n", fn, interp.Friendly(err))
		}
	}
	return out
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
	specs, perr := a.readRunFile(blob.Content)
	if perr == nil {
		specs, perr = forBranch(specs, branch, defaultBranch(owner))
		if none, ok := perr.(errNoSteps); ok {
			return nil, true, none // nothing to run here: no run at all
		}
	}
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
		plan := stepPlan{Script: sp.Script, After: sp.After, Artifacts: sp.Artifacts, NeedsSet: sp.NeedsSet, From: sp.From, FromSet: sp.FromSet, Expire: sp.Expire.Seconds()}
		for _, n := range sp.Needs {
			plan.Needs = append(plan.Needs, n.Name)
		}
		body, _ := json.Marshal(plan)
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
			if _, _, err := a.createRun(ctx, atual, run, ownerRow, branch, fmt.Sprint(u["depois"])); err != nil && !errorsAs(err, new(errNoSteps)) {
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

// advance moves a run forward. A created step becomes pending (or manual)
// when what it waits for ended well: the steps it needs (precisa) or, by
// default, every step of the earlier stages. A step whose prerequisites
// failed is skipped, unless it runs always (sempre). A manual step that may
// not fail blocks what waits for it until someone starts it; one that may
// fail does not block the stages after it. The run's state summarises its
// steps.
func (a *intentAPI) advance(ctx *interp.Context, run *ast.Entity, runID any) {
	lock := a.runLock(runID)
	lock.Lock()
	defer lock.Unlock()
	step := a.app.Entities[run.Execution.Step]
	jobs := a.steps(ctx, run, runID)
	for _, j := range progress(jobs) {
		a.in.Op(ctx, step.Singular, "atualizar", j["id"], map[string]any{"estado": j["estado"]})
	}
	st, done := summarize(jobs)
	res, _ := a.in.Op(ctx, run.Singular, "buscar", runID)
	row, _ := res.(map[string]any)
	if row == nil {
		return
	}
	current := toStr(row["estado"])
	if !done {
		if current != st {
			a.in.Op(ctx, run.Singular, "atualizar", runID, map[string]any{"estado": st})
		}
		return
	}
	if current == st && toStr(row["terminado_em"]) != "" {
		return // already finished like this: no second event
	}
	change := map[string]any{"estado": st, "terminado_em": now()}
	if started := toStr(row["iniciado_em"]); started != "" {
		if t0, err := time.Parse(time.RFC3339, started); err == nil {
			change["duracao"] = time.Since(t0).Seconds()
		}
	}
	a.in.Op(ctx, run.Singular, "atualizar", runID, change)
	if res, _ := a.in.Op(ctx, run.Singular, "buscar", runID); res != nil {
		a.emit(ctx, run, st, res.(map[string]any), nil) // run finished
		a.history(ctx, nil, run, st, nil, res.(map[string]any))
	}
}

func allowedToFail(j map[string]any) bool { b, _ := j["permitir_falha"].(bool); return b }

// progress decides, in memory, which created steps may leave that state
// (see advance) and returns the ones it changed (their estado is updated).
func progress(jobs []map[string]any) []map[string]any {
	g := newStepGraph(jobs)
	var out []map[string]any
	for changed := true; changed; {
		changed = false
		for _, j := range jobs {
			if toStr(j["estado"]) != stCreated {
				continue
			}
			explicit := g.plans[toStr(j["id"])].NeedsSet
			waiting, broken := false, false
			for _, p := range g.prerequisites(j) {
				switch toStr(p["estado"]) {
				case stSuccess:
				case stFailed:
					broken = broken || !allowedToFail(p)
				case stCanceled, stSkipped:
					broken = true
				case stManual:
					// a step that needs it by name waits for it to run;
					// a later stage waits only when it may not fail
					waiting = waiting || explicit || !allowedToFail(p)
				default: // created, pending, running
					waiting = true
				}
			}
			if waiting {
				continue
			}
			switch when := toStr(j["quando"]); {
			case broken && when != whenAlways:
				j["estado"] = stSkipped
			case when == whenManual:
				j["estado"] = stManual
			default:
				j["estado"] = stPending
			}
			out = append(out, j)
			changed = true
		}
	}
	return out
}

// summarize gives the run's state from its steps; done says it is final.
// Running steps keep it running; a step waiting for a manual one that may
// not fail leaves it manual; then a failure (not allowed) fails it, a
// cancellation cancels it, and otherwise it succeeded.
func summarize(jobs []map[string]any) (string, bool) {
	busy, blocked, failed, canceled := false, false, false, false
	for _, j := range jobs {
		switch toStr(j["estado"]) {
		case stPending, stRunning:
			busy = true
		case stCreated:
			blocked = true // waiting for a manual step
		case stManual:
			blocked = blocked || !allowedToFail(j)
		case stFailed:
			failed = failed || !allowedToFail(j)
		case stCanceled:
			canceled = true
		}
	}
	switch {
	case busy:
		return stRunning, false
	case blocked:
		return stManual, false
	case failed:
		return stFailed, true
	case canceled:
		return stCanceled, true
	}
	return stSuccess, true
}

// stepGraph answers, for the current steps of one run, what each waits for
// and whose files it receives.
type stepGraph struct {
	jobs   []map[string]any
	byName map[string]map[string]any
	plans  map[string]stepPlan
}

func newStepGraph(jobs []map[string]any) *stepGraph {
	g := &stepGraph{jobs: jobs, byName: map[string]map[string]any{}, plans: map[string]stepPlan{}}
	for _, j := range jobs {
		g.byName[toStr(j["nome"])] = j
		g.plans[toStr(j["id"])] = planOf(j)
	}
	return g
}

// prerequisites: the steps j needs by name, or every step of earlier stages.
func (g *stepGraph) prerequisites(j map[string]any) []map[string]any {
	plan := g.plans[toStr(j["id"])]
	var out []map[string]any
	if plan.NeedsSet {
		for _, n := range plan.Needs {
			if p := g.byName[n]; p != nil {
				out = append(out, p)
			}
		}
		return out
	}
	order := asNumber(j["ordem"])
	for _, p := range g.jobs {
		if asNumber(p["ordem"]) < order {
			out = append(out, p)
		}
	}
	return out
}

// sources: the steps whose artifacts j receives (recebe_artefatos_de, or
// what it waits for).
func (g *stepGraph) sources(j map[string]any) []map[string]any {
	plan := g.plans[toStr(j["id"])]
	if !plan.FromSet {
		return g.prerequisites(j)
	}
	var out []map[string]any
	for _, n := range plan.From {
		if p := g.byName[n]; p != nil {
			out = append(out, p)
		}
	}
	return out
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
// logBuffer keeps a step's log. Hidden values (GEP 0015) are masked by
// whole lines, so a secret written in two pieces is never stored half
// unmasked; String gives the complete lines, Final also the last partial one.
type logBuffer struct {
	mu      sync.Mutex
	buf     strings.Builder
	pending string
	masks   []string
}

func (l *logBuffer) mask(values ...string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, v := range values {
		if len(v) >= 4 { // too short a value would mask ordinary text
			l.masks = append(l.masks, v)
		}
	}
}

func (l *logBuffer) masked(s string) string {
	for _, m := range l.masks {
		s = strings.ReplaceAll(s, m, "[MASKED]")
	}
	return s
}

func (l *logBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.buf.Len() >= 4<<20 {
		return len(p), nil
	}
	l.pending += string(p)
	if i := strings.LastIndexByte(l.pending, '\n'); i >= 0 {
		l.buf.WriteString(l.masked(l.pending[:i+1]))
		l.pending = l.pending[i+1:]
	}
	return len(p), nil
}

func (l *logBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}

// Final is the whole log, the last partial line included.
func (l *logBuffer) Final() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String() + l.masked(l.pending)
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
	go func() { // stream the log while the job runs, appending only what grew
		t := time.NewTicker(time.Second)
		defer t.Stop()
		written := 0
		for {
			select {
			case <-done:
				return
			case <-t.C:
				text := log.String()
				if len(text) == written {
					continue
				}
				if ok, err := a.s.DB.AnexarTexto(step.Singular, int64(asNumber(id)), "log", text[written:], written); err == nil && ok {
					written = len(text)
				} else {
					// the stored log changed meanwhile: write it whole once
					a.in.Op(ctx, step.Singular, "atualizar", id, map[string]any{"log": text})
					written = len(text)
				}
			}
		}
	}()
	status := x.execute(jctx, log, step, ownerRow, runRow, job, spec.Script, spec.After)
	close(done)
	if jctx.Err() == context.Canceled {
		status = stCanceled
	}
	change := map[string]any{"estado": status, "terminado_em": now(), "log": log.Final()}
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
func (x *executor) execute(ctx context.Context, log *logBuffer, step *ast.Entity, owner, run, job map[string]any, script, after []string) string {
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
	env := []string{"PATH=" + os.Getenv("PATH"), "HOME=" + dir}
	local := x.a.localWork(owner, run, job, work)
	for k, v := range x.a.stepVariables(&interp.Context{}, local) {
		env = append(env, k+"="+v)
	}
	log.mask(hiddenValues(local)...)
	x.a.unpackArtifacts(step, job, work, log)
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
	if ok && ctx.Err() == nil {
		ok = x.a.collectArtifacts(step, job, work, log)
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
				"ordem": row["ordem"], "script": row["script"], "quando": whenAuto, "permitir_falha": row["permitir_falha"], "imagem": row["imagem"]})
			if err != nil {
				return nil, err
			}
			// what was skipped because of it waits again
			for _, j := range a.steps(ctx, run, row[x.RunField]) {
				if toStr(j["estado"]) == stSkipped {
					a.in.Op(ctx, e.Singular, "atualizar", j["id"], map[string]any{"estado": stCreated})
				}
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

// runVariables: the variables of the owner of an execution (GEP 0015):
// [{nome, valor, oculto}].
func (a *intentAPI) runVariables(ctx *interp.Context, run *ast.Entity, owner map[string]any) []any {
	x := run.Execution
	out := []any{}
	if x == nil || x.Variables == "" || owner == nil {
		return out
	}
	ve := a.app.Entities[x.Variables]
	res, err := a.in.Op(ctx, ve.Singular, "filtrar", map[string]any{x.VariablesOwner: owner["id"]}, map[string]any{"limite": 500})
	if err != nil {
		return out
	}
	nameField := "chave"
	if fieldOf(ve, "chave") == nil {
		nameField = "nome"
	}
	hidden := false
	if f := fieldOf(ve, "valor"); f != nil {
		hidden = f.Hidden || f.IsSecret()
	}
	for _, it := range res.([]any) {
		row := it.(map[string]any)
		out = append(out, map[string]any{"nome": toStr(row[nameField]), "valor": toStr(row["valor"]), "oculto": hidden})
	}
	return out
}

func hiddenValues(work map[string]any) []string {
	var out []string
	for _, it := range asList(work["variaveis"]) {
		if v, _ := it.(map[string]any); v != nil && v["oculto"] == true {
			out = append(out, toStr(v["valor"]))
		}
	}
	return out
}

// validEnvName: a name a shell accepts as a variable.
func validEnvName(s string) bool {
	for i, r := range s {
		if !(r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || i > 0 && r >= '0' && r <= '9') {
			return false
		}
	}
	return s != ""
}
