# AGENT_STATE

## CURRENT_PHASE
Fase 3 — GitLab em linguagem de intenção; próximo: Issues (fluxo 3).

## CURRENT_GOAL
Continuar o GitLab por intenção: issues → merge requests → CI/CD → páginas (UI),
evoluindo o Germanio a cada parede encontrada.

## LAST_GOOD_COMMIT
(ver `git log -1`; todos os testes verdes neste commit)

## COMPLETED
- Auditoria Germanio + GitLab FOSS (research/gitlab/*).
- Plataforma full-stack endurecida (rotas, erros com posição, mapas, cadeias, banco com filtros,
  cripto, sessão/CSRF, segurança de `/api/_eval`).
- Camada de intenção (docs/INTENCAO.md): dados, relações, login, papéis/membros/herança,
  permissões, hooks `antes de`/`quando`, integração, repositórios Git.
- Capability Git (runtime/git): CLI sem shell, smart HTTP com verificação antes de atualizar refs.
- GitLab `.ge`: identidade, tokens, grupos, subgrupos, membros, projetos, repositórios, navegação.
- E2E: fluxo 1, fluxo 2 (git real), grupos/papéis — PASS.

## IN_PROGRESS
- Issues.

## NEXT
1. Issues: `projeto tem issues`, labels, comentários, responsáveis, fechar/reabrir, confidencial.
2. Merge requests: origem/destino, diff, aprovação, merge, conflito.
3. CI/CD: pipelines, etapas, jobs, runner, logs (tarefas + processo + yaml).
4. Páginas (`crie página`) — UI conectada, sem botões zumbis.
5. `ge explain`/`ge check`/`ge graph` sobre a camada de intenção.
6. Escopos de token (G33), transação (G39), paginação SQL (G40).

## BLOCKERS
Nenhum.

## GERMANIO_GAPS
Ver GERMANIO_GAPS.md (G01–G40).

## TEST_STATUS
`go build ./...` OK · `go test ./...` OK (13 pacotes, inclui E2E GitLab) · exemplos antigos idênticos ·
teste de arquitetura: `grep -ri gitlab runtime compiler tooling cli` vazio.

## IMPORTANT_DECISIONS
- D1 — Dialeto full-stack como base; o núcleo estrito segue separado.
- D2 — Nenhum Go conhece GitLab (teste de arquitetura acima).
- D3 — Git via CLI `git` sem shell; merge em bare via `merge-tree`.
- D4 — Escopo núcleo = fluxos 1–5 + negativos; resto classificado no inventário.
- D5 — Intenção antes de técnica: CRUD, login, permissões, relações e API são derivados;
  `rota`/`requisicao`/`responder` ficam como nível 4.
- D6 — Hooks em duas fases: `antes de` (pode ajustar `dados` e recusar) e `quando` (depois).
- D7 — Mensagens em português por padrão; `mensagens em inglês` para compatibilidade de API.
- D8 — Registro invisível responde 404; dentro de algo com membros só papéis autorizam criação.
