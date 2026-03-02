package deepseek

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ModelResponse represents the response from the models API
type ModelResponse struct {
	Data   []Model `json:"data"`
	Object string  `json:"object"`
}

// Model represents a model in the response
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ChatRequest represents the request body for the chat API
type ChatRequest struct {
	Model            string         `json:"model"`
	Messages         []Message      `json:"messages"`
	FrequencyPenalty int            `json:"frequency_penalty"`
	MaxTokens        int            `json:"max_tokens"`
	PresencePenalty  int            `json:"presence_penalty"`
	ResponseFormat   ResponseFormat `json:"response_format"`
	Stop             interface{}    `json:"stop"`
	Stream           bool           `json:"stream"`
	Temperature      float64        `json:"temperature"`
	TopP             float64        `json:"top_p"`
	Tools            interface{}    `json:"tools"`
	ToolChoice       string         `json:"tool_choice"`
	TopLogprobs      interface{}    `json:"top_logprobs"`
}

// Message represents a message in the chat request
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ResponseFormat represents the response format in the chat request
type ResponseFormat struct {
	Type string `json:"type"`
}

// ChatResponse represents the response from the chat API
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a choice in the chat response
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents usage information in the chat response
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// getAPIKey retrieves the API key from environment variable
func getAPIKey() (string, error) {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("DEEPSEEK_API_KEY environment variable not set")
	}
	return apiKey, nil
}

// DSList returns the list of available models
func DSList() (*ModelResponse, error) {
	apiKey, err := getAPIKey()
	if err != nil {
		return nil, err
	}

	url := "https://api.deepseek.com/models"
	method := "GET"
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	rq, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	rq.Header.Add("Accept", "application/json")
	rq.Header.Add("Authorization", "Bearer "+apiKey)

	rs, err := client.Do(rq)
	if err != nil {
		return nil, err
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		return nil, err
	}

	if rs.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", rs.StatusCode, string(body))
	}

	var response ModelResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// DSChat sends a chat request and returns the response
func DSChat(question string) (*ChatResponse, error) {
	if question == "" {
		return nil, fmt.Errorf("question cannot be empty")
	}

	apiKey, err := getAPIKey()
	if err != nil {
		return nil, err
	}

	url := "https://api.deepseek.com/chat/completions"
	method := "POST"

	request := ChatRequest{
		Model: "deepseek-reasoner",
		Messages: []Message{
			{Role: "user", Content: question},
		},
		FrequencyPenalty: 0,
		MaxTokens:        4098,
		PresencePenalty:  0,
		ResponseFormat:   ResponseFormat{Type: "text"},
		Stop:             nil,
		Stream:           false,
		Temperature:      1,
		TopP:             1,
		Tools:            nil,
		ToolChoice:       "none",
		TopLogprobs:      nil,
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	rq, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	rq.Body = io.NopCloser(strings.NewReader(string(payload)))
	rq.Header.Add("Accept", "application/json")
	rq.Header.Add("Content-Type", "application/json")
	rq.Header.Add("Authorization", "Bearer "+apiKey)

	rs, err := client.Do(rq)
	if err != nil {
		return nil, err
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		return nil, err
	}

	if rs.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", rs.StatusCode, string(body))
	}

	var response ChatResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return &response, nil
}
