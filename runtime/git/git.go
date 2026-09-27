// Package git exposes Git repositories to Germanio programs. It drives the
// git command-line client with argument vectors (never a shell), confines
// every repository to a root directory and validates refs and paths before
// they reach git. It knows about Git only — no application concepts.
package git

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ZeroID is the all-zero object id used for ref creation and deletion.
const ZeroID = "0000000000000000000000000000000000000000"

// ErrNotFound reports a missing repository, ref or path.
var ErrNotFound = errors.New("não encontrado")

// ErrInvalid reports a rejected argument (ref name, path, repository).
type ErrInvalid struct{ What, Value string }

func (e *ErrInvalid) Error() string { return fmt.Sprintf("%s inválido: %q", e.What, e.Value) }

// Store manages bare repositories under Root.
type Store struct {
	Root    string
	Bin     string
	Timeout time.Duration
}

// NewStore creates the root directory and checks that git is available.
func NewStore(root string) (*Store, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, err
	}
	bin, err := exec.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("git não está instalado: %w", err)
	}
	return &Store{Root: abs, Bin: bin, Timeout: 2 * time.Minute}, nil
}

var repoRe = regexp.MustCompile(`^[A-Za-z0-9@._/-]+\.git$`)

// Path validates a repository path relative to Root and returns it absolute.
func (s *Store) Path(rel string) (string, error) {
	if !repoRe.MatchString(rel) || strings.Contains(rel, "..") || strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, "-") {
		return "", &ErrInvalid{"repositório", rel}
	}
	p := filepath.Join(s.Root, filepath.FromSlash(rel))
	if !strings.HasPrefix(p, s.Root+string(os.PathSeparator)) {
		return "", &ErrInvalid{"repositório", rel}
	}
	return p, nil
}

var shaRe = regexp.MustCompile(`^[0-9a-f]{4,64}$`)

// ValidRef accepts branch/tag names and revisions without option-like or
// traversal forms. It follows git-check-ref-format's main rules.
func ValidRef(ref string) bool {
	if ref == "" || len(ref) > 255 || strings.HasPrefix(ref, "-") || strings.HasPrefix(ref, "/") ||
		strings.HasSuffix(ref, "/") || strings.HasSuffix(ref, ".") || strings.HasSuffix(ref, ".lock") ||
		strings.Contains(ref, "..") || strings.Contains(ref, "//") || strings.Contains(ref, "@{") || ref == "@" {
		return false
	}
	for _, r := range ref {
		if r < 0x20 || r == 0x7f || strings.ContainsRune(" ~^:?*[\\", r) {
			return false
		}
	}
	return true
}

func validPath(p string) bool {
	if p == "" {
		return true
	}
	if strings.HasPrefix(p, "/") || strings.HasPrefix(p, "-") || strings.Contains(p, "\x00") || strings.ContainsAny(p, "\r\n") {
		return false
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." || seg == "." || seg == "" || seg == ".git" {
			return false
		}
	}
	return true
}

// run executes git in repo (bare) with a clean environment.
func (s *Store) run(repo string, stdin io.Reader, env []string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.Timeout)
	defer cancel()
	full := append([]string{"--git-dir", repo}, args...)
	cmd := exec.CommandContext(ctx, s.Bin, full...)
	cmd.Env = append(baseEnv(), env...)
	cmd.Stdin = stdin
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		return out.Bytes(), &Error{Args: args, Stderr: strings.TrimSpace(errb.String()), Err: err}
	}
	return out.Bytes(), nil
}

// Error carries git's stderr for diagnostics.
type Error struct {
	Args   []string
	Stderr string
	Err    error
}

func (e *Error) Error() string { return fmt.Sprintf("git %s: %s", e.Args[0], e.Stderr) }

// ExitCode returns git's exit status (-1 when it did not exit normally).
func (e *Error) ExitCode() int {
	var ee *exec.ExitError
	if errors.As(e.Err, &ee) {
		return ee.ExitCode()
	}
	return -1
}

func baseEnv() []string {
	env := []string{"GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0", "LC_ALL=C", "HOME=" + os.TempDir()}
	if p := os.Getenv("PATH"); p != "" {
		env = append(env, "PATH="+p)
	}
	return env
}

// Init creates a bare repository with the given default branch.
func (s *Store) Init(rel, defaultBranch string) error {
	p, err := s.Path(rel)
	if err != nil {
		return err
	}
	if defaultBranch == "" {
		defaultBranch = "main"
	}
	if !ValidRef(defaultBranch) {
		return &ErrInvalid{"branch", defaultBranch}
	}
	if _, err := os.Stat(p); err == nil {
		return fmt.Errorf("repositório já existe: %s", rel)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return err
	}
	cmd := exec.Command(s.Bin, "init", "--bare", "--quiet", "--initial-branch="+defaultBranch, p)
	cmd.Env = baseEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git init: %s", strings.TrimSpace(string(out)))
	}
	// Allow smart-HTTP pushes; git http-backend style services read this.
	_, err = s.run(p, nil, nil, "config", "http.receivepack", "true")
	return err
}

// Exists reports whether the repository exists.
func (s *Store) Exists(rel string) bool {
	p, err := s.Path(rel)
	if err != nil {
		return false
	}
	st, err := os.Stat(filepath.Join(p, "HEAD"))
	return err == nil && !st.IsDir()
}

// Remove deletes the repository directory.
func (s *Store) Remove(rel string) error {
	p, err := s.Path(rel)
	if err != nil {
		return err
	}
	return os.RemoveAll(p)
}

func (s *Store) open(rel string) (string, error) {
	p, err := s.Path(rel)
	if err != nil {
		return "", err
	}
	if !s.Exists(rel) {
		return "", fmt.Errorf("repositório %w: %s", ErrNotFound, rel)
	}
	return p, nil
}

// IsEmpty reports whether the repository has no refs.
func (s *Store) IsEmpty(rel string) (bool, error) {
	p, err := s.open(rel)
	if err != nil {
		return false, err
	}
	out, err := s.run(p, nil, nil, "for-each-ref", "--count=1", "--format=%(refname)")
	if err != nil {
		return false, err
	}
	return len(bytes.TrimSpace(out)) == 0, nil
}

// Signature identifies an author or committer.
type Signature struct {
	Name, Email string
	When        time.Time
}

// Commit is the metadata of a commit.
type Commit struct {
	ID, ShortID, Title, Message       string
	AuthorName, AuthorEmail, AuthorAt string
	CommitterName, CommitterEmail     string
	CommittedAt                       string
	ParentIDs                         []string
}

const commitFormat = "%H%x00%an%x00%ae%x00%aI%x00%cn%x00%ce%x00%cI%x00%P%x00%B%x1e"

func parseCommits(out []byte) []Commit {
	var list []Commit
	for _, rec := range bytes.Split(out, []byte{0x1e}) {
		rec = bytes.TrimLeft(rec, "\n")
		if len(rec) == 0 {
			continue
		}
		f := strings.SplitN(string(rec), "\x00", 9)
		if len(f) < 9 {
			continue
		}
		msg := strings.TrimRight(f[8], "\n")
		title := msg
		if i := strings.IndexByte(msg, '\n'); i >= 0 {
			title = msg[:i]
		}
		c := Commit{ID: f[0], ShortID: f[0][:8], Title: title, Message: msg, AuthorName: f[1], AuthorEmail: f[2], AuthorAt: f[3],
			CommitterName: f[4], CommitterEmail: f[5], CommittedAt: f[6]}
		if f[7] != "" {
			c.ParentIDs = strings.Fields(f[7])
		} else {
			c.ParentIDs = []string{}
		}
		list = append(list, c)
	}
	return list
}

// Resolve returns the commit id a revision points to, or ErrNotFound.
func (s *Store) Resolve(rel, rev string) (string, error) {
	p, err := s.open(rel)
	if err != nil {
		return "", err
	}
	if !ValidRef(rev) && !shaRe.MatchString(rev) {
		return "", &ErrInvalid{"revisão", rev}
	}
	out, err := s.run(p, nil, nil, "rev-parse", "--verify", "--quiet", "--end-of-options", rev+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("revisão %w: %s", ErrNotFound, rev)
	}
	return strings.TrimSpace(string(out)), nil
}

// GetCommit returns one commit.
func (s *Store) GetCommit(rel, rev string) (*Commit, error) {
	id, err := s.Resolve(rel, rev)
	if err != nil {
		return nil, err
	}
	p, _ := s.Path(rel)
	out, err := s.run(p, nil, nil, "log", "-1", "--format="+commitFormat, id)
	if err != nil {
		return nil, err
	}
	list := parseCommits(out)
	if len(list) == 0 {
		return nil, ErrNotFound
	}
	return &list[0], nil
}

// Log lists commits reachable from rev (newest first), optionally only
// those touching path, or the range base..rev when base is set.
func (s *Store) Log(rel, rev, base, path string, limit, skip int) ([]Commit, error) {
	id, err := s.Resolve(rel, rev)
	if err != nil {
		return nil, err
	}
	p, _ := s.Path(rel)
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	args := []string{"log", "--format=" + commitFormat, "-n", strconv.Itoa(limit), "--skip", strconv.Itoa(skip)}
	if base != "" {
		b, err := s.Resolve(rel, base)
		if err != nil {
			return nil, err
		}
		args = append(args, b+".."+id)
	} else {
		args = append(args, id)
	}
	if path != "" {
		if !validPath(path) {
			return nil, &ErrInvalid{"caminho", path}
		}
		args = append(args, "--", path)
	}
	out, err := s.run(p, nil, nil, args...)
	if err != nil {
		return nil, err
	}
	return parseCommits(out), nil
}

// Branch is a branch head.
type Branch struct {
	Name   string
	Commit Commit
}

// Branches lists refs/heads with their head commits.
func (s *Store) Branches(rel string) ([]Branch, error) { return s.refs(rel, "refs/heads/") }

// Tags lists refs/tags pointing at commits.
func (s *Store) Tags(rel string) ([]Branch, error) { return s.refs(rel, "refs/tags/") }

func (s *Store) refs(rel, prefix string) ([]Branch, error) {
	p, err := s.open(rel)
	if err != nil {
		return nil, err
	}
	out, err := s.run(p, nil, nil, "for-each-ref", "--sort=refname", "--format=%(refname)%00%(objectname)", prefix)
	if err != nil {
		return nil, err
	}
	var list []Branch
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		f := strings.SplitN(line, "\x00", 2)
		c, err := s.GetCommit(rel, f[1])
		if err != nil {
			continue // annotated tags on non-commits are skipped
		}
		list = append(list, Branch{Name: strings.TrimPrefix(f[0], prefix), Commit: *c})
	}
	return list, nil
}

// BranchExists reports whether refs/heads/name exists.
func (s *Store) BranchExists(rel, name string) bool {
	p, err := s.open(rel)
	if err != nil || !ValidRef(name) {
		return false
	}
	_, err = s.run(p, nil, nil, "show-ref", "--verify", "--quiet", "refs/heads/"+name)
	return err == nil
}

// UpdateRef moves ref from old to new atomically (compare-and-swap); old
// ZeroID means "must not exist", new ZeroID deletes.
func (s *Store) UpdateRef(rel, ref, newID, oldID string) error {
	p, err := s.open(rel)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(ref, "refs/") || !ValidRef(ref) {
		return &ErrInvalid{"ref", ref}
	}
	if newID == ZeroID {
		_, err = s.run(p, nil, nil, "update-ref", "-d", ref, oldID)
	} else {
		_, err = s.run(p, nil, nil, "update-ref", ref, newID, oldID)
	}
	return err
}

// CreateBranch creates refs/heads/name at from (branch or commit).
func (s *Store) CreateBranch(rel, name, from string) (string, error) {
	if !ValidRef(name) {
		return "", &ErrInvalid{"branch", name}
	}
	id, err := s.Resolve(rel, from)
	if err != nil {
		return "", err
	}
	if s.BranchExists(rel, name) {
		return "", fmt.Errorf("branch já existe: %s", name)
	}
	return id, s.UpdateRef(rel, "refs/heads/"+name, id, ZeroID)
}

// DeleteBranch removes refs/heads/name.
func (s *Store) DeleteBranch(rel, name string) error {
	id, err := s.Resolve(rel, "refs/heads/"+name)
	if err != nil {
		return err
	}
	return s.UpdateRef(rel, "refs/heads/"+name, ZeroID, id)
}

// TreeEntry is an item of a tree listing.
type TreeEntry struct {
	ID, Name, Type, Path, Mode string
	Size                       int64
}

// Tree lists the entries of path at rev.
func (s *Store) Tree(rel, rev, path string) ([]TreeEntry, error) {
	id, err := s.Resolve(rel, rev)
	if err != nil {
		return nil, err
	}
	if !validPath(path) {
		return nil, &ErrInvalid{"caminho", path}
	}
	p, _ := s.Path(rel)
	spec := id
	if path != "" {
		spec = id + ":" + path
	}
	out, err := s.run(p, nil, nil, "ls-tree", "-z", "--long", spec)
	if err != nil {
		return nil, fmt.Errorf("caminho %w: %s", ErrNotFound, path)
	}
	var list []TreeEntry
	for _, rec := range bytes.Split(out, []byte{0}) {
		if len(rec) == 0 {
			continue
		}
		meta, name, ok := strings.Cut(string(rec), "\t")
		if !ok {
			continue
		}
		f := strings.Fields(meta)
		if len(f) < 4 {
			continue
		}
		e := TreeEntry{Mode: f[0], Type: f[1], ID: f[2], Name: name, Path: name}
		if path != "" {
			e.Path = path + "/" + name
		}
		if f[3] != "-" {
			e.Size, _ = strconv.ParseInt(f[3], 10, 64)
		}
		list = append(list, e)
	}
	// Directories first, as repository browsers show them.
	dirs, files := []TreeEntry{}, []TreeEntry{}
	for _, e := range list {
		if e.Type == "tree" {
			dirs = append(dirs, e)
		} else {
			files = append(files, e)
		}
	}
	return append(dirs, files...), nil
}

// Blob is file content at a revision.
type Blob struct {
	ID        string
	Path      string
	Size      int64
	Binary    bool
	Truncated bool
	Content   []byte
}

// ReadFile returns up to maxBytes of path at rev.
func (s *Store) ReadFile(rel, rev, path string, maxBytes int64) (*Blob, error) {
	id, err := s.Resolve(rel, rev)
	if err != nil {
		return nil, err
	}
	if path == "" || !validPath(path) {
		return nil, &ErrInvalid{"caminho", path}
	}
	p, _ := s.Path(rel)
	spec := id + ":" + path
	info, err := s.run(p, strings.NewReader(spec+"\n"), nil, "cat-file", "--batch-check=%(objectname) %(objecttype) %(objectsize)")
	if err != nil {
		return nil, err
	}
	f := strings.Fields(string(info))
	if len(f) < 3 || f[1] != "blob" {
		return nil, fmt.Errorf("arquivo %w: %s", ErrNotFound, path)
	}
	size, _ := strconv.ParseInt(f[2], 10, 64)
	b := &Blob{ID: f[0], Path: path, Size: size}
	content, err := s.run(p, nil, nil, "cat-file", "blob", f[0])
	if err != nil {
		return nil, err
	}
	if maxBytes > 0 && int64(len(content)) > maxBytes {
		content = content[:maxBytes]
		b.Truncated = true
	}
	b.Content = content
	b.Binary = bytes.IndexByte(content[:min(len(content), 8000)], 0) >= 0
	return b, nil
}

// FileDiff is the change of one file between two commits.
type FileDiff struct {
	OldPath, NewPath                          string
	NewFile, DeletedFile, RenamedFile, Binary bool
	Diff                                      string
	Additions, Deletions                      int
}

// Diff compares from..to (from may be empty for a root commit). maxFiles
// bounds the result; truncated reports whether files were omitted.
func (s *Store) Diff(rel, from, to string, maxFiles int) (files []FileDiff, truncated bool, err error) {
	toID, err := s.Resolve(rel, to)
	if err != nil {
		return nil, false, err
	}
	p, _ := s.Path(rel)
	args := []string{"diff", "--no-color", "--no-ext-diff", "-M", "--full-index", "--src-prefix=a/", "--dst-prefix=b/"}
	if from == "" {
		args = []string{"show", "--no-color", "--no-ext-diff", "-M", "--full-index", "--src-prefix=a/", "--dst-prefix=b/", "--format=", toID}
	} else {
		fromID, err := s.Resolve(rel, from)
		if err != nil {
			return nil, false, err
		}
		args = append(args, fromID, toID)
	}
	out, err := s.run(p, nil, nil, args...)
	if err != nil {
		return nil, false, err
	}
	files = parseDiff(out)
	if maxFiles > 0 && len(files) > maxFiles {
		return files[:maxFiles], true, nil
	}
	return files, false, nil
}

func parseDiff(out []byte) []FileDiff {
	var files []FileDiff
	var cur *FileDiff
	var body strings.Builder
	inHunk := false
	flush := func() {
		if cur != nil {
			cur.Diff = body.String()
			files = append(files, *cur)
		}
		body.Reset()
		inHunk = false
	}
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 1024*1024), 64*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "diff --git ") {
			flush()
			cur = &FileDiff{}
			rest := strings.TrimPrefix(line, "diff --git ")
			if i := strings.Index(rest, " b/"); i >= 0 {
				cur.OldPath = strings.TrimPrefix(rest[:i], "a/")
				cur.NewPath = rest[i+3:]
			}
			continue
		}
		if cur == nil {
			continue
		}
		switch {
		case !inHunk && strings.HasPrefix(line, "new file mode"):
			cur.NewFile = true
		case !inHunk && strings.HasPrefix(line, "deleted file mode"):
			cur.DeletedFile = true
		case !inHunk && strings.HasPrefix(line, "rename from "):
			cur.RenamedFile = true
			cur.OldPath = strings.TrimPrefix(line, "rename from ")
		case !inHunk && strings.HasPrefix(line, "rename to "):
			cur.NewPath = strings.TrimPrefix(line, "rename to ")
		case !inHunk && strings.HasPrefix(line, "Binary files "):
			cur.Binary = true
		case !inHunk && (strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "index ") || strings.HasPrefix(line, "similarity index")):
		case strings.HasPrefix(line, "@@"):
			inHunk = true
			body.WriteString(line + "\n")
		case inHunk:
			body.WriteString(line + "\n")
			if strings.HasPrefix(line, "+") {
				cur.Additions++
			} else if strings.HasPrefix(line, "-") {
				cur.Deletions++
			}
		}
	}
	flush()
	return files
}

// MergeBase returns the best common ancestor of a and b.
func (s *Store) MergeBase(rel, a, b string) (string, error) {
	ai, err := s.Resolve(rel, a)
	if err != nil {
		return "", err
	}
	bi, err := s.Resolve(rel, b)
	if err != nil {
		return "", err
	}
	p, _ := s.Path(rel)
	out, err := s.run(p, nil, nil, "merge-base", ai, bi)
	if err != nil {
		return "", fmt.Errorf("sem ancestral comum: %w", ErrNotFound)
	}
	return strings.TrimSpace(string(out)), nil
}

// IsAncestor reports whether a is an ancestor of b.
func (s *Store) IsAncestor(rel, a, b string) (bool, error) {
	ai, err := s.Resolve(rel, a)
	if err != nil {
		return false, err
	}
	bi, err := s.Resolve(rel, b)
	if err != nil {
		return false, err
	}
	p, _ := s.Path(rel)
	_, err = s.run(p, nil, nil, "merge-base", "--is-ancestor", ai, bi)
	if err == nil {
		return true, nil
	}
	var ge *Error
	if errors.As(err, &ge) && ge.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}

// MergeCheck is the result of a trial merge.
type MergeCheck struct {
	CanMerge  bool
	Conflicts []string
	TreeID    string
}

// CheckMerge performs an in-memory merge of source into target
// (git merge-tree --write-tree, no working tree needed).
func (s *Store) CheckMerge(rel, target, source string) (*MergeCheck, error) {
	t, err := s.Resolve(rel, target)
	if err != nil {
		return nil, err
	}
	src, err := s.Resolve(rel, source)
	if err != nil {
		return nil, err
	}
	p, _ := s.Path(rel)
	out, err := s.run(p, nil, nil, "merge-tree", "--write-tree", "--name-only", "--no-messages", t, src)
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if err != nil {
		var ge *Error
		if errors.As(err, &ge) && ge.ExitCode() == 1 {
			var conflicts []string
			for _, l := range lines[1:] {
				if l != "" {
					conflicts = append(conflicts, l)
				}
			}
			return &MergeCheck{CanMerge: false, Conflicts: conflicts}, nil
		}
		return nil, err
	}
	return &MergeCheck{CanMerge: true, TreeID: lines[0], Conflicts: []string{}}, nil
}

func sigEnv(author, committer Signature) []string {
	when := func(t time.Time) string {
		if t.IsZero() {
			t = time.Now()
		}
		return t.Format(time.RFC3339)
	}
	return []string{
		"GIT_AUTHOR_NAME=" + clean(author.Name), "GIT_AUTHOR_EMAIL=" + clean(author.Email), "GIT_AUTHOR_DATE=" + when(author.When),
		"GIT_COMMITTER_NAME=" + clean(committer.Name), "GIT_COMMITTER_EMAIL=" + clean(committer.Email), "GIT_COMMITTER_DATE=" + when(committer.When),
	}
}

func clean(s string) string {
	s = strings.Map(func(r rune) rune {
		if r == '<' || r == '>' || r == '\n' || r == '\r' || r == 0 {
			return -1
		}
		return r
	}, s)
	if strings.TrimSpace(s) == "" {
		return "unknown"
	}
	return s
}

// Merge creates a merge commit of source into target and advances target
// only if it did not move meanwhile. It fails on conflicts.
func (s *Store) Merge(rel, target, source, message string, author Signature) (string, error) {
	tID, err := s.Resolve(rel, "refs/heads/"+target)
	if err != nil {
		return "", err
	}
	sID, err := s.Resolve(rel, source)
	if err != nil {
		return "", err
	}
	check, err := s.CheckMerge(rel, tID, sID)
	if err != nil {
		return "", err
	}
	if !check.CanMerge {
		return "", &ErrConflict{Files: check.Conflicts}
	}
	p, _ := s.Path(rel)
	out, err := s.run(p, strings.NewReader(message), sigEnv(author, author), "commit-tree", check.TreeID, "-p", tID, "-p", sID, "-F", "-")
	if err != nil {
		return "", err
	}
	newID := strings.TrimSpace(string(out))
	if err := s.UpdateRef(rel, "refs/heads/"+target, newID, tID); err != nil {
		return "", err
	}
	return newID, nil
}

// ErrConflict lists files that prevent a merge.
type ErrConflict struct{ Files []string }

func (e *ErrConflict) Error() string { return "conflitos em: " + strings.Join(e.Files, ", ") }

// Action changes one file in CommitFiles.
type Action struct {
	Kind     string // create, update, delete, move
	Path     string
	Previous string // for move
	Content  []byte
}

// CommitFiles writes actions as a new commit on branch (created from start
// when it does not exist yet; a missing start on an empty repository makes
// a root commit). It is the mechanism behind web edits and file APIs.
func (s *Store) CommitFiles(rel, branch, start, message string, author Signature, actions []Action) (string, error) {
	p, err := s.open(rel)
	if err != nil {
		return "", err
	}
	if !ValidRef(branch) {
		return "", &ErrInvalid{"branch", branch}
	}
	if len(actions) == 0 {
		return "", fmt.Errorf("nenhuma alteração para commitar")
	}
	parent := ""
	oldRef := ZeroID
	if id, err := s.Resolve(rel, "refs/heads/"+branch); err == nil {
		parent, oldRef = id, id
	} else if start != "" {
		if parent, err = s.Resolve(rel, start); err != nil {
			return "", err
		}
	}
	idx, err := os.CreateTemp("", "germanio-index-*")
	if err != nil {
		return "", err
	}
	idx.Close()
	os.Remove(idx.Name())
	defer os.Remove(idx.Name())
	env := []string{"GIT_INDEX_FILE=" + idx.Name()}
	if parent != "" {
		if _, err := s.run(p, nil, env, "read-tree", parent); err != nil {
			return "", err
		}
	}
	exists := func(path string) bool {
		out, _ := s.run(p, nil, env, "ls-files", "--cached", "--", path)
		return len(bytes.TrimSpace(out)) > 0
	}
	for _, a := range actions {
		if !validPath(a.Path) || a.Path == "" {
			return "", &ErrInvalid{"caminho", a.Path}
		}
		switch a.Kind {
		case "create", "update", "move":
			if a.Kind == "create" && exists(a.Path) {
				return "", fmt.Errorf("arquivo já existe: %s", a.Path)
			}
			if a.Kind == "update" && !exists(a.Path) {
				return "", fmt.Errorf("arquivo %w: %s", ErrNotFound, a.Path)
			}
			if a.Kind == "move" {
				if !validPath(a.Previous) || !exists(a.Previous) {
					return "", fmt.Errorf("arquivo %w: %s", ErrNotFound, a.Previous)
				}
				if a.Content == nil {
					b, err := s.run(p, nil, env, "cat-file", "blob", ":"+a.Previous)
					if err != nil {
						return "", err
					}
					a.Content = b
				}
				if _, err := s.run(p, nil, env, "update-index", "--force-remove", "--", a.Previous); err != nil {
					return "", err
				}
			}
			out, err := s.run(p, bytes.NewReader(a.Content), nil, "hash-object", "-w", "--stdin")
			if err != nil {
				return "", err
			}
			blob := strings.TrimSpace(string(out))
			if _, err := s.run(p, nil, env, "update-index", "--add", "--cacheinfo", "100644,"+blob+","+a.Path); err != nil {
				return "", err
			}
		case "delete":
			if !exists(a.Path) {
				return "", fmt.Errorf("arquivo %w: %s", ErrNotFound, a.Path)
			}
			if _, err := s.run(p, nil, env, "update-index", "--force-remove", "--", a.Path); err != nil {
				return "", err
			}
		default:
			return "", fmt.Errorf("ação desconhecida %q (use create, update, delete, move)", a.Kind)
		}
	}
	out, err := s.run(p, nil, env, "write-tree")
	if err != nil {
		return "", err
	}
	tree := strings.TrimSpace(string(out))
	args := []string{"commit-tree", tree, "-F", "-"}
	if parent != "" {
		args = append(args, "-p", parent)
	}
	out, err = s.run(p, strings.NewReader(message), sigEnv(author, author), args...)
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(string(out))
	if err := s.UpdateRef(rel, "refs/heads/"+branch, id, oldRef); err != nil {
		return "", err
	}
	return id, nil
}
