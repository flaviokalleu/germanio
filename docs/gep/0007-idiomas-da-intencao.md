# GEP 0007: Languages of the intent layer

- **Status:** Rascunho
- **Author:** agent (research consolidation); decision by the maintainer
- **Discussion:** none yet
- **Gaps:** G74, G92; related: G72 (accent folding), lesson A13 (show words as written)
- **Level:** 1 (what a person may write) and the older technical syntax
- **Layer:** core (lexer, parser, `ge check`)

## Problem

A person who does not read Portuguese opens Germanio, writes the example in English, runs
`ge check`, and is told the program is valid. It has no data. Nothing tells them that the intent
layer only understands Portuguese.

Behind that, two separate defects:

1. **The promise and the code disagree.** `README.md` says "The intent layer is in Portuguese
   only", and `CLAUDE.md` says the same, but `docs/INTENCAO.md:288-289` (the norm) says words
   "podem ser escritas com ou sem acento e em qualquer idioma do léxico". The intent parser does
   not implement any language other than Portuguese.
2. **The multilingual mechanism is a global word map.** `compiler/idiomas/idiomas.go` maps words
   of 19 languages to canonical Portuguese keywords, all at once, for every file. Common
   Portuguese words collide with keywords of other languages.

## Evidence

Reproduced at `0c12051` with `go build ./cmd/ge` and temporary files:

- `create system Shop` / `customers` / `has` / `name required`: `ge check` prints
  "✓ válido … modelos: 0" and exits 0 (G74).
- `create system Shop` followed by the line `zzz qqq`: also "válido", exit 0. The fallback path
  of `ge check` (`tooling/gecli/cli.go`, after `semantic.Load` and `Compilar` do not produce an
  application) hands the file to the older CLI, whose parser ignores lines it does not
  recognize. This contradicts `INTENCAO.md` › Erros ("Linhas que nenhuma construção reconhece
  são erro, nunca ignoradas").
- The global map (`compiler/idiomas/idiomas.go:30, 39, 485`; consulted in
  `compiler/lexer/lexer.go:762-769`) turns the Spanish `no` into `nao`, `o` into `ou`, `estado`
  into `status`, `si` into `se`, and the Turkish `don` into `retornar`. Measured in
  `docs/research/languages/DISCOVERIES.md` D3: `campo no projeto` lexes as `campo`, `nao`,
  `projeto`. On the intent path tested there, the field `estado` survived because the parser
  reads the original spelling (`Raw`); where else the translated value is used is unknown
  (lesson I17).
- English keywords also live in the lexer's own keyword table (`compiler/lexer/lexer.go:307,
  324`: `system`, `create`, …), so the map is not the only source of foreign words.
- Research: Hedy offers keywords in 54 languages and its authors record that "truly localizing a
  language entails more than just keywords": word order differs (Dutch conditions, Arabic has no
  copula), and punctuation changes by language (DISCOVERIES D3, with the paper). Kip treats
  Turkish grammatical case as structure, not vocabulary (D4). The Portuguese intent phrases
  depend on Portuguese grammar: prepositions (`pertence a`, `dos projetos`), gender agreement
  (`aberta` → `fechada`), possessives (`seus pedidos`), plural rules (`papeis` → `papel`).
- No user of any language other than Portuguese is documented (lesson I15).

## Current state

```text
crie sistema Loja          # understood
create system Shop         # accepted by the older syntax; intent lines after it are ignored
clientes                   # intent: Portuguese only
    tem
        nome obrigatório
customers                  # not an error today when it follows "create system"
    has
        name required
```

## Alternatives studied

**A. Portuguese only in the intent layer, with a clear error for other languages.**
The intent layer stops pretending. Unrecognized lines are errors on every path, including the
fallback to the older syntax. When an unrecognized line starts with a word that is a known
keyword of another language (the existing map, used **backwards, only for the message**), the
error says so: "the intent layer is written in Portuguese; `has` corresponds to `tem`". The
global map stops applying to intent files; it remains for files in the older technical syntax
until that syntax is retired.

**B. One language per file, declared.**
A file states its language once (`idioma: inglês`, or a file suffix). The lexer loads only
Portuguese plus that language, and a test fails the build if the pair has collisions. This
removes collisions but not the deeper problem: word-by-word substitution still cannot express
`developer pode enviar código dos projetos` in a language with another word order.

**C. Translation by phrase, not by word.**
Each construction in the closed table of sections and plain phrases gets a template per
language, with slots, written and reviewed by native speakers:

```text
pt: <papel> pode <ação> <dado>            developer pode enviar código dos projetos
en: <role> can <action> <data>            developers can push code to projects
```

Each language is a grammar that produces the same facts, proved by the equivalence tests that
already exist between the hierarchical and flat forms. `ge fmt` could then translate a file
between languages with the meaning checked by reparse. This is the only option that handles
word order; it is also the most expensive (a grammar per language, plural and gender rules per
language, diagnostics per language) and nobody in the research has done it for a
general-purpose intent language.

**D. Do nothing.**
Keeps an accepted program with no data, a norm that promises what the parser does not do, and
collisions that may change meaning in paths nobody has tested.

## Proposal

**Recommended: A now, and C as the only admissible way to add a language later, using B's
declaration.** Concretely:

1. The intent layer is Portuguese. The norm sentence at `INTENCAO.md:288-289` becomes
   "Palavras em português podem ser escritas com ou sem acento" (the multilingual part is
   removed), in the same change that implements the rest (a GEP never changes the norm
   silently).
2. No path of `ge check` or `ge run` accepts a file with lines it does not recognize. The
   fallback to the older CLI fails when the older parser ignores a line.
3. The lexer used by the intent parser does not translate. The global map applies only to files
   in the older technical syntax (`dados`/`telas` blocks), and a test fails the build when an
   entry collides with a list of common Portuguese words (articles, prepositions, conjunctions,
   and every word used in `docs/INTENCAO.md` examples).
4. Diagnostics may use the map backwards to recognize a foreign keyword and name its Portuguese
   equivalent. The map never changes the meaning of an intent file.
5. A future language enters only through its own GEP that delivers a complete phrase table
   (option C), native-speaker review, the equivalence test against Portuguese for every row of
   the section table, and diagnostics in that language. The file then declares its language
   (option B's mechanism), and one file has one language.

Before (today):

```text
create system Shop

customers
    has
        name required
```

`ge check`: "✓ válido", 0 models, exit 0.

After (proposed; the message format follows `INTENCAO.md` › Erros):

```text
app.ge:3 — não entendi a linha "customers".
Por quê: a camada de intenção do Germanio é escrita em português; "has" (linha 4) é uma
palavra-chave de outro idioma.
Como corrigir: escreva as palavras da linguagem em português — "has" corresponde a "tem" e
"create system" a "crie sistema". Os nomes dos dados ("customers") podem continuar como estão.
```

Exit status 1. No program changes meaning; a program that was silently empty now fails.

## Semantics

- An intent file is Portuguese. Keyword spelling with or without accents is accepted, as today;
  whether that folding applies to domain names is G72 and outside this GEP.
- The older technical syntax keeps its multilingual keywords for compatibility, with the
  collision test, until it is retired by its own GEP.
- The map is a diagnostic aid on the intent path: it may produce the "why" and "how to fix"
  parts of an error, never a fact. `ge explain` never shows a fact derived from a translated
  word.

## Errors

| Situation | Message | Automatic fix |
|---|---|---|
| intent file with a line that starts with a foreign keyword | the intent layer is Portuguese; the Portuguese word for it | no: translating a line can change meaning (word order, gender, plural) |
| file accepted by the older syntax with ignored lines | "não entendi a linha …" with the line number, on every path | no |
| (older syntax) new map entry equal to a common Portuguese word | build-time test failure naming the entry and the word | — |
| (future, option C) a file mixes two languages | the file's declared language and the line in the other | no |

## Evaluation

- **Technical concepts required:** none new for a Portuguese reader. For a reader of another
  language the cost is honest: Germanio is not for them at level 1 yet, and the error tells them
  in one line instead of accepting an empty program.
- **Cognitive load:** removes the invisible list of words from 19 languages that cannot be used
  as names in the older syntax without surprises.
- **Determinism:** one language per file, one meaning per word; nothing depends on which
  language a word happens to belong to.
- **Machine cost:** the intent lexer skips a map lookup per word; the collision test runs at
  build time.

## Impact

- **Lexer:** `compiler/lexer/lexer.go`: a mode without translation for the intent parser; the
  map lookup (lines 762-769) only in the older-syntax mode. English keywords in the lexer's own
  table (lines 307, 324) follow the same rule.
- **Parser:** the intent parser already reads `Raw`; the fallback chain in `tooling/gecli/cli.go`
  must not return success when the older parser ignored lines; the older parser must report
  unknown lines.
- **AST/resolver:** none.
- **Diagnostics:** the foreign-keyword hint (a reverse lookup of the map), in `compiler/diagnostics`.
- **Tooling:** the VS Code grammar generator (`vscode-germanio/tools/gerar_gramatica.py`) should
  not colour foreign keywords in intent files.
- **Documentation:** `docs/INTENCAO.md:288-289` (in the accepting change);
  `docs/INTENCAO.md` › Gramática.

## Performance and security

No measurable runtime cost. Security: a word that silently becomes a keyword (`no` → negation,
`o` → `or`) could change a permission or a rule in the older syntax; the collision test and the
removal of translation from the intent path close that class. Error messages never echo secrets.

## Compatibility and migration

- Intent programs in Portuguese: unchanged.
- Programs that `ge check` accepted with ignored lines now fail. That is the intended fix; the
  message names each line. No program in the repository depends on ignored lines (to verify
  over `examples/` and `demo/` in the implementing change).
- Programs in the older technical syntax that use foreign keywords: unchanged, unless an entry
  is removed by the collision test. Each removal is listed in the release notes with the
  Portuguese keyword to use; `ge fmt` may rewrite a removed foreign keyword to its Portuguese
  form only when the reparse proves the facts are the same.
- Public texts (README, site) already say "Portuguese only"; after acceptance the norm says the
  same.

## Trade-offs and alternatives

- **Cost of A:** it closes Germanio at level 1 to people who do not read Portuguese. That is
  already the truth; A only stops hiding it. The research did not find users in the other 19
  languages (I15).
- **Why not B now:** it fixes collisions and still promises languages whose grammar the parser
  does not know (E23). It is useful only together with C.
- **Why not C now:** it is a grammar per language, with nobody asking for one, and each new
  phrase in Portuguese would need to be written N times. It remains the recommended route when
  a real community asks, one language at a time, through its own GEP.
- **Do nothing:** rejected; it keeps an empty program reported as valid, which is the worst
  possible first experience for the person the language is meant for.

## Tests

- `create system Shop` + `customers` / `has` / `name required`: `ge check` fails with exit 1,
  names line 3, and says `has` corresponds to `tem`.
- `create system Shop` + `zzz qqq`: `ge check` fails and names the line (no path ignores it).
- An intent file containing `no`, `o`, `si`, `estado` as ordinary Portuguese words (field names,
  inside phrases) produces exactly the facts of the same file with those words replaced by
  neutral names, and `ge explain` shows them as written.
- Collision test: no entry of the older syntax's map equals a word in the list of common
  Portuguese words or in any `ge` block of `docs/INTENCAO.md`.
- Every existing example in `examples/` and `demo/` still passes `ge check` with the same facts.
