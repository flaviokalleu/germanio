# Estratégias de renderização e a recomendação para o Germanio

Data: 2026-09-28
Status: pesquisa (não normativa). A norma continua em `docs/INTENCAO.md`.

## Fontes consultadas

- Panorama das estratégias (Google, web.dev): https://web.dev/articles/rendering-on-the-web
- Hypermedia-Driven Applications (htmx): https://htmx.org/essays/hypermedia-driven-applications/
- Quando usar hypermedia (htmx): https://htmx.org/essays/when-to-use-hypermedia/
- `hx-boost`: https://htmx.org/attributes/hx-boost/
- Hotwire Turbo: https://turbo.hotwired.dev/handbook/introduction
- Phoenix LiveView: https://phoenix-live-view.hexdocs.pm/Phoenix.LiveView.html
- Svelte, "Virtual DOM is pure overhead" (Rich Harris): https://svelte.dev/blog/virtual-dom-is-pure-overhead
- Svelte 5 runes: https://svelte.dev/docs/svelte/what-are-runes
- Solid, reatividade fina: https://docs.solidjs.com/advanced-concepts/fine-grained-reactivity
- Astro islands: https://docs.astro.build/en/concepts/islands/
- Jason Miller, "Islands Architecture": https://jasonformat.com/islands-architecture/
- Qwik, resumability: https://qwik.dev/docs/concepts/resumable/
- Preact Signals: https://preactjs.com/guide/v10/signals/
- Angular zoneless: https://angular.dev/guide/zoneless
- Vue, mecanismo de renderização: https://vuejs.org/guide/extras/rendering-mechanism.html
- Go `html/template`: https://pkg.go.dev/html/template
- Go `http.CrossOriginProtection`: https://pkg.go.dev/net/http#CrossOriginProtection
- OWASP CSRF Prevention Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html
- W3C, "Using ARIA" (primeira regra): https://www.w3.org/TR/using-aria/
- View Transitions entre documentos: https://developer.chrome.com/docs/web-platform/view-transitions/cross-document
- Speculation Rules API (MDN): https://developer.mozilla.org/en-US/docs/Web/API/Speculation_Rules_API
- Offline em PWAs (MDN): https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Guides/Offline_and_background_operation
- Código do Germanio lido: `runtime/servidor/paginas.go`, `runtime/servidor/identidade.go:209`,
  `runtime/servidor/websocket.go`, `runtime/servidor/renderizador.go:33-34`,
  `runtime/interpreter/markdown.go`, `go.mod`.

## 1. O espectro

O web.dev separa: SSR (HTML completo no servidor), renderização estática (HTML no build), CSR
(tudo no navegador), rehydration (HTML do servidor + JS que reinicializa a app), streaming SSR,
hydration progressiva e trisomorphic (streaming + service worker). Sobre a rehydration: "The UI
doesn't become interactive until after bundle.js has finished loading and executing", o
"uncanny valley" em que a página parece pronta e não responde, e a duplicação (descrição da UI,
dados e código). A conclusão: "It's fine to mostly ship HTML with minimal JavaScript"
(https://web.dev/articles/rendering-on-the-web).

## 2. As famílias, pelo mecanismo

### 2.1 HTML do servidor + melhoria progressiva (htmx, Hotwire, LiveView)

- **htmx / HDA**: interatividade declarada em atributos HTML, troca de **HTML** (não JSON) com
  o servidor, script só onde necessário (https://htmx.org/essays/hypermedia-driven-applications/).
  `hx-boost` transforma links e formulários em requisições AJAX e, sem JS, "the site will
  continue to work" (https://htmx.org/attributes/hx-boost/). O próprio htmx diz onde não serve:
  muitas dependências de UI espalhadas que não permitem "update the whole UI", offline,
  atualizações muito frequentes (arrastar, jogos) (https://htmx.org/essays/when-to-use-hypermedia/).
- **Turbo**: Drive intercepta links do mesmo domínio e troca o corpo; Frames recortam a página em
  regiões com navegação própria e carregamento preguiçoso; Streams aplicam ações (append,
  replace…) com HTML vindo por WebSocket/SSE, "reuse the server-side templates"; "All the logic
  lives on the server, and the browser deals just with the final HTML"
  (https://turbo.hotwired.dev/handbook/introduction).
- **LiveView**: primeiro render HTTP estático, "a regular HTML page even if JavaScript is
  disabled"; depois um processo por conexão guarda o estado no servidor e envia **diffs**
  (https://phoenix-live-view.hexdocs.pm/Phoenix.LiveView.html). O custo é memória e latência
  por conexão: cada interação faz ida e volta.

Mecanismo comum: o servidor é a fonte da verdade e do HTML; o cliente é um aplicador genérico,
igual para qualquer aplicação.

### 2.2 SPA com VDOM (React, Vue clássico)

A árvore é recalculada no cliente e reconciliada. Vue reduz o custo com informação do
compilador (patch flags, blocks, cache estático) (https://vuejs.org/guide/extras/rendering-mechanism.html);
React com Fiber e agora o React Compiler (ver `REACT.md`). Exige bundle, roteamento cliente e,
com SSR, hydration.

### 2.3 Compilado sem VDOM (Svelte, Solid, Vue Vapor)

Svelte: o compilador "knows at build time how things could change in your app" e emite código
de atualização cirúrgico; o VDOM é "a tradeoff", não uma feature
(https://svelte.dev/blog/virtual-dom-is-pure-overhead); no Svelte 5, runes (`$state`,
`$derived`, `$effect`) são sinais controlados pelo compilador
(https://svelte.dev/docs/svelte/what-are-runes). Solid: componentes rodam uma vez; sinais
notificam só o nó dependente (https://docs.solidjs.com/advanced-concepts/fine-grained-reactivity).
Vue Vapor segue o mesmo caminho, ainda em RC (ver `VUE.md`).

Lição transferível: **se o sistema conhece estaticamente o que depende de quê, atualizar é
endereçar o dependente, não comparar árvores**.

### 2.4 Islands (Astro)

A página é HTML estático; só "ilhas" marcadas carregam JS (`client:load`, `client:visible`);
por padrão, zero JS; "server islands" (`server:defer`) renderizam no servidor partes
personalizadas sem atrasar o resto (https://docs.astro.build/en/concepts/islands/). Jason Miller
destaca que cada ilha inicializa sem depender das outras e que "the default outcome [is] the
accessible one"; a desvantagem admitida é a falta de opções prontas para decompor apps em
widgets independentes (https://jasonformat.com/islands-architecture/).

### 2.5 Resumability (Qwik)

Em vez de reexecutar a app no cliente, "pauses execution on the server, and resumes execution
on the client"; os handlers vão serializados em atributos, um único ouvinte global carrega o
código do handler na primeira interação (https://qwik.dev/docs/concepts/resumable/). Resolve o
custo da hydration de apps com muito estado no cliente; exige um compilador que fatia o código
e um formato de serialização.

### 2.6 Preact Signals e Angular zoneless

Preact: um sinal em posição de texto no JSX "will render as text and automatically update
in-place without Virtual DOM diffing" (https://preactjs.com/guide/v10/signals/). Angular v21+
é zoneless por padrão: deixa de interceptar todo código assíncrono (Zone.js) e só detecta
mudanças por notificações explícitas (sinais no template, eventos do template, `markForCheck`,
`AsyncPipe`) (https://angular.dev/guide/zoneless). Os dois são o mesmo movimento do 2.3:
reatividade por dependência explícita em vez de "algo pode ter mudado, recalcule".

## 3. Onde o Germanio está hoje

- Páginas de intenção: HTML completo por requisição com `html/template`, CSS inline, **zero
  JS**, formulários `method="post"` com `_csrf` e Post/Redirect/Get (`paginas.go:951-1027`).
  É a família 2.1 sem a camada de melhoria.
- Cookie de sessão `HttpOnly`, `SameSite=Lax`, `Secure` quando TLS (`identidade.go:209`); token
  sincronizador conferido no `post` (`paginas.go:966`).
- Markdown de usuários via goldmark sem HTML cru e com esquemas perigosos neutralizados
  (`runtime/interpreter/markdown.go`), inserido como `template.HTML`.
- SPA legado (`renderizador.go`) carrega Chart.js e o Tailwind de CDN em tempo de execução
  (linhas 33-34): dependência de rede de terceiros, contrária a offline e a uma CSP estrita.
- Hub WebSocket (`websocket.go`) sem autenticação nem verificação de `Origin`, com broadcast
  global (ver `REACT.md`, seção 10).

## 4. Avaliação pelos critérios pedidos

| Critério | 2.1 HTML + melhoria | 2.2 SPA VDOM | 2.3 Compilado | 2.4 Islands | 2.5 Resumable |
| --- | --- | --- | --- | --- | --- |
| Funciona sem JS pesado | sim, por construção | não | parcial (precisa do bundle da página) | sim fora das ilhas | sim até a interação |
| Segurança (XSS/CSRF) | escaping no servidor (`html/template` contextual) + formulários com token; superfície JS mínima | API JSON + token em header + escaping no cliente; duas superfícies | idem SPA | ilhas repetem a superfície SPA | serialização de estado no HTML é superfície nova |
| Simplicidade do runtime em Go | máxima: o Go já faz tudo; o JS é um aplicador genérico pequeno | exige gerar/servir um app JS e manter paridade servidor/cliente | exige um compilador para JS | exige compilador de ilhas | exige compilador + serializador |
| Desempenho | primeiro render ótimo; cada interação custa uma ida e volta | render inicial lento sem SSR; interações locais rápidas | ótimo no cliente | ótimo para páginas majoritariamente estáticas | ótimo no início |
| Offline | ruim sem service worker (admitido pelo htmx) | possível com cache e store local | idem SPA | parcial | parcial |
| Acessibilidade | HTML nativo, links reais, histórico do navegador; "default outcome the accessible one" | depende de disciplina (foco, rotas, anúncios) | idem SPA | boa fora das ilhas | depende |

Pontos que decidem:

1. **Segurança**: `html/template` assume "template authors are trusted, while Execute's data
   parameter is not" e escapa por contexto (HTML, atributo, URL, JS, CSS); `template.HTML`
   desliga o escape (https://pkg.go.dev/html/template). No Germanio, os templates são do core
   (autores confiáveis) e os dados vêm do banco: o modelo de confiança coincide. A fronteira a
   vigiar são os pontos que produzem `template.HTML` (`htmlOf`, `interp.Markdown`, os
   `template.HTML(b.String())` de `serve`). Qualquer runtime cliente que monte DOM a partir de
   JSON reabre XSS do lado do cliente; aplicar fragmentos HTML já escapados pelo servidor não.
2. **CSRF**: a OWASP recomenda o token sincronizador, `SameSite` como defesa em profundidade e
   Fetch Metadata (`Sec-Fetch-Site`) como bloqueio leve de requisições cross-site, e lembra que
   XSS derruba qualquer defesa de CSRF
   (https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html).
   O Go 1.25 trouxe `http.CrossOriginProtection` (Sec-Fetch-Site ou Origin vs Host, métodos
   inseguros apenas, sem token) (https://pkg.go.dev/net/http#CrossOriginProtection); o
   `go.mod` do Germanio declara `go 1.26.1`, e o código não o usa (busca por
   `CrossOriginProtection`/`Sec-Fetch` em `runtime/` sem resultados).
3. **Acessibilidade**: "If you can use a native HTML element … then do so" (https://www.w3.org/TR/using-aria/).
   HTML de servidor com `<form>`, `<a>`, `<table>`, `<label>` já é a opção nativa. Lacunas
   concretas lidas em `paginas.go`: a busca e os filtros (`searchTpl`) usam apenas
   `placeholder`, sem `<label>`; os erros de envio voltam como texto geral no topo
   (`?erro=`), sem associação ao campo e com o formulário vazio; a tabela não tem `<caption>`
   nem `scope` nos cabeçalhos.
4. **Navegação suave sem framework**: View Transitions entre documentos são opt-in por CSS
   (`@view-transition { navigation: auto }`), mesma origem, sem JS; Chrome/Edge 126+, Safari
   18.2+, Firefox ainda não (https://developer.chrome.com/docs/web-platform/view-transitions/cross-document).
   Speculation Rules pré-carregam/pré-renderizam navegações futuras com JSON declarativo; ainda
   experimental, fora do Baseline (https://developer.mozilla.org/en-US/docs/Web/API/Speculation_Rules_API).
   Ambos degradam para navegação normal.
5. **Offline** exige service worker (JS), Cache API e, para escrita, Background Sync
   (https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Guides/Offline_and_background_operation).
   Leitura offline de páginas já visitadas é mecânica genérica; escrita offline é resolução de
   conflitos, um problema de domínio que nenhum renderizador resolve sozinho.

## 5. Recomendação

**Base: HTML renderizado no servidor, completo e funcional sem JS (família 2.1), como já é.**
Por cima, em camadas opcionais, todas derivadas do modelo visual (ver `VUE.md`) e nenhuma
declarada pelo autor do `.ge`:

1. **Camada 0 (hoje)**: documentos completos, formulários reais, PRG, links reais.
2. **Camada 1 (CSS, zero JS)**: View Transitions entre documentos; tokens de tema; layout
   responsivo. Degrada sem perda.
3. **Camada 2 (um script genérico pequeno, servido pelo próprio binário)**: troca de regiões
   por HTML do servidor ao enviar formulários e ao paginar/filtrar (o papel de `hx-boost` /
   Turbo Frames), foco no primeiro campo com erro, anúncio em região `aria-live`. O script não
   conhece nenhuma aplicação; as regiões e seus endereços vêm do modelo visual.
4. **Camada 3 (tempo real)**: quando a entidade X muda, o servidor re-renderiza **por
   espectador e com as permissões dele** as regiões que dependem de X e envia o fragmento
   (o papel dos Turbo Streams / diffs do LiveView, mas por região e sem estado de UI no
   servidor). Pré-requisito: `/ws` autenticado e com `Origin` verificada.
5. **Ilhas só como escape técnico** para widgets intrinsecamente clientes (gráfico, editor de
   código, mapa, arrastar do kanban), carregadas quando visíveis. Nunca como modelo geral.

O que não recomendar: SPA com VDOM, compilação para JS no estilo Svelte/Solid, resumability.
Todos resolvem o custo de ter a aplicação no cliente; o Germanio não a tem e, pelos critérios
acima, não ganha nada ao criá-la. O argumento de Svelte/Solid/Vapor que se transfere é outro:
**dependências conhecidas estaticamente permitem atualizar só o dependente**, e isso o Germanio
aplica no servidor, por região.

## Para o Germanio

- **ADOTAR** HTML completo do servidor como camada obrigatória e funcional sem JS; toda
  melhoria degrada para ela. Resolve: "funcionar sem JS pesado", acessibilidade por HTML
  nativo e um único lugar para segurança. Afeta `runtime/servidor/paginas.go`.
- **ADOTAR** `http.CrossOriginProtection` como defesa em profundidade junto do token
  sincronizador e do `SameSite=Lax`. Resolve: proteção de Fetch Metadata recomendada pela OWASP
  e ausente hoje; custo de uma linha no mux. Afeta `runtime/servidor/servidor.go`
  (montagem do handler).
- **ADOTAR** a regra "o cliente só aplica HTML já escapado pelo servidor; nunca monta DOM a
  partir de dados". Resolve: manter o modelo de confiança do `html/template` válido quando
  surgir a camada 2/3.
- **ADAPTAR** Turbo Frames/`hx-boost` como script genérico do core, com regiões derivadas do
  modelo visual. Resolve: recarga da página inteira em cada filtro, página e envio, sem pedir
  atributos ao autor. Afeta `runtime/servidor/` (novo recurso estático embutido).
- **ADAPTAR** Turbo Streams/LiveView como atualização por região e por espectador sobre o hub
  WebSocket. Resolve: tempo real "derivado de quais dados a página usa"; exige corrigir o hub
  (autenticação, `Origin`, filtro por permissão). Afeta `runtime/servidor/websocket.go`.
- **ADAPTAR** os erros de formulário: em vez de 303 com `?erro=`, responder 422 com o
  formulário preenchido, erro junto do campo (`aria-describedby`) e foco no primeiro erro;
  mensagens de sucesso fora da query string. Resolve: perda do que foi digitado, erros não
  associados a campos e o texto arbitrário injetável via `?ok=`/`?erro=` numa URL (conteúdo
  forjado, não XSS, porque é escapado). Afeta `post` e `base` em `paginas.go`.
- **ADAPTAR** acessibilidade dos templates: `<label>` para busca e filtros (hoje só
  `placeholder`), `<caption>` e `scope` nas tabelas. Resolve: lacunas lidas em `searchTpl` e
  `tableTpl`.
- **EVITAR** SPA com VDOM, compilação para JS e resumability como modelo das páginas de
  intenção. Resolve: evita segundo runtime, segunda superfície de XSS e paridade
  servidor/cliente sem necessidade real.
- **EVITAR** dependências de CDN em tempo de execução (Tailwind/Chart.js no SPA legado). Resolve:
  offline, CSP e disponibilidade; recursos devem ir embutidos no binário.
- **INVESTIGAR** Speculation Rules (experimental) e leitura offline por service worker gerado
  (páginas visitadas, somente leitura). Escrita offline só com um exemplo real que a exija.
- **INVESTIGAR** ilhas para widgets clientes (gráfico, kanban) quando a biblioteca de blocos
  existir; decidir o contrato de uma ilha (dados de entrada, eventos de saída) sem expô-lo ao
  nível padrão.
