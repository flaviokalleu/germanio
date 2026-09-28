package servidor

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/parser"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Generic identity for `tenha login`: sessions, access tokens, OAuth
// password grant, lockout and sign-up. The app declares what; this file is
// the how, identical for every Germanio application.

var errInvalidCredential = errors.New("credencial inválida")

// identify resolves the person behind a request. A credential that is
// present but invalid is an error (401); no credential means anonymous.
func (s *Servidor) identify(ctx *interp.Context, r *http.Request) (map[string]any, error) {
	app := s.Program.App
	if app == nil || app.Login == nil {
		return nil, nil
	}
	if ctx.Values == nil {
		ctx.Values = map[string]any{}
	}
	if v, ok := ctx.Values["atual"]; ok {
		m, _ := v.(map[string]any)
		return m, nil
	}
	login := app.Login
	raw := ""
	if login.TokenHeader != "" {
		raw = r.Header.Get(login.TokenHeader)
	}
	if raw == "" {
		if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
			raw = strings.TrimSpace(h[7:])
		}
	}
	var user map[string]any
	// HTTP Basic (git clients): the password may be an access token or the
	// person's password (subject to lockout).
	if bu, bp, ok := r.BasicAuth(); ok && raw == "" {
		if login.TokenEntity != "" {
			res, err := s.Interpreter.Op(ctx, login.TokenEntity, "por_segredo", bp)
			if tok, ok := res.(map[string]any); ok && err == nil {
				user = s.loadPerson(ctx, tok[app.LoginEntity+"_id"])
				ctx.Values["token"] = tok
			}
		}
		if user == nil {
			user = s.authenticate(ctx, bu, bp)
			ctx.Values["token"] = "basic"
		}
		if user == nil || !s.active(user) {
			return nil, errInvalidCredential
		}
		ctx.Values["atual"] = user
		return user, nil
	}
	if raw != "" {
		if login.TokenEntity != "" {
			res, err := s.Interpreter.Op(ctx, login.TokenEntity, "por_segredo", raw)
			if tok, ok := res.(map[string]any); ok && err == nil {
				user = s.loadPerson(ctx, tok[app.LoginEntity+"_id"])
				ctx.Values["token"] = tok
			}
		}
		if user == nil && login.OAuthSeconds > 0 {
			if claims := interp.VerificarToken(raw); claims != nil && claims["tipo"] == "oauth" {
				user = s.loadPerson(ctx, claims["pessoa_id"])
			}
		}
		if user == nil || !s.active(user) {
			return nil, errInvalidCredential
		}
	} else if sess := interp.SessaoDaRequisicao(r); sess != nil {
		user = s.loadPerson(ctx, sess["pessoa_id"])
		if user != nil && !s.active(user) {
			user = nil
		}
	}
	ctx.Values["atual"] = user
	return user, nil
}

func (s *Servidor) loadPerson(ctx *interp.Context, id any) map[string]any {
	if id == nil {
		return nil
	}
	res, err := s.Interpreter.Op(ctx, s.Program.App.LoginEntity, "buscar", id)
	if err != nil {
		return nil
	}
	m, _ := res.(map[string]any)
	return m
}

func (s *Servidor) active(user map[string]any) bool {
	l := s.Program.App.Login
	if l.ActiveField == "" {
		return true
	}
	return interpEqual(user[strings.ToLower(l.ActiveField)], l.ActiveValue)
}

func interpEqual(a, b any) bool {
	if ab, ok := a.(bool); ok {
		bb, _ := b.(bool)
		return ab == bb
	}
	return strings.EqualFold(strings.TrimSpace(toStr(a)), toStr(b))
}

func toStr(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return strings.TrimSpace(strings.Trim(strings.ReplaceAll(jsonString(v), `"`, ""), " "))
}

// authenticate checks login + password with lockout. It returns the
// person or nil; failures never reveal whether the login exists.
// findLogin finds the person whose login field (any of login usa …) is
// login, or nil.
func (s *Servidor) findLogin(ctx *interp.Context, login string) map[string]any {
	app := s.Program.App
	if login == "" {
		return nil
	}
	for _, f := range app.Login.Fields {
		v := login
		if f == "email" {
			v = strings.ToLower(strings.TrimSpace(login))
		}
		res, err := s.Interpreter.Op(ctx, app.LoginEntity, "encontrar", map[string]any{f: v})
		if m, ok := res.(map[string]any); ok && err == nil {
			return m
		}
	}
	return nil
}

// ph is the SQL placeholder n for the database driver.
func (s *Servidor) ph(n int) string {
	if s.DB.Driver == "postgres" || s.DB.Driver == "postgresql" {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

func (s *Servidor) authenticate(ctx *interp.Context, login, password string) map[string]any {
	app := s.Program.App
	user := s.findLogin(ctx, login)
	if user == nil {
		s.Interpreter.Op(ctx, app.LoginEntity, "verificar_senha", nil, password)
		return nil
	}
	lock := app.Login.LockAttempts > 0
	now := time.Now().UTC().Format(time.RFC3339)
	if lock {
		if until := toStr(user["bloqueado_ate"]); until != "" && until > now {
			return nil
		}
	}
	ok, _ := s.Interpreter.Op(ctx, app.LoginEntity, "verificar_senha", user, password)
	if ok != true {
		if lock {
			fails := int(asNumber(user["tentativas_falhas"])) + 1
			change := map[string]any{"tentativas_falhas": fails}
			if fails >= app.Login.LockAttempts {
				change = map[string]any{"tentativas_falhas": 0, "bloqueado_ate": time.Now().UTC().Add(time.Duration(app.Login.LockMinutes) * time.Minute).Format(time.RFC3339)}
			}
			s.Interpreter.Op(ctx, app.LoginEntity, "atualizar", user["id"], change)
		}
		return nil
	}
	if !s.active(user) {
		return nil
	}
	if lock {
		s.Interpreter.Op(ctx, app.LoginEntity, "atualizar", user["id"], map[string]any{"tentativas_falhas": 0, "bloqueado_ate": nil})
	}
	return user
}

func asNumber(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int64:
		return float64(x)
	case int:
		return float64(x)
	}
	return 0
}

func (a *intentAPI) mountIdentity(mux *routeMux) {
	app := a.app
	if app.Login == nil {
		return
	}
	le := app.Entities[app.LoginEntity]
	credentials := func(body map[string]any) (string, string) {
		login := first(toStr(body["login"]), toStr(body["username"]), toStr(body["email"]))
		pass := first(toStr(body["senha"]), toStr(body["password"]))
		return login, pass
	}
	isForm := func(r *http.Request) bool {
		return strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
	}
	start := func(ctx *interp.Context, w http.ResponseWriter, user map[string]any) {
		t, _ := interp.AssinarToken(map[string]any{"pessoa_id": user["id"], "csrf": randomCSRF()}, 24*time.Hour)
		http.SetCookie(w, &http.Cookie{Name: interp.SessionCookie, Value: t, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 86400, Secure: ctx.Request.TLS != nil})
	}
	mux.HandleFunc("POST /entrar", func(w http.ResponseWriter, r *http.Request) {
		ctx := &interp.Context{Request: r, Writer: w}
		body, err := readBody(r)
		if err != nil {
			a.failErr(w, r, err)
			return
		}
		user := a.s.authenticate(ctx, first(credentials(body)), func() string { _, p := credentials(body); return p }())
		if user == nil {
			if isForm(r) {
				http.Redirect(w, r, "/entrar?erro=1", http.StatusSeeOther)
				return
			}
			a.fail(w, 401, map[string]string{"pt": "Login ou senha incorretos", "en": "Invalid login or password"}[app.Messages])
			return
		}
		start(ctx, w, user)
		if isForm(r) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		a.json(w, 200, serialize(le, user), nil)
	})
	if app.Login.Recovery {
		a.mountRecovery(mux)
	}
	if app.EmailNotices {
		if _, why := mailerFromEnv(); why != "" {
			fmt.Printf("[germanio] avisos por e-mail indisponíveis: %s\n", why)
		} else if os.Getenv("GERMANIO_URL_PUBLICA") == "" {
			fmt.Println("[germanio] avisos por e-mail sem endereço da aplicação: defina GERMANIO_URL_PUBLICA para o link do e-mail")
		}
	}
	mux.HandleFunc("POST /sair", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: interp.SessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
		if isForm(r) {
			http.Redirect(w, r, "/entrar", http.StatusSeeOther)
			return
		}
		a.json(w, http.StatusNoContent, nil, nil)
	})
	if app.Login.Signup {
		mux.HandleFunc("POST /cadastro", func(w http.ResponseWriter, r *http.Request) {
			ctx := &interp.Context{Request: r, Writer: w}
			body, err := readBody(r)
			if err != nil {
				a.failErr(w, r, err)
				return
			}
			data := map[string]any{}
			for _, f := range le.Model.Fields {
				switch f.Type {
				case ast.FieldTexto, ast.FieldEmail, ast.FieldSenha, ast.FieldTelefone, ast.FieldTextoLongo:
					key := strings.ToLower(f.Name)
					if v, ok := body[key]; ok && !f.Hidden {
						data[key] = v
					} else if f.Type == ast.FieldSenha {
						if v, ok := body["password"]; ok {
							data[key] = v
						}
					}
				}
			}
			res, err := a.in.Op(ctx, le.Singular, "criar", data)
			if err != nil {
				if isForm(r) {
					http.Redirect(w, r, "/cadastro?erro="+urlQuery(interp.Friendly(err)), http.StatusSeeOther)
					return
				}
				a.failErr(w, r, err)
				return
			}
			user := res.(map[string]any)
			if h := le.Hooks["criar"]; h != nil {
				if _, _, err := a.in.RunHook(ctx, h, a.hookVars(user, le, user, body)); err != nil {
					a.in.Op(ctx, le.Singular, "deletar", user["id"])
					a.failErr(w, r, err)
					return
				}
			}
			start(ctx, w, user)
			if isForm(r) {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
			a.json(w, 201, serialize(le, user), nil)
		})
	}
	if app.Login.OAuthSeconds > 0 {
		mux.HandleFunc("POST /oauth/token", func(w http.ResponseWriter, r *http.Request) {
			ctx := &interp.Context{Request: r, Writer: w}
			body, err := readBody(r)
			if err != nil {
				a.failErr(w, r, err)
				return
			}
			if toStr(body["grant_type"]) != "password" {
				a.json(w, 400, map[string]any{"error": "unsupported_grant_type"}, nil)
				return
			}
			user := a.s.authenticate(ctx, toStr(body["username"]), toStr(body["password"]))
			if user == nil {
				a.json(w, 400, map[string]any{"error": "invalid_grant", "error_description": "The provided authorization grant is invalid."}, nil)
				return
			}
			t, _ := interp.AssinarToken(map[string]any{"tipo": "oauth", "pessoa_id": user["id"]}, time.Duration(app.Login.OAuthSeconds)*time.Second)
			a.json(w, 200, map[string]any{"access_token": t, "token_type": "Bearer", "expires_in": app.Login.OAuthSeconds, "created_at": time.Now().Unix()}, nil)
		})
	}
	me := func(a *intentAPI) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := &interp.Context{Request: r, Writer: w}
			user, err := a.s.identify(ctx, r)
			if err != nil || user == nil {
				a.fail(w, 401, a.msg("401", le))
				return
			}
			a.json(w, 200, serialize(le, user), nil)
		}
	}
	mux.HandleFunc("GET /_ge/eu", me(a))
	if le.Integrate != "" {
		ext := *a
		ext.extern = true // same names and state values as the rest of the integration
		mux.HandleFunc("GET "+app.Integration+"/"+parser.Singular(le.Integrate), me(&ext))
	}
}

func randomCSRF() string {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func jsonString(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func urlQuery(s string) string { return url.QueryEscape(s) }

// scopeAllows: a request authenticated by an access token may only do what
// the token's scopes permit. Sessions, OAuth and passwords are not limited.
func (s *Servidor) scopeAllows(ctx *interp.Context, need string) bool {
	l := s.Program.App.Login
	tok, ok := ctx.Values["token"].(map[string]any)
	if !ok || len(l.Scopes) == 0 {
		return true
	}
	for _, sc := range strings.Split(toStr(tok["escopos"]), ",") {
		for _, perm := range l.Scopes[strings.TrimSpace(sc)] {
			if perm == "tudo" || perm == need || (need == "ler" && perm == "escrever") {
				return true
			}
		}
	}
	return false
}
