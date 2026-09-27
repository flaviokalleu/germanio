package intelligence

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IdempotencyError indicates an operation could not be idempotent-enforced.
type IdempotencyError struct {
	Operation  string
	EntityName string
	Reason     string
}

func (e IdempotencyError) Error() string {
	return fmt.Sprintf("Falha de idempotência em %s (%s): %s", e.Operation, e.EntityName, e.Reason)
}

// OverwriteProtectionError indicates a file is protected from overwrite.
type OverwriteProtectionError struct {
	FilePath string
	Status   string
	Reason   string
}

func (e OverwriteProtectionError) Error() string {
	return fmt.Sprintf("Arquivo protegido contra sobrescrita: %s (Proprietário: %s) — %s", e.FilePath, e.Status, e.Reason)
}

// ValidateDependencies checks for circular dependencies in capabilities.
func ValidateDependencies(caps map[string]*CapabilityNode) []ProjectDiagnostic {
	var diags []ProjectDiagnostic
	visited := make(map[string]bool)

	for capID := range caps {
		if hasCyclicDependency(capID, caps, visited, make(map[string]bool)) {
			diags = append(diags, ProjectDiagnostic{
				Code:        "VAL301",
				Severity:    SeverityError,
				Title:       fmt.Sprintf("Dependência circular detectada: %s", capID),
				Description: fmt.Sprintf("A capability '%s' tem uma referência circular em suas dependências. Capabilities não podem depender uma da outra de forma circular.", capID),
				Location:    capID,
				Fix:         "Remova ou reorganize as dependências para eliminar o ciclo.",
			})
		}
	}

	return diags
}

func hasCyclicDependency(capID string, caps map[string]*CapabilityNode, visited, recStack map[string]bool) bool {
	visited[capID] = true
	recStack[capID] = true

	cap, exists := caps[capID]
	if !exists {
		return false
	}

	for _, dep := range cap.Dependencies {
		if !visited[dep] {
			if hasCyclicDependency(dep, caps, visited, recStack) {
				return true
			}
		} else if recStack[dep] {
			return true
		}
	}

	delete(recStack, capID)
	return false
}

// ValidateAPIs checks for invalid API definitions.
func ValidateAPIs(apis map[string]*APINode, entities map[string]*EntityNode) []ProjectDiagnostic {
	var diags []ProjectDiagnostic

	validMethods := map[string]bool{"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true, "HEAD": true, "OPTIONS": true}

	for key, api := range apis {
		// Check valid HTTP method
		if !validMethods[api.Method] {
			diags = append(diags, ProjectDiagnostic{
				Code:        "VAL401",
				Severity:    SeverityError,
				Title:       fmt.Sprintf("Método HTTP inválido: %s", api.Method),
				Description: fmt.Sprintf("A rota '%s' usa o método '%s' que não é suportado. Use GET, POST, PUT, DELETE, PATCH, HEAD ou OPTIONS.", key, api.Method),
				Location:    key,
				Fix:         fmt.Sprintf("Corrija o método para um dos válidos."),
			})
		}

		// Check if entity reference exists
		if api.EntityRef != "" && entities[strings.ToLower(api.EntityRef)] == nil {
			diags = append(diags, ProjectDiagnostic{
				Code:        "VAL402",
				Severity:    SeverityWarning,
				Title:       fmt.Sprintf("Referência de entidade não encontrada: %s", api.EntityRef),
				Description: fmt.Sprintf("A rota '%s' referencia a entidade '%s' que não existe no projeto.", key, api.EntityRef),
				Location:    key,
				Fix:         fmt.Sprintf("Crie a entidade '%s' ou corrija a referência.", api.EntityRef),
			})
		}

		// Check path format
		if !strings.HasPrefix(api.Path, "/") {
			diags = append(diags, ProjectDiagnostic{
				Code:        "VAL403",
				Severity:    SeverityError,
				Title:       fmt.Sprintf("Path de rota sem prefixo '/': %s", api.Path),
				Description: fmt.Sprintf("O path '%s' deve começar com '/'.", api.Path),
				Location:    key,
				Fix:         fmt.Sprintf("Adicione '/' ao início do path."),
			})
		}
	}

	return diags
}

// ValidateRoutes checks for missing API route definitions.
func ValidateRoutes(pages map[string]*PageNode, apis map[string]*APINode) []ProjectDiagnostic {
	var diags []ProjectDiagnostic

	for _, page := range pages {
		for _, comp := range page.Components {
			for _, action := range comp.Actions {
				// Check if the action target exists in APIs
				if strings.HasPrefix(action.TargetAPI, "/api/") {
					apiKey := fmt.Sprintf("POST %s", action.TargetAPI) // Most common case
					found := false
					_ = apiKey
					for key := range apis {
						if strings.Contains(key, action.TargetAPI) {
							found = true
							break
						}
					}

					if !found && !action.IsZombie {
						diags = append(diags, ProjectDiagnostic{
							Code:        "VAL501",
							Severity:    SeverityWarning,
							Title:       fmt.Sprintf("Rota de API não definida: %s", action.TargetAPI),
							Description: fmt.Sprintf("O componente '%s' na página '%s' referencia a rota '%s' que não está definida.", comp.Name, page.Path, action.TargetAPI),
							Location:    page.FilePath,
							Fix:         fmt.Sprintf("Crie a rota com: ge init api %s", strings.TrimPrefix(action.TargetAPI, "/api/")),
						})
					}
				}
			}
		}
	}

	return diags
}

// ValidateComponents checks for invalid or orphaned components.
func ValidateComponents(components map[string]*ComponentNode, pages map[string]*PageNode, entities map[string]*EntityNode) []ProjectDiagnostic {
	var diags []ProjectDiagnostic

	validKinds := map[string]bool{"tabela": true, "formulario": true, "cards": true, "kanban": true, "filtros": true, "modal": true, "stats": true, "galeria": true, "calendario": true}

	for compName, comp := range components {
		// Check valid component kind
		if !validKinds[comp.Kind] {
			diags = append(diags, ProjectDiagnostic{
				Code:        "VAL601",
				Severity:    SeverityWarning,
				Title:       fmt.Sprintf("Tipo de componente inválido: %s", comp.Kind),
				Description: fmt.Sprintf("O componente '%s' usa o tipo '%s' que não é reconhecido.", compName, comp.Kind),
				Location:    comp.FilePath,
				Fix:         fmt.Sprintf("Use um tipo válido: tabela, formulario, cards, kanban, filtros, modal, stats, galeria ou calendario."),
			})
		}

		// Check if entity reference exists
		if comp.EntityRef != "" && entities[strings.ToLower(comp.EntityRef)] == nil {
			diags = append(diags, ProjectDiagnostic{
				Code:        "VAL602",
				Severity:    SeverityWarning,
				Title:       fmt.Sprintf("Componente com entidade não encontrada: %s", comp.EntityRef),
				Description: fmt.Sprintf("O componente '%s' referencia a entidade '%s' que não existe.", compName, comp.EntityRef),
				Location:    comp.FilePath,
				Fix:         fmt.Sprintf("Crie a entidade '%s' ou remova a referência.", comp.EntityRef),
			})
		}

		// Check if component is used in any page
		componentUsed := false
		for _, page := range pages {
			for _, pageComp := range page.Components {
				if pageComp.Name == compName {
					componentUsed = true
					break
				}
			}
		}

		if !componentUsed && !comp.Ejected {
			diags = append(diags, ProjectDiagnostic{
				Code:        "VAL603",
				Severity:    SeverityInfo,
				Title:       fmt.Sprintf("Componente órfão: %s", compName),
				Description: fmt.Sprintf("O componente '%s' não está sendo usado em nenhuma página.", compName),
				Location:    comp.FilePath,
				Fix:         fmt.Sprintf("Adicione o componente a uma página ou remova-o se não for necessário."),
			})
		}
	}

	return diags
}

// ValidateMigrations checks for missing or invalid database migrations.
func ValidateMigrations(entities map[string]*EntityNode, rootDir string) []ProjectDiagnostic {
	var diags []ProjectDiagnostic

	migrationsDir := filepath.Join(rootDir, "migrações")
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		if len(entities) > 0 {
			diags = append(diags, ProjectDiagnostic{
				Code:        "VAL701",
				Severity:    SeverityWarning,
				Title:       "Diretório de migrações não encontrado",
				Description: fmt.Sprintf("O diretório 'migrações' não existe, mas há entidades definidas no projeto."),
				Location:    migrationsDir,
				Fix:         fmt.Sprintf("Crie o diretório 'migrações' ou gere as migrações com: ge migrate generate"),
			})
		}
	}

	return diags
}

// ValidateRelations checks for invalid entity relationships.
func ValidateRelations(entities map[string]*EntityNode) []ProjectDiagnostic {
	var diags []ProjectDiagnostic

	validRelationTypes := map[string]bool{"1:1": true, "1:N": true, "N:N": true, "N:1": true}

	for _, ent := range entities {
		for _, rel := range ent.Relations {
			// Check valid relation type
			if !validRelationTypes[rel.Type] {
				diags = append(diags, ProjectDiagnostic{
					Code:        "VAL801",
					Severity:    SeverityError,
					Title:       fmt.Sprintf("Tipo de relacionamento inválido: %s", rel.Type),
					Description: fmt.Sprintf("O relacionamento '%s' em '%s' usa o tipo '%s' que não é válido.", rel.Name, ent.Name, rel.Type),
					Location:    ent.FilePath,
					Fix:         fmt.Sprintf("Use um tipo válido: 1:1, 1:N, N:N ou N:1."),
				})
			}

			// Check if target entity exists
			targetEnt := entities[strings.ToLower(rel.Target)]
			if targetEnt == nil {
				diags = append(diags, ProjectDiagnostic{
					Code:        "VAL802",
					Severity:    SeverityError,
					Title:       fmt.Sprintf("Relacionamento referencia entidade inexistente: %s", rel.Target),
					Description: fmt.Sprintf("O relacionamento '%s' em '%s' referencia a entidade '%s' que não existe.", rel.Name, ent.Name, rel.Target),
					Location:    ent.FilePath,
					Fix:         fmt.Sprintf("Crie a entidade '%s' ou remova o relacionamento.", rel.Target),
				})
			}

			// Check for self-references (only valid for specific cases)
			if strings.EqualFold(rel.Source, rel.Target) && rel.Type != "1:N" && rel.Type != "N:1" {
				diags = append(diags, ProjectDiagnostic{
					Code:        "VAL803",
					Severity:    SeverityWarning,
					Title:       fmt.Sprintf("Auto-referência em relacionamento: %s", rel.Name),
					Description: fmt.Sprintf("O relacionamento '%s' em '%s' referencia a si mesma com tipo '%s'.", rel.Name, ent.Name, rel.Type),
					Location:    ent.FilePath,
					Fix:         fmt.Sprintf("Use tipo 1:N ou N:1 para auto-referências, ou revise o modelo."),
				})
			}
		}
	}

	return diags
}

// TransactionManager methods for idempotency and protection.

// EnsureEntityIdempotent checks if entity already exists and avoids re-creation if it's identical.
func (t *TransactionManager) EnsureEntityIdempotent(ent *EntityNode, dryRun bool) error {
	if t.Manifest == nil {
		return nil
	}

	// Check if entity is already in the project
	for _, existingEnt := range t.Manifest.Entities {
		if strings.EqualFold(existingEnt, ent.Name) {
			// Entity already exists, check if it's customized
			dataFile := fmt.Sprintf("dados/%s.ge", strings.ToLower(ent.Plural))
			if t.Manifest.IsProtected(dataFile) {
				return IdempotencyError{
					Operation:  "InitEntity",
					EntityName: ent.Name,
					Reason:     "Entidade já existe e foi customizada pelo desenvolvedor",
				}
			}
			// If not protected, we can update silently (idempotent)
			return nil
		}
	}

	return nil
}

// EnsurePageIdempotent checks if page already exists.
func (t *TransactionManager) EnsurePageIdempotent(pagePath string) error {
	if t.Manifest == nil {
		return nil
	}

	// Check if page file is already registered
	for filePath, meta := range t.Manifest.Files {
		if strings.EqualFold(filePath, pagePath) && meta.Ownership != OwnershipManaged {
			return IdempotencyError{
				Operation:  "InitPage",
				EntityName: pagePath,
				Reason:     "Página já existe com propriedade diferente de 'managed'",
			}
		}
	}

	return nil
}

// EnsureDashboardIdempotent validates dashboard regeneration safety.
func (t *TransactionManager) EnsureDashboardIdempotent() error {
	if t.Manifest == nil {
		return nil
	}

	dashboardPath := "paginas/dashboard.ge"
	if meta, ok := t.Manifest.Files[dashboardPath]; ok {
		if meta.Ownership == OwnershipCustomized {
			return IdempotencyError{
				Operation:  "InitDashboard",
				EntityName: "dashboard",
				Reason:     "Dashboard foi customizado pelo desenvolvedor",
			}
		}
	}

	return nil
}

// EjectWithProtection ensures an ejected component is marked as user-owned.
func (t *TransactionManager) EjectWithProtection(relPath string) error {
	if t.Manifest == nil {
		return nil
	}

	fullPath := filepath.Join(t.RootDir, relPath)

	// Back up the original if it exists
	if data, err := os.ReadFile(fullPath); err == nil {
		t.backups[fullPath] = data
	}

	// Mark as ejected in manifest and persist immediately: eject is a standalone
	// ownership operation, not a queued file write.
	t.Manifest.RegisterFile(relPath, OwnershipEjected, fmt.Sprintf("Componente ejetado: %s", relPath))
	return t.Manifest.Save(t.RootDir)
}

// ValidateFileOverwriteSafety checks if a file can be safely modified.
func ValidateFileOverwriteSafety(manifest *ProjectManifest, relPath string) error {
	if manifest == nil {
		return nil
	}

	if manifest.IsProtected(relPath) {
		meta := manifest.Files[relPath]
		return OverwriteProtectionError{
			FilePath: relPath,
			Status:   string(meta.Ownership),
			Reason:   meta.Description,
		}
	}

	return nil
}

// RenderDiffPlan creates a detailed plan before large changes.
func (t *TransactionManager) RenderDiffPlan() string {
	var b strings.Builder

	if len(t.Ops) == 0 {
		return "Nenhuma operação planejada."
	}

	b.WriteString("═════════════════════════════════════════\n")
	b.WriteString("📋 PLANO DE ALTERAÇÕES DO PROJETO\n")
	b.WriteString("═════════════════════════════════════════\n\n")

	createCount := 0
	modifyCount := 0
	deleteCount := 0

	for _, op := range t.Ops {
		switch op.Action {
		case OpCreate:
			createCount++
		case OpModify:
			modifyCount++
		case OpDelete:
			deleteCount++
		}
	}

	b.WriteString(fmt.Sprintf("📊 Resumo: +%d novos, ~%d modificados, -%d removidos\n\n", createCount, modifyCount, deleteCount))

	// Group by action
	b.WriteString("✨ NOVOS ARQUIVOS:\n")
	for _, op := range t.Ops {
		if op.Action == OpCreate {
			b.WriteString(fmt.Sprintf("  ✓ %s — %s\n", op.Path, op.Description))
		}
	}

	if modifyCount > 0 {
		b.WriteString("\n🔄 MODIFICAÇÕES:\n")
		for _, op := range t.Ops {
			if op.Action == OpModify {
				oldLen := len(op.OldContent)
				newLen := len(op.NewContent)
				delta := newLen - oldLen
				sign := "+"
				if delta < 0 {
					sign = ""
				}
				b.WriteString(fmt.Sprintf("  ~ %s — %s bytes (%s%d)\n", op.Path, op.Description, sign, delta))
			}
		}
	}

	if deleteCount > 0 {
		b.WriteString("\n🗑️  REMOVIÇÕES:\n")
		for _, op := range t.Ops {
			if op.Action == OpDelete {
				b.WriteString(fmt.Sprintf("  ✗ %s\n", op.Path))
			}
		}
	}

	b.WriteString("\n═════════════════════════════════════════\n")
	b.WriteString(fmt.Sprintf("⚠️  DRY RUN: %v\n", t.DryRun))
	b.WriteString("═════════════════════════════════════════\n")

	return b.String()
}

// RollbackTransactional performs atomic rollback of all operations.
func (t *TransactionManager) RollbackTransactional() error {
	var errors []string

	// Restore all backed-up files
	for filePath, backupData := range t.backups {
		if err := os.WriteFile(filePath, backupData, 0644); err != nil {
			errors = append(errors, fmt.Sprintf("Erro ao restaurar %s: %v", filePath, err))
		}
	}

	// Remove any newly created files
	for _, op := range t.Ops {
		if op.Action == OpCreate {
			fullPath := filepath.Join(t.RootDir, op.Path)
			if err := os.Remove(fullPath); err != nil {
				errors = append(errors, fmt.Sprintf("Erro ao remover %s: %v", fullPath, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("Falhas no rollback:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}
