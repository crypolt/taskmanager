package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAICompatible talks to any /v1/chat/completions endpoint:
// Ollama, vLLM, LM Studio, OpenRouter, OpenAI itself.
type OpenAICompatible struct {
	ProviderName string
	BaseURL      string
	APIKey       string
	Client       *http.Client
}

func (o *OpenAICompatible) Name() string { return o.ProviderName }

func (o *OpenAICompatible) client() *http.Client {
	if o.Client != nil {
		return o.Client
	}
	return &http.Client{Timeout: 15 * time.Minute}
}

type oaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type oaRequest struct {
	Model       string      `json:"model"`
	Messages    []oaMessage `json:"messages"`
	MaxTokens   int         `json:"max_tokens,omitempty"`
	Temperature float64     `json:"temperature,omitempty"`
}

type oaResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (o *OpenAICompatible) Chat(ctx context.Context, req Request) (Response, error) {
	start := time.Now()
	msgs := []oaMessage{}
	if req.System != "" {
		msgs = append(msgs, oaMessage{Role: "system", Content: req.System})
	}
	msgs = append(msgs, oaMessage{Role: "user", Content: req.Prompt})
	body, err := json.Marshal(oaRequest{
		Model:       req.Model,
		Messages:    msgs,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	})
	if err != nil {
		return Response{}, err
	}
	url := strings.TrimRight(o.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if o.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+o.APIKey)
	}
	resp, err := o.client().Do(httpReq)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if err != nil {
		return Response{}, err
	}
	if resp.StatusCode >= 400 {
		return Response{}, fmt.Errorf("%s: http %d: %s", o.ProviderName, resp.StatusCode, truncate(string(raw), 500))
	}
	var out oaResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return Response{}, fmt.Errorf("%s: decode response: %w", o.ProviderName, err)
	}
	if out.Error != nil {
		return Response{}, fmt.Errorf("%s: %s", o.ProviderName, out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return Response{}, fmt.Errorf("%s: no choices in response", o.ProviderName)
	}
	text := CleanOutput(out.Choices[0].Message.Content)
	if text == "" {
		return Response{}, ErrEmptyResponse
	}
	return Response{
		Text:         text,
		Provider:     o.ProviderName,
		Model:        req.Model,
		InputTokens:  out.Usage.PromptTokens,
		OutputTokens: out.Usage.CompletionTokens,
		Latency:      time.Since(start),
	}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
