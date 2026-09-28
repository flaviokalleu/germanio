# Gerenciamento de pacotes: o que os ecossistemas erraram e acertaram, e o que o Germanio deve fazer

Data: 2026-09-28
Status: pesquisa transversal. Tudo sobre o Germanio aqui é **PROPOSTO** — hoje **não há
gerenciador de pacotes** (confirmado no código, ver abaixo).
Escopo: npm, pip/PyPI+uv, Cargo, Go modules, Maven/Gradle, NuGet, Composer, Julia Pkg, Deno/JSR
— lockfiles, reprodutibilidade, resolução, cadeia de suprimentos, checksums, versionamento,
transitivas, cache, offline. Proposta para o Germanio aproveitando o Go, e o que evitar.

Fontes:
- npm supply chain 2025 — CISA (https://www.cisa.gov/news-events/alerts/2025/09/23/widespread-supply-chain-compromise-impacting-npm-ecosystem),
  Unit 42 "Shai-Hulud" (https://unit42.paloaltonetworks.com/npm-supply-chain-attack/),
  Wiz (https://www.wiz.io/blog/shai-hulud-npm-supply-chain-attack).
- Go modules — go.dev/ref/mod (https://go.dev/ref/mod), MVS
  (https://www.ardanlabs.com/blog/2019/12/modules-03-minimal-version-selection.html), checksum
  DB (https://go.dev/blog/module-mirror-launch, https://sum.golang.org/).
- uv — https://docs.astral.sh/uv/concepts/resolution/ ; Cargo/crates.io — https://doc.rust-lang.org/cargo/ ;
  cargo-vet — https://mozilla.github.io/cargo-vet/how-it-works.html .
- Deno/JSR — https://jsr.io/docs/using-packages ; postinstall
  https://docs.deno.com/runtime/reference/cli/install/ .
- left-pad (2016) — https://en.wikipedia.org/wiki/Npm_left-pad_incident .

---

## Estado atual do Germanio (IMPLEMENTADO — verificado no código)

- **Não existe gerenciador de pacotes para `.ge`.** Não há registro remoto, resolução de
  versões, lockfile nem dependências transitivas de terceiros.
- **`importar`** (`compiler/parser/parser.go:987-1022`, `compiler/lexer/lexer.go:245`) é apenas
  **inclusão de arquivos `.ge` locais**: `importar "models.ge"`, `importar dados de "…"`,
  `importar produtos e pedidos do backend`. É composição de fontes no mesmo projeto, não
  aquisição de dependência externa.
- **Capabilities** (WhatsApp, email, cron, Git, storage, IA…) são embutidas no runtime em Go,
  não pacotes. Hoje **todas** entram no binário mesmo sem uso (G90, AUDITORIA 4).
- **A única cadeia de suprimentos é a do próprio runtime**, via Go modules: o `germanio build`
  gera um projeto Go temporário com `go.mod`/`require` (`cli/cli.go:1174`) e compila com `go`.
  Herda `go.sum` e `sum.golang.org` para as dependências do *runtime* — não das apps `.ge`.

Ou seja: o Germanio começa no melhor lugar possível — **sem** os riscos de um registro que
executa código de terceiros. A pergunta é como adicionar extensibilidade (se e quando) sem
importar os erros dos outros.

---

## O que os ecossistemas erraram e acertaram

### npm — o pior caso de cadeia de suprimentos
- **left-pad (2016):** despublicar um pacote de 11 linhas quebrou milhares de builds — mostra o
  perigo de micro-dependências e de um registro mutável.
- **install scripts:** `postinstall`/`preinstall` executam código arbitrário na instalação, com
  acesso total à máquina — vetor primário de ataque.
- **typosquatting** e **2025 "Shai-Hulud"** (CISA, Unit 42, Wiz): primeiro **worm
  auto-replicante** do npm; 500+ pacotes, começando em 14/09/2025 (ngx-bootstrap,
  @ctrl/tinycolor, namespaces da CrowdStrike). Colhe segredos de CI/CD e de **endpoints de
  metadata de nuvem**, exfiltra via repositórios GitHub e webhooks, e republica pacotes com
  qualquer token npm que encontra. Combina install scripts + tokens + metadata — tudo o que um
  gerenciador seguro deve negar por padrão.
- **Lição:** o registro **nunca deve executar código na instalação**; deve ser **imutável**
  (não despublicar); tokens de publicação com escopo mínimo; e o ambiente de build sem acesso a
  metadata.

### pip/PyPI e uv
- pip clássico: resolução frágil, sem lockfile nativo, `setup.py` executa código.
- **uv** (Astral, em Rust): resolvedor rápido com **lockfile universal** (`uv.lock`) válido para
  toda plataforma/versão de Python sem re-resolver, usando *forking* por markers quando ambientes
  divergem (https://docs.astral.sh/uv/concepts/resolution/). Reprodutibilidade real; pode fixar
  por hash e limitar idade das dependências (mitigação de ataques recém-publicados).
- **Lição:** lockfile por padrão + resolução determinística; opção de "só dependências com N
  dias" reduz janela de ataque.

### Cargo / crates.io
- **crates.io é imutável** (publicou, não some — evita o left-pad); `Cargo.lock` fixa a árvore;
  semver com resolução. Nenhum script de instalação por padrão (build scripts existem e são um
  risco conhecido, mas separados).
- **cargo-vet** (Mozilla, https://mozilla.github.io/cargo-vet/how-it-works.html): verifica
  mecanicamente que um humano auditou cada dependência e registrou propriedades; audits
  compartilháveis entre organizações (`imports.lock`). Reduz incrementalmente a superfície não
  auditada.
- **Lição:** imutabilidade + lock + auditoria compartilhável e verificável.

### Go modules — o modelo que mais serve ao Germanio
- **MVS (Minimal Version Selection):** escolhe a **menor** versão que satisfaz todos os
  `require` — resolução determinística, sem "resolver mais novo por acaso". Builds estáveis por
  construção (https://www.ardanlabs.com/blog/2019/12/modules-03-minimal-version-selection.html).
- **`go.sum` + `sum.golang.org`:** hash SHA-256 de cada versão; base de checksums **auditável e
  global** que registra o hash na primeira vez que alguém busca a versão — detecta adulteração
  mesmo com `go.sum` local incompleto (https://go.dev/blog/module-mirror-launch).
- **Nenhum código roda na instalação** — diferença fundamental para o npm.
- **proxy/cache/offline:** `GOPROXY`, `GOMODCACHE`, `GOFLAGS=-mod=vendor` dão cache e builds
  offline reprodutíveis.
- **Lição:** é o alvo. Determinismo (MVS), integridade (checksum DB), zero execução na instalação.

### Maven/Gradle, NuGet, Composer, Julia, Deno/JSR
- **Maven/Gradle "nearest wins":** em conflito de versão transitiva, vence a mais próxima na
  árvore — resultado depende da forma da árvore, **não determinístico** entre projetos. A
  evitar; o MVS do Go é superior nisso.
- **NuGet/Composer:** lockfile e resolução; Composer (PHP) tem scripts como o npm (risco).
- **Julia Pkg:** `Manifest.toml` (lock) + `Project.toml`; ambientes reprodutíveis, bom modelo de
  environment isolado.
- **Deno/JSR:** imports por URL + registro JSR; **não roda postinstall scripts por padrão**
  (https://docs.deno.com/runtime/reference/cli/install/) — a resposta explícita ao problema do
  npm. Permissões explícitas para rede/FS.
- **Lição:** determinismo (Go > Maven); imutabilidade (crates/JSR); nunca executar na instalação
  (Go/Deno); ambiente isolado e reprodutível (Julia/uv).

---

## PROPOSTA para o Germanio (se e quando tiver pacotes)

O Germanio quase não precisa de pacotes de biblioteca: as capabilities são mecanismos do core, e
as apps descrevem intenção. Se a extensibilidade crescer, ela deve entrar por **duas portas**,
não por um registro geral de código executável:

1. **Blocos de intenção reutilizáveis** (`.ge` compartilhados): um "bloco de login", um "bloco de
   comentários". São **dados/declaração**, não código executável — o risco de supply chain é
   baixo por natureza (o runtime interpreta intenção, não `postinstall`). Distribuição por Git +
   pin de commit + checksum, aproveitando `sum.golang.org`-style verification.
2. **Adaptadores de integração** (`integracoes/`, nível avançado): traduzem protocolos externos.
   São o ponto perigoso (código, rede, credenciais). Devem ser **auditados** (estilo cargo-vet),
   marcados `NÍVEL AVANÇADO / ADAPTADOR`, sem regras de produto, e sem executar nada na
   instalação.

### Contrato proposto (semântica antes de sintaxe, como manda o INTENCAO.md)
- **Determinismo por MVS**, não "nearest wins": em conflito, a menor versão que satisfaz todos.
- **Lockfile por padrão** com hash (estilo `go.sum`/`uv.lock`): build reprodutível, offline com
  cache.
- **Registro imutável** (crates/JSR): publicou, não despublica; evita left-pad.
- **Zero execução na instalação** (Go/Deno): baixar um bloco/adaptador nunca roda código; ele só
  roda quando a app o declara e o runtime o inicializa (e paga só quem usa — G90).
- **Verificação por checksum central auditável** (sum.golang.org): integridade sem confiar no
  registro.
- **Permissões explícitas** por adaptador (Deno): rede/FS/credenciais declaradas; nada de acesso
  a metadata de nuvem no build (a lição direta do Shai-Hulud).
- **Auditoria compartilhável** (cargo-vet) para adaptadores.
- `ge explain` mostra a proveniência: de onde veio cada bloco/adaptador, qual versão, qual hash.

### Aproveitando o Go
- Reusar a infraestrutura do Go modules do próprio runtime: os adaptadores em Go entram no build
  do binário via `go.mod`/`go.sum` (já é o caminho do `germanio build`, `cli/cli.go:1174`), com
  MVS e checksum DB de graça. Blocos `.ge` viajam como fontes com pin de commit + hash.

### O que evitar (explicitamente)
- **Install/postinstall scripts** (npm/Composer): nunca. É o vetor #1.
- **Registro mutável / despublicação** (left-pad): não.
- **"Nearest wins"** (Maven): não determinístico; usar MVS.
- **Micro-dependências** e árvores transitivas profundas: a cultura npm de 11-linhas amplia a
  superfície. O Germanio prefere capabilities no core a milhares de pacotinhos.
- **Tokens de publicação amplos** e ambiente de build com acesso a segredos/metadata.

---

## Para o Germanio

- **ADOTAR** (quando/se houver pacotes) o modelo Go modules: MVS + `go.sum`/checksum DB + zero
  execução na instalação. Problema resolvido: extensibilidade futura sem os riscos do npm.
  Remove da cabeça do programador: resolver conflitos de versão à mão, confiar no registro.
- **ADOTAR** lockfile por padrão (uv/Cargo/Go) e registro imutável (crates/JSR). Remove:
  "funciona na minha máquina" e o medo de left-pad.
- **ADAPTAR** cargo-vet para auditoria de **adaptadores** de `integracoes/` (o único código de
  terceiros perigoso). Arquivo: `integracoes/`. Remove: confiar cegamente em adaptador externo.
- **ADOTAR** permissões explícitas por adaptador e build sem acesso a metadata (lição
  Shai-Hulud 2025). Arquivo: pipeline de `germanio build` (`cli/cli.go`).
- **EVITAR** um registro npm-like com install scripts, mutabilidade e micro-dependências.
- **INVESTIGAR** se blocos `.ge` compartilháveis chegam a ser necessários, ou se capabilities no
  core + `importar` local já cobrem o caso. Marcar como pendência de design; hoje `importar`
  (`compiler/parser/parser.go:987`) resolve a composição interna sem nenhum gerenciador.
