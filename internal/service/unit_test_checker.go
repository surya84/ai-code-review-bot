package service

import (
	"code-reviewer-bot/internal/models"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var functionPatterns = map[string]*regexp.Regexp{
	".go":   regexp.MustCompile(`func\s+([A-Z][A-Za-z0-9_]*)\s*\(`),
	".js":   regexp.MustCompile(`(?:function\s+([A-Za-z_$][A-Za-z0-9_$]*)|const\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=)`),
	".ts":   regexp.MustCompile(`(?:function\s+([A-Za-z_$][A-Za-z0-9_$]*)|const\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=)`),
	".py":   regexp.MustCompile(`def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`),
	".java": regexp.MustCompile(`public\s+[A-Za-z<>\[\]]+\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`),
}

var testSuffixes = map[string]string{
	".go":   "_test.go",
	".js":   ".test.js",
	".ts":   ".test.ts",
	".py":   "_test.py",
	".java": "Test.java",
}

func (s *ReviewService) CheckForMissingTests(ctx context.Context, diff string, repoPath string) (*models.TestReviewResponse, error) {
	language := detectLanguage(diff)
	if language == "" {
		return &models.TestReviewResponse{HasMissing: false}, nil
	}

	functions := extractFunctions(diff, language)
	if len(functions) == 0 {
		return &models.TestReviewResponse{HasMissing: false}, nil
	}

	missingFuncs := findMissingTests(functions, repoPath, language, diff)
	if len(missingFuncs) == 0 {
		return &models.TestReviewResponse{HasMissing: false}, nil
	}

	comments := generateTestComments(missingFuncs)

	return &models.TestReviewResponse{
		MissingTests: missingFuncs,
		Comments:     comments,
		HasMissing:   true,
	}, nil
}

func generateTestComments(missingFuncs []string) []models.Comment {
	body := "## 🧪 Missing Unit Tests\n\n"
	body += "The following functions are missing unit tests:\n\n"

	for _, fn := range missingFuncs {
		body += fmt.Sprintf("- `%s()`\n", fn)
	}

	body += "\n**Why unit tests are important:**\n"
	body += "- Prevent regressions when code changes\n"
	body += "- Document expected behavior\n"
	body += "- Improve code maintainability\n"
	body += "- Catch bugs early in development\n\n"
	body += "Please add unit tests for these functions to maintain code quality."

	return []models.Comment{{
		Body: body,
	}}
}

func detectLanguage(diff string) string {
	for ext := range functionPatterns {
		if strings.Contains(diff, ext) {
			return ext
		}
	}
	return ""
}

func extractFunctions(diff, language string) map[string]bool {
	pattern, exists := functionPatterns[language]
	if !exists {
		return nil
	}

	var functions []string
	funcsMap := make(map[string]bool)
	lines := strings.Split(diff, "\n")

	for _, line := range lines {

		if !strings.HasPrefix(line, "+") || strings.HasPrefix(line, "+++") {
			continue
		}

		matches := pattern.FindAllStringSubmatch(line, -1)
		var funcName string
		for _, match := range matches {
			for i := 1; i < len(match); i++ {
				if match[i] != "" {
					if !strings.Contains(match[i], "Test") {
						functions = append(functions, match[i])
						funcName = match[i]
						funcsMap[match[i]] = false
					} else {
						break
					}
				}
			}
		}
		if funcName != "" {
			if strings.Contains(diff, "Test"+funcName) || strings.Contains(diff, funcName+"Test") {
				if _, ok := funcsMap[funcName]; ok {
					funcsMap[funcName] = true
				}
			}
		}

	}

	return funcsMap
}

func findMissingTests(functions map[string]bool, repoPath, language, diff string) []string {
	testSuffix, exists := testSuffixes[language]
	if !exists {
		return nil
	}

	var missing []string

	for fn, ok := range functions {
		if !ok {
			hasTest := false

			filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}

				// Check  test files
				if !strings.HasSuffix(path, testSuffix) {
					return nil
				}

				content, err := os.ReadFile(path)
				if err != nil {
					return nil
				}

				if strings.Contains(string(content), string("Test"+fn)) {
					hasTest = true
					return filepath.SkipDir
				}

				return nil
			})

			if !hasTest {
				missing = append(missing, fn)
			}
		}

	}

	return missing
}
