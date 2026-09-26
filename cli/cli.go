package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"

	"github.com/flaviokalleu/germanio/runtime"
	"github.com/flaviokalleu/germanio/runtime/ide"
)

const version = "0.6.0"

const banner = `
  ███████╗██╗      █████╗ ███╗   ██╗ ██████╗
  ██╔════╝██║     ██╔══██╗████╗  ██║██╔════╝
  █████╗  ██║     ███████║██╔██╗ ██║██║  ███╗
  ██╔══╝  ██║     ██╔══██╗██║╚██╗██║██║   ██║
  ██║     ███████╗██║  ██║██║ ╚████║╚██████╔╝
  ╚═╝     ╚══════╝╚═╝  ╚═╝╚═╝  ╚═══╝ ╚═════╝
  v%s - Tudo roda direto do .fg
`

// Run executes the CLI.
func Run(args []string) {
	if len(args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch args[1] {
	case "run":
		if len(args) < 3 {
			fmt.Println("Uso: flang run <arquivo.fg>")
			os.Exit(1)
		}
		porta := "8080"
		if len(args) >= 4 {
			porta = args[3]
		}
		if err := runtime.Executar(args[2], porta); err != nil {
			fmt.Printf("[flang] ERRO: %s\n", err)
			os.Exit(1)
		}

	case "check":
		if len(args) < 3 {
			fmt.Println("Uso: flang check <arquivo.fg>")
			os.Exit(1)
		}
		if err := runtime.Verificar(args[2]); err != nil {
			fmt.Printf("[flang] ERRO: %s\n", err)
			os.Exit(1)
		}

	case "new":
		if len(args) < 3 {
			fmt.Println("Uso: flang new <nome>")
			os.Exit(1)
		}
		cmdNew(args[2])

	case "version":
		fmt.Printf(banner, version)

	case "docker":
		cmdDocker()

	case "init":
		if len(args) < 3 {
			fmt.Println("Uso: flang init <nome>")
			os.Exit(1)
		}
		cmdInit(args[2])

	case "build":
		if len(args) < 3 {
			fmt.Println("Uso: flang build <arquivo.fg> [--output nome]")
			os.Exit(1)
		}
		output := ""
		for i, a := range args {
			if (a == "--output" || a == "-o") && i+1 < len(args) {
				output = args[i+1]
			}
		}
		cmdBuild(args[2], output)

	case "ide":
		dir := "."
		porta := "3000"
		if len(args) >= 3 {
			dir = args[2]
		}
		for i, a := range args {
			if (a == "--port" || a == "-p") && i+1 < len(args) {
				porta = args[i+1]
			}
		}
		cmdIDE(dir, porta)

	case "help":
		printUsage()

	default:
		// If arg ends in .fg, treat it as "run"
		if strings.HasSuffix(args[1], ".fg") {
			porta := "8080"
			if len(args) >= 3 {
				porta = args[2]
			}
			if err := runtime.Executar(args[1], porta); err != nil {
				fmt.Printf("[flang] ERRO: %s\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Printf("Comando desconhecido: %s\n", args[1])
			printUsage()
			os.Exit(1)
		}
	}
}

func printUsage() {
	fmt.Printf(banner, version)
	fmt.Print(`
Uso: flang <comando> [argumentos]

Comandos:
  run <arquivo.fg> [porta]  Executa o arquivo .fg (porta padrao: 8080)
  check <arquivo.fg>        Verifica sintaxe sem executar
  new <nome>                Cria projeto plano (tudo num arquivo so)
  init <nome>               Cria projeto organizado (pastas por responsabilidade)
  build <arquivo.fg> [-o nome]  Compila em executavel standalone
  ide [diretorio] [-p porta]  Abre a IDE web do Flang
  docker                    Gera Dockerfile para o projeto atual
  version                   Mostra a versao
  help                      Mostra esta ajuda

Modos de projeto:
  new   → Modo plano: um arquivo so, ideal para comecar rapido.
  init  → Modo organizado: dados/, telas/, eventos/ separados.
          Comece com 'new' e migre para 'init' quando crescer.

Atalho:
  flang inicio.fg           Mesmo que "flang run inicio.fg"

Exemplo:
  flang new meuapp          Cria projeto plano
  flang init meuapp         Cria projeto organizado
  flang run meuapp/inicio.fg
`)
}

func cmdNew(name string) {
	dir := name
	baseName := filepath.Base(name)
	title := strings.ToUpper(baseName[:1]) + baseName[1:]

	// Check if name is a template
	template := detectTemplate(baseName)

	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("Erro: %s\n", err)
		os.Exit(1)
	}

	var fg string
	if template != "" {
		fg = getTemplate(template, baseName)
	} else {
		// Modo plano: tudo num arquivo só, simples e direto
		fg = `sistema ` + baseName + `

tema
  cor primaria "#6366f1"
  cor secundaria "#8b5cf6"

dados

  produto
    nome: texto obrigatorio
    descricao: texto
    preco: dinheiro
    estoque: numero
    status: status

  cliente
    nome: texto obrigatorio
    email: email unico
    telefone: telefone
    status: status

telas

  tela produtos
    titulo "Produtos"
    lista produto
      mostrar nome
      mostrar preco
      mostrar estoque
      mostrar status
    botao azul
      texto "Novo Produto"

  tela clientes
    titulo "Clientes"
    lista cliente
      mostrar nome
      mostrar email
      mostrar telefone
      mostrar status
    botao verde
      texto "Novo Cliente"

eventos

  quando clicar "Novo Produto"
    criar produto

  quando clicar "Novo Cliente"
    criar cliente
`
	}

	fgPath := filepath.Join(dir, "inicio.fg")
	if err := os.WriteFile(fgPath, []byte(fg), 0644); err != nil {
		fmt.Printf("Erro: %s\n", err)
		os.Exit(1)
	}

	if template != "" {
		fmt.Printf("[flang] Projeto '%s' criado! (template: %s)\n", title, template)
	} else {
		fmt.Printf("[flang] Projeto '%s' criado! (modo plano)\n", title)
	}
	fmt.Println("[flang] Tudo num arquivo so - simples e direto.")
	fmt.Printf("[flang] Execute: flang %s\n", fgPath)
	fmt.Println()
	fmt.Println("[flang] Dica: quando crescer, use 'flang init' para modo organizado.")
}

func detectTemplate(name string) string {
	templates := map[string]string{
		"loja": "loja", "store": "loja", "shop": "loja", "ecommerce": "loja",
		"clinica": "clinica", "clinic": "clinica", "consultorio": "clinica", "hospital": "clinica",
		"escola": "escola", "school": "escola", "curso": "escola", "academy": "escola",
		"delivery": "delivery", "entrega": "delivery", "food": "delivery", "restaurante": "delivery",
		"crm": "crm", "vendas": "crm", "sales": "crm",
		"helpdesk": "helpdesk", "suporte": "helpdesk", "support": "helpdesk", "ticket": "helpdesk",
		"blog": "blog", "portfolio": "blog", "site": "blog",
		"financeiro": "financeiro", "finance": "financeiro", "contabil": "financeiro",
	}
	return templates[strings.ToLower(name)]
}

func getTemplate(template, name string) string {
	switch template {
	case "loja":
		return `sistema ` + name + `

tema moderno escuro

dados

  categoria
    nome: texto obrigatorio
    descricao: texto

  produto
    nome: texto obrigatorio
    descricao: texto_longo
    preco: dinheiro
    estoque: numero
    imagem: imagem
    categoria: texto pertence_a categoria
    status: enum(ativo, inativo, esgotado)

  cliente
    nome: texto obrigatorio
    cpf: cpf
    email: email unico
    telefone: telefone
    cep: cep
    cidade: texto
    status: status

  pedido
    cliente: texto pertence_a cliente
    produto: texto pertence_a produto
    quantidade: numero
    valor: dinheiro
    forma_pagamento: enum(pix, cartao, boleto, dinheiro)
    status: enum(pendente, pago, enviado, entregue, cancelado)

telas

  tela produtos
    titulo "Produtos"
    busca produto
    lista produto
      mostrar nome
      mostrar preco
      mostrar estoque
      mostrar status
    botao azul
      texto "Novo Produto"

  tela clientes
    titulo "Clientes"
    lista cliente
      mostrar nome
      mostrar email
      mostrar telefone
      mostrar status
    botao verde
      texto "Novo Cliente"

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

  quando clicar "Novo Produto"
    criar produto

  quando clicar "Novo Cliente"
    criar cliente

  quando clicar "Novo Pedido"
    criar pedido
`
	case "clinica":
		return `sistema ` + name + `

tema elegante

dados

  paciente
    nome: texto obrigatorio
    cpf: cpf obrigatorio unico
    data_nascimento: data
    telefone: telefone
    email: email
    cep: cep
    cidade: texto
    convenio: texto
    tipo_sanguineo: enum(A+, A-, B+, B-, AB+, AB-, O+, O-)
    alergias: tags
    status: status

  medico
    nome: texto obrigatorio
    crm: texto obrigatorio unico
    especialidade: texto
    telefone: telefone
    email: email
    status: status

  consulta
    paciente: texto pertence_a paciente
    medico: texto pertence_a medico
    data_hora: data_hora obrigatorio
    tipo: enum(consulta, retorno, exame, cirurgia)
    observacoes: texto_longo
    valor: dinheiro
    status: enum(agendada, confirmada, realizada, cancelada, faltou)

  prontuario
    paciente: texto pertence_a paciente
    medico: texto pertence_a medico
    descricao: texto_longo obrigatorio
    prescricao: texto_longo
    exames: tags

telas

  tela pacientes
    titulo "Pacientes"
    busca paciente
    lista paciente
      mostrar nome
      mostrar cpf
      mostrar telefone
      mostrar convenio
      mostrar status
    botao azul
      texto "Novo Paciente"

  tela consultas
    titulo "Agenda"
    lista consulta
      mostrar paciente
      mostrar medico
      mostrar data_hora
      mostrar tipo
      mostrar status
    botao verde
      texto "Nova Consulta"

  tela medicos
    titulo "Medicos"
    lista medico
      mostrar nome
      mostrar crm
      mostrar especialidade
      mostrar status
    botao azul
      texto "Novo Medico"

eventos

  quando clicar "Novo Paciente"
    criar paciente

  quando clicar "Nova Consulta"
    criar consulta

  quando clicar "Novo Medico"
    criar medico
`
	case "escola":
		return `sistema ` + name + `

tema corporativo

dados

  aluno
    nome: texto obrigatorio
    cpf: cpf
    data_nascimento: data
    email: email
    telefone: telefone
    turma: texto pertence_a turma
    responsavel: texto
    telefone_responsavel: telefone
    status: enum(ativo, inativo, trancado, formado)

  professor
    nome: texto obrigatorio
    email: email obrigatorio unico
    telefone: telefone
    disciplina: texto
    status: status

  turma
    nome: texto obrigatorio
    periodo: enum(manha, tarde, noite, integral)
    professor: texto pertence_a professor
    ano: numero
    vagas: numero
    status: status

  nota
    aluno: texto pertence_a aluno
    disciplina: texto
    nota1: percentual
    nota2: percentual
    nota3: percentual
    nota4: percentual
    media: percentual
    status: enum(aprovado, reprovado, recuperacao, cursando)

telas

  tela alunos
    titulo "Alunos"
    busca aluno
    lista aluno
      mostrar nome
      mostrar turma
      mostrar status
    botao azul
      texto "Novo Aluno"

  tela turmas
    titulo "Turmas"
    lista turma
      mostrar nome
      mostrar periodo
      mostrar professor
      mostrar vagas
      mostrar status
    botao verde
      texto "Nova Turma"

  tela professores
    titulo "Professores"
    lista professor
      mostrar nome
      mostrar disciplina
      mostrar email
      mostrar status
    botao azul
      texto "Novo Professor"

  tela notas
    titulo "Notas"
    lista nota
      mostrar aluno
      mostrar disciplina
      mostrar media
      mostrar status
    botao azul
      texto "Lancar Nota"

eventos

  quando clicar "Novo Aluno"
    criar aluno

  quando clicar "Nova Turma"
    criar turma

  quando clicar "Novo Professor"
    criar professor

  quando clicar "Lancar Nota"
    criar nota
`
	case "delivery":
		return `sistema ` + name + `

tema moderno

dados

  categoria
    nome: texto obrigatorio
    descricao: texto
    icone: imagem

  item_cardapio
    nome: texto obrigatorio
    descricao: texto_longo
    preco: dinheiro obrigatorio
    imagem: imagem
    categoria: texto pertence_a categoria
    tempo_preparo: numero
    disponivel: booleano

  cliente
    nome: texto obrigatorio
    telefone: telefone obrigatorio
    cep: cep
    endereco: texto
    cidade: texto
    status: status

  pedido
    cliente: texto pertence_a cliente
    itens: tags
    valor_total: dinheiro
    taxa_entrega: dinheiro
    forma_pagamento: enum(pix, cartao, dinheiro, vale)
    endereco_entrega: texto
    observacoes: texto_longo
    avaliacao: estrelas
    status: enum(recebido, preparando, saiu_entrega, entregue, cancelado)

  entregador
    nome: texto obrigatorio
    telefone: telefone obrigatorio
    veiculo: enum(moto, bicicleta, carro, a_pe)
    status: enum(disponivel, entregando, offline)

telas

  tela cardapio
    titulo "Cardapio"
    busca item_cardapio
    lista item_cardapio
      mostrar nome
      mostrar preco
      mostrar categoria
      mostrar disponivel
    botao verde
      texto "Novo Item"

  tela pedidos
    titulo "Pedidos"
    lista pedido
      mostrar cliente
      mostrar valor_total
      mostrar forma_pagamento
      mostrar avaliacao
      mostrar status
    botao azul
      texto "Novo Pedido"

  tela entregadores
    titulo "Entregadores"
    lista entregador
      mostrar nome
      mostrar telefone
      mostrar veiculo
      mostrar status
    botao azul
      texto "Novo Entregador"

eventos

  quando clicar "Novo Item"
    criar item_cardapio

  quando clicar "Novo Pedido"
    criar pedido

  quando clicar "Novo Entregador"
    criar entregador
`
	case "crm":
		return `sistema ` + name + `

tema elegante escuro

dados

  contato
    nome: texto obrigatorio
    empresa: texto
    cargo: texto
    email: email
    telefone: telefone
    site: url
    origem: enum(indicacao, site, linkedin, evento, cold_call, outro)
    tags: tags
    status: enum(lead, qualificado, cliente, inativo)

  negocio
    titulo: texto obrigatorio
    valor: dinheiro
    contato: texto pertence_a contato
    estagio: enum(prospeccao, qualificacao, proposta, negociacao, fechamento)
    probabilidade: percentual
    previsao: data
    responsavel: texto
    notas: texto_longo
    status: enum(aberto, ganho, perdido)

  atividade
    titulo: texto obrigatorio
    tipo: enum(ligacao, email, reuniao, tarefa, visita)
    contato: texto pertence_a contato
    negocio: texto pertence_a negocio
    data_hora: data_hora
    descricao: texto_longo
    status: enum(pendente, concluida, cancelada)

telas

  tela contatos
    titulo "Contatos"
    busca contato
    lista contato
      mostrar nome
      mostrar empresa
      mostrar email
      mostrar telefone
      mostrar status
    botao azul
      texto "Novo Contato"

  tela negocios
    titulo "Negocios"
    lista negocio
      mostrar titulo
      mostrar valor
      mostrar contato
      mostrar estagio
      mostrar probabilidade
      mostrar status
    botao verde
      texto "Novo Negocio"

  tela atividades
    titulo "Atividades"
    lista atividade
      mostrar titulo
      mostrar tipo
      mostrar contato
      mostrar data_hora
      mostrar status
    botao azul
      texto "Nova Atividade"

eventos

  quando clicar "Novo Contato"
    criar contato

  quando clicar "Novo Negocio"
    criar negocio

  quando clicar "Nova Atividade"
    criar atividade
`
	case "helpdesk":
		return `sistema ` + name + `

tema moderno escuro

autenticacao
  modelo: atendente
  campo_login: email
  campo_senha: senha
  roles: admin, supervisor, atendente

dados

  atendente
    nome: texto obrigatorio
    email: email obrigatorio unico
    senha: senha obrigatorio
    setor: texto
    status: status

  cliente
    nome: texto obrigatorio
    email: email
    telefone: telefone
    empresa: texto
    status: status

  ticket
    titulo: texto obrigatorio
    descricao: texto_longo
    cliente: texto pertence_a cliente
    atendente: texto pertence_a atendente
    prioridade: enum(baixa, media, alta, urgente)
    categoria: enum(bug, duvida, solicitacao, melhoria)
    status: enum(aberto, em_andamento, aguardando, resolvido, fechado)

  resposta
    ticket: texto pertence_a ticket
    autor: texto
    mensagem: texto_longo obrigatorio
    tipo: enum(publica, interna)

telas

  tela tickets
    titulo "Tickets"
    busca ticket
    lista ticket
      mostrar titulo
      mostrar cliente
      mostrar atendente
      mostrar prioridade
      mostrar categoria
      mostrar status
    botao azul
      texto "Novo Ticket"

  tela clientes
    titulo "Clientes"
    lista cliente
      mostrar nome
      mostrar email
      mostrar telefone
      mostrar empresa
      mostrar status
    botao verde
      texto "Novo Cliente"

  tela atendentes
    titulo "Atendentes"
    requer admin
    lista atendente
      mostrar nome
      mostrar email
      mostrar setor
      mostrar status
    botao azul
      texto "Novo Atendente"

eventos

  quando clicar "Novo Ticket"
    criar ticket

  quando clicar "Novo Cliente"
    criar cliente

  quando clicar "Novo Atendente"
    criar atendente
`
	case "blog":
		return `sistema ` + name + `

tema simples

dados

  post
    titulo: texto obrigatorio
    slug: texto unico
    conteudo: texto_longo obrigatorio
    imagem_capa: imagem
    autor: texto
    tags: tags
    status: enum(rascunho, publicado, arquivado)

  pagina
    titulo: texto obrigatorio
    slug: texto unico
    conteudo: texto_longo obrigatorio
    status: enum(rascunho, publicado)

  comentario
    post: texto pertence_a post
    nome: texto obrigatorio
    email: email
    mensagem: texto_longo obrigatorio
    status: enum(pendente, aprovado, spam)

telas

  tela posts
    titulo "Posts"
    busca post
    lista post
      mostrar titulo
      mostrar autor
      mostrar tags
      mostrar status
    botao azul
      texto "Novo Post"

  tela paginas
    titulo "Paginas"
    lista pagina
      mostrar titulo
      mostrar slug
      mostrar status
    botao verde
      texto "Nova Pagina"

eventos

  quando clicar "Novo Post"
    criar post

  quando clicar "Nova Pagina"
    criar pagina
`
	case "financeiro":
		return `sistema ` + name + `

tema corporativo

dados

  conta
    nome: texto obrigatorio
    tipo: enum(corrente, poupanca, investimento, caixa)
    banco: texto
    saldo: dinheiro
    moeda: moeda
    status: status

  receita
    descricao: texto obrigatorio
    valor: dinheiro obrigatorio
    categoria: enum(vendas, servicos, investimentos, outros)
    conta: texto pertence_a conta
    data: data obrigatorio
    forma: enum(pix, cartao, boleto, dinheiro, transferencia)
    cliente: texto
    status: enum(pendente, recebido, atrasado)

  despesa
    descricao: texto obrigatorio
    valor: dinheiro obrigatorio
    categoria: enum(pessoal, aluguel, salarios, impostos, fornecedores, outros)
    conta: texto pertence_a conta
    data: data obrigatorio
    forma: enum(pix, cartao, boleto, dinheiro, transferencia)
    fornecedor: texto
    status: enum(pendente, pago, atrasado)

telas

  tela receitas
    titulo "Receitas"
    lista receita
      mostrar descricao
      mostrar valor
      mostrar categoria
      mostrar data
      mostrar status
    botao verde
      texto "Nova Receita"

  tela despesas
    titulo "Despesas"
    lista despesa
      mostrar descricao
      mostrar valor
      mostrar categoria
      mostrar data
      mostrar status
    botao vermelho
      texto "Nova Despesa"

  tela contas
    titulo "Contas"
    lista conta
      mostrar nome
      mostrar tipo
      mostrar banco
      mostrar saldo
      mostrar status
    botao azul
      texto "Nova Conta"

eventos

  quando clicar "Nova Receita"
    criar receita

  quando clicar "Nova Despesa"
    criar despesa

  quando clicar "Nova Conta"
    criar conta
`
	default:
		return ""
	}
}

func cmdDocker() {
	// Find .fg files in the current directory
	fgFile := "inicio.fg"
	entries, err := os.ReadDir(".")
	if err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".fg") {
				fgFile = e.Name()
				break
			}
		}
	}

	dockerfile := fmt.Sprintf(`# Generated by flang docker
FROM golang:1.26-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o flang .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /build/flang /usr/local/bin/flang
COPY *.fg ./

EXPOSE 8080
CMD ["flang", "run", "%s"]
`, fgFile)

	if err := os.WriteFile("Dockerfile", []byte(dockerfile), 0644); err != nil {
		fmt.Printf("[flang] Erro ao criar Dockerfile: %s\n", err)
		os.Exit(1)
	}
	fmt.Println("[flang] Dockerfile gerado com sucesso!")
	fmt.Println("[flang] Execute: docker build -t meu-app . && docker run -p 8080:8080 meu-app")
}

func cmdInit(name string) {
	dir := name
	baseName := filepath.Base(name)
	title := strings.ToUpper(baseName[:1]) + baseName[1:]

	// Criar estrutura organizada por responsabilidade
	// Inspirado no React: cada pasta tem um papel claro
	dirs := []string{
		dir,
		filepath.Join(dir, "dados"),  // modelos (como models/ ou types/)
		filepath.Join(dir, "telas"),  // interfaces (como pages/ ou components/)
		filepath.Join(dir, "eventos"), // interacoes (como handlers/ ou hooks/)
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			fmt.Printf("Erro: %s\n", err)
			os.Exit(1)
		}
	}

	// ── inicio.fg ── entry point (como App.js no React)
	inicio := `sistema ` + baseName + `

importar "tema.fg"
importar "dados/produto.fg"
importar "dados/cliente.fg"
importar "telas/produtos.fg"
importar "telas/clientes.fg"
importar "eventos/acoes.fg"
`
	// ── tema.fg ── visual (como theme.js)
	tema := `tema
  cor primaria "#6366f1"
  cor secundaria "#8b5cf6"
  cor destaque "#f59e0b"
`

	// ── dados/produto.fg ── um modelo por arquivo (como um component)
	produto := `dados

  produto
    nome: texto obrigatorio
    descricao: texto
    preco: dinheiro
    estoque: numero
    categoria: texto
    status: status
`

	// ── dados/cliente.fg
	cliente := `dados

  cliente
    nome: texto obrigatorio
    email: email unico
    telefone: telefone
    cidade: texto
    status: status
`

	// ── telas/produtos.fg
	telaProdutos := `telas

  tela produtos
    titulo "Produtos"
    lista produto
      mostrar nome
      mostrar preco
      mostrar estoque
      mostrar categoria
      mostrar status
    botao azul
      texto "Novo Produto"
`

	// ── telas/clientes.fg
	telaClientes := `telas

  tela clientes
    titulo "Clientes"
    lista cliente
      mostrar nome
      mostrar email
      mostrar telefone
      mostrar cidade
      mostrar status
    botao verde
      texto "Novo Cliente"
`

	// ── eventos/acoes.fg
	acoes := `eventos

  quando clicar "Novo Produto"
    criar produto

  quando clicar "Novo Cliente"
    criar cliente
`

	// Mapa de arquivos a criar
	files := map[string]string{
		filepath.Join(dir, "inicio.fg"):           inicio,
		filepath.Join(dir, "tema.fg"):              tema,
		filepath.Join(dir, "dados", "produto.fg"):  produto,
		filepath.Join(dir, "dados", "cliente.fg"):  cliente,
		filepath.Join(dir, "telas", "produtos.fg"): telaProdutos,
		filepath.Join(dir, "telas", "clientes.fg"): telaClientes,
		filepath.Join(dir, "eventos", "acoes.fg"):  acoes,
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			fmt.Printf("Erro ao criar %s: %s\n", path, err)
			os.Exit(1)
		}
	}

	// ── .env
	envContent := `# Configuracao do projeto ` + baseName + `
FLANG_PORT=8080
FLANG_DB_TYPE=sqlite
FLANG_DB_NAME=` + baseName + `.db
`
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		fmt.Printf("Erro ao criar .env: %s\n", err)
		os.Exit(1)
	}

	// ── .gitignore
	gitignore := `*.db
*.db-shm
*.db-wal
.env
flang
flang.exe
`
	giPath := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(giPath, []byte(gitignore), 0644); err != nil {
		fmt.Printf("Erro ao criar .gitignore: %s\n", err)
		os.Exit(1)
	}

	// ── Dockerfile
	dockerfileContent := `FROM golang:1.26-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o flang .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /build/flang /usr/local/bin/flang
COPY *.fg ./
COPY dados/ ./dados/
COPY telas/ ./telas/
COPY eventos/ ./eventos/

EXPOSE 8080
CMD ["flang", "run", "inicio.fg"]
`
	dfPath := filepath.Join(dir, "Dockerfile")
	if err := os.WriteFile(dfPath, []byte(dockerfileContent), 0644); err != nil {
		fmt.Printf("Erro ao criar Dockerfile: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("[flang] Projeto '%s' criado! (modo organizado)\n", title)
	fmt.Println()
	fmt.Printf("  %s/\n", name)
	fmt.Printf("  ├── inicio.fg          (entry point)\n")
	fmt.Printf("  ├── tema.fg            (visual)\n")
	fmt.Printf("  ├── dados/\n")
	fmt.Printf("  │   ├── produto.fg     (modelo)\n")
	fmt.Printf("  │   └── cliente.fg     (modelo)\n")
	fmt.Printf("  ├── telas/\n")
	fmt.Printf("  │   ├── produtos.fg    (interface)\n")
	fmt.Printf("  │   └── clientes.fg    (interface)\n")
	fmt.Printf("  ├── eventos/\n")
	fmt.Printf("  │   └── acoes.fg       (interacoes)\n")
	fmt.Printf("  ├── .env\n")
	fmt.Printf("  ├── .gitignore\n")
	fmt.Printf("  └── Dockerfile\n")
	fmt.Println()
	fmt.Printf("[flang] Execute: flang run %s\n", filepath.Join(name, "inicio.fg"))
	fmt.Println()
	fmt.Println("[flang] Adicione novos modelos em dados/, telas em telas/,")
	fmt.Println("        e importe no inicio.fg. Cada arquivo cuida de uma coisa.")
}

func cmdBuild(arquivo string, output string) {
	// Verify the .fg file exists
	if _, err := os.Stat(arquivo); os.IsNotExist(err) {
		fmt.Printf("[flang] Erro: arquivo '%s' nao encontrado\n", arquivo)
		os.Exit(1)
	}

	// First, verify the .fg file is valid
	if err := runtime.Verificar(arquivo); err != nil {
		fmt.Printf("[flang] Erro: %s\n", err)
		os.Exit(1)
	}

	// Determine output name
	if output == "" {
		base := filepath.Base(arquivo)
		output = strings.TrimSuffix(base, filepath.Ext(base))
		if goruntime.GOOS == "windows" {
			output += ".exe"
		}
	}

	// Collect all .fg files in the directory
	dir := filepath.Dir(arquivo)
	if dir == "" || dir == "." {
		dir, _ = os.Getwd()
	} else {
		dir, _ = filepath.Abs(dir)
	}
	mainFile := filepath.Base(arquivo)

	// Create temp build directory
	tmpDir, err := os.MkdirTemp("", "flang-build-*")
	if err != nil {
		fmt.Printf("[flang] Erro ao criar diretorio temporario: %s\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	// Copy all .fg files to temp dir preserving structure
	fgFiles := []string{}
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".fg" {
			rel, _ := filepath.Rel(dir, path)
			destPath := filepath.Join(tmpDir, "app", rel)
			os.MkdirAll(filepath.Dir(destPath), 0755)
			data, _ := os.ReadFile(path)
			os.WriteFile(destPath, data, 0644)
			fgFiles = append(fgFiles, rel)
		}
		return nil
	})

	// Copy .env if exists
	envPath := filepath.Join(dir, ".env")
	if _, err := os.Stat(envPath); err == nil {
		data, _ := os.ReadFile(envPath)
		os.MkdirAll(filepath.Join(tmpDir, "app"), 0755)
		os.WriteFile(filepath.Join(tmpDir, "app", ".env"), data, 0644)
	}

	// Generate main.go for the standalone binary
	mainGo := `package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/flaviokalleu/germanio/runtime"
)

//go:embed app/*
var appFS embed.FS

func main() {
	// Extract embedded files to temp dir
	tmpDir, err := os.MkdirTemp("", "flang-app-*")
	if err != nil {
		fmt.Printf("Erro: %s\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	// Walk embedded FS and write files
	entries, _ := appFS.ReadDir("app")
	extractDir(appFS, "app", tmpDir, entries)

	porta := "8080"
	if len(os.Args) > 1 {
		porta = os.Args[1]
	}
	if envPort := os.Getenv("PORT"); envPort != "" {
		porta = envPort
	}

	arquivo := filepath.Join(tmpDir, "` + mainFile + `")
	if err := runtime.Executar(arquivo, porta); err != nil {
		fmt.Printf("Erro: %s\n", err)
		os.Exit(1)
	}
}

func extractDir(fs embed.FS, base string, dest string, entries []os.DirEntry) {
	for _, e := range entries {
		srcPath := base + "/" + e.Name()
		destPath := filepath.Join(dest, e.Name())
		if e.IsDir() {
			os.MkdirAll(destPath, 0755)
			subEntries, _ := fs.ReadDir(srcPath)
			extractDir(fs, srcPath, destPath, subEntries)
		} else {
			data, _ := fs.ReadFile(srcPath)
			os.WriteFile(destPath, data, 0644)
		}
	}
}
`

	// Write main.go
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644); err != nil {
		fmt.Printf("[flang] Erro ao gerar main.go: %s\n", err)
		os.Exit(1)
	}

	// Find the flang module path for the replace directive
	flangModPath := getFlangModPath()

	// Generate go.mod
	goMod := "module flang-app\n\ngo 1.26\n\nrequire github.com/flaviokalleu/germanio v0.0.0\n\nreplace github.com/flaviokalleu/germanio => " + flangModPath + "\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		fmt.Printf("[flang] Erro ao gerar go.mod: %s\n", err)
		os.Exit(1)
	}

	// Build
	fmt.Printf("[flang] Compilando %s...\n", output)

	absOutput, _ := filepath.Abs(output)

	cmd := exec.Command("go", "build", "-o", absOutput, "-ldflags", "-s -w", ".")
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("[flang] Erro na compilacao: %s\n", err)
		os.Exit(1)
	}

	// Get file size
	info, _ := os.Stat(absOutput)
	sizeMB := float64(info.Size()) / (1024 * 1024)

	fmt.Printf("[flang] Build concluido: %s (%.1f MB)\n", output, sizeMB)
	fmt.Printf("[flang] Execute: ./%s [porta]\n", output)
}

func getFlangModPath() string {
	// Find the flang module path by looking for go.mod
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)

	// Walk up looking for go.mod with flang module
	for {
		modPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(modPath); err == nil {
			data, _ := os.ReadFile(modPath)
			if strings.Contains(string(data), "flavio/flang") || strings.Contains(string(data), "module") {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Fallback: try GOPATH
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, _ := os.UserHomeDir()
		gopath = filepath.Join(home, "go")
	}
	return filepath.Join(gopath, "src", "github.com", "flavio", "flang")
}

func cmdIDE(dir string, porta string) {
	ideServer := ide.Novo(dir, porta)

	// Open browser
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", fmt.Sprintf("http://localhost:%s", porta))
	case "darwin":
		cmd = exec.Command("open", fmt.Sprintf("http://localhost:%s", porta))
	default:
		cmd = exec.Command("xdg-open", fmt.Sprintf("http://localhost:%s", porta))
	}
	cmd.Start()

	if err := ideServer.Iniciar(); err != nil {
		fmt.Printf("[flang] ERRO IDE: %s\n", err)
		os.Exit(1)
	}
}
