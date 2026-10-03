package git

import (
	"compress/gzip"
	"context"
	"io"
	"os/exec"
	"regexp"
	"strings"
)

// ArchiveFormats are the formats Archive writes, by file extension.
var ArchiveFormats = map[string]bool{"zip": true, "tar": true, "tar.gz": true, "tgz": true}

var prefixRe = regexp.MustCompile(`^[A-Za-z0-9._-]{1,200}/$`)

// Archive streams the tree of rev as one file in format (zip, tar, tar.gz or
// tgz) to w, every path under prefix ("name-ref/"). The revision is resolved
// first, so a bad one is an error before any byte is written. git runs with
// an argument vector (never a shell); gzip is done here, never by a filter
// command configured in git. Nothing is held in memory: bytes go from git to
// w as they are produced.
func (s *Store) Archive(ctx context.Context, rel, rev, format, prefix string, w io.Writer) error {
	if !ArchiveFormats[format] {
		return &ErrInvalid{"formato", format}
	}
	if !prefixRe.MatchString(prefix) || strings.Contains(prefix, "..") || strings.HasPrefix(prefix, "-") {
		return &ErrInvalid{"prefixo", prefix}
	}
	id, err := s.Resolve(rel, rev)
	if err != nil {
		return err
	}
	p, _ := s.Path(rel)
	gitFormat := format
	if format == "tar.gz" || format == "tgz" {
		gitFormat = "tar"
	}
	ctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, s.Bin, "--git-dir", p, "archive", "--format="+gitFormat, "--prefix="+prefix, "--end-of-options", id)
	cmd.Env = baseEnv()
	var errb strings.Builder
	cmd.Stderr = &limitedWriter{w: &errb, n: 4096}
	out := w
	var gz *gzip.Writer
	if gitFormat != format {
		gz = gzip.NewWriter(w)
		out = gz
	}
	cmd.Stdout = out
	if err := cmd.Run(); err != nil {
		return &Error{Args: []string{"archive"}, Stderr: strings.TrimSpace(errb.String()), Err: err}
	}
	if gz != nil {
		return gz.Close()
	}
	return nil
}

// limitedWriter keeps at most n bytes (the stderr of a failing command).
type limitedWriter struct {
	w io.Writer
	n int
}

func (l *limitedWriter) Write(p []byte) (int, error) {
	if l.n > 0 {
		k := min(len(p), l.n)
		l.w.Write(p[:k])
		l.n -= k
	}
	return len(p), nil
}
