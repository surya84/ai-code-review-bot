package service

import (
	"code-reviewer-bot/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/firebase/genkit/go/ai"
)

var architectureLayers = map[string][]string{
	"API/Controllers": {"handlers", "controllers", "routes", "api", "endpoints", "middlewares"},
	"Business Logic":  {"services", "business", "domain", "core", "logic", "service", "utils"},
	"Data Access":     {"repositories", "dao", "dto", "data", "models", "storage", "repository", "database"},
	"Configuration":   {"config", "env", "settings", "yaml", "configuration"},
}

func (s *ReviewService) reviewProjectArchitecture(ctx context.Context, repoPath string) (*models.ArchitectureReviewResponse, error) {
	directories := getProjectDirectories(repoPath)
	if len(directories) == 0 {
		return &models.ArchitectureReviewResponse{
			Score:    0,
			Feedback: "No meaningful directory structure found.",
			Comments: []models.Comment{{
				Body: "⚠️ **Architecture Review**\n\nNo clear project structure detected. Consider organizing code into proper architectural layers:\n- API/Controllers layer\n- Business Logic layer\n- Data Access layer\n- Configuration layer",
			}},
			NeedsComment: true,
		}, nil
	}

	foundLayers := categorizeDirectories(directories)
	missingLayers := findMissingLayers(foundLayers)
	score := calculateArchitectureScore(foundLayers)
	summary := generateStructureSummary(directories, foundLayers)

	comments, needsComment := generateArchitectureComments(ctx, s, summary, score, missingLayers)

	return &models.ArchitectureReviewResponse{
		Score:         score,
		FoundLayers:   foundLayers,
		MissingLayers: missingLayers,
		Feedback:      summary,
		Comments:      comments,
		NeedsComment:  needsComment,
	}, nil
}

func generateArchitectureComments(ctx context.Context, s *ReviewService, summary string, score int, missingLayers []string) ([]models.Comment, bool) {
	// Only generate comment if there are issues
	if score >= 8 && len(missingLayers) == 0 {
		return []models.Comment{}, false
	}

	missingStr := "None"
	if len(missingLayers) > 0 {
		missingStr = strings.Join(missingLayers, ", ")
	}

	prompt := fmt.Sprintf(s.cfg.ArchitectureReviewPrompt, summary, score, missingStr)
	response, err := genkitGenerate(ctx, s.g, ai.WithModelName(s.cfg.LLM.ModelName), ai.WithPrompt(prompt))
	if err != nil {
		// Fallback comment if AI fails
		return generateFallbackArchitectureComments(score, missingLayers), true
	}

	var aiResponse struct {
		Comments []models.Comment `json:"comments"`
	}

	if err := json.Unmarshal([]byte(response.Text()), &aiResponse); err != nil {
		return []models.Comment{{
			Body: response.Text(),
		}}, true
	}

	return aiResponse.Comments, true
}

func generateFallbackArchitectureComments(score int, missingLayers []string) []models.Comment {
	body := fmt.Sprintf("## 🏗️ Architecture Review\n\n**Score: %d/10**\n\n", score)

	if len(missingLayers) > 0 {
		body += "**Missing architectural layers:**\n"
		for _, layer := range missingLayers {
			body += fmt.Sprintf("- %s\n", layer)
		}
		body += "\n"
	}

	body += "**Recommendations:**\n"
	body += "- Consider implementing proper separation of concerns\n"
	body += "- Organize code into clear architectural layers\n"
	body += "- Follow modular design principles for better maintainability"

	return []models.Comment{{
		Body: body,
	}}
}

func getProjectDirectories(repoPath string) []string {
	var directories []string

	filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() || path == repoPath {
			return nil
		}

		// Skip common non-source directories
		name := strings.ToLower(filepath.Base(path))
		skipDirs := []string{"node_modules", "vendor", "__pycache__", ".git", ".vscode", "bin", "build"}

		for _, skip := range skipDirs {
			if name == skip || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
		}

		directories = append(directories, filepath.Base(path))
		return nil
	})

	return directories
}

func categorizeDirectories(directories []string) map[string][]string {
	found := make(map[string][]string)

	for layer, patterns := range architectureLayers {
		found[layer] = []string{}

		for _, dir := range directories {
			dirLower := strings.ToLower(dir)

			for _, pattern := range patterns {
				if strings.Contains(dirLower, pattern) {
					found[layer] = append(found[layer], dir)
					break
				}
			}
		}
	}

	return found
}

func findMissingLayers(foundLayers map[string][]string) []string {
	var missing []string

	for layer, dirs := range foundLayers {
		if len(dirs) == 0 {
			missing = append(missing, layer)
		}
	}

	return missing
}

func generateStructureSummary(directories []string, foundLayers map[string][]string) string {
	var summary strings.Builder

	summary.WriteString(fmt.Sprintf("Total directories: %d\n", len(directories)))
	summary.WriteString(fmt.Sprintf("All directories: %s\n\n", strings.Join(directories, ", ")))

	summary.WriteString("Architecture layers found:\n")
	for layer, dirs := range foundLayers {
		if len(dirs) > 0 {
			summary.WriteString(fmt.Sprintf("- %s: %s\n", layer, strings.Join(dirs, ", ")))
		} else {
			summary.WriteString(fmt.Sprintf("- %s: MISSING\n", layer))
		}
	}

	return summary.String()
}

func calculateArchitectureScore(foundLayers map[string][]string) int {
	totalLayers := len(architectureLayers)
	foundCount := 0

	for _, dirs := range foundLayers {
		if len(dirs) > 0 {
			foundCount++
		}
	}

	// Simple scoring: (found layers / total layers) * 10
	return (foundCount * 10) / totalLayers
}
