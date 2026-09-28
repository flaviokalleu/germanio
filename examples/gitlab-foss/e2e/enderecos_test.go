package e2e

import (
	"net/url"
	"testing"
)

// Endereços: grupo/subgrupo/projeto, um só espaço de nomes com os usuários,
// e renomear um grupo leva junto tudo o que está dentro dele.
func TestEnderecosHierarquicos(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	signup(t, base, "bob")

	org := ada.must("POST", "/api/v4/groups", map[string]any{"name": "Org", "path": "org"}, 201)
	sub := ada.must("POST", "/api/v4/groups", map[string]any{"name": "Sub", "path": "sub", "parent_id": org["id"]}, 201)
	if org["full_path"] != "org" || sub["full_path"] != "org/sub" {
		t.Fatalf("endereços de grupos: %v %v", org["full_path"], sub["full_path"])
	}
	app := ada.must("POST", "/api/v4/projects", map[string]any{"name": "App", "path": "app", "namespace_id": sub["id"], "initialize_with_readme": true}, 201)
	mine := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Mine", "path": "mine"}, 201)
	if app["full_path"] != "org/sub/app" || mine["full_path"] != "ada/mine" {
		t.Fatalf("endereços de projetos: %v %v", app["full_path"], mine["full_path"])
	}

	// O repositório começou com o README declarado ({nome} preenchido)
	if f := ada.must("GET", "/api/v4/projects/"+id(app)+"/repository/files/README.md", nil, 200); f["content"] != "# App\n" {
		t.Fatalf("README inicial: %q", f["content"])
	}
	ada.must("GET", "/api/v4/projects/"+id(mine)+"/repository/files/README.md", nil, 404) // sem initialize_with_readme
	// Quem cria vira owner só do projeto pessoal; no grupo, os membros vêm do grupo.
	if ms := ada.list("/api/v4/projects/" + id(mine) + "/members"); len(ms) != 1 || ms[0].(map[string]any)["access_level"] != float64(50) {
		t.Fatalf("membros do projeto pessoal: %v", ms)
	}
	if ms := ada.list("/api/v4/projects/" + id(app) + "/members"); len(ms) != 0 {
		t.Fatalf("projeto de grupo não deveria ter membro direto: %v", ms)
	}
	// Um só espaço de nomes: grupo não usa nome de pessoa, nem repete endereço.
	ada.must("POST", "/api/v4/groups", map[string]any{"name": "Bob", "path": "bob"}, 400)
	ada.must("POST", "/api/v4/groups", map[string]any{"name": "Org2", "path": "org"}, 400)
	ada.must("POST", "/api/v4/groups", map[string]any{"name": "Sub", "path": "sub", "parent_id": org["id"]}, 400)
	ada.must("POST", "/api/v4/groups", map[string]any{"name": "Sub", "path": "sub"}, 201) // outro endereço: "sub"
	// O endereço é calculado, nunca aceito da entrada.
	x := ada.must("POST", "/api/v4/groups", map[string]any{"name": "X", "path": "x", "full_path": "org/hack"}, 201)
	if x["full_path"] != "x" {
		t.Fatalf("full_path aceito da entrada: %v", x["full_path"])
	}

	// Renomear o grupo leva subgrupos e projetos junto.
	ada.must("PUT", "/api/v4/groups/"+id(org), map[string]any{"path": "empresa"}, 200)
	if g := ada.must("GET", "/api/v4/groups/"+id(sub), nil, 200); g["full_path"] != "empresa/sub" {
		t.Fatalf("subgrupo depois de renomear: %v", g["full_path"])
	}
	if p := ada.must("GET", "/api/v4/projects/"+id(app), nil, 200); p["full_path"] != "empresa/sub/app" {
		t.Fatalf("projeto depois de renomear: %v", p["full_path"])
	}
	ada.must("GET", "/api/v4/projects/"+url.PathEscape("empresa/sub/app"), nil, 200)
	ada.must("GET", "/api/v4/projects/"+url.PathEscape("org/sub/app"), nil, 404)

	pat := ada.must("POST", "/api/v4/personal_access_tokens", map[string]any{"name": "git"}, 201)
	u, _ := url.Parse(base)
	u.User = url.UserPassword("ada", pat["token"].(string))
	run(t, t.TempDir(), "git", "clone", "--quiet", u.String()+"/empresa/sub/app.git", "x")
	runFails(t, t.TempDir(), "git", "clone", "--quiet", u.String()+"/org/sub/app.git", "x")

	// Renomear a pessoa leva seus projetos; o nome não pode ser um endereço em uso.
	ada.must("PUT", "/api/v4/users/"+id(ada.must("GET", "/api/v4/user", nil, 200)), map[string]any{"username": "empresa"}, 400)
	ada.must("PUT", "/api/v4/users/"+id(ada.must("GET", "/api/v4/user", nil, 200)), map[string]any{"username": "ada2"}, 200)
	if p := ada.must("GET", "/api/v4/projects/"+id(mine), nil, 200); p["full_path"] != "ada2/mine" {
		t.Fatalf("projeto pessoal depois de renomear a pessoa: %v", p["full_path"])
	}
}
