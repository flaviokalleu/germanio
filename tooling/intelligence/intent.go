package intelligence

import (
	"regexp"
	"strings"
)

// ParsedIntent represents structured architectural understanding extracted from natural text.
type ParsedIntent struct {
	ProjectName      string
	ProjectType      string
	Audience         string
	MultiTenant      bool
	DetectedEntities []EntityNode
	Capabilities     []string
	Pages            []string
	FeaturesSummary  []string
	MissingDecisions []string
}

// ParseNaturalDescription extracts domains, capabilities and architecture from natural text.
func ParseNaturalDescription(desc string) *ParsedIntent {
	text := strings.ToLower(desc)

	intent := &ParsedIntent{
		ProjectName:      "MeuSistema",
		ProjectType:      "saas",
		Audience:         "B2B",
		MultiTenant:      false,
		Capabilities:     []string{CapDatabase, CapAuth, CapUsers},
		FeaturesSummary:  make([]string, 0),
		MissingDecisions: make([]string, 0),
	}

	// 1. Detect project type
	switch {
	case containsAny(text, "crm", "corretor", "imobiliaria", "imobiliária", "leads", "propostas", "vendas"):
		intent.ProjectType = "crm"
		intent.FeaturesSummary = append(intent.FeaturesSummary, "CRM Comercial")
	case containsAny(text, "e-commerce", "ecommerce", "loja", "produtos", "carrinho", "comprar"):
		intent.ProjectType = "ecommerce"
		intent.FeaturesSummary = append(intent.FeaturesSummary, "E-commerce")
	case containsAny(text, "dashboard", "painel", "metricas", "métricas", "analytics"):
		intent.ProjectType = "dashboard"
		intent.FeaturesSummary = append(intent.FeaturesSummary, "Dashboard de Gestão")
	case containsAny(text, "site", "landing page", "institucional"):
		intent.ProjectType = "site"
		intent.FeaturesSummary = append(intent.FeaturesSummary, "Site / Landing Page")
	default:
		intent.ProjectType = "saas"
		intent.FeaturesSummary = append(intent.FeaturesSummary, "SaaS / Sistema Web")
	}

	// 2. Detect B2B & Multi-Tenant
	if containsAny(text, "empresa", "empresas", "b2b", "equipe", "equipes", "imobiliaria", "imobiliárias", "multi-tenant", "tenant", "organizacao", "organização") {
		intent.Audience = "B2B"
		intent.MultiTenant = true
		intent.Capabilities = append(intent.Capabilities, CapOrg, CapMultiTenant, CapRBAC)
		intent.FeaturesSummary = append(intent.FeaturesSummary, "Multi-tenant B2B (Espaço por Empresa)")
	}

	// 3. Detect Domain Entities
	intent.DetectedEntities = extractEntitiesFromText(text)
	for _, e := range intent.DetectedEntities {
		intent.FeaturesSummary = append(intent.FeaturesSummary, "Entidade: "+e.Name)
	}

	// 4. Detect Specific Features
	if containsAny(text, "kanban", "funil", "etapas", "pipeline") {
		intent.FeaturesSummary = append(intent.FeaturesSummary, "Visualização Kanban de Negócios")
	}
	if containsAny(text, "dashboard", "kpi", "grafico", "gráficos", "receita", "metricas", "métricas") {
		intent.Capabilities = append(intent.Capabilities, CapDashboard)
		intent.FeaturesSummary = append(intent.FeaturesSummary, "Dashboard de Métricas")
	}
	if containsAny(text, "whatsapp", "chat", "mensagens", "conversa") {
		intent.Capabilities = append(intent.Capabilities, CapRealtime, CapChat)
		intent.FeaturesSummary = append(intent.FeaturesSummary, "Integração de Comunicação / Chat")
	}
	if containsAny(text, "assinatura", "assinaturas", "pagamento", "pagamentos", "stripe", "pix", "fatura") {
		intent.Capabilities = append(intent.Capabilities, CapBilling, CapSubscription)
		intent.FeaturesSummary = append(intent.FeaturesSummary, "Gestão de Assinaturas e Pagamentos")
	}
	if containsAny(text, "busca", "pesquisa", "filtro", "filtros") {
		intent.Capabilities = append(intent.Capabilities, CapSearch)
	}

	// 5. Generate Standard Pages based on entities
	intent.Pages = append(intent.Pages, "/dashboard")
	for _, e := range intent.DetectedEntities {
		intent.Pages = append(intent.Pages, "/"+strings.ToLower(e.Plural))
	}

	// Deduplicate capabilities
	resolvedCaps, _ := ResolveCapabilityDependencies(intent.Capabilities)
	intent.Capabilities = resolvedCaps

	return intent
}

func containsAny(s string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

func extractEntitiesFromText(text string) []EntityNode {
	var entities []EntityNode

	// Map of common domain patterns to predefined schema recommendations
	patterns := []struct {
		Keywords []string
		Entity   EntityNode
	}{
		{
			Keywords: []string{"corretor", "corretores"},
			Entity: EntityNode{
				Name:      "Corretor",
				Plural:    "Corretores",
				TableName: "corretores",
				Fields: []FieldMeta{
					{Name: "nome", Type: FieldTexto, Required: true, Searchable: true},
					{Name: "email", Type: FieldEmail, Required: true, Unique: true},
					{Name: "telefone", Type: FieldTelefone, Required: false},
					{Name: "creci", Type: FieldTexto, Required: false, Searchable: true},
					{Name: "status", Type: FieldStatus, Required: true, DefaultVal: "ativo"},
				},
				ReadRoles:   []string{"Administrador", "Corretor"},
				WriteRoles:  []string{"Administrador"},
				DeleteRoles: []string{"Administrador"},
			},
		},
		{
			Keywords: []string{"cliente", "clientes", "lead", "leads"},
			Entity: EntityNode{
				Name:      "Cliente",
				Plural:    "Clientes",
				TableName: "clientes",
				Fields: []FieldMeta{
					{Name: "nome", Type: FieldTexto, Required: true, Searchable: true},
					{Name: "email", Type: FieldEmail, Required: true, Unique: true},
					{Name: "telefone", Type: FieldTelefone, Required: false},
					{Name: "status", Type: FieldStatus, Required: true, DefaultVal: "novo"},
				},
				ReadRoles:   []string{"Administrador", "Operador"},
				WriteRoles:  []string{"Administrador", "Operador"},
				DeleteRoles: []string{"Administrador"},
			},
		},
		{
			Keywords: []string{"imovel", "imóvel", "imoveis", "imóveis"},
			Entity: EntityNode{
				Name:      "Imovel",
				Plural:    "Imoveis",
				TableName: "imoveis",
				Fields: []FieldMeta{
					{Name: "titulo", Type: FieldTexto, Required: true, Searchable: true},
					{Name: "valor", Type: FieldDinheiro, Required: true},
					{Name: "endereco", Type: FieldTexto, Required: false},
					{Name: "tipo", Type: FieldTexto, Required: true, DefaultVal: "Apartamento"},
					{Name: "status", Type: FieldStatus, Required: true, DefaultVal: "disponivel"},
				},
				ReadRoles:   []string{"Administrador", "Operador", "Cliente"},
				WriteRoles:  []string{"Administrador", "Operador"},
				DeleteRoles: []string{"Administrador"},
			},
		},
		{
			Keywords: []string{"produto", "produtos", "item", "itens"},
			Entity: EntityNode{
				Name:      "Produto",
				Plural:    "Produtos",
				TableName: "produtos",
				Fields: []FieldMeta{
					{Name: "nome", Type: FieldTexto, Required: true, Searchable: true},
					{Name: "preco", Type: FieldDinheiro, Required: true},
					{Name: "estoque", Type: FieldInteiro, Required: true, DefaultVal: "0"},
					{Name: "status", Type: FieldStatus, Required: true, DefaultVal: "ativo"},
				},
				ReadRoles:   []string{"Administrador", "Cliente"},
				WriteRoles:  []string{"Administrador"},
				DeleteRoles: []string{"Administrador"},
			},
		},
		{
			Keywords: []string{"venda", "vendas", "pedido", "pedidos"},
			Entity: EntityNode{
				Name:      "Venda",
				Plural:    "Vendas",
				TableName: "vendas",
				Fields: []FieldMeta{
					{Name: "cliente_nome", Type: FieldTexto, Required: true},
					{Name: "valor_total", Type: FieldDinheiro, Required: true},
					{Name: "data", Type: FieldData, Required: true},
					{Name: "status", Type: FieldStatus, Required: true, DefaultVal: "pendente"},
				},
				ReadRoles:   []string{"Administrador", "Operador"},
				WriteRoles:  []string{"Administrador", "Operador"},
				DeleteRoles: []string{"Administrador"},
			},
		},
	}

	seen := make(map[string]bool)
	for _, p := range patterns {
		for _, kw := range p.Keywords {
			if strings.Contains(text, kw) && !seen[p.Entity.Name] {
				entities = append(entities, p.Entity)
				seen[p.Entity.Name] = true
				break
			}
		}
	}

	// Fallback if no specific entities matched: create default based on nouns
	if len(entities) == 0 {
		entities = append(entities, EntityNode{
			Name:      "Item",
			Plural:    "Itens",
			TableName: "itens",
			Fields: []FieldMeta{
				{Name: "nome", Type: FieldTexto, Required: true, Searchable: true},
				{Name: "descricao", Type: FieldTextoLongo, Required: false},
				{Name: "status", Type: FieldStatus, Required: true, DefaultVal: "ativo"},
			},
		})
	}

	return entities
}

// Pluralize returns the idiomatic Portuguese plural of a noun.
func Pluralize(s string) string {
	lower := strings.ToLower(s)
	switch {
	case strings.HasSuffix(lower, "ao") || strings.HasSuffix(lower, "ão"):
		base := s[:len(s)-2]
		return base + "oes"
	case strings.HasSuffix(lower, "el"):
		base := s[:len(s)-2]
		return base + "eis"
	case strings.HasSuffix(lower, "m"):
		base := s[:len(s)-1]
		return base + "ns"
	case strings.HasSuffix(lower, "r") || strings.HasSuffix(lower, "z"):
		return s + "es"
	default:
		if strings.HasSuffix(lower, "s") {
			return s
		}
		return s + "s"
	}
}

// CleanIdentifier normalizes string into valid Germanio identifier
func CleanIdentifier(s string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	cleaned := reg.ReplaceAllString(s, "")
	if len(cleaned) == 0 {
		return "App"
	}
	return strings.ToUpper(cleaned[:1]) + cleaned[1:]
}
