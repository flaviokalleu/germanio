package e2e

import (
	"strings"
	"testing"
)

// NT-02: o feed de eventos é o histórico do Germanio (GEP 0011) com os nomes
// da API do GitLab (ações como opened e closed, traduzidas pelo adaptador).
// Ninguém vê eventos do que não pode ver.
func TestAtividade(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	eve := signup(t, base, "eve")

	pub := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Aberto", "path": "aberto", "visibility": "public"}, 201)
	ada.must("POST", "/api/v4/projects/"+id(pub)+"/issues", map[string]any{"title": "Bug público"}, 201)
	ada.must("POST", "/api/v4/projects/"+id(pub)+"/issues/1/close", nil, 200)
	ada.must("POST", "/api/v4/projects/"+id(pub)+"/issues", map[string]any{"title": "Falha secreta", "confidential": true}, 201)
	priv := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Fechado", "path": "fechado"}, 201)
	ada.must("POST", "/api/v4/projects/"+id(priv)+"/issues", map[string]any{"title": "Plano interno"}, 201)

	titles := func(who *api) string {
		var out []string
		for _, it := range who.list("/api/v4/events") {
			ev := it.(map[string]any)
			if ev["action_name"] == nil || ev["target_type"] == nil || ev["author_id"] == nil {
				t.Fatalf("evento sem os nomes da API: %v", ev)
			}
			out = append(out, ev["action_name"].(string)+" "+ev["target_title"].(string))
		}
		return strings.Join(out, ",")
	}
	all := titles(ada)
	for _, want := range []string{"opened #1 Bug público", "closed #1 Bug público", "opened #2 Falha secreta", "opened #1 Plano interno"} {
		if !strings.Contains(all, want) {
			t.Fatalf("a autora deveria ver %q: %s", want, all)
		}
	}
	seen := titles(eve)
	if !strings.Contains(seen, "opened #1 Bug público") {
		t.Fatalf("eventos de projeto público são visíveis: %s", seen)
	}
	for _, hidden := range []string{"Falha secreta", "Plano interno"} {
		if strings.Contains(seen, hidden) {
			t.Fatalf("evento de %q vazou para quem não pode ver: %s", hidden, seen)
		}
	}
	// o histórico de um projeto: o que pertence a ele
	if l := eve.list("/api/v4/projects/" + id(pub) + "/events?target_type=issue"); len(l) != 2 {
		t.Fatalf("histórico do projeto público para quem não é membro: %v", l)
	}
}
