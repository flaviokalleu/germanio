# Reatividade: síntese comparativa e proposta para o Germanio

Data: 2026-09-28
Status: pesquisa concluída; proposta não implementada. Complementa `SVELTE.md` e `SOLID.md`
(mesmo diretório).

## Fontes consultadas

- Svelte, "Introducing runes": https://svelte.dev/blog/runes
- Svelte, `$derived` (push-pull): https://svelte.dev/docs/svelte/$derived
- Svelte, "Virtual DOM is pure overhead": https://svelte.dev/blog/virtual-dom-is-pure-overhead
- Solid, reatividade fina: https://docs.solidjs.com/advanced-concepts/fine-grained-reactivity
- Solid, núcleo reativo: https://github.com/solidjs/solid/blob/main/packages/solid/src/reactive/signal.ts
- Vue, "Reactivity in Depth": https://vuejs.org/guide/extras/reactivity-in-depth.html
- React Compiler: https://react.dev/learn/react-compiler/introduction
- Proposta TC39 de Signals: https://github.com/tc39/proposal-signals
- SvelteKit, `load`/`invalidate`: https://svelte.dev/docs/kit/load
- Solid Router, revalidação: https://docs.solidjs.com/solid-router/data-fetching/revalidation
- Phoenix LiveView, change tracking: https://phoenix-live-view.hexdocs.pm/assigns-eex.html
- Phoenix LiveView, ciclo de vida, PubSub e streams: https://phoenix-live-view.hexdocs.pm/Phoenix.LiveView.html
- Turbo Streams: https://turbo.hotwired.dev/handbook/streams
- Turbo, page refreshes com morphing: https://turbo.hotwired.dev/handbook/page_refreshes
- turbo-rails, `Broadcastable`: https://github.com/hotwired/turbo-rails/blob/main/app/models/concerns/turbo/broadcastable.rb
- htmx: https://htmx.org/docs/
- Convex, "How Convex works": https://stack.convex.dev/how-convex-works
- Noria (OSDI 2018): https://pdos.csail.mit.edu/papers/noria:osdi18.pdf

Código do Germanio lido: `runtime/servidor/websocket.go`, `servidor.go`, `eventos.go`,
`transacao.go`, `paginas.go`, `renderizador.go`, `runtime/auth/auth.go`,
`compiler/ast/intencao.go`, `compiler/ast/autorizacao.go`.

## 1. Os modelos, lado a lado

### 1.1 No cliente

| Sistema | Unidade de estado | Como descobre dependências | Quando | Unidade de atualização |
|---------|-------------------|---------------------------|--------|------------------------|
| Svelte 3/4 | `let` do topo + `$:` | análise estática do texto | compilação | trecho do DOM ligado à variável; objeto inteiro invalidado |
| Svelte 5 | runes (`$state`, `$derived`) sobre signals | leituras síncronas em runtime | execução | nó/atributo |
| Solid | signal, store (proxy) | leituras em runtime; fontes limpas a cada execução | execução | nó/atributo |
| Vue 3 | `ref`/`reactive` (Proxy) | `track` no get, `trigger` no set | execução | componente (VDOM); Vapor mode explora nó |
| React | estado do componente | nenhuma: rerender e reconciliação | — | componente, com diff do VDOM |
| React Compiler | o mesmo | análise estática das Rules of React | compilação | componente, com memoização automática |
| TC39 Signals | `Signal.State`, `Signal.Computed` | leituras em runtime | execução | não define (o framework decide) |

Evidências:

- Svelte trocou análise estática por signals porque as dependências de `$:` "são determinadas
  quando o Svelte compila o componente" e a heurística "só funciona para `let` no topo",
  quebrando refatorações (https://svelte.dev/blog/runes). `$derived` é push-pull e não
  propaga valor idêntico (https://svelte.dev/docs/svelte/$derived).
- Solid: componentes rodam uma vez; marca a jusante e executa de cima para baixo para evitar
  glitches; `createMemo` com `equals`; `batch`
  (https://github.com/solidjs/solid/blob/main/packages/solid/src/reactive/signal.ts).
- Vue: `WeakMap<alvo, Map<chave, Set<efeito>>>`; reatividade de runtime, que dispensa build
  mas exige contêineres (`ref`); o Vue experimentou uma transformação de compilação
  (Reactivity Transform) e desistiu; o Vapor mode é inspirado no Solid
  (https://vuejs.org/guide/extras/reactivity-in-depth.html). A página chama signals de "o
  mesmo tipo de primitivo que os refs do Vue".
- React Compiler: ferramenta de build que aplica memoização automática para evitar rerenders
  em cascata e recálculos caros, exigindo que o código siga as Rules of React; `useMemo` e
  afins ficam como escape hatch (https://react.dev/learn/react-compiler/introduction). O
  React mantém o modelo "renderizar tudo e comparar" e usa o compiler para cortar trabalho
  que o programador antes cortava à mão.
- TC39: Stage 1 segundo o README (verificado em https://github.com/tc39/proposal-signals;
  estágio posterior: não verificado); `Computed` preguiçoso e com cache, `Watcher` de baixo
  nível, push-then-pull sem glitches; efeitos, escalonamento, assíncrono e transações ficam
  **fora** de propósito, porque "se ligam ao escalonamento e ao descarte, que são geridos
  pelos frameworks".

**Convergência.** Todos chegaram ao mesmo grafo: fontes, derivados preguiçosos com igualdade,
efeitos na borda, propagação em lote e sem glitches. A divergência é *onde* o grafo é
descoberto: no texto (Svelte 4, React Compiler) ou nas leituras em execução (Svelte 5, Solid,
Vue, TC39). Quem tentou análise estática sobre JavaScript de uso geral recuou (Svelte, Vue) ou
exigiu disciplina do programador (React Compiler e as Rules of React).

### 1.2 No servidor

| Sistema | O que o servidor sabe | O que envia | Quem decide o que atualizar |
|---------|----------------------|-------------|-----------------------------|
| SvelteKit / Solid Router | dependências de `load`/queries | dados; após ação, revalida tudo | framework (grosso) + `depends`/chaves |
| Phoenix LiveView | assigns usados por cada parte dinâmica do template | só as partes dinâmicas que mudaram | compiler do template + `assign` |
| Turbo Streams | nada automático | fragmentos HTML com ação e alvo | programador (ou `broadcasts_to` no modelo) |
| Turbo page refresh | quais páginas assinam o modelo | um sinal "refresh"; o cliente rebusca e faz morph | modelo (`broadcasts_refreshes`) |
| htmx | nada | HTML em resposta a gatilhos (polling, eventos, SSE/WS) | programador, por atributo |
| Convex | *read set* de cada consulta, registrado em execução | resultado novo da consulta | sobreposição write set × read set |
| Noria | grafo de dataflow das consultas parametrizadas | resultados mantidos incrementalmente | o grafo |

Evidências:

- LiveView separa estático e dinâmico no primeiro render e "só reenvia a parte dinâmica se
  ela mudar", inclusive por campo de mapa (`@user.name`); variáveis locais no template e
  passar todos os assigns a filhos **desligam** o rastreamento
  (https://phoenix-live-view.hexdocs.pm/assigns-eex.html). Primeiro render como HTTP normal,
  depois upgrade para conexão com estado; tempo real por `PubSub.subscribe` + `handle_info`;
  `stream/4` para coleções grandes sem guardá-las no servidor
  (https://phoenix-live-view.hexdocs.pm/Phoenix.LiveView.html). A mesma lição do Svelte:
  análise estática funciona até o código ficar livre demais.
- Turbo Streams: oito ações (append, prepend, replace, update, remove, before, after,
  refresh) sobre um id alvo, por HTTP, WebSocket ou SSE, reaproveitando os templates do
  servidor (https://turbo.hotwired.dev/handbook/streams).
- Turbo page refresh: com `turbo-refresh-method` = morph, o refresh só altera os elementos
  que mudaram; `turbo-refresh-scroll` = preserve mantém a rolagem; `data-turbo-permanent`
  exclui elementos; o servidor transmite `<turbo-stream action="refresh">` a quem assina
  (https://turbo.hotwired.dev/handbook/page_refreshes). `broadcasts_refreshes` no modelo, em
  `after_commit`, com debounce de chamadas seguidas ("para processar registros em massa") e
  `request_id` para não mandar o refresh a quem originou a mudança; o próprio código descreve
  isso como "boa fidelidade com um modelo de programação muito mais simples"
  (https://github.com/hotwired/turbo-rails/blob/main/app/models/concerns/turbo/broadcastable.rb).
- htmx: polling (`every 2s`, parar com HTTP 286), `HX-Trigger` na resposta, extensões SSE e
  WebSocket, morph via idiomorph (https://htmx.org/docs/).
- Convex: *read set* por consulta na sessão WebSocket; o gerenciador de assinaturas percorre o
  log de transações uma vez e testa sobreposição; reexecuta; todas as consultas de um cliente
  no mesmo timestamp; exatidão depende de determinismo (https://stack.convex.dev/how-convex-works).
- Noria: consultas parametrizadas compiladas em dataflow que pré-computa leituras e aplica
  escritas incrementalmente, com estado parcial e evicção
  (https://pdos.csail.mit.edu/papers/noria:osdi18.pdf).

**O ponto decisivo.** O refresh do Turbo resolve com um sinal o problema que os outros
resolvem com precisão: o servidor não calcula o que mudou para cada pessoa; avisa "isto pode
ter mudado" e cada navegador pede de novo a própria página, com os próprios cookies, e o morph
faz o resto. As permissões ficam corretas por construção, porque a rebusca é uma requisição
normal.

## 2. O que o Germanio tem hoje (código)

- `websocket.go`: hub em memória, `Broadcast` para todos, sem salas ou filtro; buffer de 64
  por cliente com descarte silencioso; leitor que descarta tudo, sem ping/pong nem frame de
  fechamento.
- `auth.go`: `/ws` isento de token; sem verificação de `Origin` no handshake.
- `servidor.go` (API legada `/api/`): `Broadcast` de `criar`/`atualizar`/`deletar`/`restaurar`
  com o registro inteiro. Combinado com o item anterior, qualquer conexão recebe dados de
  qualquer registro alterado, sem passar por grants ou visibilidade.
- `renderizador.go` (SPA legado): a cada mensagem com `model`, recarrega a lista do modelo,
  as listas inline e todos os gráficos.
- API e páginas de intenção (`intencao.go`, `paginas.go`): nenhum aviso em tempo real; as
  páginas são HTML do servidor com PRG (303 para `?ok=`/`?erro=`); `emit` (`eventos.go`) já é
  chamado em toda mutação e transição, e `afterCommit` (`transacao.go`) executa trabalho
  depois do commit.

Ou seja: o transporte existe, mas desligado do modelo de intenção e sem autorização; a
invalidação legada é "por entidade" e grossa; o ponto transacional certo para publicar já
existe na API de intenção.

## 3. Proposta: reatividade derivada do Knowledge Graph

Objetivo: a pessoa escreve `página Dashboard` com `total de clientes`, `vendas do mês`,
`receita do mês`, ou `página Projetos` com `mostre projetos`, e as regiões se atualizam quando
outra pessoa muda os dados, sem nenhuma palavra sobre atualização. Nenhuma sintaxe nova.

### 3.1 Compilação: o grafo de leitura

Depois do `resolver.go`, uma análise de páginas produz, para cada `PageDecl`, regiões com
identificador estável (`projetos/lista`, `dashboard/vendas-do-mes`) e, para cada região, suas
leituras:

- entidade principal, campos mostrados, filtros fixos (período, estado) e parâmetros da URL
  (a cadeia de pais em `paginas.go`/`resolve`);
- dependências derivadas do `ast.App`: entidades referenciadas cujo nome é mostrado; o
  `MemberModel` e os campos de visibilidade quando a entidade é filtrada por acesso; campos de
  estado e somente-leitura que mudam as ações disponíveis;
- dependência temporal quando há período relativo ("do mês", "hoje").

Tudo isso é determinístico porque páginas não têm código arbitrário. `ge explain página
Dashboard` mostra: "vendas do mês atualiza quando uma venda é criada, excluída ou tem valor ou
data alterados; e na virada do mês".

### 3.2 Execução: do commit ao navegador

1. **Publicação.** `emit` registra, via `afterCommit`, um fato `(entidade, ação, id, campos
   alterados)`. Nada é publicado antes do commit nem se houver rollback.
2. **Assinatura.** Ao abrir uma página, o navegador conecta ao socket **autenticado** (o mesmo
   cookie de sessão; `Origin` verificado) e informa as regiões presentes. O servidor mantém um
   índice `entidade → (conexão, região)` a partir do grafo estático.
3. **Seleção.** Para cada fato, as regiões que leem a entidade; um filtro estático barato
   descarta campos não lidos e períodos fora do filtro.
4. **Aviso.** O servidor envia só o sinal `{região, versão}`, nunca dados. Coalescência numa
   janela curta (ex.: 200 ms, valor a medir) e deduplicação por região. A conexão que originou
   a mudança é omitida (o PRG já lhe entrega a página nova), como `request_id` no Turbo.
5. **Rebusca.** O script genérico (o mesmo para toda aplicação) pede o HTML da região por GET
   normal, com os cookies da pessoa; o servidor renderiza com as permissões dela, como já
   faz; o cliente substitui a região com morph, preservando foco, rolagem e o que está sendo
   digitado.
6. **Reconexão.** Ao reconectar, todas as regiões visíveis são tratadas como possivelmente
   desatualizadas (perda de mensagens nunca produz dado velho permanente).
7. **Sem JavaScript.** A página continua completa e funcional; só não se atualiza sozinha.

### 3.3 Consistência

- Regiões que leem a mesma entidade são rebuscadas numa só requisição (uma leitura do banco,
  um instante), para que "vendas do mês" e "receita do mês" nunca discordem entre si.
- A versão por região (contador monotônico do servidor) permite ao cliente ignorar respostas
  fora de ordem.
- O que a pessoa está editando não é sobrescrito: se a região contém um formulário com
  alterações não enviadas, o cliente mostra "estes dados mudaram" em vez de trocar.

### 3.4 Custo e quando refinar

Começar pelo sinal + rebusca (custo: uma requisição por espectador afetado por janela de
coalescência; nenhum estado por conexão além da lista de regiões). Refinar só com medição:

- enviar o HTML pronto por classe de visibilidade (todos os que veem o mesmo recebem o mesmo
  fragmento), à la Turbo Streams;
- manter contagem e soma incrementalmente (delta por mutação) quando o recálculo dominar,
  à la Noria;
- streaming de regiões lentas no primeiro render (Solid `renderToStream`).

Nenhum desses refinamentos muda o `.ge`: são decisões do runtime.

### 3.5 Por que não signals no cliente

Os signals resolvem "qual nó do DOM depende de qual valor na memória do navegador". No
Germanio o valor não está no navegador, depende de quem vê e muda por ação de outra pessoa; o
custo dominante é a consulta ao banco, não o DOM. Trocar uma região com morph tem precisão
suficiente e mantém uma única fonte de renderização (o servidor), com permissões aplicadas num
só lugar. Signals no cliente só se justificariam para estado puramente local de interface
(abrir/fechar, ordenar localmente), que o Germanio pode gerar sem expor ao usuário.

## Para o Germanio

### ADOTAR

- **Invalidação por sinal + rebusca autenticada + morph (modelo Turbo page refresh).**
  Problema: páginas de intenção não se atualizam; a via legada envia registros a qualquer
  socket. Resolve os dois com permissões corretas por construção. Afeta
  `runtime/servidor/websocket.go`, `paginas.go` (regiões com id e rota para renderizar uma
  região), `renderizador_declarativo.go` (script genérico). Fontes:
  https://turbo.hotwired.dev/handbook/page_refreshes,
  https://github.com/hotwired/turbo-rails/blob/main/app/models/concerns/turbo/broadcastable.rb.
- **Grafo de leitura estático por região, derivado do `ast.App`, explicado por `ge explain`.**
  Problema: o usuário não deve declarar o que atualiza. Afeta um novo passo em `compiler/`,
  `compiler/ast/intencao.go`, `tooling/explicar/`. Fonte: o contraste com
  https://svelte.dev/blog/runes (análise estática falha só quando o código é livre).
- **Publicar em `afterCommit` a partir de `emit`, com coalescência e sem eco para o autor.**
  Afeta `runtime/servidor/eventos.go`, `transacao.go`. Fontes: Convex (consistência após
  commit) e turbo-rails (`after_commit`, debounce, `request_id`).
- **Autenticar o WebSocket e verificar `Origin`; parar de enviar registros na API legada.**
  Problema de segurança atual, independente da proposta. Afeta `runtime/auth/auth.go`,
  `runtime/servidor/servidor.go`, `websocket.go`. Registrar em `GERMANIO_GAPS.md`.

### ADAPTAR

- **Change tracking do LiveView, na granularidade de região.** Enviar só o que mudou, mas por
  região declarada, não por expressão do template; sem processo com estado por conexão.
  Fonte: https://phoenix-live-view.hexdocs.pm/assigns-eex.html.
- **Read set × write set do Convex, estático.** A sobreposição é calculada a partir do grafo
  compilado, não registrada em execução. Fonte: https://stack.convex.dev/how-convex-works.
- **Igualdade antes de propagar (`$derived`, `createMemo equals`).** No servidor, quando se
  passar a enviar conteúdo: não enviar fragmento igual ao anterior.
- **Revalidação grossa após a própria ação (SvelteKit, Solid Router).** Já é o PRG do
  Germanio; manter.

### EVITAR

- **Expor signals, efeitos, `invalidate`, `depends` ou chaves de cache ao usuário.** Todos
  existem nos frameworks porque o framework não vê tudo; o Germanio vê.
- **Rastreamento de dependências em execução no servidor.** Desnecessário com páginas
  fechadas; mais difícil de explicar e de testar.
- **Expressões livres em páginas.** Foram o que desligou a análise estática no Svelte 4 e no
  LiveView (variáveis no template). Cálculos novos entram como agregados do domínio.
- **Enviar dados pelo socket sem renderizar com as permissões de cada espectador.**
- **VDOM ou framework de componentes no cliente para este problema.**

### INVESTIGAR

- **Semântica dos indicadores (G62).** Quais formas (`total de`, `soma de ... do mês`) e como
  resolvem para (entidade, filtro, função). Pré-requisito para as regiões do Dashboard.
- **Custo com muitos espectadores** e agrupamento por classe de visibilidade; medir antes de
  enviar HTML pronto.
- **Manutenção incremental de agregados** (contagem, soma) contra recálculo com coalescência.
  Fonte: https://pdos.csail.mit.edu/papers/noria:osdi18.pdf.
- **Várias instâncias do servidor.** O hub é em memória; a publicação precisa atravessar
  processos (pelo banco ou pela fila de tarefas já usada por `emit` para webhooks).
- **Morph no cliente**: usar uma biblioteca existente (idiomorph, citada pelo Turbo e pelo
  htmx) servida localmente, ou um morph mínimo próprio; verificar licença e tamanho.
- **Dependências temporais** ("do mês", "hoje") no fuso de quem vê.
