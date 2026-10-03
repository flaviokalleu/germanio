package servidor

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/flaviokalleu/germanio/compiler/ast"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Images in formatted text (UP-01, docs/gep/0014-arquivos.md): an image
// pasted or dropped into a formatted text belongs to the record of the page
// where the text is written (the issue of a comment), is seen by whoever
// sees that record and goes with it. Sending one needs the right to edit the
// record or to create something inside it. Only images; the type comes from
// the content; the place on disk is generated here.

const imagesTable = "_germanio_imagens"

// takesImages: e has a formatted text, or something inside it does.
func (a *intentAPI) takesImages(e *ast.Entity) bool {
	has := func(x *ast.Entity) bool {
		for _, f := range x.Model.Fields {
			if f.Formatted {
				return true
			}
		}
		return false
	}
	if has(e) {
		return true
	}
	for _, c := range a.childrenOf(e) {
		if has(c) {
			return true
		}
	}
	return false
}

func (a *intentAPI) setupTextImages() {
	for _, n := range a.app.Order {
		if a.takesImages(a.app.Entities[n]) {
			a.s.DB.DB.Exec(`CREATE TABLE IF NOT EXISTS ` + imagesTable + ` (chave TEXT PRIMARY KEY, recurso TEXT NOT NULL, recurso_id INTEGER NOT NULL, nome TEXT, tipo TEXT NOT NULL, tamanho INTEGER NOT NULL)`)
			return
		}
	}
}

// mayAddImage: the person may edit the record, or create one of its
// children that has a formatted text.
func (a *intentAPI) mayAddImage(ctx *interp.Context, atual map[string]any, e *ast.Entity, row map[string]any) bool {
	if atual == nil {
		return false
	}
	if a.in.Can(ctx, atual, e, "editar", row) {
		return true
	}
	for _, c := range a.childrenOf(e) {
		data := map[string]any{}
		for field, target := range c.Parents {
			if target == e.Singular {
				data[field] = row["id"]
			}
		}
		if a.in.Can(ctx, atual, c, "criar", data) {
			return true
		}
	}
	return false
}

func imagesDir() string { return filepath.Join(filesRoot(), "imagens") }

// textImageOp serves POST …/imagens (send) and GET …/imagens/{chave}.
func (a *intentAPI) textImageOp(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual map[string]any, e *ast.Entity, row map[string]any) {
	if row == nil || !a.in.Can(ctx, atual, e, "ver", row) {
		a.fail(w, 404, a.msg("404", e))
		return
	}
	if r.Method == http.MethodGet {
		key := r.PathValue("chave")
		var kind, name string
		var size int64
		err := a.s.DB.DB.QueryRow(fmt.Sprintf(`SELECT tipo, nome, tamanho FROM %s WHERE chave = %s AND recurso = %s AND recurso_id = %s`, imagesTable, a.s.ph(1), a.s.ph(2), a.s.ph(3)), key, e.Singular, row["id"]).Scan(&kind, &name, &size)
		if err != nil || !validKey(key) || !inlineTypes[kind] {
			a.fail(w, 404, "imagem não encontrada")
			return
		}
		f, err := os.Open(filepath.Join(imagesDir(), key))
		if err != nil {
			a.fail(w, 404, "imagem não encontrada")
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", kind)
		w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": name}))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		w.Header().Set("Cache-Control", "private, max-age=3600")
		io.Copy(w, f)
		return
	}
	if !a.mayAddImage(ctx, atual, e, row) {
		a.fail(w, 403, a.msg("403", e))
		return
	}
	up, status, err := receiveFrom(r.Body, r.URL.Query().Get("nome"), &ast.Field{Name: "imagem", Type: ast.FieldImagem})
	if err != nil {
		a.fail(w, status, err.Error())
		return
	}
	key := make([]byte, 16)
	rand.Read(key)
	k := hex.EncodeToString(key)
	os.MkdirAll(imagesDir(), 0o700)
	if err := os.Rename(up.tmp, filepath.Join(imagesDir(), k)); err != nil {
		os.Remove(up.tmp)
		a.failErr(w, r, err)
		return
	}
	if _, err := a.s.DB.DB.Exec(fmt.Sprintf(`INSERT INTO %s (chave, recurso, recurso_id, nome, tipo, tamanho) VALUES (%s, %s, %s, %s, %s, %s)`, imagesTable, a.s.ph(1), a.s.ph(2), a.s.ph(3), a.s.ph(4), a.s.ph(5), a.s.ph(6)), k, e.Singular, row["id"], up.meta.Nome, up.meta.Tipo, up.meta.Tamanho); err != nil {
		os.Remove(filepath.Join(imagesDir(), k))
		a.failErr(w, r, err)
		return
	}
	url := r.URL.Path + "/" + k
	a.json(w, 201, map[string]any{"url": url, "markdown": "![" + up.meta.Nome + "](" + url + ")"}, nil)
}

// removeTextImages: the images of a deleted record go with it.
func (a *intentAPI) removeTextImages(e *ast.Entity, id any) {
	rows, err := a.s.DB.DB.Query(fmt.Sprintf(`SELECT chave FROM %s WHERE recurso = %s AND recurso_id = %s`, imagesTable, a.s.ph(1), a.s.ph(2)), e.Singular, id)
	if err != nil {
		return
	}
	var keys []string
	for rows.Next() {
		var k string
		rows.Scan(&k)
		keys = append(keys, k)
	}
	rows.Close()
	for _, k := range keys {
		if validKey(k) {
			os.Remove(filepath.Join(imagesDir(), k))
		}
	}
	a.s.DB.DB.Exec(fmt.Sprintf(`DELETE FROM %s WHERE recurso = %s AND recurso_id = %s`, imagesTable, a.s.ph(1), a.s.ph(2)), e.Singular, id)
}
