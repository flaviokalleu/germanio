package e2e

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"net"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	germanio "github.com/flaviokalleu/germanio/runtime"
)

// gitlabSSH starts the GitLab written in .ge with Git over SSH turned on
// (configuration only) and returns the web address and the SSH port.
func gitlabSSH(t *testing.T) (string, string) {
	t.Helper()
	t.Setenv("GERMANIO_SQLITE", filepath.Join(t.TempDir(), "gitlab.db"))
	t.Setenv("GERMANIO_GIT_RAIZ", filepath.Join(t.TempDir(), "repos"))
	t.Setenv("GERMANIO_ARQUIVOS", filepath.Join(t.TempDir(), "arquivos"))
	t.Setenv("GERMANIO_BCRYPT_RAPIDO", "1")
	t.Setenv("GERMANIO_ADMIN_SENHA", "rootpassword1")
	t.Setenv("GERMANIO_ADMIN_EMAIL", "admin@example.com")
	t.Setenv("GERMANIO_SSH_ENDERECO", "127.0.0.1:0")
	t.Setenv("GERMANIO_SSH_CHAVE_HOST", filepath.Join(t.TempDir(), "ssh", "chave_host"))
	app, err := germanio.Carregar("../app.ge", "0")
	if err != nil {
		t.Fatalf("Carregar: %v", err)
	}
	srv := httptest.NewServer(app.Handler)
	t.Cleanup(func() { srv.Close(); app.Fechar() })
	if app.SSHEndereco == "" {
		t.Fatal("SSH não iniciado")
	}
	_, port, _ := net.SplitHostPort(app.SSHEndereco)
	return srv.URL, port
}

// sshKeyFile writes a new private key for the system ssh client and
// returns its file and its public half (authorized_keys form).
func sshKeyFile(t *testing.T) (string, string) {
	t.Helper()
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(file, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatal(err)
	}
	signer, _ := ssh.NewSignerFromKey(priv)
	return file, strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey())))
}

func gitOverSSH(t *testing.T, key, port, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_CONFIG_NOSYSTEM=1", "HOME="+dir,
		"GIT_AUTHOR_NAME=Ada", "GIT_AUTHOR_EMAIL=ada@example.com", "GIT_COMMITTER_NAME=Ada", "GIT_COMMITTER_EMAIL=ada@example.com",
		"GIT_SSH_COMMAND=ssh -F /dev/null -i "+key+" -o IdentitiesOnly=yes -o IdentityAgent=none -o BatchMode=yes"+
			" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR -p "+port)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// ID-08 / RP-10: a key registered at /user/keys clones and pushes over SSH
// (ssh://git@host/<namespace>/<project>.git), with the project's rules: a
// non-member does not see a private project, a removed key no longer
// gets in, and a push over SSH shows up in the repository API.
func TestGitPorSSHGitLab(t *testing.T) {
	base, port := gitlabSSH(t)
	ada := signup(t, base, "ada")
	eve := signup(t, base, "eve")
	adaKey, adaPub := sshKeyFile(t)
	eveKey, evePub := sshKeyFile(t)
	k := ada.must("POST", "/api/v4/user/keys", map[string]any{"title": "Notebook", "key": adaPub + " ada@notebook"}, 201)
	eve.must("POST", "/api/v4/user/keys", map[string]any{"title": "Eve", "key": evePub}, 201)
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "App", "path": "app", "initialize_with_readme": true}, 201)

	work := t.TempDir()
	if out, err := gitOverSSH(t, adaKey, port, work, "clone", "--quiet", "ssh://git@127.0.0.1/ada/app.git", "app"); err != nil {
		t.Fatalf("clone por SSH: %v\n%s", err, out)
	}
	dir := filepath.Join(work, "app")
	if b, _ := os.ReadFile(filepath.Join(dir, "README.md")); !strings.Contains(string(b), "# App") {
		t.Fatalf("README do clone: %q", b)
	}
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644)
	for _, args := range [][]string{{"add", "."}, {"commit", "--quiet", "-m", "Pelo SSH"}, {"push", "--quiet", "origin", "HEAD:main"}} {
		if out, err := gitOverSSH(t, adaKey, port, dir, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	commits := ada.list("/api/v4/projects/" + id(p) + "/repository/commits")
	if len(commits) == 0 || commits[0].(map[string]any)["title"] != "Pelo SSH" {
		t.Fatalf("o envio por SSH não aparece nos commits: %v", commits)
	}

	// eve has a key, but the private project is not hers
	if out, err := gitOverSSH(t, eveKey, port, t.TempDir(), "clone", "ssh://git@127.0.0.1/ada/app.git", "x"); err == nil || !strings.Contains(out, "não foi encontrado") {
		t.Fatalf("quem não é membro clonou: %v\n%s", err, out)
	}
	// a removed key no longer gets in
	ada.must("DELETE", "/api/v4/user/keys/"+id(k), nil, 204)
	if out, err := gitOverSSH(t, adaKey, port, t.TempDir(), "clone", "ssh://git@127.0.0.1/ada/app.git", "x"); err == nil || !strings.Contains(out, "Permission denied") {
		t.Fatalf("chave removida entrou: %v\n%s", err, out)
	}
}
