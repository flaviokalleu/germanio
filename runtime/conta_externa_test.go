package runtime

import (
	"crypto/rand"
	"crypto/rsa"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/flaviokalleu/germanio/runtime/oidc/oidctest"
)

const appPublica = "http://app.example"

// externalEnv configures the fake provider as the server's provider.
func externalEnv(t *testing.T, fake *oidctest.Provider) {
	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "1") // the fake provider runs on 127.0.0.1
	t.Setenv("GERMANIO_OIDC_EMISSOR", fake.Issuer())
	t.Setenv("GERMANIO_OIDC_CLIENTE", oidctest.ClientID)
	t.Setenv("GERMANIO_OIDC_SEGREDO", oidctest.ClientSecret)
	t.Setenv("GERMANIO_OIDC_NOME", "Provedor Teste")
	t.Setenv("GERMANIO_URL_PUBLICA", appPublica)
}

type answer struct {
	status   int
	body     string
	location string
}

func (c *client) form(method, path string, values url.Values) answer {
	c.t.Helper()
	req, _ := http.NewRequest(method, c.base+path, strings.NewReader(values.Encode()))
	if method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return answer{resp.StatusCode, string(raw), resp.Header.Get("Location")}
}

// startExternal presses "Entrar com …" (or "Ligar", with values) and lets the
// fake provider approve; it returns the provider's return address, as a path
// of this server.
func startExternal(t *testing.T, c *client, values url.Values) string {
	t.Helper()
	got := c.form("POST", "/entrar/externo", values)
	if got.status != http.StatusSeeOther {
		t.Fatalf("começar a entrada externa: %d %s", got.status, got.body)
	}
	noFollow := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := noFollow.Get(got.location)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	back := resp.Header.Get("Location")
	if !strings.HasPrefix(back, appPublica+"/entrar/externo/retorno?") {
		t.Fatalf("o provedor devolveu para %q (o endereço de retorno vem de GERMANIO_URL_PUBLICA)", back)
	}
	return strings.TrimPrefix(back, appPublica)
}

// signInExternal does the whole sign-in and returns the server's answer to
// the provider's return.
func signInExternal(t *testing.T, c *client) answer {
	t.Helper()
	return c.form("GET", startExternal(t, c, nil), nil)
}

func withParam(path, key, value string) string {
	u, _ := url.Parse(path)
	q := u.Query()
	q.Set(key, value)
	u.RawQuery = q.Encode()
	return u.String()
}

func externalRows(t *testing.T, app *App, query string, args ...any) int {
	t.Helper()
	var n int
	if err := app.DB.DB.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return n
}

// Sign-in with an external account (GEP 0039): OpenID Connect with PKCE,
// state and nonce; the ID token is checked before anything is believed; a
// new person is created only from a verified e-mail; the external account
// is recognized by issuer and subject afterwards.
func TestContaExterna(t *testing.T) {
	fake := oidctest.New(t)
	externalEnv(t, fake)
	t.Setenv("GERMANIO_CORREIO_PASTA", t.TempDir())
	t.Setenv("GERMANIO_SEGREDO", strings.Repeat("k", 40))
	app, c := loadApp(t, "testdata/conta_externa/app.ge")

	if page := c.form("GET", "/entrar", nil); !strings.Contains(page.body, "Entrar com Provedor Teste") {
		t.Fatalf("página de entrar sem o botão:\n%s", page.body)
	}

	// happy path: a new person, created from the verified e-mail
	fake.Set(oidctest.Identity{Subject: "s-bia", Email: "Bia@X.com", EmailVerified: true, Name: "Bia Souza", PreferredUsername: "bia"})
	bia := newClient(t, c.base)
	if got := signInExternal(t, bia); got.status != http.StatusSeeOther || got.location != "/" {
		t.Fatalf("entrar com conta externa: %d %s", got.status, got.body)
	}
	me := bia.expect("GET", "/_ge/eu", nil, 200)
	if me["email"] != "bia@x.com" || me["username"] != "bia" || me["nome"] != "Bia Souza" || me["email_confirmado"] != true {
		t.Fatalf("pessoa criada: %v", me)
	}
	bia.csrf = csrfFromCookie(t, bia) // the same session (and CSRF proof) as a password sign-in
	if n := externalRows(t, app, `SELECT COUNT(*) FROM _germanio_contas_externas WHERE emissor = ? AND sujeito = 's-bia'`, fake.Issuer()); n != 1 {
		t.Fatalf("ligação: %d", n)
	}
	// later: recognized by the subject, even with another e-mail there
	fake.Set(oidctest.Identity{Subject: "s-bia", Email: "bia.nova@x.com", EmailVerified: true, PreferredUsername: "bia"})
	again := newClient(t, c.base)
	if got := signInExternal(t, again); got.status != http.StatusSeeOther {
		t.Fatalf("segunda entrada: %d %s", got.status, got.body)
	}
	if me := again.expect("GET", "/_ge/eu", nil, 200); me["email"] != "bia@x.com" {
		t.Fatalf("segunda entrada caiu em outra pessoa: %v", me)
	}
	// the same preferred name later becomes another free login
	fake.Set(oidctest.Identity{Subject: "s-bia2", Email: "bia2@x.com", EmailVerified: true, PreferredUsername: "bia"})
	other := newClient(t, c.base)
	signInExternal(t, other)
	if me := other.expect("GET", "/_ge/eu", nil, 200); me["username"] != "bia2" {
		t.Fatalf("login livre: %v", me)
	}
	people := externalRows(t, app, `SELECT COUNT(*) FROM usuario`)

	refused := func(name string, c *client, got answer, status int, want string) {
		t.Helper()
		if got.status != status || !strings.Contains(got.body, want) {
			t.Fatalf("%s: esperado %d com %q, veio %d:\n%s", name, status, want, got.status, got.body)
		}
		c.expect("GET", "/_ge/eu", nil, 401)
	}

	// state: a tampered one, and someone else's return in this browser
	fake.Set(oidctest.Identity{Subject: "s-bia", Email: "bia@x.com", EmailVerified: true})
	eve := newClient(t, c.base)
	path := startExternal(t, eve, nil)
	refused("state trocado", eve, eve.form("GET", withParam(path, "state", "outro"), nil), 400, "não pertence a uma entrada começada neste navegador")
	victim := newClient(t, c.base)
	path = startExternal(t, eve, nil)
	refused("retorno de outro navegador", victim, victim.form("GET", path, nil), 400, "não pertence")

	// replayed return and replayed code
	ok := newClient(t, c.base)
	path = startExternal(t, ok, nil)
	if got := ok.form("GET", path, nil); got.status != http.StatusSeeOther {
		t.Fatalf("entrada: %d %s", got.status, got.body)
	}
	u, _ := url.Parse(path)
	replay := newClient(t, c.base)
	replay.http.Jar.SetCookies(mustParse(t, c.base+"/entrar/externo"), []*http.Cookie{{Name: "ge_externo", Value: u.Query().Get("state"), Path: "/entrar/externo"}})
	refused("retorno repetido", replay, replay.form("GET", path, nil), 400, "já foi usado")
	fresh := newClient(t, c.base)
	path2 := startExternal(t, fresh, nil)
	calls := fake.TokenCalls
	refused("code repetido", fresh, fresh.form("GET", withParam(path2, "code", u.Query().Get("code")), nil), 400, "code inválido ou já usado")
	if fake.TokenCalls != calls+1 {
		t.Fatal("o code repetido não chegou ao provedor")
	}

	// the ID token: every check refuses on its own
	otherKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	tokens := []struct {
		name   string
		change func(p *oidctest.Provider)
		want   string
	}{
		{"assinatura falsa", func(p *oidctest.Provider) { p.SignWith = otherKey }, "assinatura"},
		{"aud errado", func(p *oidctest.Provider) { p.Tamper = func(m map[string]any) { m["aud"] = "outro-sistema" } }, "aud"},
		{"iss errado", func(p *oidctest.Provider) { p.Tamper = func(m map[string]any) { m["iss"] = "https://mal.example" } }, "emitido por"},
		{"vencido", func(p *oidctest.Provider) {
			p.Tamper = func(m map[string]any) {
				m["exp"] = time.Now().Add(-time.Hour).Unix()
				m["iat"] = time.Now().Add(-2 * time.Hour).Unix()
			}
		}, "venceu"},
		{"nonce errado", func(p *oidctest.Provider) { p.Tamper = func(m map[string]any) { m["nonce"] = "outro" } }, "nonce"},
	}
	for _, tc := range tokens {
		fake.Set(oidctest.Identity{Subject: "s-novo-" + tc.name, Email: "novo@x.com", EmailVerified: true})
		fake.Change(tc.change)
		cl := newClient(t, c.base)
		refused(tc.name, cl, signInExternal(t, cl), 400, tc.want)
	}
	if n := externalRows(t, app, `SELECT COUNT(*) FROM usuario`); n != people {
		t.Fatalf("um token recusado criou pessoas: %d → %d", people, n)
	}

	// e-mails and existing accounts: no takeover
	c.expect("POST", "/cadastro", map[string]any{"username": "ana", "nome": "Ana", "email": "ana@x.com", "senha": "senha-segura-1"}, 201)
	c.expect("POST", "/cadastro", map[string]any{"username": "caio", "nome": "Caio", "email": "caio@x.com", "senha": "senha-segura-1"}, 201)
	app.DB.DB.Exec(`UPDATE usuario SET email_confirmado = 1 WHERE email = 'ana@x.com'`) // Ana confirmed; Caio did not
	links := externalRows(t, app, `SELECT COUNT(*) FROM _germanio_contas_externas`)
	fake.Set(oidctest.Identity{Subject: "s-mal", Email: "ana@x.com", EmailVerified: false})
	mal := newClient(t, c.base)
	refused("e-mail não verificado", mal, signInExternal(t, mal), 403, "não confirma que este e-mail é seu")
	fake.Set(oidctest.Identity{Subject: "s-caio", Email: "caio@x.com", EmailVerified: true})
	cl := newClient(t, c.base)
	refused("conta daqui sem e-mail confirmado", cl, signInExternal(t, cl), 409, "Entre com a sua senha e ligue a conta externa")
	fake.Set(oidctest.Identity{Subject: "s-ninguem", Email: "ninguem@x.com", EmailVerified: false})
	cl = newClient(t, c.base)
	refused("e-mail não verificado e sem conta", cl, signInExternal(t, cl), 403, "não confirma")
	if n := externalRows(t, app, `SELECT COUNT(*) FROM _germanio_contas_externas`); n != links {
		t.Fatalf("uma entrada recusada ligou contas: %d → %d", links, n)
	}
	if n := externalRows(t, app, `SELECT COUNT(*) FROM usuario WHERE email = 'ninguem@x.com'`); n != 0 {
		t.Fatal("e-mail não verificado criou uma conta")
	}
	// verified there and confirmed here: the same person
	fake.Set(oidctest.Identity{Subject: "s-ana", Email: "ana@x.com", EmailVerified: true})
	cl = newClient(t, c.base)
	if got := signInExternal(t, cl); got.status != http.StatusSeeOther {
		t.Fatalf("e-mail verificado e confirmado: %d %s", got.status, got.body)
	}
	if me := cl.expect("GET", "/_ge/eu", nil, 200); me["username"] != "ana" {
		t.Fatalf("ligou a outra pessoa: %v", me)
	}

	// the second factor still applies
	ana := newClient(t, c.base)
	ana.expect("POST", "/entrar", map[string]any{"login": "ana", "senha": "senha-segura-1"}, 200)
	ana.csrf = csrfFromCookie(t, ana)
	secret, _ := enableFactor(t, ana, "senha-segura-1")
	cl = newClient(t, c.base)
	got := signInExternal(t, cl)
	if got.status != 200 || !strings.Contains(got.body, `action="/entrar/codigo"`) {
		t.Fatalf("dois fatores depois da conta externa: %d %s", got.status, got.body)
	}
	cl.expect("GET", "/_ge/eu", nil, 401)
	cl.expect("POST", "/entrar/codigo", map[string]any{"codigo": codeAt(secret, 1)}, 200)
	if me := cl.expect("GET", "/_ge/eu", nil, 200); me["username"] != "ana" {
		t.Fatalf("depois do código: %v", me)
	}

	// linking from a session, with its CSRF proof, whatever the e-mail there
	if got := bia.form("POST", "/entrar/externo", url.Values{"ligar": {"1"}}); got.status != 403 {
		t.Fatalf("ligar sem CSRF: %d %s", got.status, got.body)
	}
	bia.form("POST", "/conta-externa/desligar", url.Values{"_csrf": {bia.csrf}})
	if n := externalRows(t, app, `SELECT COUNT(*) FROM _germanio_contas_externas WHERE sujeito = 's-bia'`); n != 0 {
		t.Fatal("desligar não desligou")
	}
	fake.Set(oidctest.Identity{Subject: "s-bia-trabalho", Email: "bia@empresa.example", EmailVerified: false})
	back := bia.form("GET", startExternal(t, bia, url.Values{"ligar": {"1"}, "_csrf": {bia.csrf}}), nil)
	if back.status != http.StatusSeeOther || !strings.HasPrefix(back.location, "/conta-externa") {
		t.Fatalf("ligar: %d %s %s", back.status, back.location, back.body)
	}
	if page := bia.form("GET", "/conta-externa", nil); !strings.Contains(page.body, "está ligada") {
		t.Fatalf("página da conta externa: %s", page.body)
	}
	cl = newClient(t, c.base)
	signInExternal(t, cl)
	if me := cl.expect("GET", "/_ge/eu", nil, 200); me["email"] != "bia@x.com" {
		t.Fatalf("entrar pela conta ligada: %v", me)
	}
	// that external account cannot be linked to someone else
	other.csrf = csrfFromCookie(t, other)
	taken := other.form("GET", startExternal(t, other, url.Values{"ligar": {"1"}, "_csrf": {other.csrf}}), nil)
	if taken.status != 409 || !strings.Contains(taken.body, "já está ligada a outra pessoa") {
		t.Fatalf("ligar uma conta externa de outra pessoa: %d %s", taken.status, taken.body)
	}
}

func mustParse(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// A system without sign-up and without e-mail confirmation never creates
// people from external accounts, and never links one to an account by the
// e-mail alone: nothing proves that account's address.
func TestContaExternaSistemaFechado(t *testing.T) {
	fake := oidctest.New(t)
	externalEnv(t, fake)
	t.Setenv("GERMANIO_ADMIN_SENHA", "senha-do-root-1")
	app, c := loadApp(t, "testdata/conta_externa_fechada/app.ge")
	fake.Set(oidctest.Identity{Subject: "s-root", Email: "root@x.com", EmailVerified: true})
	if got := signInExternal(t, c); got.status != 409 || !strings.Contains(got.body, "ligue a conta externa") {
		t.Fatalf("conta existente sem confirmação declarada: %d %s", got.status, got.body)
	}
	fake.Set(oidctest.Identity{Subject: "s-novo", Email: "novo@x.com", EmailVerified: true})
	cl := newClient(t, c.base)
	if got := signInExternal(t, cl); got.status != 403 || !strings.Contains(got.body, "não cria contas") {
		t.Fatalf("sistema sem cadastro: %d %s", got.status, got.body)
	}
	if n := externalRows(t, app, `SELECT COUNT(*) FROM usuario`); n != 1 {
		t.Fatalf("pessoas: %d", n)
	}
	cl.expect("GET", "/_ge/eu", nil, 401)
}

// Without the configuration the application starts, the button does not
// appear, and starting says why.
func TestContaExternaSemConfiguracao(t *testing.T) {
	t.Setenv("GERMANIO_OIDC_EMISSOR", "")
	t.Setenv("GERMANIO_CORREIO_PASTA", t.TempDir())
	t.Setenv("GERMANIO_URL_PUBLICA", appPublica)
	_, c := loadApp(t, "testdata/conta_externa/app.ge")
	if page := c.form("GET", "/entrar", nil); strings.Contains(page.body, "Entrar com") {
		t.Fatal("botão sem provedor configurado")
	}
	if got := c.form("POST", "/entrar/externo", nil); got.status != 503 || !strings.Contains(got.body, "ainda não está disponível") {
		t.Fatalf("sem configuração: %d %s", got.status, got.body)
	}
}
