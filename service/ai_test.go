package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestExtractResponseText(t *testing.T) {
	result := responsesResponse{
		Output: []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		}{
			{
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}{
					{Type: "output_text", Text: "  A generated post.  "},
				},
			},
		},
	}

	got := extractResponseText(result)
	if got != "A generated post." {
		t.Fatalf("expected trimmed generated text, got %q", got)
	}
}

func TestGeneratePostTextRejectsInvalidPrompt(t *testing.T) {
	_, err := GeneratePostText(context.Background(), strings.Repeat("x", maxPromptLength+1), "alice")
	if !errors.Is(err, ErrInvalidPrompt) {
		t.Fatalf("expected ErrInvalidPrompt, got %v", err)
	}
}

func decodeResponse(t *testing.T, raw string) responsesResponse {
	t.Helper()
	var result responsesResponse
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("failed to decode fixture: %v", err)
	}
	return result
}

func TestTextFromResponse_Completed(t *testing.T) {
	result := decodeResponse(t, `{
		"status": "completed",
		"output": [
			{"type": "reasoning", "content": []},
			{"type": "message", "content": [{"type": "output_text", "text": "Hiked all weekend!"}]}
		]
	}`)

	got, err := textFromResponse(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Hiked all weekend!" {
		t.Fatalf("unexpected text %q", got)
	}
}

// A reasoning model that spends the whole token budget thinking returns an
// incomplete response with only a reasoning item and no output text.
func TestTextFromResponse_IncompleteFromTokenLimit(t *testing.T) {
	result := decodeResponse(t, `{
		"status": "incomplete",
		"incomplete_details": {"reason": "max_output_tokens"},
		"output": [{"type": "reasoning", "content": []}]
	}`)

	_, err := textFromResponse(result)
	if !errors.Is(err, ErrIncompleteOutput) {
		t.Fatalf("expected ErrIncompleteOutput, got %v", err)
	}
	if !strings.Contains(err.Error(), "max_output_tokens") {
		t.Fatalf("expected error to include the incomplete reason, got %v", err)
	}
}

func TestTextFromResponse_NoText(t *testing.T) {
	result := decodeResponse(t, `{"status": "completed", "output": []}`)

	_, err := textFromResponse(result)
	if err == nil || errors.Is(err, ErrIncompleteOutput) {
		t.Fatalf("expected a missing-text error, got %v", err)
	}
}

func TestIsReasoningModel(t *testing.T) {
	cases := map[string]bool{
		"gpt-5":      true,
		"gpt-5-mini": true,
		"o4-mini":    true,
		"gpt-4o":     false,
		"gpt-4.1":    false,
	}
	for model, want := range cases {
		if got := isReasoningModel(model); got != want {
			t.Errorf("isReasoningModel(%q) = %v, want %v", model, got, want)
		}
	}
}
