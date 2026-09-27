package intelligence

import (
	"fmt"
	"strings"
)

// ProjectTemplateOptions parameters for full project scaffolding.
type ProjectTemplateOptions struct {
	Name        string
	Type        string
	Audience    string
	MultiTenant bool
	Database    string
	Auth        string
	Roles       []string
	Entities    []EntityNode
	Design      DesignSystemConfig
}

// GenerateProjectFiles produces all .ge source files for a new Germanio project.
func GenerateProjectFiles(opts ProjectTemplateOptions) map[string]string {
	files := make(map[string]string)

	// 1. Root inicio.ge
	files["inicio.ge"] = generateRootApp(opts)

	// 2. Data models (dados/*.ge)
	for _, ent := range opts.Entities {
		path := fmt.Sprintf("dados/%s.ge", strings.ToLower(ent.Plural))
		files[path] = generateEntityModel(ent, opts.MultiTenant)
	}

	// 3. Backend Routes (rotas/*.ge)
	for _, ent := range opts.Entities {
		path := fmt.Sprintf("rotas/%s.ge", strings.ToLower(ent.Plural))
		files[path] = generateEntityRoutes(ent, opts.MultiTenant)
	}

	// 4. UI Pages (paginas/*.ge)
	for _, ent := range opts.Entities {
		path := fmt.Sprintf("paginas/%s.ge", strings.ToLower(ent.Plural))
		files[path] = generateEntityPage(ent, opts)
	}

	// 5. Dashboard (paginas/dashboard.ge)
	files["paginas/dashboard.ge"] = generateDashboardPage(opts)

	// 6. Test file (testes/sistema_teste.ge)
	files["testes/sistema_teste.ge"] = generateSystemTests(opts)

	return files
}

func generateRootApp(opts ProjectTemplateOptions) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("app %s\n\n", CleanIdentifier(opts.Name)))
	b.WriteString(fmt.Sprintf("banco %s\n\n", opts.Database))

	// Design / Theme
	b.WriteString("tema\n")
	b.WriteString(fmt.Sprintf("  cor primaria \"%s\"\n", opts.Design.PrimaryColor))
	b.WriteString(fmt.Sprintf("  estilo \"%s\"\n\n", opts.Design.Preset))

	// Imports
	b.WriteString("# Importações de Dados e Modelos\n")
	for _, ent := range opts.Entities {
		b.WriteString(fmt.Sprintf("importar \"dados/%s.ge\"\n", strings.ToLower(ent.Plural)))
	}
	b.WriteString("\n# Importações de Rotas e Backend\n")
	for _, ent := range opts.Entities {
		b.WriteString(fmt.Sprintf("importar \"rotas/%s.ge\"\n", strings.ToLower(ent.Plural)))
	}
	b.WriteString("\n# Importações de Telas e Páginas\n")
	b.WriteString("importar \"paginas/dashboard.ge\"\n")
	for _, ent := range opts.Entities {
		b.WriteString(fmt.Sprintf("importar \"paginas/%s.ge\"\n", strings.ToLower(ent.Plural)))
	}

	return b.String()
}

func generateEntityModel(ent EntityNode, multiTenant bool) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("tabela %s\n", ent.Name))

	if multiTenant {
		b.WriteString("  organizacao_id: inteiro obrigatorio\n")
	}

	for _, f := range ent.Fields {
		flags := ""
		if f.Required {
			flags += " obrigatorio"
		}
		if f.Unique {
			flags += " unico"
		}
		b.WriteString(fmt.Sprintf("  %s: %s%s\n", f.Name, f.Type, flags))
	}

	return b.String()
}

func generateEntityRoutes(ent EntityNode, multiTenant bool) string {
	var b strings.Builder
	lowerSingular := strings.ToLower(ent.Name)
	lowerPlural := strings.ToLower(ent.Plural)

	var fieldNames []string
	for _, f := range ent.Fields {
		fieldNames = append(fieldNames, f.Name)
	}
	paramsStr := strings.Join(fieldNames, ", ")

	b.WriteString(fmt.Sprintf("# Rotas de Backend para Gestão de %s\n\n", ent.Plural))

	// List route
	b.WriteString(fmt.Sprintf("quando receber listar_%s\n", lowerPlural))
	b.WriteString(fmt.Sprintf("  itens = buscar %s\n", ent.Name))
	b.WriteString("  responder sucesso com dados = itens\n\n")

	// Create route
	b.WriteString(fmt.Sprintf("quando receber cadastrar_%s com %s\n", lowerSingular, paramsStr))
	var assigns []string
	for _, f := range ent.Fields {
		assigns = append(assigns, fmt.Sprintf("%s = %s", f.Name, f.Name))
	}
	b.WriteString(fmt.Sprintf("  novo = criar %s com %s\n", ent.Name, strings.Join(assigns, ", ")))
	b.WriteString("  responder sucesso com item = novo, mensagem = \"Cadastrado com sucesso\"\n\n")

	// Delete route
	b.WriteString(fmt.Sprintf("quando receber excluir_%s com id\n", lowerSingular))
	b.WriteString(fmt.Sprintf("  deletar %s onde id = id\n", ent.Name))
	b.WriteString("  responder sucesso com mensagem = \"Removido com sucesso\"\n")

	return b.String()
}

func generateEntityPage(ent EntityNode, opts ProjectTemplateOptions) string {
	var b strings.Builder
	lowerSingular := strings.ToLower(ent.Name)
	lowerPlural := strings.ToLower(ent.Plural)
	path := "/" + lowerPlural

	b.WriteString(fmt.Sprintf("pagina \"%s\"\n", path))
	b.WriteString(fmt.Sprintf("  titulo \"Gestão de %s\"\n\n", ent.Plural))

	// Navigation bar
	b.WriteString("  navbar\n")
	b.WriteString(fmt.Sprintf("    logo \"/assets/logo.png\" \"%s\"\n", opts.Name))
	b.WriteString("    link \"Dashboard\" \"/dashboard\"\n")
	for _, other := range opts.Entities {
		b.WriteString(fmt.Sprintf("    link \"%s\" \"/%s\"\n", other.Plural, strings.ToLower(other.Plural)))
	}
	b.WriteString(fmt.Sprintf("    botao primario \"+ Novo %s\" \"/%s/novo\"\n\n", ent.Name, lowerPlural))

	// Main Section
	b.WriteString(fmt.Sprintf("  secao \"Lista de %s\"\n", ent.Plural))
	b.WriteString(fmt.Sprintf("    subtitulo \"Gerencie registros e acompanhe o status em tempo real\"\n\n"))

	// Grid with summary cards / search
	b.WriteString("    grade 3 colunas\n")
	b.WriteString("      card\n")
	b.WriteString("        icone \"📋\"\n")
	b.WriteString("        tag \"TOTAL\"\n")
	b.WriteString(fmt.Sprintf("        titulo \"Total de %s\"\n", ent.Plural))
	b.WriteString(fmt.Sprintf("        texto \"Acompanhe todos os registros de %s cadastrados\"\n", lowerPlural))
	b.WriteString("      card\n")
	b.WriteString("        icone \"🔍\"\n")
	b.WriteString("        tag \"FILTROS\"\n")
	b.WriteString("        titulo \"Busca Rápida\"\n")
	b.WriteString("        texto \"Filtre por nome, data e status do registro\"\n")
	b.WriteString("      card\n")
	b.WriteString("        icone \"⚡\"\n")
	b.WriteString("        tag \"AÇÃO\"\n")
	b.WriteString(fmt.Sprintf("        titulo \"Adicionar %s\"\n", ent.Name))
	b.WriteString(fmt.Sprintf("        botao primario \"Criar Novo\" \"/api/cadastrar_%s\"\n\n", lowerSingular))

	// Footer
	b.WriteString("  rodape\n")
	b.WriteString(fmt.Sprintf("    copyright \"%s © 2026 — Construído em Germanio\"\n", opts.Name))
	b.WriteString("    link \"Início\" \"/dashboard\"\n")

	return b.String()
}

func generateDashboardPage(opts ProjectTemplateOptions) string {
	var b strings.Builder
	b.WriteString("pagina \"/dashboard\"\n")
	b.WriteString(fmt.Sprintf("  titulo \"Dashboard — %s\"\n\n", opts.Name))

	// Navbar
	b.WriteString("  navbar\n")
	b.WriteString(fmt.Sprintf("    logo \"/assets/logo.png\" \"%s\"\n", opts.Name))
	b.WriteString("    link \"Dashboard\" \"/dashboard\"\n")
	for _, ent := range opts.Entities {
		b.WriteString(fmt.Sprintf("    link \"%s\" \"/%s\"\n", ent.Plural, strings.ToLower(ent.Plural)))
	}
	b.WriteString("    botao \"Sair\" \"/logout\"\n\n")

	// Hero KPI section
	b.WriteString("  hero\n")
	b.WriteString("    badge \"PAINEL EXECUTIVO\"\n")
	b.WriteString("    titulo \"Visão Geral do Sistema\"\n")
	b.WriteString("    destaque \"Métricas consolidadas em tempo real.\"\n")
	b.WriteString(fmt.Sprintf("    descricao \"Acompanhe o desempenho de %s com dados determinísticos e atualizados.\"\n", opts.Name))
	b.WriteString("    botao primario \"Ver Relatórios\" \"/relatorios\"\n")
	b.WriteString("    botao secundario \"Configurações\" \"/configuracoes\"\n\n")

	// Metrics Grid
	b.WriteString("  secao \"Indicadores Principais\"\n")
	b.WriteString("    subtitulo \"Resumo dos principais módulos e registros de dados ativos\"\n")
	b.WriteString("    grade 3 colunas\n")

	for _, ent := range opts.Entities {
		b.WriteString("      card\n")
		b.WriteString("        icone \"📊\"\n")
		b.WriteString(fmt.Sprintf("        tag \"%s\"\n", strings.ToUpper(ent.Plural)))
		b.WriteString(fmt.Sprintf("        titulo \"%s\"\n", ent.Plural))
		b.WriteString(fmt.Sprintf("        texto \"Módulo ativo de gestão de %s\"\n", strings.ToLower(ent.Plural)))
		b.WriteString(fmt.Sprintf("        link \"/%s\"\n", strings.ToLower(ent.Plural)))
	}

	b.WriteString("\n  rodape\n")
	b.WriteString(fmt.Sprintf("    copyright \"%s © 2026 — Germanio Engine\"\n", opts.Name))
	b.WriteString("    link \"Dashboard\" \"/dashboard\"\n")

	return b.String()
}

func generateSystemTests(opts ProjectTemplateOptions) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Testes Automatizados do Sistema %s\n\n", CleanIdentifier(opts.Name)))

	b.WriteString("teste \"Inicialização e Sanidade do Sistema\"\n")
	b.WriteString("  espera 1 + 1 == 2\n\n")

	for _, ent := range opts.Entities {
		b.WriteString(fmt.Sprintf("teste \"Validação da Entidade %s\"\n", ent.Name))
		b.WriteString(fmt.Sprintf("  nome_teste = \"Teste %s\"\n", ent.Name))
		b.WriteString("  espera nome_teste != \"\"\n\n")
	}

	return b.String()
}

// GenerateSingleEntityScaffold generates a new entity file, route, and page incrementally.
func GenerateSingleEntityScaffold(ent EntityNode, multiTenant bool) map[string]string {
	files := make(map[string]string)
	lowerPlural := strings.ToLower(ent.Plural)

	files[fmt.Sprintf("dados/%s.ge", lowerPlural)] = generateEntityModel(ent, multiTenant)
	files[fmt.Sprintf("rotas/%s.ge", lowerPlural)] = generateEntityRoutes(ent, multiTenant)
	files[fmt.Sprintf("paginas/%s.ge", lowerPlural)] = generateEntityPage(ent, ProjectTemplateOptions{
		Name:     "GermanioApp",
		Entities: []EntityNode{ent},
		Design:   DesignSystemConfig{PrimaryColor: "#00D9FF", Preset: "saas_moderno"},
	})

	return files
}
