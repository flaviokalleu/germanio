package e2e

import (
	"net/http"
	"strings"
	"sync"
	"testing"
)

// IS-09 (controle de tempo): a estimativa é um campo da issue e cada tempo
// gasto é um registro da issue; as rotas do GitLab (time_estimate,
// add_spent_time, time_stats…) só traduzem o formato humano das durações.
func TestControleDeTempo(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	eve := signup(t, base, "eve")
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Tracker", "path": "tracker", "visibility": "public"}, 201)
	pid := id(p)
	ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Refazer o login"}, 201)
	issue := "/api/v4/projects/" + pid + "/issues/1"

	stats := func(m map[string]any, estimate, spent float64, hEstimate, hSpent any) {
		t.Helper()
		if m["time_estimate"] != estimate || m["total_time_spent"] != spent || m["human_time_estimate"] != hEstimate || m["human_total_time_spent"] != hSpent {
			t.Fatalf("time_stats: %v (esperado %v %v %v %v)", m, estimate, spent, hEstimate, hSpent)
		}
	}
	stats(eve.must("GET", issue+"/time_stats", nil, 200), 0, 0, nil, nil)
	stats(ada.must("POST", issue+"/time_estimate?duration=3h30m", nil, 200), 12600, 0, "3h 30m", nil)
	stats(ada.must("POST", issue+"/time_estimate", map[string]any{"duration": "1w 2d"}, 200), 201600, 0, "1w 2d", nil)
	if i := ada.must("GET", issue, nil, 200); i["time_estimate"] != float64(201600) {
		t.Fatalf("a estimativa é um campo da issue: %v", i)
	}
	stats(ada.must("POST", issue+"/add_spent_time?duration=1h", nil, 201), 201600, 3600, "1w 2d", "1h")
	stats(ada.must("POST", issue+"/add_spent_time", map[string]any{"duration": "45m", "summary": "revisão"}, 201), 201600, 6300, "1w 2d", "1h 45m")
	stats(ada.must("POST", issue+"/add_spent_time?duration=-15m", nil, 201), 201600, 5400, "1w 2d", "1h 30m")
	if m := ada.must("POST", issue+"/add_spent_time?duration=-9h", nil, 400); !strings.Contains(jsonNum(m), "Time to subtract exceeds the total time spent") {
		t.Fatalf("descontar mais do que o gasto: %v", m)
	}
	ada.must("POST", issue+"/add_spent_time?duration=muito", nil, 400)
	ada.must("POST", issue+"/time_estimate?duration=-1h", nil, 400)

	// quem não pode estimar nem registrar tempo é recusado, e nada muda
	eve.must("POST", issue+"/time_estimate?duration=1h", nil, 403)
	eve.must("POST", issue+"/add_spent_time?duration=1h", nil, 403)
	stats(eve.must("GET", issue+"/time_stats", nil, 200), 201600, 5400, "1w 2d", "1h 30m")

	stats(ada.must("POST", issue+"/reset_spent_time", nil, 200), 201600, 0, "1w 2d", nil)
	stats(ada.must("POST", issue+"/reset_time_estimate", nil, 200), 0, 0, nil, nil)

	// uma issue que a pessoa não vê não existe para ela
	conf := ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Segredo", "confidential": true}, 201)
	secret := "/api/v4/projects/" + pid + "/issues/" + jsonNum(conf["iid"])
	ada.must("POST", secret+"/add_spent_time?duration=2h", nil, 201)
	eve.must("GET", secret+"/time_stats", nil, 404)
	eve.must("POST", secret+"/add_spent_time?duration=1h", nil, 404)
	all := eve.list("/_ge/api/tempos_gastos")
	for _, it := range all {
		if it.(map[string]any)["issue_id"] == conf["id"] {
			t.Fatalf("tempos de uma issue confidencial não aparecem para quem não a vê: %v", all)
		}
	}
	if len(all) != 4 {
		t.Fatalf("os tempos da issue pública aparecem: %v", all)
	}
}

// Descontar tempo ao mesmo tempo (FASE 4, obstáculo 7; GEP 0050): a regra
// "não pode ficar com tempo gasto negativo" é conferida pelo domínio na
// mesma mudança que grava o tempo, com a issue travada; antes, o adaptador
// lia o total e depois gravava, e descontos simultâneos passavam todos.
func TestDescontarTempoAoMesmoTempo(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	pid := id(ada.must("POST", "/api/v4/projects", map[string]any{"name": "Relogio", "path": "relogio"}, 201))
	ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Medir"}, 201)
	issue := "/api/v4/projects/" + pid + "/issues/1"
	ada.must("POST", issue+"/add_spent_time?duration=1h", nil, 201)

	// oito descontos de 30 minutos ao mesmo tempo com 1 hora gasta: dois passam
	var wg sync.WaitGroup
	var mu sync.Mutex
	statuses := map[int]int{}
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			req, _ := http.NewRequest("POST", base+issue+"/add_spent_time?duration=-30m", nil)
			req.Header.Set("Authorization", "Bearer "+ada.token)
			resp, err := http.DefaultClient.Do(req)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				statuses[-1]++
				return
			}
			resp.Body.Close()
			statuses[resp.StatusCode]++
		}()
	}
	close(start)
	wg.Wait()
	if statuses[201] != 2 || statuses[400] != 6 {
		t.Fatalf("descontos ao mesmo tempo: %v (esperado 2×201 e 6×400)", statuses)
	}
	if m := ada.must("GET", issue+"/time_stats", nil, 200); m["total_time_spent"] != float64(0) {
		t.Fatalf("o tempo gasto ficou %v", m["total_time_spent"])
	}
}
