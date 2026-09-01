package manager

import (
	"chat/globals"
	"strings"
	"testing"
)

func TestGetGenerateImageArguments(t *testing.T) {
	calls := globals.ToolCalls{{
		Type: "function",
		Function: globals.ToolCallFunction{
			Name:      generateImageToolName,
			Arguments: `{"prompt":"the same white cat beside a bag of food"}`,
		},
	}}
	args, found, err := getGenerateImageArguments(&calls)
	if err != nil {
		t.Fatal(err)
	}
	if !found || args == nil {
		t.Fatal("generate_image call was not found")
	}
	if args.Prompt != "the same white cat beside a bag of food" || args.Size != "1024x1024" || args.Quality != "medium" {
		t.Fatalf("unexpected arguments: %#v", args)
	}
}

func TestGetGenerateImageArgumentsKeepsPortraitAndHighQuality(t *testing.T) {
	calls := globals.ToolCalls{{
		Function: globals.ToolCallFunction{
			Name: generateImageToolName,
			Arguments: `{"prompt":"phone wallpaper","size":"1024x1536","quality":"high"}`,
		},
	}}
	args, found, err := getGenerateImageArguments(&calls)
	if err != nil || !found {
		t.Fatalf("found=%v error=%v", found, err)
	}
	if args.Size != "1024x1536" || args.Quality != "high" {
		t.Fatalf("unexpected arguments: %#v", args)
	}
}

func TestImageRequestPrice(t *testing.T) {
	tests := []struct {
		size, quality string
		want          float32
	}{
		{"1024x1024", "low", 0.006},
		{"1024x1024", "medium", 0.053},
		{"1024x1536", "medium", 0.041},
		{"1536x1024", "high", 0.165},
	}
	for _, tt := range tests {
		if got := imageRequestPrice(globals.GPTImage2, tt.size, tt.quality); got != tt.want {
			t.Errorf("price(%s, %s) = %v, want %v", tt.size, tt.quality, got, tt.want)
		}
	}
}

func TestGetGenerateImageArgumentsRejectsEmptyPrompt(t *testing.T) {
	calls := globals.ToolCalls{{
		Function: globals.ToolCallFunction{Name: generateImageToolName, Arguments: `{"prompt":" "}`},
	}}
	_, found, err := getGenerateImageArguments(&calls)
	if !found || err == nil || !strings.Contains(err.Error(), "prompt is empty") {
		t.Fatalf("found=%v error=%v, want empty-prompt error", found, err)
	}
}
