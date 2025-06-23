package ai

import (
	"context"
	"fmt"
	"os"

	"github.com/EkeMinusYou/gelf/internal/config"

	"google.golang.org/genai"
)

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

func (v *VertexAIClient) ReviewCode(ctx context.Context, diff string) (string, error) {
	prompt := fmt.Sprintf(`Analyze the following git diff and provide a concise code review.

REVIEW FORMAT:
1. **Overall Summary**: Brief overview of changes (1-2 sentences)
2. **Specific Comments**: Use these prefixes for each comment:
   - **must**: Critical issues that must be fixed
   - **want**: Important suggestions for improvement
   - **nits**: Minor style/formatting issues
   - **fyi**: Informational notes
   - **imo**: Opinion-based suggestions

GUIDELINES:
- Be concise and actionable
- Focus on the most important issues
- Use specific line references when possible
- If no issues found, simply state "No significant issues found"
- Limit to 5-10 comments maximum

EXAMPLE FORMAT:
## Overall Summary
Brief description of what was changed.

## Specific Comments
- **must**: Fix potential null pointer dereference in function_name
- **want**: Consider adding error handling for edge case
- **nits**: Inconsistent spacing in line 42

Git diff:
%s

Provide a concise code review:`, diff)

	resp, err := v.client.Models.GenerateContent(ctx, v.flashModel,
		[]*genai.Content{
			genai.NewUserContentFromText(prompt),
		},
		&genai.GenerateContentConfig{
			Temperature: genai.Ptr(float32(0.2)),
		})
	if err != nil {
		return "", fmt.Errorf("failed to generate code review: %w", err)
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

func (v *VertexAIClient) ReviewCodeStreaming(ctx context.Context, diff string, callback func(string)) error {
	prompt := fmt.Sprintf(`Analyze the following git diff and provide a concise code review.

REVIEW FORMAT:
1. **Overall Summary**: Brief overview of changes (1-2 sentences)
2. **Specific Comments**: Use these prefixes for each comment:
   - **must**: Critical issues that must be fixed
   - **want**: Important suggestions for improvement
   - **nits**: Minor style/formatting issues
   - **fyi**: Informational notes
   - **imo**: Opinion-based suggestions

GUIDELINES:
- Be concise and actionable
- Focus on the most important issues
- Use specific line references when possible
- If no issues found, simply state "No significant issues found"
- Limit to 5-10 comments maximum

EXAMPLE FORMAT:
## Overall Summary
Brief description of what was changed.

## Specific Comments
- **must**: Fix potential null pointer dereference in function_name
- **want**: Consider adding error handling for edge case
- **nits**: Inconsistent spacing in line 42

Git diff:
%s

Provide a concise code review:`, diff)

	iter := v.client.Models.GenerateContentStream(ctx, v.flashModel,
		[]*genai.Content{
			genai.NewUserContentFromText(prompt),
		},
		&genai.GenerateContentConfig{
			Temperature: genai.Ptr(float32(0.2)),
		})

	for resp, err := range iter {
		if err != nil {
			return fmt.Errorf("failed to get streaming response: %w", err)
		}

		if len(resp.Candidates) == 0 {
			continue
		}

		if len(resp.Candidates[0].Content.Parts) == 0 {
			continue
		}

		part := resp.Candidates[0].Content.Parts[0]
		if part.Text != "" {
			callback(part.Text)
		}
	}

	return nil
}

func (v *VertexAIClient) Close() error {
	return nil
}