package servidor

import (
	"fmt"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// The place of a new record, named by its address (docs/gep/0044-lugar-
// pelo-endereco.md, em teste). A data with `endereço dentro do grupo ou
// do criador` is created (or copied) inside one of its containers or, when
// the address may be the creator's, in the person's own space. Besides the
// reference to the container, the request may name the place by its
// address: `lugar` is "empresa/web" (a container's address) or the person's
// own name (their own space). Nothing new is granted: the container must be
// one the person sees (404 otherwise) and every rule of creating there
// still applies; another person's space is refused (403), even to an
// administrator.

const placeKey = "lugar"

// placeIn replaces body["lugar"] by the reference it names.
func (a *intentAPI) placeIn(ctx *interp.Context, atual map[string]any, e *ast.Entity, body map[string]any) error {
	raw, given := body[placeKey]
	if !given || e.Address == nil {
		return nil
	}
	delete(body, placeKey)
	where := strings.Trim(strings.TrimSpace(toStr(raw)), "/")
	if where == "" {
		return nil
	}
	containers := func(except string) {
		for _, ref := range e.Address.Within {
			if ref.Field != except && ref.Entity != a.app.LoginEntity {
				body[ref.Field] = nil
			}
		}
	}
	// a container with that address
	for _, ref := range e.Address.Within {
		ce := a.app.Entities[ref.Entity]
		if ref.Entity == a.app.LoginEntity || ce == nil || ce.Address == nil {
			continue
		}
		res, err := a.in.Op(ctx, ce.Singular, "encontrar", map[string]any{ce.Address.Field: where})
		if err != nil {
			return err
		}
		row, _ := res.(map[string]any)
		if row == nil {
			continue
		}
		if !a.in.Can(ctx, atual, ce, "ver", row) {
			return &interp.RuntimeError{Status: 404, Message: a.msg("404", ce)}
		}
		containers(ref.Field)
		body[ref.Field] = row["id"]
		return nil
	}
	// a person's own space
	for _, ref := range e.Address.Within {
		if ref.Entity != a.app.LoginEntity || a.app.Login == nil || len(a.app.Login.Fields) == 0 {
			continue
		}
		res, err := a.in.Op(ctx, ref.Entity, "encontrar", map[string]any{strings.ToLower(a.app.Login.Fields[0]): where})
		if err != nil {
			return err
		}
		person, _ := res.(map[string]any)
		if person == nil {
			break
		}
		// Only one's own space: a record created in someone else's would be
		// owned (quem cria vira owner) by whoever created it, not by the
		// person whose space it is — even for an administrator.
		if atual == nil || toStr(person["id"]) != toStr(atual["id"]) {
			msg := fmt.Sprintf("%q é o espaço de outra pessoa: crie no seu (%s) ou num lugar onde você pode criar", where, toStr(atualName(a, atual)))
			if a.app.Messages == "en" {
				msg = "403 Forbidden"
			}
			return &interp.RuntimeError{Status: 403, Message: msg}
		}
		containers("")
		return nil
	}
	msg := fmt.Sprintf("O lugar %q não existe", where)
	if a.app.Messages == "en" {
		msg = "404 Not Found"
	}
	return &interp.RuntimeError{Status: 404, Message: msg}
}

func atualName(a *intentAPI, atual map[string]any) any {
	if atual == nil || a.app.Login == nil || len(a.app.Login.Fields) == 0 {
		return ""
	}
	return atual[strings.ToLower(a.app.Login.Fields[0])]
}
