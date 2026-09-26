package semantic

import "strings"

// Inferred unannotated types are monomorphic. Rigid named parameters are
// instantiated independently at each call to a generic function.
type Type struct {
	Kind       string
	Name       string // A rigid parameter in a generic function declaration.
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
	if t.Kind == "param" {
		return t.Name
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
	return parseTypeParams(s, nil)
}
func parseTypeParams(s string, params map[string]*Type) *Type {
	if s == "" {
		return fresh()
	}
	if strings.HasSuffix(s, "?") {
		return &Type{Kind: "optional", Elem: parseTypeParams(s[:len(s)-1], params)}
	}
	if strings.HasPrefix(s, "[") {
		return &Type{Kind: "list", Elem: parseTypeParams(s[1:len(s)-1], params)}
	}
	if t, ok := params[s]; ok {
		return t
	}
	return typ(s)
}
func numeric(t *Type) bool {
	t = resolve(t)
	return t.Kind == "inteiro" || t.Kind == "decimal" || t.Kind == "param" && t.Constraint == "numeric"
}
func unresolvedOptional(t *Type) bool {
	t = resolve(t)
	if t.Kind == "optional" && resolve(t.Elem).Kind == "" {
		return true
	}
	return t.Elem != nil && unresolvedOptional(t.Elem)
}
func permits(c string, t *Type) bool {
	t = resolve(t)
	if c == "" || t.Kind == "" {
		return true
	}
	if t.Kind == "param" {
		return t.Constraint == "numeric" && (c == "numeric" || c == "addable")
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
	if a.Kind == "param" {
		return a.Name == b.Name
	}
	if a.Elem != nil {
		return unify(a.Elem, b.Elem)
	}
	return true
}

// Each call has fresh inference variables. Rigid parameters remain rigid while
// checking the generic body, so operations requiring concrete types are rejected.
func instantiate(t *Type, params map[string]*Type) *Type {
	t = resolve(t)
	if t.Kind == "param" {
		return params[t.Name]
	}
	if t.Elem != nil {
		return &Type{Kind: t.Kind, Elem: instantiate(t.Elem, params)}
	}
	return t
}
func concreteGeneric(t *Type) bool {
	t = resolve(t)
	if t.Kind == "" || t.Kind == "nulo" {
		return false
	}
	return t.Elem == nil || concreteGeneric(t.Elem)
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
	if g.Kind == "nulo" {
		return false
	}
	if w.Kind == "list" && g.Kind == "list" {
		return assignable(w.Elem, g.Elem)
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
