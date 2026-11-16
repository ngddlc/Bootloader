package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"homecloud/internal/downloader"
)

func main() {
	ctx := context.Background()

	mgr := downloader.NewManager(10*time.Second, 4)

	testURL := "https://go.dev/dl/go1.22.0.src.tar.gz"

	destDir := "./download"

	fmt.Println("Отправляем запрос разведки...")

	err := mgr.Download(ctx, testURL, destDir)
	if err != nil {
		log.Fatalf("Ошибка разведки: %v", err)
	}

	fmt.Println("Файл успешно сохранён в папку:", destDir)
}
