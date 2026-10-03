package git

import (
	"errors"
	"testing"
)

// repoWithBranches: main has a.txt; feature (from main) adds two commits;
// main then gets b.txt, so feature is behind main.
func repoWithBranches(t *testing.T) *Store {
	t.Helper()
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Init("r.git", "main"); err != nil {
		t.Fatal(err)
	}
	ada := Signature{Name: "Ada", Email: "ada@x"}
	bob := Signature{Name: "Bob", Email: "bob@x"}
	commit := func(branch, start, msg string, sig Signature, a Action) {
		t.Helper()
		if _, err := s.CommitFiles("r.git", branch, start, msg, sig, []Action{a}); err != nil {
			t.Fatal(err)
		}
	}
	commit("main", "", "primeiro", ada, Action{Kind: "create", Path: "a.txt", Content: []byte("a\n")})
	commit("feature", "main", "um", bob, Action{Kind: "create", Path: "f1.txt", Content: []byte("1\n")})
	commit("feature", "", "dois", bob, Action{Kind: "create", Path: "f2.txt", Content: []byte("2\n")})
	commit("main", "", "outro", ada, Action{Kind: "create", Path: "b.txt", Content: []byte("b\n")})
	return s
}

func paths(t *testing.T, s *Store, rev string) map[string]bool {
	t.Helper()
	list, err := s.Tree("r.git", rev, "")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, e := range list {
		out[e.Path] = true
	}
	return out
}

// Squash: one new commit on top of the target with every change of the
// source; the source keeps its own commits.
func TestSquash(t *testing.T) {
	s := repoWithBranches(t)
	before, _ := s.Resolve("r.git", "refs/heads/main")
	id, err := s.Squash("r.git", "main", "refs/heads/feature", "Recurso\n", Signature{Name: "Ada", Email: "ada@x"})
	if err != nil {
		t.Fatal(err)
	}
	c, _ := s.GetCommit("r.git", id)
	if len(c.ParentIDs) != 1 || c.ParentIDs[0] != before || c.Title != "Recurso" {
		t.Fatalf("commit juntado: %+v", c)
	}
	if p := paths(t, s, "main"); !p["a.txt"] || !p["b.txt"] || !p["f1.txt"] || !p["f2.txt"] {
		t.Fatalf("árvore depois de juntar: %v", p)
	}
	if log, _ := s.Log("r.git", "refs/heads/feature", "", "", 10, 0); len(log) != 3 {
		t.Fatalf("a origem mudou: %d commits", len(log))
	}
}

// Rebase replays the source's commits on the target, keeping authors and
// messages; then the target only fast-forwards: a linear history.
func TestRebaseFastForward(t *testing.T) {
	s := repoWithBranches(t)
	if _, err := s.FastForward("r.git", "main", "refs/heads/feature"); !errors.Is(err, ErrNotFastForward) {
		t.Fatalf("avanço sem a origem estar em dia: %v", err)
	}
	main, _ := s.Resolve("r.git", "refs/heads/main")
	head, err := s.Rebase("r.git", "feature", "refs/heads/main", Signature{Name: "Ada", Email: "ada@x"})
	if err != nil {
		t.Fatal(err)
	}
	log, _ := s.Log("r.git", head, main, "", 10, 0)
	if len(log) != 2 || log[0].Title != "dois" || log[1].Title != "um" || log[0].AuthorName != "Bob" || log[0].CommitterName != "Ada" {
		t.Fatalf("commits reaplicados: %+v", log)
	}
	if ok, _ := s.IsAncestor("r.git", main, head); !ok {
		t.Fatal("a origem não ficou sobre o destino")
	}
	// rebasing again changes nothing
	if again, err := s.Rebase("r.git", "feature", "refs/heads/main", Signature{Name: "Ada"}); err != nil || again != head {
		t.Fatalf("segundo rebase: %s %v", again, err)
	}
	id, err := s.FastForward("r.git", "main", "refs/heads/feature")
	if err != nil || id != head {
		t.Fatalf("avanço: %s %v", id, err)
	}
	c, _ := s.GetCommit("r.git", "main")
	if len(c.ParentIDs) != 1 {
		t.Fatalf("histórico linear tem commit de mescla: %+v", c)
	}
}

// A commit that does not apply refuses the rebase and names the file; the
// branch stays where it was.
func TestRebaseConflict(t *testing.T) {
	s := repoWithBranches(t)
	sig := Signature{Name: "Ada", Email: "ada@x"}
	if _, err := s.CommitFiles("r.git", "main", "", "muda a", sig, []Action{{Kind: "update", Path: "a.txt", Content: []byte("main\n")}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CommitFiles("r.git", "feature", "", "muda a também", sig, []Action{{Kind: "update", Path: "a.txt", Content: []byte("feature\n")}}); err != nil {
		t.Fatal(err)
	}
	before, _ := s.Resolve("r.git", "refs/heads/feature")
	_, err := s.Rebase("r.git", "feature", "refs/heads/main", sig)
	var conf *ErrConflict
	if !errors.As(err, &conf) || len(conf.Files) != 1 || conf.Files[0] != "a.txt" {
		t.Fatalf("conflito: %v", err)
	}
	if after, _ := s.Resolve("r.git", "refs/heads/feature"); after != before {
		t.Fatal("a branch mudou num rebase recusado")
	}
	if _, err := s.Squash("r.git", "main", "refs/heads/feature", "x", sig); !errors.As(err, &conf) {
		t.Fatalf("juntar com conflito: %v", err)
	}
}
