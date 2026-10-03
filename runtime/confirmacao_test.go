package runtime

import (
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

var confirmLink = regexp.MustCompile(`https://clube\.example/confirmar-email\?token=([A-Za-z0-9_-]+)`)

// linksTo returns the confirmation tokens sent to address, oldest first.
func linksTo(t *testing.T, dir, address string) []string {
	t.Helper()
	files, _ := filepath.Glob(filepath.Join(dir, "*.txt"))
	var out []string
	for _, f := range files {
		b, _ := os.ReadFile(f)
		if !strings.Contains(string(b), "Para: "+address+"\n") {
			continue
		}
		if m := confirmLink.FindStringSubmatch(string(b)); m != nil {
			out = append(out, m[1])
		}
	}
	return out
}

func newClient(t *testing.T, base string) *client {
	jar, _ := cookiejar.New(nil)
	return &client{t: t, base: base, http: &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, Header: http.Header{}}
}

// E-mail confirmation (GEP 0031): whoever signs up gets a one-use link
// that expires; until then the password opens nothing; asking for another
// link answers the same whether or not the account exists; a new e-mail is
// confirmed again, and an old link never confirms a new address.
func TestConfirmacaoDeEmail(t *testing.T) {
	mail := t.TempDir()
	t.Setenv("GERMANIO_CORREIO_PASTA", mail)
	t.Setenv("GERMANIO_URL_PUBLICA", "https://clube.example")
	app, c := loadApp(t, "testdata/confirmacao/app.ge")

	out := c.expect("POST", "/cadastro", map[string]any{"nome": "Ana", "email": "ana@x.com", "senha": "senha-boa-123"}, 201)
	if out["email_confirmado"] != false || !strings.Contains(toStrT(out["message"]), "link") {
		t.Fatalf("cadastro: %v", out)
	}
	c.expect("GET", "/_ge/eu", nil, 401) // no session before confirming

	refused := c.expect("POST", "/entrar", map[string]any{"login": "ana@x.com", "senha": "senha-boa-123"}, 403)
	if !strings.Contains(toStrT(refused["message"]), "Confirme seu e-mail") {
		t.Fatalf("mensagem educativa ao entrar sem confirmar: %v", refused)
	}
	wrong := c.expect("POST", "/entrar", map[string]any{"login": "ana@x.com", "senha": "senha-errada-1"}, 401)
	nobody := c.expect("POST", "/entrar", map[string]any{"login": "ninguem@x.com", "senha": "senha-errada-1"}, 401)
	if wrong["message"] != nobody["message"] {
		t.Fatalf("senha errada revela a conta: %v × %v", wrong, nobody)
	}
	// Git clients and API with the password: refused too
	req, _ := http.NewRequest("GET", c.base+"/_ge/eu", nil)
	req.SetBasicAuth("ana@x.com", "senha-boa-123")
	if resp, err := http.DefaultClient.Do(req); err != nil || resp.StatusCode != 401 {
		t.Fatalf("senha por HTTP Basic antes de confirmar: %v %v", err, resp)
	}

	// asking again: the same answer for anyone; no second e-mail within 2 minutes
	known := c.expect("POST", "/reenviar-confirmacao", map[string]any{"login": "ana@x.com"}, 202)
	unknown := c.expect("POST", "/reenviar-confirmacao", map[string]any{"login": "ninguem@x.com"}, 202)
	if known["message"] != unknown["message"] {
		t.Fatalf("reenviar revela a conta: %v × %v", known, unknown)
	}
	mails(t, mail, 1)
	time.Sleep(100 * time.Millisecond)
	links := linksTo(t, mail, "ana@x.com")
	if len(links) != 1 {
		t.Fatalf("esperado 1 link, vieram %d", len(links))
	}
	var n int
	app.DB.DB.QueryRow(`SELECT COUNT(*) FROM _germanio_confirmacao WHERE hash = ?`, links[0]).Scan(&n)
	if n != 0 {
		t.Fatal("o link está guardado em texto puro")
	}

	c.expect("POST", "/confirmar-email", map[string]any{"token": "inventado"}, 400)
	c.expect("POST", "/confirmar-email", map[string]any{"token": links[0]}, 200)
	c.expect("POST", "/confirmar-email", map[string]any{"token": links[0]}, 400) // one use
	me := c.expect("POST", "/entrar", map[string]any{"login": "ana@x.com", "senha": "senha-boa-123"}, 200)
	if me["email_confirmado"] != true {
		t.Fatalf("depois de confirmar: %v", me)
	}
	c.csrf = csrfFromCookie(t, c)

	// a new address is confirmed again; the session stays
	c.expect("PUT", "/_ge/api/usuarios/1", map[string]any{"email": "ana@novo.com"}, 200)
	if got := c.expect("GET", "/_ge/eu", nil, 200); got["email_confirmado"] != false {
		t.Fatalf("e-mail novo continua confirmado: %v", got)
	}
	// nobody marks their own e-mail as confirmed
	c.expect("PUT", "/_ge/api/usuarios/1", map[string]any{"email_confirmado": true, "nome": "Ana Maria"}, 200)
	if got := c.expect("GET", "/_ge/eu", nil, 200); got["email_confirmado"] != false || got["nome"] != "Ana Maria" {
		t.Fatalf("email_confirmado aceito da entrada: %v", got)
	}
	novo := linksTo(t, mail, "ana@novo.com")
	if len(novo) != 1 {
		t.Fatalf("link para o e-mail novo: %d", len(novo))
	}
	// an expired link does not work
	app.DB.DB.Exec(`UPDATE _germanio_confirmacao SET expira_em = '2000-01-01T00:00:00Z'`)
	c.expect("POST", "/confirmar-email", map[string]any{"token": novo[0]}, 400)
	// a link sent to an address the account no longer has confirms nothing
	c.expect("POST", "/reenviar-confirmacao", map[string]any{"login": "ana@novo.com"}, 202)
	deadline := time.Now().Add(5 * time.Second)
	for len(linksTo(t, mail, "ana@novo.com")) < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	novo = linksTo(t, mail, "ana@novo.com")
	if len(novo) != 2 {
		t.Fatalf("link reenviado: %d", len(novo))
	}
	c.expect("PUT", "/_ge/api/usuarios/1", map[string]any{"email": "ana@terceiro.com"}, 200)
	for _, l := range novo {
		c.expect("POST", "/confirmar-email", map[string]any{"token": l}, 400)
	}
	if got := c.expect("GET", "/_ge/eu", nil, 200); got["email_confirmado"] != false {
		t.Fatalf("um link antigo confirmou o endereço novo: %v", got)
	}
}

// The same link opened many times at once confirms once.
func TestConfirmacaoUsoUnicoConcorrente(t *testing.T) {
	mail := t.TempDir()
	t.Setenv("GERMANIO_CORREIO_PASTA", mail)
	t.Setenv("GERMANIO_URL_PUBLICA", "https://clube.example")
	_, c := loadApp(t, "testdata/confirmacao/app.ge")
	c.expect("POST", "/cadastro", map[string]any{"nome": "Ana", "email": "ana@x.com", "senha": "senha-boa-123"}, 201)
	mails(t, mail, 1)
	token := linksTo(t, mail, "ana@x.com")[0]
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := c.http.Post(c.base+"/confirmar-email", "application/json", strings.NewReader(`{"token":"`+token+`"}`))
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
		t.Fatalf("o link confirmou %d vezes", ok)
	}
}

// Without e-mail, an app that confirms e-mails keeps sign-up closed and
// says why (never accounts nobody can confirm).
func TestConfirmacaoSemCorreio(t *testing.T) {
	t.Setenv("GERMANIO_CORREIO_PASTA", "")
	t.Setenv("GERMANIO_SMTP_HOST", "")
	_, c := loadApp(t, "testdata/confirmacao/app.ge")
	out := c.expect("POST", "/cadastro", map[string]any{"nome": "Ana", "email": "ana@x.com", "senha": "senha-boa-123"}, 503)
	if !strings.Contains(toStrT(out["message"]), "e-mail") {
		t.Fatalf("explicação: %v", out)
	}
}
