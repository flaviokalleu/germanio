package banco

import (
	"fmt"
	"os"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"golang.org/x/crypto/bcrypt"
)

// protectPasswords replaces every password field of the input by its bcrypt
// hash before it reaches the database: a password is never stored as text,
// whichever path writes it. A value that already is a bcrypt hash is kept.
func protectPasswords(model *ast.Model, input map[string]any) error {
	for _, f := range model.Fields {
		if f.Type != ast.FieldSenha {
			continue
		}
		name := strings.ToLower(f.Name)
		v, ok := input[name].(string)
		if !ok || v == "" || isBcrypt(v) {
			continue
		}
		cost := bcrypt.DefaultCost
		if os.Getenv("GERMANIO_BCRYPT_RAPIDO") == "1" { // test suites only
			cost = bcrypt.MinCost
		}
		h, err := bcrypt.GenerateFromPassword([]byte(v), cost)
		if err != nil {
			return fmt.Errorf("senha: %w", err)
		}
		input[name] = string(h)
	}
	return nil
}

func isBcrypt(s string) bool {
	return len(s) == 60 && (strings.HasPrefix(s, "$2a$") || strings.HasPrefix(s, "$2b$") || strings.HasPrefix(s, "$2y$"))
}
