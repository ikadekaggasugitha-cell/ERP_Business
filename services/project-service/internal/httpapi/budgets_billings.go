package httpapi

import (
	"math/big"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/project-service/internal/prjdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func (s *Server) listBudgets(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	pid, err := httpx.ParsePathUUID(r, "projectId")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, err := s.store.ListBudgets(r.Context(), scope, pid)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]budgetDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toBudgetDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *Server) createBudget(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	pid, err := httpx.ParsePathUUID(r, "projectId")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		WBSElementID string `json:"wbsElementId"`
		Version      int32  `json:"version"`
		BudgetAmount string `json:"budgetAmount"`
		IsBaseline   bool   `json:"isBaseline"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	wbsID, err := uuid.Parse(req.WBSElementID)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("valid wbsElementId is required"))
		return
	}
	amt, err := parsePositiveNumeric(req.BudgetAmount, "budgetAmount")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver := req.Version
	if ver <= 0 {
		ver = 1
	}
	out, err := s.store.CreateBudget(r.Context(), scope, prjdb.CreateProjectBudgetParams{
		OrganizationID: scope[0], ProjectID: pid, WbsElementID: wbsID,
		Version: ver, BudgetAmount: amt, IsBaseline: req.IsBaseline, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "status": "Draft", "rowVersion": out.RowVersion})
}

func (s *Server) approveBudget(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct{}
	idemKey, idemHash, release, handled := beginIdemOptional(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	next, err := s.store.ApproveBudget(r.Context(), scope, id, user, ver)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, map[string]any{"id": id, "status": "Approved", "rowVersion": next})
}

func (s *Server) listCosts(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	pid, err := httpx.ParsePathUUID(r, "projectId")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, err := s.store.ListCosts(r.Context(), scope, pid)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]costDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toCostDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *Server) createCost(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	pid, err := httpx.ParsePathUUID(r, "projectId")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		WBSElementID      string  `json:"wbsElementId"`
		CostType          string  `json:"costType"`
		Amount            string  `json:"amount"`
		SourceDocumentRef *string `json:"sourceDocumentRef"`
		PostingDate       string  `json:"postingDate"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	wbsID, err := uuid.Parse(req.WBSElementID)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("valid wbsElementId is required"))
		return
	}
	switch req.CostType {
	case "Labor", "Material", "Subcontractor", "Overhead":
	default:
		httpx.WriteError(w, r, httpx.BadRequest("costType must be Labor, Material, Subcontractor, or Overhead"))
		return
	}
	amt, err := parsePositiveNumeric(req.Amount, "amount")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	pDate, err := platform.ParseDate(req.PostingDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("postingDate must be YYYY-MM-DD"))
		return
	}
	out, err := s.store.CreateCost(r.Context(), scope, prjdb.CreateProjectCostParams{
		OrganizationID: scope[0], ProjectID: pid, WbsElementID: wbsID,
		CostType: req.CostType, Amount: amt, SourceDocumentRef: req.SourceDocumentRef,
		PostingDate: pDate, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "rowVersion": out.RowVersion})
}

func (s *Server) listProgress(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	wbsID, err := httpx.ParsePathUUID(r, "wbsId")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, err := s.store.ListProgress(r.Context(), scope, wbsID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]progressDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toProgressDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *Server) upsertProgress(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	wbsID, err := httpx.ParsePathUUID(r, "wbsId")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		PeriodDate          string `json:"periodDate"`
		PhysicalProgressPct string `json:"physicalProgressPct"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	pDate, err := platform.ParseDate(req.PeriodDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("periodDate must be YYYY-MM-DD"))
		return
	}
	pct, err := platform.ParseNumeric(strings.TrimSpace(req.PhysicalProgressPct))
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("invalid physicalProgressPct"))
		return
	}
	rat := platform.NumericToRat(pct)
	if rat == nil || rat.Sign() < 0 || rat.Cmp(big.NewRat(100, 1)) > 0 {
		httpx.WriteError(w, r, httpx.BadRequest("physicalProgressPct must be between 0 and 100"))
		return
	}
	out, err := s.store.UpsertProgress(r.Context(), scope, prjdb.UpsertProjectProgressParams{
		OrganizationID: scope[0], WbsElementID: wbsID, PeriodDate: pDate,
		PhysicalProgressPct: pct, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "rowVersion": out.RowVersion})
}

func (s *Server) listBillings(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var pid *uuid.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("projectId")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid projectId"))
			return
		}
		pid = &parsed
	}
	rows, total, err := s.store.ListBillings(r.Context(), scope, pid, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]billingDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toBillingDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: items, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createBilling(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		ProjectID    string  `json:"projectId"`
		BillingType  string  `json:"billingType"`
		Amount       string  `json:"amount"`
		ArInvoiceRef *string `json:"arInvoiceRef"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	pid, err := uuid.Parse(req.ProjectID)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("valid projectId is required"))
		return
	}
	switch req.BillingType {
	case "Milestone", "Progress", "TimeMaterial":
	default:
		httpx.WriteError(w, r, httpx.BadRequest("billingType must be Milestone, Progress, or TimeMaterial"))
		return
	}
	amt, err := parsePositiveNumeric(req.Amount, "amount")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out, err := s.store.CreateBilling(r.Context(), scope, prjdb.CreateProjectBillingParams{
		OrganizationID: scope[0], ProjectID: pid, BillingType: req.BillingType,
		Amount: amt, ArInvoiceRef: req.ArInvoiceRef, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "status": "Draft", "rowVersion": out.RowVersion})
}

func (s *Server) approveBilling(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParsePathUUID(r, "id")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		CustomerID   *string `json:"customerId"`
		ArInvoiceRef *string `json:"arInvoiceRef"`
	}
	idemKey, idemHash, release, handled := beginIdemOptional(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	var custID *uuid.UUID
	if req.CustomerID != nil && strings.TrimSpace(*req.CustomerID) != "" {
		c, e := uuid.Parse(*req.CustomerID)
		if e != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid customerId"))
			return
		}
		custID = &c
	}
	next, invRef, err := s.store.ApproveBillingAndEmit(r.Context(), scope, id, scope[0], user, httpx.CorrelationUUID(r), ver, custID, req.ArInvoiceRef)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, map[string]any{"id": id, "status": "Invoiced", "arInvoiceRef": invRef, "rowVersion": next})
}

func (s *Server) listChangeOrders(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	pid, err := httpx.ParsePathUUID(r, "projectId")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, err := s.store.ListChangeOrders(r.Context(), scope, pid)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]changeOrderDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toChangeOrderDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *Server) createChangeOrder(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	pid, err := httpx.ParsePathUUID(r, "projectId")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		Description        string `json:"description"`
		BudgetImpactAmount string `json:"budgetImpactAmount"`
		ScheduleImpactDays int32  `json:"scheduleImpactDays"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	if strings.TrimSpace(req.Description) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("description is required"))
		return
	}
	amt, err := platform.ParseOptionalNumeric(req.BudgetImpactAmount)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("invalid budgetImpactAmount"))
		return
	}
	out, err := s.store.CreateChangeOrder(r.Context(), scope, prjdb.CreateChangeOrderParams{
		OrganizationID: scope[0], ProjectID: pid, Description: strings.TrimSpace(req.Description),
		BudgetImpactAmount: amt, ScheduleImpactDays: req.ScheduleImpactDays, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "status": "Draft", "rowVersion": out.RowVersion})
}

func (s *Server) decideChangeOrder(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scope, user, ok := requireWrite(w, r)
		if !ok {
			return
		}
		id, err := httpx.ParsePathUUID(r, "id")
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		ver, err := httpx.IfMatch(r)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		var req struct{}
		idemKey, idemHash, release, handled := beginIdemOptional(w, r, s.idem, scope, &req)
		if handled {
			return
		}
		defer release()
		next, err := s.store.DecideChangeOrder(r.Context(), scope, id, user, ver, status)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, map[string]any{"id": id, "status": status, "rowVersion": next})
	}
}
