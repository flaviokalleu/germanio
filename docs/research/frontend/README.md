# Pesquisa de frontend do Germanio

Data: 2026-09-28

> **Aviso.** Esta pasta é pesquisa. Nada aqui tem força normativa: a norma continua em
> [`docs/INTENCAO.md`](../../INTENCAO.md) e a filosofia em
> [`skills/germanio-simplicity/SKILL.md`](../../../skills/germanio-simplicity/SKILL.md).
> As propostas (seções de página, vocabulário de tema, contrato de formulário, tempo real)
> só passam a valer depois de uma GEP aceita ([`docs/gep/`](../../gep/README.md)), com a norma, a implementação e os testes
> alinhados na mesma unidade de trabalho (`AGENTS.md` › Documentation Gate). Nenhuma frase
> `.ge` citada aqui como proposta é sintaxe aceita.

## A questão

**Como uma pessoa descreve a interface que deseja e o Germanio a transforma numa aplicação Web
profissional, responsiva, acessível e eficiente?**

A pergunta não é "como escrever React, JSX, HTML ou CSS em português". Palavras como
`cor vermelho`, `margem 20`, `display flex` ou `borda arredondada 12 px` são CSS traduzido e
ficam fora do nível padrão. O autor comum não deve precisar de HTML, CSS, JS, DOM, Virtual
DOM, hooks, state managers, media queries, bundlers, SSR ou hydration; o usuário avançado
continua podendo chegar à Web por um nível técnico explícito.

O ponto de partida é o que o Germanio já tem e os frameworks não têm: o modelo resolvido
`ast.App` (entidades, campos, tipos, obrigatoriedade, relações, estados, permissões,
visibilidade). A interface deve ser derivada desse conhecimento, e não redeclarar o schema.

## Índice

Consolidação (comece por aqui):

- [GERMANIO_FRONTEND.md](GERMANIO_FRONTEND.md) — a arquitetura recomendada, a consolidação em
  ADOTAR / ADAPTAR / EVITAR / INVESTIGAR, as decisões que exigem proposta formal e as
  divergências conferidas na implementação atual.

Estudos de frameworks:

- [SVELTE.md](SVELTE.md) — declaração → compiler → DOM eficiente; análise como fase própria;
  avisos `a11y_*`; form actions do SvelteKit.
- [SOLID.md](SOLID.md) — reatividade fina, signals, grafo de leitura, publicação após commit.
- [VUE.md](VUE.md) — separação compiler/runtime, patch flags, Vapor, e por que o Germanio não
  precisa de três renderizadores.
- [REACT.md](REACT.md) — o modelo de componentes e o problema que cada mecanismo resolve;
  o estado de formulário como o único estado real.

Estudos transversais:

- [REACTIVITY.md](REACTIVITY.md) — síntese de reatividade e a proposta de aviso + rebusca
  autenticada + morph.
- [RENDERING.md](RENDERING.md) — o espectro de estratégias de renderização e a recomendação
  em camadas.
- [SSR_HYDRATION.md](SSR_HYDRATION.md) — SSR, hydration, ilhas, resumability, e o que o
  Germanio evita por construção.
- [COMPONENTS.md](COMPONENTS.md) — primitive × componente × bloco; a árvore semântica de
  página; formulários derivados do domínio.
- [DESIGN_SYSTEMS.md](DESIGN_SYSTEMS.md) — design tokens, cor semente, contraste por
  aritmética, tema como intenção.
- [ACCESSIBILITY.md](ACCESSIBILITY.md) — o que se deriva sozinho, APG, live regions, avisos
  a11y no `ge check`.
- [RESPONSIVE.md](RESPONSIVE.md) — layout intrínseco, container queries, tabela → cartões.

Pesquisa relacionada fora desta pasta:

- [`../performance/BENCHMARKING.md`](../performance/BENCHMARKING.md) — budgets de frontend
  (seção 12) e a suíte permanente.
- [`../performance/RUNTIMES.md`](../performance/RUNTIMES.md) — custo de `html/template` no
  binário (DCE relaxado).
- [`../sintaxe-hierarquica.md`](../sintaxe-hierarquica.md) — a base da sintaxe hierárquica,
  que a proposta de seções de página reutiliza.

## Regras desta pesquisa

- Fontes oficiais (documentação, repositórios, especificações W3C/WHATWG), com a URL ao lado
  de cada afirmação importante; o que não foi confirmado está marcado "(não verificado)".
- Cada estudo termina com "Para o Germanio", apontando o problema concreto que cada lição
  resolve e o arquivo afetado.
- Sem ranking de frameworks: a pergunta é qual decisão de arquitetura resolve um problema do
  Germanio, e não qual ferramenta é melhor.
- Afirmações sobre o código do Germanio foram conferidas nos commits `fc31daf` e `d9676b9`
  (correção do `/ws`, G65; ver GERMANIO_FRONTEND.md). Os estudos individuais citam G58/G59,
  renumerados depois para G62/G63.
