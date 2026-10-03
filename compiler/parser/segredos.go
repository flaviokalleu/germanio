package parser

import (
	"fmt"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
)

// Secrets at rest (docs/gep/0049-segredos-guardados.md, em teste), without
// new syntax: a field people write that never appears (`oculto`) exists only
// to be used by the system itself — a webhook's token sent to another
// server, a value handed to an execution. Such a text is kept encrypted in
// the database. Capabilities mark their own credentials (a mirror's
// `credencial`) the same way. Passwords and generated secrets (`senha`,
// `segredo`) are not sealed: they are kept as hashes and never read back.

// sealable: types stored as text whose value is read back as a whole.
func sealable(f *ast.Field) bool {
	switch f.Type {
	case "", ast.FieldTexto, ast.FieldTextoLongo, ast.FieldURL, ast.FieldLink:
		return true
	}
	return false
}

// sealFields marks the sealed fields and refuses what a sealed field cannot
// do: the database only holds ciphertext, so it cannot compare it.
func (r *resolver) sealFields() error {
	for _, n := range r.app.Order {
		e := r.app.Entities[n]
		for _, f := range e.Model.Fields {
			if f.Hidden && !f.System && !f.IsSecret() && sealable(f) {
				f.Sealed = true
			}
			if !f.Sealed {
				continue
			}
			if f.Unique {
				return r.errAt(f.Pos, "%s de %s é oculto e único ao mesmo tempo.\nPor quê: um campo oculto fica cifrado no banco (cada valor cifrado é diferente, mesmo quando o texto é igual), então o banco não consegue comparar os valores.\nComo corrigir: tire único de %s; se o valor precisa identificar o registro, use segredo (gerado e guardado como hash): %s segredo", f.Name, e.Plural, f.Name, f.Name)
			}
			if f.Index {
				return r.errAt(f.Pos, "%s de %s é oculto e tem índice.\nPor quê: um campo oculto fica cifrado no banco, e um índice de valores cifrados não encontra nada.\nComo corrigir: tire índice de %s", f.Name, e.Plural, f.Name)
			}
		}
	}
	return nil
}

// sealedFilterMsg explains why a sealed field cannot be filtered.
func sealedFilterMsg(e *ast.Entity, f *ast.Field, what string) string {
	return fmt.Sprintf("%s: %s de %s é oculto e fica cifrado no banco.\nPor quê: filtrar por ele revelaria o valor a quem filtra (acertando um valor por vez) e o banco não compara valores cifrados.\nComo corrigir: tire %s do filtro; para achar registros, filtre por um campo que pode aparecer",
		what, strings.ReplaceAll(f.Name, "_", " "), e.Plural, f.Name)
}
