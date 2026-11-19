package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"homecloud/internal/domain"
	"homecloud/internal/downloader"
)

func main() {
	ctx := context.Background()

	mgr := downloader.NewManager(10*time.Second, 4)

	testURL := "https://go.dev/dl/go1.22.0.src.tar.gz"

	destDir := "./download"

	fmt.Println("Запуск загрузки через HomeCloud Manager")

	onProgress := func(p domain.Progress) {
		fmt.Printf("\r[HomeCloud CLI] Прогресс: %.1f%% (%.2f / %.2f МБ) | Скорость: %.2f МБ/с",
			p.Percent,
			float64(p.DownloadedBytes)/(1024*1024),
			float64(p.TotalBytes)/(1024*1024),
			p.Speed/(1024*1024),
		)
	}

	err := mgr.Download(ctx, testURL, destDir, onProgress)
	if err != nil {
		log.Fatalf("Ошибка разведки: %v", err)
	}

	fmt.Println("\nФайл успешно сохранён в папку:", destDir)
}
