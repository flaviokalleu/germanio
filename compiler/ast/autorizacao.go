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
	Visibility string
	Rules      map[string][]*AccessRule // verb → alternatives (any grants)
	Hooks      map[string]*Hook         // verb → hook
	Integrate  string                   // exposed name for integration, "" = not exposed
	Repository bool                     // X tem repositório
	RepoKey    string                   // field whose value addresses the repository (<valor>.git)
	Search     []string                 // fields searched by pesquisar
	Filters    []string                 // fields accepted by filtrar
}

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
