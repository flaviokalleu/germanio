package git

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func oidOf(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// Objects are kept only when their content has the announced oid and size,
// never above the limit, and each repository has its own objects.
func TestLFSStore(t *testing.T) {
	l := &LFSStore{Root: t.TempDir(), MaxSize: 16}
	repo := "@hashed/ab/cd/abcd.git"
	data := []byte("conteúdo grande")
	oid := oidOf(data)

	if err := l.Put(repo, oid, int64(len(data)), bytes.NewReader([]byte("outro conteúdo!!"))); !errors.Is(err, ErrLFSMismatch) {
		t.Fatalf("conteúdo com outro sha256 aceito: %v", err)
	}
	if err := l.Put(repo, oid, int64(len(data)), bytes.NewReader(data[:5])); !errors.Is(err, ErrLFSMismatch) {
		t.Fatalf("conteúdo curto aceito: %v", err)
	}
	if err := l.Put(repo, oid, int64(len(data)), bytes.NewReader(append(data, 'x'))); !errors.Is(err, ErrLFSMismatch) {
		t.Fatalf("conteúdo longo aceito: %v", err)
	}
	big := bytes.Repeat([]byte("a"), 17)
	if err := l.Put(repo, oidOf(big), 17, bytes.NewReader(big)); !errors.Is(err, ErrLFSTooLarge) {
		t.Fatalf("objeto acima do limite aceito: %v", err)
	}
	var leftovers []string
	filepath.WalkDir(l.Root, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			leftovers = append(leftovers, p)
		}
		return nil
	})
	if len(leftovers) != 0 {
		t.Fatalf("envios recusados deixaram arquivos: %v", leftovers)
	}
	if err := l.Put(repo, oid, int64(len(data)), bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	if !l.Has(repo, oid, int64(len(data))) || l.Has(repo, oid, 3) || l.Has("@hashed/ef/01/ef01.git", oid, int64(len(data))) {
		t.Fatal("Has não confere o repositório e o tamanho")
	}
	f, n, err := l.Open(repo, oid)
	if err != nil || n != int64(len(data)) {
		t.Fatalf("Open: %v %d", err, n)
	}
	got, _ := io.ReadAll(f)
	f.Close()
	if !bytes.Equal(got, data) {
		t.Fatalf("conteúdo lido: %q", got)
	}
	for _, bad := range []string{"../x.git", "/abs.git", "-x.git", "sem-sufixo"} {
		if err := l.Put(bad, oid, int64(len(data)), bytes.NewReader(data)); err == nil {
			t.Fatalf("repositório inválido aceito: %q", bad)
		}
	}
	if _, _, err := l.Open(repo, "../../etc/passwd"); err == nil {
		t.Fatal("oid inválido aberto")
	}
	if err := l.RemoveRepo(repo); err != nil || l.Has(repo, oid, int64(len(data))) {
		t.Fatalf("RemoveRepo: %v", err)
	}
}

// The batch answer follows the official protocol: upload actions only for
// missing objects, 404 per missing download, 422 per invalid or too large
// object, and a whole-request error for unknown operations or hashes.
func TestLFSBatch(t *testing.T) {
	l := &LFSStore{Root: t.TempDir(), MaxSize: 100}
	repo := "r.git"
	have := []byte("tenho")
	l.Put(repo, oidOf(have), int64(len(have)), bytes.NewReader(have))
	act := func(op string, o LFSObject) LFSAction {
		return LFSAction{Href: "https://x/" + op + "/" + o.OID, Header: map[string]string{"Authorization": "t"}, ExpiresIn: 3600}
	}
	missing := oidOf([]byte("falta"))
	up, err := l.Batch(repo, &LFSBatchRequest{Operation: "upload", Transfers: []string{"basic"}, Objects: []LFSObject{
		{OID: oidOf(have), Size: int64(len(have))}, {OID: missing, Size: 5}, {OID: "xyz", Size: 1}, {OID: missing, Size: 101}}}, act)
	if err != nil {
		t.Fatal(err)
	}
	o := up.Objects
	if up.Transfer != "basic" || up.HashAlgo != "sha256" || len(o) != 4 {
		t.Fatalf("resposta: %+v", up)
	}
	if o[0].Actions != nil || o[0].Error != nil {
		t.Fatalf("objeto existente não deve ser enviado de novo: %+v", o[0])
	}
	if o[1].Actions["upload"].Href != "https://x/upload/"+missing || !o[1].Authenticated {
		t.Fatalf("objeto ausente sem ação de envio: %+v", o[1])
	}
	if o[2].Error == nil || o[2].Error.Code != 422 || o[3].Error == nil || o[3].Error.Code != 422 || !strings.Contains(o[3].Error.Message, "limite") {
		t.Fatalf("oid inválido / objeto grande: %+v %+v", o[2], o[3])
	}
	down, _ := l.Batch(repo, &LFSBatchRequest{Operation: "download", Objects: []LFSObject{{OID: oidOf(have), Size: int64(len(have))}, {OID: missing, Size: 5}}}, act)
	if down.Objects[0].Actions["download"].Href == "" || down.Objects[1].Error == nil || down.Objects[1].Error.Code != 404 {
		t.Fatalf("download: %+v", down.Objects)
	}
	for _, bad := range []*LFSBatchRequest{{Operation: "apagar"}, {Operation: "download", HashAlgo: "sha1"}, {Operation: "download", Transfers: []string{"tus"}}} {
		if _, err := l.Batch(repo, bad, act); err == nil {
			t.Fatalf("pedido inválido aceito: %+v", bad)
		}
	}
}
