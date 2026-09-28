package e2e

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGruposMembrosPapeis(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	bob := signup(t, base, "bob")
	eve := signup(t, base, "eve")
	bobID := id(bob.must("GET", "/api/v4/user", nil, 200))
	eveID := id(eve.must("GET", "/api/v4/user", nil, 200))

	g := ada.must("POST", "/api/v4/groups", map[string]any{"name": "Platform", "path": "platform", "visibility": "private"}, 201)
	if g["full_path"] != "platform" {
		t.Fatalf("grupo: %v", g)
	}
	// Quem cria vira owner
	ms := ada.list("/api/v4/groups/" + id(g) + "/members")
	if len(ms) != 1 || ms[0].(map[string]any)["access_level"] != float64(50) {
		t.Fatalf("owner inicial: %v", ms)
	}
	// Subgrupo herda o caminho e não pode ser mais visível que o pai
	ada.must("POST", "/api/v4/groups", map[string]any{"name": "Web", "path": "web", "parent_id": g["id"], "visibility": "public"}, 400)
	sub := ada.must("POST", "/api/v4/groups", map[string]any{"name": "Web", "path": "web", "parent_id": g["id"]}, 201)
	if sub["full_path"] != "platform/web" {
		t.Fatalf("subgrupo: %v", sub)
	}
	// Não-membro não vê grupo privado nem cria projeto nele
	bob.must("GET", "/api/v4/groups/"+id(g), nil, 404)
	bob.must("POST", "/api/v4/projects", map[string]any{"name": "X", "path": "x", "namespace_id": g["id"]}, 404)

	// Owner adiciona bob como developer (API no formato GitLab: user_id + access_level)
	ada.must("POST", "/api/v4/groups/"+id(g)+"/members", map[string]any{"user_id": bob.must("GET", "/api/v4/user", nil, 200)["id"], "access_level": 30}, 201)
	bob.must("GET", "/api/v4/groups/"+id(g), nil, 200)
	// Developer não adiciona membros
	bob.must("POST", "/api/v4/groups/"+id(g)+"/members", map[string]any{"user_id": eveID, "access_level": 10}, 403)
	// Developer cria projeto no grupo; o acesso ao projeto vem do grupo
	p := bob.must("POST", "/api/v4/projects", map[string]any{"name": "API", "path": "api", "namespace_id": g["id"], "initialize_with_readme": true}, 201)
	if p["full_path"] != "platform/api" {
		t.Fatalf("projeto do grupo: %v", p)
	}
	// Membro do subgrupo herda do grupo pai
	bob.must("GET", "/api/v4/groups/"+id(sub), nil, 200)

	// Developer envia para branch nova, mas não para a branch protegida
	pat := bob.must("POST", "/api/v4/personal_access_tokens", map[string]any{"name": "git"}, 201)
	dir := t.TempDir()
	u, _ := url.Parse(base)
	u.User = url.UserPassword("bob", pat["token"].(string))
	run(t, dir, "git", "clone", "--quiet", u.String()+"/platform/api.git", "api")
	work := filepath.Join(dir, "api")
	run(t, work, "git", "config", "user.email", "bob@example.com")
	run(t, work, "git", "config", "user.name", "Bob")
	os.WriteFile(filepath.Join(work, "x.txt"), []byte("x\n"), 0o644)
	run(t, work, "git", "add", ".")
	run(t, work, "git", "commit", "--quiet", "-m", "x")
	out := runFails(t, work, "git", "push", "origin", "main")
	if !strings.Contains(out, "protected branches") {
		t.Fatalf("mensagem de branch protegida ausente:\n%s", out)
	}
	run(t, work, "git", "push", "--quiet", "origin", "HEAD:feature")
	branches := bob.list("/api/v4/projects/" + id(p) + "/repository/branches")
	if len(branches) != 2 {
		t.Fatalf("branches: %v", branches)
	}

	// RP-06: branches protegidas por padrão (GEP 0016) — dados do projeto
	prot := ada.must("POST", "/api/v4/projects/"+id(p)+"/protected_branches", map[string]any{"name": "release/*"}, 201)
	bob.must("POST", "/api/v4/projects/"+id(p)+"/protected_branches", map[string]any{"name": "*"}, 403)
	out = runFails(t, work, "git", "push", "origin", "HEAD:release/1.0")
	if !strings.Contains(out, "protected branches") {
		t.Fatalf("push para branch protegida por padrão:\n%s", out)
	}
	bob.must("POST", "/api/v4/projects/"+id(p)+"/repository/branches", map[string]any{"branch": "release/2.0", "ref": "main"}, 403)
	run(t, work, "git", "push", "--quiet", "origin", "HEAD:hotfix")
	ada.must("DELETE", "/api/v4/projects/"+id(p)+"/protected_branches/"+id(prot), nil, 204)
	run(t, work, "git", "push", "--quiet", "origin", "HEAD:release/1.0")

	// Ninguém concede papel maior que o próprio; o último owner não sai
	bob.must("POST", "/api/v4/groups/"+id(g)+"/members", map[string]any{"user_id": eveID, "access_level": 50}, 403)
	ada.must("POST", "/api/v4/groups/"+id(g)+"/sair", nil, 400)
	var adaMember map[string]any
	for _, m := range ada.list("/api/v4/groups/" + id(g) + "/members") {
		if mm := m.(map[string]any); mm["access_level"] == float64(50) {
			adaMember = mm
		}
	}
	if adaMember == nil {
		t.Fatal("membro owner não encontrado")
	}
	ada.must("PUT", "/api/v4/groups/"+id(g)+"/members/"+id(adaMember), map[string]any{"access_level": 30}, 400) // rebaixar o último owner
	// Com outro owner, rebaixar passa a valer
	eveOwner := ada.must("POST", "/api/v4/groups/"+id(g)+"/members", map[string]any{"user_id": eveID, "access_level": 50}, 201)
	ada.must("PUT", "/api/v4/groups/"+id(g)+"/members/"+id(adaMember), map[string]any{"access_level": 40}, 200)
	ada.must("DELETE", "/api/v4/groups/"+id(g)+"/members/"+id(eveOwner), nil, 400) // Eve agora é a última owner
	// Membro pode sair
	bob.must("POST", "/api/v4/groups/"+id(g)+"/sair", nil, 204)
	bob.must("GET", "/api/v4/groups/"+id(g), nil, 404)
	_ = bobID
}
