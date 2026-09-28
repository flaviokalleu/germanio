# JavaScript: estudo dirigido para o Germanio (processo, runtime e ferramentas)

**Data:** 2026-09-28. **Status:** pesquisa, sem força normativa (a norma é `docs/INTENCAO.md`).
**Pergunta principal:** o que acontece com uma linguagem que nunca pode quebrar código
publicado, e o que os runtimes novos (Deno, Bun) fizeram para reduzir permissões e
ferramentas que o programador precisa administrar?

**Escopo.** Os problemas da linguagem em si (coerção, `null`/`undefined`, `this`, números
IEEE 754, `Date`, CommonJS vs ESM, rejeições não tratadas, supply chain do npm, UTF-16,
mutação em métodos de array) já estão em [javascript-problemas.md](javascript-problemas.md)
e não são repetidos aqui. TypeScript está em [typescript.md](typescript.md).

## Fontes consultadas

Consultadas nesta sessão (WebFetch):

- [TC39 process document](https://tc39.es/process-document/)
- [SmooshGate (Chrome for Developers)](https://developer.chrome.com/blog/smooshgate)
- [MDN: JavaScript execution model](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Execution_model)
- [Deno: Security and permissions](https://docs.deno.com/runtime/fundamentals/security/)
- [Bun docs](https://bun.com/docs)

Do conhecimento prévio, **não verificado nesta sessão**: a toolchain integrada do Deno
(`deno fmt`, `deno lint`, `deno test`, `deno check`, `deno.lock`), o incidente `left-pad`
(2016), e a linguagem em que o Bun é escrito (historicamente Zig; a documentação atual diz
"built in Rust" para o engine — não conferido no repositório).

Código do Germanio conferido: `runtime/interpreter/interpreter.go` (`chamar`, builtins
assíncronos), `runtime/hotreload.go`, `docs/INTENCAO.md` ("Evolução e refatoração retroativa").

---

## Matriz (compacta)

| Item | JavaScript | Relevância para o Germanio |
|---|---|---|
| Objetivo original / filosofia | Script tolerante para páginas (1995); ver [javascript-problemas.md](javascript-problemas.md) §1 | — |
| Gramática / parser | Especificação ECMA-262 com ASI (inserção automática de `;`) e gramáticas de cobertura; cada engine tem seu parser | Contraexemplo de regra implícita de layout (ver `sintaxe-e-configuracao.md`) |
| AST / IR | ESTree (convenção da comunidade, não da spec); engines com bytecode + JIT em camadas (não verificado) | O ecossistema tem dezenas de parsers para a mesma língua; é o risco "vários parsers" do Germanio em escala |
| Tipos | Dinâmicos; tipos via TypeScript | `typescript.md` |
| Runtime | Agente com heap, pilha e fila de jobs; cada job roda até o fim sem preempção; microtasks (promises) drenadas antes das tasks; "JavaScript execution is never blocking" ([MDN](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Execution_model)) | Ver Sintaxe/concorrência |
| Memória | GC | não se aplica |
| Stdlib | Pequena historicamente; o ecossistema preencheu com micro-pacotes | — |
| Package manager | npm, yarn, pnpm, Bun; lockfiles diferentes | [javascript-problemas.md](javascript-problemas.md) Problema 8 |
| Formatter / linter / testes | Terceiros (Prettier, ESLint, Jest, Vitest); Deno e Bun trazem no binário ([Bun](https://bun.com/docs)) | A16 (um binário `ge`) |
| LSP / IDE | tsserver serve JS e TS | `typescript.md` |
| Evolution process | TC39 em estágios 0 → 1 → 2 → 2.7 → 3 → 4; o 2.7 exige spec completa e revisão antes de implementação; o 4 exige **duas implementações compatíveis passando o Test262** e experiência de campo ([process](https://tc39.es/process-document/)) | Ver ADAPTAR |
| Backward compatibility | Absoluta: "don't break the web"; `flatten` virou `flat` porque o nome quebrava sites com MooTools, e os donos desses sites não iriam atualizá-los ([SmooshGate](https://developer.chrome.com/blog/smooshgate)) | Ver Sintaxe/evolução |
| Principais acertos | Processo com testes de conformidade como critério; run-to-completion; Deno com permissões e toolchain integrada | — |
| Principais problemas | Nada pode ser removido; nomes piores para contornar legado; fragmentação de ferramentas e de módulos | — |
| Complexidade acumulada | Máxima: toda decisão ruim desde 1995 continua válida | — |

---

## Sintaxe

| Pergunta | JavaScript | Germanio hoje (IMPLEMENTADO) | Comparação |
|---|---|---|---|
| Dados | Objetos literais, classes; sem esquema | `cada X tem` | O Germanio tem esquema desde a primeira linha |
| Funções | `function`, arrow, métodos; `this` dinâmico | Nível 3 | — |
| Módulos / imports | ESM (`import`/`export`) e CommonJS convivendo | `importar` | [javascript-problemas.md](javascript-problemas.md) Problema 6: duas formas de módulo é o que o Germanio não pode ter |
| Relações | Não há | `tem` / `pertence a` | — |
| Estado | Variáveis mutáveis, closures | `começa`, `pode` | — |
| Fluxo | `if`, `switch`, ASI | Declarativo | — |
| Erros | `throw`; promises rejeitadas | Por campo, transação | — |
| Concorrência | Uma thread por agente, fila de jobs, run-to-completion ([MDN](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Execution_model)); `async`/`await` sobre promises | Goroutines no servidor; hooks rodam dentro da transação da alteração (`INTENCAO.md`, "Garantias automáticas") | O run-to-completion é o que dá ao JS ausência de data race dentro de um job. O Germanio oferece garantia equivalente por outro meio (a transação); mas os builtins do nível 3 (`paralelo`, `timeout`) quebram as duas (ver `kotlin.md`) |
| Tipos / null | Dinâmicos; `null` e `undefined` | `obrigatório` | [javascript-problemas.md](javascript-problemas.md) Problema 2 |
| Organização | `package.json`, pastas livres | `backend/`, `frontend/` | — |
| Boilerplate | Baixo na linguagem, alto na configuração de ferramentas (bundler, linter, tsconfig) | Zero configuração | O custo cognitivo do JS moderno está nas ferramentas, não na sintaxe |
| Legibilidade em projetos grandes | Depende de TypeScript e convenções do framework | Formatter canônico, `ge explain` | — |

### "Don't break the web" e o Germanio

O TC39 não pode remover nada porque o código publicado não será atualizado: "many sites
remain unmaintained, owners may lack technical expertise" ([SmooshGate](https://developer.chrome.com/blog/smooshgate)).
O Germanio está na situação oposta, e a norma aposta nisso: "funcionar não justifica
conservar complexidade histórica" (`INTENCAO.md`, "Evolução e refatoração retroativa").
Essa aposta só se sustenta enquanto **o Germanio puder reescrever os `.ge` dos outros**.
Com o lançamento público, passa a haver `.ge` de pessoas que "não têm conhecimento técnico"
(exatamente o público-alvo) e que não vão editar à mão. A condição que torna a remoção
segura é uma ferramenta de reescrita automática verificada (o `ge fmt` já reparseia e
recusa saída que muda o significado, `tooling/formatter/`). Sem ela, o Germanio acaba no
lugar do TC39: sinônimos acumulados para não quebrar ninguém (o léxico já aceita muitos
sinônimos por idioma, `compiler/idiomas/idiomas.go`).

### Permissões por padrão (Deno)

O Deno roda sem acesso a arquivo, rede, ambiente e subprocesso até que uma flag conceda,
com escopo (`--allow-net=example.com`, `--allow-read=./data`). A documentação é honesta
sobre os limites: `--allow-run` e `--allow-ffi` equivalem a acesso total, e não há limite
para código no mesmo nível de privilégio ([Deno security](https://docs.deno.com/runtime/fundamentals/security/)).
No Germanio o equivalente é a separação em camadas: o domínio não fala protocolo; só
`integracoes/` pode. Mas `chamar(url)` existe no interpretador para qualquer hook de nível 3
(`runtime/interpreter/interpreter.go`), com proteção SSRF genérica, não com lista declarada
de destinos (não verificado se existe alguma declaração de hosts permitidos).

---

## Para o Germanio

**ADAPTAR — critério de estabilização por conformidade executável (estilo Stage 4).**
Problema: o Germanio tem dois front-ends (núcleo estrito e dialeto de aplicação), gramática
TextMate gerada e formatter; nada exige que concordem. O TC39 só estabiliza com duas
implementações passando o Test262 ([process](https://tc39.es/process-document/)). Proposta:
uma construção só vira estável quando os exemplos normativos (A15, doctest em
`tooling/doctest/`) passam no parser, no formatter (ida e volta) e na gramática do editor.
Remove da cabeça: "funciona no `ge run` mas o editor pinta errado".

**ADAPTAR — remoção só com reescrita automática verificada.** Toda mudança incompatível de
sintaxe vem com a reescrita (`ge fmt` ou comando equivalente) que converte o `.ge` antigo, e
um teste prova que o significado (`ast.App`) é o mesmo antes e depois. Complementa P7. Arquivo:
`tooling/formatter/`, `docs/INTENCAO.md`. Remove da cabeça: migrar à mão.

**ADAPTAR — capacidades externas declaradas por camada (Deno).** Problema: `chamar` está
disponível a qualquer hook. Proposta: acesso de rede, arquivo e processo só a partir de
`integracoes/` ou de declarações com destino nomeado; o domínio que precisar recebe erro
educativo apontando a camada. `ge explain` lista tudo que o app acessa fora de si. Arquivo:
`runtime/interpreter/interpreter.go`, `runtime/httpclient/`. Remove da cabeça: auditar
hooks à procura de chamadas externas. Evitar a lição negativa do Deno: `-A` como atalho
universal tornaria a permissão cerimônia.

**ADOTAR (já é A16) — uma toolchain no binário.** Deno e Bun existem em boa parte para
eliminar a escolha e a configuração de bundler, formatter, linter, test runner e gerenciador
([Bun](https://bun.com/docs)). A convergência reforça A16 em `GERMANIO_LESSONS.md`.

**EVITAR — sinônimos e nomes piores como forma de compatibilidade.** O `flat` do SmooshGate
é o custo permanente de não poder remover. No Germanio, preferir a reescrita automática à
convivência de formas.

**EVITAR — expor o modelo de fila de eventos.** Microtasks versus tasks é conhecimento que o
autor de JS precisa ter para prever ordem ([MDN](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Execution_model));
no Germanio a ordem observável deve ser a da transação (`antes de` → alteração → `quando`),
e nada mais.
