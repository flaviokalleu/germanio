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
	"github.com/flaviokalleu/germanio/runtime/germanio"
	"github.com/flaviokalleu/germanio/tooling/formatter"
)

const Version = "0.7.0-dev"
const help = `Germanio — simples para começar, explícito para evoluir.

Uso: ge <comando> [arquivo]
  rodar [inicio.ge]       Verifica e executa Germanio
  check [inicio.ge]       Verifica sem executar nem ler entrada
  testar [arquivo/pasta] [--coverage]  Executa testes .ge isolados
  fmt [arquivo ou pasta] Formata arquivos .ge; --check não escreve
  novo <diretorio>        Cria um programa inicial sem sobrescrever
  explicar <GE0000>       Explica um código de diagnóstico
  versao                 Exibe a versão
  ajuda                  Exibe esta ajuda

Compatibilidade: ge rodar app.ge [porta], ge check app.ge.
Comandos legados explícitos: ge legado <comando Flang> [argumentos].
A linguagem .ge é determinística e não utiliza LLM no compilador.
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
			return fail(fmt.Errorf("Use ge legado ajuda (help no Flang)"))
		}
		legacy.Run(append([]string{"flang"}, rest...))
		return 0
	case "testar":
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
	case "rodar", "check":
		if len(rest) == 0 {
			rest = []string{"inicio.ge"}
		}
		path := rest[0]
		if len(rest) != 1 || filepath.Ext(path) != ".ge" {
			return fail(fmt.Errorf("Uso: ge %s arquivo.ge; flags futuras não são aceitas silenciosamente", cmd))
		}
		m, err := semantic.Load(path)
		if err != nil {
			// Fallback: se não for modo semântico estrito, tentar com o engine Flang
			oldcmd := cmd
			if cmd == "rodar" {
				oldcmd = "run"
			}
			legacy.Run(append([]string{"flang", oldcmd}, rest...))
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
