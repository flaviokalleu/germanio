package servidor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// What an adapter may ask of the application itself (docs/gep/0033-
// adaptador-pede-a-aplicacao.md, em teste). An adapter translates an external
// protocol; when one external call is a different call of the application's
// own integration surface (another path, other names, a lookup first), it
// asks for it here instead of reading or writing records directly:
//
//	superficie.pedir(metodo, caminho [, corpo [, consulta]]) → {estado, corpo, cabecalhos}
//	superficie.contar(dado [, estado])                        → número
//
// pedir runs one operation of the generated integration surface as the person
// of the current request — the same identity, token scopes, rules, visibility,
// transaction, history, pending items and events as if that person had called
// it. contar counts like the indicators of a page (GEP 0012): only what the
// person may see. Neither ever grants anything the person could not do.

// innerKey marks a request made by superficie.pedir.
type innerKey struct{}

// maxInnerBody bounds the answer pedir keeps in memory.
const maxInnerBody = 16 << 20

var innerMethods = map[string]bool{"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true}

// innerWriter keeps the answer of an inner request.
type innerWriter struct {
	header http.Header
	status int
	body   bytes.Buffer
	over   bool
}

func (w *innerWriter) Header() http.Header { return w.header }

func (w *innerWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
}

func (w *innerWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if w.body.Len()+len(b) > maxInnerBody {
		w.over = true
		return len(b), nil
	}
	return w.body.Write(b)
}

// surfacePath joins the path asked for under the integration prefix. A list
// is a path of segments, each escaped (an address with "/" is one segment);
// a text is used as written and may not leave the prefix.
func surfacePath(prefix string, v any) (string, error) {
	var segs []string
	switch x := v.(type) {
	case string:
		rel := strings.Trim(x, "/")
		if rel == "" || strings.ContainsAny(rel, "?#\\") || strings.Contains(rel, "//") {
			return "", fmt.Errorf("o caminho %q precisa ser relativo à integração, sem ? nem # (a consulta vai no quarto argumento)", x)
		}
		for _, s := range strings.Split(rel, "/") {
			if s == "." || s == ".." {
				return "", fmt.Errorf("o caminho %q não pode sair da integração", x)
			}
		}
		return prefix + "/" + rel, nil
	case []any:
		for _, s := range x {
			seg := toStr(s)
			if seg == "" {
				return "", fmt.Errorf("o caminho tem um pedaço vazio: %v", x)
			}
			segs = append(segs, url.PathEscape(seg))
		}
	}
	if len(segs) == 0 {
		return "", fmt.Errorf("o caminho é um texto (\"projects/1/issues\") ou uma lista de pedaços ([\"projects\", id, \"issues\"])")
	}
	return prefix + "/" + strings.Join(segs, "/"), nil
}

func (s *Servidor) registerSurfaceModule(a *intentAPI) {
	a.in.RegisterModule("superficie", map[string]interp.ModuleFunc{
		"pedir": func(c *interp.Call, args []any) any {
			ctx := c.Ctx()
			if ctx == nil || ctx.Request == nil || ctx.Writer == nil {
				panic(c.Fail(0, "superficie.pedir existe só numa rota: é um adaptador respondendo a um pedido de alguém"))
			}
			orig := ctx.Request
			if ctx.DB != nil || txOf(orig) != nil {
				panic(c.Fail(0, "superficie.pedir não pode ser usado dentro de uma alteração (antes de, quando): cada pedido já é uma alteração inteira"))
			}
			if orig.Context().Value(innerKey{}) != nil {
				panic(c.Fail(0, "superficie.pedir não pode ser chamado por um pedido que veio de superficie.pedir"))
			}
			method := strings.ToUpper(c.Str(args, 0, "metodo"))
			if !innerMethods[method] {
				panic(c.Fail(0, "superficie.pedir: método %q (use GET, POST, PUT, PATCH ou DELETE)", method))
			}
			path, err := surfacePath(a.app.Integration, c.Arg(args, 1, "caminho"))
			if err != nil {
				panic(c.Fail(0, "superficie.pedir: %v", err))
			}
			var body []byte
			if len(args) > 2 && args[2] != nil {
				if body, err = json.Marshal(args[2]); err != nil {
					panic(c.Fail(0, "superficie.pedir: o corpo não vira JSON: %v", err))
				}
			}
			if len(args) > 3 && args[3] != nil {
				q := url.Values{}
				keys := []string{}
				m := c.Map(args, 3, "consulta")
				for k := range m {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					if m[k] != nil {
						q.Set(k, toStr(m[k]))
					}
				}
				if len(q) > 0 {
					path += "?" + q.Encode()
				}
			}
			req, err := http.NewRequestWithContext(context.WithValue(orig.Context(), innerKey{}, true), method, path, bytes.NewReader(body))
			if err != nil {
				panic(c.Fail(0, "superficie.pedir: %v", err))
			}
			// the same person: credentials and session travel, the body does not
			req.Header = orig.Header.Clone()
			for _, h := range []string{"Content-Length", "Content-Type", "Content-Range", "Content-Encoding", "Accept-Encoding", "Range", "If-Match", "If-None-Match", "If-Modified-Since", "Expect"} {
				req.Header.Del(h)
			}
			if body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			if sess := interp.SessaoDaRequisicao(orig); sess != nil {
				// the outer route already proved the request came from the app
				if csrf, _ := sess["csrf"].(string); csrf != "" {
					req.Header.Set("X-CSRF-Token", csrf)
				}
			}
			req.RemoteAddr, req.Host = orig.RemoteAddr, orig.Host
			w := &innerWriter{header: http.Header{}}
			s.mux.ServeHTTP(w, req)
			if w.over {
				panic(c.Fail(0, "superficie.pedir: a resposta de %s %s passou de %d MB", method, path, maxInnerBody>>20))
			}
			if w.status == 0 {
				w.status = http.StatusOK
			}
			headers := map[string]any{}
			for k, v := range w.header {
				if len(v) > 0 {
					headers[k] = v[0]
				}
			}
			var out any
			if raw := w.body.Bytes(); len(bytes.TrimSpace(raw)) > 0 {
				ct, _, _ := mime.ParseMediaType(w.header.Get("Content-Type"))
				if ct != "application/json" || json.Unmarshal(raw, &out) != nil {
					out = string(raw)
				}
			}
			return map[string]any{"estado": float64(w.status), "corpo": out, "cabecalhos": headers}
		},
		"contar": func(c *interp.Call, args []any) any {
			ctx := c.Ctx()
			if ctx == nil || ctx.Request == nil {
				panic(c.Fail(0, "superficie.contar existe só numa rota: conta o que a pessoa do pedido pode ver"))
			}
			name := c.Str(args, 0, "dado")
			var e *ast.Entity
			for _, n := range a.app.Order {
				if x := a.app.Entities[n]; x.Singular == name || x.Plural == name {
					e = x
				}
			}
			if e == nil {
				panic(c.Fail(0, "superficie.contar: não conheço o dado %q", name))
			}
			filters := map[string]any{}
			if len(args) > 1 && args[1] != nil {
				st := toStr(args[1])
				if !statesOf(e)[st] || e.StateField == "" {
					var known []string
					for k := range statesOf(e) {
						if k != "" {
							known = append(known, k)
						}
					}
					sort.Strings(known)
					panic(c.Fail(0, "superficie.contar: %q não é um estado de %s (estados: %s)", st, e.Plural, strings.Join(known, ", ")))
				}
				filters[e.StateField] = st
			}
			atual, err := s.identify(ctx, ctx.Request)
			if err != nil {
				panic(c.Fail(401, "401 Unauthorized"))
			}
			n, ok := a.countVisible(ctx, atual, e, filters)
			if !ok {
				return 0.0 // the person may not see this data at all
			}
			return float64(n)
		},
	})
}
