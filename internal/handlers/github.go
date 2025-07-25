package handlers

import (
	"context"
	"log"
	"net/http"

	"code-reviewer-bot/config"
	"code-reviewer-bot/constants"
	"code-reviewer-bot/internal/models"
	"code-reviewer-bot/internal/repository"
	"code-reviewer-bot/internal/service"

	"github.com/firebase/genkit/go/genkit"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v62/github"
)

// GitHubWebhookHandler handles incoming GitHub webhooks.
type GitHubWebhookHandler struct {
	reviewService *service.ReviewService
	secret        []byte
}

// NewGitHubWebhookHandler creates a new handler.
func NewGitHubWebhookHandler(g *genkit.Genkit, cfg *config.Config, secret string) (*GitHubWebhookHandler, error) {
	// The handler creates its own dependencies (repo and service).
	repo := repository.NewGitHubRepository(context.Background(), cfg.VCS.GitHub.Token)
	reviewService := service.NewReviewService(repo, g, cfg)
	return &GitHubWebhookHandler{
		reviewService: reviewService,
		secret:        []byte(secret),
	}, nil
}

// Handle is the Gin handler function.
func (h *GitHubWebhookHandler) Handle(c *gin.Context) {
	payload, err := github.ValidatePayload(c.Request, h.secret)
	if err != nil {
		log.Printf("Error validating GitHub payload: %v", err)
		c.String(http.StatusForbidden, "Forbidden")
		return
	}
	event, err := github.ParseWebHook(github.WebHookType(c.Request), payload)
	if err != nil {
		log.Printf("Error parsing GitHub event: %v", err)
		c.String(http.StatusBadRequest, "Bad Request")
		return
	}
	switch event := event.(type) {
	case *github.PullRequestEvent:
		action := event.GetAction()
		if action == constants.OPENED || action == constants.SYNCHRONIZE || action == constants.REOPENED {
			log.Printf("Received GitHub PR event: %s for PR #%d", action, event.GetNumber())
			go h.processPullRequest(event)
		} else {
			log.Printf("Ignoring GitHub PR action: %s", action)
		}
		c.String(http.StatusOK, "Event received.")
	default:
		log.Printf("Ignoring GitHub webhook event type: %T", event)
		c.String(http.StatusOK, "Event type ignored.")
	}
}

func (h *GitHubWebhookHandler) processPullRequest(event *github.PullRequestEvent) {
	pr := event.GetPullRequest()
	if pr.GetState() != constants.OPEN {
		log.Printf("Ignoring PR #%d because its state is '%s'", event.GetNumber(), pr.GetState())
		return
	}
	prDetails := &models.PRDetails{
		Owner:    pr.Base.Repo.GetOwner().GetLogin(),
		Repo:     pr.Base.Repo.GetName(),
		PRNumber: event.GetNumber(),
		Title:    pr.GetTitle(),
		Branch:   pr.GetHead().GetRef(),
		URL:      pr.GetHTMLURL(),
	}
	baseUrl := constants.GITHUB_URL
	_, err := h.reviewService.ProcessPullRequest(baseUrl, context.Background(), prDetails)
	if err != nil {
		log.Printf("Code review failed for GitHub PR #%d: %v", prDetails.PRNumber, err)

	}
}
