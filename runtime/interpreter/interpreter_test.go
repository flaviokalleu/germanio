package interpreter

import (
	"testing"

	"github.com/flaviokalleu/germanio/compiler/ast"
)

func newInterp() *Interpreter {
	return New(nil)
}

func TestWhenExecutesMatchingCaseAndConcurrencyExecutesTasks(t *testing.T) {
	interp := newInterp()
	interp.Global.Set("status", "aprovado")
	interp.Global.Set("resultado", "")
	interp.Global.Set("primeiro", "")
	interp.Global.Set("segundo", "")
	when := &ast.Statement{Type: "when", When: &ast.WhenStmt{
		Target: ast.Expression{Type: "variable", Name: "status"},
		Cases: []*ast.WhenCase{
			{Pattern: &ast.Expression{Type: "literal", Value: "recusado"}, Body: []*ast.Statement{{Type: "assign", Assign: &ast.Assignment{Target: "resultado", Value: ast.Expression{Type: "literal", Value: "nao"}}}}},
			{Pattern: &ast.Expression{Type: "literal", Value: "aprovado"}, Body: []*ast.Statement{{Type: "assign", Assign: &ast.Assignment{Target: "resultado", Value: ast.Expression{Type: "literal", Value: "sim"}}}}},
		},
	}}
	interp.ExecStatement(when, interp.Global)
	if result, _ := interp.Global.Get("resultado"); result != "sim" {
		t.Fatalf("when não executou o caso correspondente: %v", result)
	}

	concurrency := &ast.Statement{Type: "concurrency", Concurrency: &ast.ConcurrencyStmt{Tasks: []*ast.Statement{
		{Type: "assign", Assign: &ast.Assignment{Target: "primeiro", Value: ast.Expression{Type: "literal", Value: "ok"}}},
		{Type: "assign", Assign: &ast.Assignment{Target: "segundo", Value: ast.Expression{Type: "literal", Value: "ok"}}},
	}}}
	interp.ExecStatement(concurrency, interp.Global)
	if value, _ := interp.Global.Get("primeiro"); value != "ok" {
		t.Fatal("primeira tarefa de concorrência não executou")
	}
	if value, _ := interp.Global.Get("segundo"); value != "ok" {
		t.Fatal("segunda tarefa de concorrência não executou")
	}
}

func TestVariableDeclaration(t *testing.T) {
	interp := newInterp()
	stmt := &ast.Statement{
		Type: "var",
		VarDecl: &ast.VarDecl{
			Name:  "x",
			Value: ast.Expression{Type: "literal", Value: "10"},
		},
	}
	interp.ExecStatement(stmt, interp.Global)
	val, ok := interp.Global.Get("x")
	if !ok {
		t.Fatal("variable x not found")
	}
	if toNumber(val) != 10 {
		t.Errorf("expected 10, got %v", val)
	}
}

func TestArithmetic(t *testing.T) {
	interp := newInterp()

	tests := []struct {
		op       string
		left     interface{}
		right    interface{}
		expected float64
	}{
		{"+", "5", "3", 8},
		{"-", "10", "4", 6},
		{"*", "6", "7", 42},
		{"/", "20", "5", 4},
		{"/", "10", "0", 0}, // division by zero returns 0
	}

	for _, tt := range tests {
		t.Run(tt.op, func(t *testing.T) {
			expr := &ast.Expression{
				Type:     "binary",
				Operator: tt.op,
				Left:     &ast.Expression{Type: "literal", Value: tt.left},
				Right:    &ast.Expression{Type: "literal", Value: tt.right},
			}
			result := interp.EvalExpr(expr, interp.Global)
			if toNumber(result) != tt.expected {
				t.Errorf("%v %s %v = %v, want %v", tt.left, tt.op, tt.right, result, tt.expected)
			}
		})
	}
}

func TestStringConcatenation(t *testing.T) {
	interp := newInterp()
	expr := &ast.Expression{
		Type:     "binary",
		Operator: "+",
		Left:     &ast.Expression{Type: "literal", Value: "hello "},
		Right:    &ast.Expression{Type: "literal", Value: "world"},
	}
	result := interp.EvalExpr(expr, interp.Global)
	if result != "hello world" {
		t.Errorf("expected 'hello world', got %v", result)
	}
}

func TestBooleanLogic(t *testing.T) {
	interp := newInterp()

	tests := []struct {
		name     string
		op       string
		left     interface{}
		right    interface{}
		expected bool
	}{
		{"and true", "e", true, true, true},
		{"and false", "e", true, false, false},
		{"or true", "ou", false, true, true},
		{"or false", "ou", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := &ast.Expression{
				Type:     "binary",
				Operator: tt.op,
				Left:     &ast.Expression{Type: "literal", Value: tt.left},
				Right:    &ast.Expression{Type: "literal", Value: tt.right},
			}
			result := interp.EvalExpr(expr, interp.Global)
			if toBool(result) != tt.expected {
				t.Errorf("%v %s %v = %v, want %v", tt.left, tt.op, tt.right, result, tt.expected)
			}
		})
	}
}

func TestUnaryNot(t *testing.T) {
	interp := newInterp()
	expr := &ast.Expression{
		Type:     "unary",
		Operator: "nao",
		Right:    &ast.Expression{Type: "literal", Value: true},
	}
	result := interp.EvalExpr(expr, interp.Global)
	if toBool(result) != false {
		t.Errorf("nao verdadeiro = %v, want false", result)
	}
}

func TestComparisons(t *testing.T) {
	interp := newInterp()

	tests := []struct {
		name     string
		op       string
		left     interface{}
		right    interface{}
		expected bool
	}{
		{"equal", "==", "5", "5", true},
		{"not equal", "!=", "5", "3", true},
		{"greater", ">", "10", "5", true},
		{"less", "<", "3", "7", true},
		{"greater equal", ">=", "5", "5", true},
		{"less equal", "<=", "3", "5", true},
		{"equal false", "==", "5", "3", false},
		{"greater false", ">", "3", "5", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := &ast.Expression{
				Type:     "binary",
				Operator: tt.op,
				Left:     &ast.Expression{Type: "literal", Value: tt.left},
				Right:    &ast.Expression{Type: "literal", Value: tt.right},
			}
			result := interp.EvalExpr(expr, interp.Global)
			if toBool(result) != tt.expected {
				t.Errorf("%v %s %v = %v, want %v", tt.left, tt.op, tt.right, result, tt.expected)
			}
		})
	}
}

func TestIfElse(t *testing.T) {
	interp := newInterp()
	// Pre-set x so that the child scope update propagates to global
	interp.Global.Set("x", float64(0))

	// if true, set x = 1; else set x = 2
	ifStmt := &ast.Statement{
		Type: "if",
		If: &ast.IfStmt{
			Condition: ast.Expression{Type: "literal", Value: true},
			Body: []*ast.Statement{
				{
					Type:   "assign",
					Assign: &ast.Assignment{Target: "x", Value: ast.Expression{Type: "literal", Value: "1"}},
				},
			},
			Else: []*ast.Statement{
				{
					Type:   "assign",
					Assign: &ast.Assignment{Target: "x", Value: ast.Expression{Type: "literal", Value: "2"}},
				},
			},
		},
	}
	interp.ExecStatement(ifStmt, interp.Global)
	val, ok := interp.Global.Get("x")
	if !ok {
		t.Fatal("variable x not found after if")
	}
	if toNumber(val) != 1 {
		t.Errorf("expected x=1 (true branch), got %v", val)
	}

	// Now test else branch
	interp2 := newInterp()
	interp2.Global.Set("x", float64(0))
	ifStmt.If.Condition = ast.Expression{Type: "literal", Value: false}
	interp2.ExecStatement(ifStmt, interp2.Global)
	val2, ok := interp2.Global.Get("x")
	if !ok {
		t.Fatal("variable x not found after else")
	}
	if toNumber(val2) != 2 {
		t.Errorf("expected x=2 (else branch), got %v", val2)
	}
}

func TestFunctionCallAndReturn(t *testing.T) {
	interp := newInterp()

	// Register function: funcao dobro(n) retornar n * 2
	fn := &ast.FuncDecl{
		Name:   "dobro",
		Params: []string{"n"},
		Body: []*ast.Statement{
			{
				Type: "return",
				Return: &ast.Expression{
					Type:     "binary",
					Operator: "*",
					Left:     &ast.Expression{Type: "variable", Name: "n"},
					Right:    &ast.Expression{Type: "literal", Value: "2"},
				},
			},
		},
	}
	interp.RegisterFunction(fn)

	// Call dobro(5)
	expr := &ast.Expression{
		Type: "call",
		Name: "dobro",
		Args: []*ast.Expression{
			{Type: "literal", Value: "5"},
		},
	}
	result := interp.EvalExpr(expr, interp.Global)
	if toNumber(result) != 10 {
		t.Errorf("dobro(5) = %v, want 10", result)
	}
}

func TestForEachLoop(t *testing.T) {
	interp := newInterp()

	// Set up: definir soma = 0
	interp.Global.Set("soma", float64(0))

	// for_each item em [1, 2, 3]: soma = soma + item
	forEach := &ast.Statement{
		Type: "for_each",
		ForEach: &ast.ForEachStmt{
			VarName: "item",
			Collection: ast.Expression{
				Type: "list",
				Elements: []*ast.Expression{
					{Type: "literal", Value: "1"},
					{Type: "literal", Value: "2"},
					{Type: "literal", Value: "3"},
				},
			},
			Body: []*ast.Statement{
				{
					Type: "assign",
					Assign: &ast.Assignment{
						Target: "soma",
						Value: ast.Expression{
							Type:     "binary",
							Operator: "+",
							Left:     &ast.Expression{Type: "variable", Name: "soma"},
							Right:    &ast.Expression{Type: "variable", Name: "item"},
						},
					},
				},
			},
		},
	}
	interp.ExecStatement(forEach, interp.Global)

	val, _ := interp.Global.Get("soma")
	if toNumber(val) != 6 {
		t.Errorf("sum of [1,2,3] = %v, want 6", val)
	}
}

func TestBuiltinTamanho(t *testing.T) {
	interp := newInterp()

	// tamanho("hello") == 5
	expr := &ast.Expression{
		Type: "call",
		Name: "tamanho",
		Args: []*ast.Expression{
			{Type: "literal", Value: "hello"},
		},
	}
	result := interp.EvalExpr(expr, interp.Global)
	if toNumber(result) != 5 {
		t.Errorf("tamanho('hello') = %v, want 5", result)
	}

	// length of array
	interp.Global.Set("arr", []interface{}{"a", "b", "c"})
	expr2 := &ast.Expression{
		Type: "call",
		Name: "tamanho",
		Args: []*ast.Expression{
			{Type: "variable", Name: "arr"},
		},
	}
	result2 := interp.EvalExpr(expr2, interp.Global)
	if toNumber(result2) != 3 {
		t.Errorf("tamanho([a,b,c]) = %v, want 3", result2)
	}
}

func TestBuiltinStringFunctions(t *testing.T) {
	interp := newInterp()

	tests := []struct {
		name     string
		funcName string
		args     []interface{}
		expected interface{}
	}{
		{"maiusculo", "maiusculo", []interface{}{"hello"}, "HELLO"},
		{"minusculo", "minusculo", []interface{}{"HELLO"}, "hello"},
		{"contem true", "contem", []interface{}{"hello world", "world"}, true},
		{"contem false", "contem", []interface{}{"hello", "xyz"}, false},
		{"substituir", "substituir", []interface{}{"hello world", "world", "germanio"}, "hello germanio"},
		{"cortar", "cortar", []interface{}{"  hello  "}, "hello"},
		{"comeca_com", "comeca_com", []interface{}{"hello", "hel"}, true},
		{"termina_com", "termina_com", []interface{}{"hello", "llo"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := interp.callBuiltin(tt.funcName, tt.args)
			if !ok {
				t.Fatalf("builtin %q not found", tt.funcName)
			}
			switch expected := tt.expected.(type) {
			case string:
				if toString(result) != expected {
					t.Errorf("%s(%v) = %v, want %v", tt.funcName, tt.args, result, expected)
				}
			case bool:
				if toBool(result) != expected {
					t.Errorf("%s(%v) = %v, want %v", tt.funcName, tt.args, result, expected)
				}
			}
		})
	}
}

func TestBuiltinArrayOperations(t *testing.T) {
	interp := newInterp()

	// adicionar (push)
	arr := []interface{}{float64(1), float64(2)}
	result, ok := interp.callBuiltin("adicionar", []interface{}{arr, float64(3)})
	if !ok {
		t.Fatal("adicionar not found")
	}
	resultArr, ok := result.([]interface{})
	if !ok {
		t.Fatal("adicionar did not return array")
	}
	if len(resultArr) != 3 {
		t.Errorf("expected length 3, got %d", len(resultArr))
	}
	if toNumber(resultArr[2]) != 3 {
		t.Errorf("expected last element 3, got %v", resultArr[2])
	}

	// reverter (reverse)
	result2, ok := interp.callBuiltin("reverter", []interface{}{[]interface{}{"a", "b", "c"}})
	if !ok {
		t.Fatal("reverter not found")
	}
	rev := result2.([]interface{})
	if rev[0] != "c" || rev[1] != "b" || rev[2] != "a" {
		t.Errorf("reverse [a,b,c] = %v, want [c,b,a]", rev)
	}
}

func TestBuiltinDividirJuntar(t *testing.T) {
	interp := newInterp()

	// dividir (split)
	result, ok := interp.callBuiltin("dividir", []interface{}{"a,b,c", ","})
	if !ok {
		t.Fatal("dividir not found")
	}
	arr := result.([]interface{})
	if len(arr) != 3 || arr[0] != "a" || arr[1] != "b" || arr[2] != "c" {
		t.Errorf("dividir('a,b,c', ',') = %v, want [a b c]", arr)
	}

	// juntar (join)
	result2, ok := interp.callBuiltin("juntar", []interface{}{[]interface{}{"x", "y", "z"}, "-"})
	if !ok {
		t.Fatal("juntar not found")
	}
	if result2 != "x-y-z" {
		t.Errorf("juntar([x,y,z], '-') = %v, want 'x-y-z'", result2)
	}
}

func TestBuiltinMathFunctions(t *testing.T) {
	interp := newInterp()

	tests := []struct {
		name     string
		funcName string
		args     []interface{}
		expected float64
	}{
		{"abs positive", "abs", []interface{}{float64(-5)}, 5},
		{"abs negative", "abs", []interface{}{float64(5)}, 5},
		{"min", "min", []interface{}{float64(3), float64(7)}, 3},
		{"max", "max", []interface{}{float64(3), float64(7)}, 7},
		{"arredondar", "arredondar", []interface{}{float64(3.7)}, 4},
		{"inteiro", "inteiro", []interface{}{float64(3.9)}, 3},
		{"potencia", "potencia", []interface{}{float64(2), float64(3)}, 8},
		{"raiz", "raiz", []interface{}{float64(9)}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := interp.callBuiltin(tt.funcName, tt.args)
			if !ok {
				t.Fatalf("builtin %q not found", tt.funcName)
			}
			if toNumber(result) != tt.expected {
				t.Errorf("%s(%v) = %v, want %v", tt.funcName, tt.args, result, tt.expected)
			}
		})
	}
}

func TestBuiltinTipo(t *testing.T) {
	interp := newInterp()

	tests := []struct {
		input    interface{}
		expected string
	}{
		{float64(42), "numero"},
		{"hello", "texto"},
		{true, "booleano"},
		{nil, "nulo"},
		{[]interface{}{1, 2}, "lista"},
		{map[string]interface{}{"a": 1}, "objeto"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result, ok := interp.callBuiltin("tipo", []interface{}{tt.input})
			if !ok {
				t.Fatal("tipo not found")
			}
			if result != tt.expected {
				t.Errorf("tipo(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuiltinJson(t *testing.T) {
	interp := newInterp()

	// Parse JSON string
	result, ok := interp.callBuiltin("json", []interface{}{`{"name":"test","value":42}`})
	if !ok {
		t.Fatal("json not found")
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("json did not return a map")
	}
	if m["name"] != "test" {
		t.Errorf("json parse: name = %v, want 'test'", m["name"])
	}

	// Serialize to JSON
	result2, ok := interp.callBuiltin("json", []interface{}{map[string]interface{}{"a": float64(1)}})
	if !ok {
		t.Fatal("json not found")
	}
	s, ok := result2.(string)
	if !ok {
		t.Fatal("json serialize did not return string")
	}
	if s != `{"a":1}` {
		t.Errorf("json serialize = %q, want %q", s, `{"a":1}`)
	}
}

func TestScopeChaining(t *testing.T) {
	parent := NewScope(nil)
	parent.Set("x", float64(10))

	child := NewScope(parent)
	child.Set("y", float64(20))

	// Child can see parent vars
	val, ok := child.Get("x")
	if !ok || toNumber(val) != 10 {
		t.Errorf("child should see parent's x=10, got %v", val)
	}

	// Parent cannot see child vars
	_, ok = parent.Get("y")
	if ok {
		t.Error("parent should not see child's y")
	}

	// Setting x in child updates parent
	child.Set("x", float64(99))
	val, _ = parent.Get("x")
	if toNumber(val) != 99 {
		t.Errorf("setting x in child should update parent; got %v", val)
	}
}

func TestScopeSetLocal(t *testing.T) {
	parent := NewScope(nil)
	parent.Set("x", float64(10))

	child := NewScope(parent)
	child.SetLocal("x", float64(99))

	// Child's local x should shadow parent's
	val, _ := child.Get("x")
	if toNumber(val) != 99 {
		t.Errorf("child local x should be 99, got %v", val)
	}

	// Parent's x should be unchanged
	val, _ = parent.Get("x")
	if toNumber(val) != 10 {
		t.Errorf("parent x should still be 10, got %v", val)
	}
}

func TestListLiteral(t *testing.T) {
	interp := newInterp()
	expr := &ast.Expression{
		Type: "list",
		Elements: []*ast.Expression{
			{Type: "literal", Value: "1"},
			{Type: "literal", Value: "2"},
			{Type: "literal", Value: "3"},
		},
	}
	result := interp.EvalExpr(expr, interp.Global)
	arr, ok := result.([]interface{})
	if !ok {
		t.Fatal("list literal did not return []interface{}")
	}
	if len(arr) != 3 {
		t.Errorf("expected 3 elements, got %d", len(arr))
	}
	if toNumber(arr[0]) != 1 || toNumber(arr[1]) != 2 || toNumber(arr[2]) != 3 {
		t.Errorf("expected [1,2,3], got %v", arr)
	}
}

func TestPrintStatement(t *testing.T) {
	interp := newInterp()
	stmt := &ast.Statement{
		Type:  "print",
		Print: &ast.Expression{Type: "literal", Value: "hello from germanio"},
	}
	interp.ExecStatement(stmt, interp.Global)

	logs := interp.GetLogs(false)
	if len(logs) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(logs))
	}
	if logs[0] != "hello from germanio" {
		t.Errorf("expected log 'hello from germanio', got %q", logs[0])
	}
}

func TestWhileLoop(t *testing.T) {
	interp := newInterp()
	interp.Global.Set("i", float64(0))
	interp.Global.Set("soma", float64(0))

	// while i < 5: soma = soma + i; i = i + 1
	whileStmt := &ast.Statement{
		Type: "while",
		While: &ast.WhileStmt{
			Condition: ast.Expression{
				Type: "binary", Operator: "<",
				Left:  &ast.Expression{Type: "variable", Name: "i"},
				Right: &ast.Expression{Type: "literal", Value: "5"},
			},
			Body: []*ast.Statement{
				{
					Type: "assign",
					Assign: &ast.Assignment{
						Target: "soma",
						Value: ast.Expression{
							Type: "binary", Operator: "+",
							Left:  &ast.Expression{Type: "variable", Name: "soma"},
							Right: &ast.Expression{Type: "variable", Name: "i"},
						},
					},
				},
				{
					Type: "assign",
					Assign: &ast.Assignment{
						Target: "i",
						Value: ast.Expression{
							Type: "binary", Operator: "+",
							Left:  &ast.Expression{Type: "variable", Name: "i"},
							Right: &ast.Expression{Type: "literal", Value: "1"},
						},
					},
				},
			},
		},
	}
	interp.ExecStatement(whileStmt, interp.Global)

	val, _ := interp.Global.Get("soma")
	// 0+1+2+3+4 = 10
	if toNumber(val) != 10 {
		t.Errorf("while loop sum 0..4 = %v, want 10", val)
	}
}

func TestRepeatLoop(t *testing.T) {
	interp := newInterp()
	interp.Global.Set("count", float64(0))

	repeatStmt := &ast.Statement{
		Type: "repeat",
		Repeat: &ast.RepeatStmt{
			Count: ast.Expression{Type: "literal", Value: "5"},
			Body: []*ast.Statement{
				{
					Type: "assign",
					Assign: &ast.Assignment{
						Target: "count",
						Value: ast.Expression{
							Type: "binary", Operator: "+",
							Left:  &ast.Expression{Type: "variable", Name: "count"},
							Right: &ast.Expression{Type: "literal", Value: "1"},
						},
					},
				},
			},
		},
	}
	interp.ExecStatement(repeatStmt, interp.Global)

	val, _ := interp.Global.Get("count")
	if toNumber(val) != 5 {
		t.Errorf("repeat 5 times: count = %v, want 5", val)
	}
}

func TestTypeConversionHelpers(t *testing.T) {
	// toNumber
	if toNumber(float64(42)) != 42 {
		t.Error("toNumber(float64)")
	}
	if toNumber("3.14") != 3.14 {
		t.Error("toNumber(string)")
	}
	if toNumber(true) != 1 {
		t.Error("toNumber(true)")
	}
	if toNumber(false) != 0 {
		t.Error("toNumber(false)")
	}
	if toNumber(nil) != 0 {
		t.Error("toNumber(nil)")
	}

	// toString
	if toString(nil) != "nulo" {
		t.Error("toString(nil)")
	}
	if toString(float64(42)) != "42" {
		t.Errorf("toString(42) = %q", toString(float64(42)))
	}
	if toString(true) != "verdadeiro" {
		t.Error("toString(true)")
	}
	if toString(false) != "falso" {
		t.Error("toString(false)")
	}

	// toBool
	if toBool(nil) != false {
		t.Error("toBool(nil)")
	}
	if toBool(true) != true {
		t.Error("toBool(true)")
	}
	if toBool(float64(0)) != false {
		t.Error("toBool(0)")
	}
	if toBool(float64(1)) != true {
		t.Error("toBool(1)")
	}
	if toBool("") != false {
		t.Error("toBool('')")
	}
	if toBool("hello") != true {
		t.Error("toBool('hello')")
	}
	if toBool("falso") != false {
		t.Error("toBool('falso')")
	}
}
