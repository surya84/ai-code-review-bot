package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net/http"

	"code-reviewer-bot/config"
	"code-reviewer-bot/constants"
	"code-reviewer-bot/internal/models"
	"code-reviewer-bot/internal/repository"
	"code-reviewer-bot/internal/service"

	"github.com/firebase/genkit/go/genkit"
	"github.com/gin-gonic/gin"
)

// GiteaPullRequestHook represents the structure of Gitea's PR webhook payload.
type GiteaPullRequestHook struct {
	Action string `json:"action"`
	Number int64  `json:"number"`
	Repo   struct {
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
		Name string `json:"name"`
	} `json:"repository"`
	PullRequest struct {
		Title   string `json:"title"`
		HTMLURL string `json:"html_url"`
		Head    struct {
			Ref string `json:"ref"`
		} `json:"head"`
	} `json:"pull_request"`
}

// GiteaWebhookHandler handles incoming Gitea webhooks.
type GiteaWebhookHandler struct {
	reviewService *service.ReviewService
	secret        string
}

// NewGiteaWebhookHandler creates a new handler.
func NewGiteaWebhookHandler(g *genkit.Genkit, cfg *config.Config, secret string) (*GiteaWebhookHandler, error) {
	repo := repository.NewGiteaRepository(context.Background(), cfg.VCS.Gitea.BaseURL, cfg.VCS.Gitea.Token)
	reviewService := service.NewReviewService(repo, g, cfg)
	return &GiteaWebhookHandler{
		reviewService: reviewService,
		secret:        secret,
	}, nil
}

// Handle is the Gin handler function.
func (h *GiteaWebhookHandler) Handle(c *gin.Context) {
	signature := c.GetHeader("X-Gitea-Signature")
	body, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	if signature != expectedSignature {
		c.String(http.StatusForbidden, "Forbidden: Invalid signature")
		return
	}

	var payload GiteaPullRequestHook
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.String(http.StatusBadRequest, "Bad Request")
		return
	}

	action := payload.Action
	if action == "opened" || action == "synchronize" || action == "reopened" {
		log.Printf("Received Gitea PR event: %s for PR #%d", action, payload.Number)
		go h.processPullRequest(&payload)
		c.String(http.StatusOK, "Event received.")
	} else {
		log.Printf("Ignoring Gitea PR action: %s", action)
		c.String(http.StatusOK, "Event ignored.")
	}
}

func (h *GiteaWebhookHandler) processPullRequest(payload *GiteaPullRequestHook) {
	prDetails := &models.PRDetails{
		Owner:    payload.Repo.Owner.Login,
		Repo:     payload.Repo.Name,
		PRNumber: int(payload.Number),
		Title:    payload.PullRequest.Title,
		Branch:   payload.PullRequest.Head.Ref,
		URL:      payload.PullRequest.HTMLURL,
	}
	baseUrl := constants.GITEA_URL
	_, err := h.reviewService.ProcessPullRequest(baseUrl, context.Background(), prDetails)
	if err != nil {
		log.Printf("Code review failed for Gitea PR #%d: %v", prDetails.PRNumber, err)
	}
}
