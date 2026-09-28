# Germanio — camada de intenção

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
- A página diz *o que aparece*; o backend decide *quem pode*. `permita criar` numa página
  mostra o formulário apenas a quem o backend permite.

## Níveis

| Nível | Para quê | Exemplos |
|-------|----------|----------|
| 1 — Declarativo | o que existe | `tenha`, `cada X tem`, `X tem Y`, `X pertence a Y` |
| 2 — Configuração | comportamento | `login ...`, `tenha papeis`, `pode`, `permita`, `disponibilize`, `mensagens em inglês` |
| 3 — Lógica | regras específicas do produto | `antes de`, `quando`, `se`, `para cada`, `recuse`, funções em `logica` |
| 4 — Primitivas | controle técnico | `git.*`, `cripto.*`, `acesso.*`, `rotas`, `requisicao`, `responder` |

O nível 4 nunca é necessário para CRUD, login, permissões ou relações.

## O que existe

| Frase | Significado |
|-------|-------------|
| `tenha clientes, pedidos` | dados persistentes; o singular vem da regra fixa do plural (`clientes→cliente`, `papeis→papel`, `itens→item`). Se houver outra forma possível (`tokens→token`), a forma usada em `cada … tem` decide. |
| `cada cliente tem` + linhas | campos: `nome [tipo] [modificadores] [começa com valor]` |
| `cliente tem pedidos` | relação 1:N; cada pedido recebe `cliente_id` obrigatório |
| `pedido pertence a cliente [opcional] [como dono]` | referência (`cliente_id`/`dono_id`) validada ("não existe") |
| `grupo tem subgrupos` | hierarquia (`pai_id`) |
| `endereço dentro do grupo pai ou do criador` (linha de `X tem`) | `endereco` = endereço do primeiro contêiner definido + `/` + `caminho` (`empresa/web/app`); calculado, nunca aceito da entrada; único entre todos os endereçados e os nomes das pessoas; renomear um contêiner (ou a pessoa) atualiza o que está dentro; repositórios passam a ser servidos pelo endereço |
| `labels por nome` (linha de `issue tem`) | lista escrita e lida pelo nome (`"bug,ux"` → `["bug","ux"]`), procurado entre os itens do mesmo pai (labels do projeto); nome novo cria o item quando a pessoa pode criá-lo ali; `?labels=bug` filtra pelo nome. Itens de outro pai nunca entram |
| `projeto tem repositório` | cada registro tem um repositório Git criado e removido com ele e servido em `/<campo único>.git` |

**Tipo pelo nome** (quando não há tipo): `email`→email · `senha`/`password`→senha protegida ·
`telefone`→telefone · `foto`/`avatar`/`imagem`→imagem · `descricao`/`description`→texto longo ·
`visibilidade`/`visibility`→visibilidade · `preco`/`valor`→dinheiro · `quantidade`/`estoque`→inteiro ·
`admin`/`ativo`/`arquivado`/`confidencial`/`pode_*`/`is_*`→booleano · demais→texto.
`começa com 10` → inteiro; `começa com verdadeiro` → booleano.

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
- `administrador` = pessoa com `admin` verdadeiro (ou `papel` "administrador"); pode tudo.
- `todos` = qualquer visitante; `usuario` (o dado do login) = qualquer pessoa conectada;
  `membro` = qualquer papel no registro.
- `seu/sua/seus/suas` = registros que pertencem à pessoa (ou a própria pessoa: `perfil`).
- `administrar X` = ver, editar e excluir X, e criar/editar/excluir o que pertence a X e seus membros.
- Ações sinônimas: ver/listar/mostrar · criar/cadastrar/adicionar · editar/alterar ·
  excluir/remover · pesquisar/buscar. Outras ações (`bloquear`) exigem `quando bloquear usuario`.
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

```ge
antes de criar projeto          # pode ajustar `dados` ou recusar
    dados.full_path = atual.username + "/" + dados.path

quando criar projeto            # depois de salvar
    ...

antes de excluir membro
    se membro.papel == "owner" e membro.contar({recurso: membro.recurso, recurso_id: membro.recurso_id, papel: "owner"}) == 1
        recuse "O grupo precisa ter pelo menos um owner"

quando bloquear usuario         # ação própria
    usuario.atualizar(usuario.id, {state: "blocked"})

antes de enviar código para projeto
    para cada mudanca em atualizacoes
        ...

ao iniciar
    ...
```

Variáveis disponíveis: `atual` (quem age), o registro pelo nome do dado e como `registro`,
`entrada` (o que a pessoa enviou), `dados` (em `antes de`: o que será salvo),
`atualizacoes` (em código enviado).

## Login

```ge
tenha login
tenha cadastro
login usa username e email
login exige state "active"
login bloqueia após 10 tentativas por 10 minutos
login aceita tokens de acesso no cabeçalho "PRIVATE-TOKEN"
login aceita oauth por 2 horas
```

O runtime oferece `entrar`, `sair`, `cadastro`, sessão com CSRF, tokens (cabeçalho, `Bearer`,
HTTP Basic para git) e `oauth/token`. O cadastro nunca aceita campos como `admin`.

## Integração e compatibilidade

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

## Inspeção

`ge explain issue` mostra campos (declarados e internos), relações, estados, visibilidade,
quem pode cada ação, regras explícitas e endereços. `ge check app.ge` valida e avisa sobre dados
que ninguém pode usar e ações que seguem o padrão de edição.

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
