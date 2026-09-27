package servidor

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/runtime/git"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// `X tem repositório`: every record owns a Git repository, created with the
// record, removed with it, and served over smart HTTP at /<chave>.git with
// clone/push authorized by the app's rules (baixar/enviar código).

// RepoPath is the storage path of a record's repository (hashed layout,
// independent of names so renames never move data).
func RepoPath(model string, id any) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%v", model, id)))
	h := hex.EncodeToString(sum[:])
	return "@hashed/" + h[:2] + "/" + h[2:4] + "/" + h + ".git"
}

func (a *intentAPI) createRepository(ctx *interp.Context, e *ast.Entity, row map[string]any) error {
	if !e.Repository || a.s.Git == nil {
		return nil
	}
	path := RepoPath(e.Singular, row["id"])
	branch := "main"
	if b, ok := row["default_branch"].(string); ok && b != "" {
		branch = b
	} else if b, ok := row["branch_principal"].(string); ok && b != "" {
		branch = b
	}
	if err := a.s.Git.Init(path, branch); err != nil {
		return err
	}
	res, err := a.in.Op(ctx, e.Singular, "atualizar", row["id"], map[string]any{"repositorio": path})
	if err != nil {
		a.s.Git.Remove(path)
		return err
	}
	for k, v := range res.(map[string]any) {
		row[k] = v
	}
	return nil
}

func (a *intentAPI) removeRepository(e *ast.Entity, row map[string]any) {
	if e.Repository && a.s.Git != nil {
		if p, ok := row["repositorio"].(string); ok && p != "" {
			a.s.Git.Remove(p)
		}
	}
}

// gitMiddleware routes /<chave>.git/... to the repository service.
func (s *Servidor) gitMiddleware(next http.Handler) http.Handler {
	app := s.Program.App
	if app == nil || s.Git == nil {
		return next
	}
	var repoEntities []*ast.Entity
	for _, n := range app.Order {
		if app.Entities[n].Repository {
			repoEntities = append(repoEntities, app.Entities[n])
		}
	}
	if len(repoEntities) == 0 {
		return next
	}
	a := &intentAPI{s: s, app: app, in: s.Interpreter}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i := strings.Index(r.URL.Path, ".git/")
		if i < 0 {
			next.ServeHTTP(w, r)
			return
		}
		key := strings.TrimPrefix(r.URL.Path[:i], "/")
		sub := r.URL.Path[i+5:]
		service, advertise, ok := git.ServiceFromRequest(r, sub)
		if !ok {
			http.NotFound(w, r)
			return
		}
		a.serveGit(w, r, repoEntities, key, service, advertise)
	})
}

func (a *intentAPI) serveGit(w http.ResponseWriter, r *http.Request, entities []*ast.Entity, key, service string, advertise bool) {
	ctx := &interp.Context{Request: r, Writer: w}
	challenge := func() {
		w.Header().Set("WWW-Authenticate", `Basic realm="Germanio"`)
		http.Error(w, "HTTP Basic: Access denied", http.StatusUnauthorized)
	}
	atual, err := a.s.identify(ctx, r)
	if err != nil {
		challenge()
		return
	}
	var e *ast.Entity
	var row map[string]any
	for _, cand := range entities {
		res, err := a.in.Op(ctx, cand.Singular, "encontrar", map[string]any{cand.RepoKey: key})
		if m, ok := res.(map[string]any); ok && err == nil {
			e, row = cand, m
			break
		}
	}
	if row == nil || !a.in.Can(ctx, atual, e, "ver", row) {
		if atual == nil {
			challenge()
			return
		}
		http.NotFound(w, r)
		return
	}
	verb := "baixar_codigo"
	if service == "receive-pack" {
		verb = "enviar_codigo"
	}
	allowed := false
	if verb == "baixar_codigo" && len(e.Rules["baixar_codigo"]) == 0 {
		allowed = true // seeing the record is enough to clone when no rule narrows it
	} else {
		allowed = a.in.Can(ctx, atual, e, verb, row)
	}
	if !allowed {
		if atual == nil {
			challenge()
			return
		}
		http.Error(w, "You are not allowed to "+strings.ReplaceAll(verb, "_codigo", " code")+" in this repository", http.StatusForbidden)
		return
	}
	repo, _ := row["repositorio"].(string)
	updatesToMaps := func(list []git.RefUpdate) []any {
		out := make([]any, 0, len(list))
		for _, u := range list {
			m := map[string]any{"ref": u.Ref, "antes": u.Old, "depois": u.New, "tipo": u.Kind()}
			if strings.HasPrefix(u.Ref, "refs/heads/") {
				m["branch"] = strings.TrimPrefix(u.Ref, "refs/heads/")
			}
			if strings.HasPrefix(u.Ref, "refs/tags/") {
				m["tag"] = strings.TrimPrefix(u.Ref, "refs/tags/")
			}
			out = append(out, m)
		}
		return out
	}
	vars := func(list []git.RefUpdate) map[string]any {
		return map[string]any{"atual": nilIfEmpty(atual), "registro": row, e.Singular: row, "atualizacoes": updatesToMaps(list)}
	}
	var check func([]git.RefUpdate) error
	if h := e.Hooks["antes_enviar_codigo"]; h != nil && service == "receive-pack" {
		check = func(list []git.RefUpdate) error {
			_, _, err := a.in.RunHook(ctx, h, vars(list))
			if err != nil {
				return fmt.Errorf("%s", interp.Friendly(err))
			}
			return nil
		}
	}
	applied, err := a.s.Git.ServeHTTP(w, r, repo, service, advertise, check)
	if err != nil {
		http.Error(w, "git: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if service == "receive-pack" && !advertise && len(applied) > 0 {
		if h := e.Hooks["enviar_codigo"]; h != nil {
			if _, _, err := a.in.RunHook(ctx, h, vars(applied)); err != nil {
				// The push already happened; report the failure in the log.
				fmt.Printf("[germanio] quando enviar código para %s: %v\n", e.Singular, err)
			}
		}
	}
}

// mountRepository adds browsing operations to an entity with a repository.
func (a *intentAPI) mountRepository(mux *http.ServeMux, base string, e *ast.Entity) {
	seg := "repositorio"
	names := map[string]string{"branches": "branches", "commits": "commits", "tree": "arvore", "files": "arquivos", "compare": "comparar", "diff": "diff"}
	if a.app.Messages == "en" {
		seg = "repository"
		names = map[string]string{"branches": "branches", "commits": "commits", "tree": "tree", "files": "files", "compare": "compare", "diff": "diff"}
	}
	root := base + "/{ref}/" + seg
	h := func(fn func(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual map[string]any, row map[string]any, repo string), write bool) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := &interp.Context{Request: r, Writer: w}
			atual, err := a.s.identify(ctx, r)
			if err != nil {
				a.fail(w, 401, a.msg("401", e))
				return
			}
			row := a.find(ctx, e, r.PathValue("ref"), nil)
			if row == nil || !a.in.Can(ctx, atual, e, "ver", row) {
				a.fail(w, 404, a.msg("404", e))
				return
			}
			verb := "baixar_codigo"
			if write {
				verb = "enviar_codigo"
			}
			if !(verb == "baixar_codigo" && len(e.Rules[verb]) == 0) && !a.in.Can(ctx, atual, e, verb, row) {
				if atual == nil {
					a.fail(w, 401, a.msg("401", e))
				} else {
					a.fail(w, 403, a.msg("403", e))
				}
				return
			}
			repo, _ := row["repositorio"].(string)
			fn(w, r, ctx, atual, row, repo)
		}
	}
	gitErr := func(w http.ResponseWriter, err error) {
		var inv *git.ErrInvalid
		switch {
		case errorsIs(err, git.ErrNotFound):
			a.fail(w, 404, "404 Not Found")
		case errorsAs(err, &inv):
			a.fail(w, 400, err.Error())
		default:
			a.fail(w, 400, err.Error())
		}
	}
	defaultRef := func(r *http.Request, row map[string]any, keys ...string) string {
		for _, k := range keys {
			if v := r.URL.Query().Get(k); v != "" {
				return v
			}
		}
		if b, ok := row["default_branch"].(string); ok && b != "" {
			return b
		}
		return "main"
	}
	mux.HandleFunc("GET "+root+"/"+names["branches"], h(func(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual, row map[string]any, repo string) {
		list, err := a.s.Git.Branches(repo)
		if err != nil {
			gitErr(w, err)
			return
		}
		out := []any{}
		for _, b := range list {
			c := b.Commit
			out = append(out, map[string]any{"name": b.Name, "default": b.Name == row["default_branch"], "commit": commitJSON(&c)})
		}
		a.json(w, 200, out, nil)
	}, false))
	mux.HandleFunc("POST "+root+"/"+names["branches"], h(func(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual, row map[string]any, repo string) {
		body, _ := readBody(r)
		name := first(toStr(body["branch"]), toStr(body["nome"]))
		from := first(toStr(body["ref"]), toStr(body["origem"]))
		if err := a.pushCheck(ctx, atual, e, row, []git.RefUpdate{{Old: git.ZeroID, New: "(novo)", Ref: "refs/heads/" + name}}); err != nil {
			a.failErr(w, r, err)
			return
		}
		if _, err := a.s.Git.CreateBranch(repo, name, from); err != nil {
			gitErr(w, err)
			return
		}
		b, _ := a.s.Git.GetCommit(repo, "refs/heads/"+name)
		a.json(w, 201, map[string]any{"name": name, "commit": commitJSON(b)}, nil)
	}, true))
	mux.HandleFunc("DELETE "+root+"/"+names["branches"]+"/{branch...}", h(func(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual, row map[string]any, repo string) {
		name := r.PathValue("branch")
		if name == row["default_branch"] {
			a.fail(w, 400, "Cannot remove the default branch")
			return
		}
		if err := a.pushCheck(ctx, atual, e, row, []git.RefUpdate{{Old: "(atual)", New: git.ZeroID, Ref: "refs/heads/" + name}}); err != nil {
			a.failErr(w, r, err)
			return
		}
		if err := a.s.Git.DeleteBranch(repo, name); err != nil {
			gitErr(w, err)
			return
		}
		a.json(w, 204, nil, nil)
	}, true))
	mux.HandleFunc("GET "+root+"/"+names["commits"], h(func(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual, row map[string]any, repo string) {
		q := r.URL.Query()
		per := atoiDefault(q.Get("per_page"), 20)
		page := atoiDefault(q.Get("page"), 1)
		list, err := a.s.Git.Log(repo, defaultRef(r, row, "ref_name", "ref"), "", q.Get("path"), per, (page-1)*per)
		if err != nil {
			gitErr(w, err)
			return
		}
		out := []any{}
		for i := range list {
			out = append(out, commitJSON(&list[i]))
		}
		a.json(w, 200, out, nil)
	}, false))
	mux.HandleFunc("GET "+root+"/"+names["commits"]+"/{sha}", h(func(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual, row map[string]any, repo string) {
		c, err := a.s.Git.GetCommit(repo, r.PathValue("sha"))
		if err != nil {
			gitErr(w, err)
			return
		}
		a.json(w, 200, commitJSON(c), nil)
	}, false))
	mux.HandleFunc("GET "+root+"/"+names["commits"]+"/{sha}/"+names["diff"], h(func(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual, row map[string]any, repo string) {
		c, err := a.s.Git.GetCommit(repo, r.PathValue("sha"))
		if err != nil {
			gitErr(w, err)
			return
		}
		from := ""
		if len(c.ParentIDs) > 0 {
			from = c.ParentIDs[0]
		}
		files, _, err := a.s.Git.Diff(repo, from, c.ID, 1000)
		if err != nil {
			gitErr(w, err)
			return
		}
		a.json(w, 200, diffJSON(files), nil)
	}, false))
	mux.HandleFunc("GET "+root+"/"+names["tree"], h(func(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual, row map[string]any, repo string) {
		list, err := a.s.Git.Tree(repo, defaultRef(r, row, "ref"), strings.Trim(r.URL.Query().Get("path"), "/"))
		if err != nil {
			gitErr(w, err)
			return
		}
		out := []any{}
		for _, t := range list {
			out = append(out, map[string]any{"id": t.ID, "name": t.Name, "type": t.Type, "path": t.Path, "mode": t.Mode})
		}
		a.json(w, 200, out, nil)
	}, false))
	mux.HandleFunc("GET "+root+"/"+names["files"]+"/{path...}", h(func(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual, row map[string]any, repo string) {
		path := r.PathValue("path")
		raw := strings.HasSuffix(path, "/raw")
		path = strings.TrimSuffix(path, "/raw")
		b, err := a.s.Git.ReadFile(repo, defaultRef(r, row, "ref"), path, 10<<20)
		if err != nil {
			gitErr(w, err)
			return
		}
		if raw {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			if b.Binary {
				w.Header().Set("Content-Type", "application/octet-stream")
			}
			w.Write(b.Content)
			return
		}
		a.json(w, 200, map[string]any{"file_path": b.Path, "file_name": b.Path[strings.LastIndex(b.Path, "/")+1:], "size": b.Size, "blob_id": b.ID, "encoding": "text", "content": string(b.Content), "binary": b.Binary}, nil)
	}, false))
	mux.HandleFunc("GET "+root+"/"+names["compare"], h(func(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual, row map[string]any, repo string) {
		q := r.URL.Query()
		base, err := a.s.Git.MergeBase(repo, q.Get("from"), q.Get("to"))
		if err != nil {
			gitErr(w, err)
			return
		}
		commits, err := a.s.Git.Log(repo, q.Get("to"), base, "", 100, 0)
		if err != nil {
			gitErr(w, err)
			return
		}
		files, _, err := a.s.Git.Diff(repo, base, q.Get("to"), 1000)
		if err != nil {
			gitErr(w, err)
			return
		}
		cl := []any{}
		for i := range commits {
			cl = append(cl, commitJSON(&commits[i]))
		}
		a.json(w, 200, map[string]any{"commits": cl, "diffs": diffJSON(files)}, nil)
	}, false))
}

// pushCheck runs `antes de enviar código` for changes made through the API.
func (a *intentAPI) pushCheck(ctx *interp.Context, atual map[string]any, e *ast.Entity, row map[string]any, updates []git.RefUpdate) error {
	h := e.Hooks["antes_enviar_codigo"]
	if h == nil {
		return nil
	}
	list := []any{}
	for _, u := range updates {
		list = append(list, map[string]any{"ref": u.Ref, "branch": strings.TrimPrefix(u.Ref, "refs/heads/"), "antes": u.Old, "depois": u.New, "tipo": u.Kind()})
	}
	_, _, err := a.in.RunHook(ctx, h, map[string]any{"atual": nilIfEmpty(atual), "registro": row, e.Singular: row, "atualizacoes": list})
	if err != nil {
		var re *interp.RuntimeError
		if errorsAs(err, &re) && re.Status == 400 {
			re.Status = 403
		}
	}
	return err
}

func commitJSON(c *git.Commit) map[string]any {
	if c == nil {
		return nil
	}
	return map[string]any{"id": c.ID, "short_id": c.ShortID, "title": c.Title, "message": c.Message, "author_name": c.AuthorName,
		"author_email": c.AuthorEmail, "authored_date": c.AuthorAt, "committer_name": c.CommitterName, "committer_email": c.CommitterEmail,
		"committed_date": c.CommittedAt, "parent_ids": c.ParentIDs}
}

func diffJSON(files []git.FileDiff) []any {
	out := []any{}
	for _, f := range files {
		out = append(out, map[string]any{"old_path": f.OldPath, "new_path": f.NewPath, "new_file": f.NewFile, "deleted_file": f.DeletedFile, "renamed_file": f.RenamedFile, "diff": f.Diff})
	}
	return out
}

var errorsIs = errors.Is
var errorsAs = errors.As
