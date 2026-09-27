package runtime

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/lexer"
	"github.com/flaviokalleu/germanio/compiler/parser"
	authpkg "github.com/flaviokalleu/germanio/runtime/auth"
	"github.com/flaviokalleu/germanio/runtime/banco"
	cronpkg "github.com/flaviokalleu/germanio/runtime/cron"
	emailpkg "github.com/flaviokalleu/germanio/runtime/email"
	"github.com/flaviokalleu/germanio/runtime/httpclient"
	interp "github.com/flaviokalleu/germanio/runtime/interpreter"
	"github.com/flaviokalleu/germanio/runtime/servidor"
	wa "github.com/flaviokalleu/germanio/runtime/whatsapp"
)

// parseFG reads and parses a single .ge file.
func parseFG(arquivo string) (*ast.Program, error) {
	source, err := os.ReadFile(arquivo)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler %s: %w", arquivo, err)
	}

	lex := lexer.New(string(source))
	tokens, err := lex.Tokenize()
	if err != nil {
		return nil, fmt.Errorf("erro léxico em %s: %w", arquivo, err)
	}

	p := parser.New(tokens)
	p.File = arquivo
	program, err := p.Parse()
	if err != nil {
		return nil, fmt.Errorf("erro de parsing em %s: %w", arquivo, err)
	}

	return program, nil
}

// resolveImports processes all import statements recursively.
func resolveImports(program *ast.Program, baseDir string, resolved map[string]bool) error {
	if resolved == nil {
		resolved = make(map[string]bool)
	}

	for _, imp := range program.Imports {
		// Resolve path relative to the base .ge file
		importPath := filepath.Join(baseDir, imp.Path)
		absPath, err := filepath.Abs(importPath)
		if err != nil {
			return fmt.Errorf("caminho inválido: %s", imp.Path)
		}

		// Security: prevent path traversal outside project directory
		absBase, _ := filepath.Abs(baseDir)
		if !strings.HasPrefix(absPath, absBase) {
			return fmt.Errorf("importação bloqueada: '%s' fora do diretório do projeto", imp.Path)
		}

		// Avoid circular imports
		if resolved[absPath] {
			continue
		}
		resolved[absPath] = true

		fmt.Printf("[germanio] Importando: %s\n", imp.Path)

		imported, err := parseFG(absPath)
		if err != nil {
			return fmt.Errorf("erro ao importar %s: %w", imp.Path, err)
		}

		// Recursively resolve imports in the imported file
		importDir := filepath.Dir(absPath)
		if err := resolveImports(imported, importDir, resolved); err != nil {
			return err
		}

		// Merge based on what was requested
		switch imp.What {
		case "tudo":
			program.Merge(imported)
		case "dados":
			program.Models = append(program.Models, imported.Models...)
		case "telas":
			program.Screens = append(program.Screens, imported.Screens...)
		case "eventos":
			program.Events = append(program.Events, imported.Events...)
		case "tema":
			if imported.Theme != nil {
				program.Theme = imported.Theme
			}
		case "logica":
			program.Rules = append(program.Rules, imported.Rules...)
		default:
			// Import specific named items (e.g., importar produto de "dados.ge")
			for _, m := range imported.Models {
				if m.Name == imp.What {
					program.Models = append(program.Models, m)
				}
			}
			for _, s := range imported.Screens {
				if s.Name == imp.What {
					program.Screens = append(program.Screens, s)
				}
			}
		}
	}

	return nil
}

// Executar loads a .ge file and runs the application.
// App is a loaded Germanio application ready to serve.
type App struct {
	Program     *ast.Program
	DB          *banco.Banco
	Interpreter *interp.Interpreter
	Server      *servidor.Servidor
	Handler     http.Handler
	// closers run on Fechar (background workers, WhatsApp, cron).
	closers []func()
}

// Fechar stops background work and closes the database.
func (a *App) Fechar() {
	for i := len(a.closers) - 1; i >= 0; i-- {
		a.closers[i]()
	}
	if a.DB != nil {
		a.DB.Fechar()
	}
}

// Capabilities registered by other runtime packages (git, processo, ui…).
// Each one receives the app after the interpreter exists and before the
// startup scripts run.
var capabilityHooks []func(*App) error

// RegistrarCapability adds a hook that exposes a capability to .ge code.
func RegistrarCapability(hook func(*App) error) {
	capabilityHooks = append(capabilityHooks, hook)
}

// OnClose registers cleanup for a capability.
func (a *App) OnClose(f func()) { a.closers = append(a.closers, f) }

// Carregar parses, resolves imports, opens the database, wires the
// interpreter and capabilities and builds the HTTP handler.
func Carregar(arquivo string, porta string) (*App, error) {
	envPath := filepath.Join(filepath.Dir(arquivo), ".env")
	LoadEnv(envPath)

	program, err := parseFG(arquivo)
	if err != nil {
		return nil, err
	}

	baseDir := filepath.Dir(arquivo)
	if err := resolveImports(program, baseDir, nil); err != nil {
		return nil, err
	}

	if program.System == nil {
		return nil, fmt.Errorf("declaração 'sistema' não encontrada")
	}

	fmt.Printf("[germanio] Sistema: %s\n", program.System.Name)
	fmt.Printf("[germanio] Modelos: %d | Telas: %d | Rotas: %d | Funções: %d\n",
		len(program.Models), len(program.Screens), len(program.Routes), len(program.Functions))

	db, err := banco.Abrir(program.Database, program.System.Name, program.Models)
	if err != nil {
		return nil, fmt.Errorf("erro no banco: %w", err)
	}
	db.Rules = program.Rules
	app := &App{Program: program, DB: db}

	var authHandler *authpkg.Auth
	if program.Auth != nil && program.Auth.Enabled {
		jwtSecret := program.Auth.JWTSecret
		if envSecret := GetEnv("JWT_SECRET", ""); envSecret != "" {
			jwtSecret = envSecret
		}
		if jwtSecret == "germanio-secret-change-me" {
			jwtSecret = fmt.Sprintf("germanio-%d-%s", time.Now().UnixNano(), program.System.Name)
			fmt.Println("[germanio] AVISO: JWT secret gerado automaticamente. Defina JWT_SECRET no .env para produção.")
		}
		authHandler = authpkg.Novo(
			db.DB, program.Auth.UserModel, program.Auth.LoginField,
			program.Auth.PassField, jwtSecret,
		)
		authHandler.SetupTable()
		if len(program.Auth.Roles) > 0 {
			authHandler.Roles = program.Auth.Roles
		}
		fmt.Println("[germanio] Auth: ativado")
	}

	var waClient *wa.Client
	if program.WhatsApp != nil && program.WhatsApp.Enabled {
		waClient = wa.Novo(program.WhatsApp.DBPath)
		if waClient != nil {
			go func() {
				if err := waClient.Conectar(); err != nil {
					fmt.Printf("[germanio] AVISO WhatsApp: %s (continuando sem WhatsApp)\n", err)
				}
			}()
			app.OnClose(waClient.Desconectar)
		}
	}

	var emailClient *emailpkg.Client
	if program.Email != nil && program.Email.Host != "" {
		emailClient = emailpkg.Novo(emailpkg.Config{
			Host:     program.Email.Host,
			Port:     program.Email.Port,
			User:     program.Email.User,
			Password: program.Email.Password,
			From:     program.Email.From,
		})
		fmt.Println("[germanio] Email SMTP: ativado")
	}

	httpClient := httpclient.Novo()

	srv := servidor.Novo(program, db, porta)
	srv.Auth = authHandler
	srv.WA = waClient
	srv.Email = emailClient
	srv.HTTPClient = httpClient

	interpreter := interp.New(db)
	interpreter.HTTPClient = httpClient
	if waClient != nil {
		interpreter.WAClient = waClient
	}
	srv.Interpreter = interpreter
	app.Interpreter = interpreter
	app.Server = srv

	for _, hook := range capabilityHooks {
		if err := hook(app); err != nil {
			app.Fechar()
			return nil, err
		}
	}

	if len(program.Functions) > 0 || len(program.Scripts) > 0 {
		interpreter.Run(program)
	}

	if len(program.Crons) > 0 {
		scheduler := cronpkg.Novo(program.Crons)
		scheduler.Iniciar()
		app.OnClose(scheduler.Parar)
		fmt.Printf("[germanio] Cron: %d job(s) agendado(s)\n", len(program.Crons))
	}

	handler, err := srv.Handler()
	if err != nil {
		app.Fechar()
		return nil, err
	}
	app.Handler = handler
	return app, nil
}

func Executar(arquivo string, porta string) error {
	if envPort := GetEnv("PORT", ""); envPort != "" && porta == "8080" {
		porta = envPort
	}
	fmt.Printf("[germanio] Carregando: %s\n", arquivo)
	app, err := Carregar(arquivo, porta)
	if err != nil {
		return err
	}
	defer app.Fechar()

	fmt.Printf("\n[germanio] %s rodando em http://localhost:%s\n\n", app.Program.System.Name, porta)
	WatchFiles(filepath.Dir(arquivo), arquivo, porta)

	server := &http.Server{
		Addr:           ":" + porta,
		Handler:        app.Handler,
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   10 * time.Minute, // git clone/push and job logs stream for a while
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	return server.ListenAndServe()
}

func Verificar(arquivo string) error {
	program, err := parseFG(arquivo)
	if err != nil {
		return err
	}

	baseDir := filepath.Dir(arquivo)
	if err := resolveImports(program, baseDir, nil); err != nil {
		return err
	}

	if program.System == nil {
		return fmt.Errorf("declaração 'sistema' não encontrada")
	}

	fmt.Printf("[germanio] ✓ %s - válido\n", arquivo)
	fmt.Printf("  sistema:  %s\n", program.System.Name)
	fmt.Printf("  imports:  %d\n", len(program.Imports))
	fmt.Printf("  modelos:  %d\n", len(program.Models))
	fmt.Printf("  telas:    %d\n", len(program.Screens))
	fmt.Printf("  eventos:  %d\n", len(program.Events))
	fmt.Printf("  regras:   %d\n", len(program.Rules))
	if len(program.Functions) > 0 {
		fmt.Printf("  funcoes:  %d\n", len(program.Functions))
	}
	if len(program.Scripts) > 0 {
		fmt.Printf("  scripts:  %d\n", len(program.Scripts))
	}
	if program.WhatsApp != nil && program.WhatsApp.Enabled {
		fmt.Printf("  whatsapp: ativado (%d notificações)\n", len(program.Notifiers))
	}
	if program.Email != nil && program.Email.Host != "" {
		emailNotifs := 0
		for _, n := range program.Notifiers {
			if n.Channel == "email" {
				emailNotifs++
			}
		}
		fmt.Printf("  email:    ativado (%d notificações)\n", emailNotifs)
	}
	if len(program.Crons) > 0 {
		fmt.Printf("  cron:     %d job(s)\n", len(program.Crons))
	}
	return nil
}
