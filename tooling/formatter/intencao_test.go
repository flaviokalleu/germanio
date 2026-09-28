package formatter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ge fmt nos arquivos de aplicação: idempotente e sem mudar o significado,
// para todos os .ge de intenção do repositório.
func TestFormatoAplicacaoIdempotenteNoRepositorio(t *testing.T) {
	var files []string
	for _, dir := range []string{"../../examples/gitlab-foss", "../../runtime/testdata/intencao", "../../cli/modelos"} {
		filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && filepath.Ext(p) == ".ge" {
				files = append(files, p)
			}
			return nil
		})
	}
	formatted := 0
	for _, f := range files {
		src, _ := os.ReadFile(f)
		once, err := formatApplication(f, string(src))
		if err == errNotApplication {
			continue // adaptadores só com lógica/rotas
		}
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		twice, err := formatApplication(f, once)
		if err != nil || twice != once {
			t.Fatalf("%s: não é idempotente (%v)", f, err)
		}
		formatted++
	}
	if formatted < 20 {
		t.Fatalf("poucos arquivos formatados: %d", formatted)
	}
}

func TestFormatoAplicacaoCanonico(t *testing.T) {
	in := "crie sistema x\n\n\n\ntenha papeis\n\tguest   10\n\n# as issues\nissues\n  tem\n      titulo   obrigatório  # comentário do campo\n      cor padrão \"#6699cc\"\n  # quem pode\n  acesso\n      guest\n          ver\n\n\n\n"
	want := "crie sistema x\n\ntenha papeis\n    guest 10\n\n# as issues\nissues\n    tem\n        titulo obrigatório  # comentário do campo\n        cor padrão \"#6699cc\"\n    # quem pode\n    acesso\n        guest\n            ver\n"
	got, err := formatApplication("x.ge", in)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("forma canônica:\n%s\n--- esperado ---\n%s", got, want)
	}
}

func TestFormatoAplicacaoPreservaAspasTriplas(t *testing.T) {
	in := "crie sistema x\n\ntenha notas\n\ncada nota tem\n    texto\n\nao iniciar\n  definir t = \"\"\"\n     linha   com   espaços\n  \"\"\"\n  mostrar t\n"
	got, err := formatApplication("x.ge", in)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "\n     linha   com   espaços\n") {
		t.Fatalf("conteúdo das aspas triplas mudou:\n%s", got)
	}
}
