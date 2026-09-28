# Auditoria de performance do runtime Germanio

- **Data:** 2026-09-28
- **Commit auditado:** `e203c7a` (árvore limpa, copiada para um diretório temporário antes das
  medições). Durante a auditoria o `master` avançou até `765a3bb`, com mudanças em
  `runtime/engine.go`, `servidor.go`, `paginas.go`, `websocket.go` e `auth.go`. **Todas as
  referências `arquivo:linha` abaixo valem para `e203c7a`.** Nas novas revisões as linhas
  podem ter mudado, e algum achado pode já ter sido corrigido.
- **Status:** medição ad hoc. Nada no repositório foi alterado para medir. Os benchmarks, o
  baseline e o gerador de carga ficaram fora da árvore, num diretório temporário. A suíte
  permanente fica em `bench/` (ver `bench/README.md` e [BENCHMARKING.md](BENCHMARKING.md)).
  Estes números não substituem os dela.
- **Norma servida:** [docs/INTENCAO.md › Eficiência](../../INTENCAO.md#eficiência-simples-para-o-humano-eficiente-para-a-máquina).
  A ordem é correto → seguro → mensurável → rápido. Meça antes de otimizar e pague apenas
  pelo que usar.

## Ambiente de medição

| Item | Valor |
| --- | --- |
| CPU | Intel Core i7-10750H @ 2.60 GHz, 6 núcleos / 12 threads, escala de frequência ativa (governor não fixado) |
| Memória | 15 GiB, com cerca de 6 GiB disponíveis durante as medições. Swap em zram e disco |
| Disco | NVMe HFM512GDJTNI-82A0A, **99% ocupado** (8 GiB livres), sob LVM |
| SO | Pop!_OS 24.04 LTS, kernel 7.1.5-76070105-generic x86_64 |
| Go | go1.27.1 linux/amd64 (`go.mod` declara `go 1.26.1`) |
| Driver | `modernc.org/sqlite v1.48.2` (sem CGO), o mesmo no Germanio e no baseline |
| Carga | gerador de ciclo fechado em Go, na mesma máquina que o servidor. Disputa CPU com ele e sofre de *coordinated omission* sob sobrecarga |

**Ruído conhecido.** Outros agentes rodavam benchmarks na mesma máquina. Em 3 das 18
repetições apareceram outliers de 5 a 60x (por exemplo, uma página a 37 ms/op). Por isso
reportam-se faixas e medianas, e os outliers são descartados explicitamente. Nada foi fixado:
nem frequência, nem núcleo (`perflock`/`taskset`). Um número isolado deste documento vale
como ordem de grandeza. A razão entre Germanio e baseline, medida no mesmo minuto, é mais
confiável que o valor absoluto.

### Como foi medido (reprodutível)

1. `rsync` do módulo em `e203c7a` para um diretório temporário, e um arquivo
   `zz_audit_test.go` no pacote `runtime` da cópia. Os benchmarks chamam `Carregar(...)` e
   `app.Handler.ServeHTTP` com `httptest`, e as allocs vêm de `ReportAllocs`.
2. **Baseline em Go direto**, num módulo separado: `net/http` + `database/sql` +
   `modernc.org/sqlite` com o **mesmo DSN**
   (`busy_timeout(5000)`, `journal_mode(WAL)`, `foreign_keys(ON)`, `_txlock=immediate`), o
   mesmo pool (25/5), o **mesmo esquema** (copiado do banco que o Germanio criou), a mesma
   página de 20 itens ordenada por `id DESC`, o mesmo `COUNT(*)`, a mesma validação mínima,
   a transação na escrita e `html/template` na página. A resposta JSON da lista tem o mesmo
   tamanho em bytes (3233 B).
3. App pequeno: `runtime/testdata/intencao/clientes.ge` (20 linhas: 1 entidade, pesquisa,
   filtro, 1 página). App grande: `examples/gitlab-foss` (687 linhas `.ge`, 18,7 KB, 14
   entidades). Visibilidade por registro: `runtime/testdata/intencao/loja.ge`
   (`usuario pode ver seus pedidos`).
4. Processo real: `germanio run` num diretório temporário e porta alta. O startup é medido
   até o primeiro 200, com polling de 1 ms. A RSS é lida de `/proc/<pid>/status`. Também
   foram usados `GODEBUG=inittrace=1`, `GODEBUG=gctrace=1`, `go test -cpuprofile/-memprofile`
   + `go tool pprof -top` e `go tool nm`/`go version -m`. Todos os servidores iniciados foram
   encerrados.

## 1. O caminho de execução

```text
germanio run app.ge
  main.go:14-18                 binário chamado "germanio" → cli legado
  cli/cli.go:37-46              "run" → runtime.Executar
  runtime/engine.go:446         Executar → Carregar
  runtime/engine.go:316         Carregar
    :320  parseFG               lexer.New(...).Tokenize() → parser.New(tokens).Parse()   (engine.go:26-48)
    :324  resolveImports        parseFG de cada import (pasta inteira, ordem alfabética)
    :329  parser.ResolveIntent  constrói ast.App (compiler/parser/resolver.go:115-120)
    :343  banco.Abrir           conexão, pool, auto-migração (CREATE TABLE/INDEX)
    :398  servidor.Novo         fila jobs (4 goroutines), WSHub, mapas
    :404  interp.New            interpreter.App = program.App
          capabilityHooks       git (sempre), processo…
    :425  interpreter.Run       scripts de topo + `ao iniciar`
    :436  srv.Handler()         monta o ServeMux UMA vez
  runtime/engine.go:458         WatchFiles (hot reload, sempre)
  runtime/engine.go:460-468     http.Server{...}.ListenAndServe()
```

- **Não há etapa entre `ast.App` e a execução.** O modelo resolvido (`ast.App`, entidades,
  regras, transições) é consultado diretamente pelos handlers. Nenhum plano de execução ou IR
  é derivado dele.
- `compiler/semantic` **não participa** do `germanio run`. Ele só serve o `ge check`/`ge rodar`
  (`tooling/gecli/cli.go:214`). Quando `semantic.Load` falha num app com `sistema`, o `ge`
  recai no CLI legado e o arquivo é analisado duas vezes, por dois parsers
  (`tooling/gecli/cli.go:226-235`).
- **O `.ge` não é reanalisado em runtime**, nem por request nem periodicamente. Uma mudança
  de arquivo re-executa o processo inteiro (seção 3.6). A única análise por request está em
  `/api/_eval` (`servidor.go:447-464`), que é restrito a admin ou `GERMANIO_DEV_EVAL=1`.

### 1.1 Camada de intenção: handlers genéricos sobre `ast.App`

- As rotas de cada entidade são registradas uma vez, com padrões Go 1.22, em
  `runtime/servidor/intencao.go:35-135`. Cada operação recebe uma closure
  (`intencao.go:76-92`). O roteamento em si não repete trabalho por request.
- Por request, `intentAPI.serve` (`intencao.go:422`) faz:
  - `identify` e `scopeAllows`;
  - `switch op` com strings;
  - `a.find` + `a.in.Can` para cada ancestral;
  - consultas `a.app.Entities[...]` (mapas por nome).
- **O CRUD declarado passa pelo interpretador.** `a.in.Op(ctx, entidade, "paginar", ...)`
  (`intencao.go:1105`) cria um `NewScope` e um `Call` e despacha `dbCall` por nome de método
  (`runtime/interpreter/intencao.go:19-31`). O resultado volta como `map[string]any`, e o
  `total` como `float64` (`intencao.go:1112`).
- `serializeFor` (`intencao.go:291-321`) recria o mapa de campos ocultos, iterando
  `e.Model.Fields`, **para cada linha**.
- **Páginas HTML chamam a própria API em processo.** `pageSite.call` (`paginas.go:85-107`)
  faz `json.Marshal` do corpo, `httptest.NewRequest`, `s.mux.ServeHTTP` e `json.Unmarshal` da
  resposta. A página percorre o mux, os middlewares, a serialização e a desserialização JSON a
  cada seção. O HTML sai de `html/template` pré-parseado em variáveis de pacote
  (`paginas.go:1250`). **Não há parse de template por request**, mas também não há cache de
  saída.
- A interface CRUD legada (`renderizador.go`) é gerada uma vez e guardada como string
  (`servidor.go:366-379`). Páginas declarativas legadas usam `strings.Builder` sem cache.
  `handlePagina` percorre `s.Program.Pages` linearmente a cada request (`servidor.go:387`).

### 1.2 Lógica (`quando`, `antes de`, funções, rotas): tree-walking

- **Sim, é tree-walking puro sobre o AST.** `ExecStatement` faz `switch stmt.Type`
  (`interpreter.go:159`) e `EvalExpr` faz `switch expr.Type` (`interpreter.go:598`), ambos
  com strings.
- **Variáveis:** `Scope{vars map[string]interface{}; parent *Scope}`
  (`interpreter.go:41-48`). A busca sobe pela cadeia de escopos.
- **Dispatch de chamada** (`runtime.go:295`), nesta ordem:
  1. `Functions[name]`;
  2. `globalFuncs()[name]`, que **reconstrói um mapa de closures a cada chamada**
     (`runtime.go:634-635`);
  3. `callBuiltin`, um `switch name` de cerca de 111 casos (`interpreter.go:798`).
- **Sem `reflect`** no interpretador. `reflect.DeepEqual` só aparece no engine de console
  `runtime/germanio/engine.go:335`.
- **Hooks rodam por request:** `RunHook` (`interpreter/intencao.go:51`) é chamado em create,
  update, delete e ver (`servidor/intencao.go:560`, `:589`, `:688`, `:708`, `:795`, `:834`,
  `:980`, `:988`). O hook `antes_ver` roda **por linha** nas listagens (seção 3.1).
- Erros viram `panic` e são recuperados como controle de fluxo (`Op`, `ExecRoute`
  `runtime.go:710-747`).

### 1.3 Dados: `map[string]any` do banco à resposta

- **Leitura:** `scanRowsRaw` (`banco/consulta.go:571`) monta um `[]any` e um
  `map[string]any` por linha. `tipar` (`consulta.go:241`) reconstrói o mapa de tipos a cada
  chamada e converte bool, datas e listas (listas com `json.Unmarshal`). `serializeFor` copia
  cada linha para outro mapa, e `json.Encoder` a serializa.
- **Escrita:** `readBody` faz `io.ReadAll` + `json.Unmarshal` para `map[string]any`
  (`intencao.go:253-283`). `banco.Criar`/`Atualizar` fazem outro `json.Unmarshal`
  (`banco.go:447`, `:503`).
- **SQL:** é montado com `fmt.Sprintf` a cada chamada (`banco.go:399`, `:404`;
  `consulta.go:202`, `:216`, `:224`), com os valores em placeholders. **Não há `Prepare` nem
  cache de statements em `runtime/`.** O driver prepara cada consulta de novo:
  `_sqlite3Prepare` aparece com 20% das amostras de CPU do POST.
- **Toda listagem paginada executa `COUNT(*)` e depois `SELECT *`**, sem projeção
  (`consulta.go:202`, `:216`).

## 2. Medições

### 2.1 Parse e resolve (arquivo → `ast.App`, sem banco)

| Programa | Tempo/op | Memória/op | Allocs/op |
| --- | --- | --- | --- |
| `clientes.ge` (20 linhas) | 263–401 µs | 264 KB | 2 658 |
| `gitlab-foss` (687 linhas, 14 entidades) | 8,4–10,0 ms | 6,3 MB | 60 271 |

**Ponto quente medido:** `foldWord` (`compiler/parser/intencao.go:35-36`) cria um
`strings.NewReplacer` novo a cada palavra. Isso responde por **83% da memória alocada e
cerca de 48% da CPU do parse** (`pprof -top -cum`: `strings.makeGenericReplacer` +
`trieNode.add`). São 264 KB para 20 linhas. Não afeta requests, mas afeta o `ge check`, o
LSP, o startup e o hot reload.

### 2.2 Startup e memória

`Carregar` completo, em processo, com banco novo a cada iteração: small **35 ms** (341 KB);
GitLab **267 ms** (7,6 MB, 77 755 allocs).

| Processo | 1º 200, banco novo | 1º 200, banco existente | RSS no 1º 200 | RSS após 10 s ocioso | Threads |
| --- | --- | --- | --- | --- | --- |
| `germanio run clientes.ge` | 35–46 ms (5×) | 9–12 ms (4×) | 26,5–26,8 MB | 26,9 MB | 9–10 |
| `germanio run gitlab-foss/app.ge` | 244–248 ms (3×) | 22–31 ms (2×) | 28,6–30,3 MB | 28,7–30,2 MB | 12–15 |
| baseline Go direto | 26 ms (3×) | 3,6 ms | 12,4 MB | 12,4 MB | 6 |

- O startup a frio é dominado pela criação das tabelas com fsync (GitLab: 14 entidades mais
  tabelas de membros e tarefas). A quente, o Germanio leva de 3 a 8x o tempo do baseline,
  somando parse, resolve e o init de pacotes.
- **RSS em repouso: 26,9 MB contra 12,4 MB (2,2x) num app de uma entidade.** O GitLab
  acrescenta só de 2 a 3 MB, ou seja, o custo fixo é o binário e o init, não o programa.
- **Init de pacotes** (`GODEBUG=inittrace=1`, mesmo em `germanio version`): 213 pacotes,
  7,2 ms, 2,44 MB e 12 101 allocs antes de `main`. Detalhe:
  - whatsmeow/libsignal/util: 1,22 MB e 2,7 ms;
  - goldmark: 0,56 MB e 1,8 ms;
  - pacotes do próprio Germanio: 0,49 MB e 1,1 ms.

  No baseline: 102 pacotes, 0,77 ms e 94 KB.
- Pico de RSS (VmHWM) depois da bateria de carga da seção 2.4: **60,2 MB contra 35,2 MB**.
- GC sob carga de GET (`gctrace`): 5 786 ciclos em cerca de 100 s. O heap vivo fica em 4–5 MB
  com meta de 9–11 MB, então há um ciclo a cada ~3–4 ms. O GC ficou com 2% da CPU acumulada.
  Com GOGC=100 e heap vivo pequeno, a frequência de GC acompanha diretamente as allocs/op.

### 2.3 Tamanho do binário

| Binário | Tamanho |
| --- | --- |
| `go build .` padrão (CGO ligado, dinâmico, com debug) | 43,0 MB |
| `CGO_ENABLED=0` | 40,2 MB |
| `CGO_ENABLED=0 -ldflags "-s -w"` | 28,0 MB |
| equivalente ao `germanio build` (só `runtime.Executar`, `-s -w`, sem CGO) | 27,4 MB |
| baseline Go direto (`-s -w`, sem CGO) | 12,9 MB |
| programa vazio só com `net/http` (`-s -w`) | 3,6 MB |
| `net/http` + `runtime/whatsapp` | 12,7 MB (**+9,1 MB**) |
| `net/http` + `runtime/banco` (3 drivers) | 7,5 MB |
| `net/http` + `runtime/servidor` (importa whatsapp) | 21,0 MB |

### 2.4 Requests: Germanio contra Go direto

**Em processo** (`httptest`, sem rede; medianas de 3 repetições com outliers descartados):

| Operação | Germanio | Baseline | Razão |
| --- | --- | --- | --- |
| GET lista (20/página), 1 000 linhas | 249 µs · 60,7 KB · 1 187 allocs | 158 µs · 35,2 KB · 858 allocs | 1,6x tempo · 1,4x allocs |
| GET lista, 10 000 linhas | 292 µs · 1 188 allocs | 162 µs · 858 allocs | constante (paginado) |
| GET lista em paralelo (12 P) | 76 µs/op · 1 192 allocs | 61 µs/op · 861 allocs | 1,25x |
| POST criar (fsync padrão, `synchronous=FULL`) | 4,67–4,88 ms · 19,6 KB · 286 allocs | 4,75 ms · 15,0 KB · 186 allocs | ≈1x (limitado por I/O) |
| POST criar com `synchronous=NORMAL` (só no experimento) | 299 µs | 294 µs | ≈1x |
| POST em paralelo com `synchronous=NORMAL` | 260 µs/op | 128 µs/op | 2,0x |
| Página HTML `/clientes` (20 itens) | 750 µs · 159 KB · 2 990 allocs · 6,6 KB | 303 µs · 61 KB · 1 873 allocs · 3,7 KB | 2,5x tempo · 1,6x allocs |
| Pesquisa `?q=` sobre 10 000 linhas | 20,2 ms | — | varredura `LIKE '%x%'` dupla (COUNT + SELECT) |

**HTTP real** (`germanio run` contra o baseline, 1 000 linhas, 5 s por ponto, cliente na
mesma máquina):

| Operação | c | Germanio req/s · p50 · p99 | Baseline req/s · p50 · p99 |
| --- | --- | --- | --- |
| GET lista | 1 | 3 123 · 298 µs · 0,70 ms | 4 145 · 232 µs · 0,40 ms |
| GET lista | 16 | 9 798 · 1,38 ms · 5,1 ms | 11 750 · 1,03 ms · 5,1 ms |
| GET lista | 64 | 13 790 · 4,0 ms · 13,9 ms | 16 239 · 3,4 ms · 12,4 ms |
| Página HTML | 1 | 1 556 · 594 µs · 1,1 ms | 2 579 · 372 µs · 0,64 ms |
| Página HTML | 16 | 5 053 · 2,9 ms · 7,4 ms | 8 436 · 1,6 ms · 5,6 ms |
| Página HTML | 64 | 6 387 · 6,9 ms · 39 ms | 9 236 · 5,8 ms · 24 ms |
| POST criar | 1 | 199–211 · 4,6 ms · 8,2 ms | 174–188 · 4,7–5,0 ms · 10,8 ms |
| POST criar | 16 | 175 · 4,7 ms · **2,84 s** · 3 erros | 155 · 5,6 ms · **2,05 s** · 0 erros |
| POST criar | 64 | 159 · 169 ms · **3,96 s** · 7 erros | 145 · 189 ms · **4,94 s** · 9 erros |
| GET c=16 durante POST c=16 | — | GET 9 851 req/s, p99 4,2 ms; POST 102 req/s, p99 1,9 s | — |

Leitura dos números:

1. **Leituras simples: o custo da abstração é de 1,2 a 1,6x em tempo e 1,4x em allocs.**
   Boa parte do tempo pertence ao driver e é comum aos dois lados. O parse de colunas
   `DATETIME` pelo modernc (`parseTime`, `time.newParseError` ao tentar formatos) ocupa
   **cerca de 40% das allocs e cerca de 18% da CPU** do GET, **no Germanio e no baseline**.
   É um custo de esquema e de driver, não de interpretação.
2. **Página HTML: 2,5x.** O motivo é estrutural: a ida e volta JSON pelo mux interno
   (`paginas.go:85-107`) mais o dobro de HTML.
3. **Escritas: o gargalo é o fsync, não o Germanio.** Com o DSN padrão os dois lados ficam
   em cerca de 200 escritas/s. Com `synchronous=NORMAL` (seguro contra corrupção no modo
   WAL; pode perder as últimas transações numa queda de energia) os dois caem para cerca de
   0,3 ms, ou seja, 16x. Com concorrência, o Germanio custa 2x porque segura a trava de
   escrita durante todo o handler (seção 3.2).
4. **Escritas concorrentes degradam sem backpressure:** p99 de segundos e 500 quando o
   `busy_timeout` de 5 s estoura. O mesmo acontece no baseline com o mesmo DSN; o
   comportamento vem da configuração herdada, não do código do baseline.

### 2.5 Crescimento com os dados: visibilidade por registro

`loja.ge`, uma pessoa com 20 pedidos numa tabela de N pedidos, `GET /_ge/api/pedidos`
(página de 20):

| N pedidos na tabela | Tempo/op | Memória/op | Allocs/op |
| --- | --- | --- | --- |
| 100 | 2,5 ms | 206 KB | 4 574 |
| 1 000 | 8,0 ms | 1,8 MB | 43 280 |
| 10 000 | 63 ms | 17,7 MB | 431 675 |
| 50 000 | **414 ms** | **88 MB** | **2 157 749** |

O custo é linear no tamanho da **tabela**, não da página: são 1 764 bytes e 43 allocs por
linha lida. **12 das 14 entidades do GitLab** passam por esse caminho (`RecordDependent`):
projeto, pipeline, job, grupo, membro, usuario, token_de_acesso, issue, label, milestone,
merge_request e webhook. A exceção é `comentario`, que herda a visibilidade do pai e usa o
caminho paginado.

### 2.6 Outras medições dirigidas

- **Hot reload em repouso** (seção 3.6): 1 tick de CPU em 10 s num diretório pequeno. Com
  **100 000 arquivos em `repositorios/`**, o custo sobe para **2,12 s de CPU a cada 10 s
  (~21% de um núcleo) sem nenhum request**. O baseline fica em 0.
- **Log de job regravado inteiro** (padrão de `trabalho_remoto.go:342`, reproduzido sobre o
  mesmo driver): 1 024 pedaços de 4 KiB (4 MiB) levam **45,7 s** regravando a coluna, contra
  **5,0 s** anexando linhas (9,2x). A escrita lógica total é cerca de 2 GiB, contra 4 MiB.
- **Polling da fila de tarefas** (`tarefas.go:96`, 4 vezes por segundo, `SELECT ... WHERE
  estado = 'pendente' ... ORDER BY id LIMIT 1`, sem índice, e a tabela nunca é limpa):

  | Tarefas concluídas na tabela | Custo de um poll | CPU ociosa |
  | --- | --- | --- |
  | 10 000 | 0,87 ms | 0,3% de um núcleo |
  | 100 000 | 13 ms | 5,3% de um núcleo |
  | 1 000 000 | 128 ms | **51% de um núcleo** |

  Todo app com `ast.App` liga essa fila (`intencao.go:47`), e cada entrega de webhook
  acrescenta uma linha.

## 3. Riscos estruturais

Os riscos estão ordenados por gravidade. "Medido" indica que há número nesta auditoria;
"lido" indica evidência só no código.

### 3.1 Visibilidade por registro lê a tabela inteira (medido)

- `servidor/intencao.go:1116-1142`: quando `RecordDependent(e)` é verdadeiro
  (`interpreter/intencao.go:346-358`: visibilidade, papel mínimo, dono ou `antes_ver`), a
  listagem lê **todas** as linhas em lotes de 500 (`:1122-1123`) e chama `Can` por linha.
  Faz isso só para montar uma página de 20 e o `X-Total`.
- `Can` (`interpreter/intencao.go:263-300`):
  - pode rodar o hook `antes_ver`;
  - carrega os pais com `interp.load` sem cache (`:188-197`), e cada carga é um
    `BuscarRegistro`, que chama `Filtrar` e custa `COUNT` + `SELECT` (`consulta.go:313-314`);
  - consulta a tabela de membros em `Level` (`:138`).
- Resultado: N+1 sobre toda a tabela. `narrowVisible` (`intencao.go:1155`) só estreita o
  conjunto para anônimos.
- Viola "Sem ingenuidade no banco" e "sem leitura ilimitada": `mostre issues` no GitLab lê
  todas as issues de todos os projetos.

### 3.2 Um único escritor com a trava presa durante o request (medido)

- `banco.go:159`: SQLite com `_txlock=immediate`, `busy_timeout(5000)` e o `synchronous` padrão
  do modo WAL (medido: `PRAGMA synchronous` = 2, FULL).
- Todo método não seguro passa por `transactional` (`intencao.go:80-84`,
  `transacao.go:88-125`). Com a transação `IMMEDIATE`, isso segura a **trava global de
  escrita** do começo ao fim do handler: hooks `.ge`, e até HTTP externo dentro de hooks,
  rodam com a trava. A resposta inteira fica retida em buffer.
- **Não há fila de escrita nem backpressure.** Os escritores excedentes giram no busy
  handler e falham com 500 depois de 5 s. Medido: p99 de 2,8 a 4,0 s e erros já com 16
  clientes.
- **Nenhum `context` chega ao banco.** O grep por `Context(` em `runtime/banco` retorna 0 e
  `EmTransacao` usa `DB.Begin()` (`consulta.go:521`). Um cliente que desconecta não cancela
  nada.
- O pool é `MaxOpen 25 / MaxIdle 5` para qualquer driver (`banco.go:47-50`), inclusive
  SQLite, que só admite um escritor.

### 3.3 Trabalho que cresce sem limite com o tempo (medido)

- **Logs de job:**
  - o log remoto é lido, concatenado e regravado a cada pedaço (`trabalho_remoto.go:325-343`,
    teto de 4 MiB em `:50`), o que é O(n²);
  - o executor local regrava o log inteiro a cada segundo (`execucao.go:519-529`);
  - a leitura devolve o log inteiro, sem Range (`execucao.go:700-703`).
- **Fila de tarefas:** o polling a cada 250 ms (`tarefas.go:96-107`) consulta uma tabela sem
  índice (`tarefas.go:49`) e sem limpeza. Só existem `UPDATE` para `concluida`/`morta`
  (`:138`, `:143`), e nenhum `DELETE`.
- **Pesquisa:** `LOWER(col) LIKE '%x%'` (`consulta.go:177-178`) com `COUNT(*)` antes
  (`consulta.go:202`) faz duas varreduras completas (20 ms com 10 000 linhas). Não há FTS.
  Os campos de `permita filtrar ... por cidade` **não ganham índice**: no esquema medido só
  existe o UNIQUE de `email`.

### 3.4 Concorrência sem limite e sem cancelamento (lido)

- **`paralelo`:** uma goroutine por item, sem limite (`interpreter.go:1139`). As goroutines
  compartilham o interpretador e escopos cujo `map` não tem mutex (`interpreter.go:41-48`),
  o que dá **risco de data race**. O pânico vira a string `"erro: ..."` e não cancela as
  irmãs.
- **`chamar_async`** (`interpreter.go:1266`) e **`consultar_paralelo`** (`:1221`): uma
  goroutine por item, também sem limite.
- **`timeout`** (`:1182`) abandona a goroutine, que continua rodando.
- **WebSocket:** a rota `/ws` está sempre registrada (`servidor.go:110`). Não há limite de
  conexões, e cada conexão e desconexão faz broadcast para todas (`servidor.go:190-195`),
  o que dá O(N²) numa rajada.
- **Limpeza do rate limiter:** `for { Sleep }` que nunca para (`servidor.go:198-217`). Cada
  chamada a `Handler()` cria mais uma goroutine desse tipo.
- **Fila `jobs`:** 4 workers e 256 posições, sempre criada (`servidor.go:75`). Quando cheia,
  descarta em silêncio (`jobs/jobs.go:56-66`).

### 3.5 Leituras e respostas sem teto (lido)

- **API legada `/api/<modelo>`:** `?limite=` vem da query sem máximo
  (`servidor.go:627-656`). O padrão é 100 (`banco.go:344-345`).
- **Export e relacionados:** `ListarTodos` (`banco.go:673-684`) e `BuscarRelacionados`
  (`banco.go:599`) fazem `SELECT *` sem LIMIT. O export carrega tudo em `[]map[string]any`
  antes de escrever.
- **`modelo.filtrar(...)` no `.ge`:** sem `limite`, lê a tabela inteira
  (`interpreter/runtime.go:436-438`, `:487`).
- **`SELECT *` em toda consulta**, sem projeção (`consulta.go:216`; `banco.go:404`, `:423`,
  `:599`, `:684`). Um campo de log de 4 MiB viaja em toda listagem de jobs.
- **Corpos sem limite:**
  - `/upload` usa `ParseMultipartForm(128<<20)` sem `MaxBytesReader` (`servidor.go:757`);
  - `io.ReadAll` sem limite em `/api/_presence`, `/api/whatsapp/*` e `/api/_proxy`
    (`servidor.go:939`, `:995`, `:1046`, `:1085`);
  - `auth.Login` decodifica o corpo sem limite (`auth/auth.go:126`);
  - `httpclient` lê a resposta externa inteira (`httpclient.go:49`).
- **Git:** o smart HTTP faz streaming (`git/http.go:83`, `:118-121`). Já `ReadFile` lê o
  blob inteiro antes de truncar (`git/git.go:489-517`), e `Diff`, `Log` e `Tree`
  bufferizam a saída inteira (`git.go:112`, `:532+`).

### 3.6 Hot reload ligado em produção (medido)

- `Executar` chama `WatchFiles` incondicionalmente (`engine.go:458`). O watcher faz
  `filepath.Walk` do diretório inteiro a cada segundo (`hotreload.go:16-40`), entrando em
  `repositorios/` e `uploads/`. Medido: ~21% de um núcleo com 100 000 arquivos.
- A mudança de qualquer `.ge` re-executa o processo com `os.Exit(0)` (`hotreload.go:43-54`),
  o que pula `defer app.Fechar()`.
- O binário de `germanio build` também chama `Executar` (`cli/cli.go:1142`), então também
  fica vigiando o diretório temporário extraído.

### 3.7 Mapas que só crescem e limites padrão (lido)

- **Mapas sem limite:**
  - `auth.loginAttempts`/`loginLockout` são indexados pelo login *digitado* e nunca expiram
    (`auth/auth.go:26-27`, `:170-173`). Um atacante cresce o mapa à vontade;
  - `presence` tem chave vinda do cliente e não tem TTL (`servidor.go:948-955`);
  - `regexCache` (`stdlib.go:293`);
  - `runMu` (`execucao.go:45`, `:302`);
  - `WSHub.clients`.
- **`http.Server`** (`engine.go:460-468`): `ReadTimeout 60s`, `WriteTimeout 10min`,
  `IdleTimeout 60s`, **sem `ReadHeaderTimeout`**. São limites globais, pensados para
  git/logs longos e aplicados a todas as rotas. Não há limite de requisições simultâneas.
- **Rate limit:** só para POST em `/api/` (`servidor.go:255-277`). **A API gerada `/_ge/api`
  não tem rate limit.**

### 3.8 Onde o GC pesa (medido)

Cada linha lida gera vários `map[string]any`:

1. o de `scanRowsRaw`;
2. o de tipos, reconstruído a cada chamada em `tipar`;
3. a cópia de `serializeFor`, com o seu mapa `hidden`.

Numa página HTML, somam-se o `json.Marshal` e o `json.Unmarshal` internos. Na visibilidade
por registro, a lista de 50 000 linhas aloca **88 MB por request**. A 10 req/s, isso é cerca
de 880 MB/s de alocação. Com heap vivo pequeno e GOGC=100, o custo do GC cresce com as
allocs, não com o heap.

## 4. "Pague apenas pelo que usar"

Situação em `clientes.ge`, que não declara nenhuma dessas capabilities:

| Capability | Está no binário? | Inicializada sem uso? | Evidência |
| --- | --- | --- | --- |
| WhatsApp (whatsmeow, libsignal, protobuf, `mattn/go-sqlite3` com CGO) | sim, +9,1 MB | init dos pacotes: 1,2 MB e 2,7 ms; as rotas `/api/whatsapp/*` existem sempre (`servidor.go:130-133`) | `runtime/whatsapp/whatsapp.go:11-12`; `engine.go:22` importa sempre; o cliente só conecta com `WhatsApp.Enabled` (`engine.go:372`) |
| Markdown (goldmark) | sim | init: 0,56 MB e 1,8 ms | `runtime/interpreter/markdown.go:6-7` |
| MySQL e PostgreSQL | sim, os dois drivers | só o `init` de registro | `banco/banco.go:14-15` |
| Git | sim (pacote pequeno, mas `os/exec git`) | **sim**: cria `repositorios/` e registra o módulo `git` em qualquer app (medido: o diretório aparece em `clientes.ge`) | `runtime/capacidade_git.go:17-37` (`NewStore` roda mesmo com `needs == false`) |
| WebSocket | sim | **sim**: `NewWSHub()` e a rota `/ws` aberta, sem autenticação | `servidor.go:73`, `:110` |
| Fila `jobs` | sim | **sim**: 4 goroutines | `servidor.go:75` |
| Fila de tarefas | sim | **sim** com `ast.App`: tabela, goroutine e polling do banco 4 vezes por segundo | `intencao.go:47`, `tarefas.go:49`, `:96` |
| Proxy, eval, presence | sim | rotas sempre registradas | `servidor.go:127-137` |
| E-mail, cron, auth, executor | sim | não (condicionais) | `engine.go:352`, `:384`, `:429`; `execucao.go:404` |
| IDE | sim (`cli` importa) | não | `cli/cli.go:14` |
| IA | não há runtime de IA | — | — |
| Hot reload | sim | **sim**, sempre | `engine.go:458` |

Custo agregado medido: binário 27,4 MB contra 12,9 MB, RSS em repouso 26,9 MB contra
12,4 MB, init 7,2 ms contra 0,8 ms e 9 a 10 threads contra 6.

**O que um `germanio build` embute** (`cli/cli.go:1041-1200`):

- os **fontes `.ge`** e o `.env` via `go:embed`, **extraídos para um diretório temporário
  e reanalisados a cada partida**. Não há AST nem modelo pré-resolvido;
- o pacote `runtime` inteiro: todos os drivers, whatsmeow, goldmark e git.

Consequências:

- É compilado com `CGO_ENABLED=0` (`:1187`). Assim `mattn/go-sqlite3` vira stub e o
  **WhatsApp não funciona no binário gerado nem na imagem Docker** (`Dockerfile:10`), mas
  o código continua lá.
- O binário resultante roda com hot reload.
- Neste ambiente o `germanio build` **não compilou**: o `go.mod` gerado não traz `go.sum`, e o
  `go build` pede `go mod tidy`. Por isso o tamanho foi medido pelo programa equivalente
  (`runtime.Executar` com `-s -w`).

## 5. Síntese

- **O custo da interpretação não é o problema dominante hoje.** Em leituras simples o
  Germanio custa de 1,2 a 1,6x o Go direto. Nas escritas os dois são limitados pelo mesmo
  fsync. Os maiores fatores medidos são **algorítmicos e de configuração**, e compilar para
  Go não corrigiria nenhum deles:
  - visibilidade O(tabela): 414 ms e 88 MB por request;
  - log O(n²): 9x;
  - fila sem limpeza: 51% de um núcleo em repouso;
  - `LIKE` sem índice;
  - fsync por escrita: 16x;
  - trava de escrita durante o handler inteiro;
  - hot reload em produção.
- Os custos fixos (+14 MB de RSS, +15 MB de binário, 7 ms de init) vêm de dependências
  sempre presentes, principalmente o WhatsApp, e violam "pague apenas pelo que usar".
- A recomendação de arquitetura que decorre destas medições está em
  [ARQUITETURA.md](ARQUITETURA.md).
