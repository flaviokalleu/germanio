package servidor

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/runtime/banco"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Remote work — `X executam Y`: records of Y are work that executors
// (records of X, each with a secret credential) take and run elsewhere: build
// agents, workers, render farms. This is the whole mechanism; it knows no
// executor program. How one particular program talks (paths, payloads,
// headers) lives in an integration adapter written in .ge, which calls the
// `trabalho_remoto` module registered below.
//
//	executor   an active record of X found by its credential; every contact
//	           refreshes its own heartbeat (visto_em) — executor health is not
//	           the same as owning a piece of work.
//	claim      the oldest pending work the executor may run, atomically: two
//	           executors never get the same work. An executor that belongs to
//	           an owner only takes that owner's work. A retried claim with the
//	           same key returns the same work (with a fresh token).
//	lease      a claim holds the work until reserva_ate; renovar and each log
//	           chunk extend it. An expired lease sends the work back to the
//	           queue (up to remoteAttempts), then fails it. Work running past
//	           its time limit fails.
//	token      random, shown once, stored as a SHA-256 digest; it reaches only
//	           its own work (and, while it runs, reads the repository of that
//	           work's owner). When its lease expires it dies at once; when the
//	           work ends (or a cancellation is acknowledged) it only answers
//	           retries of concluir/estado for one lease period, then dies.
//	log        appended by offset; re-sending a chunk already stored is
//	           accepted without duplicating it.
//	cancel     people cancel through the entity's own action; the executor
//	           learns it from any call (cancelado: verdadeiro) and acknowledges
//	           with concluir(token, "cancelado").
//	result     concluir(token, sucesso|falhou) — repeating it is harmless.

const (
	remoteAttempts = 3
	remoteMaxLog   = 4 << 20
)

func remoteLease() time.Duration {
	if d, err := time.ParseDuration(os.Getenv("GERMANIO_RESERVA")); err == nil && d > 0 {
		return d
	}
	return 2 * time.Minute
}

func remoteLimit() time.Duration {
	if d, err := time.ParseDuration(os.Getenv("GERMANIO_JOB_TIMEOUT")); err == nil && d > 0 {
		return d
	}
	return time.Hour
}

func stepDigest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func newStepToken() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func parseStamp(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, s)
	}
	return t
}

// remoteWork: entities whose records executors take.
func (a *intentAPI) remoteWork() []*ast.Entity {
	var out []*ast.Entity
	for _, n := range a.app.Order {
		if e := a.app.Entities[n]; e.Remote != nil {
			out = append(out, e)
		}
	}
	return out
}

func (a *intentAPI) dbOf(ctx *interp.Context) *banco.Banco {
	if ctx != nil && ctx.DB != nil {
		return ctx.DB
	}
	return a.s.DB
}

// executorByCredential finds an active executor and records its heartbeat.
func (a *intentAPI) executorByCredential(ctx *interp.Context, credential string) (*ast.Entity, map[string]any) {
	if strings.TrimSpace(credential) == "" {
		return nil, nil
	}
	seen := map[string]bool{}
	for _, w := range a.remoteWork() {
		ex := a.app.Entities[w.Remote.Executor]
		if seen[ex.Singular] {
			continue
		}
		seen[ex.Singular] = true
		res, err := a.in.Op(ctx, ex.Singular, "por_segredo", credential)
		row, _ := res.(map[string]any)
		if err != nil || row == nil {
			continue
		}
		if b, ok := row["ativo"].(bool); ok && !b {
			return nil, nil
		}
		a.dbOf(ctx).AtualizarOnde(ex.Singular, banco.Consulta{Filtros: map[string]any{"id": row["id"]}}, map[string]any{"visto_em": now()})
		return ex, row
	}
	return nil, nil
}

// ownerOnly: the owner an executor is limited to (nil: any).
func (a *intentAPI) ownerOnly(w *ast.Entity, executor map[string]any) (field string, value any) {
	ex := a.app.Entities[w.Remote.Executor]
	target := ""
	if w.Execution != nil {
		target = w.Execution.Owner
	} else {
		for f, t := range w.Parents {
			for _, ef := range ex.Model.Fields {
				if ef.Reference == t {
					field, target = f, t
				}
			}
		}
	}
	for _, f := range ex.Model.Fields {
		if f.Reference == target && target != "" && executor[strings.ToLower(f.Name)] != nil {
			return field, executor[strings.ToLower(f.Name)]
		}
	}
	return "", nil
}

// ownerOf is the owner record of a piece of work (the run's owner for steps).
func (a *intentAPI) workOwnerMatches(ctx *interp.Context, w *ast.Entity, row map[string]any, field string, value any) bool {
	if value == nil {
		return true
	}
	if w.Execution != nil {
		run := a.app.Entities[w.Execution.Run]
		rr, _ := a.in.Op(ctx, run.Singular, "buscar", row[w.Execution.RunField])
		runRow, _ := rr.(map[string]any)
		return runRow != nil && toStr(runRow[run.Execution.OwnerField]) == toStr(value)
	}
	return toStr(row[field]) == toStr(value)
}

// claim reserves work for an executor (see the file comment).
func (a *intentAPI) claim(ctx *interp.Context, ex *ast.Entity, executor map[string]any, key string) (map[string]any, error) {
	db := a.dbOf(ctx)
	for _, w := range a.remoteWork() {
		if w.Remote.Executor != ex.Singular {
			continue
		}
		rw := w.Remote
		if key != "" { // a repeated claim: same work, fresh token
			res, _ := a.in.Op(ctx, w.Singular, "encontrar", map[string]any{rw.ExecutorField: executor["id"], "chave_reserva": key, "estado": a.running(w)})
			if row, _ := res.(map[string]any); row != nil && parseStamp(toStr(row["reserva_ate"])).After(time.Now()) {
				token := newStepToken()
				db.AtualizarOnde(w.Singular, banco.Consulta{Filtros: map[string]any{"id": row["id"]}}, map[string]any{"token_execucao": stepDigest(token), "reserva_ate": stamp(time.Now().Add(remoteLease()))})
				return a.payload(ctx, w, row["id"], token), nil
			}
		}
		field, value := a.ownerOnly(w, executor)
		for page := 1; ; page++ {
			res, err := a.in.Op(ctx, w.Singular, "filtrar", map[string]any{"estado": rw.Pending}, map[string]any{"ordenar": "id", "limite": 100, "pagina": page})
			if err != nil {
				return nil, err
			}
			rows := res.([]any)
			for _, it := range rows {
				row := it.(map[string]any)
				if !a.workOwnerMatches(ctx, w, row, field, value) {
					continue
				}
				token := newStepToken()
				change := map[string]any{"estado": a.running(w), "iniciado_em": now(), "reserva_ate": stamp(time.Now().Add(remoteLease())),
					rw.ExecutorField: executor["id"], "token_execucao": stepDigest(token), "chave_reserva": nilIfBlank(key)}
				n, err := db.AtualizarOnde(w.Singular, banco.Consulta{Filtros: map[string]any{"id": row["id"], "estado": rw.Pending}}, change)
				if err != nil || n != 1 {
					continue // another executor took it first
				}
				if w.Execution != nil {
					a.stepStarted(ctx, w, row)
				}
				return a.payload(ctx, w, row["id"], token), nil
			}
			if len(rows) < 100 {
				break
			}
		}
	}
	return nil, nil
}

func nilIfBlank(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (a *intentAPI) running(w *ast.Entity) string { return stRunning }

// stepStarted: a pipeline step taken remotely also starts its run.
func (a *intentAPI) stepStarted(ctx *interp.Context, w *ast.Entity, row map[string]any) {
	run := a.app.Entities[w.Execution.Run]
	rr, _ := a.in.Op(ctx, run.Singular, "buscar", row[w.Execution.RunField])
	runRow, _ := rr.(map[string]any)
	if runRow == nil {
		return
	}
	change := map[string]any{"estado": stRunning}
	if toStr(runRow["iniciado_em"]) == "" {
		change["iniciado_em"] = now()
	}
	a.in.Op(ctx, run.Singular, "atualizar", runRow["id"], change)
}

// payload is what the executor receives: the work and, for pipeline steps,
// its commands and where its code is.
func (a *intentAPI) payload(ctx *interp.Context, w *ast.Entity, id any, token string) map[string]any {
	res, _ := a.in.Op(ctx, w.Singular, "buscar", id)
	row, _ := res.(map[string]any)
	out := map[string]any{"id": row["id"], "token": token, "dados": serialize(w, row), "reserva_ate": row["reserva_ate"],
		"reserva_segundos": remoteLease().Seconds(), "limite_segundos": remoteLimit().Seconds(), "tentativa": asNumber(row["tentativas"]) + 1}
	if w.Execution == nil {
		return out
	}
	run := a.app.Entities[w.Execution.Run]
	owner := a.app.Entities[run.Execution.Owner]
	rr, _ := a.in.Op(ctx, run.Singular, "buscar", row[w.Execution.RunField])
	runRow, _ := rr.(map[string]any)
	var ownerRow map[string]any
	if runRow != nil {
		or, _ := a.in.Op(ctx, owner.Singular, "buscar", runRow[run.Execution.OwnerField])
		ownerRow, _ = or.(map[string]any)
	}
	var spec struct{ Script, After []string }
	json.Unmarshal([]byte(toStr(row["script"])), &spec)
	commands, after := []any{}, []any{}
	for _, l := range spec.Script {
		commands = append(commands, l)
	}
	for _, l := range spec.After {
		after = append(after, l)
	}
	out["comandos"], out["depois"] = commands, after
	out["execucao"], out["dono"] = serialize(run, runRow), serialize(owner, ownerRow)
	if ownerRow != nil && owner.RepoKey != "" {
		out["repositorio"] = toStr(ownerRow[owner.RepoKey])
	}
	return out
}

// byToken finds the work a token belongs to. A token of running work whose
// lease already expired is no longer valid.
func (a *intentAPI) byToken(ctx *interp.Context, token string) (*ast.Entity, map[string]any) {
	if strings.TrimSpace(token) == "" {
		return nil, nil
	}
	for _, w := range a.remoteWork() {
		res, err := a.in.Op(ctx, w.Singular, "encontrar", map[string]any{"token_execucao": stepDigest(token)})
		row, _ := res.(map[string]any)
		if err != nil || row == nil {
			continue
		}
		until := parseStamp(toStr(row["reserva_ate"]))
		if !until.After(time.Now()) && (toStr(row["estado"]) == stRunning || !until.IsZero()) {
			return nil, nil // lease expired, or the grace after the end passed
		}
		return w, row
	}
	return nil, nil
}

// stepByToken: running work holding token (used for temporary repository access).
func (a *intentAPI) stepByToken(ctx *interp.Context, token string, running bool) (*ast.Entity, map[string]any) {
	w, row := a.byToken(ctx, token)
	if row == nil || (running && toStr(row["estado"]) != stRunning) {
		return nil, nil
	}
	return w, row
}

func (a *intentAPI) status(w *ast.Entity, row map[string]any) map[string]any {
	st := toStr(row["estado"])
	return map[string]any{"id": row["id"], "estado": st, "cancelado": st == w.Remote.Canceled, "terminado": st != stRunning && st != w.Remote.Pending, "reserva_ate": row["reserva_ate"]}
}

// renew extends the lease of running work.
func (a *intentAPI) renew(ctx *interp.Context, w *ast.Entity, row map[string]any) map[string]any {
	if toStr(row["estado"]) == stRunning {
		until := stamp(time.Now().Add(remoteLease()))
		a.dbOf(ctx).AtualizarOnde(w.Singular, banco.Consulta{Filtros: map[string]any{"id": row["id"], "estado": stRunning}}, map[string]any{"reserva_ate": until})
		row["reserva_ate"] = until
	}
	return a.status(w, row)
}

// appendLog adds text at offset start (idempotent for chunks already stored).
func (a *intentAPI) appendLog(ctx *interp.Context, w *ast.Entity, row map[string]any, text string, start int) (map[string]any, error) {
	current := toStr(row["log"])
	out := a.status(w, row)
	out["aceito"], out["tamanho"] = false, float64(len(current))
	if toStr(row["estado"]) != stRunning {
		return out, nil
	}
	if start < len(current) && start+len(text) <= len(current) && current[start:start+len(text)] == text {
		out["aceito"] = true // already stored: a retry
		return a.withStatus(out, a.renew(ctx, w, row)), nil
	}
	if start != len(current) {
		return out, nil
	}
	if len(current)+len(text) > remoteMaxLog {
		text = text[:max(0, remoteMaxLog-len(current))]
	}
	// Append only the new chunk (rewriting the whole log costs the square of
	// its size); a length that changed meanwhile refuses the chunk.
	ok, err := a.dbOf(ctx).AnexarTexto(w.Singular, int64(asNumber(row["id"])), "log", text, len(current))
	if err != nil {
		return nil, err
	}
	if !ok {
		return out, nil
	}
	out["aceito"], out["tamanho"] = true, float64(len(current)+len(text))
	return a.withStatus(out, a.renew(ctx, w, row)), nil
}

func (a *intentAPI) withStatus(dst, src map[string]any) map[string]any {
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// conclude records a result; repeating it, or concluding finished work, changes nothing.
func (a *intentAPI) conclude(ctx *interp.Context, w *ast.Entity, row map[string]any, result string) error {
	return a.concludeWith(ctx, w, row, result, true)
}

// concludeWith: retryGrace=false ends the token in the same update (expired leases).
func (a *intentAPI) concludeWith(ctx *interp.Context, w *ast.Entity, row map[string]any, result string, retryGrace bool) error {
	st := toStr(row["estado"])
	var grace any = stamp(time.Now().Add(remoteLease())) // the token only answers retries until then
	switch {
	case st == w.Remote.Canceled:
		// acknowledging a cancellation: the token only answers retries now
		_, err := a.in.Op(ctx, w.Singular, "atualizar", row["id"], map[string]any{"reserva_ate": grace})
		return err
	case st != stRunning:
		return nil // already concluded: a retry
	case result == w.Remote.Canceled:
		return nil // the executor cannot cancel; only people can
	}
	change := map[string]any{"estado": result, "terminado_em": now(), "reserva_ate": grace}
	if !retryGrace {
		change["reserva_ate"], change["token_execucao"] = nil, nil
	}
	if t0 := parseStamp(toStr(row["iniciado_em"])); !t0.IsZero() {
		change["duracao"] = time.Since(t0).Seconds()
	}
	if _, err := a.in.Op(ctx, w.Singular, "atualizar", row["id"], change); err != nil {
		return err
	}
	if w.Execution != nil {
		a.advance(ctx, a.app.Entities[w.Execution.Run], row[w.Execution.RunField])
	}
	return nil
}

// expire applies the lease policy: expired leases go back to the queue (then
// fail after remoteAttempts); work past its time limit fails; tokens of
// cancelled work that nobody acknowledged die with the lease.
func (a *intentAPI) expire(clock time.Time) {
	ctx := &interp.Context{}
	for _, w := range a.remoteWork() {
		rw := w.Remote
		res, err := a.in.Op(ctx, w.Singular, "filtrar", map[string]any{"token_execucao__diferente": nil}, map[string]any{"limite": 500})
		if err != nil {
			continue
		}
		for _, it := range res.([]any) {
			row := it.(map[string]any)
			until := parseStamp(toStr(row["reserva_ate"]))
			started := parseStamp(toStr(row["iniciado_em"]))
			st := toStr(row["estado"])
			switch {
			case st != stRunning && !until.IsZero() && clock.After(until):
				// finished (or cancelled) work: the token dies with its grace period
				a.in.Op(ctx, w.Singular, "atualizar", row["id"], map[string]any{"token_execucao": nil, "reserva_ate": nil})
			case st != stRunning:
			case !started.IsZero() && clock.Sub(started) > remoteLimit():
				a.note(ctx, w, row, "tempo limite esgotado")
				a.concludeWith(ctx, w, a.reload(ctx, w, row), stFailed, false)
			case !until.IsZero() && clock.After(until):
				tries := int(asNumber(row["tentativas"])) + 1
				a.note(ctx, w, row, "reserva expirada: o executor parou de responder")
				if tries >= remoteAttempts {
					a.in.Op(ctx, w.Singular, "atualizar", row["id"], map[string]any{"tentativas": tries})
					a.concludeWith(ctx, w, a.reload(ctx, w, row), stFailed, false) // an expired lease ends the token at once
					continue
				}
				a.dbOf(ctx).AtualizarOnde(w.Singular, banco.Consulta{Filtros: map[string]any{"id": row["id"], "estado": stRunning}},
					map[string]any{"estado": rw.Pending, "token_execucao": nil, "reserva_ate": nil, "chave_reserva": nil, rw.ExecutorField: nil, "tentativas": tries})
			}
		}
	}
}

func (a *intentAPI) reload(ctx *interp.Context, w *ast.Entity, row map[string]any) map[string]any {
	res, _ := a.in.Op(ctx, w.Singular, "buscar", row["id"])
	fresh, _ := res.(map[string]any)
	if fresh == nil {
		return row
	}
	return fresh
}

func (a *intentAPI) note(ctx *interp.Context, w *ast.Entity, row map[string]any, text string) {
	if fieldOf(w, "log") != nil {
		a.in.Op(ctx, w.Singular, "atualizar", row["id"], map[string]any{"log": toStr(row["log"]) + "\n" + text + "\n"})
	}
}

func (a *intentAPI) startLeases() {
	if len(a.remoteWork()) == 0 {
		return
	}
	every := min(remoteLease()/4, 15*time.Second)
	stop := make(chan struct{})
	a.s.onClose = append(a.s.onClose, func() { close(stop) })
	go func() {
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case c := <-t.C:
				a.expire(c)
			}
		}
	}()
}

// ---------- module for integration adapters ----------

// registerRemoteModule exposes the mechanism to adapters written in .ge:
//
//	trabalho_remoto.executor(credencial)             → executor | nulo
//	trabalho_remoto.pegar(executor [, chave])        → trabalho | nulo
//	trabalho_remoto.renovar(token)                   → situação | nulo
//	trabalho_remoto.estado(token)                    → situação | nulo
//	trabalho_remoto.adicionar_log(token, texto, inicio) → situação + {aceito, tamanho} | nulo
//	trabalho_remoto.concluir(token, resultado)       → situação | nulo
//
// situação = {id, estado, cancelado, terminado, reserva_ate}; nulo = token inválido.
func (a *intentAPI) registerRemoteModule() {
	if len(a.remoteWork()) == 0 {
		return
	}
	withToken := func(c *interp.Call, args []any, fn func(w *ast.Entity, row map[string]any) any) any {
		w, row := a.byToken(c.Ctx(), c.Str(args, 0, "token"))
		if row == nil {
			return nil
		}
		return fn(w, row)
	}
	a.in.RegisterModule("trabalho_remoto", map[string]interp.ModuleFunc{
		"executor": func(c *interp.Call, args []any) any {
			_, row := a.executorByCredential(c.Ctx(), c.Str(args, 0, "credencial"))
			if row == nil {
				return nil
			}
			return row
		},
		"pegar": func(c *interp.Call, args []any) any {
			executor, _ := c.Arg(args, 0, "executor").(map[string]any)
			if executor == nil || executor["id"] == nil {
				panic(c.Fail(403, "executor desconhecido"))
			}
			key := ""
			if len(args) > 1 {
				key = toStr(args[1])
			}
			for _, w := range a.remoteWork() {
				ex := a.app.Entities[w.Remote.Executor]
				res, _ := a.in.Op(c.Ctx(), ex.Singular, "buscar", executor["id"])
				if row, _ := res.(map[string]any); row != nil {
					if b, ok := row["ativo"].(bool); ok && !b {
						panic(c.Fail(403, "executor inativo"))
					}
					out, err := a.claim(c.Ctx(), ex, row, key)
					if err != nil {
						panic(c.Fail(0, "trabalho_remoto.pegar: %v", err))
					}
					if out != nil {
						return out
					}
				}
			}
			return nil
		},
		"renovar": func(c *interp.Call, args []any) any {
			return withToken(c, args, func(w *ast.Entity, row map[string]any) any { return a.renew(c.Ctx(), w, row) })
		},
		"estado": func(c *interp.Call, args []any) any {
			return withToken(c, args, func(w *ast.Entity, row map[string]any) any { return a.status(w, row) })
		},
		"adicionar_log": func(c *interp.Call, args []any) any {
			return withToken(c, args, func(w *ast.Entity, row map[string]any) any {
				out, err := a.appendLog(c.Ctx(), w, row, c.Str(args, 1, "texto"), int(c.Num(args, 2, "inicio")))
				if err != nil {
					panic(c.Fail(0, "trabalho_remoto.adicionar_log: %v", err))
				}
				return out
			})
		},
		"concluir": func(c *interp.Call, args []any) any {
			return withToken(c, args, func(w *ast.Entity, row map[string]any) any {
				result := c.Str(args, 1, "resultado")
				if result != stSuccess && result != stFailed && result != w.Remote.Canceled {
					panic(c.Fail(400, "trabalho_remoto.concluir: resultado %q (use sucesso, falhou ou %s)", result, w.Remote.Canceled))
				}
				if err := a.conclude(c.Ctx(), w, row, result); err != nil {
					panic(c.Fail(0, "trabalho_remoto.concluir: %v", err))
				}
				return a.status(w, a.reload(c.Ctx(), w, row))
			})
		},
	})
}

// ---------- temporary repository access ----------

// stepOwnerIs: running work belongs to a run of this very record.
func (a *intentAPI) stepOwnerIs(ctx *interp.Context, w *ast.Entity, row map[string]any, e *ast.Entity, rec map[string]any) bool {
	if w.Execution == nil {
		return false
	}
	run := a.app.Entities[w.Execution.Run]
	if run.Execution.Owner != e.Singular {
		return false
	}
	rr, _ := a.in.Op(ctx, run.Singular, "buscar", row[w.Execution.RunField])
	runRow, _ := rr.(map[string]any)
	return runRow != nil && toStr(runRow[run.Execution.OwnerField]) == toStr(rec["id"])
}

// gitProtocol serves a read of the repository (no hooks: nothing changes).
func (a *intentAPI) gitProtocol(w http.ResponseWriter, r *http.Request, ctx *interp.Context, row map[string]any, service string, advertise bool) {
	repo, _ := row["repositorio"].(string)
	if _, err := a.s.Git.ServeHTTP(w, r, repo, service, advertise, nil); err != nil {
		http.Error(w, "git: "+err.Error(), http.StatusInternalServerError)
	}
}

// localWork describes a step run by this server in the same shape remote
// executors receive (without a token), so adapters see one format.
func (a *intentAPI) localWork(owner, run, job map[string]any, dir string) map[string]any {
	for _, n := range a.app.Order {
		w := a.app.Entities[n]
		if w.Execution == nil || w.Execution.Role != "step" {
			continue
		}
		out := a.payload(&interp.Context{}, w, job["id"], "")
		delete(out, "token")
		out["diretorio"] = dir
		return out
	}
	return map[string]any{"id": job["id"], "diretorio": dir}
}
