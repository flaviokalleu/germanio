package servidor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Isolated steps (CI-10): with GERMANIO_EXECUTOR=docker, a step that names
// an image runs all its commands in ONE container (a cd or a variable set
// by a command reaches the next one, as people expect), with the step's
// variables, no network, no Linux capabilities, no privilege escalation, a
// limit of processes, and the host user's id (files written in the working
// copy are not root's). Each command is echoed before it runs, like the
// local mode; the first failure stops the step.

var envName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// shellQuote quotes s for a POSIX shell.
func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// containerScript: the commands as one script that echoes each one first.
func containerScript(lines []string) string {
	var b strings.Builder
	b.WriteString("set -e\n")
	for _, l := range lines {
		b.WriteString("printf '%s\\n' " + shellQuote("$ "+l) + "\n")
		b.WriteString(l + "\n")
	}
	return b.String()
}

func (x *executor) inContainer(ctx context.Context, log *logBuffer, work, image string, env []string, lines []string) bool {
	args := []string{"run", "--rm", "--network", "none", "--cap-drop", "ALL", "--security-opt", "no-new-privileges",
		"--pids-limit", "512", "--user", strconv.Itoa(os.Getuid()) + ":" + strconv.Itoa(os.Getgid()),
		"-v", work + ":/builds/projeto", "-w", "/builds/projeto", "-e", "HOME=/builds/projeto"}
	var cmdEnv []string
	for _, kv := range env {
		k, _, _ := strings.Cut(kv, "=")
		if k == "PATH" || k == "HOME" || !envName.MatchString(k) {
			continue // the image keeps its own PATH and HOME
		}
		args = append(args, "-e", k) // the value comes from the docker client's environment, never the command line
		cmdEnv = append(cmdEnv, kv)
	}
	args = append(args, image, "sh", "-c", containerScript(lines))
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Env = append(os.Environ(), cmdEnv...)
	cmd.Stdout, cmd.Stderr = log, log
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Fprintln(log, "ERRO: tempo limite excedido")
		} else if ctx.Err() == nil {
			fmt.Fprintf(log, "ERRO: o comando terminou com %v\n", err)
		}
		return false
	}
	return true
}
