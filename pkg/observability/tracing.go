// Package observability provides structured logging and distributed tracing utilities.
package observability

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/lambdacontext"
	"github.com/aws/aws-xray-sdk-go/xray"
)

// TracingConfig holds configuration for tracing.
type TracingConfig struct {
	Enabled     bool
	ServiceName string
	Version     string
}

// Tracer wraps X-Ray tracing functionality.
type Tracer struct {
	config TracingConfig
}

// NewTracer creates a new tracer instance.
func NewTracer(config TracingConfig) *Tracer {
	return &Tracer{
		config: config,
	}
}

// StartSegment starts a new X-Ray segment if tracing is enabled.
func (t *Tracer) StartSegment(ctx context.Context, name string) (context.Context, *xray.Segment) {
	if !t.config.Enabled {
		return ctx, nil
	}

	ctx, seg := xray.BeginSegment(ctx, name)
	if seg != nil {
		// Add service metadata
		seg.AddAnnotation("service", t.config.ServiceName)
		seg.AddAnnotation("version", t.config.Version)

		// Add Lambda context if available
		if lc := GetLambdaContext(ctx); lc != nil {
			seg.AddAnnotation("aws_request_id", lc.AwsRequestID)
		}
	}

	return ctx, seg
}

// StartSubsegment starts a new X-Ray subsegment if tracing is enabled.
func (t *Tracer) StartSubsegment(ctx context.Context, name string) (context.Context, *xray.Segment) {
	if !t.config.Enabled {
		return ctx, nil
	}

	return xray.BeginSubsegment(ctx, name)
}

// AddAnnotation adds an annotation to the current segment.
func (t *Tracer) AddAnnotation(ctx context.Context, key string, value interface{}) {
	if !t.config.Enabled {
		return
	}

	if seg := xray.GetSegment(ctx); seg != nil {
		seg.AddAnnotation(key, value)
	}
}

// AddMetadata adds metadata to the current segment.
func (t *Tracer) AddMetadata(ctx context.Context, namespace string, value interface{}) {
	if !t.config.Enabled {
		return
	}

	if seg := xray.GetSegment(ctx); seg != nil {
		seg.AddMetadata(namespace, value)
	}
}

// AddError adds an error to the current segment.
func (t *Tracer) AddError(ctx context.Context, err error) {
	if !t.config.Enabled || err == nil {
		return
	}

	if seg := xray.GetSegment(ctx); seg != nil {
		seg.AddError(err)
	}
}

// SetHTTPRequest adds HTTP request information to the current segment.
func (t *Tracer) SetHTTPRequest(ctx context.Context, method, url string) {
	if !t.config.Enabled {
		return
	}

	if seg := xray.GetSegment(ctx); seg != nil {
		seg.GetHTTP().GetRequest().Method = method
		seg.GetHTTP().GetRequest().URL = url
	}
}

// SetHTTPResponse adds HTTP response information to the current segment.
func (t *Tracer) SetHTTPResponse(ctx context.Context, statusCode int, contentLength int64) {
	if !t.config.Enabled {
		return
	}

	if seg := xray.GetSegment(ctx); seg != nil {
		seg.GetHTTP().GetResponse().Status = statusCode
		seg.GetHTTP().GetResponse().ContentLength = int(contentLength)
	}
}

// Close closes a segment if it exists.
func (t *Tracer) Close(seg *xray.Segment, err error) {
	if seg == nil {
		return
	}

	if err != nil {
		seg.AddError(err)
	}

	seg.Close(err)
}

// WithTimer wraps a function with timing and tracing.
func (t *Tracer) WithTimer(ctx context.Context, name string, fn func(context.Context) error) error {
	start := time.Now()

	ctx, seg := t.StartSubsegment(ctx, name)
	defer func() {
		duration := time.Since(start)
		t.AddAnnotation(ctx, "duration_ms", duration.Milliseconds())
		t.Close(seg, nil)
	}()

	err := fn(ctx)
	if err != nil {
		t.AddError(ctx, err)
	}

	return err
}

// GetLambdaContext extracts Lambda context from Go context.
func GetLambdaContext(ctx context.Context) *lambdacontext.LambdaContext {
	if lc, ok := lambdacontext.FromContext(ctx); ok {
		return lc
	}
	return nil
}

// GetTraceID returns the current trace ID if available.
func GetTraceID(ctx context.Context) string {
	if seg := xray.GetSegment(ctx); seg != nil {
		return seg.TraceID
	}
	return ""
}

// GetSegmentID returns the current segment ID if available.
func GetSegmentID(ctx context.Context) string {
	if seg := xray.GetSegment(ctx); seg != nil {
		return seg.ID
	}
	return ""
}

// IsTracingEnabled returns true if tracing is enabled for the current context.
func (t *Tracer) IsTracingEnabled() bool {
	return t.config.Enabled
}

// CreateCorrelationID creates a correlation ID for request tracking.
// This uses the Lambda request ID if available, otherwise generates a new one.
func CreateCorrelationID(ctx context.Context) string {
	if lc := GetLambdaContext(ctx); lc != nil {
		return lc.AwsRequestID
	}

	// Fallback to trace ID if available
	if traceID := GetTraceID(ctx); traceID != "" {
		return traceID
	}

	// Generate a simple correlation ID as fallback
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// AddUserID adds user identification to the current segment.
func (t *Tracer) AddUserID(ctx context.Context, userID string) {
	if !t.config.Enabled || userID == "" {
		return
	}

	if seg := xray.GetSegment(ctx); seg != nil {
		seg.AddAnnotation("user_id", userID)
	}
}

// TraceFunction is a decorator that adds tracing to a function.
func (t *Tracer) TraceFunction(name string, fn func(context.Context) error) func(context.Context) error {
	return func(ctx context.Context) error {
		ctx, seg := t.StartSubsegment(ctx, name)
		defer t.Close(seg, nil)

		err := fn(ctx)
		if err != nil {
			t.AddError(ctx, err)
		}

		return err
	}
}
