# ecommerce

A shop: the catalog is public, administrators manage products, each customer sees only
their own orders.

## What it demonstrates

- `produtos`: everyone sees; only administrators create, edit and delete
  (`somente administrador`). `preco` is money and `estoque` an integer, inferred from the names;
  `foto` is an image; `descrição formatada` is formatted long text.
- `pedidos` › `pertence a` › `produto`, `usuario`: an order refers to an existing product
  (an unknown one is refused) and belongs to the person who places it.
- An order **starts open** and **can be paid or cancelled**; a cancelled order is final.
- `usuario` › `criar seus`, `ver seus`, `pagar seus`, `cancelar seus`: customers act only on
  their own orders; other people's orders are "not found".
- `quantidade começa com 1 min 1`: default value and minimum.

## Run

```bash
go build -o ge ./cmd/ge
./ge check examples/ecommerce/app.ge
GERMANIO_ADMIN_SENHA='choose-a-password' ./ge rodar examples/ecommerce/app.ge 8080
```

Sign in as `admin@loja.local` to add products; create a customer account to place orders.

Known limitation: "the order cannot ask for more than the stock" has no declarative form
yet, so this example does not check or reduce the stock ([G61](../../GERMANIO_GAPS.md)). The
`ge init` template does it with level-3 logic (`antes de criar` + `recuse`).

**Capabilities:** public catalog, administrator-only actions, relations, ownership, states
and transitions, final states, validation, pages.

**Keywords are Portuguese** (`pertence a` = belongs to, `pagar` = pay, `cancelar` = cancel).
See [the index](../README.md#a-note-on-language).
