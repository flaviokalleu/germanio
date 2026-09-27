package intelligence

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DiagnosticSeverity indicates issue severity level.
type DiagnosticSeverity string

const (
	SeverityError   DiagnosticSeverity = "ERRO"
	SeverityWarning DiagnosticSeverity = "AVISO"
	SeverityInfo    DiagnosticSeverity = "INFO"
)

// ProjectDiagnostic represents a validated issue in the project intelligence graph.
type ProjectDiagnostic struct {
	Code        string             `json:"code"`
	Severity    DiagnosticSeverity `json:"severity"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Location    string             `json:"location,omitempty"`
	Fix         string             `json:"fix"`
}

// ProjectValidator performs semantic and architectural validation on a project.
type ProjectValidator struct {
	RootDir string
	Graph   *ProjectKnowledgeGraph
}

// NewProjectValidator creates a validator.
func NewProjectValidator(rootDir string, graph *ProjectKnowledgeGraph) *ProjectValidator {
	return &ProjectValidator{RootDir: rootDir, Graph: graph}
}

// ValidateAll runs all integrity, security and consistency checks.
func (v *ProjectValidator) ValidateAll() []ProjectDiagnostic {
	var diags []ProjectDiagnostic

	// 1. Check for Zombie Buttons
	diags = append(diags, v.checkZombieButtons()...)

	// 2. Check for Broken Navigation Links
	diags = append(diags, v.checkBrokenLinks()...)

	// 3. Check for Orphan Entities
	diags = append(diags, v.checkOrphanEntities()...)

	// 4. Check for Hardcoded Secrets
	diags = append(diags, v.checkHardcodedSecrets()...)

	// 5. Check for Missing RBAC Permissions
	diags = append(diags, v.checkMissingPermissions()...)

	return diags
}

func (v *ProjectValidator) checkZombieButtons() []ProjectDiagnostic {
	var diags []ProjectDiagnostic

	for _, page := range v.Graph.Pages {
		for _, comp := range page.Components {
			for _, act := range comp.Actions {
				if act.IsZombie || act.TargetAPI == "" || act.TargetAPI == "#" {
					diags = append(diags, ProjectDiagnostic{
						Code:        "VAL101",
						Severity:    SeverityError,
						Title:       fmt.Sprintf("Botão Zumbi detectado: '%s'", act.Label),
						Description: fmt.Sprintf("O botão '%s' na página '%s' foi criado sem ação ou rota de backend associada.", act.Label, page.Path),
						Location:    page.FilePath,
						Fix:         fmt.Sprintf("Conecte o botão a uma rota válida (exemplo: '/api/salvar') ou remova o botão não utilizado."),
					})
				}
			}
		}
	}

	return diags
}

func (v *ProjectValidator) checkBrokenLinks() []ProjectDiagnostic {
	var diags []ProjectDiagnostic
	pagePaths := make(map[string]bool)
	for p := range v.Graph.Pages {
		pagePaths[p] = true
	}

	for _, page := range v.Graph.Pages {
		for _, comp := range page.Components {
			for _, act := range comp.Actions {
				target := act.TargetAPI
				if strings.HasPrefix(target, "/") && !strings.HasPrefix(target, "/api/") {
					// It's a page link
					if !pagePaths[target] && target != "/dashboard" && target != "/logout" && target != "/configuracoes" && target != "/relatorios" {
						diags = append(diags, ProjectDiagnostic{
							Code:        "VAL102",
							Severity:    SeverityWarning,
							Title:       fmt.Sprintf("Link de navegação para página inexistente: '%s'", target),
							Description: fmt.Sprintf("A página '%s' tenta navegar para '%s', que ainda não foi criada no projeto.", page.Path, target),
							Location:    page.FilePath,
							Fix:         fmt.Sprintf("Crie a página com 'ge init pagina %s' ou corrija o link.", strings.TrimPrefix(target, "/")),
						})
					}
				}
			}
		}
	}

	return diags
}

func (v *ProjectValidator) checkOrphanEntities() []ProjectDiagnostic {
	var diags []ProjectDiagnostic

	for _, ent := range v.Graph.Entities {
		hasPage := false
		hasAPI := false

		for _, p := range v.Graph.Pages {
			if strings.EqualFold(p.EntityRef, ent.Name) {
				hasPage = true
				break
			}
		}
		for _, a := range v.Graph.APIs {
			if strings.EqualFold(a.EntityRef, ent.Name) || strings.Contains(a.Path, strings.ToLower(ent.Plural)) {
				hasAPI = true
				break
			}
		}

		if !hasPage && !hasAPI {
			diags = append(diags, ProjectDiagnostic{
				Code:        "VAL201",
				Severity:    SeverityInfo,
				Title:       fmt.Sprintf("Entidade de domínio sem interface: '%s'", ent.Name),
				Description: fmt.Sprintf("A entidade '%s' está declarada no banco, mas não possui página ou API correspondente. Se for apenas para uso interno de backend, ignore este aviso.", ent.Name),
				Location:    ent.FilePath,
				Fix:         fmt.Sprintf("Gere a interface com 'ge init pagina %s' ou exponha o endpoint com 'ge init api %s'.", strings.ToLower(ent.Plural), strings.ToLower(ent.Plural)),
			})
		}
	}

	return diags
}

func (v *ProjectValidator) checkHardcodedSecrets() []ProjectDiagnostic {
	var diags []ProjectDiagnostic

	secretPatterns := []struct {
		Pattern *regexp.Regexp
		Name    string
	}{
		{Pattern: regexp.MustCompile(`(?i)(api_key|secret|token|password|senha)\s*=\s*["'][a-zA-Z0-9_\-]{8,}["']`), Name: "Chave/Senha hardcoded"},
		{Pattern: regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`), Name: "OpenAI API Key"},
		{Pattern: regexp.MustCompile(`ghp_[a-zA-Z0-9]{20,}`), Name: "GitHub Token"},
	}

	_ = filepath.WalkDir(v.RootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".ge" {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			content := string(data)
			relPath, _ := filepath.Rel(v.RootDir, path)

			for _, sp := range secretPatterns {
				if sp.Pattern.MatchString(content) {
					diags = append(diags, ProjectDiagnostic{
						Code:        "SEC501",
						Severity:    SeverityError,
						Title:       fmt.Sprintf("Possível segredo exposto no código: %s", sp.Name),
						Description: fmt.Sprintf("Foi detectado um padrão de credencial ou chave sensível no arquivo '%s'. Segredos nunca devem estar no código fonte.", relPath),
						Location:    relPath,
						Fix:         "Utilize variáveis de ambiente ou arquivo de configuração externo.",
					})
				}
			}
		}
		return nil
	})

	return diags
}

func (v *ProjectValidator) checkMissingPermissions() []ProjectDiagnostic {
	var diags []ProjectDiagnostic

	if len(v.Graph.Roles) > 1 {
		for _, page := range v.Graph.Pages {
			if len(page.AllowedRoles) == 0 {
				diags = append(diags, ProjectDiagnostic{
					Code:        "SEC201",
					Severity:    SeverityWarning,
					Title:       fmt.Sprintf("Página sem papéis de acesso definidos: '%s'", page.Path),
					Description: fmt.Sprintf("A página '%s' não possui restrição de papéis configurada.", page.Path),
					Location:    page.FilePath,
					Fix:         "Declare os papéis autorizados para esta página.",
				})
			}
		}
	}

	return diags
}
