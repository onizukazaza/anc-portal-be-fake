package httpclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/onizukazaza/anc-portal-be-fake/pkg/retry"
)

// ───────────────────────────────────────────────────────────────────
// Helpers >>
// ───────────────────────────────────────────────────────────────────

type testPayload struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func newTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// ───────────────────────────────────────────────────────────────────
// Tests — New / Options
// ───────────────────────────────────────────────────────────────────

func TestNew_Defaults(t *testing.T) {
	c := New()
	if c.baseURL != "" {
		t.Errorf("expected empty baseURL, got %q", c.baseURL)
	}
	if !c.tracingEnabled {
		t.Error("expected tracing enabled by default")
	}
	if c.retryEnabled {
		t.Error("expected retry disabled by default")
	}
}

func TestNew_WithOptions(t *testing.T) {
	c := New(
		BaseURL("https://api.test.com"),
		Timeout(5*time.Second),
		WithHeader("X-Api-Key", "secret"),
		WithRetry(3),
		WithoutTracing(),
		MaxIdleConns(50),
		MaxIdleConnsPerHost(5),
	)

	if c.baseURL != "https://api.test.com" {
		t.Errorf("expected base URL, got %q", c.baseURL)
	}
	if c.tracingEnabled {
		t.Error("expected tracing disabled")
	}
	if !c.retryEnabled {
		t.Error("expected retry enabled")
	}
	if c.headers["X-Api-Key"] != "secret" {
		t.Error("expected header to be set")
	}
}

func TestNew_WithHeaders(t *testing.T) {
	c := New(WithHeaders(map[string]string{
		"Authorization": "Bearer token",
		"X-Request-ID":  "abc",
	}))
	if c.headers["Authorization"] != "Bearer token" {
		t.Error("expected Authorization header")
	}
	if c.headers["X-Request-ID"] != "abc" {
		t.Error("expected X-Request-ID header")
	}
}

func TestNew_WithRetryOptions(t *testing.T) {
	c := New(WithRetryOptions(
		retry.MaxAttempts(5),
		retry.Backoff(100*time.Millisecond),
		retry.WithBackoffFunc(retry.ConstantBackoff),
	))
	if !c.retryEnabled {
		t.Error("expected retry enabled")
	}
	if len(c.retryOpts) != 3 {
		t.Errorf("expected 3 retry opts, got %d", len(c.retryOpts))
	}
}

// ───────────────────────────────────────────────────────────────────
// Tests — GetJSON
// ───────────────────────────────────────────────────────────────────

func TestGetJSON_Success(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testPayload{Name: "test", Value: 42})
	})
	defer srv.Close()

	c := New(BaseURL(srv.URL), WithoutTracing())
	var result testPayload
	if err := c.GetJSON(context.Background(), "/data", &result); err != nil {
		t.Fatal(err)
	}
	if result.Name != "test" || result.Value != 42 {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestGetJSON_404(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	})
	defer srv.Close()

	c := New(BaseURL(srv.URL), WithoutTracing())
	var result testPayload
	err := c.GetJSON(context.Background(), "/missing", &result)
	if err == nil {
		t.Fatal("expected error for 404")
	}

	respErr, ok := IsResponseError(err)
	if !ok {
		t.Fatalf("expected ResponseError, got %T", err)
	}
	if respErr.StatusCode != 404 {
		t.Errorf("expected 404, got %d", respErr.StatusCode)
	}
	if !respErr.IsClientError() {
		t.Error("expected IsClientError() = true")
	}
}

// ───────────────────────────────────────────────────────────────────
// Tests — PostJSON
// ───────────────────────────────────────────────────────────────────

func TestPostJSON_Success(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected Content-Type: application/json")
		}

		var body testPayload
		json.NewDecoder(r.Body).Decode(&body)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testPayload{Name: body.Name, Value: body.Value * 2})
	})
	defer srv.Close()

	c := New(BaseURL(srv.URL), WithoutTracing())
	var result testPayload
	err := c.PostJSON(context.Background(), "/create", testPayload{Name: "new", Value: 10}, &result)
	if err != nil {
		t.Fatal(err)
	}
	if result.Name != "new" || result.Value != 20 {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestPostJSON_NilDest(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	defer srv.Close()

	c := New(BaseURL(srv.URL), WithoutTracing())
	err := c.PostJSON(context.Background(), "/fire-and-forget", testPayload{Name: "event"}, nil)
	if err != nil {
		t.Fatal(err)
	}
}

// ───────────────────────────────────────────────────────────────────
// Tests — PutJSON / PatchJSON / DeleteJSON
// ───────────────────────────────────────────────────────────────────

func TestPutJSON_Success(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testPayload{Name: "updated", Value: 1})
	})
	defer srv.Close()

	c := New(BaseURL(srv.URL), WithoutTracing())
	var result testPayload
	err := c.PutJSON(context.Background(), "/update/1", testPayload{Name: "updated"}, &result)
	if err != nil {
		t.Fatal(err)
	}
	if result.Name != "updated" {
		t.Errorf("unexpected: %+v", result)
	}
}

func TestPatchJSON_Success(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testPayload{Name: "patched", Value: 99})
	})
	defer srv.Close()

	c := New(BaseURL(srv.URL), WithoutTracing())
	var result testPayload
	err := c.PatchJSON(context.Background(), "/patch/1", testPayload{Value: 99}, &result)
	if err != nil {
		t.Fatal(err)
	}
	if result.Value != 99 {
		t.Errorf("unexpected: %+v", result)
	}
}

func TestDeleteJSON_Success(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"deleted": true})
	})
	defer srv.Close()

	c := New(BaseURL(srv.URL), WithoutTracing())
	var result map[string]bool
	err := c.DeleteJSON(context.Background(), "/delete/1", &result)
	if err != nil {
		t.Fatal(err)
	}
	if !result["deleted"] {
		t.Error("expected deleted=true")
	}
}

func TestDeleteJSON_NilDest(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	defer srv.Close()

	c := New(BaseURL(srv.URL), WithoutTracing())
	err := c.DeleteJSON(context.Background(), "/delete/1", nil)
	if err != nil {
		t.Fatal(err)
	}
}

// ───────────────────────────────────────────────────────────────────
// Tests — Default Headers
// ───────────────────────────────────────────────────────────────────

func TestDefaultHeaders(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("expected Authorization header")
		}
		if r.Header.Get("X-Custom") != "custom-value" {
			t.Error("expected X-Custom header")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})
	defer srv.Close()

	c := New(
		BaseURL(srv.URL),
		WithoutTracing(),
		WithHeader("Authorization", "Bearer test-token"),
		WithHeader("X-Custom", "custom-value"),
	)
	var result map[string]any
	if err := c.GetJSON(context.Background(), "/", &result); err != nil {
		t.Fatal(err)
	}
}

// ───────────────────────────────────────────────────────────────────
// Tests — Retry
// ───────────────────────────────────────────────────────────────────

func TestRetry_ServerError(t *testing.T) {
	var attempts atomic.Int32
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testPayload{Name: "success", Value: int(n)})
	})
	defer srv.Close()

	c := New(
		BaseURL(srv.URL),
		WithoutTracing(),
		WithRetryOptions(
			retry.MaxAttempts(3),
			retry.Backoff(10*time.Millisecond),
			retry.WithBackoffFunc(retry.ConstantBackoff),
		),
	)
	var result testPayload
	err := c.GetJSON(context.Background(), "/flaky", &result)
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if result.Name != "success" {
		t.Errorf("unexpected: %+v", result)
	}
	if attempts.Load() != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestRetry_Exhausted(t *testing.T) {
	var attempts atomic.Int32
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	})
	defer srv.Close()

	c := New(
		BaseURL(srv.URL),
		WithoutTracing(),
		WithRetryOptions(
			retry.MaxAttempts(2),
			retry.Backoff(10*time.Millisecond),
			retry.WithBackoffFunc(retry.ConstantBackoff),
		),
	)
	var result testPayload
	err := c.GetJSON(context.Background(), "/always-fail", &result)
	if err == nil {
		t.Fatal("expected error after exhausted retries")
	}
	if attempts.Load() != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts.Load())
	}

	respErr, ok := IsResponseError(err)
	if !ok {
		t.Fatalf("expected ResponseError, got %T: %v", err, err)
	}
	if !respErr.IsServerError() {
		t.Error("expected IsServerError() = true")
	}
}

func TestRetry_NoRetryOn4xx(t *testing.T) {
	var attempts atomic.Int32
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid input"}`))
	})
	defer srv.Close()

	c := New(
		BaseURL(srv.URL),
		WithoutTracing(),
		WithRetry(3),
	)
	var result testPayload
	err := c.GetJSON(context.Background(), "/bad", &result)
	if err == nil {
		t.Fatal("expected error for 400")
	}
	if attempts.Load() != 1 {
		t.Errorf("expected 1 attempt (no retry for 4xx), got %d", attempts.Load())
	}
}

// ───────────────────────────────────────────────────────────────────
// Tests — Error types
// ───────────────────────────────────────────────────────────────────

func TestResponseError_ServerError(t *testing.T) {
	e := &ResponseError{StatusCode: 503, Method: "GET", URL: "/api"}
	if !e.IsServerError() {
		t.Error("expected IsServerError() = true")
	}
	if e.IsClientError() {
		t.Error("expected IsClientError() = false")
	}
}

func TestResponseError_ClientError(t *testing.T) {
	e := &ResponseError{StatusCode: 422, Method: "POST", URL: "/api", Body: `{"error":"invalid"}`}
	if e.IsServerError() {
		t.Error("expected IsServerError() = false")
	}
	if !e.IsClientError() {
		t.Error("expected IsClientError() = true")
	}
	if e.Error() == "" {
		t.Error("expected non-empty error message")
	}
}

func TestResponseError_ErrorMessage(t *testing.T) {
	e := &ResponseError{StatusCode: 500, Method: "GET", URL: "https://api.test.com/v1"}
	expected := "httpclient: GET https://api.test.com/v1 returned 500"
	if e.Error() != expected {
		t.Errorf("expected %q, got %q", expected, e.Error())
	}

	e.Body = "internal error"
	expected = "httpclient: GET https://api.test.com/v1 returned 500: internal error"
	if e.Error() != expected {
		t.Errorf("expected %q, got %q", expected, e.Error())
	}
}

func TestIsResponseError(t *testing.T) {
	_, ok := IsResponseError(nil)
	if ok {
		t.Error("expected false for nil")
	}

	respErr, ok := IsResponseError(&ResponseError{StatusCode: 404})
	if !ok || respErr.StatusCode != 404 {
		t.Error("expected true + 404")
	}
}

// ───────────────────────────────────────────────────────────────────
// Tests — Context cancellation
// ───────────────────────────────────────────────────────────────────

func TestDo_ContextCancelled(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	c := New(BaseURL(srv.URL), WithoutTracing(), Timeout(100*time.Millisecond))
	ctx := context.Background()
	var result testPayload
	err := c.GetJSON(ctx, "/slow", &result)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

// ───────────────────────────────────────────────────────────────────
// Tests — DoJSON (raw access)
// ───────────────────────────────────────────────────────────────────

func TestDoJSON_RawResponse(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "abc-123")
		json.NewEncoder(w).Encode(testPayload{Name: "raw", Value: 1})
	})
	defer srv.Close()

	c := New(BaseURL(srv.URL), WithoutTracing())
	resp, err := c.DoJSON(context.Background(), http.MethodGet, "/raw", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("X-Request-Id") != "abc-123" {
		t.Error("expected X-Request-Id header in response")
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestPostJSON_WithRetry(t *testing.T) {
	var attempts atomic.Int32
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		var body testPayload
		json.NewDecoder(r.Body).Decode(&body)

		n := attempts.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testPayload{Name: body.Name, Value: body.Value})
	})
	defer srv.Close()

	c := New(
		BaseURL(srv.URL),
		WithoutTracing(),
		WithRetryOptions(
			retry.MaxAttempts(3),
			retry.Backoff(10*time.Millisecond),
			retry.WithBackoffFunc(retry.ConstantBackoff),
		),
	)

	var result testPayload
	err := c.PostJSON(context.Background(), "/retry-post", testPayload{Name: "retry", Value: 42}, &result)
	if err != nil {
		t.Fatal(err)
	}
	if result.Name != "retry" || result.Value != 42 {
		t.Errorf("unexpected: %+v", result)
	}
	if attempts.Load() != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts.Load())
	}
}
