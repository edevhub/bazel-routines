package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/semaphore"
)

var ErrCancelled = errors.New("execution cancelled")
var ErrClosed = errors.New("scheduler closed")

type Task func(context.Context) error

type Queue interface {
	Push(context.Context, Task) error
}

type Scheduler struct {
	logger   *slog.Logger
	queue    *queue
	sem      *semaphore.Weighted
	inFlight atomic.Int32
}

func NewScheduler(limit int, logger *slog.Logger) *Scheduler {
	return &Scheduler{
		queue:  newQueue(limit),
		sem:    semaphore.NewWeighted(int64(limit)),
		logger: logger,
	}
}

func (s *Scheduler) Push(ctx context.Context, t Task) error {
	if s.queue.IsClosed() {
		return ErrClosed
	}
	s.inFlight.Add(1)
	if err := s.sem.Acquire(ctx, 1); err != nil {
		return ErrCancelled
	}
	return s.queue.Push(t)
}

func (s *Scheduler) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	done := make(chan struct{})
	doneClose := sync.OnceFunc(func() {
		close(done)
	})
	defer s.queue.Close()

	err := func() error {
		for {
			select {
			case <-ctx.Done():
				s.logger.Debug("Context cancelled. Stopping scheduler")
				return ctx.Err()
			case t, open := <-s.queue.Consume():
				if !open {
					return nil
				}
				wg.Add(1)
				go func(t Task) {
					s.logger.Debug(fmt.Sprintf("In flight: %d", s.inFlight.Load()))
					defer func() {
						s.sem.Release(1)
						if queued := s.inFlight.Add(-1); queued == 0 {
							s.logger.Info("Task queue is empty. Stopping scheduler")
							doneClose()
						}
						wg.Done()
					}()
					if err := t(ctx); err != nil {
						s.logger.Info("Task failed. Closing scheduler", slog.Any("error", err))
						s.queue.Close()
					}
				}(t)
			case <-done:
				return nil
			}
		}
	}()

	wg.Wait()
	return err
}

type queue struct {
	q      chan Task
	closed bool
	mx     sync.Mutex
	once   sync.Once
}

func newQueue(limit int) *queue {
	return &queue{
		q: make(chan Task, limit),
	}
}

func (q *queue) Consume() <-chan Task {
	return q.q
}

func (q *queue) Push(t Task) error {
	q.mx.Lock()
	defer q.mx.Unlock()
	if q.closed {
		return ErrClosed
	}
	q.q <- t
	return nil
}

func (q *queue) Close() {
	q.mx.Lock()
	defer q.mx.Unlock()
	q.once.Do(func() {
		q.closed = true
		close(q.q)
	})
}

func (q *queue) IsClosed() bool {
	q.mx.Lock()
	defer q.mx.Unlock()
	return q.closed
}
