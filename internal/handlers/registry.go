package handlers

import (
	"log"
	"os"

	"code-reviewer-bot/config"
	"code-reviewer-bot/constants"

	"github.com/firebase/genkit/go/genkit"
	"github.com/gin-gonic/gin"
)

// WebhookHandler defines the common interface that all our VCS handlers must implement.
type WebhookHandler interface {
	Handle(c *gin.Context)
}

// ProviderConfig holds all the necessary information to initialize a webhook handler for a specific VCS.
type ProviderConfig struct {
	Name                string
	Endpoint            string
	TokenEnvVar         string
	WebhookSecretEnvVar string
	// NewHandlerFunc is a factory function that creates the specific handler.
	NewHandlerFunc func(g *genkit.Genkit, cfg *config.Config, secret string) (WebhookHandler, error)
}

// AllProviders is a slice containing the configuration for all supported VCS providers.
var AllProviders = []ProviderConfig{
	{
		Name:                constants.GITHUB,
		Endpoint:            constants.GITHUB_ENDPOINT, // Grouped under /api
		TokenEnvVar:         constants.GITHUB_TOKEN,
		WebhookSecretEnvVar: constants.GITHUB_WEBHOOK_SECRET,
		NewHandlerFunc: func(g *genkit.Genkit, cfg *config.Config, secret string) (WebhookHandler, error) {
			// This type assertion is safe because NewGitHubWebhookHandler returns a type that satisfies the interface.
			return NewGitHubWebhookHandler(g, cfg, secret)
		},
	},
	{
		Name:                constants.GITEA,
		Endpoint:            constants.GITEA_ENDPOINT, // Grouped under /api
		TokenEnvVar:         constants.GITEA_TOKEN,
		WebhookSecretEnvVar: constants.GITEA_WEBHOOK_SECRET,
		NewHandlerFunc: func(g *genkit.Genkit, cfg *config.Config, secret string) (WebhookHandler, error) {
			return NewGiteaWebhookHandler(g, cfg, secret)
		},
	},
}

// RegisterHandlers iterates through all defined providers and dynamically registers their webhook
// handlers with the Gin router if their required secrets are present in the environment.
func RegisterHandlers(router *gin.Engine, g *genkit.Genkit, cfg *config.Config) {
	// Group all webhook handlers under a common API path for better organization.
	apiGroup := router.Group("/api")

	for _, provider := range AllProviders {
		token := os.Getenv(provider.TokenEnvVar)
		secret := os.Getenv(provider.WebhookSecretEnvVar)

		// Only activate the handler if both its token and secret are found.
		if token != "" && secret != "" {
			log.Printf("%s credentials found. Initializing handler...", provider.Name)
			handler, err := provider.NewHandlerFunc(g, cfg, secret)
			if err != nil {
				log.Printf("WARNING: Could not create %s webhook handler: %v", provider.Name, err)
				continue
			}
			// Register the route (e.g., POST /api/github/webhook)
			apiGroup.POST(provider.Endpoint, handler.Handle)
			log.Printf("✅ %s webhook endpoint is active.", provider.Name)
		} else {
			log.Printf("INFO: Secrets for %s not found. Skipping handler setup.", provider.Name)
		}
	}
}
