package servidor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Pages for `crie página X` + `mostre xs` + `permita …`. Everything is
// rendered on the server with html/template (automatic escaping). Every
// button is a form that posts to a page action, which calls the same
// internal operation as the API (same rules, validation and hooks) and
// redirects with a message: a button is drawn only when the person can use
// it, and it always does something.

type pageSite struct {
	a     *intentAPI
	slugs map[string]*ast.PageDecl // slug → page
	order []*ast.PageDecl
}

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer("á", "a", "à", "a", "â", "a", "ã", "a", "é", "e", "ê", "e", "í", "i", "ó", "o", "ô", "o", "õ", "o", "ú", "u", "ç", "c").Replace(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
				b.WriteByte('-')
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func (s *Servidor) registerPages(mux *routeMux) {
	app := s.Program.App
	if app == nil || len(app.Pages) == 0 || s.intent == nil {
		if app != nil && app.Login != nil && s.intent != nil {
			ps := &pageSite{a: s.intent, slugs: map[string]*ast.PageDecl{}}
			ps.identityPages(mux)
		}
		return
	}
	ps := &pageSite{a: s.intent, slugs: map[string]*ast.PageDecl{}}
	for _, pg := range app.Pages {
		if pg.Show == "" && len(pg.Indicators) == 0 {
			continue
		}
		sl := slug(pg.Name)
		ps.slugs[sl] = pg
		ps.order = append(ps.order, pg)
		mux.HandleFunc("GET /"+sl, ps.serve)
		mux.HandleFunc("GET /"+sl+"/{rest...}", ps.serve)
		mux.HandleFunc("POST /"+sl, ps.post)
		mux.HandleFunc("POST /"+sl+"/{rest...}", ps.post)
	}
	if len(ps.order) > 0 {
		first := "/" + slug(ps.order[0].Name)
		mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, first, http.StatusSeeOther)
		})
	}
	if app.Login != nil {
		ps.identityPages(mux)
	}
}

// ---------- internal calls: pages use the same operations as the API ----------

func (ps *pageSite) call(r *http.Request, method, path string, body any) (int, any, string) {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Cookie", r.Header.Get("Cookie"))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if method != http.MethodGet {
		if sess := interp.SessaoDaRequisicao(r); sess != nil {
			req.Header.Set("X-CSRF-Token", fmt.Sprint(sess["csrf"]))
		}
	}
	rec := httptest.NewRecorder()
	ps.a.s.mux.ServeHTTP(rec, req)
	raw := rec.Body.String()
	var out any
	json.Unmarshal([]byte(raw), &out)
	return rec.Code, out, raw
}

// message turns an API error into a sentence for people.
func message(out any) string {
	m, ok := out.(map[string]any)
	if !ok {
		return "Não foi possível concluir a ação."
	}
	switch x := m["message"].(type) {
	case string:
		return x
	case map[string]any:
		var keys []string
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var parts []string
		for _, k := range keys {
			for _, e := range asList(x[k]) {
				parts = append(parts, strings.ReplaceAll(k, "_", " ")+" "+fmt.Sprint(e))
			}
		}
		return strings.Join(parts, "; ")
	}
	if e, ok := m["error"].(string); ok {
		return e
	}
	return "Não foi possível concluir a ação."
}

func asList(v any) []any {
	if l, ok := v.([]any); ok {
		return l
	}
	return []any{v}
}

// ---------- routing: /<página>/<ref>/<filhos>/<ref>… mirrors the data ----------

type step struct {
	e   *ast.Entity
	ref string
}

// resolve maps page path parts to the chain of data and the API path.
func (ps *pageSite) resolve(pg *ast.PageDecl, parts []string) ([]step, string, []string, bool) {
	root := ps.a.app.Entities[pg.Show]
	chain := []step{{e: root}}
	api := "/_ge/api/" + root.Plural
	i := 0
	for i < len(parts) {
		cur := &chain[len(chain)-1]
		if cur.ref == "" {
			cur.ref = parts[i]
			api += "/" + url.PathEscape(parts[i])
			i++
			continue
		}
		var next *ast.Entity
		for _, c := range ps.a.childrenOf(cur.e) {
			if c.Plural == parts[i] {
				next = c
			}
		}
		if next == nil {
			return chain, api, parts[i:], true // remaining parts are a view (codigo, commit…)
		}
		chain = append(chain, step{e: next})
		api += "/" + next.Plural
		i++
	}
	return chain, api, nil, true
}

func pagePath(pg *ast.PageDecl, chain []step, upto int) string {
	p := "/" + slug(pg.Name)
	for i := 0; i <= upto && i < len(chain); i++ {
		if i > 0 {
			p += "/" + chain[i].e.Plural
		}
		if chain[i].ref != "" && (i < upto || true) {
			if i < upto {
				p += "/" + url.PathEscape(chain[i].ref)
			}
		}
	}
	return p
}

func (ps *pageSite) pageOf(r *http.Request) (*ast.PageDecl, []string) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	pg := ps.slugs[parts[0]]
	var rest []string
	for _, p := range parts[1:] {
		if p != "" {
			u, err := url.PathUnescape(p)
			if err != nil {
				u = p
			}
			rest = append(rest, u)
		}
	}
	return pg, rest
}

// ---------- rendering ----------

type link struct{ Href, Text string }

type view struct {
	System   string
	Title    string
	Nav      []link
	Crumbs   []link
	User     map[string]any
	UserName string
	Login    bool
	Signup   bool
	CSRF     string
	Flash    string
	Error    string
	Body     template.HTML
	Meta     template.HTML // description, canonical, Open Graph (seo.go)
}

func (ps *pageSite) base(r *http.Request, title string) *view {
	app := ps.a.app
	v := &view{System: ps.a.s.Program.System.Name, Title: title, Login: app.Login != nil, Signup: app.Login != nil && app.Login.Signup}
	for _, pg := range ps.order {
		v.Nav = append(v.Nav, link{"/" + slug(pg.Name), pg.Name})
	}
	ctx := &interp.Context{Request: r}
	if u, _ := ps.a.s.identify(ctx, r); u != nil {
		v.User = u
		v.UserName = first(toStr(u["nome"]), toStr(u["name"]), toStr(u["username"]), toStr(u["email"]))
	}
	if sess := interp.SessaoDaRequisicao(r); sess != nil {
		v.CSRF = fmt.Sprint(sess["csrf"])
	}
	v.Flash = r.URL.Query().Get("ok")
	v.Error = r.URL.Query().Get("erro")
	seo := ps.a.s.seo()
	icon := ""
	if t := ps.a.s.Program.Theme; t != nil && isImageRef(t.Icon) {
		icon = t.Icon
	}
	v.Meta = template.HTML(metaHTML(pageMeta{Title: title + " · " + v.System, Canonical: seo.abs(r.URL.EscapedPath()), Icon: icon, Image: seo.abs(icon)}))
	return v
}

func (ps *pageSite) render(w http.ResponseWriter, v *view, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; frame-ancestors 'none'")
	w.WriteHeader(status)
	if err := layoutTpl.Execute(w, v); err != nil {
		fmt.Fprintf(w, "erro ao desenhar a página: %v", err)
	}
}

func htmlOf(t *template.Template, data any) template.HTML {
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return template.HTML(template.HTMLEscapeString(err.Error()))
	}
	return template.HTML(b.String())
}

// columns: what a list shows for a kind of data.
func columns(e *ast.Entity) []*ast.Field {
	var cols []*ast.Field
	for _, f := range e.Model.Fields {
		if f.Hidden || f.IsSecret() || f.Private || f.Type == ast.FieldTextoLongo || f.Type == ast.FieldLista || isFileField(f) {
			continue
		}
		if f.System && strings.ToLower(f.Name) != "estado" {
			continue
		}
		if f.Reference != "" && strings.HasSuffix(f.Name, "_id") {
			continue
		}
		switch strings.ToLower(f.Name) {
		case "titulo", "nome", "name", "title", "numero":
			continue // already in the first column
		}
		cols = append(cols, f)
		if len(cols) == 6 {
			break
		}
	}
	return cols
}

// fieldLabel: the author's spelling when known.
func fieldLabel(f *ast.Field) string {
	if f.Label != "" && !strings.HasSuffix(f.Name, "_id") {
		return strings.ToUpper(f.Label[:1]) + f.Label[1:]
	}
	return label(f.Name)
}

func label(name string) string {
	name = strings.ReplaceAll(strings.TrimSuffix(name, "_id"), "_", " ")
	if name == "" {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

func display(v any) string {
	switch x := v.(type) {
	case nil:
		return "—"
	case bool:
		if x {
			return "sim"
		}
		return "não"
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case []any:
		var parts []string
		for _, it := range x {
			parts = append(parts, display(it))
		}
		if len(parts) == 0 {
			return "—"
		}
		return strings.Join(parts, ", ")
	}
	s := fmt.Sprint(v)
	if s == "" {
		return "—"
	}
	return s
}

// formattedBody: a record without a title whose main text is formatted
// (a comment) is shown whole, as safe Markdown, instead of a truncated title.
func formattedBody(e *ast.Entity, row map[string]any) (template.HTML, bool) {
	for _, k := range []string{"titulo", "nome", "name", "title", "endereco", "caminho_completo", "username", "email"} {
		if v, ok := row[k]; ok && v != nil && fmt.Sprint(v) != "" {
			return "", false
		}
	}
	for _, f := range e.Model.Fields {
		if f.Type == ast.FieldTextoLongo && !f.Hidden && !f.System {
			if !f.Formatted {
				return "", false
			}
			if t := toStr(row[strings.ToLower(f.Name)]); t != "" {
				return template.HTML(interp.Markdown(t)), true
			}
		}
	}
	return "", false
}

func titleOf(e *ast.Entity, row map[string]any) string {
	for _, k := range []string{"titulo", "nome", "name", "title", "endereco", "caminho_completo", "username", "email"} {
		if v, ok := row[k]; ok && v != nil && fmt.Sprint(v) != "" {
			if n, ok := row["numero"]; ok && n != nil {
				return "#" + display(n) + " " + fmt.Sprint(v)
			}
			return fmt.Sprint(v)
		}
	}
	// Without a title, the main text summarises the record (comments).
	for _, f := range e.Model.Fields {
		if f.Type == ast.FieldTextoLongo && !f.Hidden && !f.System {
			if t := toStr(row[strings.ToLower(f.Name)]); t != "" {
				if r := []rune(t); len(r) > 80 {
					t = string(r[:80]) + "…"
				}
				return t
			}
		}
	}
	return e.Label + " " + display(row["id"])
}

type cell struct {
	Text  string
	Badge bool
}
type tableRow struct {
	Href  string
	Title string
	Body  template.HTML // full formatted text of records without a title (comments)
	Cells []cell
}
type tableData struct {
	Heads   []string
	Rows    []tableRow
	Empty   string
	Caption string
}

func (ps *pageSite) table(e *ast.Entity, rows []any, base string) template.HTML {
	return ps.tableWith(e, rows, base, nil)
}

// tableWith renders rows with the given columns (field names, in order), or
// the entity's visible fields when names is empty.
func (ps *pageSite) tableWith(e *ast.Entity, rows []any, base string, names []string) template.HTML {
	cols := columns(e)
	if len(names) > 0 {
		cols = nil
		for _, n := range names {
			for _, f := range e.Model.Fields {
				if strings.EqualFold(f.Name, n) {
					cols = append(cols, f)
					break
				}
			}
		}
	}
	td := tableData{Empty: fmt.Sprintf("Nenhum registro de %s ainda.", strings.ToLower(e.Label)), Caption: e.Label + "s"}
	td.Heads = append(td.Heads, e.Label)
	for _, c := range cols {
		td.Heads = append(td.Heads, fieldLabel(c))
	}
	for _, it := range rows {
		row, _ := it.(map[string]any)
		ref := display(row["id"])
		if n, ok := row["numero"]; ok && n != nil {
			ref = display(n)
		}
		tr := tableRow{Href: base + "/" + url.PathEscape(ref), Title: titleOf(e, row)}
		if body, ok := formattedBody(e, row); ok {
			tr.Title, tr.Body = e.Label+" "+display(row["id"]), body
		}
		for _, c := range cols {
			tr.Cells = append(tr.Cells, cell{Text: display(row[strings.ToLower(c.Name)]), Badge: strings.ToLower(c.Name) == "estado" || c.Type == ast.FieldVisibilidade})
		}
		td.Rows = append(td.Rows, tr)
	}
	return htmlOf(tableTpl, td)
}

type input struct {
	Name, Label, Type, Value string
	Required, Checked        bool
	Options                  []option
	Multiple                 bool
}
type option struct {
	Value, Text string
	Selected    bool
}
type formData struct {
	Action, Method, Submit, CSRF, Title string
	Inputs                              []input
	Danger                              bool
}

// inputs builds form fields for the writable fields of e.
func (ps *pageSite) inputs(r *http.Request, chain []step, e *ast.Entity, values map[string]any, fixed map[string]any) []input {
	var out []input
	for _, f := range e.Model.Fields {
		key := strings.ToLower(f.Name)
		if f.Hidden || f.System || f.Type == ast.FieldSegredo || f.NumberedBy != "" {
			continue
		}
		if _, isFixed := fixed[key]; isFixed {
			continue
		}
		if e.Singular == ps.a.app.LoginEntity && (key == "admin" || key == "papel") {
			continue
		}
		owner := false
		for _, o := range e.OwnerFields {
			owner = owner || o == key
		}
		if owner || (e.Singular == ps.a.app.MemberModel && (key == "recurso" || key == "recurso_id")) {
			continue
		}
		if values != nil && f.Immutable {
			continue
		}
		if isFileField(f) {
			continue // files are sent in their own form, on the record's page
		}
		in := input{Name: key, Label: fieldLabel(f), Type: "text", Required: f.Required && values == nil}
		v := values[key]
		if v != nil {
			in.Value = display(v)
		}
		switch f.Type {
		case ast.FieldEmail:
			in.Type = "email"
		case ast.FieldSenha:
			in.Type, in.Value = "password", ""
			if values != nil {
				continue
			}
		case ast.FieldInteiro, ast.FieldNumero, ast.FieldDinheiro:
			in.Type = "number"
		case ast.FieldBooleano:
			in.Type = "checkbox"
			in.Checked = v == true || (v == nil && f.HasDefault && f.DefaultValue == true)
		case ast.FieldTextoLongo:
			in.Type = "textarea"
		case ast.FieldData:
			in.Type = "date"
		case ast.FieldVisibilidade:
			in.Type = "select"
			for _, o := range []string{"private", "internal", "public"} {
				in.Options = append(in.Options, option{o, map[string]string{"private": "Privado", "internal": "Interno", "public": "Público"}[o], display(v) == o})
			}
		case ast.FieldEnum:
			in.Type = "select"
			for _, o := range f.EnumValues {
				in.Options = append(in.Options, option{o, label(o), display(v) == o})
			}
		case ast.FieldBranch:
			in.Type = "select"
			for _, b := range ps.branches(r, chain) {
				in.Options = append(in.Options, option{b, b, display(v) == b})
			}
		case ast.FieldLista:
			if f.ListOf == "texto" {
				break
			}
			in.Type, in.Multiple = "select", true
			selected := map[string]bool{}
			for _, it := range asList(v) {
				selected[display(it)] = true
			}
			for _, opt := range ps.optionsFor(r, chain, f.ListOf) {
				opt.Selected = selected[opt.Value]
				in.Options = append(in.Options, opt)
			}
		}
		if f.Reference != "" && f.Type == ast.FieldInteiro {
			in.Type = "select"
			if !f.Required {
				in.Options = append(in.Options, option{"", "—", v == nil})
			}
			for _, opt := range ps.optionsFor(r, chain, f.Reference) {
				opt.Selected = display(v) == opt.Value
				in.Options = append(in.Options, opt)
			}
		}
		out = append(out, in)
	}
	return out
}

// optionsFor lists records of entity name the person can see, preferring
// the ones that belong to an ancestor in the current chain (labels of this
// project).
func (ps *pageSite) optionsFor(r *http.Request, chain []step, name string) []option {
	target := ps.a.app.Entities[name]
	if target == nil {
		return nil
	}
	path := "/_ge/api/" + target.Plural + "?per_page=100"
	api := "/_ge/api/" + chain[0].e.Plural
	for _, st := range chain {
		if st.ref == "" {
			break
		}
		if st.e != chain[0].e {
			api += "/" + st.e.Plural
		}
		api += "/" + url.PathEscape(st.ref)
		for _, c := range ps.a.childrenOf(st.e) {
			if c == target {
				path = api + "/" + target.Plural + "?per_page=100"
			}
		}
	}
	code, out, _ := ps.call(r, "GET", path, nil)
	if code != 200 {
		return nil
	}
	var opts []option
	for _, it := range asList(out) {
		row, _ := it.(map[string]any)
		if row != nil {
			opts = append(opts, option{Value: display(row["id"]), Text: titleOf(target, row)})
		}
	}
	return opts
}

func (ps *pageSite) branches(r *http.Request, chain []step) []string {
	api := "/_ge/api/" + chain[0].e.Plural
	for i, st := range chain {
		if st.ref == "" {
			break
		}
		if i > 0 {
			api += "/" + st.e.Plural
		}
		api += "/" + url.PathEscape(st.ref)
		if st.e.Repository {
			code, out, _ := ps.call(r, "GET", api+"/repositorio/branches", nil)
			if code != 200 {
				return nil
			}
			var names []string
			for _, it := range asList(out) {
				if m, ok := it.(map[string]any); ok {
					names = append(names, fmt.Sprint(m["name"]))
				}
			}
			return names
		}
	}
	return nil
}

func (ps *pageSite) form(action, submit, csrf, title string, inputs []input) template.HTML {
	return htmlOf(formTpl, formData{Action: action, Method: "post", Submit: submit, CSRF: csrf, Title: title, Inputs: inputs})
}

func (ps *pageSite) button(action, text, csrf string, danger bool) template.HTML {
	return htmlOf(formTpl, formData{Action: action, Method: "post", Submit: text, CSRF: csrf, Danger: danger})
}

// ---------- GET ----------

func (ps *pageSite) serve(w http.ResponseWriter, r *http.Request) {
	pg, parts := ps.pageOf(r)
	if pg == nil {
		http.NotFound(w, r)
		return
	}
	if pg.Show == "" {
		if len(parts) > 0 {
			http.NotFound(w, r)
			return
		}
		ps.dashboard(w, r, pg)
		return
	}
	chain, api, view, _ := ps.resolve(pg, parts)
	v := ps.base(r, pg.Name)
	v.Crumbs = append(v.Crumbs, link{"/" + slug(pg.Name), pg.Name})
	last := chain[len(chain)-1]
	ctx := &interp.Context{Request: r}
	atual, _ := ps.a.s.identify(ctx, r)
	var body bytes.Buffer
	status := http.StatusOK

	// crumbs for every resolved record, named by its title
	href := "/" + slug(pg.Name)
	scope := map[string]any{}
	for i, st := range chain {
		if i > 0 {
			href += "/" + st.e.Plural
			v.Crumbs = append(v.Crumbs, link{href, st.e.Label + "s"})
		}
		if st.ref != "" {
			href += "/" + url.PathEscape(st.ref)
			text := "#" + st.ref
			if row := ps.a.find(ctx, st.e, st.ref, scope); row != nil && ps.a.in.Can(ctx, atual, st.e, "ver", row) {
				text = titleOf(st.e, row)
				if i+1 < len(chain) {
					scope = ps.a.parentScope(st.e, row, chain[i+1].e)
				}
			}
			v.Crumbs = append(v.Crumbs, link{href, text})
		}
	}

	if last.ref == "" {
		// Listing
		q := r.URL.Query()
		apiQ := url.Values{}
		for _, k := range []string{"q", "pagina"} {
			if q.Get(k) != "" {
				apiQ.Set(k, q.Get(k))
			}
		}
		for _, f := range last.e.Filters {
			if q.Get(f) != "" {
				apiQ.Set(f, q.Get(f))
			}
		}
		per := 20
		if pg.PerPage > 0 && len(chain) == 1 {
			per = pg.PerPage
		}
		apiQ.Set("por_pagina", strconv.Itoa(per))
		code, out, _ := ps.call(r, "GET", api+"?"+apiQ.Encode(), nil)
		if code == 401 {
			http.Redirect(w, r, "/entrar", http.StatusSeeOther)
			return
		}
		if code != 200 {
			ps.notFound(w, v, message(out))
			return
		}
		// The page's own sections (GEP 0002, em teste) replace only their
		// default, and only on the page itself (not on nested lists).
		own := len(chain) == 1
		v.Title = last.e.Label + "s"
		if own && pg.Title != "" {
			v.Title = pg.Title
		}
		canCreate := atual != nil && ps.a.canCreateFor(ctx, atual, last.e, chain)
		createLabel := "Novo " + strings.ToLower(last.e.Label)
		body.WriteString(string(htmlOf(headingTpl, v.Title)))
		if own && pg.Text != "" {
			body.WriteString(string(htmlOf(introTpl, pg.Text)))
		}
		if own && len(pg.Indicators) > 0 {
			body.WriteString(string(ps.indicators(ctx, atual, pg)))
		}
		if own && canCreate {
			for _, act := range pg.Actions {
				label := act.Label
				if label == "" {
					label = createLabel
				}
				createLabel = label
				body.WriteString(string(htmlOf(actionLinkTpl, map[string]any{"Href": "#novo", "Label": label})))
			}
		}
		search, filters := len(last.e.Search) > 0, last.e.Filters
		if own && len(pg.Filters) > 0 {
			search, filters = false, nil
			for _, f := range pg.Filters {
				if f == "pesquisar" {
					search = true
				} else {
					filters = append(filters, f)
				}
			}
		}
		if search || len(filters) > 0 {
			body.WriteString(string(htmlOf(searchTpl, map[string]any{"Q": q.Get("q"), "Search": search, "Filters": filters, "Values": q})))
		}
		base := strings.TrimSuffix(r.URL.Path, "/")
		rows := asList(out)
		if own && len(rows) == 0 && pg.Empty != nil && q.Get("q") == "" {
			empty := map[string]any{"Title": pg.Empty.Title, "Text": pg.Empty.Text}
			if pg.Empty.Action != nil && canCreate {
				label := pg.Empty.Action.Label
				if label == "" {
					label = createLabel
				}
				empty["Action"] = map[string]any{"Href": "#novo", "Label": label}
			}
			body.WriteString(string(htmlOf(emptyStateTpl, empty)))
		} else {
			var cols []string
			if own {
				cols = pg.Columns
			}
			body.WriteString(string(ps.tableWith(last.e, rows, base, cols)))
		}
		body.WriteString(string(htmlOf(pagerTpl, pager(q, len(rows), per))))
		if canCreate {
			body.WriteString(`<div id="novo"></div>`)
			body.WriteString(string(ps.form(base+"/novo", "Criar", v.CSRF, createLabel, ps.inputs(r, chain, last.e, nil, ps.fixedFor(chain)))))
		}
		v.Body = template.HTML(body.String())
		ps.render(w, v, status)
		return
	}

	code, out, _ := ps.call(r, "GET", api, nil)
	if code == 401 {
		http.Redirect(w, r, "/entrar", http.StatusSeeOther)
		return
	}
	if code != 200 {
		ps.notFound(w, v, message(out))
		return
	}
	row, _ := out.(map[string]any)
	record := ps.record(ctx, last.e, row)
	v.Title = titleOf(last.e, row)
	if len(view) > 0 {
		ps.repositoryView(w, r, v, chain, api, view)
		return
	}
	body.WriteString(string(htmlOf(headingTpl, v.Title)))
	body.WriteString(string(htmlOf(detailTpl, ps.details(ctx, last.e, row))))
	base := strings.TrimSuffix(r.URL.Path, "/")
	body.WriteString(string(ps.actions(ctx, atual, last.e, record, base, v.CSRF)))
	body.WriteString(string(ps.fileViews(ctx, atual, last.e, record, row, api, base, v.CSRF)))
	body.WriteString(string(ps.capabilityViews(r, last.e, record, api, base)))
	if atual != nil && ps.a.in.Can(ctx, atual, last.e, "editar", record) {
		body.WriteString(string(ps.form(base+"/editar", "Salvar", v.CSRF, "Editar", ps.inputs(r, chain, last.e, row, nil))))
	}
	// Children: what belongs to this record
	for _, c := range ps.a.childrenOf(last.e) {
		ccode, cout, _ := ps.call(r, "GET", api+"/"+c.Plural+"?por_pagina=10", nil)
		if ccode >= 500 {
			// an internal failure is shown, never hidden like a permission
			body.WriteString(string(htmlOf(sectionTpl, map[string]any{"Title": c.Label + "s"})))
			body.WriteString(string(htmlOf(emptyTpl, "Não foi possível carregar esta parte agora. Tente de novo mais tarde.")))
			continue
		}
		if ccode != 200 {
			continue // not allowed to see: nothing is shown
		}
		body.WriteString(string(htmlOf(sectionTpl, map[string]any{"Title": c.Label + "s", "Href": base + "/" + c.Plural})))
		body.WriteString(string(ps.table(c, asList(cout), base+"/"+c.Plural)))
		childChain := append(append([]step{}, chain...), step{e: c})
		if atual != nil && ps.a.canCreateFor(ctx, atual, c, childChain) {
			body.WriteString(string(ps.form(base+"/"+c.Plural+"/novo", "Criar", v.CSRF, "Novo "+strings.ToLower(c.Label), ps.inputs(r, childChain, c, nil, ps.fixedFor(childChain)))))
		}
	}
	v.Body = template.HTML(body.String())
	ps.render(w, v, status)
}

func (ps *pageSite) notFound(w http.ResponseWriter, v *view, msg string) {
	v.Title = "Não encontrado"
	v.Body = htmlOf(emptyTpl, msg)
	ps.render(w, v, http.StatusNotFound)
}

// record loads the full record (with private fields removed as for the API)
// for permission checks.
func (ps *pageSite) record(ctx *interp.Context, e *ast.Entity, row map[string]any) map[string]any {
	res, err := ps.a.in.Op(ctx, e.Singular, "buscar", row["id"])
	if m, ok := res.(map[string]any); ok && err == nil {
		return m
	}
	return row
}

type detailItem struct {
	Label, Value string
	HTML         template.HTML // formatted text (safe Markdown)
}

func (ps *pageSite) details(ctx *interp.Context, e *ast.Entity, row map[string]any) []detailItem {
	var out []detailItem
	for _, f := range e.Model.Fields {
		key := strings.ToLower(f.Name)
		v, ok := row[key]
		if !ok || f.Hidden || f.IsSecret() || isFileField(f) {
			continue // files have their own section (fileViews)
		}
		text := display(v)
		// References show who/what they point to, not a number.
		if ref := f.Reference; ref != "" && v != nil {
			text = ps.nameOf(ctx, ref, v)
		} else if f.Type == ast.FieldLista && f.ListOf != "texto" {
			var names []string
			for _, it := range asList(v) {
				names = append(names, ps.nameOf(ctx, f.ListOf, it))
			}
			if len(names) > 0 {
				text = strings.Join(names, ", ")
			}
		}
		item := detailItem{Label: fieldLabel(f), Value: text}
		if f.Formatted && v != nil {
			item.HTML = template.HTML(interp.Markdown(toStr(v)))
		}
		out = append(out, item)
	}
	if v, ok := row["created_at"]; ok {
		out = append(out, detailItem{Label: "Criado em", Value: display(v)})
	}
	return out
}

// actions: one button per action the person may perform right now.
func (ps *pageSite) actions(ctx *interp.Context, atual map[string]any, e *ast.Entity, record map[string]any, base, csrf string) template.HTML {
	if atual == nil {
		return ""
	}
	var b strings.Builder
	verbs := map[string]bool{}
	for v := range e.Rules {
		if !isStandard(v) && v != "baixar_codigo" && v != "enviar_codigo" {
			verbs[v] = true
		}
	}
	for v := range e.Transitions {
		verbs[v] = true
	}
	if e.Approvals {
		verbs["aprovar"], verbs["desaprovar"] = true, true
	}
	if e.Execution != nil {
		verbs["cancelar"] = true
		verbs["repetir"] = true
		if e.Execution.Role == "step" {
			verbs["executar"] = true
		}
	}
	if (e.HasMembers || e.InheritVia != "") && ps.a.app.MemberModel != "" {
		verbs["sair"] = true
	}
	var names []string
	for v := range verbs {
		names = append(names, v)
	}
	sort.Strings(names)
	b.WriteString(`<div class="acoes">`)
	for _, v := range names {
		if !ps.available(ctx, atual, e, v, record) {
			continue
		}
		b.WriteString(string(ps.button(base+"/acao/"+v, label(v), csrf, v == "cancelar" || v == "sair")))
	}
	if ps.a.in.Can(ctx, atual, e, "excluir", record) {
		b.WriteString(string(ps.button(base+"/excluir", "Excluir", csrf, true)))
	}
	b.WriteString(`</div>`)
	return template.HTML(b.String())
}

// available: the action is allowed and makes sense in the current state.
func (ps *pageSite) available(ctx *interp.Context, atual map[string]any, e *ast.Entity, verb string, record map[string]any) bool {
	st := toStr(record["estado"])
	for _, f := range e.Finals {
		if st == f && e.Transitions[verb] != nil {
			return false
		}
	}
	if tr := e.Transitions[verb]; tr != nil && st == tr.Target {
		return false
	}
	if verb == "mesclar" && st != e.Initial {
		return false
	}
	if e.Execution != nil {
		switch verb {
		case "cancelar":
			if st != stPending && st != stRunning && st != stCreated && st != stManual {
				return false
			}
		case "repetir":
			if st != stFailed && st != stCanceled && !(e.Execution.Role == "step" && st == stSuccess) {
				return false
			}
		case "executar":
			if st != stManual {
				return false
			}
		}
	}
	if verb == "sair" {
		res, _ := ps.a.in.Op(ctx, ps.a.app.MemberModel, "encontrar", map[string]any{"recurso": e.Singular, "recurso_id": record["id"], "pessoa_id": atual["id"]})
		return res != nil
	}
	if verb == "aprovar" || verb == "desaprovar" {
		approved := false
		for _, v := range asList(record["aprovacoes"]) {
			approved = approved || display(v) == display(atual["id"])
		}
		if (verb == "aprovar") == approved {
			return false
		}
		return ps.a.in.Can(ctx, atual, e, "aprovar", record)
	}
	return ps.a.in.Can(ctx, atual, e, verb, record)
}

func (a *intentAPI) canCreateFor(ctx *interp.Context, atual map[string]any, e *ast.Entity, chain []step) bool {
	// Walk the chain like the API does (numbers are resolved inside parents).
	data := map[string]any{}
	scope := map[string]any{}
	for i := 0; i < len(chain)-1; i++ {
		row := a.find(ctx, chain[i].e, chain[i].ref, scope)
		if row == nil {
			return false
		}
		scope = a.parentScope(chain[i].e, row, chain[i+1].e)
	}
	for k, v := range scope {
		data[k] = v
	}
	for _, owner := range e.OwnerFields {
		data[owner] = atual["id"] // what is created belongs to whoever creates it
	}
	if e.Execution != nil && e.Execution.Role == "step" {
		return false
	}
	if e.Execution != nil && e.Execution.Role == "run" {
		for _, rule := range append(append([]*ast.AccessRule{}, e.Rules["criar"]...), e.Rules["executar"]...) {
			if a.in.RulePasses(ctx, atual, e, rule, data) {
				return true
			}
		}
		return a.in.IsAdmin(atual)
	}
	return a.canCreate(ctx, atual, e, data)
}

// fixedFor: values implied by the chain (the parent reference).
func (ps *pageSite) fixedFor(chain []step) map[string]any {
	out := map[string]any{}
	if len(chain) < 2 {
		return out
	}
	parent, child := chain[len(chain)-2], chain[len(chain)-1]
	if child.e.Singular == ps.a.app.MemberModel {
		out["recurso"], out["recurso_id"] = parent.e.Singular, parent.ref
		return out
	}
	for field, target := range child.e.Parents {
		if target == parent.e.Singular {
			out[field] = parent.ref
		}
	}
	return out
}

func pager(q url.Values, count, per int) map[string]any {
	page, _ := strconv.Atoi(q.Get("pagina"))
	if page < 1 {
		page = 1
	}
	mk := func(p int) string {
		v := url.Values{}
		for k, vals := range q {
			v[k] = vals
		}
		v.Set("pagina", strconv.Itoa(p))
		return "?" + v.Encode()
	}
	out := map[string]any{"Page": page}
	if page > 1 {
		out["Prev"] = mk(page - 1)
	}
	if count >= per {
		out["Next"] = mk(page + 1)
	}
	return out
}

// ---------- POST ----------

func (ps *pageSite) post(w http.ResponseWriter, r *http.Request) {
	pg, parts := ps.pageOf(r)
	if pg == nil || len(parts) == 0 && r.URL.Path != "/"+slug(pg.Name) || pg.Show == "" {
		http.NotFound(w, r) // a dashboard has nothing to change
		return
	}
	if len(parts) >= 3 && parts[len(parts)-2] == "arquivo" && strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		ps.uploadFromPage(w, r, pg, parts)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulário inválido", http.StatusBadRequest)
		return
	}
	sess := interp.SessaoDaRequisicao(r)
	if sess == nil {
		http.Redirect(w, r, "/entrar", http.StatusSeeOther)
		return
	}
	if r.PostForm.Get("_csrf") != fmt.Sprint(sess["csrf"]) {
		http.Error(w, "token CSRF ausente ou inválido", http.StatusForbidden)
		return
	}
	if len(parts) == 0 {
		http.NotFound(w, r)
		return
	}
	// The last part is the operation: novo, editar, excluir or acao/<verbo>.
	op := parts[len(parts)-1]
	verb := ""
	body := parts[:len(parts)-1]
	if len(parts) >= 2 && parts[len(parts)-2] == "acao" {
		verb, op, body = parts[len(parts)-1], "acao", parts[:len(parts)-2]
	}
	chain, api, _, _ := ps.resolve(pg, body)
	last := chain[len(chain)-1]
	back := "/" + slug(pg.Name)
	if len(body) > 0 {
		back += "/" + strings.Join(escapeAll(body), "/")
	}
	data := ps.formData(last.e, r.PostForm)
	var method, path, done string
	target := back
	switch op {
	case "novo":
		method, path, done = "POST", api, "Criado com sucesso"
	case "editar":
		method, path, done = "PATCH", api, "Alterações salvas"
	case "excluir":
		method, path, done = "DELETE", api, "Excluído"
		up := body[:len(body)-1]
		target = "/" + slug(pg.Name)
		if len(up) > 0 {
			// back to the parent record, not to its children list
			if len(up) >= 2 {
				up = up[:len(up)-1]
			}
			target += "/" + strings.Join(escapeAll(up), "/")
		}
	case "acao":
		method, path, done, data = "POST", api+"/"+verb, "Feito: "+label(verb), nil
	default:
		http.NotFound(w, r)
		return
	}
	code, out, _ := ps.call(r, method, path, data)
	if code >= 400 {
		http.Redirect(w, r, back+"?erro="+url.QueryEscape(message(out)), http.StatusSeeOther)
		return
	}
	if op == "novo" {
		if row, ok := out.(map[string]any); ok {
			ref := display(row["id"])
			if n, ok := row["numero"]; ok && n != nil {
				ref = display(n)
			}
			target = back + "/" + url.PathEscape(ref)
		}
	}
	http.Redirect(w, r, target+"?ok="+url.QueryEscape(done), http.StatusSeeOther)
}

func escapeAll(parts []string) []string {
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = url.PathEscape(p)
	}
	return out
}

// formData converts posted fields to typed values for the operation.
func (ps *pageSite) formData(e *ast.Entity, form url.Values) map[string]any {
	out := map[string]any{}
	for _, f := range e.Model.Fields {
		key := strings.ToLower(f.Name)
		vals, present := form[key]
		switch {
		case f.Type == ast.FieldBooleano:
			if _, shown := form["_campos"]; shown || present {
				out[key] = present && vals[0] != "" && vals[0] != "false"
			}
		case f.Type == ast.FieldLista && f.ListOf != "texto":
			if present || form.Get("_campos") != "" {
				var list []any
				for _, v := range vals {
					if n, err := strconv.ParseFloat(v, 64); err == nil {
						list = append(list, n)
					}
				}
				if list == nil {
					list = []any{}
				}
				out[key] = list
			}
		case present:
			v := strings.TrimSpace(vals[0])
			if v == "" && !f.Required {
				continue
			}
			if (f.Type == ast.FieldInteiro || f.Type == ast.FieldNumero || f.Type == ast.FieldDinheiro || f.Reference != "") && v != "" {
				if n, err := strconv.ParseFloat(v, 64); err == nil {
					out[key] = n
					continue
				}
			}
			out[key] = v
		}
	}
	return out
}

// ---------- capability views: code, reviews, logs ----------

func (ps *pageSite) capabilityViews(r *http.Request, e *ast.Entity, record map[string]any, api, base string) template.HTML {
	var b strings.Builder
	if e.Repository {
		code, out, _ := ps.call(r, "GET", api+"/repositorio/arvore", nil)
		if code == 200 {
			b.WriteString(string(htmlOf(sectionTpl, map[string]any{"Title": "Código", "Href": base + "/codigo"})))
			b.WriteString(string(htmlOf(treeTpl, map[string]any{"Base": base, "Path": "", "Entries": asList(out)})))
		} else if code == 404 {
			b.WriteString(string(htmlOf(sectionTpl, map[string]any{"Title": "Código"})))
			b.WriteString(string(htmlOf(emptyTpl, "O repositório ainda está vazio. Envie o primeiro commit com git push.")))
		}
		if code, out, _ := ps.call(r, "GET", api+"/repositorio/commits?per_page=5", nil); code == 200 {
			b.WriteString(string(htmlOf(commitsTpl, map[string]any{"Base": base, "Commits": asList(out)})))
		}
	}
	if e.Review != nil && e.Review.Target != "" {
		if code, out, _ := ps.call(r, "GET", api+"/mudancas", nil); code == 200 {
			m, _ := out.(map[string]any)
			b.WriteString(string(htmlOf(sectionTpl, map[string]any{"Title": "Mudanças"})))
			if can, ok := record["estado"].(string); ok && can == e.Initial {
				check := ps.a.mergeCheck(&interp.Context{Request: r}, e, record)
				if check != nil && check["pode"] == false {
					b.WriteString(string(htmlOf(emptyTpl, "Há conflitos: "+display(check["conflitos"]))))
				}
			}
			b.WriteString(string(ps.diffs(asList(m["mudancas"]))))
		}
	}
	if e.Execution != nil && e.Execution.Role == "step" {
		b.WriteString(string(htmlOf(sectionTpl, map[string]any{"Title": "Log"})))
		res, _ := ps.a.in.Op(&interp.Context{Request: r}, e.Singular, "buscar", record["id"])
		row, _ := res.(map[string]any)
		b.WriteString(string(htmlOf(logTpl, toStr(row["log"]))))
	}
	return template.HTML(b.String())
}

type diffLine struct{ Class, Text string }
type diffFile struct {
	Path  string
	Lines []diffLine
}

func (ps *pageSite) diffs(files []any) template.HTML {
	var out []diffFile
	for _, it := range files {
		m, _ := it.(map[string]any)
		if m == nil {
			continue
		}
		df := diffFile{Path: toStr(m["new_path"])}
		for _, l := range strings.Split(strings.TrimRight(toStr(m["diff"]), "\n"), "\n") {
			cls := ""
			switch {
			case strings.HasPrefix(l, "@@"):
				cls = "hunk"
			case strings.HasPrefix(l, "+"):
				cls = "add"
			case strings.HasPrefix(l, "-"):
				cls = "del"
			}
			df.Lines = append(df.Lines, diffLine{cls, l})
		}
		out = append(out, df)
	}
	return htmlOf(diffTpl, out)
}

// repositoryView: /…/<ref>/codigo?path=, /…/<ref>/commit/<sha>
func (ps *pageSite) repositoryView(w http.ResponseWriter, r *http.Request, v *view, chain []step, api string, parts []string) {
	base := strings.TrimSuffix(r.URL.Path, "/"+strings.Join(escapeAll(parts), "/"))
	var body strings.Builder
	switch parts[0] {
	case "codigo":
		path := strings.Trim(r.URL.Query().Get("caminho"), "/")
		code, out, _ := ps.call(r, "GET", api+"/repositorio/arvore?path="+url.QueryEscape(path), nil)
		if code == 200 && len(asList(out)) > 0 {
			body.WriteString(string(htmlOf(headingTpl, "Código: /"+path)))
			body.WriteString(string(htmlOf(treeTpl, map[string]any{"Base": base, "Path": path, "Entries": asList(out)})))
			break
		}
		code, out, _ = ps.call(r, "GET", api+"/repositorio/arquivos/"+path, nil)
		if code != 200 {
			ps.notFound(w, v, "Arquivo não encontrado")
			return
		}
		m, _ := out.(map[string]any)
		body.WriteString(string(htmlOf(headingTpl, path)))
		var lines []map[string]any
		for i, l := range strings.Split(toStr(m["content"]), "\n") {
			lines = append(lines, map[string]any{"N": i + 1, "Text": l})
		}
		body.WriteString(string(htmlOf(codeTpl, map[string]any{"Lines": lines, "Binary": m["binary"] == true})))
	case "commit":
		if len(parts) < 2 {
			ps.notFound(w, v, "Commit não encontrado")
			return
		}
		code, out, _ := ps.call(r, "GET", api+"/repositorio/commits/"+url.PathEscape(parts[1]), nil)
		if code != 200 {
			ps.notFound(w, v, "Commit não encontrado")
			return
		}
		c, _ := out.(map[string]any)
		body.WriteString(string(htmlOf(headingTpl, toStr(c["title"]))))
		body.WriteString(string(htmlOf(detailTpl, []detailItem{{Label: "Commit", Value: toStr(c["id"])}, {Label: "Autor", Value: toStr(c["author_name"]) + " <" + toStr(c["author_email"]) + ">"}, {Label: "Data", Value: toStr(c["authored_date"])}})))
		_, dout, _ := ps.call(r, "GET", api+"/repositorio/commits/"+url.PathEscape(parts[1])+"/diff", nil)
		body.WriteString(string(ps.diffs(asList(dout))))
	default:
		ps.notFound(w, v, "Página não encontrada")
		return
	}
	v.Body = template.HTML(body.String())
	ps.render(w, v, http.StatusOK)
}

// ---------- identity pages ----------

func (ps *pageSite) identityPages(mux *routeMux) {
	app := ps.a.app
	le := app.Entities[app.LoginEntity]
	mux.HandleFunc("GET /entrar", func(w http.ResponseWriter, r *http.Request) {
		v := ps.base(r, "Entrar")
		if r.URL.Query().Get("erro") == "1" {
			v.Error = "Login ou senha incorretos."
		}
		if r.URL.Query().Get("redefinida") == "1" {
			v.Flash = "Senha nova criada. Entre com ela."
		}
		v.Body = htmlOf(loginTpl, map[string]any{"Signup": v.Signup, "Label": strings.Join(app.Login.Fields, " ou "), "Recovery": app.Login.Recovery})
		ps.render(w, v, http.StatusOK)
	})
	if app.Login.Recovery {
		mux.HandleFunc("GET /esqueci", func(w http.ResponseWriter, r *http.Request) {
			v := ps.base(r, "Esqueci minha senha")
			if r.URL.Query().Get("enviado") == "1" {
				v.Flash = "Se houver uma conta com esse login, enviamos um link para criar uma senha nova."
			}
			if !ps.a.recoveryAvailable {
				v.Error = "A recuperação de senha ainda não está disponível neste sistema."
			}
			v.Body = htmlOf(forgotTpl, map[string]any{"Label": strings.Join(app.Login.Fields, " ou ")})
			ps.render(w, v, http.StatusOK)
		})
		mux.HandleFunc("GET /redefinir", func(w http.ResponseWriter, r *http.Request) {
			v := ps.base(r, "Criar senha nova")
			if msg := r.URL.Query().Get("erro"); msg != "" {
				v.Error = msg
			}
			v.Body = htmlOf(resetTpl, map[string]any{"Token": r.URL.Query().Get("token")})
			ps.render(w, v, http.StatusOK)
		})
	}
	if app.Login.Signup {
		mux.HandleFunc("GET /cadastro", func(w http.ResponseWriter, r *http.Request) {
			v := ps.base(r, "Criar conta")
			var inputs []input
			for _, f := range le.Model.Fields {
				switch f.Type {
				case ast.FieldTexto, ast.FieldEmail, ast.FieldSenha, ast.FieldTelefone:
					if f.Hidden || f.System {
						continue
					}
					t := "text"
					if f.Type == ast.FieldEmail {
						t = "email"
					} else if f.Type == ast.FieldSenha {
						t = "password"
					}
					inputs = append(inputs, input{Name: strings.ToLower(f.Name), Label: label(f.Name), Type: t, Required: f.Required})
				}
			}
			v.Body = htmlOf(formTpl, formData{Action: "/cadastro", Method: "post", Submit: "Criar conta", Title: "Criar conta", Inputs: inputs})
			ps.render(w, v, http.StatusOK)
		})
	}
}

// ---------- templates ----------

var tplFuncs = template.FuncMap{"label": label, "display": display, "lower": strings.ToLower,
	"get":  func(m map[string]any, k string) string { return display(m[k]) },
	"getv": func(q url.Values, k string) string { return q.Get(k) },
	"join": func(base, p string) string {
		if p == "" {
			return base
		}
		return base + "/" + p
	}}

func tpl(src string) *template.Template {
	return template.Must(template.New("").Funcs(tplFuncs).Parse(src))
}

var layoutTpl = tpl(`<!doctype html><html lang="pt-BR"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{.Title}} · {{.System}}</title>
{{.Meta}}<style>
:root{--fg:#1f2328;--muted:#656d76;--line:#d0d7de;--field:#8c959f;--bg:#fff;--soft:#f6f8fa;--accent:#6e40c9;--on-accent:#fff;--ok:#1a7f37;--bad:#cf222e;--on-bad:#fff}
@media (prefers-color-scheme:dark){:root{--fg:#e6edf3;--muted:#8d96a0;--line:#30363d;--field:#6e7681;--bg:#0d1117;--soft:#161b22;--accent:#a371f7;--on-accent:#0d1117;--bad:#f85149;--on-bad:#0d1117}}
.sr{position:absolute;width:1px;height:1px;padding:0;margin:-1px;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap;border:0}
*{box-sizing:border-box}body{margin:0;font:15px/1.5 system-ui,-apple-system,Segoe UI,Roboto,sans-serif;color:var(--fg);background:var(--bg)}
a{color:var(--accent);text-decoration:none}a:hover{text-decoration:underline}
header{display:flex;align-items:center;gap:16px;padding:10px 20px;border-bottom:1px solid var(--line);background:var(--soft);flex-wrap:wrap}
header .marca{font-weight:700;font-size:17px;color:var(--fg)}header nav{display:flex;gap:14px;flex:1;flex-wrap:wrap}
header .usuario{display:flex;gap:10px;align-items:center;color:var(--muted)}main{max-width:1100px;margin:0 auto;padding:20px}
.migalhas{color:var(--muted);font-size:13px;margin-bottom:8px}.migalhas a{color:var(--muted)}h1{font-size:24px;margin:8px 0 16px}h2{font-size:18px;margin:28px 0 10px;display:flex;justify-content:space-between;align-items:baseline}h2 a{font-size:13px;font-weight:400}
.aviso{padding:10px 14px;border-radius:6px;margin-bottom:14px}.aviso.ok{background:#dafbe1;color:#116329}.aviso.erro{background:#ffebe9;color:#a40e26}
@media (prefers-color-scheme:dark){.aviso.ok{background:#12261e;color:#56d364}.aviso.erro{background:#2d1117;color:#ff7b72}}
.tabela{overflow-x:auto;border:1px solid var(--line);border-radius:6px}table{width:100%;border-collapse:collapse}th,td{text-align:left;padding:8px 12px;border-bottom:1px solid var(--line);white-space:nowrap}
th{background:var(--soft);font-weight:600;font-size:13px;color:var(--muted)}tr:last-child td{border-bottom:0}
.selo{display:inline-block;padding:1px 8px;border-radius:999px;border:1px solid var(--line);font-size:12px}
.vazio{padding:24px;text-align:center;color:var(--muted);border:1px dashed var(--line);border-radius:6px}
form.caixa{border:1px solid var(--line);border-radius:6px;padding:16px;margin-top:16px;display:grid;gap:10px;max-width:640px}
form.caixa h3{margin:0 0 4px;font-size:16px}label{display:grid;gap:4px;font-size:13px;color:var(--muted)}
input,select,textarea{font:inherit;padding:7px 9px;border:1px solid var(--field);border-radius:6px;background:var(--bg);color:var(--fg);width:100%}
input[type=checkbox]{width:auto}textarea{min-height:90px}button{font:inherit;padding:7px 14px;border-radius:6px;border:1px solid var(--line);background:var(--accent);color:var(--on-accent);cursor:pointer;width:max-content}
button.perigo{background:var(--bad);color:var(--on-bad)}.acoes{display:flex;gap:8px;flex-wrap:wrap;margin:12px 0}.acoes form{display:inline}
dl{display:grid;grid-template-columns:max-content 1fr;gap:6px 16px;margin:0}dt{color:var(--muted)}dd{margin:0;overflow-wrap:anywhere}
.busca{display:flex;gap:8px;margin-bottom:12px;flex-wrap:wrap}.busca input,.busca select{width:auto;flex:1;min-width:140px}
pre{background:var(--soft);border:1px solid var(--line);border-radius:6px;padding:12px;overflow-x:auto;font:13px/1.45 ui-monospace,SFMono-Regular,Menlo,monospace}
.codigo td{padding:0 10px;border:0;font:13px/1.5 ui-monospace,monospace;white-space:pre}.codigo td.n{color:var(--muted);text-align:right;user-select:none}
.diff .add{background:#dafbe1}.diff .del{background:#ffebe9}.diff .hunk{color:var(--muted)}
@media (prefers-color-scheme:dark){.diff .add{background:#12261e}.diff .del{background:#2d1117}}
.paginas{display:flex;gap:12px;margin-top:10px}.intro{color:var(--muted);margin:-8px 0 16px}
.indicadores{display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:12px;margin:0 0 20px}.indicador{border:1px solid var(--line);border-radius:8px;padding:14px;background:var(--soft);display:flex;flex-direction:column}.indicador .valor{font-size:28px;font-weight:700}.indicador .rotulo{color:var(--muted)}
a.botao{display:inline-block;padding:7px 14px;border-radius:6px;background:var(--accent);color:var(--on-accent);text-decoration:none}
@media (max-width:640px){main{padding:12px}th,td{padding:6px 8px}}
</style></head><body>
<header><a class="marca" href="/">{{.System}}</a><nav>{{range .Nav}}<a href="{{.Href}}">{{.Text}}</a>{{end}}</nav>
<div class="usuario">{{if .User}}<span>{{.UserName}}</span><form method="post" action="/sair"><button>Sair</button></form>{{else if .Login}}<a href="/entrar">Entrar</a>{{if .Signup}}<a href="/cadastro">Criar conta</a>{{end}}{{end}}</div></header>
<main>{{if .Crumbs}}<div class="migalhas">{{range $i, $c := .Crumbs}}{{if $i}} / {{end}}<a href="{{$c.Href}}">{{$c.Text}}</a>{{end}}</div>{{end}}
{{if .Flash}}<div class="aviso ok">{{.Flash}}</div>{{end}}{{if .Error}}<div class="aviso erro">{{.Error}}</div>{{end}}
{{.Body}}</main></body></html>`)

var headingTpl = tpl(`<h1>{{.}}</h1>`)
var sectionTpl = tpl(`<h2>{{.Title}}{{if .Href}}<a href="{{.Href}}">ver tudo</a>{{end}}</h2>`)
var emptyTpl = tpl(`<div class="vazio">{{.}}</div>`)
var introTpl = tpl(`<p class="intro">{{.}}</p>`)
var actionLinkTpl = tpl(`<p class="acoes"><a class="botao" href="{{.Href}}">{{.Label}}</a></p>`)
var emptyStateTpl = tpl(`<div class="vazio" role="status">{{if .Title}}<h2>{{.Title}}</h2>{{end}}{{if .Text}}<p>{{.Text}}</p>{{end}}{{with .Action}}<p><a class="botao" href="{{.Href}}">{{.Label}}</a></p>{{end}}</div>`)
var tableTpl = tpl(`{{if .Rows}}<div class="tabela"><table>{{if .Caption}}<caption class="sr">{{.Caption}}</caption>{{end}}<thead><tr>{{range .Heads}}<th scope="col">{{.}}</th>{{end}}</tr></thead><tbody>
{{range .Rows}}<tr><td><a href="{{.Href}}">{{.Title}}</a>{{if .Body}}<div class="texto">{{.Body}}</div>{{end}}</td>{{range .Cells}}<td>{{if .Badge}}<span class="selo">{{.Text}}</span>{{else}}{{.Text}}{{end}}</td>{{end}}</tr>{{end}}
</tbody></table></div>{{else}}<div class="vazio">{{.Empty}}</div>{{end}}`)
var formTpl = tpl(`<form class="{{if .Title}}caixa{{end}}" method="post" action="{{.Action}}">{{if .Title}}<h3>{{.Title}}</h3>{{end}}
<input type="hidden" name="_csrf" value="{{.CSRF}}"><input type="hidden" name="_campos" value="1">
{{range .Inputs}}{{if eq .Type "checkbox"}}<label><span><input type="checkbox" name="{{.Name}}" value="true" {{if .Checked}}checked{{end}}> {{.Label}}</span></label>
{{else if eq .Type "textarea"}}<label>{{.Label}}<textarea name="{{.Name}}" {{if .Required}}required{{end}}>{{.Value}}</textarea></label>
{{else if eq .Type "select"}}<label>{{.Label}}<select name="{{.Name}}" {{if .Multiple}}multiple{{end}} {{if .Required}}required{{end}}>{{range .Options}}<option value="{{.Value}}" {{if .Selected}}selected{{end}}>{{.Text}}</option>{{end}}</select></label>
{{else}}<label>{{.Label}}<input type="{{.Type}}" name="{{.Name}}" value="{{.Value}}" {{if .Required}}required{{end}}></label>{{end}}{{end}}
<button {{if .Danger}}class="perigo"{{end}}>{{.Submit}}</button></form>`)
var detailTpl = tpl(`<dl>{{range .}}<dt>{{.Label}}</dt><dd>{{if .HTML}}<div class="texto">{{.HTML}}</div>{{else}}{{.Value}}{{end}}</dd>{{end}}</dl>`)
var searchTpl = tpl(`<form class="busca" method="get" role="search">{{if .Search}}<label><span class="sr">Pesquisar</span><input type="search" name="q" value="{{.Q}}" placeholder="Pesquisar"></label>{{end}}
{{range .Filters}}<label><span class="sr">{{label .}}</span><input name="{{.}}" value="{{getv $.Values .}}" placeholder="{{label .}}"></label>{{end}}<button>Filtrar</button></form>`)
var pagerTpl = tpl(`<div class="paginas">{{if .Prev}}<a href="{{.Prev}}">← anterior</a>{{end}}{{if .Next}}<a href="{{.Next}}">próxima →</a>{{end}}</div>`)
var treeTpl = tpl(`{{if .Entries}}<div class="tabela"><table><tbody>{{range .Entries}}<tr><td>{{if eq (get . "type") "tree"}}📁{{else}}📄{{end}}
<a href="{{$.Base}}/codigo?caminho={{get . "path"}}">{{get . "name"}}</a></td></tr>{{end}}</tbody></table></div>{{else}}<div class="vazio">Pasta vazia.</div>{{end}}`)
var commitsTpl = tpl(`{{if .Commits}}<h2>Commits recentes</h2><div class="tabela"><table><tbody>{{range .Commits}}<tr><td><a href="{{$.Base}}/commit/{{get . "id"}}">{{get . "title"}}</a></td><td>{{get . "author_name"}}</td><td><code>{{get . "short_id"}}</code></td></tr>{{end}}</tbody></table></div>{{end}}`)
var codeTpl = tpl(`{{if .Binary}}<div class="vazio">Arquivo binário.</div>{{else}}<div class="tabela"><table class="codigo"><tbody>{{range .Lines}}<tr><td class="n">{{.N}}</td><td>{{.Text}}</td></tr>{{end}}</tbody></table></div>{{end}}`)
var diffTpl = tpl(`{{range .}}<h3>{{.Path}}</h3><pre class="diff">{{range .Lines}}<div class="{{.Class}}">{{.Text}}</div>{{end}}</pre>{{else}}<div class="vazio">Sem mudanças.</div>{{end}}`)
var logTpl = tpl(`{{if .}}<pre>{{.}}</pre>{{else}}<div class="vazio">Sem saída ainda.</div>{{end}}`)
var loginTpl = tpl(`<form class="caixa" method="post" action="/entrar"><h3>Entrar</h3><label>{{.Label}}<input name="login" required autofocus></label>
<label>Senha<input type="password" name="senha" required></label><button>Entrar</button>{{if .Signup}}<a href="/cadastro">Criar conta</a>{{end}}{{if .Recovery}}<a href="/esqueci">Esqueci minha senha</a>{{end}}</form>`)
var forgotTpl = tpl(`<form class="caixa" method="post" action="/esqueci"><h3>Esqueci minha senha</h3><label>{{.Label}}<input name="login" required autofocus></label><button>Enviar link</button><a href="/entrar">Voltar</a></form>`)
var resetTpl = tpl(`<form class="caixa" method="post" action="/redefinir"><h3>Criar senha nova</h3><input type="hidden" name="token" value="{{.Token}}"><label>Senha nova<input type="password" name="senha" required autofocus autocomplete="new-password"></label><button>Salvar senha</button></form>`)

// nameOf shows a referenced record by its title (people by name).
func (ps *pageSite) nameOf(ctx *interp.Context, entity string, id any) string {
	e := ps.a.app.Entities[entity]
	if e == nil {
		return display(id)
	}
	res, err := ps.a.in.Op(ctx, e.Singular, "buscar", id)
	row, _ := res.(map[string]any)
	if err != nil || row == nil {
		return display(id)
	}
	return titleOf(e, row)
}
