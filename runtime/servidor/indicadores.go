package servidor

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/flaviokalleu/germanio/compiler/ast"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Indicators (docs/gep/0012-indicadores.md, em teste): counts on a page. A
// number counts only what the viewer could list — the list's own rules,
// record by record when visibility depends on each record.

// countVisible counts the records of e matching filters that atual may see;
// ok=false: atual may not see this data at all (the indicator is hidden).
func (a *intentAPI) countVisible(ctx *interp.Context, atual map[string]any, e *ast.Entity, filters map[string]any) (int, bool) {
	if a.in.IsAdmin(atual) {
		n, err := a.in.Op(ctx, e.Singular, "contar", filters)
		if err != nil {
			return 0, false
		}
		return int(asNumber(n)), true
	}
	if !interp.RecordDependent(e) && !interp.InheritsView(a.app, e) {
		if !a.in.Can(ctx, atual, e, "ver", nil) {
			return 0, false
		}
		n, err := a.in.Op(ctx, e.Singular, "contar", filters)
		if err != nil {
			return 0, false
		}
		return int(asNumber(n)), true
	}
	if atual == nil && e.Visibility == "" && !anyoneMay(e) && !interp.InheritsView(a.app, e) && !a.visibleThroughParents(e) {
		return 0, false
	}
	a.narrowVisible(ctx, atual, e, filters)
	interp.BeginReadCache(ctx)
	defer interp.EndReadCache(ctx)
	opts := map[string]any{}
	if groups, ok := a.visibilityPrefilter(ctx, atual, e); ok {
		opts["ou"] = groups
	}
	total := 0
	for batch := 1; ; batch++ {
		opts["limite"], opts["pagina"] = 500, batch
		res, err := a.in.Op(ctx, e.Singular, "filtrar", filters, opts)
		if err != nil {
			return 0, false
		}
		rows := res.([]any)
		for _, it := range rows {
			if a.in.Can(ctx, atual, e, "ver", it.(map[string]any)) {
				total++
			}
		}
		if len(rows) < 500 {
			return total, true
		}
	}
}

type indicatorView struct {
	Label string
	Value int
}

// indicators renders a page's numbers for this viewer.
func (ps *pageSite) indicators(ctx *interp.Context, atual map[string]any, pg *ast.PageDecl) template.HTML {
	var out []indicatorView
	for _, ind := range pg.Indicators {
		e := ps.a.app.Entities[ind.Entity]
		if e == nil {
			continue
		}
		filters := map[string]any{}
		if ind.State != "" && e.StateField != "" {
			filters[e.StateField] = ind.State
		}
		if n, ok := ps.a.countVisible(ctx, atual, e, filters); ok {
			out = append(out, indicatorView{ind.Label, n})
		}
	}
	if len(out) == 0 {
		return ""
	}
	return htmlOf(indicatorsTpl, out)
}

// dashboard is a page made only of indicators.
func (ps *pageSite) dashboard(w http.ResponseWriter, r *http.Request, pg *ast.PageDecl) {
	v := ps.base(r, pg.Name)
	if pg.Title != "" {
		v.Title = pg.Title
	}
	ctx := &interp.Context{Request: r}
	atual, _ := ps.a.s.identify(ctx, r)
	var body bytes.Buffer
	body.WriteString(string(htmlOf(headingTpl, v.Title)))
	if pg.Text != "" {
		body.WriteString(string(htmlOf(introTpl, pg.Text)))
	}
	if html := ps.indicators(ctx, atual, pg); html != "" {
		body.WriteString(string(html))
	} else if atual == nil && ps.a.app.Login != nil {
		http.Redirect(w, r, "/entrar", http.StatusSeeOther)
		return
	} else {
		body.WriteString(string(htmlOf(emptyTpl, "Nada para mostrar aqui.")))
	}
	v.Body = template.HTML(body.String())
	ps.render(w, v, http.StatusOK)
}

var indicatorsTpl = tpl(`<section class="indicadores" aria-label="Indicadores">{{range .}}<div class="indicador"><span class="valor">{{.Value}}</span><span class="rotulo">{{.Label}}</span></div>{{end}}</section>`)
