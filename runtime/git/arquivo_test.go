package git

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"testing"
)

// Archive: the tree of a revision as zip, tar or tar.gz, every path under
// the prefix; bad revisions, formats and prefixes are refused before git runs.
func TestArchive(t *testing.T) {
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Init("r.git", "main"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CommitFiles("r.git", "main", "", "primeiro", Signature{Name: "A", Email: "a@x"}, []Action{
		{Kind: "create", Path: "README.md", Content: []byte("# Olá")},
		{Kind: "create", Path: "src/main.go", Content: []byte("package main")},
	}); err != nil {
		t.Fatal(err)
	}

	var zbuf bytes.Buffer
	if err := s.Archive(context.Background(), "r.git", "main", "zip", "lib-main/", &zbuf); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(zbuf.Bytes()), int64(zbuf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, _ := f.Open()
		b, _ := io.ReadAll(rc)
		rc.Close()
		got[f.Name] = string(b)
	}
	if len(got) != 2 || got["lib-main/README.md"] != "# Olá" || got["lib-main/src/main.go"] != "package main" {
		t.Fatalf("zip: %v", got)
	}

	for _, format := range []string{"tar.gz", "tar"} {
		var buf bytes.Buffer
		if err := s.Archive(context.Background(), "r.git", "main", format, "lib-main/", &buf); err != nil {
			t.Fatal(err)
		}
		var rd io.Reader = &buf
		if format == "tar.gz" {
			gz, err := gzip.NewReader(&buf)
			if err != nil {
				t.Fatalf("%s: %v", format, err)
			}
			rd = gz
		}
		tr := tar.NewReader(rd)
		got := map[string]string{}
		for {
			h, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("%s: %v", format, err)
			}
			if h.Typeflag == tar.TypeReg {
				b, _ := io.ReadAll(tr)
				got[h.Name] = string(b)
			}
		}
		if len(got) != 2 || got["lib-main/src/main.go"] != "package main" {
			t.Fatalf("%s: %v", format, got)
		}
	}

	var sink bytes.Buffer
	for _, c := range []struct{ rev, format, prefix string }{
		{"nao-existe", "zip", "x/"},
		{"--output=/tmp/x", "zip", "x/"},
		{"main", "rar", "x/"},
		{"main", "zip", "../x/"},
		{"main", "zip", "--x/"},
		{"main", "zip", "a/b/"},
	} {
		if err := s.Archive(context.Background(), "r.git", c.rev, c.format, c.prefix, &sink); err == nil {
			t.Fatalf("aceito: %+v", c)
		}
	}
	if sink.Len() != 0 {
		t.Fatalf("bytes escritos antes de recusar: %d", sink.Len())
	}
}
