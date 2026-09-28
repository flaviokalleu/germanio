#!/usr/bin/env bash
# Germanio installer (Linux and macOS): clones the repository and builds ge from source.
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
echo -e "${BOLD}Germanio installer — builds ge from source${RESET}\n"

if ! command -v go >/dev/null 2>&1; then
    echo -e "${RED}Go is required (https://go.dev/dl/). Prebuilt binaries: https://github.com/flaviokalleu/germanio/releases${RESET}"
    exit 1
fi
if ! command -v git >/dev/null 2>&1; then
    echo -e "${RED}git is required.${RESET}"
    exit 1
fi

INSTALL_DIR="${HOME}/.germanio"
BIN_DIR="${HOME}/.local/bin"

mkdir -p "${INSTALL_DIR}"
mkdir -p "${BIN_DIR}"

echo -e "➔ Cloning or updating the repository in ${INSTALL_DIR}..."
if [ -d "${INSTALL_DIR}/.git" ]; then
    git -C "${INSTALL_DIR}" pull --quiet --ff-only origin master
else
    git clone --quiet https://github.com/flaviokalleu/germanio.git "${INSTALL_DIR}"
fi

echo -e "➔ Building ge and germanio..."
cd "${INSTALL_DIR}"
go build -o "${BIN_DIR}/ge" ./cmd/ge
go build -o "${BIN_DIR}/germanio" .

echo -e "${GREEN}${BOLD}✓ Installed ${BIN_DIR}/ge and ${BIN_DIR}/germanio${RESET}\n"

# Add ~/.local/bin to PATH when missing
SHELL_RC=""
if [ -n "${BASH_VERSION:-}" ]; then
    SHELL_RC="${HOME}/.bashrc"
elif [ -n "${ZSH_VERSION:-}" ]; then
    SHELL_RC="${HOME}/.zshrc"
fi

if [ -n "${SHELL_RC}" ] && [ -f "${SHELL_RC}" ]; then
    if ! grep -q ".local/bin" "${SHELL_RC}"; then
        echo 'export PATH="$HOME/.local/bin:$PATH"' >> "${SHELL_RC}"
        echo -e "➔ Added ~/.local/bin to PATH in ${SHELL_RC}"
    fi
fi

echo -e "Get started:"
echo -e "  ${CYAN}ge new my_app${RESET}"
echo -e "  ${CYAN}ge run my_app/app.ge${RESET}   # then open http://localhost:8080\n"
