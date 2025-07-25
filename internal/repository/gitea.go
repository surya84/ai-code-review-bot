package repository

import (
	"context"
	"fmt"
	"log"
	"strings"

	"code-reviewer-bot/internal/models"

	"code.gitea.io/sdk/gitea"
)

// GiteaRepository implements the VcsRepository interface for Gitea.
type GiteaRepository struct {
	client *gitea.Client
}

// NewGiteaRepository creates a new client for interacting with the Gitea API.
func NewGiteaRepository(ctx context.Context, baseURL, token string) *GiteaRepository {
	c, err := gitea.NewClient(baseURL, gitea.SetToken(token))
	if err != nil {
		log.Fatalf("Failed to create Gitea client: %v", err)
	}
	return &GiteaRepository{client: c}
}

func (g *GiteaRepository) GetPRDiff(ctx context.Context, owner, repo string, prIndex int) (string, error) {
	diff, _, err := g.client.GetPullRequestDiff(owner, repo, int64(prIndex), gitea.PullRequestDiffOptions{})
	return string(diff), err
}

func (g *GiteaRepository) GetPRCommitID(ctx context.Context, owner, repo string, prIndex int) (string, error) {
	pr, _, err := g.client.GetPullRequest(owner, repo, int64(prIndex))
	if err != nil {
		return "", err
	}
	return pr.Head.Sha, nil
}

func (g *GiteaRepository) PostReview(ctx context.Context, owner, repo string, prIndex int, comments []*models.Comment, commitID string) error {
	var giteaComments []gitea.CreatePullReviewComment
	for _, c := range comments {
		giteaComments = append(giteaComments, gitea.CreatePullReviewComment{
			Path:       c.Path,
			Body:       c.Body,
			NewLineNum: int64(c.Line),
		})
	}
	opts := gitea.CreatePullReviewOptions{
		State:    gitea.ReviewStateComment,
		CommitID: commitID,
		Comments: giteaComments,
	}
	_, _, err := g.client.CreatePullReview(owner, repo, int64(prIndex), opts)
	if err != nil && strings.Contains(err.Error(), "404 Not Found") {
		log.Println("WARNING: Gitea instance may be too old to support batch reviews. Falling back to a summary comment.")
		var summary strings.Builder
		summary.WriteString("### AI Code Review Summary\n\n")
		for _, c := range comments {
			summary.WriteString(fmt.Sprintf("- **File `%s` (near position %d):** %s\n", c.Path, c.Position, c.Body))
		}
		return g.PostGeneralComment(ctx, owner, repo, prIndex, summary.String())
	}
	return err
}

func (g *GiteaRepository) PostGeneralComment(ctx context.Context, owner, repo string, prIndex int, body string) error {
	opts := gitea.CreateIssueCommentOption{Body: body}
	_, _, err := g.client.CreateIssueComment(owner, repo, int64(prIndex), opts)
	return err
}
