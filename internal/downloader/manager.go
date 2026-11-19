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

func (m *Manager) Download(ctx context.Context, rawURL, destDir string, onProgress domain.ProgressCallback) error {
	if destDir != "" {
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}
	}

	fmt.Println("[Manager] Разведка URL...")
	plan, err := m.prober.Probe(ctx, rawURL)
	if err != nil {
		return fmt.Errorf("probe failed: %w", err)
	}

	destPath := filepath.Join(destDir, plan.Filename)
	fmt.Printf("[Manager] Файл: %s | Размер: %.2f МБ\n",
		plan.Filename,
		float64(plan.TotalSize)/(1024*1024))

	//Принимает решение о типе загрузки
	if plan.IsMultiPart && plan.TotalSize > 0 {
		fmt.Printf("[Manager] Сервер поддерживает Range. Запуск многопоточной загрузки (%d потоков)...\n", m.multiDownloader.numWorkers)
		return m.multiDownloader.Download(ctx, plan, destPath, onProgress)
	}
	fmt.Println("[Manager] Сервер НЕ поддерживает Range или размер неизвестен. Откат на однопоточную загрузку...")
	return m.singleDownloader.Download(ctx, plan, destPath, onProgress)
}
