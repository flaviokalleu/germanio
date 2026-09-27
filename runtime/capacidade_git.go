package runtime

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/flaviokalleu/germanio/runtime/git"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// The git capability: repositories declared with `tem repositório` are
// managed automatically; these functions give level-3 code (hooks, custom
// actions) access to their content. Names follow the Germanio vocabulary.

func init() {
	RegistrarCapability(func(app *App) error {
		root := os.Getenv("GERMANIO_GIT_RAIZ")
		if root == "" {
			root = filepath.Join(filepath.Dir(app.Program.Filename), "repositorios")
		}
		needs := false
		if app.Program.App != nil {
			for _, e := range app.Program.App.Entities {
				needs = needs || e.Repository
			}
		}
		store, err := git.NewStore(root)
		if err != nil {
			if needs {
				return err
			}
			return nil // git is optional for apps without repositories
		}
		app.Server.Git = store
		app.Interpreter.RegisterModule("git", gitModule(store))
		return nil
	})
}

func commitMap(c *git.Commit) map[string]any {
	if c == nil {
		return nil
	}
	parents := make([]any, len(c.ParentIDs))
	for i, p := range c.ParentIDs {
		parents[i] = p
	}
	return map[string]any{"id": c.ID, "short_id": c.ShortID, "title": c.Title, "message": c.Message,
		"author_name": c.AuthorName, "author_email": c.AuthorEmail, "authored_date": c.AuthorAt,
		"committer_name": c.CommitterName, "committer_email": c.CommitterEmail, "committed_date": c.CommittedAt,
		"parent_ids": parents}
}

func diffMaps(files []git.FileDiff) []any {
	out := make([]any, len(files))
	for i, f := range files {
		out[i] = map[string]any{"old_path": f.OldPath, "new_path": f.NewPath, "new_file": f.NewFile, "deleted_file": f.DeletedFile,
			"renamed_file": f.RenamedFile, "binary": f.Binary, "diff": f.Diff, "additions": float64(f.Additions), "deletions": float64(f.Deletions)}
	}
	return out
}

func gitModule(s *git.Store) map[string]interp.ModuleFunc {
	fail := func(c *interp.Call, err error) {
		var inv *git.ErrInvalid
		var conf *git.ErrConflict
		switch {
		case errors.Is(err, git.ErrNotFound):
			panic(c.Fail(404, "%s", err.Error()))
		case errors.As(err, &inv):
			panic(c.Fail(400, "%s", err.Error()))
		case errors.As(err, &conf):
			panic(c.Fail(409, "%s", err.Error()))
		}
		panic(c.Fail(0, "%s: %s", c.Name, err.Error()))
	}
	sig := func(c *interp.Call, m map[string]any) git.Signature {
		return git.Signature{Name: toStr(m["nome"]), Email: toStr(m["email"])}
	}
	return map[string]interp.ModuleFunc{
		"criar": func(c *interp.Call, a []any) any {
			branch := "main"
			if len(a) > 1 {
				branch = c.Str(a, 1, "branch")
			}
			if err := s.Init(c.Str(a, 0, "repositório"), branch); err != nil {
				fail(c, err)
			}
			return true
		},
		"existe": func(c *interp.Call, a []any) any { return s.Exists(c.Str(a, 0, "repositório")) },
		"vazio": func(c *interp.Call, a []any) any {
			ok, err := s.IsEmpty(c.Str(a, 0, "repositório"))
			if err != nil {
				fail(c, err)
			}
			return ok
		},
		"branches": func(c *interp.Call, a []any) any {
			list, err := s.Branches(c.Str(a, 0, "repositório"))
			if err != nil {
				fail(c, err)
			}
			out := make([]any, len(list))
			for i, b := range list {
				cm := b.Commit
				out[i] = map[string]any{"name": b.Name, "commit": commitMap(&cm)}
			}
			return out
		},
		"existe_branch": func(c *interp.Call, a []any) any {
			return s.BranchExists(c.Str(a, 0, "repositório"), c.Str(a, 1, "branch"))
		},
		"criar_branch": func(c *interp.Call, a []any) any {
			id, err := s.CreateBranch(c.Str(a, 0, "repositório"), c.Str(a, 1, "nome"), c.Str(a, 2, "origem"))
			if err != nil {
				fail(c, err)
			}
			return id
		},
		"remover_branch": func(c *interp.Call, a []any) any {
			if err := s.DeleteBranch(c.Str(a, 0, "repositório"), c.Str(a, 1, "branch")); err != nil {
				fail(c, err)
			}
			return true
		},
		"commit": func(c *interp.Call, a []any) any {
			cm, err := s.GetCommit(c.Str(a, 0, "repositório"), c.Str(a, 1, "revisão"))
			if errors.Is(err, git.ErrNotFound) {
				return nil
			}
			if err != nil {
				fail(c, err)
			}
			return commitMap(cm)
		},
		// commits(repo, ref, {caminho, desde, limite, pagina})
		"commits": func(c *interp.Call, a []any) any {
			opts := c.OptMap(a, 2, "opções")
			limit := int(asNum(opts["limite"], 20))
			page := int(asNum(opts["pagina"], 1))
			list, err := s.Log(c.Str(a, 0, "repositório"), c.Str(a, 1, "revisão"), toStr(opts["desde"]), toStr(opts["caminho"]), limit, (page-1)*limit)
			if err != nil {
				fail(c, err)
			}
			out := make([]any, len(list))
			for i := range list {
				out[i] = commitMap(&list[i])
			}
			return out
		},
		"arvore": func(c *interp.Call, a []any) any {
			path := ""
			if len(a) > 2 && a[2] != nil {
				path = c.Str(a, 2, "caminho")
			}
			list, err := s.Tree(c.Str(a, 0, "repositório"), c.Str(a, 1, "revisão"), path)
			if err != nil {
				fail(c, err)
			}
			out := make([]any, len(list))
			for i, e := range list {
				out[i] = map[string]any{"id": e.ID, "name": e.Name, "type": e.Type, "path": e.Path, "mode": e.Mode, "size": float64(e.Size)}
			}
			return out
		},
		"arquivo": func(c *interp.Call, a []any) any {
			b, err := s.ReadFile(c.Str(a, 0, "repositório"), c.Str(a, 1, "revisão"), c.Str(a, 2, "caminho"), 5<<20)
			if errors.Is(err, git.ErrNotFound) {
				return nil
			}
			if err != nil {
				fail(c, err)
			}
			content := string(b.Content)
			if b.Binary {
				content = ""
			}
			return map[string]any{"id": b.ID, "path": b.Path, "size": float64(b.Size), "binary": b.Binary, "truncated": b.Truncated, "content": content}
		},
		"diff": func(c *interp.Call, a []any) any {
			from := ""
			if a[1] != nil {
				from = c.Str(a, 1, "de")
			}
			files, _, err := s.Diff(c.Str(a, 0, "repositório"), from, c.Str(a, 2, "até"), 1000)
			if err != nil {
				fail(c, err)
			}
			return diffMaps(files)
		},
		"base_comum": func(c *interp.Call, a []any) any {
			id, err := s.MergeBase(c.Str(a, 0, "repositório"), c.Str(a, 1, "a"), c.Str(a, 2, "b"))
			if err != nil {
				fail(c, err)
			}
			return id
		},
		"e_ancestral": func(c *interp.Call, a []any) any {
			ok, err := s.IsAncestor(c.Str(a, 0, "repositório"), c.Str(a, 1, "a"), c.Str(a, 2, "b"))
			if err != nil {
				fail(c, err)
			}
			return ok
		},
		// pode_mesclar(repo, destino, origem) → {pode, conflitos}
		"pode_mesclar": func(c *interp.Call, a []any) any {
			m, err := s.CheckMerge(c.Str(a, 0, "repositório"), c.Str(a, 1, "destino"), c.Str(a, 2, "origem"))
			if err != nil {
				fail(c, err)
			}
			conf := make([]any, len(m.Conflicts))
			for i, f := range m.Conflicts {
				conf[i] = f
			}
			return map[string]any{"pode": m.CanMerge, "conflitos": conf}
		},
		// mesclar(repo, destino, origem, mensagem, {nome, email}) → id do commit
		"mesclar": func(c *interp.Call, a []any) any {
			id, err := s.Merge(c.Str(a, 0, "repositório"), c.Str(a, 1, "destino"), c.Str(a, 2, "origem"), c.Str(a, 3, "mensagem"), sig(c, c.Map(a, 4, "autor")))
			if err != nil {
				fail(c, err)
			}
			return id
		},
		// commitar(repo, {branch, inicio, mensagem, autor, acoes: [{acao, caminho, conteudo, anterior}]})
		"commitar": func(c *interp.Call, a []any) any {
			o := c.Map(a, 1, "commit")
			var actions []git.Action
			list, _ := o["acoes"].([]any)
			for _, it := range list {
				m, _ := it.(map[string]any)
				kind := toStr(m["acao"])
				switch kind {
				case "criar":
					kind = "create"
				case "atualizar":
					kind = "update"
				case "excluir":
					kind = "delete"
				case "mover":
					kind = "move"
				}
				var content []byte
				if v, ok := m["conteudo"]; ok && v != nil {
					content = []byte(toStr(v))
				}
				actions = append(actions, git.Action{Kind: kind, Path: toStr(m["caminho"]), Previous: toStr(m["anterior"]), Content: content})
			}
			autor, _ := o["autor"].(map[string]any)
			id, err := s.CommitFiles(c.Str(a, 0, "repositório"), toStr(o["branch"]), toStr(o["inicio"]), toStr(o["mensagem"]), sig(c, autor), actions)
			if err != nil {
				fail(c, err)
			}
			return id
		},
	}
}

func toStr(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func asNum(v any, d float64) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return d
}
