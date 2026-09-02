package service

import (
	"context"
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
