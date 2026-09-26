#Requires -Version 5.0
# =============================================================================
# Flang Programming Language - Instalador Online para Windows
#
# Uso (one-liner / one-liner usage):
#   irm https://raw.githubusercontent.com/flaviokalleu/flang/master/installer/windows/install-online.ps1 | iex
#
#   ou / or:
#   Invoke-WebRequest -Uri https://raw.githubusercontent.com/flaviokalleu/flang/master/installer/windows/install-online.ps1 -OutFile install.ps1; .\install.ps1
#
# Parametros:
#   -Versao          Versao especifica (padrao: ultima)
#   -DiretorioAlvo   Diretorio de instalacao (padrao: C:\Flang)
#   -Silencioso      Sem perguntas interativas
# =============================================================================

param(
    [string]$Versao          = "",
    [string]$DiretorioAlvo   = "C:\Flang",
    [switch]$Silencioso
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

# ---------------------------------------------------------------------------
# Constantes
# ---------------------------------------------------------------------------
$REPO       = "flaviokalleu/flang"
$REPO_URL   = "https://github.com/$REPO"
$API_URL    = "https://api.github.com/repos/$REPO/releases"
$NOME_APP   = "Flang Programming Language"
$EXE_NOME   = "flang.exe"

# ---------------------------------------------------------------------------
# Cores no terminal
# ---------------------------------------------------------------------------
function Escrever-Info   { param([string]$t) Write-Host "  $t" -ForegroundColor Gray }
function Escrever-Passo  { param([string]$t) Write-Host "  >> $t" -ForegroundColor Yellow }
function Escrever-Ok     { param([string]$t) Write-Host "  [OK] $t" -ForegroundColor Green }
function Escrever-Erro   { param([string]$t) Write-Host "  [ERRO] $t" -ForegroundColor Red; exit 1 }
function Escrever-Aviso  { param([string]$t) Write-Host "  [!] $t" -ForegroundColor DarkYellow }

function Escrever-Banner {
    Clear-Host
    Write-Host ""
    Write-Host "  ███████╗██╗      █████╗ ███╗   ██╗ ██████╗ " -ForegroundColor Cyan
    Write-Host "  ██╔════╝██║     ██╔══██╗████╗  ██║██╔════╝ " -ForegroundColor Cyan
    Write-Host "  █████╗  ██║     ███████║██╔██╗ ██║██║  ███╗" -ForegroundColor Cyan
    Write-Host "  ██╔══╝  ██║     ██╔══██║██║╚██╗██║██║   ██║" -ForegroundColor Cyan
    Write-Host "  ██║     ███████╗██║  ██║██║ ╚████║╚██████╔╝" -ForegroundColor Cyan
    Write-Host "  ╚═╝     ╚══════╝╚═╝  ╚═╝╚═╝  ╚═══╝ ╚═════╝ " -ForegroundColor Cyan
    Write-Host ""
    Write-Host "  $NOME_APP" -ForegroundColor White
    Write-Host "  Instalador Online para Windows" -ForegroundColor DarkGray
    Write-Host "  $REPO_URL" -ForegroundColor DarkGray
    Write-Host ""
}

# ---------------------------------------------------------------------------
# Detectar arquitetura
# ---------------------------------------------------------------------------
function Get-Arch {
    $arch = (Get-WmiObject Win32_Processor -ErrorAction SilentlyContinue | Select-Object -First 1).Architecture
    # 0=x86, 9=x86-64, 12=ARM64
    if ($arch -eq 12) { return "arm64" }
    return "amd64"
}

# ---------------------------------------------------------------------------
# Obter versao mais recente do GitHub
# ---------------------------------------------------------------------------
function Get-UltimaVersao {
    Escrever-Passo "Consultando versao mais recente no GitHub..."
    try {
        $resp = Invoke-RestMethod -Uri "$API_URL/latest" -UseBasicParsing -ErrorAction Stop
        $tag = $resp.tag_name -replace '^v', ''
        Escrever-Ok "Versao mais recente: v$tag"
        return $tag
    } catch {
        Escrever-Erro "Nao foi possivel consultar o GitHub. Verifique sua conexao com a internet."
    }
}

# ---------------------------------------------------------------------------
# Baixar binario do GitHub Releases
# ---------------------------------------------------------------------------
function Download-Binario {
    param([string]$VersaoAlvo, [string]$Arch, [string]$TempDir)

    $arquivo   = "flang-v${VersaoAlvo}-windows-${Arch}.zip"
    $urlDownload = "$REPO_URL/releases/download/v${VersaoAlvo}/${arquivo}"
    $destZip    = Join-Path $TempDir $arquivo

    Escrever-Passo "Baixando: $arquivo"
    Escrever-Info  "  URL: $urlDownload"

    try {
        $progressPreference = $global:ProgressPreference
        $global:ProgressPreference = 'SilentlyContinue'
        Invoke-WebRequest -Uri $urlDownload -OutFile $destZip -UseBasicParsing -ErrorAction Stop
        $global:ProgressPreference = $progressPreference
    } catch {
        Escrever-Aviso "Download do binario pre-compilado falhou. Tentando compilar do codigo-fonte..."
        return Build-DoFonte $TempDir
    }

    Escrever-Passo "Extraindo arquivos..."
    Expand-Archive -Path $destZip -DestinationPath $TempDir -Force

    $exe = Get-ChildItem -Path $TempDir -Filter $EXE_NOME -Recurse -ErrorAction SilentlyContinue | Select-Object -First 1
    if (-not $exe) {
        Escrever-Aviso "Executavel nao encontrado no arquivo. Tentando compilar do codigo-fonte..."
        return Build-DoFonte $TempDir
    }

    return $exe.FullName
}

# ---------------------------------------------------------------------------
# Fallback: compilar do codigo fonte
# ---------------------------------------------------------------------------
function Build-DoFonte {
    param([string]$TempDir)

    Escrever-Passo "Compilando do codigo-fonte (requer Go instalado)..."

    $go = Get-Command go -ErrorAction SilentlyContinue
    if (-not $go) {
        Escrever-Erro "Go nao encontrado. Instale em https://go.dev/dl/ e tente novamente."
    }

    $repoClone = Join-Path $TempDir "flang-src"
    $git = Get-Command git -ErrorAction SilentlyContinue
    if (-not $git) {
        Escrever-Erro "git nao encontrado. Instale em https://git-scm.com/download/win"
    }

    git clone --depth=1 "$REPO_URL.git" $repoClone 2>&1 | Out-Null
    Push-Location $repoClone
    try {
        $env:CGO_ENABLED = "0"
        go build -ldflags="-s -w" -o "$TempDir\$EXE_NOME" .
    } finally {
        Pop-Location
    }

    $exePath = Join-Path $TempDir $EXE_NOME
    if (-not (Test-Path $exePath)) {
        Escrever-Erro "Compilacao falhou. Verifique os logs acima."
    }

    Escrever-Ok "Compilacao concluida!"
    return $exePath
}

# ---------------------------------------------------------------------------
# Instalar binario
# ---------------------------------------------------------------------------
function Instalar-Binario {
    param([string]$ExeFonte, [string]$DirBin)

    Escrever-Passo "Instalando em $DirBin..."
    if (-not (Test-Path $DirBin)) {
        New-Item -ItemType Directory -Path $DirBin -Force | Out-Null
    }

    $dest = Join-Path $DirBin $EXE_NOME
    Copy-Item -Path $ExeFonte -Destination $dest -Force
    Escrever-Ok "flang.exe instalado"
    return $dest
}

# ---------------------------------------------------------------------------
# Adicionar ao PATH do usuario (sem admin)
# ---------------------------------------------------------------------------
function Adicionar-AoPath {
    param([string]$DirBin)

    Escrever-Passo "Adicionando $DirBin ao PATH do usuario..."
    $chaveRegistro = "HKCU:\Environment"
    $pathAtual = (Get-ItemProperty -Path $chaveRegistro -Name "Path" -ErrorAction SilentlyContinue).Path
    if (-not $pathAtual) { $pathAtual = "" }

    $entradas = $pathAtual -split ";" | Where-Object { $_ -ne "" }
    if ($entradas -contains $DirBin) {
        Escrever-Ok "$DirBin ja estava no PATH"
        return
    }

    $novoPath = ($entradas + $DirBin) -join ";"
    Set-ItemProperty -Path $chaveRegistro -Name "Path" -Value $novoPath -Type ExpandString

    # Notifica Windows sobre mudanca de PATH
    try {
        $sig = '[DllImport("user32.dll",SetLastError=true,CharSet=CharSet.Auto)]public static extern IntPtr SendMessageTimeout(IntPtr hWnd,uint Msg,UIntPtr wParam,string lParam,uint fuFlags,uint uTimeout,out UIntPtr lpdwResult);'
        $t = Add-Type -MemberDefinition $sig -Name "WinMsg" -Namespace "Win32" -PassThru -ErrorAction SilentlyContinue
        if ($t) {
            $r = [UIntPtr]::Zero
            $t::SendMessageTimeout([IntPtr]0xffff, 0x1A, [UIntPtr]::Zero, "Environment", 2, 5000, [ref]$r) | Out-Null
        }
    } catch {}

    Escrever-Ok "PATH atualizado (nivel usuario)"
}

# ---------------------------------------------------------------------------
# Associar .ge ao Flang (sem admin)
# ---------------------------------------------------------------------------
function Criar-AssociacaoFg {
    param([string]$ExeDestino)

    Escrever-Passo "Associando arquivos .ge ao Flang..."
    $classe = "FlangFile"

    New-Item -Path "HKCU:\Software\Classes\.ge"          -Force | Out-Null
    Set-ItemProperty -Path "HKCU:\Software\Classes\.ge" -Name "(Default)" -Value $classe

    New-Item -Path "HKCU:\Software\Classes\$classe"       -Force | Out-Null
    Set-ItemProperty -Path "HKCU:\Software\Classes\$classe" -Name "(Default)" -Value "Arquivo Flang (.ge)"

    New-Item -Path "HKCU:\Software\Classes\$classe\DefaultIcon" -Force | Out-Null
    Set-ItemProperty -Path "HKCU:\Software\Classes\$classe\DefaultIcon" -Name "(Default)" -Value "$ExeDestino,0"

    New-Item -Path "HKCU:\Software\Classes\$classe\shell\open\command" -Force | Out-Null
    Set-ItemProperty -Path "HKCU:\Software\Classes\$classe\shell\open\command" -Name "(Default)" -Value "`"$ExeDestino`" run `"%1`""

    try {
        $sig = '[DllImport("shell32.dll")]public static extern void SHChangeNotify(int e,int f,IntPtr a,IntPtr b);'
        $t = Add-Type -MemberDefinition $sig -Name "ShellN" -Namespace "Win32" -PassThru -ErrorAction SilentlyContinue
        if ($t) { $t::SHChangeNotify(0x08000000, 0, [IntPtr]::Zero, [IntPtr]::Zero) }
    } catch {}

    Escrever-Ok "Arquivos .ge associados ao Flang"
}

# ---------------------------------------------------------------------------
# Criar atalho no Menu Iniciar
# ---------------------------------------------------------------------------
function Criar-AtalhoMenuIniciar {
    param([string]$ExeDestino, [string]$DirInstalacao, [string]$VersaoInstalada)

    Escrever-Passo "Criando atalho no Menu Iniciar..."
    $startDir = Join-Path $env:APPDATA "Microsoft\Windows\Start Menu\Programs\Flang"
    if (-not (Test-Path $startDir)) { New-Item -ItemType Directory -Path $startDir -Force | Out-Null }

    $wsh = New-Object -ComObject WScript.Shell

    $a = $wsh.CreateShortcut("$startDir\Flang Terminal.lnk")
    $a.TargetPath       = "cmd.exe"
    $a.Arguments        = "/K `"$ExeDestino`" version"
    $a.WorkingDirectory = $DirInstalacao
    $a.Description      = "Flang Programming Language v$VersaoInstalada"
    $a.IconLocation     = "$ExeDestino,0"
    $a.Save()

    Escrever-Ok "Atalho criado no Menu Iniciar > Flang"
}

# ---------------------------------------------------------------------------
# Criar script de desinstalacao
# ---------------------------------------------------------------------------
function Criar-Desinstalador {
    param([string]$DirInstalacao, [string]$DirBin)

    $script = @"
#Requires -Version 5.0
# Desinstalador do Flang — gerado em $(Get-Date -Format 'dd/MM/yyyy')
Write-Host "`n  Desinstalando Flang..." -ForegroundColor Yellow
`$r = Read-Host "  Confirma remocao? [s/N]"
if (`$r -notmatch '^[Ss]') { Write-Host "  Cancelado." -ForegroundColor Gray; exit 0 }

if (Test-Path "$DirInstalacao") { Remove-Item -Path "$DirInstalacao" -Recurse -Force }
`$reg = "HKCU:\Environment"
`$p = (Get-ItemProperty -Path `$reg -Name Path -ErrorAction SilentlyContinue).Path
if (`$p) { Set-ItemProperty -Path `$reg -Name Path -Value ((`$p -split ';' | Where { `$_ -ne "$DirBin" }) -join ';') -Type ExpandString }
foreach (`$k in @("HKCU:\Software\Classes\.ge","HKCU:\Software\Classes\FlangFile")) {
    if (Test-Path `$k) { Remove-Item -Path `$k -Recurse -Force }
}
`$sm = Join-Path `$env:APPDATA "Microsoft\Windows\Start Menu\Programs\Flang"
if (Test-Path `$sm) { Remove-Item -Path `$sm -Recurse -Force }
Write-Host "  Flang removido com sucesso. Reinicie o terminal." -ForegroundColor Cyan
"@
    $script | Set-Content -Path (Join-Path $DirInstalacao "desinstalar.ps1") -Encoding UTF8
    Escrever-Ok "Desinstalador criado em $DirInstalacao\desinstalar.ps1"
}

# ---------------------------------------------------------------------------
# Resumo
# ---------------------------------------------------------------------------
function Mostrar-Resumo {
    param([string]$DirBin, [string]$VersaoInstalada, [string]$DirInstalacao)

    Write-Host ""
    Write-Host "  =============================================" -ForegroundColor Cyan
    Write-Host "   Flang v$VersaoInstalada instalado com sucesso!" -ForegroundColor Green
    Write-Host "  =============================================" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "  Local de instalacao: $DirInstalacao" -ForegroundColor Gray
    Write-Host ""
    Write-Host "  Proximos passos:" -ForegroundColor White
    Write-Host "    1. Reinicie o terminal (ou PowerShell)" -ForegroundColor Gray
    Write-Host "    2. Digite: flang version" -ForegroundColor Gray
    Write-Host "    3. Crie um app: flang new meu-app" -ForegroundColor Gray
    Write-Host ""
    Write-Host "  Documentacao: $REPO_URL/tree/master/docs" -ForegroundColor DarkGray
    Write-Host "  Para desinstalar: powershell -File `"$DirInstalacao\desinstalar.ps1`"" -ForegroundColor DarkGray
    Write-Host ""
}

# ===========================================================================
# EXECUCAO PRINCIPAL
# ===========================================================================
Escrever-Banner

# Determinar versao a instalar
if ([string]::IsNullOrWhiteSpace($Versao)) {
    $Versao = Get-UltimaVersao
}

# Confirmar com usuario
if (-not $Silencioso) {
    Write-Host "  Versao a instalar : v$Versao" -ForegroundColor White
    Write-Host "  Diretorio destino : $DiretorioAlvo" -ForegroundColor White
    Write-Host "  Arquitetura       : $(Get-Arch)" -ForegroundColor White
    Write-Host ""
    $r = Read-Host "  Continuar com a instalacao? [S/n]"
    if ($r -match "^[Nn]") { Write-Host "  Instalacao cancelada." -ForegroundColor Yellow; exit 0 }
}

$arch    = Get-Arch
$tempDir = Join-Path $env:TEMP "flang-install-$(Get-Date -Format 'yyyyMMddHHmmss')"
New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

try {
    $exeFonte  = Download-Binario -VersaoAlvo $Versao -Arch $arch -TempDir $tempDir
    $dirBin    = Join-Path $DiretorioAlvo "bin"
    $exeDest   = Instalar-Binario -ExeFonte $exeFonte -DirBin $dirBin

    Adicionar-AoPath   -DirBin $dirBin
    Criar-AssociacaoFg -ExeDestino $exeDest
    Criar-Desinstalador -DirInstalacao $DiretorioAlvo -DirBin $dirBin
    Criar-AtalhoMenuIniciar -ExeDestino $exeDest -DirInstalacao $DiretorioAlvo -VersaoInstalada $Versao

    Mostrar-Resumo -DirBin $dirBin -VersaoInstalada $Versao -DirInstalacao $DiretorioAlvo
} finally {
    Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
}
