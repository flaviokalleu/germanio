// Package explicar shows what Germanio inferred from intent phrases, so a
// simple .ge never becomes a black box (ge explain <dado>, ge check).
package explicar

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/diagnostics"
	"github.com/flaviokalleu/germanio/compiler/lexer"
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
	if f.Type == ast.FieldChavePublica {
		// GEP 0032
		t = "chave pública (conferida; DSA e RSA abaixo de 2048 bits recusadas; nunca muda; impressao_digital calculada)"
	}
	parts = append(parts, t)
	if f.TypeInferred && (f.Type == ast.FieldArquivo || f.Type == ast.FieldImagem) {
		// GEP 0014: only a declared type turns a field into a stored file
		parts = append(parts, fmt.Sprintf("tipo pelo nome, guardado como texto (para enviar e baixar o arquivo, declare: %s %s)", strings.ToLower(f.Name), f.Type))
	}
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
		if app.Login.Signup {
			w("  cadastro: qualquer pessoa cria a própria conta (/cadastro); campos como admin nunca são aceitos\n")
		}
		w("  bloqueio: %d senhas erradas seguidas bloqueiam a conta por %d minutos; um endereço que erra 50 logins em 10 minutos espera\n", app.Login.LockAttempts, app.Login.LockMinutes)
		if app.Login.Recovery {
			w("  recuperação de senha: /esqueci envia por e-mail um link de uso único, válido por 1 hora, para o endereço público (GERMANIO_URL_PUBLICA); a resposta não revela se a conta existe; o e-mail vem do ambiente (GERMANIO_SMTP_* ou GERMANIO_CORREIO_PASTA)\n")
		}
		if app.Login.Confirmation {
			w("  confirmação de e-mail (GEP 0031, em teste): quem se cadastra recebe um link de uso único, válido por 24 horas, e só entra depois de confirmar; mudar o e-mail pede nova confirmação; pessoas criadas por administradores já nascem confirmadas (email_confirmado)\n")
		}
		if app.Login.External {
			w("  login com conta externa (GEP 0039, em teste): botão \"Entrar com …\" para um provedor OpenID Connect configurado no servidor (GERMANIO_OIDC_EMISSOR, _CLIENTE, _SEGREDO, _NOME); código com PKCE, state e nonce; id_token conferido pela assinatura (RS256/ES256) e por iss, aud, exp e nonce; a conta externa é reconhecida pelo emissor e pelo sujeito; um e-mail só encontra uma conta existente se o provedor o verificou e este sistema também o confirmou; dois fatores continuam valendo; /conta-externa liga e desliga\n")
		}
		if app.Login.TwoFactor {
			w("  dois fatores (GEP 0032, em teste): cada pessoa pode ligar um código de aplicativo autenticador (TOTP, RFC 6238); ligado, entrar pede o código depois da senha, cada código vale uma vez, erros contam no bloqueio; senha sozinha não serve para git nem oauth (use um token de acesso); 10 códigos de recuperação de uso único; o segredo fica cifrado e nunca é mostrado de novo\n")
		}
	}
	for _, pb := range e.ProtectedBranches {
		if d := app.Entities[pb.Data]; d != nil {
			w("Branches protegidas: as que cada %s nomeia em %s (nome; * vale qualquer texto) só mudam com %s ou superior\n", d.Label, d.Plural, pb.Role)
		}
	}
	if e.History && app.ActivityEntity != "" {
		w("Histórico: cada mudança fica em %s (quem, o quê, quando e quais campos, nunca os valores)\n", app.ActivityEntity)
	}
	if vt := e.ViewThrough; vt != nil {
		w("Visibilidade: cada registro é visto por quem vê o registro que ele descreve (%s, %s); se ele não existe mais, por quem vê %s; senão, só por %s. Ninguém além do administrador muda ou exclui\n", vt.Kind, vt.ID, vt.ParentKind, strings.TrimSuffix(vt.Author, "_id"))
	}
	if len(e.PendingFields) > 0 {
		w("Pendências: quem passa a estar em %s recebe uma pendência (dado %s); quem sai perde as abertas; excluir o registro exclui as pendências\n", strings.Join(e.PendingFields, ", "), app.PendingEntity)
	}
	w("\nCampos:\n")
	for _, f := range e.Model.Fields {
		w("  %-22s %s\n", f.Name, describeField(f))
	}
	if len(e.Model.Renames) > 0 || len(e.Model.Discarded) > 0 {
		w("\nMigração (só o que o programa declara; nada é inferido):\n")
		for _, rn := range e.Model.Renames {
			w("  renomeie %s para %s — a coluna %s passa a se chamar %s, com os dados; num banco novo ou já migrado, não faz nada (%s)\n", rn.From, rn.To, rn.From, rn.To, where(rn.Pos))
		}
		for _, d := range e.Model.Discarded {
			w("  descarte %s — removido de propósito: a aplicação não usa mais; os valores antigos ficam guardados no banco\n", d)
		}
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
			if effects := ast.Effects(h.Body); len(effects) > 0 {
				w("    efeitos externos, depois de salvar, nesta ordem (se a mudança for desfeita, nenhum acontece; se um falhar, a mudança continua salva):\n")
				for i, ef := range effects {
					w("      %d. %s (%s:%d)\n", i+1, ast.EffectKind(ef), ef.Pos.File, ef.Pos.Line)
				}
			}
		}
	}
	if facts := origins(prog, app, e); len(facts) > 0 {
		w("\nDe onde vem cada fato (frase plana equivalente — origem):\n")
		for _, f := range facts {
			w("  %s — %s\n", f.phrase, f.where)
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

type fact struct {
	phrase, where string
	pos           diagnostics.Position
}

// origins lists the declarations about e, flat or hierarchical, with the
// flat phrase each one means and where it was written (file:line and, in a
// data block, its path): nothing Germanio inferred stays hidden.
func origins(prog *ast.Program, app *ast.App, e *ast.Entity) []fact {
	in := prog.Intent
	if in == nil {
		return nil
	}
	is := func(name string) bool { return name != "" && find(app, name) == e }
	var out []fact
	add := func(phrase string, pos diagnostics.Position) {
		where := fmt.Sprintf("%s:%d", filepath.Base(pos.File), pos.Line)
		if pos.Context != "" {
			where += " (" + pos.Context + ")"
		}
		out = append(out, fact{strings.Join(strings.Fields(phrase), " "), where, pos})
	}
	words := func(ts []lexer.Token) string {
		var parts []string
		for _, t := range ts {
			if t.Type == lexer.TokenString {
				parts = append(parts, fmt.Sprintf("%q", t.Value))
			} else {
				parts = append(parts, t.Name())
			}
		}
		return strings.Join(parts, " ")
	}
	for _, fb := range in.FieldBlocks {
		if is(fb.Entity) {
			for _, l := range fb.Lines {
				if len(l) > 0 {
					add(e.Singular+" tem "+words(l), posOf(l[0], fb.Pos))
				}
			}
		}
	}
	for _, r := range in.Relations {
		if is(r.From) {
			opt := ""
			if r.Optional {
				opt = " opcional"
			}
			add(e.Singular+" pertence a "+r.To+opt, r.Pos)
		}
	}
	for _, st := range in.States {
		if is(st.Entity) {
			add(e.Singular+" começa "+st.Initial, st.Pos)
		}
	}
	for _, g := range in.Capabilities {
		if is(g.Role) {
			add(e.Singular+" pode "+strings.TrimSpace(g.Verb+" "+strings.ReplaceAll(g.Target, "_", " ")), g.Pos)
		}
	}
	for _, g := range in.Grants {
		target := strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(g.Target, "branch_padrao_dos_"), "branch_padrao_das_"), "branch_padrao_")
		only := ""
		if g.Only {
			only = "somente "
		}
		switch {
		case is(g.Role) && (g.Target == "" || g.Verb == "ser"):
			add(e.Singular+" pode "+strings.TrimSpace(g.Verb+" "+strings.ReplaceAll(g.Target, "_", " ")), g.Pos)
		case is(target):
			add(only+g.Role+" pode "+strings.ReplaceAll(g.Verb, "_", " ")+" "+strings.ReplaceAll(g.Target, "_", " "), g.Pos)
		}
	}
	for _, pm := range in.Permits {
		if is(pm.Target) {
			by := ""
			if len(pm.By) > 0 {
				by = " por " + strings.Join(pm.By, ", ")
			}
			add("permita "+pm.Verb+" "+e.Plural+by, pm.Pos)
		}
	}
	for _, it := range in.Integrations {
		if is(it.Target) {
			as := ""
			if it.As != "" {
				as = fmt.Sprintf(" como %q", it.As)
			}
			add("disponibilize "+e.Plural+" para integração"+as, it.Pos)
		}
	}
	for _, c := range in.Creators {
		if is(c.Entity) {
			add("quem cria "+e.Singular+" vira "+c.Role, c.Pos)
		}
	}
	for _, m := range in.MinRoles {
		if is(m.Entity) {
			add("todo "+e.Singular+" precisa ter pelo menos um "+m.Role, m.Pos)
		}
	}
	for _, ro := range in.ReadOnly {
		if is(ro.Entity) {
			add(e.Singular+" "+ro.Flag+" é somente leitura", ro.Pos)
		}
	}
	for _, f := range in.Finals {
		if is(f.Entity) {
			add(e.Singular+" "+f.Initial+" é final", f.Pos)
		}
	}
	for _, c := range in.Ceilings {
		if is(c.Entity) {
			add(e.Singular+" não pode ser mais visível que o "+c.Parent, c.Pos)
		}
	}
	for _, v := range in.Visibility {
		if is(v.Entity) {
			add(e.Singular+" "+v.Flag+" pode ser vista por "+strings.Join(v.Who, ", "), v.Pos)
		}
	}
	for _, m := range in.Memberships {
		if is(m.Entity) && m.InheritFrom != "" {
			add(e.Singular+" herda membros do "+m.InheritFrom, m.Pos)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].pos.File != out[j].pos.File {
			return out[i].pos.File < out[j].pos.File
		}
		return out[i].pos.Line < out[j].pos.Line
	})
	return out
}

// posOf: the position of a field line (its first token), in the file and
// block of the declaration that holds it.
func posOf(t lexer.Token, decl diagnostics.Position) diagnostics.Position {
	decl.Line = t.Line
	return decl
}
