# Flang — Documento de Atualizacao e Roadmap Completo

**Versao atual: v0.5.1** | **Atualizado: Abril 2026**

---

## PARTE 1: O QUE JA FOI FEITO (116 features)

### Core da Linguagem
- [x] Lexer com 150+ keywords
- [x] Parser recursivo descendente gerando AST
- [x] 20 idiomas suportados (PT, EN, ES, FR, DE, IT, ZH, JA, KO, AR, HI, BN, RU, ID, TR, VI, PL, NL, TH, SW)
- [x] Sistema de imports com deteccao circular
- [x] Hot reload (re-exec on file change)
- [x] Comando `flang build` para gerar executavel standalone
- [x] Modo plano (1 arquivo) e organizado (pastas)

### Tipos de Dados (15 tipos)
- [x] texto, texto_longo, numero, dinheiro, email, telefone
- [x] data, booleano, imagem, arquivo, upload, link
- [x] status, senha, enum(valores)

### Relacionamentos
- [x] pertence_a (FK com dropdown automatico)
- [x] tem_muitos (1:N)
- [x] muitos_para_muitos (N:N com join table automatica)

### Banco de Dados
- [x] SQLite, MySQL, PostgreSQL
- [x] Auto-criacao de tabelas e auto-migration
- [x] Connection pooling (25 max, 5 idle)
- [x] Soft delete com restore
- [x] Validacao de campos e validacao customizada
- [x] Paginacao, filtros, busca, ordenacao
- [x] Export CSV/JSON

### Autenticacao e Seguranca (16 features)
- [x] JWT + bcrypt + roles + rate limiting
- [x] Login/registro no frontend
- [x] SSRF, XSS, path traversal, upload whitelist, body limits, CSV injection protection
- [x] JWT secret via env variable

### Frontend
- [x] SPA com Tailwind CSS + dark/light mode
- [x] Dashboard com Chart.js
- [x] Sidebar customizavel + tabs de status/enum
- [x] Modais no body level + FK/enum dropdowns
- [x] WebSocket real-time + toast notifications

### Tema e Customizacao
- [x] 5 presets (moderno, simples, elegante, corporativo, claro)
- [x] Cores por nome (14 cores) + 4 estilos visuais
- [x] CSS customizado, controle total de fonte/borda/fundo

### Scripting (30+ funcoes built-in)
- [x] Variaveis, funcoes, controle de fluxo, try/catch
- [x] Array indexing, object access, DB queries
- [x] HTTP client, JSON parse, async (paralelo, esperar, timeout)

### Integracoes
- [x] WhatsApp (whatsmeow), Email SMTP (HTML), Cron jobs
- [x] HTTP client, WebSocket hub, proxy endpoint

### Customizacao Avancada
- [x] Rotas customizadas, paginas HTML, sidebar customizavel
- [x] Telas customizadas respeitadas pelo frontend

### CLI
- [x] run, check, new, init, build, docker, ide, version, help

### IDE Web (Flang IDE)
- [x] Monaco Editor com syntax highlighting para .fg
- [x] File tree, tabs, Ctrl+S, terminal
- [x] Run/Stop/Check buttons
- [x] 3 modos: Codigo, Designer, Fluxos
- [x] Designer com canvas (Fabric.js em implementacao)
- [x] Flow editor com nodes + conexoes SVG

### Testes
- [x] 59 testes (lexer, parser, AST, interpreter)

### VS Code Extension
- [x] Syntax highlighting, 22 snippets, auto-indentacao

### Exemplos
- [x] Loja (plano + organizado)
- [x] Evoticket (34 arquivos, 24 modelos, 18 funcoes)

---

## PARTE 2: EM IMPLEMENTACAO (Onda 1)

### IDE — Canvas Visual com Fabric.js
- [ ] Canvas infinito com zoom (scroll) e pan (Alt+drag)
- [ ] 12 componentes arrastaveis com resize livre
- [ ] Snap-to-grid (20px)
- [ ] Propriedades no painel direito
- [ ] Geracao automatica de .fg em tempo real
- [ ] Controles de zoom (+, -, Reset)
- [ ] Delete com tecla Del

---

## PARTE 3: PROXIMAS IMPLEMENTACOES (50 features planejadas)

### Onda 2 — Novos Tipos de Campo (10 features)
| # | Feature | Sintaxe | Status |
|---|---------|---------|--------|
| 1 | CPF/CNPJ com validacao | `cpf: cpf` | 🔜 |
| 2 | CEP com busca automatica (ViaCEP) | `cep: cep` | 🔜 |
| 3 | Color picker | `cor_favorita: cor` | 🔜 |
| 4 | Rating/estrelas (1-5) | `avaliacao: estrelas` | 🔜 |
| 5 | Slider numerico | `quantidade: slider(0, 100)` | 🔜 |
| 6 | Date range (periodo) | `periodo: periodo` | 🔜 |
| 7 | Tags/chips (multiplos valores) | `habilidades: tags` | 🔜 |
| 8 | Rich text editor | `conteudo: richtext` | 🔜 |
| 9 | Assinatura digital (canvas) | `assinatura: assinatura` | 🔜 |
| 10 | Geolocalizacao (mapa) | `local: localizacao` | 🔜 |

### Onda 3 — Integracoes (15 features)
| # | Feature | Sintaxe | Status |
|---|---------|---------|--------|
| 11 | PIX QR Code | `pix gerar qrcode 49.90` | 🔜 |
| 12 | Stripe pagamento | `stripe cobrar 99.90` | 🔜 |
| 13 | MercadoPago | `mercadopago link 49.90` | 🔜 |
| 14 | Telegram bot | `telegram enviar "msg"` | 🔜 |
| 15 | Discord webhook | `discord enviar "msg"` | 🔜 |
| 16 | Slack | `slack enviar "msg"` | 🔜 |
| 17 | SMS (Twilio) | `sms enviar "msg" para tel` | 🔜 |
| 18 | Google Sheets | `planilha exportar produto` | 🔜 |
| 19 | Google Calendar | `calendario criar evento` | 🔜 |
| 20 | S3/MinIO storage | `storage upload arquivo` | 🔜 |
| 21 | OpenAI (ChatGPT) | `ia.completar("pergunta")` | 🔜 |
| 22 | Claude (Anthropic) | `ia.completar("pergunta")` | 🔜 |
| 23 | Gemini (Google) | `ia.completar("pergunta")` | 🔜 |
| 24 | Webhook inbound | receber POST externo | 🔜 |
| 25 | Zapier/Make compativel | webhook padrao | 🔜 |

### Onda 4 — UX para Leigos (15 features)
| # | Feature | Descricao | Status |
|---|---------|-----------|--------|
| 26 | Templates prontos | `flang new loja`, `clinica`, `escola`, `delivery` | 🔜 |
| 27 | Wizard de criacao | perguntas guiadas → gera .fg | 🔜 |
| 28 | LSP (autocomplete) | sugestoes no editor | 🔜 |
| 29 | Preview ao vivo na IDE | iframe com o app rodando | 🔜 |
| 30 | Undo/Redo no canvas | Ctrl+Z / Ctrl+Y | 🔜 |
| 31 | Copiar/Colar componentes | Ctrl+C / Ctrl+V | 🔜 |
| 32 | Alinhar componentes | esquerda, centro, direita, distribuir | 🔜 |
| 33 | Snap lines | guias ao alinhar com outro componente | 🔜 |
| 34 | Layers panel | lista de componentes com z-index | 🔜 |
| 35 | Responsive preview | mobile, tablet, desktop | 🔜 |
| 36 | Erro com sugestao | "voce quis dizer 'texto'?" | 🔜 |
| 37 | Tour guiado | primeiro uso mostra cada parte | 🔜 |
| 38 | Video tutoriais embutidos | dentro da IDE | 💡 |
| 39 | Documentacao interativa | exemplos clicaveis no browser | 🔜 |
| 40 | Export PNG/PDF do canvas | botao de exportar design | 🔜 |

### Onda 5 — Deploy e Distribuicao (10 features)
| # | Feature | Descricao | Status |
|---|---------|-----------|--------|
| 41 | Deploy 1 comando | `flang deploy` publica online | 🔜 |
| 42 | Dominio customizado | `flang deploy --dominio meuapp.com` | 🔜 |
| 43 | HTTPS automatico | Let's Encrypt integrado | 🔜 |
| 44 | Compartilhar projeto | `flang share` gera link | 🔜 |
| 45 | Atualizar sem downtime | `flang update` | 🔜 |
| 46 | Backup na nuvem | `flang backup` | 💡 |
| 47 | Git integrado na IDE | commit/push dentro da IDE | 🔜 |
| 48 | AI assistant na IDE | chat que gera .fg | 🔜 |
| 49 | Marketplace de templates | baixar e compartilhar | 💡 |
| 50 | App Electron da IDE | aplicativo desktop instalavel | 🔜 |

---

## PARTE 4: VERSOES PLANEJADAS

### v0.6.0 — Canvas Visual + Novos Tipos
- [ ] Fabric.js canvas no Designer
- [ ] 10 novos tipos de campo (CPF, CEP, cor, estrelas, slider, etc)
- [ ] Templates prontos (loja, clinica, escola, delivery)
- [ ] Preview ao vivo na IDE
- [ ] Undo/Redo no canvas

### v0.7.0 — Integracoes + AI
- [ ] PIX, Stripe, MercadoPago (pagamentos)
- [ ] Telegram, Discord, Slack, SMS (mensageria)
- [ ] Google Sheets, Calendar (produtividade)
- [ ] OpenAI/Claude/Gemini (inteligencia artificial)
- [ ] S3/MinIO (armazenamento)

### v0.8.0 — Developer Experience
- [ ] LSP (Language Server Protocol) com autocomplete
- [ ] REPL interativo (`flang repl`)
- [ ] Formatter (`flang fmt`)
- [ ] Testing framework built-in
- [ ] Erro com sugestao de correcao

### v0.9.0 — Enterprise
- [ ] Multi-tenancy nativo
- [ ] Permissoes granulares por modelo/campo
- [ ] Migrations versionadas com rollback
- [ ] Background jobs com retry
- [ ] Audit log automatico
- [ ] SSO / OAuth2

### v1.0.0 — Production Ready
- [ ] Deploy 1 comando com HTTPS
- [ ] App Electron da IDE
- [ ] Marketplace de templates e plugins
- [ ] Mobile app generation
- [ ] SSR para SEO + PWA
- [ ] PDF/report generation

---

## PARTE 5: COMPARACAO COM CONCORRENTES

| Feature | Flang | Bubble | Retool | Wasp | Adalo |
|---------|:-----:|:------:|:------:|:----:|:-----:|
| Open source | ✅ | ❌ | ❌ | ✅ | ❌ |
| 20 idiomas | ✅ | ❌ | ❌ | ❌ | ❌ |
| Declarativo (sem codigo) | ✅ | ❌ | ❌ | ✅ | ❌ |
| Canvas visual (Figma-like) | ✅ | ✅ | ❌ | ❌ | ✅ |
| Flow editor (logica visual) | ✅ | ✅ | ❌ | ❌ | ❌ |
| WhatsApp nativo | ✅ | ❌ | ❌ | ❌ | ❌ |
| Build executavel (.exe) | ✅ | ❌ | ❌ | ❌ | ❌ |
| IDE propria | ✅ | ✅ | ✅ | ❌ | ✅ |
| Multi-banco (SQLite/MySQL/PG) | ✅ | ❌ | ✅ | ❌ | ❌ |
| Self-hosted | ✅ | ❌ | ❌ | ✅ | ❌ |
| AI integrado | 🔜 | ✅ | ✅ | ❌ | ❌ |
| Mobile | 🔜 | ✅ | ✅ | ❌ | ✅ |
| Preco | **Gratis** | $29-599/mes | $10-50/user | Gratis | $36-200/mes |

### Diferenciais unicos do Flang
1. **Unica linguagem em 20 idiomas** — escreva em portugues, chines, arabe, etc
2. **Gera executavel standalone** — um .exe que roda sem instalar nada
3. **WhatsApp integrado** — nenhum concorrente tem
4. **100% gratis e open source** — sem limite de usuarios
5. **IDE com canvas visual + flow editor + code editor** — 3 em 1
6. **Zero dependencia** — nao precisa de Node, Python, Docker

---

## PARTE 6: METRICAS DO PROJETO

| Metrica | Valor |
|---------|-------|
| Linhas de Go | ~15.000 |
| Arquivos Go | 30+ |
| Keywords | 150+ |
| Idiomas | 20 |
| Funcoes built-in | 30+ |
| Testes | 59 |
| Plataformas de build | 6 |
| Exemplos | 3 (loja plano, loja organizado, evoticket) |
| Releases | 3 (v0.5.0, v0.5.1, em breve v0.6.0) |
| Features implementadas | 116 de 200 (58%) |
| Features planejadas | 50 novas |

---

*Documento atualizado em Abril 2026.*
*Baseado em pesquisa de mercado: Bubble.io, Adalo, Retool, Wasp, Budibase, Appsmith, Figma, Flowise.*
