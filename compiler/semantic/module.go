package semantic

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/diagnostics"
	"github.com/flaviokalleu/germanio/compiler/parser"
)

type Module struct {
	Program     *ast.Program
	Imports     map[string]*Module
	ImportOrder []string
	Functions   map[string]*Function
	Globals     *Scope
	checked     bool
}
type Function struct {
	Decl   *ast.FuncDecl
	Params []*Type
	Result *Type
	Module *Module
}
type Binding struct {
	Type          *Type
	Mutable, Used bool
	Pos           diagnostics.Position
}
type Scope struct {
	Parent   *Scope
	Bindings map[string]*Binding
}

func scope(parent *Scope) *Scope { return &Scope{Parent: parent, Bindings: map[string]*Binding{}} }
func (s *Scope) lookup(n string) (*Binding, bool) {
	for s != nil {
		if b, ok := s.Bindings[n]; ok {
			return b, true
		}
		s = s.Parent
	}
	return nil, false
}
func validName(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if !(r == '_' || unicode.IsLetter(r) || i > 0 && unicode.IsDigit(r)) {
			return false
		}
	}
	return true
}

// Load resolves only local .ge files, rejects cycles and confines imports to
// the entry-point directory (including symlink targets). It executes no code.
func Load(filename string) (*Module, error) {
	abs, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}
	abs, err = filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fileError(filename, err)
	}
	return load(abs, filepath.Dir(abs), map[string]*Module{}, map[string]bool{})
}
func fileError(path string, err error) error {
	return &diagnostics.Diagnostic{Code: "GE3001", Position: diagnostics.Position{File: path, Line: 1, Column: 1}, Message: "Não foi possível carregar o módulo", Reason: err.Error(), Fix: "Confira o caminho e a permissão de leitura.", Example: `usa "matematica.ge"`}
}
func load(path, root string, cache map[string]*Module, visiting map[string]bool) (*Module, error) {
	if visiting[path] {
		return nil, fileError(path, fmt.Errorf("dependência circular"))
	}
	if m, ok := cache[path]; ok {
		return m, nil
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fileError(path, err)
	}
	p, err := parser.ParseGermanio(path, string(src))
	if err != nil {
		return nil, err
	}
	m := &Module{Program: p, Imports: map[string]*Module{}}
	visiting[path] = true
	defer delete(visiting, path)
	for _, im := range p.Imports {
		alias := strings.TrimSuffix(filepath.Base(im.Path), ".ge")
		if !validName(alias) || alias == "_" || filepath.Ext(im.Path) != ".ge" || filepath.IsAbs(im.Path) {
			return nil, diag(p, im.Pos, "GE3001", "Importação inválida", "Imports precisam de arquivos locais .ge com nome de módulo válido.", `Use usa "matematica.ge".`)
		}
		if _, ok := m.Imports[alias]; ok {
			return nil, diag(p, im.Pos, "GE3001", "Import duplicado", "O nome do módulo já está em uso.", "Remova ou renomeie o módulo duplicado.")
		}
		full, err := filepath.EvalSymlinks(filepath.Join(filepath.Dir(path), im.Path))
		if err != nil {
			return nil, fileError(im.Path, err)
		}
		rel, err := filepath.Rel(root, full)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, diag(p, im.Pos, "GE3001", "Import fora do projeto", "Um módulo não pode ler arquivos externos ao diretório do programa.", "Mova o módulo para dentro do projeto.")
		}
		child, err := load(full, root, cache, visiting)
		if err != nil {
			return nil, err
		}
		m.Imports[alias] = child
		m.ImportOrder = append(m.ImportOrder, alias)
	}
	cache[path] = m
	return m, nil
}

func diag(p *ast.Program, pos diagnostics.Position, code, msg, why, fix string) error {
	return &diagnostics.Diagnostic{Code: code, Position: pos, Source: p.Source, Message: msg, Reason: why, Fix: fix, Example: diagnostics.Explanations[code]}
}

// CheckSource is useful for editor integrations and tests without filesystem IO.
func CheckSource(filename, source string) (*Module, error) {
	p, err := parser.ParseGermanio(filename, source)
	if err != nil {
		return nil, err
	}
	if len(p.Imports) > 0 {
		return nil, fileError(filename, fmt.Errorf("use Load para resolver imports"))
	}
	m := &Module{Program: p, Imports: map[string]*Module{}}
	return m, Check(m)
}
