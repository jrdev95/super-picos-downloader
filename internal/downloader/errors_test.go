package downloader

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestIsNoMediaOnly(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"none", nil, false},
		{"no media", ErrNoMedia, true},
		{"wrapped", fmt.Errorf("platform: %w", ErrNoMedia), true},
		{"fallback", fmt.Errorf("chain: %w", errors.Join(ErrUnsupported, fmt.Errorf("rss: %w", ErrNoMedia))), true},
		{"all empty", errors.Join(ErrNoMedia, ErrNoMedia), true},
		{"unsupported only", ErrUnsupported, false},
		{"network", errors.Join(errors.New("HTTP 403"), ErrNoMedia), false},
		{"authentication", errors.Join(ErrNoMedia, ErrAuthentication), false},
		{"unavailable", errors.Join(ErrNoMedia, ErrUnavailable), false},
		{"format", errors.Join(ErrNoMedia, ErrUnsupportedMedia), false},
		{"timeout", errors.Join(ErrNoMedia, context.DeadlineExceeded), false},
		{"canceled", errors.Join(ErrNoMedia, context.Canceled), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsNoMediaOnly(tc.err); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
