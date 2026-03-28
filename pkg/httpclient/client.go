// pkg/httpclient — HTTP client สำหรับเรียก external API
//
//   - Functional Options pattern — config ได้ทุกอย่าง
//   - Retry ในตัว — ใช้ pkg/retry (exponential / constant / linear / custom)
//   - Connection pool — reuse TCP connections, keep-alive
//   - OTel tracing — trace ทุก outgoing request อัตโนมัติ
//   - JSON helper — encode request / decode response ให้
//   - Timeout ทุกระดับ — client-level + per-request
//   - Error ที่อ่านง่าย — status code + body ติดมาด้วย
//
// วิธีใช้:
//
//	client := httpclient.New(
//	    httpclient.BaseURL("https://api.example.com"),
//	    httpclient.Timeout(10*time.Second),
//	    httpclient.WithRetry(3),
//	    httpclient.WithHeader("X-API-Key", "secret"),
//	)
//
//	var result MyResponse
//	err := client.GetJSON(ctx, "/v1/users/1", &result)
//
//	var created Order
//	err := client.PostJSON(ctx, "/v1/orders", orderReq, &created)
package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/retry"
	"github.com/sony/gobreaker/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// ───────────────────────────────────────────────────────────────────
// Client
// ───────────────────────────────────────────────────────────────────

// Client เป็น HTTP client ที่มี retry, tracing, circuit breaker, connection pool ในตัว
type Client struct {
	http    *http.Client
	baseURL string
	headers map[string]string

	// retry
	retryEnabled bool
	retryOpts    []retry.Option

	// tracing
	tracingEnabled bool
	tracer         trace.Tracer

	// circuit breaker
	cb *gobreaker.CircuitBreaker[struct{}]
}

// New สร้าง Client พร้อม options ที่กำหนด
func New(opts ...Option) *Client {
	cfg := defaultConfig()
	for _, o := range opts {
		o(&cfg)
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   cfg.connectTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        cfg.maxIdleConns,
		MaxIdleConnsPerHost: cfg.maxIdleConnsPerHost,
		IdleConnTimeout:     cfg.idleConnTimeout,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableCompression:  false,
	}

	c := &Client{
		http: &http.Client{
			Timeout:   cfg.timeout,
			Transport: transport,
		},
		baseURL:        cfg.baseURL,
		headers:        cfg.headers,
		retryEnabled:   cfg.retryEnabled,
		retryOpts:      cfg.retryOpts,
		tracingEnabled: cfg.tracingEnabled,
	}

	if cfg.tracingEnabled {
		c.tracer = appOtel.Tracer(appOtel.TracerHTTPClient)
	}

	if cfg.cbEnabled {
		c.cb = gobreaker.NewCircuitBreaker[struct{}](cfg.cbSettings)
	}

	return c
}

// ───────────────────────────────────────────────────────────────────
// JSON Helpers — ใช้บ่อยสุด
// ───────────────────────────────────────────────────────────────────

// GetJSON ส่ง GET แล้ว decode JSON response เข้า dest
func (c *Client) GetJSON(ctx context.Context, path string, dest any) error {
	resp, err := c.Do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decodeJSON(resp, dest)
}

// PostJSON ส่ง POST พร้อม JSON body แล้ว decode response เข้า dest
func (c *Client) PostJSON(ctx context.Context, path string, body any, dest any) error {
	resp, err := c.DoJSON(ctx, http.MethodPost, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if dest == nil {
		return nil
	}
	return decodeJSON(resp, dest)
}

// PutJSON ส่ง PUT พร้อม JSON body แล้ว decode response เข้า dest
func (c *Client) PutJSON(ctx context.Context, path string, body any, dest any) error {
	resp, err := c.DoJSON(ctx, http.MethodPut, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if dest == nil {
		return nil
	}
	return decodeJSON(resp, dest)
}

// PatchJSON ส่ง PATCH พร้อม JSON body แล้ว decode response เข้า dest
func (c *Client) PatchJSON(ctx context.Context, path string, body any, dest any) error {
	resp, err := c.DoJSON(ctx, http.MethodPatch, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if dest == nil {
		return nil
	}
	return decodeJSON(resp, dest)
}

// DeleteJSON ส่ง DELETE แล้ว decode JSON response เข้า dest (ถ้ามี)
func (c *Client) DeleteJSON(ctx context.Context, path string, dest any) error {
	resp, err := c.Do(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if dest == nil {
		return nil
	}
	return decodeJSON(resp, dest)
}

// ───────────────────────────────────────────────────────────────────
// Core Methods
// ───────────────────────────────────────────────────────────────────

// DoJSON ส่ง request พร้อม JSON body — encode ให้อัตโนมัติ
func (c *Client) DoJSON(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("httpclient: marshal request body: %w", err)
		}
		reader = bytes.NewReader(data)
	}
	return c.Do(ctx, method, path, reader)
}

// Do ส่ง HTTP request พร้อม retry + tracing
// คืน *http.Response — caller ต้อง defer resp.Body.Close()
func (c *Client) Do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := c.baseURL + path

	// อ่าน body ไว้ก่อน เพื่อ retry ได้ (body อ่านได้แค่ครั้งเดียว)
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("httpclient: read request body: %w", err)
		}
	}

	var resp *http.Response

	execute := func(ctx context.Context) error {
		var bodyReader io.Reader
		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if err != nil {
			return fmt.Errorf("httpclient: create request: %w", err)
		}

		// default headers
		for k, v := range c.headers {
			req.Header.Set(k, v)
		}
		if bodyBytes != nil && req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}

		// inject W3C trace context สำหรับ distributed tracing
		if c.tracingEnabled {
			otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
		}

		resp, err = c.http.Do(req) //nolint:bodyclose // caller closes body via drainBody or decodeJSON
		if err != nil {
			return fmt.Errorf("httpclient: %s %s: %w", method, path, err)
		}

		// retry เฉพาะ 5xx (server error) — 4xx ไม่ retry เพราะ client ผิด
		if resp.StatusCode >= http.StatusInternalServerError {
			drainBody(resp)
			return &ResponseError{
				StatusCode: resp.StatusCode,
				Method:     method,
				URL:        url,
			}
		}

		// 4xx → return error แต่ไม่ retry
		if resp.StatusCode >= http.StatusBadRequest {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return &ResponseError{
				StatusCode: resp.StatusCode,
				Method:     method,
				URL:        url,
				Body:       truncateBody(string(respBody)),
				noRetry:    true,
			}
		}

		return nil
	}

	// wrap with tracing
	traced := func(ctx context.Context) error {
		if !c.tracingEnabled {
			return execute(ctx)
		}

		ctx, span := c.tracer.Start(ctx, fmt.Sprintf("HTTP %s", method),
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				attribute.String("http.method", method),
				attribute.String("http.url", url),
			),
		)
		defer span.End()

		err := execute(ctx)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		} else if resp != nil {
			span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
		}
		return err
	}

	// wrap with circuit breaker — ห่อ traced อีกชั้น
	// CB จะ "open" เมื่อ request fail ต่อเนื่อง → return ErrCircuitOpen ทันที
	call := traced
	if c.cb != nil {
		call = func(ctx context.Context) error {
			_, cbErr := c.cb.Execute(func() (struct{}, error) {
				if err := traced(ctx); err != nil {
					return struct{}{}, err
				}
				return struct{}{}, nil
			})
			if cbErr != nil {
				return cbErr
			}
			return nil
		}
	}

	// wrap with retry
	if c.retryEnabled {
		var clientErr error // เก็บ 4xx error แยก — ไม่ retry แต่ยังคง return error

		retryErr := retry.Do(ctx, func(ctx context.Context) error {
			err := call(ctx)
			// 4xx → หยุด retry ทันที แต่เก็บ error ไว้
			var respErr *ResponseError
			if isResponseError(err, &respErr) && respErr.noRetry {
				clientErr = err
				return nil
			}
			return err
		}, c.retryOpts...)

		if clientErr != nil {
			return nil, clientErr
		}
		if retryErr != nil {
			return nil, retryErr
		}
		return resp, nil
	}

	if err := call(ctx); err != nil {
		return nil, err
	}
	return resp, nil
}

// ───────────────────────────────────────────────────────────────────
// Internal helpers
// ───────────────────────────────────────────────────────────────────

func decodeJSON(resp *http.Response, dest any) error {
	if dest == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("httpclient: decode response: %w", err)
	}
	return nil
}

func drainBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
