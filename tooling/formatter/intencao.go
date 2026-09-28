package formatter

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/lexer"
	"github.com/flaviokalleu/germanio/compiler/parser"
)

// The canonical form of application files (docs/INTENCAO.md › Layout):
// 4 spaces per level (levels come from the same level stack the parser
// uses), no trailing spaces, at most one blank line in a row, comments kept
// and placed at the level of the code they precede, text inside quotes and
// triple-quoted blocks untouched, a final newline. Tabs at the start of a
// line become spaces (they are an error for the parser). The result must
// describe exactly the same program; otherwise formatting is refused.

// errNotApplication: the file is neither the strict core nor an application.
var errNotApplication = fmt.Errorf("não é um programa da camada de intenção")

func parseApplication(filename, src string) (*ast.Program, error) {
	toks, err := lexer.New(src).Tokenize()
	if err != nil {
		return nil, err
	}
	p := parser.New(toks)
	p.File = filename
	prog, err := p.Parse()
	if err != nil {
		return nil, err
	}
	if prog.Intent == nil {
		return nil, errNotApplication
	}
	return prog, nil
}

// meaning is the program without positions: two sources with the same
// meaning format to programs that are equal here.
func meaning(prog *ast.Program) (any, error) {
	b, err := json.Marshal(struct {
		Intent    *ast.Intent
		Functions []*ast.FuncDecl
		Routes    []*ast.CustomRoute
		Scripts   []*ast.Statement
		System    *ast.System
		Imports   []*ast.Import
	}{prog.Intent, prog.Functions, prog.Routes, prog.Scripts, prog.System, prog.Imports})
	if err != nil {
		return nil, err
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	var strip func(any)
	strip = func(x any) {
		switch m := x.(type) {
		case map[string]any:
			for _, k := range []string{"Pos", "Line", "Column", "Indent", "Context", "from", "to"} {
				delete(m, k)
			}
			for _, val := range m {
				strip(val)
			}
		case []any:
			for _, val := range m {
				strip(val)
			}
		}
	}
	strip(v)
	return v, nil
}

type fline struct {
	raw     string
	indent  int
	code    string // without the comment, spaces normalized
	comment string
	kind    int // blank, comment, code, verbatim
}

const (
	blank = iota
	comment
	code
	verbatim
)

func formatApplication(filename, source string) (string, error) {
	source = strings.ReplaceAll(source, "\r\n", "\n")
	var lines []fline
	inTriple := false
	for _, raw := range strings.Split(source, "\n") {
		// leading tabs become 4 spaces each
		lead := 0
		for lead < len(raw) && (raw[lead] == ' ' || raw[lead] == '\t') {
			lead++
		}
		prefix := strings.ReplaceAll(raw[:lead], "\t", "    ")
		raw = prefix + raw[lead:]
		if inTriple || strings.Contains(raw, `"""`) {
			if strings.Count(raw, `"""`)%2 == 1 {
				inTriple = !inTriple
			}
			lines = append(lines, fline{raw: strings.TrimRight(raw, " \t"), kind: verbatim, indent: len(prefix)})
			continue
		}
		c, cm := splitComment(raw)
		l := fline{raw: raw, indent: len(prefix), comment: strings.TrimRight(cm, " \t")}
		switch {
		case strings.TrimSpace(c) == "" && cm == "":
			l.kind = blank
		case strings.TrimSpace(c) == "":
			l.kind = comment
		default:
			l.kind = code
			l.code = collapse(strings.TrimSpace(c))
		}
		lines = append(lines, l)
	}
	original, err := parseApplication(filename, strings.Join(rawLines(lines), "\n"))
	if err != nil {
		return "", err
	}

	// levels of code (and verbatim) lines by the level stack
	depth := make([]int, len(lines))
	stack := []int{0}
	for i, l := range lines {
		if l.kind != code && l.kind != verbatim {
			continue
		}
		if l.kind == verbatim && i > 0 && lines[i-1].kind == verbatim {
			depth[i] = -1 // inside a triple-quoted block: untouched
			continue
		}
		for len(stack) > 1 && l.indent < stack[len(stack)-1] {
			stack = stack[:len(stack)-1]
		}
		if l.indent > stack[len(stack)-1] {
			stack = append(stack, l.indent)
		}
		depth[i] = len(stack) - 1
	}
	// a comment takes the level of the code it precedes
	next := 0
	for i := len(lines) - 1; i >= 0; i-- {
		switch lines[i].kind {
		case code, verbatim:
			if depth[i] >= 0 {
				next = depth[i]
			}
		case comment:
			depth[i] = next
		}
	}

	var out []string
	for i, l := range lines {
		pad := strings.Repeat("    ", max(depth[i], 0))
		switch l.kind {
		case blank:
			if len(out) == 0 || out[len(out)-1] == "" {
				continue
			}
			out = append(out, "")
		case comment:
			out = append(out, pad+l.comment)
		case verbatim:
			if depth[i] < 0 {
				out = append(out, l.raw) // continuation of a triple-quoted text
			} else {
				out = append(out, pad+strings.TrimLeft(l.raw, " "))
			}
		case code:
			s := pad + l.code
			if l.comment != "" {
				s += "  " + l.comment
			}
			out = append(out, s)
		}
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	result := strings.Join(out, "\n") + "\n"

	formatted, err := parseApplication(filename, result)
	if err != nil {
		return "", fmt.Errorf("%s: a formatação produziria código inválido (%v); o arquivo não foi alterado", filename, err)
	}
	a, err1 := meaning(original)
	b, err2 := meaning(formatted)
	if err1 != nil || err2 != nil || !reflect.DeepEqual(a, b) {
		return "", fmt.Errorf("%s: a formatação mudaria o significado do programa; o arquivo não foi alterado", filename)
	}
	return result, nil
}

func rawLines(lines []fline) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = l.raw
	}
	return out
}

// collapse turns runs of spaces into one, outside quotes.
func collapse(s string) string {
	var b strings.Builder
	quoted, escape, space := false, false, false
	for _, r := range s {
		switch {
		case escape:
			escape = false
		case quoted && r == '\\':
			escape = true
		case r == '"':
			quoted = !quoted
		case !quoted && (r == ' ' || r == '\t'):
			space = true
			continue
		}
		if space {
			b.WriteByte(' ')
			space = false
		}
		b.WriteRune(r)
	}
	return b.String()
}
