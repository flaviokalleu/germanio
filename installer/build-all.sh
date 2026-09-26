#!/usr/bin/env bash
# =============================================================================
# Germanio Cross-Platform Build Script
# Script de Build Multi-Plataforma do Germanio
#
# Uso / Usage:
#   bash installer/build-all.sh
#   bash installer/build-all.sh --version 0.4.1
#   bash installer/build-all.sh --platform linux/amd64
#   bash installer/build-all.sh --skip-archive
#
# Plataformas suportadas / Supported platforms:
#   linux/amd64, linux/arm64
#   windows/amd64, windows/arm64
#   darwin/amd64, darwin/arm64 (macOS)
#
# Requer / Requires: Go 1.21+, zip, tar
# =============================================================================

set -euo pipefail

# ---------------------------------------------------------------------------
# Configuracao / Configuration
# ---------------------------------------------------------------------------
GERMANIO_VERSION="${GERMANIO_VERSION:-0.5.0}"
BINARY_NAME="germanio"

# Diretorio raiz do projeto / Project root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DIST_DIR="${PROJECT_ROOT}/dist"

# ---------------------------------------------------------------------------
# Cores / Colors
# ---------------------------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

info()    { printf "${CYAN}[build]${RESET} %s\n" "$*"; }
success() { printf "${GREEN}[ok]${RESET}    ${BOLD}%s${RESET}\n" "$*"; }
warn()    { printf "${YELLOW}[warn]${RESET}  %s\n" "$*" >&2; }
error()   { printf "${RED}[erro]${RESET}  %s\n" "$*" >&2; exit 1; }
step()    { printf "\n${BOLD}${CYAN}━━━ %s ━━━${RESET}\n" "$*"; }

# ---------------------------------------------------------------------------
# Plataformas alvo / Target platforms
# Formato: OS/ARCH
# ---------------------------------------------------------------------------
ALL_PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
    "windows/arm64"
    "darwin/amd64"
    "darwin/arm64"
)

# ---------------------------------------------------------------------------
# Flags de build Go / Go build flags
# -s -w: remove debug info (reduz tamanho) / removes debug info (reduces size)
# ---------------------------------------------------------------------------
BASE_LDFLAGS="-s -w -X main.Version=${GERMANIO_VERSION} -X main.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# ---------------------------------------------------------------------------
# Parse de argumentos / Argument parsing
# ---------------------------------------------------------------------------
SKIP_ARCHIVE=false
SPECIFIC_PLATFORM=""
DRY_RUN=false

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --version|-v)
                GERMANIO_VERSION="$2"
                BASE_LDFLAGS="-s -w -X main.Version=${GERMANIO_VERSION} -X main.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
                shift 2
                ;;
            --platform|-p)
                SPECIFIC_PLATFORM="$2"
                shift 2
                ;;
            --skip-archive)
                SKIP_ARCHIVE=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --help|-h)
                printf "Uso: %s [opcoes]\n\n" "$0"
                printf "Opcoes / Options:\n"
                printf "  --version VER      Versao a compilar (padrao: %s)\n" "$GERMANIO_VERSION"
                printf "  --platform OS/ARCH Compila apenas uma plataforma\n"
                printf "  --skip-archive     Nao cria arquivos de distribuicao\n"
                printf "  --dry-run          Mostra o que seria feito\n"
                printf "  --help             Exibe esta ajuda\n\n"
                printf "Plataformas disponiveis / Available platforms:\n"
                printf "  %s\n" "${ALL_PLATFORMS[@]}"
                exit 0
                ;;
            *)
                warn "Argumento desconhecido: $1"
                shift
                ;;
        esac
    done
}

# ---------------------------------------------------------------------------
# Verifica pre-requisitos / Check prerequisites
# ---------------------------------------------------------------------------
check_prerequisites() {
    step "Verificando pre-requisitos / Checking prerequisites"

    # Go e obrigatorio / Go is required
    command -v go &>/dev/null || error "Go nao encontrado. Instale em: https://go.dev/dl/"

    local go_version
    go_version=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+' | head -1)
    info "Go version: go${go_version}"

    # Ferramentas de compressao / Compression tools
    command -v tar  &>/dev/null || error "tar nao encontrado."
    command -v zip  &>/dev/null || warn  "zip nao encontrado. Archives .zip nao serao criados."
    command -v sha256sum &>/dev/null || command -v shasum &>/dev/null || warn "sha256sum nao encontrado. Checksums nao serao gerados."

    # Verifica que estamos na raiz do projeto / Verify we're at project root
    [[ -f "${PROJECT_ROOT}/go.mod" ]] || error "go.mod nao encontrado em ${PROJECT_ROOT}. Execute este script a partir da raiz do projeto."

    success "Pre-requisitos OK"
}

# ---------------------------------------------------------------------------
# Prepara o diretorio de distribuicao / Prepare distribution directory
# ---------------------------------------------------------------------------
prepare_dist() {
    step "Preparando diretorio dist / Preparing dist directory"

    if [[ -d "$DIST_DIR" ]]; then
        info "Limpando dist/ existente / Cleaning existing dist/..."
        rm -rf "${DIST_DIR:?}/"*
    fi

    mkdir -p "$DIST_DIR"
    success "Diretorio dist/ preparado: ${DIST_DIR}"
}

# ---------------------------------------------------------------------------
# Compila para uma plataforma / Build for a platform
# ---------------------------------------------------------------------------
build_platform() {
    local os="$1"
    local arch="$2"

    local binary_name="${BINARY_NAME}"
    local archive_name="germanio-v${GERMANIO_VERSION}-${os}-${arch}"
    local output_dir="${DIST_DIR}/${archive_name}"

    # Windows usa .exe / Windows uses .exe
    [[ "$os" == "windows" ]] && binary_name="${BINARY_NAME}.exe"

    info "Compilando / Building: ${os}/${arch} -> ${archive_name}..."

    if [[ "$DRY_RUN" == "true" ]]; then
        info "[DRY-RUN] GOOS=${os} GOARCH=${arch} go build -o ${output_dir}/${binary_name}"
        return 0
    fi

    mkdir -p "$output_dir"

    # CGO_ENABLED=0 garante binario estatico / ensures static binary
    # Exceto para SQLite que precisa de CGO / Except SQLite which needs CGO
    local cgo_enabled=0

    # Para compilacao cruzada com CGO, precisariamos de cross-compilers
    # For cross-compilation with CGO, we'd need cross-compilers
    # Usa CGO apenas para a plataforma atual / Use CGO only for current platform
    local current_os current_arch
    current_os="$(go env GOOS)"
    current_arch="$(go env GOARCH)"

    if [[ "$os" == "$current_os" ]] && [[ "$arch" == "$current_arch" ]]; then
        cgo_enabled=1
        info "  Compilacao nativa com CGO habilitado / Native build with CGO enabled"
    else
        cgo_enabled=0
        warn "  Compilacao cruzada sem CGO (SQLite usara driver puro Go)"
        warn "  Cross-compilation without CGO (SQLite will use pure Go driver)"
    fi

    # Build
    if ! (
        cd "$PROJECT_ROOT" && \
        CGO_ENABLED=$cgo_enabled \
        GOOS="$os" \
        GOARCH="$arch" \
        go build \
            -trimpath \
            -ldflags "${BASE_LDFLAGS}" \
            -o "${output_dir}/${binary_name}" \
            .
    ); then
        warn "Build falhou para ${os}/${arch}. Pulando / Build failed for ${os}/${arch}. Skipping."
        rm -rf "$output_dir"
        return 1
    fi

    # Verifica tamanho do binario / Check binary size
    local size
    size=$(du -sh "${output_dir}/${binary_name}" | cut -f1)
    success "  Compilado: ${binary_name} (${size})"

    echo "$output_dir"
}

# ---------------------------------------------------------------------------
# Copia arquivos extras para o pacote / Copy extra files to package
# ---------------------------------------------------------------------------
populate_package() {
    local pkg_dir="$1"
    local os="$2"

    # README
    [[ -f "${PROJECT_ROOT}/README.md" ]] && cp "${PROJECT_ROOT}/README.md" "${pkg_dir}/"

    # LICENSE
    [[ -f "${PROJECT_ROOT}/LICENSE" ]] && cp "${PROJECT_ROOT}/LICENSE" "${pkg_dir}/"

    # Exemplos / Examples
    if [[ -d "${PROJECT_ROOT}/exemplos" ]]; then
        mkdir -p "${pkg_dir}/examples"
        cp -r "${PROJECT_ROOT}/exemplos/." "${pkg_dir}/examples/"
        info "  Exemplos copiados / Examples copied"
    fi

    # Documentacao / Documentation
    if [[ -d "${PROJECT_ROOT}/docs" ]]; then
        mkdir -p "${pkg_dir}/docs"
        cp -r "${PROJECT_ROOT}/docs/." "${pkg_dir}/docs/"
        info "  Docs copiados / Docs copied"
    fi

    # Scripts de instalacao / Installation scripts (apenas Linux/macOS)
    if [[ "$os" != "windows" ]]; then
        cp "${SCRIPT_DIR}/linux/install.sh" "${pkg_dir}/" 2>/dev/null || true
        cp "${SCRIPT_DIR}/linux/uninstall.sh" "${pkg_dir}/" 2>/dev/null || true
        chmod +x "${pkg_dir}/install.sh" "${pkg_dir}/uninstall.sh" 2>/dev/null || true
    fi

    # Cria QUICKSTART.txt / Create QUICKSTART.txt
    cat > "${pkg_dir}/QUICKSTART.txt" << EOF
Germanio v${GERMANIO_VERSION} - Quick Start
======================================

Linux/macOS:
  chmod +x germanio
  ./germanio version
  ./germanio run examples/loja/loja.ge

  Ou instale globalmente / Or install globally:
  bash install.sh

Windows:
  germanio.exe version
  germanio.exe run examples\loja\loja.ge

Documentacao / Documentation:
  https://github.com/flaviokalleu/germanio

EOF
}

# ---------------------------------------------------------------------------
# Cria arquivo comprimido / Create compressed archive
# ---------------------------------------------------------------------------
create_archive() {
    local pkg_dir="$1"
    local os="$2"
    local archive_name
    archive_name="$(basename "$pkg_dir")"

    cd "$DIST_DIR"

    if [[ "$os" == "windows" ]]; then
        # .zip para Windows / .zip for Windows
        if command -v zip &>/dev/null; then
            zip -r "${archive_name}.zip" "${archive_name}/" -q
            success "  Criado: ${archive_name}.zip"
        else
            warn "  zip nao disponivel. Criando .tar.gz como fallback."
            tar -czf "${archive_name}.tar.gz" "${archive_name}/"
            success "  Criado: ${archive_name}.tar.gz (fallback)"
        fi
    else
        # .tar.gz para Linux/macOS / .tar.gz for Linux/macOS
        tar -czf "${archive_name}.tar.gz" "${archive_name}/"
        success "  Criado: ${archive_name}.tar.gz"
    fi

    cd - > /dev/null
}

# ---------------------------------------------------------------------------
# Gera checksums SHA256 / Generate SHA256 checksums
# ---------------------------------------------------------------------------
generate_checksums() {
    step "Gerando checksums / Generating checksums"

    cd "$DIST_DIR"

    local checksum_file="checksums.sha256"
    > "$checksum_file"

    # Coleta todos os arquivos de distribuicao / Collect all distribution files
    local files=()
    while IFS= read -r -d '' f; do
        files+=("$(basename "$f")")
    done < <(find . -maxdepth 1 \( -name "*.tar.gz" -o -name "*.zip" \) -print0 | sort -z)

    if [[ ${#files[@]} -eq 0 ]]; then
        warn "Nenhum arquivo de distribuicao encontrado."
        cd - > /dev/null
        return
    fi

    for f in "${files[@]}"; do
        if command -v sha256sum &>/dev/null; then
            sha256sum "$f" >> "$checksum_file"
        elif command -v shasum &>/dev/null; then
            shasum -a 256 "$f" >> "$checksum_file"
        fi
    done

    [[ -s "$checksum_file" ]] && success "Checksums salvos em: ${DIST_DIR}/checksums.sha256"
    cd - > /dev/null
}

# ---------------------------------------------------------------------------
# Exibe resumo do build / Display build summary
# ---------------------------------------------------------------------------
print_summary() {
    step "Resumo / Summary"

    printf "\n${BOLD}Arquivos gerados / Generated files:${RESET}\n"
    if command -v du &>/dev/null; then
        find "$DIST_DIR" -maxdepth 1 \( -name "*.tar.gz" -o -name "*.zip" -o -name "checksums.*" \) \
            -exec du -sh {} \; | sort -k2 | while read -r size file; do
            printf "  ${GREEN}%-8s${RESET} %s\n" "$size" "$(basename "$file")"
        done
    else
        ls -lh "$DIST_DIR"/*.tar.gz "$DIST_DIR"/*.zip "$DIST_DIR"/checksums.* 2>/dev/null || true
    fi

    printf "\n${BOLD}Para fazer o release no GitHub / To release on GitHub:${RESET}\n"
    printf "  git tag v${GERMANIO_VERSION}\n"
    printf "  git push origin v${GERMANIO_VERSION}\n"
    printf "  gh release create v${GERMANIO_VERSION} dist/*.tar.gz dist/*.zip dist/checksums.sha256 --generate-notes\n"
    printf "\n"
    success "Build completo! Arquivos em: ${DIST_DIR}/"
}

# ---------------------------------------------------------------------------
# Funcao principal / Main function
# ---------------------------------------------------------------------------
main() {
    parse_args "$@"

    printf "\n${BOLD}${CYAN}"
    printf "╔══════════════════════════════════════════════╗\n"
    printf "║   Germanio Build System v1.0 - Cross Platform  ║\n"
    printf "╚══════════════════════════════════════════════╝\n"
    printf "${RESET}\n"
    info "Versao / Version: v${GERMANIO_VERSION}"
    info "Projeto / Project: ${PROJECT_ROOT}"

    check_prerequisites
    prepare_dist

    # Determina quais plataformas compilar / Determine which platforms to build
    local platforms=("${ALL_PLATFORMS[@]}")
    if [[ -n "$SPECIFIC_PLATFORM" ]]; then
        platforms=("$SPECIFIC_PLATFORM")
        info "Compilando apenas / Building only: ${SPECIFIC_PLATFORM}"
    fi

    # Contadores / Counters
    local success_count=0
    local fail_count=0
    local built_dirs=()

    step "Compilando para todas as plataformas / Building for all platforms"

    for platform in "${platforms[@]}"; do
        local os arch
        IFS='/' read -r os arch <<< "$platform"

        printf "\n${BOLD}>>> ${os}/${arch}${RESET}\n"

        local pkg_dir
        if pkg_dir="$(build_platform "$os" "$arch")"; then
            # Popula o pacote com arquivos extras / Populate package with extra files
            populate_package "$pkg_dir" "$os"

            # Cria arquivo de distribuicao / Create distribution archive
            if [[ "$SKIP_ARCHIVE" == "false" ]] && [[ "$DRY_RUN" == "false" ]]; then
                create_archive "$pkg_dir" "$os"
            fi

            built_dirs+=("$pkg_dir")
            ((success_count++))
        else
            ((fail_count++))
        fi
    done

    # Gera checksums / Generate checksums
    if [[ "$SKIP_ARCHIVE" == "false" ]] && [[ "$DRY_RUN" == "false" ]] && [[ $success_count -gt 0 ]]; then
        generate_checksums
    fi

    # Remove diretorios intermediarios (mantém apenas os archives)
    # Remove intermediate directories (keep only archives)
    if [[ "$SKIP_ARCHIVE" == "false" ]] && [[ "$DRY_RUN" == "false" ]]; then
        info "Limpando diretorios temporarios / Cleaning temporary directories..."
        for dir in "${built_dirs[@]}"; do
            [[ -d "$dir" ]] && rm -rf "$dir"
        done
    fi

    print_summary

    printf "${BOLD}Resultado / Result:${RESET} ${GREEN}${success_count} ok${RESET}"
    [[ $fail_count -gt 0 ]] && printf ", ${RED}${fail_count} falhas${RESET}"
    printf "\n\n"

    [[ $fail_count -gt 0 ]] && exit 1 || exit 0
}

main "$@"
