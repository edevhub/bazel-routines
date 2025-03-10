package task

import (
	"context"
	"errors"
	"fmt"
	"github.com/edevhub/bazel-routines/internal/engine"
	"github.com/edevhub/bazel-routines/pkg/exec"
	"sync"
	"time"
)

// Status represents the state of a task.
type Status int
type Label string

const (
	StatusNotStarted Status = iota
	StatusRunning
	StatusSucceeded
	StatusFailed
	StatusCancelled
)

func (s Status) String() string {
	switch s {
	case StatusNotStarted:
		return "Not Started"
	case StatusRunning:
		return "Running"
	case StatusSucceeded:
		return "Succeeded"
	case StatusFailed:
		return "Failed"
	case StatusCancelled:
		return "Cancelled"
	default:
		return "Unknown"
	}
}

func (s Status) Is(status Status) bool {
	return s == status
}

func (s *Status) FromError(err error) {
	if err == nil {
		*s = StatusSucceeded
		return
	}
	if errors.Is(err, context.Canceled) {
		*s = StatusCancelled
		return
	}
	if errors.Is(err, engine.ErrCancelled) {
		*s = StatusCancelled
		return
	}
	*s = StatusFailed
}

type Task struct {
	Label  Label
	Status Status

	process   *exec.Process
	startTime time.Time
	endTime   time.Time

	mx sync.Mutex
}

func NewTask(label Label, p *exec.Process) *Task {
	return &Task{Label: label, process: p}
}

func (t *Task) Run(ctx context.Context) error {
	t.mx.Lock()
	if t.Status != StatusNotStarted {
		t.mx.Unlock()
		return fmt.Errorf("task was already started(%s)", t.Status)
	}
	t.Status = StatusRunning
	t.startTime = time.Now()
	t.mx.Unlock()
	defer func() {
		t.endTime = time.Now()
	}()

	if err := t.process.Start(ctx); err != nil {
		t.Status.FromError(err)
		return err
	}

	select {
	case <-ctx.Done():
		t.Status = StatusCancelled
		return nil
	case <-t.process.Done():
		err := t.process.Error()
		t.Status.FromError(err)
		return err
	}
}

func (t *Task) Done() <-chan struct{} {
	return t.process.Done()
}

func (t *Task) Error() error {
	<-t.Done()
	return t.process.Error()
}

func (t *Task) StartTime() time.Time {
	return t.startTime
}

func (t *Task) EndTime() time.Time {
	<-t.Done()
	return t.endTime
}

func (t *Task) ElapsedTime() time.Duration {
	select {
	case <-t.Done():
		return t.EndTime().Sub(t.StartTime())
	default:
		return time.Since(t.StartTime())
	}
}
