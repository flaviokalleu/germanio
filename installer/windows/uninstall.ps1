#Requires -Version 5.0
# =============================================================================
# Germanio Programming Language - Desinstalador para Windows
# Versao 0.4.0
#
# Uso:
#   .\uninstall.ps1
#   .\uninstall.ps1 -DiretorioInstalacao "D:\Germanio"
#   .\uninstall.ps1 -Silencioso
# =============================================================================

param(
    # Diretório onde o Germanio foi instalado
    [string]$DiretorioInstalacao = "C:\Germanio",

    # Remove sem perguntas interativas
    [switch]$Silencioso
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$DirBin     = Join-Path $DiretorioInstalacao "bin"
$NomeClasse = "GermanioFile"

# ---------------------------------------------------------------------------
# Funcoes de saida formatada
# ---------------------------------------------------------------------------
function Escrever-Ok    { param([string]$t) Write-Host "  [OK]   $t" -ForegroundColor Green  }
function Escrever-Aviso { param([string]$t) Write-Host "  [!]    $t" -ForegroundColor Yellow }
function Escrever-Erro  { param([string]$t) Write-Host "  [ERRO] $t" -ForegroundColor Red    }
function Escrever-Info  { param([string]$t) Write-Host "  $t"        -ForegroundColor Gray   }

# ---------------------------------------------------------------------------
# Banner
# ---------------------------------------------------------------------------
Write-Host ""
Write-Host "  Germanio Programming Language v0.4.0 - Desinstalador" -ForegroundColor Cyan
Write-Host "  ==================================================" -ForegroundColor DarkCyan
Write-Host ""
Write-Host "  Diretorio de instalacao: " -NoNewline
Write-Host $DiretorioInstalacao -ForegroundColor White
Write-Host ""

# ---------------------------------------------------------------------------
# Confirmacao
# ---------------------------------------------------------------------------
if (-not $Silencioso) {
    $resposta = Read-Host "  Deseja realmente remover o Germanio do sistema? [s/N]"
    if ($resposta -notmatch "^[Ss]") {
        Write-Host ""
        Write-Host "  Desinstalacao cancelada." -ForegroundColor Gray
        Write-Host ""
        exit 0
    }
    Write-Host ""
}

# ---------------------------------------------------------------------------
# 1. Remover o diretorio de instalacao
# ---------------------------------------------------------------------------
Write-Host "  Removendo arquivos..." -ForegroundColor Yellow

if (Test-Path $DiretorioInstalacao) {
    try {
        # Tenta encerrar o processo germanio.exe caso esteja rodando
        Get-Process -Name "germanio" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
        Start-Sleep -Milliseconds 500

        Remove-Item -Path $DiretorioInstalacao -Recurse -Force
        Escrever-Ok "Diretorio removido: $DiretorioInstalacao"
    } catch {
        Escrever-Erro "Nao foi possivel remover $DiretorioInstalacao"
        Escrever-Info "Feche todos os programas que usam o Germanio e tente novamente."
        Escrever-Info "Ou remova manualmente: $DiretorioInstalacao"
    }
} else {
    Escrever-Aviso "Diretorio nao encontrado: $DiretorioInstalacao"
    Escrever-Info  "Pode ja ter sido removido."
}

# ---------------------------------------------------------------------------
# 2. Remover do PATH do usuario (HKCU - sem admin)
# ---------------------------------------------------------------------------
Write-Host "  Removendo do PATH do usuario..." -ForegroundColor Yellow

try {
    $chaveRegistro = "HKCU:\Environment"
    $pathAtual = (Get-ItemProperty -Path $chaveRegistro -Name "Path" -ErrorAction SilentlyContinue).Path

    if ($pathAtual) {
        # Filtra todas as entradas que apontem para o DirBin (com ou sem barra no final)
        $entradas  = $pathAtual -split ";" | Where-Object { $_ -ne "" }
        $novasEntradas = $entradas | Where-Object {
            ($_ -ne $DirBin) -and
            ($_ -ne ($DirBin + "\")) -and
            ($_ -ne $DiretorioInstalacao) -and
            ($_ -ne ($DiretorioInstalacao + "\"))
        }

        $novoPath = $novasEntradas -join ";"
        Set-ItemProperty -Path $chaveRegistro -Name "Path" -Value $novoPath -Type ExpandString

        # Notifica o Windows sobre a mudanca
        try {
            $assinatura = @"
[DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)]
public static extern IntPtr SendMessageTimeout(
    IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam,
    uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);
"@
            $tipo = Add-Type -MemberDefinition $assinatura -Name "Win32SM" -Namespace "Win32Fn" -PassThru -ErrorAction SilentlyContinue
            if ($tipo) {
                $resultado = [UIntPtr]::Zero
                $tipo::SendMessageTimeout([IntPtr]0xffff, 0x001A, [UIntPtr]::Zero, "Environment", 2, 5000, [ref]$resultado) | Out-Null
            }
        } catch { }

        Escrever-Ok "Removido do PATH do usuario"
    } else {
        Escrever-Aviso "PATH do usuario nao encontrado no registro"
    }
} catch {
    Escrever-Erro "Falha ao atualizar o PATH: $_"
}

# ---------------------------------------------------------------------------
# 3. Remover associacao de arquivo .ge (HKCU - sem admin)
# ---------------------------------------------------------------------------
Write-Host "  Removendo associacao de arquivo .ge..." -ForegroundColor Yellow

$chavesRemover = @(
    "HKCU:\Software\Classes\.ge",
    "HKCU:\Software\Classes\$NomeClasse"
)

foreach ($chave in $chavesRemover) {
    if (Test-Path $chave) {
        try {
            Remove-Item -Path $chave -Recurse -Force
            Escrever-Ok "Chave removida: $chave"
        } catch {
            Escrever-Aviso "Nao foi possivel remover: $chave"
        }
    } else {
        Escrever-Info "Chave ja inexistente: $chave"
    }
}

# Notifica o Explorer sobre a mudanca de associacao
try {
    $code = @"
[DllImport("shell32.dll")]
public static extern void SHChangeNotify(int wEventId, int uFlags, IntPtr dwItem1, IntPtr dwItem2);
"@
    $shell = Add-Type -MemberDefinition $code -Name "ShellNotifyUninstall" -Namespace "Win32Uninst" -PassThru -ErrorAction SilentlyContinue
    if ($shell) {
        $shell::SHChangeNotify(0x08000000, 0x0000, [IntPtr]::Zero, [IntPtr]::Zero)
    }
} catch { }

Escrever-Ok "Associacao de arquivo .ge removida"

# ---------------------------------------------------------------------------
# 4. Remover atalhos do Menu Iniciar
# ---------------------------------------------------------------------------
Write-Host "  Removendo atalhos do Menu Iniciar..." -ForegroundColor Yellow

$startMenuGermanio = Join-Path $env:APPDATA "Microsoft\Windows\Start Menu\Programs\Germanio"
if (Test-Path $startMenuGermanio) {
    try {
        Remove-Item -Path $startMenuGermanio -Recurse -Force
        Escrever-Ok "Atalhos do Menu Iniciar removidos"
    } catch {
        Escrever-Aviso "Nao foi possivel remover atalhos: $startMenuGermanio"
    }
} else {
    Escrever-Info "Sem atalhos no Menu Iniciar para remover"
}

# ---------------------------------------------------------------------------
# Resumo final
# ---------------------------------------------------------------------------
Write-Host ""
Write-Host "  =============================================" -ForegroundColor Cyan
Write-Host "   Germanio removido com sucesso!" -ForegroundColor Green
Write-Host "  =============================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "  O que foi removido:" -ForegroundColor White
Escrever-Info "  - Arquivos em $DiretorioInstalacao"
Escrever-Info "  - Entrada no PATH do usuario"
Escrever-Info "  - Associacao de arquivo .ge"
Escrever-Info "  - Atalhos do Menu Iniciar"
Write-Host ""
Write-Host "  Reinicie o terminal para o PATH ser atualizado." -ForegroundColor Gray
Write-Host ""
