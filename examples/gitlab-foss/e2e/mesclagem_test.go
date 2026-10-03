package e2e

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

// MR-07: regra de aprovação (GEP 0026), squash, fast-forward e merge when
// pipeline succeeds (GEP 0027), pelos nomes da API do GitLab.
func TestAprovacoesEMetodosDeMesclagem(t *testing.T) {
	t.Setenv("GERMANIO_EXECUTOR", "local")
	base := gitlab(t)
	ada := signup(t, base, "ada") // owner: aprova e mescla na main
	bob := signup(t, base, "bob") // developer: propõe
	bobID := bob.must("GET", "/api/v4/user", nil, 200)["id"]
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Livro", "path": "livro", "initialize_with_readme": true}, 201)
	pid := id(p)
	if p["merge_method"] != "merge" {
		t.Fatalf("forma de mesclar padrão: %v", p["merge_method"])
	}
	ada.must("POST", "/api/v4/projects/"+pid+"/members", map[string]any{"user_id": bobID, "access_level": 30}, 201)

	pat := bob.must("POST", "/api/v4/personal_access_tokens", map[string]any{"name": "git"}, 201)
	u, _ := url.Parse(base)
	u.User = url.UserPassword("bob", pat["token"].(string))
	work := filepath.Join(t.TempDir(), "livro")
	run(t, filepath.Dir(work), "git", "clone", "--quiet", u.String()+"/ada/livro.git", "livro")
	run(t, work, "git", "config", "user.email", "bob@example.com")
	run(t, work, "git", "config", "user.name", "Bob")
	commit := func(branch, from, file string) {
		run(t, work, "git", "checkout", "--quiet", "-B", branch, from)
		os.WriteFile(filepath.Join(work, file), []byte(file+"\n"), 0o644)
		run(t, work, "git", "add", ".")
		run(t, work, "git", "commit", "--quiet", "-m", "add "+file)
		run(t, work, "git", "push", "--quiet", "origin", branch)
	}
	mrs := "/api/v4/projects/" + pid + "/merge_requests"
	head := func() map[string]any {
		return ada.list("/api/v4/projects/" + pid + "/repository/commits?ref_name=main")[0].(map[string]any)
	}
	propose := func(title, branch string, extra map[string]any) string {
		body := map[string]any{"title": title, "source_branch": branch, "target_branch": "main"}
		for k, v := range extra {
			body[k] = v
		}
		mr := bob.must("POST", mrs, body, 201)
		path := mrs + "/" + jsonNum(mr["iid"])
		ada.must("POST", path+"/approve", nil, 200)
		return path
	}

	// fast-forward: a branch atrás da main é posta em dia; nenhum commit de mescla
	commit("ff", "origin/main", "um.txt")
	ada.must("PUT", "/api/v4/projects/"+pid+"/repository/files/prefacio.txt", map[string]any{"branch": "main", "content": "prefácio\n", "commit_message": "prefácio"}, 200)
	if got := ada.must("PUT", "/api/v4/projects/"+pid, map[string]any{"merge_method": "ff"}, 200); got["merge_method"] != "ff" {
		t.Fatalf("merge_method: %v", got["merge_method"])
	}
	bob.must("PUT", "/api/v4/projects/"+pid, map[string]any{"merge_method": "merge"}, 403)
	before := head()["id"]
	mr1 := propose("Um", "ff", nil)
	if m := ada.must("POST", mr1+"/merge", nil, 200); m["state"] != "merged" {
		t.Fatalf("merge ff: %v", m)
	}
	h := head()
	if len(h["parent_ids"].([]any)) != 1 || h["title"] != "add um.txt" || h["parent_ids"].([]any)[0] != before {
		t.Fatalf("histórico linear: %v", h)
	}

	// squash pedido na criação: um commit com o título do merge request
	ada.must("PUT", "/api/v4/projects/"+pid, map[string]any{"merge_method": "merge"}, 200)
	run(t, work, "git", "fetch", "--quiet", "origin")
	commit("sq", "origin/main", "dois.txt")
	commit("sq", "sq", "tres.txt")
	mr2 := propose("Dois e três", "sq", map[string]any{"squash": true})
	if got := ada.must("GET", mr2, nil, 200); got["squash"] != true {
		t.Fatalf("squash no merge request: %v", got)
	}
	ada.must("POST", mr2+"/merge", nil, 200)
	if h := head(); len(h["parent_ids"].([]any)) != 1 || h["title"] != "Dois e três" {
		t.Fatalf("squash: %v", h)
	}

	// merge when pipeline succeeds: espera o pipeline da origem e mescla como quem pediu
	run(t, work, "git", "fetch", "--quiet", "origin")
	run(t, work, "git", "checkout", "--quiet", "-B", "ci", "origin/main")
	os.WriteFile(filepath.Join(work, ".gitlab-ci.yml"), []byte("test:\n  script:\n    - sleep 1\n    - echo ok\n"), 0o644)
	run(t, work, "git", "add", ".")
	run(t, work, "git", "commit", "--quiet", "-m", "ci")
	run(t, work, "git", "push", "--quiet", "origin", "ci")
	mr3 := propose("CI", "ci", nil)
	s := ada.must("POST", mr3+"/merge", map[string]any{"merge_when_pipeline_succeeds": true}, 200)
	if s["state"] != "opened" || s["merge_when_pipeline_succeeds"] != true {
		t.Fatalf("merge agendado: %v", s)
	}
	if m := waitState(t, ada, mr3, "merged", "closed"); m["state"] != "merged" || m["merge_when_pipeline_succeeds"] != false {
		t.Fatalf("depois do pipeline: %v", m)
	}

	// o agendamento com mínimo de aprovações: runtime TestMesclarQuandoPassarComAprovacoes
}
