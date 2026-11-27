package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/sinergiinvestasiproperti/erp/services/manufacturing-service/internal/manufacturingdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

func (s *Server) createBOM(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "manufacturing:write"); err != nil {
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
	var req bomReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	if strings.TrimSpace(req.Code) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code BOM wajib diisi"))
		return
	}
	prodID, err := httpx.ParseUUIDParam("productItemId", req.ProductItemID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	ver := req.Version
	if ver <= 0 {
		ver = 1
	}
	lines := make([]manufacturingdb.CreateBomLineParams, 0, len(req.Lines))
	seenLines := make(map[int32]struct{}, len(req.Lines))
	for _, l := range req.Lines {
		if l.LineNo <= 0 || strings.TrimSpace(l.Uom) == "" {
			httpx.WriteError(w, r, httpx.BadRequest("lines[].lineNo harus > 0 dan uom wajib diisi"))
			return
		}
		if _, dup := seenLines[l.LineNo]; dup {
			httpx.WriteError(w, r, httpx.BadRequest("lines[].lineNo tidak boleh duplikat"))
			return
		}
		seenLines[l.LineNo] = struct{}{}
		compID, err := httpx.ParseUUIDParam("lines[].componentItemId", l.ComponentItemID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		qtyNum, err := platform.ParseNumeric(l.Quantity)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("lines[].quantity tidak valid"))
			return
		}
		if qr := platform.NumericToRat(qtyNum); qr == nil || qr.Sign() <= 0 {
			httpx.WriteError(w, r, httpx.BadRequest("lines[].quantity harus > 0"))
			return
		}
		lines = append(lines, manufacturingdb.CreateBomLineParams{
			OrganizationID:  scope[0],
			LineNo:          l.LineNo,
			ComponentItemID: compID,
			Quantity:        qtyNum,
			Uom:             strings.TrimSpace(l.Uom),
			CreatedBy:       user,
		})
	}
	created, err := s.repo.CreateBOMWithLines(r.Context(), scope, manufacturingdb.CreateBillOfMaterialParams{
		OrganizationID: scope[0],
		Code:           strings.TrimSpace(req.Code),
		ProductItemID:  prodID,
		Version:        ver,
		CreatedBy:      user,
	}, lines)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "kode dan versi BOM sudah dipakai"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, bomDTO{
		ID:            created.ID.String(),
		Code:          created.Code,
		ProductItemID: created.ProductItemID.String(),
		Version:       created.Version,
		IsActive:      created.IsActive,
		RowVersion:    created.RowVersion,
	})
}

func (s *Server) listBOMs(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "manufacturing:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListBOMsWithCount(r.Context(), scope, manufacturingdb.ListBillsOfMaterialParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]bomDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, bomDTO{
			ID:            row.ID.String(),
			Code:          row.Code,
			ProductItemID: row.ProductItemID.String(),
			Version:       row.Version,
			IsActive:      row.IsActive,
			RowVersion:    row.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) getBOM(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "manufacturing:read"); err != nil {
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
	detail, err := s.repo.GetBOM(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("BOM tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	lines := make([]bomLineDTO, 0, len(detail.Lines))
	for _, l := range detail.Lines {
		lines = append(lines, bomLineDTO{
			ID:              l.ID.String(),
			LineNo:          l.LineNo,
			ComponentItemID: l.ComponentItemID.String(),
			Quantity:        platform.FormatNumeric(l.Quantity, 4),
			Uom:             l.Uom,
		})
	}
	httpx.WriteIDResponse(w, http.StatusOK, detail.Header.RowVersion, bomDTO{
		ID:            detail.Header.ID.String(),
		Code:          detail.Header.Code,
		ProductItemID: detail.Header.ProductItemID.String(),
		Version:       detail.Header.Version,
		IsActive:      detail.Header.IsActive,
		RowVersion:    detail.Header.RowVersion,
		Lines:         lines,
	})
}

