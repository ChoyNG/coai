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
	Prompt  string `json:"prompt"`
	Size    string `json:"size,omitempty"`
	Quality string `json:"quality,omitempty"`
}

var supportedImageSizes = []string{"1024x1024", "1024x1536", "1536x1024"}
var supportedImageQualities = []string{"low", "medium", "high"}

func normalizeImageSize(size string) string {
	size = strings.TrimSpace(strings.ToLower(size))
	if utils.Contains(size, supportedImageSizes) {
		return size
	}
	return "1024x1024"
}

func normalizeImageQuality(quality string) string {
	quality = strings.TrimSpace(strings.ToLower(quality))
	if utils.Contains(quality, supportedImageQualities) {
		return quality
	}
	return "medium"
}

func imageRequestPrice(model, size, quality string) float32 {
	if model != globals.GPTImage2 {
		return 0
	}
	size = normalizeImageSize(size)
	quality = normalizeImageQuality(quality)
	prices := map[string]map[string]float32{
		"low":    {"1024x1024": 0.006, "1024x1536": 0.005, "1536x1024": 0.005},
		"medium": {"1024x1024": 0.053, "1024x1536": 0.041, "1536x1024": 0.041},
		"high":   {"1024x1024": 0.211, "1024x1536": 0.165, "1536x1024": 0.165},
	}
	return prices[quality][size]
}

func imageRequestCharge(model, size, quality string) *channel.Charge {
	configured := channel.ChargeInstance.GetCharge(model)
	if configured == nil || !configured.IsBillingType(globals.TimesBilling) || model != globals.GPTImage2 {
		return configured
	}
	charge := *configured
	price := imageRequestPrice(model, size, quality)
	if price <= 0 {
		return &charge
	}
	// Preserve any administrator-defined margin relative to the documented
	// medium-square base price instead of silently replacing their rate card.
	margin := configured.GetOutput() / 0.053
	if margin <= 0 {
		margin = 1
	}
	charge.Output = price * margin
	return &charge
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
							"description": "Requested output size: square, portrait, or landscape. Use 1024x1024 when unspecified.",
							"enum":        supportedImageSizes,
						},
						"quality": globals.ToolProperty{
							"type":        "string",
							"description": "Rendering quality. Use medium unless the user explicitly asks for a draft/low quality or high quality.",
							"enum":        supportedImageQualities,
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
		args.Size = normalizeImageSize(args.Size)
		args.Quality = normalizeImageQuality(args.Quality)
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

	buffer := utils.NewBuffer(model, messages, imageRequestCharge(model, args.Size, args.Quality))
	hit, err := channel.NewChatRequestWithCache(
		cache,
		buffer,
		auth.GetGroup(db, user),
		adaptercommon.CreateChatProps(&adaptercommon.ChatProps{
			Model: model, Message: messages, ImageSize: args.Size, ImageQuality: args.Quality,
		}, buffer),
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
