// Package httpapi wires workflow-service REST endpoints (/workflow-definitions,
// /workflow-instances, /workflow-tasks with SoD-enforced approve/reject).
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/workflow-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/workflow-service/internal/workflowdb"
)

type Store interface {
	CreateDefinition(ctx context.Context, scope []uuid.UUID, arg workflowdb.CreateWorkflowDefinitionParams, steps []workflowdb.CreateWorkflowStepParams) (workflowdb.CreateWorkflowDefinitionRow, error)
	ListDefinitionsWithCount(ctx context.Context, scope []uuid.UUID, arg workflowdb.ListWorkflowDefinitionsParams) ([]workflowdb.ListWorkflowDefinitionsRow, int64, error)
	CreateAndEmit(ctx context.Context, scope []uuid.UUID, arg workflowdb.CreateWorkflowInstanceParams, assigneeID uuid.UUID, ev workflowdb.InsertOutboxParams) (workflowdb.CreateWorkflowInstanceRow, uuid.UUID, error)
	GetWithTasks(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (workflowdb.WfWorkflowInstance, []workflowdb.ListWorkflowTasksRow, error)
	ListWithCount(ctx context.Context, scope []uuid.UUID, arg workflowdb.ListWorkflowInstancesParams) ([]workflowdb.ListWorkflowInstancesRow, int64, error)
	ListTasksWithCount(ctx context.Context, scope []uuid.UUID, arg workflowdb.ListWorkflowTasksParams) ([]workflowdb.ListWorkflowTasksRow, int64, error)
	DecideTaskAndEmit(ctx context.Context, scope []uuid.UUID, taskID, actorID, correlationID uuid.UUID, expectedVer *int32, decision string) (repository.TaskDecisionResult, error)
}

type Server struct {
	repo Store
	idem *httpx.IdemStore
}

func NewServer(repo Store) *Server {
	return &Server{repo: repo, idem: httpx.NewIdemStore()}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /workflow-definitions", s.createDef)
	mux.HandleFunc("GET /workflow-definitions", s.listDefs)
	mux.HandleFunc("POST /workflow-instances", s.createInstance)
	mux.HandleFunc("GET /workflow-instances", s.listInstances)
	mux.HandleFunc("GET /workflow-instances/{id}", s.getInstance)
	mux.HandleFunc("GET /workflow-tasks", s.listTasks)
	mux.HandleFunc("POST /workflow-tasks/{id}/approve", s.approveTask)
	mux.HandleFunc("POST /workflow-tasks/{id}/reject", s.rejectTask)
}

type stepReq struct {
	StepOrder      int32           `json:"stepOrder"`
	ApproverRule   json.RawMessage `json:"approverRule"`
	EscalationRule json.RawMessage `json:"escalationRule"`
}

type defReq struct {
	Code         string    `json:"code"`
	DocumentType string    `json:"documentType"`
	Steps        []stepReq `json:"steps"`
}

type createInstanceReq struct {
	DefinitionID string `json:"definitionId"`
	DocumentRef  string `json:"documentRef"`
	AssigneeID   string `json:"assigneeId"`
}

func (s *Server) createDef(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "workflow:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req defReq
	key, hash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	code := strings.TrimSpace(req.Code)
	docType := strings.TrimSpace(req.DocumentType)
	if code == "" || docType == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code dan documentType wajib diisi"))
		return
	}
	steps := make([]workflowdb.CreateWorkflowStepParams, 0, len(req.Steps))
	for i, st := range req.Steps {
		order := st.StepOrder
		if order <= 0 {
			order = int32(i + 1)
		}
		rule := st.ApproverRule
		if len(rule) == 0 {
			rule = json.RawMessage(`{"role":"manager"}`)
		}
		steps = append(steps, workflowdb.CreateWorkflowStepParams{
			StepOrder:      order,
			ApproverRule:   rule,
			EscalationRule: st.EscalationRule,
		})
	}
	created, err := s.repo.CreateDefinition(r.Context(), scope, workflowdb.CreateWorkflowDefinitionParams{
		OrganizationID: scope[0],
		Code:           code,
		DocumentType:   docType,
		Version:        1,
		IsActive:       true,
		CreatedBy:      actor,
	}, steps)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "kode definisi workflow sudah ada"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, key, hash, http.StatusCreated, created.RowVersion, map[string]any{
		"id":           created.ID,
		"code":         code,
		"documentType": docType,
		"rowVersion":   created.RowVersion,
	})
}

func (s *Server) listDefs(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "workflow:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var docType *string
	if dt := strings.TrimSpace(r.URL.Query().Get("documentType")); dt != "" {
		docType = &dt
	}
	rows, total, err := s.repo.ListDefinitionsWithCount(r.Context(), scope, workflowdb.ListWorkflowDefinitionsParams{
		Limit:        limit,
		Offset:       offset,
		DocumentType: docType,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, d := range rows {
		out = append(out, map[string]any{
			"id":             d.ID,
			"organizationId": d.OrganizationID,
			"code":           d.Code,
			"documentType":   d.DocumentType,
			"version":        d.Version,
			"isActive":       d.IsActive,
			"rowVersion":     d.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) createInstance(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "workflow:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req createInstanceReq
	key, hash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	defID, err := httpx.ParseUUIDParam("definitionId", req.DefinitionID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	docRef := strings.TrimSpace(req.DocumentRef)
	if docRef == "" {
		httpx.WriteError(w, r, httpx.BadRequest("documentRef wajib diisi"))
		return
	}
	assigneeID := actor
	if strings.TrimSpace(req.AssigneeID) != "" {
		assigneeID, err = httpx.ParseUUIDParam("assigneeId", req.AssigneeID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
	}
	step := int32(1)
	payload, _ := json.Marshal(map[string]any{
		"definitionId": defID,
		"documentRef":  docRef,
		"requesterId":  actor,
		"assigneeId":   assigneeID,
	})
	ev := workflowdb.InsertOutboxParams{
		AggregateType:  "WorkflowInstance",
		EventType:      "workflow.approval.requested",
		Payload:        payload,
		OrganizationID: scope[0],
		CorrelationID:  httpx.CorrelationUUID(r),
	}
	created, taskID, err := s.repo.CreateAndEmit(r.Context(), scope, workflowdb.CreateWorkflowInstanceParams{
		OrganizationID: scope[0],
		DefinitionID:   defID,
		DocumentRef:    docRef,
		CurrentStep:    &step,
		CreatedBy:      actor,
	}, assigneeID, ev)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, key, hash, http.StatusCreated, created.RowVersion, map[string]any{
		"id":          created.ID,
		"taskId":      taskID,
		"documentRef": docRef,
		"status":      created.Status,
		"rowVersion":  created.RowVersion,
	})
}

func (s *Server) listInstances(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "workflow:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var stPtr *string
	if st := strings.TrimSpace(r.URL.Query().Get("status")); st != "" {
		stPtr = &st
	}
	rows, total, err := s.repo.ListWithCount(r.Context(), scope, workflowdb.ListWorkflowInstancesParams{
		Limit:  limit,
		Offset: offset,
		Status: stPtr,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, i := range rows {
		out = append(out, map[string]any{
			"id":             i.ID,
			"organizationId": i.OrganizationID,
			"definitionId":   i.DefinitionID,
			"documentRef":    i.DocumentRef,
			"status":         i.Status,
			"currentStep":    i.CurrentStep,
			"rowVersion":     i.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) getInstance(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "workflow:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	inst, tasks, err := s.repo.GetWithTasks(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("workflow instance tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, inst.RowVersion, map[string]any{
		"id":             inst.ID,
		"organizationId": inst.OrganizationID,
		"definitionId":   inst.DefinitionID,
		"documentRef":    inst.DocumentRef,
		"status":         inst.Status,
		"currentStep":    inst.CurrentStep,
		"rowVersion":     inst.RowVersion,
		"tasks":          formatTasks(tasks),
	})
}

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "workflow:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var instPtr, assigneePtr *uuid.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("instanceId")); raw != "" {
		uid, err := httpx.ParseUUIDParam("instanceId", raw)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		instPtr = &uid
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("assigneeId")); raw != "" {
		uid, err := httpx.ParseUUIDParam("assigneeId", raw)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		assigneePtr = &uid
	}
	var pendingOnly *bool
	switch strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status"))) {
	case "pending":
		t := true
		pendingOnly = &t
	case "decided":
		f := false
		pendingOnly = &f
	}
	rows, total, err := s.repo.ListTasksWithCount(r.Context(), scope, workflowdb.ListWorkflowTasksParams{
		Limit:       limit,
		Offset:      offset,
		InstanceID:  instPtr,
		AssigneeID:  assigneePtr,
		PendingOnly: pendingOnly,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: formatTasks(rows),
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) approveTask(w http.ResponseWriter, r *http.Request) {
	s.decideTask(w, r, "Approve")
}

func (s *Server) rejectTask(w http.ResponseWriter, r *http.Request) {
	s.decideTask(w, r, "Reject")
}

func (s *Server) decideTask(w http.ResponseWriter, r *http.Request, decision string) {
	if err := httpx.RequirePermission(r, "workflow:approve"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	taskID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var expectedVer *int32
	if strings.TrimSpace(r.Header.Get("If-Match")) != "" {
		v, err := httpx.IfMatch(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		expectedVer = &v
	}
	res, err := s.repo.DecideTaskAndEmit(r.Context(), scope, taskID, actor, httpx.CorrelationUUID(r), expectedVer, decision)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			httpx.WriteError(w, r, httpx.NotFound("workflow task tidak ditemukan"))
		case errors.Is(err, repository.ErrSoDViolation):
			httpx.WriteError(w, r, &httpx.APIError{Status: http.StatusForbidden, Code: "SOD_VIOLATION", Message: err.Error()})
		case errors.Is(err, repository.ErrAlreadyDecided):
			httpx.WriteError(w, r, httpx.Conflict("ALREADY_DECIDED", err.Error()))
		case errors.Is(err, repository.ErrVersionConflict):
			httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", err.Error()))
		default:
			httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		}
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, res.TaskRowVersion, res)
}

func formatTasks(rows []workflowdb.ListWorkflowTasksRow) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, t := range rows {
		out = append(out, map[string]any{
			"id":             t.ID,
			"organizationId": t.OrganizationID,
			"instanceId":     t.InstanceID,
			"assigneeId":     t.AssigneeID,
			"decision":       t.Decision,
			"documentRef":    t.DocumentRef,
			"instanceStatus": t.InstanceStatus,
			"createdBy":      t.CreatedBy,
			"rowVersion":     t.RowVersion,
		})
	}
	return out
}
