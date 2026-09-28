# Responsividade por padrão, sem media queries do usuário

Data: 2026-09-28 · Status: pesquisa (não normativa).

## Fontes consultadas

- Every Layout, índice dos layouts: https://every-layout.dev/layouts/
- Every Layout, Switcher: https://every-layout.dev/layouts/switcher/
- web.dev, "Ten modern layouts in one line of CSS" (RAM, pancake, sidebar): https://web.dev/articles/one-line-layouts
- MDN, container queries: https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_containment/Container_queries
- Tailwind CSS, responsive design (breakpoints, container queries): https://tailwindcss.com/docs/responsive-design
- Android Developers, window size classes (Material 3): https://developer.android.com/develop/ui/compose/layouts/adaptive/use-window-size-classes
- Material 3, window size classes: https://m3.material.io/foundations/layout/applying-layout/window-size-classes (404 no fetch; valores confirmados pela página do Android acima)
- WCAG 2.2, 1.4.10 Reflow (Understanding): https://www.w3.org/WAI/WCAG22/Understanding/reflow.html
- WCAG 2.2, 2.5.8 Target Size (Minimum): https://www.w3.org/WAI/WCAG22/Understanding/target-size-minimum.html
- Adrian Roselli (especialista em acessibilidade, artigo mantido com atualizações), "Tables, CSS Display Properties, and ARIA": https://adrianroselli.com/2018/02/tables-css-display-properties-and-aria.html
- Chakra UI v3 (condições de container query): https://chakra-ui.com/docs/theming/overview

Código do Germanio lido: `runtime/servidor/paginas.go` (`layoutTpl`, `tableTpl`, `searchTpl`,
`detailTpl`), `runtime/servidor/renderizador.go` (sidebar fixa, `@media(max-width:768px)`),
`runtime/servidor/renderizador_declarativo.go` (`@media (max-width: 960px)`).

## 1. Três formas de responder ao espaço

| Forma | Pergunta que responde | Onde decide | Fonte |
| --- | --- | --- | --- |
| Media query / breakpoints | qual é a largura da **janela**? | página inteira | Tailwind, M3 window classes |
| Container query | qual é a largura do **pai** deste componente? | cada componente | MDN, Tailwind `@container`, Chakra `_cqSm` |
| Layout intrínseco | quanto conteúdo cabe? (o navegador decide) | o algoritmo de layout | Every Layout, web.dev |

### 1.1 Breakpoints de referência

- Tailwind (mobile-first; utilitário sem prefixo vale para todos os tamanhos): `sm` 40rem, `md`
  48rem, `lg` 64rem, `xl` 80rem, `2xl` 96rem (tailwindcss.com/docs/responsive-design).
- Material 3 / Android, classes de janela por largura: compacta < 600dp, média 600–839,
  expandida 840–1199, grande 1200–1599, extragrande ≥ 1600; compacta cobre 99.96% dos telefones
  em retrato (developer.android.com, use-window-size-classes).

As duas escalas discordam em números e concordam no conceito: **poucas classes nomeadas**, e o
layout muda de estrutura (navegação, número de painéis) entre elas, não a cada pixel.

### 1.2 Container queries

`container-type: inline-size` declara um contexto; `@container (width > 700px)` aplica estilos
pelo tamanho do pai; há containers nomeados e unidades `cqi`, `cqw`… (MDN). Tailwind expõe
`@container` e variantes `@sm`, `@md`, `@max-md`, "com base no tamanho de um elemento pai em
vez da janela inteira" (tailwindcss.com). É o mecanismo certo para um gerador: o mesmo nó
(uma tabela, um grupo de indicadores) se adapta igual numa página larga ou numa coluna estreita.

### 1.3 Algoritmos intrínsecos (Every Layout, web.dev)

Every Layout propõe 13 primitivas (Stack, Box, Center, Cluster, Sidebar, Switcher, Cover, Grid,
Frame, Reel, Imposter, Icon, Container) para "layouts robustos e responsivos sem `@media`"
(every-layout.dev/layouts). Mecanismos centrais:

- **Switcher**: `flex-basis: calc((limiar - 100%) * 999)`; abaixo do limiar o valor é enorme e
  cada item ocupa a linha; acima, é negativo e inválido, e os itens dividem a linha. A troca
  ocorre pela largura do **pai**, não da janela; um limite por quantidade (`:nth-last-child`)
  empilha tudo quando há itens demais (every-layout.dev/layouts/switcher).
- **Grid RAM**: `repeat(auto-fit, minmax(150px, 1fr))`: colunas com mínimo e distribuição
  automática; `auto-fill` evita esticar (web.dev/articles/one-line-layouts).
- **Sidebar**: uma parte estreita e uma larga lado a lado, empilhando quando a larga ficaria
  abaixo de uma proporção; **Cluster**: itens que quebram como palavras; **Stack**: espaçamento
  vertical uniforme; **Reel**: rolagem horizontal intencional.

## 2. Reflow, tabelas e alvos

- 1.4.10 (AA): sem rolagem em duas dimensões a 320px CSS; **tabelas de dados e grades são
  exceção** ("têm relação bidimensional entre cabeçalhos e células"), mas cada célula deve
  refluir, e o que acompanha a tabela (títulos, pesquisa, paginação) não é exceção
  (Understanding 1.4.10).
- Transformar a tabela em cartões com `display` do CSS: Chrome 80+ não perde mais a semântica de
  tabela com flex/grid/inline-block; Firefox corrigiu exceto `display: contents`; Safari só no 17.
  O autor recomenda restaurar roles de tabela por ARIA quando se muda `display`, e testar com
  leitores de tela (adrianroselli.com, artigo com atualizações; fonte não oficial).
- 2.5.8 (AA): alvos de pelo menos 24×24 CSS px, com exceções de espaçamento, equivalente,
  inline, controle do agente e essencial (Understanding 2.5.8). No toque, a densidade não pode
  descer abaixo disso.

## 3. O que o Germanio faz hoje (verificado)

- Páginas de intenção (`layoutTpl`): `meta viewport` presente; `main` com `max-width:1100px`;
  cabeçalho e navegação com `flex-wrap` (um Cluster de fato); pesquisa com `flex-wrap` e
  `min-width:140px` por campo; formulário com `max-width:640px`; tabela dentro de `.tabela` com
  `overflow-x:auto` e `white-space:nowrap` nas células; uma única media query
  (`max-width:640px`) que só reduz espaçamentos. O detalhe (`dl`) é grade
  `max-content 1fr` com `overflow-wrap:anywhere` no valor.
  Resultado: no celular a tabela rola horizontalmente (permitido pela exceção de 1.4.10), mas
  `nowrap` impede que células longas refluam, e a navegação do topo cresce sem limite quando há
  muitas páginas.
- SPA legado: sidebar fixa (`ml-64`) escondida por `@media(max-width:768px)`; utilitários da
  Tailwind via Play CDN. Renderizador declarativo: uma media query de 960px.
- O autor não escreve nenhuma media query hoje: a propriedade desejada já existe; o que falta é
  **adaptação estrutural** (tabela → cartões, navegação → disclosure) e cobertura sistemática.

## 4. Proposta: responsividade como propriedade do nó semântico

O autor nunca escreve breakpoints. Cada nó da árvore semântica (ver `COMPONENTS.md` §4) tem um
**algoritmo de layout fixo** e, quando precisa mudar de estrutura, uma **container query**
interna com limiares que são tokens do tema (em rem), não sintaxe.

| Nó | Primitiva | Comportamento derivado |
| --- | --- | --- |
| Page (shell) | Sidebar (navegação + conteúdo) ou topo com Cluster | navegação lateral em janela larga; em janela estreita, um botão disclosure (APG) que abre a lista; é o único lugar com media query (classe de janela) |
| Header | Cluster | título e ações quebram de linha; ações viram coluna em container estreito |
| Actions | Cluster → Stack | botões com alvo ≥ 24px (44px no modo toque, a decidir) |
| Filters | Switcher | lado a lado quando cabem; empilham abaixo do limiar; com muitos filtros, um disclosure "Filtros (3)" |
| Indicators | Grid RAM (`auto-fit`, `minmax`) | 1 a N colunas sem breakpoint |
| Table | container query | largo: tabela; estreito: **cartões** (ver 4.1) |
| Form | Stack; pares curtos (cidade/UF) em Switcher | uma coluna no estreito; largura máxima legível |
| Detail (`dl`) | Switcher por par | rótulo ao lado do valor no largo; rótulo sobre o valor no estreito |
| EmptyState | Center + Stack | centrado, texto com medida máxima |
| Chart | Frame (proporção) | reduz proporcionalmente; a tabela alternativa segue as regras da Table |

### 4.1 Tabela que vira cartões, de forma determinística

O Germanio sabe mais que um framework: conhece o **campo-título** do registro (`titleOf`), os
campos obrigatórios, o estado e as colunas escolhidas. Então a versão estreita pode ser derivada
sem pedir nada ao autor:

1. Prioridade das colunas: título (link para o registro) > estado > campos obrigatórios na
   ordem declarada > demais. A ordem é determinística e visível em `ge explain`.
2. Container estreito: cada linha vira um cartão com o título e as N primeiras colunas por
   prioridade como pares rótulo/valor; as demais ficam no detalhe (já existe, a um toque).
3. Marcação: manter `<table>` e restaurar os roles de tabela ao mudar `display`, ou emitir uma
   lista de cartões (`<ul>` + `<dl>`) quando o container for estreito; a escolha exige testes com
   leitores de tela (INVESTIGAR). A rolagem horizontal atual continua conforme como fallback,
   desde que a região rolável tenha nome e seja focável por teclado (hoje `.tabela`, em
   `tableTpl`, não tem `tabindex` nem nome).
4. Remover `white-space:nowrap` das células de texto livre; mantê-lo só em valores curtos
   (datas, números, selos).

### 4.2 Desktop, tablet e celular por padrão

- Uma única media query estrutural (o shell da página) por classe de janela, com 2 ou 3 classes
  (compacta/média/expandida, como o M3); todo o resto por container query e layout intrínseco.
- Limiar de cada primitiva = token de tema (`DESIGN_SYSTEMS.md`), afetado por `densidade`.
- Mobile-first no CSS gerado (Tailwind): o estilo sem condição é o estreito.
- Testes: renderizar cada página de exemplo em 320, 768 e 1280px CSS e verificar ausência de
  rolagem horizontal fora das tabelas (1.4.10) e alvos ≥ 24px (2.5.8).

## Para o Germanio

**ADOTAR**

- Layout intrínseco (Stack, Cluster, Switcher, Grid `auto-fit`) como implementação de cada nó.
  Resolve: responsividade sem media queries do autor e sem crescer o CSS de `layoutTpl`.
- Container queries dentro dos componentes do runtime. Resolve: o mesmo nó em colunas de larguras
  diferentes (dashboard, detalhe com coleções) sem breakpoints por página.
- Tabela → cartões com prioridade de colunas derivada do domínio. Resolve: `.tabela` com
  `nowrap` e rolagem horizontal no celular (`paginas.go`).
- Navegação do shell como disclosure em janela estreita. Resolve: `header nav` que cresce sem
  limite com muitas páginas.

**ADAPTAR**

- Classes de janela do M3 reduzidas a 2 ou 3, só para o shell. Resolve: `@media` espalhadas em
  três renderizadores com limiares diferentes (640, 768, 960).
- Limiares como tokens (rem) afetados por `densidade`. Resolve: números mágicos no CSS gerado.
- A exceção de tabelas do 1.4.10 como fallback, não como destino.

**EVITAR**

- Palavras de breakpoint ou de dispositivo no nível padrão (`no celular mostre…`): o autor
  descreve o que aparece; a adaptação é do runtime.
- Esconder conteúdo no celular sem caminho alternativo (a coluna omitida precisa existir no
  detalhe).
- Depender da Tailwind em runtime para variantes responsivas (Play CDN não é para produção).

**INVESTIGAR**

- Marcação da tabela estreita: `display` + roles de tabela versus lista de cartões; testar com
  NVDA, VoiceOver e TalkBack.
- Se o autor precisa de alguma forma de prioridade explícita de colunas (hoje derivada), e se isso
  cabe na seção `colunas` pela ordem, sem palavra nova.
- Densidade de toque (44px) versus 24px mínimo do WCAG para o modo compacto.
- Comportamento do Switcher de filtros com muitos campos (limite por quantidade).
