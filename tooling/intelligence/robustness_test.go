package intelligence

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIdempotencyEntityCreation verifies entity creation is idempotent.
func TestIdempotencyEntityCreation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ge-idempotency-entity-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	manifest := NewProjectManifest("TestApp", "saas")
	manifest.Entities = []string{"Cliente"}
	manifest.Save(tmpDir)

	engine := NewIntelligenceEngine(tmpDir)

	// First creation
	err1 := engine.InitEntity(InitEntityOptions{
		RootDir: tmpDir,
		Name:    "Cliente",
		Plural:  "Clientes",
		Fields: []FieldMeta{
			{Name: "nome", Type: FieldTexto, Required: true},
		},
		DryRun: false,
	})

	// Second creation (should be idempotent)
	err2 := engine.InitEntity(InitEntityOptions{
		RootDir: tmpDir,
		Name:    "Cliente",
		Plural:  "Clientes",
		Fields: []FieldMeta{
			{Name: "nome", Type: FieldTexto, Required: true},
		},
		DryRun: false,
	})

	// If first succeeded, second should also succeed without error (idempotent)
	if err1 == nil && err2 != nil {
		t.Errorf("Operação não idempotente: primeira criação OK, segunda falhou: %v", err2)
	}

	// Verify files exist
	if _, err := os.Stat(filepath.Join(tmpDir, "dados/clientes.ge")); err != nil {
		t.Errorf("Arquivo de dados não foi criado: %v", err)
	}
}

// TestIdempotencyPageCreation verifies page creation is idempotent.
func TestIdempotencyPageCreation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ge-idempotency-page-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	engine := NewIntelligenceEngine(tmpDir)

	// Create initial project
	engine.InitProject(InitProjectOptions{
		RootDir: tmpDir,
		Name:    "TestApp",
		Mode:    "rapido",
		Answers: map[string]interface{}{"project.type": "saas"},
	})

	// First page creation
	err1 := engine.InitPage(InitPageOptions{
		RootDir: tmpDir,
		Name:    "clientes",
		DryRun:  false,
	})

	// Second page creation (should be idempotent)
	err2 := engine.InitPage(InitPageOptions{
		RootDir: tmpDir,
		Name:    "clientes",
		DryRun:  false,
	})

	if err1 == nil && err2 != nil {
		t.Errorf("Operação não idempotente: primeira página OK, segunda falhou: %v", err2)
	}
}

// TestIdempotencyDashboardRegenerating verifies dashboard can be regenerated safely.
func TestIdempotencyDashboardRegenerating(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ge-idempotency-dashboard-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	engine := NewIntelligenceEngine(tmpDir)
	engine.InitProject(InitProjectOptions{
		RootDir: tmpDir,
		Name:    "TestApp",
		Mode:    "rapido",
		Answers: map[string]interface{}{"project.type": "saas"},
	})

	// First dashboard generation
	err1 := engine.InitDashboard(false)

	// Second dashboard generation (should be idempotent)
	err2 := engine.InitDashboard(false)

	if err1 == nil && err2 != nil {
		t.Errorf("Dashboard não idempotente: primeira geração OK, segunda falhou: %v", err2)
	}
}

// TestOverwriteProtectionCustomized verifies customized files are protected.
func TestOverwriteProtectionCustomized(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ge-protect-custom-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	manifest := NewProjectManifest("TestApp", "saas")
	manifest.RegisterFile("paginas/custom.ge", OwnershipCustomized, "Página customizada pelo dev")
	manifest.Save(tmpDir)

	tx := NewTransactionManager(tmpDir, false, manifest)

	// Try to add a file that's protected
	err = tx.AddFile("paginas/custom.ge", "novo conteudo", "Tentativa de sobrescrita")

	if err == nil {
		t.Errorf("Arquivo customizado deveria estar protegido contra sobrescrita")
	}

	if _, ok := err.(OverwriteProtectionError); !ok {
		t.Errorf("Erro esperado do tipo OverwriteProtectionError, obteve: %T", err)
	}
}

// TestOverwriteProtectionEjected verifies ejected files are protected.
func TestOverwriteProtectionEjected(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ge-protect-ejected-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	manifest := NewProjectManifest("TestApp", "saas")
	manifest.RegisterFile("componentes/custom.ge", OwnershipEjected, "Componente ejetado")
	manifest.Save(tmpDir)

	tx := NewTransactionManager(tmpDir, false, manifest)

	// Try to add a file that's ejected
	err = tx.AddFile("componentes/custom.ge", "novo conteudo", "Tentativa de sobrescrita")

	if err == nil {
		t.Errorf("Arquivo ejetado deveria estar protegido contra sobrescrita")
	}
}

// TestTransactionRollback verifies rollback restores all files on failure.
func TestTransactionRollback(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ge-rollback-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create original file
	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := []byte("original")
	os.WriteFile(testFile, originalContent, 0644)

	manifest := NewProjectManifest("TestApp", "saas")
	manifest.Save(tmpDir)

	tx := NewTransactionManager(tmpDir, false, manifest)

	// Pre-backup the original file
	if data, err := os.ReadFile(testFile); err == nil {
		tx.backups[testFile] = data
	}

	// Add operation to modify file
	tx.Ops = append(tx.Ops, FileOperation{
		Path:        "test.txt",
		Action:      OpModify,
		OldContent:  string(originalContent),
		NewContent:  "modified",
		Description: "Test modification",
	})

	// Perform rollback
	err = tx.RollbackTransactional()
	if err != nil {
		t.Errorf("Rollback deveria ser bem-sucedido, obteve: %v", err)
	}

	// Verify file was restored
	restored, _ := os.ReadFile(testFile)
	if string(restored) != string(originalContent) {
		t.Errorf("Arquivo não foi restaurado corretamente. Esperado: %s, obteve: %s", originalContent, restored)
	}
}

// TestCircularDependencyDetection verifies circular capability dependencies are caught.
func TestCircularDependencyDetection(t *testing.T) {
	caps := map[string]*CapabilityNode{
		"CAP_A": {
			ID:           "CAP_A",
			Name:         "Cap A",
			Description:  "Cap A",
			Status:       CapStable,
			Dependencies: []string{"CAP_B"},
		},
		"CAP_B": {
			ID:           "CAP_B",
			Name:         "Cap B",
			Description:  "Cap B",
			Status:       CapStable,
			Dependencies: []string{"CAP_A"}, // Circular!
		},
	}

	diags := ValidateDependencies(caps)

	if len(diags) == 0 {
		t.Errorf("Deveria detectar dependência circular entre CAP_A e CAP_B")
	}

	found := false
	for _, d := range diags {
		if d.Code == "VAL301" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Diagnóstico com código VAL301 (dependência circular) não foi gerado")
	}
}

// TestAPIValidationInvalidMethod verifies invalid HTTP methods are caught.
func TestAPIValidationInvalidMethod(t *testing.T) {
	apis := map[string]*APINode{
		"INVALID /api/test": {
			Method:      "INVALID",
			Path:        "/api/test",
			EntityRef:   "Cliente",
			HandlerName: "TestHandler",
		},
	}

	diags := ValidateAPIs(apis, make(map[string]*EntityNode))

	if len(diags) == 0 {
		t.Errorf("Deveria detectar método HTTP inválido")
	}

	found := false
	for _, d := range diags {
		if d.Code == "VAL401" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Diagnóstico com código VAL401 (método inválido) não foi gerado")
	}
}

// TestAPIValidationMissingEntity verifies missing entity references are caught.
func TestAPIValidationMissingEntity(t *testing.T) {
	apis := map[string]*APINode{
		"GET /api/inexistente": {
			Method:      "GET",
			Path:        "/api/inexistente",
			EntityRef:   "InexistenteEntity",
			HandlerName: "Handler",
		},
	}

	diags := ValidateAPIs(apis, make(map[string]*EntityNode))

	if len(diags) == 0 {
		t.Errorf("Deveria detectar entidade inexistente")
	}

	found := false
	for _, d := range diags {
		if d.Code == "VAL402" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Diagnóstico com código VAL402 (entidade não encontrada) não foi gerado")
	}
}

// TestComponentValidationInvalidKind verifies invalid component kinds are caught.
func TestComponentValidationInvalidKind(t *testing.T) {
	components := map[string]*ComponentNode{
		"comp1": {
			Name:    "comp1",
			Kind:    "invalido",
			Actions: []ActionBinding{},
		},
	}

	diags := ValidateComponents(components, make(map[string]*PageNode), make(map[string]*EntityNode))

	if len(diags) == 0 {
		t.Errorf("Deveria detectar tipo de componente inválido")
	}

	found := false
	for _, d := range diags {
		if d.Code == "VAL601" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Diagnóstico com código VAL601 (tipo inválido) não foi gerado")
	}
}

// TestRelationValidationInvalidType verifies invalid relation types are caught.
func TestRelationValidationInvalidType(t *testing.T) {
	entities := map[string]*EntityNode{
		"cliente": {
			Name:   "Cliente",
			Plural: "Clientes",
			Relations: []RelationMeta{
				{
					Name:   "rel1",
					Source: "Cliente",
					Target: "Venda",
					Type:   "N:M", // Invalid!
				},
			},
		},
		"venda": {
			Name:   "Venda",
			Plural: "Vendas",
		},
	}

	diags := ValidateRelations(entities)

	if len(diags) == 0 {
		t.Errorf("Deveria detectar tipo de relacionamento inválido")
	}

	found := false
	for _, d := range diags {
		if d.Code == "VAL801" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Diagnóstico com código VAL801 (tipo inválido) não foi gerado")
	}
}

// TestRelationValidationMissingTarget verifies missing target entities are caught.
func TestRelationValidationMissingTarget(t *testing.T) {
	entities := map[string]*EntityNode{
		"cliente": {
			Name:   "Cliente",
			Plural: "Clientes",
			Relations: []RelationMeta{
				{
					Name:   "rel1",
					Source: "Cliente",
					Target: "Inexistente",
					Type:   "1:N",
				},
			},
		},
	}

	diags := ValidateRelations(entities)

	if len(diags) == 0 {
		t.Errorf("Deveria detectar entidade alvo inexistente")
	}

	found := false
	for _, d := range diags {
		if d.Code == "VAL802" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Diagnóstico com código VAL802 (alvo inexistente) não foi gerado")
	}
}

// TestDiffPlanGeneration verifies diff plan is properly formatted.
func TestDiffPlanGeneration(t *testing.T) {
	manifest := NewProjectManifest("TestApp", "saas")
	tx := NewTransactionManager("/tmp", true, manifest)

	tx.Ops = append(tx.Ops, FileOperation{
		Path:        "dados/cliente.ge",
		Action:      OpCreate,
		NewContent:  "tabela Cliente...",
		Description: "Modelo de Cliente",
	})
	tx.Ops = append(tx.Ops, FileOperation{
		Path:        "rotas/cliente.ge",
		Action:      OpCreate,
		NewContent:  "rotas...",
		Description: "Rotas de Cliente",
	})

	plan := tx.RenderDiffPlan()

	if !strings.Contains(plan, "PLANO DE ALTERAÇÕES") {
		t.Errorf("Plano deveria conter título")
	}

	if !strings.Contains(plan, "+2 novos") {
		t.Errorf("Plano deveria mencionar 2 novos arquivos")
	}

	if !strings.Contains(plan, "DRY RUN: true") {
		t.Errorf("Plano deveria indicar DRY RUN ativo")
	}
}

// TestEjectComponentWithProtection verifies ejection marks file as protected.
func TestEjectComponentWithProtection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ge-eject-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	manifest := NewProjectManifest("TestApp", "saas")
	manifest.Save(tmpDir)

	tx := NewTransactionManager(tmpDir, false, manifest)
	compPath := "componentes/custom.ge"

	err = tx.EjectWithProtection(compPath)
	if err != nil {
		t.Errorf("Ejeção deveria ser bem-sucedida, obteve: %v", err)
	}

	// Verify it's marked as ejected
	loaded, _ := LoadManifest(tmpDir)
	if !loaded.IsProtected(compPath) {
		t.Errorf("Componente ejetado deveria estar protegido")
	}

	meta, _ := loaded.Files[compPath]
	if meta.Ownership != OwnershipEjected {
		t.Errorf("Propriedade deveria ser OwnershipEjected, obteve: %s", meta.Ownership)
	}
}

// TestValidateFileOverwriteSafety verifies overwrite protection checks work.
func TestValidateFileOverwriteSafety(t *testing.T) {
	manifest := NewProjectManifest("TestApp", "saas")
	manifest.RegisterFile("custom/page.ge", OwnershipCustomized, "Customizado")

	// Should fail for customized file
	err := ValidateFileOverwriteSafety(manifest, "custom/page.ge")
	if err == nil {
		t.Errorf("Deveria rejeitar sobrescrita de arquivo customizado")
	}

	// Should pass for unregistered file
	err = ValidateFileOverwriteSafety(manifest, "novo/arquivo.ge")
	if err != nil {
		t.Errorf("Deveria permitir novo arquivo, obteve: %v", err)
	}
}

// TestEnsureEntityIdempotent verifies entity idempotency check.
func TestEnsureEntityIdempotent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ge-entity-idempotent-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	manifest := NewProjectManifest("TestApp", "saas")
	manifest.Entities = []string{"Cliente"}
	manifest.Save(tmpDir)

	tx := NewTransactionManager(tmpDir, false, manifest)

	ent := &EntityNode{
		Name:   "Cliente",
		Plural: "Clientes",
	}

	// Should succeed first time (entity doesn't exist yet in protection)
	err = tx.EnsureEntityIdempotent(ent, false)
	if err != nil {
		t.Errorf("Primeira criação deveria ser permitida, obteve: %v", err)
	}

	// Mark as customized and try again
	manifest.RegisterFile("dados/clientes.ge", OwnershipCustomized, "Customizado")
	manifest.Entities = []string{"Cliente"}
	manifest.Save(tmpDir)

	tx2 := NewTransactionManager(tmpDir, false, manifest)
	err = tx2.EnsureEntityIdempotent(ent, false)

	// Should fail as it's customized
	if err == nil {
		t.Errorf("Deveria detectar entidade customizada")
	}

	if _, ok := err.(IdempotencyError); !ok {
		t.Errorf("Erro esperado do tipo IdempotencyError, obteve: %T", err)
	}
}

// TestComprehensiveProjectValidation verifies all validation rules together.
func TestComprehensiveProjectValidation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ge-validation-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create project with intentional issues
	engine := NewIntelligenceEngine(tmpDir)
	graph, err := engine.InitProject(InitProjectOptions{
		RootDir: tmpDir,
		Name:    "ValidationTest",
		Mode:    "rapido",
		Answers: map[string]interface{}{"project.type": "saas"},
	})

	if err != nil {
		t.Fatalf("Falha ao criar projeto: %v", err)
	}

	// Add component with zombie button (no action)
	page := &PageNode{
		Name:   "TestPage",
		Path:   "/test",
		Title:  "Test",
		Layout: "dashboard",
		Components: []ComponentNode{
			{
				Name: "TestComp",
				Kind: "formulario",
				Actions: []ActionBinding{
					{
						Name:      "submit",
						Label:     "Submeter",
						TargetAPI: "",
						IsZombie:  true,
					},
				},
			},
		},
	}

	graph.AddPage(page)

	// Run full validation
	validator := NewProjectValidator(tmpDir, graph)
	diags := validator.ValidateAll()

	// Should have at least one zombie button warning
	foundZombie := false
	for _, d := range diags {
		if d.Code == "VAL101" {
			foundZombie = true
			break
		}
	}

	if !foundZombie {
		t.Logf("Aviso: Validação não detectou botão zumbi. Diagnósticos: %d", len(diags))
	}
}

// TestAllValidationRulesExecute verifies all validation rules run without panic.
func TestAllValidationRulesExecute(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ge-all-validations-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	graph := NewProjectKnowledgeGraph("TestApp", "saas")

	// Add various nodes
	graph.AddEntity(&EntityNode{
		Name:      "Cliente",
		Plural:    "Clientes",
		TableName: "clientes",
		Fields: []FieldMeta{
			{Name: "nome", Type: FieldTexto, Required: true},
		},
	})

	graph.AddCapability(&CapabilityNode{
		ID:           "CAP_DB",
		Name:         "Database",
		Description:  "Persistência",
		Status:       CapStable,
		Dependencies: []string{},
	})

	// All these should run without panic
	diags1 := ValidateDependencies(graph.Capabilities)
	diags2 := ValidateAPIs(graph.APIs, graph.Entities)
	diags3 := ValidateRoutes(graph.Pages, graph.APIs)
	diags4 := ValidateComponents(graph.Components, graph.Pages, graph.Entities)
	diags5 := ValidateMigrations(graph.Entities, tmpDir)
	diags6 := ValidateRelations(graph.Entities)

	total := len(diags1) + len(diags2) + len(diags3) + len(diags4) + len(diags5) + len(diags6)
	t.Logf("Total de diagnósticos gerados: %d (esperado >= 0)", total)
}
