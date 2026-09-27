package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	germanio "github.com/flaviokalleu/germanio/runtime"
)

// gitlab starts the GitLab written in .ge on a test server.
func gitlab(t *testing.T) string {
	t.Helper()
	t.Setenv("GERMANIO_SQLITE", filepath.Join(t.TempDir(), "gitlab.db"))
	t.Setenv("GERMANIO_GIT_RAIZ", filepath.Join(t.TempDir(), "repos"))
	t.Setenv("GERMANIO_BCRYPT_RAPIDO", "1")
	t.Setenv("GITLAB_ROOT_PASSWORD", "rootpassword1")
	app, err := germanio.Carregar("../app.ge", "0")
	if err != nil {
		t.Fatalf("Carregar: %v", err)
	}
	srv := httptest.NewServer(app.Handler)
	t.Cleanup(func() { srv.Close(); app.Fechar() })
	return srv.URL
}

type api struct {
	t     *testing.T
	base  string
	token string // Bearer or PRIVATE-TOKEN
	pat   bool
}

func (a *api) call(method, path string, body any) (int, any, http.Header) {
	a.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, a.base+path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if a.token != "" {
		if a.pat {
			req.Header.Set("PRIVATE-TOKEN", a.token)
		} else {
			req.Header.Set("Authorization", "Bearer "+a.token)
		}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		a.t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out any
	json.Unmarshal(raw, &out)
	return resp.StatusCode, out, resp.Header
}

func (a *api) must(method, path string, body any, status int) map[string]any {
	a.t.Helper()
	code, out, _ := a.call(method, path, body)
	if code != status {
		a.t.Fatalf("%s %s = %d, esperado %d: %v", method, path, code, status, out)
	}
	m, _ := out.(map[string]any)
	return m
}

func (a *api) list(path string) []any {
	a.t.Helper()
	code, out, _ := a.call("GET", path, nil)
	if code != 200 {
		a.t.Fatalf("GET %s = %d: %v", path, code, out)
	}
	l, _ := out.([]any)
	return l
}

// signup registers through the sign-up form and logs in with OAuth.
func signup(t *testing.T, base, username string) *api {
	anon := &api{t: t, base: base}
	anon.must("POST", "/cadastro", map[string]any{"username": username, "nome": strings.ToUpper(username[:1]) + username[1:], "email": username + "@example.com", "senha": "password123"}, 201)
	tok := anon.must("POST", "/oauth/token", map[string]any{"grant_type": "password", "username": username, "password": "password123"}, 200)
	return &api{t: t, base: base, token: tok["access_token"].(string)}
}

func run(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_CONFIG_NOSYSTEM=1", "HOME="+dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: %v\n%s", args, err, out)
	}
	return string(out)
}

func runFails(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_CONFIG_NOSYSTEM=1", "HOME="+dir)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("%v deveria falhar:\n%s", args, out)
	}
	return string(out)
}

func id(m map[string]any) string { return jsonNum(m["id"]) }

func jsonNum(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
