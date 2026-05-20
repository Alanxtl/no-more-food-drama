package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

type EnhancementResult struct {
	Restaurants []RestaurantEnhancement `json:"restaurants"`
}

type RestaurantEnhancement struct {
	ID      string   `json:"id"`
	TypeIDs []string `json:"typeIds"`
	Tags    []string `json:"tags"`
}

func NewClient(baseURL, apiKey, model string, client *http.Client) *Client {
	if client == nil {
		client = http.DefaultClient
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		client:  client,
	}
}

func (c *Client) EnhanceTags(ctx context.Context, compactRestaurantJSON string) (EnhancementResult, error) {
	payload := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{
				Role:    "system",
				Content: `You classify restaurants for a food decision app. Return JSON only, with no Markdown or explanation. Schema: {"restaurants":[{"id":"string","typeIds":["string"],"tags":["string"]}]}`,
			},
			{Role: "user", Content: compactRestaurantJSON},
		},
		Temperature: 0.2,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return EnhancementResult{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return EnhancementResult{}, err
	}
	request.Header.Set("Authorization", "Bearer "+c.apiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := c.client.Do(request)
	if err != nil {
		return EnhancementResult{}, err
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return EnhancementResult{}, fmt.Errorf("llm request failed with status %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var decoded chatResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return EnhancementResult{}, err
	}
	if len(decoded.Choices) == 0 {
		return EnhancementResult{}, fmt.Errorf("llm returned no choices")
	}

	var result EnhancementResult
	if err := json.Unmarshal([]byte(decoded.Choices[0].Message.Content), &result); err != nil {
		return EnhancementResult{}, err
	}
	return result, nil
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}
