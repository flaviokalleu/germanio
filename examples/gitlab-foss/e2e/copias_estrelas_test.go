package e2e

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

// PR-05: fork — quem pode ver o projeto (e baixar o código) faz uma cópia no
// próprio espaço ou num grupo onde pode criar projetos; a cópia tem o próprio
// repositório, lembra a origem, tem quem copiou como owner e nunca é mais
// visível que o original (GEP 0029).
func TestFork(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	bob := signup(t, base, "bob")
	eve := signup(t, base, "eve")
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Engine", "path": "engine", "visibility": "internal", "description": "motor", "initialize_with_readme": true}, 201)
	pid := id(p)

	// ada pushes a branch so the fork has more than the README
	pat := ada.must("POST", "/api/v4/personal_access_tokens", map[string]any{"name": "git"}, 201)
	u, _ := url.Parse(base)
	u.User = url.UserPassword("ada", pat["token"].(string))
	dir := t.TempDir()
	run(t, dir, "git", "clone", "--quiet", u.String()+"/ada/engine.git", "engine")
	work := filepath.Join(dir, "engine")
	os.WriteFile(filepath.Join(work, "motor.txt"), []byte("v8\n"), 0o644)
	run(t, work, "git", "checkout", "--quiet", "-b", "feature")
	run(t, work, "git", "add", ".")
	run(t, work, "git", "-c", "user.name=Ada", "-c", "user.email=ada@example.com", "commit", "--quiet", "-m", "motor")
	run(t, work, "git", "push", "--quiet", "origin", "feature")

	// bob is not a member, but the project is internal: he may fork it
	f := bob.must("POST", "/api/v4/projects/"+pid+"/fork", nil, 201)
	fid := id(f)
	if f["full_path"] != "bob/engine" || f["description"] != "motor" || f["visibility"] != "internal" {
		t.Fatalf("fork: %v", f)
	}
	origin, _ := f["forked_from_project"].(map[string]any)
	if origin == nil || id(origin) != pid || origin["full_path"] != "ada/engine" {
		t.Fatalf("a origem do fork não aparece: %v", f["forked_from_project"])
	}
	if got := bob.must("GET", "/api/v4/projects/"+fid, nil, 200); id(got["forked_from_project"].(map[string]any)) != pid {
		t.Fatalf("ver o fork: %v", got)
	}
	// whoever forks becomes owner (50) of the fork
	members := bob.list("/api/v4/projects/" + fid + "/members")
	if len(members) != 1 || jsonNum(members[0].(map[string]any)["access_level"]) != "50" {
		t.Fatalf("membros do fork: %v", members)
	}
	// the fork's repository is a copy: clone it, with the branch, and push to it
	bpat := bob.must("POST", "/api/v4/personal_access_tokens", map[string]any{"name": "git"}, 201)
	bu, _ := url.Parse(base)
	bu.User = url.UserPassword("bob", bpat["token"].(string))
	run(t, dir, "git", "clone", "--quiet", "--branch", "feature", bu.String()+"/bob/engine.git", "fork")
	fork := filepath.Join(dir, "fork")
	if b, err := os.ReadFile(filepath.Join(fork, "motor.txt")); err != nil || string(b) != "v8\n" {
		t.Fatalf("o clone do fork não tem o código da origem: %q %v", b, err)
	}
	os.WriteFile(filepath.Join(fork, "motor.txt"), []byte("v12\n"), 0o644)
	run(t, fork, "git", "-c", "user.name=Bob", "-c", "user.email=bob@example.com", "commit", "--quiet", "-am", "mais cilindros")
	run(t, fork, "git", "push", "--quiet", "origin", "feature")
	last := func(who *api, project string) any {
		list := who.list("/api/v4/projects/" + project + "/repository/commits?ref_name=feature")
		if len(list) == 0 {
			t.Fatalf("sem commits em %s", project)
		}
		return list[0].(map[string]any)["title"]
	}
	if got := last(bob, fid); got != "mais cilindros" {
		t.Fatalf("o push no fork não chegou: %v", got)
	}
	if got := last(ada, pid); got != "motor" {
		t.Fatalf("o push no fork mudou a origem: %v", got)
	}

	// the same path in the same place is taken; another path works
	bob.must("POST", "/api/v4/projects/"+pid+"/fork", nil, 400)
	bob.must("POST", "/api/v4/projects/"+pid+"/fork", map[string]any{"path": "engine-2", "name": "Engine 2"}, 201)

	// never more visible than the original
	bob.must("POST", "/api/v4/projects/"+pid+"/fork", map[string]any{"path": "engine-pub", "visibility": "public"}, 400)
	bob.must("PUT", "/api/v4/projects/"+fid, map[string]any{"visibility": "public"}, 400)
	low := bob.must("POST", "/api/v4/projects/"+pid+"/fork", map[string]any{"path": "engine-priv", "visibility": "private"}, 201)
	if low["visibility"] != "private" {
		t.Fatalf("fork privado: %v", low)
	}

	// into a group: only where the person may create projects; the fork is
	// lowered to the group's visibility
	g := bob.must("POST", "/api/v4/groups", map[string]any{"name": "Oficina", "path": "oficina", "visibility": "private"}, 201)
	gf := bob.must("POST", "/api/v4/projects/"+pid+"/fork", map[string]any{"namespace_id": g["id"]}, 201)
	if gf["full_path"] != "oficina/engine" || gf["visibility"] != "private" {
		t.Fatalf("fork no grupo: %v", gf)
	}
	eveG := eve.must("POST", "/api/v4/groups", map[string]any{"name": "Eve", "path": "eve-group"}, 201)
	if code, _, _ := bob.call("POST", "/api/v4/projects/"+pid+"/fork", map[string]any{"namespace_id": eveG["id"]}); code != 404 && code != 403 {
		t.Fatalf("fork num grupo de outra pessoa: %d", code)
	}

	// a private project cannot be forked by someone who is not a member
	priv := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Secret", "path": "secret", "initialize_with_readme": true}, 201)
	if code, _, _ := eve.call("POST", "/api/v4/projects/"+id(priv)+"/fork", nil); code != 404 {
		t.Fatalf("fork de projeto privado por quem não é membro: %d", code)
	}
	// a guest of a private project sees it but cannot download the code: no fork
	ada.must("POST", "/api/v4/projects/"+id(priv)+"/members", map[string]any{"user_id": eve.must("GET", "/api/v4/user", nil, 200)["id"], "access_level": 10}, 201)
	if code, _, _ := eve.call("POST", "/api/v4/projects/"+id(priv)+"/fork", nil); code != 403 {
		t.Fatalf("fork por guest de projeto privado: %d", code)
	}
	// a reporter may
	bm := ada.must("POST", "/api/v4/projects/"+id(priv)+"/members", map[string]any{"user_id": bob.must("GET", "/api/v4/user", nil, 200)["id"], "access_level": 20}, 201)
	pf := bob.must("POST", "/api/v4/projects/"+id(priv)+"/fork", nil, 201)
	// whoever cannot see the original does not learn it from the fork
	bob.must("PUT", "/api/v4/projects/"+id(pf), map[string]any{"visibility": "private"}, 200)
	ada.must("DELETE", "/api/v4/projects/"+id(priv)+"/members/"+id(bm), nil, 204)
	if got := bob.must("GET", "/api/v4/projects/"+id(pf), nil, 200); got["forked_from_project"] != nil || got["forked_from_project_id"] != nil {
		t.Fatalf("o fork mostra uma origem que bob não vê mais: %v", got)
	}

	// deleting the original keeps the fork, without the link
	ada.must("DELETE", "/api/v4/projects/"+pid, nil, 204)
	if got := bob.must("GET", "/api/v4/projects/"+fid, nil, 200); got["forked_from_project"] != nil {
		t.Fatalf("o fork ainda aponta para a origem excluída: %v", got)
	}
	run(t, t.TempDir(), "git", "clone", "--quiet", bu.String()+"/bob/engine.git", "ainda")
}

// PR-06: estrelas — cada pessoa que vê um projeto marca uma vez e desmarca;
// o projeto mostra quantas estrelas tem; cada pessoa lista as que marcou
// (GEP 0030).
func TestEstrelas(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	bob := signup(t, base, "bob")
	eve := signup(t, base, "eve")
	pub := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Pub", "path": "pub", "visibility": "public"}, 201)
	priv := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Priv", "path": "priv"}, 201)
	ada.must("POST", "/api/v4/projects", map[string]any{"name": "Outro", "path": "outro", "visibility": "public"}, 201)
	if jsonNum(pub["star_count"]) != "0" {
		t.Fatalf("projeto novo começa sem estrelas: %v", pub["star_count"])
	}
	got := bob.must("POST", "/api/v4/projects/"+id(pub)+"/star", nil, 200)
	if jsonNum(got["star_count"]) != "1" {
		t.Fatalf("depois de uma estrela: %v", got["star_count"])
	}
	// the same person again: nothing changes
	if code, _, _ := bob.call("POST", "/api/v4/projects/"+id(pub)+"/star", nil); code != 304 {
		t.Fatalf("estrela repetida: %d", code)
	}
	ada.must("POST", "/api/v4/projects/"+id(pub)+"/star", nil, 200)
	if got := eve.must("GET", "/api/v4/projects/"+id(pub), nil, 200); jsonNum(got["star_count"]) != "2" {
		t.Fatalf("duas pessoas, duas estrelas: %v", got["star_count"])
	}
	// a project nobody may see cannot be starred by them
	if code, _, _ := eve.call("POST", "/api/v4/projects/"+id(priv)+"/star", nil); code != 404 {
		t.Fatalf("estrela em projeto privado de outra pessoa: %d", code)
	}
	anon := &api{t: t, base: base}
	if code, _, _ := anon.call("POST", "/api/v4/projects/"+id(pub)+"/star", nil); code != 401 {
		t.Fatalf("estrela sem entrar: %d", code)
	}
	// each person lists what they starred
	ada.must("POST", "/api/v4/projects/"+id(priv)+"/star", nil, 200)
	paths := func(who *api) []string {
		var out []string
		for _, it := range who.list("/api/v4/projects?starred=true") {
			out = append(out, it.(map[string]any)["path"].(string))
		}
		return out
	}
	if got := paths(ada); len(got) != 2 || got[0] != "priv" || got[1] != "pub" {
		t.Fatalf("estrelas de ada: %v", got)
	}
	if got := paths(bob); len(got) != 1 || got[0] != "pub" {
		t.Fatalf("estrelas de bob: %v", got)
	}
	if got := paths(eve); len(got) != 0 {
		t.Fatalf("eve não marcou nada: %v", got)
	}
	// unstar
	if got := bob.must("POST", "/api/v4/projects/"+id(pub)+"/unstar", nil, 200); jsonNum(got["star_count"]) != "1" {
		t.Fatalf("depois de desmarcar: %v", got["star_count"])
	}
	if code, _, _ := bob.call("POST", "/api/v4/projects/"+id(pub)+"/unstar", nil); code != 304 {
		t.Fatalf("desmarcar o que não marcou: %d", code)
	}
	// an archived project may still be starred: the mark is the person's
	ada.must("PUT", "/api/v4/projects/"+id(pub), map[string]any{"archived": true}, 200)
	bob.must("POST", "/api/v4/projects/"+id(pub)+"/star", nil, 200)
	// nobody sets the count by hand
	ada.must("PUT", "/api/v4/projects/"+id(priv), map[string]any{"star_count": 99}, 200)
	if got := ada.must("GET", "/api/v4/projects/"+id(priv), nil, 200); jsonNum(got["star_count"]) != "1" {
		t.Fatalf("a contagem mudou por edição: %v", got["star_count"])
	}
	// a deleted person takes the stars away
	root := &api{t: t, base: base}
	tok := root.must("POST", "/oauth/token", map[string]any{"grant_type": "password", "username": "root", "password": "rootpassword1"}, 200)
	root.token = tok["access_token"].(string)
	root.must("DELETE", "/api/v4/users/"+id(bob.must("GET", "/api/v4/user", nil, 200)), nil, 204)
	if got := ada.must("GET", "/api/v4/projects/"+id(pub), nil, 200); jsonNum(got["star_count"]) != "1" {
		t.Fatalf("estrelas depois de excluir quem marcou: %v", got["star_count"])
	}
}

// PR-06: tópicos — uma lista de textos no projeto, filtrada por um tópico.
func TestTopicos(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	eve := signup(t, base, "eve")
	a := ada.must("POST", "/api/v4/projects", map[string]any{"name": "A", "path": "a", "visibility": "public", "topics": []any{"go", "compiladores"}}, 201)
	if tp, _ := a["topics"].([]any); len(tp) != 2 || tp[0] != "go" {
		t.Fatalf("tópicos: %v", a["topics"])
	}
	ada.must("POST", "/api/v4/projects", map[string]any{"name": "B", "path": "b", "visibility": "public", "topics": []any{"rust"}}, 201)
	ada.must("POST", "/api/v4/projects", map[string]any{"name": "C", "path": "c", "topics": []any{"go"}}, 201)
	ada.must("POST", "/api/v4/projects", map[string]any{"name": "D", "path": "d", "visibility": "public", "topics": []any{"golang"}}, 201)
	paths := func(who *api, q string) []string {
		var out []string
		for _, it := range who.list("/api/v4/projects?topic=" + q) {
			out = append(out, it.(map[string]any)["path"].(string))
		}
		return out
	}
	if got := paths(ada, "go"); len(got) != 2 || got[0] != "c" || got[1] != "a" {
		t.Fatalf("projetos com go (sem golang): %v", got)
	}
	// the filter never shows more than the person sees
	if got := paths(eve, "go"); len(got) != 1 || got[0] != "a" {
		t.Fatalf("eve vê só o público: %v", got)
	}
	if got := paths(ada, "haskell"); len(got) != 0 {
		t.Fatalf("tópico sem projetos: %v", got)
	}
	ada.must("PUT", "/api/v4/projects/"+id(a), map[string]any{"topics": []any{"rust"}}, 200)
	if got := paths(ada, "rust"); len(got) != 2 {
		t.Fatalf("depois de editar os tópicos: %v", got)
	}
}

// PR-06: avatar — uma imagem do projeto (GEP 0014): enviada por quem edita,
// baixada por quem vê, com o endereço em avatar_url.
func TestAvatar(t *testing.T) {
	t.Setenv("GERMANIO_ARQUIVOS", t.TempDir())
	base := gitlab(t)
	ada := signup(t, base, "ada")
	bob := signup(t, base, "bob")
	eve := signup(t, base, "eve")
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Pic", "path": "pic", "visibility": "internal"}, 201)
	if p["avatar_url"] != nil {
		t.Fatalf("projeto novo sem avatar: %v", p["avatar_url"])
	}
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 64)...)
	send := func(who *api, method, path string, body []byte) (int, http.Header, []byte) {
		req, _ := http.NewRequest(method, base+path, bytes.NewReader(body))
		if who != nil {
			req.Header.Set("Authorization", "Bearer "+who.token)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, resp.Header, b
	}
	if code, _, out := send(ada, "PUT", "/api/v4/projects/"+id(p)+"/avatar?nome=logo.png", png); code != 200 {
		t.Fatalf("enviar avatar: %d %s", code, out)
	}
	got := bob.must("GET", "/api/v4/projects/"+id(p), nil, 200)
	href, _ := got["avatar_url"].(string)
	if href != "/api/v4/projects/"+id(p)+"/avatar" {
		t.Fatalf("avatar_url: %v", got)
	}
	if meta, _ := got["avatar"].(map[string]any); meta == nil || meta["name"] != "logo.png" || meta["tipo"] != "image/png" {
		t.Fatalf("avatar: %v", got["avatar"])
	}
	// listed projects show it too
	for _, it := range bob.list("/api/v4/projects") {
		if m := it.(map[string]any); id(m) == id(p) && m["avatar_url"] != href {
			t.Fatalf("lista sem avatar_url: %v", m)
		}
	}
	code, h, body := send(bob, "GET", href, nil)
	if code != 200 || h.Get("Content-Type") != "image/png" || !bytes.Equal(body, png) {
		t.Fatalf("baixar avatar: %d %s", code, h.Get("Content-Type"))
	}
	// someone who only sees the project cannot change it; outside, nobody downloads a private one
	if code, _, _ := send(bob, "PUT", "/api/v4/projects/"+id(p)+"/avatar?nome=x.png", png); code != 403 {
		t.Fatalf("trocar avatar sem poder editar: %d", code)
	}
	ada.must("PUT", "/api/v4/projects/"+id(p), map[string]any{"visibility": "private"}, 200)
	if code, _, _ := send(eve, "GET", href, nil); code != 404 {
		t.Fatalf("avatar de projeto privado para quem não é membro: %d", code)
	}
	// the avatar is never set as text
	ada.must("PUT", "/api/v4/projects/"+id(p), map[string]any{"avatar": "http://x/y.png"}, 400)
	if code, _, _ := send(ada, "DELETE", "/api/v4/projects/"+id(p)+"/avatar", nil); code != 204 && code != 200 {
		t.Fatalf("remover avatar: %d", code)
	}
	if got := ada.must("GET", "/api/v4/projects/"+id(p), nil, 200); got["avatar_url"] != nil {
		t.Fatalf("avatar removido continua: %v", got["avatar_url"])
	}
}
