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
