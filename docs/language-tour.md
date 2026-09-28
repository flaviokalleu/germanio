# A tour of the Germanio language

Germanio programs answer four questions: **what exists, who may do what, what happens and
what appears**. This tour shows each part with a small complete program; every program here
passes `ge check`, and you can run it with `ge run file.ge`. The exact rules are in the
[specification](INTENCAO.md) (Portuguese), which is normative: if this tour and the
specification disagree, the specification wins.

Keywords are Portuguese. A short glossary is at the end.

## The one rule: what is below belongs to what is above

A line indented under another line belongs to it. Indentation is 4 spaces per level
(`ge fmt` writes it for you; tabs are an error). A data block starts with the data's name at
the left margin; under it come *aspects* such as `tem` (has), `pode` (can) and `acesso`
(access); under each aspect come its items.

```text
chamados              ← the data
    começa aberto     ← an aspect with its value on the same line
    pode              ← an aspect
        fechar        ← an item of that aspect
```

## Data and fields

```ge
crie sistema Agenda

clientes
    tem
        nome obrigatório
        email obrigatório e único
        telefone
        cidade
        foto
        aniversario data
    acesso
        todos
            ver
            criar
            editar
    permita
        pesquisar
        filtrar por cidade
```

`crie sistema Agenda` names the application. `clientes` (customers) is kept in the database;
each line under `tem` is a field. The type comes from the name when you do not give one:
`email` is validated as an e-mail and stored in lower case, `telefone` is a phone number,
`foto` is an image, everything else is text; `aniversario data` says explicitly that it is a
date. Modifiers such as `obrigatório` (required), `único` (unique), `até 120` (at most 120
characters) and `min 8` are checked on every create and edit, and all errors of a form come
back together, per field.

`ge explain clientes agenda.ge` prints the fields with their types, including the ones
Germanio adds (an id and the creation and update times).

## Relations

```ge
crie sistema Escola

turmas
    tem
        nome obrigatório
        alunos
    acesso
        todos
            administrar

alunos
    tem
        nome obrigatório
        email obrigatório e único
```

`turmas tem alunos` (classes have students) is a relation, not a field: each student belongs
to a class, and deleting a class deletes what belongs to it. You never write an id; Germanio
adds the reference and validates it. `pertence a` (belongs to) says the same thing from the
other side. `administrar` (administer) is a shortcut for seeing, editing and deleting the
class and managing what belongs to it.

## States and actions

```ge
crie sistema Suporte

chamados
    tem
        assunto obrigatório
        descrição
    começa aberto
    pode
        fechar
        reabrir
    acesso
        todos
            ver
            criar
            fechar
            reabrir
```

A ticket starts open (`começa aberto`) and can be closed (`fechar`) and reopened (`reabrir`).
Germanio creates the state field and the actions, and closing records when and by whom.
States change only through actions: editing a ticket never changes its state.

## Blocks and flat phrases say the same thing

The same program can be written as standalone phrases, which is convenient for a single fact:

```ge
crie sistema Suporte

tenha chamados

cada chamado tem
    assunto obrigatório
    descrição

chamado começa aberto
chamado pode fechar
chamado pode reabrir

todos podem ver chamados
todos podem criar chamados
todos podem fechar chamados
todos podem reabrir chamados
```

Both forms produce exactly the same facts, and the test suite proves it for every aspect.
`ge explain` shows each fact as its flat phrase together with where it was written. Use a
block when a piece of data has several aspects, and a phrase for a fact on its own.

## People, roles and permissions

```ge
crie sistema Clube

usuarios
    tem
        nome obrigatório
        email obrigatório e único
        senha min 8
    acesso
        usuario
            editar seu perfil

tenha login
tenha cadastro
login usa email

tenha papeis
    membro 10
    diretor 50

clubes
    tem
        nome obrigatório
        membros com papel
    acesso
        usuario
            criar
        membro
            ver
        diretor
            editar
            excluir

quem cria clube vira diretor
todo clube precisa ter pelo menos um diretor
```

`tenha login` and `tenha cadastro` give the application login and sign-up pages, sessions,
CSRF protection, password hashing (bcrypt) and, by default, a 10-minute lock after 10 wrong
passwords (`login bloqueia após 3 tentativas por 5 minutos` changes the numbers); the
password is never returned by the API. `usuario` (user) in `acesso` means any logged-in person.
`tenha papeis` declares roles from lowest to highest: what a `membro` may do, a `diretor` may
do too. `membros com papel` gives each club its own members, so the roles apply per club.
The last two lines are rules the runtime enforces, even under concurrent requests: whoever
creates a club becomes its director, and nobody can remove or demote the last director.

Records a person may not see answer "not found", never "forbidden", so their existence does
not leak. `seus`/`suas` (their own) restricts an action to the records that belong to the
person, as in the [to-do example](../examples/todo).

## Pages and integration

```ge
crie sistema Loja

produtos
    tem
        nome obrigatório
        preco obrigatório
        estoque começa com 0
    acesso
        todos
            ver
        somente administrador
            criar
            editar
            excluir

página Produtos
    mostre produtos
    permita
        pesquisar
        criar
        editar
        excluir

disponibilize produtos para integração
```

`página Produtos` is a page that shows (`mostre`) the products and offers search and the
create, edit and delete actions. The page does not repeat the fields or check roles: a
visitor sees the list and the search, and only the administrator sees the buttons that
change products, because that is what `acesso` says. The HTML is rendered on the server, with
every value escaped.

`disponibilize produtos para integração` publishes the products to other systems as a REST
API, with the same permissions. See the [REST API example](../examples/rest-api) for access
keys and external names.

## When a declaration is not enough

Germanio prefers declarations: states, rules and permissions cover most applications. For a
rule that is really specific, hooks such as `antes de criar` (before creating) run your own
logic inside the same transaction, and can refuse the change with a message. They are the
last resort, not the starting point, and they cannot bypass permissions or rules.

## Organizing a project

`ge new` creates `app.ge`, `backend/` and `frontend/`. `app.ge` names the system and imports
both folders; every file in them becomes part of the application. Backend files say what
exists and who may do what; frontend files hold pages. A block of the same data may appear in
several files: the blocks are merged, repeating the same fact is harmless, and two different
values for the same thing are reported as an error that shows both origins.

## Glossary

| Germanio | English |
| --- | --- |
| crie sistema | create system |
| tem | has |
| pertence a | belongs to |
| começa | starts |
| pode | can |
| acesso | access |
| todos | everyone |
| usuario | a logged-in person |
| seu, sua, seus, suas | their own |
| somente | only |
| ver, criar, editar, excluir | see, create, edit, delete |
| administrar | administer (see, edit, delete and manage what belongs to it) |
| permita | allow |
| pesquisar, filtrar por | search, filter by |
| obrigatório, único | required, unique |
| tenha login, tenha cadastro | have login, have sign-up |
| tenha papeis | have roles |
| página, mostre | page, show |
| disponibilize … para integração | publish … for integration |
