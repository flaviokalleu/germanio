package e2e

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Fluxo 1: register → login → create project → open project.
func TestFluxo1RegistroLoginProjeto(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	me := ada.must("GET", "/api/v4/user", nil, 200)
	if me["username"] != "ada" || me["email"] != "ada@example.com" || me["senha"] != nil {
		t.Fatalf("usuário atual: %v", me)
	}
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Compiler", "path": "compiler", "visibility": "private"}, 201)
	if p["full_path"] != "ada/compiler" || p["visibility"] != "private" || p["default_branch"] != "main" {
		t.Fatalf("projeto: %v", p)
	}
	got := ada.must("GET", "/api/v4/projects/"+id(p), nil, 200)
	byPath := ada.must("GET", "/api/v4/projects/"+url.PathEscape("ada/compiler"), nil, 200)
	if got["name"] != "Compiler" || byPath["id"] != p["id"] {
		t.Fatalf("abrir projeto por id/caminho: %v %v", got, byPath)
	}

	// Negativos
	anon := &api{t: t, base: base}
	anon.must("POST", "/api/v4/projects", map[string]any{"name": "x", "path": "x"}, 401)
	anon.must("GET", "/api/v4/projects/"+id(p), nil, 404)
	bob := signup(t, base, "bob")
	bob.must("GET", "/api/v4/projects/"+id(p), nil, 404) // private: existence is not revealed
	bad := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Bad", "path": "-bad.git"}, 400)
	if m, _ := bad["message"].(map[string]any); m == nil || m["path"] == nil {
		t.Fatalf("path inválido: %v", bad)
	}
	ada.must("POST", "/api/v4/projects", map[string]any{"name": "Compiler 2", "path": "compiler"}, 400)
	(&api{t: t, base: base, token: "glpat-invalid", pat: true}).must("GET", "/api/v4/user", nil, 401)
	anon.must("POST", "/oauth/token", map[string]any{"grant_type": "password", "username": "ada", "password": "wrong"}, 400)
	// Public listing shows only what each person may see
	for _, it := range anon.list("/api/v4/projects") {
		if it.(map[string]any)["full_path"] == "ada/compiler" {
			t.Fatal("projeto privado listado para anônimo")
		}
	}
}

// Fluxo 2: create project → clone → commit → push → view commit.
func TestFluxo2CloneCommitPush(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	pat := ada.must("POST", "/api/v4/personal_access_tokens", map[string]any{"name": "cli", "scopes": "api"}, 201)
	secret, _ := pat["token"].(string)
	if !strings.HasPrefix(secret, "glpat-") {
		t.Fatalf("token deve ser mostrado uma vez na criação: %v", pat)
	}
	for _, it := range ada.list("/api/v4/personal_access_tokens") {
		if it.(map[string]any)["token"] != nil {
			t.Fatal("segredo do token reapareceu na listagem")
		}
	}
	patAPI := &api{t: t, base: base, token: secret, pat: true}
	if me := patAPI.must("GET", "/api/v4/user", nil, 200); me["username"] != "ada" {
		t.Fatalf("PRIVATE-TOKEN: %v", me)
	}
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Engine", "path": "engine", "initialize_with_readme": true}, 201)

	dir := t.TempDir()
	u, _ := url.Parse(base)
	u.User = url.UserPassword("ada", secret)
	remote := u.String() + "/ada/engine.git"
	run(t, dir, "git", "clone", "--quiet", remote, "engine")
	work := filepath.Join(dir, "engine")
	if _, err := os.Stat(filepath.Join(work, "README.md")); err != nil {
		t.Fatalf("README do initialize_with_readme ausente: %v", err)
	}
	run(t, work, "git", "config", "user.email", "ada@example.com")
	run(t, work, "git", "config", "user.name", "Ada")
	os.WriteFile(filepath.Join(work, "main.go"), []byte("package main\n"), 0o644)
	run(t, work, "git", "add", ".")
	run(t, work, "git", "commit", "--quiet", "-m", "Add main.go")
	run(t, work, "git", "push", "--quiet", "origin", "main")
	sha := strings.TrimSpace(run(t, work, "git", "rev-parse", "HEAD"))

	c := ada.must("GET", "/api/v4/projects/"+id(p)+"/repository/commits/"+sha, nil, 200)
	if c["title"] != "Add main.go" || c["author_email"] != "ada@example.com" {
		t.Fatalf("ver commit: %v", c)
	}
	diff := ada.list("/api/v4/projects/" + id(p) + "/repository/commits/" + sha + "/diff")
	if len(diff) != 1 || diff[0].(map[string]any)["new_path"] != "main.go" {
		t.Fatalf("diff do commit: %v", diff)
	}
	tree := ada.list("/api/v4/projects/" + id(p) + "/repository/tree")
	if len(tree) != 2 {
		t.Fatalf("árvore: %v", tree)
	}

	// Negativos: outro usuário não clona projeto privado; token inválido é recusado.
	bob := signup(t, base, "bob")
	_ = bob
	u.User = url.UserPassword("bob", "password123")
	runFails(t, dir, "git", "clone", "--quiet", u.String()+"/ada/engine.git", "stolen")
	u.User = url.UserPassword("ada", "glpat-wrong")
	runFails(t, dir, "git", "clone", "--quiet", u.String()+"/ada/engine.git", "wrong")
	ada.must("GET", "/api/v4/projects/"+id(p)+"/repository/commits/deadbeef", nil, 404)
}

// Escopos de token: read_api só lê; read_repository só clona.
func TestEscoposDeToken(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	ada.must("POST", "/api/v4/projects", map[string]any{"name": "Scoped", "path": "scoped", "initialize_with_readme": true}, 201)
	ro := ada.must("POST", "/api/v4/personal_access_tokens", map[string]any{"name": "ro", "scopes": "read_api"}, 201)
	roAPI := &api{t: t, base: base, token: ro["token"].(string), pat: true}
	roAPI.must("GET", "/api/v4/projects/1", nil, 200)
	roAPI.must("POST", "/api/v4/projects", map[string]any{"name": "x", "path": "x"}, 403)
	u, _ := url.Parse(base)
	u.User = url.UserPassword("ada", ro["token"].(string))
	runFails(t, t.TempDir(), "git", "clone", "--quiet", u.String()+"/ada/scoped.git", "x")
	repo := ada.must("POST", "/api/v4/personal_access_tokens", map[string]any{"name": "repo", "scopes": "read_repository"}, 201)
	u.User = url.UserPassword("ada", repo["token"].(string))
	run(t, t.TempDir(), "git", "clone", "--quiet", u.String()+"/ada/scoped.git", "x")
	(&api{t: t, base: base, token: repo["token"].(string), pat: true}).must("GET", "/api/v4/projects/1", nil, 403)
	ada.must("POST", "/api/v4/personal_access_tokens", map[string]any{"name": "bad", "scopes": "sudo"}, 400)
}

// Projeto arquivado é somente leitura: nada muda nele nem no que pertence a
// ele, e o repositório não recebe código, até desarquivar.
func TestProjetoArquivado(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Old", "path": "old", "initialize_with_readme": true}, 201)
	pid := id(p)
	ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "antes"}, 201)
	if got := ada.must("PUT", "/api/v4/projects/"+pid, map[string]any{"archived": true}, 200); got["archived"] != true {
		t.Fatalf("arquivar: %v", got)
	}
	ada.must("PUT", "/api/v4/projects/"+pid, map[string]any{"name": "Novo nome"}, 403)
	ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "depois"}, 403)
	ada.must("PUT", "/api/v4/projects/"+pid+"/issues/1", map[string]any{"title": "x"}, 403)
	ada.must("POST", "/api/v4/projects/"+pid+"/issues/1/close", nil, 403)
	ada.must("GET", "/api/v4/projects/"+pid+"/issues/1", nil, 200) // ler continua valendo

	pat := ada.must("POST", "/api/v4/personal_access_tokens", map[string]any{"name": "git"}, 201)
	u, _ := url.Parse(base)
	u.User = url.UserPassword("ada", pat["token"].(string))
	dir := t.TempDir()
	run(t, dir, "git", "clone", "--quiet", u.String()+"/ada/old.git", "w")
	work := filepath.Join(dir, "w")
	run(t, work, "git", "config", "user.email", "ada@example.com")
	run(t, work, "git", "config", "user.name", "Ada")
	os.WriteFile(filepath.Join(work, "x.txt"), []byte("x"), 0o644)
	run(t, work, "git", "add", ".")
	run(t, work, "git", "commit", "--quiet", "-m", "x")
	runFails(t, work, "git", "push", "--quiet", "origin", "main")

	ada.must("PUT", "/api/v4/projects/"+pid, map[string]any{"archived": false}, 200)
	ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "de novo"}, 201)
	run(t, work, "git", "push", "--quiet", "origin", "main")
}
