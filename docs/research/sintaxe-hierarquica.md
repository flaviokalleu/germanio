# Pesquisa: sintaxe hierárquica e contextual do Germanio

**Status:** base da decisão registrada em `docs/INTENCAO.md` › Sintaxe hierárquica.
**Data:** 2026-09-28. **Natureza:** pesquisa, sem força normativa; a norma é o `INTENCAO.md`.

Fontes consultadas pela web. Kotlin, gofmt, TOML e Gleam foram citados de memória na
primeira passagem; as conclusões usadas delas são genéricas (escopo de builders, formato
canônico, chaves pontuadas, forma única) e não dependem de detalhes de versão.

## Parte 1 — Auditoria da sintaxe atual (antes desta evolução)

| Achado | Evidência | Consequência |
|---|---|---|
| Frases reconhecidas por palavra-chave solta | `isIntentLine` procura `tem`/`pode`/`começa`… em qualquer posição da linha | não há gramática formal; cada construção nova vira mais um `if` |
| Linha desconhecida some em silêncio | `projetos` + `tem` + `nome` e `issue podee fechar` passam no `check` sem efeito | viola determinismo e a promessa do `ge check` |
| Tab conta como 2 espaços no dialeto de intenção | `lexer.go`, laço de indentação | largura invisível decide estrutura; o núcleo estrito já trata tab como erro |
| Recuo aceito sem pilha de níveis | `blockLines` só compara com a coluna 1 | recuo inconsistente dentro de um bloco não é diagnosticado |
| Repetição do sujeito | `issue` aparece ~10 vezes em `issues.ge`; cada permissão repete o alvo | carga de leitura; a informação já existe no contexto |
| Mesmo dado espalhado | `projeto` é descrito em `projetos.ge`, `issues.ge`, `pipelines.ge` | necessário: fusão determinística de declarações |
| Sem formatter para a camada de intenção | `ge fmt` só aceita o núcleo estrito (`ParseGermanio`) | nenhuma forma canônica para `.ge` de aplicação |
| `ge explain` sem origem | mostra o resultado, não de onde veio cada fato | dependências escondidas (Green & Petre) |
| Autorização por "governo" herdado sem critério | `administrar milestones` governava issues (corrigido em `58d89c4`) | regras derivadas precisam de semântica precisa |

## Parte 2 — Pesquisa
## Conclusões aplicáveis ao Germanio

1. **A indentação vira tokens no lexer, e o parser continua livre de contexto.** O Python transforma indentação em INDENT/DEDENT usando uma pilha de colunas. Um recuo que não volta a um nível já empilhado dá erro léxico, e no EOF o lexer fecha todos os níveis abertos ([Python, Lexical analysis](https://docs.python.org/3/reference/lexical_analysis.html)). Haskell faz o mesmo trabalho de outro jeito: define o layout como uma tradução para `{ ; }` explícitos, a função L ([Haskell 2010, cap. 10.3](https://www.haskell.org/onlinereport/haskell2010/haskellch10.html)). Adams mostra que regras de indentação não cabem numa CFG pura, mas cabem numa extensão simples que ainda permite LR/GLR ([Adams, POPL 2013](https://michaeldadams.org/papers/layout_parsing/)). *Para o Germanio:* o lexer emite `BLOCO_ABRE/BLOCO_FECHA` e o parser atual segue recursivo-descendente, sem olhar colunas. Isso deixa a especificação formal e testável.

2. **A semântica de um bloco é o caminho, igual a uma frase plana.** O CUE trata `a: b: c: 1` como a mesma coisa que o aninhamento completo ([CUE spec](https://cuelang.org/docs/reference/spec/)). O TOML também tem chaves pontuadas equivalentes a tabelas ([TOML 1.0](https://toml.io/en/v1.0.0)). *Para o Germanio:* toda linha folha se normaliza para uma tupla canônica, por exemplo `(recurso=projetos, seção=acesso, papel=developer, ação="enviar código")`. A frase plana `developer pode enviar código para projetos` gera exatamente a mesma tupla. As regras vivem sobre as tuplas; sintaxe hierárquica e plana são só duas superfícies para a mesma representação intermediária.

3. **Blocos repetidos se unem por fusão, e conflito é erro.** O CUE permite declarar o mesmo campo várias vezes desde que os valores não conflitem, e reporta erro quando conflitam, nunca sobrescreve em silêncio ([CUE Tour, Unification](https://cuelang.org/docs/tour/basics/unification/); [The Logic of CUE](https://cuelang.org/docs/concept/the-logic-of-cue/)). O HCL separa as coisas: um atributo definido duas vezes é erro ("Attribute redefined"), mas blocos podem se repetir ([HCL spec](https://github.com/hashicorp/hcl/blob/main/hclsyntax/spec.md)). O TOML proíbe definir a mesma tabela duas vezes ([TOML](https://toml.io/en/v1.0.0)). *Para o Germanio:* conjuntos (`tem`, `pode`, `acesso`) se unem. O mesmo fato repetido é idempotente e gera um aviso de duplicata. Um valor escalar com dois valores diferentes é erro, e a mensagem mostra as duas origens.

4. **Tipos nunca são inferidos da forma do texto.** No YAML 1.1, `NO` vira `false` (o "Norway problem"), e o erro passa na validação sem ruído ([StrictYAML](https://hitchdev.com/strictyaml/why/implicit-typing-removed/)). *Para o Germanio:* o significado de uma palavra depende do contexto sintático (dentro de `tem`, é campo), nunca da aparência do valor. Literais ambíguos exigem aspas ou tipo declarado.

5. **Só espaços, com unidade fixa.** O Nim proíbe tabs na indentação ([Nim Manual](https://nim-lang.org/docs/manual.html)). O Python rejeita misturas que dependem do tamanho do tab (TabError) ([Python](https://docs.python.org/3/reference/lexical_analysis.html)), e o PEP 666, rejeitado, virou o marco que encerrou o debate ([PEP 666](https://peps.python.org/pep-0666/)). *Para o Germanio:* 4 espaços por nível, tab é erro com correção automática no formatador, e um recuo que não é múltiplo de 4 é erro.

6. **O formatador faz parte da linguagem.** O gofmt eliminou o debate de estilo ([The Go Blog, gofmt](https://go.dev/blog/gofmt)). O Black garante estabilidade entre versões e confere que a AST antes e depois é equivalente ([Black style](https://black.readthedocs.io/en/stable/the_black_code_style/index.html)). *Para o Germanio:* `germanio formatar` é idempotente (`f(f(x)) == f(x)`) e verifica que as tuplas canônicas não mudaram.

7. **Níveis progressivos na mesma gramática.** O Hedy introduz sintaxe por níveis, e cada nível acrescenta ou refina uma construção ([Hermans, ICER 2020](https://www.felienne.com/wp-content/uploads/2020/07/Hedy_paper_website_draft.pdf); [avaliação 2022](https://www.sciencedirect.com/science/article/pii/S2590118422000557)). *Para o Germanio:* os níveis são subconjuntos da mesma linguagem, não dialetos. Um arquivo de nível 1 continua válido no nível 4.

8. **As palavras-chave devem ser testadas com leigos, não escolhidas por tradição.** Stefik & Siebert mostraram que `for/while/foreach` foram as palavras de laço menos intuitivas para não programadores, e que a sintaxe estilo C rende tanto quanto uma sintaxe aleatória para novatos ([ACM TOCE 2013](https://dl.acm.org/doi/10.1145/2534973); [resumo](https://neverworkintheory.org/2014/01/29/stefik-siebert-syntax.html)). *Para o Germanio:* verbos em PT comum (`tem`, `pertence a`, `pode`) têm respaldo; convém validar novas palavras com pequenos testes de acerto com usuários reais.

9. **Parecer linguagem natural não basta: a gramática precisa ser pequena e previsível.** Cook, sobre o AppleScript: a sintaxe naturalista é fácil de ler e difícil de escrever, porque o usuário não adivinha quais frases o sistema aceita ([Cook, HOPL III](https://www.cs.utexas.edu/~wcook/Drafts/2006/ashopl.pdf)). O Inform 7 aceita um subconjunto do inglês e cria a promessa falsa de entender qualquer frase ([Inform 7 Concepts](https://catn.decontextualize.com/inform7/); [Nelson, white paper](https://www.cs.tufts.edu/comp/150FP/archive/graham-nelson/WhitePaper.pdf)). *Para o Germanio:* um número fechado e documentado de formas de frase, e o erro sempre sugere a forma aceita mais próxima.

10. **Contexto hierárquico com escopo controlado.** Os type-safe builders do Kotlin usam `@DslMarker` para impedir que um bloco interno enxergue o receptor de fora sem que o autor perceba ([Kotlin type-safe builders](https://kotlinlang.org/docs/type-safe-builders.html)). *Para o Germanio:* cada seção (`tem`, `acesso`) define quais palavras são válidas abaixo dela, e uma palavra de outra seção dá erro em vez de "subir" silenciosamente.

11. **Dimensões cognitivas que orientam o design** ([Green & Petre 1996](https://www.semanticscholar.org/paper/Usability-Analysis-of-Visual-Programming-A-Green-Petre/54f8ae5828615d1fe7d61c0038cc1ec77f4697b0); [resumo](https://en.wikipedia.org/wiki/Cognitive_dimensions_of_notations)):
    - *Closeness of mapping:* `projetos / acesso / developer` espelha como o domínio é falado.
    - *Hidden dependencies:* fusão entre arquivos esconde a origem de cada fato, então é preciso um `explicar`.
    - *Viscosity:* indentação profunda torna caro mover blocos; limite de profundidade e formatador ajudam.
    - *Premature commitment:* a forma plana deve continuar válida para quem ainda não sabe onde encaixar a regra.
    - *Error-proneness:* espaço invisível é fonte de erro, então só espaços e diagnóstico que mostra a coluna.
    - *Progressive evaluation:* `germanio check` roda em arquivos parciais.
    - *Role-expressiveness:* cada seção tem um nome de papel claro.

12. **O caminho da falha do novato é a mensagem de erro.** A revisão de Becker et al. reúne evidência de que mensagens de erro são uma barreira central para iniciantes ([ITiCSE WG 2019](https://dl.acm.org/doi/10.1145/3344429.3372508)). Entre as barreiras de Ko, Myers & Aung, as de *seleção* e *uso* ("não sei qual construção usar") dominam entre usuários finais ([VL/HCC 2004](https://faculty.washington.edu/ajko/papers/Ko2004LearningBarriers.pdf)). O Elm trata o erro como ensino, com trecho do código, explicação e dica ([Elm, Compiler Errors for Humans](https://elm-lang.org/news/compiler-errors-for-humans)). O rustc separa o erro (o quê) do `help` (como corrigir) e manda o detalhe longo para `--explain` ([rustc dev guide](https://rustc-dev-guide.rust-lang.org/diagnostics.html)).

## Riscos e antipadrões a evitar

- **Tipagem implícita pelo formato do valor** (Norway problem), que falha sem avisar ([StrictYAML](https://hitchdev.com/strictyaml/why/implicit-typing-removed/)).
- **Sobrescrever em silêncio quando um bloco se repete**, como fazem muitos merges de YAML. O CUE e o HCL mostram que o conflito precisa ser explícito ([CUE](https://cuelang.org/docs/tour/basics/unification/), [HCL](https://github.com/hashicorp/hcl/blob/main/hclsyntax/spec.md)).
- **Uma gramática "natural" aberta**, que se lê bem e se escreve mal ([Cook](https://www.cs.utexas.edu/~wcook/Drafts/2006/ashopl.pdf)).
- **Uma regra de layout sensível a colunas arbitrárias**, como o alinhamento do Haskell pela coluna do primeiro token. É poderosa, mas pouco intuitiva para leigos e difícil de formalizar ([Adams](https://michaeldadams.org/papers/layout_parsing/)). Prefira níveis discretos de 4 espaços.
- **Tabs com largura dependente do editor** ([PEP 8](https://peps.python.org/pep-0008/), [Nim](https://nim-lang.org/docs/manual.html)).
- **Várias formas de dizer a mesma coisa sem forma canônica.** Isso aumenta a carga de leitura em equipes grandes. O Gleam aposta em uma forma só e formatador embutido ([gleam.run](https://gleam.run/)).
- **Dependências escondidas entre arquivos fundidos** (a dimensão *hidden dependencies* de Green & Petre).
- **Mensagens que misturam diagnóstico e correção**, ou que usam jargão do parser ("unexpected DEDENT") ([rustc guide](https://rustc-dev-guide.rust-lang.org/diagnostics.html), [Becker et al.](https://dl.acm.org/doi/10.1145/3344429.3372508)).

## Recomendações concretas

### Regras de indentação

- **Unidade:** exatamente 4 espaços por nível. O recuo precisa ser múltiplo de 4, e subir mais de um nível de uma vez é erro ("filho sem pai").
- **Tabs:** proibidos na indentação, com erro e correção automática oferecida por `germanio formatar`. Dentro de texto entre aspas, são permitidos.
- **Algoritmo:** a pilha do Python. O lexer emite `ABRE` ao aumentar o recuo, um ou mais `FECHA` ao voltar, e erro quando volta para uma coluna que não existe na pilha.
- **Linhas vazias e linhas só de comentário:** não contam para a indentação, como no Python. Comentário é `#` até o fim da linha.
- **Comentário no fim da linha:** é permitido e fica preso à linha de código, para o formatador preservar.
- **Continuação:** só implícita, dentro de `( [ {` ou de texto em aspas triplas. Não usar `\` nem regra de "linha termina em operador".
- **EOF:** fecha todos os níveis abertos. Falta de `\n` final não é erro.
- **BOM e CRLF:** aceitos e normalizados.
- **Profundidade:** aviso a partir de 5 níveis, sugerindo dividir o bloco ou usar a forma plana.

### Equivalência entre caminho e frase plana

- Defina na especificação uma função `achatar(bloco) -> [fato]`: cada folha vira caminho mais valor.
- A frase plana é um açúcar sintático que produz o mesmo `fato`.
- Teste de propriedade: para todo programa P, `fatos(P) == fatos(achatar_texto(P))`.
- Diagnósticos e `explicar` sempre mostram a frase plana do fato, com arquivo e linha de origem.

### Fusão de blocos repetidos

- **Seções de conjunto** (`tem`, `pode`, `acesso`, `pertence a`): união.
- **Fato idêntico repetido:** aceito, com aviso "já declarado em arquivo:linha".
- **Propriedade escalar com valores diferentes** (por exemplo `nome obrigatório` e `nome opcional`): erro, com as duas origens.
- **Ordem:** a fusão é comutativa, então a ordem dos arquivos não muda o resultado (determinismo).
- **Remoção:** nunca é implícita; se um dia for necessária, precisa de uma palavra explícita.

### Níveis de progressive disclosure

1. **Frases planas:** `projetos tem nome obrigatório`.
2. **Blocos de um nível:** `projetos` e, abaixo, `tem`.
3. **Contextos aninhados**, como `acesso / papel / ação`, e condições.
4. **Referências entre arquivos**, estados e regras derivadas.
5. **Adaptadores** (camada avançada, fora do domínio).

`germanio check --nivel N` pode avisar quando o arquivo usa construções de um nível acima, como faz o Hedy.

### Formato das mensagens de erro

Siga o padrão o quê, onde, por quê e como corrigir, com o `help` separado do erro, como no rustc e no Elm:

```
erro[G0107]: recuo não corresponde a nenhum bloco aberto
  --> backend/projetos.ge:12:7
   |
10 |     acesso
11 |         developer
12 |       enviar código
   |       ^ 6 espaços; esperava 8 (dentro de "developer") ou 4 (dentro de "acesso")
   = por quê: cada nível usa 4 espaços; o texto abaixo pertence ao bloco acima
   = ajuda: use 8 espaços para "enviar código" ficar dentro de "developer"
   = equivale a: developer pode enviar código para projetos
```

- Código estável `Gnnnn` com `germanio explicar G0107` (como o `--explain` do Rust).
- Nada de nomes de token interno.
- Sugerir sempre a forma válida mais próxima, por distância de edição sobre as palavras-chave da seção.
- Um `germanio explicar projetos.acesso.developer` que lista os fatos e suas origens combate as dependências escondidas.

### Formatador canônico

- Faz parte do binário: `germanio formatar` e `germanio formatar --verificar` para CI.
- É idempotente e verifica que o conjunto de fatos antes e depois é o mesmo (como a checagem de AST do Black).
- Normaliza para 4 espaços, remove espaços no fim, reduz linhas vazias a no máximo 1 dentro de bloco e 2 entre recursos, e deixa um `\n` final.
- Preserva comentários presos ao nó seguinte ou à mesma linha, carregando-os na árvore de sintaxe concreta, como o gofmt faz com o `go/ast` e seu CommentMap ([go.dev/blog/gofmt](https://go.dev/blog/gofmt)).
- Não reordena declarações do usuário, porque a ordem é *secondary notation* (Green & Petre). No máximo, oferece um `--ordenar` opcional.
- Não converte automaticamente entre as formas plana e hierárquica. Isso fica como ação explícita (`germanio formatar --aninhar` e `--achatar`), para não impor um compromisso prematuro.

## Parte 3 — Decisões tomadas a partir da pesquisa

| # | Decisão | Base | Onde diverge da recomendação e por quê |
|---|---|---|---|
| 1 | Indentação por **pilha de níveis** (INDENT/DEDENT do Python); recuar para uma coluna que não está aberta é erro | Python, Adams | a pesquisa sugeria exigir múltiplos de 4; 80 dos ~100 `.ge` existentes usam passos de 2 no dialeto antigo. A pilha dá a mesma garantia estrutural sem quebrar código; a forma canônica de 4 espaços fica com o formatter |
| 2 | **Tab na indentação é erro**, com correção pelo formatter | Nim, PEP 8/666 | nenhum `.ge` do repositório usa tab: custo zero de migração |
| 3 | Linhas vazias e de comentário não contam para a indentação | Python | — |
| 4 | **Caminho hierárquico ≡ frase plana**: um bloco se reduz aos mesmos fatos da frase | CUE, TOML | a redução é feita por uma tabela fechada de seções, que gera as frases planas já especificadas; a equivalência vale por construção e é testada |
| 5 | Seções formam um **conjunto fechado por contexto**; palavra fora do lugar é erro com sugestão | Kotlin `@DslMarker`, Cook (AppleScript), Inform 7 | — |
| 6 | Blocos do mesmo dado **se unem** (em arquivos diferentes); o mesmo valor escalar com dois valores diferentes é erro com as duas origens | CUE, HCL | — |
| 7 | Linha no topo que nenhuma construção reconhece é **erro** com a forma mais próxima | Becker et al., Elm, rustc | corrige o descarte silencioso atual |
| 8 | A forma plana continua válida; o formatter **não** converte entre formas | Green & Petre (premature commitment) | não há `ge migrate`: a migração dos exemplos é feita uma vez, à mão, e a forma plana não é depreciada |
| 9 | Mensagens: o que aconteceu, onde, por que, como corrigir, e a frase plana equivalente quando o erro está num bloco | Elm, rustc | — |
| 10 | Formatter canônico: 4 espaços por nível, sem espaço no fim, no máximo uma linha vazia seguida, comentários preservados, idempotente, e confere que os fatos não mudaram | gofmt, Black | reordenar declarações não faz parte (ordem é notação secundária) |
| 11 | `ge explain` mostra a origem (arquivo, linha, caminho hierárquico) e a frase plana equivalente de cada fato | Green & Petre (hidden dependencies) | — |
