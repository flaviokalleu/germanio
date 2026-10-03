# Germanio Evolution Proposals (GEPs)

A GEP is a short, numbered document that proposes a deliberate change to the language and
records the decision. The process borrows the useful parts of Python's PEPs, Rust's RFCs and
Swift Evolution (a template, closed statuses, alternatives required, implementation before
acceptance) and drops what a small project does not need. The study behind it is in
[`docs/research/languages/LANGUAGE_EVOLUTION.md`](../research/languages/LANGUAGE_EVOLUTION.md).

## When a GEP is required

A GEP is required when a change alters, for the same `.ge` program:

- the **syntax** (a new construction, a new section, a new keyword, a removed form);
- the **meaning**, meaning `ge explain` would show a different fact;
- the **runtime contract** (what a declared capability guarantees: limits, security,
  transactions, what an HTTP client of a generated app observes);
- the **standard library** or a **capability** exposed to `.ge`.

A GEP is **not** required for bug fixes (the implementation disagrees with
[`docs/INTENCAO.md`](../INTENCAO.md)), clearer error messages, typos, performance work that
keeps the meaning, or internal refactoring. When in doubt, open an issue and ask.

## Lifecycle

| Status | Meaning |
| --- | --- |
| Rascunho (draft) | written, under discussion; anything may change |
| Em teste (trial) | implemented behind the proposal, tested, not yet normative |
| Aceita (accepted) | decided; the specification is updated in the same change as the implementation |
| Implementada (implemented) | in a release |
| Rejeitada (rejected) | decided against, with the reason; listed below so it is not re-proposed without new arguments |
| Retirada (withdrawn) | the author gave up |
| Substituída (superseded) | replaced by a later GEP |

Rules:

- **Numbers are never reused.** A new GEP takes the next free number.
- **Alternatives are required**, including "do nothing".
- **Implementation before acceptance.** A syntax change is accepted only with a working
  implementation, normative tests and the equivalence test when a flat form exists.
- **The author does not accept their own proposal.** The maintainer decides; an automated
  agent may write and implement a GEP but never marks it accepted.
- **The specification stays normative.** Until a GEP is accepted, `docs/INTENCAO.md`
  governs; a proposal never changes it silently.

## How to propose

1. Open an issue with the *Language proposal* template to discuss the intent.
2. Copy [`0000-template.md`](0000-template.md) to `NNNN-short-name.md` and open a pull request.
3. Evaluate the proposal with the criteria of `docs/INTENCAO.md` › *Como avaliar uma
   sintaxe* and the checklist in the simplicity skill.

## Index

| GEP | Title | Status |
| --- | --- | --- |
| [0001](0001-processo-gep.md) | The GEP process | Aceita |
| [0002](0002-secoes-de-pagina.md) | Page sections (top, actions, filters, columns, empty) | Aceita |
| [0003](0003-contrato-de-formulario.md) | The form contract (422, errors per field, no notices in the URL) | Rascunho |
| [0004](0004-tema-como-tokens.md) | Theme as deterministic tokens with guaranteed contrast | Rascunho |
| [0005](0005-concorrencia-por-intencao.md) | Concurrency by intent (bounded, cancellable, never dropped) | Rascunho |
| [0006](0006-pacotes.md) | Packages: principles now, a manager later (lockfile format reserved) | Rascunho |
| [0007](0007-idiomas-da-intencao.md) | Languages of the intent layer (Portuguese only, with a clear error; phrase tables later) | Rascunho |
| [0008](0008-recuperacao-de-senha.md) | Password recovery (`tenha recuperação de senha`) | Aceita |
| [0009](0009-pendencias.md) | Pending items (`pendência para responsaveis`) | Aceita |
| [0010](0010-descarte-de-campo.md) | Discarding a field on purpose (`descarte fax`) | Em teste |
| [0011](0011-historico.md) | History (`guarda histórico`) | Em teste |
| [0012](0012-indicadores.md) | Indicators (`indicadores` › `total de issues abertas`) | Em teste |
| [0013](0013-avisos-por-email.md) | Notices by e-mail (`tenha avisos por e-mail`) | Em teste |
| [0014](0014-arquivos.md) | Files of a record (`anexo arquivo`, `foto imagem`) | Em teste |
| [0015](0015-variaveis-das-execucoes.md) | Variables of executions (`pipelines usam as variaveis do projeto`) | Em teste |
| [0016](0016-branches-protegidas.md) | Protected branches named by data (`enviar código para as branches protegidas`) | Em teste |
| [0017](0017-mencoes.md) | Mentions create pending items (`pendência para` › `mencionados`) | Em teste |
| [0018](0018-identidade-do-esquema.md) | Schema identity: human, canonical and physical names (G112) | Rascunho |
| [0019](0019-fronteira-dos-adaptadores.md) | The boundary between domain and adapters (G114) | Rascunho |
| [0020](0020-paginas-vivas.md) | Pages stay up to date (no syntax; FASE 2) | Em teste |
| [0021](0021-presenca.md) | Presence (`tenha presença`) | Em teste |
| [0022](0022-leitura.md) | Reading and unread counts (`guarda leitura`) | Em teste |
| [0023](0023-por-estado.md) | Showing data by state, boards (`cartoes por estado`) | Em teste |
| [0026](0026-minimo-de-aprovacoes.md) | A minimum of approvals before an action (`precisa de 2 aprovações para mesclar`) | Em teste |
| [0027](0027-formas-de-mesclar.md) | How a proposal is merged (squash, linear) and merging when the executions pass (no syntax) | Em teste |
| [0029](0029-copias.md) | Copies of a record (`acesso` › `copiar`) | Em teste |
| [0030](0030-estrelas-topicos-avatar.md) | Marks by people (`recebe estrelas`), one item of a list as a filter, the address of a file | Em teste |
| [0031](0031-confirmacao-de-email.md) | E-mail confirmation (`tenha confirmação de e-mail`) | Em teste |
| [0032](0032-dois-fatores-e-chaves.md) | Credentials beyond the password: public keys (`chave pública`) and two-factor authentication (`tenha autenticação em dois fatores`) | Em teste |
| [0033](0033-adaptador-pede-a-aplicacao.md) | An adapter asks the application itself (`superficie.pedir`, `superficie.contar`) | Em teste |
| [0034](0034-mudar-de-lugar.md) | A record moves to another parent (`pode mudar de projeto`) | Em teste |
| [0035](0035-git-lfs.md) | Large files in repositories (Git LFS; no syntax) | Em teste |
| [0036](0036-espelhos.md) | Mirrors of a repository (`espelham o repositório do projeto`) | Em teste |
| [0037](0037-git-por-ssh.md) | Git over SSH (no syntax; `GERMANIO_SSH_ENDERECO`) | Em teste |
| [0039](0039-login-com-conta-externa.md) | Sign-in with an external account, OpenID Connect (`tenha login com conta externa`) | Em teste |
| [0043](0043-varios-valores-num-filtro.md) | Several values in a filter: any of them, or all the items of a list (no syntax) | Em teste |
| [0044](0044-lugar-pelo-endereco.md) | The place of a new record, named by its address (no syntax) | Em teste |
| [0047](0047-agregados.md) | Numbers of a record: counts and sums of what belongs to it, and zeroing a sum (`indicadores` › `soma do peso das issues`, `pode zerar tempo gasto`) | Em teste |
| [0048](0048-pares.md) | A link is unique per pair, never to itself (`única por par de issues`) | Em teste |
| [0049](0049-segredos-guardados.md) | Secrets kept encrypted at rest, with key rotation (no syntax: `oculto`) | Em teste |
| [0050](0050-soma-nunca-negativa.md) | A sum that never goes below zero (`não pode ficar com tempo gasto negativo`) | Em teste |

Revisão final das GEPs em teste 0010–0017, com recomendação por GEP e sem mudança de status:
[REVISAO_0010_0017.md](REVISAO_0010_0017.md).

Pending decisions that will become GEPs are listed in
[`docs/INTENCAO.md` › Pendências da sintaxe hierárquica](../INTENCAO.md#pendências-da-sintaxe-hierárquica)
and in [`GERMANIO_GAPS.md`](../../GERMANIO_GAPS.md).

## Rejected ideas

None recorded yet. Rejected GEPs are listed here with a one-line reason.
