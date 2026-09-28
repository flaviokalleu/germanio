package interpreter

import (
	"testing"
	"time"

	"github.com/flaviokalleu/germanio/compiler/lexer"
	"github.com/flaviokalleu/germanio/compiler/parser"
)

func programa(t *testing.T, src string) *Interpreter {
	t.Helper()
	toks, err := lexer.New(src).Tokenize()
	if err != nil {
		t.Fatal(err)
	}
	prog, err := parser.New(toks).Parse()
	if err != nil {
		t.Fatal(err)
	}
	in := New(nil)
	in.Run(prog)
	return in
}

// paralelo runs its tasks at the same time; tasks that write the same
// variable must not corrupt the interpreter (go test -race), and the
// results keep the order of the tasks.
func TestParaleloSemCorrida(t *testing.T) {
	in := programa(t, `logica
    funcao conta()
        contador = contador + 1
        retornar contador

    funcao dobro()
        retornar 2

definir contador = 0
definir tarefas = []
para cada i em [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20]
    tarefas = adicionar(tarefas, "conta")
tarefas = adicionar(tarefas, "dobro")
definir resultados = paralelo(tarefas)
`)
	res, _ := in.Global.Get("resultados")
	list, ok := res.([]interface{})
	if !ok || len(list) != 21 {
		t.Fatalf("resultados: %#v", res)
	}
	if toNumber(list[20]) != 2 {
		t.Fatalf("a ordem dos resultados não segue a das tarefas: %v", list[20])
	}
}

// timeout stops the task when the deadline passes: after timeout returns,
// the task no longer runs (it used to keep going in the background).
func TestTimeoutInterrompeATarefa(t *testing.T) {
	in := programa(t, `logica
    funcao lenta()
        enquanto passos < 100000
            esperar(1)
            passos = passos + 1
        retornar "fim"

definir passos = 0
definir resultado = timeout("lenta", 50)
definir depois = passos
`)
	res, _ := in.Global.Get("resultado")
	if res != nil {
		t.Fatalf("o timeout devolveu %v", res)
	}
	depois, _ := in.Global.Get("depois")
	time.Sleep(100 * time.Millisecond)
	agora, _ := in.Global.Get("passos")
	if toNumber(agora) != toNumber(depois) {
		t.Fatalf("a tarefa continuou depois do timeout: %v → %v", depois, agora)
	}
	if toNumber(depois) < 1 {
		t.Fatalf("a tarefa nem começou: %v", depois)
	}
}
