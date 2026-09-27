# Equivalência GitLab FOSS × GitLab Germanio

PARTIAL não significa concluído. Cada PASS aponta para um teste executável
(`go test ./examples/gitlab-foss/e2e/`).

| Feature | GitLab Original | Germanio | Tests | Status | Notes |
|---------|-----------------|----------|-------|--------|-------|
| Fluxo 1: register → login → create project → open project | Devise, OAuth password grant, Projects::CreateService, GET /projects/:id | `tenha login/cadastro`, `login aceita oauth`, `usuario pode criar projetos`, `disponibilize projetos` | `TestFluxo1RegistroLoginProjeto` | PASS | cadastro em `/cadastro` (GitLab: `/users` formulário) |
| Fluxo 2: create project → clone → commit → push → view commit | Gitaly + Workhorse smart HTTP, PAT, repository API | `projeto tem repositório`, `tokens de acesso` com `segredo`, `baixar/enviar código` | `TestFluxo2CloneCommitPush` | PASS | clone/push com git real via HTTP Basic + PAT |
| Grupos, subgrupos, membros e papéis | Group, GroupMember, ProjectPolicy | `tenha papeis`, `membros com papel`, `herda membros`, `administrar` | `TestGruposMembrosPapeis` | PASS | membros usam `papel` e `pessoa_id`; aceita `user_id`/`access_level` na entrada |
| Branch padrão protegida (push) | ProtectedBranch, pre-receive | `antes de enviar código para projeto` | `TestGruposMembrosPapeis` | PASS | force-push não é distinguido (ver G34) |
| Último owner não sai | Members::DestroyService | `antes de excluir membro` + `recuse` | `TestGruposMembrosPapeis` | PASS | |
| Bloquear/desbloquear usuário | Users::BlockService | `usuario começa ativo` + `usuario pode bloquear/desbloquear` | (coberto pela máquina de estados) | PARTIAL | falta teste E2E dedicado |
| Tokens de acesso: criar, listar sem segredo, revogar, expirar | PersonalAccessToken | `segredo prefixo`, `expira em`, `revogavel` | `TestFluxo2CloneCommitPush` | PARTIAL | escopos não restringem ainda (G33); rota `/personal_access_tokens` (GitLab: `/user/personal_access_tokens`) |
| Fluxo 3: issue → comment → assign → close | Issues::*Service, Notes, Labels | `issue começa aberta`, `issue pode fechar/reabrir/ser confidencial`, `issue confidencial pode ser vista por`, `numero por projeto` | `TestFluxo3Issues` | PASS | fechar/reabrir via `POST …/close|reopen` (GitLab: `PUT state_event`); labels por id (GitLab: por nome) |
| Fluxo 4: branch → push → MR → diff → review → merge | MergeRequests::*Service, MergeService, MergeabilityCheck | `origem/destino: branch`, `recebe aprovações`, `pode mesclar`, `mesclado é final`, branch padrão protegida | `TestFluxo4MergeRequest` | PASS | conflito → 406; rascunho não mescla; changes/commits a partir da base comum |
| Fluxo 5: pipeline → job → runner → logs → result | Ci::CreatePipelineService, ProcessPipelineService, gitlab-runner | `projeto executa pipelines a cada envio de código conforme ".gitlab-ci.yml"` + executor local | `TestFluxo5Pipelines` | PARTIAL | etapas, falha→ignorado, allow_failure, manual/play, retry, cancel, log, configuração inválida, execução manual PASS; runner é interno (protocolo do gitlab-runner externo não implementado); isolamento por diretório/processo, `docker` opcional sem rede; artefatos e needs/rules ainda não |
| Negativos: acesso proibido, input inválido, repo/branch inexistente, token inválido, grupo privado | policies/validações | automáticos do runtime + regras | fluxos 1/2/grupos | PARTIAL | faltam: conflito, job falhando, upload inválido |
| Interface web | HAML/Vue | páginas `crie página` | — | NOT_STARTED | runtime de páginas pendente (G19) |
