# Go: linguagem pequena, tooling e runtime grandes

- **Data:** 2026-09-28
- **Status:** estudo dirigido concluído (fontes oficiais e código-fonte do Go 1.27.1 instalado
  localmente em `$(go env GOROOT)/src`; os caminhos citados abaixo foram vistos nessa árvore).
- **Pergunta principal:** como Go mantém uma linguagem pequena enquanto o tooling e o runtime
  carregam enorme complexidade? E, como o Germanio é escrito em Go, o que dos pacotes `go/*`
  vale reproduzir como padrão?

## Fontes consultadas

- Repositório: https://github.com/golang/go (árvore local `src/cmd/compile`, `src/go/*`,
  `src/cmd/gofmt`, `src/cmd/vet`, `src/internal/godebugs`)
- README do compilador: https://go.dev/src/cmd/compile/README
- Especificação: https://go.dev/ref/spec
- Promessa Go 1: https://go.dev/doc/go1compat
- GODEBUG e compatibilidade (Russ Cox, 2023): https://go.dev/blog/compat
- Semântica de loop por versão de módulo: https://go.dev/blog/loopvar-preview
- Módulos (MVS, go.sum, diretivas `go`/`toolchain`): https://go.dev/ref/mod
- MVS, justificativa: https://research.swtch.com/vgo-mvs
- Processo de proposals: https://github.com/golang/proposal/blob/master/README.md
- `go/token`: https://pkg.go.dev/go/token
- `go/format`: https://pkg.go.dev/go/format
- Framework de análise: https://pkg.go.dev/golang.org/x/tools/go/analysis
- Design do gopls: https://github.com/golang/tools/blob/master/gopls/doc/design/design.md
- gofmt (blog): https://go.dev/blog/gofmt
- Go Proverbs (Rob Pike, Gopherfest 2015): https://go-proverbs.github.io/
- "Go at Google" (Rob Pike, 2012): https://go.dev/talks/2012/splash.article
- Guia do GC: https://go.dev/doc/gc-guide
- Exemplos testáveis: https://go.dev/blog/examples

Já coberto em `docs/research/sintaxe-hierarquica.md` e não repetido aqui: o gofmt como
formatador canônico e o CommentMap como técnica geral. Este arquivo aprofunda a arquitetura.

---

## Matriz

### Objetivo original
Resolver problemas de engenharia de software em escala do Google: builds lentos, dependências
descontroladas, bases de milhões de linhas mantidas por milhares de pessoas
([splash](https://go.dev/talks/2012/splash.article)). O objetivo nunca foi expressividade
máxima; foi custo de manutenção baixo.

### Filosofia
"Clear is better than clever"; "A little copying is better than a little dependency"
([proverbs](https://go-proverbs.github.io/)). A linguagem é pequena de propósito para que as
ferramentas possam ser grandes: "The grammar is easy to reason about and therefore tools are
easy to write" ([splash](https://go.dev/talks/2012/splash.article)). A complexidade não some,
é deslocada para lugares com um só dono (compilador, runtime, `go` command) em vez de ficar
espalhada em cada programa.

### Sintaxe
Chaves e ponto e vírgula inseridos automaticamente pelo lexer; 25 palavras reservadas
([splash](https://go.dev/talks/2012/splash.article)). Uma forma idiomática para quase tudo
(um só laço `for`).

### Gramática
EBNF na própria especificação ([spec](https://go.dev/ref/spec)). Analisável sem tabela de
símbolos nem informação de tipos ([splash](https://go.dev/talks/2012/splash.article)): esse é o
requisito que torna `gofmt`, `go vet` e `gopls` baratos de escrever. A especificação é um
único documento e é normativa; o compilador é que tem de concordar com ela.

### Lexer/tokenizer
`go/scanner` para as ferramentas; `cmd/compile/internal/syntax` tem o seu próprio. A inserção
automática de `;` é decisão de lexer, não de parser, o que mantém a gramática livre de
contexto.

### Parser
Descida recursiva à mão em dois lugares: `go/parser` (ferramentas) e
`cmd/compile/internal/syntax` (compilador) ([README](https://go.dev/src/cmd/compile/README)).
Ambos recuperam de erros e continuam, acumulando vários diagnósticos por arquivo.

### AST
Duas ASTs: `go/ast` (pública, estável pela promessa Go 1, com `CommentGroup` e
`ast.CommentMap`, visto em `src/go/ast/commentmap.go`) e a árvore de `syntax` (interna,
livre para mudar). Posições não são structs: cada nó guarda um `token.Pos`, um inteiro.

**Posições (`go/token`).** Um `FileSet` atribui a cada arquivo um intervalo disjunto
`[base, base+size]`; um `Pos` é um inteiro dentro desse intervalo e `offset = Pos - base`. Cada
`token.File` guarda apenas a tabela de inícios de linha; `FileSet.Position(p)` reconstrói
arquivo, linha, coluna e offset quando alguém precisa exibir
([go/token](https://pkg.go.dev/go/token)). `NoPos` (zero) é "sem posição". Consequências:
posições são comparáveis e ordenáveis; um intervalo (`Pos`, `End()`) identifica exatamente um
trecho de fonte, o que é o que o formatter, o `SuggestedFix` e o LSP precisam.

### Representações intermediárias
`syntax` → `types2` → IR unificada (`noder`, `ir`) → inlining, devirtualização, escape analysis
→ `walk` (desaçucaramento) → SSA genérico (`ssa`, `ssagen`) → passes por arquitetura →
`cmd/internal/obj` → export data em formato "unified" com decodificação preguiçosa
([README](https://go.dev/src/cmd/compile/README); diretórios vistos em
`src/cmd/compile/internal/`). Cada fase tem um pacote com um nome; o README é o mapa.

### Análise semântica
`go/types` (ferramentas) e `types2` (compilador). `types2` "is a port of `go/types` to use the
syntax package's AST instead of `go/ast`" ([README](https://go.dev/src/cmd/compile/README)).
Para não manter dois type checkers à mão, 61 dos 76 arquivos não-teste de `src/go/types` são
**gerados** a partir de `types2` (`// Code generated by "go test -run=Generate -write=all"`), e
`generate_test.go` falha se as cópias divergirem (visto em `src/go/types/generate_test.go`). O
`types.Config` aceita um `Error func(error)` para continuar após o primeiro erro.

### Sistema de tipos
Estrutural para interfaces, nominal para o resto; genéricos por type parameters desde 1.18.
Pequeno por decisão: sem herança, sem sobrecarga, sem exceções.

### Type inference
Local: `:=` e inferência de argumentos de tipo. Nunca global; o leitor sempre acha o tipo
perto do uso.

### Compiler/interpreter
Compilador AOT próprio (`cmd/compile`), rápido por causa do export data: cada import lê um
único arquivo de objeto, sem includes transitivos
([splash](https://go.dev/talks/2012/splash.article)).

### Runtime
Grande e escondido: escalonador de goroutines, GC, mapas, canais, pilhas crescentes, tudo em
`src/runtime`. O programador não configura quase nada.

### Memory management
GC concorrente mark-sweep, não-móvel, não-geracional; exatamente dois botões, `GOGC` e
`GOMEMLIMIT`; a escolha pilha/heap é da escape analysis do compilador, não do programador
([gc-guide](https://go.dev/doc/gc-guide)). É o exemplo mais puro do padrão "o mecanismo
decide, a linguagem não expõe".

### Standard library
Ampla (HTTP, crypto, JSON, templates) e sob a promessa de compatibilidade. Reduz dependências
externas, coerente com "a little copying".

### Package manager
Módulos no próprio `go` command. **MVS**: escolhe a *menor* versão que satisfaz todos os
requisitos; resultado determinístico, não muda quando sai versão nova, sem lock file separado,
implementação de "a few hundred lines" em tempo polinomial em vez de SAT
([vgo-mvs](https://research.swtch.com/vgo-mvs), [ref/mod](https://go.dev/ref/mod)). `go.sum`
guarda hashes conferidos contra `sum.golang.org`. Regra de compatibilidade de import: mesmo
caminho implica compatível; quebra exige `/v2` no caminho ([ref/mod](https://go.dev/ref/mod)).

### Formatter
`gofmt` = `go/parser` + `go/printer` sobre a AST com comentários; sem configuração. "Gofmt's
style is no one's favorite, yet gofmt is everyone's favorite" (Rob Pike,
[proverbs](https://go-proverbs.github.io/)). `go/format` avisa que o estilo muda com o tempo e
que quem precisa de estabilidade deve fixar a versão do binário
([go/format](https://pkg.go.dev/go/format)). O printer é testado por pares `.input`/`.golden`
(`src/go/printer/testdata`). `gofmt -r 'padrão -> substituição'` e o antigo `gofix` só são
possíveis porque entrada e saída estão no formato canônico: "the only changes made to the
source code are semantic ones" ([splash](https://go.dev/talks/2012/splash.article),
[blog gofmt](https://go.dev/blog/gofmt)).

### Linter
`go vet` roda analisadores do framework `golang.org/x/tools/go/analysis` (40 passes vendorados
em `src/cmd/vendor/golang.org/x/tools/go/analysis/passes`). Um `Analyzer` declara `Name`, `Doc`,
`Run`, `Requires` (dependências entre analisadores), `ResultType` e `FactTypes` (fatos
serializáveis entre pacotes); `Run` recebe um `Pass` com AST e tipos e reporta `Diagnostic`s
com `SuggestedFix` feito de `TextEdit`s. O mesmo analisador roda em três drivers:
`singlechecker`, `multichecker` e `unitchecker` (usado pelo `go vet`), e também no gopls
([analysis](https://pkg.go.dev/golang.org/x/tools/go/analysis)).

### Language server
`gopls`: um processo longo, uma instância para todos os editores, reusa `go/packages` e
`go/types` em vez de infraestrutura própria, trabalha com snapshots imutáveis e recheca só os
pacotes afetados; diagnósticos do `go/analysis` viram diagnósticos do LSP
([design](https://github.com/golang/tools/blob/master/gopls/doc/design/design.md)).

### IDE tooling
Tudo via gopls. Não há gramática "de editor" independente como fonte de verdade; o realce
semântico vem do mesmo type checker.

### Diagnostics
`arquivo:linha:coluna: mensagem`, curtos. Erros por arquivo acumulados; imports e variáveis
não usados são erro, não aviso ([splash](https://go.dev/talks/2012/splash.article)). O ponto
fraco é a pedagogia: mensagens não explicam o porquê nem como corrigir.

### Testing
`go test` embutido no `go` command, convenção `_test.go`, `testing.T`, benchmarks e fuzzing no
mesmo pacote. **Exemplos testáveis**: `ExampleFoo()` com comentário `// Output:` é executado
pelo `go test`, comparado com a saída e exibido na documentação
([examples](https://go.dev/blog/examples)): documentação que não pode ficar mentirosa.

### Documentation
Comentários de doc ligados às declarações; `go doc` e pkg.go.dev. "Documentation is for users"
([proverbs](https://go-proverbs.github.io/)).

### Evolution process
Issue curta → discussão → aceitar, recusar ou pedir design doc (`design/NNNN-nome.md`) →
decisão; um grupo de revisão semanal publica atas; estados Incoming/Active/Likely
Accept/Likely Decline (uma semana de espera)/Accepted/Declined/Hold; sem consenso, decidem os
arquitetos e depois um árbitro. Mudanças de linguagem têm formulário próprio e precisam
resolver problema importante para muitos com impacto mínimo nos demais
([proposal README](https://github.com/golang/proposal/blob/master/README.md)).

### Backward compatibility
Três camadas:
1. **Promessa Go 1**: compatibilidade de fonte, com exceções explícitas e listadas (segurança,
   comportamento não especificado, erro da spec, bugs, literais de struct sem chave, dot
   imports, `unsafe`); ferramentas e desempenho fora da promessa
   ([go1compat](https://go.dev/doc/go1compat)).
2. **GODEBUG**: mudança compatível-mas-quebradora ganha um nome (`panicnil`, `http2client`,
   `x509sha1`); o padrão de cada nome é derivado da linha `go` do `go.mod`; `//go:debug` no
   pacote main sobrescreve; mínimo de dois anos (quatro releases) de suporte
   ([compat](https://go.dev/blog/compat)). A tabela é dado, não código espalhado:
   `src/internal/godebugs/table.go` lista nome, pacote, versão em que mudou e valor antigo, e
   uma lista `Removed`; um teste exige que `doc/godebug.md` esteja em dia.
3. **Versão da linguagem por módulo**: a mudança da variável de laço em 1.22 só vale em
   módulos que declaram `go 1.22`; "Old code will continue to mean exactly what it means
   today" ([loopvar](https://go.dev/blog/loopvar-preview)). Desde 1.21 a linha `go` é
   obrigatória e o compilador recusa recursos posteriores a ela ([ref/mod](https://go.dev/ref/mod)).
   "Go 2, in the sense of breaking with the past... is never going to happen"
   ([compat](https://go.dev/blog/compat)).

### Principais acertos
- Gramática sem contexto semântico, o que barateia todas as ferramentas.
- Um único formato; refatoração mecânica (`gofmt -r`, `gofix`) só gera diffs semânticos.
- Complexidade de runtime atrás de dois botões.
- Compatibilidade como *dado versionado* (tabela GODEBUG, linha `go`), não como promessa vaga.
- Um framework de análise com três drivers; uma regra escrita uma vez aparece no CLI e no
  editor.

### Principais problemas
- Duas ASTs, dois parsers e dois type checkers (`go/*` vs `syntax`/`types2`); a duplicação só
  é suportável porque um é *gerado* do outro e um teste acusa divergência.
- `go/ast` congelado pela promessa Go 1 limita a evolução (por isso o compilador tem a sua
  própria árvore).
- Diagnósticos secos, sem explicação.
- Erro explícito verboso (`if err != nil`) empurrado ao programador.

### Complexidade acumulada
Concentrada em quatro donos: `cmd/compile` (SSA, escape analysis, inlining, PGO), `src/runtime`,
`cmd/go` (módulos, proxy, checksum db, toolchain switching) e gopls. A linguagem cresceu pouco
(genéricos, range-over-func); o `go` command e o gopls cresceram muito.

### O que Germanio pode aprender
Ver "Para o Germanio". Resumo: posições compactas num `FileSet`; `ge check` como driver de
analisadores declarados; printer sobre a árvore com comentários; testes golden; compatibilidade
como tabela; versão da linguagem declarada no projeto; um só dono para cada complexidade.

### O que Germanio NÃO deve copiar
- A dupla AST/type checker sem geração automática: o Germanio já tem dois front-ends (núcleo
  estrito e dialeto de aplicação) e um terceiro "parser" em regex no VS Code; Go mostra o custo
  e exige máquina (gerador + teste) para contê-lo.
- Diagnósticos no estilo Go: o formato de quatro partes do Germanio é superior para o público.
- Tratamento explícito de erros pelo programador: o público do Germanio não pode carregar isso.
- Semantic import versioning (`/v2` no caminho): resolve um problema de ecossistema de
  bibliotecas que o Germanio não tem.
- Congelar a API da AST como contrato público cedo demais.

---

## Para o Germanio

**ADOTAR: posições compactas num FileSet.** Hoje cada nó de `compiler/ast/ast.go` carrega um
`diagnostics.Position` com `File string, Line, Column int, Context string`
(`compiler/diagnostics/diagnostics.go`). Não há intervalo (fim), então o formatter, um
`SuggestedFix` ou um LSP não conseguem saber que trecho substituir. Proposta: um tipo `Pos`
inteiro e um `FileSet` com tabela de linhas por arquivo, no padrão de
[go/token](https://pkg.go.dev/go/token); cada nó ganha `Pos` e `End`; `Position` (com
`Context`) é calculado sob demanda para `ge explain` e para as mensagens. Afeta
`compiler/diagnostics`, `compiler/ast`, `compiler/lexer`, `compiler/parser/hierarquia.go`,
`tooling/explicar`. Pode até reutilizar `go/token` diretamente, pois é biblioteca padrão
estável.

**ADOTAR: `ge check` como driver de analisadores.** `semantic.Check` retorna o primeiro `error`
(`compiler/semantic/check.go`, várias saídas `return err`) e as regras de contradição,
ambiguidade e segurança estão espalhadas entre `semantic`, `tooling/intelligence/validator.go`
e o resolver. O modelo `go/analysis` resolve isso: cada regra é um `Analisador{Nome, Doc,
Requer, Executar(passo)}` que lê o `ast.App` resolvido e reporta `Diagnostic`s (já no formato
de quatro partes) com correção sugerida opcional; um driver os ordena por dependência, coleta
todos os diagnósticos e serve tanto o `ge check` quanto o futuro LSP
([analysis](https://pkg.go.dev/golang.org/x/tools/go/analysis)). Não precisa de `FactTypes`
entre pacotes no início. Afeta `compiler/semantic`, `tooling/intelligence/validator.go`,
`tooling/gecli/cli.go` (caso `check`). A pendência de `docs/INTENCAO.md` "cobertura dos
diagnósticos de contradição e ambiguidade em `ge check`" vira "um analisador por regra, com
teste".

**ADAPTAR: printer sobre a árvore, não sobre o texto.** O `ge fmt` do dialeto de aplicação
(`tooling/formatter/intencao.go`) reindenta linhas de texto com a própria pilha off-side e
depois reparseia e compara o significado. A verificação por significado é boa (Go não faz;
Black faz). O defeito é a arquitetura: o formatter reimplementa a pilha de níveis que já existe
em `layoutTree` (`compiler/parser/hierarquia.go`), o que é exatamente o risco "vários parsers"
já registrado. Adaptação: o parser produz uma árvore concreta de linhas com comentários presos
(`CommentMap`-like) e posições; o formatter imprime essa árvore. Manter a checagem de
significado como rede de segurança. Casos `.input`/`.golden` no estilo de
`src/go/printer/testdata`. Afeta `tooling/formatter`, `compiler/parser/hierarquia.go`.

**ADOTAR: gramática que não precisa de semântica para ser analisada.** O princípio de Pike
(parse sem tabela de símbolos) é o que deixa formatter, realce e LSP triviais. Toda palavra
nova de seção (`tem`, `pode`, `acesso`, `regras`) deve ser reconhecível pelo layout e por uma
tabela fechada, nunca por consulta às entidades declaradas. Afeta `compiler/parser/hierarquia.go`,
`vscode-germanio/tools/gerar_gramatica.py` (que deve ser gerado da mesma tabela).

**ADAPTAR: quando houver duas implementações, gere uma da outra e teste a igualdade.** Go
contém a duplicação `types2`/`go/types` com gerador + `generate_test.go`. O Germanio já faz isso
para a gramática TextMate (`gerar_gramatica.py`); falta um teste que falhe se a tabela de
palavras do parser e a gramática gerada divergirem. Afeta `vscode-germanio/tools/`,
`compiler/lexer`.

**ADOTAR: compatibilidade como tabela versionada.** Quando uma frase do `.ge` mudar de
significado, registre-a numa tabela (`nome`, versão em que mudou, comportamento antigo) e
derive o padrão da versão declarada no projeto, como GODEBUG + linha `go`
([compat](https://go.dev/blog/compat), [loopvar](https://go.dev/blog/loopvar-preview)).
Subtração: o leigo não escolhe flag nenhuma; o projeto declara `sistema ... versão germanio 1`
uma vez (ou `ge init` escreve) e `ge explain` mostra "esta frase segue o significado da versão
1". Afeta `compiler/parser/resolver.go`, `docs/INTENCAO.md` (seção a criar), `cli/modelos`.

**ADOTAR: exemplos executáveis como documentação.** Os exemplos normativos de
`docs/INTENCAO.md` já "devem virar testes"; o padrão `Example` + `// Output:`
([examples](https://go.dev/blog/examples)) dá a forma: um bloco de exemplo com o resultado
esperado, extraído e executado por `ge testar`. Afeta `tooling/gecli/cli.go` (`testar`),
`docs/INTENCAO.md`.

**ADOTAR: complexidade com um dono e poucos botões.** GC com dois botões e escape analysis
decidindo pilha/heap é o modelo para banco, cache, pool e transações do runtime do Germanio:
o `.ge` não configura; o runtime decide, e `ge explain` diz o que foi decidido. Afeta
`runtime/banco`, `runtime/servidor`, `docs/INTENCAO.md`.

**INVESTIGAR: MVS para `integracoes/`.** Se adaptadores virarem pacotes versionados, MVS dá
builds reprodutíveis sem lock file e sem solver
([vgo-mvs](https://research.swtch.com/vgo-mvs)). Só vale quando existir ecossistema de
adaptadores de terceiros; hoje não resolve problema concreto.

**INVESTIGAR: processo de proposals com estados publicados.** Uma pasta
`docs/propostas/NNNN-nome.md` com estados (proposta, provável aceite, aceita, recusada) e uma
semana de "provável" antes da decisão ([proposal
README](https://github.com/golang/proposal/blob/master/README.md)) daria rastreabilidade às
mudanças da camada de intenção. Depende do tamanho da equipe.

**EVITAR:** dois front-ends mantidos à mão (núcleo `SPEC.md` e dialeto de aplicação) sem
gerador nem teste de equivalência; diagnósticos sem "por quê"; expor ao `.ge` qualquer decisão
que o runtime pode tomar sozinho.
