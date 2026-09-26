// Package germanio executes the strict Germanio AST. The legacy full-stack
// engine remains separate because its coercions are part of .ge compatibility.
package germanio

import (
	"bufio"
	"fmt"
	"html"
	"io"
	"math"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/diagnostics"
	"github.com/flaviokalleu/germanio/compiler/semantic"
)

type env struct {
	parent *env
	vars   map[string]any
}

func newEnv(p *env) *env { return &env{parent: p, vars: map[string]any{}} }
func (e *env) get(n string) any {
	for s := e; s != nil; s = s.parent {
		if v, ok := s.vars[n]; ok {
			return v
		}
	}
	return nil
}
func (e *env) set(n string, v any) {
	if n == "_" {
		return
	}
	for s := e; s != nil; s = s.parent {
		if _, ok := s.vars[n]; ok {
			s.vars[n] = v
			return
		}
	}
	e.vars[n] = v
}

type flow struct {
	kind  string
	value any
}
type Engine struct {
	Input        io.Reader
	Output       io.Writer
	MaxSteps     int
	steps, depth int
	input        *bufio.Reader
	module       *semantic.Module
	testName     string
	coverageAll  map[diagnostics.Position]bool
	coverageHit  map[diagnostics.Position]bool
}

func (e *Engine) Run(m *semantic.Module) (err error) {
	e.coverageAll, e.coverageHit = nil, nil
	if err = semantic.Check(m); err != nil {
		return err
	}
	if e.Input == nil {
		e.Input = strings.NewReader("")
	}
	if e.Output == nil {
		e.Output = io.Discard
	}
	if e.MaxSteps <= 0 {
		e.MaxSteps = 1000000
	}
	e.input = bufio.NewReader(e.Input)
	e.steps = 0
	e.depth = 0
	e.module = m
	defer func() {
		if p := recover(); p != nil {
			if d, ok := p.(*diagnostics.Diagnostic); ok {
				err = d
			} else {
				panic(p)
			}
		}
	}()
	// Imported modules export functions only and never execute top-level code.
	e.block(m.Program.Scripts, newEnv(nil))
	return nil
}

// RunTests checks the complete module, then executes only named tests with
// fresh local scopes. Neither top-level scripts nor imported tests are run.
func (e *Engine) RunTests(m *semantic.Module) (passed []string, err error) {
	if err = semantic.Check(m); err != nil {
		return nil, err
	}
	if e.Input == nil {
		e.Input = strings.NewReader("")
	}
	if e.Output == nil {
		e.Output = io.Discard
	}
	if e.MaxSteps <= 0 {
		e.MaxSteps = 1000000
	}
	e.input = bufio.NewReader(e.Input)
	e.module = m
	e.coverageAll, e.coverageHit = map[diagnostics.Position]bool{}, map[diagnostics.Position]bool{}
	e.collectCoverage(m, map[*semantic.Module]bool{}, true)
	defer func() {
		if p := recover(); p != nil {
			if d, ok := p.(*diagnostics.Diagnostic); ok {
				err = d
			} else {
				panic(p)
			}
		}
	}()
	for _, test := range m.Program.Tests {
		e.steps, e.depth = 0, 0
		e.testName = test.Name
		e.block(test.Body, newEnv(nil))
		passed = append(passed, test.Name)
	}
	return passed, nil
}

func (e *Engine) collectCoverage(m *semantic.Module, visited map[*semantic.Module]bool, entry bool) {
	if visited[m] {
		return
	}
	visited[m] = true
	for _, f := range m.Program.Functions {
		e.collectStatements(f.Body)
	}
	if entry {
		for _, test := range m.Program.Tests {
			e.collectStatements(test.Body)
		}
	}
	for _, name := range m.ImportOrder {
		e.collectCoverage(m.Imports[name], visited, false)
	}
}

func (e *Engine) collectStatements(stmts []*ast.Statement) {
	for _, st := range stmts {
		e.coverageAll[st.Pos] = true
		switch st.Type {
		case "if":
			e.collectStatements(st.If.Body)
			e.collectStatements(st.If.Else)
		case "while":
			e.collectStatements(st.While.Body)
		case "for_each":
			e.collectStatements(st.ForEach.Body)
		}
	}
}

// CoverageLines returns the positions reached by ge testar and all eligible
// statement positions. Consumers can union sets across multiple test files.
func (e *Engine) CoverageLines() (hit, all map[diagnostics.Position]bool) {
	hit, all = map[diagnostics.Position]bool{}, map[diagnostics.Position]bool{}
	for p := range e.coverageHit {
		hit[p] = true
	}
	for p := range e.coverageAll {
		all[p] = true
	}
	return hit, all
}
func (e *Engine) fail(pos diagnostics.Position, code, msg, why, fix string) {
	panic(&diagnostics.Diagnostic{Code: code, Position: pos, Source: e.module.Program.Source, Message: msg, Reason: why, Fix: fix, Example: diagnostics.Explanations[code]})
}
func (e *Engine) tick(pos diagnostics.Position) {
	e.steps++
	if e.steps > e.MaxSteps {
		e.fail(pos, "GE9001", "Limite de execução excedido", "O programa ultrapassou o orçamento de instruções.", "Confira a condição de parada do loop ou da recursão.")
	}
}
func (e *Engine) write(pos diagnostics.Position, s string) {
	if _, err := io.WriteString(e.Output, s); err != nil {
		e.fail(pos, "GE3001", "Falha de saída", err.Error(), "Confira o destino de saída.")
	}
}
func (e *Engine) block(stmts []*ast.Statement, s *env) flow {
	for _, st := range stmts {
		e.tick(st.Pos)
		if e.coverageHit != nil {
			e.coverageHit[st.Pos] = true
		}
		switch st.Type {
		case "expect":
			if !e.eval(st.Expect, s).(bool) {
				e.fail(st.Pos, "GE2008", "Teste falhou: "+e.testName, "A condição após espera resultou falso.", "Confira a função testada e a comparação esperada.")
			}
		case "bind":
			s.set(st.VarDecl.Name, e.eval(&st.VarDecl.Value, s))
		case "assign":
			s.set(st.Assign.Target, e.eval(&st.Assign.Value, s))
		case "print":
			e.write(st.Pos, text(e.eval(st.Print, s))+"\n")
		case "expr":
			e.eval(st.Expr, s)
		case "return":
			return flow{kind: "return", value: e.eval(st.Return, s)}
		case "break", "continue":
			return flow{kind: st.Type}
		case "if":
			body := st.If.Else
			if e.eval(&st.If.Condition, s).(bool) {
				body = st.If.Body
			}
			if f := e.block(body, newEnv(s)); f.kind != "" {
				return f
			}
		case "while":
			for e.eval(&st.While.Condition, s).(bool) {
				e.tick(st.Pos)
				f := e.block(st.While.Body, newEnv(s))
				if f.kind == "return" {
					return f
				}
				if f.kind == "break" {
					break
				}
			}
		case "for_each":
			for _, v := range e.eval(&st.ForEach.Collection, s).([]any) {
				child := newEnv(s)
				child.vars[st.ForEach.VarName] = v
				f := e.block(st.ForEach.Body, child)
				if f.kind == "return" {
					return f
				}
				if f.kind == "break" {
					break
				}
			}
		case "ui":
			e.write(st.Pos, RenderUI(st.UI)+"\n")
		}
	}
	return flow{}
}
func text(v any) string {
	if v == nil {
		return "nulo"
	}
	if b, ok := v.(bool); ok {
		if b {
			return "verdadeiro"
		}
		return "falso"
	}
	if a, ok := v.([]any); ok {
		parts := make([]string, len(a))
		for i, v := range a {
			parts[i] = text(v)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	return fmt.Sprint(v)
}
func number(v any) float64 {
	if n, ok := v.(int64); ok {
		return float64(n)
	}
	return v.(float64)
}
func (e *Engine) finite(pos diagnostics.Position, n float64) float64 {
	if math.IsNaN(n) || math.IsInf(n, 0) {
		e.fail(pos, "GE2005", "Número fora dos limites", "A operação produziu um decimal não finito.", "Reduza os operandos e confira o divisor.")
	}
	return n
}
func (e *Engine) eval(x *ast.Expression, s *env) any {
	e.tick(x.Pos)
	switch x.Type {
	case "literal":
		return x.Value
	case "variable":
		return s.get(x.Name)
	case "interpolate":
		var b strings.Builder
		for _, part := range x.Elements {
			b.WriteString(text(e.eval(part, s)))
		}
		return b.String()
	case "list":
		out := make([]any, len(x.Elements))
		for i, v := range x.Elements {
			out[i] = e.eval(v, s)
		}
		return out
	case "index":
		a := e.eval(x.Left, s).([]any)
		i := e.eval(x.Index, s).(int64)
		if i < 0 || i >= int64(len(a)) {
			e.fail(x.Pos, "GE2006", "Índice fora dos limites", fmt.Sprintf("Índice %d em lista com %d itens.", i, len(a)), "Confira quantidade(lista) antes de acessar o índice.")
		}
		return a[i]
	case "unary":
		v := e.eval(x.Right, s)
		if x.Operator == "nao" {
			return !v.(bool)
		}
		if i, ok := v.(int64); ok {
			if i == math.MinInt64 {
				e.fail(x.Pos, "GE2005", "Overflow de inteiro", "O negativo não cabe em 64 bits.", "Use um valor dentro dos limites.")
			}
			return -i
		}
		return e.finite(x.Pos, -v.(float64))
	case "binary":
		a := e.eval(x.Left, s)
		if x.Operator == "e" && !a.(bool) {
			return false
		}
		if x.Operator == "ou" && a.(bool) {
			return true
		}
		b := e.eval(x.Right, s)
		switch x.Operator {
		case "e", "ou":
			return b
		case "==":
			return reflect.DeepEqual(a, b)
		case "!=":
			return !reflect.DeepEqual(a, b)
		}
		if str, ok := a.(string); ok {
			return str + b.(string)
		}
		ai, aint := a.(int64)
		bi, bint := b.(int64)
		if aint && bint {
			switch x.Operator {
			case "<":
				return ai < bi
			case ">":
				return ai > bi
			case "<=":
				return ai <= bi
			case ">=":
				return ai >= bi
			case "%":
				if bi == 0 {
					e.fail(x.Pos, "GE2005", "Divisão por zero", "O divisor resultou em zero.", "Verifique o divisor antes da operação.")
				}
				return ai % bi
			case "+", "-", "*":
				z := new(big.Int)
				left, right := big.NewInt(ai), big.NewInt(bi)
				switch x.Operator {
				case "+":
					z.Add(left, right)
				case "-":
					z.Sub(left, right)
				case "*":
					z.Mul(left, right)
				}
				if !z.IsInt64() {
					e.fail(x.Pos, "GE2005", "Overflow de inteiro", "O resultado não cabe em inteiro de 64 bits.", "Reduza os valores ou converta explicitamente para decimal.")
				}
				return z.Int64()
			}
		}
		af, bf := number(a), number(b)
		switch x.Operator {
		case "+":
			return e.finite(x.Pos, af+bf)
		case "-":
			return e.finite(x.Pos, af-bf)
		case "*":
			return e.finite(x.Pos, af*bf)
		case "/":
			if bf == 0 {
				e.fail(x.Pos, "GE2005", "Divisão por zero", "O divisor resultou em zero.", "Verifique o divisor antes da operação.")
			}
			return e.finite(x.Pos, af/bf)
		case "<":
			return af < bf
		case ">":
			return af > bf
		case "<=":
			return af <= bf
		case ">=":
			return af >= bf
		}
	case "call":
		args := make([]any, len(x.Args))
		for i, a := range x.Args {
			args[i] = e.eval(a, s)
		}
		switch x.Name {
		case "pergunte":
			e.write(x.Pos, args[0].(string)+" ")
			line, err := e.input.ReadString('\n')
			if err != nil && err != io.EOF {
				e.fail(x.Pos, "GE3001", "Não foi possível ler a entrada", err.Error(), "Confira a entrada padrão.")
			}
			return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		case "texto":
			return text(args[0])
		case "quantidade":
			switch v := args[0].(type) {
			case string:
				return int64(utf8.RuneCountInString(v))
			case []any:
				return int64(len(v))
			}
		case "numero", "decimal", "inteiro":
			if x.Name == "inteiro" {
				if v, ok := args[0].(int64); ok {
					return v
				}
				if v, ok := args[0].(string); ok {
					n, err := strconv.ParseInt(v, 10, 64)
					if err != nil {
						e.fail(x.Pos, "GE2004", "Conversão inválida", err.Error(), "Forneça texto representando um inteiro de 64 bits.")
					}
					return n
				}
			}
			var n float64
			if v, ok := args[0].(string); ok {
				var err error
				n, err = strconv.ParseFloat(v, 64)
				if err != nil {
					e.fail(x.Pos, "GE2004", "Conversão inválida", fmt.Sprintf("%q não representa número.", v), "Use texto numérico como \"5\".")
				}
			} else {
				n = number(args[0])
			}
			n = e.finite(x.Pos, n)
			if x.Name == "inteiro" {
				if n >= 9223372036854775808.0 || n < -9223372036854775808.0 || math.Trunc(n) != n {
					e.fail(x.Pos, "GE2005", "Conversão perde informação", "O valor é fracionário ou não cabe em inteiro.", "Use um valor inteiro dentro dos limites.")
				}
				return int64(n)
			}
			return n
		}
		mod := e.module
		name := x.Name
		if i := strings.Index(name, "."); i >= 0 {
			mod = mod.Imports[name[:i]]
			name = name[i+1:]
		}
		f := mod.Functions[name]
		e.depth++
		if e.depth > 256 {
			e.fail(x.Pos, "GE9001", "Recursão excessiva", "Mais de 256 chamadas aninhadas.", "Adicione uma condição de parada.")
		}
		child := newEnv(nil)
		for i, n := range f.Decl.Params {
			child.vars[n] = args[i]
		}
		previous := e.module
		e.module = mod
		result := e.block(f.Decl.Body, child)
		e.module = previous
		e.depth--
		return result.value
	}
	e.fail(x.Pos, "GE1001", "Expressão sem execução definida", x.Type, "Confira a SPEC.md.")
	return nil
}

// RenderUI emits escaped static HTML. Natural UI is not a server RPC shortcut.
func RenderUI(u *ast.UIElement) string {
	style := "background:" + ast.ResolveColor(u.Color) + ";color:#fff;border-radius:" + u.Radius + ";padding:12px 20px"
	if u.Kind == "navbar" {
		return `<nav aria-label="Navegação principal" style="` + html.EscapeString(style) + `">` + html.EscapeString(u.Text) + `</nav>`
	}
	return `<button type="button" style="` + html.EscapeString(style) + `">` + html.EscapeString(u.Text) + `</button>`
}
