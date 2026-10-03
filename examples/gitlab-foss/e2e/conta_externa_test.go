package e2e

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"

	"github.com/flaviokalleu/germanio/runtime/oidc/oidctest"
)

// ID-07 (OmniAuth): o GitLab escrito em .ge oferece "Entrar com …" para um
// provedor OpenID Connect configurado no servidor. Uma pessoa nova nasce do
// e-mail verificado, com um username livre; uma conta que já existe não é
// tomada pelo e-mail (o GitLab não declara confirmação de e-mail): a pessoa
// entra com a senha e liga a conta externa.
func TestContaExternaGitLab(t *testing.T) {
	fake := oidctest.New(t)
	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "1")
	t.Setenv("GERMANIO_OIDC_EMISSOR", fake.Issuer())
	t.Setenv("GERMANIO_OIDC_CLIENTE", oidctest.ClientID)
	t.Setenv("GERMANIO_OIDC_SEGREDO", oidctest.ClientSecret)
	t.Setenv("GERMANIO_OIDC_NOME", "SSO da Empresa")
	t.Setenv("GERMANIO_URL_PUBLICA", "http://gitlab.example")
	base := gitlab(t)
	signup(t, base, "ada") // ada@example.com, signed up with a password

	browser := func() *http.Client {
		jar, _ := cookiejar.New(nil)
		return &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	get := func(c *http.Client, url string) (int, string, string) {
		resp, err := c.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(raw), resp.Header.Get("Location")
	}
	signIn := func(c *http.Client) (int, string) {
		resp, err := c.Post(base+"/entrar/externo", "application/x-www-form-urlencoded", strings.NewReader(""))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		_, _, back := get(c, resp.Header.Get("Location")) // the provider approves at once
		code, body, _ := get(c, base+strings.TrimPrefix(back, "http://gitlab.example"))
		return code, body
	}

	if _, page, _ := get(browser(), base+"/entrar"); !strings.Contains(page, "Entrar com SSO da Empresa") {
		t.Fatal("página de entrar do GitLab sem o botão do provedor")
	}

	fake.Set(oidctest.Identity{Subject: "u-42", Email: "grace@example.com", EmailVerified: true, Name: "Grace Hopper", PreferredUsername: "grace.hopper"})
	grace := browser()
	if code, body := signIn(grace); code != http.StatusSeeOther {
		t.Fatalf("entrar com o provedor: %d %s", code, body)
	}
	_, raw, _ := get(grace, base+"/api/v4/user")
	var me map[string]any
	json.Unmarshal([]byte(raw), &me)
	if me["username"] != "grace.hopper" || me["email"] != "grace@example.com" {
		t.Fatalf("pessoa criada pelo provedor: %s", raw)
	}

	// the same verified e-mail as an existing account: refused, never taken
	fake.Set(oidctest.Identity{Subject: "u-mal", Email: "ada@example.com", EmailVerified: true})
	mal := browser()
	if code, body := signIn(mal); code != http.StatusConflict || !strings.Contains(body, "ligue a conta externa") {
		t.Fatalf("conta existente pelo e-mail: %d %s", code, body)
	}
	if code, _, _ := get(mal, base+"/api/v4/user"); code != 401 {
		t.Fatalf("sessão depois da recusa: %d", code)
	}
}
