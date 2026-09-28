package interpreter

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/diagnostics"
)

// Runtime of the intent layer: who can do what (derived from `pode`,
// `permita`, papéis, membros, visibilidade) and operations callable from Go
// that run through the same schema rules and hooks as .ge code.

// Op performs a database operation as .ge code would, returning a
// RuntimeError instead of panicking.
func (interp *Interpreter) Op(ctx *Context, model, method string, args ...any) (res any, err error) {
	scope := NewScope(interp.Global)
	if ctx == nil {
		ctx = &Context{}
	}
	scope.ctx = ctx
	defer func() {
		if r := recover(); r != nil {
			err = asRuntimeError(r)
		}
	}()
	c := &Call{Interp: interp, Scope: scope, Name: model + "." + method}
	return interp.dbCall(c, model, method, args), nil
}

func asRuntimeError(r any) *RuntimeError {
	switch x := r.(type) {
	case *RuntimeError:
		return x
	case signal:
		if x.Type == signalRespond {
			return &RuntimeError{Status: 0, Message: "responder dentro de operação", Payload: x.Value}
		}
		return &RuntimeError{Message: "sinal de controle fora de lugar"}
	case error:
		return &RuntimeError{Message: x.Error()}
	}
	return &RuntimeError{Message: fmt.Sprint(r)}
}

// RunHook executes a `quando` block with the given bindings. Its value is
// what the block returned with retornar (nil otherwise).
func (interp *Interpreter) RunHook(ctx *Context, h *ast.Hook, vars map[string]any) (result any, resp *Response, err error) {
	scope := NewScope(interp.Global)
	scope.ctx = ctx
	for k, v := range vars {
		scope.SetLocal(k, v)
	}
	defer func() {
		if r := recover(); r != nil {
			if sig, ok := r.(signal); ok {
				switch sig.Type {
				case signalReturn:
					result = sig.Value
					return
				case signalRespond:
					resp = sig.Value.(*Response)
					return
				}
			}
			err = asRuntimeError(r)
		}
	}()
	interp.ExecStatements(h.Body, scope)
	return nil, nil, nil
}

// IsAdmin: the person has admin verdadeiro or papel "administrador".
func (interp *Interpreter) IsAdmin(atual map[string]any) bool {
	if atual == nil {
		return false
	}
	if b, ok := atual["admin"].(bool); ok && b {
		return true
	}
	return toString(atual["papel"]) == "administrador"
}

const adminLevel = 1 << 20

type levelKey struct {
	model string
	id    int64
}

// Level returns the access level of atual on a record: its own
// membership, the membership inherited from parents (herda membros,
// subgrupos) or, for records without members, the level on the parent
// that has them.
func (interp *Interpreter) Level(ctx *Context, atual map[string]any, e *ast.Entity, record map[string]any) int {
	if atual == nil || record == nil || interp.App == nil {
		return 0
	}
	if interp.IsAdmin(atual) {
		return adminLevel
	}
	var memo map[levelKey]int
	if ctx != nil {
		if ctx.Values == nil {
			ctx.Values = map[string]any{}
		}
		if m, ok := ctx.Values["__niveis"].(map[levelKey]int); ok {
			memo = m
		} else {
			memo = map[levelKey]int{}
			ctx.Values["__niveis"] = memo
		}
	}
	return interp.level(ctx, memo, atual, e, record, 0)
}

func (interp *Interpreter) level(ctx *Context, memo map[levelKey]int, atual map[string]any, e *ast.Entity, record map[string]any, depth int) int {
	if depth > 32 || record == nil {
		return 0
	}
	id := int64(toNumber(record["id"]))
	key := levelKey{e.Singular, id}
	if id > 0 && memo != nil {
		if v, ok := memo[key]; ok {
			return v
		}
	}
	app := interp.App
	lv := 0
	if e.Singular == app.MemberModel {
		if target, ok := app.Entities[toString(record["recurso"])]; ok {
			lv = interp.level(ctx, memo, atual, target, interp.load(ctx, target, record["recurso_id"]), depth+1)
		}
	} else {
		if e.HasMembers && id > 0 {
			m, _ := interp.Op(ctx, app.MemberModel, "encontrar", map[string]any{"recurso": e.Singular, "recurso_id": id, "pessoa_id": atual["id"]})
			if row, ok := m.(map[string]any); ok {
				lv = app.Level(toString(row["papel"]))
			}
		}
		if e.HierarchyField != "" && record[e.HierarchyField] != nil {
			lv = max(lv, interp.level(ctx, memo, atual, e, interp.load(ctx, e, record[e.HierarchyField]), depth+1))
		}
		vias := []string{}
		if e.InheritVia != "" {
			vias = append(vias, e.InheritVia)
		} else if !e.HasMembers {
			// Records without members take the highest level among the
			// parents they have that (transitively) have members — every
			// filled one, never "the first" of a map.
			for field, target := range e.Parents {
				if target != app.LoginEntity && interp.hasMembersChain(app.Entities[target], 0) {
					vias = append(vias, field)
				}
			}
		}
		for _, via := range vias {
			if record[via] != nil {
				parent := app.Entities[e.Parents[via]]
				lv = max(lv, interp.level(ctx, memo, atual, parent, interp.load(ctx, parent, record[via]), depth+1))
			}
		}
	}
	if id > 0 && memo != nil {
		memo[key] = lv
	}
	return lv
}

func (interp *Interpreter) hasMembersChain(e *ast.Entity, depth int) bool {
	if e == nil || depth > 16 {
		return false
	}
	if e.HasMembers {
		return true
	}
	for _, t := range e.Parents {
		if interp.hasMembersChain(interp.App.Entities[t], depth+1) {
			return true
		}
	}
	return false
}

func (interp *Interpreter) load(ctx *Context, e *ast.Entity, id any) map[string]any {
	if e == nil || id == nil {
		return nil
	}
	key := levelKey{e.Singular, int64(toNumber(id))}
	cache := readCache(ctx)
	if cache != nil {
		if row, ok := cache[key]; ok {
			return row
		}
	}
	row, err := interp.dbOf(ctx).BuscarRegistro(e.Singular, key.id)
	if err != nil {
		return nil
	}
	row = stripSecrets(e.Model, row)
	if cache != nil {
		cache[key] = row
	}
	return row
}

// readCacheKey marks a read-only scan (a list): while it runs, a record
// loaded to decide visibility (the project of each issue) is read once, not
// once per row (the N+1 that made one page of a big list take seconds).
const readCacheKey = "__linhas_lidas"

// BeginReadCache starts a read-only scan on ctx; EndReadCache ends it.
func BeginReadCache(ctx *Context) {
	if ctx == nil {
		return
	}
	if ctx.Values == nil {
		ctx.Values = map[string]any{}
	}
	ctx.Values[readCacheKey] = map[levelKey]map[string]any{}
}

func EndReadCache(ctx *Context) {
	if ctx != nil && ctx.Values != nil {
		delete(ctx.Values, readCacheKey)
	}
}

func readCache(ctx *Context) map[levelKey]map[string]any {
	if ctx == nil || ctx.Values == nil {
		return nil
	}
	m, _ := ctx.Values[readCacheKey].(map[levelKey]map[string]any)
	return m
}

// MayReach: atual has a role in record (directly or through its parents) or
// the record is visible to atual (public, internal). A record that belongs to
// something atual cannot reach is never visible through it; lists use this to
// narrow what they read (a superset: Can still decides each row).
func (interp *Interpreter) MayReach(ctx *Context, atual map[string]any, e *ast.Entity, record map[string]any) bool {
	return interp.Level(ctx, atual, e, record) > 0 || interp.visible(ctx, atual, e, record)
}

// HasMembersChain: e or one of its ancestors has members.
func (interp *Interpreter) HasMembersChain(e *ast.Entity) bool {
	return interp.hasMembersChain(e, 0)
}

// Owns: the record is the person, or belongs to the person.
func (interp *Interpreter) Owns(atual map[string]any, e *ast.Entity, record map[string]any) bool {
	if atual == nil || record == nil || interp.App == nil {
		return false
	}
	app := interp.App
	uid := toNumber(atual["id"])
	switch {
	case e.Singular == app.LoginEntity:
		return toNumber(record["id"]) == uid
	case e.Singular == app.MemberModel:
		return toNumber(record["pessoa_id"]) == uid
	}
	for _, f := range e.OwnerFields {
		if v, ok := record[f]; ok && v != nil && toNumber(v) == uid {
			return true
		}
	}
	return false
}

// visible applies the visibilidade field: public → anyone, internal →
// signed in; private falls through to the rules.
func (interp *Interpreter) visible(ctx *Context, atual map[string]any, e *ast.Entity, record map[string]any) bool {
	return interp.visibleDepth(ctx, atual, e, record, 0)
}

// visibleDepth: records without their own visibility take it from their
// parent — what belongs to something public is public (issues of a public
// project). Confidentiality is expressed with `antes de ver`.
func (interp *Interpreter) visibleDepth(ctx *Context, atual map[string]any, e *ast.Entity, record map[string]any, depth int) bool {
	if record == nil || depth > 8 {
		return false
	}
	if e.Visibility == "" {
		if e.Singular == interp.App.MemberModel {
			return false
		}
		// every filled parent must be visible (a record of two parents is
		// never more visible than the stricter one)
		seen := false
		for field, target := range e.Parents {
			pe := interp.App.Entities[target]
			if target == interp.App.LoginEntity || record[field] == nil || (pe.Visibility == "" && len(pe.Parents) == 0) {
				continue
			}
			if !interp.visibleDepth(ctx, atual, pe, interp.load(ctx, pe, record[field]), depth+1) {
				return false
			}
			seen = true
		}
		return seen
	}
	switch toString(record[e.Visibility]) {
	case "public":
		return true
	case "internal":
		return atual != nil
	}
	return false
}

// Can reports whether atual may perform verb on record (record may be the
// data of a record being created, with its parent references).
func (interp *Interpreter) Can(ctx *Context, atual map[string]any, e *ast.Entity, verb string, record map[string]any) bool {
	if verb == "ver" && record != nil && record["id"] != nil {
		if h := e.Hooks["antes_ver"]; h != nil {
			// `antes de ver` may hide a record from this person (recuse).
			if _, _, err := interp.RunHook(ctx, h, map[string]any{"atual": nilMap(atual), "registro": record, e.Singular: record}); err != nil {
				return interp.IsAdmin(atual)
			}
		}
	}
	if interp.IsAdmin(atual) {
		return true
	}
	if vt := e.ViewThrough; vt != nil && record != nil {
		// the history (GEP 0011): seen by whoever sees what it describes;
		// nobody but an administrator changes it
		return verb == "ver" && interp.seesDescribed(ctx, atual, vt, record)
	}
	if verb == "ver" && record != nil {
		// `X confidencial pode ser vista por …`: when the flag is set, only
		// those listed see it, whatever else would allow it.
		for _, rs := range e.Restrictions {
			if b, _ := record[rs.Flag].(bool); b {
				return interp.restrictedAllows(ctx, atual, e, rs, record)
			}
		}
	}
	if verb == "ver" && InheritsView(interp.App, e) && record != nil {
		// No viewing rule of its own: whoever sees the parent sees it.
		// whoever sees every filled parent sees it
		seen := false
		for field, target := range e.Parents {
			if target == interp.App.LoginEntity || record[field] == nil {
				continue
			}
			pe := interp.App.Entities[target]
			if !interp.Can(ctx, atual, pe, "ver", interp.load(ctx, pe, record[field])) {
				return false
			}
			seen = true
		}
		if seen {
			return true
		}
	}
	if (verb == "ver" || verb == "baixar_codigo") && interp.visible(ctx, atual, e, record) {
		return true
	}
	rules := e.Rules[verb]
	if len(rules) == 0 && e.Transitions[verb] != nil {
		rules = e.Rules["editar"] // who may edit may move it between states
	}
	if verb == "ver" {
		// Whoever may change a record may also see it.
		rules = append(append(append([]*ast.AccessRule{}, rules...), e.Rules["editar"]...), e.Rules["excluir"]...)
	}
	for _, r := range rules {
		if interp.RulePasses(ctx, atual, e, r, record) {
			return true
		}
	}
	return false
}

func (interp *Interpreter) RulePasses(ctx *Context, atual map[string]any, e *ast.Entity, r *ast.AccessRule, record map[string]any) bool {
	if r.Own && !interp.Owns(atual, e, record) {
		return false
	}
	// Inside something that has members, generic rules (anyone, any signed-in
	// person) only count when that thing is public/internal; otherwise only
	// roles decide. Ownership rules keep their own meaning.
	if (r.Anyone || r.SignedIn) && !r.Own && record != nil && interp.hiddenMemberedAncestor(ctx, atual, e, record, 0) {
		return false
	}
	switch {
	case r.Anyone:
		return true
	case r.SignedIn:
		return atual != nil
	case r.MinRole == "administrador":
		return interp.IsAdmin(atual)
	case r.MinRole != "":
		return interp.Level(ctx, atual, e, record) >= interp.App.Level(r.MinRole)
	}
	return r.Own && atual != nil
}

// RecordDependent reports whether seeing records of e depends on each
// record (visibility, roles or ownership) — lists must then be filtered.
func RecordDependent(e *ast.Entity) bool {
	// restrictions (X confidencial pode ser vista por …) hide single records:
	// a list must check each one
	if e.Visibility != "" || e.ViewThrough != nil || len(e.Restrictions) > 0 || len(e.Hooks["antes_ver"].GetBody()) > 0 {
		return true
	}
	for _, v := range []string{"ver", "editar", "excluir"} {
		for _, r := range e.Rules[v] {
			if r.Own || (r.MinRole != "" && r.MinRole != "administrador") {
				return true
			}
		}
	}
	return false
}

// Friendly converts a runtime error into a message for people.
func Friendly(err error) string {
	var re *RuntimeError
	if errors.As(err, &re) {
		return re.Message
	}
	return err.Error()
}

var _ = diagnostics.Position{}
var _ = strings.ToLower

// registerAccess exposes the authorization engine to level-3 code.
func registerAccess(interp *Interpreter) {
	atualOf := func(c *Call) map[string]any {
		if v, ok := c.Scope.Get("atual"); ok {
			m, _ := v.(map[string]any)
			return m
		}
		return nil
	}
	entity := func(c *Call, name string) *ast.Entity {
		if interp.App == nil {
			panic(c.Fail(0, "acesso exige dados declarados com tenha"))
		}
		e, ok := interp.App.Entities[name]
		if !ok {
			panic(c.Fail(0, "%s não é um dado declarado", name))
		}
		return e
	}
	interp.RegisterModule("acesso", map[string]ModuleFunc{
		// acesso.nivel("projeto", registro) → nível de quem está agindo
		"nivel": func(c *Call, args []any) any {
			e := entity(c, c.Str(args, 0, "dado"))
			rec, _ := c.Arg(args, 1, "registro").(map[string]any)
			return float64(interp.Level(c.Ctx(), atualOf(c), e, rec))
		},
		// acesso.papel("maintainer") → nível numérico do papel
		"papel": func(c *Call, args []any) any {
			if interp.App == nil {
				return 0.0
			}
			return float64(interp.App.Level(c.Str(args, 0, "papel")))
		},
		// acesso.pode("editar", "projeto", registro)
		"pode": func(c *Call, args []any) any {
			e := entity(c, c.Str(args, 1, "dado"))
			rec, _ := c.Arg(args, 2, "registro").(map[string]any)
			return interp.Can(c.Ctx(), atualOf(c), e, c.Str(args, 0, "ação"), rec)
		},
		// acesso.nome_papel(30) → "developer" (maior papel com nível ≤ n)
		"nome_papel": func(c *Call, args []any) any {
			if interp.App == nil {
				return nil
			}
			n := int(c.Num(args, 0, "nível"))
			name := any(nil)
			for _, r := range interp.App.Roles {
				if r.Level <= n {
					name = r.Name
				}
			}
			return name
		},
		"administrador": func(c *Call, args []any) any {
			return interp.IsAdmin(atualOf(c))
		},
	})
}

func nilMap(m map[string]any) any {
	if m == nil {
		return nil
	}
	return m
}

// VisibleByVisibility reports visibility through the visibility field only.
func (interp *Interpreter) VisibleByVisibility(ctx *Context, atual map[string]any, e *ast.Entity, record map[string]any) bool {
	return interp.visible(ctx, atual, e, record)
}

// InheritsView: the entity has no rule about seeing it and belongs to
// something else, so visibility follows the parent.
func InheritsView(app *ast.App, e *ast.Entity) bool {
	if len(e.Rules["ver"])+len(e.Rules["editar"])+len(e.Rules["excluir"]) > 0 || e.Singular == app.MemberModel {
		return false
	}
	for _, target := range e.Parents {
		if target != app.LoginEntity {
			return true
		}
	}
	return false
}

func (interp *Interpreter) restrictedAllows(ctx *Context, atual map[string]any, e *ast.Entity, rs *ast.Restriction, record map[string]any) bool {
	if atual == nil {
		return false
	}
	uid := toNumber(atual["id"])
	for _, f := range rs.Owners {
		if record[f] != nil && toNumber(record[f]) == uid {
			return true
		}
	}
	for _, f := range rs.Lists {
		if list, ok := record[f].([]any); ok {
			for _, it := range list {
				if toNumber(it) == uid {
					return true
				}
			}
		}
	}
	return rs.MinRole != "" && interp.Level(ctx, atual, e, record) >= interp.App.Level(rs.MinRole)
}

// Transition moves a record through its state machine and stamps when and
// by whom (fechada_em, fechada_por_id). Returning to the initial state
// clears the stamps.
func (interp *Interpreter) Transition(ctx *Context, atual map[string]any, e *ast.Entity, tr *ast.Transition, record map[string]any) (map[string]any, error) {
	for _, final := range e.Finals {
		if toString(record[e.StateField]) == final {
			msg := fmt.Sprintf("%s está %s e não muda mais", e.Label, final)
			if interp.App.Messages == "en" {
				msg = fmt.Sprintf("%s is already %s", e.Label, interp.external(final))
			}
			return nil, &RuntimeError{Status: 405, Message: msg}
		}
	}
	if toString(record[e.StateField]) == tr.Target {
		msg := fmt.Sprintf("%s já está %s", e.Label, tr.Target)
		if interp.App.Messages == "en" {
			msg = fmt.Sprintf("%s is already %s", e.Label, interp.external(tr.Target))
		}
		return nil, &RuntimeError{Status: 400, Message: msg}
	}
	change := map[string]any{e.StateField: tr.Target}
	if tr.Stamp {
		change[tr.Target+"_em"] = time.Now().UTC().Format(time.RFC3339)
		if atual != nil {
			change[tr.Target+"_por_id"] = atual["id"]
		}
	} else {
		for _, other := range e.Transitions {
			if other.Stamp {
				change[other.Target+"_em"] = nil
				change[other.Target+"_por_id"] = nil
			}
		}
	}
	res, err := interp.Op(ctx, e.Singular, "atualizar", record["id"], change)
	if err != nil {
		return nil, err
	}
	return res.(map[string]any), nil
}

func (interp *Interpreter) external(name string) string {
	if v, ok := interp.App.Vocabulary[name]; ok {
		return v
	}
	return name
}

// hiddenMemberedAncestor: the nearest ancestor that has members (or the
// record itself when it has members) is not visible through visibility.
func (interp *Interpreter) hiddenMemberedAncestor(ctx *Context, atual map[string]any, e *ast.Entity, record map[string]any, depth int) bool {
	if depth > 8 || record == nil || interp.App == nil {
		return false
	}
	app := interp.App
	if e.Singular == app.MemberModel {
		if target, ok := app.Entities[toString(record["recurso"])]; ok {
			return interp.hiddenMemberedAncestor(ctx, atual, target, interp.load(ctx, target, record["recurso_id"]), depth+1)
		}
		return false
	}
	if e.HasMembers || e.InheritVia != "" {
		if record["id"] == nil && depth == 0 {
			// being created: judged by its parent below
		} else {
			return !interp.visible(ctx, atual, e, record)
		}
	}
	for field, target := range e.Parents {
		if target == app.LoginEntity || record[field] == nil {
			continue
		}
		pe := app.Entities[target]
		if interp.hasMembersChain(pe, 0) && interp.hiddenMemberedAncestor(ctx, atual, pe, interp.load(ctx, pe, record[field]), depth+1) {
			return true // any hidden ancestor hides it
		}
	}
	return false
}

// seesDescribed: atual sees the record a history entry describes; once that
// record is gone, its parent decides; with neither, only the author sees it.
func (interp *Interpreter) seesDescribed(ctx *Context, atual map[string]any, vt *ast.ViewThrough, record map[string]any) bool {
	look := func(kind, id string) (*ast.Entity, map[string]any) {
		e := interp.App.Entities[toString(record[kind])]
		if e == nil || record[id] == nil {
			return nil, nil
		}
		res, _ := interp.Op(ctx, e.Singular, "buscar", record[id])
		row, _ := res.(map[string]any)
		return e, row
	}
	if e, row := look(vt.Kind, vt.ID); row != nil {
		return interp.Can(ctx, atual, e, "ver", row)
	}
	if e, row := look(vt.ParentKind, vt.ParentID); row != nil {
		return interp.Can(ctx, atual, e, "ver", row)
	}
	return atual != nil && record[vt.Author] != nil && toString(record[vt.Author]) == toString(atual["id"])
}
