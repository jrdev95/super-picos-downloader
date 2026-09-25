package tools

import "testing"

func TestDetect(t *testing.T) {
	tools, err := Detect()
	if err != nil {
		t.Skipf(
			"dependências externas não disponíveis neste ambiente: %v",
			err,
		)
	}

	if tools.YTDLP == "" {
		t.Fatal("caminho do yt-dlp não deveria estar vazio")
	}

	if tools.FFmpeg == "" {
		t.Fatal("caminho do ffmpeg não deveria estar vazio")
	}

	if tools.GalleryDL == "" {
		t.Fatal("caminho do gallery-dl não deveria estar vazio")
	}
}
