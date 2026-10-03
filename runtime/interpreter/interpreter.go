package interpreter

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/diagnostics"
	"github.com/flaviokalleu/germanio/runtime/banco"
)

const maxIterations = 10000

// signalType is used for control flow via panic/recover.
type signalType int

const (
	signalReturn signalType = iota
	signalBreak
	signalContinue
)

// signal carries a control flow signal and an optional value.
type signal struct {
	Type  signalType
	Value interface{}
}

// Scope holds variable bindings with parent scope chaining.
type Scope struct {
	mu     *sync.RWMutex // only the global scope is shared between tasks; nil elsewhere
	vars   map[string]interface{}
	parent *Scope
	// ctx is the execution context inherited by nested scopes.
	ctx *Context
	// global marks the interpreter's top-level scope.
	global bool
}

func NewScope(parent *Scope) *Scope {
	s := &Scope{vars: make(map[string]interface{}), parent: parent}
	if parent != nil {
		s.ctx = parent.ctx
	}
	return s
}

func (s *Scope) lock() {
	if s.mu != nil {
		s.mu.Lock()
	}
}

func (s *Scope) unlock() {
	if s.mu != nil {
		s.mu.Unlock()
	}
}

func (s *Scope) rlock() {
	if s.mu != nil {
		s.mu.RLock()
	}
}

func (s *Scope) runlock() {
	if s.mu != nil {
		s.mu.RUnlock()
	}
}

func (s *Scope) Get(name string) (interface{}, bool) {
	s.rlock()
	v, ok := s.vars[name]
	s.runlock()
	if ok {
		return v, true
	}
	if s.parent != nil {
		return s.parent.Get(name)
	}
	return nil, false
}

// Set updates the nearest scope that has name, or creates it here. Code
// running inside a request or task never writes the global scope: a global
// with the same name is shadowed locally, so concurrent executions cannot
// race on shared state.
func (s *Scope) Set(name string, value interface{}) {
	for cur := s; cur != nil; cur = cur.parent {
		if cur.global && s.ctx != nil && !s.ctx.AllowGlobal {
			break
		}
		cur.lock()
		if _, ok := cur.vars[name]; ok {
			cur.vars[name] = value
			cur.unlock()
			return
		}
		cur.unlock()
	}
	s.SetLocal(name, value)
}

func (s *Scope) SetLocal(name string, value interface{}) {
	s.lock()
	s.vars[name] = value
	s.unlock()
}

// Interpreter executes Germanio AST scripts.
type Interpreter struct {
	// ReservedNames: top-level addresses kept by the app (see endereco.go).
	ReservedNames map[string]bool
	Global        *Scope
	Functions     map[string]*ast.FuncDecl
	DB            *banco.Banco
	LogBuffer     []string
	logMu         sync.Mutex
	HTTPClient    interface {
		Chamar(method, url string, body []byte) ([]byte, error)
	}
	WAClient interface {
		EnviarMensagem(telefone, mensagem string) error
	}
	// Modules are capability namespaces callable as modulo.funcao(...).
	Modules map[string]map[string]ModuleFunc
	// App is the resolved intent layer (entities, roles, rules).
	App *ast.App
	// OnChange, when set, is told of every record written through the
	// interpreter (before is nil on create, after is nil on delete), so the
	// server can announce it after the commit (GEP 0020).
	OnChange func(ctx *Context, model string, before, after map[string]any)
	// Online, when set, tells whether the person with this id has a page of
	// the application open (presence, GEP 0021).
	Online func(id any) bool
	// Decorate, when set, adds values computed for the viewer to a record
	// being shown (unread counts, GEP 0022).
	Decorate func(atual map[string]any, e *ast.Entity, row, out map[string]any)
}

// New creates a new interpreter.
func New(db *banco.Banco) *Interpreter {
	global := NewScope(nil)
	global.global = true
	global.mu = new(sync.RWMutex)
	interp := &Interpreter{
		Global:    global,
		Functions: make(map[string]*ast.FuncDecl),
		DB:        db,
	}
	registerStdlib(interp)
	return interp
}

// AppendLog adds a message to the log buffer.
func (interp *Interpreter) AppendLog(msg string) {
	interp.logMu.Lock()
	defer interp.logMu.Unlock()
	interp.LogBuffer = append(interp.LogBuffer, msg)
	// Keep only last 1000 log entries
	if len(interp.LogBuffer) > 1000 {
		interp.LogBuffer = interp.LogBuffer[len(interp.LogBuffer)-1000:]
	}
}

// GetLogs returns and optionally clears the log buffer.
func (interp *Interpreter) GetLogs(clear bool) []string {
	interp.logMu.Lock()
	defer interp.logMu.Unlock()
	logs := make([]string, len(interp.LogBuffer))
	copy(logs, interp.LogBuffer)
	if clear {
		interp.LogBuffer = nil
	}
	return logs
}

// RegisterFunction registers a function declaration.
func (interp *Interpreter) RegisterFunction(fn *ast.FuncDecl) {
	interp.Functions[fn.Name] = fn
}

// ExecStatements executes a list of statements in the given scope.
func (interp *Interpreter) ExecStatements(stmts []*ast.Statement, scope *Scope) {
	for _, stmt := range stmts {
		interp.ExecStatement(stmt, scope)
	}
}

// ExecStatement executes a single statement.
func (interp *Interpreter) ExecStatement(stmt *ast.Statement, scope *Scope) {
	if stmt == nil {
		return
	}
	if scope != nil && scope.ctx != nil && scope.ctx.stop != nil && scope.ctx.stop.Load() {
		panic(errTarefaCancelada)
	}

	switch stmt.Type {
	case "var":
		val := interp.EvalExpr(&stmt.VarDecl.Value, scope)
		scope.Set(stmt.VarDecl.Name, val)

	case "assign":
		val := interp.EvalExpr(&stmt.Assign.Value, scope)
		if stmt.Assign.TargetExpr != nil {
			interp.assignTo(stmt.Assign.TargetExpr, val, scope)
		} else if stmt.Assign.Field != "" {
			// Object field assignment
			obj, ok := scope.Get(stmt.Assign.Target)
			if !ok {
				obj = make(map[string]interface{})
				scope.Set(stmt.Assign.Target, obj)
			}
			if m, ok := obj.(map[string]interface{}); ok {
				m[stmt.Assign.Field] = val
			} else {
				panic(interp.errAt(stmt.Pos, 0, "só é possível atribuir '%s' em um mapa, '%s' é %s", stmt.Assign.Field, stmt.Assign.Target, typeName(obj)))
			}
		} else {
			scope.Set(stmt.Assign.Target, val)
		}

	case "if":
		interp.execIf(stmt.If, scope)

	case "for_each":
		interp.execForEach(stmt.ForEach, scope)

	case "while":
		interp.execWhile(stmt.While, scope)

	case "repeat":
		interp.execRepeat(stmt.Repeat, scope)

	case "return":
		val := interp.EvalExpr(stmt.Return, scope)
		panic(signal{Type: signalReturn, Value: val})

	case "break":
		panic(signal{Type: signalBreak})

	case "continue":
		panic(signal{Type: signalContinue})

	case "pause":
		time.Sleep(1 * time.Second)

	case "print":
		val := interp.EvalExpr(stmt.Print, scope)
		msg := toString(val)
		fmt.Println("[germanio]", msg)
		interp.AppendLog(msg)
		if scope.ctx != nil {
			scope.ctx.Output = append(scope.ctx.Output, msg)
		}

	case "call":
		interp.execCall(stmt.Call, scope, stmt.Pos)

	case "expr":
		if stmt.Expr != nil && stmt.Expr.Type == "call" && scope != nil && scope.ctx != nil {
			scope.ctx.discard = stmt.Expr
		}
		interp.EvalExpr(stmt.Expr, scope)

	case "try":
		interp.execTry(stmt.Try, scope)

	case "when":
		interp.execWhen(stmt.When, scope)

	case "concurrency":
		interp.execConcurrency(stmt.Concurrency, scope)
	}
}

func (interp *Interpreter) execWhen(when *ast.WhenStmt, scope *Scope) {
	if when == nil {
		return
	}
	target := interp.EvalExpr(&when.Target, scope)
	for _, c := range when.Cases {
		if c == nil {
			continue
		}
		if c.IsDefault || (c.Pattern != nil && isEqual(target, interp.evalPattern(c.Pattern, scope))) {
			interp.ExecStatements(c.Body, NewScope(scope))
			return
		}
	}
}

func (interp *Interpreter) execConcurrency(concurrency *ast.ConcurrencyStmt, scope *Scope) {
	if concurrency == nil {
		return
	}
	// The base interpreter deliberately executes a structured concurrency block
	// in source order. It preserves deterministic state and avoids concurrent
	// mutation of interpreter scopes, database handles and log buffers. A future
	// worker runtime may schedule independent tasks in parallel behind the same
	// join boundary, but may not change observable program semantics.
	for _, task := range concurrency.Tasks {
		interp.ExecStatement(task, NewScope(scope))
	}
}

func (interp *Interpreter) execIf(ifStmt *ast.IfStmt, scope *Scope) {
	if toBool(interp.EvalExpr(&ifStmt.Condition, scope)) {
		interp.ExecStatements(ifStmt.Body, NewScope(scope))
		return
	}

	for _, elseIf := range ifStmt.ElseIfs {
		if toBool(interp.EvalExpr(&elseIf.Condition, scope)) {
			interp.ExecStatements(elseIf.Body, NewScope(scope))
			return
		}
	}

	if len(ifStmt.Else) > 0 {
		interp.ExecStatements(ifStmt.Else, NewScope(scope))
	}
}

func (interp *Interpreter) execForEach(forEach *ast.ForEachStmt, scope *Scope) {
	collection := interp.EvalExpr(&forEach.Collection, scope)

	var items []interface{}
	switch c := collection.(type) {
	case []interface{}:
		items = c
	case []map[string]interface{}:
		for _, m := range c {
			items = append(items, m)
		}
	case string:
		// If it looks like a model name, query the DB
		if interp.DB != nil {
			if _, ok := interp.DB.Models[strings.ToLower(c)]; ok {
				rows, _, err := interp.DB.Listar(strings.ToLower(c), nil)
				if err == nil {
					for _, r := range rows {
						items = append(items, r)
					}
				}
			}
		}
		if items == nil {
			// Iterate over characters
			for _, ch := range c {
				items = append(items, string(ch))
			}
		}
	case map[string]interface{}:
		keys := make([]string, 0, len(c))
		for k := range c {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			items = append(items, k)
		}
	case nil:
		panic(interp.errAt(forEach.Collection.Pos, 0, "para cada sobre nulo"))
	default:
		panic(interp.errAt(forEach.Collection.Pos, 0, "para cada exige lista, texto ou mapa, recebido %s", typeName(collection)))
	}

	loopScope := NewScope(scope)
	iterations := 0
	for _, item := range items {
		iterations++
		if iterations > maxIterations {
			panic(interp.errAt(forEach.Collection.Pos, 0, "limite de %d iterações excedido em para cada", maxIterations))
		}
		loopScope.SetLocal(forEach.VarName, item)
		shouldBreak := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					if sig, ok := r.(signal); ok {
						switch sig.Type {
						case signalBreak:
							shouldBreak = true
							return
						case signalContinue:
							return
						default:
							panic(r) // re-panic return signals
						}
					}
					panic(r)
				}
			}()
			interp.ExecStatements(forEach.Body, loopScope)
		}()
		if shouldBreak {
			break
		}
	}
}

func (interp *Interpreter) execWhile(whileStmt *ast.WhileStmt, scope *Scope) {
	iterations := 0
	loopScope := NewScope(scope)
	for {
		iterations++
		if iterations > maxIterations {
			panic(interp.errAt(whileStmt.Condition.Pos, 0, "limite de %d iterações excedido em enquanto", maxIterations))
		}
		if !toBool(interp.EvalExpr(&whileStmt.Condition, loopScope)) {
			break
		}
		shouldBreak := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					if sig, ok := r.(signal); ok {
						switch sig.Type {
						case signalBreak:
							shouldBreak = true
							return
						case signalContinue:
							return
						default:
							panic(r)
						}
					}
					panic(r)
				}
			}()
			interp.ExecStatements(whileStmt.Body, loopScope)
		}()
		if shouldBreak {
			break
		}
	}
}

func (interp *Interpreter) execRepeat(repeatStmt *ast.RepeatStmt, scope *Scope) {
	countVal := interp.EvalExpr(&repeatStmt.Count, scope)
	count := int(toNumber(countVal))
	if count > maxIterations {
		count = maxIterations
		interp.AppendLog("AVISO: repeat limitado a 10000 iteracoes")
	}

	loopScope := NewScope(scope)
	for i := 0; i < count; i++ {
		loopScope.SetLocal("i", float64(i))
		shouldBreak := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					if sig, ok := r.(signal); ok {
						switch sig.Type {
						case signalBreak:
							shouldBreak = true
							return
						case signalContinue:
							return
						default:
							panic(r)
						}
					}
					panic(r)
				}
			}()
			interp.ExecStatements(repeatStmt.Body, loopScope)
		}()
		if shouldBreak {
			break
		}
	}
}

func (interp *Interpreter) execCall(call *ast.FuncCall, scope *Scope, pos diagnostics.Position) interface{} {
	if call == nil {
		return nil
	}
	expr := &ast.Expression{Type: "call", Name: call.Name, Object: call.Object, Args: call.Args, Pos: pos}
	if scope != nil && scope.ctx != nil {
		scope.ctx.discard = expr // a call on its own line: the result is thrown away
	}
	return interp.evalCall(expr, scope)
}

const maxCallDepth = 200

// maxParalelo is how many tasks of paralelo run at the same time.
const maxParalelo = 8

// errTarefaCancelada stops a task whose deadline passed.
var errTarefaCancelada = errors.New("tarefa interrompida pelo prazo")

// runTask runs fn as a separate task: its own copy of the execution context
// (call depth, output) so tasks never share mutable interpreter state, and an
// optional stop flag checked before every instruction. A failure becomes the
// task's result ("erro: …"), as before.
func (interp *Interpreter) runTask(fn *ast.FuncDecl, parent *Scope, stop *atomic.Bool) (result interface{}) {
	caller := NewScope(interp.Global)
	if parent != nil && parent.ctx != nil {
		c := *parent.ctx
		c.Output = nil
		c.stop = stop
		caller.ctx = &c
	} else {
		caller.ctx = &Context{stop: stop}
	}
	defer func() {
		if r := recover(); r != nil {
			if sig, ok := r.(signal); ok && sig.Type == signalReturn {
				result = sig.Value
				return
			}
			result = fmt.Sprintf("erro: %v", r)
		}
	}()
	return interp.callFunctionIn(fn, nil, caller, diagnostics.Position{})
}

// callFunction is kept for callers outside a scope (legacy paths).
func (interp *Interpreter) callFunction(fn *ast.FuncDecl, args []interface{}) interface{} {
	return interp.callFunctionIn(fn, args, interp.Global, diagnostics.Position{})
}

func (interp *Interpreter) callFunctionIn(fn *ast.FuncDecl, args []interface{}, caller *Scope, pos diagnostics.Position) interface{} {
	if len(args) > len(fn.Params) {
		panic(interp.errAt(pos, 0, "a função '%s' recebe %d argumento(s), recebeu %d", fn.Name, len(fn.Params), len(args)))
	}
	fnScope := NewScope(interp.Global)
	if caller != nil {
		fnScope.ctx = caller.ctx
	}
	if fnScope.ctx != nil {
		fnScope.ctx.depth++
		defer func() { fnScope.ctx.depth-- }()
		if fnScope.ctx.depth > maxCallDepth {
			panic(interp.errAt(pos, 0, "profundidade máxima de chamadas (%d) excedida em '%s'", maxCallDepth, fn.Name))
		}
	}
	for i, param := range fn.Params {
		if i < len(args) {
			fnScope.SetLocal(param, args[i])
		} else {
			fnScope.SetLocal(param, nil)
		}
	}

	var result interface{}
	func() {
		defer func() {
			if r := recover(); r != nil {
				if sig, ok := r.(signal); ok && sig.Type == signalReturn {
					result = sig.Value
					return
				}
				panic(r)
			}
		}()
		interp.ExecStatements(fn.Body, fnScope)
	}()

	return result
}

func (interp *Interpreter) callDBMethod(object, method string, args []interface{}, scope *Scope) interface{} {
	if interp.DB == nil {
		return nil
	}

	modelName := strings.ToLower(object)
	if _, ok := interp.DB.Models[modelName]; !ok {
		return nil
	}

	switch method {
	case "listar", "list", "todos", "all":
		rows, _, err := interp.DB.Listar(modelName, nil)
		if err != nil {
			interp.AppendLog(fmt.Sprintf("ERRO DB listar: %s", err))
			return nil
		}
		result := make([]interface{}, len(rows))
		for i, r := range rows {
			result[i] = r
		}
		return result

	case "criar", "create":
		if len(args) > 0 {
			data, err := json.Marshal(args[0])
			if err != nil {
				interp.AppendLog(fmt.Sprintf("ERRO DB criar: %s", err))
				return nil
			}
			row, err := interp.DB.Criar(modelName, data)
			if err != nil {
				interp.AppendLog(fmt.Sprintf("ERRO DB criar: %s", err))
				return nil
			}
			return row
		}

	case "atualizar", "update":
		if len(args) >= 2 {
			id := int64(toNumber(args[0]))
			data, err := json.Marshal(args[1])
			if err != nil {
				return nil
			}
			row, err := interp.DB.Atualizar(modelName, id, data)
			if err != nil {
				interp.AppendLog(fmt.Sprintf("ERRO DB atualizar: %s", err))
				return nil
			}
			return row
		}

	case "deletar", "delete":
		if len(args) > 0 {
			id := int64(toNumber(args[0]))
			if err := interp.DB.Deletar(modelName, id); err != nil {
				interp.AppendLog(fmt.Sprintf("ERRO DB deletar: %s", err))
				return nil
			}
			return true
		}

	case "contar", "count":
		rows, total, err := interp.DB.Listar(modelName, nil)
		if err != nil {
			return float64(0)
		}
		if total > 0 {
			return float64(total)
		}
		return float64(len(rows))
	}

	return nil
}

func (interp *Interpreter) execTry(tryStmt *ast.TryStmt, scope *Scope) {
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Check if it's a control flow signal — re-panic those
				if sig, ok := r.(signal); ok {
					panic(sig)
				}
				// Handle the error
				if len(tryStmt.Catch) > 0 {
					catchScope := NewScope(scope)
					if tryStmt.ErrVar != "" {
						msg := fmt.Sprintf("%v", r)
						if re, ok := r.(*RuntimeError); ok {
							msg = re.Message
						}
						catchScope.SetLocal(tryStmt.ErrVar, msg)
					}
					interp.ExecStatements(tryStmt.Catch, catchScope)
				}
			}
		}()
		interp.ExecStatements(tryStmt.Body, NewScope(scope))
	}()
}

// EvalExpr evaluates an expression and returns the result.
func (interp *Interpreter) EvalExpr(expr *ast.Expression, scope *Scope) interface{} {
	if expr == nil {
		return nil
	}

	switch expr.Type {
	case "literal":
		return expr.Value

	case "variable":
		if val, ok := scope.Get(expr.Name); ok {
			return val
		}
		if expr.Canon != "" {
			if val, ok := scope.Get(expr.Canon); ok {
				return val
			}
		}
		// A model name evaluates to itself (para cada t em ticket).
		if interp.isModel(expr.Name) {
			return expr.Name
		}
		panic(interp.errAt(expr.Pos, 0, "variável '%s' não definida.%s", expr.Name, suggest(expr.Name, scope.names())))

	case "binary":
		return interp.evalBinary(expr, scope)

	case "unary":
		return interp.evalUnary(expr, scope)

	case "call":
		return interp.evalCall(expr, scope)

	case "field_access":
		if obj, ok := scope.Get(expr.Object); ok {
			return interp.member(expr.Pos, obj, expr.Field)
		}
		if interp.isModel(expr.Object) {
			// Legacy lazy reference used by older screens and rules.
			return fmt.Sprintf("%s.%s", expr.Object, expr.Field)
		}
		panic(interp.errAt(expr.Pos, 0, "variável '%s' não definida.%s", expr.Object, suggest(expr.Object, scope.names())))

	case "member":
		return interp.member(expr.Pos, interp.EvalExpr(expr.Target, scope), expr.Field)

	case "method":
		target := interp.EvalExpr(expr.Target, scope)
		args := make([]interface{}, len(expr.Args))
		for i, a := range expr.Args {
			args[i] = interp.EvalExpr(a, scope)
		}
		return interp.valueMethod(&Call{Interp: interp, Scope: scope, Pos: expr.Pos, Name: expr.Name}, target, expr.Name, args)

	case "list":
		result := make([]interface{}, len(expr.Elements))
		for i, elem := range expr.Elements {
			result[i] = interp.EvalExpr(elem, scope)
		}
		return result

	case "map":
		result := make(map[string]interface{}, len(expr.Keys))
		for i, k := range expr.Keys {
			result[k] = interp.EvalExpr(expr.Elements[i], scope)
		}
		return result

	case "index":
		var collection interface{}
		switch {
		case expr.Left != nil:
			collection = interp.EvalExpr(expr.Left, scope)
		case expr.Object != "" && expr.Field != "":
			obj, ok := scope.Get(expr.Object)
			if !ok {
				panic(interp.errAt(expr.Pos, 0, "variável '%s' não definida", expr.Object))
			}
			collection = interp.member(expr.Pos, obj, expr.Field)
		default:
			v, ok := scope.Get(expr.Name)
			if !ok {
				panic(interp.errAt(expr.Pos, 0, "variável '%s' não definida", expr.Name))
			}
			collection = v
		}
		return interp.index(expr.Pos, collection, interp.EvalExpr(expr.Index, scope))
	}

	panic(interp.errAt(expr.Pos, 0, "expressão desconhecida: %s", expr.Type))
}

// evalPattern evaluates a quando-case pattern; an undefined bare name is a
// symbol (aprovado -> ...), matching the documented pattern syntax.
func (interp *Interpreter) evalPattern(expr *ast.Expression, scope *Scope) interface{} {
	if expr.Type == "variable" {
		if v, ok := scope.Get(expr.Name); ok {
			return v
		}
		return expr.Name
	}
	return interp.EvalExpr(expr, scope)
}

// names lists visible variable names (for suggestions).
func (s *Scope) names() []string {
	var out []string
	for cur := s; cur != nil; cur = cur.parent {
		cur.rlock()
		for k := range cur.vars {
			out = append(out, k)
		}
		cur.runlock()
	}
	return out
}

func (interp *Interpreter) evalBinary(expr *ast.Expression, scope *Scope) interface{} {
	left := interp.EvalExpr(expr.Left, scope)
	// e/ou stop as soon as the answer is known: `r == nulo ou r.ativo` is safe.
	switch expr.Operator {
	case "e":
		if !toBool(left) {
			return false
		}
		return toBool(interp.EvalExpr(expr.Right, scope))
	case "ou":
		if toBool(left) {
			return true
		}
		return toBool(interp.EvalExpr(expr.Right, scope))
	}
	right := interp.EvalExpr(expr.Right, scope)

	switch expr.Operator {
	case "+":
		// String concatenation if either is string
		if isString(left) || isString(right) {
			return toString(left) + toString(right)
		}
		return toNumber(left) + toNumber(right)
	case "-":
		return toNumber(left) - toNumber(right)
	case "*":
		return toNumber(left) * toNumber(right)
	case "/":
		r := toNumber(right)
		if r == 0 {
			panic(interp.errAt(expr.Pos, 0, "divisão por zero"))
		}
		return toNumber(left) / r
	case "%":
		r := toNumber(right)
		if r == 0 {
			panic(interp.errAt(expr.Pos, 0, "divisão por zero"))
		}
		return math.Mod(toNumber(left), r)
	case "==":
		return isEqual(left, right)
	case "!=":
		return !isEqual(left, right)
	case ">":
		return toNumber(left) > toNumber(right)
	case "<":
		return toNumber(left) < toNumber(right)
	case ">=":
		return toNumber(left) >= toNumber(right)
	case "<=":
		return toNumber(left) <= toNumber(right)
	}

	return nil
}

func (interp *Interpreter) evalUnary(expr *ast.Expression, scope *Scope) interface{} {
	val := interp.EvalExpr(expr.Right, scope)
	switch expr.Operator {
	case "nao":
		return !toBool(val)
	case "-":
		return -toNumber(val)
	}
	return nil
}

func (interp *Interpreter) callObjMethod(obj interface{}, method string, args []interface{}) interface{} {
	switch method {
	case "tamanho", "length", "len":
		switch v := obj.(type) {
		case string:
			return float64(len(v))
		case []interface{}:
			return float64(len(v))
		case map[string]interface{}:
			return float64(len(v))
		}
	}
	return nil
}

// ==================== Built-in Functions ====================

func (interp *Interpreter) callBuiltin(name string, args []interface{}) (interface{}, bool) {
	switch name {
	case "tamanho", "length", "len":
		if len(args) < 1 {
			return float64(0), true
		}
		switch v := args[0].(type) {
		case string:
			return float64(len(v)), true
		case []interface{}:
			return float64(len(v)), true
		case map[string]interface{}:
			return float64(len(v)), true
		default:
			return float64(0), true
		}

	case "tipo", "type":
		if len(args) < 1 {
			return "nulo", true
		}
		switch args[0].(type) {
		case float64:
			return "numero", true
		case string:
			return "texto", true
		case bool:
			return "booleano", true
		case nil:
			return "nulo", true
		case []interface{}:
			return "lista", true
		case map[string]interface{}:
			return "objeto", true
		default:
			return "desconhecido", true
		}

	case "texto", "string":
		if len(args) < 1 {
			return "", true
		}
		return toString(args[0]), true

	case "numero", "number":
		if len(args) < 1 {
			return float64(0), true
		}
		return toNumber(args[0]), true

	case "arredondar", "round":
		if len(args) < 1 {
			return float64(0), true
		}
		return math.Round(toNumber(args[0])), true

	case "aleatorio", "random":
		return rand.Float64(), true

	case "agora", "now":
		return time.Now().Format(time.RFC3339), true

	case "maiusculo", "uppercase", "upper":
		if len(args) < 1 {
			return "", true
		}
		return strings.ToUpper(toString(args[0])), true

	case "minusculo", "lowercase", "lower":
		if len(args) < 1 {
			return "", true
		}
		return strings.ToLower(toString(args[0])), true

	case "contem", "contains":
		if len(args) < 2 {
			return false, true
		}
		if list, ok := args[0].([]interface{}); ok {
			for _, it := range list {
				if isEqual(it, args[1]) {
					return true, true
				}
			}
			return false, true
		}
		if m, ok := args[0].(map[string]interface{}); ok {
			_, has := m[toString(args[1])]
			return has, true
		}
		return strings.Contains(toString(args[0]), toString(args[1])), true

	case "dividir", "split":
		if len(args) < 2 {
			return []interface{}{}, true
		}
		parts := strings.Split(toString(args[0]), toString(args[1]))
		result := make([]interface{}, len(parts))
		for i, p := range parts {
			result[i] = p
		}
		return result, true

	case "juntar", "join":
		if len(args) < 2 {
			return "", true
		}
		if arr, ok := args[0].([]interface{}); ok {
			strs := make([]string, len(arr))
			for i, v := range arr {
				strs[i] = toString(v)
			}
			return strings.Join(strs, toString(args[1])), true
		}
		return "", true

	case "abs":
		if len(args) < 1 {
			return float64(0), true
		}
		return math.Abs(toNumber(args[0])), true

	case "min":
		if len(args) < 2 {
			return float64(0), true
		}
		return math.Min(toNumber(args[0]), toNumber(args[1])), true

	case "max":
		if len(args) < 2 {
			return float64(0), true
		}
		return math.Max(toNumber(args[0]), toNumber(args[1])), true

	case "inteiro", "int":
		if len(args) < 1 {
			return float64(0), true
		}
		return math.Floor(toNumber(args[0])), true

	case "substituir", "replace":
		if len(args) < 3 {
			return "", true
		}
		return strings.ReplaceAll(toString(args[0]), toString(args[1]), toString(args[2])), true

	case "cortar", "trim":
		if len(args) < 1 {
			return "", true
		}
		return strings.TrimSpace(toString(args[0])), true

	case "comeca_com", "starts_with":
		if len(args) < 2 {
			return false, true
		}
		return strings.HasPrefix(toString(args[0]), toString(args[1])), true

	case "termina_com", "ends_with":
		if len(args) < 2 {
			return false, true
		}
		return strings.HasSuffix(toString(args[0]), toString(args[1])), true

	case "substring", "substr", "fatiar":
		if len(args) < 2 {
			return "", true
		}
		s := toString(args[0])
		start := int(toNumber(args[1]))
		if start < 0 {
			start = 0
		}
		if start >= len(s) {
			return "", true
		}
		if len(args) >= 3 {
			end := int(toNumber(args[2]))
			if end > len(s) {
				end = len(s)
			}
			return s[start:end], true
		}
		return s[start:], true

	case "adicionar", "push", "append":
		if len(args) < 2 {
			return args, true
		}
		if arr, ok := args[0].([]interface{}); ok {
			out := make([]interface{}, len(arr), len(arr)+1)
			copy(out, arr)
			return append(out, args[1]), true
		}
		return []interface{}{args[0], args[1]}, true

	case "remover", "pop":
		if len(args) < 1 {
			return nil, true
		}
		if arr, ok := args[0].([]interface{}); ok && len(arr) > 0 {
			return arr[:len(arr)-1], true
		}
		return args[0], true

	case "reverter", "reverse":
		if len(args) < 1 {
			return nil, true
		}
		if arr, ok := args[0].([]interface{}); ok {
			result := make([]interface{}, len(arr))
			for i, v := range arr {
				result[len(arr)-1-i] = v
			}
			return result, true
		}
		if s, ok := args[0].(string); ok {
			runes := []rune(s)
			for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
				runes[i], runes[j] = runes[j], runes[i]
			}
			return string(runes), true
		}
		return nil, true

	case "chaves", "keys":
		if len(args) < 1 {
			return []interface{}{}, true
		}
		if m, ok := args[0].(map[string]interface{}); ok {
			names := make([]string, 0, len(m))
			for k := range m {
				names = append(names, k)
			}
			sort.Strings(names)
			keys := make([]interface{}, len(names))
			for i, k := range names {
				keys[i] = k
			}
			return keys, true
		}
		return []interface{}{}, true

	case "valores", "values":
		if len(args) < 1 {
			return []interface{}{}, true
		}
		if m, ok := args[0].(map[string]interface{}); ok {
			vals := make([]interface{}, 0, len(m))
			for _, v := range m {
				vals = append(vals, v)
			}
			return vals, true
		}
		return []interface{}{}, true

	case "json":
		if len(args) < 1 {
			return "", true
		}
		if s, ok := args[0].(string); ok {
			var result interface{}
			if err := json.Unmarshal([]byte(s), &result); err != nil {
				return nil, true
			}
			return result, true
		}
		data, _ := json.Marshal(args[0])
		return string(data), true

	case "formato_data", "format_date":
		if len(args) < 1 {
			return "", true
		}
		t, err := time.Parse(time.RFC3339, toString(args[0]))
		if err != nil {
			return toString(args[0]), true
		}
		if len(args) >= 2 {
			format := toString(args[1])
			// Simple format mapping
			format = strings.ReplaceAll(format, "DD", "02")
			format = strings.ReplaceAll(format, "MM", "01")
			format = strings.ReplaceAll(format, "YYYY", "2006")
			format = strings.ReplaceAll(format, "HH", "15")
			format = strings.ReplaceAll(format, "mm", "04")
			format = strings.ReplaceAll(format, "ss", "05")
			return t.Format(format), true
		}
		return t.Format("02/01/2006"), true

	case "potencia", "pow", "power":
		if len(args) < 2 {
			return float64(0), true
		}
		return math.Pow(toNumber(args[0]), toNumber(args[1])), true

	case "raiz", "sqrt":
		if len(args) < 1 {
			return float64(0), true
		}
		return math.Sqrt(toNumber(args[0])), true

	case "chamar", "call", "http":
		if len(args) < 1 {
			return nil, true
		}
		url := toString(args[0])
		method := "GET"
		if len(args) >= 2 {
			method = strings.ToUpper(toString(args[1]))
		}
		var body []byte
		if len(args) >= 3 {
			body = []byte(toString(args[2]))
		}
		if interp.HTTPClient != nil {
			resp, err := interp.HTTPClient.Chamar(method, url, body)
			if err != nil {
				interp.AppendLog(fmt.Sprintf("ERRO HTTP: %s", err))
				return nil, true
			}
			var result interface{}
			if json.Unmarshal(resp, &result) == nil {
				return result, true
			}
			return string(resp), true
		}
		return nil, true

	case "paralelo", "parallel":
		// paralelo([f1, f2, …]) — run the functions at the same time, at most
		// maxParalelo at once; the results keep the order of the tasks.
		if len(args) < 1 {
			return []interface{}{}, true
		}
		tasks, ok := args[0].([]interface{})
		if !ok {
			return []interface{}{}, true
		}
		results := make([]interface{}, len(tasks))
		slots := make(chan struct{}, maxParalelo)
		var wg sync.WaitGroup
		for i, task := range tasks {
			name, _ := task.(string)
			fn, exists := interp.Functions[name]
			if !exists {
				continue
			}
			wg.Add(1)
			slots <- struct{}{}
			go func(idx int, fn *ast.FuncDecl) {
				defer wg.Done()
				defer func() { <-slots }()
				results[idx] = interp.runTask(fn, interp.Global, nil)
			}(i, fn)
		}
		wg.Wait()
		return results, true

	case "esperar", "await", "wait":
		// esperar(milliseconds) — async sleep
		if len(args) < 1 {
			return nil, true
		}
		ms := int(toNumber(args[0]))
		if ms > 30000 {
			ms = 30000 // max 30 seconds
		}
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return nil, true

	case "timeout":
		// timeout(func_name, milliseconds) — run a function with a deadline.
		// When the deadline passes the task is stopped (at its next
		// instruction) and timeout waits for it to end: nothing keeps running
		// after the caller has moved on.
		if len(args) < 2 {
			return nil, true
		}
		funcName := toString(args[0])
		ms := int(toNumber(args[1]))
		if ms > 60000 {
			ms = 60000
		}
		fn, exists := interp.Functions[funcName]
		if !exists {
			return nil, true
		}
		stop := new(atomic.Bool)
		resultCh := make(chan interface{}, 1)
		go func() { resultCh <- interp.runTask(fn, interp.Global, stop) }()
		select {
		case result := <-resultCh:
			return result, true
		case <-time.After(time.Duration(ms) * time.Millisecond):
			stop.Store(true)
			<-resultCh
			interp.AppendLog(fmt.Sprintf("AVISO: timeout em '%s' após %dms; a tarefa foi interrompida", funcName, ms))
			return nil, true
		}

	case "consultar_paralelo", "parallel_query":
		// consultar_paralelo(["modelo1", "modelo2"]) — query multiple models in parallel
		if len(args) < 1 || interp.DB == nil {
			return []interface{}{}, true
		}
		if models, ok := args[0].([]interface{}); ok {
			results := make(map[string]interface{})
			var wg sync.WaitGroup
			var mu sync.Mutex
			for _, m := range models {
				modelName := strings.ToLower(toString(m))
				if _, exists := interp.DB.Models[modelName]; !exists {
					continue
				}
				wg.Add(1)
				go func(name string) {
					defer wg.Done()
					rows, _, err := interp.DB.Listar(name, nil)
					if err != nil {
						return
					}
					items := make([]interface{}, len(rows))
					for i, r := range rows {
						items[i] = r
					}
					mu.Lock()
					results[name] = items
					mu.Unlock()
				}(modelName)
			}
			wg.Wait()
			return results, true
		}
		return map[string]interface{}{}, true

	case "chamar_async", "async_call", "http_async":
		// chamar_async(["url1", "url2"]) — fetch multiple URLs in parallel
		if len(args) < 1 {
			return []interface{}{}, true
		}
		if urls, ok := args[0].([]interface{}); ok {
			results := make([]interface{}, len(urls))
			slots := make(chan struct{}, maxParalelo)
			var wg sync.WaitGroup
			for i, u := range urls {
				wg.Add(1)
				slots <- struct{}{}
				go func(idx int, urlStr string) {
					defer wg.Done()
					defer func() { <-slots }()
					if interp.HTTPClient != nil {
						resp, err := interp.HTTPClient.Chamar("GET", urlStr, nil)
						if err != nil {
							results[idx] = map[string]interface{}{"erro": err.Error()}
							return
						}
						var parsed interface{}
						if json.Unmarshal(resp, &parsed) == nil {
							results[idx] = parsed
						} else {
							results[idx] = string(resp)
						}
					}
				}(i, toString(u))
			}
			wg.Wait()
			return results, true
		}
		return []interface{}{}, true

	case "ia_completar", "ai_complete", "ai":
		// ia_completar("prompt") or ia_completar("prompt", "provedor")
		// Uses OPENAI_KEY, CLAUDE_KEY, or GEMINI_KEY from env
		if len(args) < 1 {
			return "", true
		}
		prompt := toString(args[0])
		provider := "openai"
		if len(args) >= 2 {
			provider = strings.ToLower(toString(args[1]))
		}

		var apiKey, aiURL, bodyJSON string
		switch provider {
		case "openai", "chatgpt", "gpt":
			apiKey = os.Getenv("OPENAI_KEY")
			if apiKey == "" {
				apiKey = os.Getenv("OPENAI_API_KEY")
			}
			aiURL = "https://api.openai.com/v1/chat/completions"
			bodyJSON = fmt.Sprintf(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":%q}],"max_tokens":1000}`, prompt)
		case "claude", "anthropic":
			apiKey = os.Getenv("CLAUDE_KEY")
			if apiKey == "" {
				apiKey = os.Getenv("ANTHROPIC_API_KEY")
			}
			aiURL = "https://api.anthropic.com/v1/messages"
			bodyJSON = fmt.Sprintf(`{"model":"claude-sonnet-4-20250514","max_tokens":1000,"messages":[{"role":"user","content":%q}]}`, prompt)
		case "gemini", "google":
			apiKey = os.Getenv("GEMINI_KEY")
			if apiKey == "" {
				apiKey = os.Getenv("GOOGLE_API_KEY")
			}
			aiURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=" + apiKey
			bodyJSON = fmt.Sprintf(`{"contents":[{"parts":[{"text":%q}]}]}`, prompt)
		default:
			interp.AppendLog("ERRO IA: provedor desconhecido: " + provider)
			return nil, true
		}

		if apiKey == "" && provider != "gemini" {
			interp.AppendLog("ERRO IA: defina " + strings.ToUpper(provider) + "_KEY no .env")
			return nil, true
		}

		if interp.HTTPClient == nil {
			interp.AppendLog("ERRO IA: HTTP client nao disponivel")
			return nil, true
		}

		// Make the API call
		resp, err := interp.HTTPClient.Chamar("POST", aiURL, []byte(bodyJSON))
		if err != nil {
			interp.AppendLog(fmt.Sprintf("ERRO IA: %s", err))
			return nil, true
		}

		// Parse response based on provider
		var aiResult interface{}
		if err := json.Unmarshal(resp, &aiResult); err != nil {
			return string(resp), true
		}

		// Extract text from response
		if m, ok := aiResult.(map[string]interface{}); ok {
			// OpenAI format
			if choices, ok := m["choices"].([]interface{}); ok && len(choices) > 0 {
				if choice, ok := choices[0].(map[string]interface{}); ok {
					if msg, ok := choice["message"].(map[string]interface{}); ok {
						return msg["content"], true
					}
				}
			}
			// Claude format
			if content, ok := m["content"].([]interface{}); ok && len(content) > 0 {
				if block, ok := content[0].(map[string]interface{}); ok {
					return block["text"], true
				}
			}
			// Gemini format
			if candidates, ok := m["candidates"].([]interface{}); ok && len(candidates) > 0 {
				if cand, ok := candidates[0].(map[string]interface{}); ok {
					if cont, ok := cand["content"].(map[string]interface{}); ok {
						if parts, ok := cont["parts"].([]interface{}); ok && len(parts) > 0 {
							if part, ok := parts[0].(map[string]interface{}); ok {
								return part["text"], true
							}
						}
					}
				}
			}
			// Return raw if can't parse
			return string(resp), true
		}
		return string(resp), true

	case "ia_classificar", "ai_classify":
		// ia_classificar("texto", "categoria1, categoria2, categoria3")
		if len(args) < 2 {
			return "", true
		}
		texto := toString(args[0])
		categorias := toString(args[1])
		prompt := fmt.Sprintf("Classifique o seguinte texto em uma dessas categorias: %s. Responda APENAS com o nome da categoria, nada mais.\n\nTexto: %s", categorias, texto)
		result, _ := interp.callBuiltin("ia_completar", []interface{}{prompt})
		return result, true

	case "ia_resumir", "ai_summarize":
		// ia_resumir("texto longo")
		if len(args) < 1 {
			return "", true
		}
		prompt := "Resuma o seguinte texto em no maximo 2 frases:\n\n" + toString(args[0])
		result, _ := interp.callBuiltin("ia_completar", []interface{}{prompt})
		return result, true

	case "ia_traduzir", "ai_translate":
		// ia_traduzir("texto", "ingles")
		if len(args) < 2 {
			return "", true
		}
		prompt := fmt.Sprintf("Traduza para %s. Responda APENAS com a traducao:\n\n%s", toString(args[1]), toString(args[0]))
		result, _ := interp.callBuiltin("ia_completar", []interface{}{prompt})
		return result, true

	case "ia_sentimento", "ai_sentiment":
		// ia_sentimento("texto") returns "positivo", "negativo", or "neutro"
		if len(args) < 1 {
			return "", true
		}
		prompt := "Analise o sentimento do texto e responda APENAS com: positivo, negativo, ou neutro.\n\nTexto: " + toString(args[0])
		result, _ := interp.callBuiltin("ia_completar", []interface{}{prompt})
		return result, true

	// ==================== Payment Integrations ====================

	case "pix_qrcode", "pix":
		// pix_qrcode(valor, chave_pix, nome_recebedor)
		// Returns a PIX copy-paste code (EMV format)
		if len(args) < 2 {
			return nil, true
		}
		valor := toNumber(args[0])
		chave := toString(args[1])
		nome := "Germanio App"
		if len(args) >= 3 {
			nome = toString(args[2])
		}
		// Generate PIX EMV code
		pixCode := gerarPixCode(valor, chave, nome)
		return pixCode, true

	case "stripe_link", "stripe":
		// stripe_link(valor, descricao) - requires STRIPE_KEY env
		if len(args) < 1 {
			return nil, true
		}
		apiKey := os.Getenv("STRIPE_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("STRIPE_SECRET_KEY")
		}
		if apiKey == "" {
			interp.AppendLog("ERRO: defina STRIPE_KEY no .env")
			return nil, true
		}
		valor := int(toNumber(args[0]) * 100) // cents
		desc := "Pagamento"
		if len(args) >= 2 {
			desc = toString(args[1])
		}
		// Create Stripe payment link via API
		body := fmt.Sprintf("unit_amount=%d&currency=brl&product_data[name]=%s", valor, desc)
		if interp.HTTPClient != nil {
			// Note: Stripe uses form-encoded, not JSON - simplified
			interp.AppendLog(fmt.Sprintf("Stripe: R$%.2f - %s (requer integracao completa)", toNumber(args[0]), desc))
		}
		_ = body
		return fmt.Sprintf("stripe://pay/%d/%s", valor, desc), true

	case "mercadopago_link", "mercadopago", "mp":
		// mercadopago_link(valor, descricao)
		if len(args) < 1 {
			return nil, true
		}
		apiKey := os.Getenv("MP_ACCESS_TOKEN")
		if apiKey == "" {
			interp.AppendLog("ERRO: defina MP_ACCESS_TOKEN no .env")
			return nil, true
		}
		valor := toNumber(args[0])
		desc := "Pagamento"
		if len(args) >= 2 {
			desc = toString(args[1])
		}
		if interp.HTTPClient != nil {
			body := fmt.Sprintf(`{"items":[{"title":%q,"quantity":1,"unit_price":%.2f}],"back_urls":{"success":"http://localhost:8080"}}`, desc, valor)
			resp, err := interp.HTTPClient.Chamar("POST", "https://api.mercadopago.com/checkout/preferences?access_token="+apiKey, []byte(body))
			if err != nil {
				interp.AppendLog(fmt.Sprintf("ERRO MercadoPago: %s", err))
				return nil, true
			}
			var result map[string]interface{}
			if json.Unmarshal(resp, &result) == nil {
				if link, ok := result["init_point"].(string); ok {
					return link, true
				}
			}
			return string(resp), true
		}
		return nil, true

	// ==================== Messaging Integrations ====================

	case "telegram_enviar", "telegram_send", "telegram":
		// telegram_enviar(chat_id, mensagem) - requires TELEGRAM_BOT_TOKEN
		if len(args) < 2 {
			return nil, true
		}
		token := os.Getenv("TELEGRAM_BOT_TOKEN")
		if token == "" {
			interp.AppendLog("ERRO: defina TELEGRAM_BOT_TOKEN no .env")
			return nil, true
		}
		chatID := toString(args[0])
		msg := toString(args[1])
		tgURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
		body := fmt.Sprintf(`{"chat_id":%q,"text":%q,"parse_mode":"HTML"}`, chatID, msg)
		if interp.HTTPClient != nil {
			resp, err := interp.HTTPClient.Chamar("POST", tgURL, []byte(body))
			if err != nil {
				interp.AppendLog(fmt.Sprintf("ERRO Telegram: %s", err))
				return nil, true
			}
			return string(resp), true
		}
		return nil, true

	case "discord_enviar", "discord_send", "discord":
		// discord_enviar(webhook_url, mensagem)
		if len(args) < 2 {
			return nil, true
		}
		webhookURL := toString(args[0])
		if webhookURL == "" {
			webhookURL = os.Getenv("DISCORD_WEBHOOK")
		}
		msg := toString(args[1])
		body := fmt.Sprintf(`{"content":%q}`, msg)
		if interp.HTTPClient != nil {
			_, err := interp.HTTPClient.Chamar("POST", webhookURL, []byte(body))
			if err != nil {
				interp.AppendLog(fmt.Sprintf("ERRO Discord: %s", err))
				return false, true
			}
			return true, true
		}
		return nil, true

	case "slack_enviar", "slack_send", "slack":
		// slack_enviar(webhook_url, mensagem)
		if len(args) < 2 {
			return nil, true
		}
		webhookURL := toString(args[0])
		if webhookURL == "" {
			webhookURL = os.Getenv("SLACK_WEBHOOK")
		}
		msg := toString(args[1])
		body := fmt.Sprintf(`{"text":%q}`, msg)
		if interp.HTTPClient != nil {
			_, err := interp.HTTPClient.Chamar("POST", webhookURL, []byte(body))
			if err != nil {
				interp.AppendLog(fmt.Sprintf("ERRO Slack: %s", err))
				return false, true
			}
			return true, true
		}
		return nil, true

	case "sms_enviar", "sms_send", "sms":
		// sms_enviar(telefone, mensagem) - requires TWILIO_SID, TWILIO_TOKEN, TWILIO_FROM
		if len(args) < 2 {
			return nil, true
		}
		sid := os.Getenv("TWILIO_SID")
		token := os.Getenv("TWILIO_TOKEN")
		from := os.Getenv("TWILIO_FROM")
		if sid == "" || token == "" {
			interp.AppendLog("ERRO: defina TWILIO_SID, TWILIO_TOKEN e TWILIO_FROM no .env")
			return nil, true
		}
		to := toString(args[0])
		msg := toString(args[1])
		interp.AppendLog(fmt.Sprintf("SMS para %s: %s (via Twilio %s)", to, msg, from))
		// Twilio uses Basic Auth + form encoding - log for now
		return true, true

	// ==================== Productivity Integrations ====================

	case "planilha_exportar", "sheets_export", "google_sheets":
		// planilha_exportar(modelo) - exports model data as array for sheets
		if len(args) < 1 || interp.DB == nil {
			return nil, true
		}
		modelName := strings.ToLower(toString(args[0]))
		rows, _, err := interp.DB.Listar(modelName, nil)
		if err != nil {
			interp.AppendLog(fmt.Sprintf("ERRO Sheets: %s", err))
			return nil, true
		}
		// Convert to array of arrays for sheets compatibility
		sheetResult := make([]interface{}, len(rows))
		for i, r := range rows {
			sheetResult[i] = r
		}
		interp.AppendLog(fmt.Sprintf("Exportado %d registros de %s", len(rows), modelName))
		return sheetResult, true

	case "webhook_enviar", "webhook_send", "webhook":
		// webhook_enviar(url, dados)
		if len(args) < 2 {
			return nil, true
		}
		whURL := toString(args[0])
		var whBody []byte
		switch v := args[1].(type) {
		case string:
			whBody = []byte(v)
		case map[string]interface{}:
			whBody, _ = json.Marshal(v)
		default:
			whBody, _ = json.Marshal(args[1])
		}
		if interp.HTTPClient != nil {
			resp, err := interp.HTTPClient.Chamar("POST", whURL, whBody)
			if err != nil {
				interp.AppendLog(fmt.Sprintf("ERRO Webhook: %s", err))
				return nil, true
			}
			var whResult interface{}
			if json.Unmarshal(resp, &whResult) == nil {
				return whResult, true
			}
			return string(resp), true
		}
		return nil, true

	case "gerar_pdf", "generate_pdf", "pdf":
		// gerar_pdf(titulo, conteudo) - generates HTML that can be printed as PDF
		if len(args) < 2 {
			return nil, true
		}
		titulo := toString(args[0])
		conteudo := toString(args[1])
		html := fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"><title>%s</title><style>body{font-family:system-ui;padding:40px;max-width:800px;margin:0 auto}h1{color:#333;border-bottom:2px solid #6366f1;padding-bottom:10px}@media print{body{padding:20px}}</style></head><body><h1>%s</h1>%s<footer style="margin-top:40px;text-align:center;color:#999;font-size:12px">Gerado por Germanio</footer></body></html>`, titulo, titulo, conteudo)
		return html, true

	case "gerar_csv", "generate_csv", "csv":
		// gerar_csv(modelo) - returns CSV string
		if len(args) < 1 || interp.DB == nil {
			return "", true
		}
		modelName := strings.ToLower(toString(args[0]))
		rows, _, err := interp.DB.Listar(modelName, nil)
		if err != nil {
			return "", true
		}
		if len(rows) == 0 {
			return "", true
		}
		// Build CSV
		var csvBuilder strings.Builder
		// Headers
		first := true
		for k := range rows[0] {
			if !first {
				csvBuilder.WriteString(",")
			}
			csvBuilder.WriteString(k)
			first = false
		}
		csvBuilder.WriteString("\n")
		// Data
		for _, row := range rows {
			first = true
			for _, v := range row {
				if !first {
					csvBuilder.WriteString(",")
				}
				csvBuilder.WriteString(fmt.Sprintf("%v", v))
				first = false
			}
			csvBuilder.WriteString("\n")
		}
		return csvBuilder.String(), true

	case "notificar", "notify":
		// notificar(mensagem) - adds to log and broadcasts via WebSocket concept
		if len(args) < 1 {
			return nil, true
		}
		msg := toString(args[0])
		interp.AppendLog("NOTIFICACAO: " + msg)
		fmt.Printf("[germanio] NOTIFICACAO: %s\n", msg)
		return true, true

	case "data_formatada", "formatted_date":
		// data_formatada() - returns current date in DD/MM/YYYY HH:MM format
		return time.Now().Format("02/01/2006 15:04"), true

	case "dias_entre", "days_between":
		// dias_entre(data1, data2) - returns number of days between dates
		if len(args) < 2 {
			return float64(0), true
		}
		dateFormats := []string{time.RFC3339, "2006-01-02", "02/01/2006", "2006-01-02T15:04:05Z"}
		var t1, t2 time.Time
		var err1, err2 error
		s1, s2 := toString(args[0]), toString(args[1])
		for _, f := range dateFormats {
			t1, err1 = time.Parse(f, s1)
			if err1 == nil {
				break
			}
		}
		for _, f := range dateFormats {
			t2, err2 = time.Parse(f, s2)
			if err2 == nil {
				break
			}
		}
		if err1 != nil || err2 != nil {
			return float64(0), true
		}
		diff := t2.Sub(t1).Hours() / 24
		return math.Abs(diff), true

	case "uuid", "gerar_id":
		// uuid() - generates a simple unique ID
		return fmt.Sprintf("%x-%x-%x", time.Now().UnixNano(), rand.Int63(), rand.Int31()), true

	case "base64_codificar", "base64_encode":
		if len(args) < 1 {
			return "", true
		}
		return base64.StdEncoding.EncodeToString([]byte(toString(args[0]))), true

	case "base64_decodificar", "base64_decode":
		if len(args) < 1 {
			return "", true
		}
		decoded, err := base64.StdEncoding.DecodeString(toString(args[0]))
		if err != nil {
			return "", true
		}
		return string(decoded), true

	case "hash_md5", "md5":
		if len(args) < 1 {
			return "", true
		}
		h := md5.Sum([]byte(toString(args[0])))
		return fmt.Sprintf("%x", h), true

	case "hash_sha256", "sha256":
		if len(args) < 1 {
			return "", true
		}
		h := sha256.Sum256([]byte(toString(args[0])))
		return fmt.Sprintf("%x", h), true
	}

	return extraBuiltin(name, args)
}

// gerarPixCode generates a simplified PIX EMV code
func gerarPixCode(valor float64, chave, nome string) string {
	// Simplified PIX payload (real implementation needs CRC16)
	payload := fmt.Sprintf("00020126330014br.gov.bcb.pix01%02d%s5204000053039865802BR5913%s6009SAO PAULO62070503***6304", len(chave), chave, nome)
	// CRC16-CCITT placeholder
	return payload + "ABCD"
}

// ==================== Type Helpers ====================

func toNumber(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0
		}
		return f
	case bool:
		if val {
			return 1
		}
		return 0
	default:
		return 0
	}
}

func toString(v interface{}) string {
	if v == nil {
		return "nulo"
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == math.Floor(val) && !math.IsInf(val, 0) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case bool:
		if val {
			return "verdadeiro"
		}
		return "falso"
	case []interface{}:
		parts := make([]string, len(val))
		for i, v := range val {
			parts[i] = toString(v)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]interface{}:
		data, _ := json.Marshal(val)
		return string(data)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func toBool(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case float64:
		return val != 0
	case int:
		return val != 0
	case string:
		return val != "" && val != "0" && val != "falso" && val != "false"
	case []interface{}:
		return len(val) > 0
	case map[string]interface{}:
		return len(val) > 0
	default:
		return true
	}
}

func isString(v interface{}) bool {
	_, ok := v.(string)
	return ok
}

func isEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	// Try numeric comparison
	aNum, aIsNum := tryNumber(a)
	bNum, bIsNum := tryNumber(b)
	if aIsNum && bIsNum {
		return aNum == bNum
	}
	// Fall back to string comparison
	return toString(a) == toString(b)
}

func tryNumber(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err == nil {
			return f, true
		}
	}
	return 0, false
}

// Run executes an entire program's scripts and registers functions.
func (interp *Interpreter) Run(program *ast.Program) {
	// Register functions
	for _, fn := range program.Functions {
		interp.RegisterFunction(fn)
	}

	// Execute top-level scripts
	func() {
		defer func() {
			if r := recover(); r != nil {
				if sig, ok := r.(signal); ok && sig.Type == signalReturn {
					return // top-level return is fine
				}
				interp.AppendLog(fmt.Sprintf("ERRO runtime: %v", r))
				fmt.Printf("[germanio] ERRO runtime: %v\n", r)
			}
		}()
		interp.Global.ctx = &Context{AllowGlobal: true}
		defer func() { interp.Global.ctx = nil }()
		interp.ExecStatements(program.Scripts, interp.Global)
	}()
}

// Eval parses and executes a code snippet, returning output.
func (interp *Interpreter) Eval(code string) ([]string, error) {
	// We need to import the lexer and parser here
	// But to avoid circular imports, we accept pre-parsed statements
	return nil, fmt.Errorf("use EvalStatements instead")
}

// EvalStatements executes statements and returns the log output.
func (interp *Interpreter) EvalStatements(stmts []*ast.Statement, fns []*ast.FuncDecl) []string {
	// Register any new functions
	for _, fn := range fns {
		interp.RegisterFunction(fn)
	}

	// Clear logs for this eval
	interp.logMu.Lock()
	startIdx := len(interp.LogBuffer)
	interp.logMu.Unlock()

	func() {
		defer func() {
			if r := recover(); r != nil {
				if sig, ok := r.(signal); ok && sig.Type == signalReturn {
					return
				}
				interp.AppendLog(fmt.Sprintf("ERRO: %v", r))
			}
		}()
		scope := NewScope(interp.Global)
		scope.ctx = &Context{}
		interp.ExecStatements(stmts, scope)
	}()

	interp.logMu.Lock()
	defer interp.logMu.Unlock()
	if startIdx >= len(interp.LogBuffer) {
		return nil
	}
	return interp.LogBuffer[startIdx:]
}
