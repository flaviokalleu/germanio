# Galeria de Exemplos - Germanio

## Comece pela intenção

A referência normativa é [INTENCAO.md](INTENCAO.md). Aplicações novas começam por
existência, relações, permissões e páginas; lógica explícita é último recurso.

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

O exemplo é introdutório; defina login e permissões adequados antes de publicar dados reais.
Para gerar uma estrutura organizada, use `germanio init loja`. Para verificar um arquivo,
use `ge check app.ge`; `ge explain cliente` inspeciona a entidade no contexto do projeto.
Consulte a ajuda da CLI para argumentos e seleção do arquivo.

**As seções seguintes preservam a referência da sintaxe anterior e recursos avançados.**
Elas não substituem a norma nem tornam HTTP, IDs, CRUD manual ou lógica imperativa
obrigatórios. Não misture exemplos de modos diferentes sem verificar parser e execução.


Colecao de exemplos completos e comentados para inspirar e acelerar o desenvolvimento com Germanio.

---

## Indice

1. [Loja Simples](#1-loja-simples)
2. [Restaurante Modular](#2-restaurante-modular)
3. [Restaurante com WhatsApp](#3-restaurante-com-whatsapp)
4. [Loja em Ingles (English Mode)](#4-loja-em-ingles-english-mode)
5. [Pizzaria Bilinguie (Mixed)](#5-pizzaria-bilinguie-mixed)
6. [Loja Completa (Auth + CRUD)](#6-loja-completa-auth--crud)
7. [Template SaaS Multi-Tenant](#7-template-saas-multi-tenant)
8. [Blog / CMS](#8-blog--cms)
9. [Sistema de RH](#9-sistema-de-rh)
10. [Monitoramento de Servicos](#10-monitoramento-de-servicos)

---

## 1. Loja Simples

O exemplo mais basico: produtos, clientes e pedidos com CRUD completo.

**Arquivo:** `exemplos/loja/inicio.ge`

```
sistema loja

dados

  produto
    nome: texto
    preco: numero
    imagem: upload

  cliente
    nome: texto
    telefone: telefone
    email: email

telas

  tela produtos

    titulo "Produtos"

    lista produtos

      mostrar nome
      mostrar preco

    botao azul
      texto "Comprar"

  tela clientes

    titulo "Clientes"

    lista clientes

      mostrar nome
      mostrar email
      mostrar telefone

    botao verde
      texto "Novo Cliente"

eventos

  quando clicar "Comprar"
    criar pedido

  quando clicar "Novo Cliente"
    criar cliente
```

**O que este exemplo demonstra:**
- Estrutura minima de um sistema Germanio
- Tres modelos de dados independentes
- Duas telas com listagens e botoes de acao
- Eventos de clique vinculados a criacao de registros
- Tipo `upload` para imagens de produtos

**Como rodar:**

```bash
./germanio run exemplos/loja/inicio.ge
```

Acesse `http://localhost:8080` para ver o resultado.

---

## 2. Restaurante Modular

Demonstra o sistema de **imports** — dividindo um sistema grande em varios arquivos `.ge` separados para melhor organizacao.

**Estrutura de arquivos:**

```
restaurante-modular/
  inicio.ge    <- ponto de entrada (so imports)
  dados.ge     <- modelos
  telas.ge     <- telas e componentes
  eventos.ge   <- eventos
  regras.ge    <- logica e validacoes
  tema.ge      <- tema visual
```

**inicio.ge**

```
sistema restaurante

importar "tema.ge"
importar "dados.ge"
importar "telas.ge"
importar "eventos.ge"
importar "regras.ge"
```

**tema.ge**

```
tema
  cor primaria "#6366f1"
  cor secundaria "#8b5cf6"
  cor destaque "#f59e0b"
  cor sidebar "#1e1b4b"
```

**dados.ge**

```
dados

  prato
    nome: texto obrigatorio
    descricao: texto
    preco: dinheiro obrigatorio
    categoria: texto
    status: status

  mesa
    numero: numero obrigatorio unico
    capacidade: numero
    status: status

  pedido
    mesa_id: numero pertence_a mesa
    prato: texto obrigatorio
    quantidade: numero obrigatorio
    valor: dinheiro
    observacao: texto
    status: status

  funcionario
    nome: texto obrigatorio
    cargo: texto
    email: email unico
    telefone: telefone
    status: status
```

**telas.ge**

```
telas

  tela cardapio
    titulo "Cardápio"
    lista prato
      mostrar nome
      mostrar preco
      mostrar categoria
      mostrar status
    botao azul
      texto "Novo Prato"

  tela mesas
    titulo "Mesas"
    lista mesa
      mostrar numero
      mostrar capacidade
      mostrar status
    botao verde
      texto "Nova Mesa"

  tela pedidos
    titulo "Pedidos"
    lista pedido
      mostrar prato
      mostrar quantidade
      mostrar valor
      mostrar status
    botao azul
      texto "Novo Pedido"

  tela equipe
    titulo "Equipe"
    lista funcionario
      mostrar nome
      mostrar cargo
      mostrar email
      mostrar status
    botao azul
      texto "Novo Funcionário"
```

**eventos.ge**

```
eventos

  quando clicar "Novo Prato"
    criar prato

  quando clicar "Nova Mesa"
    criar mesa

  quando clicar "Novo Pedido"
    criar pedido

  quando clicar "Novo Funcionário"
    criar funcionario
```

**regras.ge**

```
logica

  validar email obrigatorio unico
  validar preco maior 0

  se status igual "cancelado"
    mudar cor vermelho

  se status igual "ativo"
    mudar cor verde

  se quantidade maior 10
    validar observacao obrigatorio
```

**O que este exemplo demonstra:**
- Sistema de imports (`importar "arquivo.ge"`)
- Separacao de responsabilidades por arquivo
- Relacionamentos (`pertence_a`)
- Modificadores combinados (`obrigatorio unico`)
- Regras condicionais no bloco `logica`
- Validacao condicional (`se quantidade maior 10`)

---

## 3. Restaurante com WhatsApp

Sistema completo de restaurante com notificacoes automaticas via WhatsApp para clientes ao criar e atualizar pedidos.

**Arquivo:** `exemplos/restaurante-whatsapp/inicio.ge`

```
sistema restaurante

tema
  cor primaria "#6366f1"
  cor secundaria "#8b5cf6"
  cor destaque "#f59e0b"

dados

  prato
    nome: texto obrigatorio
    preco: dinheiro
    categoria: texto
    status: status

  cliente
    nome: texto obrigatorio
    telefone: telefone obrigatorio
    email: email

  pedido
    cliente: texto obrigatorio
    telefone: telefone obrigatorio
    prato: texto obrigatorio
    quantidade: numero
    valor: dinheiro
    status: status

telas

  tela cardapio
    titulo "Cardápio"
    lista prato
      mostrar nome
      mostrar preco
      mostrar status
    botao azul
      texto "Novo Prato"

  tela clientes
    titulo "Clientes"
    lista cliente
      mostrar nome
      mostrar telefone
      mostrar email
    botao verde
      texto "Novo Cliente"

  tela pedidos
    titulo "Pedidos"
    lista pedido
      mostrar cliente
      mostrar prato
      mostrar valor
      mostrar status
    botao azul
      texto "Novo Pedido"

eventos

  quando clicar "Novo Prato"
    criar prato

  quando clicar "Novo Cliente"
    criar cliente

  quando clicar "Novo Pedido"
    criar pedido

integracoes

  whatsapp

    quando criar pedido
      enviar mensagem para telefone
        texto "Ola {cliente}! Seu pedido de {prato} foi recebido! Valor: R${valor}"

    quando atualizar pedido
      enviar mensagem para telefone
        texto "Atualizacao do seu pedido: status agora e {status}"
```

**O que este exemplo demonstra:**
- Bloco `integracoes` com `whatsapp`
- Gatilhos `quando criar` e `quando atualizar`
- Templates de mensagem com variaveis `{campo}`
- Destino da mensagem via campo `telefone` do proprio modelo
- Tres modelos relacionados logicamente

**Como testar:**

```bash
./germanio run exemplos/restaurante-whatsapp/inicio.ge
# Na primeira execucao: escanear QR Code no terminal
# Crie um pedido com numero de telefone valido
# A mensagem chega no WhatsApp em segundos
```

---

## 4. Loja em Ingles (English Mode)

Demonstra que o Germanio funciona **100% em ingles**, com todas as palavras-chave traduzidas.

**Arquivo:** `exemplos/english/inicio.ge`

```
system store

theme
  color primary "#3b82f6"
  color secondary "#8b5cf6"
  color accent "#f59e0b"
  dark

models

  product
    name: text required
    price: money required
    category: text
    status: status

  customer
    name: text required
    email: email unique
    phone: phone

  order
    product: text required
    customer: text
    quantity: number required
    total: money
    status: status

screens

  screen products
    title "Products"
    list product
      show name
      show price
      show category
      show status
    button blue
      text "New Product"

  screen customers
    title "Customers"
    list customer
      show name
      show email
      show phone
    button green
      text "New Customer"

  screen orders
    title "Orders"
    list order
      show product
      show customer
      show quantity
      show total
      show status
    button blue
      text "New Order"

events

  when click "New Product"
    create product

  when click "New Customer"
    create customer

  when click "New Order"
    create order

logic

  validate email required unique
  validate price greater 0
```

**O que este exemplo demonstra:**
- Todas as palavras-chave em ingles
- `dark` mode habilitado no tema
- `validate` com `required`, `unique` e `greater`
- Estrutura identica ao exemplo em portugues — so muda o idioma das palavras-chave

**Equivalencias de palavras-chave:**

| Portugues       | Ingles         |
|-----------------|----------------|
| `sistema`       | `system`       |
| `dados`         | `models`       |
| `telas`         | `screens`      |
| `tela`          | `screen`       |
| `titulo`        | `title`        |
| `lista`         | `list`         |
| `mostrar`       | `show`         |
| `botao`         | `button`       |
| `texto`         | `text`         |
| `eventos`       | `events`       |
| `quando`        | `when`         |
| `clicar`        | `click`        |
| `criar`         | `create`       |
| `logica`        | `logic`        |
| `validar`       | `validate`     |
| `obrigatorio`   | `required`     |
| `unico`         | `unique`       |
| `maior`         | `greater`      |
| `menor`         | `less`         |
| `tema`          | `theme`        |
| `cor`           | `color`        |
| `escuro`        | `dark`         |

---

## 5. Pizzaria Bilinguie (Mixed)

Demonstra que e possivel **misturar** palavras-chave em portugues e ingles no mesmo arquivo. O Germanio aceita qualquer combinacao.

**Arquivo:** `exemplos/mixed/inicio.ge`

```
system pizzaria

theme
  cor primaria "#ef4444"
  cor secundaria "#f97316"
  color accent "#fbbf24"
  dark

dados

  pizza
    nome: text required
    preco: money
    tamanho: texto
    status: status

  cliente
    name: texto required
    email: email unique
    phone: telefone

screens

  tela cardapio
    title "Cardápio"
    list pizza
      show nome
      show preco
      show tamanho
      mostrar status
    button vermelho
      texto "Nova Pizza"

  screen clientes
    titulo "Clientes"
    lista cliente
      mostrar name
      show email
      show phone
    botao azul
      text "Novo Cliente"

events

  when click "Nova Pizza"
    criar pizza

  quando clicar "Novo Cliente"
    create cliente
```

**O que este exemplo demonstra:**
- `system` e `dados` no mesmo arquivo — funciona
- `theme` com `cor` em portugues dentro do bloco ingles
- `screens` com `tela` (PT) e `screen` (EN) misturados
- `show` e `mostrar` na mesma lista
- `when click` e `quando clicar` ambos validos
- Bilinguismo real: times internacionais podem contribuir no idioma preferido

---

## 6. Loja Completa (Auth + CRUD)

O exemplo mais completo: autenticacao com roles, multiplos modelos, validacoes e tema customizado.

**Arquivo:** `exemplos/loja-completa/inicio.ge`

```
sistema loja

tema
  cor primaria "#3b82f6"
  cor secundaria "#6366f1"

autenticacao
  modelo: usuario
  campo_login: email
  campo_senha: senha
  roles: admin, vendedor, cliente

dados

  usuario
    nome: texto obrigatorio
    email: email obrigatorio unico
    senha: senha obrigatorio
    telefone: telefone
    role: texto

  produto
    nome: texto obrigatorio
    descricao: texto_longo
    preco: dinheiro obrigatorio
    categoria: texto
    estoque: numero
    status: status

  pedido
    cliente: texto obrigatorio
    produto: texto obrigatorio
    quantidade: numero obrigatorio
    valor: dinheiro
    telefone: telefone
    status: status

telas

  tela produtos
    titulo "Produtos"
    lista produto
      mostrar nome
      mostrar preco
      mostrar categoria
      mostrar estoque
      mostrar status
    botao azul
      texto "Novo Produto"

  tela pedidos
    titulo "Pedidos"
    lista pedido
      mostrar cliente
      mostrar produto
      mostrar quantidade
      mostrar valor
      mostrar status
    botao verde
      texto "Novo Pedido"

  tela usuarios
    titulo "Usuários"
    requer admin
    lista usuario
      mostrar nome
      mostrar email
      mostrar role
    botao azul
      texto "Novo Usuário"

eventos

  quando clicar "Novo Produto"
    criar produto

  quando clicar "Novo Pedido"
    criar pedido

  quando clicar "Novo Usuário"
    criar usuario

logica

  validar email obrigatorio unico
  validar preco maior 0
```

**O que este exemplo demonstra:**
- Bloco `autenticacao` completo com roles
- Tela restrita a role `admin` (`requer admin`)
- Tipo `texto_longo` para descricoes
- Validacoes no bloco `logica`
- Modelo de usuario com `senha` (bcrypt automatico)
- Tres roles definidas: admin, vendedor, cliente

**Fluxo de uso:**

```bash
# 1. Inicie o servidor
./germanio run exemplos/loja-completa/inicio.ge

# 2. Crie um admin
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"nome":"Admin","email":"admin@loja.com","senha":"admin123","role":"admin"}'

# 3. Faca login
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@loja.com","senha":"admin123"}'

# 4. Use o token retornado para acessar rotas protegidas
curl http://localhost:8080/api/usuario \
  -H "Authorization: Bearer <token>"
```

---

## 7. Template SaaS Multi-Tenant

Um ponto de partida para sistemas SaaS com multiplas empresas (tenants) em uma unica instancia.

```
sistema saas-platform

tema
  cor primaria "#6366f1"
  cor secundaria "#8b5cf6"
  escuro

autenticacao
  modelo: usuario
  campo_login: email
  campo_senha: senha
  roles: superadmin, admin, membro

dados

  empresa
    nome: texto obrigatorio
    dominio: texto unico
    plano: texto
    status: status
    criada_em: data

  usuario
    nome: texto obrigatorio
    email: email obrigatorio unico
    senha: senha obrigatorio
    role: texto
    empresa_id: numero pertence_a empresa
    ativo: booleano

  projeto
    nome: texto obrigatorio
    descricao: texto_longo
    empresa_id: numero pertence_a empresa
    responsavel: texto
    prazo: data
    status: status

  tarefa
    titulo: texto obrigatorio
    descricao: texto_longo
    projeto_id: numero pertence_a projeto
    responsavel: texto
    prioridade: texto
    status: status
    prazo: data

telas

  tela dashboard
    titulo "Dashboard"
    lista projeto
      mostrar nome
      mostrar responsavel
      mostrar prazo
      mostrar status
    botao azul
      texto "Novo Projeto"

  tela projetos
    titulo "Projetos"
    lista projeto
      mostrar nome
      mostrar descricao
      mostrar status
    botao azul
      texto "Novo Projeto"

  tela tarefas
    titulo "Tarefas"
    lista tarefa
      mostrar titulo
      mostrar responsavel
      mostrar prioridade
      mostrar prazo
      mostrar status
    botao verde
      texto "Nova Tarefa"

  tela empresas
    titulo "Empresas"
    requer superadmin
    lista empresa
      mostrar nome
      mostrar dominio
      mostrar plano
      mostrar status
    botao azul
      texto "Nova Empresa"

  tela usuarios
    titulo "Usuários"
    requer admin
    lista usuario
      mostrar nome
      mostrar email
      mostrar role
      mostrar ativo
    botao verde
      texto "Novo Usuário"

eventos

  quando clicar "Novo Projeto"
    criar projeto

  quando clicar "Nova Tarefa"
    criar tarefa

  quando clicar "Nova Empresa"
    criar empresa

  quando clicar "Novo Usuário"
    criar usuario

logica

  validar email obrigatorio unico
  validar dominio unico

  se status igual "inativo"
    mudar cor vermelho

  se prioridade igual "alta"
    mudar cor vermelho

banco
  driver: "postgres"
  host: "localhost"
  porta: "5432"
  nome: "saas_db"
  usuario: "saas_user"
  senha: "senha-producao"
```

**O que este exemplo demonstra:**
- SaaS com hierarquia de roles: superadmin > admin > membro
- Relacionamentos com `pertence_a` em multiplos niveis
- PostgreSQL para producao
- Telas restritas por role diferente (`superadmin`, `admin`)
- Modelos que representam a estrutura multi-tenant
- Status visuais com regras de cor

---

## 8. Blog / CMS

Sistema de gerenciamento de conteudo com categorias, posts e autores.

```
sistema blog

tema
  cor primaria "#1e40af"
  cor secundaria "#3730a3"
  cor destaque "#d97706"

autenticacao
  modelo: autor
  campo_login: email
  campo_senha: senha
  roles: admin, editor, colaborador

dados

  autor
    nome: texto obrigatorio
    email: email obrigatorio unico
    senha: senha obrigatorio
    bio: texto_longo
    avatar: upload
    role: texto

  categoria
    nome: texto obrigatorio unico
    slug: texto unico
    descricao: texto
    cor: texto

  post
    titulo: texto obrigatorio
    slug: texto unico
    conteudo: texto_longo
    resumo: texto
    capa: upload
    autor_id: numero pertence_a autor
    categoria_id: numero pertence_a categoria
    publicado: booleano
    publicado_em: data
    status: status

  comentario
    post_id: numero pertence_a post
    autor_nome: texto obrigatorio
    autor_email: email obrigatorio
    conteudo: texto_longo obrigatorio
    aprovado: booleano
    criado_em: data

telas

  tela posts
    titulo "Posts"
    lista post
      mostrar titulo
      mostrar autor_id
      mostrar categoria_id
      mostrar publicado
      mostrar status
    botao azul
      texto "Novo Post"

  tela categorias
    titulo "Categorias"
    lista categoria
      mostrar nome
      mostrar slug
    botao verde
      texto "Nova Categoria"

  tela comentarios
    titulo "Comentários"
    requer admin
    lista comentario
      mostrar autor_nome
      mostrar post_id
      mostrar aprovado
      mostrar criado_em
    botao azul
      texto "Aprovar"

  tela autores
    titulo "Autores"
    requer admin
    lista autor
      mostrar nome
      mostrar email
      mostrar role
    botao verde
      texto "Novo Autor"

eventos

  quando clicar "Novo Post"
    criar post

  quando clicar "Nova Categoria"
    criar categoria

  quando clicar "Novo Autor"
    criar autor

logica

  validar email obrigatorio unico
  validar slug unico
  validar titulo obrigatorio

  se publicado igual "true"
    mudar cor verde

  se status igual "rascunho"
    mudar cor cinza

integracoes

  email
    servidor: "smtp.gmail.com"
    porta: "587"
    usuario: "blog@meusite.com"
    senha: "app-password"

    quando criar comentario
      enviar email para autor_email
        assunto "Novo comentario no seu post"
        texto "Ola! {autor_nome} comentou: {conteudo}"
```

**O que este exemplo demonstra:**
- CMS completo com autenticacao por roles
- Relacionamentos de dois niveis (comentario -> post -> autor)
- Campos `booleano` para flags (publicado, aprovado)
- Tipo `slug` como `texto unico`
- Notificacao por email ao criar comentarios
- Tipo `texto_longo` para conteudo rico

---

## 9. Sistema de RH

Gerenciamento de funcionarios, departamentos e folha de ponto.

```
sistema rh

tema
  cor primaria "#059669"
  cor secundaria "#0d9488"
  cor destaque "#d97706"

autenticacao
  modelo: usuario
  campo_login: email
  campo_senha: senha
  roles: admin, rh, funcionario

dados

  usuario
    nome: texto obrigatorio
    email: email obrigatorio unico
    senha: senha obrigatorio
    role: texto

  departamento
    nome: texto obrigatorio unico
    descricao: texto
    gerente: texto
    orcamento: dinheiro
    status: status

  funcionario
    nome: texto obrigatorio
    email: email obrigatorio unico
    telefone: telefone
    cpf: texto unico
    cargo: texto obrigatorio
    salario: dinheiro
    data_admissao: data
    departamento_id: numero pertence_a departamento
    status: status

  ponto
    funcionario_id: numero pertence_a funcionario
    data: data obrigatorio
    entrada: texto
    saida: texto
    horas_trabalhadas: numero
    justificativa: texto
    status: status

  ferias
    funcionario_id: numero pertence_a funcionario
    data_inicio: data obrigatorio
    data_fim: data obrigatorio
    dias: numero
    aprovado: booleano
    aprovado_por: texto
    status: status

telas

  tela funcionarios
    titulo "Funcionários"
    lista funcionario
      mostrar nome
      mostrar cargo
      mostrar departamento_id
      mostrar data_admissao
      mostrar status
    botao azul
      texto "Novo Funcionário"

  tela departamentos
    titulo "Departamentos"
    lista departamento
      mostrar nome
      mostrar gerente
      mostrar status
    botao verde
      texto "Novo Departamento"

  tela ponto
    titulo "Folha de Ponto"
    lista ponto
      mostrar funcionario_id
      mostrar data
      mostrar entrada
      mostrar saida
      mostrar horas_trabalhadas
      mostrar status
    botao azul
      texto "Registrar Ponto"

  tela ferias
    titulo "Férias"
    lista ferias
      mostrar funcionario_id
      mostrar data_inicio
      mostrar data_fim
      mostrar dias
      mostrar aprovado
      mostrar status
    botao verde
      texto "Solicitar Férias"

eventos

  quando clicar "Novo Funcionário"
    criar funcionario

  quando clicar "Novo Departamento"
    criar departamento

  quando clicar "Registrar Ponto"
    criar ponto

  quando clicar "Solicitar Férias"
    criar ferias

logica

  validar email obrigatorio unico
  validar cpf unico
  validar salario maior 0

integracoes

  email
    servidor: "smtp.empresa.com"
    porta: "587"
    usuario: "rh@empresa.com"
    senha: "senha-rh"

    quando criar ferias
      enviar email para funcionario_id
        assunto "Solicitação de Férias Recebida"
        texto "Sua solicitacao de ferias de {dias} dias foi registrada. Aguarde aprovacao."

  cron

    cada 1 dia
      chamar api "http://localhost:8080/api/ponto?status=aberto"
```

**O que este exemplo demonstra:**
- Sistema de RH completo com 5 modelos
- Multiplos relacionamentos `pertence_a`
- Campos especializados: `cpf`, `data_admissao`, `horas_trabalhadas`
- Email de notificacao para solicitacoes de ferias
- Cron diario para verificar pontos em aberto
- Hierarquia clara: admin > rh > funcionario

---

## 10. Monitoramento de Servicos

Sistema de monitoramento de saude de APIs e servicos externos com alertas automaticos.

```
sistema monitoramento

tema
  cor primaria "#7c3aed"
  cor secundaria "#6d28d9"
  escuro

dados

  servico
    nome: texto obrigatorio
    url: link obrigatorio
    tipo: texto
    intervalo_minutos: numero
    timeout_segundos: numero
    status: status
    ultimo_check: data
    tempo_resposta: numero

  incidente
    servico_id: numero pertence_a servico
    tipo: texto
    descricao: texto_longo
    iniciado_em: data
    resolvido_em: data
    status: status

  contato
    nome: texto obrigatorio
    email: email obrigatorio
    telefone: telefone
    notificar_whatsapp: booleano
    notificar_email: booleano
    ativo: booleano

telas

  tela servicos
    titulo "Serviços Monitorados"
    lista servico
      mostrar nome
      mostrar url
      mostrar status
      mostrar ultimo_check
      mostrar tempo_resposta
    botao azul
      texto "Novo Serviço"

  tela incidentes
    titulo "Incidentes"
    lista incidente
      mostrar servico_id
      mostrar tipo
      mostrar iniciado_em
      mostrar status
    botao vermelho
      texto "Registrar Incidente"

  tela contatos
    titulo "Contatos de Alerta"
    lista contato
      mostrar nome
      mostrar email
      mostrar telefone
      mostrar ativo
    botao verde
      texto "Novo Contato"

eventos

  quando clicar "Novo Serviço"
    criar servico

  quando clicar "Registrar Incidente"
    criar incidente

  quando clicar "Novo Contato"
    criar contato

logica

  se status igual "down"
    mudar cor vermelho

  se status igual "degraded"
    mudar cor amarelo

  se status igual "up"
    mudar cor verde

  validar url obrigatorio

integracoes

  whatsapp

    quando criar incidente
      enviar mensagem para telefone
        texto "ALERTA: {servico_id} esta {tipo}! Iniciado em {iniciado_em}. Verifique imediatamente."

  email

    servidor: "smtp.gmail.com"
    porta: "587"
    usuario: "alertas@empresa.com"
    senha: "app-password"

    quando criar incidente
      enviar email para email
        assunto "INCIDENTE: {servico_id} - {tipo}"
        texto "Incidente detectado em {servico_id}. Tipo: {tipo}. Descricao: {descricao}. Iniciado: {iniciado_em}"

  cron

    cada 5 minutos
      chamar api "http://localhost:8080/api/servico"

    cada 1 hora
      chamar api "https://meuapp.com/relatorio-saude"
```

**O que este exemplo demonstra:**
- Sistema de monitoramento com alertas duplos (WhatsApp + Email)
- Cores condicionais por status (`down`/`degraded`/`up`)
- Cron a cada 5 minutos para verificacao continua
- Modelo de `contato` para gerenciar destinatarios dos alertas
- Relacionamento `incidente` -> `servico`
- Modo escuro habilitado (ideal para monitoramento)

---

## Padroes Comuns

### Adicionar Busca a uma Tela

```
telas

  tela produtos
    titulo "Produtos"
    busca produto        // adiciona campo de busca
    lista produto
      mostrar nome
      mostrar preco
```

### Tela com Dashboard e Grafico

```
telas

  tela analytics
    titulo "Analytics"
    dashboard vendas
    grafico vendas_mensais
    tabela top_produtos
```

### Modelo com Soft Delete

```
dados

  pedido soft_delete     // registros deletados ficam no banco
    cliente: texto
    valor: dinheiro
    status: status
```

### Importar Apenas Dados de um Arquivo

```
// importar dados de arquivo especifico
importar dados de "modelos-compartilhados.ge"

// importar tudo
importar "config-base.ge"
```

### Validacao com Logica Condicional

```
logica

  validar email obrigatorio unico
  validar preco maior 0

  se status igual "pago"
    validar data_pagamento obrigatorio

  se quantidade maior 100
    validar aprovacao obrigatorio
```

---

## Rodando os Exemplos

```bash
# Loja simples
./germanio run exemplos/loja/inicio.ge

# Restaurante completo
./germanio run exemplos/restaurante/inicio.ge

# Restaurante modular (importa multiplos arquivos)
./germanio run exemplos/restaurante-modular/inicio.ge

# Restaurante com WhatsApp
./germanio run exemplos/restaurante-whatsapp/inicio.ge

# Loja em ingles
./germanio run exemplos/english/inicio.ge

# Pizzaria bilinguie
./germanio run exemplos/mixed/inicio.ge

# Loja com autenticacao
./germanio run exemplos/loja-completa/inicio.ge
```

Todos os exemplos iniciam o servidor na porta `8080` por padrao. Acesse `http://localhost:8080` no navegador.
