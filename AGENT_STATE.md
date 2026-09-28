# AGENT_STATE

## CURRENT_PHASE
Fase 3 — GitLab em linguagem de intenção; próximo: paginação SQL (G40), notificações (todos/eventos), admin, busca global.

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
- Interface web gerada por `crie página` (sem botões zumbis, verificada por rastreador e capturas de tela).
- E2E: fluxos 1–5 (git real, issues, merge requests, pipelines com executor local), grupos/papéis, vazamento de issues privadas — PASS (também sob -race).
- `ge explain <dado>` / `ge check` entendem a camada de intenção.
- Skill de simplicidade em `skills/germanio-simplicity/SKILL.md` (gate no CLAUDE.md); issues refeitas por ela.
- Domínio GitLab em português; nomes da API v4 em `compatibilidade.ge`.
- Webhooks (fila persistente), escopos de token, Markdown seguro (`formatado`), bloqueio de usuário com E2E.
- Segurança: IP real só de proxies declarados (G25); SQLite com busy_timeout.
- Estrutura de projeto `backend/` + `frontend/` (`germanio init`, importação de pastas,
  `importar X do backend`, papéis por pasta); GitLab reorganizado nela.
- Extensão VS Code (ícone cristal, gramática de intenção, temas) em `vscode-germanio/`.

## IN_PROGRESS
- Próximo: G40 (paginação SQL), notificações/todos, admin, busca global, G54/G55 (runner externo, artefatos).

## NEXT
5. `ge explain`/`ge check`/`ge graph` sobre a camada de intenção.
6. Escopos de token (G33), transação (G39), paginação SQL (G40).

## FUTURE_PHASE (pedido do usuário, 2026-09-28)
Depois do GitLab, provar os formatos que ele não cobre, cada um com um app de referência
em `.ge` e as capabilities genéricas que faltarem:
1. Tempo real pesado — chat, colaboração ao vivo, jogos.
2. Interface muito interativa — editores, painéis arrastáveis, apps estilo Figma.
3. Escala muito alta — milhões de registros por lista (inclui terminar G40: visibilidade
   por registro em SQL para pessoas conectadas).
4. Outros tipos de programa — processamento de dados, mobile/offline, programas de sistema.

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
- D9 — Capability × permissão: `X pode <verbo>` sem objeto é capability do dado; com objeto é permissão.
- D10 — Domínio sempre em português; compatibilidade externa só por vocabulário.
- D11 — Pendências de simplificação em regras.ge: endereço hierárquico (caminho_completo), teto de visibilidade, teto de papel ao adicionar membro, branch protegida declarativa (G41–G44).
