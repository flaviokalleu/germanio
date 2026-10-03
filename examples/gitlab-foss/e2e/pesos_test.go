package e2e

import "testing"

// IS-09 (pesos): o peso de uma issue é um dado comum — um número inteiro,
// nunca negativo, que se filtra como os outros campos.
func TestPesosDasIssues(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	eve := signup(t, base, "eve")
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Tracker", "path": "tracker", "visibility": "public"}, 201)
	pid := id(p)

	pesada := ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Refazer o login", "weight": 5}, 201)
	if pesada["weight"] != float64(5) || pesada["peso"] != nil {
		t.Fatalf("peso com o nome do GitLab: %v", pesada)
	}
	ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Corrigir um texto", "weight": 1}, 201)
	ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Sem peso"}, 201)

	if l := eve.list("/api/v4/projects/" + pid + "/issues?weight=5"); len(l) != 1 || l[0].(map[string]any)["title"] != "Refazer o login" {
		t.Fatalf("filtro por peso: %v", l)
	}
	// o peso muda pela edição, por quem pode editar
	ada.must("PUT", "/api/v4/projects/"+pid+"/issues/2", map[string]any{"weight": 3}, 200)
	if l := ada.list("/api/v4/projects/" + pid + "/issues?weight=3"); len(l) != 1 {
		t.Fatalf("peso editado: %v", l)
	}
	eve.must("PUT", "/api/v4/projects/"+pid+"/issues/2", map[string]any{"weight": 8}, 403)
	// peso negativo ou que não é número é recusado, campo a campo
	code, body, _ := ada.call("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "x", "weight": -1})
	if code != 400 {
		t.Fatalf("peso negativo: %d %v", code, body)
	}
	ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "x", "weight": "muito"}, 400)
}
