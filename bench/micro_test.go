// Package bench is the permanent benchmark suite of Germanio (see README.md).
// MICRO: the language pipeline. APLICAÇÃO: a running application compared
// with the same application written directly in Go (bench/baseline).
// STRESS lives in bench/stress (a separate program, never part of go test).
package bench

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/lexer"
	"github.com/flaviokalleu/germanio/compiler/parser"
	"github.com/flaviokalleu/germanio/runtime"
	"github.com/flaviokalleu/germanio/runtime/interpreter"
)

func read(b *testing.B, path string) string {
	b.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		b.Fatal(err)
	}
	return string(src)
}

// synthetic builds a program with n data blocks in the hierarchical form, so
// the cost can be followed as programs grow.
func synthetic(n int) string {
	var s strings.Builder
	s.WriteString("crie sistema Sintetico\n\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&s, "coisa%ss\n    tem\n        nome obrigatório\n        descrição\n        quantidade número\n    começa aberto\n    pode\n        fechar\n        reabrir\n    permita\n        pesquisar\n\n", letters(i))
	}
	return s.String()
}

// letters names data without digits and without letters that Portuguese
// plural rules would fold together (coisais and coisals both end in coisal).
func letters(i int) string {
	const alphabet = "bcdfgkpt"
	s := string(alphabet[i%8])
	for i /= 8; i > 0; i /= 8 {
		s = string(alphabet[i%8]) + s
	}
	return s
}

func parse(b *testing.B, src string) *ast.Program {
	toks, err := lexer.New(src).Tokenize()
	if err != nil {
		b.Fatal(err)
	}
	prog, err := parser.New(toks).Parse()
	if err != nil {
		b.Fatal(err)
	}
	return prog
}

func BenchmarkLexer(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		src := synthetic(n)
		b.Run(fmt.Sprintf("dados=%d", n), func(b *testing.B) {
			b.SetBytes(int64(len(src)))
			b.ReportAllocs()
			for b.Loop() {
				if _, err := lexer.New(src).Tokenize(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkParser(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		src := synthetic(n)
		toks, err := lexer.New(src).Tokenize()
		if err != nil {
			b.Fatal(err)
		}
		b.Run(fmt.Sprintf("dados=%d", n), func(b *testing.B) {
			b.SetBytes(int64(len(src)))
			b.ReportAllocs()
			for b.Loop() {
				// the parser may consume the slice: give it a fresh copy
				if _, err := parser.New(append([]lexer.Token(nil), toks...)).Parse(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkResolver: the intent → resolved application model (ast.App).
func BenchmarkResolver(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		src := synthetic(n)
		b.Run(fmt.Sprintf("dados=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				prog := parse(b, src)
				b.StartTimer()
				if err := parser.ResolveIntent(prog); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkCompilarProjeto: a whole real project with imports, from files to
// the resolved model, as `ge check` does it.
func BenchmarkCompilarProjeto(b *testing.B) {
	for _, p := range []struct{ name, file string }{
		{"clientes", "testdata/clientes.ge"},
		{"gitlab", "../examples/gitlab-foss/app.ge"},
	} {
		b.Run(p.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := runtime.Compilar(p.file); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkInterpretador: the tree-walking interpreter over the logic level
// (recursion, arithmetic, lists).
func BenchmarkInterpretador(b *testing.B) {
	prog := parse(b, read(b, "testdata/logica.ge"))
	if len(prog.Scripts) == 0 || len(prog.Functions) == 0 {
		b.Fatalf("logica.ge sem funções ou instruções: %d %d", len(prog.Functions), len(prog.Scripts))
	}
	in := interpreter.New(nil)
	in.EvalStatements(prog.Scripts, prog.Functions)
	if v, _ := in.Global.Get("resultado"); fmt.Sprint(v) != "610" {
		// EvalStatements runs in a child scope; read it from the log-free path
		scope := interpreter.NewScope(in.Global)
		in.ExecStatements(prog.Scripts, scope)
		if v, _ := scope.Get("resultado"); fmt.Sprint(v) != "610" {
			b.Fatalf("fib(15) = %v, esperado 610", v)
		}
	}
	b.ReportAllocs()
	for b.Loop() {
		in.ExecStatements(prog.Scripts, interpreter.NewScope(in.Global))
	}
}
