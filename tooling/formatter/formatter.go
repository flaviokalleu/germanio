// Package formatter provides a single conservative, comment-preserving style.
package formatter

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/flaviokalleu/germanio/compiler/lexer"
	"github.com/flaviokalleu/germanio/compiler/parser"
)

func Format(filename, source string) (string, error) {
	if _, err := parser.ParseGermanio(filename, source); err != nil {
		return "", err
	}
	source = strings.ReplaceAll(source, "\r\n", "\n")
	lines := strings.Split(source, "\n")
	var out []string
	for _, line := range lines {
		code, comment := splitComment(line)
		indent := len(code) - len(strings.TrimLeft(code, " "))
		if strings.TrimSpace(code) == "" {
			if comment != "" {
				out = append(out, strings.Repeat(" ", indent)+strings.TrimRight(comment, " \t"))
			} else {
				out = append(out, "")
			}
			continue
		}
		ts, err := lexer.NewGermanio(filename, strings.TrimSpace(code)).Tokenize()
		if err != nil {
			return "", err
		}
		genericOpen, genericClose := -1, -1
		start := 0
		if len(ts) > 0 && ts[0].Value == "privado" {
			start = 1
		}
		if len(ts) > start+3 && ts[start+1].Value == "<" {
			for j := start + 2; j < len(ts)-1; j++ {
				if ts[j].Value == ">" && ts[j+1].Value == "(" {
					genericOpen, genericClose = start+1, j
					break
				}
			}
		}
		var b strings.Builder
		var prev lexer.Token
		unaryMinus := false
		for i, t := range ts {
			if t.Type == lexer.TokenEOF {
				break
			}
			s := t.Value
			if t.Type == lexer.TokenString {
				s = strconv.Quote(s)
			}
			space := i > 0
			if t.Type != lexer.TokenString && (t.Value == ")" || t.Value == "]" || t.Value == "," || t.Value == ":" || t.Value == "." || t.Value == "?") {
				space = false
			}
			if prev.Type != lexer.TokenString && (prev.Value == "(" || prev.Value == "[" || prev.Value == ".") {
				space = false
			}
			if t.Type == lexer.TokenLParen && prev.Type == lexer.TokenIdentifier {
				space = false
			}
			if t.Type == lexer.TokenLBracket && (prev.Type == lexer.TokenIdentifier || prev.Type == lexer.TokenRBracket || prev.Type == lexer.TokenRParen) {
				if prev.Value != "em" && prev.Value != "mostre" && prev.Value != "retorne" {
					space = false
				}
			}
			if prev.Type == lexer.TokenNumber && t.Type == lexer.TokenIdentifier && (t.Value == "px" || t.Value == "rem") {
				space = false
			}
			if i == genericOpen || i == genericOpen+1 && genericOpen >= 0 || i == genericClose || i == genericClose+1 && genericClose >= 0 {
				space = false
			}
			if unaryMinus {
				space = false
			}
			if space {
				b.WriteByte(' ')
			}
			b.WriteString(s)
			unaryMinus = t.Value == "-" && (i == 0 || isUnaryContext(prev.Value))
			prev = t
		}
		formatted := strings.Repeat(" ", indent) + b.String()
		if comment != "" {
			formatted += "  " + strings.TrimRightFunc(comment, unicode.IsSpace)
		}
		out = append(out, formatted)
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	result := strings.Join(out, "\n") + "\n"
	// Refuse to return a formatting result that no longer parses.
	if _, err := parser.ParseGermanio(filename, result); err != nil {
		return "", err
	}
	return result, nil
}
func isUnaryContext(prev string) bool {
	switch prev {
	case "(", "[", ",", "=", "+", "-", "*", "/", "%", "==", "!=", "<", ">", "<=", ">=", "mostre", "retorne", "espera", "em":
		return true
	}
	return false
}
func splitComment(s string) (string, string) {
	quoted, escape := false, false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escape {
			escape = false
			continue
		}
		if quoted && ch == '\\' {
			escape = true
			continue
		}
		if ch == '"' {
			quoted = !quoted
			continue
		}
		if !quoted && (ch == '#' || ch == '/' && i+1 < len(s) && s[i+1] == '/') {
			return s[:i], s[i:]
		}
	}
	return s, ""
}
