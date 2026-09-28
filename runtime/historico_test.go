package runtime

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"
)

// History (GEP 0011, em teste): one activity per change, in the change's
// transaction; field names, never values; seen by whoever sees the record it
// describes; a deleted record leaves no title behind.

func signIn(t *testing.T, base, nome, email string) *client {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	c := &client{t: t, base: base, http: &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, Header: http.Header{}}
	c.expect("POST", "/cadastro", map[string]any{"nome": nome, "email": email, "senha": "senha-segura-1"}, 201)
	c.csrf = csrfFromCookie(t, c)
	return c
}

func activities(t *testing.T, c *client, query string) []map[string]any {
	t.Helper()
	code, _, raw := c.do("GET", "/_ge/api/atividades"+query, nil)
	if code != 200 {
		t.Fatalf("atividades: %d %s", code, raw)
	}
	var out []map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("atividades: %v %s", err, raw)
	}
	return out
}

func actions(list []map[string]any) string {
	var out []string
	for i := len(list) - 1; i >= 0; i-- { // oldest first
		out = append(out, list[i]["acao"].(string)+" "+list[i]["recurso"].(string))
	}
	return strings.Join(out, ",")
}

func TestHistorico(t *testing.T) {
	_, c := loadApp(t, "testdata/historico/app.ge")
	ana := signIn(t, c.base, "Ana", "ana@x.com")
	bia := signIn(t, c.base, "Bia", "bia@x.com")

	ana.expect("POST", "/_ge/api/projetos", map[string]any{"nome": "Loja"}, 201)
	ana.expect("POST", "/_ge/api/projetos/1/issues", map[string]any{"titulo": "Erro no login", "descricao": "segredo da descrição"}, 201)
	ana.expect("PUT", "/_ge/api/projetos/1/issues/1", map[string]any{"descricao": "outro segredo"}, 200)
	ana.expect("PUT", "/_ge/api/projetos/1/issues/1", map[string]any{"descricao": "outro segredo"}, 200) // nothing changed
	ana.expect("POST", "/_ge/api/projetos/1/issues/1/fechar", nil, 200)
	// an undone change leaves no activity
	ana.expect("POST", "/_ge/api/projetos/1/issues", map[string]any{"descricao": "sem título"}, 400)

	got := activities(t, bia, "?dentro=projeto&dentro_id=1")
	if actions(got) != "criar issue,editar issue,fechar issue" {
		t.Fatalf("histórico do projeto: %s", actions(got))
	}
	edit := got[1]
	if edit["campos"] != "descricao" || edit["resumo"] != "Erro no login" {
		t.Fatalf("a edição diz quais campos mudaram, sem os valores: %v", edit)
	}
	raw, _ := json.Marshal(got)
	if strings.Contains(string(raw), "segredo") {
		t.Fatalf("o histórico guardou valores: %s", raw)
	}
	if got[0]["autor_id"] == nil {
		t.Fatalf("o histórico diz quem fez: %v", got[0])
	}

	// a confidential issue: its activity is seen only by who sees the issue
	ana.expect("POST", "/_ge/api/projetos/1/issues", map[string]any{"titulo": "Falha de segurança", "confidencial": true}, 201)
	for _, a := range activities(t, bia, "") {
		if a["resumo"] == "Falha de segurança" {
			t.Fatalf("a atividade de uma issue confidencial apareceu para quem não a vê: %v", a)
		}
	}
	found := false
	for _, a := range activities(t, ana, "") {
		found = found || a["resumo"] == "Falha de segurança"
	}
	if !found {
		t.Fatal("a autora deveria ver a atividade da sua issue confidencial")
	}

	// nobody edits or deletes history
	id := got[0]["id"]
	if code, _, _ := bia.do("DELETE", "/_ge/api/atividades/"+jsonID(id), nil); code < 400 {
		t.Fatalf("excluir atividade: %d", code)
	}
	if code, _, _ := ana.do("PUT", "/_ge/api/atividades/"+jsonID(id), map[string]any{"resumo": "x"}); code < 400 {
		t.Fatalf("editar atividade: %d", code)
	}

	// deleting a record: its history stays, without its title, seen through the project
	ana.expect("DELETE", "/_ge/api/projetos/1/issues/2", nil, 204)
	for _, a := range activities(t, bia, "?recurso=issue&recurso_id=2") {
		if a["resumo"] != nil && a["resumo"] != "" {
			t.Fatalf("o título de uma issue excluída continua no histórico: %v", a)
		}
	}
	if acts := activities(t, bia, "?recurso=issue&recurso_id=2"); actions(acts) != "criar issue,excluir issue" {
		t.Fatalf("a exclusão fica no histórico: %v", acts)
	}
}

func jsonID(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
