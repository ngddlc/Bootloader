package downloader

import (
	"context"
	"fmt"
	"homecloud/internal/domain"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

type Prober struct {
	client *http.Client
}

func NewProber(timeout time.Duration) *Prober {
	return &Prober{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (p *Prober) Probe(ctx context.Context, rawURL string) (*domain.DownloadPlan, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Range", "bytes=0-0")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("probe request failed: %w", err)
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body)

	filename := extractFilename(resp, rawURL)

	finalURL := rawURL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}

	switch resp.StatusCode {
	case http.StatusPartialContent:
		contentRange := resp.Header.Get("Content-Range")
		totalSize := parseContentRange(contentRange)

		return &domain.DownloadPlan{
			URL:         finalURL,
			Filename:    filename,
			TotalSize:   totalSize,
			IsMultiPart: true,
			ContentType: resolveContentType(resp.Header.Get("Content-Type"), filename),
		}, nil

	case http.StatusOK:
		return &domain.DownloadPlan{
			URL:         finalURL,
			Filename:    filename,
			TotalSize:   resp.ContentLength,
			IsMultiPart: false,
			ContentType: resolveContentType(resp.Header.Get("Content-Type"), filename),
		}, nil

	default:
		return nil, fmt.Errorf("server returned unexpected status: %s", resp.Status)
	}
}

func parseContentRange(rawHeader string) int64 {
	if rawHeader == "" {
		return -1
	}

	slashIdx := strings.LastIndex(rawHeader, "/")
	if slashIdx == -1 || slashIdx+1 >= len(rawHeader) {
		return -1
	}

	totalStr := rawHeader[slashIdx+1:]
	if totalStr == "*" {
		return -1
	}

	size, err := strconv.ParseInt(totalStr, 10, 64)
	if err != nil {
		return -1
	}
	return size
}

func extractFilename(resp *http.Response, rawURL string) string {
	cd := resp.Header.Get("Content-Disposition")
	if cd != "" {
		_, params, err := mime.ParseMediaType(cd)
		if err == nil {
			if filename, ok := params["filename"]; ok && filename != "" {
				return filename
			}
		}
	}

	parsedURL, err := url.Parse(rawURL)
	if err == nil {
		base := path.Base(parsedURL.Path)
		if base != "" && base != "." && base != "/" {
			return base
		}
	}

	return "downloaded_file"
}

func resolveContentType(headerType, filename string) string {
	if headerType != "" {
		return headerType
	}
	ext := path.Ext(filename)
	if mimeType := mime.TypeByExtension(ext); mimeType != "" {
		return mimeType
	}
	return "application/octet-stream"
}
