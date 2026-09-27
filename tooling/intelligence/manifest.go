package intelligence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// FileOwnershipStatus tracks who owns a given file or component in the project.
type FileOwnershipStatus string

const (
	OwnershipManaged    FileOwnershipStatus = "managed"    // Automatically managed by Germanio
	OwnershipCustomized FileOwnershipStatus = "customized" // Modified by developer, protected from overwrite
	OwnershipEjected    FileOwnershipStatus = "ejected"    // Ejected from standard registry
	OwnershipUserOwned  FileOwnershipStatus = "user-owned" // Created manually by user
)

// FileMeta stores ownership and checksum information for a file.
type FileMeta struct {
	Path        string              `json:"path"`
	Ownership   FileOwnershipStatus `json:"ownership"`
	Description string              `json:"description,omitempty"`
	Checksum    string              `json:"checksum,omitempty"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// DesignSystemConfig stores design system tokens and presets.
type DesignSystemConfig struct {
	Preset       string   `json:"preset"`        // "saas_moderno", "enterprise", "minimalista", "developer", "financeiro", etc.
	Theme        string   `json:"theme"`         // "escuro", "claro", "auto"
	PrimaryColor string   `json:"primary_color"` // "#00D9FF", "#3b82f6", etc.
	FontFamily   string   `json:"font_family"`   // "Plus Jakarta Sans", "Inter", etc.
	Radius       string   `json:"radius"`        // "suave", "arredondado", "reto"
	Density      string   `json:"density"`       // "confortavel", "compacto"
	Navigation   string   `json:"navigation"`    // "sidebar", "topbar", "hibrido"
	Supported    []string `json:"supported_breakpoints,omitempty"`
}

// ProjectManifest represents the persistent, versionable intelligence metadata of a Germanio project.
type ProjectManifest struct {
	ManifestVersion string                 `json:"manifest_version"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`     // "saas", "crm", "ecommerce", "api", "dashboard", "site", etc.
	Audience        string                 `json:"audience"` // "B2B", "B2C", "B2B2C", "interno"
	MultiTenant     bool                   `json:"multi_tenant"`
	Database        string                 `json:"database"` // "sqlite", "postgres", "mysql"
	AuthStrategy    string                 `json:"auth_strategy"`
	Capabilities    []string               `json:"capabilities"`
	Entities        []string               `json:"entities"`
	Pages           []string               `json:"pages"`
	Roles           []string               `json:"roles"`
	Design          DesignSystemConfig     `json:"design"`
	Files           map[string]FileMeta    `json:"files"`
	EjectedBlocks   map[string]string      `json:"ejected_blocks,omitempty"`
	Config          map[string]interface{} `json:"config,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

const ManifestFileName = "germanio.project.json"
const LegacyManifestFileName = ".germanio/project.json"

// NewProjectManifest creates a default manifest for a project.
func NewProjectManifest(name, projType string) *ProjectManifest {
	return &ProjectManifest{
		ManifestVersion: "1.0",
		Name:            name,
		Type:            projType,
		Audience:        "B2B",
		MultiTenant:     false,
		Database:        "sqlite",
		AuthStrategy:    "email_senha",
		Capabilities:    make([]string, 0),
		Entities:        make([]string, 0),
		Pages:           make([]string, 0),
		Roles:           []string{"Administrador", "Usuario"},
		Design: DesignSystemConfig{
			Preset:       "saas_moderno",
			Theme:        "escuro",
			PrimaryColor: "#00D9FF",
			FontFamily:   "Plus Jakarta Sans",
			Radius:       "suave",
			Density:      "confortavel",
			Navigation:   "sidebar",
		},
		Files:         make(map[string]FileMeta),
		EjectedBlocks: make(map[string]string),
		Config:        make(map[string]interface{}),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// LoadManifest loads the manifest from directory, or nil if not found.
func LoadManifest(dir string) (*ProjectManifest, error) {
	path := filepath.Join(dir, ManifestFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var m ProjectManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.Files == nil {
		m.Files = make(map[string]FileMeta)
	}
	if m.EjectedBlocks == nil {
		m.EjectedBlocks = make(map[string]string)
	}
	return &m, nil
}

// Save writes the manifest atomically to disk.
func (m *ProjectManifest) Save(dir string) error {
	m.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dir, ManifestFileName)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}
	legacyPath := filepath.Join(dir, LegacyManifestFileName)
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(legacyPath, data, 0644)
}

// RegisterFile marks a file in the manifest.
func (m *ProjectManifest) RegisterFile(relPath string, status FileOwnershipStatus, desc string) {
	if m.Files == nil {
		m.Files = make(map[string]FileMeta)
	}
	m.Files[relPath] = FileMeta{
		Path:        relPath,
		Ownership:   status,
		Description: desc,
		UpdatedAt:   time.Now(),
	}
}

// IsProtected returns true if the file is customized or ejected by user and cannot be silently overwritten.
func (m *ProjectManifest) IsProtected(relPath string) bool {
	if m.Files == nil {
		return false
	}
	meta, ok := m.Files[relPath]
	if !ok {
		return false
	}
	return meta.Ownership == OwnershipCustomized || meta.Ownership == OwnershipEjected || meta.Ownership == OwnershipUserOwned
}
