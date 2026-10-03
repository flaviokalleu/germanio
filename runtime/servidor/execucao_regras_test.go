package servidor

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func nativeSpecs(t *testing.T, src string) ([]stepSpec, error) {
	t.Helper()
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatal(err)
	}
	return parseNativeRun(geValue(doc).(map[string]any))
}

// job builds a created step row the way createRun stores it.
func job(id int, name string, order int, when string, allow bool, plan stepPlan) map[string]any {
	b, _ := json.Marshal(plan)
	return map[string]any{"id": float64(id), "nome": name, "ordem": float64(order), "quando": when, "permitir_falha": allow, "estado": stCreated, "script": string(b)}
}

func states(jobs []map[string]any) map[string]string {
	out := map[string]string{}
	for _, j := range jobs {
		out[toStr(j["nome"])] = toStr(j["estado"])
	}
	return out
}

func finish(jobs []map[string]any, name, st string) {
	for _, j := range jobs {
		if j["nome"] == name {
			j["estado"] = st
		}
	}
}

// precisa: uma etapa começa quando as que ela precisa terminam, sem esperar
// o estágio anterior inteiro.
func TestEtapasPrecisamDeOutras(t *testing.T) {
	jobs := []map[string]any{
		job(1, "compilar_a", 1, whenAuto, false, stepPlan{}),
		job(2, "compilar_b", 1, whenAuto, false, stepPlan{}),
		job(3, "testar_a", 2, whenAuto, false, stepPlan{Needs: []string{"compilar_a"}, NeedsSet: true}),
		job(4, "testar_b", 2, whenAuto, false, stepPlan{}), // stage order: waits for the whole stage
		job(5, "lint", 2, whenAuto, false, stepPlan{NeedsSet: true}),
	}
	progress(jobs)
	if s := states(jobs); s["compilar_a"] != stPending || s["compilar_b"] != stPending || s["testar_a"] != stCreated || s["testar_b"] != stCreated || s["lint"] != stPending {
		t.Fatalf("início: %v", s)
	}
	finish(jobs, "compilar_a", stSuccess)
	progress(jobs)
	if s := states(jobs); s["testar_a"] != stPending || s["testar_b"] != stCreated {
		t.Fatalf("testar_a deveria começar sem esperar compilar_b: %v", s)
	}
	if st, done := summarize(jobs); st != stRunning || done {
		t.Fatalf("execução: %s %v", st, done)
	}
	finish(jobs, "compilar_b", stFailed)
	finish(jobs, "testar_a", stSuccess)
	finish(jobs, "lint", stSuccess)
	progress(jobs)
	if s := states(jobs); s["testar_b"] != stSkipped {
		t.Fatalf("testar_b deveria ser ignorada depois da falha: %v", s)
	}
	if st, done := summarize(jobs); st != stFailed || !done {
		t.Fatalf("execução: %s %v", st, done)
	}
}

// pode_falhar não para a execução; sempre roda mesmo depois de falhas; o que
// depende de uma etapa ignorada também é ignorado.
func TestPodeFalharESempre(t *testing.T) {
	jobs := []map[string]any{
		job(1, "instavel", 1, whenAuto, true, stepPlan{}),
		job(2, "quebra", 1, whenAuto, false, stepPlan{}),
		job(3, "depois", 2, whenAuto, false, stepPlan{}),
		job(4, "limpar", 3, whenAlways, false, stepPlan{}),
		job(5, "publicar", 3, whenAuto, false, stepPlan{Needs: []string{"depois"}, NeedsSet: true}),
	}
	progress(jobs)
	finish(jobs, "instavel", stFailed)
	finish(jobs, "quebra", stSuccess)
	progress(jobs)
	if s := states(jobs); s["depois"] != stPending {
		t.Fatalf("falha permitida não deveria parar o estágio seguinte: %v", s)
	}
	finish(jobs, "depois", stFailed)
	progress(jobs)
	if s := states(jobs); s["limpar"] != stPending || s["publicar"] != stSkipped {
		t.Fatalf("sempre deveria rodar e publicar ser ignorada: %v", s)
	}
	finish(jobs, "limpar", stSuccess)
	if st, done := summarize(jobs); st != stFailed || !done {
		t.Fatalf("execução: %s %v", st, done)
	}

	// só falhas permitidas: a execução passa
	ok := []map[string]any{job(1, "a", 1, whenAuto, true, stepPlan{}), job(2, "b", 2, whenAuto, false, stepPlan{})}
	progress(ok)
	finish(ok, "a", stFailed)
	progress(ok)
	finish(ok, "b", stSuccess)
	if st, done := summarize(ok); st != stSuccess || !done {
		t.Fatalf("falha permitida: %s %v", st, done)
	}
}

// Etapa manual: a que pode falhar não segura os estágios seguintes; a que não
// pode falhar segura o que vem depois e deixa a execução em manual; quem
// precisa dela pelo nome espera que alguém a execute.
func TestEtapasManuais(t *testing.T) {
	jobs := []map[string]any{
		job(1, "aprovar", 1, whenManual, true, stepPlan{}),
		job(2, "seguir", 2, whenAuto, false, stepPlan{}),
		job(3, "entregar", 2, whenAuto, false, stepPlan{Needs: []string{"aprovar"}, NeedsSet: true}),
	}
	progress(jobs)
	if s := states(jobs); s["aprovar"] != stManual || s["seguir"] != stPending || s["entregar"] != stCreated {
		t.Fatalf("manual que pode falhar: %v", s)
	}
	finish(jobs, "seguir", stSuccess)
	if st, done := summarize(jobs); st != stManual || done {
		t.Fatalf("entregar espera a manual: %s %v", st, done)
	}
	finish(jobs, "aprovar", stSuccess)
	progress(jobs)
	if s := states(jobs); s["entregar"] != stPending {
		t.Fatalf("depois de executar a manual: %v", s)
	}

	blocking := []map[string]any{
		job(1, "liberar", 1, whenManual, false, stepPlan{}),
		job(2, "implantar", 2, whenAuto, false, stepPlan{}),
	}
	progress(blocking)
	if s := states(blocking); s["liberar"] != stManual || s["implantar"] != stCreated {
		t.Fatalf("manual que não pode falhar segura o estágio seguinte: %v", s)
	}
	if st, done := summarize(blocking); st != stManual || done {
		t.Fatalf("execução bloqueada: %s %v", st, done)
	}

	// manual no fim, que pode falhar: a execução terminou com sucesso
	last := []map[string]any{job(1, "a", 1, whenAuto, false, stepPlan{}), job(2, "release", 2, whenManual, true, stepPlan{})}
	progress(last)
	finish(last, "a", stSuccess)
	progress(last)
	if st, done := summarize(last); st != stSuccess || !done || states(last)["release"] != stManual {
		t.Fatalf("manual opcional no fim: %s %v %v", st, done, states(last))
	}
}

func TestArtefatosRecebidos(t *testing.T) {
	jobs := []map[string]any{
		job(1, "a", 1, whenAuto, false, stepPlan{}),
		job(2, "b", 1, whenAuto, false, stepPlan{}),
		job(3, "c", 2, whenAuto, false, stepPlan{}),
		job(4, "d", 2, whenAuto, false, stepPlan{Needs: []string{"a"}, NeedsSet: true}),
		job(5, "e", 2, whenAuto, false, stepPlan{From: []string{"b"}, FromSet: true}),
		job(6, "f", 2, whenAuto, false, stepPlan{FromSet: true}),
	}
	g := newStepGraph(jobs)
	names := func(list []map[string]any) string {
		var out []string
		for _, j := range list {
			out = append(out, toStr(j["nome"]))
		}
		return strings.Join(out, ",")
	}
	for name, want := range map[string]string{"c": "a,b", "d": "a", "e": "b", "f": ""} {
		if got := names(g.sources(g.byName[name])); got != want {
			t.Fatalf("%s recebe de %q, esperado %q", name, got, want)
		}
	}
}

// somente_em, exceto_em e regras escolhem as etapas de cada branch.
func TestEtapasPorBranch(t *testing.T) {
	specs, err := nativeSpecs(t, `
estagios: [testar, publicar]
etapas:
  testes:
    comandos: make test
  so_main:
    comandos: make x
    somente_em: [branch padrão]
  releases:
    comandos: make r
    somente_em: ["release/*"]
  fora_rascunho:
    comandos: make y
    exceto_em: [rascunho/*]
  publicar:
    estagio: publicar
    comandos: make publish
    precisa: [testes, {etapa: releases, opcional: sim}]
    regras:
      - em: [main]
        quando: manual
        pode_falhar: não
      - exceto_em: [main, release/*]
        quando: nunca
      - quando: automatico
`)
	if err != nil {
		t.Fatal(err)
	}
	pick := func(branch string) map[string]stepSpec {
		got, err := forBranch(specs, branch, "main")
		if err != nil {
			t.Fatalf("%s: %v", branch, err)
		}
		out := map[string]stepSpec{}
		for _, s := range got {
			out[s.Name] = s
		}
		return out
	}
	main := pick("main")
	if _, ok := main["so_main"]; !ok || len(main) != 4 || main["publicar"].When != whenManual || main["publicar"].AllowFailure {
		t.Fatalf("main: %+v", main)
	}
	if n := main["publicar"].Needs; len(n) != 1 || n[0].Name != "testes" {
		t.Fatalf("a necessidade opcional some quando a etapa não vale: %+v", n)
	}
	rel := pick("release/1.0")
	if _, ok := rel["releases"]; !ok || rel["publicar"].When != whenAuto || len(rel["publicar"].Needs) != 2 {
		t.Fatalf("release/1.0: %+v", rel)
	}
	draft := pick("rascunho/ideia")
	if len(draft) != 1 || draft["testes"].Name == "" {
		t.Fatalf("rascunho: %+v", draft)
	}

	if _, err := forBranch([]stepSpec{{Name: "x", OnlySet: true, Only: []string{"main"}}}, "dev", "main"); !errorsAs(err, new(errNoSteps)) {
		t.Fatalf("sem etapas para a branch: %v", err)
	}
	needy, _ := nativeSpecs(t, "etapas:\n  a:\n    comandos: x\n    somente_em: [main]\n  b:\n    comandos: y\n    precisa: [a]\n")
	if _, err := forBranch(needy, "dev", "main"); err == nil || !strings.Contains(err.Error(), "b precisa de a, que não vale para a branch dev") {
		t.Fatalf("necessidade obrigatória ausente: %v", err)
	}
}

func TestPadraoDeBranch(t *testing.T) {
	for _, c := range []struct {
		pattern, branch string
		want            bool
	}{
		{"main", "main", true}, {"main", "mainline", false}, {"release/*", "release/1.0", true},
		{"release/*", "release/a/b", true}, {"release/*", "release", false}, {"*", "qualquer/coisa", true},
		{"v1.*", "v1x", false}, {"v1.*", "v1.2", true}, {"branch padrão", "trunk", true}, {"branch padrão", "main", false},
	} {
		if got := branchMatches(c.pattern, c.branch, "trunk"); got != c.want {
			t.Fatalf("%q × %q: %v", c.pattern, c.branch, got)
		}
	}
}

func TestExpiracaoDosArtefatos(t *testing.T) {
	for in, want := range map[string]time.Duration{
		"7 dias": 7 * 24 * time.Hour, "1 hora 30 minutos": 90 * time.Minute, "2 semanas": 14 * 24 * time.Hour,
		"1 mês": 30 * 24 * time.Hour, "1 ano": 365 * 24 * time.Hour, "3 segundos": 3 * time.Second, "nunca": 0,
	} {
		got, err := parseKeep(in)
		if err != nil || got != want {
			t.Fatalf("%q: %v %v", in, got, err)
		}
	}
	for in, want := range map[any]string{"7": "número e a unidade", "7 luas": "unidade \"luas\"", "x dias": "não é um número", 3600.0: "número e a unidade", "0 dias": "maior que zero"} {
		if _, err := parseKeep(in); err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("%v: esperado %q, veio %v", in, want, err)
		}
	}
	specs, err := nativeSpecs(t, "etapas:\n  a:\n    comandos: x\n    artefatos: [out/]\n    artefatos_expiram_em: 1 hora\n")
	if err != nil || specs[0].Expire != time.Hour {
		t.Fatalf("artefatos_expiram_em: %+v %v", specs, err)
	}
	if got := artifactExpiry(job(1, "a", 1, whenAuto, false, stepPlan{Expire: 3600})); got == nil {
		t.Fatal("expiração não calculada")
	}
	if got := artifactExpiry(job(1, "a", 1, whenAuto, false, stepPlan{})); got != nil {
		t.Fatalf("sem expiração: %v", got)
	}
	past := map[string]any{expiryField: time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)}
	future := map[string]any{expiryField: time.Now().Add(time.Hour).UTC().Format(time.RFC3339)}
	if !artifactsExpired(past) || artifactsExpired(future) || artifactsExpired(map[string]any{}) {
		t.Fatal("artifactsExpired")
	}
}

// Erros do formato nativo ensinam a forma certa.
func TestErrosDoFluxo(t *testing.T) {
	for src, want := range map[string]string{
		"etapas:\n  a:\n    comandos: x\n    precisa: [b]\n":                                          `precisa de "b", que não é uma etapa`,
		"etapas:\n  a:\n    comandos: x\n    precisa: [a]\n":                                          "não pode precisar de si mesma",
		"etapas:\n  a:\n    comandos: x\n    precisa: [b]\n  b:\n    comandos: y\n    precisa: [a]\n": "em círculo (a → b → a)",
		"etapas:\n  a:\n    comandos: x\n    regras: [{quando: talvez}]\n":                            "use automatico, manual, sempre ou nunca",
		"etapas:\n  a:\n    comandos: x\n    regras: [{se: main}]\n":                                  `tem a chave "se"`,
		"etapas:\n  a:\n    comandos: x\n    somente_em: {main: 1}\n":                                 "não é uma lista de branches",
		"etapas:\n  a:\n    comandos: x\n    recebe_artefatos_de: [z]\n":                              `recebe artefatos de "z"`,
		"etapas:\n  a:\n    comandos: x\n    artefatos_expiram_em: 5 luas\n":                          "unidade \"luas\"",
		"etapas:\n  a:\n    comandos: x\n    pode_falhar: talvez\n":                                   "use sim ou não",
		"etapas:\n  a:\n    comandos: x\n    precisa: [{opcional: sim}]\n":                            "sem etapa",
	} {
		if _, err := nativeSpecs(t, src); err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("%q: esperado %q, veio %v", src, want, err)
		}
	}
	if specs, err := nativeSpecs(t, "etapas:\n  a:\n    comandos: x\n    pode_falhar: sim\n"); err != nil || !specs[0].AllowFailure {
		t.Fatalf("pode_falhar: sim: %+v %v", specs, err)
	}
}
