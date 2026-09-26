# Documentação histórica do Germanio

> Referência do modo `.ge`. Os comandos, versões e instaladores abaixo são históricos.
> Para Germanio `.ge`, consulte o [README atual](../README.md) e a [SPEC](../SPEC.md).

<p align="center">
  <img src="../logo.png" alt="Germanio" width="400">
</p>

<p align="center">
  <strong>A linguagem declarativa que transforma ideias em aplicacoes full-stack.</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Version-0.5.0-6366f1?style=for-the-badge" alt="v0.5.0">
  <img src="https://img.shields.io/badge/Extension-.ge-6366f1?style=for-the-badge" alt="Germanio .ge">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go 1.21+">
  <img src="https://img.shields.io/badge/20%20Idiomas-PT%20EN%20ES%20FR%20DE%20...-f59e0b?style=for-the-badge" alt="20 Languages">
  <img src="https://img.shields.io/badge/Tests-59%20passing-brightgreen?style=for-the-badge" alt="59 Tests">
  <img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" alt="MIT License">
</p>

<p align="center">
  <a href="#o-que-e-o-germanio">O que e</a> ·
  <a href="#quick-start">Quick Start</a> ·
  <a href="#exemplo-completo">Exemplo</a> ·
  <a href="#20-idiomas">20 Idiomas</a> ·
  <a href="#temas-e-estilos">Temas</a> ·
  <a href="#cli">CLI</a> ·
  <a href="#seguranca">Seguranca</a>
</p>

---

## O que e o Germanio?

**Germanio** e uma linguagem de programacao declarativa escrita em Go que gera aplicacoes web full-stack a partir de arquivos `.ge`. Escreva o que voce quer — dados, telas, eventos, logica — e o Germanio gera backend, frontend, banco de dados, API REST, autenticacao e muito mais.

```
sistema loja

tema moderno
cor primaria azul

dados
  produto
    nome: texto obrigatorio
    preco: dinheiro
    status: enum(ativo, inativo, pendente)

telas
  tela produtos
    titulo "Produtos"
    lista produto
      mostrar nome, preco, status
    botao azul
      texto "Novo Produto"

eventos
  quando clicar "Novo Produto"
    criar produto
```

```bash
germanio run loja.ge
# App completa rodando em http://localhost:8080
```

---

## Quick Start

### Instalacao

```bash
# Build do fonte
git clone https://github.com/flaviokalleu/germanio.git
cd germanio
go build -o germanio .

# Ou via instalador (Linux/macOS)
curl -fsSL https://github.com/flaviokalleu/germanio/releases/latest/download/install.sh | sh
```

### Criar e rodar

```bash
# Criar novo projeto
germanio new meu-app

# Rodar
cd meu-app
germanio run inicio.ge

# Abrir no navegador
# http://localhost:8080
```

---

## Exemplo Completo

```
sistema restaurante

tema elegante
cor primaria vermelho
estilo glassmorphism

dados
  prato
    nome: texto obrigatorio
    descricao: texto_longo
    preco: dinheiro
    categoria: texto
    disponivel: booleano
    foto: imagem
    status: enum(disponivel, esgotado, em_preparo)

  pedido
    cliente: texto obrigatorio
    telefone: telefone
    prato: pertence_a prato
    quantidade: numero
    observacao: texto_longo
    status: enum(recebido, preparando, pronto, entregue)

telas
  tela cardapio
    titulo "Cardapio"
    lista prato
      mostrar nome, preco, status, foto
    botao vermelho
      texto "Novo Prato"

  tela pedidos
    titulo "Pedidos"
    lista pedido
      mostrar cliente, prato, quantidade, status

eventos
  quando clicar "Novo Prato"
    criar prato

  quando criar pedido
    notificar "Novo pedido recebido!"

logica
  funcao calcular_total(preco, quantidade)
    retornar preco * quantidade

autenticacao
  modelo: usuario
  campo_login: email
  campo_senha: senha
  roles: admin, garcom, cozinha

integracoes
  whatsapp
    quando criar pedido
      enviar mensagem para telefone
        texto "Pedido recebido! {prato} x{quantidade}"

  email
    servidor: "smtp.gmail.com"
    porta: "587"
    quando criar pedido
      enviar email para cliente.email
        assunto "Pedido confirmado"
        texto "Seu pedido foi confirmado!"

  cron
    cada 30 minutos
      executar verificar_estoque
```

---

## Features

### Linguagem e Runtime

| Feature | Descricao |
|---------|-----------|
| 20 idiomas | Keywords em PT, EN, ES, FR, DE, IT, ZH, JA, KO, AR, HI, BN, RU, ID, TR, VI, PL, NL, TH, SW |
| Mistura livre | Use keywords de qualquer idioma no mesmo arquivo |
| Variaveis | `definir x = 10` / `set x = 10` |
| Funcoes | `funcao soma(a, b)` / `function sum(a, b)` |
| Condicionais | `se/senao/senao se` / `if/else/else if` |
| Loops | `repetir N vezes`, `enquanto`, `para cada` |
| Controle de fluxo | `parar`, `continuar`, `retornar` |
| Tratamento de erros | `tentar/erro` (try/catch) |
| Enums | `status: enum(ativo, inativo, pendente)` |
| 30+ funcoes built-in | Texto, numero, async, e mais |

### Full-Stack Automatico

| Feature | Descricao |
|---------|-----------|
| Backend | Servidor HTTP embutido com API REST completa |
| Frontend | Dashboard moderno com dark mode e sidebar |
| Banco de dados | SQLite, MySQL, PostgreSQL |
| WebSocket | Tempo real entre multiplos usuarios |
| Upload | Arquivos com preview automatico |
| Paginacao | `?pagina=1&limite=10&busca=texto&ordenar=preco` |
| Export | CSV e JSON com um clique |
| Hot reload | Alteracoes refletidas automaticamente |

### Seguranca

| Feature | Descricao |
|---------|-----------|
| JWT Auth | Login, registro, tokens JWT com bcrypt |
| Roles | Controle de acesso por papel (admin, user, etc.) |
| Rate limiting | Protecao contra abuso de requisicoes |
| SSRF protection | Bloqueio de requisicoes a redes internas |
| XSS prevention | Sanitizacao automatica de inputs |
| SQL Injection | Queries parametrizadas |
| Security headers | X-Content-Type-Options, X-Frame-Options, CSP |

### Integracoes

| Feature | Descricao |
|---------|-----------|
| WhatsApp | Mensagens automaticas via whatsmeow |
| Email | SMTP com templates HTML |
| Cron | Agendamentos (`cada 5 minutos`, `cada 1 hora`) |
| HTTP Client | Chamar APIs externas |
| Webhooks | Receber notificacoes |

---

## 20 Idiomas

Germanio suporta keywords em **20 idiomas**. Voce pode misturar idiomas livremente no mesmo arquivo.

### Portugues

```
sistema loja

dados
  produto
    nome: texto obrigatorio
    preco: dinheiro
```

### English

```
system store

models
  product
    name: text required
    price: money
```

### Espanol

```
sistema tienda

datos
  producto
    nombre: texto obligatorio
    precio: dinero
```

### Francais

```
systeme magasin

donnees
  produit
    nom: texte obligatoire
    prix: argent
```

### Deutsch

```
system laden

daten
  produkt
    name: text erforderlich
    preis: geld
```

### Italiano

```
sistema negozio

dati
  prodotto
    nome: testo obbligatorio
    prezzo: denaro
```

**Idiomas suportados:** Portugues, English, Espanol, Francais, Deutsch, Italiano, Zhongwen, Nihongo, Hangugeo, Alarabiya, Hindi, Bangla, Russkiy, Bahasa Indonesia, Turkce, Tieng Viet, Polski, Nederlands, Thai, Kiswahili.

---

## Temas e Estilos

### Presets de Tema

```
tema moderno       # Design moderno com sombras e gradientes
tema simples       # Clean e minimalista
tema elegante      # Sofisticado com tipografia refinada
tema corporativo   # Profissional para empresas
tema claro         # Leve e luminoso
```

### Cores por Nome

Sem necessidade de codigos hex — use nomes de cores diretamente:

```
cor primaria azul
cor primaria vermelho
cor primaria verde
cor primaria roxo
cor primaria laranja
```

### Estilos Visuais

```
estilo glassmorphism    # Transparencia e blur (padrao)
estilo flat             # Cores solidas sem sombras
estilo neumorphism      # Efeito 3D suave
estilo minimal          # Minimo de elementos visuais
```

### Exemplo Combinado

```
sistema dashboard

tema corporativo
cor primaria azul
estilo glassmorphism
```

---

## Tipos de Dados

| Tipo PT | Tipo EN | Descricao | Input HTML |
|---------|---------|-----------|------------|
| `texto` | `text` | Texto simples | text |
| `texto_longo` | `long_text` | Texto longo | textarea |
| `numero` | `number` | Numero | number |
| `dinheiro` | `money` | Valor monetario | number |
| `email` | `email` | Com validacao | email |
| `telefone` | `phone` | Com validacao | tel |
| `data` | `date` | Data | date |
| `booleano` | `boolean` | Sim/Nao | checkbox |
| `senha` | `password` | Mascarado + bcrypt | password |
| `imagem` | `image` | Upload de imagem | file |
| `arquivo` | `file` | Upload de arquivo | file |
| `link` | `link` | URL | url |
| `status` | `status` | Badge colorido | text |
| `enum(...)` | `enum(...)` | Valores pre-definidos | select |

### Modificadores

| PT | EN | Efeito |
|----|----|--------|
| `obrigatorio` | `required` | NOT NULL + validacao |
| `unico` | `unique` | UNIQUE constraint |
| `indice` | `index` | CREATE INDEX |
| `soft_delete` | `soft_delete` | Exclusao logica |

---

## Relacionamentos

```
dados
  categoria
    nome: texto obrigatorio

  produto
    nome: texto obrigatorio
    preco: dinheiro
    categoria: pertence_a categoria       # FK: produto pertence a uma categoria

  tag
    nome: texto obrigatorio
    produtos: muitos_para_muitos produto   # Tabela intermediaria automatica

  pedido
    cliente: texto
    itens: tem_muitos item_pedido          # Um pedido tem muitos itens
```

| Tipo | Descricao |
|------|-----------|
| `pertence_a` / `belongs_to` | Foreign key (N:1) |
| `tem_muitos` / `has_many` | Relacao inversa (1:N) |
| `muitos_para_muitos` / `many_to_many` | Tabela intermediaria (N:N) |

---

## Rotas, Paginas e Sidebar Personalizadas

### Rotas Personalizadas

```
rotas_personalizadas
  rota "/dashboard"
    metodo "GET"
    resposta "Pagina principal"

  rota "/relatorio"
    metodo "GET"
    resposta "Relatorio mensal"
```

### Paginas Personalizadas

```
paginas_personalizadas
  pagina "sobre"
    titulo "Sobre Nos"
    conteudo "Texto da pagina sobre a empresa."
```

### Sidebar Personalizada

```
sidebar_personalizada
  item "Dashboard" icone "home" link "/dashboard"
  item "Relatorios" icone "chart" link "/relatorio"
  item "Configuracoes" icone "settings" link "/config"
```

---

## Funcoes Async

Germanio inclui funcoes async para operacoes paralelas e assincronas:

```
logica
  # Executar funcoes em paralelo
  paralelo
    chamar buscar_dados()
    chamar processar_relatorio()
    chamar enviar_notificacoes()

  # Esperar resultado
  definir resultado = esperar chamar_async(buscar_dados)

  # Timeout
  timeout 5000
    chamar operacao_lenta()

  # Consultas em paralelo
  definir dados = consultar_paralelo
    buscar usuarios
    buscar pedidos
    buscar produtos
```

### Lista de Funcoes Built-in (30+)

**Texto:** `texto`, `maiusculo`, `minusculo`, `contem`, `dividir`, `juntar`, `tamanho`

**Numero:** `numero`, `arredondar`, `abs`, `min`, `max`, `inteiro`, `aleatorio`

**Utilitarios:** `tipo`, `agora`, `mostrar`

**Async:** `paralelo`, `esperar`, `timeout`, `chamar_async`, `consultar_paralelo`

---

## Seguranca

```
autenticacao
  modelo: usuario
  campo_login: email
  campo_senha: senha
  roles: admin, vendedor, cliente
```

| Recurso | Implementacao |
|---------|---------------|
| Autenticacao | JWT tokens com expiracao |
| Senhas | Hash bcrypt automatico |
| Roles | Controle de acesso baseado em papel |
| Rate Limiting | Limite de requisicoes por IP |
| SSRF | Bloqueio de acesso a redes internas |
| XSS | Sanitizacao de inputs e outputs |
| SQL Injection | Queries parametrizadas |
| Headers | CSP, X-Frame-Options, X-Content-Type-Options |
| Validacao | `obrigatorio`, `unico`, `email`, `telefone` |

---

## CLI

| Comando | Descricao |
|---------|-----------|
| `germanio run arquivo.ge` | Executa o arquivo .ge |
| `germanio arquivo.ge` | Atalho para run |
| `germanio check arquivo.ge` | Verifica sintaxe sem executar |
| `germanio new nome` | Cria novo projeto com estrutura basica |
| `germanio init nome` | Cria projeto com .env e Docker |
| `germanio build arquivo.ge` | Gera executavel standalone |
| `germanio docker` | Gera Dockerfile e docker-compose |
| `germanio version` | Mostra a versao atual |
| `germanio help` | Mostra ajuda |

---

## API REST

Cada modelo gera automaticamente endpoints REST:

| Metodo | Rota | Acao |
|--------|------|------|
| `GET` | `/api/{modelo}` | Listar (com paginacao, filtros, busca) |
| `GET` | `/api/{modelo}/{id}` | Buscar por ID |
| `POST` | `/api/{modelo}` | Criar |
| `PUT` | `/api/{modelo}/{id}` | Atualizar |
| `DELETE` | `/api/{modelo}/{id}` | Deletar |

### Query Parameters

```
?pagina=1&limite=10          # Paginacao
?busca=texto                 # Busca full-text
?status=ativo                # Filtro por campo
?ordenar=preco&ordem=ASC     # Ordenacao
```

### Endpoints Especiais

| Rota | Descricao |
|------|-----------|
| `/api/login` | Login (JWT) |
| `/api/registro` | Registro |
| `/api/me` | Usuario atual |
| `/api/_stats` | Estatisticas |
| `/api/{modelo}/export/csv` | Exportar CSV |
| `/api/{modelo}/export/json` | Exportar JSON |
| `/upload` | Upload de arquivos |
| `/health` | Health check |
| `/ws` | WebSocket |

---

## Extensao VS Code

A extensao `vscode-germanio` oferece:

- **Syntax highlighting** para arquivos `.ge`
- **22 snippets** para produtividade rapida
- **Reconhecimento de keywords** nos 20 idiomas suportados

### Instalacao

```bash
# Na pasta do projeto
cd vscode-germanio
# Copie para extensoes do VS Code
cp -r . ~/.vscode/extensions/vscode-germanio
```

---

## Estrutura do Projeto

```
germanio/
├── main.go                  # Ponto de entrada
├── cli/                     # Comandos CLI
├── compiler/
│   ├── lexer/               # Tokenizador (20 idiomas)
│   ├── parser/              # Parser -> AST
│   └── ast/                 # Nodes da AST
├── runtime/
│   ├── interpreter/         # Motor de scripting
│   ├── auth/                # JWT + bcrypt + roles
│   ├── banco/               # SQLite, MySQL, PostgreSQL
│   ├── servidor/            # HTTP + WebSocket + Renderer
│   ├── whatsapp/            # Integracao WhatsApp
│   ├── email/               # SMTP com templates HTML
│   ├── cron/                # Agendamentos
│   └── httpclient/          # HTTP client
├── vscode-germanio/            # Extensao VS Code
├── examples/                # Programas de exemplo
├── demo/                    # Aplicacoes completas
├── docs/                    # Documentacao
└── installer/               # Instaladores Windows/Linux
```

---

## Contribuir

```bash
git clone https://github.com/flaviokalleu/germanio.git
cd germanio
go build -o germanio .
go test ./... -v
```

1. Fork o repositorio
2. Crie sua branch (`git checkout -b feature/nova-feature`)
3. Commit suas alteracoes (`git commit -m 'Add: nova feature'`)
4. Push para a branch (`git push origin feature/nova-feature`)
5. Abra um Pull Request

---

## Licenca

MIT License — veja [LICENSE](LICENSE)

---

<p align="center">
  <img src="logo.png" alt="Germanio" width="120">
  <br>
  <strong>Germanio v0.5.0</strong> — Descreva. Programe. Execute.
  <br>
  <sub>Feito por <a href="https://github.com/flaviokalleu">@flaviokalleu</a></sub>
</p>

