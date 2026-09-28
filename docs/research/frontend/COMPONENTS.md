# Componentes, blocos e a árvore semântica da página

Data: 2026-09-28 · Status: pesquisa (não normativa). As seções de página propostas são direção
(G62 em `GERMANIO_GAPS.md`; `docs/INTENCAO.md` › Pendências da sintaxe hierárquica), não contrato.

## Fontes consultadas

- shadcn/ui, introdução ("This is not a component library"): https://ui.shadcn.com/docs
- shadcn/ui, blocks: https://ui.shadcn.com/blocks
- shadcn/ui, registry: https://ui.shadcn.com/docs/registry
- Radix Primitives, introdução: https://www.radix-ui.com/primitives/docs/overview/introduction
- Radix Themes, getting started: https://www.radix-ui.com/themes/docs/overview/getting-started
- Base UI, about: https://base-ui.com/react/overview/about
- React Aria: https://react-aria.adobe.com/
- React Aria, forms: https://react-aria.adobe.com/forms
- Chakra UI v3, theming (recipes, slot recipes): https://chakra-ui.com/docs/theming/overview
- Ant Design, tema (component tokens): https://ant.design/docs/react/customize-theme
- Ant Design, data display (estados extremos, vazio): https://ant.design/docs/spec/data-display
- WCAG 2.2: https://www.w3.org/TR/WCAG22/ · 4.1.3 Status Messages: https://www.w3.org/WAI/WCAG22/Understanding/status-messages.html

Código do Germanio lido: `compiler/ast/intencao.go` (`PageDecl`), `compiler/ast/ast.go`
(`Field`, `CustomPage`, `PageHero`, `PageNavbar`…), `runtime/servidor/paginas.go` (tabela,
formulário, filtros, vazio, avisos), `runtime/servidor/renderizador_declarativo.go`,
`examples/gitlab-foss/frontend/paginas.ge`, `docs/INTENCAO.md` (Sintaxe hierárquica, Página).

## 1. Três coisas diferentes chamadas "componente"

| Nível | O que entrega | Exemplos | Quem é dono do código |
| --- | --- | --- | --- |
| Primitive (comportamento) | semântica, teclado, foco, ARIA; sem estilo | Radix Primitives, Base UI, React Aria | a biblioteca |
| Componente estilizado | primitive + tema + variants | Radix Themes, MUI, Ant Design, Chakra | a biblioteca |
| Bloco | composição de componentes numa tela ou seção inteira | shadcn blocks (dashboard, sidebar, login, signup, calendar, charts) | **quem o instala** |

- Radix Primitives: "os componentes são entregues sem estilos" e "seguem os WAI-ARIA design
  patterns onde possível"; cada parte do componente é acessível individualmente ("open component
  architecture") e são "não controlados por padrão" (radix-ui.com/primitives). Radix Themes é
  "uma biblioteca pré-estilizada" sobre os Primitives (radix-ui.com/themes).
- Base UI, dos times de Radix, Material UI e Floating UI: sem estilo, acessível, APIs abertas
  (base-ui.com). React Aria oferece três níveis à escolha: componentes com DOM, composição por
  contexto e hooks de baixo nível (react-aria.adobe.com).
- shadcn/ui: "Isto não é uma biblioteca de componentes. É como você constrói a sua"; o código é
  **copiado** para o projeto ("open code"), com interface composicional comum e "beautiful
  defaults" (ui.shadcn.com/docs). Os blocos se instalam por CLI (`npx shadcn add dashboard-01`)
  e trazem páginas e componentes (ui.shadcn.com/blocks). O registry distribui itens tipados
  (`registry:ui`, `registry:component`, `registry:block`, `registry:theme`…) descritos em
  `registry.json`/`registry-item.json` e "funciona com qualquer framework"
  (ui.shadcn.com/docs/registry).

Lição de arquitetura: **comportamento acessível** (primitive) e **aparência** (tema) são camadas
separadas, e o **bloco** é código-fonte do usuário, não API da biblioteca. O shadcn resolve o
problema de "wrapper e override" dando o código; o preço é que atualizações não chegam sozinhas.

### 1.1 Slots e variants

- Slots: componentes com partes nomeadas (as partes dos Primitives; as slot recipes do Chakra,
  "um objeto de estilo por slot") (chakra-ui.com/docs/theming/overview).
- Variants: combinações enumeradas (recipes via `cva`), e component tokens da Ant que mudam um
  componente sem afetar os outros (ant.design/docs/react/customize-theme).

Tradução para o Germanio: **as seções de uma página são os slots de um layout fixo**, e uma
variant é uma escolha fechada que o renderizador mapeia para tokens (ver `DESIGN_SYSTEMS.md`).
O autor nunca escreve a estrutura do DOM.

## 2. O estado atual no Germanio (verificado)

- A página de intenção é `PageDecl{Name, Manage, Show, Permits, PerPage}`
  (`compiler/ast/intencao.go`). O renderizador (`paginas.go`) deriva tudo o mais do `ast.App`:
  colunas (`columns`), rótulos (`fieldLabel`), formulário (`inputs`), opções de referências
  (`optionsFor`), ações disponíveis pela permissão (`available`), vazio (`"Nenhum registro de
  … ainda."`), paginação, avisos `?ok=`/`?erro=`.
- Existe um **segundo** vocabulário de página, legado, em que os blocos são nós do AST e do
  parser: `CustomPage` com `PageNavbar`, `PageHero`, `PageSection`, `PageCard`, `PageCodeBlock`,
  `PageFooter` (`compiler/ast/ast.go`), desenhados por `renderizador_declarativo.go`. É o
  modelo "cada bloco vira keyword" que o contexto pede para evitar.

## 3. Quatro camadas separadas

| Camada | O que é | Onde vive | Muda o compiler? |
| --- | --- | --- | --- |
| LINGUAGEM | tabela fechada de **seções de página** (slots), com gramática e frase plana | parser + `ast` | sim, raramente e com norma |
| COMPONENTES | renderização de cada **nó semântico** (Table, Form, EmptyState…) com primitives acessíveis | `runtime/servidor` (Go + templates) | não |
| BLOCOS | arquivos `.ge` prontos que usam só a LINGUAGEM (login, kanban, CRM) | biblioteca de exemplos copiáveis (`ge` copia o `.ge` para `frontend/`) | não |
| TEMA | seed → tokens | `DESIGN_SYSTEMS.md` | não |

Critério para algo entrar na LINGUAGEM (proposta): (a) tem significado de domínio, não visual
("vazio", "filtros", "ações"), (b) pode ser reduzido a um nó da árvore semântica e (c) a sua
ausência tem um padrão derivável. Hero, pricing, navbar **não** passam: são composições (BLOCOS)
ou decisões do renderizador (COMPONENTES).

Consequência: o bloco "à la shadcn" no Germanio é **um arquivo `.ge` copiado**, não um pacote
Go. O usuário o possui, o `ge fmt` o formata, o `ge explain` o explica e o `ge check` o valida
contra o domínio dele. Um bloco que não se expressa só com a LINGUAGEM revela uma capability
faltante (regra da skill), não uma nova keyword.

## 4. Árvore semântica da página (proposta)

A mesma técnica da sintaxe hierárquica de dados: layout → árvore de linhas → cada seção de uma
**tabela fechada** vira um nó; linhas fora da tabela são erro que lista as seções válidas; cada
nó tem uma frase plana equivalente; um teste prova a equivalência (`docs/INTENCAO.md` › Tabela
de seções).

```text
Page(nome)
├── Header        ← topo
│   ├── Title     ← título "…"          (padrão: nome da página)
│   ├── Text      ← texto "…"           (opcional)
│   └── Actions   ← ações › verbos      (padrão: ações de coleção permitidas, ex.: criar)
├── Content       ← conteúdo            (padrão: a região derivada de `mostre`)
│   ├── Filters   ← filtros › pesquisar | campo   (padrão: de `permita pesquisar/filtrar`)
│   ├── Table     ← tabela › dado + colunas › campos  (padrão: `columns(e)`)
│   ├── Indicators← indicadores › "total de …"  (G62, depende de agregações)
│   └── Chart     ← gráfico …                     (G62)
└── EmptyState    ← vazio › título, texto, ação  (padrão: mensagem derivada + ação criar)
```

Tabela de seções (candidata; "Pai" é o contexto obrigatório):

| Seção | Pai | Conteúdo | Validação contra o domínio | Frase plana (candidata) |
| --- | --- | --- | --- | --- |
| `topo` | página | `título`, `texto`, `ações` | — | — |
| `título "X"` | topo, vazio | um texto | — | `página P tem título "X"` |
| `ações` | topo, vazio | um verbo por linha, opcional `"rótulo"` | o verbo existe e é de coleção | `página P oferece criar` |
| `conteúdo` | página | `filtros`, `tabela`, `lista`, `indicadores`, `gráfico` | ao menos uma região | — |
| `filtros` | conteúdo | `pesquisar` ou nome de campo | o campo existe no dado mostrado; `pesquisar` exige `permita pesquisar` | `página P filtra por status` |
| `tabela` | conteúdo | o dado; `colunas` › campos | dado e campos existem; campo `privado`/`oculto` gera erro | `página P mostra clientes com nome, email` |
| `vazio` | página | `título`, `texto`, `ação` | a ação é permitida | `página P quando vazia diz "…"` |

Três decisões que a pesquisa sustenta e que **corrigem** o exemplo de direção do contexto:

1. `ações` › `"Novo cliente"` (só texto) não é determinístico: o texto não diz qual capability
   executa. A ação deve ser um **verbo** do domínio (`criar`), com rótulo opcional
   (`criar "Novo cliente"`). O rótulo padrão é derivado (`Novo cliente` a partir do singular).
2. `tabela` › `clientes` repete o que `mostre clientes` já declarou; a hierarquia deve dispensar
   a repetição quando a página mostra um só dado (critério 3 de "Como avaliar uma sintaxe").
3. A presença de um botão nunca é decidida na página: `ações` › `criar` só **pede** o botão; ele
   só aparece para quem tem o grant (como `available` já faz). Nada de `se papel é admin`.

Nível 0 continua valendo: `página Clientes` + `mostre clientes` produz a árvore inteira pelos
padrões. Cada seção escrita **substitui só o seu slot**; `ge explain` mostra, para cada nó, se
veio do `.ge` (arquivo:linha) ou do padrão (e de qual fato do domínio).

### 4.1 Progressive disclosure

| Degrau | O que a pessoa escreve | O que ganha |
| --- | --- | --- |
| 0 | `página Clientes` / `mostre clientes` | CRUD, filtros, vazio, responsivo, acessível |
| 1 | seções (`topo`, `vazio`, `filtros`, `colunas`) | textos e escolhas próprias |
| 2 | extrair uma parte nomeada para reuso em várias páginas (mecanismo a definir) | a mesma seção em muitas páginas |
| 3 | nível técnico/adaptador: template próprio para um nó | controle total, fora do nível padrão |

O degrau 2 é o "componente" do usuário. Ele só deve ser introduzido quando houver repetição real
entre páginas (critério 3 da avaliação de sintaxe), e deve ser uma parte da LINGUAGEM (um
subárvore com nome), não um componente Go.

## 5. Estados de feedback

Toda região de dados tem quatro estados; os sistemas estudados os tratam como partes do
componente (a Ant pede atenção a "situações extremas… o estado inicial quando o conteúdo está
vazio", ant.design/docs/spec/data-display; o React Aria separa `FieldError`).

| Estado | Padrão derivado | Anúncio (ver `ACCESSIBILITY.md`) | Personalização |
| --- | --- | --- | --- |
| carregando | só em atualização assíncrona (a página é SSR); indicador na região e `aria-busy` | nenhum anúncio em cada tique | não precisa de seção |
| vazio | título e texto derivados do dado + ação `criar` se permitida; distinguir "sem registros" de "nenhum resultado para o filtro" | não anuncia | seção `vazio` |
| erro | mensagem do domínio (a mesma do diagnóstico), perto do que falhou; valores preservados | `role="alert"` | mensagens em `regras`/`mensagens em` |
| sucesso | aviso curto + destino derivado (o registro criado) | `role="status"` | texto do aviso |

Hoje: vazio existe e é derivado (bom); sucesso e erro chegam como `?ok=`/`?erro=` e aparecem num
`div.aviso` sem papel de live region; o erro não diz qual campo; e, após o redirect, os valores
digitados se perdem.

## 6. Formulários derivados do domínio

`formulário cliente` (ou o formulário implícito do `permita criar`) é uma projeção do
`ast.Field`. O mapeamento deve ser uma tabela única, testada, que o `ge explain` pode mostrar.

| Fato do domínio (`ast.Field`) | Hoje em `inputs()` | Deveria derivar |
| --- | --- | --- |
| `Label` / nome | `<label>` envolvendo o controle | idem (associação implícita é válida) |
| `Required` | `required` só na criação (`values == nil`) | também na edição; indicação visual + textual |
| `email` | `type=email` | idem + `autocomplete=email` |
| `telefone`, `url`/`link`, `hora`, `data_hora`, `cor` | `type=text` | `tel`, `url`, `time`, `datetime-local`, `color` |
| `inteiro`, `numero`, `dinheiro` | `type=number` | `inputmode` adequado; `step` para dinheiro; unidade no rótulo |
| `cpf`, `cep` | `type=text` | `inputmode=numeric`, `autocomplete=postal-code` (cep) |
| `Min`/`Max` (texto: tamanho) | ignorado | `minlength`/`maxlength`; para números `min`/`max` |
| `Format` (RE2) | ignorado | `pattern` **só** quando a expressão é válida nos dois dialetos (RE2 ≠ JS); senão, só no servidor |
| `Unique` | só servidor | só servidor; mensagem no campo |
| `EnumValues` | `select` | `select` (ou grupo de rádio para poucas opções: a decidir) |
| referência | `select` com todas as opções visíveis | `select`; busca quando o número de opções for grande |
| `Immutable`, `System`, `Hidden`, segredo | omitidos | idem (correto) |

Estado de envio (proposta): botão desabilitado e `aria-busy` durante o envio; em erro, o servidor
**re-renderiza** o formulário com os valores e o erro de cada campo (em vez de redirect com
`?erro=`), um resumo de erros no topo com links para os campos e o foco movido para o resumo;
em sucesso, redirect para o registro com aviso `role="status"`. O React Aria mostra a mesma
divisão: validação nativa por restrições (required, tipo, tamanho, padrão), erros do servidor
por campo ("exibidos assim que definidos e limpos quando a pessoa altera o valor")
(react-aria.adobe.com/forms). A validação no cliente é conveniência; a do servidor
(`banco.Validar`) continua sendo a autoridade.

## 7. Biblioteca oficial de blocos

Cada bloco é um `.ge` que usa só a LINGUAGEM. A coluna "Falta" é o que impede escrevê-lo hoje e
aponta a capability, não uma keyword.

| Bloco | Deriva de | Falta |
| --- | --- | --- |
| login, cadastro | `tenha login`, `login …` (já existem páginas `/entrar`, `/cadastro`) | nada essencial; o bloco só personaliza textos |
| navbar, sidebar | páginas declaradas + permissões | regra de navegação (quais páginas, ordem): hoje derivada |
| dashboard, stats | indicadores (`total de clientes`), gráfico | agregações declarativas (G62) |
| charts | gráfico sobre campo/estado | G62 + alternativa textual (tabela) obrigatória |
| tables, cards | `mostre` + colunas | colunas escolhidas (seção `colunas`) |
| forms | formulário derivado | tabela de mapeamento da seção 6 |
| kanban | estados e transições (`começa`, `pode`) | vista por estado + mover por teclado (não só arrastar) |
| chat | dado de mensagens + tempo real | reatividade derivada (websocket existe no runtime) |
| checkout | pedido + pagamento | adaptador de pagamento (`integracoes/`), nunca no core |
| settings, profile | a pessoa logada e seus campos | escopo "meus dados" (`seu/seus`) |
| CRM | composição de clientes, oportunidades, estados, donos | nenhum mecanismo novo: é prova de generalização |
| hero, pricing | conteúdo estático (marketing) | pertence a páginas estáticas (`pagina "/caminho"`), não à intenção de dados |

## Para o Germanio

**ADOTAR**

- Primitive (comportamento acessível) separado de aparência (tema), como Radix/Base UI/React
  Aria: os COMPONENTES do runtime implementam o comportamento uma vez; o tema só troca tokens.
  Resolve: acessibilidade e tema misturados nos templates de `paginas.go`.
- Seções de página como tabela fechada de slots reduzida a uma árvore semântica (Page, Header,
  Actions, Filters, Table, EmptyState), com frase plana e teste de equivalência, exatamente como
  `hierarquia.go`/`dataSection`. Resolve: G62 sem abrir uma keyword por componente. Afeta
  `compiler/parser/hierarquia.go`, `compiler/ast/intencao.go` (`PageDecl` ganha a árvore),
  `runtime/servidor/paginas.go`, `tooling/explicar/`.
- Ação de página como verbo do domínio com rótulo opcional. Resolve: `ações "Novo cliente"`
  sem significado verificável.
- Tabela única campo → controle (seção 6), testada. Resolve: `telefone`, `url`, `hora`, `Min`,
  `Max` ignorados em `inputs()`.
- Quatro estados por região, com padrão derivado. Resolve: erro sem campo, valores perdidos no
  redirect, aviso sem live region (`paginas.go`, `post()` e `layoutTpl`).

**ADAPTAR**

- "Copy the code" do shadcn aplicado a `.ge`: o bloco é um arquivo de intenção copiado para
  `frontend/`, possuído pelo usuário e validado contra o domínio dele; um comando (nome a
  definir) só copia. Resolve: biblioteca de blocos sem crescer o compiler.
- Registry tipado (shadcn) como índice de blocos e temas: útil depois de haver blocos; não antes.
- Slot recipes/variants (Chakra) como valores fechados por nó (por exemplo densidade da tabela),
  mapeados para tokens.

**EVITAR**

- Crescer `CustomPage`/`PageHero`/`PageNavbar` (blocos como nós do AST) no nível padrão.
  Resolve: o risco "cada componente vira keyword". Congelar como nível técnico/legado.
- Expor a estrutura do DOM (children, props, render props, hooks) ao autor.
- Validação só no cliente, ou `pattern` gerado de RE2 sem conferir o dialeto.
- Blocos que embutem regras de produto no core (checkout com nomes de gateway em `runtime/`).

**INVESTIGAR**

- O mecanismo do degrau 2 (parte nomeada reutilizável) e a sua fusão entre arquivos.
- Se `conteúdo` é necessário ou só profundidade extra (profundidade hierárquica é custo; talvez
  `filtros` e `tabela` possam ser filhos diretos de `página`).
- Radio versus select para enum pequeno; busca para referências grandes (limiar determinístico).
- Como atualizar um bloco copiado quando a biblioteca evolui (o custo conhecido do shadcn).
