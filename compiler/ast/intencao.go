package ast

import (
	"fmt"

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
	// VocabularyPos: where each translation was declared.
	VocabularyPos map[string]diagnostics.Position
	// Conflicts: the same fact declared with different values in two files
	// (found while merging; the resolver reports them with both origins).
	Conflicts  []string
	Ceilings   []*VisibilityCeiling
	Creators   []*CreatorRole
	Approvals  []string     // X recebe aprovações
	Finals     []*StateDecl // X <estado> é final
	Executions []*ExecutionDecl
	// RunVariables: `pipelines usam as variaveis do projeto` (GEP 0015).
	RunVariables      []*RunVariablesDecl
	Subscriptions     []*SubscriptionDecl
	RemoteExecutors   []*RemoteExecutorDecl
	Translators       map[string]string // traduza <ponto> com <função> (integracoes/)
	InitialFiles      []*InitialFileDecl
	ReservedAddresses []string // endereços reservados
	InitialAdmin      string   // tenha administrador inicial "root"
	// Capabilities: the grants that turned out to be capabilities of data
	// (`issue pode fechar`), kept for ge explain after resolution.
	Capabilities []*Grant
	GlobalSearch []string          // tenha busca geral em projetos, issues
	ReadOnly     []*VisibilityRule // projeto arquivado é somente leitura (Flag; Who unused)
	MinRoles     []*CreatorRole    // todo grupo precisa ter pelo menos um owner
	// Pending items (GEP 0009): issue gera pendência para responsaveis.
	PendingItems []*PendingRule
	// Presence: `tenha presença` (GEP 0021, em teste).
	Presence bool
	// EmailNotices: `tenha avisos por e-mail` (GEP 0013, em teste).
	EmailNotices    bool
	EmailNoticesPos diagnostics.Position
	// History: `issue guarda histórico` (GEP 0011, em teste).
	History []*HistoryDecl
	// Reading: `mensagem guarda leitura` (GEP 0022, em teste).
	Reading []*HistoryDecl
	// Renames: renomeie nome de clientes para nome_completo (G93).
	Renames []*RenameDecl
	// Discards: descarte telefone de clientes (removed on purpose, data kept).
	Discards []*RenameDecl
}

// RenameDecl: a field of Entity renamed (From → To), or discarded (To "").
type RenameDecl struct {
	Entity, From, To string
	Pos              diagnostics.Position
}

// HistoryDecl: every change of Entity is recorded in the history.
type HistoryDecl struct {
	Entity string
	Pos    diagnostics.Position
}

// PendingRule: a person placed in one of Fields of Entity receives a pending
// item for the record.
type PendingRule struct {
	Entity string
	Fields []string
	Pos    diagnostics.Position
}

// EntityDecl: `tenha clientes` plus the fields from `cada cliente tem`.
type EntityDecl struct {
	Name     string // as written in `tenha` (plural)
	Singular string // explicit or derived
	Label    string // `chamado "Project"` — name used in messages
	Fields   []*Field
	Internal bool
	// Implicit: declared by a data block (the block names the data); it
	// merges with other blocks and with an explicit tenha.
	Implicit bool
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
	// Context: the data block the grant was written in (acesso); its target
	// must be that data or something that belongs to it.
	Context string
	Pos     diagnostics.Position
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
	Recovery     bool     // tenha recuperação de senha (GEP 0008, em teste)
	Confirmation bool     // tenha confirmação de e-mail (GEP 0031, em teste)
	TwoFactor    bool     // tenha autenticação em dois fatores (GEP 0032, em teste)
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
	// Page sections (GEP 0002, em teste): each one only replaces its own
	// default; none is required.
	Title   string        // topo › título "X"
	Text    string        // topo › texto "X"
	Actions []*PageAction // topo › ações › verbo ["rótulo"]
	Filters []string      // filtros › pesquisar | campo
	Columns []string      // colunas › campo
	Empty   *PageEmpty    // vazio › título, texto, ação
	// Indicators (GEP 0012, em teste): indicadores › total de <dado> [estado].
	Indicators []*PageIndicator
	// ByState: data shown in columns by state (GEP 0023, em teste).
	ByState []*PageByState
}

// PageByState: Data (resolved to its singular) is shown by state on the page.
type PageByState struct {
	Data string
	Pos  diagnostics.Position
}

// PageIndicator counts the records of Entity (in State, when given) the
// viewer can see. Words holds the data and state as written, resolved later.
type PageIndicator struct {
	Words  []string
	Entity string
	State  string
	Label  string
	Pos    diagnostics.Position
}

// PageAction is a verb the page offers, with an optional label. The verb
// must already be allowed by the page (permita): the page asks, the domain
// decides who sees it.
type PageAction struct {
	Verb  string
	Label string
	Pos   diagnostics.Position
}

// PageEmpty is what the page says when there is nothing to show.
type PageEmpty struct {
	Title  string
	Text   string
	Action *PageAction
	Pos    diagnostics.Position
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
	a.RunVariables = append(a.RunVariables, b.RunVariables...)
	a.Subscriptions = append(a.Subscriptions, b.Subscriptions...)
	a.RemoteExecutors = append(a.RemoteExecutors, b.RemoteExecutors...)
	for k, v := range b.Translators {
		if a.Translators == nil {
			a.Translators = map[string]string{}
		}
		a.Translators[k] = v
	}
	a.MinRoles = append(a.MinRoles, b.MinRoles...)
	a.Capabilities = append(a.Capabilities, b.Capabilities...)
	a.PendingItems = append(a.PendingItems, b.PendingItems...)
	a.History = append(a.History, b.History...)
	a.Reading = append(a.Reading, b.Reading...)
	a.EmailNotices = a.EmailNotices || b.EmailNotices
	a.Presence = a.Presence || b.Presence
	if a.EmailNoticesPos.Line == 0 {
		a.EmailNoticesPos = b.EmailNoticesPos
	}
	a.Renames = append(a.Renames, b.Renames...)
	a.Discards = append(a.Discards, b.Discards...)
	a.InitialFiles = append(a.InitialFiles, b.InitialFiles...)
	a.ReservedAddresses = append(a.ReservedAddresses, b.ReservedAddresses...)
	a.ReadOnly = append(a.ReadOnly, b.ReadOnly...)
	a.GlobalSearch = append(a.GlobalSearch, b.GlobalSearch...)
	a.Visibility = append(a.Visibility, b.Visibility...)
	if a.Vocabulary == nil {
		a.Vocabulary = b.Vocabulary
	} else {
		for k, v := range b.Vocabulary {
			if old, ok := a.Vocabulary[k]; ok && old != v {
				a.Conflicts = append(a.Conflicts, fmt.Sprintf("%s é traduzido como %q em %s e como %q em %s: um nome tem uma só tradução na integração", k, old, where(a.VocabularyPos[k]), v, where(b.VocabularyPos[k])))
				continue
			}
			a.Vocabulary[k] = v
		}
	}
	if a.VocabularyPos == nil {
		a.VocabularyPos = b.VocabularyPos
	} else {
		for k, p := range b.VocabularyPos {
			if _, ok := a.VocabularyPos[k]; !ok {
				a.VocabularyPos[k] = p
			}
		}
	}
	a.Conflicts = append(a.Conflicts, b.Conflicts...)
	if a.Messages == "" {
		a.Messages = b.Messages
	}
	if a.InitialAdmin == "" {
		a.InitialAdmin = b.InitialAdmin
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
	a.Recovery = a.Recovery || b.Recovery
	a.Confirmation = a.Confirmation || b.Confirmation
	a.TwoFactor = a.TwoFactor || b.TwoFactor
	if b.TokenEntity != "" {
		a.TokenEntity = b.TokenEntity
	}
	if b.TokenHeader != "" {
		a.TokenHeader = b.TokenHeader
	}
	if b.OAuthSeconds > 0 {
		a.OAuthSeconds = b.OAuthSeconds
	}
	if b.LockAttempts > 0 {
		a.LockAttempts = b.LockAttempts
	}
	if b.LockMinutes > 0 {
		a.LockMinutes = b.LockMinutes
	}
	if b.ActiveField != "" {
		a.ActiveField = b.ActiveField
	}
	if b.ActiveValue != nil {
		a.ActiveValue = b.ActiveValue
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

// RunVariablesDecl: records of Data that belong to the owner of the
// executions Runs become the environment of their steps.
type RunVariablesDecl struct {
	Runs, Data, Owner string
	Pos               diagnostics.Position
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

func where(p diagnostics.Position) string {
	if p.File == "" {
		return fmt.Sprintf("linha %d", p.Line)
	}
	return fmt.Sprintf("%s:%d", p.File, p.Line)
}
