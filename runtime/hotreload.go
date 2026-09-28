package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// hotReloadOn: reloading on file changes is a development convenience. A
// production server (GERMANIO_PRODUCAO=1, set by the Docker image and by
// binaries made with germanio build) does not watch its files.
func hotReloadOn() bool {
	return os.Getenv("GERMANIO_PRODUCAO") != "1"
}

// skipDir: folders that hold data, never .ge sources worth watching.
func skipDir(name string) bool {
	switch name {
	case ".git", "node_modules", "repositorios", "uploads", "dist", "assets":
		return true
	}
	return false
}

// scanChanged updates stamps with the .ge files under dir and reports whether
// any of them is new or changed.
func scanChanged(dir string, stamps map[string]time.Time) bool {
	changed := false
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != dir && skipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".ge" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		mod := info.ModTime()
		if prev, ok := stamps[path]; !ok || mod.After(prev) {
			changed = true
			fmt.Printf("[germanio] Arquivo modificado: %s\n", filepath.Base(path))
		}
		stamps[path] = mod
		return nil
	})
	return changed
}

// reloadable checks the program before the server is replaced: a program
// that does not compile never replaces a running one (and never migrates
// the database).
func reloadable(arquivo string) error {
	_, err := Compilar(arquivo)
	return err
}

// WatchFiles monitors .ge files for changes and restarts the process once the
// new program compiles.
func WatchFiles(dir string, arquivo string, porta string) {
	if !hotReloadOn() {
		return
	}
	stamps := make(map[string]time.Time)
	scanChanged(dir, stamps)

	go func() {
		for {
			time.Sleep(1 * time.Second)
			if !scanChanged(dir, stamps) {
				continue
			}
			if err := reloadable(arquivo); err != nil {
				fmt.Printf("[germanio] O programa novo tem erro; o servidor continua com a versão anterior:\n%s\n", err)
				continue
			}
			fmt.Println("[germanio] Recarregando...")
			cmd := exec.Command(os.Args[0], "run", arquivo, porta)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			if err := cmd.Start(); err != nil {
				fmt.Printf("[germanio] Erro ao recarregar: %s\n", err)
				continue
			}
			os.Exit(0)
		}
	}()
}
