package downloader

import (
	"context"
	"fmt"
	"homecloud/internal/domain"
	"io"
	"net/http"
	"os"
	"time"
)

type SingleDownloader struct {
	client *http.Client
}

func NewSingleDownloader(timeout time.Duration) *SingleDownloader {
	return &SingleDownloader{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (d *SingleDownloader) Download(ctx context.Context, plan *domain.DownloadPlan, destPath string) error {
	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	req, err := http.NewRequestWithContext(ctx, "GET", plan.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %s", resp.Status)
	}

	written, err := io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save data: %w", err)
	}
	fmt.Printf("Скачивание завершено, скачано байт: %d", written)
	return nil
}
