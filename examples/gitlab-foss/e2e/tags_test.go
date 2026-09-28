package e2e

import (
	"net/url"
	"path/filepath"
	"testing"
)

// RP-07: tags do repositório — enviadas por git push ou criadas pela API,
// com as mesmas regras do código.
func TestTags(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	eve := signup(t, base, "eve")
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Lib", "path": "lib", "initialize_with_readme": true}, 201)
	pid := id(p)
	pat := ada.must("POST", "/api/v4/personal_access_tokens", map[string]any{"name": "git"}, 201)
	u, _ := url.Parse(base)
	u.User = url.UserPassword("ada", pat["token"].(string))
	dir := t.TempDir()
	run(t, dir, "git", "clone", "--quiet", u.String()+"/ada/lib.git", "lib")
	work := filepath.Join(dir, "lib")
	run(t, work, "git", "tag", "v1.0")
	run(t, work, "git", "push", "--quiet", "origin", "v1.0")

	names := func(who *api) map[string]bool {
		out := map[string]bool{}
		for _, it := range who.list("/api/v4/projects/" + pid + "/repository/tags") {
			out[it.(map[string]any)["name"].(string)] = true
		}
		return out
	}
	if !names(ada)["v1.0"] {
		t.Fatalf("a tag enviada por git push não aparece: %v", names(ada))
	}
	tag := ada.must("POST", "/api/v4/projects/"+pid+"/repository/tags", map[string]any{"tag_name": "v1.1", "ref": "main"}, 201)
	if tag["name"] != "v1.1" || tag["target"] == nil {
		t.Fatalf("criar tag: %v", tag)
	}
	ada.must("POST", "/api/v4/projects/"+pid+"/repository/tags", map[string]any{"tag_name": "v1.1", "ref": "main"}, 400)
	ada.must("POST", "/api/v4/projects/"+pid+"/repository/tags", map[string]any{"tag_name": "../x", "ref": "main"}, 400)
	// private project: someone outside does not see nor create tags
	if code, _, _ := eve.call("GET", "/api/v4/projects/"+pid+"/repository/tags", nil); code != 404 {
		t.Fatalf("tags de projeto privado para quem não é membro: %d", code)
	}
	if code, _, _ := eve.call("POST", "/api/v4/projects/"+pid+"/repository/tags", map[string]any{"tag_name": "v9", "ref": "main"}); code != 404 {
		t.Fatalf("criar tag em projeto privado sem ser membro: %d", code)
	}
	ada.must("DELETE", "/api/v4/projects/"+pid+"/repository/tags/v1.1", nil, 204)
	if got := names(ada); got["v1.1"] || !got["v1.0"] {
		t.Fatalf("remover tag: %v", got)
	}
}
