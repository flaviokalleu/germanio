package e2e

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type delivery struct {
	Event, Token string
	Body         map[string]any
}

func TestWebhooks(t *testing.T) {
	t.Setenv("GERMANIO_PERMITIR_REDE_LOCAL", "1") // receiver runs on localhost in the test
	var mu sync.Mutex
	var got []delivery
	fails := 1
	recv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if fails > 0 { // first delivery fails: the queue must retry
			fails--
			w.WriteHeader(500)
			return
		}
		b, _ := io.ReadAll(r.Body)
		var body map[string]any
		json.Unmarshal(b, &body)
		got = append(got, delivery{r.Header.Get("X-Gitlab-Event"), r.Header.Get("X-Gitlab-Token"), body})
	}))
	defer recv.Close()
	waitFor := func(kind string) delivery {
		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			mu.Lock()
			for _, d := range got {
				if d.Event == kind {
					mu.Unlock()
					return d
				}
			}
			mu.Unlock()
			time.Sleep(100 * time.Millisecond)
		}
		t.Fatalf("nenhuma entrega %q; recebidas: %v", kind, got)
		return delivery{}
	}

	base := gitlab(t)
	ada := signup(t, base, "ada")
	eve := signup(t, base, "eve")
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "Hooks", "path": "hooks", "initialize_with_readme": true}, 201)
	pid := id(p)
	eve.must("POST", "/api/v4/projects/"+pid+"/hooks", map[string]any{"url": recv.URL, "token": "x"}, 404)
	ada.must("POST", "/api/v4/projects/"+pid+"/hooks", map[string]any{"url": "not a url"}, 400)
	h := ada.must("POST", "/api/v4/projects/"+pid+"/hooks", map[string]any{"url": recv.URL, "token": "s3cret", "note_events": false}, 201)
	if h["token"] != nil || h["push_events"] != true || h["note_events"] != false {
		t.Fatalf("webhook: %v", h)
	}

	ada.must("POST", "/api/v4/projects/"+pid+"/issues", map[string]any{"title": "Hooked"}, 201)
	d := waitFor("issue")
	attrs, _ := d.Body["object_attributes"].(map[string]any)
	user, _ := d.Body["user"].(map[string]any)
	proj, _ := d.Body["project"].(map[string]any)
	if d.Token != "s3cret" || d.Body["object_kind"] != "issue" || attrs["title"] != "Hooked" || user["username"] != "ada" || proj["full_path"] != "ada/hooks" {
		t.Fatalf("entrega de issue: %+v", d)
	}
	// comentários desligados para este webhook
	ada.must("POST", "/api/v4/projects/"+pid+"/issues/1/notes", map[string]any{"body": "silencioso"}, 201)

	// push
	pat := ada.must("POST", "/api/v4/personal_access_tokens", map[string]any{"name": "git"}, 201)
	u, _ := url.Parse(base)
	u.User = url.UserPassword("ada", pat["token"].(string))
	dir := t.TempDir()
	run(t, dir, "git", "clone", "--quiet", u.String()+"/ada/hooks.git", "hooks")
	work := filepath.Join(dir, "hooks")
	run(t, work, "git", "config", "user.email", "ada@example.com")
	run(t, work, "git", "config", "user.name", "Ada")
	os.WriteFile(filepath.Join(work, "a.txt"), []byte("a\n"), 0o644)
	run(t, work, "git", "add", ".")
	run(t, work, "git", "commit", "--quiet", "-m", "a")
	run(t, work, "git", "push", "--quiet", "origin", "main")
	push := waitFor("push")
	if push.Body["object_kind"] != "push" {
		t.Fatalf("entrega de push: %+v", push)
	}
	time.Sleep(500 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	for _, d := range got {
		if d.Event == "note" {
			t.Fatalf("evento desligado foi entregue: %+v", d)
		}
	}
}
