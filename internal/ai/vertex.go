package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/EkeMinusYou/gelf/internal/config"

	"google.golang.org/genai"
)

// ReviewComment represents a single review comment
type ReviewComment struct {
	FileName string `json:"fileName"`
	LineNo   int    `json:"lineNo,omitempty"`
	Type     string `json:"type"` // must, want, nits, fyi, imo
	Message  string `json:"message"`
}

// FileReview represents review data for a single file
type FileReview struct {
	FileName string          `json:"fileName"`
	DiffText string          `json:"diffText"`
	Comments []ReviewComment `json:"comments"`
}

// StructuredReview represents the complete review data
type StructuredReview struct {
	Summary     string       `json:"summary"`
	FileReviews []FileReview `json:"fileReviews"`
}

type VertexAIClient struct {
	client     *genai.Client
	flashModel string
	proModel   string
}

func NewVertexAIClient(ctx context.Context, cfg *config.Config) (*VertexAIClient, error) {
	// Check for GELF_CREDENTIALS first, then fall back to GOOGLE_APPLICATION_CREDENTIALS
	credentialsPath := os.Getenv("GELF_CREDENTIALS")
	if credentialsPath == "" {
		credentialsPath = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	}

	// If we have a credentials path, set it as GOOGLE_APPLICATION_CREDENTIALS
	// since genai.NewClient uses this environment variable internally
	if credentialsPath != "" {
		originalCreds := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", credentialsPath)

		// Restore original credentials after client creation
		defer func() {
			if originalCreds != "" {
				os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", originalCreds)
			} else {
				os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")
			}
		}()
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  cfg.ProjectID,
		Location: cfg.Location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Vertex AI client: %w", err)
	}

	return &VertexAIClient{
		client:     client,
		flashModel: cfg.FlashModel,
		proModel:   cfg.ProModel,
	}, nil
}

func (v *VertexAIClient) GenerateCommitMessage(ctx context.Context, diff string) (string, error) {
	prompt := fmt.Sprintf(`Analyze the following git diff and generate a precise commit message following the Conventional Commits specification.

DIFF ANALYSIS GUIDE:
1. Look at file paths to understand what parts of the codebase are affected
2. Examine +/- lines to understand what was added, removed, or modified
3. Pay attention to function names, variable names, and code structure changes
4. Consider the context lines (prefixed with space) to understand the surrounding code
5. Identify the primary purpose: new feature, bug fix, refactoring, etc.

COMMIT MESSAGE REQUIREMENTS:
1. Use English only
2. Follow format: <type>[optional scope]: <description>
3. Valid types: feat, fix, docs, style, refactor, test, chore, perf, ci, build, revert
4. Keep under 72 characters total
5. Use imperative mood ("add" not "added")
6. Start description with lowercase letter
7. No period at the end
8. If multiple changes, focus on the most significant one
9. Use scope when it helps clarify the area of change (e.g., auth, api, ui)

EXAMPLES:
- feat(auth): add JWT token validation
- fix(api): resolve null pointer in user service
- refactor(db): simplify connection pooling logic
- test(payment): add unit tests for stripe integration
- chore(deps): update react to version 18.2.0

Git diff:
%s

Respond with only the commit message, no additional text or formatting.`, diff)

	resp, err := v.client.Models.GenerateContent(ctx, v.flashModel,
		[]*genai.Content{
			genai.NewUserContentFromText(prompt),
		},
		&genai.GenerateContentConfig{
			Temperature: genai.Ptr(float32(0.3)),
		})
	if err != nil {
		return "", fmt.Errorf("failed to generate commit message: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}

	if len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content parts in response")
	}

	part := resp.Candidates[0].Content.Parts[0]
	if part.Text == "" {
		return "", fmt.Errorf("empty text in response part")
	}

	return part.Text, nil
}

// ReviewCodeStructured generates a structured review with file-specific comments
func (v *VertexAIClient) ReviewCodeStructured(ctx context.Context, diff string) (*StructuredReview, error) {
	// First, parse the diff to extract file information
	files := v.parseDiffFiles(diff)

	var fileReviews []FileReview
	var allComments []ReviewComment

	for _, file := range files {
		// Generate review for each file individually
		fileReview, err := v.reviewSingleFile(ctx, file.fileName, file.diffText)
		if err != nil {
			// Continue with other files if one fails
			continue
		}

		fileReviews = append(fileReviews, *fileReview)
		allComments = append(allComments, fileReview.Comments...)
	}

	// Generate overall summary
	summary, err := v.generateReviewSummary(ctx, diff, allComments)
	if err != nil {
		summary = "Failed to generate summary"
	}

	return &StructuredReview{
		Summary:     summary,
		FileReviews: fileReviews,
	}, nil
}

// parseDiffFiles extracts file information from git diff
func (v *VertexAIClient) parseDiffFiles(diff string) []struct {
	fileName string
	diffText string
} {
	var files []struct {
		fileName string
		diffText string
	}

	lines := strings.Split(diff, "\n")
	var currentFile string
	var currentDiff []string

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") {
			// Save previous file if exists
			if currentFile != "" && len(currentDiff) > 0 {
				files = append(files, struct {
					fileName string
					diffText string
				}{
					fileName: currentFile,
					diffText: strings.Join(currentDiff, "\n"),
				})
			}

			// Extract filename from "diff --git a/file b/file"
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				currentFile = strings.TrimPrefix(parts[3], "b/")
			}
			currentDiff = []string{line}
		} else if currentFile != "" {
			currentDiff = append(currentDiff, line)
		}
	}

	// Add the last file
	if currentFile != "" && len(currentDiff) > 0 {
		files = append(files, struct {
			fileName string
			diffText string
		}{
			fileName: currentFile,
			diffText: strings.Join(currentDiff, "\n"),
		})
	}

	return files
}

// reviewSingleFile generates review for a single file
func (v *VertexAIClient) reviewSingleFile(ctx context.Context, fileName, diffText string) (*FileReview, error) {
	prompt := fmt.Sprintf(`Analyze the following git diff for file "%s" and provide specific code review comments.

RESPONSE REQUIREMENTS:
1. Respond with ONLY a valid JSON object
2. No markdown formatting, no code blocks, no additional text
3. Use this exact structure:

{
  "comments": [
    {
      "fileName": "%s",
      "lineNo": 42,
      "type": "must",
      "message": "Fix potential null pointer dereference"
    }
  ]
}

COMMENT TYPES:
- "must": Critical issues that must be fixed
- "want": Important suggestions for improvement  
- "nits": Minor style/formatting issues
- "fyi": Informational notes
- "imo": Opinion-based suggestions

GUIDELINES:
- Focus on the most important issues only
- Be specific and actionable
- Include line numbers when possible (use approximate line numbers from diff context)
- Maximum 5 comments per file
- If no issues, return: {"comments": []}

File diff:
%s`, fileName, fileName, diffText)

	resp, err := v.client.Models.GenerateContent(ctx, v.flashModel,
		[]*genai.Content{
			genai.NewUserContentFromText(prompt),
		},
		&genai.GenerateContentConfig{
			Temperature: genai.Ptr(float32(0.1)),
		})
	if err != nil {
		return nil, fmt.Errorf("failed to generate file review: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	if len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content parts in response")
	}

	part := resp.Candidates[0].Content.Parts[0]
	if part.Text == "" {
		return nil, fmt.Errorf("empty text in response part")
	}

	// Parse JSON response
	var result struct {
		Comments []ReviewComment `json:"comments"`
	}

	if err := json.Unmarshal([]byte(part.Text), &result); err != nil {
		// Fallback: try to extract JSON from response if it's wrapped in markdown
		text := strings.TrimSpace(part.Text)
		if strings.HasPrefix(text, "```json") {
			text = strings.TrimPrefix(text, "```json")
			text = strings.TrimSuffix(text, "```")
			text = strings.TrimSpace(text)
		}

		if err := json.Unmarshal([]byte(text), &result); err != nil {
			return nil, fmt.Errorf("failed to parse JSON response: %w", err)
		}
	}

	return &FileReview{
		FileName: fileName,
		DiffText: diffText,
		Comments: result.Comments,
	}, nil
}

// generateReviewSummary creates an overall summary of the review
func (v *VertexAIClient) generateReviewSummary(ctx context.Context, diff string, comments []ReviewComment) (string, error) {
	if len(comments) == 0 {
		return "No significant issues found in the code changes.", nil
	}

	mustCount := 0
	wantCount := 0
	nitsCount := 0

	for _, comment := range comments {
		switch comment.Type {
		case "must":
			mustCount++
		case "want":
			wantCount++
		case "nits":
			nitsCount++
		}
	}

	prompt := fmt.Sprintf(`Based on the following code review findings, generate a brief summary (1-2 sentences):

FINDINGS:
- Critical issues (must fix): %d
- Important suggestions (want): %d  
- Minor issues (nits): %d
- Total comments: %d

Provide a concise summary of the overall code quality and main areas of concern.`,
		mustCount, wantCount, nitsCount, len(comments))

	resp, err := v.client.Models.GenerateContent(ctx, v.flashModel,
		[]*genai.Content{
			genai.NewUserContentFromText(prompt),
		},
		&genai.GenerateContentConfig{
			Temperature: genai.Ptr(float32(0.3)),
		})
	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}

	if len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content parts in response")
	}

	part := resp.Candidates[0].Content.Parts[0]
	return strings.TrimSpace(part.Text), nil
}

func (v *VertexAIClient) Close() error {
	return nil
}
