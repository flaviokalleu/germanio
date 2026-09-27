package ast

import (
	"github.com/flaviokalleu/germanio/compiler/diagnostics"
	"github.com/flaviokalleu/germanio/compiler/lexer"
)

// Intent is the human-level description of an application: what exists,
// who can do what, what is allowed, what must happen and what appears.
// The parser records phrases as written; compiler/intencao resolves names
// (plural/singular, entities, roles) after every file has been merged and
// turns them into models, relations and authorization tables.
type Intent struct {
	Entities          []*EntityDecl
	Relations         []*RelationDecl
	Roles             []*Role
	Memberships       []*MembershipDecl
	Grants            []*Grant
	Permits           []*Permit
	Hooks             []*Hook
	Integrations      []*Integration
	IntegrationPrefix string
	Login             *LoginDecl
	Pages             []*PageDecl
	Init              []*Statement
	FieldBlocks       []*FieldsDecl
	Messages          string // mensagens em inglês → "en"
}

// EntityDecl: `tenha clientes` plus the fields from `cada cliente tem`.
type EntityDecl struct {
	Name     string // as written in `tenha` (plural)
	Singular string // explicit or derived
	Label    string // `chamado "Project"` — name used in messages
	Fields   []*Field
	Internal bool
	Pos      diagnostics.Position
}

// FieldsDecl: `cada X tem` / `X tem` block, resolved to an entity later.
type FieldsDecl struct {
	Entity string
	Lines  [][]lexer.Token // fields or relations, decided by the resolver
	Pos    diagnostics.Position
}

// RelationDecl: `cliente tem pedidos` (Kind "tem"), `pedido pertence a
// cliente` (Kind "pertence"), `grupo tem subgrupos` (Kind "hierarquia").
type RelationDecl struct {
	Kind     string
	From, To string
	Optional bool
	As       string // pertence a cliente como dono
	Pos      diagnostics.Position
}

// Role is one access level. Roles are declared from lowest to highest; a
// permission granted to a role holds for every higher role.
type Role struct {
	Name  string
	Level int
	Pos   diagnostics.Position
}

// MembershipDecl: `grupo tem membros com papel` and
// `projeto herda membros do grupo`.
type MembershipDecl struct {
	Entity      string // the resource that has members
	Member      string // entity holding the memberships (membros)
	InheritFrom string // when set: Entity inherits members from this relation
	Pos         diagnostics.Position
}

// Grant: `PAPEL pode VERBO ALVO` (Only = `somente PAPEL pode`).
type Grant struct {
	Role   string
	Only   bool
	Verb   string
	Target string
	Own    bool // seu/sua/seus/suas
	Pos    diagnostics.Position
}

// Permit: `permita VERBO ALVO [por campos]` — enabled for any signed-in
// person (or anyone when the app has no login) unless a grant restricts it.
type Permit struct {
	Verb   string
	Target string
	By     []string
	Pos    diagnostics.Position
}

// Hook: `quando VERBO ALVO` + body. For create/edit/delete it runs after
// the operation; for any other verb the body is the action itself.
type Hook struct {
	Before bool // antes de <verbo>: runs first and may refuse or adjust dados
	Verb   string
	Target string
	Body   []*Statement
	Pos    diagnostics.Position
}

// Integration: `disponibilize X para integração [como "nome"]`.
type Integration struct {
	Target string
	As     string
	Pos    diagnostics.Position
}

// LoginDecl: `tenha login` plus `login ...` configuration lines.
type LoginDecl struct {
	Fields       []string // login usa username e email
	Signup       bool     // tenha cadastro
	TokenEntity  string   // login aceita tokens de acesso
	TokenHeader  string   // no cabeçalho "PRIVATE-TOKEN"
	OAuthSeconds int      // login aceita oauth por 2 horas
	LockAttempts int      // login bloqueia após 10 tentativas por 10 minutos
	LockMinutes  int
	ActiveField  string // login exige estado "active"
	ActiveValue  any
	Pos          diagnostics.Position
}

// PageDecl: `crie página Nome` (+ body) or `crie página para gerenciar X`.
type PageDecl struct {
	Name    string
	Manage  string // entity managed by the page
	Show    string // mostre X
	Permits []string
	PerPage int
	Pos     diagnostics.Position
}

// MergeIntent combines the intent of imported files.
func MergeIntent(a, b *Intent) *Intent {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	a.Entities = append(a.Entities, b.Entities...)
	a.Relations = append(a.Relations, b.Relations...)
	a.Roles = append(a.Roles, b.Roles...)
	a.Memberships = append(a.Memberships, b.Memberships...)
	a.Grants = append(a.Grants, b.Grants...)
	a.Permits = append(a.Permits, b.Permits...)
	a.Hooks = append(a.Hooks, b.Hooks...)
	a.Integrations = append(a.Integrations, b.Integrations...)
	a.Pages = append(a.Pages, b.Pages...)
	a.Init = append(a.Init, b.Init...)
	a.FieldBlocks = append(a.FieldBlocks, b.FieldBlocks...)
	if a.Messages == "" {
		a.Messages = b.Messages
	}
	if a.IntegrationPrefix == "" {
		a.IntegrationPrefix = b.IntegrationPrefix
	}
	if a.Login == nil {
		a.Login = b.Login
	} else if b.Login != nil {
		mergeLogin(a.Login, b.Login)
	}
	return a
}

func mergeLogin(a, b *LoginDecl) {
	if len(b.Fields) > 0 {
		a.Fields = b.Fields
	}
	a.Signup = a.Signup || b.Signup
	if b.TokenEntity != "" {
		a.TokenEntity, a.TokenHeader = b.TokenEntity, b.TokenHeader
	}
	if b.OAuthSeconds > 0 {
		a.OAuthSeconds = b.OAuthSeconds
	}
	if b.LockAttempts > 0 {
		a.LockAttempts, a.LockMinutes = b.LockAttempts, b.LockMinutes
	}
	if b.ActiveField != "" {
		a.ActiveField, a.ActiveValue = b.ActiveField, b.ActiveValue
	}
}

// GetBody returns the statements of a hook (nil-safe).
func (h *Hook) GetBody() []*Statement {
	if h == nil {
		return nil
	}
	return h.Body
}
