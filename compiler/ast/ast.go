package ast

import "github.com/flaviokalleu/germanio/compiler/diagnostics"

// Node is the base interface for all AST nodes.
type Node interface {
	NodeType() string
}

// Program is the root AST node.
type Program struct {
	// Germanio metadata is additive; the legacy full-stack AST stays intact.
	Source, Filename, Domain string
	System                   *System
	Theme                    *Theme
	Database                 *DatabaseConfig
	Auth                     *AuthConfig
	WhatsApp                 *WhatsAppConfig
	Email                    *EmailConfig
	Imports                  []*Import
	Models                   []*Model
	Screens                  []*Screen
	Events                   []*Event
	Actions                  []*Action
	Rules                    []*Rule
	Notifiers                []*Notifier
	Crons                    []*CronJob
	Env                      map[string]string
	Functions                []*FuncDecl
	Tests                    []*TestDecl
	Scripts                  []*Statement
	Routes                   []*CustomRoute
	Pages                    []*CustomPage
	SidebarItems             []*SidebarItem
	// Intent layer (compiler/ast/intencao.go)
	Intent *Intent
	// App is the resolved intent (nil for programs without intent phrases).
	App *App
}

func (p *Program) NodeType() string { return "Program" }

// Merge combines another program into this one (for imports).
func (p *Program) Merge(other *Program) {
	if other.Theme != nil && p.Theme == nil {
		p.Theme = other.Theme
	}
	if other.Auth != nil && p.Auth == nil {
		p.Auth = other.Auth
	}
	if other.WhatsApp != nil && p.WhatsApp == nil {
		p.WhatsApp = other.WhatsApp
	}
	if other.Email != nil && p.Email == nil {
		p.Email = other.Email
	}
	p.Models = append(p.Models, other.Models...)
	p.Screens = append(p.Screens, other.Screens...)
	p.Events = append(p.Events, other.Events...)
	p.Actions = append(p.Actions, other.Actions...)
	p.Rules = append(p.Rules, other.Rules...)
	p.Notifiers = append(p.Notifiers, other.Notifiers...)
	p.Crons = append(p.Crons, other.Crons...)
	p.Functions = append(p.Functions, other.Functions...)
	p.Tests = append(p.Tests, other.Tests...)
	p.Scripts = append(p.Scripts, other.Scripts...)
	p.Routes = append(p.Routes, other.Routes...)
	p.Pages = append(p.Pages, other.Pages...)
	p.SidebarItems = append(p.SidebarItems, other.SidebarItems...)
	p.Intent = MergeIntent(p.Intent, other.Intent)
}

// ==================== System ====================

type System struct {
	Name string
}

func (s *System) NodeType() string { return "System" }

// ==================== Import ====================

type Import struct {
	Pos  diagnostics.Position
	What string
	Path string
	// Names: `importar produtos e pedidos do backend` — the data this file
	// uses from another part of the project. Path is then a folder or file
	// name taken from the project root (FromRoot) or a quoted path.
	Names    []string
	FromRoot bool
}

func (i *Import) NodeType() string { return "Import" }

// ==================== Theme ====================

type Theme struct {
	Primary    string
	Secondary  string
	Accent     string
	Dark       bool
	Sidebar    string
	Icon       string
	Font       string // custom font family
	Radius     string // border radius (e.g. "12px")
	Background string // custom background color
	CardBg     string // card background
	TextColor  string // text color
	Style      string // "glassmorphism", "flat", "neumorphism", "minimal"
	CustomCSS  string // raw CSS injection from user
}

func (t *Theme) NodeType() string { return "Theme" }

func DefaultTheme() *Theme {
	return &Theme{
		Primary: "#6366f1", Secondary: "#8b5cf6",
		Accent: "#f59e0b", Sidebar: "#1e1b4b",
		Font: "Inter", Radius: "12px", Style: "glassmorphism",
		Background: "#0f0b2d", CardBg: "rgba(255,255,255,0.05)",
		TextColor: "#e2e8f0",
	}
}

// ColorName maps color names (PT/EN) to hex values.
var ColorName = map[string]string{
	// PT
	"azul": "#3b82f6", "verde": "#22c55e", "vermelho": "#ef4444",
	"roxo": "#8b5cf6", "laranja": "#f97316", "rosa": "#ec4899",
	"amarelo": "#eab308", "ciano": "#06b6d4", "indigo": "#6366f1",
	"cinza": "#6b7280", "branco": "#ffffff", "preto": "#000000",
	"esmeralda": "#10b981", "ambar": "#f59e0b", "violeta": "#7c3aed",
	// EN
	"blue": "#3b82f6", "green": "#22c55e", "red": "#ef4444",
	"purple": "#8b5cf6", "orange": "#f97316", "pink": "#ec4899",
	"yellow": "#eab308", "cyan": "#06b6d4", "gray": "#6b7280",
	"white": "#ffffff", "black": "#000000", "emerald": "#10b981",
	"amber": "#f59e0b", "violet": "#7c3aed",
}

// ThemePreset applies a named preset to a theme.
func ThemePreset(name string) *Theme {
	switch name {
	case "moderno", "modern":
		return &Theme{
			Primary: "#6366f1", Secondary: "#8b5cf6", Accent: "#f59e0b",
			Sidebar: "#1e1b4b", Font: "Inter", Radius: "12px",
			Style: "glassmorphism", Background: "#0f0b2d",
			CardBg: "rgba(255,255,255,0.05)", TextColor: "#e2e8f0", Dark: true,
		}
	case "claro", "light":
		return &Theme{
			Primary: "#3b82f6", Secondary: "#6366f1", Accent: "#f59e0b",
			Sidebar: "#1e293b", Font: "Inter", Radius: "8px",
			Style: "flat", Background: "#f8fafc",
			CardBg: "#ffffff", TextColor: "#1e293b", Dark: false,
		}
	case "simples", "simple":
		return &Theme{
			Primary: "#2563eb", Secondary: "#4f46e5", Accent: "#059669",
			Sidebar: "#111827", Font: "system-ui", Radius: "6px",
			Style: "minimal", Background: "#ffffff",
			CardBg: "#f9fafb", TextColor: "#111827", Dark: false,
		}
	case "elegante", "elegant":
		return &Theme{
			Primary: "#7c3aed", Secondary: "#6d28d9", Accent: "#c084fc",
			Sidebar: "#0f0720", Font: "Inter", Radius: "16px",
			Style: "neumorphism", Background: "#1a1025",
			CardBg: "rgba(255,255,255,0.03)", TextColor: "#e8e0f0", Dark: true,
		}
	case "corporativo", "corporate":
		return &Theme{
			Primary: "#0f766e", Secondary: "#115e59", Accent: "#f59e0b",
			Sidebar: "#1e293b", Font: "Inter", Radius: "4px",
			Style: "flat", Background: "#f1f5f9",
			CardBg: "#ffffff", TextColor: "#334155", Dark: false,
		}
	default:
		return DefaultTheme()
	}
}

// ResolveColor converts a color name to hex, or returns the value as-is if already hex.
func ResolveColor(val string) string {
	if hex, ok := ColorName[val]; ok {
		return hex
	}
	return val
}

// ==================== Database ====================

type DatabaseConfig struct {
	Driver   string // sqlite, mysql, postgres
	Host     string
	Port     string
	Name     string
	User     string
	Password string
}

func (d *DatabaseConfig) NodeType() string { return "DatabaseConfig" }

func DefaultDatabase() *DatabaseConfig {
	return &DatabaseConfig{Driver: "sqlite"}
}

// ==================== Auth ====================

type AuthConfig struct {
	Enabled    bool
	UserModel  string   // model name for users (default: "usuario")
	LoginField string   // field used for login (default: "email")
	PassField  string   // password field (default: "senha")
	Roles      []string // available roles
	JWTSecret  string
}

func (a *AuthConfig) NodeType() string { return "AuthConfig" }

// ==================== WhatsApp ====================

type WhatsAppConfig struct {
	Enabled      bool
	DBPath       string
	Provider     string
	MultiSession bool
	Presence     bool
	QRCodeFlow   bool
}

func (w *WhatsAppConfig) NodeType() string { return "WhatsAppConfig" }

// ==================== Email ====================

type EmailConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
	Template string // HTML template for emails
}

func (e *EmailConfig) NodeType() string { return "EmailConfig" }

// ==================== Model ====================

// FieldRename: the field From is now called To (the data moves with it).
type FieldRename struct {
	From, To string
	Pos      diagnostics.Position
}

type Model struct {
	Name       string
	Icon       string
	Fields     []*Field
	SoftDelete bool
	// Internal models get no automatic REST endpoints nor generated screens;
	// only .ge code (rotas, funcoes) can read or write them.
	Internal bool
	// UniqueTogether / IndexTogether are composite constraints: unico(a, b).
	UniqueTogether [][]string
	IndexTogether  [][]string
	// Pairs: two reference fields that never hold the same record and whose
	// values, in any order, appear in one record only (GEP 0048).
	Pairs [][2]string
	// Renames are explicit field renames (renomeie nome para nome_completo):
	// the migration keeps the data. Discarded names fields that were removed
	// on purpose (descarte telefone): their old data is left alone.
	Renames   []FieldRename
	Discarded []string
	// ExpiresDays > 0 adds expires_at (data) defaulting to today + N days;
	// secret lookups ignore expired records.
	ExpiresDays int
	// Revocable adds revoked (booleano) and modelo.revogar(id).
	Revocable  bool
	Label      string // name used in messages ("Project")
	Pos        diagnostics.Position
	IsAuth     bool     // is this the auth user model?
	HasMany    []string // model names for 1:N relationships
	ManyToMany []string // model names for N:N relationships
}

func (m *Model) NodeType() string { return "Model" }

// ==================== Field ====================

type FieldType string

const (
	FieldTexto        FieldType = "texto"
	FieldNumero       FieldType = "numero"
	FieldInteiro      FieldType = "inteiro"
	FieldSegredo      FieldType = "segredo"
	FieldVisibilidade FieldType = "visibilidade"
	FieldLista        FieldType = "lista"
	FieldBranch       FieldType = "branch"
	FieldData         FieldType = "data"
	FieldBooleano     FieldType = "booleano"
	FieldEmail        FieldType = "email"
	FieldTelefone     FieldType = "telefone"
	FieldImagem       FieldType = "imagem"
	FieldArquivo      FieldType = "arquivo"
	FieldUpload       FieldType = "upload"
	FieldLink         FieldType = "link"
	FieldStatus       FieldType = "status"
	FieldDinheiro     FieldType = "dinheiro"
	FieldSenha        FieldType = "senha"
	FieldTextoLongo   FieldType = "texto_longo"
	FieldEnum         FieldType = "enum"
	FieldCPF          FieldType = "cpf"
	FieldCEP          FieldType = "cep"
	FieldCor          FieldType = "cor"
	FieldEstrelas     FieldType = "estrelas"
	FieldHora         FieldType = "hora"
	FieldDataHora     FieldType = "data_hora"
	FieldPercentual   FieldType = "percentual"
	FieldTags         FieldType = "tags"
	FieldURL          FieldType = "url"
	FieldMoeda        FieldType = "moeda"
	// FieldChavePublica: a public key (`chave pública`, GEP 0032): checked,
	// written in one canonical form, with its fingerprint derived.
	FieldChavePublica FieldType = "chave_publica"
)

type Field struct {
	Name string
	Type FieldType
	// TypeInferred: the type came from the name (docs/INTENCAO.md › Tipo pelo
	// nome), not from a declaration.
	TypeInferred bool
	Required     bool
	Unique       bool
	Default      string
	Reference    string   // pertence_a model
	EnumValues   []string // for enum type
	Index        bool

	// Declarative schema (nível 1/2). All are enforced by the runtime on
	// every create/update, whatever route or function performs it.
	Protected    bool     // senha protegida: hashed, never readable
	Hidden       bool     // oculto: never serialized
	Sealed       bool     // encrypted at rest, never compared: oculto text or a capability's credential (GEP 0049)
	Private      bool     // privado: shown only to its owner and administrators
	NumberedBy   string   // numerado por <entidade>: sequence per parent (holds the FK field after resolution)
	ListOf       string   // lista de <texto|entidade>
	ByName       string   // labels por nome: items are named by this field of ListOf
	System       bool     // maintained by the runtime (estado, fechada_em); never written by people
	Label        string   // spelling for people (accents kept): "descrição"
	Formatted    bool     // formatado: Markdown rendered safely on pages
	Immutable    bool     // imutavel: cannot change after creation
	Min, Max     *float64 // length for text, value for numbers
	Format       string   // formato "regex" (RE2)
	Validator    string   // valida funcao: .ge predicate
	Prefix       string   // prefixo of generated secrets
	HasDefault   bool
	DefaultValue any // typed default (= valor)
	Pos          diagnostics.Position
}

// IsSecret reports whether the field stores a hash or digest that must
// never leave the runtime (senha, segredo).
func (f *Field) IsSecret() bool { return f.Type == FieldSenha || f.Type == FieldSegredo || f.Protected }

func (f *Field) NodeType() string { return "Field" }

func (ft FieldType) SQLType() string {
	switch ft {
	case FieldNumero, FieldDinheiro, FieldEstrelas, FieldPercentual:
		return "REAL"
	case FieldBooleano, FieldInteiro:
		return "INTEGER"
	case FieldData:
		return "DATE" // a calendar day: no time of day
	case FieldDataHora:
		return "DATETIME"
	default:
		return "TEXT"
	}
}

// ==================== Screen ====================

type Screen struct {
	Name       string
	Title      string
	Public     bool   // accessible without login
	Requires   string // required role
	Components []*Component
}

func (s *Screen) NodeType() string { return "Screen" }

// ==================== Component ====================

type ComponentType string

const (
	CompList     ComponentType = "lista"
	CompShow     ComponentType = "mostrar"
	CompButton   ComponentType = "botao"
	CompForm     ComponentType = "formulario"
	CompChat     ComponentType = "chat"
	CompInput    ComponentType = "entrada"
	CompImage    ComponentType = "imagem"
	CompText     ComponentType = "texto"
	CompSearch   ComponentType = "busca"
	CompChart    ComponentType = "grafico"
	CompSelect   ComponentType = "selecionar"
	CompTextarea ComponentType = "area_texto"
)

type Component struct {
	Type       ComponentType
	Target     string
	Properties map[string]string
	Children   []*Component
}

func (c *Component) NodeType() string { return "Component" }

// ==================== Event ====================

type Event struct {
	Trigger   string
	Target    string
	ActionRef string
}

func (e *Event) NodeType() string { return "Event" }

// ==================== Action ====================

type Action struct {
	Name  string
	Steps []*ActionStep
}

func (a *Action) NodeType() string { return "Action" }

type ActionStep struct {
	Command string
	Args    []string
}

func (s *ActionStep) NodeType() string { return "ActionStep" }

// ==================== Rule ====================

type Rule struct {
	Field     string
	Operator  string
	Value     string
	Action    string
	ActionArg string
}

func (r *Rule) NodeType() string { return "Rule" }

// ==================== Notifier ====================

type Notifier struct {
	Trigger string
	Model   string
	Field   string
	Value   string
	SendTo  string
	Message string
	Subject string // email subject
	Channel string // whatsapp, email, webhook
}

func (n *Notifier) NodeType() string { return "Notifier" }

// ==================== CronJob ====================

type CronJob struct {
	Every  string // "1 hora", "30 minutos", etc
	Action string // what to do
	Target string // model or URL
}

func (c *CronJob) NodeType() string { return "CronJob" }

// ==================== Custom Route ====================

type CustomRoute struct {
	Pos     diagnostics.Position
	Method  string       // GET, POST, PUT, DELETE
	Path    string       // /api/relatorio, /api/custom/stats
	Handler []*Statement // code to execute
}

func (c *CustomRoute) NodeType() string { return "CustomRoute" }

// ==================== Custom Page ====================

type PageUIBlock interface {
	UIBlockType() string
}

type PageNavbar struct {
	Logo    string
	Brand   string
	Links   []*PageNavLink
	Buttons []*PageNavButton
}

func (p *PageNavbar) UIBlockType() string { return "Navbar" }

type PageNavLink struct {
	Label string
	URL   string
}

type PageNavButton struct {
	Label   string
	URL     string
	Primary bool
}

type PageHero struct {
	Badge       string
	Title       string
	Highlight   string
	Description string
	Buttons     []*PageHeroButton
	Command     string
	Mascot      string
	CodePreview *PageCodePreview
}

func (p *PageHero) UIBlockType() string { return "Hero" }

type PageHeroButton struct {
	Label   string
	URL     string
	Primary bool
}

type PageCodePreview struct {
	Filename string
	Code     string
}

type PageSection struct {
	Title      string
	Subtitle   string
	Columns    int
	Cards      []*PageCard
	CodeBlocks []*PageCodeBlock
	TextBlocks []string
}

func (p *PageSection) UIBlockType() string { return "Section" }

type PageCard struct {
	Icon          string
	Title         string
	Text          string
	Link          string
	Tag           string
	Code          string
	ButtonLabel   string
	ButtonURL     string
	ButtonPrimary bool
}

type PageCodeBlock struct {
	Title    string
	Language string
	Code     string
}

type PageFooter struct {
	Text      string
	Copyright string
	Links     []*PageNavLink
}

func (p *PageFooter) UIBlockType() string { return "Footer" }

type CustomPage struct {
	Path    string // URL path
	Title   string
	Content string // raw HTML content (fallback)
	Blocks  []PageUIBlock
}

func (c *CustomPage) NodeType() string { return "CustomPage" }

// ==================== Sidebar Item ====================

type SidebarItem struct {
	Label string
	Icon  string
	Link  string // model name, screen name, or URL
	Order int
}

func (s *SidebarItem) NodeType() string { return "SidebarItem" }

// ==================== Scripting/Logic AST ====================

// Expression represents any value expression.
type Expression struct {
	Pos      diagnostics.Position
	Type     string      // "literal", "variable", "binary", "unary", "call", "field_access", "list"
	Value    interface{} // for literals (string, float64, bool, nil)
	Name     string      // for variables and function calls
	Left     *Expression
	Right    *Expression
	Operator string        // +, -, *, /, ==, !=, >, <, >=, <=, e/and, ou/or
	Args     []*Expression // for function calls
	Object   string        // for field access (object.field)
	Field    string
	Elements []*Expression // for list literals
	Index    *Expression   // for array[index] access
	// Keys holds map literal keys, parallel to Elements ({chave: valor}).
	Keys []string
	// Target is the receiver of "member" (x.campo) and "method" (x.f(...))
	// when the receiver is an arbitrary expression rather than a plain name.
	Target *Expression
	// Canon is the canonical multilingual keyword for Name when it differs
	// from the spelling in the source (e.g. "texte" → "texto").
	Canon string
}

func (e *Expression) NodeType() string { return "Expression" }

// VarDecl represents: definir x = 10
type VarDecl struct {
	Mutable, Constant bool
	Annotation        string
	Name              string
	Value             Expression
}

func (v *VarDecl) NodeType() string { return "VarDecl" }

// Assignment represents: x = 10 or object.field = value
type Assignment struct {
	Target string
	Field  string // for object.field = value
	Value  Expression
	// TargetExpr is set for nested targets: a.b.c = v, lista[0] = v.
	TargetExpr *Expression
}

func (a *Assignment) NodeType() string { return "Assignment" }

// FuncDecl represents: funcao name(params) ... body
type FuncDecl struct {
	Pos                  diagnostics.Position
	Private              bool // Accessible only inside the declaring module in .ge.
	TypeParams           []string
	TypeParamConstraints []string // Parallel to TypeParams; "numeric" means numero.
	ParamTypes           []string
	ResultType           string
	Name                 string
	Params               []string
	Body                 []*Statement
}

func (f *FuncDecl) NodeType() string { return "FuncDecl" }

// TestDecl is a named, isolated .ge test. It never executes during ge rodar.
type TestDecl struct {
	Pos  diagnostics.Position
	Name string
	Body []*Statement
}

func (t *TestDecl) NodeType() string { return "TestDecl" }

// Statement represents any executable statement.
type Statement struct {
	Pos         diagnostics.Position
	Expr        *Expression
	UI          *UIElement
	Type        string // "var", "assign", "if", "for_each", "while", "repeat", "return", "break", "continue", "pause", "call", "print", "try"
	VarDecl     *VarDecl
	Assign      *Assignment
	If          *IfStmt
	ForEach     *ForEachStmt
	While       *WhileStmt
	Repeat      *RepeatStmt
	Return      *Expression
	Call        *FuncCall
	Print       *Expression
	Expect      *Expression
	Try         *TryStmt
	When        *WhenStmt
	Concurrency *ConcurrencyStmt
}

// WhenCase represents a single case in pattern matching:
// existe -> ...
// vazio -> ...
// aprovado -> ...
// ou -> ...
type WhenCase struct {
	Pos       diagnostics.Position
	Pattern   *Expression // nil for default (ou)
	IsDefault bool
	Body      []*Statement
}

// WhenStmt represents pattern matching or null-safe presence check:
//
//	quando status {
//	  aprovado -> liberar()
//	  ou -> revisar()
//	}
type WhenStmt struct {
	Target Expression
	Cases  []*WhenCase
}

// ConcurrencyStmt represents structured concurrency:
//
//	ao mesmo tempo {
//	  buscar_usuarios()
//	  buscar_pedidos()
//	}
type ConcurrencyStmt struct {
	Tasks []*Statement
}

// UIElement is the deterministic, client-only foundation of natural UI syntax.
// It has no access to server bindings or arbitrary JavaScript.
type UIElement struct {
	Kind, Text, Color, Radius string
}

func (s *Statement) NodeType() string { return "Statement" }

// IfStmt represents: se/if ... senao se/else if ... senao/else
type IfStmt struct {
	Condition Expression
	Body      []*Statement
	ElseIfs   []*ElseIfClause
	Else      []*Statement
}

// ElseIfClause represents a single else-if branch.
type ElseIfClause struct {
	Condition Expression
	Body      []*Statement
}

// ForEachStmt represents: para cada x em collection
type ForEachStmt struct {
	VarName    string
	Collection Expression
	Body       []*Statement
}

// WhileStmt represents: enquanto condition
type WhileStmt struct {
	Condition Expression
	Body      []*Statement
}

// RepeatStmt represents: repetir N vezes
type RepeatStmt struct {
	Count Expression
	Body  []*Statement
}

// FuncCall represents: name(args)
type FuncCall struct {
	Name   string
	Object string // for method calls: object.method(args)
	Args   []*Expression
}

func (f *FuncCall) NodeType() string { return "FuncCall" }

// TryStmt represents: tentar ... erro ...
type TryStmt struct {
	Body   []*Statement
	Catch  []*Statement
	ErrVar string
}
