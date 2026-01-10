package metrics

import (
	"context"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Collector manages OpenTelemetry metrics
type Collector struct {
	enabled       bool
	meterProvider *sdkmetric.MeterProvider
	meter         metric.Meter

	// Counters
	iterationsTotal   metric.Int64Counter
	commitsTotal      metric.Int64Counter
	claudeErrorsTotal metric.Int64Counter

	// Gauges (using UpDownCounters for simplicity)
	sessionDuration metric.Float64ObservableGauge
	todosPending    metric.Int64ObservableGauge
	todosCompleted  metric.Int64ObservableGauge
	activeSessions  metric.Int64UpDownCounter

	// Histograms
	iterationDuration metric.Float64Histogram

	// Labels
	projectAttr   attribute.KeyValue
	sessionIDAttr attribute.KeyValue

	// State for gauges
	startTime       time.Time
	currentPending  atomic.Int64
	currentComplete atomic.Int64
}

// CollectorConfig holds configuration for the metrics collector
type CollectorConfig struct {
	Enabled       bool
	Endpoint      string
	MetricsPrefix string
	ProjectName   string
	SessionID     string
}

// NewCollector creates a new metrics collector
func NewCollector(cfg CollectorConfig) (*Collector, error) {
	c := &Collector{
		enabled:   cfg.Enabled,
		startTime: time.Now(),
	}

	if !cfg.Enabled {
		return c, nil
	}

	ctx := context.Background()

	// Create OTLP exporter
	conn, err := grpc.NewClient(cfg.Endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	exporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, err
	}

	// Create resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("ralph"),
			semconv.ServiceVersion("0.1.0"),
		),
	)
	if err != nil {
		return nil, err
	}

	// Create meter provider
	c.meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(10*time.Second))),
	)
	otel.SetMeterProvider(c.meterProvider)

	// Create meter
	c.meter = c.meterProvider.Meter(cfg.MetricsPrefix)

	// Set labels
	c.projectAttr = attribute.String("project", cfg.ProjectName)
	c.sessionIDAttr = attribute.String("session_id", cfg.SessionID)

	// Initialize metrics
	if err := c.initMetrics(cfg.MetricsPrefix); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Collector) initMetrics(prefix string) error {
	var err error

	// Counters
	c.iterationsTotal, err = c.meter.Int64Counter(
		prefix+"_iterations_total",
		metric.WithDescription("Total iterations completed"),
	)
	if err != nil {
		return err
	}

	c.commitsTotal, err = c.meter.Int64Counter(
		prefix+"_commits_total",
		metric.WithDescription("Git commits made during session"),
	)
	if err != nil {
		return err
	}

	c.claudeErrorsTotal, err = c.meter.Int64Counter(
		prefix+"_claude_errors_total",
		metric.WithDescription("Claude execution errors"),
	)
	if err != nil {
		return err
	}

	// Active sessions gauge (using UpDownCounter)
	c.activeSessions, err = c.meter.Int64UpDownCounter(
		prefix+"_active_sessions",
		metric.WithDescription("Currently running ralph instances"),
	)
	if err != nil {
		return err
	}

	// Histograms
	c.iterationDuration, err = c.meter.Float64Histogram(
		prefix+"_iteration_duration_seconds",
		metric.WithDescription("Time per iteration"),
		metric.WithExplicitBucketBoundaries(1, 5, 10, 30, 60, 120, 300, 600),
	)
	if err != nil {
		return err
	}

	// Observable gauges
	c.sessionDuration, err = c.meter.Float64ObservableGauge(
		prefix+"_session_duration_seconds",
		metric.WithDescription("Current session runtime"),
		metric.WithFloat64Callback(func(_ context.Context, o metric.Float64Observer) error {
			o.Observe(time.Since(c.startTime).Seconds(), metric.WithAttributes(c.projectAttr, c.sessionIDAttr))
			return nil
		}),
	)
	if err != nil {
		return err
	}

	c.todosPending, err = c.meter.Int64ObservableGauge(
		prefix+"_todos_pending",
		metric.WithDescription("Current pending todo items"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			o.Observe(c.currentPending.Load(), metric.WithAttributes(c.projectAttr, c.sessionIDAttr))
			return nil
		}),
	)
	if err != nil {
		return err
	}

	c.todosCompleted, err = c.meter.Int64ObservableGauge(
		prefix+"_todos_completed",
		metric.WithDescription("Current completed todo items"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			o.Observe(c.currentComplete.Load(), metric.WithAttributes(c.projectAttr, c.sessionIDAttr))
			return nil
		}),
	)
	if err != nil {
		return err
	}

	return nil
}

// RecordIterationComplete records a completed iteration
func (c *Collector) RecordIterationComplete(ctx context.Context, duration time.Duration, exitReason string) {
	if !c.enabled {
		return
	}

	c.iterationsTotal.Add(ctx, 1,
		metric.WithAttributes(c.projectAttr, c.sessionIDAttr, attribute.String("exit_reason", exitReason)))
	c.iterationDuration.Record(ctx, duration.Seconds(),
		metric.WithAttributes(c.projectAttr, c.sessionIDAttr))
}

// RecordCommits records new git commits
func (c *Collector) RecordCommits(ctx context.Context, count int) {
	if !c.enabled || count <= 0 {
		return
	}

	c.commitsTotal.Add(ctx, int64(count), metric.WithAttributes(c.projectAttr, c.sessionIDAttr))
}

// RecordError records a Claude error
func (c *Collector) RecordError(ctx context.Context, errorType string) {
	if !c.enabled {
		return
	}

	c.claudeErrorsTotal.Add(ctx, 1,
		metric.WithAttributes(c.projectAttr, c.sessionIDAttr, attribute.String("error_type", errorType)))
}

// UpdateTodoCounts updates the todo gauge values
func (c *Collector) UpdateTodoCounts(pending, completed int) {
	c.currentPending.Store(int64(pending))
	c.currentComplete.Store(int64(completed))
}

// SessionStart marks the session as active
func (c *Collector) SessionStart(ctx context.Context) {
	if !c.enabled {
		return
	}

	c.activeSessions.Add(ctx, 1, metric.WithAttributes(c.projectAttr))
}

// SessionEnd marks the session as inactive
func (c *Collector) SessionEnd(ctx context.Context) {
	if !c.enabled {
		return
	}

	c.activeSessions.Add(ctx, -1, metric.WithAttributes(c.projectAttr))
}

// Shutdown gracefully shuts down the metrics provider
func (c *Collector) Shutdown(ctx context.Context) error {
	if !c.enabled || c.meterProvider == nil {
		return nil
	}

	// Force flush to ensure all pending metrics are exported
	if err := c.meterProvider.ForceFlush(ctx); err != nil {
		// Log but continue with shutdown
		_ = err
	}

	return c.meterProvider.Shutdown(ctx)
}

// IsEnabled returns whether metrics collection is enabled
func (c *Collector) IsEnabled() bool {
	return c.enabled
}
