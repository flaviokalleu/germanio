package e2e

import (
	"net/http"
	"sync"
	"testing"
)

// GEP 0047 (indicadores de um registro): a milestone shows the total weight
// of its issues and an issue the total time spent on it, computed by the
// core with one aggregate query and counting only what the viewer sees.
func TestPesoTotalDaMilestone(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	eve := signup(t, base, "eve")
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Tracker", "path": "tracker", "visibility": "public"}, 201)
	pid := id(p)
	m := ada.must("POST", "/api/v4/projects/"+pid+"/milestones", map[string]any{"title": "v1.0"}, 201)
	mid := id(m)
	if m["total_weight"] != float64(0) {
		t.Fatalf("uma milestone sem issues pesa 0: %v", m)
	}
	ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Login", "weight": 5, "milestone_id": m["id"]}, 201)
	ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Busca", "weight": 3, "milestone_id": m["id"]}, 201)
	ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Segredo", "weight": 8, "milestone_id": m["id"], "confidential": true}, 201)
	ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Fora", "weight": 2}, 201)
	ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Sem peso", "milestone_id": m["id"]}, 201)

	milestone := "/api/v4/projects/" + pid + "/milestones/" + mid
	if got := ada.must("GET", milestone, nil, 200)["total_weight"]; got != float64(16) {
		t.Fatalf("a dona vê as issues confidenciais e soma o peso delas: %v", got)
	}
	// eve does not see the confidential issue: its weight does not count for her
	if got := eve.must("GET", milestone, nil, 200)["total_weight"]; got != float64(8) {
		t.Fatalf("quem não vê uma issue não soma o peso dela: %v", got)
	}
	if l := eve.list("/api/v4/projects/" + pid + "/milestones"); len(l) != 1 || l[0].(map[string]any)["total_weight"] != float64(8) {
		t.Fatalf("a lista de milestones mostra o peso total de quem vê: %v", l)
	}
	// it follows the issues: a new weight, an issue leaving the milestone
	ada.must("PUT", "/api/v4/projects/"+pid+"/issues/2", map[string]any{"weight": 4}, 200)
	ada.must("PUT", "/api/v4/projects/"+pid+"/issues/1", map[string]any{"milestone_id": nil}, 200)
	if got := eve.must("GET", milestone, nil, 200)["total_weight"]; got != float64(4) {
		t.Fatalf("o peso total segue as issues: %v", got)
	}
	// the number is never written
	ada.must("PUT", milestone, map[string]any{"total_weight": 99, "title": "v1.0.1"}, 200)
	if got := ada.must("GET", milestone, nil, 200); got["total_weight"] != float64(12) || got["title"] != "v1.0.1" {
		t.Fatalf("o peso total vem das issues, nunca da entrada: %v", got)
	}
}

func TestTempoGastoTotal(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	eve := signup(t, base, "eve")
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Tracker", "path": "tracker", "visibility": "public"}, 201)
	pid := id(p)
	ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Refazer o login"}, 201)
	issue := "/api/v4/projects/" + pid + "/issues/1"
	for _, d := range []string{"1h", "30m", "2h"} {
		ada.must("POST", issue+"/add_spent_time?duration="+d, nil, 201)
	}
	if got := eve.must("GET", issue, nil, 200)["total_time_spent"]; got != float64(12600) {
		t.Fatalf("a issue mostra o tempo gasto total: %v", got)
	}
	if got := eve.must("GET", issue+"/time_stats", nil, 200)["total_time_spent"]; got != float64(12600) {
		t.Fatalf("time_stats lê o indicador da issue: %v", got)
	}
	// many entries: the core sums them, nothing is paged by hand
	for i := 0; i < 20; i++ {
		ada.must("POST", issue+"/tempos_gastos", map[string]any{"duration": 60}, 201)
	}
	if got := eve.must("GET", issue+"/time_stats", nil, 200)["total_time_spent"]; got != float64(12600+20*60) {
		t.Fatalf("o total soma todos os registros: %v", got)
	}
	entries := func() int {
		t.Helper()
		_, _, h := ada.call("GET", issue+"/tempos_gastos?per_page=1", nil)
		return atoi(h.Get("X-Total"))
	}
	before := entries()

	// eve may not reset; many resets at once subtract once
	eve.must("POST", issue+"/reset_spent_time", nil, 403)
	var wg sync.WaitGroup
	statuses := make([]int, 8)
	start := make(chan struct{})
	for i := range statuses {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			req, _ := http.NewRequest("POST", base+issue+"/reset_spent_time", nil)
			req.Header.Set("Authorization", "Bearer "+ada.token)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return
			}
			resp.Body.Close()
			statuses[i] = resp.StatusCode
		}(i)
	}
	close(start)
	wg.Wait()
	for _, s := range statuses {
		if s != 200 {
			t.Fatalf("zerar ao mesmo tempo: %v", statuses)
		}
	}
	if got := ada.must("GET", issue+"/time_stats", nil, 200); got["total_time_spent"] != float64(0) || got["human_total_time_spent"] != nil {
		t.Fatalf("o tempo gasto termina zerado: %v", got)
	}
	if after := entries(); after != before+1 {
		t.Fatalf("oito pedidos ao mesmo tempo registram um só desconto: %d registros antes, %d depois", before, after)
	}
}

// GEP 0048 (único por par): an issue is never linked to itself, and two
// issues are linked once, in any order.
func TestLigacoesUnicas(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Web", "path": "web", "visibility": "public"}, 201)
	pid := id(p)
	ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Um"}, 201)
	ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Dois"}, 201)
	one := "/api/v4/projects/" + pid + "/issues/1/links"
	two := "/api/v4/projects/" + pid + "/issues/2/links"

	ada.must("POST", one, map[string]any{"target_project_id": p["id"], "target_issue_iid": 2}, 201)
	for _, path := range []string{one, two} {
		target := 2
		if path == two {
			target = 1
		}
		if dup := ada.must("POST", path, map[string]any{"target_project_id": p["id"], "target_issue_iid": target}, 409); dup["message"] != "Issue(s) already assigned" {
			t.Fatalf("ligação repetida (%s): %v", path, dup)
		}
	}
	ada.must("POST", one, map[string]any{"target_project_id": p["id"], "target_issue_iid": 1}, 400)
	if l := ada.list(one); len(l) != 1 {
		t.Fatalf("uma só ligação entre as duas issues: %v", l)
	}
	if l := ada.list(two); len(l) != 1 {
		t.Fatalf("a mesma ligação, vista do outro lado: %v", l)
	}
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return -1
		}
		n = n*10 + int(r-'0')
	}
	return n
}
