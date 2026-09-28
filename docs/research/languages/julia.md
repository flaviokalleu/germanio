# Estudo dirigido: Julia — multiple dispatch, latência do JIT e ambientes reprodutíveis

**Data:** 2026-09-28
**Status:** pesquisa, sem força normativa (a norma é `docs/INTENCAO.md`).
**Pergunta principal:** o que o "time to first plot" ensina sobre latência de startup num
runtime que compila tarde; como o multiple dispatch compõe código genérico (e o que custa);
e o que o par Project.toml/Manifest.toml ensina sobre reprodutibilidade?

## Fontes consultadas

- Manual, "Methods" (dispatch, ambiguidade, especialização) — https://docs.julialang.org/en/v1/manual/methods/
- Blog oficial, "Julia 1.9 Highlights" (cache de código nativo), 2023 — https://julialang.org/blog/2023/04/julia-1.9-highlights/
- Blog oficial, "Julia 1.11 Highlights", 2024 — https://julialang.org/blog/2024/10/julia-1.11-highlights/
- Blog oficial, "Analyzing sources of compiler latency in Julia: method invalidations", 2020 — https://julialang.org/blog/2020/08/invalidations/
- Pkg, "Project.toml and Manifest.toml" — https://pkgdocs.julialang.org/v1/toml-files/
- Stefan Karpinski, "The Unreasonable Effectiveness of Multiple Dispatch", JuliaCon 2019 — https://pretalx.com/juliacon2019/talk/BCYWZJ/ (resumo da palestra; vídeo não assistido)

---

## Matriz (compacta)

| Item | Julia | Relevância para o Germanio |
|---|---|---|
| Objetivo original | Computação científica com a facilidade de uma linguagem dinâmica e a velocidade de C (o "problema das duas linguagens") | — |
| Filosofia | Tipos opcionais nas assinaturas, genéricos por padrão, desempenho por especialização | — |
| Sintaxe | Parecida com MATLAB/Python: `function ... end`, `struct`, macros `@` | ver "Sintaxe" |
| Gramática / Parser | Parser em Julia (JuliaSyntax.jl, padrão desde 1.10) (não verificado nesta passagem) | — |
| AST / IR | AST → IR tipada → LLVM → nativo, por combinação de tipos na primeira chamada | — |
| Análise semântica | Inferência de tipos para otimizar, não para rejeitar programas | o Germanio infere para decidir e explicar |
| Sistema de tipos | Dinâmico, com hierarquia nominal e tipos paramétricos; dispatch sobre todos os argumentos | — |
| Compiler/interpreter | JIT por método e por assinatura concreta; especialização automática (manual, Methods) | ver "Performance" |
| Runtime | GC geracional, tarefas (corrotinas), threads | — |
| Memory management | GC | igual em espírito ao Germanio |
| Package manager | Pkg: `Project.toml` (dependências diretas e `[compat]`) + `Manifest.toml` ("an absolute record of the state of the packages in the environment", gerado, "should never be modified manually") | ver "Para o Germanio" |
| Formatter / Linter | Não oficiais (JuliaFormatter) (não verificado nesta passagem) | — |
| Language server | LanguageServer.jl (não verificado nesta passagem) | — |
| Diagnostics | `MethodError` de ambiguidade lista os candidatos e sugere o método que desfaz: "Possible fix, define `g(::Float64, ::Float64)`" (manual) | modelo de fix-it concreto |
| Evolution | 1.x com LTS (1.10 substituiu 1.6 como LTS em 2024) | — |

### Principais acertos
- **Reuso entre pacotes.** Karpinski atribui ao dispatch múltiplo externo (métodos não vivem
  "dentro" de classes) o compartilhamento incomum de tipos e algoritmos genéricos entre pacotes
  (JuliaCon 2019).
- **Ambiguidade é erro, não escolha arbitrária** — e o erro diz qual definição a resolve.
- **Reprodutibilidade de fábrica**: o manifesto é gerado, completo (dependências diretas e
  indiretas) e basta `Pkg.instantiate` para recriar o ambiente.
- **Latência tratada como problema de primeira classe e medida publicamente.** O 1.9 passou a
  guardar código nativo nos caches de pré-compilação: CSV de 11,66 s para 0,08 s, DataFrames de
  17,39 s para 0,38 s, ModelingToolkit de 73,53 s para 4,81 s (blog 1.9). Custo declarado:
  pré-compilação 10–50% mais lenta e caches maiores. O 1.11 reduziu `julia -e "1+1"` de ~113 ms
  para ~92 ms (blog 1.11).

### Principais problemas
1. **Time to first plot.** Por anos, a primeira execução de uma operação comum levou segundos a
   minutos porque o código só era compilado na primeira chamada.
2. **Invalidações.** Um método novo invalida código já compilado que supunha outra coisa: com o
   SIMD carregado no Julia 1.5, "time to second plot was nearly as bad as time to first plot"
   (blog 2020). É o preço do dispatch aberto: qualquer pacote pode acrescentar um método a
   qualquer função.
3. **Poder demais no dispatch** para quem lê: saber qual método roda exige conhecer todos os
   métodos carregados.

### Complexidade acumulada
Moderada na linguagem; grande no compilador e no ecossistema de ferramentas de latência
(PrecompileTools, SnoopCompile, PackageCompiler).

---

## Sintaxe (perguntas obrigatórias)

| Pergunta | Julia | Germanio hoje |
|---|---|---|
| Dados | `struct` imutável por padrão, `mutable struct` | `tem` |
| Funções | funções genéricas com vários métodos por tipos | ausentes do domínio |
| Módulos / imports | `module`, `using`/`import`, `export`, `public` (1.11) | `importar "backend"` (pasta inteira, ordem alfabética) e `importar produtos do backend` (`docs/INTENCAO.md`) |
| Relações | referências | nomes resolvidos |
| Estado | mutável por escolha | estados declarados |
| Erros | exceções | diagnostics na compilação |
| Concorrência | `@spawn`, `Channel`, threads | runtime administra |
| Null | `nothing`/`missing` (dois conceitos) | `obrigatório`; um único conceito de ausência |
| Boilerplate | baixo | derivado |
| Projetos grandes | dispatch dificulta saber o que roda | `ge explain` com origem |

Custo cognitivo: `nothing` e `missing` são dois nulos com semânticas diferentes (ausência vs.
dado faltante estatístico); o Germanio deve manter um só conceito de ausência.

## Performance e startup (comparação com `performance/AUDITORIA.md`)

- **Onde o Germanio está no mesmo ponto em que o Julia estava.** O `germanio build` embute os
  fontes `.ge` e **reanalisa a cada partida**; "não há AST nem modelo pré-resolvido" (AUDITORIA
  §4). É o análogo do Julia antes do 1.9: o trabalho de compilação refeito a cada execução. Os
  números são pequenos (parse+resolve do GitLab 8–10 ms; `Carregar` 267 ms com banco novo,
  22–31 ms com banco existente, §2.2), mas o princípio vale: o que é determinístico e não muda
  entre execuções pode ser cacheado no artefato.
- **O custo que o Julia assumiu é o que o Germanio evita.** O Germanio não compila código de
  usuário para nativo: interpreta handlers genéricos sobre `ast.App`, e as leituras custam
  1,2–1,6x o Go direto (§2.4). Não há "primeira chamada lenta" por JIT; a latência de startup
  vem de init de pacotes (7,2 ms, 213 pacotes, §2.2) e criação de tabelas.
- **Invalidação.** O equivalente no Germanio é o hot reload, que re-executa o processo inteiro
  (`runtime/hotreload.go`) e em produção custa CPU sem requests (§3.6). O Julia mostra que
  "recompilar tudo quando algo muda" precisa de um grafo de dependências para ser barato.
- **Números.** O `Int` do Julia tem overflow com wraparound definido (não verificado nesta
  passagem) — comportamento definido, ao contrário do C. O Germanio usa `float64` para todos os
  números da lógica (`runtime/interpreter/interpreter.go`); ver `c.md`, item 7.

## Para o Germanio

| # | Classe | Lição | Problema do Germanio | Arquivo | O que remove da cabeça do programador |
|---|---|---|---|---|---|
| 1 | ADOTAR | Cachear o resultado determinístico da compilação no artefato (package images): o `germanio build` embute o `ast.App` resolvido e verificado, não os fontes para reanalisar | reparse a cada partida e extração para diretório temporário (AUDITORIA §4) | `cli/cli.go` (build), `compiler/ast` (serialização) | — (startup e superfície menores) |
| 2 | ADOTAR | Medir e publicar a latência de startup por versão, como o Julia faz nos highlights | startup medido só na auditoria | `bench/`, `CHANGELOG.md` | — |
| 3 | ADOTAR | Ambiguidade é erro com a correção concreta ("Possible fix, define ...") | dois blocos/regras que se aplicam ao mesmo caso devem falhar com a frase que desfaz | `compiler/parser/resolver.go`, `compiler/diagnostics` | adivinhar qual regra venceu |
| 4 | ADAPTAR | Par manifesto declarado + manifesto gerado e travado (Project/Manifest) quando o Germanio tiver dependências externas (adaptadores, bibliotecas `.ge`) | não há gerenciador de pacotes nem lockfile; `importar` só vê pastas locais | futuro `cli`, `docs/INTENCAO.md` › importar | lembrar versões; "funciona na minha máquina" |
| 5 | ADAPTAR | `[compat]` como faixa declarada da versão do Germanio que o projeto aceita | nenhum campo diz qual versão da linguagem um `.ge` espera | `docs/INTENCAO.md`, `cli` | — |
| 6 | EVITAR | Extensão aberta (qualquer pacote acrescenta método a qualquer função) no domínio | o `.ge` não deve permitir que um arquivo mude o comportamento de uma capability de outro sem aparecer no `ge explain` | `compiler/parser/resolver.go` | saber tudo o que foi carregado |
| 7 | EVITAR | Dois conceitos de ausência (`nothing`/`missing`) | manter só "ausente" | `docs/INTENCAO.md` | — |
| 8 | INVESTIGAR | Hot reload incremental por grafo de dependências em vez de re-exec | re-exec do processo; custo em produção (§3.6) | `runtime/hotreload.go` | — |
