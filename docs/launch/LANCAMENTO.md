# Lançamento público: estado, pendências do proprietário e plano editorial

Documento de trabalho (2026-09-28). Não é norma nem documentação da linguagem.

## O que já foi configurado

**GitHub About** (aplicado com `gh repo edit`):

- *Description*: "Open-source, intent-oriented programming language built in Go: describe
  data, people, permissions, states and pages; get a running web application. Deterministic,
  no AI inside."
- *Website*: vazio. O site em `examples/site-germanio` ainda não está hospedado; a URL
  anterior apontava para o repositório do Flang.
- *Topics* (19; o limite do GitHub é 20, com letras minúsculas, números e hífen, até 50
  caracteres cada): programming-language, compiler, golang, language-design,
  declarative-programming, intent-oriented, full-stack, web-development, developer-tools,
  beginner-friendly, deterministic, parser, static-analysis, backend, frontend,
  developer-experience, portuguese, open-source, germanio. Saíram `flang`, `dsl`,
  `interpreter` e `declarative-language`. Ficaram de fora, de propósito, `code-generation` (o
  Germanio executa o modelo; não gera código) e `ast` (não ajuda a busca).
- Labels `idea` e `gep`, usados pelos modelos de issue.
- Relato privado de vulnerabilidades ligado (o `SECURITY.md` depende disso).

## O que só o proprietário pode fazer

1. **Social preview**: Settings › General › Social preview › *Edit* › enviar
   `assets/social/social-preview.png` (1280×640, 416 KB; o GitHub pede pelo menos 640×320 e
   menos de 1 MB). A API do GitHub não permite enviar essa imagem.
2. **Publicar os commits** (`git push`): o About e os links do README apontam para arquivos
   que por enquanto só existem localmente.
3. **Primeiro release do Germanio**: quando decidir, siga `docs/RELEASING.md`
   (`git tag -a v0.7.0 …`, `git push origin v0.7.0`). Os releases v0.2–v0.6 continuam
   publicados como Flang; o CHANGELOG explica.
4. **Site**: hospedar `examples/site-germanio` (qualquer servidor que rode `germanio run`, ou o
   Dockerfile) e, com o domínio definido, configurar `GERMANIO_URL_PUBLICA`
   (`docs/DEPLOY.md`), que ativa `canonical`, `og:url` e o sitemap com URLs absolutas. Depois,
   preencher o *Website* do About.
5. **Extensão do VS Code**: publicar `vscode-germanio` no Marketplace e no Open VSX (exige
   conta de publisher). Hoje ela só se instala a partir do código.
6. **Discussions**: desligado. Ligue só se houver quem responda; o site e o CONTRIBUTING
   apontam para Issues.

## Intenção de busca (pesquisa de 2026-09-28)

- **"easiest / beginner / simple programming language"**: dominada por listas ("as 10
  linguagens mais fáceis") que respondem Python e JavaScript para quem quer aprender a
  programar em geral ([exemplo](https://www.guvi.in/blog/easiest-programming-languages-for-beginners/),
  [exemplo](https://www.devpebble.com/blog/easiest-programming-language/)). O Germanio não
  compete aí e não deve tentar: quem busca isso quer uma primeira linguagem de carreira. O
  público do Germanio é quem quer **construir uma aplicação** sem aprender a pilha web; o
  conteúdo deve falar disso, não de "a linguagem mais fácil".
- **"declarative full-stack / web app without framework boilerplate"**: o vizinho mais
  próximo é o [Wasp](https://wasp.sh/resources/2026/02/24/best-frameworks-web-dev-2026), uma
  DSL que gera React/Node. Diferenças reais do Germanio: runtime próprio sem pilha JavaScript,
  público que não programa, determinismo com `ge explain`, palavras-chave em português. Não
  escrever comparação com o Wasp sem estudá-lo de fato.
- **"intent-oriented programming"**: termo de nicho, hoje associado sobretudo a projetos
  nativos de IA ([IntentLang](https://github.com/l3yx/intentlang), um
  [artigo sobre IOPL](https://www.researchgate.net/publication/400563484_Intent-Oriented_Programming_Language_IOPL_A_Deterministic_Foundation_for_Intent-Centric_and_AI-Assisted_Software_Development)).
  O "determinístico, sem IA dentro" do Germanio é uma diferença verdadeira e deve aparecer
  sempre junto do termo.
- **"programming without AI"**, **"Go compiler"**, **"language design"**: interesse técnico;
  o caminho são artigos de engenharia (abaixo), não páginas de palavras-chave.

Regra: nenhuma página existe só para capturar uma busca. Cada artigo precisa ter conteúdo
que se sustente sozinho, com código que compila (`tooling/doctest`) e números com contexto.

## Plano editorial

Cada artigo lista a evidência que já existe no repositório. Nada de resultados inventados.

| Artigo | Ângulo | Evidência disponível |
| --- | --- | --- |
| Building an intent-oriented programming language in Go | do `.ge` ao modelo resolvido e ao runtime; por que executar o modelo em vez de gerar código | `docs/ARCHITECTURE.md`, `docs/research/performance/ARQUITETURA.md` |
| Can programming be simple without AI? | determinismo, `ge explain`, erros educativos; IA como ajuda opcional, nunca como intérprete | `docs/INTENCAO.md`, `tooling/explicar` |
| Designing an indentation-based language: what Python, Haskell, YAML and CUE taught us | off-side rule, seções fechadas, a redução a frases planas e as dívidas que ela deixou | `docs/research/languages/SYNTAX_COMPARISON.md`, `GERMANIO_LESSONS.md` |
| How Germanio turns intent into deterministic software | bloco → fatos → `ast.App`; fusão e conflitos com as duas origens | `compiler/parser/hierarquia.go`, `hierarquia_test.go` |
| Full-stack applications without framework boilerplate | o exemplo de tarefas do README, linha a linha, contra o que a mesma app exige numa pilha comum | `README.md`, `examples/` |
| What we learned implementing GitLab concepts in Germanio | cada parede virou capability genérica (trabalho remoto, endereços, membros); o adaptador do gitlab-runner | `GERMANIO_EVOLUTION.md`, `GERMANIO_GAPS.md`, `examples/gitlab-foss` |
| Designing a programming language for people who are not programmers | o teste do leigo, a skill de simplicidade, o que o Germanio deixa de exigir | `skills/germanio-simplicity/SKILL.md`, `GERMANIO_LESSONS.md` › subtração |
| What does an abstraction cost? Germanio vs the same app in Go | só depois de um baseline em máquina ociosa; mostrar também os custos altos encontrados | `bench/`, `docs/research/performance/AUDITORIA.md` |

## Teste do visitante (30 segundos) — conferido no README

| Pergunta | Onde se responde |
| --- | --- |
| O que é? | primeira frase, abaixo do logo |
| Por que existe? | "Why Germanio exists" |
| Como é o código? | "Show me", na primeira tela, com o que o `ge explain` mostra |
| Como experimentar? | "Try it": quatro comandos |
| É real e ativo? | badge da CI, CHANGELOG, testes, o teste de estresse do GitLab, "Project status" com os limites |
| Onde está a documentação? | "Documentation", separada para quem aprende e para quem desenvolve |
| Como participar? | "Contributing" e CONTRIBUTING.md |
