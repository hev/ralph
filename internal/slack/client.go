package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles Slack API interactions
type Client struct {
	webhookURL string
	botToken   string
	channel    string
	httpClient *http.Client
	// apiBaseURL allows overriding the Slack API URL for testing
	apiBaseURL string
}

// NewClient creates a new Slack client
func NewClient(webhookURL, botToken, channel string) *Client {
	return &Client{
		webhookURL: webhookURL,
		botToken:   botToken,
		channel:    channel,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// WebhookMessage represents a Slack webhook message
type WebhookMessage struct {
	Text        string       `json:"text,omitempty"`
	Channel     string       `json:"channel,omitempty"`
	Blocks      []Block      `json:"blocks,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Block represents a Slack block
type Block struct {
	Type string      `json:"type"`
	Text *BlockText  `json:"text,omitempty"`
	Fields []BlockText `json:"fields,omitempty"`
}

// BlockText represents text within a block
type BlockText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Attachment represents a Slack attachment
type Attachment struct {
	Color string `json:"color,omitempty"`
	Text  string `json:"text,omitempty"`
}

// ChatPostMessageRequest represents a chat.postMessage API request
type ChatPostMessageRequest struct {
	Channel  string  `json:"channel"`
	Text     string  `json:"text,omitempty"`
	Blocks   []Block `json:"blocks,omitempty"`
	ThreadTS string  `json:"thread_ts,omitempty"`
}

// ChatPostMessageResponse represents a chat.postMessage API response
type ChatPostMessageResponse struct {
	OK        bool   `json:"ok"`
	Error     string `json:"error,omitempty"`
	TS        string `json:"ts,omitempty"`
	Channel   string `json:"channel,omitempty"`
}

// PostWebhook sends a message via webhook (for initial thread creation)
func (c *Client) PostWebhook(ctx context.Context, msg *WebhookMessage) error {
	if c.webhookURL == "" {
		return fmt.Errorf("webhook URL not configured")
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// PostMessage sends a message via chat.postMessage API (for thread replies)
func (c *Client) PostMessage(ctx context.Context, req *ChatPostMessageRequest) (*ChatPostMessageResponse, error) {
	if c.botToken == "" {
		return nil, fmt.Errorf("bot token not configured")
	}

	if req.Channel == "" {
		req.Channel = c.channel
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	apiURL := "https://slack.com/api/chat.postMessage"
	if c.apiBaseURL != "" {
		apiURL = c.apiBaseURL + "/chat.postMessage"
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.botToken)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	var result ChatPostMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("slack api error: %s", result.Error)
	}

	return &result, nil
}

// PostWithRetry sends a message with exponential backoff retry
func (c *Client) PostWithRetry(ctx context.Context, fn func() error) error {
	backoff := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}

	var lastErr error
	for i := 0; i <= len(backoff); i++ {
		if err := fn(); err != nil {
			lastErr = err
			if i < len(backoff) {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(backoff[i]):
					continue
				}
			}
		} else {
			return nil
		}
	}

	return fmt.Errorf("after %d retries: %w", len(backoff), lastErr)
}

// IsConfigured returns true if the client has the minimum configuration to send messages
func (c *Client) IsConfigured() bool {
	return c.webhookURL != "" || (c.botToken != "" && c.channel != "")
}
