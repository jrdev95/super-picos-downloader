package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"
)

// ConsoleHandler keeps each event and its details together across workers.
type ConsoleHandler struct {
	out   io.Writer
	mu    *sync.Mutex
	attrs []slog.Attr
	group string
}

func NewConsoleHandler(out io.Writer) *ConsoleHandler {
	return &ConsoleHandler{out: out, mu: &sync.Mutex{}}
}

func (h *ConsoleHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= slog.LevelInfo
}

func (h *ConsoleHandler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	fmt.Fprintf(&b, "[%s] %-5s %s\n", r.Time.Format("02/01 15:04:05"), r.Level, strconv.Quote(r.Message))
	for _, a := range h.attrs {
		writeAttr(&b, "", a)
	}
	r.Attrs(func(a slog.Attr) bool { writeAttr(&b, h.group, a); return true })
	b.WriteByte('\n')
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.out, b.String())
	return err
}

func writeAttr(b *strings.Builder, prefix string, a slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}
	key := prefix + a.Key
	if a.Value.Kind() == slog.KindGroup {
		if a.Key != "" {
			prefix = key + "."
		}
		for _, child := range a.Value.Group() {
			writeAttr(b, prefix, child)
		}
		return
	}
	value := a.Value.String()
	if a.Value.Kind() == slog.KindDuration {
		value = a.Value.Duration().Round(1000000).String()
	}
	// Quote control characters so external output cannot manipulate the terminal.
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		line = strconv.Quote(strings.TrimSuffix(line, "\r"))
		if i == 0 {
			fmt.Fprintf(b, "  %s: %s\n", key, line)
		} else {
			fmt.Fprintf(b, "    %s\n", line)
		}
	}
}

func (h *ConsoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = append([]slog.Attr(nil), h.attrs...)
	for _, a := range attrs {
		if h.group != "" {
			a = slog.Group(strings.TrimSuffix(h.group, "."), a)
		}
		clone.attrs = append(clone.attrs, a)
	}
	return &clone
}

func (h *ConsoleHandler) WithGroup(name string) slog.Handler {
	clone := *h
	if name != "" {
		clone.group += name + "."
	}
	return &clone
}
