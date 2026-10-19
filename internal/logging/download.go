package logging

import (
	"context"
	"strings"
	"sync"
)

type downloadKey struct{}
type DownloadTrace struct {
	mu    sync.Mutex
	steps []string
}

func WithDownloadTrace(ctx context.Context) (context.Context, *DownloadTrace) {
	t := &DownloadTrace{}
	return context.WithValue(ctx, downloadKey{}, t), t
}
func DownloadStep(ctx context.Context, step string) {
	if t, ok := ctx.Value(downloadKey{}).(*DownloadTrace); ok {
		t.mu.Lock()
		defer t.mu.Unlock()
		step = strings.Join(strings.Fields(step), " ")
		if r := []rune(step); len(r) > 350 {
			step = string(r[:350]) + "…"
		}
		t.steps = append(t.steps, step)
	}
}
func (t *DownloadTrace) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return strings.Join(t.steps, "; ")
}

func (t *DownloadTrace) SuccessfulMethod() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i := len(t.steps) - 1; i >= 0; i-- {
		if strings.HasSuffix(t.steps[i], ": OK") {
			return strings.TrimSuffix(t.steps[i], ": OK")
		}
	}
	return "direto"
}

// Diagnostics omits successful strategies already identified in the download event.
func (t *DownloadTrace) Diagnostics() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	var details []string
	for _, step := range t.steps {
		if !strings.HasSuffix(step, ": OK") {
			details = append(details, step)
		}
	}
	return strings.Join(details, "; ")
}
