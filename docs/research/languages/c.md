# Estudo dirigido: C — especificação, comportamento indefinido e diagnostics

**Data:** 2026-09-28
**Status:** pesquisa, sem força normativa (a norma é `docs/INTENCAO.md`).
**Pergunta principal:** o que o comportamento indefinido (UB) do C ensina sobre especificar uma
linguagem e garantir determinismo; que classes de vulnerabilidade o Germanio evita por
construção; e o que os diagnostics do Clang (caret, ranges, fix-its) oferecem de referência.
**Complementos:** `DIAGNOSTICS.md` (Elm, rustc) e `COMPILER_ARCHITECTURE.md` já cobrem a
arquitetura geral de diagnostics; aqui entra só o que é específico do C e do Clang.

## Fontes consultadas

- Chris Lattner, "What Every C Programmer Should Know About Undefined Behavior", parte 1 —
  https://blog.llvm.org/2011/05/what-every-c-programmer-should-know.html
- Idem, parte 2 (interação de otimizações, checagens de segurança removidas) —
  https://blog.llvm.org/2011/05/what-every-c-programmer-should-know_14.html
- Clang, "Expressive Diagnostics" — https://clang.llvm.org/diagnostics.html
- ISRG/Prossimo, "What is memory safety" (estatísticas Microsoft, Google, Apple) —
  https://www.memorysafety.org/docs/memory-safety/
- WG14 N2412, "Two's complement sign representation for C2x" —
  https://www.open-std.org/jtc1/sc22/wg14/www/docs/n2412.pdf (só o título e o escopo, pela
  busca; o PDF não foi lido)

Não verificados nesta passagem: a contagem exata de itens do Anexo J (lista de UB) no C23 e
as opções `-fixit` / `-fdiagnostics-parseable-fixits` do Clang (conhecidas, mas o manual do
usuário não foi relido).

---

## Matriz (compacta)

| Item | C | Relevância para o Germanio |
|---|---|---|
| Objetivo original | Linguagem de sistemas eficiente e portável para escrever o Unix; "an extremely efficient low-level programming language" (Lattner, parte 1) | oposto do público do Germanio |
| Filosofia | "Confie no programador"; o padrão deixa lacunas (UB, unspecified, implementation-defined) para que cada compilador e cada CPU gerem o código mais rápido | o Germanio escolhe o inverso: comportamento definido, custo aceito |
| Gramática | Formal no padrão ISO, mas ambígua sem tabela de símbolos (o "typedef problem": `a * b;` é declaração ou expressão conforme `a` seja tipo) (não verificado nesta passagem, fato clássico) | o `.ge` deve continuar analisável sem saber o que um nome é |
| Lexer | Pré-processador textual antes do lexer real; macros operam sobre tokens sem escopo | ver "Complexidade acumulada" |
| Parser / AST | Clang: parser à mão, AST fiel ao código com posições completas e ranges | referência para `ast.Intent` guardar faixa, não só ponto |
| IR | Clang → LLVM IR; é na IR que as otimizações exploram UB | não se aplica |
| Análise semântica | Tipos fracos, conversões implícitas, aliasing baseado em tipo | — |
| Sistema de tipos | Estático, fraco; `void*`, casts livres | — |
| Type inference | Nenhuma (C23 acrescentou `auto` para objetos) (não verificado nesta passagem) | — |
| Compiler/interpreter | Muitos compiladores independentes (GCC, Clang, MSVC); o padrão é o contrato entre eles | o Germanio tem uma implementação; o risco equivalente são os seus vários front-ends |
| Runtime | Mínimo (libc, `crt0`) | — |
| Memory management | Manual (`malloc`/`free`), sem verificação de limites | ver seção "Performance" |
| Standard library | Pequena, com funções historicamente inseguras (`gets`, removida; `strcpy`) | — |
| Package manager | Nenhum oficial; ecossistema fragmentado (make, CMake, pkg-config, vcpkg, Conan) | lição negativa: ferramenta oficial desde o início (o Germanio já tem um binário) |
| Formatter / Linter | Externos (clang-format, clang-tidy), com dezenas de estilos | `ge fmt` único continua certo |
| Language server | clangd, construído sobre o próprio Clang | confirma "LSP sobre o compilador" (já em `nim.md`) |
| Diagnostics | Clang tornou-se referência: coluna, caret, ranges, fix-its, "aka" para typedefs, diff de tipos, notas de expansão de macro (clang.llvm.org/diagnostics.html) | ver seção "Diagnostics do Clang" |
| Testing | Fora da linguagem; sanitizers (ASan, UBSan) como rede de segurança dinâmica | — |
| Evolution process | Comitê ISO WG14, ciclos de ~6–12 anos (C89, C99, C11, C17, C23) | ver `cpp.md` |
| Backward compatibility | Extrema: código de 1989 ainda compila; por isso UB nunca é removido, só documentado | o Germanio ainda pode definir tudo antes de ter legado |

### Principais acertos
Especificação independente da implementação; linguagem pequena o bastante para caber na
cabeça; ABI estável que virou a língua franca entre linguagens; e, no Clang, a ideia de que
diagnostics são uma funcionalidade do produto, não um subproduto do parser.

### Principais problemas
1. **UB como contrato de otimização.** O compilador pode supor que UB nunca acontece. Resultado
   documentado: `if (size > size+1) abort();` é removido, porque overflow de `int` com sinal é
   UB; uma checagem de ponteiro nulo depois de uma desreferência também some. "The compiler
   hasn't malfunctioned; the source code invoked undefined behavior" (Lattner, parte 2). O
   código revisado e o binário dizem coisas diferentes.
2. **Nenhuma ferramenta prova ausência de UB.** "We have lots of tools ... but no good way to
   prove that an application is free of undefined behavior" (parte 2).
3. **UB não se remove, só se estreita.** O C23 passou a exigir complemento de dois para inteiros
   com sinal (N2412), mas o overflow continua indefinido. Definir a representação não definiu a
   operação.
4. **Memória manual.** Leituras e escritas fora dos limites e use-after-free respondem por cerca
   de 70% das vulnerabilidades da Microsoft em uma década e 90% das do Android (memorysafety.org,
   citando MSRC 2019 e Google 2019).

### Complexidade acumulada
Três linguagens em uma: o pré-processador (textual), o C (tipado fracamente) e o dialeto de cada
compilador (atributos, extensões). A falta de um sistema de build oficial criou uma quarta.

---

## Sintaxe (perguntas obrigatórias)

| Pergunta | C | Germanio hoje |
|---|---|---|
| Dados | `struct` com layout de memória explícito | `projetos tem nome obrigatório`: campo, restrição e persistência na mesma frase |
| Funções | `tipo nome(params) { ... }` | lógica em `quando`/`antes de`/funções no interpretador; o domínio quase não usa funções |
| Módulos / imports | `#include` textual; cabeçalho e implementação duplicam declarações | arquivos `.ge` fundidos pelo resolver; sem cabeçalhos |
| Relações | ponteiros | nomes de dados (`issues` dentro de `projetos tem`), resolvidos pelo `resolver.go` |
| Estado | variáveis globais mutáveis | `começa ativo` e estados declarados |
| Fluxo | `if`/`for`/`goto` | regras e eventos; laços só na lógica |
| Erros | códigos de retorno e `errno`, fáceis de ignorar | diagnostics em 4 partes na compilação; `panic` com posição no interpretador |
| Concorrência | threads de biblioteca (C11 `<threads.h>`), data race é UB | goroutines internas; `paralelo()` na lógica |
| Tipos / null | `NULL` em qualquer ponteiro | campos opcionais por padrão, `obrigatório` explícito |
| Boilerplate | cabeçalhos, alocação, liberação, checagens | CRUD, API, auth derivados |
| Projetos grandes | disciplina de cabeçalhos e convenções externas | `ge explain` com origem por fato |

Custo cognitivo do C para o público do Germanio: endereço, tamanho, tempo de vida, dono,
representação numérica e ordem de avaliação são conceitos obrigatórios. Nenhum deles aparece
no `.ge`, e é assim que deve ficar.

## Performance e memória (comparação com `performance/AUDITORIA.md`)

- O C paga zero por segurança e cobra do programador. O Germanio faz o inverso: o usuário nunca
  administra memória; o runtime em Go usa GC. Custo medido: RSS em repouso de 26,9 MB contra
  12,4 MB do Go direto e GC com 2% de CPU sob carga de GET (AUDITORIA 2.2). O C não seria a
  resposta: os maiores custos medidos são algorítmicos (visibilidade O(tabela): 414 ms e 88 MB
  por request, AUDITORIA 2.5), não de gerenciamento de memória.
- O que o C ensina que vale para o Germanio é o **modelo de custo explícito**: em C se sabe o que
  aloca. No Germanio o autor do `.ge` não sabe que uma regra de visibilidade lê a tabela inteira.
  A lição não é expor memória, é expor complexidade quando ela muda de ordem (ver "Para o
  Germanio", item 5).

## Classes de vulnerabilidade que o Germanio evita por construção

| Classe (C) | Por que não existe no `.ge` | Onde ainda pode reaparecer |
|---|---|---|
| Buffer overflow / leitura fora dos limites | não há ponteiros nem índices crus no domínio; Go verifica limites | IMPLEMENTADO: `arr[i]` fora do intervalo vira erro com posição "índice N fora da lista de tamanho M" (`runtime/interpreter/runtime.go:336-349`); índice negativo conta do fim, e índice fracionário é truncado por `int(i)` em silêncio (EVITAR) |
| Use-after-free, double free | GC do Go | — |
| Variável não inicializada | valores zero do Go; campos com padrão | valor ausente num `map[string]any` vira `nil` silencioso (INVESTIGAR) |
| Overflow de inteiro com sinal | números do interpretador são `float64` (`interpreter.go`, `toNumber`) | perda de precisão acima de 2^53 e `0.1 + 0.2`; dinheiro em `float64` é bug de domínio, não de memória |
| Divisão por zero | erro definido "divisão por zero" (`interpreter.go:744`, testado) | — |
| Format string | não há `printf` exposto | — |
| Data race | o domínio não compartilha estado; o banco serializa escritas | `paralelo()` na lógica com escopo compartilhado (INVESTIGAR) |

Implementado: tudo o que está na coluna do meio. Proposto: os itens marcados.

## Diagnostics do Clang como referência

O `compiler/diagnostics/diagnostics.go` já tem código, posição (arquivo, linha, coluna), caret,
motivo, correção e exemplo. Comparação:

| Clang | Germanio hoje | Diferença |
|---|---|---|
| Caret exato, inclusive dentro de string | caret por coluna | equivalente |
| **Range**: sublinha a expressão inteira | só um ponto (`Position`) | falta posição de fim |
| **Fix-it** como transformação de código aplicável por máquina | `Fix` é texto humano | o editor não pode aplicar a correção |
| Preserva o nome do usuário e mostra o subjacente ("aka") | — | útil quando um nome em outro idioma é normalizado para o token canônico: mostrar o que o usuário escreveu e, entre parênteses, o canônico |
| Notas encadeadas (expansão de macro) | — | útil para conflitos entre dois blocos do mesmo dado: a segunda posição como nota |

## Para o Germanio

| # | Classe | Lição | Problema do Germanio | Arquivo | O que remove da cabeça do programador |
|---|---|---|---|---|---|
| 1 | ADOTAR | Nenhum comportamento indefinido: toda operação da linguagem tem resultado ou erro definido e documentado (o C mostra que UB não se remove depois) | a lógica converte tipos em silêncio (`+` concatena se um lado é texto; `toNumber` de texto) sem tabela normativa | `runtime/interpreter/interpreter.go`, `SPEC.md` | "o que acontece se…" — cada caso tem resposta escrita |
| 2 | ADOTAR | A especificação é o contrato e cada frase normativa tem teste (o que o C tem para vários compiladores, o Germanio precisa para vários front-ends) | dois front-ends, formatter e gramática do VS Code | `SPEC.md`, `docs/INTENCAO.md`, testes do parser | não precisa saber qual ferramenta está "certa" |
| 3 | ADAPTAR | Ranges nos diagnostics (Clang) | `Position` só tem início | `compiler/diagnostics/diagnostics.go`, `compiler/ast` | adivinhar onde termina o trecho errado |
| 4 | ADAPTAR | Fix-it estruturado (substituir faixa X por texto Y), aplicável por `ge fmt --fix` ou LSP; texto humano continua | `Fix` é só prosa | `compiler/diagnostics`, futuro LSP | digitar a correção que o compilador já sabe |
| 5 | ADAPTAR | Modelo de custo explícito, sem expor memória: `ge explain` diz quando uma regra implica varredura da tabela | visibilidade por registro O(tabela) invisível ao autor (AUDITORIA 2.5) | `tooling/explicar`, `compiler/parser/resolver.go` | descobrir lentidão em produção |
| 6 | ADAPTAR | "aka" do Clang: mostrar a palavra digitada e o token canônico | normalização multilíngue pode confundir o erro | `compiler/diagnostics`, `compiler/lexer` | traduzir mentalmente a mensagem |
| 7 | EVITAR | Tipos e representação numérica expostos ao leigo; `float64` para dinheiro | cálculos monetários na lógica | `runtime/interpreter/interpreter.go` | (INVESTIGAR um tipo decimal interno para `dinheiro`) |
| 8 | EVITAR | Pré-processador / macros textuais | manteria o `.ge` analisável sem expansão | `compiler/parser` | — |
| 9 | ADAPTAR | Fuzzing e sanitizers como rede dinâmica | já existe `FuzzGermanioParser` (`compiler/parser/germanio_test.go:111`) só para um front-end; faltam alvos para o parser hierárquico, o resolver, o formatter (idempotência) e `-race` no interpretador | `compiler/parser/*_test.go`, `tooling/formatter`, `runtime/interpreter` | — |
