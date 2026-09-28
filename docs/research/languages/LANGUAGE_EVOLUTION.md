# Evolução da linguagem: processos comparados e o GEP

**Data:** 2026-09-28. **Status:** pesquisa, sem força normativa. O GEP abaixo é a proposta
de pesquisa. Durante a redação deste estudo apareceu no repositório
[docs/gep/](../../gep/README.md) (GEP 0001, "The GEP process", com status Aceita), que cita
este arquivo como base. Onde os dois diferirem, vale `docs/gep/`; as diferenças de detalhe
(por exemplo a seção "Como ensinar" e a lista inicial de rejeitados, sugeridas aqui) ficam
como insumo para decisão. Fontes:
seções "Evolution process" e "Backward compatibility" de [python.md](python.md),
[rust.md](rust.md), [swift.md](swift.md), [go.md](go.md), [elixir.md](elixir.md),
[gleam.md](gleam.md), [zig.md](zig.md), [nim.md](nim.md) e [typescript.md](typescript.md).
Estado do Germanio: `AGENTS.md` (Documentation Gate), `docs/INTENCAO.md` › Evolução e
refatoração retroativa (linhas 735-750), `GERMANIO_GAPS.md`, `GERMANIO_EVOLUTION.md`,
`AGENT_STATE.md`.

---

## 1. Processos comparados

| Processo | Escopo | Artefato e seções | Decisão | Compatibilidade | Fonte |
|---|---|---|---|---|---|
| **Python PEP** | mudanças de linguagem e processo | PEP 1: Motivation, Rationale, Backwards Compatibility, **How to Teach This**, Reference Implementation, **Rejected Ideas** | Steering Council ou PEP-Delegate | PEP 387: deprecação por no mínimo duas versões, preferencialmente cinco; quebra exige "a large benefit to breakage ratio"; PEP 617 trocou o parser sem mudar a linguagem, com `-X oldparser` por uma versão | [PEP 1](https://peps.python.org/pep-0001/), [PEP 387](https://peps.python.org/pep-0387/), [PEP 617](https://peps.python.org/pep-0617/) |
| **Rust RFC + editions** | "any semantic or syntactic change to the language that is not a bugfix" e remoções | RFC em PR; Final Comment Period de dez dias (merge/close/postpone); aceitar não prioriza implementar | equipes | editions (2015/2018/2021/2024) opt-in por crate, interoperáveis, "skin deep" porque tudo compila para a mesma representação; migração por `cargo fix --edition`, com os mesmos lints de sugestão aplicável dos diagnósticos | [RFCs](https://github.com/rust-lang/rfcs/blob/master/README.md), [edition guide](https://doc.rust-lang.org/edition-guide/editions/index.html), [RFC 2052](https://rust-lang.github.io/rfcs/2052-epochs.html) |
| **Swift Evolution** | "applies only to the design of features"; bugs de recurso não lançado "can be freely fixed" | pitch → proposal no template (Motivation, Proposed solution, Detailed design, **Source compatibility**, ABI, Implications on adoption, Future directions, **Alternatives considered**) → **protótipo obrigatório** ("a viable proof of concept") → review ≥ 10 dias | Language Steering Group; review manager conduz, não decide | status fechados (Awaiting review … Implemented (versão)); `commonly_proposed.md`; vision documents; *upcoming feature flags* que se autolimpam (SE-0362) | [process.md](https://github.com/swiftlang/swift-evolution/blob/main/process.md), [template](https://github.com/swiftlang/swift-evolution/blob/main/proposal-templates/0000-swift-template.md), [commonly_proposed](https://github.com/swiftlang/swift-evolution/blob/main/commonly_proposed.md), [SE-0362](https://github.com/swiftlang/swift-evolution/blob/main/proposals/0362-piecemeal-future-features.md) |
| **Go proposals** | mudanças visíveis; formulário próprio para linguagem | issue curta → discussão → design doc `design/NNNN-nome.md` se preciso | grupo de revisão semanal com atas; estados Incoming/Active/Likely Accept/Likely Decline (uma semana)/Accepted/Declined/Hold | **Go 1**: compatibilidade de fonte com exceções listadas; **GODEBUG**: mudança compatível-mas-quebradora ganha um nome, padrão derivado da linha `go` do `go.mod`, tabela como dado (`internal/godebugs/table.go`), mínimo de dois anos; **loopvar**: "Old code will continue to mean exactly what it means today" | [proposal README](https://github.com/golang/proposal/blob/master/README.md), [go1compat](https://go.dev/doc/go1compat), [compat](https://go.dev/blog/compat), [loopvar](https://go.dev/blog/loopvar-preview) |
| **Elixir** | mailing list (rito formal não verificado) | — | mantenedores | contrato escrito: soft deprecation (só documentação) → hard (aviso) → remoção só em major; a alternativa "MUST exist for AT LEAST THREE minor versions" antes do aviso; `mix format --migrate` | [compatibility and deprecations](https://elixir.hexdocs.pm/compatibility-and-deprecations.html) |
| **Gleam** | centralizado (processo formal não verificado) | anúncios de versão | mantenedor principal | **poda antes da v1**: recurso geral (`use`, v0.25) → depreciação do redundante (`try`, v0.27) com `gleam fix` → remoção (v0.28); v0.32 "Polishing syntax for stability"; a v1 cobre linguagem, compiler, build tool, formatter, LSP | [v0.27](https://gleam.run/news/v0.27-hello-panic-goodbye-try/), [v0.28](https://gleam.run/news/v0.28-monorepos-fast-maps-and-more/), [v0.32](https://gleam.run/news/v0.32-polishing-syntax-for-stability/), [v1](https://gleam.run/news/gleam-version-1/) |
| Zig (contraexemplo) | "Please do not file a proposal to change the language" | — | poucos mantenedores | nenhuma promessa antes de 1.0; cada release quebra | [zig.md](zig.md), [0.14.0](https://ziglang.org/download/0.14.0/release-notes.html) |
| Nim (contraexemplo) | RFCs como issues | — | criador | RFC 456: 268 comentários, votação empatada, fechada pelo autor | [nim.md](nim.md) |

## 2. O que converge

1. **Escopo explícito**: só mudanças de significado passam por processo; bugs contra a norma,
   otimizações e mensagens são livres (Swift, Rust). Sem isso, ou tudo vira processo ou nada
   vira.
2. **Um artefato numerado por decisão**, com seções fixas; **alternativas consideradas** e
   **como ensinar** obrigatórias (Swift, Python).
3. **Implementação antes da aceitação** (Swift): nada é aceito só no papel.
4. **Status fechados e IDs nunca reutilizados** (Swift, Go, PEP).
5. **Memória das rejeições** (`commonly_proposed.md`, "Rejected Ideas").
6. **Compatibilidade como mecanismo, não como promessa**: versão declarada pelo projeto que
   fixa o significado (Go `go.mod`, Rust editions), tabela de mudanças como dado (GODEBUG),
   migração automática por sugestões aplicáveis (`cargo fix`, `gleam fix`, `mix format
   --migrate`), prazo de deprecação escrito (Elixir: três minors).
7. **Podar antes de prometer estabilidade** (Gleam): a v1 cobre uma linguagem já enxuta.

## 3. O Germanio hoje

As peças existem, mas espalhadas ([swift.md](swift.md) › Comparação):

- norma única em `INTENCAO.md` e um Documentation Gate forte em `AGENTS.md`, que exige
  documentação e testes na mesma unidade (mais forte que o protótipo do Swift);
- critérios de julgamento ("Como avaliar uma sintaxe", 8 critérios; checklist de 15 itens),
  sem artefato que registre a resposta;
- decisões espalhadas: `AGENT_STATE.md` D1-D11 (uma linha cada), a tabela da Parte 3 de
  `docs/research/sintaxe-hierarquica.md`, `GERMANIO_EVOLUTION.md` (narrativa depois do fato);
- `GERMANIO_GAPS.md` com **IDs duplicados**: G57, G58 e G59 aparecem duas vezes cada, com
  significados diferentes (linhas 70-75, conferido);
- a mesma pendência em dois lugares (`INTENCAO.md` › Pendências e G62/G63 (antes G58/G59));
- nenhum mecanismo de deprecação (nenhuma ocorrência em `compiler/`, `tooling/`, `runtime/`,
  segundo [elixir.md](elixir.md)), embora a "refatoração retroativa" de `INTENCAO.md` vá
  mudar formas;
- o Gate vale para "qualquer mudança de comportamento" e não distingue bug de design.

## 4. Proposta: GEP (Germanio Evolution Proposal)

Um processo **leve**: o objetivo é concentrar o que hoje está espalhado, não acrescentar
burocracia. Nomes, local e números são sugestões.

### 4.1 Quando um GEP é exigido

Pergunta de corte: **`ge explain` mostraria um fato diferente para o mesmo `.ge`?** Se sim, é
GEP. Além disso, é GEP toda mudança de:

- **sintaxe**: nova palavra, seção, frase, modificador; mudança no layout ou na tabela de
  seções; remoção de forma aceita;
- **semântica**: nova inferência, mudança de padrão (default), de segurança padrão, de regra
  de fusão, de significado de algo que algum `.ge` usa;
- **contrato do runtime**: o que a aplicação gerada publica (nomes de rota, formato de erro,
  esquema do banco, `disponibilize … como "x"`), o análogo Germanio da "ABI" do Swift;
- **stdlib** (funções do nível 3) e **capabilities** (nova capability genérica, mudança de
  contrato de uma existente).

### 4.2 Quando não é exigido

- correção de bug de algo que contradiz `INTENCAO.md` (registrado em `GERMANIO_GAPS.md` como
  BUG); as divergências de [GERMANIO_LESSONS.md](GERMANIO_LESSONS.md) são deste tipo, salvo as
  marcadas "exige decisão";
- erro de digitação em mensagem ou documento;
- desempenho sem mudança de semântica (sob o Performance Gate);
- texto de diagnóstico que não muda o que é aceito;
- tooling que não muda significado (LSP, cor do editor).

Na dúvida, aplica-se a pergunta de corte.

### 4.3 Template (cabe em uma ou duas telas)

```text
GEP-0007 — <título curto>
Status: rascunho | em teste | aceita | rejeitada | retirada | implementada (<commit>) | substituída por GEP-NNNN
Autor: <nome ou agente>   Decide: <responsável pelo design>   Lacunas: G63   Nível: 1 | 2 | 3 | avançado
```

1. **Intenção humana** (até 50 palavras): o que a pessoa quer dizer e hoje não consegue, com a
   frase que ela escreveria.
2. **Hoje**: como se expressa agora e o custo (conceitos técnicos, repetição, lógica nível 3).
3. **Proposta**: exemplo `.ge` hierárquico e a frase plana equivalente.
4. **Semântica**: os fatos produzidos no `ast.App`; o que `ge explain` mostra (declarado ou
   inferido, origem); o que `ge check` recusa.
5. **Avaliação** pelos 8 critérios de "Como avaliar uma sintaxe", na ordem, uma linha cada;
   checklist de novas construções, marcando só o que não é óbvio.
6. **Camada**: domínio, core ou adaptador; a capability genérica criada; **um segundo domínio**
   onde ela serve (teste de generalização).
7. **Como ensinar** (PEP 1): a frase que o tutorial usaria para um leigo.
8. **Compatibilidade**: `.ge` do repositório afetados (com contagem), contrato externo
   afetado, plano (migração na mesma unidade; ou deprecação com prazo e migração automática
   por `ge fmt`/`ge fix`, verificada por reparse).
9. **Alternativas consideradas**: pelo menos uma, e por que perdeu. **Obrigatória.**
10. **Testes normativos**: as linhas novas da tabela de `INTENCAO.md` › Testes normativos e os
    casos de baseline.
11. **Fora do escopo / direções futuras**: sem "faremos".

### 4.4 Status e ciclo de vida

- *rascunho* → *em teste* quando existe implementação em branch com testes (o protótipo do
  Swift) → *aceita* → *implementada* quando o commit que atualiza `INTENCAO.md`, testes e
  exemplos entra na mesma unidade (o Documentation Gate continua valendo).
- *rejeitada* e *retirada* são finais; *substituída por* aponta a sucessora.
- **Um agente nunca aceita a própria proposta.** A aceitação é do responsável pelo design
  (hoje o mantenedor). Um agente pode escrever, implementar em branch e levar a *em teste*.
- Aceita, a norma passa para `INTENCAO.md`; o GEP fica como registro do **porquê** e não é
  normativo (como as proposals antigas no Swift).

### 4.5 Identificadores

Numeração sequencial, **nunca reutilizada**, nem depois de rejeição ou retirada. A mesma regra
vale para `GERMANIO_GAPS.md` (violada em G57-G59 até a renumeração para G61-G64 em 2026-09-28) e para os códigos de diagnóstico
([DIAGNOSTICS.md](DIAGNOSTICS.md) §7).

### 4.6 Lista de rejeitados

Um índice curto (`rejeitadas.md`, no espírito de `commonly_proposed.md`): título, uma linha de
motivo, o GEP. Começa com o que os estudos já classificaram como "não copiar": macros e DSLs
de usuário, sintaxe configurável, formatter com opções, prioridades de merge, continuação de
linha por recuo, layout pela coluna do primeiro token, linguagem natural livre, LLM no
compilador (a ajuda do `ge` já diz que "não utiliza LLM no compilador"). Cada item do índice
tem a pergunta "que problema do Germanio isso resolveria?" respondida.

### 4.7 Encaixe nos documentos existentes

- `GERMANIO_GAPS.md` continua sendo o registro de lacunas; uma lacuna de linguagem aponta para
  o GEP que a resolve.
- `INTENCAO.md` › Pendências lista só títulos com o número do GEP ou da lacuna, para que a
  pendência exista em um único lugar.
- `AGENT_STATE.md` › decisões cita o GEP quando a decisão é de linguagem.
- `GERMANIO_EVOLUTION.md` continua como narrativa medida; cada etapa cita o GEP.
- Documentation Gate, passo 1: "se a mudança está no escopo de GEP, ele existe e está *em
  teste* ou *aceita*".

Estas integrações alteram documentos normativos e de processo; ficam para decisão, não foram
feitas.

## 5. Compatibilidade: quando o Germanio precisar

Hoje todo `.ge` relevante está no repositório, e migrar na mesma unidade de trabalho é mais
simples e já funcionou ([swift.md](swift.md): EVITAR modos de linguagem e flags enquanto for
assim). O gatilho para os mecanismos abaixo é o primeiro `.ge` de terceiros:

| Mecanismo | Origem | Forma para o Germanio | Classe |
|---|---|---|---|
| versão declarada pelo projeto | `go.mod`, editions | uma linha escrita por `ge init` (`versão germanio 1`); `ge explain` diz "esta frase segue o significado da versão 1"; o leigo não escolhe flag nenhuma | ADOTAR quando houver terceiros |
| tabela de mudanças como dado | GODEBUG `table.go` | nome, versão em que mudou, comportamento antigo; teste que exige a documentação em dia | ADOTAR com o item acima |
| deprecação com prazo escrito | Elixir, PEP 387 | a forma nova existe por N versões antes do aviso; aviso educativo; remoção só em versão maior | ADOTAR |
| migração automática verificada | `cargo fix`, `gleam fix`, `mix format --migrate` | `ge fix` aplica só sugestões **automáticas** (fatos idênticos), ou a migração declarada no GEP, provada por reparse | ADAPTAR |
| poda antes de estabilizar | Gleam | dois front-ends e formas equivalentes (plana e hierárquica são intencionais; o resto, avaliar) antes de declarar uma "v1" | INVESTIGAR |
| flags de recurso futuro | SE-0362 | só com `.ge` de terceiros | INVESTIGAR |
