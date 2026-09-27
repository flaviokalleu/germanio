# GERMANIO_GAPS

Lacunas do Germanio descobertas ao reimplementar o GitLab FOSS em `.ge`.
Toda solução deve ser **genérica** (útil para qualquer app `.ge`) e exposta a `.ge`.

Classificação: SUPPORTED · PARTIAL · MISSING · BUG — Prioridade: P0 (bloqueia o núcleo) … P3.
Status: OPEN · IN_PROGRESS · DONE (com testes) · WONTFIX (com motivo).

Dialeto alvo: **dialeto full-stack** (`compiler/parser/parser.go` + `runtime/interpreter` + `runtime/servidor`).
Motivo em AGENT_STATE.md › IMPORTANT_DECISIONS D1.

| ID | Necessidade | Onde surgiu | Por que não suporta | Categoria | Classe | Sev | Solução genérica | Status |
|----|-------------|-------------|---------------------|-----------|--------|-----|------------------|--------|
| G01 | Rota acessar requisição (corpo, params de caminho, query, headers, usuário) | toda API REST do GitLab | `rota` executa handler sem nenhum contexto HTTP | HTTP | MISSING | P0 | variável `requisicao` + padrões `:id` → `{id}` do `net/http` | OPEN |
| G02 | Rota definir status, corpo JSON, headers, redirect | toda API | só devolvia saída de `mostrar` | HTTP | MISSING | P0 | `retornar valor` = 200 JSON; `responder(status, valor)`; `redirecionar(url)` | OPEN |
| G03 | Erros em tempo de execução silenciosos | auditoria do interpreter | variável desconhecida vira o próprio nome; função desconhecida → nulo; erro de banco → nulo + log | Semântica | BUG | P0 | erro de execução com arquivo:linha; `falhar(status, msg)`; `tentar/erro` captura | OPEN |
| G04 | Literais de mapa `{chave: valor}` | payloads JSON, filtros | parser não aceita `{}` em expressão | Sintaxe | MISSING | P0 | literal de mapa na gramática de expressão | OPEN |
| G05 | Acesso encadeado `a.b.c`, `f(x).campo`, `lista[0].nome` | qualquer domínio | parser só aceita um nível `obj.campo` | Sintaxe | MISSING | P0 | cadeia pós-fixa genérica (membro, índice, chamada) | OPEN |
| G06 | Consultas ao banco: buscar por id, filtros, ordenação, paginação, contagem filtrada | todos os domínios | `.ge` só tinha listar/criar/atualizar/deletar/contar sem filtro; `buscar` não existia | Banco | MISSING | P0 | `modelo.buscar/encontrar/filtrar/contar/existe` com mapas de filtro parametrizados | OPEN |
| G07 | Modelos sem CRUD automático público | autorização do GitLab (issues confidenciais, projetos privados) | todo modelo vira `/api/<modelo>` aberto | Segurança | MISSING | P0 | modificador de modelo `interno` (sem CRUD/tela automáticos) | OPEN |
| G08 | Unicidade composta `(projeto, iid)`, índices compostos | issues/MRs/pipelines numerados por projeto | só `unico` por coluna | Banco | MISSING | P1 | `unico(a, b)` / `indice(a, b)` no modelo | OPEN |
| G09 | Contador atômico por escopo (iid) | issues `#1..n` por projeto | sem primitive | Banco | MISSING | P1 | `banco.sequencia(chave)` atômico | OPEN |
| G10 | Hash de senha, tokens seguros, HMAC, comparação constante | identidade, PAT, runners, webhooks | só md5/sha256 expostos | Crypto | MISSING | P0 | módulo `cripto.*` (bcrypt, crypto/rand, hmac) | OPEN |
| G11 | Sessão/usuário atual controlado pela aplicação | login/registro do GitLab em `.ge` | auth embutido fixo em Go (`/api/login`) | Auth | PARTIAL | P0 | `token.assinar/verificar` + `sessao.iniciar/encerrar` (cookie assinado) + `requisicao.usuario` | OPEN |
| G12 | Identificadores reescritos pelo léxico multilíngue | `issue.state`, `projeto.estado` | léxico troca palavra-chave/tradução pelo canônico mesmo após `.` | Léxico | BUG | P0 | preservar texto original (`Raw`) e usá-lo em nomes de campo/variável/chave | OPEN |
| G13 | Git: bare repo, refs, árvore, blob, commits, diff, merge, commit via web | repositórios, MRs | inexistente | Git | MISSING | P0 | `runtime/git` (CLI git sem shell, caminhos confinados) exposto como `git.*` | OPEN |
| G14 | Git smart HTTP (clone/push) com autorização na app | fluxo 2 | inexistente | Git/HTTP | MISSING | P0 | `git.servir_http(repo, servico)` entrega a conexão ao protocolo e devolve refs atualizadas | OPEN |
| G15 | Fila de tarefas persistente com retry/backoff/dead | CI, webhooks, notificações | `runtime/jobs` só em memória, não exposto a `.ge` | Jobs | PARTIAL | P1 | `tarefas.enfileirar("funcao", dados)` + tabela `_germanio_tarefas` | OPEN |
| G16 | Execução de processo segura | runner de CI | inexistente | Processo | MISSING | P1 | `processo.executar(prog, args, opcoes)` sem shell, timeout, dir confinado | OPEN |
| G17 | YAML | `.gitlab-ci.yml` | inexistente | Serialização | MISSING | P1 | `yaml.ler(texto)` | OPEN |
| G18 | Markdown seguro | issues, MRs, notas, README | inexistente | Texto | MISSING | P2 | `markdown.html(texto)` sanitizado | OPEN |
| G19 | UI de aplicação (não só CRUD): layout, tabelas, formulários, CodeViewer, DiffViewer, árvore, logs | frontend GitLab | renderer só gera CRUD de telas | UI | MISSING | P0 | registro `ui.*` renderizado no servidor com escape + CSRF | OPEN |
| G20 | CSRF para formulários com sessão por cookie | UI | inexistente | Segurança | MISSING | P0 | token CSRF automático no registro UI + verificação no servidor | OPEN |
| G21 | `ge check`/`graph`/`explain` entenderem apps do dialeto full-stack | stress test | tooling lê o modelo de intelligence, não rotas/funções | Tooling | PARTIAL | P2 | análise de rotas, funções chamadas e modelos referenciados | OPEN |
| G22 | Handler concorrente com escopo isolado | servidor | `EvalStatements` usa escopo global compartilhado | Concorrência | BUG | P0 | escopo por requisição filho do global | OPEN |
| G23 | Parser tolera tokens desconhecidos silenciosamente | auditoria | `default: p.advance()` em vários blocos | Diagnóstico | BUG | P2 | erro com linha dentro de corpos de rota/função | OPEN |
