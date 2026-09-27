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
| Tokens de acesso: criar, listar sem segredo, revogar, expirar | PersonalAccessToken | `segredo prefixo`, `expira em`, `revogavel` | `TestFluxo2CloneCommitPush` | PARTIAL | escopos não restringem ainda (G33); rota `/personal_access_tokens` (GitLab: `/user/personal_access_tokens`) |
| Fluxo 3: issue → comment → assign → close | Issues::*Service | — | — | NOT_STARTED | próximo |
| Fluxo 4: branch → push → MR → diff → review → merge | MergeRequests::*Service | capability git pronta (merge-tree, merge) | — | NOT_STARTED | |
| Fluxo 5: pipeline → job → runner → logs → result | Ci::* + runner API | — | — | NOT_STARTED | |
| Negativos: acesso proibido, input inválido, repo/branch inexistente, token inválido, grupo privado | policies/validações | automáticos do runtime + regras | fluxos 1/2/grupos | PARTIAL | faltam: conflito, job falhando, upload inválido |
| Interface web | HAML/Vue | páginas `crie página` | — | NOT_STARTED | runtime de páginas pendente (G19) |
