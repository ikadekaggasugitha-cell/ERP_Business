package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/procurement-service/internal/procurementdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func (s *Server) createPR(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "procurement:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
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
	var req prCreateReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	if strings.TrimSpace(req.PrNo) == "" || len(req.Lines) == 0 {
		httpx.WriteError(w, r, httpx.BadRequest("prNo dan minimal satu line wajib diisi"))
		return
	}
	reqID := user
	if strings.TrimSpace(req.RequesterID) != "" {
		if reqID, err = httpx.ParseUUIDParam("requesterId", req.RequesterID); err != nil {
			httpx.WriteError(w, r, err)
			return
		}
	}
	var neededDate pgtype.Date
	if strings.TrimSpace(req.NeededDate) != "" {
		if neededDate, err = platform.ParseDate(req.NeededDate); err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("neededDate harus YYYY-MM-DD"))
			return
		}
	}
	var wfRef *string
	if w := strings.TrimSpace(req.WorkflowRef); w != "" {
		wfRef = &w
	}
	lines := make([]procurementdb.CreatePurchaseRequisitionLineParams, 0, len(req.Lines))
	for _, l := range req.Lines {
		if l.LineNo <= 0 {
			httpx.WriteError(w, r, httpx.BadRequest("lineNo harus > 0"))
			return
		}
		itemID, err := httpx.ParseUUIDParam("lines[].itemId", l.ItemID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		qtyNum, err := platform.ParseNumeric(l.RequestQty)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("lines[].requestQty tidak valid"))
			return
		}
		if qtyRat := platform.NumericToRat(qtyNum); qtyRat == nil || qtyRat.Sign() <= 0 {
			httpx.WriteError(w, r, httpx.BadRequest("lines[].requestQty harus > 0"))
			return
		}
		var lDate pgtype.Date
		if strings.TrimSpace(l.NeededDate) != "" {
			if lDate, err = platform.ParseDate(l.NeededDate); err != nil {
				httpx.WriteError(w, r, httpx.BadRequest("lines[].neededDate harus YYYY-MM-DD"))
				return
			}
		}
		lines = append(lines, procurementdb.CreatePurchaseRequisitionLineParams{
			OrganizationID: scope[0],
			LineNo:         l.LineNo,
			ItemID:         itemID,
			RequestQty:     qtyNum,
			NeededDate:     lDate,
			CreatedBy:      user,
		})
	}
	created, err := s.repo.CreateRequisitionWithLines(r.Context(), scope, procurementdb.CreatePurchaseRequisitionParams{
		OrganizationID: scope[0],
		PrNo:           strings.TrimSpace(req.PrNo),
		RequesterID:    reqID,
		WorkflowRef:    wfRef,
		NeededDate:     neededDate,
		CreatedBy:      user,
	}, lines)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "nomor PR sudah dipakai"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, prDTO{
		ID:          created.ID.String(),
		PrNo:        strings.TrimSpace(req.PrNo),
		RequesterID: reqID.String(),
		NeededDate:  platform.FormatDate(neededDate),
		Status:      created.Status,
		RowVersion:  created.RowVersion,
	})
}

func (s *Server) listPRs(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "procurement:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var statusFilter *string
	if st := strings.TrimSpace(r.URL.Query().Get("status")); st != "" {
		statusFilter = &st
	}
	rows, total, err := s.repo.ListRequisitionsWithCount(r.Context(), scope, procurementdb.ListPurchaseRequisitionsParams{
		Limit:  limit,
		Offset: offset,
		Status: statusFilter,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]prDTO, 0, len(rows))
	for _, row := range rows {
		dto := prDTO{
			ID:          row.ID.String(),
			PrNo:        row.PrNo,
			RequesterID: row.RequesterID.String(),
			NeededDate:  platform.FormatDate(row.NeededDate),
			Status:      row.Status,
			RowVersion:  row.RowVersion,
		}
		if row.WorkflowRef != nil {
			dto.WorkflowRef = *row.WorkflowRef
		}
		out = append(out, dto)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) getPR(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "procurement:read"); err != nil {
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
	detail, err := s.repo.GetRequisition(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("purchase requisition tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	lines := make([]prLineDTO, 0, len(detail.Lines))
	for _, l := range detail.Lines {
		lines = append(lines, prLineDTO{
			ID:         l.ID.String(),
			LineNo:     l.LineNo,
			ItemID:     l.ItemID.String(),
			RequestQty: platform.FormatNumeric(l.RequestQty, 4),
			NeededDate: platform.FormatDate(l.NeededDate),
		})
	}
	dto := prDTO{
		ID:          detail.Header.ID.String(),
		PrNo:        detail.Header.PrNo,
		RequesterID: detail.Header.RequesterID.String(),
		NeededDate:  platform.FormatDate(detail.Header.NeededDate),
		Status:      detail.Header.Status,
		RowVersion:  detail.Header.RowVersion,
		Lines:       lines,
	}
	if detail.Header.WorkflowRef != nil {
		dto.WorkflowRef = *detail.Header.WorkflowRef
	}
	httpx.WriteIDResponse(w, http.StatusOK, detail.Header.RowVersion, dto)
}

func (s *Server) submitPR(w http.ResponseWriter, r *http.Request) {
	s.transitionPR(w, r, "Draft", "Submitted")
}

func (s *Server) approvePR(w http.ResponseWriter, r *http.Request) {
	s.transitionPR(w, r, "Submitted", "Approved")
}

func (s *Server) transitionPR(w http.ResponseWriter, r *http.Request, fromStatus, toStatus string) {
	if toStatus == "Approved" {
		if !httpx.HasPermission(r.Context(), "procurement:approve") && !httpx.HasPermission(r.Context(), "procurement.approve") {
			httpx.WriteError(w, r, httpx.Forbidden("izin procurement:approve dibutuhkan"))
			return
		}
	} else if err := httpx.RequirePermission(r, "procurement:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
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
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	rowVer, err := httpx.IfMatch(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if toStatus == "Approved" {
		prDetail, err := s.repo.GetRequisition(r.Context(), scope, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				httpx.WriteError(w, r, httpx.NotFound("purchase requisition tidak ditemukan"))
				return
			}
			httpx.WriteError(w, r, httpx.MapDBError(err, ""))
			return
		}
		if (prDetail.Header.CreatedBy != uuid.Nil && user == prDetail.Header.CreatedBy || user == prDetail.Header.RequesterID) && !httpx.HasRole(r.Context(), "SYSTEM_ADMIN") {
			httpx.WriteError(w, r, &httpx.APIError{Status: http.StatusForbidden, Code: "SOD_VIOLATION", Message: "pemohon purchase requisition tidak boleh menyetujui PR sendiri"})
			return
		}
	}
	row, n, err := s.repo.TransitionRequisition(r.Context(), scope, procurementdb.TransitionPurchaseRequisitionParams{
		ID:         id,
		RowVersion: rowVer,
		Status:     toStatus,
		UpdatedBy:  &user,
		Status_2:   fromStatus,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau status PR tidak sesuai"))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, row.RowVersion, prDTO{
		ID:         id.String(),
		Status:     row.Status,
		RowVersion: row.RowVersion,
	})
}

