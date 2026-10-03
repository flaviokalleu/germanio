package git

import (
	"net/http"
	"net/http/cgi"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// remoteServer serves the bare repositories of dir over smart HTTP with git's
// own http-backend (an independent Git server on 127.0.0.1), optionally
// behind basic authentication.
func remoteServer(t *testing.T, dir, user, pass string) *httptest.Server {
	t.Helper()
	execPath, err := exec.Command("git", "--exec-path").Output()
	if err != nil {
		t.Skip("git indisponível")
	}
	backend := filepath.Join(strings.TrimSpace(string(execPath)), "git-http-backend")
	if _, err := os.Stat(backend); err != nil {
		t.Skip("git-http-backend indisponível")
	}
	h := &cgi.Handler{Path: backend, Env: []string{"GIT_PROJECT_ROOT=" + dir, "GIT_HTTP_EXPORT_ALL=1", "REMOTE_USER=espelho"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user != "" {
			if u, p, ok := r.BasicAuth(); !ok || u != user || p != pass {
				w.Header().Set("WWW-Authenticate", `Basic realm="x"`)
				http.Error(w, "auth", http.StatusUnauthorized)
				return
			}
		}
		h.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func commitOn(t *testing.T, s *Store, rel, branch, file, content string) string {
	t.Helper()
	id, err := s.CommitFiles(rel, branch, "", "muda "+file, Signature{Name: "A", Email: "a@x"}, []Action{{Kind: "create", Path: file, Content: []byte(content)}})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// A push mirror copies branches and tags (and removals) to another Git
// server; a fetch mirror brings them back; credentials are sent in a header
// and never appear in errors.
func TestEspelhos(t *testing.T) {
	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "1")
	remoteDir := t.TempDir()
	remote, err := NewStore(remoteDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := remote.Init("destino.git", "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := remote.run(filepath.Join(remoteDir, "destino.git"), nil, nil, "config", "http.receivepack", "true"); err != nil {
		t.Fatal(err)
	}
	srv := remoteServer(t, remoteDir, "ana", "s3gr3do-forte")
	s, _ := NewStore(t.TempDir())
	s.Init("p.git", "main")
	head := commitOn(t, s, "p.git", "main", "a.txt", "a")
	s.CreateBranch("p.git", "dev", "main")
	s.CreateTag("p.git", "v1", "main")

	r := Remote{URL: srv.URL + "/destino.git", User: "ana", Password: "s3gr3do-forte"}
	if err := s.PushMirror(t.Context(), "p.git", r); err != nil {
		t.Fatalf("espelho de envio: %v", err)
	}
	if got, _ := remote.Resolve("destino.git", "refs/heads/main"); got != head {
		t.Fatalf("main no destino = %q, esperado %q", got, head)
	}
	if !remote.BranchExists("destino.git", "dev") {
		t.Fatal("branch dev não chegou ao destino")
	}
	if tags, _ := remote.Tags("destino.git"); len(tags) != 1 {
		t.Fatalf("tags no destino: %v", tags)
	}
	// a removed branch disappears from the mirror too
	s.DeleteBranch("p.git", "dev")
	if err := s.PushMirror(t.Context(), "p.git", r); err != nil {
		t.Fatal(err)
	}
	if remote.BranchExists("destino.git", "dev") {
		t.Fatal("branch removida continua no espelho")
	}
	// wrong password: refused, and the password never shows in the error
	bad := r
	bad.Password = "senha-errada-123"
	err = s.PushMirror(t.Context(), "p.git", bad)
	if err == nil || strings.Contains(err.Error(), "senha-errada-123") {
		t.Fatalf("senha errada: %v", err)
	}

	// fetch mirror: another repository becomes a copy of the remote
	commitOn(t, remote, "destino.git", "main", "b.txt", "b")
	want, _ := remote.Resolve("destino.git", "refs/heads/main")
	s.Init("copia.git", "main")
	updates, err := s.FetchMirror(t.Context(), "copia.git", r)
	if err != nil {
		t.Fatalf("espelho de recebimento: %v", err)
	}
	if got, _ := s.Resolve("copia.git", "refs/heads/main"); got != want || len(updates) != 2 {
		t.Fatalf("main na cópia = %q (esperado %q), atualizações %v", got, want, updates)
	}
	if again, err := s.FetchMirror(t.Context(), "copia.git", r); err != nil || len(again) != 0 {
		t.Fatalf("sem mudanças no remoto não há atualizações: %v %v", again, err)
	}
}

// Without GERMANIO_PERMITIR_REDE_LOCAL, local addresses are refused both when
// the address is read and when git would connect; only http(s) is accepted.
func TestEspelhoRecusaRedeLocalEOutrosTransportes(t *testing.T) {
	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "")
	for _, u := range []string{"http://127.0.0.1:1/x.git", "https://localhost/x.git", "http://[::1]/x.git", "http://10.0.0.5/x.git", "http://2130706433/x.git"} {
		if _, _, _, err := ParseRemoteURL(u); err == nil || !strings.Contains(err.Error(), "GERMANIO_PERMITIR_REDE_LOCAL") {
			t.Fatalf("%s aceito: %v", u, err)
		}
	}
	for _, u := range []string{"file:///etc", "/srv/repos/x.git", "ext::sh -c id", "ssh://git@exemplo.com/x.git", "git://exemplo.com/x.git", "-uhttps://x", ""} {
		if _, _, _, err := ParseRemoteURL(u); err == nil {
			t.Fatalf("%q aceito", u)
		}
	}
	clean, user, pass, err := ParseRemoteURL("https://ana:p%40ss@exemplo.com/g/p.git")
	if err != nil || clean != "https://exemplo.com/g/p.git" || user != "ana" || pass != "p@ss" {
		t.Fatalf("credenciais: %q %q %q %v", clean, user, pass, err)
	}
	// a public name that resolves to a local address is refused when git
	// connects (the guarded dialer checks the resolved address)
	s, _ := NewStore(t.TempDir())
	s.Init("p.git", "main")
	commitOn(t, s, "p.git", "main", "a.txt", "a")
	err = s.PushMirror(t.Context(), "p.git", Remote{URL: "http://localtest.me:9/x.git"})
	if err == nil {
		t.Fatal("envio para endereço que resolve para a rede local aceito")
	}
	t.Log(err)
	if !strings.Contains(err.Error(), "rede local") && !strings.Contains(err.Error(), "conectar") {
		t.Fatalf("erro sem explicação: %v", err)
	}
}
