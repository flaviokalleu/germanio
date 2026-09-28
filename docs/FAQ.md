# FAQ - Perguntas Frequentes sobre Germanio

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


Respostas para as duvidas mais comuns sobre o Germanio.

---

## Indice

- [Primeiros Passos](#primeiros-passos)
- [Sintaxe e Linguagem](#sintaxe-e-linguagem)
- [Banco de Dados](#banco-de-dados)
- [Frontend e Interface](#frontend-e-interface)
- [Autenticacao](#autenticacao)
- [Integracoes](#integracoes)
- [Deploy e Producao](#deploy-e-producao)
- [Performance](#performance)
- [Erros Comuns](#erros-comuns)
- [Comparacao com Outras Ferramentas](#comparacao-com-outras-ferramentas)

---

## Primeiros Passos

### Q: O que e o Germanio?

O Germanio e uma linguagem de programacao declarativa e bilinguie (Portugues/Ingles) que gera aplicacoes full-stack completas a partir de arquivos `.ge`. Voce descreve **o que** o sistema deve fazer, e o Germanio cuida de todo o codigo de backend, frontend, banco de dados e APIs REST.

### Q: Preciso saber programar para usar o Germanio?

Nao e necessario conhecimento de programacao tradicional. Se voce consegue descrever o que seu sistema precisa fazer em frases simples, consegue usar o Germanio. Para integracoes avancadas (WebSocket, APIs externas), um conhecimento basico ajuda.

### Q: Quais sao os requisitos para instalar?

- Go 1.21 ou superior
- Sistema operacional: Linux, macOS ou Windows
- Para WhatsApp: numero de celular com WhatsApp ativo

```bash
# Verifique a versao do Go
go version
# go version go1.21.0 linux/amd64
```

### Q: Como instalo o Germanio?

```bash
git clone https://github.com/flaviokalleu/germanio.git
cd germanio
go build -o germanio .
./germanio version
```

### Q: Como crio meu primeiro app?

```bash
# Crie um arquivo
nano meuapp.ge
```

Conteudo minimo:

```
sistema meuapp

dados

  tarefa
    nome: texto obrigatorio
    status: status

telas

  tela tarefas
    titulo "Minhas Tarefas"
    lista tarefa
      mostrar nome
      mostrar status
    botao verde
      texto "Nova Tarefa"

eventos

  quando clicar "Nova Tarefa"
    criar tarefa
```

```bash
./germanio run meuapp.ge
# Acesse http://localhost:8080
```

### Q: Posso usar o Germanio em Windows?

Sim. O Germanio compila e roda normalmente no Windows. Use `go build -o germanio.exe .` para gerar o executavel.

### Q: Existe um instalador ou pacote?

Ha um instalador disponivel em `installer/`. Consulte o `README.md` principal para instrucoes de instalacao especificas por plataforma.

### Q: Existe extensao para editor de texto?

Sim. Ha uma extensao para VS Code em `vscode-germanio/` no repositorio. Ela fornece realce de sintaxe e autocomplete para arquivos `.ge`.

---

## Sintaxe e Linguagem

### Q: Posso usar Portugues e Ingles no mesmo arquivo?

Sim! O Germanio e totalmente bilinguie. Voce pode misturar palavras-chave em qualquer idioma:

```
system pizzaria

dados

  pizza
    nome: text required
    preco: money
    tamanho: texto

screens

  tela cardapio
    title "Cardápio"
    list pizza
      show nome
      mostrar preco
```

### Q: As palavras-chave sao case-sensitive?

Nao. `Sistema`, `SISTEMA` e `sistema` sao equivalentes. O lexer converte tudo para minusculo antes de processar.

### Q: Como funciona a indentacao?

O Germanio usa indentacao para estruturar o codigo. Use **2 ou 4 espacos** (ou tabs). O importante e ser consistente dentro de cada bloco. Tabs e espacos podem ser misturados — um tab equivale a 2 espacos no contador interno.

### Q: Posso usar comentarios?

Sim. Dois estilos:

```
// Este e um comentario de linha (estilo C)
# Este tambem e um comentario de linha (estilo Python/Shell)
```

### Q: Quais sao todos os tipos de campo disponíveis?

| Tipo PT       | Tipo EN       | Armazenamento SQL | Descricao                          |
|---------------|---------------|-------------------|------------------------------------|
| `texto`       | `text`        | TEXT              | Texto curto                        |
| `texto_longo` | `long_text`   | TEXT              | Texto longo (textarea)             |
| `numero`      | `number`      | REAL              | Numero inteiro ou decimal          |
| `dinheiro`    | `money`       | REAL              | Valor monetario                    |
| `email`       | `email`       | TEXT              | Email (validado)                   |
| `telefone`    | `phone`       | TEXT              | Telefone (formatado)               |
| `senha`       | `password`    | TEXT              | Senha (bcrypt automatico)          |
| `data`        | `date`        | DATETIME          | Data e hora                        |
| `booleano`    | `boolean`     | INTEGER           | Verdadeiro/Falso                   |
| `status`      | `status`      | TEXT              | Campo de status com cores          |
| `imagem`      | `image`       | TEXT              | URL da imagem                      |
| `upload`      | `upload`      | TEXT              | Upload de arquivo                  |
| `arquivo`     | `file`        | TEXT              | Referencia a arquivo               |
| `link`        | `link`        | TEXT              | URL                                |
| `enum`        | `enum`        | TEXT              | Valor de lista fixa                |

### Q: Como defino um campo obrigatorio?

```
nome: texto obrigatorio     // Portugues
name: text required         // Ingles
```

### Q: Como defino um campo unico (sem duplicatas)?

```
email: email obrigatorio unico    // Portugues
email: email required unique      // Ingles
```

### Q: Como crio relacionamentos entre modelos?

Use `pertence_a` (ou `belongs_to`):

```
dados

  categoria
    nome: texto obrigatorio

  produto
    nome: texto obrigatorio
    preco: dinheiro
    categoria_id: numero pertence_a categoria
```

Para relacionamento um-para-muitos, use `tem_muitos` (ou `has_many`):

```
dados

  autor
    nome: texto
    livros: tem_muitos livro

  livro
    titulo: texto
    autor_id: numero pertence_a autor
```

### Q: O que e o soft_delete?

Soft delete e uma estrategia onde registros deletados nao sao removidos do banco — eles recebem um campo `deleted_at`. Isso permite recuperacao de dados e auditoria.

```
dados

  pedido soft_delete
    cliente: texto
    valor: dinheiro
```

### Q: Como importo um arquivo .ge dentro de outro?

```
// Importa tudo do arquivo
importar "dados.ge"
importar "telas.ge"

// Ou em ingles
import "dados.ge"
from "telas.ge"
```

O arquivo importado deve conter blocos validos (`dados`, `telas`, etc.). Ver o exemplo `restaurante-modular` para uso completo.

### Q: Posso ter multiplos modelos em um arquivo .ge?

Sim. Todos os modelos ficam dentro do mesmo bloco `dados`:

```
dados

  produto
    nome: texto
    preco: dinheiro

  categoria
    nome: texto

  fornecedor
    nome: texto
    email: email
```

### Q: Como defino valores padrao para campos?

Use o modificador `padrao` (ou `default`):

```
dados

  produto
    nome: texto obrigatorio
    status: texto padrao "ativo"
    quantidade: numero padrao 0
```

---

## Banco de Dados

### Q: Qual banco de dados o Germanio usa por padrao?

SQLite. Nao requer configuracao — o banco e criado automaticamente na pasta do projeto com o nome `<sistema>.db`.

### Q: Como uso MySQL ou PostgreSQL?

Adicione o bloco `banco` (ou `database`/`db`) no arquivo `.ge`:

```
// MySQL
banco
  driver: "mysql"
  host: "localhost"
  port: "3306"
  nome: "minha_loja"
  usuario: "root"
  senha: "senha123"

// PostgreSQL
banco postgres
  host: "localhost"
  port: "5432"
  nome: "minha_loja"
  usuario: "postgres"
  senha: "senha123"
```

### Q: As tabelas sao criadas automaticamente?

Sim. O Germanio roda `CREATE TABLE IF NOT EXISTS` ao iniciar. Se um modelo ja existir, as colunas novas sao adicionadas via `ALTER TABLE`. Colunas removidas NAO sao deletadas automaticamente (seguranca dos dados).

### Q: Posso criar indices para performance?

```
dados

  produto
    nome: texto indice     // cria index em nome
    email: email unico     // unico tambem cria index
    preco: dinheiro
```

### Q: Como faço backup do banco SQLite?

```bash
# Backup simples
cp meuapp.db backup/meuapp.db.$(date +%Y%m%d)

# Backup online (enquanto o app roda)
sqlite3 meuapp.db ".backup backup.db"
```

### Q: O Germanio suporta migracoes de banco?

Migracoes automaticas basicas estao incluídas — novos campos sao adicionados ao ALTER TABLE. Para migracoes complexas (renomear colunas, mudar tipos), e necessario fazer manualmente no banco.

### Q: Posso usar o banco de dados existente com o Germanio?

Sim, para MySQL e PostgreSQL. Aponte para o banco existente no bloco `banco`. O Germanio criara apenas as tabelas que nao existem. Tabelas existentes com campos correspondentes serao usadas normalmente.

---

## Frontend e Interface

### Q: Que tecnologias o frontend usa?

O frontend gerado e em HTML puro com CSS (Tailwind inline) e JavaScript vanilla. Nao usa React, Vue ou Angular — roda em qualquer navegador sem build step.

### Q: Posso personalizar o visual?

Use o bloco `tema`:

```
tema
  cor primaria "#3b82f6"      // azul
  cor secundaria "#8b5cf6"    // roxo
  cor destaque "#f59e0b"      // amarelo
  cor sidebar "#1e1b4b"       // sidebar escura
  escuro                       // modo escuro
```

### Q: Quais cores posso usar nos botoes?

Qualquer cor CSS ou as palavras `azul`/`blue`, `verde`/`green`, `vermelho`/`red`:

```
botao azul
  texto "Confirmar"

botao vermelho
  texto "Cancelar"

botao "#6366f1"
  texto "Personalizado"
```

### Q: O frontend e responsivo (mobile)?

Sim. O layout gerado usa CSS flexbox/grid e e adaptado para diferentes tamanhos de tela.

### Q: O Germanio suporta tempo real (WebSocket)?

Sim. O frontend se conecta automaticamente via WebSocket ao servidor. Qualquer alteracao no banco de dados e transmitida em tempo real para todos os clientes conectados — as listas atualizam automaticamente sem refresh.

### Q: Posso adicionar JavaScript customizado?

Nao diretamente no `.ge`. Para logica de frontend customizada, adicione via API REST — o Germanio expoe toda a logica via endpoints que qualquer cliente pode consumir.

### Q: O que e o tipo `status` e como ele aparece no frontend?

O tipo `status` e exibido com badges coloridas no frontend. O Germanio detecta valores comuns e aplica cores:

| Valor       | Cor exibida  |
|-------------|--------------|
| `ativo`     | Verde        |
| `inativo`   | Cinza        |
| `pendente`  | Amarelo      |
| `pago`      | Verde        |
| `cancelado` | Vermelho     |
| `entregue`  | Azul         |

### Q: Como adiciono busca em uma tela?

```
telas

  tela produtos
    titulo "Produtos"
    busca produtos
    lista produto
      mostrar nome
      mostrar preco
```

O componente `busca` adiciona um campo de pesquisa que filtra a lista em tempo real.

### Q: Posso ter um dashboard com graficos?

```
telas

  tela dashboard
    titulo "Dashboard"
    dashboard vendas
    grafico vendas_mensais
    tabela pedidos_recentes
```

---

## Autenticacao

### Q: Como habilito autenticacao no meu app?

Adicione o bloco `autenticacao` e um modelo de usuario:

```
autenticacao
  modelo: usuario
  campo_login: email
  campo_senha: senha
  roles: admin, usuario

dados

  usuario
    nome: texto obrigatorio
    email: email obrigatorio unico
    senha: senha obrigatorio
    role: texto
```

### Q: Como crio o primeiro usuario admin?

Apos iniciar o servidor, use a API:

```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "nome": "Admin",
    "email": "admin@meuapp.com",
    "senha": "senhaforte123",
    "role": "admin"
  }'
```

### Q: Os tokens JWT expiram?

Sim, por padrao em 24 horas. Apos a expiracao, o usuario precisa fazer login novamente.

### Q: Como protejo apenas algumas rotas?

Use `requer` em telas especificas:

```
telas

  tela catalogo
    titulo "Catalogo"
    publico              // qualquer um acessa

  tela admin
    titulo "Admin"
    requer admin         // so admins

  tela vendas
    titulo "Vendas"
    // sem publico = requer login (qualquer role)
```

### Q: Posso usar email ou username para login?

Por padrao o campo de login e `email`. Voce pode mudar:

```
autenticacao
  modelo: usuario
  campo_login: username    // usa campo "username"
  campo_senha: senha
```

### Q: Como funciona o logout?

O logout e feito no cliente removendo o token JWT do `localStorage`. O servidor nao mantém estado de sessao — e stateless por design.

---

## Integracoes

### Q: Preciso de conta especial para WhatsApp?

Nao. O Germanio usa o **WhatsApp comum** (nao Business API pago). Voce precisa apenas de um numero com WhatsApp ativo para escanear o QR Code. A sessao e mantida localmente.

### Q: O WhatsApp desconecta com frequencia?

Depende do seu celular. O WhatsApp pode desconectar dispositivos secundarios se:
- O celular ficar sem bateria por muito tempo
- A conta for acessada em muitos dispositivos (limite de ~4)
- A sessao expirar por inatividade

O Germanio reconecta automaticamente quando possivel.

### Q: Posso enviar imagens ou arquivos pelo WhatsApp?

A versao atual suporta apenas mensagens de texto. Suporte a midia (imagens, PDFs) esta no roadmap.

### Q: Como configuro o Gmail para enviar emails?

1. Ative a verificacao em duas etapas na conta Google
2. Va em Seguranca > Senhas de app
3. Crie uma senha para "Outro aplicativo"
4. Use essa senha (nao a senha do Gmail) no campo `senha` do Germanio

### Q: Os cron jobs param quando o servidor reinicia?

Sim. Os cron jobs rodam na memoria do processo. Ao reiniciar o servidor, eles recomeçam. Use um supervisor de processo (systemd, PM2, Docker restart policy) para garantir que o servidor fique sempre rodando.

### Q: Posso chamar APIs com autenticacao (Bearer token, etc.)?

O cliente HTTP atual suporta apenas GET sem headers customizados. Para APIs que exigem autenticacao, crie um endpoint proxy no seu proprio servidor que o Germanio pode chamar via cron.

### Q: Posso receber webhooks externos no Germanio?

Sim. A API REST do Germanio aceita qualquer POST. Configure o sistema externo (gateway de pagamento, etc.) para enviar para `POST /api/<modelo>` com os dados no formato JSON.

---

## Deploy e Producao

### Q: Como faco deploy do app Germanio?

```bash
# 1. Compile para producao
GOOS=linux GOARCH=amd64 go build -o germanio .

# 2. Copie para o servidor
scp germanio usuario@servidor:/opt/meuapp/
scp meuapp.ge usuario@servidor:/opt/meuapp/

# 3. Rode com systemd ou Docker
./germanio run meuapp.ge
```

### Q: O Germanio tem suporte a Docker?

Sim. Ha um `Dockerfile` e `docker-compose.yml` no repositorio:

```bash
docker-compose up -d
```

### Q: Como uso um dominio proprio com HTTPS?

Coloque um Nginx na frente:

```nginx
server {
    listen 443 ssl;
    server_name meuapp.com;
    ssl_certificate /etc/letsencrypt/live/meuapp.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/meuapp.com/privkey.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
    }
}
```

O header `Upgrade` e necessario para WebSocket funcionar.

### Q: Posso rodar multiplos apps na mesma maquina?

Sim. Cada app Germanio roda em uma porta diferente. Configure a porta via variavel de ambiente ou argumento:

```bash
./germanio run loja.ge --port 8080
./germanio run restaurante.ge --port 8081
```

### Q: Como configuro variaveis de ambiente em producao?

```bash
# No systemd unit file
[Service]
Environment=JWT_SECRET=minha-chave
Environment=DB_PASSWORD=senha-db
ExecStart=/opt/meuapp/germanio run meuapp.ge
```

### Q: Que recursos de servidor preciso?

Para um app pequeno a medio (ate ~1000 usuarios simultaneos):
- CPU: 1 vCore
- RAM: 512 MB
- Disco: 10 GB (SSD recomendado para SQLite)

Para cargas maiores, use MySQL/PostgreSQL e escale horizontalmente.

---

## Performance

### Q: O Germanio e rapido o suficiente para producao?

O runtime do Germanio e escrito em Go, uma das linguagens mais eficientes. Para a maioria dos casos de uso de CRUD (lojas, restaurantes, sistemas internos), o Germanio e mais que suficiente.

Benchmarks aproximados numa VPS basica (1 vCore, 1 GB RAM):
- GET listagem: ~5000 req/s
- POST criacao: ~2000 req/s
- WebSocket: ~10000 conexoes simultaneas

### Q: Devo usar SQLite em producao?

Para apps pequenos a medios com **um unico servidor** e menos de 10.000 registros, SQLite funciona muito bem em producao. E simples, confiavel e sem configuracao adicional.

Para:
- Multiplos servidores (horizontal scaling)
- +100.000 registros com muitas escritas simultaneas
- Backup em tempo real

...use MySQL ou PostgreSQL.

### Q: Como otimizo queries lentas?

1. Adicione indices nos campos usados em filtros/buscas:

```
dados

  produto
    nome: texto indice         // indexado
    email: email unico         // unico = indexado
    categoria_id: numero indice
```

2. Use paginacao na API:

```bash
GET /api/produto?page=1&limit=50
```

3. Para MySQL/PostgreSQL, monitore com `EXPLAIN QUERY PLAN`.

---

## Erros Comuns

### Q: "unexpected character" no arquivo .ge

Certifique-se de que:
- Strings estao entre aspas duplas: `"texto"` (nao aspas simples `'texto'`)
- Nao ha caracteres especiais fora de strings (use `#` para comentarios)
- O arquivo esta salvo em UTF-8

### Q: "tipo desconhecido" ao rodar

O campo tem um tipo invalido. Verifique os tipos disponiveis:
```
texto, numero, email, telefone, senha, data, booleano,
dinheiro, status, imagem, upload, arquivo, link, texto_longo, enum
```

### Q: O servidor nao inicia e retorna "address already in use"

A porta 8080 ja esta em uso. Mate o processo anterior:

```bash
# Linux/Mac
lsof -ti:8080 | xargs kill -9

# Windows
netstat -ano | findstr :8080
taskkill /PID <PID> /F
```

### Q: O WhatsApp mostra "timeout ao esperar QR code"

- O QR Code expira em ~60 segundos. Seja rapido ao escanear.
- Delete `whatsapp.db` e tente novamente.

### Q: Email retorna "authentication failed"

- Gmail: use Senha de App, nao a senha normal
- Verifique se a verificacao em duas etapas esta ativa
- Verifique `servidor`, `porta`, `usuario` e `senha`
- Teste: `telnet smtp.gmail.com 587`

### Q: A lista nao atualiza em tempo real

- Verifique se o WebSocket esta conectado (console do navegador: `ws://localhost:8080/ws`)
- Se estiver atrás de Nginx, configure o proxy para WebSocket (ver secao Deploy)

### Q: Campos com acentos nao funcionam como nome de modelo

Use nomes sem acentos nos modelos:

```
// Correto
producao     // sem acento
categorias   // sem acento

// Pode causar problemas
produção    // com acento no nome do modelo
```

Textos com acentos em strings sao perfeitamente suportados:

```
titulo "Produção"    // ok - dentro de string
```

---

## Comparacao com Outras Ferramentas

### Q: Qual a diferenca entre Germanio e Bubble/AppMaster?

| Caracteristica     | Germanio              | Bubble/AppMaster   |
|--------------------|--------------------|--------------------|
| Interface          | Codigo (texto)     | Visual (arrastar)  |
| Idioma             | PT/EN              | Ingles             |
| Preco              | Open source        | Pago               |
| Self-hosted        | Sim                | Sim/Nao            |
| WhatsApp nativo    | Sim                | Nao                |
| Curva de aprendizado | Baixa            | Media              |
| Versionamento Git  | Sim (texto)        | Dificil            |

### Q: Qual a diferenca entre Germanio e Django/Rails/Laravel?

| Caracteristica     | Germanio              | Django/Rails/Laravel |
|--------------------|--------------------|-----------------------|
| Conhecimento necessario | Minimo       | Alto (Python/Ruby/PHP)|
| Velocidade para MVP | Horas             | Dias/Semanas          |
| Customizacao       | Media              | Total                 |
| WhatsApp           | Nativo             | Requer biblioteca     |
| Frontend           | Gerado automaticamente | Manual ou outro framework |
| Bilinguie          | Sim                | Nao                   |

### Q: Qual a diferenca entre Germanio e Retool/Appsmith?

Retool e Appsmith sao ferramentas de construcao de paineis internos. O Germanio gera aplicacoes completas com frontend publico/privado e integracoes nativas (WhatsApp, Email). Alem disso, o Germanio e open source e self-hosted sem custo de licenca.

### Q: Quando devo usar Germanio vs um framework tradicional?

**Use Germanio quando:**
- Precisa de um MVP ou sistema interno rapidamente
- A equipe nao tem muitos desenvolvedores
- As funcionalidades sao principalmente CRUD
- Precisa de WhatsApp ou Email nativo
- Quer uma linguagem em Portugues

**Use um framework tradicional quando:**
- Precisa de logica de negocio muito complexa
- Requer integracao profunda com sistemas legados
- O time ja tem expertise no framework
- Precisa de customizacao total do frontend

### Q: O Germanio substitui um desenvolvedor?

Nao completamente. O Germanio acelera drasticamente o desenvolvimento de sistemas CRUD, integracoes e dashboards. Para logicas de negocio complexas, algoritmos customizados ou interfaces muito especificas, a intervencao de um desenvolvedor ainda e necessaria.

O Germanio e ideal para: sistemas internos, MVPs, automacoes de negocio, dashboards, e-commerce simples e apps de gerenciamento.

### Q: O Germanio tem comunidade ou suporte?

O projeto esta em desenvolvimento ativo. Contribuicoes, issues e pull requests sao bem-vindos no repositorio GitHub. Para suporte comercial, entre em contato com os mantenedores do projeto.
