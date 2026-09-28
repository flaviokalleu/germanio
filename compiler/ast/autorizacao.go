package ast

// Resolved intent consumed by the runtime (built by parser.ResolveIntent).

// App is the resolved application: entities with their models and the
// authorization table.
type App struct {
	Entities    map[string]*Entity // by model name (singular)
	Order       []string           // declaration order
	Roles       []*Role            // lowest → highest
	LoginEntity string             // model name of the people who sign in
	Login       *LoginDecl
	Integration string // prefix, default "/api"
	Pages       []*PageDecl
	Init        []*Statement
	MemberModel string // model holding memberships (polymorphic)
	Messages    string // "pt" (default) or "en"
	Vocabulary  map[string]string
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
	// Address: `endereço dentro do grupo pai ou do criador`.
	Address    *Address
	Repository bool     // X tem repositório
	RepoKey    string   // field whose value addresses the repository (<valor>.git)
	Search     []string // fields searched by pesquisar
	Filters    []string // fields accepted by filtrar
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
}

// Subscription delivers events of Owner records to the URL of each record.
type Subscription struct {
	Owner      string
	OwnerField string
	Kinds      []string // enviar_codigo, issues, merge_requests…
}
