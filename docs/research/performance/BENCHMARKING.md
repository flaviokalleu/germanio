# Benchmarks confiáveis: metodologia para o Germanio

- **Data:** 2026-09-28
- **Status:** estudo dirigido concluído. Fontes oficiais consultadas por WebFetch; papers
  citados pelas páginas da ACM/arXiv/SPEC. A suíte `bench/` do repositório estava sendo
  construída em paralelo a este estudo; o que se diz dela abaixo foi lido em
  `bench/micro_test.go`, `bench/app_test.go` e `bench/baseline/baseline.go` nesta data.
- **Pergunta:** como medir performance de modo que um número publicado seja reproduzível e que
  um CI detecte uma regressão grande (por exemplo 5x) sem alarmes falsos?
- **Princípio do Germanio servido aqui:** a ordem é correto, seguro, **mensurável**, rápido.
  Nenhuma otimização entra sem uma medição antes e depois.

## Fontes consultadas

Go
- `testing.B.Loop`, `ReportAllocs`, `ReportMetric`: https://pkg.go.dev/testing#B.Loop
- benchstat: https://pkg.go.dev/golang.org/x/perf/cmd/benchstat
- Painel de performance do Go: https://perf.golang.org/dashboard/
- Sweet (golang.org/x/benchmarks): https://pkg.go.dev/golang.org/x/benchmarks/sweet
- Repositório x/benchmarks: https://go.googlesource.com/benchmarks (README devolveu 503 no
  fetch; conteúdo não verificado diretamente)
- perflock: https://github.com/aclements/perflock
- Guia do GC (heap, `alloc_space`): https://go.dev/doc/gc-guide
- `runtime/metrics`: https://pkg.go.dev/runtime/metrics
- Diagnósticos: https://go.dev/doc/diagnostics
- PGO (profiles representativos): https://go.dev/doc/pgo

Ruído e viés
- Mytkowicz, Diwan, Hauswirth, Sweeney, "Producing wrong data without doing anything obviously
  wrong!", ASPLOS 2009: https://dl.acm.org/doi/10.1145/1508244.1508275
- intel_pstate, `no_turbo`: https://docs.kernel.org/admin-guide/pm/intel_pstate.html
- Daly et al., change point detection no CI do MongoDB, ICPE 2020: https://arxiv.org/abs/2003.00584
- MongoDB, "Creating a Virtuous Cycle in Performance Testing": https://arxiv.org/pdf/2101.10231

Carga
- wrk2 (Gil Tene): https://github.com/giltene/wrk2
- Gil Tene, "How NOT to Measure Latency": https://www.infoq.com/presentations/latency-response-time/
- k6, modelos aberto e fechado: https://grafana.com/docs/k6/latest/using-k6/scenarios/concepts/open-vs-closed/
- hey: https://github.com/rakyll/hey
- vegeta: https://github.com/tsenart/vegeta

Comparações públicas e web
- TechEmpower, visão geral dos testes: https://github.com/TechEmpower/FrameworkBenchmarks/wiki/Project-Information-Framework-Tests-Overview
- web.dev, performance budgets: https://web.dev/articles/performance-budgets-101
- web.dev, Web Vitals: https://web.dev/articles/vitals

---

## 1. Microbenchmarks em Go: `testing.B` e `b.Loop`

`b.Loop` (Go 1.24) substitui o laço sobre `b.N`:

- reinicia o timer na primeira chamada e o para quando retorna falso, então preparação e limpeza
  não entram na medida;
- mantém vivos argumentos, resultados e variáveis atribuídas dentro do laço, "preventing the
  compiler from fully optimizing away the loop body";
- executa a função de benchmark uma vez por medição, enquanto benchmarks com `b.N` rodam a
  função (e sua preparação) várias vezes (https://pkg.go.dev/testing#B.Loop).

`ReportAllocs` liga as estatísticas de alocação só naquele benchmark; `ReportMetric` publica
métricas próprias (por exemplo `req/s`, `bytes-rss`) no mesmo formato que o benchstat entende
(https://pkg.go.dev/testing#B.Loop).

Regras decorrentes:
1. Todo benchmark novo do Germanio usa `for b.Loop()` e `b.ReportAllocs()` (a suíte `bench/`
   já faz isso).
2. `allocs/op` é o número mais estável que um benchmark Go produz: em código determinístico ele
   não depende de frequência de CPU. É o melhor candidato a portão de CI (seção 9).
3. Rodar com `-count` de pelo menos 10 (seção 2) e `-run '^$'` para não misturar testes.

## 2. benchstat e significância estatística

- "Each benchmark should be run at least 10 times", 20 é melhor
  (https://pkg.go.dev/golang.org/x/perf/cmd/benchstat).
- Resumo por mediana e intervalo de confiança de 95%; comparação A/B por teste U de
  Mann-Whitney (não paramétrico); alfa de 0,05, "which means it is *expected* to show a
  difference 5% of the time even if there is no difference"; `~` significa sem diferença
  significativa (mesma fonte).
- Rodar em máquina ociosa, não em bateria, e **intercalar** as execuções antes/depois para
  distribuir o ruído igualmente (mesma fonte).

Consequência direta para CI: com 50 benchmarks e alfa 0,05, espera-se cerca de 2 a 3 "diferenças"
por execução mesmo sem mudança alguma. Significância estatística sozinha não pode ser o portão;
é preciso também um **tamanho de efeito mínimo** (seção 9).

## 3. Ruído: CPU scaling, turbo, GC, layout

- **Turbo.** Não há garantia de que a CPU sustente estados turbo, porque a distribuição de
  energia e o envelope térmico mudam com o tempo; o driver `intel_pstate` expõe `no_turbo`
  para desligá-los (https://docs.kernel.org/admin-guide/pm/intel_pstate.html).
- **Máquinas compartilhadas.** O perflock é "a simple locking wrapper for running benchmarks
  on shared hosts" (https://github.com/aclements/perflock); a limitação de frequência que ele
  também faria não foi confirmada no fetch (não verificado). O Sweet recomenda hardware
  dedicado com perflock e desaconselha nuvem pela variabilidade
  (https://pkg.go.dev/golang.org/x/benchmarks/sweet).
- **GC.** O custo do GC depende do heap vivo e do GOGC (https://go.dev/doc/gc-guide); um
  benchmark que muda de tamanho de heap muda de custo de GC. Fixar GOGC/GOMEMLIMIT no ambiente
  do benchmark e registrá-los.
- **Viés de medida.** Mudanças inócuas no ambiente (por exemplo tamanho das variáveis de
  ambiente, ordem de link) produzem conclusões erradas; o viés é "significant and commonplace";
  os autores propõem randomizar a configuração e análise causal
  (https://dl.acm.org/doi/10.1145/1508244.1508275). Para o Germanio: diferenças abaixo de ~5%
  entre dois commits em tempo de CPU não devem ser tratadas como fato sem repetição em outra
  máquina.
- **Ferramentas interferem entre si.** Profile de memória distorce profile de CPU; "use tools
  in isolation" (https://go.dev/doc/diagnostics). Nunca medir tempo com profile ligado.

## 4. Três tipos de benchmark, três perguntas

| Tipo | Pergunta | Unidade | Onde roda |
|---|---|---|---|
| MICRO | quanto custa esta função? | ns/op, B/op, allocs/op | `go test -bench`, qualquer máquina |
| APLICAÇÃO | quanto custa uma requisição real do app, comparada a Go direto? | ns/req, allocs/req, razão Germanio/Go | `go test -bench` sem rede (handler direto) |
| STRESS | onde o sistema quebra e como degrada? | vazão sustentável, p99/p99.9 a taxa fixa, RSS máximo, erros | programa separado, rede real, máquina dedicada |

Microbenchmarks são maus preditores de produção e maus profiles para PGO
(https://go.dev/doc/pgo). O Sweet existe porque o projeto Go precisou de "a breadth of
real-world applications" além dos micro (https://pkg.go.dev/golang.org/x/benchmarks/sweet).
Stress tests não servem de portão de CI: são longos e ruidosos; servem para achar limites e
calibrar os padrões de `RUNTIMES.md` (tabela de limites).

## 5. Ferramentas de carga e coordinated omission

- **Coordinated omission.** Um gerador de carga em modelo fechado espera a resposta lenta antes
  de mandar a próxima requisição e assim "coordena" com o servidor para não medir durante os
  períodos ruins. O wrk2 mede a latência a partir do instante em que a requisição *deveria* ter
  saído pelo plano de taxa constante (`-R`); no exemplo do README, o p99 passa de 6,04 ms
  (medição ingênua) para 1,27 s (corrigida) (https://github.com/giltene/wrk2).
- **k6.** Distingue modelo fechado (a próxima iteração começa quando a anterior termina) e
  aberto (chegadas independentes); no fechado, "the target system's response time can
  influence the throughput of the test"; os executores `constant-arrival-rate` e
  `ramping-arrival-rate` implementam o modelo aberto
  (https://grafana.com/docs/k6/latest/using-k6/scenarios/concepts/open-vs-closed/).
- **vegeta.** Taxa constante, declara evitar coordinated omission, relatórios de percentis,
  histogramas e formato HDR (https://github.com/tsenart/vegeta). Escrito em Go, fácil de
  instalar e de usar como biblioteca (esta última afirmação é inferência pelo repositório; não
  verificada como API estável).
- **hey.** Substituto do ApacheBench com `-n`, `-c` e `-q` (limite de QPS **por worker**)
  (https://github.com/rakyll/hey). A documentação não diz se corrige coordinated omission;
  com `-c` fixo ele opera como modelo fechado (inferência). Útil para fumaça, não para
  latência de cauda.

Regra: latência de cauda só é publicada a partir de gerador de **taxa fixa** (wrk2, vegeta,
k6 com arrival-rate). Vazão máxima pode vir de modelo fechado, rotulada como tal.

## 6. Percentis de latência

Médias escondem outliers; máximo sozinho não diz a frequência; relate a distribuição, com
atenção a p99 e p99.9; o HdrHistogram existe para registrar isso com precisão
(https://www.infoq.com/presentations/latency-response-time/). Para o Germanio:

- sempre p50, p90, p99, p99.9 e máximo, **junto com a taxa oferecida** e a taxa de erros;
- um percentil sem a taxa de chegada não significa nada;
- nunca promediar percentis de execuções diferentes; juntar os histogramas.

## 7. Memória: RSS, heap, alocações

- `B/op` e `allocs/op` (via `ReportAllocs`) medem alocação, não memória residente
  (https://pkg.go.dev/testing#B.Loop).
- `/memory/classes/total:bytes` é toda a memória que o runtime mapeou como leitura e escrita,
  mas exclui cgo e syscalls; RSS é a visão do SO e inclui isso
  (https://pkg.go.dev/runtime/metrics). Como o Germanio usa SQLite em Go puro (sem cgo), a
  diferença deve ser pequena, mas não há medição que o prove (não verificado).
- `pprof -sample_index=alloc_space` aponta onde alocar menos rende mais (https://go.dev/doc/gc-guide).
- Para o requisito "5 GB de arquivo não exige 5 GB de RAM", a medida é **RSS máximo** do
  processo durante o upload/download (`/proc/<pid>/status` VmHWM, ou `getrusage` ru_maxrss;
  conhecimento geral de Linux, não citado de fonte nesta pesquisa), comparado com o tamanho do
  arquivo. É um teste de propriedade (RSS máximo < constante), não de velocidade.

## 8. Como o projeto Go detecta regressões

- O painel https://perf.golang.org/dashboard/ mostra cada benchmark **relativo a um commit
  baseline, que é a última versão estável** no momento do teste, com intervalo de confiança
  de 95% em cinza, e uma visão dedicada de regressões; nem todo commit é testado
  (https://perf.golang.org/dashboard/).
- O Sweet compara dois toolchains com uma suíte de aplicações reais (CockroachDB, etcd,
  esbuild, gopher-lua, tile38, bleve, markdown...), exige cerca de 16 GB de RAM, Linux/amd64,
  horas por execução, e produz saída para benchstat
  (https://pkg.go.dev/golang.org/x/benchmarks/sweet).
- Lição: o baseline é **fixo e versionado** (a última release), não "o commit anterior";
  comparar sempre com o anterior deixa passar regressões lentas, 2% por vez.

Fora do Go, o MongoDB trocou limiares fixos por **detecção de ponto de mudança** (E-Divisive
means) sobre o histórico, com triagem humana, e relata queda dramática de falsos positivos
(https://arxiv.org/abs/2003.00584; contexto em https://arxiv.org/pdf/2101.10231). É a técnica
para tendências; não é necessária para pegar uma regressão de 5x.

## 9. CI que pega 5x sem flakiness

Uma regressão de 5x está muito acima do ruído de qualquer máquina de CI; o risco não é deixar
de vê-la, é disparar alarmes falsos com limiares apertados. Desenho proposto:

1. **Portão determinístico primeiro:** `allocs/op` dos benchmarks MICRO e APLICAÇÃO. Falha se
   subir mais de 2x em relação ao baseline versionado (em código determinístico o número é
   exato; a folga cobre mudanças de toolchain). Nenhum ruído de CPU o afeta.
2. **Portão de tempo por razão, na mesma execução:** medir o Germanio e o baseline Go direto
   no **mesmo job, intercalados**, e comparar a **razão** Germanio/Go, não o tempo absoluto.
   A máquina de CI mais lenta ou mais rápida se cancela na razão. Falha se a razão piorar
   mais de 3x em relação à razão registrada no baseline (um 5x real ultrapassa com folga).
3. **Significância e efeito, juntos:** só falha se benchstat der p < 0,05 **e** o efeito
   exceder o limiar. Isso elimina os ~5% de falsos positivos por benchmark do alfa
   (https://pkg.go.dev/golang.org/x/perf/cmd/benchstat).
4. **`-count=10` intercalado** e, em caso de falha, **uma** reexecução confirmatória; falha
   só se as duas falharem. Não repetir indefinidamente até passar.
5. **Baseline versionado**: arquivo de resultados da última release commitado com o contexto
   (seção 10); atualizado só por decisão explícita, nunca automaticamente.
6. **STRESS fora do portão**: roda agendado em máquina dedicada; publica tendências; pode usar
   change point detection no futuro (https://arxiv.org/abs/2003.00584).
7. **Budget de frontend** (seção 11) como portão determinístico de tamanho de bundle.

## 10. Registro obrigatório de contexto

Todo resultado publicado (commit de baseline, `GERMANIO_EVOLUTION.md`, relatório) leva,
sem exceção:

- hardware: modelo da CPU, núcleos/threads, RAM, disco (SSD/NVMe), e se é VM/container;
- SO e kernel; governor de CPU e estado do turbo (`no_turbo`);
- versão do Go (`go version`) e `GOMAXPROCS`, `GOGC`, `GOMEMLIMIT`, `GOEXPERIMENT`;
- versão e commit do Germanio (`go version -m germanio` mostra o módulo e o estado dirty);
- configuração do app: banco (SQLite/Postgres/MySQL), modo do journal, tamanho do pool,
  variáveis relevantes (ex. `GERMANIO_BCRYPT_RAPIDO` usado em `bench/app_test.go`);
- dataset: número de linhas, tamanho, semente do gerador;
- comando exato (com `-count`, `-benchtime`, `-cpu`) e ferramenta de carga com versão, taxa,
  duração, conexões;
- data e duração da execução.

Motivo: sem isso o número não é reproduzível e, pelo viés de medida, pode estar simplesmente
errado (https://dl.acm.org/doi/10.1145/1508244.1508275). Recomenda-se que a própria suíte
imprima esse cabeçalho (o `go test -bench` já imprime `goos`, `goarch`, `pkg`, `cpu`; o resto
pode sair via linhas `chave: valor`, que o benchstat trata como configuração; não verificado
na documentação do formato).

## 11. TechEmpower e as críticas

- Sete tipos de teste: JSON, consulta única, múltiplas consultas, Fortunes (ORM, template,
  escape de XSS, UTF-8), atualizações, plaintext (com pipelining) e cache; exigência de código
  "production-grade"; implementações sem HTTP realista são marcadas "Stripped"
  (https://github.com/TechEmpower/FrameworkBenchmarks/wiki/Project-Information-Framework-Tests-Overview).
- Críticas (inferências a partir das próprias regras, não de uma fonte crítica verificada):
  plaintext com pipelining mede o parser HTTP e a rede, não aplicações; "production-grade" é
  julgado por mantenedores e muitas entradas são altamente especializadas; nenhum teste inclui
  autenticação, autorização, validação ou regras, que são justamente o que o Germanio gera.
- Uso correto para o Germanio: **Fortunes e múltiplas consultas** são os únicos formatos
  próximos de uma app Germanio e podem inspirar cenários da suíte APLICAÇÃO; não publicar
  posição em ranking.

## 12. Budgets de performance (frontend)

- "A performance budget is a set of limits imposed on metrics that affect site performance";
  três tipos: quantidade (tamanho de JS, imagens, fontes), marcos de tempo e regras
  (Lighthouse); ponto de partida citado: recursos do caminho crítico abaixo de 170 KB
  comprimidos para mobile (https://web.dev/articles/performance-budgets-101).
- Web Vitals "bons": LCP até 2,5 s, INP até 200 ms, CLS até 0,1, medidos no **percentil 75**
  por dispositivo (https://web.dev/articles/vitals).
- Para o Germanio: o renderizador gera HTML/CSS/JS; o tamanho do que ele emite por tela é
  determinístico e pode ser portão de CI (bytes comprimidos por página gerada). LCP/INP exigem
  navegador e ficam na suíte STRESS/manual.

---

## Recomendação: a suíte do Germanio

### MICRO (portão de CI; `go test -bench`)

Medir o pipeline e os mecanismos do runtime isoladamente, com `b.Loop` e `ReportAllocs`:
- lexer, parser, resolver e compilação de projeto, em tamanhos crescentes (a suíte `bench/` já
  tem um gerador sintético de N blocos; manter os tamanhos em progressão geométrica para
  expor complexidade não linear);
- interpretador: expressão, chamada de função, laço;
- mecanismos: validação de um registro, serialização JSON de uma lista, renderização de uma
  página, verificação de permissão (`acesso`), montagem de SQL.
- Portão: `allocs/op` (2x) e tempo (3x com significância), contra baseline versionado.

### APLICAÇÃO (portão de CI; handler direto, sem rede)

Apps de referência pequenos e realistas em `bench/testdata/` (hoje `clientes.ge`), cada um com
o **mesmo app escrito em Go direto** (`bench/baseline/`), atrás das mesmas operações: listar,
ler um, criar, página HTML, e acrescentar com o tempo: listar com relação (N+1), listar
página profunda (OFFSET vs keyset), criar com validação e permissão, upload pequeno.
- Métrica principal: **razão Germanio/Go** em ns/req e allocs/req, medida no mesmo job.
- O baseline Go deve ser idiomático e com o mesmo comportamento observável (mesmo banco, mesmo
  pool, mesmas validações, mesmo escape de HTML, mesmo hash de senha); caso contrário a razão
  compara coisas diferentes. Isto é o "What you do use, you couldn't hand code any better" de
  Stroustrup transformado em teste (https://www.stroustrup.com/ETAPS-corrected-draft.pdf).
- Meta a registrar (não medida): razão próxima de 1 onde o Germanio só orquestra (banco,
  HTTP), e razão explicada (com profile) onde ele interpreta.

### STRESS (programa separado; agendado; não é portão)

`bench/stress` como programa próprio, rede real, máquina dedicada, contexto da seção 10:
- vazão sustentável a taxa fixa crescente (vegeta ou wrk2), com p50/p99/p99.9 e erros por
  degrau, até o joelho da curva;
- muitas conexões ociosas (keep-alive, websockets) para medir memória por conexão;
- clientes lentos (slowloris) para validar `ReadHeaderTimeout` e taxas mínimas;
- upload/download de arquivo muito maior que a RAM disponível ao processo, medindo RSS máximo;
- pico acima da capacidade para verificar rejeição cedo (503) em vez de fila ilimitada;
- escrita concorrente em SQLite para validar o caminho de escritor único;
- cancelamento: cliente desconecta no meio e nenhuma goroutine sobra (contagem de goroutines
  antes e depois; o Go 1.26 tem perfil experimental de goroutines vazadas,
  https://go.dev/doc/go1.26).

### Baselines "Germanio vs Go direto"

1. Mesmo job, mesma máquina, execuções intercaladas, `-count=10`.
2. Reportar a razão com intervalo (benchstat sobre os dois conjuntos).
3. Registrar a razão da última release como baseline versionado; o CI compara a razão atual
   com ela (portão 3x) e o relatório mostra a tendência.
4. Cada vez que uma frase `.ge` ganhar um mecanismo novo (streaming, keyset, lote), o baseline
   Go ganha o equivalente escrito à mão, para que a razão continue medindo o custo da
   abstração e não a diferença de algoritmo.

## Para o Germanio

- **ADOTAR:** `b.Loop` + `ReportAllocs` em toda a suíte (https://pkg.go.dev/testing#B.Loop).
  Resolve: benchmarks otimizados pelo compilador ou contaminados pela preparação.
- **ADOTAR:** benchstat com `-count>=10` intercalado
  (https://pkg.go.dev/golang.org/x/perf/cmd/benchstat). Resolve: decisões tomadas por uma
  única execução.
- **ADOTAR:** geradores de taxa fixa para latência (https://github.com/giltene/wrk2,
  https://github.com/tsenart/vegeta). Resolve: p99 subestimado por coordinated omission.
- **ADOTAR:** portão de CI por `allocs/op` e por razão Germanio/Go no mesmo job, com
  significância e tamanho de efeito. Resolve: detectar 5x sem flakiness em máquina de CI
  compartilhada.
- **ADOTAR:** cabeçalho de contexto obrigatório em todo resultado. Resolve: números não
  reproduzíveis em `GERMANIO_EVOLUTION.md` e em comparações públicas.
- **ADAPTAR:** baseline versionado da última release, como no painel do Go
  (https://perf.golang.org/dashboard/). Resolve: regressões lentas acumuladas.
- **ADAPTAR:** budget de bytes por página gerada (https://web.dev/articles/performance-budgets-101).
  Resolve: crescimento silencioso do HTML/JS emitido pelo renderizador.
- **EVITAR:** publicar posição em ranking tipo TechEmpower; usar hey para latência de cauda;
  médias de latência; stress test como portão de CI.
- **INVESTIGAR:** change point detection sobre o histórico do STRESS
  (https://arxiv.org/abs/2003.00584). Resolve: tendências abaixo do limiar do portão.
- **INVESTIGAR:** randomização de configuração para medir o viés (ordem de link, ambiente)
  quando uma otimização alegar menos de 5% (https://dl.acm.org/doi/10.1145/1508244.1508275).
