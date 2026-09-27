package parser

import (
	"fmt"
	"sort"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/diagnostics"
	"github.com/flaviokalleu/germanio/compiler/lexer"
)

// Singular derives the singular of a Portuguese (or English) plural with a
// fixed, documented table. Multi-word names singularize the head word:
// tokens_de_acesso → token_de_acesso, merge_requests → merge_request.
func Singular(name string) string {
	parts := strings.Split(name, "_")
	head := len(parts) - 1
	for i, w := range parts {
		if (w == "de" || w == "do" || w == "da") && i > 0 {
			head = 0
			break
		}
	}
	parts[head] = singularWord(parts[head])
	return strings.Join(parts, "_")
}

// altSingular is the plain "drop the s" form of the head word.
func altSingular(name string) string {
	parts := strings.Split(name, "_")
	head := len(parts) - 1
	for i, w := range parts {
		if (w == "de" || w == "do" || w == "da") && i > 0 {
			head = 0
			break
		}
	}
	parts[head] = strings.TrimSuffix(parts[head], "s")
	return strings.Join(parts, "_")
}

func singularWord(w string) string {
	switch {
	case len(w) <= 3:
		return strings.TrimSuffix(w, "s")
	case strings.HasSuffix(w, "oes"), strings.HasSuffix(w, "aes"), strings.HasSuffix(w, "aos"):
		return w[:len(w)-3] + "ao"
	case strings.HasSuffix(w, "eis"):
		return w[:len(w)-3] + "el"
	case strings.HasSuffix(w, "ais"), strings.HasSuffix(w, "ois"), strings.HasSuffix(w, "uis"):
		return w[:len(w)-2] + "l"
	case strings.HasSuffix(w, "ns"):
		return w[:len(w)-2] + "m"
	case strings.HasSuffix(w, "res"), strings.HasSuffix(w, "zes"), strings.HasSuffix(w, "ses"):
		return w[:len(w)-2]
	case strings.HasSuffix(w, "s"):
		return w[:len(w)-1]
	}
	return w
}

// CanonVerb maps synonyms to the operation they mean.
func CanonVerb(v string) string {
	switch v {
	case "ver", "listar", "mostrar", "visualizar", "acessar", "consultar":
		return "ver"
	case "criar", "cadastrar", "adicionar", "incluir":
		return "criar"
	case "editar", "alterar", "atualizar", "mudar":
		return "editar"
	case "excluir", "remover", "apagar", "deletar":
		return "excluir"
	case "pesquisar", "buscar", "procurar":
		return "pesquisar"
	case "administrar", "gerenciar":
		return "administrar"
	}
	return v
}

type resolver struct {
	app    *ast.App
	byName map[string]*ast.Entity // plural and singular → entity
	file   string
}

func (r *resolver) errAt(pos diagnostics.Position, format string, a ...any) error {
	return fmt.Errorf("%s:%d: %s", pos.File, pos.Line, fmt.Sprintf(format, a...))
}

func (r *resolver) entity(name string, pos diagnostics.Position) (*ast.Entity, error) {
	if e, ok := r.byName[name]; ok {
		return e, nil
	}
	var names []string
	for _, n := range r.app.Order {
		names = append(names, r.app.Entities[n].Plural)
	}
	sort.Strings(names)
	return nil, r.errAt(pos, "não conheço %q. Declare com: tenha %s (dados declarados: %s)", name, name, strings.Join(names, ", "))
}

func titleCase(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// ResolveIntent turns the intent phrases of a merged program into models
// and the authorization table (program.App). Programs without intent are
// left untouched.
func ResolveIntent(prog *ast.Program) error {
	in := prog.Intent
	if in == nil {
		return nil
	}
	app := &ast.App{Entities: map[string]*ast.Entity{}, Roles: in.Roles, Login: in.Login, Integration: in.IntegrationPrefix, Pages: in.Pages, Init: in.Init, Messages: "pt"}
	if app.Integration == "" {
		app.Integration = "/api"
	}
	if in.Messages != "" {
		app.Messages = in.Messages
	}
	app.Vocabulary = in.Vocabulary
	r := &resolver{app: app, byName: map[string]*ast.Entity{}}
	prog.App = app

	// 1. Entities
	for _, d := range in.Entities {
		sing := d.Singular
		if sing == "" {
			sing = Singular(d.Name)
		}
		if _, dup := r.byName[d.Name]; dup {
			return r.errAt(d.Pos, "%q declarado mais de uma vez", d.Name)
		}
		if _, dup := r.byName[sing]; dup {
			return r.errAt(d.Pos, "%q conflita com outro dado de mesmo singular %q", d.Name, sing)
		}
		label := d.Label
		if label == "" {
			label = titleCase(sing)
		}
		e := &ast.Entity{Plural: d.Name, Singular: sing, Label: label, Parents: map[string]string{},
			Model: &ast.Model{Name: sing, Internal: true, Label: label, Pos: d.Pos},
			Rules: map[string][]*ast.AccessRule{}, Hooks: map[string]*ast.Hook{}}
		app.Entities[sing] = e
		app.Order = append(app.Order, sing)
		r.byName[d.Name] = e
		r.byName[sing] = e
		// Alternative singular ("tokens" → "token" besides "tokem"): the form
		// used in `cada <singular> tem` decides which one names the data.
		if alt := altSingular(d.Name); alt != sing {
			if _, taken := r.byName[alt]; !taken {
				r.byName[alt] = e
			}
		}
	}
	for _, b := range in.FieldBlocks {
		if e := r.byName[b.Entity]; e != nil && b.Entity != e.Singular && b.Entity != e.Plural {
			delete(app.Entities, e.Singular)
			for i, n := range app.Order {
				if n == e.Singular {
					app.Order[i] = b.Entity
				}
			}
			e.Singular, e.Model.Name = b.Entity, b.Entity
			if e.Label == titleCase(Singular(e.Plural)) {
				e.Label, e.Model.Label = titleCase(b.Entity), titleCase(b.Entity)
			}
			app.Entities[b.Entity] = e
		}
	}

	// 2. Field blocks: each line is a relation (names another entity,
	// "membros com papel", "sub<plural>"), a person (autor, responsaveis)
	// or a field.
	fp := &Parser{}
	type pendingTem struct {
		owner, child *ast.Entity
		pos          diagnostics.Position
	}
	type pendingPerson struct {
		e    *ast.Entity
		name string
		many bool
		pos  diagnostics.Position
	}
	var tems []pendingTem
	var people []pendingPerson
	for _, b := range in.FieldBlocks {
		e, err := r.entity(b.Entity, b.Pos)
		if err != nil {
			return err
		}
		for _, line := range b.Lines {
			w := wordsOf(line)
			joined, _ := phrase(w)
			switch {
			case len(w) >= 3 && w[len(w)-2] == "com" && w[len(w)-1] == "papel":
				member, _ := phrase(w[:len(w)-2])
				if err := r.membership(e, member, b.Pos); err != nil {
					return err
				}
			case joined == "repositorio" || joined == "repositorio_git":
				e.Repository = true
				e.Model.Fields = append(e.Model.Fields, &ast.Field{Name: "repositorio", Type: ast.FieldTexto, Hidden: true, Pos: b.Pos})
			case joined == "sub"+e.Plural || joined == "sub_"+e.Plural:
				e.HierarchyField = "pai_id"
				e.Model.Fields = append(e.Model.Fields, &ast.Field{Name: "pai_id", Type: ast.FieldInteiro, Reference: e.Singular, Index: true, Pos: b.Pos})
				e.Parents["pai_id"] = e.Singular
			case r.byName[joined] != nil && len(w) >= 1 && !strings.Contains(joined, "="):
				tems = append(tems, pendingTem{e, r.byName[joined], b.Pos})
			case len(w) == 1 && personRoles[w[0]] != "":
				people = append(people, pendingPerson{e, w[0], personRoles[w[0]] == "muitos", b.Pos})
			default:
				fp.File = b.Pos.File
				before := len(e.Model.Fields)
				if err := fp.modelMember(e.Model, line); err != nil {
					return err
				}
				for _, f := range e.Model.Fields[before:] {
					for _, g := range e.Model.Fields[:before] {
						if strings.EqualFold(f.Name, g.Name) {
							return r.errAt(f.Pos, "%s já tem o campo %s", e.Singular, f.Name)
						}
					}
				}
			}
		}
	}

	// 2b. "X tem Ys": Y belongs to X, unless Y is claimed by an ancestor of X
	// too — then the ancestor owns Y and X keeps a list of them
	// (projeto tem labels; issue tem labels → labels of the project).
	claimers := map[*ast.Entity][]pendingTem{}
	var order []*ast.Entity
	for _, t := range tems {
		if len(claimers[t.child]) == 0 {
			order = append(order, t.child)
		}
		claimers[t.child] = append(claimers[t.child], t)
	}
	for _, child := range order {
		if len(claimers[child]) == 1 {
			t := claimers[child][0]
			r.hasMany(t.owner, t.child, t.pos)
		}
	}
	for _, child := range order {
		list := claimers[child]
		if len(list) < 2 {
			continue
		}
		var owner *pendingTem
		for i := range list {
			ok := true
			for j := range list {
				if i != j && !r.isAncestor(list[i].owner, list[j].owner) {
					ok = false
				}
			}
			if ok {
				owner = &list[i]
			}
		}
		if owner == nil {
			// Unrelated owners (issue tem comentarios; merge request tem
			// comentarios): each record belongs to one of them.
			for _, t := range list {
				r.hasMany(t.owner, child, t.pos)
				fieldByNameAST(child.Model, t.owner.Singular+"_id").Required = false
			}
			continue
		}
		r.hasMany(owner.owner, child, owner.pos)
		for _, t := range list {
			if t.owner != owner.owner {
				t.owner.Model.Fields = append(t.owner.Model.Fields, &ast.Field{Name: child.Plural, Type: ast.FieldLista, ListOf: child.Singular, Pos: t.pos})
			}
		}
	}

	// 3. pertence a
	for _, rel := range in.Relations {
		from, err := r.entity(rel.From, rel.Pos)
		if err != nil {
			return err
		}
		to, err := r.entity(rel.To, rel.Pos)
		if err != nil {
			return err
		}
		field := to.Singular + "_id"
		if rel.As != "" {
			field = rel.As + "_id"
		}
		if f := fieldByNameAST(from.Model, field); f != nil {
			f.Required = f.Required && !rel.Optional
			continue
		}
		from.Model.Fields = append(from.Model.Fields, &ast.Field{Name: field, Type: ast.FieldInteiro, Reference: to.Singular, Index: true, Required: !rel.Optional, Pos: rel.Pos})
		from.Parents[field] = to.Singular
		to.Children = appendUnique(to.Children, from.Singular)
	}

	// 3b. Numbering per parent and list targets.
	for _, n := range app.Order {
		e := app.Entities[n]
		for _, f := range e.Model.Fields {
			if f.NumberedBy != "" {
				pe, err := r.entity(f.NumberedBy, f.Pos)
				if err != nil {
					return err
				}
				fk := ""
				for field, target := range e.Parents {
					if target == pe.Singular {
						fk = field
					}
				}
				if fk == "" {
					return r.errAt(f.Pos, "%s numerado por %s: %s precisa pertencer a %s", f.Name, pe.Singular, e.Singular, pe.Singular)
				}
				f.NumberedBy = fk
				e.Model.UniqueTogether = append(e.Model.UniqueTogether, []string{fk, strings.ToLower(f.Name)})
			}
			if f.Type == ast.FieldLista && f.ListOf != "texto" && f.ListOf != "numero" {
				le, err := r.entity(f.ListOf, f.Pos)
				if err != nil {
					return err
				}
				f.ListOf = le.Singular
			}
		}
	}

	// 7b. States: `issue começa aberta`
	for _, st := range in.States {
		e, err := r.entity(st.Entity, st.Pos)
		if err != nil {
			return err
		}
		e.StateField, e.Initial = "estado", st.Initial
		e.Transitions = map[string]*ast.Transition{}
		e.Model.Fields = append(e.Model.Fields, &ast.Field{Name: "estado", Type: ast.FieldTexto, HasDefault: true, DefaultValue: st.Initial, System: true, Index: true, Pos: st.Pos})
	}
	// 4. Login entity: the one with a senha field.
	var withPassword []string
	for _, n := range app.Order {
		for _, f := range app.Entities[n].Model.Fields {
			if f.Type == ast.FieldSenha {
				withPassword = append(withPassword, n)
				break
			}
		}
	}
	if in.Login == nil && len(people) > 0 {
		return r.errAt(people[0].pos, "%s é uma pessoa: declare tenha login para que o sistema saiba quem são as pessoas", people[0].name)
	}
	if in.Login != nil {
		if len(withPassword) != 1 {
			return r.errAt(in.Login.Pos, "tenha login precisa de exatamente um dado com senha (encontrados: %v)", withPassword)
		}
		app.LoginEntity = withPassword[0]
		le := app.Entities[app.LoginEntity]
		if app.MemberModel != "" {
			me := app.Entities[app.MemberModel]
			if f := fieldByNameAST(me.Model, "pessoa_id"); f != nil {
				f.Reference = app.LoginEntity
				me.Parents["pessoa_id"] = app.LoginEntity
				le.Children = appendUnique(le.Children, me.Singular)
			}
		}
		if len(in.Login.Fields) == 0 {
			for _, cand := range []string{"email", "username", "login"} {
				if fieldByNameAST(le.Model, cand) != nil {
					in.Login.Fields = append(in.Login.Fields, cand)
				}
			}
		}
		for _, f := range in.Login.Fields {
			if fieldByNameAST(le.Model, f) == nil {
				return r.errAt(in.Login.Pos, "login usa %s, mas %s não tem esse campo", f, le.Singular)
			}
		}
		// People named in `tem` blocks: autor → autor_id, responsaveis → list.
		for _, pp := range people {
			if pp.many {
				pp.e.Model.Fields = append(pp.e.Model.Fields, &ast.Field{Name: pp.name, Type: ast.FieldLista, ListOf: app.LoginEntity, Pos: pp.pos})
				continue
			}
			field := pp.name + "_id"
			if fieldByNameAST(pp.e.Model, field) == nil {
				pp.e.Model.Fields = append(pp.e.Model.Fields, &ast.Field{Name: field, Type: ast.FieldInteiro, Reference: app.LoginEntity, Index: true, Pos: pp.pos})
			}
			pp.e.Parents[field] = app.LoginEntity
		}
		for _, n := range app.Order {
			e := app.Entities[n]
			for field, target := range e.Parents {
				if target != app.LoginEntity || e.Singular == app.MemberModel {
					continue
				}
				switch field {
				case app.LoginEntity + "_id", "autor_id", "criador_id", "dono_id":
					e.OwnerFields = appendUnique(e.OwnerFields, field)
				}
			}
			sort.Strings(e.OwnerFields)
		}
		if in.Login.LockAttempts > 0 {
			le.Model.Fields = append(le.Model.Fields,
				&ast.Field{Name: "tentativas_falhas", Type: ast.FieldInteiro, HasDefault: true, DefaultValue: 0.0, Hidden: true},
				&ast.Field{Name: "bloqueado_ate", Type: ast.FieldTexto, Hidden: true})
		}
		if in.Login.TokenEntity != "" {
			te, err := r.entity(in.Login.TokenEntity, in.Login.Pos)
			if err != nil {
				return err
			}
			if te.Parents[app.LoginEntity+"_id"] == "" {
				return r.errAt(in.Login.Pos, "%s precisa pertencer a %s para servir de login", te.Plural, le.Singular)
			}
			hasSecret := false
			for _, f := range te.Model.Fields {
				hasSecret = hasSecret || f.Type == ast.FieldSegredo
			}
			if !hasSecret {
				return r.errAt(in.Login.Pos, "%s precisa de um campo segredo (ex.: token segredo prefixo \"tok-\")", te.Plural)
			}
			in.Login.TokenEntity = te.Singular
		}
		if in.Login.ActiveField != "" && fieldByNameAST(le.Model, in.Login.ActiveField) == nil {
			return r.errAt(in.Login.Pos, "login exige %s, mas %s não tem esse campo", in.Login.ActiveField, le.Singular)
		}
	}

	// 5. Memberships inherited
	for _, m := range in.Memberships {
		if m.InheritFrom == "" {
			continue
		}
		e, err := r.entity(m.Entity, m.Pos)
		if err != nil {
			return err
		}
		src, err := r.entity(m.InheritFrom, m.Pos)
		if err != nil {
			return err
		}
		field := ""
		for f, target := range e.Parents {
			if target == src.Singular {
				field = f
			}
		}
		if field == "" {
			return r.errAt(m.Pos, "%s herda membros do %s, mas não pertence a %s", e.Singular, src.Singular, src.Singular)
		}
		if !src.HasMembers && src.HierarchyField == "" {
			return r.errAt(m.Pos, "%s não tem membros com papel", src.Singular)
		}
		e.InheritVia = field
	}

	// 6. Visibility field (privado/interno/publico)
	for _, n := range app.Order {
		e := app.Entities[n]
		for _, f := range e.Model.Fields {
			if f.Type == ast.FieldVisibilidade {
				e.Visibility = strings.ToLower(f.Name)
			}
		}
	}

	// Repository URL key: first unique text field (e.g. full_path).
	for _, n := range app.Order {
		e := app.Entities[n]
		if !e.Repository {
			continue
		}
		for _, f := range e.Model.Fields {
			if f.Unique && f.Type == ast.FieldTexto {
				e.RepoKey = strings.ToLower(f.Name)
				break
			}
		}
		if e.RepoKey == "" {
			return fmt.Errorf("%s tem repositório: declare um campo de texto único para o endereço (ex.: caminho único)", e.Plural)
		}
	}

	// 7. Hooks
	for _, h := range in.Hooks {
		e, err := r.entity(h.Target, h.Pos)
		if err != nil {
			return err
		}
		verb := CanonVerb(h.Verb)
		if h.Before {
			verb = "antes_" + verb
		}
		if _, dup := e.Hooks[verb]; dup {
			return r.errAt(h.Pos, "quando %s %s declarado duas vezes", h.Verb, h.Target)
		}
		e.Hooks[verb] = h
	}

	// 7c. Capabilities: `issue pode fechar / ser confidencial` (subject is data, not a role)
	var grants []*ast.Grant
	for _, g := range in.Grants {
		// A capability has no object: "issue pode fechar", "issue pode ser
		// confidencial". With an object it is a permission for people.
		e := r.byName[g.Role]
		if e == nil || app.Level(g.Role) > 0 || reservedRoles[g.Role] || (g.Target != "" && g.Verb != "ser") {
			grants = append(grants, g)
			continue
		}
		if g.Verb == "ser" {
			flag := g.Target
			if fieldByNameAST(e.Model, flag) == nil {
				e.Model.Fields = append(e.Model.Fields, &ast.Field{Name: flag, Type: ast.FieldBooleano, HasDefault: true, DefaultValue: false, Pos: g.Pos})
			}
			continue
		}
		if e.Transitions == nil {
			return r.errAt(g.Pos, "%s pode %s: diga como %s começa, por exemplo: %s começa aberto", e.Singular, g.Verb, e.Singular, e.Singular)
		}
		target := transitionTarget(g.Verb, e.Initial)
		tr := &ast.Transition{Verb: g.Verb, Target: target, Stamp: target != e.Initial}
		e.Transitions[g.Verb] = tr
		if tr.Stamp && fieldByNameAST(e.Model, target+"_em") == nil {
			e.Model.Fields = append(e.Model.Fields,
				&ast.Field{Name: target + "_em", Type: ast.FieldTexto, System: true, Pos: g.Pos},
				&ast.Field{Name: target + "_por_id", Type: ast.FieldInteiro, Reference: app.LoginEntity, System: true, Pos: g.Pos})
		}
	}
	in.Grants = grants

	// 7e. Visibility ceilings: a record is never more visible than its parent.
	for _, c := range in.Ceilings {
		e, err := r.entity(c.Entity, c.Pos)
		if err != nil {
			return err
		}
		field := ""
		if strings.HasSuffix(c.Parent, "_pai") && e.HierarchyField != "" {
			field = e.HierarchyField
		} else if pe := r.byName[c.Parent]; pe != nil {
			for f, t := range e.Parents {
				if t == pe.Singular {
					field = f
				}
			}
		}
		for _, f := range e.Model.Fields {
			if f.Type == ast.FieldVisibilidade {
				e.Visibility = strings.ToLower(f.Name)
			}
		}
		if field == "" || e.Visibility == "" {
			return r.errAt(c.Pos, "%s não pode ser mais visível que %s: é preciso que %s tenha visibilidade e pertença a %s", e.Singular, c.Parent, e.Singular, c.Parent)
		}
		e.CeilingFields = appendUnique(e.CeilingFields, field)
	}

	for _, c := range in.Creators {
		e, err := r.entity(c.Entity, c.Pos)
		if err != nil {
			return err
		}
		if !e.HasMembers || app.Level(c.Role) == 0 {
			return r.errAt(c.Pos, "quem cria %s vira %s: %s precisa ter membros com papel e %s precisa ser um papel", e.Singular, c.Role, e.Singular, c.Role)
		}
		e.CreatorRole = c.Role
	}

	// 7f. Code review: two branch fields (origem/destino).
	for _, n := range app.Order {
		e := app.Entities[n]
		var branches []string
		for _, f := range e.Model.Fields {
			if f.Type == ast.FieldBranch {
				branches = append(branches, strings.ToLower(f.Name))
			}
		}
		if len(branches) == 0 {
			continue
		}
		via := ""
		for field, t := range e.Parents {
			if app.Entities[t].Repository {
				via = field
			}
		}
		if via == "" {
			return r.errAt(e.Model.Pos, "%s tem branch, mas não pertence a algo que tem repositório", e.Plural)
		}
		if len(branches) >= 2 {
			src, dst := branches[0], branches[1]
			for _, b := range branches {
				switch b {
				case "origem", "source":
					src = b
				case "destino", "target":
					dst = b
				}
			}
			e.Review = &ast.Review{Source: src, Target: dst, RepoVia: via}
			e.Model.Fields = append(e.Model.Fields, &ast.Field{Name: "commit_mesclagem", Type: ast.FieldTexto, System: true})
		} else {
			e.Review = &ast.Review{Source: branches[0], RepoVia: via}
		}
	}
	for _, name := range in.Approvals {
		e, err := r.entity(name, diagnostics.Position{})
		if err != nil {
			return err
		}
		e.Approvals = true
		e.Model.Fields = append(e.Model.Fields, &ast.Field{Name: "aprovacoes", Type: ast.FieldLista, ListOf: app.LoginEntity, System: true})
	}

	for _, f := range in.Finals {
		e, err := r.entity(f.Entity, f.Pos)
		if err != nil {
			return err
		}
		known := f.Initial == e.Initial
		for _, tr := range e.Transitions {
			known = known || tr.Target == f.Initial
		}
		if !known {
			return r.errAt(f.Pos, "%s não chega ao estado %q; estados: começa %s e as ações de \"pode\"", e.Singular, f.Initial, e.Initial)
		}
		e.Finals = appendUnique(e.Finals, f.Initial)
	}

	// 7g. Executions: runs (pipelines) of steps (jobs) defined by a file.
	for _, x := range in.Executions {
		owner, err := r.entity(x.Owner, x.Pos)
		if err != nil {
			return err
		}
		run, err := r.entity(x.Entity, x.Pos)
		if err != nil {
			return err
		}
		if !owner.Repository {
			return r.errAt(x.Pos, "%s executa %s: %s precisa ter repositório", owner.Singular, run.Plural, owner.Singular)
		}
		ownerField := ""
		for f, t := range run.Parents {
			if t == owner.Singular {
				ownerField = f
			}
		}
		if ownerField == "" || len(run.Children) == 0 {
			return r.errAt(x.Pos, "%s precisa pertencer a %s e ter etapas (ex.: pipeline tem jobs)", run.Singular, owner.Singular)
		}
		step := app.Entities[run.Children[0]]
		runField := ""
		for f, t := range step.Parents {
			if t == run.Singular {
				runField = f
			}
		}
		run.Execution = &ast.Execution{Role: "run", File: x.File, Owner: owner.Singular, OwnerField: ownerField, Step: step.Singular}
		step.Execution = &ast.Execution{Role: "step", File: x.File, Owner: owner.Singular, Run: run.Singular, RunField: runField}
		sys := func(name string, t ast.FieldType) *ast.Field {
			return &ast.Field{Name: name, Type: t, System: true, Pos: x.Pos}
		}
		run.StateField, run.Initial = "estado", "pendente"
		run.Model.Fields = append(run.Model.Fields,
			&ast.Field{Name: "estado", Type: ast.FieldTexto, HasDefault: true, DefaultValue: "pendente", System: true, Index: true},
			&ast.Field{Name: "branch", Type: ast.FieldTexto, Pos: x.Pos}, sys("versao", ast.FieldTexto),
			sys("iniciado_em", ast.FieldTexto), sys("terminado_em", ast.FieldTexto), sys("duracao", ast.FieldNumero),
			sys("erro_configuracao", ast.FieldTextoLongo))
		run.Model.Fields = append(run.Model.Fields, &ast.Field{Name: app.LoginEntity + "_id", Type: ast.FieldInteiro, Reference: app.LoginEntity, System: true})
		step.StateField, step.Initial = "estado", "criado"
		step.Model.Fields = append(step.Model.Fields,
			&ast.Field{Name: "estado", Type: ast.FieldTexto, HasDefault: true, DefaultValue: "criado", System: true, Index: true},
			sys("nome", ast.FieldTexto), sys("etapa", ast.FieldTexto), sys("ordem", ast.FieldInteiro),
			&ast.Field{Name: "script", Type: ast.FieldTextoLongo, System: true, Hidden: true},
			sys("quando", ast.FieldTexto), sys("permitir_falha", ast.FieldBooleano),
			&ast.Field{Name: "log", Type: ast.FieldTextoLongo, System: true, Hidden: true},
			sys("iniciado_em", ast.FieldTexto), sys("terminado_em", ast.FieldTexto), sys("duracao", ast.FieldNumero),
			sys("repetido", ast.FieldBooleano), sys("imagem", ast.FieldTexto))
	}

	// 7d. Restricted visibility
	for _, vr := range in.Visibility {
		words := strings.Fields(vr.Entity)
		var e *ast.Entity
		flag := ""
		for k := len(words); k > 0; k-- {
			if cand := r.byName[strings.Join(words[:k], "_")]; cand != nil {
				e, flag = cand, strings.Join(words[k:], "_")
				break
			}
		}
		if e == nil || flag == "" {
			return r.errAt(vr.Pos, "use: <dado> <condição> pode ser visto por …, por exemplo: issue confidencial pode ser vista por autor")
		}
		if f := fieldByNameAST(e.Model, flag); f == nil || f.Type != ast.FieldBooleano {
			return r.errAt(vr.Pos, "%s não tem a condição %q; declare: %s pode ser %s", e.Singular, flag, e.Singular, flag)
		}
		rs := &ast.Restriction{Flag: flag}
		for _, who := range vr.Who {
			switch {
			case strings.HasSuffix(who, "_ou_superior"):
				role := strings.TrimSuffix(who, "_ou_superior")
				if app.Level(role) == 0 {
					return r.errAt(vr.Pos, "papel %q não declarado", role)
				}
				rs.MinRole = role
			case app.Level(who) > 0:
				rs.MinRole = who
			case fieldByNameAST(e.Model, who+"_id") != nil:
				rs.Owners = append(rs.Owners, who+"_id")
			case fieldByNameAST(e.Model, who) != nil && fieldByNameAST(e.Model, who).Type == ast.FieldLista:
				rs.Lists = append(rs.Lists, who)
			default:
				return r.errAt(vr.Pos, "quem é %q? use uma pessoa do dado (autor, responsaveis) ou um papel (reporter ou superior)", who)
			}
		}
		e.Restrictions = append(e.Restrictions, rs)
	}

	// 8. Permits and grants
	for _, pm := range in.Permits {
		e, err := r.entity(pm.Target, pm.Pos)
		if err != nil {
			return err
		}
		verb := CanonVerb(pm.Verb)
		switch verb {
		case "pesquisar":
			// Search changes how people find records, never who sees them.
			e.Search = searchable(e)
		case "filtrar":
			for i, f := range pm.By {
				if fieldByNameAST(e.Model, f) == nil && fieldByNameAST(e.Model, f+"_id") != nil {
					f = f + "_id"
					pm.By[i] = f
				}
				if fieldByNameAST(e.Model, f) == nil {
					return r.errAt(pm.Pos, "permita filtrar %s por %s: %s não tem esse campo", pm.Target, f, e.Singular)
				}
				e.Filters = appendUnique(e.Filters, f)
			}
		default:
			if err := r.checkVerb(e, verb, pm.Pos); err != nil {
				return err
			}
			addRule(e, &ast.AccessRule{Verb: verb, SignedIn: app.LoginEntity != "", Anyone: app.LoginEntity == ""})
		}
	}
	onlyCleared := map[string]bool{}
	for _, g := range in.Grants {
		role := g.Role
		rule := &ast.AccessRule{Own: g.Own}
		switch {
		case role == "todos" || role == "qualquer_pessoa" || role == "visitante":
			rule.Anyone = true
		case role == "administrador" || role == "admin":
			rule.MinRole = "administrador"
		case app.LoginEntity != "" && (role == app.LoginEntity || role == app.Entities[app.LoginEntity].Plural):
			rule.SignedIn = true
		case role == "autor" || role == "dono" || role == "criador":
			rule.SignedIn, rule.Own = true, true
		case role == "membro" || role == "membros":
			if len(app.Roles) == 0 {
				return r.errAt(g.Pos, "membro pode… exige tenha papeis")
			}
			rule.MinRole = app.Roles[0].Name
		case app.Level(role) > 0:
			rule.MinRole = role
		default:
			return r.errAt(g.Pos, "papel %q não declarado. Use tenha papeis ou um papel existente (administrador, todos, %s)", role, app.LoginEntity)
		}
		target := g.Target
		verb := CanonVerb(g.Verb)
		if verb == "enviar_codigo" && strings.HasPrefix(target, "branch_padrao") {
			rest := strings.TrimPrefix(target, "branch_padrao")
			for _, p := range []string{"_dos_", "_das_", "_do_", "_da_", "_de_", "_"} {
				if strings.HasPrefix(rest, p) {
					rest = strings.TrimPrefix(rest, p)
					break
				}
			}
			target = rest
		}
		if target == "perfil" && app.LoginEntity != "" {
			target, rule.Own = app.LoginEntity, true
		}
		e, err := r.entity(target, g.Pos)
		if err != nil {
			return err
		}
		if g.Only && !(verb == "enviar_codigo" && strings.HasPrefix(g.Target, "branch_padrao")) {
			key := e.Singular + ":" + verb
			if !onlyCleared[key] {
				delete(e.Rules, verb)
				onlyCleared[key] = true
			}
		}
		if verb == "administrar" {
			for _, v := range []string{"ver", "editar", "excluir"} {
				cp := *rule
				cp.Verb = v
				addRule(e, &cp)
			}
			for _, child := range r.governed(e) {
				for _, v := range []string{"ver", "criar", "editar", "excluir"} {
					cp := *rule
					cp.Verb = v
					addRule(child, &cp)
				}
			}
			continue
		}
		if verb == "enviar_codigo" && strings.HasPrefix(g.Target, "branch_padrao") {
			// somente maintainer pode enviar código para a branch padrão dos projetos
			if rule.MinRole == "" {
				return r.errAt(g.Pos, "a branch padrão é protegida por um papel, por exemplo: somente maintainer pode enviar código para a branch padrão dos projetos")
			}
			e.ProtectedBranchRole = rule.MinRole
			continue
		}
		if child := r.derivedChild(e, verb); child != nil {
			// comentar issues → criar comentarios (que pertencem a issues)
			rule.Verb = "criar"
			addRule(child, rule)
			continue
		}
		if err := r.checkVerb(e, verb, g.Pos); err != nil {
			return err
		}
		rule.Verb = verb
		addRule(e, rule)
	}

	// 9. Pages imply the operations they show and permit.
	for _, pg := range app.Pages {
		target := pg.Show
		if target == "" {
			target = pg.Manage
		}
		if target == "" {
			continue
		}
		e, err := r.entity(target, pg.Pos)
		if err != nil {
			return err
		}
		pg.Show = e.Singular
		if len(e.Rules["ver"]) == 0 {
			addRule(e, &ast.AccessRule{Verb: "ver", SignedIn: app.LoginEntity != "", Anyone: app.LoginEntity == ""})
		}
		for _, v := range pg.Permits {
			cv := CanonVerb(v)
			if cv == "pesquisar" {
				e.Search = searchable(e)
				continue
			}
			if len(e.Rules[cv]) == 0 {
				addRule(e, &ast.AccessRule{Verb: cv, SignedIn: app.LoginEntity != "", Anyone: app.LoginEntity == ""})
			}
		}
	}

	// 10. Integration
	for _, it := range in.Integrations {
		e, err := r.entity(it.Target, it.Pos)
		if err != nil {
			return err
		}
		e.Integrate = e.Plural
		if it.As != "" {
			e.Integrate = it.As
		}
	}

	// Custom verbs must have a definition.
	for _, n := range app.Order {
		e := app.Entities[n]
		for verb, rules := range e.Rules {
			if !standardVerb(verb) {
				builtin := (verb == "sair" && (e.HasMembers || e.InheritVia != "")) || (verb == "revogar" && e.Model.Revocable) || e.Transitions[verb] != nil ||
					((verb == "aprovar" || verb == "desaprovar") && e.Approvals) ||
					(e.Execution != nil && (verb == "cancelar" || verb == "repetir" || verb == "executar"))
				if _, ok := e.Hooks[verb]; !ok && !builtin {
					return fmt.Errorf("a ação %q sobre %s não tem definição. Escreva:\n\nquando %s %s\n    ...", verb, e.Plural, verb, e.Singular)
				}
				for _, rl := range rules {
					rl.Custom = true
				}
			}
		}
		prog.Models = append(prog.Models, e.Model)
	}
	if app.MemberModel != "" {
		// Members are ordinary data too (listed after their owners).
	}
	return nil
}

func standardVerb(v string) bool {
	switch v {
	case "ver", "criar", "editar", "excluir", "baixar_codigo", "enviar_codigo":
		return true
	}
	return false
}

func (r *resolver) checkVerb(e *ast.Entity, verb string, pos diagnostics.Position) error {
	if verb == "" {
		return r.errAt(pos, "ação ausente")
	}
	return nil
}

// governed lists entities administered through e: its children and, when
// e has members, the membership records.
func (r *resolver) governed(e *ast.Entity) []*ast.Entity {
	var out []*ast.Entity
	for _, c := range e.Children {
		out = append(out, r.app.Entities[c])
	}
	if e.HasMembers && r.app.MemberModel != "" {
		out = append(out, r.app.Entities[r.app.MemberModel])
	}
	return out
}

func (r *resolver) hasMany(parent, child *ast.Entity, pos diagnostics.Position) {
	field := parent.Singular + "_id"
	if fieldByNameAST(child.Model, field) == nil {
		child.Model.Fields = append(child.Model.Fields, &ast.Field{Name: field, Type: ast.FieldInteiro, Reference: parent.Singular, Index: true, Required: true, Pos: pos})
	}
	child.Parents[field] = parent.Singular
	parent.Children = appendUnique(parent.Children, child.Singular)
}

// membership: `grupo tem membros com papel`. The member entity is shared by
// every resource that has members (polymorphic: recurso, recurso_id).
func (r *resolver) membership(e *ast.Entity, member string, pos diagnostics.Position) error {
	if len(r.app.Roles) == 0 {
		return r.errAt(pos, "%s tem %s com papel: declare os papéis com tenha papeis", e.Plural, member)
	}
	m := r.byName[member]
	if m == nil {
		sing := Singular(member)
		m = &ast.Entity{Plural: member, Singular: sing, Label: titleCase(sing), Parents: map[string]string{},
			Model: &ast.Model{Name: sing, Internal: true, Label: titleCase(sing), Pos: pos},
			Rules: map[string][]*ast.AccessRule{}, Hooks: map[string]*ast.Hook{}}
		r.app.Entities[sing] = m
		r.app.Order = append(r.app.Order, sing)
		r.byName[member] = m
		r.byName[sing] = m
	}
	if r.app.MemberModel != "" && r.app.MemberModel != m.Singular {
		return r.errAt(pos, "todos os membros devem usar o mesmo dado (%s)", r.app.MemberModel)
	}
	if r.app.MemberModel == "" {
		r.app.MemberModel = m.Singular
		var roles []string
		for _, ro := range r.app.Roles {
			roles = append(roles, ro.Name)
		}
		m.Model.Fields = append(m.Model.Fields,
			&ast.Field{Name: "recurso", Type: ast.FieldTexto, Required: true, Immutable: true, Pos: pos},
			&ast.Field{Name: "recurso_id", Type: ast.FieldInteiro, Required: true, Immutable: true, Index: true, Pos: pos},
			&ast.Field{Name: "pessoa_id", Type: ast.FieldInteiro, Required: true, Immutable: true, Index: true, Pos: pos},
			&ast.Field{Name: "papel", Type: ast.FieldEnum, EnumValues: roles, Required: true, Pos: pos})
		m.Model.UniqueTogether = append(m.Model.UniqueTogether, []string{"recurso", "recurso_id", "pessoa_id"})
	}
	e.HasMembers = true
	return nil
}

func searchable(e *ast.Entity) []string {
	var out []string
	for _, f := range e.Model.Fields {
		if (f.Type == ast.FieldTexto || f.Type == ast.FieldEmail || f.Type == ast.FieldTextoLongo) && !f.IsSecret() && !f.Hidden && !f.System && !f.Private {
			out = append(out, strings.ToLower(f.Name))
		}
	}
	return out
}

func addRule(e *ast.Entity, rule *ast.AccessRule) {
	e.Rules[rule.Verb] = append(e.Rules[rule.Verb], rule)
}

func appendUnique(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}

func fieldByNameAST(m *ast.Model, name string) *ast.Field {
	for _, f := range m.Fields {
		if strings.EqualFold(f.Name, name) {
			return f
		}
	}
	return nil
}

var _ = lexer.TokenEOF

// personRoles: nouns that name people in `tem` blocks ("um" = one person,
// "muitos" = several).
var personRoles = map[string]string{
	"autor": "um", "dono": "um", "criador": "um", "responsavel": "um", "revisor": "um", "aprovador": "um",
	"relator": "um", "solicitante": "um", "atendente": "um", "vendedor": "um", "cliente_responsavel": "um",
	"responsaveis": "muitos", "revisores": "muitos", "aprovadores": "muitos", "participantes": "muitos",
	"seguidores": "muitos", "atendentes": "muitos", "interessados": "muitos",
}

func (r *resolver) isAncestor(a, b *ast.Entity) bool {
	seen := map[*ast.Entity]bool{}
	var walk func(x *ast.Entity) bool
	walk = func(x *ast.Entity) bool {
		if x == nil || seen[x] {
			return false
		}
		seen[x] = true
		for _, t := range x.Parents {
			p := r.app.Entities[t]
			if p == a || walk(p) {
				return true
			}
		}
		return false
	}
	return walk(b)
}

// derivedChild: `comentar issues` → comentarios that belong to issues. The
// verb stem (comentar → coment) must start the child's name.
func (r *resolver) derivedChild(e *ast.Entity, verb string) *ast.Entity {
	if standardVerb(verb) || len(verb) < 5 || e.Transitions[verb] != nil || e.Hooks[verb] != nil {
		return nil
	}
	stem := verb[:len(verb)-2]
	for _, c := range e.Children {
		if strings.HasPrefix(c, stem) {
			return r.app.Entities[c]
		}
	}
	return nil
}

// transitionTarget: the state a verb leads to. re-/des- verbs return to the
// initial state (reabrir, desbloquear); others use the past participle with
// the gender of the initial state (aberta → fechada, aberto → fechado).
func transitionTarget(verb, initial string) string {
	if strings.HasPrefix(verb, "re") || strings.HasPrefix(verb, "des") {
		return initial
	}
	fem := strings.HasSuffix(initial, "a")
	irregular := map[string]string{"abrir": "abert", "fazer": "feit", "pagar": "pag", "aceitar": "aceit", "escrever": "escrit",
		"cobrir": "cobert", "ganhar": "ganh", "gastar": "gast", "entregar": "entregu", "por": "post", "ver": "vist"}
	base, ok := irregular[verb]
	switch {
	case ok:
	case strings.HasSuffix(verb, "ar"):
		base = verb[:len(verb)-2] + "ad"
	case strings.HasSuffix(verb, "er"), strings.HasSuffix(verb, "ir"):
		base = verb[:len(verb)-2] + "id"
	default:
		base = verb
	}
	if base == "entregu" {
		return "entregue"
	}
	if fem {
		return base + "a"
	}
	return base + "o"
}

// reservedRoles always name people, never data.
var reservedRoles = map[string]bool{"membro": true, "membros": true, "administrador": true, "admin": true, "todos": true,
	"qualquer_pessoa": true, "visitante": true, "autor": true, "dono": true, "criador": true}
