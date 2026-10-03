package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// O fluxo das execuções sem GitLab nem CI: obras montam passos a partir do
// arquivo nativo do próprio repositório. Um passo que precisa de outro começa
// sem esperar o estágio inteiro e recebe seus artefatos; somente_em e
// exceto_em escolhem os passos da branch; o manual que pode falhar não segura
// a montagem; artefatos vencidos são apagados.
func TestExecucaoNativaSemGitLab(t *testing.T) {
	t.Setenv("GERMANIO_EXECUTOR", "local")
	t.Setenv("GERMANIO_ARQUIVOS", filepath.Join(t.TempDir(), "arquivos"))
	app, c := loadApp(t, "testdata/obras/app.ge")
	c.expect("POST", "/cadastro", map[string]any{"nome": "Ana", "email": "ana@x.com", "senha": "senha-da-ana"}, 201)
	c.expect("POST", "/entrar", map[string]any{"email": "ana@x.com", "senha": "senha-da-ana"}, 200)
	c.csrf = csrfFromCookie(t, c)
	obra := c.expect("POST", "/_ge/api/obras", map[string]any{"nome": "Ponte", "caminho": "ponte", "iniciar_repositorio": true}, 201)
	oid := fmt.Sprint(obra["id"])
	m := c.expect("POST", "/_ge/api/obras/"+oid+"/montagens", map[string]any{"branch": "main"}, 201)
	base := "/_ge/api/obras/" + oid + "/montagens/" + fmt.Sprint(m["id"])

	passos := func() map[string]map[string]any {
		_, _, raw := c.do("GET", base+"/passos", nil)
		var list []map[string]any
		if err := json.Unmarshal([]byte(raw), &list); err != nil {
			t.Fatalf("passos: %v %s", err, raw)
		}
		out := map[string]map[string]any{}
		for _, p := range list {
			out[fmt.Sprint(p["nome"])] = p
		}
		return out
	}
	wait := func(name string, want ...string) map[string]any {
		deadline := time.Now().Add(30 * time.Second)
		for {
			p := passos()[name]
			for _, w := range want {
				if p != nil && p["estado"] == w {
					return p
				}
			}
			if time.Now().After(deadline) {
				t.Fatalf("%s: %v, esperado %v", name, p, want)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	all := passos()
	if all["fora"] != nil || all["so_principal"] == nil || len(all) != 5 {
		t.Fatalf("passos da main: %v", all)
	}
	usa := wait("usa", "sucesso", "falhou", "ignorado")
	lento := wait("lento", "sucesso")
	if usa["estado"] != "sucesso" || fmt.Sprint(usa["iniciado_em"]) >= fmt.Sprint(lento["terminado_em"]) {
		t.Fatalf("usa deveria começar antes de lento terminar e receber os artefatos: %v × %v", usa, lento)
	}
	if p := wait("aprovar", "manual"); p == nil {
		t.Fatal("aprovar")
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		got := c.expect("GET", base, nil, 200)
		if got["estado"] == "sucesso" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("montagem: %v (o manual que pode falhar não segura)", got)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// artefatos: guardados com prazo; vencidos, a limpeza apaga o arquivo
	b := passos()["base"]
	if exp, _ := b["artefatos_expiram_em"].(string); exp == "" || b["artefatos"] == nil {
		t.Fatalf("base sem artefatos ou sem prazo: %v", b)
	}
	files, _ := filepath.Glob(filepath.Join(os.Getenv("GERMANIO_ARQUIVOS"), "passo", "*"))
	if len(files) != 1 {
		t.Fatalf("arquivos guardados: %v", files)
	}
	if n := app.Server.ExpirarArtefatos(time.Now()); n != 0 {
		t.Fatalf("nada venceu ainda, mas %d foram apagados", n)
	}
	if _, err := app.Interpreter.Op(&interp.Context{}, "passo", "atualizar", b["id"], map[string]any{"artefatos_expiram_em": time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)}); err != nil {
		t.Fatal(err)
	}
	if code, _, raw := c.do("GET", base+"/passos/"+fmt.Sprint(b["id"])+"/artefatos", nil); code != 404 || !strings.Contains(raw, "expiraram") {
		t.Fatalf("artefatos vencidos ainda entregues: %d %s", code, raw)
	}
	if n := app.Server.ExpirarArtefatos(time.Now()); n != 1 {
		t.Fatalf("limpeza: %d", n)
	}
	if _, err := os.Stat(files[0]); !os.IsNotExist(err) {
		t.Fatalf("o arquivo vencido continua no disco: %v", err)
	}
	if b := passos()["base"]; b["artefatos"] != nil {
		t.Fatalf("o passo ainda aponta para o arquivo: %v", b)
	}
}
