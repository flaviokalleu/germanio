# React: o modelo de componentes e que problema cada mecanismo resolve

Data: 2026-09-28
Status: pesquisa (não normativa). A norma continua em `docs/INTENCAO.md`.

## Fontes consultadas

- Repositório: https://github.com/facebook/react
- Estado e posição na árvore: https://react.dev/learn/preserving-and-resetting-state
- Efeitos desnecessários: https://react.dev/learn/you-might-not-need-an-effect
- React Compiler: https://react.dev/learn/react-compiler/introduction
- Server Components: https://react.dev/reference/rsc/server-components
- Streaming SSR e hydration seletiva (grupo de trabalho do React 18): https://github.com/reactwg/react-18/discussions/37
- `renderToPipeableStream`: https://react.dev/reference/react-dom/server/renderToPipeableStream
- `hydrateRoot`: https://react.dev/reference/react-dom/client/hydrateRoot
- `Suspense`: https://react.dev/reference/react/Suspense
- Error boundaries: https://react.dev/reference/react/Component#catching-rendering-errors-with-an-error-boundary
- `useActionState` (e `useOptimistic`): https://react.dev/reference/react/useActionState
- Arquitetura Fiber (Andrew Clark, equipe React): https://github.com/acdlite/react-fiber-architecture
- Código do Germanio lido: `runtime/servidor/paginas.go`, `runtime/servidor/websocket.go`,
  `runtime/servidor/servidor.go` (trechos de broadcast), `compiler/ast/intencao.go`.

## 1. O modelo de componentes

Um componente React é uma função pura dos dados (props, estado, contexto) que devolve uma
descrição da interface. A composição é aninhamento de funções. O modelo mental central é
"UI = f(estado)": a cada mudança, a função roda de novo e o React reconcilia o resultado.

Duas consequências estruturam tudo o resto:

1. **Re-executar é o padrão.** Mudou o estado de um componente, ele e os filhos rodam de novo
   (a menos que se memorize). Daí nasce toda a família de otimização (seção 5).
2. **O estado pertence à posição na árvore, não ao componente.** "Same position = same state";
   mudar o tipo na mesma posição destrói o estado; a `key` dá identidade explícita e força o
   reset (https://react.dev/learn/preserving-and-resetting-state).

## 2. Reconciliation e Fiber

A reconciliação usa duas heurísticas: tipos diferentes produzem árvores diferentes (troca-se a
subárvore sem diff) e listas são comparadas por `key` estável e única. O Fiber reimplementou
o motor como "virtual stack frame" em memória para poder pausar, abortar, reutilizar e
priorizar trabalho (renderização incremental em vários frames) sem mudar as heurísticas
(https://github.com/acdlite/react-fiber-architecture).

Problema resolvido: no cliente, uma re-renderização grande trava a thread principal; o Fiber
fatia o trabalho. Esse problema **só existe porque a árvore é recalculada no navegador**.

## 3. Eventos

Eventos são funções passadas como props (`onClick`); o React delega ouvintes na raiz. A
documentação separa o que roda "porque o usuário fez algo" (evento) do que roda "porque o
componente apareceu" (efeito): "Code that runs because a component was displayed should be in
Effects, the rest should be in events" (https://react.dev/learn/you-might-not-need-an-effect).

## 4. Os hooks, pelo problema que resolvem

A pergunta não é "o Germanio deve ter `useState`?", e sim "o problema que o hook resolve
existe no Germanio?". O Germanio hoje renderiza por requisição no servidor, sem árvore no
cliente (`runtime/servidor/paginas.go`, nenhum `<script>` nas páginas de intenção).

| Mecanismo | Problema do modelo React | Existe no Germanio? |
| --- | --- | --- |
| `useState` | a função do componente é re-executada; valores locais seriam perdidos entre execuções, então o estado precisa viver fora da função, ligado à posição | Quase não. O estado relevante é do **domínio** (banco) ou da **URL** (`q`, `pagina`, filtros já são query string em `serve`). Resta estado efêmero de UI (menu aberto, aba ativa), que HTML nativo cobre em grande parte (`<details>`, `<dialog>`, âncoras) |
| `useEffect` | sincronizar com sistemas externos após a renderização (assinaturas, APIs do navegador, widgets); a doc lista os usos indevidos: estado derivado, reação a evento, reset por prop | Não no `.ge`. Efeitos de domínio são `quando …` (servidor). A única sincronização externa da UI seria assinar mudanças em tempo real, e ela é **derivável** das dependências da página |
| `useMemo`, `useCallback`, `memo` | re-execução em cascata e identidade de funções/objetos mudando a cada render, invalidando memoização dos filhos | Não. Uma renderização por requisição não tem "renderizar de novo" para evitar |
| `useContext` | passar dados por muitos níveis ("prop drilling") | Não. Usuário atual, tema, sistema são implícitos no runtime (`view.User`, `view.System`) |
| `useReducer` | transições de estado complexas centralizadas | Existe, mas no domínio: estados e transições (`começa aberta`, `pode fechar`) já são a máquina de estados do Germanio |
| `useRef` | escape imperativo para nós do DOM e valores mutáveis sem re-render | Não no domínio. Foco, rolagem e medição são comportamento do alvo Web, decididos pelo runtime (ex.: foco no primeiro erro) |
| `useSyncExternalStore` | assinar um store externo sem tearing em renderização concorrente | Só se houver runtime cliente; no Germanio, "store externo" é o servidor, e a assinatura seria do runtime, não do autor |
| `useTransition` | marcar atualizações como não urgentes para não travar a UI | Não: não há renderização concorrente no cliente |
| `useActionState` | estado de um envio de formulário (resultado, `isPending`), inclusive antes da hydration via `permalink` | **Sim, o problema existe**: todo formulário tem enviando/erro/sucesso. No Germanio isso deve ser **derivado** da entidade e do resultado da operação, não declarado |
| `useOptimistic` | mostrar o resultado antes da confirmação do servidor | Problema real apenas para interações de alta frequência; em CRUD com PRG é dispensável. Investigar |
| `key` | identidade de itens de lista para preservar/resetar estado | Já resolvido: o registro tem `id`/`numero` (`table`, `paginas.go:410`) |

O dado mais revelador está na própria documentação do `useActionState`: o parâmetro
`permalink` existe para que "the form is submitted before the JavaScript bundle loads" navegue
para uma URL real (https://react.dev/reference/react/useActionState). Ou seja, o React
reconstrói, por cima do JS, o comportamento que um `<form method="post">` já tem. O Germanio
já parte desse comportamento (`formTpl`, `post` com 303).

## 5. React Compiler: por que existe

A documentação: "manual memoization is tedious, easy to get wrong, and adds extra code to
maintain"; o compilador aplica a memoização automaticamente e "assumes your code follows the
Rules of React"; está estável e "in the future some features may require the compiler"
(https://react.dev/learn/react-compiler/introduction).

Leitura: o compilador existe para **pagar uma dívida do modelo** (re-executar tudo por padrão)
sem mudar o modelo. A lição para o Germanio não é "ter um compilador que memoiza", é "escolher
um modelo em que a memoização não seja responsabilidade de ninguém". É o mesmo princípio de
subtração do Germanio: não exigir do autor o que o sistema pode inferir, e melhor ainda, não
criar o problema.

## 6. Server Components

São componentes que "render ahead of time, before bundling, in an environment separate from
your client app", no build ou por requisição; o cliente "will only see the rendered output",
e as bibliotecas usadas para renderizar ficam fora do bundle; podem ser `async` e ler o banco
diretamente; não têm estado nem interatividade; compõem-se com Client Components marcados
`"use client"` (https://react.dev/reference/rsc/server-components).

É o React chegando, por outro caminho, ao ponto em que o Germanio já está: a maior parte da
interface é função de dados no servidor e não precisa de código no cliente. A diferença é que
o React mantém um protocolo próprio (o payload RSC) para compor servidor e cliente na mesma
árvore, o que exige empacotador, fronteiras `"use client"`/`"use server"` e regras de
serialização. O Germanio não tem a metade cliente e não deveria adotar esse protocolo.

## 7. SSR streaming, hydration, hydration seletiva, Suspense

O grupo de trabalho do React 18 descreve a cascata do SSR clássico: buscar tudo antes de
mostrar algo, carregar todo o JS antes de hidratar algo, hidratar tudo antes de interagir com
algo. A resposta é HTML em streaming com `<Suspense>` (fallback primeiro, conteúdo depois no
mesmo fluxo) e hydration seletiva, priorizando a região que o usuário clicou
(https://github.com/reactwg/react-18/discussions/37). O `renderToPipeableStream` envia o
"shell" primeiro; depois de começar o `pipe()` o código de status HTTP não muda mais
(https://react.dev/reference/react-dom/server/renderToPipeableStream).

`Suspense` só é acionado por fontes integradas (lazy, `use`, frameworks com Suspense,
streaming) e "does not detect when data is fetched inside an Effect or event handler"
(https://react.dev/reference/react/Suspense). A hydration exige que o cliente produza "the same
output" que o servidor; divergências podem, no pior caso, fazer "event handlers … get attached
to the wrong elements" (https://react.dev/reference/react-dom/client/hydrateRoot).

Detalhes e o que o Germanio evita em `SSR_HYDRATION.md`.

## 8. Error boundaries

Capturam erros de renderização dos descendentes; não capturam erros em event handlers, código
assíncrono, SSR, nem no próprio boundary; ainda exigem componente de classe
(https://react.dev/reference/react/Component#catching-rendering-errors-with-an-error-boundary).

O problema real é: **uma região que falha não deve derrubar a página inteira, e a falha deve
ser visível**. No Germanio, `serve` já isola regiões na prática: ao listar os filhos de um
registro, `if ccode != 200 { continue }` (`paginas.go`, laço "Children"). Isso trata igual
"sem permissão para ver" (esconder é correto) e "erro interno" (esconder é silêncio, contra a
regra dos diagnósticos em quatro partes). A lição é por região, com semântica do domínio, não
um componente especial.

## 9. Ecossistema

A força do React é o ecossistema (roteadores, frameworks como Next, bibliotecas de
componentes, React Native). O custo é que a aplicação típica é montada a partir de dezenas de
escolhas (roteamento, dados, formulários, estilos, SSR) que o autor precisa conhecer. O
Germanio quer o oposto para o nível padrão: nenhuma dessas escolhas visível. O ecossistema é
útil como catálogo do que as aplicações precisam (formulários com estado, tabelas,
paginação, diálogos), não como API a imitar.

## 10. Achados colaterais no código do Germanio

- O `/ws` (`runtime/servidor/websocket.go:67`) faz o handshake sem verificar sessão nem
  `Origin`, e o CRUD legado transmite o registro inteiro a todos os clientes
  (`servidor.go:692`, `:726`: `Broadcast(WSMessage{…, Data: item})`). Se o tempo real das
  páginas de intenção for construído sobre esse hub, vazará dados que as permissões escondem.
  Qualquer atualização em tempo real precisa renderizar **por espectador**, com as mesmas
  permissões do `serve`.

## Para o Germanio

- **ADOTAR** "UI = f(dados do domínio, permissões, URL)" como modelo único das páginas de
  intenção, sem estado de componente declarado no `.ge`. Resolve: evitar a importação de
  `useState`/`useEffect`/`useMemo`, que resolvem problemas que o modelo por requisição não tem.
  Afeta `docs/INTENCAO.md` (seção "Página", quando G62 for decidido).
- **ADOTAR** a derivação do estado de formulário (vazio, enviando, erro por campo, sucesso) a
  partir da entidade e do resultado da operação, que é o problema de `useActionState`.
  Resolve: `formulário cliente` derivar required, validação, mensagens e estados sem
  declaração (direção do contexto de frontend); hoje `post` redireciona com `?erro=` e o
  formulário volta vazio. Afeta `runtime/servidor/paginas.go` (`post`, `formTpl`).
- **ADAPTAR** error boundaries como falha por região: sem permissão, a região some (como
  hoje); erro interno, a região mostra uma mensagem explicada e o resto da página renderiza.
  Resolve: o `continue` silencioso na listagem de filhos em `serve`.
- **ADAPTAR** a `key` como identidade de registro já existente (`id`/`numero`) para qualquer
  atualização parcial futura (substituir a linha do registro X). Resolve: identidade estável
  em listas atualizadas em tempo real, sem pedir nada ao autor.
- **EVITAR** hooks, componentes com estado, Fiber, renderização concorrente e o protocolo RSC.
  Resolve: nenhum problema atual; cada um compensa o custo de re-executar a UI no cliente.
- **EVITAR** que o autor precise de memoização ou de marcar fronteiras servidor/cliente
  (`"use client"`). A existência do React Compiler é a evidência de que isso é dívida do
  modelo. Resolve: princípio de subtração (`skills/germanio-simplicity/SKILL.md`, seção 3).
- **INVESTIGAR** resposta otimista (`useOptimistic`) apenas para interações de alta frequência
  (curtir, marcar tarefa), depois que existir runtime cliente mínimo; para CRUD com PRG, não.
- **INVESTIGAR**, antes de qualquer tempo real nas páginas: autenticação e verificação de
  `Origin` no `/ws` e broadcast filtrado por permissão. Resolve: vazamento potencial descrito na
  seção 10 (`runtime/servidor/websocket.go`, `runtime/servidor/servidor.go`).
