package slack

import "context"

// Messenger defines the interface for sending Slack messages.
// This interface enables mocking the Slack client for testing.
type Messenger interface {
	// PostMessage sends a message via the chat.postMessage API (for thread replies)
	PostMessage(ctx context.Context, req *ChatPostMessageRequest) (*ChatPostMessageResponse, error)

	// PostWebhook sends a message via webhook (for initial thread creation)
	PostWebhook(ctx context.Context, msg *WebhookMessage) error

	// IsConfigured returns true if the client has the minimum configuration to send messages
	IsConfigured() bool

	// PostWithRetry sends a message with exponential backoff retry
	PostWithRetry(ctx context.Context, fn func() error) error
}

// Ensure Client implements Messenger
var _ Messenger = (*Client)(nil)
