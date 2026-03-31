package http

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/cmi/app"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/cmi/domain"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/dto"
	"github.com/onizukazaza/anc-portal-be-fake/internal/testkit"
)

// setupApp creates a Fiber app with the CMI handler route.
func setupApp(repo *fakeRepo) *fiber.App {
	fiberApp := fiber.New()
	svc := app.NewService(repo)
	h := NewHandler(svc)
	fiberApp.Get("/cmi/:job_id/request-policy-single-cmi", h.GetPolicyByJobID)
	return fiberApp
}

// doRequest sends a GET request to the handler and returns the parsed response.
func doRequest(t *testing.T, fiberApp *fiber.App, jobID string) (int, dto.ApiResponse) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/cmi/"+jobID+"/request-policy-single-cmi", nil)
	resp, err := fiberApp.Test(req, -1)
	testkit.MustNoError(t, err, "fiber.Test")
	defer resp.Body.Close()

	statusCode := resp.StatusCode
	body, err := io.ReadAll(resp.Body)
	testkit.MustNoError(t, err, "read body")

	var apiResp dto.ApiResponse
	testkit.MustNoError(t, json.Unmarshal(body, &apiResp), "unmarshal response")
	return statusCode, apiResp
}

func TestGetPolicyByJobID_Success(t *testing.T) {
	repo := &fakeRepo{
		exists: true,
		policy: &domain.CMIPolicy{
			JobID:     "job-001",
			JobType:   "cmi_only",
			JobStatus: "quotations",
			AgentID:   "agent-001",
			Motor:     &domain.MotorInfo{Year: "2025", Brand: "Toyota", Model: "Camry"},
		},
	}

	fiberApp := setupApp(repo)
	statusCode, apiResp := doRequest(t, fiberApp, "job-001")

	testkit.Equal(t, statusCode, http.StatusOK, "status code")
	testkit.Equal(t, apiResp.Status, "OK", "response status")
	testkit.Equal(t, apiResp.Message, "success", "message")
	testkit.NotNil(t, apiResp.Result, "result should not be nil")
}

func TestGetPolicyByJobID_JobNotFound(t *testing.T) {
	repo := &fakeRepo{exists: false}

	fiberApp := setupApp(repo)
	statusCode, apiResp := doRequest(t, fiberApp, "missing-job")

	testkit.Equal(t, statusCode, http.StatusNotFound, "status code")
	testkit.Equal(t, apiResp.Status, "ERROR", "response status")
	testkit.Equal(t, apiResp.Message, "job not found", "message")
	testkit.Contains(t, extractTraceID(t, apiResp), dto.TraceCMIJobNotFound, "trace_id")
}

func TestGetPolicyByJobID_RepoError(t *testing.T) {
	repo := &fakeRepo{existErr: errDB}

	fiberApp := setupApp(repo)
	statusCode, apiResp := doRequest(t, fiberApp, "job-001")

	testkit.Equal(t, statusCode, http.StatusInternalServerError, "status code")
	testkit.Equal(t, apiResp.Status, "ERROR", "response status")
	testkit.Equal(t, apiResp.Message, "internal error", "message")
	testkit.Contains(t, extractTraceID(t, apiResp), dto.TraceCMIInternalError, "trace_id")
}

func TestGetPolicyByJobID_FindPolicyError(t *testing.T) {
	repo := &fakeRepo{exists: true, findErr: errDB}

	fiberApp := setupApp(repo)
	statusCode, apiResp := doRequest(t, fiberApp, "job-001")

	testkit.Equal(t, statusCode, http.StatusInternalServerError, "status code")
	testkit.Equal(t, apiResp.Status, "ERROR", "response status")
	testkit.Contains(t, extractTraceID(t, apiResp), dto.TraceCMIInternalError, "trace_id")
}

// extractTraceID reads the trace_id from the result field of an error response.
func extractTraceID(t *testing.T, apiResp dto.ApiResponse) string {
	t.Helper()
	if apiResp.Result == nil {
		return ""
	}
	// Result is deserialized as map[string]interface{} from JSON.
	m, ok := apiResp.Result.(map[string]interface{})
	if !ok {
		return ""
	}
	tid, _ := m["trace_id"].(string)
	return tid
}
