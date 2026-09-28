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
	States            []*StateDecl
	Visibility        []*VisibilityRule
	Vocabulary        map[string]string
	Ceilings          []*VisibilityCeiling
	Creators          []*CreatorRole
	Approvals         []string     // X recebe aprovações
	Finals            []*StateDecl // X <estado> é final
	Executions        []*ExecutionDecl
	Subscriptions     []*SubscriptionDecl
	RemoteExecutors   []*RemoteExecutorDecl
	Translators       map[string]string // traduza <ponto> com <função> (integracoes/)
	InitialFiles      []*InitialFileDecl
	MinRoles          []*CreatorRole // todo grupo precisa ter pelo menos um owner
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
	Scopes       map[string][]string // escopo "x" permite ler|escrever|tudo|baixar código|enviar código
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
	a.States = append(a.States, b.States...)
	a.Ceilings = append(a.Ceilings, b.Ceilings...)
	a.Creators = append(a.Creators, b.Creators...)
	a.Approvals = append(a.Approvals, b.Approvals...)
	a.Finals = append(a.Finals, b.Finals...)
	a.Executions = append(a.Executions, b.Executions...)
	a.Subscriptions = append(a.Subscriptions, b.Subscriptions...)
	a.RemoteExecutors = append(a.RemoteExecutors, b.RemoteExecutors...)
	for k, v := range b.Translators {
		if a.Translators == nil {
			a.Translators = map[string]string{}
		}
		a.Translators[k] = v
	}
	a.MinRoles = append(a.MinRoles, b.MinRoles...)
	a.InitialFiles = append(a.InitialFiles, b.InitialFiles...)
	a.Visibility = append(a.Visibility, b.Visibility...)
	if a.Vocabulary == nil {
		a.Vocabulary = b.Vocabulary
	} else {
		for k, v := range b.Vocabulary {
			a.Vocabulary[k] = v
		}
	}
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
	for k, v := range b.Scopes {
		if a.Scopes == nil {
			a.Scopes = map[string][]string{}
		}
		a.Scopes[k] = v
	}
}

// GetBody returns the statements of a hook (nil-safe).
func (h *Hook) GetBody() []*Statement {
	if h == nil {
		return nil
	}
	return h.Body
}

// StateDecl: `issue começa aberta` — the entity has a state machine whose
// transitions come from `issue pode fechar / reabrir`.
type StateDecl struct {
	Entity  string
	Initial string
	Pos     diagnostics.Position
}

// VisibilityRule: `issue confidencial pode ser vista por autor,
// responsaveis, reporter ou superior` — when the flag is set, only these
// may see the record (administrators always may).
type VisibilityRule struct {
	Entity string
	Flag   string
	Who    []string
	Pos    diagnostics.Position
}

// VisibilityCeiling: `grupo não pode ser mais visível que o grupo pai`.
type VisibilityCeiling struct {
	Entity string
	Parent string // entity name, or "pai" for the hierarchy parent
	Pos    diagnostics.Position
}

// CreatorRole: `quem cria grupo vira owner`.
type CreatorRole struct {
	Entity string
	Role   string
	Pos    diagnostics.Position
}

// ExecutionDecl: `projeto executa pipelines a cada envio de código conforme
// "pipeline.yml"` — each push creates a run whose steps come from the file.
type ExecutionDecl struct {
	Owner, Entity, File string
	Pos                 diagnostics.Position
}

// SubscriptionDecl: `webhook recebe eventos do projeto` + kinds.
type SubscriptionDecl struct {
	Subscriber, Owner string
	Kinds             []string
	Pos               diagnostics.Position
}

// RemoteExecutorDecl: `runners executam jobs` — records of Executor, each
// with a secret credential, take pending steps and run them elsewhere.
type RemoteExecutorDecl struct {
	Executor, Steps string
	Pos             diagnostics.Position
}

// InitialFileDecl: `repositório do projeto pode começar com "README.md" contendo "# {nome}"`.
type InitialFileDecl struct {
	Entity, Path, Content string
	Pos                   diagnostics.Position
}
