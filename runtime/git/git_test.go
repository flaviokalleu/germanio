package git

import "testing"

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
