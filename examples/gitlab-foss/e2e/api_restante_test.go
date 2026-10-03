package e2e

import (
	"net/http"
	"net/http/cgi"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func paths(l []any) string {
	var out []string
	for _, it := range l {
		out = append(out, it.(map[string]any)["full_path"].(string))
	}
	return strings.Join(out, ",")
}

// PR-05: GET /projects/:id/forks lists the copies of a project (GEP 0029)
// that the person sees, of a project the person sees.
func TestForksDoProjeto(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	bob := signup(t, base, "bob")
	eve := signup(t, base, "eve")
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Engine", "path": "engine", "visibility": "public", "initialize_with_readme": true}, 201)
	pid := id(p)
	bob.must("POST", "/api/v4/projects/"+pid+"/fork", nil, 201)
	eve.must("POST", "/api/v4/projects/"+pid+"/fork", map[string]any{"visibility": "private"}, 201)
	ada.must("POST", "/api/v4/projects", map[string]any{"name": "Outro", "path": "outro", "visibility": "public"}, 201)

	// eve's fork is private: ada does not see it among the forks
	code, body, h := ada.call("GET", "/api/v4/projects/"+pid+"/forks", nil)
	if l, _ := body.([]any); code != 200 || paths(l) != "bob/engine" || h.Get("X-Total") != "1" {
		t.Fatalf("forks para a dona: %d %v %v", code, body, h)
	}
	if got := paths(eve.list("/api/v4/projects/" + pid + "/forks")); got != "eve/engine,bob/engine" {
		t.Fatalf("forks para eve: %s", got)
	}
	// by the project's address too, with the list's search
	if got := paths(ada.list("/api/v4/projects/ada%2Fengine/forks?search=engine")); got != "bob/engine" {
		t.Fatalf("forks pelo endereço: %s", got)
	}
	// a project nobody forked
	if got := ada.list("/api/v4/projects/" + pid + "/forks?per_page=1&page=2"); len(got) != 0 {
		t.Fatalf("página além do fim: %v", got)
	}

	// a private project: its forks do not exist for whoever does not see it
	priv := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Secret", "path": "secret", "initialize_with_readme": true}, 201)
	bobID := bob.must("GET", "/api/v4/user", nil, 200)["id"]
	ada.must("POST", "/api/v4/projects/"+id(priv)+"/members", map[string]any{"user_id": bobID, "access_level": 20}, 201)
	bf := bob.must("POST", "/api/v4/projects/"+id(priv)+"/fork", map[string]any{"path": "secret-fork"}, 201)
	bob.must("PUT", "/api/v4/projects/"+id(bf), map[string]any{"visibility": "private"}, 200)
	eve.must("GET", "/api/v4/projects/"+id(priv)+"/forks", nil, 404)
	if got := paths(ada.list("/api/v4/projects/" + id(priv) + "/forks")); got != "" {
		t.Fatalf("o fork privado de bob não aparece para ada: %s", got)
	}
	if got := paths(bob.list("/api/v4/projects/" + id(priv) + "/forks")); got != "bob/secret-fork" {
		t.Fatalf("bob vê o próprio fork: %s", got)
	}
	// filtering the list by an original one does not see says nothing
	bob.must("PUT", "/api/v4/projects/"+id(bf), map[string]any{"visibility": "private"}, 200)
	if l := eve.list("/api/v4/projects?forked_from_project_id=" + id(priv)); len(l) != 0 {
		t.Fatalf("filtro por um original invisível: %v", l)
	}
	(&api{t: t, base: base}).must("GET", "/api/v4/projects/"+id(priv)+"/forks", nil, 404)
}

// PR-05: forking (and creating) into a place named by its address
// (namespace_path): one's own space, a group one may create in; never
// another person's space nor a group one does not see (GEP 0044).
func TestForkNoEspacoPessoal(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	bob := signup(t, base, "bob")
	eve := signup(t, base, "eve")
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Engine", "path": "engine", "visibility": "public", "initialize_with_readme": true}, 201)
	pid := id(p)
	g := bob.must("POST", "/api/v4/groups", map[string]any{"name": "Oficina", "path": "oficina", "visibility": "public"}, 201)
	sub := bob.must("POST", "/api/v4/groups", map[string]any{"name": "Motores", "path": "motores", "parent_id": g["id"]}, 201)
	eve.must("POST", "/api/v4/groups", map[string]any{"name": "Segredo", "path": "segredo", "visibility": "private"}, 201)

	if f := bob.must("POST", "/api/v4/projects/"+pid+"/fork", map[string]any{"namespace_path": "bob"}, 201); f["full_path"] != "bob/engine" || f["namespace_id"] != nil {
		t.Fatalf("fork no espaço de bob: %v", f)
	}
	if f := bob.must("POST", "/api/v4/projects/"+pid+"/fork", map[string]any{"namespace_path": "oficina/motores"}, 201); f["full_path"] != "oficina/motores/engine" || jsonNum(f["namespace_id"]) != id(sub) {
		t.Fatalf("fork no subgrupo: %v", f)
	}
	// the address wins over the reference sent with it
	if f := bob.must("POST", "/api/v4/projects/"+pid+"/fork", map[string]any{"namespace_path": "oficina", "namespace_id": sub["id"]}, 201); f["full_path"] != "oficina/engine" {
		t.Fatalf("fork no grupo: %v", f)
	}
	// someone else's space, a group one does not see, an address nobody has
	bob.must("POST", "/api/v4/projects/"+pid+"/fork", map[string]any{"namespace_path": "ada", "path": "e2"}, 403)
	bob.must("POST", "/api/v4/projects/"+pid+"/fork", map[string]any{"namespace_path": "segredo", "path": "e3"}, 404)
	bob.must("POST", "/api/v4/projects/"+pid+"/fork", map[string]any{"namespace_path": "ninguem", "path": "e4"}, 404)
	// a group one sees but may not create in
	eve.must("POST", "/api/v4/projects/"+pid+"/fork", map[string]any{"namespace_path": "oficina", "path": "e5"}, 403)
	// creating a project names its place the same way
	if c := bob.must("POST", "/api/v4/projects", map[string]any{"name": "Caixa", "path": "caixa", "namespace_path": "oficina"}, 201); c["full_path"] != "oficina/caixa" {
		t.Fatalf("projeto criado pelo endereço do grupo: %v", c)
	}
	// not even an administrator creates in someone else's space: the copy
	// would be owned by the administrator, not by the person
	anon := &api{t: t, base: base}
	tok := anon.must("POST", "/oauth/token", map[string]any{"grant_type": "password", "username": "root", "password": "rootpassword1"}, 200)
	root := &api{t: t, base: base, token: tok["access_token"].(string)}
	root.must("POST", "/api/v4/projects/"+pid+"/fork", map[string]any{"namespace_path": "eve"}, 403)
	if f := root.must("POST", "/api/v4/projects/"+pid+"/fork", map[string]any{"namespace_path": "root"}, 201); f["full_path"] != "root/engine" {
		t.Fatalf("fork do administrador no próprio espaço: %v", f)
	}
}

// PR-06: the projects a person starred — only for the person (GEP 0030:
// who marked what is never shown to anyone else). /starrers is not offered
// for the same reason.
func TestProjetosMarcadosPorUmaPessoa(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	bob := signup(t, base, "bob")
	pub := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Pub", "path": "pub", "visibility": "public"}, 201)
	priv := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Priv", "path": "priv"}, 201)
	ada.must("POST", "/api/v4/projects/"+id(pub)+"/star", nil, 200)
	ada.must("POST", "/api/v4/projects/"+id(priv)+"/star", nil, 200)
	bob.must("POST", "/api/v4/projects/"+id(pub)+"/star", nil, 200)
	adaID := jsonNum(ada.must("GET", "/api/v4/user", nil, 200)["id"])

	code, body, h := ada.call("GET", "/api/v4/users/"+adaID+"/starred_projects", nil)
	if l, _ := body.([]any); code != 200 || paths(l) != "ada/priv,ada/pub" || h.Get("X-Total") != "2" {
		t.Fatalf("projetos marcados por ada: %d %v %v", code, body, h)
	}
	if got := paths(ada.list("/api/v4/users/ada/starred_projects?search=priv")); got != "ada/priv" {
		t.Fatalf("pelo nome de usuário, com pesquisa: %s", got)
	}
	if got := paths(bob.list("/api/v4/users/bob/starred_projects")); got != "ada/pub" {
		t.Fatalf("projetos marcados por bob: %s", got)
	}
	// another person's marks are private; an unknown person does not exist
	bob.must("GET", "/api/v4/users/"+adaID+"/starred_projects", nil, 403)
	bob.must("GET", "/api/v4/users/ninguem/starred_projects", nil, 404)
	(&api{t: t, base: base}).must("GET", "/api/v4/users/"+adaID+"/starred_projects", nil, 401)
	// who starred a project is never listed
	if code, _, _ := ada.call("GET", "/api/v4/projects/"+id(pub)+"/starrers", nil); code != 404 {
		t.Fatalf("/starrers não existe: %d", code)
	}
}

// NT-02: the action filter of the events, with GitLab's action names.
func TestFiltroDeAcaoDosEventos(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	eve := signup(t, base, "eve")
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Aberto", "path": "aberto", "visibility": "public"}, 201)
	pid := id(p)
	ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Um"}, 201)
	ada.must("PUT", "/api/v4/projects/"+pid+"/issues/1", map[string]any{"title": "Um!"}, 200)
	ada.must("POST", "/api/v4/projects/"+pid+"/issues/1/close", nil, 200)
	ada.must("POST", "/api/v4/projects/"+pid+"/issues/1/reopen", nil, 200)
	ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Dois", "confidential": true}, 201)
	ada.must("POST", "/api/v4/projects/"+pid+"/issues/2/close", nil, 200)
	ada.must("POST", "/api/v4/projects/"+pid+"/issues/1/notes", map[string]any{"body": "olá"}, 201)

	names := func(who *api, path string) string {
		var out []string
		for _, it := range who.list(path) {
			ev := it.(map[string]any)
			out = append(out, ev["action_name"].(string)+" "+ev["target_title"].(string))
		}
		return strings.Join(out, ", ")
	}
	events := "/api/v4/projects/" + pid + "/events"
	if got := names(ada, events+"?action=closed"); got != "closed #2 Dois, closed #1 Um!" {
		t.Fatalf("action=closed: %s", got)
	}
	if got := names(ada, events+"?action=created"); got != "opened #2 Dois, opened #1 Um" {
		t.Fatalf("action=created: %s", got)
	}
	if got := names(ada, events+"?action=updated"); got != "updated #1 Um!" {
		t.Fatalf("action=updated: %s", got)
	}
	if got := names(ada, events+"?action=reopened&target_type=issue"); got != "reopened #1 Um!" {
		t.Fatalf("action=reopened: %s", got)
	}
	// filtering never shows more: the confidential issue stays hidden
	if got := names(eve, events+"?action=closed"); got != "closed #1 Um!" {
		t.Fatalf("action=closed para eve: %s", got)
	}
	if got := names(eve, "/api/v4/events?action=created"); got != "opened #1 Um" {
		t.Fatalf("/events?action=created para eve: %s", got)
	}
	// actions that do not happen here, and pages of a filtered list
	code, body, h := ada.call("GET", events+"?action=joined", nil)
	if l, _ := body.([]any); code != 200 || len(l) != 0 || h.Get("X-Total") != "0" {
		t.Fatalf("action=joined: %d %v %v", code, body, h)
	}
	if got := names(ada, events+"?action=commented"); got != "" {
		t.Fatalf("comentários não guardam histórico aqui: %s", got)
	}
	code, body, h = ada.call("GET", events+"?action=closed&per_page=1&page=2", nil)
	if l, _ := body.([]any); code != 200 || len(l) != 1 || h.Get("X-Total") != "2" {
		t.Fatalf("página 2 de action=closed: %d %v %v", code, body, h)
	}
	if code, body, _ := ada.call("GET", events+"?action=barulho", nil); code != 400 || !strings.Contains(jsonNum(body), "action") {
		t.Fatalf("ação desconhecida: %d %v", code, body)
	}
}

// MR-07: merge by PUT (the same action as POST) and the approvals summary.
func TestMergePorPutEAprovacoes(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada") // owner
	bob := signup(t, base, "bob") // developer
	eve := signup(t, base, "eve") // not a member
	bobID := bob.must("GET", "/api/v4/user", nil, 200)["id"]
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Livro", "path": "livro", "initialize_with_readme": true}, 201)
	pid := id(p)
	ada.must("POST", "/api/v4/projects/"+pid+"/members", map[string]any{"user_id": bobID, "access_level": 30}, 201)
	ada.must("POST", "/api/v4/projects/"+pid+"/repository/branches", map[string]any{"branch": "capitulo", "ref": "main"}, 201)
	bob.must("PUT", "/api/v4/projects/"+pid+"/repository/files/um.txt", map[string]any{"branch": "capitulo", "content": "um\n", "commit_message": "um"}, 200)
	mr := bob.must("POST", "/api/v4/projects/"+pid+"/merge_requests", map[string]any{"title": "Capítulo", "source_branch": "capitulo", "target_branch": "main"}, 201)
	path := "/api/v4/projects/" + pid + "/merge_requests/" + jsonNum(mr["iid"])

	ap := ada.must("GET", path+"/approvals", nil, 200)
	// GitLab FOSS has no approval minimum: nothing required, nothing left
	if jsonNum(ap["approvals_required"]) != "0" || jsonNum(ap["approvals_left"]) != "0" || ap["approved"] != true || len(ap["approved_by"].([]any)) != 0 || ap["title"] != "Capítulo" || jsonNum(ap["iid"]) != jsonNum(mr["iid"]) {
		t.Fatalf("aprovações antes: %v", ap)
	}
	// approving is still possible, and shows who approved
	bob.must("POST", path+"/approve", nil, 200)
	if ap := bob.must("GET", path+"/approvals", nil, 200); jsonNum(ap["approvals_left"]) != "0" || len(ap["approved_by"].([]any)) != 1 {
		t.Fatalf("aprovação do autor: %v", ap)
	}
	ada.must("POST", path+"/approve", nil, 200)
	ap = bob.must("GET", path+"/approvals", nil, 200)
	who := map[string]bool{}
	for _, it := range ap["approved_by"].([]any) {
		u := it.(map[string]any)["user"].(map[string]any)
		who[u["username"].(string)] = true
		if u["email"] != nil {
			t.Fatalf("o e-mail de quem aprovou é privado: %v", u)
		}
	}
	if jsonNum(ap["approvals_left"]) != "0" || ap["approved"] != true || !who["ada"] || !who["bob"] {
		t.Fatalf("aprovações depois: %v", ap)
	}
	// someone outside a private project learns nothing, and merges nothing
	eve.must("GET", path+"/approvals", nil, 404)
	eve.must("PUT", path+"/merge", nil, 404)
	if code, _, _ := (&api{t: t, base: base}).call("GET", path+"/approvals", nil); code != 404 && code != 401 {
		t.Fatalf("aprovações sem entrar: %d", code)
	}
	// a developer may not merge into the default branch (only maintainers send code there)
	if code, _, _ := bob.call("PUT", path+"/merge", nil); code != 403 && code != 405 {
		t.Fatalf("PUT merge por developer na branch padrão: %d", code)
	}
	m := ada.must("PUT", path+"/merge", map[string]any{"squash": true}, 200)
	if m["state"] != "merged" {
		t.Fatalf("PUT merge: %v", m)
	}
	if h := ada.list("/api/v4/projects/" + pid + "/repository/commits?ref_name=main")[0].(map[string]any); h["title"] != "Capítulo" {
		t.Fatalf("squash pedido no PUT: %v", h)
	}
	ada.must("PUT", path+"/merge", nil, 405) // merged is final
}

// RP-10: "update now" of a remote mirror (POST …/remote_mirrors/:id/sync,
// GEP 0036): the same background update, with the same protections.
func TestEspelhoAtualizarAgora(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	bob := signup(t, base, "bob")
	eve := signup(t, base, "eve")
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Site", "path": "site", "initialize_with_readme": true}, 201)
	pid := id(p)
	bobID := bob.must("GET", "/api/v4/user", nil, 200)["id"]
	ada.must("POST", "/api/v4/projects/"+pid+"/members", map[string]any{"user_id": bobID, "access_level": 30}, 201)

	remoteDir := t.TempDir()
	remote := filepath.Join(remoteDir, "site.git")
	if out, err := exec.Command("git", "init", "--bare", "--quiet", "--initial-branch=main", remote).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	exec.Command("git", "--git-dir", remote, "config", "http.receivepack", "true").Run()
	execPath, _ := exec.Command("git", "--exec-path").Output()
	backend := &cgi.Handler{Path: filepath.Join(strings.TrimSpace(string(execPath)), "git-http-backend"), Env: []string{"GIT_PROJECT_ROOT=" + remoteDir, "GIT_HTTP_EXPORT_ALL=1", "REMOTE_USER=x"}}
	srv := httptest.NewServer(backend)
	defer srv.Close()
	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "1") // the other server runs on localhost in the test
	mirrors := "/api/v4/projects/" + pid + "/remote_mirrors"
	m := ada.must("POST", mirrors, map[string]any{"url": srv.URL + "/site.git", "enabled": true}, 201)
	mid := id(m)
	wait := func(what string, ok func(map[string]any) bool) map[string]any {
		t.Helper()
		var last map[string]any
		for end := time.Now().Add(20 * time.Second); time.Now().Before(end); time.Sleep(100 * time.Millisecond) {
			if last = ada.must("GET", mirrors+"/"+mid, nil, 200); ok(last) {
				return last
			}
		}
		t.Fatalf("%s: %v", what, last)
		return nil
	}
	refs := func() string {
		out, _ := exec.Command("git", "--git-dir", remote, "for-each-ref", "--format=%(refname)").Output()
		return strings.TrimSpace(string(out))
	}
	wait("a primeira atualização", func(m map[string]any) bool { return m["update_status"] == "finished" && refs() == "refs/heads/main" })

	// the other server changes behind our back; "update now" makes it an exact copy again
	if out, err := exec.Command("git", "--git-dir", remote, "update-ref", "refs/heads/lixo", "refs/heads/main").CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	bob.must("POST", mirrors+"/"+mid+"/sync", nil, 404) // a developer does not even see the mirrors
	eve.must("POST", mirrors+"/"+mid+"/sync", nil, 404) // a private project does not exist for her
	(&api{t: t, base: base}).must("POST", mirrors+"/"+mid+"/sync", nil, 404)
	if s := ada.must("POST", mirrors+"/"+mid+"/sync", nil, 200); s["update_status"] != "scheduled" && s["update_status"] != "started" && s["update_status"] != "finished" {
		t.Fatalf("sync: %v", s)
	}
	wait("o espelho voltar a ser cópia exata", func(m map[string]any) bool { return m["update_status"] == "finished" && refs() == "refs/heads/main" })

	// the protections stay: without the permission a local address fails when connecting
	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "")
	ada.must("POST", mirrors+"/"+mid+"/sync", nil, 200)
	last := wait("a atualização ser recusada", func(m map[string]any) bool { return m["update_status"] == "failed" })
	if e, _ := last["last_error"].(string); !strings.Contains(e, "rede local") {
		t.Fatalf("o motivo da recusa: %v", last)
	}
	// a disabled mirror is not updated
	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "1")
	ada.must("PUT", mirrors+"/"+mid, map[string]any{"enabled": false}, 200)
	ada.must("POST", mirrors+"/"+mid+"/sync", nil, 400)
	// an archived project is read-only: its mirrors are not updated by hand
	ada.must("PUT", mirrors+"/"+mid, map[string]any{"enabled": true}, 200)
	ada.must("PUT", "/api/v4/projects/"+pid, map[string]any{"archived": true}, 200)
	if code, _, _ := ada.call("POST", mirrors+"/"+mid+"/sync", nil); code < 400 || code == http.StatusNotFound {
		t.Fatalf("sync num projeto arquivado: %d", code)
	}
}

// PR-06: ?topic=a,b keeps the projects with all the topics (GEP 0043).
func TestVariosTopicos(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	eve := signup(t, base, "eve")
	ada.must("POST", "/api/v4/projects", map[string]any{"name": "A", "path": "a", "visibility": "public", "topics": []any{"go", "web"}}, 201)
	ada.must("POST", "/api/v4/projects", map[string]any{"name": "B", "path": "b", "visibility": "public", "topics": []any{"go"}}, 201)
	ada.must("POST", "/api/v4/projects", map[string]any{"name": "C", "path": "c", "topics": []any{"go", "web"}}, 201)
	ada.must("POST", "/api/v4/projects", map[string]any{"name": "D", "path": "d", "visibility": "public", "topics": []any{"golang", "web"}}, 201)
	list := func(who *api, q string) string { return paths(who.list("/api/v4/projects?" + q)) }
	if got := list(ada, "topic=go,web"); got != "ada/c,ada/a" {
		t.Fatalf("go e web: %s", got)
	}
	if got := list(ada, "topic=web,%20go,go"); got != "ada/c,ada/a" {
		t.Fatalf("espaços e repetição: %s", got)
	}
	if got := list(eve, "topic=go,web"); got != "ada/a" {
		t.Fatalf("o filtro não mostra o que eve não vê: %s", got)
	}
	if got := list(ada, "topic=go,rust"); got != "" {
		t.Fatalf("nenhum projeto com go e rust: %s", got)
	}
	if got := list(ada, "topic=go"); got != "ada/c,ada/b,ada/a" {
		t.Fatalf("um tópico continua: %s", got)
	}
}
