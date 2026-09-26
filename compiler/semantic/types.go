package semantic

import "strings"

// Type variables are unified once per module: Phase 1 inference is monomorphic.
// They are independent of Go types and can feed a future IR/backend.
type Type struct {
	Kind       string
	Elem, Link *Type
	Constraint string
}

func fresh() *Type       { return &Type{} }
func typ(k string) *Type { return &Type{Kind: k} }
func resolve(t *Type) *Type {
	if t.Link != nil {
		t.Link = resolve(t.Link)
		return t.Link
	}
	return t
}
func (t *Type) String() string {
	t = resolve(t)
	if t.Kind == "" {
		return "tipo inferido"
	}
	if t.Kind == "list" {
		return "[" + t.Elem.String() + "]"
	}
	if t.Kind == "optional" {
		return t.Elem.String() + "?"
	}
	return t.Kind
}
func parseType(s string) *Type {
	if s == "" {
		return fresh()
	}
	if strings.HasSuffix(s, "?") {
		return &Type{Kind: "optional", Elem: parseType(s[:len(s)-1])}
	}
	if strings.HasPrefix(s, "[") {
		return &Type{Kind: "list", Elem: parseType(s[1 : len(s)-1])}
	}
	return typ(s)
}
func numeric(t *Type) bool { t = resolve(t); return t.Kind == "inteiro" || t.Kind == "decimal" }
func permits(c string, t *Type) bool {
	t = resolve(t)
	if c == "" || t.Kind == "" {
		return true
	}
	if numeric(t) {
		return true
	}
	return c == "addable" && t.Kind == "texto"
}
func occurs(a, b *Type) bool { b = resolve(b); return a == b || b.Elem != nil && occurs(a, b.Elem) }
func unify(a, b *Type) bool {
	a, b = resolve(a), resolve(b)
	if a == b {
		return true
	}
	if a.Kind == "" {
		if occurs(a, b) || !permits(a.Constraint, b) {
			return false
		}
		if b.Kind == "" && (b.Constraint == "" || a.Constraint == "numeric") {
			b.Constraint = a.Constraint
		}
		a.Link = b
		return true
	}
	if b.Kind == "" {
		return unify(b, a)
	}
	if a.Kind != b.Kind {
		return false
	}
	if a.Elem != nil {
		return unify(a.Elem, b.Elem)
	}
	return true
}
func assignable(want, got *Type) bool {
	w, g := resolve(want), resolve(got)
	if w.Kind == "optional" {
		if g.Kind == "nulo" {
			return true
		}
		if g.Kind == "optional" {
			return unify(w.Elem, g.Elem)
		}
		return unify(w.Elem, g)
	}
	return unify(w, g)
}
func constrain(t *Type, c string) bool {
	t = resolve(t)
	if t.Kind == "" {
		if t.Constraint == "" || c == "numeric" {
			t.Constraint = c
		}
		return true
	}
	return permits(c, t)
}
