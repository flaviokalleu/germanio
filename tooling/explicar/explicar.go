// Package explicar shows what Germanio inferred from intent phrases, so a
// simple .ge never becomes a black box (ge explain <dado>, ge check).
package explicar

import (
	"fmt"
	"sort"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/parser"
)

func find(app *ast.App, name string) *ast.Entity {
	name = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", "_"))
	if e, ok := app.Entities[name]; ok {
		return e
	}
	for _, e := range app.Entities {
		if e.Plural == name || e.Singular == parser.Singular(name) {
			return e
		}
	}
	return nil
}

func describeField(f *ast.Field) string {
	var parts []string
	t := string(f.Type)
	if f.Type == ast.FieldLista {
		t = "lista de " + f.ListOf
	}
	if f.Reference != "" {
		t = "referência a " + f.Reference
	}
	parts = append(parts, t)
	if f.Required {
		parts = append(parts, "obrigatório")
	}
	if f.Unique {
		parts = append(parts, "único")
	}
	if f.Max != nil {
		parts = append(parts, fmt.Sprintf("até %v", *f.Max))
	}
	if f.Min != nil {
		parts = append(parts, fmt.Sprintf("mínimo %v", *f.Min))
	}
	if f.HasDefault {
		parts = append(parts, fmt.Sprintf("começa com %v", f.DefaultValue))
	}
	if f.Private {
		parts = append(parts, "privado (só dono e administradores veem)")
	}
	if f.Hidden {
		parts = append(parts, "oculto")
	}
	if f.IsSecret() {
		if f.Type == ast.FieldSegredo {
			parts = append(parts, "segredo gerado, guardado como hash, mostrado só na criação")
		} else {
			parts = append(parts, "guardada com bcrypt, nunca devolvida")
		}
	}
	if f.NumberedBy != "" {
		parts = append(parts, "numerado 1, 2, 3… por "+strings.TrimSuffix(f.NumberedBy, "_id"))
	}
	if f.Validator != "" {
		parts = append(parts, "valida "+f.Validator)
	}
	if f.System {
		parts = append(parts, "mantido pelo sistema")
	}
	return strings.Join(parts, ", ")
}

func describeRule(app *ast.App, r *ast.AccessRule) string {
	who := ""
	switch {
	case r.Anyone:
		who = "qualquer visitante"
	case r.MinRole == "administrador":
		who = "administrador"
	case r.MinRole != "":
		who = r.MinRole + " ou superior"
	case r.SignedIn:
		who = "qualquer pessoa conectada"
	}
	if r.Own {
		if who == "qualquer pessoa conectada" {
			who = "o dono/autor"
		} else {
			who += " (só os próprios)"
		}
	}
	return who
}

// Entidade explains one kind of data.
func Entidade(prog *ast.Program, nome string) (string, error) {
	app := prog.App
	if app == nil {
		return "", fmt.Errorf("este programa não usa a camada de intenção (tenha …)")
	}
	e := find(app, nome)
	if e == nil {
		var names []string
		for _, n := range app.Order {
			names = append(names, app.Entities[n].Singular)
		}
		sort.Strings(names)
		return "", fmt.Errorf("não conheço %q. Dados: %s", nome, strings.Join(names, ", "))
	}
	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	w("%s (%s)\n\n", e.Label, e.Plural)
	w("Persistência: automática, tabela %q (id, criado_em, atualizado_em)\n", e.Singular)
	if app.LoginEntity == e.Singular {
		w("Login: são as pessoas que entram no sistema (%s)\n", strings.Join(app.Login.Fields, " ou "))
	}
	w("\nCampos:\n")
	for _, f := range e.Model.Fields {
		w("  %-22s %s\n", f.Name, describeField(f))
	}
	w("\nRelações:\n")
	var parents []string
	for field, t := range e.Parents {
		parents = append(parents, fmt.Sprintf("  pertence a %s (%s)", t, field))
	}
	sort.Strings(parents)
	for _, p := range parents {
		w("%s\n", p)
	}
	for _, c := range e.Children {
		w("  tem %s\n", app.Entities[c].Plural)
	}
	if e.HierarchyField != "" {
		w("  tem sub%s (%s)\n", e.Plural, e.HierarchyField)
	}
	if e.HasMembers {
		w("  tem membros com papel (%s)\n", app.MemberModel)
	}
	if e.InheritVia != "" {
		w("  herda membros de %s\n", e.Parents[e.InheritVia])
	}
	if e.Repository {
		w("  tem repositório Git em /<%s>.git\n", e.RepoKey)
	}
	if e.StateField != "" {
		w("\nEstados: começa %s\n", e.Initial)
		var verbs []string
		for v := range e.Transitions {
			verbs = append(verbs, v)
		}
		sort.Strings(verbs)
		for _, v := range verbs {
			tr := e.Transitions[v]
			stamp := ""
			if tr.Stamp {
				stamp = fmt.Sprintf(" (registra %s_em e %s_por_id)", tr.Target, tr.Target)
			}
			w("  %s → %s%s\n", v, tr.Target, stamp)
		}
	}
	if e.Visibility != "" {
		w("\nVisibilidade (%s): public = todos veem; internal = quem está conectado; private = membros\n", e.Visibility)
	}
	for _, f := range e.CeilingFields {
		w("  nunca mais visível que %s\n", e.Parents[f])
	}
	for _, rs := range e.Restrictions {
		var who []string
		for _, o := range rs.Owners {
			who = append(who, strings.TrimSuffix(o, "_id"))
		}
		who = append(who, rs.Lists...)
		if rs.MinRole != "" {
			who = append(who, rs.MinRole+" ou superior")
		}
		w("  quando %s: só %s (e administradores) veem\n", rs.Flag, strings.Join(who, ", "))
	}
	w("\nQuem pode:\n")
	var verbs []string
	for v := range e.Rules {
		verbs = append(verbs, v)
	}
	sort.Strings(verbs)
	for _, v := range verbs {
		var who []string
		for _, r := range e.Rules[v] {
			who = append(who, describeRule(app, r))
		}
		w("  %-14s %s\n", strings.ReplaceAll(v, "_", " "), strings.Join(unique(who), "; "))
	}
	if len(e.Rules["ver"]) == 0 {
		w("  %-14s quem pode ver %s\n", "ver", firstParent(app, e))
	}
	w("  %-14s tudo\n", "administrador")
	if e.ProtectedBranchRole != "" {
		w("  branch padrão  só %s ou superior envia código direto\n", e.ProtectedBranchRole)
	}
	if e.CreatorRole != "" {
		w("  quem cria vira %s\n", e.CreatorRole)
	}
	if len(e.Search) > 0 || len(e.Filters) > 0 {
		w("\nPesquisa: %s\nFiltros: %s\n", strings.Join(e.Search, ", "), strings.Join(e.Filters, ", "))
	}
	if len(e.Hooks) > 0 {
		w("\nRegras explícitas:\n")
		var hs []string
		for k := range e.Hooks {
			hs = append(hs, k)
		}
		sort.Strings(hs)
		for _, k := range hs {
			h := e.Hooks[k]
			phase := "quando"
			if h.Before {
				phase = "antes de"
			}
			w("  %s %s %s (%s:%d)\n", phase, h.Verb, e.Singular, h.Pos.File, h.Pos.Line)
		}
	}
	w("\nOperações (páginas): /_ge/api/%s\n", e.Plural)
	if e.Integrate != "" {
		w("Integração: %s/%s", app.Integration, e.Integrate)
		if len(app.Vocabulary) > 0 {
			w(" (nomes externos pelo vocabulário da integração)")
		}
		w("\n")
	}
	return b.String(), nil
}

func firstParent(app *ast.App, e *ast.Entity) string {
	for _, t := range e.Parents {
		if t != app.LoginEntity {
			return t
		}
	}
	return "(ninguém além do administrador)"
}

func unique(list []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range list {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// Verificar lists warnings about the intent: data nobody can reach and
// actions that fall back to defaults.
func Verificar(prog *ast.Program) []string {
	app := prog.App
	if app == nil {
		return nil
	}
	var warn []string
	for _, n := range app.Order {
		e := app.Entities[n]
		if len(e.Rules) == 0 && e.Integrate == "" && e.Singular != app.MemberModel && !ast_inheritsView(app, e) {
			warn = append(warn, fmt.Sprintf("%s: ninguém além do administrador pode usar (falta permita ou pode)", e.Plural))
		}
		for v := range e.Transitions {
			if len(e.Rules[v]) == 0 {
				warn = append(warn, fmt.Sprintf("%s: %s segue quem pode editar (nenhum \"pode %s\" declarado)", e.Plural, v, v))
			}
		}
	}
	sort.Strings(warn)
	return warn
}

func ast_inheritsView(app *ast.App, e *ast.Entity) bool {
	for _, t := range e.Parents {
		if t != app.LoginEntity {
			return true
		}
	}
	return false
}
