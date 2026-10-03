package servidor

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Actions that need a choice from the person get a short form on the
// record's page instead of a bare button. Nothing here is declared by the
// author: the forms are derived from capabilities the data already has, and
// every form posts to the same page action, which calls the same internal
// operation as the API (same rules, CSRF, validation and transaction).
//   - merging a proposal (GEP 0027): squash or not and, when the record with
//     the repository runs executions, "merge when they pass";
//   - copying (GEP 0029): the values that place the copy (its name, its
//     path, what must be unique), so a copy next to its original does not
//     fail on the same path;
//   - moving to another parent (GEP 0034): only the parents where the person
//     may create such a record;
//   - how many approvals are still missing before an action (GEP 0026).

// formActions: verbs drawn as forms by actionForms, not as plain buttons.
func formActions(e *ast.Entity, verb string) bool {
	switch verb {
	case "mesclar":
		return e.Review != nil && e.Review.Target != "" && e.Transitions["mesclar"] != nil
	case "copiar":
		return e.Copies
	}
	return false
}

type actionForm struct {
	Action, CSRF, Title, Submit, Hint string
	Inputs                            []input
	// Extra: a second submit button that sends Name=true (merge when the
	// executions pass), and its label.
	ExtraName, ExtraLabel string
}

var actionFormTpl = tpl(`<form class="caixa" method="post" action="{{.Action}}"><h3>{{.Title}}</h3>
<input type="hidden" name="_csrf" value="{{.CSRF}}"><input type="hidden" name="_campos" value="1">{{if .Hint}}<p class="intro">{{.Hint}}</p>{{end}}
{{range .Inputs}}{{if eq .Type "checkbox"}}<label><span><input type="checkbox" name="{{.Name}}" value="true" {{if .Checked}}checked{{end}}> {{.Label}}</span></label>
{{else if eq .Type "select"}}<label>{{.Label}}<select name="{{.Name}}" required>{{range .Options}}<option value="{{.Value}}" {{if .Selected}}selected{{end}}>{{.Text}}</option>{{end}}</select></label>
{{else}}<label>{{.Label}}<input type="{{.Type}}" name="{{.Name}}" value="{{.Value}}" {{if .Required}}required{{end}}></label>{{end}}{{end}}
<div class="acoes"><button>{{.Submit}}</button>{{if .ExtraName}}<button name="{{.ExtraName}}" value="true">{{.ExtraLabel}}</button>{{end}}</div></form>`)

// actionForms draws, for the record's page, the actions that ask something.
func (ps *pageSite) actionForms(r *http.Request, ctx *interp.Context, atual map[string]any, chain []step, e *ast.Entity, record map[string]any, base, csrf string) template.HTML {
	if atual == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(string(ps.approvalNotice(e, record)))
	if formActions(e, "mesclar") && ps.available(ctx, atual, e, "mesclar", record) {
		b.WriteString(string(ps.mergeForm(e, record, base, csrf)))
	}
	if formActions(e, "copiar") && ps.available(ctx, atual, e, "copiar", record) && (!e.Repository || ps.a.in.Can(ctx, atual, e, "baixar_codigo", record)) {
		b.WriteString(string(ps.copyForm(r, chain, e, record, base, csrf)))
	}
	if e.MoveField != "" {
		b.WriteString(string(ps.moveForm(r, ctx, atual, e, record, base, csrf)))
	}
	return template.HTML(b.String())
}

// approvalNotice: how many approvals each waiting action still needs, next
// to where its button would be (the button only appears once they are all
// there). It is what the record shows as aprovacoes_faltando.
func (ps *pageSite) approvalNotice(e *ast.Entity, record map[string]any) template.HTML {
	if len(e.ApprovalsNeeded) == 0 || toStr(record[e.StateField]) != e.Initial {
		return ""
	}
	var b strings.Builder
	for _, verb := range sortedKeys(e.ApprovalsNeeded) {
		need, have := approvals(e, verb, record)
		if have >= need {
			continue
		}
		missing := need - have
		word := map[bool]string{true: "aprovação", false: "aprovações"}
		verbWord := map[bool]string{true: "Falta", false: "Faltam"}
		fmt.Fprintf(&b, `<p class="aviso" data-aprovacoes-faltando="%d">%s %d %s para %s: tem %d de %d (a aprovação do autor não conta).</p>`,
			missing, verbWord[missing == 1], missing, word[missing == 1], template.HTMLEscapeString(strings.ToLower(label(verb))), have, need)
	}
	return template.HTML(b.String())
}

func sortedKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// mergeForm: merge now (squash or not) or, when the record with the
// repository runs executions, merge when the latest one passes.
func (ps *pageSite) mergeForm(e *ast.Entity, record map[string]any, base, csrf string) template.HTML {
	f := actionForm{Action: base + "/acao/mesclar", CSRF: csrf, Title: label("mesclar"), Submit: label("mesclar"),
		Inputs: []input{{Name: "juntar_commits", Label: "Juntar os commits em um só", Type: "checkbox", Checked: truthy(record["juntar_commits"])}}}
	if e.Review.Runs != "" && !truthy(record["mesclar_quando_passar"]) {
		f.ExtraName, f.ExtraLabel = "mesclar_quando_passar", "Mesclar quando passar"
		f.Hint = "Mesclar quando passar espera a última execução de " + toStr(record[e.Review.Source]) + " e mescla se ela passar."
	}
	return htmlOf(actionFormTpl, f)
}

// copyFields: the values of a copy the page asks for — what places it or
// must be unique (its path, a unique name) and its title — among the values
// a copy takes.
func copyFields(e *ast.Entity) map[string]bool {
	copied := copiedValues(e, allFieldsPresent(e))
	out := map[string]bool{}
	for _, f := range e.Model.Fields {
		key := strings.ToLower(f.Name)
		if _, ok := copied[key]; !ok {
			continue
		}
		if f.Unique || (e.Address != nil && key == e.Address.Segment) {
			out[key] = true
		}
	}
	for _, k := range []string{"titulo", "nome", "name", "title"} {
		if _, ok := copied[k]; ok {
			out[k] = true
			break
		}
	}
	return out
}

// allFieldsPresent: a record with every field set, to ask copiedValues
// which fields a copy takes.
func allFieldsPresent(e *ast.Entity) map[string]any {
	row := map[string]any{}
	for _, f := range e.Model.Fields {
		row[strings.ToLower(f.Name)] = true
	}
	return row
}

// copyForm: the copy's name and path, filled with the original's. A value
// that must be unique everywhere starts empty, since the original's would
// always be refused.
func (ps *pageSite) copyForm(r *http.Request, chain []step, e *ast.Entity, record map[string]any, base, csrf string) template.HTML {
	wanted := copyFields(e)
	var ins []input
	var unique []string
	for _, in := range ps.inputs(r, chain, e, nil, nil) {
		if !wanted[in.Name] {
			continue
		}
		var field *ast.Field
		for _, f := range e.Model.Fields {
			if strings.ToLower(f.Name) == in.Name {
				field = f
			}
		}
		in.Required = true
		if field != nil && field.Unique {
			in.Value = ""
		} else if v := record[in.Name]; v != nil {
			in.Value = display(v)
		}
		if field != nil && (field.Unique || e.Address != nil && in.Name == e.Address.Segment) {
			unique = append(unique, strings.ToLower(in.Label))
		}
		ins = append(ins, in)
	}
	f := actionForm{Action: base + "/acao/copiar", CSRF: csrf, Title: label("copiar"), Submit: label("copiar"), Inputs: ins}
	if len(unique) > 0 {
		f.Hint = "A cópia é sua. Se ela ficar no mesmo lugar do original, mude: " + strings.Join(unique, " e ") + "."
	}
	return htmlOf(actionFormTpl, f)
}

// moveForm: the parents the record may move to — the ones the person sees
// and may create such a record in, not read-only, not where it already is.
// The server checks everything again when the form is sent.
func (ps *pageSite) moveForm(r *http.Request, ctx *interp.Context, atual map[string]any, e *ast.Entity, record map[string]any, base, csrf string) template.HTML {
	if !ps.a.in.Can(ctx, atual, e, "editar", record) || ps.a.frozenFor(ctx, "acao", e, record, nil) != nil {
		return ""
	}
	field := e.MoveField
	pe := ps.a.app.Entities[e.Parents[field]]
	if pe == nil {
		return ""
	}
	code, out, _ := ps.call(r, "GET", "/_ge/api/"+pe.Plural+"?per_page=100", nil)
	if code != 200 {
		return ""
	}
	var opts []option
	for _, it := range asList(out) {
		dest, _ := it.(map[string]any)
		if dest == nil || toStr(dest["id"]) == toStr(record[field]) {
			continue
		}
		data := map[string]any{}
		for k, v := range record {
			data[k] = v
		}
		delete(data, "id")
		data[field] = dest["id"]
		if !ps.a.canCreate(ctx, atual, e, data) || ps.a.frozenFor(ctx, "criar", e, data, data) != nil {
			continue
		}
		opts = append(opts, option{Value: display(dest["id"]), Text: titleOf(pe, dest)})
	}
	if len(opts) == 0 {
		return ""
	}
	title := "Mudar de " + strings.ToLower(pe.Label)
	return htmlOf(actionFormTpl, actionForm{Action: base + "/acao/mudar", CSRF: csrf, Title: title, Submit: "Mudar",
		Hint:   fmt.Sprintf("%s ganha o próximo número do destino; o que é do lugar atual (como etiquetas que o destino não tem) não vai junto.", e.Label),
		Inputs: []input{{Name: field, Label: "Para " + strings.ToLower(pe.Label), Type: "select", Options: opts}}})
}

// actionData: the values a page action sends with it. Only actions that
// ask something send values, and only the ones they accept.
func (ps *pageSite) actionData(e *ast.Entity, verb string, form url.Values) map[string]any {
	switch {
	case formActions(e, "mesclar") && verb == "mesclar":
		out := map[string]any{}
		if form.Get("_campos") != "" {
			out["juntar_commits"] = form.Get("juntar_commits") == "true"
		}
		if form.Get("mesclar_quando_passar") == "true" {
			out["mesclar_quando_passar"] = true
		}
		return out
	case verb == "mudar" && e.MoveField != "":
		out := map[string]any{}
		if v := strings.TrimSpace(form.Get(e.MoveField)); v != "" {
			out[e.MoveField] = v
			if n, ok := asNumberOK(v); ok {
				out[e.MoveField] = n
			}
		}
		return out
	case formActions(e, "copiar") && verb == "copiar":
		// only what the form asked: the rest of the copy comes from the
		// original
		out := map[string]any{}
		for k, v := range ps.formData(e, form) {
			if _, sent := form[k]; sent {
				out[k] = v
			}
		}
		return out
	}
	return nil
}

func asNumberOK(s string) (float64, bool) {
	n, err := strconv.ParseFloat(s, 64)
	return n, err == nil
}

// movedTarget: where the page goes after a move. When the old parent is the
// page's own data, the record's new address; otherwise the old parent.
func movedTarget(pg *ast.PageDecl, chain []step, body []string, e *ast.Entity, out any) string {
	row, _ := out.(map[string]any)
	root := "/" + slug(pg.Name)
	if row != nil && len(chain) == 2 && row[e.MoveField] != nil {
		ref := display(row["id"])
		if n, ok := row["numero"]; ok && n != nil {
			ref = display(n)
		}
		return root + "/" + url.PathEscape(display(row[e.MoveField])) + "/" + url.PathEscape(e.Plural) + "/" + url.PathEscape(ref)
	}
	if len(body) >= 2 {
		return root + "/" + strings.Join(escapeAll(body[:len(body)-2]), "/")
	}
	return root
}
