# Guia de Seguranca - Germanio

> **Escopo desta referência:** Referência técnica de segurança, não garantia de segurança total nem contrato único para todos os modos. Na camada de intenção, autorização, relações, transições e invariantes são declarativos conforme [INTENCAO.md](INTENCAO.md). Exemplos de JWT/rotas abaixo não exigem implementação manual de autenticação pelo iniciante.

Este documento descreve todos os mecanismos de seguranca do Germanio: autenticacao, autorizacao, protecao de dados e boas praticas.

---

## Indice

1. [Correcoes de Seguranca v0.5.0](#correcoes-de-seguranca-v050)
2. [Sistema de Autenticacao](#sistema-de-autenticacao)
3. [Como o JWT Funciona no Germanio](#como-o-jwt-funciona-no-germanio)
4. [Rotas Protegidas vs Publicas](#rotas-protegidas-vs-publicas)
5. [Roles e Permissoes](#roles-e-permissoes)
6. [Validacao de Entrada](#validacao-de-entrada)
7. [Protecao contra XSS](#protecao-contra-xss)
8. [Prevencao de SQL Injection](#prevencao-de-sql-injection)
9. [Headers de Seguranca](#headers-de-seguranca)
10. [Configuracao de CORS](#configuracao-de-cors)
11. [Senhas e Bcrypt](#senhas-e-bcrypt)
12. [Variaveis de Ambiente](#variaveis-de-ambiente)
13. [Boas Praticas](#boas-praticas)

---

## Correcoes de Seguranca v0.5.0

A versao 0.5.0 inclui 9 correcoes de seguranca importantes:

### 1. Auth Bypass Corrigido

Corrigido um cenario onde certas rotas de API podiam ser acessadas sem token JWT valido quando o header `Authorization` continha um formato inesperado. Agora todas as rotas protegidas validam rigorosamente o formato `Bearer <token>`.

### 2. Protecao SSRF no Proxy

O endpoint `/api/_proxy` agora valida URLs de destino para impedir Server-Side Request Forgery (SSRF). URLs apontando para enderecos internos (`127.0.0.1`, `localhost`, `10.x.x.x`, `192.168.x.x`, `169.254.x.x`) sao bloqueadas.

### 3. `/api/_eval` Requer Admin

O endpoint de avaliacao de expressoes (`/api/_eval`) agora exige autenticacao com role `admin`. Anteriormente podia ser acessado por qualquer usuario autenticado.

### 4. XSS Escaping em Valores do Usuario

Todos os valores inseridos por usuarios sao agora sanitizados com escaping HTML antes de serem renderizados no frontend. Caracteres `<`, `>`, `"`, `'` e `&` sao convertidos para entidades HTML.

### 5. Prevencao de Path Traversal em Imports

A instrucao `importar` agora valida caminhos de arquivo para impedir acesso a arquivos fora do diretorio do projeto. Sequencias como `../` e caminhos absolutos sao rejeitados.

### 6. Whitelist de Extensoes em Upload

O endpoint `/upload` agora aceita apenas extensoes de arquivo permitidas: `.jpg`, `.jpeg`, `.png`, `.gif`, `.webp`, `.pdf`, `.doc`, `.docx`, `.xls`, `.xlsx`, `.csv`, `.txt`, `.zip`. Arquivos com extensoes nao permitidas sao rejeitados com erro 400.

### 7. Limites de Body (1MB POST/PUT)

Requisicoes POST e PUT agora tem um limite de tamanho de body de 1MB. Requisicoes maiores retornam erro 413 (Payload Too Large). Uploads de arquivo mantem o limite de 32MB via multipart.

### 8. JWT Secret via Env Variable

O JWT secret agora pode ser configurado via variavel de ambiente `GERMANIO_JWT_SECRET`, que tem prioridade sobre o valor definido no arquivo `.ge`. Isso evita que segredos sejam commitados no codigo-fonte.

```bash
GERMANIO_JWT_SECRET="minha-chave-secreta-64-chars" germanio run app.ge
```

### 9. Protecao contra CSV Injection

A exportacao CSV agora sanitiza valores que comecam com `=`, `+`, `-`, `@`, `|` ou `\t`, prefixando-os com aspas simples para prevenir ataques de formula injection em planilhas.

---

## Sistema de Autenticacao

O Germanio implementa autenticacao baseada em **JWT** (JSON Web Tokens) com hashing de senhas via **bcrypt**. O sistema e declarado com o bloco `autenticacao` (ou `auth` em ingles).

### Configuracao Minima

```
autenticacao
  modelo: usuario
  campo_login: email
  campo_senha: senha
  roles: admin, usuario
```

### Todos os Campos do Bloco

| Campo PT            | Campo EN             | Descricao                                    |
|---------------------|----------------------|----------------------------------------------|
| `modelo`            | `model`              | Nome do modelo que representa o usuario      |
| `campo_login`       | `login_field`        | Campo usado para login (padrao: `email`)     |
| `campo_senha`       | `password_field`     | Campo da senha (padrao: `senha`)             |
| `roles`             | `roles`              | Lista de roles separadas por virgula         |
| `secret`/`segredo`  | `jwt_secret`         | Chave secreta do JWT (use em producao!)      |

### Modelo de Usuario

O modelo de usuario deve ter ao menos os campos de login e senha:

```
dados

  usuario
    nome: texto obrigatorio
    email: email obrigatorio unico
    senha: senha obrigatorio
    telefone: telefone
    role: texto
```

O tipo `senha` e tratado especialmente:
- **Armazenamento**: a senha e sempre hasheada com bcrypt antes de ser salva
- **Leitura**: a senha nunca aparece em respostas da API
- **Comparacao**: o Germanio compara via bcrypt.CompareHashAndPassword

### Endpoints de Autenticacao Gerados

Quando `autenticacao` esta habilitada, o Germanio gera automaticamente:

| Metodo | Endpoint            | Descricao                         |
|--------|---------------------|-----------------------------------|
| POST   | `/auth/register`    | Registra novo usuario             |
| POST   | `/auth/login`       | Autentica e retorna JWT           |
| GET    | `/auth/me`          | Retorna dados do usuario atual    |

#### Registro

```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "nome": "Joao Silva",
    "email": "joao@exemplo.com",
    "senha": "minhasenha123",
    "role": "usuario"
  }'
```

Resposta:

```json
{
  "message": "Usuario criado com sucesso",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

#### Login

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "joao@exemplo.com",
    "senha": "minhasenha123"
  }'
```

Resposta:

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "nome": "Joao Silva",
    "email": "joao@exemplo.com",
    "role": "usuario"
  }
}
```

#### Dados do Usuario Atual

```bash
curl http://localhost:8080/auth/me \
  -H "Authorization: Bearer eyJhbGc..."
```

---

## Como o JWT Funciona no Germanio

### Estrutura do Token

O JWT gerado pelo Germanio contem:

```json
{
  "header": {
    "alg": "HS256",
    "typ": "JWT"
  },
  "payload": {
    "user_id": 1,
    "email": "joao@exemplo.com",
    "role": "admin",
    "exp": 1714521600
  }
}
```

### Tempo de Expiracao

Por padrao, os tokens expiram em **24 horas**. Apos a expiracao, o usuario precisa fazer login novamente.

### Chave Secreta

A chave padrao e `germanio-secret-change-me`. **Sempre mude em producao:**

```
autenticacao
  secret: "minha-chave-secreta-muito-longa-e-aleatoria-aqui"
```

Ou via variavel de ambiente antes de iniciar o servidor.

### Validacao do Token

Em cada requisicao a uma rota protegida, o Germanio:

1. Le o header `Authorization: Bearer <token>`
2. Verifica a assinatura do JWT com a chave secreta
3. Verifica se o token expirou
4. Extrai o `user_id` e `role` do payload
5. Injeta esses dados no contexto da requisicao

Se o token for invalido ou expirado, a API retorna:

```json
{
  "error": "Token invalido ou expirado",
  "code": 401
}
```

### Usando o Token no Frontend

O frontend gerado pelo Germanio armazena o token no `localStorage` e o envia automaticamente em todas as requisicoes:

```javascript
// Automaticamente gerenciado pelo Germanio
headers: {
  'Authorization': `Bearer ${localStorage.getItem('germanio_token')}`
}
```

---

## Rotas Protegidas vs Publicas

### Comportamento Padrao

Quando `autenticacao` esta habilitada:
- Todas as rotas da API sao **protegidas** por padrao
- As rotas `/auth/login` e `/auth/register` sao sempre **publicas**
- O frontend de login e sempre **publico**

### Telas Publicas

Para tornar uma tela acessivel sem login:

```
telas

  tela catalogo
    titulo "Catalogo de Produtos"
    publico                         // esta tela nao requer login
    lista produto
      mostrar nome
      mostrar preco

  tela admin
    titulo "Painel Admin"
    requer admin                    // apenas usuarios com role "admin"
    lista usuario
      mostrar nome
      mostrar email
```

Palavras-chave:
- `publico` / `public` — acessivel sem autenticacao
- `requer <role>` / `requires <role>` — exige role especifica

### Rotas da API com Controle de Acesso

```
// Qualquer usuario autenticado pode listar produtos
GET /api/produto          -> requer autenticacao

// Apenas admins podem deletar
DELETE /api/produto/:id   -> requer role "admin"
```

O controle por role na API e configurado atraves dos roles definidos no bloco `autenticacao`.

---

## Roles e Permissoes

### Definindo Roles

```
autenticacao
  modelo: usuario
  campo_login: email
  campo_senha: senha
  roles: admin, gerente, vendedor, cliente
```

### Hierarquia de Roles

O Germanio usa um sistema simples de roles sem hierarquia implicita. Cada role e independente. Voce pode controlar o acesso por tela:

```
telas

  tela relatorios
    titulo "Relatorios"
    requer admin          // so admin acessa

  tela vendas
    titulo "Vendas"
    requer vendedor       // so vendedor acessa

  tela catalogo
    titulo "Catalogo"
    publico               // qualquer um acessa
```

### Atribuindo Roles

Ao criar um usuario via API:

```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "nome": "Maria Admin",
    "email": "maria@empresa.com",
    "senha": "senha-segura",
    "role": "admin"
  }'
```

### Exemplo Completo com Roles

```
sistema empresa

autenticacao
  modelo: usuario
  campo_login: email
  campo_senha: senha
  roles: admin, gerente, funcionario

dados

  usuario
    nome: texto obrigatorio
    email: email obrigatorio unico
    senha: senha obrigatorio
    role: texto
    departamento: texto

  relatorio
    titulo: texto obrigatorio
    conteudo: texto_longo
    criado_por: texto
    status: status

telas

  tela dashboard
    titulo "Dashboard"
    lista relatorio
      mostrar titulo
      mostrar status
    botao azul
      texto "Novo Relatorio"

  tela admin-usuarios
    titulo "Gerenciar Usuarios"
    requer admin
    lista usuario
      mostrar nome
      mostrar email
      mostrar role
    botao verde
      texto "Novo Usuario"

  tela relatorios-gerente
    titulo "Todos os Relatorios"
    requer gerente
    lista relatorio
      mostrar titulo
      mostrar criado_por
      mostrar status
```

---

## Validacao de Entrada

O Germanio valida os dados antes de salva-los no banco de dados.

### Modificadores de Validacao

```
dados

  usuario
    nome: texto obrigatorio       // nao pode ser vazio
    email: email obrigatorio unico // valida formato e unicidade
    telefone: telefone             // valida formato de telefone
    cpf: texto unico               // deve ser unico no banco
    idade: numero                  // deve ser numerico
```

### Tipos com Validacao Automatica

| Tipo         | Validacao Automatica                              |
|--------------|---------------------------------------------------|
| `email`      | Formato de email valido (RFC 5322)                |
| `telefone`   | Apenas digitos, DDD + numero                      |
| `senha`      | Hasheada com bcrypt, nunca retornada na API       |
| `numero`     | Deve ser numerico (inteiro ou decimal)            |
| `dinheiro`   | Deve ser numerico, armazenado como REAL           |
| `data`       | Deve ser data valida                              |
| `booleano`   | Deve ser true/false ou 1/0                        |

### Validacao no Bloco Logica

```
logica

  validar email obrigatorio unico
  validar preco maior 0
  validar quantidade maior 0
```

Syntax de validacao:

```
validar <campo> <condicao> [<valor>]

// Condicoes disponíveis:
validar campo obrigatorio        // campo nao pode ser vazio
validar campo unico              // campo deve ser unico
validar campo maior 0            // campo deve ser maior que 0
validar campo menor 100          // campo deve ser menor que 100
validar campo igual "ativo"      // campo deve ser igual ao valor
```

### Resposta de Erro de Validacao

Quando uma validacao falha, a API retorna HTTP 422:

```json
{
  "error": "Validacao falhou",
  "fields": {
    "email": "Email ja cadastrado",
    "preco": "Deve ser maior que 0"
  }
}
```

---

## Protecao contra XSS

O Germanio aplica escaping automatico de HTML em todas as saidas do frontend gerado.

### Como Funciona

O frontend e gerado em HTML/JavaScript com as seguintes protecoes:

- Todos os valores de texto sao renderizados via `textContent` (nao `innerHTML`)
- Strings vindas da API sao escapadas antes de exibicao
- Caracteres especiais como `<`, `>`, `"`, `'`, `&` sao convertidos para entidades HTML

### Valores Seguros no Frontend

```javascript
// Automaticamente gerado pelo Germanio - forma segura
element.textContent = data.nome;     // seguro
element.innerHTML = data.nome;       // NUNCA feito pelo Germanio
```

### Headers Anti-XSS

O servidor do Germanio envia automaticamente:

```
X-Content-Type-Options: nosniff
X-XSS-Protection: 1; mode=block
```

---

## Prevencao de SQL Injection

O Germanio usa **queries parametrizadas** em toda interacao com o banco de dados. Nunca ha concatenacao de strings SQL com dados do usuario.

### Como e Gerado

Para cada operacao CRUD, o runtime usa placeholders do driver:

```go
// O runtime do Germanio gera internamente:
db.Query("SELECT * FROM produto WHERE id = ?", userInput)
db.Exec("INSERT INTO usuario (nome, email) VALUES (?, ?)", nome, email)
```

Os placeholders `?` (SQLite/MySQL) ou `$1, $2` (PostgreSQL) sao resolvidos pelo driver, que nunca interpreta os valores como SQL.

### Campos Protegidos

Todos os dados vindos de:
- Body JSON das requisicoes POST/PUT
- Query params das requisicoes GET
- Path params (`:id`)

Sao tratados como valores parametrizados, nunca interpolados em SQL.

### Exemplo de Ataque Bloqueado

Um atacante que envie:
```json
{
  "nome": "'; DROP TABLE usuario; --"
}
```

Sera armazenado literalmente como o texto `'; DROP TABLE usuario; --` sem executar nenhum SQL.

---

## Headers de Seguranca

O servidor Germanio adiciona automaticamente os seguintes headers HTTP em todas as respostas:

| Header                        | Valor                                    | Protecao                        |
|-------------------------------|------------------------------------------|---------------------------------|
| `X-Content-Type-Options`      | `nosniff`                                | Previne MIME sniffing           |
| `X-Frame-Options`             | `DENY`                                   | Previne clickjacking            |
| `X-XSS-Protection`            | `1; mode=block`                          | Protecao XSS do navegador       |
| `Referrer-Policy`             | `strict-origin-when-cross-origin`        | Controle de referrer            |
| `Content-Security-Policy`     | `default-src 'self'`                     | Restricao de fontes de conteudo |

### X-Frame-Options: DENY

Impede que seu app seja embutido em `<iframe>` por outros sites, prevenindo ataques de clickjacking.

### X-Content-Type-Options: nosniff

Impede que o navegador tente adivinhar o tipo MIME de respostas, prevenindo execucao de scripts disfarçados.

---

## Configuracao de CORS

Por padrao, o Germanio permite requisicoes de qualquer origem em desenvolvimento. Para producao, configure o CORS de forma restritiva.

### Comportamento Padrao (Desenvolvimento)

```
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization
```

### Restricao por Origem (Producao)

Configure via variavel de ambiente antes de iniciar o servidor:

```bash
GERMANIO_CORS_ORIGIN=https://meusite.com ./germanio run app.ge
```

Ou monte o servidor atras de um reverse proxy (Nginx, Caddy) que gerencie CORS.

### CORS com Autenticacao

Quando o JWT e enviado pelo frontend, a requisicao usa `credentials: include`. Nesse caso, `Access-Control-Allow-Origin` nao pode ser `*` e deve ser a origem especifica.

---

## Senhas e Bcrypt

### Como as Senhas sao Armazenadas

O Germanio usa **bcrypt** com cost factor 12 (padrao seguro). Nunca armazena senhas em texto puro.

Fluxo de cadastro:
```
senha_usuario -> bcrypt.GenerateFromPassword(senha, 12) -> hash_armazenado
```

Fluxo de login:
```
senha_digitada + hash_banco -> bcrypt.CompareHashAndPassword -> ok/erro
```

### Por que bcrypt?

- Funcao de hash lenta por design — dificulta ataques de forca bruta
- Inclui salt automatico — previne ataques com rainbow tables
- Adaptavel — o cost factor pode ser aumentado no futuro

### O Tipo `senha`

Campos do tipo `senha` tem comportamento especial:

1. **Escrita**: sempre hasheada com bcrypt ao salvar
2. **Leitura**: nunca retornada em respostas GET da API
3. **Comparacao**: apenas via endpoint de login, usando bcrypt

```
dados

  usuario
    nome: texto
    email: email unico
    senha: senha    // nunca aparece em GET /api/usuario
```

---

## Variaveis de Ambiente

Nunca comite segredos no codigo-fonte. Use variaveis de ambiente para:
- Chave JWT
- Senha de banco de dados
- Senha de email SMTP
- Chaves de API externas

### .env Recomendado

```bash
# .env (adicione ao .gitignore!)
JWT_SECRET=chave-jwt-muito-longa-e-aleatoria-aqui-use-64-chars-ou-mais
DB_PASSWORD=senha-do-banco
EMAIL_PASSWORD=senha-do-smtp
```

### .gitignore

Sempre adicione ao `.gitignore`:

```
.env
*.env
whatsapp.db
*.db-shm
*.db-wal
```

### Gerando uma Chave JWT Segura

```bash
# Linux/Mac
openssl rand -base64 64

# Ou via Go
go run -e 'import "crypto/rand"; import "encoding/base64"; b := make([]byte, 48); rand.Read(b); fmt.Println(base64.StdEncoding.EncodeToString(b))'
```

---

## Boas Praticas

### 1. Mude o JWT Secret em Producao

O secret padrao `germanio-secret-change-me` e publico. Use:

```
autenticacao
  secret: "mude-este-valor-para-algo-muito-secreto-em-producao"
```

### 2. Use HTTPS em Producao

Sempre rode atrás de um reverse proxy com TLS:

```nginx
# Nginx com Let's Encrypt
server {
    listen 443 ssl;
    server_name meuapp.com;

    ssl_certificate /etc/letsencrypt/live/meuapp.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/meuapp.com/privkey.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### 3. Valide Todos os Campos Obrigatorios

```
dados

  usuario
    nome: texto obrigatorio        // nunca opcional
    email: email obrigatorio unico // formato + unicidade
    senha: senha obrigatorio       // bcrypt automatico
```

### 4. Use Roles Minimas Necessarias

Aplique o principio do menor privilegio:

```
telas

  tela dados-sensiveis
    requer admin      // apenas admins

  tela relatorios
    requer gerente    // gerentes e acima

  tela vendas
    // sem restricao = qualquer usuario autenticado
```

### 5. Limite o Acesso ao Banco de Dados

Para MySQL/PostgreSQL em producao, crie um usuario dedicado com permissoes minimas:

```sql
-- MySQL
CREATE USER 'germanio_app'@'localhost' IDENTIFIED BY 'senha-forte';
GRANT SELECT, INSERT, UPDATE, DELETE ON minha_loja.* TO 'germanio_app'@'localhost';
```

### 6. Rate Limiting Nativo

A partir da v0.5.0, o Germanio inclui rate limiting nativo: 100 requisicoes POST por minuto por IP. Nao e mais necessario configurar no reverse proxy, mas voce pode adicionar limites extras:

```nginx
# Nginx (opcional, para protecao adicional)
limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;

location /api/ {
    limit_req zone=api burst=20 nodelay;
    proxy_pass http://localhost:8080;
}

location /auth/ {
    limit_req zone=api burst=5 nodelay;
    proxy_pass http://localhost:8080;
}
```

### 7. Monitore Logs de Autenticacao

O Germanio registra tentativas de login no stdout:

```
[auth] Login bem-sucedido: joao@exemplo.com
[auth] Tentativa de login falhou: email@invalido.com
[auth] Token expirado para usuario ID: 42
```

Direcione para um sistema de logs centralizado em producao.

### 8. Use Soft Delete para Dados Importantes

O Germanio suporta soft delete — registros sao marcados como deletados, nao removidos:

```
dados

  usuario soft_delete
    nome: texto
    email: email
```

Isso permite auditoria e recuperacao de dados.

### 9. Rotate o JWT Secret Periodicamente

Em producao, rotacione o JWT secret periodicamente. Isso invalida todos os tokens existentes e forca relogin — planeje uma janela de manutencao ou implemente refresh tokens.

### 10. Backup do Banco de Dados

Para SQLite:

```bash
# Backup diario
cp meuapp.db meuapp.db.$(date +%Y%m%d)

# Ou use o modo WAL para backup online
sqlite3 meuapp.db ".backup backup_$(date +%Y%m%d).db"
```

---

## Checklist de Seguranca para Producao

Antes de ir para producao, verifique:

- [ ] JWT Secret alterado para valor unico e longo
- [ ] Servidor rodando atrás de HTTPS (Nginx + Let's Encrypt)
- [ ] Arquivo `.env` nao commitado no repositorio
- [ ] `whatsapp.db` e arquivos `.db` no `.gitignore`
- [ ] Backup automatico do banco de dados configurado
- [ ] Rate limiting no reverse proxy
- [ ] Logs centralizados configurados
- [ ] Banco de dados MySQL/PostgreSQL com usuario de permissoes minimas
- [ ] Senha de app Gmail (nao senha normal) para SMTP
- [ ] Variaveis de ambiente configuradas no servidor de producao
- [ ] Firewall bloqueando acesso direto a porta 8080 (apenas Nginx acessa)
