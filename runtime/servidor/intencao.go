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
	h := func(op string, child *ast.Entity, verb string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) { a.serve(w, r, e, op, child, verb) }
	}
	mux.HandleFunc("GET "+base, h("listar", nil, ""))
	mux.HandleFunc("POST "+base, h("criar", nil, ""))
	mux.HandleFunc("GET "+base+"/{ref}", h("ver", nil, ""))
	mux.HandleFunc("PUT "+base+"/{ref}", h("editar", nil, ""))
	mux.HandleFunc("PATCH "+base+"/{ref}", h("editar", nil, ""))
	mux.HandleFunc("DELETE "+base+"/{ref}", h("excluir", nil, ""))
	for verb := range e.Rules {
		if !isStandard(verb) {
			mux.HandleFunc("POST "+base+"/{ref}/"+verb, h("acao", nil, verb))
		}
	}
	if (e.HasMembers || e.InheritVia != "") && a.app.MemberModel != "" {
		mux.HandleFunc("POST "+base+"/{ref}/sair", h("acao", nil, "sair"))
	}
	for _, c := range a.childrenOf(e) {
		name := c.Plural
		if integration && c.Integrate != "" {
			name = c.Integrate
		}
		cb := base + "/{ref}/" + name
		mux.HandleFunc("GET "+cb, h("listar", c, ""))
		mux.HandleFunc("POST "+cb, h("criar", c, ""))
		mux.HandleFunc("GET "+cb+"/{cref}", h("ver", c, ""))
		mux.HandleFunc("PUT "+cb+"/{cref}", h("editar", c, ""))
		mux.HandleFunc("PATCH "+cb+"/{cref}", h("editar", c, ""))
		mux.HandleFunc("DELETE "+cb+"/{cref}", h("excluir", c, ""))
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

// serialize produces the public representation of a record.
func serialize(e *ast.Entity, row map[string]any) map[string]any {
	if row == nil {
		return nil
	}
	out := make(map[string]any, len(row))
	hidden := map[string]bool{}
	for _, f := range e.Model.Fields {
		if f.Hidden || f.IsSecret() {
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
	if owner := a.app.LoginEntity + "_id"; atual != nil && e.Singular != a.app.LoginEntity && e.Singular != a.app.MemberModel && e.Parents[owner] != "" {
		if _, given := out[owner]; !given || !a.in.IsAdmin(atual) {
			out[owner] = atual["id"]
		}
	}
	return out
}

// ---------- operations ----------

func (a *intentAPI) serve(w http.ResponseWriter, r *http.Request, root *ast.Entity, op string, child *ast.Entity, verb string) {
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
	e := root
	scope := map[string]any{}
	var parentRow map[string]any
	ref := r.PathValue("ref")
	if child != nil {
		parentRow = a.find(ctx, root, ref, nil)
		if parentRow == nil || !a.in.Can(ctx, atual, root, "ver", parentRow) {
			a.fail(w, 404, a.msg("404", root))
			return
		}
		scope = a.parentScope(root, parentRow, child)
		e = child
		ref = r.PathValue("cref")
	}
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
		a.json(w, 200, serialize(e, row), nil)
	case "criar":
		body, err := readBody(r)
		if err != nil {
			a.failErr(w, r, err)
			return
		}
		data := a.writable(atual, e, body, scope)
		if e.Singular == a.app.MemberModel && data["pessoa_id"] == nil {
			if v, ok := body["user_id"]; ok {
				data["pessoa_id"] = v
			}
		}
		if !a.canCreate(ctx, atual, e, data) {
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
		a.json(w, 200, serialize(e, urow), nil)
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
			result = serialize(e, a.find(ctx, e, fmt.Sprint(row["id"]), nil))
		}
		a.json(w, 200, result, nil)
	}
}

func (a *intentAPI) hookVars(atual map[string]any, e *ast.Entity, row map[string]any, body map[string]any) map[string]any {
	return map[string]any{"atual": nilIfEmpty(atual), "registro": row, e.Singular: row, "dados": body}
}

// canCreate checks `criar`; a generic signed-in permission does not let
// anyone create things inside a resource that has members — they must be a
// member of it (or hold a role granted for this action).
func (a *intentAPI) canCreate(ctx *interp.Context, atual map[string]any, e *ast.Entity, data map[string]any) bool {
	if a.in.IsAdmin(atual) {
		return true
	}
	for _, rule := range e.Rules["criar"] {
		ok := false
		switch {
		case rule.Anyone:
			ok = true
		case rule.SignedIn:
			ok = atual != nil
		default:
			ok = a.in.Can(ctx, atual, e, "criar", data)
		}
		if !ok {
			continue
		}
		if (rule.Anyone || rule.SignedIn) && !rule.Own {
			if lvl, membered := a.memberedParentLevel(ctx, atual, e, data); membered && lvl < a.app.Roles[0].Level {
				continue
			}
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
	res, err := a.in.Op(ctx, e.Singular, "criar", data)
	if err != nil {
		a.failErr(w, r, err)
		return
	}
	row := res.(map[string]any)
	out := serialize(e, row)
	for _, f := range e.Model.Fields {
		if f.Type == ast.FieldSegredo {
			out[strings.ToLower(f.Name)] = row[strings.ToLower(f.Name)] // shown once
		}
	}
	if h := e.Hooks["criar"]; h != nil {
		if _, _, err := a.in.RunHook(ctx, h, a.hookVars(atual, e, row, body)); err != nil {
			a.in.Op(ctx, e.Singular, "deletar", row["id"]) // nothing stays half-created
			a.failErr(w, r, err)
			return
		}
		fresh := serialize(e, a.find(ctx, e, fmt.Sprint(row["id"]), nil))
		for k, v := range fresh {
			out[k] = v
		}
	}
	a.json(w, 201, out, nil)
}

// remove runs `quando excluir` first (it may refuse), then deletes the
// record and everything that belongs to it.
func (a *intentAPI) remove(ctx *interp.Context, atual map[string]any, e *ast.Entity, row map[string]any) error {
	if h := e.Hooks["excluir"]; h != nil {
		if _, _, err := a.in.RunHook(ctx, h, a.hookVars(atual, e, row, nil)); err != nil {
			return err
		}
	}
	return a.cascade(ctx, e, row, 0)
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
	_, err := a.in.Op(ctx, e.Singular, "deletar", row["id"])
	return err
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
		if v := q.Get(f); v != "" {
			filters[f] = v
		}
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
	if atual == nil && e.Visibility == "" && !anyoneMay(e) {
		a.fail(w, 401, a.msg("401", e))
		return
	}
	var items []any
	total := 0
	if !interp.RecordDependent(e) {
		if !a.in.Can(ctx, atual, e, "ver", nil) {
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
			items = append(items, serialize(e, it.(map[string]any)))
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
				visible = append(visible, serialize(e, row))
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
