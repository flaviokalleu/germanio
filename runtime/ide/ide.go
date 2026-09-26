package ide

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type IDE struct {
	Dir    string
	Porta  string
	AppCmd *exec.Cmd
}

func Novo(dir, porta string) *IDE {
	absDir, _ := filepath.Abs(dir)
	return &IDE{Dir: absDir, Porta: porta}
}

func (ide *IDE) Iniciar() error {
	mux := http.NewServeMux()

	// Serve the IDE HTML
	mux.HandleFunc("/", ide.handleIndex)

	// File API
	mux.HandleFunc("/api/files", ide.handleFiles)
	mux.HandleFunc("/api/file", ide.handleFile)
	mux.HandleFunc("/api/file/save", ide.handleSave)
	mux.HandleFunc("/api/file/create", ide.handleCreate)
	mux.HandleFunc("/api/file/delete", ide.handleDelete)

	// Run/Check API
	mux.HandleFunc("/api/run", ide.handleRun)
	mux.HandleFunc("/api/stop", ide.handleStop)
	mux.HandleFunc("/api/check", ide.handleCheck)

	fmt.Printf("\n")
	fmt.Printf("  ╔══════════════════════════════════════════╗\n")
	fmt.Printf("  ║         Flang IDE v0.6.0                 ║\n")
	fmt.Printf("  ║  http://localhost:%-23s║\n", ide.Porta)
	fmt.Printf("  ╚══════════════════════════════════════════╝\n")
	fmt.Printf("\n")
	fmt.Printf("  Diretorio: %s\n", ide.Dir)
	fmt.Printf("  Ctrl+C para sair\n\n")

	return http.ListenAndServe(":"+ide.Porta, mux)
}

func (ide *IDE) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(ideHTML))
}

func (ide *IDE) handleFiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	type FileInfo struct {
		Name     string     `json:"name"`
		Path     string     `json:"path"`
		IsDir    bool       `json:"isDir"`
		Children []FileInfo `json:"children,omitempty"`
	}

	var walkDir func(dir string, prefix string) []FileInfo
	walkDir = func(dir string, prefix string) []FileInfo {
		var files []FileInfo
		entries, err := os.ReadDir(dir)
		if err != nil {
			return files
		}
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || strings.HasSuffix(name, ".db") || strings.HasSuffix(name, ".db-shm") || strings.HasSuffix(name, ".db-wal") || strings.HasSuffix(name, ".exe") {
				continue
			}
			rel := prefix + name
			fi := FileInfo{Name: name, Path: rel, IsDir: e.IsDir()}
			if e.IsDir() {
				fi.Children = walkDir(filepath.Join(dir, name), rel+"/")
			}
			files = append(files, fi)
		}
		return files
	}

	tree := walkDir(ide.Dir, "")
	json.NewEncoder(w).Encode(tree)
}

func (ide *IDE) handleFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path required", 400)
		return
	}
	fullPath := filepath.Join(ide.Dir, filepath.Clean(path))
	// Security: ensure within dir
	if !strings.HasPrefix(fullPath, ide.Dir) {
		http.Error(w, "forbidden", 403)
		return
	}
	data, err := os.ReadFile(fullPath)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(data)
}

func (ide *IDE) handleSave(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", 400)
		return
	}
	fullPath := filepath.Join(ide.Dir, filepath.Clean(req.Path))
	if !strings.HasPrefix(fullPath, ide.Dir) {
		http.Error(w, "forbidden", 403)
		return
	}
	os.MkdirAll(filepath.Dir(fullPath), 0755)
	if err := os.WriteFile(fullPath, []byte(req.Content), 0644); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (ide *IDE) handleCreate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	var req struct {
		Path  string `json:"path"`
		IsDir bool   `json:"isDir"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	fullPath := filepath.Join(ide.Dir, filepath.Clean(req.Path))
	if !strings.HasPrefix(fullPath, ide.Dir) {
		http.Error(w, "forbidden", 403)
		return
	}
	if req.IsDir {
		os.MkdirAll(fullPath, 0755)
	} else {
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(""), 0644)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (ide *IDE) handleDelete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	fullPath := filepath.Join(ide.Dir, filepath.Clean(req.Path))
	if !strings.HasPrefix(fullPath, ide.Dir) {
		http.Error(w, "forbidden", 403)
		return
	}
	os.RemoveAll(fullPath)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (ide *IDE) handleRun(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	// Stop existing app
	if ide.AppCmd != nil && ide.AppCmd.Process != nil {
		ide.AppCmd.Process.Kill()
		ide.AppCmd = nil
	}

	var req struct {
		File string `json:"file"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.File == "" {
		req.File = "inicio.ge"
	}

	exe, _ := os.Executable()
	cmd := exec.Command(exe, "run", filepath.Join(ide.Dir, req.File), "8080")
	cmd.Dir = ide.Dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": err.Error()})
		return
	}
	ide.AppCmd = cmd

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "running", "url": "http://localhost:8080"})
}

func (ide *IDE) handleStop(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if ide.AppCmd != nil && ide.AppCmd.Process != nil {
		ide.AppCmd.Process.Kill()
		ide.AppCmd = nil
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
}

func (ide *IDE) handleCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	var req struct {
		File string `json:"file"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.File == "" {
		req.File = "inicio.ge"
	}

	exe, _ := os.Executable()
	cmd := exec.Command(exe, "check", filepath.Join(ide.Dir, req.File))
	output, err := cmd.CombinedOutput()

	status := "ok"
	if err != nil {
		status = "error"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": status, "output": string(output)})
}
