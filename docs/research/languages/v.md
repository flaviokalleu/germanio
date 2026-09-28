# Estudo dirigido: V — promessa de simplicidade, compilação rápida e o custo de prometer antes de entregar

**Data:** 2026-09-28
**Status:** pesquisa, sem força normativa (a norma é `docs/INTENCAO.md`).
**Pergunta principal:** o que o V acerta (simplicidade, compilação rápida, C legível como saída)
e o que a distância documentada entre as promessas e a implementação ensina ao lançamento do
Germanio?
**Cuidado de método:** as críticas abaixo vêm de revisões públicas com testes reproduzíveis,
datadas; o V mudou desde então. Cada afirmação traz a data da fonte. Não se afirma aqui o
estado atual de nenhum item além do que o README e a documentação oficiais dizem hoje.

## Fontes consultadas

- README oficial — https://github.com/vlang/v (lido em 2026-09-28)
- Documentação oficial, "Memory management" — https://docs.vlang.io/memory-management.html
- Xe Iaso, "V is for Vaporware", 2019-06-23 (primeira versão pública) — https://xeiaso.net/blog/v-vaporware-2019-06-23/
- mawfig, "V Language Review (2022)", 2022-06-18 — https://mawfig.github.io/2022/06/18/v-lang-in-2022.html
- Resposta dos mantenedores à revisão de 2022 — https://github.com/vlang/v/discussions/15614
- Existência de uma revisão de 2023 do mesmo autor, discutida em https://news.ycombinator.com/item?id=39492680 (não lida)

---

## Matriz (compacta)

| Item | V | Relevância para o Germanio |
|---|---|---|
| Objetivo original | Linguagem de sistemas simples, rápida de compilar, próxima do Go, com saída em C | — |
| Filosofia | "can be learned over the course of a weekend"; uma forma de fazer cada coisa; imutável por padrão; "no null, no globals, no undefined behavior (wip)" (README) | alvo parecido em simplicidade; público diferente |
| Sintaxe | Parecida com Go: `fn`, `struct`, `mut` explícito, `?`/`!` para opção e erro | ver seção "Sintaxe" |
| Gramática / Parser | Parser à mão, sem especificação formal separada (não verificado nesta passagem) | o Germanio tem `SPEC.md` e `INTENCAO.md` |
| IR / Compiler | Principal backend "compiles to human-readable C" (README); também backends nativo e JS; tcc para compilação rápida | a ideia de saída legível é boa; ver "Performance" |
| Sistema de tipos | Estático, com sum types, opção/resultado | — |
| Type inference | Local | — |
| Runtime | Pequeno; GC tracing por padrão | — |
| Memory management | GC por padrão; `-gc none` manual; `-prealloc` arena só para programas "short lived, single-threaded, batch-like"; `-autofree`: "Autofree is still WIP. Until it stabilises and becomes the default, please avoid using it" (docs oficiais) | quatro modelos por flag = quatro dialetos de runtime |
| Standard library | Ampla (HTTP, ORM, UI, gráficos) para o tamanho do projeto | amplitude cedo cria superfície sem manutenção |
| Package manager | `v install` / VPM | — |
| Formatter | `v fmt` oficial | igual ao `ge fmt` |
| Language server | v-analyzer (não verificado nesta passagem) | — |
| Diagnostics | não avaliado | — |
| Evolution process | Decisões concentradas no criador; promessa de "feature freeze" e nenhuma quebra depois do 1.0 (README) | compromisso sem data |
| Backward compatibility | Pré-1.0, quebras frequentes (não verificado nesta passagem) | — |

### Principais acertos
- Compilação rápida como requisito de projeto, não otimização tardia; README atual declara
  ≈110 mil linhas/s com Clang e ≈500 mil com tcc/nativo, **com o hardware especificado**.
- Uma forma canônica e formatter oficial desde cedo.
- Saída em C legível: o usuário pode inspecionar o que o compilador gerou.
- A documentação oficial hoje é franca sobre o autofree ("please avoid using it") e o README
  marca "no undefined behavior" como "(wip)".

### Principais problemas (documentados, datados)
| Promessa | Evidência pública | Data |
|---|---|---|
| "1,2 milhão de linhas/s" | arquivo de 1,2 milhão de linhas travou com "more than 50 000 statements in function main"; 50 mil linhas levaram mais de 2 s (Xe Iaso) | 2019 |
| memória segura / sem vazamentos | "hello world" vazava; o compilador vazava ~3,8 MB compilando o teste (Xe Iaso) | 2019 |
| "zero dependências", build de 400 KB | exigia compilador C, libcurl, glfw; ~3,7 MB em disco (Xe Iaso) | 2019 |
| "≈1 milhão de linhas/s" | 169 mil a 386 mil linhas/s em código simples, 5,7 mil com sum types complexos (mawfig) | 2022 |
| sem null, sem UB, imutável, sem globais | ponteiro nulo gerou segfault; overflow com sinal, divisão por zero e ponteiro pendente compilaram; imutabilidade contornada; global recriado via `const` (mawfig) | 2022 |
| autofree | vazava ao forçar alocação no heap (mawfig); hoje a doc oficial pede para não usar | 2022 / hoje |

A resposta dos mantenedores à revisão de 2022 reconheceu "a couple of simple type checker
bugs, which have been fixed" e qualificou o texto como "an anonymous attack on the language"
(discussão #15614). A discussão não mostra alteração do site em resposta.

### Complexidade acumulada
Amplitude antes de profundidade: UI, ORM, gráficos, vários backends e quatro modelos de
memória num projeto pré-1.0. Cada frente abre promessas que precisam ser mantidas.

---

## Sintaxe (perguntas obrigatórias)

| Pergunta | V | Germanio hoje |
|---|---|---|
| Dados | `struct` com campos `mut`/`pub` | `tem` com restrições |
| Funções | `fn`, métodos com receptor | ausentes do domínio |
| Módulos / imports | `module`/`import` por diretório | fusão de arquivos no resolver |
| Estado | imutável por padrão, `mut` explícito | estados declarados; mutação só por ações |
| Erros | `?T`/`!T`, `or { }` | diagnostics de compilação; erros de runtime com posição |
| Concorrência | `go`/`spawn`, canais | administrada pelo runtime |
| Null | promessa de ausência (contestada em 2022) | `obrigatório` explícito; ausência é valor definido |
| Boilerplate | baixo para uma linguagem de sistemas | derivado |
| Projetos grandes | não avaliado | `ge explain` |

Custo cognitivo: V exige menos conceitos que C ou Rust, mas ainda exige tipos, ponteiros e,
com `-gc none`, memória. Pior: a flag de memória muda o que é seguro, então o leitor precisa
saber com qual flag o programa roda.

## Performance e memória (comparação com `performance/AUDITORIA.md`)

- **Transpilar para C.** A AUDITORIA §5 conclui que "compilar para Go não corrigiria nenhum"
  dos maiores custos medidos (visibilidade O(tabela), log O(n²), fila sem limpeza, fsync). O V
  mostra o mesmo por outro ângulo: gerar C não impediu que os programas testados em 2022
  ficassem 2,9 a 4,6x mais lentos que o C escrito à mão (mawfig) — o backend não substitui o
  algoritmo nem o runtime.
- **Vários modelos de memória por flag** contra o Germanio, em que o usuário nunca administra
  memória: o Germanio está certo em ter um só modelo (GC do Go). O que o V ensina é que um
  modelo "automático sem GC" anunciado cedo vira dívida pública.
- **Números com hardware.** O README atual do V passou a publicar o hardware junto com a
  velocidade. O Germanio já exige isso: "makes no speed claims without such a reproducible
  comparison" (`README.md`, seção Benchmarks; `bench/README.md`).

## O que o V ensina sobre o lançamento do Germanio

1. **Cada frase do README é um teste.** As revisões de 2019 e 2022 não descobriram nada
   sofisticado: pegaram frases do site e escreveram o menor programa que as contradiz. Qualquer
   revisor fará o mesmo com o Germanio no lançamento.
2. **Onde o Germanio hoje se expõe da mesma forma:**
   - "secure by default" (`README.md:117`, `README.pt-BR.md:115`). A AUDITORIA registra a rota
     `/ws` aberta sem autenticação em todo app (`servidor.go:73`, `:110`), a API gerada
     `/_ge/api` sem rate limit, `http.Server` sem `ReadHeaderTimeout` e rotas de proxy/eval
     sempre registradas (§3.7, §4). Cada item é um "hello world que vaza" em potencial.
   - "deterministic: the same `.ge` file always means the same program" (`README.md`). É
     verificável e deve ter teste de propriedade (mesma entrada → mesmo `ast.App` e mesma saída
     de `ge explain`, byte a byte).
   - O `CLAUDE.md` interno ainda diz "20 languages", enquanto o README diz, corretamente, que a
     camada de intenção só entende português. Documentação interna defasada vira promessa
     pública quando alguém a cita.
3. **Marcar "(wip)" cedo, no lugar da promessa.** O V acabou marcando UB como wip e o autofree
   como "avoid"; o custo de reputação veio de ter anunciado antes. O README do Germanio já tem
   "Project status" e "known gaps" (`GERMANIO_GAPS.md`); o padrão é bom.
4. **Responder à crítica com a correção, não com o autor.** A resposta de 2022 atacou a
   motivação da revisão; o registro público ficou com a revisão.

## Para o Germanio

| # | Classe | Lição | Problema do Germanio | Arquivo | O que remove da cabeça do programador |
|---|---|---|---|---|---|
| 1 | ADOTAR | Toda afirmação pública tem um teste ou uma medição que a sustenta, com link | "secure by default" e "deterministic" sem teste nomeado | `README.md`, `README.pt-BR.md`, `docs/launch/LANCAMENTO.md` | desconfiar do que o site diz |
| 2 | ADOTAR | Corrigir antes do lançamento as exceções conhecidas à promessa de segurança | `/ws` sem autenticação, `/_ge/api` sem rate limit, sem `ReadHeaderTimeout` (AUDITORIA §3.7, §4) | `runtime/servidor/servidor.go`, `runtime/engine.go` | configurar segurança que foi prometida por padrão |
| 3 | ADOTAR | Teste de determinismo: mesmo `.ge` → mesmo modelo resolvido e mesmo `ge explain`, em execuções repetidas e ordem de arquivos diferente | a promessa do README não tem teste de propriedade nomeado (não verificado se existe) | `compiler/parser/resolver_test.go`, `tooling/explicar` | — |
| 4 | ADOTAR | Um só modelo de memória, sem flag | já é assim; registrar como decisão | `docs/INTENCAO.md` ou GEP | escolher modo de runtime |
| 5 | ADAPTAR | Publicar números só com hardware e comando (o V passou a fazer; o Germanio já exige) | manter a regra também em posts de lançamento | `bench/README.md`, `docs/launch/` | — |
| 6 | EVITAR | Amplitude antes de profundidade (UI, ORM, gráficos, vários backends pré-1.0) | capabilities como WhatsApp embutidas sempre, antes de estabilizar o núcleo | `runtime/whatsapp`, `runtime/engine.go` | — |
| 7 | EVITAR | Tratar revisão crítica como ataque | processo de resposta pública ainda não escrito | `docs/launch/LANCAMENTO.md` | — |
| 8 | INVESTIGAR | "Ler o que o compilador gerou" (C legível do V) como ferramenta de confiança: `ge explain` já mostra fatos; falta mostrar rotas, tabelas e permissões geradas | usuário não vê o SQL e as rotas derivadas | `tooling/explicar` | adivinhar o que o Germanio gerou |

Nota: o `CLAUDE.md` com "20 languages" foi observado, não editado (fora do escopo deste arquivo).
