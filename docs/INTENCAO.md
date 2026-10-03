# Germanio — camada de intenção

## Autoridade e escopo normativo

Este arquivo, `docs/INTENCAO.md`, é a **especificação normativa da camada de intenção**.
É o documento correspondente ao nome `GERMANIO_INTENT_LAYER.md` usado na solicitação
original; mantém-se o caminho existente para não criar duas fontes concorrentes.

O usuário descreve o que quer que exista e aconteça. Germanio resolve como isso será
implementado. O público padrão pode ter praticamente zero conhecimento de programação.

**DEVE**, **NÃO DEVE** e **PODE** indicam obrigações, proibições e opções. Exemplos `.ge`
expressam contratos de comportamento; fragmentos não são necessariamente aplicações
completas. Saídas conceituais e decisões pendentes são identificadas como tais.
Uma obrigação normativa não é, por si só, prova de implementação no commit atual.

Se implementação, exemplo antigo ou aplicação de teste contradisserem esta especificação,
não altere silenciosamente a norma para justificar o código. Determine se a implementação
está errada, a especificação está incompleta ou uma nova decisão arquitetural é necessária.
Mudanças semânticas devem ser deliberadas, registradas e acompanhadas de testes.

| Referência | Responsabilidade |
| --- | --- |
| [INTENCAO.md](INTENCAO.md) | Como a camada de intenção deve funcionar |
| [Skill de simplicidade](../skills/germanio-simplicity/SKILL.md) | Como agentes aplicam a filosofia |
| [AGENTS.md](../AGENTS.md) | Obriga a consulta à norma e à skill antes do trabalho |
| Código e testes | Evidência de conformidade e de limitações da implementação |
| [SPEC da fundação](../SPEC.md) | Subconjunto técnico da fundação, em seu próprio escopo |
| [SPEC histórica](SPEC.md) | Referência da sintaxe anterior; não redefine o nível padrão |

Nenhum desses documentos substitui os outros. O [índice](README.md) define a ordem de leitura.

Um arquivo `.ge` responde quatro perguntas: **o que existe, quem pode fazer o quê,
o que deve acontecer e o que deve aparecer.** O runtime responde *como*
(banco, validação, HTTP, segurança, interface). Cada frase tem um único
significado; nada é adivinhado e nenhuma IA participa da execução.

```ge
crie sistema Clientes

clientes
    tem
        nome obrigatório
        email obrigatório e único
        telefone

    permita
        pesquisar

página Clientes
    mostre clientes
    permita
        cadastrar
        editar
        excluir
```

O mesmo programa em frases planas (`tenha clientes`, `cada cliente tem …`,
`permita pesquisar clientes`, `crie página Clientes …`) é equivalente; veja
[Sintaxe hierárquica e contextual](#sintaxe-hierárquica-e-contextual).

## Organização do projeto

`germanio init loja` cria um projeto com dois lados, como um app com backend e
frontend separados (Laravel, Next.js), mas sem rotas nem controllers:

```text
loja/
├── app.ge              crie sistema Loja + importar "backend" + importar "frontend"
├── backend/            o que existe, quem pode fazer o quê, o que acontece
│   ├── pessoas.ge      tenha usuarios, tenha login
│   ├── produtos.ge     tenha produtos, cada produto tem…, somente administrador pode…
│   └── pedidos.ge      pedido começa aberto, pedido pode pagar, antes de criar pedido…
├── frontend/           o que aparece
│   └── loja.ge         importar produtos e pedidos do backend + crie página …
├── .env.exemplo
└── .gitignore
```

- `importar "backend"` importa a pasta inteira (todos os `.ge`, em ordem alfabética):
  um arquivo novo entra no sistema sem registro manual.
- Cada arquivo do `frontend/` diz o que usa: `importar produtos e pedidos do backend`.
  Uma página só mostra dados importados, e o que se importa precisa existir no backend —
  os erros dizem a linha a escrever.
- Cada lado mantém seu papel: `frontend/` só tem páginas; `tenha`, `pode`, `quando`,
  login e integração ficam no `backend/`, e páginas nunca ficam no `backend/`.
- `integracoes/` (opcional, **nível avançado**) guarda adaptadores de protocolos externos
  (por exemplo o `gitlab-runner`): podem ser técnicos, só traduzem para capabilities do
  Germanio e não podem declarar dados, permissões ou páginas.
- A página diz *o que aparece*; o backend decide *quem pode*. `permita criar` numa página
  mostra o formulário apenas a quem o backend permite.

## Níveis

| Nível | Uso | Exemplos |
| --- | --- | --- |
| 1 — Intenção / declaração | padrão: existência, relações, permissões, estados e páginas | `clientes` + `tem`, `issue pode fechar`, `acesso` |
| 2 — Configuração | personalizar comportamento sem implementar mecanismos | `login bloqueia após 10 tentativas por 10 minutos` |
| 3 — Lógica | último recurso para regra específica não expressável declarativamente | `quando`, `antes de`, `se`, `para cada` |
| 4 — Primitivas | controle técnico avançado | `git.*`, `cripto.*`, `acesso.*`, `rotas`, `requisicao`, `responder` |
| 5 — Interop / adaptadores | protocolos de sistemas externos, só em `integracoes/` | `integracoes/gitlab_runner.ge` |

Antes de usar nível 3, verifique se falta uma abstração genérica. Níveis 4 e 5 não devem ser
necessários para aplicações comuns. O iniciante começa no nível 1 e só desce quando precisa;
nenhuma operação comum pode exigir aprender um nível inferior. Os níveis são subconjuntos da
mesma linguagem: um arquivo de nível 1 continua válido quando o projeto usa os demais.

Simplicidade não significa linguagem natural livre: cada construção tem significado formal,
determinístico e documentado. Não há IA interpretando programas em runtime.
“Faça as issues funcionarem bem” não é uma construção válida.

O nível normal não deve exigir HTTP, REST, GET/POST/PUT/DELETE, JSON, SQL, ORM, chaves
estrangeiras, controllers, services, repositories, middlewares, serializers, migrations,
bcrypt, JWT, WebSocket, filas, workers, transações, status codes, headers, IDs ou payloads
quando esses detalhes puderem ser derivados deterministicamente.

## Três mundos arquiteturais

| Mundo | Responsabilidade | Limite |
| --- | --- | --- |
| Domínio / aplicação | descrever produto, por exemplo `runners executam jobs` | intenção simples; sem protocolo externo |
| Core Germanio | mecanismos genéricos: persistência, HTTP, Git, criptografia, storage, transações, fila, executor, lease, heartbeat, concorrência | não conhece regras ou formatos específicos de um produto |
| Adaptadores / integrações | traduzir protocolo externo para capabilities Germanio | sem declaração de domínio, páginas ou regras de negócio |

**Adaptadores podem conhecer a complexidade do sistema externo. O domínio não.**

Um adaptador como `integracoes/gitlab_runner.ge`, identificado como **NÍVEL AVANÇADO /
ADAPTADOR DE COMPATIBILIDADE**, pode conhecer `/api/v4/...`, `CI_JOB_ID`, `JOB-TOKEN`,
`Content-Range`, cabeçalhos, status HTTP e payloads do gitlab-runner. Deve somente traduzir
mundo externo ↔ capabilities genéricas. Não pode substituir autorização ou invariantes.
HTTP e Git são protocolos genéricos permitidos no core; a proibição é acoplar o core ao
contrato particular de um produto, mesmo que seus campos sejam renomeados em português.

## Eficiência: simples para o humano, eficiente para a máquina

Germanio minimiza ao mesmo tempo o **custo cognitivo para o humano** e o **custo
computacional para a máquina**, e não sacrifica um pelo outro em silêncio. A simplicidade da
superfície não justifica aplicações pesadas; a engenharia fica embaixo. Desempenho, memória,
concorrência e carga são requisitos da linguagem, e não uma otimização futura.

Ordem de prioridade: **correto → seguro → mensurável → rápido**. Nenhuma otimização
dispensa uma garantia de correção ou de segurança; `unsafe` e técnicas equivalentes só entram
com necessidade medida e isolamento rigoroso.

Obrigações:

- **Meça antes de otimizar.** Uma decisão de performance cita a medição que a motivou. Go é
  uma vantagem potencial, não uma garantia: interpretação, reflexão, `map` genérico,
  conversões e serializações repetidas têm custo e são auditados
  (`docs/research/performance/`).
- **Custos considerados** em todo subsistema (parser, AST, análise, runtime, servidor, banco,
  frontend): CPU, memória, alocações, GC, I/O, startup, latência, throughput, tamanho do
  binário e o JavaScript enviado.
- **Pague apenas pelo que usar.** Uma aplicação que não declara uma capability (chat,
  WebSocket, fila, IA, storage, Git, e-mail…) NÃO DEVE inicializá-la nem mantê-la em memória;
  o que está no binário sem uso é dívida registrada.
- **Sem ingenuidade no banco.** `mostre clientes` NÃO DEVE virar uma leitura sem limite. O
  runtime pagina, projeta só o necessário, evita N+1, usa índices derivados do que é declarado
  e mantém as transações curtas.
- **Streaming.** Arquivos, uploads, downloads, logs, Git e respostas grandes são processados
  de forma incremental quando possível: um arquivo de 5 GB não exige 5 GB de memória.
- **Concorrência limitada e com backpressure.** O autor declara intenção; o runtime decide
  workers, limites, cancelamento, timeouts, propagação de erro e limpeza. Nenhuma construção
  cria trabalho ilimitado: produtor saturado é contido de forma previsível.
- **Limites seguros por padrão** para conexões, workers, jobs concorrentes, timeouts, tamanho
  de upload, memória por operação, pool do banco e taxa de requisições. `ge explain` mostra
  os limites em vigor.
- **Nada de mecanismo no código comum.** goroutine, channel, mutex, ponteiro, buffer pool,
  connection pool e allocator são mecanismos do core; não aparecem no nível padrão. O nível
  avançado pode chegar a eles quando realmente necessário.
- **Benchmarks permanentes** em três categorias: micro (lexer, parser, resolver,
  expressões, serialização), aplicação (HTTP, JSON, banco, WebSocket, upload, download, jobs,
  renderização) e stress (muitas conexões, filas e arquivos grandes). Todo resultado
  publicado registra o hardware, o SO, a versão do Go e do Germanio, a configuração, o
  dataset, o comando e o commit. Um número sem contexto não é publicado.
- **O custo da abstração é medido** contra um baseline equivalente em Go direto (e, no
  frontend, contra um alvo equivalente). O objetivo não é vencer benchmark; é saber quanto
  custa o Germanio.
- **Budgets vêm depois do baseline.** Não se fixam números antes de medir. Com o baseline
  medido, definem-se budgets para o startup do CLI, a compilação, o overhead do runtime, a
  memória em repouso, as alocações, o overhead HTTP e o bundle do frontend.
- **Regressão consciente.** Uma feature que torna algo significativamente mais pesado
  (ordem de grandeza: várias vezes) não entra em silêncio: é identificada, documentada,
  investigada e decidida.

A intenção de carga e de concorrência (por exemplo "aceite muitas conexões", "processe
pedidos em paralelo") só vira sintaxe depois que a semântica, os limites padrão e os testes
estiverem definidos. Até lá, ela é pendência de design, e não uma frase aceita.

## Sintaxe hierárquica e contextual

**O que está abaixo pertence ao contexto que está acima.** A indentação carrega significado:
o caminho de blocos fornece o sujeito, o aspecto e o ator que uma frase plana precisaria repetir.

```ge
projetos

    tem
        nome obrigatório
        caminho obrigatório
        visibilidade
        repositório
        membros com papel

    pertence a
        grupo opcional

    pode
        ser arquivado

    regras
        herda membros do grupo
        quem cria vira owner
        precisa de pelo menos um owner
        não pode ser mais visível que o grupo
        arquivado é somente leitura

    acesso
        guest
            ver
        reporter
            baixar código
        developer
            criar
            enviar código
        somente maintainer
            enviar código para a branch padrão
        somente owner
            excluir

    integração
        nome "projects"
```

`projetos › acesso › developer › enviar código` **é** `developer pode enviar código para projetos`:
os dois produzem o mesmo fato, com a mesma semântica, validação e diagnóstico. A forma
hierárquica é a forma recomendada para descrever um dado; a frase plana continua válida para
fatos isolados e para quem ainda não sabe onde encaixar a regra. O formatter não converte uma
na outra.

Germanio não busca o menor número de caracteres; busca o menor número de conceitos. Por isso
o aspecto nunca é omitido: `projetos` › `developer` › `enviar` é inválido; `acesso` diz que o
que vem abaixo são pessoas e o que elas podem fazer.

### Como avaliar uma sintaxe

Uma construção nova é julgada nesta ordem de prioridade:

1. conceitos técnicos que ela exige (menos é melhor);
2. carga cognitiva de quem lê e escreve;
3. repetição, reduzida só quando a hierarquia já determina a informação;
4. previsibilidade visual da hierarquia;
5. clareza semântica;
6. determinismo;
7. leitura rápida por pessoas;
8. só depois, linhas, palavras e caracteres.

Mais linhas podem ser melhores quando criam estrutura visual; menos palavras podem ser
piores quando removem contexto necessário. Nenhuma métrica quantitativa, sozinha, prova que
uma forma é melhor. As noções abaixo são diferentes e não devem ser confundidas:

| Noção | O que mede | Exemplo |
| --- | --- | --- |
| densidade textual | quanto texto por fato | `issue pode fechar` é mais denso que o bloco `issues` › `pode` › `fechar` |
| repetição | quantas vezes a mesma informação é escrita | `issue` repetido em dez frases; no bloco, uma vez |
| complexidade sintática | quantas regras de forma é preciso saber | seções fixas e um item por linha têm poucas regras |
| complexidade conceitual | quantas ideias é preciso entender | `acesso` › papel › ação são três ideias do domínio; chave estrangeira e status HTTP seriam técnicas |
| profundidade hierárquica | quantos níveis até um fato | `dado › acesso › papel › ação` são quatro; mais que isso pede outro bloco ou a frase plana |
| legibilidade | o quanto alguém encontra e entende o que procura | avaliada lendo exemplos pequenos, médios e grandes, não por contagem |

### Layout (normativo)

- Indentação só com **espaços**. Tab no início de linha é erro; `ge fmt` o substitui.
- Uma linha mais recuada que a anterior **abre um nível** filho dela; voltar a um recuo
  **fecha** os níveis até ele. Voltar para um recuo que não pertence a nenhum nível aberto
  é erro (a linha não tem pai). O fim do arquivo fecha todos os níveis.
- A forma canônica é **4 espaços por nível**; `ge fmt` a produz. Outros passos consistentes
  (por exemplo 2) são aceitos, porque só a estrutura tem significado.
- Linhas vazias e linhas só de comentário (`#`) não abrem nem fecham níveis.
- Comentário no fim da linha pertence à linha. Texto entre aspas nunca é indentação.
- Uma linha lógica não continua na linha seguinte. Na linha de uma seção cabe **um** item
  (`tem repositório`, `pertence a grupo opcional`, `pode fechar`); listas usam um item por
  linha, porque a vírgula já pertence aos modificadores (`email obrigatório, único e privado`)
  e uma lista na mesma linha seria ambígua.

### Gramática (sobre os símbolos do layout)

`NL` é fim de linha; `ABRE`/`FECHA` são símbolos abstratos: a abertura e o fechamento de
um nível, calculados pela pilha de níveis do layout (não são tokens do lexer). As palavras podem
ser escritas com ou sem acento. A camada de intenção entende hoje só palavras em português: uma
linha em outra língua é erro que mostra as palavras em português (a decisão sobre outras
línguas é a [GEP 0007](gep/0007-idiomas-da-intencao.md)).

```ebnf
programa     = { item } ;
item         = frase_plana | bloco_dado | bloco_pagina | construcao_do_sistema ;
bloco_dado   = nome_do_dado NL ABRE secao { secao } FECHA ;
bloco_pagina = "página" Nome NL ABRE { secao_pagina } FECHA ;

secao        = "tem" ( item NL | NL ABRE { linha_de_campo NL } FECHA )
             | "pertence a" ( alvo NL | NL ABRE { alvo NL } FECHA )
             | "começa" estado NL
             | "pode" ( capacidade NL | NL ABRE { capacidade NL } FECHA )
             | "regras" NL ABRE { regra } FECHA
             | "acesso" NL ABRE { bloco_ator } FECHA
             | "permita" NL ABRE { permissao NL } FECHA
             | "integração" NL [ ABRE "nome" Texto NL FECHA ]
             | ("quando" | "antes de") verbo NL ABRE instrucoes FECHA
             | frase_do_dado NL [ ABRE { linha NL } FECHA ] ;

bloco_ator   = [ "somente" ] ator { "ou" ator } NL ABRE { acao NL } FECHA ;
acao         = verbo [ "seu" | "sua" | "seus" | "suas" ] [ alvo ] ;
```

`nome_do_dado` é o nome no plural usado em `tenha` (o bloco também **declara** o dado, dispensando
`tenha`). `frase_do_dado` são as frases cujo sujeito é o próprio dado (tabela abaixo). Uma seção
fora desta lista, uma seção vazia ou uma palavra de outro contexto são erros que listam as seções
válidas. `construcao_do_sistema` são as construções sem sujeito único (`tenha papeis`,
`tenha login`, `login …`, `crie sistema`, `importar`, `vocabulário da integração`,
`integração em`, `mensagens em`, `escopo`, `ao iniciar`, `tenha busca geral em`,
`tenha administrador inicial`, `endereços reservados`, `traduza`).

### Tabela de seções (dentro de um dado D)

| Seção e conteúdo | Frase plana equivalente |
| --- | --- |
| `tem` + campos, relações e pessoas (um por linha) ou `tem repositório` (um item na mesma linha) | `cada d tem` + linhas / `d tem repositório` |
| `pertence a` + `grupo opcional`, `cliente como dono` | `d pertence a grupo opcional` |
| `começa aberta` | `d começa aberta` |
| `pode` + `fechar`, `reabrir`, `ser confidencial` | `d pode fechar` … |
| `regras` › `quem cria vira owner` | `quem cria d vira owner` |
| `regras` › `precisa de pelo menos um owner` | `todo d precisa ter pelo menos um owner` |
| `regras` › `herda membros do grupo` | `d herda membros do grupo` |
| `regras` › `não pode ser mais visível que o grupo` | `d não pode ser mais visível que o grupo` |
| `regras` › `arquivado é somente leitura` | `d arquivado é somente leitura` |
| `regras` › `mesclado é final` | `d mesclado é final` |
| `regras` › `confidencial pode ser vista por` + pessoas | `d confidencial pode ser vista por` + pessoas |
| `acesso` › `developer` › `enviar código` | `developer pode enviar código para D` |
| `acesso` › `usuario` › `criar seus` | `usuario pode criar seus D` |
| `acesso` › `maintainer` › `adicionar membros` | `maintainer pode adicionar membros` |
| `acesso` › `somente owner` › `excluir` | `somente owner pode excluir D` |
| `acesso` › `somente maintainer` › `enviar código para a branch padrão` | `somente maintainer pode enviar código para a branch padrão dos D` |
| `acesso` › `somente maintainer` › `enviar código para as branches protegidas` ([GEP 0016](gep/0016-branches-protegidas.md), em teste) | `somente maintainer pode enviar código para as branches protegidas dos D` (um dado de D com `nome`; `*` vale qualquer texto) |
| `permita` › `pesquisar`, `filtrar por estado, autor` | `permita pesquisar D`, `permita filtrar D por estado, autor` |
| `integração` / `integração` › `nome "projects"` | `disponibilize D para integração [como "projects"]` |
| `quando criar` / `antes de excluir` + corpo | `quando criar d` / `antes de excluir d` |
| `recebe aprovações`, `recebe eventos do projeto` + tipos | `d recebe …` |
| `executa pipelines a cada envio de código conforme "arquivo"` | `d executa …` |
| `executam jobs` (em `runners`) | `runners executam jobs` |
| `repositório pode começar com "README.md" contendo "# {nome}"` | `repositório do d pode começar com …` |
| `singular token de acesso` | `cada token de acesso tem` (a forma que nomeia um registro, quando o plural admite duas leituras: tokens → token ou tokem; vale também quando a regra erra, como em palavras estrangeiras: `singular branch protegida`) |
| `pendência para` + campos de pessoas (um por linha) | `d gera pendência para responsaveis` (veja Pendências) |
| `renomeie nome para nome_completo` | `renomeie nome de D para nome_completo` (veja Migração) |
| `descarte fax` (GEP 0010, em teste) | `descarte fax de D` (veja Migração) |
| `guarda histórico` ([GEP 0011](gep/0011-historico.md), em teste) | `d guarda histórico` |
| `guarda leitura` ([GEP 0022](gep/0022-leitura.md), em teste) | `d guarda leitura` (abrir o contêiner lê; ele mostra `nao_lidas` a cada pessoa) |
| `usam variaveis do projeto` ([GEP 0015](gep/0015-variaveis-das-execucoes.md), em teste; nas execuções) | `pipelines usam as variaveis do projeto` |

Uma **ação sem alvo** vale para o próprio dado (a coleção: `administrar` sem alvo inclui
criar, como `administrar projetos`). Uma ação com alvo explícito (`adicionar membros`,
`ver labels`) só é aceita se o alvo for o dado ou algo que pertence a ele; para outro dado, a
regra vai no bloco dele. `seu/seus` restringe aos registros da própria pessoa.

### Página

`página Clientes` + `mostre clientes`, `permita` + ações e `20 por página` **é** `crie página
Clientes` com o mesmo corpo. `pagina "/caminho"` (com texto entre aspas) continua sendo a página
estática do nível técnico; os dois se distinguem pelo que vem depois da palavra.

Seções de página ([GEP 0002](gep/0002-secoes-de-pagina.md), aceita): uma tabela fechada de
regiões com significado de domínio. Nenhuma é obrigatória; cada uma troca só o próprio padrão;
`página Clientes` + `mostre clientes` sozinho continua produzindo a página inteira.

```ge
crie sistema Cadastro

usuarios
    tem
        nome obrigatório
        email obrigatório e único
        senha min 8

tenha login

clientes
    tem
        nome obrigatório
        email obrigatório e único
        cidade
    acesso
        usuario
            ver
            criar

página Clientes
    mostre clientes
    topo
        título "Clientes"
        texto "Todas as pessoas atendidas."
        ações
            criar "Novo cliente"
    filtros
        pesquisar
        cidade
    colunas
        nome
        email
    vazio
        título "Nenhum cliente"
        texto "Cadastre o primeiro."
        ação criar "Cadastrar cliente"
    permita
        criar
```

| Seção | Conteúdo | Regra | Padrão quando ausente |
| --- | --- | --- | --- |
| `topo` | `título "…"`, `texto "…"`, `ações` | — | título = nome dos dados |
| `ações` (em `topo`) | um verbo por linha, rótulo opcional | o verbo precisa estar no `permita` da página; hoje só `criar` cabe no topo (as ações de um registro aparecem nele) | a ação criar, se permitida |
| `filtros` | `pesquisar` ou um campo por linha | o campo precisa existir; declarar um filtro implica `permita filtrar por` (filtrar só estreita o que já se vê) | a pesquisa e os filtros do dado |
| `colunas` | um campo por linha, na ordem | o campo precisa existir e não ser privado, oculto nem secreto | os campos visíveis |
| `vazio` | `título`, `texto`, `ação verbo "rótulo"` | a ação segue a regra de `ações` | "Nenhum registro de … ainda." |

`tenha presença` ([GEP 0021](gep/0021-presenca.md), **em teste**) mostra quem está com a aplicação
aberta: as pessoas ganham `online`, nunca gravado, visto por quem pode ver a pessoa.

Uma página aberta acompanha as mudanças do que mostra ([GEP 0020](gep/0020-paginas-vivas.md),
**em teste**, sem sintaxe): cada mudança salva avisa quem pode ver o registro, e só as regiões
vivas da página (lista, indicadores, detalhes, filhos) são trocadas, sem mexer no que a pessoa
está digitando. O transporte é do runtime; o programa não o nomeia.

A página pede; o domínio decide: uma ação só aparece para quem pode fazê-la, como qualquer botão
de `permita`. Um rótulo sozinho (`"Novo cliente"`) é erro: um texto não é uma ação. As seções
valem na página da coleção (não nas listas aninhadas). `ge explain pagina Clientes` mostra cada
seção como declarada ou padrão e quem vê cada ação. Indicadores (`indicadores` ›
`total de issues abertas`) estão em teste na [GEP 0012](gep/0012-indicadores.md), não normativa
até a decisão do mantenedor: cada número conta só o que quem vê a página pode ver. Gráficos,
somas e médias ainda não existem: cada um depende de uma GEP própria.

### Fusão e conflitos

O mesmo dado pode ser descrito em vários blocos e arquivos, e em forma plana ou hierárquica:
os fatos se somam. O mesmo fato repetido não muda nada. Um valor que só pode ter um valor
(estado inicial, nome de integração) declarado com valores diferentes é erro que mostra as
duas origens. A ordem dos arquivos não muda o resultado.

### Erros

Linhas que nenhuma construção reconhece são erro, nunca ignoradas. Todo erro de layout ou de
contexto diz **o que aconteceu, onde (arquivo:linha e caminho do bloco), por que e como
corrigir**, e, dentro de um bloco, a frase plana equivalente do que foi entendido:

```text
backend/projetos.ge:12 — "enviar código" não tem um nível aberto acima dele.
Por quê: cada nível abre embaixo da linha de cima; este recuo (6) não corresponde a nenhum
bloco aberto (4 = acesso, 8 = developer).
Como corrigir: recue para 12 espaços para ficar dentro de "developer".
Equivale a: developer pode enviar código para projetos
```

## O que existe

| Frase | Significado |
|-------|-------------|
| `tenha clientes, pedidos` | dados persistentes; o singular vem da regra fixa do plural (`clientes→cliente`, `papeis→papel`, `itens→item`). Se houver outra forma possível (`tokens→token`), a forma usada em `cada … tem` decide. |
| `cada cliente tem` + linhas | campos: `nome [tipo] [modificadores] [começa com valor]` |
| `cliente tem pedidos` | relação de um cliente com seus pedidos; cada pedido exige um cliente (a referência interna é derivada) |
| `pedido pertence a cliente [opcional] [como dono]` | relação com cliente, opcional ou nomeada como dono; referência validada |
| `grupo tem subgrupos` | hierarquia; a referência ao pai é interna |
| `endereço dentro do grupo pai ou do criador` (linha de `X tem`) | `endereco` = endereço do primeiro contêiner definido + `/` + `caminho` (`empresa/web/app`); calculado, nunca aceito da entrada; único entre todos os endereçados e os nomes das pessoas; renomear um contêiner (ou a pessoa) atualiza o que está dentro; repositórios passam a ser servidos pelo endereço |
| `endereços reservados` + nomes | nomes de primeiro nível que o produto guarda para si; o runtime também reserva as rotas do próprio app (`api`, `entrar`, `cadastro`, `oauth`, páginas, rotas declaradas). Todo segmento de endereço — e o nome das pessoas que dividem esse espaço — usa letras, números, `_ . -`, começa por letra, número ou `_`, não termina em `.`, não tem `..` nem termina em `.git`; até 255 caracteres |
| `tenha busca geral em projetos, issues e merge requests` | um único ponto de busca (`/_ge/api/busca?tipo_busca=issues&q=…`; nomes externos pelo vocabulário) responde com a listagem do tipo escolhido: mesma pesquisa, filtros, páginas e regra de quem vê. Cada tipo precisa de `permita pesquisar` |
| `labels por nome` (linha de `issue tem`) | lista escrita e lida pelo nome (`"bug,ux"` → `["bug","ux"]`), procurado entre os itens do mesmo pai (labels do projeto); nome novo cria o item quando a pessoa pode criá-lo ali; `?labels=bug` filtra pelo nome. Itens de outro pai nunca entram |
| `runners executam jobs` | trabalho remoto: cada runner (um `token secreto`, `ativo`, opcionalmente `pode pertencer a projeto`) pega trabalhos pendentes com reserva renovável; reserva vencida devolve o trabalho à fila (3 tentativas, depois falha); cancelamento visível ao executor; token temporário (só do próprio trabalho, lê o repositório só enquanto executa). Vale para qualquer dado: `trabalhadores executam conversoes`. Protocolos externos falam com isso por um adaptador em `integracoes/` usando `trabalho_remoto.*` |
| `projeto executa pipelines a cada envio de código conforme "arquivo.yml"` | o arquivo, no formato nativo (`estagios`, `etapas` com `comandos`, `depois`, `quando: automatico / manual / sempre`, `pode_falhar`, `imagem`, `artefatos`), vira uma execução por envio; um adaptador pode ler outro formato com `traduza arquivos de execução com f` e dar variáveis às etapas com `traduza variáveis das etapas com f` (só em `integracoes/`) |
| `quem cria projeto vira owner` | quem cria vira membro com esse papel — exceto quando o dado herda membros de um pai e foi criado dentro dele (os membros já vêm do pai) |
| `todo grupo precisa ter pelo menos um owner` | ninguém remove nem rebaixa o último membro com esse papel (ou superior) |
| `repositório do projeto pode começar com "README.md" contendo "# {nome}"` | ao criar com `iniciar_repositorio` (nome externo pelo vocabulário), o repositório nasce com esse arquivo; `{campo}` vira o valor do registro |
| `projeto tem repositório` | cada registro tem um repositório Git criado e removido com ele e servido em `/<campo único>.git`; branches, tags, commits, árvore, arquivos e comparação seguem as regras de baixar e enviar código |

**Tipo pelo nome** (quando não há tipo): `email`→email · `senha`/`password`→senha protegida ·
`telefone`→telefone · `foto`/`avatar`/`imagem`→imagem · `descricao`/`description`→texto longo ·
`visibilidade`/`visibility`→visibilidade · `preco`/`valor`→dinheiro · `quantidade`/`estoque`→inteiro ·
`admin`/`ativo`/`arquivado`/`confidencial`/`pode_*`/`is_*`→booleano · demais→texto.
`arquivo`/`anexo`→arquivo. Um campo só guarda um arquivo de verdade (envio, download para quem vê
o registro, limites, remoção) quando o tipo é declarado — `anexo arquivo`, `foto imagem` —, em
teste na [GEP 0014](gep/0014-arquivos.md); pelo nome sozinho, continua guardando texto.
`começa com 10` → inteiro; `começa com verdadeiro` → booleano.

**Inferência é conveniência, não verdade absoluta.** Uma declaração explícita sempre
vence a heurística: `valor texto` é texto, mesmo que `valor` normalmente sugira dinheiro.
`ge explain` deve informar o tipo inferido, sua origem e o motivo. Uma inferência ambígua
não pode vencer declaração explícita nem ser resolvida por adivinhação: deve gerar erro
educativo. O padrão “demais → texto” só vale quando nenhuma relação ou outro significado
concorrente torna o contexto ambíguo. Precedências ainda não comprovadas precisam de testes.

Relações são escritas como `issue tem autor`, `issue tem responsaveis` e
`pedido pertence a cliente`. IDs podem aparecer na persistência, inspeção ou integração,
mas não são exigidos do autor do domínio.

**Modificadores**: `obrigatório`, `único`, `privado` (só o dono e administradores veem),
`oculto` (nunca aparece), `imutável`, `min N`, `max N`, `formato "regex"`, `valida função`,
`segredo prefixo "x"` (gerado, guardado como hash, mostrado só na criação),
`expira em N dias`, `revogavel`. `senha sem criptografia` é recusado.

**Garantias automáticas**: senha com bcrypt e nunca devolvida; e-mails normalizados;
erros de validação reunidos por campo (`{"message": {"email": ["já está em uso"]}}`);
excluir remove o que pertence ao registro; cada alteração (criar, editar, excluir, ações)
roda numa transação junto com seus `antes de`/`quando`: se algo falha ou é recusado, nada
fica — nem registros criados pelos hooks, nem números consumidos, nem eventos na fila.
Excluir uma pessoa leva junto o que pertence a ela (referência obrigatória à pessoa, como
`pertence a usuario`) e as participações dela; o que só a nomeia (referência opcional: autor,
quem concluiu) fica, sem o nome. A regra do último dono continua valendo: excluir a última
pessoa com o papel mínimo de algo é recusado.

Mudar os campos declarados nunca perde dados em silêncio: um campo novo vira coluna (a
falha é erro, não é ignorada); uma coluna que ainda tem valores e deixou de ser declarada é
avisada na partida, e os valores continuam nela; se ao mesmo tempo aparece um campo novo, a
partida para, porque pode ser um rename e Germanio não infere renames (veja Migração); um
campo `único` vale também no banco, mesmo quando se tornou único depois (valores repetidos
impedem a partida com erro educativo).

**Efeitos externos depois de salvar.** O que age fora do sistema dentro de uma alteração —
`chamar` com POST, PUT, PATCH ou DELETE, `webhook_enviar`, `telegram_enviar`, `slack_enviar`,
`discord_enviar`, `sms_enviar`, `mercadopago_link` — nunca acontece com a transação (e a trava
de escrita do banco) presa. Germanio registra o efeito e o executa depois do commit, na ordem
em que foi pedido, antes de responder:

- se a alteração é desfeita (recusa, erro, falha do próprio commit), nenhum efeito acontece;
- se um efeito falha depois do commit, a alteração continua salva: a falha vai para o log e os
  efeitos seguintes rodam;
- um efeito lento atrasa só a resposta de quem pediu, nunca as escritas dos outros;
- a resposta de um efeito não existe dentro da alteração: usá-la (`r = chamar(url, "POST",
  …)`) é erro de compilação no hook e erro de execução numa função chamada por ele. Para
  trabalhar com a resposta, use `tarefas.enfileirar("funcao", dados)`, que roda fora da
  alteração, com novas tentativas;
- leituras (`chamar` com GET, as funções de IA) não são efeitos e rodam na hora;
- `ge explain <dado>` lista, em cada hook, os efeitos na ordem em que acontecem.

Eventos entregues a integrações (`recebe eventos`) já passavam pela fila persistente, gravada
na mesma transação.

## Estados, condições e pessoas

```ge
issue tem
    numero por projeto          # 1, 2, 3… dentro de cada projeto
    titulo obrigatório até 255
    autor                       # uma pessoa (dona do registro)
    responsaveis                # várias pessoas
    labels                      # dados já conhecidos (do projeto)
    comentarios                 # o que pertence à issue

issue começa aberta
issue pode
    fechar                      # → fechada; registra fechada_em e fechada_por
    reabrir                     # re-/des- voltam ao estado inicial
    ser confidencial            # condição sim/não

issue confidencial pode ser vista por
    autor
    responsaveis
    reporter ou superior

projeto pode ser arquivado
projeto arquivado é somente leitura
grupo não pode ser mais visível que o grupo pai
quem cria grupo vira owner
somente maintainer pode enviar código para a branch padrão dos projetos
```

- `X <condição> é somente leitura`: enquanto a condição vale, o registro só muda para
  desfazê-la, nada que pertence a ele é criado, editado, excluído ou muda de estado, e o
  repositório não recebe código; ler e excluir o próprio registro continuam possíveis.
- `X pode <verbo>` **sem objeto** é capacidade do dado; **com objeto** (`usuario pode criar issues`) é permissão.
- Estados mudam só por ação; editar não altera `estado` nem carimbos.
- Particípio segue o gênero do estado inicial (aberta → fechada; ativo → bloqueado).
- Pessoas reconhecidas em `tem`: autor, dono, criador, responsavel, revisor, aprovador… (uma);
  responsaveis, revisores, participantes, seguidores… (várias).
- Quando dois `tem` citam o mesmo dado (`projeto tem labels`, `issue tem labels`), o ancestral é o dono
  e o descendente guarda uma lista.
- `comentar issues` = criar comentarios da issue (o verbo começa o nome do dado filho).
- Nomes de campos ignoram acento (`descrição` = `descricao`).

## Quem pode fazer o quê

```ge
tenha papeis
    guest 10
    developer 30
    maintainer 40
    owner 50

grupo tem membros com papel
projeto herda membros do grupo

usuario pode criar grupos
guest pode ver projetos
developer pode enviar código para projetos
maintainer pode administrar projeto
somente owner pode excluir projetos
usuario pode editar seu perfil
cliente pode ver seus pedidos
todos podem ver produtos
```

- Papéis vão do menor para o maior; o que vale para um papel vale para os superiores.
  Um papel declarado vence as palavras de pessoa: com `dono 50` em `tenha papeis`,
  `dono pode …` fala do papel, não de quem possui o registro.
- `administrador` = pessoa com `admin` verdadeiro (ou `papel` "administrador"); tem autorização ampla, mas deve respeitar validações e invariantes de integridade.
- `todos` = qualquer visitante; `usuario` (o dado do login) = qualquer pessoa conectada;
  `membro` = qualquer papel no registro.
- `seu/sua/seus/suas` = registros que pertencem à pessoa (ou a própria pessoa: `perfil`).
- `administrar X` = ver, editar e excluir X, e criar/editar/excluir o que pertence a X e seus membros.
- Ações sinônimas: ver/listar/mostrar · criar/cadastrar/adicionar · editar/alterar ·
  excluir/remover · pesquisar/buscar. Transições como `bloquear` são declaradas por `usuario pode bloquear`; não exigem hook manual.
  Ação específica sem semântica declarativa só usa `quando` como escape hatch do nível 3.
  Embutidas: `sair` (deixa de ser membro), `revogar` (dados revogáveis), `baixar código`/`enviar código`.
- Um campo `visibility` (`private`/`internal`/`public`) libera ver e baixar código para todos
  (público) ou para quem está conectado (interno).
- Dentro de algo que tem membros, regras genéricas (todos, qualquer pessoa conectada) só valem se esse algo
  for público ou interno; caso contrário só papéis decidem — para ver, criar, editar ou excluir.
- Pesquisa e filtros mudam como se encontra, nunca quem vê.
- Uma referência (lista ou única) a algo que pertence a um pai que o registro também tem
  — a label ou o milestone do projeto de uma issue — só aceita itens do mesmo pai; o resto
  é tratado como inexistente. Pessoas e referências de sistema não contam como pai.
- Ninguém concede papel acima do próprio.
- Nada que pertence a uma pessoa é criado em nome de outra, exceto por administradores.
- Registros invisíveis respondem "não encontrado", nunca "proibido".

`permita criar projetos` libera para qualquer pessoa conectada (ou qualquer visitante se não
há login). `permita filtrar clientes por cidade` e `permita pesquisar clientes` habilitam filtro e busca.

## Pendências

`pendência para` ([GEP 0009](gep/0009-pendencias.md), aceita), no bloco do dado, ou
`issue gera pendência para responsaveis`: quem passa a estar num campo de pessoas (uma pessoa ou
uma lista) recebe uma pendência do registro (com o título só se ela puder ver o registro); quem
faz a mudança não recebe para si; quem sai do
campo perde as pendências abertas; excluir o registro exclui as pendências. Tudo acontece na
transação da mudança. As pendências são um dado comum, `pendencias` (motivo, recurso, dono;
começa aberta; pode concluir; cada pessoa vê, conclui e exclui só as suas), que o Germanio
escreve quando o programa não o declara (`ge explain pendencias` mostra a origem). Para
mostrá-las, a aplicação declara `página Pendências` + `mostre pendências`.

Em `pendência para`, `mencionados` ([GEP 0017](gep/0017-mencoes.md), **em teste**) dá pendência a
quem é escrito como `@nome de usuário` nos textos do registro, se essa pessoa puder vê-lo.

`tenha avisos por e-mail` ([GEP 0013](gep/0013-avisos-por-email.md), **em teste**, não normativa
até a decisão do mantenedor) manda cada nova pendência também por e-mail ao dono, depois de a
mudança ser salva, com a configuração de e-mail da recuperação de senha.

## Migração

O banco acompanha o programa sozinho quando isso não pode perder nada: um campo novo vira uma
coluna nova, um campo que passa a ser `único` ganha o índice único (e valores repetidos impedem
a partida, com erro educativo). **Germanio nunca infere um rename.** Quando uma decisão pode
alterar o significado ou destruir dados, o programa a declara:

```ge
crie sistema Clientes

tenha clientes

clientes
    tem
        nome_completo
        email
    renomeie nome para nome_completo
```

- `renomeie nome para nome_completo` troca o nome da coluna, com os dados, numa transação: todos
  os renames de um dado acontecem, ou nenhum. `nome_completo` precisa estar em `tem` e `nome`
  não. Num banco novo, ou já migrado, a frase não faz nada; ela pode ficar no programa.
- `descarte fax` ([GEP 0010](gep/0010-descarte-de-campo.md), **em teste**, não normativa até a
  decisão do mantenedor) diz que o campo saiu de propósito: a aplicação deixa de usá-lo, e os
  valores antigos continuam guardados no banco (nada é apagado).
- Se um campo **com dados** some do programa enquanto outro aparece, a partida para e explica:
  pode ser um rename, pode ser remover um e criar outro, e os nomes sozinhos não provam nada.
  O erro sugere as duas frases. `ge check` faz a mesma verificação no banco existente, sem mudar
  nada, e anuncia os renames que a próxima partida aplicará.
- Um rename não sobrescreve dados (o campo novo já com valores é recusado) e não muda o tipo ao
  mesmo tempo (faça em duas etapas). Renames em cadeia e dois campos virando o mesmo são
  recusados na compilação.
- `ge explain <dado>` mostra os renames e descartes declarados, com a origem.

## O que deve acontecer

O padrão é declarar invariantes e transições:

```ge
todo grupo precisa ter pelo menos um owner

usuario começa ativo
usuario pode
    bloquear
    desbloquear
```

A regra do grupo deve impedir remover ou rebaixar o último owner e qualquer operação que
produza um grupo sobrevivente sem owner. Mantém-se a hierarquia já documentada: papel
superior também satisfaz o mínimo. A verificação deve ser atômica, inclusive sob operações
concorrentes, saídas de membros, mudanças indiretas e operações em lote. Administradores
não devem contornar invariantes de integridade. Excluir o próprio grupo não exige manter
membros de um grupo que deixa de existir. Criação e herança precisam respeitar a mesma regra;
limites da implementação devem ser documentados, nunca escondidos em hooks do produto.

Não use contagem manual de membros num `antes de excluir membro` para substituir essa
regra. Não use `usuario.atualizar(usuario.id, {state: "blocked"})` para implementar bloquear.
Também não concatene manualmente caminhos quando a declaração de endereço já os deriva.

### Lógica específica — nível 3, último recurso

Hooks continuam disponíveis para regras realmente específicas. O esquema abaixo é
**ilustrativo, não executável**: `...` representa uma regra a definir, não sintaxe nova.

```text
antes de criar projeto
    ... regra específica não coberta por declaração ...
quando criar projeto
    ... efeito específico ...
antes de enviar código para projeto
    para cada mudanca em atualizacoes
        ... regra específica ...
ao iniciar
    ... inicialização específica ...
```

Variáveis disponíveis: `atual` (quem age), o registro pelo nome do dado e como `registro`,
`entrada` (o que a pessoa enviou), `dados` (em `antes de`: o que será salvo),
`atualizacoes` (em código enviado). Hooks não autorizam bypass de políticas, transições
ou invariantes. A transação das alterações e dos hooks é preservada.

## Login

```ge
tenha login
tenha cadastro
login usa username e email
login exige state "active"
login bloqueia após 10 tentativas por 10 minutos
```

Por padrão, o login bloqueia a conta por 10 minutos depois de 10 senhas erradas seguidas, e a
senha certa é recusada enquanto o bloqueio dura; `login bloqueia após N tentativas por M
minutos` só ajusta os números. Além disso, um mesmo endereço que erra 50 logins em 10
minutos, em quaisquer contas, espera antes de tentar de novo (resposta 429); logins certos não
contam. Nenhuma aplicação com login fica sem essas proteções.

`tenha recuperação de senha` ([GEP 0008](gep/0008-recuperacao-de-senha.md), aceita; implica o
login, como `tenha cadastro`) oferece `/esqueci` e `/redefinir`:

- a resposta de `/esqueci` é a mesma exista ou não a conta;
- o link vai por e-mail para o endereço público declarado (`GERMANIO_URL_PUBLICA`), nunca para o
  `Host` da requisição; o token é aleatório (32 bytes), guardado só como SHA-256, vale **uma
  hora** e **uma vez**, mesmo com o mesmo link enviado várias vezes ao mesmo tempo (o link é
  consumido na transação que troca a senha; uma senha recusada o devolve); usá-lo invalida
  todos os links da pessoa e limpa o bloqueio do login;
- o e-mail sai depois de o token estar salvo e fora do caminho da resposta: o tempo da resposta
  não diz se a conta existe;
- a senha nova obedece às regras do campo de senha;
- no máximo um e-mail por conta a cada 2 minutos, e os pedidos contam no limite por endereço
  do login;
- o e-mail vem do ambiente (`GERMANIO_SMTP_HOST`, `_PORTA`, `_USUARIO`, `_SENHA`, `_REMETENTE`,
  ou `GERMANIO_CORREIO_PASTA` em desenvolvimento), nunca do `.ge`; sem ele, a página diz que a
  recuperação ainda não está disponível.

`tenha administrador inicial "root"` cria a primeira pessoa administradora (login `root`)
quando ainda não existe ninguém e o servidor recebeu `GERMANIO_ADMIN_SENHA`
(e, opcionalmente, `GERMANIO_ADMIN_EMAIL`); a senha nunca aparece no `.ge`.

O runtime oferece `entrar`, `sair`, `cadastro`, sessão com CSRF, tokens (cabeçalho, `Bearer`,
HTTP Basic para git) e `oauth/token`. O cadastro nunca aceita campos como `admin`.

A configuração de protocolos externos, como `login aceita tokens de acesso no cabeçalho
"PRIVATE-TOKEN"` ou `login aceita oauth por 2 horas`, pertence à configuração técnica de
compatibilidade separada do domínio comum. `login exige state "active"` acima é uma
configuração explícita de campo/valor; não exige atualização manual de estados.

## Integração e compatibilidade — nível avançado

A exposição de entidades é declarativa; o vocabulário externo abaixo é configuração de
compatibilidade separada. Não copie nomes externos para os campos do domínio.

```ge
disponibilize para integração
    issues
    comentarios como "notes"

vocabulário da integração        # só nomes externos; o domínio segue em português
    titulo é "title"
    aberta é "opened"
    fechar é "close"
    papel é "access_level"       # papéis viajam como seus níveis
```

## Estados e transições: contrato genérico

`issue começa aberta` com `fechar` e `reabrir` usa o mesmo mecanismo de estados que ticket,
pedido, fatura, pipeline, tarefa e usuario. Nenhuma keyword ou mecanismo pode depender do
nome `issue`. A execução deve controlar estado, quem agiu e quando, sem campos técnicos
obrigatórios no domínio. Fechar registra os metadados de fechamento; reabrir retorna ao
estado inicial e limpa esses metadados. `ge explain` deve revelar campos e relações internas
(inclusive nomes físicos como `fechada_por_id`, quando usados).

As regras existentes de gênero e dos prefixos `re-`/`des-` são conveniências determinísticas,
não interpretação livre de verbos. Verbo ou destino ambíguo deve produzir diagnóstico;
a tabela de transições válidas e eventuais configurações explícitas devem ser documentadas.
Não invente sintaxe para destinos arbitrários neste documento.

## Inspeção: `ge explain`

Toda inferência deve ser inspecionável. Para cada fato, `ge explain` mostra a origem
(arquivo, linha e caminho hierárquico, como `projetos › acesso › developer`) e a frase plana
equivalente. `ge explain issue` deve mostrar campos declarados,
campos e tipos inferidos com motivo, campos internos, relações, estados, transições,
permissões, visibilidade, validações, capabilities ativadas, integrações e persistência derivada.
Nunca deve revelar valores secretos. Exemplo **conceitual**, não transcrição da CLI atual:

```text
issue
Declarado: titulo, descrição, autor, responsaveis
Inferido: titulo → texto (regra de tipo padrão); autor → usuario (relação de pessoa)
Interno: id, criado_em, atualizado_em, estado, fechada_em, fechada_por
Capabilities: entidade persistente, estados, transições
```

A implementação em `tooling/explicar/explicar.go` já descreve campos, relações, estados,
permissões e operações; a separação completa declarado/inferido/interno e a justificativa
de cada inferência são requisitos normativos, não cobertura integral comprovada nesta revisão.
Também mostra, em cada hook, os efeitos externos na ordem em que acontecem depois de salvar, e
a migração declarada (`renomeie`, `descarte`) com a origem.

## Verificação: `ge check`

`ge check app.ge` deve detectar, quando estaticamente possível: indentação inválida (tab,
recuo sem pai), seção fora do lugar ou desconhecida, linha não reconhecida, valor único
declarado com valores diferentes, relações inválidas,
entidades desconhecidas, ações inexistentes, permissões contraditórias, estados ou transições
impossíveis, inferências ambíguas, capabilities incompletas, integrações inválidas e regras
contraditórias. Campos não utilizados devem gerar aviso quando a análise puder demonstrá-lo;
uso por integração ou reflexão não pode ser ignorado. Sem certeza suficiente, não adivinhe.

Erros de validade impedem aceitação; avisos descrevem situações válidas que merecem revisão.
Condições dependentes de dados ou concorrência exigem verificação em runtime e testes.
`check` não prova ausência de todos os erros nem substitui testes de execução.
Hoje o parser/resolver faz validações estruturais; `explicar.Verificar` avisa sobre dados
inacessíveis e ações que herdam permissão de edição. Com um banco já existente, `ge check`
também planeja a migração sem mudar nada: anuncia os renames que a próxima partida aplicará e
para, com o erro educativo, quando um campo com dados sumiu enquanto outro apareceu. Usar a
resposta de um efeito externo dentro de um hook é erro de compilação. A lista normativa
completa acima não é uma alegação de que todos os diagnósticos já foram implementados.

## Erros educativos

Todo erro deve explicar **o que aconteceu, onde, por que e como corrigir**, com arquivo e
posição quando disponíveis. Exemplo ilustrativo para um contexto realmente ambíguo:

```text
app.ge:8 — Não entendi o tipo de "valor" neste contexto.
Normalmente "valor" sugere dinheiro, mas há interpretações concorrentes aqui.
Declare explicitamente: valor dinheiro
ou: valor texto
```

O nome `valor` sozinho não deve causar erro: a heurística continua válida quando não há
conflito. Não exponha stack traces ou detalhes de banco como única explicação ao iniciante.

## Integração (detalhes)

`disponibilize projetos para integração como "projects"` + `integração em "/api/v4"`
publica listar (busca, filtros, paginação com `X-Total`), ver (por id ou campo único),
criar, editar, excluir, ações próprias, filhos (`/projects/:id/members`) e, para dados com
repositório, `repository/{branches,commits,tree,files,compare}`. A pessoa conectada
aparece em `/<prefixo>/<singular>` (`/api/v4/user`). As páginas usam as mesmas operações em `/_ge/api`.

## Servidor (operação)

Nada disto aparece no `.ge`; é configuração de quem hospeda a aplicação.

| Variável | Para quê |
|----------|----------|
| `GERMANIO_SEGREDO` | chave das sessões e tokens (≥ 32 bytes); sem ela, uma chave aleatória por processo |
| `GERMANIO_SQLITE` | arquivo do banco SQLite |
| `GERMANIO_GIT_RAIZ` | pasta dos repositórios |
| `GERMANIO_EXECUTOR` | `local` ou `docker` para executar etapas neste servidor (padrão: nenhum) |
| `GERMANIO_JOB_TIMEOUT` | tempo máximo de uma etapa |
| `GERMANIO_PROXIES_CONFIAVEIS` | IPs/CIDRs dos proxies reversos; só deles se aceita `X-Forwarded-For`/`X-Real-IP`/`CF-Connecting-IP` (padrão: nenhum — o IP é o da conexão) |
| `GERMANIO_ADMIN_SENHA` / `GERMANIO_ADMIN_EMAIL` | senha (e e-mail) do `administrador inicial`, usados só enquanto não existe ninguém |
| `GERMANIO_PERMITIR_REDE_LOCAL` | permite webhooks/chamadas para endereços internos (desligado por padrão, proteção SSRF) |

## Trabalho remoto e teste de generalização

`runners executam jobs` ativa uma capability genérica de trabalho remoto, também utilizada
por `trabalhadores executam conversoes`. Internamente pode fornecer executor, credencial,
claim atômico, lease, heartbeat, retry, cancelamento, token temporário, log incremental,
resultado, timeout, concorrência e acesso temporário a recurso. O acesso deve ter escopo
mínimo e expirar com a reserva; recurso Git só existe quando a aplicação declara repositório.

O runtime não deve conhecer GitLab Runner. Funcionar somente com ele reprova a generalidade,
mesmo com tipos renomeados. Deve existir teste com executor sem GitLab, Git ou CI.
`runtime/trabalho_remoto_test.go` contém testes genéricos para essa capability.

Antes de adicionar mecanismo ao runtime, pergunte: **isso faria sentido em pelo menos outra
aplicação sem modificar sua essência?** Git, HTTP, executor remoto e token temporário são
mecanismos genéricos. GitLab Runner, `CI_JOB_ID` e `access_level` são contratos específicos
de adaptador. Regras próprias do produto pertencem à aplicação.

## Detector de complexidade acidental

| Repetição no domínio | Abstração a investigar antes de escrever mais lógica |
| --- | --- |
| buscar X → se não existe → erro | resolução e validação de relação |
| atualizar estado → salvar data → salvar usuário | transição |
| verificar papel → comparar nível → recusar | política declarativa |
| listar → paginar → serializar → responder | coleção ou integração |
| IDs, payloads, headers, status HTTP | detalhe técnico vazando do adaptador |

## Teste do leigo e teste do profissional

Para cada construção dos níveis 1 e 2: **uma pessoa que nunca programou consegue ter uma
ideia razoável do que esta frase significa?** Ela precisa entender a intenção, não o mecanismo.
Se não, procure uma abstração mais alta.

Todo comportamento inferido deve ser determinístico, documentado, testável, inspecionável
e sobrescrevível quando necessário por configuração ou níveis avançados documentados.
Personalização não pode desativar silenciosamente autorização ou invariantes. O iniciante
usa `issue pode fechar`; o profissional inspeciona e controla o comportamento sem sacrificar
as garantias comuns.

## Evolução e refatoração retroativa

Quando uma aplicação não puder expressar uma necessidade, não implemente imediatamente
sua feature em Go. Siga esta sequência:

1. Identificar a intenção humana e o conceito geral.
2. Verificar se uma capability existente já resolve.
3. Se faltar mecanismo genérico, definir seu contrato e implementar no core.
4. Expor sintaxe simples, formal e determinística para `.ge`.
5. Testar a capability isoladamente, inclusive em outro domínio.
6. Usar na aplicação real e refatorar o código antigo que ficou desnecessário.

Esta sequência orienta implementações futuras; esta revisão altera somente documentação.
**Go implementa mecanismos. .ge implementa produtos.** Quando uma abstração nova substitui
boilerplate antigo, ele deve ser refatorado. O runtime ganha capabilities genéricas enquanto
as aplicações perdem código repetitivo; funcionar não justifica conservar complexidade histórica.

## Testes normativos

Exemplos normativos devem virar testes de parser, validação e execução conforme aplicável.
A tabela é um contrato de aceitação, não uma declaração de que todos os testes estão verdes.

| Construção | Casos obrigatórios |
| --- | --- |
| `todo grupo precisa ter pelo menos um owner` | dois owners permitem remover um; um owner não permite removê-lo nem rebaixá-lo; adicionar outro permite remover o anterior; concorrência não permite remover ambos; criação, herança e mudanças indiretas preservam a regra |
| `issue começa aberta`, `issue pode fechar`, `issue pode reabrir` | inicia aberta; fechar muda estado, registra autor e instante; reabrir volta ao inicial e limpa metadados; operação sem permissão é recusada |
| `runners executam jobs` | mesma capability com executor sem GitLab; claim concorrente, expiração, retry, cancelamento, log e token limitado ao trabalho |
| `valor texto` | declaração explícita vence heurística; explain informa a origem; contexto ambíguo gera diagnóstico educativo |
| adaptador | traduz protocolo; não declara domínio, páginas nem regras do produto; não contorna permissões |
| sintaxe hierárquica | cada linha da tabela de seções produz o mesmo aplicativo que sua frase plana; um app inteiro (GitLab) em forma hierárquica passa nos mesmos testes de execução |
| layout | tab e recuo sem pai são erros com posição; linhas vazias e comentários não mudam a estrutura; EOF fecha os níveis |
| contexto | seção desconhecida, seção fora do lugar e linha não reconhecida são erros com sugestão; ação com alvo de outro dado é recusada |
| fusão | blocos do mesmo dado em arquivos diferentes se somam; valor único com dois valores é erro com as duas origens |
| `ge fmt` | idempotente; 4 espaços por nível; preserva comentários; os fatos antes e depois são idênticos |

## Decisões e limites que exigem acompanhamento

- Completar a proveniência de inferências no AST e sua apresentação no `ge explain`.
- Definir e testar a cobertura dos diagnósticos de contradição e ambiguidade em `ge check`.
- Precisar a configuração explícita de transições fora das convenções atuais, sem inventar
  verbos ou destinos silenciosamente; manter estado e metadados coerentes.
- Invariante de papel mínimo: comprovado em execução por `runtime/papel_minimo_test.go`
  (domínio sem GitLab) — remover, sair e rebaixar; administrador não contorna; dois donos
  permitem remover um; adicionar outro libera o anterior; criação exige titular; herança
  conta os titulares do pai; mover para fora do pai e excluir a pessoa são recusados quando
  deixariam o registro sem titular; saídas simultâneas deixam exatamente uma passar (SQLite
  serializa as transações). Pendente: teste da trava `FOR UPDATE` em PostgreSQL/MySQL e de
  operações em lote (ainda não existem).
- Verificar a fronteira entre configuração externa e adaptador para cada construção já
  suportada. Não presumir que mover um arquivo para `integracoes/` torna qualquer sintaxe válida.

### Pendências da sintaxe hierárquica

- Verbos que ligam e desligam uma condição sem máquina de estados (`pode` › `arquivar`,
  `restaurar`) ainda não existem; hoje a forma é `pode` › `ser arquivado`.
- `gráfico`, `lista` e indicadores como `total de clientes` nas páginas seguem como direção,
  dependentes de uma GEP de agregados (as seções `topo`, `filtros`, `colunas` e `vazio` são
  norma: Página).
- `ge check --nivel N` (avisar construções acima de um nível) ainda não existe.

## Checklist de design de novas construções

- [ ] Descreve intenção em vez de implementação?
- [ ] Um leigo consegue aproximadamente entender?
- [ ] É determinística?
- [ ] O comportamento está documentado?
- [ ] `ge explain` mostra o que foi inferido e por quê?
- [ ] `ge check` detecta uso inválido quando possível?
- [ ] É genérica ou pertence a uma aplicação?
- [ ] Está na camada correta?
- [ ] Evita conhecimento técnico desnecessário?
- [ ] Foi verificado se já existe uma abstração suficiente?
- [ ] Evita keyword específica demais?
- [ ] Serve para outro domínio sem mudar sua essência?
- [ ] Segurança tem padrão seguro?
- [ ] Há escape hatch documentado para casos avançados?
- [ ] Existem testes normativos?

## Regra suprema

**Germanio não deve ensinar o iniciante a pensar como o computador.
Germanio deve ensinar o computador a executar uma descrição humana, formal e determinística do software.**

**Go constrói mecanismos. Germanio constrói produtos.**
