package logging

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

func TestConsole(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(NewConsoleHandler(&out)).With("worker", 2).WithGroup("download")
	logger.Debug("hidden")
	logger.Error("download falhou", "error", errors.New("login required\nsecond strategy\x1b[31m"))
	got := out.String()
	for _, want := range []string{"ERROR", "worker: \"2\"", "download.error: \"login required\"", "    \"second strategy\\x1b[31m\""} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, "hidden") || strings.Contains(got, "\x1b") {
		t.Fatal("debug or terminal escape leaked")
	}
}

func TestConcurrentEvents(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(NewConsoleHandler(&out))
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); logger.With("worker", 1).Info("event", "items", 2) }()
	}
	wg.Wait()
	blocks := strings.Split(strings.TrimSuffix(out.String(), "\n\n"), "\n\n")
	if len(blocks) != 20 {
		t.Fatalf("got %d events", len(blocks))
	}
	for _, block := range blocks {
		if strings.Count(block, "\n") != 2 {
			t.Fatalf("interleaved event: %q", block)
		}
	}
}
