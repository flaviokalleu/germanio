package e2e

import (
	"strings"
	"testing"
)

// Fluxo 3: create issue → comment → assign → close.
func TestFluxo3Issues(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	bob := signup(t, base, "bob")
	eve := signup(t, base, "eve")
	bobID := bob.must("GET", "/api/v4/user", nil, 200)["id"]

	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Tracker", "path": "tracker", "visibility": "public"}, 201)
	pid := id(p)
	i1 := bob.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Crash on start", "description": "stack", "labels": "bug, ui"}, 201)
	i2 := ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Docs"}, 201)
	if i1["iid"].(float64) != 1 || i2["iid"].(float64) != 2 || i1["state"] != "opened" || i1["autor_id"] != bobID {
		t.Fatalf("iid por projeto / autor: %v %v", i1, i2)
	}
	if labels := i1["labels"].([]any); len(labels) != 2 || labels[0] != "bug" {
		t.Fatalf("labels: %v", i1["labels"])
	}
	// Issue é endereçada pelo número dentro do projeto
	got := eve.must("GET", "/api/v4/projects/"+pid+"/issues/1", nil, 200)
	if got["title"] != "Crash on start" {
		t.Fatalf("issue #1: %v", got)
	}
	// Comentar
	c := eve.must("POST", "/api/v4/projects/"+pid+"/issues/1/notes", map[string]any{"body": "same here"}, 201)
	if c["body"] != "same here" {
		t.Fatalf("comentário: %v", c)
	}
	if notes := ada.list("/api/v4/projects/" + pid + "/issues/1/notes"); len(notes) != 1 {
		t.Fatalf("comentários: %v", notes)
	}
	// Atribuir (dono do projeto), filtrar por responsável e label
	ada.must("PUT", "/api/v4/projects/"+pid+"/issues/1", map[string]any{"responsaveis": []any{bobID}}, 200)
	if l := ada.list("/api/v4/projects/" + pid + "/issues?responsaveis=" + jsonNum(bobID)); len(l) != 1 {
		t.Fatalf("filtro por responsável: %v", l)
	}
	if l := ada.list("/api/v4/projects/" + pid + "/issues?labels=ui"); len(l) != 1 {
		t.Fatalf("filtro por label: %v", l)
	}
	if l := ada.list("/api/v4/projects/" + pid + "/issues?search=docs"); len(l) != 1 || l[0].(map[string]any)["title"] != "Docs" {
		t.Fatalf("pesquisa: %v", l)
	}
	ada.must("PUT", "/api/v4/projects/"+pid+"/issues/1", map[string]any{"responsaveis": []any{9999}}, 400)
	// Quem não é autor nem planner não fecha
	eve.must("POST", "/api/v4/projects/"+pid+"/issues/1/fechar", nil, 403)
	closed := bob.must("POST", "/api/v4/projects/"+pid+"/issues/1/fechar", nil, 200)
	if closed["state"] != "closed" || closed["closed_by_id"] != bobID {
		t.Fatalf("fechar: %v", closed)
	}
	if l := ada.list("/api/v4/projects/" + pid + "/issues?state=opened"); len(l) != 1 {
		t.Fatalf("filtro por estado: %v", l)
	}
	reopened := ada.must("POST", "/api/v4/projects/"+pid+"/issues/1/reabrir", nil, 200)
	if reopened["state"] != "opened" || reopened["closed_at"] != nil {
		t.Fatalf("reabrir: %v", reopened)
	}
	// Confidencial: autor e reporter+ veem; outros não
	conf := bob.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Security hole", "confidential": true}, 201)
	path := "/api/v4/projects/" + pid + "/issues/" + jsonNum(conf["iid"])
	bob.must("GET", path, nil, 200)
	ada.must("GET", path, nil, 200)
	eve.must("GET", path, nil, 404)
	(&api{t: t, base: base}).must("GET", path, nil, 404)
	for _, it := range eve.list("/api/v4/projects/" + pid + "/issues") {
		if strings.Contains(it.(map[string]any)["title"].(string), "Security") {
			t.Fatal("issue confidencial listada para quem não pode ver")
		}
	}
	// Projeto privado: não-membro não cria issue nem descobre o projeto
	priv := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Secret", "path": "secret"}, 201)
	eve.must("POST", "/api/v4/projects/"+id(priv)+"/issues", map[string]any{"title": "x"}, 404)
	// Validação
	bob.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"description": "sem título"}, 400)
	// Labels: só reporter+ administra
	eve.must("POST", "/api/v4/projects/"+pid+"/labels", map[string]any{"name": "wontfix"}, 403)
	ada.must("POST", "/api/v4/projects/"+pid+"/labels", map[string]any{"name": "wontfix", "color": "#ff0000"}, 201)
}
