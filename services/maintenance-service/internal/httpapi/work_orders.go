package httpapi

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/maintenance-service/internal/mntdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func (s *Server) listWorkOrders(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var st *string
	if v := strings.TrimSpace(r.URL.Query().Get("status")); v != "" {
		st = &v
	}
	rows, total, err := s.store.ListWorkOrders(r.Context(), scope, st, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]workOrderDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toWorkOrderDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: items, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createWorkOrder(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		WoNo            string  `json:"woNo"`
		AssetRef        string  `json:"assetRef"`
		MaintenanceType string  `json:"maintenanceType"`
		AssigneeRef     *string `json:"assigneeRef"`
		VendorRef       *string `json:"vendorRef"`
		ScheduledDate   *string `json:"scheduledDate"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	assetRef, err := uuid.Parse(req.AssetRef)
	if err != nil || strings.TrimSpace(req.WoNo) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("woNo and valid assetRef are required"))
		return
	}
	if req.MaintenanceType != "Preventive" && req.MaintenanceType != "Corrective" {
		httpx.WriteError(w, r, httpx.BadRequest("maintenanceType must be Preventive or Corrective"))
		return
	}
	var assigneeRef, vendorRef *uuid.UUID
	if req.AssigneeRef != nil && strings.TrimSpace(*req.AssigneeRef) != "" {
		u, e := uuid.Parse(*req.AssigneeRef)
		if e != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid assigneeRef"))
			return
		}
		assigneeRef = &u
	}
	if req.VendorRef != nil && strings.TrimSpace(*req.VendorRef) != "" {
		u, e := uuid.Parse(*req.VendorRef)
		if e != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid vendorRef"))
			return
		}
		vendorRef = &u
	}
	sd, err := parseOptDate(req.ScheduledDate)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out, err := s.store.CreateWorkOrder(r.Context(), scope, mntdb.CreateWorkOrderParams{
		OrganizationID: scope[0], WoNo: strings.TrimSpace(req.WoNo), AssetRef: assetRef,
		MaintenanceType: req.MaintenanceType, AssigneeRef: assigneeRef, VendorRef: vendorRef,
		ScheduledDate: sd, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "status": "Open", "rowVersion": out.RowVersion})
}

func (s *Server) getWorkOrder(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	row, err := s.store.GetWorkOrder(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, row.RowVersion, toWorkOrderDTO(row))
}

func (s *Server) assignWorkOrder(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
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
		AssigneeRef *string `json:"assigneeRef"`
		VendorRef   *string `json:"vendorRef"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	var assigneeRef, vendorRef *uuid.UUID
	if req.AssigneeRef != nil && strings.TrimSpace(*req.AssigneeRef) != "" {
		u, e := uuid.Parse(*req.AssigneeRef)
		if e != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid assigneeRef"))
			return
		}
		assigneeRef = &u
	}
	if req.VendorRef != nil && strings.TrimSpace(*req.VendorRef) != "" {
		u, e := uuid.Parse(*req.VendorRef)
		if e != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid vendorRef"))
			return
		}
		vendorRef = &u
	}
	if assigneeRef == nil && vendorRef == nil {
		httpx.WriteError(w, r, httpx.BadRequest("assigneeRef or vendorRef is required"))
		return
	}
	next, err := s.store.AssignWorkOrder(r.Context(), scope, mntdb.AssignWorkOrderParams{
		ID: id, AssigneeRef: assigneeRef, VendorRef: vendorRef, UpdatedBy: &user, RowVersion: ver,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, map[string]any{"id": id, "status": "Assigned", "rowVersion": next})
}

func (s *Server) startWorkOrder(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
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
	next, err := s.store.StartWorkOrder(r.Context(), scope, id, user, ver)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, map[string]any{"id": id, "status": "InProgress", "rowVersion": next})
}

func (s *Server) completeWorkOrder(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
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
	next, totalCost, err := s.store.CompleteWorkOrderAndEmit(r.Context(), scope, id, scope[0], user, httpx.CorrelationUUID(r), ver)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, map[string]any{"id": id, "status": "Completed", "totalCost": totalCost, "rowVersion": next})
}

func (s *Server) closeWorkOrder(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
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
	next, err := s.store.CloseWorkOrder(r.Context(), scope, id, user, ver)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusOK, next, map[string]any{"id": id, "status": "Closed", "rowVersion": next})
}

func (s *Server) listParts(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	woID, err := httpx.ParseUUIDParam("woId", r.PathValue("woId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, err := s.store.ListParts(r.Context(), scope, woID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]partDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toPartDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *Server) addPart(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	woID, err := httpx.ParseUUIDParam("woId", r.PathValue("woId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		ItemRef     string  `json:"itemRef"`
		Quantity    string  `json:"quantity"`
		UnitCost    string  `json:"unitCost"`
		WarehouseID *string `json:"warehouseId"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	itemRef, err := uuid.Parse(req.ItemRef)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("valid itemRef is required"))
		return
	}
	qty, err := parsePositiveDecimal(req.Quantity, "quantity")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	uc, err := platform.ParseOptionalNumeric(req.UnitCost)
	if err != nil || (platform.NumericToRat(uc) != nil && platform.NumericToRat(uc).Sign() < 0) {
		httpx.WriteError(w, r, httpx.BadRequest("unitCost must be a non-negative decimal string"))
		return
	}
	var whID *uuid.UUID
	if req.WarehouseID != nil && strings.TrimSpace(*req.WarehouseID) != "" {
		u, e := uuid.Parse(*req.WarehouseID)
		if e != nil {
			httpx.WriteError(w, r, httpx.BadRequest("invalid warehouseId"))
			return
		}
		whID = &u
	}
	out, err := s.store.AddPartAndEmit(r.Context(), scope, mntdb.CreateWorkOrderPartParams{
		OrganizationID: scope[0], WorkOrderID: woID, ItemRef: itemRef,
		Quantity: qty, UnitCost: uc, CreatedBy: user,
	}, whID, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "rowVersion": out.RowVersion})
}

func (s *Server) listChecklist(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	woID, err := httpx.ParseUUIDParam("woId", r.PathValue("woId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, err := s.store.ListChecklist(r.Context(), scope, woID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]checklistDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toChecklistDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

func (s *Server) addChecklist(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	woID, err := httpx.ParseUUIDParam("woId", r.PathValue("woId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		StepNo      int32  `json:"stepNo"`
		Description string `json:"description"`
		IsDone      bool   `json:"isDone"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	if req.StepNo <= 0 || strings.TrimSpace(req.Description) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("stepNo > 0 and description are required"))
		return
	}
	out, err := s.store.AddChecklist(r.Context(), scope, mntdb.CreateWorkOrderChecklistParams{
		OrganizationID: scope[0], WorkOrderID: woID, StepNo: req.StepNo,
		Description: strings.TrimSpace(req.Description), IsDone: req.IsDone, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "rowVersion": out.RowVersion})
}

func (s *Server) updateChecklist(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
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
		Description string `json:"description"`
		IsDone      bool   `json:"isDone"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Description) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("description is required"))
		return
	}
	next, err := s.store.UpdateChecklist(r.Context(), scope, mntdb.UpdateWorkOrderChecklistParams{
		ID: id, Description: strings.TrimSpace(req.Description), IsDone: req.IsDone, UpdatedBy: &user, RowVersion: ver,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "rowVersion": next})
}

func (s *Server) listCosts(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	woID, err := httpx.ParseUUIDParam("woId", r.PathValue("woId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rows, err := s.store.ListCosts(r.Context(), scope, woID)
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

func (s *Server) addCost(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	woID, err := httpx.ParseUUIDParam("woId", r.PathValue("woId"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		CostType string `json:"costType"`
		Amount   string `json:"amount"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	switch req.CostType {
	case "Labor", "Part", "Vendor":
	default:
		httpx.WriteError(w, r, httpx.BadRequest("costType must be Labor, Part, or Vendor"))
		return
	}
	amt, err := parsePositiveDecimal(req.Amount, "amount")
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out, err := s.store.AddCost(r.Context(), scope, mntdb.CreateMaintenanceCostParams{
		OrganizationID: scope[0], WorkOrderID: woID, CostType: req.CostType,
		Amount: amt, CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "rowVersion": out.RowVersion})
}
