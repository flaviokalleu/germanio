package servidor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// CI-10: a step's commands run in one isolated container.
func TestPassoEmConteiner(t *testing.T) {
	image := os.Getenv("GERMANIO_DOCKER_IMAGEM")
	if image == "" {
		image = "redis:7-alpine"
	}
	if exec.Command("docker", "image", "inspect", image).Run() != nil {
		t.Skip("docker ou a imagem " + image + " não disponível")
	}
	work := t.TempDir()
	os.MkdirAll(filepath.Join(work, "sub"), 0o755)
	x := &executor{mode: "docker"}
	log := &logBuffer{}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	ok := x.inContainer(ctx, log, work, image, []string{"PATH=/host/bin", "SEGREDO=s3cr3t-valor"}, []string{
		"cd sub",
		"pwd",
		`echo "$SEGREDO" > segredo.txt`,
		"grep CapEff /proc/self/status",
		"wget -q -T 2 -O- http://1.1.1.1 >/dev/null 2>&1 && echo COM_REDE || echo SEM_REDE",
	})
	out := log.Final()
	if !ok {
		t.Fatalf("o passo falhou:\n%s", out)
	}
	for _, want := range []string{"$ cd sub", "/builds/projeto/sub", "CapEff:\t0000000000000000", "SEM_REDE"} {
		if !strings.Contains(out, want) {
			t.Errorf("saída sem %q:\n%s", want, out)
		}
	}
	b, err := os.ReadFile(filepath.Join(work, "sub", "segredo.txt"))
	if err != nil || strings.TrimSpace(string(b)) != "s3cr3t-valor" {
		t.Fatalf("a variável não chegou ao container: %q %v", b, err)
	}
	if st, _ := os.Stat(filepath.Join(work, "sub", "segredo.txt")); st.Sys().(*syscall.Stat_t).Uid != uint32(os.Getuid()) {
		t.Fatal("o arquivo criado no container não é do usuário do host")
	}
	// the first failure stops the step; later commands do not run
	log = &logBuffer{}
	if x.inContainer(ctx, log, work, image, nil, []string{"echo antes", "exit 3", "echo depois"}) {
		t.Fatal("um comando que falha não parou o passo")
	}
	if out := log.Final(); !strings.Contains(out, "antes") || strings.Contains(out, "\ndepois") {
		t.Fatalf("saída depois da falha:\n%s", out)
	}
}
