package servidor

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/runtime/git"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Git LFS for `X tem repositório` (docs/gep/0035-git-lfs.md, em teste):
// <chave>.git/info/lfs/objects/batch answers the official batch API and
// <chave>.git/info/lfs/objects/<oid> receives and serves the objects. The
// same people who may clone may download, the same who may push may upload
// (baixar código / enviar código, read-only records included), with the
// same credentials as git over HTTP. Objects are streamed to disk, verified
// by SHA-256 and size, never above GERMANIO_LFS_MAX_MB, and never touch the
// database.

const defaultLFSMB = 100

// lfsTokenLife is how long the transfer link of one object stays valid.
const lfsTokenLife = time.Hour

func lfsLimit() int64 {
	if n, err := strconv.Atoi(os.Getenv("GERMANIO_LFS_MAX_MB")); err == nil && n > 0 {
		return int64(n) << 20
	}
	return defaultLFSMB << 20
}

// lfsStore keeps the objects under the files root (GERMANIO_ARQUIVOS).
func lfsStore() *git.LFSStore {
	return &git.LFSStore{Root: filepath.Join(filesRoot(), "lfs"), MaxSize: lfsLimit()}
}

func lfsFail(w http.ResponseWriter, status int, msg string) {
	if status == http.StatusUnauthorized {
		w.Header().Set("LFS-Authenticate", `Basic realm="Germanio"`)
		w.Header().Set("WWW-Authenticate", `Basic realm="Germanio"`)
		if msg == "" {
			msg = "Credenciais necessárias: use o nome de usuário e um token de acesso, como no git"
		}
	}
	if msg == "" {
		msg = http.StatusText(status)
	}
	w.Header().Set("Content-Type", git.LFSMediaType)
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"message": msg})
}

// serveLFS answers <chave>.git/info/lfs/<sub>.
func (a *intentAPI) serveLFS(w http.ResponseWriter, r *http.Request, entities []*ast.Entity, key, sub string) {
	switch {
	case sub == "objects/batch" && r.Method == http.MethodPost:
		a.lfsBatch(w, r, entities, key)
	case strings.HasPrefix(sub, "objects/") && (r.Method == http.MethodGet || r.Method == http.MethodPut):
		a.lfsObject(w, r, entities, key, strings.TrimPrefix(sub, "objects/"))
	default:
		// locks and anything else of the protocol are not offered; the
		// official client goes on without them
		lfsFail(w, http.StatusNotFound, "Não oferecido por este servidor")
	}
}

func (a *intentAPI) lfsBatch(w http.ResponseWriter, r *http.Request, entities []*ast.Entity, key string) {
	var req git.LFSBatchRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		lfsFail(w, http.StatusUnprocessableEntity, "Pedido LFS inválido: o corpo deve ser o JSON do batch (operation e objects)")
		return
	}
	write := req.Operation == "upload"
	ctx := &interp.Context{Request: r, Writer: w}
	acc, ok := a.codeAccess(ctx, r, entities, key, write, true, func(status int, msg string) { lfsFail(w, status, msg) })
	if !ok {
		return
	}
	repo, _ := acc.row["repositorio"].(string)
	base := publicBase()
	if base == "" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		base = scheme + "://" + r.Host
	}
	href := base + "/" + key + ".git/info/lfs/objects/"
	exp := time.Now().Add(lfsTokenLife).Unix()
	res, err := lfsStore().Batch(repo, &req, func(op string, o git.LFSObject) git.LFSAction {
		tok := lfsSign(lfsClaims{Repo: repo, OID: o.OID, Size: o.Size, Op: op, Exp: exp})
		return git.LFSAction{Href: href + o.OID, Header: map[string]string{"Authorization": lfsScheme + tok}, ExpiresIn: int(lfsTokenLife.Seconds())}
	})
	if err != nil {
		lfsFail(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.Header().Set("Content-Type", git.LFSMediaType)
	json.NewEncoder(w).Encode(res)
}

// lfsObject receives (PUT) or serves (GET) one object. The transfer link
// from the batch carries a signed permission for exactly this repository,
// object, size and direction; without it, the request is checked like any
// git request.
func (a *intentAPI) lfsObject(w http.ResponseWriter, r *http.Request, entities []*ast.Entity, key, oid string) {
	if !git.ValidOID(oid) {
		lfsFail(w, http.StatusNotFound, "Objeto não encontrado")
		return
	}
	op := "download"
	if r.Method == http.MethodPut {
		op = "upload"
	}
	ctx := &interp.Context{Request: r, Writer: w}
	var repo string
	size := int64(-1)
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, lfsScheme) {
		c := lfsVerify(strings.TrimPrefix(h, lfsScheme))
		_, row := a.repoRecord(ctx, entities, key)
		if c == nil || row == nil || c.Repo != toStr(row["repositorio"]) || c.OID != oid || c.Op != op {
			lfsFail(w, http.StatusUnauthorized, "O link de transferência é inválido ou venceu: peça de novo (git lfs push / git lfs pull)")
			return
		}
		repo, size = c.Repo, c.Size
	} else {
		acc, ok := a.codeAccess(ctx, r, entities, key, op == "upload", true, func(status int, msg string) { lfsFail(w, status, msg) })
		if !ok {
			return
		}
		repo = toStr(acc.row["repositorio"])
	}
	store := lfsStore()
	if op == "download" {
		f, n, err := store.Open(repo, oid)
		if err != nil || (size >= 0 && n != size) {
			if f != nil {
				f.Close()
			}
			lfsFail(w, http.StatusNotFound, "Objeto não encontrado")
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", "application/octet-stream")
		http.ServeContent(w, r, "", time.Time{}, f)
		return
	}
	if size < 0 {
		size = r.ContentLength
	} else if r.ContentLength >= 0 && r.ContentLength != size {
		lfsFail(w, http.StatusUnprocessableEntity, "O tamanho enviado não é o anunciado no batch")
		return
	}
	if size < 0 {
		lfsFail(w, http.StatusLengthRequired, "Informe o tamanho do objeto (Content-Length)")
		return
	}
	if size > store.MaxSize {
		lfsFail(w, http.StatusRequestEntityTooLarge, "O objeto passa do limite deste servidor (GERMANIO_LFS_MAX_MB)")
		return
	}
	switch err := store.Put(repo, oid, size, r.Body); {
	case err == nil:
		w.WriteHeader(http.StatusOK)
	case errors.Is(err, git.ErrLFSTooLarge):
		lfsFail(w, http.StatusRequestEntityTooLarge, "O objeto passa do limite deste servidor (GERMANIO_LFS_MAX_MB)")
	case errors.Is(err, git.ErrLFSMismatch):
		lfsFail(w, http.StatusUnprocessableEntity, "O conteúdo enviado não tem o sha256 e o tamanho anunciados; nada foi guardado")
	default:
		fmt.Printf("[germanio] objeto LFS: %v\n", err)
		lfsFail(w, http.StatusInternalServerError, "Não foi possível guardar o objeto")
	}
}

// ---------- signed transfer links ----------

const lfsScheme = "Germanio-LFS "

type lfsClaims struct {
	Repo string `json:"r"`
	OID  string `json:"o"`
	Size int64  `json:"s"`
	Op   string `json:"p"`
	Exp  int64  `json:"e"`
}

// lfsKey separates these links from sessions and other signed tokens: a
// link is never accepted as anything else, nor anything else as a link.
func lfsKey() []byte {
	m := hmac.New(sha256.New, interp.Segredo())
	m.Write([]byte("germanio/lfs/v1"))
	return m.Sum(nil)
}

func lfsSign(c lfsClaims) string {
	b, _ := json.Marshal(c)
	p := base64.RawURLEncoding.EncodeToString(b)
	m := hmac.New(sha256.New, lfsKey())
	m.Write([]byte(p))
	return p + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func lfsVerify(tok string) *lfsClaims {
	p, sig, ok := strings.Cut(strings.TrimSpace(tok), ".")
	if !ok {
		return nil
	}
	got, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil {
		return nil
	}
	m := hmac.New(sha256.New, lfsKey())
	m.Write([]byte(p))
	if !hmac.Equal(got, m.Sum(nil)) {
		return nil
	}
	b, err := base64.RawURLEncoding.DecodeString(p)
	if err != nil {
		return nil
	}
	var c lfsClaims
	if json.Unmarshal(b, &c) != nil || time.Now().Unix() > c.Exp {
		return nil
	}
	return &c
}
