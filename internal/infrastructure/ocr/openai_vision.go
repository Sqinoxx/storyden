package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
)

// OpenAIVisionClient extracts text from images using the OpenAI Vision API.
// It only ever handles image MIME types — PDF handling belongs to the
// layered client, which rasterises pages before handing them here.
type OpenAIVisionClient struct {
	apiKey string
}

func NewOpenAIVisionClient(apiKey string) *OpenAIVisionClient {
	return &OpenAIVisionClient{apiKey: apiKey}
}

func (o *OpenAIVisionClient) Available() bool { return o.apiKey != "" }

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content []any  `json:"content"`
}

type openAIChatRequest struct {
	Model     string              `json:"model"`
	Messages  []openAIChatMessage `json:"messages"`
	MaxTokens int                 `json:"max_tokens"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (o *OpenAIVisionClient) ExtractText(ctx context.Context, data []byte, mimeType string) (Result, error) {
	if !strings.HasPrefix(strings.ToLower(mimeType), "image/") {
		return Result{}, fault.Wrap(ErrUnsupportedMIME, fctx.With(ctx))
	}

	if o.apiKey == "" {
		return Result{}, fault.Wrap(ErrEngineUnavailable, fctx.With(ctx))
	}

	base64Image := base64.StdEncoding.EncodeToString(data)
	dataURI := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Image)

	reqBody := openAIChatRequest{
		Model: "gpt-4o-mini",
		Messages: []openAIChatMessage{
			{
				Role: "user",
				Content: []any{
					map[string]any{"type": "text", "text": "Extract and transcribe all text contained in this image accurately. Output only the extracted text and nothing else."},
					map[string]any{"type": "image_url", "image_url": map[string]string{"url": dataURI}},
				},
			},
		},
		MaxTokens: 1000,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return Result{}, fault.Wrap(err, fctx.With(ctx))
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(jsonBytes))
	if err != nil {
		return Result{}, fault.Wrap(err, fctx.With(ctx))
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return Result{}, fault.Wrap(err, fctx.With(ctx))
	}
	defer resp.Body.Close()

	var chatResp openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return Result{}, fault.Wrap(err, fctx.With(ctx))
	}

	if chatResp.Error != nil {
		return Result{}, fault.New(chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return Result{Engine: "openai-vision"}, nil
	}

	return Result{
		Text:   strings.TrimSpace(chatResp.Choices[0].Message.Content),
		Engine: "openai-vision",
	}, nil
}
