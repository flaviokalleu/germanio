package intelligence

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/compiler/parser"
)

// ProjectAnalyzer scans and understands an existing Germanio codebase.
type ProjectAnalyzer struct {
	RootDir string
}

// NewProjectAnalyzer creates an analyzer for the given directory.
func NewProjectAnalyzer(dir string) *ProjectAnalyzer {
	return &ProjectAnalyzer{RootDir: dir}
}

// Analyze scans all .ge files and populates a full ProjectKnowledgeGraph.
func (a *ProjectAnalyzer) Analyze() (*ProjectKnowledgeGraph, error) {
	manifest, _ := LoadManifest(a.RootDir)
	projName := "GermanioApp"
	projType := "saas"
	if manifest != nil {
		projName = manifest.Name
		projType = manifest.Type
	}

	graph := NewProjectKnowledgeGraph(projName, projType)
	if manifest != nil {
		for _, capID := range manifest.Capabilities {
			if def, ok := StandardCapabilities[capID]; ok {
				graph.AddCapability(&CapabilityNode{
					ID:           def.ID,
					Name:         def.Name,
					Description:  def.Description,
					Status:       def.Status,
					Dependencies: def.Dependencies,
				})
			}
		}
		graph.Roles = manifest.Roles
	}

	// Walk all .ge files in project
	err := filepath.WalkDir(a.RootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		if filepath.Ext(path) == ".ge" {
			relPath, _ := filepath.Rel(a.RootDir, path)
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			a.analyzeFile(relPath, string(data), graph)
		}
		return nil
	})

	return graph, err
}

func (a *ProjectAnalyzer) analyzeFile(relPath, content string, graph *ProjectKnowledgeGraph) {
	// 1. Check for table / model declarations
	extractTablesFromSource(relPath, content, graph)

	// 2. Check for pages and UI components
	extractPagesFromSource(relPath, content, graph)

	// 3. Check for backend event handlers / APIs
	extractAPIsFromSource(relPath, content, graph)
}

func extractTablesFromSource(relPath, content string, graph *ProjectKnowledgeGraph) {
	// Match: tabela Nome / modelo Nome
	lines := strings.Split(content, "\n")
	var currentEntity *EntityNode

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "tabela ") || strings.HasPrefix(trimmed, "modelo ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				name := parts[1]
				currentEntity = &EntityNode{
					Name:        name,
					Plural:      name + "s",
					TableName:   strings.ToLower(name) + "s",
					Fields:      make([]FieldMeta, 0),
					Relations:   make([]RelationMeta, 0),
					ReadRoles:   []string{"Administrador"},
					WriteRoles:  []string{"Administrador"},
					DeleteRoles: []string{"Administrador"},
					FilePath:    relPath,
				}
				graph.AddEntity(currentEntity)
			}
			continue
		}

		// If indented inside an entity
		if currentEntity != nil {
			if strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t") {
				// field definition e.g. "nome: texto obrigatorio", "preco: dinheiro", "email: email unico"
				fParts := strings.SplitN(trimmed, ":", 2)
				if len(fParts) == 2 {
					fName := strings.TrimSpace(fParts[0])
					fTypeFull := strings.TrimSpace(fParts[1])
					fTypeParts := strings.Fields(fTypeFull)
					fType := FieldTexto
					req := false
					uniq := false

					if len(fTypeParts) > 0 {
						fType = FieldType(fTypeParts[0])
					}
					if strings.Contains(fTypeFull, "obrigatorio") || strings.Contains(fTypeFull, "obrigatoria") {
						req = true
					}
					if strings.Contains(fTypeFull, "unico") || strings.Contains(fTypeFull, "unica") {
						uniq = true
					}

					currentEntity.Fields = append(currentEntity.Fields, FieldMeta{
						Name:       fName,
						Type:       fType,
						Required:   req,
						Unique:     uniq,
						Searchable: fType == FieldTexto || fType == FieldEmail,
						Filterable: fType == FieldStatus || fType == FieldData,
					})
				}
			} else if trimmed != "" && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "//") {
				currentEntity = nil
			}
		}
	}

	// Try AST parse for full structure if available
	prog, err := parser.ParseGermanio(relPath, content)
	if err == nil && prog != nil {
		for _, m := range prog.Models {
			if graph.FindEntity(m.Name) == nil {
				ent := &EntityNode{
					Name:      m.Name,
					Plural:    m.Name + "s",
					TableName: strings.ToLower(m.Name) + "s",
					FilePath:  relPath,
				}
				for _, f := range m.Fields {
					ent.Fields = append(ent.Fields, FieldMeta{
						Name:     f.Name,
						Type:     FieldType(f.Type),
						Required: f.Required,
						Unique:   f.Unique,
					})
				}
				graph.AddEntity(ent)
			}
		}
	}
}

func extractPagesFromSource(relPath, content string, graph *ProjectKnowledgeGraph) {
	// Look for: pagina "/rota"
	pageRegex := regexp.MustCompile(`pagina\s+"([^"]+)"`)
	titleRegex := regexp.MustCompile(`titulo\s+"([^"]+)"`)

	matches := pageRegex.FindAllStringSubmatchIndex(content, -1)
	for i, match := range matches {
		path := content[match[2]:match[3]]
		endPos := len(content)
		if i+1 < len(matches) {
			endPos = matches[i+1][0]
		}
		pageBlock := content[match[0]:endPos]

		title := strings.TrimPrefix(path, "/")
		if tMatch := titleRegex.FindStringSubmatch(pageBlock); len(tMatch) > 1 {
			title = tMatch[1]
		}

		page := &PageNode{
			Name:         strings.TrimPrefix(path, "/"),
			Path:         path,
			Title:        title,
			Layout:       "dashboard",
			FilePath:     relPath,
			AllowedRoles: []string{"Administrador", "Usuario"},
		}

		// Check buttons & components inside page
		btnRegex := regexp.MustCompile(`botao\s+(?:primario\s+|secundario\s+)?"([^"]+)"\s+"([^"]+)"`)
		for _, bMatch := range btnRegex.FindAllStringSubmatch(pageBlock, -1) {
			label := bMatch[1]
			target := bMatch[2]
			page.Components = append(page.Components, ComponentNode{
				Name: fmt.Sprintf("Botao_%s", label),
				Kind: "botao",
				Actions: []ActionBinding{
					{
						Name:      strings.ToLower(label),
						Label:     label,
						TargetAPI: target,
						IsZombie:  target == "" || target == "#",
					},
				},
			})
		}

		// Infer Entity relation from path (e.g. /clientes -> Cliente)
		for _, ent := range graph.Entities {
			if strings.Contains(strings.ToLower(path), strings.ToLower(ent.Plural)) ||
				strings.Contains(strings.ToLower(path), strings.ToLower(ent.Name)) {
				page.EntityRef = ent.Name
				break
			}
		}

		graph.AddPage(page)
	}

	// Also check custom pages in AST
	prog, err := parser.ParseGermanio(relPath, content)
	if err == nil && prog != nil {
		for _, cp := range prog.Pages {
			if graph.FindPage(cp.Path) == nil {
				page := &PageNode{
					Name:         strings.TrimPrefix(cp.Path, "/"),
					Path:         cp.Path,
					Title:        cp.Title,
					Layout:       "dashboard",
					FilePath:     relPath,
					AllowedRoles: []string{"Administrador"},
				}
				for _, b := range cp.Blocks {
					switch bl := b.(type) {
					case *ast.PageHero:
						page.Components = append(page.Components, ComponentNode{
							Name: "HeroBlock",
							Kind: "hero",
						})
						for _, btn := range bl.Buttons {
							page.Components = append(page.Components, ComponentNode{
								Name: "Botao_" + btn.Label,
								Kind: "botao",
								Actions: []ActionBinding{
									{
										Name:      btn.Label,
										Label:     btn.Label,
										TargetAPI: btn.URL,
										IsZombie:  btn.URL == "",
									},
								},
							})
						}
					case *ast.PageSection:
						page.Components = append(page.Components, ComponentNode{
							Name: "Secao_" + bl.Title,
							Kind: "secao",
						})
					}
				}
				graph.AddPage(page)
			}
		}
	}
}

func extractAPIsFromSource(relPath, content string, graph *ProjectKnowledgeGraph) {
	// Look for: quando receber <acao> com <params>
	// or: rota GET "/api/..."
	apiRegex := regexp.MustCompile(`quando\s+receber\s+([a-zA-Z0-9_]+)`)
	for _, match := range apiRegex.FindAllStringSubmatch(content, -1) {
		action := match[1]
		graph.AddAPI(&APINode{
			Method:      "POST",
			Path:        "/api/" + action,
			HandlerName: action,
			FilePath:    relPath,
		})
	}
}
