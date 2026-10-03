# Textos de lançamento e primeiras issues

Documento de trabalho (2026-09-29), complemento de [LANCAMENTO.md](LANCAMENTO.md). Não é norma
nem documentação da linguagem. Os textos só afirmam o que o repositório demonstra hoje.

## Antes de publicar qualquer texto

A ordem importa: quem clicar no link precisa conseguir rodar o Germanio em poucos minutos.

1. CI verde no `master` (o selo aparece no topo do README).
2. Release `v0.7.0` com binários (`docs/RELEASING.md`). Sem binário, metade das pessoas
   desiste antes de testar.
3. Social preview enviado (`assets/social/social-preview.png`, ver LANCAMENTO.md).
4. As issues de "Primeiras issues" (abaixo) abertas, com o label `good first issue`.
5. Reservar o dia da publicação para responder comentários. A primeira hora decide o
   alcance no Hacker News e no Reddit.

Publique um canal por dia, não todos juntos: cada um traz perguntas diferentes, e as
respostas melhoram o texto do próximo.

## TabNews (pt-BR)

**Título:** Criei uma linguagem em que você descreve o sistema e ele roda: Germanio

**Corpo:**

> Há um tempo venho construindo o Germanio, uma linguagem de programação open source feita
> em Go. A ideia é simples de dizer e difícil de fazer: você descreve **o que existe, quem
> pode fazer o quê, o que acontece e o que aparece**, e o Germanio entrega a aplicação web
> funcionando: banco, validação, login, permissões, estados, API REST e páginas.
>
> Este é um app completo de tarefas, em que cada pessoa só vê as próprias:
>
> ```
> tarefas
>     tem
>         título obrigatório até 120
>         prazo data
>     pertence a usuario
>     começa aberta
>     pode
>         concluir
>         reabrir
>     acesso
>         usuario
>             criar seus
>             ver seus
>             concluir seus
> ```
>
> (o arquivo inteiro, com login e a página, tem 38 linhas; o GIF no README mostra rodando)
>
> Três decisões que eu gostaria de discutir com vocês:
>
> 1. **Sem IA dentro.** O mesmo `.ge` sempre significa o mesmo programa. `ge explain`
>    mostra cada fato que o compilador inferiu e de qual linha ele veio.
> 2. **Segurança como padrão.** CSRF, senha com hash, proteção contra SSRF, uma pessoa não vê
>    o dado de outra: nada disso precisa ser escrito.
> 3. **Quando algo não cabe, a linguagem cresce, não o app.** Para testar isso, reimplementei
>    conceitos do GitLab em `.ge` (grupos, projetos, Git por HTTP, issues, merge requests e
>    jobs de CI executados pelo gitlab-runner oficial). Cada parede virou uma capability
>    genérica, registrada em `GERMANIO_GAPS.md`.
>
> Ainda é pré-1.0 e as palavras-chave são só em português. Críticas são muito bem-vindas,
> principalmente de quem já tentou criar um sistema simples e travou na pilha web.
>
> Repositório: https://github.com/flaviokalleu/germanio

## Show HN (inglês)

**Título** (até 80 caracteres):
`Show HN: Germanio – describe data, permissions and pages; get a running web app`

**Primeiro comentário** (o autor comenta logo após publicar):

> Hi HN, I'm the author. Germanio is an open-source language, written in Go, where a program
> says what exists, who may do what, what happens and what appears. The runtime provides the
> database, validation, login, permissions, state transitions, a REST API and server-rendered
> pages.
>
> The GIF in the README is the whole 38-line to-do app from the "Show me" section running:
> sign-up, creating and completing tasks, and a second user who sees none of the first user's
> tasks. Nobody wrote an authorization check; `ver seus` ("see their own") is the rule.
>
> Things I'd love feedback on:
>
> - Determinism instead of AI: `ge explain tarefas app.ge` prints every inferred fact with the
>   line it came from. There is no model in the loop at compile time or runtime.
> - The escape hatch: when the intent layer cannot say something, the rule is to add a
>   generic capability to the core, never app-specific Go. I stress-tested this by
>   reimplementing GitLab concepts (groups, Git smart HTTP, issues, merge requests, CI jobs run
>   by the official gitlab-runner); every gap is logged in GERMANIO_GAPS.md.
> - Keywords are Portuguese only for now. I know that limits the audience; the reasons and
>   the plan are in the README.
>
> It is pre-1.0 and the language can still change. Happy to answer anything.

Dicas: publicar numa terça ou quarta, entre 8h e 10h no horário de Nova York (9h–11h em
Brasília no horário de verão americano). Não pedir votos a ninguém: o HN penaliza.

## Reddit

**r/ProgrammingLanguages** — o público quer design, não marketing.

- Título: `Germanio: an intent-oriented, indentation-based language where the compiler shows the source of every inferred fact`
- Corpo: o parágrafo das decisões do Show HN, mais um link para `docs/INTENCAO.md` (a norma)
  e para `docs/gep/` (como a linguagem evolui). Pergunta final sugerida: "How would you
  handle the cliff between the declarative layer and explicit logic?"

**r/golang** — o ângulo é a implementação.

- Título: `I built a language runtime in Go that turns declarative app descriptions into web apps (SQLite in pure Go, no cgo)`
- Corpo: arquitetura (`docs/ARCHITECTURE.md`), SSRF no dialer, Git sem shell, suíte de
  benchmarks contra a mesma app escrita em Go (`bench/`).

**r/brdev** — mesmo texto do TabNews, encurtado.

## Vídeo curto (Shorts, Reels, TikTok)

Roteiro de 45 segundos, gravado em tela:

1. (0–5 s) "Um sistema com login e permissões em 38 linhas. Sem framework."
2. (5–20 s) Rolar o `app.ge`, lendo em voz alta `tarefas`, `pertence a usuario`,
   `ver seus`.
3. (20–35 s) `ge run app.ge`, cadastro, criar tarefa, concluir.
4. (35–45 s) Outra pessoa entra e não vê nada. "Ninguém escreveu essa regra de segurança:
   ela vem de `ver seus`." Link na bio.

## Primeiras issues (label `good first issue`)

Todas foram vistas ao gravar a demonstração do README; nenhuma exige conhecer o compilador
inteiro. Abrir cada uma com o trecho de código indicado.

1. **A mensagem depois de uma ação diz "Feito: Concluir".** Deveria dizer o que aconteceu
   com o dado, concordando em gênero: "Tarefa concluída", "Pedido pago". O estado de destino
   está em `ast.Transition`; a concordância já existe em `ast.Entity.Concorda`.
   Onde: `runtime/servidor/paginas.go`, `case "acao"`.
2. **Estados aparecem sem acento ("concluida").** O nome do estado é derivado do verbo e
   normalizado; a tela deveria mostrar a grafia de pessoa ("concluída"), como já acontece
   com os campos (`Field.Label`). Onde: `compiler/parser` (derivação do estado) e
   `runtime/servidor/paginas.go` (`display` das células com selo).
3. **O filtro por estado é uma caixa de texto livre.** Com `permita filtrar por estado`, o
   filtro deveria oferecer os estados declarados (aberta, concluída) numa lista.
4. **Campos de referência mostram o nome do dado sem acento ("Usuario").** O rótulo deveria
   usar a grafia de pessoa quando o autor a escreveu, ou "Usuário" pela regra de acentos já
   usada no léxico.
5. **`ge check` deveria avisar quando `backend/` ou `frontend/` usam primitivas técnicas**
   (`chamar`, `requisicao`, `responder`, `ambiente.ler`). É a regra "primitiva disponível não
   é autorização para usá-la" de `docs/INTENCAO.md`, verificada pelo compilador. Aviso, não
   erro, com a sugestão do nível de intenção quando houver.
6. **G23: blocos de tela e tema ainda ignoram tokens desconhecidos em silêncio.** Devem falhar
   com arquivo:linha e sugestão, como os corpos de rota e função já fazem.

Para cada issue: o que acontece hoje, o que deveria acontecer, onde começar e como testar
(`go test ./...`, e `ge check` no exemplo afetado).
