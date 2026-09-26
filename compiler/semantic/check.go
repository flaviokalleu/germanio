package semantic

import (
	"fmt"
	"sort"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/diagnostics"
)

type checker struct {
	m       *Module
	imports map[string]bool
	fn      *Function
	loops   int
}

func (c *checker) err(pos diagnostics.Position, code, msg, why, fix string) error {
	return diag(c.m.Program, pos, code, msg, why, fix)
}
func (c *checker) mismatch(pos diagnostics.Position, want, got *Type) error {
	if resolve(got).Kind == "nulo" {
		return c.err(pos, "GE2004", "nulo exige tipo opcional", fmt.Sprintf("O destino tem tipo %s; somente T? aceita nulo.", want), "Declare explicitamente um tipo opcional, por exemplo valor: texto? = nulo ou f(x: texto?).")
	}
	return c.err(pos, "GE2004", "Tipos incompatíveis", fmt.Sprintf("Esperado %s, recebido %s. Não há coerção implícita.", want, got), "Use valores do mesmo tipo ou uma conversão explícita.")
}

var reserved = map[string]bool{"mostre": true, "pergunte": true, "mut": true, "const": true, "se": true, "senao": true, "enquanto": true, "para": true, "em": true, "e": true, "ou": true, "nao": true, "usa": true, "retorne": true, "pare": true, "continue": true, "crie": true, "verdadeiro": true, "falso": true, "nulo": true, "numero": true, "inteiro": true, "decimal": true, "texto": true, "quantidade": true}

func Check(m *Module) error {
	if m.checked {
		return nil
	}
	for _, name := range m.ImportOrder {
		if err := Check(m.Imports[name]); err != nil {
			return err
		}
	}
	m.Functions = map[string]*Function{}
	m.Globals = scope(nil)
	c := &checker{m: m, imports: map[string]bool{}}
	for _, d := range m.Program.Functions {
		if reserved[d.Name] || d.Name == "_" {
			return c.err(d.Pos, "GE2001", "Nome reservado", d.Name+" pertence à linguagem.", "Escolha outro nome.")
		}
		if _, ok := m.Functions[d.Name]; ok {
			return c.err(d.Pos, "GE2001", "Função duplicada", d.Name+" já existe.", "Renomeie ou remova a duplicata.")
		}
		if _, ok := m.Imports[d.Name]; ok {
			return c.err(d.Pos, "GE2001", "Nome de módulo em uso", d.Name+" já identifica um import.", "Renomeie a função.")
		}
		f := &Function{Decl: d, Result: parseType(d.ResultType), Module: m}
		for _, t := range d.ParamTypes {
			f.Params = append(f.Params, parseType(t))
		}
		m.Functions[d.Name] = f
	}
	// Function signatures exist before checking bodies, supporting recursion.
	for _, d := range m.Program.Functions {
		f := m.Functions[d.Name]
		c.fn = f
		env := scope(nil)
		for i, n := range d.Params {
			if n == "_" {
				continue
			}
			if reserved[n] {
				return c.err(d.Pos, "GE2001", "Parâmetro reservado", n+" pertence à linguagem.", "Escolha outro nome.")
			}
			if _, ok := env.Bindings[n]; ok {
				return c.err(d.Pos, "GE2001", "Parâmetro duplicado", n+" aparece duas vezes.", "Use nomes distintos.")
			}
			env.Bindings[n] = &Binding{Type: f.Params[i], Pos: d.Pos}
		}
		returns, err := c.block(d.Body, env)
		if err != nil {
			return err
		}
		if !returns {
			if !unify(f.Result, typ("vazio")) {
				return c.err(d.Pos, "GE2004", "Retorno incompleto", "Nem todos os caminhos devolvem o valor declarado.", "Use retorne em todos os caminhos ou uma expressão final.")
			}
		}
		if err := c.unused(env); err != nil {
			return err
		}
	}
	c.fn = nil
	if _, err := c.block(m.Program.Scripts, m.Globals); err != nil {
		return err
	}
	if err := c.unused(m.Globals); err != nil {
		return err
	}
	for _, im := range m.Program.Imports {
		alias := strings.TrimSuffix(strings.ReplaceAll(im.Path, "\\", "/"), ".ge")
		if i := strings.LastIndex(alias, "/"); i >= 0 {
			alias = alias[i+1:]
		}
		if !c.imports[alias] {
			return c.err(im.Pos, "GE2003", "Import não utilizado", alias+" foi importado mas não é usado.", "Use uma função exportada ou remova o import.")
		}
	}
	m.checked = true
	return nil
}
func (c *checker) unused(s *Scope) error {
	names := make([]string, 0, len(s.Bindings))
	for n := range s.Bindings {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		b := s.Bindings[n]
		if !b.Used {
			return c.err(b.Pos, "GE2003", "Variável não utilizada", n+" não foi lida.", "Use o valor, remova a declaração ou escreva _ = "+n+" para descarte explícito.")
		}
	}
	return nil
}

// presenceGuard recognizes only direct, deterministic presence comparisons.
// Arbitrary boolean expressions cannot safely establish a variable's type.
func presenceGuard(x *ast.Expression) (string, bool) {
	if x.Type == "unary" && x.Operator == "nao" {
		name, present := presenceGuard(x.Right)
		return name, !present
	}
	if x.Type != "binary" || x.Operator != "==" && x.Operator != "!=" {
		return "", false
	}
	if x.Left.Type == "variable" && x.Right.Type == "literal" && x.Right.Value == nil {
		return x.Left.Name, x.Operator == "!="
	}
	if x.Right.Type == "variable" && x.Left.Type == "literal" && x.Left.Value == nil {
		return x.Right.Name, x.Operator == "!="
	}
	return "", false
}
func refine(s *Scope, name string) {
	if name == "" {
		return
	}
	if b, ok := s.lookup(name); ok && !b.Mutable && resolve(b.Type).Kind == "optional" {
		s.Bindings[name] = &Binding{Type: resolve(b.Type).Elem, Used: true, Pos: b.Pos, Origin: b}
	}
}
func (c *checker) child(stmts []*ast.Statement, parent *Scope, refinement string) (bool, error) {
	s := scope(parent)
	refine(s, refinement)
	r, e := c.block(stmts, s)
	if e == nil {
		e = c.unused(s)
	}
	return r, e
}
func (c *checker) block(stmts []*ast.Statement, s *Scope) (bool, error) {
	terminated := false
	returns := false
	for _, st := range stmts {
		if terminated {
			return false, c.err(st.Pos, "GE2007", "Código inalcançável", "A instrução anterior encerra este bloco.", "Remova ou mova esta instrução.")
		}
		switch st.Type {
		case "bind":
			d := st.VarDecl
			t, err := c.expr(&d.Value, s)
			if err != nil {
				return false, err
			}
			if d.Name == "_" {
				continue
			}
			if reserved[d.Name] {
				return false, c.err(st.Pos, "GE2001", "Nome reservado", d.Name+" pertence à linguagem.", "Escolha outro nome.")
			}
			if _, ok := c.m.Functions[d.Name]; ok {
				return false, c.err(st.Pos, "GE2001", "Nome de função em uso", d.Name+" já é uma função.", "Escolha outro nome.")
			}
			if _, ok := c.m.Imports[d.Name]; ok {
				return false, c.err(st.Pos, "GE2001", "Nome de módulo em uso", d.Name+" já é um módulo.", "Escolha outro nome.")
			}
			if b, ok := s.lookup(d.Name); ok {
				if d.Mutable || d.Constant || d.Annotation != "" {
					return false, c.err(st.Pos, "GE2001", "Redeclaração ou shadowing", d.Name+" já foi declarado.", "Use outro nome ou atribua a uma variável mutável existente.")
				}
				if !b.Mutable {
					return false, c.err(st.Pos, "GE2002", "Valor imutável", d.Name+" foi declarado sem mut.", "Declare mut "+d.Name+" = valor para permitir alterações.")
				}
				if !assignable(b.Type, t) {
					return false, c.mismatch(st.Pos, b.Type, t)
				}
				continue
			}
			want := parseType(d.Annotation)
			if d.Annotation == "" && (resolve(t).Kind == "nulo" || unresolvedOptional(t)) {
				return false, c.err(st.Pos, "GE2004", "Opcional sem tipo", "nulo não informa qual tipo poderá ser armazenado.", "Declare nome: texto? = nulo.")
			}
			if !assignable(want, t) {
				return false, c.mismatch(st.Pos, want, t)
			}
			s.Bindings[d.Name] = &Binding{Type: want, Mutable: d.Mutable, Pos: st.Pos}
		case "assign":
			b, ok := s.lookup(st.Assign.Target)
			if !ok {
				return false, c.unknown(st.Pos, st.Assign.Target)
			}
			if !b.Mutable {
				return false, c.err(st.Pos, "GE2002", "Valor imutável", st.Assign.Target+" não foi declarado com mut.", "Use mut na declaração inicial.")
			}
			t, e := c.expr(&st.Assign.Value, s)
			if e != nil {
				return false, e
			}
			if !assignable(b.Type, t) {
				return false, c.mismatch(st.Pos, b.Type, t)
			}
		case "print", "expr", "return":
			x := st.Expr
			if st.Type == "print" {
				x = st.Print
			}
			if st.Type == "return" {
				x = st.Return
			}
			t, e := c.expr(x, s)
			if e != nil {
				return false, e
			}
			if st.Type == "return" {
				if c.fn == nil {
					return false, c.err(st.Pos, "GE1001", "retorne fora de função", "Não há chamada para receber esse resultado.", "Use mostre ou coloque a instrução em uma função.")
				}
				if !assignable(c.fn.Result, t) {
					return false, c.mismatch(st.Pos, c.fn.Result, t)
				}
				terminated = true
				returns = true
			}
			if st.Type == "expr" && x.Type != "call" {
				return false, c.err(st.Pos, "GE1001", "Expressão sem destino", "O resultado não é mostrado nem armazenado.", "Use mostre expressão ou nome = expressão.")
			}
		case "if":
			t, e := c.expr(&st.If.Condition, s)
			if e != nil {
				return false, e
			}
			if !unify(t, typ("bool")) {
				return false, c.mismatch(st.Pos, typ("bool"), t)
			}
			name, present := presenceGuard(&st.If.Condition)
			thenName, elseName := "", ""
			if present {
				thenName = name
			} else {
				elseName = name
			}
			a, e := c.child(st.If.Body, s, thenName)
			if e != nil {
				return false, e
			}
			b, e := c.child(st.If.Else, s, elseName)
			if e != nil {
				return false, e
			}
			terminated = a && b
			returns = terminated
			if a && !b {
				refine(s, elseName)
			} else if b && !a {
				refine(s, thenName)
			}
		case "while":
			t, e := c.expr(&st.While.Condition, s)
			if e != nil {
				return false, e
			}
			if !unify(t, typ("bool")) {
				return false, c.mismatch(st.Pos, typ("bool"), t)
			}
			c.loops++
			_, e = c.child(st.While.Body, s, "")
			c.loops--
			if e != nil {
				return false, e
			}
		case "for_each":
			t, e := c.expr(&st.ForEach.Collection, s)
			if e != nil {
				return false, e
			}
			item := fresh()
			if !unify(t, &Type{Kind: "list", Elem: item}) {
				return false, c.mismatch(st.Pos, &Type{Kind: "list", Elem: item}, t)
			}
			env := scope(s)
			n := st.ForEach.VarName
			if reserved[n] {
				return false, c.unknown(st.Pos, n)
			}
			if _, ok := s.lookup(n); ok {
				return false, c.err(st.Pos, "GE2001", "Shadowing no loop", n+" já existe no escopo externo.", "Escolha outro nome para o item.")
			}
			if n != "_" {
				env.Bindings[n] = &Binding{Type: item, Pos: st.Pos}
			}
			c.loops++
			_, e = c.block(st.ForEach.Body, env)
			c.loops--
			if e != nil {
				return false, e
			}
			if e = c.unused(env); e != nil {
				return false, e
			}
		case "break", "continue":
			if c.loops == 0 {
				return false, c.err(st.Pos, "GE1001", "Controle de loop fora de loop", "Não há repetição para interromper ou continuar.", "Use dentro de para ou enquanto.")
			}
			terminated = true
		case "ui": // This AST contains client-only static data, never server expressions.
		default:
			return false, c.err(st.Pos, "GE1001", "Instrução não suportada", st.Type, "Use uma instrução da SPEC.md.")
		}
	}
	return returns, nil
}
func (c *checker) unknown(p diagnostics.Position, name string) error {
	return c.err(p, "GE2001", "Nome desconhecido: "+name, "O nome não está declarado neste escopo.", "Confira a grafia e declare o nome antes de usá-lo. Para mostrar texto, use mostre.")
}
func (c *checker) expr(x *ast.Expression, s *Scope) (*Type, error) {
	switch x.Type {
	case "literal":
		switch x.Value.(type) {
		case nil:
			return typ("nulo"), nil
		case bool:
			return typ("bool"), nil
		case string:
			return typ("texto"), nil
		case int64:
			return typ("inteiro"), nil
		case float64:
			return typ("decimal"), nil
		}
	case "variable":
		if b, ok := s.lookup(x.Name); ok {
			b.Used = true
			for origin := b.Origin; origin != nil; origin = origin.Origin {
				origin.Used = true
			}
			return b.Type, nil
		}
		return nil, c.unknown(x.Pos, x.Name)
	case "interpolate":
		for _, v := range x.Elements {
			if _, e := c.expr(v, s); e != nil {
				return nil, e
			}
		}
		return typ("texto"), nil
	case "list":
		item := fresh()
		optional := false
		for _, v := range x.Elements {
			t, e := c.expr(v, s)
			if e != nil {
				return nil, e
			}
			t = resolve(t)
			if t.Kind == "nulo" {
				optional = true
				continue
			}
			if t.Kind == "optional" {
				optional = true
				t = t.Elem
			}
			if !unify(item, t) {
				return nil, c.mismatch(v.Pos, item, t)
			}
		}
		if optional {
			item = &Type{Kind: "optional", Elem: item}
		}
		return &Type{Kind: "list", Elem: item}, nil
	case "index":
		t, e := c.expr(x.Left, s)
		if e != nil {
			return nil, e
		}
		item := fresh()
		if !unify(t, &Type{Kind: "list", Elem: item}) {
			return nil, c.mismatch(x.Pos, &Type{Kind: "list", Elem: item}, t)
		}
		idx, e := c.expr(x.Index, s)
		if e != nil {
			return nil, e
		}
		if !unify(idx, typ("inteiro")) {
			return nil, c.mismatch(x.Index.Pos, typ("inteiro"), idx)
		}
		return item, nil
	case "unary":
		t, e := c.expr(x.Right, s)
		if e != nil {
			return nil, e
		}
		if x.Operator == "nao" {
			if !unify(t, typ("bool")) {
				return nil, c.mismatch(x.Pos, typ("bool"), t)
			}
		} else if !constrain(t, "numeric") {
			return nil, c.mismatch(x.Pos, typ("decimal"), t)
		}
		return t, nil
	case "binary":
		a, e := c.expr(x.Left, s)
		if e != nil {
			return nil, e
		}
		b, e := c.expr(x.Right, s)
		if e != nil {
			return nil, e
		}
		op := x.Operator
		if op == "e" || op == "ou" {
			if !unify(a, typ("bool")) {
				return nil, c.mismatch(x.Pos, typ("bool"), a)
			}
			if !unify(b, typ("bool")) {
				return nil, c.mismatch(x.Pos, typ("bool"), b)
			}
			return typ("bool"), nil
		}
		if op == "==" || op == "!=" {
			ar, br := resolve(a), resolve(b)
			if ar.Kind == "nulo" && br.Kind == "optional" || br.Kind == "nulo" && ar.Kind == "optional" {
				return typ("bool"), nil
			}
			if ar.Kind == "nulo" || br.Kind == "nulo" {
				return nil, c.err(x.Pos, "GE2004", "Comparação sem tipo opcional", "nulo só pode ser comparado com um valor de tipo T?.", "Declare nome: texto? = nulo e compare nome == nulo.")
			}
			if !unify(a, b) {
				return nil, c.mismatch(x.Pos, a, b)
			}
			return typ("bool"), nil
		}
		class := "numeric"
		if op == "+" {
			class = "addable"
		}
		if !constrain(a, class) || !constrain(b, class) {
			return nil, c.mismatch(x.Pos, typ("inteiro ou decimal"), b)
		}
		result := a
		if numeric(a) && numeric(b) {
			if resolve(a).Kind == "decimal" || resolve(b).Kind == "decimal" {
				result = typ("decimal")
			}
		} else if !unify(a, b) {
			return nil, c.mismatch(x.Pos, a, b)
		}
		if op == "%" {
			if !unify(a, typ("inteiro")) || !unify(b, typ("inteiro")) {
				return nil, c.mismatch(x.Pos, typ("inteiro"), result)
			}
		}
		if op == "/" || op == "%" {
			if x.Right.Type == "literal" && (x.Right.Value == int64(0) || x.Right.Value == float64(0)) {
				return nil, c.err(x.Right.Pos, "GE2005", "Divisão por zero", "O divisor é zero.", "Use um divisor diferente de zero.")
			}
		}
		if op == "/" {
			result = typ("decimal")
		}
		if op == "<" || op == ">" || op == "<=" || op == ">=" {
			result = typ("bool")
		}
		return result, nil
	case "call":
		args := make([]*Type, len(x.Args))
		for i, a := range x.Args {
			t, e := c.expr(a, s)
			if e != nil {
				return nil, e
			}
			args[i] = t
		}
		if x.Name == "pergunte" || x.Name == "texto" || x.Name == "numero" || x.Name == "inteiro" || x.Name == "decimal" || x.Name == "quantidade" {
			if len(args) != 1 {
				return nil, c.err(x.Pos, "GE2004", "Quantidade de argumentos incorreta", x.Name+" espera um argumento.", "Passe exatamente um valor.")
			}
			a := resolve(args[0])
			switch x.Name {
			case "pergunte":
				if !unify(a, typ("texto")) {
					return nil, c.mismatch(x.Pos, typ("texto"), a)
				}
				return typ("texto"), nil
			case "texto":
				return typ("texto"), nil
			case "quantidade":
				if a.Kind != "texto" && a.Kind != "list" {
					return nil, c.mismatch(x.Pos, typ("texto ou lista"), a)
				}
				return typ("inteiro"), nil
			default:
				if a.Kind != "texto" && !numeric(a) {
					return nil, c.mismatch(x.Pos, typ("texto ou número"), a)
				}
				if x.Name == "inteiro" {
					return typ("inteiro"), nil
				}
				return typ("decimal"), nil
			}
		}
		mod := c.m
		name := x.Name
		if i := strings.Index(name, "."); i >= 0 {
			alias := name[:i]
			var ok bool
			mod, ok = c.m.Imports[alias]
			if !ok {
				return nil, c.unknown(x.Pos, alias)
			}
			c.imports[alias] = true
			name = name[i+1:]
		}
		f, ok := mod.Functions[name]
		if !ok {
			return nil, c.unknown(x.Pos, x.Name)
		}
		if len(args) != len(f.Params) {
			return nil, c.err(x.Pos, "GE2004", "Quantidade de argumentos incorreta", fmt.Sprintf("%s espera %d; recebeu %d.", x.Name, len(f.Params), len(args)), "Confira a assinatura da função.")
		}
		for i, a := range args {
			if !assignable(f.Params[i], a) {
				return nil, c.mismatch(x.Args[i].Pos, f.Params[i], a)
			}
		}
		return f.Result, nil
	}
	return nil, c.err(x.Pos, "GE1001", "Expressão não suportada", x.Type, "Confira a sintaxe na SPEC.md.")
}
