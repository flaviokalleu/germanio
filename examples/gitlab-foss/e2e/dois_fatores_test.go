package e2e

import (
	"bytes"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/flaviokalleu/germanio/runtime/totp"
)

// ID-07: dois fatores no GitLab escrito em .ge — ligados pela sessão do
// navegador; depois, a senha sozinha não serve para OAuth nem para HTTP
// Basic (clientes git), e o token de acesso continua valendo.
func TestDoisFatoresGitLab(t *testing.T) {
	t.Setenv("GERMANIO_SEGREDO", strings.Repeat("s", 40))
	base := gitlab(t)
	ada := signup(t, base, "ada")
	pat := ada.must("POST", "/api/v4/personal_access_tokens", map[string]any{"name": "cli"}, 201)

	jar, _ := cookiejar.New(nil)
	web := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	post := func(path string, body map[string]any, csrf string) (int, map[string]any) {
		b, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", base+path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		if csrf != "" {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		resp, err := web.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		var out map[string]any
		json.Unmarshal(raw, &out)
		return resp.StatusCode, out
	}
	if code, out := post("/entrar", map[string]any{"login": "ada", "senha": "password123"}, ""); code != 200 {
		t.Fatalf("entrar: %d %v", code, out)
	}
	csrf := ""
	for _, ck := range web.Jar.Cookies(mustURL(t, base)) {
		parts := strings.Split(ck.Value, ".")
		if len(parts) == 3 {
			raw, _ := base64.RawURLEncoding.DecodeString(parts[1])
			var claims map[string]any
			json.Unmarshal(raw, &claims)
			csrf, _ = claims["csrf"].(string)
		}
	}
	code, info := post("/dois-fatores/ativar", map[string]any{"senha": "password123"}, csrf)
	if code != 200 {
		t.Fatalf("ativar: %d %v", code, info)
	}
	secret, _ := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(info["segredo"].(string))
	if code, out := post("/dois-fatores/confirmar", map[string]any{"codigo": totp.CodeAt(secret, totp.Step(time.Now()), totp.Digits)}, csrf); code != 200 {
		t.Fatalf("confirmar: %d %v", code, out)
	}

	if me := ada.must("GET", "/api/v4/user", nil, 200); me["two_factor_enabled"] != true {
		t.Fatalf("two_factor_enabled: %v", me)
	}
	anon := &api{t: t, base: base}
	if code, out, _ := anon.call("POST", "/oauth/token", map[string]any{"grant_type": "password", "username": "ada", "password": "password123"}); code != 400 {
		t.Fatalf("oauth com senha e dois fatores: %d %v", code, out)
	}
	req, _ := http.NewRequest("GET", base+"/api/v4/user", nil)
	req.SetBasicAuth("ada", "password123")
	if resp, err := http.DefaultClient.Do(req); err != nil || resp.StatusCode != 401 {
		t.Fatalf("HTTP Basic com senha e dois fatores: %v %v", err, resp)
	}
	withPAT := &api{t: t, base: base, token: pat["token"].(string), pat: true}
	if me := withPAT.must("GET", "/api/v4/user", nil, 200); me["username"] != "ada" {
		t.Fatalf("token de acesso com dois fatores: %v", me)
	}
	// the password on the login page now asks for the code
	if code, out := post("/entrar", map[string]any{"login": "ada", "senha": "password123"}, ""); code != 202 || out["dois_fatores"] != true {
		t.Fatalf("entrar com dois fatores: %d %v", code, out)
	}
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
