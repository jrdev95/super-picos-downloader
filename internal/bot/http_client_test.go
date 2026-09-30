package bot

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTelegramClientReusesConcurrentConnections(t *testing.T) {
	const parallel = 6
	var opened atomic.Int32
	var arrived atomic.Int32
	var reused atomic.Int32
	reuseRelease := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/warmup" {
			if arrived.Add(1) == parallel {
				close(release)
			}
			select {
			case <-release:
			case <-r.Context().Done():
				return
			}
		}
		if r.URL.Path == "/reuse" {
			if reused.Add(1) == parallel {
				close(reuseRelease)
			}
			select {
			case <-reuseRelease:
			case <-r.Context().Done():
				return
			}
		}
		_, _ = io.WriteString(w, "ok")
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			opened.Add(1)
		}
	}
	server.Start()
	defer server.Close()
	client := newTelegramClient(3)
	client.Timeout = 5 * time.Second
	defer client.CloseIdleConnections()
	fetch := func(path string) {
		resp, err := client.Get(server.URL + path)
		if err != nil {
			t.Error(err)
			return
		}
		_, err = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Error(err)
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < parallel; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); fetch("/warmup") }()
	}
	wg.Wait()
	before := opened.Load()
	for i := 0; i < parallel; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); fetch("/reuse") }()
	}
	wg.Wait()
	if got := opened.Load(); got != before {
		t.Fatalf("opened additional connections: before=%d after=%d", before, got)
	}
}
