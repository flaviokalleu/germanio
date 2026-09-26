# Germanio — Referência da API REST (API Reference)

> Versao 0.5.0 | Ultima atualizacao: 2026-04-10

---

## Sumário

1. [Visao Geral](#1-visão-geral)
2. [Autenticacao](#2-autenticação)
3. [Endpoints de Modelos (CRUD)](#3-endpoints-de-modelos-crud)
4. [Parametros de Query](#4-parâmetros-de-query)
5. [Formato de Resposta](#5-formato-de-resposta)
6. [Endpoints de Autenticacao](#6-endpoints-de-autenticação)
7. [Endpoints Especiais](#7-endpoints-especiais)
8. [Exportacao de Dados](#8-exportação-de-dados)
9. [Soft Delete e Restauracao](#9-soft-delete-e-restauração)
10. [Upload de Arquivos](#10-upload-de-arquivos)
11. [WebSocket](#11-websocket)
12. [Proxy de API](#12-proxy-de-api)
13. [Erros](#13-erros)
14. [Cabecalhos HTTP](#14-cabeçalhos-http)
15. [Referencia Rapida (Tabela)](#15-referência-rápida-tabela)
16. [Rotas Customizadas](#16-rotas-customizadas)
17. [Relacionamentos (Expansao)](#17-relacionamentos-expansao)
18. [Rate Limiting](#18-rate-limiting)
19. [Limites de Body](#19-limites-de-body)

---

## 1. Visão Geral

O Germanio gera automaticamente uma API REST completa para cada modelo declarado no arquivo `.ge`. O servidor roda por padrão na porta `8080`.

**Base URL:** `http://localhost:8080`

**Convenções:**
- Todas as respostas são em JSON (`Content-Type: application/json`)
- Nomes de modelos e campos são sempre em **minúsculas** na API
- Datas são retornadas no formato `YYYY-MM-DD HH:MM:SS` (SQLite) ou ISO 8601 (PostgreSQL)
- IDs são inteiros positivos sequenciais

**Dado um modelo:**
```germanio
dados
  produto
    nome: texto obrigatorio
    preco: dinheiro
    ativo: booleano
```

Os seguintes endpoints são gerados automaticamente:

| Método | Rota | Ação |
|---|---|---|
| `GET` | `/api/produto` | Listar todos os produtos |
| `GET` | `/api/produto/{id}` | Buscar produto por ID |
| `POST` | `/api/produto` | Criar novo produto |
| `PUT` | `/api/produto/{id}` | Atualizar produto |
| `DELETE` | `/api/produto/{id}` | Deletar produto |

---

## 2. Autenticação

### 2.1 Token Bearer (JWT)

Quando o bloco `autenticacao` está configurado, a maioria dos endpoints exige autenticação via token JWT.

**Header de autorização:**
```
Authorization: Bearer <token>
```

**Alternativa via query string:**
```
GET /api/produto?token=<token>
```

### 2.2 Regras de Acesso

| Cenário | GET | POST / PUT / DELETE |
|---|---|---|
| Sem autenticação configurada | Livre | Livre |
| Com autenticação, sem token | Permitido | Negado (401) |
| Com autenticação, token válido | Permitido | Permitido |
| Com autenticação, token expirado | Negado (401) | Negado (401) |

### 2.3 Rotas Sempre Públicas

As seguintes rotas nunca exigem token, independente da configuração:

- `POST /api/login`
- `POST /api/registro`
- `POST /api/register`
- `GET /ws` (WebSocket)
- `GET /` (frontend)
- `GET /health`

### 2.4 Estrutura do Token JWT

O token usa algoritmo **HS256** (HMAC-SHA256) e expira em **72 horas**.

**Payload decodificado:**
```json
{
  "id": 1,
  "login": "admin@empresa.com",
  "role": "admin",
  "exp": 1712852400
}
```

---

## 3. Endpoints de Modelos (CRUD)

### 3.1 Listar Registros

```
GET /api/{modelo}
```

Retorna uma lista paginada de registros. O total de registros é informado no header `X-Total-Count`.

**curl:**
```bash
curl http://localhost:8080/api/produto
```

**Com autenticação:**
```bash
curl -H "Authorization: Bearer SEU_TOKEN" \
  http://localhost:8080/api/produto
```

**Resposta (200 OK):**
```json
[
  {
    "id": 1,
    "nome": "Notebook Pro",
    "preco": 4999.90,
    "ativo": 1,
    "criado_em": "2026-01-15 10:30:00",
    "atualizado_em": "2026-01-15 10:30:00"
  },
  {
    "id": 2,
    "nome": "Mouse Gamer",
    "preco": 299.90,
    "ativo": 1,
    "criado_em": "2026-01-16 09:00:00",
    "atualizado_em": "2026-01-16 09:00:00"
  }
]
```

**Headers de resposta:**
```
Content-Type: application/json
X-Total-Count: 42
```

---

### 3.2 Buscar por ID

```
GET /api/{modelo}/{id}
```

Retorna um único registro pelo seu ID.

**curl:**
```bash
curl http://localhost:8080/api/produto/1
```

**Resposta (200 OK):**
```json
{
  "id": 1,
  "nome": "Notebook Pro",
  "preco": 4999.90,
  "ativo": 1,
  "criado_em": "2026-01-15 10:30:00",
  "atualizado_em": "2026-01-15 10:30:00"
}
```

**Resposta (404 Not Found):**
```json
{
  "erro": "registro 99 não encontrado"
}
```

---

### 3.3 Criar Registro

```
POST /api/{modelo}
Content-Type: application/json
```

Cria um novo registro. Campos `id`, `criado_em` e `atualizado_em` são gerados automaticamente.

**curl:**
```bash
curl -X POST http://localhost:8080/api/produto \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer SEU_TOKEN" \
  -d '{
    "nome": "Teclado Mecânico",
    "preco": 549.90,
    "ativo": true
  }'
```

**Resposta (201 Created):**
```json
{
  "id": 3,
  "nome": "Teclado Mecânico",
  "preco": 549.90,
  "ativo": 1,
  "criado_em": "2026-04-09 14:22:00",
  "atualizado_em": "2026-04-09 14:22:00"
}
```

**Resposta de erro — campo obrigatório faltando (400 Bad Request):**
```json
{
  "erro": "campo 'nome' é obrigatório"
}
```

**Efeitos colaterais ao criar:**
- Evento WebSocket `{ "type": "criar", "model": "produto", "id": 3, "data": {...} }` é broadcast para todos os clientes conectados
- Notificadores configurados (WhatsApp, e-mail) são disparados

---

### 3.4 Atualizar Registro

```
PUT /api/{modelo}/{id}
Content-Type: application/json
```

Atualiza campos de um registro existente. Apenas os campos enviados são modificados; `atualizado_em` é atualizado automaticamente.

**curl:**
```bash
curl -X PUT http://localhost:8080/api/produto/3 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer SEU_TOKEN" \
  -d '{
    "preco": 499.90,
    "ativo": false
  }'
```

**Resposta (200 OK):**
```json
{
  "id": 3,
  "nome": "Teclado Mecânico",
  "preco": 499.90,
  "ativo": 0,
  "criado_em": "2026-04-09 14:22:00",
  "atualizado_em": "2026-04-09 15:00:00"
}
```

**Efeitos colaterais:**
- Evento WebSocket `{ "type": "atualizar", ... }` é broadcast
- Notificadores de `atualizar` são disparados

---

### 3.5 Deletar Registro

```
DELETE /api/{modelo}/{id}
```

Remove um registro pelo ID.

- Se o modelo tem `soft_delete`: preenche `deletado_em` com o timestamp atual (o registro permanece no banco mas some das listagens)
- Sem `soft_delete`: executa `DELETE FROM` permanentemente

**curl:**
```bash
curl -X DELETE http://localhost:8080/api/produto/3 \
  -H "Authorization: Bearer SEU_TOKEN"
```

**Resposta (204 No Content):** *(sem corpo)*

**Efeitos colaterais:**
- Evento WebSocket `{ "type": "deletar", "model": "produto", "id": 3 }` é broadcast

---

## 4. Parâmetros de Query

Todos os parâmetros de query aceitam tanto o nome em Português quanto em Inglês.

### 4.1 Paginação

| Parâmetro PT | Parâmetro EN | Tipo | Padrão | Descrição |
|---|---|---|---|---|
| `pagina` | `page` | inteiro | `1` | Número da página (começa em 1) |
| `limite` | `limit` | inteiro | `100` | Registros por página |

**Exemplo:**
```bash
# Página 2, 20 registros por página
curl "http://localhost:8080/api/produto?pagina=2&limite=20"

# Equivalente em inglês
curl "http://localhost:8080/api/produto?page=2&limit=20"
```

**Como calcular total de páginas:**
```
total_paginas = ceil(X-Total-Count / limite)
```

---

### 4.2 Ordenação

| Parâmetro PT | Parâmetro EN | Tipo | Padrão | Descrição |
|---|---|---|---|---|
| `ordenar` | `sort` | string | `id` | Campo para ordenar |
| `ordem` | `order` | `asc` / `desc` | `desc` | Direção da ordenação |

**Exemplo:**
```bash
# Ordenar por nome, A-Z
curl "http://localhost:8080/api/produto?ordenar=nome&ordem=asc"

# Ordenar por preço, maior primeiro
curl "http://localhost:8080/api/produto?sort=preco&order=desc"

# Combinado com paginação
curl "http://localhost:8080/api/produto?ordenar=criado_em&ordem=desc&limite=10"
```

---

### 4.3 Busca Full-Text

| Parâmetro PT | Parâmetro EN | Tipo | Descrição |
|---|---|---|---|
| `busca` | `search` | string | Busca em todos os campos do tipo `TEXT` |

A busca usa `LIKE '%valor%'` em todos os campos de texto do modelo (OR entre eles).

**Exemplo:**
```bash
# Buscar produtos que contenham "notebook" em qualquer campo de texto
curl "http://localhost:8080/api/produto?busca=notebook"

# Equivalente em inglês
curl "http://localhost:8080/api/produto?search=notebook"
```

---

### 4.4 Filtros por Campo

Qualquer campo do modelo pode ser usado como filtro de igualdade exata:

```
GET /api/{modelo}?{campo}={valor}
```

**Exemplos:**
```bash
# Filtrar por campo booleano
curl "http://localhost:8080/api/produto?ativo=1"

# Filtrar por status
curl "http://localhost:8080/api/pedido?status=aprovado"

# Filtrar por chave estrangeira
curl "http://localhost:8080/api/pedido?cliente_id=5"

# Combinar filtros com paginação e ordenação
curl "http://localhost:8080/api/pedido?status=pendente&ordenar=criado_em&ordem=asc&limite=50"
```

---

### 4.5 Combinando Parâmetros

```bash
# Exemplo completo: produtos ativos, ordenados por preço, página 2, com 10 itens
curl "http://localhost:8080/api/produto?ativo=1&ordenar=preco&ordem=asc&pagina=2&limite=10"

# Busca com filtro e ordenação
curl "http://localhost:8080/api/cliente?busca=silva&ordenar=nome&ordem=asc"
```

---

## 5. Formato de Resposta

### 5.1 Resposta de Sucesso (lista)

```json
[
  { "id": 1, "campo1": "valor1", "campo2": 42, "criado_em": "...", "atualizado_em": "..." },
  { "id": 2, "campo1": "valor2", "campo2": 99, "criado_em": "...", "atualizado_em": "..." }
]
```

**Headers de lista:**
```
Content-Type: application/json
X-Total-Count: 150
```

### 5.2 Resposta de Sucesso (item único)

```json
{
  "id": 1,
  "nome": "Exemplo",
  "preco": 99.90,
  "ativo": 1,
  "criado_em": "2026-01-01 00:00:00",
  "atualizado_em": "2026-01-01 00:00:00"
}
```

### 5.3 Campos Sempre Presentes

Todo registro retornado pela API inclui:

| Campo | Tipo | Descrição |
|---|---|---|
| `id` | inteiro | Identificador único |
| `criado_em` | string (datetime) | Data/hora de criação |
| `atualizado_em` | string (datetime) | Data/hora da última atualização |
| `deletado_em` | string ou `null` | Presente apenas em modelos com `soft_delete` |

### 5.4 Mapeamento de Tipos

| Tipo Germanio | Tipo JSON | Exemplo |
|---|---|---|
| `texto` | string | `"Notebook Pro"` |
| `numero` | number | `42` ou `3.14` |
| `dinheiro` | number | `4999.90` |
| `booleano` | number (0/1) | `1` (true) ou `0` (false) |
| `data` | string | `"2026-04-09 14:00:00"` |
| `email` | string | `"user@example.com"` |
| `telefone` | string | `"+5511999999999"` |
| `imagem` | string | `"/uploads/1234567890.jpg"` |
| `arquivo` | string | `"/uploads/1234567890.pdf"` |
| `link` | string | `"https://example.com"` |
| `status` | string | `"aprovado"` |
| `enum` | string | `"opcao_a"` |
| `senha` | *(nunca retornado)* | — |

---

## 6. Endpoints de Autenticação

### 6.1 Login

```
POST /api/login
Content-Type: application/json
```

Autentica um usuário e retorna um token JWT.

**curl:**
```bash
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@empresa.com",
    "senha": "minhasenha123"
  }'
```

**Nota:** Os nomes dos campos dependem da configuração do bloco `autenticacao`. Se `login email` e o campo for `senha`, use `email` e `senha` no corpo.

**Resposta (200 OK):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6MSwibG9naW4iOiJhZG1pbkBlbXByZXNhLmNvbSIsInJvbGUiOiJhZG1pbiIsImV4cCI6MTcxMjg1MjQwMH0.signature",
  "id": 1,
  "role": "admin"
}
```

**Resposta (401 Unauthorized):**
```json
{
  "erro": "Credenciais inválidas"
}
```

**Resposta (400 Bad Request) — campos faltando:**
```json
{
  "erro": "Campo 'email' e 'senha' são obrigatórios"
}
```

---

### 6.2 Registro

```
POST /api/registro
POST /api/register
Content-Type: application/json
```

Cria um novo usuário com senha hasheada (bcrypt) e retorna token JWT.

**curl:**
```bash
curl -X POST http://localhost:8080/api/registro \
  -H "Content-Type: application/json" \
  -d '{
    "nome": "João Silva",
    "email": "joao@example.com",
    "senha": "senha123",
    "role": "usuario"
  }'
```

**Resposta (201 Created):**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "id": 5,
  "message": "Conta criada com sucesso"
}
```

**Resposta (409 Conflict) — email já cadastrado:**
```json
{
  "erro": "email já cadastrado"
}
```

**Resposta (400 Bad Request) — senha muito curta:**
```json
{
  "erro": "Senha deve ter no mínimo 6 caracteres"
}
```

**Regras de validação:**
- Campo de login (`email` por padrão) e campo de senha são obrigatórios
- Senha mínima: 6 caracteres
- O campo de login deve ser único na tabela
- Role padrão: `"usuario"` (se não enviado)

---

### 6.3 Dados do Usuário Autenticado

```
GET /api/me
Authorization: Bearer <token>
```

Retorna as informações do usuário autenticado extraídas do token.

**curl:**
```bash
curl http://localhost:8080/api/me \
  -H "Authorization: Bearer SEU_TOKEN"
```

**Resposta (200 OK):**
```json
{
  "id": "1",
  "login": "admin@empresa.com",
  "role": "admin"
}
```

**Resposta (401 Unauthorized) — sem token:**
```json
{
  "erro": "Não autenticado"
}
```

---

## 7. Endpoints Especiais

### 7.1 Health Check

```
GET /health
```

Verifica se o servidor está respondendo. Útil para load balancers e monitoramento.

**curl:**
```bash
curl http://localhost:8080/health
```

**Resposta (200 OK):**
```json
{"status":"ok"}
```

---

### 7.2 Estatísticas

```
GET /api/_stats
```

Retorna contagem de registros por modelo e breakdown por campo `status`.

**curl:**
```bash
curl http://localhost:8080/api/_stats \
  -H "Authorization: Bearer SEU_TOKEN"
```

**Resposta (200 OK):**
```json
{
  "produto": {
    "count": 150
  },
  "pedido": {
    "count": 42,
    "statuses": {
      "pendente": 15,
      "aprovado": 20,
      "cancelado": 5,
      "entregue": 2
    }
  },
  "cliente": {
    "count": 88
  }
}
```

**Notas:**
- A chave `statuses` só aparece se o modelo tiver um campo do tipo `status`
- Registros soft-deleted não são contados
- Apenas `GET` é suportado

---

## 8. Exportação de Dados

### 8.1 Exportar como CSV

```
GET /api/{modelo}/export/csv
```

Exporta todos os registros do modelo em formato CSV (UTF-8 com BOM para compatibilidade com Excel).

**curl:**
```bash
curl http://localhost:8080/api/produto/export/csv \
  -H "Authorization: Bearer SEU_TOKEN" \
  -o produto_2026-04-09.csv
```

**Headers de resposta:**
```
Content-Type: text/csv; charset=utf-8
Content-Disposition: attachment; filename="produto_2026-04-09.csv"
```

**Formato CSV:**
```csv
id,nome,preco,ativo,criado_em,atualizado_em
1,Notebook Pro,4999.9,1,2026-01-15 10:30:00,2026-01-15 10:30:00
2,Mouse Gamer,299.9,1,2026-01-16 09:00:00,2026-01-16 09:00:00
```

**Notas:**
- A primeira linha é o cabeçalho com todos os campos
- Campos na ordem: `id`, campos do modelo em ordem de declaração, `criado_em`, `atualizado_em`
- Registros soft-deleted **não** são incluídos
- Nome do arquivo segue o padrão: `{modelo}_{YYYY-MM-DD}.csv`

---

### 8.2 Exportar como JSON

```
GET /api/{modelo}/export/json
```

Exporta todos os registros em formato JSON (download como arquivo).

**curl:**
```bash
curl http://localhost:8080/api/produto/export/json \
  -H "Authorization: Bearer SEU_TOKEN" \
  -o produto_2026-04-09.json
```

**Headers de resposta:**
```
Content-Type: application/json; charset=utf-8
Content-Disposition: attachment; filename="produto_2026-04-09.json"
```

**Formato:**
```json
[
  {"id": 1, "nome": "Notebook Pro", "preco": 4999.9, ...},
  {"id": 2, "nome": "Mouse Gamer", "preco": 299.9, ...}
]
```

**Notas:**
- Registros soft-deleted **não** são incluídos
- Todos os campos são exportados (exceto `deletado_em`)
- Nome do arquivo: `{modelo}_{YYYY-MM-DD}.json`

---

## 9. Soft Delete e Restauração

### 9.1 Como Funciona o Soft Delete

Quando um modelo é declarado com `soft_delete`:

```germanio
dados
  pedido soft_delete
    numero: texto
    status: status
```

- `DELETE /api/pedido/{id}` **não remove** o registro do banco
- Em vez disso, define `deletado_em = CURRENT_TIMESTAMP`
- Registros com `deletado_em IS NOT NULL` são excluídos automaticamente de todas as listagens (`GET /api/pedido`)
- O registro ainda existe no banco e pode ser restaurado

### 9.2 Restaurar Registro Deletado

```
PUT /api/{modelo}/{id}/restaurar
Authorization: Bearer <token>
```

Restaura um registro soft-deleted, definindo `deletado_em = NULL`.

**curl:**
```bash
curl -X PUT http://localhost:8080/api/pedido/5/restaurar \
  -H "Authorization: Bearer SEU_TOKEN"
```

**Resposta (200 OK):**
```json
{
  "id": 5,
  "numero": "PED-2026-005",
  "status": "pendente",
  "deletado_em": null,
  "criado_em": "2026-03-01 10:00:00",
  "atualizado_em": "2026-04-09 14:30:00"
}
```

**Resposta (400 Bad Request) — modelo sem soft_delete:**
```json
{
  "erro": "modelo 'produto' não suporta soft delete"
}
```

**Efeitos colaterais:**
- Evento WebSocket `{ "type": "restaurar", "model": "pedido", "id": 5, "data": {...} }` é broadcast

---

## 10. Upload de Arquivos

### 10.1 Fazer Upload

```
POST /upload
Content-Type: multipart/form-data
```

Faz upload de um arquivo para o servidor. Retorna o caminho do arquivo salvo.

**Limite:** 32 MB por arquivo

**curl:**
```bash
# Upload de imagem
curl -X POST http://localhost:8080/upload \
  -F "file=@/caminho/para/imagem.jpg"

# Upload de PDF
curl -X POST http://localhost:8080/upload \
  -F "file=@/caminho/para/documento.pdf"
```

**Resposta (200 OK):**
```json
{
  "path": "/uploads/1712852400123456789.jpg",
  "name": "imagem.jpg"
}
```

**Notas:**
- O arquivo é salvo no diretório `uploads/` do servidor
- O nome do arquivo gerado usa timestamp em nanosegundos para garantir unicidade
- A extensão original é preservada
- Use o valor de `path` no campo correspondente do modelo (`imagem`, `arquivo`, etc.)

### 10.2 Acessar Arquivo Salvo

Arquivos enviados ficam disponíveis estaticamente em:

```
GET /uploads/{nome-do-arquivo}
```

**Exemplo:**
```bash
curl http://localhost:8080/uploads/1712852400123456789.jpg
```

### 10.3 Fluxo Completo de Upload

```bash
# 1. Fazer upload do arquivo
UPLOAD=$(curl -s -X POST http://localhost:8080/upload \
  -F "file=@produto.jpg")

# 2. Extrair o caminho
PATH_ARQUIVO=$(echo $UPLOAD | jq -r '.path')

# 3. Criar o produto com a imagem
curl -X POST http://localhost:8080/api/produto \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer SEU_TOKEN" \
  -d "{\"nome\": \"Produto com Foto\", \"foto\": \"$PATH_ARQUIVO\"}"
```

---

## 11. WebSocket

### 11.1 Conectar

```
GET /ws
Upgrade: websocket
```

**JavaScript:**
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = () => {
  console.log('Conectado ao Germanio WebSocket');
};

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  console.log('Evento recebido:', msg);
};

ws.onclose = () => {
  console.log('Conexão encerrada');
};
```

### 11.2 Formato das Mensagens

Todas as mensagens são JSON com o seguinte formato:

```json
{
  "type": "<tipo>",
  "model": "<nome-do-modelo>",
  "id": 42,
  "data": { ... }
}
```

### 11.3 Tipos de Eventos

| `type` | Descrição | `data` |
|---|---|---|
| `criar` | Novo registro criado | Objeto completo do registro |
| `atualizar` | Registro atualizado | Objeto atualizado do registro |
| `deletar` | Registro deletado | `null` (apenas `model` e `id`) |
| `restaurar` | Registro restaurado (soft delete) | Objeto restaurado do registro |

### 11.4 Exemplos de Mensagens

**Criação:**
```json
{
  "type": "criar",
  "model": "produto",
  "id": 10,
  "data": {
    "id": 10,
    "nome": "Novo Produto",
    "preco": 99.90,
    "criado_em": "2026-04-09 14:00:00",
    "atualizado_em": "2026-04-09 14:00:00"
  }
}
```

**Atualização:**
```json
{
  "type": "atualizar",
  "model": "produto",
  "id": 10,
  "data": {
    "id": 10,
    "nome": "Produto Atualizado",
    "preco": 89.90,
    "criado_em": "2026-04-09 14:00:00",
    "atualizado_em": "2026-04-09 15:00:00"
  }
}
```

**Deleção:**
```json
{
  "type": "deletar",
  "model": "produto",
  "id": 10,
  "data": null
}
```

### 11.5 Implementação Reativa (JavaScript)

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');
let produtos = [];

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);

  if (msg.model !== 'produto') return;

  switch (msg.type) {
    case 'criar':
      produtos.push(msg.data);
      renderizarLista();
      break;

    case 'atualizar':
      const idx = produtos.findIndex(p => p.id === msg.id);
      if (idx !== -1) produtos[idx] = msg.data;
      renderizarLista();
      break;

    case 'deletar':
      produtos = produtos.filter(p => p.id !== msg.id);
      renderizarLista();
      break;
  }
};
```

---

## 12. Proxy de API

O endpoint `/api/_proxy` permite que o frontend faça requisições a APIs externas através do servidor Germanio, contornando restrições de CORS.

### 12.1 Fazer Requisição Proxy

```
POST /api/_proxy
Content-Type: application/json
```

**Corpo da requisição:**

| Campo | Tipo | Obrigatório | Padrão | Descrição |
|---|---|---|---|---|
| `url` | string | Sim | — | URL da API externa |
| `method` | string | Não | `"GET"` | Método HTTP |
| `body` | string | Não | `""` | Corpo da requisição (JSON como string) |

**curl — GET externo:**
```bash
curl -X POST http://localhost:8080/api/_proxy \
  -H "Content-Type: application/json" \
  -d '{
    "method": "GET",
    "url": "https://api.exchangerate.host/latest?base=BRL"
  }'
```

**curl — POST externo:**
```bash
curl -X POST http://localhost:8080/api/_proxy \
  -H "Content-Type: application/json" \
  -d '{
    "method": "POST",
    "url": "https://api.externa.com/webhook",
    "body": "{\"evento\": \"novo_pedido\", \"id\": 42}"
  }'
```

**Resposta (200 OK):** Retorna o corpo da resposta da API externa como JSON.

**Resposta (502 Bad Gateway) — erro de conexão:**
```json
{
  "erro": "Get \"https://api.externa.com\": connection refused"
}
```

---

## 13. Erros

### 13.1 Formato de Erro

Todas as respostas de erro seguem o padrão:

```json
{
  "erro": "descrição do erro em português"
}
```

**Nota:** O campo de erro é sempre `"erro"` (não `"error"`).

### 13.2 Códigos HTTP

| Código | Situação |
|---|---|
| `200 OK` | Sucesso (GET, PUT) |
| `201 Created` | Criação bem-sucedida (POST) |
| `204 No Content` | Deleção bem-sucedida (DELETE) |
| `400 Bad Request` | Dados inválidos, campo obrigatório faltando, validação falhou |
| `401 Unauthorized` | Token ausente, inválido ou expirado |
| `404 Not Found` | Registro ou modelo não encontrado |
| `405 Method Not Allowed` | Método HTTP não suportado nesta rota |
| `409 Conflict` | Conflito (ex: email duplicado no registro) |
| `500 Internal Server Error` | Erro interno do servidor |
| `502 Bad Gateway` | Falha ao chamar API externa (proxy) |

### 13.3 Exemplos de Erros Comuns

**Modelo não encontrado:**
```json
{"erro": "modelo 'xyz' não existe"}
```

**ID inválido:**
```json
{"erro": "ID inválido"}
```

**Campo obrigatório:**
```json
{"erro": "campo 'nome' é obrigatório"}
```

**Email inválido:**
```json
{"erro": "email inválido no campo 'email'"}
```

**Telefone inválido:**
```json
{"erro": "telefone inválido no campo 'telefone'"}
```

**Nenhum campo para atualizar:**
```json
{"erro": "nenhum campo para atualizar"}
```

**Nenhum campo fornecido (POST):**
```json
{"erro": "nenhum campo fornecido"}
```

---

## 14. Cabeçalhos HTTP

### 14.1 Cabeçalhos de Segurança (sempre presentes)

| Header | Valor |
|---|---|
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `X-XSS-Protection` | `1; mode=block` |
| `Referrer-Policy` | `strict-origin-when-cross-origin` |

### 14.2 Cabeçalhos CORS (rotas `/api/*`)

| Header | Valor |
|---|---|
| `Access-Control-Allow-Origin` | `*` |
| `Access-Control-Allow-Methods` | `GET, POST, PUT, DELETE, OPTIONS` |
| `Access-Control-Allow-Headers` | `Content-Type` |

**Requisições preflight** (`OPTIONS`) retornam `200 OK` imediatamente.

### 14.3 Cabeçalhos de Resposta de Lista

| Header | Descrição |
|---|---|
| `X-Total-Count` | Total de registros (sem paginação) |
| `Content-Type` | `application/json` |

### 14.4 Cabeçalhos Injetados pelo Middleware de Auth

| Header | Descrição |
|---|---|
| `X-User-ID` | ID do usuário autenticado (string) |
| `X-User-Login` | Login do usuário (email/username) |
| `X-User-Role` | Papel do usuário |

---

## 15. Referência Rápida (Tabela)

### Rotas CRUD por Modelo

| Metodo | Rota | Descricao | Auth Necessaria |
|---|---|---|---|
| `GET` | `/api/{modelo}` | Listar registros | Nao (GET publico) |
| `GET` | `/api/{modelo}/{id}` | Buscar por ID | Nao (GET publico) |
| `POST` | `/api/{modelo}` | Criar registro | Sim |
| `PUT` | `/api/{modelo}/{id}` | Atualizar registro | Sim |
| `DELETE` | `/api/{modelo}/{id}` | Deletar registro | Sim |
| `GET` | `/api/{modelo}/{id}/{relacao}` | Expandir relacionamento | Nao (GET publico) |
| `GET` | `/api/{modelo}/export/csv` | Exportar CSV | Sim |
| `GET` | `/api/{modelo}/export/json` | Exportar JSON | Sim |
| `PUT` | `/api/{modelo}/{id}/restaurar` | Restaurar (soft delete) | Sim |

### Rotas Fixas

| Método | Rota | Descrição | Auth Necessária |
|---|---|---|---|
| `POST` | `/api/login` | Autenticar usuário | Não |
| `POST` | `/api/registro` | Registrar usuário | Não |
| `POST` | `/api/register` | Registrar usuário (EN) | Não |
| `GET` | `/api/me` | Dados do usuário atual | Sim |
| `GET` | `/api/_stats` | Estatísticas por modelo | Não (GET público) |
| `POST` | `/api/_proxy` | Proxy para APIs externas | Não |
| `GET` | `/health` | Health check | Não |
| `GET` | `/ws` | WebSocket | Não |
| `POST` | `/upload` | Upload de arquivo | Não |
| `GET` | `/uploads/{arquivo}` | Servir arquivo estático | Não |
| `GET` | `/` | Frontend gerado | Não |

### Parâmetros de Query Suportados

| Parâmetro (PT / EN) | Tipo | Padrão | Rota |
|---|---|---|---|
| `pagina` / `page` | int | `1` | `GET /api/{modelo}` |
| `limite` / `limit` | int | `100` | `GET /api/{modelo}` |
| `ordenar` / `sort` | string | `id` | `GET /api/{modelo}` |
| `ordem` / `order` | `asc`/`desc` | `desc` | `GET /api/{modelo}` |
| `busca` / `search` | string | — | `GET /api/{modelo}` |
| `{campo}` | string | — | `GET /api/{modelo}` |
| `token` | string | — | Qualquer rota autenticada |

---

## 16. Rotas Customizadas

A partir da v0.5.0, o bloco `rotas` no arquivo `.ge` permite definir endpoints personalizados alem dos CRUD automaticos.

### 16.1 Rota GET Customizada

```
GET /api/relatorio/vendas
```

**curl:**
```bash
curl http://localhost:8080/api/relatorio/vendas \
  -H "Authorization: Bearer SEU_TOKEN"
```

**Resposta (200 OK):**
```json
[
  {"mes": "2026-01", "total": 15420.50},
  {"mes": "2026-02", "total": 22100.00}
]
```

### 16.2 Rota POST Customizada

```
POST /api/acao/aprovar-pedido
```

**curl:**
```bash
curl -X POST http://localhost:8080/api/acao/aprovar-pedido \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer SEU_TOKEN" \
  -d '{"pedido_id": 42}'
```

**Resposta (200 OK):**
```json
{"message": "OK", "affected": 1}
```

### 16.3 Autenticacao

Rotas customizadas seguem as mesmas regras de autenticacao da API principal:
- GET: publico por padrao (se auth configurada)
- POST/PUT/DELETE: requer token JWT

---

## 17. Relacionamentos (Expansao)

### 17.1 Expandir Relacionamento

```
GET /api/{modelo}/{id}/{relacao}
```

Retorna os registros relacionados a um registro especifico.

**Exemplo:** Buscar todos os pedidos do cliente 1:

```bash
curl http://localhost:8080/api/cliente/1/pedidos
```

**Resposta (200 OK):**
```json
[
  {"id": 10, "valor": 150.00, "status": "pago", "cliente_id": 1, "criado_em": "..."},
  {"id": 15, "valor": 89.90, "status": "pendente", "cliente_id": 1, "criado_em": "..."}
]
```

**Resposta (404 Not Found) — registro nao existe:**
```json
{"erro": "registro 99 nao encontrado"}
```

**Resposta (400 Bad Request) — relacao nao existe:**
```json
{"erro": "relacao 'xyz' nao encontrada no modelo 'cliente'"}
```

### 17.2 Relacionamentos Suportados

| Tipo | Exemplo de Rota | Descricao |
|------|----------------|-----------|
| `tem_muitos` | `GET /api/cliente/1/pedidos` | Retorna registros filhos |
| `muitos_para_muitos` | `GET /api/produto/1/categorias` | Retorna registros via join table |

---

## 18. Rate Limiting

A partir da v0.5.0, o Germanio inclui rate limiting nativo.

### Limites

| Metodo | Limite | Janela |
|--------|--------|--------|
| POST | 100 requisicoes | 1 minuto |
| PUT | 100 requisicoes | 1 minuto |
| DELETE | 100 requisicoes | 1 minuto |
| GET | sem limite | — |

### Resposta quando limite excedido

**Resposta (429 Too Many Requests):**
```json
{"erro": "limite de requisicoes excedido, tente novamente em 1 minuto"}
```

### Headers de Rate Limiting

O servidor nao retorna headers `X-RateLimit-*` nesta versao. O rate limiter usa limpeza automatica periodica para liberar memoria.

---

## 19. Limites de Body

Requisicoes POST e PUT tem limite de tamanho de body:

| Tipo | Limite |
|------|--------|
| JSON body (POST/PUT) | 1 MB |
| Upload multipart | 32 MB |

**Resposta quando limite excedido (413 Payload Too Large):**
```json
{"erro": "tamanho do body excede o limite de 1MB"}
```
