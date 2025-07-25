package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"code-reviewer-bot/config"

	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v62/github"
	"github.com/stretchr/testify/assert"
)

// createGitHubTestContext creates a mock Gin context with a request for testing.
func createGitHubTestContext(t *testing.T, payload []byte, secret, eventType string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, err := http.NewRequest("POST", "/api/github/webhook", bytes.NewBuffer(payload))
	assert.NoError(t, err)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", eventType)
	req.Header.Set("X-Hub-Signature-256", signature)
	c.Request = req
	return c, w
}

func TestGitHubWebhookHandler_Handle(t *testing.T) {
	secret := "my-super-secret-key"
	// For these unit tests, we can pass nil for Genkit and an empty config
	// because we are only testing the handler's routing logic, not the full service call.
	handler, err := NewGitHubWebhookHandler(nil, &config.Config{}, secret)
	assert.NoError(t, err)

	t.Run("Success - Handles 'opened' pull request event", func(t *testing.T) {
		payload := github.PullRequestEvent{Action: github.String("opened")}
		jsonPayload, _ := json.Marshal(payload)
		c, w := createGitHubTestContext(t, jsonPayload, secret, "pull_request")

		handler.Handle(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "Event received.", w.Body.String())
	})

	t.Run("Success - Ignores 'closed' pull request event", func(t *testing.T) {
		payload := github.PullRequestEvent{Action: github.String("closed")}
		jsonPayload, _ := json.Marshal(payload)
		c, w := createGitHubTestContext(t, jsonPayload, secret, "pull_request")

		handler.Handle(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "Event received.", w.Body.String())
	})

	t.Run("Failure - Invalid Signature", func(t *testing.T) {
		c, w := createGitHubTestContext(t, []byte("{}"), "wrong-secret", "pull_request")
		handler.Handle(c)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}
