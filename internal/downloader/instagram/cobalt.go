package instagram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jrdev95/super-picos-downloader/internal/downloader"
	"github.com/jrdev95/super-picos-downloader/internal/media"
	"github.com/jrdev95/super-picos-downloader/internal/platform"
)

type CobaltStrategy struct {
	endpoint string
	client   *http.Client
}

func NewCobaltStrategy(endpoint string) *CobaltStrategy {
	return &CobaltStrategy{strings.TrimSpace(endpoint), &http.Client{Timeout: 30 * time.Second}}
}
func (s *CobaltStrategy) Name() string { return "cobalt-instagram" }

type cobaltItem struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}
type cobaltResponse struct {
	Status string       `json:"status"`
	URL    string       `json:"url"`
	Picker []cobaltItem `json:"picker"`
	Error  struct {
		Code string `json:"code"`
	} `json:"error"`
}

func (s *CobaltStrategy) Download(parent context.Context, site platform.Platform, rawURL, outputDir string) (result *media.Result, err error) {
	if site != platform.Instagram || s.endpoint == "" {
		return nil, downloader.ErrUnsupported
	}
	// A falha/timeout local deve permitir a próxima estratégia; cancelamento do pai não.
	defer func() {
		if err != nil {
			if parent.Err() != nil {
				err = parent.Err()
			} else {
				slog.Warn("Cobalt falhou; tentando próximo downloader", "error", err)
				err = fmt.Errorf("Cobalt: %v", err)
			}
		}
	}()
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	body, _ := json.Marshal(map[string]string{"url": rawURL})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	response, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API HTTP %d", response.StatusCode)
	}
	var data cobaltResponse
	if err = json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&data); err != nil {
		return nil, err
	}
	var items []cobaltItem
	switch data.Status {
	case "picker":
		items = data.Picker
	case "redirect", "tunnel":
		items = []cobaltItem{{URL: data.URL}}
	case "error":
		return nil, fmt.Errorf("API: %s", data.Error.Code)
	default:
		return nil, fmt.Errorf("resposta não suportada: %s", data.Status)
	}
	if len(items) == 0 || len(items) > 100 {
		return nil, fmt.Errorf("quantidade de mídias inválida: %d", len(items))
	}
	dir, err := os.MkdirTemp(outputDir, "cobalt-")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(dir)
		}
	}()
	result = &media.Result{Platform: site, URL: rawURL}
	for i, item := range items {
		downloaded, e := s.fetch(ctx, item, dir, i)
		if e != nil {
			return nil, e
		}
		result.Items = append(result.Items, downloaded)
	}
	slog.Info("download via Cobalt concluído", "items", len(result.Items))
	return result, nil
}
func (s *CobaltStrategy) fetch(ctx context.Context, item cobaltItem, dir string, index int) (media.Item, error) {
	parsed, err := url.Parse(item.URL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return media.Item{}, fmt.Errorf("URL de mídia inválida")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, item.URL, nil)
	if err != nil {
		return media.Item{}, err
	}
	response, err := s.client.Do(req)
	if err != nil {
		return media.Item{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return media.Item{}, fmt.Errorf("mídia HTTP %d", response.StatusCode)
	}
	// Valida os bytes, em vez de aceitar páginas HTML ou confiar no nome remoto.
	head := make([]byte, 512)
	n, err := io.ReadFull(response.Body, head)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return media.Item{}, err
	}
	head = head[:n]
	kind, ext := media.Unknown, ""
	switch http.DetectContentType(head) {
	case "image/jpeg":
		kind, ext = media.Photo, ".jpg"
	case "image/png":
		kind, ext = media.Photo, ".png"
	case "image/webp":
		kind, ext = media.Photo, ".webp"
	case "video/mp4":
		kind, ext = media.Video, ".mp4"
	case "video/webm":
		kind, ext = media.Video, ".webm"
	default:
		return media.Item{}, fmt.Errorf("formato de mídia não suportado")
	}
	filename := fmt.Sprintf("%03d%s", index+1, ext)
	path := filepath.Join(dir, filename)
	file, err := os.Create(path)
	if err != nil {
		return media.Item{}, err
	}
	const limit = 100 << 20
	size, copyErr := io.Copy(file, io.LimitReader(io.MultiReader(bytes.NewReader(head), response.Body), limit+1))
	closeErr := file.Close()
	if copyErr != nil {
		return media.Item{}, copyErr
	}
	if closeErr != nil {
		return media.Item{}, closeErr
	}
	if size > limit {
		return media.Item{}, fmt.Errorf("mídia excede limite de 100 MiB")
	}
	return media.Item{Type: kind, Path: path, Filename: filename}, nil
}
