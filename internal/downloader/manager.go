package downloader

import (
	"context"
	"fmt"
	"homecloud/internal/domain"
	"os"
	"path/filepath"
	"time"
)

type Manager struct {
	prober           *Prober
	singleDownloader *SingleDownloader
	multiDownloader  *MultiDownloader
}

func NewManager(timeout time.Duration, numWorkers int) *Manager {
	return &Manager{
		prober:           NewProber(timeout),
		singleDownloader: NewSingleDownloader(0),
		multiDownloader:  NewMultiDownloader(0, numWorkers),
	}
}

func (m *Manager) Probe(ctx context.Context, rawURL string) (*domain.DownloadPlan, error) {
	return m.prober.Probe(ctx, rawURL)
}

func (m *Manager) DownloadPlan(ctx context.Context, plan *domain.DownloadPlan, destDir string, onProgress domain.ProgressCallback) error {
	if destDir != "" {
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}
	}

	destPath := filepath.Join(destDir, plan.Filename)

	if plan.IsMultiPart && plan.TotalSize > 0 {
		return m.multiDownloader.Download(ctx, plan, destPath, onProgress)
	}

	return m.singleDownloader.Download(ctx, plan, destPath, onProgress)
}

func (m *Manager) Download(ctx context.Context, rawURL, destDir string, onProgress domain.ProgressCallback) error {
	plan, err := m.Probe(ctx, rawURL)
	if err != nil {
		return fmt.Errorf("probe failed: %w", err)
	}

	return m.DownloadPlan(ctx, plan, destDir, onProgress)
}
