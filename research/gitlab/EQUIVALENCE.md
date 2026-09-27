# Equivalência GitLab FOSS × GitLab Germanio

PARTIAL não significa concluído. Cada linha PASS aponta para um teste executável.

| Feature | GitLab Original | Germanio | Tests | Status | Notes |
|---------|-----------------|----------|-------|--------|-------|
| Fluxo 1: register → login → create project → open project | Devise + Projects::CreateService | — | — | NOT_STARTED | |
| Fluxo 2: create project → clone → commit → push → view commit | Gitaly + Workhorse | — | — | NOT_STARTED | |
| Fluxo 3: issue → comment → assign → close | Issues::*Service | — | — | NOT_STARTED | |
| Fluxo 4: branch → push → MR → diff → review → merge | MergeRequests::*Service | — | — | NOT_STARTED | |
| Fluxo 5: pipeline → job → runner → logs → result | Ci::CreatePipelineService + runner API | — | — | NOT_STARTED | |
| Negativos: acesso proibido, input inválido, repo/branch inexistente, conflito, job falhando, token inválido, upload inválido | policies/validações | — | — | NOT_STARTED | |
