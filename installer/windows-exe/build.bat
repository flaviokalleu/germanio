@echo off
setlocal

echo.
echo  Construindo instalador Flang...
echo.

set BASE=%~dp0..\..
set PAYLOAD=%~dp0payload

:: Limpar payload anterior
if exist "%PAYLOAD%" rd /s /q "%PAYLOAD%"

:: Criar estrutura
mkdir "%PAYLOAD%\bin" 2>nul
mkdir "%PAYLOAD%\exemplos\loja-completa" 2>nul
mkdir "%PAYLOAD%\exemplos\english" 2>nul
mkdir "%PAYLOAD%\exemplos\mixed" 2>nul
mkdir "%PAYLOAD%\exemplos\restaurante-modular" 2>nul
mkdir "%PAYLOAD%\docs" 2>nul

:: Compilar Flang
echo [1/4] Compilando flang.exe...
pushd "%BASE%"
go build -ldflags="-s -w" -o "%PAYLOAD%\bin\flang.exe" .
if errorlevel 1 (
    echo ERRO ao compilar flang
    exit /b 1
)
popd
echo    OK

:: Copiar exemplos
echo [2/4] Copiando exemplos...
xcopy /s /q /y "%BASE%\exemplos\loja-completa\*" "%PAYLOAD%\exemplos\loja-completa\" >nul 2>nul
xcopy /s /q /y "%BASE%\exemplos\english\*" "%PAYLOAD%\exemplos\english\" >nul 2>nul
xcopy /s /q /y "%BASE%\exemplos\mixed\*" "%PAYLOAD%\exemplos\mixed\" >nul 2>nul
xcopy /s /q /y "%BASE%\exemplos\restaurante-modular\*" "%PAYLOAD%\exemplos\restaurante-modular\" >nul 2>nul
echo    OK

:: Copiar docs
echo [3/4] Copiando documentação...
copy /y "%BASE%\README.md" "%PAYLOAD%\docs\" >nul 2>nul
copy /y "%BASE%\LICENSE" "%PAYLOAD%\docs\" >nul 2>nul
copy /y "%BASE%\docs\TUTORIAL.md" "%PAYLOAD%\docs\" >nul 2>nul
copy /y "%BASE%\docs\CHEATSHEET.md" "%PAYLOAD%\docs\" >nul 2>nul
echo    OK

:: Compilar instalador
echo [4/4] Compilando instalador...
pushd "%~dp0"
go build -tags installer -ldflags="-s -w -H windowsgui" -o "%BASE%\dist\FlangSetup-0.5.0.exe" .
if errorlevel 1 (
    echo ERRO ao compilar instalador
    exit /b 1
)
popd
echo    OK

:: Também compilar versão console (com output no terminal)
pushd "%~dp0"
go build -tags installer -ldflags="-s -w" -o "%BASE%\dist\FlangSetup-0.5.0-console.exe" .
popd

echo.
echo  ══════════════════════════════════════════════
echo   Instalador criado: dist\FlangSetup-0.5.0.exe
echo  ══════════════════════════════════════════════
echo.

:: Limpar payload
rd /s /q "%PAYLOAD%" 2>nul

endlocal
