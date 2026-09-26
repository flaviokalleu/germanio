#!/usr/bin/env bash
# =============================================================================
# Germanio Uninstaller - Linux / macOS
# Desinstalador do Germanio - Linux / macOS
#
# Uso / Usage:
#   bash uninstall.sh
#   bash install.sh --uninstall
#
# Remove todos os arquivos instalados pelo install.sh
# Removes all files installed by install.sh
# =============================================================================

set -euo pipefail

# ---------------------------------------------------------------------------
# Cores / Colors
# ---------------------------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

info()    { printf "${CYAN}[germanio]${RESET} %s\n" "$*"; }
success() { printf "${GREEN}[germanio]${RESET} ${BOLD}%s${RESET}\n" "$*"; }
warn()    { printf "${YELLOW}[aviso]${RESET} %s\n" "$*" >&2; }
error()   { printf "${RED}[erro]${RESET}  %s\n" "$*" >&2; exit 1; }

# ---------------------------------------------------------------------------
# Confirma a desinstalacao / Confirm uninstall
# ---------------------------------------------------------------------------
confirm() {
    printf "\n${BOLD}Desinstalar o Germanio do sistema?${RESET}\n"
    printf "Isso removerá:\n"
    printf "  - Binário em /usr/local/bin/germanio e/ou ~/.local/bin/germanio\n"
    printf "  - Bibliotecas em /usr/local/lib/germanio e/ou ~/.local/lib/germanio\n"
    printf "  - Man page germanio(1)\n"
    printf "  - Arquivo .desktop\n"
    printf "  - Entradas de PATH nos arquivos de shell\n"
    printf "\n${YELLOW}Continuar? [s/N]${RESET} "

    read -r reply
    case "$reply" in
        [sS][iI]|[sS]|[yY][eE][sS]|[yY]) return 0 ;;
        *) info "Desinstalacao cancelada."; exit 0 ;;
    esac
}

# ---------------------------------------------------------------------------
# Remove arquivo com ou sem sudo / Remove file with or without sudo
# ---------------------------------------------------------------------------
safe_remove() {
    local path="$1"
    local is_dir="${2:-false}"

    [[ ! -e "$path" ]] && return 0

    local remove_cmd
    if [[ "$is_dir" == "true" ]]; then
        remove_cmd="rm -rf"
    else
        remove_cmd="rm -f"
    fi

    # Usa sudo para caminhos do sistema / Use sudo for system paths
    if [[ "$path" == /usr/* ]] || [[ "$path" == /etc/* ]]; then
        if sudo $remove_cmd "$path" 2>/dev/null; then
            info "Removido: $path"
            return 0
        else
            warn "Nao foi possivel remover: $path (sem permissao)"
            return 1
        fi
    else
        $remove_cmd "$path" 2>/dev/null && info "Removido: $path" || warn "Nao foi possivel remover: $path"
    fi
}

# ---------------------------------------------------------------------------
# Remove entradas do PATH dos arquivos de shell
# Remove PATH entries from shell config files
# ---------------------------------------------------------------------------
clean_shell_configs() {
    local files=("${HOME}/.bashrc" "${HOME}/.zshrc" "${HOME}/.profile" "${HOME}/.bash_profile")
    local cleaned=false

    for rc in "${files[@]}"; do
        [[ ! -f "$rc" ]] && continue

        if grep -q "germanio\|\.local/bin" "$rc" 2>/dev/null; then
            # Cria backup antes de modificar / Create backup before modifying
            cp "$rc" "${rc}.germanio-backup" 2>/dev/null || true

            # Remove linhas relacionadas ao Germanio / Remove Germanio-related lines
            local tmp
            tmp="$(mktemp)"
            grep -v "# Germanio\|germanio\|\.local/bin.*germanio\|germanio.*\.local/bin" "$rc" > "$tmp" 2>/dev/null || cp "$rc" "$tmp"
            mv "$tmp" "$rc"

            info "Configuracao de shell limpa: $rc"
            info "Backup salvo em: ${rc}.germanio-backup"
            cleaned=true
        fi
    done

    $cleaned || info "Nenhuma entrada do Germanio encontrada nos arquivos de shell."
}

# ---------------------------------------------------------------------------
# Funcao principal / Main function
# ---------------------------------------------------------------------------
main() {
    printf "\n${BOLD}${RED}"
    printf "  ╔═══════════════════════════════════╗\n"
    printf "  ║   Germanio Uninstaller / v1.0.0     ║\n"
    printf "  ╚═══════════════════════════════════╝\n"
    printf "${RESET}\n"

    # Pula confirmacao se --yes ou -y / Skip confirmation if --yes or -y
    local skip_confirm=false
    for arg in "$@"; do
        case "$arg" in
            --yes|-y) skip_confirm=true ;;
            --help|-h)
                printf "Uso: %s [--yes] [--help]\n" "$0"
                printf "  --yes   Pula confirmacao\n"
                printf "  --help  Exibe esta ajuda\n"
                exit 0
                ;;
        esac
    done

    [[ "$skip_confirm" == "false" ]] && confirm

    info "Iniciando desinstalacao / Starting uninstall..."
    local total_removed=0

    # -------------------------------------------------------------------------
    # 1. Remove binarios / Remove binaries
    # -------------------------------------------------------------------------
    info "Removendo binarios / Removing binaries..."
    safe_remove "/usr/local/bin/germanio"      && ((total_removed++)) || true
    safe_remove "${HOME}/.local/bin/germanio"  && ((total_removed++)) || true

    # -------------------------------------------------------------------------
    # 2. Remove bibliotecas e exemplos / Remove libraries and examples
    # -------------------------------------------------------------------------
    info "Removendo bibliotecas / Removing libraries..."
    safe_remove "/usr/local/lib/germanio" "true"      && ((total_removed++)) || true
    safe_remove "${HOME}/.local/lib/germanio" "true"  && ((total_removed++)) || true

    # -------------------------------------------------------------------------
    # 3. Remove man pages / Remove man pages
    # -------------------------------------------------------------------------
    info "Removendo man pages / Removing man pages..."
    safe_remove "/usr/local/share/man/man1/germanio.1"    && ((total_removed++)) || true
    safe_remove "/usr/local/share/man/man1/germanio.1.gz" && ((total_removed++)) || true
    safe_remove "${HOME}/.local/share/man/man1/germanio.1"    && ((total_removed++)) || true
    safe_remove "${HOME}/.local/share/man/man1/germanio.1.gz" && ((total_removed++)) || true

    # -------------------------------------------------------------------------
    # 4. Remove arquivo .desktop / Remove .desktop file
    # -------------------------------------------------------------------------
    info "Removendo arquivo .desktop / Removing .desktop file..."
    safe_remove "/usr/share/applications/germanio.desktop"               && ((total_removed++)) || true
    safe_remove "${HOME}/.local/share/applications/germanio.desktop"     && ((total_removed++)) || true

    # Atualiza cache de aplicativos / Update application cache
    command -v update-desktop-database &>/dev/null && {
        update-desktop-database "${HOME}/.local/share/applications" 2>/dev/null || true
        sudo update-desktop-database /usr/share/applications 2>/dev/null || true
    }

    # -------------------------------------------------------------------------
    # 5. Limpa configuracoes de shell / Clean shell configs
    # -------------------------------------------------------------------------
    info "Limpando configuracoes de shell / Cleaning shell configs..."
    clean_shell_configs

    # -------------------------------------------------------------------------
    # Resultado final / Final result
    # -------------------------------------------------------------------------
    printf "\n"
    if [[ $total_removed -gt 0 ]]; then
        success "Germanio desinstalado com sucesso! / Germanio uninstalled successfully!"
        printf "\n"
        printf "  Obrigado por usar o Germanio! / Thank you for using Germanio!\n"
        printf "  ${CYAN}${BOLD}github.com/flaviokalleu/germanio${RESET}\n\n"
    else
        warn "Germanio nao encontrado no sistema. Nada foi removido."
    fi
}

main "$@"
