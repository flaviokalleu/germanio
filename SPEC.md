# Germanio — especificação executável da fundação

Status: 0.7.0-dev. Este documento distingue o subconjunto implementado da visão
de longo prazo. Arquivos `.ge` usam a gramática estrita abaixo; arquivos `.fg`
continuam no compilador/runtime Flang. Trocar apenas a extensão não migra código.

## Princípios

1. Previsibilidade; uma maneira principal de fazer cada coisa.
2. Explícito quando houver ambiguidade, inferido quando for seguro.
3. Remover código não utilizado; comportamento óbvio para código válido.
4. Diagnósticos ensinam a correção; formatação pertence à linguagem.
5. Complexidade progressiva: esconder complexidade, nunca capacidade.
6. Código simples deve ser rápido; otimizações futuras precisam de evidência.
7. Frontend e backend pertencem ao mesmo ecossistema, com isolamento seguro.
8. Segurança padrão; nenhuma interpretação probabilística na semântica.

## Léxico

UTF-8, identificadores Unicode sensíveis a maiúsculas. Palavras reservadas
canônicas em português: `mostre`, `pergunte`, `variavel`, `const`, `se`, `senao`,
`enquanto`, `para`, `em`, `retorne`, `pare`, `continue`, `usa`, `crie`, `e`, `ou`,
`nao`, `verdadeiro`, `falso`, `nulo`.
`mut` é a grafia curta de `variavel` e continua aceita por compatibilidade.

Uma instrução por linha. Dois espaços por nível de bloco; tabulações são erro.
Linhas vazias e comentários `#` ou `//` são ignorados pelo parser e preservados
pelo formatter. Strings usam aspas duplas e escapes `\n`, `\t`, `\"`, `\\`.
Interpolação aceita identificadores simples: `"Olá {nome}"`.

Literais inteiros decimais não negativos até 9223372036854775807; negativos são
expressões unárias, com reconhecimento do mínimo -9223372036854775808.
Valores inteiros de execução têm 64 bits com sinal, com
overflow verificado. Literais decimais usam ponto; notação exponencial não faz
parte desta versão. Decimais em execução são IEEE-754 de 64 bits, sempre finitos.

## Gramática resumida

```ebnf
programa    = { import | funcao | instrucao } ;
import      = "usa" string ;
funcao      = nome "(" [param {"," param}] ")" ["->" tipo]
              ("=" expressao | NOVALINHA bloco) ;
param       = nome [":" tipo] ;
tipo        = ("texto" | "inteiro" | "decimal" | "bool" | "[" tipo "]") ["?"] ;
declaracao  = ["variavel" | "mut" | "const"] nome [":" tipo] "=" expressao ;
atribuicao  = nome ("=" | "+=" | "-=") expressao ;
condicional = "se" expressao NOVALINHA bloco
              ["senao" NOVALINHA bloco] ;
repeticao   = "enquanto" expressao NOVALINHA bloco
            | "para" nome "em" expressao NOVALINHA bloco ;
saida       = "mostre" expressao ;
retorno     = "retorne" expressao ;
lista       = "[" [expressao {"," expressao}] "]" ;
```

Blocos exigem ao menos uma instrução. Não há `main` obrigatório, ponto e vírgula,
chaves de bloco, aliases `let/var/fn/function`, nem execução de trechos desconhecidos.
`pare` e `continue` pertencem a loops. `retorne` pertence a funções.

## Variáveis e tipos

```ge
nome = "Flavio"
idade: inteiro = 30
variavel contador = 0
contador += 1
const PI = 3.14159
telefone: texto? = nulo
valores: [inteiro] = []
```

Declaração é imutável por padrão. `=` posterior altera somente variável declarada
com `variavel` ou `mut`;
não redeclara nem cria shadowing. `const` só no topo do módulo. Alterações mantêm
o tipo inferido/declarado. Locais, parâmetros e imports sem leitura são erros;
`_ = expressao` descarta explicitamente, sem criar uma variável `_`.

Inferência inicial é **monomórfica** por função/módulo: não equivale a generics.
Tipos suportados: texto, inteiro, decimal, bool, listas homogêneas e opcionais,
inclusive composições `[texto?]`. Listas vazias podem usar anotação explícita.
Uma lista que mistura `nulo` e texto infere `[texto?]`; uma lista de `texto`
pode ser atribuída a `[texto?]`, mas a conversão inversa é rejeitada.
Mapas, dinheiro exato, byte, datas, horas e larguras numéricas adicionais ainda
não fazem parte do núcleo `.ge` (tipos Flang não foram removidos do modo `.fg`).

Só `T?` aceita `nulo`. Não há truthy/falsy. Comparações `==` e `!=` são estritas;
listas comparam estruturalmente. Opcionais podem ser mostrados, atribuídos,
passados a funções e comparados com `nulo`. `se valor != nulo` refina um valor
**imutável** `T?` para `T` no ramo verdadeiro; `se valor == nulo` refina no
`senao`. A comparação invertida e `nao` também funcionam. `e` verifica seu
segundo operando quando o primeiro é verdadeiro; `ou`, quando é falso. Somente
presenças garantidas em **todos** os caminhos alcançam o bloco seguinte. Isso
permite `se valor != nulo e quantidade(valor) > 0`. Uma condição verdadeira
também refina o corpo de `enquanto` quando o valor é imutável. Se o ramo oposto
termina com `retorne`, o refinamento vale até o fim do bloco para valores
imutáveis. Variáveis `variavel`/`mut` também refinam dentro de um ramo se
nenhum bloco interno puder reatribuir a mesma variável. Escritas diretas no
ramo invalidam o refinamento naquela linha: leituras anteriores à escrita
são aceitas; leituras posteriores exigem
novo teste. O refinamento não continua após o `se` para variáveis mutáveis.
Não há desreferência implícita de opcional.

`+` soma números ou concatena dois textos; `-`, `*`, `/`, `%` são numéricos.
Mistura aritmética de inteiro e decimal promove para decimal; isso não permite
atribuir decimal a inteiro. `/` devolve decimal. `%` exige inteiros.
`"5" + 2` é erro; `numero("5") + 2` é válido. Igualdade não promove tipos.
`e`/`ou` são booleanos com curto-circuito. Precedência, da menor à maior:
`ou`, `e`, igualdade, ordenação, `+/-`, `*/%`, unários, chamada/indexação.
Índices são inteiros, começam em zero e têm limites verificados em execução.

## Funções e escopos

```ge
dobro(x) = x * 2

somar(a: inteiro, b: inteiro) -> inteiro
  a + b
```

A última expressão simples de um corpo é retorno implícito. Retornos condicionais
usam `retorne`; funções com retorno precisam cobrir todos os caminhos. Funções
são declaradas no topo do módulo, podem chamar outras funções e ser recursivas.
Nesta fundação, dados externos entram por parâmetros: funções não capturam
variáveis de topo/closures. Isso evita dependência de ordem e estado oculto;
escopos mais avançados e contratos ficam para a Fase 2.

Built-ins: `pergunte "Mensagem"` (texto, stdin), `texto(valor)`, `numero(valor)`
e `decimal(valor)` (decimal), `inteiro(valor)` (conversão sem perda fracionária),
`quantidade(texto_ou_lista)` (inteiro; textos contam pontos de código Unicode).
Conversão inválida, valores não finitos e overflow produzem diagnóstico.

## Módulos

```ge
usa "matematica.ge"
mostre matematica.somar(2, 3)
```

Imports locais são nomes de arquivo `.ge` entre aspas; o nome-base forma o
namespace. Exportam funções, não estado. O código de topo do arquivo importado
é verificado, mas não executado. Arquivos dentro de subdiretórios são permitidos;
caminhos fora da pasta do entrypoint, inclusive por symlinks, são bloqueados.
Ciclos, módulos duplicados, funções inexistentes e imports não utilizados são
erros. Arquivos ausentes, pastas com extensão `.ge` e tentativas de escapar da
raiz do projeto são diagnosticados na instrução `usa`. Não há download
automático, stdlib remota, manifesto ou lockfile ainda.

## Fundação de UI natural

```ge
crie navbar escuro
crie botão escrito "Comprar" azul
  borda arredondada 16px
```

Este subconjunto gera HTML estático no stdout em `ge rodar`, que pode ser
redirecionado para um arquivo. Não abre servidor nem conecta um botão a uma ação.
Texto é escapado; cores vêm da tabela existente do Flang; bordas aceitam `px` ou
`rem`. Unidade como `45deg` produz GE4102. Propriedades desconhecidas são erros,
não ignoradas. Hover, menus compostos, eventos, responsividade e RPC são Fase 3.

O AST mantém os modelos/screens/rotas/temas legados, adiciona metadados de origem
e domínio (`shared`) para módulos `.ge` e um nó estático de UI exclusivamente
cliente, que não recebe expressões de servidor. Isso não é um analisador completo
de segredos nem uma ponte cliente/servidor: esses recursos não estão implementados.

## Execução, diagnósticos e ferramentas

O backend inicial é um interpretador AST escrito em Go, não compilação nativa de
`.ge`. `ge rodar` analisa todo o módulo antes de executar. `ge check` não executa
código, não lê stdin nem inicia banco/servidor. Loop/recursão são limitados por
orçamento de 1 milhão de passos de AST e 256 chamadas aninhadas por execução.

Diagnósticos possuem código estável, arquivo, linha/coluna, trecho, motivo e
correção. `ge explicar GE2004` consulta o catálogo sem conexão externa.
Famílias reservadas: GE1 sintaxe; GE2 tipos/lógica; GE3 módulos/dados; GE4 UI;
GE5 backend/rede; GE6 fronteira cliente/servidor; GE7 concorrência; GE8 performance;
GE9 segurança/limites. Reservar uma família não implementa os seus recursos.

`ge fmt` preserva comentários, normaliza espaços e adiciona newline final.
É idempotente, recusa sintaxe inválida e não tenta adivinhar blocos mal indentados.
`ge fmt --check` não modifica arquivos e retorna 1 se houver diferenças.
`ge novo pasta` cria `inicio.ge`, sem sobrescrever uma pasta existente.

## Próximas fases (não implementadas)

- Fase 2: inferência mais completa, refinamento de opcionais, módulos avançados,
  contratos, generics, erros tipados, tipos especializados e escopos avançados.
- Fase 3: UI completa, frontend/backend `.ge`, modelos, RPC seguro, banco e auth.
- Fase 4: manifesto, dependências, stdlib, LSP/Quick Fix, formatter/linter completos,
  comandos nativos de testes, benchmark, fuzzing e profiling.
- Fase 5: HIR/MIR, otimização, compilação incremental e backends compilados.
- Fase 6: FFI, WASM/JIT, SIMD/GPU, memória avançada e metaprogramação.

Nenhuma pasta/flag vazia é apresentada como implementação desses recursos.
