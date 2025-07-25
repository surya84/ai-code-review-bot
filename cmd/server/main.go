// main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"code-reviewer-bot/config"
	"code-reviewer-bot/constants"
	"code-reviewer-bot/internal/handlers"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/anthropic"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/gin-gonic/gin"
	"github.com/openai/openai-go/option"
)

func main() {
	log.Println("Starting AI Code Reviewer in server mode with Gin...")
	ctx := context.Background()

	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	if err := RunServer(ctx, cfg); err != nil {
		log.Fatalf("Failed to start Gin server: %v", err)
	}
}

// RunServer initializes Genkit, sets up routes, and runs the Gin server.
func RunServer(ctx context.Context, cfg *config.Config) error {
	g, err := initGenkit(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize Genkit: %w", err)
	}

	router := gin.Default()
	handlers.RegisterHandlers(router, g, cfg)

	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "AI Code Reviewer Bot is running.")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Listening for webhooks on port %s", port)
	return router.Run(":" + port)
}

// initGenkit initializes the Genkit instance and loads the appropriate LLM plugin.
func initGenkit(ctx context.Context, cfg *config.Config) (*genkit.Genkit, error) {
	var plugin genkit.Plugin
	switch cfg.LLM.Provider {
	case constants.GOOGLEAI:
		plugin = &googlegenai.GoogleAI{APIKey: cfg.LLM.APIKey}
	case constants.OPENAI:
		plugin = &openai.OpenAI{APIKey: cfg.LLM.APIKey}
	case constants.CLAUDAI:
		plugin = &anthropic.Anthropic{Opts: []option.RequestOption{option.WithAPIKey(cfg.LLM.APIKey)}}
	default:
		return nil, fmt.Errorf("unsupported LLM provider in config: %s", cfg.LLM.Provider)
	}
	return genkit.Init(ctx, genkit.WithPlugins(plugin))
}
