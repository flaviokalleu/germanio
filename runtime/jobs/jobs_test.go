package jobs

import (
	"testing"
	"time"
)

// A full queue refuses work visibly: Submit says so and the refusal is
// counted, never lost in silence.
func TestFilaCheiaRecusaVisivel(t *testing.T) {
	q := Nova(1, 1)
	defer q.Close()
	block := make(chan struct{})
	q.Submit("ocupa", func() { <-block })
	time.Sleep(20 * time.Millisecond) // the worker takes it
	q.Submit("espera", func() {})
	if q.Submit("excesso", func() {}) {
		t.Fatal("a fila cheia aceitou mais trabalho")
	}
	close(block)
	if q.Stats()["rejected"] != 1 {
		t.Fatalf("recusa não contada: %v", q.Stats())
	}
}
