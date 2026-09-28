# GERMANIO_EVOLUTION

Como o GitLab está mudando o Germanio. Cada linha: necessidade real → abstração genérica.

## Medidas (GitLab: identidade + grupos + projetos + repositórios)

| Versão | Linhas `.ge` (sem brancos/comentários) | Cobre |
|--------|---------------------------------------:|-------|
| Imperativa (`research/gitlab/baseline-imperativo`) | 832 | usuários, tokens, grupos, projetos (sem git, sem testes) |
| Intenção v1 (commit d6ce7a5) | 152 | o anterior + repositórios, clone/push, branches protegidas, navegação de commits, herança de papéis; 3 testes E2E |
| Intenção v2 + skill de simplicidade | 235 (36 são só vocabulário de compatibilidade) | o anterior + issues, comentários, labels, confidencialidade, estados; domínio em português; 4 testes E2E |

Das 152 linhas, ~40 são lógica de produto explícita (nível 3: caminhos, visibilidade,
último owner, branch protegida); o resto é declaração.

## Linha do tempo

| Etapa | Necessidade no GitLab | Abstração criada (genérica) |
|-------|-----------------------|-----------------------------|
| 1 | API REST real | `rota` com `requisicao`, `responder`, erros com posição (nível 4) |
| 2 | 100+ linhas repetindo busca/404/serialização | `tenha` + operações derivadas por dado |
| 3 | validação espalhada | `cada X tem` com tipo pelo nome e modificadores |
| 4 | bcrypt, digest de token, expiração | `senha`, `segredo prefixo`, `expira em`, `revogavel` |
| 5 | login/PAT/OAuth/bloqueio manual | `tenha login` + `login usa/aceita/bloqueia/exige` |
| 6 | `se nivel < X falhar(403)` em cada rota | `tenha papeis`, `membros com papel`, `herda membros`, `<papel> pode …` |
| 7 | email visível a todos | modificador `privado` |
| 8 | regras que recusam (último owner) | `antes de <ação>` + `recuse` |
| 9 | repositório por projeto, clone/push HTTP | `projeto tem repositório`, `baixar/enviar código` |
| 10 | API compatível com clientes GitLab | `disponibilize … como "projects"`, `integração em`, `mensagens em inglês` |

## Princípios confirmados na prática

- Regra específica do produto continua explícita (último owner, branch protegida).
- Segurança nunca depende do autor: senha, CSRF, 404 para o invisível, cadastro sem admin,
  nada criado em nome de outra pessoa.
- Poucas palavras novas no léxico: as frases são reconhecidas pelo parser (linhas na coluna 1),
  não viraram keywords.

## Skill de simplicidade (skills/germanio-simplicity)

Aplicada às issues, removeu do `.ge`: `issue.atualizar(...)`, `state começa com "opened"`,
`closed_at`, `closed_by_id`, `lista de usuarios`, `autor_id` e a política confidencial em
booleanos. Capabilities genéricas criadas:

| Necessidade | Capability |
|-------------|------------|
| abrir/fechar/reabrir, bloquear/desbloquear | máquina de estados: `X começa <estado>` + `X pode <verbo>` (particípio com gênero; re-/des- voltam ao inicial; carimbos `<estado>_em`, `<estado>_por`) |
| confidencial | `X pode ser <adjetivo>` (flag) + `X <adjetivo> pode ser vista por …` |
| autor, responsáveis | pessoas inferidas em `tem` (autor → uma pessoa dona; responsaveis → várias) |
| labels do projeto na issue | `tem` reivindicado por ancestral e descendente → o ancestral é dono, o descendente guarda lista |
| comentar issues | verbo derivado do nome do filho (comentar → comentarios) |
| nomes da API GitLab | `vocabulário da integração` separado do domínio |
| #1, #2 por projeto | `numero por projeto` |

Bug de segurança encontrado no caminho: editar um registro reatribuía o autor para quem editava.

## Sintaxe hierárquica e contextual (2026-09-28)

Norma em `docs/INTENCAO.md` › Sintaxe hierárquica; pesquisa em `docs/research/sintaxe-hierarquica.md`.
O domínio inteiro do GitLab foi migrado para blocos; os mesmos testes de execução (inclusive o
gitlab-runner oficial) passam sem alteração, e o aplicativo resolvido é idêntico ao da forma plana,
exceto `administrar` num bloco, que vale para a coleção (inclui criar) — sem efeito prático no
GitLab, porque os papéis que ganharam criar já podiam criar por serem developer ou superior.

Resultado **deste experimento** (não são metas da linguagem; ver `docs/INTENCAO.md` ›
Como avaliar uma sintaxe):

| Medida (domínio GitLab sem a compatibilidade) | Plano | Hierárquico |
|---|---:|---:|
| Linhas de código | 265 | 325 (+23%) |
| Palavras | 772 | 598 (−23%) |
| Menções a nomes de dados | 141 | 59 (−58%) |

Leitura: mais linhas aqui criam estrutura visual (cada aspecto com título, um item por
linha); a queda de repetição vem de informação que a hierarquia já determina. Nenhum desses
números, sozinho, prova que a forma é melhor; a avaliação qualitativa (conceitos técnicos,
carga cognitiva, previsibilidade da hierarquia, clareza, determinismo, leitura rápida) está
em `docs/research/sintaxe-hierarquica.md` › Parte 4.

Problemas gerais revelados no caminho: linhas desconhecidas eram ignoradas em silêncio;
tab contava como 2 espaços; recuo sem pai não era diagnosticado; `administrar` governava dados
que só se referiam opcionalmente ao alvo (falha de segurança); `ge fmt` não formatava a camada
de intenção; `ge explain` não mostrava a origem dos fatos; o singular de estrangeirismos
(`tokens`) era adivinhado.

## Diretriz de eficiência (2026-09-28)

Nova norma: "Simples para o humano. Eficiente para a máquina." (`docs/INTENCAO.md` ›
Eficiência). Desempenho, memória, concorrência e carga passam a ser requisitos da linguagem,
com a ordem correto → seguro → mensurável → rápido. Por decisão explícita, nenhum budget
numérico é fixado antes do baseline medido. Começou pela auditoria do caminho de execução
(`docs/research/performance/AUDITORIA.md`) e por uma suíte permanente de benchmarks
(`bench/`) que compara Germanio com Go direto. A intenção de concorrência ("processe em
paralelo", "aceite muitas conexões") fica como pendência de design até ter semântica,
limites e testes.
