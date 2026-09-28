package runtime

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
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

// resolveImports processes all import statements recursively. The folder
// of the entry file is the project: imports never leave it, and the folders
// backend/ and frontend/ directly inside it keep their roles (checkRole).
func resolveImports(program *ast.Program, entry string, resolved map[string]bool) error {
	entry, err := filepath.Abs(entry)
	if err != nil {
		return err
	}
	root := filepath.Dir(entry)
	if resolved == nil {
		resolved = map[string]bool{}
	}
	l := &loader{root: root, resolved: resolved, declares: map[string][]string{}}
	_, err = l.importsFrom(program, entry)
	return err
}

type loader struct {
	root     string
	resolved map[string]bool
	declares map[string][]string // file → data it declares (tenha …)
}

// importsFrom loads the imports of one program and returns the data it
// named (importar produtos e pedidos do backend).
func (l *loader) importsFrom(program *ast.Program, from string) (map[string]bool, error) {
	uses := map[string]bool{}
	dir := filepath.Dir(from)
	for _, imp := range program.Imports {
		base := dir
		if imp.FromRoot {
			base = l.root
		}
		absPath, err := filepath.Abs(filepath.Join(base, imp.Path))
		if err != nil {
			return nil, fmt.Errorf("caminho inválido: %s", imp.Path)
		}
		// Security: imports never leave the project folder.
		if absPath != l.root && !strings.HasPrefix(absPath, l.root+string(filepath.Separator)) {
			return nil, fmt.Errorf("importação bloqueada: '%s' fora do diretório do projeto", imp.Path)
		}
		if imp.FromRoot {
			if _, err := os.Stat(absPath); err != nil {
				if _, err2 := os.Stat(absPath + ".ge"); err2 == nil {
					absPath += ".ge"
				} else {
					return nil, fmt.Errorf("%s: importar %s do %s: não existe a pasta %s/", l.rel(from), strings.Join(imp.Names, ", "), imp.Path, imp.Path)
				}
			}
		}
		files := []string{absPath}
		if info, err := os.Stat(absPath); err == nil && info.IsDir() {
			// importar "backend": every .ge inside, in alphabetical order.
			if imp.What != "tudo" {
				return nil, fmt.Errorf("importar %s de \"%s\": pastas são importadas inteiras (use: importar \"%s\")", imp.What, imp.Path, imp.Path)
			}
			files = nil
			filepath.WalkDir(absPath, func(path string, d os.DirEntry, err error) error {
				if err == nil && !d.IsDir() && filepath.Ext(path) == ".ge" {
					files = append(files, path)
				}
				return nil
			})
			if len(files) == 0 {
				return nil, fmt.Errorf("importar \"%s\": a pasta não tem arquivos .ge", imp.Path)
			}
		}
		for _, file := range files {
			if l.resolved[file] {
				continue // already imported (or circular)
			}
			l.resolved[file] = true
			rel := l.rel(file)
			fmt.Printf("[germanio] Importando: %s\n", rel)
			imported, err := parseFG(file)
			if err != nil {
				return nil, fmt.Errorf("erro ao importar %s: %w", rel, err)
			}
			if in := imported.Intent; in != nil {
				for _, e := range in.Entities {
					l.declares[file] = append(l.declares[file], e.Name)
				}
			}
			own, err := l.importsFrom(imported, file)
			if err != nil {
				return nil, err
			}
			if err := checkRole(rel, imported, own); err != nil {
				return nil, err
			}
			mergeImport(program, imported, imp.What)
		}
		if len(imp.Names) > 0 {
			offered := map[string]bool{}
			for _, f := range files {
				for _, n := range l.declares[f] {
					offered[dataKey(n)] = true
				}
			}
			for _, n := range imp.Names {
				if !offered[dataKey(n)] {
					return nil, fmt.Errorf("%s: importar %s do %s: %s não tem %s%s", l.rel(from), n, imp.Path, imp.Path, n, listOffered(files, l))
				}
				uses[dataKey(n)] = true
			}
		}
	}
	return uses, nil
}

func (l *loader) rel(file string) string {
	r, err := filepath.Rel(l.root, file)
	if err != nil {
		return file
	}
	return filepath.ToSlash(r)
}

// dataKey compares data names as people write them: plural or singular,
// with or without accents.
func dataKey(name string) string {
	n := strings.ToLower(strings.NewReplacer("á", "a", "à", "a", "â", "a", "ã", "a", "é", "e", "ê", "e", "í", "i", "ó", "o", "ô", "o", "õ", "o", "ú", "u", "ç", "c", " ", "_").Replace(name))
	return parser.Singular(n)
}

func listOffered(files []string, l *loader) string {
	var names []string
	for _, f := range files {
		names = append(names, l.declares[f]...)
	}
	if len(names) == 0 {
		return ""
	}
	sort.Strings(names)
	return " (tem: " + strings.Join(names, ", ") + ")"
}

// checkRole keeps the two sides of an organized project honest:
// frontend/ describes what appears (pages, menu, theme); backend/ describes
// what exists, who may do what and what happens — never screens.
func checkRole(rel string, p *ast.Program, uses map[string]bool) error {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) < 2 {
		return nil
	}
	file := filepath.ToSlash(rel)
	in := p.Intent
	switch parts[0] {
	case "frontend":
		what := ""
		switch {
		case in != nil && (len(in.Entities) > 0 || len(in.FieldBlocks) > 0 || len(in.Relations) > 0 || len(in.States) > 0):
			what = "o que existe (tenha, tem, pertence, começa)"
		case in != nil && (len(in.Grants) > 0 || len(in.Permits) > 0 || len(in.Roles) > 0 || len(in.Visibility) > 0 || len(in.Creators) > 0):
			what = "quem pode fazer o quê (pode, permita, papéis)"
		case in != nil && (len(in.Hooks) > 0 || len(in.Init) > 0 || len(in.Executions) > 0 || len(in.Subscriptions) > 0):
			what = "o que acontece (quando, antes de, ao iniciar)"
		case in != nil && (len(in.Integrations) > 0 || in.Login != nil || len(in.Vocabulary) > 0 || in.IntegrationPrefix != ""):
			what = "login ou integração"
		case in != nil && len(in.Translators) > 0:
			what = "traduza (adaptador)"
		case len(p.Models) > 0 || len(p.Functions) > 0 || len(p.Rules) > 0 || len(p.Routes) > 0 || len(p.Crons) > 0 || p.Database != nil || p.Auth != nil:
			what = "dados, lógica ou rotas"
		}
		if what != "" {
			if what == "traduza (adaptador)" {
				return fmt.Errorf("%s: traduza é trabalho de adaptador e vai em integracoes/", file)
			}
			return fmt.Errorf("%s: frontend/ mostra o que aparece (páginas, menu, tema); %s vai em backend/", file, what)
		}
		// Each page file says what it uses from the backend.
		if in != nil {
			for _, pg := range in.Pages {
				target := pg.Show
				if target == "" {
					target = pg.Manage
				}
				if target != "" && !uses[dataKey(target)] {
					return fmt.Errorf("%s:%d: a página %s mostra %s, mas o arquivo não importa — escreva no início: importar %s do backend", file, pg.Pos.Line, pg.Name, target, target)
				}
			}
		}
	case "integracoes":
		// Adapters translate an external protocol; they never declare the product.
		if in != nil && (len(in.Entities) > 0 || len(in.FieldBlocks) > 0 || len(in.Relations) > 0 || len(in.Grants) > 0 || len(in.Permits) > 0 ||
			len(in.Roles) > 0 || len(in.States) > 0 || len(in.Hooks) > 0 || len(in.Pages) > 0 || len(in.RemoteExecutors) > 0 || in.Login != nil) ||
			len(p.Models) > 0 || len(p.Screens) > 0 || len(p.Pages) > 0 {
			return fmt.Errorf("%s: integracoes/ só traduz protocolos externos (rotas e lógica); o produto (dados, permissões, páginas) fica em backend/ e frontend/", file)
		}
	case "backend":
		if (in != nil && len(in.Pages) > 0) || len(p.Screens) > 0 || len(p.Pages) > 0 || len(p.SidebarItems) > 0 || p.Theme != nil {
			return fmt.Errorf("%s: backend/ descreve o que existe e as regras; páginas, menu e tema vão em frontend/", file)
		}
		if in != nil && len(in.Translators) > 0 {
			return fmt.Errorf("%s: traduza é trabalho de adaptador e vai em integracoes/", file)
		}
	}
	return nil
}

func mergeImport(program, imported *ast.Program, what string) {
	switch what {
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
			if m.Name == what {
				program.Models = append(program.Models, m)
			}
		}
		for _, s := range imported.Screens {
			if s.Name == what {
				program.Screens = append(program.Screens, s)
			}
		}
	}
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

	if err := resolveImports(program, arquivo, nil); err != nil {
		return nil, err
	}

	if err := parser.ResolveIntent(program); err != nil {
		return nil, err
	}
	if program.System == nil {
		return nil, fmt.Errorf("declaração 'sistema' não encontrada (use: crie sistema Nome)")
	}
	if err := checarDuplicatas(program); err != nil {
		return nil, err
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
	if dir := filepath.Join(filepath.Dir(arquivo), "assets"); dirExists(dir) {
		srv.AssetsDir = dir // the project's assets, wherever it is run from
	}
	srv.Auth = authHandler
	srv.WA = waClient
	srv.Email = emailClient
	srv.HTTPClient = httpClient

	interpreter := interp.New(db)
	interpreter.App = program.App
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

	if program.App != nil {
		program.Scripts = append(program.Scripts, program.App.Init...)
	}
	if len(program.Functions) > 0 || len(program.Scripts) > 0 {
		interpreter.Run(program)
	}
	ensureInitialAdmin(interpreter, program)

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
	app.OnClose(srv.Fechar)
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
		Addr:              ":" + porta,
		Handler:           app.Handler,
		ReadHeaderTimeout: 10 * time.Second, // a slow client cannot hold a connection just sending headers
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      10 * time.Minute, // git clone/push and job logs stream for a while
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	return server.ListenAndServe()
}

func Verificar(arquivo string) error {
	program, err := parseFG(arquivo)
	if err != nil {
		return err
	}

	if err := resolveImports(program, arquivo, nil); err != nil {
		return err
	}
	if err := parser.ResolveIntent(program); err != nil {
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

// checarDuplicatas rejects two functions or two models with the same name
// across all imported files; the later one used to replace the earlier
// silently.
func checarDuplicatas(program *ast.Program) error {
	fns := map[string]*ast.FuncDecl{}
	for _, f := range program.Functions {
		if prev, ok := fns[f.Name]; ok {
			return fmt.Errorf("%s:%d: função '%s' já definida em %s:%d", f.Pos.File, f.Pos.Line, f.Name, prev.Pos.File, prev.Pos.Line)
		}
		fns[f.Name] = f
	}
	models := map[string]bool{}
	for _, m := range program.Models {
		name := strings.ToLower(m.Name)
		if models[name] {
			return fmt.Errorf("modelo '%s' definido mais de uma vez", m.Name)
		}
		models[name] = true
	}
	return nil
}

// Compilar parses a program with its imports and resolves the intent layer
// without opening a database or starting anything (ge check / ge explain).
func Compilar(arquivo string) (*ast.Program, error) {
	program, err := parseFG(arquivo)
	if err != nil {
		return nil, err
	}
	if err := resolveImportsQuiet(program, arquivo); err != nil {
		return nil, err
	}
	if err := parser.ResolveIntent(program); err != nil {
		return nil, err
	}
	if err := checarDuplicatas(program); err != nil {
		return nil, err
	}
	return program, nil
}

func resolveImportsQuiet(program *ast.Program, entry string) error {
	stdout := os.Stdout
	devnull, _ := os.Open(os.DevNull)
	os.Stdout = devnull
	defer func() { os.Stdout = stdout; devnull.Close() }()
	return resolveImports(program, entry, nil)
}

// ensureInitialAdmin: `tenha administrador inicial "root"` creates the first
// administrator when nobody exists yet and the server was given its password
// (GERMANIO_ADMIN_SENHA; optional GERMANIO_ADMIN_EMAIL). The password never
// appears in .ge files.
func ensureInitialAdmin(in *interp.Interpreter, program *ast.Program) {
	app := program.App
	if app == nil || app.InitialAdmin == "" {
		return
	}
	ctx := &interp.Context{}
	if n, err := in.Op(ctx, app.LoginEntity, "contar"); err != nil || toFloat(n) > 0 {
		return
	}
	pass := os.Getenv("GERMANIO_ADMIN_SENHA")
	if pass == "" {
		fmt.Printf("[germanio] Administrador inicial %q: defina GERMANIO_ADMIN_SENHA para criá-lo.\n", app.InitialAdmin)
		return
	}
	model := app.Entities[app.LoginEntity].Model
	data := map[string]any{"admin": true}
	loginField := "email"
	if app.Login != nil && len(app.Login.Fields) > 0 {
		loginField = strings.ToLower(app.Login.Fields[0])
	}
	data[loginField] = app.InitialAdmin
	for _, f := range model.Fields {
		name := strings.ToLower(f.Name)
		switch {
		case f.Type == ast.FieldSenha:
			data[name] = pass
		case name == "email" && loginField != "email":
			data[name] = os.Getenv("GERMANIO_ADMIN_EMAIL")
			if data[name] == "" {
				data[name] = "admin@" + strings.ToLower(strings.ReplaceAll(program.System.Name, " ", "")) + ".local"
			}
		case (name == "nome" || name == "name") && data[name] == nil:
			data[name] = map[string]string{"pt": "Administrador", "en": "Administrator"}[app.Messages]
		}
	}
	if _, err := in.Op(ctx, app.LoginEntity, "criar", data); err != nil {
		fmt.Printf("[germanio] Administrador inicial %q não foi criado: %s\n", app.InitialAdmin, interp.Friendly(err))
		return
	}
	fmt.Printf("[germanio] Administrador inicial %q criado.\n", app.InitialAdmin)
}

func toFloat(v any) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
