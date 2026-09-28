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

tenha clientes

cada cliente tem
    nome obrigatório
    email obrigatório e único
    telefone

crie página Clientes
    mostre clientes
    permita
        cadastrar
        editar
        excluir

permita pesquisar clientes
```

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
| 1 — Intenção / declaração | padrão: existência, relações, permissões, estados e páginas | `tenha clientes`, `cliente tem pedidos`, `issue pode fechar` |
| 2 — Configuração | personalizar comportamento sem implementar mecanismos | `login bloqueia após 10 tentativas por 10 minutos` |
| 3 — Lógica | último recurso para regra específica não expressável declarativamente | `quando`, `antes de`, `se`, `para cada` |
| 4 — Primitivas | controle técnico avançado | `git.*`, `cripto.*`, `acesso.*`, `rotas`, `requisicao`, `responder` |

Antes de usar nível 3, verifique se falta uma abstração genérica. Nível 4 não deve ser
necessário para aplicações comuns. A progressão é **intenção → configuração → lógica →
primitivas → interop/adaptador**. Adaptador é uma fronteira arquitetural, não um quinto
nível obrigatório para o iniciante.

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
| `labels por nome` (linha de `issue tem`) | lista escrita e lida pelo nome (`"bug,ux"` → `["bug","ux"]`), procurado entre os itens do mesmo pai (labels do projeto); nome novo cria o item quando a pessoa pode criá-lo ali; `?labels=bug` filtra pelo nome. Itens de outro pai nunca entram |
| `runners executam jobs` | trabalho remoto: cada runner (um `token secreto`, `ativo`, opcionalmente `pode pertencer a projeto`) pega trabalhos pendentes com reserva renovável; reserva vencida devolve o trabalho à fila (3 tentativas, depois falha); cancelamento visível ao executor; token temporário (só do próprio trabalho, lê o repositório só enquanto executa). Vale para qualquer dado: `trabalhadores executam conversoes`. Protocolos externos falam com isso por um adaptador em `integracoes/` usando `trabalho_remoto.*` |
| `projeto executa pipelines a cada envio de código conforme "arquivo.yml"` | o arquivo, no formato nativo (`estagios`, `etapas` com `comandos`, `depois`, `quando: automatico / manual / sempre`, `pode_falhar`, `imagem`), vira uma execução por envio; um adaptador pode ler outro formato com `traduza arquivos de execução com f` e dar variáveis às etapas com `traduza variáveis das etapas com f` (só em `integracoes/`) |
| `quem cria projeto vira owner` | quem cria vira membro com esse papel — exceto quando o dado herda membros de um pai e foi criado dentro dele (os membros já vêm do pai) |
| `todo grupo precisa ter pelo menos um owner` | ninguém remove nem rebaixa o último membro com esse papel (ou superior) |
| `repositório do projeto pode começar com "README.md" contendo "# {nome}"` | ao criar com `iniciar_repositorio` (nome externo pelo vocabulário), o repositório nasce com esse arquivo; `{campo}` vira o valor do registro |
| `projeto tem repositório` | cada registro tem um repositório Git criado e removido com ele e servido em `/<campo único>.git` |

**Tipo pelo nome** (quando não há tipo): `email`→email · `senha`/`password`→senha protegida ·
`telefone`→telefone · `foto`/`avatar`/`imagem`→imagem · `descricao`/`description`→texto longo ·
`visibilidade`/`visibility`→visibilidade · `preco`/`valor`→dinheiro · `quantidade`/`estoque`→inteiro ·
`admin`/`ativo`/`arquivado`/`confidencial`/`pode_*`/`is_*`→booleano · demais→texto.
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

grupo não pode ser mais visível que o grupo pai
quem cria grupo vira owner
somente maintainer pode enviar código para a branch padrão dos projetos
```

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
- Ninguém concede papel acima do próprio.
- Nada que pertence a uma pessoa é criado em nome de outra, exceto por administradores.
- Registros invisíveis respondem "não encontrado", nunca "proibido".

`permita criar projetos` libera para qualquer pessoa conectada (ou qualquer visitante se não
há login). `permita filtrar clientes por cidade` e `permita pesquisar clientes` habilitam filtro e busca.

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

Toda inferência deve ser inspecionável. `ge explain issue` deve mostrar campos declarados,
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

## Verificação: `ge check`

`ge check app.ge` deve detectar, quando estaticamente possível: relações inválidas,
entidades desconhecidas, ações inexistentes, permissões contraditórias, estados ou transições
impossíveis, inferências ambíguas, capabilities incompletas, integrações inválidas e regras
contraditórias. Campos não utilizados devem gerar aviso quando a análise puder demonstrá-lo;
uso por integração ou reflexão não pode ser ignorado. Sem certeza suficiente, não adivinhe.

Erros de validade impedem aceitação; avisos descrevem situações válidas que merecem revisão.
Condições dependentes de dados ou concorrência exigem verificação em runtime e testes.
`check` não prova ausência de todos os erros nem substitui testes de execução.
Hoje o parser/resolver faz validações estruturais; `explicar.Verificar` avisa sobre dados
inacessíveis e ações que herdam permissão de edição. A lista normativa completa acima não
é uma alegação de que todos os diagnósticos já foram implementados.

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
