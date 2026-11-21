package main

import (
	"fmt"
	"log"
	"time"

	"homecloud/internal/downloader"
	"homecloud/internal/service"
	"homecloud/internal/storage"
)

func main() {
	// 1. Инициализируем базу данных SQLite (файл homecloud.db создастся сам)
	store, err := storage.NewStorage("homecloud.db")
	if err != nil {
		log.Fatalf("Ошибка базы данных: %v", err)
	}
	defer store.Close()

	// 2. Смотрим, что уже есть в базе (история)
	tasks, err := store.GetAllTasks()
	if err != nil {
		log.Fatalf("Ошибка чтения истории: %v", err)
	}
	fmt.Printf("История в БД: найдено задач: %d\n", len(tasks))
	for _, t := range tasks {
		fmt.Printf("- [%s] %s (Статус: %s)\n", t.ID, t.Filename, t.Status)
	}

	// 3. Создаём сетевой движок и наш Менеджер задач
	dl := downloader.NewManager(10*time.Second, 4)
	tm := service.NewTaskManager(dl, store, "./download")

	// 4. Запускаем скачивание новой задачи
	testURL := "https://go.dev/dl/go1.22.0.src.tar.gz"
	fmt.Println("Запускаем задачу через TaskManager...")

	task, err := tm.StartDownload(testURL)
	if err != nil {
		log.Fatalf("Ошибка старта: %v", err)
	}

	fmt.Printf("Задача успешно запущена в фоне! ID задачи: %s\n", task.ID)
	fmt.Println("Ожидаем завершения фоновой загрузки...")

	// 5. Временный цикл ожидания для консоли (в Wails приложение будет просто открыто)
	for {
		time.Sleep(500 * time.Millisecond)

		// Проверяем статус в памяти
		if task.Status != "DOWNLOADING" {
			fmt.Printf("\nЗагрузка завершилась со статусом: %s!\n", task.Status)
			if task.Error != "" {
				fmt.Printf("Ошибка: %s\n", task.Error)
			}
			break
		}
	}
}
