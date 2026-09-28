# Ruby (e Rails): estudo dirigido para o Germanio

**Data:** 2026-09-28. **Status:** pesquisa, sem força normativa (a norma é `docs/INTENCAO.md`).
**Pergunta principal:** o Rails é a referência mais próxima da filosofia do Germanio
(convenção no lugar de configuração, esquema e validações declarados no modelo). Onde as
convenções dele falharam, e o Germanio repete ou evita essas falhas?

## Fontes consultadas

Consultadas nesta sessão (WebFetch):

- [The Rails Doctrine](https://rubyonrails.org/doctrine)
- [Rails Guides: Action Controller, Strong Parameters](https://guides.rubyonrails.org/action_controller_overview.html)
- [Rails Guides: Active Record Validations](https://guides.rubyonrails.org/active_record_validations.html)
- [Rails Guides: Active Record Migrations](https://guides.rubyonrails.org/active_record_migrations.html)
- [GitHub Blog, mar/2012: Public Key Security Vulnerability and Mitigation](https://github.blog/news-insights/the-library/public-key-security-vulnerability-and-mitigation/)
- [RubyGems Guides: Dependency management (Gemfile.lock)](https://guides.rubygems.org/dependency_management/)

Do conhecimento prévio, **não verificado nesta sessão**: `method_missing` e
`respond_to_missing?` como base de finders dinâmicos antigos (`find_by_nome_and_email`),
refinements (Ruby 2.0) como resposta parcial ao monkey patching, Zeitwerk (autoloader por
convenção de nome de arquivo desde Rails 6), RBS/Sorbet/Steep como tipagem opcional, Prism
como parser oficial desde Ruby 3.4, e o `attr_accessible` do Rails 3 que o Strong Parameters
substituiu no Rails 4.

Código do Germanio conferido: `runtime/servidor/intencao.go:378-418` (`writable`),
`runtime/banco/banco.go:217,262-277` (UNIQUE e ALTER), `compiler/ast/ast.go:331-334`,
`cli/modelos/organizado/`, `versao/versao.go`.

---

## Matriz (compacta)

| Item | Ruby / Rails | Relevância para o Germanio |
|---|---|---|
| Objetivo original | Ruby: linguagem orientada a objetos para "felicidade do programador" (Matz, 1995). Rails: framework web integrado (DHH, 2004) | O objetivo do Germanio é o mesmo, com outro público |
| Filosofia | "Optimize for programmer happiness", "Convention over Configuration", "The menu is omakase", "Provide sharp knives", "Value integrated systems" ([Doctrine](https://rubyonrails.org/doctrine)) | Convenção e sistema integrado: o Germanio concorda. Facas afiadas: o Germanio discorda (público leigo) |
| Sintaxe | Ver seção "Sintaxe" | — |
| Gramática / parser | Gramática famosa por ser difícil (parse.y com yacc); Prism como parser novo e reutilizável (não verificado) | Exemplo do risco "vários parsers" até haver um oficial |
| Tipos | Dinâmicos, duck typing; RBS/Sorbet opcionais (não verificado) | — |
| Runtime | YARV, GVL; Ractors e fibers (não verificado) | não se aplica |
| Package manager | RubyGems + Bundler; `Gemfile.lock` é o snapshot das versões exatas "da última vez que você sabe que tudo funcionou"; aplicações commitam, bibliotecas não; `bundle update gem` conservador ([RubyGems](https://guides.rubygems.org/dependency_management/)) | Ver INVESTIGAR (versão da linguagem) |
| Formatter / linter | RuboCop (terceiro, configurável); sem formatter oficial único (não verificado) | Contraexemplo (P13) |
| LSP | ruby-lsp (Shopify) (não verificado) | — |
| Diagnostics | Erros de runtime; muita coisa só aparece ao executar | O Germanio checa antes (`ge check`) |
| Evolution process | Matz decide; propostas no bugs.ruby-lang.org (não verificado) | — |
| Backward compatibility | Ruby 1.8 → 1.9 foi traumático (encoding); Rails quebra entre majors com guias de upgrade (não verificado) | — |
| Principais acertos | Convenção sobre configuração; validações e associações declarativas no modelo; migrations reversíveis e `schema.rb` como estado derivado; lockfile | — |
| Principais problemas | Mágica implícita; validações que só valem em alguns métodos; `uniqueness` sem índice tem condição de corrida; mass assignment permissivo até 2012 | — |
| Complexidade acumulada | Callbacks de modelo, concerns, metaprogramação em gems | — |

---

## Sintaxe

| Pergunta | Ruby / Rails | Germanio hoje (IMPLEMENTADO) | Comparação |
|---|---|---|---|
| Dados | Classe `ActiveRecord` vazia; colunas vêm do banco em tempo de execução; tabela pelo plural do nome ([Doctrine](https://rubyonrails.org/doctrine)) | `cada X tem` + campos; tabela derivada do `.ge` | **Direção oposta**: no Rails o banco é a fonte e o modelo lê as colunas; no Germanio o `.ge` é a fonte e o banco é derivado. Ler um modelo Rails não mostra seus campos; ler um `.ge` mostra |
| Funções | Métodos; blocos; parênteses opcionais | Nível 3 | — |
| Módulos / imports | `require`; no Rails, autoload por nome de arquivo (Zeitwerk, não verificado) | `importar "backend"` carrega a pasta inteira em ordem alfabética (`INTENCAO.md`, "Organização do projeto") | Mesma convenção; o Germanio precisa garantir que a ordem não muda significado (fusão comutativa, A4) |
| Relações | `has_many :pedidos`, `belongs_to :cliente` | `cliente tem pedidos`, `pedido pertence a cliente` | Quase idêntico; o Germanio vem da mesma linhagem. O Rails exige a migração com a FK à parte |
| Estado | Gems (AASM) ou enum | `começa`, `pode` nativos | Nativo no Germanio |
| Fluxo | `if`/`unless`, modificadores pós-fixados | Declarativo | — |
| Erros | Exceções; `save` devolve `false` e `errors` por campo | Erros por campo, transação | Mesmo formato de erro por campo |
| Concorrência | GVL; threads; o Rails isola por requisição | Goroutines | — |
| Tipos | Dinâmicos | Pelo nome | A inferência pelo nome do Germanio é uma convenção à Rails aplicada a tipos |
| Null | `nil`; `presence: true` opt-in | `obrigatório` opt-in | Mesmo padrão (ver `kotlin.md`, INVESTIGAR) |
| Organização | `app/models`, `app/controllers`, `db/migrate` fixos | `backend/`, `frontend/`, `integracoes/` fixos | Mesmo princípio |
| Boilerplate | Scaffolds, geradores, convenções | Derivação total, sem geradores | O Germanio não gera código para o autor manter; o Rails gera e o autor passa a mantê-lo |
| Legibilidade em projetos grandes | Cai com callbacks, concerns e metaprogramação: o comportamento de um modelo está espalhado | `ge explain` junta cada fato com origem | Ver abaixo |

### O custo da mágica implícita

A Doctrine assume o custo: facas afiadas "can be misused", e o ambiente é "for chefs and
those who wish to become chefs" ([Doctrine](https://rubyonrails.org/doctrine)). Três formas
dessa mágica, e a posição do Germanio:

1. **`method_missing` / finders dinâmicos** (não verificado nesta sessão): métodos que não
   existem em lugar nenhum do código; nenhuma ferramenta estática os encontra. Germanio:
   toda frase é resolvida contra tabelas fechadas; não há nome que surja em execução.
2. **Monkey patching**: qualquer gem pode alterar `String` ou o próprio Rails. Germanio:
   `integracoes/` não pode declarar dados, permissões ou páginas (`INTENCAO.md`); manter
   essa proibição quando surgir reutilização entre projetos.
3. **Validações que dependem do caminho**: `create`/`save`/`update` validam, mas
   `update_column`, `update_all`, `insert_all`, `upsert`, `toggle!`, `touch` e
   `save(validate: false)` pulam ([Validations](https://guides.rubyonrails.org/active_record_validations.html)).
   Germanio: validação no ponto único de escrita (`banco.Validar`); ver `java.md` para os
   caminhos a auditar.

### Mass assignment: o incidente do GitHub (2012) e o Germanio

Em março de 2012 um usuário usou o formulário de chave pública do GitHub para associar a
própria chave à organização Rails e fazer push; a causa foi "a mass-assignment
vulnerability" ([GitHub Blog](https://github.blog/news-insights/the-library/public-key-security-vulnerability-and-mitigation/)).
A resposta do Rails foi Strong Parameters: nada da requisição entra em `create`/`update` sem
ser listado; tentar levanta `ForbiddenAttributesError` ([Strong Parameters](https://guides.rubyonrails.org/action_controller_overview.html)).

Germanio, conferido em `runtime/servidor/intencao.go:378-418` (`writable`):

- **Acerto (IMPLEMENTADO):** a lista permitida é implícita e fechada: só entram campos
  declarados; campos do runtime (`System: true`: `estado`, carimbos, `endereco`) e segredos
  nunca são aceitos; o dono é sempre quem cria (só administrador age em nome de outro), e
  autoria não muda por edição. A classe de bug do GitHub (trocar a FK do dono) está coberta
  por construção, sem que o autor liste nada.
- **Lacuna (verificada):** fora isso, **todo campo declarado é gravável por quem pode
  editar o registro**. A única exceção por nome é `admin`/`papel` na entidade de login
  (`intencao.go:393`), uma regra fixa no core por nome de campo. Um campo como `saldo`,
  `verificado`, `desconto` ou `aprovado`, declarado com `tem`, pode ser enviado por quem
  edita. `imutável` só impede depois da criação; `oculto` e `privado` são sobre leitura. Não
  existe hoje frase para "só o sistema / só administrador / só o papel X altera este campo".
  É o mesmo padrão permissivo que o Rails abandonou em 2012, restrito aos campos declarados.

### `uniqueness` sem índice e o `único` do Germanio

O guia do Rails avisa que `uniqueness` "does not create a uniqueness constraint in the
database" e duas conexões podem gravar o mesmo valor; é preciso índice único
([Validations](https://guides.rubyonrails.org/active_record_validations.html)). No Germanio,
`único` vira `UNIQUE` na coluna **só quando a tabela é criada** (`runtime/banco/banco.go:217`);
um campo `único` acrescentado depois passa pelo `ALTER TABLE … ADD COLUMN` sem `UNIQUE`
(`banco.go:273-276`), e marcar como `único` um campo já existente não altera nada. Nesses
casos só resta a checagem da aplicação por consulta (`runtime/interpreter/schema.go:251`,
"já está em uso"), com a mesma corrida que o Rails documenta.

### Migrations

O Rails gera arquivos de migração por nome (`AddXToY`), mantém `db/schema.rb` como estado
derivado mais confiável que reexecutar migrações antigas, marca operações irreversíveis e
proíbe editar migração já commitada ([Migrations](https://guides.rubyonrails.org/active_record_migrations.html)).
O Germanio não deve copiar os arquivos (é boilerplate que o autor passa a manter), mas o
`schema.rb` é o mesmo conceito do snapshot proposto em `csharp.md`.

---

## Para o Germanio

**ADOTAR — permissão de escrita por campo, com padrão seguro para campos sensíveis.**
Problema: todo campo declarado é gravável por quem edita (`runtime/servidor/intencao.go:378-418`);
a proteção de `admin`/`papel` é caso especial por nome (`:393`). Proposta: uma frase de
domínio para "quem altera" um campo (por exemplo `somente administrador altera desconto`,
sintaxe a decidir pelas regras de `INTENCAO.md`), reaproveitando papéis e a regra geral de
`somente`; a proteção de `admin`/`papel` passa a ser caso dessa regra, não `if` por nome.
Remove da cabeça: auditar cada formulário à procura do campo que ninguém deveria mandar.
Lição de [GitHub 2012](https://github.blog/news-insights/the-library/public-key-security-vulnerability-and-mitigation/).

**ADOTAR — `único` sempre com índice no banco, inclusive em campo acrescentado depois.**
Problema: `runtime/banco/banco.go:273-276` não cria o índice. Proposta: `CREATE UNIQUE INDEX
IF NOT EXISTS` para todo campo `único`, e falha explicada se há duplicatas já gravadas. Remove
da cabeça: condição de corrida que o autor nem sabe que existe.

**ADAPTAR — convenção sobre configuração, com a explicação que o Rails não tem.** O Germanio
já deriva tabela, rota e tipo por convenção; a diferença a manter é que toda convenção
aplicada aparece em `ge explain` com origem e motivo, e declaração explícita vence
(`INTENCAO.md`, "Tipo pelo nome"). Arquivo: `tooling/explicar/`.

**EVITAR — "facas afiadas"**: `method_missing`, monkey patching, callbacks encadeados e
extensão do core por pacotes. O público do Germanio não é "chef". Escape hatches ficam no
nível 3, sem poder contornar políticas (`INTENCAO.md`).

**EVITAR — geradores de código que o autor passa a manter** (scaffold, arquivos de migração).

**INVESTIGAR — o projeto registrar a versão do Germanio com que funcionou (lockfile da
linguagem).** O `Gemfile.lock` existe para que a aplicação seja "a single package of both
your own code and the third-party code it ran the last time you know for sure that
everything worked" ([RubyGems](https://guides.rubygems.org/dependency_management/)). No
Germanio a "dependência" principal é o próprio runtime; os modelos de `germanio init`
(`cli/modelos/organizado/`) não registram versão (verificado por inspeção). Com o formatter
versionado (P13) e remoções com reescrita (`javascript.md`), a versão da linguagem no
`app.ge` (como a linha `go` do `go.mod`) diria qual comportamento o projeto espera.
