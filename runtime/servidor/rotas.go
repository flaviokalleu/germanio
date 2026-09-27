package servidor

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
)

// maxRouteBody bounds request bodies read for .ge routes.
const maxRouteBody = 10 << 20

// RoutePattern converts a Germanio route path into a net/http pattern:
// "/api/projetos/:id" → "/api/projetos/{id}", "/raw/*caminho" →
// "/raw/{caminho...}" and "/" → "/{$}" (exact root).
func RoutePattern(method, path string) (string, error) {
	if !strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("a rota %q deve começar com /", path)
	}
	segs := strings.Split(path, "/")
	seen := map[string]bool{}
	for i, seg := range segs {
		switch {
		case strings.HasPrefix(seg, ":"):
			name := seg[1:]
			if name == "" || seen[name] {
				return "", fmt.Errorf("parâmetro inválido ou repetido em %q", path)
			}
			seen[name] = true
			segs[i] = "{" + name + "}"
		case strings.HasPrefix(seg, "*"):
			name := seg[1:]
			if name == "" || i != len(segs)-1 {
				return "", fmt.Errorf("*%s só pode ser o último segmento de %q", name, path)
			}
			segs[i] = "{" + name + "...}"
		case strings.ContainsAny(seg, "{}"):
			return "", fmt.Errorf("use :nome para parâmetros em %q", path)
		}
	}
	p := strings.Join(segs, "/")
	if p == "/" {
		p = "/{$}"
	}
	if method != "" {
		p = method + " " + p
	}
	return p, nil
}

func paramNames(path string) []string {
	var names []string
	for _, seg := range strings.Split(path, "/") {
		if strings.HasPrefix(seg, ":") || strings.HasPrefix(seg, "*") {
			names = append(names, seg[1:])
		}
	}
	return names
}

// registerRoutes wires every `rota` of the program into mux.
func (s *Servidor) registerRoutes(mux *http.ServeMux) error {
	seen := map[string]*ast.CustomRoute{}
	for _, route := range s.Program.Routes {
		r := route
		pattern, err := RoutePattern(r.Method, r.Path)
		if err != nil {
			return fmt.Errorf("%s:%d: %w", r.Pos.File, r.Pos.Line, err)
		}
		if prev, dup := seen[pattern]; dup {
			return fmt.Errorf("rota duplicada %s %s (%s:%d e %s:%d)", r.Method, r.Path, prev.Pos.File, prev.Pos.Line, r.Pos.File, r.Pos.Line)
		}
		seen[pattern] = r
		var regErr error
		func() {
			defer func() {
				if p := recover(); p != nil {
					regErr = fmt.Errorf("rota %s %s conflita com outra rota: %v", r.Method, r.Path, p)
				}
			}()
			mux.HandleFunc(pattern, func(w http.ResponseWriter, req *http.Request) { s.serveRoute(w, req, r) })
		}()
		if regErr != nil {
			return regErr
		}
	}
	return nil
}

// buildRequest converts an HTTP request into the .ge `requisicao` map.
func buildRequest(req *http.Request, route *ast.CustomRoute) (map[string]any, error) {
	params := map[string]any{}
	for _, n := range paramNames(route.Path) {
		params[n] = req.PathValue(n)
	}
	query := map[string]any{}
	for k, v := range req.URL.Query() {
		if len(v) > 0 {
			query[k] = v[0]
		}
	}
	headers := map[string]any{}
	for k, v := range req.Header {
		if len(v) > 0 {
			headers[strings.ToLower(k)] = v[0]
		}
	}
	var body any
	if req.Body != nil && req.Method != http.MethodGet && req.Method != http.MethodHead {
		ct, _, _ := mime.ParseMediaType(req.Header.Get("Content-Type"))
		switch ct {
		case "application/json":
			data, err := io.ReadAll(io.LimitReader(req.Body, maxRouteBody+1))
			if err != nil {
				return nil, err
			}
			if len(data) > maxRouteBody {
				return nil, &interp.RuntimeError{Status: http.StatusRequestEntityTooLarge, Message: "corpo grande demais"}
			}
			if len(strings.TrimSpace(string(data))) > 0 {
				if err := json.Unmarshal(data, &body); err != nil {
					return nil, &interp.RuntimeError{Status: http.StatusBadRequest, Message: "JSON inválido: " + err.Error()}
				}
			}
		case "application/x-www-form-urlencoded", "multipart/form-data":
			req.Body = http.MaxBytesReader(nil, req.Body, maxRouteBody)
			var err error
			if ct == "multipart/form-data" {
				err = req.ParseMultipartForm(maxRouteBody)
			} else {
				err = req.ParseForm()
			}
			if err != nil {
				return nil, &interp.RuntimeError{Status: http.StatusBadRequest, Message: "formulário inválido: " + err.Error()}
			}
			form := map[string]any{}
			for k, v := range req.PostForm {
				if len(v) == 1 {
					form[k] = v[0]
				} else {
					list := make([]any, len(v))
					for i, x := range v {
						list[i] = x
					}
					form[k] = list
				}
			}
			body = form
		}
	}
	return map[string]any{
		"metodo":     req.Method,
		"caminho":    req.URL.Path,
		"parametros": params,
		"consulta":   query,
		"corpo":      body,
		"cabecalhos": headers,
		"sessao":     nilIfEmpty(interp.SessaoDaRequisicao(req)),
		"ip":         getRealIP(req),
	}, nil
}

func nilIfEmpty(m map[string]any) any {
	if m == nil {
		return nil
	}
	return m
}

var unsafeMethods = map[string]bool{"POST": true, "PUT": true, "PATCH": true, "DELETE": true}

// checkCSRF: a request carrying a session cookie that changes state must
// prove it came from the application (header X-CSRF-Token or field _csrf).
// Requests without the session cookie (API tokens, git clients) are not
// affected because browsers cannot attach those credentials cross-site.
func checkCSRF(req *http.Request, requisicao map[string]any) bool {
	if !unsafeMethods[req.Method] {
		return true
	}
	sess, ok := requisicao["sessao"].(map[string]any)
	if !ok {
		return true
	}
	want, _ := sess["csrf"].(string)
	got := req.Header.Get("X-CSRF-Token")
	if got == "" {
		if form, ok := requisicao["corpo"].(map[string]any); ok {
			got, _ = form["_csrf"].(string)
			delete(form, "_csrf")
		}
	}
	return want != "" && got == want
}

func (s *Servidor) serveRoute(w http.ResponseWriter, req *http.Request, route *ast.CustomRoute) {
	if s.Interpreter == nil {
		s.jsonError(w, "interpretador não inicializado", http.StatusInternalServerError)
		return
	}
	requisicao, err := buildRequest(req, route)
	if err != nil {
		s.writeRouteError(w, req, route, err)
		return
	}
	if !checkCSRF(req, requisicao) {
		s.writeRouteError(w, req, route, &interp.RuntimeError{Status: http.StatusForbidden, Message: "token CSRF ausente ou inválido"})
		return
	}
	ctx := &interp.Context{Request: req, Writer: w}
	resp, output, err := s.Interpreter.ExecRoute(route.Handler, ctx, map[string]any{"requisicao": requisicao})
	for _, c := range ctx.SetCookies {
		http.SetCookie(w, c)
	}
	if ctx.Written {
		return
	}
	if err != nil {
		s.writeRouteError(w, req, route, err)
		return
	}
	if resp == nil {
		// Legacy routes answered with what they printed.
		w.Header().Set("Content-Type", "application/json")
		if len(output) > 0 {
			json.NewEncoder(w).Encode(map[string]any{"resultado": output[len(output)-1], "output": output})
		} else {
			json.NewEncoder(w).Encode(map[string]any{"ok": true})
		}
		return
	}
	s.writeResponse(w, resp)
}

// Renderer turns special response bodies (for example UI trees) into bytes.
// It returns ok=false for bodies it does not handle.
type Renderer func(req *http.Request, body any) (contentType string, data []byte, ok bool)

func (s *Servidor) writeResponse(w http.ResponseWriter, resp *interp.Response) {
	for k, v := range resp.Headers {
		if strings.ContainsAny(k+v, "\r\n") {
			continue
		}
		w.Header().Set(k, v)
	}
	status := resp.Status
	if status == 0 {
		status = http.StatusOK
	}
	if resp.Redirect != "" {
		w.Header().Set("Location", resp.Redirect)
		w.WriteHeader(status)
		return
	}
	if resp.Raw != nil {
		if resp.ContentType != "" {
			w.Header().Set("Content-Type", resp.ContentType)
		}
		w.WriteHeader(status)
		w.Write(resp.Raw)
		return
	}
	if s.Render != nil {
		if ct, data, ok := s.Render(nil, resp.Body); ok {
			w.Header().Set("Content-Type", ct)
			w.WriteHeader(status)
			w.Write(data)
			return
		}
	}
	if str, ok := resp.Body.(string); ok && w.Header().Get("Content-Type") != "" {
		w.WriteHeader(status)
		io.WriteString(w, str)
		return
	}
	if resp.Body == nil && status == http.StatusNoContent {
		w.WriteHeader(status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp.Body)
}

func (s *Servidor) writeRouteError(w http.ResponseWriter, req *http.Request, route *ast.CustomRoute, err error) {
	var re *interp.RuntimeError
	status, msg := http.StatusInternalServerError, "500 Internal Server Error"
	if errors.As(err, &re) && re.Status >= 400 && re.Status < 600 {
		status, msg = re.Status, re.Message
	} else {
		// Unexpected failures are logged with position and hidden from clients
		// unless GERMANIO_DEBUG=1.
		log.Printf("[germanio] ERRO %s %s: %v", req.Method, req.URL.Path, err)
		if os.Getenv("GERMANIO_DEBUG") == "1" {
			msg = err.Error()
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"message": msg})
}
