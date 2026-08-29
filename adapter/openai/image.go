package openai

import (
	adaptercommon "chat/adapter/common"
	"chat/globals"
	"chat/utils"
	"fmt"
	"strings"
)

type ImageProps struct {
	Model  string
	Prompt string
	Size   ImageSize
	Proxy  globals.ProxyConfig
}

func (c *ChatInstance) GetImageEndpoint() string {
	return fmt.Sprintf("%s/v1/images/generations", c.GetEndpoint())
}

// CreateImageRequest will create a dalle image from prompt, return url of image, base64 data and error
func (c *ChatInstance) CreateImageRequest(props ImageProps) (string, string, error) {
	res, err := utils.Post(
		c.GetImageEndpoint(),
		c.GetHeader(), ImageRequest{
			Model:  props.Model,
			Prompt: props.Prompt,
			Size: utils.Multi[ImageSize](
				props.Model == globals.Dalle3 || globals.IsOpenAIGPTImageModel(props.Model),
				ImageSize1024,
				ImageSize512,
			),
			N: 1,
		}, props.Proxy)
	if err != nil {
		return "", "", fmt.Errorf("openai image request failed: %w", err)
	}
	if res == nil {
		return "", "", fmt.Errorf("openai image error: upstream returned an empty response")
	}

	data := utils.MapToStruct[ImageResponse](res)
	if data == nil {
		return "", "", fmt.Errorf("openai image error: cannot parse upstream response")
	}
	if data.Error.Message != "" {
		return "", "", fmt.Errorf("openai image error: %s", data.Error.Message)
	}
	if len(data.Data) == 0 {
		return "", "", fmt.Errorf("openai image error: upstream response contains no image data")
	}

	// for gpt-image-1 / gpt-image-1.5 / gpt-image-2, return base64 data if available
	if globals.IsOpenAIGPTImageModel(props.Model) && data.Data[0].B64Json != "" {
		return "", data.Data[0].B64Json, nil
	}

	// fall back to b64_json for any other model returning base64 (e.g. gateways
	// such as Sub2API that normalize every image model to base64)
	if data.Data[0].Url == "" && data.Data[0].B64Json != "" {
		return "", data.Data[0].B64Json, nil
	}
	if data.Data[0].Url == "" {
		return "", "", fmt.Errorf("openai image error: upstream response contains neither url nor b64_json")
	}

	return data.Data[0].Url, "", nil
}

// CreateImage will create a dalle image from prompt, return markdown of image
func (c *ChatInstance) CreateImage(props *adaptercommon.ChatProps) (string, error) {
	url, b64Json, err := c.CreateImageRequest(ImageProps{
		Model:  props.Model,
		Prompt: c.GetLatestPrompt(props),
		Proxy:  props.Proxy,
	})
	if err != nil {
		if strings.Contains(err.Error(), "safety") {
			return err.Error(), nil
		}
		return "", err
	}

	if b64Json != "" {
		return utils.GetBase64ImageMarkdown(b64Json), nil
	}

	storedUrl := utils.StoreImage(url)
	return utils.GetImageMarkdown(storedUrl), nil
}
