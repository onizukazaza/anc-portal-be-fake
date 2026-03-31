package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/onizukazaza/anc-portal-be-fake/config"
	"github.com/onizukazaza/anc-portal-be-fake/internal/database"
	"github.com/onizukazaza/anc-portal-be-fake/internal/testkit"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/kafka"
)

type mockDBProvider struct {
	healthCheckFn func(ctx context.Context) error
}

func (m *mockDBProvider) Main() *pgxpool.Pool {
	return nil
}

func (m *mockDBProvider) External(_ string) (database.ExternalConn, error) {
	return nil, nil
}

func (m *mockDBProvider) Read() *pgxpool.Pool {
	return nil
}

func (m *mockDBProvider) Write() *pgxpool.Pool {
	return nil
}

func (m *mockDBProvider) HealthCheck(ctx context.Context) error {
	if m.healthCheckFn != nil {
		return m.healthCheckFn(ctx)
	}
	return nil
}

func (m *mockDBProvider) Close() {}

type mockKafkaProducer struct {
	publishFn func(ctx context.Context, msg kafka.Message) error
	calls     int
}

func (m *mockKafkaProducer) PublishMessage(ctx context.Context, msg kafka.Message) error {
	m.calls++
	if m.publishFn != nil {
		return m.publishFn(ctx, msg)
	}
	return nil
}

func testConfig(stage string) *config.Config {
	return &config.Config{
		StageStatus: stage,
		Server: config.Server{
			Port:         8080,
			AllowOrigins: []string{"*"},
			BodyLimit:    1024 * 1024,
			Timeout:      2 * time.Second,
			JWTSecretKey: "test-secret",
			JWTExpiry:    24 * time.Hour,
		},
		Swagger: config.Swagger{Enabled: false},
	}
}

func doRequest(t *testing.T, s *Server, method string, path string, body []byte) (int, map[string]any) {
	t.Helper()

	reader := bytes.NewReader(body)
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")

	res, err := s.app.Test(req, -1)
	testkit.MustNoError(t, err, "request")
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	testkit.MustNoError(t, err, "read response")

	if len(data) == 0 {
		return res.StatusCode, map[string]any{}
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return res.StatusCode, map[string]any{"raw": string(data)}
	}

	return res.StatusCode, payload
}

func TestHealthEndpoints(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		healthErr  error
		wantStatus int
		wantValue  string
	}{
		{
			name:       "healthz returns ok",
			path:       "/healthz",
			healthErr:  nil,
			wantStatus: http.StatusOK,
			wantValue:  "OK",
		},
		{
			name:       "healthz returns degraded when database fails",
			path:       "/healthz",
			healthErr:  errors.New("database unavailable"),
			wantStatus: http.StatusServiceUnavailable,
			wantValue:  "ERROR",
		},
		{
			name:       "ready returns ready",
			path:       "/ready",
			healthErr:  nil,
			wantStatus: http.StatusOK,
			wantValue:  "OK",
		},
		{
			name:       "ready returns not_ready when database fails",
			path:       "/ready",
			healthErr:  errors.New("database unavailable"),
			wantStatus: http.StatusServiceUnavailable,
			wantValue:  "ERROR",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := &mockDBProvider{
				healthCheckFn: func(_ context.Context) error {
					return tc.healthErr
				},
			}
			s := New(testConfig("local"), db, nil, nil, nil)

			status, payload := doRequest(t, s, http.MethodGet, tc.path, nil)
			testkit.Equal(t, status, tc.wantStatus, "status")

			actual, _ := payload["status"].(string)
			testkit.Equal(t, actual, tc.wantValue, "status payload")

			if tc.path == "/ready" && tc.wantStatus == http.StatusOK {
				result, _ := payload["result"].(map[string]any)
				if result != nil {
					data, _ := result["data"].(map[string]any)
					if data != nil {
						timestamp, _ := data["timestamp"].(string)
						if timestamp == "" {
							t.Fatal("expected timestamp in ready response")
						}
					}
				}
			}
		})
	}
}

func TestKafkaPublishEndpoint(t *testing.T) {
	tests := []struct {
		name          string
		stage         string
		producer      KafkaProducer
		producerMock  *mockKafkaProducer
		body          string
		wantStatus    int
		wantErr       string
		wantCalls     int
		wantRouteOpen bool
	}{
		{
			name:  "publish accepted when local and producer available",
			stage: "local",
			producer: &mockKafkaProducer{publishFn: func(_ context.Context, msg kafka.Message) error {
				if msg.Key != "u1" {
					return errors.New("unexpected key")
				}
				if msg.Type != "debug.message" {
					return errors.New("unexpected event type")
				}
				var payload map[string]any
				if err := json.Unmarshal(msg.Payload, &payload); err != nil {
					return err
				}
				if payload["message"] != "hello" {
					return errors.New("unexpected message")
				}
				return nil
			}},
			producerMock: &mockKafkaProducer{publishFn: func(_ context.Context, msg kafka.Message) error {
				if msg.Key != "u1" {
					return errors.New("unexpected key")
				}
				if msg.Type != "debug.message" {
					return errors.New("unexpected event type")
				}
				var payload map[string]any
				if err := json.Unmarshal(msg.Payload, &payload); err != nil {
					return err
				}
				if payload["message"] != "hello" {
					return errors.New("unexpected message")
				}
				return nil
			}},
			body:          `{"key":"u1","message":"hello"}`,
			wantStatus:    http.StatusAccepted,
			wantCalls:     1,
			wantRouteOpen: true,
		},
		{
			name:  "publish returns bad gateway on producer error",
			stage: "local",
			producer: &mockKafkaProducer{publishFn: func(_ context.Context, _ kafka.Message) error {
				return errors.New("broker down")
			}},
			producerMock: &mockKafkaProducer{publishFn: func(_ context.Context, _ kafka.Message) error {
				return errors.New("broker down")
			}},
			body:          `{"key":"u1","message":"hello"}`,
			wantStatus:    http.StatusBadGateway,
			wantErr:       "broker down",
			wantCalls:     1,
			wantRouteOpen: true,
		},
		{
			name:  "publish accepts explicit event type",
			stage: "local",
			producer: &mockKafkaProducer{publishFn: func(_ context.Context, msg kafka.Message) error {
				if msg.Type != "notification.created" {
					return errors.New("unexpected event type")
				}
				return nil
			}},
			producerMock: &mockKafkaProducer{publishFn: func(_ context.Context, msg kafka.Message) error {
				if msg.Type != "notification.created" {
					return errors.New("unexpected event type")
				}
				return nil
			}},
			body:          `{"key":"u1","eventType":"notification.created","message":"hello"}`,
			wantStatus:    http.StatusAccepted,
			wantCalls:     1,
			wantRouteOpen: true,
		},
		{
			name:          "publish validates required message",
			stage:         "local",
			producer:      &mockKafkaProducer{},
			producerMock:  &mockKafkaProducer{},
			body:          `{"key":"u1","message":"   "}`,
			wantStatus:    http.StatusBadRequest,
			wantErr:       "message is required",
			wantCalls:     0,
			wantRouteOpen: true,
		},
		{
			name:          "publish endpoint disabled outside local",
			stage:         "staging",
			producer:      &mockKafkaProducer{},
			producerMock:  &mockKafkaProducer{},
			body:          `{"key":"u1","message":"hello"}`,
			wantStatus:    http.StatusNotFound,
			wantCalls:     0,
			wantRouteOpen: false,
		},
		{
			name:          "publish endpoint disabled when producer is nil",
			stage:         "local",
			producer:      nil,
			producerMock:  nil,
			body:          `{"key":"u1","message":"hello"}`,
			wantStatus:    http.StatusNotFound,
			wantCalls:     0,
			wantRouteOpen: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.producerMock != nil {
				tc.producer = tc.producerMock
			}
			db := &mockDBProvider{}
			s := New(testConfig(tc.stage), db, tc.producer, nil, nil)

			status, payload := doRequest(t, s, http.MethodPost, "/v1/kafka/publish", []byte(tc.body))
			testkit.Equal(t, status, tc.wantStatus, "status")

			if tc.wantErr != "" {
				actualErr, _ := payload["message"].(string)
				testkit.Equal(t, actualErr, tc.wantErr, "error")
			}

			if tc.wantRouteOpen && tc.wantStatus == http.StatusAccepted {
				actual, _ := payload["status"].(string)
				testkit.Equal(t, actual, "OK", "status payload")
				actualMsg, _ := payload["message"].(string)
				testkit.Equal(t, actualMsg, "published", "message payload")
			}

			if tc.producerMock != nil {
				testkit.Equal(t, tc.producerMock.calls, tc.wantCalls, "publish calls")
			}
		})
	}
}
