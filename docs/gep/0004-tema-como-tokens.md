# GEP 0004: Theme as deterministic tokens

- **Status:** Rascunho
- **Author:** agent (research consolidation); decision by the maintainer
- **Gaps:** G75 (contrast, theme ignored by intent pages, `estilo` without effect)
- **Level:** 1
- **Layer:** domain (vocabulary), core (token generation)

## Problem

`tema azul` produces buttons with white text on blue at 3.68:1, below the 4.5:1 WCAG minimum;
of the 15 color names only two pass. Intent pages ignore the theme entirely. `estilo
glassmorphism/flat/neumorphism/minimal` is parsed and has no effect. A person cannot say
"blue accent, soft corners, compact" and get a consistent, accessible interface.

## Evidence

`docs/research/frontend/GERMANIO_FRONTEND.md` §6 and "Divergências" (contrast computed with
the WCAG formula; `paginas.go` fixed tokens; `renderizador.go` fields never read).
`DESIGN_SYSTEMS.md`: Ant Design (seed → map → alias), Material 3 (a whole scheme from one
source color; HCT tone differences of 40/50 guarantee 3:1/4.5:1).

## Current state

`ast.Theme`, `ThemePreset()` and `ColorName` in `compiler/ast`; the older SPA renderer uses a
few of the fields; intent pages use hard-coded colors.

## Alternatives studied

Tailwind-style utilities (visual vocabulary: rejected for level 1, it is CSS in other words);
free hex colors (no contrast guarantee; kept for the technical level); seed-based generation
(Ant, Material): chosen.

## Proposal

A closed theme vocabulary whose every word has a testable effect:

```text
tema
    destaque azul
    cantos suaves
    densidade compacta
    modo automático
```

| Word | Values | Effect |
| --- | --- | --- |
| `destaque` | the names in `ColorName` | the seed color of the scheme |
| `cantos` | `retos`, `suaves`, `arredondados` | the base radius |
| `densidade` | `compacta`, `normal`, `confortável` | spacing step and control height |
| `modo` | `claro`, `escuro`, `automático` | the scheme(s) emitted |

`estilo …` and presets such as `tema moderno` stay accepted but without meaning in level 1
until they are defined as fixed sets of the words above; `ge check` warns that they have no
effect.

## Semantics

From the seed, Germanio derives the roles (surface, text-on-surface, primary,
text-on-primary, border, focus, success, error) for light and dark. Every text/background pair
is generated with a guaranteed contrast and verified with the WCAG 2.2 formula at build time;
a pair below 4.5:1 (3:1 for large text and borders) is a bug, never shipped. Tokens are
emitted as a small static CSS file; no CSS framework runs in the browser.

## Impact

Parser: the theme block as a section table. Resolver: the token set. Runtime: intent pages
and the SPA read the tokens. Tooling: `ge explain tema` lists every token and its contrast.

## Performance and security

Tokens computed once per load; the CSS is static and cacheable. Removes the Tailwind Play CDN
dependency from the pages that adopt it (a CSP and offline improvement).

## Compatibility and migration

`cor primaria "#…"` and hex colors stay in the technical level. `tema azul` maps to
`destaque azul`. `estilo …` keeps parsing, now with a warning.

## Trade-offs and alternatives

Fewer visual choices at level 1, by design. "Do nothing" keeps shipping inaccessible contrast.

## Tests

A golden test over every `ColorName` in both modes asserting the contrast of every pair;
equivalence of `tema azul` and `destaque azul`; the warning for `estilo`.
