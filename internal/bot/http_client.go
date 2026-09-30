package bot

import "net/http"

// Keep enough idle connections for workers, polling and status messages.
// Clone instead of mutating the transport shared by unrelated HTTP clients.
func newTelegramClient(workers int) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = min(max(workers+4, 8), 128)
	transport.MaxIdleConns = max(transport.MaxIdleConns, transport.MaxIdleConnsPerHost)
	return &http.Client{Transport: transport}
}
