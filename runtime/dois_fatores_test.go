package runtime

import (
	"encoding/base32"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/flaviokalleu/germanio/runtime/totp"
)

func secretOf(t *testing.T, b32 string) []byte {
	t.Helper()
	s, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(b32)
	if err != nil {
		t.Fatalf("segredo: %v", err)
	}
	return s
}

func codeAt(secret []byte, offset int) string {
	return totp.CodeAt(secret, uint64(int64(totp.Step(time.Now()))+int64(offset)), totp.Digits)
}

func basicStatus(t *testing.T, base, login, pass string) int {
	t.Helper()
	req, _ := http.NewRequest("GET", base+"/_ge/eu", nil)
	req.SetBasicAuth(login, pass)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// enableFactor turns the second factor on for a signed-in client and
// returns the secret and the recovery codes.
func enableFactor(t *testing.T, c *client, password string) ([]byte, []string) {
	t.Helper()
	info := c.expect("POST", "/dois-fatores/ativar", map[string]any{"senha": password}, 200)
	secret := secretOf(t, info["segredo"].(string))
	if !strings.HasPrefix(toStrT(info["uri"]), "otpauth://totp/") {
		t.Fatalf("uri: %v", info)
	}
	out := c.expect("POST", "/dois-fatores/confirmar", map[string]any{"codigo": codeAt(secret, 0)}, 200)
	var codes []string
	for _, x := range out["codigos_de_recuperacao"].([]any) {
		codes = append(codes, x.(string))
	}
	return secret, codes
}

// Two-factor authentication (GEP 0032): opt-in, set up with the password
// and a first code; then the password alone opens nothing (session, Git or
// API), each code works once, recovery codes work once, the secret and the
// codes are never stored in clear nor shown again.
func TestDoisFatores(t *testing.T) {
	t.Setenv("GERMANIO_SEGREDO", strings.Repeat("k", 40))
	app, c := loadApp(t, "testdata/dois_fatores/app.ge")
	ana := signIn(t, c.base, "Ana", "ana@x.com")
	const pass = "senha-segura-1"

	// a browser session and its CSRF proof are required to change it
	noCSRF := *ana
	noCSRF.csrf = ""
	noCSRF.expect("POST", "/dois-fatores/ativar", map[string]any{"senha": pass}, 403)
	c.expect("POST", "/dois-fatores/ativar", map[string]any{"senha": pass}, 401)
	ana.expect("POST", "/dois-fatores/ativar", map[string]any{"senha": "errada-123"}, 400)
	ana.expect("POST", "/dois-fatores/confirmar", map[string]any{"codigo": "123456"}, 400) // nothing being set up

	info := ana.expect("POST", "/dois-fatores/ativar", map[string]any{"senha": pass}, 200)
	secret := secretOf(t, info["segredo"].(string))
	if got := ana.expect("GET", "/_ge/eu", nil, 200); got["dois_fatores"] != false {
		t.Fatalf("ligado antes do primeiro código: %v", got)
	}
	wrong := "000000"
	if wrong == codeAt(secret, 0) || wrong == codeAt(secret, 1) || wrong == codeAt(secret, -1) {
		wrong = "999999"
	}
	ana.expect("POST", "/dois-fatores/confirmar", map[string]any{"codigo": wrong}, 400)
	out := ana.expect("POST", "/dois-fatores/confirmar", map[string]any{"codigo": codeAt(secret, 0)}, 200)
	codes := out["codigos_de_recuperacao"].([]any)
	if len(codes) != 10 {
		t.Fatalf("códigos de recuperação: %v", out)
	}
	if got := ana.expect("GET", "/_ge/eu", nil, 200); got["dois_fatores"] != true {
		t.Fatalf("depois de ligar: %v", got)
	}
	ana.expect("POST", "/dois-fatores/ativar", map[string]any{"senha": pass}, 409)

	// never in clear: not in the database, not on the page, not in the answers
	b32 := info["segredo"].(string)
	var stored string
	app.DB.DB.QueryRow(`SELECT segredo FROM _germanio_dois_fatores`).Scan(&stored)
	if !strings.HasPrefix(stored, "v1:") || strings.Contains(stored, b32) {
		t.Fatalf("segredo guardado sem cifra: %q", stored)
	}
	var n int
	app.DB.DB.QueryRow(`SELECT COUNT(*) FROM _germanio_codigos_recuperacao WHERE hash = ?`, codes[0]).Scan(&n)
	if n != 0 {
		t.Fatal("código de recuperação guardado em texto puro")
	}
	if _, _, page := ana.do("GET", "/dois-fatores", nil); strings.Contains(page, b32) || !strings.Contains(page, "ligados") {
		t.Fatalf("página dos dois fatores mostra o segredo ou o estado errado:\n%s", page)
	}
	if _, _, raw := ana.do("GET", "/_ge/eu", nil); strings.Contains(raw, b32) {
		t.Fatal("o segredo aparece nos dados da pessoa")
	}

	// signing in: the password gives a challenge, not a session
	bob := newClient(t, c.base)
	ch := bob.expect("POST", "/entrar", map[string]any{"login": "ana@x.com", "senha": pass}, 202)
	desafio := toStrT(ch["desafio"])
	if ch["dois_fatores"] != true || desafio == "" {
		t.Fatalf("desafio: %v", ch)
	}
	bob.expect("GET", "/_ge/eu", nil, 401)
	// the password alone: refused for Git/API clients (an access token is the way)
	if code := basicStatus(t, c.base, "ana@x.com", pass); code != 401 {
		t.Fatalf("senha sozinha por HTTP Basic com dois fatores: %d", code)
	}
	// a signed challenge never passes for a session cookie: it is not one
	bob.expect("POST", "/entrar/codigo", map[string]any{"desafio": desafio, "codigo": wrong}, 401)
	bob.expect("POST", "/entrar/codigo", map[string]any{"desafio": desafio, "codigo": codeAt(secret, 0)}, 401) // used to turn it on
	bob.expect("POST", "/entrar/codigo", map[string]any{"desafio": "inventado", "codigo": codeAt(secret, 1)}, 401)
	good := codeAt(secret, 1)
	me := bob.expect("POST", "/entrar/codigo", map[string]any{"desafio": desafio, "codigo": good}, 200)
	if me["email"] != "ana@x.com" {
		t.Fatalf("entrar com código: %v", me)
	}
	bob.expect("GET", "/_ge/eu", nil, 200)
	bob.expect("POST", "/entrar/codigo", map[string]any{"desafio": desafio, "codigo": good}, 401) // challenge used

	// the same code with a new challenge: refused (each code works once)
	eve := newClient(t, c.base)
	ch = eve.expect("POST", "/entrar", map[string]any{"login": "ana@x.com", "senha": pass}, 202)
	eve.expect("POST", "/entrar/codigo", map[string]any{"desafio": ch["desafio"], "codigo": good}, 401)
	// a recovery code works once
	rc := strings.ToUpper(codes[0].(string))
	eve.expect("POST", "/entrar/codigo", map[string]any{"desafio": ch["desafio"], "codigo": rc}, 200)
	ch = eve.expect("POST", "/entrar", map[string]any{"login": "ana@x.com", "senha": pass}, 202)
	eve.expect("POST", "/entrar/codigo", map[string]any{"desafio": ch["desafio"], "codigo": rc}, 401)

	// new recovery codes need the password and an app code; the old ones die
	ana.expect("POST", "/dois-fatores/codigos", map[string]any{"senha": pass, "codigo": codes[1]}, 400)
	// (as if a minute passed: the codes of this window were all used)
	app.DB.DB.Exec(`UPDATE _germanio_dois_fatores SET ultimo_passo = ultimo_passo - 3`)
	fresh := ana.expect("POST", "/dois-fatores/codigos", map[string]any{"senha": pass, "codigo": codeAt(secret, 0)}, 200)["codigos_de_recuperacao"].([]any)
	ch = eve.expect("POST", "/entrar", map[string]any{"login": "ana@x.com", "senha": pass}, 202)
	eve.expect("POST", "/entrar/codigo", map[string]any{"desafio": ch["desafio"], "codigo": codes[2]}, 401)

	// turning it off needs the password and a code
	ana.expect("POST", "/dois-fatores/desativar", map[string]any{"senha": pass, "codigo": wrong}, 400)
	ana.expect("POST", "/dois-fatores/desativar", map[string]any{"senha": "errada-123", "codigo": fresh[0]}, 400)
	ana.expect("POST", "/dois-fatores/desativar", map[string]any{"senha": pass, "codigo": fresh[0]}, 200)
	if got := ana.expect("GET", "/_ge/eu", nil, 200); got["dois_fatores"] != false {
		t.Fatalf("depois de desligar: %v", got)
	}
	newClient(t, c.base).expect("POST", "/entrar", map[string]any{"login": "ana@x.com", "senha": pass}, 200)
	if code := basicStatus(t, c.base, "ana@x.com", pass); code != 200 {
		t.Fatalf("HTTP Basic depois de desligar: %d", code)
	}
}

// Wrong codes count toward the lock of the login: after 10, even the right
// code (and the password) are refused for a while; typing the right password
// again never clears the count.
func TestDoisFatoresBloqueio(t *testing.T) {
	t.Setenv("GERMANIO_SEGREDO", strings.Repeat("k", 40))
	_, c := loadApp(t, "testdata/dois_fatores/app.ge")
	ana := signIn(t, c.base, "Ana", "ana@x.com")
	const pass = "senha-segura-1"
	secret, _ := enableFactor(t, ana, pass)
	wrong := "000000"
	for _, o := range []int{-1, 0, 1, 2} {
		if codeAt(secret, o) == wrong {
			wrong = "999999"
		}
	}
	bob := newClient(t, c.base)
	for i := 0; i < 10; i++ {
		// a fresh challenge each time: the right password does not reset the count
		ch := bob.expect("POST", "/entrar", map[string]any{"login": "ana@x.com", "senha": pass}, 202)
		bob.expect("POST", "/entrar/codigo", map[string]any{"desafio": ch["desafio"], "codigo": wrong}, 401)
	}
	bob.expect("POST", "/entrar", map[string]any{"login": "ana@x.com", "senha": pass}, 401) // locked
}

// The same code and challenge sent many times at once sign in once.
func TestDoisFatoresUsoUnicoConcorrente(t *testing.T) {
	t.Setenv("GERMANIO_SEGREDO", strings.Repeat("k", 40))
	_, c := loadApp(t, "testdata/dois_fatores/app.ge")
	ana := signIn(t, c.base, "Ana", "ana@x.com")
	secret, _ := enableFactor(t, ana, "senha-segura-1")
	bob := newClient(t, c.base)
	ch := bob.expect("POST", "/entrar", map[string]any{"login": "ana@x.com", "senha": "senha-segura-1"}, 202)
	body := `{"desafio":"` + toStrT(ch["desafio"]) + `","codigo":"` + codeAt(secret, 1) + `"}`
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Post(c.base+"/entrar/codigo", "application/json", strings.NewReader(body))
			if err != nil {
				t.Error(err)
				return
			}
			resp.Body.Close()
			mu.Lock()
			if resp.StatusCode == 200 {
				ok++
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	if ok != 1 {
		t.Fatalf("o código entrou %d vezes", ok)
	}
}

// Without a lasting server key the factor is not turned on (a secret saved
// now would be unreadable after a restart).
func TestDoisFatoresSemChave(t *testing.T) {
	t.Setenv("GERMANIO_SEGREDO", "")
	_, c := loadApp(t, "testdata/dois_fatores/app.ge")
	ana := signIn(t, c.base, "Ana", "ana@x.com")
	out := ana.expect("POST", "/dois-fatores/ativar", map[string]any{"senha": "senha-segura-1"}, 503)
	if !strings.Contains(toStrT(out["message"]), "GERMANIO_SEGREDO") {
		t.Fatalf("explicação: %v", out)
	}
}

// Without JavaScript: the login form asks for the code on its own page
// (the challenge travels in a short HttpOnly cookie, never in the address).
func TestDoisFatoresPelaPagina(t *testing.T) {
	t.Setenv("GERMANIO_SEGREDO", strings.Repeat("k", 40))
	_, c := loadApp(t, "testdata/dois_fatores/app.ge")
	ana := signIn(t, c.base, "Ana", "ana@x.com")
	secret, _ := enableFactor(t, ana, "senha-segura-1")
	bob := newClient(t, c.base)
	resp, err := bob.http.PostForm(c.base+"/entrar", url.Values{"login": {"ana@x.com"}, "senha": {"senha-segura-1"}})
	if err != nil || resp.StatusCode != 303 || resp.Header.Get("Location") != "/entrar/codigo" {
		t.Fatalf("formulário de login com dois fatores: %v %v", err, resp)
	}
	resp.Body.Close()
	if code, _, page := bob.do("GET", "/entrar/codigo", nil); code != 200 || !strings.Contains(page, `name="codigo"`) {
		t.Fatalf("página do código: %d\n%s", code, page)
	}
	resp, err = bob.http.PostForm(c.base+"/entrar/codigo", url.Values{"codigo": {codeAt(secret, 1)}})
	if err != nil || resp.StatusCode != 303 || resp.Header.Get("Location") != "/" {
		t.Fatalf("código pelo formulário: %v %v", err, resp)
	}
	resp.Body.Close()
	bob.expect("GET", "/_ge/eu", nil, 200)
}
