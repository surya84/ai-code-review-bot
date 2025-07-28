package utils

import (
	"code-reviewer-bot/internal/models"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

func CloneRepoIfNotExists(baseURL, token, owner, repo string) (string, error) {
	localPath := fmt.Sprintf("./repos/%s_%s", owner, repo)
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		cloneURL := fmt.Sprintf("https://x-access-token:%s@%s/%s/%s.git", token, baseURL, owner, repo)
		fmt.Println("Cloning:", cloneURL)
		cmd := exec.Command("git", "clone", cloneURL, localPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Printf("Error cloning repository: %v , %v ", err, cmd)
			return "", err
		}
	}
	return localPath, nil
}

func FormatArchitectureReviewComment(arch *models.ArchitectureReviewResponse) string {
	var b strings.Builder

	b.WriteString("### 🧱 **Project Architecture Review**\n\n")
	b.WriteString(fmt.Sprintf("**Score:** `%d/10`\n\n", arch.Score))

	// Detected Layers
	b.WriteString("#### 📦 Detected Layers\n")
	b.WriteString("| Layer | File Count |\n|-------|-------------|\n")
	for layer, files := range arch.FoundLayers {
		b.WriteString(fmt.Sprintf("| %s | %d file(s) |\n", layer, len(files)))
	}
	b.WriteString("\n")

	// Missing Layers
	if len(arch.MissingLayers) > 0 {
		b.WriteString("#### ❌ Missing Layers\n")
		for _, layer := range arch.MissingLayers {
			b.WriteString(fmt.Sprintf("- [ ] %s\n", layer))
		}
		b.WriteString("\n")
	}

	// Feedback
	// if arch.Feedback != "" {
	// 	b.WriteString("#### 💬 Feedback\n")
	// 	b.WriteString(fmt.Sprintf("> %s\n\n", arch.Feedback))
	// }

	// Detailed Comments
	if len(arch.Comments) > 0 {
		b.WriteString("<details>\n<summary>📝 Detailed Comments</summary>\n\n")
		for i, comment := range arch.Comments {
			// 🔥 FIX: Convert literal \n into actual line breaks
			formatted := cleanCommentBody(comment.Body)
			b.WriteString(fmt.Sprintf("**%d.** %s\n\n", i+1, formatted))
		}
		b.WriteString("</details>\n")
	}

	return b.String()
}

func cleanCommentBody(body string) string {
	// Replace escaped newlines with actual newlines
	cleaned := strings.ReplaceAll(body, `\n`, "\n")

	// Remove JSON code block markers
	cleaned = strings.ReplaceAll(cleaned, "```json", "")
	cleaned = strings.ReplaceAll(cleaned, "```", "")

	// Remove leading/trailing quotes if present
	cleaned = strings.Trim(cleaned, `"`)

	// Remove extra spaces and normalize whitespace
	lines := strings.Split(cleaned, "\n")
	var cleanedLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanedLines = append(cleanedLines, line)
		}
	}

	return strings.Join(cleanedLines, "\n")
}
