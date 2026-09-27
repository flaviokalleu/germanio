package intelligence

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileOpAction represents filesystem change type.
type FileOpAction string

const (
	OpCreate FileOpAction = "CREATE"
	OpModify FileOpAction = "MODIFY"
	OpDelete FileOpAction = "DELETE"
)

// FileOperation describes a single file mutation.
type FileOperation struct {
	Path        string
	Action      FileOpAction
	OldContent  string
	NewContent  string
	Description string
}

// TransactionManager executes file operations with diffing, dry-run, and automatic rollback on error.
type TransactionManager struct {
	RootDir  string
	DryRun   bool
	Manifest *ProjectManifest
	Ops      []FileOperation
	backups  map[string][]byte
}

// NewTransactionManager creates a transaction manager.
func NewTransactionManager(rootDir string, dryRun bool, manifest *ProjectManifest) *TransactionManager {
	return &TransactionManager{
		RootDir:  rootDir,
		DryRun:   dryRun,
		Manifest: manifest,
		backups:  make(map[string][]byte),
	}
}

func validateRelativeProjectPath(relPath string) error {
	clean := filepath.Clean(relPath)
	if relPath == "" || filepath.IsAbs(relPath) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("caminho fora do projeto não é permitido: %q", relPath)
	}
	return nil
}

// AddFile registers a file to be created or overwritten.
func (t *TransactionManager) AddFile(relPath, content, description string) error {
	if err := validateRelativeProjectPath(relPath); err != nil {
		return err
	}
	fullPath := filepath.Join(t.RootDir, relPath)

	// Check protection through the shared typed validator so CLI callers can
	// distinguish a user-owned/customized file from ordinary I/O failures.
	if err := ValidateFileOverwriteSafety(t.Manifest, relPath); err != nil {
		return err
	}

	action := OpCreate
	var oldContent string
	if existingData, err := os.ReadFile(fullPath); err == nil {
		action = OpModify
		oldContent = string(existingData)
	}

	t.Ops = append(t.Ops, FileOperation{
		Path:        relPath,
		Action:      action,
		OldContent:  oldContent,
		NewContent:  content,
		Description: description,
	})

	return nil
}

// RenderDiffSummary formats a concise summary of changes.
func (t *TransactionManager) RenderDiffSummary() string {
	var b strings.Builder
	b.WriteString("Resumo das Alterações no Projeto:\n")

	for _, op := range t.Ops {
		switch op.Action {
		case OpCreate:
			b.WriteString(fmt.Sprintf("  + Criar:  %s (%s)\n", op.Path, op.Description))
		case OpModify:
			b.WriteString(fmt.Sprintf("  ~ Editar: %s (%s)\n", op.Path, op.Description))
		case OpDelete:
			b.WriteString(fmt.Sprintf("  - Excluir:%s\n", op.Path))
		}
	}

	return b.String()
}

// Commit executes all operations atomically.
func (t *TransactionManager) Commit() error {
	if t.DryRun {
		return nil
	}

	// 1. Create backups of existing files
	for _, op := range t.Ops {
		fullPath := filepath.Join(t.RootDir, op.Path)
		if data, err := os.ReadFile(fullPath); err == nil {
			t.backups[fullPath] = data
		}
	}

	// 2. Perform writes
	for _, op := range t.Ops {
		fullPath := filepath.Join(t.RootDir, op.Path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Rollback()
			return fmt.Errorf("Falha ao criar diretório %s: %w", dir, err)
		}

		if err := os.WriteFile(fullPath, []byte(op.NewContent), 0644); err != nil {
			t.Rollback()
			return fmt.Errorf("Falha ao escrever arquivo %s: %w", fullPath, err)
		}

		if t.Manifest != nil {
			t.Manifest.RegisterFile(op.Path, OwnershipManaged, op.Description)
		}
	}

	// 3. Save updated manifest
	if t.Manifest != nil {
		if err := t.Manifest.Save(t.RootDir); err != nil {
			t.Rollback()
			return fmt.Errorf("Falha ao salvar manifesto %s: %w", ManifestFileName, err)
		}
	}

	return nil
}

// Rollback restores backed-up files and cleans up created files on failure.
func (t *TransactionManager) Rollback() {
	for _, op := range t.Ops {
		fullPath := filepath.Join(t.RootDir, op.Path)
		if backupData, exists := t.backups[fullPath]; exists {
			_ = os.WriteFile(fullPath, backupData, 0644)
		} else {
			// File was created in this transaction, remove it
			_ = os.Remove(fullPath)
		}
	}
}
