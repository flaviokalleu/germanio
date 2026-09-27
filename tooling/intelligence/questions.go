package intelligence

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// QuestionType defines the input format for a question.
type QuestionType string

const (
	TypeSingle   QuestionType = "single"
	TypeMultiple QuestionType = "multiple"
	TypeText     QuestionType = "text"
	TypeNumber   QuestionType = "number"
	TypeBoolean  QuestionType = "boolean"
	TypeColor    QuestionType = "color"
)

// QuestionOption represents a choice in single/multiple questions.
type QuestionOption struct {
	ID           string   `json:"id"`
	Label        string   `json:"label"`
	Description  string   `json:"description,omitempty"`
	ProducesCaps []string `json:"produces_caps,omitempty"`
}

// Question defines an interactive decision node in the adaptive tree.
type Question struct {
	ID           string                                    `json:"id"`
	Category     string                                    `json:"category"`
	Text         string                                    `json:"text"`
	Description  string                                    `json:"description,omitempty"`
	Type         QuestionType                              `json:"type"`
	Options      []QuestionOption                          `json:"options,omitempty"`
	DefaultVal   interface{}                               `json:"default_val,omitempty"`
	Condition    func(answers map[string]interface{}) bool `json:"-"`
	Validate     func(val interface{}) error               `json:"-"`
	ProducesCaps []string                                  `json:"produces_caps,omitempty"`
}

// BuildStandardQuestionTree registers the complete adaptive decision tree for Germanio Init.
func BuildStandardQuestionTree() []Question {
	return []Question{
		// 1. Projeto e Tipo
		{
			ID:          "project.name",
			Category:    "Projeto",
			Text:        "Qual é o nome do seu projeto?",
			Description: "Identificador da aplicação e título principal",
			Type:        TypeText,
			DefaultVal:  "MeuApp",
			Validate: func(val interface{}) error {
				s, ok := val.(string)
				if !ok || strings.TrimSpace(s) == "" {
					return fmt.Errorf("Nome do projeto não pode ser vazio")
				}
				return nil
			},
		},
		{
			ID:          "project.type",
			Category:    "Produto",
			Text:        "O que você quer criar?",
			Description: "Determina os blocos e fluxos padrão",
			Type:        TypeSingle,
			DefaultVal:  "saas",
			Options: []QuestionOption{
				{ID: "saas", Label: "Sistema / SaaS", Description: "Software como serviço com autenticação, planos e painel", ProducesCaps: []string{CapAuth, CapUsers, CapDashboard, CapAPI}},
				{ID: "crm", Label: "CRM / Gestão Comercial", Description: "Gestão de leads, clientes, contatos, kanban e vendas", ProducesCaps: []string{CapAuth, CapUsers, CapDashboard, CapSearch, CapAPI}},
				{ID: "ecommerce", Label: "E-commerce / Loja Virtual", Description: "Catálogo de produtos, carrinho, pedidos e checkout", ProducesCaps: []string{CapDatabase, CapBilling, CapStorage, CapAPI}},
				{ID: "dashboard", Label: "Dashboard Administrativo", Description: "Visualização de métricas, gráficos e tabelas de dados", ProducesCaps: []string{CapDashboard, CapDatabase}},
				{ID: "site", Label: "Site Institucional / Landing Page", Description: "Páginas públicas com apresentação, hero e contato", ProducesCaps: []string{CapSEO}},
				{ID: "api", Label: "API Backend Pura", Description: "Serviços REST sem interface gráfica direta", ProducesCaps: []string{CapAPI, CapDatabase}},
				{ID: "portal", Label: "Blog / Portal de Conteúdo", Description: "Artigos, categorias, comentários e SEO", ProducesCaps: []string{CapDatabase, CapSEO, CapStorage}},
			},
		},

		// 2. Público e Arquitetura Multi-Tenant
		{
			ID:          "saas.audience",
			Category:    "Público",
			Text:        "Qual é o público-alvo principal?",
			Description: "B2B necessita de controle por empresa/organização",
			Type:        TypeSingle,
			DefaultVal:  "B2B",
			Condition: func(ans map[string]interface{}) bool {
				t, _ := ans["project.type"].(string)
				return t == "saas" || t == "crm"
			},
			Options: []QuestionOption{
				{ID: "B2B", Label: "B2B (Empresas e Equipes)", Description: "Organizações com múltiplos usuários e convites", ProducesCaps: []string{CapOrg, CapMultiTenant, CapRBAC}},
				{ID: "B2C", Label: "B2C (Usuários Finais Individuais)", Description: "Cada usuário possui sua conta privada", ProducesCaps: []string{CapUsers, CapAuth}},
				{ID: "B2B2C", Label: "B2B2C (Empresas que atendem clientes finais)", Description: "Controle de tenant com portal do cliente", ProducesCaps: []string{CapOrg, CapMultiTenant, CapRBAC}},
			},
		},
		{
			ID:          "tenant.enabled",
			Category:    "Organizações",
			Text:        "Cada empresa terá seu próprio espaço de trabalho isolado (Multi-tenant)?",
			Description: "Garante que clientes de uma empresa nunca vejam dados de outra",
			Type:        TypeBoolean,
			DefaultVal:  true,
			Condition: func(ans map[string]interface{}) bool {
				aud, _ := ans["saas.audience"].(string)
				return aud == "B2B" || aud == "B2B2C"
			},
			ProducesCaps: []string{CapMultiTenant},
		},

		// 3. Banco de Dados
		{
			ID:          "database.engine",
			Category:    "Banco de Dados",
			Text:        "Qual banco de dados deseja utilizar?",
			Description: "SQLite é embutido sem dependências; PostgreSQL é ideal para escala",
			Type:        TypeSingle,
			DefaultVal:  "sqlite",
			Options: []QuestionOption{
				{ID: "sqlite", Label: "SQLite (Embutido, zero-configuração)", Description: "Roda localmente direto no arquivo .db", ProducesCaps: []string{CapDatabase}},
				{ID: "postgres", Label: "PostgreSQL (Produção e Alta Concorrência)", Description: "Conexão com servidor PostgreSQL", ProducesCaps: []string{CapDatabase}},
			},
		},

		// 4. Autenticação e Perfis
		{
			ID:         "auth.strategy",
			Category:   "Autenticação",
			Text:       "Qual formato de login deseja oferecer?",
			Type:       TypeSingle,
			DefaultVal: "email_senha",
			Condition: func(ans map[string]interface{}) bool {
				t, _ := ans["project.type"].(string)
				return t != "site" && t != "api"
			},
			Options: []QuestionOption{
				{ID: "email_senha", Label: "Email e Senha", Description: "Login clássico com recuperação de senha", ProducesCaps: []string{CapAuth}},
				{ID: "magic_link", Label: "Magic Link por Email (Passwordless)", Description: "Acesso por link seguro sem senha fixa", ProducesCaps: []string{CapAuth, CapEmail}},
				{ID: "oauth_google", Label: "Email/Senha + Google OAuth", Description: "Login com credenciais ou botão Google", ProducesCaps: []string{CapAuth}},
			},
		},
		{
			ID:          "rbac.roles",
			Category:    "Papéis",
			Text:        "Quais papéis de usuário devem existir?",
			Description: "Define níveis de acesso e permissões",
			Type:        TypeMultiple,
			DefaultVal:  []string{"admin", "operador"},
			Condition: func(ans map[string]interface{}) bool {
				t, _ := ans["project.type"].(string)
				return t == "saas" || t == "crm" || t == "dashboard"
			},
			Options: []QuestionOption{
				{ID: "admin", Label: "Administrador (Acesso total)", ProducesCaps: []string{CapRBAC}},
				{ID: "gestor", Label: "Gestor / Supervisor", ProducesCaps: []string{CapRBAC}},
				{ID: "operador", Label: "Operador / Vendedor", ProducesCaps: []string{CapRBAC}},
				{ID: "cliente", Label: "Cliente / Visualizador", ProducesCaps: []string{CapRBAC}},
			},
		},

		// 5. Recursos Avançados (Tempo Real, Pagamentos, Chat)
		{
			ID:         "features.realtime",
			Category:   "Tempo Real",
			Text:       "Precisa de atualizações em tempo real (notificações imediatas, status ao vivo)?",
			Type:       TypeBoolean,
			DefaultVal: true,
			Condition: func(ans map[string]interface{}) bool {
				t, _ := ans["project.type"].(string)
				return t == "saas" || t == "crm"
			},
			ProducesCaps: []string{CapRealtime},
		},
		{
			ID:         "features.billing",
			Category:   "Pagamentos",
			Text:       "Deseja controle de assinaturas recorrentes e cobranças?",
			Type:       TypeBoolean,
			DefaultVal: false,
			Condition: func(ans map[string]interface{}) bool {
				t, _ := ans["project.type"].(string)
				return t == "saas" || t == "ecommerce"
			},
			ProducesCaps: []string{CapBilling, CapSubscription},
		},

		// 6. Design System
		{
			ID:         "design.preset",
			Category:   "Design",
			Text:       "Qual estilo visual prefere para a interface?",
			Type:       TypeSingle,
			DefaultVal: "saas_moderno",
			Condition: func(ans map[string]interface{}) bool {
				t, _ := ans["project.type"].(string)
				return t != "api"
			},
			Options: []QuestionOption{
				{ID: "saas_moderno", Label: "SaaS Moderno (Dark mode, detalhes ciano/azul)", Description: "Padrão de produtos tecnológicos modernos"},
				{ID: "enterprise", Label: "Enterprise (Sóbrio, azul corporativo)", Description: "Aplicações corporativas e sistemas internos"},
				{ID: "minimalista", Label: "Minimalista (Alto contraste, tipografia limpa)", Description: "Foco total na leitura e dados"},
				{ID: "financeiro", Label: "Financeiro (Verde esmeralda, tabelas densas)", Description: "Ideal para finanças e números"},
			},
		},
	}
}

// AdaptiveQuestionEngine runs the dynamic question tree based on accumulating answers.
type AdaptiveQuestionEngine struct {
	Questions []Question
}

// NewAdaptiveQuestionEngine creates a new question engine instance.
func NewAdaptiveQuestionEngine() *AdaptiveQuestionEngine {
	return &AdaptiveQuestionEngine{
		Questions: BuildStandardQuestionTree(),
	}
}

// GetNextQuestions returns the list of questions that satisfy their conditions given current answers.
func (e *AdaptiveQuestionEngine) CollectAnswers(in io.Reader, mode string, seed map[string]interface{}) (map[string]interface{}, error) {
	answers := make(map[string]interface{}, len(seed)+len(e.Questions))
	for key, value := range seed {
		answers[key] = value
	}
	for _, q := range e.GetActiveQuestions(answers) {
		if _, present := answers[q.ID]; present {
			continue
		}
		if mode != "guiado" {
			answers[q.ID] = q.DefaultVal
			continue
		}
		if in == nil {
			return nil, fmt.Errorf("modo guiado requer entrada")
		}
		fmt.Fprintf(io.Discard, "%s", q.Text)
		line, err := bufio.NewReader(in).ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			answers[q.ID] = q.DefaultVal
		} else {
			answers[q.ID] = line
		}
		if q.Validate != nil {
			if err := q.Validate(answers[q.ID]); err != nil {
				return nil, err
			}
		}
	}
	return answers, nil
}

// GetActiveQuestions returns questions whose conditions are satisfied.
func (e *AdaptiveQuestionEngine) GetActiveQuestions(answers map[string]interface{}) []Question {
	var active []Question
	for _, q := range e.Questions {
		if q.Condition == nil || q.Condition(answers) {
			active = append(active, q)
		}
	}
	return active
}
