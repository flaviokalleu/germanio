package e2e

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func waitState(t *testing.T, a *api, path string, want ...string) map[string]any {
	t.Helper()
	return waitStateFor(t, a, path, 30*time.Second, want...)
}

func waitStateFor(t *testing.T, a *api, path string, limit time.Duration, want ...string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(limit)
	for {
		m := a.must("GET", path, nil, 200)
		for _, w := range want {
			if m["state"] == w {
				return m
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s: estado %v, esperado %v", path, m["state"], want)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func trace(t *testing.T, a *api, path string) string {
	req, _ := http.NewRequest("GET", a.base+path, nil)
	req.Header.Set("Authorization", "Bearer "+a.token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}

func jobByName(t *testing.T, a *api, pid, pipe, name string) map[string]any {
	for _, j := range a.list("/api/v4/projects/" + pid + "/pipelines/" + pipe + "/jobs") {
		m := j.(map[string]any)
		if m["name"] == name && m["retried"] != true {
			return m
		}
	}
	t.Fatalf("job %s não encontrado", name)
	return nil
}

// Fluxo 5: pipeline → job → runner → logs → result.
func TestFluxo5Pipelines(t *testing.T) {
	t.Setenv("GERMANIO_EXECUTOR", "local")
	base := gitlab(t)
	ada := signup(t, base, "ada")
	eve := signup(t, base, "eve")
	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "App", "path": "app", "initialize_with_readme": true}, 201)
	pid := id(p)
	pat := ada.must("POST", "/api/v4/personal_access_tokens", map[string]any{"name": "git"}, 201)
	u, _ := url.Parse(base)
	u.User = url.UserPassword("ada", pat["token"].(string))
	dir := t.TempDir()
	run(t, dir, "git", "clone", "--quiet", u.String()+"/ada/app.git", "app")
	work := filepath.Join(dir, "app")
	run(t, work, "git", "config", "user.email", "ada@example.com")
	run(t, work, "git", "config", "user.name", "Ada")
	push := func(ci string) {
		os.WriteFile(filepath.Join(work, ".gitlab-ci.yml"), []byte(ci), 0o644)
		run(t, work, "git", "add", ".")
		run(t, work, "git", "commit", "--quiet", "-m", "ci")
		run(t, work, "git", "push", "--quiet", "origin", "main")
	}

	// 1. sucesso em duas etapas
	push("stages: [build, test]\nbuild:\n  stage: build\n  script:\n    - echo compilando $CI_COMMIT_REF_NAME\n    - echo ok > build.txt\ntest:\n  stage: test\n  script: test -f README.md && echo testes passaram\n")
	pipes := ada.list("/api/v4/projects/" + pid + "/pipelines")
	if len(pipes) != 1 {
		t.Fatalf("push deveria criar 1 pipeline: %v", pipes)
	}
	pipe := id(pipes[0].(map[string]any))
	done := waitState(t, ada, "/api/v4/projects/"+pid+"/pipelines/"+pipe, "success", "failed")
	if done["state"] != "success" || done["ref"] != "main" || done["sha"] == nil {
		t.Fatalf("pipeline: %v", done)
	}
	b := jobByName(t, ada, pid, pipe, "build")
	if log := trace(t, ada, "/api/v4/jobs/"+id(b)+"/trace"); !strings.Contains(log, "compilando main") || !strings.Contains(log, "$ echo compilando") {
		t.Fatalf("log do job: %q", log)
	}
	if tj := jobByName(t, ada, pid, pipe, "test"); tj["state"] != "success" || tj["stage"] != "test" {
		t.Fatalf("job test: %v", tj)
	}

	// 2. falha: etapa seguinte é ignorada; repetir
	push("stages: [test, deploy]\nunit:\n  stage: test\n  script: [\"echo rodando\", \"exit 3\"]\ndeploy:\n  stage: deploy\n  script: echo deploy\n")
	pipe2 := id(ada.list("/api/v4/projects/" + pid + "/pipelines")[0].(map[string]any))
	failed := waitState(t, ada, "/api/v4/projects/"+pid+"/pipelines/"+pipe2, "failed", "success")
	if failed["state"] != "failed" {
		t.Fatalf("pipeline deveria falhar: %v", failed)
	}
	if d := jobByName(t, ada, pid, pipe2, "deploy"); d["state"] != "skipped" {
		t.Fatalf("deploy deveria ser ignorado: %v", d)
	}
	unit := jobByName(t, ada, pid, pipe2, "unit")
	eve.must("POST", "/api/v4/jobs/"+id(unit)+"/retry", nil, 404)
	retried := ada.must("POST", "/api/v4/jobs/"+id(unit)+"/retry", nil, 200)
	if retried["id"] == unit["id"] {
		t.Fatalf("retry deve criar novo job: %v", retried)
	}
	waitState(t, ada, "/api/v4/jobs/"+id(retried), "failed")

	// 3. job manual espera "play"; cancelamento de job longo
	push("stages: [test, deploy]\nslow:\n  stage: test\n  script: sleep 20\nrelease:\n  stage: deploy\n  when: manual\n  script: echo release\n")
	pipe3 := id(ada.list("/api/v4/projects/" + pid + "/pipelines")[0].(map[string]any))
	slow := jobByName(t, ada, pid, pipe3, "slow")
	waitState(t, ada, "/api/v4/jobs/"+id(slow), "running")
	ada.must("POST", "/api/v4/jobs/"+id(slow)+"/cancel", nil, 200)
	waitState(t, ada, "/api/v4/jobs/"+id(slow), "canceled")
	waitState(t, ada, "/api/v4/projects/"+pid+"/pipelines/"+pipe3, "canceled")

	push("stages: [test, deploy]\nquick:\n  stage: test\n  script: echo quick\nrelease:\n  stage: deploy\n  when: manual\n  script: echo released\n")
	pipe4 := id(ada.list("/api/v4/projects/" + pid + "/pipelines")[0].(map[string]any))
	waitState(t, ada, "/api/v4/projects/"+pid+"/pipelines/"+pipe4, "success")
	rel := jobByName(t, ada, pid, pipe4, "release")
	if rel["state"] != "manual" {
		t.Fatalf("job manual: %v", rel)
	}
	ada.must("POST", "/api/v4/jobs/"+id(rel)+"/play", nil, 200)
	waitState(t, ada, "/api/v4/jobs/"+id(rel), "success")

	// 4. configuração inválida gera pipeline com erro
	push("stages: [build]\nbroken:\n  stage: nowhere\n  script: echo x\n")
	bad := ada.list("/api/v4/projects/" + pid + "/pipelines")[0].(map[string]any)
	if bad["state"] != "failed" || !strings.Contains(bad["erro_configuracao"].(string), "nowhere") {
		t.Fatalf("configuração inválida: %v", bad)
	}

	// 5. executar manualmente; permissões
	manual := ada.must("POST", "/api/v4/projects/"+pid+"/pipelines", map[string]any{"ref": "main"}, 201)
	if manual["ref"] != "main" {
		t.Fatalf("pipeline manual: %v", manual)
	}
	eve.must("GET", "/api/v4/projects/"+pid+"/pipelines", nil, 404)
	eve.must("POST", "/api/v4/projects/"+pid+"/pipelines", map[string]any{"ref": "main"}, 404)
	ada.must("POST", "/api/v4/projects/"+pid+"/pipelines", map[string]any{"ref": "nope"}, 400)
}
