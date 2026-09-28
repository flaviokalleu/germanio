package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Notices by e-mail (GEP 0013, em teste): every new pending item is sent to
// its owner after the change is saved.
func TestAvisosPorEmail(t *testing.T) {
	mail := t.TempDir()
	t.Setenv("GERMANIO_CORREIO_PASTA", mail)
	t.Setenv("GERMANIO_URL_PUBLICA", "https://suporte.example")
	_, c := loadApp(t, "testdata/avisos/app.ge")
	ana := signIn(t, c.base, "Ana", "ana@x.com")
	bia := signIn(t, c.base, "Bia", "bia@x.com")
	me := bia.expect("GET", "/_ge/eu", nil, 200)
	ana.expect("POST", "/_ge/api/chamados", map[string]any{"titulo": "Impressora", "responsaveis": []any{me["id"], ana.expect("GET", "/_ge/eu", nil, 200)["id"]}}, 201)
	ana.expect("POST", "/_ge/api/chamados", map[string]any{"titulo": "recusado", "responsaveis": []any{me["id"]}}, 400)
	ana.expect("POST", "/_ge/api/chamados", map[string]any{"titulo": "Demissão do diretor", "confidencial": true, "responsaveis": []any{me["id"]}}, 201)

	mails(t, mail, 2)
	time.Sleep(200 * time.Millisecond) // an extra e-mail, if any, would be here by now
	files, _ := filepath.Glob(filepath.Join(mail, "*"))
	var bodies []string
	for _, f := range files {
		b, _ := os.ReadFile(f)
		bodies = append(bodies, string(b))
	}
	all := strings.Join(bodies, "\n---\n")
	if len(files) != 2 {
		t.Fatalf("esperados 2 e-mails (Bia, duas pendências; nada para Ana, nada do chamado recusado), vieram %d:\n%s", len(files), all)
	}
	for _, b := range bodies {
		if !strings.Contains(b, "Para: bia@x.com") || !strings.Contains(b, "https://suporte.example") {
			t.Fatalf("e-mail inesperado:\n%s", b)
		}
	}
	if !strings.Contains(all, "Impressora") || strings.Contains(all, "Demissão") || strings.Contains(all, "recusado") {
		t.Fatalf("o e-mail diz o que a pessoa pode ver, e nada de uma mudança desfeita:\n%s", all)
	}
}
