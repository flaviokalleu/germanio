package servidor

import (
	"archive/zip"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/runtime/banco"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Artifacts of steps: the files a step keeps (`artefatos` in the run file)
// go to the step's file field; later steps that wait for it (or name it in
// recebe_artefatos_de) receive them; `artefatos_expiram_em` deletes them
// after a while. The local executor collects and unpacks them itself; a
// remote executor sends them with trabalho_remoto.guardar_arquivo and reads
// the ones it receives with its work token (see workMayRead).

// expiryField is the step's system field with the moment its artifacts expire.
const expiryField = "artefatos_expiram_em"

// artifactField is the step's file field that keeps its artifacts: the one
// named artefatos, otherwise the only file field it has.
func artifactField(e *ast.Entity) *ast.Field {
	files := fileFields(e)
	for _, f := range files {
		if strings.EqualFold(f.Name, "artefatos") {
			return f
		}
	}
	if len(files) == 1 {
		return files[0]
	}
	return nil
}

// isArtifactField: f keeps the artifacts of a pipeline step.
func isArtifactField(e *ast.Entity, f *ast.Field) bool {
	return e.Execution != nil && e.Execution.Role == "step" && f != nil && artifactField(e) == f
}

// artifactExpiry: when artifacts stored now expire (nil: they are kept).
func artifactExpiry(job map[string]any) any {
	if s := planOf(job).Expire; s > 0 {
		return stampAt(time.Now().Add(time.Duration(s * float64(time.Second))))
	}
	return nil
}

func stampAt(t time.Time) string { return t.UTC().Format(time.RFC3339) }

// artifactsExpired: the step's artifacts passed their expiry.
func artifactsExpired(row map[string]any) bool {
	at := toStr(row[expiryField])
	if at == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, at)
	return err == nil && !time.Now().Before(t)
}

// stepSources: the steps (of the same run) whose artifacts job receives.
func (a *intentAPI) stepSources(ctx *interp.Context, step *ast.Entity, job map[string]any) []map[string]any {
	run := a.app.Entities[step.Execution.Run]
	jobs := a.steps(ctx, run, job[step.Execution.RunField])
	g := newStepGraph(jobs)
	// a retried step is not in jobs; plan its row the same way
	g.plans[toStr(job["id"])] = planOf(job)
	return g.sources(job)
}

// receivedArtifacts: the sources of job that have artifacts to give now.
func (a *intentAPI) receivedArtifacts(ctx *interp.Context, step *ast.Entity, job map[string]any) []map[string]any {
	f := artifactField(step)
	if f == nil {
		return nil
	}
	var out []map[string]any
	for _, s := range a.stepSources(ctx, step, job) {
		if metaOf(s[strings.ToLower(f.Name)]) != nil && !artifactsExpired(s) {
			out = append(out, s)
		}
	}
	return out
}

// dependencies describes, for an executor, the artifacts its work receives:
// [{id, nome, arquivo: {nome, tamanho}}].
func (a *intentAPI) dependencies(ctx *interp.Context, step *ast.Entity, job map[string]any) []any {
	out := []any{}
	f := artifactField(step)
	for _, s := range a.receivedArtifacts(ctx, step, job) {
		m := metaOf(s[strings.ToLower(f.Name)])
		out = append(out, map[string]any{"id": s["id"], "nome": s["nome"], "arquivo": map[string]any{"nome": m.Nome, "tamanho": float64(m.Tamanho)}})
	}
	return out
}

// workMayRead: a request without a person, carrying the token of running
// work (in the header the vocabulary names cabecalho_trabalho), may read the
// artifacts of a step that work receives — and nothing else.
func (a *intentAPI) workMayRead(ctx *interp.Context, r *http.Request, e *ast.Entity, f *ast.Field, row map[string]any) bool {
	if !isArtifactField(e, f) {
		return false
	}
	token := r.Header.Get(headerName(*a, "cabecalho_trabalho", "X-Germanio-Trabalho"))
	if token == "" {
		return false
	}
	w, job := a.stepByToken(ctx, token, true)
	if job == nil || w != e || toStr(job[e.Execution.RunField]) != toStr(row[e.Execution.RunField]) {
		return false
	}
	for _, s := range a.stepSources(ctx, e, job) {
		if toStr(s["id"]) == toStr(row["id"]) {
			return true
		}
	}
	return false
}

// keepFile puts a file already on disk (tmp) into the record's file field,
// replacing the previous one; for artifacts it also sets their expiry.
func (a *intentAPI) keepFile(ctx *interp.Context, w *ast.Entity, row map[string]any, f *ast.Field, tmp string, meta fileMeta) error {
	key := strings.ToLower(f.Name)
	old := metaOf(row[key])
	b, _ := json.Marshal(meta)
	change := map[string]any{key: string(b)}
	if isArtifactField(w, f) {
		change[expiryField] = artifactExpiry(row)
	}
	if _, err := a.in.Op(ctx, w.Singular, "atualizar", row["id"], change); err != nil {
		os.Remove(tmp)
		return err
	}
	final := a.filePath(w, meta.Chave)
	os.MkdirAll(filepath.Dir(final), 0o700)
	if err := os.Rename(tmp, final); err != nil {
		return err
	}
	if old != nil {
		os.Remove(a.filePath(w, old.Chave))
	}
	return nil
}

// collectArtifacts zips the files a step kept (its artefatos patterns,
// relative to the code) into its artifact field. Paths that leave the
// code's directory and symbolic links are never followed.
func (a *intentAPI) collectArtifacts(step *ast.Entity, job map[string]any, work string, log io.Writer) bool {
	patterns := planOf(job).Artifacts
	f := artifactField(step)
	if len(patterns) == 0 || f == nil {
		return true
	}
	files := map[string]string{} // name in the zip → path on disk
	for _, p := range patterns {
		clean := filepath.Clean(filepath.FromSlash(p))
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			fmt.Fprintf(log, "AVISO: o artefato %s fica fora do código e foi ignorado\n", p)
			continue
		}
		matches, _ := filepath.Glob(filepath.Join(work, clean))
		for _, m := range matches {
			filepath.WalkDir(m, func(path string, d fs.DirEntry, err error) error {
				if err != nil || d.Type()&fs.ModeSymlink != 0 {
					return nil
				}
				rel, rerr := filepath.Rel(work, path)
				if rerr != nil || strings.HasPrefix(rel, "..") {
					return nil
				}
				if d.IsDir() {
					if d.Name() == ".git" {
						return filepath.SkipDir
					}
					return nil
				}
				if d.Type().IsRegular() {
					files[filepath.ToSlash(rel)] = path
				}
				return nil
			})
		}
	}
	if len(files) == 0 {
		fmt.Fprintln(log, "AVISO: nenhum arquivo corresponde aos artefatos")
		return true
	}
	dir := filepath.Join(filesRoot(), ".recebendo")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		fmt.Fprintf(log, "ERRO ao guardar os artefatos: %v\n", err)
		return false
	}
	tmp, err := os.CreateTemp(dir, "art-*")
	if err != nil {
		fmt.Fprintf(log, "ERRO ao guardar os artefatos: %v\n", err)
		return false
	}
	limit := fileLimit()
	counted := &limitedWriter{w: tmp, left: limit}
	zw := zip.NewWriter(counted)
	fail := func(msg string) bool {
		tmp.Close()
		os.Remove(tmp.Name())
		fmt.Fprintln(log, msg)
		return false
	}
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		src, err := os.Open(files[n])
		if err != nil {
			continue
		}
		dst, err := zw.Create(n)
		if err == nil {
			_, err = io.Copy(dst, src)
		}
		src.Close()
		if err != nil {
			return fail(fmt.Sprintf("ERRO: os artefatos passam do limite de %d MB e não foram guardados", limit>>20))
		}
		fmt.Fprintf(log, "Guardando artefato %s\n", n)
	}
	if err := zw.Close(); err != nil {
		return fail(fmt.Sprintf("ERRO: os artefatos passam do limite de %d MB e não foram guardados", limit>>20))
	}
	tmp.Close()
	key := make([]byte, 16)
	rand.Read(key)
	meta := fileMeta{Nome: "artefatos.zip", Tamanho: limit - counted.left, Tipo: "application/zip", Chave: hex.EncodeToString(key)}
	ctx := &interp.Context{}
	res, _ := a.in.Op(ctx, step.Singular, "buscar", job["id"])
	row, _ := res.(map[string]any)
	if row == nil {
		os.Remove(tmp.Name())
		return true
	}
	if err := a.keepFile(ctx, step, row, f, tmp.Name(), meta); err != nil {
		fmt.Fprintf(log, "ERRO ao guardar os artefatos: %v\n", err)
		return false
	}
	return true
}

// limitedWriter refuses to write past left bytes.
type limitedWriter struct {
	w    io.Writer
	left int64
}

func (l *limitedWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > l.left {
		return 0, fmt.Errorf("limite excedido")
	}
	n, err := l.w.Write(p)
	l.left -= int64(n)
	return n, err
}

// unpackArtifacts puts the artifacts a step receives into its code
// directory before its commands run. Entries that would leave the
// directory, and symbolic links, are skipped; the total unpacked is bounded.
func (a *intentAPI) unpackArtifacts(step *ast.Entity, job map[string]any, work string, log io.Writer) {
	f := artifactField(step)
	if f == nil {
		return
	}
	budget := 20 * fileLimit()
	for _, s := range a.receivedArtifacts(&interp.Context{}, step, job) {
		m := metaOf(s[strings.ToLower(f.Name)])
		zr, err := zip.OpenReader(a.filePath(step, m.Chave))
		if err != nil {
			fmt.Fprintf(log, "AVISO: os artefatos de %s não puderam ser lidos: %v\n", toStr(s["nome"]), err)
			continue
		}
		fmt.Fprintf(log, "Recebendo artefatos de %s\n", toStr(s["nome"]))
		for _, zf := range zr.File {
			target := filepath.Join(work, filepath.FromSlash(zf.Name))
			rel, err := filepath.Rel(work, target)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || zf.Mode()&fs.ModeSymlink != 0 {
				continue
			}
			if zf.FileInfo().IsDir() {
				os.MkdirAll(target, 0o755)
				continue
			}
			if budget <= 0 {
				fmt.Fprintln(log, "AVISO: artefatos grandes demais para descompactar; o resto foi ignorado")
				break
			}
			os.MkdirAll(filepath.Dir(target), 0o755)
			src, err := zf.Open()
			if err != nil {
				continue
			}
			dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err == nil {
				n, _ := io.Copy(dst, io.LimitReader(src, budget))
				budget -= n
				dst.Close()
			}
			src.Close()
		}
		zr.Close()
	}
}

// expireArtifacts deletes the artifacts whose time passed (a bounded batch
// per step data each call); the step keeps the moment they expired.
func (a *intentAPI) expireArtifacts(clock time.Time) int {
	ctx := &interp.Context{}
	removed := 0
	for _, n := range a.app.Order {
		e := a.app.Entities[n]
		f := artifactField(e)
		if e.Execution == nil || e.Execution.Role != "step" || f == nil {
			continue
		}
		key := strings.ToLower(f.Name)
		res, err := a.in.Op(ctx, e.Singular, "filtrar", map[string]any{expiryField + "__menor_igual": stampAt(clock), key + "__diferente": nil}, map[string]any{"limite": 200, "ordenar": "id"})
		if err != nil {
			continue
		}
		for _, it := range res.([]any) {
			row := it.(map[string]any)
			m := metaOf(row[key])
			if n, err := a.s.DB.AtualizarOnde(e.Singular, banco.Consulta{Filtros: map[string]any{"id": row["id"]}}, map[string]any{key: nil}); err == nil && n == 1 {
				if m != nil {
					os.Remove(a.filePath(e, m.Chave))
				}
				removed++
			}
		}
	}
	return removed
}

// startArtifactCleanup runs expireArtifacts every minute, only for
// programs that have pipeline steps with artifacts.
func (a *intentAPI) startArtifactCleanup() {
	needed := false
	for _, n := range a.app.Order {
		e := a.app.Entities[n]
		if e.Execution != nil && e.Execution.Role == "step" && artifactField(e) != nil {
			needed = true
		}
	}
	if !needed {
		return
	}
	stop := make(chan struct{})
	a.s.onClose = append(a.s.onClose, func() { close(stop) })
	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case c := <-t.C:
				a.expireArtifacts(c)
			}
		}
	}()
}

// ExpirarArtefatos deletes now the step artifacts whose time passed (the
// server also does it every minute) and answers how many were deleted.
func (s *Servidor) ExpirarArtefatos(agora time.Time) int {
	if s.intent == nil {
		return 0
	}
	return s.intent.expireArtifacts(agora)
}
