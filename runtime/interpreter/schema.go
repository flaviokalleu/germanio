package interpreter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

// ptMessages translates validation messages when the app speaks Portuguese.
var ptMessages = []struct{ en, pt string }{
	{"can't be blank", "é obrigatório"},
	{"has already been taken", "já está em uso"},
	{"is invalid", "é inválido"},
	{"cannot be changed", "não pode ser alterado"},
	{"must be an integer", "deve ser um número inteiro"},
	{"is not a number", "deve ser um número"},
	{"must be true or false", "deve ser sim ou não"},
	{"is not included in the list", "não é uma opção válida"},
	{"is not a valid date", "não é uma data válida"},
	{"is not a valid URL", "não é um endereço válido"},
	{"must exist", "não existe"},
	{"is too short (minimum is ", "é muito curto (mínimo de "},
	{"is too long (maximum is ", "é muito longo (máximo de "},
	{" characters)", " caracteres)"},
	{"must be greater than or equal to ", "deve ser pelo menos "},
	{"must be less than or equal to ", "deve ser no máximo "},
}

// Lang is the language of messages: "pt" for apps with intent (unless
// `mensagens em inglês`), "en" otherwise (API-compatible default).
func (interp *Interpreter) Lang() string {
	if interp.App != nil {
		return interp.App.Messages
	}
	return "en"
}

func translate(lang, msg string) string {
	if lang != "pt" {
		return msg
	}
	for _, t := range ptMessages {
		msg = strings.ReplaceAll(msg, t.en, t.pt)
	}
	return msg
}

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

// sentence renders errors for people: "email já está em uso".
func (e fieldErrors) sentence() string {
	keys := make([]string, 0, len(e))
	for k := range e {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		for _, m := range e[k] {
			parts = append(parts, k+" "+m)
		}
	}
	return strings.Join(parts, "; ")
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
	db := interp.dbOf(c.Ctx())
	var current map[string]any // the record before this update, read on demand
	for k, v := range data {
		f := fieldByName(m, k)
		if f == nil {
			panic(c.Fail(0, "%s: o modelo '%s' não tem o campo '%s'", c.Name, m.Name, k))
		}
		out[strings.ToLower(f.Name)] = v
	}
	interp.dropAddress(m, out)
	if create {
		for _, f := range m.Fields {
			key := strings.ToLower(f.Name)
			if _, given := out[key]; given {
				continue
			}
			switch {
			case f.HasDefault:
				out[key] = f.DefaultValue
			case f.Name == "expira_em" && m.ExpiresDays > 0:
				out[key] = time.Now().UTC().AddDate(0, 0, m.ExpiresDays).Format("2006-01-02")
			case f.Type == ast.FieldSegredo:
				secret := f.Prefix + randomToken(20)
				out[key] = secret
			case f.Type == ast.FieldVisibilidade:
				out[key] = "private"
			case f.NumberedBy != "" && out[strings.ToLower(f.NumberedBy)] != nil:
				n, err := db.Sequencia(fmt.Sprintf("%s.%s:%v", strings.ToLower(m.Name), key, toString(out[strings.ToLower(f.NumberedBy)])))
				if err != nil {
					panic(c.Fail(0, "numeração de %s: %s", f.Name, err))
				}
				out[key] = float64(n)
			case f.Type == ast.FieldLista:
				out[key] = []any{}
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
		if f.Type == ast.FieldLista {
			list, msg := interp.checkList(db, f, v, func(k string) any {
				if x, ok := out[k]; ok {
					return x
				}
				if !create {
					if current == nil {
						current, _ = db.BuscarRegistro(strings.ToLower(m.Name), id)
					}
					return current[k]
				}
				return nil
			}, m)
			if msg != "" {
				errs.add(f.Name, msg)
				continue
			}
			out[key] = list
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
			if n, err := db.ContarFiltro(strings.ToLower(m.Name), banco.Consulta{Filtros: filters}); err == nil && n > 0 {
				errs.add(f.Name, "has already been taken")
				continue
			}
		}
		if f.Reference != "" {
			if row, _ := db.BuscarRegistro(strings.ToLower(f.Reference), int64(toNumber(v))); row == nil {
				errs.add(strings.TrimSuffix(f.Name, "_id"), "must exist")
				continue
			}
		}
		out[key] = v
	}
	interp.address(db, m, out, create, id, errs)
	if len(errs) > 0 {
		lang := interp.Lang()
		for k, list := range errs {
			for i := range list {
				list[i] = translate(lang, list[i])
			}
			errs[k] = list
		}
		panic(&RuntimeError{Status: 400, Message: errs.sentence(), Payload: errs.payload(), Pos: c.Pos})
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
	case ast.FieldVisibilidade:
		if isText {
			switch strings.ToLower(strings.TrimSpace(s)) {
			case "public", "publico", "público", "publica", "pública":
				return "public"
			case "internal", "interno", "interna":
				return "internal"
			case "private", "privado", "privada":
				return "private"
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
	case ast.FieldVisibilidade:
		switch toString(v) {
		case "public", "internal", "private":
			return ""
		}
		return "is not included in the list"
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
		if b, ok := row["revogado"].(bool); ok && b {
			return false
		}
	}
	if m.ExpiresDays > 0 || fieldByName(m, "expira_em") != nil {
		if exp, ok := row["expira_em"].(string); ok && exp != "" && exp[:min(10, len(exp))] < time.Now().UTC().Format("2006-01-02") {
			return false
		}
	}
	return true
}

// checkList normalizes a list field ("a, b" or [..]) and validates that
// referenced records exist. It is stored as JSON text.
// checkList validates a list of references. Items that belong to a parent
// the record also has (labels of the project an issue is in) must share it:
// anything else is treated as inexistent, so nothing of other records leaks.
func (interp *Interpreter) checkList(db *banco.Banco, f *ast.Field, v any, own func(string) any, m *ast.Model) (string, string) {
	var shared []string
	if tm := interp.modelAST(f.ListOf); tm != nil && f.ListOf != "texto" && f.ListOf != "numero" {
		for _, tf := range tm.Fields {
			if tf.Reference != "" && tf.Reference != f.ListOf {
				if rf := fieldByName(m, tf.Name); rf != nil && rf.Reference == tf.Reference {
					shared = append(shared, strings.ToLower(tf.Name))
				}
			}
		}
	}
	var items []any
	switch x := v.(type) {
	case []any:
		items = x
	case string:
		for _, part := range strings.Split(x, ",") {
			if p := strings.TrimSpace(part); p != "" {
				items = append(items, p)
			}
		}
	default:
		items = []any{x}
	}
	seen := map[string]bool{}
	out := []string{}
	for _, it := range items {
		sv := strings.TrimSpace(toString(it))
		if sv == "" || seen[sv] {
			continue
		}
		seen[sv] = true
		if f.ListOf != "texto" && f.ListOf != "numero" {
			id, ok := tryNumber(it)
			if !ok {
				return "", "must contain ids"
			}
			row, _ := db.BuscarRegistro(f.ListOf, int64(id))
			if row == nil {
				return "", "must exist"
			}
			for _, k := range shared {
				if toString(row[k]) != toString(own(k)) {
					return "", "must exist"
				}
			}
			sv = toString(id)
		}
		out = append(out, sv)
	}
	b, _ := json.Marshal(out)
	return string(b), ""
}
