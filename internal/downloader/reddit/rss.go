package reddit

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

const redditUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/153.0.0.0 Safari/537.36"

var (
	firstEntryPattern = regexp.MustCompile(
		`(?is)<entry\b.*?</entry>`,
	)

	galleryPattern = regexp.MustCompile(
		`(?i)https?://(?:www\.)?reddit\.com/gallery/[^\s<>"']+`,
	)

	originalImagePattern = regexp.MustCompile(
		`(?i)https://i\.redd\.it/[^\s<>"']+`,
	)

	previewImagePattern = regexp.MustCompile(
		`(?i)https://preview\.redd\.it/[^\s<>"']+`,
	)
)

type RSSStrategy struct {
	client *http.Client
}

func NewRSSStrategy() *RSSStrategy {
	return &RSSStrategy{
		client: &http.Client{},
	}
}

func (s *RSSStrategy) Name() string {
	return "reddit-rss"
}

func (s *RSSStrategy) Download(
	ctx context.Context,
	detectedPlatform platform.Platform,
	rawURL string,
	outputDir string,
) (*media.Result, error) {
	if detectedPlatform != platform.Reddit {
		return nil, downloader.ErrUnsupported
	}

	canonicalURL, err := s.resolveURL(
		ctx,
		rawURL,
	)
	if err != nil {
		return nil, err
	}

	rssURL, err := makeRSSURL(
		canonicalURL,
	)
	if err != nil {
		return nil, err
	}

	body, err := s.fetch(
		ctx,
		rssURL,
	)
	if err != nil {
		return nil, err
	}

	entry := firstEntryPattern.Find(body)
	if len(entry) == 0 {
		return nil, downloader.ErrNoMedia
	}

	entryText := html.UnescapeString(
		string(entry),
	)

	// O RSS de galerias só expõe uma thumbnail.
	// Não vamos fingir que isso é a galeria completa.
	if galleryPattern.MatchString(entryText) {
		return nil, fmt.Errorf(
			"%w: galeria do Reddit",
			downloader.ErrUnsupportedMedia,
		)
	}

	imageURL := firstImageURL(
		entryText,
	)

	if imageURL == "" {
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

	item, err := s.downloadImage(
		ctx,
		imageURL,
		outputDir,
	)
	if err != nil {
		return nil, err
	}

	return &media.Result{
		Platform: platform.Reddit,
		URL:      rawURL,
		Items: []media.Item{
			item,
		},
	}, nil
}

func (s *RSSStrategy) resolveURL(
	ctx context.Context,
	rawURL string,
) (string, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		rawURL,
		nil,
	)
	if err != nil {
		return "", err
	}

	request.Header.Set(
		"User-Agent",
		redditUserAgent,
	)

	response, err := s.client.Do(request)
	if err != nil {
		return "", fmt.Errorf(
			"falha ao resolver URL do Reddit: %w",
			err,
		)
	}
	defer response.Body.Close()

	if response.Request == nil ||
		response.Request.URL == nil {
		return rawURL, nil
	}

	return response.Request.URL.String(), nil
}

func makeRSSURL(
	rawURL string,
) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf(
			"URL inválida do Reddit: %w",
			err,
		)
	}

	parsed.RawQuery = ""
	parsed.Fragment = ""

	path := strings.TrimSuffix(
		parsed.Path,
		"/",
	)

	if !strings.Contains(
		path,
		"/comments/",
	) {
		return "", downloader.ErrInvalidURL
	}

	parsed.Path = path + "/.rss"

	return parsed.String(), nil
}

func (s *RSSStrategy) fetch(
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
		return nil, err
	}

	request.Header.Set(
		"User-Agent",
		redditUserAgent,
	)

	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf(
			"falha ao acessar RSS do Reddit: %w",
			err,
		)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 ||
		response.StatusCode >= 300 {
		return nil, fmt.Errorf(
			"RSS do Reddit retornou HTTP %d",
			response.StatusCode,
		)
	}

	return io.ReadAll(response.Body)
}

func firstImageURL(
	entry string,
) string {
	if found := originalImagePattern.FindString(
		entry,
	); found != "" {
		return html.UnescapeString(found)
	}

	if found := previewImagePattern.FindString(
		entry,
	); found != "" {
		return html.UnescapeString(found)
	}

	return ""
}

func (s *RSSStrategy) downloadImage(
	ctx context.Context,
	rawURL string,
	outputDir string,
) (media.Item, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return media.Item{}, err
	}

	extension := strings.ToLower(
		filepath.Ext(parsed.Path),
	)

	switch extension {
	case ".jpg", ".jpeg", ".png", ".webp":
	default:
		extension = ".jpg"
	}

	filename := "reddit_001" + extension

	destination := filepath.Join(
		outputDir,
		filename,
	)

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		rawURL,
		nil,
	)
	if err != nil {
		return media.Item{}, err
	}

	request.Header.Set(
		"User-Agent",
		redditUserAgent,
	)

	response, err := s.client.Do(request)
	if err != nil {
		return media.Item{}, fmt.Errorf(
			"falha ao baixar imagem do Reddit: %w",
			err,
		)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 ||
		response.StatusCode >= 300 {
		return media.Item{}, fmt.Errorf(
			"CDN do Reddit retornou HTTP %d",
			response.StatusCode,
		)
	}

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
		Type:     media.Photo,
		Path:     destination,
		Filename: filename,
	}, nil
}
