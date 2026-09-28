package servidor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// Events: every operation on data emits an event (kind = the data's plural,
// or enviar_codigo for pushes). Subscribers such as webhooks
// (`webhook recebe eventos do projeto`) get a delivery task for each event
// of their owner record.

// ownerOf walks up the parents of row until a record of the owner entity.
func (a *intentAPI) ownerOf(ctx *interp.Context, e *ast.Entity, row map[string]any, owner string, depth int) map[string]any {
	if e.Singular == owner {
		return row
	}
	if depth > 8 || row == nil {
		return nil
	}
	if e.Singular == a.app.MemberModel {
		if t := a.app.Entities[toStr(row["recurso"])]; t != nil {
			res, _ := a.in.Op(ctx, t.Singular, "buscar", row["recurso_id"])
			m, _ := res.(map[string]any)
			return a.ownerOf(ctx, t, m, owner, depth+1)
		}
		return nil
	}
	for field, target := range e.Parents {
		if target == a.app.LoginEntity || row[field] == nil {
			continue
		}
		pe := a.app.Entities[target]
		res, _ := a.in.Op(ctx, pe.Singular, "buscar", row[field])
		m, _ := res.(map[string]any)
		if found := a.ownerOf(ctx, pe, m, owner, depth+1); found != nil {
			return found
		}
	}
	return nil
}

// emit queues deliveries for subscribers interested in this event.
func (a *intentAPI) emit(ctx *interp.Context, e *ast.Entity, action string, row map[string]any, atual map[string]any) {
	kind := e.Plural
	if action == "enviar_codigo" {
		kind = "enviar_codigo"
	}
	for _, n := range a.app.Order {
		sub := a.app.Entities[n]
		s := sub.Subscription
		if s == nil {
			continue
		}
		wanted := false
		for _, k := range s.Kinds {
			wanted = wanted || k == kind
		}
		if !wanted {
			continue
		}
		owner := a.ownerOf(ctx, e, row, s.Owner, 0)
		if owner == nil {
			continue
		}
		res, err := a.in.Op(ctx, sub.Singular, "filtrar", map[string]any{s.OwnerField: owner["id"], "eventos_" + kind: true}, map[string]any{"limite": 100})
		if err != nil {
			continue
		}
		ownerEntity := a.app.Entities[s.Owner]
		for _, it := range res.([]any) {
			hook := it.(map[string]any)
			payload := map[string]any{"evento": kind, "acao": action, "dados": serialize(e, row), ownerEntity.Singular: serialize(ownerEntity, owner)}
			if atual != nil {
				le := a.app.Entities[a.app.LoginEntity]
				payload["quem"] = serializeFor(a.in, nil, le, atual, false)
			}
			a.s.tasks().enqueue(ctx, "entrega", map[string]any{"entidade": sub.Singular, "id": hook["id"], "evento": kind, "payload": payload})
		}
	}
}

// deliver posts one event to a subscriber's URL with its token.
func (a *intentAPI) deliver(d map[string]any) error {
	ctx := &interp.Context{}
	sub := a.app.Entities[toStr(d["entidade"])]
	if sub == nil {
		return nil
	}
	res, _ := a.in.Op(ctx, sub.Singular, "buscar", d["id"])
	hook, _ := res.(map[string]any)
	if hook == nil {
		return nil // removed meanwhile
	}
	ext := *a
	ext.extern = true
	payload := d["payload"]
	if len(a.app.Vocabulary) > 0 {
		out := ext.outward(payload).(map[string]any)
		for _, k := range []string{"evento", ext.ext("evento")} {
			if v, ok := out[k].(string); ok {
				out[k] = ext.ext(v)
			}
		}
		payload = out
	}
	body, _ := json.Marshal(payload)
	url := toStr(hook["url"])
	req, err := newJSONRequest(url, body)
	if err != nil {
		return err
	}
	req.Header.Set(headerName(ext, "cabecalho_evento", "X-Germanio-Event"), ext.ext(toStr(d["evento"])))
	if tok := toStr(hook["token"]); tok != "" {
		req.Header.Set(headerName(ext, "cabecalho_token", "X-Germanio-Token"), tok)
	}
	resp, err := safeHTTPClient().Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s respondeu %d", url, resp.StatusCode)
	}
	return nil
}

func headerName(a intentAPI, key, def string) string {
	if v := a.ext(key); v != key {
		return v
	}
	return def
}

func newJSONRequest(url string, body []byte) (*httpRequest, error) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("url inválida: %s", url)
	}
	req, err := newRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Germanio-Webhook/1.0")
	return req, nil
}

type httpRequest = http.Request

var newRequest = http.NewRequest
