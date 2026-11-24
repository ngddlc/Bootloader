package main

import (
	"fmt"
	"log"
	"time"

	"homecloud/internal/domain"
	"homecloud/internal/downloader"
	"homecloud/internal/service"
	"homecloud/internal/storage"
)

func main() {
	store, err := storage.NewStorage("homecloud.db")
	if err != nil {
		log.Fatalf("Ошибка базы данных: %v", err)
	}
	defer store.Close()

	// Вывод существующей истории задач
	tasks, err := store.GetAllTasks()
	if err != nil {
		log.Fatalf("Ошибка чтения истории: %v", err)
	}
	fmt.Printf("История в БД: найдено задач: %d\n", len(tasks))
	for _, t := range tasks {
		dupInfo := ""
		if t.DuplicateOf != "" {
			dupInfo = fmt.Sprintf(" | ДУБЛИКАТ: %s", t.DuplicateOf)
		}
		shaInfo := "нет хеша"
		if t.Checksum != "" {
			if len(t.Checksum) > 12 {
				shaInfo = t.Checksum[:12] + "..."
			} else {
				shaInfo = t.Checksum
			}
		}
		fmt.Printf("- [%s] %s | Кат: %s | Потоков: %d | SHA: %s%s (Статус: %s)\n",
			t.ID, t.Filename, t.Category, t.Threads, shaInfo, dupInfo, t.Status)
	}

	dl := downloader.NewManager(10*time.Second, 4)
	tm := service.NewTaskManager(dl, store, "./download")

	// Тестовый запуск загрузки через CLI
	testURL := "https://go.dev/dl/go1.22.0.src.tar.gz"
	fmt.Println("Запускаем задачу через TaskManager...")

	task, err := tm.StartDownload(testURL, "", 4)
	if err != nil {
		log.Fatalf("Ошибка старта: %v", err)
	}

	fmt.Printf("Задача успешно запущена в фоне! ID задачи: %s\n", task.ID)
	fmt.Println("Ожидаем завершения фоновой загрузки...")

	for {
		time.Sleep(500 * time.Millisecond)

		if task.Status != domain.StatusDownloading {
			fmt.Printf("\nЗагрузка завершилась со статусом: %s!\n", task.Status)
			if task.Error != "" {
				fmt.Printf("Ошибка: %s\n", task.Error)
			}
			break
		}
	}
}
