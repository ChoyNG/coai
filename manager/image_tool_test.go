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
	if args.Prompt != "the same white cat beside a bag of food" || args.Size != "1024x1024" {
		t.Fatalf("unexpected arguments: %#v", args)
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
