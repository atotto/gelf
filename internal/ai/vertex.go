package ai

import (
	"context"
	"fmt"

	"github.com/EkeMinusYou/gelf/internal/config"

	"google.golang.org/genai"
)

type VertexAIClient struct {
	client *genai.Client
	model  string
}

func NewVertexAIClient(ctx context.Context, cfg *config.Config) (*VertexAIClient, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:  cfg.ProjectID,
		Location: cfg.Location,
		Backend:  genai.BackendVertexAI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Vertex AI client: %w", err)
	}

	return &VertexAIClient{
		client: client,
		model:  cfg.Model,
	}, nil
}

func (v *VertexAIClient) GenerateCommitMessage(ctx context.Context, diff string) (string, error) {
	prompt := fmt.Sprintf(`Based on the following git diff, generate a concise and descriptive commit message following the Conventional Commits specification.

REQUIREMENTS:
1. Use English only
2. Follow Conventional Commits format: <type>[optional scope]: <description>
3. Valid types: feat, fix, docs, style, refactor, test, chore, perf, ci, build, revert
4. Keep the entire message under 72 characters
5. Use imperative mood (e.g., "add" not "added" or "adds")
6. Start description with lowercase letter
7. No period at the end
8. Focus on the most significant change if there are multiple changes

EXAMPLES:
- feat: add user authentication system
- fix: resolve memory leak in data processing
- docs: update installation instructions
- refactor: simplify error handling logic
- test: add unit tests for payment module
- chore: update dependencies to latest versions

Git diff:
%s

Respond with only the commit message, no additional text or formatting.`, diff)

	resp, err := v.client.Models.GenerateContent(ctx, v.model, 
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

func (v *VertexAIClient) Close() error {
	return nil
}