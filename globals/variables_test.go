package globals

import "testing"

func TestIsOpenAIDalleModel(t *testing.T) {
	cases := map[string]bool{
		"dalle":         true,
		"dall-e-2":      true,
		"dall-e-3":      true,
		"gpt-image-1":   true,
		"gpt-image-1.5": true,
		"gpt-image-2":   true,
		// gpt-image-* must be routed to /v1/images/generations, not chat completions
		"gpt-4-dalle":     false,
		"gpt-4o":          false,
		"gpt-5":           false,
		"gpt-3.5-turbo":   false,
		"deepseek-chat":   false,
		"midjourney":      false,
		"imagen-3.0-generate-002": false,
	}

	for model, expected := range cases {
		if got := IsOpenAIDalleModel(model); got != expected {
			t.Errorf("IsOpenAIDalleModel(%q) = %v, want %v", model, got, expected)
		}
	}
}

func TestIsOpenAIGPTImageModel(t *testing.T) {
	cases := map[string]bool{
		"gpt-image-1":   true,
		"gpt-image-1.5": true,
		"gpt-image-2":   true,
		"dall-e-3":      false,
		"dalle":         false,
		"gpt-4o":        false,
		"gpt-5":         false,
	}

	for model, expected := range cases {
		if got := IsOpenAIGPTImageModel(model); got != expected {
			t.Errorf("IsOpenAIGPTImageModel(%q) = %v, want %v", model, got, expected)
		}
	}
}
