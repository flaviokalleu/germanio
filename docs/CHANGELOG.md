# CHANGELOG - Germanio

> **Escopo desta referência:** Histórico de versões; não substitui a especificação normativa [INTENCAO.md](INTENCAO.md). Esta revisão documental não altera compiler/runtime nem comprova todas as funcionalidades históricas.

Historico completo de versoes e mudancas do Germanio.

O formato segue [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## Revisão documental — 2026-09-28

- `INTENCAO.md` elevado a especificação normativa, preservando o caminho existente.
- Adicionados: autoridade documental, três mundos arquiteturais, testes do leigo e do
  profissional, evolução de capabilities, refatoração retroativa e checklist de design.
- Ampliados: quatro níveis, inferência explícita, estados, trabalho remoto, `ge explain`,
  `ge check`, erros educativos e testes normativos.
- Corrigidos: hooks manuais para último owner e bloqueio de usuário; exigência de hook para
  toda ação própria; roadmap que confundia a fundação com ausência de full-stack.
- Preservados: relações, login, papéis e herança, páginas, configuração, primitivas,
  adaptadores, transações, repositórios e trabalho remoto genérico.
- Pendências arquiteturais identificadas na norma: proveniência completa de inferências,
  diagnósticos, transições fora das convenções e cobertura de invariantes indiretos.
- Criados índice de documentação e `AGENTS.md`; `CLAUDE.md` aponta a ordem obrigatória.
- Removidos dois documentos legados redundantes; referências corrigidas. Guias técnicos
  mantidos com escopo explícito, sem promover exemplos antigos a norma da intenção.
- Validação: links locais dos documentos alterados, cercas de código e `git diff --check`.
  Não executados testes Go: toolchain indisponível no ambiente desta revisão.
  Nenhum compiler, runtime ou arquivo `.ge` foi modificado.

---

## v0.5.0 (2026-04-10)

### Novidades
- **20 Idiomas**: Portugues, Ingles, Espanhol, Frances, Alemao, Italiano, Chines, Japones, Coreano, Arabe, Hindi, Bengali, Russo, Indonesio, Turco, Vietnamita, Polones, Holandes, Tailandes, Suaili
- **Tema Presets**: `tema moderno`, `tema simples`, `tema elegante`, `tema corporativo`, `tema claro`
- **Cores por nome**: `cor primaria azul` (sem hex)
- **4 estilos visuais**: glassmorphism, flat, neumorphism, minimal
- **Rotas customizadas**: bloco `rotas` para endpoints de API personalizados
- **Paginas customizadas**: bloco `paginas` para paginas HTML personalizadas
- **Sidebar customizavel**: bloco `sidebar` com icones e links
- **Email HTML**: deteccao automatica de HTML no corpo do email
- **Enum com valores**: `status: enum(ativo, inativo, pendente)`
- **FK Dropdowns**: campos com `pertence_a` renderizam select populado automaticamente
- **textarea**: `texto_longo` renderiza como textarea
- **Charts**: graficos com Chart.js (barra, pizza, doughnut)
- **Relacionamentos**: `tem_muitos` e `muitos_para_muitos` com join tables automaticas
- **Auto-migration**: colunas novas adicionadas automaticamente
- **`germanio build`**: compila .ge em executavel standalone distribuivel
- **Async**: `paralelo()`, `esperar()`, `timeout()`, `chamar_async()`, `consultar_paralelo()`
- **15 novos built-in functions**: substituir, cortar, comeca_com, termina_com, substring, adicionar, remover, reverter, chaves, valores, json, formato_data, potencia, raiz, chamar/http
- **Array indexing**: `arr[0]`, `obj.campo[0]`

### Seguranca
- Auth bypass corrigido
- Protecao SSRF no proxy
- `/api/_eval` requer admin
- XSS escaping em valores do usuario
- Prevencao de path traversal em imports
- Whitelist de extensoes em upload
- Limites de body (1MB POST/PUT)
- JWT secret via env variable
- Protecao contra CSV injection

### Performance
- Connection pooling no banco (25 max, 5 idle)
- Timeouts no servidor HTTP (15s read, 30s write)
- Limpeza automatica do rate limiter
- Cache de HTML

### Testes
- 59 testes (lexer, parser, AST, interpreter)
- Todos passando

---

## [0.4.0] - 2025 Q1

### Visao Geral

A versao 0.4.0 e a mais completa ate agora, adicionando autenticacao JWT com bcrypt, sistema de pagina com paginacao, uploads de arquivos, notificacoes por email, agendamento de tarefas (cron), graficos no dashboard, containerizacao Docker e suporte ao VS Code com extensao dedicada.

### Adicionado

#### Autenticacao e Seguranca
- Bloco `autenticacao` / `auth` para configurar JWT + bcrypt
- Endpoints automaticos: `POST /auth/login`, `POST /auth/register`, `GET /auth/me`
- Sistema de roles: `roles: admin, gerente, usuario`
- Controle de acesso por tela: `requer <role>` / `requires <role>`
- Telas publicas sem login: `publico` / `public`
- Hashing bcrypt automatico para campos do tipo `senha` / `password`
- JWT com expiracao de 24 horas
- Headers de seguranca: `X-Content-Type-Options`, `X-Frame-Options`, `X-XSS-Protection`

#### Email SMTP
- Bloco `email` dentro de `integracoes`
- Suporte a qualquer servidor SMTP (Gmail, Outlook, SendGrid, Mailgun, Amazon SES)
- Gatilhos: `quando criar`, `quando atualizar`, `quando deletar`
- Templates com variaveis `{campo}` no assunto e no corpo
- Destino via campo do modelo: `enviar email para cliente.email`
- Palavra-chave `assunto` / `subject` para linha de assunto

#### Cron Jobs
- Bloco `cron` dentro de `integracoes`
- Agendamento com `cada <N> <unidade>` / `every <N> <unit>`
- Unidades: `segundos`, `minutos`, `horas`, `dias` (PT e EN)
- Acao `chamar api <url>` para chamadas HTTP GET periodicas
- Logs de execucao no stdout: `[cron] Chamando: <url>`
- Acoes genericas registradas em log para extensao futura

#### Upload de Arquivos
- Tipo de campo `upload` para arquivos enviados via formulario
- Armazenamento local automatico
- Endpoint gerado: `POST /api/<modelo>/upload`

#### Graficos e Dashboard
- Componente `grafico` / `chart` nas telas
- Componente `dashboard` para visao geral
- Componente `tabela` / `table` para dados tabulares
- Integracao automatica com dados dos modelos

#### Paginacao
- API REST com paginacao automatica: `?page=1&limit=20`
- Resposta com metadados: `total`, `page`, `limit`, `pages`
- Frontend com navegacao por paginas

#### Docker e DevOps
- `Dockerfile` otimizado para build multi-stage Go
- `docker-compose.yml` com servicos para app + banco de dados
- Suporte a variaveis de ambiente para configuracao de producao
- Documentacao de deploy com Nginx e Let's Encrypt

#### VS Code Extension
- Extensao `vscode-germanio/` com realce de sintaxe para `.ge`
- Autocomplete para palavras-chave PT e EN
- Snippets para blocos comuns
- Icone de arquivo `.ge` personalizado

### Melhorado

- Parser mais robusto para palavras-chave usadas como nomes de campo
- Mensagens de erro mais descritivas com numero de linha
- Reconexao automatica do WhatsApp em caso de queda
- Performance do WebSocket com broadcast seletivo

---

## [0.3.0] - 2024 Q3

### Visao Geral

A versao 0.3.0 trouxe o suporte bilinguie completo (Portugues e Ingles), suporte a MySQL e PostgreSQL, comunicacao em tempo real via WebSocket, e as primeiras integracoes nativas.

### Adicionado

#### Suporte Bilinguie (PT/EN)
- Todas as palavras-chave disponíveis em Portugues E Ingles simultaneamente
- Mistura de idiomas no mesmo arquivo (ex: `system` + `dados`)
- Lexer unificado: `sistema` = `system`, `dados` = `models`, etc.
- Equivalencias completas para todos os blocos, tipos, modificadores e eventos
- Exemplos em `exemplos/english/` e `exemplos/mixed/`

#### MySQL e PostgreSQL
- Bloco `banco` / `database` / `db` para configurar banco externo
- Driver MySQL com `driver: "mysql"` ou `banco mysql`
- Driver PostgreSQL com `driver: "postgres"` ou `banco postgres`
- Configuracao de host, porta, nome, usuario e senha
- Migrations automaticas: `ALTER TABLE` para novos campos
- SQLite permanece como padrao sem configuracao

#### WebSocket / Tempo Real
- Endpoint `/ws` para conexao WebSocket
- Broadcast automatico ao criar, atualizar ou deletar registros
- Frontend atualiza listas em tempo real sem refresh de pagina
- Multiplos clientes sincronizados simultaneamente
- Reconexao automatica em caso de queda

#### Integracoes Iniciais (WhatsApp)
- Bloco `integracoes` / `integrations`
- Sub-bloco `whatsapp` com autenticacao via QR Code
- Biblioteca whatsmeow para protocolo WhatsApp nativo
- Gatilhos `quando criar`, `quando atualizar`, `quando deletar`
- Templates de mensagem com variaveis `{campo}`
- Persistencia de sessao em arquivo SQLite local (`whatsapp.db`)
- Normalizacao automatica de numeros de telefone (padrao Brasil +55)

#### Sistema de Imports
- Keyword `importar` / `import` para incluir arquivos externos
- Sintaxe `importar "arquivo.ge"` para importar tudo
- Sintaxe `importar dados de "arquivo.ge"` para importar bloco especifico
- Merge de programas: modelos, telas, eventos e regras combinados
- Exemplo modular em `exemplos/restaurante-modular/`

#### Relacionamentos entre Modelos
- Keyword `pertence_a` / `belongs_to` para FK
- Keyword `tem_muitos` / `has_many` para relacionamento inverso
- Keyword `muitos_para_muitos` / `many_to_many`
- Geracao automatica de coluna de FK no banco
- Validacao referencial nos endpoints

#### Tipos de Dados Novos
- `texto_longo` / `long_text` — textarea no formulario
- `enum` — campo com lista fixa de opcoes
- `link` — URL com validacao de formato
- `soft_delete` — modificador de modelo para exclusao suave

### Melhorado

- Lexer reescrito para suporte a unicode (acentos em strings)
- API REST com suporte a filtros via query params
- Frontend com sidebar responsiva
- Tema visual com suporte a modo escuro (`escuro` / `dark`)

---

## [0.2.0] - 2024 Q1

### Visao Geral

A versao 0.2.0 foi uma reescrita completa do Germanio: de gerador de codigo estatico para um **interpretador com runtime proprio**. O arquivo `.ge` agora e interpretado diretamente pelo binario `germanio`, sem necessidade de gerar codigo intermediario.

### Adicionado

#### Runtime Proprio
- Interpretador Go que le `.ge` e serve a aplicacao diretamente
- Servidor HTTP embutido baseado em `net/http`
- Banco de dados SQLite embutido via `go-sqlite3`
- Sem necessidade de gerar codigo — o `.ge` e o codigo-fonte final
- Comando `germanio run arquivo.ge` para iniciar o servidor

#### API REST Completa
- Geracao automatica de endpoints CRUD para cada modelo
- `GET /api/<modelo>` — lista todos
- `GET /api/<modelo>/:id` — busca um
- `POST /api/<modelo>` — cria
- `PUT /api/<modelo>/:id` — atualiza
- `DELETE /api/<modelo>/:id` — remove
- Respostas em JSON padronizadas

#### Frontend Gerado Dinamicamente
- HTML/CSS/JS gerado em tempo de execucao pelo runtime
- Sidebar com navegacao entre telas
- Listas com colunas configuradas pelo bloco `mostrar` / `show`
- Formularios automaticos baseados nos campos do modelo
- Botoes com eventos vinculados

#### Temas Visuais
- Bloco `tema` / `theme` para customizacao de cores
- Suporte a cores hex: `cor primaria "#6366f1"`
- Sidebar customizavel: `cor sidebar "#1e1b4b"`
- Cor de destaque: `cor destaque "#f59e0b"`

#### Logica e Validacoes
- Bloco `logica` / `logic` para regras declarativas
- `validar <campo> <condicao>` para validacoes de campo
- `se <campo> <operador> <valor>` para logica condicional
- Regras de cor condicional: `mudar cor <cor>`
- Operadores: `igual`, `maior`, `menor` / `equals`, `greater`, `less`

#### Modificadores de Campo
- `obrigatorio` / `required` — campo nao pode ser vazio
- `unico` / `unique` — campo deve ser unico no banco
- `padrao` / `default` — valor padrao ao criar
- `indice` / `index` — cria indice no banco para performance
- `pertence_a` / `belongs_to` — chave estrangeira

### Mudancas Incompativeis com 0.1.x

- Arquivo `.ge` nao gera mais codigo em outra linguagem
- Comando mudou de `germanio generate` para `germanio run`
- Estrutura de blocos ligeiramente diferente (mais consistente)
- Tipos de dados renomeados para ser mais intuitivos

---

## [0.1.0] - 2023 Q3

### Visao Geral

A versao inicial do Germanio como **gerador de codigo**. O arquivo `.ge` era processado para gerar codigo em Python (Django) ou JavaScript (Express + React). Focado em prototipagem rapida.

### Adicionado

#### Linguagem Base
- Sintaxe declarativa inicial com blocos `sistema`, `dados`, `telas`, `eventos`
- Apenas palavras-chave em Portugues
- Tipos basicos: `texto`, `numero`, `email`, `data`, `booleano`
- Modificador `obrigatorio` para campos obrigatorios

#### Gerador de Codigo
- Geracao de modelos Django (Python) a partir do bloco `dados`
- Geracao de views e URLs Django para CRUD
- Geracao de templates HTML simples
- Comando `germanio generate --target django arquivo.ge`

#### Frontend Basico
- Geracao de componentes React simples
- Listagem e formularios para cada modelo
- CSS basico sem personalizacao de tema

#### Banco de Dados
- Suporte apenas a SQLite via Django ORM
- Migrations Django geradas automaticamente

#### Ferramentas
- CLI inicial: `germanio generate`, `germanio validate`
- Validacao de sintaxe sem executar

### Limitacoes Conhecidas

- Sem suporte a Ingles (apenas PT)
- Sem servidor embutido — requeria instalacao do Django
- Sem WebSocket ou tempo real
- Sem integracoes (WhatsApp, Email, etc.)
- Sem autenticacao nativa
- Frontend basico sem personalizacao

---

## Roadmap

### [0.6.0] - Visao Futura

- [ ] Editor visual online (Germanio Studio)
- [ ] Marketplace de templates `.ge`
- [ ] CLI interativa: `germanio new` com perguntas guiadas
- [ ] Plugins em Go para estender o runtime
- [ ] Suporte a GraphQL alem de REST
- [ ] Modo offline / PWA para o frontend gerado

---

## Convencoes de Versao

O Germanio segue [Semantic Versioning](https://semver.org/):

- **MAJOR** (`1.x.x`): mudancas incompativeis com versoes anteriores
- **MINOR** (`x.1.x`): novas funcionalidades sem quebrar compatibilidade
- **PATCH** (`x.x.1`): correcoes de bugs

Enquanto o projeto estiver em `0.x.x`, mudancas incompativeis podem ocorrer entre versoes minor. A partir de `1.0.0`, a compatibilidade sera garantida.

---

## Como Contribuir

Veja as issues abertas no GitHub para:
- Bugs reportados pela comunidade
- Features planejadas para proximas versoes
- Documentacao que precisa de melhorias

Pull requests sao bem-vindos. O projeto usa Go modules e o codigo esta organizado em:

```
compiler/    <- lexer, parser, AST
runtime/     <- servidor HTTP, banco, integracoes
cli/         <- comandos da linha de comando
exemplos/    <- exemplos de .ge
docs/        <- documentacao
vscode-germanio/ <- extensao VS Code
```
