package models

import "time"

// PRDetails holds information about the pull request being reviewed.
type PRDetails struct {
	Owner    string
	Repo     string
	PRNumber int
	Title    string
	Branch   string
	URL      string
}

// Comment represents a single review comment to be posted.
type Comment struct {
	Body     string
	Path     string
	Position int // For GitHub and Gitea's review endpoint
	Line     int // For Gitea's fallback line comment endpoint
}

// ReviewComment represents the structured response from the LLM.
type ReviewComment struct {
	LineContent string `json:"line_content"`
	Message     string `json:"message"`
}

// DiffChunk represents a single block of changes in a diff.
type DiffChunk struct {
	FilePath     string
	CodeSnippet  string
	StartLineNew int
}

type ArchitectureReviewResponse struct {
	Score         int                 `json:"score"`
	FoundLayers   map[string][]string `json:"found_layers"`
	MissingLayers []string            `json:"missing_layers"`
	Feedback      string              `json:"feedback"`
	Comments      []Comment           `json:"comments"`
	NeedsComment  bool                `json:"needs_comment"`
}

type TestReviewResponse struct {
	MissingTests []string  `json:"missing_tests"`
	Comments     []Comment `json:"comments"`
	HasMissing   bool      `json:"has_missing"`
}

type Project struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"uniqueIndex;not null"`
	Status    string
	CreatedAt time.Time
}

type PullRequest struct {
	ID         uint   `gorm:"primaryKey"`
	ProjectID  uint   `gorm:"not null"`
	Title      string `gorm:"not null"`
	Branch     string `gorm:"not null"`
	Reviewer   string
	Status     string
	ReviewedAt time.Time
	PrURL      string
	Project    Project `gorm:"foreignKey:ProjectID"`
}

type ReviewStats struct {
	ID           uint `gorm:"primaryKey"`
	ProjectID    uint `gorm:"uniqueIndex;not null"`
	SuccessCount int
	FailedCount  int
	TotalCount   int
	UpdatedAt    time.Time
	Project      Project `gorm:"foreignKey:ProjectID"`
}

// PRComment is the new GORM model for the pr_comments table.
type PRComment struct {
	ID          uint   `gorm:"primaryKey"`
	PrID        uint   `gorm:"column:pr_id;not null"` // Explicitly map column name
	FilePath    string `gorm:"size:512;not null"`
	LineNumber  int
	CommentText string `gorm:"not null"`
	CommentType string `gorm:"size:50"`
	Severity    string `gorm:"size:20"`
	CreatedAt   time.Time
	Resolved    bool
}
