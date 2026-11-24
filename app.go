package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"homecloud/internal/domain"
	"homecloud/internal/downloader"
	"homecloud/internal/service"
	"homecloud/internal/storage"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context
	tm  *service.TaskManager
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Инициализируем хранилище
	store, err := storage.NewStorage("homecloud.db")
	if err != nil {
		runtime.LogErrorf(ctx, "Failed to initialize storage: %v", err)
		return
	}

	// Инициализируем сетевой движок (8 параллельных потоков)
	dl := downloader.NewManager(10*time.Second, 8)

	// Инициализируем менеджер задач
	a.tm = service.NewTaskManager(dl, store, "./download")

	a.tm.SetOnTaskProgress(func(task *domain.DownloadTask) {
		runtime.EventsEmit(a.ctx, "task:progress", task)
	})
}

// StartDownload запускает скачивание (с поддержкой выбора папки и числа потоков)
func (a *App) StartDownload(rawURL, customDir string, numWorkers int) (*domain.DownloadTask, error) {
	if a.tm == nil {
		return nil, fmt.Errorf("task manager not initialized")
	}
	return a.tm.StartDownload(rawURL, customDir, numWorkers)
}

// SelectDownloadFolder открывает системное диалоговое окно выбора папки Windows
func (a *App) SelectDownloadFolder() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Выберите папку для сохранения файлов",
	})
}

// CancelDownload отменяет задачу по ID
func (a *App) CancelDownload(taskID string) error {
	if a.tm == nil {
		return fmt.Errorf("task manager not initialized")
	}
	return a.tm.Cancel(taskID)
}

// GetTasks возвращает историю всех задач
func (a *App) GetTasks() ([]domain.DownloadTask, error) {
	if a.tm == nil {
		return nil, fmt.Errorf("task manager not initialized")
	}
	return a.tm.GetTasks()
}

// DeleteTask удаляет запись о задаче из базы данных и опционально удаляет файл с диска
func (a *App) DeleteTask(taskID string, deleteFile bool) error {
	if a.tm == nil {
		return fmt.Errorf("task manager not initialized")
	}
	return a.tm.DeleteTask(taskID, deleteFile)
}

// OpenFileFolder открывает проводник с выделенным загруженным файлом (или саму папку, если файл был перемещён/удалён)
func (a *App) OpenFileFolder(filePath string) error {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	if _, err := os.Stat(absPath); err == nil {
		return exec.Command("explorer.exe", "/select,", absPath).Start()
	}

	dir := filepath.Dir(absPath)
	if _, err := os.Stat(dir); err == nil {
		return exec.Command("explorer.exe", dir).Start()
	}

	return fmt.Errorf("path not found: %s", filePath)
}
