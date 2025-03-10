package task

import (
	"context"
	"fmt"
	"github.com/edevhub/bazel-routines/internal/engine"
	"sync"
	"time"
)

type Routine struct {
	Label    Label
	Summary  string
	Metadata map[string]string
	Status   Status
	Tasks    []*Task

	startTime time.Time
	endTime   time.Time
	done      chan struct{}

	mx sync.Mutex
}

func NewRoutine(label Label, tasks ...*Task) *Routine {
	return &Routine{
		Label: label,
		Tasks: tasks,
		done:  make(chan struct{}),
	}
}

func (r *Routine) Run(ctx context.Context, q engine.Queue) error {
	r.mx.Lock()
	if r.Status != StatusNotStarted {
		r.mx.Unlock()
		return fmt.Errorf("routine was already started(%s)", r.Status)
	}
	r.Status = StatusRunning
	r.mx.Unlock()

	r.startTime = time.Now()
	defer func() {
		r.endTime = time.Now()
		close(r.done)
	}()

	for _, task := range r.Tasks {
		if err := q.Push(ctx, task.Run); err != nil {
			r.Status.FromError(err)
			return err
		}
		<-task.Done()
		if task.Status.Is(StatusSucceeded) {
			continue
		}

		if err := task.Error(); err != nil {
			r.Status.FromError(err)
			return err
		}
	}

	r.Status = StatusSucceeded
	return nil
}

func (r *Routine) Done() <-chan struct{} {
	return r.done
}

func (r *Routine) StartTime() time.Time {
	return r.startTime
}

func (r *Routine) EndTime() time.Time {
	<-r.Done()
	return r.endTime
}

func (r *Routine) ElapsedTime() time.Duration {
	select {
	case <-r.Done():
		return r.EndTime().Sub(r.StartTime())
	default:
		return time.Since(r.StartTime())
	}
}
