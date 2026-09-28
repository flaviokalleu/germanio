package jobs

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type Job struct {
	Name string
	Run  func()
}

type Queue struct {
	jobs      chan Job
	queued    int64
	running   int64
	completed int64
	failed    int64
	rejected  int64 // refused because the queue was full
	closed    atomic.Bool
	wg        sync.WaitGroup
}

func Nova(workers int, buffer int) *Queue {
	if workers <= 0 {
		workers = 2
	}
	if buffer <= 0 {
		buffer = 128
	}
	q := &Queue{jobs: make(chan Job, buffer)}
	for range workers {
		q.wg.Add(1)
		go q.worker()
	}
	return q
}

func (q *Queue) worker() {
	defer q.wg.Done()
	for job := range q.jobs {
		atomic.AddInt64(&q.queued, -1)
		atomic.AddInt64(&q.running, 1)
		func() {
			defer func() {
				if recover() != nil {
					atomic.AddInt64(&q.failed, 1)
				}
				atomic.AddInt64(&q.running, -1)
			}()
			job.Run()
			atomic.AddInt64(&q.completed, 1)
		}()
	}
}

func (q *Queue) Submit(name string, fn func()) bool {
	if fn == nil || q == nil || q.closed.Load() {
		return false
	}
	atomic.AddInt64(&q.queued, 1)
	select {
	case q.jobs <- Job{Name: name, Run: fn}:
		return true
	default:
		atomic.AddInt64(&q.queued, -1)
		atomic.AddInt64(&q.rejected, 1)
		fmt.Printf("[germanio] aviso: fila de trabalhos cheia; %q não foi executado\n", name)
		return false
	}
}

func (q *Queue) Stats() map[string]int64 {
	if q == nil {
		return map[string]int64{}
	}
	return map[string]int64{
		"queued":    atomic.LoadInt64(&q.queued),
		"running":   atomic.LoadInt64(&q.running),
		"completed": atomic.LoadInt64(&q.completed),
		"failed":    atomic.LoadInt64(&q.failed),
		"rejected":  atomic.LoadInt64(&q.rejected),
	}
}

func (q *Queue) Close() {
	if q == nil || !q.closed.CompareAndSwap(false, true) {
		return
	}
	close(q.jobs)
	q.wg.Wait()
}
