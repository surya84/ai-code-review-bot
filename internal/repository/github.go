package repository

import (
	"code-reviewer-bot/internal/models"
	"context"
	"fmt"

	"github.com/google/go-github/v62/github"
	"golang.org/x/oauth2"
)

// GitHubRepository implements the VcsRepository interface for GitHub.
type GitHubRepository struct {
	client *github.Client
}

// NewGitHubRepository creates a new client for interacting with the GitHub API.
func NewGitHubRepository(ctx context.Context, token string) *GitHubRepository {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	return &GitHubRepository{client: client}
}

func (g *GitHubRepository) GetPRDiff(ctx context.Context, owner, repo string, prNumber int) (string, error) {
	diff, _, err := g.client.PullRequests.GetRaw(ctx, owner, repo, prNumber, github.RawOptions{Type: github.Diff})
	if err != nil {
		return "", fmt.Errorf("failed to get PR diff from GitHub: %w", err)
	}
	return diff, nil
}

func (g *GitHubRepository) GetPRCommitID(ctx context.Context, owner, repo string, prNumber int) (string, error) {
	pr, _, err := g.client.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		return "", fmt.Errorf("failed to get pull request details: %w", err)
	}
	return pr.GetHead().GetSHA(), nil
}

func (g *GitHubRepository) PostReview(ctx context.Context, owner, repo string, prNumber int, comments []*models.Comment, commitID string) error {
	var reviewComments []*github.DraftReviewComment
	for _, c := range comments {
		reviewComments = append(reviewComments, &github.DraftReviewComment{
			Path:     &c.Path,
			Position: &c.Position,
			Body:     &c.Body,
		})
	}
	reviewRequest := &github.PullRequestReviewRequest{
		CommitID: &commitID,
		Event:    github.String("COMMENT"),
		Comments: reviewComments,
	}
	_, _, err := g.client.PullRequests.CreateReview(ctx, owner, repo, prNumber, reviewRequest)
	return err
}

func (g *GitHubRepository) PostGeneralComment(ctx context.Context, owner, repo string, prNumber int, body string) error {
	issueComment := &github.IssueComment{Body: &body}
	_, _, err := g.client.Issues.CreateComment(ctx, owner, repo, prNumber, issueComment)
	return err
}
