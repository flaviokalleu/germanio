package e2e

import (
	"strings"
	"testing"
)

// Fluxo 3: create issue → comment → assign → close (API com nomes do GitLab).
func TestFluxo3Issues(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	bob := signup(t, base, "bob")
	eve := signup(t, base, "eve")
	bobID := bob.must("GET", "/api/v4/user", nil, 200)["id"]

	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Tracker", "path": "tracker", "visibility": "public"}, 201)
	pid := id(p)
	bug := ada.must("POST", "/api/v4/projects/"+pid+"/labels", map[string]any{"name": "bug", "color": "#ff0000"}, 201)
	eve.must("POST", "/api/v4/projects/"+pid+"/labels", map[string]any{"name": "wontfix"}, 403)

	i1 := bob.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Crash on start", "description": "stack", "labels": []any{bug["id"]}}, 201)
	i2 := ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Docs"}, 201)
	if i1["iid"].(float64) != 1 || i2["iid"].(float64) != 2 || i1["state"] != "opened" || i1["author_id"] != bobID {
		t.Fatalf("iid por projeto / autor / estado inicial: %v %v", i1, i2)
	}
	if i1["estado"] != nil || i1["titulo"] != nil {
		t.Fatalf("nomes do domínio vazaram para a integração: %v", i1)
	}
	got := eve.must("GET", "/api/v4/projects/"+pid+"/issues/1", nil, 200)
	if got["title"] != "Crash on start" {
		t.Fatalf("issue #1: %v", got)
	}
	// Comentar (qualquer pessoa conectada em projeto público)
	c := eve.must("POST", "/api/v4/projects/"+pid+"/issues/1/notes", map[string]any{"body": "same here"}, 201)
	if c["body"] != "same here" {
		t.Fatalf("comentário: %v", c)
	}
	if notes := ada.list("/api/v4/projects/" + pid + "/issues/1/notes"); len(notes) != 1 {
		t.Fatalf("comentários: %v", notes)
	}
	// Atribuir e filtrar
	ada.must("PUT", "/api/v4/projects/"+pid+"/issues/1", map[string]any{"assignee_ids": []any{bobID}}, 200)
	if l := ada.list("/api/v4/projects/" + pid + "/issues?assignee_ids=" + jsonNum(bobID)); len(l) != 1 {
		t.Fatalf("filtro por responsável: %v", l)
	}
	if l := ada.list("/api/v4/projects/" + pid + "/issues?labels=" + jsonNum(bug["id"])); len(l) != 1 {
		t.Fatalf("filtro por label: %v", l)
	}
	if l := ada.list("/api/v4/projects/" + pid + "/issues?search=docs"); len(l) != 1 || l[0].(map[string]any)["title"] != "Docs" {
		t.Fatalf("pesquisa: %v", l)
	}
	ada.must("PUT", "/api/v4/projects/"+pid+"/issues/1", map[string]any{"assignee_ids": []any{9999}}, 400)
	// O estado só muda por ação, nunca por edição direta
	ada.must("PUT", "/api/v4/projects/"+pid+"/issues/1", map[string]any{"state": "closed"}, 200)
	if again := ada.must("GET", "/api/v4/projects/"+pid+"/issues/1", nil, 200); again["state"] != "opened" {
		t.Fatalf("estado não pode ser editado diretamente: %v", again)
	}
	// Fechar: autor ou planner+; outros não
	eve.must("POST", "/api/v4/projects/"+pid+"/issues/1/close", nil, 403)
	closed := bob.must("POST", "/api/v4/projects/"+pid+"/issues/1/close", nil, 200)
	if closed["state"] != "closed" || closed["closed_by_id"] != bobID || closed["closed_at"] == nil {
		t.Fatalf("fechar: %v", closed)
	}
	bob.must("POST", "/api/v4/projects/"+pid+"/issues/1/close", nil, 400)
	if l := ada.list("/api/v4/projects/" + pid + "/issues?state=opened"); len(l) != 1 {
		t.Fatalf("filtro por estado: %v", l)
	}
	reopened := ada.must("POST", "/api/v4/projects/"+pid+"/issues/1/reopen", nil, 200)
	if reopened["state"] != "opened" || reopened["closed_at"] != nil {
		t.Fatalf("reabrir: %v", reopened)
	}
	// Confidencial: autor, responsáveis e reporter+ veem; outros não
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
	// Validação: título obrigatório e até 255
	bob.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"description": "sem título"}, 400)
	bob.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": strings.Repeat("x", 256)}, 400)
	// Só maintainer exclui
	bob.must("DELETE", "/api/v4/projects/"+pid+"/issues/2", nil, 403)
	ada.must("DELETE", "/api/v4/projects/"+pid+"/issues/2", nil, 204)
}

// Issues de projeto privado nunca aparecem para quem não é membro,
// mesmo com pesquisa liberada.
func TestIssuesPrivadasNaoVazam(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	eve := signup(t, base, "eve")
	priv := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Secret", "path": "secret"}, 201)
	ada.must("POST", "/api/v4/projects/"+id(priv)+"/issues", map[string]any{"title": "plano secreto"}, 201)
	eve.must("GET", "/api/v4/projects/"+id(priv)+"/issues", nil, 404)
	eve.must("GET", "/api/v4/projects/"+id(priv)+"/issues/1", nil, 404)
	for _, it := range eve.list("/api/v4/issues?search=plano") {
		t.Fatalf("issue privada vazou na listagem geral: %v", it)
	}
	if l := ada.list("/api/v4/issues?search=plano"); len(l) != 1 {
		t.Fatalf("dona deve encontrar sua issue: %v", l)
	}
}

// Uma issue só usa labels do próprio projeto: labels de outro projeto
// (inclusive privado) são tratados como inexistentes.
func TestLabelsDeOutroProjetoNaoEntram(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	eve := signup(t, base, "eve")
	priv := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Secret", "path": "secret"}, 201)
	secret := ada.must("POST", "/api/v4/projects/"+id(priv)+"/labels", map[string]any{"name": "segredo"}, 201)
	mine := eve.must("POST", "/api/v4/projects", map[string]any{"name": "Mine", "path": "mine"}, 201)
	eve.must("POST", "/api/v4/projects/"+id(mine)+"/issues", map[string]any{"title": "x", "labels": []any{secret["id"]}}, 400)
	i := eve.must("POST", "/api/v4/projects/"+id(mine)+"/issues", map[string]any{"title": "y"}, 201)
	eve.must("PUT", "/api/v4/projects/"+id(mine)+"/issues/"+jsonNum(i["iid"]), map[string]any{"labels": []any{secret["id"]}}, 400)
}
