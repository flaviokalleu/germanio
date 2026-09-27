# GitLab FOSS — Arquitetura auditada e mapeamento para Germanio

Referência auditada: `gitlab-org/gitlab-foss` @ `c8c0c71a33ba` (2026-09-27), clone raso.

## Tamanho medido

| Área | Arquivos | Observação |
|------|---------:|-----------|
| `app/models` | 1245 (232 na raiz, 75 em `ci/`) | ActiveRecord; 1458 tabelas em `db/structure.sql` |
| `app/services` | 1559 | regras de negócio (`*Service#execute`) |
| `app/controllers` | 562 | UI Rails (HAML + Vue) |
| `lib/api` | 788 (159 endpoints-raiz) | REST v4 (Grape) |
| `app/graphql` | 1494 | GraphQL (UI moderna usa muito) |
| `app/workers` | 626 | Sidekiq |
| `app/policies` | 194 | DeclarativePolicy |
| `lib/gitlab/ci` | 634 | parser/processamento de `.gitlab-ci.yml` |
| `app/assets/javascripts` | 6767 | Vue 2/3 |
| Ruby (`app`+`lib`) | ~806 mil LOC | |
| JS/Vue | ~602 mil LOC | |

## Componentes de processo (GitLab real)

```
Navegador ─► Workhorse (Go) ─► Puma/Rails (UI + API v4 + GraphQL)
   git clone/push HTTP ─► Workhorse ─► Gitaly (gRPC, Go) ─► repositórios bare
   git SSH ─► gitlab-shell ─► Gitaly
Rails ─► PostgreSQL · Redis (cache, sessão, fila) · Sidekiq (workers)
gitlab-runner (Go, externo) ─► API /api/v4/jobs/request|:id|:id/trace|artifacts
```

## Fluxos seguidos no código

1. **Registro/login**: `RegistrationsController` → `Users::CreateService` → Devise (`encrypted_password` bcrypt);
   sessão por cookie; API por `PRIVATE-TOKEN` (`personal_access_tokens.token_digest`).
2. **Namespace**: `namespaces.type ∈ {User, Group, Project}`; `parent_id` + `traversal_ids` para subgrupos;
   caminho completo `grupo/sub/projeto` resolvido pela tabela `routes`.
3. **Projeto**: `Projects::CreateService` cria linha, `ProjectNamespace`, membro Owner/Maintainer e o repositório via Gitaly;
   `visibility_level` 0 privado / 10 interno / 20 público.
4. **Permissões**: `members(access_level, source_type=Namespace|Project)`; níveis 10 Guest, 15 Planner, 20 Reporter,
   30 Developer, 40 Maintainer, 50 Owner; admin global. `ProjectPolicy`: anônimo só lê projeto público; Guest lê;
   Reporter+ baixa código privado; Developer+ faz push em branch não protegida e cria MR; Maintainer+ administra
   projeto/labels/branches protegidas; Owner remove projeto.
5. **Push**: `git-receive-pack` → hook `/internal/post_receive` → `PostReceiveService` → `Git::BranchPushService`
   (cria pipeline `Ci::CreatePipelineService`, atualiza MRs `MergeRequests::RefreshService`, eventos, webhooks).
6. **Issue**: `Issues::CreateService`; `iid` por projeto (`InternalId`); `state_id` 1 aberto / 2 fechado;
   notas (`notes`, polimórfico `noteable_type`); labels via `label_links`; milestones; assignees (`issue_assignees`);
   issues confidenciais só Reporter+ ou autor/assignee.
7. **Merge Request**: `source_branch`/`target_branch`, `merge_request_diffs` (base/head/start sha),
   `MergeRequests::MergeService` (merge commit via Gitaly `UserMergeBranch`), `merge_status` checado por `MergeabilityCheckService`.
8. **CI**: `Ci::CreatePipelineService` lê `.gitlab-ci.yml` (stages, jobs, script, needs, when, allow_failure, variables),
   cria `p_ci_pipelines`, `p_ci_stages`, `p_ci_builds`; runner faz long-poll `POST /jobs/request`, envia log
   `PATCH /jobs/:id/trace`, estado final `PUT /jobs/:id`, artefatos `POST /jobs/:id/artifacts`.
   Estados: created → pending → running → success|failed|canceled|skipped (`commit_status.rb`); pipeline agrega estados dos jobs.
9. **Webhooks**: `web_hooks` com flags por evento; `WebHookService` assina com `X-Gitlab-Token`, execução em worker.
10. **Notificações**: `NotificationService` + `todos` + e-mail.

## Mapeamento para Germanio (arquitetura alvo)

```
examples/gitlab-foss/ (.ge — produto)
  identity · groups · projects · repositories · issues · merge_requests · ci · notifications · admin · web
        │ usa apenas capabilities genéricas
        ▼
Germanio runtime (Go — mecanismos, sem conhecimento de GitLab)
  servidor/rotas (requisicao, responder) · banco (filtros, sequência, índices compostos)
  cripto · token/sessao · git (runtime/git) · git smart HTTP · tarefas (fila persistente)
  processo · yaml · markdown · ui (registro de componentes) · csrf
```

Equivalências de processo:

| GitLab | Germanio |
|--------|----------|
| Rails controllers/API Grape | `rotas` em `.ge` |
| ActiveRecord models | `dados` com `interno` |
| DeclarativePolicy | funções `.ge` em `projects/permissions.ge` sobre `members` |
| Gitaly | capability `git.*` (CLI git, sem shell) |
| Workhorse git HTTP | `git.servir_http` chamado depois da autorização `.ge` |
| Sidekiq | `tarefas.enfileirar` executando função `.ge` |
| gitlab-runner | runner escrito em `.ge` usando `processo.executar` e a mesma API `/api/v4/jobs/*` |
| HAML/Vue | páginas `.ge` com o registro `ui.*` |

## Fora do escopo do núcleo (classificado em FEATURE_INVENTORY)

GraphQL, EE-only, Pages, Container/Package registries, Kubernetes/clusters, Wiki, Snippets, Service Desk, LDAP/SAML/OAuth,
2FA, Elasticsearch/Zoekt, Import/Export, Mirroring, Geo, Releases, Environments/Deployments, SSH git transport.
Cada um aparece no inventário com status e justificativa — nenhum é omitido silenciosamente.
