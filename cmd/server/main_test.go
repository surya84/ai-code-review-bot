// main_test.go
package main

import (
	"code-reviewer-bot/config"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitGenkit(t *testing.T) {
	ctx := context.Background()

	t.Run("Success - initializes googleai provider", func(t *testing.T) {
		cfg := &config.Config{
			LLM: config.LLMConfig{
				Provider: "googleai",
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
				Provider: "openai",
				APIKey:   "fake-openai-key",
			},
		}
		g, err := initGenkit(ctx, cfg)
		assert.NoError(t, err)
		assert.NotNil(t, g)
	})

	t.Run("Failure - unsupported provider", func(t *testing.T) {
		cfg := &config.Config{
			LLM: config.LLMConfig{Provider: "unsupported-provider"},
		}
		_, err := initGenkit(ctx, cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported LLM provider")
	})

	t.Run("Failure - missing API key for selected provider", func(t *testing.T) {
		cfg := &config.Config{
			LLM: config.LLMConfig{
				Provider: "googleai",
			},
		}
		_, err := initGenkit(ctx, cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "is not set")
	})
}

func TestRunServer_InvalidGenkit(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider: "invalid_provider",
			APIKey:   "fake-key",
		},
	}
	err := RunServer(ctx, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported LLM provider")
}
