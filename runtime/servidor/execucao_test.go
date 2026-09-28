package servidor

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// O formato nativo de arquivo de execução, sem nenhum adaptador.
func TestArquivoDeExecucaoNativo(t *testing.T) {
	src := `
estagios: [construir, testar]
etapas:
  compilar:
    estagio: construir
    comandos:
      - make
      - make install
    imagem: golang:1.23
  unidade:
    estagio: testar
    comandos: make test
    depois: [make clean]
    pode_falhar: true
  publicar:
    estagio: testar
    comandos: make publish
    quando: manual
`
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(src), &doc); err != nil {
		t.Fatal(err)
	}
	specs, err := parseNativeRun(geValue(doc).(map[string]any))
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 3 || specs[0].Name != "compilar" || specs[0].Order != 1 || specs[0].Image != "golang:1.23" || len(specs[0].Script) != 2 {
		t.Fatalf("etapas: %+v", specs)
	}
	for _, s := range specs[1:] {
		if s.Order != 2 {
			t.Fatalf("estágio de %s: %d", s.Name, s.Order)
		}
		switch s.Name {
		case "unidade":
			if !s.AllowFailure || s.When != whenAuto || len(s.After) != 1 {
				t.Fatalf("unidade: %+v", s)
			}
		case "publicar":
			if s.When != whenManual || s.AllowFailure {
				t.Fatalf("publicar: %+v", s)
			}
		}
	}
	for src, want := range map[string]string{
		"etapas:\n  a:\n    estagio: x\n    comandos: ls\n":    `usa o estágio "x"`,
		"etapas:\n  a:\n    comandos: []\n":                    "não tem comandos",
		"etapas:\n  a:\n    comandos: ls\n    quando: nunca\n": "use automatico, manual ou sempre",
		"estagios: [a]\n": "nenhuma etapa",
	} {
		var d map[string]any
		yaml.Unmarshal([]byte(src), &d)
		if _, err := parseNativeRun(geValue(d).(map[string]any)); err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("%q: esperado %q, veio %v", src, want, err)
		}
	}
}
