package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// NT-03: quem é posto como responsável recebe um e-mail depois de salvo
// (GEP 0013), com o que pode ver.
func TestAvisosPorEmail(t *testing.T) {
	mail := t.TempDir()
	t.Setenv("GERMANIO_CORREIO_PASTA", mail)
	t.Setenv("GERMANIO_URL_PUBLICA", "https://gitlab.example")
	base := gitlab(t)
	ada := signup(t, base, "ada")
	bob := signup(t, base, "bob")
	bobID := bob.must("GET", "/api/v4/user", nil, 200)["id"]
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Tracker", "path": "tracker", "visibility": "public"}, 201)
	ada.must("POST", "/api/v4/projects/"+id(p)+"/issues", map[string]any{"title": "Crash", "assignee_ids": []any{bobID}}, 201)

	var files []string
	for deadline := time.Now().Add(5 * time.Second); len(files) == 0 && time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		files, _ = filepath.Glob(filepath.Join(mail, "*"))
	}
	time.Sleep(200 * time.Millisecond)
	files, _ = filepath.Glob(filepath.Join(mail, "*"))
	if len(files) != 1 {
		t.Fatalf("esperado 1 e-mail (para bob), vieram %d", len(files))
	}
	b, _ := os.ReadFile(files[0])
	if msg := string(b); !strings.Contains(msg, "Para: bob@example.com") || !strings.Contains(msg, "Crash") || !strings.Contains(msg, "https://gitlab.example") {
		t.Fatalf("e-mail de aviso:\n%s", msg)
	}
}
