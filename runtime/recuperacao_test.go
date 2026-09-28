package runtime

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// Password recovery (GEP 0008, em teste): the answer never reveals which
// accounts exist; the e-mail carries a link that works once, expires, and
// whose new password obeys the field's rules.
func TestRecuperacaoDeSenha(t *testing.T) {
	mail := t.TempDir()
	t.Setenv("GERMANIO_CORREIO_PASTA", mail)
	t.Setenv("GERMANIO_URL_PUBLICA", "https://contas.example")
	app, c := loadApp(t, "testdata/recuperacao/app.ge")
	c.expect("POST", "/cadastro", map[string]any{"nome": "Ana", "email": "ana@x.com", "senha": "senha-antiga-1"}, 201)
	c.expect("POST", "/sair", nil, 204)

	known := c.expect("POST", "/esqueci", map[string]any{"login": "ana@x.com"}, 202)
	unknown := c.expect("POST", "/esqueci", map[string]any{"login": "ninguem@x.com"}, 202)
	if known["message"] != unknown["message"] {
		t.Fatalf("a resposta revela quais contas existem: %v × %v", known, unknown)
	}
	// repeated requests for the same account: the same answer, no new e-mail
	for i := 0; i < 5; i++ {
		c.expect("POST", "/esqueci", map[string]any{"login": "ana@x.com"}, 202)
	}
	files, _ := filepath.Glob(filepath.Join(mail, "*"))
	if len(files) != 1 {
		t.Fatalf("esperado 1 e-mail, vieram %d", len(files))
	}
	msg, _ := os.ReadFile(files[0])
	m := regexp.MustCompile(`https://contas\.example/redefinir\?token=([A-Za-z0-9_-]+)`).FindStringSubmatch(string(msg))
	if m == nil {
		t.Fatalf("o e-mail não traz o link: %s", msg)
	}
	token := m[1]

	c.expect("POST", "/redefinir", map[string]any{"token": token, "senha": "curta"}, 400)
	c.expect("POST", "/redefinir", map[string]any{"token": token, "senha": "senha-nova-123"}, 200)
	c.expect("POST", "/redefinir", map[string]any{"token": token, "senha": "outra-senha-123"}, 400) // one use
	c.expect("POST", "/entrar", map[string]any{"login": "ana@x.com", "senha": "senha-antiga-1"}, 401)
	c.expect("POST", "/entrar", map[string]any{"login": "ana@x.com", "senha": "senha-nova-123"}, 200)

	// an expired link does not work (the resend gap is over once the first
	// link was used: using it deletes the person's links)
	c.expect("POST", "/esqueci", map[string]any{"login": "ana@x.com"}, 202)
	files, _ = filepath.Glob(filepath.Join(mail, "*"))
	var fresh string
	for _, f := range files {
		b, _ := os.ReadFile(f)
		if mm := regexp.MustCompile(`token=([A-Za-z0-9_-]+)`).FindStringSubmatch(string(b)); mm != nil && mm[1] != token {
			fresh = mm[1]
		}
	}
	if _, err := app.DB.DB.Exec(`UPDATE _germanio_recuperacao SET expira_em = '2000-01-01T00:00:00Z'`); err != nil {
		t.Fatal(err)
	}
	c.expect("POST", "/redefinir", map[string]any{"token": fresh, "senha": "mais-uma-senha-1"}, 400)
}

// Without e-mail configured the application starts and says recovery is
// not available (never a silent failure).
func TestRecuperacaoSemCorreio(t *testing.T) {
	t.Setenv("GERMANIO_CORREIO_PASTA", "")
	t.Setenv("GERMANIO_SMTP_HOST", "")
	_, c := loadApp(t, "testdata/recuperacao/app.ge")
	c.expect("POST", "/esqueci", map[string]any{"login": "ana@x.com"}, 503)
}
