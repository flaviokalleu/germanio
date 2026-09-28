# Estudo dirigido: Lua e Luau — linguagem pequena, embutível, e tipos graduais com sandbox para milhões de iniciantes

**Data:** 2026-09-28
**Status:** pesquisa, sem força normativa (a norma é `docs/INTENCAO.md`).
**Pergunta principal:** como o Lua se mantém pequeno por décadas (processo, economia de
conceitos, uma estrutura de dados), e como o Luau (Roblox) acrescentou tipos graduais e sandbox
para uma base enorme de criadores iniciantes sem quebrar o que existia?

## Fontes consultadas

- lua.org, "About" — https://www.lua.org/about.html
- Ierusalimschy, de Figueiredo, Celes, "The Evolution of Lua", HOPL III, 2007 — https://www.lua.org/doc/hopl.pdf (texto extraído e lido nos trechos citados)
- Luau, "Why Luau?" — https://luau.org/why
- Luau, "Sandboxing" — https://luau.org/sandbox
- Luau, "Type checking" — https://luau.org/types

Não verificados nesta passagem: o novo solver de tipos do Luau e números de usuários do Roblox.

---

## Matriz (compacta)

| Item | Lua (e Luau quando indicado) | Relevância para o Germanio |
|---|---|---|
| Objetivo original | Linguagem de extensão para aplicações (Tecgraf/PUC-Rio): "powerful, efficient, lightweight, embeddable" (About) | o Germanio também é uma linguagem sobre um host (o runtime) |
| Filosofia | "Meta-mechanisms for implementing features, instead of providing a host of features directly in the language" (About); "simplicity is our most important asset" (HOPL) | tensão com o Germanio: ver "O que não copiar" |
| Sintaxe | Pequena, com palavras (`then`, `end`, `local`, `function`), pouca pontuação | — |
| Gramática | Curta, publicada no manual de referência | o `SPEC.md` pode aspirar ao mesmo tamanho |
| Lexer/Parser | Parser de passo único que gera bytecode direto, sem AST; o Luau precisou de uma AST e de um compilador multi-passo para análise ("Lua's integrated parser-compiler design was insufficient", Why Luau) | confirma a AST do Germanio como base de ferramentas |
| Runtime | VM de registradores; implementação de ~32 000 linhas de C, tarball de 390 KB comprimido (About, Lua 5.5.1) | contraste com os 27,4 MB do Germanio |
| Memory management | GC incremental (5.1) e geracional (5.4) (não verificado nesta passagem para 5.4) | — |
| Sistema de tipos | Dinâmico; Luau: gradual e estrutural, três modos por arquivo (`--!nocheck`, `--!nonstrict` padrão, `--!strict`); em não estrito, o que não se sabe vira `any` (luau.org/types) | ver "Para o Germanio" |
| Standard library | Mínima, por desenho | — |
| Package manager | Nenhum oficial (LuaRocks é externo) | — |
| Formatter / LSP | Externos (StyLua, luau-lsp) (não verificado nesta passagem) | — |
| Diagnostics | Luau: análise estática e lint próprios (Why Luau) | — |
| Evolution process | Três autores; "we include a new feature in Lua only when all three of us agree; otherwise, it is left for the future. It is much easier to add features later than to remove them" (HOPL) | processo leve e conservador, compatível com o GEP |
| Backward compatibility | Lua remove o que foi substituído: "we considered simplicity and elegance more important than compatibility"; a Tecgraf nunca migrou de 3.2 para 4.0 (HOPL). Luau congelou a base em Lua 5.1 porque não podia quebrar milhões de criadores (Why Luau) | ver "Principais problemas" |

### Principais acertos
- **Economia de conceitos.** Tabelas são "the sole data-structuring mechanism" (HOPL): array,
  registro, objeto, módulo e namespace são a mesma coisa.
- **Coroutines como mecanismo único** para geradores, cooperação e fluxo assíncrono, em vez de
  threads ou callbacks (HOPL, Lua 5.0).
- **Tamanho como disciplina.** Unanimidade entre três autores para cada recurso.
- **Luau: segurança pelo host, não pelo usuário.** `io` removido ("gives access to files and
  allows running processes"), `os`, `debug` e `package` restritos, bibliotecas internas
  somente leitura, **uma tabela global por script** ("all scripts are isolated from each
  other" sem VMs separadas), destrutores só para o host, e um **mecanismo de interrupção global**
  que o código Luau garante chamar "eventually", usado pelo Roblox para matar scripts que passam
  de 10 s no Studio (luau.org/sandbox).
- **Luau: tipos graduais sem quebrar ninguém.** O padrão (`nonstrict`) não acusa o que não
  consegue provar; o `strict` é opt-in por arquivo.

### Principais problemas
1. **Mecanismos sem políticas criam dialetos.** Cada projeto Lua monta sua orientação a objetos
   e seu sistema de módulos; o próprio HOPL registra que, "despite our 'mechanisms, not policy'
   guideline", tiveram de definir políticas de módulos no 5.1.
2. **Remover sem compatibilidade fragmentou usuários** (Tecgraf presa ao 3.2) e hoje há dialetos
   de fato: 5.1 (LuaJIT, Luau), 5.2, 5.3, 5.4.
3. **Uma estrutura para tudo tem arestas**: tabelas usadas como array com "buracos" (`nil`) têm
   comprimento mal definido (não verificado nesta passagem, fato conhecido do operador `#`).
4. **Luau `nonstrict` com `any`** esconde erros exatamente no modo padrão.

### Complexidade acumulada
Baixa na linguagem; transferida para o ecossistema (cada host decide biblioteca, módulos,
sandbox).

---

## Sintaxe (perguntas obrigatórias)

| Pergunta | Lua / Luau | Germanio hoje |
|---|---|---|
| Dados | tabelas; Luau acrescenta anotações de tipo estruturais | `tem`, com tipo e restrição por campo |
| Funções | `function`, valores de primeira classe | ausentes do domínio |
| Módulos / imports | `require` retorna uma tabela | `importar` pasta ou nomes |
| Relações | referências entre tabelas | nomes resolvidos |
| Estado | mutável | estados declarados |
| Erros | `error`/`pcall` | diagnostics na compilação |
| Concorrência | coroutines cooperativas | runtime administra |
| Null | `nil` (apagar campo = atribuir `nil`) | "ausente"; `obrigatório` |
| Boilerplate | baixo | derivado |
| Projetos grandes | Roblox precisou de tipos e lint para escalar (Why Luau) | `ge check`, `ge explain` |

Custo cognitivo: Lua ensina poucos conceitos (tabela, função, coroutine, metatable), mas as
metatables são o ponto em que a linguagem pequena vira grande. O Germanio não expõe nada
equivalente a metatables no domínio, e não deve.

## Performance e memória (comparação com `performance/AUDITORIA.md`)

- **Tamanho.** Lua inteira: ~32 000 linhas de C, interpretador de 293 KB mais biblioteca de
  484 KB (About). Germanio: binário de 27,4 MB (`germanio build` equivalente), dos quais +9,1 MB
  só do WhatsApp; 213 pacotes no init (AUDITORIA §2.2–2.3). Não é comparação justa (o Germanio
  inclui servidor HTTP, banco, auth e HTML), mas mostra o que "embutível" exige: o núcleo não
  arrasta capabilities que o programa não usa.
- **Isolamento sem VM separada.** O Luau isola scripts com uma tabela global por script sobre
  bibliotecas somente leitura. O interpretador do Germanio executa lógica de vários requests;
  isolar escopos sem custo de uma instância por request é a mesma ideia.
- **Orçamento de execução.** Luau: interrupção global que o host controla (tempo). Germanio hoje
  (IMPLEMENTADO): `maxIterations = 10000` **por laço** e `maxCallDepth = 200`
  (`runtime/interpreter/interpreter.go:23`, `:447`). Consequências: um laço legítimo sobre
  10 001 registros falha, e dois laços aninhados permitem 10^8 iterações; não há teto de tempo
  nem cancelamento por request (AUDITORIA §3.4).

## Para o Germanio

| # | Classe | Lição | Problema do Germanio | Arquivo | O que remove da cabeça do programador |
|---|---|---|---|---|---|
| 1 | ADOTAR | "Only when all agree; easier to add later than to remove" como critério de aceitação de GEP (subtração por padrão) | risco de acumular formas | `docs/gep/README.md` | — |
| 2 | ADOTAR | Orçamento de execução por request controlado pelo host (interrupção do Luau): tempo/passos totais ligados ao `context` do request, no lugar de limite fixo por laço | limite de 10 000 por laço rejeita casos legítimos e não limita aninhamento; sem cancelamento (AUDITORIA §3.4) | `runtime/interpreter/interpreter.go`, `runtime/servidor` | calcular limites ou temer travar o servidor |
| 3 | ADOTAR | Sandbox por construção: a lógica `.ge` não tem acesso a arquivos, processos nem rede fora das capabilities declaradas; bibliotecas internas imutáveis | `eval` exige admin, `chamar` tem proteção SSRF, mas a superfície não está descrita como sandbox normativa | `runtime/interpreter`, `docs/INTENCAO.md` | pensar em segurança ao escrever lógica |
| 4 | ADAPTAR | Uma estrutura de dados no domínio (tabela do Lua) — no Germanio, a entidade — e nenhuma outra forma de agregado exposta | a lógica ainda tem listas e mapas soltos (`map[string]any`) | `runtime/interpreter` | escolher entre estruturas |
| 5 | ADAPTAR | Rigor gradual por arquivo (`--!strict`) só para a camada avançada; no domínio, o rigor máximo é o único modo | risco de criar "modo leniente" que esconde erro, como o `nonstrict` com `any` | `compiler/parser`, `integracoes/` | escolher modo |
| 6 | ADOTAR | AST separada do gerador (lição que o Luau teve de aprender ao sair do parser de passo único) | já é assim; manter para LSP | `compiler/ast` | — |
| 7 | EVITAR | "Mecanismos, não políticas" no domínio: o `.ge` fornece políticas (autorização, estados, páginas); mecanismos ficam no core | a filosofia do Lua levaria a cada app montar sua autorização | `docs/INTENCAO.md` | montar infraestrutura |
| 8 | EVITAR | Remover sem migração (Tecgraf presa ao 3.2) e dialetos de versão | exigir `ge fix`/migração automática em toda quebra | `tooling/formatter`, `docs/gep` | reescrever à mão ao atualizar |
| 9 | INVESTIGAR | Núcleo embutível: `runtime` mínimo sem capabilities, com as demais ligadas só se declaradas | 27,4 MB e 213 pacotes para qualquer app | `runtime/engine.go`, `cli/cli.go` | — |
