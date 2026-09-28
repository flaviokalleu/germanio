package ast

import "strings"

// OutwardEffect tells whether the built-in name acts outside the program
// (it sends something) when called with this HTTP method ("" when the
// built-in takes none), and how to name it. Reads — chamar with GET, the AI
// and parallel GET helpers — only bring information in and are not effects.
// Inside a change, effects run after the commit (G86; runtime/interpreter).
func OutwardEffect(name, method string) (string, bool) {
	switch name {
	case "chamar", "call", "http":
		switch m := strings.ToUpper(method); m {
		case "", "GET", "HEAD", "OPTIONS":
			return "", false
		default:
			return "chamar " + m, true
		}
	case "telegram_enviar", "telegram_send", "telegram",
		"discord_enviar", "discord_send", "discord",
		"slack_enviar", "slack_send", "slack",
		"sms_enviar", "sms_send", "sms",
		"webhook_enviar", "webhook_send", "webhook",
		"mercadopago_link", "mercadopago", "mp":
		return name, true
	}
	return "", false
}

// EffectsUsed returns the outward calls of stmts whose result is used
// (assigned, compared, passed on). A call on its own line is fine: it runs
// after the commit. The method of chamar must be a literal to be known here;
// otherwise the runtime decides.
func EffectsUsed(stmts []*Statement) []*Expression {
	var out []*Expression
	walkEffects(stmts, func(e *Expression, top bool) {
		if !top {
			out = append(out, e)
		}
	})
	return out
}

// Effects returns the outward calls of stmts in the order they are written:
// the order in which they run after the commit (when they are reached).
func Effects(stmts []*Statement) []*Expression {
	var out []*Expression
	walkEffects(stmts, func(e *Expression, _ bool) { out = append(out, e) })
	return out
}

// EffectKind names an outward call expression ("chamar POST").
func EffectKind(e *Expression) string {
	method := ""
	if len(e.Args) >= 2 && e.Args[1] != nil && e.Args[1].Type == "literal" {
		method, _ = e.Args[1].Value.(string)
	}
	name := e.Name
	if e.Canon != "" {
		name = e.Canon
	}
	kind, _ := OutwardEffect(name, method)
	return kind
}

func walkEffects(stmts []*Statement, found func(e *Expression, top bool)) {
	var expr func(e *Expression, top bool)
	expr = func(e *Expression, top bool) {
		if e == nil {
			return
		}
		if e.Type == "call" && e.Object == "" && EffectKind(e) != "" {
			found(e, top)
		}
		expr(e.Left, false)
		expr(e.Right, false)
		expr(e.Index, false)
		expr(e.Target, false)
		for _, a := range e.Args {
			expr(a, false)
		}
		for _, a := range e.Elements {
			expr(a, false)
		}
	}
	var walk func([]*Statement)
	walk = func(list []*Statement) {
		for _, s := range list {
			if s == nil {
				continue
			}
			switch s.Type {
			case "expr":
				expr(s.Expr, true)
			case "call":
				if s.Call != nil {
					expr(&Expression{Type: "call", Name: s.Call.Name, Object: s.Call.Object, Args: s.Call.Args, Pos: s.Pos}, true)
				}
			case "var":
				expr(&s.VarDecl.Value, false)
			case "assign":
				expr(&s.Assign.Value, false)
			case "return":
				expr(s.Return, false)
			case "print":
				expr(s.Print, false)
			case "if":
				expr(&s.If.Condition, false)
				walk(s.If.Body)
				for _, ei := range s.If.ElseIfs {
					expr(&ei.Condition, false)
					walk(ei.Body)
				}
				walk(s.If.Else)
			case "for_each":
				expr(&s.ForEach.Collection, false)
				walk(s.ForEach.Body)
			case "while":
				expr(&s.While.Condition, false)
				walk(s.While.Body)
			case "repeat":
				expr(&s.Repeat.Count, false)
				walk(s.Repeat.Body)
			case "try":
				walk(s.Try.Body)
				walk(s.Try.Catch)
			case "when":
				expr(&s.When.Target, false)
				for _, c := range s.When.Cases {
					walk(c.Body)
				}
			case "concurrency":
				walk(s.Concurrency.Tasks)
			}
		}
	}
	walk(stmts)
}
