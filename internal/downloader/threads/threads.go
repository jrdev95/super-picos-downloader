package threads

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jrdev95/super-picos-downloader/internal/downloader"
	"github.com/jrdev95/super-picos-downloader/internal/media"
	"github.com/jrdev95/super-picos-downloader/internal/platform"
)

const googlebotUserAgent = "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"

var applicationJSONPattern = regexp.MustCompile(
	`(?is)<script[^>]*type=["']application/json["'][^>]*>(.*?)</script>`,
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
	return detectedPlatform == platform.Threads
}

func (d *Downloader) Download(
	ctx context.Context,
	rawURL string,
	outputDir string,
) (*media.Result, error) {
	page, finalURL, err := d.fetchPage(
		ctx,
		rawURL,
	)
	if err != nil {
		return nil, err
	}

	postCode := extractPostCode(finalURL)
	if postCode == "" {
		postCode = extractPostCode(rawURL)
	}

	rootMedia, err := extractRootMedia(
		page,
		postCode,
	)
	if err != nil {
		return nil, err
	}

	entries := extractMediaEntries(rootMedia)
	if len(entries) == 0 {
		return nil, downloader.ErrNoMedia
	}

	if err := os.MkdirAll(
		outputDir,
		0o755,
	); err != nil {
		return nil, fmt.Errorf(
			"falha ao criar diretório de saída: %w",
			err,
		)
	}

	items := make(
		[]media.Item,
		0,
		len(entries),
	)

	for index, entry := range entries {
		item, err := d.downloadEntry(
			ctx,
			entry,
			outputDir,
			index+1,
		)
		if err != nil {
			return nil, err
		}

		items = append(
			items,
			item,
		)
	}

	return &media.Result{
		Platform: platform.Threads,
		URL:      rawURL,
		Items:    items,
	}, nil
}

func (d *Downloader) fetchPage(
	ctx context.Context,
	rawURL string,
) ([]byte, string, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		rawURL,
		nil,
	)
	if err != nil {
		return nil, "", fmt.Errorf(
			"falha ao criar requisição do Threads: %w",
			err,
		)
	}

	request.Header.Set(
		"User-Agent",
		googlebotUserAgent,
	)

	response, err := d.client.Do(request)
	if err != nil {
		return nil, "", fmt.Errorf(
			"falha ao acessar Threads: %w",
			err,
		)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 ||
		response.StatusCode >= 300 {
		return nil, "", fmt.Errorf(
			"Threads retornou HTTP %d",
			response.StatusCode,
		)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, "", fmt.Errorf(
			"falha ao ler página do Threads: %w",
			err,
		)
	}

	finalURL := rawURL
	if response.Request != nil && response.Request.URL != nil {
		finalURL = response.Request.URL.String()
	}

	return body, finalURL, nil
}

func extractPostCode(
	rawURL string,
) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	parts := strings.Split(
		strings.Trim(parsed.Path, "/"),
		"/",
	)

	for index := 0; index < len(parts)-1; index++ {
		if parts[index] == "post" {
			return parts[index+1]
		}
	}

	return ""
}

func extractRootMedia(
	page []byte,
	postCode string,
) (map[string]any, error) {
	if strings.TrimSpace(postCode) == "" {
		return nil, downloader.ErrNoMedia
	}

	matches := applicationJSONPattern.FindAllSubmatch(
		page,
		-1,
	)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		rawJSON := html.UnescapeString(
			string(match[1]),
		)

		var payload any

		if err := json.Unmarshal(
			[]byte(rawJSON),
			&payload,
		); err != nil {
			continue
		}

		if found := findMediaByCode(
			payload,
			postCode,
		); found != nil {
			return found, nil
		}
	}

	return nil, downloader.ErrNoMedia
}

func findMediaByCode(
	value any,
	postCode string,
) map[string]any {
	switch current := value.(type) {
	case map[string]any:
		code, _ := current["code"].(string)

		if code == postCode &&
			hasDownloadableMedia(current) {
			return current
		}

		for _, child := range current {
			if found := findMediaByCode(
				child,
				postCode,
			); found != nil {
				return found
			}
		}

	case []any:
		for _, child := range current {
			if found := findMediaByCode(
				child,
				postCode,
			); found != nil {
				return found
			}
		}
	}

	return nil
}

func hasDownloadableMedia(
	value map[string]any,
) bool {
	if _, ok := value["video_versions"]; ok {
		return true
	}

	if _, ok := value["image_versions2"]; ok {
		return true
	}

	if _, ok := value["carousel_media"]; ok {
		return true
	}

	return false
}

type mediaEntry struct {
	mediaType media.Type
	url       string
}

func extractMediaEntries(
	root map[string]any,
) []mediaEntry {
	if carousel, ok := root["carousel_media"].([]any); ok {
		entries := make(
			[]mediaEntry,
			0,
			len(carousel),
		)

		for _, rawItem := range carousel {
			item, ok := rawItem.(map[string]any)
			if !ok {
				continue
			}

			if entry, ok := extractSingleEntry(item); ok {
				entries = append(
					entries,
					entry,
				)
			}
		}

		if len(entries) > 0 {
			return entries
		}
	}

	if entry, ok := extractSingleEntry(root); ok {
		return []mediaEntry{
			entry,
		}
	}

	return nil
}

func extractSingleEntry(
	value map[string]any,
) (mediaEntry, bool) {
	// Vídeo primeiro.
	// image_versions2 em posts de vídeo é normalmente a capa.
	if versions, ok := value["video_versions"].([]any); ok {
		for _, rawVersion := range versions {
			version, ok := rawVersion.(map[string]any)
			if !ok {
				continue
			}

			mediaURL, _ := version["url"].(string)
			mediaURL = strings.TrimSpace(mediaURL)

			if mediaURL != "" {
				return mediaEntry{
					mediaType: media.Video,
					url:       mediaURL,
				}, true
			}
		}
	}

	imageVersions, ok := value["image_versions2"].(map[string]any)
	if !ok {
		return mediaEntry{}, false
	}

	candidates, ok := imageVersions["candidates"].([]any)
	if !ok {
		return mediaEntry{}, false
	}

	bestURL := ""
	bestArea := int64(-1)

	for _, rawCandidate := range candidates {
		candidate, ok := rawCandidate.(map[string]any)
		if !ok {
			continue
		}

		mediaURL, _ := candidate["url"].(string)
		if strings.TrimSpace(mediaURL) == "" {
			continue
		}

		width := numberAsInt64(
			candidate["width"],
		)

		height := numberAsInt64(
			candidate["height"],
		)

		area := width * height

		if bestURL == "" || area > bestArea {
			bestURL = mediaURL
			bestArea = area
		}
	}

	if bestURL == "" {
		return mediaEntry{}, false
	}

	return mediaEntry{
		mediaType: media.Photo,
		url:       bestURL,
	}, true
}

func numberAsInt64(
	value any,
) int64 {
	switch number := value.(type) {
	case float64:
		return int64(number)

	case json.Number:
		parsed, _ := strconv.ParseInt(
			number.String(),
			10,
			64,
		)

		return parsed

	default:
		return 0
	}
}

func (d *Downloader) downloadEntry(
	ctx context.Context,
	entry mediaEntry,
	outputDir string,
	index int,
) (media.Item, error) {
	extension := extensionFor(
		entry.mediaType,
		entry.url,
	)

	filename := fmt.Sprintf(
		"threads_%03d%s",
		index,
		extension,
	)

	destination := filepath.Join(
		outputDir,
		filename,
	)

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		entry.url,
		nil,
	)
	if err != nil {
		return media.Item{}, fmt.Errorf(
			"falha ao criar requisição da mídia: %w",
			err,
		)
	}

	request.Header.Set(
		"User-Agent",
		googlebotUserAgent,
	)

	response, err := d.client.Do(request)
	if err != nil {
		return media.Item{}, fmt.Errorf(
			"falha ao baixar mídia do Threads: %w",
			err,
		)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 ||
		response.StatusCode >= 300 {
		return media.Item{}, fmt.Errorf(
			"CDN do Threads retornou HTTP %d",
			response.StatusCode,
		)
	}

	file, err := os.Create(destination)
	if err != nil {
		return media.Item{}, fmt.Errorf(
			"falha ao criar arquivo: %w",
			err,
		)
	}

	_, copyErr := io.Copy(
		file,
		response.Body,
	)

	closeErr := file.Close()

	if copyErr != nil {
		return media.Item{}, fmt.Errorf(
			"falha ao salvar mídia: %w",
			copyErr,
		)
	}

	if closeErr != nil {
		return media.Item{}, fmt.Errorf(
			"falha ao fechar mídia: %w",
			closeErr,
		)
	}

	return media.Item{
		Type:     entry.mediaType,
		Path:     destination,
		Filename: filename,
	}, nil
}

func extensionFor(
	mediaType media.Type,
	rawURL string,
) string {
	if mediaType == media.Video {
		return ".mp4"
	}

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
