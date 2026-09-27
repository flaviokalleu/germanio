package servidor

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
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
}

func (s *Servidor) registerIntent(mux *http.ServeMux) error {
	app := s.Program.App
	if app == nil {
		return nil
	}
	a := &intentAPI{s: s, app: app, in: s.Interpreter}
	for _, name := range app.Order {
		e := app.Entities[name]
		if len(e.Rules) == 0 && e.Integrate == "" {
			continue
		}
		a.mount(mux, "/_ge/api/"+e.Plural, e, false)
		if e.Integrate != "" {
			a.mount(mux, app.Integration+"/"+e.Integrate, e, true)
		}
	}
	a.mountIdentity(mux)
	return nil
}

func (a *intentAPI) mount(mux *http.ServeMux, base string, e *ast.Entity, integration bool) {
	a.mountLevel(mux, base, []*ast.Entity{e}, integration)
}

// mountLevel mounts the collection at base for the last entity of chain
// (earlier entities are its ancestors, each addressed by {rN}).
func (a *intentAPI) mountLevel(mux *http.ServeMux, base string, chain []*ast.Entity, integration bool) {
	e := chain[len(chain)-1]
	item := base + "/{r" + strconv.Itoa(len(chain)-1) + "}"
	h := func(op, verb string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) { a.serve(w, r, chain, op, verb) }
	}
	mux.HandleFunc("GET "+base, h("listar", ""))
	mux.HandleFunc("POST "+base, h("criar", ""))
	mux.HandleFunc("GET "+item, h("ver", ""))
	mux.HandleFunc("PUT "+item, h("editar", ""))
	mux.HandleFunc("PATCH "+item, h("editar", ""))
	mux.HandleFunc("DELETE "+item, h("excluir", ""))
	for verb := range e.Rules {
		if !isStandard(verb) && verb != "sair" {
			mux.HandleFunc("POST "+item+"/"+verb, h("acao", verb))
		}
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
		return label + " não encontrado"
	case "403":
		return "Você não tem permissão para fazer isso"
	case "401":
		return "Entre para continuar"
	}
	return kind
}

func (a *intentAPI) fail(w http.ResponseWriter, status int, msg any) {
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
	return serializeFor(nil, nil, e, row, true)
}

func serializeFor(in *interp.Interpreter, atual map[string]any, e *ast.Entity, row map[string]any, trusted bool) map[string]any {
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
	for k, v := range row {
		switch {
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
	out := map[string]any{}
	for _, f := range e.Model.Fields {
		key := strings.ToLower(f.Name)
		v, ok := body[f.Name]
		if !ok {
			v, ok = body[key]
		}
		if !ok || f.Hidden || f.Type == ast.FieldSegredo {
			continue
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
			if _, given := out[owner]; !given || !a.in.IsAdmin(atual) {
				out[owner] = atual["id"]
			}
		}
	}
	return out
}

// ---------- operations ----------

func (a *intentAPI) serve(w http.ResponseWriter, r *http.Request, chain []*ast.Entity, op string, verb string) {
	root := chain[0]
	ctx := &interp.Context{Request: r, Writer: w}
	atual, err := a.s.identify(ctx, r)
	if err != nil {
		a.fail(w, 401, a.msg("401", root))
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
	case "listar":
		a.list(w, r, ctx, atual, e, scope, deny)
	case "ver":
		row := a.find(ctx, e, ref, scope)
		if row == nil || !a.in.Can(ctx, atual, e, "ver", row) {
			a.fail(w, 404, a.msg("404", e))
			return
		}
		a.json(w, 200, serializeFor(a.in, atual, e, row, false), nil)
	case "criar":
		body, err := readBody(r)
		if err != nil {
			a.failErr(w, r, err)
			return
		}
		data := a.writable(atual, e, body, scope)
		if !a.canCreate(ctx, atual, e, data) {
			if pe := a.hiddenParent(ctx, atual, e, data); pe != nil {
				a.fail(w, 404, a.msg("404", pe))
				return
			}
			deny(nil)
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
		data := a.writable(atual, e, body, nil)
		for k := range scope {
			delete(data, k)
		}
		if h := e.Hooks["antes_editar"]; h != nil {
			if _, _, err := a.in.RunHook(ctx, h, map[string]any{"atual": nilIfEmpty(atual), "registro": row, e.Singular: row, "dados": data, "entrada": body}); err != nil {
				a.failErr(w, r, err)
				return
			}
		}
		updated, err := a.in.Op(ctx, e.Singular, "atualizar", row["id"], data)
		if err != nil {
			a.failErr(w, r, err)
			return
		}
		urow := updated.(map[string]any)
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
		a.json(w, 200, serializeFor(a.in, atual, e, urow, false), nil)
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
		if !a.in.Can(ctx, atual, e, verb, row) {
			deny(row)
			return
		}
		if verb == "revogar" && e.Hooks[verb] == nil {
			res, err := a.in.Op(ctx, e.Singular, "revogar", row["id"])
			if err != nil {
				a.failErr(w, r, err)
				return
			}
			a.json(w, 200, serializeFor(a.in, atual, e, res.(map[string]any), false), nil)
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
			result = serializeFor(a.in, atual, e, a.find(ctx, e, fmt.Sprint(row["id"]), nil), false)
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
			return a.in.Level(ctx, atual, pe, m), true
		}
	}
	return 0, false
}

func (a *intentAPI) create(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual map[string]any, e *ast.Entity, data, body map[string]any) {
	if h := e.Hooks["antes_criar"]; h != nil {
		// dados is the record about to be created; the hook may adjust it.
		if _, _, err := a.in.RunHook(ctx, h, map[string]any{"atual": nilIfEmpty(atual), "dados": data, "entrada": body}); err != nil {
			a.failErr(w, r, err)
			return
		}
	}
	res, err := a.in.Op(ctx, e.Singular, "criar", data)
	if err != nil {
		a.failErr(w, r, err)
		return
	}
	row := res.(map[string]any)
	if err := a.createRepository(ctx, e, row); err != nil {
		a.in.Op(ctx, e.Singular, "deletar", row["id"])
		a.failErr(w, r, err)
		return
	}
	out := serializeFor(a.in, atual, e, row, false)
	for _, f := range e.Model.Fields {
		if f.Type == ast.FieldSegredo {
			out[strings.ToLower(f.Name)] = row[strings.ToLower(f.Name)] // shown once
		}
	}
	if h := e.Hooks["criar"]; h != nil {
		if _, _, err := a.in.RunHook(ctx, h, a.hookVars(atual, e, row, body)); err != nil {
			a.removeRepository(e, row)
			a.in.Op(ctx, e.Singular, "deletar", row["id"]) // nothing stays half-created
			a.failErr(w, r, err)
			return
		}
		fresh := serializeFor(a.in, atual, e, a.find(ctx, e, fmt.Sprint(row["id"]), nil), false)
		for k, v := range fresh {
			out[k] = v
		}
	}
	a.json(w, 201, out, nil)
}

// remove runs `quando excluir` first (it may refuse), then deletes the
// record and everything that belongs to it.
func (a *intentAPI) remove(ctx *interp.Context, atual map[string]any, e *ast.Entity, row map[string]any) error {
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
		kids, err := a.in.Op(ctx, c.Singular, "filtrar", sc, map[string]any{"limite": 1000})
		if err != nil {
			return err
		}
		for _, k := range kids.([]any) {
			if err := a.cascade(ctx, c, k.(map[string]any), depth+1); err != nil {
				return err
			}
		}
	}
	if _, err := a.in.Op(ctx, e.Singular, "deletar", row["id"]); err != nil {
		return err
	}
	a.removeRepository(e, row)
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
	for _, f := range e.Filters {
		v := q.Get(f)
		if v == "" {
			continue
		}
		if fd := fieldOf(e, f); fd != nil && fd.Type == ast.FieldLista {
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
			items = append(items, serializeFor(a.in, atual, e, it.(map[string]any), false))
		}
		total = int(m["total"].(float64))
	} else {
		// Visibility depends on each record: filter, then paginate.
		opts["limite"] = 1000
		res, err := a.in.Op(ctx, e.Singular, "filtrar", filters, opts)
		if err != nil {
			a.failErr(w, r, err)
			return
		}
		var visible []any
		for _, it := range res.([]any) {
			row := it.(map[string]any)
			if a.in.Can(ctx, atual, e, "ver", row) {
				visible = append(visible, serializeFor(a.in, atual, e, row, false))
			}
		}
		total = len(visible)
		start := min((page-1)*per, total)
		items = visible[start:min(start+per, total)]
	}
	if items == nil {
		items = []any{}
	}
	pages := (total + per - 1) / per
	a.json(w, 200, items, map[string]string{"X-Total": strconv.Itoa(total), "X-Page": strconv.Itoa(page), "X-Per-Page": strconv.Itoa(per), "X-Total-Pages": strconv.Itoa(pages)})
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
	for field, target := range e.Parents {
		if data[field] == nil || target == a.app.LoginEntity {
			continue
		}
		pe := a.app.Entities[target]
		res, _ := a.in.Op(ctx, pe.Singular, "buscar", data[field])
		if row, ok := res.(map[string]any); ok && a.in.VisibleByVisibility(atual, pe, row) {
			return true
		}
	}
	return false
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
