package metrics

import (
	"context"
	"testing"
	"time"
)

func TestNewCollector_Disabled(t *testing.T) {
	t.Parallel()

	cfg := CollectorConfig{
		Enabled:       false,
		Endpoint:      "",
		MetricsPrefix: "test",
		ProjectName:   "testproject",
		SessionID:     "test-session-123",
	}

	collector, err := NewCollector(cfg)
	if err != nil {
		t.Fatalf("NewCollector() error = %v", err)
	}

	if collector.enabled {
		t.Error("Expected collector to be disabled")
	}

	if collector.meterProvider != nil {
		t.Error("Expected meterProvider to be nil when disabled")
	}
}

func TestCollector_IsEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{
			name:     "enabled",
			enabled:  true,
			expected: true,
		},
		{
			name:     "disabled",
			enabled:  false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			collector := &Collector{enabled: tt.enabled}
			if got := collector.IsEnabled(); got != tt.expected {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCollector_UpdateTodoCounts(t *testing.T) {
	t.Parallel()

	collector := &Collector{enabled: false}

	// Initial values should be 0
	if collector.currentPending.Load() != 0 {
		t.Errorf("Initial currentPending = %d, want 0", collector.currentPending.Load())
	}
	if collector.currentComplete.Load() != 0 {
		t.Errorf("Initial currentComplete = %d, want 0", collector.currentComplete.Load())
	}

	// Update counts
	collector.UpdateTodoCounts(5, 3)

	if collector.currentPending.Load() != 5 {
		t.Errorf("After update currentPending = %d, want 5", collector.currentPending.Load())
	}
	if collector.currentComplete.Load() != 3 {
		t.Errorf("After update currentComplete = %d, want 3", collector.currentComplete.Load())
	}

	// Update again
	collector.UpdateTodoCounts(10, 15)

	if collector.currentPending.Load() != 10 {
		t.Errorf("After second update currentPending = %d, want 10", collector.currentPending.Load())
	}
	if collector.currentComplete.Load() != 15 {
		t.Errorf("After second update currentComplete = %d, want 15", collector.currentComplete.Load())
	}
}

func TestCollector_RecordIterationComplete_Disabled(t *testing.T) {
	t.Parallel()

	collector := &Collector{enabled: false}
	ctx := context.Background()

	// Should not panic when disabled
	collector.RecordIterationComplete(ctx, 5*time.Second, "completed")
	collector.RecordIterationComplete(ctx, 10*time.Second, "error")
	collector.RecordIterationComplete(ctx, 30*time.Second, "timeout")
}

func TestCollector_RecordCommits_Disabled(t *testing.T) {
	t.Parallel()

	collector := &Collector{enabled: false}
	ctx := context.Background()

	// Should not panic when disabled
	collector.RecordCommits(ctx, 5)
	collector.RecordCommits(ctx, 0)
	collector.RecordCommits(ctx, -1) // Edge case
}

func TestCollector_RecordCommits_ZeroOrNegative(t *testing.T) {
	t.Parallel()

	// Even with enabled collector, zero or negative counts should be no-ops
	// We can't fully test this without OTEL setup, but we verify the guard condition
	collector := &Collector{enabled: true}
	ctx := context.Background()

	// Should not panic (the enabled check + count <= 0 check should exit early)
	collector.RecordCommits(ctx, 0)
	collector.RecordCommits(ctx, -1)
}

func TestCollector_RecordError_Disabled(t *testing.T) {
	t.Parallel()

	collector := &Collector{enabled: false}
	ctx := context.Background()

	// Should not panic when disabled
	collector.RecordError(ctx, "execution_error")
	collector.RecordError(ctx, "timeout_error")
}

func TestCollector_SessionStartEnd_Disabled(t *testing.T) {
	t.Parallel()

	collector := &Collector{enabled: false}
	ctx := context.Background()

	// Should not panic when disabled
	collector.SessionStart(ctx)
	collector.SessionEnd(ctx)
}

func TestCollector_Shutdown_Disabled(t *testing.T) {
	t.Parallel()

	collector := &Collector{enabled: false}
	ctx := context.Background()

	err := collector.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown() error = %v, want nil", err)
	}
}

func TestCollector_Shutdown_NilProvider(t *testing.T) {
	t.Parallel()

	collector := &Collector{
		enabled:       true,
		meterProvider: nil,
	}
	ctx := context.Background()

	// Should handle nil provider gracefully
	err := collector.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown() with nil provider error = %v, want nil", err)
	}
}

func TestCollectorConfig(t *testing.T) {
	t.Parallel()

	cfg := CollectorConfig{
		Enabled:       true,
		Endpoint:      "localhost:4317",
		MetricsPrefix: "ralph",
		ProjectName:   "myproject",
		SessionID:     "abc-123",
	}

	if !cfg.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if cfg.Endpoint != "localhost:4317" {
		t.Errorf("Endpoint = %s, want localhost:4317", cfg.Endpoint)
	}
	if cfg.MetricsPrefix != "ralph" {
		t.Errorf("MetricsPrefix = %s, want ralph", cfg.MetricsPrefix)
	}
	if cfg.ProjectName != "myproject" {
		t.Errorf("ProjectName = %s, want myproject", cfg.ProjectName)
	}
	if cfg.SessionID != "abc-123" {
		t.Errorf("SessionID = %s, want abc-123", cfg.SessionID)
	}
}

func TestCollector_StartTime(t *testing.T) {
	t.Parallel()

	before := time.Now()
	collector := &Collector{
		enabled:   false,
		startTime: time.Now(),
	}
	after := time.Now()

	if collector.startTime.Before(before) {
		t.Error("startTime should not be before test start")
	}
	if collector.startTime.After(after) {
		t.Error("startTime should not be after test end")
	}
}

func TestCollector_ImplementsMetricsCollector(t *testing.T) {
	t.Parallel()

	// Verify that Collector implements the MetricsCollector interface
	var _ MetricsCollector = (*Collector)(nil)
}

func TestCollectorConfig_AllFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config CollectorConfig
	}{
		{
			name: "minimal config",
			config: CollectorConfig{
				Enabled: false,
			},
		},
		{
			name: "full config",
			config: CollectorConfig{
				Enabled:       true,
				Endpoint:      "localhost:4317",
				MetricsPrefix: "ralph",
				ProjectName:   "myproject",
				SessionID:     "session-abc-123",
			},
		},
		{
			name: "custom endpoint",
			config: CollectorConfig{
				Enabled:  true,
				Endpoint: "otel-collector.example.com:4317",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Just verify the struct can be created with various configurations
			_ = tt.config
		})
	}
}

func TestCollector_Attributes(t *testing.T) {
	t.Parallel()

	// Test that collectors can store attributes
	collector := &Collector{
		enabled: false,
	}

	// Access currentPending and currentComplete (these are public through UpdateTodoCounts)
	collector.UpdateTodoCounts(10, 5)

	if collector.currentPending.Load() != 10 {
		t.Errorf("currentPending = %d, want 10", collector.currentPending.Load())
	}
	if collector.currentComplete.Load() != 5 {
		t.Errorf("currentComplete = %d, want 5", collector.currentComplete.Load())
	}
}

func TestCollector_MultipleUpdates(t *testing.T) {
	t.Parallel()

	collector := &Collector{enabled: false}
	ctx := context.Background()

	// Simulate multiple recording calls
	for i := 0; i < 10; i++ {
		collector.RecordIterationComplete(ctx, time.Duration(i)*time.Second, "completed")
		collector.RecordCommits(ctx, i)
		collector.RecordError(ctx, "test_error")
		collector.UpdateTodoCounts(i, i*2)
	}

	// Final state should reflect last update
	if collector.currentPending.Load() != 9 {
		t.Errorf("After 10 updates, currentPending = %d, want 9", collector.currentPending.Load())
	}
	if collector.currentComplete.Load() != 18 {
		t.Errorf("After 10 updates, currentComplete = %d, want 18", collector.currentComplete.Load())
	}
}

func TestCollector_SessionLifecycle(t *testing.T) {
	t.Parallel()

	collector := &Collector{enabled: false}
	ctx := context.Background()

	// Full lifecycle should not panic
	collector.SessionStart(ctx)
	collector.RecordIterationComplete(ctx, 5*time.Second, "success")
	collector.RecordCommits(ctx, 3)
	collector.UpdateTodoCounts(5, 2)
	collector.SessionEnd(ctx)

	err := collector.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}

func TestCollector_ConcurrentUpdates(t *testing.T) {
	t.Parallel()

	collector := &Collector{enabled: false}
	ctx := context.Background()

	// Run concurrent updates (should not panic)
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				collector.UpdateTodoCounts(n, j)
				collector.RecordCommits(ctx, 1)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}
}

func TestNewCollector_DisabledNoError(t *testing.T) {
	t.Parallel()

	cfg := CollectorConfig{
		Enabled:       false,
		Endpoint:      "", // Empty endpoint should be fine when disabled
		MetricsPrefix: "",
		ProjectName:   "",
		SessionID:     "",
	}

	collector, err := NewCollector(cfg)
	if err != nil {
		t.Fatalf("NewCollector() with disabled config should not error: %v", err)
	}

	if collector.enabled {
		t.Error("Collector should be disabled")
	}
}
