package gecli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	legacy "github.com/flaviokalleu/germanio/cli"
	"github.com/flaviokalleu/germanio/compiler/diagnostics"
	"github.com/flaviokalleu/germanio/compiler/semantic"
	germanioRuntime "github.com/flaviokalleu/germanio/runtime"
	"github.com/flaviokalleu/germanio/runtime/germanio"
	"github.com/flaviokalleu/germanio/tooling/explicar"
	"github.com/flaviokalleu/germanio/tooling/formatter"
	"github.com/flaviokalleu/germanio/tooling/intelligence"
	"github.com/flaviokalleu/germanio/versao"
)

// Version is the Germanio version (set by releases at link time, see versao).
var Version = versao.Versao

const help = `Germanio — simples para começar, explícito para evoluir.

Uso: ge <comando> [arquivo]

Aplicações
  new <nome>              Cria uma aplicação (app.ge, backend/, frontend/)
  run [app.ge] [porta]    Verifica e executa (também: rodar; porta padrão 8080)
  check [app.ge]          Verifica sem executar nem ler entrada
  explain <dado> [app.ge] Mostra o que o Germanio sabe de um dado e de onde vem cada fato
  fmt [arquivo ou pasta]  Formata arquivos .ge; --check não escreve

Programas do núcleo
  novo <pasta>            Cria um programa de terminal (inicio.ge)
  test [arquivo ou pasta] Executa testes .ge do núcleo (também: testar); --coverage
  explicar <GE0000>       Explica um código de diagnóstico

Project Intelligence (experimental)
  init [subcomando]       Inicia ou expande um projeto guiado (entidade N, pagina N, dashboard)
  graph                   Exibe o Project Knowledge Graph textual
  explain pagina <N>      Explica as conexões de uma página
  eject <componente>      Transfere componente para controle manual

  versao                  Exibe a versão (também: version, --version)
  ajuda                   Exibe esta ajuda

Comandos da CLI anterior: ge legado <comando> [argumentos] (build, docker, ide).
A linguagem .ge é determinística e não utiliza IA no compilador nem na execução.
`

func Run(args []string, in io.Reader, out, stderr io.Writer) int {
	if len(args) == 0 {
		args = []string{"ge"}
	}
	if len(args) == 1 {
		fmt.Fprint(out, help)
		return 0
	}
	cmd := args[1]
	rest := args[2:]
	if strings.HasSuffix(cmd, ".ge") || strings.HasSuffix(cmd, ".ge") {
		rest = append([]string{cmd}, rest...)
		cmd = "rodar"
	}
	fail := func(err error) int { fmt.Fprintln(stderr, err); return 1 }
	switch cmd {
	case "ajuda", "help", "--help", "-h":
		fmt.Fprint(out, help)
		return 0
	case "versao", "version", "--version":
		fmt.Fprintln(out, "Germanio", Version)
		return 0
	case "legado":
		if len(rest) == 0 {
			return fail(fmt.Errorf("Use ge legado ajuda (help no Germanio)"))
		}
		legacy.Run(append([]string{"germanio"}, rest...))
		return 0
	case "init":
		return handleInit(rest, in, out, stderr)
	case "graph":
		cwd, _ := os.Getwd()
		graph, err := intelligence.NewProjectAnalyzer(cwd).Analyze()
		if err != nil {
			return fail(err)
		}
		fmt.Fprint(out, graph.RenderTextTree())
		return 0
	case "explain":
		if len(rest) >= 1 && rest[0] != "pagina" && rest[0] != "tela" {
			// ge explain <dado> [app.ge]
			file := findEntry(rest[1:])
			prog, err := germanioRuntime.Compilar(file)
			if err != nil {
				return fail(err)
			}
			text, err := explicar.Entidade(prog, rest[0])
			if err != nil {
				return fail(err)
			}
			fmt.Fprint(out, text)
			return 0
		}
		if len(rest) < 2 {
			return fail(fmt.Errorf("Uso: ge explain <dado> [app.ge] ou ge explain pagina <nome> [app.ge]"))
		}
		// pages of the intent layer (docs/INTENCAO.md › Página)
		if file := findEntry(rest[2:]); file != "" {
			if prog, err := germanioRuntime.Compilar(file); err == nil && prog.App != nil && len(prog.App.Pages) > 0 {
				text, err := explicar.Pagina(prog, rest[1])
				if err != nil {
					return fail(err)
				}
				fmt.Fprint(out, text)
				return 0
			}
		}
		cwd, _ := os.Getwd()
		graph, err := intelligence.NewProjectAnalyzer(cwd).Analyze()
		if err != nil {
			return fail(err)
		}
		explanation, err := graph.ExplainPage(rest[1])
		if err != nil {
			return fail(err)
		}
		fmt.Fprintf(out, "Página: %s\nRota: %s\nOrigem: %s\nBanco: %s\nBackend: %s\n", explanation.PageName, explanation.Path, explanation.Entity, explanation.DatabaseTable, explanation.BackendService)
		fmt.Fprintf(out, "Componentes: %s\nAções: %s\n", strings.Join(explanation.Components, ", "), strings.Join(explanation.Actions, "; "))
		return 0
	case "eject":
		if len(rest) != 1 {
			return fail(fmt.Errorf("Uso: ge eject <componente>"))
		}
		cwd, _ := os.Getwd()
		file, err := intelligence.NewIntelligenceEngine(cwd).EjectComponent(rest[0])
		if err != nil {
			return fail(err)
		}
		fmt.Fprintln(out, "Componente ejetado:", file)
		return 0
	case "new":
		if len(rest) != 1 {
			return fail(fmt.Errorf("Uso: ge new minha_aplicacao"))
		}
		if err := legacy.CriarProjeto(rest[0], out); err != nil {
			return fail(err)
		}
		return 0
	case "testar", "test":
		path := "."
		coverage, hasPath := false, false
		for _, arg := range rest {
			if arg == "--coverage" && !coverage {
				coverage = true
			} else if !strings.HasPrefix(arg, "-") && !hasPath {
				path, hasPath = arg, true
			} else {
				return fail(fmt.Errorf("Uso: ge testar [arquivo.ge ou pasta] [--coverage]; --race ainda não é suportado"))
			}
		}
		info, err := os.Lstat(path)
		if err != nil {
			return fail(err)
		}
		files, err := geFiles(path)
		if err != nil {
			return fail(err)
		}
		selected := 0
		all, hit := map[diagnostics.Position]bool{}, map[diagnostics.Position]bool{}
		for _, file := range files {
			if info.IsDir() && !strings.HasSuffix(file, "_teste.ge") {
				continue
			}
			m, err := semantic.Load(file)
			if err != nil {
				return fail(err)
			}
			engine := &germanio.Engine{Input: strings.NewReader(""), Output: io.Discard}
			passed, err := engine.RunTests(m)
			for _, name := range passed {
				fmt.Fprintf(out, "ok %s: %s\n", file, name)
			}
			if err != nil {
				return fail(err)
			}
			if coverage {
				covered, eligible := engine.CoverageLines()
				for p := range covered {
					hit[p] = true
				}
				for p := range eligible {
					all[p] = true
				}
			}
			selected += len(passed)
		}
		if selected == 0 {
			return fail(fmt.Errorf("Nenhum teste encontrado em %s. Use teste \"nome\" com espera condição em um arquivo *_teste.ge.", path))
		}
		fmt.Fprintf(out, "%d teste(s) passaram.\n", selected)
		if coverage {
			fmt.Fprintf(out, "Cobertura de instruções: %d/%d (%.1f%%).\n", len(hit), len(all), 100*float64(len(hit))/float64(len(all)))
			missing := make([]diagnostics.Position, 0, len(all)-len(hit))
			for p := range all {
				if !hit[p] {
					missing = append(missing, p)
				}
			}
			sort.Slice(missing, func(i, j int) bool {
				if missing[i].File != missing[j].File {
					return missing[i].File < missing[j].File
				}
				if missing[i].Line != missing[j].Line {
					return missing[i].Line < missing[j].Line
				}
				return missing[i].Column < missing[j].Column
			})
			for _, p := range missing {
				fmt.Fprintf(out, "Sem cobertura: %s:%d:%d\n", p.File, p.Line, p.Column)
			}
		}
		return 0
	case "rodar", "run", "check":
		if cmd == "run" {
			cmd = "rodar"
		}
		if len(rest) == 0 {
			rest = []string{"inicio.ge"}
		}
		path := rest[0]
		if filepath.Ext(path) != ".ge" {
			return fail(fmt.Errorf("Uso: ge %s arquivo.ge", cmd))
		}
		for _, arg := range rest[1:] {
			if strings.HasPrefix(arg, "-") {
				return fail(fmt.Errorf("Flags futuras não são aceitas silenciosamente: %s", arg))
			}
		}
		m, err := semantic.Load(path)
		if err != nil && cmd == "check" {
			// Programas de aplicação (intenção / full-stack)
			prog, cerr := germanioRuntime.Compilar(path)
			if cerr != nil {
				return fail(cerr)
			}
			if prog.App != nil {
				for _, w := range explicar.Verificar(prog) {
					fmt.Fprintln(out, "aviso:", w)
				}
				name := "programa"
				if prog.System != nil {
					name = prog.System.Name
				}
				fmt.Fprintf(out, "Germanio: %s verificado — %d dados, %d papéis.\n", name, len(prog.App.Entities), len(prog.App.Roles))
				return 0
			}
		}
		if err != nil {
			// Fallback: se não for modo semântico estrito, tentar com o engine Germanio
			oldcmd := cmd
			if cmd == "rodar" {
				oldcmd = "run"
			}
			legacy.Run(append([]string{"germanio", oldcmd}, rest...))
			return 0
		}
		if err != nil {
			return fail(err)
		}
		if err = semantic.Check(m); err != nil {
			return fail(err)
		}
		if cmd == "check" {
			fmt.Fprintln(out, "Germanio: verificação concluída.")
			return 0
		}
		if err = (&germanio.Engine{Input: in, Output: out}).Run(m); err != nil {
			return fail(err)
		}
		return 0
	case "explicar":
		if len(rest) != 1 {
			return fail(fmt.Errorf("Uso: ge explicar GE2004"))
		}
		if msg, ok := diagnostics.Explanations[rest[0]]; ok {
			fmt.Fprintln(out, rest[0]+" — "+msg)
			return 0
		}
		return fail(fmt.Errorf("Código desconhecido: %s", rest[0]))
	case "novo":
		if len(rest) != 1 {
			return fail(fmt.Errorf("Uso: ge novo meu_programa"))
		}
		if err := os.Mkdir(rest[0], 0755); err != nil {
			return fail(err)
		}
		path := filepath.Join(rest[0], "inicio.ge")
		if err := os.WriteFile(path, []byte("nome = pergunte \"Qual seu nome?\"\nmostre \"Olá {nome}\"\n"), 0644); err != nil {
			return fail(err)
		}
		fmt.Fprintln(out, "Criado:", path)
		return 0
	case "fmt":
		check := false
		path := "."
		hasPath := false
		for _, a := range rest {
			if a == "--check" {
				check = true
			} else if strings.HasPrefix(a, "-") || hasPath {
				return fail(fmt.Errorf("Uso: ge fmt [arquivo ou pasta] [--check]"))
			} else {
				path = a
				hasPath = true
			}
		}
		files, err := geFiles(path)
		if err != nil {
			return fail(err)
		}
		different := false
		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				return fail(err)
			}
			formatted, err := formatter.Format(file, string(data))
			if err != nil {
				return fail(err)
			}
			if formatted != string(data) {
				different = true
				fmt.Fprintln(out, file)
				if !check {
					if err = replace(file, []byte(formatted)); err != nil {
						return fail(err)
					}
				}
			}
		}
		if check && different {
			return 1
		}
		return 0
	default:
		return fail(fmt.Errorf("Comando desconhecido ou ainda não implementado: %s. Use ge ajuda", cmd))
	}
}

func handleInit(args []string, in io.Reader, out, stderr io.Writer) int {
	_ = in
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	engine := intelligence.NewIntelligenceEngine(cwd)
	dryRun := false
	var exportPath, importPath string
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--dry-run":
			dryRun = true
		case "--yes", "-y":
			// Deterministic non-interactive mode currently uses safe defaults.
		case "--export", "--from":
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, "A flag %s requer um arquivo.\n", arg)
				return 1
			}
			value := args[i+1]
			if arg == "--export" {
				exportPath = value
			} else {
				importPath = value
			}
			i++
		default:
			positional = append(positional, arg)
		}
	}
	if importPath != "" {
		if dryRun {
			fmt.Fprintln(out, "DRY RUN: especificação não foi importada.")
			return 0
		}
		if err := engine.ImportSpec(importPath); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(out, "✓ Projeto importado de %s.\n", importPath)
		return 0
	}

	if len(positional) >= 2 {
		switch positional[0] {
		case "entidade", "modelo":
			name := positional[1]
			if err := engine.InitEntity(intelligence.InitEntityOptions{RootDir: cwd, Name: name, DryRun: dryRun}); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			fmt.Fprintf(out, "✓ Entidade '%s' criada e conectada.\n", name)
			return 0
		case "pagina", "tela":
			name := positional[1]
			if err := engine.InitPage(intelligence.InitPageOptions{RootDir: cwd, Name: name, DryRun: dryRun}); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			fmt.Fprintf(out, "✓ Página '%s' criada e conectada.\n", name)
			return 0
		}
	}
	if len(positional) == 1 && positional[0] == "dashboard" {
		if err := engine.InitDashboard(dryRun); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintln(out, "✓ Dashboard criado com métricas derivadas do grafo.")
		return 0
	}
	if len(positional) == 1 {
		contextualCaps := map[string]string{
			"api":         intelligence.CapAPI,
			"componente":  intelligence.CapDashboard,
			"auth":        intelligence.CapAuth,
			"pagamento":   intelligence.CapBilling,
			"chat":        intelligence.CapChat,
			"busca":       intelligence.CapSearch,
			"upload":      intelligence.CapStorage,
			"notificacao": intelligence.CapNotification,
		}
		if capID, ok := contextualCaps[positional[0]]; ok {
			cap := intelligence.StandardCapabilities[capID]
			fmt.Fprintf(out, "%s — %s [%s].\n", cap.ID, cap.Name, cap.Status)
			return 0
		}
	}

	name := "MeuApp"
	mode := "rapido"
	description := ""
	if len(positional) > 0 {
		name = positional[0]
	}
	if len(positional) > 1 {
		mode = "prompt"
		description = strings.Join(positional[1:], " ")
	}
	graph, err := engine.InitProject(intelligence.InitProjectOptions{
		RootDir: cwd, Name: name, Mode: mode, Description: description,
		DryRun: dryRun, NonInteractive: true,
	})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if dryRun {
		fmt.Fprintf(out, "dry-run: %s", intelligence.NewTransactionManager(cwd, true, nil).RenderDiffPlan())
	}
	if exportPath != "" && !dryRun {
		if err := engine.ExportSpec(exportPath); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(out, "✓ Especificação exportada para %s.\n", exportPath)
	}
	fmt.Fprintf(out, "✓ Projeto '%s' inicializado. Entidades: %d; páginas: %d.\n", name, len(graph.Entities), len(graph.Pages))
	return 0
}

func geFiles(path string) ([]string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("formatter não altera links simbólicos: %s", path)
	}
	if !info.IsDir() {
		if filepath.Ext(path) != ".ge" {
			return nil, fmt.Errorf("formatter requer arquivo .ge")
		}
		return []string{path}, nil
	}
	var files []string
	err = filepath.WalkDir(path, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "vendor" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() && filepath.Ext(p) == ".ge" {
			files = append(files, p)
		}
		return nil
	})
	return files, err
}
func replace(path string, data []byte) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".ge-fmt-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(info.Mode().Perm()); err == nil {
		_, err = f.Write(data)
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), path)
}

// findEntry picks the program file: the argument, or app.ge / inicio.ge.
func findEntry(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	for _, f := range []string{"app.ge", "inicio.ge"} {
		if _, err := os.Stat(f); err == nil {
			return f
		}
	}
	return "app.ge"
}
