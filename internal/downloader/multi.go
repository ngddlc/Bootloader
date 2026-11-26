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

	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    true,
	}

	return &MultiDownloader{
		client: &http.Client{
			Transport: tr,
			Timeout:   timeout,
		},
		numWorkers: numWorker,
	}
}

func (d *MultiDownloader) Download(ctx context.Context, plan *domain.DownloadPlan, destPath string, workers int, onProgress domain.ProgressCallback) error {
	if workers <= 0 {
		workers = d.numWorkers
	}

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

	chunkSize := plan.TotalSize / int64(workers)

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
		var smoothedSpeed float64

		for {
			select {
			case <-progressDone:
				return
			case now := <-progressTicker.C:
				current := downloaded.Load()
				deltaBytes := current - prevDownloaded
				deltaTime := now.Sub(prevTime).Seconds()

				if deltaTime > 0 && deltaBytes >= 0 {
					instantSpeed := float64(deltaBytes) / deltaTime
					if smoothedSpeed == 0 {
						smoothedSpeed = instantSpeed
					} else {
						// Экспоненциальное сглаживание скорости для плавных показаний в GUI
						smoothedSpeed = smoothedSpeed*0.65 + instantSpeed*0.35
					}
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
						Speed:           smoothedSpeed,
					})
				}

				prevDownloaded = current
				prevTime = now
			}
		}
	}()

	for i := 0; i < workers; i++ {
		start := int64(i) * chunkSize
		end := start + chunkSize - 1

		if i == workers-1 {
			end = plan.TotalSize - 1
		}

		wg.Add(1)

		go func(chunkIndex int, chunkStart, chunkEnd int64) {
			defer wg.Done()

			err := d.downloadChunk(ctx, plan.URL, chunkStart, chunkEnd, file, &downloaded)
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

	// Если произошла ошибка или пользователь нажал "Отмена"
	if downloadErr != nil {
		return downloadErr
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Уведомляем об успешном 100% завершении
	if onProgress != nil {
		onProgress(domain.Progress{
			DownloadedBytes: plan.TotalSize,
			TotalBytes:      plan.TotalSize,
			Percent:         100.0,
			Speed:           0,
		})
	}

	return nil
}

func (d *MultiDownloader) downloadChunk(ctx context.Context, rawURL string, start, end int64, file *os.File, counter *atomic.Int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("create chunk request failed: %w", err)
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Encoding", "identity")

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

	// Буфер 128 КБ снижает число системных вызовов WriteAt в Windows
	buf := make([]byte, 128*1024)
	_, err = io.CopyBuffer(writer, reader, buf)
	if err != nil {
		return fmt.Errorf("failed to copy chunk data to file: %w", err)
	}

	return nil
}
