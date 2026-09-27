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
excluir remove o que pertence ao registro; nada fica pela metade se `antes de`/`quando`
falhar em uma criação.

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
- Dentro de algo que tem membros, só papéis decidem (ser conectado não basta para criar projeto em grupo).
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

## Integração

`disponibilize projetos para integração como "projects"` + `integração em "/api/v4"`
publica listar (busca, filtros, paginação com `X-Total`), ver (por id ou campo único),
criar, editar, excluir, ações próprias, filhos (`/projects/:id/members`) e, para dados com
repositório, `repository/{branches,commits,tree,files,compare}`. A pessoa conectada
aparece em `/<prefixo>/<singular>` (`/api/v4/user`). As páginas usam as mesmas operações em `/_ge/api`.
