# SSR, hydration e o que o Germanio pode evitar

Data: 2026-09-28
Status: pesquisa (não normativa). A norma continua em `docs/INTENCAO.md`.

## Fontes consultadas

- Panorama e custo da rehydration: https://web.dev/articles/rendering-on-the-web
- Streaming SSR e hydration seletiva (React 18): https://github.com/reactwg/react-18/discussions/37
- `renderToPipeableStream`: https://react.dev/reference/react-dom/server/renderToPipeableStream
- `hydrateRoot` (requisito de saída idêntica, causas de divergência): https://react.dev/reference/react-dom/client/hydrateRoot
- `Suspense`: https://react.dev/reference/react/Suspense
- `useActionState` (`permalink` antes da hydration): https://react.dev/reference/react/useActionState
- Angular, hydration, event replay e restrições: https://angular.dev/guide/hydration
- Vue, patch flags e hydration: https://vuejs.org/guide/extras/rendering-mechanism.html e
  https://github.com/vuejs/core/blob/main/packages/shared/src/patchFlags.ts
- Astro islands: https://docs.astro.build/en/concepts/islands/
- Jason Miller, "Islands Architecture": https://jasonformat.com/islands-architecture/
- Qwik, resumability: https://qwik.dev/docs/concepts/resumable/
- Phoenix LiveView: https://phoenix-live-view.hexdocs.pm/Phoenix.LiveView.html
- Código do Germanio lido: `runtime/servidor/paginas.go` (`serve`, `call`, `post`, templates).

## 1. Definições

- **SSR**: o servidor produz o HTML da página para a requisição.
- **Hydration**: o cliente baixa o código dos componentes, reexecuta-os sobre o HTML recebido
  e liga ouvintes e estado aos nós existentes, sem recriá-los. Angular descreve o ganho como
  evitar que a app "destroy and re-render the application's DOM", o que causaria cintilação
  (https://angular.dev/guide/hydration).
- **Hydration progressiva / parcial / seletiva**: hidratar por partes, no tempo (progressiva),
  só onde há interatividade (parcial) ou priorizando a região com que o usuário interage
  (seletiva) (https://web.dev/articles/rendering-on-the-web; https://github.com/reactwg/react-18/discussions/37).
- **Islands**: HTML estático com regiões isoladas que hidratam independentemente
  (https://docs.astro.build/en/concepts/islands/; https://jasonformat.com/islands-architecture/).
- **Resumability**: não reexecutar; retomar a partir de estado e ouvintes serializados no HTML
  (https://qwik.dev/docs/concepts/resumable/).
- **Streaming**: enviar o HTML em partes à medida que fica pronto
  (https://react.dev/reference/react-dom/server/renderToPipeableStream).

## 2. O custo da hydration

Qwik enumera o que a hydration precisa restaurar: os ouvintes, a árvore de componentes e o
estado da aplicação; e diz que ela é cara porque o framework precisa "download all of the
component code associated with the current page" e "execute the templates associated with the
components on the page" (https://qwik.dev/docs/concepts/resumable/). O web.dev acrescenta o
"uncanny valley" (a página parece pronta e não responde até o bundle executar) e a duplicação:
descrição da UI, dados de origem e código de implementação são enviados juntos
(https://web.dev/articles/rendering-on-the-web).

Há ainda o custo de **correção**:

- A árvore do cliente precisa produzir "the same output" do servidor; espaços extras, testes de
  `typeof window`, APIs só do navegador e dados diferentes causam divergência; no pior caso,
  "event handlers can get attached to the wrong elements"
  (https://react.dev/reference/react-dom/client/hydrateRoot).
- Angular exige DOM idêntico, proíbe manipulação direta do DOM e `innerHTML` nos componentes
  hidratados, e exige HTML válido (por exemplo `<tbody>` explícito); oferece `ngSkipHydration`
  como saída, perdendo o benefício (https://angular.dev/guide/hydration).
- Interações antes do fim da hydration se perdem, daí mecanismos como o event replay do Angular
  (https://angular.dev/guide/hydration) e o `permalink` do `useActionState`, que faz um
  formulário enviado "before the JavaScript bundle loads" navegar para uma URL real
  (https://react.dev/reference/react/useActionState).

Cada técnica da seção 1 é um jeito de pagar menos desse custo: hidratar menos (ilhas, parcial),
mais tarde (progressiva, seletiva), ou nunca (resumability).

## 3. Streaming

O React 18 descreve três cascatas do SSR clássico: buscar tudo antes de mostrar algo, carregar
todo o JS antes de hidratar algo, hidratar tudo antes de interagir
(https://github.com/reactwg/react-18/discussions/37). O streaming ataca a primeira: o "shell"
sai primeiro e as regiões suspensas chegam depois no mesmo fluxo. Custos documentados: depois
do início do `pipe()` o status HTTP não muda mais; erros no shell e erros dentro de
`<Suspense>` têm tratamentos diferentes (fallback no servidor e nova tentativa no cliente)
(https://react.dev/reference/react-dom/server/renderToPipeableStream; https://react.dev/reference/react/Suspense).

As outras duas cascatas **só existem se houver hydration**. O streaming de HTML em si não
depende de framework: é `http.Flusher` em Go.

## 4. O Germanio, lido no código

- `serve` monta a página inteira e só então escreve (`ps.render`), com status definido no fim.
- Não há código de componente no cliente, nem estado serializado, nem ouvintes JS: os
  formulários são `method="post"` e os links são `<a href>` (`formTpl`, `tableTpl`, `pagerTpl`).
- Os dados de uma página de registro vêm de chamadas internas sequenciais à própria API via
  `httptest` (`call`, `paginas.go:85`): uma para o registro e uma para cada tipo de filho
  (`?por_pagina=10`), além das consultas dos nomes nas migalhas. Cada chamada serializa e
  desserializa JSON dentro do mesmo processo.

Consequência direta: **o Germanio não tem hydration, e portanto não tem nenhum dos custos da
seção 2**. Nada a baixar para ficar interativo; nada a reexecutar; nenhuma divergência
servidor/cliente possível; nenhum clique perdido antes da hydration, porque o clique num link
ou botão de formulário é tratado pelo próprio navegador. A resumability do Qwik é, no limite,
o que o HTML de formulários e links já é: o "handler" está serializado no próprio documento
(`action`, `method`, `href`), e o "ouvinte global" é o navegador.

## 5. O que o Germanio pode evitar completamente, e sob que condição

Evita tudo o que existe para reconciliar uma app cliente com o HTML do servidor: hydration de
qualquer tipo, event replay, `permalink`, marcadores de hidratação, `ngSkipHydration`,
serialização de estado de componentes, e os erros de "hydration mismatch".

A condição é uma regra de arquitetura, não uma otimização: **nenhum script do Germanio assume
a posse de uma árvore de DOM renderizada pelo servidor para reconstruí-la**. Um script de
melhoria (ver `RENDERING.md`, camadas 2 e 3) pode:

- interceptar a navegação e o envio de um formulário, pedir ao servidor o HTML de uma região e
  substituí-la;
- aplicar fragmentos HTML recebidos em tempo real sobre regiões identificadas;
- mover o foco e anunciar mudanças.

Ele não pode manter estado de UI próprio que precise coincidir com o servidor. Com isso, a
granularidade do que muda é a **região** do modelo visual (a tabela de issues, o formulário, o
cabeçalho do registro), a identidade de uma linha é o `id`/`numero` do registro, e a fonte da
verdade continua única. É a mesma posição do htmx, do Turbo e do primeiro render do LiveView,
sem o processo com estado por conexão do LiveView
(https://phoenix-live-view.hexdocs.pm/Phoenix.LiveView.html).

Onde a regra se rompe: widgets intrinsecamente clientes (gráfico interativo, editor, mapa,
arrastar). Para eles, ilhas: uma região que carrega seu próprio script quando visível e recebe
dados do servidor, isolada das demais (https://docs.astro.build/en/concepts/islands/). Mesmo aí,
"hidratar" é inicializar um widget sobre dados, não reconciliar uma árvore.

## 6. Streaming no Germanio: quando valeria

O problema que o streaming resolve (a região lenta segura a página toda) pode surgir no
Germanio: páginas de registro com muitos tipos de filhos fazem chamadas internas sequenciais.
Antes de streaming, há soluções mais simples e sem os custos da seção 3:

1. medir o tempo por região (o modelo visual permite atribuir tempo a cada nó);
2. buscar as regiões em paralelo, ou chamar a camada de dados diretamente em vez de passar por
   `httptest` + JSON, mantendo as mesmas verificações de permissão;
3. só então considerar enviar o cabeçalho e o conteúdo principal com `Flush` e as regiões
   secundárias depois, aceitando que o status HTTP fica fixo após o primeiro envio.

Uma alternativa sem streaming é o carregamento preguiçoso por região (o papel dos Turbo Frames
preguiçosos e dos "server islands" do Astro): a região secundária vem como link/placeholder e é
pedida à parte. Sem JS, o link continua navegável.

## Para o Germanio

- **ADOTAR** como regra de arquitetura: nenhum script do core reconstrói ou reconcilia DOM
  renderizado pelo servidor; scripts só trocam regiões por HTML do servidor, aplicam fragmentos
  e cuidam de foco e anúncios. Resolve: elimina por construção hydration, divergências
  servidor/cliente, cliques perdidos e bundle por página. Afeta `docs/INTENCAO.md` (quando a
  camada de melhoria for normatizada) e `runtime/servidor/`.
- **ADOTAR** formulários e links reais como o "estado serializado" da página: `action`,
  `method` e `href` gerados pelo modelo visual continuam sendo o contrato, com ou sem script.
  Resolve: o que o `permalink` do React e o event replay do Angular reconstroem. Afeta
  `formTpl`, `tableTpl`, `pagerTpl` em `runtime/servidor/paginas.go`.
- **ADAPTAR** a região como unidade de atualização (equivalente, no servidor, a Suspense
  boundary + ilha): cada nó de primeiro nível do modelo visual tem endereço próprio e pode ser
  renderizado sozinho, com as permissões do espectador. Resolve: atualização parcial,
  carregamento preguiçoso e tempo real sem hydration.
- **ADAPTAR** o carregamento preguiçoso de regiões secundárias (filhos de um registro) antes de
  qualquer streaming. Resolve: páginas de registro lentas por chamadas internas sequenciais em
  `serve`, sem o custo do status HTTP fixo.
- **EVITAR** hydration total, parcial, progressiva ou seletiva, e resumability com
  serialização de estado. Resolve: nenhum problema atual; todos pressupõem uma app cliente que
  o Germanio não tem.
- **EVITAR** estado de UI por conexão no servidor (modelo LiveView). Resolve: memória por
  conexão e reconexão com estado; o Germanio já tem o estado relevante no banco e na URL.
- **INVESTIGAR** o custo real das chamadas internas via `httptest` + JSON em `call`
  (`paginas.go:85`) em páginas com muitos filhos, medido, antes de decidir entre paralelismo,
  chamada direta à camada de dados, regiões preguiçosas ou streaming.
- **INVESTIGAR** o contrato de uma ilha (dados de entrada vindos da região, eventos de saída
  como envios de formulário comuns) para gráficos e kanban, sem expô-lo ao nível padrão.
