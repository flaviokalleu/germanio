package e2e

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Um gitlab-runner oficial (executor shell) pega os jobs pela API v4,
// clona com o token do job, envia o log e o resultado.
// Defina GITLAB_RUNNER_BIN com o caminho do binário para executar.
func TestRunnerOficial(t *testing.T) {
	bin := os.Getenv("GITLAB_RUNNER_BIN")
	if bin == "" {
		t.Skip("GITLAB_RUNNER_BIN não definido")
	}
	base := gitlab(t)
	anon := &api{t: t, base: base}
	tok := anon.must("POST", "/oauth/token", map[string]any{"grant_type": "password", "username": "root", "password": "rootpassword1"}, 200)
	root := &api{t: t, base: base, token: tok["access_token"].(string)}
	runner := root.must("POST", "/api/v4/runners", map[string]any{"description": "shell"}, 201)
	rtoken, _ := runner["token"].(string)
	if rtoken == "" {
		t.Fatalf("token do runner: %v", runner)
	}
	ada := signup(t, base, "ada")
	ada.must("POST", "/api/v4/runners", map[string]any{"description": "x"}, 403)
	anon.must("POST", "/api/v4/runners/verify", map[string]any{"token": "glrt-falso"}, 403)
	anon.must("POST", "/api/v4/runners/verify", map[string]any{"token": rtoken}, 200)

	p := ada.must("POST", "/api/v4/projects", map[string]any{"name": "App", "path": "app", "initialize_with_readme": true}, 201)
	pid := id(p)
	ada.must("POST", "/api/v4/projects/"+pid+"/variables", map[string]any{"key": "DEPLOY_TOKEN", "value": "s3cr3t-valor"}, 201)
	pat := ada.must("POST", "/api/v4/personal_access_tokens", map[string]any{"name": "git"}, 201)
	u, _ := url.Parse(base)
	u.User = url.UserPassword("ada", pat["token"].(string))
	dir := t.TempDir()
	run(t, dir, "git", "clone", "--quiet", u.String()+"/ada/app.git", "app")
	work := filepath.Join(dir, "app")
	run(t, work, "git", "config", "user.email", "ada@example.com")
	run(t, work, "git", "config", "user.name", "Ada")
	os.WriteFile(filepath.Join(work, ".gitlab-ci.yml"), []byte("stages: [build, test]\nbuild:\n  stage: build\n  script:\n    - echo compilando $CI_COMMIT_REF_NAME em $CI_PROJECT_PATH\n    - test -f README.md\n    - echo resultado do build > saida.txt\n    - echo token=$DEPLOY_TOKEN\n    - test \"$DEPLOY_TOKEN\" = s3cr3t-valor\n  artifacts:\n    paths:\n      - saida.txt\ntest:\n  stage: test\n  script:\n    - echo falhando de propósito\n    - exit 3\n"), 0o644)
	run(t, work, "git", "add", ".")
	run(t, work, "git", "commit", "--quiet", "-m", "ci")
	run(t, work, "git", "push", "--quiet", "origin", "main")
	pipe := id(ada.list("/api/v4/projects/" + pid + "/pipelines")[0].(map[string]any))

	home := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "run-single", "--url", base, "--token", rtoken, "--executor", "shell",
		"--max-builds", "2", "--wait-timeout", "60", "--builds-dir", filepath.Join(home, "builds"), "--cache-dir", filepath.Join(home, "cache"))
	cmd.Env = append(os.Environ(), "HOME="+home, "GIT_CONFIG_NOSYSTEM=1")
	var out strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cancel()
		cmd.Wait()
		if t.Failed() {
			t.Logf("saída do runner:\n%s", out.String())
		}
	}()

	done := waitStateFor(t, ada, "/api/v4/projects/"+pid+"/pipelines/"+pipe, 80*time.Second, "success", "failed")
	if done["state"] != "failed" {
		t.Fatalf("pipeline: %v", done)
	}
	b := jobByName(t, ada, pid, pipe, "build")
	if b["state"] != "success" {
		t.Fatalf("build: %v", b)
	}
	if log := trace(t, ada, "/api/v4/jobs/"+id(b)+"/trace"); !strings.Contains(log, "compilando main em ada/app") || !strings.Contains(log, "Job succeeded") || strings.Contains(log, "s3cr3t-valor") || !strings.Contains(log, "[MASKED]") {
		t.Fatalf("log do build:\n%s", log)
	}
	// CI-08: the runner uploads the artifacts (GEP 0014); they come back as a
	// zip to whoever may see the job, never to anyone else
	code, zipped := download(t, ada, "/api/v4/jobs/"+id(b)+"/artifacts")
	if code != 200 {
		t.Fatalf("artefatos do build: %d", code)
	}
	zr, err := zip.NewReader(bytes.NewReader(zipped), int64(len(zipped)))
	if err != nil {
		t.Fatalf("artefatos não são um zip: %v", err)
	}
	found := false
	for _, zf := range zr.File {
		if zf.Name == "saida.txt" {
			rc, _ := zf.Open()
			content, _ := io.ReadAll(rc)
			rc.Close()
			found = strings.Contains(string(content), "resultado do build")
		}
	}
	if !found {
		t.Fatalf("saida.txt não está nos artefatos: %v", zr.File)
	}
	eve := signup(t, base, "eve")
	if code, _ := download(t, eve, "/api/v4/jobs/"+id(b)+"/artifacts"); code != 404 {
		t.Fatalf("artefatos de projeto privado para quem não é membro: %d", code)
	}
	f := jobByName(t, ada, pid, pipe, "test")
	if f["state"] != "failed" {
		t.Fatalf("test: %v", f)
	}
	if log := trace(t, ada, "/api/v4/jobs/"+id(f)+"/trace"); !strings.Contains(log, "falhando de propósito") || !strings.Contains(log, "exit status 3") {
		t.Fatalf("log do test:\n%s", log)
	}
}

// O protocolo do runner sem o binário: segurança do token do job.
func TestProtocoloRunner(t *testing.T) {
	base := gitlab(t)
	anon := &api{t: t, base: base}
	tok := anon.must("POST", "/oauth/token", map[string]any{"grant_type": "password", "username": "root", "password": "rootpassword1"}, 200)
	root := &api{t: t, base: base, token: tok["access_token"].(string)}
	ada := signup(t, base, "ada")
	bob := signup(t, base, "bob")
	ada.must("POST", "/api/v4/projects", map[string]any{"name": "App", "path": "app", "initialize_with_readme": true}, 201)
	other := bob.must("POST", "/api/v4/projects", map[string]any{"name": "Other", "path": "other", "initialize_with_readme": true}, 201)

	pushCI := func(owner *api, user, path string) {
		pat := owner.must("POST", "/api/v4/personal_access_tokens", map[string]any{"name": "git"}, 201)
		u, _ := url.Parse(base)
		u.User = url.UserPassword(user, pat["token"].(string))
		dir := t.TempDir()
		run(t, dir, "git", "clone", "--quiet", u.String()+"/"+path+".git", "w")
		work := filepath.Join(dir, "w")
		run(t, work, "git", "config", "user.email", user+"@example.com")
		run(t, work, "git", "config", "user.name", user)
		os.WriteFile(filepath.Join(work, ".gitlab-ci.yml"), []byte("job:\n  script: echo oi\n"), 0o644)
		run(t, work, "git", "add", ".")
		run(t, work, "git", "commit", "--quiet", "-m", "ci")
		run(t, work, "git", "push", "--quiet", "origin", "main")
	}
	pushCI(ada, "ada", "ada/app")

	// Runner do projeto de Bob não pega o job de Ada.
	bobRunner := root.must("POST", "/api/v4/runners", map[string]any{"description": "bob", "project_id": other["id"]}, 201)
	if code, _, _ := anon.call("POST", "/api/v4/jobs/request", map[string]any{"token": bobRunner["token"]}); code != 204 {
		t.Fatalf("runner de outro projeto pegou job: %d", code)
	}
	off := root.must("POST", "/api/v4/runners", map[string]any{"description": "off"}, 201)
	root.must("PUT", "/api/v4/runners/"+id(off), map[string]any{"active": false}, 200)
	anon.must("POST", "/api/v4/jobs/request", map[string]any{"token": off["token"]}, 403)

	shared := root.must("POST", "/api/v4/runners", map[string]any{"description": "shared"}, 201)
	job := anon.must("POST", "/api/v4/jobs/request", map[string]any{"token": shared["token"]}, 201)
	jobToken := job["token"].(string)
	jid := jsonNum(job["id"])
	if code, _, _ := anon.call("POST", "/api/v4/jobs/request", map[string]any{"token": shared["token"]}); code != 204 {
		t.Fatalf("o mesmo job foi entregue duas vezes: %d", code)
	}

	u, _ := url.Parse(base)
	u.User = url.UserPassword("gitlab-ci-token", jobToken)
	work := filepath.Join(t.TempDir(), "c")
	run(t, filepath.Dir(work), "git", "clone", "--quiet", u.String()+"/ada/app.git", "c")
	runFails(t, t.TempDir(), "git", "clone", "--quiet", u.String()+"/bob/other.git", "x") // outro projeto
	run(t, work, "git", "config", "user.email", "ci@example.com")
	run(t, work, "git", "config", "user.name", "ci")
	os.WriteFile(filepath.Join(work, "x.txt"), []byte("x"), 0o644)
	run(t, work, "git", "add", ".")
	run(t, work, "git", "commit", "--quiet", "-m", "x")
	runFails(t, work, "git", "push", "--quiet", "origin", "main") // token do job não envia código

	trace := func(start int, text, token string) int {
		req, _ := http.NewRequest("PATCH", base+"/api/v4/jobs/"+jid+"/trace", strings.NewReader(text))
		req.Header.Set("JOB-TOKEN", token)
		req.Header.Set("Content-Type", "text/plain")
		req.Header.Set("Content-Range", strconv.Itoa(start)+"-"+strconv.Itoa(start+len(text)-1))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	if c := trace(0, "linha 1\n", jobToken); c != 202 {
		t.Fatalf("trace: %d", c)
	}
	if c := trace(0, "repetida\n", jobToken); c != 416 {
		t.Fatalf("trace fora de ordem: %d", c)
	}
	if c := trace(8, "linha 2\n", "token-falso"); c != 403 {
		t.Fatalf("trace com token falso: %d", c)
	}
	anon.must("PUT", "/api/v4/jobs/"+jid, map[string]any{"token": "token-falso", "state": "success"}, 403)
	anon.must("PUT", "/api/v4/jobs/"+jid, map[string]any{"token": jobToken, "state": "success"}, 200)
	if got := ada.must("GET", "/api/v4/jobs/"+jid, nil, 200); got["state"] != "success" {
		t.Fatalf("job: %v", got)
	}
	// Terminado o job: o token não lê mais o repositório; repetir o resultado não muda nada.
	runFails(t, t.TempDir(), "git", "clone", "--quiet", u.String()+"/ada/app.git", "y")
	anon.must("PUT", "/api/v4/jobs/"+jid, map[string]any{"token": jobToken, "state": "failed"}, 200)
	if got := ada.must("GET", "/api/v4/jobs/"+jid, nil, 200); got["state"] != "success" {
		t.Fatalf("resultado mudou depois de concluído: %v", got)
	}
	if c := trace(8, "tarde\n", jobToken); c == 202 {
		t.Fatal("log aceito depois de concluído")
	}
}

func download(t *testing.T, who *api, path string) (int, []byte) {
	t.Helper()
	req, _ := http.NewRequest("GET", who.base+path, nil)
	req.Header.Set("Authorization", "Bearer "+who.token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}
