package runtime

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"
)

// Mentions (GEP 0017, em teste): @name in a text gives that person a pending
// item, only if they may see the record.
func TestMencoes(t *testing.T) {
	_, c := loadApp(t, "testdata/mencoes/app.ge")
	join := func(nome, user string) *client {
		jar, _ := cookiejar.New(nil)
		cl := &client{t: t, base: c.base, http: &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, Header: http.Header{}}
		cl.expect("POST", "/cadastro", map[string]any{"nome": nome, "username": user, "email": user + "@x.com", "senha": "senha-segura-1"}, 201)
		cl.csrf = csrfFromCookie(t, cl)
		return cl
	}
	ana, bia := join("Ana", "ana"), join("Bia", "bia")
	pend := func(who *client) []map[string]any {
		_, _, raw := who.do("GET", "/_ge/api/pendencias", nil)
		var out []map[string]any
		json.Unmarshal([]byte(raw), &out)
		return out
	}
	ana.expect("POST", "/_ge/api/chamados", map[string]any{"titulo": "Impressora", "descricao": "@bia pode olhar? (eu, @ana, já tentei; escreva para suporte@bia.com; @ninguem)"}, 201)
	got := pend(bia)
	if len(got) != 1 || !strings.HasPrefix(got[0]["motivo"].(string), "Menção: Chamado") {
		t.Fatalf("Bia mencionada recebe uma pendência: %v", got)
	}
	if n := len(pend(ana)); n != 0 {
		t.Fatalf("quem escreve não recebe pendência por se mencionar: %d", n)
	}
	// the same mention again on edit: nothing new; a new person: one more
	ana.expect("PUT", "/_ge/api/chamados/1", map[string]any{"descricao": "@bia ainda falha"}, 200)
	if n := len(pend(bia)); n != 1 {
		t.Fatalf("a mesma menção de novo criou outra pendência: %d", n)
	}
	// a confidential record: mentioning someone who cannot see it creates nothing
	ana.expect("POST", "/_ge/api/chamados", map[string]any{"titulo": "Demissão", "descricao": "@bia confidencial", "confidencial": true}, 201)
	if n := len(pend(bia)); n != 1 {
		t.Fatalf("uma menção revelou um chamado confidencial: %v", pend(bia))
	}
}
