# Estudo dirigido: C++ — complexidade acumulada, zero overhead e evolução por comitê

**Data:** 2026-09-28
**Status:** pesquisa, sem força normativa (a norma é `docs/INTENCAO.md`).
**Pergunta principal:** o que o C++ ensina sobre acumular funcionalidades, sobre "pague só pelo
que usar", sobre um processo de evolução pesado e sobre mensagens de erro de mecanismos
genéricos (templates → concepts)?
**Complementos:** `c.md` (UB, diagnostics do Clang), `LANGUAGE_EVOLUTION.md` (processos de
Rust, Python, Swift) e `docs/gep/` (o processo leve que o Germanio já adotou).

## Fontes consultadas

- Stroustrup, "Remember the Vasa!", WG21 P0977r0, 2018 — https://www.stroustrup.com/P0977-remember-the-vasa.pdf (lido)
- Stroustrup, texto para o ETAPS com a formulação do zero-overhead principle — https://www.stroustrup.com/ETAPS-corrected-draft.pdf (lido)
- cppreference, "Constraints and concepts" — https://en.cppreference.com/w/cpp/language/constraints
- cppreference, `std::pmr::monotonic_buffer_resource` — https://en.cppreference.com/w/cpp/memory/monotonic_buffer_resource
- Chuanqi Xu (desenvolvedor de modules no Clang), "C++20 Modules: Practical Insights, Status and TODOs", 2025 — https://chuanqixu9.github.io/c++/2025/08/14/C++20-Modules.en.html
- Histórico de modules (N1736 de 2004, SG2 em 2012, Modules TS, fusão em 2019) — resumo de busca apontando para https://www.modernescpp.com/index.php/cpp20-a-first-module/ e https://en.wikipedia.org/wiki/Modules_(C++) (secundárias; datas não conferidas nos papers originais)

---

## Matriz (compacta)

| Item | C++ | Relevância para o Germanio |
|---|---|---|
| Objetivo original | "C com classes": abstração sem perder o mapeamento direto para o hardware | — |
| Filosofia | Zero-overhead: "What you don't use, you don't pay for. And further: What you do use, you couldn't hand code any better" (ETAPS) | o Germanio mede o oposto hoje (AUDITORIA §4) |
| Sintaxe | Herdada do C e estendida por 40 anos; várias formas para a mesma coisa (inicialização, declaração de função, `typedef`/`using`) | contraexemplo direto de "uma forma canônica" |
| Gramática | Não LR(k); a análise depende de tipos e de instanciação de templates ("most vexing parse") (não verificado nesta passagem, fato clássico) | — |
| Lexer / Parser | Pré-processador + parser sensível ao contexto semântico | — |
| AST / IR | Clang AST muito rica; templates exigem instanciação antes da checagem completa | — |
| Análise semântica | Resolução de sobrecarga, ADL, SFINAE: regras implícitas que interagem | cada regra implícita do `.ge` precisa aparecer no `ge explain` |
| Sistema de tipos | Estático, nominal, com templates Turing-completos | — |
| Type inference | `auto`, dedução de argumentos de template | — |
| Compiler | GCC, Clang, MSVC, EDG; compilação lenta por reprocessar cabeçalhos | — |
| Runtime | Mínimo; exceções e RTTI opcionais por flag (dialetos) | flags que mudam a linguagem = dialetos |
| Memory management | RAII (destrutor determinístico no fim do escopo), smart pointers, alocadores e arenas (`std::pmr::monotonic_buffer_resource`, C++17: desalocação individual é no-op, libera tudo de uma vez) | ver "Performance" |
| Standard library | Grande, mas sem rede, JSON ou HTTP padrão | — |
| Package manager | Nenhum oficial (vcpkg, Conan, CMake FetchContent) | — |
| Formatter / Linter | clang-format (configurável), clang-tidy | — |
| Language server | clangd | — |
| Diagnostics | Erros de template historicamente em dezenas de linhas; concepts (C++20) trocam isso por "concept X was not satisfied" (cppreference) | ver seção própria |
| Testing / Docs | Externos (GoogleTest, Catch2; Doxygen) | — |
| Evolution process | ISO WG21, ciclo trienal, centenas de papers por reunião | ver seção própria |
| Backward compatibility | Quase absoluta; nada sai, tudo se soma | o Germanio pode remover porque ainda não tem legado |

### Principais acertos
RAII: liberação determinística atada ao escopo, sem GC e sem `free` manual; o princípio de
zero overhead como regra de design verificável; concepts, que transformam um requisito
implícito em interface nomeada com erro legível.

### Principais problemas
1. **Soma sem subtração.** Stroustrup, em 2018, listou 43 propostas em andamento e escreveu:
   "Individually, many (most?) proposals make sense. Together they are insanity to the point of
   endangering the future of C++" e "Hardly any paper contains extensive discussions of the
   proposed feature's effect in combination with other new features ... in 'ordinary code'
   written by 'ordinary programmers'. Few present details of experience of use or teaching"
   (P0977r0). Ele cita de 1992 a advertência do navio Vasa: redesenhado durante a construção
   para caber mais estátuas e canhões, afundou no porto.
2. **Modules levaram cerca de duas décadas.** Primeira proposta em 2004 (N1736), grupo de
   estudo só em 2012, Modules TS em 2018, entrada no C++20 em 2019 (fontes secundárias). Em 2025,
   um mantenedor do Clang ainda descreve: GCC "unproven in production", "CMake seems to be a
   blocking issue", ganho de compilação de 25% a 45% quando funciona, e contágio — "once a
   project uses C++20 Modules ... its downstream dependents must also use C++20 Modules" (Xu,
   2025). A lição: uma decisão estrutural adiada (inclusão textual) fica cara para sempre,
   porque o ecossistema inteiro se organiza em torno dela.
3. **Erros de mecanismos genéricos.** Sem concepts, um `std::sort` numa lista gera um erro
   dentro da biblioteca, longe do código do usuário; com concepts, o erro diz qual requisito o
   argumento não satisfaz (cppreference).

### Complexidade acumulada
É o caso de estudo: cada camada foi razoável quando entrou, e nenhuma pôde sair. O custo
recai sobre quem aprende, que precisa conhecer as formas antigas para ler código existente.

---

## Sintaxe (perguntas obrigatórias)

| Pergunta | C++ | Germanio hoje |
|---|---|---|
| Dados | `class`/`struct` com construtores, regra de 0/3/5 | `tem` com campos e restrições |
| Funções | livres, membros, lambdas, templates, sobrecarga | quase ausentes no domínio |
| Módulos / imports | `#include` e, desde C++20, `import` (coexistem) | arquivos fundidos pelo resolver; uma forma só |
| Relações | ponteiros, referências, smart pointers | nomes resolvidos |
| Estado | membros mutáveis, `const` opcional | estados declarados |
| Erros | exceções, códigos, `std::expected` (C++23), `noexcept`: várias estratégias convivem | diagnostics na compilação; erro com posição no runtime |
| Concorrência | `std::thread`, atomics, coroutines C++20 sem runtime padrão | o runtime administra |
| Null | `nullptr`, `std::optional` | `obrigatório` explícito |
| Boilerplate | construtores, operadores, cabeçalhos | derivado |
| Projetos grandes | guias externos (C++ Core Guidelines) para escolher um subconjunto | a norma já é o subconjunto |

Custo cognitivo: o C++ precisa de guias para dizer qual parte da linguagem **não** usar. Se o
Germanio algum dia precisar de um "Germanio Core Guidelines" dizendo o que evitar, será sinal de
que acumulou formas equivalentes demais.

## Performance e memória (comparação com `performance/AUDITORIA.md`)

- **Zero overhead vs. o que o Germanio mede.** A AUDITORIA §4 é, na prática, uma auditoria de
  zero overhead: `clientes.ge` não usa WhatsApp, git, WebSocket nem fila, e paga +9,1 MB de
  binário, 7,2 ms de init contra 0,8 ms, 26,9 MB de RSS contra 12,4 MB e 4 goroutines de fila.
  O princípio do C++ é o critério certo; o Germanio o viola hoje. Como o Germanio conhece o
  modelo inteiro (`ast.App`), pode decidir estaticamente quais capabilities existem — algo que
  um compilador C++ não sabe sobre um programa aberto.
- **RAII e arenas.** O usuário do Germanio nunca vê memória, mas o runtime pode usar a ideia: o
  request é o escopo natural. A AUDITORIA §3.8 mostra vários `map[string]any` por linha e 88 MB
  por request na visibilidade por registro. Uma arena por request (padrão
  `monotonic_buffer_resource`: aloca em sequência, libera tudo no fim) reduziria pressão de GC.
  Em Go isso não é nativo (o experimento `arena` do Go está suspenso — não verificado nesta
  passagem); o equivalente praticável é reduzir allocs (estruturas tipadas por entidade,
  `sync.Pool` de buffers). O primeiro ganho, porém, é algorítmico (filtro de visibilidade no SQL),
  não de alocador.

## Diagnostics: templates e concepts

O Germanio tem mecanismos genéricos equivalentes a templates: capabilities e seções que
aceitam qualquer entidade. Quando uma capability exige algo da entidade (um campo de data,
um dono, um estado), o erro precisa sair **no `.ge` do usuário e nomear o requisito**, não
dentro do handler genérico em runtime. Concepts são a forma de declarar esse requisito como
interface.

## Processo de evolução: ISO vs. GEP

O WG21 é o exemplo do que o `docs/gep/0001-processo-gep.md` já rejeitou por pesado. O que o
P0977r0 aponta como falha do comitê, porém, vale para qualquer processo: propostas que não
discutem a interação com o resto da linguagem, nem a experiência de ensino, nem as objeções. O
template de GEP pode exigir exatamente isso.

## Para o Germanio

| # | Classe | Lição | Problema do Germanio | Arquivo | O que remove da cabeça do programador |
|---|---|---|---|---|---|
| 1 | ADOTAR | Zero overhead como critério verificável: capability não declarada no `.ge` não entra no binário nem inicializa | WhatsApp, git, WS, fila e hot reload sempre presentes (AUDITORIA §4) | `runtime/engine.go`, `runtime/servidor/servidor.go`, `runtime/capacidade_git.go`, `cli/cli.go` (build) | nada: o programador já não pensa nisso; remove custo que ele não escolheu |
| 2 | ADOTAR | Todo GEP responde: interação com as formas existentes, experiência de ensino, objeções (P0977r0) | o template de GEP pode aceitar propostas isoladas | `docs/gep/0000-template.md` | aprender duas formas para a mesma coisa |
| 3 | ADOTAR | Subtração como regra: cada forma nova substitui uma antiga ou é recusada; sem "guidelines" de subconjunto | dois front-ends (núcleo estrito e dialeto de aplicação) já são duas linguagens | `SPEC.md`, `docs/INTENCAO.md`, `compiler/parser` | saber qual dialeto está lendo |
| 4 | ADAPTAR | Concepts: requisito de capability declarado e checado no resolver, erro no `.ge` nomeando o requisito | capability genérica pode falhar só em runtime | `compiler/parser/resolver.go`, `compiler/diagnostics` | depurar um handler genérico que não escreveu |
| 5 | ADAPTAR | Escopo determinístico (RAII) aplicado ao request no runtime: recursos e buffers presos ao request | allocs por linha e pressão de GC (AUDITORIA §3.8) | `runtime/banco/banco.go`, `runtime/servidor` | — (interno) |
| 6 | EVITAR | Decisão estrutural adiada (inclusão textual → modules em ~20 anos) | o modelo de módulos/pacotes do `.ge` ainda é "fundir arquivos"; decidir cedo como projetos e bibliotecas se compõem | `compiler/parser/resolver.go`, `docs/INTENCAO.md` | migrações futuras de todo o ecossistema |
| 7 | EVITAR | Flags de compilação que mudam a semântica (exceções/RTTI desligados) | opções de runtime que alteram significado do `.ge` | `runtime/engine.go`, `.env` | "este programa roda com qual modo?" |
| 8 | INVESTIGAR | Arena/`sync.Pool` por request no caminho de leitura | medir após o filtro de visibilidade ir para o SQL | `runtime/banco`, `runtime/servidor` | — |
