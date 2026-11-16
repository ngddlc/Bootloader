package domain

type DownloadStatus string

const (
	StatusIdle        DownloadStatus = "IDLE"
	StatusProming     DownloadStatus = "PROBING"
	StatusDownloading DownloadStatus = "DOWNLOADING"
	StatusPaused      DownloadStatus = "PAUSED"
	StatusCompleted   DownloadStatus = "COMPLETED"
	StatusFailed      DownloadStatus = "FAILED"
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
