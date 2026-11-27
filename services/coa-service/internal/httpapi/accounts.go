package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/coa-service/internal/coadb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
)

func (s *Server) createChart(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req chartReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code dan name wajib diisi"))
		return
	}
	st := req.Structure
	if len(st) == 0 {
		st = []byte(`{}`)
	} else if !json.Valid(st) {
		httpx.WriteError(w, r, httpx.BadRequest("structure harus JSON valid"))
		return
	}
	created, err := s.repo.CreateChartOfAccounts(r.Context(), scope, coadb.CreateChartOfAccountsParams{
		OrganizationID: scope[0],
		Code:           strings.TrimSpace(req.Code),
		Name:           strings.TrimSpace(req.Name),
		Structure:      st,
		CreatedBy:      user,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "kode chart of accounts sudah dipakai"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, map[string]any{
		"id":         created.ID.String(),
		"code":       strings.TrimSpace(req.Code),
		"name":       strings.TrimSpace(req.Name),
		"rowVersion": created.RowVersion,
	})
}

func (s *Server) listCharts(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListChartsWithCount(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"id":         row.ID.String(),
			"code":       row.Code,
			"name":       row.Name,
			"structure":  json.RawMessage(row.Structure),
			"rowVersion": row.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{Data: out, Page: httpx.NewPageMeta(int(page), int(size), int(total))})
}

func (s *Server) getChart(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	c, err := s.repo.GetChartOfAccounts(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, c.RowVersion, map[string]any{
		"id":         c.ID.String(),
		"code":       c.Code,
		"name":       c.Name,
		"structure":  json.RawMessage(c.Structure),
		"rowVersion": c.RowVersion,
	})
}

func (s *Server) updateChart(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
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
	var req chartReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("name wajib diisi"))
		return
	}
	st := req.Structure
	if len(st) == 0 {
		existing, err := s.repo.GetChartOfAccounts(r.Context(), scope, id)
		if err != nil {
			httpx.WriteError(w, r, httpx.MapDBError(err, ""))
			return
		}
		st = existing.Structure
		if len(st) == 0 {
			st = []byte(`{}`)
		}
	} else if !json.Valid(st) {
		httpx.WriteError(w, r, httpx.BadRequest("structure harus JSON valid"))
		return
	}
	newVer, err := s.repo.UpdateChartOfAccounts(r.Context(), scope, coadb.UpdateChartOfAccountsParams{
		ID:         id,
		RowVersion: rowVer,
		Name:       strings.TrimSpace(req.Name),
		Structure:  st,
		UpdatedBy:  &user,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, newVer, map[string]any{"id": id.String(), "name": strings.TrimSpace(req.Name), "rowVersion": newVer})
}

func (s *Server) deleteChart(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
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
	n, err := s.repo.DeleteChartOfAccounts(r.Context(), scope, coadb.DeleteChartOfAccountsParams{ID: id, RowVersion: rowVer, UpdatedBy: &user})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau data tidak ditemukan"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) createAccount(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
		return
	}
	var req accountReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	coaID, err := httpx.ParseUUIDParam("chartOfAccountsId", req.ChartOfAccountsID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code dan name wajib diisi"))
		return
	}
	if !isValidAccountType(req.AccountType) {
		httpx.WriteError(w, r, httpx.BadRequest("accountType harus Asset, Liability, Equity, Revenue, atau Expense"))
		return
	}
	var accClass *string
	if s := strings.TrimSpace(req.AccountClass); s != "" {
		accClass = &s
	}
	var parentID *uuid.UUID
	if strings.TrimSpace(req.ParentID) != "" {
		p, err := httpx.ParseUUIDParam("parentId", req.ParentID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		parentID = &p
	}
	allowPosting := true
	if req.AllowPosting != nil {
		allowPosting = *req.AllowPosting
	}

	acct := coadb.CreateAccountParams{
		OrganizationID:    scope[0],
		ChartOfAccountsID: coaID,
		Code:              strings.TrimSpace(req.Code),
		Name:              strings.TrimSpace(req.Name),
		AccountType:       req.AccountType,
		AccountClass:      accClass,
		IsControl:         req.IsControl,
		IsMonetary:        req.IsMonetary,
		AllowPosting:      allowPosting,
		ParentID:          parentID,
		CreatedBy:         user,
	}
	payload, _ := json.Marshal(map[string]any{
		"code": acct.Code, "name": acct.Name, "accountType": acct.AccountType,
		"isMonetary": acct.IsMonetary, "allowPosting": acct.AllowPosting,
	})
	ev := coadb.InsertOutboxParams{
		AggregateType:  "Account",
		EventType:      "coa.account.upserted",
		Payload:        payload,
		OrganizationID: scope[0],
		CorrelationID:  httpx.CorrelationUUID(r),
	}
	created, err := s.repo.CreateAndEmit(r.Context(), scope, acct, ev)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "kode akun sudah dipakai"))
		return
	}
	dto := accountDTO{
		ID:                created.ID.String(),
		ChartOfAccountsID: coaID.String(),
		Code:              acct.Code,
		Name:              acct.Name,
		AccountType:       acct.AccountType,
		AccountClass:      accClass,
		IsControl:         acct.IsControl,
		IsMonetary:        acct.IsMonetary,
		AllowPosting:      acct.AllowPosting,
		RowVersion:        created.RowVersion,
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, dto)
}

func (s *Server) getAccount(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	a, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, a.RowVersion, accountDTO{
		ID:                a.ID.String(),
		ChartOfAccountsID: a.ChartOfAccountsID.String(),
		Code:              a.Code,
		Name:              a.Name,
		AccountType:       a.AccountType,
		AccountClass:      a.AccountClass,
		IsControl:         a.IsControl,
		IsMonetary:        a.IsMonetary,
		AllowPosting:      a.AllowPosting,
		RowVersion:        a.RowVersion,
	})
}

func (s *Server) listAccounts(w http.ResponseWriter, r *http.Request) {
	scope, ok := requireRead(w, r)
	if !ok {
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	acctType := strings.TrimSpace(r.URL.Query().Get("accountType"))
	rows, total, err := s.repo.ListWithCount(r.Context(), scope, acctType, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]accountDTO, 0, len(rows))
	for _, a := range rows {
		out = append(out, accountDTO{
			ID:                a.ID.String(),
			ChartOfAccountsID: a.ChartOfAccountsID.String(),
			Code:              a.Code,
			Name:              a.Name,
			AccountType:       a.AccountType,
			AccountClass:      a.AccountClass,
			IsControl:         a.IsControl,
			IsMonetary:        a.IsMonetary,
			AllowPosting:      a.AllowPosting,
			RowVersion:        a.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) updateAccount(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
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
	var req accountReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("name wajib diisi"))
		return
	}
	if !isValidAccountType(req.AccountType) {
		httpx.WriteError(w, r, httpx.BadRequest("accountType tidak valid"))
		return
	}
	var accClass *string
	if s := strings.TrimSpace(req.AccountClass); s != "" {
		accClass = &s
	}
	var parentID *uuid.UUID
	if strings.TrimSpace(req.ParentID) != "" {
		p, err := httpx.ParseUUIDParam("parentId", req.ParentID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		if p == id {
			httpx.WriteError(w, r, httpx.BadRequest("parentId tidak boleh mereferensikan diri sendiri"))
			return
		}
		parentID = &p
	}
	existing, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	allowPosting := true
	if req.AllowPosting != nil {
		allowPosting = *req.AllowPosting
	}
	payload, _ := json.Marshal(map[string]any{
		"code": existing.Code, "name": strings.TrimSpace(req.Name), "accountType": req.AccountType,
		"isMonetary": req.IsMonetary, "allowPosting": allowPosting,
	})
	newVer, err := s.repo.UpdateAccountAndEmit(r.Context(), scope, coadb.UpdateAccountParams{
		ID:           id,
		RowVersion:   rowVer,
		Name:         strings.TrimSpace(req.Name),
		AccountType:  req.AccountType,
		AccountClass: accClass,
		IsControl:    req.IsControl,
		IsMonetary:   req.IsMonetary,
		AllowPosting: allowPosting,
		ParentID:     parentID,
		UpdatedBy:    &user,
	}, coadb.InsertOutboxParams{
		AggregateType:  "Account",
		AggregateID:    id,
		EventType:      "coa.account.upserted",
		Payload:        payload,
		OrganizationID: scope[0],
		CorrelationID:  httpx.CorrelationUUID(r),
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, newVer, accountDTO{
		ID:                id.String(),
		ChartOfAccountsID: existing.ChartOfAccountsID.String(),
		Code:              existing.Code,
		Name:              strings.TrimSpace(req.Name),
		AccountType:       req.AccountType,
		AccountClass:      accClass,
		IsControl:         req.IsControl,
		IsMonetary:        req.IsMonetary,
		AllowPosting:      allowPosting,
		RowVersion:        newVer,
	})
}

func (s *Server) deleteAccount(w http.ResponseWriter, r *http.Request) {
	scope, user, ok := requireWrite(w, r)
	if !ok {
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
	n, err := s.repo.DeleteAccount(r.Context(), scope, coadb.DeleteAccountParams{ID: id, RowVersion: rowVer, UpdatedBy: &user})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("STALE_ROW_VERSION", "rowVersion basi atau account tidak ditemukan"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) accountHierarchy(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "coa:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	acctType := strings.TrimSpace(r.URL.Query().Get("accountType"))
	rows, _, err := s.repo.ListWithCount(r.Context(), scope, acctType, 10000, 0)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}

	nodes := make(map[uuid.UUID]*accountTreeNode, len(rows))
	order := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		parentStr := ""
		if row.ParentID != nil {
			parentStr = row.ParentID.String()
		}
		nodes[row.ID] = &accountTreeNode{
			ID:              row.ID.String(),
			Code:            row.Code,
			Name:            row.Name,
			AccountType:     row.AccountType,
			ParentAccountID: parentStr,
			AllowPosting:    row.AllowPosting,
			RowVersion:      row.RowVersion,
			Children:        make([]*accountTreeNode, 0),
		}
		order = append(order, row.ID)
	}

	roots := make([]*accountTreeNode, 0)
	for _, id := range order {
		n := nodes[id]
		if n.ParentAccountID != "" {
			if pid, perr := uuid.Parse(n.ParentAccountID); perr == nil {
				if parent, ok := nodes[pid]; ok && pid != id {
					parent.Children = append(parent.Children, n)
					continue
				}
			}
		}
		roots = append(roots, n)
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"data":       roots,
		"totalNodes": len(rows),
	})
}
