package intelligence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// InitProjectOptions configures the creation of a complete project.
type InitProjectOptions struct {
	RootDir        string
	Name           string
	Mode           string // "guiado", "rapido", "prompt"
	Description    string // For prompt mode
	Answers        map[string]interface{}
	DryRun         bool
	NonInteractive bool
}

// InitEntityOptions configures creation of a domain entity.
type InitEntityOptions struct {
	RootDir string
	Name    string
	Plural  string
	Fields  []FieldMeta
	DryRun  bool
}

// InitPageOptions configures creation of a page.
type InitPageOptions struct {
	RootDir string
	Name    string
	Entity  string
	Views   []string
	DryRun  bool
}

// IntelligenceEngine provides the high-level API for all Germanio Init & Project Intelligence operations.
type IntelligenceEngine struct {
	RootDir string
}

// NewIntelligenceEngine creates an engine instance.
func NewIntelligenceEngine(rootDir string) *IntelligenceEngine {
	return &IntelligenceEngine{RootDir: rootDir}
}

func isSafeIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		if !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || (i > 0 && r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

func (e *IntelligenceEngine) ensureProjectCanInitialize() error {
	manifest, err := LoadManifest(e.RootDir)
	if err != nil || manifest != nil {
		return err
	}
	entries, err := os.ReadDir(e.RootDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() != ".germanio" && entry.Name() != ManifestFileName {
			return fmt.Errorf("diretório já contém %q, mas não possui manifesto Germanio; recuso sobrescrever um projeto existente", entry.Name())
		}
	}
	return nil
}

// InitProject orchestrates the full project creation flow.
func (e *IntelligenceEngine) InitProject(opts InitProjectOptions) (*ProjectKnowledgeGraph, error) {
	if err := e.ensureProjectCanInitialize(); err != nil {
		return nil, err
	}
	manifest, _ := LoadManifest(opts.RootDir)
	if manifest == nil {
		manifest = NewProjectManifest(opts.Name, "saas")
	}

	var entities []EntityNode
	projType := "saas"
	audience := "B2B"
	multiTenant := false
	dbEngine := "sqlite"
	primaryColor := "#00D9FF"
	preset := "saas_moderno"

	if opts.Mode == "prompt" && opts.Description != "" {
		intent := ParseNaturalDescription(opts.Description)
		projType = intent.ProjectType
		audience = intent.Audience
		multiTenant = intent.MultiTenant
		entities = intent.DetectedEntities
		manifest.Capabilities = intent.Capabilities
	} else {
		// Use answers or defaults
		if t, ok := opts.Answers["project.type"].(string); ok && t != "" {
			projType = t
		}
		if a, ok := opts.Answers["saas.audience"].(string); ok && a != "" {
			audience = a
		}
		if mt, ok := opts.Answers["tenant.enabled"].(bool); ok {
			multiTenant = mt
		}
		if db, ok := opts.Answers["database.engine"].(string); ok && db != "" {
			dbEngine = db
		}
		if p, ok := opts.Answers["design.preset"].(string); ok && p != "" {
			preset = p
		}

		// Default initial entities based on type
		if projType == "crm" {
			entities = []EntityNode{
				{
					Name:      "Cliente",
					Plural:    "Clientes",
					TableName: "clientes",
					Fields: []FieldMeta{
						{Name: "nome", Type: FieldTexto, Required: true, Searchable: true},
						{Name: "email", Type: FieldEmail, Required: true, Unique: true},
						{Name: "telefone", Type: FieldTelefone, Required: false},
						{Name: "status", Type: FieldStatus, Required: true, DefaultVal: "novo"},
					},
				},
				{
					Name:      "Venda",
					Plural:    "Vendas",
					TableName: "vendas",
					Fields: []FieldMeta{
						{Name: "cliente_nome", Type: FieldTexto, Required: true},
						{Name: "valor_total", Type: FieldDinheiro, Required: true},
						{Name: "data", Type: FieldData, Required: true},
						{Name: "status", Type: FieldStatus, Required: true, DefaultVal: "concluida"},
					},
				},
			}
		} else {
			entities = []EntityNode{
				{
					Name:      "Item",
					Plural:    "Itens",
					TableName: "itens",
					Fields: []FieldMeta{
						{Name: "nome", Type: FieldTexto, Required: true, Searchable: true},
						{Name: "descricao", Type: FieldTextoLongo, Required: false},
						{Name: "status", Type: FieldStatus, Required: true, DefaultVal: "ativo"},
					},
				},
			}
		}
	}

	manifest.Name = opts.Name
	manifest.Type = projType
	manifest.Audience = audience
	manifest.MultiTenant = multiTenant
	manifest.Database = dbEngine
	manifest.Design.Preset = preset
	manifest.Design.PrimaryColor = primaryColor
	for _, e := range entities {
		manifest.Entities = append(manifest.Entities, e.Name)
	}

	templateOpts := ProjectTemplateOptions{
		Name:        opts.Name,
		Type:        projType,
		Audience:    audience,
		MultiTenant: multiTenant,
		Database:    dbEngine,
		Roles:       manifest.Roles,
		Entities:    entities,
		Design:      manifest.Design,
	}

	files := GenerateProjectFiles(templateOpts)
	tx := NewTransactionManager(opts.RootDir, opts.DryRun, manifest)

	for path, content := range files {
		if err := tx.AddFile(path, content, fmt.Sprintf("Arquivo gerado para %s", path)); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Re-analyze and return graph
	analyzer := NewProjectAnalyzer(opts.RootDir)
	return analyzer.Analyze()
}

// InitEntity adds a new domain entity, its table, backend routes and UI page.
func (e *IntelligenceEngine) InitEntity(opts InitEntityOptions) error {
	manifest, _ := LoadManifest(e.RootDir)
	if manifest == nil {
		manifest = NewProjectManifest("GermanioApp", "saas")
	}

	plural := opts.Plural
	if plural == "" {
		plural = Pluralize(opts.Name)
	}

	ent := EntityNode{
		Name:      CleanIdentifier(opts.Name),
		Plural:    CleanIdentifier(plural),
		TableName: strings.ToLower(plural),
		Fields:    opts.Fields,
		ReadRoles: []string{"Administrador"},
	}
	if len(ent.Fields) == 0 {
		ent.Fields = []FieldMeta{
			{Name: "nome", Type: FieldTexto, Required: true, Searchable: true},
			{Name: "status", Type: FieldStatus, Required: true, DefaultVal: "ativo"},
		}
	}

	files := GenerateSingleEntityScaffold(ent, manifest.MultiTenant)
	tx := NewTransactionManager(e.RootDir, opts.DryRun, manifest)

	for path, content := range files {
		if err := tx.AddFile(path, content, fmt.Sprintf("Scaffold da entidade %s", ent.Name)); err != nil {
			return err
		}
	}

	// Update inicio.ge imports if exists
	inicioPath := filepath.Join(e.RootDir, "inicio.ge")
	if inicioData, err := os.ReadFile(inicioPath); err == nil {
		content := string(inicioData)
		lowerPlural := strings.ToLower(ent.Plural)
		dataImport := fmt.Sprintf("importar \"dados/%s.ge\"\n", lowerPlural)
		rotaImport := fmt.Sprintf("importar \"rotas/%s.ge\"\n", lowerPlural)
		paginaImport := fmt.Sprintf("importar \"paginas/%s.ge\"\n", lowerPlural)

		if !strings.Contains(content, dataImport) {
			content += dataImport
		}
		if !strings.Contains(content, rotaImport) {
			content += rotaImport
		}
		if !strings.Contains(content, paginaImport) {
			content += paginaImport
		}
		_ = tx.AddFile("inicio.ge", content, "Atualização de imports para nova entidade")
	}

	return tx.Commit()
}

// InitPage adds or updates a UI page connected to a domain entity.
func (e *IntelligenceEngine) InitPage(opts InitPageOptions) error {
	analyzer := NewProjectAnalyzer(e.RootDir)
	graph, err := analyzer.Analyze()
	if err != nil {
		return err
	}

	manifest, _ := LoadManifest(e.RootDir)
	cleanName := CleanIdentifier(opts.Name)
	ent := graph.FindEntity(cleanName)

	// Outside-in Domain proposal if entity does not exist!
	if ent == nil {
		if strings.EqualFold(opts.Name, "financeiro") || strings.EqualFold(opts.Name, "financas") {
			// Propose and create financial domain entities
			ent = &EntityNode{
				Name:      "Transacao",
				Plural:    "Transacoes",
				TableName: "transacoes",
				Fields: []FieldMeta{
					{Name: "descricao", Type: FieldTexto, Required: true, Searchable: true},
					{Name: "valor", Type: FieldDinheiro, Required: true},
					{Name: "tipo", Type: FieldTexto, Required: true, DefaultVal: "receita"},
					{Name: "data", Type: FieldData, Required: true},
					{Name: "status", Type: FieldStatus, Required: true, DefaultVal: "pago"},
				},
			}
			if err := e.InitEntity(InitEntityOptions{
				RootDir: e.RootDir,
				Name:    "Transacao",
				Plural:  "Transacoes",
				Fields:  ent.Fields,
				DryRun:  opts.DryRun,
			}); err != nil {
				return err
			}
		} else {
			// Create standard entity for the page
			ent = &EntityNode{
				Name:      cleanName,
				Plural:    cleanName + "s",
				TableName: strings.ToLower(cleanName) + "s",
				Fields: []FieldMeta{
					{Name: "nome", Type: FieldTexto, Required: true, Searchable: true},
					{Name: "status", Type: FieldStatus, Required: true, DefaultVal: "ativo"},
				},
			}
			if err := e.InitEntity(InitEntityOptions{
				RootDir: e.RootDir,
				Name:    cleanName,
				Fields:  ent.Fields,
				DryRun:  opts.DryRun,
			}); err != nil {
				return err
			}
		}
	}

	// Generate page
	pagePath := fmt.Sprintf("paginas/%s.ge", strings.ToLower(ent.Plural))
	pageContent := generateEntityPage(*ent, ProjectTemplateOptions{
		Name:     manifest.Name,
		Entities: []EntityNode{*ent},
		Design:   manifest.Design,
	})

	tx := NewTransactionManager(e.RootDir, opts.DryRun, manifest)
	if err := tx.AddFile(pagePath, pageContent, fmt.Sprintf("Página para %s", ent.Plural)); err != nil {
		return err
	}

	return tx.Commit()
}

// InitDashboard generates or refreshes the metrics dashboard based on real entities.
func (e *IntelligenceEngine) InitDashboard(dryRun bool) error {
	analyzer := NewProjectAnalyzer(e.RootDir)
	graph, err := analyzer.Analyze()
	if err != nil {
		return err
	}
	manifest, _ := LoadManifest(e.RootDir)
	if manifest == nil {
		manifest = NewProjectManifest("GermanioApp", "saas")
	}

	var entities []EntityNode
	for _, ent := range graph.Entities {
		entities = append(entities, *ent)
	}

	dashboardContent := generateDashboardPage(ProjectTemplateOptions{
		Name:     manifest.Name,
		Entities: entities,
		Design:   manifest.Design,
	})

	tx := NewTransactionManager(e.RootDir, dryRun, manifest)
	if err := tx.AddFile("paginas/dashboard.ge", dashboardContent, "Dashboard de métricas baseado no grafo de dados"); err != nil {
		return err
	}

	return tx.Commit()
}

// EjectComponent detaches a component from Germanio core management so the user can own it.
func (e *IntelligenceEngine) EjectComponent(compName string) (string, error) {
	if !isSafeIdentifier(compName) {
		return "", fmt.Errorf("nome de componente inválido %q: use um identificador sem espaços, barras ou pontuação", compName)
	}
	manifest, _ := LoadManifest(e.RootDir)
	if manifest == nil {
		manifest = NewProjectManifest("GermanioApp", "saas")
	}

	targetDir := filepath.Join(e.RootDir, "componentes")
	_ = os.MkdirAll(targetDir, 0755)

	targetFile := filepath.Join(targetDir, fmt.Sprintf("%s.ge", strings.ToLower(compName)))
	content := fmt.Sprintf("# Componente ejetado: %s\n# Este arquivo agora é de propriedade direta do desenvolvedor (user-owned)\n\ncomponente %s\n  estilo \"personalizado\"\n", compName, compName)

	if err := os.WriteFile(targetFile, []byte(content), 0644); err != nil {
		return "", err
	}

	manifest.RegisterFile(fmt.Sprintf("componentes/%s.ge", strings.ToLower(compName)), OwnershipEjected, fmt.Sprintf("Componente ejetado %s", compName))
	_ = manifest.Save(e.RootDir)

	return targetFile, nil
}

// ExportSpec exports the project intelligence state to a .geinit specification file.
func (e *IntelligenceEngine) ExportSpec(exportPath string) error {
	manifest, err := LoadManifest(e.RootDir)
	if err != nil || manifest == nil {
		return fmt.Errorf("Nenhum projeto Germanio ativo encontrado para exportar")
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(exportPath, data, 0644)
}

// ImportSpec recreates a project from a .geinit specification file.
func (e *IntelligenceEngine) ImportSpec(importPath string) error {
	data, err := os.ReadFile(importPath)
	if err != nil {
		return fmt.Errorf("Falha ao ler arquivo de especificação: %w", err)
	}
	var manifest ProjectManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("Especificação .geinit inválida: %w", err)
	}

	var entities []EntityNode
	for _, entName := range manifest.Entities {
		if !isSafeIdentifier(entName) {
			return fmt.Errorf("especificação .geinit contém entidade inválida %q", entName)
		}
		plural := Pluralize(entName)
		entities = append(entities, EntityNode{
			Name:      entName,
			Plural:    plural,
			TableName: strings.ToLower(plural),
			Fields: []FieldMeta{
				{Name: "nome", Type: FieldTexto, Required: true, Searchable: true},
				{Name: "status", Type: FieldStatus, Required: true, DefaultVal: "ativo"},
			},
		})
	}

	files := GenerateProjectFiles(ProjectTemplateOptions{
		Name:        manifest.Name,
		Type:        manifest.Type,
		Audience:    manifest.Audience,
		MultiTenant: manifest.MultiTenant,
		Database:    manifest.Database,
		Roles:       manifest.Roles,
		Entities:    entities,
		Design:      manifest.Design,
	})

	tx := NewTransactionManager(e.RootDir, false, &manifest)
	for p, content := range files {
		if err := tx.AddFile(p, content, "Importado de especificação .geinit"); err != nil {
			return err
		}
	}

	return tx.Commit()
}
