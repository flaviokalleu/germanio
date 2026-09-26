# Model Context Protocol (MCP) — Germanio

O **Germanio MCP Server** permite que LLMs e agentes de IA (Claude Desktop, Cursor, Hermes, Cline, Roo Code, VS Code Copilot) criem, verifiquem, compilem, executem e manipulem sistemas inteiros escritos em Germanio (`.ge`) diretamente via protocolo MCP.

---

## 🚀 Configuração Rápida

### Claude Desktop (`claude_desktop_config.json`) ou Cursor:

```json
{
  "mcpServers": {
    "germanio": {
      "command": "ge",
      "args": ["mcp"],
      "env": {
        "GERMANIO_ENV": "development"
      }
    }
  }
}
```

---

## 🛠️ Ferramentas (MCP Tools) Disponíveis

### 1. `germanio_check`
Valida a sintaxe e a semântica de um arquivo `.ge` sem executá-lo.
- **Parâmetros:**
  - `arquivo` *(string, obrigatório)*: Caminho do arquivo `.ge` a ser validado.
- **Retorno:** Status de validação, contagem de modelos/telas/eventos ou lista de erros com linha e coluna.

```json
{
  "name": "germanio_check",
  "arguments": {
    "arquivo": "app.ge"
  }
}
```

---

### 2. `germanio_run`
Executa um sistema ou script `.ge` iniciando o backend HTTP, banco de dados e UI reativa.
- **Parâmetros:**
  - `arquivo` *(string, obrigatório)*: Caminho do arquivo `.ge`.
  - `porta` *(string, opcional)*: Porta HTTP (padrão `8080`).
- **Retorno:** URL do servidor ativo e logs de inicialização.

---

### 3. `germanio_build`
Compila a aplicação `.ge` em um executável binário único e autônomo (standalone).
- **Parâmetros:**
  - `arquivo` *(string, obrigatório)*: Arquivo `.ge` de entrada.
  - `output` *(string, opcional)*: Nome do binário de saída.
- **Retorno:** Caminho do binário compilado.

---

### 4. `germanio_format`
Formata o código `.ge` seguindo a especificação canônica do Germanio.
- **Parâmetros:**
  - `caminho` *(string, obrigatório)*: Arquivo ou pasta `.ge`.
  - `check` *(boolean, opcional)*: Se verdadeiro, apenas reporta se precisa de formatação.

---

### 5. `germanio_scaffold`
Gera um projeto completo a partir de um prompt em linguagem natural.
- **Parâmetros:**
  - `nome` *(string, obrigatório)*: Nome do sistema/app.
  - `descricao` *(string, obrigatório)*: O que o sistema deve fazer.
  - `banco` *(string, opcional)*: `sqlite` ou `postgres`.
  - `tema` *(string, opcional)*: `moderno`, `escuro`, `elegante`.

---

## 📚 Recursos (MCP Resources)

O servidor MCP expõe documentações e templates para IA consultar:

- `germanio://spec` — Especificação completa da linguagem.
- `germanio://cheatsheet` — Tabela rápida de sintaxe, tipos e comandos.
- `germanio://examples/saas` — Exemplo completo de SaaS com banco, API e telas.
- `germanio://examples/whatsapp` — Exemplo de automação com WhatsApp nativo.
