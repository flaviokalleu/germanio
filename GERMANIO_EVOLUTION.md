# GERMANIO_EVOLUTION

Como o GitLab está mudando o Germanio. Cada linha: necessidade real → abstração genérica.

## Medidas (GitLab: identidade + grupos + projetos + repositórios)

| Versão | Linhas `.ge` (sem brancos/comentários) | Cobre |
|--------|---------------------------------------:|-------|
| Imperativa (`research/gitlab/baseline-imperativo`) | 832 | usuários, tokens, grupos, projetos (sem git, sem testes) |
| Intenção (`examples/gitlab-foss`) | 152 | o anterior + repositórios, clone/push, branches protegidas, navegação de commits, herança de papéis; 3 testes E2E passando |

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
