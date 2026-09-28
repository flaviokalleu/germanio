# Vue: separação compiler/runtime, compilação guiada e o que isso significa para o Germanio

Data: 2026-09-28
Status: pesquisa (não normativa). A norma continua em `docs/INTENCAO.md`.

## Fontes consultadas

- Repositório: https://github.com/vuejs/core (lista de `packages/`: https://github.com/vuejs/core/tree/main/packages)
- Estrutura do projeto e regras de import: https://github.com/vuejs/core/blob/main/.github/contributing.md
- Guia "Rendering Mechanism": https://vuejs.org/guide/extras/rendering-mechanism.html
- Patch flags (fonte): https://github.com/vuejs/core/blob/main/packages/shared/src/patchFlags.ts
- Custom renderer API: https://vuejs.org/api/custom-renderer.html
- Vapor mode, notas das pré-versões 3.6 no branch `minor`: https://raw.githubusercontent.com/vuejs/core/minor/CHANGELOG.md
- Lista de releases: https://github.com/vuejs/core/releases
- Código do Germanio lido: `runtime/servidor/paginas.go`, `runtime/servidor/renderizador_declarativo.go`,
  `runtime/servidor/renderizador.go` (cabeçalho), `runtime/servidor/websocket.go`,
  `compiler/ast/intencao.go` (`PageDecl`), `examples/gitlab-foss/frontend/paginas.ge`.

## 1. A separação em pacotes

No branch `main`, `packages/` contém: `compiler-core`, `compiler-dom`, `compiler-sfc`,
`compiler-ssr`, `reactivity`, `runtime-core`, `runtime-dom`, `runtime-test`, `server-renderer`,
`shared`, `vue-compat`, `vue` (https://github.com/vuejs/core/tree/main/packages). O guia de
contribuição descreve cada um (https://github.com/vuejs/core/blob/main/.github/contributing.md):

| Pacote | Papel |
| --- | --- |
| `reactivity` | o sistema reativo; "can be used standalone as a framework-agnostic package" |
| `runtime-core` | "the platform-agnostic runtime core": VDOM, componentes, API para renderers próprios |
| `runtime-dom` | alvo navegador: APIs nativas do DOM, atributos, propriedades, eventos |
| `runtime-test` | runtime leve que produz árvores de objetos JS simples, para asserções nos testes |
| `compiler-core` | "the platform-agnostic compiler core", com plugins base |
| `compiler-dom` | plugins do compilador específicos do navegador |
| `compiler-sfc` | utilitários para compilar Single File Components |
| `compiler-ssr` | gera render functions otimizadas para SSR |
| `server-renderer` | a renderização no servidor |
| `shared` | utilitários independentes de ambiente, usados dos dois lados |

Duas regras do mesmo documento importam mais que a lista:

1. "Compiler packages should not import items from the runtime, and vice versa"; o que os dois
   lados precisam vai para `shared`.
2. As dependências fluem em uma direção: `runtime-dom → runtime-core → reactivity` e
   `compiler-sfc → compiler-dom → compiler-core`.

A lição arquitetural é a separação por dois eixos: **tempo** (compilação versus execução) e
**plataforma** (núcleo agnóstico versus alvo DOM/SSR). O "alvo" é um plugin fino em cima de
um núcleo que não sabe onde vai rodar.

## 2. Custom renderer API

`createRenderer()` recebe as operações de nó da plataforma (`createElement`, `insert`,
`remove`, `patchProp`, `createText`, `setText`, `setElementText`, `parentNode`,
`nextSibling`, `createComment`) e devolve `render` e `createApp`
(https://vuejs.org/api/custom-renderer.html). As implementações oficiais são `runtime-dom` e
`runtime-test`. O ponto relevante para o Germanio é o segundo: **o próprio Vue usa um alvo
"de teste" que produz uma árvore inspecionável**, em vez de testar HTML. Canvas, WebGL e
nativo são citados como possibilidades, mas o uso interno que se paga todo dia é o de testes.

## 3. Pipeline de renderização e a compilação guiada

O guia descreve três estágios: compilar templates em render functions que devolvem árvores de
VDOM; montar (o renderer executa a render function como efeito reativo); e fazer patch quando
as dependências mudam, comparando a árvore nova com a antiga
(https://vuejs.org/guide/extras/rendering-mechanism.html).

O diferencial do Vue é o "compiler-informed virtual DOM": como compilador e runtime são do
mesmo projeto, o compilador anota a saída com o que pode mudar:

- **Static caching**: nós estáticos são criados uma vez e reutilizados em toda re-renderização.
- **Patch flags**: cada vnode dinâmico carrega um bitmask do tipo de mudança possível
  (`TEXT=1`, `CLASS=2`, `STYLE=4`, `PROPS=8`, `FULL_PROPS=16`, `NEED_HYDRATION=32`,
  fragmentos estáveis/chaveados/sem chave etc.), e dois valores especiais: `CACHED=-1`
  ("hints hydration to skip subtrees") e `BAIL=-2` (sai do modo otimizado para render
  functions escritas à mão) (https://github.com/vuejs/core/blob/main/packages/shared/src/patchFlags.ts).
- **Tree flattening**: um "block" é uma parte do template com estrutura interna estável; só os
  descendentes dinâmicos entram num array achatado, e o patch percorre só esse array.
- **Efeito na hydration**: patch flags e blocks dão caminhos rápidos e "partial hydration at
  the template level".

O que isso ensina: **quem conhece a estrutura estática em tempo de compilação pode dizer ao
runtime exatamente o que é dinâmico**. O Vue precisa disso porque o seu runtime refaz a árvore
a cada mudança; o conhecimento estático serve para pular trabalho.

## 4. Vapor mode

As notas da 3.6.0-rc.1 no branch `minor` descrevem o Vapor como "a new compilation mode for
Vue Single-File Components (SFCs) with the goal of reducing baseline bundle size and improving
performance", "100% opt-in", suportando "a subset of existing Vue APIs"; ativado por
`<script setup vapor>` ou `<template vapor>`; sem Options API; sem recursos que dependem de
VNodes ou do proxy da instância; interoperável com VDOM por um plugin, mas recomendando
"distinct regions in an app where one rendering mode or the other is used, and avoiding mixed
nesting as much as possible" (https://raw.githubusercontent.com/vuejs/core/minor/CHANGELOG.md).
A última entrada vista é a 3.6.0-rc.9 (2026-09-18): **em release candidate, não estável**,
apesar de blogs afirmarem o contrário. Os pacotes `compiler-vapor`/`runtime-vapor` não
aparecem na listagem de `main` (https://github.com/vuejs/core/tree/main/packages); em que
diretórios eles vivem no `minor` ficou (não verificado).

Leitura: o Vapor é a admissão, pelo próprio Vue, de que com um compilador que conhece o
template, o VDOM vira custo opcional. Mesma conclusão do Svelte e do Solid (ver
`RENDERING.md`). O preço é um segundo modo de runtime e regras de convivência entre os dois.

## 5. Estado atual do Germanio, lido no código

- `registerPages` (`runtime/servidor/paginas.go:50`) registra `GET`/`POST /<slug>` por página
  de intenção. `serve` (`:607`) resolve a cadeia de registros, chama a própria API por
  `httptest` (`call`, `:85`) com o cookie e o CSRF da sessão, e concatena fragmentos
  `template.HTML` (título, busca, tabela, paginação, formulário, ações, filhos).
- Templates são strings Go (`layoutTpl`, `tableTpl`, `formTpl`…, `:1250-1310`) com
  `html/template`; CSS inline no layout; **nenhum `<script>`** nas páginas de intenção.
- Mutação: `post` (`:951`) confere o `_csrf`, mapeia `novo/editar/excluir/acao/<verbo>` para a
  API e responde `303` (Post/Redirect/Get) com `?ok=` ou `?erro=`.
- `ast.PageDecl` (`compiler/ast/intencao.go:152`) tem apenas `Name`, `Manage`, `Show`,
  `Permits`, `PerPage`, `Pos`. **Não há árvore de página**: a estrutura visual (título, busca,
  tabela, formulário, filhos) está codificada na ordem das chamadas dentro de `serve`.
- Existem outros dois renderizadores: `renderizador_declarativo.go` (páginas `CustomPage`,
  HTML/CSS por `strings.Builder` e fontes do Google) e `renderizador.go` (SPA legado; carrega
  `cdn.jsdelivr.net/npm/chart.js` e `cdn.tailwindcss.com`, linhas 33-34).

Portanto, hoje o Germanio já faz, sem nome, o equivalente a "compiler-ssr + server-renderer":
intenção → HTML no servidor. O que falta é o **meio**: um modelo visual explícito.

## 6. A pergunta: intenção visual → modelo visual → target Web → runtime Web?

Comparando com necessidades reais, não com o Vue:

| Camada proposta | Necessidade real hoje | Veredito |
| --- | --- | --- |
| Intenção visual (`página`, seções) | existe (`PageDecl`), mas pobre; G62 pede `topo`, `vazio`, `filtros`, `tabela`, indicadores | já existe; cresce com G62 |
| Modelo visual (árvore semântica tipada, sem HTML) | `serve` mistura busca de dados, autorização e emissão de HTML; não há como testar "a página mostra o botão Criar para developer" sem raspar HTML; `ge explain` não pode dizer por que um elemento aparece | **necessária**, é o ganho principal |
| Target Web (modelo → HTML) | um só alvo; hoje `html/template` | existe; vira função pura do modelo |
| Runtime Web (JS no navegador) | nenhuma página de intenção precisa de JS hoje; realtime e melhorias de navegação são futuras | mínimo e opcional |

O que o modelo visual resolve, concretamente:

1. **G62**: `topo › título/ações`, `conteúdo › filtros/tabela`, `vazio` são nós, não texto
   indentado. O parser produz a árvore; o resolver completa com o `ast.App` (colunas a partir
   dos campos, opções a partir das relações, ações a partir dos grants) e acusa referências
   inválidas (`colunas › emial`) em `ge check`, antes de qualquer requisição.
2. **Testes como o `runtime-test`**: afirmar sobre a árvore resolvida ("para `reporter`, a
   tabela de issues não tem a ação Excluir") em vez de HTML. É exatamente o uso que o Vue faz
   do seu renderer alternativo.
3. **`ge explain` de interface**: cada nó leva a origem (a frase do `.ge` ou a regra derivada:
   "coluna `estado` porque `issues começa aberta`"; "botão Excluir porque `maintainer pode
   excluir`").
4. **Separação de responsabilidades em `serve`**: resolver (o que aparece, por permissão) é
   um passo; buscar dados é outro; emitir HTML é o terceiro. Hoje estão intercalados.

O que **não** copiar:

- Um VDOM, patch flags por nó ou um runtime-core no navegador. O Vue precisa deles porque
  re-executa a árvore no cliente; o Germanio renderiza por requisição no servidor e não tem
  árvore no cliente para comparar.
- Um "custom renderer API" com interface de plataforma. Existe um alvo. Abstrair para um
  segundo alvo hipotético (nativo, terminal, PDF) antes de existir o segundo é complexidade
  especulativa; o modelo visual já deixa a porta aberta sem a interface.
- Dois modos de renderização convivendo (VDOM + Vapor). O próprio Vue recomenda evitar o
  aninhamento misto; o Germanio já tem três renderizadores e deveria convergir para um.
- A granularidade: o análogo útil de "o que é dinâmico" no Germanio é a **região** (a tabela
  de issues depende da entidade issue, filtros e permissões), não o atributo `class` de um nó.

## Para o Germanio

- **ADOTAR** a separação "núcleo agnóstico / alvo fino" em dois passos de Go: `ast.PageDecl`
  estendido (intenção) → modelo visual resolvido (árvore tipada de `Página`, `Topo`, `Ações`,
  `Filtros`, `Tabela`, `Formulário`, `Vazio`, cada nó com origem) → emissor `html/template`.
  Resolve: G62; `serve` (`runtime/servidor/paginas.go:607`) mistura dados, permissão e HTML;
  a ausência de diagnóstico estático para páginas. Afeta `compiler/ast/intencao.go`,
  `compiler/parser/hierarquia.go`, `compiler/parser/resolver.go`, `runtime/servidor/paginas.go`.
- **ADOTAR** a regra de direção de dependências do Vue ("compiler não importa runtime e vice
  versa"): o modelo visual vive no `compiler/ast` (ou num pacote próprio sem `net/http`); o
  emissor HTML vive no `runtime/servidor`. Resolve: acoplamento entre resolução e servidor que
  impediria `ge check`/`ge explain` de usar o modelo visual sem subir o servidor.
- **ADAPTAR** o `runtime-test`: testes normativos de página afirmam sobre a árvore resolvida
  por papel (developer, reporter, anônimo), não sobre HTML. Resolve: hoje só é possível testar
  páginas por HTML ou pela API. Afeta testes em `runtime/servidor/`.
- **ADAPTAR** a ideia de patch flags como **dependências por região**: o modelo visual sabe
  que a região "tabela de issues" lê a entidade issue com tais filtros. Isso é o que
  permitirá atualizar só essa região quando issue mudar (ver `RENDERING.md`), derivado e
  explicável, sem declarar reatividade no `.ge`. Resolve: "reatividade derivada do
  conhecimento de quais dados a página usa" (direção registrada no contexto de frontend).
- **EVITAR** VDOM, runtime de componentes no navegador e a interface `createRenderer`.
  Resolve: nenhuma necessidade atual; adicionaria um runtime JS a manter e a proteger.
- **EVITAR** manter três renderizadores divergentes (`paginas.go`, `renderizador_declarativo.go`,
  `renderizador.go`). A lição do Vapor é que modos convivendo exigem regras de fronteira;
  o caminho do Germanio é convergir para o emissor do modelo visual e marcar o SPA legado como
  legado. Resolve: estilos, segurança e acessibilidade divergentes entre os três.
- **INVESTIGAR** um segundo alvo (e-mail transacional, PDF de relatório, terminal) somente
  quando um exemplo real o pedir; o modelo visual é o ponto de extensão, sem interface de
  plataforma antecipada.
