package common

// ServerStats server statistics
type ServerStats struct {
	Users            int   `json:"users"`
	Uploads          int   `json:"uploads"`
	AnonymousUploads int   `json:"anonymousUploads"`
	Files            int   `json:"files"`
	TotalSize        int64 `json:"totalSize"`
	AnonymousSize    int64 `json:"anonymousTotalSize"`
}

// UserStats user statistics
type UserStats struct {
	Uploads   int   `json:"uploads"`
	Files     int   `json:"files"`
	TotalSize int64 `json:"totalSize"`
}

// CleaningStats cleaning statistics
type CleaningStats struct {
	RemovedUploads      int
	DeletedFiles        int
	DeletedUploads      int
	OrphanFilesCleaned  int
	OrphanTokensCleaned int
}
