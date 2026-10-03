package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Notices by e-mail honour each person's choice (GEP 0013, em teste): with
// avisos_por_email off, the pending item is created but no e-mail goes out.
func TestAvisosPorEmailEscolhaDaPessoa(t *testing.T) {
	mail := t.TempDir()
	t.Setenv("GERMANIO_CORREIO_PASTA", mail)
	_, c := loadApp(t, "testdata/avisos_escolha/app.ge")
	ana := signIn(t, c.base, "Ana", "ana@x.com")
	bia := signIn(t, c.base, "Bia", "bia@x.com")
	cid := signIn(t, c.base, "Cid", "cid@x.com")
	biaID := bia.expect("GET", "/_ge/eu", nil, 200)["id"]
	cidID := cid.expect("GET", "/_ge/eu", nil, 200)["id"]

	// the choice is the person's own: nobody else turns it off, and it is
	// private (others do not even see it)
	ana.expect("PUT", "/_ge/api/usuarios/"+jsonID(biaID), map[string]any{"avisos_por_email": false}, 403)
	if other := ana.expect("GET", "/_ge/api/usuarios/"+jsonID(biaID), nil, 200); other["avisos_por_email"] != nil {
		t.Fatalf("a escolha de outra pessoa é privada: %v", other)
	}
	if me := bia.expect("PUT", "/_ge/api/usuarios/"+jsonID(biaID), map[string]any{"avisos_por_email": false}, 200); me["avisos_por_email"] != false {
		t.Fatalf("a própria pessoa desliga os avisos: %v", me)
	}

	ana.expect("POST", "/_ge/api/chamados", map[string]any{"titulo": "Impressora", "responsaveis": []any{biaID, cidID}}, 201)
	mails(t, mail, 1)
	time.Sleep(200 * time.Millisecond) // a second e-mail, if any, would be here by now
	files, _ := filepath.Glob(filepath.Join(mail, "*.txt"))
	if len(files) != 1 {
		t.Fatalf("só Cid quer avisos por e-mail; vieram %d e-mails", len(files))
	}
	if b, _ := os.ReadFile(files[0]); !strings.Contains(string(b), "Para: cid@x.com") {
		t.Fatalf("o e-mail é de Cid:\n%s", b)
	}
	// the pending item is still there: only the e-mail is off
	if code, _, raw := bia.do("GET", "/_ge/api/pendencias", nil); code != 200 || !strings.Contains(raw, "Impressora") {
		t.Fatalf("a pendência de Bia continua: %d %s", code, raw)
	}
}
