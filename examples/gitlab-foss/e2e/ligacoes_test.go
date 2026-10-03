package e2e

import (
	"sort"
	"strings"
	"testing"
)

// IS-09 (ligações): relacionar duas issues, de qualquer projeto. A pessoa
// precisa ver as duas; quem não vê uma delas não vê a ligação; a ligação sai
// com qualquer uma das issues.
func TestLigacoesEntreIssues(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	eve := signup(t, base, "eve")
	web := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Web", "path": "web", "visibility": "public"}, 201)
	apiP := ada.must("POST", "/api/v4/projects", map[string]any{"name": "API", "path": "api", "visibility": "public"}, 201)
	priv := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Interno", "path": "interno"}, 201)
	ada.must("POST", "/api/v4/projects/"+id(web)+"/issues", map[string]any{"title": "Botão quebrado"}, 201)
	ada.must("POST", "/api/v4/projects/"+id(apiP)+"/issues", map[string]any{"title": "Erro 500"}, 201)
	ada.must("POST", "/api/v4/projects/"+id(apiP)+"/issues", map[string]any{"title": "Lento"}, 201)
	ada.must("POST", "/api/v4/projects/"+id(priv)+"/issues", map[string]any{"title": "Plano secreto"}, 201)
	webIssue := "/api/v4/projects/" + id(web) + "/issues/1"

	// ligar a issue da Web à issue 1 da API (outro projeto)
	l := ada.must("POST", webIssue+"/links", map[string]any{"target_project_id": apiP["id"], "target_issue_iid": 1}, 201)
	if l["link_type"] != "relates_to" || l["source_issue"].(map[string]any)["title"] != "Botão quebrado" || l["target_issue"].(map[string]any)["title"] != "Erro 500" {
		t.Fatalf("ligação criada: %v", l)
	}
	// também pelo endereço do projeto, como os clientes do GitLab pedem
	ada.must("POST", "/api/v4/projects/ada%2Fapi/issues/2/links?target_project_id="+id(priv)+"&target_issue_iid=1", nil, 201)

	titles := func(who *api, path string) string {
		t.Helper()
		var out []string
		for _, it := range who.list(path) {
			m := it.(map[string]any)
			if m["issue_link_id"] == nil || m["link_type"] != "relates_to" {
				t.Fatalf("issue ligada no formato do GitLab: %v", m)
			}
			out = append(out, m["title"].(string))
		}
		sort.Strings(out)
		return strings.Join(out, ",")
	}
	// a ligação aparece dos dois lados
	if got := titles(eve, webIssue+"/links"); got != "Erro 500" {
		t.Fatalf("ligações da issue da Web: %s", got)
	}
	if got := titles(eve, "/api/v4/projects/"+id(apiP)+"/issues/1/links"); got != "Botão quebrado" {
		t.Fatalf("ligações vistas do outro lado: %s", got)
	}
	// quem não vê o projeto interno não vê a ligação com ele
	if got := titles(ada, "/api/v4/projects/"+id(apiP)+"/issues/2/links"); got != "Plano secreto" {
		t.Fatalf("a dona vê a ligação com o projeto interno: %s", got)
	}
	if got := titles(eve, "/api/v4/projects/"+id(apiP)+"/issues/2/links"); got != "" {
		t.Fatalf("a ligação com uma issue que eve não vê não aparece: %s", got)
	}
	for _, it := range eve.list("/api/v4/ligacoes") {
		if it.(map[string]any)["relacionada_id"] != nil && strings.Contains(jsonNum(it), `"issue_id":3`) {
			t.Fatalf("a ligação com o projeto interno vazou na lista geral: %v", it)
		}
	}

	// ninguém liga uma issue que não vê: para eve, ela não existe
	eve.must("POST", webIssue+"/links", map[string]any{"target_project_id": priv["id"], "target_issue_iid": 1}, 404)
	// ligar exige ao menos guest no projeto; eve não é membro da Web
	eve.must("POST", webIssue+"/links", map[string]any{"target_project_id": apiP["id"], "target_issue_iid": 2}, 403)
	ada.must("POST", webIssue+"/links", map[string]any{"target_project_id": apiP["id"]}, 400)
	ada.must("POST", webIssue+"/links", map[string]any{"target_project_id": apiP["id"], "target_issue_iid": 2, "link_type": "blocks"}, 400)

	// desfazer: pela issue de qualquer lado; quem não pode é recusado
	lid := jsonNum(eve.list(webIssue + "/links")[0].(map[string]any)["issue_link_id"])
	eve.must("DELETE", webIssue+"/links/"+lid, nil, 403)
	ada.must("DELETE", "/api/v4/projects/"+id(web)+"/issues/1/links/9999", nil, 404)
	if d := ada.must("DELETE", "/api/v4/projects/"+id(apiP)+"/issues/1/links/"+lid, nil, 200); d["target_issue"].(map[string]any)["title"] != "Botão quebrado" {
		t.Fatalf("ligação desfeita: %v", d)
	}
	if got := titles(ada, webIssue+"/links"); got != "" {
		t.Fatalf("a ligação foi desfeita: %s", got)
	}

	// excluir uma das issues leva a ligação junto
	ada.must("DELETE", "/api/v4/projects/"+id(priv)+"/issues/1", nil, 204)
	if got := titles(ada, "/api/v4/projects/"+id(apiP)+"/issues/2/links"); got != "" {
		t.Fatalf("a ligação sai com a issue excluída: %s", got)
	}
}
