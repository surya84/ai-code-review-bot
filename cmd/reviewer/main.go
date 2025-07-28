package main

import (
	"code-reviewer-bot/internal/models"
	"code-reviewer-bot/internal/repository"
	"code-reviewer-bot/internal/service"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"code-reviewer-bot/config"
	"code-reviewer-bot/constants"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

var (
	configPath string
	repoOwner  string
	repoName   string
	prNumber   int
)

func main() {
	Execute()
}

var rootCmd = &cobra.Command{
	Use:   "code-reviewer-bot",
	Short: "AI-powered Code Reviewer",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		g, err := initGenkit(ctx, cfg)
		if err != nil {
			log.Fatalf("Failed to initialize Genkit: %v", err)
		}

		prDetails, err := getPRDetailsFromEnv(cfg.VCS.Provider, cfg.VCS.Gitea.BaseURL)
		if err != nil {
			log.Fatalf("Failed to get PR details: %v", err)
		}

		vcsClient, err := repository.New(ctx, &cfg.VCS)
		if err != nil {
			log.Fatalf("Failed to create VCS client: %v", err)
		}

		reviewService := service.NewReviewService(vcsClient, g, cfg)
		var baseUrl string

		if cfg.VCS.Provider == "Github" {
			baseUrl = constants.GITHUB_URL
		} else if cfg.VCS.Provider == "Gitea" {
			baseUrl = constants.GITEA_URL
		} else {
			log.Fatalf("Unsupported VCS provider: %s", cfg.VCS.Provider)
		}

		result, err := reviewService.ProcessPullRequest(baseUrl, ctx, prDetails)
		if err != nil {
			log.Fatalf("Code review process failed: %v", err)
		}

		log.Printf("Process finished: %s", result)
	},
}

func init() {
	rootCmd.Flags().StringVar(&configPath, "config", "/app/config/config.yaml", "Path to config.yaml")
	rootCmd.Flags().StringVar(&repoOwner, "repo-owner", "", "Repository owner (overrides env)")
	rootCmd.Flags().StringVar(&repoName, "repo-name", "", "Repository name (overrides env)")
	rootCmd.Flags().IntVar(&prNumber, "pr-number", 0, "PR number (overrides env)")
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

// initGenkit initializes the Genkit instance and loads the appropriate LLM plugin.
func initGenkit(ctx context.Context, cfg *config.Config) (*genkit.Genkit, error) {
	var plugin genkit.Plugin
	switch cfg.LLM.Provider {
	case constants.GOOGLEAI:
		plugin = &googlegenai.GoogleAI{APIKey: cfg.LLM.APIKey}
	case constants.OPENAI:
		plugin = &openai.OpenAI{APIKey: cfg.LLM.APIKey}
	default:
		return nil, fmt.Errorf("unsupported LLM provider in config: %s", cfg.LLM.Provider)
	}
	return genkit.Init(ctx, genkit.WithPlugins(plugin))
}

// getPRDetailsFromEnv retrieves PR information from environment variables.
func getPRDetailsFromEnv(provider string, baseUrl string) (*models.PRDetails, error) {
	var repoSlug string
	var parts []string
	if provider == "Github" {
		repoSlug = os.Getenv(constants.GITHUB_REPOSITORY)
	} else if provider == constants.GITEA {
		repoSlug = os.Getenv(constants.GITEA_REPOSITORY)
	} else {
		return nil, fmt.Errorf("unsupported VCS provider in config: %s", provider)
	}

	if repoSlug == "" {
		repoOwner := os.Getenv("REPO_OWNER")
		repoName := os.Getenv("REPO_NAME")
		if repoOwner == "" || repoName == "" {
			return nil, fmt.Errorf("GITHUB_REPOSITORY (or REPO_OWNER/REPO_NAME) env var not set")
		}
		repoSlug = fmt.Sprintf("%s/%s", repoOwner, repoName)
	}

	parts = strings.Split(repoSlug, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid GITHUB_REPOSITORY format: %s", repoSlug)
	}

	prNumberStr := os.Getenv("PR_NUMBER")
	if prNumberStr == "" {
		return nil, fmt.Errorf("PR_NUMBER env var not set")
	}

	prNumber, err := strconv.Atoi(prNumberStr)
	if err != nil {
		return nil, fmt.Errorf("invalid PR_NUMBER: %w", err)
	}

	return &models.PRDetails{
		Owner:    parts[0],
		Repo:     parts[1],
		PRNumber: prNumber,
	}, nil
}
