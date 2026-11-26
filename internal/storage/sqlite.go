package storage

import (
	"database/sql"
	"fmt"
	"homecloud/internal/crypto"
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
	categoriesQuery := `
	CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		folder_name TEXT NOT NULL,
		icon TEXT NOT NULL,
		extensions TEXT NOT NULL
	);`
	if _, err := s.db.Exec(categoriesQuery); err != nil {
		return fmt.Errorf("failed to create categories table: %w", err)
	}

	categoryInit := `
	INSERT OR IGNORE INTO categories (name, folder_name, icon, extensions) VALUES
	('Архивы', 'Архивы', '📦', '.zip,.tar.gz,.tar,.gz,.rar,.7z,.bz2,.xz,.tgz,.iso'),
	('Документы', 'Документы', '📄', '.pdf,.docx,.doc,.xlsx,.xls,.pptx,.ppt,.txt,.rtf,.odt,.csv,.md'),
	('Медиа', 'Медиа', '🎬', '.mp4,.mkv,.avi,.mov,.webm,.flv,.mp3,.wav,.flac,.ogg,.jpg,.jpeg,.png,.gif,.webp,.svg'),
	('Программы', 'Программы', '⚙️', '.exe,.msi,.apk,.dmg,.deb,.rpm,.appimage'),
	('Другое', 'Другое', '📁', '');`
	if _, err := s.db.Exec(categoryInit); err != nil {
		return fmt.Errorf("failed to populate categories: %w", err)
	}

	tasksQuery := `
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		url TEXT NOT NULL,
		filename TEXT NOT NULL,
		dest_path TEXT NOT NULL,
		category_id INTEGER DEFAULT 5 REFERENCES categories(id),
		checksum TEXT NOT NULL DEFAULT '',
		threads INTEGER DEFAULT 8,
		is_multi_part BOOLEAN DEFAULT 0,
		total_size INTEGER NOT NULL,
		downloaded_bytes INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL,
		error TEXT DEFAULT ''
	);`
	if _, err := s.db.Exec(tasksQuery); err != nil {
		return fmt.Errorf("failed to create tasks table: %w", err)
	}

	// Миграции на случай, если база уже существовала
	_, _ = s.db.Exec(`ALTER TABLE tasks ADD COLUMN category_id INTEGER DEFAULT 5 REFERENCES categories(id);`)
	_, _ = s.db.Exec(`ALTER TABLE tasks ADD COLUMN checksum TEXT NOT NULL DEFAULT '';`)
	_, _ = s.db.Exec(`ALTER TABLE tasks ADD COLUMN threads INTEGER DEFAULT 8;`)
	_, _ = s.db.Exec(`ALTER TABLE tasks ADD COLUMN is_multi_part BOOLEAN DEFAULT 0;`)
	_, _ = s.db.Exec(`ALTER TABLE tasks DROP COLUMN category;`)

	return nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) SaveTask(task *domain.DownloadTask) error {
	catID := task.CategoryID
	if catID <= 0 {
		catID = 5 // По умолчанию "Другое"
	}
	threads := task.Threads
	if threads <= 0 {
		threads = 8
	}

	isMultiPartInt := 0
	if task.IsMultiPart {
		isMultiPartInt = 1
	}

	query := `
	INSERT INTO tasks (id, url, filename, dest_path, category_id, checksum, threads, is_multi_part, total_size, downloaded_bytes, status, error)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		filename = excluded.filename,
		dest_path = excluded.dest_path,
		category_id = excluded.category_id,
		checksum = excluded.checksum,
		threads = excluded.threads,
		is_multi_part = excluded.is_multi_part,
		total_size = excluded.total_size,
		downloaded_bytes = excluded.downloaded_bytes,
		status = excluded.status,
		error = excluded.error;`
	_, err := s.db.Exec(query,
		task.ID,
		task.URL,
		task.Filename,
		task.DestPath,
		catID,
		task.Checksum,
		threads,
		isMultiPartInt,
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

func (s *Storage) UpdateTaskChecksum(id, checksum string) error {
	query := `UPDATE tasks SET checksum = ? WHERE id = ?;`
	_, err := s.db.Exec(query, checksum, id)
	return err
}

func (s *Storage) UpdateTaskStatus(id string, status domain.DownloadStatus, errDesc string) error {
	query := `UPDATE tasks SET status = ?, error = ? WHERE id = ?;`
	_, err := s.db.Exec(query, string(status), errDesc, id)
	if err != nil {
		return fmt.Errorf("failed to update status for task %s: %w", id, err)
	}
	return nil
}

func (s *Storage) GetTaskDestPath(id string) (string, error) {
	query := `SELECT dest_path FROM tasks WHERE id = ?;`
	row := s.db.QueryRow(query, id)
	var destPath string
	err := row.Scan(&destPath)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return destPath, nil
}

func (s *Storage) DeleteTask(id string) error {
	query := `DELETE FROM tasks WHERE id = ?;`
	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task %s: %w", id, err)
	}
	return nil
}

// FindDuplicate ищет среди завершённых задач файл с такой же контрольной суммой SHA-256
func (s *Storage) FindDuplicate(checksum, excludeTaskID string) (string, error) {
	if checksum == "" {
		return "", nil
	}
	query := `SELECT filename FROM tasks WHERE checksum = ? AND id != ? AND status = 'COMPLETED' LIMIT 1;`
	row := s.db.QueryRow(query, checksum, excludeTaskID)
	var originalFilename string
	err := row.Scan(&originalFilename)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return originalFilename, nil
}

func (s *Storage) GetAllTasks() ([]domain.DownloadTask, error) {
	query := `
	SELECT 
		t.id, t.url, t.filename, t.dest_path, 
		COALESCE(c.id, 5) AS category_id,
		COALESCE(c.name, 'Другое') AS category_name,
		t.checksum, COALESCE(t.threads, 8) AS threads,
		COALESCE(t.is_multi_part, 0) AS is_multi_part,
		t.total_size, t.downloaded_bytes, t.status, t.error 
	FROM tasks t
	LEFT JOIN categories c ON t.category_id = c.id
	ORDER BY t.rowid ASC;`
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
		var isMultiPartInt int

		err := rows.Scan(
			&t.ID,
			&t.URL,
			&t.Filename,
			&t.DestPath,
			&t.CategoryID,
			&t.Category,
			&t.Checksum,
			&t.Threads,
			&isMultiPartInt,
			&t.TotalSize,
			&downloadedBytes,
			&statusStr,
			&t.Error,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan task row: %w", err)
		}

		t.IsMultiPart = isMultiPartInt == 1

		t.Status = domain.DownloadStatus(statusStr)
		t.Progress.DownloadedBytes = downloadedBytes
		t.Progress.TotalBytes = t.TotalSize
		if t.TotalSize > 0 {
			t.Progress.Percent = (float64(downloadedBytes) / float64(t.TotalSize)) * 100
		}

		tasks = append(tasks, t)
	}

	// Дедупликация и автоматический подсчет хеша для ранее скачанных файлов
	seenChecksums := make(map[string]string)
	for i := range tasks {
		// Если задача завершена, но контрольная сумма пустая, вычисляем с диска
		if tasks[i].Status == domain.StatusCompleted && tasks[i].Checksum == "" && tasks[i].DestPath != "" {
			if hash, err := crypto.CalculateSHA256(tasks[i].DestPath); err == nil && hash != "" {
				tasks[i].Checksum = hash
				_ = s.UpdateTaskChecksum(tasks[i].ID, hash)
			}
		}

		if tasks[i].Status == domain.StatusCompleted && tasks[i].Checksum != "" {
			if orig, found := seenChecksums[tasks[i].Checksum]; found {
				tasks[i].DuplicateOf = orig
			} else {
				seenChecksums[tasks[i].Checksum] = tasks[i].Filename
			}
		}
	}

	return tasks, nil
}
