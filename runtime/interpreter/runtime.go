package interpreter

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/diagnostics"
	"github.com/flaviokalleu/germanio/runtime/banco"
	"golang.org/x/crypto/bcrypt"
)

// RuntimeError is a failure raised while executing .ge code. Status is the
// HTTP status a route answers with when the error is not caught (0 = 500).
type RuntimeError struct {
	Status  int
	Message string
	Pos     diagnostics.Position
	// Payload is the structured value given to falhar (maps, lists).
	Payload any
}

func (e *RuntimeError) Error() string {
	if e.Pos.Line > 0 {
		file := e.Pos.File
		if file == "" {
			file = "<fonte>"
		}
		return fmt.Sprintf("%s:%d: %s", file, e.Pos.Line, e.Message)
	}
	return e.Message
}

// Context carries per-execution state: the HTTP exchange of a route (nil for
// startup scripts and background tasks), call depth and whether top-level
// globals may be written.
type Context struct {
	// DB, when set, is the transaction every operation of this execution joins.
	DB      *banco.Banco
	Request *http.Request
	Writer  http.ResponseWriter
	// Written is set by a capability that answered the request itself
	// (for example the git smart HTTP handler).
	Written bool
	// SetCookies are added to the response by the server.
	SetCookies []*http.Cookie
	// Values lets capabilities attach request-scoped state.
	Values map[string]any
	// Output collects what mostrar printed during this execution.
	Output      []string
	AllowGlobal bool
	depth       int
	// stop, when set, interrupts the execution at its next instruction.
	stop *atomic.Bool
}

// Response is what a route produced.
type Response struct {
	Status   int
	Body     any
	Headers  map[string]string
	Redirect string
	// Raw, when non-nil, is written verbatim with ContentType.
	Raw         []byte
	ContentType string
}

const signalRespond signalType = 100

// Call is handed to module functions.
type Call struct {
	Interp *Interpreter
	Scope  *Scope
	Pos    diagnostics.Position
	Name   string
}

// dbOf is the database of an execution: its transaction, if any.
func (interp *Interpreter) dbOf(ctx *Context) *banco.Banco {
	if ctx != nil && ctx.DB != nil {
		return ctx.DB
	}
	return interp.DB
}

// Ctx returns the execution context (never nil).
func (c *Call) Ctx() *Context {
	if c.Scope != nil && c.Scope.ctx != nil {
		return c.Scope.ctx
	}
	return &Context{}
}

// Fail builds a RuntimeError at the call position.
func (c *Call) Fail(status int, format string, a ...any) *RuntimeError {
	return &RuntimeError{Status: status, Message: fmt.Sprintf(format, a...), Pos: c.Pos}
}

// Arg helpers panic with a positioned error when an argument is missing or
// has the wrong type, so module functions stay short and failures are loud.
func (c *Call) Arg(args []any, i int, what string) any {
	if i >= len(args) {
		panic(c.Fail(0, "%s: falta o argumento %d (%s)", c.Name, i+1, what))
	}
	return args[i]
}

func (c *Call) Str(args []any, i int, what string) string {
	v := c.Arg(args, i, what)
	s, ok := v.(string)
	if !ok {
		panic(c.Fail(0, "%s: o argumento %d (%s) deve ser texto, recebido %s", c.Name, i+1, what, typeName(v)))
	}
	return s
}

func (c *Call) Num(args []any, i int, what string) float64 {
	v := c.Arg(args, i, what)
	f, ok := tryNumber(v)
	if !ok {
		panic(c.Fail(0, "%s: o argumento %d (%s) deve ser número, recebido %s", c.Name, i+1, what, typeName(v)))
	}
	return f
}

func (c *Call) Map(args []any, i int, what string) map[string]any {
	v := c.Arg(args, i, what)
	m, ok := v.(map[string]any)
	if !ok {
		panic(c.Fail(0, "%s: o argumento %d (%s) deve ser um mapa {…}, recebido %s", c.Name, i+1, what, typeName(v)))
	}
	return m
}

// OptMap returns an optional map argument or an empty map.
func (c *Call) OptMap(args []any, i int, what string) map[string]any {
	if i >= len(args) || args[i] == nil {
		return map[string]any{}
	}
	return c.Map(args, i, what)
}

func (c *Call) List(args []any, i int, what string) []any {
	v := c.Arg(args, i, what)
	l, ok := v.([]any)
	if !ok {
		panic(c.Fail(0, "%s: o argumento %d (%s) deve ser uma lista, recebido %s", c.Name, i+1, what, typeName(v)))
	}
	return l
}

// ModuleFunc implements name.funcao(args) for a capability module.
type ModuleFunc func(c *Call, args []any) any

// RegisterModule exposes Go capabilities to .ge as modulo.funcao(...).
func (interp *Interpreter) RegisterModule(name string, funcs map[string]ModuleFunc) {
	if interp.Modules == nil {
		interp.Modules = map[string]map[string]ModuleFunc{}
	}
	if interp.Modules[name] == nil {
		interp.Modules[name] = map[string]ModuleFunc{}
	}
	for k, f := range funcs {
		interp.Modules[name][k] = f
	}
}

func typeName(v any) string {
	switch v.(type) {
	case nil:
		return "nulo"
	case string:
		return "texto"
	case float64, int64, int:
		return "número"
	case bool:
		return "booleano"
	case []any:
		return "lista"
	case map[string]any:
		return "mapa"
	}
	return fmt.Sprintf("%T", v)
}

func (interp *Interpreter) errAt(pos diagnostics.Position, status int, format string, a ...any) *RuntimeError {
	return &RuntimeError{Status: status, Message: fmt.Sprintf(format, a...), Pos: pos}
}

// suggest returns " Você quis dizer X?" for the closest candidate.
func suggest(name string, candidates []string) string {
	best, bestD := "", 3
	for _, c := range candidates {
		if d := levenshtein(name, c); d < bestD && d > 0 {
			best, bestD = c, d
		}
	}
	if best == "" {
		return ""
	}
	return fmt.Sprintf(" Você quis dizer '%s'?", best)
}

func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}

func (interp *Interpreter) isModel(name string) bool {
	if interp.DB == nil {
		return false
	}
	_, ok := interp.DB.Models[strings.ToLower(name)]
	return ok
}

var dbMethods = map[string]bool{
	"buscar": true, "encontrar": true, "filtrar": true, "paginar": true, "contar": true, "existe": true,
	"criar": true, "atualizar": true, "deletar": true, "apagar_onde": true,
	"verificar_senha": true, "por_segredo": true, "revogar": true,
	"listar": true, "todos": true, "list": true, "all": true, "create": true, "update": true, "delete": true, "count": true,
}

func (interp *Interpreter) knownFunctionNames() []string {
	names := []string{"responder", "redirecionar", "falhar", "obter", "tem"}
	for n := range interp.Functions {
		names = append(names, n)
	}
	names = append(names, builtinNames...)
	return names
}

// evalCall resolves f(args), modelo.op(args), valor.op(args) and
// modulo.funcao(args). The order is fixed and documented in docs/FULLSTACK.md:
//  1. modelo.<operação de banco> when the object is a model
//  2. variável.op(...) — op is a builtin applied to the value
//  3. modulo.funcao(...) for registered capability modules
func (interp *Interpreter) evalCall(expr *ast.Expression, scope *Scope) any {
	args := make([]any, len(expr.Args))
	for i, a := range expr.Args {
		args[i] = interp.EvalExpr(a, scope)
	}
	c := &Call{Interp: interp, Scope: scope, Pos: expr.Pos, Name: expr.Name}
	if expr.Object == "" {
		return interp.callNamed(c, expr.Name, expr.Canon, args)
	}
	obj := expr.Object
	c.Name = obj + "." + expr.Name
	// Records are plain maps without methods, so modelo.operação always
	// means the database even when a variable holds a record of that model.
	if interp.isModel(obj) && dbMethods[expr.Name] {
		return interp.dbCall(c, strings.ToLower(obj), expr.Name, args)
	}
	if v, ok := scope.Get(obj); ok {
		return interp.valueMethod(c, v, expr.Name, args)
	}
	if mod, ok := interp.Modules[obj]; ok {
		if f, ok := mod[expr.Name]; ok {
			return f(c, args)
		}
		names := make([]string, 0, len(mod))
		for n := range mod {
			names = append(names, n)
		}
		sort.Strings(names)
		panic(c.Fail(0, "o módulo '%s' não tem a função '%s'.%s Disponíveis: %s", obj, expr.Name, suggest(expr.Name, names), strings.Join(names, ", ")))
	}
	if interp.isModel(obj) {
		panic(c.Fail(0, "o modelo '%s' não tem a operação '%s'. Use buscar, encontrar, filtrar, paginar, contar, existe, criar, atualizar, deletar ou apagar_onde", obj, expr.Name))
	}
	var candidates []string
	for n := range interp.Modules {
		candidates = append(candidates, n)
	}
	panic(c.Fail(0, "'%s' não é variável, modelo nem módulo.%s", obj, suggest(obj, candidates)))
}

func (interp *Interpreter) callNamed(c *Call, name, canon string, args []any) any {
	if fn, ok := interp.Functions[name]; ok {
		return interp.callFunctionIn(fn, args, c.Scope, c.Pos)
	}
	if f, ok := interp.globalFuncs()[name]; ok {
		return f(c, args)
	}
	if r, ok := interp.callBuiltin(name, args); ok {
		return r
	}
	if canon != "" {
		if fn, ok := interp.Functions[canon]; ok {
			return interp.callFunctionIn(fn, args, c.Scope, c.Pos)
		}
		if r, ok := interp.callBuiltin(canon, args); ok {
			return r
		}
	}
	panic(c.Fail(0, "função '%s' não existe.%s", name, suggest(name, interp.knownFunctionNames())))
}

// valueMethod applies builtin op to v: lista.tamanho() == tamanho(lista).
func (interp *Interpreter) valueMethod(c *Call, v any, name string, args []any) any {
	full := append([]any{v}, args...)
	if r, ok := interp.callBuiltin(name, full); ok {
		return r
	}
	panic(c.Fail(0, "um valor do tipo %s não tem a operação '%s'.%s", typeName(v), name, suggest(name, builtinNames)))
}

// member reads v.campo.
func (interp *Interpreter) member(pos diagnostics.Position, v any, field string) any {
	switch x := v.(type) {
	case map[string]any:
		return x[field]
	case nil:
		panic(interp.errAt(pos, 0, "não é possível ler '%s' de nulo", field))
	}
	panic(interp.errAt(pos, 0, "um valor do tipo %s não tem o campo '%s'", typeName(v), field))
}

func (interp *Interpreter) index(pos diagnostics.Position, coll, idx any) any {
	switch c := coll.(type) {
	case []any:
		i, ok := tryNumber(idx)
		if !ok {
			panic(interp.errAt(pos, 0, "índice de lista deve ser número, recebido %s", typeName(idx)))
		}
		n := int(i)
		if n < 0 {
			n += len(c)
		}
		if n < 0 || n >= len(c) {
			panic(interp.errAt(pos, 0, "índice %d fora da lista de tamanho %d", int(i), len(c)))
		}
		return c[n]
	case map[string]any:
		return c[toString(idx)]
	case string:
		r := []rune(c)
		n := int(toNumber(idx))
		if n < 0 || n >= len(r) {
			panic(interp.errAt(pos, 0, "índice %d fora do texto de tamanho %d", n, len(r)))
		}
		return string(r[n])
	case nil:
		panic(interp.errAt(pos, 0, "não é possível indexar nulo"))
	}
	panic(interp.errAt(pos, 0, "um valor do tipo %s não pode ser indexado", typeName(coll)))
}

// assignTo implements a.b = v and lista[i] = v.
func (interp *Interpreter) assignTo(target *ast.Expression, val any, scope *Scope) {
	switch target.Type {
	case "member":
		container := interp.EvalExpr(target.Target, scope)
		m, ok := container.(map[string]any)
		if !ok {
			panic(interp.errAt(target.Pos, 0, "só é possível atribuir '%s' em um mapa, recebido %s", target.Field, typeName(container)))
		}
		m[target.Field] = val
	case "index":
		container := interp.EvalExpr(target.Left, scope)
		key := interp.EvalExpr(target.Index, scope)
		switch c := container.(type) {
		case map[string]any:
			c[toString(key)] = val
		case []any:
			n := int(toNumber(key))
			if n < 0 || n >= len(c) {
				panic(interp.errAt(target.Pos, 0, "índice %d fora da lista de tamanho %d", n, len(c)))
			}
			c[n] = val
		default:
			panic(interp.errAt(target.Pos, 0, "não é possível atribuir por índice em %s", typeName(container)))
		}
	default:
		panic(interp.errAt(target.Pos, 0, "alvo de atribuição inválido"))
	}
}

// dbCall implements modelo.operação(...). Failures raise errors with HTTP
// meaning (400 validação, 404 inexistente, 409 conflito) instead of nulo.
func (interp *Interpreter) dbCall(c *Call, model, method string, args []any) any {
	db := interp.dbOf(c.Ctx())
	m := interp.modelAST(model)
	clean := func(row map[string]any) map[string]any { return stripSecrets(m, row) }
	fail := func(err error) {
		var ec *banco.ErrCampo
		switch {
		case errors.Is(err, banco.ErrNaoEncontrado):
			panic(c.Fail(404, "%s: registro não encontrado", c.Name))
		case errors.Is(err, banco.ErrConflito):
			panic(c.Fail(409, "%s: %s", c.Name, err.Error()))
		case errors.As(err, &ec):
			panic(c.Fail(0, "%s: %s", c.Name, err.Error()))
		default:
			panic(c.Fail(400, "%s: %s", c.Name, err.Error()))
		}
	}
	consulta := func(filtros, opcoes map[string]any) banco.Consulta {
		q := banco.Consulta{Filtros: filtros}
		for k, v := range opcoes {
			switch k {
			case "ordenar":
				q.Ordenar = toString(v)
			case "limite":
				q.Limite = int(toNumber(v))
			case "pagina":
				q.Pagina = int(toNumber(v))
			case "busca":
				q.Busca = toString(v)
			case "campos_busca":
				if l, ok := v.([]any); ok {
					for _, f := range l {
						q.BuscaCampos = append(q.BuscaCampos, toString(f))
					}
				}
			case "ou":
				// groups of filters, at least one must hold (lists narrowed by
				// what a person may reach)
				if gs, ok := v.([]map[string]any); ok {
					q.Ou = gs
				}
			default:
				panic(c.Fail(0, "%s: opção desconhecida '%s' (use ordenar, limite, pagina, busca, campos_busca)", c.Name, k))
			}
		}
		if q.Limite > 1000 {
			q.Limite = 1000
		}
		return q
	}
	rowsToList := func(rows []map[string]any) []any {
		out := make([]any, len(rows))
		for i, r := range rows {
			out[i] = clean(r)
		}
		return out
	}
	filtros := func(i int, required bool) map[string]any {
		var f map[string]any
		if required {
			f = c.Map(args, i, "filtros")
		} else {
			f = c.OptMap(args, i, "filtros")
		}
		checkSecretFilters(c, m, f)
		return f
	}
	switch method {
	case "buscar":
		id, ok := tryNumber(c.Arg(args, 0, "id"))
		if !ok {
			return nil
		}
		row, err := db.BuscarRegistro(model, int64(id))
		if errors.Is(err, banco.ErrNaoEncontrado) {
			return nil
		}
		if err != nil {
			fail(err)
		}
		return clean(row)
	case "encontrar":
		q := consulta(filtros(0, true), c.OptMap(args, 1, "opcoes"))
		q.Limite = 1
		rows, _, err := db.Filtrar(model, q)
		if err != nil {
			fail(err)
		}
		if len(rows) == 0 {
			return nil
		}
		return clean(rows[0])
	case "filtrar":
		rows, _, err := db.Filtrar(model, consulta(filtros(0, false), c.OptMap(args, 1, "opcoes")))
		if err != nil {
			fail(err)
		}
		return rowsToList(rows)
	case "paginar":
		q := consulta(filtros(0, false), c.OptMap(args, 1, "opcoes"))
		if q.Limite <= 0 {
			q.Limite = 20
		}
		if q.Pagina <= 0 {
			q.Pagina = 1
		}
		rows, total, err := db.Filtrar(model, q)
		if err != nil {
			fail(err)
		}
		return map[string]any{"itens": rowsToList(rows), "total": float64(total), "pagina": float64(q.Pagina), "limite": float64(q.Limite)}
	case "contar", "count":
		n, err := db.ContarFiltro(model, consulta(filtros(0, false), nil))
		if err != nil {
			fail(err)
		}
		return float64(n)
	case "existe":
		n, err := db.ContarFiltro(model, consulta(filtros(0, true), nil))
		if err != nil {
			fail(err)
		}
		return n > 0
	case "criar", "create":
		data, reveal := interp.prepareWrite(c, m, c.Map(args, 0, "dados"), true, 0)
		row, err := db.CriarMapa(model, data)
		if err != nil {
			fail(err)
		}
		row = clean(row)
		for k, v := range reveal {
			row[k] = v // shown once, only in the creation result
		}
		return row
	case "atualizar", "update":
		id := c.Num(args, 0, "id")
		before, _ := db.BuscarRegistro(model, int64(id))
		data, _ := interp.prepareWrite(c, m, c.Map(args, 1, "dados"), false, int64(id))
		row, err := db.AtualizarMapa(model, int64(id), data)
		if err != nil {
			fail(err)
		}
		interp.followAddress(db, m, int64(id), before, row)
		return clean(row)
	case "verificar_senha":
		// modelo.verificar_senha(registro_ou_id, senha[, campo]) — tempo constante
		var id float64
		switch r := c.Arg(args, 0, "registro").(type) {
		case map[string]any:
			id = toNumber(r["id"])
		case nil:
			id = -1
		default:
			id = toNumber(r)
		}
		senha := toString(c.Arg(args, 1, "senha"))
		campo := ""
		if len(args) > 2 {
			campo = c.Str(args, 2, "campo")
		} else {
			for _, f := range m.Fields {
				if f.Type == ast.FieldSenha || (f.Protected && f.Type != ast.FieldSegredo) {
					campo = strings.ToLower(f.Name)
					break
				}
			}
		}
		if campo == "" {
			panic(c.Fail(0, "%s: o modelo não tem campo senha", c.Name))
		}
		hash := ""
		if row, _ := db.BuscarRegistro(model, int64(id)); row != nil {
			hash, _ = row[campo].(string)
		}
		if hash == "" {
			dummyOnce.Do(func() { dummyHash, _ = bcrypt.GenerateFromPassword([]byte("germanio"), bcryptCost) })
			bcrypt.CompareHashAndPassword(dummyHash, []byte(senha))
			return false
		}
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(senha)) == nil
	case "por_segredo":
		// modelo.por_segredo(valor[, campo]) → registro ativo (não expirado nem revogado) ou nulo
		valor, _ := c.Arg(args, 0, "segredo").(string)
		campo := ""
		if len(args) > 1 {
			campo = c.Str(args, 1, "campo")
		} else {
			for _, f := range m.Fields {
				if f.Type == ast.FieldSegredo {
					campo = strings.ToLower(f.Name)
					break
				}
			}
		}
		if campo == "" {
			panic(c.Fail(0, "%s: o modelo não tem campo segredo", c.Name))
		}
		if valor == "" {
			return nil
		}
		rows, _, err := db.Filtrar(model, banco.Consulta{Filtros: map[string]any{campo: digest(valor)}, Limite: 1})
		if err != nil {
			fail(err)
		}
		if len(rows) == 0 || !secretActive(m, rows[0]) {
			return nil
		}
		return clean(rows[0])
	case "revogar":
		if !m.Revocable {
			panic(c.Fail(0, "%s: o modelo não é revogavel", c.Name))
		}
		row, err := db.AtualizarMapa(model, int64(c.Num(args, 0, "id")), map[string]any{"revogado": true})
		if err != nil {
			fail(err)
		}
		return clean(row)
	case "deletar", "delete":
		id := c.Num(args, 0, "id")
		n, err := db.DeletarFiltro(model, banco.Consulta{Filtros: map[string]any{"id": id}})
		if err != nil {
			fail(err)
		}
		return n > 0
	case "apagar_onde":
		n, err := db.DeletarFiltro(model, consulta(filtros(0, true), nil))
		if err != nil {
			fail(err)
		}
		return float64(n)
	case "listar", "todos", "list", "all":
		// Legacy: newest 100 rows, same as before filters existed.
		rows, _, err := db.Listar(model, nil)
		if err != nil {
			fail(err)
		}
		return rowsToList(rows)
	}
	panic(c.Fail(0, "operação de banco desconhecida: %s", method))
}

// globalFuncs are builtins that need the execution context.
func (interp *Interpreter) globalFuncs() map[string]ModuleFunc {
	return map[string]ModuleFunc{
		// responder(status, corpo[, cabecalhos]) ends the route with a response.
		"responder": func(c *Call, args []any) any {
			r := &Response{Status: int(c.Num(args, 0, "status"))}
			if len(args) > 1 {
				r.Body = args[1]
			}
			if len(args) > 2 {
				r.Headers = map[string]string{}
				for k, v := range c.Map(args, 2, "cabecalhos") {
					r.Headers[k] = toString(v)
				}
			}
			panic(signal{Type: signalRespond, Value: r})
		},
		// redirecionar("/caminho") answers 303; only local paths are allowed.
		"redirecionar": func(c *Call, args []any) any {
			to := c.Str(args, 0, "caminho")
			if !strings.HasPrefix(to, "/") || strings.HasPrefix(to, "//") || strings.ContainsAny(to, "\\\r\n") {
				panic(c.Fail(0, "redirecionar aceita apenas caminhos locais como \"/projetos\", recebido %q", to))
			}
			panic(signal{Type: signalRespond, Value: &Response{Status: http.StatusSeeOther, Redirect: to}})
		},
		// falhar(status, mensagem) raises an error that tentar/erro can catch.
		"falhar": func(c *Call, args []any) any {
			status := int(c.Num(args, 0, "status"))
			e := &RuntimeError{Status: status, Pos: c.Pos}
			if len(args) > 1 {
				e.Message = toString(args[1])
				if _, isText := args[1].(string); !isText {
					e.Payload = args[1]
				}
			}
			panic(e)
		},
		// recuse("mensagem") refuses the current action with a message for people.
		"recuse": func(c *Call, args []any) any {
			panic(&RuntimeError{Status: 400, Message: toString(c.Arg(args, 0, "mensagem")), Pos: c.Pos})
		},
		// obter(mapa, chave, padrao) reads an optional key.
		"obter": func(c *Call, args []any) any {
			v := c.Arg(args, 0, "mapa")
			if v == nil {
				if len(args) > 2 {
					return args[2]
				}
				return nil
			}
			m, ok := v.(map[string]any)
			if !ok {
				panic(c.Fail(0, "obter: o primeiro argumento deve ser um mapa, recebido %s", typeName(v)))
			}
			if val, ok := m[c.Str(args, 1, "chave")]; ok && val != nil {
				return val
			}
			if len(args) > 2 {
				return args[2]
			}
			return nil
		},
		// tem(mapa, chave) reports whether the key exists.
		"tem": func(c *Call, args []any) any {
			m, ok := c.Arg(args, 0, "mapa").(map[string]any)
			if !ok {
				return false
			}
			_, has := m[c.Str(args, 1, "chave")]
			return has
		},
	}
}

// ExecRoute runs a route handler in an isolated scope. vars are bound as
// local variables (requisicao, ...). The returned Response is nil when the
// handler neither returned nor answered.
func (interp *Interpreter) ExecRoute(stmts []*ast.Statement, ctx *Context, vars map[string]any) (resp *Response, output []string, err error) {
	if ctx == nil {
		ctx = &Context{}
	}
	scope := NewScope(interp.Global)
	scope.ctx = ctx
	for k, v := range vars {
		scope.SetLocal(k, v)
	}
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		switch x := r.(type) {
		case signal:
			switch x.Type {
			case signalReturn:
				resp = &Response{Status: http.StatusOK, Body: x.Value}
				return
			case signalRespond:
				resp = x.Value.(*Response)
				return
			}
			err = &RuntimeError{Message: "pare/continue fora de laço"}
		case *RuntimeError:
			output = ctx.Output
			err = x
		case error:
			err = &RuntimeError{Message: x.Error()}
		default:
			err = &RuntimeError{Message: fmt.Sprint(x)}
		}
	}()
	interp.ExecStatements(stmts, scope)
	return nil, ctx.Output, nil
}

// RunFunction calls a user function by name from Go (tasks, events).
func (interp *Interpreter) RunFunction(name string, args []any, ctx *Context) (result any, err error) {
	fn, ok := interp.Functions[name]
	if !ok {
		return nil, &RuntimeError{Message: fmt.Sprintf("função '%s' não existe.%s", name, suggest(name, interp.knownFunctionNames()))}
	}
	scope := NewScope(interp.Global)
	scope.ctx = ctx
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case *RuntimeError:
				err = x
			case signal:
				err = &RuntimeError{Message: "sinal de controle inesperado em " + name}
			case error:
				err = &RuntimeError{Message: x.Error()}
			default:
				err = &RuntimeError{Message: fmt.Sprint(x)}
			}
		}
	}()
	return interp.callFunctionIn(fn, args, scope, diagnostics.Position{}), nil
}
