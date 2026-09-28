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

// protectedTarget recognises `enviar código para as <dado> [dos <dono>]`
// where <dado> belongs to <dono> (or to the block's data): the records of
// <dado> name protected branches of <dono> (GEP 0016, em teste).
func (r *resolver) protectedTarget(target, context string) (*ast.Entity, *ast.Entity) {
	dataName, ownerName := target, context
	for _, sep := range []string{"_dos_", "_das_", "_do_", "_da_"} {
		if i := strings.LastIndex(target, sep); i > 0 {
			dataName, ownerName = target[:i], target[i+len(sep):]
			break
		}
	}
	d, o := r.byName[dataName], r.byName[ownerName]
	if d == nil || o == nil || d == o {
		return nil, nil
	}
	for _, t := range d.Parents {
		if t == o.Singular {
			return o, d
		}
	}
	return nil, nil
}

// protectedBranches records that the records of d (belonging to e) name
// branches only rule's role may change.
func (r *resolver) protectedBranches(e, d *ast.Entity, g *ast.Grant, rule *ast.AccessRule) error {
	field := ""
	for f, t := range d.Parents {
		if t == e.Singular {
			field = f
		}
	}
	if !e.Repository {
		return r.errAt(g.Pos, "enviar código para as %s: %s não tem repositório", d.Plural, e.Plural)
	}
	if fieldByNameAST(d.Model, "nome") == nil {
		return r.errAt(g.Pos, "enviar código para as %s: %s precisa ter nome (a branch, ou um padrão com *)", d.Plural, d.Plural)
	}
	if rule.MinRole == "" || !g.Only {
		return r.errAt(g.Pos, "as %s são protegidas por um papel, por exemplo: somente maintainer pode enviar código para as %s dos %s", d.Plural, d.Plural, e.Plural)
	}
	e.ProtectedBranches = append(e.ProtectedBranches, &ast.ProtectedBranches{Data: d.Singular, OwnerField: field, Role: rule.MinRole})
	return nil
}

// mentions: people written as @<nome de usuário> in e's long texts receive
// pending items (GEP 0017, em teste).
func (r *resolver) mentions(e *ast.Entity, app *ast.App, pr *ast.PendingRule) error {
	long := false
	for _, f := range e.Model.Fields {
		long = long || f.Type == ast.FieldTextoLongo
	}
	if !long {
		return r.errAt(pr.Pos, "%s gera pendência para mencionados, mas %s não tem texto onde alguém seja mencionado (um campo como descricao ou texto)", e.Plural, e.Plural)
	}
	le := app.Entities[app.LoginEntity]
	if le == nil {
		return r.errAt(pr.Pos, "mencionados são pessoas: declare tenha login")
	}
	for _, name := range []string{"username", "usuario", "apelido", "login"} {
		if f := fieldByNameAST(le.Model, name); f != nil && f.Unique {
			app.HandleField = strings.ToLower(f.Name)
			break
		}
	}
	if app.HandleField == "" {
		return r.errAt(pr.Pos, "mencionados são escritos como @nome: %s precisa de um nome de usuário único (por exemplo: username único)", le.Plural)
	}
	e.PendingMentions = true
	return nil
}
