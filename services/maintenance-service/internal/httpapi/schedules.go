package httpapi

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/maintenance-service/internal/mntdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

func (s *Server) listSchedules(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.store.ListSchedules(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	items := make([]scheduleDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, toScheduleDTO(row))
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: items, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) createSchedule(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req struct {
		AssetRef      string  `json:"assetRef"`
		ScheduleType  string  `json:"scheduleType"`
		IntervalValue int32   `json:"intervalValue"`
		IntervalUnit  string  `json:"intervalUnit"`
		NextDueDate   *string `json:"nextDueDate"`
		IsActive      *bool   `json:"isActive"`
	}
	idemKey, idemHash, release, handled := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if handled {
		return
	}
	defer release()
	assetRef, err := uuid.Parse(req.AssetRef)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("valid assetRef is required"))
		return
	}
	if req.ScheduleType != "TimeBased" && req.ScheduleType != "UsageBased" {
		httpx.WriteError(w, r, httpx.BadRequest("scheduleType must be TimeBased or UsageBased"))
		return
	}
	switch req.IntervalUnit {
	case "Day", "Month", "Hour", "Cycle":
	default:
		httpx.WriteError(w, r, httpx.BadRequest("intervalUnit must be Day, Month, Hour, or Cycle"))
		return
	}
	if req.IntervalValue <= 0 {
		httpx.WriteError(w, r, httpx.BadRequest("intervalValue must be > 0"))
		return
	}
	nd, err := parseOptDate(req.NextDueDate)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	out, err := s.store.CreateSchedule(r.Context(), scope, mntdb.CreateMaintenanceScheduleParams{
		OrganizationID: scope[0], AssetRef: assetRef, ScheduleType: req.ScheduleType,
		IntervalValue: req.IntervalValue, IntervalUnit: req.IntervalUnit, NextDueDate: nd, IsActive: active,
		CreatedBy: user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, idemHash, http.StatusCreated, out.RowVersion, map[string]any{"id": out.ID, "rowVersion": out.RowVersion})
}

func (s *Server) updateSchedule(w http.ResponseWriter, r *http.Request) {
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
		ScheduleType  string  `json:"scheduleType"`
		IntervalValue int32   `json:"intervalValue"`
		IntervalUnit  string  `json:"intervalUnit"`
		NextDueDate   *string `json:"nextDueDate"`
		IsActive      bool    `json:"isActive"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if (req.ScheduleType != "TimeBased" && req.ScheduleType != "UsageBased") || req.IntervalValue <= 0 {
		httpx.WriteError(w, r, httpx.BadRequest("valid scheduleType and positive intervalValue required"))
		return
	}
	nd, err := parseOptDate(req.NextDueDate)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	next, err := s.store.UpdateSchedule(r.Context(), scope, mntdb.UpdateMaintenanceScheduleParams{
		ID: id, ScheduleType: req.ScheduleType, IntervalValue: req.IntervalValue, IntervalUnit: req.IntervalUnit,
		NextDueDate: nd, IsActive: req.IsActive, UpdatedBy: &user, RowVersion: ver,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, next, map[string]any{"id": id, "rowVersion": next})
}

func (s *Server) deleteSchedule(w http.ResponseWriter, r *http.Request) {
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
	if err := s.store.DeleteSchedule(r.Context(), scope, id, user, ver); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

