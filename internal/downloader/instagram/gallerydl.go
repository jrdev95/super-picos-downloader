package instagram

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jrdev95/super-picos-downloader/internal/downloader"
	"github.com/jrdev95/super-picos-downloader/internal/executil"
	"github.com/jrdev95/super-picos-downloader/internal/media"
	"github.com/jrdev95/super-picos-downloader/internal/platform"
)

// GalleryDLStrategy usa gallery-dl para baixar publicações do Instagram.
//
// O arquivo de cookies é opcional. Quando informado, deve estar no formato
// Netscape/Mozilla aceito pelo gallery-dl. O caminho pode conter variáveis
// de ambiente, como $HOME.
type GalleryDLStrategy struct {
	path        string
	cookiesFile string
	run         func(context.Context, []string) (string, string, error)
}

func NewGalleryDLStrategy(
	path string,
	cookiesFile string,
) *GalleryDLStrategy {
	return &GalleryDLStrategy{
		path:        path,
		cookiesFile: cookiesFile,
	}
}

func (s *GalleryDLStrategy) Name() string {
	return "gallery-dl-instagram"
}

func (s *GalleryDLStrategy) Download(
	ctx context.Context,
	detectedPlatform platform.Platform,
	rawURL string,
	outputDir string,
) (*media.Result, error) {
	if detectedPlatform != platform.Instagram {
		return nil, downloader.ErrUnsupported
	}

	if strings.TrimSpace(s.path) == "" {
		return nil, fmt.Errorf("caminho do gallery-dl não foi definido")
	}

	if strings.TrimSpace(rawURL) == "" {
		return nil, downloader.ErrInvalidURL
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf(
			"falha ao criar diretório de saída: %w",
			err,
		)
	}

	result, err := s.attempt(ctx, rawURL, outputDir, "")
	if !errors.Is(err, downloader.ErrAuthentication) || ctx.Err() != nil {
		return result, err
	}
	cookiesFile := strings.TrimSpace(os.ExpandEnv(s.cookiesFile))
	if cookiesFile == "" {
		return nil, err
	}
	info, statErr := os.Stat(cookiesFile)
	if statErr != nil {
		return nil, fmt.Errorf("%w: arquivo de cookies indisponível: %v", downloader.ErrAuthentication, statErr)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: cookies devem ser um arquivo", downloader.ErrAuthentication)
	}
	slog.Info("Instagram exige autenticação; repetindo download com cookies")
	return s.attempt(ctx, rawURL, outputDir, cookiesFile)
}

func (s *GalleryDLStrategy) attempt(ctx context.Context, rawURL, outputDir, cookiesFile string) (*media.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Isola arquivos parciais de cada tentativa para não misturar resultados.
	attemptDir, err := os.MkdirTemp(outputDir, "instagram-")
	if err != nil {
		return nil, err
	}
	success := false
	defer func() {
		if !success {
			_ = os.RemoveAll(attemptDir)
		}
	}()
	outputDir = attemptDir
	// Ignora configurações e sessões persistidas fora do arquivo explícito.
	args := []string{"--no-input", "--config-ignore", "--cache-file", ":memory:"}
	if cookiesFile != "" {
		args = append(args, "--cookies", cookiesFile)
	}

	args = append(
		args,
		"-D",
		outputDir,
		"--",
		rawURL,
	)

	run := s.run
	if run == nil {
		run = s.runCommand
	}
	stdout, stderr, commandErr := run(ctx, args)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	if commandErr != nil {
		message := strings.TrimSpace(stderr)
		if message == "" {
			message = strings.TrimSpace(stdout)
		}
		if requiresAuthentication(message) {
			return nil, fmt.Errorf("%w: %s", downloader.ErrAuthentication, message)
		}
		return nil, fmt.Errorf("gallery-dl falhou: %w: %s", commandErr, message)
	}

	items := itemsFromOutput(
		stdout,
		outputDir,
	)

	if len(items) == 0 {
		var err error

		items, err = scanDownloadedMedia(outputDir)
		if err != nil {
			return nil, err
		}
	}

	if len(items) == 0 {
		return nil, downloader.ErrNoMedia
	}

	success = true
	return &media.Result{
		Platform: platform.Instagram,
		URL:      rawURL,
		Items:    items,
	}, nil
}

// itemsFromOutput preserva a ordem em que o gallery-dl informou os arquivos.
// Isso é útil para manter a ordem original de carrosséis do Instagram.
func itemsFromOutput(
	output string,
	outputDir string,
) []media.Item {
	scanner := bufio.NewScanner(
		strings.NewReader(output),
	)

	items := make([]media.Item, 0)
	seen := make(map[string]struct{})

	for scanner.Scan() {
		path := strings.TrimSpace(
			scanner.Text(),
		)

		if path == "" {
			continue
		}

		if !filepath.IsAbs(path) {
			path = filepath.Join(
				outputDir,
				path,
			)
		}

		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}

		mediaType := instagramMediaType(path)
		if mediaType == media.Unknown {
			continue
		}

		if _, exists := seen[path]; exists {
			continue
		}

		seen[path] = struct{}{}

		items = append(
			items,
			media.Item{
				Type:     mediaType,
				Path:     path,
				Filename: filepath.Base(path),
			},
		)
	}

	return items
}

func scanDownloadedMedia(
	outputDir string,
) ([]media.Item, error) {
	items := make([]media.Item, 0)

	err := filepath.WalkDir(
		outputDir,
		func(
			path string,
			entry os.DirEntry,
			err error,
		) error {
			if err != nil {
				return err
			}

			if entry.IsDir() {
				return nil
			}

			mediaType := instagramMediaType(path)
			if mediaType == media.Unknown {
				return nil
			}

			items = append(
				items,
				media.Item{
					Type:     mediaType,
					Path:     path,
					Filename: filepath.Base(path),
				},
			)

			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"falha ao localizar mídias baixadas: %w",
			err,
		)
	}

	sort.Slice(
		items,
		func(i, j int) bool {
			return items[i].Path < items[j].Path
		},
	)

	return items, nil
}

func instagramMediaType(
	path string,
) media.Type {
	switch strings.ToLower(
		filepath.Ext(path),
	) {
	case ".jpg", ".jpeg", ".png", ".webp":
		return media.Photo

	case ".mp4", ".mkv", ".webm", ".mov", ".m4v":
		return media.Video

	default:
		return media.Unknown
	}
}

func (s *GalleryDLStrategy) runCommand(ctx context.Context, args []string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd := executil.CommandContext(ctx, s.path, args...)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func requiresAuthentication(message string) bool {
	message = strings.ToLower(message)
	for _, marker := range []string{"loginrequired", "login required", "login_required", "please log in", "authentication required", "cookies are required", "requires authentication", "you must be logged in"} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}
