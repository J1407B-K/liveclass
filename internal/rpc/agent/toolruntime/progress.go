package toolruntime

import (
	"context"
	"sync"
)

type ProgressTracker struct {
	mu                           sync.Mutex
	sinceProgress, reminderEvery int
}

func NewProgressTracker(reminderEvery int) *ProgressTracker {
	if reminderEvery <= 0 {
		reminderEvery = 5
	}
	return &ProgressTracker{reminderEvery: reminderEvery}
}

type progressKey struct{}

func WithProgressTracker(ctx context.Context, tracker *ProgressTracker) context.Context {
	return context.WithValue(ctx, progressKey{}, tracker)
}
func ProgressTrackerFromContext(ctx context.Context) *ProgressTracker {
	tracker, _ := ctx.Value(progressKey{}).(*ProgressTracker)
	return tracker
}
func (t *ProgressTracker) ObserveTool(name string, success bool) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if success && (name == "create_task_plan" || name == "update_task_step" || name == "update_task_plan") {
		t.sinceProgress = 0
		return
	}
	t.sinceProgress++
}
func (t *ProgressTracker) ConsumeReminder() bool {
	if t == nil {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.sinceProgress < t.reminderEvery {
		return false
	}
	t.sinceProgress = 0
	return true
}
