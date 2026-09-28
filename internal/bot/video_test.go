package bot

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jrdev95/super-picos-downloader/internal/media"
)

func sizedVideo(t *testing.T, path string, size int64) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(size); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPrepareVideosPreservesSmallFiles(t *testing.T) {
	for _, size := range []int64{1, maxVideoBytes - 1, maxVideoBytes} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "small.mp4")
			sizedVideo(t, path, size)
			original := &media.Result{Items: []media.Item{{Type: media.Photo, Path: "photo.jpg", Caption: "photo"}, {Type: media.Video, Path: path, Caption: "video"}}}
			got, cleanup, err := prepareVideos(context.Background(), original, func(context.Context, ...string) error { t.Fatal("unexpected ffmpeg"); return nil })
			defer cleanup()
			if err != nil || !reflect.DeepEqual(got, original) {
				t.Fatalf("got %v, %v", got, err)
			}
		})
	}
}

func TestPrepareVideosCompressesAlbumAndCleansUp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large video.mp4")
	sizedVideo(t, path, maxVideoBytes+1)
	original := &media.Result{URL: "url", Items: []media.Item{{Type: media.Photo, Path: "photo.jpg"}, {Type: media.Video, Path: path, Filename: "original.mp4", Caption: "caption"}}}
	calls := 0
	var output string
	got, cleanup, err := prepareVideos(context.Background(), original, func(ctx context.Context, args ...string) error {
		calls++
		output = args[len(args)-1]
		joined := strings.Join(args, " ")
		for _, required := range []string{"-nostdin", "-preset veryfast", "force_original_aspect_ratio=decrease:force_divisible_by=2", "-c:v libx264", "-c:a aac", "0:a:0?", "+faststart"} {
			if !strings.Contains(joined, required) {
				t.Errorf("missing %s", required)
			}
		}
		if strings.Contains(joined, "-fs ") || strings.Contains(joined, "-t ") {
			t.Fatal("must not truncate")
		}
		size := targetVideoBytes
		if calls == 1 {
			size = maxVideoBytes + 1
		}
		sizedVideo(t, output, size)
		return nil
	})
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(got.Items) != 2 || got.Items[0] != original.Items[0] || got.Items[1].Caption != "caption" || got.Items[1].Path != output {
		t.Fatalf("album changed: %+v", got)
	}
	if original.Items[1].Path != path {
		t.Fatal("mutated result")
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() != maxVideoBytes+1 {
		t.Fatal("original changed")
	}
	cleanup()
	if _, err := os.Stat(filepath.Dir(output)); !os.IsNotExist(err) {
		t.Fatal("temporary directory remains")
	}
}

func TestPrepareVideosFailures(t *testing.T) {
	for _, scenario := range []string{"oversize", "unsafe-margin", "command", "empty", "missing", "canceled"} {
		t.Run(scenario, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "large.mp4")
			sizedVideo(t, path, maxVideoBytes+1)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			var output string
			got, cleanup, err := prepareVideos(ctx, &media.Result{Items: []media.Item{{Type: media.Video, Path: path}}}, func(ctx context.Context, args ...string) error {
				calls++
				output = args[len(args)-1]
				switch scenario {
				case "command":
					return errors.New("ffmpeg unavailable")
				case "canceled":
					cancel()
					return ctx.Err()
				case "missing":
					return nil
				case "empty":
					sizedVideo(t, output, 0)
				case "unsafe-margin":
					sizedVideo(t, output, targetVideoBytes+1)
				default:
					sizedVideo(t, output, maxVideoBytes+1)
				}
				return nil
			})
			defer cleanup()
			if err == nil || got != nil {
				t.Fatal("expected preparation failure")
			}
			if scenario == "canceled" {
				if !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
			} else {
				var sizeErr *videoSizeError
				if !errors.As(err, &sizeErr) {
					t.Fatalf("wrong error: %v", err)
				}
				if !strings.Contains(sendErrorMessage(fmt.Errorf("wrapped: %w", err)), "50 MB") {
					t.Fatal("missing specific message")
				}
			}
			expected := 1
			if scenario == "oversize" || scenario == "unsafe-margin" {
				expected = 3
			}
			if calls != expected {
				t.Fatalf("calls=%d", calls)
			}
			cleanup()
			if _, err := os.Stat(filepath.Dir(output)); !os.IsNotExist(err) {
				t.Fatal("temporary directory remains")
			}
		})
	}
}

func TestPrepareVideosRejectsMissingAndEmptyInput(t *testing.T) {
	for _, exists := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "input.mp4")
		if exists {
			sizedVideo(t, path, 0)
		}
		_, cleanup, err := prepareVideos(context.Background(), &media.Result{Items: []media.Item{{Type: media.Video, Path: path}}}, func(context.Context, ...string) error { t.Fatal("unexpected subprocess"); return nil })
		cleanup()
		if err == nil {
			t.Fatal("expected error")
		}
	}
}

func TestSendErrorMessage(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want string
	}{
		{context.DeadlineExceeded, "cancelada"},
		{context.Canceled, "cancelado"},
		{errors.New("network"), "concluir o envio"},
		{&videoSizeError{size: 63_716_779, cause: errors.New("ffmpeg")}, "63.7 MB"},
		{&videoSizeError{size: 51_000_000}, "Mesmo após a compressão"},
	} {
		if got := sendErrorMessage(fmt.Errorf("wrapped: %w", tc.err)); !strings.Contains(got, tc.want) {
			t.Errorf("%q missing %q", got, tc.want)
		}
	}
}

func TestNextVideoCRF(t *testing.T) {
	for _, tc := range []struct {
		current int
		size    int64
		want    int
	}{
		{28, targetVideoBytes + 1, 32},
		{28, targetVideoBytes * 2, 34},
		{28, targetVideoBytes * 4, 40},
		{36, targetVideoBytes * 10, 40},
	} {
		if got := nextVideoCRF(tc.current, tc.size); got != tc.want {
			t.Errorf("got %d, want %d", got, tc.want)
		}
	}
}

func TestPrepareVideosStopsAfterFirstSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.mp4")
	sizedVideo(t, path, maxVideoBytes+1)
	calls := 0
	_, cleanup, err := prepareVideos(context.Background(), &media.Result{Items: []media.Item{{Type: media.Video, Path: path}}}, func(_ context.Context, args ...string) error {
		calls++
		sizedVideo(t, args[len(args)-1], targetVideoBytes)
		return nil
	})
	defer cleanup()
	if err != nil || calls != 1 {
		t.Fatalf("calls=%d error=%v", calls, err)
	}
}
