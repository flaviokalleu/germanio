package interpreter

import (
	"fmt"
	"sync"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/diagnostics"
)

// External effects (G86): work that acts outside the program — an HTTP
// request that changes something, a message, a webhook. Inside a change (a
// request transaction) an effect never runs while the transaction, and so the
// database's write lock, is held: it is recorded and runs after the commit.
// A change that is undone takes its effects with it; an effect that fails
// after the commit does not undo the change (it is logged).
//
// Effect is the unit a durable outbox would store: Kind and Args describe the
// work; Run performs it. The dispatch happens in one place (the server's
// transaction), so moving effects to the persistent task queue, with
// retries, changes that place only.

// Effect is one external action recorded during a change.
type Effect struct {
	Kind string // the built-in, e.g. "chamar POST", "telegram_enviar"
	Args []any
	Pos  diagnostics.Position
	Run  func() error
}

// Describe is how logs and ge explain name an effect.
func (e Effect) Describe() string {
	if e.Pos.Line > 0 {
		return fmt.Sprintf("%s (%s:%d)", e.Kind, e.Pos.File, e.Pos.Line)
	}
	return e.Kind
}

// Effects collects the effects of one change, in the order they were asked.
// Tasks started by paralelo share it, so it is safe for concurrent use.
type Effects struct {
	mu   sync.Mutex
	list []Effect
}

// Add records an effect.
func (q *Effects) Add(e Effect) {
	q.mu.Lock()
	q.list = append(q.list, e)
	q.mu.Unlock()
}

// Take returns the recorded effects and empties the list.
func (q *Effects) Take() []Effect {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := q.list
	q.list = nil
	return out
}

// outward tells whether the built-in name, called with args, acts outside
// the program (ast.OutwardEffect is the single list).
func outward(name string, args []any) (string, bool) {
	method := ""
	if len(args) >= 2 {
		method = toString(args[1])
	}
	if (name == "chamar" || name == "call" || name == "http") && method == "" {
		method = "GET"
	}
	return ast.OutwardEffect(name, method)
}

// deferEffect handles an outward built-in called inside a change: it records
// the effect to run after the commit. ok=false: not an effect, or not inside
// a change — the built-in runs now, as always.
func (interp *Interpreter) deferEffect(c *Call, name string, args []any) (any, bool) {
	ctx := c.Ctx()
	if ctx == nil || ctx.Effects == nil {
		return nil, false
	}
	kind, isEffect := outward(name, args)
	if !isEffect {
		return nil, false
	}
	if !c.Unused {
		panic(c.Fail(0, "%s age fora do sistema e é chamado durante uma mudança usando a resposta. Um efeito externo só acontece depois que a mudança é salva, então a resposta ainda não existe aqui. Chame %s sem usar o resultado (ele acontece logo depois de salvar), ou use tarefas.enfileirar(\"funcao\", dados) para trabalhar com a resposta fora da mudança", kind, name))
	}
	saved := append([]any(nil), args...)
	ctx.Effects.Add(Effect{Kind: kind, Args: saved, Pos: c.Pos, Run: func() error {
		r, _ := interp.callBuiltin(name, saved)
		if r == nil || r == false {
			return fmt.Errorf("%s não foi concluído (veja o log)", kind)
		}
		return nil
	}})
	return nil, true
}
