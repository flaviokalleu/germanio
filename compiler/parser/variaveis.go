package parser

import (
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
)

// runVariables resolves `pipelines usam as variaveis do projeto` (GEP 0015,
// em teste): the data belongs to the owner of the executions and has a name
// (chave or nome) and a valor.
func (r *resolver) runVariables(in *ast.Intent) error {
	for _, d := range in.RunVariables {
		run, err := r.entity(d.Runs, d.Pos)
		if err != nil {
			return err
		}
		if run.Execution == nil || run.Execution.Role != "run" {
			return r.errAt(d.Pos, "%s usam variáveis, mas %s não são execuções. Declare antes: <dono> executa %s a cada envio de código conforme \"arquivo\"", run.Plural, run.Plural, run.Plural)
		}
		owner, err := r.entity(d.Owner, d.Pos)
		if err != nil {
			return err
		}
		if owner.Singular != run.Execution.Owner {
			return r.errAt(d.Pos, "%s são executados por %s, não por %s: escreva %s usam as %s do %s", run.Plural, run.Execution.Owner, owner.Singular, run.Plural, d.Data, run.Execution.Owner)
		}
		vars, err := r.entity(d.Data, d.Pos)
		if err != nil {
			return err
		}
		field := ""
		for f, t := range vars.Parents {
			if t == owner.Singular {
				field = f
			}
		}
		var missing []string
		if field == "" {
			missing = append(missing, "pertence a "+owner.Singular)
		}
		if fieldByNameAST(vars.Model, "chave") == nil && fieldByNameAST(vars.Model, "nome") == nil {
			missing = append(missing, "chave (ou nome)")
		}
		if v := fieldByNameAST(vars.Model, "valor"); v == nil {
			missing = append(missing, "valor")
		} else if v.Type != ast.FieldTexto && v.Type != ast.FieldTextoLongo {
			return r.errAt(d.Pos, "%s viram variáveis das execuções, e uma variável é um texto, mas valor é %s (pelo nome, valor sugere dinheiro). Declare: valor texto oculto", vars.Plural, v.Type)
		}
		if len(missing) > 0 {
			return r.errAt(d.Pos, "%s viram variáveis das execuções, mas falta em %s: %s", vars.Plural, vars.Plural, strings.Join(missing, ", "))
		}
		run.Execution.Variables, run.Execution.VariablesOwner = vars.Singular, field
	}
	return nil
}
