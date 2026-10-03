package e2e

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Fluxo 4: branch → push → merge request → diff → review → merge.
func TestFluxo4MergeRequest(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada") // owner
	bob := signup(t, base, "bob") // developer
	eve := signup(t, base, "eve") // sem acesso
	bobID := bob.must("GET", "/api/v4/user", nil, 200)["id"]
	adaID := ada.must("GET", "/api/v4/user", nil, 200)["id"]

	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Core", "path": "core", "initialize_with_readme": true}, 201)
	pid := id(p)
	ada.must("POST", "/api/v4/projects/"+pid+"/members", map[string]any{"user_id": bobID, "access_level": 30}, 201)

	// bob cria uma branch e envia um commit
	pat := bob.must("POST", "/api/v4/personal_access_tokens", map[string]any{"name": "git"}, 201)
	u, _ := url.Parse(base)
	u.User = url.UserPassword("bob", pat["token"].(string))
	dir := t.TempDir()
	run(t, dir, "git", "clone", "--quiet", u.String()+"/ada/core.git", "core")
	work := filepath.Join(dir, "core")
	run(t, work, "git", "config", "user.email", "bob@example.com")
	run(t, work, "git", "config", "user.name", "Bob")
	run(t, work, "git", "checkout", "--quiet", "-b", "feature")
	os.WriteFile(filepath.Join(work, "feature.txt"), []byte("new feature\n"), 0o644)
	run(t, work, "git", "add", ".")
	run(t, work, "git", "commit", "--quiet", "-m", "Add feature")
	run(t, work, "git", "push", "--quiet", "origin", "feature")

	// validação: branches precisam existir e ser diferentes
	bad := bob.must("POST", "/api/v4/projects/"+pid+"/merge_requests", map[string]any{"title": "x", "source_branch": "nope", "target_branch": "main"}, 400)
	if m, _ := bad["message"].(map[string]any); m["source_branch"] == nil {
		t.Fatalf("branch inexistente: %v", bad)
	}
	bob.must("POST", "/api/v4/projects/"+pid+"/merge_requests", map[string]any{"title": "x", "source_branch": "main", "target_branch": "main"}, 400)
	eve.must("POST", "/api/v4/projects/"+pid+"/merge_requests", map[string]any{"title": "x", "source_branch": "feature", "target_branch": "main"}, 404)

	mr := bob.must("POST", "/api/v4/projects/"+pid+"/merge_requests", map[string]any{"title": "Add feature", "source_branch": "feature", "target_branch": "main", "reviewer_ids": []any{adaID}}, 201)
	if mr["iid"].(float64) != 1 || mr["state"] != "opened" || mr["author_id"] != bobID {
		t.Fatalf("merge request: %v", mr)
	}
	mrPath := "/api/v4/projects/" + pid + "/merge_requests/1"
	got := ada.must("GET", mrPath, nil, 200)
	if got["can_be_merged"] != true {
		t.Fatalf("deveria poder mesclar: %v", got)
	}
	ch := ada.must("GET", mrPath+"/changes", nil, 200)
	changes, _ := ch["changes"].([]any)
	if len(changes) != 1 || changes[0].(map[string]any)["new_path"] != "feature.txt" || !strings.Contains(changes[0].(map[string]any)["diff"].(string), "+new feature") {
		t.Fatalf("mudanças: %v", ch)
	}
	if commits := ada.list(mrPath + "/commits"); len(commits) != 1 || commits[0].(map[string]any)["title"] != "Add feature" {
		t.Fatalf("commits do MR: %v", commits)
	}
	// regra de aprovação (GEP 0026): uma aprovação de quem não é o autor
	if got["approvals_required"] != float64(1) || got["approvals_left"] != float64(1) {
		t.Fatalf("aprovações necessárias: %v", got)
	}
	bob.must("POST", mrPath+"/approve", nil, 200) // o autor aprova, mas não conta
	ada.must("POST", mrPath+"/merge", nil, 405)
	bob.must("POST", mrPath+"/unapprove", nil, 200)
	// revisão: comentário e aprovação
	ada.must("POST", mrPath+"/notes", map[string]any{"body": "LGTM"}, 201)
	appr := ada.must("POST", mrPath+"/approve", nil, 200)
	if l, _ := appr["approved_by_ids"].([]any); len(l) != 1 {
		t.Fatalf("aprovação: %v", appr)
	}
	eve.must("POST", mrPath+"/approve", nil, 404)
	// developer não mescla na branch padrão (protegida); owner mescla
	bob.must("POST", mrPath+"/merge", nil, 403)
	merged := ada.must("POST", mrPath+"/merge", nil, 200)
	if merged["state"] != "merged" || merged["merged_by_id"] != adaID || merged["merge_commit_sha"] == nil {
		t.Fatalf("merge: %v", merged)
	}
	files := ada.list("/api/v4/projects/" + pid + "/repository/tree")
	found := false
	for _, f := range files {
		found = found || f.(map[string]any)["name"] == "feature.txt"
	}
	if !found {
		t.Fatalf("arquivo não chegou à main: %v", files)
	}
	// estado final: não reabre nem mescla de novo
	ada.must("POST", mrPath+"/reopen", nil, 405)
	ada.must("POST", mrPath+"/merge", nil, 405)

	// conflito
	run(t, work, "git", "checkout", "--quiet", "-b", "conflict", "origin/main")
	os.WriteFile(filepath.Join(work, "README.md"), []byte("bob's readme\n"), 0o644)
	run(t, work, "git", "commit", "--quiet", "-am", "bob readme")
	run(t, work, "git", "push", "--quiet", "origin", "conflict")
	uAda, _ := url.Parse(base)
	adaPat := ada.must("POST", "/api/v4/personal_access_tokens", map[string]any{"name": "git"}, 201)
	uAda.User = url.UserPassword("ada", adaPat["token"].(string))
	dir2 := t.TempDir()
	run(t, dir2, "git", "clone", "--quiet", uAda.String()+"/ada/core.git", "core")
	w2 := filepath.Join(dir2, "core")
	run(t, w2, "git", "config", "user.email", "ada@example.com")
	run(t, w2, "git", "config", "user.name", "Ada")
	os.WriteFile(filepath.Join(w2, "README.md"), []byte("ada's readme\n"), 0o644)
	run(t, w2, "git", "commit", "--quiet", "-am", "ada readme")
	run(t, w2, "git", "push", "--quiet", "origin", "main")
	bob.must("POST", "/api/v4/projects/"+pid+"/merge_requests", map[string]any{"title": "Conflict", "source_branch": "conflict", "target_branch": "main"}, 201)
	c := ada.must("GET", "/api/v4/projects/"+pid+"/merge_requests/2", nil, 200)
	if c["can_be_merged"] != false || len(c["conflicts"].([]any)) != 1 {
		t.Fatalf("conflito não detectado: %v", c)
	}
	ada.must("POST", "/api/v4/projects/"+pid+"/merge_requests/2/merge", nil, 406)
	// rascunho não é mesclado
	bob.must("POST", "/api/v4/projects/"+pid+"/merge_requests", map[string]any{"title": "WIP", "source_branch": "feature", "target_branch": "conflict", "draft": true}, 201)
	ada.must("POST", "/api/v4/projects/"+pid+"/merge_requests/3/merge", nil, 406)
}
