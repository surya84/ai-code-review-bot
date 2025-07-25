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
	"github.com/stretchr/testify/assert"
)

// createGiteaTestContext creates a mock Gin context for testing Gitea webhooks.
func createGiteaTestContext(t *testing.T, payload []byte, secret string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, err := http.NewRequest("POST", "/api/gitea/webhook", bytes.NewBuffer(payload))
	assert.NoError(t, err)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gitea-Event", "pull_request")
	req.Header.Set("X-Gitea-Signature", signature)
	c.Request = req
	return c, w
}

func TestGiteaWebhookHandler_Handle(t *testing.T) {
	secret := "my-gitea-secret"
	handler, err := NewGiteaWebhookHandler(nil, &config.Config{}, secret)
	assert.NoError(t, err)

	t.Run("Success - Handles 'opened' pull request event", func(t *testing.T) {
		payload := GiteaPullRequestHook{Action: "opened"}
		jsonPayload, _ := json.Marshal(payload)
		c, w := createGiteaTestContext(t, jsonPayload, secret)

		handler.Handle(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "Event received.", w.Body.String())
	})

	t.Run("Success - Ignores 'closed' pull request event", func(t *testing.T) {
		payload := GiteaPullRequestHook{Action: "closed"}
		jsonPayload, _ := json.Marshal(payload)
		c, w := createGiteaTestContext(t, jsonPayload, secret)

		handler.Handle(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "Event ignored.", w.Body.String())
	})

	t.Run("Failure - Invalid Signature", func(t *testing.T) {
		c, w := createGiteaTestContext(t, []byte("{}"), "wrong-gitea-secret")
		handler.Handle(c)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}
