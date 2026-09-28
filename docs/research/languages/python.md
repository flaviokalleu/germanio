# Estudo dirigido: Python (CPython) — legibilidade com gramática rigorosa

**Data:** 2026-09-28
**Status:** pesquisa, sem força normativa (a norma é `docs/INTENCAO.md`).
**Pergunta principal:** o que Python aprendeu sobre tornar uma linguagem legível sem perder uma
gramática rigorosa?
**Complementos:** a auditoria de problemas de runtime, tipos, concorrência e empacotamento está em
[`python-problemas.md`](python-problemas.md) e não é repetida aqui. A pilha INDENT/DEDENT, o PEP 666
e o Black já aparecem em [`../sintaxe-hierarquica.md`](../sintaxe-hierarquica.md); este arquivo
aprofunda a parte que aquele só cita: a gramática PEG, as regras `invalid_`, a AST em ASDL, a
migração de parser e o processo.

## Fontes consultadas

Repositório (lidos diretamente, ramo `main` em 2026-09-28):
- `Grammar/python.gram` (1625 linhas, 71 regras `invalid_`) — https://github.com/python/cpython/blob/main/Grammar/python.gram
- `Grammar/Tokens` — https://github.com/python/cpython/blob/main/Grammar/Tokens
- `Parser/Python.asdl` (154 linhas) — https://github.com/python/cpython/blob/main/Parser/Python.asdl
- `Lib/test/test_syntax.py` (3860 linhas, doctests de mensagens) — https://github.com/python/cpython/blob/main/Lib/test/test_syntax.py
- `InternalDocs/parser.md` — https://github.com/python/cpython/blob/main/InternalDocs/parser.md
- `InternalDocs/compiler.md` — https://github.com/python/cpython/blob/main/InternalDocs/compiler.md

Documentação e PEPs:
- Language Reference, Lexical analysis — https://docs.python.org/3/reference/lexical_analysis.html
- Full Grammar specification — https://docs.python.org/3/reference/grammar.html
- What's New 3.9 (troca de parser, lib2to3) — https://docs.python.org/3/whatsnew/3.9.html
- What's New 3.10 (Better error messages) — https://docs.python.org/3/whatsnew/3.10.html
- PEP 617 (novo parser PEG) — https://peps.python.org/pep-0617/
- PEP 657 (localização fina em tracebacks) — https://peps.python.org/pep-0657/
- PEP 701 (f-strings na gramática) — https://peps.python.org/pep-0701/
- PEP 1 (processo) — https://peps.python.org/pep-0001/
- PEP 387 (compatibilidade) — https://peps.python.org/pep-0387/
- Citados sem releitura nesta passagem (conteúdo amplamente conhecido; conferir antes de usar
  como prova): PEP 8 https://peps.python.org/pep-0008/, PEP 20 https://peps.python.org/pep-0020/,
  PEP 13 https://peps.python.org/pep-0013/, PEP 634 https://peps.python.org/pep-0634/,
  PEP 3131 https://peps.python.org/pep-3131/.

A série de artigos de Guido van Rossum sobre PEG (Medium) retornou 403; não foi lida.

---

## Matriz

### Objetivo original
Linguagem de script legível para quem não é especialista em sistemas, sucessora do ABC. O ABC
já usava indentação como estrutura; Python herdou isso (não verificado nesta passagem; ver a
FAQ "Why does Python use indentation for grouping of statements?" em docs.python.org/3/faq/design.html).

### Filosofia
PEP 20 ("Readability counts", "There should be one ... obvious way to do it") e PEP 8 ("code is
read much more often than it is written"). O ponto relevante para o Germanio não é o slogan, mas
a consequência operacional: a legibilidade é defendida por *restrições* na gramática (bloco só
por indentação, sem chaves opcionais, sem `;` exigido) e não por convenção.

### Sintaxe
Estrutura por indentação; uma instrução por linha lógica; continuação implícita dentro de
`( [ {` e explícita por `\` (com restrições: nada de comentário depois da `\`)
([Lexical analysis](https://docs.python.org/3/reference/lexical_analysis.html)). Soft keywords
(`match`, `case`, `_`, `type`) são palavras reservadas só no contexto sintático em que aparecem,
o que permitiu adicionar `match` sem quebrar variáveis chamadas `match`
([InternalDocs/parser.md](https://github.com/python/cpython/blob/main/InternalDocs/parser.md)).

### Gramática
Desde 3.9 a gramática é PEG, escrita em `Grammar/python.gram` e compilada por
`Tools/peg_generator` para `Parser/parser.c` ([PEP 617](https://peps.python.org/pep-0617/)).
Pontos de desenho que importam:

- **Escolha ordenada.** "a PEG parser will check if the first alternative succeeds and only if it
  fails, will it continue" — não há ambiguidade, mas a ordem das alternativas vira semântica. O
  documento interno avisa o erro clássico: a alternativa contida antes da maior
  (`'if' ... | 'if' ... 'else' ...`) ([parser.md](https://github.com/python/cpython/blob/main/InternalDocs/parser.md)).
- **Ações constroem a AST diretamente**, sem CST intermediária; o CST do parser LL(1) antigo era
  construído, quase não usado e custava memória ([PEP 617](https://peps.python.org/pep-0617/)).
- **Regra dura sobre ações:** "Actions must **never** be used to accept or reject rules", porque
  isso tornaria a gramática oficial incompleta; e ações não mutam nós (a memoização os compartilha)
  ([parser.md](https://github.com/python/cpython/blob/main/InternalDocs/parser.md)). Ou seja: o
  que o arquivo `.gram` diz é *toda* a linguagem aceita; validação semântica vai para outra fase.
- **Memoização seletiva** (`(memo)` por regra), obrigatória só em regras recursivas à esquerda.
- **Cut `~`** para se comprometer com uma alternativa (usado para desempenho e mensagens).
- **A gramática publicada é derivada da gramática executável.** A página Full Grammar diz: "derived
  directly from the grammar used to generate the CPython parser ... omits details related to code
  generation and error recovery" ([grammar.html](https://docs.python.org/3/reference/grammar.html)).
  Documentação e parser não podem divergir porque vêm do mesmo arquivo.

Motivação da troca: o LL(1) obrigava a escrever regras mais largas do que a linguagem (a
atribuição aceitava qualquer expressão à esquerda e a validação ia para a geração da AST) e
impedia `with (open(a) as x, open(b) as y):` ([PEP 617](https://peps.python.org/pep-0617/)). A
lição: quando o formalismo não expressa a linguagem, a gramática "oficial" passa a mentir e a
verdade se espalha por código ad hoc.

### Lexer/tokenizer
Separado do parser (atípico em PEG): `Parser/lexer` e `Parser/tokenizer` tratam indentação,
encoding, modo interativo ([parser.md](https://github.com/python/cpython/blob/main/InternalDocs/parser.md)).
A indentação vira `INDENT`/`DEDENT` com pilha; linhas em branco e só de comentário não geram
`NEWLINE`; tabs expandem para múltiplos de 8 e mistura dependente da largura dá `TabError`;
identificadores são normalizados em NFKC no léxico
([Lexical analysis](https://docs.python.org/3/reference/lexical_analysis.html)). O PEP 701 levou
as f-strings do parser manual separado para tokens (`FSTRING_START/MIDDLE/END`) e para a gramática,
porque o parser à parte impedia aspas aninhadas, comentários, boas mensagens, e "other Python
implementations have no way to know if they have implemented f-strings correctly"
([PEP 701](https://peps.python.org/pep-0701/)).

### Parser
Gerado (pegen). Duas passadas: a primeira ignora as regras `invalid_`; só se falhar, uma segunda
passada as inclui para produzir a mensagem específica. Se a segunda passada também só chegar a
um erro genérico, usa-se a localização da primeira, "this avoids reporting incorrect locations due
to the invalid rules" (cabeçalho de `python.gram`). O erro genérico é reportado no token mais
distante que se tentou casar ([parser.md](https://github.com/python/cpython/blob/main/InternalDocs/parser.md)).
Custo: zero no caminho feliz, porque "there is typically no need to be fast because execution is
going to stop anyway".

### AST
Definida em ASDL (`Parser/Python.asdl`), linguagem de especificação "independent of its
realization in any particular programming language"; `Parser/asdl_c.py` gera as structs C e o
módulo `ast` do Python ([compiler.md](https://github.com/python/cpython/blob/main/InternalDocs/compiler.md)).
São 154 linhas para toda a AST. A AST é exposta como API estável (`ast.parse`, `ast.unparse`),
e na migração de parser a 3.9 garantiu que "the ast module uses the new parser and produces the
same AST as the old parser" ([What's New 3.9](https://docs.python.org/3/whatsnew/3.9.html)).

### Representações intermediárias
AST → tabela de símbolos (`Python/symtable.c`) → sequência de pseudo-instruções
(`Python/compile.c`) → CFG e otimização (`Python/flowgraph.c`) → montagem do code object
(`Python/assemble.c`); validação em `Python/ast.c`, pré-processamento em `Python/ast_preprocess.c`;
memória do compilador em arena, liberada de uma vez
([compiler.md](https://github.com/python/cpython/blob/main/InternalDocs/compiler.md)).

### Análise semântica
Mínima e posterior ao parse: escopos na symtable, validação da AST, erros como "name 'x' is
parameter and global" levantados em `symtable.c`/`compile.c` e testados separadamente dos erros
de parser em `test_syntax.py` (docstring do arquivo).

### Sistema de tipos
Dinâmico, com anotações opcionais sem efeito em runtime. Ver
[`python-problemas.md`](python-problemas.md), Problema 3.

### Type inference
Não se aplica ao CPython (é trabalho de ferramentas externas: mypy, pyright).

### Compiler/interpreter
Compila para bytecode e interpreta (`Python/ceval.c`). Não é o foco deste estudo.

### Runtime
Ver [`python-problemas.md`](python-problemas.md) (GIL, desempenho).

### Memory management
Contagem de referências + GC de ciclos; ver [`python-problemas.md`](python-problemas.md).

### Standard library
"Batteries included" (`Lib/`). Lição de sintaxe: módulos que embutiam seu próprio parser
(`lib2to3`, `parser`) morreram quando a gramática mudou — ver Backward compatibility.

### Package manager
Externo e fragmentado (pip, venv, poetry, uv); ver [`python-problemas.md`](python-problemas.md), Problema 6.

### Formatter
Não há formatter oficial. PEP 8 é guia; Black e Ruff são externos. O Black confere a equivalência
da AST antes/depois (já citado em `sintaxe-hierarquica.md`). Consequência: por anos a comunidade
discutiu estilo que a linguagem poderia ter fixado; Go e Gleam decidiram diferente.

### Linter
Externo (pyflakes, pylint, Ruff). O compilador emite `SyntaxWarning` para casos suspeitos.

### Language server
Nenhum oficial (pyright, pylsp, Jedi são de terceiros). Cada um reimplementa ou reusa parser
próprio, o que é exatamente o risco "vários parsers" do Germanio.

### IDE tooling
IDLE na stdlib; o resto é externo. O módulo `tokenize` passou a usar o tokenizer C a partir do
PEP 701 para não divergir (não verificado em detalhe além do PEP).

### Diagnostics
O avanço grande do 3.10 veio da gramática, não de pós-processamento
([What's New 3.10](https://docs.python.org/3/whatsnew/3.10.html)):
- bracket não fechado aponta para **onde abriu** (`'{' was never closed`), não para o EOF;
- `expected ':'`; `Perhaps you forgot a comma?`; `Maybe you meant '==' instead of '='?`;
- `IndentationError: expected an indented block after 'if' statement on line 2` — o erro de layout
  nomeia a construção dona do bloco e a linha dela. A regra é literalmente
  `a='if' named_expression ':' NEWLINE !INDENT` → mensagem (`invalid_if_stmt` em `python.gram`);
- `NameError`/`AttributeError` com `Did you mean: ...?`.
PEP 626 garantiu números de linha precisos; o PEP 657 acrescentou colunas início/fim por
instrução de bytecode e sublinha a subexpressão que falhou, a um custo de ~22% nos `.pyc` da
stdlib, desligável por `-X no_debug_ranges` ([PEP 657](https://peps.python.org/pep-0657/)).
Nenhuma mensagem usa nome de token interno ("unexpected DEDENT" sumiu das mensagens comuns).

### Testing
`Lib/test/test_syntax.py` é uma coleção de doctests: o programa inválido e a mensagem exata
esperada lado a lado (563 ocorrências de `>>>`/doctest). A mensagem é contrato testado, não
detalhe. Há ainda testes do gerador PEG (`Lib/test/test_peg_generator`, não aberto nesta
passagem).

### Documentation
Language Reference (normativa de fato), Full Grammar gerada do `.gram`, `InternalDocs/` para
mantenedores (parser, compiler), PEPs como registro de decisões com "Rejected Ideas".

### Evolution process
PEP 1: ideia no Discourse → patrocinador core → rascunho revisado por editores → discussão →
decisão do Steering Council (ou PEP-Delegate). Seções exigidas incluem Motivation, Rationale,
Backwards Compatibility, **How to Teach This**, Reference Implementation e **Rejected Ideas**
([PEP 1](https://peps.python.org/pep-0001/)). O Steering Council (PEP 13) nasceu depois da saída
de Guido como BDFL em 2018, após a disputa do PEP 572 (fato histórico; não relido aqui).

### Backward compatibility
PEP 387: deprecação por no mínimo dois anos/duas versões, preferencialmente cinco; mudança
incompatível precisa de "a large benefit to breakage ratio" e ser fácil de corrigir
([PEP 387](https://peps.python.org/pep-0387/)). A troca de parser foi feita sem mudar a linguagem:
3.9 trouxe o PEG como padrão com `-X oldparser` de escape, e nenhuma sintaxe nova dependente de
PEG entrou antes de remover o antigo em 3.10 ([PEP 617](https://peps.python.org/pep-0617/),
[What's New 3.9](https://docs.python.org/3/whatsnew/3.9.html)). Efeito colateral documentado:
`lib2to3`, com parser LL(1) próprio, foi deprecado porque "Python 3.10 may include new language
syntax that is not parsable by lib2to3's LL(1) parser" ([What's New 3.9](https://docs.python.org/3/whatsnew/3.9.html)).
O mesmo destino atinge qualquer ferramenta que mantenha uma cópia da gramática.

### Principais acertos
1. Indentação como única forma de bloco, com algoritmo de pilha no lexer e gramática sobre tokens.
2. Gramática executável única, da qual a documentação é derivada.
3. Separação estrita: gramática decide o que é aceito; ações só constroem; semântica depois.
4. Diagnósticos especializados como regras da própria gramática, em segunda passada.
5. AST especificada numa linguagem neutra (ASDL) e exposta como API estável.
6. Mensagens de erro como contrato testado (doctests).
7. Migração de parser com equivalência de AST e sem mudança de linguagem.
8. Soft keywords para crescer sem quebrar nomes.

### Principais problemas
Sintáticos: a continuação por `\` é frágil (espaço invisível depois dela quebra); tabs ainda são
aceitos quando consistentes; ausência de formatter oficial; ordem de alternativas PEG é semântica
escondida. Os problemas de runtime estão em [`python-problemas.md`](python-problemas.md).

### Complexidade acumulada
71 regras `invalid_` num arquivo de 1625 linhas: parte substancial da gramática existe só para
ensinar. É complexidade útil, mas mostra o preço de uma sintaxe rica em expressões; cada
construção nova (walrus, match, f-strings, t-strings) exige novas regras `invalid_`.

### O que Germanio pode aprender
Ver a seção final.

### O que Germanio NÃO deve copiar
- **PEG como formalismo.** O Germanio tem seções fechadas por contexto e um item por linha; a
  escolha ordenada esconderia prioridade entre formas de frase e é desnecessária. Uma gramática
  LL simples sobre ABRE/FECHA basta e é mais fácil de explicar a leigos.
- **Continuação explícita por `\`.** Já rejeitada pelo `INTENCAO.md` (uma linha lógica não continua).
- **Tabs "consistentes" aceitos.** O Germanio já os proíbe; manter.
- **Expressões ricas na mesma linha** (compreensões, walrus, lambdas): geram exatamente as regras
  `invalid_` que o Germanio não quer precisar.
- **Tooling fora da linguagem** (formatter, LSP de terceiros): o Germanio já decidiu `ge fmt` no
  binário; não recuar.
- **NFKC como normalização de identificadores.** NFKC não remove acentos; o Germanio já faz uma
  dobra própria (acento), que é mais forte — ver o arquivo do Nim sobre os riscos disso.

---

## Para o Germanio

| # | Classe | Lição | Problema concreto do Germanio | Arquivo afetado |
|---|---|---|---|---|
| 1 | ADOTAR | O arquivo da gramática aceita **toda** a linguagem e nada mais; ações não aceitam nem rejeitam; validação vai para o resolver ([parser.md](https://github.com/python/cpython/blob/main/InternalDocs/parser.md)) | `isIntentLine`/`sectionOf` decidem aceitação por palavra solta; parte da aceitação está espalhada em `if`s de `declarativo.go` | `compiler/parser/hierarquia.go`, `compiler/parser/declarativo.go`, `docs/INTENCAO.md` (EBNF) |
| 2 | ADAPTAR | Diagnósticos especializados como alternativas de erro **da própria gramática**, tentadas só depois da falha, sem mudar a localização do erro genérico (`invalid_*`, duas passadas) | mensagens de `teach` são escritas no ponto de falha, caso a caso; não há lugar único que liste os erros conhecidos de cada seção | `compiler/parser/hierarquia.go` (`teach`), `compiler/diagnostics` |
| 3 | ADOTAR | O erro de layout nomeia **a linha dona** do bloco: "expected an indented block after 'if' statement on line 2" | `layoutTree` lista os níveis abertos, mas uma seção vazia (`tem` sem filhos) deve dizer "esperava itens abaixo de `tem` (linha N)" | `compiler/parser/hierarquia.go` |
| 4 | ADOTAR | Bracket/aspas não fechados apontam para **onde abriram** | no lexer, texto não fechado gera `unterminated string at line %d, column %d` com a posição de abertura no texto, mas o `Diagnostic` (GE1001) recebe `l.line/l.col`, a posição onde a leitura parou, e a mensagem está em inglês ("Código não reconhecido" + motivo) | `compiler/lexer/lexer.go` |
| 5 | ADOTAR | Mensagens como contrato testado: programa inválido + mensagem exata lado a lado (`test_syntax.py`) | `TestHierarquiaErros` existe; ampliar para um arquivo de casos (golden) por código `GEnnnn` | `compiler/parser/hierarquia_test.go`, `compiler/diagnostics` |
| 6 | ADOTAR | Documentação da gramática **derivada** da gramática executável ([grammar.html](https://docs.python.org/3/reference/grammar.html)); cópias independentes morrem quando a gramática muda (lib2to3, [What's New 3.9](https://docs.python.org/3/whatsnew/3.9.html)) | `gerar_gramatica.py` copia à mão a lista de seções e palavras (`tem`, `pode`, `pertence`…) que já existe em `sections` e no mapa do lexer; o EBNF do `INTENCAO.md` também é cópia | `vscode-germanio/tools/gerar_gramatica.py`, `compiler/parser/hierarquia.go` (exportar a tabela), `tooling/formatter` |
| 7 | ADAPTAR | AST especificada num formato neutro (ASDL) e exposta como API estável | `ast.Intent`/`ast.App` só existem como structs Go; `ge explain` e o futuro LSP dependem deles; uma descrição declarativa dos nós (ou `ge explain --json` estável) evitaria o acoplamento | `compiler/ast`, `tooling/explicar` |
| 8 | ADOTAR | Trocar o front-end sem mudar a linguagem, provando AST igual e com chave de escape por uma versão | há dois front-ends (núcleo estrito e dialeto de aplicação); a unificação deve seguir o roteiro do PEP 617: mesma AST em todo o corpus, depois remoção | `compiler/parser/germanio.go`, `compiler/parser/parser.go`, `examples/` |
| 9 | ADAPTAR | Soft keywords: palavra reservada só na posição em que é seção | já é o modelo de `sectionOf` (só a primeira palavra de uma linha filha); documentar como regra e testar que `tem` pode ser nome de campo onde não é seção | `compiler/parser/hierarquia.go`, `docs/INTENCAO.md` |
| 10 | ADOTAR | Sub-linguagens dentro de texto (f-strings) pertencem à gramática, não a um parser à parte ([PEP 701](https://peps.python.org/pep-0701/)) | interpolação e expressões em textos `.ge`, se existirem, não devem ter um mini-parser separado | `compiler/lexer/lexer.go` |
| 11 | ADAPTAR | Seções "How to Teach This" e "Rejected Ideas" obrigatórias em cada proposta ([PEP 1](https://peps.python.org/pep-0001/)) | decisões de sintaxe hoje ficam em pesquisas e commits; a tabela "Decisões" de `sintaxe-hierarquica.md` é o embrião | `docs/` (modelo de proposta) |
| 12 | ADAPTAR | Política de compatibilidade com prazo e razão benefício/quebra ([PEP 387](https://peps.python.org/pep-0387/)) | a forma plana não é depreciada; quando algo for, precisa de aviso por versões e correção automática pelo `ge fmt` | `docs/INTENCAO.md`, `tooling/formatter` |
| 13 | EVITAR | PEG e escolha ordenada como formalismo | ordem de alternativas viraria semântica escondida numa linguagem para leigos | — |
| 14 | EVITAR | Formatter e LSP fora da distribuição | cada ferramenta externa reimplementaria o parser | `tooling/` |
| 15 | INVESTIGAR | Colunas início/fim por fato (PEP 657) para sublinhar a palavra exata | `ge explain` mostra arquivo:linha; coluna final permitiria sublinhar a seção inteira na mensagem | `compiler/diagnostics`, `tooling/explicar` |

Resposta curta à pergunta principal: Python manteve a legibilidade por **restrição** (um único
jeito de fazer bloco) e manteve o rigor por **fonte única** (uma gramática executável da qual saem
parser, AST e documentação). O que tornou os erros humanos não foi relaxar a gramática, e sim
acrescentar a ela, numa segunda passada, as formas erradas conhecidas com o nome da construção
dona. As ferramentas que mantiveram cópias próprias da gramática (lib2to3) ficaram para trás.
