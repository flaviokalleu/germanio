// Package oidc is the relying-party side of OpenID Connect (Core 1.0,
// Authorization Code flow) with PKCE (RFC 7636, S256), state and nonce.
//
// It knows the protocol, never a provider: the issuer, the client and its
// secret come from configuration. Everything the provider says is checked
// before it is believed: the discovery document must name the configured
// issuer; ID tokens are accepted only signed with RS256 or ES256 by a key of
// the provider's JWKS (verified with the standard library), with the right
// iss, aud (and azp), exp, iat, nbf and nonce. Every request goes through
// the SSRF-protected transport of runtime/httpclient, never follows a
// redirect and reads a bounded body.
package oidc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/flaviokalleu/germanio/runtime/httpclient"
)

const (
	maxBody       = 1 << 20 // the most any answer of the provider may carry
	maxToken      = 16 << 10
	clockSkew     = 60 * time.Second
	metadataTTL   = time.Hour
	keysTTL       = time.Hour
	requestTimout = 10 * time.Second
)

// Config is what the server operator configures.
type Config struct {
	Issuer       string // exact issuer identifier (https://…)
	ClientID     string
	ClientSecret string // optional: a public client relies on PKCE alone
	RedirectURI  string
	// AllowHTTP accepts http:// endpoints (development against a local
	// provider only; production providers must use https).
	AllowHTTP bool
	// HTTP overrides the client (tests); nil uses the SSRF-protected one.
	HTTP *http.Client
}

// Provider talks to one OpenID provider; safe for concurrent use.
type Provider struct {
	cfg    Config
	client *http.Client

	mu       sync.Mutex
	meta     *metadata
	metaAt   time.Time
	keys     map[string]jwk
	keyList  []jwk
	keysAt   time.Time
	keysTry  time.Time
	keysFrom string
}

type metadata struct {
	Issuer                string   `json:"issuer"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	UserinfoEndpoint      string   `json:"userinfo_endpoint"`
	JWKSURI               string   `json:"jwks_uri"`
	TokenAuthMethods      []string `json:"token_endpoint_auth_methods_supported"`
	ChallengeMethods      []string `json:"code_challenge_methods_supported"`
}

// New prepares a provider; nothing is fetched until it is used.
func New(cfg Config) *Provider {
	c := cfg.HTTP
	if c == nil {
		c = &http.Client{
			Transport: httpclient.Transport(),
			Timeout:   requestTimout,
			// a redirect could carry the client secret or a code elsewhere
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		}
	}
	return &Provider{cfg: cfg, client: c}
}

// Error is a refusal with a message fit for the person (Portuguese).
type Error struct{ Msg string }

func (e *Error) Error() string { return e.Msg }

func fail(format string, args ...any) error { return &Error{Msg: fmt.Sprintf(format, args...)} }

// Random returns n random bytes, URL-safe base64 (no padding).
func Random(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Challenge is the S256 code challenge of a PKCE verifier.
func Challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (p *Provider) checkURL(raw, what string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" {
		return fail("o provedor anunciou um endereço inválido para %s: %q", what, raw)
	}
	if u.Scheme != "https" && !(p.cfg.AllowHTTP && u.Scheme == "http") {
		return fail("o provedor anunciou %s sem https (%q); só https é aceito fora do desenvolvimento", what, raw)
	}
	return nil
}

func (p *Provider) get(ctx context.Context, raw string, header http.Header, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	for k, v := range header {
		req.Header[k] = v
	}
	return p.do(req, out)
}

func (p *Provider) do(req *http.Request, out any) error {
	req.Header.Set("User-Agent", "Germanio")
	resp, err := p.client.Do(req)
	if err != nil {
		return fail("o provedor não respondeu: %v", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		return fail("o provedor não respondeu: %v", err)
	}
	if len(raw) > maxBody {
		return fail("a resposta do provedor é grande demais")
	}
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error string `json:"error"`
			Desc  string `json:"error_description"`
		}
		json.Unmarshal(raw, &e)
		if e.Error != "" {
			return fail("o provedor recusou (%s) %s", short(e.Error), short(e.Desc))
		}
		return fail("o provedor respondeu %d", resp.StatusCode)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fail("a resposta do provedor não é JSON válido")
	}
	return nil
}

// short keeps provider text short and on one line before it is shown.
func short(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

// discover returns the provider's metadata (cached for an hour).
func (p *Provider) discover(ctx context.Context) (*metadata, error) {
	p.mu.Lock()
	if p.meta != nil && time.Since(p.metaAt) < metadataTTL {
		m := p.meta
		p.mu.Unlock()
		return m, nil
	}
	p.mu.Unlock()
	if err := p.checkURL(p.cfg.Issuer, "o emissor"); err != nil {
		return nil, fail("o emissor configurado (GERMANIO_OIDC_EMISSOR) precisa ser um endereço https: %q", p.cfg.Issuer)
	}
	var m metadata
	if err := p.get(ctx, strings.TrimSuffix(p.cfg.Issuer, "/")+"/.well-known/openid-configuration", nil, &m); err != nil {
		return nil, err
	}
	// OpenID Connect Discovery §4.3: the issuer in the document must be
	// exactly the one configured, or tokens could come from anyone
	if m.Issuer != p.cfg.Issuer {
		return nil, fail("o provedor se anuncia como %q, mas o emissor configurado é %q; use exatamente o valor anunciado em GERMANIO_OIDC_EMISSOR", short(m.Issuer), p.cfg.Issuer)
	}
	for what, u := range map[string]string{"a autorização": m.AuthorizationEndpoint, "o token": m.TokenEndpoint, "as chaves (jwks_uri)": m.JWKSURI} {
		if err := p.checkURL(u, what); err != nil {
			return nil, err
		}
	}
	if m.UserinfoEndpoint != "" && p.checkURL(m.UserinfoEndpoint, "userinfo") != nil {
		m.UserinfoEndpoint = ""
	}
	if len(m.ChallengeMethods) > 0 && !contains(m.ChallengeMethods, "S256") {
		return nil, fail("o provedor não aceita PKCE com S256; a entrada externa exige S256")
	}
	p.mu.Lock()
	p.meta, p.metaAt = &m, time.Now()
	p.mu.Unlock()
	return &m, nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// AuthURL is where the browser goes to sign in at the provider.
func (p *Provider) AuthURL(ctx context.Context, state, nonce, verifier string) (string, error) {
	m, err := p.discover(ctx)
	if err != nil {
		return "", err
	}
	u, _ := url.Parse(m.AuthorizationEndpoint)
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", p.cfg.ClientID)
	q.Set("redirect_uri", p.cfg.RedirectURI)
	q.Set("scope", "openid email profile")
	q.Set("state", state)
	q.Set("nonce", nonce)
	q.Set("code_challenge", Challenge(verifier))
	q.Set("code_challenge_method", "S256")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// Tokens is the answer of the token endpoint.
type Tokens struct {
	IDToken     string `json:"id_token"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// Exchange trades the authorization code (and the PKCE verifier) for tokens.
func (p *Provider) Exchange(ctx context.Context, code, verifier string) (*Tokens, error) {
	m, err := p.discover(ctx)
	if err != nil {
		return nil, err
	}
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {p.cfg.RedirectURI}, "code_verifier": {verifier}}
	basic := false
	switch {
	case p.cfg.ClientSecret == "":
		form.Set("client_id", p.cfg.ClientID)
	case len(m.TokenAuthMethods) == 0 || contains(m.TokenAuthMethods, "client_secret_basic"):
		basic = true // the default of OpenID Connect Core §9
	case contains(m.TokenAuthMethods, "client_secret_post"):
		form.Set("client_id", p.cfg.ClientID)
		form.Set("client_secret", p.cfg.ClientSecret)
	default:
		return nil, fail("o provedor não aceita client_secret_basic nem client_secret_post para o token")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	if basic {
		// RFC 6749 §2.3.1: both parts form-encoded before Basic
		req.SetBasicAuth(url.QueryEscape(p.cfg.ClientID), url.QueryEscape(p.cfg.ClientSecret))
	}
	var t Tokens
	if err := p.do(req, &t); err != nil {
		return nil, err
	}
	if t.IDToken == "" {
		return nil, fail("o provedor não devolveu um id_token (o escopo openid foi pedido)")
	}
	return &t, nil
}

// Claims are the facts of a verified ID token (or of userinfo).
type Claims struct {
	Issuer            string
	Subject           string
	Email             string
	EmailVerified     bool
	Name              string
	PreferredUsername string
}

// Verify checks a compact ID token and returns its claims. nonce is the
// value sent with this sign-in; it must come back unchanged.
func (p *Provider) Verify(ctx context.Context, raw, nonce string, now time.Time) (*Claims, error) {
	m, err := p.discover(ctx)
	if err != nil {
		return nil, err
	}
	if len(raw) > maxToken {
		return nil, fail("id_token grande demais")
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return nil, fail("id_token malformado")
	}
	var head struct {
		Alg  string          `json:"alg"`
		Kid  string          `json:"kid"`
		Crit json.RawMessage `json:"crit"`
	}
	if err := decodePart(parts[0], &head); err != nil {
		return nil, fail("id_token malformado (cabeçalho)")
	}
	if head.Crit != nil {
		return nil, fail("id_token com extensões críticas desconhecidas")
	}
	if head.Alg != "RS256" && head.Alg != "ES256" {
		// never "none", never HMAC (the client secret is not a signing key here)
		return nil, fail("id_token assinado com %q; só RS256 e ES256 são aceitos", short(head.Alg))
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fail("id_token malformado (assinatura)")
	}
	key, err := p.key(ctx, m, head.Kid, head.Alg)
	if err != nil {
		return nil, err
	}
	if !key.verify(head.Alg, []byte(parts[0]+"."+parts[1]), sig) {
		return nil, fail("a assinatura do id_token não confere com as chaves do provedor")
	}
	var c struct {
		Iss           string          `json:"iss"`
		Sub           string          `json:"sub"`
		Aud           json.RawMessage `json:"aud"`
		Azp           string          `json:"azp"`
		Exp           *json.Number    `json:"exp"`
		Iat           *json.Number    `json:"iat"`
		Nbf           *json.Number    `json:"nbf"`
		Nonce         string          `json:"nonce"`
		Email         string          `json:"email"`
		EmailVerified any             `json:"email_verified"`
		Name          string          `json:"name"`
		Preferred     string          `json:"preferred_username"`
	}
	if err := decodePart(parts[1], &c); err != nil {
		return nil, fail("id_token malformado (conteúdo)")
	}
	if c.Iss != m.Issuer {
		return nil, fail("o id_token foi emitido por %q, não pelo emissor configurado", short(c.Iss))
	}
	aud, err := audiences(c.Aud)
	if err != nil || !contains(aud, p.cfg.ClientID) {
		return nil, fail("o id_token não é destinado a este sistema (aud)")
	}
	if (len(aud) > 1 || c.Azp != "") && c.Azp != p.cfg.ClientID {
		return nil, fail("o id_token foi pedido por outro cliente (azp)")
	}
	exp, ok := seconds(c.Exp)
	if !ok {
		return nil, fail("o id_token não diz quando vence (exp)")
	}
	if !now.Before(exp.Add(clockSkew)) {
		return nil, fail("o id_token venceu")
	}
	iat, ok := seconds(c.Iat)
	if !ok || iat.After(now.Add(clockSkew)) {
		return nil, fail("o id_token não tem uma data de emissão válida (iat)")
	}
	if nbf, ok := seconds(c.Nbf); ok && nbf.After(now.Add(clockSkew)) {
		return nil, fail("o id_token ainda não vale (nbf)")
	}
	if nonce == "" || subtle.ConstantTimeCompare([]byte(c.Nonce), []byte(nonce)) != 1 {
		return nil, fail("o id_token não pertence a esta entrada (nonce)")
	}
	if c.Sub == "" || len(c.Sub) > 255 {
		return nil, fail("o id_token não identifica a pessoa (sub)")
	}
	return &Claims{Issuer: c.Iss, Subject: c.Sub, Email: strings.TrimSpace(c.Email), EmailVerified: truthy(c.EmailVerified),
		Name: strings.TrimSpace(c.Name), PreferredUsername: strings.TrimSpace(c.Preferred)}, nil
}

// UserInfo completes claims missing from the ID token (e-mail, name) with
// the userinfo endpoint; its sub must be the token's (Core §5.3.2).
func (p *Provider) UserInfo(ctx context.Context, accessToken string, c *Claims) error {
	m, err := p.discover(ctx)
	if err != nil || m.UserinfoEndpoint == "" || accessToken == "" {
		return err
	}
	var u struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified any    `json:"email_verified"`
		Name          string `json:"name"`
		Preferred     string `json:"preferred_username"`
	}
	if err := p.get(ctx, m.UserinfoEndpoint, http.Header{"Authorization": {"Bearer " + accessToken}}, &u); err != nil {
		return err
	}
	if u.Sub != c.Subject {
		return fail("o userinfo do provedor descreve outra pessoa (sub)")
	}
	if c.Email == "" {
		c.Email, c.EmailVerified = strings.TrimSpace(u.Email), truthy(u.EmailVerified)
	}
	if c.Name == "" {
		c.Name = strings.TrimSpace(u.Name)
	}
	if c.PreferredUsername == "" {
		c.PreferredUsername = strings.TrimSpace(u.Preferred)
	}
	return nil
}

func decodePart(s string, out any) error {
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return err
	}
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.UseNumber()
	return d.Decode(out)
}

func audiences(raw json.RawMessage) ([]string, error) {
	var one string
	if json.Unmarshal(raw, &one) == nil {
		return []string{one}, nil
	}
	var many []string
	if err := json.Unmarshal(raw, &many); err != nil {
		return nil, err
	}
	return many, nil
}

func seconds(n *json.Number) (time.Time, bool) {
	if n == nil {
		return time.Time{}, false
	}
	f, err := n.Float64()
	if err != nil || f <= 0 {
		return time.Time{}, false
	}
	return time.Unix(int64(f), 0), true
}

// truthy reads email_verified, which some providers send as a string.
func truthy(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return strings.EqualFold(x, "true")
	}
	return false
}

var errNoKey = errors.New("no key")

// keysMinRetry is the least time between two fetches of the JWKS caused by
// an unknown key: a token with a made-up kid cannot make the server hammer
// the provider.
var keysMinRetry = 30 * time.Second

// key finds the signing key kid of the provider, fetching the JWKS again
// (at most every 30 seconds) when the key is unknown: providers rotate keys.
func (p *Provider) key(ctx context.Context, m *metadata, kid, alg string) (jwk, error) {
	p.mu.Lock()
	k, err := p.pick(m, kid, alg)
	stale := p.keysFrom != m.JWKSURI || time.Since(p.keysAt) > keysTTL
	canRetry := time.Since(p.keysTry) > keysMinRetry
	p.mu.Unlock()
	if err == nil && !stale {
		return k, nil
	}
	if !stale && !canRetry {
		return jwk{}, fail("a chave que assinou o id_token não está entre as chaves do provedor")
	}
	if err := p.fetchKeys(ctx, m); err != nil {
		return jwk{}, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if k, err := p.pick(m, kid, alg); err == nil {
		return k, nil
	}
	return jwk{}, fail("a chave que assinou o id_token não está entre as chaves do provedor")
}

func (p *Provider) pick(m *metadata, kid, alg string) (jwk, error) {
	if p.keysFrom != m.JWKSURI {
		return jwk{}, errNoKey
	}
	if kid != "" {
		if k, ok := p.keys[kid]; ok && k.fits(alg) {
			return k, nil
		}
		return jwk{}, errNoKey
	}
	// no kid: only unambiguous when a single key fits
	var found []jwk
	for _, k := range p.keyList {
		if k.fits(alg) {
			found = append(found, k)
		}
	}
	if len(found) == 1 {
		return found[0], nil
	}
	return jwk{}, errNoKey
}

func (p *Provider) fetchKeys(ctx context.Context, m *metadata) error {
	p.mu.Lock()
	p.keysTry = time.Now()
	p.mu.Unlock()
	var set struct {
		Keys []json.RawMessage `json:"keys"`
	}
	if err := p.get(ctx, m.JWKSURI, nil, &set); err != nil {
		return err
	}
	keys := map[string]jwk{}
	var list []jwk
	for _, raw := range set.Keys {
		k, err := parseJWK(raw)
		if err != nil {
			continue // unknown or weak kinds are skipped, never trusted
		}
		list = append(list, k)
		if k.kid != "" {
			keys[k.kid] = k
		}
	}
	p.mu.Lock()
	p.keys, p.keyList, p.keysAt, p.keysFrom = keys, list, time.Now(), m.JWKSURI
	p.mu.Unlock()
	return nil
}
