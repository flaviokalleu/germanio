# C#: estudo dirigido para o Germanio

**Data:** 2026-09-28. **Status:** pesquisa, sem força normativa (a norma é `docs/INTENCAO.md`).
**Pergunta principal:** como o C# introduziu regras mais rígidas num ecossistema enorme sem
quebrá-lo, como o compilador virou plataforma de ferramentas, e o que o Entity Framework
ensina sobre esquema derivado do modelo?

## Fontes consultadas

Consultadas nesta sessão (WebFetch):

- [Nullable migration strategies](https://learn.microsoft.com/en-us/dotnet/csharp/nullable-migration-strategies) (atualizado em 2026)
- [Roslyn Overview (wiki do repositório)](https://github.com/dotnet/roslyn/blob/main/docs/wiki/Roslyn-Overview.md)
- [.NET Compiler Platform SDK: analyzers, code fixes, source generators](https://learn.microsoft.com/en-us/dotnet/csharp/roslyn-sdk/)
- [Native AOT deployment overview](https://learn.microsoft.com/en-us/dotnet/core/deploying/native-aot/)
- [EF Core: Migrations overview](https://learn.microsoft.com/en-us/ef/core/managing-schemas/migrations/)
- [EF Core: Managing migrations](https://learn.microsoft.com/en-us/ef/core/managing-schemas/migrations/managing)

Do conhecimento prévio, **não verificado nesta sessão**: detalhes de Blazor (componentes
Razor, modos Server/WebAssembly/Auto, SignalR no modo Server), a crítica de "coloração de
funções" aplicada a `async/await` (Bob Nystrom, "What Color is Your Function?", 2015), e o
processo de design em `dotnet/csharplang` (LDM notes públicas).

Código do Germanio conferido: `runtime/banco/banco.go:230-300` (auto-migração),
`cli/cli.go` (`germanio build`), `runtime/hotreload.go`.

---

## Matriz (compacta)

| Item | C# | Relevância para o Germanio |
|---|---|---|
| Objetivo original | Linguagem gerenciada para .NET (Microsoft, 2000) | não se aplica |
| Filosofia | Evolução rápida e anual, com mudanças rígidas introduzidas como opt-in e avisos | Modelo de migração gradual (ver ADOTAR) |
| Gramática / lexer / parser | Parser escrito à mão, tolerante a erros: insere tokens ausentes (`IsMissing`) ou guarda tokens pulados como trivia, e a árvore continua navegável ([Roslyn Overview](https://github.com/dotnet/roslyn/blob/main/docs/wiki/Roslyn-Overview.md)) | Recuperação sem cascata (A8) |
| AST | Árvore imutável, thread-safe, "snapshot" do código; fidelidade total (espaços, comentários, diretivas), volta ao texto original byte a byte ([Roslyn Overview](https://github.com/dotnet/roslyn/blob/main/docs/wiki/Roslyn-Overview.md)) | Mesma meta de P4 e P14 em `GERMANIO_LESSONS.md` |
| Análise semântica | Semantic model sobre a árvore; workspaces imutáveis de solução/projeto/documento | Workspace para o futuro `ge lsp` (A17, P14) |
| Tipos / inferência | Nominal, estático, `var`; nullable reference types (NRT) como análise de fluxo com avisos | Ver Sintaxe |
| Compiler / runtime | IL + JIT (CoreCLR); ou Native AOT: sem JIT, sem carga dinâmica, sem `Reflection.Emit`, exige trimming; ganha partida rápida e menos memória ([Native AOT](https://learn.microsoft.com/en-us/dotnet/core/deploying/native-aot/)) | `germanio build` já gera binário único (Go); a lição está na análise, não no binário |
| Memória | GC | não se aplica |
| Package manager | NuGet; analyzers distribuídos junto com bibliotecas ([SDK](https://learn.microsoft.com/en-us/dotnet/csharp/roslyn-sdk/)) | Ver EVITAR |
| Formatter / linter | `dotnet format` + regras de estilo como analyzers configuráveis por `.editorconfig` (não verificado em detalhe) | Configurável demais para o Germanio (P13) |
| LSP / IDE | Roslyn é a base do Visual Studio e do C# Dev Kit: um compilador, muitos consumidores | A6, A17 |
| Diagnostics | Avisos com código estável e code fixes (edições aplicáveis) ([SDK](https://learn.microsoft.com/en-us/dotnet/csharp/roslyn-sdk/)) | A1 (diagnóstico estruturado com sugestão como edição) |
| Evolution process | `dotnet/csharplang`, notas do LDM públicas (não verificado nesta sessão) | Ver `kotlin.md` e `java.md` |
| Backward compatibility | NRT não altera o runtime: é só anotação e aviso; código "oblivious" convive com código anotado ([migration](https://learn.microsoft.com/en-us/dotnet/csharp/nullable-migration-strategies)) | Ver ADOTAR |
| Principais acertos | Compilador como plataforma; migração por avisos; source generators; EF Migrations com snapshot | — |
| Principais problemas | Muitos modos simultâneos (NRT on/off/warnings/annotations; JIT/AOT; Blazor Server/WASM/Auto); `async` contamina assinaturas; renomear propriedade no EF vira apagar coluna + criar coluna ([managing](https://learn.microsoft.com/en-us/ef/core/managing-schemas/migrations/managing)) | — |
| Complexidade acumulada | Alta: 13+ versões de sintaxe convivendo; várias formas de declarar a mesma coisa | O Germanio deve ter **uma** forma normal (P5) |

---

## Sintaxe

| Pergunta | C# | Germanio hoje (IMPLEMENTADO) | Comparação |
|---|---|---|---|
| Dados | `record`, classes com propriedades auto-implementadas; entidades EF são classes simples | `cada X tem` | O EF deriva o esquema da classe, como o Germanio; mas exige `DbContext`, `DbSet`, configuração fluente |
| Funções | Métodos, lambdas, funções locais, extensões | Nível 3 | — |
| Módulos / imports | `namespace` + `using`; `global using` e "implicit usings" nos templates (não verificado) | `importar` explícito no frontend | Imports implícitos reduzem ruído mas escondem a origem do nome; o Germanio mostra a origem com `ge explain` |
| Relações | Propriedades de navegação + convenções do EF (FK inferida pelo nome) | `tem` / `pertence a` | Mesmo princípio de convenção; o Germanio é explícito na frase |
| Estado | Propriedades mutáveis | `começa`, `pode` | — |
| Fluxo | `if`, `switch` com patterns | Condições declarativas | — |
| Erros | Exceções | Erros por campo, transação | — |
| Concorrência | `async`/`await`; `Task`; toda função que espera vira `async` e contamina os chamadores (coloração; não verificado nesta sessão) | Servidor Go sem coloração; nível 3 com builtins | O Germanio não deve ter "cor" de função; Java e Go mostram que dá para escrever sequencial e o runtime escalar (`java.md`) |
| Tipos | Nominais | Pelo nome ou explícitos | — |
| Null | NRT: `string` não nulo, `string?` nulo, mas só por **aviso**; o contexto tem duas chaves independentes (anotações, avisos) com quatro modos ([migration](https://learn.microsoft.com/en-us/dotnet/csharp/nullable-migration-strategies)) | `obrigatório` | Quatro modos são conceitos demais; o que vale é a *técnica* de migração |
| Organização | Solução / projeto / pasta; `.csproj` | `backend/`, `frontend/`, `integracoes/` | — |
| Boilerplate | Records, primary constructors, top-level statements, minimal APIs, source generators | Inferência | Source generators geram código em compilação e só **acrescentam** (não alteram o código do autor) ([SDK](https://learn.microsoft.com/en-us/dotnet/csharp/roslyn-sdk/)) |
| Legibilidade em projetos grandes | Boa com IDE; ruim com mistura de estilos de épocas diferentes | Formatter canônico e uma forma normal | — |

### Migrations do EF versus a auto-migração do Germanio

O EF compara o modelo atual com um **snapshot** versionado do modelo anterior, gera um
arquivo de migração revisável, registra as aplicadas numa tabela de histórico e avisa
quando uma operação pode perder dados. A própria documentação admite o limite: renomear
`Name` para `FullName` vira `DropColumn` + `AddColumn`, porque o EF "is generally unable to
know" se era renomeação; aplicado assim, "all your customer names will be lost"
([managing](https://learn.microsoft.com/en-us/ef/core/managing-schemas/migrations/managing)).
Desde o EF Core 9, aplicar migrações com mudanças de modelo pendentes lança exceção, e há
`has-pending-model-changes` para CI (mesma fonte).

**O Germanio está errado aqui.** Conferido em `runtime/banco/banco.go:262-277`: a migração é
`CREATE TABLE IF NOT EXISTS` seguido de `ALTER TABLE … ADD COLUMN` para cada campo, com o erro
**ignorado** (`b.DB.Exec(alterSQL) // ignore error if column already exists`). Consequências:

1. Renomear um campo no `.ge` cria coluna nova vazia; a antiga fica órfã e os dados
   "somem" da aplicação sem aviso (pior que o EF, que ao menos avisa perda de dados).
2. Mudar o tipo ou tornar um campo `obrigatório`/`único` não altera a coluna existente.
3. Um `ADD COLUMN` que falha por outro motivo (tipo inválido no driver, valor padrão com
   aspas) é indistinguível de "já existe".
4. O valor padrão é interpolado no SQL entre aspas simples (`DEFAULT '%s'`, linha 275): um
   `começa com "d'água"` quebra a instrução, e o erro é engolido (item 3).
5. Não há snapshot nem histórico: o Germanio não sabe o que mudou entre duas execuções.

O Germanio não deve copiar o arquivo de migração em código (é o boilerplate que ele elimina),
mas precisa do **snapshot** e do **diff**.

---

## Para o Germanio

**ADOTAR — introduzir regras mais rígidas primeiro como aviso, depois como erro.**
A migração de NRT separa "o compilador avisa" de "o tipo muda" e permite avançar arquivo
por arquivo ([migration](https://learn.microsoft.com/en-us/dotnet/csharp/nullable-migration-strategies)).
Problema: toda nova checagem do `ge check` quebra projetos existentes de uma vez. Proposta:
uma checagem nova nasce como aviso com prazo (P7) e vira erro na versão seguinte; sem os
quatro modos do C#. Arquivo: `compiler/diagnostics/`, `tooling/gecli/`. Remove da cabeça:
"a atualização vai quebrar meu projeto?".

**ADAPTAR — snapshot do esquema e diff explicado, sem arquivos de migração.**
Problema: auto-migração só aditiva, com erros ignorados (`runtime/banco/banco.go:262-277`).
Proposta: gravar no banco o esquema derivado (`ast.App` → tabelas) aplicado por último; na
partida, calcular o diff; mudanças aditivas seguem automáticas; mudanças destrutivas ou
ambíguas (campo sumiu e outro do mesmo tipo apareceu) param com erro educativo: "o campo
`nome` sumiu e `nome_completo` apareceu; se é o mesmo, escreva `nome_completo antes chamado
nome`" (sintaxe a decidir). `ge explain` / `ge check` mostram o diff pendente. Remove da
cabeça: SQL, arquivos de migração e medo de perder dados. Corrigir também o erro ignorado e
a interpolação do valor padrão (usar o valor como literal escapado ou omitir `DEFAULT` e
preencher pela aplicação).

**ADAPTAR — árvore imutável, fiel e tolerante a erros como base do `ge lsp`.** Já é P4/P14;
Roslyn é a evidência de que um único front-end serve compilador e IDE
([Roslyn Overview](https://github.com/dotnet/roslyn/blob/main/docs/wiki/Roslyn-Overview.md)).
Arquivo: `compiler/parser/hierarquia.go`.

**EVITAR — analyzers e geradores distribuídos por pacotes de terceiros.** Eles mudam o que
é erro e o que existe no programa conforme as dependências instaladas; para o Germanio,
regras vêm só do core (`compiler/`), e `integracoes/` não acrescenta checagens nem sintaxe.

**EVITAR — múltiplos modos de execução visíveis ao autor** (JIT/AOT, Blazor Server/WASM/Auto,
quatro modos de NRT). Se o runtime precisar de modos, o padrão decide e `ge explain` mostra.

**INVESTIGAR — o que no runtime do Germanio é "reflexão" que poderia ser geração.** O
interpretador trabalha com `map[string]any` e resolve campos por nome em tempo de execução;
`germanio build` embute os `.ge` e interpreta (não verificado se há pré-resolução). O Native
AOT mostra o ganho de mover trabalho para a compilação (partida, memória). Medir antes em
`docs/research/performance/`.

**INVESTIGAR — Blazor como frontend sem JavaScript.** O modo Server mantém estado da UI no
servidor e troca diffs por conexão persistente (não verificado nesta sessão); é o modelo mais
próximo de páginas `.ge` geradas no servidor com tempo real (`runtime/servidor/websocket.go`).
Custo conhecido do modelo: conexão por usuário e latência em cada interação. Comparar com
`docs/research/frontend/`.
