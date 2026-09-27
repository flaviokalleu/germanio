package interpreter

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/runtime/banco"
	"golang.org/x/crypto/bcrypt"
)

// The declarative schema is enforced here, for every write made from .ge
// code (functions, routes or generated API resources), so a rule declared
// once in `dados` cannot be bypassed by a hand-written route.

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

type fieldErrors map[string][]string

func (e fieldErrors) add(field, msg string) { e[field] = append(e[field], msg) }

func (e fieldErrors) payload() map[string]any {
	out := map[string]any{}
	keys := make([]string, 0, len(e))
	for k := range e {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		list := make([]any, len(e[k]))
		for i, m := range e[k] {
			list[i] = m
		}
		out[k] = list
	}
	return out
}

func (interp *Interpreter) modelAST(name string) *ast.Model {
	if interp.DB == nil {
		return nil
	}
	return interp.DB.Models[strings.ToLower(name)]
}

func fieldByName(m *ast.Model, name string) *ast.Field {
	for _, f := range m.Fields {
		if strings.EqualFold(f.Name, name) {
			return f
		}
	}
	return nil
}

func digest(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

var bcryptCost = bcrypt.DefaultCost

// prepareWrite validates and transforms data for a create or update.
// It returns the values to store and the secrets to reveal once.
func (interp *Interpreter) prepareWrite(c *Call, m *ast.Model, data map[string]any, create bool, id int64) (map[string]any, map[string]any) {
	out := map[string]any{}
	reveal := map[string]any{}
	errs := fieldErrors{}
	for k, v := range data {
		f := fieldByName(m, k)
		if f == nil {
			panic(c.Fail(0, "%s: o modelo '%s' não tem o campo '%s'", c.Name, m.Name, k))
		}
		out[strings.ToLower(f.Name)] = v
	}
	if create {
		for _, f := range m.Fields {
			key := strings.ToLower(f.Name)
			if _, given := out[key]; given {
				continue
			}
			switch {
			case f.HasDefault:
				out[key] = f.DefaultValue
			case f.Name == "expires_at" && m.ExpiresDays > 0:
				out[key] = time.Now().UTC().AddDate(0, 0, m.ExpiresDays).Format("2006-01-02")
			case f.Type == ast.FieldSegredo:
				secret := f.Prefix + randomToken(20)
				out[key] = secret
			}
		}
	}
	for _, f := range m.Fields {
		key := strings.ToLower(f.Name)
		v, present := out[key]
		if !present {
			if create && f.Required {
				errs.add(f.Name, "can't be blank")
			}
			continue
		}
		if !create && f.Immutable {
			errs.add(f.Name, "cannot be changed")
			continue
		}
		if !create && f.Type == ast.FieldSegredo {
			errs.add(f.Name, "cannot be changed")
			continue
		}
		v = normalizeValue(f, v)
		if v == nil || v == "" {
			if f.Required {
				errs.add(f.Name, "can't be blank")
			}
			out[key] = nil
			continue
		}
		if msg := checkType(f, v); msg != "" {
			errs.add(f.Name, msg)
			continue
		}
		if msg := checkBounds(f, v); msg != "" {
			errs.add(f.Name, msg)
			continue
		}
		if f.Format != "" {
			re, err := regexp.Compile(f.Format)
			if err != nil {
				panic(c.Fail(0, "formato inválido no campo %s: %s", f.Name, err))
			}
			if !re.MatchString(toString(v)) {
				errs.add(f.Name, "is invalid")
				continue
			}
		}
		if f.Validator != "" {
			fn, ok := interp.Functions[f.Validator]
			if !ok {
				panic(c.Fail(0, "valida %s: função não existe", f.Validator))
			}
			if !toBool(interp.callFunctionIn(fn, []any{v}, c.Scope, f.Pos)) {
				errs.add(f.Name, "is invalid")
				continue
			}
		}
		if f.Unique && f.Type != ast.FieldSenha {
			filters := map[string]any{key: v}
			if f.Type == ast.FieldSegredo {
				filters[key] = digest(toString(v))
			}
			if !create {
				filters["id__diferente"] = id
			}
			if n, err := interp.DB.ContarFiltro(strings.ToLower(m.Name), banco.Consulta{Filtros: filters}); err == nil && n > 0 {
				errs.add(f.Name, "has already been taken")
				continue
			}
		}
		if f.Reference != "" {
			if row, _ := interp.DB.BuscarRegistro(strings.ToLower(f.Reference), int64(toNumber(v))); row == nil {
				errs.add(strings.TrimSuffix(f.Name, "_id"), "must exist")
				continue
			}
		}
		out[key] = v
	}
	if len(errs) > 0 {
		panic(&RuntimeError{Status: 400, Message: "validação falhou", Payload: errs.payload(), Pos: c.Pos})
	}
	// Secrets are transformed only after every rule passed.
	for _, f := range m.Fields {
		key := strings.ToLower(f.Name)
		v, ok := out[key]
		if !ok || v == nil {
			continue
		}
		switch {
		case f.Type == ast.FieldSenha || (f.Protected && f.Type != ast.FieldSegredo):
			h, err := bcrypt.GenerateFromPassword([]byte(toString(v)), bcryptCost)
			if err != nil {
				panic(c.Fail(400, "%s: %s", f.Name, err))
			}
			out[key] = string(h)
		case f.Type == ast.FieldSegredo:
			reveal[key] = v
			out[key] = digest(toString(v))
		}
	}
	return out, reveal
}

func normalizeValue(f *ast.Field, v any) any {
	s, isText := v.(string)
	switch f.Type {
	case ast.FieldTexto, ast.FieldEmail:
		if isText {
			s = strings.TrimSpace(s)
			if f.Type == ast.FieldEmail {
				s = strings.ToLower(s)
			}
			return s
		}
	case ast.FieldInteiro, ast.FieldNumero:
		if isText {
			if n, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
				return n
			}
		}
	case ast.FieldBooleano:
		if isText {
			switch strings.ToLower(s) {
			case "true", "1", "on", "sim", "verdadeiro":
				return true
			case "false", "0", "off", "nao", "não", "falso":
				return false
			}
		}
		if n, ok := v.(float64); ok {
			return n != 0
		}
	}
	return v
}

func checkType(f *ast.Field, v any) string {
	switch f.Type {
	case ast.FieldEmail:
		if !emailRe.MatchString(toString(v)) {
			return "is invalid"
		}
	case ast.FieldInteiro:
		n, ok := tryNumber(v)
		if _, isStr := v.(string); isStr || !ok || n != math.Trunc(n) {
			return "must be an integer"
		}
	case ast.FieldNumero, ast.FieldDinheiro:
		if _, isStr := v.(string); isStr {
			return "is not a number"
		}
		if _, ok := tryNumber(v); !ok {
			return "is not a number"
		}
	case ast.FieldBooleano:
		if _, ok := v.(bool); !ok {
			return "must be true or false"
		}
	case ast.FieldEnum:
		for _, e := range f.EnumValues {
			if e == toString(v) {
				return ""
			}
		}
		return "is not included in the list"
	case ast.FieldData:
		if _, err := time.Parse("2006-01-02", toString(v)); err != nil {
			return "is not a valid date"
		}
	case ast.FieldURL:
		s := toString(v)
		if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
			return "is not a valid URL"
		}
	}
	return ""
}

func checkBounds(f *ast.Field, v any) string {
	if f.Min == nil && f.Max == nil {
		return ""
	}
	if n, ok := v.(float64); ok && (f.Type == ast.FieldInteiro || f.Type == ast.FieldNumero) {
		if f.Min != nil && n < *f.Min {
			return fmt.Sprintf("must be greater than or equal to %s", toString(*f.Min))
		}
		if f.Max != nil && n > *f.Max {
			return fmt.Sprintf("must be less than or equal to %s", toString(*f.Max))
		}
		return ""
	}
	l := float64(len([]rune(toString(v))))
	if f.Min != nil && l < *f.Min {
		return fmt.Sprintf("is too short (minimum is %s characters)", toString(*f.Min))
	}
	if f.Max != nil && l > *f.Max {
		return fmt.Sprintf("is too long (maximum is %s characters)", toString(*f.Max))
	}
	return ""
}

// stripSecrets removes hashes and digests from a row given to .ge code.
func stripSecrets(m *ast.Model, row map[string]any) map[string]any {
	if row == nil || m == nil {
		return row
	}
	for _, f := range m.Fields {
		if f.IsSecret() {
			delete(row, strings.ToLower(f.Name))
		}
	}
	return row
}

// checkSecretFilters refuses queries on hashed/digested columns.
func checkSecretFilters(c *Call, m *ast.Model, filtros map[string]any) {
	for k := range filtros {
		name := k
		if i := strings.Index(k, "__"); i > 0 {
			name = k[:i]
		}
		if f := fieldByName(m, name); f != nil && f.IsSecret() {
			panic(c.Fail(0, "%s: o campo '%s' é protegido e não pode ser filtrado; use verificar_senha ou por_segredo", c.Name, f.Name))
		}
	}
}

// secretActive reports whether a record found by secret may authenticate.
func secretActive(m *ast.Model, row map[string]any) bool {
	if m.Revocable {
		if b, ok := row["revoked"].(bool); ok && b {
			return false
		}
	}
	if m.ExpiresDays > 0 || fieldByName(m, "expires_at") != nil {
		if exp, ok := row["expires_at"].(string); ok && exp != "" && exp[:min(10, len(exp))] < time.Now().UTC().Format("2006-01-02") {
			return false
		}
	}
	return true
}
