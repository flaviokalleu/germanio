// Package oidctest is a fake OpenID provider for tests: discovery, an
// authorization endpoint that approves at once, a token endpoint that
// checks the client, the redirect URI, PKCE and single use of codes, a
// userinfo endpoint and a JWKS with a test RSA key. Tokens can be tampered
// with to test refusals.
package oidctest

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	ClientID     = "germanio-teste"
	ClientSecret = "segredo-do-cliente-de-teste"
	KeyID        = "chave-1"
)

// Identity is who is signed in at the provider.
type Identity struct {
	Subject           string
	Email             string
	EmailVerified     bool
	Name              string
	PreferredUsername string
	// OmitEmail leaves the e-mail out of the ID token (userinfo has it).
	OmitEmail bool
}

type grant struct {
	who       Identity
	nonce     string
	challenge string
	redirect  string
	tamper    func(map[string]any)
	signWith  *rsa.PrivateKey
}

// Provider is the fake provider; set Next before each sign-in.
type Provider struct {
	Server *httptest.Server
	Key    *rsa.PrivateKey

	mu     sync.Mutex
	Next   Identity
	Tamper func(claims map[string]any) // applied to the next ID tokens
	// SignWith signs the next ID tokens with another key (bad signature).
	SignWith *rsa.PrivateKey
	codes    map[string]*grant
	access   map[string]Identity
	// TokenCalls counts the requests to the token endpoint.
	TokenCalls int
}

// New starts a fake provider, closed when the test ends.
func New(t testing.TB) *Provider {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	p := &Provider{Key: key, codes: map[string]*grant{}, access: map[string]Identity{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/openid-configuration", p.discovery)
	mux.HandleFunc("GET /authorize", p.authorize)
	mux.HandleFunc("POST /token", p.token)
	mux.HandleFunc("GET /userinfo", p.userinfo)
	mux.HandleFunc("GET /jwks", p.jwks)
	p.Server = httptest.NewServer(mux)
	t.Cleanup(p.Server.Close)
	return p
}

// Issuer is the provider's issuer identifier.
func (p *Provider) Issuer() string { return p.Server.URL }

// Set changes who signs in next and clears tampering.
func (p *Provider) Set(who Identity) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Next, p.Tamper, p.SignWith = who, nil, nil
}

// Change runs f with the provider locked (to set Tamper or SignWith).
func (p *Provider) Change(f func(p *Provider)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	f(p)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (p *Provider) discovery(w http.ResponseWriter, _ *http.Request) {
	u := p.Server.URL
	writeJSON(w, 200, map[string]any{
		"issuer": u, "authorization_endpoint": u + "/authorize", "token_endpoint": u + "/token",
		"userinfo_endpoint": u + "/userinfo", "jwks_uri": u + "/jwks",
		"response_types_supported": []string{"code"}, "subject_types_supported": []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic"},
		"scopes_supported":                      []string{"openid", "email", "profile"},
	})
}

func random() string {
	b := make([]byte, 24)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// authorize approves at once, as if the person were signed in there.
func (p *Provider) authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("response_type") != "code" || q.Get("client_id") != ClientID || q.Get("code_challenge_method") != "S256" ||
		q.Get("code_challenge") == "" || q.Get("state") == "" || q.Get("nonce") == "" || !strings.Contains(q.Get("scope"), "openid") {
		http.Error(w, "pedido de autorização inválido", http.StatusBadRequest)
		return
	}
	code := random()
	p.mu.Lock()
	p.codes[code] = &grant{who: p.Next, nonce: q.Get("nonce"), challenge: q.Get("code_challenge"), redirect: q.Get("redirect_uri"), tamper: p.Tamper, signWith: p.SignWith}
	p.mu.Unlock()
	back, _ := url.Parse(q.Get("redirect_uri"))
	v := back.Query()
	v.Set("code", code)
	v.Set("state", q.Get("state"))
	back.RawQuery = v.Encode()
	http.Redirect(w, r, back.String(), http.StatusFound)
}

func (p *Provider) token(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	p.TokenCalls++
	p.mu.Unlock()
	id, secret, ok := r.BasicAuth()
	if ok {
		id, _ = url.QueryUnescape(id)
		secret, _ = url.QueryUnescape(secret)
	}
	if !ok || id != ClientID || secret != ClientSecret {
		writeJSON(w, 401, map[string]any{"error": "invalid_client"})
		return
	}
	r.ParseForm()
	code := r.PostForm.Get("code")
	p.mu.Lock()
	g := p.codes[code]
	delete(p.codes, code) // a code works once
	p.mu.Unlock()
	if r.PostForm.Get("grant_type") != "authorization_code" || g == nil {
		writeJSON(w, 400, map[string]any{"error": "invalid_grant", "error_description": "code inválido ou já usado"})
		return
	}
	sum := sha256.Sum256([]byte(r.PostForm.Get("code_verifier")))
	if base64.RawURLEncoding.EncodeToString(sum[:]) != g.challenge || r.PostForm.Get("redirect_uri") != g.redirect {
		writeJSON(w, 400, map[string]any{"error": "invalid_grant", "error_description": "PKCE ou redirect_uri não conferem"})
		return
	}
	now := time.Now()
	claims := map[string]any{"iss": p.Server.URL, "sub": g.who.Subject, "aud": ClientID, "exp": now.Add(5 * time.Minute).Unix(),
		"iat": now.Unix(), "nonce": g.nonce, "name": g.who.Name, "preferred_username": g.who.PreferredUsername}
	if !g.who.OmitEmail {
		claims["email"], claims["email_verified"] = g.who.Email, g.who.EmailVerified
	}
	if g.tamper != nil {
		g.tamper(claims)
	}
	key := p.Key
	if g.signWith != nil {
		key = g.signWith
	}
	access := random()
	p.mu.Lock()
	p.access[access] = g.who
	p.mu.Unlock()
	writeJSON(w, 200, map[string]any{"access_token": access, "token_type": "Bearer", "expires_in": 300, "id_token": Sign(key, KeyID, claims)})
}

func (p *Provider) userinfo(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	who, ok := p.access[strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")]
	p.mu.Unlock()
	if !ok {
		writeJSON(w, 401, map[string]any{"error": "invalid_token"})
		return
	}
	writeJSON(w, 200, map[string]any{"sub": who.Subject, "email": who.Email, "email_verified": who.EmailVerified, "name": who.Name, "preferred_username": who.PreferredUsername})
}

func (p *Provider) jwks(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]any{"keys": []any{JWK(&p.Key.PublicKey, KeyID)}})
}

// JWK is the public RSA key as a JSON Web Key.
func JWK(pub *rsa.PublicKey, kid string) map[string]any {
	return map[string]any{"kty": "RSA", "kid": kid, "use": "sig", "alg": "RS256",
		"n": base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())}
}

// Sign makes a compact RS256 JWS of claims.
func Sign(key *rsa.PrivateKey, kid string, claims map[string]any) string {
	head, _ := json.Marshal(map[string]any{"alg": "RS256", "typ": "JWT", "kid": kid})
	body, _ := json.Marshal(claims)
	in := base64.RawURLEncoding.EncodeToString(head) + "." + base64.RawURLEncoding.EncodeToString(body)
	sum := sha256.Sum256([]byte(in))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		panic(err)
	}
	return in + "." + base64.RawURLEncoding.EncodeToString(sig)
}
