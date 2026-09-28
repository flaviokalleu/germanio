# Lições para o Germanio

**Data:** 2026-09-28. **Status:** consolidação da pesquisa, **sem força normativa**
([docs/README.md](../../README.md): `research/` não é norma). Nenhuma lição aqui altera
`docs/INTENCAO.md`, `SPEC.md`, `GERMANIO_GAPS.md` ou a skill; as que contradizem a norma
estão marcadas **exige decisão deliberada** e ficam para o responsável pelo design.
Revisão do código conferida: `fc31daf` (master). A seção
[Segunda rodada (ecossistema)](#segunda-rodada-ecossistema), no fim, e as anotações
"conferido em `0c12051`" foram acrescentadas depois, sobre a revisão `0c12051`; o texto da
primeira rodada não foi reescrito.

**Não é um ranking de linguagens.** As linguagens aparecem só como fonte de evidência. Para
evitar o Frankenstein, cada lição passou por uma pergunta: *isso resolve um problema
fundamental do Germanio, que existe hoje no código ou na norma?* O que não passou está em
EVITAR ou foi omitido. Muitas lições convergem de várias linguagens; quando isso acontece, a
convergência é a evidência, não a linguagem.

Formato de cada item: **Lição** · **Fonte** (estudo deste diretório + URL) · **Problema do
Germanio** · **Arquivo afetado** · **Norma** (compatível, ou "contradiz: exige decisão
deliberada").

---

## ADOTAR

**A1. Diagnóstico como estrutura, não como string.**
Lição: código estável, span primário, spans secundários com label, origens relacionadas,
sugestões como edições (span + texto), texto derivado por renderizadores (CLI, JSON, LSP).
Fonte: [rust.md](rust.md) ([diagnostic structs](https://rustc-dev-guide.rust-lang.org/diagnostics/diagnostic-structs.html),
[JSON](https://doc.rust-lang.org/rustc/json.html)); [gleam.md](gleam.md)
([error.rs](https://github.com/gleam-lang/gleam/blob/main/compiler-core/src/error.rs));
[typescript.md](typescript.md) ([diagnostic.go](https://github.com/microsoft/TypeScript/blob/main/tsc/internal/ast/diagnostic.go)).
Problema: no dialeto de aplicação `teach` concatena texto e `errorf` devolve `fmt.Errorf` sem
código; o resolver descarta o caminho hierárquico; "como corrigir" é prosa; nada é
consumível por editor. Arquivo: `compiler/diagnostics/diagnostics.go`,
`compiler/parser/hierarquia.go:93-105`, `parser.go:32-38`, `resolver.go:88-90`.
Norma: compatível; o formato de 4 partes vira a renderização de texto. Detalhes em
[DIAGNOSTICS.md](DIAGNOSTICS.md).

**A2. Schema de seções como dado, com filhos proibidos.**
Lição: para cada seção, o papel de cada nível abaixo, a cardinalidade e o que é proibido; a
árvore interpretada por um schema fechado produz fatos tipados.
Fonte: [sintaxe-e-configuracao.md](sintaxe-e-configuracao.md) §3
([HCL body schema](https://github.com/hashicorp/hcl/blob/main/spec.md);
[CUE](https://cuelang.org/docs/reference/spec/): definições fechadas, campo não previsto é
erro). Problema: linhas abaixo de ações, campos, capacidades e alvos de `pertence a` são
ignoradas ou achatadas em silêncio (D1). Arquivo: `compiler/parser/hierarquia.go`;
também alimenta `vscode-germanio/tools/gerar_gramatica.py`. Norma: compatível; é o que
`INTENCAO.md:369` ("nunca ignoradas") exige.

**A3. Layout função só das linhas; o tipo de bloco decidido pela estrutura.**
Lição: nenhuma decisão de estrutura consulta o conteúdo nem o que o parser espera; nada de
`parse-error(t)`. Fonte: [sintaxe-e-configuracao.md](sintaxe-e-configuracao.md) §1
([Haskell 2010 §10.3](https://www.haskell.org/onlinereport/haskell2010/haskellch10.html),
[Adams, POPL 2013](https://michaeldadams.org/papers/layout_parsing/)). Problema: `isDataBlock`
decide pelo conteúdo da primeira linha filha, e um erro de digitação na primeira seção desfaz
o bloco (D4). Arquivo: `hierarquia.go:150-198`. Norma: compatível.

**A4. Fusão comutativa, associativa e idempotente, testada por propriedade.**
Lição: permutar e duplicar blocos e arquivos não muda o modelo; escalar conflitante mostra as
duas origens. Fonte: [sintaxe-e-configuracao.md](sintaxe-e-configuracao.md) §5
([CUE spec](https://cuelang.org/docs/reference/spec/),
[The Logic of CUE](https://cuelang.org/docs/concept/the-logic-of-cue/));
[typescript.md](typescript.md) (`relatedInformation`). Problema: o mesmo campo repetido é erro,
e o conflito real tem a mesma mensagem sem a segunda origem (D2). Arquivo:
`compiler/parser/resolver.go:250-264`, `compiler/parser/hierarquia_test.go`. Norma: compatível
(`INTENCAO.md:360-366`); a implementação diverge.

**A5. Spans (início e fim) em tokens e posições, com FileSet.**
Lição: `Pos` compacto + tabela de linhas por arquivo; `Position` calculada sob demanda;
conversão para UTF-16 só na borda do LSP. Fonte: [go.md](go.md)
([go/token](https://pkg.go.dev/go/token)); [typescript.md](typescript.md)
([lsconv](https://github.com/microsoft/TypeScript/blob/main/tsc/internal/ls/lsconv/converters.go));
[python.md](python.md) ([PEP 657](https://peps.python.org/pep-0657/)). Problema: `Token` e
`Position` só têm início (`lexer.go:214-224`, `diagnostics.go:9-15`); não há o que sublinhar,
substituir ou recortar; a origem de `pertence a` cai na linha errada (D10). Arquivo:
`compiler/lexer/lexer.go`, `compiler/diagnostics`, `compiler/ast`. Norma: compatível.

**A6. Um front-end para todas as ferramentas.**
Lição: formatter, `ge graph`, `ge explain`, VS Code e LSP pedem a árvore e os fatos ao
front-end; nenhum tokeniza ou mede indentação. Cópias da gramática morrem.
Fonte: [typescript.md](typescript.md), [gleam.md](gleam.md), [zig.md](zig.md),
[python.md](python.md) (lib2to3, [What's New 3.9](https://docs.python.org/3/whatsnew/3.9.html)),
[swift.md](swift.md) (dois parsers). Problema: cinco leituras independentes da sintaxe e três
cópias da tabela de seções dentro do próprio parser (TOOLING.md §1); G21 aberto. Arquivo:
`tooling/formatter/intencao.go:127-145`, `tooling/intelligence/analyzer.go:180-181`,
`vscode-germanio/tools/gerar_gramatica.py`, `hierarquia.go:127-135, 434-436`. Norma: compatível.

**A7. Testes de saída inteira (snapshot) e de correção aplicada (run-rustfix).**
Lição: cada caso de erro guarda a saída renderizada inteira e o arquivo corrigido; o teste
aplica as sugestões, reparseia e compara os fatos; modo de regravação explícito. Baselines
também para `explain`, `fmt` e `graph`. Fonte: [rust.md](rust.md)
([UI tests](https://rustc-dev-guide.rust-lang.org/tests/ui.html)); [typescript.md](typescript.md)
(baselines, fourslash); [gleam.md](gleam.md). Problema: os testes de erro conferem
substrings; uma regressão de mensagem ou uma sugestão errada passam. Arquivo:
`compiler/parser/hierarquia_test.go`, novo diretório de casos. Norma: compatível; torna
verificáveis as mensagens exigidas por `INTENCAO.md` › Erros.

**A8. Todos os diagnósticos de uma vez, sem cascata.**
Lição: acumular num coletor único (`ErrorBundle` do Zig) e não emitir erro derivado de um erro
já emitido (`ErrorGuaranteed` do Rust); unidade de continuação = bloco de dado.
Fonte: [zig.md](zig.md); [rust.md](rust.md)
([ErrorGuaranteed](https://rustc-dev-guide.rust-lang.org/diagnostics/error-guaranteed.html));
[gleam.md](gleam.md) ([v1.2](https://gleam.run/news/fault-tolerant-gleam/));
[elixir.md](elixir.md) ([v1.15](https://elixir-lang.org/blog/2023/06/19/elixir-v1-15-0-released/)).
Problema: um erro por execução obriga o leigo a um ciclo corrigir-um-rodar-de-novo (D3).
Arquivo: `compiler/parser/parser.go`, `hierarquia.go`, `resolver.go`,
`compiler/semantic/check.go`. Norma: compatível. (A política detalhada de recuperação com nó
ERROR está em I1.)

**A9. Verbos de várias palavras vêm do léxico das capabilities, não de `if` no parser.**
Lição: maior casamento contra um léxico fechado que cada capability declara.
Fonte: [sintaxe-e-configuracao.md](sintaxe-e-configuracao.md) §3(c); [nim.md](nim.md), lição 9
(seções não registráveis por bibliotecas, mas léxico fechado do core). Problema: `enviar
código`, `baixar código` e `branch padrão` estão escritos no parser e no resolver (D5).
Arquivo: `hierarquia.go:404,419`, `intencao.go:257,697`, `resolver.go:1031-1085`. Norma:
compatível com a regra das três camadas (`CLAUDE.md` › Três camadas).

**A10. GEP: um artefato numerado por decisão de linguagem, com escopo, status, IDs nunca
reutilizados e lista de rejeitados.**
Fonte: [swift.md](swift.md) ([process.md](https://github.com/swiftlang/swift-evolution/blob/main/process.md),
[commonly_proposed](https://github.com/swiftlang/swift-evolution/blob/main/commonly_proposed.md));
[python.md](python.md) ([PEP 1](https://peps.python.org/pep-0001/)); [go.md](go.md)
([proposal README](https://github.com/golang/proposal/blob/master/README.md)). Problema:
decisões espalhadas em `AGENT_STATE.md`, na pesquisa e em `GERMANIO_EVOLUTION.md`; IDs
duplicados em GAPS (D8); o Gate não distingue bug de design. Arquivo: `docs/gep/` (criado em
paralelo a este estudo, GEP 0001 com status Aceita); `AGENTS.md`; `GERMANIO_GAPS.md`. Norma:
acrescenta processo; a integração ao Documentation Gate de `AGENTS.md` **exige decisão
deliberada**. Base de pesquisa em [LANGUAGE_EVOLUTION.md](LANGUAGE_EVOLUTION.md).

**A11. Catálogo único de diagnósticos como dado.**
Lição: código, categoria, mensagem com parâmetros, por quê, como corrigir, grupo educativo e
exemplo testado; códigos nunca reutilizados. Fonte: [typescript.md](typescript.md)
([diagnosticMessages](https://github.com/microsoft/TypeScript/tree/main/tsc/internal/diagnostics));
[swift.md](swift.md) ([SE-0443](https://github.com/swiftlang/swift-evolution/blob/main/proposals/0443-warning-control-flags.md));
[rust.md](rust.md) ([error codes](https://rustc-dev-guide.rust-lang.org/diagnostics/error-codes.html)).
Problema: `Explanations` cobre só o núcleo estrito, uma linha por código, sem exemplo; os erros
da camada de intenção não têm código; GE1002 carrega uma convenção contrária (D6). Arquivo:
`compiler/diagnostics/diagnostics.go:36-56`. Norma: compatível.

**A12. Nome parecido pelo algoritmo testado, em todos os namespaces.**
Lição: igualdade sem caixa primeiro; distância até um terço do tamanho (mínimo 1); procurar
também entre papéis, ações e dados, não só entre seções. Fonte: [rust.md](rust.md)
([edit_distance](https://doc.rust-lang.org/nightly/nightly-rustc/rustc_span/edit_distance/fn.find_best_match_for_name.html),
[name resolution](https://rustc-dev-guide.rust-lang.org/name-resolution.html));
[gleam.md](gleam.md). Problema: `suggest` usa limite fixo `< 3` só contra seções
(`hierarquia.go:439-450`); `unknownLine` só aceita distância exatamente 1 contra uma lista
curta (`parser.go:182-188`), por isso `tme` na primeira seção não recebe sugestão. Norma:
compatível.

**A13. Mostrar as palavras na língua em que a pessoa escreveu.**
Fonte: [gleam.md](gleam.md) ([v1.6](https://gleam.run/news/context-aware-compilation/)).
Problema: `displaySections` e as sugestões saem sempre em português, embora `Token.Raw` guarde
a grafia original; quem escreve `access` recebe `acesso`. Arquivo: `hierarquia.go:434-450`,
`lexer.go:220-223`. Norma: compatível com o multilinguismo.

**A14. Nomes que só diferem pela dobra de acento são conflito, não identidade.**
Lição: aceitar grafias equivalentes para palavras da linguagem exige, para nomes do domínio,
verificar a consistência por uso e acusar colisão. Fonte: [nim.md](nim.md), lições 5-7 (RFC
456, `--styleCheck:usages`). Problema: `foldWord` dobra acentos também nos nomes (D7).
Arquivo: `compiler/parser/intencao.go:30,35-36`, `declarativo.go:292`, `resolver.go`.
Norma: `INTENCAO.md:288-289` diz que "palavras em português podem ser escritas com ou sem
acento"; se isso vale para nomes do domínio, a lição **contradiz: exige decisão deliberada**.

**A15. Exemplos normativos executáveis (doctest).**
Lição: os blocos `ge` de `INTENCAO.md` são extraídos e passam por `ge check` (e, quando têm
fatos esperados, por `ge explain`). Fonte: [elixir.md](elixir.md); [go.md](go.md)
([examples](https://go.dev/blog/examples)). Problema: a norma e o parser podem divergir sem
que nada acuse; é mais um "parser" (o do documento). Arquivo: testes em `compiler/parser`,
`docs/INTENCAO.md` (só leitura). Norma: compatível ("Exemplos normativos devem virar testes",
`INTENCAO.md:754`).

**A16. Um binário `ge`.**
Fonte: [zig.md](zig.md), [gleam.md](gleam.md), [go.md](go.md). Problema: `ge` e `germanio`
coexistem (`main.go:14-16`, `cli/cli.go`), `build` e `docker` só no legado. Arquivo:
`main.go`, `cli/cli.go`, `tooling/gecli/cli.go`. Norma: compatível; a aposentadoria do legado
segue a lição de deprecação (P7).

**A17. `ge lsp` padrão, no mesmo binário, consumidor do mesmo front-end.**
Fonte: [typescript.md](typescript.md) (port em Go, [native port](https://devblogs.microsoft.com/typescript/typescript-native-port/));
[nim.md](nim.md) (nimsuggest); [roc.md](roc.md). Problema: não há LSP; o VS Code só tem
TextMate. Arquivo: `tooling/gecli`, `vscode-germanio/`. Norma: compatível. Pré-requisitos:
A1, A5, A8.

**A18. Mensagem ruim é bug.**
Lição: um tipo de issue "diagnóstico confuso" (versão, entrada, mensagem esperada); cada
correção vira caso de snapshot. Fonte: [zig.md](zig.md) (`error_message.yml`);
[elixir.md](elixir.md). Problema: o público é leigo e a mensagem é o caminho da falha.
Arquivo: processo do repositório, `compiler/diagnostics/diagnostics_test.go`. Norma: compatível.

**A19. Avisar só o que é verificado; "não sei" vira pergunta ou erro.**
Fonte: [elixir.md](elixir.md) (type checker da v1.18+). Problema: um aviso incerto ensina o
leigo a ignorar avisos; e o Germanio promete determinismo. Arquivo: `compiler/semantic`,
`resolver.go`, `tooling/explicar`. Norma: compatível com "inferência ambígua … deve gerar erro
educativo" (`INTENCAO.md` › O que existe).

**A20. Regra escrita: comentário nunca muda fato.**
Fonte: [roc.md](roc.md); [sintaxe-e-configuracao.md](sintaxe-e-configuracao.md) §6 (Dhall:
hash sobre a forma normal). Problema: é o que torna segura a verificação de significado do
`ge fmt` e qualquer `ge fix`. Arquivo: `docs/INTENCAO.md` › Layout (texto a acrescentar, por
decisão). Norma: compatível; explicita o implícito.

## ADAPTAR

**P1. Fatos gerados direto da árvore; a frase plana vira renderização do fato.**
Lição: denotação por recursão estrutural sobre a árvore de linhas; a frase plana tem a sua
própria denotação no mesmo conjunto de fatos; `ge explain` e "Equivale a:" geram a frase a
partir do fato. Fonte: [sintaxe-e-configuracao.md](sintaxe-e-configuracao.md) §3 e §11
([Inform 7 calculus](https://ganelson.github.io/inform/calculus-module/index.html)). Adaptação:
sem cálculo de predicados; só a ideia de que a superfície não é o significado. Problema: a
síntese de tokens (`synth`, `placed`) é uma IR escondida que causa posições erradas (D10),
diagnósticos sobre a frase que o usuário não escreveu (D3) e cirurgia de tokens por seção.
Arquivo: `hierarquia.go:213-432`, `intencao.go`, `compiler/ast/intencao.go`,
`tooling/explicar`. Norma: compatível ("os dois produzem o mesmo fato", `INTENCAO.md:235-237`);
a norma não exige a redução por reparse, só a equivalência.

**P2. Applicability com regra própria: automática só com fatos idênticos; o que cria
permissão é, no máximo, provável.**
Fonte: [rust.md](rust.md) ([Applicability](https://doc.rust-lang.org/nightly/nightly-rustc/rustc_errors/enum.Applicability.html),
[cargo fix](https://doc.rust-lang.org/cargo/commands/cargo-fix.html)); [swift.md](swift.md)
("single, obvious, and very likely correct"). Adaptação: das duas condições do
`MachineApplicable`, só "preserva o significado" é aceita, porque é verificável por reparse;
nenhuma edição que cria grant, papel, regra de acesso ou integração é automática. Problema:
sem regra, um editor ou `ge fix` poderia conceder permissões por inferência (o caso
`projetos / developer / excluir`). Arquivo: `compiler/diagnostics`, futuro `ge fix`.
Norma: regra nova; **exige decisão deliberada**. Detalhes em [DIAGNOSTICS.md](DIAGNOSTICS.md) §5-6.

**P3. Hipótese a partir de outro namespace e da forma da subárvore.**
Fonte: [rust.md](rust.md); [gleam.md](gleam.md) (`type_with_name_in_scope`). Adaptação: a
evidência "é um papel" é do resolver (o papel pode estar em outro arquivo); a evidência "os
filhos são ações" é local ao parser. Problema: `developer` como seção recebe só a lista de
seções. Arquivo: `hierarquia.go:285-289`, `resolver.go`. Norma: compatível.

**P4. Árvore de linhas sem perdas, única, impressa pelo formatter.**
Fonte: [sintaxe-e-configuracao.md](sintaxe-e-configuracao.md) §2
([rust-analyzer](https://github.com/rust-lang/rust-analyzer/blob/master/docs/book/src/contributing/syntax.md),
[Roslyn](https://github.com/dotnet/roslyn/blob/main/docs/wiki/Roslyn-Overview.md));
[gleam.md](gleam.md) (comentários em tabela lateral); [zig.md](zig.md) (renderizador).
Adaptação: sem perdas **de linha**, não de token, porque o `.ge` não tem expressões
multilinha. Problema: o formatter recalcula a pilha de níveis (`intencao.go:127-145`) com uma
regra diferente da de `layoutTree`; o lexer descarta comentários. Arquivo: `lexer.go:495-500`,
`hierarquia.go`, `declarativo.go:22`, `tooling/formatter/intencao.go`. Norma: compatível.

**P5. Uma forma normal única de fatos.**
Fonte: [sintaxe-e-configuracao.md](sintaxe-e-configuracao.md) §6
([Dhall](https://docs.dhall-lang.org/discussions/Safety-guarantees.html)). Adaptação: vale para
os níveis 1 e 2; corpos de nível 3 entram como texto verbatim, sem pretensão de igualdade
semântica. Problema: o formatter compara o programa antes do resolver (`intencao.go:44-60`) e
os testes comparam o modelo resolvido. Arquivo: `tooling/formatter`, `tooling/explicar`,
testes de equivalência. Norma: compatível.

**P6. Blame: todo fato e todo erro derivado carregam a origem.**
Fonte: [sintaxe-e-configuracao.md](sintaxe-e-configuracao.md) §7
([Nickel contracts](https://nickel-lang.org/user-manual/contracts)); [nim.md](nim.md), lição
10 (proveniência na IR). Adaptação: só o conceito de culpa; sem contratos em runtime.
Problema: erros do resolver sem "Onde" para fatos de bloco (D3); `ast.App` sem mapa de origem.
Arquivo: `resolver.go`, `compiler/ast`, `tooling/explicar`. Norma: compatível (pendência
"proveniência de inferências", `INTENCAO.md` › Decisões e limites).

**P7. Deprecação com prazo escrito e migração verificada.**
Fonte: [elixir.md](elixir.md) ([compatibility](https://elixir.hexdocs.pm/compatibility-and-deprecations.html));
[python.md](python.md) ([PEP 387](https://peps.python.org/pep-0387/)); [gleam.md](gleam.md)
([v0.27](https://gleam.run/news/v0.27-hello-panic-goodbye-try/)); [rust.md](rust.md)
(editions). Adaptação: a migração (`ge fmt`/`ge fix`) só é automática se provar por reparse
que os fatos são os da forma nova. Problema: não há mecanismo de deprecação, e a
"refatoração retroativa" vai aposentar formas; o legado `germanio` precisa de prazo.
Arquivo: `compiler/diagnostics`, `tooling/formatter`, `docs/INTENCAO.md` › Evolução.
Norma: acrescenta contrato de compatibilidade; **exige decisão deliberada**.

**P8. Erros especializados como segunda passada da gramática.**
Fonte: [python.md](python.md) (regras `invalid_*`,
[What's New 3.10](https://docs.python.org/3/whatsnew/3.10.html)). Adaptação: sem PEG; no
Germanio, a segunda passada é o schema de seções: para cada seção, a lista dos erros
conhecidos e a mensagem que nomeia a linha dona ("esperava itens abaixo de `tem` (linha
N)"). Problema: mensagens escritas caso a caso no ponto de falha. Arquivo: `hierarquia.go`,
`compiler/diagnostics`. Norma: compatível.

**P9. `ge check` como driver de analisadores.**
Fonte: [go.md](go.md) ([go/analysis](https://pkg.go.dev/golang.org/x/tools/go/analysis)).
Adaptação: sem fatos entre pacotes no início; cada regra de contradição, ambiguidade e
segurança é um analisador sobre `ast.App`, com teste. Problema: regras espalhadas em
`compiler/semantic`, `tooling/intelligence/validator.go` e resolver. Norma: compatível
(resolve a pendência "cobertura dos diagnósticos de contradição e ambiguidade").

**P10. Checagem por arquivo antes da checagem do projeto.**
Fonte: [zig.md](zig.md) (`ast-check`). Adaptação: a fronteira já existe (`ast.Intent` por
arquivo, `ast.App` por projeto); diagnósticos que dependem só do arquivo saem na primeira
fase, com resposta imediata no editor. Arquivo: `hierarquia.go`, `resolver.go`. Norma:
compatível.

**P11. Fronteira entre camadas verificada pelo compilador.**
Fonte: [roc.md](roc.md) (plataformas, "no escape hatches"; tratado como experimento, não como
prova). Adaptação: a ideia não depende do Roc: é a própria regra das três camadas virando
verificação em `ge check` (domínio não cita primitiva de nível 4/5; adaptador não declara
produto). Problema: hoje a verificação é parcial (`checkRole` em `runtime/engine.go`, segundo
o estudo) e roda no carregamento, não no `ge check`. Norma: compatível (`SKILL.md` §34b).

**P12. Inferir consequências pelo grafo, com a cadeia no `ge explain`.**
Fonte: [roc.md](roc.md) (pureza inferida). Adaptação: propriedades de domínio ("exige login",
"gera notificação", "altera estado"), nunca notação de tipos. Problema: o autor declara o que
é consequência. Arquivo: `tooling/explicar`, `tooling/intelligence/graph.go`. Norma:
compatível; a condição é que `ge explain` mostre a cadeia (senão vira dependência escondida).

**P13. Versionar o estilo do formatter junto com a versão da linguagem, sem opções.**
Fonte: [rust.md](rust.md) ([RFC 3338](https://rust-lang.github.io/rfcs/3338-style-evolution.html));
[go.md](go.md) ([go/format](https://pkg.go.dev/go/format)). Problema: mudar a saída canônica
quebraria o CI de quem roda `ge fmt --check`. Norma: compatível.

**P14. Workspace com snapshot imutável e texto do editor.**
Fonte: [typescript.md](typescript.md) ([snapshot.go](https://github.com/microsoft/TypeScript/blob/main/tsc/internal/project/snapshot.go)).
Adaptação: sem incremental semântico. Problema: `Compilar` lê do disco. Arquivo:
`runtime/engine.go`, novo pacote em `tooling/`. Norma: compatível.

## EVITAR

| # | O que evitar | Fonte | Por que não resolve um problema do Germanio (ou cria um) |
|---|---|---|---|
| E1 | layout pela coluna do primeiro token e fechamento de bloco por `parse-error(t)` | Haskell, F#; [Adams](https://michaeldadams.org/papers/layout_parsing/) | a aceitação passaria a depender da recuperação de erro |
| E2 | macros, `use`, DSLs e seções registráveis por bibliotecas | [elixir.md](elixir.md), [nim.md](nim.md) | quebra `ge check`/`ge explain` determinísticos e o teste do leigo; capability nova entra pelo core |
| E3 | formatter configurável, opções por pacote, dois formatters | [rust.md](rust.md), [swift.md](swift.md), [nim.md](nim.md), [elixir.md](elixir.md) | cada opção é um formato a mais; o leigo não escolhe estilo |
| E4 | dois parsers mantidos à mão para a mesma linguagem | [swift.md](swift.md), [go.md](go.md) | é o risco "vários parsers" já existente; se houver dois, gerar um do outro e testar |
| E5 | muitas formas para a mesma coisa, tipos implícitos pela aparência, âncoras | [YAML](https://yaml.org/spec/1.2.2/), [yaml bool](https://yaml.org/type/bool.html) | dependência escondida e implementações que não concordam |
| E6 | prioridades de merge, `force`, amending com late binding | [Nickel](https://nickel-lang.org/user-manual/merging), [Pkl](https://pkl-lang.org/main/current/language-reference/index.html) | sobrescrita contraria "conflito é erro" |
| E7 | red-green trees, reparse incremental, queries, HIR separado, AST tipada sobre o CST | rust-analyzer, Roslyn, rustc | escala e número de consumidores que o Germanio não tem; reavaliar só com medição |
| E8 | infraestrutura de tradução de mensagens separada do código | [MCP #959](https://github.com/rust-lang/compiler-team/issues/959) | o próprio Rust a está removendo; A13 resolve o multilinguismo pelo `Token.Raw` |
| E9 | PEG com escolha ordenada | [python.md](python.md) | a ordem das alternativas viraria semântica escondida |
| E10 | continuação de linha por recuo | [nim.md](nim.md) | ambiguidade filho × continuação; a norma já proíbe |
| E11 | Hindley-Milner, solver de restrições, type classes | [gleam.md](gleam.md), [swift.md](swift.md) | nenhum problema do nível padrão pede; inferência deve seguir tabelas determinísticas |
| E12 | protocolo próprio de editor; LSP fora do projeto | [typescript.md](typescript.md) (tsserver), [zig.md](zig.md) | LSP padrão no mesmo binário basta |
| E13 | build como programa, alocadores explícitos, `comptime`, "não usado" como erro | [zig.md](zig.md) | conceitos técnicos expostos ao autor |
| E14 | rodar programa com erro de compilação | [roc.md](roc.md) | contradiz o `ge check` como portão |
| E15 | algoritmo de Wadler no formatter | [roc.md](roc.md) | o `.ge` não quebra linhas |
| E16 | linter separado com centenas de regras configuráveis | [rust.md](rust.md) (Clippy) | o leigo não escolhe lints |
| E17 | modos de linguagem e flags de recurso enquanto todo `.ge` está no repositório | [swift.md](swift.md) | migrar na mesma unidade é mais simples |
| E18 | estilo de mensagem para programador ("expected X, found Y") | [rust.md](rust.md) | público leigo |
| E19 | duas sintaxes concretas para o mesmo modelo | [HCL JSON](https://github.com/hashicorp/hcl/blob/main/json/spec.md) | `ge fmt` já dá a forma canônica; as formas plana e hierárquica são outra coisa (as duas são para pessoas) |

## INVESTIGAR

| # | O quê | Fonte | Condição para decidir |
|---|---|---|---|
| I1 | política de recuperação: nó ERROR, sincronização por linha, supressão de cascata, limite por arquivo | [sintaxe-e-configuracao.md](sintaxe-e-configuracao.md) §8 ([swift-syntax](https://github.com/swiftlang/swift-syntax), [Resilient LL](https://matklad.github.io/2023/05/21/resilient-ll-parsing-tutorial.html), [hcl/v2](https://pkg.go.dev/github.com/hashicorp/hcl/v2)) | validar com os exemplos reais (GitLab) e com casos de snapshot antes de adotar |
| I2 | unificar os dois front-ends (núcleo estrito e aplicação) seguindo o roteiro do PEP 617 | [python.md](python.md), [typescript.md](typescript.md) | exige decisão sobre `SPEC.md`, que é normativo para o núcleo |
| I3 | relações de indentação na gramática (Adams) ou pseudo-terminais (Nim) para escrever o layout na EBNF | [Adams](https://michaeldadams.org/papers/layout_parsing/), [nim.md](nim.md) | hoje a EBNF usa `ABRE`/`FECHA`, que o código não emite; decidir qual é a definição normativa |
| I4 | resolução sob demanda (request evaluator, checker preguiçoso) para hover em apps grandes | [swift.md](swift.md), [typescript.md](typescript.md) | medir `Compilar` no GitLab inteiro |
| I5 | `ge reduzir`: minimizar um `.ge` que reproduz um erro | [zig.md](zig.md) (`zig reduce`) | depende do printer sobre a árvore |
| I6 | declarar uma "v1" da sintaxe de intenção, depois da poda | [gleam.md](gleam.md) ([v1](https://gleam.run/news/gleam-version-1/)) | primeiro usuário externo |
| I7 | versão da linguagem declarada no projeto + tabela de mudanças (GODEBUG) | [go.md](go.md) ([compat](https://go.dev/blog/compat)) | primeiro `.ge` fora do repositório |
| I8 | hash da forma normal para CI ("esta refatoração não mudou a aplicação") | Dhall | barato depois de P5 |
| I9 | `ge check` recusar capability que o alvo não oferece | [roc.md](roc.md) | depende de um manifesto de capabilities por alvo |
| I10 | processos supervisionados e memória por requisição no runtime | [elixir.md](elixir.md), [roc.md](roc.md) | fase de tempo real e grande escala; mecanismo do core, nunca sintaxe |
| I11 | MVS para adaptadores versionados | [go.md](go.md) ([vgo-mvs](https://research.swtch.com/vgo-mvs)) | só com ecossistema de adaptadores de terceiros |

---

## Princípio de subtração: o que o Germanio pode deixar de exigir do programador

A pergunta não é "o que acrescentar", e sim "o que a pessoa não precisa mais saber, escrever
ou fazer", sob duas condições: a inferência é determinística e `ge explain` a mostra com a
origem. Onde a inferência seria ambígua, o Germanio pergunta ou recusa (A19), nunca adivinha.

| O que deixa de ser exigido | Como | Lição | Estado |
|---|---|---|---|
| repetir o sujeito em cada frase | o caminho do bloco o fornece | norma atual | existe |
| declarar o dado antes do bloco (`tenha`) | o bloco declara | norma atual | existe |
| escolher estilo, recuo, alinhamento | `ge fmt` canônico sem opções | E3 | existe |
| corrigir um erro por vez | todos os diagnósticos por execução, sem cascata | A8 | falta |
| adivinhar a correção de uma mensagem | sugestão como edição verificada; automática só se os fatos não mudam | A1, P2 | falta |
| saber em que seção uma linha deveria estar, quando a forma da subárvore a determina | hipótese por outro namespace e pela forma; sempre como sugestão provável se criar permissão | P3, P2 | falta |
| ler mensagens em português escrevendo em outro idioma | renderizar com `Token.Raw` | A13 | falta |
| distinguir se uma linha foi ignorada | nenhuma linha é ignorada: filho não previsto é erro | A2 | falta (D1) |
| saber que repetir um fato em outro bloco é seguro | fusão idempotente | A4 | falta (D2) |
| declarar consequências ("exige login", "notifica") | inferência pelo grafo, cadeia no `ge explain` | P12 | falta |
| escolher flags de compatibilidade | versão declarada uma vez por `ge init`; o significado antigo é preservado | I7, P7 | futuro |
| migrar à mão uma forma aposentada | `ge fix`/`ge fmt` com prova por reparse | P7 | futuro |
| configurar banco, pool, cache, transação | o runtime decide ("complexidade com um dono e poucos botões", o GC do Go) e `ge explain` diz o que foi decidido | [go.md](go.md) | parcialmente |
| aprender um segundo "dialeto" de layout (2 × 4 espaços) | uma convenção só | D6 | falta |
| escrever tipos quando o nome ou o uso bastam | tabela de tipos pelo nome; explícito sempre vence; ambíguo é erro | norma atual, A19 | existe |
| configurar proteção contra tentativa de senha em massa | bloqueio da conta por padrão (10 tentativas, 10 minutos) | G83 | existe (conferido em `0c12051`); falta o limite por IP no login de intenção e mostrar os valores no `ge explain` |
| autenticar e filtrar o tempo real | `/ws` exige sessão e a mesma origem; o aviso leva só tipo, modelo e id | G65 | existe; salas por destinatário faltam (G66) |
| esconder arquivos ocultos e listagens das pastas servidas | pastas servidas só entregam arquivos | G91 | existe |
| saber que o dialeto anterior não protegia ordem, proxy, cadastro, senha e upload | as mesmas defesas em todo caminho | G98–G101, A23–A25 | existe (conferido em `0c12051`) |
| escolher limite, cancelamento e coleta de erro do trabalho concorrente | concorrência estruturada no runtime | A21, A22, P19 | falta (G89, G96; [GEP 0005](../../gep/0005-concorrencia-por-intencao.md)) |
| pagar por capabilities que o app não declara | só inicializa e só embute o que é declarado | A28 | falta (G90) |
| escrever migração ou temer perder dados ao renomear | snapshot do esquema e diff explicado | P16 | falta (G93) |
| conhecer as palavras de 19 idiomas que não pode usar como nome | um idioma por arquivo | A32 | falta (G92; [GEP 0007](../../gep/0007-idiomas-da-intencao.md)) |

**Limites da subtração** (o que o Germanio **não** deve deixar de exigir, porque removeria
contexto ou decidiria pela pessoa): o aspecto do bloco (`acesso`, `tem`...; `INTENCAO.md:242`);
a concessão de qualquer permissão (nunca inferida nem aplicada sozinha, P2); a escolha entre
duas leituras quando um conflito ou ambiguidade existe; o valor de um escalar único declarado
com dois valores. "Germanio não busca o menor número de caracteres; busca o menor número de
conceitos" (`INTENCAO.md:241`).

---

## Divergências encontradas entre a norma e a implementação

Todas conferidas no código da revisão `fc31daf`. "Reproduzido" significa: `go build ./cmd/ge`
e `ge check` / `ge explain` sobre arquivos temporários fora do repositório (no scratchpad da
sessão). Cada divergência é classificada como o `AGENTS.md` pede (implementação errada,
especificação incompleta ou decisão pendente), mas **não foi registrada** em `GERMANIO_GAPS.md`:
isso fica para o orquestrador.

**D1. Linhas abaixo de ações, campos, capacidades e alvos são ignoradas ou achatadas.**
Reproduzido.
- `acesso › developer › ver › (linha a mais)`: `ge check` passa e `ge explain` só mostra
  `developer pode ver projetos`; a linha some. `access` percorre `actor.children` e nunca
  visita `a.children` (`compiler/parser/hierarquia.go:401-423`).
- `tem › nome obrigatório › email` (recuada abaixo do campo): vira o campo irmão `email`
  (`ge explain`: "projeto tem email"), porque `flattenLines` achata a subárvore
  (`hierarquia.go:108-115`, usada em `hierarquia.go:294`).
- `pode › fechar › reabrir`: vira `projeto pode reabrir` irmão (`hierarquia.go:299`).
- `pertence a › grupo opcional › (linha a mais)`: a linha é descartada; o laço usa só
  `c.line` (`hierarquia.go:304-310`).
Norma violada: "Linhas que nenhuma construção reconhece são erro, nunca ignoradas"
(`INTENCAO.md:369`). Classe: implementação errada. Lição: A2.

**D2. O mesmo fato de campo repetido dá erro; o conflito real não mostra a segunda origem.**
Reproduzido. Dois blocos `projetos › tem › nome obrigatório`: `projeto já tem o campo nome`.
`nome obrigatório` contra `nome opcional`: a **mesma** mensagem, uma origem só.
Código: `compiler/parser/resolver.go:258-262` (compara só o nome; `errAt` não tem segunda
posição, `resolver.go:88-90`). Grants repetidos são aceitos (conferido), então a fusão é
idempotente para acesso e não para campos. Norma violada: "O mesmo fato repetido não muda
nada. Um valor que só pode ter um valor … é erro que mostra as duas origens"
(`INTENCAO.md:363-365`). Classe: implementação errada. Lição: A4.

**D3. Erros avaliados na frase plana sintetizada; um erro por execução.**
Reproduzido. `acesso › developer › enviar códigu` + `maintainer › excluirr` no mesmo arquivo:
saída única `b3b.ge:12: não conheço "codigu". Declare com: tenha codigu (dados declarados:
projetos)`. Sem "Onde", sem "Por quê", sem "Equivale a", com uma correção errada (declarar um
dado `codigu`), porque o erro foi detectado na frase sintetizada `developer pode enviar codigu
projetos` (`hierarquia.go:400-427`) e emitido pelo resolver (`resolver.go:101`) com `errAt`,
que descarta `Position.Context` (`resolver.go:88-90`). O segundo erro (`excluirr`) não aparece.
Norma violada: "Todo erro de layout ou de contexto diz o que aconteceu, onde (arquivo:linha e
caminho do bloco), por que e como corrigir, e … a frase plana equivalente" (`INTENCAO.md:369-372`).
Classe: implementação errada. Lições: A1, A8, P1, P6. Não confirmado: a observação do estudo
de que a lista "dados declarados" incluía `maintainer` (aqui a lista foi só `projetos`).

**D4. Erro de digitação na primeira seção desfaz o bloco.**
Reproduzido. `projetos › tme › nome obrigatório` dá `não entendi a linha "projetos"` sem
sugestão; `acesos` como segunda seção dá `"acesos" não é uma seção de projetos … (você quis
dizer "acesso"?)`. Causa: `isDataBlock` só reconhece o bloco se a primeira linha filha
começar com uma seção (`hierarquia.go:197`); a linha cai em `unknownLine`
(`compiler/parser/parser.go:173-192`), cuja sugestão só aceita distância exatamente 1
(`parser.go:185`), e `tme`→`tem` tem distância 2. Norma violada: `INTENCAO.md` › Testes
normativos, "contexto": "seção desconhecida … são erros com sugestão". Classe: implementação
errada. Lições: A3, A12.

**D5. Vocabulário específico no parser do core.**
Confirmado no código. `access` trata `enviar codigo`/`baixar codigo` como verbo de duas
palavras e `branch_padrao` como alvo especial (`hierarquia.go:404, 419`); o mesmo em
`compiler/parser/intencao.go:257-258, 697-698` e `resolver.go:1031-1032, 1056, 1082-1085,
1222`. Ressalva: a própria tabela normativa de seções inclui `acesso › somente maintainer ›
enviar código para a branch padrão` (`INTENCAO.md` › Tabela de seções), e `projeto tem
repositório` é capability genérica. Classe: **decisão pendente** entre "vocabulário da
capability genérica `repositório`" (então deve vir do léxico da capability, A9) e "vocabulário
de produto no core" (proibido por `CLAUDE.md` › Três camadas). Em nenhum dos casos o `if` no
parser é o mecanismo certo.

**D6. GE1002 diz "dois espaços"; a forma canônica da camada de intenção é 4.**
Confirmado no código. `compiler/diagnostics/diagnostics.go:38` ("Use dois espaços por bloco")
e `compiler/parser/germanio.go:50, 72, 87` ("Use dois espaços por nível", "mais dois
espaços"). O lexer do núcleo conta tab como 2 espaços antes de o parser recusar
(`lexer.go:463`). Ressalva: `SPEC.md:27` fixa dois espaços para o núcleo estrito, então cada
front-end segue a sua norma; a divergência é **entre as normas** (`SPEC.md` × `INTENCAO.md:277`)
sob o mesmo código de erro, e o leigo vê um único Germanio. Classe: decisão pendente. Lições:
A11, I2.

**D7. `foldWord` dobra acentos também dos nomes do domínio.**
Reproduzido. `foldWord` (`compiler/parser/intencao.go:35-36`) é aplicado a toda palavra
(`intencao.go:30`) e ao nome de campo (`declarativo.go:292`). Consequências observadas:
`tem › secretaria` + `secretária` → `clinica já tem o campo secretaria`; `regras › arquivado
e somente leitura` (conjunção `e`) é aceito e explicado como `projeto arquivado é somente
leitura`; a seção `pôde` é aceita como `pode` (`clinica pode fechar`). Norma: `INTENCAO.md:288-289`
permite escrever "palavras em português" com ou sem acento, sem dizer se vale para nomes do
domínio. Classe: especificação incompleta; exige decisão. Lição: A14.

**D8. IDs duplicados em `GERMANIO_GAPS.md`.**
Confirmado (corrigido depois: renumerados para G61-G64). G57, G58 e G59 apareciam duas vezes cada, com significados diferentes
(`GERMANIO_GAPS.md:70-75`). Classe: erro de registro. Lição: A10 (IDs nunca reutilizados).

Achados laterais durante a verificação (fora da lista pedida):

**D9. `ge check` entra em pânico sem `crie sistema`.** **Corrigido** (conferido em `0c12051`:
o mesmo arquivo dá "programa verificado — 1 dados"; G73 continua aberto pelos outros itens).
Texto original: Reproduzido. Um arquivo de intenção
válido sem `crie sistema` termina em `panic: runtime error: invalid memory address or nil
pointer dereference` em `tooling/gecli/cli.go:225` (`prog.System.Name` com `prog.System == nil`);
`ge explain` no mesmo arquivo funciona. Classe: implementação errada.

**D10. A origem de `pertence a` aponta para a linha da seção, não do item.** Reproduzido:
`projeto pertence a grupo opcional — b1c.ge:12`, sendo 12 a linha `pertence a` e 13 a do item;
o sujeito sintetizado é posicionado no token da seção (`hierarquia.go:277, 306`). Classe:
implementação errada (`INTENCAO.md` › Inspeção pede a origem do fato). Lições: A5, P1.

**D11. Texto não fechado: mensagem em inglês.** Reproduzido: `erro léxico em s1.ge:
unterminated string at line 1, column 8` (`compiler/lexer/lexer.go:684, 712`). Não reproduzido:
o estudo do Python aponta que, no modo do núcleo, o `Diagnostic` GE1001 recebe a posição em
que a leitura parou (`l.line/l.col`, `lexer.go:402`) e não a de abertura; o código confirma o
uso de `l.line/l.col`, mas o caminho não foi exercitado aqui.

**D12. Lista de papéis com item vazio.** Reproduzido: `papel "maintainer" não declarado. Use
tenha papeis ou um papel existente (administrador, todos, )` quando não há entidade de login
(`resolver.go:1027`, `app.LoginEntity` vazio). Classe: implementação errada, menor.

**Estado em `0c12051`** (reconferido com `go build ./cmd/ge` e arquivos temporários no
scratchpad, segunda rodada): D1 continua (`nome obrigatório › email` vira o campo irmão
`email`; a linha abaixo de `todos › ver` some sem erro; G67); D2 continua (`cliente já tem o
campo nome` para o mesmo fato repetido; G68); D4 continua (`tme` dá "não entendi a linha
\"clientes\"", sem sugestão; G69); D12 continua ("administrador, todos, )"; G73); D9 foi
corrigido. D3, D5, D6, D7, D10 e D11 não foram reconferidos nesta rodada; as lacunas
correspondentes (G69, G70, G71, G72, G73) seguem OPEN em `GERMANIO_GAPS.md`.

**Correções de segurança feitas depois da primeira rodada** (não eram divergências D, mas
mudam o que a pesquisa pode afirmar; estado conforme `GERMANIO_GAPS.md` em `0c12051`):

| Lacuna | O que era | Correção e teste | Estado |
|---|---|---|---|
| G65 | `/ws` aceitava conexão sem sessão e difundia o registro inteiro | token e mesma origem; aviso com tipo, modelo e id (`TestTempoRealNaoVazaDados`) | DONE; G66 (salas) aberto |
| G83 | `tenha login` sem nenhuma proteção contra tentativa em massa | bloqueio por padrão, 10/10 min (`TestLoginBloqueiaPorPadrao`) | **IN_PROGRESS**: falta o limite por IP no login de intenção e mostrar os valores no `ge explain` |
| G91 | pastas servidas listavam o conteúdo e entregavam `.env` | só arquivos; diretório e nome com ponto dão 404 (`TestPastaServidaNaoListaNemEntregaOcultos`) | DONE |
| G98 | `?ordem=` cru no `ORDER BY` do dialeto anterior | só coluna existente e ASC/DESC; identificadores escapados; página de até 1000 (`TestListarRecusaOrdemInjetada`) | DONE |
| G99 | SSRF em `/api/_proxy` por filtro de texto | destino verificado sobre o IP resolvido, sem redirects, sem CORS (`TestProxyNaoAlcancaRedeLocal`) | DONE |
| G100 | cadastro aceitava `role: "admin"`, IDOR, leitura anônima, senha em texto | colunas reais, dono ou admin, leitura com login, bcrypt em todo caminho (`TestDialetoAnteriorNaoAbrePortas`, `TestSenhaNuncaGravadaComoTexto`) | DONE |
| G101 | SVG servido inline; upload sem teto | CSP `sandbox`, `nosniff`, SVG como anexo, 64 MB (`TestUploadSVGNaoExecuta`) | DONE |

A pesquisa de segurança ([SECURITY_DEFAULTS.md](SECURITY_DEFAULTS.md), V1–V4) descreve o estado
**anterior** a G98–G101; o texto dela não foi reescrito.

**Verificado e coerente com a norma:** tab numa linha só de espaço em branco é erro
(`linha 6: tab na indentação`, `lexer.go:458-462`); a norma diz "Tab no início de linha é
erro", então o comportamento segue a letra; se uma linha em branco com tab deveria ser
tratada como vazia é uma pergunta de especificação, não um bug.

---

## Checagem anti-Frankenstein

Cada ADOTAR e ADAPTAR acima responde a um problema que existe hoje (uma divergência D, um
risco já registrado como "vários parsers" ou G21, ou uma pendência de `INTENCAO.md`). As
lições convergem em cinco decisões, não em vinte recursos:

1. **Uma árvore** (tokens com span → árvore de linhas sem perdas) para todas as ferramentas.
2. **Um schema de seções** que dá papel a cada nível e proíbe o resto.
3. **Fatos com origem** gerados direto da árvore; a frase plana é renderização.
4. **Diagnósticos como dados**, todos de uma vez, com sugestões verificadas e confiança
   declarada; nenhuma permissão aplicada sozinha.
5. **Um processo leve** (GEP) para o que muda significado, e compatibilidade como mecanismo
   quando houver `.ge` de terceiros.

Nenhuma delas acrescenta um conceito ao `.ge`. Todas acrescentam complexidade só dentro do
compilador e do tooling, que é onde ela deve ficar.

---

## Segunda rodada (ecossistema)

**Data:** 2026-09-28. **Revisão conferida:** `0c12051` (master). **Fontes:** os estudos de
[kotlin.md](kotlin.md), [java.md](java.md), [csharp.md](csharp.md),
[javascript.md](javascript.md), [ruby.md](ruby.md), [dart.md](dart.md), [scala.md](scala.md),
[c.md](c.md), [cpp.md](cpp.md), [v.md](v.md), [crystal.md](crystal.md), [julia.md](julia.md),
[lua.md](lua.md), [haskell.md](haskell.md) e os transversais [CONCURRENCY.md](CONCURRENCY.md),
[PACKAGE_MANAGEMENT.md](PACKAGE_MANAGEMENT.md), [SECURITY_DEFAULTS.md](SECURITY_DEFAULTS.md),
[ECOSYSTEM_TRIAGE.md](ECOSYSTEM_TRIAGE.md) e [DISCOVERIES.md](DISCOVERIES.md), mais
`docs/research/performance/{AUDITORIA,ARQUITETURA}.md` e
`docs/research/frontend/GERMANIO_FRONTEND.md`. A matriz está em
[ECOSYSTEM_MATRIX.md](ECOSYSTEM_MATRIX.md); as respostas às 15 perguntas, em
[RESPOSTAS.md](RESPOSTAS.md). As URLs estão nos estudos citados.

Os ids continuam a numeração da primeira rodada (A21…, P15…, E20…, I12…). Formato de cada
item: **Fonte** · **Problema do Germanio** · **Arquivo afetado** · **Remove da cabeça** ·
**Estado**. Estados: **IMPLEMENTADO** (existe no código, com teste citado), **PROPOSTO** (não
existe), **EXPERIMENTAL** (existe sem força normativa) e **REJEITADO** (EVITAR). Quando só
parte existe, o estado diz qual parte.

A regra desta rodada foi procurar onde o Germanio está errado. Por isso vários itens são
correções do que o runtime já faz, não recursos novos.

### ADOTAR

**A21. Concorrência estruturada como invariante do runtime, invisível ao autor.**
Fonte: [CONCURRENCY.md](CONCURRENCY.md), [kotlin.md](kotlin.md), [java.md](java.md) (JEP 505),
[go.md](go.md) (`errgroup`); convergência de quatro linguagens. Problema: `paralelo` cria uma
goroutine por item sem teto; o pânico vira a string `"erro: …"` e não cancela as irmãs;
`timeout` devolve nulo e deixa a goroutine rodando (G89, G96). Arquivo:
`runtime/interpreter/interpreter.go:1128-1280`. Remove da cabeça: limite, cancelamento, coleta
de erro, limpeza e "isso continua rodando depois?". Estado: **PROPOSTO**
([GEP 0005](../../gep/0005-concorrencia-por-intencao.md)).

**A22. Fila com teto que contém o produtor ou recusa na borda; nunca descarta.**
Fonte: [CONCURRENCY.md](CONCURRENCY.md) (Tokio, .NET Pipelines),
`performance/ARQUITETURA.md` §7.2. Problema: `jobs.Submit` devolve `false` quando a fila de 256
enche (`runtime/jobs/jobs.go:56-67`) e os três chamadores ignoram o retorno
(`runtime/servidor/servidor.go:976, 989, 1076`): a notificação some sem registro. Arquivo:
`runtime/jobs`, `runtime/servidor/servidor.go`. Remove: descobrir em produção que avisos se
perderam. Estado: **PROPOSTO** (GEP 0005).

**A23. Um único cliente HTTP de saída, o seguro, em todo caminho.**
Fonte: [SECURITY_DEFAULTS.md](SECURITY_DEFAULTS.md) (V2; OWASP SSRF). Problema: o proxy
(`servidor.go:1182`, G99), os webhooks (`eventos.go:123`) e as tarefas usam `safeHTTPClient`
(`tarefas.go:194`). **Mas `chamar` do nível 3, `chamar_async`, as chamadas de IA e o cron
usam `httpclient.Novo()`** (`runtime/engine.go:396, 409`; `runtime/cron/cron.go:24`), um
`http.Client` sem verificação do IP resolvido, que segue redirects e lê a resposta inteira
com `io.ReadAll` sem teto (`runtime/httpclient/httpclient.go:17-49`). Lido no código, não
reproduzido em execução; contradiz a descrição de G59 ("httpclient aceitava qualquer
destino" → DONE). Arquivo: `runtime/httpclient`, `runtime/engine.go`, `runtime/cron`. Remove:
saber quais caminhos protegem a rede interna. Estado: **IMPLEMENTADO** no proxy, webhooks e
tarefas; **PROPOSTO** para `chamar` e cron (achado desta rodada, a registrar em
`GERMANIO_GAPS.md` pelo orquestrador).

**A24. Nenhum identificador de SQL vem do cliente.**
Fonte: [SECURITY_DEFAULTS.md](SECURITY_DEFAULTS.md) (V1, CWE-89). Problema: era o `ORDER BY`
cru do dialeto anterior. Arquivo: `runtime/banco/banco.go`, `runtime/banco/consulta.go`.
Remove: pensar em injeção. Estado: **IMPLEMENTADO** para ordem e coluna
(G98, `TestListarRecusaOrdemInjetada`); o construtor único de consultas derivado de `ast.App`,
com a visibilidade como predicado SQL, é **PROPOSTO** (G85, `performance/ARQUITETURA.md` §3.2).

**A25. Lista branca de campos graváveis em todo caminho de escrita.**
Fonte: [ruby.md](ruby.md) (mass assignment, GitHub 2012), [SECURITY_DEFAULTS.md](SECURITY_DEFAULTS.md)
(V3). Problema: o cadastro do dialeto anterior copiava o corpo inteiro; no dialeto de
intenção, quem edita pode alterar qualquer campo declarado e só `admin`/`papel` têm proteção,
por nome (G97). Arquivo: `runtime/servidor/intencao.go:378` (`writable`), `runtime/auth`.
Remove: auditar formulários à procura do campo que ninguém deveria mandar. Estado:
**IMPLEMENTADO** (`writable`; G100, `TestDialetoAnteriorNaoAbrePortas`); a frase "quem altera
um campo" é **PROPOSTO** (G97, exige GEP).

**A26. Nenhum comportamento indefinido na lógica de nível 3.**
Fonte: [c.md](c.md) (UB não se remove depois). Problema: `+` concatena se um lado é texto,
`toNumber` converte texto em silêncio, e não há tabela normativa dessas conversões. Arquivo:
`runtime/interpreter/interpreter.go`, `SPEC.md`. Remove: "o que acontece se…". Estado:
**PROPOSTO**.

**A27. Toda afirmação pública tem um teste ou uma medição citada.**
Fonte: [v.md](v.md) (distância entre promessa e implementação). Problema: "secure by default"
e "20 languages" estavam no texto público sem teste que os sustentasse. Arquivo: `README.md`,
`CLAUDE.md`, `SECURITY.md`. Remove: desconfiar do que o site diz. Estado: **IMPLEMENTADO em
parte**: `bf3cd51` alinhou as afirmações ao que está provado (o README lista "Known limits",
inclusive "The intent layer is in Portuguese only") e cada correção de segurança tem teste
nomeado; falta o teste de determinismo (A31).

**A28. "Pague só pelo que usar" como critério verificável.**
Fonte: [cpp.md](cpp.md) (zero overhead), [lua.md](lua.md) (núcleo embutível),
`performance/AUDITORIA.md` §4. Problema: um app de uma entidade carrega WhatsApp (+9,1 MB),
goldmark, três drivers e Git, e inicializa WebSocket, filas e hot reload; RSS 26,9 MB contra
12,4 MB (G90). Arquivo: `runtime/engine.go`, `runtime/servidor/servidor.go`, `cli/cli.go`.
Remove: nada que o autor pense hoje; remove custo que ele não escolheu. Estado: **PROPOSTO**.

**A29. Recarga só depois de `ge check`; hot reload só em desenvolvimento.**
Fonte: [dart.md](dart.md), [csharp.md](csharp.md). Problema: o hot reload troca o processo
sem validar o `.ge` e roda a migração a cada recarga (G95); fica ligado em produção e no
binário de `germanio build`, varrendo o diretório a cada segundo (G88). Arquivo:
`runtime/hotreload.go`, `runtime/engine.go`. Remove: "salvei com erro e o servidor morreu" e
colunas criadas por nomes digitados errado. Estado: **PROPOSTO**.

**A30. `único` sempre vira índice único no banco.**
Fonte: [ruby.md](ruby.md). Problema: só na criação da tabela (G94); campo único acrescentado
depois fica só com a checagem da aplicação (condição de corrida). Arquivo:
`runtime/banco/banco.go` (criação das tabelas; `GERMANIO_GAPS.md` cita `schema.go`). Remove: uma corrida que o autor nem sabe que existe. Estado:
**PROPOSTO**.

**A31. Teste de determinismo por propriedade.**
Fonte: [v.md](v.md), [haskell.md](haskell.md) (ordem não escrita), [julia.md](julia.md).
Problema: a promessa "determinístico" não tem teste nomeado que permute a ordem dos arquivos
e repita a compilação comparando `ast.App` e `ge explain`. Arquivo: testes de
`compiler/parser` e `tooling/explicar`. Remove: "a ordem dos arquivos importa?". Estado:
**PROPOSTO** (complementa A4).

**A32. Um idioma por arquivo e teste de colisão com palavras comuns do português.**
Fonte: [DISCOVERIES.md](DISCOVERIES.md) (Hedy, D3). Problema: o mapa global
`compiler/idiomas/idiomas.go` transforma "no" em `nao`, "o" em `ou`, "estado" em `status`
(`idiomas.go:30, 39`; `lexer.go:762-769`) (G92); a camada de intenção só existe em português
e um programa em inglês passa no `ge check` com 0 dados (G74, reproduzido em `0c12051`).
Arquivo: `compiler/idiomas`, `compiler/lexer`, `tooling/gecli`. Remove: a lista invisível de
palavras de 19 idiomas que não se podem usar como nome. Estado: **PROPOSTO**
([GEP 0007](../../gep/0007-idiomas-da-intencao.md)); **exige decisão deliberada**, porque
`INTENCAO.md:288-289` ainda diz "em qualquer idioma do léxico".

**A33. Princípios de pacotes antes de qualquer gerenciador: nada executa na instalação,
lockfile com hash, registro imutável, resolução mínima (MVS).**
Fonte: [PACKAGE_MANAGEMENT.md](PACKAGE_MANAGEMENT.md) (npm 2025, Go modules, crates.io,
Deno). Problema: não há gerenciador (o que é bom); o risco é criar um às pressas quando
adaptadores de terceiros aparecerem. Arquivo: nenhum hoje; futuro `cli`. Remove: resolver
conflitos de versão à mão; confiar no registro. Estado: **PROPOSTO** como princípio
([GEP 0006](../../gep/0006-pacotes.md), que conclui que ainda não é hora de um gerenciador).

**A34. A inferência sobre um dado vem do próprio bloco ou de fatos declarados, nunca de como
ele é usado em outro lugar.**
Fonte: [crystal.md](crystal.md) (inferência global: compilação e LSP caros). Problema: o
resolver funde blocos de vários arquivos; sem a regra, uma inferência futura por uso tornaria
o LSP caro e a origem difícil de mostrar. Arquivo: `compiler/parser/resolver.go`,
`docs/INTENCAO.md`. Remove: procurar em todos os arquivos a origem de um fato. Estado:
**IMPLEMENTADO de fato** (o resolver infere pelo nome e pela declaração), **PROPOSTO** como
regra escrita.

**A35. Limites seguros por padrão, com o valor visível.**
Fonte: [SECURITY_DEFAULTS.md](SECURITY_DEFAULTS.md), [lua.md](lua.md),
`performance/ARQUITETURA.md` §7.2. Problema: limites existem em pontos soltos e não aparecem
no `ge explain` como a norma exige ("`ge explain` mostra os limites em vigor"). Arquivo:
`runtime/servidor`, `runtime/engine.go`, `tooling/explicar`. Remove: calcular limites. Estado:
**IMPLEMENTADO em parte**: `ReadHeaderTimeout` de 10 s (`engine.go:466`,
`servidor.go:99`), upload até 64 MB (G101), página de até 1000 (G98), bloqueio de login
(G83); **PROPOSTO**: mostrar no `ge explain`, teto para a resposta de `chamar`, limite por IP
no login de intenção.

### ADAPTAR

**P15. Supervisão declarativa (OTP) para trabalho em segundo plano; uma capability de
"trabalho", local ou remota.**
Fonte: [elixir.md](elixir.md), [CONCURRENCY.md](CONCURRENCY.md). Adaptação: sem mailbox nem
estratégias de supervisão; o autor diz "tente de novo 3 vezes" e o runtime é o supervisor.
Problema: três mecanismos (fila `jobs` em memória, tabela `tarefas` com polling 4×/s sem
índice nem limpeza, trabalho remoto com lease) e só o terceiro tem a disciplina certa (G87).
Arquivo: `runtime/servidor/trabalho_remoto.go`, `tarefas.go`, `runtime/jobs`. Remove: escolher
entre filas. Estado: **IMPLEMENTADO** para trabalho remoto
(`runtime/trabalho_remoto_test.go`); **PROPOSTO** para o local (GEP 0005).

**P16. Snapshot do esquema aplicado e diff explicado, sem arquivos de migração.**
Fonte: [csharp.md](csharp.md) (EF Core), [dart.md](dart.md). Adaptação: sem arquivos de
migração escritos pelo autor; mudança aditiva segue automática; renomear ou remover para com
erro educativo. Problema: renomear um campo cria coluna vazia e os dados somem da aplicação
sem aviso; o valor padrão vai colado no SQL e o erro do `ALTER` é engolido (G93). Arquivo:
`runtime/banco/banco.go`. Remove: SQL, arquivos de migração e medo de perder dados. Estado:
**PROPOSTO** (a frase de renomeação exige GEP).

**P17. Efeitos visíveis sem mônadas: `ge explain` lista, por ação, o que ela dispara.**
Fonte: [haskell.md](haskell.md); reforça P12 (Roc). Adaptação: marcação feita pelo
`ge explain`, nunca pelo autor. Problema: o efeito de uma ação (webhook, e-mail, tarefa) só se
descobre lendo vários blocos. Arquivo: `tooling/explicar`, `compiler/parser/resolver.go`.
Remove: rastrear "o que acontece quando clico". Estado: **PROPOSTO**.

**P18. Acesso externo por camada e lógica em sandbox.**
Fonte: [javascript.md](javascript.md) (permissões do Deno), [lua.md](lua.md) (sandbox do
Luau). Adaptação: sem flags de permissão na linha de comando; a camada decide: rede, arquivo e
processo só em `integracoes/` ou em declarações com destino nomeado. Problema: `chamar` está
disponível a qualquer hook do domínio, e o core conhece `OPENAI_KEY`, `STRIPE_KEY`
(`interpreter.go:1303, 1441`), contra a regra das três camadas. Arquivo:
`runtime/interpreter/interpreter.go`, `runtime/httpclient`. Remove: auditar hooks à procura
de chamadas externas. Estado: **PROPOSTO**.

**P19. Orçamento de execução por request ligado ao `context`.**
Fonte: [lua.md](lua.md) (interrupção do Luau pelo host). Adaptação: o prazo é do request ou
da tarefa, não um limite fixo por laço. Problema: o limite de 10 000 iterações por laço recusa
casos legítimos e não limita o aninhamento; nada cancela (AUDITORIA §3.4). Arquivo:
`runtime/interpreter`, `runtime/servidor`. Remove: temer travar o servidor. Estado:
**PROPOSTO** (GEP 0005).

**P20. O formulário espelha as regras que o servidor verifica.**
Fonte: [java.md](java.md) (Bean Validation). Adaptação: a mesma fonte (`ast.Field`) gera
`minlength`, `maxlength`, `pattern`; o servidor continua a autoridade. Problema:
`min`, `max` e `formato` não viram atributos (`runtime/servidor/paginas.go`). Remove:
descobrir o erro só depois de enviar. Estado: **PROPOSTO** (junto de
[GEP 0003](../../gep/0003-contrato-de-formulario.md)).

**P21. Artefato de execução pré-resolvido (planos imutáveis a partir de `ast.App`).**
Fonte: [julia.md](julia.md) (package images), `performance/ARQUITETURA.md` §3.2 (IR
executada). Adaptação: sem gerar Go; os planos (colunas projetadas, statements preparados,
visibilidade em SQL, páginas sem JSON interno) são a mesma semântica pré-avaliada. Problema:
reparse a cada partida, `germanio build` embute os fontes e os extrai para um diretório
temporário, visibilidade O(tabela) (G85). Arquivo: `runtime/`, `cli/cli.go`. Remove: nada
visível; remove latência e memória. Estado: **PROPOSTO**.

**P22. Modelo de custo explícito, sem expor memória.**
Fonte: [c.md](c.md), `performance/ARQUITETURA.md` §8 (`ge profile`). Adaptação: o
`ge explain` diz quando uma regra obriga a ler a tabela inteira e o futuro `ge profile`
atribui tempo a linhas do `.ge`, nunca a funções Go. Problema: a visibilidade O(tabela) é
invisível ao autor (414 ms e 88 MB por página com 50 000 linhas). Arquivo: `tooling/explicar`.
Remove: descobrir lentidão em produção. Estado: **PROPOSTO**.

**P23. Toda convenção aplicada aparece no `ge explain` com o motivo.**
Fonte: [ruby.md](ruby.md) (convenção sem explicação), [java.md](java.md) (autoconfiguração
só no log). Adaptação: o Germanio já tem convenções (tabela, tipo pelo nome); a diferença é
explicá-las. Problema: a separação declarado/inferido/interno e o motivo de cada inferência
ainda não estão completos (`INTENCAO.md` › Inspeção). Arquivo: `tooling/explicar`. Remove:
"de onde veio isto?". Estado: **IMPLEMENTADO em parte** (origem de cada fato);
**PROPOSTO** (motivo de cada inferência).

**P24. Reescrita entre forma plana e hierárquica, nos dois sentidos, com equivalência
provada.**
Fonte: [scala.md](scala.md) (duas sintaxes no Scala 3). Adaptação: um comando explícito, não o
`ge fmt` padrão. Problema: duas formas válidas sem conversão. Arquivo: `tooling/formatter`.
Remove: "qual forma uso e como passo de uma para a outra?". Estado: **PROPOSTO**.

**P25. Construção nova nasce em *preview* ou como aviso com prazo.**
Fonte: [java.md](java.md) (JEP 12), [kotlin.md](kotlin.md) (KEEP), [csharp.md](csharp.md)
(NRT como avisos). Adaptação: sem os quatro modos do C#; o status "Em teste" da GEP já é o
preview; uma checagem nova do `ge check` nasce aviso e vira erro na versão seguinte.
Problema: toda nova checagem quebraria projetos de uma vez. Arquivo: `docs/gep/README.md`,
`compiler/diagnostics`. Remove: "a atualização vai quebrar meu projeto?". Estado:
**IMPLEMENTADO em parte** (status "Em teste" no processo de GEP); **PROPOSTO** (aviso com
prazo).

**P26. Remoção só com reescrita automática verificada.**
Fonte: [javascript.md](javascript.md) (não poder remover), [lua.md](lua.md) (remover sem
migração); reforça P7. Adaptação: a reescrita prova por reparse que `ast.App` não mudou.
Arquivo: `tooling/formatter`. Remove: migrar à mão. Estado: **PROPOSTO**.

**P27. Ausência explícita na fronteira externa.**
Fonte: [kotlin.md](kotlin.md) (*platform types*), [crystal.md](crystal.md) (`Nil` em uniões).
Adaptação: sem símbolos de nulidade no `.ge`; o adaptador declara a forma do que recebe e o
campo ausente vira erro educativo na borda. Problema: campo ausente vira `nil` num
`map[string]any` e segue em silêncio na lógica. Arquivo: `runtime/interpreter` (`chamar`),
`integracoes/`. Remove: checar nulo em cada uso. Estado: **PROPOSTO**.

**P28. Mudança de sintaxe ou de estilo decidida com o diff sobre o corpus.**
Fonte: [dart.md](dart.md) (estilo *tall*), [scala.md](scala.md). Adaptação: o corpus são
`demo/` e `examples/` (o GitLab tem 687 linhas). Problema: decidir sintaxe por gosto. Arquivo:
`docs/gep/0000-template.md` (seção Evaluation). Remove: nada do autor; protege o autor de
mudanças não medidas. Estado: **PROPOSTO**.

### EVITAR

Todos com estado **REJEITADO** para o nível padrão.

| # | O que evitar | Fonte | Por que (problema que criaria no Germanio) |
|---|---|---|---|
| E20 | coloração de função (`async`, `suspend`), atores, `Sendable`, goroutine, canal ou mutex no `.ge` | [CONCURRENCY.md](CONCURRENCY.md), [rust.md](rust.md), [swift.md](swift.md) | conceitos técnicos que `INTENCAO.md` › Eficiência proíbe no nível padrão |
| E21 | registro com scripts de instalação, mutável, micro-dependências, tokens amplos | [PACKAGE_MANAGEMENT.md](PACKAGE_MANAGEMENT.md) (npm 2025, left-pad) | é o vetor dos ataques estudados |
| E22 | resolução de versão "nearest wins" | [PACKAGE_MANAGEMENT.md](PACKAGE_MANAGEMENT.md) (Maven) | o resultado depende da forma da árvore; não determinístico |
| E23 | tradução palavra a palavra para línguas de outra ordem de frase | [DISCOVERIES.md](DISCOVERIES.md) D3 (Hedy), D4 (Kip) | `developer pode enviar código dos projetos` não se traduz por substituição |
| E24 | editor ou IDE próprios | [DISCOVERIES.md](DISCOVERIES.md) D5, D7 (Eve, Darklang) | foi a parte mais cara e menos amada; o caminho é LSP (A17) |
| E25 | gerar código de terceiros para o autor manter (eject, scaffold, arquivos de migração) | [DISCOVERIES.md](DISCOVERIES.md) D2, [ruby.md](ruby.md) | traria de volta tudo o que o Germanio remove |
| E26 | garantias que só se leem entendendo um sistema de tipos | [DISCOVERIES.md](DISCOVERIES.md) D6 (Ur/Web) | as garantias vêm de o runtime ser o único autor |
| E27 | modos, flags ou extensões por arquivo que mudam o significado | [cpp.md](cpp.md), [csharp.md](csharp.md), [haskell.md](haskell.md), [v.md](v.md) | "este programa roda com qual modo?"; reforça E17 |
| E28 | dois gerenciadores de pacote oficiais | [haskell.md](haskell.md) (Cabal e Stack) | escolher ferramenta |
| E29 | "mecanismos, não políticas" no domínio | [lua.md](lua.md) | cada app montaria sua autorização |
| E30 | sinônimos como forma de compatibilidade | [javascript.md](javascript.md) (SmooshGate) | custo permanente; preferir reescrita (P26) |
| E31 | amplitude antes de profundidade | [v.md](v.md) | WhatsApp embutido antes de estabilizar o núcleo (G90) |
| E32 | fila que descarta e prazo que abandona | [CONCURRENCY.md](CONCURRENCY.md) | é o comportamento atual de `jobs` e `timeout` (A21, A22) |
| E33 | semântica que depende de ordem não escrita (preguiça, ordem de arquivos) | [haskell.md](haskell.md) | `importar "backend"` em ordem alfabética não pode mudar o significado (A31) |
| E34 | "sem IA" como diferencial único | [DISCOVERIES.md](DISCOVERIES.md) D2 | três pares de 2026 dizem o mesmo; o diferencial precisa de aplicações reais e medidas |

### INVESTIGAR

Estado de todos: **PROPOSTO (a investigar)**, salvo indicação.

| # | O quê | Fonte | Condição para decidir |
|---|---|---|---|
| I12 | ambiente sem instalação: `ge check` e `ge explain` em WebAssembly no navegador | [DISCOVERIES.md](DISCOVERIES.md) D4 (Portugol Webstudio, Hedy, IDEgua) | medir o tamanho do WASM do front-end (Go puro, sem CGO) |
| I13 | tipo decimal interno para `dinheiro` | [c.md](c.md) | auditar onde a lógica usa `float64` para valores monetários |
| I14 | ordem dos resultados paralelos e o que paralelizar sob o escritor único | [CONCURRENCY.md](CONCURRENCY.md) | depende de G86 e da suíte STRESS |
| I15 | quantos idiomas prometer | [DISCOVERIES.md](DISCOVERIES.md) I2 | usuários reais de algum idioma além do português |
| I16 | blocos `.ge` compartilháveis são necessários? | [PACKAGE_MANAGEMENT.md](PACKAGE_MANAGEMENT.md) | um caso real que `importar` local e capabilities do core não cubram |
| I17 | onde o valor traduzido (`Value`) é usado no lugar da grafia (`Raw`) | [DISCOVERIES.md](DISCOVERIES.md) I1 | teste com palavras colidentes em `logica`, `telas`, `eventos` |
| I18 | traces das últimas requisições em desenvolvimento | [DISCOVERIES.md](DISCOVERIES.md) D7 (Darklang) | segurança: só em `ge run`, sem segredos |
| I19 | tempo real por diff do servidor (Blazor Server, LiveView) contra aviso + rebusca | [csharp.md](csharp.md), [elixir.md](elixir.md), `frontend/GERMANIO_FRONTEND.md` | custo por espectador medido; G66 resolvido antes |
| I20 | padrão de opcionalidade dos campos | [kotlin.md](kotlin.md) | teste do leigo; no mínimo `ge explain` mostrar "opcional (padrão)" |
| I21 | bloco colado no nível errado; blocos longos | [scala.md](scala.md) | teste de propriedade sobre `examples/` |
| I22 | conjunto curado e auditado de adaptadores (Stackage, cargo-vet) | [haskell.md](haskell.md), [PACKAGE_MANAGEMENT.md](PACKAGE_MANAGEMENT.md) | primeiro adaptador de terceiros |
| I23 | arena ou `sync.Pool` por request | [cpp.md](cpp.md) | medir depois de P21 |

### Checagem anti-Frankenstein (segunda rodada)

As oito perguntas para cada ADOTAR e ADAPTAR novo. "Problema" é a lacuna ou o arquivo que o
item resolve; "Conceito novo no `.ge`?" é a pergunta de custo cognitivo.

| Item | Problema do Germanio | Generaliza? | Reduz complexidade? | Determinístico? | Ensinável/explicável? | Eficiente? | Combina? | Conceito novo no `.ge`? |
|---|---|---|---|---|---|---|---|---|
| A21, A22, P19 | G89, G96, `jobs` | sim | sim | sim (ordem da entrada) | sim, pelo `ge explain` | sim | sim | só com a GEP 0005, e em palavras de intenção |
| A23, A24, A25, A35 | G59 parcial, G97–G101 | sim | sim | sim | sim | sim | sim | não (A25 por campo exige GEP) |
| A26 | conversões silenciosas | sim | sim | sim | sim | neutro | sim | não |
| A27, A31 | promessas sem teste | sim | sim | é o próprio teste | sim | neutro | sim | não |
| A28, P21 | G90, G85 | sim | sim | sim | sim | é o objetivo | sim | não |
| A29, A30, P16 | G88, G93–G95 | sim | sim | sim | sim | sim | sim | uma frase de renomeação (GEP) |
| A32 | G74, G92 | sim | sim | sim | sim | sim | **contradiz a promessa de 20 idiomas** | uma declaração de idioma (GEP 0007) |
| A33 | nenhum ainda | sim | sim | sim | sim | sim | sim | não, enquanto não houver pacotes |
| A34 | inferência futura | sim | sim | sim | sim | sim | sim | não |
| P15 | três filas | sim | sim | sim | sim | sim | sim | "tente de novo N vezes" (GEP 0005) |
| P17, P22, P23 | explicação incompleta | sim | sim | sim | é a explicação | sim | sim | não |
| P18, P27 | `chamar` em qualquer hook; `nil` silencioso | sim | sim | sim | sim | sim | sim (três camadas) | não |
| P20, P24, P25, P26, P28 | forma, evolução | sim | sim | sim | sim | sim | sim | não |

As lições desta rodada convergem em cinco decisões, que se somam às cinco da primeira:

6. **O runtime é dono de todo trabalho, e todo trabalho tem limite**: concorrência
   estruturada, filas que não descartam, prazos que cancelam, limites visíveis.
7. **A mesma defesa em todo caminho**: um cliente HTTP de saída, um construtor de consultas,
   uma lista branca de campos. Onde há dois caminhos, um deles fica sem defesa (foi o
   dialeto anterior, G98–G101; ainda é o `chamar`, A23).
8. **Pague só pelo que usar** como teste, não como intenção (G90).
9. **Uma língua por arquivo**, ou nenhuma promessa de outras línguas (GEP 0007).
10. **Nada executa na instalação** quando houver pacotes; até lá, nenhum gerenciador
    (GEP 0006).

Nenhuma acrescenta conceito técnico ao `.ge`. Três pedem uma frase nova de intenção, e cada
uma passa por GEP: o trabalho concorrente (GEP 0005), a declaração de idioma (GEP 0007) e a
renomeação de campo (P16).
