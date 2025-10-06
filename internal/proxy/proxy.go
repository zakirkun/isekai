package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/zakirkun/isekai/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("isekai-proxy")

// Proxy handles request forwarding
type Proxy struct {
	client  *http.Client
	log     *logger.Logger
	timeout time.Duration
}

// New creates a new proxy instance
func New(timeout time.Duration, log *logger.Logger) *Proxy {
	return &Proxy{
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		log:     log,
		timeout: timeout,
	}
}

// Forward forwards a request to the target URL
func (p *Proxy) Forward(ctx context.Context, targetURL string, r *http.Request) (*http.Response, error) {
	// Start tracing span
	ctx, span := tracer.Start(ctx, "proxy.Forward",
		trace.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
			attribute.String("target.url", targetURL),
			attribute.String("client.ip", r.RemoteAddr),
		),
	)
	defer span.End()

	// Create a new request with context
	req, err := http.NewRequestWithContext(ctx, r.Method, targetURL, r.Body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Copy headers from original request
	req.Header = r.Header.Clone()

	// Inject trace context into headers for propagation
	otel.GetTextMapPropagator().Inject(ctx, NewHeaderCarrier(req.Header))

	// Set X-Forwarded headers
	req.Header.Set("X-Forwarded-For", r.RemoteAddr)
	req.Header.Set("X-Forwarded-Proto", "http")
	if r.TLS != nil {
		req.Header.Set("X-Forwarded-Proto", "https")
	}
	req.Header.Set("X-Forwarded-Host", r.Host)

	// Execute the request
	startTime := time.Now()
	resp, err := p.client.Do(req)
	duration := time.Since(startTime)

	// Record response metrics in span
	span.SetAttributes(
		attribute.Int64("http.response_time_ms", duration.Milliseconds()),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to forward request")
		p.log.Errorf("Failed to forward request to %s: %v (took %v)", targetURL, err, duration)
		return nil, fmt.Errorf("failed to forward request: %w", err)
	}

	// Record status code
	span.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode),
	)

	if resp.StatusCode >= 500 {
		span.SetStatus(codes.Error, fmt.Sprintf("server error: %d", resp.StatusCode))
	} else if resp.StatusCode >= 400 {
		span.SetStatus(codes.Error, fmt.Sprintf("client error: %d", resp.StatusCode))
	} else {
		span.SetStatus(codes.Ok, "success")
	}

	p.log.Debugf("Forwarded %s %s to %s - Status: %d (took %v)",
		r.Method, r.URL.Path, targetURL, resp.StatusCode, duration)

	return resp, nil
}

// CopyResponse copies the response to the response writer
func (p *Proxy) CopyResponse(w http.ResponseWriter, resp *http.Response) error {
	// Copy headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Write status code
	w.WriteHeader(resp.StatusCode)

	// Copy body
	_, err := io.Copy(w, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy response body: %w", err)
	}

	return nil
}

// ForwardAndCopy forwards a request and copies the response
func (p *Proxy) ForwardAndCopy(ctx context.Context, w http.ResponseWriter, r *http.Request, targetURL string) error {
	// Start tracing span for combined operation
	ctx, span := tracer.Start(ctx, "proxy.ForwardAndCopy",
		trace.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
			attribute.String("target.url", targetURL),
		),
	)
	defer span.End()

	resp, err := p.Forward(ctx, targetURL, r)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "forward failed")
		return err
	}
	defer resp.Body.Close()

	err = p.CopyResponse(w, resp)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "copy response failed")
	} else {
		span.SetStatus(codes.Ok, "success")
	}

	return err
}

// HeaderCarrier adapts http.Header to satisfy the TextMapCarrier interface
type HeaderCarrier http.Header

// Get returns the value associated with the passed key
func (hc HeaderCarrier) Get(key string) string {
	return http.Header(hc).Get(key)
}

// Set stores the key-value pair
func (hc HeaderCarrier) Set(key string, value string) {
	http.Header(hc).Set(key, value)
}

// Keys lists the keys stored in this carrier
func (hc HeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(hc))
	for k := range hc {
		keys = append(keys, k)
	}
	return keys
}

// NewHeaderCarrier creates a new header carrier
func NewHeaderCarrier(h http.Header) HeaderCarrier {
	return HeaderCarrier(h)
}
