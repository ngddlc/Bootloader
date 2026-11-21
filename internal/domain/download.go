package domain

type DownloadStatus string

const (
	StatusIdle        DownloadStatus = "IDLE"
	StatusProming     DownloadStatus = "PROBING"
	StatusDownloading DownloadStatus = "DOWNLOADING"
	StatusPaused      DownloadStatus = "PAUSED"
	StatusCompleted   DownloadStatus = "COMPLETED"
	StatusFailed      DownloadStatus = "FAILED"
	StatusCanceled    DownloadStatus = "CANCELED"
)

type DownloadPlan struct {
	URL         string
	Filename    string
	TotalSize   int64
	IsMultiPart bool
	ContentType string
}

type Progress struct {
	DownloadedBytes int64
	TotalBytes      int64
	Percent         float64
	Speed           float64
}

type ProgressCallback func(p Progress)

type DownloadTask struct {
	ID        string         `json:"id"`
	URL       string         `json:"url"`
	Filename  string         `json:"filename"`
	DestPath  string         `json:"dest_path"`
	TotalSize int64          `json:"total_size"`
	Progress  Progress       `json:"progress"`
	Status    DownloadStatus `json:"status"`
	Error     string         `json:"error"`
}
