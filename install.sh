#!/usr/bin/env bash
# Script oficial de instalação do Germanio (Linux & macOS)
set -euo pipefail

CYAN='\033[0;36m'
GREEN='\033[0;32m'
RED='\033[0;31m'
RESET='\033[0m'
BOLD='\033[1m'

echo -e "${CYAN}${BOLD}"
echo "  ███████╗███████╗██████╗ ███╗   ███╗ █████╗ ███╗   ██╗██╗ ██████╗ "
echo "  ██╔════╝██╔════╝██╔══██╗████╗ ████║██╔══██╗████╗  ██║██║██╔═══██╗"
echo "  ██║  ███╗█████╗  ██████╔╝██╔████╔██║███████║██╔██╗ ██║██║██║   ██║"
echo "  ██║   ██║██╔══╝  ██╔══██╗██║╚██╔╝██║██╔══██║██║╚██╗██║██║██║   ██║"
echo "  ╚██████╔╝███████╗██║  ██║██║ ╚═╝ ██║██║  ██║██║ ╚████║██║╚██████╔╝"
echo "   ╚═════╝ ╚══════╝╚═╝  ╚═╝╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═╝ ╚═════╝ "
echo -e "${RESET}"
echo -e "${BOLD}Instalador do Germanio — A Linguagem Declarativa Moderna${RESET}\n"

INSTALL_DIR="${HOME}/.germanio"
BIN_DIR="${HOME}/.local/bin"

mkdir -p "${INSTALL_DIR}"
mkdir -p "${BIN_DIR}"

echo -e "➔ Clonando/Atualizando repositório oficial..."
if [ -d "${INSTALL_DIR}/.git" ]; then
    git -C "${INSTALL_DIR}" pull --quiet origin master
else
    git clone --quiet https://github.com/flaviokalleu/germanio.git "${INSTALL_DIR}"
fi

echo -e "➔ Compilando o compilador Germanio..."
cd "${INSTALL_DIR}"
go build -o "${BIN_DIR}/ge" ./cmd/ge
go build -o "${BIN_DIR}/germanio" .

echo -e "${GREEN}${BOLD}✓ Germanio instalado com sucesso em ${BIN_DIR}/ge e ${BIN_DIR}/germanio!${RESET}\n"

# Adicionar ao PATH se não existir
SHELL_RC=""
if [ -n "${BASH_VERSION:-}" ]; then
    SHELL_RC="${HOME}/.bashrc"
elif [ -n "${ZSH_VERSION:-}" ]; then
    SHELL_RC="${HOME}/.zshrc"
fi

if [ -n "${SHELL_RC}" ] && [ -f "${SHELL_RC}" ]; then
    if ! grep -q ".local/bin" "${SHELL_RC}"; then
        echo 'export PATH="$HOME/.local/bin:$PATH"' >> "${SHELL_RC}"
        echo -e "➔ PATH atualizado em ${SHELL_RC}"
    fi
fi

echo -e "Para começar a usar agora mesmo:"
echo -e "  ${CYAN}ge novo meu_app${RESET}"
echo -e "  ${CYAN}germanio run examples/prompt-saas.ge${RESET}\n"
