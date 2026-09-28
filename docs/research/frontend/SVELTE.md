# Svelte e SvelteKit: declaração, compiler e DOM eficiente

Data: 2026-09-28
Status: pesquisa concluída; nada implementado a partir dela.

## Fontes consultadas

- Repositório: https://github.com/sveltejs/svelte (diretórios `packages/svelte/src/compiler/phases`, `phases/2-analyze`, `phases/2-analyze/visitors/shared`, `phases/3-transform`, arquivo `packages/svelte/src/compiler/index.js`)
- SvelteKit: https://github.com/sveltejs/kit
- "Introducing runes": https://svelte.dev/blog/runes
- "Svelte 5 is alive": https://svelte.dev/blog/svelte-5-is-alive
- "Virtual DOM is pure overhead" (Rich Harris): https://svelte.dev/blog/virtual-dom-is-pure-overhead
- `$state`: https://svelte.dev/docs/svelte/$state
- `$derived`: https://svelte.dev/docs/svelte/$derived
- `$effect`: https://svelte.dev/docs/svelte/$effect
- Estilos com escopo: https://svelte.dev/docs/svelte/scoped-styles
- Warnings do compiler: https://svelte.dev/docs/svelte/compiler-warnings
- API do compiler: https://svelte.dev/docs/svelte/svelte-compiler
- SvelteKit, opções de página (ssr/csr/prerender): https://svelte.dev/docs/kit/page-options
- SvelteKit, `load` e invalidação: https://svelte.dev/docs/kit/load
- SvelteKit, form actions: https://svelte.dev/docs/kit/form-actions
- Language tools: https://github.com/sveltejs/language-tools
- Marcadores de hydration: https://github.com/sveltejs/svelte/issues/15200

Código do Germanio lido: `runtime/servidor/paginas.go`, `runtime/servidor/websocket.go`,
`runtime/servidor/servidor.go`, `runtime/servidor/transacao.go`, `runtime/servidor/eventos.go`,
`compiler/ast/intencao.go` (`PageDecl`), `compiler/ast/autorizacao.go` (`App`, `Entity`).

## 1. A tese do Svelte

Rich Harris formula o Svelte como "um compiler que sabe em *build time* como as coisas podem
mudar na sua aplicação, em vez de esperar para fazer o trabalho em *run time*"
(https://svelte.dev/blog/virtual-dom-is-pure-overhead). O argumento contra o Virtual DOM: o
diff tem três passos (a estrutura bate? quais atributos mudaram? atualizar o DOM real) e só o
terceiro tem valor quando a estrutura da aplicação não muda; o resto é trabalho a mais, feito
além das mudanças reais no DOM. O compiler, sabendo o que é estático e o que é dinâmico, emite
diretamente algo como "atribuir o texto deste nó" quando um valor muda.

A lição que interessa ao Germanio não é "sem Virtual DOM"; é a divisão estático/dinâmico
conhecida antes da execução. No Germanio essa divisão é ainda mais forte: a página não tem
código arbitrário, então o conjunto de dados que cada parte lê é conhecido inteiramente pelo
compiler (ver REACTIVITY.md).

## 2. O compiler em fases (confirmado no Svelte 5)

`packages/svelte/src/compiler/phases` contém `1-parse`, `2-analyze` e `3-transform`, além de
`scope.js`, `bindings.js`, `css.js`, `nodes.js`, `patterns.js`
(https://github.com/sveltejs/svelte/tree/main/packages/svelte/src/compiler/phases). A grafia
no repositório é "analyze", não "analyse".

`compile()` em `packages/svelte/src/compiler/index.js`: remove o BOM, chama `_parse(source)`,
remove nós de TypeScript, chama `analyze_component(parsed, source, options)`, depois
`transform_component(analysis, source, options)` e converte a AST para a forma pública com
`to_public_ast` (https://github.com/sveltejs/svelte/blob/main/packages/svelte/src/compiler/index.js).
O resultado traz `js`, `css`, `warnings` (cada um com `code`, `message`, `start`/`end`) e
`metadata` (https://svelte.dev/docs/svelte/svelte-compiler).

- **Parse**: produz uma AST do template, do `<script>` (ESTree) e do `<style>` (CSS). Existe
  uma AST "legacy" e uma "modern" (`parse(source, { modern: true })`), com a moderna prevista
  como padrão no Svelte 6 (https://svelte.dev/docs/svelte/svelte-compiler). Duas ASTs
  públicas convivendo é custo de compatibilidade que eles aceitaram.
- **Analyze**: constrói escopos e bindings (quem declara, quem lê, quem reatribui cada
  variável), resolve runes, analisa o CSS contra o template e emite diagnostics. Os visitors
  compartilhados ficam em `2-analyze/visitors/shared` e incluem uma pasta `a11y`
  (https://github.com/sveltejs/svelte/tree/main/packages/svelte/src/compiler/phases/2-analyze/visitors/shared).
  A acessibilidade é, portanto, parte da análise semântica, não um linter externo.
- **Transform**: `3-transform` tem `client`, `server`, `css` e `shared`
  (https://github.com/sveltejs/svelte/tree/main/packages/svelte/src/compiler/phases/3-transform).
  A mesma análise gera dois alvos: código de DOM para o navegador e código de concatenação de
  strings para o servidor (opção `generate: 'client' | 'server'`,
  https://svelte.dev/docs/svelte/svelte-compiler).

Paralelo com o Germanio: `hierarquia.go` (layout) e o parser correspondem ao parse;
`resolver.go` (Intent → App) corresponde ao analyze; o que falta é um "transform" de página
explícito, com dois alvos (HTML do servidor e um pequeno programa de atualização no cliente)
derivados da mesma análise. Hoje `paginas.go` mistura análise (quais campos, quais ações) com
renderização, a cada requisição.

## 3. Reatividade: de `$:` às runes

### Svelte 3/4

Reatividade por atribuição: `let` no topo do componente era estado, e `$:` marcava uma
declaração a reexecutar. As dependências eram determinadas pelo compiler, lendo o texto. O post
oficial lista os problemas (https://svelte.dev/blog/runes):

1. As dependências de `$:` são as que aparecem textualmente na declaração, fixadas na
   compilação; ao refatorar (mover uma leitura para dentro de uma função), a dependência some
   sem aviso.
2. A heurística só vale para `let` no topo do componente; o mesmo código fora do componente
   (num módulo) deixa de ser reativo, o que impede extrair lógica.
3. Conceitos demais: stores, o prefixo `$`, `export let`, funções de ciclo de vida, `$$props`.
4. Granularidade: no Svelte 4 mudar uma propriedade de um objeto invalidava o objeto inteiro
   (https://svelte.dev/blog/svelte-5-is-alive).

### Svelte 5

As runes (`$state`, `$derived`, `$effect`, `$props`) são sintaxe reconhecida pelo compiler, mas
o rastreamento de dependências passou para o runtime: "signals", que "são essencialmente o que o
Knockout fazia em 2010", como detalhe de implementação, não como API
(https://svelte.dev/blog/runes).

- `$state` de objeto ou array cria um proxy profundo; ler e escrever propriedades é rastreado,
  inclusive `push`; `$state.raw` evita o proxy para coleções grandes que só são substituídas
  (https://svelte.dev/docs/svelte/$state). A desestruturação quebra a reatividade (vira cópia).
- `$derived`: depende de tudo o que é lido de forma síncrona durante a avaliação; modelo
  push-pull (a notificação é empurrada na hora, o recálculo acontece só quando alguém lê); se o
  novo valor é idêntico ao anterior, os dependentes não são atualizados
  (https://svelte.dev/docs/svelte/$derived).
- `$effect`: roda depois de montar e, em microtask, depois das mudanças, em lote; só depende do
  que leu na última execução. A documentação o chama de "escape hatch" e manda usar `$derived`
  em vez de sincronizar estado com efeito (https://svelte.dev/docs/svelte/$effect).
- Limite reconhecido: o compiler opera um arquivo por vez, então estado exportado de módulos
  `.svelte.js` não pode ser reatribuído diretamente (https://svelte.dev/docs/svelte/$state).

**Leitura para o Germanio.** O Svelte abandonou a análise estática de dependências porque, numa
linguagem de uso geral (JavaScript), o texto não determina as leituras: funções, módulos e
condições escondem-nas. O Germanio está na situação oposta: a página só pode ler dados
declarados, por construções fechadas (`mostre projetos`, um indicador `total de clientes`),
sem funções arbitrárias no meio. Por isso a análise estática, que falhou no Svelte, é viável e
exata no Germanio, desde que o Germanio **não** abra a porta para expressões arbitrárias em
páginas. Isso é uma restrição de design, não um detalhe: cada escape de código em página
reintroduz o problema que levou o Svelte às runes.

A segunda lição: os signals ficaram "por baixo". O usuário do Germanio também não deve ver
signals, stores, efeitos ou invalidações; a reatividade é mecanismo do runtime.

## 4. Geração de DOM direta e hydration

O cliente compilado clona templates HTML estáticos e liga só os pontos dinâmicos a efeitos que
escrevem no nó exato (atributo, texto). Na hydration, o Svelte 5 reaproveita o DOM vindo do
servidor e usa comentários como marcadores de fronteira de blocos de controle (`<!--[-->`,
`<!--]-->`); há uma issue registrando que os marcadores aparecem até quando o componente não
hidrata (https://github.com/sveltejs/svelte/issues/15200). Detalhes do algoritmo de hydration
além disso: (não verificado nesta pesquisa).

No SvelteKit, por padrão `ssr = true` e `csr = true`: o servidor renderiza HTML e o navegador
"renderiza o componente de novo para torná-lo interativo", que é a hydration; `csr = false`
entrega a página sem JavaScript; `prerender` gera HTML estático, desde que "quaisquer dois
usuários que acessem a página recebam o mesmo conteúdo" (https://svelte.dev/docs/kit/page-options).

Custo intrínseco da hydration: o cliente precisa reexecutar a lógica de renderização (ou pelo
menos percorrer o DOM) e receber os dados serializados de novo. Para o Germanio, cuja interface
já é renderizada no servidor com permissões aplicadas lá (`paginas.go` chama a API interna com
o cookie da pessoa), hydration de componentes é desnecessária para o caso comum. O que falta é
atualizar pedaços da página quando os dados mudam (seção 7 e REACTIVITY.md).

## 5. CSS com escopo

O Svelte adiciona uma classe derivada do hash do CSS do componente (`svelte-123xyz`) aos
elementos afetados; cada seletor ganha especificidade 0-1-0, e ocorrências seguintes usam
`:where(...)` para não acumular especificidade; `@keyframes` também recebem escopo
(https://svelte.dev/docs/svelte/scoped-styles). A análise do CSS contra o template acontece
na fase analyze (pasta `2-analyze/css`), o que permite avisar sobre seletores sem uso (código
`css_unused_selector`, na lista de warnings de CSS de https://svelte.dev/docs/svelte/compiler-warnings;
nome exato do código: não verificado).

Para o Germanio: o usuário não escreve CSS no nível padrão, então escopo por componente não é
o problema. O problema análogo é garantir que blocos e temas não vazem estilos entre si; a
lição útil é que o CSS é gerado e verificado pelo compiler contra a estrutura conhecida, e
que só se emite o CSS dos blocos realmente usados na aplicação.

## 6. Componentes, eventos e estado

No Svelte 5, handlers de evento "são apenas props como quaisquer outras" e `{#snippet}`
substitui slots (https://svelte.dev/blog/svelte-5-is-alive): menos conceitos, mais
uniformidade. A direção (colapsar mecanismos especiais em um mecanismo geral) é a mesma do
princípio de subtração do Germanio. Mas componentes com props, slots e eventos são vocabulário
de programador; no Germanio o componente é extraído só quando necessário (progressive
disclosure), e os "eventos" da interface são as ações do domínio (`criar`, `editar`, `excluir`,
transições de estado), cuja permissão já está no `ast.App`.

## 7. SvelteKit: dados, invalidação e formulários

- `load` rastreia automaticamente as dependências que lê: `params`, `url`, `parent()` e cada
  `fetch()`; reexecuta só as funções afetadas numa navegação; `depends(chave)` +
  `invalidate(chave)` cobrem dependências que o framework não vê; `invalidateAll()` reexecuta
  tudo (https://svelte.dev/docs/kit/load).
- Form actions funcionam sem JavaScript ("o JavaScript do cliente é opcional"); `use:enhance`
  intercepta o envio e evita a recarga; depois de uma ação bem-sucedida o SvelteKit chama
  `invalidateAll`; erros de validação voltam por `fail(status, dados)` sem redirecionar
  (https://svelte.dev/docs/kit/form-actions).

O Germanio já faz o equivalente "sem JavaScript": `paginas.go` recebe o POST do formulário,
chama a API e redireciona com 303 para `?ok=` ou `?erro=` (padrão Post/Redirect/Get). Duas
lições diretas:

1. **Invalidação grossa depois da própria mutação é aceitável.** O SvelteKit, com todo o seu
   rastreamento, depois de uma ação invalida tudo. Para quem fez a mudança, recarregar a
   página (que o PRG já faz) é correto e simples.
2. **O erro de validação não deveria perder o que a pessoa digitou.** O Germanio redireciona
   com `?erro=` na query; o SvelteKit devolve 400/422 com os dados do formulário. Isso é uma
   lacuna de formulário (estado de envio, erro por campo), não de reatividade.

## 8. Diagnostics e acessibilidade no compiler

A lista oficial tem 40+ warnings `a11y_*`, por exemplo `a11y_missing_attribute`,
`a11y_label_has_associated_control`, `a11y_click_events_have_key_events`,
`a11y_no_static_element_interactions`, `a11y_role_has_required_aria_props`,
`a11y_media_has_caption`, `a11y_autofocus`, `a11y_positive_tabindex`
(https://svelte.dev/docs/svelte/compiler-warnings). São códigos estáveis, suprimíveis com o
comentário `svelte-ignore <código>`, e existem porque o autor escreve HTML à mão e pode errar.

No Germanio o autor não escreve HTML: a maioria dessas regras vira **invariante do gerador**,
não aviso ao usuário. Um formulário derivado da entidade sempre tem `<label for>` ligado ao
campo; uma ação é sempre `<button>` ou `<a>`, nunca `<div onclick>`; toda imagem de upload
tem texto alternativo derivado do título do registro. A lista do Svelte serve como checklist
de testes do renderizador (cada código vira um teste que falharia se o HTML gerado violasse a
regra), e só sobram como diagnostics ao usuário os casos em que falta informação de intenção
(uma imagem sem descrição possível, uma página sem título).

## 9. Tooling

`sveltejs/language-tools` contém `svelte-language-server` (LSP), `svelte-check` (CLI de
diagnostics e tipos), `svelte2tsx` (converte `.svelte` em TSX para o TypeScript checar), a
extensão `svelte-vscode` e um plugin de TypeScript; a base foi inspirada no Vetur do Vue
(https://github.com/sveltejs/language-tools). O ponto arquitetural: `svelte-check` e o LSP
chamam o próprio compiler para os diagnostics do Svelte, não uma reimplementação; o que não é
Svelte (tipos) é delegado ao TypeScript por tradução. O Germanio já tem o risco inverso
registrado: gramática TextMate gerada por script, formatter e parser como "parsers"
separados. Um `ge check` e um futuro LSP devem chamar o mesmo `parser` + `resolver`.

## 10. O que o Germanio aprende com DECLARAÇÃO → COMPILER → DOM EFICIENTE

1. A eficiência vem de saber, antes da execução, o que é estático e o que depende de quais
   dados. No Svelte isso é parcial (JavaScript esconde leituras); no Germanio pode ser total,
   se as páginas continuarem fechadas a código arbitrário.
2. A análise é uma fase própria, com um produto (o "grafo de leitura" da página), consumida
   por vários alvos (HTML do servidor, atualização no cliente, `ge explain`, diagnostics).
3. O mecanismo reativo fica escondido; a pessoa descreve o que aparece.
4. Os mesmos fatos que geram a interface geram as garantias (a11y, labels) como invariantes.
5. Não é necessário um framework de componentes no cliente para ter atualização eficiente: o
   servidor já sabe o que cada região mostra.

## Para o Germanio

### ADOTAR

- **Fase de análise de página separada da renderização.** Problema: `paginas.go` decide
  campos, ações e dados a cada requisição, dentro do renderizador; não existe um artefato que
  diga "esta página lê projetos (lista, filtrada pela visibilidade) e membros (contagem)".
  Solução: um passo, depois do `resolver.go`, que produza para cada `PageDecl` a lista de
  regiões e as entidades/agregados que cada uma lê; `paginas.go` e o futuro mecanismo reativo
  consomem esse resultado; `ge explain` o mostra. Afeta `compiler/ast/intencao.go`
  (`PageDecl`), um novo arquivo de análise em `compiler/`, `runtime/servidor/paginas.go`.
  Fonte: fases parse/analyze/transform do Svelte.
- **Regras de acessibilidade do Svelte como testes do renderizador.** Problema: a11y "padrão"
  sem verificação é promessa. Solução: um teste que percorre o HTML gerado para cada tipo de
  página e falha nos equivalentes de `a11y_label_has_associated_control`,
  `a11y_missing_attribute`, `a11y_no_static_element_interactions`. Afeta testes de
  `runtime/servidor/`. Fonte: https://svelte.dev/docs/svelte/compiler-warnings.
- **Um só front-end para compilar, checar e editar.** Problema: vários "parsers" (TextMate,
  formatter, parser). Solução: `ge check` e o futuro LSP chamam o parser e o resolver reais,
  como `svelte-check` chama o compiler. Afeta `cli/`, `tooling/`, `vscode-germanio/`.
  Fonte: https://github.com/sveltejs/language-tools.

### ADAPTAR

- **Dois alvos da mesma análise (server e client).** No Svelte, `generate: 'server' |
  'client'`. No Germanio: HTML do servidor (já existe) + um pequeno script genérico, igual
  para todas as aplicações, que só sabe substituir regiões marcadas quando o servidor avisa.
  Não gerar JavaScript por aplicação. Afeta `runtime/servidor/paginas.go` (marcar regiões com
  identificadores estáveis). Problema resolvido: páginas de intenção hoje não se atualizam
  sozinhas.
- **Invalidação grossa depois de ação própria (SvelteKit `invalidateAll`).** Manter o PRG
  atual para quem fez a mudança; a atualização fina só é necessária para os outros
  espectadores. Fonte: https://svelte.dev/docs/kit/form-actions.
- **Retorno de erro de formulário sem perder o digitado** (`fail(422, dados)`). Problema:
  `paginas.go` redireciona com `?erro=` e o formulário volta vazio. Afeta `paginas.go`
  (`post`). Fonte: https://svelte.dev/docs/kit/form-actions.
- **CSS emitido só para blocos usados e verificado contra a estrutura.** Útil quando a
  biblioteca de blocos existir; não é problema atual.

### EVITAR

- **Expressões arbitrárias em páginas.** O motivo de o Svelte trocar análise estática por
  signals em runtime foi que o JavaScript esconde leituras (https://svelte.dev/blog/runes).
  Se `.ge` aceitar código livre dentro de páginas, o Germanio perde a análise exata e terá de
  rastrear em runtime. Cálculos novos devem entrar como agregados declarados do domínio.
- **Hydration de componentes no cliente.** Custo de enviar dados duas vezes e reexecutar
  renderização, sem ganho para páginas renderizadas no servidor com permissões aplicadas lá.
- **Expor runes, stores ou efeitos ao usuário.** Mesmo o Svelte chama `$effect` de escape
  hatch (https://svelte.dev/docs/svelte/$effect).
- **Duas ASTs públicas (legacy e modern).** Custo de compatibilidade que o Svelte carrega;
  o Germanio deve ter uma única representação resolvida (`ast.App`).
- **Copiar a sintaxe `{#if}`, `{#each}`, `bind:`.** É HTML aumentado, o oposto de intenção.

### INVESTIGAR

- **Push-pull e igualdade de valor para agregados.** `$derived` só propaga se o valor mudou
  (https://svelte.dev/docs/svelte/$derived). No servidor do Germanio: recalcular o indicador
  e só avisar os clientes se o número mudou. Medir o custo de recálculo por mutação.
- **Pré-renderização de páginas públicas.** A regra do SvelteKit ("dois usuários quaisquer
  recebem o mesmo conteúdo") é derivável do `ast.App`: páginas sem login e sem dados
  privados. Investigar se há páginas assim nas aplicações do Germanio.
