package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// WatchFiles monitors .ge files for changes and restarts the process.
func WatchFiles(dir string, arquivo string, porta string) {
	stamps := make(map[string]time.Time)
	scanFG(dir, stamps)

	go func() {
		for {
			time.Sleep(1 * time.Second)

			changed := false
			filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}
				if filepath.Ext(path) != ".ge" {
					return nil
				}
				mod := info.ModTime()
				if prev, ok := stamps[path]; ok {
					if mod.After(prev) {
						fmt.Printf("[germanio] Arquivo modificado: %s\n", filepath.Base(path))
						changed = true
					}
				} else {
					changed = true
				}
				stamps[path] = mod
				return nil
			})

			if changed {
				fmt.Println("[germanio] Recarregando...")
				// Re-exec the process
				cmd := exec.Command(os.Args[0], "run", arquivo, porta)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Stdin = os.Stdin
				if err := cmd.Start(); err != nil {
					fmt.Printf("[germanio] Erro ao recarregar: %s\n", err)
					continue
				}
				// Exit the current process
				os.Exit(0)
			}
		}
	}()
}

func scanFG(dir string, stamps map[string]time.Time) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".ge" {
			stamps[path] = info.ModTime()
		}
		return nil
	})
}
