package httpapi

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func (h *Handler) handleListLeaveRequests(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "hr:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	empFilter := strings.TrimSpace(r.URL.Query().Get("employeeId"))
	statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))

	h.mu.RLock()
	out := make([]leaveRequestRecord, 0, len(h.leaves))
	for _, lv := range h.leaves {
		if !slices.Contains(scope, lv.OrganizationID) {
			continue
		}
		if empFilter != "" && lv.EmployeeID.String() != empFilter {
			continue
		}
		if statusFilter != "" && !strings.EqualFold(lv.Status, statusFilter) {
			continue
		}
		out = append(out, lv)
	}
	h.mu.RUnlock()

	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(1, max(1, len(out)), len(out)),
	})
}

func (h *Handler) handleCreateLeaveRequest(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	user, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "hr:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		EmployeeID string `json:"employeeId"`
		LeaveType  string `json:"leaveType"`
		StartDate  string `json:"startDate"`
		EndDate    string `json:"endDate"`
		Reason     string `json:"reason"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, h.idem, scope, &req)
	if done {
		return
	}
	defer release()

	empID, err := httpx.ParseUUIDParam("employeeId", req.EmployeeID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	leaveType := strings.TrimSpace(req.LeaveType)
	switch leaveType {
	case "Annual", "Sick", "Maternity", "Unpaid", "Personal":
	default:
		httpx.WriteError(w, r, httpx.BadRequest("leaveType harus Annual, Sick, Maternity, Unpaid, atau Personal"))
		return
	}
	startDate, err := platform.ParseDate(req.StartDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("startDate harus YYYY-MM-DD"))
		return
	}
	endDate, err := platform.ParseDate(req.EndDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("endDate harus YYYY-MM-DD"))
		return
	}
	if endDate.Time.Before(startDate.Time) {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_DATE_RANGE", "endDate tidak boleh lebih awal dari startDate"))
		return
	}
	days := int(endDate.Time.Sub(startDate.Time).Hours()/24) + 1

	rec := leaveRequestRecord{
		ID:             uuid.New(),
		OrganizationID: scope[0],
		EmployeeID:     empID,
		LeaveType:      leaveType,
		StartDate:      platform.FormatDate(startDate),
		EndDate:        platform.FormatDate(endDate),
		DaysCount:      days,
		Reason:         strings.TrimSpace(req.Reason),
		Status:         "Pending",
		CreatedBy:      user,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	h.mu.Lock()
	h.leaves = append(h.leaves, rec)
	h.mu.Unlock()

	httpx.SaveIDResponse(w, h.idem, scope, idemKey, reqHash, http.StatusCreated, 1, rec)
}

func (h *Handler) handleApproveLeaveRequest(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	user, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "hr:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	for i := range h.leaves {
		if h.leaves[i].ID == id && slices.Contains(scope, h.leaves[i].OrganizationID) {
			if h.leaves[i].Status != "Pending" {
				httpx.WriteError(w, r, httpx.Conflict("INVALID_STATE", "leave request tidak berstatus Pending"))
				return
			}
			if h.leaves[i].CreatedBy == user && !httpx.HasRole(r.Context(), "SYSTEM_ADMIN") {
				httpx.WriteError(w, r, httpx.Forbidden("pembuat pengajuan cuti tidak dapat menyetujui pengajuannya sendiri (SoD)"))
				return
			}
			h.leaves[i].Status = "Approved"
			h.leaves[i].ApprovedBy = &user
			httpx.WriteJSON(w, http.StatusOK, h.leaves[i])
			return
		}
	}
	httpx.WriteError(w, r, httpx.NotFound("leave request tidak ditemukan"))
}

func (h *Handler) handleListAttendanceLogs(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "hr:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	empFilter := strings.TrimSpace(r.URL.Query().Get("employeeId"))
	dateFilter := strings.TrimSpace(r.URL.Query().Get("date"))

	h.mu.RLock()
	out := make([]attendanceLogRecord, 0, len(h.attendances))
	for _, a := range h.attendances {
		if !slices.Contains(scope, a.OrganizationID) {
			continue
		}
		if empFilter != "" && a.EmployeeID.String() != empFilter {
			continue
		}
		if dateFilter != "" && a.LogDate != dateFilter {
			continue
		}
		out = append(out, a)
	}
	h.mu.RUnlock()

	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(1, max(1, len(out)), len(out)),
	})
}

func (h *Handler) handleCreateAttendanceLog(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	user, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "hr:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		EmployeeID string `json:"employeeId"`
		LogDate    string `json:"logDate"`
		Status     string `json:"status"`
		ClockIn    string `json:"clockIn,omitempty"`
		ClockOut   string `json:"clockOut,omitempty"`
		Notes      string `json:"notes,omitempty"`
	}
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, h.idem, scope, &req)
	if done {
		return
	}
	defer release()

	empID, err := httpx.ParseUUIDParam("employeeId", req.EmployeeID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	logDate, err := platform.ParseDate(req.LogDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("logDate harus YYYY-MM-DD"))
		return
	}
	status := strings.TrimSpace(req.Status)
	switch status {
	case "Present", "Late", "Absent", "Leave", "Remote":
	default:
		httpx.WriteError(w, r, httpx.BadRequest("status harus Present, Late, Absent, Leave, atau Remote"))
		return
	}

	rec := attendanceLogRecord{
		ID:             uuid.New(),
		OrganizationID: scope[0],
		EmployeeID:     empID,
		LogDate:        platform.FormatDate(logDate),
		Status:         status,
		ClockIn:        strings.TrimSpace(req.ClockIn),
		ClockOut:       strings.TrimSpace(req.ClockOut),
		Notes:          strings.TrimSpace(req.Notes),
		CreatedBy:      user,
	}
	h.mu.Lock()
	h.attendances = append(h.attendances, rec)
	h.mu.Unlock()

	httpx.SaveIDResponse(w, h.idem, scope, idemKey, reqHash, http.StatusCreated, 1, rec)
}
