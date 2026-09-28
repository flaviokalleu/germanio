# Diagnósticos: estudo profundo e proposta para o Germanio

**Data:** 2026-09-28. **Status:** pesquisa, sem força normativa. A norma é `docs/INTENCAO.md`
› Erros (linhas 367-380) e › Erros educativos. Fontes: [rust.md](rust.md) › Diagnostics em
profundidade, [gleam.md](gleam.md) › Diagnostics em profundidade, [swift.md](swift.md),
[typescript.md](typescript.md), [python.md](python.md), [elixir.md](elixir.md),
[zig.md](zig.md) e [../sintaxe-hierarquica.md](../sintaxe-hierarquica.md) (Elm).

---

## 1. Por que diagnósticos são o centro, não um acabamento

Para o público do Germanio, a mensagem de erro é o caminho da falha. A revisão de Becker et
al. reúne evidência de que mensagens de erro são uma barreira central para iniciantes
([ITiCSE WG 2019](https://dl.acm.org/doi/10.1145/3344429.3372508), citado em
`sintaxe-hierarquica.md`). O Elm tornou isso princípio: o erro mostra o trecho, explica e dá
uma dica ([Compiler Errors for Humans](https://elm-lang.org/news/compiler-errors-for-humans)).
O Zig tem um formulário de issue só para "mensagem de erro inútil"
(`.forgejo/ISSUE_TEMPLATE/error_message.yml`, [zig.md](zig.md)): **mensagem ruim é bug**.

O Germanio já tem o formato de quatro partes (o quê, onde, por quê, como corrigir) e a
exigência de "Equivale a:" dentro de blocos. O que os estudos acrescentam é tratar o
diagnóstico como **dado estruturado** e a correção como **edição com confiança declarada e
verificada**.

## 2. O que cada referência ensina

| Referência | Mecanismo | Fonte |
|---|---|---|
| Rust | diagnostic como struct: nível, código, mensagem que "stands on its own", span primário, spans secundários com label, subdiagnostics (`note`, `help`, sugestão); definido declarativamente (`#[derive(Diagnostic)]`, `#[primary_span]`, `#[label]`, `#[suggestion(applicability = …)]`); saída JSON com `byte_start/byte_end`, `is_primary`, `suggested_replacement`, `suggestion_applicability`, `children` | [diagnostics](https://rustc-dev-guide.rust-lang.org/diagnostics.html), [structs](https://rustc-dev-guide.rust-lang.org/diagnostics/diagnostic-structs.html), [JSON](https://doc.rust-lang.org/rustc/json.html) |
| Rust | `Applicability`: `MachineApplicable` ("definitely what the user intended, or maintains the exact meaning of the code"), `MaybeIncorrect`, `HasPlaceholders`, `Unspecified`; `rustfix` filtra `MachineApplicableOnly` × `Everything` | [Applicability](https://doc.rust-lang.org/nightly/nightly-rustc/rustc_errors/enum.Applicability.html), [rustfix](https://github.com/rust-lang/cargo/blob/master/crates/rustfix/src/lib.rs) |
| Rust | `ErrorGuaranteed`: prova no tipo que um erro já saiu; sem diagnósticos derivados | [ErrorGuaranteed](https://rustc-dev-guide.rust-lang.org/diagnostics/error-guaranteed.html) |
| Rust | nome parecido limitado a um terço do tamanho, sem caixa; procurar em outros namespaces | [edit_distance](https://doc.rust-lang.org/nightly/nightly-rustc/rustc_span/edit_distance/fn.find_best_match_for_name.html), [name resolution](https://rustc-dev-guide.rust-lang.org/name-resolution.html) |
| Rust | UI tests: `.stderr` com a saída inteira e `.fixed` compilado depois de aplicar as sugestões (`run-rustfix`), com `--bless` | [UI tests](https://rustc-dev-guide.rust-lang.org/tests/ui.html) |
| Gleam | erro é variante de enum com **os fatos e as hipóteses** (`discarded_location`, `type_with_name_in_scope`, `possible_modules`); um único `to_diagnostics()` escolhe o texto pela hipótese presente | [error.rs](https://github.com/gleam-lang/gleam/blob/main/compiler-core/src/error.rs) |
| Gleam | nomes na forma que o usuário escreveria naquele ponto | [v1.6](https://gleam.run/news/context-aware-compilation/) |
| Gleam | tolerância a falhas por definição | [v1.2](https://gleam.run/news/fault-tolerant-gleam/) |
| Swift | fix-it só quando "the single, obvious, and very likely correct way to fix the issue"; mensagem como regra, curta, sem palavras que não informam | `docs/Diagnostics.md` em [swift](https://github.com/swiftlang/swift/blob/main/docs/Diagnostics.md) |
| Swift | *diagnostic groups* (SE-0443) com página educativa curta por grupo, "teachable moment" | [SE-0443](https://github.com/swiftlang/swift-evolution/blob/main/proposals/0443-warning-control-flags.md), [userdocs/diagnostics](https://github.com/swiftlang/swift/tree/main/userdocs/diagnostics) |
| TypeScript | catálogo único `diagnosticMessages.json` (texto com `{0}` → categoria e código estável, 2213 entradas), gerador; `relatedInformation`; code fixes indexados pelo código, "corrigir todos" | [diagnostics](https://github.com/microsoft/TypeScript/tree/main/tsc/internal/diagnostics), [diagnostic.go](https://github.com/microsoft/TypeScript/blob/main/tsc/internal/ast/diagnostic.go), [Compiler Notes](https://github.com/microsoft/TypeScript-Compiler-Notes) |
| Python 3.10+ | o avanço veio **da gramática**: bracket não fechado aponta onde abriu; `expected an indented block after 'if' statement on line 2` nomeia a construção dona e a linha dela (regra `invalid_if_stmt` numa segunda passada); nenhum nome de token interno na mensagem | [What's New 3.10](https://docs.python.org/3/whatsnew/3.10.html), [parser.md](https://github.com/python/cpython/blob/main/InternalDocs/parser.md) |
| Python 3.11 | PEP 657: colunas início/fim para sublinhar a subexpressão, ~22% nos `.pyc` | [PEP 657](https://peps.python.org/pep-0657/) |
| Elixir | vários erros por arquivo (v1.15), trecho com cursor e delimitador que abriu (v1.16); o type checker só avisa bug verificado | [v1.15](https://elixir-lang.org/blog/2023/06/19/elixir-v1-15-0-released/), [v1.16](https://elixir-lang.org/blog/2023/12/22/elixir-v1-16-0-released/) |
| Elm | erro como ensino: trecho, explicação, dica | [Elm](https://elm-lang.org/news/compiler-errors-for-humans) |
| HCL | `Summary`, `Detail`, `Subject` ("a tight range"), `Context` (faixa mais larga), `Extra` para máquina | [hcl/v2](https://pkg.go.dev/github.com/hashicorp/hcl/v2) |

Não copiar: o estilo de mensagem para programador do rustc (minúscula, "expected X, found
Y"), a infraestrutura de tradução separada do código que o próprio Rust está removendo
([MCP #959](https://github.com/rust-lang/compiler-team/issues/959)), o `hint` em texto livre
do Gleam, um arquivo central de erros com milhares de linhas, correções refeitas no LSP em
vez de vir do diagnóstico.

## 3. O Germanio hoje (conferido no código)

| Aspecto | Estado | Evidência |
|---|---|---|
| Estrutura | núcleo estrito: `Diagnostic{Code, Position, Source, Message, Reason, Fix, Example}`; dialeto de aplicação: `teach` concatena texto e `errorf` devolve `fmt.Errorf`, sem código | `compiler/diagnostics/diagnostics.go:16-20`, `compiler/parser/hierarquia.go:93-105`, `compiler/parser/parser.go:32-38` |
| Resolver | `errAt` devolve `fmt.Errorf("%s:%d: …")`, **descarta `Position.Context`** e a coluna | `compiler/parser/resolver.go:88-90` |
| Onde | uma posição (linha, coluna) + caminho; sem fim; o caret aponta um caractere | `diagnostics.go:9-15, 27` |
| Como corrigir | prosa; nenhuma edição aplicável | `teach` |
| Confiança | não existe | — |
| Vários erros | não; o primeiro interrompe | reproduzido: `enviar códigu` + `excluirr` dão um só erro |
| Nome parecido | distância fixa `< 3`, só contra as seções (`suggest`); `unknownLine` só aceita distância exatamente 1 contra uma lista curta | `hierarquia.go:439-450`, `parser.go:182-188` |
| Idioma | sugestões e lista de seções sempre em português (`displaySections`), embora `Token.Raw` guarde a grafia original | `hierarquia.go:434-436`, `lexer.go:220-223` |
| Explicação longa | `Explanations`: uma linha por código, só núcleo estrito, sem exemplo testado; GE1002 diz "dois espaços por bloco" | `diagnostics.go:36-56` |
| Máquina | sem JSON | `tooling/gecli/cli.go` |
| Testes | `strings.Contains` em partes da mensagem | `compiler/parser/hierarquia_test.go` |

## 4. Proposta de estrutura

Estender, não substituir, o formato de quatro partes:

```text
Diagnostic
  Code        string            // GE…, estável
  Severity    erro | aviso
  Message     string            // o quê, sozinho faz sentido
  Primary     Span              // onde: arquivo, início, fim
  Path        string            // caminho hierárquico (projetos › acesso › developer)
  Labels      []Label{Span, Text}          // secundários: "o bloco abre aqui", "primeira declaração"
  Related     []Related{Span, Message}     // outra origem (conflito de fusão), como relatedInformation
  Reason      string            // por quê
  Suggestions []Suggestion{Message, Edits []Edit{Span, NewText}, Applicability, Equivalent []string}
  Hypotheses  (campos da variante, não texto) // ex.: PapelDeclaradoEm *Origem; FilhosSaoAcoes bool
```

- **Erro como variante com fatos e hipóteses** (Gleam): quem detecta preenche os campos; o
  renderizador escolhe o texto. Mantém as variantes perto de quem as emite; só a renderização
  é central.
- **Renderizadores**: texto para a CLI (igual ao de hoje: o quê / onde / por quê / como
  corrigir / Equivale a), JSON para CI e editor, LSP depois. Snapshot por renderizador
  ([roc.md](roc.md)).
- **`Equivalent`** são as frases planas que a correção produziria; vira a linha "Equivale a:"
  que `INTENCAO.md:369-380` exige.
- **Palavras na língua do usuário**: seções e sugestões renderizadas a partir de `Token.Raw`
  e do idioma detectado na linha (Gleam v1.6).
- **Nome parecido**: o algoritmo do rustc/Gleam (igualdade sem caixa primeiro; distância até
  um terço do tamanho, mínimo 1), contra todos os namespaces relevantes (seções, papéis,
  ações, dados), não só as seções.
- **Sem cascata**: um fato que falta por causa de um erro de sintaxe no mesmo bloco não gera
  erro no resolver (a ideia do `ErrorGuaranteed`, com um marcador no nó ERROR).

## 5. Applicability com a regra própria do Germanio

O `MachineApplicable` do rustc junta duas condições: (a) certeza de intenção **ou** (b)
preservação exata de significado. Só (b) é verificável mecanicamente, e é exatamente a prova
que o `ge fmt` já faz (reparsear e comparar). Regra proposta:

| Nível | Quando | Exemplos | Quem aplica |
|---|---|---|---|
| **automática** | a edição deixa os **fatos idênticos**, verificado por reparse e comparação da forma normal | tab → espaços; recuo inconsistente com um único pai possível; sinônimo antigo → forma canônica | `ge fix` por padrão; ação rápida sem confirmação |
| **provável** | a edição cria, remove ou muda qualquer fato | seção com erro de digitação (`acesos` → `acesso`); inserir `acesso` acima de um papel; **toda edição que cria grant, papel, regra de acesso ou integração**, mesmo com distância 1 | só com confirmação explícita (o `Filter::Everything` do rustfix) |
| **com lacunas** | a hipótese precisa de um valor que o Germanio não sabe | "a quem pertence isto?" | nunca aplicada |

Duas regras adicionais: sem `Unspecified` (toda sugestão declara a confiança); com mais de uma
hipótese sobrevivendo à verificação, mostrar todas como prováveis e não escolher. A
justificativa é de segurança, não de estilo: aplicar sozinho `acesos` → `acesso` significa
**conceder permissões por inferência**. Isso é coerente com o princípio de "padrão seguro" do
checklist de `INTENCAO.md` e com o critério do Swift ("single, obvious, and very likely
correct"), mas mais estrito que ambos.

Relação com a norma: a norma não fala de correções automáticas; a regra é nova e **exige
decisão deliberada** (GEP) antes de existir `ge fix`.

## 6. O caso `projetos / developer / excluir`

```text
projetos
    developer
        excluir
```

**Hoje** (`hierarquia.go:285-289`): `dataSection` cai no `case ""`, responde `"developer" não
é uma seção de projetos`, lista as seções e chama `suggest("developer")`, que não acha nada a
menos de 3 edições. Correto, mas sem a hipótese certa. (Se `developer` fosse a **primeira**
linha do bloco, nem isso: `isDataBlock` não reconheceria o bloco e o erro seria "não entendi a
linha projetos"; ver D4.)

**Mecanismo proposto**, passo a passo:

1. **Variante**: `LinhaNaoESecao{Linha, Dado, PapelDeclaradoEm *Origem, FilhosSaoAcoes bool,
   SecaoParecida string}`.
2. **Spans**: primário = a palavra `developer` (linha 2, colunas 5-13); secundário = a linha
   `projetos` com label "dentro deste dado, o primeiro nível diz de que aspecto se trata".
3. **Evidências, em ordem**, parando na primeira que dá hipótese única:
   - (a) distância contra as seções com limite relativo: resolve `acesos`, `tme`;
   - (b) **outro namespace**: `developer` é um papel. O papel pode estar em outro arquivo
     ("a ordem dos arquivos não muda o resultado"), então a evidência é do **resolver**: o
     parser registra a linha como seção desconhecida pendente e o resolver, com os papéis
     completos, emite o diagnóstico (a *late resolution* do rustc);
   - (c) **forma da subárvore**: todos os filhos (`excluir`) são verbos do vocabulário de
     ações. É a forma de um ator dentro de `acesso`; evidência local e determinística.
   A hipótese "faltou `acesso`" só é levantada se (b) ou (c) valer; com as duas, a mensagem
   afirma a origem do papel.
4. **Edição mínima**: inserir `    acesso` antes de `developer` e somar 4 espaços a
   `developer` e à sua subárvore. Não é preciso mover para um bloco `acesso` existente: a
   fusão soma os fatos.
5. **Verificação especulativa** (o `run-rustfix` em tempo de execução): aplicar em memória,
   reparsear, exigir que o diagnóstico original suma, que nenhum novo apareça na subárvore e
   que os fatos da subárvore sejam exatamente `Equivalent` (`developer pode excluir
   projetos`). Se falhar, a sugestão é descartada: uma sugestão que não compila nunca é
   mostrada.
6. **Applicability**: **provável**. A edição cria um grant (a permissão de excluir), então não
   é automática em hipótese nenhuma.
7. **Renderização para o leigo**:

```text
backend/projetos.ge:2 — Germanio não sabe o que "developer" representa dentro de projetos.
Onde: projetos
Por quê: dentro de um dado, cada linha do primeiro nível diz de que aspecto se trata
(tem, pode, acesso, regras...). "developer" é um papel (backend/papeis.ge:3), e abaixo dele
há uma ação (excluir).
Como corrigir: talvez você quisesse declarar acesso:
    projetos
        acesso
            developer
                excluir
Equivale a: developer pode excluir projetos
```

   O "talvez" é a confiança *provável* dita em português; uma sugestão automática seria
   afirmativa ("use 4 espaços; `ge fix` corrige").

## 7. Catálogo de códigos

Hoje os códigos existem só no núcleo estrito (`GE1001`, `GE1002`, `GE2001`-`GE2010`,
`GE3001`-`GE3002`, `GE4102`, `GE5001`-`GE5002`, `GE6001`, `GE9001`, `diagnostics.go:36-56`),
e o mesmo `GE1002` carrega a convenção de dois espaços. Proposta (ilustrativa; os números
exigem decisão):

- **Um catálogo único como dado** (TypeScript): código, categoria, mensagem com parâmetros,
  por quê, como corrigir, grupo educativo, exemplo mínimo que dispara o erro e exemplo
  corrigido. Um gerador ou um teste garante que todo código emitido está no catálogo e que
  todo exemplo do catálogo produz exatamente aquele código (doctest, [elixir.md](elixir.md)).
- **Grupos** (Swift SE-0443) com uma página curta cada, pensados para o leigo: layout,
  contexto e seção, nomes, fusão e conflito, acesso e segurança, capability.
- **Faixas separadas** para a camada de intenção, sem reutilizar números do núcleo estrito
  com outra convenção. Códigos nunca são reutilizados; um código aposentado fica no catálogo
  como "removido".
- **Código só quando a explicação agrega** algo além da mensagem (critério do rustc,
  [error codes](https://rustc-dev-guide.rust-lang.org/diagnostics/error-codes.html)).
- `ge explicar GE…` passa a mostrar a página do grupo com os exemplos testados.

## 8. Testes: snapshot e run-rustfix

Um diretório de casos (por exemplo `compiler/parser/testdata/erros/NNN-nome.ge`), cada um com:

- `.saida`: a renderização de texto **inteira**, comparada literalmente;
- `.json`: a forma estruturada (código, spans, confiança);
- `.corrigido`: o arquivo depois de aplicar as sugestões **automáticas**; o teste exige que
  ele compile e que seus fatos sejam iguais aos do original (automática = fatos idênticos);
- para sugestões prováveis, `.corrigido-provavel` e os fatos esperados (`Equivalent`).

Uma flag de regravação (o `--bless` do rustc; `-atualizar`) evita manutenção à mão. O mesmo
diretório serve de suíte de conformidade para a gramática do VS Code e para o LSP
([TOOLING.md](TOOLING.md)). Os casos iniciais saem das divergências reproduzidas em
[GERMANIO_LESSONS.md](GERMANIO_LESSONS.md) (D1-D4) e do caso da seção 6.
