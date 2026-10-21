package instagram

import (
	"context"
	"errors"
	"github.com/jrdev95/super-picos-downloader/internal/downloader"
	"github.com/jrdev95/super-picos-downloader/internal/media"
	"github.com/jrdev95/super-picos-downloader/internal/platform"
	"os"
	"path/filepath"
	"testing"
)

type verificationFake struct {
	kind  media.Type
	err   error
	calls int
}

func (f *verificationFake) Name() string { return "fake" }
func (f *verificationFake) Download(ctx context.Context, p platform.Platform, u, d string) (*media.Result, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	path := filepath.Join(d, "media")
	if err := os.WriteFile(path, []byte("test"), 0600); err != nil {
		return nil, err
	}
	return &media.Result{Platform: p, URL: u, Items: []media.Item{{Type: f.kind, Path: path}}}, nil
}
func TestCobaltPhotoVerification(t *testing.T) {
	for _, tc := range []struct {
		name, path    string
		source, other media.Type
		fail          bool
		calls         int
	}{
		{"video replacement", "/reel/test/", media.Photo, media.Video, false, 1},
		{"confirmed photo", "/reel/test/", media.Photo, media.Photo, false, 1},
		{"unverified", "/reel/test/", media.Photo, media.Photo, true, 1},
		{"normal photo", "/p/test/", media.Photo, media.Video, false, 0},
		{"normal video", "/reel/test/", media.Video, media.Photo, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := &verificationFake{kind: tc.source}
			other := &verificationFake{kind: tc.other}
			if tc.fail {
				other.err = errors.New("HTTP 400")
			}
			v := &verifiedCobalt{source: src, alternatives: downloader.NewChain(other)}
			result, err := downloader.NewChain(v, other).Download(context.Background(), platform.Instagram, "https://www.instagram.com"+tc.path, t.TempDir())
			if other.calls != tc.calls {
				t.Fatalf("calls=%d", other.calls)
			}
			if tc.fail {
				if err == nil || result != nil {
					t.Fatal("unverified photo accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			want := tc.source
			if tc.calls > 0 {
				want = tc.other
			}
			if result.Items[0].Type != want {
				t.Fatal(result)
			}
			if _, err := os.Stat(result.Items[0].Path); err != nil {
				t.Fatal(err)
			}
		})
	}
}
