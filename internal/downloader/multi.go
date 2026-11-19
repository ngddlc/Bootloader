package downloader

import (
	"context"
	"fmt"
	"homecloud/internal/domain"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type MultiDownloader struct {
	client     *http.Client
	numWorkers int
}

func NewMultiDownloader(timeout time.Duration, numWorker int) *MultiDownloader {
	if numWorker <= 0 {
		numWorker = 4
	}

	return &MultiDownloader{
		client: &http.Client{
			Timeout: timeout,
		},
		numWorkers: numWorker,
	}
}

func (d *MultiDownloader) Download(ctx context.Context, plan *domain.DownloadPlan, destPath string, onProgress domain.ProgressCallback) error {
	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	err = file.Truncate(plan.TotalSize)
	if err != nil {
		return fmt.Errorf("failed to allocate disk space: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	chuncSize := plan.TotalSize / int64(d.numWorkers)

	var wg sync.WaitGroup
	var downloadErr error
	var errOnce sync.Once

	var downloaded atomic.Int64

	progressTicker := time.NewTicker(300 * time.Millisecond)
	defer progressTicker.Stop()

	progressDone := make(chan struct{})

	go func() {
		var prevDownloaded int64
		prevTime := time.Now()

		for {
			select {
			case <-progressDone:
				return
			case now := <-progressTicker.C:
				current := downloaded.Load()
				deltaBytes := current - prevDownloaded
				deltaTime := now.Sub(prevTime).Seconds()

				var speed float64
				if deltaTime > 0 {
					speed = float64(deltaBytes) / deltaTime
				}

				var percent float64
				if plan.TotalSize > 0 {
					percent = (float64(current) / float64(plan.TotalSize)) * 100
				}

				if onProgress != nil {
					onProgress(domain.Progress{
						DownloadedBytes: current,
						TotalBytes:      plan.TotalSize,
						Percent:         percent,
						Speed:           speed,
					})
				}

				prevDownloaded = current
				prevTime = now
			}
		}
	}()

	for i := 0; i < d.numWorkers; i++ {
		start := int64(i) * chuncSize
		end := start + chuncSize - 1

		if i == d.numWorkers-1 {
			end = plan.TotalSize - 1
		}

		wg.Add(1)

		go func(chunkIndex int, chunkStart, chunkEnd int64) {
			defer wg.Done()

			err := d.downloadChunc(ctx, plan.URL, chunkStart, chunkEnd, file, &downloaded)
			if err != nil {
				errOnce.Do(func() {
					downloadErr = fmt.Errorf("chunk %d [%d-%d] failed: %w", chunkIndex, chunkStart, chunkEnd, err)
					cancel()
				})
			}
		}(i, start, end)
	}
	wg.Wait()
	close(progressDone)

	fmt.Printf("\r[HomeCloud] Прогресс: 100.0%% (%.2f / %.2f МБ) | Скачивание завершено!      \n",
		float64(plan.TotalSize)/(1024*1024),
		float64(plan.TotalSize)/(1024*1024),
	)

	if downloadErr == nil && onProgress != nil {
		onProgress(domain.Progress{
			DownloadedBytes: plan.TotalSize,
			TotalBytes:      plan.TotalSize,
			Percent:         100.0,
			Speed:           0,
		})
	}

	if downloadErr != nil {
		return downloadErr
	}

	return nil
}

func (d *MultiDownloader) downloadChunc(ctx context.Context, rawURL string, start, end int64, file *os.File, counter *atomic.Int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("create chunk request failed: %w", err)
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	req.Header.Set("User-Agent", "HomeCloud-Downloader/1.0")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute chunk request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("unexpected status code: %d (expected 206 Partial Content)", resp.StatusCode)
	}

	writer := io.NewOffsetWriter(file, start)

	reader := newProgressReader(resp.Body, counter)

	_, err = io.Copy(writer, reader)
	if err != nil {
		return fmt.Errorf("failed to copy chunk data to file: %w", err)
	}

	return nil
}
