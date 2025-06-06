package ai

import (
	"context"
	"fmt"

	"geminielf/internal/config"

	"cloud.google.com/go/vertexai/genai"
)

type VertexAIClient struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

func NewVertexAIClient(ctx context.Context, cfg *config.Config) (*VertexAIClient, error) {
	client, err := genai.NewClient(ctx, cfg.ProjectID, cfg.Location)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vertex AI client: %w", err)
	}

	model := client.GenerativeModel(cfg.Model)
	model.SetTemperature(0.3)
	model.SetMaxOutputTokens(1000)
	
	// Set safety settings to allow more content
	model.SafetySettings = []*genai.SafetySetting{
		{
			Category:  genai.HarmCategoryHarassment,
			Threshold: genai.HarmBlockOnlyHigh,
		},
		{
			Category:  genai.HarmCategoryHateSpeech,
			Threshold: genai.HarmBlockOnlyHigh,
		},
		{
			Category:  genai.HarmCategorySexuallyExplicit,
			Threshold: genai.HarmBlockOnlyHigh,
		},
		{
			Category:  genai.HarmCategoryDangerousContent,
			Threshold: genai.HarmBlockOnlyHigh,
		},
	}

	return &VertexAIClient{
		client: client,
		model:  model,
	}, nil
}

func (v *VertexAIClient) GenerateCommitMessage(ctx context.Context, diff string) (string, error) {
	prompt := fmt.Sprintf(`Based on the following git diff, generate a concise and descriptive commit message following conventional commit format.
The message should be in Japanese and explain what was changed and why.
Keep it under 80 characters and focus on the most significant changes.

Git diff:
%s

Respond with only the commit message, no additional text or formatting.`, diff)

	resp, err := v.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate commit message: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}
	
	if len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content parts in response")
	}

	return fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]), nil
}

func (v *VertexAIClient) Close() error {
	return v.client.Close()
}