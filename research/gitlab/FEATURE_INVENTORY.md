# GitLab FOSS — Inventário de features

Legenda — **Germanio Support**: SUPPORTED / PARTIAL / MISSING (ver GERMANIO_GAPS.md).
**Status** da implementação `.ge`: PASS / PARTIAL / FAIL / BLOCKED / NOT_STARTED. Evidência em EQUIVALENCE.md.

Escopo núcleo v1 = features marcadas **[N]** (fluxos 1–5 da missão + testes negativos).

## Identidade

| ID | Nome | Comportamento GitLab | Frontend | Backend | Dados | Permissões | Dependências | Germanio Support | Status | Tests |
|----|------|----------------------|----------|---------|-------|-----------|--------------|------------------|--------|-------|
| ID-01 [N] | Registro | username/email únicos, senha ≥ 8, cria namespace de usuário | `/users/sign_up` | `POST /api/v4/users` (admin) / form | users, namespaces | anônimo (se signup habilitado) | cripto | G10 | PASS | TestFluxo1 |
| ID-02 [N] | Login/logout | senha bcrypt, bloqueio após tentativas, sessão cookie | `/users/sign_in` | form | users | anônimo | sessao, cripto | G11 | PASS | TestFluxo1, TestInterfaceWeb |
| ID-03 [N] | Personal Access Token | token com escopos, digest armazenado, expiração, revogação | perfil | `POST /api/v4/user/personal_access_tokens`, header `PRIVATE-TOKEN` | personal_access_tokens | dono | cripto | G10 | PASS | TestFluxo2, TestEscoposDeToken |
| ID-04 | Usuário atual | `GET /api/v4/user` | — | API | users | autenticado | | | PASS | TestFluxo1 |
| ID-05 | Admin: bloquear usuário | `state=blocked` impede login e API | admin | `POST /users/:id/block` | users | admin | | | PASS | TestBloquearUsuario |
| ID-06 | Recuperação de senha / confirmação de e-mail | tokens por e-mail | forms | Devise | users | anônimo | email | SUPPORTED (GEP 0008, 0031) | PASS (confirmação de e-mail: capability `tenha confirmação de e-mail`, GEP 0031 em teste) | TestRecuperacaoDeSenha, TestConfirmacaoDeEmail |
| ID-07 | 2FA, OAuth, LDAP, SAML, WebAuthn | TOTP, códigos de recuperação | perfil, login | Devise two-factor | users | dono | — | PARTIAL (GEP 0032) | PARTIAL (2FA TOTP feito; OAuth/OIDC em andamento; LDAP/SAML/WebAuthn fora) | TestDoisFatores, TestDoisFatoresGitLab |
| ID-08 | Chaves SSH | cadastro de chave pública | perfil | `/user/keys` | keys | dono | SSH | SUPPORTED (GEP 0032, `chave pública`) | PASS (cadastro, impressão digital; transporte SSH em RP-10) | TestChavesSSH |

## Grupos e namespaces

| ID | Nome | Comportamento | Frontend | Backend | Dados | Permissões | Deps | Support | Status | Tests |
|----|------|--------------|----------|---------|-------|-----------|------|---------|--------|-------|
| GR-01 [N] | Criar grupo | path único global com namespaces, criador vira Owner | `/groups/new` | `POST /api/v4/groups` | namespaces(type=Group), members | autenticado com `can_create_group` | | | PASS | TestGruposMembrosPapeis |
| GR-02 | Subgrupos | `parent_id`, caminho completo | | `parent_id` | namespaces | Owner/Maintainer do pai | | | PASS | TestGruposMembrosPapeis |
| GR-03 [N] | Membros de grupo | access_level, herdado pelos projetos do grupo | members | `/groups/:id/members` | members | Owner (Maintainer p/ ≤ Maintainer) | | | PASS | TestGruposMembrosPapeis |
| GR-04 | Visibilidade de grupo | privado/interno/público | | | | | | | PASS | TestGruposMembrosPapeis |

## Projetos

| ID | Nome | Comportamento | Frontend | Backend | Dados | Permissões | Deps | Support | Status | Tests |
|----|------|--------------|----------|---------|-------|-----------|------|---------|--------|-------|
| PR-01 [N] | Criar projeto | no namespace do usuário ou grupo; cria repositório bare; criador Owner | `/projects/new` | `POST /api/v4/projects` | projects, members | autenticado / Developer+ no grupo | git | G13 | PASS | TestFluxo1 |
| PR-02 [N] | Ver projeto | resolve `ns/path`; README, branches, visibilidade | `/:ns/:proj` | `GET /api/v4/projects/:id` (id ou path url-encoded) | projects | por visibilidade/membro | | | PASS | TestFluxo1 |
| PR-03 [N] | Membros do projeto | adicionar/remover/alterar nível | members | `/projects/:id/members` | members | Maintainer+ | | | PASS | TestFluxo4 (adiciona membro) |
| PR-04 | Editar/arquivar/remover projeto | | settings | `PUT/DELETE /projects/:id`, archive | projects | Maintainer / Owner | git | | PASS | TestProjetoArquivado (arquivar por `archived`; endpoints POST archive/unarchive ainda não) |
| PR-05 | Fork | cópia com repositório | botão Copiar | `POST /projects/:id/fork` | projects | quem vê e baixa código | git | SUPPORTED (GEP 0029, `copiar`) | PASS (sem `/forks` e fork para namespace de pessoa) | TestFork, TestCopiasDeReceitas, TestCopy |
| PR-06 | Estrelas, tópicos, avatar | marcar, filtrar, imagem | projeto | star/unstar, `topic`, avatar | projects | quem vê | uploads | SUPPORTED (GEP 0030, `recebe estrelas`) | PASS (sem `/starrers`; um tópico por consulta) | TestEstrelas, TestTopicos, TestAvatar |

## Repositórios

| ID | Nome | Comportamento | Frontend | Backend | Dados | Permissões | Deps | Support | Status | Tests |
|----|------|--------------|----------|---------|-------|-----------|------|---------|--------|-------|
| RP-01 [N] | Clone HTTP | smart HTTP upload-pack | — | `/:ns/:proj.git/info/refs`, `git-upload-pack` | repo | Reporter+ ou público | git http | G14 | PASS | TestFluxo2 |
| RP-02 [N] | Push HTTP | receive-pack; branch protegida exige Maintainer | — | `git-receive-pack` | repo | Developer+ | git http | G14 | PASS | TestFluxo2, TestGrupos |
| RP-03 [N] | Árvore e arquivo | navegação por ref/caminho | tree/blob | `GET /repository/tree`, `/repository/files/:path/raw` | repo | leitura | git | G13 | PASS | TestFluxo2 |
| RP-04 [N] | Commits | lista e detalhe com diff | commits | `/repository/commits`, `/commits/:sha`, `/commits/:sha/diff` | repo | leitura | git | G13 | PASS | TestFluxo2 |
| RP-05 [N] | Branches | listar/criar/remover | branches | `/repository/branches` | repo | Developer+ para criar | git | G13 | PASS | TestGrupos |
| RP-06 | Branches protegidas | push/merge por nível | settings | `/protected_branches` | protected_branches | Maintainer+ | | | PASS (branch padrão e branches protegidas por nome ou padrão com *, como dados do projeto; quem pode fazer merge numa branch protegida ainda segue mesclar) | TestGruposMembrosPapeis |
| RP-07 | Tags | criar, listar, remover | `/-/tags` | `/repository/tags` | refs | Developer+ para criar | git | SUPPORTED | PASS (tags leves por git push e pela API, com as regras de enviar código; tags anotadas e protegidas ainda não) | TestTags |
| RP-08 | Editar arquivo pela web | commit direto | editor | `POST /repository/commits` | repo | Developer+ | git | G13 | PASS (FASE 3: editar pela API e pela página do arquivo; commit da pessoa com as regras de enviar código e branches protegidas; as execuções começam como num push) | TestEditarArquivoPelaWeb |
| RP-09 | Compare | diff entre refs | compare | `/repository/compare` | repo | leitura | git | | PASS (API) | — |
| RP-10 | SSH, LFS, mirrors, archive download | download do código, arquivos grandes, espelhos | repositório | `repository/archive`, `info/lfs`, `remote_mirrors` | remote_mirrors | baixar/enviar código, maintainer | git | PARTIAL (GEP 0035, 0036) | PARTIAL (archive, LFS e espelhos feitos; transporte SSH em andamento) | TestArchive, TestBaixarCodigo, TestGitLFS, TestGitLFSNoProjeto, TestEspelhos, TestEspelhosRemotos |

## Issues

| ID | Nome | Comportamento | Frontend | Backend | Dados | Permissões | Deps | Support | Status | Tests |
|----|------|--------------|----------|---------|-------|-----------|------|---------|--------|-------|
| IS-01 [N] | Criar issue | iid sequencial por projeto; título obrigatório | new issue | `POST /projects/:id/issues` | issues | Guest+ (e público autenticado) | sequência | G09 | PASS | TestFluxo3 |
| IS-02 [N] | Editar / fechar / reabrir | `state_event=close|reopen`, `closed_at`, `closed_by` | issue | `PUT /projects/:id/issues/:iid` | issues | autor ou Planner/Reporter+ | | | PASS | TestFluxo3 |
| IS-03 [N] | Comentários | notas + notas de sistema | issue | `/issues/:iid/notes` | notes | Guest+ | markdown | G18 | PASS | TestFluxo3 |
| IS-04 [N] | Assignees | um ou mais | issue | `assignee_ids` | issue_assignees | Reporter+ | | | PASS | TestFluxo3 |
| IS-05 [N] | Labels | por projeto, cor, vínculo | labels | `/labels`, `labels=` | labels, label_links | Reporter+ administra | | | PASS | TestFluxo3, TestLabelsDeOutroProjetoNaoEntram |
| IS-06 | Milestones | | milestones | `/milestones` | milestones | Reporter+ | | | PASS | TestMilestones |
| IS-07 [N] | Filtros e busca | state, labels, assignee, author, search | lista | `GET /issues?state&labels&search` | issues | leitura | | | PASS | TestFluxo3 |
| IS-08 [N] | Confidencial | visível só a Reporter+, autor, assignees | | `confidential` | issues | | | | PASS | TestFluxo3, TestIssuesPrivadasNaoVazam |
| IS-09 | Boards, weights, time tracking, links, moves | colunas por estado, peso, estimativa e tempo gasto, ligações, mover | quadro, API | `weight`, `time_stats`, `links`, `move` | issues | developer | — | SUPPORTED (GEP 0023, 0033, 0034) | PASS (soma de pesos por milestone fica para uma GEP de agregados; só `relates_to`; páginas ainda sem mover/tempo/ligações) | TestPorEstado, TestPesosDasIssues, TestControleDeTempo, TestLigacoesEntreIssues, TestMoverIssue, TestMudarDeLugar |

## Merge Requests

| ID | Nome | Comportamento | Frontend | Backend | Dados | Permissões | Deps | Support | Status | Tests |
|----|------|--------------|----------|---------|-------|-----------|------|---------|--------|-------|
| MR-01 [N] | Criar MR | branches existentes e diferentes; iid por projeto | new MR | `POST /projects/:id/merge_requests` | merge_requests | Developer+ | git | G13 | PASS | TestFluxo4 |
| MR-02 [N] | Diff / changes / commits | base = merge-base | changes | `/changes`, `/diffs`, `/commits` | repo | leitura | git diff | G13 | PASS | TestFluxo4 |
| MR-03 [N] | Review: comentários e aprovação | notas, `approve` | discussion | `/notes`, `/approve` | notes, approvals | Developer+ aprova | | | PASS | TestFluxo4 |
| MR-04 [N] | Merge | merge commit; conflito → `cannot_be_merged` | merge btn | `PUT /merge` | merge_requests | Developer+ (Maintainer se protegida) | git merge | G13 | PASS | TestFluxo4 |
| MR-05 [N] | Conflito detectado | `merge_status=cannot_be_merged`, 406/405 no merge | | | | | git merge-tree | G13 | PASS | TestFluxo4 |
| MR-06 | Fechar/reabrir, draft | | | `state_event` | | | | | PASS | TestFluxo4 |
| MR-07 | Squash, rebase, merge when pipeline succeeds, approvals rules | squash, linear, merge agendado, mínimo de aprovações | botões do MR | `merge` (`squash`, `merge_when_pipeline_succeeds`), `merge_method`, `approvals_required` | merge_requests | developer | git, CI | SUPPORTED (GEP 0026, 0027) | PASS (sem `PUT /merge`, `GET /approvals` e regras de aprovação por papel; botão web de agendar ainda não) | TestAprovacoesEMetodosDeMesclagem, TestMinimoDeAprovacoes, TestFormasDeMesclar, TestMesclarQuandoPassar, TestSquash, TestRebaseFastForward |

## CI/CD

| ID | Nome | Comportamento | Frontend | Backend | Dados | Permissões | Deps | Support | Status | Tests |
|----|------|--------------|----------|---------|-------|-----------|------|---------|--------|-------|
| CI-01 [N] | Pipeline no push | lê `.gitlab-ci.yml` do commit; stages + jobs | pipelines | post-receive | ci_pipelines, ci_stages, ci_builds | Developer+ | yaml, tarefas | G15 G17 | PASS | TestFluxo5 |
| CI-02 [N] | Runner registro e token | `POST /api/v4/runners` | admin | runners | ci_runners | admin/Maintainer | cripto | | PASS | TestRunnerOficial, TestProtocoloRunner |
| CI-03 [N] | Runner request job | `POST /api/v4/jobs/request` entrega próximo job pendente | | API | ci_builds | token runner | | | PASS | TestRunnerOficial |
| CI-04 [N] | Log do job | `PATCH /jobs/:id/trace` incremental; visualização | job log | API | trace | leitura | | | PASS | TestFluxo5 |
| CI-05 [N] | Resultado do job | `PUT /jobs/:id state=success|failed`; avança stages; status agregado | pipeline graph | API | | | | | PASS | TestFluxo5 |
| CI-06 [N] | Retry / cancel | novo build / estado canceled | job | `/jobs/:id/retry|cancel` | | Developer+ | | | PASS | TestFluxo5 |
| CI-07 | Variáveis de CI | projeto; mascaradas | settings | `/variables` | ci_variables | Maintainer+ | cripto | SUPPORTED (GEP 0015, em teste) | PASS (variáveis do projeto chegam às etapas do executor local e do runner oficial; valor oculto nunca volta pela API e aparece como [MASKED] no log; os nomes CI_* vencem; variáveis de grupo, protegidas e por ambiente ainda não) | TestVariaveisDeCI, TestRunnerOficial |
| CI-08 | Artefatos | upload/download | job | `/jobs/:id/artifacts` | ci_job_artifacts | | uploads | SUPPORTED (GEP 0014, em teste) | PASS (o runner oficial envia `artifacts: paths`, que voltam em zip para quem vê o job; o executor local junta e entrega artefatos; jobs recebem os artefatos das etapas de que dependem, pelo protocolo do runner com JOB-TOKEN; `expire_in` vence e apaga. Navegação dentro do zip, `artifacts: when/reports` não) | TestRunnerOficial, TestProtocoloRunner, TestArquivos, TestExpiracaoDosArtefatos |
| CI-09 | needs/DAG, rules/only/except, when:manual, allow_failure | | | | | | | | PASS (`needs`/DAG, `dependencies`, `rules` com `if` sobre branch/fonte, `only`/`except`, `when: manual/always/never`, `allow_failure`, no formato nativo genérico e traduzidos pelo adaptador; `workflow: rules`, `rules: changes/exists/variables` e `needs` entre projetos não — recusados com explicação) | TestPipelineGrafoERegras, TestExecucaoNativaSemGitLab, TestEtapasPrecisamDeOutras |
| CI-10 | Isolamento de execução (containers) | runner docker executor | | | | | docker | MISSING | PASS (executor local com GERMANIO_EXECUTOR=docker: todos os comandos de um passo num único container, com as variáveis do passo, sem rede, sem capabilities, sem escalada de privilégio, limite de processos e o usuário do host; o runner oficial usa o próprio executor docker) | TestPassoEmConteiner |

## Integrações, notificações, admin

| ID | Nome | Comportamento | Frontend | Backend | Dados | Permissões | Deps | Support | Status | Tests |
|----|------|--------------|----------|---------|-------|-----------|------|---------|--------|-------|
| WH-01 | Webhooks de projeto | eventos push/issue/MR/pipeline com `X-Gitlab-Token`, em worker | settings | `/hooks` | web_hooks | Maintainer+ | tarefas, http | G15 | PASS | TestWebhooks |
| NT-01 | Todos | atribuição/menção gera todo | `/dashboard/todos` | `/todos` | todos | dono | | SUPPORTED (GEP 0009) | PASS (atribuição e menção @nome em issues e comentários, só para quem pode ver; marcar como feito e excluir) | TestPendencias, TestMencoes, TestFluxo3Issues |
| NT-02 | Eventos de atividade | histórico | — | `/events`, `/projects/:id/events` | events | quem vê | — | SUPPORTED (histórico + adaptador, GEP 0033) | PASS (nomes de ação do GitLab na borda, paginação, sem vazamento; filtro `action` ainda não) | TestAtividade, TestEventosDoProjeto |
| NT-03 | E-mail de notificação | e-mail por pendência e menção | perfil | `/notification_settings` | users | dono | email | SUPPORTED (GEP 0013, emenda) | PASS (pendências e menções por e-mail depois do commit, sem vazar títulos; escolha por pessoa `avisos_por_email`; níveis finos e resumos não) | TestPreferenciaDeAvisos, TestAvisosPorEmailEscolhaDaPessoa |
| AD-01 | Admin dashboard | contagens | `/painel` | `/application/statistics` | — | admin | — | SUPPORTED (GEP 0012, 0033) | PASS (só o administrador recebe os números; 403/401 para os outros) | TestEstatisticasDaAplicacao |
| SR-01 | Busca global | projetos/issues/MRs | search | `/search` | | leitura | | | PASS | TestBuscaGeral (API; página de busca ainda não) |
| UP-01 | Uploads em markdown | `/uploads` | | | uploads | | | PARTIAL (upload genérico existe) | PASS (FASE 3: imagem colada ou solta num texto formatado vai para o registro da página — a issue de um comentário — e vira Markdown; só imagens; vista por quem vê o registro; sai com ele. Pela superfície de integração `/uploads` ainda não) | TestImagensNoTexto, TestNavegador |

## Fora do núcleo (classificado, não omitido)

| ID | Área | Status | Justificativa |
|----|------|--------|---------------|
| X-01 | GraphQL API | BLOCKED | Germanio não tem servidor GraphQL genérico; REST v4 cobre os fluxos núcleo |
| X-02 | Wiki, Snippets, Pages, Releases, Environments, Deployments | NOT_STARTED | dependem do núcleo primeiro |
| X-03 | Container/Package registry, Kubernetes, Terraform state | BLOCKED | exigem protocolos (OCI, k8s) sem capability Germanio |
| X-04 | Import/Export, mirrors, Geo | NOT_STARTED | |
| X-05 | Busca avançada (Elasticsearch/Zoekt) | BLOCKED | sem capability de índice full-text |
| X-06 | Recursos EE | WONTFIX | fora do FOSS |
