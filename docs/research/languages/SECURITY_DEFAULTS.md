# Defesas por padrão: classes de erro que linguagens e frameworks impedem, e como o Germanio as trata hoje

Data: 2026-09-28
Status: pesquisa transversal (segunda rodada do estudo do ecossistema)
Escopo: confere no código do `runtime/` o que o Germanio já defende, aponta lacunas e propõe
capabilities gerais. NÃO corrige nada (o orquestrador corrige). Verificado lendo
`runtime/servidor`, `runtime/banco`, `runtime/auth`, `runtime/interpreter`,
`runtime/httpclient` e `runtime/git`, e reproduzindo em execução com `demo/plano/inicio.ge`.

Fontes externas (conceitos, não código):
- OWASP API Security Top 10 (BOLA/IDOR, mass assignment, SSRF) — https://owasp.org/API-Security/editions/2023/en/0x11-t10/
- Mass assignment / Rails-GitHub 2012 — https://en.wikipedia.org/wiki/Mass_assignment_vulnerability e http://homakov.blogspot.com/2012/03/how-to.html
- CWE-89 (SQLi), CWE-918 (SSRF), CWE-639 (IDOR), CWE-915 (mass assignment).

---

## VULNERABILIDADES REAIS ENCONTRADAS HOJE (não corrigidas aqui)

As três primeiras estão no **dialeto de aplicação legado** (`autenticacao` + `telas` +
`/api/<modelo>`), que é o que `germanio new` e os exemplos/demos ainda geram e o runtime ainda
compila. O dialeto de intenção (`runtime/servidor/intencao.go`, com `Can`/`Owns`/CSRF/
`serializeFor`) é o alvo documentado (`GERMANIO_GAPS.md` D1) e **não** tem estes furos — mas o
código legado continua no binário e serve as afirmações de segurança do `CLAUDE.md` ("auth
bypass fixed") como falsas para esse caminho. Reproduzido com `demo/plano/inicio.ge`, que usa
`autenticacao`.

### V1 — Injeção de SQL por `ORDER BY` na API legada (CWE-89) — P0

- **Onde:** `runtime/banco/banco.go:404`
  `fmt.Sprintf("SELECT * FROM %s%s ORDER BY %s %s LIMIT %d OFFSET %d", q(modelo), whereSQL, q(params.Ordenar), params.Ordem, ...)`.
  `params.Ordem` é interpolado **cru**, sem placeholder e sem lista branca.
- `Banco.Listar` (`banco.go:335`) só define defaults; nunca valida `Ordem` nem confere
  `Ordenar` contra as colunas. Contraste: `Banco.Filtrar` (`runtime/banco/consulta.go:206-214`)
  valida `order` contra `cols[order]`. A API legada não passa por lá.
- **Origem do dado:** `runtime/servidor/servidor.go` (handler GET de `/api/<modelo>`) lê
  `q.Get("ordem")`/`q.Get("order")` sem verificação e coloca em `banco.ListarParams{Ordem: ...}`.
- **Cenário concreto:** qualquer app legado expõe `GET /api/<modelo>`. Pedir
  `GET /api/usuario?ordem=<expressão>` insere a expressão logo após `ORDER BY "id"`, o que
  permite exfiltração cega linha a linha via subconsulta em `ORDER BY (CASE WHEN (subselect) ...)`
  — inclusive ler o hash bcrypt de senha ou qualquer coluna de qualquer tabela. Não precisa de
  autenticação (GET anônimo é liberado, ver V3).
- Nota adjacente: `q()` (`banco.go:164`) só envolve o identificador em aspas duplas, sem
  escapar aspas internas; identificadores vêm do modelo, mas a regra é frágil.

### V2 — SSRF no proxy `/api/_proxy` (CWE-918) — P0

- **Onde:** `runtime/servidor/servidor.go:1120-1151`. O filtro anti-SSRF é por **prefixo de
  string** sobre `parsedURL.Hostname()` (lista `"127."`, `"10."`, `"169.254."`, `localhost`…),
  e a chamada usa `httpclient.Novo()` (`servidor.go:1150-1151`) — o cliente **sem proteção**,
  que segue redirects por padrão. NÃO usa `safeHTTPClient()` (`runtime/servidor/tarefas.go:193`),
  que valida o IP resolvido no momento do dial (a defesa do G59).
- **Bypasses:** redirect HTTP de uma URL pública para `http://169.254.169.254/…` (metadata de
  nuvem) — o cliente segue sem revalidar; `http://[::1]/` e `http://[fd00::…]/` (IPv6 não está
  na lista); IP decimal/octal/hex (`http://2130706433/` = 127.0.0.1); DNS rebinding (o host
  resolve para IP interno — o prefixo confere o hostname, não o IP).
- **Cenário concreto:** em app **sem** bloco `autenticacao`, `/api/_proxy` fica totalmente
  aberto (o `Auth.Middleware` nem existe). Um atacante faz `POST /api/_proxy` com
  `{"url":"http://servidor-publico-que-redireciona-para-metadata/"}` e lê credenciais da
  instância. O G59 corrigiu o `chamar` do interpretador; o endpoint de proxy ficou de fora.

### V3 — Dialeto legado: mass assignment + IDOR + leitura anônima (CWE-915, CWE-639) — P0

- **Mass assignment de papel:** `runtime/auth/auth.go:52-107` (`Registrar`) copia o corpo JSON
  inteiro para o INSERT (`for k, v := range input` em `:96-100`). O campo `role` é controlado
  pelo cliente → auto-registro como `admin`. Reproduzido: `POST /api/registro` com
  `{"nome":"m","email":"…","senha":"…","role":"admin"}` devolve token com `"role":"admin"`.
  É exatamente a classe Rails/GitHub 2012.
- **IDOR / account takeover:** `handleAPIComID` (`runtime/servidor/servidor.go:722+`, PUT e
  DELETE) não tem checagem de dono. Reproduzido: um usuário logado faz
  `PUT /api/usuario/1` e sobrescreve o registro de outra pessoa (nome/email); e
  `DELETE /api/produto/1` remove qualquer linha (204). Nenhuma verificação de propriedade.
- **Leitura anônima:** `Auth.Middleware` (`runtime/auth/auth.go:233-238`) libera qualquer
  `GET /api/` sem token como `anonymous`; o handler GET não faz autorização por registro.
  Reproduzido: `GET /api/usuario` e `GET /api/usuario/export/csv` despejam todas as linhas
  (emails, papéis) sem login. A senha é mascarada em `banco.go:851`, o resto vaza.
- `CheckRole` (`auth.go:259-271`) só protege métodos não-GET quando existe uma tela `telas` com
  `requires`; não há modelo de dono/linha no dialeto legado.

### V4 — Upload permite SVG servido inline (XSS armazenado) — P2

- `runtime/servidor/servidor.go:794-802`: a lista branca de upload inclui `.svg`; o arquivo é
  servido de `/uploads/` com seu próprio content-type (`fileServer`), e um SVG pode conter
  `<script>`. `POST /upload` não exige token em app sem `autenticacao` e não tem
  `MaxBytesReader` (só `ParseMultipartForm(128<<20)`), fora do rate limit (que só cobre POST em
  `/api/`). XSS armazenado no mesmo domínio → sequestro de sessão.

---

## Como o Germanio trata cada classe hoje (tabela)

Legenda: **Padrão** = defesa que já existe; **Lacuna** = o que falta/erra; **Capability geral**
= mecanismo proposto no core (sem virar centenas de palavras-chave). "intenção" = dialeto novo
(`intencao.go`); "legado" = dialeto `autenticacao`/`telas`.

| Classe de erro | Defesa padrão no Germanio (arquivo:linha) | Lacuna | Capability geral proposta |
|---|---|---|---|
| **Null / ausência** | tipos do domínio têm zero-value definido; `obrigatorio` validado em `banco.Validar` (`banco.go:706+`); resolver injeta defaults (`resolver.go:408`) | erro de execução vira nulo silencioso já foi corrigido no interpretador (G03), mas a API legada devolve `null` de coluna sem distinção | manter G03 (RuntimeError com arquivo:linha) e nunca "variável desconhecida = o próprio nome" |
| **Input não validado** | `banco.Validar` aplica regras `validar` (`banco.go`), tipos (`email`, `dinheiro`); JSON limitado a `maxRouteBody` no dialeto de intenção (`intencao.go:271,283`) | corpo sem limite em `/upload`, `/api/_proxy`, `/api/_presence`, `auth.Login` (`auth.go:126`), `httpclient` (V4 e AUDITORIA 3.5) | limite de corpo padrão por rota derivado do tipo do campo; `MaxBytesReader` em todo ponto de entrada |
| **SQL injection** | dialeto de intenção e a maior parte do banco usam placeholders (`b.ph(n)`); `Filtrar` valida `order` contra colunas (`consulta.go:206`) | **V1**: `ORDER BY` cru em `Listar` (`banco.go:404`); `q()` não escapa (`banco.go:164`) | toda coluna/ordem vem de um catálogo derivado de `ast.App`; identificador nunca de string do cliente; um único construtor de consulta com lista branca |
| **XSS** | escape em templates `html/template` (`paginas.go`); markdown sanitizado (`formatado`, goldmark, G18) | **V4**: SVG inline; CSP com `'unsafe-inline' 'unsafe-eval'` (`servidor.go:~245`) enfraquece defesa | content-type seguro por padrão para uploads (SVG servido como `text/plain`/download); CSP sem `unsafe-*` como meta declarativa |
| **CSRF** | dialeto de intenção exige token CSRF em escritas por sessão (`intencao.go:438-449`); cookie `HttpOnly`/`SameSite=Lax`/`Secure` sob TLS (`identidade.go:209`) | legado usa JWT Bearer (CSRF n/a lá), mas `/upload` e `/api/_proxy` não têm CSRF nem origem | mesma verificação de origem/token que o dialeto de intenção, aplicada a todo endpoint mutante |
| **Auth (autenticação)** | bcrypt (`auth.go:81`, `stdlib.go` `cripto`); JWT HS256 com segredo de env (`stdlib.go:46` exige ≥32 bytes, senão chave aleatória por processo com aviso); bloqueio de login por padrão (G83, 10/10min) | legado gera segredo aleatório se `JWT_SECRET` não definido (`engine.go:356`) — sessões não sobrevivem a restart e cada worker teria chave diferente; `validarToken` legado (`auth.go:294`) não exige `alg` no header (mitigado: recomputa HMAC e ignora header) | segredo único obrigatório em produção; um só verificador de token (o de intenção, `stdlib.go:96`, já checa `alg`) |
| **Autorização / IDOR (BOLA)** | dialeto de intenção: `Can`/`Owns`/papel mínimo por registro (`interpreter/intencao.go:263-300`), autoria só na criação (G45), busca não concede ver (G47) | **V3**: dialeto legado não tem autorização por linha; qualquer logado edita/apaga qualquer id | aposentar o CRUD legado aberto; o predicado de visibilidade vira SQL (G85) e vale para todo caminho |
| **Mass assignment** | dialeto de intenção: `writable()` filtra campos `System`/`Segredo` e impede setar `admin`/`papel` sem ser admin (`intencao.go:378-419`); dono só na criação | **V3**: `Registrar` legado (`auth.go:96`) copia o corpo inteiro, `role` incluído | lista branca de campos graváveis derivada do `ast.App` em **todo** create/update, inclusive no cadastro |
| **Segredos** | `GERMANIO_SEGREDO`/`JWT_SECRET` de env; senha do admin inicial nunca no `.ge` (`INTENCAO.md` Login); `.env` nunca servido (G91) | chaves de integração lidas de env fixos e opacos (`interpreter.go:1303+`: `OPENAI_KEY`, `STRIPE_KEY`…) — nomes de sistemas externos no core | cofre de credenciais como capability de adaptador; core não conhece `STRIPE_KEY` |
| **Senhas** | bcrypt `DefaultCost` (`auth.go:81`); mascaramento de `senha`/`password` na API automática (`banco.go:851`); tipo `senha`/`Segredo` nunca serializado (`intencao.go:299`) | custo fixo; `GERMANIO_BCRYPT_RAPIDO` existe para testes (`stdlib.go:144`) — risco se vazar para produção | política de senha/hash como padrão do runtime; nunca ler colunas-senha na resposta |
| **Serialização (deserialização insegura)** | JSON via `encoding/json` em `map[string]any` (sem gob/pickle); sem execução na desserialização | `map[string]any` do banco à resposta é largo (AUDITORIA 1.3); campos extras do corpo são ignorados só quando há lista branca | serialização sempre projetada pelo schema (nunca `SELECT *` → resposta) |
| **Uploads** | lista branca de extensões (`servidor.go:794`); nome gerado por timestamp (sem traversal na escrita); `resolveUploadPath` confina em `uploads/` (`servidor.go:867+`) | **V4**: SVG; sem limite de tamanho; `/upload` sem auth em app sem login | capability `arquivo` com tipo/limite declarados; validar por conteúdo (magic bytes), servir como download |
| **Path / traversal** | `resolveUploadPath` usa `filepath.Clean`+prefixo (`servidor.go:867`); pastas servidas recusam diretório e arquivos ocultos (G91); Git confina em `GERMANIO_GIT_RAIZ` | `git.ReadFile` lê blob inteiro antes de truncar (AUDITORIA 3.5) — DoS, não traversal | caminho sempre resolvido contra uma raiz declarada; nenhum caminho concatenado à mão |
| **Execução de processos** | `processo.executar` sem shell, timeout, dir confinado, `docker --network none` opcional (G16); `/api/_eval` só admin (G26) | executor local roda como o usuário do servidor (G53, isolamento fraco); recomenda runners externos | executor como capability com isolamento declarado (namespaces/container) por padrão |
| **SSRF** | `safeHTTPClient` valida IP resolvido no dial, bloqueia loopback/privado/link-local salvo `GERMANIO_PERMITIR_REDE_LOCAL` (`tarefas.go:193`, G59); webhooks pela fila | **V2**: `/api/_proxy` usa `httpclient.Novo()` sem guarda e segue redirects | **um** cliente HTTP de saída no core (o seguro), usado por todo caminho; sem redirect para destino não revalidado |
| **Dependências (supply chain)** | Go modules com `go.sum`/`sum.golang.org`; sem gerenciador de pacotes próprio (nada executado na instalação) | binário embute WhatsApp, drivers, Git sempre (G90) — superfície maior que a declarada | build só com as capabilities declaradas (ver PACKAGE_MANAGEMENT.md) |

---

## Padrões que aparecem em linguagens/frameworks e o que o Germanio já faz certo

- **Whitelist de campos por padrão** (Rails strong_parameters depois de 2012): o dialeto de
  intenção já faz (`writable`, `intencao.go:378`). O erro é o legado ainda existir.
- **Autorização por objeto, não só por rota** (OWASP API #1 BOLA): o dialeto de intenção acerta
  com `Can`/`Owns`; o legado só tem papel por tela.
- **Validar o IP resolvido, não a string** (defesa canônica de SSRF): `safeHTTPClient` acerta;
  o proxy não usa.
- **Segredo forte obrigatório**: exigido no dialeto de intenção; o fallback aleatório do legado
  é uma armadilha de "funciona em dev, quebra sessões em prod".

## Para o Germanio

- **EVITAR / remover** o CRUD legado aberto e o `Registrar` de `runtime/auth`: são a origem de
  V1–V3. Remove da cabeça do programador a necessidade de saber que "o dialeto antigo não
  protege nada". Arquivos: `runtime/auth/auth.go`, o ramo `/api/<modelo>` de
  `runtime/servidor/servidor.go`. (Decisão de produto — D1 já aponta para o dialeto de intenção.)
- **ADOTAR** um único cliente HTTP de saída seguro no core, usado por `chamar`, cron, proxy e
  webhooks. Corrige V2 e centraliza a defesa de SSRF. Arquivo: `runtime/httpclient`,
  `runtime/servidor/servidor.go` (proxy), `runtime/cron`.
- **ADOTAR** um único construtor de consulta com lista branca de coluna/ordem derivada de
  `ast.App`; nenhum identificador vem de string do cliente. Corrige V1 e alinha com G85.
  Arquivo: `runtime/banco/banco.go`, `runtime/banco/consulta.go`.
- **ADAPTAR** a política de upload: validar por conteúdo, limite declarado no campo `arquivo`,
  servir tipos de risco (SVG, HTML) como download. Corrige V4. Arquivo: `runtime/servidor`.
- **INVESTIGAR** cofre de credenciais como capability de adaptador, tirando `STRIPE_KEY`,
  `OPENAI_KEY` etc. do core (`interpreter.go:1303+`) — hoje o core conhece nomes de sistemas
  externos, contra a regra das três camadas do `CLAUDE.md`.
