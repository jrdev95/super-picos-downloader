package tools

import (
	"fmt"
	"os/exec"
)

type Tools struct {
	YTDLP     string
	FFmpeg    string
	GalleryDL string
}

func Detect() (*Tools, error) {
	ytdlp, err := exec.LookPath("yt-dlp")
	if err != nil {
		return nil, fmt.Errorf(
			"yt-dlp não encontrado no PATH: %w",
			err,
		)
	}

	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf(
			"ffmpeg não encontrado no PATH: %w",
			err,
		)
	}

	galleryDL, err := exec.LookPath("gallery-dl")
	if err != nil {
		return nil, fmt.Errorf(
			"gallery-dl não encontrado no PATH: %w",
			err,
		)
	}

	return &Tools{
		YTDLP:     ytdlp,
		FFmpeg:    ffmpeg,
		GalleryDL: galleryDL,
	}, nil
}
