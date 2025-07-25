package repository

import (
	"code-reviewer-bot/internal/models"
	"context"
)

// VcsRepository defines the interface for data access operations related to a VCS.
//go:generate mockgen -source=adapter.go -destination=repository_mock.go -package=repository
type VcsRepository interface {
	GetPRDiff(ctx context.Context, owner, repo string, prNumber int) (string, error)
	GetPRCommitID(ctx context.Context, owner, repo string, prNumber int) (string, error)
	PostReview(ctx context.Context, owner, repo string, prNumber int, comments []*models.Comment, commitID string) error
	PostGeneralComment(ctx context.Context, owner, repo string, prNumber int, body string) error
}
