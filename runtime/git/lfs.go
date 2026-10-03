package git

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Git LFS (the open Git Large File Storage protocol, batch API and basic
// transfer: github.com/git-lfs/git-lfs/blob/main/docs/api): large files live
// outside the repository as objects named by the SHA-256 of their content.
// This file is the mechanism — where objects are kept, how an upload is
// verified, what a batch answers. Who may read or write is decided by the
// caller with the same rules as clone and push.

// LFSMediaType is the content type of every batch request and response.
const LFSMediaType = "application/vnd.git-lfs+json"

// LFSMaxBatch bounds the objects of one batch request (the official client
// sends 100 at a time).
const LFSMaxBatch = 1000

var oidRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ValidOID reports whether oid is a lowercase hex SHA-256.
func ValidOID(oid string) bool { return oidRe.MatchString(oid) }

// ErrLFSTooLarge: the object is larger than the configured limit.
var ErrLFSTooLarge = errors.New("objeto LFS maior que o limite")

// ErrLFSMismatch: the uploaded content does not have the announced SHA-256
// or size. Nothing is kept.
var ErrLFSMismatch = errors.New("o conteúdo enviado não confere com o oid e o tamanho anunciados")

// LFSStore keeps the LFS objects of each repository in its own directory
// under Root (objects are never shared between repositories, so access to
// one never reveals another's content).
type LFSStore struct {
	Root    string
	MaxSize int64 // largest object accepted; <= 0 means no object is accepted
}

func (l *LFSStore) dir(repo string) (string, error) {
	if !repoRe.MatchString(repo) || strings.Contains(repo, "..") || strings.HasPrefix(repo, "/") || strings.HasPrefix(repo, "-") {
		return "", &ErrInvalid{"repositório", repo}
	}
	root, err := filepath.Abs(l.Root)
	if err != nil {
		return "", err
	}
	d := filepath.Join(root, filepath.FromSlash(strings.TrimSuffix(repo, ".git")))
	if !strings.HasPrefix(d, root+string(os.PathSeparator)) {
		return "", &ErrInvalid{"repositório", repo}
	}
	return d, nil
}

func (l *LFSStore) path(repo, oid string) (string, error) {
	if !ValidOID(oid) {
		return "", &ErrInvalid{"oid", oid}
	}
	d, err := l.dir(repo)
	if err != nil {
		return "", err
	}
	return filepath.Join(d, oid[:2], oid[2:4], oid), nil
}

// Has reports whether the object exists with exactly this size.
func (l *LFSStore) Has(repo, oid string, size int64) bool {
	p, err := l.path(repo, oid)
	if err != nil {
		return false
	}
	st, err := os.Stat(p)
	return err == nil && st.Mode().IsRegular() && st.Size() == size
}

// Put streams body to disk, computing its SHA-256 on the way, and keeps it
// only when the content has exactly the announced oid and size. At most
// size+1 bytes are read, so a client cannot fill the disk past the limit.
func (l *LFSStore) Put(repo, oid string, size int64, body io.Reader) error {
	p, err := l.path(repo, oid)
	if err != nil {
		return err
	}
	if size < 0 {
		return ErrLFSMismatch
	}
	if size > l.MaxSize {
		return ErrLFSTooLarge
	}
	if l.Has(repo, oid, size) {
		return nil // content-addressed: the same oid is the same content
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".envio-*")
	if err != nil {
		return err
	}
	keep := false
	defer func() {
		if !keep {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(body, size+1))
	if err != nil {
		return err
	}
	if n != size || hex.EncodeToString(h.Sum(nil)) != oid {
		return ErrLFSMismatch
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), p); err != nil {
		return err
	}
	keep = true
	return nil
}

// Open returns the object's content and size, or ErrNotFound.
func (l *LFSStore) Open(repo, oid string) (*os.File, int64, error) {
	p, err := l.path(repo, oid)
	if err != nil {
		return nil, 0, err
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, 0, ErrNotFound
	}
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() {
		f.Close()
		return nil, 0, ErrNotFound
	}
	return f, st.Size(), nil
}

// RemoveRepo deletes every object of the repository.
func (l *LFSStore) RemoveRepo(repo string) error {
	d, err := l.dir(repo)
	if err != nil {
		return err
	}
	return os.RemoveAll(d)
}

// LFSBatchRequest is the body of POST <repo>/info/lfs/objects/batch.
type LFSBatchRequest struct {
	Operation string      `json:"operation"`
	Transfers []string    `json:"transfers,omitempty"`
	Ref       *LFSRef     `json:"ref,omitempty"`
	Objects   []LFSObject `json:"objects"`
	HashAlgo  string      `json:"hash_algo,omitempty"`
}

// LFSRef is the ref a batch request is made for (informative).
type LFSRef struct {
	Name string `json:"name"`
}

// LFSObject names an object by its oid and size.
type LFSObject struct {
	OID  string `json:"oid"`
	Size int64  `json:"size"`
}

// LFSAction tells the client where to transfer an object.
type LFSAction struct {
	Href      string            `json:"href"`
	Header    map[string]string `json:"header,omitempty"`
	ExpiresIn int               `json:"expires_in,omitempty"`
}

// LFSError is a per-object (or whole-request) error.
type LFSError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// LFSObjectResponse is the answer for one object.
type LFSObjectResponse struct {
	OID           string               `json:"oid"`
	Size          int64                `json:"size"`
	Authenticated bool                 `json:"authenticated,omitempty"`
	Actions       map[string]LFSAction `json:"actions,omitempty"`
	Error         *LFSError            `json:"error,omitempty"`
}

// LFSBatchResponse is the body answered to a batch request.
type LFSBatchResponse struct {
	Transfer string              `json:"transfer"`
	Objects  []LFSObjectResponse `json:"objects"`
	HashAlgo string              `json:"hash_algo"`
}

// Batch answers a batch request for repo. action builds the transfer action
// (href and headers) of an object the client must upload or download. An
// error is a problem with the whole request (the client sees 422).
func (l *LFSStore) Batch(repo string, req *LFSBatchRequest, action func(op string, o LFSObject) LFSAction) (*LFSBatchResponse, error) {
	if req.Operation != "upload" && req.Operation != "download" {
		return nil, fmt.Errorf("operação LFS desconhecida %q: use upload ou download", req.Operation)
	}
	if req.HashAlgo != "" && req.HashAlgo != "sha256" {
		return nil, fmt.Errorf("algoritmo %q não suportado: os objetos são identificados por sha256", req.HashAlgo)
	}
	if len(req.Transfers) > 0 {
		basic := false
		for _, t := range req.Transfers {
			basic = basic || t == "basic"
		}
		if !basic {
			return nil, fmt.Errorf("só a transferência basic é oferecida")
		}
	}
	if len(req.Objects) > LFSMaxBatch {
		return nil, fmt.Errorf("objetos demais num pedido (%d): no máximo %d", len(req.Objects), LFSMaxBatch)
	}
	out := &LFSBatchResponse{Transfer: "basic", HashAlgo: "sha256", Objects: make([]LFSObjectResponse, 0, len(req.Objects))}
	for _, o := range req.Objects {
		r := LFSObjectResponse{OID: o.OID, Size: o.Size}
		switch {
		case !ValidOID(o.OID) || o.Size < 0:
			r.Error = &LFSError{Code: 422, Message: "oid ou tamanho inválido: o oid é o sha256 do conteúdo (64 caracteres hexadecimais) e o tamanho não é negativo"}
		case req.Operation == "upload" && o.Size > l.MaxSize:
			r.Error = &LFSError{Code: 422, Message: fmt.Sprintf("o objeto tem %d bytes, mais que o limite de %d bytes deste servidor", o.Size, l.MaxSize)}
		case req.Operation == "upload":
			if !l.Has(repo, o.OID, o.Size) {
				r.Authenticated = true
				r.Actions = map[string]LFSAction{"upload": action("upload", o)}
			}
		default:
			if l.Has(repo, o.OID, o.Size) {
				r.Authenticated = true
				r.Actions = map[string]LFSAction{"download": action("download", o)}
			} else {
				r.Error = &LFSError{Code: 404, Message: "objeto não encontrado"}
			}
		}
		out.Objects = append(out.Objects, r)
	}
	return out, nil
}
