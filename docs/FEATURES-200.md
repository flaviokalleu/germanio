# Germanio — 200 Features Para Leigos

> **Escopo desta referência:** Catálogo de ideias e registros históricos; não é especificação nem comprovação de entrega. Marcações “implementado” abaixo precisam ser conferidas no código e nos testes da revisão usada. Prioridades atuais: [ROADMAP.md](ROADMAP.md). Semântica normativa: [INTENCAO.md](INTENCAO.md).

**Objetivo:** Tornar o Germanio a plataforma mais acessivel do mundo para qualquer pessoa criar aplicacoes completas, sem precisar saber programar.

**Legenda:** ✅ Ja implementado | 🔜 Planejado | 💡 Ideia nova

---

## 1. COMPONENTES VISUAIS (UI) — 40 features

### Campos de Entrada
| # | Feature | Status | Como usar no .ge |
|---|---------|--------|-----------------|
| 1 | Campo de texto simples | ✅ | `nome: texto` |
| 2 | Campo de texto longo (textarea) | ✅ | `descricao: texto_longo` |
| 3 | Campo numerico | ✅ | `quantidade: numero` |
| 4 | Campo de dinheiro (formatado) | ✅ | `preco: dinheiro` |
| 5 | Campo de email (com validacao) | ✅ | `email: email` |
| 6 | Campo de telefone (com validacao) | ✅ | `telefone: telefone` |
| 7 | Campo de data | ✅ | `nascimento: data` |
| 8 | Campo de senha (oculto) | ✅ | `senha: senha` |
| 9 | Toggle on/off | ✅ | `ativo: booleano` |
| 10 | Dropdown de status | ✅ | `status: status` |
| 11 | Dropdown com opcoes fixas | ✅ | `cor: enum(azul, verde, vermelho)` |
| 12 | Dropdown de modelo relacionado | ✅ | `categoria: texto pertence_a categoria` |
| 13 | Upload de imagem com preview | ✅ | `foto: imagem` |
| 14 | Upload de arquivo | ✅ | `documento: arquivo` |
| 15 | Campo de link/URL | ✅ | `site: link` |
| 16 | Campo de CPF/CNPJ | 🔜 | `cpf: cpf` |
| 17 | Campo de CEP (com busca automatica) | 🔜 | `cep: cep` |
| 18 | Campo de cor (color picker) | 🔜 | `cor_favorita: cor` |
| 19 | Campo de horario | 🔜 | `horario: hora` |
| 20 | Campo de data e hora junto | 🔜 | `agendamento: data_hora` |
| 21 | Campo de porcentagem | 🔜 | `desconto: percentual` |
| 22 | Campo de estrelas (rating) | 🔜 | `avaliacao: estrelas` |
| 23 | Campo de assinatura (desenho) | 💡 | `assinatura: assinatura` |
| 24 | Campo de localizacao (mapa) | 💡 | `endereco: localizacao` |
| 25 | Campo de codigo de barras | 💡 | `codigo: codigo_barras` |

### Componentes de Tela
| # | Feature | Status | Como usar no .ge |
|---|---------|--------|-----------------|
| 26 | Lista/tabela de registros | ✅ | `lista produto` |
| 27 | Botao de acao | ✅ | `botao azul texto "Novo"` |
| 28 | Busca/filtro | ✅ | `busca produto` |
| 29 | Grafico de barras | ✅ | `grafico vendas tipo barra` |
| 30 | Grafico de pizza | ✅ | `grafico vendas tipo pizza` |
| 31 | Dashboard com cards | ✅ | `dashboard` |
| 32 | Tabs de status | ✅ | automatico para campos enum/status |
| 33 | Formulario modal | ✅ | automatico ao clicar "Novo" |
| 34 | Card de informacao | 🔜 | `card "Total Vendas" valor vendas.contar()` |
| 35 | Calendario visual | 🔜 | `calendario evento` |
| 36 | Kanban (arrastar e soltar) | 🔜 | `kanban tarefa por status` |
| 37 | Timeline/historico | 🔜 | `timeline atividade` |
| 38 | Galeria de imagens | 🔜 | `galeria produto.foto` |
| 39 | Mapa com marcadores | 💡 | `mapa cliente por cidade` |
| 40 | QR Code generator | 💡 | `qrcode link` |

---

## 2. LOGICA SEM CODIGO — 30 features

### Palavras que viram codigo
| # | Feature | Status | Como escrever |
|---|---------|--------|--------------|
| 41 | Criar registro | ✅ | `criar produto` |
| 42 | Atualizar registro | ✅ | `atualizar produto` |
| 43 | Deletar registro | ✅ | `deletar produto` |
| 44 | Se/senao (condicao) | ✅ | `se preco maior 100` |
| 45 | Repetir N vezes | ✅ | `repetir 10 vezes` |
| 46 | Para cada item | ✅ | `para cada item em lista` |
| 47 | Enquanto (loop) | ✅ | `enquanto contador menor 100` |
| 48 | Definir variavel | ✅ | `definir total = 0` |
| 49 | Funcao com retorno | ✅ | `funcao calcular(a, b) retornar a + b` |
| 50 | Mostrar mensagem | ✅ | `mostrar "Ola mundo"` |
| 51 | Validar campo | ✅ | `validar preco maior 0` |
| 52 | Validar obrigatorio | ✅ | `validar nome obrigatorio` |
| 53 | Listar todos | ✅ | `produto.listar()` |
| 54 | Contar registros | ✅ | `produto.contar()` |
| 55 | Buscar por ID | ✅ | `produto.buscar(1)` |
| 56 | Chamar API externa | ✅ | `chamar("https://api.com/dados")` |
| 57 | Executar em paralelo | ✅ | `paralelo(["func1", "func2"])` |
| 58 | Esperar (delay) | ✅ | `esperar(1000)` |
| 59 | Timeout | ✅ | `timeout("funcao", 5000)` |
| 60 | Manipular texto | ✅ | `maiusculo("ola")` → `"OLA"` |
| 61 | Matematica | ✅ | `arredondar(3.7)` → `4` |
| 62 | Trabalhar com datas | ✅ | `formato_data(agora(), "DD/MM/YYYY")` |
| 63 | Parse JSON | ✅ | `json('{"a":1}')` → objeto |
| 64 | Substituir texto | ✅ | `substituir("ola mundo", "mundo", "germanio")` |
| 65 | Quando clicar botao | ✅ | `quando clicar "Novo" criar produto` |
| 66 | Quando criar registro | ✅ | `quando criar produto enviar mensagem` |
| 67 | Quando atualizar registro | ✅ | `quando atualizar ticket enviar email` |
| 68 | Agendar tarefa | ✅ | `cada 5 minutos chamar api "url"` |
| 69 | Importar de outro arquivo | ✅ | `importar "dados/produto.ge"` |
| 70 | Calcular soma | 🔜 | `vendas.somar(valor)` |

---

## 3. VISUAL E DESIGN — 25 features

| # | Feature | Status | Como usar |
|---|---------|--------|----------|
| 71 | Tema escuro | ✅ | `tema moderno escuro` |
| 72 | Tema claro | ✅ | `tema claro` |
| 73 | Preset moderno | ✅ | `tema moderno` |
| 74 | Preset simples | ✅ | `tema simples` |
| 75 | Preset elegante | ✅ | `tema elegante` |
| 76 | Preset corporativo | ✅ | `tema corporativo` |
| 77 | Cores por nome | ✅ | `cor primaria azul` |
| 78 | Estilo glassmorphism | ✅ | `estilo glassmorphism` |
| 79 | Estilo flat | ✅ | `estilo flat` |
| 80 | Estilo neumorphism | ✅ | `estilo neumorphism` |
| 81 | Estilo minimal | ✅ | `estilo minimal` |
| 82 | Fonte customizada | ✅ | `fonte "Poppins"` |
| 83 | Borda arredondada | ✅ | `borda "16px"` |
| 84 | CSS customizado | ✅ | `css ".minha-classe { ... }"` |
| 85 | Sidebar personalizada | ✅ | `sidebar item "Vendas" icone "dollar"` |
| 86 | Responsivo (mobile) | ✅ | automatico |
| 87 | Logo/icone customizado | ✅ | `icone "logo.png"` |
| 88 | Cores degradê | 🔜 | `fundo gradiente azul roxo` |
| 89 | Animacoes de entrada | 🔜 | `animacao fadeIn` |
| 90 | Favicon customizado | 🔜 | `favicon "icon.ico"` |
| 91 | Tela de loading | 🔜 | `loading "Carregando..."` |
| 92 | Notificacao push | 🔜 | `notificar "Novo pedido!"` |
| 93 | Som de notificacao | 💡 | `som "alerta"` |
| 94 | Modo kiosk (tela cheia) | 💡 | `modo kiosk` |
| 95 | Tema por usuario | 💡 | cada usuario escolhe seu tema |

---

## 4. SEGURANCA — 20 features

| # | Feature | Status | Como funciona |
|---|---------|--------|--------------|
| 96 | Login com email/senha | ✅ | `autenticacao campo_login: email` |
| 97 | Registro de conta | ✅ | automatico com auth |
| 98 | Senha criptografada (bcrypt) | ✅ | automatico |
| 99 | Token JWT | ✅ | automatico |
| 100 | Roles de usuario | ✅ | `roles: admin, vendedor` |
| 101 | Tela requer role | ✅ | `requer admin` |
| 102 | Bloqueio por tentativas | ✅ | 5 erros = bloqueio 5min |
| 103 | Rate limiting | ✅ | 100 POST/min por IP |
| 104 | Protecao XSS | ✅ | escape automatico |
| 105 | Protecao SSRF | ✅ | proxy bloqueia IPs privados |
| 106 | Protecao path traversal | ✅ | imports validados |
| 107 | Upload seguro | ✅ | whitelist de extensoes |
| 108 | Limite de body | ✅ | 1MB max |
| 109 | JWT via variavel ambiente | ✅ | `JWT_SECRET` no .env |
| 110 | Protecao CSV injection | ✅ | prefixo em formulas |
| 111 | CORS configurado | ✅ | automatico |
| 112 | Login com Google | 🔜 | `autenticacao google` |
| 113 | Login com GitHub | 🔜 | `autenticacao github` |
| 114 | Verificacao de email | 🔜 | enviar link de confirmacao |
| 115 | Dois fatores (2FA) | 🔜 | codigo no celular |

---

## 5. BANCO DE DADOS — 20 features

| # | Feature | Status | Como usar |
|---|---------|--------|----------|
| 116 | SQLite (zero config) | ✅ | padrao |
| 117 | MySQL | ✅ | `banco tipo: mysql` |
| 118 | PostgreSQL | ✅ | `banco tipo: postgresql` |
| 119 | Criar tabela automatico | ✅ | automatico |
| 120 | Adicionar coluna automatico | ✅ | auto-migration |
| 121 | Relacionamento 1:N | ✅ | `pertence_a / tem_muitos` |
| 122 | Relacionamento N:N | ✅ | `muitos_para_muitos` |
| 123 | Paginacao | ✅ | `?pagina=1&limite=10` |
| 124 | Busca em todos os campos | ✅ | `?busca=texto` |
| 125 | Filtro por campo | ✅ | `?status=ativo` |
| 126 | Ordenacao | ✅ | `?ordenar=nome&ordem=ASC` |
| 127 | Soft delete | ✅ | `soft_delete` no modelo |
| 128 | Restaurar deletado | ✅ | `/api/modelo/1/restaurar` |
| 129 | Connection pooling | ✅ | automatico |
| 130 | Validacao de campos | ✅ | `obrigatorio`, `unico` |
| 131 | Backup automatico | 🔜 | `cada 24 horas backup banco` |
| 132 | Importar CSV | 🔜 | `importar "dados.csv" em produto` |
| 133 | Exportar CSV/JSON | ✅ | `/api/modelo/export/csv` |
| 134 | MongoDB | 💡 | `banco tipo: mongodb` |
| 135 | Redis (cache) | 💡 | `cache tipo: redis` |

---

## 6. INTEGRACOES — 25 features

| # | Feature | Status | Como usar |
|---|---------|--------|----------|
| 136 | WhatsApp (enviar mensagem) | ✅ | `whatsapp enviar mensagem` |
| 137 | WhatsApp (QR code) | ✅ | automatico |
| 138 | Email SMTP | ✅ | `email servidor: "smtp.gmail.com"` |
| 139 | Email HTML | ✅ | deteccao automatica |
| 140 | Cron jobs | ✅ | `cada 5 minutos` |
| 141 | HTTP client | ✅ | `chamar("url")` |
| 142 | WebSocket real-time | ✅ | automatico |
| 143 | Proxy para APIs | ✅ | `/api/_proxy` |
| 144 | Telegram | 🔜 | `telegram enviar mensagem` |
| 145 | Instagram DM | 🔜 | `instagram enviar mensagem` |
| 146 | Facebook Messenger | 🔜 | `facebook enviar mensagem` |
| 147 | Slack | 🔜 | `slack enviar mensagem` |
| 148 | Discord | 🔜 | `discord enviar mensagem` |
| 149 | SMS (Twilio) | 🔜 | `sms enviar para telefone` |
| 150 | PIX (pagamento) | 🔜 | `pix gerar qrcode valor` |
| 151 | Stripe (pagamento) | 🔜 | `stripe cobrar valor` |
| 152 | MercadoPago | 🔜 | `mercadopago cobrar valor` |
| 153 | Google Sheets | 🔜 | `google_sheets exportar dados` |
| 154 | Google Calendar | 🔜 | `google_calendar criar evento` |
| 155 | Google Maps | 💡 | `mapa endereco` |
| 156 | OpenAI (ChatGPT) | 🔜 | `ia.completar("pergunta")` |
| 157 | Claude (Anthropic) | 🔜 | `ia.completar("pergunta")` |
| 158 | Gemini (Google) | 🔜 | `ia.completar("pergunta")` |
| 159 | S3 (armazenamento) | 🔜 | `storage s3 bucket "meu-bucket"` |
| 160 | Webhook outbound | 🔜 | `webhook enviar para "url"` |

---

## 7. PARA LEIGOS — 30 features especificas

### Facilitar a Vida
| # | Feature | Status | O que faz |
|---|---------|--------|----------|
| 161 | Tema com uma palavra | ✅ | `tema moderno` — pronto, bonito |
| 162 | Cor por nome | ✅ | `cor primaria azul` — sem hex |
| 163 | 20 idiomas | ✅ | escreva em qualquer lingua |
| 164 | Modo plano (1 arquivo) | ✅ | `germanio new meuapp` |
| 165 | Modo organizado (pastas) | ✅ | `germanio init meuapp` |
| 166 | Hot reload | ✅ | muda o arquivo, atualiza sozinho |
| 167 | Exemplos prontos | ✅ | loja, evoticket inclusos |
| 168 | Snippets no VS Code | ✅ | digita `dados` + Tab = pronto |
| 169 | Syntax highlighting | ✅ | cores no codigo |
| 170 | Mensagens de erro em portugues | ✅ | `campo 'nome' e obrigatorio` |

### Coisas que Leigos Precisam
| # | Feature | Status | O que faz |
|---|---------|--------|----------|
| 171 | Assistente IA no editor | 🔜 | descreve o que quer, IA gera o .ge |
| 172 | Templates prontos | 🔜 | `germanio new loja`, `germanio new clinica`, `germanio new escola` |
| 173 | Wizard de criacao | 🔜 | perguntas guiadas para criar o app |
| 174 | Preview ao vivo no editor | 🔜 | veja o resultado enquanto escreve |
| 175 | Documentacao interativa | 🔜 | exemplos clicaveis no browser |
| 176 | Video tutoriais embutidos | 💡 | videos no help |
| 177 | Modo tutorial (passo a passo) | 💡 | guia interativo no primeiro uso |
| 178 | Sugestao de erro com correcao | 🔜 | "voce quis dizer 'texto'?" |
| 179 | Auto-completar inteligente | 🔜 | LSP sugere campos e tipos |
| 180 | Validacao ao salvar | ✅ | `germanio check` valida antes de rodar |

### Publicar sem Dor de Cabeca
| # | Feature | Status | O que faz |
|---|---------|--------|----------|
| 181 | Gerar executavel | ✅ | `germanio build` — um .exe |
| 182 | Gerar Docker | ✅ | `germanio docker` |
| 183 | Instalar com 1 clique (Windows) | ✅ | GermanioSetup.exe |
| 184 | Instalar com 1 linha (Linux) | ✅ | `curl ... \| sh` |
| 185 | Deploy em 1 comando | 🔜 | `germanio deploy` |
| 186 | Dominio customizado | 🔜 | `germanio deploy --dominio meuapp.com` |
| 187 | HTTPS automatico | 🔜 | Let's Encrypt integrado |
| 188 | Backup na nuvem | 💡 | `germanio backup` |
| 189 | Atualizar app rodando | 🔜 | `germanio update` sem downtime |
| 190 | Compartilhar projeto | 🔜 | `germanio share` gera link |

---

## 8. DADOS PRONTOS — 10 features

| # | Feature | Status | Como usar |
|---|---------|--------|----------|
| 191 | Dados iniciais (seeds) | 🔜 | `sementes usuario nome "Admin"` |
| 192 | Importar de planilha | 🔜 | `germanio import dados.xlsx` |
| 193 | Exportar para planilha | ✅ | `/api/modelo/export/csv` |
| 194 | Exportar para JSON | ✅ | `/api/modelo/export/json` |
| 195 | Importar de outro banco | 💡 | `germanio import --de mysql://...` |
| 196 | Relatorio PDF | 🔜 | `relatorio vendas formato pdf` |
| 197 | Grafico exportavel | 🔜 | botao "Baixar grafico" |
| 198 | Notificacao por email de relatorio | 🔜 | `cada semana enviar relatorio` |
| 199 | Auditoria (quem fez o que) | 🔜 | log automatico de acoes |
| 200 | Historico de mudancas | 🔜 | versoes anteriores de cada registro |

---

## RESUMO

| Categoria | Total | ✅ Feito | 🔜 Planejado | 💡 Ideia |
|-----------|-------|---------|-------------|---------|
| Componentes UI | 40 | 17 | 13 | 10 |
| Logica sem codigo | 30 | 29 | 1 | 0 |
| Visual e Design | 25 | 17 | 5 | 3 |
| Seguranca | 20 | 16 | 4 | 0 |
| Banco de Dados | 20 | 15 | 3 | 2 |
| Integracoes | 25 | 8 | 14 | 3 |
| Para Leigos | 30 | 11 | 13 | 6 |
| Dados Prontos | 10 | 3 | 5 | 2 |
| **TOTAL** | **200** | **116** | **58** | **26** |

**116 de 200 features ja implementadas (58%).**
**58 planejadas para as proximas versoes.**
**26 ideias para o futuro.**

---

*Pesquisa baseada em: Bubble.io, Adalo, Glide, Retool, Budibase, Appsmith, Wasp, WordPress, ToolJet e tendencias de low-code 2026.*
