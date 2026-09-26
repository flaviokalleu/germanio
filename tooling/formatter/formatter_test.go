package formatter

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/flaviokalleu/germanio/compiler/semantic"
	"github.com/flaviokalleu/germanio/runtime/germanio"
)

func TestFormatterIdempotentAndPreservesBehavior(t *testing.T) {
	for _, source := range []string{
		"# comentário\nx= 2 // valor\nmostre x+1\n",
		"f(a:inteiro,b:inteiro)->inteiro\n  a+b\nmostre f(1,2)",
		"xs:[inteiro]=[1,2]\nmostre xs[0]\n",
		`mostre "https://site/#anchor"`,
		"nome=\"Flavio\"\nmostre \"Olá {nome}\"\n",
		"crie botão azul\n  borda arredondada 16px\n",
		"mut i=0\n# teste\nenquanto i<2\n  i+=1\n  mostre i\n",
		"variavel i=0\n# teste\nenquanto i<2\n  i+=1\n  mostre i\n",
		"id<T>(x:T)->T\n  x\nmostre id(1)\nmostre id(\"A\")\n",
	} {
		t.Run(source, func(t *testing.T) {
			a, err := Format("x.ge", source)
			if err != nil {
				t.Fatal(err)
			}
			b, err := Format("x.ge", a)
			if err != nil || a != b {
				t.Fatalf("not idempotent: %q %q %v", a, b, err)
			}
			outputs := []string{}
			for _, s := range []string{source, a} {
				m, e := semantic.CheckSource("x.ge", s)
				if e != nil {
					t.Fatal(e)
				}
				var buf bytes.Buffer
				if e = (&germanio.Engine{Output: &buf}).Run(m); e != nil {
					t.Fatal(e)
				}
				outputs = append(outputs, buf.String())
			}
			if !reflect.DeepEqual(outputs[:1], outputs[1:]) {
				t.Fatal(outputs)
			}
			if strings.Contains(source, "comentário") && !strings.Contains(a, "comentário") {
				t.Fatal("comment lost")
			}
		})
	}
}
func TestFormatterStyle(t *testing.T) {
	got, err := Format("x.ge", "x=1\nmostre x+2\n")
	if err != nil || got != "x = 1\nmostre x + 2\n" {
		t.Fatalf("%q %v", got, err)
	}
	got, err = Format("x.ge", "privado id<T>(x:T)->T\n  x\n")
	if err != nil || got != "privado id<T>(x: T) -> T\n  x\n" {
		t.Fatalf("generic style %q %v", got, err)
	}
	got, err = Format("x.ge", "mostre -1\nmostre 2-1\n")
	if err != nil || got != "mostre -1\nmostre 2 - 1\n" {
		t.Fatalf("unary minus style %q %v", got, err)
	}
}
func TestFormatterRejectsInvalid(t *testing.T) {
	if _, err := Format("x.ge", "mostre [1"); err == nil {
		t.Fatal("invalid source accepted")
	}
}
