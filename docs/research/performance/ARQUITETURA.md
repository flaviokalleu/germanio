# Arquitetura de execução: compilar ou interpretar o modelo semântico

- **Data:** 2026-09-28
- **Status:** recomendação de pesquisa. **Não é norma e não autoriza implementação.** As
  mudanças de runtime seguem o Documentation Gate de `AGENTS.md`. Qualquer sintaxe proposta
  aqui é pendência de design, como exige [docs/INTENCAO.md › Eficiência](../../INTENCAO.md#eficiência-simples-para-o-humano-eficiente-para-a-máquina).
- **Base empírica:** [AUDITORIA.md](AUDITORIA.md), commit `e203c7a`. Os números citados vêm
  de lá, com o mesmo ambiente e as mesmas ressalvas de ruído.

## 1. A pergunta

Existem duas opções:

- **(A) Pipeline compilado:** Lexer → Parser → AST → Análise semântica → **IR Germanio** →
  Otimização → Target. O backend gera Go e compila nativo, o frontend gera JS mínimo e WASM
  entra só com benefício real.
- **(B)** Manter a **execução interpretada do modelo semântico** (`ast.App`) e otimizar os
  pontos quentes.

## 2. O que as medições dizem

| Fator medido | Magnitude | Compilar para Go resolveria? |
| --- | --- | --- |
| Visibilidade por registro lê a tabela inteira | 414 ms e 88 MB por request com 50 000 linhas; 12/14 entidades do GitLab | **Não.** É algoritmo: falta empurrar a regra para o SQL |
| fsync por escrita (`synchronous=FULL`) | 4,7 ms contra 0,3 ms (16x), igual no Go direto | **Não.** É configuração |
| Trava de escrita durante o handler inteiro | 2x em escrita concorrente; p99 de segundos; 500 após 5 s | **Não.** É desenho de transação e de fila |
| Log regravado inteiro | 9,2x (45,7 s contra 5,0 s para 4 MiB) | **Não** |
| Fila de tarefas sem índice e sem limpeza | 51% de um núcleo em repouso com 1 milhão de linhas | **Não** |
| Hot reload em produção | ~21% de um núcleo com 100 000 arquivos | **Não** |
| Dependências sempre presentes | +9,1 MB de binário (WhatsApp), +14 MB de RSS, 7 ms de init | Em parte: *tree shaking* por capability resolveria. Mas build tags ou registro por capability resolvem o mesmo **sem** codegen |
| Página HTML chamando a API por JSON interno | 2,5x o Go direto | **Não.** É desenho: chamar a camada de dados sem HTTP interno |
| Overhead da abstração em leitura simples | 1,2–1,6x em tempo, 1,4x em allocs (+330 allocs/op) | **Sim**, em parte. Mas especializar em tempo de carga recupera boa parte |
| Parse de `DATETIME` pelo driver | ~40% das allocs e ~18% da CPU do GET, **também no Go direto** | **Não.** É esquema e driver |
| `foldWord` no parser | 83% da memória do parse | **Não.** É um `NewReplacer` por palavra |

**Conclusão empírica.** Sobre o que foi medido, compilar para Go atacaria só a última linha
estrutural: o overhead de 1,2 a 1,6x nas leituras simples. Isso é **menos de 100 µs por
request** num caminho em que o driver e a rede já custam de 150 a 300 µs. Os fatores de
10 a 1000x são algorítmicos, de configuração ou de desenho, e continuariam iguais num
backend gerado. Os casos de lentidão medidos não pedem uma troca de arquitetura de
execução.

## 3. Recomendação

**Manter (B) agora, e fazê-lo evoluir para uma IR executada, sem gerar Go.** Em outras
palavras: interpretar um *plano* derivado de `ast.App`, e não o `ast.App` cru. Isso também
é o primeiro passo de (A), caso um dia (A) se justifique.

### 3.1 Por que não codegen para Go agora

1. **Pouco ganho medido.** Ver a seção 2.
2. **Duas semânticas.** Um backend gerado e o runtime interpretado precisariam concordar em
   cada regra:
   - visibilidade, herança de papéis e invariantes;
   - "registros invisíveis respondem não encontrado";
   - transações e hooks.

   Essa é a parte mais delicada do Germanio. A norma exige que a semântica seja determinística
   e documentada, e duas implementações dobram a superfície em que ela pode divergir.
3. **Custo operacional.** Gerar Go exige toolchain Go na máquina do usuário. O
   `germanio build` atual já falha por falta de `go.sum` (AUDITORIA, seção 4). Acrescenta
   também tempo de compilação ao ciclo editar-rodar e dificulta o hot reload.
4. **`ast.App` já é um modelo semântico resolvido.** O resolver já fez a análise difícil:
   relações, FKs, estados, membros, índices. O que falta não é compilar, é **pré-computar o
   que hoje é recomputado a cada request** e **exprimir as regras numa forma que o banco
   execute**.

### 3.2 O que seria a "IR Germanio" nesta fase

A IR é um conjunto de **planos imutáveis**, construídos uma vez em `Carregar` a partir de
`ast.App` e executados pelo runtime.

- **Plano de entidade:**
  - a lista de colunas projetadas (fim do `SELECT *`), os campos ocultos e privados e os
    decodificadores por coluna, com a posição fixa em vez de `map[string]any` por linha;
  - os statements preparados (listar, contar, buscar por id, criar, atualizar);
  - os índices derivados, inclusive dos filtros declarados.
- **Plano de visibilidade:** a regra de `ver` compilada para um **predicado SQL** (`dono =
  :atual`, `EXISTS (SELECT 1 FROM membros ...)`, `visibility IN (...)`). A listagem passa
  a pedir ao banco só o que a pessoa vê, com `LIMIT` e `COUNT` sobre o mesmo predicado.
  O `Can` em Go continua como **oráculo** para os casos que o predicado não cobre, por
  exemplo `antes_ver` com lógica arbitrária. Nesses casos o limite de leitura precisa ser
  explícito e diagnosticado por `ge explain`.
- **Plano de página:** as seções de uma página chamam os planos de entidade direto, sem
  `httptest` nem JSON interno, reaproveitando a mesma autorização.
- **Lógica de nível 3:** compilação do AST para **closures**. É a técnica clássica que tira o
  `switch` de strings, a busca de variáveis por nome e o `globalFuncs()` reconstruído. Só
  depois disso, e se medido, compensaria um bytecode.

Essa IR é **a mesma semântica, pré-avaliada**. O interpretador atual continua como
referência durante a migração, e os dois caminhos são comparados pelos mesmos testes
(seção 5).

### 3.3 Quando reconsiderar (A)

Reabra a decisão se as três condições valerem **depois** da IR executada e das correções
algorítmicas:

1. em carga representativa (fluxos do GitLab e2e, não CRUD sintético), o overhead de CPU do
   Germanio sobre o baseline Go direto passar de **2x** nos endpoints limitados por CPU;
2. esse overhead **dominar** o tempo total do request, ou seja, maior que o banco somado
   à rede;
3. um protótipo gerado para **uma** entidade mostrar ganho de pelo menos 2x sobre a IR
   executada, com os testes de equivalência passando.

**Frontend (JS mínimo):** a decisão é independente do backend. Hoje as páginas são
renderizadas no servidor. Meça primeiro os bytes de JS e CSS enviados por página e o custo do
SSR (a página já custa 2,5x o Go direto). A norma exige medir "o JavaScript enviado".

**WASM:** só com um caso concreto que exija executar a *mesma* regra no navegador, como a
validação offline idêntica à do servidor. Mesmo nesse caso, é preferível gerar a validação
declarativa para JS e compará-la com o runtime por testes diferenciais.

## 4. O que medir antes de decidir

Cada item abaixo é uma medição, não uma opinião, e deve registrar o contexto que a norma
exige.

1. **Decomposição do tempo por request** em fluxos reais do GitLab (listar issues de um
   projeto com membros, abrir MR, push, log de job): a fração em driver/SQLite, em Germanio
   (planejamento, autorização, serialização), em `net/http` e em GC. Usar `pprof` com
   *labels* por operação.
2. **Consultas por operação:** a contagem de statements por request, com meta O(1) em
   relação ao tamanho da tabela, para cada entidade e para cada ação.
3. **Escala:** tempo e memória por request com 10³, 10⁴, 10⁵ e 10⁶ linhas, mantendo a página
   fixa. A curva precisa ser plana. Hoje é linear na visibilidade (AUDITORIA, seção 2.5).
4. **Escrita concorrente:** throughput e p99 com 1, 16 e 64 escritores. Comparar
   `synchronous` FULL com NORMAL e trava por request com fila de escrita.
5. **Custo fixo:** RSS em repouso, init e binário por capability declarada, contra o
   baseline.
6. **Lógica de nível 3:** um benchmark de hooks (`antes de criar` com validações e
   laços), comparando tree-walking, closures e o equivalente em Go. Sem isso não se sabe se
   o interpretador importa.
7. **Frontend:** bytes enviados, tempo até interativo e custo do SSR por página.

## 5. Riscos e como contê-los

| Risco | Contenção |
| --- | --- |
| **Duas semânticas divergindo** (o `Can` em Go e o predicado SQL; o interpretador e as closures; um dia o interpretado e o gerado) | Um único dono da semântica: o plano é *derivado* da regra resolvida, nunca escrito à mão por entidade. **Testes diferenciais** rodam os testes de `runtime/` e os e2e do GitLab nos dois caminhos e comparam status, corpo e efeitos. Fuzz de regras de visibilidade gera dados e pessoas aleatórios e compara o conjunto visível pelos dois caminhos. O interpretador continua como oráculo até o caminho novo passar. |
| Predicado SQL diferente entre SQLite, PostgreSQL e MySQL | O plano gera SQL por dialeto, e os testes diferenciais rodam nos três no CI |
| Regra não traduzível (`antes_ver` arbitrário) | Recaída explícita para o filtro em Go, **com teto de linhas lidas** e diagnóstico em `ge explain` ("esta lista lê até N registros porque `antes de ver` usa lógica") |
| Otimização que muda o comportamento observável (ordem, `X-Total`, 404 contra 403) | Os testes normativos existentes (`intencao_test.go`, e2e) passam a ser a porta de entrada. Toda mudança de plano vem com a medição antes e depois (BENCHMARKING.md) |
| Durabilidade trocada por velocidade (`synchronous=NORMAL`) | É decisão de produto, não otimização silenciosa. Se entrar, fica documentada e visível em `ge explain` ("uma queda de energia pode perder os últimos segundos de escrita"), e o autor pode pedir durabilidade estrita |

## 6. Caminho incremental

A ordem segue a norma (correto → seguro → mensurável → rápido) e o ganho medido:

1. **Mensurável:** a suíte `bench/` reproduz os casos da AUDITORIA (visibilidade por
   escala, escrita concorrente, log, fila, hot reload) e passa a servir de catraca.
2. **Correções algorítmicas e de configuração, sem mudar a semântica:**
   - hot reload só em `ge run` de desenvolvimento, nunca em `germanio build` nem em
     produção;
   - `foldWord` com um `Replacer` único;
   - índice e limpeza na fila de tarefas;
   - log de job em pedaços (append), servido por Range;
   - teto em `?limite`, `ListarTodos` e `filtrar`;
   - `ReadHeaderTimeout`, `MaxBytesReader` e limites de conexões de WebSocket;
   - expiração dos mapas de login e de presence.
3. **Visibilidade em SQL** (o plano de visibilidade), com testes diferenciais contra `Can`.
   É o maior ganho medido.
4. **Transação curta e fila de escrita:** a trava só enquanto se grava, não durante hooks
   com HTTP. Uma conexão de escrita e várias de leitura no SQLite. `context` do request
   chegando ao banco.
5. **Plano de entidade:** projeção, statements preparados, linhas com posição fixa e o fim do
   JSON interno nas páginas.
6. **Pague pelo que usar:** capabilities registradas só quando declaradas (WebSocket, jobs,
   git, tarefas) e o WhatsApp atrás de build tag ou fora do binário padrão.
7. **Lógica em closures**, se a medição do item 6 da seção 4 mostrar que importa.
8. **Reavaliar (A)** pelos critérios da seção 3.3.

## 7. Intenção de concorrência para o autor

**Pendência de design, não sintaxe aceita** (INTENCAO › Eficiência). O objetivo é que o
autor declare *o que* pode acontecer ao mesmo tempo e *até quando*, e que o runtime decida
workers, filas, cancelamento e limpeza. Goroutine, channel e mutex nunca aparecem.

### 7.1 Formas propostas

```text
pedidos são processados em paralelo
aceite muitas conexões
importações terminam em até 10 minutos
envie os e-mails em segundo plano
```

Frases de configuração (nível 2), opcionais:

```text
pedidos são processados em paralelo, até 8 ao mesmo tempo
se houver muitos pedidos esperando, recuse novos pedidos
```

No nível 3, a iteração limitada:

```text
para cada pedido em pedidos, em paralelo
    ...
```

### 7.2 Semântica que o runtime resolveria (proposta de limites padrão)

| Intenção | O runtime faz | Padrão proposto (a validar por medição) |
| --- | --- | --- |
| `processados em paralelo` | um pool de workers por declaração, com fila **limitada** e persistente (a tabela de tarefas, indexada e limpa) | workers = núcleos disponíveis para trabalho de CPU, até 16 para trabalho de I/O; fila de 64 × workers |
| fila cheia | **backpressure previsível**: a origem HTTP recebe "tente mais tarde" (503 com `Retry-After`); uma origem interna espera com prazo | nunca descartar em silêncio (hoje `jobs.Submit` descarta, `jobs.go:56-66`) |
| `terminam em até X` | prazo por tarefa, com `context` propagado a banco, HTTP e processos | request: o prazo do servidor; tarefa: 1 h (hoje é o do executor) |
| cancelamento | o cliente desconectou, desligamento ou prazo → cancela a tarefa e as irmãs de um `em paralelo`; limpeza garantida | desligamento gracioso: espera N s e depois cancela |
| erro em `para cada ..., em paralelo` | o primeiro erro cancela as demais e o erro chega ao chamador, com a mensagem da regra | resultados na ordem da entrada |
| estado compartilhado | `ge check` rejeita atribuir a variáveis de fora do bloco paralelo; cada item tem escopo próprio (hoje `paralelo` compartilha um `map` sem mutex, `interpreter.go:41-48`, `:1139`) | — |
| `aceite muitas conexões` | limite de requests simultâneos com fila curta e 503 além dela; `ReadHeaderTimeout`; `IdleTimeout`; teto de WebSockets; memória por conexão; pool do banco dimensionado por driver (SQLite: 1 escritor e N leitores); `GOMEMLIMIT` a partir da memória disponível | valores iniciais tirados do STRESS de `bench/`, não inventados |
| `em segundo plano` | uma tarefa persistente com novas tentativas e *backoff* para efeitos idempotentes (e-mail, webhook) | 5 tentativas; limpeza das concluídas após N dias |

`ge explain` mostra, para cada frase, os limites efetivos e de onde vieram (padrão, frase do
autor, variável de ambiente), como a norma exige ("`ge explain` mostra os limites em vigor").

## 8. Comandos futuros: o problema de cada um

A proposta é pelo problema a resolver, não pelo nome. Um comando só se justifica se o
problema não couber num comando existente.

### `ge benchmark`: "minha mudança deixou o *meu* app mais pesado?"

- **Quem precisa:** o autor de um app, ou o mantenedor de um exemplo como o GitLab, depois
  de mudar o `.ge` ou de atualizar o Germanio. A suíte `bench/` mede o *runtime*. Ela não
  mede o app de cada pessoa.
- **O que resolveria:**
  - gerar, a partir das declarações, as operações representativas do app: listar, ver, criar
    e editar cada entidade, cada página e cada ação;
  - rodá-las sobre um banco de teste descartável com volume escolhido (`com 10 000 issues`);
  - registrar o contexto obrigatório;
  - comparar com o resultado anterior e apontar regressões de ordem de grandeza (a norma
    fala em "regressão consciente").
- **Fora do escopo:** gerar carga contra servidores de produção ou hosts externos. Por
  segurança, roda só contra uma instância local efêmera.
- **Critério de pronto:** detectar, sem falso positivo, uma regressão introduzida de
  propósito de 5x (por exemplo, remover o plano de visibilidade).

### `ge profile`: "por que esta operação está lenta, em termos do meu `.ge`?"

- **Quem precisa:** alguém que não sabe ler `pprof`. Hoje o perfil diz
  `scanRowsRaw`/`tipar`/`Can`. A pessoa precisa ouvir algo como "listar issues: leu 12 000
  registros para mostrar 20 porque `issue confidencial pode ser vista por ...`
  (backend/issues/issues.ge:14)".
- **O que resolveria:** atribuir CPU, allocs, consultas e linhas lidas a **declarações** do
  `.ge` (arquivo:linha), usando *labels* de profiling por operação e plano. Mostraria as
  consultas por request e as N+1, com a sugestão da capability ou declaração que resolve.
- **Pré-requisito:** o runtime precisa carregar a origem de cada plano, e os planos da seção
  3.2 facilitam isso. Sem essa atribuição, o comando seria um `pprof` renomeado e não
  resolveria o problema.

### `ge inspect`: "o que o runtime decidiu e o que está acontecendo *agora*?"

- **Distinção com `ge explain`.** O `ge explain` já responde "de onde vem cada fato" e, pela
  norma, deve mostrar os limites em vigor. Essa é a parte **estática**: capabilities
  ligadas, índices derivados, listas que leem por registro, timeouts, tamanhos de página e de
  upload, pool. **Ela deve ficar no `ge explain`, não num comando novo.**
- **O problema que sobra para `ge inspect`:** observar uma instância **em execução**:
  - conexões abertas e WebSockets;
  - requests em andamento e os mais lentos;
  - profundidade das filas e tarefas mortas;
  - uso do pool do banco e espera pela trava de escrita;
  - memória, GC e goroutines;
  - tudo isso expresso nos nomes do app (fila de "entregas", lista de "issues").
- **Requisito de segurança:** endpoint local ou autenticado como administrador. Nada de
  expor internos por padrão.
- **Critério de pronto:** um operador consegue diagnosticar a saturação de escrita da
  AUDITORIA, seção 2.4 ("escritores esperando a trava: 63; p99 3,9 s"), sem ler código.

## 9. Resumo da decisão

- **Não gerar Go agora.** Os ganhos medidos que importam (10 a 1000x) são algorítmicos, de
  configuração e de desenho. O overhead de interpretação medido é de 1,2 a 1,6x em leituras
  simples.
- **Evoluir para uma IR executada:** planos pré-computados a partir de `ast.App`, com a
  visibilidade em SQL, a projeção, os statements preparados, as páginas sem JSON interno e
  a lógica em closures. Esse caminho dá o ganho medido e evita duas semânticas.
- **Condição para reabrir:** os três critérios da seção 3.3, medidos depois dos passos 2 a 5
  do caminho incremental.
