package servidor

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/flaviokalleu/germanio/compiler/ast"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Files of a record (docs/gep/0014-arquivos.md, em teste): a field whose
// type is a file or an image keeps one file. The file is streamed to disk
// before the database is touched, the record keeps only {nome, tamanho,
// tipo, chave}, and the file takes its place after the commit. The place on
// disk is always generated here, never taken from the client.

const defaultFileMB = 25

// isFileField: a field declared as a file or an image (`anexo arquivo`,
// `foto imagem`). A type that comes only from the name keeps the text it has
// always kept: turning it into a file would change what existing programs
// store, so it must be declared (GEP 0014).
func isFileField(f *ast.Field) bool {
	return !f.TypeInferred && (f.Type == ast.FieldArquivo || f.Type == ast.FieldImagem || f.Type == ast.FieldUpload)
}

func fileFields(e *ast.Entity) []*ast.Field {
	var out []*ast.Field
	for _, f := range e.Model.Fields {
		if isFileField(f) {
			out = append(out, f)
		}
	}
	return out
}

func fileLimit() int64 {
	if n, err := strconv.Atoi(os.Getenv("GERMANIO_ARQUIVO_MAX_MB")); err == nil && n > 0 {
		return int64(n) << 20
	}
	return defaultFileMB << 20
}

func filesRoot() string {
	if d := os.Getenv("GERMANIO_ARQUIVOS"); d != "" {
		return d
	}
	return "arquivos"
}

// fileMeta is what the record keeps about its file.
type fileMeta struct {
	Nome    string `json:"nome"`
	Tamanho int64  `json:"tamanho"`
	Tipo    string `json:"tipo"`
	Chave   string `json:"chave"`
}

func metaOf(v any) *fileMeta {
	s, _ := v.(string)
	if s == "" {
		return nil
	}
	var m fileMeta
	if json.Unmarshal([]byte(s), &m) != nil || !validKey(m.Chave) {
		return nil
	}
	return &m
}

// validKey: only keys this server generates (hex) ever reach the disk.
func validKey(k string) bool {
	if len(k) != 32 {
		return false
	}
	_, err := hex.DecodeString(k)
	return err == nil
}

func (a *intentAPI) filePath(e *ast.Entity, key string) string {
	return filepath.Join(filesRoot(), e.Singular, key)
}

// cleanName keeps a file name people recognise, without paths or control
// characters.
func cleanName(n string) string {
	n = filepath.Base(strings.ReplaceAll(n, `\`, "/"))
	n = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == '"' {
			return -1
		}
		return r
	}, n)
	if n == "." || n == "/" || n == "" {
		n = "arquivo"
	}
	if r := []rune(n); len(r) > 200 {
		n = string(r[:200])
	}
	return n
}

// inlineTypes are shown in the page; everything else is downloaded.
var inlineTypes = map[string]bool{"image/png": true, "image/jpeg": true, "image/gif": true, "image/webp": true}

type uploadKey struct{}

// upload is the file already on disk (a temporary place) for this request.
type upload struct {
	field *ast.Field
	tmp   string
	meta  fileMeta
}

// mountFiles adds GET/PUT/DELETE <item>/<campo> for each file field.
func (a *intentAPI) mountFiles(mux *routeMux, item string, chain []*ast.Entity, integration bool) {
	e := chain[len(chain)-1]
	for _, f := range fileFields(e) {
		f := f
		path := strings.ToLower(f.Name)
		if integration {
			path = a.ext(path)
		}
		mux.HandleFunc("GET "+item+"/"+path, func(w http.ResponseWriter, r *http.Request) {
			a.serve(w, r.WithContext(context.WithValue(r.Context(), uploadKey{}, &upload{field: f})), chain, "arquivo_ver", "")
		})
		mux.HandleFunc("DELETE "+item+"/"+path, func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), uploadKey{}, &upload{field: f}))
			a.s.transactional(w, r, func(w http.ResponseWriter, r *http.Request) { a.serve(w, r, chain, "arquivo_remover", "") })
		})
		mux.HandleFunc("PUT "+item+"/"+path, func(w http.ResponseWriter, r *http.Request) {
			// the file goes to disk first, outside any transaction
			up, status, err := receive(r, f)
			if err != nil {
				a.fail(w, status, err.Error())
				return
			}
			committed := false
			defer func() {
				if !committed {
					os.Remove(up.tmp)
				}
			}()
			r = r.WithContext(context.WithValue(r.Context(), uploadKey{}, up))
			a.s.transactional(w, r, func(w http.ResponseWriter, r *http.Request) { a.serve(w, r, chain, "arquivo_enviar", "") })
			if _, err := os.Stat(up.tmp); errors.Is(err, os.ErrNotExist) {
				committed = true // moved into place after the commit
			}
		})
	}
}

// receive streams the body to a temporary file, with the size limit.
func receive(r *http.Request, f *ast.Field) (*upload, int, error) {
	name := r.URL.Query().Get("nome")
	if name == "" {
		if _, params, err := mime.ParseMediaType(r.Header.Get("Content-Disposition")); err == nil {
			name = params["filename"]
		}
	}
	return receiveFrom(r.Body, name, f)
}

// receiveFrom streams body to a temporary file, with the size limit, and
// detects its type from the content.
func receiveFrom(body io.Reader, name string, f *ast.Field) (*upload, int, error) {
	limit := fileLimit()
	dir := filepath.Join(filesRoot(), ".recebendo")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, 500, err
	}
	tmp, err := os.CreateTemp(dir, "up-*")
	if err != nil {
		return nil, 500, err
	}
	defer tmp.Close()
	n, err := io.Copy(tmp, io.LimitReader(body, limit+1))
	if err != nil {
		os.Remove(tmp.Name())
		return nil, 400, fmt.Errorf("não consegui receber o arquivo: %v", err)
	}
	if n > limit {
		os.Remove(tmp.Name())
		return nil, 413, fmt.Errorf("o arquivo passa do limite de %d MB", limit>>20)
	}
	if n == 0 {
		os.Remove(tmp.Name())
		return nil, 400, fmt.Errorf("o arquivo está vazio")
	}
	head := make([]byte, 512)
	tmp.Seek(0, io.SeekStart)
	k, _ := io.ReadFull(tmp, head)
	kind := http.DetectContentType(head[:k])
	if i := strings.Index(kind, ";"); i > 0 {
		kind = kind[:i]
	}
	if f.Type == ast.FieldImagem && !inlineTypes[kind] {
		os.Remove(tmp.Name())
		return nil, 400, fmt.Errorf("%s precisa ser uma imagem (PNG, JPEG, GIF ou WebP)", f.Name)
	}
	key := make([]byte, 16)
	rand.Read(key)
	return &upload{field: f, tmp: tmp.Name(), meta: fileMeta{Nome: cleanName(name), Tamanho: n, Tipo: kind, Chave: hex.EncodeToString(key)}}, 0, nil
}

func uploadOf(r *http.Request) *upload {
	up, _ := r.Context().Value(uploadKey{}).(*upload)
	return up
}

// fileOp serves arquivo_ver / arquivo_enviar / arquivo_remover for row.
func (a *intentAPI) fileOp(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual map[string]any, e *ast.Entity, row map[string]any, op string, deny func(map[string]any)) {
	up := uploadOf(r)
	if up == nil || row == nil || !a.in.Can(ctx, atual, e, "ver", row) {
		a.fail(w, 404, a.msg("404", e))
		return
	}
	field := strings.ToLower(up.field.Name)
	old := metaOf(row[field])
	if op == "arquivo_ver" {
		if old == nil {
			a.fail(w, 404, "nenhum arquivo em "+field)
			return
		}
		file, err := os.Open(a.filePath(e, old.Chave))
		if err != nil {
			a.fail(w, 404, "nenhum arquivo em "+field)
			return
		}
		defer file.Close()
		disposition := "attachment"
		kind := "application/octet-stream"
		if inlineTypes[old.Tipo] {
			disposition, kind = "inline", old.Tipo
		}
		w.Header().Set("Content-Type", kind)
		w.Header().Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": old.Nome}))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
		w.Header().Set("Content-Length", strconv.FormatInt(old.Tamanho, 10))
		io.Copy(w, file)
		return
	}
	if !a.in.Can(ctx, atual, e, "editar", row) {
		deny(row)
		return
	}
	var value any
	if op == "arquivo_enviar" {
		b, _ := json.Marshal(up.meta)
		value = string(b)
	}
	if err := a.frozenFor(ctx, "editar", e, row, map[string]any{field: value}); err != nil {
		a.failErr(w, r, err)
		return
	}
	if _, err := a.in.Op(ctx, e.Singular, "atualizar", row["id"], map[string]any{field: value}); err != nil {
		a.failErr(w, r, err)
		return
	}
	// sending or removing a file is an edit: its rules run, and may refuse
	if h := e.Hooks["editar"]; h != nil {
		updated := a.find(ctx, e, fmt.Sprint(row["id"]), nil)
		if _, _, err := a.in.RunHook(ctx, h, a.hookVars(atual, e, updated, map[string]any{})); err != nil {
			a.failErr(w, r, err)
			return
		}
	}
	afterCommit(ctx, func() {
		if op == "arquivo_enviar" {
			final := a.filePath(e, up.meta.Chave)
			os.MkdirAll(filepath.Dir(final), 0o700)
			if err := os.Rename(up.tmp, final); err != nil {
				fmt.Printf("[germanio] arquivo de %s: %v\n", e.Singular, err)
			}
		}
		if old != nil {
			os.Remove(a.filePath(e, old.Chave))
		}
	})
	if op == "arquivo_remover" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	a.json(w, 200, map[string]any{"nome": up.meta.Nome, "tamanho": up.meta.Tamanho, "tipo": up.meta.Tipo}, nil)
}

// removeFiles deletes the files of a deleted record (after the commit).
func (a *intentAPI) removeFiles(e *ast.Entity, row map[string]any) {
	for _, f := range fileFields(e) {
		if m := metaOf(row[strings.ToLower(f.Name)]); m != nil {
			os.Remove(a.filePath(e, m.Chave))
		}
	}
}

// publicMeta is how a record shows its file: never the place on disk.
func publicMeta(v any) any {
	m := metaOf(v)
	if m == nil {
		return nil
	}
	return map[string]any{"nome": m.Nome, "tamanho": m.Tamanho, "tipo": m.Tipo}
}

// fileAddressKey carries, on a request of a collection, how to address the
// files of its records (GEP 0030: the record shows `<campo>_endereco`).
type fileAddressKey struct{}

type fileAddressing struct {
	base   string            // the collection, with {rN} for its ancestors
	entity *ast.Entity       // the data of the collection
	paths  map[string]string // field -> path segment on this surface
}

func (a *intentAPI) fileAddressing(base string, e *ast.Entity, integration bool) *fileAddressing {
	fields := fileFields(e)
	if len(fields) == 0 {
		return nil
	}
	fa := &fileAddressing{base: base, entity: e, paths: map[string]string{}}
	for _, f := range fields {
		key := strings.ToLower(f.Name)
		fa.paths[key] = key
		if integration {
			fa.paths[key] = a.ext(key)
		}
	}
	return fa
}

// fileAddress: where the file of field k of row is downloaded, on the
// surface of the request ("" when the request does not address row's data).
func fileAddress(ctx *interp.Context, e *ast.Entity, row map[string]any, k string) string {
	if ctx == nil || ctx.Request == nil {
		return ""
	}
	fa, _ := ctx.Request.Context().Value(fileAddressKey{}).(*fileAddressing)
	if fa == nil || fa.entity != e || fa.paths[k] == "" || row["id"] == nil {
		return ""
	}
	path := fa.base
	nested := strings.Contains(path, "{r")
	for i := 0; strings.Contains(path, "{r") && i < 8; i++ {
		path = strings.Replace(path, "{r"+strconv.Itoa(i)+"}", url.PathEscape(ctx.Request.PathValue("r"+strconv.Itoa(i))), 1)
	}
	ref := display(row["id"])
	if nested {
		// inside a parent, numbered records are addressed by their number
		for _, f := range e.Model.Fields {
			if f.NumberedBy != "" && row[strings.ToLower(f.Name)] != nil {
				ref = display(row[strings.ToLower(f.Name)])
			}
		}
	}
	href := path + "/" + url.PathEscape(ref) + "/" + fa.paths[k]
	if public := strings.TrimSuffix(os.Getenv("GERMANIO_URL_PUBLICA"), "/"); public != "" {
		href = public + href
	}
	return href
}

// ---------- pages ----------

type fileView struct {
	Label, Href, Name, Size, Action, CSRF string
	Image                                 bool
}

// fileViews: each file of the record, with a form to send one for those
// who may edit it.
func (ps *pageSite) fileViews(ctx *interp.Context, atual map[string]any, e *ast.Entity, record, row map[string]any, api, base, csrf string) template.HTML {
	fields := fileFields(e)
	if len(fields) == 0 {
		return ""
	}
	edit := atual != nil && ps.a.in.Can(ctx, atual, e, "editar", record)
	var out []fileView
	for _, f := range fields {
		key := strings.ToLower(f.Name)
		v := fileView{Label: fieldLabel(f), Image: f.Type == ast.FieldImagem}
		if m, _ := row[key].(map[string]any); m != nil {
			v.Href, v.Name = api+"/"+key, toStr(m["nome"])
			v.Size = humanSize(int64(asNumber(m["tamanho"])))
		}
		if edit {
			v.Action, v.CSRF = base+"/arquivo/"+key, csrf
		}
		if v.Href != "" || v.Action != "" {
			out = append(out, v)
		}
	}
	return htmlOf(filesTpl, out)
}

func humanSize(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d bytes", n)
}

var filesTpl = tpl(`<section class="arquivos">{{range .}}<div class="arquivo"><strong>{{.Label}}</strong> {{if .Href}}{{if .Image}}<img src="{{.Href}}" alt="{{.Label}}" style="max-width:240px;display:block">{{end}}<a href="{{.Href}}">{{.Name}}</a> ({{.Size}}){{else}}<span class="muted">nenhum arquivo</span>{{end}}{{if .Action}}<form method="post" action="{{.Action}}" enctype="multipart/form-data"><input type="hidden" name="_csrf" value="{{.CSRF}}"><label>Enviar {{.Label}} <input type="file" name="arquivo" required></label> <button>Enviar</button></form>{{end}}</div>{{end}}</section>`)

// uploadFromPage takes a page form (multipart: _csrf first, then the file)
// and streams the file to the record's address, without keeping it in
// memory.
func (ps *pageSite) uploadFromPage(w http.ResponseWriter, r *http.Request, pg *ast.PageDecl, parts []string) {
	back := "/" + slug(pg.Name) + "/" + strings.Join(escapeAll(parts[:len(parts)-2]), "/")
	field := parts[len(parts)-1]
	sess := interp.SessaoDaRequisicao(r)
	if sess == nil {
		http.Redirect(w, r, "/entrar", http.StatusSeeOther)
		return
	}
	mr, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "formulário inválido", http.StatusBadRequest)
		return
	}
	first, err := mr.NextPart()
	if err != nil || first.FormName() != "_csrf" {
		http.Error(w, "token CSRF ausente ou inválido", http.StatusForbidden)
		return
	}
	token, _ := io.ReadAll(io.LimitReader(first, 256))
	if string(token) != fmt.Sprint(sess["csrf"]) {
		http.Error(w, "token CSRF ausente ou inválido", http.StatusForbidden)
		return
	}
	file, err := mr.NextPart()
	if err != nil || file.FileName() == "" {
		http.Redirect(w, r, back+"?erro="+urlQuery("escolha um arquivo"), http.StatusSeeOther)
		return
	}
	_, api, _, _ := ps.resolve(pg, parts[:len(parts)-2])
	req := httptest.NewRequest("PUT", api+"/"+field+"?nome="+urlQuery(file.FileName()), file)
	req.Header.Set("Cookie", r.Header.Get("Cookie"))
	req.Header.Set("X-CSRF-Token", fmt.Sprint(sess["csrf"]))
	rec := httptest.NewRecorder()
	ps.a.s.mux.ServeHTTP(rec, req)
	if rec.Code >= 300 {
		var out any
		json.Unmarshal(rec.Body.Bytes(), &out)
		http.Redirect(w, r, back+"?erro="+urlQuery(message(out)), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, back+"?ok="+urlQuery("Arquivo enviado"), http.StatusSeeOther)
}

// storeWorkFile keeps the file an executor sends for its work (for example a
// job's artifacts): the multipart part `part` of the current request is
// streamed into the work's file field `field` (trabalho_remoto.guardar_arquivo).
func (a *intentAPI) storeWorkFile(ctx *interp.Context, w *ast.Entity, row map[string]any, field, part string) (map[string]any, error) {
	var f *ast.Field
	for _, x := range fileFields(w) {
		if strings.EqualFold(x.Name, field) {
			f = x
		}
	}
	if f == nil {
		return nil, fmt.Errorf("%s não é um campo de arquivo de %s", field, w.Plural)
	}
	r := ctx.Request
	if r == nil {
		return nil, fmt.Errorf("guardar_arquivo só vale dentro de uma rota")
	}
	mr, err := r.MultipartReader()
	if err != nil {
		return nil, fmt.Errorf("o envio não é multipart: %v", err)
	}
	for {
		p, err := mr.NextPart()
		if err != nil {
			return nil, fmt.Errorf("o envio não tem a parte %q", part)
		}
		if p.FormName() != part {
			continue
		}
		up, _, err := receiveFrom(p, p.FileName(), f)
		if err != nil {
			return nil, err
		}
		key := strings.ToLower(f.Name)
		old := metaOf(row[key])
		b, _ := json.Marshal(up.meta)
		if _, err := a.in.Op(ctx, w.Singular, "atualizar", row["id"], map[string]any{key: string(b)}); err != nil {
			os.Remove(up.tmp)
			return nil, err
		}
		final := a.filePath(w, up.meta.Chave)
		os.MkdirAll(filepath.Dir(final), 0o700)
		if err := os.Rename(up.tmp, final); err != nil {
			return nil, err
		}
		if old != nil {
			os.Remove(a.filePath(w, old.Chave))
		}
		return map[string]any{"nome": up.meta.Nome, "tamanho": up.meta.Tamanho, "tipo": up.meta.Tipo}, nil
	}
}

// fileIn names a file field that a JSON or form body tries to set.
func fileIn(e *ast.Entity, body map[string]any) string {
	for _, f := range fileFields(e) {
		key := strings.ToLower(f.Name)
		if _, ok := body[key]; ok {
			return key
		}
		if _, ok := body[f.Name]; ok {
			return key
		}
	}
	return ""
}
