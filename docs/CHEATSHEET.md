# Germanio v0.5.0 - Cheatsheet / Referencia Rapida

## CLI

```bash
germanio run arquivo.ge [porta]     # Executa o app (padrao porta 8080)
germanio check arquivo.ge           # Valida sintaxe sem executar
germanio new nome                   # Cria novo projeto
germanio init                       # Inicializa projeto no diretorio atual
germanio build arquivo.ge           # Compila em executavel standalone
germanio docker arquivo.ge          # Gera Dockerfile e docker-compose.yml
germanio version                    # Mostra versao
germanio help                       # Ajuda
```

## Estrutura Minima

```
sistema nome
dados
  modelo
    campo: tipo
telas
  tela nome
    titulo "Titulo"
    lista modelo
      mostrar campo
    botao azul
      texto "Acao"
eventos
  quando clicar "Acao"
    criar modelo
```

## Tipos de Dados

```
texto / text              numero / number          dinheiro / money
email                     telefone / phone         status
data / date               booleano / boolean       senha / password
imagem / image            arquivo / file           upload
link                      texto_longo / long_text   enum
```

### Enum com Valores

```
status: enum(ativo, inativo, pendente)
categoria: enum(A, B, C)
```

## Modificadores

```
campo: tipo obrigatorio / required
campo: tipo unico / unique
campo: tipo padrao "valor" / default "value"
campo: tipo indice / index
campo: tipo pertence_a modelo / belongs_to model
```

## Relacionamentos

```
dados
  empresa
    nome: texto
    funcionarios: tem_muitos funcionario        # 1:N
    projetos: muitos_para_muitos projeto         # N:N

  funcionario
    nome: texto
    empresa_id: numero pertence_a empresa        # FK (gera dropdown)
```

## Telas

```
tela nome / screen name
  titulo "X" / title "X"
  publico / public                # sem autenticacao
  requer admin / requires admin   # exige role
  lista modelo / list model
    mostrar campo / show field
  botao azul / button blue
    texto "X" / text "X"
```

## Eventos

```
quando clicar "X" / when click "X"
  criar modelo / create model

quando criar modelo / when create model
  notificar usuario
```

## Tema - Presets

```
tema moderno          # glassmorphism, gradientes, sombras
tema simples          # flat, clean, minimalista
tema elegante         # tipografia refinada, cores suaves
tema corporativo      # profissional, sidebar escura
tema claro            # fundo branco, cores claras
```

## Tema - Cores por Nome

```
tema
  cor primaria azul         # sem hex necessario
  cor secundaria verde
  cor destaque amarelo
  cor sidebar escuro
```

Nomes disponiveis: `azul`, `verde`, `vermelho`, `roxo`, `laranja`, `amarelo`, `rosa`, `escuro`, `claro`

## Tema - Cores por Hex

```
tema
  cor primaria "#6366f1"
  cor secundaria "#8b5cf6"
  cor destaque "#f59e0b"
  cor sidebar "#1e1b4b"
  escuro
```

## Tema - Estilos Visuais

```
tema
  estilo glassmorphism      # vidro translucido
  estilo flat               # sem sombras
  estilo neumorphism        # relevo suave
  estilo minimal            # espacamento amplo
```

## Rotas Customizadas

```
rotas
  GET /api/relatorio/vendas
    consultar "SELECT SUM(valor) as total FROM pedido WHERE status = 'pago'"

  POST /api/acao/notificar
    corpo "mensagem"
    executar "INSERT INTO log (acao) VALUES ('notificacao')"
```

## Paginas Customizadas

```
paginas
  pagina sobre
    caminho "/sobre"
    html """
      <h1>Sobre nos</h1>
      <p>Descricao da empresa</p>
    """
```

## Sidebar Customizada

```
sidebar
  item "Dashboard" icone "home" link "/"
  item "Produtos" icone "box" link "/produtos"
  item "Relatorios" icone "chart" link "/relatorios"
  separador
  item "Configuracoes" icone "settings" link "/config"
```

## Banco

```
banco / database
  driver: sqlite / mysql / postgres
  host: "localhost"
  porta / port: "5432"
  nome / name: "db"
  usuario / user: "user"
  senha / password: "pass"
```

## Imports

```
importar "arquivo.ge" / import "file.ge"
importar dados de "x.ge" / import models from "x.ge"
```

## Logica e Scripting

```
logica
  validar campo obrigatorio / validate field required
  validar preco maior 0

  se campo igual "valor" / if field equals "value"
    mudar cor verde / change color green
```

### Built-in Functions (Scripting)

```
# Texto
tamanho("texto")                 # 5
maiusculo("abc")                 # "ABC"
minusculo("ABC")                 # "abc"
substituir("ab", "a", "x")      # "xb"
cortar("  ab  ")                 # "ab"
comeca_com("abc", "a")           # verdadeiro
termina_com("abc", "c")          # verdadeiro
substring("abcde", 1, 3)        # "bcd"

# Arrays
tamanho([1,2,3])                 # 3
adicionar([1,2], 3)              # [1,2,3]
remover([1,2,3], 1)              # [1,3]
reverter([1,2,3])                # [3,2,1]

# Objetos
chaves({"a":1})                  # ["a"]
valores({"a":1})                 # [1]
json({"a":1})                    # '{"a":1}'

# Data
formato_data(agora(), "02/01/2006")

# Matematica
potencia(2, 10)                  # 1024
raiz(144)                        # 12

# HTTP
chamar("GET", "https://api.com/dados")
```

### Array Indexing

```
arr = [10, 20, 30]
arr[0]              # 10
obj.campo[0]        # primeiro elemento de obj.campo
```

## Async / Paralelismo

```
paralelo(func1, func2, func3)       # executa em paralelo
esperar(promessa)                     # aguarda resultado
timeout(funcao, 5000)                # timeout em ms
chamar_async("GET", "https://...")    # HTTP async
consultar_paralelo(q1, q2, q3)       # queries em paralelo
```

## Graficos (Charts)

```
grafico vendas
  tipo: barra / pizza / doughnut
  dados: pedido
  campo: valor
  agrupar: status
```

## Integracoes

### WhatsApp

```
integracoes
  whatsapp
    quando criar modelo
      enviar mensagem para campo
        texto "Msg {var}"
```

### Email

```
integracoes
  email
    host: "smtp.gmail.com"
    porta: "587"
    usuario: "email@gmail.com"
    senha: "apppassword"
```

### Cron

```
integracoes
  cron
    cada 1 hora
      chamar "https://api.com/sync"
```

## Autenticacao

```
autenticacao
  modelo: usuario
  campo_login: email
  campo_senha: senha
  roles: admin, usuario
```

## API REST (automatica)

```
GET    /api/{modelo}                 Listar
GET    /api/{modelo}/{id}            Buscar
POST   /api/{modelo}                 Criar
PUT    /api/{modelo}/{id}            Atualizar
DELETE /api/{modelo}/{id}            Deletar
GET    /api/{modelo}/{id}/{relacao}  Expandir relacionamento
GET    /api/{modelo}/export/csv      Exportar CSV
GET    /api/{modelo}/export/json     Exportar JSON
```

## WebSocket

```
ws://localhost:8080/ws
```

## 20 Idiomas Suportados

Portugues, Ingles, Espanhol, Frances, Alemao, Italiano, Chines, Japones, Coreano, Arabe, Hindi, Bengali, Russo, Indonesio, Turco, Vietnamita, Polones, Holandes, Tailandes, Suaili

Exemplo em Espanhol:
```
sistema mi_tienda
datos
  producto
    nombre: texto obligatorio
```
