# Performance em runtimes maduros: o que serve a um runtime de intenção

- **Data:** 2026-09-28
- **Status:** estudo dirigido concluído. Fontes oficiais consultadas por WebFetch; o código do
  Go foi conferido na árvore local do Go 1.27.1 (`$(go env GOROOT)/src`); o código do Germanio
  citado foi lido neste repositório na data acima.
- **Diretriz normativa que este estudo serve:** "Simples para o humano. Eficiente para a
  máquina." O usuário escreve intenção (`servidor / aceite muitas conexões`, `processe pedidos
  em paralelo`); o runtime resolve workers, limites, cancelamento, backpressure, timeouts,
  propagação de erro e limpeza. Ordem: correto, seguro, mensurável, rápido. "Pague apenas pelo
  que usar."
- **Pergunta:** que mecanismos de performance os runtimes maduros escondem atrás de uma
  interface simples, e quais deles o core do Germanio (`runtime/`) deve assumir como padrão?

## Fontes consultadas

Go
- Guia do GC (GOGC, GOMEMLIMIT, escape analysis): https://go.dev/doc/gc-guide
- Estruturas do scheduler (G, M, P): https://go.dev/src/runtime/HACKING
- Go 1.25 (GOMAXPROCS ciente de cgroup, Green Tea GC experimental, FlightRecorder): https://go.dev/doc/go1.25
- Go 1.26 (Green Tea GC padrão, `io.ReadAll` mais rápido, mais alocação em pilha): https://go.dev/doc/go1.26
- `sync.Pool`: https://pkg.go.dev/sync#Pool
- `sync.WaitGroup.Go` (Go 1.25): https://pkg.go.dev/sync#WaitGroup.Go
- `net/http.Server` (campos de timeout; texto conferido em `src/net/http/server.go` local): https://pkg.go.dev/net/http#Server
- `ParseMultipartForm` e `MultipartReader`: https://pkg.go.dev/net/http#Request.ParseMultipartForm
- `database/sql` (pool): https://pkg.go.dev/database/sql#DB.SetMaxOpenConns
- PGO: https://go.dev/doc/pgo
- Linker, flags: https://pkg.go.dev/cmd/link
- Dead code do linker e REFLECTMETHOD: https://go.dev/src/cmd/link/internal/ld/deadcode.go
- Proposta de forçar DCE com reflect: https://github.com/golang/go/issues/72888
- Pedido de diagnóstico de "por que o DCE não rodou": https://github.com/golang/go/issues/60221
- Caso real (kube-apiserver): https://github.com/kubernetes/kubernetes/issues/132216
- `plugin` e seus avisos: https://pkg.go.dev/plugin
- `context`: https://go.dev/blog/context
- `errgroup`: https://pkg.go.dev/golang.org/x/sync/errgroup
- `semaphore`: https://pkg.go.dev/golang.org/x/sync/semaphore
- `runtime/metrics`: https://pkg.go.dev/runtime/metrics
- Diagnósticos (pprof, trace): https://go.dev/doc/diagnostics

Rust
- Custo zero (blog oficial, 2015): https://blog.rust-lang.org/2015/05/11/traits/
- Custo zero no Embedded Book: https://doc.rust-lang.org/beta/embedded-book/static-guarantees/zero-cost-abstractions.html
- Tokio mpsc: https://docs.rs/tokio/latest/tokio/sync/mpsc/index.html
- Tutorial de canais do Tokio: https://tokio.rs/tokio/tutorial/channels
- tower: https://docs.rs/tower/latest/tower/ ; `Service`: https://docs.rs/tower/latest/tower/trait.Service.html ; `load_shed`: https://docs.rs/tower/latest/tower/load_shed/index.html

Zig
- Visão geral: https://ziglang.org/learn/overview/
- Referência da linguagem: https://ziglang.org/documentation/master/

C/C++
- Stroustrup, "Abstraction and the C++ machine model" (rascunho ETAPS): https://www.stroustrup.com/ETAPS-corrected-draft.pdf
- nginx, métodos de eventos: https://nginx.org/en/docs/events.html
- nginx em "The Architecture of Open Source Applications": https://aosabook.org/en/v2/nginx.html
- io_uring(7): https://man7.org/linux/man-pages/man7/io_uring.7.html

Java
- HotSpot (tiered compilation, escape analysis): https://docs.oracle.com/en/java/javase/21/vm/java-hotspot-virtual-machine-performance-enhancements.html
- Coletores disponíveis (Serial, Parallel, G1, ZGC): https://docs.oracle.com/en/java/javase/21/gctuning/available-collectors.html
- JEP 439, ZGC geracional: https://openjdk.org/jeps/439
- JEP 444, virtual threads: https://openjdk.org/jeps/444
- JEP 505, structured concurrency (quinta prévia): https://openjdk.org/jeps/505

.NET
- System.IO.Pipelines: https://learn.microsoft.com/en-us/dotnet/standard/io/pipelines
- Memory<T>/Span<T>, regras de posse: https://learn.microsoft.com/en-us/dotnet/standard/memory-and-spans/memory-t-usage-guidelines
- ArrayPool<T>: https://learn.microsoft.com/en-us/dotnet/api/system.buffers.arraypool-1
- Kestrel, limites: https://learn.microsoft.com/en-us/aspnet/core/fundamentals/servers/kestrel/options
- Native AOT: https://learn.microsoft.com/en-us/dotnet/core/deploying/native-aot/

Banco
- SQLite WAL: https://www.sqlite.org/wal.html
- SQLite row values (paginação por janela): https://www.sqlite.org/rowvalue.html
- SQLite índices automáticos: https://www.sqlite.org/optoverview.html
- DataLoader: https://github.com/graphql/dataloader

## Estado atual do Germanio (lido no código em 2026-09-28)

Estes fatos ancoram as recomendações; não são críticas genéricas.

- `runtime/servidor/servidor.go:88` e `runtime/engine.go:460` criam `http.Server` com
  `ReadTimeout`, `WriteTimeout`, `IdleTimeout` e `MaxHeaderBytes`, mas **sem
  `ReadHeaderTimeout`**. O engine usa `ReadTimeout: 60s` e `WriteTimeout: 10min` para todas as
  rotas, com o comentário de que git clone/push e logs de job fazem streaming longo.
- `runtime/servidor/servidor.go:757` faz `r.ParseMultipartForm(128 << 20)`; várias rotas fazem
  `io.ReadAll(r.Body)` (por exemplo `servidor.go:673`, `:716`, `:939`, `:995`, `:1046`,
  `:1085`); algumas usam `MaxBytesReader`/`LimitReader` (`servidor.go:431`, `rotas.go:149`,
  `intencao.go:272`), outras não aparentam limite no mesmo trecho (não auditei o middleware
  de cada rota).
- `runtime/httpclient/httpclient.go:49` lê a resposta externa inteira com `io.ReadAll`, sem
  limite de tamanho; o timeout do cliente é 30 s (`httpclient.go:20`).
- `runtime/banco/banco.go:404` e `runtime/banco/consulta.go:225` paginam com `LIMIT ... OFFSET`;
  há 5 ocorrências de `SELECT *` em `runtime/banco/`.
- `runtime/banco/banco.go:47` fixa o pool em `SetMaxOpenConns(25)`, `SetMaxIdleConns(5)`,
  `SetConnMaxLifetime(5min)` para todos os bancos; `banco.go:159` abre o SQLite com
  `journal_mode(WAL)`, `busy_timeout(5000)` e `_txlock=immediate`.
- `runtime/interpreter/interpreter.go:1128` (`paralelo`) dispara uma goroutine por tarefa, sem
  limite, sem `context`, e converte pânico em uma *string* `"erro: ..."` no resultado: o erro
  não se propaga nem cancela as irmãs.
- `runtime/servidor/paginas.go` importa `html/template`; `text/template` chama
  `reflect.Value.MethodByName` com nome dinâmico (`src/text/template/exec.go:704` na árvore
  local). Isso marca o binário como REFLECTMETHOD (ver seção Go/linker).
- Existe um diretório `bench/` (MICRO/APLICAÇÃO/STRESS, com `bench/baseline/baseline.go`)
  sendo construído em paralelo a este estudo; ele usa `b.Loop` e `ReportAllocs`.

---

## Go

### Scheduler

O runtime casa três recursos: G (goroutine), M (thread do SO) e P (o direito de executar código
Go, com estado do scheduler e do alocador); "There are exactly `GOMAXPROCS` Ps"
(https://go.dev/src/runtime/HACKING). Desde o Go 1.25, no Linux, o padrão de `GOMAXPROCS`
respeita o limite de CPU do cgroup e é atualizado periodicamente se o limite mudar; isso é
desligado se o usuário fixar `GOMAXPROCS` (https://go.dev/doc/go1.25).

Consequência para o Germanio: o runtime já recebe paralelismo correto em container sem nenhuma
configuração. Fixar `GOMAXPROCS` no Germanio seria pior que o padrão. O paralelismo da CPU não
é o problema; o problema é **concorrência ilimitada** (goroutines baratas criadas sem teto),
que o scheduler não resolve.

### GC, GOGC e GOMEMLIMIT

- GOGC define a troca CPU x memória: "Doubling GOGC will double heap memory overheads and
  roughly halve GC CPU cost" (https://go.dev/doc/gc-guide).
- GOMEMLIMIT (Go 1.19) limita a memória total do runtime; é um limite **suave**, e se for baixo
  demais para o heap vivo o GC entra em thrashing; o runtime limita o GC a cerca de 50% da CPU
  para mitigar. Recomendação oficial: use em containers com memória reservada, com 5 a 10% de
  folga; **não** use em ambientes não controlados (CLI, desktop) nem com outros processos
  disputando a mesma memória (https://go.dev/doc/gc-guide).
- Green Tea GC: experimental no 1.25, padrão no 1.26, "somewhere between a 10—40% reduction in
  garbage collection overhead" em programas que usam muito o GC (https://go.dev/doc/go1.25,
  https://go.dev/doc/go1.26). O Germanio o recebe de graça ao compilar com Go >= 1.26.
- O guia diz que a maior alavanca é alocar menos, não ajustar o GC: medir com
  `pprof -sample_index=alloc_space`, entender escapes com `-gcflags=-m=3`, preferir valores sem
  ponteiros e agrupar campos de ponteiro no início das structs (https://go.dev/doc/gc-guide).

Para o Germanio: GOMEMLIMIT é um bom exemplo de "intenção → mecanismo". Uma frase de
implantação (por exemplo `use até 512 MB de memória`) pode virar `debug.SetMemoryLimit` com a
folga recomendada; em `germanio run` local não se deve definir nada.

### Escape analysis

O compilador decide pilha vs heap; valores que escapam custam alocação e trabalho de GC
(https://go.dev/doc/gc-guide). O Go 1.26 aloca mais slices na pilha
(https://go.dev/doc/go1.26). Para um interpretador que representa valores como
`interface{}` (caso de `runtime/interpreter`), quase todo valor escapa: esse é o ponto quente
provável, a ser confirmado com profile antes de qualquer mudança.

### sync.Pool e os seus riscos

A documentação define o uso: "cache allocated but unused items for later reuse, relieving
pressure on the garbage collector"; não serve como free list de objeto de vida curta; "any item
stored in the Pool may be removed automatically at any time without notification"
(https://pkg.go.dev/sync#Pool). Riscos práticos (dedução, não citação): reter buffers enormes
após um pico (o pool devolve o buffer de 100 MB para quem pediu 1 KB), vazar dados entre
requisições se o objeto não for zerado, e uso após devolução (o mesmo tipo de erro que o .NET
documenta para `ArrayPool`/`PipeReader`, ver abaixo). Regra para o Germanio: `sync.Pool` só no
core, só para buffers de tamanho limitado, com teto de capacidade na devolução, e só depois de
um benchmark mostrar ganho.

### io.Reader/io.Writer e streaming

O modelo Reader/Writer é o mecanismo de streaming do Go: `io.Copy` move dados em blocos, sem
materializar o todo. `ParseMultipartForm(maxMemory)` guarda o que excede `maxMemory` em
arquivos temporários; `MultipartReader` é a alternativa de streaming; `ServeContent` trata
`Range` e cabeçalhos condicionais que `io.Copy` não trata
(https://pkg.go.dev/net/http#Request.ParseMultipartForm). O Go 1.26 tornou `io.ReadAll` cerca
de 2x mais rápido (https://go.dev/doc/go1.26), mas `ReadAll` continua exigindo memória
proporcional à entrada: é o oposto de streaming.

Para o Germanio: um arquivo de 5 GB não pode exigir 5 GB de RAM. Hoje o upload usa
`ParseMultipartForm(128 MB)` (o excedente vai para disco temporário, então a RAM está
limitada, mas o disco dobra) e várias rotas usam `io.ReadAll`. O mecanismo genérico é
"arquivo = stream": `MultipartReader` → `io.Copy` para o destino final, com limite de tamanho
declarado e hash calculado no caminho (`io.TeeReader`).

### net/http e os timeouts do http.Server

Texto dos campos (conferido em `src/net/http/server.go` local; https://pkg.go.dev/net/http#Server):

- `ReadTimeout`: tempo para ler a requisição inteira, incluindo o corpo; "most users will
  prefer to use ReadHeaderTimeout", porque `ReadTimeout` não deixa o handler decidir por
  requisição.
- `ReadHeaderTimeout`: tempo para ler os cabeçalhos; depois disso o deadline é reiniciado e o
  handler decide o que é lento demais para o corpo. Se zero, usa `ReadTimeout`.
- `WriteTimeout`: não permite decisão por requisição.
- `IdleTimeout`: se zero, usa `ReadTimeout`.
- Por requisição: `http.ResponseController` (`SetReadDeadline`, `SetWriteDeadline`),
  `http.TimeoutHandler` e `http.MaxBytesReader`.

Para o Germanio: o padrão correto é **servidor com timeouts curtos e globais** (cabeçalho,
ociosidade) e **deadlines por rota** derivados da intenção. Uma rota de download ou de log em
streaming ganha um deadline de escrita estendido via `ResponseController`; o resto continua
curto. O `WriteTimeout: 10min` global do engine é o sintoma de um mecanismo que falta: o
timeout por tipo de rota.

### database/sql e os pools

`DB` é "a pool of zero or more underlying connections", seguro para uso concorrente;
`SetMaxOpenConns` (padrão ilimitado), `SetMaxIdleConns` (padrão 2), `SetConnMaxLifetime`,
`SetConnMaxIdleTime`, e `Stats()` com contadores de espera (`WaitCount`, `WaitDuration`)
(https://pkg.go.dev/database/sql#DB.SetMaxOpenConns). O pool de conexões **é** o limite de
concorrência do banco; `WaitDuration` é o sinal de backpressure que o runtime pode expor em
`ge explain`/métricas. Para SQLite, 25 conexões de escrita não fazem sentido (ver seção Banco).

### PGO

Desde o Go 1.20; `default.pgo` no pacote main ativa PGO automaticamente; ganhos de 2 a 14% nos
benchmarks do Go 1.22; o profile precisa ser representativo de produção, microbenchmarks são
maus candidatos; build mais lento e binário um pouco maior (https://go.dev/doc/pgo).

Para o Germanio: o binário do Germanio é o mesmo para qualquer app, então um `default.pgo`
coletado de uma carga de aplicação representativa (a suíte APLICAÇÃO de
`BENCHMARKING.md`) beneficia todos os usuários sem que eles saibam. Em `germanio build` (app
standalone), um profile do próprio app seria possível, mas é INVESTIGAR.

### Linker, dead-code elimination e reflect.Method

O linker só inclui o que é alcançável. Porém, quando qualquer função alcançável chama
`reflect.Value.Method`/`MethodByName` (ou `reflect.Type.Method*`) com argumento não constante,
ela é marcada REFLECTMETHOD e o linker "give[s] up on static analysis, and mark[s] all exported
methods of all reachable types as reachable"
(https://go.dev/src/cmd/link/internal/ld/deadcode.go). Chamadas com nome constante são tratadas
à parte desde a CL 522436 (resumo do resultado de busca; não verificado na CL). Há proposta
aberta para forçar DCE mesmo assim (https://github.com/golang/go/issues/72888) e pedido para o
linker explicar por que o DCE não rodou (https://github.com/golang/go/issues/60221); o
kube-apiserver é um caso real (https://github.com/kubernetes/kubernetes/issues/132216).

`text/template` (e portanto `html/template`) chama `MethodByName` com o nome vindo do template
(`src/text/template/exec.go:704` local). Como `runtime/servidor/paginas.go` importa
`html/template`, **o binário do Germanio já está no modo de DCE relaxado**: todo método exportado
de todo tipo alcançável fica no binário. Isso afeta tamanho do binário (hoje cerca de 32 MB
para `./germanio`, medido com `ls -la`) e tempo de link; não afeta velocidade de execução de
forma direta. `-s -w` removem símbolos e DWARF, não código (https://pkg.go.dev/cmd/link).

### Modularidade: build tags e plugins

Build tags permitem compilar variantes do runtime sem o código não usado. O pacote `plugin` é
desaconselhado pela própria documentação: só Linux/FreeBSD/macOS, exige o mesmo toolchain, as
mesmas build tags e as mesmas dependências byte a byte, mal suportado pelo race detector, e
risco de carregar bibliotecas não confiáveis; a documentação sugere IPC como alternativa
(https://pkg.go.dev/plugin).

---

## Rust

- **Custo zero.** O blog oficial retoma Stroustrup ("What you don't use, you don't pay for") e
  mostra as duas formas: genéricos monomorfizados (despacho estático, a abstração some) e trait
  objects (despacho dinâmico com vtable, custo previsível) sob o mesmo conceito
  (https://blog.rust-lang.org/2015/05/11/traits/). O Embedded Book define custo zero como mover
  comportamento para tempo de compilação; tipos de estado sem dados somem no código gerado
  (https://doc.rust-lang.org/beta/embedded-book/static-guarantees/zero-cost-abstractions.html).
  Lição transferível: **a mesma intenção pode ter duas implementações**, e o compilador escolhe
  a estática quando o modelo semântico permite. O Germanio, que resolve o `ast.App` antes de
  executar, está na posição de fazer isso (por exemplo compilar regras de validação para
  closures em vez de interpretar a AST a cada requisição).
- **Tokio e backpressure.** Canais limitados "provide backpressure": o envio espera quando a
  fila enche; o ilimitado nunca espera (https://docs.rs/tokio/latest/tokio/sync/mpsc/index.html).
  O tutorial é direto: "Unbounded queues will eventually fill up all available memory and cause
  the system to fail in unpredictable ways" e "picking good bounds is a big part of writing
  reliable Tokio applications" (https://tokio.rs/tokio/tutorial/channels).
- **tower.** O contrato de `Service` separa "estou pronto?" (`poll_ready`) de "processe"
  (`call`); "Calling a Service which is at capacity ... should result in an error. The caller is
  responsible for ensuring that the service is ready"
  (https://docs.rs/tower/latest/tower/trait.Service.html). Os middlewares
  `ConcurrencyLimit`, `RateLimit`, `LoadShed`, `Timeout` e `Buffer` se compõem
  (https://docs.rs/tower/latest/tower/); `LoadShed` "sheds load when the inner service isn't
  ready" (https://docs.rs/tower/latest/tower/load_shed/index.html). Lição: backpressure é uma
  **propriedade do contrato**, não um detalhe de implementação; e rejeitar cedo (503 com
  `Retry-After`) é melhor que enfileirar sem teto.

## Zig

- "No hidden memory allocations": a biblioteca padrão recebe o alocador como parâmetro; "no
  hidden control flow" (https://ziglang.org/learn/overview/). A referência reforça que funções
  que alocam recebem `Allocator` (https://ziglang.org/documentation/master/; o conteúdo da
  seção "Choosing an Allocator" não veio no fetch, não verificado em detalhe).
- `comptime` executa código na compilação; inicializações globais são avaliadas em comptime
  por padrão (https://ziglang.org/learn/overview/).
- Transferência correta para o Germanio: **não** expor alocadores ao usuário (o público nunca
  programou). O que se transfere é a ideia de *orçamento explícito por escopo*: uma requisição
  tem um orçamento de memória e tempo que o runtime conhece e aplica, e o `ge explain` mostra
  qual é. O "comptime" do Germanio é a fase de resolução (`compiler/parser/resolver.go`):
  tudo que depende só do `.ge` deve ser calculado ali, não por requisição.

## C/C++

- **Princípio de zero overhead** (Stroustrup): "What you don't use, you don't pay for. And
  further: What you do use, you couldn't hand code any better"
  (https://www.stroustrup.com/ETAPS-corrected-draft.pdf, trecho extraído do PDF). A segunda
  metade é a meta do baseline "Germanio vs Go direto" em `BENCHMARKING.md`.
- **nginx.** Orientado a eventos, workers single-threaded com run-loop que atende milhares de
  conexões; não cria processo ou thread por conexão, por isso a memória é conservadora
  (https://aosabook.org/en/v2/nginx.html). Escolhe automaticamente o método mais eficiente da
  plataforma (epoll, kqueue...) (https://nginx.org/en/docs/events.html). Go já entrega o mesmo
  efeito pelo netpoller + goroutines; o Germanio não precisa de event loop próprio.
- **io_uring.** Filas de submissão e conclusão compartilhadas entre kernel e processo, várias
  operações por syscall, opção de polling sem syscall
  (https://man7.org/linux/man-pages/man7/io_uring.7.html). Não é exposto pelo runtime do Go;
  adotá-lo exigiria bibliotecas externas e código específico de Linux. Fora de escopo.

## Java

- **JIT.** Tiered compilation (C1 coleta profile, C2 otimiza) e escape analysis com
  substituição escalar e eliminação de locks (https://docs.oracle.com/en/java/javase/21/vm/java-hotspot-virtual-machine-performance-enhancements.html).
  O Go compila antes; o PGO é o análogo mais próximo (profile offline em vez de online).
- **GCs.** G1 é o padrão (concorrente, metas de pausa com throughput); ZGC tem pausas abaixo de
  1 ms independentes do tamanho do heap; a primeira recomendação oficial é deixar a VM escolher
  e ajustar só o tamanho do heap (https://docs.oracle.com/en/java/javase/21/gctuning/available-collectors.html,
  https://openjdk.org/jeps/439). Lição: o bom padrão vence a configuração; o Germanio não deve
  expor escolha de GC.
- **Virtual threads (JEP 444, JDK 21).** Objetivo: permitir o estilo thread-por-requisição com
  utilização quase ótima; "should never be pooled"; para limitar concorrência, use semáforos,
  não pools; pinning em `synchronized`/nativo degrada a escala (https://openjdk.org/jeps/444).
  Isto é o modelo de goroutines do Go chegando ao Java, e confirma que o Germanio deve limitar
  **recursos** (semáforo por recurso: banco, HTTP externo, CPU), não "threads".
- **Structured concurrency (JEP 505, prévia no JDK 25).** Trata tarefas relacionadas como uma
  unidade; objetivo de eliminar vazamento de threads e atraso de cancelamento; `Joiner` define a
  política (`allSuccessfulOrThrow`, `anySuccessfulResultOrThrow`, `awaitAll`)
  (https://openjdk.org/jeps/505). Ainda prévia: não tratar como consolidado. A tabela de
  políticas é, porém, exatamente o vocabulário que uma frase `.ge` precisa ("todas precisam dar
  certo", "a primeira que responder", "espere todas").

## .NET

- **Kestrel.** Limites com padrão explícito: corpo máximo de 30.000.000 bytes, taxa mínima de
  corpo de 240 bytes/s com 5 s de carência (derruba clientes lentos, com carência para TCP
  slow-start), timeout de cabeçalho de 30 s, keep-alive de 2 min, 100 streams HTTP/2 por
  conexão; conexões ilimitadas por padrão; limites ajustáveis por requisição
  (https://learn.microsoft.com/en-us/aspnet/core/fundamentals/servers/kestrel/options). É o
  melhor modelo encontrado de "limites padrão seguros e sobrescrevíveis por rota".
- **Span<T>/Memory<T>.** `Span` só vive na pilha (não atravessa `await`); `Memory` pode ir para
  o heap; regras de posse, consumo e "lease": quem recebe um `Memory` e retorna não pode usá-lo
  depois (https://learn.microsoft.com/en-us/dotnet/standard/memory-and-spans/memory-t-usage-guidelines).
  É a formalização do zero-copy seguro: fatia sem cópia, com dono único e prazo de uso.
- **System.IO.Pipelines.** O `PipeWriter` fornece buffers do pool (sem alocação explícita), o
  `PipeReader` devolve memória quando o consumidor avança; backpressure por dois limiares
  (`PauseWriterThreshold`, `ResumeWriterThreshold`, dois para evitar oscilação); cancelamento
  sem exceção. A própria documentação lista os modos de falha: usar o buffer depois de
  `AdvanceTo` é "use after free", não completar vaza memória, ausência de tamanho máximo de
  mensagem leva a OOM (https://learn.microsoft.com/en-us/dotnet/standard/io/pipelines). Lição:
  zero-copy tem custo de correção alto; só vale no core, com API que torna o erro impossível.
- **ArrayPool<T>.** Aluga e devolve arrays para aliviar o GC quando arrays são criados e
  destruídos com frequência (https://learn.microsoft.com/en-us/dotnet/api/system.buffers.arraypool-1).
  Mesmo papel do `sync.Pool`.
- **Trimming / Native AOT.** Startup mais rápido e menos memória; sem carga dinâmica, sem
  geração de código em runtime, exige trimming; a publicação analisa o projeto e emite avisos
  para cada limitação (https://learn.microsoft.com/en-us/dotnet/core/deploying/native-aot/).
  Lição: o análogo para o Germanio é "o que não pode ser removido pelo linker deve ser
  apontado por uma ferramenta", como os avisos de AOT.

---

## Temas transversais

### Zero/low-copy seguro

O padrão maduro é: o core pode fatiar sem copiar, mas só com dono único e prazo de uso
explícitos (.NET `Memory<T>`), ou por construção de API que não entrega o buffer ao código do
usuário (`PipeReader`). Em Go, `[]byte` compartilhado entre goroutines sem posse definida é
fonte de corrupção silenciosa. Para o Germanio, o código `.ge` nunca vê buffers; o core usa
streaming (`io.Reader`) em vez de cópias grandes e copia defensivamente ao entregar valores ao
interpretador. Zero-copy é otimização do core, nunca conceito da linguagem.

### Streaming de grandes dados

Critério: memória O(tamanho do bloco), não O(tamanho do arquivo). Mecanismos em Go:
`MultipartReader` + `io.Copy` para upload; `http.ServeContent`/`ServeFile` para download com
`Range`; `rows.Next()` com `Flush` periódico para exportações CSV/JSON grandes; `json.Encoder`
em vez de `json.Marshal` do resultado inteiro; `io.LimitReader` em respostas externas. Todos
os `io.ReadAll` listados em "Estado atual" são candidatos.

### Backpressure e filas limitadas

Consenso entre Tokio (https://tokio.rs/tokio/tutorial/channels), tower
(https://docs.rs/tower/latest/tower/trait.Service.html), Pipelines
(https://learn.microsoft.com/en-us/dotnet/standard/io/pipelines) e JEP 444
(https://openjdk.org/jeps/444): toda fila tem teto; quando cheia, o produtor espera (dentro do
processo) ou é rejeitado cedo (na borda HTTP). Em Go, o teto natural é canal com capacidade,
`errgroup.SetLimit` (https://pkg.go.dev/golang.org/x/sync/errgroup) ou
`semaphore.Weighted.Acquire(ctx, n)`, que respeita cancelamento
(https://pkg.go.dev/golang.org/x/sync/semaphore).

### Structured concurrency

Em Go, o equivalente idiomático é `context` + `errgroup`: "When a request is canceled or times
out, all the goroutines working on that request should exit quickly"
(https://go.dev/blog/context); `errgroup.WithContext` cancela o contexto no primeiro erro e
`Wait` devolve o primeiro erro (https://pkg.go.dev/golang.org/x/sync/errgroup);
`WaitGroup.Go` (Go 1.25) remove o par Add/Done (https://pkg.go.dev/sync#WaitGroup.Go). O
`paralelo` atual viola as três propriedades (sem teto, sem cancelamento, erro como string).
A política de junção do JEP 505 (todas, a primeira, espere todas) é o vocabulário da intenção.

### Limites de recursos padrão

O Kestrel mostra o padrão: cada limite tem valor padrão documentado e sobrescrita por
requisição. O Go deixa quase tudo sem limite por padrão (`SetMaxOpenConns` ilimitado,
timeouts zero), o que torna obrigatório que o Germanio defina os seus e os documente. Proposta
de tabela padrão do Germanio (valores a validar com a suíte STRESS, não medidos):

| Recurso | Padrão proposto | De onde vem a intenção |
|---|---|---|
| Cabeçalho HTTP | `ReadHeaderTimeout` 10 s, `MaxHeaderBytes` 1 MB | nenhum; sempre ligado |
| Ociosidade keep-alive | 60 s | nenhum |
| Corpo JSON | 1 MB | campo de texto longo pode elevar |
| Upload | tamanho declarado no campo `arquivo`; streaming | `arquivo até 5 GB` |
| Tempo de handler | 30 s; rotas de streaming com deadline próprio | `download`, `acompanhar ao vivo` |
| Concorrência por recurso externo | semáforo por integração | `integração ... no máximo N ao mesmo tempo` |
| Paralelismo interno | `GOMAXPROCS` para CPU; teto por recurso para I/O | `processe em paralelo` |
| Resposta de API externa | limite de bytes (ex. 10 MB) | adaptador pode elevar |
| Memória total | nada local; `SetMemoryLimit` só em implantação declarada | `use até 512 MB` |

### Banco

- **Paginação por keyset.** OFFSET "requires time proportional to the offset value": o SQLite
  calcula `LIMIT x+y` e descarta `y`; row values `(a,b) > (?,?)` usam o índice e saltam
  direto (https://www.sqlite.org/rowvalue.html). Postgres e MySQL têm o mesmo comportamento
  (não verificado nesta pesquisa). `banco.go:404` e `consulta.go:225` usam OFFSET.
- **Projeção.** `SELECT *` lê colunas que a tela não mostra; o modelo semântico sabe quais
  campos cada tela e cada API exibem, então pode projetar só esses (inferência, sem o usuário
  escrever nada).
- **Batching e N+1.** O DataLoader agrupa as cargas de um mesmo "tick" em uma chamada em lote e
  memoiza por requisição, e recomenda uma instância por requisição para não vazar dados entre
  usuários (https://github.com/graphql/dataloader). No Germanio, relações declaradas
  (`projetos tem issues`) permitem ao runtime gerar `WHERE fk IN (...)` em vez de uma consulta
  por linha, sem API de dataloader exposta.
- **Índices derivados.** O SQLite cria índices automáticos transitórios quando não há índice e
  a consulta compensa, e emite `SQLITE_WARNING_AUTOINDEX`; a documentação diz que os
  desenvolvedores "should use these warnings to identify the need for new persistent indexes"
  (https://www.sqlite.org/optoverview.html). Como o Germanio conhece filtros, ordenações e
  chaves estrangeiras declarados, pode criar os índices na migração e usar o aviso de
  autoindex como teste de regressão ("nenhuma consulta declarada gera autoindex").
- **Transações curtas e SQLite WAL com um escritor.** Em WAL, leitores e escritor não se
  bloqueiam, mas só há um escritor por vez; checkpoint automático a 1000 páginas; leitores
  longos causam "checkpoint starvation" e crescimento sem limite do WAL; WAL não funciona em
  sistema de arquivos de rede (https://www.sqlite.org/wal.html). Consequência: para SQLite, o
  runtime deve ter **um** caminho de escrita serializado (uma conexão de escrita, `BEGIN
  IMMEDIATE`, que o `_txlock=immediate` já faz) e um pool separado de leitura; 25 conexões
  abertas competindo pelo único lock de escrita só convertem espera em `SQLITE_BUSY` e
  `busy_timeout`. Transações nunca devem englobar chamada HTTP externa.

### Modularidade do runtime

"Pague apenas pelo que usar" tem três níveis em Go, do mais barato ao mais caro:
1. **Custo zero em runtime** para o que não é usado: inicialização preguiçosa (não subir fila
   de jobs, websockets, cron, se o `.ge` não declara). É o nível que mais importa para
   memória e startup, e não exige nada do linker.
2. **Link seletivo** em `germanio build`: o gerador do projeto temporário pode importar só os
   pacotes de capabilities usadas pelo `ast.App` (ou usar build tags). Só funciona se o DCE
   não estiver no modo relaxado (REFLECTMETHOD via `html/template`) ou se os pacotes nem forem
   importados.
3. **Plugins** (`plugin`): evitar, pelos avisos oficiais (https://pkg.go.dev/plugin).

---

## Para o Germanio

### ADOTAR

1. **`ReadHeaderTimeout` em todo `http.Server` e deadlines por rota via
   `http.ResponseController`.** Resolve: o servidor não tem timeout de cabeçalho e o engine
   aplica `WriteTimeout` de 10 min a todas as rotas para acomodar poucas rotas de streaming.
   Afeta `runtime/engine.go:460`, `runtime/servidor/servidor.go:88`, rotas de streaming.
   Fonte: https://pkg.go.dev/net/http#Server.
2. **Teto de corpo em toda entrada e toda resposta externa** (`http.MaxBytesReader` na borda,
   `io.LimitReader` no cliente). Resolve: `io.ReadAll(r.Body)` e `io.ReadAll(resp.Body)` sem
   limite visível no mesmo trecho (`runtime/servidor/servidor.go`, `runtime/httpclient/httpclient.go:49`).
   Fonte: https://pkg.go.dev/net/http#Server (MaxBytesReader).
3. **Concorrência estruturada no `paralelo`**: `errgroup.WithContext` + `SetLimit`, contexto
   da requisição, erro propagado como erro (não string), cancelamento das irmãs. Resolve:
   goroutines ilimitadas, sem cancelamento e com erro silencioso em
   `runtime/interpreter/interpreter.go:1128`. Fontes: https://pkg.go.dev/golang.org/x/sync/errgroup,
   https://go.dev/blog/context.
4. **Paginação por keyset** para listas ordenadas por chave. Resolve: OFFSET linear em
   `runtime/banco/banco.go:404` e `runtime/banco/consulta.go:225`. Fonte:
   https://www.sqlite.org/rowvalue.html.
5. **Pool de escrita único para SQLite** (uma conexão de escrita, leituras em pool separado).
   Resolve: `SetMaxOpenConns(25)` igual para todos os bancos em `runtime/banco/banco.go:47`,
   contra o modelo de um escritor do WAL. Fonte: https://www.sqlite.org/wal.html.
6. **Upload e download em streaming** (`MultipartReader` + `io.Copy`, `ServeContent`).
   Resolve: arquivo grande exigindo memória ou disco temporário proporcional
   (`runtime/servidor/servidor.go:757`). Fonte: https://pkg.go.dev/net/http#Request.ParseMultipartForm.
7. **Tabela de limites padrão documentada** (modelo Kestrel), exibida por `ge explain`.
   Resolve: limites hoje espalhados como constantes (64 KB, 128 MB, 25 conexões) sem origem
   explicável. Afeta `runtime/servidor/`, `tooling/explicar/`. Fonte:
   https://learn.microsoft.com/en-us/aspnet/core/fundamentals/servers/kestrel/options.

### ADAPTAR

1. **Backpressure como contrato (tower) traduzido para HTTP**: limite de concorrência por
   recurso e rejeição cedo com 503 + `Retry-After` quando o semáforo não é obtido dentro de um
   prazo curto. Resolve: sob pico, requisições hoje esperam no pool do banco sem teto de
   espera. Afeta `runtime/servidor/`. Fontes: https://docs.rs/tower/latest/tower/trait.Service.html,
   https://docs.rs/tower/latest/tower/load_shed/index.html.
2. **Políticas de junção do JEP 505 como vocabulário `.ge`** ("todas precisam dar certo", "a
   primeira que responder", "espere todas"), implementadas com `errgroup`/`context`. Resolve:
   `paralelo` só conhece "espere todas e misture erros". Afeta `runtime/interpreter/`,
   `docs/INTENCAO.md` (proposta, não normativa). Fonte: https://openjdk.org/jeps/505 (prévia).
3. **DataLoader implícito** a partir das relações declaradas: `IN (...)` em lote e cache por
   requisição, sem API exposta. Resolve: N+1 provável em telas com relações. Afeta
   `runtime/banco/`. Fonte: https://github.com/graphql/dataloader.
4. **Índices derivados das consultas declaradas + teste "sem autoindex"**. Resolve: índices
   dependem de o usuário saber de índices. Afeta `runtime/banco/` (migração). Fonte:
   https://www.sqlite.org/optoverview.html.
5. **Projeção derivada das telas/APIs** em vez de `SELECT *`. Resolve: 5 `SELECT *` em
   `runtime/banco/`. Fonte: inferência sobre o modelo semântico (sem fonte externa).
6. **GOMEMLIMIT só por intenção de implantação** (`use até N`), com 5 a 10% de folga, nunca
   em `germanio run`. Resolve: o usuário não sabe o que é GOGC. Afeta `runtime/engine.go`.
   Fonte: https://go.dev/doc/gc-guide.
7. **Inicialização preguiçosa por capability** ("pague só pelo que usar" no nível de
   runtime). Resolve: subsistemas (jobs, tarefas, hot reload) ligados mesmo sem uso declarado
   (não auditado caso a caso). Afeta `runtime/engine.go`. Fonte: princípio de Stroustrup
   (https://www.stroustrup.com/ETAPS-corrected-draft.pdf).
8. **Resolver = comptime**: pré-compilar regras e validações em closures na resolução, não
   interpretar a AST por requisição. Resolve: custo por requisição do interpretador baseado em
   `interface{}`. Afeta `compiler/parser/resolver.go`, `runtime/interpreter/`. Fontes:
   https://ziglang.org/learn/overview/, https://blog.rust-lang.org/2015/05/11/traits/. Só com
   benchmark MICRO que prove o ganho.

### EVITAR

1. **Expor goroutine, canal, mutex, pool, alocador ou GC no `.ge`.** Contraria a diretriz e o
   público. Zig e .NET mostram o custo cognitivo e de correção (use-after-free documentado em
   https://learn.microsoft.com/en-us/dotnet/standard/io/pipelines).
2. **`sync.Pool` especulativo.** Só com benchmark que prove ganho, teto de capacidade e zeragem
   (https://pkg.go.dev/sync#Pool).
3. **Filas ilimitadas** em jobs, eventos ou websockets (https://tokio.rs/tokio/tutorial/channels).
4. **Plugins Go** para modularidade (https://pkg.go.dev/plugin).
5. **io_uring/event loop próprio**: Go já resolve com netpoller; ganho não comprovado para apps
   CRUD (https://man7.org/linux/man-pages/man7/io_uring.7.html).
6. **Fixar `GOMAXPROCS`**: perde o ajuste automático a cgroup do Go 1.25 (https://go.dev/doc/go1.25).
7. **Pools de "workers" como limite de concorrência** quando o limite real é um recurso
   (conselho do JEP 444: semáforo, não pool; https://openjdk.org/jeps/444).

### INVESTIGAR

1. **Tirar o binário do modo REFLECTMETHOD.** `html/template` provavelmente mantém o DCE
   relaxado (`src/text/template/exec.go:704`). Medir com `go build -ldflags=-dumpdep` (não
   verificado como técnica oficial) ou comparando tamanhos com/sem `paginas.go`. Resolve:
   tamanho do binário e do `germanio build`. Fontes:
   https://go.dev/src/cmd/link/internal/ld/deadcode.go, https://github.com/golang/go/issues/72888.
2. **`default.pgo` do Germanio** coletado da suíte APLICAÇÃO. Resolve: 2 a 14% de CPU sem
   mudar código, se o profile for representativo (https://go.dev/doc/pgo).
3. **Link seletivo em `germanio build`** por capability usada no `ast.App`. Resolve: binário
   standalone com código de capabilities não usadas. Afeta `cli/cli.go`.
4. **Métricas de runtime em `ge explain --runtime` ou endpoint admin** (`runtime/metrics`,
   `DB.Stats().WaitDuration`, fila de jobs). Resolve: "mensurável" antes de "rápido".
   Fontes: https://pkg.go.dev/runtime/metrics, https://pkg.go.dev/database/sql#DB.SetMaxOpenConns.
5. **Checkpoint starvation do WAL** sob leitura contínua (telas ao vivo). Resolve: crescimento
   do arquivo `-wal` em apps com tempo real (https://www.sqlite.org/wal.html).
6. **Streaming de exportações** (CSV/JSON grandes) via `rows.Next` + `Flush`. Resolve:
   exportação que materializa a tabela. Não auditado no código.

### Não copiar

- Configuração de GC por coletor (Java): o bom padrão vence a escolha.
- `Span`/`Memory` como tipos de linguagem: útil no core .NET, sem sentido para quem nunca
  programou.
- Alocadores explícitos do Zig no nível do domínio.
- A API `poll_ready`/`call` do tower: copiar o contrato, não a forma.
