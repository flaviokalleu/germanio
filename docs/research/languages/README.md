# Estudo dirigido de linguagens para o Germanio

**Data:** 2026-09-28. **Status:** pesquisa consolidada, **sem força normativa**. Pela regra de
[docs/README.md](../../README.md), `research/` reúne "ideias e pesquisa, sem força normativa":
a norma continua em [docs/INTENCAO.md](../../INTENCAO.md) (camada de intenção), em
[SPEC.md](../../../SPEC.md) (núcleo estrito) e em
[skills/germanio-simplicity/SKILL.md](../../../skills/germanio-simplicity/SKILL.md). Nada aqui
altera a norma; toda conclusão que contradiz a norma está marcada "exige decisão deliberada".

## Objetivo

Aprender com décadas de engenharia de linguagens para que o Germanio precise de **menos**
conceitos, não de mais. A meta é a assimetria:

- **complexidade interna alta**: o compilador, o resolver, o formatter, o `ge check`, o
  `ge explain` e o runtime podem ser sofisticados (tabelas fechadas, spans, fatos com origem,
  diagnósticos estruturados, verificação por reparse);
- **complexidade exposta baixa**: quem escreve `.ge` (no nível padrão, quem nunca programou)
  não aprende nenhum desses mecanismos. A pergunta que guia cada lição é "o que o Germanio
  pode deixar de exigir do programador?", com a condição de que a inferência seja
  determinística e explicável por `ge explain`.

Não é um ranking de linguagens nem uma lista de recursos a copiar. Para cada ideia a
pergunta é: "isso resolve um problema fundamental do Germanio?". Se não resolve, ela está em
"EVITAR" ou "não copiar", mesmo que funcione bem na linguagem de origem.

## Índice

### Consolidação (leia nesta ordem)

| Arquivo | Conteúdo |
|---|---|
| [GERMANIO_LESSONS.md](GERMANIO_LESSONS.md) | **o documento principal**: lições em ADOTAR / ADAPTAR / EVITAR / INVESTIGAR, o princípio de subtração e as divergências entre norma e implementação conferidas no código |
| [SYNTAX_COMPARISON.md](SYNTAX_COMPARISON.md) | off-side rule, layout, whitespace, estruturas aninhadas, contexto, ambiguidade, recuperação de erros, formatter; como `projetos / acesso / developer / enviar código` vira fatos; quais camadas fazem sentido |
| [COMPILER_ARCHITECTURE.md](COMPILER_ARCHITECTURE.md) | arquiteturas comparadas e a recomendada (quatro representações), com o caminho incremental a partir do código atual |
| [DIAGNOSTICS.md](DIAGNOSTICS.md) | diagnóstico estruturado, spans, labels, sugestões com confiança declarada, a regra própria do Germanio para correções automáticas, o caso `projetos / developer / excluir`, catálogo de códigos, snapshots |
| [FORMATTERS.md](FORMATTERS.md) | gofmt, rustfmt, zig fmt, mix format, gleam format, nimpretty/nph, swift-format, Black; forma canônica, idempotência, impressão a partir da árvore; situação do `ge fmt` |
| [TOOLING.md](TOOLING.md) | um front-end para `ge fmt/check/explain/graph`, `ge lsp` no mesmo binário, gramática do VS Code gerada das tabelas, analisadores, binário único |
| [LANGUAGE_EVOLUTION.md](LANGUAGE_EVOLUTION.md) | PEP, RFC e editions, Swift Evolution, Go proposals e Go 1/GODEBUG, deprecações do Elixir, poda do Gleam; proposta do GEP |

### Estudos por linguagem (fontes primárias da consolidação)

Cada um segue a matriz do estudo (objetivo, filosofia, sintaxe, gramática, lexer, parser,
AST, IR, análise semântica, tipos, runtime, formatter, LSP, diagnostics, evolução,
compatibilidade, acertos, problemas, o que aprender e o que não copiar) e termina com
"Para o Germanio".

| Arquivo | Foco principal para o Germanio |
|---|---|
| [go.md](go.md) | FileSet e posições compactas; `go/analysis`; printer sobre a AST; Go 1 + GODEBUG |
| [zig.md](zig.md) | binário único; `zig fmt --check`; `ErrorBundle`; `ast-check` por arquivo; mensagem ruim é bug |
| [rust.md](rust.md) | diagnostics em profundidade: structs, spans, `Applicability`, `run-rustfix`, UI tests, editions |
| [gleam.md](gleam.md) | erro como variante com hipóteses; nomes como o usuário escreveu; tolerância a falhas; poda antes da v1 |
| [swift.md](swift.md) | Swift Evolution (template, status, `commonly_proposed`); fix-it "único e óbvio"; swift-syntax |
| [typescript.md](typescript.md) | um front-end para compiler e editor; catálogo de diagnósticos; `relatedInformation`; baselines; port para Go |
| [python.md](python.md) | gramática única e executável; erros especializados do 3.10; PEP 617; PEP 1/387 |
| [nim.md](nim.md) | layout como pseudo-terminais; tab é erro; igualdade de grafias e consistência por uso |
| [elixir.md](elixir.md) | contrato de deprecação; `mix format --migrate`; doctest; macros/DSL como contraexemplo |
| [roc.md](roc.md) | fronteira de efeitos verificada; inferência explicada; tratado como experimento, não como prova |
| [sintaxe-e-configuracao.md](sintaxe-e-configuracao.md) | Haskell layout, Adams, YAML, CUE, HCL, Dhall, Nickel, Pkl, KDL, Inform 7; árvores sem perdas; sondagens reproduzidas no código |

### Auditorias anteriores no mesmo diretório

[python-problemas.md](python-problemas.md) e [javascript-problemas.md](javascript-problemas.md)
são auditorias de problemas das linguagens, anteriores a este estudo; não foram usadas como
fonte da consolidação.

### Pesquisas vizinhas

- [../sintaxe-hierarquica.md](../sintaxe-hierarquica.md): base da sintaxe hierárquica atual
  (pilha do Python, CUE/TOML, Hedy, Cognitive Dimensions, Elm/rustc, gofmt/Black). Este estudo
  não a repete; aprofunda.
- [../frontend/](../frontend/README.md) e [../performance/](../performance/RUNTIMES.md):
  estudos paralelos de frontends e de desempenho.

## Como ler

1. Comece por [GERMANIO_LESSONS.md](GERMANIO_LESSONS.md). Cada lição tem a fonte (arquivo do
   estudo + URL), o problema concreto do Germanio, o arquivo afetado e a relação com a norma.
2. Para uma lição de sintaxe, siga para [SYNTAX_COMPARISON.md](SYNTAX_COMPARISON.md); de
   arquitetura, [COMPILER_ARCHITECTURE.md](COMPILER_ARCHITECTURE.md); de mensagens,
   [DIAGNOSTICS.md](DIAGNOSTICS.md); de processo, [LANGUAGE_EVOLUTION.md](LANGUAGE_EVOLUTION.md).
3. Para verificar uma afirmação sobre outra linguagem, vá ao estudo por linguagem citado e
   à URL. Afirmações que os estudos não conseguiram verificar estão marcadas
   "(não verificado)" lá e aqui.
4. Afirmações sobre o código do Germanio citam `arquivo:linha` da revisão `fc31daf` (master,
   2026-09-28). As divergências foram **reproduzidas** com o binário `ge` construído dessa
   revisão, sobre arquivos temporários fora do repositório; as não reproduzidas estão
   marcadas.

## Convenções

- **ADOTAR**: resolve um problema concreto do Germanio e pode entrar quase como está.
- **ADAPTAR**: a ideia serve, mas precisa mudar para o público leigo, o determinismo ou a
  escala do Germanio.
- **EVITAR**: funciona na origem, mas aqui criaria conceito, dependência escondida ou custo
  sem problema correspondente.
- **INVESTIGAR**: promissor, sem evidência suficiente; exige experimento ou medição.
- "Exige decisão deliberada": a lição contradiz a norma atual. A pesquisa não altera a norma;
  a decisão é do responsável pelo design (ver o GEP em
  [LANGUAGE_EVOLUTION.md](LANGUAGE_EVOLUTION.md)).
