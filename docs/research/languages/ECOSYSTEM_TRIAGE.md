# Triagem do ecossistema de linguagens (GitHub `topic:programming-language`)

**Data:** 2026-09-28. **Status:** pesquisa, sem força normativa (ver [README.md](README.md)).
**Fase:** 1 (mapear o ecossistema). As descobertas e as lições estão em
[DISCOVERIES.md](DISCOVERIES.md).

Fontes: ponto de partida <https://github.com/topics/programming-language>, consultado pela API
de busca do GitHub (`gh api search/repositories`), os READMEs dos projetos selecionados
(`gh api repos/<dono>/<repo>/readme`) e as fontes primárias citadas em DISCOVERIES.md. Nenhum
repositório foi clonado.

## 1. Metodologia

1. **Coleta.** Buscas pela API, não pelo HTML da página do tópico. Cada busca devolve no
   máximo 100 itens por página; a API limita qualquer busca a 1000 resultados, por isso as
   contagens totais (`total_count`) são registradas separadamente dos itens lidos.
2. **Extração.** Só nome, estrelas, linguagem de implementação, data do último push,
   `archived` e descrição (`--jq`), em TSV.
3. **Triagem.** Classificação em categorias (seção 4) pela descrição e, para os candidatos,
   pelo README. Um projeto pode estar em mais de uma categoria. Repositórios que não são
   linguagens (livros, listas "awesome", tutoriais) foram separados.
4. **Seleção.** Critérios da seção 5; os projetos em Go, Python, Rust, Swift, Kotlin,
   TypeScript, JavaScript, C, C++, C#, Java, Ruby, Elixir, Gleam, Nim, Zig, V, Crystal, Julia,
   Lua, Dart, Haskell, Scala e Roc foram excluídos da seleção (já estudados ou fora do escopo
   desta fase). A exclusão é da *linguagem estudada*, não da linguagem de implementação: Wasp
   é implementado em Haskell/TypeScript e entra.
5. **Leitura.** README de cada selecionado e, quando a história do projeto importa (projetos
   que morreram ou mudaram de rumo), o texto dos próprios autores.

Limite: as estrelas medem atenção, não uso. As contagens por categoria da seção 3.3 vêm de
expressões regulares sobre as descrições e servem só como ordem de grandeza.

## 2. Comandos

`J` é o filtro comum:

```sh
J='.total_count as $t | .items[] | [$t, .full_name, .stargazers_count, (.language//"-"),
   (.pushed_at[:10]), .archived, ((.description//"")[:140])] | @tsv'
gh api -X GET search/repositories -f q='topic:programming-language' -f sort=stars -f per_page=100 -f page=1 --jq "$J"
gh api -X GET search/repositories -f q='topic:programming-language' -f sort=stars -f per_page=100 -f page=2 --jq "$J"
gh api -X GET search/repositories -f q='topic:programming-language' -f sort=updated -f per_page=100 --jq "$J"
gh api -X GET search/repositories -f q='topic:programming-language created:>2024-01-01' -f sort=stars -f per_page=100 --jq "$J"
gh api -X GET search/repositories -f q='topic:programming-language topic:dsl' -f sort=stars -f per_page=100 --jq "$J"
gh api -X GET search/repositories -f q='topic:programming-language topic:declarative-programming' ... 
gh api -X GET search/repositories -f q='topic:declarative-programming' ...
gh api -X GET search/repositories -f q='topic:programming-language topic:educational' ...
gh api -X GET search/repositories -f q='topic:programming-language topic:education' ...
gh api -X GET search/repositories -f q='topic:programming-language language:Go' ...
gh api -X GET search/repositories -f q='topic:programming-language topic:full-stack' ...
gh api -X GET search/repositories -f q='topic:programming-language topic:web-framework' ...
gh api -X GET search/repositories -f q='topic:programming-language topic:low-code' ...
gh api -X GET search/repositories -f q='topic:programming-language topic:portuguese' ...
gh api -X GET search/repositories -f q='topic:programming-language topic:chinese' ...
gh api -X GET search/repositories -f q='topic:non-english-programming-language' ...
gh api -X GET search/repositories -f q='portugol' -f sort=stars -f per_page=30 ...
gh api repos/<dono>/<repo> --jq '[.full_name,.stargazers_count,.language,.pushed_at,.archived,.created_at]'
gh api repos/<dono>/<repo>/readme -H 'Accept: application/vnd.github.raw'
```

## 3. Números

### 3.1 Totais por busca (2026-09-28)

| Busca | `total_count` | Itens lidos |
|---|---:|---:|
| `topic:programming-language` | 11 702 | 300 (200 por estrelas, 100 por atualização) |
| `topic:programming-language created:>2024-01-01` | 4 341 | 100 |
| `topic:programming-language language:Go` | 573 | 100 |
| `topic:programming-language topic:dsl` | 155 | 100 |
| `topic:declarative-programming` | 278 | 100 |
| `topic:programming-language topic:declarative-programming` | 19 | 19 |
| `topic:programming-language topic:education` | 120 | 100 |
| `topic:programming-language topic:educational` | 74 | 74 |
| `topic:programming-language topic:web-framework` | 18 | 18 |
| `topic:programming-language topic:low-code` | 11 | 11 |
| `topic:programming-language topic:full-stack` | 10 | 10 |
| `topic:programming-language topic:chinese` | 21 | 21 |
| `topic:programming-language topic:portuguese` | 6 | 6 |
| `topic:non-english-programming-language` | 0 | 0 |
| `portugol` (texto livre) | 8 461 | 30 |

Observações:

- **37% dos repositórios do tópico foram criados depois de 2024-01-01** (4 341 de 11 702). O
  tópico cresce rápido, mas a maioria dos novos tem poucas estrelas.
- A interseção **declarativa ∩ linguagem** tem 19 repositórios; **full-stack ∩ linguagem**
  tem 10; o próprio Germanio (`flaviokalleu/germanio`) aparece em `declarative-programming`,
  `full-stack` e `portuguese`. O nicho do Germanio, medido por tópicos, é quase vazio; medido
  por projetos que não usam o tópico (Wasp, Darklang) não é.

### 3.2 Os 200 mais estrelados

- **37 de 200 não são linguagens** (livros, listas "awesome", tutoriais, cheatsheets:
  `charlax/professional-programming`, `LeCoupa/awesome-cheatsheets`, `trekhleb/learn-python`,
  `chai2010/advanced-go-programming-book`…). O tópico é ruidoso.
- **42 de 200 não recebem push há mais de dois anos**; 8 estão arquivados. Entre os parados
  estão os três exemplos mais estudados de "linguagem para quem não programa" ou "full-stack
  declarativa": `witheve/Eve` (último push 2018-03-20), `wenyan-lang/wenyan` (2023-10-20) e,
  fora do top 200, `MLstate/opalang` (2020-09-30) e `urweb/urweb` (2024-05-28).
- Linguagem de implementação no top 200: Rust 29, C 26, **Go 18**, C++ 18, sem linguagem 15,
  Python 13, JavaScript 10, TypeScript 10, Haskell 8, OCaml 5.
- Nos criados desde 2024 (top 100): Rust 35, Go 10, Python 10, C 7, C++ 7, TypeScript 5,
  Zig 4. **Rust domina a nova geração de compiladores**; Go é o segundo lugar empatado.

### 3.3 Ordem de grandeza por tema (regex sobre descrições; indicativo)

| Tema | top 200 | criados desde 2024 (top 100) | Go (top 100) |
|---|---:|---:|---:|
| IA/LLM/agentes | 7 | 10 | 0 |
| transpiler ("compiles to") | 9 | 10 | 7 |
| embarcável/scripting | 17 | 5 | 7 |
| educacional/iniciantes | 9 | 5 | 3 |
| web/full-stack | 2 | 4 | 1 |
| sistemas/baixo nível | 11 | 11 | 3 |
| verificação/prova | 8 | 4 | 1 |
| idioma natural não inglês | 6 | 6 | 6 |

Leitura: o tema que mais cresce nos novos é "linguagem para IA/agentes" (BAML, Vera, Sui,
Corvid, Bend 2 "blocks AI mistakes via proof"); web/full-stack continua raro entre os
projetos que se declaram *linguagens*.

## 4. Categorias (com projetos)

Estrelas e último push entre parênteses quando relevantes.

- **Compiladas (nativo/LLVM/C):** Odin (12 020), Carbon, Mojo (dentro de `modular/modular`),
  C3 (`c3lang/c3c`, 5 848), Hare (não verificado pela API), Jakt, Beef, Vale, Virgil, Nature,
  Rue ("mais alto que Rust, mais baixo que Go"), Cyrus, Kairo, Muon.
- **Interpretadas/embarcáveis:** Wren (8 140), Janet (4 428), Luau (5 915), Gravity, Umka,
  pocketpy, pocketlang, Rune, Koto, Steel, ArkScript, daScript, Pluto; em Go: Tengo
  (3 842), Anko, Elvish, ABS, Rye, Rad.
- **Transpilers:** Haxe, Fusion (`fut`), Wax, Borgo e Lisette (compilam para Go), Sky
  ("Elm-inspired… compiles to Go", 433, criado em 2026), Gauntlet, Soppo, Coconut,
  RacketScript, Reason.
- **DSLs:** Pkl (11 533), KCL, Jsonnet (7 571), Starlark (3 096), Penrose (diagramas), Alda
  (música, em Go), Cherri (atalhos da Siri, em Go), Noir e Leo (provas ZK), Solidity,
  Motoko, P (máquinas de estado), Catala (lei), Numbat (unidades), SATySFi (tipografia).
- **Educacionais:** Hedy (1 690, 54 idiomas), Snap!, Portugol Studio e Portugol Webstudio,
  Potigol, Égua, Plush, Bril (IR educacional).
- **Experimentais/pesquisa:** Unison (código endereçado por conteúdo), Eve, Lamdu, Egison,
  Koka, Flix, Kip, EO, Hylo (1 562), Inko (1 303), Pony (6 192).
- **Sistemas:** Odin, C3, Zig-likes, Jakt, Hylo, Vale, Virgil, Perk, Checked C, Wuffs.
- **Web/full-stack:** Wasp (18 755, fora do tópico), Imba (6 509, "the friendly full-stack
  language"), Mint (4 267, front-end), Elm + Lamdera, Darklang (2 169), Ur/Web (852), Opa
  (1 263), Links (359), Fanx, Coi (WASM reativo, 568), Wing (5 402, nuvem), Ballerina (3 861,
  integração).
- **Declarativas:** Datalog/Soufflé (1 174), Clingo, DDlog (arquivado), Logtalk, Nucleoid,
  Wybe, Catala, Pkl, Eve; frameworks declarativos: ApiLogicServer (regras → API + admin),
  DirectToSwiftUI (CRUD por regras).
- **Funcionais:** Elm, Koka, Flix, Unison, Grain, Tao, Gluon, Hamler, Egison, Reason.
- **Orientadas a dados/consulta:** Soufflé, DDlog, Eve (relacional), Numbat, DaCe, PRQL-like
  (não verificado no tópico), `knowledge-graph-language`.
- **Pequenas/runtime mínimo:** Wren, Janet, pocketpy, Umka, Tiny-Lua-Compiler, b1fipl,
  Tengo, Gravity.
- **Aplicações descritas em linguagem controlada (os pares diretos do Germanio):** Human
  (`barun-bash/human`, Go, 7 estrelas, 2026), IntentLang (`sethiramicrosoft/intentlang`,
  TypeScript, 1 estrela, 2026), Amana (`akleeko2/amana-dsl`, Rust, 2026, RTL/árabe),
  IntentLang de `krakenhavoc` (especificação para agentes), madilang, Theia, Jounce,
  UniStack, Zap, EPL. Todos criados em 2026, todos com menos de 10 estrelas.
- **Idiomas naturais não ingleses:**
  - chinês: wenyan (20 279, parado desde 2023-10), Cantonese (1 194), 凹语言/Wa (1 770, em
    Go), qi (401), `program-in-chinese/overview` (402), vários com 0 a 2 estrelas;
  - português: Portugol Studio (764, último push 2023-04), Portugol Webstudio (375, ativo),
    Potigol (268), Égua (850, ativo), Libra (41), Portuscript (em Go), Germanio;
  - turco: Kip (880, casos gramaticais como tipos); coreano: Han; urdu, bengali (Borno),
    hebraico (Codesh), hindi/"Bhai" (bhai-lang, 4 090, e clones), árabe (Amana na interface,
    não nas palavras-chave);
  - multilíngue por tradução de palavras-chave: Hedy (54 idiomas), Scratch/Snap! (interface
    e blocos).

## 5. Critérios de seleção

Um projeto entra na lista curta quando atende a pelo menos dois critérios e não está na lista
de exclusão:

1. **declarativo** ou orientado à intenção (descreve o quê, não o como);
2. **gera ou executa uma aplicação web completa** (dados, API, interface, autenticação);
3. **público que não programa** ou iniciante;
4. **implementado em Go** ou com arquitetura comparável (binário único, runtime próprio);
5. **história instrutiva**: morreu, mudou de rumo ou reverteu uma decisão de desenho que o
   Germanio também tomou;
6. **idioma natural não inglês** nas palavras-chave.

Descartados da lista curta, com motivo: Mojo, Odin, Hare, C3, Hylo, Inko, Pony, Luau, Wren,
Janet, Red, Grain, Koka (linguagens de propósito geral ou de sistemas: nada a ensinar sobre o
problema do Germanio além do que Go/Rust/Zig já cobriram); Pkl, Jsonnet, Starlark, CUE, HCL
(já em [sintaxe-e-configuracao.md](sintaxe-e-configuracao.md)); Soufflé/Datalog (a ideia útil,
consulta declarativa, aparece via Eve); Imba e Mint (sintaxe de front-end para programadores,
não intenção); Ballerina e Unison (lidos, citados em DISCOVERIES.md só como apoio);
Wasp foi mantido apesar de estar fora do tópico.

## 6. Selecionados (15)

| # | Projeto | Repositório | Por que |
|---|---|---|---|
| 1 | Wasp | <https://github.com/wasp-lang/wasp> | especificação declarativa de app full-stack; **abandonou a própria linguagem em 2026** |
| 2 | IntentLang | <https://github.com/sethiramicrosoft/intentlang> | inglês controlado → app completa, determinístico, sem IA; par mais próximo do Germanio |
| 3 | Human | <https://github.com/barun-bash/human> | inglês estruturado → código gerado; escrito em Go; "Intent IR" |
| 4 | Amana | <https://github.com/akleeko2/amana-dsl> | DSL full-stack declarativa; segura por padrão; RTL/árabe |
| 5 | Hedy | <https://github.com/hedyorg/hedy> | gradual, 54 idiomas de palavras-chave, pesquisa publicada |
| 6 | Portugol / Potigol / Égua | <https://github.com/UNIVALI-LITE/Portugol-Studio>, <https://github.com/potigol/potigol>, <https://github.com/eguadev/egua> | o que aconteceu com linguagens em português |
| 7 | Kip e wenyan | <https://github.com/kip-dili/kip>, <https://github.com/wenyan-lang/wenyan> | idioma natural levado a sério (morfologia) vs. como curiosidade |
| 8 | Eve | <https://github.com/witheve/Eve> | "programação para humanos", relacional; morreu em 2018 |
| 9 | Opa | <https://github.com/MLstate/opalang> | full-stack em uma linguagem; morreu por volta de 2014 |
| 10 | Ur/Web | <https://github.com/urweb/urweb> | full-stack com garantias fortes; nicho acadêmico |
| 11 | Links | <https://github.com/links-lang/links> | a formulação do "impedance mismatch" entre camadas |
| 12 | Darklang | <https://github.com/darklang/dark> | linguagem + editor + infraestrutura; reescreveu tudo em 2023 |
| 13 | Elm / Lamdera | <https://github.com/elm/compiler>, <https://github.com/lamdera/compiler> | confiabilidade por construção; full-stack via plataforma |
| 14 | Catala | <https://github.com/CatalaLang/catala> | especialistas de domínio (juristas) revisando o código |
| 15 | Wing | <https://github.com/winglang/wing> | infraestrutura e aplicação no mesmo programa; simulador local |

As fichas (problema, ideia, trade-off, aplicabilidade) estão em [DISCOVERIES.md](DISCOVERIES.md).
