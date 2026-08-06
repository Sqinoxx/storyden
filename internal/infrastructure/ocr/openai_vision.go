package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Southclaws/fault"
)

type OpenAIVisionClient struct {
	apiKey string
}

func NewOpenAIVisionClient(apiKey string) *OpenAIVisionClient {
	return &OpenAIVisionClient{apiKey: apiKey}
}

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

func (o *OpenAIVisionClient) ExtractText(ctx context.Context, r io.Reader, mimeType string) (string, error) {
	if o.apiKey == "" {
		return "", fault.New("OpenAI API key is missing for Vision OCR")
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return "", fault.Wrap(err)
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
		return "", fault.Wrap(err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(jsonBytes))
	if err != nil {
		return "", fault.Wrap(err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fault.Wrap(err)
	}
	defer resp.Body.Close()

	var chatResp openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fault.Wrap(err)
	}

	if chatResp.Error != nil {
		return "", fault.New(chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", nil
	}

	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}
