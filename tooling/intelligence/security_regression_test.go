package intelligence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestTransactionRejectsPathsOutsideProject(t *testing.T) {
	tx := NewTransactionManager(t.TempDir(), false, NewProjectManifest("Teste", "saas"))
	for _, path := range []string{"../escape.ge", "/tmp/escape.ge", ""} {
		if err := tx.AddFile(path, "x", "teste"); err == nil {
			t.Fatalf("AddFile aceitou caminho inseguro %q", path)
		}
	}
}

func TestEjectRejectsUnsafeComponentName(t *testing.T) {
	root := t.TempDir()
	engine := NewIntelligenceEngine(root)
	if _, err := engine.EjectComponent("../../escape"); err == nil {
		t.Fatal("eject aceitou nome de componente com traversal")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(root), "escape.ge")); !os.IsNotExist(err) {
		t.Fatal("eject escreveu fora do diretório do projeto")
	}
}

func TestInitProjectRefusesUnknownExistingProjectFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "inicio.ge"), []byte("mostre \"manual\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := NewIntelligenceEngine(root).InitProject(InitProjectOptions{RootDir: root, Name: "Teste", Mode: "rapido"})
	if err == nil {
		t.Fatal("init sobrescreveu projeto existente sem manifesto")
	}
}

func TestImportSpecRejectsUnsafeEntityName(t *testing.T) {
	root := t.TempDir()
	spec := filepath.Join(t.TempDir(), "unsafe.geinit")
	manifest := NewProjectManifest("Teste", "saas")
	manifest.Entities = []string{"../../escape"}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(spec, data, 0644); err != nil {
		t.Fatal(err)
	}
	if err := NewIntelligenceEngine(root).ImportSpec(spec); err == nil {
		t.Fatal("import aceitou entidade com traversal")
	}
}
