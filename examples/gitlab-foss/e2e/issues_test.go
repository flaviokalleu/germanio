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
	ada.must("POST", "/api/v4/projects/"+pid+"/labels", map[string]any{"name": "bug", "color": "#ff0000"}, 201)
	eve.must("POST", "/api/v4/projects/"+pid+"/labels", map[string]any{"name": "wontfix"}, 403)

	i1 := bob.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Crash on start", "description": "stack", "labels": "bug"}, 201)
	if l, _ := i1["labels"].([]any); len(l) != 1 || l[0] != "bug" {
		t.Fatalf("labels por nome: %v", i1["labels"])
	}
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
	// Pendências (GEP 0009): quem foi atribuído recebe uma; quem atribuiu, não
	pend := func(who *api) []any {
		t.Helper()
		code, body, _ := who.call("GET", "/_ge/api/pendencias", nil)
		l, _ := body.([]any)
		if code != 200 {
			t.Fatalf("pendências: %d %v", code, body)
		}
		return l
	}
	if l := pend(bob); len(l) != 1 || l[0].(map[string]any)["recurso"] != "issue" || l[0].(map[string]any)["estado"] != "aberta" {
		t.Fatalf("o responsável deveria ter uma pendência aberta da issue: %v", l)
	}
	if l := pend(ada); len(l) != 0 {
		t.Fatalf("quem atribui não recebe pendência para si: %v", l)
	}
	if l := ada.list("/api/v4/projects/" + pid + "/issues?assignee_ids=" + jsonNum(bobID)); len(l) != 1 {
		t.Fatalf("filtro por responsável: %v", l)
	}
	if l := ada.list("/api/v4/projects/" + pid + "/issues?labels=bug"); len(l) != 1 {
		t.Fatalf("filtro por label: %v", l)
	}
	if l := ada.list("/api/v4/projects/" + pid + "/issues?labels=nenhuma"); len(l) != 0 {
		t.Fatalf("filtro por label inexistente: %v", l)
	}
	// Label nova pelo nome: criada no projeto por quem pode (reporter+); guest não cria.
	up := ada.must("PUT", "/api/v4/projects/"+pid+"/issues/2", map[string]any{"labels": "docs,bug"}, 200)
	if l, _ := up["labels"].([]any); len(l) != 2 || l[0] != "docs" || l[1] != "bug" {
		t.Fatalf("labels criadas pelo nome: %v", up["labels"])
	}
	if ls := ada.list("/api/v4/projects/" + pid + "/labels"); len(ls) != 2 {
		t.Fatalf("labels do projeto: %v", ls)
	}
	eve.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "x", "labels": "inventada"}, 400)
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
	// Quem sai dos responsáveis perde a pendência aberta; eve não vê a de bob
	if l := pend(eve); len(l) != 0 {
		t.Fatalf("pendências são privadas: %v", l)
	}
	ada.must("PUT", "/api/v4/projects/"+pid+"/issues/1", map[string]any{"assignee_ids": []any{}}, 200)
	if l := pend(bob); len(l) != 0 {
		t.Fatalf("quem saiu dos responsáveis mantém a pendência: %v", l)
	}
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
	secret := ada.must("POST", "/api/v4/projects/"+id(priv)+"/labels", map[string]any{"name": "segredo", "color": "#123456"}, 201)
	mine := eve.must("POST", "/api/v4/projects", map[string]any{"name": "Mine", "path": "mine"}, 201)
	eve.must("POST", "/api/v4/projects/"+id(mine)+"/issues", map[string]any{"title": "x", "labels": []any{secret["id"]}}, 400)
	i := eve.must("POST", "/api/v4/projects/"+id(mine)+"/issues", map[string]any{"title": "y"}, 201)
	eve.must("PUT", "/api/v4/projects/"+id(mine)+"/issues/"+jsonNum(i["iid"]), map[string]any{"labels": []any{secret["id"]}}, 400)
	// Pelo nome, é outra label: a do próprio projeto (criada), nunca a de Ada.
	eve.must("PUT", "/api/v4/projects/"+id(mine)+"/issues/"+jsonNum(i["iid"]), map[string]any{"labels": "segredo"}, 200)
	for _, l := range eve.list("/api/v4/projects/" + id(mine) + "/labels") {
		if l.(map[string]any)["color"] == "#123456" {
			t.Fatalf("label do projeto privado vazou: %v", l)
		}
	}
}

// Milestones: criados por reporter+, reúnem issues do mesmo projeto, fecham e reabrem.
func TestMilestones(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	eve := signup(t, base, "eve")
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Road", "path": "road", "visibility": "public"}, 201)
	pid := id(p)
	eve.must("POST", "/api/v4/projects/"+pid+"/milestones", map[string]any{"title": "v1"}, 403)
	m := ada.must("POST", "/api/v4/projects/"+pid+"/milestones", map[string]any{"title": "v1", "due_date": "2026-12-31"}, 201)
	if m["state"] != "active" || m["due_date"] != "2026-12-31" {
		t.Fatalf("milestone: %v", m)
	}
	i := ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "entrega", "milestone_id": m["id"]}, 201)
	ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "outra"}, 201)
	if l := ada.list("/api/v4/projects/" + pid + "/issues?milestone_id=" + id(m)); len(l) != 1 || l[0].(map[string]any)["iid"] != i["iid"] {
		t.Fatalf("filtro por milestone: %v", l)
	}
	// O milestone de outro projeto não entra (nem vaza).
	q := eve.must("POST", "/api/v4/projects", map[string]any{"name": "Other", "path": "other"}, 201)
	eve.must("POST", "/api/v4/projects/"+id(q)+"/issues", map[string]any{"title": "x", "milestone_id": m["id"]}, 400)
	if c := ada.must("POST", "/api/v4/projects/"+pid+"/milestones/"+id(m)+"/close", nil, 200); c["state"] != "closed" {
		t.Fatalf("fechar milestone: %v", c)
	}
	if r := ada.must("POST", "/api/v4/projects/"+pid+"/milestones/"+id(m)+"/reopen", nil, 200); r["state"] != "active" {
		t.Fatalf("reabrir milestone: %v", r)
	}
}

// Busca geral: um lugar, vários tipos; cada resultado respeita quem pode ver.
func TestBuscaGeral(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	eve := signup(t, base, "eve")
	pub := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Farol Público", "path": "farol", "visibility": "public"}, 201)
	priv := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Farol Secreto", "path": "farol-secreto"}, 201)
	ada.must("POST", "/api/v4/projects/"+id(pub)+"/issues", map[string]any{"title": "farol quebrado"}, 201)
	ada.must("POST", "/api/v4/projects/"+id(priv)+"/issues", map[string]any{"title": "farol secreto"}, 201)

	if l := ada.list("/api/v4/search?scope=issues&search=farol"); len(l) != 2 {
		t.Fatalf("Ada vê as duas issues: %v", l)
	}
	if l := eve.list("/api/v4/search?scope=issues&search=farol"); len(l) != 1 || l[0].(map[string]any)["title"] != "farol quebrado" {
		t.Fatalf("Eve só vê a pública: %v", l)
	}
	if l := eve.list("/api/v4/search?scope=projects&search=farol"); len(l) != 1 {
		t.Fatalf("projetos visíveis para Eve: %v", l)
	}
	if l := ada.list("/api/v4/search?scope=merge_requests&search=farol"); len(l) != 0 {
		t.Fatalf("nenhum merge request: %v", l)
	}
	ada.must("GET", "/api/v4/search?scope=wikis&search=farol", nil, 400)
}

// Administrar milestones não dá poder sobre as issues que apenas estão num
// milestone: só maintainer exclui issues (regressão).
func TestAdministrarNaoGovernaReferenciaOpcional(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	bob := signup(t, base, "bob")
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "P", "path": "p"}, 201)
	bobID := bob.must("GET", "/api/v4/user", nil, 200)["id"]
	ada.must("POST", "/api/v4/projects/"+id(p)+"/members", map[string]any{"user_id": bobID, "access_level": 20}, 201)
	m := bob.must("POST", "/api/v4/projects/"+id(p)+"/milestones", map[string]any{"title": "v1"}, 201) // reporter administra milestones
	ada.must("POST", "/api/v4/projects/"+id(p)+"/issues", map[string]any{"title": "x", "milestone_id": m["id"]}, 201)
	bob.must("DELETE", "/api/v4/projects/"+id(p)+"/issues/1", nil, 403)
	ada.must("DELETE", "/api/v4/projects/"+id(p)+"/issues/1", nil, 204)
}
