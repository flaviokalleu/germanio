package servidor

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/runtime/banco"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
	"github.com/flaviokalleu/germanio/runtime/oidc"
)

// Sign-in with an external account (docs/gep/0039-login-com-conta-externa.md,
// em teste): OpenID Connect, Authorization Code flow with PKCE (S256), state
// and nonce, ID tokens verified against the provider's keys. The provider
// is configuration of the server (GERMANIO_OIDC_*), never of the program;
// this file knows the protocol, not any provider.
//
// Who the external account is: the pair (issuer, subject), never the
// e-mail. An e-mail only finds an existing person when the provider says it
// is verified AND this system confirmed it too (tenha confirmação de
// e-mail); otherwise the person signs in with the password and links the
// external account herself, so nobody takes over an account just by
// controlling an address at another service. People with the second factor
// on still type their code.

const (
	externalPending = "_germanio_entradas_externas"
	externalLinks   = "_germanio_contas_externas"
	externalTTL     = 10 * time.Minute
	externalCookie  = "ge_externo"
	externalStart   = "/entrar/externo"
	externalReturn  = "/entrar/externo/retorno"
	externalAccount = "/conta-externa"
)

const (
	msgExternalOff      = "A entrada com conta externa ainda não está disponível neste sistema."
	msgExternalMismatch = "Este retorno do provedor não pertence a uma entrada começada neste navegador. Comece de novo pelo botão de entrar."
	msgExternalUsed     = "Este retorno do provedor já foi usado ou venceu. Comece de novo pelo botão de entrar."
	msgExternalTaken    = "Já existe uma conta aqui com este e-mail. Entre com a sua senha e ligue a conta externa em " + externalAccount + ": assim ninguém toma uma conta só por usar o mesmo e-mail em outro serviço."
)

// externalLogin is the configured provider.
type externalLogin struct {
	provider *oidc.Provider
	name     string // shown on the button: "Entrar com <name>"
	secure   bool   // the public address is https: cookies are Secure
}

// mountExternal prepares the tables and reads the configuration. Without
// it the application starts, the button does not appear and the log says
// what is missing.
func (a *intentAPI) mountExternal() {
	a.s.DB.DB.Exec(`CREATE TABLE IF NOT EXISTS ` + externalPending + ` (id ` + a.idColumn() + `, hash TEXT NOT NULL UNIQUE, verificador TEXT NOT NULL, nonce TEXT NOT NULL, pessoa_id INTEGER NOT NULL DEFAULT 0, expira_em TEXT NOT NULL)`)
	a.s.DB.DB.Exec(`CREATE TABLE IF NOT EXISTS ` + externalLinks + ` (id ` + a.idColumn() + `, emissor TEXT NOT NULL, sujeito TEXT NOT NULL, pessoa_id INTEGER NOT NULL, criado_em TEXT NOT NULL, UNIQUE (emissor, sujeito), UNIQUE (emissor, pessoa_id))`)
	issuer := strings.TrimSpace(os.Getenv("GERMANIO_OIDC_EMISSOR"))
	client := strings.TrimSpace(os.Getenv("GERMANIO_OIDC_CLIENTE"))
	public := strings.TrimSuffix(strings.TrimSpace(os.Getenv("GERMANIO_URL_PUBLICA")), "/")
	var missing []string
	for name, v := range map[string]string{"GERMANIO_OIDC_EMISSOR": issuer, "GERMANIO_OIDC_CLIENTE": client, "GERMANIO_URL_PUBLICA": public} {
		if v == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		fmt.Printf("[germanio] entrada com conta externa indisponível: defina %s\n", strings.Join(sortedStrings(missing), ", "))
		return
	}
	pu, err := url.Parse(public)
	if err != nil || pu.Host == "" || (pu.Scheme != "https" && pu.Scheme != "http") {
		fmt.Printf("[germanio] entrada com conta externa indisponível: GERMANIO_URL_PUBLICA não é um endereço (%q)\n", public)
		return
	}
	name := strings.TrimSpace(os.Getenv("GERMANIO_OIDC_NOME"))
	if name == "" {
		if u, err := url.Parse(issuer); err == nil && u.Host != "" {
			name = u.Hostname()
		} else {
			name = "conta externa"
		}
	}
	a.external = &externalLogin{
		name:   name,
		secure: pu.Scheme == "https",
		provider: oidc.New(oidc.Config{
			Issuer:   issuer,
			ClientID: client,
			// the secret comes only from the environment and is never logged
			ClientSecret: os.Getenv("GERMANIO_OIDC_SEGREDO"),
			// built from the declared public address, never from Host
			RedirectURI: public + externalReturn,
			AllowHTTP:   os.Getenv("GERMANIO_PERMITIR_REDE_LOCAL") == "1",
		}),
	}
	fmt.Printf("[germanio] entrada com conta externa: %s (%s)\n", name, issuer)
}

func sortedStrings(in []string) []string {
	for i := 1; i < len(in); i++ {
		for j := i; j > 0 && in[j] < in[j-1]; j-- {
			in[j], in[j-1] = in[j-1], in[j]
		}
	}
	return in
}

type externalRequest struct {
	verifier, nonce string
	person          int64 // linking from a session; 0 when signing in
}

// claimExternal uses up the pending sign-in of state: it works once, even
// when the same return arrives twice at once, and only before it expires.
func (a *intentAPI) claimExternal(state string) (externalRequest, bool) {
	var req externalRequest
	var expira string
	h := hashToken(state)
	if a.s.DB.DB.QueryRow(fmt.Sprintf(`SELECT verificador, nonce, pessoa_id, expira_em FROM %s WHERE hash = %s`, externalPending, a.s.ph(1)), h).Scan(&req.verifier, &req.nonce, &req.person, &expira) != nil {
		return req, false
	}
	res, err := a.s.DB.Executar(fmt.Sprintf(`DELETE FROM %s WHERE hash = %s`, externalPending, a.s.ph(1)), h)
	if err != nil {
		return req, false
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return req, false
	}
	t, err := time.Parse(time.RFC3339, expira)
	return req, err == nil && time.Now().Before(t)
}

func (a *intentAPI) linkedPerson(issuer, subject string) int64 {
	var person int64
	a.s.DB.DB.QueryRow(fmt.Sprintf(`SELECT pessoa_id FROM %s WHERE emissor = %s AND sujeito = %s`, externalLinks, a.s.ph(1), a.s.ph(2)), issuer, subject).Scan(&person)
	return person
}

func (a *intentAPI) link(db *banco.Banco, issuer, subject string, person any) error {
	_, err := db.Executar(fmt.Sprintf(`INSERT INTO %s (emissor, sujeito, pessoa_id, criado_em) VALUES (%s, %s, %s, %s)`, externalLinks, a.s.ph(1), a.s.ph(2), a.s.ph(3), a.s.ph(4)),
		issuer, subject, person, time.Now().UTC().Format(time.RFC3339))
	return err
}

var errExternalRefused = errors.New("conta externa recusada")

// externalPerson finds (or, when the system lets people sign up, creates)
// the person of a verified external account. It returns the person, or the
// status and the message of the refusal.
func (a *intentAPI) externalPerson(ctx *interp.Context, c *oidc.Claims) (map[string]any, int, string) {
	app := a.app
	le := app.Entities[app.LoginEntity]
	if id := a.linkedPerson(c.Issuer, c.Subject); id != 0 {
		if user := a.s.loadPerson(ctx, id); user != nil {
			return user, 0, ""
		}
		// the person was deleted: the old link means nothing any more
		a.s.DB.Executar(fmt.Sprintf(`DELETE FROM %s WHERE emissor = %s AND sujeito = %s`, externalLinks, a.s.ph(1), a.s.ph(2)), c.Issuer, c.Subject)
	}
	if c.Email == "" {
		return nil, http.StatusBadRequest, "O provedor não informou o e-mail desta conta. Peça a quem administra o provedor para liberar o e-mail (escopo email) a este sistema."
	}
	if !c.EmailVerified {
		// never trust an address the provider itself did not verify: not
		// to find an account, not to create one
		return nil, http.StatusForbidden, "O provedor não confirma que este e-mail é seu. Confirme o e-mail na sua conta de lá e tente de novo."
	}
	field := emailField(le)
	email := strings.ToLower(strings.TrimSpace(c.Email))
	found, err := a.in.Op(ctx, le.Singular, "encontrar", map[string]any{field: email})
	if existing, ok := found.(map[string]any); ok && err == nil {
		// only an address this system confirmed too proves that the account
		// belongs to whoever controls the address (see the GEP)
		if !app.Login.Confirmation || a.s.unconfirmed(existing) {
			return nil, http.StatusConflict, msgExternalTaken
		}
		if err := a.link(a.s.DB, c.Issuer, c.Subject, existing["id"]); err != nil {
			return nil, http.StatusConflict, "Esta conta já está ligada a outra conta deste provedor."
		}
		return existing, 0, ""
	}
	if !app.Login.Signup {
		return nil, http.StatusForbidden, "Este sistema não cria contas pela entrada externa. Peça a quem administra o sistema para criar a sua conta; depois, entre com a senha e ligue a conta externa em " + externalAccount + "."
	}
	return a.createExternal(ctx, le, c, email)
}

// createExternal creates the person of an external account, like a sign-up:
// the same validations and the same hook. The e-mail is the one the provider
// verified; the password is random (the person can create one through
// password recovery).
func (a *intentAPI) createExternal(ctx *interp.Context, le *ast.Entity, c *oidc.Claims, email string) (map[string]any, int, string) {
	field := emailField(le)
	local := email
	if i := strings.IndexByte(email, '@'); i > 0 {
		local = email[:i]
	}
	data := map[string]any{field: email}
	logins := map[string]bool{}
	for _, f := range a.app.Login.Fields {
		logins[strings.ToLower(f)] = true
	}
	for _, f := range le.Model.Fields {
		key := strings.ToLower(f.Name)
		switch {
		case key == field || f.System || f.Hidden:
		case f.Type == ast.FieldSenha:
			pass, err := newToken()
			if err != nil {
				return nil, http.StatusInternalServerError, err.Error()
			}
			data[key] = pass
		case logins[key] && f.Type == ast.FieldTexto:
			data[key] = a.freeLogin(ctx, le, key, first(c.PreferredUsername, local))
		case f.Type == ast.FieldTexto && (key == "nome" || key == "name" || key == "nome_completo"):
			data[key] = first(c.Name, c.PreferredUsername, local)
		}
	}
	if a.app.Login.Confirmation {
		data["email_confirmado"] = true // verified by the provider (see the GEP)
	}
	var user map[string]any
	err := a.s.DB.EmTransacao(func(tx *banco.Banco) error {
		c2 := &interp.Context{Request: ctx.Request, Writer: ctx.Writer, DB: tx}
		res, err := a.in.Op(c2, le.Singular, "criar", data)
		if err != nil {
			return err
		}
		user, _ = res.(map[string]any)
		if user == nil {
			return errExternalRefused
		}
		return a.link(tx, c.Issuer, c.Subject, user["id"])
	})
	if err != nil {
		return nil, http.StatusBadRequest, "Não foi possível criar a sua conta com os dados do provedor: " + interp.Friendly(err)
	}
	if h := le.Hooks["criar"]; h != nil {
		if _, _, err := a.in.RunHook(ctx, h, a.hookVars(user, le, user, map[string]any{})); err != nil {
			a.in.Op(ctx, le.Singular, "deletar", user["id"])
			a.s.DB.Executar(fmt.Sprintf(`DELETE FROM %s WHERE pessoa_id = %s`, externalLinks, a.s.ph(1)), user["id"])
			return nil, http.StatusBadRequest, "Não foi possível criar a sua conta: " + interp.Friendly(err)
		}
	}
	return user, 0, ""
}

// freeLogin turns a name from the provider into a login nobody uses yet:
// letters, digits, _ . - (the rule of addresses), numbered when taken.
func (a *intentAPI) freeLogin(ctx *interp.Context, le *ast.Entity, key, want string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(want) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '.', r == '-':
			b.WriteRune(r)
		}
	}
	base := strings.TrimRight(strings.TrimLeft(b.String(), ".-"), ".")
	if len(base) > 40 {
		base = base[:40]
	}
	if base == "" {
		base = "pessoa"
	}
	for i := 0; i < 100; i++ {
		cand := base
		if i > 0 {
			cand = base + strconv.Itoa(i+1)
		}
		if found, err := a.in.Op(ctx, le.Singular, "encontrar", map[string]any{key: cand}); err != nil || found == nil {
			return cand
		} else if m, ok := found.(map[string]any); !ok || m == nil {
			return cand
		}
	}
	tail, _ := newToken()
	return base + "-" + strings.ToLower(tail[:6])
}

// ---------- pages ----------

func (ps *pageSite) externalCookie(w http.ResponseWriter, value string, maxAge int) {
	ext := ps.a.external
	// Lax, not Strict: the provider sends the browser back with a top-level
	// navigation from its own site, and this cookie must come along
	http.SetCookie(w, &http.Cookie{Name: externalCookie, Value: value, Path: externalStart, HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: maxAge, Secure: ext != nil && ext.secure})
}

// loginPage draws the sign-in page with a problem (or a notice).
func (ps *pageSite) loginPage(w http.ResponseWriter, r *http.Request, status int, problem, flash string) {
	app := ps.a.app
	v := ps.base(r, "Entrar")
	if problem != "" {
		v.Error = problem
	}
	if flash != "" {
		v.Flash = flash
	}
	data := map[string]any{"Signup": v.Signup, "Label": strings.Join(app.Login.Fields, " ou "), "Recovery": app.Login.Recovery}
	if ext := ps.a.external; ext != nil {
		data["External"] = ext.name
	}
	v.Body = htmlOf(loginTpl, data)
	ps.render(w, v, status)
}

func (ps *pageSite) externalPages(mux *routeMux) {
	a := ps.a
	mux.HandleFunc("POST "+externalStart, func(w http.ResponseWriter, r *http.Request) {
		if a.external == nil {
			ps.loginPage(w, r, http.StatusServiceUnavailable, msgExternalOff, "")
			return
		}
		r.ParseForm()
		var person int64
		if r.PostForm.Get("ligar") == "1" {
			// linking changes the signed-in account: a browser session and
			// its CSRF proof
			_, user, ok := ps.sessionPerson(w, r, true)
			if !ok {
				return
			}
			person = pid(user["id"])
		}
		state, err1 := oidc.Random(32)
		nonce, err2 := oidc.Random(32)
		verifier, err3 := oidc.Random(32) // 43 characters (RFC 7636 §4.1)
		if err := errors.Join(err1, err2, err3); err != nil {
			a.failErr(w, r, err)
			return
		}
		now := time.Now().UTC()
		a.s.DB.Executar(fmt.Sprintf(`DELETE FROM %s WHERE expira_em < %s`, externalPending, a.s.ph(1)), now.Format(time.RFC3339))
		if _, err := a.s.DB.Executar(fmt.Sprintf(`INSERT INTO %s (hash, verificador, nonce, pessoa_id, expira_em) VALUES (%s, %s, %s, %s, %s)`, externalPending, a.s.ph(1), a.s.ph(2), a.s.ph(3), a.s.ph(4), a.s.ph(5)),
			hashToken(state), verifier, nonce, person, now.Add(externalTTL).Format(time.RFC3339)); err != nil {
			a.failErr(w, r, err)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		dest, err := a.external.provider.AuthURL(ctx, state, nonce, verifier)
		if err != nil {
			a.s.DB.Executar(fmt.Sprintf(`DELETE FROM %s WHERE hash = %s`, externalPending, a.s.ph(1)), hashToken(state))
			fmt.Printf("[germanio] entrada com conta externa: %v\n", err)
			ps.loginPage(w, r, http.StatusBadGateway, "Não foi possível falar com "+a.external.name+": "+err.Error(), "")
			return
		}
		ps.externalCookie(w, state, int(externalTTL.Seconds()))
		http.Redirect(w, r, dest, http.StatusSeeOther)
	})

	mux.HandleFunc("GET "+externalReturn, func(w http.ResponseWriter, r *http.Request) {
		refuse := func(status int, msg string) {
			ps.loginPage(w, r, status, msg, "")
		}
		if a.external == nil {
			refuse(http.StatusServiceUnavailable, msgExternalOff)
			return
		}
		q := r.URL.Query()
		state := q.Get("state")
		ck, err := r.Cookie(externalCookie)
		ps.externalCookie(w, "", -1)
		// the state must be the one this browser started with (login CSRF:
		// someone else's return link carries someone else's state)
		if state == "" || err != nil || subtle.ConstantTimeCompare([]byte(ck.Value), []byte(state)) != 1 {
			refuse(http.StatusBadRequest, msgExternalMismatch)
			return
		}
		pending, ok := a.claimExternal(state)
		if !ok {
			refuse(http.StatusBadRequest, msgExternalUsed)
			return
		}
		if e := q.Get("error"); e != "" {
			if e == "access_denied" {
				refuse(http.StatusBadRequest, "A entrada foi cancelada em "+a.external.name+".")
				return
			}
			refuse(http.StatusBadRequest, a.external.name+" recusou a entrada ("+short(e)+").")
			return
		}
		code := q.Get("code")
		if code == "" {
			refuse(http.StatusBadRequest, msgExternalUsed)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		prov := a.external.provider
		tokens, err := prov.Exchange(ctx, code, pending.verifier)
		var claims *oidc.Claims
		if err == nil {
			claims, err = prov.Verify(ctx, tokens.IDToken, pending.nonce, time.Now())
		}
		if err == nil && claims.Email == "" {
			err = prov.UserInfo(ctx, tokens.AccessToken, claims)
		}
		if err != nil {
			fmt.Printf("[germanio] entrada com conta externa recusada: %v\n", err)
			refuse(http.StatusBadRequest, "Não foi possível conferir a resposta de "+a.external.name+": "+err.Error())
			return
		}
		ictx := &interp.Context{Request: r, Writer: w}
		if pending.person != 0 {
			ps.finishLink(w, r, pending.person, claims)
			return
		}
		user, status, msg := a.externalPerson(ictx, claims)
		if user == nil {
			refuse(status, msg)
			return
		}
		if !a.s.active(user) {
			refuse(http.StatusForbidden, "Esta conta não pode entrar agora.")
			return
		}
		if a.s.unconfirmed(user) {
			refuse(http.StatusForbidden, msgConfirmFirst)
			return
		}
		if a.s.secondFactorOn(user) {
			// the person turned the second factor on here: the provider's
			// sign-in does not replace it
			token, err := a.newChallenge(user)
			if err != nil {
				a.failErr(w, r, err)
				return
			}
			setChallengeCookie(w, r, token)
			// drawn here, not redirected: the challenge cookie (Strict) is
			// then sent with the form, a same-site post
			v := ps.base(r, "Código de verificação")
			v.Body = htmlOf(codeLoginTpl, nil)
			ps.render(w, v, http.StatusOK)
			return
		}
		startSession(ictx, w, user)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	mux.HandleFunc("GET "+externalAccount, func(w http.ResponseWriter, r *http.Request) {
		_, user, ok := ps.sessionPerson(w, r, false)
		if !ok {
			return
		}
		ps.accountPage(w, r, user, http.StatusOK, "", "")
	})
	mux.HandleFunc("POST "+externalAccount+"/desligar", func(w http.ResponseWriter, r *http.Request) {
		_, user, ok := ps.sessionPerson(w, r, true)
		if !ok {
			return
		}
		a.s.DB.Executar(fmt.Sprintf(`DELETE FROM %s WHERE pessoa_id = %s`, externalLinks, a.s.ph(1)), user["id"])
		http.Redirect(w, r, externalAccount+"?ok="+urlQuery("Conta externa desligada."), http.StatusSeeOther)
	})
}

// finishLink links the external account to the person who started linking
// while signed in; the same browser must still be signed in as her.
func (ps *pageSite) finishLink(w http.ResponseWriter, r *http.Request, person int64, c *oidc.Claims) {
	a := ps.a
	ctx := &interp.Context{Request: r, Writer: w}
	user, _ := a.s.identify(ctx, r)
	if user == nil || pid(user["id"]) != person || interp.SessaoDaRequisicao(r) == nil {
		ps.loginPage(w, r, http.StatusForbidden, "Entre de novo com a sua conta e ligue a conta externa outra vez.", "")
		return
	}
	switch other := a.linkedPerson(c.Issuer, c.Subject); {
	case other == person:
		http.Redirect(w, r, externalAccount+"?ok="+urlQuery("Esta conta externa já está ligada à sua conta."), http.StatusSeeOther)
		return
	case other != 0:
		ps.accountPage(w, r, user, http.StatusConflict, "", "Esta conta de "+a.external.name+" já está ligada a outra pessoa daqui.")
		return
	}
	if err := a.link(a.s.DB, c.Issuer, c.Subject, person); err != nil {
		ps.accountPage(w, r, user, http.StatusConflict, "", "Você já ligou outra conta de "+a.external.name+". Desligue-a antes de ligar esta.")
		return
	}
	http.Redirect(w, r, externalAccount+"?ok="+urlQuery("Conta externa ligada. Agora você também pode entrar com "+a.external.name+"."), http.StatusSeeOther)
}

func (ps *pageSite) accountPage(w http.ResponseWriter, r *http.Request, user map[string]any, status int, flash, problem string) {
	a := ps.a
	v := ps.base(r, "Conta externa")
	if flash != "" {
		v.Flash = flash
	}
	if problem != "" {
		v.Error = problem
	}
	var since string
	linked := a.s.DB.DB.QueryRow(fmt.Sprintf(`SELECT criado_em FROM %s WHERE pessoa_id = %s`, externalLinks, a.s.ph(1)), user["id"]).Scan(&since) == nil
	name := ""
	if a.external != nil {
		name = a.external.name
	}
	v.Body = htmlOf(externalAccountTpl, map[string]any{"Linked": linked, "Since": since, "Name": name, "CSRF": v.CSRF, "Available": a.external != nil})
	ps.render(w, v, status)
}

// short keeps a provider's word short before it is shown.
func short(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 80 {
		s = s[:80]
	}
	return s
}

var externalAccountTpl = tpl(`<div class="caixa"><h3>Conta externa</h3>
{{if .Linked}}<p>Sua conta está ligada a uma conta de <strong>{{.Name}}</strong> desde {{.Since}}: você pode entrar com ela.</p>
<form method="post" action="/conta-externa/desligar"><input type="hidden" name="_csrf" value="{{.CSRF}}"><p>Depois de desligar, entre com a senha (se nunca criou uma, use "Esqueci minha senha").</p><button class="perigo">Desligar</button></form>
{{else if .Available}}<p>Ligue uma conta de <strong>{{.Name}}</strong> para entrar com ela, sem digitar a senha daqui.</p>
<form method="post" action="/entrar/externo"><input type="hidden" name="_csrf" value="{{.CSRF}}"><input type="hidden" name="ligar" value="1"><button>Ligar conta de {{.Name}}</button></form>
{{else}}<p>A entrada com conta externa ainda não está disponível neste sistema.</p>{{end}}
</div>`)
