#Requires -Version 5.0
# =============================================================================
# Flang Programming Language - Instalador para Windows
# Versao 0.2.0
#
# Uso:
#   .\install.ps1
#   .\install.ps1 -DiretorioInstalacao "D:\Flang"
#   .\install.ps1 -Silencioso
#
# Nao requer privilégios de administrador (instala por usuario).
# =============================================================================

param(
    # Diretório de instalação (padrão: C:\Flang)
    [string]$DiretorioInstalacao = "C:\Flang",

    # Instala sem perguntas interativas
    [switch]$Silencioso
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# ---------------------------------------------------------------------------
# Variaveis globais
# ---------------------------------------------------------------------------
$VersaoFlang    = "0.2.0"
$NomeApp        = "Flang Programming Language"
$ExeNome        = "flang.exe"
$DirBin         = Join-Path $DiretorioInstalacao "bin"
$DirExemplos    = Join-Path $DiretorioInstalacao "exemplos"
$DirDocs        = Join-Path $DiretorioInstalacao "docs"
$ExeDestino     = Join-Path $DirBin $ExeNome

# Caminho do binário ao lado deste script (ou na raiz do projeto)
$ScriptDir      = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjetoRaiz    = Split-Path -Parent (Split-Path -Parent $ScriptDir)
$ExeFonte       = Join-Path $ProjetoRaiz $ExeNome

# ---------------------------------------------------------------------------
# Funcoes auxiliares
# ---------------------------------------------------------------------------

function Escrever-Titulo {
    param([string]$Texto)
    Write-Host ""
    Write-Host "  $Texto" -ForegroundColor Cyan
    Write-Host ("  " + ("=" * ($Texto.Length))) -ForegroundColor DarkCyan
}

function Escrever-Passo {
    param([string]$Texto)
    Write-Host "  >> $Texto" -ForegroundColor Yellow
}

function Escrever-Ok {
    param([string]$Texto)
    Write-Host "  [OK] $Texto" -ForegroundColor Green
}

function Escrever-Erro {
    param([string]$Texto)
    Write-Host "  [ERRO] $Texto" -ForegroundColor Red
}

function Escrever-Info {
    param([string]$Texto)
    Write-Host "  $Texto" -ForegroundColor Gray
}

# ---------------------------------------------------------------------------
# Banner de boas-vindas
# ---------------------------------------------------------------------------
function Mostrar-Banner {
    Clear-Host
    Write-Host ""
    Write-Host "  ███████╗██╗      █████╗ ███╗   ██╗ ██████╗ " -ForegroundColor Cyan
    Write-Host "  ██╔════╝██║     ██╔══██╗████╗  ██║██╔════╝ " -ForegroundColor Cyan
    Write-Host "  █████╗  ██║     ███████║██╔██╗ ██║██║  ███╗" -ForegroundColor Cyan
    Write-Host "  ██╔══╝  ██║     ██╔══██║██║╚██╗██║██║   ██║" -ForegroundColor Cyan
    Write-Host "  ██║     ███████╗██║  ██║██║ ╚████║╚██████╔╝" -ForegroundColor Cyan
    Write-Host "  ╚═╝     ╚══════╝╚═╝  ╚═╝╚═╝  ╚═══╝ ╚═════╝ " -ForegroundColor Cyan
    Write-Host ""
    Write-Host "  $NomeApp v$VersaoFlang" -ForegroundColor White
    Write-Host "  Instalador para Windows" -ForegroundColor DarkGray
    Write-Host ""
}

# ---------------------------------------------------------------------------
# Confirmacao do usuario
# ---------------------------------------------------------------------------
function Confirmar-Instalacao {
    if ($Silencioso) { return }

    Write-Host "  Diretório de instalacao: " -NoNewline
    Write-Host $DiretorioInstalacao -ForegroundColor White
    Write-Host ""
    $resposta = Read-Host "  Continuar? [S/n]"
    if ($resposta -match "^[Nn]") {
        Write-Host ""
        Write-Host "  Instalacao cancelada." -ForegroundColor Yellow
        exit 0
    }
}

# ---------------------------------------------------------------------------
# Verificar se o binario fonte existe
# ---------------------------------------------------------------------------
function Verificar-BinarioFonte {
    Escrever-Passo "Procurando flang.exe..."

    # Tenta encontrar o exe em varios locais possiveis
    $candidatos = @(
        $ExeFonte,
        (Join-Path $ScriptDir $ExeNome),
        (Join-Path $ScriptDir "..\..\$ExeNome"),
        (Join-Path (Get-Location) $ExeNome)
    )

    foreach ($c in $candidatos) {
        if (Test-Path $c) {
            $script:ExeFonte = $c
            Escrever-Ok "Binario encontrado: $c"
            return
        }
    }

    Escrever-Erro "flang.exe nao encontrado. Compile primeiro com:"
    Escrever-Info "  go build -o flang.exe ."
    Escrever-Erro "ou execute build-installer.bat"
    exit 1
}

# ---------------------------------------------------------------------------
# Criar estrutura de diretorios
# ---------------------------------------------------------------------------
function Criar-Diretorios {
    Escrever-Passo "Criando estrutura de diretorios em $DiretorioInstalacao..."

    $dirs = @($DiretorioInstalacao, $DirBin, $DirExemplos, $DirDocs)
    foreach ($dir in $dirs) {
        if (-not (Test-Path $dir)) {
            New-Item -ItemType Directory -Path $dir -Force | Out-Null
        }
    }

    Escrever-Ok "Diretorios criados"
}

# ---------------------------------------------------------------------------
# Copiar o binario principal
# ---------------------------------------------------------------------------
function Instalar-Binario {
    Escrever-Passo "Instalando flang.exe em $DirBin..."

    Copy-Item -Path $ExeFonte -Destination $ExeDestino -Force
    Escrever-Ok "flang.exe instalado"
}

# ---------------------------------------------------------------------------
# Copiar exemplos .ge
# ---------------------------------------------------------------------------
function Instalar-Exemplos {
    Escrever-Passo "Copiando exemplos..."

    # Localiza pasta de exemplos relativa ao projeto
    $exemplosFonte = Join-Path $ProjetoRaiz "exemplos"

    if (Test-Path $exemplosFonte) {
        # Copia cada subpasta de exemplo
        Get-ChildItem -Path $exemplosFonte -Directory | ForEach-Object {
            $destino = Join-Path $DirExemplos $_.Name
            Copy-Item -Path $_.FullName -Destination $destino -Recurse -Force
            Escrever-Info "    + $($_.Name)"
        }
        Escrever-Ok "Exemplos copiados para $DirExemplos"
    } else {
        # Cria um exemplo basico inline caso nao haja pasta de exemplos
        $exemploBasico = @"
# Exemplo basico - sistema de loja Flang
system loja

theme
  color primary "#3b82f6"
  color secondary "#8b5cf6"

models

  produto
    nome: texto obrigatorio
    preco: dinheiro obrigatorio
    categoria: texto
    status: status

screens

  tela produtos
    titulo "Meus Produtos"
    lista produto
      mostrar nome
      mostrar preco
      mostrar categoria
    botao azul
      texto "Novo Produto"

events

  quando clicar "Novo Produto"
    criar produto

logic

  validar preco maior 0
"@
        $exemploDir = Join-Path $DirExemplos "ola-mundo"
        New-Item -ItemType Directory -Path $exemploDir -Force | Out-Null
        $exemploBasico | Set-Content -Path (Join-Path $exemploDir "inicio.ge") -Encoding UTF8
        Escrever-Ok "Exemplo basico criado em $exemploDir"
    }
}

# ---------------------------------------------------------------------------
# Copiar documentacao
# ---------------------------------------------------------------------------
function Instalar-Docs {
    Escrever-Passo "Copiando documentacao..."

    $docsFonte = Join-Path $ProjetoRaiz "docs"
    if (Test-Path $docsFonte) {
        Copy-Item -Path "$docsFonte\*" -Destination $DirDocs -Recurse -Force
        Escrever-Ok "Documentacao copiada"
    }

    # Cria README rapido
    $readme = @"
Flang Programming Language v$VersaoFlang
========================================

Instalado em: $DiretorioInstalacao

COMO USAR
---------
  flang version
  flang run meu_app.ge
  flang new minha_loja

EXEMPLOS
--------
  Os exemplos estao em: $DirExemplos

  Para rodar um exemplo:
    flang run "$DirExemplos\ola-mundo\inicio.ge"

MAIS INFORMACOES
----------------
  GitHub: https://github.com/flaviokalleu/flang
  Data de instalacao: $(Get-Date -Format "dd/MM/yyyy HH:mm")
"@
    $readme | Set-Content -Path (Join-Path $DiretorioInstalacao "LEIA-ME.txt") -Encoding UTF8
    Escrever-Ok "LEIA-ME.txt criado"
}

# ---------------------------------------------------------------------------
# Adicionar ao PATH do usuario (sem admin - via registro HKCU)
# ---------------------------------------------------------------------------
function Adicionar-AoPath {
    Escrever-Passo "Adicionando $DirBin ao PATH do usuario..."

    $chaveRegistro = "HKCU:\Environment"
    $pathAtual = (Get-ItemProperty -Path $chaveRegistro -Name "Path" -ErrorAction SilentlyContinue).Path

    if ($null -eq $pathAtual) {
        $pathAtual = ""
    }

    # Verifica se ja esta no PATH
    $entradas = $pathAtual -split ";" | Where-Object { $_ -ne "" }
    if ($entradas -contains $DirBin) {
        Escrever-Ok "$DirBin ja esta no PATH"
        return
    }

    # Acrescenta o novo caminho
    $novoPath = ($entradas + $DirBin) -join ";"

    # Grava no registro de forma permanente (nivel usuario)
    Set-ItemProperty -Path $chaveRegistro -Name "Path" -Value $novoPath -Type ExpandString

    # Notifica o Windows sobre a mudanca de ambiente (sem reiniciar)
    try {
        $assinatura = @"
[DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)]
public static extern IntPtr SendMessageTimeout(
    IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam,
    uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);
"@
        $tipo = Add-Type -MemberDefinition $assinatura -Name "Win32SendMessage" -Namespace "Win32Functions" -PassThru -ErrorAction SilentlyContinue
        if ($tipo) {
            $HWND_BROADCAST  = [IntPtr]0xffff
            $WM_SETTINGCHANGE = 0x001A
            $result = [UIntPtr]::Zero
            $tipo::SendMessageTimeout($HWND_BROADCAST, $WM_SETTINGCHANGE, [UIntPtr]::Zero, "Environment", 2, 5000, [ref]$result) | Out-Null
        }
    } catch {
        # Ignora erros na notificacao — o PATH estara correto apos reabrir o terminal
    }

    Escrever-Ok "PATH atualizado (nivel usuario, permanente)"
    Escrever-Info "    Reinicie o terminal para o PATH entrar em vigor"
}

# ---------------------------------------------------------------------------
# Criar associacao de arquivo .ge (sem admin - HKCU)
# ---------------------------------------------------------------------------
function Criar-AssociacaoArquivo {
    Escrever-Passo "Criando associacao de arquivo .ge..."

    # Classe do programa
    $classeNome = "FlangFile"
    $descricao  = "Arquivo Flang (.ge)"
    $icone      = "$ExeDestino,0"

    # 1. Registra a extensao .ge apontando para a classe
    $extKey = "HKCU:\Software\Classes\.ge"
    New-Item -Path $extKey -Force | Out-Null
    Set-ItemProperty -Path $extKey -Name "(Default)" -Value $classeNome

    # 2. Define a classe (descricao, icone, comando de abertura)
    $classeKey = "HKCU:\Software\Classes\$classeNome"
    New-Item -Path $classeKey -Force | Out-Null
    Set-ItemProperty -Path $classeKey -Name "(Default)" -Value $descricao

    # Icone
    $iconeKey = "$classeKey\DefaultIcon"
    New-Item -Path $iconeKey -Force | Out-Null
    Set-ItemProperty -Path $iconeKey -Name "(Default)" -Value $icone

    # Comando de abertura (flang run "arquivo.ge")
    $cmdKey = "$classeKey\shell\open\command"
    New-Item -Path $cmdKey -Force | Out-Null
    Set-ItemProperty -Path $cmdKey -Name "(Default)" -Value "`"$ExeDestino`" run `"%1`""

    # Notifica o Explorer sobre a mudanca
    try {
        $code = @"
[DllImport("shell32.dll")]
public static extern void SHChangeNotify(int wEventId, int uFlags, IntPtr dwItem1, IntPtr dwItem2);
"@
        $shell = Add-Type -MemberDefinition $code -Name "ShellNotify" -Namespace "Win32" -PassThru -ErrorAction SilentlyContinue
        if ($shell) {
            $SHCNE_ASSOCCHANGED = 0x08000000
            $SHCNF_IDLIST       = 0x0000
            $shell::SHChangeNotify($SHCNE_ASSOCCHANGED, $SHCNF_IDLIST, [IntPtr]::Zero, [IntPtr]::Zero)
        }
    } catch { }

    Escrever-Ok "Arquivos .ge agora abrem com Flang ao dar duplo clique"
}

# ---------------------------------------------------------------------------
# Criar o desinstalador
# ---------------------------------------------------------------------------
function Criar-Desinstalador {
    Escrever-Passo "Criando desinstalador..."

    $conteudo = @"
#Requires -Version 5.0
# =============================================================================
# Flang Programming Language - Desinstalador
# Gerado automaticamente em $(Get-Date -Format "dd/MM/yyyy HH:mm")
# =============================================================================

`$DiretorioInstalacao = "$DiretorioInstalacao"
`$DirBin              = "$DirBin"

Write-Host ""
Write-Host "  Desinstalando Flang Programming Language..." -ForegroundColor Yellow
Write-Host ""

# Confirmacao
`$resposta = Read-Host "  Tem certeza que deseja remover o Flang? [s/N]"
if (`$resposta -notmatch "^[Ss]") {
    Write-Host "  Desinstalacao cancelada." -ForegroundColor Gray
    exit 0
}

# 1. Remove o diretorio de instalacao
if (Test-Path `$DiretorioInstalacao) {
    Write-Host "  Removendo `$DiretorioInstalacao..." -ForegroundColor Gray
    Remove-Item -Path `$DiretorioInstalacao -Recurse -Force
    Write-Host "  [OK] Diretorio removido" -ForegroundColor Green
} else {
    Write-Host "  [!] Diretorio nao encontrado: `$DiretorioInstalacao" -ForegroundColor Yellow
}

# 2. Remove do PATH do usuario
`$chaveRegistro = "HKCU:\Environment"
`$pathAtual = (Get-ItemProperty -Path `$chaveRegistro -Name "Path" -ErrorAction SilentlyContinue).Path
if (`$pathAtual) {
    `$novoPath = (`$pathAtual -split ";" | Where-Object { `$_ -ne `$DirBin }) -join ";"
    Set-ItemProperty -Path `$chaveRegistro -Name "Path" -Value `$novoPath -Type ExpandString
    Write-Host "  [OK] Removido do PATH" -ForegroundColor Green
}

# 3. Remove associacao de arquivo .ge
`$chaves = @(
    "HKCU:\Software\Classes\.ge",
    "HKCU:\Software\Classes\FlangFile"
)
foreach (`$chave in `$chaves) {
    if (Test-Path `$chave) {
        Remove-Item -Path `$chave -Recurse -Force
    }
}
Write-Host "  [OK] Associacao de arquivo .ge removida" -ForegroundColor Green

# 4. Notifica o Explorer
try {
    `$code = @"
[DllImport("shell32.dll")]
public static extern void SHChangeNotify(int wEventId, int uFlags, IntPtr dwItem1, IntPtr dwItem2);
"@
    `$shell = Add-Type -MemberDefinition `$code -Name "ShellNotify2" -Namespace "Win32" -PassThru -ErrorAction SilentlyContinue
    if (`$shell) {
        `$shell::SHChangeNotify(0x08000000, 0x0000, [IntPtr]::Zero, [IntPtr]::Zero)
    }
} catch { }

Write-Host ""
Write-Host "  Flang foi desinstalado com sucesso." -ForegroundColor Cyan
Write-Host "  Reinicie o terminal para o PATH ser atualizado." -ForegroundColor Gray
Write-Host ""
"@

    $conteudo | Set-Content -Path (Join-Path $DiretorioInstalacao "desinstalar.ps1") -Encoding UTF8
    Escrever-Ok "Desinstalador criado em $DiretorioInstalacao\desinstalar.ps1"
}

# ---------------------------------------------------------------------------
# Criar atalho no Menu Iniciar (nivel usuario, sem admin)
# ---------------------------------------------------------------------------
function Criar-AtalhoMenuIniciar {
    Escrever-Passo "Criando atalhos no Menu Iniciar..."

    $startMenuDir = Join-Path $env:APPDATA "Microsoft\Windows\Start Menu\Programs\Flang"
    if (-not (Test-Path $startMenuDir)) {
        New-Item -ItemType Directory -Path $startMenuDir -Force | Out-Null
    }

    $wsh = New-Object -ComObject WScript.Shell

    # Atalho principal - terminal Flang
    $atalho = $wsh.CreateShortcut("$startMenuDir\Flang.lnk")
    $atalho.TargetPath      = "cmd.exe"
    $atalho.Arguments       = "/K `"$ExeDestino`" version"
    $atalho.WorkingDirectory = $DiretorioInstalacao
    $atalho.Description     = "Flang Programming Language v$VersaoFlang"
    $atalho.IconLocation    = "$ExeDestino,0"
    $atalho.Save()

    # Atalho para a pasta de exemplos
    $atalhoEx = $wsh.CreateShortcut("$startMenuDir\Exemplos Flang.lnk")
    $atalhoEx.TargetPath = $DirExemplos
    $atalhoEx.Save()

    # Atalho para o desinstalador
    $atalhoDesins = $wsh.CreateShortcut("$startMenuDir\Desinstalar Flang.lnk")
    $atalhoDesins.TargetPath  = "powershell.exe"
    $atalhoDesins.Arguments   = "-ExecutionPolicy Bypass -File `"$DiretorioInstalacao\desinstalar.ps1`""
    $atalhoDesins.Description = "Remover o Flang do sistema"
    $atalhoDesins.Save()

    Escrever-Ok "Atalhos criados no Menu Iniciar > Flang"
}

# ---------------------------------------------------------------------------
# Resumo final
# ---------------------------------------------------------------------------
function Mostrar-Resumo {
    Write-Host ""
    Write-Host "  =============================================" -ForegroundColor Cyan
    Write-Host "   Instalacao concluida com sucesso!" -ForegroundColor Green
    Write-Host "  =============================================" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "  Flang $VersaoFlang instalado em:" -ForegroundColor White
    Write-Host "    $DiretorioInstalacao" -ForegroundColor Gray
    Write-Host ""
    Write-Host "  Exemplos disponiveis em:" -ForegroundColor White
    Write-Host "    $DirExemplos" -ForegroundColor Gray
    Write-Host ""
    Write-Host "  Para rodar um exemplo:" -ForegroundColor White
    Write-Host "    flang run `"$DirExemplos\ola-mundo\inicio.ge`"" -ForegroundColor Gray
    Write-Host ""
    Write-Host "  Para desinstalar:" -ForegroundColor White
    Write-Host "    powershell -File `"$DiretorioInstalacao\desinstalar.ps1`"" -ForegroundColor Gray
    Write-Host ""
    Write-Host "  Flang instalado! Reinicie o terminal e digite: flang version" -ForegroundColor Cyan
    Write-Host ""
}

# ---------------------------------------------------------------------------
# EXECUCAO PRINCIPAL
# ---------------------------------------------------------------------------
Mostrar-Banner
Confirmar-Instalacao
Escrever-Titulo "Iniciando instalacao do Flang v$VersaoFlang"

Verificar-BinarioFonte
Criar-Diretorios
Instalar-Binario
Instalar-Exemplos
Instalar-Docs
Adicionar-AoPath
Criar-AssociacaoArquivo
Criar-Desinstalador
Criar-AtalhoMenuIniciar

Mostrar-Resumo
