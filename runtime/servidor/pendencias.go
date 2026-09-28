package servidor

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Pending items (docs/gep/0009-pendencias.md, em teste): a person placed in
// a pending field of a record (the assignees of an issue) receives a pending
// item for it; a person removed from the field loses the open ones; deleting
// the record deletes its pending items. Whoever makes the change does not
// get one for themselves.

func peopleIn(row map[string]any, field string) map[int64]bool {
	out := map[int64]bool{}
	if row == nil {
		return out
	}
	switch v := row[field].(type) {
	case nil:
	case []any:
		for _, it := range v {
			if n := int64(asNumber(it)); n > 0 {
				out[n] = true
			}
		}
	case string:
		var list []any
		if json.Unmarshal([]byte(v), &list) == nil {
			for _, it := range list {
				if n := int64(asNumber(it)); n > 0 {
					out[n] = true
				}
			}
		} else if n := int64(asNumber(v)); n > 0 {
			out[n] = true
		}
	default:
		if n := int64(asNumber(v)); n > 0 {
			out[n] = true
		}
	}
	return out
}

func (a *intentAPI) pending(ctx *interp.Context, atual map[string]any, e *ast.Entity, before, after map[string]any) error {
	pe := a.app.PendingEntity
	if pe == "" || after == nil {
		return nil
	}
	if err := a.mentioned(ctx, atual, e, before, after); err != nil {
		return err
	}
	if len(e.PendingFields) == 0 {
		return nil
	}
	self := int64(0)
	if atual != nil {
		self = int64(asNumber(atual["id"]))
	}
	for _, f := range e.PendingFields {
		old, now := peopleIn(before, f), peopleIn(after, f)
		for id := range now {
			if old[id] || id == self {
				continue
			}
			// the title only for someone who may see the record: a pending
			// item never tells its owner what they could not read
			motivo := fmt.Sprintf("%s: %s", label(f), e.Label)
			if a.personSees(ctx, id, e, after) {
				motivo += " " + titleOf(e, after)
			}
			if _, err := a.in.Op(ctx, pe, "criar", map[string]any{"dono_id": id, "recurso": e.Singular, "recurso_id": after["id"], "motivo": motivo}); err != nil {
				return err
			}
			a.noticeByEmail(ctx, id, motivo)
		}
		for id := range old {
			if now[id] {
				continue
			}
			if err := a.dropPending(ctx, e, after["id"], map[string]any{"dono_id": id, "estado": "aberta"}); err != nil {
				return err
			}
		}
	}
	return nil
}

// dropPending deletes the pending items of a record (optionally narrowed).
func (a *intentAPI) dropPending(ctx *interp.Context, e *ast.Entity, id any, extra map[string]any) error {
	pe := a.app.PendingEntity
	if pe == "" || len(e.PendingFields) == 0 {
		return nil
	}
	filters := map[string]any{"recurso": e.Singular, "recurso_id": id}
	for k, v := range extra {
		filters[k] = v
	}
	for {
		rows, err := a.in.Op(ctx, pe, "filtrar", filters, map[string]any{"limite": 500, "ordenar": "id"})
		if err != nil {
			return err
		}
		list := rows.([]any)
		if len(list) == 0 {
			return nil
		}
		for _, it := range list {
			if _, err := a.in.Op(ctx, pe, "deletar", it.(map[string]any)["id"]); err != nil {
				return err
			}
		}
	}
}

// personSees: the person with this id may see record of e.
func (a *intentAPI) personSees(ctx *interp.Context, id int64, e *ast.Entity, record map[string]any) bool {
	res, err := a.in.Op(ctx, a.app.LoginEntity, "buscar", id)
	person, _ := res.(map[string]any)
	if err != nil || person == nil {
		return false
	}
	return a.in.Can(ctx, person, e, "ver", record)
}

// noticeByEmail sends the pending item to its owner by e-mail (GEP 0013, em
// teste) — after the change is saved, like every external effect (G86).
func (a *intentAPI) noticeByEmail(ctx *interp.Context, id int64, motivo string) {
	if !a.app.EmailNotices {
		return
	}
	send, why := mailerFromEnv()
	le := a.app.Entities[a.app.LoginEntity]
	field := emailField(le)
	if why != "" || field == "" {
		return // said once at start
	}
	res, _ := a.in.Op(ctx, le.Singular, "buscar", id)
	person, _ := res.(map[string]any)
	to := toStr(person[field])
	if to == "" {
		return
	}
	subject := "Nova pendência — " + a.s.Program.System.Name
	text := motivo
	if public := strings.TrimSuffix(os.Getenv("GERMANIO_URL_PUBLICA"), "/"); public != "" {
		text += "\n\nVeja suas pendências em " + public // the declared address, never the request's Host
	}
	run := func() error { return send(to, subject, text) }
	if ctx != nil && ctx.Effects != nil {
		ctx.Effects.Add(interp.Effect{Kind: "e-mail de aviso", Run: run})
		return
	}
	if err := run(); err != nil {
		fmt.Printf("[germanio] aviso por e-mail: %v\n", err)
	}
}

var mentionRe = regexp.MustCompile(`(?:^|[^\w@])@([A-Za-z0-9_][A-Za-z0-9_.-]*)`)

// mentionsIn: the names written as @name in the long texts of row.
func mentionsIn(e *ast.Entity, row map[string]any) map[string]bool {
	out := map[string]bool{}
	if row == nil {
		return out
	}
	for _, f := range e.Model.Fields {
		if f.Type != ast.FieldTextoLongo {
			continue
		}
		for _, m := range mentionRe.FindAllStringSubmatch(toStr(row[strings.ToLower(f.Name)]), 50) {
			out[strings.TrimRight(m[1], ".-")] = true
		}
	}
	return out
}

// mentioned: people newly mentioned receive a pending item (GEP 0017),
// only if they may see the record — a mention never reveals it.
func (a *intentAPI) mentioned(ctx *interp.Context, atual map[string]any, e *ast.Entity, before, after map[string]any) error {
	if !e.PendingMentions || a.app.HandleField == "" {
		return nil
	}
	old, now := mentionsIn(e, before), mentionsIn(e, after)
	self := int64(0)
	if atual != nil {
		self = int64(asNumber(atual["id"]))
	}
	for name := range now {
		if old[name] {
			continue
		}
		res, err := a.in.Op(ctx, a.app.LoginEntity, "encontrar", map[string]any{a.app.HandleField: name})
		person, _ := res.(map[string]any)
		if err != nil || person == nil {
			continue
		}
		id := int64(asNumber(person["id"]))
		if id == self || !a.in.Can(ctx, person, e, "ver", after) {
			continue
		}
		motivo := fmt.Sprintf("Menção: %s %s", e.Label, titleOf(e, after))
		if _, err := a.in.Op(ctx, a.app.PendingEntity, "criar", map[string]any{"dono_id": id, "recurso": e.Singular, "recurso_id": after["id"], "motivo": motivo}); err != nil {
			return err
		}
		a.noticeByEmail(ctx, id, motivo)
	}
	return nil
}
