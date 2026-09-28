package ast

// Resolved intent consumed by the runtime (built by parser.ResolveIntent).

// App is the resolved application: entities with their models and the
// authorization table.
type App struct {
	Entities    map[string]*Entity // by model name (singular)
	Order       []string           // declaration order
	Roles       []*Role            // lowest → highest
	LoginEntity string             // model name of the people who sign in
	// PendingEntity is the data of pending items ("" when none is asked for).
	PendingEntity string
	// ActivityEntity is the data of the history ("" when no data keeps one;
	// GEP 0011, em teste).
	ActivityEntity string
	// EmailNotices: every new pending item is also sent by e-mail (GEP 0013).
	EmailNotices bool
	Login        *LoginDecl
	Integration  string // prefix, default "/api"
	Pages        []*PageDecl
	Init         []*Statement
	MemberModel  string // model holding memberships (polymorphic)
	Messages     string // "pt" (default) or "en"
	Vocabulary   map[string]string
	// GlobalSearch: entities searched together (`tenha busca geral em …`).
	GlobalSearch []string
	// InitialAdmin: `tenha administrador inicial "root"` — login name of the
	// first administrator, created when nobody exists yet.
	InitialAdmin string
	// ReservedAddresses: top-level addresses the product keeps for itself
	// (`endereços reservados`); the runtime adds the app's own routes.
	ReservedAddresses []string
	// Translators: adapter functions for translation points
	// (arquivos_de_execucao, variaveis_das_etapas) — `traduza X com f`.
	Translators map[string]string
}

// Entity is one kind of data the application has.
type Entity struct {
	Model    *Model
	Plural   string
	Singular string
	Label    string
	// Parent relations (pertence a): field → entity model name.
	Parents map[string]string
	// Children (tem): entity model names.
	Children []string
	// HierarchyField links an entity to itself (grupo tem subgrupos).
	HierarchyField string
	// HasMembers: records of this entity have members with roles.
	HasMembers bool
	// InheritVia: field (a parent relation) whose record's members also
	// count here (projeto herda membros do grupo).
	InheritVia string
	// Visibility: field holding privado/interno/publico, when present.
	Visibility   string
	Rules        map[string][]*AccessRule // verb → alternatives (any grants)
	Hooks        map[string]*Hook         // verb → hook
	Integrate    string                   // exposed name for integration, "" = not exposed
	OwnerFields  []string                 // fields pointing to the owner (usuario_id, autor_id, criador_id, dono_id)
	StateField   string
	Initial      string
	Transitions  map[string]*Transition
	Restrictions []*Restriction
	// VisibilityCeiling: parent fields whose visibility bounds this record's.
	CeilingFields []string
	// ProtectedBranchRole: minimum role to change the main branch directly.
	ProtectedBranchRole string
	CreatorRole         string // quem cria X vira <papel>
	MinRole             string // todo X precisa ter pelo menos um <papel>
	// PendingFields: people fields whose new people receive a pending item
	// (issue gera pendência para responsaveis; GEP 0009, em teste).
	PendingFields []string
	// History: every change of this data is recorded in the history
	// (issue guarda histórico; GEP 0011, em teste).
	History bool
	// ViewThrough: a record of this data is seen by whoever sees the record
	// it describes (the history); nil for ordinary data.
	ViewThrough *ViewThrough
	// Review: two branch fields make the record a proposal of changes
	// (changes, commits, mergeability, mesclar performs the merge).
	Review *Review
	// Approvals: `X recebe aprovações` (aprovar/desaprovar, people list).
	Approvals bool
	// Finals: states no transition leaves (mesclado é final).
	Finals []string
	// Execution: this entity is a run of steps (pipelines) or a step (jobs).
	Execution *Execution
	// Subscription: records of this entity receive events of an owner (webhooks).
	Subscription *Subscription
	// ReadOnlyWhen: flag that freezes the record and what belongs to it
	// (`projeto arquivado é somente leitura`).
	ReadOnlyWhen string
	// Remote: records are work taken by remote executors.
	Remote *RemoteWork
	// Address: `endereço dentro do grupo pai ou do criador`.
	Address    *Address
	Repository bool // X tem repositório
	// InitialFile: `repositório do projeto pode começar com "README.md" contendo "# {nome}"`.
	InitialFile *RepoFile
	RepoKey     string   // field whose value addresses the repository (<valor>.git)
	Search      []string // fields searched by pesquisar
	Filters     []string // fields accepted by filtrar
}

// Address: the record's address is its container's address + "/" + its
// own segment (grupo pai "empresa" + caminho "web" → "empresa/web").
// The first container set, in order, is used; without one the segment
// alone is the address. Addresses (and, when people contain records, the
// people's names) form one namespace: none repeats.
type Address struct {
	Field   string // "endereco"
	Segment string // "caminho"
	Within  []AddressRef
}

// RepoFile: a file a new repository may start with ({campo} is replaced by the record's value).
type RepoFile struct{ Path, Content string }

// AddressRef: a parent field and the entity it points to.
type AddressRef struct{ Field, Entity string }

// AccessRule says who may perform a verb on an entity.
type AccessRule struct {
	Verb     string
	Anyone   bool   // todos
	SignedIn bool   // any signed-in person
	MinRole  string // lowest role allowed (membership on the record)
	Own      bool   // only own records
	Custom   bool   // verb defined by a quando hook
}

// Level returns the numeric level of a role name (0 when unknown).
func (a *App) Level(role string) int {
	for _, r := range a.Roles {
		if r.Name == role {
			return r.Level
		}
	}
	return 0
}

// Transition moves a record to Target; Stamp records when and by whom.
type Transition struct {
	Verb   string
	Target string
	Stamp  bool // target is not the initial state: records <alvo>_em / <alvo>_por_id
}

// Restriction limits who sees records whose Flag is true.
type Restriction struct {
	Flag    string
	Owners  []string // owner fields (autor_id…)
	Lists   []string // list fields of people (responsaveis)
	MinRole string
}

// Review ties a record to a repository through two branches.
type Review struct {
	Source, Target string // field names
	RepoVia        string // parent field leading to the record with the repository
}

// Execution describes runs defined by a file in the repository.
type Execution struct {
	Role       string // "run" (pipeline) or "step" (job)
	File       string // configuration file in the repository
	Owner      string // entity with the repository
	OwnerField string // run → owner reference field
	Run        string // step → run entity
	Step       string // run → step entity
	RunField   string // step → run reference field
	// Variables: the data whose records (of the owner) become the steps'
	// environment, and its field pointing at the owner (GEP 0015).
	Variables      string
	VariablesOwner string
}

// RemoteWork: records of this entity are work that executors (records of
// Executor, each holding a secret credential) take and run elsewhere —
// `runners executam jobs`, `trabalhadores executam conversoes`.
type RemoteWork struct {
	Executor      string // entity of the executors
	ExecutorField string // work → executor reference field
	ExecutorKey   string // executor's secret field (its credential)
	Pending       string // state of work waiting for an executor
	Canceled      string // state people set to cancel it
}

// Subscription delivers events of Owner records to the URL of each record.
type Subscription struct {
	Owner      string
	OwnerField string
	Kinds      []string // enviar_codigo, issues, merge_requests…
}

// ViewThrough names the fields of a record that point at the record it
// describes (Kind holds the data, ID its id) and at that record's parent,
// used once the described record no longer exists. Author: the person who
// sees it when neither exists.
type ViewThrough struct {
	Kind, ID             string
	ParentKind, ParentID string
	Author               string
}
