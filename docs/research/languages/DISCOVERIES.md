# Descobertas do ecossistema de linguagens

**Data:** 2026-09-28. **Status:** pesquisa, sem força normativa (ver [README.md](README.md)).
**Fase:** 16 (descobertas inesperadas), a partir da triagem em
[ECOSYSTEM_TRIAGE.md](ECOSYSTEM_TRIAGE.md).

Fontes consultadas (além dos READMEs listados na triagem):

- Wasp, "5 Years and $5M Later: Inventing a New Programming Language for Web Development Was a
  Mistake" (2026-05-13): <https://wasp.sh/blog/2026/05/13/new-language-for-web-dev-was-a-mistake>
- Wasp, "Wasp now lets you write your full-stack logic as a spec in TypeScript" (2026-06-15):
  <https://wasp.sh/blog/2026/06/15/wasp-typescript-spec>; docs: <https://wasp.sh/docs/general/spec>
- Chris Granger, "Two years of Eve" (2016): <https://chris-granger.com/2016/07/21/two-years-of-eve/>;
  anúncio do encerramento: <https://groups.google.com/g/eve-talk/c/YFguOGkNrBo/m/EozaCfheAQAJ>
- Darklang, "Darklang is going all-in on AI": <https://blog.darklang.com/gpt/>;
  "An overdue status update": <https://blog.darklang.com/an-overdue-status-update/>
- Hermans et al., "Design, implementation and evaluation of the Hedy programming language"
  (2022): <https://hedy.org/research/Design_Implementation_and_evaluation_of_the_Hedy_programming_language_2022.pdf>
- Swidan e Hermans, "A Framework for the Localization of Programming Languages" (SPLASH-E
  2023): <https://dl.acm.org/doi/10.1145/3622780.3623645> (o PDF recusou acesso; só o
  título e os autores foram verificados)
- Chlipala, "Ur/Web: A Simple Model for Programming the Web" (POPL 2015):
  <http://adam.chlipala.net/papers/UrWebPOPL15/UrWebPOPL15.pdf> (lido via resumo de busca;
  detalhes marcados "não verificado")
- Opa: <https://en.wikipedia.org/wiki/Opa_(programming_language)> e o README do repositório

Convenção: **IMPLEMENTADO** = existe no Germanio hoje (com arquivo); **PROPOSTO** = não existe.

---

## D1. Wasp abandonou a própria linguagem depois de cinco anos

**Repositório:** <https://github.com/wasp-lang/wasp> (18 755 estrelas, ativo).

**Problema que resolve.** Aplicações web completas (React, Node.js, Prisma) sem boilerplate: um
arquivo de especificação declara app, rotas, páginas, autenticação, queries, jobs; o
compilador gera e orquestra o resto (README).

**O que aconteceu.** Em 2026 o Wasp trocou a DSL `main.wasp` por uma especificação em
TypeScript (`main.wasp.ts`, com construtores `app`, `route`, `page`, `query`). Os autores
escreveram que inventar a linguagem "was more trouble than it was worth" e que ela passou a
bloquear o crescimento do projeto. Motivos que eles dão: o ecossistema de ferramentas é feito
para JavaScript/TypeScript e "anything else, and you're on your own"; construíram LSP e extensão
do VS Code próprios e chegaram a ~80% do que queriam; o sufixo "lang" fazia as pessoas acharem
que o Wasp substituía o JavaScript; o difícil não era reter usuários, era fazê-los
experimentar; e "the ergonomics we aimed for with the language didn't turn out to be as
important as we thought". Pediram ainda: dividir a especificação em vários arquivos e
importar entre eles.

**Ideia interessante (o que eles dizem que estava certo).** O valor estava em a ferramenta ter
"a full understanding of their entire app via a high-level spec", e o domínio de aplicações
web muda devagar (páginas, rotas, endpoints, modelos) enquanto as técnicas de implementação
mudam depressa. A especificação sobreviveu; a sintaxe própria não.

**Trade-off.** Uma sintaxe própria dá ergonomia e controle; custa LSP, realce, formatter,
diagnósticos, documentação e a barreira de "mais uma linguagem". Para programadores que já
vivem em TypeScript, o custo não se pagou.

**Aplicabilidade ao Germanio.** É o alerta mais forte desta pesquisa, e ele vale só em parte.
O Wasp mira *programadores JavaScript*: para eles, TypeScript não é conceito novo. O Germanio
mira quem **nunca programou**; para esse público TypeScript é o custo, não a economia, e a
sintaxe própria é o produto. O que **vale integralmente**: (a) o custo de ferramentas de uma
linguagem própria é real e o Wasp não o pagou nem com US$ 5 milhões; o Germanio hoje tem
gramática TextMate gerada (`vscode-germanio/tools/gerar_gramatica.py`), `ge fmt`
(`tooling/formatter/`) e `ge explain` (`tooling/explicar/`), e **não tem LSP** (IMPLEMENTADO /
ausência conferida em [TOOLING.md](TOOLING.md)); (b) o valor está no *modelo semântico da
aplicação inteira* (`ast.App`, `compiler/parser/resolver.go`), não na sintaxe; (c) a
especificação precisa caber em vários arquivos (o Germanio já funde blocos do mesmo dado —
IMPLEMENTADO em `resolver.go`).

---

## D2. O nicho do Germanio ficou cheio em 2026, e os pares fazem a mesma promessa

**Repositórios:** IntentLang <https://github.com/sethiramicrosoft/intentlang> (TypeScript,
criado em 2026-07), Human <https://github.com/barun-bash/human> (Go, 2026-02), Amana
<https://github.com/akleeko2/amana-dsl> (Rust, 2026-06), mais Theia, Jounce, UniStack, Zap,
madilang (triagem, seção 4). Todos com menos de 10 estrelas.

**Problema que resolvem.** O mesmo do Germanio: descrever dados, relações, regras, papéis e
fluxos e obter a aplicação completa.

**Ideias interessantes.**

- **IntentLang** diz explicitamente "This is not prompt-to-code… Accepted sentences have
  defined semantics; unsupported or ambiguous instructions fail explicitly" e "The English is
  the source code. The compiler is the authority." Gera, além da aplicação, "canonical source,
  semantic fingerprints, manifests, and source-to-artifact traceability"; tem permissões com
  escopo de dono (`allow Member to update Task where owner is self`), máquinas de estado,
  idempotência, concorrência otimista e registros de auditoria gerados; um "App Builder" que
  "reports unsupported requests instead of silently improvising"; e casos de estudo com 155 e
  171 permissões declaradas.
- **Human** tem uma "Intent IR" tipada e serializável entre a fonte e os geradores e várias
  saídas (React/Angular/Vue/Svelte; Node/FastAPI/Gin); declara testes e auditoria de segurança
  "compiler-enforced"; e se diz "LLM-optional".
- **Amana** gera Express + SQLite com CSRF, Argon2, autorização por papel/linha/campo e tem
  "IR snapshots, JSON diagnostics, a formatter, and an LSP"; trata RTL/árabe na interface.

**Trade-off.** Os três **geram código** de terceiros (Node, React…), que o usuário passa a
manter ou "ejetar"; o Germanio **interpreta** o modelo no próprio runtime Go
(`runtime/engine.go`, `runtime/servidor/`) e entrega um binário (`germanio build`). Geração dá
portabilidade e saída legível; interpretação dá um único artefato e nenhum código gerado para
manter.

**Aplicabilidade ao Germanio.** (a) A tese "sem IA, determinístico, rejeita o ambíguo" deixou
de ser diferencial por si só; o diferencial precisa ser demonstrado (aplicações reais, como o
exemplo `examples/gitlab-foss/`, e medidas). (b) IntentLang tem duas coisas que o Germanio não
tem como produto: **impressão digital semântica** da fonte (dois arquivos com o mesmo
significado têm a mesma impressão) e **rastreabilidade fonte→artefato** navegável. O Germanio
tem o núcleo disso: `ge explain` mostra cada fato com a frase plana e a origem arquivo:linha
(IMPLEMENTADO, `tooling/explicar/`), e o formatter recusa saída que muda o significado
(IMPLEMENTADO, `tooling/formatter/`). (c) Nenhum dos pares declara palavras-chave em mais de um
idioma nas palavras-chave (Amana localiza a interface, não a sintaxe).

---

## D3. Hedy mostra que localizar palavras-chave é fácil e localizar a linguagem não é

**Repositório:** <https://github.com/hedyorg/hedy> (1 690 estrelas; 54 idiomas; ~500 mil
usuários mensais segundo a apresentação no ICPEC 2024, não verificado diretamente).

**Problema que resolve.** Ensinar programação a crianças por níveis graduais (o nível 1 não tem
aspas, parênteses nem indentação) e na língua delas.

**Ideia interessante.** No estudo de 2022, os alunos holandeses pediram palavras-chave em
holandês; a implementação troca cada token por uma disjunção na gramática (`"print" |
"imprimir"`), e **permite o inglês e um idioma nativo**, para bilíngues. Os autores registram
dois problemas abertos: (1) "truly localizing a language entails more than just keywords":
em holandês a condição tem outra ordem de palavras ("[if] number 5 is"), em árabe não há verbo
"ser" e `x is 5` não tem tradução direta, e aspas e vírgulas mudam por idioma (« », a vírgula
árabe); (2) a "deslocalização" gradual quando o objetivo é chegar ao Python.

**Trade-off.** Tradução por palavra funciona para idiomas com a mesma ordem de frase do
inglês; quebra onde a gramática difere. Misturar idiomas amplia o vocabulário reservado.

**Aplicabilidade ao Germanio (onde o Germanio está errado).** O Germanio aceita 20 idiomas
**no mesmo arquivo** por um único mapa global palavra→palavra-chave canônica
(`compiler/idiomas/idiomas.go`, `Translations`, consultado em `compiler/lexer/lexer.go`
linhas 762-773). Medido nesta pesquisa com o lexer do dialeto de aplicação (`lexer.New`, usado
por `runtime/engine.go` e `tooling/formatter/intencao.go`):

| Entrada | Tokens produzidos (valor canônico / grafia original) |
|---|---|
| `campo no projeto` | `campo`, **`nao`/`no`**, `projeto` |
| `o nome` | **`ou`/`o`**, `nome` |
| `tarefa tem estado` | `tarefa`, `tem`, **`status`/`estado`** |
| `don ham kazi` | `retornar`, `funcao`, `funcao` (turco, vietnamita, suaíli) |

Isto é, palavras comuns do português ("no", "o") viram palavras-chave de outros idiomas
(espanhol `no`→`nao`, `o`→`ou`). No caminho de intenção a colisão foi contida no caso testado:
`tarefas / tem / estado` resultou no campo `estado` (o parser usa a grafia original,
`Raw`), confirmado por `ge explain tarefa`. Não foi verificado em que outros pontos o valor
canônico é usado no lugar do `Raw`; os exemplos reais usam "o" e "estado" fora de aspas
(`examples/gitlab-foss/backend/grupos/grupos.ge:22`,
`examples/gitlab-foss/backend/identidade/usuarios.ge:57`). Além disso, a tradução por
palavra assume a ordem de frase do português, que o próprio Hedy mostrou não servir para
holandês e árabe.

---

## D4. O que aconteceu com as linguagens em português e em outros idiomas

**Repositórios:** Portugol Studio <https://github.com/UNIVALI-LITE/Portugol-Studio> (764,
último push 2023-04), Portugol Webstudio <https://github.com/dgadelha/Portugol-Webstudio>
(375, ativo), Potigol <https://github.com/potigol/potigol> (268), Égua
<https://github.com/eguadev/egua> (850, ativo), wenyan <https://github.com/wenyan-lang/wenyan>
(20 279, parado desde 2023-10), Kip <https://github.com/kip-dili/kip> (880).

**O padrão.** As linguagens em português que duram são **didáticas e vivem dentro de um
ambiente** (IDE no navegador): Portugol sobreviveu porque o Portugol Studio saiu e o Webstudio
continuou; Égua oferece "IDEgua" no navegador "sem instalar nada" e se declara "voltada ao
ensino"; Potigol tem mais de 800 problemas resolvidos como material. Nenhuma virou ferramenta
de produção. As chinesas se dividem em curiosidades de enorme atenção e pouca continuidade
(wenyan: 20 mil estrelas, dois anos sem push) e muitos projetos de 0 a 2 estrelas; a busca
`topic:non-english-programming-language` devolve **zero** repositórios.

**Ideia interessante.** Kip é o único que trata a língua como estrutura, não como
vocabulário: os casos gramaticais do turco decidem o papel dos argumentos, e a ordem dos
argumentos fica livre ("As long as arguments have different case suffixes or different types,
Kip can determine which argument is which"); declara-se experimental.

**Trade-off.** Idioma nativo baixa a barreira de entrada; tira o aluno do caminho das
ferramentas, dos exemplos e das respostas em inglês, e divide uma comunidade pequena por
idioma.

**Aplicabilidade ao Germanio.** O público do Germanio (quem nunca programou, no Brasil)
coincide com o do Portugol, e o Portugol mostra que **o ambiente importa tanto quanto a
sintaxe**: uma pessoa que nunca programou não instala Go. A frase em português do Germanio é
mais próxima de Kip (a preposição/relação carrega significado: `pertence a`, `pode … dos
projetos`) do que de Portugol (tradução de `if/while`).

---

## D5. Eve: o que Chris Granger aprendeu depois de 30 protótipos

**Repositório:** <https://github.com/witheve/Eve> (7 223 estrelas; "no longer under active
development"; encerrado em 2018).

**Problema.** "Human-first programming": tornar programação acessível a quem não é
programador. Financiado (US$ 2,3 milhões, segundo o ensaio da Future of Coding, não lido
diretamente).

**O que aconteceu.** Em dois anos, mais de 30 protótipos (exploradores gráficos de banco,
documentos com consultas embutidas, consultas em linguagem natural). Conclusão do autor: "we
can't just slap a UI onto Javascript and expect it to work; the platform has to allow for the
representation"; separaram a plataforma relacional da interface e lançaram uma sintaxe
**textual** para desenvolvedores primeiro, porque "the first users of general tools like this
tend to be developers". O projeto terminou em 2018 sem comprador; Granger disse que é difícil
justificar uma linguagem nova antes de ela estar pronta (fio de encerramento).

**Ideia interessante.** Modelo relacional único (registros e blocos que reagem a padrões) em
vez de variáveis e chamadas; programação literária.

**Trade-off.** Refazer a computação por baixo dá coerência; adia indefinidamente o momento
em que alguém constrói algo real.

**Aplicabilidade.** O Germanio fez a escolha que Eve acabou fazendo: texto primeiro, modelo
semântico próprio por baixo (`ast.App`), sem editor visual. A lição de risco é a de Eve:
nenhuma quantidade de pesquisa substitui aplicações reais em produção.

---

## D6. Opa, Ur/Web e Links: a promessa "uma linguagem para todas as camadas"

**Repositórios:** Opa <https://github.com/MLstate/opalang> (último push 2020), Ur/Web
<https://github.com/urweb/urweb> (último push 2024-05), Links
<https://github.com/links-lang/links>.

**Problema.** O "impedance mismatch": uma aplicação web mistura servidor, HTML, JavaScript e
SQL, e "there is no easy way to link these" (README do Links). Os três compilam um único
programa para cliente, servidor e banco.

**Ideias interessantes.** Ur/Web promete, por tipos, que a aplicação não sofre injeção de
código, não gera HTML inválido, não tem links internos mortos, não tem formulário
incompatível com o tratador e não tenta SQL inválido (README), sem coletor de lixo. Links
traduz parte do código para SQL e tem sistema de efeitos para consultas.

**O que aconteceu.** Opa: criado em 2011, última versão estável em 2014; a empresa passou a
usar Opa em produto próprio, documentação e blog sumiram, e a experiência de instalação era
ruim (Wikipedia e discussões citadas nela). Ur/Web: tem usuários em produção, mas o ponto de
dor mais citado são **mensagens de erro que confundem até programadores experientes**, vindas
da análise que separa código de cliente e de servidor (paper do POPL 2015; não verificado no
texto integral). Links continua como pesquisa.

**Trade-off.** Garantias fortes pagas com um sistema de tipos que o usuário precisa entender
para ler os erros.

**Aplicabilidade.** O Germanio promete as mesmas garantias por outro caminho: não por tipos
que o usuário escreve, mas porque o **runtime é o único autor** de HTML, SQL e rotas
(`runtime/servidor/renderizador.go`, `runtime/banco/banco.go`), e as operações nascem do
modelo. Isso remove a classe de erro sem expor o mecanismo. O risco herdado de Ur/Web é o
diagnóstico: quando a garantia falha, o erro tem de falar a língua do domínio (o formato de 4
partes do Germanio, `compiler/diagnostics`, IMPLEMENTADO).

---

## D7. Darklang: linguagem + editor + infraestrutura, e o editor morreu

**Repositório:** <https://github.com/darklang/dark> (2 169 estrelas; "dark-next" não pronto
para produção; a versão de produção fica em `darklang/classic-dark`).

**Problema.** Backends sem infraestrutura: escrever código e ele já está no ar ("deployless"),
desenvolvimento guiado por traces de requisições reais.

**O que aconteceu.** Em 2023 removeram o editor estruturado próprio (cerca de 50 mil linhas
de ReScript), que os usuários avaliavam entre "Ok I guess" e "probably the worst part of
Darklang", porque ele estava desconectado de onde as pessoas programam; abandonaram a
compatibilidade retroativa para destravar a linguagem; depois reorientaram o produto para
código gerado por IA.

**Ideia interessante.** Traces: cada requisição real fica disponível como exemplo para
depurar e testar. "Deployless" como propriedade da plataforma, não do usuário.

**Trade-off.** Tudo integrado dá experiência coesa e prende o usuário ao fornecedor; o editor
próprio foi o componente mais caro e o menos amado.

**Aplicabilidade.** Confirma D1 por outro ângulo: ferramentas proprietárias (editor, IDE) são
o maior custo de uma linguagem nova. A hot reload do Germanio (`runtime/hotreload.go`,
IMPLEMENTADO) já dá parte do "sem deploy" local.

---

## D8. Elm/Lamdera, Catala e Wing: três formas de tirar coisas da cabeça do usuário

- **Elm + Lamdera** (<https://github.com/elm/compiler>, <https://github.com/lamdera/compiler>).
  Elm é "a delightful language for reliable webapps"; Lamdera estende o compilador para
  front-end *e* back-end tipados, com a comunicação entre eles gerada ("Lamdera Wire") e
  `lamdera live` com recarga. O modelo de negócio é a plataforma paga que financia o
  compilador aberto. Lição: o fio entre cliente e servidor é derivado, não escrito — o que o
  Germanio já faz ao gerar a API a partir do modelo.
- **Catala** (<https://github.com/CatalaLang/catala>): cada trecho de lei é anotado com seu
  significado em código ("literate programming"), para que juristas revisem a fidelidade
  código↔lei. Lição: o público especialista **revisa**, não escreve; a fonte é legível por quem
  valida a regra. Para o Germanio: a frase plana de `ge explain` é o equivalente —
  o gestor lê "developer pode enviar código dos projetos" e confirma.
- **Wing** (<https://github.com/winglang/wing>): infraestrutura e aplicação no mesmo programa,
  com duas fases ("preflight" gera a infraestrutura na compilação, "inflight" roda) e um
  **simulador local** completo ("no internet required"). Lição: a distinção entre o que é
  decidido na compilação e o que roda é explícita e o teste local não depende de nuvem.

Apoio: **Unison** (<https://github.com/unisonweb/unison>) guarda código endereçado pelo
conteúdo, o que dá renomeação sem quebra e cache perfeito de testes determinísticos;
**Ballerina** (<https://github.com/ballerina-platform/ballerina-lang>) alterna entre desenho
visual e código sobre a mesma fonte ("low code and pro code").

---

## Padrões recorrentes

1. **Linguagens full-stack declarativas morrem pela periferia, não pelo núcleo.** Opa (setup
   ruim, documentação que sumiu), Ur/Web (mensagens de erro), Eve (nada construído a tempo),
   Darklang (editor próprio), Wasp (LSP e extensão a 80%, "mais uma linguagem"). Em nenhum caso
   o motivo registrado foi "a ideia de descrever a aplicação estava errada"; Wasp afirma o
   contrário. O que mata é o custo total: instalação, editor, erros, documentação, ecossistema,
   e a pergunta "isso vem com ecossistema próprio?".
2. **A especificação da aplicação sobrevive; a sintaxe é a parte descartável.** Wasp manteve
   `route`, `page`, `query`, `auth`; Eve manteve o modelo relacional e trocou a interface;
   Darklang manteve deployless e traces e jogou fora o editor.
3. **Quem conquista não-programadores vive dentro de um ambiente pronto.** Hedy, Portugol
   Webstudio, IDEgua, Scratch: navegador, zero instalação. Os projetos em CLI para iniciantes
   não aparecem entre os que duram.
4. **Público-alvo declarado e público real divergem.** Eve e Wasp concluíram que os primeiros
   usuários são desenvolvedores; Catala é escrita por programadores e *revisada* por juristas.
5. **Localização por troca de palavras é um teto baixo.** Hedy documenta a ordem das palavras e
   a pontuação como problemas não resolvidos; os projetos chineses e portugueses que só
   traduzem palavras-chave viram material didático ou curiosidade.
6. **2026 trouxe concorrência direta e uma nova justificativa.** Os pares (IntentLang, Human,
   Amana) e os novos do tópico (BAML, Vera, Sui) se justificam por IA: "especificação
   determinística que a IA escreve e o humano revisa". Wasp e Darklang também passaram a se
   descrever como feitos "for the AI era".

**O que o Germanio deve aprender disso.** O núcleo (modelo semântico da aplicação inteira,
runtime como único autor das camadas) é a parte que todos confirmam; a periferia (instalação,
editor, erros, documentação, exemplos reais) é a parte que mata. O Germanio já investiu na
periferia certa (diagnósticos em 4 partes, `ge fmt` com verificação por reparse, `ge explain`,
binário único) e ainda não tem as duas que mais pesaram nos casos estudados: **LSP** e **um
ambiente sem instalação** para quem nunca programou.

---

## Para o Germanio

### ADOTAR

- **A1 — Palavras-chave de um idioma por arquivo, não de 20 misturados** (Hedy, D3).
  *Problema:* o mapa global de `compiler/idiomas/idiomas.go` transforma "no" em `nao`, "o" em
  `ou` e "estado" em `status` em texto português (medido; D3). *Proposta:* o idioma do arquivo
  é o português canônico mais **um** idioma declarado ou inferido do projeto; as colisões
  entre esse idioma e o português viram erro de construção da tabela, testado. *Arquivos:*
  `compiler/idiomas/idiomas.go`, `compiler/lexer/lexer.go`. *Remove da cabeça:* a lista de
  palavras de 19 idiomas estrangeiros que o usuário não pode usar como nome sem saber.
  (PROPOSTO; muda a promessa "todos intercambiáveis no mesmo arquivo" do CLAUDE.md — exige
  decisão deliberada.)
- **A2 — Teste de colisão entre o léxico e palavras comuns do português.** *Problema:* nenhum
  teste impede que uma entrada nova do mapa capture uma preposição ou artigo. *Arquivo:* testes
  de `compiler/idiomas/`. *Remove da cabeça:* surpresas como "campo no projeto" virar negação.
  (PROPOSTO)
- **A3 — Tratar o modelo semântico como o produto** (Wasp, D1). A documentação e o `ge
  explain` devem mostrar primeiro o modelo (`ast.App`), porque é o que os usuários de Wasp
  diziam valorizar. *Arquivos:* `tooling/explicar/`, `docs/`. *Remove:* a necessidade de ler a
  sintaxe para entender a aplicação. (Parcialmente IMPLEMENTADO em `ge explain`.)

### ADAPTAR

- **P1 — Impressão digital semântica e rastreabilidade fonte→artefato** (IntentLang, D2).
  O Germanio já tem a origem de cada fato (`ge explain`) e a equivalência hierárquico↔plano
  testada; falta um hash estável do `ast.App` normalizado (mesmo significado, mesmo hash),
  útil para revisão de mudanças e cache. *Arquivos:* `compiler/parser/resolver.go`,
  `tooling/explicar/`. *Remove:* "essa mudança de formatação alterou algo?". (PROPOSTO)
- **P2 — Ambiente sem instalação para o nível 1** (Portugol Webstudio, Hedy, IDEgua; D4). O
  runtime é Go puro sem CGO, o que torna viável (INVESTIGAR o custo) compilar parser + `ge
  check` + `ge explain` para WebAssembly e rodar no navegador. *Remove:* instalar Go, terminal
  e caminhos de arquivo antes da primeira aplicação. (PROPOSTO)
- **P3 — Traces de requisições reais como exemplos** (Darklang, D7). Em modo de
  desenvolvimento, guardar as últimas requisições por operação e exibi-las em `ge explain`.
  *Arquivos:* `runtime/servidor/`. *Remove:* reproduzir à mão o que o usuário fez para
  entender um erro. (PROPOSTO)
- **P4 — Separação explícita do que é decidido na verificação e do que roda** (Wing, D8).
  *Arquivo:* `docs/INTENCAO.md` (documentação, não sintaxe). (PROPOSTO)

### EVITAR

- **E1 — Editor ou IDE proprietários** (Darklang, Eve, D5/D7). Foram o componente mais caro e
  menos amado. O caminho é LSP sobre o front-end existente ([TOOLING.md](TOOLING.md)).
- **E2 — Garantias que dependem de o usuário entender um sistema de tipos** (Ur/Web, D6).
  As garantias do Germanio vêm de o runtime ser o único autor; mantenha assim.
- **E3 — Tradução palavra a palavra como estratégia para idiomas de outra ordem de frase**
  (Hedy, D3). Árabe, japonês, coreano, turco, hindi e bengali não compartilham a ordem das
  frases do português; frases como `developer pode enviar código dos projetos` não se traduzem
  por substituição. Declarar suporte a esses idiomas sem gramática própria promete o que a
  implementação não entrega.
- **E4 — Gerar código de terceiros para o usuário manter** (Human, Amana, Wasp). O Germanio
  interpreta o modelo; um "eject" traria de volta tudo o que ele remove.
- **E5 — Competir em "sem IA" como diferencial único** (D2). Três pares recém-criados dizem o
  mesmo; o diferencial precisa vir de aplicações reais e medidas.

### INVESTIGAR

- **I1 — Onde o valor canônico traduzido (`Value`) é usado no lugar da grafia (`Raw`)** fora
  do caminho de intenção testado (D3). `runtime/engine.go` e `runtime/servidor/servidor.go`
  usam `lexer.New`; é preciso um teste com as palavras colidentes em `logica`, `telas` e
  `eventos`.
- **I2 — Quantos idiomas o Germanio deveria prometer.** Hedy chegou a 54 com uma comunidade de
  tradutores (Weblate) e ainda considera a localização incompleta. INVESTIGAR com usuários
  reais se algum público usa hoje os 19 idiomas não portugueses.
- **I3 — Se o custo de ferramentas (LSP, realce, formatter, docs) está orçado.** Wasp chegou a
  80% com cinco anos e US$ 5 milhões; o Germanio precisa saber qual o mínimo para o público de
  nível 1.

### O que isto remove da cabeça do programador

- A1/A2: a lista invisível de palavras de outros idiomas que viram palavras-chave.
- P1: a dúvida "o que mudou de fato" entre duas versões da fonte.
- P2: toda a instalação antes da primeira aplicação.
- P3: a reprodução manual de um erro de uso real.
- E2/E4: sistemas de tipos e código gerado — o usuário continua sem precisar saber que existem.
