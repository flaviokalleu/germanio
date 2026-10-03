package git

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"
)

// Git over SSH (the transport every Git client speaks besides HTTP). The
// server authenticates by public key only, runs only git-upload-pack and
// git-receive-pack, never a shell, and refuses forwarding, agents and
// terminals. Who owns a key and what that person may do are decided by
// the callbacks: this file knows Git and SSH, nothing about applications.

// SSHAccess is what a person may do with the repository they asked for.
type SSHAccess struct {
	// Repo is the repository's path in the store.
	Repo string
	// Check receives a push's updates before anything is written; an error
	// refuses the push and its message is shown by the client.
	Check func([]RefUpdate) error
	// After runs once a push changed refs, with the updates applied.
	After func([]RefUpdate)
}

// SSHServer serves the repositories of a Store over SSH.
type SSHServer struct {
	Store *Store
	// Identify returns who owns key, "" when nobody does (refused).
	Identify func(key ssh.PublicKey) string
	// Authorize decides a git command of who, coming from remote, for the
	// repository at address. Its error is shown to the person as it is.
	Authorize func(ctx context.Context, who, remote, service, address string) (*SSHAccess, error)
	// Greet is the text shown to who when they open a session without a
	// command (ssh -T), so people can check their key.
	Greet func(who string) string
	// MaxConns and MaxPerAddr bound the connections served at once, in all
	// and from one address.
	MaxConns, MaxPerAddr int
	// HandshakeTimeout bounds the time until a command is asked for;
	// IdleTimeout closes a connection where nothing moves.
	HandshakeTimeout, IdleTimeout time.Duration

	config  *ssh.ServerConfig
	ln      net.Listener
	mu      sync.Mutex
	conns   map[net.Conn]bool
	perAddr map[string]int
	closed  bool
	wg      sync.WaitGroup
}

// Listen starts serving on addr ("host:port"; port 0 picks a free one) with
// hostKey, and returns the address actually bound.
func (s *SSHServer) Listen(addr string, hostKey ssh.Signer) (net.Addr, error) {
	if s.MaxConns <= 0 {
		s.MaxConns = 64
	}
	if s.MaxPerAddr <= 0 {
		s.MaxPerAddr = 8
	}
	if s.HandshakeTimeout <= 0 {
		s.HandshakeTimeout = 15 * time.Second
	}
	if s.IdleTimeout <= 0 {
		s.IdleTimeout = 2 * time.Minute
	}
	s.config = &ssh.ServerConfig{
		ServerVersion: "SSH-2.0-Germanio",
		MaxAuthTries:  6,
		// public keys only: no password, no keyboard-interactive
		PublicKeyCallback: func(meta ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			who := s.Identify(key)
			if who == "" {
				return nil, errors.New("chave desconhecida")
			}
			return &ssh.Permissions{Extensions: map[string]string{"pessoa": who}}, nil
		},
	}
	s.config.AddHostKey(hostKey)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	s.ln = ln
	s.conns = map[net.Conn]bool{}
	s.perAddr = map[string]int{}
	s.wg.Add(1)
	go s.accept()
	return ln.Addr(), nil
}

// Close stops accepting, closes the open connections and waits for them.
func (s *SSHServer) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	err := s.ln.Close()
	for c := range s.conns {
		c.Close()
	}
	s.mu.Unlock()
	s.wg.Wait()
	return err
}

func (s *SSHServer) accept() {
	defer s.wg.Done()
	for {
		c, err := s.ln.Accept()
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			return
		}
		host, _, _ := net.SplitHostPort(c.RemoteAddr().String())
		s.mu.Lock()
		if s.closed || len(s.conns) >= s.MaxConns || s.perAddr[host] >= s.MaxPerAddr {
			s.mu.Unlock()
			c.Close() // over the limit: the client retries later
			continue
		}
		s.conns[c] = true
		s.perAddr[host]++
		s.wg.Add(1)
		s.mu.Unlock()
		go func() {
			defer s.wg.Done()
			defer func() {
				c.Close()
				s.mu.Lock()
				delete(s.conns, c)
				if s.perAddr[host]--; s.perAddr[host] <= 0 {
					delete(s.perAddr, host)
				}
				s.mu.Unlock()
			}()
			s.serveConn(&idleConn{Conn: c, idle: s.IdleTimeout})
		}()
	}
}

// idleConn closes a connection where nothing is read or written for idle;
// any traffic, in either direction, keeps it open. Until the handshake
// ends, the deadline never goes past hard.
type idleConn struct {
	net.Conn
	idle time.Duration
	hard atomic.Int64 // unix nanoseconds; 0 = none
}

func (c *idleConn) touch() {
	d := time.Now().Add(c.idle)
	if h := c.hard.Load(); h != 0 && time.Unix(0, h).Before(d) {
		d = time.Unix(0, h)
	}
	c.Conn.SetDeadline(d)
}

func (c *idleConn) Read(b []byte) (int, error) {
	c.touch()
	return c.Conn.Read(b)
}

func (c *idleConn) Write(b []byte) (int, error) {
	c.touch()
	return c.Conn.Write(b)
}

func (s *SSHServer) serveConn(c *idleConn) {
	// the handshake (and the authentication) has its own short limit
	c.hard.Store(time.Now().Add(s.HandshakeTimeout).UnixNano())
	conn, chans, reqs, err := ssh.NewServerConn(c, s.config)
	if err != nil {
		return
	}
	c.hard.Store(0)
	defer conn.Close()
	go ssh.DiscardRequests(reqs) // no forwarding, no keepalive answers needed
	who := conn.Permissions.Extensions["pessoa"]
	sessions := 0
	var wg sync.WaitGroup
	for nc := range chans {
		if nc.ChannelType() != "session" {
			nc.Reject(ssh.Prohibited, "este servidor só atende git")
			continue
		}
		if sessions >= 4 {
			nc.Reject(ssh.ResourceShortage, "sessões demais nesta conexão")
			continue
		}
		ch, creqs, err := nc.Accept()
		if err != nil {
			continue
		}
		sessions++
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.serveSession(conn, ch, creqs, who)
		}()
	}
	wg.Wait()
}

func (s *SSHServer) serveSession(conn *ssh.ServerConn, ch ssh.Channel, reqs <-chan *ssh.Request, who string) {
	defer ch.Close()
	timer := time.AfterFunc(s.HandshakeTimeout, func() { ch.Close() })
	defer timer.Stop()
	for req := range reqs {
		switch req.Type {
		case "exec":
			timer.Stop()
			var p struct{ Command string }
			if err := ssh.Unmarshal(req.Payload, &p); err != nil {
				req.Reply(false, nil)
				return
			}
			req.Reply(true, nil)
			go ssh.DiscardRequests(reqs)
			exit(ch, s.run(conn, ch, who, p.Command))
			return
		case "shell":
			timer.Stop()
			req.Reply(true, nil)
			go ssh.DiscardRequests(reqs)
			msg := "Este servidor só atende git (clone, fetch e push); não há terminal aqui.\r\n"
			if s.Greet != nil {
				msg = s.Greet(who) + "\r\n" + msg
			}
			io.WriteString(ch.Stderr(), msg)
			exit(ch, 0)
			return
		default: // env, pty-req, x11, agent forwarding, subsystems: refused
			req.Reply(false, nil)
		}
	}
}

// run serves one git command and returns the exit status.
func (s *SSHServer) run(conn *ssh.ServerConn, ch ssh.Channel, who, command string) uint32 {
	say := func(msg string) { io.WriteString(ch.Stderr(), "germanio: "+msg+"\n") }
	service, address, err := ParseSSHCommand(command)
	if err != nil {
		say("só é possível usar git por aqui (git clone, git fetch, git pull, git push), " +
			"com um endereço como ssh://servidor/grupo/projeto.git; o pedido recebido não é um deles.")
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.Store.Timeout*5)
	defer cancel()
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	acc, err := s.Authorize(ctx, who, conn.RemoteAddr().String(), service, address)
	if err != nil {
		say(err.Error())
		return 1
	}
	if service == "upload-pack" {
		if err := s.Store.ServeUploadPack(ctx, acc.Repo, ch, ch); err != nil {
			say("não foi possível enviar o código; tente de novo.")
			return 1
		}
		return 0
	}
	applied, err := s.Store.ServeReceivePack(ctx, acc.Repo, ch, ch, acc.Check)
	if err != nil {
		say("não foi possível receber o código; tente de novo.")
		return 1
	}
	if len(applied) > 0 && acc.After != nil {
		acc.After(applied)
	}
	return 0
}

func exit(ch ssh.Channel, status uint32) {
	ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{status}))
	ch.CloseWrite()
}

// LoadOrCreateHostKey reads the server's host key at path, or creates one
// (ed25519) the first time, written with permissions 0600 in a folder only
// the owner can enter. A key that others can read is refused.
func LoadOrCreateHostKey(path string) (ssh.Signer, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, err
		}
		_, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}
		block, err := ssh.MarshalPrivateKey(priv, "germanio")
		if err != nil {
			return nil, err
		}
		b = pem.EncodeToMemory(block)
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return nil, err
		}
		if _, err := f.Write(b); err != nil {
			f.Close()
			os.Remove(path)
			return nil, err
		}
		if err := f.Close(); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if st.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("a chave do servidor SSH em %s pode ser lida por outras pessoas (permissões %o).\n"+
			"Por quê: quem lê essa chave pode se passar por este servidor.\n"+
			"Como corrigir: chmod 600 %s", path, st.Mode().Perm(), path)
	}
	signer, err := ssh.ParsePrivateKey(b)
	if err != nil {
		return nil, fmt.Errorf("a chave do servidor SSH em %s não pôde ser lida: %s", path, strings.TrimSpace(err.Error()))
	}
	return signer, nil
}
