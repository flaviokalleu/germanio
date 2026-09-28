# Design systems, design tokens e tema como intenção

Data: 2026-09-28 · Status: pesquisa (não normativa). Nada aqui altera `docs/INTENCAO.md`; as
palavras de tema propostas (`estilo`, `destaque`, `cantos`, `densidade`) são candidatas, não sintaxe.

## Fontes consultadas

- Design Tokens Community Group, *Design Tokens Format Module 2025.10* (Final Community Group Report, 28/10/2025): https://www.designtokens.org/tr/2025.10/format/ — rascunho seguinte: https://www.designtokens.org/tr/drafts/format/
- Material Color Utilities (Google): https://github.com/material-foundation/material-color-utilities
- MCU, conceito de esquema dinâmico: https://github.com/material-foundation/material-color-utilities/blob/main/concepts/dynamic_color_scheme.md
- MCU, `hct.ts` (garantia tom → contraste): https://github.com/material-foundation/material-color-utilities/blob/main/typescript/hct/hct.ts
- MCU, `dynamic_scheme.ts` (`contrastLevel`): https://github.com/material-foundation/material-color-utilities/blob/main/typescript/dynamiccolor/dynamic_scheme.ts
- MCU, `variant.ts` (variantes de esquema): https://github.com/material-foundation/material-color-utilities/blob/main/typescript/dynamiccolor/variant.ts
- Material 3, sistema de cor: https://m3.material.io/styles/color/system/how-the-system-works (a página não renderizou no fetch; fatos confirmados pelas fontes do MCU acima)
- Material 3, design tokens: https://m3.material.io/foundations/design-tokens/overview (idem: só o título foi obtido; a divisão ref/sys/comp está marcada como não verificada no texto)
- MUI, paleta (`tonalOffset`, `contrastThreshold`): https://mui.com/material-ui/customization/palette/
- Ant Design, tema e tokens: https://ant.design/docs/react/customize-theme
- Radix Colors, a escala de 12 passos: https://www.radix-ui.com/colors/docs/palette-composition/understanding-the-scale
- Radix Colors, paletas personalizadas: https://www.radix-ui.com/colors/docs/overview/custom-palettes
- Radix Themes, `Theme`: https://www.radix-ui.com/themes/docs/theme/overview
- shadcn/ui, theming: https://ui.shadcn.com/docs/theming
- Chakra UI v3, theming: https://chakra-ui.com/docs/theming/overview
- Tailwind CSS v4, theme variables: https://tailwindcss.com/docs/theme
- Tailwind, Play CDN: https://tailwindcss.com/docs/installation/play-cdn
- WCAG 2.2 (W3C Recommendation, versão de 12/12/2024): https://www.w3.org/TR/WCAG22/

Código do Germanio lido: `compiler/ast/ast.go` (`Theme`, `DefaultTheme`, `ColorName`,
`ThemePreset`, `ResolveColor`), `compiler/parser/parser.go` (`parseTema`),
`compiler/parser/germanio.go` (cores no núcleo estrito), `runtime/servidor/renderizador.go`
(SPA legado: `applyThemeDefaults`, `generateCSS`, `styleVariantCSS`),
`runtime/servidor/paginas.go` (`layoutTpl`, as páginas de intenção).

## 1. O que os sistemas estudados têm em comum

Todos os sistemas maduros separam **o valor** (uma cor, um raio) **do papel** (a superfície de
um botão principal, o texto sobre ela) e fazem os componentes consumirem só papéis. Diferem em
quantas camadas existem e em quem calcula a camada de baixo.

| Sistema | Camadas | Quem gera a paleta | Fonte |
| --- | --- | --- | --- |
| Material 3 | referência → sistema → componente (`md.ref`, `md.sys`, `md.comp`, não verificado no texto da página) | algoritmo HCT a partir de **uma** cor-fonte | MCU |
| Ant Design | Seed → Map → Alias, mais Component Token | `defaultAlgorithm`, `darkAlgorithm`, `compactAlgorithm`, combináveis | ant.design |
| Chakra v3 | tokens → semantic tokens; recipes e slot recipes | o autor escreve os valores | chakra-ui.com |
| shadcn/ui | pares `x` / `x-foreground` em CSS variables (OKLCH) | o autor escolhe (ou um tema pronto) | ui.shadcn.com |
| Radix Colors/Themes | escala de 12 passos com uso definido por passo; `Theme` com `accentColor`, `grayColor`, `radius`, `scaling` | escalas feitas à mão; gerador para cor personalizada | radix-ui.com |
| Tailwind v4 | `@theme` com namespaces (`--color-*`, `--radius-*`, `--spacing`, `--breakpoint-*`) que **geram utilitários** | paleta fixa em OKLCH | tailwindcss.com |
| DTCG 2025.10 | formato de intercâmbio: `$value`, `$type`, grupos, aliases `{grupo.token}` | não se aplica (é formato, não algoritmo) | designtokens.org |

### 1.1 Três camadas de derivação (Ant Design)

A Ant Design descreve os tokens como "três camadas de derivação", não agrupamentos: Seed Token
(`colorPrimary`, `borderRadius` 6, `fontSize` 14, `controlHeight` 32, `sizeUnit`, `sizeStep`,
`wireframe`, `motion`) → Map Token (`colorPrimaryBg`, `colorPrimaryBorder`, `borderRadiusLG`…,
"calculados a partir do seed por algoritmos", como "calcular uma paleta em gradiente a partir de
uma cor base, ou raios de vários tamanhos a partir de um raio base") → Alias Token (controle em
lote de componentes comuns). Os algoritmos são funções puras e compõem: escuro + compacto é
`[darkAlgorithm, compactAlgorithm]` (https://ant.design/docs/react/customize-theme). A v6 tem um
modo `zeroRuntime`, que não gera estilos em runtime (idem).

Lição estrutural: **a intenção do autor mora só na camada Seed**; tudo acima é função. É
exatamente a forma que o Germanio precisa: poucas palavras na `.ge`, o resto calculado e
mostrado por `ge explain`.

### 1.2 Cor dinâmica a partir de uma cor-fonte (Material 3 / HCT)

- "Os esquemas de cor do Material começam de uma cor-fonte, uma única cor da qual todas as
  outras cores do esquema são derivadas"; o algoritmo cria quatro cores complementares, total
  de cinco cores-chave, cada uma com uma paleta tonal de 13 tons de 0 (preto) a 100 (branco);
  tons são atribuídos a **papéis** (color roles) (MCU, `concepts/dynamic_color_scheme.md`).
- HCT = matiz e croma do CAM16 + L* do L*a*b*. A garantia central, literal no código: "Uma
  diferença de 40 no tom HCT garante razão de contraste >= 3.0, e uma diferença de 50 garante
  >= 4.5" (MCU, `typescript/hct/hct.ts`). Ou seja, o contraste WCAG vira uma **aritmética de
  tons**, independente da matiz.
- Exemplo de papel: no esquema claro, `primary` é o tom 40 e `onPrimary` o tom 100 (diferença
  60, portanto ≥ 4.5:1 por construção). (Fonte: resumo de busca que aponta
  m3.material.io/styles/color/system/how-the-system-works; o texto da página não foi obtido
  diretamente: não verificado na fonte primária.)
- `DynamicScheme` recebe `sourceColorHct`, `variant`, `isDark` e `contrastLevel`: "valor de -1
  a 1. -1 representa contraste mínimo, 0 o padrão (o design como especificado) e 1 o contraste
  máximo" (MCU, `dynamic_scheme.ts`). Variantes: MONOCHROME, NEUTRAL, TONAL_SPOT, VIBRANT,
  EXPRESSIVE, FIDELITY, CONTENT, RAINBOW, FRUIT_SALAD, CMF (MCU, `variant.ts`).
- A biblioteca existe em C++, Dart, Java, Swift, TypeScript e Kotlin (README do MCU). Não há
  porte oficial em Go; a busca mostrou portes de terceiros (por exemplo
  `cogentcore.org/core/colors/cam/hct`, `github.com/worldiety/material-color-utilities`), não
  avaliados quanto a licença, fidelidade e manutenção (não verificado).

### 1.3 Outras formas de garantir contraste

- **Radix Colors**: cada passo tem uso definido (1–2 fundos, 3–5 fundos de componente
  normal/hover/pressionado, 6–8 bordas e anel de foco, 9–10 sólidos, 11–12 texto) e "os passos
  11 e 12 têm garantidos Lc 60 e Lc 90 de contraste APCA sobre o passo 2 da mesma escala"
  (radix-ui.com/colors/docs/palette-composition/understanding-the-scale). Observação: APCA não é
  o método normativo do WCAG 2.2, que usa a razão de luminância relativa (1.4.3); uma garantia
  APCA não é, por si, conformidade WCAG 2.2.
- **MUI**: deriva `light`/`dark` deslocando a luminância com `tonalOffset` (0.2 por padrão) e
  escolhe `contrastText` pelo `contrastThreshold`, cujo padrão é **3:1**; para 1.4.3 a própria
  documentação sugere 4.5 e adverte que o parâmetro "pode produzir resultados contraproducentes"
  (https://mui.com/material-ui/customization/palette/). Isto é, o padrão da MUI não garante AA
  para texto normal.
- **shadcn/ui**: a convenção `primary` / `primary-foreground` torna o par explícito, mas os
  valores são escolhidos à mão; não há verificação (https://ui.shadcn.com/docs/theming).

### 1.4 Tema como configuração fechada (Radix Themes)

`Theme` aceita poucos parâmetros com **valores enumerados**: `accentColor` e `grayColor` (nomes
de escalas), `radius` (`none`, `small`, `medium`, `large`, `full`), `scaling` de 90% a 110%,
`panelBackground`, aparência clara/escura (https://www.radix-ui.com/themes/docs/theme/overview).
É o mais próximo, entre os estudados, de "tema como intenção": um vocabulário pequeno e fechado,
cada valor com efeito conhecido.

### 1.5 Formato de intercâmbio (DTCG 2025.10)

O *Format Module 2025.10* é um Final Community Group Report ("considerado estável"; não é
padrão W3C nem está na trilha de padrões). Define `$value`, `$type`, `$description`,
`$extensions` (ferramentas "DEVEM preservar extensões que não entendem"), tipos atômicos
(color com `colorSpace` e `components`, dimension, fontFamily, fontWeight, duration,
cubicBezier, number), compostos (strokeStyle, border, transition, shadow, gradient,
typography), grupos com herança de `$type` e aliases `{grupo.token}`
(https://www.designtokens.org/tr/2025.10/format/). O rascunho seguinte avisa para não
implementar a versão em rascunho (https://www.designtokens.org/tr/drafts/format/).

## 2. Dark mode, densidade e variants

- **Dark mode** não é inverter cores: no M3 é o mesmo esquema com `isDark` (outros tons das
  mesmas paletas); na Ant, um algoritmo; no shadcn e no Chakra, os mesmos nomes semânticos
  redefinidos sob um seletor/condição (`.dark`, `_dark`). Em todos, **os componentes não
  mudam**, só os valores dos papéis.
- **Densidade**: a Ant a trata como algoritmo (`compactAlgorithm`) sobre `sizeStep`/
  `controlHeight`; o Radix, como `scaling`. É uma dimensão ortogonal à cor.
- **Variants** (solid/soft/outline/ghost; tamanhos): recipes do Chakra (`cva`) e slot recipes
  (`sva`) por parte do componente (https://chakra-ui.com/docs/theming/overview). Uma variant é
  uma **escolha enumerada de papéis**, não CSS livre.

## 3. O que o Germanio tem hoje (verificado no código)

1. **Dois renderizadores com dois temas independentes.**
   - SPA legado (`renderizador.go`): carrega `https://cdn.tailwindcss.com` e configura
     `tailwind.config` com `primary`, `secondary`, `accent` e a fonte. A Tailwind diz que o Play
     CDN "é projetado só para desenvolvimento e não é destinado a produção"
     (https://tailwindcss.com/docs/installation/play-cdn).
   - Páginas de intenção (`paginas.go`, `layoutTpl`): tokens fixos em `:root` (`--fg`,
     `--muted`, `--line`, `--bg`, `--soft`, `--accent`, `--ok`, `--bad`) com redefinição sob
     `prefers-color-scheme: dark`. **Não leem `ast.Theme`**: `tema azul` não muda nada nelas.
2. **Campos do tema sem efeito.** No SPA, só `Primary`, `Secondary`, `Accent`, `Font` e `Dark`
   chegam ao HTML; `Radius`, `Style`, `Background`, `CardBg`, `TextColor` e `Sidebar` são
   preenchidos por `applyThemeDefaults` e não usados; `styleVariantCSS` devolve `""`. Os "4
   estilos" (glassmorphism/flat/neumorphism/minimal) citados no CLAUDE.md não existem de fato.
3. **`ColorName`** mapeia 15 nomes (PT e EN) para hex que coincidem com os tons 500 da paleta
   clássica da Tailwind v3 (por exemplo `azul` `#3b82f6`). O SPA pinta botões com
   `bg-primary … text-white`. Contraste calculado (fórmula WCAG) do branco sobre cada cor:
   azul 3.68, verde 2.28, vermelho 3.76, laranja 2.80, rosa 3.53, amarelo 1.92, ciano 2.43,
   esmeralda 2.54, âmbar 2.15, índigo 4.47, roxo 4.23; só cinza (4.83) e violeta (5.70)
   passam 4.5:1. **`tema azul` produz botões que falham o 1.4.3 (AA)**.
4. **Páginas de intenção, modo escuro**: botão com texto `#fff` sobre `--accent` `#a371f7`
   = 3.35:1 (falha 1.4.3 para o texto de 15px). A borda de campos `--line` `#d0d7de` sobre
   branco = 1.45:1; se a borda é o que identifica o campo, falha o 1.4.11 (3:1). No claro,
   `--accent` `#6e40c9` com branco = 6.48 e `--muted` sobre branco = 5.25 passam.
5. **`CustomCSS`** injeta CSS bruto do usuário; o núcleo estrito aceita `borda arredondada 12
   px` (`germanio.go`), que é exatamente "CSS em português".
6. **Presets** (`moderno`, `claro`, `simples`, `elegante`, `corporativo`) são tabelas de hex
   escritas à mão, sem papéis semânticos, sem garantia de contraste e sem relação entre o
   preset e o que o renderizador de intenção desenha.

## 4. Proposta: intenção visual → tokens determinísticos

Pipeline (espelha seed → map → alias da Ant e source → palettes → roles do M3):

```text
.ge (poucas palavras, valores fechados)
  → Seed (struct Go tipada, com a origem de cada valor: arquivo:linha ou "padrão")
  → algoritmos puros e versionados (cor, forma, densidade, modo)
  → papéis semânticos em pares (superfície / texto-sobre)
  → tokens de componente (botão, campo, tabela, aviso, foco)
  → CSS custom properties + (opcional) export DTCG 2025.10
```

### 4.1 Tabela fechada (candidata)

| Intenção (candidata) | Valores aceitos | Seed afetado | Padrão |
| --- | --- | --- | --- |
| `estilo` | `simples`, `moderno`, `elegante`, `corporativo` | variante de esquema (análoga a TONAL_SPOT/NEUTRAL/VIBRANT…), família tipográfica, raio base, sombra | `simples` |
| `destaque` | nomes de `ColorName` (e só eles no nível padrão) | cor-fonte | a do `estilo` |
| `cantos` | `retos`, `suaves`, `arredondados` | raio base (por exemplo 0 / 6 / 12 px) e sua escala | do `estilo` |
| `densidade` | `compacta`, `normal`, `confortável` | passo de espaçamento, altura de controle | `normal` |
| `modo` | `claro`, `escuro`, `automático` | `isDark` ou os dois conjuntos com `prefers-color-scheme` | `automático` |
| `contraste` | `normal`, `alto` | `contrastLevel` 0 / 1 | `normal` |

Regras (propostas): cada palavra é um **seed**, nunca uma propriedade CSS; uma palavra fora da
tabela é erro que lista os valores válidos (como as seções de dados); o mesmo seed com dois
valores é conflito que mostra as duas origens (mesma regra de fusão de `docs/INTENCAO.md`);
hex e pixels só no nível técnico (arquivo de tema de um adaptador ou `tema` técnico), nunca no
nível padrão.

### 4.2 Geração da paleta com garantia

1. Converter a cor-fonte para HCT; gerar as paletas tonais (primária, secundária, neutra,
   neutra-variante, erro) conforme a variante do `estilo`.
2. Atribuir papéis por **tom**, com as diferenças mínimas do MCU: texto normal sobre superfície
   Δtom ≥ 50 (≥ 4.5:1); bordas de controle, anel de foco e ícones Δtom ≥ 40 (≥ 3:1, para 1.4.11).
3. **Verificação independente**: calcular a razão WCAG 2.2 de cada par efetivamente emitido
   (a garantia do HCT é um argumento; o teste é a fórmula normativa). Se um par falhar (por
   arredondamento do gamut sRGB, por exemplo), afastar o tom e registrar em `ge explain`.
4. Nunca escolher `text-white` fixo: o texto-sobre é sempre um papel derivado (`sobre-destaque`).
5. O algoritmo é **versionado**: a mesma entrada produz os mesmos hex numa versão; mudar o
   algoritmo é mudança registrada (o autor não vê cores mudarem por atualização silenciosa).
   Um teste de ouro fixa a saída de cada cor de `ColorName` × modo × contraste.

Com isso `destaque amarelo` deixa de ser um erro de acessibilidade: o botão usa o tom 40 da
paleta amarela (um âmbar escuro) com texto claro, ou o tom 80–90 com texto escuro, conforme a
variante; o autor não precisa saber qual.

### 4.3 Papéis semânticos mínimos

Pares no estilo shadcn/M3, suficientes para as páginas de intenção atuais: `fundo`/`texto`,
`superfície`/`texto-superfície`, `suave`/`texto-suave`, `destaque`/`sobre-destaque`,
`perigo`/`sobre-perigo`, `sucesso`/`sobre-sucesso`, `aviso`/`sobre-aviso`, `borda`,
`borda-controle`, `foco`. Os nomes internos são de implementação (CSS custom properties
geradas); o autor nunca os escreve.

### 4.4 `ge explain` do tema

Mostrar a cadeia: `destaque azul` (frontend/tema.ge:3) → cor-fonte `#3b82f6` → HCT (h, c, t) →
`destaque` = tom 40 `#…` → `sobre-destaque` = tom 100 → contraste 5.x:1 (AA texto). Isso torna a
inferência explicável, como os fatos de dados.

## 5. Tensões e contradições com o estado atual

- `tema azul` hoje define `Primary` = hex e o SPA assume texto branco: é a abordagem "cor como
  valor", que a pesquisa contradiz; a cor deve ser **semente**, e o texto-sobre, derivado.
- Os presets escrevem `Background`, `CardBg`, `TextColor` à mão: é a camada Map sendo escrita
  pelo autor do preset, sem garantia; e boa parte não é consumida.
- `CustomCSS` e `borda arredondada 12 px` expõem CSS no domínio; conflitam com a regra "nada de
  CSS em português".
- As páginas de intenção, que são o frontend normativo, ignoram o tema; qualquer sintaxe de
  tema nova deve mirar **um** gerador de tokens usado por todos os renderizadores.

## Para o Germanio

**ADOTAR**

- Seed → algoritmo → papéis → componente (Ant Design, M3). Resolve: os presets escrevem valores
  derivados à mão e o renderizador de intenção não os usa. Afeta `compiler/ast/ast.go` (`Theme`
  vira seed), um pacote novo de geração de tokens em `runtime/` e `runtime/servidor/paginas.go`
  (`layoutTpl` passa a consumir variáveis geradas).
- Garantia de contraste por diferença de tom HCT (Δ50 → 4.5:1, Δ40 → 3:1, `hct.ts`) **mais**
  verificação pela fórmula WCAG de cada par emitido. Resolve: `tema azul`/`verde`/`amarelo`
  produzem botões abaixo de 4.5:1; o modo escuro das páginas tem botão a 3.35:1 e borda a
  1.45:1. Teste de ouro sobre `ColorName` inteiro.
- Pares superfície/texto-sobre como papéis (shadcn, M3). Resolve: `text-white` fixo em
  `renderizador.go`.
- Vocabulário fechado e enumerado para forma e densidade (Radix Themes `radius`/`scaling`, Ant
  `compactAlgorithm`). Resolve: `Radius` livre em px e `borda arredondada 12 px`.

**ADAPTAR**

- `contrastLevel` do M3 como uma palavra de dois valores (`contraste alto`), não um número de -1
  a 1. Resolve: pedido de acessibilidade sem conceito técnico.
- Export opcional para DTCG 2025.10 (`ge explain --tokens` ou similar). Resolve: interoperar com
  Figma e ferramentas de design sem criar formato próprio; não é necessário para o leigo.
- Variantes de esquema do M3 como a implementação interna de `estilo`; o autor escolhe
  `moderno`, não `TONAL_SPOT`.

**EVITAR**

- Tailwind Play CDN em produção (a própria Tailwind proíbe) e, de modo geral, depender de um
  framework CSS em runtime: o Germanio conhece o modelo e pode emitir CSS pequeno e estático
  (como o `zeroRuntime` da Ant). Afeta `renderizador.go`.
- Escolher texto por limiar 3:1 (padrão MUI): insuficiente para texto normal.
- Hex, px, `css` bruto e nomes de propriedade CSS no nível padrão (`CustomCSS`,
  `borda arredondada 12 px`).
- Garantias só em APCA (Radix) como se fossem conformidade WCAG 2.2.
- Mais presets escritos à mão: cada preset novo deve ser só um seed.

**INVESTIGAR**

- Porte de HCT/CAM16 para Go: portar o MCU (licença Apache-2.0, a confirmar) ou avaliar os
  portes de terceiros encontrados; o algoritmo precisa ser determinístico e testável sem CGO.
- OKLCH (Tailwind, shadcn) versus HCT: OKLCH é nativo em CSS, mas não traz a garantia
  tom → contraste; comparar a manutenção de contraste das duas abordagens com testes.
- Se `estilo` deve continuar a existir: sem semântica formal (o que `elegante` muda,
  exatamente?), é uma palavra vaga; talvez só `destaque`, `cantos`, `densidade` e `modo` tenham
  significado verificável. Resolve o risco de keyword sem semântica (skill, seção 19b).
- Onde mora a declaração de tema na árvore de arquivos (`frontend/tema.ge`?) e como se funde
  entre arquivos (regras de `Fusão e conflitos`).
