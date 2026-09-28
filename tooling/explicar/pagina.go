package explicar

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/diagnostics"
	"github.com/flaviokalleu/germanio/compiler/parser"
)

// Pagina explains a page of the intent layer: what it shows and, for each
// section (docs/INTENCAO.md › Página), whether it was declared or comes from
// the default, and who sees each action.
func Pagina(prog *ast.Program, nome string) (string, error) {
	app := prog.App
	if app == nil {
		return "", fmt.Errorf("o programa não tem páginas de intenção")
	}
	var pg *ast.PageDecl
	var names []string
	for _, p := range app.Pages {
		names = append(names, p.Name)
		if strings.EqualFold(p.Name, nome) {
			pg = p
		}
	}
	if pg == nil {
		return "", fmt.Errorf("não conheço a página %q. Páginas: %s", nome, strings.Join(names, ", "))
	}
	e := app.Entities[pg.Show]
	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	w("Página %s (%s)\n\n", pg.Name, where(pg.Pos))
	if len(pg.Indicators) > 0 {
		w("Indicadores (cada número conta só o que quem vê a página pode ver; o administrador vê o total):\n")
		for _, ind := range pg.Indicators {
			what := ind.Entity
			if ie := app.Entities[ind.Entity]; ie != nil {
				what = ie.Plural
			}
			if ind.State != "" {
				what += " no estado " + ind.State
			}
			w("  %q — conta %s (%s)\n", ind.Label, what, where(ind.Pos))
		}
		w("\n")
	}
	if e == nil {
		if len(pg.Indicators) > 0 {
			w("Um painel: só indicadores, sem lista nem ações.\n")
		}
		return b.String(), nil
	}
	w("Mostra: %s\n", e.Plural)
	who := func(verb string) string {
		var out []string
		for _, r := range e.Rules[parser.CanonVerb(verb)] {
			out = append(out, describeRule(app, r))
		}
		if len(out) == 0 {
			return "só o administrador"
		}
		return strings.Join(unique(out), "; ")
	}
	origin := func(declared bool) string {
		if declared {
			return "declarado"
		}
		return "padrão"
	}
	title := e.Label + "s"
	if pg.Title != "" {
		title = pg.Title
	}
	w("\nTopo:\n  título  %q (%s)\n", title, origin(pg.Title != ""))
	if pg.Text != "" {
		w("  texto   %q (declarado)\n", pg.Text)
	}
	for _, a := range pg.Actions {
		label := a.Label
		if label == "" {
			label = "Novo " + strings.ToLower(e.Label)
		}
		w("  ação    %s %q — aparece para: %s (%s)\n", a.Verb, label, who(a.Verb), where(a.Pos))
	}
	if len(pg.Actions) == 0 {
		w("  ação    criar — aparece para quem pode criar: %s (padrão)\n", who("criar"))
	}
	w("\nFiltros (%s):\n", origin(len(pg.Filters) > 0))
	filters := pg.Filters
	if len(filters) == 0 {
		if len(e.Search) > 0 {
			filters = append(filters, "pesquisar")
		}
		filters = append(filters, e.Filters...)
	}
	for _, f := range filters {
		w("  %s\n", f)
	}
	w("\nColunas (%s):\n", origin(len(pg.Columns) > 0))
	cols := pg.Columns
	if len(cols) == 0 {
		for _, f := range e.Model.Fields {
			if !f.Hidden && !f.System && !f.IsSecret() && !f.Private {
				cols = append(cols, strings.ToLower(f.Name))
			}
		}
	}
	for _, c := range cols {
		w("  %s\n", c)
	}
	w("\nQuando não há nada:\n")
	if pg.Empty == nil {
		w("  \"Nenhum registro de %s ainda.\" (padrão)\n", strings.ToLower(e.Label))
	} else {
		if pg.Empty.Title != "" {
			w("  título  %q\n", pg.Empty.Title)
		}
		if pg.Empty.Text != "" {
			w("  texto   %q\n", pg.Empty.Text)
		}
		if a := pg.Empty.Action; a != nil {
			w("  ação    %s %q — aparece para: %s\n", a.Verb, a.Label, who(a.Verb))
		}
	}
	if pg.PerPage > 0 {
		w("\nPor página: %d\n", pg.PerPage)
	}
	return b.String(), nil
}

func where(pos diagnostics.Position) string {
	if pos.File == "" {
		return fmt.Sprintf("linha %d", pos.Line)
	}
	return fmt.Sprintf("%s:%d", filepath.Base(pos.File), pos.Line)
}
