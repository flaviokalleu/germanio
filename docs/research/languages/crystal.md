# Estudo dirigido: Crystal — sintaxe de Ruby com inferência global e o preço na compilação

**Data:** 2026-09-28
**Status:** pesquisa, sem força normativa (a norma é `docs/INTENCAO.md`).
**Pergunta principal:** o que custa inferir "tudo" sobre o programa inteiro, e o que isso diz sobre
a inferência do Germanio (resolver que funde blocos de vários arquivos) e sobre um futuro LSP?
Secundária: fibers e canais como modelo de concorrência.

## Fontes consultadas

- Referência oficial, "Type inference" — https://crystal-lang.org/reference/latest/syntax_and_semantics/type_inference.html
- Referência oficial, "Concurrency" — https://crystal-lang.org/reference/latest/guides/concurrency.html
- Issue #1824 "The next step" (Ary Borenszweig/asterite, 2015-10-26) — https://github.com/crystal-lang/crystal/issues/1824
- Issue #10568 "Status of incremental compilation in 2021" — https://github.com/crystal-lang/crystal/issues/10568 (só o pedido; as respostas não foram carregadas)
- Fórum, "Propose: Incremental compilation" (jan. 2024) — https://forum.crystal-lang.org/t/propose-incremental-compilation/6302
- Issue #2390 (especificação semiformal do novo algoritmo de inferência) — https://github.com/crystal-lang/crystal/issues/2390 (vista só pelo título na busca)

---

## Matriz (compacta)

| Item | Crystal | Relevância para o Germanio |
|---|---|---|
| Objetivo original | "Rápido como C, bonito como Ruby": sintaxe de Ruby, tipos estáticos, código nativo via LLVM | — |
| Filosofia | "require as few type restrictions as possible" (referência) | a mesma ambição do princípio de subtração |
| Sintaxe | Ruby quase literal: `def`/`end`, blocos, classes reabríveis, macros | ver seção "Sintaxe" |
| Gramática / Parser | Parser à mão | — |
| AST / IR | AST → análise semântica global → LLVM IR | — |
| Análise semântica | Inferência de **programa inteiro**: tipos dos métodos dependem de como são chamados; métodos instanciados por combinação de tipos de argumentos | análogo ao resolver do Germanio, que também olha todos os arquivos |
| Sistema de tipos | Estático, com **uniões** inferidas (`Int32 \| Nil`) e checagem de `Nil` em compilação | ver "Para o Germanio", item 4 |
| Type inference | Local para variáveis e global para métodos; desde 2015–2016 exige tipo (ou atribuição simples em `initialize`) para variáveis de instância, de classe e globais (issue #1824; referência) | — |
| Compiler | Um compilador; sem compilação incremental da análise semântica | ver "Performance" |
| Runtime | GC Boehm (não verificado nesta passagem), fibers, event loop | — |
| Memory management | GC | igual em espírito ao Germanio (usuário não administra memória) |
| Standard library | Rica em web (HTTP server, JSON) | — |
| Package manager | Shards, com `shard.lock` | — |
| Formatter | `crystal tool format` oficial | — |
| Language server | Não oficial (Crystalline); limitado porque a análise precisa do programa inteiro (não verificado nesta passagem) | risco direto para o LSP do Germanio |
| Evolution / compat | 1.0 em 2021; compatibilidade na série 1.x (não verificado nesta passagem) | — |

### Principais acertos
- Legibilidade de Ruby com erros de tipo em compilação; `Nil` não é um valor que entra em
  qualquer tipo, é um membro de união que o compilador obriga a tratar.
- Recuo honesto em 2015: ao ver que a inferência global inviabilizava compilação incremental,
  a equipe tornou obrigatório declarar o tipo das variáveis de instância, aceitando menos
  inferência por um compilador viável (issue #1824). A regra ficou ensinável: tipo explícito ou
  atribuição óbvia no `initialize`.
- Concorrência simples por padrão: fibers cooperativas (pilha começa em 4 KB), canais no estilo
  CSP, um só thread por padrão; multithread é opt-in (referência, Concurrency).

### Principais problemas
1. **Inferência global recomeça do zero.** "The global type inference algorithm always has to
   start from scratch" (issue #1824). O motivo: "because their type might change at any point in
   the program, they can affect the type of any method that uses them. This means that local
   type inference for a method can't be done" (idem).
2. **Compilação incremental continua ausente anos depois.** Em 2021 um usuário ainda pedia o
   recurso (issue #10568); em 2024, no fórum, a dificuldade apontada continua sendo a checagem
   dos métodos genéricos por uso: "what ensures that whatever passed to the `sum` method has a
   `+` operator ... if we don't continuously recheck the body?" (fórum, 2024). A referência
   oficial resume: sem tipos declarados, otimizações e compilação incremental ficam "nearly
   impossible".
3. **Classes reabríveis e macros** somam-se à inferência global: qualquer arquivo pode mudar o
   que um tipo é.

### Complexidade acumulada
Pouca na sintaxe; concentrada no compilador e no tempo de compilação, que cresce com o
programa inteiro.

---

## Sintaxe (perguntas obrigatórias)

| Pergunta | Crystal | Germanio hoje |
|---|---|---|
| Dados | `class`/`struct` com `property`, tipos das variáveis de instância declarados | `tem` com campos e restrições; tipo inferido do nome/restrição |
| Funções | `def` sem tipos obrigatórios nos parâmetros | ausentes do domínio |
| Módulos / imports | `require` de arquivos; classes reabríveis em qualquer arquivo | arquivos fundidos; blocos do mesmo dado fundidos pelo resolver |
| Relações | referências a objetos | nomes resolvidos |
| Estado | mutável | estados declarados |
| Erros | exceções; `Nil` em uniões checado | diagnostics em 4 partes |
| Concorrência | `spawn` + `Channel` | runtime administra; `paralelo()` na lógica |
| Null | `Nil` explícito em união, verificado | `obrigatório` explícito; campo opcional pode faltar |
| Boilerplate | baixo | derivado |
| Projetos grandes | tempo de compilação cresce com o todo | parse+resolve do GitLab: 8–10 ms (AUDITORIA §2.1) |

Custo cognitivo: o Crystal parece Ruby, mas o leitor precisa saber que o tipo de um método
depende de todos os lugares que o chamam — uma regra implícita e não local. O Germanio tem o
risco equivalente: um fato sobre `projetos` pode vir de qualquer arquivo.

## Performance (comparação com `performance/AUDITORIA.md`)

- **Hoje o Germanio não tem o problema do Crystal.** Parse e resolve do GitLab (687 linhas, 14
  entidades) levam 8,4–10,0 ms; `Carregar` completo 267 ms, dominado por criar tabelas com fsync
  (AUDITORIA §2.1–2.2). O ponto quente do parse é `foldWord` recriando um `strings.NewReplacer`
  por palavra (83% da memória do parse), um custo constante e corrigível, não uma consequência de
  inferência global.
- **O problema aparece no LSP.** Um editor precisa rechecar a cada tecla. Se um bloco em
  `a.ge` pode alterar a entidade declarada em `b.ge` (fusão no `resolver.go`), a checagem de
  qualquer arquivo depende de todos. Com projetos pequenos isso cabe no orçamento; o Crystal
  mostra que a decisão de linguagem, e não a otimização do compilador, é que determina se o
  incremental é possível depois.
- **Fibers vs. goroutines.** O modelo do Crystal (fibers leves, um thread por padrão) é
  semelhante às goroutines que o runtime do Germanio já usa, mas o usuário do Germanio não vê
  nenhum dos dois. A AUDITORIA §3.4 registra concorrência sem limite e sem cancelamento; o
  Crystal não oferece solução para isso (canais não são backpressure por si).

## Para o Germanio

| # | Classe | Lição | Problema do Germanio | Arquivo | O que remove da cabeça do programador |
|---|---|---|---|---|---|
| 1 | ADOTAR | Localidade da inferência como regra de linguagem: tudo o que se infere sobre uma entidade vem do próprio bloco ou de fatos declarados, nunca de "como ela é usada" em outro lugar | o resolver funde blocos do mesmo dado vindos de vários arquivos | `compiler/parser/resolver.go`, `docs/INTENCAO.md` | procurar em todos os arquivos a origem de um fato |
| 2 | ADAPTAR | Tornar a fusão entre arquivos explícita e rastreável (o Crystal pagou por classes reabríveis): `ge explain` já mostra origem; o resolver pode avisar quando uma entidade é completada em mais de um arquivo | fusão silenciosa | `compiler/parser/resolver.go`, `tooling/explicar` | — |
| 3 | ADOTAR | Desenhar o grafo de dependências entre arquivos agora, antes do LSP (o que o Crystal não conseguiu acrescentar depois) | não há LSP; parse é barato hoje | futuro `tooling/lsp`, `compiler/parser` | esperar o editor |
| 4 | ADAPTAR | Ausência como caso verificado (uniões com `Nil`): regra/cálculo que lê campo opcional deve tratar a ausência, com diagnóstico | campo opcional ausente vira `nil` num `map[string]any` e segue em silêncio na lógica | `runtime/interpreter`, `compiler/parser/resolver.go` | lembrar que um campo pode faltar |
| 5 | ADAPTAR | Quando a inferência não é óbvia para um humano, peça a declaração (critério da referência: "not obvious for a human reading the code") | usar como critério do que o Germanio infere vs. pergunta | `docs/INTENCAO.md`, `skills/germanio-simplicity/SKILL.md` (só referência) | — |
| 6 | EVITAR | Macros e reabertura arbitrária de tipos | manter a tabela de seções fechada | `compiler/parser/hierarquia.go` | — |
| 7 | INVESTIGAR | Corrigir `foldWord` (replacer único) e medir parse+resolve em 10x o GitLab para saber quando o custo do todo passa do orçamento de um LSP (~50 ms) | sem medição acima de 687 linhas | `compiler/parser/intencao.go`, `bench/` | — |
