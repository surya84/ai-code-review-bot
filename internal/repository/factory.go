package repository

import (
	"code-reviewer-bot/config"
	"context"
	"fmt"
)

// New creates and returns the appropriate VCS repository client based on the configuration.
// This factory function is primarily used by the CLI tool to select the correct provider.
func New(ctx context.Context, cfg *config.VCSConfig) (VcsRepository, error) {
	switch cfg.Provider {
	case "github":
		if cfg.GitHub.Token == "" {
			return nil, fmt.Errorf("github provider selected but GITHUB_TOKEN is not configured")
		}
		return NewGitHubRepository(ctx, cfg.GitHub.Token), nil
	case "Gitea":
		if cfg.Gitea.Token == "" {
			return nil, fmt.Errorf("gitea provider selected but GITEA_TOKEN is not configured")
		}
		return NewGiteaRepository(ctx, cfg.Gitea.BaseURL, cfg.Gitea.Token), nil
	default:
		return nil, fmt.Errorf("unsupported VCS provider in config: '%s'", cfg.Provider)
	}
}
