package intelligence

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAdaptiveQuestionEngine(t *testing.T) {
	engine := NewAdaptiveQuestionEngine()
	if len(engine.Questions) == 0 {
		t.Fatalf("Esperado árvore de perguntas não vazia")
	}

	// Initial answers
	answers := map[string]interface{}{
		"project.type": "saas",
	}

	active := engine.GetActiveQuestions(answers)
	foundAudience := false
	for _, q := range active {
		if q.ID == "saas.audience" {
			foundAudience = true
		}
	}

	if !foundAudience {
		t.Errorf("Pergunta saas.audience deveria estar ativa para tipo saas")
	}
}

func TestIntentParser(t *testing.T) {
	prompt := "Quero um CRM imobiliário para empresas, com corretores, clientes, imóveis, kanban, WhatsApp, dashboard e assinatura."
	intent := ParseNaturalDescription(prompt)

	if intent.ProjectType != "crm" {
		t.Errorf("Esperado ProjectType crm, obteve %s", intent.ProjectType)
	}
	if !intent.MultiTenant {
		t.Errorf("Esperado MultiTenant true para CRM imobiliário de empresas")
	}
	if intent.Audience != "B2B" {
		t.Errorf("Esperado Audience B2B, obteve %s", intent.Audience)
	}

	foundCliente := false
	foundImovel := false
	for _, e := range intent.DetectedEntities {
		if e.Name == "Cliente" {
			foundCliente = true
		}
		if e.Name == "Imovel" {
			foundImovel = true
		}
	}

	if !foundCliente || !foundImovel {
		t.Errorf("Entidades esperadas (Cliente, Imovel) não encontradas em: %+v", intent.DetectedEntities)
	}
}

func TestCapabilityResolver(t *testing.T) {
	req := []string{CapChat, CapSubscription}
	resolved, err := ResolveCapabilityDependencies(req)
	if err != nil {
		t.Fatalf("Erro ao resolver capabilities: %v", err)
	}

	hasDatabase := false
	hasBilling := false
	hasUsers := false
	for _, c := range resolved {
		if c == CapDatabase {
			hasDatabase = true
		}
		if c == CapBilling {
			hasBilling = true
		}
		if c == CapUsers {
			hasUsers = true
		}
	}

	if !hasDatabase || !hasBilling || !hasUsers {
		t.Errorf("Dependências transitivas não resolvidas: %+v", resolved)
	}
}

func TestManifestProtectionAndOwnership(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ge-manifest-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	manifest := NewProjectManifest("AppTeste", "saas")
	manifest.RegisterFile("paginas/custom.ge", OwnershipCustomized, "Página customizada pelo dev")

	if err := manifest.Save(tmpDir); err != nil {
		t.Fatalf("Erro ao salvar manifesto: %v", err)
	}

	loaded, err := LoadManifest(tmpDir)
	if err != nil {
		t.Fatalf("Erro ao carregar manifesto: %v", err)
	}

	if !loaded.IsProtected("paginas/custom.ge") {
		t.Errorf("Arquivo custom.ge deveria estar protegido contra sobrescrita")
	}
	if loaded.IsProtected("paginas/outro.ge") {
		t.Errorf("Arquivo outro.ge não deveria estar protegido")
	}
}

func TestFullProjectScaffoldAndGraph(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ge-e2e-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	engine := NewIntelligenceEngine(tmpDir)
	graph, err := engine.InitProject(InitProjectOptions{
		RootDir:     tmpDir,
		Name:        "MeuCRM",
		Mode:        "prompt",
		Description: "Quero um CRM para empresas com clientes e vendas",
	})
	if err != nil {
		t.Fatalf("Falha em InitProject: %v", err)
	}

	if graph == nil {
		t.Fatalf("Knowledge Graph retornado é nil")
	}

	// Check generated files
	if _, err := os.Stat(filepath.Join(tmpDir, "inicio.ge")); err != nil {
		t.Errorf("inicio.ge não foi gerado: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "dados/clientes.ge")); err != nil {
		t.Errorf("dados/clientes.ge não foi gerado: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "paginas/dashboard.ge")); err != nil {
		t.Errorf("paginas/dashboard.ge não foi gerado: %v", err)
	}

	// Validate with Validator
	validator := NewProjectValidator(tmpDir, graph)
	diags := validator.ValidateAll()

	for _, d := range diags {
		if d.Severity == SeverityError {
			t.Errorf("Diagnóstico de erro inesperado: %s — %s (%s)", d.Code, d.Title, d.Location)
		}
	}

	// Test Explain
	exp, err := graph.ExplainPage("/clientes")
	if err != nil {
		t.Fatalf("Erro ao explicar página /clientes: %v", err)
	}
	if exp.Entity != "Cliente" {
		t.Errorf("Esperado Entity Cliente, obteve %s", exp.Entity)
	}

	// Test Text Tree
	tree := graph.RenderTextTree()
	if !strings.Contains(tree, "MeuCRM") || !strings.Contains(tree, "Cliente") {
		t.Errorf("Árvore de texto não contém nós esperados:\n%s", tree)
	}
}

func TestIncrementalDomainAndOutsideIn(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ge-outside-in-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	engine := NewIntelligenceEngine(tmpDir)
	_, err = engine.InitProject(InitProjectOptions{
		RootDir: tmpDir,
		Name:    "LojaSimples",
		Mode:    "rapido",
		Answers: map[string]interface{}{
			"project.type": "saas",
		},
	})
	if err != nil {
		t.Fatalf("Falha no init inicial: %v", err)
	}

	// Outside-in: add financeiro page (entity does not exist, should propose & create Transacao)
	if err := engine.InitPage(InitPageOptions{
		RootDir: tmpDir,
		Name:    "financeiro",
	}); err != nil {
		t.Fatalf("Falha em InitPage financeiro: %v", err)
	}

	// Check if transacoes data model was created
	if _, err := os.Stat(filepath.Join(tmpDir, "dados/transacoes.ge")); err != nil {
		t.Errorf("dados/transacoes.ge não foi criado pelo outside-in scaffolding")
	}
}
