package servidor

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/runtime/banco"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Remote executors (`runners executam jobs`): records holding a credential
// take pending steps, stream their log and report the result, through the
// same state machine the local executor uses. The runtime speaks one remote
// executor protocol with Portuguese names; the integration vocabulary gives
// them the external names a given executor program expects:
//
//	POST  <executores>/verificar  {token}          → 200 | 403
//	POST  <etapas>/pedir          {token}          → 201 trabalho | 204 nada a fazer
//	PUT   <etapas>/<id>           {token, estado}  → 200 | 403
//	PATCH <etapas>/<id>/log       texto, Content-Range, token no cabeçalho → 202 | 416 | 403
//
// The step token is shown once, when the step is taken; it is kept as a
// digest, dies when the step ends and lets its holder read the repository
// of that step only (never write).

const maxRemoteLog = 4 << 20

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

// remoteSteps: the step entities some executor may run remotely.
func (a *intentAPI) remoteSteps() []*ast.Entity {
	var out []*ast.Entity
	for _, n := range a.app.Order {
		if e := a.app.Entities[n]; e.Execution != nil && e.Execution.Role == "step" && e.Execution.Executor != "" {
			out = append(out, e)
		}
	}
	return out
}

// executorOwnerField: an executor that belongs to an owner (a project
// runner) only takes that owner's steps.
func (a *intentAPI) executorOwnerField(step *ast.Entity) string {
	ex := a.app.Entities[step.Execution.Executor]
	for _, f := range ex.Model.Fields {
		if f.Reference == step.Execution.Owner {
			return strings.ToLower(f.Name)
		}
	}
	return ""
}

// work is what an executor receives with a step.
type work struct {
	step           *ast.Entity
	job, run, dono map[string]any
	token          string
	script, after  []any
}

// take claims the oldest pending step the executor may run (atomically).
func (a *intentAPI) take(ctx *interp.Context, step *ast.Entity, executor map[string]any) (*work, error) {
	run := a.app.Entities[step.Execution.Run]
	owner := a.app.Entities[run.Execution.Owner]
	ownerOnly := executor[a.executorOwnerField(step)]
	db := a.s.DB
	if ctx.DB != nil {
		db = ctx.DB
	}
	for page := 1; ; page++ {
		res, err := a.in.Op(ctx, step.Singular, "filtrar", map[string]any{"estado": stPending}, map[string]any{"ordenar": "id", "limite": 100, "pagina": page})
		if err != nil {
			return nil, err
		}
		rows := res.([]any)
		for _, it := range rows {
			job := it.(map[string]any)
			rr, _ := a.in.Op(ctx, run.Singular, "buscar", job[step.Execution.RunField])
			runRow, _ := rr.(map[string]any)
			if runRow == nil || (ownerOnly != nil && toStr(runRow[run.Execution.OwnerField]) != toStr(ownerOnly)) {
				continue
			}
			token := newStepToken()
			n, err := db.AtualizarOnde(step.Singular, banco.Consulta{Filtros: map[string]any{"id": job["id"], "estado": stPending}},
				map[string]any{"estado": stRunning, "iniciado_em": now(), step.Execution.ExecutorField: executor["id"], "token_execucao": stepDigest(token)})
			if err != nil || n != 1 {
				continue // taken by someone else meanwhile
			}
			change := map[string]any{"estado": stRunning}
			if toStr(runRow["iniciado_em"]) == "" {
				change["iniciado_em"] = now()
			}
			a.in.Op(ctx, run.Singular, "atualizar", runRow["id"], change)
			or, _ := a.in.Op(ctx, owner.Singular, "buscar", runRow[run.Execution.OwnerField])
			ownerRow, _ := or.(map[string]any)
			jr, _ := a.in.Op(ctx, step.Singular, "buscar", job["id"])
			job, _ = jr.(map[string]any)
			var spec struct{ Script, After []string }
			json.Unmarshal([]byte(toStr(job["script"])), &spec)
			w := &work{step: step, job: job, run: runRow, dono: ownerRow, token: token, script: []any{}, after: []any{}}
			for _, l := range spec.Script {
				w.script = append(w.script, l)
			}
			for _, l := range spec.After {
				w.after = append(w.after, l)
			}
			return w, nil
		}
		if len(rows) < 100 {
			return nil, nil
		}
	}
}

// stepByToken finds the running step that holds token.
func (a *intentAPI) stepByToken(ctx *interp.Context, token string) (*ast.Entity, map[string]any) {
	if strings.TrimSpace(token) == "" {
		return nil, nil
	}
	for _, step := range a.remoteSteps() {
		res, err := a.in.Op(ctx, step.Singular, "encontrar", map[string]any{"token_execucao": stepDigest(token), "estado": stRunning})
		if row, _ := res.(map[string]any); err == nil && row != nil {
			return step, row
		}
	}
	return nil, nil
}

// appendLog adds text at offset start; it is refused (ok=false) unless start
// is the current length. It returns the length after the call.
func (a *intentAPI) appendLog(ctx *interp.Context, step *ast.Entity, row map[string]any, text string, start int) (int, bool, error) {
	current := toStr(row["log"])
	if start != len(current) {
		return len(current), false, nil
	}
	if len(current)+len(text) > maxRemoteLog {
		text = text[:max(0, maxRemoteLog-len(current))]
	}
	if _, err := a.in.Op(ctx, step.Singular, "atualizar", row["id"], map[string]any{"log": current + text}); err != nil {
		return 0, false, err
	}
	return len(current) + len(text), true, nil
}

// finish records the result of a running step and moves its run forward.
func (a *intentAPI) finish(ctx *interp.Context, step *ast.Entity, row map[string]any, state string) error {
	if toStr(row["estado"]) != stRunning {
		return nil // already finished (or canceled meanwhile)
	}
	change := map[string]any{"estado": state, "terminado_em": now(), "token_execucao": nil}
	if t0, err := time.Parse(time.RFC3339, toStr(row["iniciado_em"])); err == nil {
		change["duracao"] = time.Since(t0).Seconds()
	}
	if _, err := a.in.Op(ctx, step.Singular, "atualizar", row["id"], change); err != nil {
		return err
	}
	a.advance(ctx, a.app.Entities[step.Execution.Run], row[step.Execution.RunField])
	return nil
}

// ---------- protocol ----------

// mountRemote serves the remote executor protocol for every `X executam Y`,
// under the integration prefix with its vocabulary when both are integrated.
func (a *intentAPI) mountRemote(mux *routeMux) {
	for _, step := range a.remoteSteps() {
		ex := a.app.Entities[step.Execution.Executor]
		p := *a
		p.extern = ex.Integrate != "" && step.Integrate != ""
		exBase, stBase := "/_ge/api/"+ex.Plural, "/_ge/api/"+step.Plural
		if p.extern {
			exBase, stBase = a.app.Integration+"/"+ex.Integrate, a.app.Integration+"/"+step.Integrate
		}
		step, ex, pp := step, ex, &p
		handle := func(pattern string, fn func(w http.ResponseWriter, r *http.Request, ctx *interp.Context)) {
			mux.claim(pattern, func(w http.ResponseWriter, r *http.Request) {
				a.s.transactional(w, r, func(w http.ResponseWriter, r *http.Request) { fn(w, r, newContext(w, r)) })
			})
		}
		handle("POST "+exBase+"/"+pp.ext("verificar"), func(w http.ResponseWriter, r *http.Request, ctx *interp.Context) {
			body, _ := readBody(r)
			if row := pp.executorOf(ctx, ex, body); row != nil {
				pp.json(w, 200, map[string]any{"id": row["id"], "token": body["token"]}, nil)
				return
			}
			pp.fail(w, 403, "403 Forbidden")
		})
		handle("POST "+stBase+"/"+pp.ext("pedir"), func(w http.ResponseWriter, r *http.Request, ctx *interp.Context) {
			body, _ := readBody(r)
			row := pp.executorOf(ctx, ex, body)
			if row == nil {
				pp.fail(w, 403, "403 Forbidden")
				return
			}
			wk, err := pp.take(ctx, step, row)
			if err != nil {
				pp.failErr(w, r, err)
				return
			}
			if wk == nil {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			pp.json(w, 201, pp.workPayload(r, wk), nil)
		})
		handle("PUT "+stBase+"/{id}", func(w http.ResponseWriter, r *http.Request, ctx *interp.Context) {
			body, _ := readBody(r)
			body = pp.inwardBody(step, body)
			st, row := pp.stepByToken(ctx, toStr(body["token"]))
			if row == nil || toStr(row["id"]) != r.PathValue("id") {
				pp.fail(w, 403, "403 Forbidden")
				return
			}
			state := toStr(body["estado"])
			switch state {
			case stSuccess, stFailed, stCanceled:
				if err := pp.finish(ctx, st, row, state); err != nil {
					pp.failErr(w, r, err)
					return
				}
			default:
				state = stRunning
			}
			w.Header().Set(pp.ext("cabecalho_estado_etapa"), pp.ext(state))
			w.WriteHeader(http.StatusOK)
		})
		handle("PATCH "+stBase+"/{id}/"+pp.ext("log"), func(w http.ResponseWriter, r *http.Request, ctx *interp.Context) {
			st, row := pp.stepByToken(ctx, r.Header.Get(pp.ext("cabecalho_token_etapa")))
			if row == nil || toStr(row["id"]) != r.PathValue("id") {
				pp.fail(w, 403, "403 Forbidden")
				return
			}
			start := 0
			if cr := r.Header.Get("Content-Range"); cr != "" {
				start, _ = strconv.Atoi(strings.TrimSpace(strings.SplitN(strings.TrimPrefix(cr, "bytes "), "-", 2)[0]))
			}
			text, err := io.ReadAll(io.LimitReader(r.Body, maxRemoteLog))
			if err != nil {
				pp.fail(w, 400, "log inválido")
				return
			}
			size, ok, err := pp.appendLog(ctx, st, row, string(text), start)
			if err != nil {
				pp.failErr(w, r, err)
				return
			}
			w.Header().Set("Range", "0-"+strconv.Itoa(size))
			w.Header().Set(pp.ext("cabecalho_estado_etapa"), pp.ext(stRunning))
			w.Header().Set(pp.ext("cabecalho_intervalo_log"), "3")
			if !ok {
				w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return
			}
			w.WriteHeader(http.StatusAccepted)
		})
	}
}

// executorOf finds an active executor by the credential in body["token"].
func (a *intentAPI) executorOf(ctx *interp.Context, ex *ast.Entity, body map[string]any) map[string]any {
	token := toStr(body["token"])
	if token == "" {
		return nil
	}
	res, err := a.in.Op(ctx, ex.Singular, "por_segredo", token)
	row, _ := res.(map[string]any)
	if err != nil || row == nil {
		return nil
	}
	if b, ok := row["ativo"].(bool); ok && !b {
		return nil
	}
	return row
}

// workPayload describes a taken step: where its code is, what to run and
// the variables it sees. Names follow the vocabulary; a variable exists only
// when the vocabulary names it (variavel_<nome> é "NOME_EXTERNO").
func (a *intentAPI) workPayload(r *http.Request, w *work) map[string]any {
	run := a.app.Entities[w.step.Execution.Run]
	owner := a.app.Entities[run.Execution.Owner]
	server := "http://" + r.Host
	if r.TLS != nil {
		server = "https://" + r.Host
	}
	path := ""
	if w.dono != nil && owner.RepoKey != "" {
		path = toStr(w.dono[owner.RepoKey])
	}
	branch, sha := toStr(w.run["branch"]), toStr(w.run["versao"])
	repo := strings.Replace(server, "://", "://"+a.ext("usuario_token_etapa")+":"+w.token+"@", 1) + "/" + path + ".git"
	values := map[string]any{
		"ci": "true", "servidor": server, "etapa_id": w.job["id"], "etapa_nome": w.job["nome"], "etapa_estagio": w.job["etapa"],
		"token_etapa": w.token, "execucao_id": w.run["id"], "versao": sha, "branch": branch,
		"dono_id": w.dono["id"], "dono_caminho": path, "endereco_git": repo,
	}
	var vars []any
	for _, k := range []string{"ci", "servidor", "etapa_id", "etapa_nome", "etapa_estagio", "token_etapa", "execucao_id", "versao", "branch", "dono_id", "dono_caminho", "endereco_git"} {
		name := a.ext("variavel_" + k)
		if name == "variavel_"+k {
			continue // not named by the vocabulary: not sent
		}
		public := k != "token_etapa" && k != "endereco_git"
		vars = append(vars, map[string]any{"chave": name, "valor": toStr(values[k]), "publica": public, "mascarada": !public})
	}
	steps := []any{map[string]any{"nome": "script", "script": w.script, "tempo_limite": 3600, "quando": "on_success", "permitir_falha": false}}
	if len(w.after) > 0 {
		steps = append(steps, map[string]any{"nome": "after_script", "script": w.after, "tempo_limite": 3600, "quando": "always", "permitir_falha": true})
	}
	out := map[string]any{
		"id": w.job["id"], "token": w.token, "buscar_git": true,
		"info_etapa":    map[string]any{"id": w.job["id"], "nome": w.job["nome"], "etapa": w.job["etapa"], owner.Singular + "_id": w.dono["id"], owner.Singular + "_nome": w.dono["nome"]},
		"git":           map[string]any{"endereco_git": repo, "branch": branch, "versao": sha, "versao_anterior": strings.Repeat("0", 40), "tipo_ref": "branch", "refspecs": []any{"+refs/heads/" + branch + ":refs/remotes/origin/" + branch}, "profundidade": 20},
		"info_executor": map[string]any{"tempo_limite": 3600},
		"variaveis":     vars, "passos": steps,
		"servicos": []any{}, "artefatos": []any{}, "cache": []any{}, "credenciais": []any{}, "dependencias": []any{},
	}
	if img := toStr(w.job["imagem"]); img != "" {
		out["imagem"] = map[string]any{"nome": img}
	}
	return out
}

// stepOwnerIs: the step belongs to a run of this very record.
func (a *intentAPI) stepOwnerIs(ctx *interp.Context, step *ast.Entity, job map[string]any, e *ast.Entity, row map[string]any) bool {
	run := a.app.Entities[step.Execution.Run]
	if run.Execution.Owner != e.Singular {
		return false
	}
	rr, _ := a.in.Op(ctx, run.Singular, "buscar", job[step.Execution.RunField])
	runRow, _ := rr.(map[string]any)
	return runRow != nil && toStr(runRow[run.Execution.OwnerField]) == toStr(row["id"])
}

// gitProtocol serves a read of the repository (no hooks: nothing changes).
func (a *intentAPI) gitProtocol(w http.ResponseWriter, r *http.Request, ctx *interp.Context, row map[string]any, service string, advertise bool) {
	repo, _ := row["repositorio"].(string)
	if _, err := a.s.Git.ServeHTTP(w, r, repo, service, advertise, nil); err != nil {
		http.Error(w, "git: "+err.Error(), http.StatusInternalServerError)
	}
}
