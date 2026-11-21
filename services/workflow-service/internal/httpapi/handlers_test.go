package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/workflow-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/workflow-service/internal/workflowdb"
)

type fakeWFStore struct {
	requesterID uuid.UUID
}

func (f *fakeWFStore) CreateDefinition(_ context.Context, _ []uuid.UUID, _ workflowdb.CreateWorkflowDefinitionParams, _ []workflowdb.CreateWorkflowStepParams) (workflowdb.CreateWorkflowDefinitionRow, error) {
	return workflowdb.CreateWorkflowDefinitionRow{ID: uuid.MustParse("10000000-0000-0000-0000-000000000050"), RowVersion: 1}, nil
}

func (f *fakeWFStore) ListDefinitionsWithCount(_ context.Context, scope []uuid.UUID, _ workflowdb.ListWorkflowDefinitionsParams) ([]workflowdb.ListWorkflowDefinitionsRow, int64, error) {
	return []workflowdb.ListWorkflowDefinitionsRow{{ID: uuid.New(), OrganizationID: scope[0], Code: "PO_APR", DocumentType: "PurchaseOrder", Version: 1, IsActive: true, RowVersion: 1}}, 1, nil
}

func (f *fakeWFStore) CreateAndEmit(_ context.Context, _ []uuid.UUID, _ workflowdb.CreateWorkflowInstanceParams, _ uuid.UUID, _ workflowdb.InsertOutboxParams) (workflowdb.CreateWorkflowInstanceRow, uuid.UUID, error) {
	return workflowdb.CreateWorkflowInstanceRow{ID: uuid.MustParse("10000000-0000-0000-0000-000000000051"), Status: "Pending", RowVersion: 1},
		uuid.MustParse("10000000-0000-0000-0000-000000000052"), nil
}

func (f *fakeWFStore) GetWithTasks(_ context.Context, scope []uuid.UUID, id uuid.UUID) (workflowdb.WfWorkflowInstance, []workflowdb.ListWorkflowTasksRow, error) {
	return workflowdb.WfWorkflowInstance{ID: id, OrganizationID: scope[0], DocumentRef: "PO-001", Status: "Pending", RowVersion: 1}, nil, nil
}

func (f *fakeWFStore) ListWithCount(_ context.Context, _ []uuid.UUID, _ workflowdb.ListWorkflowInstancesParams) ([]workflowdb.ListWorkflowInstancesRow, int64, error) {
	return nil, 0, nil
}

func (f *fakeWFStore) ListTasksWithCount(_ context.Context, scope []uuid.UUID, _ workflowdb.ListWorkflowTasksParams) ([]workflowdb.ListWorkflowTasksRow, int64, error) {
	return []workflowdb.ListWorkflowTasksRow{{
		ID:             uuid.MustParse("10000000-0000-0000-0000-000000000052"),
		OrganizationID: scope[0],
		InstanceID:     uuid.MustParse("10000000-0000-0000-0000-000000000051"),
		DocumentRef:    "PO-001",
		InstanceStatus: "Pending",
		CreatedBy:      f.requesterID,
		RowVersion:     1,
	}}, 1, nil
}

func (f *fakeWFStore) DecideTaskAndEmit(_ context.Context, _ []uuid.UUID, taskID, actorID, _ uuid.UUID, _ *int32, decision, comment string) (repository.TaskDecisionResult, error) {
	if decision == "Approve" && actorID == f.requesterID {
		return repository.TaskDecisionResult{}, repository.ErrSoDViolation
	}
	st := "Approved"
	if decision == "Reject" {
		st = "Rejected"
	}
	return repository.TaskDecisionResult{
		TaskID:         taskID,
		InstanceID:     uuid.MustParse("10000000-0000-0000-0000-000000000051"),
		DocumentRef:    "PO-001",
		Decision:       decision,
		Comment:        comment,
		InstanceStatus: st,
		TaskRowVersion: 2,
	}, nil
}

func wfHdrs(userID, perms string) map[string]string {
	return map[string]string{
		"X-Org-Scope":   "00000000-0000-0000-0000-000000000001",
		"X-User-ID":     userID,
		"X-Permissions": perms,
	}
}

func callWF(t *testing.T, h http.Handler, method, path string, body any, hdrs map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	for k, v := range hdrs {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestWorkflowSoDRBACApproveAndReject(t *testing.T) {
	requester := "00000000-0000-0000-0000-000000000010"
	approver := "00000000-0000-0000-0000-000000000020"
	store := &fakeWFStore{requesterID: uuid.MustParse(requester)}
	mux := http.NewServeMux()
	NewServer(store).Register(mux)
	h := httpx.Middleware(mux)

	if rr := callWF(t, h, http.MethodGet, "/workflow-tasks", nil, wfHdrs(requester, "ap:read")); rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without workflow:read, got %d", rr.Code)
	}

	// Missing If-Match must be rejected with 428 Precondition Required.
	if rr := callWF(t, h, http.MethodPost, "/workflow-tasks/10000000-0000-0000-0000-000000000052/approve", nil, wfHdrs(approver, "workflow:approve")); rr.Code != http.StatusPreconditionRequired {
		t.Fatalf("expected 428 without If-Match, got %d (%s)", rr.Code, rr.Body.String())
	}

	reqHdrs := wfHdrs(requester, "workflow:approve")
	reqHdrs["If-Match"] = `"1"`
	rr := callWF(t, h, http.MethodPost, "/workflow-tasks/10000000-0000-0000-0000-000000000052/approve", nil, reqHdrs)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 SOD_VIOLATION when requester self-approves, got %d (%s)", rr.Code, rr.Body.String())
	}

	aprHdrs := wfHdrs(approver, "workflow:approve")
	aprHdrs["If-Match"] = `"1"`
	rr = callWF(t, h, http.MethodPost, "/workflow-tasks/10000000-0000-0000-0000-000000000052/approve", map[string]any{"comment": "disetujui finance"}, aprHdrs)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 when distinct approver approves, got %d (%s)", rr.Code, rr.Body.String())
	}
	var dec repository.TaskDecisionResult
	_ = json.Unmarshal(rr.Body.Bytes(), &dec)
	if dec.Comment != "disetujui finance" {
		t.Fatalf("expected comment to be preserved, got %q", dec.Comment)
	}

	rr = callWF(t, h, http.MethodPost, "/workflow-tasks/10000000-0000-0000-0000-000000000052/reject", map[string]any{"notes": "kurang dokumen"}, aprHdrs)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 on reject, got %d (%s)", rr.Code, rr.Body.String())
	}

	rr = callWF(t, h, http.MethodGet, "/workflow-tasks?status=pending", nil, wfHdrs(approver, "workflow:read"))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 listing workflow tasks, got %d", rr.Code)
	}
}

