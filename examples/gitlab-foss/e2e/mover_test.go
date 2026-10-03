package e2e

import "testing"

// IS-09 (mover): uma issue muda de projeto (GEP 0034). Quem move precisa
// poder editá-la onde está e criar issues no destino; ela ganha o próximo
// número do destino; comentários vão junto; labels seguem pelo nome e o
// milestone do projeto antigo fica para trás.
func TestMoverIssue(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	eve := signup(t, base, "eve")
	bob := signup(t, base, "bob")
	web := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Web", "path": "web", "visibility": "public"}, 201)
	apiP := ada.must("POST", "/api/v4/projects", map[string]any{"name": "API", "path": "api", "visibility": "public"}, 201)
	priv := bob.must("POST", "/api/v4/projects", map[string]any{"name": "Cofre", "path": "cofre"}, 201)
	wid, aid := id(web), id(apiP)
	ada.must("POST", "/api/v4/projects/"+wid+"/labels", map[string]any{"name": "bug"}, 201)
	ada.must("POST", "/api/v4/projects/"+wid+"/labels", map[string]any{"name": "ux"}, 201)
	ada.must("POST", "/api/v4/projects/"+aid+"/labels", map[string]any{"name": "bug"}, 201)
	m := ada.must("POST", "/api/v4/projects/"+wid+"/milestones", map[string]any{"title": "v1"}, 201)
	ada.must("POST", "/api/v4/projects/"+wid+"/issues", map[string]any{"title": "Login quebrado", "labels": "bug,ux", "milestone_id": m["id"]}, 201)
	ada.must("POST", "/api/v4/projects/"+wid+"/issues/1/notes", map[string]any{"body": "acontece no celular"}, 201)
	ada.must("POST", "/api/v4/projects/"+aid+"/issues", map[string]any{"title": "Erro 500"}, 201)

	moved := ada.must("POST", "/api/v4/projects/"+wid+"/issues/1/move", map[string]any{"to_project_id": apiP["id"]}, 201)
	if moved["iid"] != float64(2) || moved["project_id"] != apiP["id"] || moved["title"] != "Login quebrado" {
		t.Fatalf("a issue ganha o próximo número do destino: %v", moved)
	}
	if l, _ := moved["labels"].([]any); len(l) != 1 || l[0] != "bug" {
		t.Fatalf("labels seguem pelo nome, só as que o destino tem: %v", moved["labels"])
	}
	if moved["milestone_id"] != nil {
		t.Fatalf("o milestone do projeto antigo fica para trás: %v", moved)
	}
	if notes := ada.list("/api/v4/projects/" + aid + "/issues/2/notes"); len(notes) != 1 {
		t.Fatalf("os comentários vão com a issue: %v", notes)
	}
	ada.must("GET", "/api/v4/projects/"+wid+"/issues/1", nil, 404)
	// o próximo número do destino não repete
	if n := ada.must("POST", "/api/v4/projects/"+aid+"/issues", map[string]any{"title": "Outra"}, 201); n["iid"] != float64(3) {
		t.Fatalf("a numeração do destino continua: %v", n)
	}
	// o feed do destino conta a mudança
	found := false
	for _, it := range ada.list("/api/v4/projects/" + aid + "/events") {
		ev := it.(map[string]any)
		found = found || (ev["action_name"] == "updated" && ev["target_title"] == "#2 Login quebrado")
	}
	if !found {
		t.Fatalf("a mudança aparece nos eventos do projeto de destino")
	}

	// quem não pode editar a issue onde ela está é recusado
	eve.must("POST", "/api/v4/projects/"+aid+"/issues/2/move", map[string]any{"to_project_id": web["id"]}, 403)
	// quem não vê o destino: para essa pessoa, ele não existe
	ada.must("POST", "/api/v4/projects/"+aid+"/issues/2/move", map[string]any{"to_project_id": priv["id"]}, 404)
	ada.must("POST", "/api/v4/projects/"+aid+"/issues/2/move", map[string]any{"to_project_id": apiP["id"]}, 400)
	ada.must("POST", "/api/v4/projects/"+aid+"/issues/2/move", nil, 400)
	// a autora de uma issue a leva para onde ela pode criar issues
	eve.must("POST", "/api/v4/projects/"+aid+"/issues", map[string]any{"title": "Sugestão"}, 201)
	if mv := eve.must("POST", "/api/v4/projects/"+aid+"/issues/4/move", map[string]any{"to_project_id": web["id"]}, 201); mv["iid"] != float64(2) { // Web already gave number 1 to the issue that left: numbers are never reused
		t.Fatalf("a issue de eve no projeto público Web: %v", mv)
	}
	eve.must("POST", "/api/v4/projects/"+wid+"/issues/2/move", map[string]any{"to_project_id": priv["id"]}, 404)
	// um projeto arquivado não recebe nem deixa sair
	ada.must("PUT", "/api/v4/projects/"+wid, map[string]any{"archived": true}, 200)
	ada.must("POST", "/api/v4/projects/"+aid+"/issues/2/move", map[string]any{"to_project_id": web["id"]}, 403)
	ada.must("POST", "/api/v4/projects/"+wid+"/issues/2/move", map[string]any{"to_project_id": apiP["id"]}, 403)
}
