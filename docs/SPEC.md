# Germanio — Especificação da Linguagem (Language Specification)

> Versao 0.5.0 | Ultima atualizacao: 2026-04-10

---

## Sumario

1. [Visao Geral](#1-visão-geral)
2. [Formato de Arquivo](#2-formato-de-arquivo)
3. [Sintaxe Fundamental](#3-sintaxe-fundamental)
4. [Palavras-Chave Completas](#4-palavras-chave-completas)
5. [Blocos de Nivel Superior](#5-blocos-de-nível-superior)
6. [Tipos de Dados](#6-tipos-de-dados)
7. [Modificadores de Campo](#7-modificadores-de-campo)
8. [Relacionamentos](#8-relacionamentos)
9. [Componentes de Tela](#9-componentes-de-tela)
10. [Eventos](#10-eventos)
11. [Logica e Regras](#11-lógica-e-regras)
12. [Autenticacao](#12-autenticação)
13. [Integracoes](#13-integrações)
14. [Tema](#14-tema)
15. [Sistema de Importacao](#15-sistema-de-importação)
16. [Banco de Dados](#16-banco-de-dados)
17. [Operadores](#17-operadores)
18. [Palavras Reservadas](#18-palavras-reservadas)
19. [Gramatica Formal (EBNF)](#19-gramática-formal-ebnf)
20. [Idiomas Suportados](#20-idiomas-suportados)
21. [Tema Presets e Cores por Nome](#21-tema-presets-e-cores-por-nome)
22. [Enum com Valores](#22-enum-com-valores)
23. [Rotas Customizadas](#23-rotas-customizadas)
24. [Paginas Customizadas](#24-paginas-customizadas)
25. [Sidebar Customizada](#25-sidebar-customizada)
26. [Funcoes Async](#26-funcoes-async)
27. [Array Indexing](#27-array-indexing)

---

## 1. Visão Geral

Germanio e uma linguagem de programacao declarativa multilingual (20 idiomas) que gera aplicacoes full-stack completas a partir de arquivos `.ge`. Um unico arquivo `.ge` descreve modelos de dados, telas, eventos, autenticacao, integracoes e configuracoes de banco de dados — e o motor Germanio (escrito em Go) gera e executa tudo automaticamente.

**Caracteristicas principais:**
- Sintaxe declarativa baseada em indentacao (sem chaves, sem ponto-e-virgula)
- 20 idiomas suportados (Portugues, Ingles, Espanhol, Frances, Alemao, etc.)
- API REST gerada automaticamente para cada modelo
- Banco de dados integrado (SQLite padrao, PostgreSQL e MySQL suportados)
- Servidor web embutido com WebSocket
- Autenticacao JWT com bcrypt integrada
- Suporte a Soft Delete, paginacao, filtros, busca e exportacao
- Tema presets, cores por nome, 4 estilos visuais
- Rotas e paginas customizadas
- Funcoes async e paralelismo
- Compilacao para executavel standalone (`germanio build`)

---

## 2. Formato de Arquivo

| Atributo | Valor |
|---|---|
| Extensão | `.ge` |
| Codificação | UTF-8 |
| Terminador de linha | `\n` (LF) ou `\r\n` (CRLF, ignorado) |
| Indentação | Espaços ou tabs (2 espaços por nível recomendado; tab = 2 espaços) |
| Sensibilidade a maiúsculas | Insensível (case-insensitive) para palavras-chave |

**Arquivo mínimo válido:**

```germanio
sistema meuapp
```

**Arquivo de exemplo completo:**

```germanio
sistema loja

dados

  produto
    nome: texto obrigatorio
    preco: dinheiro obrigatorio
    descricao: texto_longo
    ativo: booleano padrao "true"

telas

  tela produtos
    titulo "Produtos"
    lista produto
      mostrar nome
      mostrar preco
    botao verde
      texto "Novo Produto"

eventos

  quando clicar "Novo Produto"
    criar produto
```

---

## 3. Sintaxe Fundamental

### 3.1 Indentação

Germanio usa indentação para delimitar blocos — **não** usa chaves `{}` nem palavras `begin/end`.

- O nível de indentação é detectado automaticamente por contagem de espaços
- Tabs são convertidos internamente: 1 tab = 2 espaços
- Inconsistências de indentação podem causar erros de parsing
- O parser usa tokens `INDENT` para rastrear a hierarquia

```germanio
dados          # bloco de nível superior (indentação 0)

  produto      # modelo (indentação 2)
    nome: texto  # campo (indentação 4)
```

### 3.2 Comentários

Germanio suporta dois estilos de comentário — ambos comentam até o fim da linha:

```germanio
// Comentário estilo C/Go
# Comentário estilo Python/Shell

sistema minhaApp  // comentário inline
```

### 3.3 Strings

- Delimitadas por aspas duplas `"..."`
- Strings não podem conter quebras de linha (são de linha única)
- Sequências de escape suportadas:

| Escape | Significado |
|---|---|
| `\"` | Aspas duplas |
| `\\` | Barra invertida |
| `\n` | Nova linha |
| `\t` | Tab |

```germanio
titulo "Minha Aplicação"
mensagem "Olá, {nome}!\nBem-vindo ao sistema."
```

### 3.4 Números

- Literais numéricos: inteiros e decimais com ponto
- Exemplos: `42`, `3.14`, `1000`, `0.5`

### 3.5 Identificadores

- Podem conter letras (Unicode), dígitos e underscores `_`
- Não podem começar com dígito
- Case-insensitive para palavras-chave; case-sensitive para nomes de modelos/campos
- Exemplos válidos: `produto`, `nomeCliente`, `data_nascimento`, `Produto2`

### 3.6 Blocos

Blocos são seções de nível superior identificadas por palavras-chave reservadas. Cada bloco começa na coluna 0 e seu conteúdo é indentado:

```germanio
dados          # início do bloco 'dados'

  modelo       # conteúdo do bloco
    campo: tipo

telas          # próximo bloco de nível superior
```

---

## 4. Palavras-Chave Completas

### 4.1 Blocos de Nível Superior

| Português | Inglês | Token | Descrição |
|---|---|---|---|
| `sistema` | `system` | `TokenSistema` | Nome da aplicação |
| `dados` | `models` | `TokenDados` | Define os modelos de dados |
| `telas` | `screens` | `TokenTelas` | Define as telas da UI |
| `acoes` | `actions` | `TokenAcoes` | Define ações nomeadas |
| `eventos` | `events` | `TokenEventos` | Define handlers de eventos |
| `integracoes` | `integrations` | `TokenIntegracoes` | Configurações de integrações |
| `tema` | `theme` | `TokenTema` | Tema visual da aplicação |
| `logica` | `logic` | `TokenLogica` | Regras de negócio |
| `banco` | `database` / `db` | `TokenBanco` | Configuração do banco de dados |
| `autenticacao` | `auth` / `authentication` | `TokenAutenticacao` | Configuração de autenticação |
| `config` | `config` | `TokenConfig` | Configurações gerais |

### 4.2 Importação

| Português | Inglês | Token | Descrição |
|---|---|---|---|
| `importar` | `import` | `TokenImportar` | Importa arquivo externo |
| `de` | `from` | `TokenDe` | Especifica origem do import |

### 4.3 Palavras-Chave de Tela

| Português | Inglês | Token | Descrição |
|---|---|---|---|
| `tela` | `screen` | `TokenTela` | Define uma tela |
| `titulo` | `title` | `TokenTitulo` | Título da tela |
| `lista` | `list` | `TokenLista` | Componente de listagem |
| `mostrar` | `show` | `TokenMostrar` | Exibe um campo |
| `botao` | `button` | `TokenBotao` | Componente de botão |
| `formulario` | `form` | `TokenFormulario` | Componente de formulário |
| `entrada` | `input` | `TokenEntrada` | Campo de entrada |
| `busca` | `search` | `TokenBusca` | Campo de busca |
| `dashboard` | `dashboard` | `TokenDashboard` | Painel de controle |
| `grafico` | `chart` | `TokenGrafico` | Componente de gráfico |
| `tabela` | `table` | `TokenTabela` | Componente de tabela |
| `campo` | `field` | `TokenCampo` | Referência a campo |
| `selecionar` | `select` | `TokenSelecionar` | Dropdown/select |
| `area_texto` | `textarea` | `TokenAreaTexto` | Área de texto longo |

### 4.4 Tipos de Dados

| Português | Inglês | Token | Descrição |
|---|---|---|---|
| `texto` | `text` | `TokenTexto` | Texto curto |
| `numero` | `number` | `TokenNumero` | Número (inteiro ou decimal) |
| `data` | `date` | `TokenData` | Data e hora |
| `booleano` | `boolean` | `TokenBooleano` | Verdadeiro/Falso |
| `email` | `email` | `TokenEmail` | Endereço de e-mail |
| `telefone` | `phone` | `TokenTelefone` | Número de telefone |
| `imagem` | `image` | `TokenImagem` | Upload de imagem |
| `arquivo` | `file` | `TokenArquivo` | Upload de arquivo genérico |
| `upload` | `upload` | `TokenUpload` | Upload genérico |
| `link` | `link` | `TokenLink` | URL/hyperlink |
| `status` | `status` | `TokenStatus` | Campo de status/estado |
| `dinheiro` | `money` / `currency` | `TokenDinheiro` | Valor monetário |
| `senha` | `password` | `TokenSenha` | Senha (armazenada com hash) |
| `texto_longo` | `long_text` | `TokenTextoLongo` | Texto longo (textarea) |
| `enum` | `enum` | `TokenEnum` | Enumeração de valores |

### 4.5 Eventos

| Português | Inglês | Token | Descrição |
|---|---|---|---|
| `quando` | `when` | `TokenQuando` | Define trigger de evento |
| `clicar` | `click` | `TokenClicar` | Evento de clique |
| `criar` | `create` | `TokenCriar` | Evento de criação de registro |
| `atualizar` | `update` | `TokenAtualizar` | Evento de atualização |
| `deletar` | `delete` | `TokenDeletar` | Evento de exclusão |
| `enviar` | `send` | `TokenEnviar` | Evento de envio (formulário) |

### 4.6 Lógica

| Português | Inglês | Token | Descrição |
|---|---|---|---|
| `se` | `if` | `TokenSe` | Condicional |
| `senao` | `else` | `TokenSenao` | Alternativa condicional |
| `igual` | `equals` / `equal` | `TokenIgual` | Operador de igualdade |
| `maior` | `greater` | `TokenMaior` | Operador maior que |
| `menor` | `less` | `TokenMenor` | Operador menor que |
| `e` | `and` | `TokenE` | Operador lógico E |
| `ou` | `or` | `TokenOu` | Operador lógico OU |
| `entao` | `then` | `TokenEntao` | Consequência de condição |
| `validar` | `validate` | `TokenValidar` | Regra de validação |
| `calcular` | `compute` | `TokenCalcular` | Cálculo derivado |
| `definir` | `set` | `TokenDefinir` | Atribuição de valor |
| `retornar` | `return` | `TokenRetornar` | Retorno de valor |
| `mudar` | `change` | `TokenMudar` | Mudança de estado |
| `para` | `to` | `TokenPara` | Destino de atribuição |
| `para_cada` | `for_each` | `TokenParaCada` | Iteração |
| `funcao` | `function` | `TokenFuncao` | Definição de função |
| `tentar` | `try` | `TokenTentar` | Tratamento de erro |
| `erro` | `error` | `TokenErro` | Bloco de erro |

### 4.7 Autenticação

| Português | Inglês | Token | Descrição |
|---|---|---|---|
| `login` | `login` | `TokenLogin` | Configuração de login |
| `registro` | `register` | `TokenRegistro` | Configuração de registro |
| `usuario` | `user` | `TokenUsuario` | Modelo de usuário |
| `permissao` | `permission` | `TokenPermissao` | Definição de permissão |
| `requer` | `requires` | `TokenRequer` | Restrição de acesso |
| `admin` | `admin` | `TokenAdmin` | Papel de administrador |
| `publico` | `public` | `TokenPublico` | Acesso público (sem auth) |

### 4.8 Integrações

| Português | Inglês | Token | Descrição |
|---|---|---|---|
| `whatsapp` | `whatsapp` | `TokenWhatsapp` | Integração WhatsApp |
| `mensagem` | `message` | `TokenMensagem` | Conteúdo de mensagem |
| `notificar` | `notify` | `TokenNotificar` | Dispara notificação |
| `cron` | `cron` | `TokenCron` | Job agendado |
| `cada` | `every` | `TokenCada` | Intervalo de repetição |
| `hora` | `hour` | `TokenHora` | Unidade de tempo: hora |
| `minuto` | `minute` | `TokenMinuto` | Unidade de tempo: minuto |
| `chamar` | `call` | `TokenChamar` | Chama URL externa |
| `api` | `api` | `TokenApi` | Referência a API |
| `webhook` | `webhook` | `TokenWebhook` | Webhook HTTP |
| `pagamento` | `payment` | `TokenPagamento` | Integração de pagamento |

### 4.9 Relacionamentos

| Português | Inglês | Token | Descrição |
|---|---|---|---|
| `pertence_a` | `belongs_to` | `TokenPertenceA` | Chave estrangeira (N→1) |
| `tem_muitos` | `has_many` | `TokenTemMuitos` | Relacionamento 1→N |
| `muitos_para_muitos` | `many_to_many` | `TokenMuitosParaMuitos` | Relacionamento N→N |

### 4.10 Tema

| Português | Inglês | Token | Descrição |
|---|---|---|---|
| `cor` | `color` | `TokenCor` | Define cor |
| `icone` | `icon` | `TokenIcone` | Define ícone |
| `escuro` | `dark` | `TokenEscuro` | Modo escuro |

### 4.11 Modificadores

| Português | Inglês | Token | Descrição |
|---|---|---|---|
| `obrigatorio` | `required` | `TokenObrigatorio` | Campo obrigatório (NOT NULL) |
| `unico` | `unique` | `TokenUnico` | Valor único (UNIQUE) |
| `padrao` | `default` | `TokenPadrao` | Valor padrão |
| `indice` | `index` | `TokenIndice` | Cria índice no banco |
| `soft_delete` | *(apenas PT)* | `TokenSoftDelete` | Exclusão suave (marca deletado_em) |

---

## 5. Blocos de Nível Superior

### 5.1 `sistema` / `system`

Declara o nome da aplicação. Deve ser a primeira declaração do arquivo.

**Gramática:**
```
sistema_block = "sistema" IDENTIFIER
```

**Exemplo:**
```germanio
sistema minhaLoja
```
```germanio
system "My Store"
```

### 5.2 `dados` / `models`

Declara os modelos de dados. Cada modelo gera automaticamente:
- Uma tabela no banco de dados
- Endpoints REST completos (`GET`, `POST`, `PUT`, `DELETE`)
- Colunas automáticas: `id` (PK auto-increment), `criado_em`, `atualizado_em`

**Gramática:**
```
dados_block   = "dados" NEWLINE model+
model         = IDENTIFIER modifier* NEWLINE field*
field         = IDENTIFIER ":" type modifier* NEWLINE
modifier      = "obrigatorio" | "unico" | "indice" | "padrao" STRING | relationship
relationship  = "pertence_a" IDENTIFIER
```

**Exemplo:**
```germanio
dados

  cliente
    nome: texto obrigatorio
    email: email unico obrigatorio
    telefone: telefone
    ativo: booleano padrao "true"

  pedido soft_delete
    numero: texto unico
    valor: dinheiro obrigatorio
    status: status padrao "pendente"
    cliente_id: numero pertence_a cliente
```

**Colunas automáticas geradas:**

| Coluna | Tipo SQL | Descrição |
|---|---|---|
| `id` | `INTEGER PRIMARY KEY AUTOINCREMENT` | Identificador único |
| `criado_em` | `DATETIME DEFAULT CURRENT_TIMESTAMP` | Data de criação |
| `atualizado_em` | `DATETIME DEFAULT CURRENT_TIMESTAMP` | Data de atualização |
| `deletado_em` | `DATETIME DEFAULT NULL` | Somente se `soft_delete` ativo |

### 5.3 `telas` / `screens`

Declara as telas da interface do usuário. Cada tela é uma rota no frontend gerado.

**Gramática:**
```
telas_block = "telas" NEWLINE screen+
screen      = "tela" IDENTIFIER NEWLINE screen_item*
screen_item = titulo | lista | botao | formulario | mostrar | busca
```

**Exemplo:**
```germanio
telas

  tela produtos
    titulo "Catálogo de Produtos"
    lista produto
      mostrar nome
      mostrar preco
    botao verde
      texto "Adicionar"
```

### 5.4 `eventos` / `events`

Associa ações a gatilhos de interface.

**Gramática:**
```
eventos_block = "eventos" NEWLINE event+
event         = "quando" trigger [STRING] NEWLINE action_ref
trigger       = "clicar" | "enviar" | "criar" | "atualizar" | "deletar"
action_ref    = IDENTIFIER (IDENTIFIER)*
```

**Exemplo:**
```germanio
eventos

  quando clicar "Novo Produto"
    criar produto

  quando enviar formulario
    salvar produto
```

### 5.5 `logica` / `logic`

Define regras de negócio e validações condicionais.

**Gramática:**
```
logica_block  = "logica" NEWLINE rule*
rule          = "se" IDENTIFIER operator value NEWLINE action IDENTIFIER
operator      = "igual" | "maior" | "menor"
action        = "mudar" | "validar" | "calcular" | "definir" | "retornar"
```

**Exemplo:**
```germanio
logica

  se status igual "aprovado"
    mudar cor verde

  se valor maior 1000
    notificar gerente

  validar email obrigatorio
```

### 5.6 `autenticacao` / `auth`

Configura o sistema de autenticação JWT.

**Gramática:**
```
auth_block   = "autenticacao" NEWLINE auth_item*
auth_item    = "usuario" IDENTIFIER
             | "login" IDENTIFIER
             | "permissao" IDENTIFIER+
             | "publico"
```

**Exemplo:**
```germanio
autenticacao

  usuario cliente
  login email
  permissao admin gerente vendedor
```

### 5.7 `banco` / `database` / `db`

Configura a conexão com banco de dados.

**Gramática:**
```
banco_block = ("banco" | "database" | "db") NEWLINE banco_item*
banco_item  = IDENTIFIER ":" (STRING | IDENTIFIER)
```

**Exemplo — SQLite (padrão):**
```germanio
banco
  driver: sqlite
  nome: minha_loja.db
```

**Exemplo — PostgreSQL:**
```germanio
banco
  driver: postgres
  host: "localhost"
  porta: "5432"
  nome: "minha_loja"
  usuario: "postgres"
  senha: "minhasenha"
```

**Exemplo — MySQL:**
```germanio
banco
  driver: mysql
  host: "localhost"
  porta: "3306"
  nome: "minha_loja"
  usuario: "root"
  senha: "minhasenha"
```

### 5.8 `integracoes` / `integrations`

Configura integrações externas (WhatsApp, e-mail, cron, webhooks).

**Exemplo:**
```germanio
integracoes

  email
    host: "smtp.gmail.com"
    porta: "587"
    usuario: "meu@gmail.com"
    senha: "minhasenha"

  cron
    cada 1 hora
      chamar "https://api.example.com/sync"

  cron
    cada 30 minutos
      chamar "https://api.example.com/limpar"
```

### 5.9 `tema` / `theme`

Personaliza a aparencia visual. Suporta presets, cores por nome, cores hex e estilos visuais.

**Gramatica:**
```
tema_block = "tema" [PRESET] NEWLINE tema_item*
tema_item  = "cor" COLOR_NAME (STRING | COLOR_WORD)
           | "estilo" STYLE_NAME
           | "escuro"
           | "icone" STRING
```

**Presets disponiveis:** `moderno`, `simples`, `elegante`, `corporativo`, `claro`

**Exemplo com preset:**
```germanio
tema moderno
```

**Exemplo com cores hex:**
```germanio
tema
  cor primaria "#6366f1"
  cor secundaria "#8b5cf6"
  cor destaque "#f59e0b"
  cor sidebar "#1e1b4b"
  escuro
  icone "rocket"
```

**Exemplo com cores por nome:**
```germanio
tema
  cor primaria azul
  cor destaque laranja
  estilo glassmorphism
```

**Nomes de cor disponiveis:** `azul`, `verde`, `vermelho`, `roxo`, `laranja`, `amarelo`, `rosa`, `escuro`, `claro`

**Estilos visuais:** `glassmorphism`, `flat`, `neumorphism`, `minimal`

**Chaves de cor:**

| Chave PT | Chave EN | Descricao |
|---|---|---|
| `primaria` | `primary` | Cor principal |
| `secundaria` | `secondary` | Cor secundaria |
| `destaque` | `accent` | Cor de destaque |
| `sidebar` | `sidebar` | Cor da barra lateral |

---

## 6. Tipos de Dados

### Tabela Completa de Tipos

| Tipo PT | Tipo EN | SQL (SQLite/PG) | SQL MySQL | Input HTML | Validação |
|---|---|---|---|---|---|
| `texto` | `text` | `TEXT` | `VARCHAR(500)` | `<input type="text">` | — |
| `numero` | `number` | `REAL` | `REAL` | `<input type="number">` | — |
| `data` | `date` | `DATETIME` | `DATETIME` | `<input type="date">` | — |
| `booleano` | `boolean` | `INTEGER` | `INTEGER` | `<input type="checkbox">` | — |
| `email` | `email` | `TEXT` | `VARCHAR(500)` | `<input type="email">` | Deve conter `@` e `.` |
| `telefone` | `phone` | `TEXT` | `VARCHAR(500)` | `<input type="tel">` | Mínimo 7 dígitos |
| `imagem` | `image` | `TEXT` | `VARCHAR(500)` | `<input type="file">` | — |
| `arquivo` | `file` | `TEXT` | `VARCHAR(500)` | `<input type="file">` | — |
| `upload` | `upload` | `TEXT` | `VARCHAR(500)` | `<input type="file">` | — |
| `link` | `link` | `TEXT` | `VARCHAR(500)` | `<input type="url">` | — |
| `status` | `status` | `TEXT` | `VARCHAR(500)` | `<select>` | — |
| `dinheiro` | `money` / `currency` | `REAL` | `REAL` | `<input type="number">` | — |
| `senha` | `password` | `TEXT` | `VARCHAR(500)` | `<input type="password">` | Mín. 6 chars |
| `texto_longo` | `long_text` | `TEXT` | `VARCHAR(500)` | `<textarea>` | — |
| `enum` | `enum` | `TEXT` | `VARCHAR(500)` | `<select>` | Valores definidos |

### Enum com Valores

A partir da v0.5.0, enums suportam lista de valores entre parenteses:

```germanio
dados
  pedido
    status: enum(pendente, aprovado, enviado, entregue, cancelado)
    prioridade: enum(baixa, media, alta)
```

O formulario gerado renderiza um `<select>` com as opcoes definidas.

### Notas de Tipo

**`booleano` / `boolean`:** Armazenado como `INTEGER` (0 = falso, 1 = verdadeiro). Use `padrao "true"` ou `padrao "false"`.

**`dinheiro` / `money`:** Armazenado como `REAL`. Para precisão monetária em produção, recomenda-se usar `numero` e operar em centavos.

**`senha` / `password`:** O valor recebido pela API é automaticamente hasheado com bcrypt (custo padrão) antes do armazenamento. O hash nunca é retornado nas respostas da API.

**`status`:** Campo especial que habilita agrupamento por status na rota `GET /api/_stats`. Recomenda-se usar com `padrao`.

**`imagem` / `arquivo` / `upload`:** Armazenam o caminho do arquivo no servidor (ex: `/uploads/1234567890.jpg`). O upload real é feito via `POST /upload`.

---

## 7. Modificadores de Campo

Modificadores são aplicados após o tipo do campo na mesma linha:

```germanio
campo: tipo modificador1 modificador2 ...
```

| Modificador PT | Modificador EN | Efeito SQL | Efeito de Validação |
|---|---|---|---|
| `obrigatorio` | `required` | `NOT NULL` | Erro se campo ausente ou vazio |
| `unico` | `unique` | `UNIQUE` | Erro se valor duplicado |
| `padrao "valor"` | `default "value"` | `DEFAULT 'valor'` | Preenche automaticamente se omitido |
| `indice` | `index` | Cria `INDEX` no campo | Acelera buscas neste campo |
| `soft_delete` | *(no model level)* | Adiciona coluna `deletado_em` | DELETE marca ao invés de remover |

**Exemplos:**

```germanio
dados

  produto
    nome: texto obrigatorio unico
    preco: dinheiro obrigatorio padrao "0"
    estoque: numero padrao "0" indice
    sku: texto unico indice
    ativo: booleano padrao "true"

  pedido soft_delete
    numero: texto unico obrigatorio
    status: status padrao "pendente"
```

---

## 8. Relacionamentos

### 8.1 `pertence_a` / `belongs_to` — Muitos para Um (N→1)

Cria uma chave estrangeira no modelo atual apontando para outro modelo.

```germanio
dados

  categoria
    nome: texto obrigatorio

  produto
    nome: texto obrigatorio
    categoria_id: numero pertence_a categoria
```

Isso gera:
```sql
"categoria_id" REAL REFERENCES "categoria"("id")
```

### 8.2 `tem_muitos` / `has_many` — Um para Muitos (1→N)

Declaracao semantica no modelo pai. Nao cria coluna extra — a FK fica no filho. O Germanio gera automaticamente o endpoint de expansao `GET /api/{modelo}/{id}/{relacao}`.

```germanio
dados

  cliente
    nome: texto
    pedidos: tem_muitos pedido

  pedido
    valor: dinheiro
    cliente_id: numero pertence_a cliente
```

Exemplo de uso via API:
```bash
# Buscar todos os pedidos do cliente 1
GET /api/cliente/1/pedidos
```

### 8.3 `muitos_para_muitos` / `many_to_many` — N→N

Cria automaticamente uma join table intermediaria. Por exemplo, `produto_categoria` com colunas `produto_id` e `categoria_id`.

```germanio
dados

  produto
    nome: texto
    categorias: muitos_para_muitos categoria

  categoria
    nome: texto
    produtos: muitos_para_muitos produto
```

Join table gerada automaticamente:
```sql
CREATE TABLE IF NOT EXISTS "produto_categoria" (
  "produto_id" INTEGER REFERENCES "produto"("id"),
  "categoria_id" INTEGER REFERENCES "categoria"("id"),
  PRIMARY KEY ("produto_id", "categoria_id")
);
```

---

## 9. Componentes de Tela

### 9.1 `lista` / `list`

Exibe registros de um modelo em forma de lista ou tabela.

```germanio
lista <modelo>
  mostrar <campo>
  mostrar <campo>
```

**Exemplo:**
```germanio
lista produto
  mostrar nome
  mostrar preco
  mostrar ativo
```

### 9.2 `mostrar` / `show`

Exibe um campo específico dentro de um componente `lista` ou na tela.

```germanio
mostrar nome
mostrar email
mostrar criado_em
```

### 9.3 `botao` / `button`

Componente clicável com cor e texto opcionais.

```germanio
botao <cor>
  texto "Rótulo do Botão"
```

**Cores predefinidas:** `azul` (blue), `verde` (green), `vermelho` (red) — e qualquer outro identificador de cor.

**Exemplo:**
```germanio
botao azul
  texto "Novo Produto"

botao vermelho
  texto "Excluir"

botao verde
  texto "Salvar"
```

### 9.4 `formulario` / `form`

Gera um formulário para criação/edição de um modelo.

```germanio
formulario <modelo>
```

**Exemplo:**
```germanio
formulario produto
```

### 9.5 `busca` / `search`

Campo de busca em tempo real.

```germanio
busca <modelo>
```

### 9.6 `titulo` / `title`

Define o título visível da tela.

```germanio
titulo "Gerenciamento de Produtos"
```

### 9.7 `entrada` / `input`

Campo de entrada individual dentro de um formulário.

```germanio
entrada nome
entrada email obrigatorio
```

### 9.8 `selecionar` / `select`

Dropdown de seleção.

```germanio
selecionar status
  opcao "ativo"
  opcao "inativo"
```

### 9.9 `area_texto` / `textarea`

Área de texto para conteúdo longo.

```germanio
area_texto descricao
```

### 9.10 `grafico` / `chart`

Componente de visualização gráfica.

```germanio
grafico vendas
```

### 9.11 `dashboard`

Painel de controle com métricas agregadas.

```germanio
dashboard
  titulo "Painel Administrativo"
```

---

## 10. Eventos

Eventos associam gatilhos de interface a ações.

**Sintaxe:**
```germanio
quando <gatilho> [<alvo>]
  <acao> [<argumento>]
```

### Gatilhos Disponíveis

| Gatilho PT | Gatilho EN | Descrição |
|---|---|---|
| `clicar` | `click` | Usuário clicou em um botão |
| `enviar` | `send` | Formulário foi enviado |
| `criar` | `create` | Registro foi criado |
| `atualizar` | `update` | Registro foi atualizado |
| `deletar` | `delete` | Registro foi deletado |

### Exemplos

```germanio
eventos

  quando clicar "Novo Produto"
    criar produto

  quando clicar "Salvar"
    atualizar produto

  quando clicar "Excluir"
    deletar produto

  quando enviar formulario
    salvar cliente

  quando criar pedido
    notificar cliente
```

---

## 11. Lógica e Regras

O bloco `logica` permite definir regras de negócio condicionais.

### 11.1 Regras Condicionais

```germanio
logica

  se <campo> <operador> <valor>
    <acao> <argumento>
```

**Exemplo:**
```germanio
logica

  se status igual "aprovado"
    mudar cor verde

  se valor maior 5000
    notificar administrador

  se estoque menor 10
    alertar "Estoque baixo"
```

### 11.2 Validações

```germanio
logica

  validar email obrigatorio
  validar senha obrigatorio
  validar cpf unico
```

### 11.3 Operadores de Condição

| Operador PT | Operador EN | Símbolo lógico |
|---|---|---|
| `igual` | `equals` / `equal` | `==` |
| `maior` | `greater` | `>` |
| `menor` | `less` | `<` |
| `e` | `and` | `&&` |
| `ou` | `or` | `\|\|` |

---

## 12. Autenticação

### 12.1 Configuração

```germanio
autenticacao

  usuario cliente        # modelo que representa os usuários
  login email            # campo usado para login
  permissao admin gerente  # papéis/roles disponíveis
```

### 12.2 Como Funciona

- Senhas armazenadas com **bcrypt** (custo padrão 10)
- Tokens **JWT** com algoritmo **HMAC-SHA256** (HS256)
- Expiração do token: **72 horas**
- Token passado via header `Authorization: Bearer <token>` ou query `?token=<token>`

### 12.3 Claims do JWT

```json
{
  "id": 42,
  "login": "usuario@email.com",
  "role": "admin",
  "exp": 1712765432
}
```

### 12.4 Cabeçalhos Injetados pelo Middleware

Após validação do token, o middleware injeta:

| Header | Valor |
|---|---|
| `X-User-ID` | ID do usuário autenticado |
| `X-User-Login` | Login (email/username) do usuário |
| `X-User-Role` | Papel (role) do usuário |

### 12.5 Regras de Acesso

- **GET sem token:** Permitido por padrão (dados públicos)
- **POST/PUT/DELETE sem token:** Negado (401)
- **Rotas sempre públicas:** `/api/login`, `/api/registro`, `/api/register`, `/ws`, `/`
- **Tela pública:** Use `publico` na definição da tela

```germanio
telas

  tela home
    publico
    titulo "Bem-vindo"

  tela admin
    requer admin
    titulo "Painel Admin"
```

---

## 13. Integrações

### 13.1 WhatsApp

```germanio
integracoes

  whatsapp
    notificar quando criar pedido
      para telefone
      mensagem "Novo pedido #{id} recebido!"
```

### 13.2 E-mail

```germanio
integracoes

  email
    host: "smtp.gmail.com"
    porta: "587"
    usuario: "sistema@empresa.com"
    senha: "apppassword"
    de: "Sistema <sistema@empresa.com>"
```

**Notificação por e-mail:**
```germanio
integracoes

  notificar quando criar usuario
    canal email
    para email
    assunto "Bem-vindo ao sistema!"
    mensagem "Olá {nome}, sua conta foi criada."
```

### 13.3 Cron Jobs

Cron jobs executam em goroutines em background e suportam chamadas HTTP a URLs externas.

```germanio
integracoes

  cron
    cada 1 hora
      chamar "https://api.external.com/sync"

  cron
    cada 30 minutos
      chamar "https://meuapp.com/api/limpeza"

  cron
    cada 1 dia
      chamar "https://api.example.com/relatorio"
```

**Unidades de tempo suportadas:**

| PT | EN | Exemplo |
|---|---|---|
| `segundo(s)` | `second(s)` | `cada 30 segundos` |
| `minuto(s)` | `minute(s)` | `cada 5 minutos` |
| `hora(s)` | `hour(s)` | `cada 1 hora` |
| `dia(s)` | `day(s)` | `cada 1 dia` |

### 13.4 Webhook

```germanio
integracoes

  webhook quando criar pedido
    url "https://api.example.com/webhook"
    metodo "POST"
```

### 13.5 Templates de Mensagem

Campos do registro podem ser interpolados em mensagens usando `{nome_do_campo}`:

```
"Olá {nome}, seu pedido #{id} foi {status} em {criado_em}"
```

---

## 14. Tema

O tema define a aparência visual da interface gerada.

```germanio
tema
  cor primaria "#6366f1"      # cor principal (botões, links)
  cor secundaria "#8b5cf6"   # cor secundária
  cor destaque "#f59e0b"     # cor de destaque
  cor sidebar "#1e1b4b"      # cor da barra lateral
  escuro                      # ativa modo escuro
  icone "rocket"              # ícone do app (emoji ou nome)
```

**Valores padrão:**

| Propriedade | Padrão |
|---|---|
| `primaria` | `#6366f1` (índigo) |
| `secundaria` | `#8b5cf6` (violeta) |
| `destaque` | `#f59e0b` (âmbar) |
| `sidebar` | `#1e1b4b` (azul escuro) |
| `escuro` | `false` |
| `icone` | *(vazio)* |

---

## 15. Sistema de Importação

Germanio permite dividir o código em múltiplos arquivos `.ge` e importá-los.

### 15.1 Sintaxe

```germanio
# Importar tudo de um arquivo
importar "modelos.ge"

# Importar seleção específica
importar dados de "modelos.ge"
importar tela de "telas/clientes.ge"

# Inglês equivalente
import "models.ge"
import models from "models.ge"
```

### 15.2 Regras de Importação

- O caminho do arquivo é relativo ao arquivo principal
- Importações são processadas recursivamente (imports dentro de imports são resolvidos)
- Conflitos de modelos com mesmo nome: o último importado prevalece
- `Theme`, `Auth` e `Email`: somente o primeiro encontrado é usado (não são sobrescritos)
- Modelos, Telas, Eventos, Ações, Regras, Notificadores e Crons são **concatenados** (merged)

### 15.3 Exemplo de Projeto Multi-arquivo

```
minha-loja/
  inicio.ge         # arquivo principal
  modelos/
    produto.ge
    cliente.ge
    pedido.ge
  telas/
    produtos.ge
    clientes.ge
  auth.ge
  tema.ge
```

**inicio.ge:**
```germanio
sistema minhaLoja

importar "auth.ge"
importar "tema.ge"
importar dados de "modelos/produto.ge"
importar dados de "modelos/cliente.ge"
importar dados de "modelos/pedido.ge"
importar tela de "telas/produtos.ge"
importar tela de "telas/clientes.ge"
```

---

## 16. Banco de Dados

### 16.1 Drivers Suportados

| Driver | Configuração `driver:` | Padrão |
|---|---|---|
| SQLite | `sqlite` | Sim (default) |
| PostgreSQL | `postgres` / `postgresql` | Não |
| MySQL | `mysql` | Não |

### 16.2 Configurações por Driver

**SQLite:**
```germanio
banco
  driver: sqlite
  nome: "app.db"       # opcional, padrão: <sistema>.db
```
- Usa WAL (Write-Ahead Logging) para performance
- Foreign keys habilitadas automaticamente
- Arquivo criado automaticamente no diretório corrente

**PostgreSQL:**
```germanio
banco
  driver: postgres
  host: "localhost"    # padrão: localhost
  porta: "5432"        # padrão: 5432
  nome: "minha_db"    # padrão: <sistema>
  usuario: "postgres"  # padrão: postgres
  senha: "senha123"
```

**MySQL:**
```germanio
banco
  driver: mysql
  host: "localhost"    # padrão: localhost
  porta: "3306"        # padrão: 3306
  nome: "minha_db"    # padrão: <sistema>
  usuario: "root"      # padrão: root
  senha: "senha123"
```

### 16.3 Schema Gerado

Para cada modelo `produto` com campos `nome: texto` e `preco: dinheiro`:

**SQLite:**
```sql
CREATE TABLE IF NOT EXISTS "produto" (
  "id"           INTEGER PRIMARY KEY AUTOINCREMENT,
  "nome"         TEXT,
  "preco"        REAL,
  "criado_em"    DATETIME DEFAULT CURRENT_TIMESTAMP,
  "atualizado_em" DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

**PostgreSQL:**
```sql
CREATE TABLE IF NOT EXISTS "produto" (
  "id"           SERIAL PRIMARY KEY,
  "nome"         TEXT,
  "preco"        REAL,
  "criado_em"    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  "atualizado_em" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

---

## 17. Operadores

### Operadores Aritméticos

| Símbolo | Descrição |
|---|---|
| `+` | Adição |
| `-` | Subtração |
| `*` | Multiplicação |
| `/` | Divisão |

### Operadores de Comparação (em regras)

| PT | EN | Significado |
|---|---|---|
| `igual` | `equal` / `equals` | Igualdade (`==`) |
| `maior` | `greater` | Maior que (`>`) |
| `menor` | `less` | Menor que (`<`) |

### Operadores Lógicos (em regras)

| PT | EN | Significado |
|---|---|---|
| `e` | `and` | E lógico (`&&`) |
| `ou` | `or` | OU lógico (`\|\|`) |

### Operadores de Atribuição

| Símbolo | Descrição |
|---|---|
| `=` | Atribuição direta |
| `:` | Declaração de tipo de campo |

---

## 18. Palavras Reservadas

As seguintes palavras são reservadas e **não podem** ser usadas como nomes de modelos, campos ou variáveis:

**Blocos:** `sistema`, `system`, `dados`, `models`, `telas`, `screens`, `acoes`, `actions`, `eventos`, `events`, `integracoes`, `integrations`, `tema`, `theme`, `logica`, `logic`, `banco`, `database`, `db`, `autenticacao`, `auth`, `authentication`, `config`

**Tela:** `tela`, `screen`, `titulo`, `title`, `lista`, `list`, `mostrar`, `show`, `botao`, `button`, `formulario`, `form`, `entrada`, `input`, `busca`, `search`, `dashboard`, `grafico`, `chart`, `tabela`, `table`, `campo`, `field`, `selecionar`, `select`, `area_texto`, `textarea`

**Tipos:** `texto`, `text`, `numero`, `number`, `data`, `date`, `booleano`, `boolean`, `email`, `telefone`, `phone`, `imagem`, `image`, `arquivo`, `file`, `upload`, `link`, `status`, `dinheiro`, `money`, `currency`, `senha`, `password`, `texto_longo`, `long_text`, `enum`

**Lógica:** `se`, `if`, `senao`, `else`, `igual`, `equal`, `equals`, `maior`, `greater`, `menor`, `less`, `e`, `and`, `ou`, `or`, `entao`, `then`, `validar`, `validate`, `calcular`, `compute`, `definir`, `set`, `retornar`, `return`, `mudar`, `change`, `para`, `to`, `para_cada`, `for_each`, `funcao`, `function`, `tentar`, `try`, `erro`, `error`

**Eventos:** `quando`, `when`, `clicar`, `click`, `criar`, `create`, `atualizar`, `update`, `deletar`, `delete`, `enviar`, `send`

**Auth:** `login`, `registro`, `register`, `usuario`, `user`, `permissao`, `permission`, `requer`, `requires`, `admin`, `publico`, `public`

**Integrações:** `whatsapp`, `mensagem`, `message`, `notificar`, `notify`, `cron`, `cada`, `every`, `hora`, `hour`, `minuto`, `minute`, `chamar`, `call`, `api`, `webhook`, `pagamento`, `payment`

**Relacionamentos:** `pertence_a`, `belongs_to`, `tem_muitos`, `has_many`, `muitos_para_muitos`, `many_to_many`

**Modificadores:** `obrigatorio`, `required`, `unico`, `unique`, `padrao`, `default`, `indice`, `index`, `soft_delete`

**Importação:** `importar`, `import`, `de`, `from`

---

## 19. Gramática Formal (EBNF)

```ebnf
program         = import* sistema? block*
block           = dados_block | telas_block | eventos_block
                | logica_block | auth_block | banco_block
                | integracoes_block | tema_block | acoes_block

sistema_block   = "sistema" name NEWLINE

import_stmt     = "importar" (STRING | name "de" STRING) NEWLINE

dados_block     = "dados" NEWLINE model+
model           = name ("soft_delete")? NEWLINE field*
field           = name ":" type modifier* NEWLINE
type            = "texto" | "numero" | "data" | "booleano" | "email"
                | "telefone" | "imagem" | "arquivo" | "upload" | "link"
                | "status" | "dinheiro" | "senha" | "texto_longo" | "enum"
                | (* English equivalents *)
modifier        = "obrigatorio" | "required"
                | "unico" | "unique"
                | "indice" | "index"
                | ("padrao" | "default") STRING
                | ("pertence_a" | "belongs_to") name

telas_block     = "telas" NEWLINE screen+
screen          = "tela" name NEWLINE screen_item*
screen_item     = titulo_item | lista_item | botao_item
                | formulario_item | mostrar_item
titulo_item     = "titulo" STRING NEWLINE
lista_item      = "lista" name NEWLINE mostrar_item*
mostrar_item    = "mostrar" name NEWLINE
botao_item      = "botao" [name] NEWLINE ("texto" STRING NEWLINE)?
formulario_item = "formulario" name NEWLINE

eventos_block   = "eventos" NEWLINE event+
event           = "quando" trigger [STRING] NEWLINE action_stmt
trigger         = "clicar" | "enviar" | "criar" | "atualizar" | "deletar"
                | (* English equivalents *)
action_stmt     = name name? NEWLINE

logica_block    = "logica" NEWLINE rule*
rule            = "se" name operator value NEWLINE action_line
                | "validar" name modifier NEWLINE
operator        = "igual" | "maior" | "menor" | (* EN *)
value           = STRING | name | NUMBER
action_line     = action_kw name? NEWLINE
action_kw       = "mudar" | "definir" | "calcular" | "retornar" | "notificar"
                | (* EN equivalents *)

auth_block      = "autenticacao" NEWLINE auth_item*
auth_item       = "usuario" name NEWLINE
                | "login" name NEWLINE
                | "permissao" name+ NEWLINE
                | "publico" NEWLINE

banco_block     = "banco" NEWLINE kv_pair*
kv_pair         = name ":" (STRING | name) NEWLINE

tema_block      = "tema" NEWLINE tema_item*
tema_item       = "cor" name STRING NEWLINE
                | "escuro" NEWLINE
                | "icone" STRING NEWLINE

integracoes_block = "integracoes" NEWLINE integ_item*
integ_item      = email_config | cron_item | webhook_item
email_config    = "email" NEWLINE kv_pair*
cron_item       = "cron" NEWLINE "cada" NUMBER time_unit NEWLINE cron_action
time_unit       = "segundo" | "minuto" | "hora" | "dia" | (* EN + plural *)
cron_action     = "chamar" STRING NEWLINE | name name? NEWLINE

name            = IDENTIFIER
STRING          = '"' char* '"'
NUMBER          = DIGIT+ ('.' DIGIT+)?
NEWLINE         = '\n'
INDENT          = SPACE+
SPACE           = ' ' | '\t'
DIGIT           = '0'..'9'
LETTER          = 'a'..'z' | 'A'..'Z' | unicode_letter
IDENTIFIER      = (LETTER | '_') (LETTER | DIGIT | '_')*

rotas_block     = "rotas" NEWLINE route+
route           = METHOD PATH NEWLINE route_item*
route_item      = "consultar" STRING NEWLINE
                | "executar" STRING NEWLINE
                | "corpo" STRING NEWLINE

paginas_block   = "paginas" NEWLINE pagina+
pagina          = "pagina" name NEWLINE pagina_item*
pagina_item     = "caminho" STRING NEWLINE
                | "html" STRING NEWLINE

sidebar_block   = "sidebar" NEWLINE sidebar_item*
sidebar_item    = "item" STRING ("icone" STRING)? ("link" STRING)? NEWLINE
                | "separador" NEWLINE
```

---

## 20. Idiomas Suportados

O Germanio v0.5.0 suporta 20 idiomas. Cada idioma tem seu proprio conjunto de palavras-chave que mapeiam para os mesmos tokens internos.

| # | Idioma | Exemplo `sistema` | Exemplo `dados` |
|---|--------|-------------------|-----------------|
| 1 | Portugues | `sistema` | `dados` |
| 2 | Ingles | `system` | `models` |
| 3 | Espanhol | `sistema` | `datos` |
| 4 | Frances | `systeme` | `donnees` |
| 5 | Alemao | `system` | `daten` |
| 6 | Italiano | `sistema` | `dati` |
| 7 | Chines | 系统 | 数据 |
| 8 | Japones | システム | データ |
| 9 | Coreano | 시스템 | 데이터 |
| 10 | Arabe | نظام | بيانات |
| 11 | Hindi | प्रणाली | डेटा |
| 12 | Bengali | সিস্টেম | ডেটা |
| 13 | Russo | система | данные |
| 14 | Indonesio | sistem | data |
| 15 | Turco | sistem | veriler |
| 16 | Vietnamita | he_thong | du_lieu |
| 17 | Polones | system | dane |
| 18 | Holandes | systeem | gegevens |
| 19 | Tailandes | ระบบ | ข้อมูล |
| 20 | Suaili | mfumo | data |

Todos os idiomas podem ser misturados livremente no mesmo arquivo.

---

## 21. Tema Presets e Cores por Nome

### Presets

Presets sao configuracoes de tema pre-definidas que podem ser aplicadas com uma unica linha:

```germanio
tema moderno
```

| Preset | Estilo | Descricao |
|--------|--------|-----------|
| `moderno` | glassmorphism | Gradientes, transparencia, sombras suaves |
| `simples` | flat | Clean, sem distracao, cores solidas |
| `elegante` | serif | Tipografia refinada, espacamento generoso |
| `corporativo` | dark sidebar | Visual profissional, sidebar escura |
| `claro` | light | Fundo branco, cores leves e acessiveis |

### Cores por Nome

Alem de valores hex, cores podem ser definidas por nome:

```germanio
tema
  cor primaria azul
```

| Nome | Hex Aproximado |
|------|---------------|
| `azul` | `#3b82f6` |
| `verde` | `#10b981` |
| `vermelho` | `#ef4444` |
| `roxo` | `#8b5cf6` |
| `laranja` | `#f97316` |
| `amarelo` | `#f59e0b` |
| `rosa` | `#ec4899` |
| `escuro` | `#1e1b4b` |
| `claro` | `#f8fafc` |

### Estilos Visuais

```germanio
tema
  estilo glassmorphism
```

| Estilo | Descricao |
|--------|-----------|
| `glassmorphism` | Fundo translucido com blur, bordas de vidro |
| `flat` | Sem sombras, cores solidas, bordas definidas |
| `neumorphism` | Relevo suave, sombras internas e externas |
| `minimal` | Espacamento amplo, poucos elementos visuais |

---

## 22. Enum com Valores

Enums podem ser declarados com uma lista de valores entre parenteses:

```germanio
dados
  tarefa
    status: enum(aberta, em_progresso, concluida, cancelada)
    prioridade: enum(baixa, media, alta, critica)
```

**Sintaxe:**
```
field = name ":" "enum" "(" value ("," value)* ")"
value = IDENTIFIER
```

O formulario HTML gerado renderiza um `<select>` com cada valor como `<option>`. A validacao no backend garante que apenas os valores definidos sejam aceitos.

---

## 23. Rotas Customizadas

O bloco `rotas` permite definir endpoints de API personalizados alem dos CRUD automaticos.

```germanio
rotas

  GET /api/relatorio/vendas-mensais
    consultar "SELECT strftime('%Y-%m', criado_em) as mes, SUM(valor) as total FROM pedido GROUP BY mes"

  POST /api/acao/aprovar-pedido
    corpo "pedido_id"
    executar "UPDATE pedido SET status = 'aprovado' WHERE id = ?"
```

**Sintaxe:**
- Linha de rota: `METODO /caminho/da/rota`
- `consultar` - executa SELECT e retorna resultado como JSON
- `executar` - executa INSERT/UPDATE/DELETE
- `corpo` - define campos esperados no body JSON (usados como parametros `?`)

---

## 24. Paginas Customizadas

O bloco `paginas` permite criar paginas HTML servidas pelo Germanio.

```germanio
paginas

  pagina sobre
    caminho "/sobre"
    html """
      <h1>Sobre</h1>
      <p>Nossa empresa foi fundada em 2024.</p>
    """
```

**Sintaxe:**
- `pagina <nome>` - identificador da pagina
- `caminho "<path>"` - URL onde a pagina sera servida
- `html "<conteudo>"` - conteudo HTML da pagina (suporta strings multiline com `"""`)

As paginas utilizam o layout e tema da aplicacao automaticamente.

---

## 25. Sidebar Customizada

O bloco `sidebar` permite definir a estrutura da barra lateral do frontend.

```germanio
sidebar
  item "Dashboard" icone "home" link "/"
  item "Produtos" icone "box" link "/produtos"
  separador
  item "Config" icone "settings" link "/config"
```

**Sintaxe:**
- `item "<texto>" icone "<nome>" link "<caminho>"` - item de menu
- `separador` - linha divisoria entre grupos

Quando o bloco `sidebar` nao esta presente, a sidebar e gerada automaticamente a partir das telas definidas.

---

## 26. Funcoes Async

O Germanio v0.5.0 adiciona funcoes built-in para operacoes assincronas:

| Funcao | Descricao | Exemplo |
|--------|-----------|---------|
| `paralelo()` | Executa funcoes em paralelo | `paralelo(f1, f2, f3)` |
| `esperar()` | Aguarda resultado async | `esperar(promessa)` |
| `timeout()` | Define tempo maximo | `timeout(func, 5000)` |
| `chamar_async()` | HTTP request async | `chamar_async("GET", url)` |
| `consultar_paralelo()` | Queries em paralelo | `consultar_paralelo(q1, q2)` |

Exemplo de uso:

```germanio
logica
  resultados = consultar_paralelo(
    "SELECT COUNT(*) FROM cliente",
    "SELECT SUM(valor) FROM pedido"
  )
```

---

## 27. Array Indexing

O Germanio v0.5.0 suporta acesso a elementos de arrays e campos aninhados por indice:

```germanio
arr = [10, 20, 30]
primeiro = arr[0]           // 10
ultimo = arr[2]             // 30

obj = {"itens": [1, 2, 3]}
item = obj.itens[0]         // 1
```

**Sintaxe:**
```
array_access = name "[" NUMBER "]"
nested_access = name "." name "[" NUMBER "]"
```

Indices comecam em 0. Acesso fora dos limites retorna `nulo` / `null`.
