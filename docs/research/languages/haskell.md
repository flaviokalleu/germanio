# Estudo dirigido: Haskell — pureza, efeitos no tipo, preguiça e o custo da previsibilidade

**Data:** 2026-09-28
**Status:** pesquisa, sem força normativa (a norma é `docs/INTENCAO.md`).
**Pergunta principal:** o que o Haskell ensina sobre separar descrição de efeito, sobre o custo
de uma semântica de avaliação que o programador não vê (space leaks), sobre type classes e
derivação, e sobre fragmentação por extensões e por gerenciadores de pacotes?
**Complementos:** a regra de layout (off-side) do Haskell já está em
[`sintaxe-e-configuracao.md`](sintaxe-e-configuracao.md) e em
[`../sintaxe-hierarquica.md`](../sintaxe-hierarquica.md); aqui só entra o que falta.

## Fontes consultadas

- Hudak, Hughes, Peyton Jones, Wadler, "A History of Haskell: Being Lazy With Class", HOPL III,
  2007 — https://www.microsoft.com/en-us/research/wp-content/uploads/2016/07/history.pdf (texto extraído; trechos citados lidos)
- HaskellWiki, "Space leak" — https://wiki.haskell.org/Space_leak
- GHC proposal 0380, "GHC2021" — https://github.com/ghc-proposals/ghc-proposals/blob/master/proposals/0380-ghc2021.rst
- Yesod blog, "Installing application dependencies using Stackage, sandboxes, and freezing", 2014 — https://www.yesodweb.com/blog/2014/11/cabal-sandbox-stackage (vista pela busca; contexto do "cabal hell")
- Stackage Server FAQ — https://github.com/fpco/stackage/wiki/Stackage-Server-FAQ (vista pela busca)

Não verificados nesta passagem: detalhes do modelo "nix-style" do `cabal v2-build` (a página da
documentação retornou 404) e o estado atual do GHC2024.

---

## Matriz (compacta)

| Item | Haskell | Relevância para o Germanio |
|---|---|---|
| Objetivo original | Uma linguagem funcional preguiçosa comum para pesquisa e ensino, unificando dezenas de linguagens semelhantes (HOPL) | — |
| Filosofia | Pureza como consequência da preguiça: "Once we were committed to a lazy language, a pure one was inescapable" (HOPL §3.2) | — |
| Sintaxe | Equações, casamento de padrões, layout off-side, operadores definíveis | layout já estudado (ver complementos) |
| Gramática | Relatório Haskell 98/2010; a regra de layout ficou "fairly complex ... formally specified for the first time in the Haskell 98 Report" (HOPL §4.1) | o `.ge` tem layout mais simples e deve mantê-lo assim |
| Parser / AST / IR | GHC: AST → Core (lambda cálculo tipado pequeno) → STG → Cmm (não verificado nesta passagem) | a ideia de um núcleo pequeno para onde tudo se reduz é a mesma de `ast.App` |
| Análise semântica | Inferência Hindley-Milner estendida; type classes resolvidas por dicionários | — |
| Sistema de tipos | Estático, forte; **efeitos no tipo** (`IO a`): uma função `Int -> Int` "will not read or write any mutable variables, nor will it perform any input/output" (HOPL §3.2) | ver "Para o Germanio", item 1 |
| Type inference | Global dentro do módulo, anotações opcionais no nível superior | — |
| Runtime | GC geracional, threads verdes, STM | — |
| Memory management | GC; avaliação preguiçosa cria *thunks* | ver "Performance" |
| Standard library | `base` pequena; o resto em Hackage | — |
| Package manager | Cabal (resolução por restrições sobre o Hackage) vs. Stack (conjuntos curados do Stackage); a divisão nasceu do "cabal hell" | ver "Principais problemas" |
| Formatter | Vários (ormolu, fourmolu, stylish-haskell); nenhum oficial | — |
| Language server | HLS (haskell-language-server) (não verificado nesta passagem) | — |
| Diagnostics | Erros de tipo historicamente difíceis para iniciantes; GHC passou a ter códigos de erro indexados (não verificado nesta passagem) | — |
| Evolution process | Comitê do relatório (98, 2010) e depois GHC Steering Committee com propostas; o GHC virou o padrão de fato | — |
| Backward compatibility | Mudanças na `base` (ex.: Functor-Applicative-Monad) quebraram muito código (não verificado nesta passagem) | — |

### Principais acertos
- **Descrição separada do efeito.** A pureza força a fronteira entre calcular e agir; o tipo
  mostra quem age. Isso tornou o "raciocínio equacional" acessível: "much simpler to apply than
  a formal denotational or operational semantics, thanks to Haskell's purity" (HOPL).
- **Type classes** resolveram sobrecarga de forma geral e viraram o traço mais distintivo da
  linguagem (HOPL §6). `deriving` gera instâncias (igualdade, ordem, exibição) a partir da
  declaração do tipo — o precedente mais próximo da derivação que o Germanio faz a partir de
  `tem`.
- **Núcleo pequeno** (Core) para o qual toda a sintaxe se reduz.
- **GHC2021** transformou 29 extensões comuns num padrão nomeado, escolhido com dados de 13 951
  pacotes do Hackage e 1 348 respostas de pesquisa, exigindo 8 de 11 votos (proposta 0380).

### Principais problemas
1. **Space leaks.** Thunks acumulam memória que o código não mostra; causas típicas: acumulação
   preguiçosa e `foldl`; correções: `seq`, *bang patterns*, estruturas estritas (HaskellWiki).
   O custo não é de desempenho médio, é de **previsibilidade**: o consumo de memória depende da
   ordem de avaliação, que o programador não escreve.
2. **Extensões como fragmentação.** Cada arquivo pode ligar `LANGUAGE` pragmas que mudam a
   sintaxe e o sistema de tipos; "Haskell" passou a ser um conjunto de dialetos. O GHC2021 é um
   remédio posterior.
3. **"Cabal hell".** Instalação global e resolução de restrições produziam conflitos de versão;
   a resposta foram dois caminhos concorrentes (Stack com Stackage curado; depois o Cabal com
   builds locais isolados), e a comunidade dividida entre ferramentas.
4. **Layout complexo** para "do what the programmer expects" (HOPL §4.1): a regra ficou difícil
   de especificar.

### Complexidade acumulada
Grande no sistema de tipos (famílias de tipos, GADTs, tipos dependentes parciais), toda
opcional por extensão; o iniciante lê código com extensões que não conhece.

---

## Sintaxe (perguntas obrigatórias)

| Pergunta | Haskell | Germanio hoje |
|---|---|---|
| Dados | `data` com construtores; registros; `deriving` | `tem`; derivação de CRUD, API e telas |
| Funções | equações puras, currying | ausentes do domínio |
| Módulos / imports | `module`/`import` com listas explícitas | `importar` pasta ou nomes |
| Relações | valores/referências imutáveis | nomes resolvidos |
| Estado | sem mutação implícita; `IORef`, `State`, STM | estados declarados; mudanças só por ações/eventos |
| Fluxo | recursão, `case`, guards | regras, `quando` |
| Erros | `Maybe`/`Either`, exceções em `IO` | diagnostics na compilação; erro com posição no runtime |
| Concorrência | threads verdes, STM | runtime administra |
| Null | não existe; `Maybe` explícito | `obrigatório` / ausente |
| Boilerplate | baixo com `deriving`; alto com transformadores de mônadas | derivado |
| Projetos grandes | tipos ajudam; extensões atrapalham | `ge explain` |

Custo cognitivo: para o público do Germanio, mônadas, currying e type classes são inacessíveis.
O que sobrevive é a **ideia** — frases que descrevem vs. frases que agem — sem a maquinaria.
No `.ge`, `tem`/`acesso`/`regras` são descrição; `quando`, ações e chamadas externas são efeito.

## Performance e memória (comparação com `performance/AUDITORIA.md`)

- **Previsibilidade vs. média.** O Haskell mostra que memória imprevisível é pior que memória
  alta: um space leak aparece em produção, não no teste. A AUDITORIA do Germanio encontra a mesma
  classe por outro caminho — **trabalho e memória que crescem com o tempo sem que o `.ge` diga
  nada**: fila de tarefas nunca limpa (51% de um núcleo ocioso com 1 milhão de linhas, §2.6),
  visibilidade por registro O(tabela) (88 MB por request com 50 000 linhas, §2.5), mapas que só
  crescem (§3.7). São "leaks" de projeto, não de avaliação. O usuário nunca administra memória;
  por isso o runtime precisa garantir limites.
- **Estrito por padrão.** O Germanio avalia de forma estrita (handlers em Go sobre `ast.App`);
  não deve introduzir preguiça no domínio. Onde o Haskell precisou de `seq`, o Germanio precisa
  de **teto e limpeza por construção** (retenção de tarefas, paginação obrigatória, limites de
  leitura).
- **Efeitos visíveis como otimização.** Separar descrição de efeito também é o que permite ao
  Germanio decidir estaticamente o que precisa rodar (AUDITORIA §4: capabilities inicializadas
  sem uso); uma capability sem nenhuma frase de efeito no `.ge` não deveria existir no processo.

## Para o Germanio

| # | Classe | Lição | Problema do Germanio | Arquivo | O que remove da cabeça do programador |
|---|---|---|---|---|---|
| 1 | ADAPTAR | Efeitos visíveis sem mônadas: `ge explain` marca cada frase como descrição ou efeito (grava, envia, chama externo) e lista, por ação, os efeitos que ela dispara | o efeito de uma ação (webhook, e-mail, tarefa) só se descobre lendo vários blocos | `tooling/explicar`, `compiler/parser/resolver.go` | rastrear "o que acontece quando clico" |
| 2 | ADOTAR | Nenhuma extensão por arquivo que mude a linguagem; se houver edições, uma por projeto e nomeada (lição do GHC2021) | dois front-ends convivem (núcleo estrito e dialeto de aplicação) | `SPEC.md`, `docs/INTENCAO.md`, `compiler/parser` | saber qual dialeto um arquivo usa |
| 3 | ADOTAR | Memória e trabalho previsíveis por construção: todo crescimento com o tempo tem teto ou retenção padrão declarada pelo runtime | fila sem limpeza, visibilidade O(tabela), mapas sem teto (AUDITORIA §2.5, §2.6, §3.7) | `runtime/servidor/tarefas.go`, `runtime/interpreter/intencao.go`, `runtime/banco` | descobrir o leak em produção |
| 4 | ADOTAR | `deriving` como modelo: derivação a partir da declaração, desligável por item, e explicada | já é o centro do Germanio; falta a forma uniforme de "não derive isto" | `docs/INTENCAO.md`, `compiler/parser/resolver.go` | escrever CRUD à mão |
| 5 | ADAPTAR | Núcleo pequeno (Core) para onde toda a sintaxe se reduz, com teste de equivalência | já existe a redução bloco → frases planas → `ast.App`; os dois front-ends deveriam reduzir ao mesmo núcleo | `compiler/parser/hierarquia.go`, `compiler/parser/resolver.go` | — |
| 6 | EVITAR | Layout que tenta "do what the programmer expects" com exceções | manter o off-side do `.ge` sem casos especiais | `compiler/parser/hierarquia.go` | memorizar exceções |
| 7 | EVITAR | Avaliação preguiçosa ou semântica em que a ordem não escrita muda o resultado ou a memória | a ordem alfabética de `importar "backend"` não deve alterar o significado; testar | `compiler/parser/resolver.go` | pensar em ordem de arquivos |
| 8 | EVITAR | Dois gerenciadores de pacotes concorrentes (Cabal vs. Stack) | quando houver dependências `.ge`, um só mecanismo oficial com conjunto travado | futuro `cli` | escolher ferramenta |
| 9 | INVESTIGAR | Conjunto curado e testado de adaptadores (Stackage) como forma de distribuir `integracoes/` | não há distribuição de adaptadores | `integracoes/` | resolver versões |
