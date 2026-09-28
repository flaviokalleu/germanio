# O frontend do Germanio: consolidação da pesquisa

Data: 2026-09-28
Status: **pesquisa, sem força normativa.** A norma continua em `docs/INTENCAO.md`. Tudo o que
aparece aqui como PROPOSTA exige uma **GEP** (Germanio Evolution Proposal, processo em
`docs/gep/README.md`, modelo em `docs/gep/0000-template.md`, aceito pela GEP 0001) antes de
mudar a norma, o parser ou o runtime.

## Fontes

Os estudos desta pasta, cada um com as suas fontes primárias no topo:
[SVELTE.md](SVELTE.md), [SOLID.md](SOLID.md), [REACTIVITY.md](REACTIVITY.md),
[VUE.md](VUE.md), [REACT.md](REACT.md), [RENDERING.md](RENDERING.md),
[SSR_HYDRATION.md](SSR_HYDRATION.md), [COMPONENTS.md](COMPONENTS.md),
[DESIGN_SYSTEMS.md](DESIGN_SYSTEMS.md), [ACCESSIBILITY.md](ACCESSIBILITY.md),
[RESPONSIVE.md](RESPONSIVE.md). Norma de eficiência: `docs/INTENCAO.md` › "Eficiência";
budgets de frontend: `docs/research/performance/BENCHMARKING.md` §12. As URLs abaixo são as
citadas nesses estudos; cada afirmação traz o arquivo de estudo entre parênteses.

Código do Germanio conferido para este documento: `runtime/servidor/paginas.go`,
`renderizador.go`, `renderizador_declarativo.go`, `websocket.go`, `servidor.go`,
`compiler/ast/ast.go`, `compiler/ast/intencao.go`, `compiler/parser/germanio.go`,
`GERMANIO_GAPS.md`. Conferência feita sobre `fc31daf` e revista sobre `d9676b9`, que
corrigiu o `/ws` (G65) e renumerou as lacunas G57–G60 duplicadas para G61–G64.

## 0. A questão e a resposta curta

"Como uma pessoa descreve a interface que deseja e o Germanio a transforma numa aplicação Web
profissional, responsiva, acessível e eficiente?"

Resposta que a pesquisa sustenta: a pessoa descreve **o que aparece** em termos do domínio
(`página Clientes` › `mostre clientes`); o Germanio já sabe, pelo `ast.App`, quais campos,
tipos, obrigatoriedades, relações, estados e permissões existem. A interface é uma **função
desse conhecimento** ("UI = f(dados do domínio, permissões, URL)", REACT.md), renderizada como
HTML completo no servidor, com melhorias progressivas genéricas por cima. Nada de
componente com estado, hook, VDOM, hydration ou CSS no nível padrão. O que os frameworks
precisam rastrear em execução (dependências, reatividade, memoização) o Germanio pode calcular
na compilação, porque as páginas não contêm código arbitrário (SVELTE.md §10, REACTIVITY.md).

## 1. Arquitetura recomendada

```text
intenção visual (.ge: página, mostre, permita, [seções propostas], [tema proposto])
   │  parser + resolver (um só front-end para compilar, checar, formatar e o LSP)
   ▼
modelo visual derivado de ast.App      ← árvore tipada: Page/Header/Actions/Filters/Table/
   │  (compiler; sem net/http)            Form/EmptyState…, cada nó com origem (arquivo:linha
   │                                      ou "padrão" + o fato do domínio que o gerou) e as
   │                                      dependências de dados de cada região
   ├──► ge explain / ge check (inclusive avisos a11y)   ← consomem o modelo, sem servidor
   ▼
HTML do servidor (runtime/servidor, html/template)     ← camada 0: completo e funcional sem JS;
   │                                                      permissões aplicadas por espectador
   ▼
melhorias progressivas que degradam para a camada 0:
   camada 1  CSS (tokens do tema, layout intrínseco, container queries, View Transitions)
   camada 2  um script genérico do core: troca de regiões por HTML do servidor, foco, anúncios
   camada 3  tempo real: aviso → rebusca autenticada da região → morph
   ilhas     só como escape técnico (gráfico, editor, mapa, arrastar), nunca como modelo
```

Regras de arquitetura (ADOTAR):

- **HTML do servidor é a camada obrigatória.** Formulários, links e paginação reais são o
  contrato; toda melhoria degrada para eles (RENDERING.md §5; htmx,
  https://htmx.org/essays/hypermedia-driven-applications/; Turbo,
  https://turbo.hotwired.dev/handbook/introduction).
- **Nenhum script do core reconstrói ou reconcilia DOM renderizado pelo servidor.** Scripts
  só trocam regiões por HTML já escapado pelo servidor, cuidam de foco e de anúncios. Isso
  elimina por construção hydration, divergência servidor/cliente e bundle por página
  (SSR_HYDRATION.md; custo da rehydration em https://web.dev/articles/rendering-on-the-web;
  requisito de saída idêntica em https://react.dev/reference/react-dom/client/hydrateRoot).
- **Sem VDOM e sem runtime de componentes no navegador.** O VDOM compensa o custo de
  reexecutar a UI no cliente (https://svelte.dev/blog/virtual-dom-is-pure-overhead); o
  Germanio não tem UI no cliente para reexecutar.
- **O script é o mesmo para todas as aplicações.** Nenhum JavaScript gerado por aplicação; as
  regiões e seus endereços vêm do modelo visual (SVELTE.md, ADAPTAR "dois alvos").

**Por que não separar em compiler-core / compiler-dom / compiler-ssr como o Vue.** O Vue tem
`compiler-core`, `compiler-dom`, `compiler-ssr`, `runtime-core`, `runtime-dom`,
`server-renderer` (https://github.com/vuejs/core/tree/main/packages) porque precisa emitir o
mesmo componente para dois alvos (render function no cliente e string no servidor) e aceitar
renderizadores de plataforma (https://vuejs.org/api/custom-renderer.html). O Germanio tem um
alvo (HTML do servidor) e nenhuma plataforma alternativa pedida por um exemplo real. O que se
transfere do Vue é a **regra de direção de dependências** ("compiler não importa runtime e
vice-versa", https://github.com/vuejs/core/blob/main/.github/contributing.md): o modelo visual
mora no `compiler/` sem `net/http`, o emissor HTML mora no `runtime/servidor`. Dois passos em
Go, não seis pacotes (VUE.md). Um segundo alvo (e-mail, PDF, terminal) fica em INVESTIGAR até
um exemplo real pedi-lo. Também não se transfere `createRenderer` nem o Vapor: a lição do
Vapor é que modos convivendo exigem regras de fronteira
(https://raw.githubusercontent.com/vuejs/core/minor/CHANGELOG.md); o Germanio já tem três
renderizadores divergentes e deve convergir para um.

## 2. Reatividade derivada do Knowledge Graph

Os frameworks rastreiam dependências porque o código do componente as esconde: o Svelte 5
trocou a análise estática por signals em execução justamente porque o JavaScript esconde
leituras (https://svelte.dev/blog/runes); o Solid rastreia dependências dinâmicas em
`signal.ts` (https://github.com/solidjs/solid/blob/main/packages/solid/src/reactive/signal.ts);
o SvelteKit precisa de `depends`/`invalidate` para o que não vê
(https://svelte.dev/docs/kit/load). Uma página do Germanio não tem código livre, então o
conjunto de leitura de cada região é **exato e calculável na compilação** (SVELTE.md,
SOLID.md, REACTIVITY.md).

Proposta (não implementada):

1. **Dependências por região na compilação.** Uma fase de análise depois do `resolver.go`
   produz, para cada região do modelo visual, o que ela lê: entidade, campos, filtros,
   referências por nome, membros, visibilidade, agregados e dependências temporais ("hoje").
   É o análogo estático do read set do Convex (https://stack.convex.dev/how-convex-works) e das
   patch flags do Vue (https://github.com/vuejs/core/blob/main/packages/shared/src/patchFlags.ts),
   por região e não por expressão. `ge explain` mostra o resultado.
2. **Aviso + rebusca com as permissões de quem vê.** Depois do commit (`afterCommit` em
   `runtime/servidor/transacao.go`, a partir de `emit` em `eventos.go`), o servidor avisa as
   conexões cujas regiões dependem do que mudou; cada navegador **rebusca a região com o
   próprio cookie** e aplica o HTML por morph. É o modelo "page refresh com morphing" do Turbo
   (https://turbo.hotwired.dev/handbook/page_refreshes; `after_commit`, debounce e
   `request_id` em
   https://github.com/hotwired/turbo-rails/blob/main/app/models/concerns/turbo/broadcastable.rb).
   As permissões ficam corretas por construção, porque o fragmento é renderizado pelo mesmo
   caminho que a página inteira. Agrupar avisos da mesma transação e de uma janela curta;
   não avisar o autor da mudança (ele já recebe o PRG); não propagar se o valor não mudou
   (`$derived`, https://svelte.dev/docs/svelte/$derived).
3. **Nada exposto ao usuário.** Nenhum signal, efeito, `invalidate`, chave de cache ou palavra
   "reativo" no `.ge` (REACTIVITY.md, EVITAR). O autor descreve o que aparece; o que atualiza é
   consequência.
4. **Expressões livres em páginas continuam proibidas.** São o que desligaria a análise exata;
   cálculos novos entram como agregados declarados do domínio (G62, indicadores).
5. **Pague só pelo que usar.** O hub e o script de tempo real só são iniciados e enviados
   quando uma página tem regiões atualizáveis e a decisão de ligá-los estiver tomada
   (`docs/INTENCAO.md` › Eficiência). Se o tempo real é automático para toda página ou
   declarado, é decisão de GEP.

**Requisito de segurança antes de qualquer tempo real.** Nenhuma região em tempo real pode
existir enquanto o canal enviar dados sem autorização por espectador. No commit conferido, o
`/ws` aceitava conexões sem sessão e a API legada difundia eventos de criar/atualizar/deletar/
restaurar a todos os sockets (`servidor.go:692`, `:726`, `:735`, `:1386`), com um hub em memória
que descarta mensagens em silêncio quando o buffer de 64 enche (`websocket.go:74`, `:120`).
**Esse problema está em correção pelo orquestrador**: o commit `d9676b9` (G65) passou a
exigir token no `/ws` quando a aplicação tem autenticação, recusa `Origin` de outro site e
reduz o aviso a tipo, modelo e id; continua aberto o G66 (o hub não tem salas nem
destinatários, descarta em silêncio com o buffer cheio e não atravessa processos). Condições mínimas para a camada
3: canal autenticado e com `Origin` verificada; o canal transporta **apenas avisos** (qual
região mudou), nunca registros; todo conteúdo passa pela rebusca autenticada; e, com mais de um
processo, o aviso atravessa processos pelo banco ou pela fila já usada por `emit` (SOLID.md,
INVESTIGAR). Defesa em profundidade adicional: `http.CrossOriginProtection`
(https://pkg.go.dev/net/http#CrossOriginProtection), ausente hoje (`grep` sem resultado em
`runtime/`), com Go 1.26.1 no `go.mod` (RENDERING.md).

## 3. Seções de página — PROPOSTA (exige GEP)

### 3.1 O que a norma diz hoje

`docs/INTENCAO.md` › "Página": `página Clientes` + `mostre clientes`, `permita` + ações e
`20 por página` **é** `crie página Clientes` com o mesmo corpo. A gramática do mesmo documento
usa `bloco_pagina = "página" Nome NL ABRE { secao_pagina } FECHA`, mas **não define
`secao_pagina`** (lacuna da especificação, não da implementação). "Pendências da sintaxe
hierárquica" diz que `topo`, `vazio`, `gráfico`, `lista` e indicadores "são direção, não
contrato implementado". A lacuna está registrada como **G62** em `GERMANIO_GAPS.md:76`
(registrada antes como G58, identificador que duplicava o de webhooks; renumerada em
`d9676b9`). O AST atual é
`PageDecl{Name, Manage, Show, Permits, PerPage, Pos}` (`compiler/ast/intencao.go:152`).

### 3.2 A proposta

Seções de página como **tabela fechada**, com a mesma técnica da tabela de seções de dados:
cada seção é um nó; linha fora da tabela é erro que lista as seções válidas; cada nó tem uma
frase plana equivalente e um teste de equivalência; cada seção escrita substitui só o seu slot;
a ausência de uma seção tem um padrão derivado do domínio (COMPONENTS.md §4). Critério para
uma palavra entrar na LINGUAGEM: significado de domínio (não visual), redutível a um nó da
árvore, e ausência com padrão derivável. `hero`, `pricing`, `navbar` não passam.

Árvore semântica (candidata):

```text
Page(nome)
├── Header          ← topo
│   ├── Title       ← título "…"             padrão: o nome da página
│   ├── Text        ← texto "…"              opcional
│   └── Actions     ← ações › verbo ["rótulo"]  padrão: ações de coleção permitidas (criar)
├── Filters         ← filtros › pesquisar | campo   padrão: de permita pesquisar/filtrar
├── Table | List    ← colunas › campos       padrão: columns(e); dado = o de `mostre`
├── Indicators      ← indicadores › …        depende da semântica de agregados (GEP própria)
├── Chart           ← gráfico …              idem
└── EmptyState      ← vazio › título, texto, ação   padrão: mensagem derivada + criar
```

Tabela fechada (candidata):

| Seção | Pai | Conteúdo | Validação contra o domínio | Frase plana (candidata) |
| --- | --- | --- | --- | --- |
| `topo` | página | `título`, `texto`, `ações` | — | — |
| `título "X"` | topo, vazio | um texto | — | `página P tem título "X"` |
| `texto "X"` | topo, vazio | um texto | — | `página P tem texto "X"` |
| `ações` | topo | um verbo por linha, rótulo opcional | o verbo existe e é de coleção | `página P oferece criar "Novo cliente"` |
| `filtros` | página | `pesquisar` ou um campo por linha | o campo existe no dado mostrado; `pesquisar` exige `permita pesquisar` no dado | `página P filtra por status` |
| `colunas` | página (ou `tabela`) | um campo por linha, na ordem | o campo existe; `privado`/`oculto` é erro | `página P mostra nome, email` |
| `vazio` | página | `título`, `texto`, `ação` verbo ["rótulo"] | a ação é permitida a alguém | `página P quando vazia diz "…"` |
| `indicadores` | página | agregados (`total de clientes`) | depende da GEP de agregados | — |

Exemplo, na forma proposta:

```ge
página Clientes
    mostre clientes
    topo
        título "Clientes"
        ações
            criar "Novo cliente"
    filtros
        pesquisar
        status
        cidade
    colunas
        nome
        email
    vazio
        título "Nenhum cliente"
        texto "Cadastre seu primeiro cliente."
        ação criar "Cadastrar cliente"
```

Três correções em relação ao esboço de direção:

1. **Determinismo das ações.** `ações` › `"Novo cliente"` não diz qual ação executa: um texto
   não é uma capability. A ação é um **verbo do domínio com rótulo opcional**: `criar "Novo
   cliente"`. O rótulo padrão é derivado ("Novo cliente" a partir do singular). O mesmo vale
   para `vazio` › `ação criar "Cadastrar cliente"` (COMPONENTS.md §4, decisão 1).
2. **Sem repetição.** `tabela` › `clientes` repete `mostre clientes`; quando a página mostra um
   só dado, a hierarquia já o fornece (`docs/INTENCAO.md` › "Como avaliar uma sintaxe",
   critério 3). Se `conteúdo` é necessário ou só profundidade extra fica em INVESTIGAR; o
   exemplo acima o omite.
3. **A página pede, o backend decide.** `ações` › `criar` só pede o botão; ele aparece apenas
   para quem tem o grant, como `available` já faz em `paginas.go:830`. Nada de `se papel é
   admin` na página.

Confronto com a norma atual, a resolver na GEP:

- **`ações` × `permita`.** Hoje a página diz `permita` › `criar`, `editar`, `excluir`. A
  proposta cria um segundo lugar para o verbo `criar`. Opções: (a) `topo › ações` só dá rótulo
  e posição a verbos já listados em `permita`, e um verbo em `ações` fora de `permita` é erro;
  (b) `ações` substitui `permita` para ações de coleção e `permita` fica para ações de linha;
  (c) `ações` implica `permita`. A regra de fusão exige que só uma delas seja adotada.
- **Nível 0 continua valendo.** `página Clientes` + `mostre clientes` sozinho deve produzir a
  árvore inteira pelos padrões; nenhuma seção é obrigatória.
- **Frase plana e `ge explain`.** Cada nó mostra se veio do `.ge` (arquivo:linha, caminho
  `Clientes › topo › ações`) ou do padrão (e de qual fato do domínio).
- **Profundidade.** `página › vazio › ação` são três níveis; `página › topo › ações › verbo`
  são quatro, o limite indicado pela norma para a profundidade hierárquica.

## 4. Formulários derivados do domínio

O formulário é uma projeção do `ast.Field`, não uma declaração. Da entidade vêm os campos (sem
`Hidden`, `System`, `Immutable`, segredo), o rótulo (o nome, com acento), `required`, o tipo do
controle, os limites e as opções de referências e enums. Da operação vêm os estados: vazio,
enviando, erro por campo, sucesso. Esse é exatamente o problema que o React resolve com
`useActionState` (https://react.dev/reference/react/useActionState) e o SvelteKit com
`fail(422, dados)` (https://svelte.dev/docs/kit/form-actions); no Germanio ele é derivado, sem
estado declarado (REACT.md, COMPONENTS.md §6).

Proposta:

- **Tabela única campo → controle, testada e mostrada por `ge explain`**: `telefone` → `tel`,
  `url` → `url`, `hora` → `time`, `data_hora` → `datetime-local`; `Min`/`Max` → `minlength`/
  `maxlength` (texto) ou `min`/`max` (número); `Format` → `pattern` só quando a expressão vale
  nos dois dialetos (RE2 ≠ JavaScript), senão só no servidor; `required` também na edição.
- **Erro sem perder o digitado.** Em erro de validação, o servidor responde 422 e
  **re-renderiza** o formulário com os valores e a mensagem de cada campo (`aria-describedby`,
  `aria-invalid`), com um resumo no topo que recebe o foco (WCAG 3.3.1 e 3.3.7,
  https://www.w3.org/TR/WCAG22/; divisão validação nativa × erro do servidor por campo em
  https://react-aria.adobe.com/forms). A API já devolve os erros por campo
  (`{"message": {"email": [...]}}`, `docs/INTENCAO.md` › Garantias automáticas); é a camada de
  página que os achata numa frase (seção "Divergências").
- **Sucesso** por redirect para o registro (PRG, como hoje), com o aviso vindo do servidor
  (por exemplo um aviso de uso único na sessão), não da query string, e anunciado por
  `role="status"` (4.1.3, https://www.w3.org/WAI/WCAG22/Understanding/status-messages.html).
- **Estado de envio** (botão desabilitado, `aria-busy`) só na camada 2; sem script, o envio
  normal do navegador já é o estado.
- **A autoridade é o servidor** (`banco.Validar`); a validação do navegador é conveniência.
  Duas mensagens diferentes para o mesmo erro (a do navegador e a de `mensagens em`) é
  INVESTIGAR (ACCESSIBILITY.md).

## 5. Permissões do domínio aplicadas à interface

A interface não decide quem pode: ela pergunta ao mesmo mecanismo que a API usa. Isso já é
verdade em `paginas.go` (`available`, `canCreateFor`, `ps.a.in.Can` em `serve`) e deve continuar
sendo a única via: nada de `se papel é admin` ou `if role` em páginas (`docs/INTENCAO.md` ›
Organização do projeto: "A página diz o que aparece; o backend decide quem pode").

Consequências para o modelo visual:

- Cada nó que representa uma ação carrega o verbo do domínio; o emissor o omite para quem não
  tem o grant. Os testes normativos de página afirmam sobre a árvore resolvida **por papel**
  (anônimo, reporter, developer), não sobre HTML (VUE.md, ADAPTAR `runtime-test`).
- Registro invisível é "não encontrado", nunca "proibido" (norma); uma região cujo dado a pessoa
  não pode ver some, mas um **erro interno** não pode sumir junto (ver o `continue` nas
  divergências). Falha por região, à maneira de uma error boundary
  (https://react.dev/reference/react/Component#catching-rendering-errors-with-an-error-boundary):
  a região mostra a falha explicada e o resto da página renderiza (REACT.md).
- Pesquisa e filtros mudam como se encontra, nunca quem vê (norma); o mesmo vale para o tempo
  real (seção 2).

## 6. Tema como tokens determinísticos

O que os design systems estudados têm em comum: **uma semente, um algoritmo, papéis em pares**.
Ant Design deriva Seed → Map → Alias (https://ant.design/docs/react/customize-theme); o
Material 3 gera o esquema inteiro de uma cor-fonte
(https://github.com/material-foundation/material-color-utilities/blob/main/concepts/dynamic_color_scheme.md);
shadcn e M3 tratam superfície e texto-sobre como par (https://ui.shadcn.com/docs/theming)
(DESIGN_SYSTEMS.md §1).

Proposta:

- **Cor semente, não paleta escrita à mão.** `destaque azul` escolhe a cor-fonte; o Germanio
  deriva papéis (superfície, texto-sobre, borda, foco, sucesso, erro) nos modos claro e escuro.
- **Contraste garantido por aritmética e verificado.** No espaço HCT, diferença de tom 40
  garante 3:1 e diferença 50 garante 4.5:1
  (https://github.com/material-foundation/material-color-utilities/blob/main/typescript/hct/hct.ts);
  cada par emitido é conferido ainda pela fórmula de contraste do WCAG 2.2
  (https://www.w3.org/TR/WCAG22/). Um teste de ouro cobre todo o `ColorName`. Texto sobre cor
  nunca é branco fixo; limiar 3:1 para texto (padrão do MUI,
  https://mui.com/material-ui/customization/palette/) é insuficiente; APCA não é conformidade
  WCAG 2.2.
- **Vocabulário fechado, com semântica verificável.** Só entram palavras cujo efeito se pode
  testar: `destaque` (nomes de `ColorName`), `cantos` (`retos`/`suaves`/`arredondados` → raio
  base), `densidade` (`compacta`/`normal`/`confortável` → passo de espaçamento e altura de
  controle), `modo` (`claro`/`escuro`/`automático`) e, candidata, `contraste alto`
  (o `contrastLevel` do M3 reduzido a duas palavras,
  https://github.com/material-foundation/material-color-utilities/blob/main/typescript/dynamiccolor/dynamic_scheme.ts).
  Palavra fora da tabela é erro que lista as válidas; o mesmo seed com dois valores é conflito
  com as duas origens (regra de Fusão). Hex, px e CSS só no nível técnico.
- **`estilo moderno` só depois de ter semântica formal.** Hoje nenhuma definição diz o que
  `elegante` muda; enquanto não disser (por exemplo como um conjunto fixo de seeds: variante de
  esquema, família tipográfica, raio, sombra), é palavra vaga e não entra no nível padrão
  (skill, seção 19b; DESIGN_SYSTEMS.md, INVESTIGAR). Presets novos, se existirem, são só seeds.
- **Tokens emitidos como CSS estático pequeno**, gerado pelo Germanio, sem framework CSS em
  execução (como o `zeroRuntime` da Ant). Export opcional para DTCG 2025.10
  (https://www.designtokens.org/tr/2025.10/format/) só para interoperar com ferramentas de
  design.

## 7. Acessibilidade por padrão

Princípio: HTML nativo primeiro, ARIA só como promessa cumprida ("No ARIA is better than bad
ARIA", https://www.w3.org/WAI/ARIA/apg/practices/read-me-first/; páginas com ARIA têm em média
mais erros detectados que as sem, WebAIM Million 2025, https://webaim.org/projects/million/2025)
(ACCESSIBILITY.md §1).

- **Invariantes do gerador, não avisos ao autor.** O autor não escreve HTML, então a maioria
  das regras `a11y_*` do Svelte (https://svelte.dev/docs/svelte/compiler-warnings) vira teste do
  renderizador: todo controle tem `<label>`; ação é `<button>` ou `<a>`; tabela tem `<caption>`
  e `scope`; toda região rolável tem nome e foco por teclado; link "pular para o conteúdo".
- **Labels vindos dos nomes dos campos.** A mesma fonte (`Field.Label`) serve ao `<label>`, ao
  `<th>`, ao `<dt>`, ao rótulo de filtro, à mensagem de erro e ao nome acessível de ações de
  linha ("Editar cliente Maria", a partir de `titleOf`). `placeholder` nunca é rótulo.
- **`<dialog>` com `showModal()`** para confirmar exclusão e formulários rápidos: foco, Esc,
  inércia do resto da página e retorno do foco vêm do navegador
  (https://html.spec.whatwg.org/multipage/interactive-elements.html#the-dialog-element;
  padrão em https://www.w3.org/WAI/ARIA/apg/patterns/dialog-modal/). Não reimplementar trap de
  foco. Navegação do site como disclosure, não `role="menu"`
  (https://www.w3.org/WAI/ARIA/apg/patterns/disclosure/examples/disclosure-navigation/).
- **Live regions fixas no layout**: `role="status"` para sucesso, `role="alert"` para erro;
  tempo real anuncia só um resumo ("3 novos registros") por `aria-live="polite"`
  (https://www.w3.org/WAI/WCAG22/Understanding/status-messages.html).
- **`lang` e `dir` do idioma declarado** (`mensagens em`), incluindo `dir="rtl"` para árabe,
  que o léxico aceita.
- **Avisos a11y no `ge check` só sobre escolhas do autor** que o gerador não conserta sozinho:
  par de tokens do tema abaixo de 4.5:1/3:1 depois do ajuste, imagem sem descrição possível,
  gráfico sem alternativa derivável, rótulos de ação repetidos ou genéricos, títulos de página
  duplicados, idioma RTL sem suporte. Formato normativo de erro (o quê, onde, por quê, como
  corrigir); supressão só com motivo, visível no `ge explain`, nunca global. O `check` não
  substitui teste com tecnologia assistiva (APG read-me-first).

## 8. Responsividade sem media queries do usuário

O autor nunca escreve breakpoints nem palavras de dispositivo (`no celular mostre…`). Cada nó
da árvore semântica tem um algoritmo de layout fixo (RESPONSIVE.md §4):

- **Layout intrínseco** (Stack, Cluster, Switcher, Grid `auto-fit`/`minmax`), que se adapta sem
  breakpoint (https://every-layout.dev/layouts/, https://web.dev/articles/one-line-layouts).
- **Container queries** dentro dos componentes, para o mesmo nó funcionar numa coluna estreita
  de dashboard e na página inteira
  (https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_containment/Container_queries). Uma
  única media query estrutural, a do shell, com 2 ou 3 classes de janela
  (https://developer.android.com/develop/ui/compose/layouts/adaptive/use-window-size-classes).
  Limiares são tokens (rem), afetados por `densidade`.
- **Tabela → cartões de forma determinística.** O Germanio conhece o campo-título, o estado e os
  obrigatórios; a prioridade das colunas é título > estado > obrigatórios na ordem declarada >
  demais, visível em `ge explain`. Em container estreito, cada linha vira um cartão com as N
  primeiras colunas; as demais continuam no detalhe. A rolagem horizontal atual continua como
  fallback conforme à exceção de tabelas do 1.4.10
  (https://www.w3.org/WAI/WCAG22/Understanding/reflow.html), desde que a região rolável tenha
  nome e foco. A marcação (display + roles de tabela restaurados × lista de cartões) exige teste
  com leitores de tela (https://adrianroselli.com/2018/02/tables-css-display-properties-and-aria.html).
- **Alvos** de pelo menos 24×24 CSS px (2.5.8,
  https://www.w3.org/WAI/WCAG22/Understanding/target-size-minimum.html), verificados também em
  `densidade compacta`.

## 9. Componentes e blocos como progressive disclosure

Quatro camadas que não se misturam (COMPONENTS.md §3):

| Camada | O que é | Onde vive | Muda o compiler? |
| --- | --- | --- | --- |
| LINGUAGEM | a tabela fechada de seções de página | parser + `ast` | sim, raramente, por GEP |
| COMPONENTES | a renderização de cada nó semântico com primitives acessíveis | `runtime/servidor` | não |
| BLOCOS | arquivos `.ge` prontos que só usam a LINGUAGEM (login, kanban, CRM, dashboard) | biblioteca copiável | não |
| TEMA | seed → tokens | gerador de tokens no runtime | não |

- **Primitive separado de aparência**, como Radix Primitives
  (https://www.radix-ui.com/primitives/docs/overview/introduction), Base UI
  (https://base-ui.com/react/overview/about) e React Aria (https://react-aria.adobe.com/): o
  comportamento acessível é implementado uma vez; o tema só troca tokens.
- **O bloco é um arquivo `.ge` copiado e validado, nunca uma keyword.** Como no shadcn ("This is
  not a component library", o código é copiado para o projeto, https://ui.shadcn.com/docs;
  blocos por CLI, https://ui.shadcn.com/blocks), o bloco é código-fonte do usuário: copiado
  para `frontend/`, formatado por `ge fmt`, explicado por `ge explain`, validado por `ge check`
  contra o domínio dele. Um bloco que não se expressa só com a LINGUAGEM revela uma capability
  faltante, e não justifica uma keyword nova. O preço conhecido do shadcn (atualizações não
  chegam sozinhas) fica em INVESTIGAR.
- **Degraus:** 0, `página Clientes` + `mostre clientes` (CRUD, filtros, vazio, responsivo,
  acessível); 1, seções próprias; 2, uma parte nomeada reutilizada entre páginas (o "componente"
  do usuário, mecanismo a definir e só quando houver repetição real); 3, nível técnico, template
  próprio para um nó. Nenhum degrau exige aprender o seguinte.
- **Congelar o vocabulário legado de blocos como nós do AST** (`CustomPage`, `PageNavbar`,
  `PageHero`, `PageSection`, `PageCard`, `PageCodeBlock`, `PageFooter`): é exatamente o modelo
  "cada bloco vira keyword" que a pesquisa recomenda evitar.

## 10. Eficiência

Norma: o JavaScript enviado é um custo considerado; pague só pelo que usar; budgets só depois do
baseline (`docs/INTENCAO.md` › Eficiência).

Estado conferido:

- **Páginas de intenção: zero JavaScript** (nenhum `<script>` em `paginas.go` nem em
  `renderizador_declarativo.go`); CSS embutido no `layoutTpl` de cerca de 4 KB (medido como o
  tamanho em bytes das linhas 1250-1285 do código-fonte, não da resposta HTTP).
- **SPA legado**: carrega em execução o Tailwind Play CDN, o Chart.js e fontes do Google
  (`renderizador.go:31-34`) em toda página, usada ou não a capability de gráfico. Tamanho
  transferido: não medido.
- O binário está no modo de DCE relaxado por causa de `html/template` (cerca de 32 MB,
  `docs/research/performance/RUNTIMES.md`); não afeta o JS enviado, mas é custo de frontend no
  binário.
- As páginas chamam a própria API por `httptest` + JSON (`paginas.go:85-106`), uma chamada por
  filho em série na página de registro; custo não medido.

Medições a fazer (antes de qualquer budget):

1. Bytes comprimidos (gzip e brotli) de HTML, CSS e JS por página gerada, por aplicação de
   exemplo, como portão determinístico de CI (BENCHMARKING.md §12; referência de
   170 KB comprimidos no caminho crítico para mobile, https://web.dev/articles/performance-budgets-101).
2. Tempo de renderização de página e alocações por requisição, já em `bench/app_test.go`
   ("pagina"), contra o baseline em Go direto; acrescentar a página de registro com filhos.
3. O custo das chamadas internas `httptest` + JSON contra chamada direta à camada de dados,
   antes de decidir entre paralelismo, regiões preguiçosas ou streaming (SSR_HYDRATION.md).
4. LCP, INP e CLS no percentil 75 (https://web.dev/articles/vitals) nas páginas de exemplo,
   na suíte STRESS/manual.
5. Quando a camada 2 existir: o tamanho do script genérico (com ou sem biblioteca de morph,
   por exemplo idiomorph, licença e tamanho a verificar) e o custo por espectador de rebuscar
   regiões com muitos espectadores (REACTIVITY.md, INVESTIGAR).

---

## Consolidação

Formato de cada item: **lição** — fonte (estudo; URL) — problema do Germanio que resolve —
arquivo afetado.

### ADOTAR

1. **Modelo visual explícito entre a intenção e o HTML** — VUE.md, SVELTE.md; fases do Svelte
   (https://github.com/sveltejs/svelte) e direção de dependências do Vue
   (https://github.com/vuejs/core/blob/main/.github/contributing.md) — `serve`
   (`paginas.go:607`) mistura permissão, dados e HTML a cada requisição; `ge check`/`ge
   explain` não enxergam páginas — `compiler/ast/intencao.go`, novo passo em `compiler/`,
   `runtime/servidor/paginas.go`, `tooling/explicar/`.
2. **HTML do servidor como camada obrigatória e sem hydration** — RENDERING.md,
   SSR_HYDRATION.md; https://web.dev/articles/rendering-on-the-web — evita segundo runtime,
   divergência servidor/cliente e JS por página — `runtime/servidor/`.
3. **Dependências por região calculadas na compilação** — REACTIVITY.md, SOLID.md;
   https://svelte.dev/blog/runes, https://stack.convex.dev/how-convex-works — páginas de
   intenção não se atualizam, e o autor não deve declarar o que atualiza — novo passo em
   `compiler/`, `tooling/explicar/`.
4. **Aviso após o commit + rebusca autenticada + morph** — REACTIVITY.md;
   https://turbo.hotwired.dev/handbook/page_refreshes — tempo real com permissões corretas por
   construção — `eventos.go`, `transacao.go`, `websocket.go`, `paginas.go`.
5. **Canal de tempo real autenticado, com `Origin`, transportando só avisos** (em correção) —
   SOLID.md, REACT.md — vazamento de registros a qualquer socket — `auth.go`, `websocket.go`,
   `servidor.go`.
6. **`http.CrossOriginProtection` como defesa em profundidade** — RENDERING.md;
   https://pkg.go.dev/net/http#CrossOriginProtection,
   https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html
   — ausente — montagem do handler em `servidor.go`.
7. **Ação de página como verbo do domínio com rótulo opcional** — COMPONENTS.md §4 —
   `ações "Novo cliente"` não determinístico — `compiler/parser/hierarquia.go` (após a GEP).
8. **Formulário derivado da entidade, com erro por campo sem perder o digitado** — REACT.md,
   SVELTE.md, COMPONENTS.md §6; https://react.dev/reference/react/useActionState,
   https://svelte.dev/docs/kit/form-actions — 303 com `?erro=`, valores perdidos, campo não
   indicado — `paginas.go` (`post`, `inputs`, `formTpl`, `message`).
9. **Tabela única campo → controle** — COMPONENTS.md §6 — `telefone`, `url`, `hora`, `Min`,
   `Max` ignorados; `required` só na criação — `paginas.go` (`inputs`).
10. **Permissões só pelo mecanismo do domínio; testes de página sobre a árvore por papel** —
    VUE.md, REACT.md — evitar `if role` e testar sem HTML — testes em `runtime/servidor/`.
11. **Tema como semente → papéis em pares → tokens** — DESIGN_SYSTEMS.md;
    https://ant.design/docs/react/customize-theme — presets de hex à mão, páginas de intenção
    que ignoram o tema — `compiler/ast/ast.go` (`Theme`), gerador de tokens em `runtime/`,
    `layoutTpl`.
12. **Contraste por tom HCT e reverificado pela fórmula WCAG, com teste de ouro** —
    DESIGN_SYSTEMS.md, ACCESSIBILITY.md;
    https://github.com/material-foundation/material-color-utilities/blob/main/typescript/hct/hct.ts,
    https://www.w3.org/TR/WCAG22/ — botões e bordas abaixo de 4.5:1/3:1 — gerador de tokens.
13. **HTML nativo primeiro; `<dialog>`; live regions; labels dos nomes** — ACCESSIBILITY.md;
    https://www.w3.org/WAI/ARIA/apg/practices/read-me-first/,
    https://html.spec.whatwg.org/multipage/interactive-elements.html#the-dialog-element —
    busca e filtros sem label, avisos mudos, tabela sem caption — `paginas.go`
    (`searchTpl`, `tableTpl`, `layoutTpl`).
14. **Regras `a11y_*` do Svelte como testes do renderizador** — SVELTE.md;
    https://svelte.dev/docs/svelte/compiler-warnings — a11y "padrão" sem verificação é
    promessa — testes de `runtime/servidor/`.
15. **Layout intrínseco e container queries; tabela → cartões por prioridade derivada** —
    RESPONSIVE.md; https://every-layout.dev/layouts/,
    https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_containment/Container_queries —
    `nowrap` e rolagem horizontal no celular; navegação que cresce sem limite — `layoutTpl`,
    `tableTpl`.
16. **Primitive separado de aparência** — COMPONENTS.md;
    https://www.radix-ui.com/primitives/docs/overview/introduction — acessibilidade e tema
    misturados nos templates — `runtime/servidor/`.
17. **Um só front-end para compilar, checar, formatar e o LSP** — SVELTE.md;
    https://github.com/sveltejs/language-tools — vários "parsers" (TextMate, formatter, parser)
    — `cli/`, `tooling/`, `vscode-germanio/`.

### ADAPTAR

1. **Troca de regiões à maneira de `hx-boost`/Turbo Frames, por um script genérico do core** —
   RENDERING.md; https://htmx.org/attributes/hx-boost/ — recarga inteira em filtro, página e
   envio — recurso estático embutido em `runtime/servidor/`.
2. **Change tracking do LiveView na granularidade de região, sem estado por conexão** —
   REACTIVITY.md, SSR_HYDRATION.md; https://phoenix-live-view.hexdocs.pm/assigns-eex.html —
   enviar só o que mudou sem memória por conexão — `websocket.go`.
3. **Estados de região (`pending`, `refreshing`, `errored`) como padrão gerado** — SOLID.md;
   https://docs.solidjs.com/guides/fetching-data — região que atualiza sem feedback — emissor.
4. **Error boundary como falha por região** — REACT.md — o `continue` silencioso em `serve` —
   `paginas.go:714`.
5. **`key` como a identidade de registro já existente (`id`/`numero`)** — REACT.md — identidade
   estável em listas atualizadas — emissor.
6. **`contrastLevel` do M3 como `contraste alto`; variantes do M3 como implementação interna de
   um eventual `estilo`** — DESIGN_SYSTEMS.md — acessibilidade sem conceito técnico — gerador
   de tokens.
7. **Avisos a11y estáticos no `ge check` sobre escolhas do autor, com supressão justificada** —
   ACCESSIBILITY.md §5; https://svelte.dev/docs/svelte/compiler-warnings — contraste do tema,
   imagem sem descrição, rótulo ambíguo — `tooling/explicar` (`Verificar`),
   `compiler/diagnostics`.
8. **`lang` e `dir` a partir de `mensagens em`** — ACCESSIBILITY.md — `lang="pt-BR"` fixo com
   léxico de 20 idiomas — os quatro renderizadores.
9. **"Copy the code" do shadcn aplicado a `.ge`** — COMPONENTS.md; https://ui.shadcn.com/docs,
   https://ui.shadcn.com/docs/registry — biblioteca de blocos sem crescer o compiler — CLI.
10. **Classes de janela do M3 reduzidas a 2 ou 3, só para o shell; limiares como tokens** —
    RESPONSIVE.md — `@media` com 640, 768 e 960 espalhados em três renderizadores —
    `layoutTpl`, `renderizador.go`, `renderizador_declarativo.go`.
11. **Carregamento preguiçoso de regiões secundárias antes de qualquer streaming** —
    SSR_HYDRATION.md, SOLID.md; https://docs.solidjs.com/reference/rendering/render-to-stream
    — página de registro com filhos em chamadas sequenciais — `serve`.

### EVITAR

1. **VDOM, runtime de componentes no navegador, `createRenderer`, três renderizadores** —
   VUE.md, RENDERING.md; https://svelte.dev/blog/virtual-dom-is-pure-overhead — runtime JS a
   manter e proteger sem necessidade — `renderizador.go` (convergir e marcar como legado).
2. **Hydration de qualquer tipo e resumability** — SSR_HYDRATION.md;
   https://angular.dev/guide/hydration, https://qwik.dev/docs/concepts/resumable/ — pressupõem
   uma aplicação cliente que o Germanio não tem.
3. **Hooks, signals, stores, `invalidate` ou memoização no `.ge`** — REACT.md, REACTIVITY.md;
   https://react.dev/learn/react-compiler/introduction (a existência do compiler é a evidência
   de que memoização manual é dívida) — princípio de subtração.
4. **Expressões livres em páginas** — SVELTE.md; https://svelte.dev/blog/runes — perderiam a
   análise exata de dependências.
5. **Estado de UI por conexão no servidor (modelo LiveView)** — SSR_HYDRATION.md — memória por
   conexão; o estado relevante já está no banco e na URL.
6. **Dependências de CDN em execução (Tailwind Play CDN, Chart.js)** — DESIGN_SYSTEMS.md,
   RENDERING.md; https://tailwindcss.com/docs/installation/play-cdn ("não é destinado a
   produção") — offline, CSP e disponibilidade — `renderizador.go:31-34`.
7. **CSS em português, hex e px no nível padrão (`CustomCSS`, `borda arredondada 12 px`)** —
   DESIGN_SYSTEMS.md — "CSS em português" — `compiler/parser/germanio.go:769`,
   `compiler/ast/ast.go:111`.
8. **Novos blocos como nós do AST** — COMPONENTS.md — cada bloco vira keyword —
   `compiler/ast/ast.go:485-566`.
9. **Palavras de breakpoint ou de dispositivo; esconder conteúdo no celular sem caminho
   alternativo** — RESPONSIVE.md.
10. **`role="menu"` para navegação; roles, tabindex e ARIA escritos pelo autor** —
    ACCESSIBILITY.md; https://www.w3.org/WAI/ARIA/apg/patterns/disclosure/examples/disclosure-navigation/.
11. **Texto por limiar 3:1; APCA como se fosse WCAG 2.2** — DESIGN_SYSTEMS.md;
    https://mui.com/material-ui/customization/palette/.
12. **Enviar dados pelo socket sem renderizar com as permissões de cada espectador** —
    SOLID.md, REACTIVITY.md.

### INVESTIGAR

1. **Semântica de indicadores e agregados** (`total de`, `soma de … do mês`) e manutenção
   incremental contra recálculo com coalescência — REACTIVITY.md, SOLID.md;
   https://pdos.csail.mit.edu/papers/noria:osdi18.pdf — pré-requisito de indicadores e gráfico.
2. **Custo com muitos espectadores**, agrupamento por classe de visibilidade, e o aviso entre
   vários processos — SOLID.md, REACTIVITY.md.
3. **Morph no cliente**: biblioteca existente servida localmente ou morph mínimo próprio
   (licença e tamanho) — REACTIVITY.md.
4. **Dependências temporais ("hoje", "do mês") no fuso de quem vê** — SOLID.md.
5. **Se `conteúdo` é necessário** ou só profundidade extra — COMPONENTS.md.
6. **O mecanismo do degrau 2** (parte nomeada reutilizável) e a sua fusão entre arquivos —
   COMPONENTS.md.
7. **Atualizar um bloco copiado quando a biblioteca evolui** — COMPONENTS.md.
8. **Porte de HCT/CAM16 para Go** (licença do MCU a confirmar) e OKLCH × HCT —
   DESIGN_SYSTEMS.md; https://github.com/material-foundation/material-color-utilities.
9. **Se `estilo` deve existir**, e onde mora o tema (`frontend/tema.ge`?) — DESIGN_SYSTEMS.md.
10. **Marcação da tabela estreita** testada com NVDA, VoiceOver e TalkBack; limiar de toque 44px
    × 24px — RESPONSIVE.md.
11. **Mensagens de validação do navegador × do domínio** — ACCESSIBILITY.md.
12. **Radio × select para enum pequeno; busca para referências grandes** (limiar
    determinístico) — COMPONENTS.md.
13. **Ilhas para gráfico e kanban**: contrato de dados de entrada e eventos de saída, sem
    expô-lo ao nível padrão — RENDERING.md, SSR_HYDRATION.md;
    https://docs.astro.build/en/concepts/islands/.
14. **Pré-renderização de páginas públicas** (as que não dependem de quem vê, deriváveis do
    `ast.App`) — SVELTE.md; https://svelte.dev/docs/kit/page-options.
15. **Resposta otimista** só para interações de alta frequência — REACT.md;
    https://react.dev/reference/react/useActionState.
16. **Tirar o binário do modo de DCE relaxado** causado por `html/template` —
    `docs/research/performance/RUNTIMES.md`.

## Decisões que exigem GEP

1. **Tabela fechada de seções de página e árvore semântica** (G62, antes G58): as seções, os pais, os
   padrões, as frases planas, a definição de `secao_pagina` na gramática, e se `conteúdo`
   existe.
2. **Ações de página**: verbo com rótulo (`criar "Novo cliente"`) e a relação com o `permita`
   da página (opções a/b/c da seção 3).
3. **Indicadores e gráficos**: a semântica dos agregados como fatos do domínio (entidade,
   filtro, função, período), sem expressões livres.
4. **Vocabulário do tema**: `destaque`, `cantos`, `densidade`, `modo`, `contraste`; o destino
   de `estilo`, dos presets atuais, de `CustomCSS` e de `borda arredondada 12 px`; onde o tema
   mora e como se funde.
5. **Contrato de formulário e de feedback**: 422 com re-renderização, erro por campo, fim de
   `?ok=`/`?erro=`, aviso de uso único, `required` na edição, tabela campo → controle.
6. **Camada de melhoria progressiva e tempo real**: a regra "nenhum script reconstrói o DOM do
   servidor", o contrato de regiões, aviso + rebusca, os pré-requisitos de segurança, e se o
   tempo real é automático ou declarado (pague só pelo que usar).
7. **Blocos e legado**: o bloco como arquivo `.ge` copiado (comando, índice), o congelamento ou
   a remoção de `CustomPage`/`PageHero`/`PageNavbar` e do SPA legado.
8. **Avisos a11y no `ge check`**: os códigos, o formato, a supressão com motivo, e o que é
   invariante do gerador (teste) versus aviso ao autor.

## Divergências encontradas na implementação atual

Cada item foi conferido no código (commit `d9676b9`; as linhas citadas de `paginas.go`,
`renderizador.go`, `renderizador_declarativo.go` e `compiler/` não mudaram desde `fc31daf`). Os números de contraste foram recalculados para este
documento pela fórmula de luminância relativa do WCAG 2.2.

| # | Divergência | Evidência | Estado |
| --- | --- | --- | --- |
| 1 | **Contraste do tema abaixo de 4.5:1.** O SPA pinta botões `bg-primary … text-white`; branco sobre cada `ColorName`: azul 3.68, verde 2.28, vermelho 3.76, laranja 2.80, rosa 3.53, amarelo 1.92, ciano 2.43, esmeralda 2.54, âmbar 2.15, roxo 4.23, índigo 4.47 (também a primária padrão `#6366f1`); só cinza 4.83 e violeta 5.70 passam. Nas páginas de intenção, modo escuro: texto `#fff` sobre `--accent #a371f7` = 3.35; borda de campo `--line #d0d7de` sobre branco = 1.45 (1.4.11 pede 3:1 quando a borda identifica o campo) | `compiler/ast/ast.go:127-140`; `renderizador.go:106, 124, 292, 514, 530, 579`; `paginas.go:1252-1253, 1268-1269` | confirmada |
| 2 | **Páginas de intenção ignoram `ast.Theme`**: tokens fixos no `:root`; `tema azul` não muda nada nelas | `paginas.go:1252-1253`; nenhuma referência a `Theme` em `paginas.go` | confirmada |
| 3 | **Campos de tema nunca usados e `styleVariantCSS` vazio.** `Radius`, `Style`, `Background`, `CardBg`, `TextColor` e `Sidebar` recebem valor em `applyThemeDefaults` e nunca são lidos; `styleVariantCSS` devolve `""` e nem é chamado. Os "4 estilos" (glassmorphism/flat/neumorphism/minimal) descritos no `CLAUDE.md` não existem no código | `compiler/ast/ast.go:98-112`; `renderizador.go:122-164, 868-872` | confirmada |
| 4 | **Tailwind Play CDN** (e Chart.js e Google Fonts) carregados em execução pelo SPA legado | `renderizador.go:31-34` | confirmada |
| 5 | **Busca e filtros sem `<label>`**: só `placeholder`; filtros são sempre `<input>` de texto, inclusive para enum | `paginas.go:1300-1301` | confirmada |
| 6 | **Tabela sem `<caption>` nem `scope`**; a região `.tabela` rolável não tem nome nem `tabindex` | `paginas.go:1289-1291` | confirmada |
| 7 | **Erro de formulário via `?erro=`**: 303 para a página de origem, o formulário volta vazio, e `message` achata o mapa de erros por campo que a API devolve numa única frase, sem indicar o campo; `required` só na criação | `paginas.go:1014`, `110-136`, `466` | confirmada |
| 8 | **`?ok=`/`?erro=` forjáveis**: o texto da query string é mostrado como aviso do sistema (escapado, portanto não é XSS, mas é conteúdo falso injetável por um link); os avisos não têm `role="status"`/`"alert"` | `paginas.go:247-248`, `1283` | confirmada |
| 9 | **`lang` fixo sem `dir`**: `lang="pt-BR"` nos quatro geradores de HTML, sem `dir="rtl"`, embora o léxico aceite árabe | `paginas.go:1250`; `renderizador.go:28`; `renderizador_declarativo.go:21`; `servidor.go:1279` | confirmada |
| 10 | **O `continue` que esconde erro interno**: na página de registro, qualquer status diferente de 200 ao listar os filhos pula a seção; 403/404 (correto esconder) e 500 (erro interno) têm o mesmo tratamento, sem mensagem nem registro | `paginas.go:713-717` | confirmada |
| 11 | **Blocos legados como nós do AST**: `PageNavbar`, `PageHero`, `PageHeroButton`, `PageSection`, `PageCard`, `PageCodeBlock`, `PageFooter`, `CustomPage`, desenhados por um renderizador próprio; no mesmo espírito, `CustomCSS` e `borda arredondada 12 px` | `compiler/ast/ast.go:485-566`; `renderizador_declarativo.go:12`; `compiler/ast/ast.go:111`; `compiler/parser/germanio.go:769` | confirmada |
| 12 | **`/ws` sem autenticação e difusão de eventos a todos os sockets** | `servidor.go:110, 692, 726, 735, 1386`; `websocket.go` | **em correção** pelo orquestrador: `d9676b9` (G65) exige token e `Origin` e reduz o aviso a tipo/modelo/id; G66 aberto |
| 13 | **Identificadores de lacuna duplicados** (G58/G59 usados duas vezes) em `GERMANIO_GAPS.md` | renumerados para G61–G64 em `d9676b9` (`GERMANIO_GAPS.md:8`) | resolvida durante esta pesquisa |
| 14 | **`secao_pagina` usada e não definida** na gramática normativa | `docs/INTENCAO.md:295` | confirmada (lacuna da especificação) |
| 15 | **`http.CrossOriginProtection` ausente** | nenhuma ocorrência em `runtime/` | confirmada |
| 16 | **Hub descarta mensagens em silêncio** quando o buffer de 64 de um cliente enche | `websocket.go:74, 120` | confirmada; registrada como G66 |
| 17 | **Emojis como único conteúdo de ícone** na árvore de arquivos | `paginas.go:1303` | confirmada |

Não confirmadas (citadas nos estudos, não verificadas aqui): o tamanho transferido do Tailwind
Play CDN e do Chart.js; a afirmação de que a inserção de uma live region já com conteúdo nem
sempre é anunciada (prática corrente, sem fonte primária, ACCESSIBILITY.md §4); o custo das
chamadas internas `httptest` + JSON por página (não medido).

Estas divergências são registros de pesquisa. Pela regra do Documentation Gate, cada uma deve
ser classificada (implementação errada, especificação incompleta ou decisão nova) e registrada
em `GERMANIO_GAPS.md` por quem for resolvê-la; este documento não altera arquivos fora de
`docs/research/frontend/`.
