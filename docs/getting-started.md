# Getting started with Germanio

This guide takes you from nothing to a running web application in a few minutes, and shows
how to read what Germanio understood from your code. It assumes no web development
knowledge. The language keywords are Portuguese; each one is translated the first time it
appears.

## 1. Install

You need [Go](https://go.dev/dl/) (the version in [`go.mod`](../go.mod) or newer) and Git.
Nothing else: the database (SQLite) is built in.

```bash
git clone https://github.com/flaviokalleu/germanio.git
cd germanio
go build -o ge ./cmd/ge
./ge --version
```

Put `ge` somewhere on your `PATH` (or use the [install script](../install.sh), which builds
it into `~/.local/bin`). The rest of this guide writes `ge` for the command.

## 2. Create an application

```bash
ge new shop
```

This creates:

```text
shop/
├── app.ge          the entry point: the system's name and what it imports
├── backend/        what exists, who may do what, what happens
├── frontend/       what appears: pages
└── .env.exemplo    settings (copy it to .env)
```

## 3. Run it

```bash
cp shop/.env.exemplo shop/.env
ge run shop/app.ge
```

Open <http://localhost:8080>. You get a working store: sign-up and login, a product list
anyone can see, products only an administrator can change, and orders each customer sees
only for themselves.

The template declares an initial administrator, `admin@loja.local`. Its password never
appears in `.ge` files: before the first run, set `GERMANIO_ADMIN_SENHA` in `shop/.env` to a
long password of your own (and `GERMANIO_SEGREDO` to a long random phrase). The
administrator is created the first time the application starts; log in with that e-mail and
password.

## 4. Read the code

Open `shop/backend/produtos.ge`:

```ge
produtos
    tem
        nome obrigatório
        descrição formatada
        preco obrigatório
        estoque começa com 0

    acesso
        todos
            ver
        somente administrador
            criar
            editar
            excluir

    permita
        pesquisar por nome
```

Read it from top to bottom. What is indented belongs to the line above:

- `produtos` (products) is a piece of data the application keeps.
- `tem` (has) lists its fields: a required name, a formatted description, a required price,
  and a stock that starts at 0. Germanio picks a type from the name (`preco` is money,
  `estoque` is a whole number) unless you say otherwise.
- `acesso` (access) says who may do what: `todos` (everyone) may see; only the administrator
  may create, edit and delete.
- `permita pesquisar por nome` (allow searching by name) turns on search.

## 5. Ask Germanio what it understood

```bash
ge explain produtos shop/app.ge
```

`explain` lists every field (including the ones Germanio adds, like creation dates), the
relations, the states, who may do each action, and for every fact the line it came from.
Nothing is hidden and nothing is guessed: if Germanio cannot decide something, `ge check`
stops with an error that says what is wrong, where, why and how to fix it.

## 6. Change something

Add a field to `produtos`, below `estoque`:

```ge
produtos
    tem
        nome obrigatório
        descrição formatada
        preco obrigatório
        estoque começa com 0
        categoria
```

Save, and run `ge check shop/app.ge`. The database gains the column the next time the
application starts; you never write a migration. Run `ge fmt shop` to put every file in the
one canonical layout (4 spaces per level).

## Next steps

- The [language tour](language-tour.md): data, relations, states, people and permissions,
  pages and integration, each with a runnable example.
- The [examples](../examples/): a to-do list, a CRUD register, authentication with roles, a
  REST API, a blog, a CRM and a store.
- The [specification](INTENCAO.md) (Portuguese) when you need the exact rules.
