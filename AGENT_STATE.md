# AGENT_STATE

## CURRENT_PHASE
Fase 2 — capabilities fundamentais do Germanio (P0 da GERMANIO_GAPS).

## CURRENT_GOAL
Tornar o dialeto full-stack capaz de expressar uma API REST real em `.ge`
(G01–G07, G12, G22), depois cripto/sessão (G10, G11).

## LAST_GOOD_COMMIT
ab8c77b (baseline: `go build ./...` e `go test ./...` verdes, 14 pacotes com testes)

## COMPLETED
- Auditoria do Germanio (lexer, dois parsers, semantic, dois engines, servidor, banco, auth, jobs, CLI, tooling).
- Baseline verde registrado.
- Auditoria de implementações GitLab anteriores: **nenhuma existe** (nenhum `runtime/gitlab/`, nenhuma referência a GitLab).
- Auditoria do GitLab FOSS @ c8c0c71 → research/gitlab/ARCHITECTURE.md, FEATURE_INVENTORY.md.
- GERMANIO_GAPS.md inicial (G01–G23).

## IN_PROGRESS
- G01–G07, G12, G22.

## NEXT
1. cripto + token/sessão + CSRF (G10, G11, G20)
2. examples/gitlab-foss: identity, groups, projects (fluxo 1)
3. runtime/git + git smart HTTP (G13, G14) → fluxo 2
4. issues (fluxo 3), merge requests (fluxo 4)
5. tarefas + processo + yaml → CI (fluxo 5)
6. registro ui.* → frontend
7. webhooks, notificações, admin; ge check/graph/explain para o dialeto full-stack

## BLOCKERS
Nenhum no momento.

## GERMANIO_GAPS
Ver GERMANIO_GAPS.md (23 abertos no início).

## TEST_STATUS
`go build ./...` OK · `go test ./...` OK (baseline).

## IMPORTANT_DECISIONS
- **D1 — Dialeto alvo**: o repositório tem dois front-ends. O núcleo estrito (`parser/germanio.go` + `semantic` +
  `runtime/germanio`) é tipado e tem bons diagnósticos, mas não tem mapas, registros, stdlib, servidor, banco nem UI
  (Fase 3 do ROADMAP pendente). O dialeto full-stack (`parser/parser.go` + `runtime/interpreter` + `runtime/servidor` +
  `runtime/banco`) já executa aplicações web. O GitLab será escrito no **dialeto full-stack**, que será endurecido
  (erros visíveis, contexto HTTP, consultas, capabilities). Levar o núcleo estrito ao full-stack é trabalho futuro
  registrado; não é escondido.
- **D2 — Sem runtime/gitlab**: nenhuma linha de Go conhece GitLab. Teste de arquitetura: `grep -ri gitlab runtime compiler tooling` deve ser vazio.
- **D3 — Git via CLI `git`** (2.43 disponível) com argumentos em array, sem shell, caminhos confinados à raiz
  configurada; merge em repositório bare via `git merge-tree --write-tree` (git ≥ 2.38).
- **D4 — Escopo núcleo v1** = fluxos 1–5 da missão + testes negativos; o resto do GitLab fica classificado no inventário.
