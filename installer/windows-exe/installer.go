//go:build windows && installer

package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

//go:embed payload/*
var payload embed.FS

const (
	installDir = `C:\Flang`
	version    = "0.6.0"
)

func main() {
	banner()

	// Check for uninstall flag
	if len(os.Args) > 1 && (os.Args[1] == "--uninstall" || os.Args[1] == "/uninstall") {
		uninstall()
		return
	}

	// Clean previous installation
	if _, err := os.Stat(installDir); err == nil {
		fmt.Println("[0/6] Removendo versão anterior...")
		os.RemoveAll(filepath.Join(installDir, "bin"))
		os.RemoveAll(filepath.Join(installDir, "exemplos"))
		os.RemoveAll(filepath.Join(installDir, "docs"))
		fmt.Println("   OK")
	}

	fmt.Println("[1/6] Criando diretórios...")
	dirs := []string{
		installDir,
		filepath.Join(installDir, "bin"),
		filepath.Join(installDir, "exemplos"),
		filepath.Join(installDir, "docs"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			fail("Erro ao criar diretório %s: %s", d, err)
		}
	}
	fmt.Println("   OK")

	fmt.Println("[2/6] Instalando Flang...")
	extractAll("payload", installDir)
	fmt.Println("   OK")

	fmt.Println("[3/6] Adicionando ao PATH...")
	addToPath(filepath.Join(installDir, "bin"))
	fmt.Println("   OK")

	fmt.Println("[4/6] Associando arquivos .fg...")
	associateExtension()
	fmt.Println("   OK")

	fmt.Println("[5/6] Criando atalhos...")
	createShortcuts()
	fmt.Println("   OK")

	fmt.Println("[6/6] Verificando instalação...")
	out, err := exec.Command(filepath.Join(installDir, "bin", "flang.exe"), "version").CombinedOutput()
	if err != nil {
		fmt.Println("   AVISO: não foi possível verificar (reinicie o terminal)")
	} else {
		fmt.Print("   ", string(out))
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println("  Flang instalado com sucesso!")
	fmt.Println()
	fmt.Printf("  Localização: %s\n", installDir)
	fmt.Println()
	fmt.Println("  Reinicie o terminal e digite:")
	fmt.Println()
	fmt.Println("    flang version")
	fmt.Println("    flang new meu-projeto")
	fmt.Println("    flang run inicio.fg")
	fmt.Println()
	fmt.Println("  Para desinstalar:")
	fmt.Printf("    %s\\uninstall.exe\n", installDir)
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Println()
	fmt.Print("Pressione ENTER para sair...")
	fmt.Scanln()
}

func banner() {
	fmt.Println()
	fmt.Println("  ███████╗██╗      █████╗ ███╗   ██╗ ██████╗")
	fmt.Println("  ██╔════╝██║     ██╔══██╗████╗  ██║██╔════╝")
	fmt.Println("  █████╗  ██║     ███████║██╔██╗ ██║██║  ███╗")
	fmt.Println("  ██╔══╝  ██║     ██╔══██╗██║╚██╗██║██║   ██║")
	fmt.Println("  ██║     ███████╗██║  ██║██║ ╚████║╚██████╔╝")
	fmt.Println("  ╚═╝     ╚══════╝╚═╝  ╚═╝╚═╝  ╚═══╝ ╚═════╝")
	fmt.Printf("  Instalador v%s\n", version)
	fmt.Println()
	fmt.Printf("  Instalar em: %s\n", installDir)
	fmt.Println()
	fmt.Print("  Pressione ENTER para instalar ou CTRL+C para cancelar...")
	fmt.Scanln()
	fmt.Println()
}

func extractAll(root, dest string) {
	fs.WalkDir(payload, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath := strings.TrimPrefix(path, root+"/")
		if relPath == root || relPath == "" {
			return nil
		}
		destPath := filepath.Join(dest, filepath.FromSlash(relPath))

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		data, err := payload.ReadFile(path)
		if err != nil {
			return err
		}

		os.MkdirAll(filepath.Dir(destPath), 0755)
		return os.WriteFile(destPath, data, 0755)
	})
}

// addToPath adds a directory to the user's PATH permanently via registry.
func addToPath(dir string) {
	// Read current PATH from registry
	k, err := openRegKey(`Environment`)
	if err != nil {
		fmt.Printf("   AVISO: não foi possível abrir registro: %s\n", err)
		return
	}
	defer syscall.RegCloseKey(k)

	currentPath, _ := readRegString(k, "Path")

	// Check if already in PATH
	parts := strings.Split(currentPath, ";")
	for _, p := range parts {
		if strings.EqualFold(strings.TrimSpace(p), dir) {
			return // Already in PATH
		}
	}

	// Append
	newPath := currentPath
	if newPath != "" && !strings.HasSuffix(newPath, ";") {
		newPath += ";"
	}
	newPath += dir

	writeRegString(k, "Path", newPath)

	// Notify Windows that environment changed
	broadcastEnvChange()
}

// associateExtension creates .fg file association in the registry.
func associateExtension() {
	flangExe := filepath.Join(installDir, "bin", "flang.exe")

	// Create .fg extension key
	k1, err := createRegKey(`Software\Classes\.fg`)
	if err != nil {
		return
	}
	writeRegString(k1, "", "FlangFile")
	syscall.RegCloseKey(k1)

	// Create FlangFile key
	k2, err := createRegKey(`Software\Classes\FlangFile`)
	if err != nil {
		return
	}
	writeRegString(k2, "", "Arquivo Flang (.fg)")
	syscall.RegCloseKey(k2)

	// Create open command
	k3, err := createRegKey(`Software\Classes\FlangFile\shell\open\command`)
	if err != nil {
		return
	}
	writeRegString(k3, "", fmt.Sprintf(`"%s" run "%%1"`, flangExe))
	syscall.RegCloseKey(k3)
}

func createShortcuts() {
	// Create start menu folder
	startMenu := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Flang")
	os.MkdirAll(startMenu, 0755)

	// Create a simple .cmd shortcut to open terminal in examples
	cmdContent := fmt.Sprintf("@echo off\ncd /d \"%s\\exemplos\"\ncmd\n", installDir)
	os.WriteFile(filepath.Join(startMenu, "Flang Terminal.cmd"), []byte(cmdContent), 0755)

	// Create uninstaller shortcut
	uninstContent := fmt.Sprintf("@echo off\n\"%s\\uninstall.exe\"\n", installDir)
	os.WriteFile(filepath.Join(startMenu, "Desinstalar Flang.cmd"), []byte(uninstContent), 0755)
}

func uninstall() {
	fmt.Println()
	fmt.Println("  Desinstalar Flang?")
	fmt.Println()
	fmt.Print("  Digite S para confirmar: ")
	var input string
	fmt.Scanln(&input)
	if strings.ToUpper(strings.TrimSpace(input)) != "S" {
		fmt.Println("  Cancelado.")
		return
	}

	fmt.Println()
	fmt.Println("[1/4] Removendo do PATH...")
	removeFromPath(filepath.Join(installDir, "bin"))
	fmt.Println("   OK")

	fmt.Println("[2/4] Removendo associação .fg...")
	deleteRegKey(`Software\Classes\.fg`)
	deleteRegKey(`Software\Classes\FlangFile`)
	fmt.Println("   OK")

	fmt.Println("[3/4] Removendo atalhos...")
	startMenu := filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs", "Flang")
	os.RemoveAll(startMenu)
	fmt.Println("   OK")

	fmt.Println("[4/4] Removendo arquivos...")
	// Can't delete ourselves while running, so create a delayed delete
	bat := filepath.Join(os.TempDir(), "flang-uninstall.bat")
	batContent := fmt.Sprintf("@echo off\nping -n 2 127.0.0.1 > nul\nrd /s /q \"%s\"\necho Flang desinstalado!\ndel \"%%~f0\"\n", installDir)
	os.WriteFile(bat, []byte(batContent), 0755)
	exec.Command("cmd", "/c", "start", "/min", bat).Start()
	fmt.Println("   OK")

	fmt.Println()
	fmt.Println("  Flang desinstalado com sucesso!")
	fmt.Println()
	fmt.Print("Pressione ENTER para sair...")
	fmt.Scanln()
}

func removeFromPath(dir string) {
	k, err := openRegKey(`Environment`)
	if err != nil {
		return
	}
	defer syscall.RegCloseKey(k)

	currentPath, _ := readRegString(k, "Path")
	parts := strings.Split(currentPath, ";")
	var newParts []string
	for _, p := range parts {
		if !strings.EqualFold(strings.TrimSpace(p), dir) && strings.TrimSpace(p) != "" {
			newParts = append(newParts, p)
		}
	}
	writeRegString(k, "Path", strings.Join(newParts, ";"))
	broadcastEnvChange()
}

// ==================== Windows Registry Helpers ====================

var (
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")

	procRegOpenKeyExW    = advapi32.NewProc("RegOpenKeyExW")
	procRegCreateKeyExW  = advapi32.NewProc("RegCreateKeyExW")
	procRegSetValueExW   = advapi32.NewProc("RegSetValueExW")
	procRegQueryValueExW = advapi32.NewProc("RegQueryValueExW")
	procRegDeleteKeyW    = advapi32.NewProc("RegDeleteKeyW")
	procSendMessageTimeoutW = user32.NewProc("SendMessageTimeoutW")
)

const (
	HKEY_CURRENT_USER = 0x80000001
	KEY_ALL_ACCESS    = 0xF003F
	REG_SZ            = 1
	REG_EXPAND_SZ     = 2
	HWND_BROADCAST    = 0xFFFF
	WM_SETTINGCHANGE  = 0x001A
	SMTO_ABORTIFHUNG  = 0x0002
)

func openRegKey(subkey string) (syscall.Handle, error) {
	var handle syscall.Handle
	sub, _ := syscall.UTF16PtrFromString(subkey)
	ret, _, _ := procRegOpenKeyExW.Call(
		uintptr(HKEY_CURRENT_USER), uintptr(unsafe.Pointer(sub)),
		0, uintptr(KEY_ALL_ACCESS), uintptr(unsafe.Pointer(&handle)),
	)
	if ret != 0 {
		return 0, fmt.Errorf("RegOpenKeyEx failed: %d", ret)
	}
	return handle, nil
}

func createRegKey(subkey string) (syscall.Handle, error) {
	var handle syscall.Handle
	var disposition uint32
	sub, _ := syscall.UTF16PtrFromString(subkey)
	ret, _, _ := procRegCreateKeyExW.Call(
		uintptr(HKEY_CURRENT_USER), uintptr(unsafe.Pointer(sub)),
		0, 0, 0, uintptr(KEY_ALL_ACCESS), 0,
		uintptr(unsafe.Pointer(&handle)), uintptr(unsafe.Pointer(&disposition)),
	)
	if ret != 0 {
		return 0, fmt.Errorf("RegCreateKeyEx failed: %d", ret)
	}
	return handle, nil
}

func readRegString(key syscall.Handle, name string) (string, error) {
	var typ uint32
	var size uint32

	namePtr, _ := syscall.UTF16PtrFromString(name)

	// Get size first
	procRegQueryValueExW.Call(
		uintptr(key), uintptr(unsafe.Pointer(namePtr)),
		0, uintptr(unsafe.Pointer(&typ)), 0, uintptr(unsafe.Pointer(&size)),
	)

	if size == 0 {
		return "", nil
	}

	buf := make([]uint16, size/2+1)
	ret, _, _ := procRegQueryValueExW.Call(
		uintptr(key), uintptr(unsafe.Pointer(namePtr)),
		0, uintptr(unsafe.Pointer(&typ)),
		uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)),
	)
	if ret != 0 {
		return "", fmt.Errorf("RegQueryValueEx failed: %d", ret)
	}

	return syscall.UTF16ToString(buf), nil
}

func writeRegString(key syscall.Handle, name, value string) {
	namePtr, _ := syscall.UTF16PtrFromString(name)
	valUTF16, _ := syscall.UTF16FromString(value)
	size := uint32(len(valUTF16) * 2)

	procRegSetValueExW.Call(
		uintptr(key), uintptr(unsafe.Pointer(namePtr)),
		0, uintptr(REG_EXPAND_SZ),
		uintptr(unsafe.Pointer(&valUTF16[0])), uintptr(size),
	)
}

func deleteRegKey(subkey string) {
	sub, _ := syscall.UTF16PtrFromString(subkey)
	procRegDeleteKeyW.Call(uintptr(HKEY_CURRENT_USER), uintptr(unsafe.Pointer(sub)))
}

func broadcastEnvChange() {
	env, _ := syscall.UTF16PtrFromString("Environment")
	var result uintptr
	procSendMessageTimeoutW.Call(
		uintptr(HWND_BROADCAST), uintptr(WM_SETTINGCHANGE),
		0, uintptr(unsafe.Pointer(env)),
		uintptr(SMTO_ABORTIFHUNG), 5000, uintptr(unsafe.Pointer(&result)),
	)
}

func fail(format string, args ...interface{}) {
	fmt.Printf("\n  ERRO: "+format+"\n", args...)
	fmt.Print("\nPressione ENTER para sair...")
	fmt.Scanln()
	os.Exit(1)
}
