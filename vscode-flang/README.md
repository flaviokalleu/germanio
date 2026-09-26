# Flang for VS Code

Suporte completo para a linguagem Flang (`.ge`) no Visual Studio Code.

## Recursos

- **Syntax highlighting** — Todos os 130+ keywords bilíngues (português/inglês)
- **Snippets** — Atalhos para estruturas comuns (`sistema`, `dados`, `tela`, `quando`, `funcao`, etc.)
- **Auto-indentação** — Indenta automaticamente dentro de blocos
- **Comentários** — Toggle com `Ctrl+/` (usa `#`)

## Snippets disponíveis

| Prefixo | O que cria |
|---------|-----------|
| `sistema` | Declaração do sistema |
| `dados` | Bloco de dados com modelo |
| `modelo` | Novo modelo dentro de dados |
| `telas` | Bloco de telas |
| `tela` | Nova tela |
| `eventos` | Bloco de eventos |
| `quando` | Novo evento |
| `autenticacao` | Configuração de auth |
| `logica` | Bloco de lógica |
| `funcao` | Definir função |
| `se` | Condicional se/senão |
| `definir` | Definir variável |
| `repetir` | Loop repetir N vezes |
| `enquanto` | Loop enquanto |
| `para cada` | Loop para cada |
| `importar` | Importar arquivo |
| `tema` | Tema visual |
| `cron` | Tarefa agendada |
| `whatsapp` | Integração WhatsApp |
| `banco` | Configuração do banco |
| `flang-plano` | Projeto completo (modo plano) |
| `flang-inicio` | Entry point (modo organizado) |

## Instalação manual

```bash
# Copiar para as extensões do VS Code
# Windows
cp -r vscode-flang %USERPROFILE%/.vscode/extensions/flang

# Linux/macOS
cp -r vscode-flang ~/.vscode/extensions/flang
```

Ou empacote como `.vsix`:

```bash
cd vscode-flang
npm install -g @vscode/vsce
vsce package
code --install-extension flang-0.4.0.vsix
```

## Exemplo

```flang
sistema loja

tema
  cor primaria "#6366f1"

dados

  produto
    nome: texto obrigatorio
    preco: dinheiro
    status: status

telas

  tela produtos
    titulo "Produtos"
    lista produto
      mostrar nome
      mostrar preco
      mostrar status
    botao azul
      texto "Novo"

eventos

  quando clicar "Novo"
    criar produto
```
