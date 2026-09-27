package intelligence

import (
	"fmt"
	"sort"
	"strings"
)

// FieldType represents standard data types recognized across frontend, backend, and DB in Germanio.
type FieldType string

const (
	FieldTexto      FieldType = "texto"
	FieldTextoLongo FieldType = "texto_longo"
	FieldInteiro    FieldType = "inteiro"
	FieldDecimal    FieldType = "decimal"
	FieldDinheiro   FieldType = "dinheiro"
	FieldPercentual FieldType = "percentual"
	FieldBooleano   FieldType = "booleano"
	FieldEmail      FieldType = "email"
	FieldTelefone   FieldType = "telefone"
	FieldURL        FieldType = "url"
	FieldUUID       FieldType = "uuid"
	FieldData       FieldType = "data"
	FieldHora       FieldType = "hora"
	FieldDataHora   FieldType = "data_hora"
	FieldArquivo    FieldType = "arquivo"
	FieldImagem     FieldType = "imagem"
	FieldStatus     FieldType = "status"
	FieldEnum       FieldType = "enum"
	FieldRelacao    FieldType = "relacao"
)

// FieldMeta defines a field on an entity.
type FieldMeta struct {
	Name        string    `json:"name"`
	Type        FieldType `json:"type"`
	Required    bool      `json:"required"`
	Unique      bool      `json:"unique"`
	Searchable  bool      `json:"searchable"`
	Filterable  bool      `json:"filterable"`
	DefaultVal  string    `json:"default_val,omitempty"`
	EnumValues  []string  `json:"enum_values,omitempty"`
	TargetRef   string    `json:"target_ref,omitempty"` // For relation
	Description string    `json:"description,omitempty"`
}

// RelationMeta defines relationships between entities.
type RelationMeta struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	Target      string `json:"target"`
	Type        string `json:"type"` // "1:1", "1:N", "N:N"
	Description string `json:"description,omitempty"`
}

// ActionBinding defines a button or UI action connected to a backend handler.
type ActionBinding struct {
	Name         string `json:"name"`          // e.g. "salvar", "excluir", "buscar"
	Label        string `json:"label"`         // e.g. "Salvar", "Excluir"
	TargetAPI    string `json:"target_api"`    // e.g. "POST /api/clientes"
	HandlerEvent string `json:"handler_event"` // e.g. "quando receber salvar_cliente"
	IsZombie     bool   `json:"is_zombie"`     // true if button has NO connected action
}

// EntityNode represents a domain entity in the Project Knowledge Graph.
type EntityNode struct {
	Name        string         `json:"name"`
	Plural      string         `json:"plural"`
	TableName   string         `json:"table_name"`
	Description string         `json:"description,omitempty"`
	Fields      []FieldMeta    `json:"fields"`
	Relations   []RelationMeta `json:"relations"`
	ReadRoles   []string       `json:"read_roles"`
	WriteRoles  []string       `json:"write_roles"`
	DeleteRoles []string       `json:"delete_roles"`
	HasAudit    bool           `json:"has_audit"`
	HasSoftDel  bool           `json:"has_soft_delete"`
	FilePath    string         `json:"file_path,omitempty"`
}

// ComponentNode represents a UI component with state and actions.
type ComponentNode struct {
	Name      string          `json:"name"`
	Kind      string          `json:"kind"` // "tabela", "formulario", "cards", "kanban", "filtros", "modal", "stats"
	EntityRef string          `json:"entity_ref,omitempty"`
	Actions   []ActionBinding `json:"actions"`
	Ejected   bool            `json:"ejected"`
	FilePath  string          `json:"file_path,omitempty"`
}

// APINode represents a backend endpoint route.
type APINode struct {
	Method       string   `json:"method"` // GET, POST, PUT, DELETE
	Path         string   `json:"path"`   // e.g. "/api/clientes"
	EntityRef    string   `json:"entity_ref"`
	HandlerName  string   `json:"handler_name"`
	AllowedRoles []string `json:"allowed_roles"`
	FilePath     string   `json:"file_path,omitempty"`
}

// PageNode represents a page in the web app.
type PageNode struct {
	Name          string          `json:"name"`
	Path          string          `json:"path"` // e.g. "/clientes"
	Title         string          `json:"title"`
	EntityRef     string          `json:"entity_ref,omitempty"`
	Layout        string          `json:"layout"`     // "dashboard", "site", "minimal"
	ViewTypes     []string        `json:"view_types"` // "tabela", "cards", "kanban", "grafico"
	Components    []ComponentNode `json:"components"`
	ConnectedAPIs []string        `json:"connected_apis"`
	AllowedRoles  []string        `json:"allowed_roles"`
	FilePath      string          `json:"file_path,omitempty"`
}

// MetricNode represents a dashboard metric card or chart.
type MetricNode struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	EntityRef   string `json:"entity_ref"`
	Aggregation string `json:"aggregation"` // "count", "sum", "avg"
	Field       string `json:"field,omitempty"`
	Format      string `json:"format"`               // "dinheiro", "numero", "percentual"
	ChartType   string `json:"chart_type,omitempty"` // "card", "barras", "linhas", "pizza"
}

// CapabilityStatus describes implementation state of a capability.
type CapabilityStatus string

const (
	CapStable       CapabilityStatus = "stable"
	CapImplemented  CapabilityStatus = "implemented"
	CapExperimental CapabilityStatus = "experimental"
	CapPlanned      CapabilityStatus = "planned"
)

// CapabilityNode represents a functional module.
type CapabilityNode struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	Status       CapabilityStatus `json:"status"`
	Dependencies []string         `json:"dependencies"`
}

// ProjectKnowledgeGraph maintains complete semantic awareness of a Germanio application.
type ProjectKnowledgeGraph struct {
	ProjectName  string                     `json:"project_name"`
	ProjectType  string                     `json:"project_type"`
	Entities     map[string]*EntityNode     `json:"entities"`
	Pages        map[string]*PageNode       `json:"pages"`
	Components   map[string]*ComponentNode  `json:"components"`
	APIs         map[string]*APINode        `json:"apis"`
	Metrics      map[string]*MetricNode     `json:"metrics"`
	Capabilities map[string]*CapabilityNode `json:"capabilities"`
	Roles        []string                   `json:"roles"`
}

// NewProjectKnowledgeGraph creates an empty graph.
func NewProjectKnowledgeGraph(name, pType string) *ProjectKnowledgeGraph {
	return &ProjectKnowledgeGraph{
		ProjectName:  name,
		ProjectType:  pType,
		Entities:     make(map[string]*EntityNode),
		Pages:        make(map[string]*PageNode),
		Components:   make(map[string]*ComponentNode),
		APIs:         make(map[string]*APINode),
		Metrics:      make(map[string]*MetricNode),
		Capabilities: make(map[string]*CapabilityNode),
		Roles:        []string{"Administrador", "Usuario"},
	}
}

// AddEntity registers or updates an entity.
func (g *ProjectKnowledgeGraph) AddEntity(e *EntityNode) {
	if e.Plural == "" {
		e.Plural = e.Name + "s"
	}
	if e.TableName == "" {
		e.TableName = strings.ToLower(e.Plural)
	}
	g.Entities[strings.ToLower(e.Name)] = e
}

// FindEntity looks up an entity by singular or plural name.
func (g *ProjectKnowledgeGraph) FindEntity(name string) *EntityNode {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if e, ok := g.Entities[normalized]; ok {
		return e
	}
	// Try match by plural or singular
	for _, e := range g.Entities {
		if strings.ToLower(e.Name) == normalized || strings.ToLower(e.Plural) == normalized {
			return e
		}
	}
	return nil
}

// AddPage registers a page and its components.
func (g *ProjectKnowledgeGraph) AddPage(p *PageNode) {
	g.Pages[p.Path] = p
	for i := range p.Components {
		c := &p.Components[i]
		g.Components[c.Name] = c
	}
}

// FindPage looks up a page by path or title/name.
func (g *ProjectKnowledgeGraph) FindPage(pathOrName string) *PageNode {
	query := strings.ToLower(strings.TrimSpace(pathOrName))
	if !strings.HasPrefix(query, "/") {
		query = "/" + query
	}
	if p, ok := g.Pages[query]; ok {
		return p
	}
	for _, p := range g.Pages {
		if strings.ToLower(p.Name) == strings.TrimPrefix(query, "/") || strings.ToLower(p.Title) == strings.TrimPrefix(query, "/") {
			return p
		}
	}
	return nil
}

// AddAPI registers an API route.
func (g *ProjectKnowledgeGraph) AddAPI(a *APINode) {
	key := fmt.Sprintf("%s %s", a.Method, a.Path)
	g.APIs[key] = a
}

// AddMetric registers a dashboard metric.
func (g *ProjectKnowledgeGraph) AddMetric(m *MetricNode) {
	g.Metrics[m.ID] = m
}

// AddCapability registers a capability node.
func (g *ProjectKnowledgeGraph) AddCapability(c *CapabilityNode) {
	g.Capabilities[c.ID] = c
}

// PageExplanation provides human-readable deep explanation of a page.
type PageExplanation struct {
	PageName       string
	Path           string
	Entity         string
	DatabaseTable  string
	BackendService string
	APIRoutes      []string
	AllowedRoles   []string
	Components     []string
	Actions        []string
	RelatedPages   []string
}

// ExplainPage generates explanation for a given page name or path.
func (g *ProjectKnowledgeGraph) ExplainPage(nameOrPath string) (*PageExplanation, error) {
	p := g.FindPage(nameOrPath)
	if p == nil {
		// Suggest similar pages
		var available []string
		for _, page := range g.Pages {
			available = append(available, page.Path)
		}
		return nil, fmt.Errorf("Página '%s' não encontrada no projeto. Páginas disponíveis: %s", nameOrPath, strings.Join(available, ", "))
	}

	exp := &PageExplanation{
		PageName:     p.Name,
		Path:         p.Path,
		Entity:       p.EntityRef,
		AllowedRoles: p.AllowedRoles,
	}

	if exp.Entity != "" {
		ent := g.FindEntity(exp.Entity)
		if ent != nil {
			exp.DatabaseTable = ent.TableName
			exp.BackendService = ent.Name + "Service"
		}
	}

	for _, a := range g.APIs {
		if a.EntityRef == p.EntityRef || strings.Contains(a.Path, strings.TrimPrefix(p.Path, "/")) {
			exp.APIRoutes = append(exp.APIRoutes, fmt.Sprintf("%s %s", a.Method, a.Path))
		}
	}

	for _, c := range p.Components {
		exp.Components = append(exp.Components, fmt.Sprintf("%s (%s)", c.Name, c.Kind))
		for _, act := range c.Actions {
			status := "✓ ativa"
			if act.IsZombie {
				status = "⚠️ sem handler"
			}
			exp.Actions = append(exp.Actions, fmt.Sprintf("%s -> %s [%s]", act.Label, act.TargetAPI, status))
		}
	}

	// Related pages
	for _, other := range g.Pages {
		if other.Path != p.Path {
			if other.EntityRef == p.EntityRef || other.Path == "/dashboard" {
				exp.RelatedPages = append(exp.RelatedPages, other.Path)
			}
		}
	}

	return exp, nil
}

// RenderTextTree outputs a visual hierarchy tree of the entire project knowledge graph.
func (g *ProjectKnowledgeGraph) RenderTextTree() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Projeto: %s (%s)\n", g.ProjectName, g.ProjectType))
	b.WriteString("│\n")

	// Capabilities
	b.WriteString("├── Capabilities Ativas\n")
	var caps []string
	for id := range g.Capabilities {
		caps = append(caps, id)
	}
	sort.Strings(caps)
	for i, id := range caps {
		c := g.Capabilities[id]
		prefix := "│   ├──"
		if i == len(caps)-1 {
			prefix = "│   └──"
		}
		b.WriteString(fmt.Sprintf("%s %s [%s]: %s\n", prefix, c.ID, c.Status, c.Name))
	}
	b.WriteString("│\n")

	// Entities
	b.WriteString("├── Entidades de Domínio\n")
	var entNames []string
	for name := range g.Entities {
		entNames = append(entNames, name)
	}
	sort.Strings(entNames)

	for i, name := range entNames {
		ent := g.Entities[name]
		isLastEnt := i == len(entNames)-1
		ePrefix := "│   ├──"
		if isLastEnt {
			ePrefix = "│   └──"
		}
		b.WriteString(fmt.Sprintf("%s %s (Tabela: %s)\n", ePrefix, ent.Name, ent.TableName))
		// Fields
		b.WriteString(fmt.Sprintf("│   │   ├── Campos (%d): ", len(ent.Fields)))
		var fieldStrs []string
		for _, f := range ent.Fields {
			fieldStrs = append(fieldStrs, fmt.Sprintf("%s:%s", f.Name, f.Type))
		}
		b.WriteString(strings.Join(fieldStrs, ", ") + "\n")

		// Associated Pages
		var linkedPages []string
		for _, p := range g.Pages {
			if strings.EqualFold(p.EntityRef, ent.Name) {
				linkedPages = append(linkedPages, p.Path)
			}
		}
		if len(linkedPages) > 0 {
			b.WriteString(fmt.Sprintf("│   │   ├── Páginas: %s\n", strings.Join(linkedPages, ", ")))
		}

		// Associated APIs
		var linkedAPIs []string
		for _, a := range g.APIs {
			if strings.EqualFold(a.EntityRef, ent.Name) {
				linkedAPIs = append(linkedAPIs, fmt.Sprintf("%s %s", a.Method, a.Path))
			}
		}
		if len(linkedAPIs) > 0 {
			b.WriteString(fmt.Sprintf("│   │   └── APIs: %s\n", strings.Join(linkedAPIs, ", ")))
		}
	}
	b.WriteString("│\n")

	// Pages
	b.WriteString("└── Páginas e Telas\n")
	var pagePaths []string
	for path := range g.Pages {
		pagePaths = append(pagePaths, path)
	}
	sort.Strings(pagePaths)
	for i, path := range pagePaths {
		p := g.Pages[path]
		pPrefix := "    ├──"
		if i == len(pagePaths)-1 {
			pPrefix = "    └──"
		}
		b.WriteString(fmt.Sprintf("%s %s — %s (Layout: %s)\n", pPrefix, p.Path, p.Title, p.Layout))
		for _, c := range p.Components {
			b.WriteString(fmt.Sprintf("        ├── Componente: %s [%s]\n", c.Name, c.Kind))
		}
	}

	return b.String()
}
