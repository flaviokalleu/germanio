package runtime

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// O core (compiler, runtime, cli, tooling) é genérico: nenhum nome, caminho,
// formato ou protocolo de um sistema externo. Isso mora em adaptadores
// (integracoes/ de cada aplicação). Ver skills/germanio-simplicity (34b).
func TestCoreNaoConheceSistemasExternos(t *testing.T) {
	forbidden := regexp.MustCompile(`(?i)gitlab|\bCI_[A-Z]|JOB-TOKEN|/api/v4|after_script|before_script|git_info|job_info|glrt-|glpat-|on_success|allow_failure`)
	for _, dir := range []string{"../compiler", "../runtime", "../cli", "../tooling"} {
		filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, _ := os.ReadFile(path)
			for i, line := range strings.Split(string(data), "\n") {
				if m := forbidden.FindString(line); m != "" {
					t.Errorf("%s:%d menciona %q (pertence a um adaptador em integracoes/)", path, i+1, m)
				}
			}
			return nil
		})
	}
}
