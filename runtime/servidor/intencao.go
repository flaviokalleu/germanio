package servidor

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/parser"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// The intent runtime turns `tenha`, `permita`, `pode`, `quando` and
// `disponibilize` into working operations. Nobody writes routes for them:
// each entity gets the same well-defined surface, mounted for the app's own
// pages (/_ge/api) and, when integrated, under the integration prefix.

type intentAPI struct {
	s   *Servidor
	app *ast.App
	in  *interp.Interpreter
	// extern marks the integration surface: names and state values follow
	// the integration vocabulary (vocabulário da integração).
	extern bool
	// exec runs steps of executions (nil when no local executor is enabled).
	exec *executor
	// recoveryAvailable: password recovery is declared and configured.
	recoveryAvailable bool
	// live announces changes to open pages (GEP 0020).
	live *liveHub
}

func (s *Servidor) registerIntent(mux *routeMux) error {
	app := s.Program.App
	if app == nil {
		return nil
	}
	a := &intentAPI{s: s, app: app, in: s.Interpreter}
	for _, name := range app.Order {
		if x := app.Entities[name].Execution; x != nil && x.Role == "step" && s.Git != nil {
			a.startExecutor()
			break
		}
	}
	s.intent = a
	a.live = newLiveHub(a)
	a.setupReading()
	a.setupTextImages()
	s.registerTaskModule()
	s.tasks().handle("entrega", a.deliver)
	a.mountSearch(mux)
	a.registerRemoteModule()
	a.startLeases()
	for _, name := range app.Order {
		e := app.Entities[name]
		if len(e.Rules) == 0 && e.Integrate == "" {
			continue
		}
		a.mount(mux, "/_ge/api/"+e.Plural, e, false)
		if e.Integrate != "" {
			ext := *a
			ext.extern = true
			ext.mount(mux, app.Integration+"/"+e.Integrate, e, true)
		}
	}
	a.mountIdentity(mux)
	return nil
}

func (a *intentAPI) mount(mux *routeMux, base string, e *ast.Entity, integration bool) {
	a.mountLevel(mux, base, []*ast.Entity{e}, integration)
}

// mountLevel mounts the collection at base for the last entity of chain
// (earlier entities are its ancestors, each addressed by {rN}).
func (a *intentAPI) mountLevel(mux *routeMux, base string, chain []*ast.Entity, integration bool) {
	e := chain[len(chain)-1]
	item := base + "/{r" + strconv.Itoa(len(chain)-1) + "}"
	h := func(op, verb string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !unsafeMethods[r.Method] {
				a.serve(w, r, chain, op, verb)
				return
			}
			a.s.transactional(w, r, func(w http.ResponseWriter, r *http.Request) { a.serve(w, r, chain, op, verb) })
		}
	}
	mux.HandleFunc("GET "+base, h("listar", ""))
	mux.HandleFunc("POST "+base, h("criar", ""))
	mux.HandleFunc("GET "+item, h("ver", ""))
	mux.HandleFunc("PUT "+item, h("editar", ""))
	mux.HandleFunc("PATCH "+item, h("editar", ""))
	mux.HandleFunc("DELETE "+item, h("excluir", ""))
	actions := map[string]bool{}
	for verb := range e.Rules {
		if !isStandard(verb) && verb != "sair" {
			actions[verb] = true
		}
	}
	for verb := range e.Transitions {
		actions[verb] = true
	}
	if e.Approvals {
		actions["aprovar"], actions["desaprovar"] = true, true
	}
	if e.Execution != nil {
		actions["cancelar"] = true
		if e.Execution.Role == "step" {
			actions["repetir"], actions["executar"] = true, true
			path := "log"
			if integration {
				path = a.ext("log")
			}
			mux.HandleFunc("GET "+item+"/"+path, h("log", ""))
		} else {
			actions["repetir"] = true
		}
	}
	if e.Review != nil && e.Review.Target != "" {
		for _, sub := range []string{"mudancas", "commits"} {
			path := sub
			if integration {
				path = a.ext(sub)
			}
			mux.HandleFunc("GET "+item+"/"+path, h("revisao_"+sub, ""))
		}
	}
	a.mountFiles(mux, item, chain, integration)
	if !integration && a.takesImages(e) {
		mux.HandleFunc("GET "+item+"/imagens/{chave}", h("imagem_texto", ""))
		// streamed outside any transaction (the write lock is never held
		// while bytes arrive); the record of the image is one insert after
		mux.HandleFunc("POST "+item+"/imagens", func(w http.ResponseWriter, r *http.Request) { a.serve(w, r, chain, "imagem_texto", "") })
	}
	for verb := range actions {
		path := verb
		if integration {
			path = a.ext(verb)
		}
		mux.HandleFunc("POST "+item+"/"+path, h("acao", verb))
	}
	if (e.HasMembers || e.InheritVia != "") && a.app.MemberModel != "" {
		mux.HandleFunc("POST "+item+"/sair", h("acao", "sair"))
	}
	if len(chain) == 1 && e.Repository && a.s.Git != nil {
		a.mountRepository(mux, base, e)
	}
	if len(chain) >= 3 {
		return
	}
	for _, c := range a.childrenOf(e) {
		name := c.Plural
		if integration && c.Integrate != "" {
			name = c.Integrate
		}
		next := append(append([]*ast.Entity{}, chain...), c)
		if c == e { // subgrupos: one level, no further nesting
			a.mountLevel(mux, item+"/"+name, next, integration)
			continue
		}
		recursive := false
		for _, anc := range chain {
			recursive = recursive || anc == c
		}
		if !recursive {
			a.mountLevel(mux, item+"/"+name, next, integration)
		}
	}
}

func isStandard(v string) bool {
	switch v {
	case "ver", "criar", "editar", "excluir":
		return true
	}
	return false
}

// childrenOf: entities that belong to e, plus members when e has members.
func (a *intentAPI) childrenOf(e *ast.Entity) []*ast.Entity {
	var out []*ast.Entity
	for _, c := range e.Children {
		if c != e.Singular {
			out = append(out, a.app.Entities[c])
		}
	}
	if e.HierarchyField != "" {
		out = append(out, e) // subgrupos
	}
	if e.HasMembers && a.app.MemberModel != "" {
		out = append(out, a.app.Entities[a.app.MemberModel])
	}
	return out
}

// ---------- messages ----------

func (a *intentAPI) msg(kind string, e *ast.Entity) string {
	label := e.Label
	if a.app.Messages == "en" {
		switch kind {
		case "404":
			return "404 " + label + " Not Found"
		case "403":
			return "403 Forbidden"
		case "401":
			return "401 Unauthorized"
		}
	}
	switch kind {
	case "404":
		return label + " inexistente" // "inexistente" has no gender: a tarefa, o projeto
	case "403":
		return "Você não tem permissão para fazer isso"
	case "401":
		return "Entre para continuar"
	}
	return kind
}

func (a *intentAPI) fail(w http.ResponseWriter, status int, msg any) {
	if a.extern && len(a.app.Vocabulary) > 0 {
		msg = a.outward(msg)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"message": msg})
}

func (a *intentAPI) failErr(w http.ResponseWriter, r *http.Request, err error) {
	var re *interp.RuntimeError
	if errors.As(err, &re) && re.Status >= 400 {
		if re.Payload != nil {
			a.fail(w, re.Status, re.Payload)
		} else {
			a.fail(w, re.Status, re.Message)
		}
		return
	}
	a.s.writeRouteError(w, r, &ast.CustomRoute{}, err)
}

func (a *intentAPI) json(w http.ResponseWriter, status int, v any, headers map[string]string) {
	if a.extern && len(a.app.Vocabulary) > 0 {
		v = a.outward(v)
	}
	for k, x := range headers {
		w.Header().Set(k, x)
	}
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// ---------- helpers ----------

func readBody(r *http.Request) (map[string]any, error) {
	out := map[string]any{}
	if r.Body == nil {
		return out, nil
	}
	ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	switch ct {
	case "application/x-www-form-urlencoded", "multipart/form-data":
		r.Body = http.MaxBytesReader(nil, r.Body, maxRouteBody)
		if err := r.ParseForm(); err != nil {
			return nil, err
		}
		for k, v := range r.PostForm {
			if len(v) > 0 {
				out[k] = v[0]
			}
		}
		return out, nil
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, maxRouteBody+1))
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, &interp.RuntimeError{Status: 400, Message: "JSON inválido: " + err.Error()}
	}
	return out, nil
}

// serialize produces the public representation of a record. Private
// fields appear only to the record's owner and administrators.
func serialize(e *ast.Entity, row map[string]any) map[string]any {
	return serializeFor(nil, nil, nil, e, row, true)
}

func serializeFor(ctx *interp.Context, in *interp.Interpreter, atual map[string]any, e *ast.Entity, row map[string]any, trusted bool) map[string]any {
	if row == nil {
		return nil
	}
	out := make(map[string]any, len(row))
	hidden := map[string]bool{}
	showPrivate := trusted || (in != nil && (in.IsAdmin(atual) || in.Owns(atual, e, row)))
	for _, f := range e.Model.Fields {
		if f.Hidden || f.IsSecret() || (f.Private && !showPrivate) {
			hidden[strings.ToLower(f.Name)] = true
		}
	}
	defer func() {
		if in != nil && ctx != nil {
			namesOut(ctx, in, e, out)
		}
	}()
	files := map[string]bool{}
	for _, f := range fileFields(e) {
		files[strings.ToLower(f.Name)] = true
	}
	if in != nil && in.Online != nil && in.App != nil && e.Singular == in.App.LoginEntity && row["id"] != nil {
		out["online"] = in.Online(row["id"])
	}
	if in != nil && in.Decorate != nil {
		in.Decorate(atual, e, row, out)
	}
	for k, v := range row {
		switch {
		case files[k]:
			out[k] = publicMeta(v)
		case hidden[k]:
		case k == "criado_em":
			out["created_at"] = v
		case k == "atualizado_em":
			out["updated_at"] = v
		default:
			out[k] = v
		}
	}
	return out
}

// find loads a record by id or by any unique text field (paths, emails).
func (a *intentAPI) find(ctx *interp.Context, e *ast.Entity, ref string, scope map[string]any) map[string]any {
	try := func(filters map[string]any) map[string]any {
		for k, v := range scope {
			filters[k] = v
		}
		res, err := a.in.Op(ctx, e.Singular, "encontrar", filters)
		if err != nil {
			return nil
		}
		row, _ := res.(map[string]any)
		return row
	}
	if n, err := strconv.ParseInt(ref, 10, 64); err == nil {
		// Inside a parent, numbered children are addressed by their number
		// (/projects/1/issues/3 is issue #3 of project 1).
		if len(scope) > 0 {
			for _, f := range e.Model.Fields {
				if f.NumberedBy != "" {
					if _, ok := scope[strings.ToLower(f.NumberedBy)]; ok {
						return try(map[string]any{strings.ToLower(f.Name): n})
					}
				}
			}
		}
		if row := try(map[string]any{"id": n}); row != nil {
			return row
		}
	}
	for _, f := range e.Model.Fields {
		if f.Unique && (f.Type == ast.FieldTexto || f.Type == ast.FieldEmail) {
			if row := try(map[string]any{strings.ToLower(f.Name): ref}); row != nil {
				return row
			}
		}
	}
	return nil
}

// parentScope returns the filters that tie a child to its parent record.
func (a *intentAPI) parentScope(parent *ast.Entity, parentRow map[string]any, child *ast.Entity) map[string]any {
	if child.Singular == a.app.MemberModel && parent.Singular != a.app.MemberModel {
		return map[string]any{"recurso": parent.Singular, "recurso_id": parentRow["id"]}
	}
	if child == parent && parent.HierarchyField != "" {
		return map[string]any{parent.HierarchyField: parentRow["id"]}
	}
	for field, target := range child.Parents {
		if target == parent.Singular {
			return map[string]any{field: parentRow["id"]}
		}
	}
	return map[string]any{}
}

// writable filters a payload to the fields people may set.
func (a *intentAPI) writable(atual map[string]any, e *ast.Entity, body map[string]any, fixed map[string]any) map[string]any {
	creating := fixed != nil
	out := map[string]any{}
	for _, f := range e.Model.Fields {
		key := strings.ToLower(f.Name)
		v, ok := body[f.Name]
		if !ok {
			v, ok = body[key]
		}
		if !ok || f.System || f.Type == ast.FieldSegredo {
			continue // hidden fields can be set (a webhook token), never read
		}
		if isFileField(f) {
			continue // a file is sent to its own address (GEP 0014), never set as text
		}
		if _, isFixed := fixed[key]; isFixed {
			continue
		}
		if e.Singular == a.app.LoginEntity && (key == "admin" || key == "papel") && !a.in.IsAdmin(atual) {
			continue
		}
		out[key] = v
	}
	for k, v := range fixed {
		out[k] = v
	}
	// Something that belongs to a person is created in the name of whoever
	// creates it; only administrators act in someone else's name.
	if atual != nil && e.Singular != a.app.LoginEntity && e.Singular != a.app.MemberModel {
		for _, owner := range e.OwnerFields {
			_, given := out[owner]
			switch {
			case !creating:
				// Authorship never changes by editing (except by administrators).
				if !a.in.IsAdmin(atual) {
					delete(out, owner)
				}
			case !given || !a.in.IsAdmin(atual):
				out[owner] = atual["id"]
			}
		}
	}
	return out
}

// ---------- operations ----------

func (a *intentAPI) serve(w http.ResponseWriter, r *http.Request, chain []*ast.Entity, op string, verb string) {
	root := chain[0]
	ctx := newContext(w, r)
	atual, err := a.s.identify(ctx, r)
	if err != nil {
		a.fail(w, 401, a.msg("401", root))
		return
	}
	need := "ler"
	if unsafeMethods[r.Method] {
		need = "escrever"
	}
	if !a.s.scopeAllows(ctx, need) {
		a.fail(w, 403, map[string]any{"pt": "O token não tem escopo para esta ação", "en": "insufficient_scope"}[a.app.Messages])
		return
	}
	// Browser sessions must prove writes came from the app (CSRF).
	if _, byToken := ctx.Values["token"]; !byToken && unsafeMethods[r.Method] && r.Header.Get("Authorization") == "" {
		if sess := interp.SessaoDaRequisicao(r); sess != nil {
			got := r.Header.Get("X-CSRF-Token")
			if got == "" {
				got = r.FormValue("_csrf")
			}
			if want, _ := sess["csrf"].(string); want == "" || got != want {
				a.fail(w, 403, "token CSRF ausente ou inválido")
				return
			}
		}
	}
	// Resolve each ancestor inside the previous one; invisible → 404.
	scope := map[string]any{}
	for i, anc := range chain[:len(chain)-1] {
		row := a.find(ctx, anc, r.PathValue("r"+strconv.Itoa(i)), scope)
		if row == nil || !a.in.Can(ctx, atual, anc, "ver", row) {
			a.fail(w, 404, a.msg("404", anc))
			return
		}
		scope = a.parentScope(anc, row, chain[i+1])
	}
	e := chain[len(chain)-1]
	ref := r.PathValue("r" + strconv.Itoa(len(chain)-1))
	deny := func(row map[string]any) {
		if atual == nil {
			a.fail(w, 401, a.msg("401", e))
		} else if row != nil && !a.in.Can(ctx, atual, e, "ver", row) {
			a.fail(w, 404, a.msg("404", e))
		} else {
			a.fail(w, 403, a.msg("403", e))
		}
	}
	switch op {
	case "imagem_texto":
		a.textImageOp(w, r, ctx, atual, e, a.find(ctx, e, ref, scope))
	case "arquivo_ver", "arquivo_enviar", "arquivo_remover":
		a.fileOp(w, r, ctx, atual, e, a.find(ctx, e, ref, scope), op, deny)
	case "listar":
		a.list(w, r, ctx, atual, e, scope, deny)
	case "log":
		row := a.find(ctx, e, ref, scope)
		if row == nil || !a.in.Can(ctx, atual, e, "ver", row) {
			a.fail(w, 404, a.msg("404", e))
			return
		}
		a.stepLog(w, row)
	case "ver", "revisao_mudancas", "revisao_commits":
		row := a.find(ctx, e, ref, scope)
		if row == nil || !a.in.Can(ctx, atual, e, "ver", row) {
			a.fail(w, 404, a.msg("404", e))
			return
		}
		if op != "ver" {
			a.reviewView(w, r, ctx, e, row, op)
			return
		}
		out := serializeFor(ctx, a.in, atual, e, row, false)
		approvalStatus(e, row, out)
		if e.Review != nil && e.Review.Target != "" && fmt.Sprint(row[e.StateField]) == e.Initial {
			if check := a.mergeCheck(ctx, e, row); check != nil {
				out["pode_mesclar"] = check["pode"]
				out["conflitos"] = check["conflitos"]
			}
		}
		a.json(w, 200, out, nil)
	case "criar":
		body, err := readBody(r)
		if err != nil {
			a.failErr(w, r, err)
			return
		}
		body = a.inwardBody(e, body)
		if scope == nil {
			scope = map[string]any{}
		}
		if f := fileIn(e, body); f != "" {
			a.fail(w, 400, fmt.Sprintf("%s é um arquivo: envie-o para o endereço do registro (PUT …/%s, com o arquivo no corpo), não como texto", f, f))
			return
		}
		data := a.writable(atual, e, body, scope)
		if err := a.namesIn(ctx, atual, e, data, nil); err != nil {
			a.failErr(w, r, err)
			return
		}
		if e.Execution != nil && e.Execution.Role == "run" {
			a.manualRun(w, r, ctx, atual, e, data)
			return
		}
		if !a.canCreate(ctx, atual, e, data) {
			if pe := a.hiddenParent(ctx, atual, e, data); pe != nil {
				a.fail(w, 404, a.msg("404", pe))
				return
			}
			deny(nil)
			return
		}
		if err := a.frozenFor(ctx, "criar", e, data, data); err != nil {
			a.failErr(w, r, err)
			return
		}
		a.create(w, r, ctx, atual, e, data, body)
	case "editar":
		row := a.find(ctx, e, ref, scope)
		if row == nil || !a.in.Can(ctx, atual, e, "ver", row) {
			a.fail(w, 404, a.msg("404", e))
			return
		}
		if !a.in.Can(ctx, atual, e, "editar", row) {
			deny(row)
			return
		}
		body, err := readBody(r)
		if err != nil {
			a.failErr(w, r, err)
			return
		}
		body = a.inwardBody(e, body)
		if f := fileIn(e, body); f != "" {
			a.fail(w, 400, fmt.Sprintf("%s é um arquivo: envie-o para o endereço do registro (PUT …/%s, com o arquivo no corpo), não como texto", f, f))
			return
		}
		data := a.writable(atual, e, body, nil)
		for k := range scope {
			delete(data, k)
		}
		if err := a.namesIn(ctx, atual, e, data, row); err != nil {
			a.failErr(w, r, err)
			return
		}
		if err := a.frozenFor(ctx, "editar", e, row, data); err != nil {
			a.failErr(w, r, err)
			return
		}
		if h := e.Hooks["antes_editar"]; h != nil {
			if _, _, err := a.in.RunHook(ctx, h, map[string]any{"atual": nilIfEmpty(atual), "registro": row, e.Singular: row, "dados": data, "entrada": body}); err != nil {
				a.failErr(w, r, err)
				return
			}
		}
		merged := map[string]any{}
		for k, v := range row {
			merged[k] = v
		}
		for k, v := range data {
			merged[k] = v
		}
		if err := a.guards(ctx, atual, e, merged, row); err != nil {
			a.failErr(w, r, err)
			return
		}
		updated, err := a.in.Op(ctx, e.Singular, "atualizar", row["id"], data)
		if err != nil {
			a.failErr(w, r, err)
			return
		}
		urow := updated.(map[string]any)
		if e.MinRole != "" && (movedTo(data, row, e.HierarchyField) || movedTo(data, row, e.InheritVia)) {
			if err := a.keepsHolderAfter(ctx, e, urow); err != nil {
				a.failErr(w, r, err)
				return
			}
		}
		if h := e.Hooks["editar"]; h != nil {
			if _, _, err := a.in.RunHook(ctx, h, a.hookVars(atual, e, urow, body)); err != nil {
				// Compensate: restore the previous values.
				restore := map[string]any{}
				for k := range data {
					restore[k] = row[k]
				}
				a.in.Op(ctx, e.Singular, "atualizar", row["id"], restore)
				a.failErr(w, r, err)
				return
			}
			urow = a.find(ctx, e, fmt.Sprint(row["id"]), nil)
		}
		if err := a.pending(ctx, atual, e, row, urow); err != nil {
			a.failErr(w, r, err)
			return
		}
		if err := a.history(ctx, atual, e, "editar", row, urow); err != nil {
			a.failErr(w, r, err)
			return
		}
		a.emit(ctx, e, "editar", urow, atual)
		a.json(w, 200, serializeFor(ctx, a.in, atual, e, urow, false), nil)
	case "excluir":
		row := a.find(ctx, e, ref, scope)
		if row == nil || !a.in.Can(ctx, atual, e, "ver", row) {
			a.fail(w, 404, a.msg("404", e))
			return
		}
		if !a.in.Can(ctx, atual, e, "excluir", row) {
			deny(row)
			return
		}
		if err := a.frozenFor(ctx, "excluir", e, row, nil); err != nil {
			a.failErr(w, r, err)
			return
		}
		a.emit(ctx, e, "excluir", row, atual) // before removal: the owner must still exist
		if err := a.history(ctx, atual, e, "excluir", row, nil); err != nil {
			a.failErr(w, r, err)
			return
		}
		if err := a.remove(ctx, atual, e, row); err != nil {
			a.failErr(w, r, err)
			return
		}
		a.json(w, http.StatusNoContent, nil, nil)
	case "acao":
		row := a.find(ctx, e, ref, scope)
		if row == nil || !a.in.Can(ctx, atual, e, "ver", row) {
			a.fail(w, 404, a.msg("404", e))
			return
		}
		body, err := readBody(r)
		if err != nil {
			a.failErr(w, r, err)
			return
		}
		if verb == "sair" {
			a.leave(w, r, ctx, atual, e, row)
			return
		}
		if err := a.frozenFor(ctx, "acao", e, row, nil); err != nil {
			a.failErr(w, r, err)
			return
		}
		checkVerb := verb
		if verb == "desaprovar" && len(e.Rules[verb]) == 0 {
			checkVerb = "aprovar"
		}
		if !a.in.Can(ctx, atual, e, checkVerb, row) {
			deny(row)
			return
		}
		if e.Execution != nil && (verb == "cancelar" || verb == "repetir" || verb == "executar") {
			updated, err := a.executionAction(ctx, atual, e, row, verb)
			if err != nil {
				a.failErr(w, r, err)
				return
			}
			a.json(w, 200, serializeFor(ctx, a.in, atual, e, updated, false), nil)
			return
		}
		if verb == "aprovar" || verb == "desaprovar" {
			if len(e.Rules[verb]) > 0 || verb == "aprovar" {
				if !a.in.Can(ctx, atual, e, "aprovar", row) {
					deny(row)
					return
				}
			}
			updated, err := a.approve(ctx, atual, e, row, verb == "aprovar")
			if err != nil {
				a.failErr(w, r, err)
				return
			}
			a.json(w, 200, serializeFor(ctx, a.in, atual, e, updated, false), nil)
			return
		}
		if tr := e.Transitions[verb]; tr != nil && verb == "mesclar" && e.Review != nil && e.Review.Target != "" {
			if err := a.merge(ctx, atual, e, row); err != nil {
				a.failErr(w, r, err)
				return
			}
			row = a.find(ctx, e, fmt.Sprint(row["id"]), nil)
		}
		if tr := e.Transitions[verb]; tr != nil && !(verb == "mesclar" && e.Review != nil && e.Review.Target != "") {
			if err := a.approvalGate(e, verb, row); err != nil {
				a.failErr(w, r, err)
				return
			}
		}
		if tr := e.Transitions[verb]; tr != nil {
			updated, err := a.in.Transition(ctx, atual, e, tr, row)
			if err != nil {
				a.failErr(w, r, err)
				return
			}
			if h := e.Hooks[verb]; h != nil {
				if _, _, err := a.in.RunHook(ctx, h, a.hookVars(atual, e, updated, body)); err != nil {
					a.failErr(w, r, err)
					return
				}
				updated = a.find(ctx, e, fmt.Sprint(row["id"]), nil)
			}
			if err := a.history(ctx, atual, e, verb, row, updated); err != nil {
				a.failErr(w, r, err)
				return
			}
			a.emit(ctx, e, verb, updated, atual)
			a.json(w, 200, serializeFor(ctx, a.in, atual, e, updated, false), nil)
			return
		}
		if verb == "revogar" && e.Hooks[verb] == nil {
			res, err := a.in.Op(ctx, e.Singular, "revogar", row["id"])
			if err != nil {
				a.failErr(w, r, err)
				return
			}
			a.json(w, 200, serializeFor(ctx, a.in, atual, e, res.(map[string]any), false), nil)
			return
		}
		result, resp, err := a.in.RunHook(ctx, e.Hooks[verb], a.hookVars(atual, e, row, body))
		if err != nil {
			a.failErr(w, r, err)
			return
		}
		if resp != nil {
			a.s.writeResponse(w, resp)
			return
		}
		if result == nil {
			result = serializeFor(ctx, a.in, atual, e, a.find(ctx, e, fmt.Sprint(row["id"]), nil), false)
		}
		a.json(w, 200, result, nil)
	}
}

func (a *intentAPI) hookVars(atual map[string]any, e *ast.Entity, row map[string]any, body map[string]any) map[string]any {
	if body == nil {
		body = map[string]any{}
	}
	return map[string]any{"atual": nilIfEmpty(atual), "registro": row, e.Singular: row, "entrada": body}
}

// canCreate checks `criar`; a generic signed-in permission does not let
// anyone create things inside a resource that has members — they must be a
// member of it (or hold a role granted for this action).
func (a *intentAPI) canCreate(ctx *interp.Context, atual map[string]any, e *ast.Entity, data map[string]any) bool {
	if a.in.IsAdmin(atual) {
		return true
	}
	_, membered := a.memberedParentLevel(ctx, atual, e, data)
	openParent := membered && a.visibleParent(ctx, atual, e, data)
	for _, rule := range e.Rules["criar"] {
		ok := false
		switch {
		case rule.Anyone || rule.SignedIn:
			// Inside something that has members, only roles decide — unless
			// the thing is visible to everyone (public/internal) and the rule
			// is not about owning what is created.
			ok = (!membered || (openParent && !rule.Own)) && (rule.Anyone || atual != nil)
		default:
			ok = a.in.RulePasses(ctx, atual, e, rule, data)
		}
		if !ok {
			continue
		}
		if rule.Own && !a.in.Owns(atual, e, data) {
			continue
		}
		return true
	}
	return false
}

func (a *intentAPI) memberedParentLevel(ctx *interp.Context, atual map[string]any, e *ast.Entity, data map[string]any) (int, bool) {
	if len(a.app.Roles) == 0 {
		return 0, false
	}
	if e.Singular == a.app.MemberModel {
		target := a.app.Entities[fmt.Sprint(data["recurso"])]
		if target == nil {
			return 0, false
		}
		row, _ := a.in.Op(ctx, target.Singular, "buscar", data["recurso_id"])
		m, _ := row.(map[string]any)
		return a.in.Level(ctx, atual, target, m), true
	}
	level, membered := 0, false
	for field, target := range e.Parents {
		v := data[field]
		if v == nil {
			continue
		}
		pe := a.app.Entities[target]
		row, _ := a.in.Op(ctx, pe.Singular, "buscar", v)
		m, _ := row.(map[string]any)
		if m == nil {
			continue
		}
		if pe.HasMembers || pe.InheritVia != "" || pe.HierarchyField != "" {
			level, membered = max(level, a.in.Level(ctx, atual, pe, m)), true
		}
	}
	return level, membered
}

func (a *intentAPI) create(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual map[string]any, e *ast.Entity, data, body map[string]any) {
	if h := e.Hooks["antes_criar"]; h != nil {
		// dados is the record about to be created; the hook may adjust it.
		if _, _, err := a.in.RunHook(ctx, h, map[string]any{"atual": nilIfEmpty(atual), "dados": data, "entrada": body}); err != nil {
			a.failErr(w, r, err)
			return
		}
	}
	if err := a.guards(ctx, atual, e, data, nil); err != nil {
		a.failErr(w, r, err)
		return
	}
	res, err := a.in.Op(ctx, e.Singular, "criar", data)
	if err != nil {
		a.failErr(w, r, err)
		return
	}
	row := res.(map[string]any)
	// quem cria X vira Y — unless X inherits members from a parent it was
	// created in (a project inside a group): the members already come from it.
	if e.CreatorRole != "" && atual != nil && (e.InheritVia == "" || row[e.InheritVia] == nil) {
		if _, err := a.in.Op(ctx, a.app.MemberModel, "criar", map[string]any{"recurso": e.Singular, "recurso_id": row["id"], "pessoa_id": atual["id"], "papel": e.CreatorRole}); err != nil {
			a.failErr(w, r, err) // the transaction undoes the creation
			return
		}
	}
	if err := a.createRepository(ctx, e, row); err != nil {
		a.failErr(w, r, err)
		return
	}
	if err := a.initialFile(atual, e, row, body); err != nil {
		a.failErr(w, r, err)
		return
	}
	out := serializeFor(ctx, a.in, atual, e, row, false)
	for _, f := range e.Model.Fields {
		if f.Type == ast.FieldSegredo {
			out[strings.ToLower(f.Name)] = row[strings.ToLower(f.Name)] // shown once
		}
	}
	if h := e.Hooks["criar"]; h != nil {
		if _, _, err := a.in.RunHook(ctx, h, a.hookVars(atual, e, row, body)); err != nil {
			a.failErr(w, r, err) // nothing stays half-created: the transaction is undone
			return
		}
		fresh := serializeFor(ctx, a.in, atual, e, a.find(ctx, e, fmt.Sprint(row["id"]), nil), false)
		for k, v := range fresh {
			out[k] = v
		}
	}
	if e.MinRole != "" {
		if err := a.keepsHolderAfter(ctx, e, a.find(ctx, e, fmt.Sprint(row["id"]), nil)); err != nil {
			a.failErr(w, r, err) // the transaction undoes the creation
			return
		}
	}
	if err := a.pending(ctx, atual, e, nil, row); err != nil {
		a.failErr(w, r, err)
		return
	}
	if err := a.history(ctx, atual, e, "criar", nil, row); err != nil {
		a.failErr(w, r, err)
		return
	}
	a.emit(ctx, e, "criar", row, atual)
	a.json(w, 201, out, nil)
}

// remove runs `quando excluir` first (it may refuse), then deletes the
// record and everything that belongs to it.
// The minimum role (`todo grupo precisa ter pelo menos um owner`): a record
// always has someone holding the role or a higher one — directly, or through
// the parent it inherits members from. The check runs inside the request's
// transaction, after locking the record.

// hasHolder: record (of e) has someone with level ≥ min — directly or through
// the parents it inherits members from — ignoring the membership skip
// (removed or demoted) and the person gone (deleted).
func (a *intentAPI) hasHolder(ctx *interp.Context, e *ast.Entity, record map[string]any, min int, skip, gone any, depth int) bool {
	if record == nil || depth > 16 {
		return false
	}
	var enough []any
	for _, role := range a.app.Roles {
		if role.Level >= min {
			enough = append(enough, role.Name)
		}
	}
	f := map[string]any{"recurso": e.Singular, "recurso_id": record["id"], "papel__em": enough}
	if skip != nil {
		f["id__diferente"] = skip
	}
	if gone != nil {
		f["pessoa_id__diferente"] = gone
	}
	if n, err := a.in.Op(ctx, a.app.MemberModel, "contar", f); err == nil && asNumber(n) > 0 {
		return true
	}
	for _, field := range []string{e.HierarchyField, e.InheritVia} {
		if field == "" || record[field] == nil {
			continue
		}
		pe := a.app.Entities[e.Parents[field]]
		res, _ := a.in.Op(ctx, pe.Singular, "buscar", record[field])
		if parent, _ := res.(map[string]any); parent != nil && pe.HasMembers && a.hasHolder(ctx, pe, parent, min, skip, gone, depth+1) {
			return true
		}
	}
	return false
}

func (a *intentAPI) minRoleError(e *ast.Entity) error {
	msg := map[string]string{
		"pt": fmt.Sprintf("%s precisa ter pelo menos um %s", e.Label, e.MinRole),
		"en": fmt.Sprintf("The last %s cannot leave or be removed", e.MinRole),
	}[a.app.Messages]
	return &interp.RuntimeError{Status: 400, Message: msg}
}

// keepsMinRole: removing (newRole "") or demoting a membership keeps a holder.
func (a *intentAPI) keepsMinRole(ctx *interp.Context, member map[string]any, newRole string) error {
	target := a.app.Entities[toStr(member["recurso"])]
	if target == nil || target.MinRole == "" {
		return nil
	}
	min := a.app.Level(target.MinRole)
	if a.app.Level(toStr(member["papel"])) < min || (newRole != "" && a.app.Level(newRole) >= min) {
		return nil
	}
	a.dbOf(ctx).Travar(target.Singular, member["recurso_id"])
	res, _ := a.in.Op(ctx, target.Singular, "buscar", member["recurso_id"])
	rec, _ := res.(map[string]any)
	if rec == nil || a.hasHolder(ctx, target, rec, min, member["id"], nil, 0) {
		return nil
	}
	return a.minRoleError(target)
}

// movedTo: the edit changes the parent in field.
func movedTo(data, row map[string]any, field string) bool {
	if field == "" {
		return false
	}
	v, given := data[field]
	return given && toStr(v) != toStr(row[field])
}

// keepsHolderAfter: a record created or moved to another parent must end with a holder.
func (a *intentAPI) keepsHolderAfter(ctx *interp.Context, e *ast.Entity, record map[string]any) error {
	if e.MinRole == "" || record == nil {
		return nil
	}
	if a.hasHolder(ctx, e, record, a.app.Level(e.MinRole), nil, nil, 0) {
		return nil
	}
	return a.minRoleError(e)
}

// personLeaves: deleting a person keeps every record they hold a holder.
func (a *intentAPI) personLeaves(ctx *interp.Context, person map[string]any) error {
	if a.app.MemberModel == "" {
		return nil
	}
	res, err := a.in.Op(ctx, a.app.MemberModel, "filtrar", map[string]any{"pessoa_id": person["id"]}, map[string]any{"limite": 1000})
	if err != nil {
		return err
	}
	for _, it := range res.([]any) {
		m := it.(map[string]any)
		target := a.app.Entities[toStr(m["recurso"])]
		if target == nil || target.MinRole == "" || a.app.Level(toStr(m["papel"])) < a.app.Level(target.MinRole) {
			continue
		}
		a.dbOf(ctx).Travar(target.Singular, m["recurso_id"])
		rr, _ := a.in.Op(ctx, target.Singular, "buscar", m["recurso_id"])
		rec, _ := rr.(map[string]any)
		if rec != nil && !a.hasHolder(ctx, target, rec, a.app.Level(target.MinRole), nil, person["id"], 0) {
			return a.minRoleError(target)
		}
	}
	// The person's memberships go with the person (in the same transaction).
	for {
		rows, err := a.in.Op(ctx, a.app.MemberModel, "filtrar", map[string]any{"pessoa_id": person["id"]}, map[string]any{"limite": 500, "ordenar": "id"})
		if err != nil {
			return err
		}
		list := rows.([]any)
		if len(list) == 0 {
			return nil
		}
		for _, it := range list {
			if _, err := a.in.Op(ctx, a.app.MemberModel, "deletar", it.(map[string]any)["id"]); err != nil {
				return err
			}
		}
	}
}

// personReferences settles what refers to a person being deleted: records
// that belong to the person (a required reference) are deleted with their
// own contents; records that only name the person (an optional reference:
// author, whoever closed it) stay, without the name.
func (a *intentAPI) personReferences(ctx *interp.Context, person map[string]any) error {
	for _, n := range a.app.Order {
		c := a.app.Entities[n]
		if c == nil || c.Singular == a.app.MemberModel || c.Singular == a.app.LoginEntity {
			continue
		}
		for _, f := range c.Model.Fields {
			if f.Reference != a.app.LoginEntity {
				continue
			}
			field := strings.ToLower(f.Name)
			for {
				rows, err := a.in.Op(ctx, c.Singular, "filtrar", map[string]any{field: person["id"]}, map[string]any{"limite": 500, "ordenar": "id"})
				if err != nil {
					return err
				}
				list := rows.([]any)
				if len(list) == 0 {
					break
				}
				for _, it := range list {
					rec := it.(map[string]any)
					if f.Required {
						if err := a.cascade(ctx, c, rec, 1); err != nil {
							return err
						}
					} else if _, err := a.in.Op(ctx, c.Singular, "atualizar", rec["id"], map[string]any{field: nil}); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (a *intentAPI) remove(ctx *interp.Context, atual map[string]any, e *ast.Entity, row map[string]any) error {
	if e.Singular == a.app.MemberModel {
		if err := a.keepsMinRole(ctx, row, ""); err != nil {
			return err
		}
	}
	if e.Singular == a.app.LoginEntity {
		if err := a.personLeaves(ctx, row); err != nil {
			return err
		}
		if a.in.Decorate != nil { // the person's reading marks go with them (GEP 0022)
			if _, err := a.dbOf(ctx).Executar(fmt.Sprintf(`DELETE FROM %s WHERE pessoa_id = %s`, readTable, a.s.ph(1)), row["id"]); err != nil {
				return err
			}
		}
		if err := a.personReferences(ctx, row); err != nil {
			return err
		}
	}
	if h := e.Hooks["antes_excluir"]; h != nil {
		if _, _, err := a.in.RunHook(ctx, h, a.hookVars(atual, e, row, nil)); err != nil {
			return err
		}
	}
	if err := a.cascade(ctx, e, row, 0); err != nil {
		return err
	}
	if h := e.Hooks["excluir"]; h != nil {
		if _, _, err := a.in.RunHook(ctx, h, a.hookVars(atual, e, row, nil)); err != nil {
			return err
		}
	}
	return nil
}

func (a *intentAPI) cascade(ctx *interp.Context, e *ast.Entity, row map[string]any, depth int) error {
	if depth > 16 {
		return fmt.Errorf("exclusão aninhada demais")
	}
	for _, c := range a.childrenOf(e) {
		sc := a.parentScope(e, row, c)
		if len(sc) == 0 {
			continue
		}
		// Each pass deletes what it read, so the first batch is always the next one.
		for {
			kids, err := a.in.Op(ctx, c.Singular, "filtrar", sc, map[string]any{"limite": 500, "ordenar": "id"})
			if err != nil {
				return err
			}
			if len(kids.([]any)) == 0 {
				break
			}
			for _, k := range kids.([]any) {
				if err := a.cascade(ctx, c, k.(map[string]any), depth+1); err != nil {
					return err
				}
			}
		}
	}
	if err := a.dropPending(ctx, e, row["id"], nil); err != nil {
		return err
	}
	if _, err := a.in.Op(ctx, e.Singular, "deletar", row["id"]); err != nil {
		return err
	}
	afterCommit(ctx, func() { a.removeRepository(e, row); a.removeFiles(e, row); a.removeTextImages(e, row["id"]) })
	return nil
}

// leave removes atual's own membership (quando excluir membro may refuse).
func (a *intentAPI) leave(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual map[string]any, e *ast.Entity, row map[string]any) {
	if atual == nil {
		a.fail(w, 401, a.msg("401", e))
		return
	}
	me := a.app.Entities[a.app.MemberModel]
	res, err := a.in.Op(ctx, me.Singular, "encontrar", map[string]any{"recurso": e.Singular, "recurso_id": row["id"], "pessoa_id": atual["id"]})
	m, _ := res.(map[string]any)
	if err != nil || m == nil {
		a.fail(w, 404, a.msg("404", me))
		return
	}
	if err := a.remove(ctx, atual, me, m); err != nil {
		a.failErr(w, r, err)
		return
	}
	a.json(w, http.StatusNoContent, nil, nil)
}

func (a *intentAPI) list(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual map[string]any, e *ast.Entity, scope map[string]any, deny func(map[string]any)) {
	q := r.URL.Query()
	page := atoiDefault(first(q.Get("page"), q.Get("pagina")), 1)
	per := atoiDefault(first(q.Get("per_page"), q.Get("por_pagina")), 20)
	if page < 1 {
		page = 1
	}
	if per < 1 || per > 100 {
		per = 20
	}
	filters := map[string]any{}
	for k, v := range scope {
		filters[k] = v
	}
	states := a.stateFields()
	for _, f := range e.Filters {
		v := q.Get(f)
		if a.extern {
			v = q.Get(a.ext(f))
			if states[f] {
				v = a.inwardState(e, v)
			}
		}
		if v == "" {
			continue
		}
		if fd := fieldOf(e, f); fd != nil && fd.Type == ast.FieldLista {
			if fd.ByName != "" {
				ids, _ := a.nameFilter(ctx, e, fd, v, filters)
				filters[f+"__contem_algum"] = ids // no match: an empty list matches nothing
				continue
			}
			filters[f+"__contem"] = `"` + strings.ReplaceAll(v, `"`, "") + `"`
			continue
		}
		filters[f] = v
	}
	opts := map[string]any{"ordenar": "-id"}
	if s := first(q.Get("search"), q.Get("q"), q.Get("pesquisa")); s != "" && len(e.Search) > 0 {
		opts["busca"] = s
		fields := make([]any, len(e.Search))
		for i, f := range e.Search {
			fields[i] = f
		}
		opts["campos_busca"] = fields
	}
	if atual == nil && e.Visibility == "" && !anyoneMay(e) && !interp.InheritsView(a.app, e) && !a.visibleThroughParents(e) {
		a.fail(w, 401, a.msg("401", e))
		return
	}
	var items []any
	total := 0
	nestedInherit := interp.InheritsView(a.app, e) && len(scope) > 0 // parent already checked
	if !interp.RecordDependent(e) && (nestedInherit || !interp.InheritsView(a.app, e)) {
		if !nestedInherit && !a.in.Can(ctx, atual, e, "ver", nil) {
			deny(nil)
			return
		}
		opts["limite"], opts["pagina"] = per, page
		res, err := a.in.Op(ctx, e.Singular, "paginar", filters, opts)
		if err != nil {
			a.failErr(w, r, err)
			return
		}
		m := res.(map[string]any)
		for _, it := range m["itens"].([]any) {
			items = append(items, serializeFor(ctx, a.in, atual, e, it.(map[string]any), false))
		}
		total = int(m["total"].(float64))
	} else {
		// Visibility depends on each record: read in batches, keep what this
		// person may see, count all of it and keep the requested page.
		a.narrowVisible(ctx, atual, e, filters)
		interp.BeginReadCache(ctx)
		defer interp.EndReadCache(ctx)
		var groups []map[string]any
		if g, ok := a.visibilityPrefilter(ctx, atual, e); ok {
			groups = g
		}
		start := (page - 1) * per
		for batch := 1; ; batch++ {
			opts["limite"], opts["pagina"] = 500, batch
			if groups != nil {
				opts["ou"] = groups
			}
			res, err := a.in.Op(ctx, e.Singular, "filtrar", filters, opts)
			if err != nil {
				a.failErr(w, r, err)
				return
			}
			rows := res.([]any)
			for _, it := range rows {
				row := it.(map[string]any)
				if !a.in.Can(ctx, atual, e, "ver", row) {
					continue
				}
				if total >= start && total < start+per {
					items = append(items, serializeFor(ctx, a.in, atual, e, row, false))
				}
				total++
			}
			if len(rows) < 500 {
				break
			}
		}
	}
	if items == nil {
		items = []any{}
	}
	pages := (total + per - 1) / per
	a.json(w, 200, items, map[string]string{"X-Total": strconv.Itoa(total), "X-Page": strconv.Itoa(page), "X-Per-Page": strconv.Itoa(per), "X-Total-Pages": strconv.Itoa(pages)})
}

// narrowVisible adds SQL filters that follow exactly from the rules, so
// fewer records are read: a person who is not signed in only ever sees the
// public records of data that has members.
func (a *intentAPI) narrowVisible(ctx *interp.Context, atual map[string]any, e *ast.Entity, filters map[string]any) {
	if atual == nil && e.Visibility != "" && (e.HasMembers || e.InheritVia != "") {
		filters[e.Visibility] = "public"
	}
}

func first(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func atoiDefault(s string, d int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return d
}

var _ = parser.Singular

func anyoneMay(e *ast.Entity) bool {
	for _, v := range []string{"ver", "editar", "excluir"} {
		for _, r := range e.Rules[v] {
			if r.Anyone {
				return true
			}
		}
	}
	return false
}

// hiddenParent returns the parent entity referenced by data that atual
// cannot see (its existence must not be revealed).
func (a *intentAPI) hiddenParent(ctx *interp.Context, atual map[string]any, e *ast.Entity, data map[string]any) *ast.Entity {
	for field, target := range e.Parents {
		if data[field] == nil {
			continue
		}
		pe := a.app.Entities[target]
		res, _ := a.in.Op(ctx, pe.Singular, "buscar", data[field])
		row, _ := res.(map[string]any)
		if row == nil || !a.in.Can(ctx, atual, pe, "ver", row) {
			return pe
		}
	}
	return nil
}

// visibleParent: the membered parent referenced by data is visible through
// its visibility (public/internal), not through membership.
func (a *intentAPI) visibleParent(ctx *interp.Context, atual map[string]any, e *ast.Entity, data map[string]any) bool {
	seen := false
	for field, target := range e.Parents {
		if data[field] == nil || target == a.app.LoginEntity {
			continue
		}
		pe := a.app.Entities[target]
		res, _ := a.in.Op(ctx, pe.Singular, "buscar", data[field])
		row, ok := res.(map[string]any)
		if !ok || !a.in.VisibleByVisibility(ctx, atual, pe, row) {
			return false // every filled parent must be open
		}
		seen = true
	}
	return seen
}

func fieldOf(e *ast.Entity, name string) *ast.Field {
	for _, f := range e.Model.Fields {
		if strings.EqualFold(f.Name, name) {
			return f
		}
	}
	return nil
}

// visibleThroughParents: records may be public because a parent is.
func (a *intentAPI) visibleThroughParents(e *ast.Entity) bool {
	for _, t := range e.Parents {
		if t != a.app.LoginEntity {
			pe := a.app.Entities[t]
			if pe.Visibility != "" || a.visibleThroughParents(pe) {
				return true
			}
		}
	}
	return false
}

// ext translates a domain name to the integration vocabulary.
func (a *intentAPI) ext(name string) string {
	if v, ok := a.app.Vocabulary[name]; ok {
		return v
	}
	return name
}

// stateFields: fields whose values are words of the domain (states and
// enumerated values) — the vocabulary translates their values too.
func (a *intentAPI) stateFields() map[string]bool {
	out := map[string]bool{}
	for _, e := range a.app.Entities {
		if e.StateField != "" {
			out[e.StateField] = true
		}
		for _, f := range e.Model.Fields {
			if f.Type == ast.FieldEnum && f.System {
				out[strings.ToLower(f.Name)] = true
			}
		}
	}
	return out
}

// outward renames keys (and state values) to the integration vocabulary.
func (a *intentAPI) outward(v any) any {
	states := a.stateFields()
	switch x := v.(type) {
	case []any:
		out := make([]any, len(x))
		for i, it := range x {
			out[i] = a.outward(it)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			if s, ok := val.(string); ok && states[k] {
				val = a.ext(s)
			}
			// Roles renamed in the vocabulary travel as their levels (30 = developer).
			if s, ok := val.(string); ok && k == "papel" && a.ext("papel") != "papel" && a.app.Level(s) > 0 {
				val = a.app.Level(s)
			}
			switch val.(type) {
			case map[string]any, []any:
				val = a.outward(val) // nested objects (webhook payloads)
			}
			out[a.ext(k)] = val
		}
		return out
	}
	return v
}

// inward translates incoming names back to the domain. Several domain
// names may share one external name (aberta/aberto → opened); the choice is
// deterministic (sorted) and inwardState prefers the states of the data.
func (a *intentAPI) inward(name string) string {
	if !a.extern {
		return name
	}
	for _, k := range a.candidates(name) {
		return k
	}
	return name
}

func (a *intentAPI) candidates(external string) []string {
	var out []string
	for k, v := range a.app.Vocabulary {
		if v == external {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func statesOf(e *ast.Entity) map[string]bool {
	out := map[string]bool{}
	if e == nil {
		return out
	}
	out[e.Initial] = true
	for _, t := range e.Transitions {
		out[t.Target] = true
	}
	if e.Execution != nil {
		for _, st := range []string{stCreated, stPending, stRunning, stSuccess, stFailed, stCanceled, stSkipped, stManual} {
			out[st] = true
		}
	}
	return out
}

// inwardState translates an external state value for entity e.
func (a *intentAPI) inwardState(e *ast.Entity, v string) string {
	if !a.extern {
		return v
	}
	own := statesOf(e)
	cands := a.candidates(v)
	for _, k := range cands {
		if own[k] {
			return k
		}
	}
	if len(cands) > 0 {
		return cands[0]
	}
	return v
}

func (a *intentAPI) inwardBody(e *ast.Entity, body map[string]any) map[string]any {
	if !a.extern || len(a.app.Vocabulary) == 0 {
		return body
	}
	states := a.stateFields()
	out := make(map[string]any, len(body))
	for k, v := range body {
		key := a.inward(k)
		if s, ok := v.(string); ok && states[key] {
			v = a.inwardState(e, s)
		}
		if n, ok := v.(float64); ok && key == "papel" {
			for _, role := range a.app.Roles {
				if float64(role.Level) == n {
					v = role.Name
				}
			}
		}
		out[key] = v
	}
	return out
}

// guards applies built-in safety rules before a record is saved:
//   - visibility ceilings (never more visible than the parent);
//   - nobody grants a role above their own (memberships).
func (a *intentAPI) guards(ctx *interp.Context, atual map[string]any, e *ast.Entity, data, before map[string]any) error {
	if err := a.checkBranches(ctx, e, data); err != nil {
		return err
	}
	if l := a.app.Login; l != nil && e.Singular == l.TokenEntity && len(l.Scopes) > 0 && data["escopos"] != nil {
		for _, sc := range strings.Split(toStr(data["escopos"]), ",") {
			if _, ok := l.Scopes[strings.TrimSpace(sc)]; !ok {
				msg := map[string]string{"pt": "contém escopo desconhecido " + strings.TrimSpace(sc), "en": "can only contain available scopes"}[a.app.Messages]
				return &interp.RuntimeError{Status: 400, Message: msg, Payload: map[string]any{"escopos": []any{msg}}}
			}
		}
	}
	order := map[string]int{"private": 0, "internal": 1, "public": 2}
	for _, field := range e.CeilingFields {
		if data[field] == nil {
			continue
		}
		pe := a.app.Entities[e.Parents[field]]
		res, _ := a.in.Op(ctx, pe.Singular, "buscar", data[field])
		parent, _ := res.(map[string]any)
		if parent == nil {
			continue
		}
		mine := fmt.Sprint(data[e.Visibility])
		if mine == "<nil>" || mine == "" {
			mine = "private"
		}
		if order[mine] > order[fmt.Sprint(parent[pe.Visibility])] {
			msg := fmt.Sprintf("A visibilidade não pode ser maior que a de %s", pe.Label)
			if a.app.Messages == "en" {
				msg = fmt.Sprintf("Visibility level %s is not allowed since the %s has a more restrictive visibility", mine, strings.ToLower(pe.Label))
			}
			return &interp.RuntimeError{Status: 400, Message: msg, Payload: map[string]any{e.Visibility: []any{msg}}}
		}
	}
	if e.Singular == a.app.MemberModel && before != nil && data["papel"] != nil {
		if err := a.keepsMinRole(ctx, before, toStr(data["papel"])); err != nil {
			return err
		}
	}
	if e.Singular == a.app.MemberModel && !a.in.IsAdmin(atual) {
		target := a.app.Entities[fmt.Sprint(data["recurso"])]
		if target != nil {
			res, _ := a.in.Op(ctx, target.Singular, "buscar", data["recurso_id"])
			rec, _ := res.(map[string]any)
			mine := a.in.Level(ctx, atual, target, rec)
			want := a.app.Level(fmt.Sprint(data["papel"]))
			had := 0
			if before != nil {
				had = a.app.Level(fmt.Sprint(before["papel"]))
			}
			if want > mine || had > mine {
				return &interp.RuntimeError{Status: 403, Message: a.msg("403", e)}
			}
		}
	}
	return nil
}

// manualRun: creating a run by hand (executar pipelines) runs the file of
// the given branch at its current commit.
func (a *intentAPI) manualRun(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual map[string]any, e *ast.Entity, data map[string]any) {
	x := e.Execution
	owner := a.app.Entities[x.Owner]
	res, _ := a.in.Op(ctx, owner.Singular, "buscar", data[x.OwnerField])
	ownerRow, _ := res.(map[string]any)
	if ownerRow == nil || !a.in.Can(ctx, atual, owner, "ver", ownerRow) {
		a.fail(w, 404, a.msg("404", owner))
		return
	}
	allowed := a.in.IsAdmin(atual)
	for _, rule := range append(append([]*ast.AccessRule{}, e.Rules["criar"]...), e.Rules["executar"]...) {
		allowed = allowed || a.in.RulePasses(ctx, atual, e, rule, data)
	}
	if !allowed {
		if atual == nil {
			a.fail(w, 401, a.msg("401", e))
		} else {
			a.fail(w, 403, a.msg("403", e))
		}
		return
	}
	branch := toStr(data["branch"])
	if branch == "" {
		branch = defaultBranch(ownerRow)
	}
	sha, err := a.s.Git.Resolve(toStr(ownerRow["repositorio"]), "refs/heads/"+branch)
	if err != nil {
		a.fail(w, 400, map[string]any{"branch": []any{map[string]string{"pt": "não existe", "en": "does not exist"}[a.app.Messages]}})
		return
	}
	row, found, err := a.createRun(ctx, atual, e, ownerRow, branch, sha)
	if err != nil {
		a.failErr(w, r, err)
		return
	}
	if !found {
		msg := fmt.Sprintf("O arquivo %s não existe em %s", x.File, branch)
		if a.app.Messages == "en" {
			msg = "Missing CI config file"
		}
		a.fail(w, 400, msg)
		return
	}
	a.json(w, 201, serializeFor(ctx, a.in, atual, e, row, false), nil)
}

// mountSearch serves `tenha busca geral em …`: one place to search several
// kinds of data. The kind (tipo) picks the collection and the request is
// answered by that collection's own listing — visibility, search fields,
// filters and pages are exactly the same.
func (a *intentAPI) mountSearch(mux *routeMux) {
	if len(a.app.GlobalSearch) == 0 {
		return
	}
	for _, extern := range []bool{false, true} {
		p := *a
		p.extern = extern
		base := "/_ge/api/busca"
		if extern {
			if a.app.Integration == "" {
				continue
			}
			base = a.app.Integration + "/" + p.ext("busca")
		}
		pp := &p
		mux.HandleFunc("GET "+base, func(w http.ResponseWriter, r *http.Request) {
			kind := r.URL.Query().Get(pp.ext("tipo_busca"))
			var names []string
			for _, n := range pp.app.GlobalSearch {
				e := pp.app.Entities[n]
				name := e.Plural
				if pp.extern && e.Integrate != "" {
					name = e.Integrate
				}
				names = append(names, name)
				if kind == name {
					pp.serve(w, r, []*ast.Entity{e}, "listar", "")
					return
				}
			}
			pp.fail(w, 400, map[string]string{
				"pt": "escolha o tipo de busca: " + strings.Join(names, ", "),
				"en": pp.ext("tipo_busca") + " does not have a valid value (" + strings.Join(names, ", ") + ")",
			}[pp.app.Messages])
		})
	}
}
