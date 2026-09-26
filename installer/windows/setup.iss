; =============================================================================
; Germanio Programming Language - Script Inno Setup
; Versao 0.4.0
;
; Cria um instalador .exe profissional para Windows, similar ao do Python.
;
; Para compilar este script:
;   1. Instale o Inno Setup: https://jrsoftware.org/isdl.php
;   2. Abra este arquivo no Inno Setup Compiler
;   3. Pressione F9 (Build) ou use o menu Build > Compile
;
; O instalador gerado estara em: Output\GermanioSetup-0.4.0.exe
; =============================================================================


; -----------------------------------------------
; Metadados da aplicacao
; -----------------------------------------------
#define AppNome        "Germanio Programming Language"
#define AppVersao      "0.4.0"
#define AppEditora     "Flavio Kalleu"
#define AppURL         "https://github.com/flavio/germanio"
#define AppExe         "germanio.exe"
#define AppDescricao   "Linguagem de programacao brasileira para criar sistemas web"
#define DirPadrao      "C:\Germanio"

; Caminho do binario compilado (relativo a este .iss)
#define BinFonte       "..\..\germanio.exe"

; Pasta de exemplos do projeto
#define ExemplosFonte  "..\..\exemplos\*"


[Setup]

; ----- Identificacao do app -----
AppId={{F1A06C2E-3B7D-4A8F-9E21-5C6D0B2A7F43}
AppName={#AppNome}
AppVersion={#AppVersao}
AppVerName={#AppNome} {#AppVersao}
AppPublisher={#AppEditora}
AppPublisherURL={#AppURL}
AppSupportURL={#AppURL}/issues
AppUpdatesURL={#AppURL}/releases
AppComments={#AppDescricao}

; ----- Diretorio de instalacao -----
; Permite alterar durante a instalacao (como Go e Python fazem)
DefaultDirName={#DirPadrao}
DirExistsWarning=no
; Instala para o usuario atual (sem precisar de admin)
; Mude para "no" se quiser exigir admin
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog

; ----- Grupo no Menu Iniciar -----
DefaultGroupName=Germanio
AllowNoIcons=yes

; ----- Arquivo de saida -----
OutputDir=Output
OutputBaseFilename=GermanioSetup-{#AppVersao}
SetupIconFile=

; ----- Compressao -----
Compression=lzma2/ultra64
SolidCompression=yes
LZMAUseSeparateProcess=yes

; ----- Visual e estilo -----
; Estilo moderno "wizard" similar ao Python/Go
WizardStyle=modern
WizardResizable=yes
WizardSizePercent=130

; ----- Informacoes da licenca -----
LicenseFile=..\..\LICENSE

; ----- Paginas do instalador -----
; Habilita as paginas de boas-vindas e conclusao
DisableWelcomePage=no
DisableFinishedPage=no

; ----- Versao minima do Windows -----
MinVersion=10.0

; ----- Arquitetura -----
; Instala apenas em sistemas de 64 bits
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible

; ----- Desinstalador -----
Uninstallable=yes
UninstallDisplayName={#AppNome}
UninstallDisplayIcon={app}\bin\{#AppExe}
CreateUninstallRegKey=yes

; ----- Registro de versao -----
VersionInfoVersion={#AppVersao}
VersionInfoCompany={#AppEditora}
VersionInfoDescription={#AppDescricao}
VersionInfoProductName={#AppNome}
VersionInfoProductVersion={#AppVersao}


; =============================================================================
; Arquivos a serem instalados
; =============================================================================
[Files]

; Binario principal
Source: "{#BinFonte}"; DestDir: "{app}\bin"; Flags: ignoreversion

; Exemplos .ge
Source: "{#ExemplosFonte}"; DestDir: "{app}\exemplos"; \
    Flags: ignoreversion recursesubdirs createallsubdirs

; Documentacao (se existir)
Source: "..\..\docs\*"; DestDir: "{app}\docs"; \
    Flags: ignoreversion recursesubdirs createallsubdirs skipifsourcedoesntexist

; Licenca
Source: "..\..\LICENSE"; DestDir: "{app}"; Flags: ignoreversion

; Script de desinstalacao PowerShell (complementar)
Source: "uninstall.ps1"; DestDir: "{app}"; Flags: ignoreversion

; README
Source: "..\..\README.md"; DestDir: "{app}"; Flags: ignoreversion skipifsourcedoesntexist


; =============================================================================
; Atalhos no Menu Iniciar
; =============================================================================
[Icons]

; Atalho principal - abre cmd com germanio
Name: "{group}\Germanio"; \
    Filename: "{cmd}"; \
    Parameters: "/K ""{app}\bin\germanio.exe"" version"; \
    WorkingDir: "{app}"; \
    Comment: "Abre o terminal com Germanio {#AppVersao}"

; Atalho para a pasta de exemplos
Name: "{group}\Exemplos Germanio"; \
    Filename: "{app}\exemplos"; \
    Comment: "Exemplos de programas .ge"

; Atalho para o README
Name: "{group}\Documentacao Germanio"; \
    Filename: "{app}\docs"; \
    Comment: "Documentacao do Germanio"

; Atalho para desinstalar
Name: "{group}\Desinstalar Germanio"; \
    Filename: "{uninstallexe}"; \
    Comment: "Remove o Germanio do sistema"

; Atalho na area de trabalho (opcional - o usuario pode desmarcar)
Name: "{userdesktop}\Germanio"; \
    Filename: "{cmd}"; \
    Parameters: "/K ""{app}\bin\germanio.exe"" version"; \
    WorkingDir: "{app}"; \
    Comment: "Germanio {#AppVersao}"; \
    Tasks: desktopicon


; =============================================================================
; Tarefas opcionais (checkboxes na tela de opcoes)
; =============================================================================
[Tasks]

; Adicionar ao PATH (marcado por padrao, igual ao instalador do Python)
Name: "addtopath"; \
    Description: "Adicionar Germanio ao PATH do sistema (recomendado)"; \
    GroupDescription: "Opcoes adicionais:"; \
    Flags: checked

; Criar associacao de arquivo .ge
Name: "assocfg"; \
    Description: "Associar arquivos .ge ao Germanio (duplo clique abre com germanio)"; \
    GroupDescription: "Opcoes adicionais:"; \
    Flags: checked

; Atalho na area de trabalho
Name: "desktopicon"; \
    Description: "Criar atalho na area de trabalho"; \
    GroupDescription: "Opcoes adicionais:"; \
    Flags: unchecked


; =============================================================================
; Modificacoes no registro do Windows
; =============================================================================
[Registry]

; ----- Associacao de arquivo .ge -----
; Registra a extensao .ge (nivel usuario - sem precisar de admin)

; Associa .ge a classe GermanioFile
Root: HKCU; Subkey: "Software\Classes\.ge"; \
    ValueType: string; ValueName: ""; ValueData: "GermanioFile"; \
    Flags: uninsdeletekey; Tasks: assocfg

; Define descricao da classe
Root: HKCU; Subkey: "Software\Classes\GermanioFile"; \
    ValueType: string; ValueName: ""; ValueData: "Arquivo Germanio"; \
    Flags: uninsdeletekey; Tasks: assocfg

; Icone do arquivo .ge (usa o icone do germanio.exe)
Root: HKCU; Subkey: "Software\Classes\GermanioFile\DefaultIcon"; \
    ValueType: string; ValueName: ""; ValueData: "{app}\bin\{#AppExe},0"; \
    Flags: uninsdeletekey; Tasks: assocfg

; Comando para abrir o arquivo (germanio run "arquivo.ge")
Root: HKCU; Subkey: "Software\Classes\GermanioFile\shell\open\command"; \
    ValueType: string; ValueName: ""; \
    ValueData: """{app}\bin\{#AppExe}"" run ""%1"""; \
    Flags: uninsdeletekey; Tasks: assocfg

; Descricao amigavel na caixa de dialogo "Abrir com"
Root: HKCU; Subkey: "Software\Classes\GermanioFile\shell\open"; \
    ValueType: string; ValueName: "FriendlyAppName"; \
    ValueData: "{#AppNome}"; \
    Flags: uninsdeletekey; Tasks: assocfg

; ----- Informacoes de instalacao no Painel de Controle -----
Root: HKCU; Subkey: "Software\{#AppEditora}\{#AppNome}"; \
    ValueType: string; ValueName: "InstallPath"; ValueData: "{app}"; \
    Flags: uninsdeletekey

Root: HKCU; Subkey: "Software\{#AppEditora}\{#AppNome}"; \
    ValueType: string; ValueName: "Version"; ValueData: "{#AppVersao}"; \
    Flags: uninsdeletekey


; =============================================================================
; Codigo Pascal para logica personalizada
; =============================================================================
[Code]

// ---------------------------------------------------------------------------
// Adiciona C:\Germanio\bin ao PATH do usuario (nivel HKCU, sem admin)
// Logica identica a do instalador oficial do Go.
// ---------------------------------------------------------------------------
procedure AdicionarAoPath(DirBin: string);
var
  PathAtual: string;
  NovoPath: string;
begin
  // Le o PATH atual do usuario no registro
  if not RegQueryStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', PathAtual) then
    PathAtual := '';

  // Verifica se ja esta no PATH
  if Pos(LowerCase(DirBin), LowerCase(PathAtual)) > 0 then
  begin
    Log('PATH: ' + DirBin + ' ja esta no PATH do usuario.');
    Exit;
  end;

  // Acrescenta o novo caminho
  if PathAtual = '' then
    NovoPath := DirBin
  else if PathAtual[Length(PathAtual)] = ';' then
    NovoPath := PathAtual + DirBin
  else
    NovoPath := PathAtual + ';' + DirBin;

  // Grava no registro (permanente, nivel usuario)
  RegWriteExpandStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', NovoPath);
  Log('PATH: adicionado ' + DirBin + ' ao PATH do usuario.');
end;

// ---------------------------------------------------------------------------
// Remove C:\Germanio\bin do PATH do usuario durante desinstalacao
// ---------------------------------------------------------------------------
procedure RemoverDoPath(DirBin: string);
var
  PathAtual: string;
  Entradas: TArrayOfString;
  NovoPath: string;
  i: Integer;
begin
  if not RegQueryStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', PathAtual) then
    Exit;

  // Reconstroi o PATH sem a entrada do Germanio
  Entradas := SplitString(PathAtual, ';');
  NovoPath := '';
  for i := 0 to GetArrayLength(Entradas) - 1 do
  begin
    if LowerCase(Entradas[i]) <> LowerCase(DirBin) then
    begin
      if NovoPath = '' then
        NovoPath := Entradas[i]
      else
        NovoPath := NovoPath + ';' + Entradas[i];
    end;
  end;

  RegWriteExpandStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', NovoPath);
  Log('PATH: removido ' + DirBin + ' do PATH do usuario.');
end;

// ---------------------------------------------------------------------------
// Executado apos a instalacao ser concluida
// ---------------------------------------------------------------------------
procedure CurStepChanged(CurStep: TSetupStep);
var
  DirBin: string;
begin
  if CurStep = ssPostInstall then
  begin
    DirBin := ExpandConstant('{app}\bin');

    // Adiciona ao PATH se a tarefa foi selecionada
    if IsTaskSelected('addtopath') then
      AdicionarAoPath(DirBin);
  end;
end;

// ---------------------------------------------------------------------------
// Executado durante a desinstalacao
// ---------------------------------------------------------------------------
procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  DirBin: string;
begin
  if CurUninstallStep = usPostUninstall then
  begin
    DirBin := ExpandConstant('{app}\bin');
    RemoverDoPath(DirBin);
  end;
end;

// ---------------------------------------------------------------------------
// Pagina de boas-vindas personalizada
// Exibe descricao do Germanio antes de comecar a instalacao.
// ---------------------------------------------------------------------------
function GetWelcomeLabel2Text(def: string): string;
begin
  Result :=
    'Bem-vindo ao instalador do Germanio Programming Language v{#AppVersao}!' + #13#10 +
    #13#10 +
    'Germanio e uma linguagem de programacao brasileira para criar sistemas ' +
    'de gestao completos com banco de dados, telas e logica de negocio — ' +
    'usando uma sintaxe simples em portugues.' + #13#10 +
    #13#10 +
    'O que sera instalado:' + #13#10 +
    '  * germanio.exe — o compilador/interpretador' + #13#10 +
    '  * Exemplos de sistemas prontos (.ge)' + #13#10 +
    '  * Atalhos no Menu Iniciar' + #13#10 +
    #13#10 +
    'Nao e necessario ser administrador do sistema.' + #13#10 +
    #13#10 +
    'Clique em Proximo para continuar.';
end;

// ---------------------------------------------------------------------------
// Mensagem de conclusao personalizada
// ---------------------------------------------------------------------------
function GetFinishedHeadingLabel(def: string): string;
begin
  Result := 'Germanio instalado com sucesso!';
end;

function GetFinishedLabel(def: string): string;
var
  DirBin: string;
begin
  DirBin := ExpandConstant('{app}\bin');
  Result :=
    'O Germanio Programming Language foi instalado no seu computador.' + #13#10 +
    #13#10 +
    'Para comecar a usar:' + #13#10 +
    '  1. Reinicie o terminal (cmd ou PowerShell)' + #13#10 +
    '  2. Digite: germanio version' + #13#10 +
    '  3. Para rodar um exemplo:' + #13#10 +
    '     germanio run "' + ExpandConstant('{app}') + '\exemplos\ola-mundo\inicio.ge"' + #13#10 +
    #13#10 +
    'Germanio instalado! Reinicie o terminal e digite: germanio version';
end;
