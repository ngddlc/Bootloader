package storage

import (
	"database/sql"
	"fmt"
	"homecloud/internal/domain"

	_ "modernc.org/sqlite"
)

type Storage struct {
	db *sql.DB
}

func NewStorage(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	s := &Storage{db: db}

	if err := s.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to init database schema: %w", err)
	}

	return s, nil
}

func (s *Storage) initSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		url TEXT NOT NULL,
		filename TEXT NOT NULL,
		dest_path TEXT NOT NULL,
		total_size INTEGER NOT NULL,
		downloaded_bytes INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL,
		error TEXT DEFAULT ''
	);`
	_, err := s.db.Exec(query)
	return err
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) SaveTask(task *domain.DownloadTask) error {
	query := `
	INSERT INTO tasks (id, url, filename, dest_path, total_size, downloaded_bytes, status, error)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		downloaded_bytes = excluded.downloaded_bytes,
		status = excluded.status,
		error = excluded.error;`
	_, err := s.db.Exec(query,
		task.ID,
		task.URL,
		task.Filename,
		task.DestPath,
		task.TotalSize,
		task.Progress.DownloadedBytes,
		string(task.Status),
		task.Error,
	)
	if err != nil {
		return fmt.Errorf("failed to save task %s: %w", task.ID, err)
	}
	return nil
}

func (s *Storage) UpdateTaskStatus(id string, status domain.DownloadStatus, errDesc string) error {
	query := `UPDATE tasks SET status = ?, error = ? WHERE id = ?;`
	_, err := s.db.Exec(query, string(status), errDesc, id)
	if err != nil {
		return fmt.Errorf("failed to update status for task %s: %w", id, err)
	}
	return nil
}

func (s *Storage) GetAllTasks() ([]domain.DownloadTask, error) {
	query := `SELECT id, url, filename, dest_path, total_size, downloaded_bytes, status, error FROM tasks;`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []domain.DownloadTask

	for rows.Next() {
		var t domain.DownloadTask
		var statusStr string
		var downloadedBytes int64

		err := rows.Scan(
			&t.ID,
			&t.URL,
			&t.Filename,
			&t.DestPath,
			&t.TotalSize,
			&downloadedBytes,
			&statusStr,
			&t.Error,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan task row: %w", err)
		}

		t.Status = domain.DownloadStatus(statusStr)
		t.Progress.DownloadedBytes = downloadedBytes
		t.Progress.TotalBytes = t.TotalSize
		if t.TotalSize > 0 {
			t.Progress.Percent = (float64(downloadedBytes) / float64(t.TotalSize)) * 100
		}

		tasks = append(tasks, t)
	}
	return tasks, nil
}
