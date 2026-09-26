package cron

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/flaviokalleu/germanio/compiler/ast"
	"github.com/flaviokalleu/germanio/runtime/httpclient"
)

// Scheduler manages ticker-based cron jobs.
type Scheduler struct {
	jobs   []*ast.CronJob
	client *httpclient.Client
	stopCh chan struct{}
}

// Novo creates a new scheduler.
func Novo(jobs []*ast.CronJob) *Scheduler {
	return &Scheduler{
		jobs:   jobs,
		client: httpclient.Novo(),
		stopCh: make(chan struct{}),
	}
}

// Iniciar starts all cron jobs in background goroutines.
func (s *Scheduler) Iniciar() {
	for _, job := range s.jobs {
		dur := parseDuration(job.Every)
		if dur == 0 {
			fmt.Printf("[cron] Intervalo inválido: %q, ignorando\n", job.Every)
			continue
		}

		fmt.Printf("[cron] Agendado: %s a cada %s\n", describeJob(job), dur)

		go s.runJob(job, dur)
	}
}

// Parar stops all running cron jobs.
func (s *Scheduler) Parar() {
	close(s.stopCh)
}

func (s *Scheduler) runJob(job *ast.CronJob, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.executeJob(job)
		case <-s.stopCh:
			return
		}
	}
}

func (s *Scheduler) executeJob(job *ast.CronJob) {
	switch job.Action {
	case "chamar", "call":
		// Call an external URL
		url := job.Target
		if url == "" {
			fmt.Printf("[cron] Erro: URL vazia para job 'chamar'\n")
			return
		}
		fmt.Printf("[cron] Chamando: %s\n", url)
		resp, err := s.client.Chamar("GET", url, nil)
		if err != nil {
			fmt.Printf("[cron] Erro ao chamar %s: %s\n", url, err)
			return
		}
		// Truncate response for logging
		body := string(resp)
		if len(body) > 200 {
			body = body[:200] + "..."
		}
		fmt.Printf("[cron] Resposta de %s: %s\n", url, body)

	default:
		// Generic action (e.g., "limpar sessoes")
		fmt.Printf("[cron] Executando ação: %s %s\n", job.Action, job.Target)
	}
}

// parseDuration converts "5 minutos" / "1 hora" / "30 segundos" to time.Duration.
func parseDuration(spec string) time.Duration {
	spec = strings.TrimSpace(strings.ToLower(spec))
	parts := strings.Fields(spec)

	if len(parts) < 2 {
		return 0
	}

	n, err := strconv.Atoi(parts[0])
	if err != nil || n <= 0 {
		return 0
	}

	unit := parts[1]
	switch {
	case strings.HasPrefix(unit, "segundo") || strings.HasPrefix(unit, "second"):
		return time.Duration(n) * time.Second
	case strings.HasPrefix(unit, "minuto") || strings.HasPrefix(unit, "minute"):
		return time.Duration(n) * time.Minute
	case strings.HasPrefix(unit, "hora") || strings.HasPrefix(unit, "hour"):
		return time.Duration(n) * time.Hour
	case strings.HasPrefix(unit, "dia") || strings.HasPrefix(unit, "day"):
		return time.Duration(n) * 24 * time.Hour
	default:
		return 0
	}
}

func describeJob(job *ast.CronJob) string {
	if job.Target != "" {
		return job.Action + " " + job.Target
	}
	return job.Action
}
