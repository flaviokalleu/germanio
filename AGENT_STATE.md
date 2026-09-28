# AGENT_STATE

## CURRENT_PHASE
2026-09-28, em ordem (histórico resumido; detalhes nos commits e em GERMANIO_GAPS.md):
1. Estudo dirigido de linguagens, frontend e performance: FEITO (`docs/research/`).
2. Norma de eficiência e suíte de benchmarks: FEITO; primeiro baseline em
   `bench/resultados/2026-09-28-e421c8e.txt` e limites de regressão em `bench/README.md`.
3. Lançamento e descoberta: FEITO (README en/pt-BR, docs públicos com `tooling/doctest`,
   exemplos testados, comunidade, release, About do GitHub, SEO, plano editorial). O que só o
   proprietário pode fazer está em `docs/launch/LANCAMENTO.md`.
4. Pesquisa do ecossistema: FEITO (`docs/research/languages/`: triagem, 24 linguagens,
   descobertas, pacotes, concorrência, segurança, matriz, lições, respostas; GEPs 0002–0007
   em rascunho).
5. Correções que a pesquisa revelou: FEITO para os P0/P1 de segurança e dados — G65, G83,
   G91, G98–G102 (segurança); G85 (visibilidade 220× mais rápida), G87, G88, G89 (quase),
   G93 (parcial), G94, G95, G96 (performance/dados); G67, G68, G74, G84, G103 (linguagem);
   G75 (acessibilidade, parcial); G76, G80 (parcial).
6. GEPs 0002 (seções de página), 0008 (recuperação de senha) e 0009 (pendências): ACEITAS
   pelo mantenedor e normativas. Rascunhos esperando decisão: 0003–0007.
7. G86 (efeitos externos depois do commit) e G93 (renames explícitos, nenhum inferido): FEITO,
   com testes; composição de arquivos protegida por teste de reflexão e de equivalência.
Próximo: GitLab pelo inventário (NT-02 atividade, AD-01 painel — depende de agregados), com
uma GEP por construção nova.

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
