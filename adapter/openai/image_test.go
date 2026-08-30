package openai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"chat/globals"
)

// newImageServer returns a test server that records the request path/body and
// replies with the given payload.
func newImageServer(t *testing.T, path *string, body *ImageRequest, response interface{}) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*path = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(body); err != nil {
			t.Errorf("cannot decode request body: %s", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
}

func TestCreateImageRequestRejectsEmptyDataWithoutPanic(t *testing.T) {
	var path string
	var body ImageRequest
	instance := NewChatInstance("", "sk-test")
	server := newImageServer(t, &path, &body, map[string]interface{}{"data": []interface{}{}})
	instance.Endpoint = server.URL
	defer server.Close()

	_, _, err := instance.CreateImageRequest(ImageProps{Model: globals.GPTImage2, Prompt: "cat"})
	if err == nil || !strings.Contains(err.Error(), "no image data") {
		t.Fatalf("error = %v, want clear empty-data error", err)
	}
}

func TestCreateImageRequestReportsOpenAIError(t *testing.T) {
	var path string
	var body ImageRequest
	instance := NewChatInstance("", "sk-test")
	server := newImageServer(t, &path, &body, map[string]interface{}{
		"error": map[string]string{"message": "no available account"},
	})
	instance.Endpoint = server.URL
	defer server.Close()

	_, _, err := instance.CreateImageRequest(ImageProps{Model: globals.GPTImage2, Prompt: "cat"})
	if err == nil || !strings.Contains(err.Error(), "no available account") {
		t.Fatalf("error = %v, want upstream error message", err)
	}
}

func TestCreateImageRequestReportsTopLevelGatewayError(t *testing.T) {
	var path string
	var body ImageRequest
	instance := NewChatInstance("", "sk-test")
	server := newImageServer(t, &path, &body, map[string]interface{}{
		"code":    "INSUFFICIENT_BALANCE",
		"message": "Insufficient account balance",
	})
	instance.Endpoint = server.URL
	defer server.Close()

	_, _, err := instance.CreateImageRequest(ImageProps{Model: globals.GPTImage2, Prompt: "cat"})
	if err == nil || !strings.Contains(err.Error(), "Insufficient account balance (INSUFFICIENT_BALANCE)") {
		t.Fatalf("error = %v, want top-level gateway error", err)
	}
}

func TestCreateImageRequestRejectsImageWithoutPayload(t *testing.T) {
	var path string
	var body ImageRequest
	instance := NewChatInstance("", "sk-test")
	server := newImageServer(t, &path, &body, map[string]interface{}{
		"data": []map[string]string{{}},
	})
	instance.Endpoint = server.URL
	defer server.Close()

	_, _, err := instance.CreateImageRequest(ImageProps{Model: globals.GPTImage2, Prompt: "cat"})
	if err == nil || !strings.Contains(err.Error(), "neither url nor b64_json") {
		t.Fatalf("error = %v, want missing-payload error", err)
	}
}

// TestCreateImageRequestGPTImage asserts that gpt-image-2 is sent to the image
// generations endpoint (not /v1/chat/completions) and that its b64_json payload
// is returned as base64 data.
func TestCreateImageRequestGPTImage(t *testing.T) {
	for _, model := range []string{globals.GPTImage1, globals.GPTImage15, globals.GPTImage2} {
		var path string
		var body ImageRequest

		instance := NewChatInstance("", "sk-test")

		server := newImageServer(t, &path, &body, map[string]interface{}{
			"data": []map[string]string{{"b64_json": "aGVsbG8="}},
		})
		instance.Endpoint = server.URL
		defer server.Close()

		url, b64, err := instance.CreateImageRequest(ImageProps{
			Model:  model,
			Prompt: "a fluffy cat",
		})
		if err != nil {
			t.Fatalf("[%s] unexpected error: %s", model, err)
		}

		if want := "/v1/images/generations"; path != want {
			t.Errorf("[%s] request path = %q, want %q", model, path, want)
		}

		if b64 != "aGVsbG8=" {
			t.Errorf("[%s] b64_json = %q, want %q", model, b64, "aGVsbG8=")
		}

		if url != "" {
			t.Errorf("[%s] url = %q, want empty", model, url)
		}

		if body.Model != model {
			t.Errorf("[%s] request model = %q, want %q", model, body.Model, model)
		}

		// gpt-image-* only accepts 1024x1024 in this MVP
		if body.Size != ImageSize1024 {
			t.Errorf("[%s] request size = %q, want %q", model, body.Size, ImageSize1024)
		}

		if body.Quality != "medium" {
			t.Errorf("[%s] request quality = %q, want %q", model, body.Quality, "medium")
		}
	}
}

// TestCreateImageRequestURLModel keeps dall-e compatible: url responses must
// still be returned as url.
func TestCreateImageRequestURLModel(t *testing.T) {
	var path string
	var body ImageRequest

	instance := NewChatInstance("", "sk-test")

	server := newImageServer(t, &path, &body, map[string]interface{}{
		"data": []map[string]string{{"url": "https://example.com/cat.png"}},
	})
	instance.Endpoint = server.URL
	defer server.Close()

	url, b64, err := instance.CreateImageRequest(ImageProps{
		Model:  globals.Dalle3,
		Prompt: "a fluffy cat",
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if url != "https://example.com/cat.png" {
		t.Errorf("url = %q, want %q", url, "https://example.com/cat.png")
	}

	if b64 != "" {
		t.Errorf("b64_json = %q, want empty", b64)
	}

	if path != "/v1/images/generations" {
		t.Errorf("request path = %q, want %q", path, "/v1/images/generations")
	}

	if body.Quality != "" {
		t.Errorf("dall-e request quality = %q, want omitted", body.Quality)
	}
}

// TestGetImageEndpoint makes sure the endpoint has no trailing-slash surprises
// for gateways configured as http://host:8080 (without /v1).
func TestGetImageEndpoint(t *testing.T) {
	instance := NewChatInstance("http://host.docker.internal:8080", "sk-test")

	want := "http://host.docker.internal:8080/v1/images/generations"
	if got := instance.GetImageEndpoint(); got != want {
		t.Errorf("GetImageEndpoint() = %q, want %q", got, want)
	}
}
