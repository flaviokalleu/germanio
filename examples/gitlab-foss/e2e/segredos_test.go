package e2e

import (
	"bytes"
	"database/sql"
	"net/http"
	"net/http/cgi"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	germanio "github.com/flaviokalleu/germanio/runtime"
)

// Secrets at rest (GEP 0049, em teste) in the GitLab written in .ge, whose
// domain did not change: the credential of a remote mirror, the secret token
// of a webhook and the value of a CI variable are sealed in the database
// (nothing in clear in the file), never come back through the API, and the
// mirror and the webhook keep working with them.
func TestSegredosNoBanco(t *testing.T) {
	t.Setenv("GERMANIO_SEGREDO", "segredo-do-gitlab-de-teste-com-mais-de-32-caracteres")
	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "1") // the other servers run on localhost
	db := filepath.Join(t.TempDir(), "gitlab.db")
	t.Setenv("GERMANIO_SQLITE", db)
	t.Setenv("GERMANIO_GIT_RAIZ", filepath.Join(t.TempDir(), "repos"))
	t.Setenv("GERMANIO_ARQUIVOS", filepath.Join(t.TempDir(), "arquivos"))
	t.Setenv("GERMANIO_BCRYPT_RAPIDO", "1")
	app, err := germanio.Carregar("../app.ge", "0")
	if err != nil {
		t.Fatalf("Carregar: %v", err)
	}
	srv := httptest.NewServer(app.Handler)
	closed := false
	stop := func() {
		if !closed {
			closed = true
			srv.Close()
			app.Fechar()
		}
	}
	defer stop()
	base := srv.URL

	const (
		mirrorPass = "senha-do-espelho-QWE-741"
		hookToken  = "token-do-webhook-RTY-852"
		varValue   = "valor-da-variavel-UIO-963"
	)
	ada := signup(t, base, "ada")
	pid := id(ada.must("POST", "/api/v4/projects", map[string]any{"name": "Cofre", "path": "cofre", "initialize_with_readme": true}, 201))

	// another Git server behind basic authentication
	remoteDir := t.TempDir()
	if out, err := exec.Command("git", "init", "--bare", "--quiet", "--initial-branch=main", filepath.Join(remoteDir, "cofre.git")).CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	exec.Command("git", "--git-dir", filepath.Join(remoteDir, "cofre.git"), "config", "http.receivepack", "true").Run()
	execPath, _ := exec.Command("git", "--exec-path").Output()
	backend := &cgi.Handler{Path: filepath.Join(strings.TrimSpace(string(execPath)), "git-http-backend"), Env: []string{"GIT_PROJECT_ROOT=" + remoteDir, "GIT_HTTP_EXPORT_ALL=1", "REMOTE_USER=x"}}
	gitSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, pw, ok := r.BasicAuth(); !ok || u != "espelho" || pw != mirrorPass {
			w.Header().Set("WWW-Authenticate", `Basic realm="x"`)
			http.Error(w, "auth", 401)
			return
		}
		backend.ServeHTTP(w, r)
	}))
	defer gitSrv.Close()

	// a receiver of webhooks
	var mu sync.Mutex
	var tokens []string
	recv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		tokens = append(tokens, r.Header.Get("X-Gitlab-Token"))
		mu.Unlock()
	}))
	defer recv.Close()

	mirrors := "/api/v4/projects/" + pid + "/remote_mirrors"
	m := ada.must("POST", mirrors, map[string]any{"url": strings.Replace(gitSrv.URL, "http://", "http://espelho:"+mirrorPass+"@", 1) + "/cofre.git", "enabled": true}, 201)
	h := ada.must("POST", "/api/v4/projects/"+pid+"/hooks", map[string]any{"url": recv.URL, "token": hookToken}, 201)
	v := ada.must("POST", "/api/v4/projects/"+pid+"/variables", map[string]any{"key": "DEPLOY_TOKEN", "value": varValue}, 201)
	for _, out := range []map[string]any{m, h, v} {
		raw := jsonNum(out)
		if strings.Contains(raw, mirrorPass) || strings.Contains(raw, hookToken) || strings.Contains(raw, varValue) {
			t.Fatalf("um segredo voltou na resposta: %s", raw)
		}
	}

	// the database holds ciphertext only
	conn, err := sql.Open("sqlite", db)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	for _, col := range []struct{ table, column, clear string }{
		{"espelho", "credencial", mirrorPass},
		{"webhook", "token", hookToken},
		{"variavel", "valor", varValue},
	} {
		var stored string
		if err := conn.QueryRow(`SELECT ` + col.column + ` FROM ` + col.table).Scan(&stored); err != nil {
			t.Fatalf("%s.%s: %v", col.table, col.column, err)
		}
		if !strings.HasPrefix(stored, "ge1:") || strings.Contains(stored, col.clear) {
			t.Fatalf("%s.%s guardado em claro: %q", col.table, col.column, stored)
		}
	}

	// the mirror still authenticates to the other server
	ada.must("POST", mirrors+"/"+id(m)+"/sync", nil, 200)
	var last map[string]any
	for end := time.Now().Add(20 * time.Second); time.Now().Before(end); time.Sleep(100 * time.Millisecond) {
		last = ada.must("GET", mirrors+"/"+id(m), nil, 200)
		got, _ := exec.Command("git", "--git-dir", filepath.Join(remoteDir, "cofre.git"), "rev-parse", "--verify", "--quiet", "refs/heads/main").Output()
		if len(bytes.TrimSpace(got)) > 0 && last["update_status"] == "finished" {
			break
		}
	}
	if last["update_status"] != "finished" {
		t.Fatalf("o espelho não chegou ao outro servidor com a credencial cifrada: %v", last)
	}
	// the webhook still sends its token
	ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Segredo"}, 201)
	deadline := time.Now().Add(15 * time.Second)
	for {
		mu.Lock()
		n := len(tokens)
		mu.Unlock()
		if n > 0 || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	mu.Lock()
	if len(tokens) == 0 || tokens[0] != hookToken {
		t.Fatalf("o webhook não enviou o token: %v", tokens)
	}
	mu.Unlock()
	// the lists never show them either
	for _, path := range []string{mirrors, "/api/v4/projects/" + pid + "/hooks", "/api/v4/projects/" + pid + "/variables"} {
		if raw := jsonNum(ada.list(path)); strings.Contains(raw, mirrorPass) || strings.Contains(raw, hookToken) || strings.Contains(raw, varValue) {
			t.Fatalf("%s mostra um segredo: %s", path, raw)
		}
	}

	// and the file on disk has none of them in clear
	conn.Close()
	stop()
	var all []byte
	for _, f := range []string{db, db + "-wal"} {
		if b, err := os.ReadFile(f); err == nil {
			all = append(all, b...)
		}
	}
	for _, s := range []string{mirrorPass, hookToken, varValue} {
		if bytes.Contains(all, []byte(s)) {
			t.Fatalf("o arquivo do banco contém %q em texto puro", s)
		}
	}
}
