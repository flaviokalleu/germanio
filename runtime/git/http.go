package git

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
)

// RefUpdate is one ref change requested by a push.
type RefUpdate struct {
	Old, New, Ref string
}

// Kind returns criar, excluir or atualizar.
func (u RefUpdate) Kind() string {
	switch {
	case u.Old == ZeroID:
		return "criar"
	case u.New == ZeroID:
		return "excluir"
	}
	return "atualizar"
}

// ServiceFromRequest identifies the smart-HTTP service of a request:
// "upload-pack" (clone/fetch) or "receive-pack" (push), and whether it is
// the ref advertisement.
func ServiceFromRequest(r *http.Request, sub string) (service string, advertise bool, ok bool) {
	switch {
	case r.Method == http.MethodGet && sub == "info/refs":
		s := r.URL.Query().Get("service")
		if s == "git-upload-pack" || s == "git-receive-pack" {
			return strings.TrimPrefix(s, "git-"), true, true
		}
	case r.Method == http.MethodPost && sub == "git-upload-pack":
		return "upload-pack", false, true
	case r.Method == http.MethodPost && sub == "git-receive-pack":
		return "receive-pack", false, true
	}
	return "", false, false
}

func pktLine(s string) []byte { return []byte(fmt.Sprintf("%04x%s", len(s)+4, s)) }

// ServeHTTP answers one smart-HTTP request for repository rel. For pushes,
// check receives the requested updates before anything is written and may
// refuse them (its error message is shown by the git client).
func (s *Store) ServeHTTP(w http.ResponseWriter, r *http.Request, rel, service string, advertise bool, check func([]RefUpdate) error) ([]RefUpdate, error) {
	p, err := s.open(rel)
	if err != nil {
		return nil, err
	}
	w.Header().Set("Cache-Control", "no-cache")
	if advertise {
		w.Header().Set("Content-Type", "application/x-git-"+service+"-advertisement")
		var buf bytes.Buffer
		buf.Write(pktLine("# service=git-" + service + "\n"))
		buf.WriteString("0000")
		out, err := s.gitService(p, service, nil, true)
		if err != nil {
			return nil, err
		}
		buf.Write(out)
		w.Write(buf.Bytes())
		return nil, nil
	}
	var body io.Reader = r.Body
	if r.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		body = gz
	}
	body = io.LimitReader(body, MaxPush)
	var updates []RefUpdate
	if service == "receive-pack" {
		br := bufio.NewReader(body)
		var head *bytes.Buffer
		var caps string
		var err error
		updates, caps, head, err = readCommands(br)
		if err != nil {
			return nil, err
		}
		if check != nil {
			if err := check(updates); err != nil {
				w.Header().Set("Content-Type", "application/x-git-receive-pack-result")
				writeRefusal(w, updates, caps, err.Error())
				return nil, nil
			}
		}
		body = io.MultiReader(head, br)
	}
	w.Header().Set("Content-Type", "application/x-git-"+service+"-result")
	ctx, cancel := context.WithTimeout(r.Context(), s.Timeout*5)
	defer cancel()
	cmd := exec.CommandContext(ctx, s.Bin, service, "--stateless-rpc", p)
	cmd.Env = baseEnv()
	cmd.Stdin = body
	cmd.Stdout = w
	var errb bytes.Buffer
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %v %s", service, err, errb.String())
	}
	return s.applied(p, updates), nil
}

// MaxPush bounds what one push may send (commands and pack).
const MaxPush = 2 << 30

// readCommands reads the ref updates a push asks for, up to the flush that
// ends them. head keeps the raw bytes read, so git can read them again.
func readCommands(br *bufio.Reader) (updates []RefUpdate, caps string, head *bytes.Buffer, err error) {
	head = &bytes.Buffer{}
	for {
		line, err := readPkt(br, head)
		if err != nil {
			return nil, "", head, err
		}
		if line == nil {
			return updates, caps, head, nil // flush: end of commands
		}
		cmd := string(line)
		if i := strings.IndexByte(cmd, 0); i >= 0 {
			caps = cmd[i+1:]
			cmd = cmd[:i]
		}
		f := strings.Fields(strings.TrimSpace(cmd))
		if len(f) == 3 {
			updates = append(updates, RefUpdate{Old: f[0], New: f[1], Ref: f[2]})
		}
	}
}

// applied keeps only the updates git actually made.
func (s *Store) applied(p string, updates []RefUpdate) []RefUpdate {
	var out []RefUpdate
	for _, u := range updates {
		cur, err := s.run(p, nil, nil, "rev-parse", "--verify", "--quiet", u.Ref)
		now := strings.TrimSpace(string(cur))
		if (u.New == ZeroID && err != nil) || now == u.New {
			out = append(out, u)
		}
	}
	return out
}

// readPkt reads one pkt-line, copying its raw bytes to raw. nil = flush.
func readPkt(br *bufio.Reader, raw *bytes.Buffer) ([]byte, error) {
	hdr := make([]byte, 4)
	if _, err := io.ReadFull(br, hdr); err != nil {
		return nil, err
	}
	raw.Write(hdr)
	n, err := strconv.ParseUint(string(hdr), 16, 16)
	if err != nil {
		return nil, fmt.Errorf("pkt-line inválido")
	}
	if n == 0 {
		return nil, nil
	}
	if n < 4 {
		return nil, fmt.Errorf("pkt-line inválido")
	}
	data := make([]byte, n-4)
	if _, err := io.ReadFull(br, data); err != nil {
		return nil, err
	}
	raw.Write(data)
	return data, nil
}

// writeRefusal answers a push with "ng" for every ref (report-status),
// using the side band when the client asked for it.
func writeRefusal(w io.Writer, updates []RefUpdate, caps, reason string) {
	var status bytes.Buffer
	status.Write(pktLine("unpack ok\n"))
	reason = strings.ReplaceAll(reason, "\n", " ")
	for _, u := range updates {
		status.Write(pktLine("ng " + u.Ref + " " + reason + "\n"))
	}
	status.WriteString("0000")
	if strings.Contains(caps, "side-band") {
		var out bytes.Buffer
		msg := "\x02" + reason + "\n"
		out.Write(pktLine(msg))
		data := status.Bytes()
		for len(data) > 0 {
			n := min(len(data), 65515)
			out.Write(pktLine("\x01" + string(data[:n])))
			data = data[n:]
		}
		out.WriteString("0000")
		w.Write(out.Bytes())
		return
	}
	w.Write(status.Bytes())
}

func (s *Store) gitService(repo, service string, stdin io.Reader, advertise bool) ([]byte, error) {
	args := []string{service, "--stateless-rpc"}
	if advertise {
		args = append(args, "--advertise-refs")
	}
	args = append(args, repo)
	cmd := exec.Command(s.Bin, args...)
	cmd.Env = baseEnv()
	cmd.Stdin = stdin
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %s", service, errb.String())
	}
	return out.Bytes(), nil
}
