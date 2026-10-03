package git

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flaviokalleu/germanio/runtime/httpclient"
)

// Mirrors: a repository keeps a copy of itself in another Git server (push)
// or keeps itself a copy of another one (fetch). Only http and https remotes
// are accepted, and git never opens a connection by itself: every byte goes
// through a proxy inside this process whose dialer is the SSRF-protected one
// of runtime/httpclient (local networks refused unless
// GERMANIO_PERMITIR_REDE_LOCAL=1). Credentials travel in an Authorization
// header given to git through its environment — never in the command line,
// never in the URL kept by the application, never in an error message.

// Remote is an external repository reached over HTTP(S).
type Remote struct {
	URL      string // http(s) address without credentials
	User     string
	Password string
	// MaxBytes bounds the bytes moved in one operation, both ways (0 = 2 GiB,
	// the same bound as a push received by this server).
	MaxBytes int64
}

// ErrRemoteURL reports an address a mirror cannot use.
type ErrRemoteURL struct{ Reason string }

func (e *ErrRemoteURL) Error() string { return e.Reason }

// ParseRemoteURL validates a mirror address and splits the credentials out
// of it: https://ana:segredo@exemplo.com/a.git → https://exemplo.com/a.git,
// "ana", "segredo". Only http and https are accepted (file, ssh and git's
// "ext::" transports would reach the server itself), and a local address is
// refused unless GERMANIO_PERMITIR_REDE_LOCAL=1.
func ParseRemoteURL(raw string) (clean, user, password string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.ContainsAny(raw, " \t\r\n\x00") || strings.HasPrefix(raw, "-") {
		return "", "", "", &ErrRemoteURL{"endereço inválido: escreva um endereço completo, como https://exemplo.com/grupo/projeto.git"}
	}
	u, perr := url.Parse(raw)
	if perr != nil || u.Host == "" || u.Opaque != "" {
		return "", "", "", &ErrRemoteURL{"endereço inválido: escreva um endereço completo, como https://exemplo.com/grupo/projeto.git"}
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return "", "", "", &ErrRemoteURL{fmt.Sprintf("o endereço usa %q, mas um espelho só fala http ou https (outros transportes poderiam ler arquivos deste servidor): use https://…", u.Scheme)}
	}
	if u.Fragment != "" {
		return "", "", "", &ErrRemoteURL{"o endereço não pode ter # (fragmento)"}
	}
	if !httpclient.LocalAllowed() && httpclient.LocalHost(u.Hostname()) {
		return "", "", "", &ErrRemoteURL{httpclient.ErrRedeLocal.Error()}
	}
	if u.User != nil {
		user = u.User.Username()
		password, _ = u.User.Password()
		u.User = nil
	}
	return u.String(), user, password, nil
}

// PushMirror sends every branch and tag of repository rel to the remote,
// replacing what differs and removing branches and tags that no longer exist
// here (a mirror is an exact copy).
func (s *Store) PushMirror(ctx context.Context, rel string, r Remote) error {
	p, err := s.open(rel)
	if err != nil {
		return err
	}
	_, err = s.remote(ctx, p, r, "push", "--porcelain", "--prune", "--quiet", "--", r.URL,
		"+refs/heads/*:refs/heads/*", "+refs/tags/*:refs/tags/*")
	return err
}

// FetchMirror makes the branches and tags of repository rel an exact copy of
// the remote's. It reports the ref updates it made.
func (s *Store) FetchMirror(ctx context.Context, rel string, r Remote) ([]RefUpdate, error) {
	p, err := s.open(rel)
	if err != nil {
		return nil, err
	}
	before, err := s.refMap(p)
	if err != nil {
		return nil, err
	}
	if _, err := s.remote(ctx, p, r, "fetch", "--prune", "--quiet", "--no-write-fetch-head", "--", r.URL,
		"+refs/heads/*:refs/heads/*", "+refs/tags/*:refs/tags/*"); err != nil {
		return nil, err
	}
	after, err := s.refMap(p)
	if err != nil {
		return nil, err
	}
	var out []RefUpdate
	for ref, id := range after {
		if old, ok := before[ref]; !ok {
			out = append(out, RefUpdate{Old: ZeroID, New: id, Ref: ref})
		} else if old != id {
			out = append(out, RefUpdate{Old: old, New: id, Ref: ref})
		}
	}
	for ref, id := range before {
		if _, ok := after[ref]; !ok {
			out = append(out, RefUpdate{Old: id, New: ZeroID, Ref: ref})
		}
	}
	return out, nil
}

func (s *Store) refMap(repo string) (map[string]string, error) {
	out, err := s.run(repo, nil, nil, "for-each-ref", "--format=%(objectname) %(refname)", "refs/heads/", "refs/tags/")
	if err != nil {
		return nil, err
	}
	m := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if id, ref, ok := strings.Cut(line, " "); ok {
			m[ref] = id
		}
	}
	return m, nil
}

// remote runs a git command that talks to r through the guarded proxy.
func (s *Store) remote(ctx context.Context, repo string, r Remote, args ...string) ([]byte, error) {
	if _, _, _, err := ParseRemoteURL(r.URL); err != nil {
		return nil, err
	}
	max := r.MaxBytes
	if max <= 0 {
		max = 2 << 30
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.Timeout*5)
		defer cancel()
	}
	px, err := startMirrorProxy(max)
	if err != nil {
		return nil, err
	}
	defer px.Close()
	full := []string{"--git-dir", repo,
		"-c", "http.proxy=http://" + px.Addr(),
		"-c", "http.followRedirects=false",
		"-c", "credential.helper=",
		"-c", "core.askPass=",
		"-c", "transfer.fsckObjects=true",
		"-c", "http.lowSpeedLimit=1000",
		"-c", "http.lowSpeedTime=60",
	}
	full = append(full, args...)
	cmd := exec.CommandContext(ctx, s.Bin, full...)
	// GIT_ALLOW_PROTOCOL: nothing but http(s), even for submodules or
	// redirects; the proxy variables are cleared so nothing bypasses ours.
	env := append(baseEnv(), "GIT_ALLOW_PROTOCOL=http:https", "GIT_ASKPASS=", "SSH_ASKPASS=",
		"http_proxy=", "https_proxy=", "HTTP_PROXY=", "HTTPS_PROXY=", "ALL_PROXY=", "all_proxy=", "NO_PROXY=", "no_proxy=")
	secret := ""
	if r.User != "" || r.Password != "" {
		secret = base64.StdEncoding.EncodeToString([]byte(r.User + ":" + r.Password))
		env = append(env, "GIT_CONFIG_COUNT=1", "GIT_CONFIG_KEY_0=http.extraHeader", "GIT_CONFIG_VALUE_0=Authorization: Basic "+secret)
	}
	cmd.Env = env
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	runErr := cmd.Run()
	if runErr == nil {
		return out.Bytes(), nil
	}
	msg := strings.TrimSpace(errb.String() + "\n" + out.String())
	if perr := px.Err(); perr != nil {
		msg = perr.Error() // the real reason (refused address, limit) beats git's "proxy" message
	}
	if ctx.Err() != nil {
		msg = "o tempo limite acabou antes de terminar"
	}
	return nil, errors.New(scrub(msg, r, secret))
}

// scrub removes credentials from a message and keeps it short.
func scrub(msg string, r Remote, secret string) string {
	for _, s := range []string{secret, r.Password, url.QueryEscape(r.Password), url.PathEscape(r.Password)} {
		if len(s) >= 3 {
			msg = strings.ReplaceAll(msg, s, "*****")
		}
	}
	if r.User != "" && r.Password != "" {
		msg = strings.ReplaceAll(msg, r.User+":", "*****:")
	}
	msg = strings.Join(strings.Fields(msg), " ")
	if len(msg) > 1000 {
		msg = msg[:1000] + "…"
	}
	if msg == "" {
		msg = "o git terminou com erro"
	}
	return msg
}

// mirrorProxy is a forward proxy on 127.0.0.1 used by one git command. It
// connects only through httpclient's guarded dialer and stops the transfer
// when the byte budget runs out.
type mirrorProxy struct {
	ln     net.Listener
	srv    *http.Server
	dial   func(ctx context.Context, network, addr string) (net.Conn, error)
	rt     http.RoundTripper
	budget atomic.Int64
	mu     sync.Mutex
	err    error
	conns  map[net.Conn]struct{}
}

var errBudget = errors.New("a transferência passou do limite de tamanho permitido")

func startMirrorProxy(max int64) (*mirrorProxy, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	tr := httpclient.Transport()
	tr.ResponseHeaderTimeout = 2 * time.Minute
	p := &mirrorProxy{ln: ln, dial: tr.DialContext, rt: tr, conns: map[net.Conn]struct{}{}}
	p.budget.Store(max)
	p.srv = &http.Server{Handler: p, ReadHeaderTimeout: 30 * time.Second}
	go p.srv.Serve(ln)
	return p, nil
}

func (p *mirrorProxy) Addr() string { return p.ln.Addr().String() }

// Err is the first refusal the proxy met (blocked address, budget).
func (p *mirrorProxy) Err() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.err
}

func (p *mirrorProxy) fail(err error) {
	p.mu.Lock()
	if p.err == nil {
		p.err = err
	}
	p.mu.Unlock()
}

func (p *mirrorProxy) Close() {
	p.srv.Close()
	p.mu.Lock()
	for c := range p.conns {
		c.Close()
	}
	p.mu.Unlock()
}

func (p *mirrorProxy) track(c net.Conn) {
	p.mu.Lock()
	p.conns[c] = struct{}{}
	p.mu.Unlock()
}

// reason keeps the refusal of the guarded dialer readable.
func (p *mirrorProxy) reason(err error) error {
	if errors.Is(err, httpclient.ErrRedeLocal) {
		return httpclient.ErrRedeLocal
	}
	return fmt.Errorf("não foi possível conectar: %v", err)
}

var hopHeaders = []string{"Connection", "Proxy-Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization", "Te", "Trailer", "Transfer-Encoding", "Upgrade"}

func (p *mirrorProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.tunnel(w, r)
		return
	}
	if r.URL.Scheme != "http" || r.URL.Host == "" {
		http.Error(w, "proxy: só pedidos http absolutos", http.StatusBadRequest)
		return
	}
	out := r.Clone(r.Context())
	out.RequestURI = ""
	for _, h := range hopHeaders {
		out.Header.Del(h)
	}
	if r.Body != nil {
		out.Body = &budgetReader{r: r.Body, p: p}
	}
	resp, err := p.rt.RoundTrip(out)
	if err != nil {
		p.fail(p.reason(err))
		http.Error(w, "proxy: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	for _, h := range hopHeaders {
		resp.Header.Del(h)
	}
	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, &budgetReader{r: resp.Body, p: p})
}

func (p *mirrorProxy) tunnel(w http.ResponseWriter, r *http.Request) {
	up, err := p.dial(r.Context(), "tcp", r.Host)
	if err != nil {
		p.fail(p.reason(err))
		http.Error(w, "proxy: "+err.Error(), http.StatusBadGateway)
		return
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		up.Close()
		http.Error(w, "proxy: sem túnel", http.StatusInternalServerError)
		return
	}
	down, buf, err := hj.Hijack()
	if err != nil {
		up.Close()
		return
	}
	p.track(up)
	p.track(down)
	down.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
	done := make(chan struct{}, 2)
	go func() {
		io.Copy(up, &budgetReader{r: io.MultiReader(buf.Reader, down), p: p})
		done <- struct{}{}
	}()
	go func() {
		io.Copy(down, &budgetReader{r: up, p: p})
		done <- struct{}{}
	}()
	<-done
	up.Close()
	down.Close()
}

// budgetReader stops reading once the proxy's byte budget is spent.
type budgetReader struct {
	r io.Reader
	p *mirrorProxy
}

func (b *budgetReader) Read(buf []byte) (int, error) {
	if b.p.budget.Load() <= 0 {
		b.p.fail(errBudget)
		return 0, errBudget
	}
	n, err := b.r.Read(buf)
	if b.p.budget.Add(-int64(n)) < 0 {
		b.p.fail(errBudget)
		return n, errBudget
	}
	return n, err
}

func (b *budgetReader) Close() error {
	if c, ok := b.r.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
