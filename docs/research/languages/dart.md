# Dart (e Flutter): estudo dirigido para o Germanio

**Data:** 2026-09-28. **Status:** pesquisa, sem força normativa (a norma é `docs/INTENCAO.md`).
**Pergunta principal:** como o Dart migrou um ecossistema inteiro para uma garantia mais
forte (null safety sólida), como o Flutter torna a interface declarativa e o ciclo de edição
instantâneo, e o que a mudança de estilo do formatter em 2024-2025 ensina sobre um formatter
sem opções?

## Fontes consultadas

Consultadas nesta sessão (WebFetch):

- [Understanding null safety](https://dart.dev/null-safety/understanding-null-safety)
- [Migrating to null safety](https://github.com/dart-community/migrate-to-null-safety/blob/main/docs/migrate.md) (destino do redirecionamento de `dart.dev/null-safety/migration-guide`)
- [Flutter: Hot reload](https://docs.flutter.dev/tools/hot-reload)
- [dart_style#1253: proposta do estilo "tall"](https://github.com/dart-lang/dart_style/issues/1253)
- [dart format (documentação)](https://dart.dev/tools/dart-format)

Do conhecimento prévio, **não verificado nesta sessão**: o modelo de widgets do Flutter
(`StatelessWidget`/`StatefulWidget`, `build` reexecutado a cada `setState`, reconciliação
por tipo e chave), o `pub` (`pubspec.yaml`, `pubspec.lock`, pontuação do pub.dev,
resolução com uma única versão por pacote), o analisador compartilhado por `dart analyze`
e pelo LSP, e a data exata de lançamento do estilo tall (Dart 3.7, início de 2025).

Código do Germanio conferido: `runtime/hotreload.go`, `runtime/banco/banco.go:262-277`,
`tooling/formatter/`, `docs/research/languages/FORMATTERS.md` (não repetido aqui).

---

## Matriz (compacta)

| Item | Dart / Flutter | Relevância para o Germanio |
|---|---|---|
| Objetivo original | Linguagem para a web estruturada (Google, 2011); hoje linguagem do Flutter | não se aplica |
| Filosofia | Null safety "safe by default", fácil de escrever e sólida ([understanding](https://dart.dev/null-safety/understanding-null-safety)); ferramenta única (`dart`) | — |
| Gramática / parser / AST | Front-end compartilhado (CFE) produz kernel; analisador separado para IDE (não verificado) | Dois front-ends é o risco já apontado para o Germanio (A6) |
| Tipos / inferência | Estático, sólido, com inferência local; promoção de tipo por fluxo (`if (x != null)`) ([understanding](https://dart.dev/null-safety/understanding-null-safety)) | — |
| Compiler / runtime | VM com JIT (desenvolvimento, hot reload) e AOT (produção); JS e Wasm | O modo JIT existe *para* o hot reload |
| Memória | GC geracional | não se aplica |
| Package manager | pub, `pubspec.yaml`/`pubspec.lock` (não verificado) | — |
| Formatter | `dart format`, historicamente sem opções; o estilo tall trata a vírgula final como espaço em branco, adicionada ou removida automaticamente ([#1253](https://github.com/dart-lang/dart_style/issues/1253)); estilo escolhido pela **versão da linguagem** do pacote; depois vieram `page_width` (3.7) e `trailing_commas: preserve` (3.8) em `analysis_options.yaml` ([dart format](https://dart.dev/tools/dart-format)) | Ver Sintaxe/formatter |
| Linter / LSP / IDE | `dart analyze` e LSP no SDK (não verificado) | A16, A17 |
| Diagnostics | Mensagens com correção sugerida (não verificado) | — |
| Evolution process | Propostas em `dart-lang/language` (não verificado); mudança de estilo com protótipo, corpus de diffs reais, pesquisa pública e prazo ([#1253](https://github.com/dart-lang/dart_style/issues/1253)) | Método de decisão aproveitável |
| Backward compatibility | Language versioning por pacote: cada pacote declara a versão mínima do SDK e recebe a semântica dessa versão | Ver ADOTAR |
| Principais acertos | Migração de null safety de baixo para cima com ferramenta; soundness que habilita otimização; hot reload com estado preservado | — |
| Principais problemas | Migração exigiu esperar as dependências ("leaves of the dependency graph being migrated first"); fase de programas mistos sem garantia; o Dart 3 cortou quem não migrou ([migrate](https://github.com/dart-community/migrate-to-null-safety/blob/main/docs/migrate.md)); o formatter sem opções ganhou opções | — |
| Complexidade acumulada | `late`, `required`, `!`, `?` e hints de migração (`/*!*/`, `/*?*/`) | Conceitos que o Germanio não deve precisar |

---

## Sintaxe

| Pergunta | Dart / Flutter | Germanio hoje (IMPLEMENTADO) | Comparação |
|---|---|---|---|
| Dados | Classes com campos `final`, construtores nomeados; records (Dart 3) | `cada X tem` | — |
| Funções | Funções de primeira classe, argumentos nomeados `required` | Nível 3 | — |
| Módulos / imports | `import 'package:x/y.dart'`; biblioteca = arquivo | `importar` | — |
| Relações | Não há | `tem` / `pertence a` | — |
| Estado | UI: `StatefulWidget` + `setState`, ou gerenciadores externos (Provider, Riverpod, Bloc) (não verificado) | Estado de registro por ação (`começa`, `pode`) | A escolha de gerenciador de estado é uma decisão que o Flutter deixa ao programador; o Germanio não deve deixar |
| Fluxo | `if`, `switch` com patterns (Dart 3) | Declarativo | — |
| Erros | Exceções | Por campo, transação | — |
| Concorrência | `async`/`await` com coloração; isolates sem memória compartilhada | Goroutines; nível 3 | — |
| Tipos | Sólidos | Pelo nome | — |
| Null | `T` não nulo, `T?` nulo, garantido em execução (sólido): "no possible execution of that expression can ever evaluate to null" ([understanding](https://dart.dev/null-safety/understanding-null-safety)) | `obrigatório` | Ver INVESTIGAR em `kotlin.md`; o Dart mostra que só a solidez permite ao compilador remover checagens |
| Organização | `lib/`, `test/`, `pubspec.yaml` | `backend/`, `frontend/` | — |
| Boilerplate | Interface por árvore de widgets aninhados em código; muita indentação e vírgulas finais | Páginas declaradas em `.ge` (`crie página …`) | A árvore de widgets é código imperativo com forma de árvore; o estilo tall existe porque essa forma dominou o código real |
| Legibilidade em projetos grandes | Árvores de widgets profundas; extrair widgets é a técnica padrão | Seções fechadas e frases planas | — |

### Hot reload: Flutter versus Germanio

O Flutter injeta o código alterado na VM em execução, reconstrói a árvore de widgets e
**preserva o estado**; mudanças que não cabem nisso (inicializadores estáticos, `main`,
enum → classe, genéricos) exigem hot restart, e a documentação lista cada caso
([hot reload](https://docs.flutter.dev/tools/hot-reload)).

O Germanio (conferido em `runtime/hotreload.go`) faz uma varredura de todos os `.ge` a cada
segundo e, ao detectar mudança, **inicia um novo processo e sai** (`os.Exit(0)`). Como o
estado dos dados está no banco, reiniciar é aceitável; os problemas são outros:

1. **O novo código não é verificado antes de derrubar o processo antigo.** Se o `.ge`
   salvo tem erro, o processo novo falha e o servidor de desenvolvimento deixa de existir
   (não verificado em execução; deduzido do código: o antigo sai logo após `cmd.Start()`).
   O Flutter mantém o app rodando e mostra o erro.
2. **Cada recarga roda a auto-migração aditiva** (`runtime/banco/banco.go:262-277`): um nome
   de campo digitado errado, salvo por um instante, vira coluna permanente no banco de
   desenvolvimento. Nenhum aviso.
3. Não há distinção documentada entre o que recarrega e o que exige reinício (o Flutter tem
   a tabela).

### O formatter sem opções que ganhou opções

A proposta do estilo tall foi decidida com protótipo, corpus de diffs reais de 2.000 pacotes
(71% dos casos distinguíveis já preferiam o estilo alto) e pesquisa pública; o autor
admitiu que "no change of this scale will please everyone"
([#1253](https://github.com/dart-lang/dart_style/issues/1253)). O novo estilo passou a valer
conforme a versão da linguagem do pacote, e depois o `dart format` passou a aceitar
`page_width` e `trailing_commas: preserve` em `analysis_options.yaml`, mais um comentário
por arquivo ([dart format](https://dart.dev/tools/dart-format)). É evidência para P13
(`GERMANIO_LESSONS.md`: versionar o estilo com a versão da linguagem, sem opções) e ao mesmo
tempo um aviso: mudar o estilo de um formatter canônico gera pressão por opções. A sintaxe
hierárquica do Germanio reduz esse risco porque a quebra de linha é significado, não estilo
(não há vírgulas finais nem largura de linha para decidir), desde que o formatter não
precise escolher entre forma plana e forma hierárquica — se precisar, é exatamente a
decisão "tall versus short" do Dart.

---

## Para o Germanio

**ADOTAR — recarga que valida antes de trocar.** Problema: `runtime/hotreload.go` troca o
processo sem checar o `.ge` novo. Proposta: parsear e resolver no processo atual (ou num
filho) e só trocar se `ge check` passa; senão manter o servidor e mostrar o diagnóstico no
terminal e na página aberta. Remove da cabeça: "salvei com erro e o servidor morreu".
Também trocar a varredura por segundo por notificação de arquivos quando possível
(desempenho, `docs/research/performance/`).

**ADOTAR — mudança de esquema em desenvolvimento só depois de estabilizar.** Problema: a
auto-migração em cada recarga cria colunas para nomes temporários. Proposta: ligada ao
snapshot/diff de `csharp.md`: o diff é mostrado e aplicado quando o `.ge` passa em
`ge check`, e colunas órfãs aparecem em `ge explain` como "sem campo no `.ge`". Arquivo:
`runtime/banco/banco.go`, `runtime/hotreload.go`.

**ADOTAR — semântica escolhida pela versão da linguagem declarada no projeto.** O Dart
seleciona regras e estilo pela versão de cada pacote ([dart format](https://dart.dev/tools/dart-format)).
Junta-se ao INVESTIGAR de `ruby.md` (versão no `app.ge`) e a P13. Remove da cabeça: "por que
o `ge fmt` mudou meu arquivo depois de atualizar?".

**ADAPTAR — decidir mudanças de estilo ou sintaxe com corpus real.** O processo do tall
(protótipo + diff de corpus + prazo) é aplicável aos exemplos e demos do repositório
(`demo/`, `examples/`): toda mudança de formatter ou sintaxe publica o diff sobre eles antes.

**EVITAR — migração que depende de todo o ecossistema andar junto.** A null safety do Dart
exigiu migrar folhas primeiro e cortou quem não migrou no Dart 3
([migrate](https://github.com/dart-community/migrate-to-null-safety/blob/main/docs/migrate.md)).
No Germanio, uma garantia nova deve ser aplicável por projeto, sem esperar adaptadores de
terceiros.

**EVITAR — deixar a gestão de estado da interface como escolha do programador** (o
ecossistema Flutter tem várias). O estado das páginas do Germanio deve continuar derivado
dos dados e das ações declaradas.

**INVESTIGAR — interface declarativa com atualização local.** O Flutter reconstrói a árvore
e reconcilia (não verificado nesta sessão). Para páginas `.ge` com tempo real
(`runtime/servidor/websocket.go`), comparar com `docs/research/frontend/` antes de qualquer
decisão.
