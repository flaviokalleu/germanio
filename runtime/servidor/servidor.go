package servidor

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/lexer"
	"github.com/flaviokalleu/germanio/compiler/parser"
	authpkg "github.com/flaviokalleu/germanio/runtime/auth"
	"github.com/flaviokalleu/germanio/runtime/banco"
	emailpkg "github.com/flaviokalleu/germanio/runtime/email"
	"github.com/flaviokalleu/germanio/runtime/httpclient"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
	"github.com/flaviokalleu/germanio/runtime/jobs"
	wa "github.com/flaviokalleu/germanio/runtime/whatsapp"
)

// Servidor is the embedded Germanio web server.
type Servidor struct {
	Program     *ast.Program
	DB          *banco.Banco
	Porta       string
	WS          *WSHub
	WA          *wa.Client
	Auth        *authpkg.Auth
	Email       *emailpkg.Client
	HTTPClient  *httpclient.Client
	Interpreter *interp.Interpreter
	Jobs        *jobs.Queue
	rateLimiter map[string][]time.Time
	rateMu      sync.Mutex
	presence    map[string]map[string]any
	presenceMu  sync.RWMutex
	htmlCache   string
	htmlCacheMu sync.RWMutex
}

// Novo creates a new server.
func Novo(program *ast.Program, db *banco.Banco, porta string) *Servidor {
	return &Servidor{
		Program: program, DB: db, Porta: porta, WS: NewWSHub(),
		Jobs: jobs.Nova(4, 256), rateLimiter: make(map[string][]time.Time),
		presence: make(map[string]map[string]any),
	}
}

// Iniciar starts the HTTP server.
func (s *Servidor) Iniciar() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", s.handlePagina)
	mux.HandleFunc("/api/", s.handleAPI)
	mux.HandleFunc("/upload", s.handleUpload)
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))
	mux.HandleFunc("/media/stream", s.handleMediaStream)
	mux.HandleFunc("/ws", s.WS.HandleWS)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/api/docs", s.handleOpenAPISwagger)
	mux.HandleFunc("/api/openapi.json", s.handleOpenAPIJSON)

	// Auth routes
	if s.Auth != nil {
		mux.HandleFunc("/api/login", s.Auth.Login)
		mux.HandleFunc("/api/registro", s.Auth.Registrar)
		mux.HandleFunc("/api/register", s.Auth.Registrar)
		mux.HandleFunc("/api/me", s.Auth.Me)
	}

	// Proxy endpoint for frontend to call external APIs
	mux.HandleFunc("/api/_proxy", s.handleProxy)
	mux.HandleFunc("/api/_presence", s.handlePresence)
	mux.HandleFunc("/api/_jobs/status", s.handleJobsStatus)
	mux.HandleFunc("/api/whatsapp/sessions", s.handleWASessions)
	mux.HandleFunc("/api/whatsapp/connect", s.handleWAConnect)
	mux.HandleFunc("/api/whatsapp/qr", s.handleWAQR)
	mux.HandleFunc("/api/whatsapp/send", s.handleWASend)

	// Scripting endpoints
	mux.HandleFunc("/api/_eval", s.handleEval)
	mux.HandleFunc("/api/_log", s.handleLog)

	// Custom routes
	for _, route := range s.Program.Routes {
		r := route // capture for closure
		mux.HandleFunc(r.Path, func(w http.ResponseWriter, req *http.Request) {
			if r.Method != "" && req.Method != r.Method {
				s.jsonError(w, "método não permitido", http.StatusMethodNotAllowed)
				return
			}
			if s.Interpreter != nil {
				output := s.Interpreter.EvalStatements(r.Handler, nil)
				w.Header().Set("Content-Type", "application/json")
				if len(output) > 0 {
					json.NewEncoder(w).Encode(map[string]interface{}{
						"resultado": output[len(output)-1],
						"output":    output,
					})
				} else {
					json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
				}
			} else {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
			}
		})
	}

	// Prompt Events routes (quando receber / quando chamar)
	for _, evt := range s.Program.Events {
		if evt.Trigger == "receber" || evt.Trigger == "receive" || evt.Trigger == "chamar" || evt.Trigger == "call" {
			actionName := strings.TrimSpace(evt.Target)
			if actionName == "" {
				actionName = strings.TrimSpace(evt.ActionRef)
			}
			parts := strings.Fields(actionName)
			if len(parts) > 0 {
				slug := strings.ToLower(parts[0])
				routePath := "/api/" + slug
				mux.HandleFunc(routePath, func(w http.ResponseWriter, req *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{
						"status":   "sucesso",
						"acao":     slug,
						"executou": true,
					})
				})
			}
		}
	}

	// `handlePagina` is registered once at `/` and dispatches custom pages by
	// exact path. A single dispatcher avoids duplicate ServeMux registrations
	// while preserving the generated interface as the fallback.

	// Apply middleware chain
	var handler http.Handler = mux
	if s.Auth != nil {
		handler = s.Auth.Middleware(handler)
	}
	handler = s.middleware(handler)

	if s.WA != nil {
		s.WA.SetEventHandler(func(evt wa.Event) {
			s.WS.Broadcast(WSMessage{Type: evt.Type, Session: evt.Session, Data: evt})
		})
	}
	s.WS.OnConnect = func(count int) {
		s.WS.Broadcast(WSMessage{Type: "presenca_socket", Data: map[string]any{"connections": count}})
	}
	s.WS.OnDisconnect = func(count int) {
		s.WS.Broadcast(WSMessage{Type: "presenca_socket", Data: map[string]any{"connections": count}})
	}
	defer s.Jobs.Close()

	// Periodic rate limiter cleanup
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			s.rateMu.Lock()
			now := time.Now()
			for ip, times := range s.rateLimiter {
				var recent []time.Time
				for _, t := range times {
					if now.Sub(t) < time.Minute {
						recent = append(recent, t)
					}
				}
				if len(recent) == 0 {
					delete(s.rateLimiter, ip)
				} else {
					s.rateLimiter[ip] = recent
				}
			}
			s.rateMu.Unlock()
		}
	}()

	server := &http.Server{
		Addr:           ":" + s.Porta,
		Handler:        handler,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}
	return server.ListenAndServe()
}

func (s *Servidor) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self' https: data: 'unsafe-inline' 'unsafe-eval'; frame-ancestors 'none';")

		// CORS configurável via env CORS_ALLOWED_ORIGINS
		allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
		if allowedOrigins == "" {
			allowedOrigins = "*"
		}

		if strings.HasPrefix(r.URL.Path, "/api/") {
			origin := r.Header.Get("Origin")
			if allowedOrigins == "*" || (origin != "" && strings.Contains(allowedOrigins, origin)) {
				if origin != "" {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				} else {
					w.Header().Set("Access-Control-Allow-Origin", allowedOrigins)
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		// Rate limiting inteligente por IP real
		if strings.HasPrefix(r.URL.Path, "/api/") && r.Method == http.MethodPost {
			ip := getRealIP(r)
			s.rateMu.Lock()
			now := time.Now()
			var recent []time.Time
			for _, t := range s.rateLimiter[ip] {
				if now.Sub(t) < time.Minute {
					recent = append(recent, t)
				}
			}
			s.rateLimiter[ip] = append(recent, now)
			count := len(s.rateLimiter[ip])
			s.rateMu.Unlock()

			// Limite mais restrito para login (brute-force defense: 10/min) e geral (100/min)
			limit := 100
			if r.URL.Path == "/api/login" || r.URL.Path == "/api/auth/login" {
				limit = 10
			}
			if count > limit {
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"erro":"Muitas requisições. Bloqueio temporário ativado por segurança."}`))
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// getRealIP extrai o IP real do cliente considerando proxies confiáveis (Cloudflare, Nginx, etc.)
func getRealIP(r *http.Request) string {
	if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
		return strings.TrimSpace(cfIP)
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			return strings.TrimSpace(parts[0])
		}
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func (s *Servidor) getCachedHTML() string {
	s.htmlCacheMu.RLock()
	if s.htmlCache != "" {
		defer s.htmlCacheMu.RUnlock()
		return s.htmlCache
	}
	s.htmlCacheMu.RUnlock()

	html := s.renderHTML()
	s.htmlCacheMu.Lock()
	s.htmlCache = html
	s.htmlCacheMu.Unlock()
	return html
}

func (s *Servidor) handlePagina(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Custom pages take priority over the generated CRUD interface, including
	// the root path. This lets a Germanio project use `pagina "/"` as its
	// public site without losing the generated interface as fallback.
	for _, page := range s.Program.Pages {
		if page.Path != r.URL.Path {
			continue
		}
		if len(page.Blocks) > 0 {
			rendered := RenderDeclarativePage(page)
			_, _ = w.Write([]byte(rendered))
			return
		}
		trimmed := strings.TrimSpace(page.Content)
		if strings.HasPrefix(strings.ToLower(trimmed), "<!doctype") || strings.HasPrefix(strings.ToLower(trimmed), "<html") {
			_, _ = w.Write([]byte(trimmed))
			return
		}
		title := page.Title
		if title == "" && s.Program.System != nil {
			title = s.Program.System.Name
		}
		_, _ = fmt.Fprintf(w, `<!DOCTYPE html><html><head><meta charset="utf-8"><title>%s</title><style>body{font-family:system-ui;margin:0;padding:20px}</style></head><body>%s</body></html>`, title, page.Content)
		return
	}

	_, _ = w.Write([]byte(s.getCachedHTML()))
}

func (s *Servidor) handleEval(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Require authentication for code execution
	if s.Auth != nil {
		role := r.Header.Get("X-User-Role")
		if role != "admin" {
			s.jsonError(w, "Apenas administradores podem executar código", http.StatusForbidden)
			return
		}
	}

	if s.Interpreter == nil {
		http.Error(w, `{"error":"interpreter not initialized"}`, http.StatusInternalServerError)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 64*1024) // 64KB max

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}

	if req.Code == "" {
		http.Error(w, `{"error":"empty code"}`, http.StatusBadRequest)
		return
	}

	// Wrap code in a logica block for parsing
	wrappedCode := "sistema eval\nlogica\n"
	for _, line := range strings.Split(req.Code, "\n") {
		wrappedCode += "  " + line + "\n"
	}

	lex := lexer.New(wrappedCode)
	tokens, err := lex.Tokenize()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  fmt.Sprintf("erro lexico: %s", err),
			"output": []string{},
		})
		return
	}

	p := parser.New(tokens)
	program, err := p.Parse()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  fmt.Sprintf("erro de parsing: %s", err),
			"output": []string{},
		})
		return
	}

	output := s.Interpreter.EvalStatements(program.Scripts, program.Functions)
	if output == nil {
		output = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"output": output,
	})
}

func (s *Servidor) handleLog(w http.ResponseWriter, r *http.Request) {
	if s.Interpreter == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"logs":[]}`))
		return
	}

	clear := r.URL.Query().Get("clear") == "true"
	logs := s.Interpreter.GetLogs(clear)
	if logs == nil {
		logs = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs": logs,
	})
}

func (s *Servidor) handleAPI(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/")
	path = strings.TrimSuffix(path, "/")
	parts := strings.Split(path, "/")

	// Handle /api/_stats
	if parts[0] == "_stats" {
		s.handleStats(w, r)
		return
	}

	if parts[0] == "docs" {
		s.handleOpenAPISwagger(w, r)
		return
	}

	if parts[0] == "openapi.json" {
		s.handleOpenAPIJSON(w, r)
		return
	}

	// Handle /api/_proxy (already routed via mux, but skip model check)
	if parts[0] == "_proxy" {
		return
	}

	// Handle scripting endpoints routed via mux
	if parts[0] == "_eval" || parts[0] == "_log" {
		return
	}

	modelo := parts[0]
	if modelo == "" {
		s.jsonError(w, "modelo não especificado", http.StatusBadRequest)
		return
	}

	if _, ok := s.DB.Models[modelo]; !ok {
		s.jsonError(w, fmt.Sprintf("modelo '%s' não existe", modelo), http.StatusNotFound)
		return
	}

	// Role-based access control for write operations
	if r.Method != http.MethodGet && s.Auth != nil {
		for _, screen := range s.Program.Screens {
			if strings.ToLower(screen.Name) == modelo || screenMatchesModel(screen, modelo) {
				if screen.Requires != "" && !s.Auth.CheckRole(r, screen.Requires) {
					s.jsonError(w, "Permissão negada", http.StatusForbidden)
					return
				}
				break
			}
		}
	}

	// Handle /api/{model}/export/csv and /api/{model}/export/json
	if len(parts) >= 2 && parts[1] == "export" {
		format := "json"
		if len(parts) >= 3 {
			format = parts[2]
		}
		s.handleExport(w, r, modelo, format)
		return
	}

	// Handle /api/{model}/{id}/restaurar
	if len(parts) == 3 && parts[2] == "restaurar" {
		id, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			s.jsonError(w, "ID inválido", http.StatusBadRequest)
			return
		}
		s.handleRestaurar(w, r, modelo, id)
		return
	}

	// Handle /api/{model}/{id}/{relation} - relationship expansion
	if len(parts) == 3 && parts[2] != "restaurar" && parts[2] != "export" {
		id, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			s.jsonError(w, "ID inválido", http.StatusBadRequest)
			return
		}
		relacao := parts[2]
		items, err := s.DB.BuscarRelacionados(modelo, id, relacao)
		if err != nil {
			s.jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.jsonOK(w, items)
		return
	}

	if len(parts) >= 2 && parts[1] != "" {
		id, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			s.jsonError(w, "ID inválido", http.StatusBadRequest)
			return
		}
		s.handleAPIComID(w, r, modelo, id)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Parse query params for pagination, filters, search
		q := r.URL.Query()
		pagina, _ := strconv.Atoi(q.Get("pagina"))
		if pagina == 0 {
			pagina, _ = strconv.Atoi(q.Get("page"))
		}
		limite, _ := strconv.Atoi(q.Get("limite"))
		if limite == 0 {
			limite, _ = strconv.Atoi(q.Get("limit"))
		}
		ordenar := q.Get("ordenar")
		if ordenar == "" {
			ordenar = q.Get("sort")
		}
		ordem := q.Get("ordem")
		if ordem == "" {
			ordem = q.Get("order")
		}
		busca := q.Get("busca")
		if busca == "" {
			busca = q.Get("search")
		}

		// Collect field filters
		filtros := make(map[string]string)
		if model, ok := s.DB.Models[modelo]; ok {
			for _, f := range model.Fields {
				fname := strings.ToLower(f.Name)
				if val := q.Get(fname); val != "" {
					filtros[fname] = val
				}
			}
		}

		params := &banco.ListarParams{
			Pagina: pagina, Limite: limite,
			Ordenar: ordenar, Ordem: ordem,
			Busca: busca, Filtros: filtros,
		}

		items, total, err := s.DB.Listar(modelo, params)
		if err != nil {
			s.jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Return with pagination metadata
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Total-Count", fmt.Sprintf("%d", total))
		json.NewEncoder(w).Encode(items)

	case http.MethodPost:
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB max
		body, err := io.ReadAll(r.Body)
		if err != nil {
			s.jsonError(w, "erro ao ler dados", http.StatusBadRequest)
			return
		}
		item, err := s.DB.Criar(modelo, body)
		if err != nil {
			s.jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Broadcast via WebSocket
		if id, ok := item["id"]; ok {
			var idInt int64
			switch v := id.(type) {
			case int64:
				idInt = v
			case float64:
				idInt = int64(v)
			}
			s.WS.Broadcast(WSMessage{Type: "criar", Model: modelo, ID: idInt, Data: item})
		}
		// Trigger WhatsApp notifications
		s.triggerNotifiers("criar", modelo, item)
		w.WriteHeader(http.StatusCreated)
		s.jsonOK(w, item)

	default:
		s.jsonError(w, "método não permitido", http.StatusMethodNotAllowed)
	}
}

func (s *Servidor) handleAPIComID(w http.ResponseWriter, r *http.Request, modelo string, id int64) {
	switch r.Method {
	case http.MethodGet:
		item, err := s.DB.Buscar(modelo, id)
		if err != nil {
			s.jsonError(w, err.Error(), http.StatusNotFound)
			return
		}
		s.jsonOK(w, item)

	case http.MethodPut:
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB max
		body, err := io.ReadAll(r.Body)
		if err != nil {
			s.jsonError(w, "erro ao ler dados", http.StatusBadRequest)
			return
		}
		item, err := s.DB.Atualizar(modelo, id, body)
		if err != nil {
			s.jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.WS.Broadcast(WSMessage{Type: "atualizar", Model: modelo, ID: id, Data: item})
		s.triggerNotifiers("atualizar", modelo, item)
		s.jsonOK(w, item)

	case http.MethodDelete:
		if err := s.DB.Deletar(modelo, id); err != nil {
			s.jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.WS.Broadcast(WSMessage{Type: "deletar", Model: modelo, ID: id})
		w.WriteHeader(http.StatusNoContent)

	default:
		s.jsonError(w, "método não permitido", http.StatusMethodNotAllowed)
	}
}

func (s *Servidor) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		s.jsonError(w, "metodo nao permitido", http.StatusMethodNotAllowed)
		return
	}

	// Limit to 128MB for audio/video attachments
	r.ParseMultipartForm(128 << 20)
	file, header, err := r.FormFile("file")
	if err != nil {
		s.jsonError(w, "erro ao ler arquivo: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Ensure uploads directory exists
	if err := os.MkdirAll("uploads", 0755); err != nil {
		s.jsonError(w, "erro ao criar diretorio de uploads", http.StatusInternalServerError)
		return
	}

	// Generate unique filename
	ext := filepath.Ext(header.Filename)

	// Whitelist allowed extensions
	allowedExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
		".svg": true, ".pdf": true, ".doc": true, ".docx": true, ".xls": true,
		".xlsx": true, ".csv": true, ".txt": true, ".mp4": true, ".mp3": true,
		".wav": true, ".ogg": true, ".webm": true, ".m4a": true, ".mov": true,
		".avi": true, ".aac": true,
	}
	if !allowedExts[strings.ToLower(ext)] {
		s.jsonError(w, "Tipo de arquivo não permitido", http.StatusBadRequest)
		return
	}

	name := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	destPath := filepath.Join("uploads", name)

	dst, err := os.Create(destPath)
	if err != nil {
		s.jsonError(w, "erro ao salvar arquivo", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		s.jsonError(w, "erro ao escrever arquivo", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"path": "/uploads/" + name, "name": header.Filename})
}

func (s *Servidor) handleMediaStream(w http.ResponseWriter, r *http.Request) {
	pathValue := r.URL.Query().Get("path")
	if pathValue == "" {
		s.jsonError(w, "path é obrigatório", http.StatusBadRequest)
		return
	}
	fullPath, err := resolveUploadPath(pathValue)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	file, err := os.Open(fullPath)
	if err != nil {
		s.jsonError(w, "arquivo não encontrado", http.StatusNotFound)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		s.jsonError(w, "erro ao ler arquivo", http.StatusInternalServerError)
		return
	}
	ctype := mime.TypeByExtension(strings.ToLower(filepath.Ext(fullPath)))
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Disposition", "inline; filename=\""+filepath.Base(fullPath)+"\"")
	http.ServeContent(w, r, filepath.Base(fullPath), info.ModTime(), file)
}

func (s *Servidor) jsonOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *Servidor) jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"erro": msg})
}

func resolveUploadPath(pathValue string) (string, error) {
	clean := filepath.Clean(strings.TrimPrefix(pathValue, "/"))
	if !strings.HasPrefix(clean, "uploads") {
		return "", fmt.Errorf("path fora do diretório de uploads")
	}
	fullPath, err := filepath.Abs(clean)
	if err != nil {
		return "", fmt.Errorf("path inválido")
	}
	baseUploads, _ := filepath.Abs("uploads")
	if !strings.HasPrefix(fullPath, baseUploads) {
		return "", fmt.Errorf("path bloqueado por segurança")
	}
	return fullPath, nil
}

// triggerNotifiers checks and fires WhatsApp/email/other notifications.
func (s *Servidor) triggerNotifiers(triggerType string, modelo string, data map[string]any) {
	for _, notif := range s.Program.Notifiers {
		// Match trigger type and model
		match := false
		switch {
		case notif.Trigger == triggerType && notif.Model == modelo:
			match = true
		case notif.Trigger == triggerType && notif.Model == "":
			match = true
		case notif.Field != "" && notif.Value != "":
			// Conditional: when field equals value
			if val, ok := data[notif.Field]; ok {
				if fmt.Sprintf("%v", val) == notif.Value {
					match = true
				}
			}
		}

		if !match {
			continue
		}

		// Resolve destination
		dest := resolveField(notif.SendTo, data)
		if dest == "" {
			continue
		}

		// Resolve message template (replace {field} with values)
		msg := resolveTemplate(notif.Message, data)

		// Send via WhatsApp
		if notif.Channel == "whatsapp" && s.WA != nil {
			s.Jobs.Submit("whatsapp-notifier", func() {
				if err := s.WA.EnviarMensagem(dest, msg); err != nil {
					fmt.Printf("[whatsapp] Erro ao enviar: %s\n", err)
				}
			})
		}

		// Send via Email
		if notif.Channel == "email" && s.Email != nil {
			subject := resolveTemplate(notif.Subject, data)
			if subject == "" {
				subject = "Notificação"
			}
			s.Jobs.Submit("email-notifier", func() {
				if err := s.Email.EnviarEmail(dest, subject, msg); err != nil {
					fmt.Printf("[email] Erro ao enviar: %s\n", err)
				}
			})
		}
	}
}

func (s *Servidor) handlePresence(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.presenceMu.RLock()
		items := make([]map[string]any, 0, len(s.presence))
		for _, item := range s.presence {
			items = append(items, item)
		}
		s.presenceMu.RUnlock()
		s.jsonOK(w, items)
		return
	}
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		s.jsonError(w, "método não permitido", http.StatusMethodNotAllowed)
		return
	}
	var req map[string]any
	if r.Method == http.MethodPost {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			s.jsonError(w, "erro ao ler requisição", http.StatusBadRequest)
			return
		}
		if err := json.Unmarshal(body, &req); err != nil {
			s.jsonError(w, "JSON inválido", http.StatusBadRequest)
			return
		}
		userKey := fmt.Sprintf("%v", req["user"])
		if userKey == "" {
			userKey = r.RemoteAddr
		}
		req["updated_at"] = time.Now().Format(time.RFC3339)
		s.presenceMu.Lock()
		s.presence[userKey] = req
		s.presenceMu.Unlock()
		msgType := "presenca"
		if typing, ok := req["typing"].(bool); ok && typing {
			msgType = "digitando"
		}
		s.WS.Broadcast(WSMessage{Type: msgType, Session: fmt.Sprintf("%v", req["session"]), Data: req})
		s.jsonOK(w, map[string]any{"ok": true})
		return
	}
	userKey := r.URL.Query().Get("user")
	if userKey == "" {
		userKey = r.RemoteAddr
	}
	s.presenceMu.Lock()
	delete(s.presence, userKey)
	s.presenceMu.Unlock()
	s.WS.Broadcast(WSMessage{Type: "presenca", Data: map[string]any{"user": userKey, "status": "offline"}})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Servidor) handleJobsStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.jsonError(w, "método não permitido", http.StatusMethodNotAllowed)
		return
	}
	s.jsonOK(w, s.Jobs.Stats())
}

func (s *Servidor) handleWASessions(w http.ResponseWriter, r *http.Request) {
	if s.WA == nil {
		s.jsonError(w, "whatsapp não configurado", http.StatusNotFound)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.jsonOK(w, s.WA.ListarSessoes())
	case http.MethodPost:
		var req struct {
			Session string `json:"session"`
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)
		if req.Session == "" {
			req.Session = "default"
		}
		s.Jobs.Submit("wa-connect-"+req.Session, func() {
			if err := s.WA.ConectarSessao(req.Session); err != nil {
				fmt.Printf("[whatsapp] erro ao conectar sessão %s: %v\n", req.Session, err)
			}
		})
		w.WriteHeader(http.StatusAccepted)
		s.jsonOK(w, map[string]any{"ok": true, "session": req.Session})
	default:
		s.jsonError(w, "método não permitido", http.StatusMethodNotAllowed)
	}
}

func (s *Servidor) handleWAConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.jsonError(w, "método não permitido", http.StatusMethodNotAllowed)
		return
	}
	s.handleWASessions(w, r)
}

func (s *Servidor) handleWAQR(w http.ResponseWriter, r *http.Request) {
	if s.WA == nil {
		s.jsonError(w, "whatsapp não configurado", http.StatusNotFound)
		return
	}
	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		sessionID = "default"
	}
	s.jsonOK(w, s.WA.SessionInfo(sessionID))
}

func (s *Servidor) handleWASend(w http.ResponseWriter, r *http.Request) {
	if s.WA == nil {
		s.jsonError(w, "whatsapp não configurado", http.StatusNotFound)
		return
	}
	if r.Method != http.MethodPost {
		s.jsonError(w, "método não permitido", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Session string `json:"session"`
		Phone   string `json:"phone"`
		Message string `json:"message"`
	}
	body, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(body, &req); err != nil {
		s.jsonError(w, "JSON inválido", http.StatusBadRequest)
		return
	}
	if req.Session == "" {
		req.Session = "default"
	}
	if err := s.WA.EnviarMensagemSessao(req.Session, req.Phone, req.Message); err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.jsonOK(w, map[string]any{"ok": true})
}

// handleProxy allows the frontend to call external APIs through the server.
// POST /api/_proxy
// Body: {"method": "GET", "url": "https://...", "body": "..."}
func (s *Servidor) handleProxy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		s.jsonError(w, "método não permitido", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Method string `json:"method"`
		URL    string `json:"url"`
		Body   string `json:"body"`
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.jsonError(w, "erro ao ler requisição", http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(body, &req); err != nil {
		s.jsonError(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		s.jsonError(w, "URL é obrigatória", http.StatusBadRequest)
		return
	}

	// SSRF protection: block private/internal URLs
	parsedURL, err := url.Parse(req.URL)
	if err != nil {
		s.jsonError(w, "URL inválida", http.StatusBadRequest)
		return
	}
	host := parsedURL.Hostname()
	// Block private IP ranges and dangerous hosts
	blockedPrefixes := []string{"127.", "10.", "192.168.", "172.16.", "172.17.", "172.18.", "172.19.", "172.20.", "172.21.", "172.22.", "172.23.", "172.24.", "172.25.", "172.26.", "172.27.", "172.28.", "172.29.", "172.30.", "172.31.", "169.254.", "0."}
	for _, prefix := range blockedPrefixes {
		if strings.HasPrefix(host, prefix) {
			s.jsonError(w, "URL bloqueada por segurança", http.StatusForbidden)
			return
		}
	}
	if host == "localhost" || host == "" || parsedURL.Scheme == "file" {
		s.jsonError(w, "URL bloqueada por segurança", http.StatusForbidden)
		return
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		s.jsonError(w, "Apenas HTTP/HTTPS permitido", http.StatusBadRequest)
		return
	}

	if req.Method == "" {
		req.Method = "GET"
	}

	if s.HTTPClient == nil {
		s.HTTPClient = httpclient.Novo()
	}

	var reqBody []byte
	if req.Body != "" {
		reqBody = []byte(req.Body)
	}

	resp, err := s.HTTPClient.Chamar(req.Method, req.URL, reqBody)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

// resolveField gets a value from data, supports dotted paths.
func resolveField(field string, data map[string]any) string {
	if field == "" {
		return ""
	}
	// Direct phone number
	if field[0] >= '0' && field[0] <= '9' || field[0] == '+' {
		return field
	}
	// Try direct field
	parts := strings.SplitN(field, ".", 2)
	key := parts[len(parts)-1] // use last part (e.g. "telefone" from "cliente.telefone")
	if val, ok := data[key]; ok {
		return fmt.Sprintf("%v", val)
	}
	if val, ok := data[parts[0]]; ok {
		return fmt.Sprintf("%v", val)
	}
	return ""
}

// resolveTemplate replaces {field} placeholders with actual values.
func resolveTemplate(tmpl string, data map[string]any) string {
	result := tmpl
	for key, val := range data {
		result = strings.ReplaceAll(result, "{"+key+"}", fmt.Sprintf("%v", val))
	}
	return result
}

// handleStats returns record counts and status breakdowns per model.
func (s *Servidor) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.jsonError(w, "método não permitido", http.StatusMethodNotAllowed)
		return
	}

	type modelStats struct {
		Count    int64            `json:"count"`
		Statuses map[string]int64 `json:"statuses,omitempty"`
	}

	result := make(map[string]modelStats)
	for name := range s.DB.Models {
		count, _ := s.DB.Contar(name)
		ms := modelStats{Count: count}
		statuses, err := s.DB.ContarPorStatus(name)
		if err == nil && len(statuses) > 0 {
			ms.Statuses = statuses
		}
		result[name] = ms
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleExport exports all records as CSV or JSON.
func (s *Servidor) handleExport(w http.ResponseWriter, r *http.Request, modelo string, format string) {
	if r.Method != http.MethodGet {
		s.jsonError(w, "método não permitido", http.StatusMethodNotAllowed)
		return
	}

	items, err := s.DB.ListarTodos(modelo)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	model := s.DB.Models[modelo]
	timestamp := time.Now().Format("2006-01-02")

	switch format {
	case "csv":
		filename := fmt.Sprintf("%s_%s.csv", modelo, timestamp)
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

		writer := csv.NewWriter(w)
		// Write BOM for Excel UTF-8 compatibility
		w.Write([]byte{0xEF, 0xBB, 0xBF})

		// Header row
		headers := []string{"id"}
		for _, f := range model.Fields {
			headers = append(headers, strings.ToLower(f.Name))
		}
		headers = append(headers, "criado_em", "atualizado_em")
		writer.Write(headers)

		// Data rows
		for _, item := range items {
			var row []string
			for _, h := range headers {
				val := ""
				if v, ok := item[h]; ok && v != nil {
					val = fmt.Sprintf("%v", v)
					// Prevent CSV formula injection
					if len(val) > 0 && (val[0] == '=' || val[0] == '+' || val[0] == '-' || val[0] == '@') {
						val = "'" + val
					}
				}
				row = append(row, val)
			}
			writer.Write(row)
		}
		writer.Flush()

	default: // json
		filename := fmt.Sprintf("%s_%s.json", modelo, timestamp)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		json.NewEncoder(w).Encode(items)
	}
}

// screenMatchesModel checks if a screen references the given model via a list component.
func screenMatchesModel(screen *ast.Screen, modelo string) bool {
	for _, comp := range screen.Components {
		if comp.Type == ast.CompList && strings.ToLower(comp.Target) == modelo {
			return true
		}
	}
	return false
}

// handleRestaurar restores a soft-deleted record.
func (s *Servidor) handleOpenAPISwagger(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Germanio API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js" crossorigin></script>
  <script>
    window.onload = () => {
      window.ui = SwaggerUIBundle({
        url: '/api/openapi.json',
        dom_id: '#swagger-ui',
      });
    };
  </script>
</body>
</html>`
	w.Write([]byte(html))
}

func (s *Servidor) handleOpenAPIJSON(w http.ResponseWriter, r *http.Request) {
	paths := make(map[string]any)

	// Add CRUD models
	for _, m := range s.Program.Models {
		mName := strings.ToLower(m.Name)
		paths["/api/"+mName] = map[string]any{
			"get": map[string]any{
				"summary":     "Listar " + m.Name,
				"tags":        []string{m.Name},
				"responses":   map[string]any{"200": map[string]any{"description": "Lista de " + m.Name}},
			},
			"post": map[string]any{
				"summary":     "Criar " + m.Name,
				"tags":        []string{m.Name},
				"responses":   map[string]any{"201": map[string]any{"description": m.Name + " criado"}},
			},
		}
		paths["/api/"+mName+"/{id}"] = map[string]any{
			"get": map[string]any{
				"summary":     "Obter " + m.Name + " por ID",
				"tags":        []string{m.Name},
				"responses":   map[string]any{"200": map[string]any{"description": "Detalhes de " + m.Name}},
			},
			"put": map[string]any{
				"summary":     "Atualizar " + m.Name,
				"tags":        []string{m.Name},
				"responses":   map[string]any{"200": map[string]any{"description": m.Name + " atualizado"}},
			},
			"delete": map[string]any{
				"summary":     "Deletar " + m.Name,
				"tags":        []string{m.Name},
				"responses":   map[string]any{"200": map[string]any{"description": m.Name + " removido"}},
			},
		}
	}

	// Add Prompt events
	for _, evt := range s.Program.Events {
		if evt.Trigger == "receber" || evt.Trigger == "chamar" {
			actionName := strings.TrimSpace(evt.Target)
			if actionName == "" {
				actionName = strings.TrimSpace(evt.ActionRef)
			}
			parts := strings.Fields(actionName)
			if len(parts) > 0 {
				slug := strings.ToLower(parts[0])
				paths["/api/"+slug] = map[string]any{
					"post": map[string]any{
						"summary":   "Ação Prompt: " + slug,
						"tags":      []string{"Ações"},
						"responses": map[string]any{"200": map[string]any{"description": "Ação executada com sucesso"}},
					},
				}
			}
		}
	}

	doc := map[string]any{
		"openapi": "3.0.0",
		"info": map[string]any{
			"title":       s.Program.System.Name + " API",
			"version":     "1.0.0",
			"description": "API REST gerada automaticamente pelo Germanio",
		},
		"paths": paths,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func (s *Servidor) handleRestaurar(w http.ResponseWriter, r *http.Request, modelo string, id int64) {
	if r.Method != http.MethodPut {
		s.jsonError(w, "método não permitido", http.StatusMethodNotAllowed)
		return
	}

	item, err := s.DB.Restaurar(modelo, id)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.WS.Broadcast(WSMessage{Type: "restaurar", Model: modelo, ID: id, Data: item})
	s.jsonOK(w, item)
}
