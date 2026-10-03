package servidor

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

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
	branch := defaultBranch(row)
	if err := a.s.Git.Init(path, branch); err != nil {
		return err
	}
	undoOnRollback(ctx, func() { a.s.Git.Remove(path) })
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

// initialFile commits the file a new repository may start with, when the
// creation asked for it (iniciar_repositorio).
func (a *intentAPI) initialFile(atual map[string]any, e *ast.Entity, row, body map[string]any) error {
	f := e.InitialFile
	if f == nil || a.s.Git == nil || !truthy(body["iniciar_repositorio"]) {
		return nil
	}
	content := placeholderRe.ReplaceAllStringFunc(f.Content, func(m string) string { return toStr(row[m[1:len(m)-1]]) })
	author := git.Signature{Name: "Germanio", Email: "germanio@localhost", When: time.Now()}
	if atual != nil {
		author.Name, author.Email = first(toStr(atual["nome"]), toStr(atual["username"]), author.Name), first(toStr(atual["email"]), author.Email)
	}
	msg := map[string]string{"pt": "Primeiro commit", "en": "Initial commit"}[a.app.Messages]
	_, err := a.s.Git.CommitFiles(toStr(row["repositorio"]), defaultBranch(row), "", msg, author, []git.Action{{Kind: "create", Path: f.Path, Content: []byte(content)}})
	return err
}

var placeholderRe = regexp.MustCompile(`\{[a-z_]+\}`)

func truthy(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return x == "true" || x == "1" || x == "sim"
	case float64:
		return x != 0
	}
	return false
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
	var e *ast.Entity
	var row map[string]any
	for _, cand := range entities {
		res, err := a.in.Op(ctx, cand.Singular, "encontrar", map[string]any{cand.RepoKey: key})
		if m, ok := res.(map[string]any); ok && err == nil {
			e, row = cand, m
			break
		}
	}
	// A running step's token reads the repository of that step (remote executors clone with it).
	if _, pass, ok := r.BasicAuth(); ok && row != nil && service == "upload-pack" {
		if step, job := a.stepByToken(ctx, pass, true); job != nil && a.stepOwnerIs(ctx, step, job, e, row) {
			a.gitProtocol(w, r, ctx, row, service, advertise)
			return
		}
	}
	atual, err := a.s.identify(ctx, r)
	if err != nil {
		challenge()
		return
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
		if le, _ := a.lockedAncestor(ctx, e, row, true, 0); le != nil {
			http.Error(w, interp.Friendly(a.readOnlyError(le)), http.StatusForbidden)
			return
		}
	}
	if !a.s.scopeAllows(ctx, verb) {
		http.Error(w, "The token does not have the scope for this operation", http.StatusForbidden)
		return
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
	vars := func(list []git.RefUpdate) map[string]any {
		return map[string]any{"atual": nilIfEmpty(atual), "registro": row, e.Singular: row, "atualizacoes": updatesToMaps(list)}
	}
	var check func([]git.RefUpdate) error
	if service == "receive-pack" {
		check = func(list []git.RefUpdate) error {
			if err := a.protectedBranch(ctx, atual, e, row, list); err != nil {
				return fmt.Errorf("%s", interp.Friendly(err))
			}
			if h := e.Hooks["antes_enviar_codigo"]; h != nil {
				if _, _, err := a.in.RunHook(ctx, h, vars(list)); err != nil {
					return fmt.Errorf("%s", interp.Friendly(err))
				}
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
		pushed := map[string]any{"atualizacoes": updatesToMaps(applied), "id": row["id"]}
		for k, v := range row {
			if _, taken := pushed[k]; !taken {
				pushed[k] = v
			}
		}
		a.emit(ctx, e, "enviar_codigo", pushed, atual)
		if err := a.history(ctx, atual, e, "enviar_codigo", nil, row); err != nil {
			// the git answer is already written: report, do not answer twice
			fmt.Printf("[germanio] histórico do envio de código: %v\n", err)
		}
		a.startRuns(ctx, atual, e, row, updatesToMaps(applied))
		if h := e.Hooks["enviar_codigo"]; h != nil {
			if _, _, err := a.in.RunHook(ctx, h, vars(applied)); err != nil {
				// The push already happened; report the failure in the log.
				fmt.Printf("[germanio] quando enviar código para %s: %v\n", e.Singular, err)
			}
		}
	}
}

// mountRepository adds browsing operations to an entity with a repository.
func (a *intentAPI) mountRepository(mux *routeMux, base string, e *ast.Entity) {
	seg := "repositorio"
	names := map[string]string{"branches": "branches", "tags": "tags", "commits": "commits", "tree": "arvore", "files": "arquivos", "compare": "comparar", "diff": "diff", "archive": "baixar"}
	if a.extern && a.app.Messages == "en" {
		seg = "repository"
		names = map[string]string{"branches": "branches", "tags": "tags", "commits": "commits", "tree": "tree", "files": "files", "compare": "compare", "diff": "diff", "archive": "archive"}
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
			need := "ler"
			if write {
				need = "escrever"
			}
			if !a.s.scopeAllows(ctx, need) {
				a.fail(w, 403, map[string]any{"pt": "O token não tem escopo para esta ação", "en": "insufficient_scope"}[a.app.Messages])
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
		return defaultBranch(row)
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
			out = append(out, map[string]any{"name": b.Name, "default": b.Name == defaultBranch(row), "commit": commitJSON(&c)})
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
		if name == defaultBranch(row) {
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
	// tags: the same rules as code (see: baixar; create or remove: enviar)
	mux.HandleFunc("GET "+root+"/"+names["tags"], h(func(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual, row map[string]any, repo string) {
		list, err := a.s.Git.Tags(repo)
		if err != nil {
			gitErr(w, err)
			return
		}
		out := []any{}
		for _, b := range list {
			c := b.Commit
			out = append(out, map[string]any{"name": b.Name, "target": c.ID, "commit": commitJSON(&c)})
		}
		a.json(w, 200, out, nil)
	}, false))
	mux.HandleFunc("POST "+root+"/"+names["tags"], h(func(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual, row map[string]any, repo string) {
		body, _ := readBody(r)
		name := first(toStr(body["tag_name"]), toStr(body["nome"]))
		from := first(toStr(body["ref"]), toStr(body["origem"]), defaultBranch(row))
		if err := a.pushCheck(ctx, atual, e, row, []git.RefUpdate{{Old: git.ZeroID, New: "(novo)", Ref: "refs/tags/" + name}}); err != nil {
			a.failErr(w, r, err)
			return
		}
		id, err := a.s.Git.CreateTag(repo, name, from)
		if err != nil {
			gitErr(w, err)
			return
		}
		c, _ := a.s.Git.GetCommit(repo, id)
		a.json(w, 201, map[string]any{"name": name, "target": id, "commit": commitJSON(c)}, nil)
	}, true))
	mux.HandleFunc("DELETE "+root+"/"+names["tags"]+"/{tag...}", h(func(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual, row map[string]any, repo string) {
		name := r.PathValue("tag")
		if err := a.pushCheck(ctx, atual, e, row, []git.RefUpdate{{Old: "(atual)", New: git.ZeroID, Ref: "refs/tags/" + name}}); err != nil {
			a.failErr(w, r, err)
			return
		}
		if err := a.s.Git.DeleteTag(repo, name); err != nil {
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
	// editing a file on the web is a commit by the person, under the same
	// rules as pushing (protected branches included); executions start as
	// after a push
	mux.HandleFunc("PUT "+root+"/"+names["files"]+"/{path...}", h(func(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual, row map[string]any, repo string) {
		body, err := readBody(r)
		if err != nil {
			a.failErr(w, r, err)
			return
		}
		path := strings.Trim(r.PathValue("path"), "/")
		branch := first(toStr(body["branch"]), defaultBranch(row))
		content := first(toStr(body["conteudo"]), toStr(body["content"]))
		message := first(toStr(body["mensagem"]), toStr(body["commit_message"]), "Atualiza "+path)
		if atual == nil {
			a.fail(w, 401, a.msg("401", e))
			return
		}
		old, _ := a.s.Git.Resolve(repo, "refs/heads/"+branch)
		if old == "" {
			old = git.ZeroID
		}
		update := git.RefUpdate{Old: old, New: "(novo)", Ref: "refs/heads/" + branch}
		if err := a.pushCheck(ctx, atual, e, row, []git.RefUpdate{update}); err != nil {
			a.failErr(w, r, err)
			return
		}
		kind := "update"
		if _, err := a.s.Git.ReadFile(repo, "refs/heads/"+branch, path, 1); err != nil {
			kind = "create"
		}
		name := first(toStr(atual["nome"]), toStr(atual["name"]), toStr(atual["username"]), "Germanio")
		email := first(toStr(atual["email"]), "sem-email@germanio.local")
		id, err := a.s.Git.CommitFiles(repo, branch, "", message, git.Signature{Name: name, Email: email, When: time.Now()}, []git.Action{{Kind: kind, Path: path, Content: []byte(content)}})
		if err != nil {
			gitErr(w, err)
			return
		}
		update.New = id
		a.startRuns(ctx, atual, e, row, updatesToMaps([]git.RefUpdate{update}))
		a.json(w, 200, map[string]any{"file_path": path, "branch": branch, "commit_id": id}, nil)
	}, true))
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
	// downloading the code of a revision as one file, for whoever may
	// download code (baixar código): streamed from git, a bounded number at
	// a time
	archive := func(format string) http.HandlerFunc {
		return h(func(w http.ResponseWriter, r *http.Request, ctx *interp.Context, atual, row map[string]any, repo string) {
			ref := defaultRef(r, row, "sha", "ref")
			id, err := a.s.Git.Resolve(repo, ref)
			if err != nil {
				gitErr(w, err)
				return
			}
			select {
			case a.archives <- struct{}{}:
				defer func() { <-a.archives }()
			case <-time.After(10 * time.Second):
				a.fail(w, http.StatusServiceUnavailable, "Muitos downloads do código ao mesmo tempo. Tente de novo em instantes.")
				return
			case <-r.Context().Done():
				return
			}
			key := toStr(row[e.RepoKey])
			key = key[strings.LastIndex(key, "/")+1:]
			name := archiveName(key + "-" + ref)
			ext := map[string]string{"zip": "zip", "tar": "tar", "tar.gz": "tar.gz", "tgz": "tar.gz"}[format]
			types := map[string]string{"zip": "application/zip", "tar": "application/x-tar", "tar.gz": "application/gzip"}
			w.Header().Set("Content-Type", types[ext])
			w.Header().Set("Content-Disposition", `attachment; filename="`+name+"."+ext+`"`)
			w.Header().Set("X-Content-Type-Options", "nosniff")
			if err := a.s.Git.Archive(r.Context(), repo, id, format, name+"/", w); err != nil {
				// the headers are gone; the truncated file is the signal
				fmt.Printf("[germanio] download do código interrompido: %v\n", err)
			}
		}, false)
	}
	arch := names["archive"]
	mux.HandleFunc("GET "+root+"/"+arch, archive("tar.gz"))
	for _, f := range []string{"zip", "tar", "tar.gz", "tgz"} {
		mux.HandleFunc("GET "+root+"/"+arch+"."+f, archive(f))
	}
}

// archiveName keeps letters, digits, dot, dash and underscore of s (the
// rest becomes a dash): the name of a downloaded file and of its folder.
func archiveName(s string) string {
	b := []byte(s)
	for i, c := range b {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-') {
			b[i] = '-'
		}
	}
	out := strings.Trim(strings.ReplaceAll(string(b), "..", "-"), ".-")
	if len(out) > 150 {
		out = out[:150]
	}
	if out == "" {
		out = "codigo"
	}
	return out
}

// pushCheck runs `antes de enviar código` for changes made through the API.
func (a *intentAPI) pushCheck(ctx *interp.Context, atual map[string]any, e *ast.Entity, row map[string]any, updates []git.RefUpdate) error {
	if err := a.protectedBranch(ctx, atual, e, row, updates); err != nil {
		return err
	}
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

// defaultBranch reads the record's main branch (branch_padrao,
// branch_principal or default_branch), "main" when absent.
func defaultBranch(row map[string]any) string {
	for _, k := range []string{"branch_padrao", "branch_principal", "default_branch"} {
		if b, ok := row[k].(string); ok && b != "" {
			return b
		}
	}
	return "main"
}

// protectedBranch: `somente <papel> pode enviar código para a branch padrão`.
func (a *intentAPI) protectedBranch(ctx *interp.Context, atual map[string]any, e *ast.Entity, row map[string]any, updates []git.RefUpdate) error {
	if err := a.protectedByData(ctx, atual, e, row, updates); err != nil {
		return err
	}
	if e.ProtectedBranchRole == "" || a.in.IsAdmin(atual) {
		return nil
	}
	main := "refs/heads/" + defaultBranch(row)
	for _, u := range updates {
		if u.Ref == main && a.in.Level(ctx, atual, e, row) < a.app.Level(e.ProtectedBranchRole) {
			msg := fmt.Sprintf("Somente %s pode enviar código para a branch padrão", e.ProtectedBranchRole)
			if a.app.Messages == "en" {
				msg = "You are not allowed to push code to protected branches on this project."
			}
			return &interp.RuntimeError{Status: 403, Message: msg}
		}
	}
	return nil
}

// ---------- code review (records with origem/destino branches) ----------

func (a *intentAPI) reviewRepo(ctx *interp.Context, e *ast.Entity, row map[string]any) (string, map[string]any) {
	pe := a.app.Entities[e.Parents[e.Review.RepoVia]]
	res, _ := a.in.Op(ctx, pe.Singular, "buscar", row[e.Review.RepoVia])
	parent, _ := res.(map[string]any)
	if parent == nil {
		return "", nil
	}
	repo, _ := parent["repositorio"].(string)
	return repo, parent
}

func (a *intentAPI) mergeCheck(ctx *interp.Context, e *ast.Entity, row map[string]any) map[string]any {
	if a.s.Git == nil {
		return nil
	}
	repo, _ := a.reviewRepo(ctx, e, row)
	m, err := a.s.Git.CheckMerge(repo, "refs/heads/"+fmt.Sprint(row[e.Review.Target]), "refs/heads/"+fmt.Sprint(row[e.Review.Source]))
	if err != nil {
		return map[string]any{"pode": false, "conflitos": []any{}}
	}
	conf := make([]any, len(m.Conflicts))
	for i, c := range m.Conflicts {
		conf[i] = c
	}
	return map[string]any{"pode": m.CanMerge, "conflitos": conf}
}

// checkBranches: branch fields must name existing branches of the
// repository the record belongs to, and origem ≠ destino.
func (a *intentAPI) checkBranches(ctx *interp.Context, e *ast.Entity, data map[string]any) error {
	if e.Review == nil || a.s.Git == nil {
		return nil
	}
	repo, _ := a.reviewRepo(ctx, e, data)
	errs := map[string]any{}
	for _, f := range []string{e.Review.Source, e.Review.Target} {
		if f == "" || data[f] == nil {
			continue
		}
		if !a.s.Git.BranchExists(repo, fmt.Sprint(data[f])) {
			errs[f] = []any{map[string]string{"pt": "não existe", "en": "does not exist"}[a.app.Messages]}
		}
	}
	if e.Review.Target != "" && data[e.Review.Source] != nil && fmt.Sprint(data[e.Review.Source]) == fmt.Sprint(data[e.Review.Target]) {
		errs[e.Review.Target] = []any{map[string]string{"pt": "deve ser diferente da origem", "en": "must be different from the source"}[a.app.Messages]}
	}
	if len(errs) > 0 {
		return &interp.RuntimeError{Status: 400, Message: "branches inválidas", Payload: errs}
	}
	return nil
}

// merge joins origem into destino in the repository. Merging into the main
// branch counts as sending code to it (protected branch rules apply);
// drafts (rascunho) cannot be merged; conflicts refuse with the file list.
func (a *intentAPI) merge(ctx *interp.Context, atual map[string]any, e *ast.Entity, row map[string]any) error {
	en := a.app.Messages == "en"
	for _, final := range e.Finals {
		if fmt.Sprint(row[e.StateField]) == final {
			return &interp.RuntimeError{Status: 405, Message: map[string]string{"pt": e.Label + " não muda mais", "en": "405 Method Not Allowed"}[a.app.Messages]}
		}
	}
	if fmt.Sprint(row[e.StateField]) != e.Initial {
		msg := fmt.Sprintf("%s não está %s", e.Label, e.Initial)
		if en {
			msg = "405 Method Not Allowed"
		}
		return &interp.RuntimeError{Status: 405, Message: msg}
	}
	if b, _ := row["rascunho"].(bool); b {
		msg := "Rascunhos não podem ser mesclados"
		if en {
			msg = "Draft merge requests cannot be merged"
		}
		return &interp.RuntimeError{Status: 406, Message: msg}
	}
	repo, parent := a.reviewRepo(ctx, e, row)
	pe := a.app.Entities[e.Parents[e.Review.RepoVia]]
	target := fmt.Sprint(row[e.Review.Target])
	if err := a.protectedBranch(ctx, atual, pe, parent, []git.RefUpdate{{Old: "(atual)", New: "(mescla)", Ref: "refs/heads/" + target}}); err != nil {
		return err
	}
	title := fmt.Sprint(first(toStr(row["titulo"]), toStr(row["title"])))
	msg := fmt.Sprintf("Merge branch '%s' into '%s'\n\n%s\n", row[e.Review.Source], target, title)
	author := git.Signature{Name: toStr(atual["nome"]), Email: toStr(atual["email"])}
	if author.Name == "" {
		author.Name = toStr(atual["username"])
	}
	sha, err := a.s.Git.Merge(repo, target, "refs/heads/"+fmt.Sprint(row[e.Review.Source]), msg, author)
	if err != nil {
		var conf *git.ErrConflict
		if errorsAs(err, &conf) {
			m := "Há conflitos entre as branches: " + strings.Join(conf.Files, ", ")
			if en {
				m = "Branch cannot be merged"
			}
			return &interp.RuntimeError{Status: 406, Message: m}
		}
		return err
	}
	_, err = a.in.Op(ctx, e.Singular, "atualizar", row["id"], map[string]any{"commit_mesclagem": sha})
	return err
}

func (a *intentAPI) approve(ctx *interp.Context, atual map[string]any, e *ast.Entity, row map[string]any, add bool) (map[string]any, error) {
	list, _ := row["aprovacoes"].([]any)
	uid := atual["id"]
	var out []any
	has := false
	for _, v := range list {
		if fmt.Sprint(v) == fmt.Sprint(uid) {
			has = true
			if add {
				out = append(out, v)
			}
			continue
		}
		out = append(out, v)
	}
	if add && !has {
		out = append(out, uid)
	}
	if out == nil {
		out = []any{}
	}
	res, err := a.in.Op(ctx, e.Singular, "atualizar", row["id"], map[string]any{"aprovacoes": out})
	if err != nil {
		return nil, err
	}
	return res.(map[string]any), nil
}

// reviewView answers the changes (diff from the merge base) and commits.
func (a *intentAPI) reviewView(w http.ResponseWriter, r *http.Request, ctx *interp.Context, e *ast.Entity, row map[string]any, op string) {
	repo, _ := a.reviewRepo(ctx, e, row)
	src := "refs/heads/" + fmt.Sprint(row[e.Review.Source])
	dst := "refs/heads/" + fmt.Sprint(row[e.Review.Target])
	if sha, ok := row["commit_mesclagem"].(string); ok && sha != "" {
		// after merging, compare the merge commit with its first parent
		if c, err := a.s.Git.GetCommit(repo, sha); err == nil && len(c.ParentIDs) == 2 {
			dst, src = c.ParentIDs[0], c.ParentIDs[1]
		}
	}
	base, err := a.s.Git.MergeBase(repo, dst, src)
	if err != nil {
		a.fail(w, 404, err.Error())
		return
	}
	if op == "revisao_commits" {
		list, err := a.s.Git.Log(repo, src, base, "", 250, 0)
		if err != nil {
			a.fail(w, 400, err.Error())
			return
		}
		out := []any{}
		for i := range list {
			out = append(out, commitJSON(&list[i]))
		}
		a.json(w, 200, out, nil)
		return
	}
	files, _, err := a.s.Git.Diff(repo, base, src, 1000)
	if err != nil {
		a.fail(w, 400, err.Error())
		return
	}
	out := serialize(e, row)
	out["mudancas"] = diffJSON(files)
	a.json(w, 200, out, nil)
}

// protectedByData: branches named by the records of a data (GEP 0016),
// `*` standing for any text, need the declared role to change.
func (a *intentAPI) protectedByData(ctx *interp.Context, atual map[string]any, e *ast.Entity, row map[string]any, updates []git.RefUpdate) error {
	if len(e.ProtectedBranches) == 0 || a.in.IsAdmin(atual) {
		return nil
	}
	for _, pb := range e.ProtectedBranches {
		res, err := a.in.Op(ctx, pb.Data, "filtrar", map[string]any{pb.OwnerField: row["id"]}, map[string]any{"limite": 500})
		if err != nil {
			return err
		}
		var patterns []string
		for _, it := range res.([]any) {
			patterns = append(patterns, toStr(it.(map[string]any)["nome"]))
		}
		for _, u := range updates {
			branch, ok := strings.CutPrefix(u.Ref, "refs/heads/")
			if !ok {
				continue
			}
			for _, p := range patterns {
				if globMatch(p, branch) && a.in.Level(ctx, atual, e, row) < a.app.Level(pb.Role) {
					msg := fmt.Sprintf("Somente %s pode enviar código para a branch protegida %s", pb.Role, branch)
					if a.app.Messages == "en" {
						msg = "You are not allowed to push code to protected branches on this project."
					}
					return &interp.RuntimeError{Status: 403, Message: msg}
				}
			}
		}
	}
	return nil
}

// globMatch: `*` stands for any text (slashes included), everything else
// must be equal.
func globMatch(pattern, s string) bool {
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == s
	}
	if !strings.HasPrefix(s, parts[0]) {
		return false
	}
	s = s[len(parts[0]):]
	for i, p := range parts[1:] {
		last := i == len(parts)-2
		if last {
			return strings.HasSuffix(s, p)
		}
		k := strings.Index(s, p)
		if k < 0 {
			return false
		}
		s = s[k+len(p):]
	}
	return true
}

// updatesToMaps describes ref updates the way hooks and executions read them.
func updatesToMaps(list []git.RefUpdate) []any {
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
