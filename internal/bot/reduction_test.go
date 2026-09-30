package bot

import (
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/jrdev95/super-picos-downloader/internal/media"
)

func TestReduceMediaVideoSelection(t *testing.T) {
	for _, tc := range []struct {
		name      string
		size, out int64
		fail      bool
		calls     int
		replaced  bool
	}{
		{"small", 4_999_999, 0, false, 0, false},
		{"oversized", maxVideoBytes + 1, 0, false, 0, false},
		{"smaller", 5_000_000, 3_000_000, false, 1, true},
		{"larger", 5_000_000, 6_000_000, false, 1, false},
		{"little-saving", 5_000_000, 4_900_000, false, 1, false},
		{"failure", 5_000_000, 0, true, 1, false},
		{"empty", 5_000_000, 0, false, 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "original.mp4")
			sizedVideo(t, path, tc.size)
			original := &media.Result{Items: []media.Item{{Type: media.Video, Path: path, Caption: "caption"}}}
			calls := 0
			var output string
			got, cleanup, err := reduceMedia(context.Background(), original, func(_ context.Context, args ...string) error {
				calls++
				output = args[len(args)-1]
				if tc.fail {
					return errors.New("encoder failure")
				}
				sizedVideo(t, output, tc.out)
				return nil
			})
			defer cleanup()
			if err != nil {
				t.Fatal(err)
			}
			if calls != tc.calls || (got.Items[0].Path != path) != tc.replaced {
				t.Fatalf("calls=%d result=%+v", calls, got)
			}
			if got.Items[0].Caption != "caption" || original.Items[0].Path != path {
				t.Fatal("metadata changed")
			}
			cleanup()
			if output != "" {
				if _, err := os.Stat(filepath.Dir(output)); !os.IsNotExist(err) {
					t.Fatal("temporary directory remains")
				}
			}
			if info, err := os.Stat(path); err != nil || info.Size() != tc.size {
				t.Fatal("original modified")
			}
		})
	}
}

func TestReduceMediaPhotoAndMixedAlbum(t *testing.T) {
	dir := t.TempDir()
	photo := filepath.Join(dir, "photo.png")
	video := filepath.Join(dir, "video.mp4")
	sizedVideo(t, photo, 500_000)
	sizedVideo(t, video, 100)
	original := &media.Result{Items: []media.Item{{Type: media.Photo, Path: photo, Caption: "photo"}, {Type: media.Video, Path: video, Caption: "video"}}}
	got, cleanup, err := reduceMedia(context.Background(), original, func(_ context.Context, args ...string) error {
		f, err := os.Create(args[len(args)-1])
		if err != nil {
			return err
		}
		// Transparent input must become white, not black, in JPEG.
		err = png.Encode(f, image.NewNRGBA(image.Rect(0, 0, 20, 10)))
		closeErr := f.Close()
		if err != nil {
			return err
		}
		return closeErr
	})
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 2 || got.Items[1] != original.Items[1] || got.Items[0].Caption != "photo" {
		t.Fatal("album changed")
	}
	if filepath.Ext(got.Items[0].Path) != ".jpg" {
		t.Fatal("not JPEG")
	}
	f, err := os.Open(got.Items[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 20 || img.Bounds().Dy() != 10 {
		t.Fatal("unexpected dimensions")
	}
	pixel := color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA)
	if pixel.R < 250 || pixel.G < 250 || pixel.B < 250 {
		t.Fatalf("background=%v", pixel)
	}
}

func TestReduceMediaCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "video.mp4")
	sizedVideo(t, path, 5_000_000)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, cleanup, err := reduceMedia(ctx, &media.Result{Items: []media.Item{{Type: media.Video, Path: path}}}, func(context.Context, ...string) error { cancel(); return context.Canceled })
	defer cleanup()
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}
