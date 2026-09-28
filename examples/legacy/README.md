# Legacy examples

These examples use the **earlier technical syntax** of Germanio: `sistema` / `dados` /
`telas` / `eventos` blocks with typed fields (`nome: texto`), or the `app` / `tabela` /
`quando receber` form. That syntax is still supported — it is documented in the historical
reference [`docs/SPEC.md`](../../docs/SPEC.md) and these files are loaded by the smoke test
`runtime/examples_smoke_test.go` — but it is **not the default level** of the language.

**Do not copy them into new code.** New applications start from the intent layer
([`docs/INTENCAO.md`](../../docs/INTENCAO.md)); see the [examples index](../README.md) and the
`ge init` template.

| File | What it is |
| --- | --- |
| `blog.ge` | blog with categories, posts, comments and a newsletter |
| `cadastro-simples.ge` | a single register with a screen |
| `crm.ge` | companies, contacts, opportunities, activities and proposals |
| `ecommerce.ge` | products, brands, customers, orders, coupons and reviews |
| `english-mode.ge` | the earlier syntax written with English keywords |
| `prompt-saas.ge` | the `app` / `tabela` / `quando receber` form |
| `whaticket.ge`, `whaticket/` | a WhatsApp ticketing system (single file and organized in folders) |
| `evoticket/` | a larger ticketing and CRM system organized in folders |

Run them with the legacy CLI:

```bash
go build -o germanio .
./germanio check examples/legacy/crm.ge
./germanio run examples/legacy/crm.ge 8080
```
