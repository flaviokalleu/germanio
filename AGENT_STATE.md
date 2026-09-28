# AGENT_STATE

## CURRENT_PHASE
Estudo dirigido (pedido do usuário, 2026-09-28): implementações open source de linguagens
(`docs/research/languages/`) e de frontend (`docs/research/frontend/`). Nenhuma mudança de
arquitetura, sintaxe, parser, runtime, formatter ou tooling até a pesquisa ser confrontada com
a norma (`docs/INTENCAO.md`, a skill, AGENTS.md, EVOLUTION, GAPS).
Antes da pausa: sintaxe hierárquica implementada; G64 (`ge fmt` para todo arquivo de
aplicação) commitado em e562a8c. Os arquivos `auth.ge` com o bloco `autenticacao` indentado
em 2 espaços (examples/whaticket, demo/organizado) não são formatáveis (o `fmt` também recusava
antes); anotado para depois.
Depois da pesquisa: G62 (seções de página), G61, G63; retomar o GitLab pelo inventário.

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
- Três camadas (domínio/core/adaptador) com teste de arquitetura `runtime/arquitetura_test.go`.
- Trabalho remoto genérico (`X executam Y`, `trabalho_remoto.*`, lease/heartbeat/cancelamento/
  idempotência) + adaptadores `integracoes/gitlab_runner.ge` e `integracoes/gitlab_ci.ge`;
  gitlab-runner oficial 19.4.1 passa (`TestRunnerOficial`, precisa `GITLAB_RUNNER_BIN`).
- Transações por requisição (G39), endereços hierárquicos (G41), labels por nome, milestones,
  busca geral, arquivamento (`X <condição> é somente leitura`), papel mínimo com herança e
  concorrência, administrador inicial, validação de segmentos e `endereços reservados`.
- Domínio GitLab (`backend/`, `frontend/`) sem nenhuma lógica: `regras.ge` removido.

## IN_PROGRESS
Nenhum trabalho pela metade. Último commit verde: `afeba1b` (busca geral). Árvore limpa.
Parado ANTES de começar pendências/todos (NT-01): só houve leitura de código, nenhuma edição.
Desenho previsto para quando retomar (sujeito à nova diretriz de sintaxe):
`tenha pendências`, `issue avisa responsaveis e mencionados`, `comentario avisa mencionados`;
só avisa quem pode ver o registro; quem agiu não avisa a si mesmo.

## NEXT (depois da nova diretriz)
- Pendências/todos (NT-01), painel de administração (AD-01), página de busca.
- G40: pré-filtro SQL de visibilidade para pessoas conectadas.
- G55: artefatos de job; trava `FOR UPDATE` do papel mínimo testada em PostgreSQL/MySQL.

## BACKLOG (antigo)
- `ge explain`/`ge check` completos conforme docs/INTENCAO.md (proveniência de inferências).

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
