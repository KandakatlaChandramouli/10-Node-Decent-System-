package runtime

import (
	"context"
	"sync"
	"time"
)

type TaskFunc func(ctx context.Context) error

type Scheduler struct {
	mu     sync.Mutex
	tasks  map[string]time.Duration
	stopCh chan struct{}
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		tasks:  make(map[string]time.Duration),
		stopCh: make(chan struct{}),
	}
}

func (s *Scheduler) ScheduleRecurring(ctx context.Context, name string, interval time.Duration, task TaskFunc) {
	s.mu.Lock()
	s.tasks[name] = interval
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				_ = task(ctx)
			case <-s.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	select {
	case <-s.stopCh:
	default:
		close(s.stopCh)
	}
}
