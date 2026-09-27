package intelligence

import (
	"fmt"
	"sort"
)

// Standard Capability IDs.
const (
	CapAuth         = "CAP_AUTH"
	CapUsers        = "CAP_USERS"
	CapRBAC         = "CAP_RBAC"
	CapOrg          = "CAP_ORGANIZATIONS"
	CapMultiTenant  = "CAP_MULTI_TENANT"
	CapDatabase     = "CAP_DATABASE"
	CapStorage      = "CAP_STORAGE"
	CapAPI          = "CAP_API"
	CapWebhook      = "CAP_WEBHOOK"
	CapRealtime     = "CAP_REALTIME"
	CapChat         = "CAP_CHAT"
	CapEmail        = "CAP_EMAIL"
	CapNotification = "CAP_NOTIFICATION"
	CapQueue        = "CAP_QUEUE"
	CapCache        = "CAP_CACHE"
	CapBilling      = "CAP_BILLING"
	CapSubscription = "CAP_SUBSCRIPTION"
	CapSearch       = "CAP_SEARCH"
	CapAI           = "CAP_AI"
	CapSEO          = "CAP_SEO"
	CapAudit        = "CAP_AUDIT"
	CapDashboard    = "CAP_DASHBOARD"
)

// CapabilityDef defines static metadata and dependencies for a capability.
type CapabilityDef struct {
	ID           string
	Name         string
	Description  string
	Status       CapabilityStatus
	Dependencies []string
}

// Registry of all known capabilities in Germanio.
var StandardCapabilities = map[string]CapabilityDef{
	CapDatabase: {
		ID:          CapDatabase,
		Name:        "Banco de Dados Declarativo",
		Description: "Persistência relacional com SQLite/PostgreSQL e migrações automáticas",
		Status:      CapStable,
	},
	CapAuth: {
		ID:           CapAuth,
		Name:         "Autenticação e Sessões",
		Description:  "Login seguro, hashing de senhas, sessões e proteção contra ataques",
		Status:       CapStable,
		Dependencies: []string{CapDatabase},
	},
	CapUsers: {
		ID:           CapUsers,
		Name:         "Gestão de Usuários e Perfis",
		Description:  "Cadastro, edição de perfil, recuperação de senha e estados de conta",
		Status:       CapStable,
		Dependencies: []string{CapAuth, CapDatabase},
	},
	CapRBAC: {
		ID:           CapRBAC,
		Name:         "Controle de Acesso por Papéis (RBAC)",
		Description:  "Perfis (Admin, Vendedor, Cliente), permissões por rota e validação em camada",
		Status:       CapStable,
		Dependencies: []string{CapUsers},
	},
	CapOrg: {
		ID:           CapOrg,
		Name:         "Organizações e Empresas",
		Description:  "Gestão de múltiplos membros por empresa e convites de equipe",
		Status:       CapImplemented,
		Dependencies: []string{CapUsers, CapRBAC},
	},
	CapMultiTenant: {
		ID:           CapMultiTenant,
		Name:         "Isolamento Multi-Tenant B2B",
		Description:  "Separação lógica estrita de dados por tenant no banco e rotas",
		Status:       CapImplemented,
		Dependencies: []string{CapOrg, CapDatabase},
	},
	CapAPI: {
		ID:           CapAPI,
		Name:         "APIs REST e Contratos Automáticos",
		Description:  "Endpoints determinísticos com validação de payload, headers e OpenAPI",
		Status:       CapStable,
		Dependencies: []string{CapDatabase},
	},
	CapWebhook: {
		ID:           CapWebhook,
		Name:         "Webhooks Inbound e Outbound",
		Description:  "Recepção e disparo de eventos HTTP assinados com retry automático",
		Status:       CapImplemented,
		Dependencies: []string{CapAPI},
	},
	CapRealtime: {
		ID:          CapRealtime,
		Name:        "Eventos em Tempo Real (WebSocket)",
		Description: "Canais de comunicação bidirecionais e sincronização de estado",
		Status:      CapStable,
	},
	CapChat: {
		ID:           CapChat,
		Name:         "Chat e Mensageria Instantânea",
		Description:  "Salas, histórico de conversas, envio de mídias e status online",
		Status:       CapImplemented,
		Dependencies: []string{CapRealtime, CapUsers, CapDatabase},
	},
	CapEmail: {
		ID:          CapEmail,
		Name:        "Envio de Emails Transacionais",
		Description: "Templates responsivos, fila SMTP e rastreamento de entregas",
		Status:      CapStable,
	},
	CapNotification: {
		ID:           CapNotification,
		Name:         "Notificações In-App e Alertas",
		Description:  "Central de notificações no frontend e push updates",
		Status:       CapImplemented,
		Dependencies: []string{CapUsers},
	},
	CapStorage: {
		ID:          CapStorage,
		Name:        "Uploads de Arquivos e Mídia",
		Description: "Validação de MIME types, armazenamento seguro e CDN URLs",
		Status:      CapImplemented,
	},
	CapQueue: {
		ID:          CapQueue,
		Name:        "Filas e Processamento Assíncrono",
		Description: "Tarefas em background com retries, dead-letter e backoff exponencial",
		Status:      CapImplemented,
	},
	CapCache: {
		ID:          CapCache,
		Name:        "Cache de Alta Performance",
		Description: "In-memory e Redis caching para queries e sessões frequentes",
		Status:      CapImplemented,
	},
	CapBilling: {
		ID:           CapBilling,
		Name:         "Pagamentos e Checkout",
		Description:  "Gateway unificado (Stripe/Pix/MercadoPago) com webhooks e faturas",
		Status:       CapImplemented,
		Dependencies: []string{CapAPI, CapDatabase},
	},
	CapSubscription: {
		ID:           CapSubscription,
		Name:         "Assinaturas Recorrentes e Planos",
		Description:  "Controle de limites por plano, upgrades, downgrades e período de testes",
		Status:       CapImplemented,
		Dependencies: []string{CapBilling, CapUsers},
	},
	CapSearch: {
		ID:           CapSearch,
		Name:         "Busca Textual e Filtros Avançados",
		Description:  "Busca em múltiplas colunas com debounce no frontend e índices no banco",
		Status:       CapStable,
		Dependencies: []string{CapDatabase},
	},
	CapDashboard: {
		ID:           CapDashboard,
		Name:         "Dashboard de Métricas e KPIs",
		Description:  "Cards de estatísticas, agregações em tempo real e gráficos comparativos",
		Status:       CapStable,
		Dependencies: []string{CapDatabase},
	},
	CapSEO: {
		ID:          CapSEO,
		Name:        "Otimização para Mecanismos de Busca (SEO)",
		Description: "Tags Open Graph, JSON-LD structured data, sitemap e robots.txt automáticos",
		Status:      CapStable,
	},
	CapAudit: {
		ID:           CapAudit,
		Name:         "Logs Estruturados e Trilha de Auditoria",
		Description:  "Registro imutável de ações de usuários para conformidade e segurança",
		Status:       CapStable,
		Dependencies: []string{CapDatabase},
	},
	CapAI: {
		ID:          CapAI,
		Name:        "Decision Engine e Inteligência",
		Description: "Blocos declarativos de decisão e classificação por IA determinística",
		Status:      CapExperimental,
	},
}

// ResolveCapabilityDependencies resolves full dependency DAG for requested capabilities.
func ResolveCapabilityDependencies(requested []string) ([]string, error) {
	resolved := make(map[string]bool)
	var visit func(id string, path []string) error

	visit = func(id string, path []string) error {
		// Detect cycles
		for _, p := range path {
			if p == id {
				return fmt.Errorf("Ciclo de dependência detectado em capabilities: %s -> %s", p, id)
			}
		}

		if resolved[id] {
			return nil
		}

		def, ok := StandardCapabilities[id]
		if !ok {
			return fmt.Errorf("Capability desconhecida: %s", id)
		}

		newPath := append(path, id)
		for _, dep := range def.Dependencies {
			if err := visit(dep, newPath); err != nil {
				return err
			}
		}

		resolved[id] = true
		return nil
	}

	for _, capID := range requested {
		if err := visit(capID, nil); err != nil {
			return nil, err
		}
	}

	var result []string
	for id := range resolved {
		result = append(result, id)
	}
	sort.Strings(result)
	return result, nil
}
