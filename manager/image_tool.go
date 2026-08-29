package manager

import (
	adaptercommon "chat/adapter/common"
	"chat/admin"
	"chat/auth"
	"chat/channel"
	"chat/globals"
	"chat/utils"
	"encoding/json"
	"fmt"
	"strings"
)

const generateImageToolName = "generate_image"

type generateImageArguments struct {
	Prompt string `json:"prompt"`
	Size   string `json:"size,omitempty"`
}

func selectImageToolModel() string {
	for _, model := range []string{globals.GPTImage2, globals.GPTImage15, globals.GPTImage1} {
		if channel.ConduitInstance.HasChannel(model) {
			return model
		}
	}
	return ""
}

func getImageGenerationTools(model string) *globals.FunctionTools {
	if globals.IsOpenAIDalleModel(model) || selectImageToolModel() == "" {
		return nil
	}
	required := []string{"prompt"}
	tools := globals.FunctionTools{
		{
			Type: "function",
			Function: globals.ToolFunction{
				Name:        generateImageToolName,
				Description: "Generate an image when the user asks to create, draw, render, revise, or visually modify an image. The prompt must be self-contained and preserve relevant visual details from the conversation.",
				Parameters: globals.ToolParameters{
					Type: "object",
					Properties: globals.ToolProperties{
						"prompt": globals.ToolProperty{
							"type":        "string",
							"description": "A complete standalone image prompt incorporating relevant details and requested changes from the conversation.",
						},
						"size": globals.ToolProperty{
							"type":        "string",
							"description": "Requested output size. Use 1024x1024 when unspecified.",
							"enum":        []string{"1024x1024"},
						},
					},
					Required: &required,
				},
			},
		},
	}
	return &tools
}

func getGenerateImageArguments(calls *globals.ToolCalls) (*generateImageArguments, bool, error) {
	if calls == nil {
		return nil, false, nil
	}
	for _, call := range *calls {
		if call.Function.Name != generateImageToolName {
			continue
		}
		var args generateImageArguments
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			return nil, true, fmt.Errorf("invalid generate_image arguments: %w", err)
		}
		args.Prompt = strings.TrimSpace(args.Prompt)
		if args.Prompt == "" {
			return nil, true, fmt.Errorf("generate_image prompt is empty")
		}
		if args.Size == "" {
			args.Size = "1024x1024"
		}
		return &args, true, nil
	}
	return nil, false, nil
}

func executeGenerateImageTool(conn *Connection, user *auth.User, args *generateImageArguments) (string, float32, bool, error) {
	model := selectImageToolModel()
	if model == "" {
		return "", 0, false, fmt.Errorf("no image generation model is configured")
	}

	db := conn.GetDB()
	cache := conn.GetCache()
	messages := []globals.Message{{Role: globals.User, Content: args.Prompt}}
	check, plan := auth.CanEnableModelWithSubscription(db, cache, user, model, messages)
	if check != nil {
		return "", 0, plan, check
	}

	buffer := utils.NewBuffer(model, messages, channel.ChargeInstance.GetCharge(model))
	hit, err := channel.NewChatRequestWithCache(
		cache,
		buffer,
		auth.GetGroup(db, user),
		adaptercommon.CreateChatProps(&adaptercommon.ChatProps{Model: model, Message: messages}, buffer),
		func(data *globals.Chunk) error {
			buffer.WriteChunk(data)
			return nil
		},
	)
	admin.AnalyseRequest(model, buffer, err)
	if err != nil {
		auth.RevertSubscriptionUsage(db, cache, user, model)
		return "", 0, plan, err
	}
	if !hit {
		CollectQuota(conn.GetCtx(), user, buffer, plan, nil)
	}

	imageMarkdown := buffer.ReadWithDefault("")
	if imageMarkdown == "" {
		return "", 0, plan, fmt.Errorf("image model returned an empty response")
	}
	content := fmt.Sprintf("Generated image using this prompt: %s\n\n%s", args.Prompt, imageMarkdown)
	return content, buffer.GetQuota(), plan, nil
}
