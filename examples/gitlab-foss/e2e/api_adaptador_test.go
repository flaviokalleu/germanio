package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// NT-02: os eventos de um projeto (/projects/:id/events) são o histórico do
// que pertence a ele, com os nomes de ação do GitLab, páginas e sem vazar.
func TestEventosDoProjeto(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	eve := signup(t, base, "eve")
	pub := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Aberto", "path": "aberto", "visibility": "public"}, 201)
	pid := id(pub)
	ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Um"}, 201)
	ada.must("POST", "/api/v4/projects/"+pid+"/issues/1/close", nil, 200)
	ada.must("POST", "/api/v4/projects/"+pid+"/issues/1/reopen", nil, 200)
	ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Dois", "confidential": true}, 201)
	priv := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Fechado", "path": "fechado"}, 201)
	ada.must("POST", "/api/v4/projects/"+id(priv)+"/issues", map[string]any{"title": "Interno"}, 201)

	names := func(l []any) string {
		var out []string
		for i := len(l) - 1; i >= 0; i-- { // oldest first
			ev := l[i].(map[string]any)
			if ev["project_id"] == nil || ev["target_type"] != "Issue" || ev["acao"] != nil || ev["dentro"] != nil {
				t.Fatalf("evento no formato do GitLab: %v", ev)
			}
			out = append(out, ev["action_name"].(string)+" "+ev["target_title"].(string))
		}
		return strings.Join(out, ", ")
	}
	if got := names(ada.list("/api/v4/projects/" + pid + "/events")); got != "opened #1 Um, closed #1 Um, reopened #1 Um, opened #2 Dois" {
		t.Fatalf("eventos do projeto para a dona: %s", got)
	}
	// pelo endereço do projeto, como os clientes do GitLab também pedem
	if got := names(ada.list("/api/v4/projects/ada%2Faberto/events")); !strings.HasPrefix(got, "opened #1 Um") {
		t.Fatalf("eventos pelo endereço: %s", got)
	}
	if got := names(eve.list("/api/v4/projects/" + pid + "/events")); strings.Contains(got, "Dois") || !strings.Contains(got, "reopened #1 Um") {
		t.Fatalf("a issue confidencial não aparece para quem não pode vê-la: %s", got)
	}
	// páginas, com os cabeçalhos do GitLab
	code, body, h := ada.call("GET", "/api/v4/projects/"+pid+"/events?per_page=2&page=2", nil)
	if l, _ := body.([]any); code != 200 || len(l) != 2 || h.Get("X-Total") != "4" || h.Get("X-Page") != "2" {
		t.Fatalf("página 2 dos eventos: %d %v %v", code, body, h)
	}
	// um projeto privado não existe para quem não é membro
	eve.must("GET", "/api/v4/projects/"+id(priv)+"/events", nil, 404)
}

// AD-01: as estatísticas da instância, só para o administrador, contam como
// os indicadores (GEP 0012).
func TestEstatisticasDaAplicacao(t *testing.T) {
	base := gitlab(t)
	anon := &api{t: t, base: base}
	tok := anon.must("POST", "/oauth/token", map[string]any{"grant_type": "password", "username": "root", "password": "rootpassword1"}, 200)
	root := &api{t: t, base: base, token: tok["access_token"].(string)}
	ada := signup(t, base, "ada")
	signup(t, base, "eve")
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Aberto", "path": "aberto", "visibility": "public"}, 201)
	ada.must("POST", "/api/v4/projects/"+id(p)+"/issues", map[string]any{"title": "Um"}, 201)
	ada.must("POST", "/api/v4/projects/"+id(p)+"/issues", map[string]any{"title": "Dois"}, 201)
	ada.must("POST", "/api/v4/projects/"+id(p)+"/issues/1/notes", map[string]any{"body": "olá"}, 201)
	priv := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Fechado", "path": "fechado"}, 201)
	ada.must("POST", "/api/v4/projects/"+id(priv)+"/issues", map[string]any{"title": "Três"}, 201)

	st := root.must("GET", "/api/v4/application/statistics", nil, 200)
	want := map[string]string{"users": "3", "active_users": "3", "projects": "2", "issues": "3", "notes": "1", "merge_requests": "0", "groups": "0", "forks": "0"}
	for k, v := range want {
		if st[k] != v {
			t.Fatalf("estatísticas: %s = %v, esperado %q (%v)", k, st[k], v, st)
		}
	}
	ada.must("GET", "/api/v4/application/statistics", nil, 403)
	anon.must("GET", "/api/v4/application/statistics", nil, 401)
}

// NT-03: a pessoa desliga os avisos por e-mail (preferência de notificação do
// GitLab); a pendência continua, o e-mail não vai.
func TestPreferenciaDeAvisos(t *testing.T) {
	mail := t.TempDir()
	t.Setenv("GERMANIO_CORREIO_PASTA", mail)
	t.Setenv("GERMANIO_URL_PUBLICA", "https://gitlab.example")
	base := gitlab(t)
	ada := signup(t, base, "ada")
	bob := signup(t, base, "bob")
	cid := signup(t, base, "cid")
	bobID := bob.must("GET", "/api/v4/user", nil, 200)["id"]
	cidID := cid.must("GET", "/api/v4/user", nil, 200)["id"]

	if s := bob.must("GET", "/api/v4/notification_settings", nil, 200); s["level"] != "participating" || s["notification_email"] != "bob@example.com" {
		t.Fatalf("preferência inicial: %v", s)
	}
	if s := bob.must("PUT", "/api/v4/notification_settings?level=disabled", nil, 200); s["level"] != "disabled" {
		t.Fatalf("desligar avisos: %v", s)
	}
	bob.must("PUT", "/api/v4/notification_settings?level=barulho", nil, 400)
	if s := bob.must("GET", "/api/v4/notification_settings", nil, 200); s["level"] != "disabled" {
		t.Fatalf("a preferência fica guardada: %v", s)
	}
	// é da própria pessoa: os outros nem a veem
	if u := ada.must("GET", "/api/v4/users/"+jsonNum(bobID), nil, 200); u["avisos_por_email"] != nil {
		t.Fatalf("a preferência de outra pessoa é privada: %v", u)
	}
	(&api{t: t, base: base}).must("GET", "/api/v4/notification_settings", nil, 401)

	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Tracker", "path": "tracker", "visibility": "public"}, 201)
	ada.must("POST", "/api/v4/projects/"+id(p)+"/issues", map[string]any{"title": "Crash", "assignee_ids": []any{bobID, cidID}}, 201)
	var files []string
	for deadline := time.Now().Add(5 * time.Second); len(files) == 0 && time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		files, _ = filepath.Glob(filepath.Join(mail, "*.txt"))
	}
	time.Sleep(200 * time.Millisecond)
	files, _ = filepath.Glob(filepath.Join(mail, "*.txt"))
	if len(files) != 1 {
		t.Fatalf("só cid quer e-mail; vieram %d", len(files))
	}
	if b, _ := os.ReadFile(files[0]); !strings.Contains(string(b), "Para: cid@example.com") {
		t.Fatalf("e-mail de aviso:\n%s", b)
	}
	if code, body, _ := bob.call("GET", "/_ge/api/pendencias", nil); code != 200 || len(body.([]any)) != 1 {
		t.Fatalf("a pendência de bob continua: %d %v", code, body)
	}
	// religar volta a mandar
	if s := bob.must("PUT", "/api/v4/notification_settings", map[string]any{"level": "global"}, 200); s["level"] != "participating" {
		t.Fatalf("religar avisos: %v", s)
	}
}
