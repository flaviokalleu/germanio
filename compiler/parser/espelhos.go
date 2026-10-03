package parser

import (
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
)

// mirrors resolves `espelhos espelham o repositório do projeto` (GEP 0036,
// em teste): the data belongs to a data with a repository and has a url.
// The capability adds what every mirror needs: the direction people choose
// (sentido: enviar or receber), whether it is on (habilitado), the
// credentials taken out of the url (credencial, never shown) and what the
// runtime records about each update (situacao, ultima_atualizacao,
// ultimo_sucesso, ultimo_erro).
func (r *resolver) mirrors(in *ast.Intent) error {
	for _, md := range in.Mirrors {
		m, err := r.entity(md.Data, md.Pos)
		if err != nil {
			return err
		}
		owner, err := r.entity(md.Owner, md.Pos)
		if err != nil {
			return err
		}
		if m == owner {
			return r.errAt(md.Pos, "um dado não espelha o próprio repositório: os espelhos são outro dado, que pertence a %s", owner.Singular)
		}
		if !owner.Repository {
			return r.errAt(md.Pos, "%s espelham o repositório de %s, mas %s não tem repositório. Declare em %s: tem repositório", m.Plural, owner.Singular, owner.Plural, owner.Plural)
		}
		field := ""
		for f, t := range m.Parents {
			if t == owner.Singular {
				field = f
			}
		}
		var missing []string
		if field == "" {
			missing = append(missing, "pertence a "+owner.Singular)
		}
		if u := fieldByNameAST(m.Model, "url"); u == nil {
			missing = append(missing, "url obrigatório")
		} else if u.Type != ast.FieldTexto && u.Type != ast.FieldURL {
			return r.errAt(md.Pos, "a url de %s é o endereço do outro repositório, um texto, mas foi declarada como %s. Declare: url obrigatório", m.Plural, u.Type)
		}
		if len(missing) > 0 {
			return r.errAt(md.Pos, "%s espelham o repositório de %s, mas falta em %s: %s", m.Plural, owner.Singular, m.Plural, strings.Join(missing, ", "))
		}
		add := func(f *ast.Field) {
			if fieldByNameAST(m.Model, f.Name) == nil {
				f.Pos = md.Pos
				m.Model.Fields = append(m.Model.Fields, f)
			}
		}
		add(&ast.Field{Name: "sentido", Type: ast.FieldEnum, EnumValues: []string{"enviar", "receber"}, HasDefault: true, DefaultValue: "enviar"})
		add(&ast.Field{Name: "habilitado", Type: ast.FieldBooleano, HasDefault: true, DefaultValue: true})
		add(&ast.Field{Name: "credencial", Type: ast.FieldTexto, System: true, Hidden: true})
		add(&ast.Field{Name: "situacao", Type: ast.FieldEnum, EnumValues: []string{"nova", "agendada", "atualizando", "atualizada", "falhou"}, System: true, HasDefault: true, DefaultValue: "nova"})
		add(&ast.Field{Name: "ultima_atualizacao", Type: ast.FieldTexto, System: true})
		add(&ast.Field{Name: "ultimo_sucesso", Type: ast.FieldTexto, System: true})
		add(&ast.Field{Name: "ultimo_erro", Type: ast.FieldTexto, System: true})
		m.Mirror = &ast.Mirror{Owner: owner.Singular, OwnerField: field}
	}
	return nil
}
