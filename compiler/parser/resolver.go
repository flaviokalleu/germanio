package parser

import (
	"fmt"
	"reflect"
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
	if len(in.Conflicts) > 0 {
		return fmt.Errorf("%s", in.Conflicts[0])
	}
	if err := withPendingData(in); err != nil {
		return err
	}
	if err := withHistoryData(in); err != nil {
		return err
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
	declared := map[string]*ast.EntityDecl{}
	for _, d := range in.Entities {
		sing := d.Singular
		if sing == "" {
			sing = Singular(d.Name)
		}
		if e, dup := r.byName[d.Name]; dup {
			// A data block declares its data; several blocks (and an explicit
			// tenha) describing the same data merge. Two explicit tenha conflict.
			if prev := declared[d.Name]; d.Implicit || (prev != nil && prev.Implicit) {
				if !d.Implicit {
					if d.Label != "" {
						e.Label, e.Model.Label = d.Label, d.Label
					}
					e.Model.Internal = e.Model.Internal || d.Internal
					declared[d.Name] = d
				}
				continue
			}
			return r.errAt(d.Pos, "%q declarado mais de uma vez", d.Name)
		}
		declared[d.Name] = d
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
	type pendingAddress struct {
		e     *ast.Entity
		words []string
		pos   diagnostics.Position
	}
	var tems []pendingTem
	var people []pendingPerson
	var addresses []pendingAddress
	type pendingNamed struct {
		e, child *ast.Entity
		key      string
		pos      diagnostics.Position
	}
	var named []pendingNamed
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
			case len(w) == 3 && w[1] == "por" && r.byName[w[0]] != nil:
				// labels por nome: a list whose items are named, not numbered
				tems = append(tems, pendingTem{e, r.byName[w[0]], b.Pos})
				named = append(named, pendingNamed{e, r.byName[w[0]], w[2], b.Pos})
			case len(w) >= 3 && w[0] == "endereco" && w[1] == "dentro":
				// endereço dentro do grupo pai [ou do criador]
				addresses = append(addresses, pendingAddress{e, w[2:], b.Pos})
			case joined == "repositorio" || joined == "repositorio_git":
				e.Repository = true
				e.Model.Fields = append(e.Model.Fields, &ast.Field{Name: "repositorio", Type: ast.FieldTexto, Hidden: true, System: true, Pos: b.Pos})
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
				// Blocks merge: the same field written again changes nothing;
				// a different definition is a conflict showing both origins.
				kept := e.Model.Fields[:before]
				for _, f := range e.Model.Fields[before:] {
					dup := false
					for _, g := range e.Model.Fields[:before] {
						if !strings.EqualFold(f.Name, g.Name) {
							continue
						}
						if !sameField(f, g) {
							return r.errAt(f.Pos, "o campo %s de %s já foi definido de outro jeito em %s; aqui a definição é diferente. Deixe uma só definição", f.Name, e.Plural, where(g.Pos))
						}
						dup = true
					}
					if !dup {
						kept = append(kept, f)
					}
				}
				e.Model.Fields = kept
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

	// 2c. Lists named by a field: `labels por nome`.
	for _, n := range named {
		var list *ast.Field
		for _, f := range n.e.Model.Fields {
			if f.Type == ast.FieldLista && f.ListOf == n.child.Singular {
				list = f
			}
		}
		if list == nil {
			return r.errAt(n.pos, "%s por %s: vale para dados que %s usa de outro dono (ex.: issue tem labels por nome, e projeto tem labels)", n.child.Plural, n.key, n.e.Singular)
		}
		if fieldByNameAST(n.child.Model, n.key) == nil {
			return r.errAt(n.pos, "%s por %s: %s não tem o campo %s", n.child.Plural, n.key, n.child.Singular, n.key)
		}
		list.ByName = n.key
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
	stateOrigin := map[*ast.Entity]*ast.StateDecl{}
	for _, st := range in.States {
		e, err := r.entity(st.Entity, st.Pos)
		if err != nil {
			return err
		}
		if prev := stateOrigin[e]; prev != nil {
			// one value only: the same fact again changes nothing; two values are a conflict
			if prev.Initial == st.Initial {
				continue
			}
			return r.errAt(st.Pos, "%s começa %s, mas já começa %s em %s\nPor quê: um dado tem um único estado inicial\nComo corrigir: mantenha só uma das declarações", e.Singular, st.Initial, prev.Initial, where(prev.Pos))
		}
		stateOrigin[e] = st
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
		if in.Login.LockAttempts == 0 {
			// Safe default: guessing passwords is stopped even when the app does
			// not say how (docs/INTENCAO.md › Login).
			in.Login.LockAttempts, in.Login.LockMinutes = 10, 10
		}
		if in.Login.LockAttempts > 0 {
			le.Model.Fields = append(le.Model.Fields,
				&ast.Field{Name: "tentativas_falhas", Type: ast.FieldInteiro, HasDefault: true, DefaultValue: 0.0, Hidden: true, System: true},
				&ast.Field{Name: "bloqueado_ate", Type: ast.FieldTexto, Hidden: true, System: true})
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
			if len(in.Login.Scopes) > 0 && fieldByNameAST(te.Model, "escopos") == nil {
				return r.errAt(in.Login.Pos, "escopos declarados, mas %s não tem o campo escopos", te.Plural)
			}
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

	// Addresses: `endereço dentro do grupo pai ou do criador`.
	for _, pa := range addresses {
		e := pa.e
		if fieldByNameAST(e.Model, "caminho") == nil {
			return r.errAt(pa.pos, "%s tem endereço: declare também o campo caminho (a parte do endereço que é só de cada %s)", e.Singular, e.Singular)
		}
		if fieldByNameAST(e.Model, "endereco") != nil {
			return r.errAt(pa.pos, "%s já tem o campo endereco", e.Singular)
		}
		addr := &ast.Address{Field: "endereco", Segment: "caminho"}
		var part []string
		flush := func() error {
			name, _ := phrase(part)
			part = nil
			ref, err := r.addressRef(app, e, name, pa.pos)
			if err != nil {
				return err
			}
			addr.Within = append(addr.Within, ref)
			return nil
		}
		for _, w := range pa.words {
			if w == "ou" {
				if err := flush(); err != nil {
					return err
				}
				continue
			}
			part = append(part, w)
		}
		if err := flush(); err != nil {
			return err
		}
		e.Address = addr
		e.Model.Fields = append(e.Model.Fields, &ast.Field{Name: "endereco", Type: ast.FieldTexto, Unique: true, System: true, Index: true, Pos: pa.pos})
	}

	// Initial files of new repositories.
	for _, f := range in.InitialFiles {
		e, err := r.entity(f.Entity, f.Pos)
		if err != nil {
			return err
		}
		if !e.Repository {
			return r.errAt(f.Pos, "repositório do %s: %s não tem repositório (use \"%s tem repositório\")", e.Singular, e.Singular, e.Singular)
		}
		e.InitialFile = &ast.RepoFile{Path: f.Path, Content: f.Content}
	}

	// Repository URL key: the address, else the first unique text field.
	for _, n := range app.Order {
		e := app.Entities[n]
		if !e.Repository {
			continue
		}
		if e.Address != nil {
			e.RepoKey = e.Address.Field
		}
		for _, f := range e.Model.Fields {
			if e.RepoKey == "" && f.Unique && f.Type == ast.FieldTexto {
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
		// external effects run after the commit (G86): their answer does
		// not exist yet inside the change
		if used := ast.EffectsUsed(h.Body); len(used) > 0 {
			return r.errAt(used[0].Pos, "%s age fora do sistema e só acontece depois que a mudança é salva, então a resposta ainda não existe dentro de quando %s %s. Chame %s numa linha sozinha, sem usar o resultado (acontece logo depois de salvar), ou use tarefas.enfileirar(\"funcao\", dados) para trabalhar com a resposta fora da mudança", ast.EffectKind(used[0]), h.Verb, h.Target, used[0].Name)
		}
		h.Target = e.Singular // the resolved name, whatever form was written
		e.Hooks[verb] = h
	}

	// 7b2. Remote work that is not a pipeline step starts waiting for an
	// executor, so `conversao pode cancelar` has a state to leave.
	steps := map[string]bool{}
	for _, x := range in.Executions {
		if run, err := r.entity(x.Entity, x.Pos); err == nil {
			for _, c := range run.Children {
				steps[c] = true
			}
		}
	}
	for _, x := range in.RemoteExecutors {
		work, err := r.entity(x.Steps, x.Pos)
		if err != nil {
			return err
		}
		if !steps[work.Singular] && work.StateField == "" {
			work.StateField, work.Initial = "estado", "pendente"
			work.Transitions = map[string]*ast.Transition{}
			work.Model.Fields = append(work.Model.Fields, &ast.Field{Name: "estado", Type: ast.FieldTexto, HasDefault: true, DefaultValue: "pendente", System: true, Index: true, Pos: x.Pos})
		}
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
		in.Capabilities = append(in.Capabilities, g)
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

	// 7d2. Read-only while a condition holds: `projeto arquivado é somente leitura`.
	for _, ro := range in.ReadOnly {
		e, err := r.entity(ro.Entity, ro.Pos)
		if err != nil {
			return err
		}
		if f := fieldByNameAST(e.Model, ro.Flag); f == nil || f.Type != ast.FieldBooleano {
			return r.errAt(ro.Pos, "%s %s é somente leitura: declare antes a condição (ex.: %s pode ser %s)", e.Singular, ro.Flag, e.Singular, ro.Flag)
		}
		e.ReadOnlyWhen = ro.Flag
	}

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
			&ast.Field{Name: "quando", Type: ast.FieldEnum, EnumValues: []string{"automatico", "manual", "sempre"}, System: true, Pos: x.Pos}, sys("permitir_falha", ast.FieldBooleano),
			&ast.Field{Name: "log", Type: ast.FieldTextoLongo, System: true, Hidden: true},
			sys("iniciado_em", ast.FieldTexto), sys("terminado_em", ast.FieldTexto), sys("duracao", ast.FieldNumero),
			sys("repetido", ast.FieldBooleano), sys("imagem", ast.FieldTexto))
	}
	if err := r.runVariables(in); err != nil {
		return err
	}

	// 7g1. Minimum role: `todo grupo precisa ter pelo menos um owner`.
	for _, m := range in.MinRoles {
		e, err := r.entity(m.Entity, m.Pos)
		if err != nil {
			return err
		}
		if !e.HasMembers {
			return r.errAt(m.Pos, "%s precisa ter pelo menos um %s: %s não tem membros (use \"%s tem membros com papel\")", e.Singular, m.Role, e.Singular, e.Singular)
		}
		if app.Level(m.Role) == 0 {
			return r.errAt(m.Pos, "papel desconhecido %q", m.Role)
		}
		e.MinRole = m.Role
	}

	// 7g2. Remote work: `runners executam jobs`, `trabalhadores executam conversoes`.
	for _, x := range in.RemoteExecutors {
		ex, err := r.entity(x.Executor, x.Pos)
		if err != nil {
			return err
		}
		work, err := r.entity(x.Steps, x.Pos)
		if err != nil {
			return err
		}
		key := ""
		for _, f := range ex.Model.Fields {
			if f.Type == ast.FieldSegredo {
				key = strings.ToLower(f.Name)
			}
		}
		if key == "" {
			return r.errAt(x.Pos, "%s executam %s: cada %s precisa de uma credencial (ex.: token secreto)", ex.Plural, work.Plural, ex.Singular)
		}
		sys := func(name string, t ast.FieldType, hidden bool) *ast.Field {
			return &ast.Field{Name: name, Type: t, System: true, Hidden: hidden, Pos: x.Pos}
		}
		rw := &ast.RemoteWork{Executor: ex.Singular, ExecutorField: ex.Singular + "_id", ExecutorKey: key, Pending: "pendente", Canceled: "cancelado"}
		if work.Execution == nil {
			// Plain work: it gets the work state machine unless it declared its own start.
			rw.Pending = work.Initial
			if t := work.Transitions["cancelar"]; t != nil {
				rw.Canceled = t.Target
			}
			for _, st := range []string{"sucesso", "falhou", rw.Canceled} {
				work.Finals = appendUnique(work.Finals, st)
			}
			work.Model.Fields = append(work.Model.Fields, sys("log", ast.FieldTextoLongo, true),
				sys("iniciado_em", ast.FieldTexto, false), sys("terminado_em", ast.FieldTexto, false), sys("duracao", ast.FieldNumero, false))
		}
		work.Model.Fields = append(work.Model.Fields,
			&ast.Field{Name: rw.ExecutorField, Type: ast.FieldInteiro, Reference: ex.Singular, System: true, Index: true, Pos: x.Pos},
			&ast.Field{Name: "token_execucao", Type: ast.FieldTexto, System: true, Hidden: true, Index: true, Pos: x.Pos},
			sys("reserva_ate", ast.FieldTexto, true), sys("chave_reserva", ast.FieldTexto, true), sys("tentativas", ast.FieldInteiro, false))
		ex.Model.Fields = append(ex.Model.Fields, sys("visto_em", ast.FieldTexto, false))
		work.Remote = rw
	}

	// 7h. Subscriptions: webhooks receive events of their owner.
	for _, sd := range in.Subscriptions {
		sub, err := r.entity(sd.Subscriber, sd.Pos)
		if err != nil {
			return err
		}
		owner, err := r.entity(sd.Owner, sd.Pos)
		if err != nil {
			return err
		}
		field := ""
		for f, t := range sub.Parents {
			if t == owner.Singular {
				field = f
			}
		}
		if field == "" || fieldByNameAST(sub.Model, "url") == nil {
			return r.errAt(sd.Pos, "%s recebe eventos de %s: %s precisa pertencer a %s e ter url", sub.Singular, owner.Singular, sub.Singular, owner.Singular)
		}
		var kinds []string
		for _, k := range sd.Kinds {
			if k != "enviar_codigo" {
				ke, err := r.entity(k, sd.Pos)
				if err != nil {
					return err
				}
				k = ke.Plural
			}
			kinds = append(kinds, k)
			sub.Model.Fields = append(sub.Model.Fields, &ast.Field{Name: "eventos_" + k, Type: ast.FieldBooleano, HasDefault: true, DefaultValue: true, Pos: sd.Pos})
		}
		sub.Subscription = &ast.Subscription{Owner: owner.Singular, OwnerField: field, Kinds: kinds}
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
		case app.Level(role) > 0:
			// a declared role wins over the owner words below (`tenha papeis … dono 50`)
			rule.MinRole = role
		case role == "autor" || role == "dono" || role == "criador":
			rule.SignedIn, rule.Own = true, true
		case role == "membro" || role == "membros":
			if len(app.Roles) == 0 {
				return r.errAt(g.Pos, "membro pode… exige tenha papeis")
			}
			rule.MinRole = app.Roles[0].Name
		default:
			existing := []string{"administrador", "todos"}
			if app.LoginEntity != "" {
				existing = append(existing, app.LoginEntity)
			}
			for _, rl := range app.Roles {
				existing = append(existing, rl.Name)
			}
			return r.errAt(g.Pos, "papel %q não declarado. Use tenha papeis ou um papel existente (%s)", role, strings.Join(existing, ", "))
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
		if verb == "enviar_codigo" {
			// somente maintainer pode enviar código para as branches protegidas
			// (dos projetos): the rule is about the data with the repository
			if owner, data := r.protectedTarget(target, g.Context); owner != nil {
				if err := r.protectedBranches(owner, data, g, rule); err != nil {
					return err
				}
				continue
			}
		}
		e, err := r.entity(target, g.Pos)
		if err != nil {
			return err
		}
		// Inside a data block (acesso), an action speaks of that data or of
		// what belongs to it; a rule about other data goes in that data's block.
		if g.Context != "" {
			if c, cerr := r.entity(g.Context, g.Pos); cerr == nil && !r.belongsTo(e, c) {
				return r.errAt(g.Pos, "a ação \"%s %s\" está no bloco de %s, mas fala de %s\nOnde: %s\nPor quê: dentro de um dado, acesso trata desse dado ou do que pertence a ele\nComo corrigir: escreva essa regra no bloco de %s (ou como frase: %s pode %s %s)",
					g.Verb, strings.ReplaceAll(g.Target, "_", " "), c.Plural, e.Plural, g.Pos.Context, e.Plural, g.Role, g.Verb, e.Plural)
			}
		}
		if g.Only && !(verb == "enviar_codigo" && strings.HasPrefix(g.Target, "branch_padrao")) {
			key := e.Singular + ":" + verb
			if !onlyCleared[key] {
				delete(e.Rules, verb)
				onlyCleared[key] = true
			}
		}
		if verb == "administrar" {
			verbs := []string{"ver", "editar", "excluir"}
			if g.Target == e.Plural || Singular(g.Target) == e.Singular && g.Target != e.Singular {
				verbs = append(verbs, "criar") // administrar runners: the whole collection
			}
			for _, v := range verbs {
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
		if err := r.indicators(pg); err != nil {
			return err
		}
		target := pg.Show
		if target == "" {
			target = pg.Manage
		}
		if target == "" {
			if len(pg.Indicators) == 0 {
				return r.errAt(pg.Pos, "a página %s não mostra nada. Escreva abaixo dela: mostre <dados>, ou indicadores com total de <dados>", pg.Name)
			}
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
		if err := r.pageSections(e, pg); err != nil {
			return err
		}
	}

	// 10a. Renames and discards (G93): explicit, checked here without the
	// database (ge check); the migration applies them preserving the data.
	if err := r.renames(in); err != nil {
		return err
	}

	// 10. Pending items (GEP 0009, em teste): each field must hold people.
	for _, pr := range in.PendingItems {
		e, err := r.entity(pr.Entity, pr.Pos)
		if err != nil {
			return err
		}
		for _, name := range pr.Fields {
			if name == "mencionados" && fieldByNameAST(e.Model, "mencionados") == nil {
				// pendência para › mencionados (GEP 0017, em teste)
				if err := r.mentions(e, app, pr); err != nil {
					return err
				}
				continue
			}
			var found string
			for _, f := range e.Model.Fields {
				fn := strings.ToLower(f.Name)
				if (fn == name && f.Type == ast.FieldLista && f.ListOf == app.LoginEntity) || (fn == name+"_id" && f.Reference == app.LoginEntity) {
					found = fn
				}
			}
			if found == "" {
				return r.errAt(pr.Pos, "%s gera pendência para %s, mas %s não é um campo de pessoas de %s (use, por exemplo, responsaveis ou revisor)", e.Plural, name, name, e.Plural)
			}
			e.PendingFields = appendUnique(e.PendingFields, found)
		}
		if pe := r.byName["pendencias"]; pe != nil {
			app.PendingEntity = pe.Singular
		}
	}

	// 10a2. Notices by e-mail (GEP 0013, em teste) tell people about their
	// pending items.
	if in.EmailNotices {
		if len(in.PendingItems) == 0 {
			return r.errAt(in.EmailNoticesPos, "tenha avisos por e-mail avisa cada pessoa das suas pendências, mas nenhum dado gera pendências. Declare, no bloco do dado: pendência para › responsaveis")
		}
		app.EmailNotices = true
	}

	// 10a3. Presence (GEP 0021, em teste) is about the people who log in.
	if in.Presence {
		if app.LoginEntity == "" {
			return fmt.Errorf("tenha presença mostra quem está com a aplicação aberta: declare também tenha login")
		}
		app.Presence = true
	}

	// 10b. History (GEP 0011, em teste) and reading (GEP 0022, em teste).
	if err := r.history(in, app); err != nil {
		return err
	}
	if err := r.reading(in, app); err != nil {
		return err
	}

	app.ReservedAddresses = in.ReservedAddresses
	for _, name := range in.GlobalSearch {
		e, err := r.entity(name, diagnostics.Position{})
		if err != nil {
			return fmt.Errorf("tenha busca geral: %w", err)
		}
		if len(e.Search) == 0 {
			return fmt.Errorf("tenha busca geral em %s: declare também permita pesquisar %s", e.Plural, e.Plural)
		}
		app.GlobalSearch = append(app.GlobalSearch, e.Singular)
	}
	if in.InitialAdmin != "" {
		if app.LoginEntity == "" {
			return fmt.Errorf("tenha administrador inicial: declare antes tenha login")
		}
		if f := fieldByNameAST(app.Entities[app.LoginEntity].Model, "admin"); f == nil || f.Type != ast.FieldBooleano {
			return fmt.Errorf("tenha administrador inicial: %s precisa do campo admin (ex.: admin começa com falso)", app.LoginEntity)
		}
		app.InitialAdmin = in.InitialAdmin
	}

	// 9b. Translation points filled by adapters (`traduza X com f`).
	for point, fn := range in.Translators {
		switch point {
		case "arquivos_de_execucao", "variaveis_das_etapas":
		default:
			return fmt.Errorf("traduza %s: ponto desconhecido (use: arquivos de execução, variáveis das etapas)", strings.ReplaceAll(point, "_", " "))
		}
		found := false
		for _, f := range prog.Functions {
			if f.Name == fn {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("traduza %s com %s: a função %s não existe", strings.ReplaceAll(point, "_", " "), fn, fn)
		}
		if app.Translators == nil {
			app.Translators = map[string]string{}
		}
		app.Translators[point] = fn
	}

	// 10. Integration
	integOrigin := map[*ast.Entity]*ast.Integration{}
	for _, it := range in.Integrations {
		e, err := r.entity(it.Target, it.Pos)
		if err != nil {
			return err
		}
		name := e.Plural
		if it.As != "" {
			name = it.As
		}
		if prev := integOrigin[e]; prev != nil {
			if e.Integrate == name {
				continue // the same fact again
			}
			return r.errAt(it.Pos, "%s é integrado como %q, mas já é %q em %s\nPor quê: um dado tem um único nome na integração\nComo corrigir: mantenha só um nome", e.Plural, name, e.Integrate, where(prev.Pos))
		}
		integOrigin[e] = it
		e.Integrate = name
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
// governed: what `administrar X` also covers — the data that belongs to X
// (a required reference to it), never data that only refers to X optionally
// (an issue may sit in a milestone; it does not belong to it).
func (r *resolver) governed(e *ast.Entity) []*ast.Entity {
	var out []*ast.Entity
	for _, c := range e.Children {
		child := r.app.Entities[c]
		if child == nil {
			continue
		}
		owned := false
		for field, target := range child.Parents {
			if target == e.Singular {
				if f := fieldByNameAST(child.Model, field); f != nil && f.Required {
					owned = true
				}
			}
		}
		if owned {
			out = append(out, child)
		}
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

// addressRef resolves one container of an address: "grupo pai"/"pai" (the
// hierarchy), a person (criador, dono) or an entity the record belongs to.
func (r *resolver) addressRef(app *ast.App, e *ast.Entity, name string, pos diagnostics.Position) (ast.AddressRef, error) {
	if name == "pai" || name == e.Singular+"_pai" {
		if e.HierarchyField == "" {
			return ast.AddressRef{}, r.errAt(pos, "%s: endereço dentro do %s pai precisa de \"%s tem sub%s\"", e.Singular, e.Singular, e.Singular, e.Plural)
		}
		return ast.AddressRef{Field: e.HierarchyField, Entity: e.Singular}, nil
	}
	if personRoles[name] != "" || name == app.LoginEntity {
		field := name + "_id"
		if e.Parents[field] == "" {
			return ast.AddressRef{}, r.errAt(pos, "%s: endereço dentro do %s precisa de %s na lista do que %s tem", e.Singular, name, name, e.Singular)
		}
		return ast.AddressRef{Field: field, Entity: e.Parents[field]}, nil
	}
	target := r.byName[name]
	if target == nil {
		return ast.AddressRef{}, r.errAt(pos, "%s: endereço dentro de %q: não conheço esse dado", e.Singular, name)
	}
	var found []string
	for field, t := range e.Parents {
		if t == target.Singular {
			found = append(found, field)
		}
	}
	if len(found) != 1 {
		return ast.AddressRef{}, r.errAt(pos, "%s: endereço dentro do %s precisa de \"%s pertence a %s\"", e.Singular, target.Singular, e.Singular, target.Singular)
	}
	return ast.AddressRef{Field: found[0], Entity: target.Singular}, nil
}

// renames checks `renomeie a para b` and `descarte a`: the new name must be
// declared and the old one not (a rename replaces one name by the other);
// a field is renamed at most once, no two fields get the same new name, and
// renames cannot chain: the middle name would have to be declared and not
// declared at once, so the checks above already refuse it.
func (r *resolver) renames(in *ast.Intent) error {
	type key struct{ entity, name string }
	from := map[key]*ast.RenameDecl{}
	to := map[key]*ast.RenameDecl{}
	for _, rn := range in.Renames {
		e, err := r.entity(rn.Entity, rn.Pos)
		if err != nil {
			return err
		}
		switch {
		case rn.From == rn.To:
			return r.errAt(rn.Pos, "renomeie %s para %s: o nome novo é o mesmo", rn.From, rn.To)
		case fieldByNameAST(e.Model, rn.To) == nil:
			return r.errAt(rn.Pos, "renomeie %s para %s: %s não está declarado em %s; declare o campo novo em tem (com o nome novo)", rn.From, rn.To, rn.To, e.Plural)
		case fieldByNameAST(e.Model, rn.From) != nil:
			return r.errAt(rn.Pos, "renomeie %s para %s: %s continua declarado em %s. Um rename troca um nome pelo outro: tire %s de tem", rn.From, rn.To, rn.From, e.Plural, rn.From)
		}
		kf, kt := key{e.Singular, rn.From}, key{e.Singular, rn.To}
		if prev := from[kf]; prev != nil {
			return r.errAt(rn.Pos, "%s de %s já é renomeado para %s em %s", rn.From, e.Plural, prev.To, where(prev.Pos))
		}
		if prev := to[kt]; prev != nil {
			return r.errAt(rn.Pos, "%s e %s não podem virar o mesmo campo %s (o outro rename está em %s)", prev.From, rn.From, rn.To, where(prev.Pos))
		}
		from[kf], to[kt] = rn, rn
		e.Model.Renames = append(e.Model.Renames, ast.FieldRename{From: rn.From, To: rn.To, Pos: rn.Pos})
	}
	for _, d := range in.Discards {
		e, err := r.entity(d.Entity, d.Pos)
		if err != nil {
			return err
		}
		if fieldByNameAST(e.Model, d.From) != nil {
			return r.errAt(d.Pos, "descarte %s: %s continua declarado em %s", d.From, d.From, e.Plural)
		}
		e.Model.Discarded = appendUnique(e.Model.Discarded, d.From)
	}
	return nil
}

// pageSections checks the sections of a page against the data it shows
// (GEP 0002, em teste): an action must already be allowed by the page (the
// page asks, the domain decides who sees it), filters imply the filter (it
// only narrows what is already visible), columns must exist and not be
// secret.
func (r *resolver) pageSections(e *ast.Entity, pg *ast.PageDecl) error {
	allowed := map[string]bool{}
	for _, v := range pg.Permits {
		allowed[CanonVerb(v)] = true
	}
	checkAction := func(a *ast.PageAction) error {
		if a == nil {
			return nil
		}
		if !allowed[CanonVerb(a.Verb)] {
			return r.errAt(a.Pos, "a página %s oferece a ação %s, mas não a permite. Acrescente %s em permita da página (quem pode fazer continua decidido pelo acesso de %s)", pg.Name, a.Verb, a.Verb, e.Plural)
		}
		return nil
	}
	for _, a := range pg.Actions {
		if err := checkAction(a); err != nil {
			return err
		}
		if CanonVerb(a.Verb) != "criar" {
			return r.errAt(a.Pos, "no topo da página só cabe a ação criar por enquanto; %s é uma ação de cada registro e aparece nele", a.Verb)
		}
	}
	if pg.Empty != nil {
		if err := checkAction(pg.Empty.Action); err != nil {
			return err
		}
	}
	for _, f := range pg.Filters {
		if f == "pesquisar" {
			if len(e.Search) == 0 {
				e.Search = searchable(e)
			}
			continue
		}
		fd := fieldByNameAST(e.Model, f)
		if fd == nil {
			return r.errAt(pg.Pos, "a página %s filtra por %s, mas %s não tem esse campo", pg.Name, f, e.Plural)
		}
		name := strings.ToLower(fd.Name)
		known := false
		for _, have := range e.Filters {
			known = known || have == name
		}
		if !known {
			e.Filters = append(e.Filters, name)
		}
	}
	for _, c := range pg.Columns {
		fd := fieldByNameAST(e.Model, c)
		if fd == nil {
			return r.errAt(pg.Pos, "a página %s mostra a coluna %s, mas %s não tem esse campo", pg.Name, c, e.Plural)
		}
		if fd.Hidden || fd.IsSecret() || fd.Private {
			return r.errAt(pg.Pos, "a coluna %s de %s é privada ou secreta e não aparece em tabelas", c, e.Plural)
		}
	}
	return nil
}

// sameField: two declarations of a field that mean the same (position aside).
func sameField(a, b *ast.Field) bool {
	x, y := *a, *b
	x.Pos, y.Pos = diagnostics.Position{}, diagnostics.Position{}
	return reflect.DeepEqual(x, y)
}

// where shows a declaration's origin: file:line and, inside a block, its path.
func where(p diagnostics.Position) string {
	out := fmt.Sprintf("%s:%d", p.File, p.Line)
	if p.File == "" {
		out = fmt.Sprintf("linha %d", p.Line)
	}
	if p.Context != "" {
		out += " (" + p.Context + ")"
	}
	return out
}

// belongsTo: e is c, belongs to c, or is c's membership.
func (r *resolver) belongsTo(e, c *ast.Entity) bool {
	if e == c {
		return true
	}
	for _, t := range e.Parents {
		if t == c.Singular {
			return true
		}
	}
	return e.Singular == r.app.MemberModel && c.HasMembers
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
