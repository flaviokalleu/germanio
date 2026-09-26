<p align="center">
  <img src="assets/germanio.png" alt="Germanio — mascote cristalino azul com rosto de terminal" width="340">
</p>

<h1 align="center">Germanio</h1>

<p align="center">
  <strong>Simplíssima para começar. Difícil de usar errado.<br>Sem teto como direção de evolução.</strong>
</p>

<p align="center">
  <a href="https://github.com/flaviokalleu/germanio/actions/workflows/ci.yml"><img src="https://github.com/flaviokalleu/germanio/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <img src="https://img.shields.io/badge/Germanio-0.7.0--dev-00b8ff" alt="0.7.0-dev">
  <img src="https://img.shields.io/badge/c%C3%B3digo-.ge-00b8ff" alt="extensão .ge">
  <img src="https://img.shields.io/badge/CLI-ge-00b8ff" alt="CLI ge">
  <a href="LICENSE"><img src="https://img.shields.io/badge/licen%C3%A7a-MIT-blue" alt="MIT"></a>
</p>

<p align="center">
  <a href="#comece-em-minutos">Começar</a> ·
  <a href="#a-linguagem">Linguagem</a> ·
  <a href="#comandos-disponíveis">Comandos</a> ·
  <a href="SPEC.md">Especificação</a> ·
  <a href="docs/PHASE1.md">Entrega da fundação</a> ·
  <a href="docs/ROADMAP.md">Roadmap Germanio</a> ·
  <a href="#flang-continua-funcionando">Compatibilidade</a>
</p>

## Uma linguagem que começa com uma conversa

```ge
nome = pergunte "Qual seu nome?"
mostre "Olá {nome}"
```

Germanio é a evolução gradual do Flang para uma linguagem simples, determinística
e pensada para frontend e backend. Código natural continua sendo código:
**não existe LLM dentro do compilador**.

**Estado atual:** fundação `.ge` executável, com análise estática inicial e
interpretador AST em Go. A camada full-stack histórica continua disponível em
`.ge`. A integração full-stack na nova sintaxe `.ge` é uma próxima fase, não uma
funcionalidade já concluída. A [SPEC.md](SPEC.md) define o suporte real e os limites.
O [roadmap](docs/ROADMAP.md) apresenta as fases restantes e os critérios para
considerá-las concluídas.

## Comece em minutos

Requisito: **Go 1.26.1 ou superior compatível**, conforme `go.mod`.

```bash
git clone https://github.com/flaviokalleu/germanio.git
cd germanio
go build -o ge ./cmd/ge

./ge novo meu_programa
./ge rodar meu_programa/inicio.ge
```

No Windows, compile com `go build -o ge.exe ./cmd/ge` e execute `.\ge.exe`.
Para instalar a CLI a partir deste checkout: `go install ./cmd/ge`.
Os antigos binários e instaladores Flang **não são releases Germanio**.

Experimente os exemplos reais:

```bash
./ge rodar examples/germanio/ola.ge
./ge rodar examples/germanio/fundamentos.ge
./ge rodar examples/germanio/opcionais.ge
./ge rodar examples/germanio/genericos.ge
./ge rodar examples/germanio/numeros_genericos.ge
./ge testar examples/germanio
./ge rodar examples/germanio/imports.ge
./ge check examples/germanio/entrada.ge
./ge fmt examples/germanio --check
```

## A linguagem

### Valores seguros por padrão

```ge
nome = "Flavio"
variavel contador = 0
contador += 1

mostre "Olá {nome}"
mostre contador
```

Valores são imutáveis por padrão. Escreva `variavel` quando precisar alterar o
valor depois; `mut` continua aceito para os programas existentes. Tipos são inferidos, mas
anotações são permitidas: `idade: inteiro = 30`. Tipos iniciais: texto, inteiro,
decimal, bool, listas homogêneas e opcionais `T?`.
Por exemplo, `nomes: [texto?] = [nulo, "Ada"]` guarda valores
opcionais; `nulo` não entra em um parâmetro ou lista de tipo obrigatório.
Após `se nome != nulo`, um `nome: texto?` imutável pode ser usado como texto
dentro do ramo. Condições `e`/`ou` respeitam o curto circuito: `nome != nulo e
quantidade(nome) > 0` é seguro. Veja [opcionais.ge](examples/germanio/opcionais.ge).
Isso também funciona com `variavel` quando o ramo não altera o valor; se houver
reatribuição, mantenha a verificação e o uso em um trecho sem escrita.
Se todos os caminhos atribuírem um texto, o uso após o `se` também é seguro:

```ge
variavel saudacao: texto? = nulo
se verdadeiro
  saudacao = "Olá"
senao
  saudacao = "Oi"
mostre saudacao + "!"
```

Um caminho que ainda possa deixar `saudacao` como `nulo` produz erro na linha
de uso; loops que a alteram exigem nova verificação de presença.

### Funções sem cerimônia

```ge
dobro(x) = x * 2

somar(a: inteiro, b: inteiro) -> inteiro
  a + b

mostre dobro(21)
mostre somar(2, 3)
```

Não é preciso declarar `main`. Blocos usam dois espaços. Inferência de funções
sem parâmetros de tipo é monomórfica. Para uma função reutilizável com tipos
distintos, declare o parâmetro de tipo:

```ge
identidade<T>(valor: T) -> T
  valor

mostre identidade(7)
mostre identidade("olá")
```

Cada chamada infere `T` pelo argumento. Operações que só funcionam em tipos
específicos não são permitidas sobre um `T` sem restrição. Para contas, use a
restrição numérica básica:

```ge
somar<T: numero>(a: T, b: T) -> T
  a + b

mostre somar(2, 3)
mostre somar(1.5, 2.5)
```

`somar("a", "b")` é rejeitado antes de executar. Contratos estruturais,
outras restrições e especialização explícita continuam no roadmap.

### Coleções, condições e repetição

```ge
variavel total = 0
para valor em [10, 20, 30]
  total += valor

se total >= 50
  mostre "Total: {total}"
senao
  mostre "Abaixo de 50"
```

Também existem `enquanto`, `pare`, `continue`, retorno explícito `retorne`,
recursão, índices com verificação de limites e imports locais com namespace:

```ge
usa "matematica.ge"
mostre matematica.somar(2, 3)
```

Dentro de `matematica.ge`, `privado auxiliar(x) = x * 2` pode servir a outras
funções do módulo. Chamadas externas a `matematica.auxiliar(...)` recebem um
diagnóstico; funções sem `privado` compõem a API pública.

### Testes executáveis

```ge
usa "matematica.ge"

teste "soma"
  espera matematica.somar(2, 2) == 4
```

Salve como `matematica_teste.ge` e execute `ge testar` na pasta ou
`ge testar matematica_teste.ge`. Os testes têm variáveis locais isoladas.
`ge check` verifica suas condições sem executá-las; `ge rodar` executa apenas
o programa. Falhas apontam a linha `espera`. `ge testar --coverage` informa
instruções executadas e posições sem cobertura, incluindo funções importadas.
Detecção de corrida pela CLI ainda não está disponível.

### Erros que ajudam a corrigir

`"5" + 2` é erro, não uma coerção escondida. Use `numero("5") + 2`.
Condições exigem booleanos. Valores não utilizados, alterações de imutáveis,
imports circulares, divisão por zero e overflow possuem diagnósticos próprios.

Diagnósticos incluem código estável, arquivo, linha/coluna, trecho, motivo,
correção e exemplo. Consulte uma explicação sem conexão externa:

```bash
./ge explicar GE2004
```

### Base determinística da sintaxe natural

```ge
crie navbar escuro
crie botão escrito "Comprar" azul
  borda arredondada 16px
```

Esse subconjunto produz **HTML estático escapado no terminal**:

```bash
./ge rodar examples/germanio/interface.ge > interface.html
```

Ainda não há eventos, hover, responsividade ou conexão automática com backend
na sintaxe nova. Propriedades desconhecidas são rejeitadas. `45deg` em uma borda
produz GE4102, com orientação para usar comprimento, como `45px`.

## Comandos disponíveis

| Comando | Comportamento real |
| --- | --- |
| `ge rodar arquivo.ge` | Verifica e executa; usa `inicio.ge` se omitido |
| `ge check arquivo.ge` | Analisa sem executar entrada, banco ou servidor |
| `ge testar [arquivo-ou-pasta]` | Executa testes `.ge`; pastas buscam `*_teste.ge` |
| `ge testar --coverage` | Mede instruções executadas nos testes e nas funções |
| `ge fmt arquivo-ou-pasta` | Formatação inicial com preservação de comentários |
| `ge fmt --check` | Retorna erro se há diferenças; não escreve |
| `ge novo pasta` | Cria um programa; não sobrescreve pasta existente |
| `ge explicar GE2004` | Explica um diagnóstico conhecido |
| `ge versao` / `ge ajuda` | Versão e ajuda |
| `ge legado ...` | Acesso explícito à CLI Flang |

`ge build`, package manager, LSP, JIT, WASM, GPU e SIMD **não estão
implementados**. Não há comandos vazios simulando esses recursos. Os testes do
projeto são executados com Go.

## Flang continua funcionando

```bash
./ge check demo/plano/inicio.ge
./ge rodar demo/plano/inicio.ge 8080

# CLI histórica, com os comandos originais:
go build -o flang ./cmd/flang
./flang run demo/plano/inicio.ge
```

Foram preservados o parser/lexer multilíngue, o AST full-stack, o interpretador
legado, modelos, telas, temas, rotas, banco, autenticação e integrações. `.ge`
mantém sua semântica anterior; não recebe automaticamente as garantias `.ge`.
**Não basta renomear um arquivo `.ge` para `.ge`.**

Consulte a [documentação histórica](docs/FLANG_LEGACY.md). Os exemplos antigos
continuam em `examples/` e `demo/`; os novos estão em `examples/germanio/`.

## Desenvolvimento e validação

```bash
go test ./...
go vet ./...
go test -race ./compiler/... ./runtime/germanio ./tooling/...
go test ./compiler/parser -run '^$' -fuzz FuzzGermanioParser -fuzztime 15s
go build -o ge ./cmd/ge
```

O instalador Windows só é compilado com a tag `installer` e o payload preparado;
não deve bloquear testes normais. A CI verifica testes, builds, exemplos `.ge`,
formatter e exemplos `.ge`.

## Evolução com honestidade

O próximo foco é aprofundar tipos, opcionais, inferência, escopos e módulos.
Depois vêm a ponte segura cliente/servidor e a UI natural completa. Otimizações,
HIR/MIR, backends nativos e recursos avançados só serão marcados como disponíveis
quando funcionarem e tiverem testes.

Veja [SPEC.md](SPEC.md) e o [relatório da fundação](docs/PHASE1.md).

Licença [MIT](LICENSE).
