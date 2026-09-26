#!/usr/bin/env bash
# =============================================================================
# Flang Installer - Linux / macOS
# Instalador do Flang - Linux / macOS
#
# Uso / Usage:
#   curl -fsSL https://raw.githubusercontent.com/flaviokalleu/flang/master/installer/linux/install.sh | bash
#   ou / or:
#   bash install.sh
#   bash install.sh --uninstall
#
# Inspirado em / Inspired by: rustup (https://rustup.rs) e go install
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

# ---------------------------------------------------------------------------
# Versao do Flang / Flang version
# ---------------------------------------------------------------------------
FLANG_VERSION="${FLANG_VERSION:-0.5.0}"
FLANG_REPO="https://github.com/flaviokalleu/flang"
RELEASES_URL="${FLANG_REPO}/releases/download/v${FLANG_VERSION}"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
info()    { printf "${CYAN}[flang]${RESET} %s\n" "$*"; }
success() { printf "${GREEN}[flang]${RESET} ${BOLD}%s${RESET}\n" "$*"; }
warn()    { printf "${YELLOW}[flang]${RESET} %s\n" "$*" >&2; }
error()   { printf "${RED}[erro]${RESET}  %s\n" "$*" >&2; exit 1; }

# ---------------------------------------------------------------------------
# Detecta OS e arquitetura / Detect OS and architecture
# ---------------------------------------------------------------------------
detect_platform() {
    local os arch

    # Sistema operacional / Operating system
    case "$(uname -s)" in
        Linux*)  os="linux"  ;;
        Darwin*) os="darwin" ;;
        *)       error "Sistema operacional nao suportado: $(uname -s). Suporte para Linux e macOS." ;;
    esac

    # Arquitetura / Architecture
    case "$(uname -m)" in
        x86_64|amd64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        armv7l)        arch="arm"   ;;
        *)             error "Arquitetura nao suportada: $(uname -m). Suporte para amd64 e arm64." ;;
    esac

    echo "${os}-${arch}"
}

# ---------------------------------------------------------------------------
# Verifica dependencias / Check dependencies
# ---------------------------------------------------------------------------
check_deps() {
    local missing=()
    for cmd in curl tar; do
        command -v "$cmd" &>/dev/null || missing+=("$cmd")
    done
    if [[ ${#missing[@]} -gt 0 ]]; then
        error "Dependencias ausentes: ${missing[*]}. Instale com seu gerenciador de pacotes."
    fi
}

# ---------------------------------------------------------------------------
# Determina diretorio de instalacao / Determine install directory
# Segue XDG Base Directory Specification
# ---------------------------------------------------------------------------
determine_install_dirs() {
    if [[ $EUID -eq 0 ]] || sudo -n true 2>/dev/null; then
        # Com privilegios de root / With root privileges
        BIN_DIR="/usr/local/bin"
        LIB_DIR="/usr/local/lib/flang"
        MAN_DIR="/usr/local/share/man/man1"
        PRIVILEGED=true
    else
        # Sem sudo / Without sudo - instala para usuario atual / install for current user
        BIN_DIR="${HOME}/.local/bin"
        LIB_DIR="${HOME}/.local/lib/flang"
        MAN_DIR="${HOME}/.local/share/man/man1"
        PRIVILEGED=false
        warn "Sem privilegios root. Instalando em ${BIN_DIR}."
    fi
}

# ---------------------------------------------------------------------------
# Baixa o binario do Flang / Download the Flang binary
# ---------------------------------------------------------------------------
download_flang() {
    local platform="$1"
    local archive ext

    # Formato do arquivo / Archive format
    if [[ "$platform" == *"windows"* ]]; then
        ext="zip"
    else
        ext="tar.gz"
    fi

    archive="flang-v${FLANG_VERSION}-${platform}.${ext}"
    local url="${RELEASES_URL}/${archive}"
    local tmpdir
    tmpdir="$(mktemp -d)"

    info "Baixando / Downloading: ${url}"
    if ! curl -fsSL --progress-bar --retry 3 --retry-delay 2 -o "${tmpdir}/${archive}" "$url"; then
        # Fallback: tenta compilar do codigo fonte / Fallback: try to build from source
        warn "Download falhou. Tentando compilar do codigo fonte..."
        build_from_source "$tmpdir"
        echo "${tmpdir}/flang"
        return
    fi

    info "Extraindo / Extracting..."
    if [[ "$ext" == "tar.gz" ]]; then
        tar -xzf "${tmpdir}/${archive}" -C "$tmpdir"
    else
        command -v unzip &>/dev/null || error "unzip nao encontrado. Instale com: sudo apt install unzip"
        unzip -q "${tmpdir}/${archive}" -d "$tmpdir"
    fi

    # Localiza o binario extraido / Locate extracted binary
    local binary
    binary="$(find "$tmpdir" -name "flang" -not -name "*.tar.gz" -type f | head -1)"
    [[ -z "$binary" ]] && error "Binario nao encontrado no arquivo. Verifique o download."

    echo "$binary"
}

# ---------------------------------------------------------------------------
# Compila do codigo fonte (fallback) / Build from source (fallback)
# ---------------------------------------------------------------------------
build_from_source() {
    local outdir="$1"
    command -v go &>/dev/null || error "Go nao encontrado. Instale em https://go.dev/dl/ ou baixe o binario pre-compilado."

    info "Compilando Flang do codigo fonte com Go..."
    local tmpgit
    tmpgit="$(mktemp -d)"

    if command -v git &>/dev/null; then
        git clone --depth=1 "${FLANG_REPO}.git" "$tmpgit" 2>&1 | tail -1
    else
        error "git nao encontrado. Instale git ou baixe o codigo-fonte manualmente."
    fi

    (cd "$tmpgit" && go build -ldflags="-s -w -X main.Version=${FLANG_VERSION}" -o "${outdir}/flang" .)
    rm -rf "$tmpgit"
    success "Compilacao concluida!"
}

# ---------------------------------------------------------------------------
# Cria estrutura de diretorios / Create directory structure
# ---------------------------------------------------------------------------
create_dirs() {
    local run_as_root="$1"

    if [[ "$run_as_root" == "true" ]]; then
        sudo mkdir -p "${BIN_DIR}" "${LIB_DIR}/examples" "${LIB_DIR}/docs" "${MAN_DIR}"
    else
        mkdir -p "${BIN_DIR}" "${LIB_DIR}/examples" "${LIB_DIR}/docs" "${MAN_DIR}"
    fi
}

# ---------------------------------------------------------------------------
# Instala o binario / Install the binary
# ---------------------------------------------------------------------------
install_binary() {
    local binary="$1"
    local run_as_root="$2"

    chmod +x "$binary"

    if [[ "$run_as_root" == "true" ]]; then
        sudo cp "$binary" "${BIN_DIR}/flang"
        sudo chmod 755 "${BIN_DIR}/flang"
    else
        cp "$binary" "${BIN_DIR}/flang"
        chmod 755 "${BIN_DIR}/flang"
    fi

    success "Binario instalado em: ${BIN_DIR}/flang"
}

# ---------------------------------------------------------------------------
# Instala exemplos / Install examples
# ---------------------------------------------------------------------------
install_examples() {
    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local source_examples="${script_dir}/../../exemplos"

    if [[ -d "$source_examples" ]]; then
        info "Copiando exemplos / Copying examples..."
        if [[ "$PRIVILEGED" == "true" ]]; then
            sudo cp -r "${source_examples}/." "${LIB_DIR}/examples/"
        else
            cp -r "${source_examples}/." "${LIB_DIR}/examples/"
        fi
        success "Exemplos instalados em: ${LIB_DIR}/examples/"
    else
        # Cria exemplos basicos / Create basic examples
        local ex_content='# Exemplo Flang / Flang Example
sistema loja

tabela produto
  nome texto
  preco decimal
  estoque inteiro

rota GET /produtos -> lista produto
rota POST /produto -> cria produto

serve 8080
'
        if [[ "$PRIVILEGED" == "true" ]]; then
            echo "$ex_content" | sudo tee "${LIB_DIR}/examples/loja.ge" > /dev/null
        else
            echo "$ex_content" > "${LIB_DIR}/examples/loja.ge"
        fi
        success "Exemplo basico criado em: ${LIB_DIR}/examples/loja.ge"
    fi
}

# ---------------------------------------------------------------------------
# Cria pagina de manual / Create man page
# ---------------------------------------------------------------------------
install_man_page() {
    local man_content
    man_content='.TH FLANG 1 "'$(date +%Y-%m-%d)'" "Flang v'"${FLANG_VERSION}"'" "Flang Manual"
.SH NOME
flang \- linguagem de programacao declarativa para aplicacoes full-stack
.SH SINOPSE
.B flang
[\fICOMANDO\fR] [\fIARQUIVO.ge\fR] [\fIOPCOES\fR]
.SH DESCRICAO
Flang e uma linguagem declarativa e bilingue (Portugues/English) que gera
aplicacoes completas (backend, frontend, banco de dados, API REST) a partir
de arquivos .ge.
.SH COMANDOS
.TP
.B run \fIarquivo.ge\fR
Executa um arquivo Flang
.TP
.B version
Exibe a versao do Flang
.TP
.B help
Exibe ajuda
.SH EXEMPLOS
.TP
flang run minha-app.ge
.TP
flang version
.SH ARQUIVOS
.TP
.I ~/.local/lib/flang/examples/
Exemplos de programas Flang
.SH AUTOR
Flavio <github.com/flaviokalleu/flang>
.SH LICENCA
MIT License
'
    if [[ "$PRIVILEGED" == "true" ]]; then
        echo "$man_content" | sudo tee "${MAN_DIR}/flang.1" > /dev/null
        sudo gzip -f "${MAN_DIR}/flang.1" 2>/dev/null || true
    else
        echo "$man_content" > "${MAN_DIR}/flang.1"
        gzip -f "${MAN_DIR}/flang.1" 2>/dev/null || true
    fi
    success "Man page instalada em: ${MAN_DIR}/flang.1.gz"
}

# ---------------------------------------------------------------------------
# Configura PATH / Configure PATH
# ---------------------------------------------------------------------------
configure_path() {
    # Nao precisa configurar PATH se for instalacao global / No PATH config needed for global install
    if [[ "$PRIVILEGED" == "true" ]]; then
        return
    fi

    # Verifica se BIN_DIR ja esta no PATH / Check if BIN_DIR is already in PATH
    if [[ ":${PATH}:" == *":${BIN_DIR}:"* ]]; then
        info "${BIN_DIR} ja esta no PATH."
        return
    fi

    local path_line="export PATH=\"\$PATH:${BIN_DIR}\""

    # Adiciona ao bashrc / Add to bashrc
    if [[ -f "${HOME}/.bashrc" ]]; then
        if ! grep -qF "$BIN_DIR" "${HOME}/.bashrc"; then
            printf '\n# Flang - adicionado pelo instalador / added by installer\n%s\n' "$path_line" >> "${HOME}/.bashrc"
            success "PATH adicionado ao ~/.bashrc"
        fi
    fi

    # Adiciona ao zshrc / Add to zshrc
    if [[ -f "${HOME}/.zshrc" ]]; then
        if ! grep -qF "$BIN_DIR" "${HOME}/.zshrc"; then
            printf '\n# Flang - adicionado pelo instalador / added by installer\n%s\n' "$path_line" >> "${HOME}/.zshrc"
            success "PATH adicionado ao ~/.zshrc"
        fi
    fi

    # Adiciona ao profile para outros shells / Add to profile for other shells
    if [[ -f "${HOME}/.profile" ]]; then
        if ! grep -qF "$BIN_DIR" "${HOME}/.profile"; then
            printf '\n# Flang\n%s\n' "$path_line" >> "${HOME}/.profile"
        fi
    fi

    warn "Execute: source ~/.bashrc  (ou abra um novo terminal)"
}

# ---------------------------------------------------------------------------
# Instala arquivo .desktop no Linux / Install .desktop file on Linux
# ---------------------------------------------------------------------------
install_desktop_file() {
    [[ "$(uname -s)" != "Linux" ]] && return

    local desktop_dir
    if [[ "$PRIVILEGED" == "true" ]]; then
        desktop_dir="/usr/share/applications"
    else
        desktop_dir="${HOME}/.local/share/applications"
        mkdir -p "$desktop_dir"
    fi

    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local desktop_src="${script_dir}/flang.desktop"

    if [[ -f "$desktop_src" ]]; then
        if [[ "$PRIVILEGED" == "true" ]]; then
            sudo cp "$desktop_src" "${desktop_dir}/flang.desktop"
        else
            cp "$desktop_src" "${desktop_dir}/flang.desktop"
        fi

        # Atualiza cache de aplicativos / Update application cache
        command -v update-desktop-database &>/dev/null && update-desktop-database "$desktop_dir" 2>/dev/null || true
        success "Arquivo .desktop instalado."
    fi
}

# ---------------------------------------------------------------------------
# Desinstala o Flang / Uninstall Flang
# ---------------------------------------------------------------------------
uninstall() {
    info "Iniciando desinstalacao do Flang / Starting Flang uninstall..."

    local removed=0

    # Remove binarios / Remove binaries
    for bin in "/usr/local/bin/flang" "${HOME}/.local/bin/flang"; do
        if [[ -f "$bin" ]]; then
            if [[ "$bin" == /usr/* ]]; then
                sudo rm -f "$bin" && info "Removido: $bin" && ((removed++)) || true
            else
                rm -f "$bin" && info "Removido: $bin" && ((removed++)) || true
            fi
        fi
    done

    # Remove diretorios de biblioteca / Remove library directories
    for lib in "/usr/local/lib/flang" "${HOME}/.local/lib/flang"; do
        if [[ -d "$lib" ]]; then
            if [[ "$lib" == /usr/* ]]; then
                sudo rm -rf "$lib" && info "Removido: $lib" && ((removed++)) || true
            else
                rm -rf "$lib" && info "Removido: $lib" && ((removed++)) || true
            fi
        fi
    done

    # Remove man pages / Remove man pages
    for man in "/usr/local/share/man/man1/flang.1.gz" "${HOME}/.local/share/man/man1/flang.1.gz"; do
        if [[ -f "$man" ]]; then
            if [[ "$man" == /usr/* ]]; then
                sudo rm -f "$man" && ((removed++)) || true
            else
                rm -f "$man" && ((removed++)) || true
            fi
        fi
    done

    # Remove arquivo .desktop / Remove .desktop file
    for desktop in "/usr/share/applications/flang.desktop" "${HOME}/.local/share/applications/flang.desktop"; do
        if [[ -f "$desktop" ]]; then
            if [[ "$desktop" == /usr/* ]]; then
                sudo rm -f "$desktop" && ((removed++)) || true
            else
                rm -f "$desktop" && ((removed++)) || true
            fi
        fi
    done

    # Remove entradas do PATH dos arquivos de configuracao de shell
    # Remove PATH entries from shell config files
    for rc in "${HOME}/.bashrc" "${HOME}/.zshrc" "${HOME}/.profile"; do
        if [[ -f "$rc" ]] && grep -q "flang" "$rc"; then
            # Remove linhas relacionadas ao Flang / Remove Flang-related lines
            sed -i '/# Flang/d; /flang/d' "$rc" 2>/dev/null || true
            info "Entradas do Flang removidas de: $rc"
        fi
    done

    if [[ $removed -gt 0 ]]; then
        success "Flang desinstalado com sucesso! / Flang uninstalled successfully!"
    else
        warn "Flang nao encontrado. Nada a desinstalar."
    fi
}

# ---------------------------------------------------------------------------
# Verifica instalacao / Verify installation
# ---------------------------------------------------------------------------
verify_install() {
    local flang_bin
    flang_bin="${BIN_DIR}/flang"

    if [[ -x "$flang_bin" ]]; then
        local version_output
        version_output=$("$flang_bin" version 2>/dev/null || echo "v${FLANG_VERSION}")
        success "Verificacao concluida: $version_output"
        return 0
    else
        warn "Binario nao encontrado em ${flang_bin}. Verifique a instalacao."
        return 1
    fi
}

# ---------------------------------------------------------------------------
# Exibe banner / Show banner
# ---------------------------------------------------------------------------
print_banner() {
    printf "\n"
    printf "${BOLD}${CYAN}"
    printf "  ███████╗██╗      █████╗ ███╗   ██╗ ██████╗ \n"
    printf "  ██╔════╝██║     ██╔══██╗████╗  ██║██╔════╝ \n"
    printf "  █████╗  ██║     ███████║██╔██╗ ██║██║  ███╗\n"
    printf "  ██╔══╝  ██║     ██╔══██║██║╚██╗██║██║   ██║\n"
    printf "  ██║     ███████╗██║  ██║██║ ╚████║╚██████╔╝\n"
    printf "  ╚═╝     ╚══════╝╚═╝  ╚═╝╚═╝  ╚═══╝ ╚═════╝ \n"
    printf "${RESET}"
    printf "  ${BOLD}v${FLANG_VERSION}${RESET} - Linguagem full-stack declarativa\n"
    printf "\n"
}

# ---------------------------------------------------------------------------
# Funcao principal / Main function
# ---------------------------------------------------------------------------
main() {
    # Processa argumentos / Process arguments
    local uninstall_mode=false
    for arg in "$@"; do
        case "$arg" in
            --uninstall|-u) uninstall_mode=true ;;
            --version|-v)   echo "Flang Installer v1.0.0"; exit 0 ;;
            --help|-h)
                printf "Uso: %s [--uninstall] [--help] [--version]\n" "$0"
                printf "  --uninstall  Remove o Flang do sistema\n"
                printf "  --help       Exibe esta ajuda\n"
                printf "  --version    Exibe versao do instalador\n"
                exit 0
                ;;
        esac
    done

    print_banner

    if [[ "$uninstall_mode" == "true" ]]; then
        uninstall
        exit 0
    fi

    info "Iniciando instalacao do Flang v${FLANG_VERSION}..."

    # Verificacoes / Checks
    check_deps
    determine_install_dirs

    local platform
    platform="$(detect_platform)"
    info "Plataforma detectada / Platform detected: ${platform}"

    # Cria diretorios / Create directories
    create_dirs "$PRIVILEGED"

    # Baixa ou compila o binario / Download or build the binary
    local binary
    binary="$(download_flang "$platform")"

    # Instala / Install
    install_binary "$binary" "$PRIVILEGED"
    install_examples
    install_man_page
    install_desktop_file
    configure_path

    # Verifica / Verify
    printf "\n"
    verify_install || true

    # Mensagem final / Final message
    printf "\n"
    printf "${GREEN}${BOLD}╔══════════════════════════════════════════════╗${RESET}\n"
    printf "${GREEN}${BOLD}║  Flang instalado! Digite: flang version      ║${RESET}\n"
    printf "${GREEN}${BOLD}╚══════════════════════════════════════════════╝${RESET}\n"
    printf "\n"
    printf "  ${BOLD}Proximos passos / Next steps:${RESET}\n"
    printf "  1. Abra um novo terminal ou execute: source ~/.bashrc\n"
    printf "  2. Teste: ${CYAN}flang version${RESET}\n"
    printf "  3. Crie seu primeiro app: ${CYAN}flang run ${LIB_DIR}/examples/loja.ge${RESET}\n"
    printf "  4. Documentacao: ${CYAN}${FLANG_REPO}${RESET}\n"
    printf "\n"
}

# Ponto de entrada / Entry point
main "$@"
