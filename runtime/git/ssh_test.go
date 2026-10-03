package git

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

func TestParseSSHCommand(t *testing.T) {
	good := map[string][2]string{
		"git-upload-pack '/grupo/projeto.git'": {"upload-pack", "grupo/projeto"},
		"git-receive-pack 'projeto.git'":       {"receive-pack", "projeto"},
		"git upload-pack '/a/b/c'":             {"upload-pack", "a/b/c"},
		"git-upload-pack /x_1.y-z.git":         {"upload-pack", "x_1.y-z"},
	}
	for cmd, want := range good {
		s, a, err := ParseSSHCommand(cmd)
		if err != nil || s != want[0] || a != want[1] {
			t.Fatalf("%q → %q %q %v", cmd, s, a, err)
		}
	}
	for _, cmd := range []string{
		"", "ls", "bash", "git-upload-archive '/p.git'", "git-upload-pack", "git-upload-pack ''",
		"git-upload-pack '../p.git'", "git-upload-pack '/a/../p.git'", "git-upload-pack '/a/./p.git'",
		"git-upload-pack '/a//p.git'", "git-upload-pack '-p.git'", "git-upload-pack '/.hidden.git'",
		"git-upload-pack '/p.git' x", "git-upload-pack '/p.git';id", "git-upload-pack '/p q.git'",
		"git-upload-pack '/p\\'.git'", "git-upload-pack '/p$(id).git'", "git-upload-pack '/p.git'\nid",
		"git-receive-pack --help", "git-upload-pack '/" + strings.Repeat("a", 1100) + "'",
	} {
		if s, a, err := ParseSSHCommand(cmd); err == nil {
			t.Fatalf("comando aceito: %q → %q %q", cmd, s, a)
		}
	}
}

// A real pack (made by git) is read to its end and nothing more.
func TestSkipPack(t *testing.T) {
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.Init("r.git", "main")
	big := make([]byte, 300<<10)
	rand.Read(big)
	s.CommitFiles("r.git", "main", "", "um", Signature{Name: "A", Email: "a@x"}, []Action{{Kind: "create", Path: "a.bin", Content: big}})
	id, _ := s.CommitFiles("r.git", "main", "", "dois", Signature{Name: "A", Email: "a@x"}, []Action{{Kind: "create", Path: "b.txt", Content: bytes.Repeat([]byte("abc\n"), 5000)}})
	p, _ := s.Path("r.git")
	cmd := exec.Command(s.Bin, "--git-dir", p, "pack-objects", "--stdout", "--revs", "--delta-base-offset")
	cmd.Stdin = strings.NewReader(id + "\n")
	pack, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	br := bufio.NewReader(io.MultiReader(bytes.NewReader(pack), strings.NewReader("DEPOIS")))
	if err := skipPack(br); err != nil {
		t.Fatal(err)
	}
	if rest, _ := io.ReadAll(br); string(rest) != "DEPOIS" {
		t.Fatalf("leu além do pacote: %q", rest)
	}
	if err := skipPack(bufio.NewReader(strings.NewReader("PACX\x00\x00\x00\x02\x00\x00\x00\x01"))); err == nil {
		t.Fatal("pacote inválido aceito")
	}
	if err := skipPack(bufio.NewReader(bytes.NewReader(pack[:len(pack)/2]))); err == nil {
		t.Fatal("pacote cortado aceito")
	}
}

// The host key is created once with 0600, read back the same, and refused
// when others can read it.
func TestHostKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "novo", "chave")
	k1, err := LoadOrCreateHostKey(path)
	if err != nil {
		t.Fatal(err)
	}
	st, _ := os.Stat(path)
	dir, _ := os.Stat(filepath.Dir(path))
	if st.Mode().Perm() != 0o600 || dir.Mode().Perm() != 0o700 {
		t.Fatalf("permissões: %o %o", st.Mode().Perm(), dir.Mode().Perm())
	}
	k2, err := LoadOrCreateHostKey(path)
	if err != nil || !bytes.Equal(k1.PublicKey().Marshal(), k2.PublicKey().Marshal()) {
		t.Fatalf("a chave mudou ao ler de novo: %v", err)
	}
	os.Chmod(path, 0o644)
	if _, err := LoadOrCreateHostKey(path); err == nil || !strings.Contains(err.Error(), "chmod 600") {
		t.Fatalf("chave legível por outros aceita: %v", err)
	}
}

func testSSHServer(t *testing.T, srv *SSHServer) (string, ssh.Signer) {
	t.Helper()
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store.Init("r.git", "main")
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	person, _ := ssh.NewSignerFromKey(priv)
	_, hpriv, _ := ed25519.GenerateKey(rand.Reader)
	host, _ := ssh.NewSignerFromKey(hpriv)
	srv.Store = store
	srv.Identify = func(k ssh.PublicKey) string {
		if bytes.Equal(k.Marshal(), person.PublicKey().Marshal()) {
			return "1"
		}
		return ""
	}
	srv.Authorize = func(ctx context.Context, who, remote, service, address string) (*SSHAccess, error) {
		if address != "r" {
			return nil, errors.New("não encontrado")
		}
		return &SSHAccess{Repo: "r.git"}, nil
	}
	addr, err := srv.Listen("127.0.0.1:0", host)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { srv.Close() })
	return addr.String(), person
}

// Only sessions with a git command: forwarding, terminals and environment
// are refused; a connection that does not finish the handshake is closed.
func TestSSHServerRecusa(t *testing.T) {
	addr, person := testSSHServer(t, &SSHServer{HandshakeTimeout: 300 * time.Millisecond, MaxPerAddr: 2})
	conn, err := ssh.Dial("tcp", addr, &ssh.ClientConfig{User: "git", Auth: []ssh.AuthMethod{ssh.PublicKeys(person)}, HostKeyCallback: ssh.InsecureIgnoreHostKey()})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, _, err := conn.OpenChannel("direct-tcpip", ssh.Marshal(struct {
		Host     string
		Port     uint32
		OrigHost string
		OrigPort uint32
	}{"127.0.0.1", 22, "127.0.0.1", 1})); err == nil {
		t.Fatal("encaminhamento aceito")
	}
	if ok, _, _ := conn.SendRequest("tcpip-forward", true, ssh.Marshal(struct {
		Addr string
		Port uint32
	}{"127.0.0.1", 0})); ok {
		t.Fatal("encaminhamento remoto aceito")
	}
	sess, _ := conn.NewSession()
	if err := sess.Setenv("GIT_PROTOCOL", "x"); err == nil {
		t.Fatal("variável de ambiente aceita")
	}
	if err := sess.RequestPty("xterm", 24, 80, ssh.TerminalModes{}); err == nil {
		t.Fatal("terminal aceito")
	}
	if err := sess.RequestSubsystem("sftp"); err == nil {
		t.Fatal("sftp aceito")
	}
	sess.Close()

	// a session that never asks for a command is closed
	sess, _ = conn.NewSession()
	done := make(chan error, 1)
	stdout, _ := sess.StdoutPipe()
	go func() { _, err := io.Copy(io.Discard, stdout); done <- err }() // ends when the server closes it
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("sessão sem comando continua aberta")
	}

	// a connection that never speaks SSH is closed after the handshake time
	raw, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	raw.SetReadDeadline(time.Now().Add(5 * time.Second))
	start := time.Now()
	io.Copy(io.Discard, raw)
	if time.Since(start) > 4*time.Second {
		t.Fatal("conexão muda não foi fechada")
	}
}

// Connections from one address are limited: over the limit, the next one
// is closed before the handshake.
func TestSSHServerLimite(t *testing.T) {
	addr, _ := testSSHServer(t, &SSHServer{MaxPerAddr: 1, HandshakeTimeout: 3 * time.Second})
	first, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	first.SetReadDeadline(time.Now().Add(2 * time.Second))
	banner := make([]byte, 4)
	if _, err := io.ReadFull(first, banner); err != nil || string(banner) != "SSH-" {
		t.Fatalf("primeira conexão: %q %v", banner, err)
	}
	second, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	second.SetReadDeadline(time.Now().Add(2 * time.Second))
	if n, err := second.Read(banner); err == nil && n > 0 {
		t.Fatalf("conexão acima do limite foi atendida: %q", banner[:n])
	}
}
