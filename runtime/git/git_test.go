package git

import (
	"strings"
	"testing"
)

// Copy: branches, tags and the default branch come along; the copy changes
// on its own and keeps no link to the original.
func TestCopy(t *testing.T) {
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Init("a.git", "trunk"); err != nil {
		t.Fatal(err)
	}
	sig := Signature{Name: "A", Email: "a@x"}
	first, err := s.CommitFiles("a.git", "trunk", "", "primeiro", sig, []Action{{Kind: "create", Path: "a.txt", Content: []byte("a")}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateBranch("a.git", "feature", "trunk"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateTag("a.git", "v1", "trunk"); err != nil {
		t.Fatal(err)
	}
	if err := s.Copy("a.git", "copias/b.git"); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Resolve("copias/b.git", "refs/heads/trunk"); got != first {
		t.Fatalf("trunk da cópia = %q, esperado %q", got, first)
	}
	if !s.BranchExists("copias/b.git", "feature") {
		t.Fatal("a branch feature não veio na cópia")
	}
	if tags, _ := s.Tags("copias/b.git"); len(tags) != 1 || tags[0].Name != "v1" {
		t.Fatalf("tags da cópia: %v", tags)
	}
	p, _ := s.Path("copias/b.git")
	if out, _ := s.run(p, nil, nil, "symbolic-ref", "HEAD"); strings.TrimSpace(string(out)) != "refs/heads/trunk" {
		t.Fatalf("HEAD da cópia = %q", out)
	}
	if out, _ := s.run(p, nil, nil, "config", "--get-regexp", `^remote\.`); len(out) != 0 {
		t.Fatalf("a cópia guarda o caminho do original: %s", out)
	}
	// A commit in the copy does not reach the original.
	if _, err := s.CommitFiles("copias/b.git", "trunk", "", "segundo", sig, []Action{{Kind: "create", Path: "b.txt", Content: []byte("b")}}); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Resolve("a.git", "refs/heads/trunk"); got != first {
		t.Fatal("um commit na cópia mudou o original")
	}
	// Removing the original keeps the copy readable.
	if err := s.Remove("a.git"); err != nil {
		t.Fatal(err)
	}
	if b, err := s.ReadFile("copias/b.git", "trunk", "a.txt", 1<<20); err != nil || string(b.Content) != "a" {
		t.Fatalf("cópia sem o original: %v %v", b, err)
	}
	// An empty repository copies too, with its default branch.
	if err := s.Init("vazio.git", "principal"); err != nil {
		t.Fatal(err)
	}
	if err := s.Copy("vazio.git", "vazio2.git"); err != nil {
		t.Fatal(err)
	}
	p, _ = s.Path("vazio2.git")
	if out, _ := s.run(p, nil, nil, "symbolic-ref", "HEAD"); strings.TrimSpace(string(out)) != "refs/heads/principal" {
		t.Fatalf("HEAD da cópia vazia = %q", out)
	}
	// Refused: an existing destination, a missing source, paths outside the root.
	if err := s.Copy("vazio.git", "vazio2.git"); err == nil {
		t.Fatal("copiou por cima de um repositório existente")
	}
	if err := s.Copy("nao-existe.git", "x.git"); err == nil {
		t.Fatal("copiou um repositório inexistente")
	}
	if err := s.Copy("vazio.git", "../fora.git"); err == nil {
		t.Fatal("copiou para fora da raiz")
	}
	if err := s.Copy("-x.git", "y.git"); err == nil {
		t.Fatal("aceitou um nome com cara de opção")
	}
}

// Tags: created at a revision, never moved, removed by name.
func TestTags(t *testing.T) {
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Init("r.git", "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CommitFiles("r.git", "main", "", "primeiro", Signature{Name: "A", Email: "a@x"}, []Action{{Kind: "create", Path: "a.txt", Content: []byte("a")}}); err != nil {
		t.Fatal(err)
	}
	id, err := s.CreateTag("r.git", "v1", "main")
	if err != nil || id == "" {
		t.Fatalf("criar tag: %q %v", id, err)
	}
	if _, err := s.CreateTag("r.git", "v1", "main"); err == nil {
		t.Fatal("uma tag existente foi movida")
	}
	if _, err := s.CreateTag("r.git", "../fora", "main"); err == nil {
		t.Fatal("nome de tag inválido aceito")
	}
	if list, _ := s.Tags("r.git"); len(list) != 1 || list[0].Name != "v1" {
		t.Fatalf("tags: %v", list)
	}
	if err := s.DeleteTag("r.git", "v1"); err != nil {
		t.Fatal(err)
	}
	if list, _ := s.Tags("r.git"); len(list) != 0 {
		t.Fatalf("tag removida continua: %v", list)
	}
}
