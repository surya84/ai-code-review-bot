package service

import (
	"context"
	"errors"
	"testing"

	"code-reviewer-bot/config"
	"code-reviewer-bot/internal/diffparser"
	"code-reviewer-bot/internal/models"
	"code-reviewer-bot/internal/repository"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/mock/gomock"
)

//var genkitGenerate = genkit.Generate

func TestProcessPullRequest(t *testing.T) {
	// Common setup for all ProcessPullRequest tests
	ctx := context.Background()
	prDetails := &models.PRDetails{Owner: "test", Repo: "repo", PRNumber: 1}
	cfg := &config.Config{
		LLM:          config.LLMConfig{Provider: "test-provider", ModelName: "test-model"},
		ReviewPrompt: `{{.CodeSnippet}}`,
	}
	var g *genkit.Genkit

	t.Run("Success - posts comments when AI finds issues", func(t *testing.T) {

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := repository.NewMockVcsRepository(ctrl)
		reviewService := NewReviewService(mockRepo, g, cfg)

		mockRepo.EXPECT().GetPRCommitID(gomock.Any(), "test", "repo", 1).Return("commit123", nil)
		mockRepo.EXPECT().GetPRDiff(gomock.Any(), "test", "repo", 1).Return("diff --git a/main.go b/main.go\n@@ -1,0 +1,1 @@\n+ some change", nil)
		mockRepo.EXPECT().PostReview(gomock.Any(), "test", "repo", 1, gomock.Any(), "commit123").Return(nil)

		originalGenerate := genkitGenerate
		genkitGenerate = func(ctx context.Context, g *genkit.Genkit, options ...ai.GenerateOption) (*ai.ModelResponse, error) {
			return &ai.ModelResponse{
				Message: &ai.Message{
					Content: []*ai.Part{
						ai.NewTextPart(`[{"line_content": "+ some change", "message": "A valid comment"}]`),
					},
				},
			}, nil
		}
		defer func() { genkitGenerate = originalGenerate }()

		_, err := reviewService.ProcessPullRequest("", ctx, prDetails)

		assert.NoError(t, err)

		mock.AssertExpectationsForObjects(t)

	})

	t.Run("Success - posts general comment when AI finds no issues", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := repository.NewMockVcsRepository(ctrl)
		reviewService := NewReviewService(mockRepo, g, cfg)

		mockRepo.EXPECT().GetPRCommitID(gomock.Any(), "test", "repo", 1).Return("commit123", nil)
		mockRepo.EXPECT().GetPRDiff(gomock.Any(), "test", "repo", 1).Return("diff --git a/main.go b/main.go\n@@ -1,0 +1,1 @@\n+ some change", nil)
		mockRepo.EXPECT().PostGeneralComment(gomock.Any(), "test", "repo", 1, gomock.Any()).Return(nil)

		originalGenerate := genkitGenerate
		genkitGenerate = func(ctx context.Context, g *genkit.Genkit, options ...ai.GenerateOption) (*ai.ModelResponse, error) {
			return &ai.ModelResponse{
				Message: &ai.Message{
					Content: []*ai.Part{
						ai.NewTextPart(`[]`),
					},
				},
			}, nil
		}
		defer func() { genkitGenerate = originalGenerate }()

		_, err := reviewService.ProcessPullRequest("", ctx, prDetails)
		assert.NoError(t, err)
		mock.AssertExpectationsForObjects(t)
	})

	t.Run("Failure - returns error if getting PR diff fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockRepo := repository.NewMockVcsRepository(ctrl)

		reviewService := NewReviewService(mockRepo, g, cfg)

		mockRepo.EXPECT().GetPRCommitID(gomock.Any(), "test", "repo", 1).Return("commit123", nil)
		mockRepo.EXPECT().GetPRDiff(gomock.Any(), "test", "repo", 1).Return("", errors.New("network error"))

		_, err := reviewService.ProcessPullRequest("", ctx, prDetails)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "network error")

	})
}

func TestAnalyzeChunk(t *testing.T) {
	cfg := &config.Config{
		LLM:          config.LLMConfig{ModelName: "test-model"},
		ReviewPrompt: `{{.CodeSnippet}}`,
	}
	chunk := &diffparser.DiffChunk{FilePath: "main.go", CodeSnippet: "+ test line"}
	var g *genkit.Genkit

	reviewService := NewReviewService(nil, g, cfg)

	t.Run("Success - parses valid JSON", func(t *testing.T) {
		originalGenerate := genkitGenerate
		genkitGenerate = func(ctx context.Context, g *genkit.Genkit, options ...ai.GenerateOption) (*ai.ModelResponse, error) {
			return &ai.ModelResponse{
				Message: &ai.Message{
					Content: []*ai.Part{
						ai.NewTextPart(`[{"line_content": "+ test line", "message": "A good comment"}]`),
					},
				},
			}, nil
		}
		defer func() { genkitGenerate = originalGenerate }()

		comments, err := reviewService.analyzeChunk(context.Background(), chunk)
		assert.NoError(t, err)
		assert.Len(t, comments, 1)
		assert.Equal(t, "A good comment", comments[0].Message)
	})

	t.Run("Failure - returns error on LLM error", func(t *testing.T) {
		originalGenerate := genkitGenerate
		genkitGenerate = func(ctx context.Context, g *genkit.Genkit, options ...ai.GenerateOption) (*ai.ModelResponse, error) {
			return &ai.ModelResponse{
				Message: &ai.Message{
					Content: []*ai.Part{
						ai.NewTextPart(`"`),
					},
				},
			}, errors.New("internal server error")
		}
		defer func() { genkitGenerate = originalGenerate }()

		_, err := reviewService.analyzeChunk(context.Background(), chunk)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "internal server error")
	})
}

func TestFindLocationForLineContent(t *testing.T) {
	chunk := &diffparser.DiffChunk{
		FilePath:     "main.go",
		StartLineNew: 15,
		CodeSnippet: `@@ -15,4 +15,6 @@
 func main() {
-	// old line
+	// a new line
+	fmt.Println("hello")
+	
+	// another new line
 }`,
	}

	testCases := []struct {
		name          string
		targetContent string
		expectedPos   int
		expectedLine  int
		expectError   bool
	}{
		{"Valid First Added Line", "+	// a new line", 4, 16, false},
		{"Valid Middle Added Line", "+	fmt.Println(\"hello\")", 5, 17, false},
		{"Content Not Found", "+	// this line does not exist", -1, -1, true},
		{"Content is a Context Line", "func main() {", -1, -1, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pos, line, err := findLocationForLineContent(chunk, tc.targetContent)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedPos, pos)
				assert.Equal(t, tc.expectedLine, line)
			}
		})
	}
}
