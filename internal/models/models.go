// Package models defines the SQLite schema for all persisted entities.
package models

import "time"

// Job states. Persisted so workers can recover after a crash.
const (
	JobQueued      = "queued"
	JobRunning     = "running"
	JobPaused      = "paused"
	JobCancelling  = "cancelling"
	JobCancelled   = "cancelled"
	JobCompleted   = "completed"
	JobFailed      = "failed"
	JobInterrupted = "interrupted"
)

// RepositoryStatus values.
const (
	RepoScanning = "scanning"
	RepoReady    = "ready"
	RepoFailed   = "failed"
)

// Repository is a scanned git repository plus its rolled-up summary stats.
type Repository struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	Name          string     `gorm:"index" json:"name"`
	Path          string     `gorm:"uniqueIndex" json:"path"`
	Status        string     `json:"status"`
	GitRemote     string     `json:"gitRemote"`
	HeadCommit    string     `json:"headCommit"`
	DefaultBranch string     `json:"defaultBranch"`
	LastScannedAt *time.Time `json:"lastScannedAt"`

	FileCount        int   `json:"fileCount"`
	TotalLOC         int   `json:"totalLOC"`
	TotalCode        int   `json:"totalCode"`
	TotalComments    int   `json:"totalComments"`
	TotalBlank       int   `json:"totalBlank"`
	TotalSize        int64 `json:"totalSize"`
	CommitCount      int   `json:"commitCount"`
	ContributorCount int   `json:"contributorCount"`
	DependencyCount  int   `json:"dependencyCount"`
	DupGroupCount    int   `json:"dupGroupCount"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// File is one scanned source file with its measured properties.
type File struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	RepoID        uint       `gorm:"index:idx_file_repo_path,unique;index:idx_file_repo_folder;index:idx_file_repo_lang;index:idx_file_repo_ext;index:idx_file_repo_author" json:"repoId"`
	Path          string     `json:"path"`
	Name          string     `json:"name"`
	Folder        string     `json:"folder"`
	Extension     string     `json:"extension"`
	Language      string     `json:"language"`
	Size          int64      `json:"size"`
	LinesTotal    int        `json:"linesTotal"`
	LinesCode     int        `json:"linesCode"`
	LinesComment  int        `json:"linesComment"`
	LinesBlank    int        `json:"linesBlank"`
	Complexity    int        `json:"complexity"`
	Imports       int        `json:"imports"`
	Exports       int        `json:"exports"`
	Author        string     `json:"author"`
	FirstCommitAt *time.Time `json:"firstCommitAt"`
	LastCommitAt  *time.Time `json:"lastCommitAt"`
	Commits       int        `json:"commits"`
}

// Commit is one commit in the repository history.
type Commit struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	RepoID       uint      `gorm:"index:idx_commit_repo_hash,unique;index:idx_commit_repo_author;index:idx_commit_repo_date" json:"repoId"`
	Hash         string    `json:"hash"`
	Author       string    `json:"author"`
	Email        string    `json:"email"`
	Date         time.Time `json:"date"`
	Message      string    `json:"message"`
	FilesChanged int       `json:"filesChanged"`
	Insertions   int       `json:"insertions"`
	Deletions    int       `json:"deletions"`
	IsMerge      bool      `json:"isMerge"`
}

// Branch is a git branch pointing at a commit hash.
type Branch struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	RepoID     uint   `gorm:"index" json:"repoId"`
	Name       string `json:"name"`
	CommitHash string `json:"commitHash"`
	IsCurrent  bool   `json:"isCurrent"`
}

// Tag is a git tag pointing at a commit hash.
type Tag struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	RepoID     uint   `gorm:"index" json:"repoId"`
	Name       string `json:"name"`
	CommitHash string `json:"commitHash"`
}

// Contributor is a rolled-up git author.
type Contributor struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	RepoID        uint      `gorm:"index:idx_contrib_repo_email,unique" json:"repoId"`
	Name          string    `json:"name"`
	Email         string    `json:"email"`
	Commits       int       `json:"commits"`
	Insertions    int       `json:"insertions"`
	Deletions     int       `json:"deletions"`
	FirstCommitAt time.Time `json:"firstCommitAt"`
	LastCommitAt  time.Time `json:"lastCommitAt"`
}

// FileOwnership records the primary author share of a file.
type FileOwnership struct {
	ID      uint    `gorm:"primaryKey" json:"id"`
	RepoID  uint    `gorm:"index" json:"repoId"`
	Path    string  `json:"path"`
	Author  string  `json:"author"`
	Commits int     `json:"commits"`
	Share   float64 `json:"share"`
}

// Dependency is one declared package from a manifest file.
type Dependency struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	RepoID   uint   `gorm:"index:idx_dep_repo_file" json:"repoId"`
	FilePath string `json:"filePath"`
	Manager  string `json:"manager"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Scope    string `json:"scope"`
}

// ImportEdge is a resolved import relationship between two files.
type ImportEdge struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	RepoID     uint   `gorm:"index:idx_import_repo_from;index:idx_import_repo_to" json:"repoId"`
	FromFile   string `json:"fromFile"`
	ToFile     string `json:"toFile"`
	ImportType string `json:"importType"`
	Resolved   bool   `json:"resolved"`
}

// DuplicateGroup is a set of similar code blocks across files.
type DuplicateGroup struct {
	ID         uint    `gorm:"primaryKey" json:"id"`
	RepoID     uint    `gorm:"index" json:"repoId"`
	Fragment   string  `json:"fragment"`
	Lines      int     `json:"lines"`
	Similarity float64 `json:"similarity"`
	FileCount  int     `json:"fileCount"`
}

// DuplicateBlock is one occurrence of a duplicated fragment.
type DuplicateBlock struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	GroupID   uint   `gorm:"index" json:"groupId"`
	RepoID    uint   `gorm:"index" json:"repoId"`
	FilePath  string `json:"filePath"`
	StartLine int    `json:"startLine"`
	EndLine   int    `json:"endLine"`
}

// Job is a background work item. State is persisted for crash recovery.
type Job struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	RepoID     uint       `gorm:"index" json:"repoId"`
	Kind       string     `json:"kind"`
	Status     string     `gorm:"index" json:"status"`
	Progress   float64    `json:"progress"`
	Current    int        `json:"current"`
	Total      int        `json:"total"`
	Message    string     `json:"message"`
	Error      string     `json:"error"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	StartedAt  *time.Time `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt"`
}

// Setting is a persisted key/value preference.
type Setting struct {
	Key   string `gorm:"primaryKey"`
	Value string
}
