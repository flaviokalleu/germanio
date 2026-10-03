package e2e

import (
	"archive/zip"
	"bytes"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// CI-09 e CI-08: needs (grafo), rules/only/except pela branch, when: manual,
// allow_failure, artefatos entre jobs e expiração, com o executor local.
func TestPipelineGrafoERegras(t *testing.T) {
	t.Setenv("GERMANIO_EXECUTOR", "local")
	base := gitlab(t)
	ada := signup(t, base, "ada")
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
	commit := func(branch, ci string) {
		os.WriteFile(filepath.Join(work, ".gitlab-ci.yml"), []byte(ci), 0o644)
		run(t, work, "git", "add", ".")
		run(t, work, "git", "commit", "--quiet", "--allow-empty", "-m", "ci "+branch)
		run(t, work, "git", "push", "--quiet", "origin", "HEAD:"+branch)
	}
	push := func(branch, ci string) string {
		commit(branch, ci)
		return id(ada.list("/api/v4/projects/" + pid + "/pipelines")[0].(map[string]any))
	}
	names := func(pipe string) map[string]map[string]any {
		out := map[string]map[string]any{}
		for _, j := range ada.list("/api/v4/projects/" + pid + "/pipelines/" + pipe + "/jobs") {
			m := j.(map[string]any)
			out[m["name"].(string)] = m
		}
		return out
	}
	ci := `stages: [build, test, deploy]
build_a:
  stage: build
  script:
    - mkdir -p out
    - echo A > out/a.txt
  artifacts:
    paths: [out/]
    expire_in: 1 week
build_b:
  stage: build
  script: sleep 5
test_a:
  stage: test
  needs: [build_a]
  script:
    - test "$(cat out/a.txt)" = A
instavel:
  stage: test
  script: exit 1
  allow_failure: true
deploy:
  stage: deploy
  script: echo deploy
  rules:
    - if: $CI_COMMIT_BRANCH == "main"
      when: manual
so_feature:
  stage: test
  script: echo feature
  only: [/^feature-.*$/]
fora_da_main:
  stage: test
  script: echo fora
  except: [main]
`
	// 1. na main: rules e only/except escolhem os jobs; test_a não espera build_b
	pipe := push("main", ci)
	jobs := names(pipe)
	if len(jobs) != 5 || jobs["so_feature"] != nil || jobs["fora_da_main"] != nil || jobs["deploy"] == nil {
		t.Fatalf("jobs da main: %v", jobs)
	}
	ta := waitState(t, ada, "/api/v4/jobs/"+id(jobs["test_a"]), "success", "failed", "skipped")
	log := trace(t, ada, "/api/v4/jobs/"+id(jobs["test_a"])+"/trace")
	if ta["state"] != "success" || !strings.Contains(log, "Recebendo artefatos de build_a") {
		t.Fatalf("test_a deveria receber os artefatos de build_a: %v\n%s", ta["state"], log)
	}
	bb := waitState(t, ada, "/api/v4/jobs/"+id(jobs["build_b"]), "success")
	if toText(ta["started_at"]) >= toText(bb["finished_at"]) {
		t.Fatalf("test_a deveria começar antes de build_b terminar: %v × %v", ta["started_at"], bb["finished_at"])
	}
	// deploy é manual e, por vir de rules, não pode falhar: a pipeline espera
	done := waitState(t, ada, "/api/v4/projects/"+pid+"/pipelines/"+pipe, "manual", "success", "failed")
	if done["state"] != "manual" {
		t.Fatalf("pipeline deveria esperar o deploy manual: %v", done)
	}
	if in := waitState(t, ada, "/api/v4/jobs/"+id(jobs["instavel"]), "failed"); in["allow_failure"] != true {
		t.Fatalf("instavel: %v", in)
	}
	ada.must("POST", "/api/v4/jobs/"+id(jobs["deploy"])+"/play", nil, 200)
	if done := waitState(t, ada, "/api/v4/projects/"+pid+"/pipelines/"+pipe, "success", "failed"); done["state"] != "success" {
		t.Fatalf("depois do deploy (a falha permitida não reprova): %v", done)
	}
	ba := ada.must("GET", "/api/v4/jobs/"+id(jobs["build_a"]), nil, 200)
	if toText(ba["artifacts_expire_at"]) == "" {
		t.Fatalf("expire_in não virou prazo: %v", ba)
	}
	code, zipped := download(t, ada, "/api/v4/jobs/"+id(jobs["build_a"])+"/artifacts")
	if code != 200 {
		t.Fatalf("artefatos do build_a: %d", code)
	}
	if zr, err := zip.NewReader(bytes.NewReader(zipped), int64(len(zipped))); err != nil || len(zr.File) != 1 || zr.File[0].Name != "out/a.txt" {
		t.Fatalf("zip do build_a: %v", err)
	}
	// sem pessoa nem token de job em execução, ninguém baixa
	if code, _ := download(t, &api{t: t, base: base}, "/api/v4/jobs/"+id(jobs["build_a"])+"/artifacts"); code < 400 {
		t.Fatalf("artefatos sem credencial: %d", code)
	}

	// 2. numa branch feature-*: entram so_feature e fora_da_main; deploy não
	jobs = names(push("feature-x", ci))
	if jobs["so_feature"] == nil || jobs["fora_da_main"] == nil || jobs["deploy"] != nil {
		t.Fatalf("jobs da feature-x: %v", jobs)
	}

	// 3. artefatos expiram
	pipe = push("main", "guarda:\n  script: echo x > f.txt\n  artifacts:\n    paths: [f.txt]\n    expire_in: 3 sec\n")
	g := waitState(t, ada, "/api/v4/jobs/"+id(names(pipe)["guarda"]), "success", "failed")
	if code, _ := download(t, ada, "/api/v4/jobs/"+id(g)+"/artifacts"); code != 200 {
		t.Fatalf("artefatos antes de expirar: %d", code)
	}
	time.Sleep(4 * time.Second)
	if code, body := download(t, ada, "/api/v4/jobs/"+id(g)+"/artifacts"); code != 404 || !strings.Contains(string(body), "expired") {
		t.Fatalf("artefatos expirados: %d %s", code, body)
	}

	// 4. erros de configuração explicam o que escrever
	for src, want := range map[string]string{
		"a:\n  script: x\n  needs: [b]\nb:\n  script: y\n  needs: [a]\n":         "em círculo",
		"a:\n  script: x\n  rules:\n    - if: $CI_COMMIT_MESSAGE =~ /wip/\n":     "compare a branch",
		"a:\n  script: x\n  only: [feature]\nb:\n  script: y\n  needs: [a]\n":    "b precisa de a, que não vale para a branch main",
		"a:\n  script: x\n  artifacts:\n    paths: [f]\n    expire_in: 3 luas\n": `unidade "luas"`,
	} {
		bad := ada.must("GET", "/api/v4/projects/"+pid+"/pipelines/"+push("main", src), nil, 200)
		if bad["state"] != "failed" || !strings.Contains(toText(bad["erro_configuracao"]), want) {
			t.Fatalf("%q: esperado %q, veio %v", src, want, bad)
		}
	}
	// nenhum job para a branch: nenhuma pipeline
	before := len(ada.list("/api/v4/projects/" + pid + "/pipelines"))
	commit("outra", "a:\n  script: x\n  only: [main]\n")
	if after := len(ada.list("/api/v4/projects/" + pid + "/pipelines")); after != before {
		t.Fatalf("pipeline criada sem jobs para a branch: %d → %d", before, after)
	}
	ada.must("POST", "/api/v4/projects/"+pid+"/pipelines", map[string]any{"ref": "outra"}, 400)
}

func toText(v any) string {
	s, _ := v.(string)
	return s
}
