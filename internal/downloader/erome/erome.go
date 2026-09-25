package erome

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jrdev95/super-picos-downloader/internal/downloader"
	"github.com/jrdev95/super-picos-downloader/internal/media"
	"github.com/jrdev95/super-picos-downloader/internal/platform"
)

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"

var (
	videoPattern = regexp.MustCompile(
		`(?i)<source[^>]+src=["']([^"']+)["']`,
	)

	imagePattern = regexp.MustCompile(
		`(?i)<img[^>]+class=["'][^"']*\bimg-(?:front|back)\b[^"']*["'][^>]+(?:data-src|src)=["']([^"']+)["']`,
	)
)

type Downloader struct {
	client *http.Client
}

func New() *Downloader {
	return &Downloader{
		client: &http.Client{},
	}
}

func (d *Downloader) CanHandle(
	detectedPlatform platform.Platform,
) bool {
	return detectedPlatform == platform.Erome
}

func (d *Downloader) Download(
	ctx context.Context,
	rawURL string,
	outputDir string,
) (*media.Result, error) {
	page, err := d.fetchPage(
		ctx,
		rawURL,
	)
	if err != nil {
		return nil, err
	}

	videoURLs := uniqueURLs(
		extractURLs(videoPattern, page),
	)

	imageURLs := uniqueURLs(
		extractURLs(imagePattern, page),
	)

	if len(videoURLs) == 0 &&
		len(imageURLs) == 0 {
		return nil, downloader.ErrNoMedia
	}

	if err := os.MkdirAll(
		outputDir,
		0o755,
	); err != nil {
		return nil, fmt.Errorf(
			"falha ao criar diretório: %w",
			err,
		)
	}

	items := make(
		[]media.Item,
		0,
		len(videoURLs)+len(imageURLs),
	)

	for index, mediaURL := range videoURLs {
		item, err := d.downloadFile(
			ctx,
			rawURL,
			mediaURL,
			outputDir,
			fmt.Sprintf(
				"erome_video_%03d.mp4",
				index+1,
			),
			media.Video,
		)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	for index, mediaURL := range imageURLs {
		extension := imageExtension(mediaURL)

		item, err := d.downloadFile(
			ctx,
			rawURL,
			mediaURL,
			outputDir,
			fmt.Sprintf(
				"erome_photo_%03d%s",
				index+1,
				extension,
			),
			media.Photo,
		)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if len(items) == 0 {
		return nil, downloader.ErrNoMedia
	}

	return &media.Result{
		Platform: platform.Erome,
		URL:      rawURL,
		Items:    items,
	}, nil
}

func (d *Downloader) fetchPage(
	ctx context.Context,
	rawURL string,
) ([]byte, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		rawURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"falha ao criar requisição do Erome: %w",
			err,
		)
	}

	setHeaders(request)

	response, err := d.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf(
			"falha ao acessar Erome: %w",
			err,
		)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 ||
		response.StatusCode >= 300 {
		return nil, fmt.Errorf(
			"Erome retornou HTTP %d",
			response.StatusCode,
		)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"falha ao ler página do Erome: %w",
			err,
		)
	}

	return body, nil
}

func extractURLs(
	pattern *regexp.Regexp,
	page []byte,
) []string {
	matches := pattern.FindAllSubmatch(
		page,
		-1,
	)

	urls := make(
		[]string,
		0,
		len(matches),
	)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		mediaURL := strings.TrimSpace(
			html.UnescapeString(
				string(match[1]),
			),
		)

		if mediaURL == "" {
			continue
		}

		if strings.HasPrefix(
			mediaURL,
			"//",
		) {
			mediaURL = "https:" + mediaURL
		}

		urls = append(
			urls,
			mediaURL,
		)
	}

	return urls
}

func uniqueURLs(
	urls []string,
) []string {
	seen := make(
		map[string]struct{},
	)

	result := make(
		[]string,
		0,
		len(urls),
	)

	for _, mediaURL := range urls {
		if _, exists := seen[mediaURL]; exists {
			continue
		}

		seen[mediaURL] = struct{}{}

		result = append(
			result,
			mediaURL,
		)
	}

	return result
}

func (d *Downloader) downloadFile(
	ctx context.Context,
	referer string,
	rawURL string,
	outputDir string,
	filename string,
	mediaType media.Type,
) (media.Item, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		rawURL,
		nil,
	)
	if err != nil {
		return media.Item{}, err
	}

	setHeaders(request)

	request.Header.Set(
		"Referer",
		referer,
	)

	response, err := d.client.Do(request)
	if err != nil {
		return media.Item{}, fmt.Errorf(
			"falha ao baixar mídia do Erome: %w",
			err,
		)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 ||
		response.StatusCode >= 300 {
		return media.Item{}, fmt.Errorf(
			"CDN do Erome retornou HTTP %d",
			response.StatusCode,
		)
	}

	destination := filepath.Join(
		outputDir,
		filename,
	)

	file, err := os.Create(destination)
	if err != nil {
		return media.Item{}, err
	}

	_, copyErr := io.Copy(
		file,
		response.Body,
	)

	closeErr := file.Close()

	if copyErr != nil {
		return media.Item{}, copyErr
	}

	if closeErr != nil {
		return media.Item{}, closeErr
	}

	return media.Item{
		Type:     mediaType,
		Path:     destination,
		Filename: filename,
	}, nil
}

func setHeaders(
	request *http.Request,
) {
	request.Header.Set(
		"User-Agent",
		userAgent,
	)

	request.Header.Set(
		"Referer",
		"https://www.erome.com/",
	)
}

func imageExtension(
	rawURL string,
) string {
	parsed, err := url.Parse(rawURL)
	if err == nil {
		extension := strings.ToLower(
			filepath.Ext(parsed.Path),
		)

		switch extension {
		case ".jpg", ".jpeg", ".png", ".webp":
			return extension
		}
	}

	return ".jpg"
}
