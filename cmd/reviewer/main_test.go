package main

import (
	"code-reviewer-bot/config"
	"code-reviewer-bot/constants"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitGenkit(t *testing.T) {
	ctx := context.Background()

	t.Run("Success - initializes googleai provider", func(t *testing.T) {
		cfg := &config.Config{
			LLM: config.LLMConfig{
				Provider: constants.GOOGLEAI,
				APIKey:   "fake-googleai-key",
			},
		}

		g, err := initGenkit(ctx, cfg)
		assert.NoError(t, err)
		assert.NotNil(t, g)
	})

	t.Run("Success - initializes openai provider", func(t *testing.T) {
		cfg := &config.Config{
			LLM: config.LLMConfig{
				Provider: constants.OPENAI,
				APIKey:   "fake-openai-key",
			},
		}

		g, err := initGenkit(ctx, cfg)
		assert.NoError(t, err)
		assert.NotNil(t, g)
	})

	t.Run("Failure - unsupported provider", func(t *testing.T) {
		cfg := &config.Config{
			LLM: config.LLMConfig{
				Provider: "unsupported-provider",
				APIKey:   "fake-api-key",
			},
		}

		g, err := initGenkit(ctx, cfg)
		assert.Error(t, err)
		assert.Nil(t, g)
		assert.Contains(t, err.Error(), "unsupported LLM provider")
	})
}

func TestGetPRDetailsFromEnv(t *testing.T) {
	t.Run("Success - reads standard GITHUB_REPOSITORY", func(t *testing.T) {
		t.Setenv("GITHUB_REPOSITORY", "test-owner/test-repo")
		t.Setenv("PR_NUMBER", "123")

		details, err := getPRDetailsFromEnv("", "")

		assert.NoError(t, err)
		assert.NotNil(t, details)
		assert.Equal(t, "test-owner", details.Owner)
		assert.Equal(t, "test-repo", details.Repo)
		assert.Equal(t, 123, details.PRNumber)
	})

	t.Run("Success - uses fallback REPO_OWNER and REPO_NAME", func(t *testing.T) {
		t.Setenv("GITHUB_REPOSITORY", "")
		t.Setenv("REPO_OWNER", "fallback-owner")
		t.Setenv("REPO_NAME", "fallback-repo")
		t.Setenv("PR_NUMBER", "456")

		details, err := getPRDetailsFromEnv("", "")

		assert.NoError(t, err)
		assert.NotNil(t, details)
		assert.Equal(t, "fallback-owner", details.Owner)
		assert.Equal(t, "fallback-repo", details.Repo)
		assert.Equal(t, 456, details.PRNumber)
	})

	t.Run("Failure - missing repository info", func(t *testing.T) {
		t.Setenv("GITHUB_REPOSITORY", "")
		t.Setenv("REPO_OWNER", "")
		t.Setenv("PR_NUMBER", "123")

		_, err := getPRDetailsFromEnv("", "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "env var not set")
	})

	t.Run("Failure - missing PR number", func(t *testing.T) {
		t.Setenv("GITHUB_REPOSITORY", "owner/repo")
		t.Setenv("PR_NUMBER", "")

		_, err := getPRDetailsFromEnv("", "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "PR_NUMBER env var not set")
	})

	t.Run("Failure - invalid PR number", func(t *testing.T) {
		t.Setenv("GITHUB_REPOSITORY", "owner/repo")
		t.Setenv("PR_NUMBER", "not-a-number")

		_, err := getPRDetailsFromEnv("", "")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid PR_NUMBER")
	})
}
