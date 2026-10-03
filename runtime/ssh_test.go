package runtime

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// sshPerson is someone with a key on disk (for the system ssh client) and
// the same key as a signer (for the Go client).
type sshPerson struct {
	c       *client
	id      float64
	keyFile string
	signer  ssh.Signer
	public  string
}

func newSSHKey(t *testing.T) (string, ssh.Signer, string) {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(priv, "teste")
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "chave")
	if err := os.WriteFile(file, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return file, signer, strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey())))
}

func sshSignUp(t *testing.T, other *client, username string, register bool) *sshPerson {
	t.Helper()
	_, c := other.fresh(t)
	me := c.expect("POST", "/cadastro", map[string]any{"username": username, "nome": username, "email": username + "@x.com", "senha": "senha-segura-1"}, 201)
	c.csrf = csrfFromCookie(t, c)
	p := &sshPerson{c: c, id: me["id"].(float64)}
	p.keyFile, p.signer, p.public = newSSHKey(t)
	if register {
		c.expect("POST", "/_ge/api/chaves", map[string]any{"titulo": "Notebook", "conteudo": p.public + " " + username + "@notebook"}, 201)
	}
	return p
}

// git runs git in dir with the system ssh client and the key of p.
func (p *sshPerson) git(t *testing.T, port, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_CONFIG_NOSYSTEM=1", "HOME="+dir,
		"GIT_AUTHOR_NAME=Teste", "GIT_AUTHOR_EMAIL=t@x.com", "GIT_COMMITTER_NAME=Teste", "GIT_COMMITTER_EMAIL=t@x.com",
		"GIT_SSH_COMMAND=ssh -F /dev/null -i "+p.keyFile+" -o IdentitiesOnly=yes -o IdentityAgent=none -o BatchMode=yes"+
			" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR -p "+port)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (p *sshPerson) mustGit(t *testing.T, port, dir string, args ...string) string {
	t.Helper()
	out, err := p.git(t, port, dir, args...)
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return out
}

func (p *sshPerson) gitFails(t *testing.T, port, dir, want string, args ...string) {
	t.Helper()
	out, err := p.git(t, port, dir, args...)
	if err == nil {
		t.Fatalf("git %v deveria falhar:\n%s", args, out)
	}
	if !strings.Contains(out, want) {
		t.Fatalf("git %v: esperava %q na saída:\n%s", args, want, out)
	}
	t.Logf("git %v recusado:\n%s", args, out)
}

// exec runs one command over SSH with the Go client and returns the exit
// status and what the server wrote on stderr.
func sshExec(t *testing.T, addr string, signer ssh.Signer, command string) (int, string, error) {
	t.Helper()
	conn, err := ssh.Dial("tcp", addr, &ssh.ClientConfig{User: "git", Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), Timeout: 5 * time.Second})
	if err != nil {
		return -1, "", err
	}
	defer conn.Close()
	sess, err := conn.NewSession()
	if err != nil {
		return -1, "", err
	}
	defer sess.Close()
	var stderr bytes.Buffer
	sess.Stderr = &stderr
	sess.Stdin = strings.NewReader("")
	err = sess.Run(command)
	if ee, ok := err.(*ssh.ExitError); ok {
		return ee.ExitStatus(), stderr.String(), nil
	}
	return 0, stderr.String(), err
}

func activityActions(t *testing.T, c *client) string {
	t.Helper()
	var out []string
	for _, a := range activities(t, c, "") {
		out = append(out, fmt.Sprint(a["acao"]))
	}
	return strings.Join(out, ",")
}

// Git over SSH (no GitLab: a collection of music scores in Git). People
// are known by the public key they registered; cloning and pushing follow
// the rules of smart HTTP: members clone the private collection, anyone
// connected clones the public one, the main branch only takes the
// conductor's pushes, someone without the right cannot push, an unknown
// key does not get in, a blocked person is refused, and nothing but
// git-upload-pack / git-receive-pack runs, on a repository that exists.
func TestGitPorSSH(t *testing.T) {
	hostKey := filepath.Join(t.TempDir(), "host", "chave")
	t.Setenv("GERMANIO_SSH_ENDERECO", "127.0.0.1:0")
	t.Setenv("GERMANIO_SSH_CHAVE_HOST", hostKey)
	t.Setenv("GERMANIO_ADMIN_SENHA", "senha-do-root-1")
	app, admin := loadApp(t, "testdata/ssh/app.ge")
	if app.SSHEndereco == "" {
		t.Fatal("o SSH não foi iniciado")
	}
	if st, err := os.Stat(hostKey); err != nil || st.Mode().Perm() != 0o600 {
		t.Fatalf("chave do servidor: %v %v", st, err)
	}
	_, port, _ := net.SplitHostPort(app.SSHEndereco)
	addr := app.SSHEndereco

	ana := sshSignUp(t, admin, "ana", true)   // owner
	bia := sshSignUp(t, admin, "bia", true)   // musician of the private collection
	caio := sshSignUp(t, admin, "caio", true) // no role anywhere
	dora := sshSignUp(t, admin, "dora", false)

	ana.c.expect("POST", "/_ge/api/acervos", map[string]any{"nome": "privado"}, 201)
	ana.c.expect("POST", "/_ge/api/acervos", map[string]any{"nome": "aberto", "visibilidade": "public"}, 201)
	ana.c.expect("POST", "/_ge/api/acervos/1/membros", map[string]any{"pessoa_id": bia.id, "papel": "musico"}, 201)

	// the owner fills the private collection over SSH
	work := t.TempDir()
	ana.mustGit(t, port, work, "clone", "--quiet", "ssh://git@127.0.0.1/privado.git", "p")
	p := filepath.Join(work, "p")
	os.WriteFile(filepath.Join(p, "sonata.ly"), []byte("\\relative c' { c d e f }\n"), 0o644)
	ana.mustGit(t, port, p, "add", ".")
	ana.mustGit(t, port, p, "commit", "--quiet", "-m", "sonata")
	ana.mustGit(t, port, p, "push", "--quiet", "origin", "HEAD:main")
	if got := activityActions(t, ana.c); !strings.Contains(got, "enviar_codigo") {
		t.Fatalf("o envio por SSH não passou pelo que segue um envio (histórico): %s", got)
	}
	// the same for the public one, without the .git suffix
	ana.mustGit(t, port, work, "clone", "--quiet", "ssh://git@127.0.0.1/aberto", "a")
	a := filepath.Join(work, "a")
	os.WriteFile(filepath.Join(a, "LEIA.md"), []byte("aberto\n"), 0o644)
	ana.mustGit(t, port, a, "add", ".")
	ana.mustGit(t, port, a, "commit", "--quiet", "-m", "leia")
	ana.mustGit(t, port, a, "push", "--quiet", "origin", "HEAD:main")

	// a member clones the private collection; the main branch is protected
	bw := t.TempDir()
	bia.mustGit(t, port, bw, "clone", "--quiet", "ssh://git@127.0.0.1/privado.git", "p")
	bp := filepath.Join(bw, "p")
	if b, _ := os.ReadFile(filepath.Join(bp, "sonata.ly")); !strings.Contains(string(b), "relative") {
		t.Fatalf("clone de quem é membro: %q", b)
	}
	os.WriteFile(filepath.Join(bp, "ideia.ly"), []byte("{ g a b }\n"), 0o644)
	// a pack larger than the SSH window: refused, it is still read to the end
	big := make([]byte, 6<<20)
	rand.Read(big)
	os.WriteFile(filepath.Join(bp, "gravacao.bin"), big, 0o644)
	bia.mustGit(t, port, bp, "add", ".")
	bia.mustGit(t, port, bp, "commit", "--quiet", "-m", "ideia")
	before := activityActions(t, ana.c)
	bia.gitFails(t, port, bp, "Somente regente pode enviar código para a branch padrão", "push", "origin", "HEAD:main")
	if after := activityActions(t, ana.c); after != before {
		t.Fatalf("um envio recusado deixou rastro: %s → %s", before, after)
	}
	bia.mustGit(t, port, bp, "push", "--quiet", "origin", "HEAD:ideias") // another branch is fine
	if out := ana.mustGit(t, port, p, "ls-remote", "origin"); !strings.Contains(out, "refs/heads/ideias") {
		t.Fatalf("a branch enviada não chegou: %s", out)
	}

	// someone without a role: the private collection does not exist for
	// them, the public one is cloned but takes no push
	cw := t.TempDir()
	caio.gitFails(t, port, cw, "não foi encontrado", "clone", "ssh://git@127.0.0.1/privado.git", "p")
	caio.mustGit(t, port, cw, "clone", "--quiet", "ssh://git@127.0.0.1/aberto.git", "a")
	ca := filepath.Join(cw, "a")
	os.WriteFile(filepath.Join(ca, "x.txt"), []byte("x\n"), 0o644)
	caio.mustGit(t, port, ca, "add", ".")
	caio.mustGit(t, port, ca, "commit", "--quiet", "-m", "x")
	caio.gitFails(t, port, ca, "não pode enviar código", "push", "origin", "HEAD:outra")
	caio.gitFails(t, port, cw, "não foi encontrado", "clone", "ssh://git@127.0.0.1/nao-existe.git", "n")

	// a key nobody registered does not get in
	dora.gitFails(t, port, t.TempDir(), "Permission denied", "clone", "ssh://git@127.0.0.1/aberto.git", "a")
	if _, _, err := sshExec(t, addr, dora.signer, "git-upload-pack '/aberto.git'"); err == nil {
		t.Fatal("chave desconhecida entrou")
	}

	// nothing but git, on a repository address without tricks
	for _, cmd := range []string{
		"ls -la",
		"sh -c 'cat /etc/passwd'",
		"git-upload-pack '/aberto.git'; id",
		"git-upload-pack '../aberto.git'",
		"git-upload-pack '/a/../aberto.git'",
		"git-upload-pack '-aberto.git'",
		"git-upload-pack '/aberto.git' --help",
		"git-upload-archive '/aberto.git'",
		"git-receive-pack '/aberto.git' && touch /tmp/x",
		"git-upload-pack '/" + strings.Repeat("a/", 600) + "x.git'",
	} {
		status, stderr, err := sshExec(t, addr, caio.signer, cmd)
		if err != nil || status == 0 || !strings.Contains(stderr, "só é possível usar git") {
			t.Fatalf("comando %q: status %d, erro %v, saída %q", cmd, status, err, stderr)
		}
	}
	if status, stderr, err := sshExec(t, addr, caio.signer, "git-upload-pack 'aberto/../privado.git'"); err != nil || status == 0 || strings.Contains(stderr, "sonata") {
		t.Fatalf("caminho com ..: %d %v %q", status, err, stderr)
	}
	out, err := exec.Command("ssh", "-F", "/dev/null", "-i", caio.keyFile, "-o", "IdentitiesOnly=yes", "-o", "IdentityAgent=none",
		"-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-o", "LogLevel=ERROR",
		"-p", port, "git@127.0.0.1", "cat", "/etc/passwd").CombinedOutput()
	if err == nil || strings.Contains(string(out), "root:") || !strings.Contains(string(out), "só é possível usar git") {
		t.Fatalf("ssh com outro comando: %v\n%s", err, out)
	}
	// a session without a command only says who the key belongs to
	if out, err := exec.Command("ssh", "-T", "-F", "/dev/null", "-i", caio.keyFile, "-o", "IdentitiesOnly=yes", "-o", "IdentityAgent=none",
		"-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-o", "LogLevel=ERROR",
		"-p", port, "git@127.0.0.1").CombinedOutput(); err != nil || !strings.Contains(string(out), "Olá, caio") {
		t.Fatalf("ssh -T: %v\n%s", err, out)
	}

	// a blocked person is refused, with the reason
	admin.expect("POST", "/entrar", map[string]any{"login": "root", "senha": "senha-do-root-1"}, 200)
	admin.csrf = csrfFromCookie(t, admin)
	admin.expect("POST", fmt.Sprintf("/_ge/api/usuarios/%d/bloquear", int(caio.id)), nil, 200)
	caio.gitFails(t, port, t.TempDir(), "não está ativa", "clone", "ssh://git@127.0.0.1/aberto.git", "a")
}

// SSH is opt-in: without GERMANIO_SSH_ENDERECO nothing listens; with it, a
// program without repositories or without people's keys does not start,
// and the error says what to declare.
func TestGitPorSSHConfiguracao(t *testing.T) {
	t.Setenv("GERMANIO_SQLITE", filepath.Join(t.TempDir(), "a.db"))
	t.Setenv("GERMANIO_GIT_RAIZ", filepath.Join(t.TempDir(), "repos"))
	app, err := Carregar("testdata/baixar_codigo/app.ge", "0")
	if err != nil {
		t.Fatal(err)
	}
	if app.SSHEndereco != "" {
		t.Fatalf("SSH ligado sem configuração: %s", app.SSHEndereco)
	}
	app.Fechar()
	t.Setenv("GERMANIO_SSH_ENDERECO", "127.0.0.1:0")
	for file, want := range map[string]string{
		"testdata/baixar_codigo/app.ge": "chave pública",
		"testdata/chaves/app.ge":        "não tem repositórios",
	} {
		t.Setenv("GERMANIO_SQLITE", filepath.Join(t.TempDir(), "a.db"))
		if app, err := Carregar(file, "0"); err == nil || !strings.Contains(err.Error(), want) {
			if app != nil {
				app.Fechar()
			}
			t.Fatalf("%s: esperava erro com %q, veio %v", file, want, err)
		}
	}
}
