package engine

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/edevhub/bazel-routines/pkg/log"
	"github.com/stretchr/testify/suite"
)

func TestSchedulerSuite(t *testing.T) {
	suite.Run(t, &SchedulerTestSuite{
		logger: log.DefaultJSONLogger(slog.LevelWarn),
	})
}

type SchedulerTestSuite struct {
	suite.Suite
	logger *slog.Logger
}

func (s *SchedulerTestSuite) TestSuccessfulTaskExecution() {
	scheduler := NewScheduler(1, s.logger)
	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(1)

	executed := false
	task := func(ctx context.Context) error {
		executed = true
		wg.Done()
		return nil
	}

	go func() {
		s.NoError(scheduler.Run(ctx))
	}()

	err := scheduler.Push(ctx, task)
	s.NoError(err, "Failed to push task")

	wg.Wait()
	s.True(executed, "Task was not executed")
}

func (s *SchedulerTestSuite) TestConcurrentTasksExecution() {
	concurrency := 3
	scheduler := NewScheduler(concurrency, s.logger)
	ctx := context.Background()

	var wg sync.WaitGroup
	var inFlight atomic.Int32
	taskCount := 50
	wg.Add(taskCount)

	executed := make([]bool, taskCount)
	for i := 0; i < taskCount; i++ {
		id := i // capture loop variable
		task := func(ctx context.Context) error {
			defer wg.Done()

			inf := inFlight.Add(1)
			s.LessOrEqual(inf, int32(concurrency))
			executed[id] = true
			inFlight.Add(-1)
			return nil
		}

		go func(t Task) {
			err := scheduler.Push(ctx, t)
			s.NoError(err, "Failed to push task %d", i)
		}(task)
	}

	go func() {
		s.NoError(scheduler.Run(ctx))
	}()

	wg.Wait()

	for i, e := range executed {
		s.True(e, "Task %d was not executed", i)
	}

	s.Equal(int32(0), inFlight.Load(), "Resources were not properly released")
}

func (s *SchedulerTestSuite) TestTaskFailure() {
	scheduler := NewScheduler(1, s.logger)
	ctx := context.Background()

	expectedErr := errors.New("task error")
	var wg sync.WaitGroup
	wg.Add(1)

	task := func(ctx context.Context) error {
		defer wg.Done()
		return expectedErr
	}

	go func() {
		s.NoError(scheduler.Run(ctx))
	}()

	err := scheduler.Push(ctx, task)
	s.NoError(err, "Failed to push task")

	wg.Wait()
	time.Sleep(10 * time.Millisecond) // Give scheduler time to process the error

	err = scheduler.Push(ctx, func(ctx context.Context) error { return nil })
	s.True(errors.Is(err, ErrClosed), "Expected ErrClosed, got %v", err)
}

func (s *SchedulerTestSuite) TestSchedulerClosure() {
	scheduler := NewScheduler(1, s.logger)
	ctx := context.Background()

	ctxWithCancel, cancel := context.WithCancel(ctx)
	var runErr error
	go func() {
		runErr = scheduler.Run(ctxWithCancel)
	}()

	time.Sleep(10 * time.Millisecond)
	cancel()

	time.Sleep(10 * time.Millisecond)

	err := scheduler.Push(ctx, func(ctx context.Context) error { return nil })
	s.True(errors.Is(err, ErrClosed), "Expected ErrCancelled, got %v", err)
	s.True(errors.Is(runErr, context.Canceled), "Expected context.Canceled error, got %v", runErr)
}

func (s *SchedulerTestSuite) TestGracefulShutdown() {
	scheduler := NewScheduler(2, s.logger)
	ctx, cancel := context.WithCancel(context.Background())

	var completedTasks atomic.Int32
	taskCount := 10

	for i := 0; i < taskCount; i++ {
		task := func(ctx context.Context) error {
			<-time.After(100 * time.Millisecond)
			completedTasks.Add(1)
			return nil
		}
		go func(t Task) {
			_ = scheduler.Push(ctx, t)
		}(task)
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := scheduler.Run(ctx)
	s.ErrorIs(err, context.Canceled, "Expected context.Canceled error, got %v", err)

	completed := completedTasks.Load()
	s.Greater(completed, int32(0), "No tasks completed before cancellation")
	s.Less(completed, int32(taskCount), "All tasks completed despite cancellation")
}
