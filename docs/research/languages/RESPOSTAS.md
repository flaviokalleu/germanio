# Quinze perguntas sobre o Germanio, respondidas com a pesquisa

**Data:** 2026-09-28. **Status:** pesquisa, **sem força normativa** (a norma é
[`docs/INTENCAO.md`](../../INTENCAO.md)). **Código conferido:** revisão `0c12051` (master),
com `go build ./cmd/ge` e arquivos temporários fora do repositório quando o texto diz
"reproduzido".

A regra destas respostas é **procurar onde o Germanio está errado**, não provar que ele é
melhor. Cada resposta separa:

- **IMPLEMENTADO**: existe no código hoje, com arquivo ou teste;
- **PROPOSTO**: está na pesquisa, numa GEP em rascunho ou na norma, mas não no código.

Uma obrigação da norma não é prova de implementação (`INTENCAO.md` › Autoridade). As lições
citadas (`A…`, `P…`, `E…`, `I…`) estão em [GERMANIO_LESSONS.md](GERMANIO_LESSONS.md); a
comparação linha a linha está em [ECOSYSTEM_MATRIX.md](ECOSYSTEM_MATRIX.md); as lacunas `G…`
estão em [`GERMANIO_GAPS.md`](../../../GERMANIO_GAPS.md).

---

## 1. O que o Germanio faz diferente?

**IMPLEMENTADO.**

- **Interpreta o modelo, não gera código.** O resolver produz um modelo semântico da
  aplicação inteira (`ast.App`, `compiler/parser/resolver.go`), e o runtime em Go o executa
  (`runtime/engine.go`, `runtime/servidor/`). Não existe código gerado para o autor manter.
  Os pares de 2026 (IntentLang, Human, Amana) e o Wasp **geram** Node/React
  ([DISCOVERIES.md](DISCOVERIES.md) D1, D2).
- **O runtime é o único autor de SQL, HTML e rotas.** Isso remove classes de erro sem expor o
  mecanismo, que é o que Ur/Web tentou por tipos ([DISCOVERIES.md](DISCOVERIES.md) D6).
- **Duas formas equivalentes da mesma intenção**: a frase plana e o bloco hierárquico
  produzem os mesmos fatos, e há teste de equivalência (`INTENCAO.md` › Sintaxe hierárquica).
- **`ge explain` mostra cada fato com a frase plana e a origem**, e **`ge fmt` recusa a saída
  que muda o significado** (`tooling/explicar`, `tooling/formatter`).
- Nenhuma IA participa da compilação nem da execução.

**Onde a diferença é menor do que parece.** "Determinístico, sem IA, rejeita o ambíguo" já não
é exclusivo: IntentLang diz o mesmo e ainda gera impressão digital semântica e rastreabilidade
fonte→artefato, que o Germanio não tem ([DISCOVERIES.md](DISCOVERIES.md) D2; lição I8). O
diferencial precisa ser demonstrado por aplicações reais e medidas (E34), e hoje há uma
aplicação grande (`examples/gitlab-foss`, 687 linhas, 14 entidades) e nenhum usuário externo
documentado.

## 2. O que aprendemos com linguagens maduras?

A convergência entre várias linguagens é a evidência, não uma linguagem isolada:

| Lição | Quem converge | Estado no Germanio |
|---|---|---|
| diagnóstico é estrutura (código, span, sugestão como edição), não texto | Rust, Gleam, TypeScript, Clang ([rust.md](rust.md), [c.md](c.md)) | PROPOSTO (A1); hoje há o formato de 4 partes em texto (IMPLEMENTADO) |
| todos os erros de uma vez, sem cascata | Zig, Rust, Gleam, Elixir | PROPOSTO (A8); hoje um erro por execução (G69, reproduzido) |
| um front-end para compilador, formatter e editor | TypeScript, Gleam, Roslyn, Zig | PROPOSTO (A6); hoje cinco leitores da sintaxe (G21) |
| concorrência estruturada | Go `errgroup`, Kotlin, Java JEP 505, Trio | PROPOSTO (A21, [GEP 0005](../../gep/0005-concorrencia-por-intencao.md)) |
| evolução por propostas numeradas, com rejeitadas registradas | PEP, RFC, Swift Evolution | IMPLEMENTADO: [GEP 0001](../../gep/0001-processo-gep.md) (Aceita) |
| formatter sem opções | gofmt, zig fmt, Black, dart format | IMPLEMENTADO (`ge fmt`, 4 espaços) |
| deprecação com prazo e migração automática | Elixir, Python PEP 387, Rust editions | PROPOSTO (P7, P26) |
| nada executa na instalação; lockfile com hash | Go modules, Deno, crates.io | PROPOSTO como princípio (A33, [GEP 0006](../../gep/0006-pacotes.md)) |
| uma toolchain num binário | Go, Zig, Gleam, Deno, Bun | IMPLEMENTADO: `main.go` chama `tooling/gecli`; a CLI anterior vive em `ge legado` |

Detalhes em [DIAGNOSTICS.md](DIAGNOSTICS.md), [TOOLING.md](TOOLING.md),
[LANGUAGE_EVOLUTION.md](LANGUAGE_EVOLUTION.md) e [CONCURRENCY.md](CONCURRENCY.md).

## 3. Quais erros históricos estamos evitando?

**Evitados de fato (IMPLEMENTADO):**

- **Macros e DSLs registráveis por bibliotecas** (Elixir `use`, Nim, Scala implicits, Ruby
  monkey patching): a tabela de seções é fechada (`compiler/parser/hierarquia.go`) (E2).
- **Formatter configurável** (rustfmt, swift-format): `ge fmt` não tem opções (E3).
- **Registro de pacotes com scripts de instalação** (npm, worm de 2025): não há registro
  ([PACKAGE_MANAGEMENT.md](PACKAGE_MANAGEMENT.md)).
- **Mass assignment e injeção de SQL por identificador** (Rails/GitHub 2012, CWE-89): corrigidos
  também no dialeto anterior (G98, G100, com testes).
- **Tipos que o usuário precisa entender para ler o erro** (Ur/Web): o nível padrão não tem
  anotações de tipo (E26).
- **Gerar código para o autor manter** (Wasp, scaffolds): o modelo é interpretado (E25).

**Erros históricos que o Germanio está repetindo hoje** (a parte honesta):

- **Duas linguagens sob o mesmo nome**, como Swift com dois parsers e Haskell com extensões:
  o núcleo estrito (`SPEC.md`), a camada de intenção e o dialeto técnico anterior convivem, e o
  mesmo código GE1002 manda "dois espaços" num e a forma canônica é 4 no outro (G71).
- **Tradução palavra a palavra de palavras-chave** (o teto documentado pelo Hedy): o mapa
  global de 20 idiomas transforma "no" em `nao` e "o" em `ou` (G92).
- **Amplitude antes de profundidade** (V): WhatsApp embutido em todo binário antes de o núcleo
  estabilizar (G90).
- **Descartar em silêncio** (JavaScript: rejeição não tratada): a fila `jobs` recusa quando
  cheia e os chamadores ignoram o retorno (`runtime/servidor/servidor.go:976, 989, 1076`).
- **Promessa pública à frente do código** (V): foi corrigido em `bf3cd51` para o README, mas
  `docs/INTENCAO.md:288-289` ainda diz "em qualquer idioma do léxico" enquanto um programa em
  inglês passa no `ge check` com 0 dados (G74, reproduzido).

## 4. Qual é o menor conjunto de conceitos do Germanio?

Conferido em `docs/INTENCAO.md` (seções "Níveis", "Gramática", "Tabela de seções", "O que
existe", "Estados", "Quem pode"). O nível 1 responde quatro perguntas: o que existe, quem pode
fazer o quê, o que deve acontecer, o que deve aparecer.

**Conceitos de domínio do nível 1 (IMPLEMENTADO): 10.**

| # | Conceito | Como aparece | Observação |
|---|---|---|---|
| 1 | **dado** | `tenha clientes` ou o bloco `clientes` | o bloco declara o dado |
| 2 | **campo** | `tem` › `nome obrigatório` | tipo pelo nome; explícito vence |
| 3 | **relação** | `tem pedidos`, `pertence a cliente`, `grupo tem subgrupos` | IDs nunca exigidos |
| 4 | **pessoa** | `autor`, `responsaveis` em `tem` | papel do registro, não do sistema |
| 5 | **papel** | `tenha papeis` › `guest 10` | ordenados; o superior herda |
| 6 | **estado e ação** | `começa aberta`, `pode` › `fechar` | transição; editar não muda estado |
| 7 | **condição** | `pode` › `ser confidencial` | sim/não, sem máquina de estados; a norma a separa de estado |
| 8 | **permissão** | `acesso` › papel › ação; `permita`; `somente`; `seu` | o backend decide, a página só mostra |
| 9 | **regra** | `regras` › `precisa de pelo menos um owner`, `arquivado é somente leitura`, `confidencial pode ser vista por` | invariantes verificadas no runtime |
| 10 | **página** | `página Clientes` › `mostre`, `permita`, `N por página` | o que aparece |

Mais três conceitos de organização: **sistema** (`crie sistema`), **arquivo/importação**
(`importar "backend"`, `backend/` e `frontend/`) e **contexto hierárquico** (o que está abaixo
pertence ao que está acima). **Integração** e **login** são capabilities nomeadas: a
integração é "nível avançado" pela própria norma (`INTENCAO.md` › Integração e
compatibilidade); `tenha login` é uma linha.

Portanto a lista "dado, campo, relação, estado/ação, pessoa/papel, permissão, regra, página,
integração" tem 9 itens, e a norma a refina em **10 conceitos de domínio** (pessoa ≠ papel;
condição ≠ estado) **+ 3 de organização**, com integração fora do nível 1.

**Onde o conjunto é maior do que parece (a parte errada).** Os conceitos são poucos; a
**superfície** que os expressa não é:

- 10 palavras de seção (`tem`, `pertence a`, `começa`, `pode`, `regras`, `acesso`, `permita`,
  `integração`, `quando`, `antes de`);
- 14 construções de sistema listadas na gramática (`tenha papeis`, `tenha login`, `login …`,
  `crie sistema`, `importar`, `vocabulário da integração`, `integração em`, `mensagens em`,
  `escopo`, `ao iniciar`, `tenha busca geral em`, `tenha administrador inicial`, `endereços
  reservados`, `traduza`);
- cerca de 19 modificadores de campo (`obrigatório`, `único`, `privado`, `oculto`, `imutável`,
  `min`, `max`, `até`, `formato`, `valida`, `segredo prefixo`, `expira em`, `revogavel`,
  `opcional`, `como dono`, `por nome`, `por projeto`, `com papel`, `começa com`);
- seis frases de dado especiais na tabela normativa (`recebe aprovações`, `recebe eventos`,
  `executa pipelines … conforme`, `executam jobs`, `repositório pode começar com`, `singular`);
- verbos de várias palavras escritos no parser (`enviar código`, `baixar código`, `branch
  padrão`), que são vocabulário de um produto no core (G70, decisão pendente).

Várias dessas construções nasceram do GitLab e ainda não foram testadas em outro domínio
(exceção: `executam` tem teste com executor sem GitLab, `runtime/trabalho_remoto_test.go`).
É o risco que o C++ documenta: formas que se acumulam sem que uma substitua outra
([cpp.md](cpp.md), lição 3). **PROPOSTO:** `ge check --nivel N` (pendência da norma) e uma
auditoria que classifique cada construção como conceito, sinônimo ou capability de produto.

## 5. Como a sintaxe deve permanecer organizada?

**IMPLEMENTADO:** off-side só com espaços, tab é erro, 4 espaços canônicos, um item por linha,
seções como tabela fechada, fusão de blocos do mesmo dado, formatter sem opções
(`INTENCAO.md` › Layout; `compiler/parser/hierarquia.go`; `tooling/formatter`).

**Onde está errado hoje (reproduzido em `0c12051`):**

- linhas recuadas a mais somem ou viram irmãs (`nome obrigatório` › `email` vira o campo
  `email`; a linha abaixo de `todos` › `ver` desaparece), contra "nenhuma linha é ignorada"
  (G67);
- repetir o mesmo campo em outro bloco é erro, contra "o mesmo fato repetido não muda nada"
  (G68);
- um erro de digitação na primeira seção desfaz o bloco inteiro (`tme` → "não entendi a linha
  \"clientes\"", sem sugestão) (G69).

**PROPOSTO:** schema de seções como dado, com os filhos permitidos por nível (A2); o tipo de
bloco decidido pela estrutura, não pelo conteúdo (A3); fusão idempotente testada por
propriedade (A4, A31); reescrita plana↔hierárquica com equivalência provada (P24); toda
mudança de sintaxe medida contra o corpus de `examples/` (P28). Base:
[SYNTAX_COMPARISON.md](SYNTAX_COMPARISON.md), [sintaxe-e-configuracao.md](sintaxe-e-configuracao.md),
[scala.md](scala.md) (o custo de ter duas sintaxes).

## 6. Como manter simplicidade sem perder poder?

A resposta que a pesquisa sustenta é **assimetria**: complexidade alta dentro do compilador e
do runtime, baixa no `.ge` ([README.md](README.md) › Objetivo). O poder vem de três fontes:

1. **Capabilities genéricas no core, expostas por uma frase** (IMPLEMENTADO: estados e
   transições, papel mínimo, trabalho remoto com lease, repositório Git). A regra que as
   mantém genéricas é o teste em outro domínio (`INTENCAO.md` › Trabalho remoto).
2. **Níveis como subconjuntos da mesma linguagem** (IMPLEMENTADO na norma): o iniciante fica
   no nível 1; o nível 3 (`quando`, `antes de`) e o 4 existem sem contaminar o 1.
3. **Inferência determinística e explicada** (IMPLEMENTADO em parte: `ge explain` mostra a
   origem de cada fato; o motivo de cada inferência é PROPOSTO, P23).

**O que a pesquisa diz para não fazer:** reduzir boilerplate acrescentando mecanismos gerais
(implicits, macros, tipos de ordem superior), que é o caminho do Scala ([scala.md](scala.md));
"mecanismos, não políticas" no domínio, que é a filosofia do Lua e faria cada app montar sua
autorização ([lua.md](lua.md), E29).

**Onde o Germanio perde poder hoje:** transições sem estado de origem (G78), visibilidade só
por condição sim/não (G79), ator restrito a `autor`/`dono`/`criador` (G77), sem permissão por
campo (G97), sem invariante de quantidade (G61). Cada uma é uma capability genérica que falta,
não um motivo para abrir o nível 3.

## 7. Como manter alta performance?

**Medido** ([AUDITORIA.md](../performance/AUDITORIA.md)): em leituras simples o Germanio custa
de 1,2 a 1,6x o Go direto; a página HTML custa 2,5x; nas escritas os dois são limitados pelo
mesmo fsync. **O custo da interpretação não é o problema dominante.** Os fatores grandes são
algorítmicos e de configuração:

| Problema | Número | Lacuna |
|---|---|---|
| visibilidade por registro lê a tabela inteira | 414 ms e 88 MB por página de 20 com 50 000 linhas | G85 |
| log de job regravado inteiro | 45,7 s contra 5,0 s para 4 MiB | G87 |
| fila de tarefas sem índice nem limpeza | 51% de um núcleo ocioso com 1 milhão de concluídas | G87 |
| trava de escrita durante o handler inteiro | p99 de 2,8 s com 16 clientes | G86 |
| hot reload em produção | 21% de um núcleo com 100 000 arquivos | G88 |
| `foldWord` cria um `Replacer` por palavra | 83% da memória do parse | — |

**PROPOSTO** ([ARQUITETURA.md](../performance/ARQUITETURA.md)): **não gerar Go agora**;
evoluir para uma IR executada (planos imutáveis a partir de `ast.App`: colunas projetadas,
statements preparados, visibilidade como predicado SQL, páginas sem ida e volta JSON, lógica
em closures) (P21). Reabrir a decisão só com três critérios medidos (ARQUITETURA §3.3). A
lição do Julia (cachear o resultado determinístico da compilação no artefato) e a do C
(modelo de custo explícito no `ge explain`, P22) apontam na mesma direção.

## 8. Como reduzir memória?

**Medido:** RSS em repouso de 26,9 MB contra 12,4 MB do mesmo app em Go direto; o GitLab
acrescenta só 2 a 3 MB, logo o custo fixo é o binário e o init (213 pacotes; WhatsApp 1,2 MB
antes de `main`) (AUDITORIA §2.2, §4).

**PROPOSTO:**

- **Pague só pelo que usar** como teste (A28, G90): inicializar e embutir só as capabilities
  declaradas; o maior item isolado é o WhatsApp (+9,1 MB de binário).
- **Fim do `map[string]any` por linha** do banco à resposta: projeção e decodificadores por
  coluna nos planos (P21; AUDITORIA 1.3).
- **Leitura com teto** onde hoje não há: `chamar` lê a resposta inteira (`io.ReadAll`,
  `runtime/httpclient/httpclient.go:49`); `git.ReadFile` lê o blob antes de truncar
  (SECURITY_DEFAULTS).
- **Crescimento com o tempo sempre com retenção** (Haskell: previsibilidade por construção,
  [haskell.md](haskell.md)): fila limpa, log por anexação (G87).
- Arena ou `sync.Pool` por request só depois de medir (I23).

## 9. Como escalar concorrência?

**IMPLEMENTADO:** cada request tem escopo próprio filho do global (G22); o trabalho remoto tem
claim atômico, lease, heartbeat, retry, cancelamento visível e token temporário
(`runtime/servidor/trabalho_remoto.go`, testado sem GitLab).

**Errado hoje:** `paralelo` cria uma goroutine por item sem teto; `timeout` devolve nulo e
deixa a goroutine rodando, possivelmente fora da transação; o pânico vira a string `"erro: …"`
(`runtime/interpreter/interpreter.go:1128-1280`; G89, G96). A fila `jobs` descarta. O escritor
único do SQLite segura a trava durante o handler inteiro, sem fila nem `context` até o banco
(G86).

**PROPOSTO** ([GEP 0005](../../gep/0005-concorrencia-por-intencao.md), a partir de
[CONCURRENCY.md](CONCURRENCY.md)): concorrência estruturada como invariante do runtime
(nada sobrevive ao bloco que o criou; prazo cancela; o primeiro erro cancela as irmãs; limite
padrão), backpressure que contém ou recusa e nunca descarta, `context` do request ao banco,
e a intenção escrita em palavras do domínio ("em paralelo", "até 8 ao mesmo tempo", "em
segundo plano", "tente de novo 3 vezes"). Goroutine, canal e mutex nunca aparecem (E20).
**Não resolvido:** o que é seguro paralelizar sob um único escritor (I14).

## 10. Como criar frontend sem expor frameworks?

**IMPLEMENTADO:** as páginas de intenção são HTML completo do servidor, com **zero
JavaScript** e cerca de 4 KB de CSS embutido; a página diz o que aparece e o backend decide
quem pode ([GERMANIO_FRONTEND.md](../frontend/GERMANIO_FRONTEND.md) §10).

**PROPOSTO** (mesma fonte, §1): interface como função do conhecimento do domínio ("UI =
f(dados, permissões, URL)"); melhorias progressivas genéricas do core em camadas (CSS, um
script genérico de troca de regiões, tempo real como aviso + rebusca autenticada); ilhas só
como escape técnico. Nada de componente com estado, hook, VDOM ou hydration no nível padrão.
GEPs em rascunho: [0002](../../gep/0002-secoes-de-pagina.md) (seções de página),
[0003](../../gep/0003-contrato-de-formulario.md) (formulário),
[0004](../../gep/0004-tema-como-tokens.md) (tema).

**Errado hoje** (confirmado no estudo de frontend; G75): branco sobre `azul` tem contraste
3,68:1 (abaixo de 4,5); busca e filtros sem `<label>`; o erro de formulário volta por `?erro=`,
apaga o que foi digitado e é forjável; as páginas de intenção ignoram o tema; os "4 estilos"
descritos antes não existem no código; o SPA do dialeto anterior carrega Tailwind Play CDN,
Chart.js e Google Fonts em execução, com CSP `'unsafe-inline' 'unsafe-eval'`
(`runtime/servidor/servidor.go:238`). O tempo real não tem salas por destinatário (G66).

## 11. Como garantir segurança por padrão?

**A tese:** o runtime é o único autor das camadas perigosas (SQL, HTML, rotas, autorização),
então o autor não tem onde errar. **Ela só vale se todo caminho passar pela mesma defesa.**

**IMPLEMENTADO** (com teste nomeado): senha com bcrypt em todo caminho (G100); CSRF e cookie
`HttpOnly`/`SameSite` nas páginas de intenção; autorização por registro, "não encontrado" em
vez de "proibido"; lista branca de campos graváveis (`writable`); bloqueio de login por
padrão (G83); `/ws` autenticado (G65); pastas servidas sem listagem nem ocultos (G91); ordem e
coluna validadas (G98); proxy com verificação do IP resolvido (G99); upload com CSP `sandbox`
e teto de 64 MB (G101); `ReadHeaderTimeout` de 10 s. A tabela completa está em
[SECURITY_DEFAULTS.md](SECURITY_DEFAULTS.md) (que descreve o estado anterior às correções) e
em [GERMANIO_LESSONS.md](GERMANIO_LESSONS.md) › Divergências.

**Onde ainda está errado:**

- **`chamar` e o cron não usam o cliente seguro.** `runtime/engine.go:396` cria
  `httpclient.Novo()`, um `http.Client` sem verificação do IP resolvido, que segue redirects e
  lê a resposta inteira; ele é entregue ao interpretador (`engine.go:409`) e o cron cria o
  seu (`runtime/cron/cron.go:24`). Lido no código nesta rodada, não reproduzido em execução;
  contradiz a descrição de G59 como DONE (A23).
- o limite por IP no login de intenção falta (G83 IN_PROGRESS);
- quem edita pode alterar qualquer campo declarado (G97);
- o core conhece nomes de sistemas externos (`OPENAI_KEY`, `STRIPE_KEY`,
  `runtime/interpreter/interpreter.go:1303, 1441`), contra a regra das três camadas (P18);
- o dialeto técnico anterior continua no binário; cada correção dele (G98–G101) mostra que
  dois caminhos pedem duas defesas.

**PROPOSTO:** um cliente HTTP, um construtor de consultas e uma lista branca de campos para
todo caminho (A23–A25); acesso externo só por camada (P18); limites visíveis no `ge explain`
(A35).

## 12. Como manter tooling simples?

**IMPLEMENTADO:** um binário `ge` (`main.go` → `tooling/gecli`; a CLI anterior em `ge
legado`); `ge check`, `ge explain`, `ge fmt` (conferência por reparse), `ge test`; o doctest
compila os blocos `ge` dos documentos públicos (`tooling/doctest`), mas **não** os de
`docs/INTENCAO.md` (A15 parcial); gramática TextMate gerada por script.

**Errado hoje:** cinco consumidores leem a sintaxe cada um do seu jeito (`ge check`, `ge
explain`, `ge graph` por regex, `ge fmt` com pilha própria, a gramática do VS Code), e a
tabela de seções tem três cópias no próprio parser ([TOOLING.md](TOOLING.md) §1; G21). Não há
LSP.

**PROPOSTO:** um front-end para todas as ferramentas (A6); `ge lsp` no mesmo binário (A17);
`ge check` como driver de analisadores (P9). A lição mais cara do ecossistema é a de Wasp,
Darklang e Eve: o custo das ferramentas de uma linguagem própria é real e mata projetos
([DISCOVERIES.md](DISCOVERIES.md), padrão 1). Por isso: nada de editor próprio (E24), e o
mínimo de ferramentas para o nível 1 ainda precisa ser orçado (DISCOVERIES I3).

## 13. Como evoluir a linguagem sem quebrá-la?

**IMPLEMENTADO:** o processo de GEP (template, status fechados, números nunca reutilizados,
implementação antes de aceitação; [GEP 0001](../../gep/0001-processo-gep.md) Aceita); o
formatter conferindo que o significado não muda; todo `.ge` conhecido está no repositório, o
que permite migrar na mesma unidade de trabalho (E17).

**PROPOSTO:** deprecação com prazo escrito e migração verificada (P7); remoção só com
reescrita automática (P26); checagem nova primeiro como aviso (P25); versão da linguagem
declarada no projeto, como a linha `go` do `go.mod` (I7; hoje os modelos de `germanio init`
não registram versão, conforme [ruby.md](ruby.md)); decisões de sintaxe medidas contra o
corpus (P28). Base: [LANGUAGE_EVOLUTION.md](LANGUAGE_EVOLUTION.md), [java.md](java.md)
(JEP 12), [kotlin.md](kotlin.md) (KEEP), [csharp.md](csharp.md) (NRT), [lua.md](lua.md)
("mais fácil acrescentar depois que remover").

**Risco:** quando existir o primeiro `.ge` fora do repositório, todas as quebras atuais
(inclusive as correções de G67 e G68, que tornam erro o que hoje passa) precisarão de prazo.

## 14. Quais ideias foram rejeitadas e por quê?

Rejeitadas na pesquisa (nenhuma GEP foi rejeitada formalmente ainda;
[`docs/gep/README.md`](../../gep/README.md) › Rejected ideas está vazio):

| Ideia | Por quê | Onde |
|---|---|---|
| gerar Go a partir do `.ge` agora | ganho medido pequeno (1,2–1,6x); duas semânticas para manter iguais; exige toolchain Go do usuário | [ARQUITETURA.md](../performance/ARQUITETURA.md) §3.1 |
| macros, DSLs e seções registráveis por bibliotecas | quebra `ge check`/`ge explain` determinísticos | E2 |
| formatter configurável | cada opção é um formato a mais | E3 |
| Hindley-Milner, type classes | nenhum problema do nível padrão pede | E11 |
| PEG com escolha ordenada | a ordem das alternativas vira semântica escondida | E9 |
| editor ou IDE próprios | a parte mais cara e menos amada de Eve e Darklang | E24 |
| coloração de função e primitivas de concorrência no `.ge` | conceitos que a norma proíbe | E20 |
| registro de pacotes com scripts de instalação, "nearest wins" | vetor de ataque; não determinístico | E21, E22 |
| modos e flags que mudam o significado | "este programa roda com qual modo?" | E17, E27 |
| gerar código para o autor manter | traz de volta o que o Germanio remove | E25 |
| tradução palavra a palavra para línguas de outra ordem de frase | não traduz `developer pode enviar código dos projetos` | E23 |
| duas sintaxes concretas para o mesmo modelo (estilo HCL JSON) | a forma canônica já existe | E19 |

## 15. Quais problemas ainda não sabemos resolver?

1. **Se quem nunca programou consegue de fato escrever `.ge`.** Não há estudo com usuários.
   Todos os projetos parecidos concluíram que os primeiros usuários eram programadores (Eve,
   Wasp; [DISCOVERIES.md](DISCOVERIES.md) padrão 4). O Germanio mira o leigo sem evidência de
   que ele chega.
2. **Idiomas de outra ordem de frase.** Ninguém resolveu (Hedy registra como problema aberto);
   a GEP 0007 propõe não prometer o que não se sabe fazer
   ([GEP 0007](../../gep/0007-idiomas-da-intencao.md)).
3. **Tempo real com permissões por espectador**, sem difundir dados e sem rebuscar tudo para
   cada espectador (G66; I19).
4. **O que paralelizar sob um único escritor**, e a ordem dos resultados (I14).
5. **Renomear um campo sem perder dados** sem pedir SQL: o diff de esquema detecta, mas a frase
   que desfaz a ambiguidade ainda não existe (P16, G93).
6. **Regras de visibilidade que o SQL não expressa** (`antes de ver` com lógica arbitrária):
   quanto ler e como avisar o autor (ARQUITETURA §3.2).
7. **Quanto tooling é o mínimo** para o nível 1 e quanto custa (Wasp chegou a 80% com cinco
   anos e US$ 5 milhões; DISCOVERIES I3). Não há LSP nem ambiente sem instalação (I12).
8. **Onde o valor traduzido pelo mapa de idiomas vaza** fora do caminho de intenção testado
   (I17).
9. **Se a superfície do nível 1 cresce mais rápido que os conceitos** (pergunta 4): não há
   métrica nem teste que acuse uma construção nova que não substitui nenhuma.
10. **Se os blocos `.ge` compartilháveis são necessários** (I16) e, portanto, se um gerenciador
    de pacotes algum dia será (GEP 0006).
