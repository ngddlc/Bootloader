package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"homecloud/internal/category"
	"homecloud/internal/crypto"
	"homecloud/internal/domain"
	"homecloud/internal/downloader"
	"homecloud/internal/storage"
	"os"
	"sync"
)

type TaskManager struct {
	downloader     *downloader.Manager
	storage        *storage.Storage
	downloadDir    string
	cancels        map[string]context.CancelFunc
	mu             sync.RWMutex
	onTaskProgress func(task *domain.DownloadTask)
}

func NewTaskManager(dl *downloader.Manager, store *storage.Storage, downloadDir string) *TaskManager {
	return &TaskManager{
		downloader:  dl,
		storage:     store,
		downloadDir: downloadDir,
		cancels:     make(map[string]context.CancelFunc),
	}
}

func (tm *TaskManager) SetOnTaskProgress(fn func(task *domain.DownloadTask)) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.onTaskProgress = fn
}

func (tm *TaskManager) StartDownload(rawURL, customDir string, numWorkers int) (*domain.DownloadTask, error) {
	if numWorkers <= 0 {
		numWorkers = 8
	}

	taskID := generateID()

	task := &domain.DownloadTask{
		ID:         taskID,
		URL:        rawURL,
		Filename:   "probing...", // Настоящее имя определит разведчик
		DestPath:   tm.downloadDir,
		CategoryID: 5,
		Category:   "Другое",
		Threads:    numWorkers,
		Status:     domain.StatusDownloading,
	}

	if err := tm.storage.SaveTask(task); err != nil {
		return nil, fmt.Errorf("failed to save initial task to db: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	tm.mu.Lock()
	tm.cancels[taskID] = cancel
	tm.mu.Unlock()

	go tm.runDownload(ctx, task, customDir, numWorkers)

	return task, nil
}

func (tm *TaskManager) runDownload(ctx context.Context, task *domain.DownloadTask, customDir string, numWorkers int) {
	defer func() {
		tm.mu.Lock()
		delete(tm.cancels, task.ID)
		tm.mu.Unlock()
	}()

	// Быстрая сетевая разведка
	plan, err := tm.downloader.Probe(ctx, task.URL)
	if err != nil {
		task.Status = domain.StatusFailed
		task.Error = err.Error()
		_ = tm.storage.UpdateTaskStatus(task.ID, domain.StatusFailed, err.Error())
		tm.notifyFinal(task)
		return
	}

	// Определение категории и итогового пути сохранения
	destPath, uniqueName, cat, err := category.ResolveDestination(tm.downloadDir, customDir, plan.Filename)
	if err != nil {
		task.Status = domain.StatusFailed
		task.Error = err.Error()
		_ = tm.storage.UpdateTaskStatus(task.ID, domain.StatusFailed, err.Error())
		tm.notifyFinal(task)
		return
	}

	// Обновление метаданных задачи перед началом загрузки
	task.Filename = uniqueName
	task.DestPath = destPath
	task.CategoryID = cat.ID
	task.Category = cat.Name
	task.TotalSize = plan.TotalSize
	task.IsMultiPart = plan.IsMultiPart && plan.TotalSize > 0
	if !task.IsMultiPart {
		task.Threads = 1
	}
	_ = tm.storage.SaveTask(task)
	tm.notifyFinal(task)

	onProgress := func(p domain.Progress) {
		task.Progress = p

		tm.mu.RLock()
		notify := tm.onTaskProgress
		tm.mu.RUnlock()
		if notify != nil {
			notify(task)
		}
	}

	// Запуск скачивания по плану с заданным числом потоков
	err = tm.downloader.DownloadPlan(ctx, plan, destPath, numWorkers, onProgress)
	if err != nil {
		if ctx.Err() == context.Canceled {
			task.Status = domain.StatusCanceled
			_ = tm.storage.UpdateTaskStatus(task.ID, domain.StatusCanceled, "Download canceled by user")

			if task.DestPath != "" {
				_ = os.Remove(task.DestPath)
			}
			tm.notifyFinal(task)
			return
		}

		task.Status = domain.StatusFailed
		task.Error = err.Error()
		_ = tm.storage.UpdateTaskStatus(task.ID, domain.StatusFailed, err.Error())
		tm.notifyFinal(task)
		return
	}

	task.Status = domain.StatusCompleted
	task.Progress.Percent = 100
	task.Progress.DownloadedBytes = task.TotalSize
	task.Progress.Speed = 0

	// Потоковый расчёт контрольной суммы SHA-256 и проверка дедупликации
	if checksum, err := crypto.CalculateSHA256(destPath); err == nil {
		task.Checksum = checksum
		if dup, err := tm.storage.FindDuplicate(checksum, task.ID); err == nil && dup != "" {
			task.DuplicateOf = dup
		}
	}

	_ = tm.storage.SaveTask(task)
	tm.notifyFinal(task)
}

func (tm *TaskManager) notifyFinal(task *domain.DownloadTask) {
	tm.mu.RLock()
	notify := tm.onTaskProgress
	tm.mu.RUnlock()
	if notify != nil {
		notify(task)
	}
}

func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (tm *TaskManager) Cancel(taskID string) error {
	tm.mu.RLock()
	cancel, exists := tm.cancels[taskID]
	tm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task %s is not active or already finished", taskID)
	}

	cancel()
	return nil
}

func (tm *TaskManager) GetTasks() ([]domain.DownloadTask, error) {
	return tm.storage.GetAllTasks()
}

func (tm *TaskManager) DeleteTask(taskID string, deleteFile bool) error {
	tm.mu.RLock()
	cancel, exists := tm.cancels[taskID]
	tm.mu.RUnlock()

	if exists {
		cancel()
	}

	if deleteFile {
		if destPath, err := tm.storage.GetTaskDestPath(taskID); err == nil && destPath != "" {
			_ = os.Remove(destPath)
		}
	}

	return tm.storage.DeleteTask(taskID)
}
