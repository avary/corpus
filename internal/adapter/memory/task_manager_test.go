package memory

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bornholm/corpus/internal/core/port"
	"github.com/pkg/errors"
)

func TestTaskManager(t *testing.T) {
	tr := NewTaskRunner(10, 24*time.Hour, time.Minute)

	var executed atomic.Int64

	tr.Register("dummy", port.TaskHandlerFunc(func(ctx context.Context, task port.Task, events chan port.TaskEvent) error {
		t.Logf("[%s] start", task.ID())
		events <- port.NewTaskEvent(port.WithTaskProgress(0.1))
		events <- port.NewTaskEvent(port.WithTaskProgress(0.5))
		events <- port.NewTaskEvent(port.WithTaskProgress(1))
		t.Logf("[%s] done", task.ID())
		executed.Add(1)
		return nil
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	total := int64(100)

	for range total {
		task := &dummyTask{
			id: port.NewTaskID(),
		}
		t.Logf("Scheduling task %s", task.ID())
		tr.Schedule(ctx, task)
	}

	if err := tr.Run(ctx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("%+v", errors.WithStack(err))
	}

	t.Logf("executed: %d", executed.Load())

	if e, g := total, executed.Load(); e != g {
		t.Logf("executed: expected %d, got %d", e, g)
	}

	taskHeaders, err := tr.List(ctx)
	if err != nil {
		t.Fatalf("%+v", errors.WithStack(err))
	}

	if e, g := int(total), len(taskHeaders); e != g {
		t.Logf("len(taskHeaders): expected %d, got %d", e, g)
	}

	for _, header := range taskHeaders {
		state, err := tr.State(ctx, header.ID)
		if err != nil {
			t.Fatalf("%+v", errors.WithStack(err))
		}

		if state.ScheduledAt.IsZero() {
			t.Errorf("task.ScheduledAt should not be zero value")
		}
	}
}

type dummyTask struct {
	id port.TaskID
}

// ID implements port.Task.
func (d *dummyTask) ID() port.TaskID {
	return d.id
}

// Type implements port.Task.
func (d *dummyTask) Type() port.TaskType {
	return "dummy"
}

var _ port.Task = &dummyTask{}
