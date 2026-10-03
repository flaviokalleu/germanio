package e2e

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cgi"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// lfsCall speaks the Git LFS batch API (and basic transfer) as the official
// client does, with git's basic credentials or the headers of an action.
func lfsCall(t *testing.T, method, u, user, pass string, header map[string]string, body []byte) (int, []byte) {
	t.Helper()
	req, _ := http.NewRequest(method, u, bytes.NewReader(body))
	req.Header.Set("Accept", "application/vnd.git-lfs+json")
	if method == "POST" {
		req.Header.Set("Content-Type", "application/vnd.git-lfs+json")
	}
	if user != "" {
		req.SetBasicAuth(user, pass)
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

type lfsBatch struct {
	Objects []struct {
		OID     string `json:"oid"`
		Actions map[string]struct {
			Href   string            `json:"href"`
			Header map[string]string `json:"header"`
		} `json:"actions"`
		Error *struct {
			Code int `json:"code"`
		} `json:"error"`
	} `json:"objects"`
}

// RP-10 (LFS): large files of a project travel through the LFS batch API
// with the same people and credentials as git push and clone.
func TestGitLFSNoProjeto(t *testing.T) {
	t.Setenv("GERMANIO_ARQUIVOS", filepath.Join(t.TempDir(), "arquivos"))
	base := gitlab(t)
	ada := signup(t, base, "ada")
	bob := signup(t, base, "bob")
	eve := signup(t, base, "eve")
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Jogo", "path": "jogo", "initialize_with_readme": true}, 201)
	pid := id(p)
	bobID := bob.must("GET", "/api/v4/user", nil, 200)["id"]
	ada.must("POST", "/api/v4/projects/"+pid+"/members", map[string]any{"user_id": bobID, "access_level": 20}, 201) // reporter
	pat := func(who *api, name string) string {
		return who.must("POST", "/api/v4/personal_access_tokens", map[string]any{"name": "git"}, 201)["token"].(string)
	}
	adaPAT, bobPAT, evePAT := pat(ada, "ada"), pat(bob, "bob"), pat(eve, "eve")
	lfs := base + "/ada/jogo.git/info/lfs/objects/batch"
	asset := bytes.Repeat([]byte("textura;"), 4096)
	sum := sha256.Sum256(asset)
	oid := hex.EncodeToString(sum[:])
	req := func(op string) []byte {
		b, _ := json.Marshal(map[string]any{"operation": op, "transfers": []string{"basic"}, "hash_algo": "sha256", "objects": []any{map[string]any{"oid": oid, "size": len(asset)}}})
		return b
	}

	if code, _ := lfsCall(t, "POST", lfs, "eve", evePAT, nil, req("download")); code != 404 {
		t.Fatalf("projeto privado para quem não é membro: %d", code)
	}
	if code, _ := lfsCall(t, "POST", lfs, "bob", bobPAT, nil, req("upload")); code != 403 {
		t.Fatalf("reporter enviando arquivo grande: %d", code)
	}

	if _, err := exec.LookPath("git-lfs"); err == nil {
		// the official client, end to end
		u, _ := url.Parse(base)
		u.User = url.UserPassword("ada", adaPAT)
		dir := t.TempDir()
		run(t, dir, "git", "clone", "--quiet", u.String()+"/ada/jogo.git", "jogo")
		work := filepath.Join(dir, "jogo")
		run(t, work, "git", "lfs", "install", "--local")
		run(t, work, "git", "lfs", "track", "*.bin")
		os.WriteFile(filepath.Join(work, "textura.bin"), asset, 0o644)
		run(t, work, "git", "add", ".gitattributes", "textura.bin")
		run(t, work, "git", "-c", "user.name=Ada", "-c", "user.email=ada@example.com", "commit", "--quiet", "-m", "textura")
		run(t, work, "git", "push", "--quiet", "origin", "main")
		// the repository keeps a pointer; the content comes back through LFS
		f := ada.must("GET", "/api/v4/projects/"+pid+"/repository/files/textura.bin", nil, 200)
		if !strings.Contains(f["content"].(string), "oid sha256:"+oid) {
			t.Fatalf("o repositório deveria guardar o ponteiro LFS: %v", f["content"])
		}
		ub, _ := url.Parse(base)
		ub.User = url.UserPassword("bob", bobPAT)
		run(t, dir, "git", "clone", "--quiet", ub.String()+"/ada/jogo.git", "copia")
		copia := filepath.Join(dir, "copia")
		run(t, copia, "git", "lfs", "install", "--local")
		run(t, copia, "git", "lfs", "pull")
		if got, _ := os.ReadFile(filepath.Join(copia, "textura.bin")); !bytes.Equal(got, asset) {
			t.Fatalf("o reporter baixou %d bytes, esperado o arquivo de %d", len(got), len(asset))
		}
	} else {
		// no official client here: the same exchange, as the specification describes it
		t.Log("git-lfs não instalado: protocolo testado com requisições HTTP")
		code, raw := lfsCall(t, "POST", lfs, "ada", adaPAT, nil, req("upload"))
		var up lfsBatch
		json.Unmarshal(raw, &up)
		if code != 200 || len(up.Objects) != 1 {
			t.Fatalf("batch de envio: %d %s", code, raw)
		}
		a := up.Objects[0].Actions["upload"]
		if code, raw := lfsCall(t, "PUT", a.Href, "", "", a.Header, asset); code != 200 {
			t.Fatalf("envio do objeto: %d %s", code, raw)
		}
	}

	code, raw := lfsCall(t, "POST", lfs, "bob", bobPAT, nil, req("download"))
	var down lfsBatch
	json.Unmarshal(raw, &down)
	if code != 200 || len(down.Objects) != 1 || down.Objects[0].Error != nil {
		t.Fatalf("batch de download do reporter: %d %s", code, raw)
	}
	a := down.Objects[0].Actions["download"]
	if code, got := lfsCall(t, "GET", a.Href, "", "", a.Header, nil); code != 200 || !bytes.Equal(got, asset) {
		t.Fatalf("download: %d (%d bytes)", code, len(got))
	}
	if code, _ := lfsCall(t, "GET", a.Href, "eve", evePAT, nil, nil); code != 404 {
		t.Fatalf("download direto por quem não é membro: %d", code)
	}
}

// RP-10 (mirrors): remote mirrors are ordinary data of the project; after a
// git push the repository goes to the other server, with the credentials
// written in the url hidden and the result recorded.
func TestEspelhosRemotos(t *testing.T) {
	base := gitlab(t)
	ada := signup(t, base, "ada")
	bob := signup(t, base, "bob")
	eve := signup(t, base, "eve")
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Site", "path": "site", "initialize_with_readme": true}, 201)
	pid := id(p)
	bobID := bob.must("GET", "/api/v4/user", nil, 200)["id"]
	ada.must("POST", "/api/v4/projects/"+pid+"/members", map[string]any{"user_id": bobID, "access_level": 30}, 201) // developer

	// another Git server: git's own http-backend, behind basic authentication
	remoteDir := t.TempDir()
	if out, err := exec.Command("git", "init", "--bare", "--quiet", "--initial-branch=main", filepath.Join(remoteDir, "site.git")).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	exec.Command("git", "--git-dir", filepath.Join(remoteDir, "site.git"), "config", "http.receivepack", "true").Run()
	execPath, _ := exec.Command("git", "--exec-path").Output()
	backend := &cgi.Handler{Path: filepath.Join(strings.TrimSpace(string(execPath)), "git-http-backend"), Env: []string{"GIT_PROJECT_ROOT=" + remoteDir, "GIT_HTTP_EXPORT_ALL=1", "REMOTE_USER=x"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, pw, ok := r.BasicAuth(); !ok || u != "espelho" || pw != "token-do-espelho-123" {
			w.Header().Set("WWW-Authenticate", `Basic realm="x"`)
			http.Error(w, "auth", 401)
			return
		}
		backend.ServeHTTP(w, r)
	}))
	defer srv.Close()
	mirrorURL := strings.Replace(srv.URL, "http://", "http://espelho:token-do-espelho-123@", 1) + "/site.git"
	mirrors := "/api/v4/projects/" + pid + "/remote_mirrors"

	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "")
	if code, out, _ := ada.call("POST", mirrors, map[string]any{"url": mirrorURL, "enabled": true}); code != 400 || !strings.Contains(jsonNum(out), "GERMANIO_PERMITIR_REDE_LOCAL") {
		t.Fatalf("espelho na rede local sem permissão: %d %v", code, out)
	}
	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "1") // the other server runs on localhost in the test
	bob.must("POST", mirrors, map[string]any{"url": mirrorURL}, 403)
	if code, _, _ := eve.call("GET", mirrors, nil); code != 404 {
		t.Fatalf("espelhos de projeto privado para quem não é membro: %d", code)
	}
	m := ada.must("POST", mirrors, map[string]any{"url": mirrorURL, "enabled": true}, 201)
	if m["url"] != srv.URL+"/site.git" || m["enabled"] != true || strings.Contains(jsonNum(m), "token-do-espelho") {
		t.Fatalf("espelho criado (a credencial nunca volta): %v", m)
	}
	mid := id(m)

	// a real git push reaches the mirror
	pat := ada.must("POST", "/api/v4/personal_access_tokens", map[string]any{"name": "git"}, 201)["token"].(string)
	u, _ := url.Parse(base)
	u.User = url.UserPassword("ada", pat)
	dir := t.TempDir()
	run(t, dir, "git", "clone", "--quiet", u.String()+"/ada/site.git", "site")
	work := filepath.Join(dir, "site")
	os.WriteFile(filepath.Join(work, "index.html"), []byte("<h1>oi</h1>"), 0o644)
	run(t, work, "git", "add", "index.html")
	run(t, work, "git", "-c", "user.name=Ada", "-c", "user.email=ada@example.com", "commit", "--quiet", "-m", "página")
	run(t, work, "git", "push", "--quiet", "origin", "main")
	head := strings.TrimSpace(run(t, work, "git", "rev-parse", "HEAD"))
	var last map[string]any
	for end := time.Now().Add(20 * time.Second); time.Now().Before(end); time.Sleep(100 * time.Millisecond) {
		last = ada.must("GET", mirrors+"/"+mid, nil, 200)
		got, _ := exec.Command("git", "--git-dir", filepath.Join(remoteDir, "site.git"), "rev-parse", "refs/heads/main").Output()
		if strings.TrimSpace(string(got)) == head && last["update_status"] == "finished" {
			break
		}
	}
	if last["update_status"] != "finished" || last["last_successful_update_at"] == nil {
		t.Fatalf("o push não chegou ao espelho: %v", last)
	}
	if list := ada.list(mirrors); len(list) != 1 || strings.Contains(jsonNum(list), "token-do-espelho") {
		t.Fatalf("lista de espelhos: %v", list)
	}
}
