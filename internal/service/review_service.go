package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"regexp"
	"strings"

	"code-reviewer-bot/config"
	"code-reviewer-bot/constants"
	"code-reviewer-bot/internal/diffparser"
	"code-reviewer-bot/internal/models"
	"code-reviewer-bot/internal/repository"
	"code-reviewer-bot/internal/utils"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// ReviewService encapsulates the core business logic for reviewing a pull request.
type ReviewService struct {
	repo repository.VcsRepository
	g    *genkit.Genkit
	cfg  *config.Config
}

var genkitGenerate = genkit.Generate

// NewReviewService creates a new service instance.
func NewReviewService(vcsRepo repository.VcsRepository, g *genkit.Genkit, cfg *config.Config) *ReviewService {
	return &ReviewService{repo: vcsRepo, g: g, cfg: cfg}
}

// ProcessPullRequest is the main orchestration method.
func (s *ReviewService) ProcessPullRequest(baseUrl string, ctx context.Context, prDetails *models.PRDetails) (string, error) {
	log.Printf("Starting review for PR #%d in %s/%s", prDetails.PRNumber, prDetails.Owner, prDetails.Repo)
	var allComments []*models.Comment

	// Step 1: Project Architecture Review
	var token string
	if strings.EqualFold(baseUrl, constants.GITHUB_URL) {
		token = s.cfg.VCS.GitHub.Token
	} else if strings.EqualFold(baseUrl, constants.GITEA_URL) {
		token = s.cfg.VCS.Gitea.Token
	}
	repoPath, err := utils.CloneRepoIfNotExists(baseUrl, token, prDetails.Owner, prDetails.Repo)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := os.RemoveAll(repoPath); err != nil {
			log.Printf("Warning: failed to clean up repo path %s: %v", repoPath, err)
		}
		log.Printf("Cleaned up repo path %s", repoPath)
	}()

	archReview, err := s.reviewProjectArchitecture(ctx, repoPath)
	if err != nil {
		log.Printf("Architecture review failed: %v", err)
	} else if archReview != nil && archReview.NeedsComment {
		comment := utils.FormatArchitectureReviewComment(archReview)
		err := s.repo.PostGeneralComment(ctx, prDetails.Owner, prDetails.Repo, prDetails.PRNumber, comment)
		if err != nil {
			log.Printf("Error posting general comment: %v", err)
		}
	}

	commitID, err := s.repo.GetPRCommitID(ctx, prDetails.Owner, prDetails.Repo, prDetails.PRNumber)
	if err != nil {
		log.Printf("Warning: could not get PR commit ID: %v", err)
	} else {
		log.Printf("Found PR HEAD commit SHA: %s", commitID)
	}

	diff, err := s.repo.GetPRDiff(ctx, prDetails.Owner, prDetails.Repo, prDetails.PRNumber)
	if err != nil {
		return "", fmt.Errorf("failed to get PR diff: %w", err)
	}
	log.Println("Successfully fetched PR diff.")

	testComment, err := s.CheckForMissingTests(ctx, diff, repoPath)
	if err == nil && testComment != nil {
		//var testReviewComments []*models.Comment
		for _, comment := range testComment.Comments {
			err := s.repo.PostGeneralComment(ctx, prDetails.Owner, prDetails.Repo, prDetails.PRNumber, comment.Body)
			if err != nil {
				log.Printf("Error posting general comment: %v", err)
			}
		}
	}

	chunks := diffparser.Parse(diff)
	if len(chunks) == 0 {
		return "No reviewable changes found.", nil
	}
	log.Printf("Parsed diff into %d chunks.", len(chunks))

	for _, chunk := range chunks {
		comments, err := s.analyzeChunk(ctx, chunk)
		if err != nil {
			log.Printf("Error analyzing chunk for file %s: %v", chunk.FilePath, err)
			continue
		}

		for _, llmComment := range comments {
			positionInHunk, fileLineNumber, err := findLocationForLineContent(chunk, llmComment.LineContent)
			if err != nil {
				log.Printf("Could not find location for line content in file %s: %v", chunk.FilePath, err)
				continue
			}
			allComments = append(allComments, &models.Comment{
				Body:     llmComment.Message,
				Path:     chunk.FilePath,
				Position: positionInHunk,
				Line:     fileLineNumber,
			})
		}
	}

	if len(allComments) > 0 {
		log.Printf("Submitting a review with %d comments.", len(allComments))
		err := s.repo.PostReview(ctx, prDetails.Owner, prDetails.Repo, prDetails.PRNumber, allComments, commitID)
		if err != nil {
			// reviewErr = err
			return "", fmt.Errorf("failed to post review: %w", err)
		}
	} else {
		log.Println("No comments to post. Submitting a general comment.")
		s.repo.PostGeneralComment(ctx, prDetails.Owner, prDetails.Repo, prDetails.PRNumber, "✅ AI Review Complete: No issues found.")
	}

	resultMessage := fmt.Sprintf("Review complete. Submitted %d comments.", len(allComments))
	log.Println(resultMessage)
	return resultMessage, nil
}

func (s *ReviewService) analyzeChunk(ctx context.Context, chunk *diffparser.DiffChunk) ([]models.ReviewComment, error) {
	prompt, err := preparePrompt(s.cfg.ReviewPrompt, chunk.FilePath, chunk.CodeSnippet)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare prompt: %w", err)
	}

	res, err := genkitGenerate(ctx, s.g, ai.WithModelName(s.cfg.LLM.ModelName), ai.WithPrompt(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate LLM response: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to generate LLM response: %w", err)
	}

	responseText := res.Text()
	if responseText == "" {
		return nil, fmt.Errorf("failed to get text from LLM response")
	}

	sanitizedJSON := sanitizeJSONString(responseText)
	if sanitizedJSON == "" {
		return nil, nil
	}

	var comments []models.ReviewComment
	if err := json.Unmarshal([]byte(sanitizedJSON), &comments); err != nil {
		return nil, fmt.Errorf("failed to parse LLM JSON response: %w", err)
	}
	return comments, nil
}

// Helper functions (can remain in this file or be moved to a utility package)
func sanitizeJSONString(s string) string {
	startIndex := strings.Index(s, "[")
	endIndex := strings.LastIndex(s, "]")
	if startIndex == -1 || endIndex == -1 || endIndex < startIndex {
		return ""
	}
	s = s[startIndex : endIndex+1]
	replacer := strings.NewReplacer("\n", " ", "\t", " ", "\r", " ")
	s = replacer.Replace(s)
	re := regexp.MustCompile(`,(\s*[\}\]])`)
	return re.ReplaceAllString(s, "$1")
}

func findLocationForLineContent(chunk *diffparser.DiffChunk, lineContent string) (int, int, error) {
	normalize := func(s string) string { return strings.Join(strings.Fields(s), " ") }
	target := normalize(lineContent)
	if target == "" {
		return -1, -1, fmt.Errorf("LLM provided empty line content")
	}
	lines := strings.Split(chunk.CodeSnippet, "\n")
	hunkPosition := 0
	currentFileNumber := chunk.StartLineNew
	for i, line := range lines {
		hunkPosition = i + 1
		if strings.HasPrefix(line, "@@") || strings.HasPrefix(line, "-") {
			continue
		}
		current := normalize(line)
		if current == target {
			if !strings.HasPrefix(strings.TrimSpace(line), "+") {
				return -1, -1, fmt.Errorf("matched line is not an added line ('+'): '%s'", line)
			}
			return hunkPosition, currentFileNumber, nil
		}
		currentFileNumber++
	}
	return -1, -1, fmt.Errorf("line content not found in diff hunk: '%s'", lineContent)
}

func preparePrompt(promptTmpl, filePath, codeSnippet string) (string, error) {
	tmpl, err := template.New("review_prompt").Parse(promptTmpl)
	if err != nil {
		return "", err
	}
	data := struct {
		FilePath    string
		CodeSnippet string
	}{
		FilePath:    filePath,
		CodeSnippet: codeSnippet,
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
