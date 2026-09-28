# crm

A small CRM: companies, their contacts and sales opportunities.

## What it demonstrates

- Relations without IDs in the file: `empresas` › `tem` › `contatos`, `oportunidades`. Each
  contact and opportunity requires a company; deleting a company removes what belongs to it.
- `nome obrigatório e único`, `valor dinheiro` (money), `site`, `cidade`.
- An opportunity **starts open** and **can be won or lost** (`pode` › `ganhar`, `perder`);
  both outcomes are final (`regras` › `ganha é final`, `perdida é final`). Each action records
  who and when (`ganha_em`, `ganha_por_id`).
- The person who creates an opportunity is its owner (`dono`): only they edit it or mark it
  won or lost. Every signed-in person sees and creates; only administrators delete.
- Search, filters by city, state and owner; two pages.

## Run

```bash
go build -o ge ./cmd/ge
./ge check examples/crm/app.ge
GERMANIO_ADMIN_SENHA='choose-a-password' ./ge rodar examples/crm/app.ge 8080
```

Open http://localhost:8080. Opportunities are created inside a company:
`POST /_ge/api/empresas/<id>/oportunidades`, then `POST /_ge/api/oportunidades/<id>/ganhar`.

Known limitations: a separate "account manager" person (`responsavel`) cannot be given
permissions yet, so the owner is the creator ([G77](../../GERMANIO_GAPS.md)); transitions
have no origin state, which is why won and lost are declared final
([G78](../../GERMANIO_GAPS.md)). Dashboards with totals are not supported yet
([G62](../../GERMANIO_GAPS.md)).

**Capabilities:** relations, cascading delete, states and transitions, final states,
ownership, initial administrator, search, filters, pages.

**Keywords are Portuguese** (`ganhar` = win, `perder` = lose, `é final` = is final).
See [the index](../README.md#a-note-on-language).
