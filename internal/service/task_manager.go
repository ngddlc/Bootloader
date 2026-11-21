package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"homecloud/internal/domain"
	"homecloud/internal/downloader"
	"homecloud/internal/storage"
	"os"
	"sync"
)

type TaskManager struct {
	downloader  *downloader.Manager
	storage     *storage.Storage
	downloadDir string
	cancels     map[string]context.CancelFunc
	mu          sync.RWMutex
}

func NewTaskManager(dl *downloader.Manager, store *storage.Storage, downloadDir string) *TaskManager {
	return &TaskManager{
		downloader:  dl,
		storage:     store,
		downloadDir: downloadDir,
		cancels:     make(map[string]context.CancelFunc),
	}
}

func (tm *TaskManager) StartDownload(rawURL string) (*domain.DownloadTask, error) {
	taskID := generateID()

	task := &domain.DownloadTask{
		ID:       taskID,
		URL:      rawURL,
		Filename: "probing...", // Настоящее имя определит разведчик
		DestPath: tm.downloadDir,
		Status:   domain.StatusDownloading,
	}

	if err := tm.storage.SaveTask(task); err != nil {
		return nil, fmt.Errorf("failed to save initial task to db: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	tm.mu.Lock()
	tm.cancels[taskID] = cancel
	tm.mu.Unlock()

	go tm.runDownload(ctx, task)

	return task, nil
}

func (tm *TaskManager) runDownload(ctx context.Context, task *domain.DownloadTask) {
	defer func() {
		tm.mu.Lock()
		delete(tm.cancels, task.ID)
		tm.mu.Unlock()
	}()

	onProgress := func(p domain.Progress) {
		task.Progress = p
	}

	err := tm.downloader.Download(ctx, task.URL, tm.downloadDir, onProgress)
	if err != nil {
		if ctx.Err() == context.Canceled {
			task.Status = domain.StatusCanceled
			_ = tm.storage.UpdateTaskStatus(task.ID, domain.StatusCanceled, "Download canceled by user")

			if task.DestPath != "" {
				_ = os.Remove(task.DestPath)
			}
			return
		}

		task.Status = domain.StatusFailed
		task.Error = err.Error()
		_ = tm.storage.UpdateTaskStatus(task.ID, domain.StatusFailed, err.Error())
		return
	}

	task.Status = domain.StatusCompleted
	_ = tm.storage.UpdateTaskStatus(task.ID, domain.StatusCompleted, "")
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
