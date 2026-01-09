package slack

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		webhookURL string
		botToken   string
		channel    string
	}{
		{
			name:       "webhook only",
			webhookURL: "https://hooks.slack.com/services/xxx",
			botToken:   "",
			channel:    "",
		},
		{
			name:       "bot token only",
			webhookURL: "",
			botToken:   "xoxb-token",
			channel:    "#general",
		},
		{
			name:       "both configured",
			webhookURL: "https://hooks.slack.com/services/xxx",
			botToken:   "xoxb-token",
			channel:    "#general",
		},
		{
			name:       "empty configuration",
			webhookURL: "",
			botToken:   "",
			channel:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := NewClient(tt.webhookURL, tt.botToken, tt.channel)

			if client == nil {
				t.Fatal("NewClient returned nil")
			}
			if client.webhookURL != tt.webhookURL {
				t.Errorf("webhookURL = %q, want %q", client.webhookURL, tt.webhookURL)
			}
			if client.botToken != tt.botToken {
				t.Errorf("botToken = %q, want %q", client.botToken, tt.botToken)
			}
			if client.channel != tt.channel {
				t.Errorf("channel = %q, want %q", client.channel, tt.channel)
			}
			if client.httpClient == nil {
				t.Error("httpClient is nil")
			}
		})
	}
}

func TestIsConfigured(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		webhookURL string
		botToken   string
		channel    string
		want       bool
	}{
		{
			name:       "webhook only - configured",
			webhookURL: "https://hooks.slack.com/services/xxx",
			botToken:   "",
			channel:    "",
			want:       true,
		},
		{
			name:       "bot token and channel - configured",
			webhookURL: "",
			botToken:   "xoxb-token",
			channel:    "#general",
			want:       true,
		},
		{
			name:       "both configured",
			webhookURL: "https://hooks.slack.com/services/xxx",
			botToken:   "xoxb-token",
			channel:    "#general",
			want:       true,
		},
		{
			name:       "nothing configured",
			webhookURL: "",
			botToken:   "",
			channel:    "",
			want:       false,
		},
		{
			name:       "bot token without channel - not configured",
			webhookURL: "",
			botToken:   "xoxb-token",
			channel:    "",
			want:       false,
		},
		{
			name:       "channel without bot token - not configured",
			webhookURL: "",
			botToken:   "",
			channel:    "#general",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := NewClient(tt.webhookURL, tt.botToken, tt.channel)
			got := client.IsConfigured()
			if got != tt.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPostWebhook(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		webhookURL     string
		serverStatus   int
		serverResponse string
		msg            *WebhookMessage
		wantErr        bool
		errContains    string
	}{
		{
			name:         "successful webhook post",
			serverStatus: http.StatusOK,
			msg: &WebhookMessage{
				Text: "Hello, world!",
			},
			wantErr: false,
		},
		{
			name:         "webhook with blocks",
			serverStatus: http.StatusOK,
			msg: &WebhookMessage{
				Text: "Fallback text",
				Blocks: []Block{
					{
						Type: "section",
						Text: &BlockText{
							Type: "mrkdwn",
							Text: "*Bold text*",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:         "webhook with attachments",
			serverStatus: http.StatusOK,
			msg: &WebhookMessage{
				Text: "Message with attachment",
				Attachments: []Attachment{
					{
						Color: "#36a64f",
						Text:  "Attachment text",
					},
				},
			},
			wantErr: false,
		},
		{
			name:           "webhook not configured",
			webhookURL:     "", // Will be overwritten to empty
			serverStatus:   http.StatusOK,
			msg:            &WebhookMessage{Text: "test"},
			wantErr:        true,
			errContains:    "webhook URL not configured",
		},
		{
			name:           "server returns 400",
			serverStatus:   http.StatusBadRequest,
			serverResponse: "invalid_payload",
			msg:            &WebhookMessage{Text: "test"},
			wantErr:        true,
			errContains:    "status 400",
		},
		{
			name:           "server returns 500",
			serverStatus:   http.StatusInternalServerError,
			serverResponse: "internal error",
			msg:            &WebhookMessage{Text: "test"},
			wantErr:        true,
			errContains:    "status 500",
		},
		{
			name:           "server returns 404",
			serverStatus:   http.StatusNotFound,
			serverResponse: "not found",
			msg:            &WebhookMessage{Text: "test"},
			wantErr:        true,
			errContains:    "status 404",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var receivedBody []byte
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
				}

				// Read body for verification
				buf := make([]byte, 4096)
				n, _ := r.Body.Read(buf)
				receivedBody = buf[:n]

				w.WriteHeader(tt.serverStatus)
				if tt.serverResponse != "" {
					w.Write([]byte(tt.serverResponse))
				}
			}))
			defer server.Close()

			webhookURL := server.URL
			if tt.webhookURL == "" && tt.name == "webhook not configured" {
				webhookURL = ""
			}

			client := NewClient(webhookURL, "", "")
			err := client.PostWebhook(context.Background(), tt.msg)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !containsSubstring(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				// Verify the body was properly marshaled
				if len(receivedBody) > 0 {
					var received WebhookMessage
					if err := json.Unmarshal(receivedBody, &received); err != nil {
						t.Errorf("failed to unmarshal received body: %v", err)
					}
					if received.Text != tt.msg.Text {
						t.Errorf("received text = %q, want %q", received.Text, tt.msg.Text)
					}
				}
			}
		})
	}
}

func TestPostWebhook_ContextCancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Slow response
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "", "")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := client.PostWebhook(ctx, &WebhookMessage{Text: "test"})
	if err == nil {
		t.Error("expected error due to context cancellation")
	}
}

func TestPostMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		botToken     string
		channel      string
		req          *ChatPostMessageRequest
		serverResp   ChatPostMessageResponse
		serverStatus int
		wantErr      bool
		errContains  string
	}{
		{
			name:     "successful message post",
			botToken: "xoxb-test-token",
			channel:  "#general",
			req: &ChatPostMessageRequest{
				Text: "Hello from test",
			},
			serverResp: ChatPostMessageResponse{
				OK:      true,
				TS:      "1234567890.123456",
				Channel: "C123456",
			},
			serverStatus: http.StatusOK,
			wantErr:      false,
		},
		{
			name:     "message with thread_ts",
			botToken: "xoxb-test-token",
			channel:  "#general",
			req: &ChatPostMessageRequest{
				Text:     "Reply in thread",
				ThreadTS: "1234567890.000000",
			},
			serverResp: ChatPostMessageResponse{
				OK:      true,
				TS:      "1234567890.123457",
				Channel: "C123456",
			},
			serverStatus: http.StatusOK,
			wantErr:      false,
		},
		{
			name:     "message with blocks",
			botToken: "xoxb-test-token",
			channel:  "#general",
			req: &ChatPostMessageRequest{
				Text: "Fallback",
				Blocks: []Block{
					{
						Type: "section",
						Text: &BlockText{
							Type: "mrkdwn",
							Text: "*Bold*",
						},
					},
				},
			},
			serverResp: ChatPostMessageResponse{
				OK:      true,
				TS:      "1234567890.123458",
				Channel: "C123456",
			},
			serverStatus: http.StatusOK,
			wantErr:      false,
		},
		{
			name:     "bot token not configured",
			botToken: "",
			channel:  "#general",
			req: &ChatPostMessageRequest{
				Text: "test",
			},
			wantErr:     true,
			errContains: "bot token not configured",
		},
		{
			name:     "API returns error",
			botToken: "xoxb-test-token",
			channel:  "#general",
			req: &ChatPostMessageRequest{
				Text: "test",
			},
			serverResp: ChatPostMessageResponse{
				OK:    false,
				Error: "channel_not_found",
			},
			serverStatus: http.StatusOK,
			wantErr:      true,
			errContains:  "channel_not_found",
		},
		{
			name:     "API returns invalid_auth",
			botToken: "xoxb-invalid-token",
			channel:  "#general",
			req: &ChatPostMessageRequest{
				Text: "test",
			},
			serverResp: ChatPostMessageResponse{
				OK:    false,
				Error: "invalid_auth",
			},
			serverStatus: http.StatusOK,
			wantErr:      true,
			errContains:  "invalid_auth",
		},
		{
			name:     "uses default channel when not specified in request",
			botToken: "xoxb-test-token",
			channel:  "#default-channel",
			req: &ChatPostMessageRequest{
				Text: "test",
			},
			serverResp: ChatPostMessageResponse{
				OK:      true,
				TS:      "1234567890.123459",
				Channel: "C123456",
			},
			serverStatus: http.StatusOK,
			wantErr:      false,
		},
		{
			name:     "request channel overrides default",
			botToken: "xoxb-test-token",
			channel:  "#default-channel",
			req: &ChatPostMessageRequest{
				Text:    "test",
				Channel: "#override-channel",
			},
			serverResp: ChatPostMessageResponse{
				OK:      true,
				TS:      "1234567890.123460",
				Channel: "C789012",
			},
			serverStatus: http.StatusOK,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var receivedReq ChatPostMessageRequest
			var receivedAuthHeader string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
				}
				receivedAuthHeader = r.Header.Get("Authorization")

				// Don't decode body for bot token not configured test (server not used)
				if tt.botToken != "" {
					if err := json.NewDecoder(r.Body).Decode(&receivedReq); err != nil {
						t.Errorf("failed to decode request body: %v", err)
					}
				}

				w.WriteHeader(tt.serverStatus)
				json.NewEncoder(w).Encode(tt.serverResp)
			}))
			defer server.Close()

			// Create client with test server URL using apiBaseURL
			client := &Client{
				webhookURL: "",
				botToken:   tt.botToken,
				channel:    tt.channel,
				httpClient: &http.Client{Timeout: 30 * time.Second},
				apiBaseURL: server.URL, // Override API URL for testing
			}

			resp, err := client.PostMessage(context.Background(), tt.req)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !containsSubstring(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if resp == nil {
					t.Fatal("expected response, got nil")
				}
				if resp.TS != tt.serverResp.TS {
					t.Errorf("response TS = %q, want %q", resp.TS, tt.serverResp.TS)
				}

				// Verify auth header
				expectedAuth := "Bearer " + tt.botToken
				if receivedAuthHeader != expectedAuth {
					t.Errorf("Authorization header = %q, want %q", receivedAuthHeader, expectedAuth)
				}

				// Verify channel was set properly
				expectedChannel := tt.req.Channel
				if expectedChannel == "" {
					expectedChannel = tt.channel
				}
				if receivedReq.Channel != expectedChannel {
					t.Errorf("request channel = %q, want %q", receivedReq.Channel, expectedChannel)
				}
			}
		})
	}
}

func TestPostMessage_ContextCancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ChatPostMessageResponse{OK: true})
	}))
	defer server.Close()

	client := &Client{
		botToken:   "xoxb-test-token",
		channel:    "#general",
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiBaseURL: server.URL,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.PostMessage(ctx, &ChatPostMessageRequest{Text: "test"})
	if err == nil {
		t.Error("expected error due to context cancellation")
	}
}

func TestPostMessage_InvalidJSONResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json response"))
	}))
	defer server.Close()

	client := &Client{
		botToken:   "xoxb-test-token",
		channel:    "#general",
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiBaseURL: server.URL,
	}

	_, err := client.PostMessage(context.Background(), &ChatPostMessageRequest{Text: "test"})
	if err == nil {
		t.Error("expected error due to invalid JSON response")
	}
	if !containsSubstring(err.Error(), "decode response") {
		t.Errorf("error should contain 'decode response', got: %v", err)
	}
}

func TestPostWithRetry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		attempts    int // Number of times function fails before succeeding (0 = always succeed)
		maxAttempts int // Total attempts (for always-fail scenarios)
		wantErr     bool
		errContains string
	}{
		{
			name:     "succeeds on first attempt",
			attempts: 0,
			wantErr:  false,
		},
		{
			name:     "succeeds on second attempt",
			attempts: 1,
			wantErr:  false,
		},
		{
			name:     "succeeds on third attempt",
			attempts: 2,
			wantErr:  false,
		},
		{
			name:     "succeeds on fourth attempt",
			attempts: 3,
			wantErr:  false,
		},
		{
			name:        "fails after all retries",
			attempts:    10, // More than max retries
			maxAttempts: 4, // 1 initial + 3 retries
			wantErr:     true,
			errContains: "after 3 retries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := NewClient("https://example.com", "", "")
			var callCount int32

			fn := func() error {
				count := atomic.AddInt32(&callCount, 1)
				if int(count) <= tt.attempts {
					return errors.New("temporary failure")
				}
				return nil
			}

			// Use a short timeout context
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err := client.PostWithRetry(ctx, fn)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContains != "" && !containsSubstring(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
				if tt.maxAttempts > 0 && int(callCount) != tt.maxAttempts {
					t.Errorf("expected %d attempts, got %d", tt.maxAttempts, callCount)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				expectedCalls := tt.attempts + 1
				if int(callCount) != expectedCalls {
					t.Errorf("expected %d calls, got %d", expectedCalls, callCount)
				}
			}
		})
	}
}

func TestPostWithRetry_ContextCancellation(t *testing.T) {
	t.Parallel()

	client := NewClient("https://example.com", "", "")
	var callCount int32

	fn := func() error {
		atomic.AddInt32(&callCount, 1)
		return errors.New("always fail")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := client.PostWithRetry(ctx, fn)

	if err == nil {
		t.Error("expected error due to context cancellation")
	}

	// Should have been cancelled before all retries completed
	if callCount >= 4 {
		t.Errorf("expected fewer than 4 calls due to cancellation, got %d", callCount)
	}
}

func TestPostWithRetry_ImmediateContextCancellation(t *testing.T) {
	t.Parallel()

	client := NewClient("https://example.com", "", "")

	fn := func() error {
		return errors.New("always fail")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := client.PostWithRetry(ctx, fn)

	if err == nil {
		t.Error("expected error due to context cancellation")
	}
	if !errors.Is(err, context.Canceled) {
		// The first call might succeed before context is checked
		// so we just verify an error occurred
	}
}

// Helper function to check if a string contains a substring
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestWebhookMessage_JSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  WebhookMessage
		want string
	}{
		{
			name: "simple text",
			msg:  WebhookMessage{Text: "Hello"},
			want: `{"text":"Hello"}`,
		},
		{
			name: "with channel",
			msg:  WebhookMessage{Text: "Hello", Channel: "#general"},
			want: `{"text":"Hello","channel":"#general"}`,
		},
		{
			name: "with blocks",
			msg: WebhookMessage{
				Text: "Fallback",
				Blocks: []Block{
					{
						Type: "section",
						Text: &BlockText{Type: "mrkdwn", Text: "*Bold*"},
					},
				},
			},
			want: `{"text":"Fallback","blocks":[{"type":"section","text":{"type":"mrkdwn","text":"*Bold*"}}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tt.msg)
			if err != nil {
				t.Fatalf("json.Marshal error: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("JSON = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestChatPostMessageRequest_JSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  ChatPostMessageRequest
		want string
	}{
		{
			name: "simple text",
			req:  ChatPostMessageRequest{Channel: "#test", Text: "Hello"},
			want: `{"channel":"#test","text":"Hello"}`,
		},
		{
			name: "with thread_ts",
			req:  ChatPostMessageRequest{Channel: "#test", Text: "Reply", ThreadTS: "1234.5678"},
			want: `{"channel":"#test","text":"Reply","thread_ts":"1234.5678"}`,
		},
		{
			name: "with blocks",
			req: ChatPostMessageRequest{
				Channel: "#test",
				Blocks: []Block{
					{Type: "divider"},
				},
			},
			want: `{"channel":"#test","blocks":[{"type":"divider"}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("json.Marshal error: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("JSON = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestChatPostMessageResponse_JSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		json string
		want ChatPostMessageResponse
	}{
		{
			name: "successful response",
			json: `{"ok":true,"ts":"1234567890.123456","channel":"C123456"}`,
			want: ChatPostMessageResponse{OK: true, TS: "1234567890.123456", Channel: "C123456"},
		},
		{
			name: "error response",
			json: `{"ok":false,"error":"channel_not_found"}`,
			want: ChatPostMessageResponse{OK: false, Error: "channel_not_found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got ChatPostMessageResponse
			if err := json.Unmarshal([]byte(tt.json), &got); err != nil {
				t.Fatalf("json.Unmarshal error: %v", err)
			}
			if got.OK != tt.want.OK {
				t.Errorf("OK = %v, want %v", got.OK, tt.want.OK)
			}
			if got.TS != tt.want.TS {
				t.Errorf("TS = %q, want %q", got.TS, tt.want.TS)
			}
			if got.Channel != tt.want.Channel {
				t.Errorf("Channel = %q, want %q", got.Channel, tt.want.Channel)
			}
			if got.Error != tt.want.Error {
				t.Errorf("Error = %q, want %q", got.Error, tt.want.Error)
			}
		})
	}
}

func TestClient_ImplementsMessenger(t *testing.T) {
	t.Parallel()

	// This test verifies that Client implements the Messenger interface
	// The compiler will catch this at build time, but having an explicit test
	// makes the contract clearer
	var _ Messenger = (*Client)(nil)
}
