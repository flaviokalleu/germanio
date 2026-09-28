# GEP 0013: Notices by e-mail

- **Status:** Em teste (implemented; not normative until the maintainer decides)
- **Author:** agent (GitLab stress test, inventory NT-03); decision by the maintainer
- **Level:** 1
- **Layer:** domain (one phrase), core (composition of pending items, effects after the commit
  and the e-mail configuration)

## Problem

A person who receives a pending item (GEP 0009) only learns about it when they open the
application. Every product with responsibility — GitLab, ticketing, CRMs — also tells them by
e-mail. Writing it by hand means a hook that finds the person, formats a message, calls an
e-mail function and, until G86, did it while holding the database's write lock.

## Evidence

GitLab inventory NT-03 (notification e-mails on assignment). The mechanisms already exist:
pending items decide who must know; G86 runs external effects after the commit; GEP 0008
configures e-mail outside the source code.

## Proposal

```text
tenha avisos por e-mail
```

## Semantics

- Every new pending item sends one e-mail to its owner, with the same text as the pending item
  (so the record's title only if the owner may see it, G107) and the address of the
  application (`GERMANIO_URL_PUBLICA`, never the request's `Host`).
- The e-mail is an external effect: it is sent after the change is saved, never while the
  transaction is open; an undone change sends nothing; a failure to send is logged and does not
  undo the change.
- The e-mail configuration is the one of GEP 0008 (`GERMANIO_SMTP_*` or
  `GERMANIO_CORREIO_PASTA`); without it the application starts and says the notices are not
  available (never a silent failure).
- Only pending items are notified. Mentions, digests, per-person preferences and other channels
  (push, chat) are not in this GEP.

## Errors

`tenha avisos por e-mail` without any `pendência para`: the error says notices tell people about
their pending items and shows how to declare them.

## Evaluation

One phrase; no address, template, function or transaction for the author to know.

## Performance and security

One e-mail per new pending item, sent after the commit (it never holds the write lock). The
address comes from a validated e-mail field (no header injection); the subject is fixed.

## Compatibility and migration

Additive.

## Tests

`TestAvisosPorEmail` (the assignee gets one e-mail after the change; the one assigning gets
none; an undone change sends none; a restricted record's title is not in the e-mail), the
missing-pending error.
