package git

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
)

// The git protocol over one two-way stream (what SSH carries): the client
// and git talk directly, except that a push's ref updates are read first so
// the rules can refuse them before anything is written, exactly as on smart
// HTTP.

// ServeUploadPack serves a clone or fetch of repository rel over a stream.
func (s *Store) ServeUploadPack(ctx context.Context, rel string, in io.Reader, out io.Writer) error {
	p, err := s.open(rel)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, s.Timeout*5)
	defer cancel()
	cmd := exec.CommandContext(ctx, s.Bin, "upload-pack", "--strict", p)
	return runStream(cmd, in, out)
}

// ServeReceivePack serves a push to repository rel over a stream. check
// receives the requested updates before git reads the pack and may refuse
// them; its message reaches the client as git's own refusal. It returns the
// updates git actually applied.
func (s *Store) ServeReceivePack(ctx context.Context, rel string, in io.Reader, out io.Writer, check func([]RefUpdate) error) ([]RefUpdate, error) {
	p, err := s.open(rel)
	if err != nil {
		return nil, err
	}
	adv, err := s.gitService(p, "receive-pack", nil, true)
	if err != nil {
		return nil, err
	}
	if _, err := out.Write(adv); err != nil {
		return nil, err
	}
	br := bufio.NewReaderSize(io.LimitReader(in, MaxPush), 64<<10)
	updates, caps, head, err := readCommands(br)
	if err != nil {
		if head.Len() == 0 && errors.Is(err, io.EOF) {
			return nil, nil // the client only wanted the refs
		}
		return nil, err
	}
	if len(updates) == 0 {
		return nil, nil // nothing to send
	}
	if check != nil {
		if reason := check(updates); reason != nil {
			// the client sends the pack before reading the answer: read it
			// (and throw it away) so it is not blocked writing
			if needsPack(updates) {
				if err := skipPack(br); err != nil {
					return nil, err
				}
			}
			writeRefusal(out, updates, caps, reason.Error())
			return nil, nil
		}
	}
	ctx, cancel := context.WithTimeout(ctx, s.Timeout*5)
	defer cancel()
	cmd := exec.CommandContext(ctx, s.Bin, "receive-pack", "--stateless-rpc", p)
	if err := runStream(cmd, io.MultiReader(head, br), out); err != nil {
		return nil, err
	}
	return s.applied(p, updates), nil
}

// runStream runs git with in as its input and out as its output. The
// input is copied by hand: the client keeps its side open while it waits
// for the answer, so the command must not wait for the input to end.
func runStream(cmd *exec.Cmd, in io.Reader, out io.Writer) error {
	cmd.Env = baseEnv()
	cmd.Stdout = out
	var errb bytes.Buffer
	cmd.Stderr = &limitedBuffer{buf: &errb, max: 8 << 10}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		io.Copy(stdin, in)
		stdin.Close()
	}()
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("git %s: %v %s", cmd.Args[1], err, strings.TrimSpace(errb.String()))
	}
	return nil
}

// limitedBuffer keeps at most max bytes of what is written to it.
type limitedBuffer struct {
	buf *bytes.Buffer
	max int
}

func (l *limitedBuffer) Write(p []byte) (int, error) {
	if room := l.max - l.buf.Len(); room > 0 {
		l.buf.Write(p[:min(len(p), room)])
	}
	return len(p), nil
}

// needsPack: a push that creates or moves a ref carries a pack.
func needsPack(updates []RefUpdate) bool {
	for _, u := range updates {
		if u.New != ZeroID {
			return true
		}
	}
	return false
}

// skipPack reads one pack (format 2 or 3) from br and throws it away. The
// pack does not say its length: every object is walked (its header, the
// base of a delta, the compressed data), then the trailing checksum.
func skipPack(br *bufio.Reader) error {
	var hdr [12]byte
	if _, err := io.ReadFull(br, hdr[:]); err != nil {
		return err
	}
	if string(hdr[:4]) != "PACK" {
		return fmt.Errorf("pacote inválido")
	}
	if v := binary.BigEndian.Uint32(hdr[4:8]); v != 2 && v != 3 {
		return fmt.Errorf("pacote inválido")
	}
	count := binary.BigEndian.Uint32(hdr[8:12])
	var zr io.ReadCloser
	for i := uint32(0); i < count; i++ {
		c, err := br.ReadByte()
		if err != nil {
			return err
		}
		kind := (c >> 4) & 7
		for c&0x80 != 0 {
			if c, err = br.ReadByte(); err != nil {
				return err
			}
		}
		switch kind {
		case 6: // delta against an earlier offset
			for {
				if c, err = br.ReadByte(); err != nil {
					return err
				}
				if c&0x80 == 0 {
					break
				}
			}
		case 7: // delta against an object id
			if _, err := br.Discard(20); err != nil {
				return err
			}
		case 0, 5:
			return fmt.Errorf("pacote inválido")
		}
		// bufio.Reader is a ByteReader: zlib reads exactly the object
		if zr == nil {
			if zr, err = zlib.NewReader(br); err != nil {
				return err
			}
		} else if err := zr.(zlib.Resetter).Reset(br, nil); err != nil {
			return err
		}
		if _, err := io.Copy(io.Discard, zr); err != nil {
			return err
		}
	}
	_, err := br.Discard(20)
	return err
}

// Commands a client may ask for over SSH: only these two, never a shell.
var sshCommandRe = regexp.MustCompile(`^git[- ](upload-pack|receive-pack) +(?:'([^']*)'|([^'\s]+))$`)

// sshSegmentRe is one segment of a repository address: letters, digits,
// "_", "-" and ".", never starting with "." or "-".
var sshSegmentRe = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._-]*$`)

// ParseSSHCommand reads the command a Git client sends over SSH
// ("git-upload-pack '/grupo/projeto.git'") and returns the service
// (upload-pack or receive-pack) and the repository address without the
// leading slash and the ".git" suffix. Anything else is refused: other
// commands, options, quotes, spaces, empty or "."/".." segments.
func ParseSSHCommand(command string) (service, address string, err error) {
	m := sshCommandRe.FindStringSubmatch(strings.TrimSpace(command))
	if m == nil || len(command) > 1100 {
		return "", "", &ErrInvalid{"comando", command}
	}
	address = m[2] + m[3]
	address = strings.TrimPrefix(address, "/")
	address = strings.TrimSuffix(address, "/")
	address = strings.TrimSuffix(address, ".git")
	if address == "" || len(address) > 1024 || strings.Contains(address, "..") {
		return "", "", &ErrInvalid{"repositório", address}
	}
	for _, seg := range strings.Split(address, "/") {
		if !sshSegmentRe.MatchString(seg) {
			return "", "", &ErrInvalid{"repositório", address}
		}
	}
	return m[1], address, nil
}
