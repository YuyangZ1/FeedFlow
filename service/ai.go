package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultOpenAIModel = "gpt-5"
	maxPromptLength    = 1000

	// Reasoning tokens count toward max_output_tokens, so the budget has to
	// leave room for them on top of the ~500-character post itself.
	maxOutputTokens = 1000
)

var (
	ErrMissingOpenAIKey = errors.New("OPENAI_API_KEY environment variable must be set")
	ErrInvalidPrompt    = errors.New("prompt must be between 1 and 1000 characters")
	ErrIncompleteOutput = errors.New("openai response was cut off before the post text was finished")
)

type responsesRequest struct {
	Model           string     `json:"model"`
	Input           string     `json:"input"`
	MaxOutputTokens int        `json:"max_output_tokens"`
	Store           bool       `json:"store"`
	Reasoning       *reasoning `json:"reasoning,omitempty"`
}

type reasoning struct {
	Effort string `json:"effort"`
}

type responsesResponse struct {
	Status            string `json:"status"`
	IncompleteDetails *struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
	Output []struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

type openAIErrorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func GeneratePostText(ctx context.Context, prompt, username string) (string, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" || len(prompt) > maxPromptLength {
		return "", ErrInvalidPrompt
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", ErrMissingOpenAIKey
	}

	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = defaultOpenAIModel
	}

	input := fmt.Sprintf(
		"Write one polished social media post for user %q based on this idea: %s\n\nReturn only the post text. Keep it under 500 characters. Do not include hashtags unless they naturally fit.",
		username,
		prompt,
	)
	reqBody := responsesRequest{
		Model:           model,
		Input:           input,
		MaxOutputTokens: maxOutputTokens,
		Store:           false,
	}
	// Writing a short post needs little reasoning. Non-reasoning models
	// (e.g. gpt-4o via OPENAI_MODEL) reject the parameter, so only send it
	// to reasoning models.
	if isReasoningModel(model) {
		reqBody.Reasoning = &reasoning{Effort: "low"}
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	reqCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, "https://api.openai.com/v1/responses", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr openAIErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr.Error.Message != "" {
			return "", fmt.Errorf("openai request failed: %s", apiErr.Error.Message)
		}
		return "", fmt.Errorf("openai request failed with status %d", resp.StatusCode)
	}

	var result responsesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return textFromResponse(result)
}

func textFromResponse(result responsesResponse) (string, error) {
	if result.Status == "incomplete" {
		reason := "unknown"
		if result.IncompleteDetails != nil && result.IncompleteDetails.Reason != "" {
			reason = result.IncompleteDetails.Reason
		}
		return "", fmt.Errorf("%w (reason: %s)", ErrIncompleteOutput, reason)
	}

	text := extractResponseText(result)
	if text == "" {
		return "", errors.New("openai response did not include post text")
	}
	return text, nil
}

func isReasoningModel(model string) bool {
	return strings.HasPrefix(model, "gpt-5") || strings.HasPrefix(model, "o")
}

func extractResponseText(result responsesResponse) string {
	for _, output := range result.Output {
		for _, content := range output.Content {
			if content.Type == "output_text" && strings.TrimSpace(content.Text) != "" {
				return strings.TrimSpace(content.Text)
			}
		}
	}
	return ""
}
