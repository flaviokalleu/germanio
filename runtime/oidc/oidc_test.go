package oidc_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/flaviokalleu/germanio/runtime/oidc"
	"github.com/flaviokalleu/germanio/runtime/oidc/oidctest"
)

func provider(t *testing.T, fake *oidctest.Provider) *oidc.Provider {
	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "1")
	return oidc.New(oidc.Config{Issuer: fake.Issuer(), ClientID: oidctest.ClientID, ClientSecret: oidctest.ClientSecret,
		RedirectURI: "https://app.example/entrar/externo/retorno", AllowHTTP: true})
}

func claims(fake *oidctest.Provider, nonce string) map[string]any {
	now := time.Now()
	return map[string]any{"iss": fake.Issuer(), "sub": "123", "aud": oidctest.ClientID, "exp": now.Add(time.Minute).Unix(),
		"iat": now.Unix(), "nonce": nonce, "email": "ana@x.com", "email_verified": true}
}

// RFC 7636 Appendix B.
func TestChallengeS256(t *testing.T) {
	if got := oidc.Challenge("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"); got != "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM" {
		t.Fatalf("S256: %s", got)
	}
}

// Every check of an ID token refuses on its own: signature, algorithm,
// key, issuer, audience, authorized party, expiry, issue time, nonce.
func TestVerifyRefusals(t *testing.T) {
	fake := oidctest.New(t)
	p := provider(t, fake)
	ctx := context.Background()
	ok := oidctest.Sign(fake.Key, oidctest.KeyID, claims(fake, "n1"))
	c, err := p.Verify(ctx, ok, "n1", time.Now())
	if err != nil || c.Subject != "123" || c.Email != "ana@x.com" || !c.EmailVerified || c.Issuer != fake.Issuer() {
		t.Fatalf("token válido: %+v %v", c, err)
	}
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	mod := func(f func(m map[string]any)) string {
		m := claims(fake, "n1")
		f(m)
		return oidctest.Sign(fake.Key, oidctest.KeyID, m)
	}
	none := func() string {
		h := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
		b, _ := json.Marshal(claims(fake, "n1"))
		return h + "." + base64.RawURLEncoding.EncodeToString(b) + "."
	}()
	hs := func() string {
		h := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","kid":"` + oidctest.KeyID + `"}`))
		b, _ := json.Marshal(claims(fake, "n1"))
		return h + "." + base64.RawURLEncoding.EncodeToString(b) + ".AAAA"
	}()
	parts := strings.Split(ok, ".")
	swapped := parts[0] + "." + strings.Split(mod(func(m map[string]any) { m["sub"] = "999" }), ".")[1] + "." + parts[2]
	cases := map[string]struct {
		tok, nonce, want string
	}{
		"assinatura de outra chave": {oidctest.Sign(other, oidctest.KeyID, claims(fake, "n1")), "n1", "assinatura"},
		"conteúdo trocado":          {swapped, "n1", "assinatura"},
		"alg none":                  {none, "n1", "RS256"},
		"alg HS256":                 {hs, "n1", "RS256"},
		"chave desconhecida":        {oidctest.Sign(fake.Key, "outra", claims(fake, "n1")), "n1", "chave"},
		"emissor errado":            {mod(func(m map[string]any) { m["iss"] = "https://mal.example" }), "n1", "emitido"},
		"aud errado":                {mod(func(m map[string]any) { m["aud"] = "outro-cliente" }), "n1", "aud"},
		"aud lista sem azp":         {mod(func(m map[string]any) { m["aud"] = []string{oidctest.ClientID, "outro"} }), "n1", "azp"},
		"azp de outro":              {mod(func(m map[string]any) { m["azp"] = "outro" }), "n1", "azp"},
		"vencido":                   {mod(func(m map[string]any) { m["exp"] = time.Now().Add(-time.Hour).Unix() }), "n1", "venceu"},
		"sem exp":                   {mod(func(m map[string]any) { delete(m, "exp") }), "n1", "exp"},
		"iat no futuro":             {mod(func(m map[string]any) { m["iat"] = time.Now().Add(time.Hour).Unix() }), "n1", "iat"},
		"nbf no futuro":             {mod(func(m map[string]any) { m["nbf"] = time.Now().Add(time.Hour).Unix() }), "n1", "nbf"},
		"nonce errado":              {ok, "n2", "nonce"},
		"nonce vazio":               {mod(func(m map[string]any) { m["nonce"] = "" }), "", "nonce"},
		"sem sub":                   {mod(func(m map[string]any) { m["sub"] = "" }), "n1", "sub"},
		"malformado":                {"a.b", "n1", "malformado"},
	}
	for name, tc := range cases {
		if _, err := p.Verify(ctx, tc.tok, tc.nonce, time.Now()); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: esperado erro com %q, veio %v", name, tc.want, err)
		}
	}
	// a list of audiences is fine when azp names this client
	if _, err := p.Verify(ctx, mod(func(m map[string]any) { m["aud"] = []string{oidctest.ClientID, "x"}; m["azp"] = oidctest.ClientID }), "n1", time.Now()); err != nil {
		t.Fatalf("aud lista com azp: %v", err)
	}
	// email_verified as a string (some providers)
	if c, err := p.Verify(ctx, mod(func(m map[string]any) { m["email_verified"] = "true" }), "n1", time.Now()); err != nil || !c.EmailVerified {
		t.Fatalf("email_verified em texto: %+v %v", c, err)
	}
}

// The discovery document must name the configured issuer exactly, and
// endpoints must be https outside development.
func TestDiscovery(t *testing.T) {
	fake := oidctest.New(t)
	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "1")
	ctx := context.Background()
	wrong := oidc.New(oidc.Config{Issuer: fake.Issuer() + "/", ClientID: oidctest.ClientID, AllowHTTP: true})
	if _, err := wrong.AuthURL(ctx, "s", "n", "v"); err == nil || !strings.Contains(err.Error(), "se anuncia como") {
		t.Fatalf("emissor diferente do anunciado: %v", err)
	}
	plain := oidc.New(oidc.Config{Issuer: fake.Issuer(), ClientID: oidctest.ClientID})
	if _, err := plain.AuthURL(ctx, "s", "n", "v"); err == nil || !strings.Contains(err.Error(), "https") {
		t.Fatalf("http fora do desenvolvimento: %v", err)
	}
	p := provider(t, fake)
	u, err := p.AuthURL(ctx, "st", "no", "verificador")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"code_challenge_method=S256", "code_challenge=" + oidc.Challenge("verificador"), "state=st", "nonce=no", "scope=openid+email+profile", "response_type=code"} {
		if !strings.Contains(u, want) {
			t.Fatalf("endereço de autorização sem %s: %s", want, u)
		}
	}
}

// Without the local-network permission the SSRF guard refuses a provider
// on the server's own network.
func TestProvedorNaRedeLocalRecusado(t *testing.T) {
	fake := oidctest.New(t)
	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "")
	p := oidc.New(oidc.Config{Issuer: fake.Issuer(), ClientID: oidctest.ClientID, AllowHTTP: true})
	if _, err := p.AuthURL(context.Background(), "s", "n", "v"); err == nil || !strings.Contains(err.Error(), "rede local") {
		t.Fatalf("rede local: %v", err)
	}
}

// ES256 keys (P-256) are verified too; a key rotated after the first fetch
// is found by fetching the JWKS again.
func TestES256ERotacao(t *testing.T) {
	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "1")
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	kid := "ec-1"
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			json.NewEncoder(w).Encode(map[string]any{"issuer": srv.URL, "authorization_endpoint": srv.URL + "/a", "token_endpoint": srv.URL + "/t", "jwks_uri": srv.URL + "/k"})
		case "/k":
			raw, _ := key.PublicKey.Bytes()
			json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{"kty": "EC", "crv": "P-256", "kid": kid,
				"x": base64.RawURLEncoding.EncodeToString(raw[1:33]), "y": base64.RawURLEncoding.EncodeToString(raw[33:])}}})
		}
	}))
	defer srv.Close()
	p := oidc.New(oidc.Config{Issuer: srv.URL, ClientID: oidctest.ClientID, AllowHTTP: true})
	sign := func() string {
		head, _ := json.Marshal(map[string]any{"alg": "ES256", "kid": kid})
		now := time.Now()
		body, _ := json.Marshal(map[string]any{"iss": srv.URL, "sub": "e1", "aud": oidctest.ClientID, "exp": now.Add(time.Minute).Unix(), "iat": now.Unix(), "nonce": "n"})
		in := base64.RawURLEncoding.EncodeToString(head) + "." + base64.RawURLEncoding.EncodeToString(body)
		sum := sha256.Sum256([]byte(in))
		r, s, _ := ecdsa.Sign(rand.Reader, key, sum[:])
		sig := make([]byte, 64)
		r.FillBytes(sig[:32])
		s.FillBytes(sig[32:])
		return in + "." + base64.RawURLEncoding.EncodeToString(sig)
	}
	if c, err := p.Verify(context.Background(), sign(), "n", time.Now()); err != nil || c.Subject != "e1" {
		t.Fatalf("ES256: %+v %v", c, err)
	}
	// rotation: the provider signs with a new key under a new kid; the
	// keys are fetched again (here without waiting for the retry gap)
	defer oidc.SetKeysRetryForTest(0)()
	key, _ = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	kid = "ec-2"
	if c, err := p.Verify(context.Background(), sign(), "n", time.Now()); err != nil || c.Subject != "e1" {
		t.Fatalf("chave rotacionada: %+v %v", c, err)
	}
}
