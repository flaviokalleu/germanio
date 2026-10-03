package e2e

import "testing"

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
	ada.must("POST", issue+"/add_spent_time?duration=-9h", nil, 400)
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
