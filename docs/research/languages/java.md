# Java: estudo dirigido para o Germanio

**Data:** 2026-09-28. **Status:** pesquisa, sem força normativa (a norma é `docs/INTENCAO.md`).
**Pergunta principal:** como uma linguagem evolui por décadas sem quebrar ninguém, e o que
acontece quando regras de dados e comportamento de aplicação ficam em anotações e em
configuração automática implícita?

## Fontes consultadas

Consultadas nesta sessão (WebFetch):

- [JEP 12: Preview Features](https://openjdk.org/jeps/12)
- [JEP 395: Records](https://openjdk.org/jeps/395)
- [JEP 444: Virtual Threads](https://openjdk.org/jeps/444)
- [JEP 505: Structured Concurrency (Fifth Preview, JDK 25)](https://openjdk.org/jeps/505)
- [Jakarta Bean Validation 3.0 spec](https://jakarta.ee/specifications/bean-validation/3.0/jakarta-bean-validation-spec-3.0.html)
- [Spring Boot: Auto-configuration](https://docs.spring.io/spring-boot/reference/using/auto-configuration.html)

Do conhecimento prévio, **não verificado nesta sessão**: JEP 409 (sealed classes, Java 17),
JEP 441 (pattern matching para `switch`, Java 21), o processo JEP 1 e o JCP, detalhes de
javac (árvore `com.sun.source.tree`) e o fato de o JDK não ter formatter oficial.

Código do Germanio conferido: `runtime/banco/banco.go:451,507,695` (Validar em criar e
atualizar), `runtime/servidor/paginas.go:1303-1305` (formulário), `runtime/interpreter/interpreter.go:1182-1219`.

---

## Matriz (compacta)

| Item | Java | Relevância para o Germanio |
|---|---|---|
| Objetivo original | Linguagem portátil (JVM), orientada a objetos, segura para rede (Sun, 1995) | não se aplica |
| Filosofia | Conservadorismo: legibilidade e compatibilidade acima de concisão; records "não declaram guerra ao boilerplate", modelam "dados como dados" ([JEP 395](https://openjdk.org/jeps/395)) | Mesma preferência do Germanio por significado sobre número de caracteres |
| Gramática / parser / AST | Gramática da JLS; javac com árvore pública para ferramentas (não verificado) | não se aplica |
| IR | Bytecode; JIT (HotSpot C1/C2) | não se aplica |
| Tipos / inferência | Nominal, estático; `var` local (Java 10); sem nulabilidade no tipo | Contraste com Kotlin (`kotlin.md`) |
| Runtime / memória | JVM com GC; virtual threads em Java 21 ([JEP 444](https://openjdk.org/jeps/444)) | Modelo thread-por-requisição é o do servidor Go do Germanio |
| Stdlib | Enorme e estável | não se aplica |
| Package manager | Maven/Gradle (terceiros), coordenadas `grupo:artefato:versão` | não se aplica |
| Formatter / linter | Nenhum oficial (não verificado); ecossistema de Checkstyle, google-java-format | Contraexemplo; ver `FORMATTERS.md` |
| LSP / IDE | IDEs fortes (IntelliJ, Eclipse JDT); jdtls | não se aplica |
| Diagnostics | Mensagens de javac tradicionais; "helpful NullPointerExceptions" (JEP 358) (não verificado) | Ver `DIAGNOSTICS.md` |
| Evolution process | JEPs; preview = feature completa (padrão de qualidade de feature final), mas impermanente; exige `--enable-preview` na compilação **e** na execução; tipicamente dois ciclos de preview antes de final ou remoção ([JEP 12](https://openjdk.org/jeps/12)) | Ver ADAPTAR |
| Backward compatibility | Quase absoluta; class files com preview marcados (`minor_version` 65535) para não rodarem sem opt-in ([JEP 12](https://openjdk.org/jeps/12)) | Marcar o artefato que usa preview é aproveitável |
| Principais acertos | Evolução previsível; records; virtual threads sem mudar o modelo mental; structured concurrency | — |
| Principais problemas | Null não está no tipo; anotações sem efeito até que um framework as leia; Spring faz a aplicação depender do classpath | — |
| Complexidade acumulada | Genéricos com apagamento, checked exceptions, três modelos de concorrência (threads, `CompletableFuture`/reativo, virtual threads) | O reativo foi, na prática, substituído pelas virtual threads |
| Aprender / não copiar | Ver "Para o Germanio" | — |

---

## Sintaxe

| Pergunta | Java | Germanio hoje (IMPLEMENTADO) | Comparação |
|---|---|---|---|
| Dados | `record Ponto(int x, int y)`: portadores transparentes de dados imutáveis ([JEP 395](https://openjdk.org/jeps/395)); entidades JPA com `@Entity` | `cada ponto tem` + campos | O record é bom exatamente porque diz "isso é dado" e deixa o compilador derivar o resto — o mesmo princípio do Germanio, em escala de objeto |
| Funções | Métodos em classes; lambdas desde Java 8 | Nível 3 apenas | — |
| Módulos / imports | pacotes + `import`; JPMS (`module-info.java`) raramente usado (não verificado) | `importar` de pastas e dados (`INTENCAO.md`, "Organização do projeto") | JPMS é o caso de sistema de módulos que chegou tarde e ninguém adotou; o Germanio acertou em ter o seu desde cedo |
| Relações | Anotações JPA `@OneToMany(mappedBy=…)`, `@ManyToOne`, fetch LAZY/EAGER | `cliente tem pedidos` (`compiler/parser/resolver.go`) | O JPA exige saber lado dono, `mappedBy`, estratégia de carga; o Germanio deriva |
| Estado | Campos mutáveis; enums; sealed interfaces para estados fechados | `começa aberta`, `pode fechar` | O sealed + pattern matching (não verificado nesta sessão) garante exaustividade no `switch`; o Germanio garante por construção (estados só por ação) |
| Fluxo | `if`, `switch` com patterns (Java 21) | Condições declarativas | — |
| Erros | Exceções checadas e não checadas | Erros por campo, transação | Checked exceptions são o exemplo clássico de "coloração" de erro; o Germanio não expõe exceção |
| Concorrência | Virtual threads: estilo thread-por-requisição, sem async ([JEP 444](https://openjdk.org/jeps/444)); structured concurrency: "se uma tarefa se divide em subtarefas, todas voltam ao mesmo lugar"; falha de uma cancela as irmãs ([JEP 505](https://openjdk.org/jeps/505)) | Servidor Go (goroutine por requisição); nível 3 com `paralelo`/`timeout` sem escopo (`interpreter.go:1182-1219`, ver `kotlin.md`) | Java confirma a direção: o autor escreve código sequencial, o runtime escala. O Germanio já está nessa posição no nível 1/2 |
| Tipos | Nominais | Inferidos pelo nome | — |
| Null | Qualquer referência pode ser nula; `Optional` só em retornos | Opcional por padrão; `obrigatório` | — |
| Organização | Convenção Maven `src/main/java`; pacotes = diretórios | `backend/`, `frontend/`, `integracoes/` | — |
| Boilerplate | Records, `var`, Lombok (processador de anotações que altera a AST, fora da linguagem) | Inferência total | Lombok é o sintoma: quando a linguagem não reduz boilerplate, uma ferramenta implícita reduz, e passa a ser necessária para entender o código |
| Legibilidade em projetos grandes | Alta no código explícito; baixa no comportamento vindo de anotações e auto-configuração | `ge explain` com origem de cada fato | Ver abaixo |

### Anotações como regras declarativas de dados

Bean Validation nasceu do mesmo problema que o Germanio resolve: "the same validation logic
is implemented in each layer, proving to be time consuming and error-prone"
([Bean Validation](https://jakarta.ee/specifications/bean-validation/3.0/jakarta-bean-validation-spec-3.0.html)).
A solução é metadado declarativo (`@NotNull`, `@Size`) mais relatório estruturado com o
caminho da propriedade. Problemas conhecidos:

1. **Anotação é inerte.** Só vale onde algum framework chama o validador; um caminho de
   escrita que não chama (SQL direto, atualização em lote) ignora a regra (conhecimento
   prévio, não verificado em fonte nesta sessão).
2. **Grupos e sequências** existem porque a mesma regra não vale em todo contexto
   (cadastro vs edição) ([spec](https://jakarta.ee/specifications/bean-validation/3.0/jakarta-bean-validation-spec-3.0.html)):
   um conceito a mais, com nomes de classe como marcadores.

No Germanio a validação está no ponto único de escrita: `banco.Validar` é chamado em criar e
atualizar (`runtime/banco/banco.go:451,507`), então o item 1 não se aplica aos caminhos que
passam pelo banco. O equivalente a grupos é resolvido por palavra com semântica fixa
(`imutável` só pesa na edição, `runtime/interpreter/schema.go:178`). Isso é tecnicamente
melhor para o público. Lacuna encontrada: o formulário gerado só espelha `obrigatório`
(`required`) em `paginas.go:1303-1305`; `min`, `max` e `formato` só aparecem depois do envio.

### Spring: mágica implícita

O Spring Boot configura a aplicação "com base nas dependências jar do classpath" e "recua"
quando o autor define o seu próprio bean; para saber o que foi aplicado e por quê, é preciso
iniciar com `--debug` e ler o "conditions report"
([auto-configuration](https://docs.spring.io/spring-boot/reference/using/auto-configuration.html)).
Ou seja: o comportamento depende de algo que não está no código (o classpath), e a
explicação é um log opcional. O Germanio também tem padrões que "recuam" diante de
declaração explícita ("uma declaração explícita sempre vence a heurística",
`INTENCAO.md`), mas a explicação é um comando de primeira classe (`ge explain`) e a entrada
é só o `.ge`. A diferença a preservar: **nenhum padrão do Germanio pode depender de algo fora
dos arquivos `.ge` e do `.env` declarado** (por exemplo, da presença de um binário, pasta ou
variável não citada).

---

## Para o Germanio

**ADOTAR — o padrão de qualidade do preview do JEP 12.** Uma construção em preview é
completa e testada como final, só que impermanente, e o artefato que a usa é marcado
([JEP 12](https://openjdk.org/jeps/12)). Problema: o Germanio não distingue construção
estável de experimental. Arquivo: `docs/INTENCAO.md` (Evolução), `compiler/parser/`. Remove
da cabeça: "posso confiar nisso?". Casa com a proposta de KEEP em `kotlin.md`.

**ADOTAR — structured concurrency como garantia do runtime** (mesma lição de `kotlin.md`;
Java a chega por outro caminho, [JEP 505](https://openjdk.org/jeps/505)). Arquivo:
`runtime/interpreter/interpreter.go:1128-1260`. A convergência de duas linguagens é a evidência.

**ADAPTAR — espelhar no formulário todas as regras que o servidor verifica.** Problema:
`min`, `max`, `formato` não viram atributos do formulário (`runtime/servidor/paginas.go:1303-1305`).
A mesma fonte (o `ast.Field`) deve gerar `minlength`/`maxlength`/`pattern`, com o servidor
continuando a ser a autoridade. Remove da cabeça: "por que só descobri o erro depois de enviar?".

**EVITAR — comportamento derivado do ambiente (classpath) e explicação só em log.** Regra
proposta: todo padrão aplicado aparece em `ge explain` com o motivo, e nenhum depende de
algo fora dos arquivos do projeto. Arquivo: `tooling/explicar/`, `runtime/engine.go`.

**EVITAR — grupos de validação nomeados pelo autor.** Contextos (criar/editar) ficam em
palavras de semântica fixa (`imutável`, estados), não em marcadores.

**INVESTIGAR — caminhos de escrita que não passam por `banco.Validar`.** `DeletarFiltro`
e as operações em lote de `runtime/banco/consulta.go` devem ser auditadas para garantir que
nenhum caminho de alteração salte validação e invariantes (a lição de "anotação inerte").
