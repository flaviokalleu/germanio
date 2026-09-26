# Guia de Integracoes - Germanio

Este documento cobre todas as integracoes disponíveis no Germanio: WhatsApp, Email SMTP, Cron Jobs, HTTP Client e Webhooks.

---

## Indice

1. [Visao Geral](#visao-geral)
2. [WhatsApp](#whatsapp)
3. [Email SMTP](#email-smtp)
4. [Cron Jobs](#cron-jobs)
5. [HTTP Client](#http-client)
6. [Webhooks](#webhooks)
7. [Pagamentos (Roadmap)](#pagamentos)
8. [Combinando Integracoes](#combinando-integracoes)

---

## Visao Geral

Todas as integracoes sao declaradas dentro do bloco `integracoes` (ou `integrations` em ingles).

```
integracoes

  whatsapp
    ...

  email
    ...

  cron
    ...
```

O bloco `integracoes` pode conter quantas integracoes forem necessarias. A ordem nao importa.

---

## WhatsApp

O Germanio usa a biblioteca **whatsmeow** para enviar mensagens pelo WhatsApp Business. A autenticacao e feita via QR Code no terminal, e a sessao e persistida em um arquivo `.db` local para que futuras execucoes nao exijam nova autenticacao.

### Como Funciona

1. Na primeira execucao, o Germanio exibe um QR Code no terminal
2. Voce abre o WhatsApp no celular > Dispositivos Conectados > Conectar Dispositivo
3. Escaneia o QR Code
4. A sessao e salva em `whatsapp.db` (ou o caminho configurado)
5. Nas proximas execucoes, a conexao e restaurada automaticamente

### Configuracao

Dentro do bloco `whatsapp`, voce define os gatilhos de envio:

```
integracoes

  whatsapp

    quando criar pedido
      enviar mensagem para telefone
        texto "Ola {cliente}! Seu pedido foi recebido. Total: R${valor}"

    quando atualizar pedido
      enviar mensagem para telefone
        texto "Atualizacao: seu pedido agora esta {status}"
```

### Sintaxe dos Gatilhos

| Portugues                     | Ingles                        | Descricao                           |
|-------------------------------|-------------------------------|-------------------------------------|
| `quando criar <modelo>`       | `when create <model>`         | Dispara ao criar um registro        |
| `quando atualizar <modelo>`   | `when update <model>`         | Dispara ao atualizar um registro    |
| `quando deletar <modelo>`     | `when delete <model>`         | Dispara ao deletar um registro      |

### Destino da Mensagem

O campo `enviar mensagem para <campo>` define qual campo do modelo contem o numero de telefone. O campo pode ser:

- `telefone` — campo chamado "telefone" no modelo atual
- `cliente.telefone` — campo "telefone" dentro de um relacionamento "cliente"

### Templates com Variaveis

Use `{nome_do_campo}` dentro do texto para inserir valores do registro:

```
texto "Ola {nome}! Pedido #{id} recebido. Valor: R${valor}. Status: {status}"
```

Variaveis disponiveis: qualquer campo do modelo que disparou o gatilho.

### Normalizacao de Telefone

O Germanio normaliza automaticamente os numeros:
- Remove caracteres nao numericos (`(`, `)`, `-`, espacos)
- Adiciona o codigo do Brasil `55` se nao estiver presente
- Numeros com menos de 10 digitos sao ignorados com aviso

Exemplos validos:
```
(11) 99999-9999   ->  5511999999999
+55 11 99999-9999 ->  5511999999999
11999999999       ->  5511999999999
```

### Exemplo Completo - Restaurante com WhatsApp

```
sistema restaurante

tema
  cor primaria "#6366f1"
  cor secundaria "#8b5cf6"

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

### Arquivo de Sessao

Por padrao, a sessao WhatsApp e salva em `whatsapp.db` na pasta do projeto. Voce pode versionar esse arquivo para manter a sessao entre deploys (nao recomendado para producao — use variaveis de ambiente).

---

## Email SMTP

O Germanio envia emails via **net/smtp** padrao do Go. Compativel com Gmail, Outlook, SendGrid, Mailgun, e qualquer servidor SMTP.

### Configuracao do Servidor

```
integracoes

  email
    servidor: "smtp.gmail.com"
    porta: "587"
    usuario: "seu@gmail.com"
    senha: "sua-senha-de-app"
    de: "Sistema <seu@gmail.com>"
```

| Campo       | Alternativas EN    | Descricao                          |
|-------------|--------------------|------------------------------------|
| `servidor`  | `server`, `host`   | Endereco do servidor SMTP          |
| `porta`     | `port`             | Porta SMTP (geralmente 587 ou 465) |
| `usuario`   | `user`, `username` | Login do servidor SMTP             |
| `senha`     | `password`, `pass` | Senha ou App Password              |
| `de`        | `from`, `remetente`| Nome e email do remetente          |

### Gmail - Configuracao

Para Gmail, voce precisa de uma **Senha de App** (nao a senha normal da conta):

1. Acesse myaccount.google.com
2. Seguranca > Verificacao em duas etapas (ative se nao tiver)
3. Seguranca > Senhas de app
4. Gere uma senha para "Outro aplicativo"
5. Use essa senha no campo `senha`

### Gatilhos de Email

```
integracoes

  email
    servidor: "smtp.gmail.com"
    porta: "587"
    usuario: "loja@gmail.com"
    senha: "xxxx-xxxx-xxxx-xxxx"

    quando criar pedido
      enviar email para cliente.email
        assunto "Pedido recebido - #{id}"
        texto "Ola {cliente}, seu pedido foi confirmado! Total: R${valor}"

    quando atualizar pedido
      enviar email para cliente.email
        assunto "Atualizacao do pedido #{id}"
        texto "Seu pedido foi atualizado. Novo status: {status}"
```

### Sintaxe de Email

```
quando <gatilho> <modelo>
  enviar email para <campo.email>
    assunto "Assunto aqui"
    texto "Corpo da mensagem com {variaveis}"
```

| Elemento          | Descricao                                       |
|-------------------|-------------------------------------------------|
| `assunto`         | Linha de assunto do email (`subject` em ingles) |
| `texto`           | Corpo do email em texto puro                    |
| `{variavel}`      | Substitui pelo valor do campo no registro       |

### Provedores Populares

| Provedor     | Servidor                  | Porta |
|--------------|---------------------------|-------|
| Gmail        | smtp.gmail.com            | 587   |
| Outlook      | smtp-mail.outlook.com     | 587   |
| Yahoo        | smtp.mail.yahoo.com       | 587   |
| SendGrid     | smtp.sendgrid.net         | 587   |
| Mailgun      | smtp.mailgun.org          | 587   |
| Amazon SES   | email-smtp.us-east-1.amazonaws.com | 587 |

### Exemplo Completo - Loja com Notificacao por Email

```
sistema loja

autenticacao
  modelo: usuario
  campo_login: email
  campo_senha: senha
  roles: admin, cliente

dados

  usuario
    nome: texto obrigatorio
    email: email obrigatorio unico
    senha: senha obrigatorio

  pedido
    cliente: texto obrigatorio
    produto: texto obrigatorio
    valor: dinheiro obrigatorio
    email_cliente: email
    status: status

telas

  tela pedidos
    titulo "Pedidos"
    lista pedido
      mostrar cliente
      mostrar produto
      mostrar valor
      mostrar status
    botao azul
      texto "Novo Pedido"

eventos

  quando clicar "Novo Pedido"
    criar pedido

integracoes

  email
    servidor: "smtp.gmail.com"
    porta: "587"
    usuario: "loja@gmail.com"
    senha: "app-password-aqui"
    de: "Loja Virtual <loja@gmail.com>"

    quando criar pedido
      enviar email para email_cliente
        assunto "Pedido Confirmado!"
        texto "Ola {cliente}! Seu pedido de {produto} foi confirmado. Total: R${valor}. Obrigado!"

    quando atualizar pedido
      enviar email para email_cliente
        assunto "Atualizacao do Pedido"
        texto "Seu pedido foi atualizado. Status atual: {status}"
```

---

## Cron Jobs

Os Cron Jobs permitem executar acoes periodicas sem precisar de ferramentas externas como crontab ou Task Scheduler.

### Sintaxe

```
integracoes

  cron

    cada 5 minutos
      chamar api "https://minha-api.com/health"

    cada 1 hora
      chamar api "https://minha-api.com/relatorio"

    cada 24 horas
      chamar api "https://minha-api.com/limpeza"
```

### Unidades de Tempo

| Portugues   | Ingles    | Exemplo               |
|-------------|-----------|-----------------------|
| `segundos`  | `seconds` | `cada 30 segundos`    |
| `minutos`   | `minutes` | `cada 5 minutos`      |
| `hora`      | `hour`    | `cada 1 hora`         |
| `horas`     | `hours`   | `cada 6 horas`        |
| `dia`       | `day`     | `cada 1 dia`          |
| `dias`      | `days`    | `cada 7 dias`         |

Palavra-chave em ingles: `every` (equivale a `cada`)

```
// Ingles
every 5 minutes
  call api "https://example.com/sync"
```

### Acao: chamar api

A acao `chamar api` (ou `call api`) faz uma requisicao HTTP GET para a URL especificada. A resposta e registrada no log do servidor.

```
cada 10 minutos
  chamar api "https://api.externa.com/sync"
```

O cliente HTTP usa:
- Timeout: 30 segundos
- User-Agent: `Germanio/1.0`
- Header `Accept: application/json`

### Acoes Genericas

Alem de `chamar api`, voce pode usar acoes descritivas que sao registradas no log:

```
cada 1 hora
  limpar sessoes

cada 1 dia
  gerar relatorio
```

Essas acoes sao registradas em log mas nao executam codigo personalizado na versao atual — use `chamar api` para acionar endpoints proprios.

### Exemplo Completo - App com Cron

```
sistema monitoramento

dados

  servico
    nome: texto obrigatorio
    url: link obrigatorio
    status: status
    ultimo_check: data

telas

  tela servicos
    titulo "Servicos Monitorados"
    lista servico
      mostrar nome
      mostrar url
      mostrar status
      mostrar ultimo_check
    botao azul
      texto "Novo Servico"

eventos

  quando clicar "Novo Servico"
    criar servico

integracoes

  cron

    cada 5 minutos
      chamar api "https://meuapp.com/api/check-services"

    cada 1 hora
      chamar api "https://meuapp.com/api/relatorio"

    cada 1 dia
      chamar api "https://meuapp.com/api/limpeza"
```

### Logs do Cron

Ao iniciar, o scheduler exibe os jobs configurados:

```
[cron] Agendado: chamar https://meuapp.com/api/check-services a cada 5m0s
[cron] Agendado: chamar https://meuapp.com/api/relatorio a cada 1h0m0s
[cron] Agendado: chamar https://meuapp.com/api/limpeza a cada 24h0m0s
```

Durante a execucao:

```
[cron] Chamando: https://meuapp.com/api/check-services
[cron] Resposta de https://meuapp.com/api/check-services: {"status":"ok"}
```

---

## HTTP Client

O Germanio expoe uma API REST automaticamente para todos os modelos. Alem disso, o runtime inclui um cliente HTTP interno usado pelos cron jobs e que pode ser acionado via endpoints proxy.

### API REST Automatica

Para cada modelo, o Germanio gera automaticamente:

| Metodo   | Endpoint               | Descricao               |
|----------|------------------------|-------------------------|
| `GET`    | `/api/<modelo>`        | Lista todos os registros|
| `GET`    | `/api/<modelo>/:id`    | Busca um registro       |
| `POST`   | `/api/<modelo>`        | Cria um registro        |
| `PUT`    | `/api/<modelo>/:id`    | Atualiza um registro    |
| `DELETE` | `/api/<modelo>/:id`    | Remove um registro      |

Exemplos para o modelo `produto`:

```bash
# Listar todos
curl http://localhost:8080/api/produto

# Buscar um
curl http://localhost:8080/api/produto/1

# Criar
curl -X POST http://localhost:8080/api/produto \
  -H "Content-Type: application/json" \
  -d '{"nome": "Notebook", "preco": 2999.99}'

# Atualizar
curl -X PUT http://localhost:8080/api/produto/1 \
  -H "Content-Type: application/json" \
  -d '{"status": "ativo"}'

# Remover
curl -X DELETE http://localhost:8080/api/produto/1
```

### Paginacao

A API suporta paginacao via query params:

```bash
curl "http://localhost:8080/api/produto?page=2&limit=20"
```

Resposta:

```json
{
  "data": [...],
  "total": 150,
  "page": 2,
  "limit": 20,
  "pages": 8
}
```

### Autenticacao na API

Se o sistema tiver `autenticacao` habilitada, endpoints protegidos exigem JWT:

```bash
# Login
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@loja.com", "senha": "123456"}'

# Resposta: {"token": "eyJhbGc..."}

# Usar o token
curl http://localhost:8080/api/produto \
  -H "Authorization: Bearer eyJhbGc..."
```

### Chamando APIs Externas (via Cron)

Use o bloco `cron` para chamar APIs externas periodicamente:

```
integracoes

  cron

    cada 15 minutos
      chamar api "https://api.cotacao.com/dolar"

    cada 1 hora
      chamar api "https://webhook.site/meu-webhook"
```

### Headers Automaticos

O cliente HTTP do Germanio envia automaticamente:

```
User-Agent: Germanio/1.0
Accept: application/json
Content-Type: application/json  (apenas em POST/PUT)
```

---

## Webhooks

Webhooks permitem que sistemas externos notifiquem seu app Germanio de eventos.

### Recebendo Webhooks

O Germanio aceita requisicoes POST em qualquer endpoint da API. Para receber um webhook de um sistema externo (ex: gateway de pagamento), configure o sistema externo para enviar POST para:

```
POST http://seuapp.com/api/pedido
```

Com payload JSON:

```json
{
  "cliente": "Joao Silva",
  "produto": "Notebook",
  "valor": 2999.99,
  "status": "pago"
}
```

O Germanio ira criar automaticamente o registro no banco de dados e disparar os eventos configurados (WhatsApp, Email, etc.).

### Exemplo: Webhook de Pagamento

```
sistema loja

dados

  pagamento
    cliente: texto obrigatorio
    valor: dinheiro obrigatorio
    status: status
    telefone: telefone
    email_cliente: email

integracoes

  whatsapp

    quando criar pagamento
      enviar mensagem para telefone
        texto "Pagamento de R${valor} confirmado! Obrigado, {cliente}."

  email

    servidor: "smtp.gmail.com"
    porta: "587"
    usuario: "financeiro@loja.com"
    senha: "app-password"

    quando criar pagamento
      enviar email para email_cliente
        assunto "Pagamento Confirmado"
        texto "Ola {cliente}! Recebemos seu pagamento de R${valor}. Obrigado!"
```

Ao receber o POST do gateway de pagamento em `/api/pagamento`, o Germanio:
1. Salva o registro no banco
2. Envia WhatsApp para o cliente
3. Envia email de confirmacao

---

## Pagamentos

> **Roadmap - Em desenvolvimento**

O suporte nativo a pagamentos esta planejado para versoes futuras. O bloco `pagamento` ja e reconhecido pelo lexer/parser mas ainda nao esta implementado no runtime.

### Syntax Prevista

```
integracoes

  pagamento
    provedor: "stripe"
    chave: "sk_live_..."

    quando criar pedido
      cobrar valor de cliente.email
        descricao "Pedido #{id} - {produto}"
        moeda "BRL"
```

### Provedores Planejados

- Stripe
- Mercado Pago
- PagSeguro
- Gerencianet / Efi

### Alternativa Atual

Enquanto o suporte nativo nao esta disponivel, use webhooks do seu gateway de pagamento apontando para a API REST do Germanio. O gateway chama seu endpoint e o Germanio persiste o registro e dispara notificacoes.

---

## Combinando Integracoes

E possivel ter WhatsApp, Email e Cron no mesmo arquivo:

```
sistema loja-completa

dados

  pedido
    cliente: texto obrigatorio
    produto: texto obrigatorio
    valor: dinheiro obrigatorio
    status: status
    telefone: telefone obrigatorio
    email_cliente: email obrigatorio

telas

  tela pedidos
    titulo "Pedidos"
    lista pedido
      mostrar cliente
      mostrar produto
      mostrar valor
      mostrar status
    botao azul
      texto "Novo Pedido"

eventos

  quando clicar "Novo Pedido"
    criar pedido

integracoes

  whatsapp

    quando criar pedido
      enviar mensagem para telefone
        texto "Ola {cliente}! Pedido de {produto} recebido. Aguarde!"

    quando atualizar pedido
      enviar mensagem para telefone
        texto "Seu pedido esta {status}. Valor: R${valor}"

  email

    servidor: "smtp.gmail.com"
    porta: "587"
    usuario: "sistema@loja.com"
    senha: "app-password"

    quando criar pedido
      enviar email para email_cliente
        assunto "Pedido Confirmado - #{id}"
        texto "Ola {cliente}! Seu pedido de {produto} foi confirmado. Valor: R${valor}."

  cron

    cada 30 minutos
      chamar api "http://localhost:8080/api/pedido?status=pendente"

    cada 1 dia
      chamar api "https://meuapp.com/relatorio-diario"
```

### Ordem de Execucao

Quando um registro e criado/atualizado, os notificadores sao disparados na seguinte ordem:
1. WhatsApp (se configurado)
2. Email (se configurado)
3. Webhooks de saida (futuro)

Os cron jobs sao independentes e rodam em goroutines separadas.

---

## Variaveis de Ambiente para Segredos

Nunca comite senhas no codigo-fonte. Use variaveis de ambiente e o arquivo `.env`:

```bash
# .env
EMAIL_SENHA=sua-senha-real
JWT_SECRET=chave-secreta-longa
```

No arquivo `.ge`, use os valores diretamente por enquanto — suporte a `${ENV_VAR}` esta no roadmap. A alternativa atual e configurar via variaveis de ambiente do sistema operacional antes de rodar o Germanio.

---

## Solucao de Problemas

### WhatsApp nao conecta

```
[whatsapp] timeout ao esperar QR code
```

- Certifique-se de escanear o QR no celular dentro de 60 segundos
- Verifique se o numero tem WhatsApp ativo
- Delete `whatsapp.db` e reconecte

### Email: "authentication failed"

- Gmail requer Senha de App, nao a senha normal
- Verifique se a verificacao em duas etapas esta ativa na conta Google
- Teste a conexao: `telnet smtp.gmail.com 587`

### Cron nao executa

- Verifique os logs: o intervalo deve ser `<numero> <unidade>`
- `cada 5 minuto` nao funciona — use `cada 5 minutos`
- URLs de API devem comecar com `http://` ou `https://`

### Numero de telefone invalido

O WhatsApp rejeita numeros com menos de 10 digitos apos limpeza. Garanta que o campo `telefone` do modelo tenha o numero completo com DDD.
