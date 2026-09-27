package aiclient

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

const (
	defaultTimeout   = 2 * time.Minute
	maxResponseBytes = 20 << 20
)

type Client struct {
	baseUrl    string
	httpClient *http.Client
}

type Option func(*Client)

func WithTimeout(timeout time.Duration) Option {
	return func(client *Client) {
		client.httpClient.Timeout = timeout
	}
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(client *Client) {
		client.httpClient = httpClient
	}
}

func New(baseUrl string, options ...Option) *Client {
	client := &Client{
		baseUrl:    strings.TrimRight(baseUrl, "/"),
		httpClient: &http.Client{Timeout: defaultTimeout},
	}

	for _, option := range options {
		option(client)
	}

	return client
}

type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (err *APIError) Error() string {
	return fmt.Sprintf("ai-service responded %d %s: %s", err.StatusCode, err.Code, err.Message)
}

func (client *Client) Analyze(ctx context.Context, request AnalyzeRequest) (AnalyzeResult, error) {
	body, err := json.Marshal(request)

	if err != nil {
		return AnalyzeResult{}, fmt.Errorf("failed to encode analysis request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseUrl+"/analyze", bytes.NewReader(body))

	if err != nil {
		return AnalyzeResult{}, fmt.Errorf("failed to build analysis request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")

	response, err := client.httpClient.Do(httpRequest)

	if err != nil {
		return AnalyzeResult{}, fmt.Errorf("failed to call ai-service: %w", err)
	}
	defer response.Body.Close()

	limited := io.LimitReader(response.Body, maxResponseBytes)

	if response.StatusCode != http.StatusOK {
		return AnalyzeResult{}, decodeAPIError(response.StatusCode, limited)
	}

	var result AnalyzeResult

	if err := json.NewDecoder(limited).Decode(&result); err != nil {
		return AnalyzeResult{}, fmt.Errorf("failed to decode ai-service response: %w", err)
	}

	return result, nil
}

func decodeAPIError(statusCode int, body io.Reader) error {
	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(body).Decode(&payload); err != nil || payload.Error.Code == "" {
		return &APIError{StatusCode: statusCode, Code: "UNEXPECTED_RESPONSE", Message: http.StatusText(statusCode)}
	}

	return &APIError{StatusCode: statusCode, Code: payload.Error.Code, Message: payload.Error.Message}
}
