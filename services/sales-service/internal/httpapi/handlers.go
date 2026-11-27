package httpapi

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/sales-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/sales-service/internal/salesdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// Store abstracts persistence for HTTP handlers to enable unit testing.
type Store interface {
	CreateWithLines(ctx context.Context, scope []uuid.UUID, in repository.CreateSOInput) (repository.SalesOrderDetail, error)
	GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (repository.SalesOrderDetail, error)
	ListWithCount(ctx context.Context, scope []uuid.UUID, status *string, limit, offset int32) ([]salesdb.ListSalesOrdersRow, int64, error)
	UpdateDraftWithLines(ctx context.Context, scope []uuid.UUID, in repository.UpdateDraftSOInput) (repository.SalesOrderDetail, error)
	DeleteDraft(ctx context.Context, scope []uuid.UUID, id uuid.UUID, version int32, updatedBy uuid.UUID) error
	ConfirmAndEmit(ctx context.Context, scope []uuid.UUID, id uuid.UUID, version int32, warehouseID *uuid.UUID, updatedBy, corrID uuid.UUID) (repository.SalesOrderDetail, error)
	CreateDelivery(ctx context.Context, scope []uuid.UUID, in repository.CreateDeliveryInput) (repository.DeliveryDetail, error)
	GetDelivery(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (repository.DeliveryDetail, error)
	ListDeliveriesWithCount(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]salesdb.ListDeliveriesRow, int64, error)
}

type Handler struct {
	store      Store
	idem       *httpx.IdemStore
	qMu        sync.RWMutex
	quotations map[uuid.UUID]salesQuotationDTO
}

func NewRouter(store Store, metrics *httpx.Metrics, checks ...httpx.ReadinessCheck) http.Handler {
	h := &Handler{
		store:      store,
		idem:       httpx.NewIdemStore(),
		quotations: make(map[uuid.UUID]salesQuotationDTO),
	}
	if metrics == nil {
		metrics = httpx.NewMetrics("sales-service")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "sales-service"})
	})
	mux.HandleFunc("GET /readyz", httpx.ReadinessHandler("sales-service", checks...))
	mux.HandleFunc("GET /metrics", metrics.Handler())

	mux.HandleFunc("GET /sales-quotations", h.handleListQuotations)
	mux.HandleFunc("POST /sales-quotations", h.handleCreateQuotation)
	mux.HandleFunc("POST /sales-quotations/{id}/convert-to-order", h.handleConvertQuotationToOrder)

	mux.HandleFunc("POST /sales-orders", h.handleCreateSO)
	mux.HandleFunc("GET /sales-orders", h.handleListSO)
	mux.HandleFunc("GET /sales-orders/{id}", h.handleGetSO)
	mux.HandleFunc("PUT /sales-orders/{id}", h.handleUpdateDraftSO)
	mux.HandleFunc("DELETE /sales-orders/{id}", h.handleDeleteDraftSO)
	mux.HandleFunc("POST /sales-orders/{id}/confirm", h.handleConfirmSO)
	mux.HandleFunc("POST /sales-orders/{id}/return", h.handleReturnSO)
	mux.HandleFunc("POST /sales-orders/{id}/deliveries", h.handleCreateDelivery)
	mux.HandleFunc("GET /deliveries", h.handleListDeliveries)
	mux.HandleFunc("GET /deliveries/{id}", h.handleGetDelivery)
	return metrics.Wrap(httpx.Middleware(mux))
}

type soLineReq struct {
	LineNo    int32  `json:"lineNo"`
	ItemID    string `json:"itemId"`
	OrderQty  string `json:"orderQty"`
	UnitPrice string `json:"unitPrice"`
}

type soMutationReq struct {
	SoNo         string      `json:"soNo"`
	CustomerID   string      `json:"customerId"`
	OrderDate    string      `json:"orderDate"`
	CurrencyCode string      `json:"currencyCode"`
	TotalAmount  string      `json:"totalAmount"`
	Lines        []soLineReq `json:"lines"`
}

type soLineResp struct {
	ID           string `json:"id"`
	LineNo       int32  `json:"lineNo"`
	ItemID       string `json:"itemId"`
	OrderQty     string `json:"orderQty"`
	UnitPrice    string `json:"unitPrice"`
	DeliveredQty string `json:"deliveredQty"`
	RowVersion   int32  `json:"rowVersion"`
}

type soDetailResp struct {
	ID             string       `json:"id"`
	OrganizationID string       `json:"organizationId"`
	SoNo           string       `json:"soNo"`
	CustomerID     string       `json:"customerId"`
	OrderDate      string       `json:"orderDate"`
	CurrencyCode   string       `json:"currencyCode"`
	TotalAmount    string       `json:"totalAmount"`
	Status         string       `json:"status"`
	CorrelationID  string       `json:"correlationId"`
	RowVersion     int32        `json:"rowVersion"`
	Lines          []soLineResp `json:"lines"`
}

func toSalesOrderResp(d repository.SalesOrderDetail) soDetailResp {
	lines := make([]soLineResp, 0, len(d.Lines))
	for _, l := range d.Lines {
		lines = append(lines, soLineResp{
			ID:           l.ID.String(),
			LineNo:       l.LineNo,
			ItemID:       l.ItemID.String(),
			OrderQty:     platform.FormatNumeric(l.OrderQty, 4),
			UnitPrice:    platform.FormatNumeric(l.UnitPrice, 4),
			DeliveredQty: platform.FormatNumeric(l.DeliveredQty, 4),
			RowVersion:   l.RowVersion,
		})
	}
	return soDetailResp{
		ID:             d.Header.ID.String(),
		OrganizationID: d.Header.OrganizationID.String(),
		SoNo:           d.Header.SoNo,
		CustomerID:     d.Header.CustomerID.String(),
		OrderDate:      platform.FormatDate(d.Header.OrderDate),
		CurrencyCode:   d.Header.CurrencyCode,
		TotalAmount:    platform.FormatNumeric(d.Header.TotalAmount, 4),
		Status:         d.Header.Status,
		CorrelationID:  d.Header.CorrelationID.String(),
		RowVersion:     d.Header.RowVersion,
		Lines:          lines,
	}
}

type deliveryLineResp struct {
	ID           string `json:"id"`
	SoLineID     string `json:"soLineId"`
	LineNo       int32  `json:"lineNo"`
	ItemID       string `json:"itemId"`
	DeliveredQty string `json:"deliveredQty"`
	RowVersion   int32  `json:"rowVersion"`
}

type deliveryResp struct {
	ID             string             `json:"id"`
	OrganizationID string             `json:"organizationId"`
	DeliveryNo     string             `json:"deliveryNo"`
	SalesOrderID   string             `json:"salesOrderId"`
	WarehouseID    string             `json:"warehouseId"`
	DeliveryDate   string             `json:"deliveryDate"`
	Status         string             `json:"status"`
	CorrelationID  string             `json:"correlationId"`
	RowVersion     int32              `json:"rowVersion"`
	Lines          []deliveryLineResp `json:"lines"`
}

func toDeliveryResp(d repository.DeliveryDetail) deliveryResp {
	lines := make([]deliveryLineResp, 0, len(d.Lines))
	for _, l := range d.Lines {
		lines = append(lines, deliveryLineResp{
			ID:           l.ID.String(),
			SoLineID:     l.SoLineID.String(),
			LineNo:       l.LineNo,
			ItemID:       l.ItemID.String(),
			DeliveredQty: platform.FormatNumeric(l.DeliveredQty, 4),
			RowVersion:   l.RowVersion,
		})
	}
	return deliveryResp{
		ID:             d.Header.ID.String(),
		OrganizationID: d.Header.OrganizationID.String(),
		DeliveryNo:     d.Header.DeliveryNo,
		SalesOrderID:   d.Header.SalesOrderID.String(),
		WarehouseID:    d.Header.WarehouseID.String(),
		DeliveryDate:   platform.FormatDate(d.Header.DeliveryDate),
		Status:         d.Header.Status,
		CorrelationID:  d.Header.CorrelationID.String(),
		RowVersion:     d.Header.RowVersion,
		Lines:          lines,
	}
}

func parseSOMutationBody(req soMutationReq, requireSoNo bool) (
	soNo string,
	customerID uuid.UUID,
	orderDate pgtype.Date,
	currency string,
	totalNum pgtype.Numeric,
	lines []repository.CreateSOLineInput,
	err error,
) {
	soNo = strings.TrimSpace(req.SoNo)
	if requireSoNo && soNo == "" {
		return "", uuid.Nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, httpx.BadRequest("soNo wajib", httpx.Detail{Field: "soNo", Issue: "required"})
	}
	customerID, err = httpx.ParseUUIDParam("customerId", req.CustomerID)
	if err != nil {
		return "", uuid.Nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, err
	}
	orderDate, err = platform.ParseDate(req.OrderDate)
	if err != nil {
		return "", uuid.Nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, httpx.BadRequest("orderDate harus YYYY-MM-DD", httpx.Detail{Field: "orderDate", Issue: err.Error()})
	}
	currency = strings.ToUpper(strings.TrimSpace(req.CurrencyCode))
	if currency == "" {
		currency = "IDR"
	}
	if len(currency) != 3 {
		return "", uuid.Nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, httpx.BadRequest("currencyCode harus 3 huruf ISO", httpx.Detail{Field: "currencyCode", Issue: "must be 3-letter ISO code"})
	}
	totalNum, err = platform.ParseNumeric(req.TotalAmount)
	if err != nil {
		return "", uuid.Nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, httpx.BadRequest("totalAmount tidak valid", httpx.Detail{Field: "totalAmount", Issue: err.Error()})
	}
	totRat := platform.NumericToRat(totalNum)
	if totRat == nil || totRat.Sign() <= 0 {
		return "", uuid.Nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, httpx.BadRequest("totalAmount harus > 0", httpx.Detail{Field: "totalAmount", Issue: "must be > 0"})
	}
	if len(req.Lines) == 0 {
		return "", uuid.Nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, httpx.BadRequest("lines minimal 1 baris", httpx.Detail{Field: "lines", Issue: "required"})
	}

	seenLine := make(map[int32]bool, len(req.Lines))
	lines = make([]repository.CreateSOLineInput, 0, len(req.Lines))
	sumRat := new(big.Rat)
	for idx, l := range req.Lines {
		if l.LineNo <= 0 {
			return "", uuid.Nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, httpx.BadRequest("lineNo harus > 0", httpx.Detail{Field: fmt.Sprintf("lines[%d].lineNo", idx), Issue: "must be > 0"})
		}
		if seenLine[l.LineNo] {
			return "", uuid.Nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, httpx.BadRequest("duplikat lineNo pada sales order", httpx.Detail{Field: fmt.Sprintf("lines[%d].lineNo", idx), Issue: "duplicate"})
		}
		seenLine[l.LineNo] = true

		itemID, err := httpx.ParseUUIDParam("lines.itemId", l.ItemID)
		if err != nil {
			return "", uuid.Nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, err
		}
		qtyNum, err := platform.ParseNumeric(l.OrderQty)
		if err != nil {
			return "", uuid.Nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, httpx.BadRequest("orderQty tidak valid", httpx.Detail{Field: fmt.Sprintf("lines[%d].orderQty", idx), Issue: err.Error()})
		}
		qtyRat := platform.NumericToRat(qtyNum)
		if qtyRat == nil || qtyRat.Sign() <= 0 {
			return "", uuid.Nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, httpx.BadRequest("orderQty harus > 0", httpx.Detail{Field: fmt.Sprintf("lines[%d].orderQty", idx), Issue: "must be > 0"})
		}
		priceNum, err := platform.ParseNumeric(l.UnitPrice)
		if err != nil {
			return "", uuid.Nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, httpx.BadRequest("unitPrice tidak valid", httpx.Detail{Field: fmt.Sprintf("lines[%d].unitPrice", idx), Issue: err.Error()})
		}
		priceRat := platform.NumericToRat(priceNum)
		if priceRat == nil || priceRat.Sign() < 0 {
			return "", uuid.Nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, httpx.BadRequest("unitPrice harus >= 0", httpx.Detail{Field: fmt.Sprintf("lines[%d].unitPrice", idx), Issue: "must be >= 0"})
		}
		sumRat.Add(sumRat, new(big.Rat).Mul(qtyRat, priceRat))
		lines = append(lines, repository.CreateSOLineInput{
			LineNo:    l.LineNo,
			ItemID:    itemID,
			OrderQty:  qtyNum,
			UnitPrice: priceNum,
		})
	}
	if sumRat.Cmp(totRat) != 0 {
		return "", uuid.Nil, pgtype.Date{}, "", pgtype.Numeric{}, nil, httpx.Unprocessable(
			"TOTAL_MISMATCH",
			fmt.Sprintf("totalAmount %s tidak sama dengan jumlah (orderQty * unitPrice) baris %s", totRat.FloatString(4), sumRat.FloatString(4)),
			httpx.Detail{Field: "totalAmount", Issue: "must equal sum(orderQty * unitPrice)"},
		)
	}
	return soNo, customerID, orderDate, currency, totalNum, lines, nil
}

func (h *Handler) handleCreateSO(w http.ResponseWriter, r *http.Request) {
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
	if err := httpx.RequirePermission(r, "sales:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req soMutationReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, h.idem, scope, &req)
	if done {
		return
	}
	defer release()

	soNo, custID, orderDate, currency, totalNum, lines, err := parseSOMutationBody(req, true)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	detail, err := h.store.CreateWithLines(r.Context(), scope, repository.CreateSOInput{
		OrganizationID: scope[0],
		SoNo:           soNo,
		CustomerID:     custID,
		OrderDate:      orderDate,
		CurrencyCode:   currency,
		TotalAmount:    totalNum,
		Lines:          lines,
		CorrelationID:  httpx.CorrelationUUID(r),
		CreatedBy:      user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.SaveIDResponse(w, h.idem, scope, idemKey, reqHash, http.StatusCreated, detail.Header.RowVersion, toSalesOrderResp(detail))
}

func (h *Handler) handleListSO(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "sales:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var statusFilter *string
	if s := strings.TrimSpace(r.URL.Query().Get("status")); s != "" {
		statusFilter = &s
	}
	rows, total, err := h.store.ListWithCount(r.Context(), scope, statusFilter, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"id":             row.ID.String(),
			"organizationId": row.OrganizationID.String(),
			"soNo":           row.SoNo,
			"customerId":     row.CustomerID.String(),
			"orderDate":      platform.FormatDate(row.OrderDate),
			"currencyCode":   row.CurrencyCode,
			"totalAmount":    platform.FormatNumeric(row.TotalAmount, 4),
			"status":         row.Status,
			"rowVersion":     row.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (h *Handler) handleGetSO(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "sales:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	detail, err := h.store.GetByID(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, detail.Header.RowVersion, toSalesOrderResp(detail))
}

func (h *Handler) handleUpdateDraftSO(w http.ResponseWriter, r *http.Request) {
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
	if err := httpx.RequirePermission(r, "sales:write"); err != nil {
		httpx.WriteError(w, r, err)
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
	var req soMutationReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	_, custID, orderDate, currency, totalNum, lines, err := parseSOMutationBody(req, false)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	detail, err := h.store.UpdateDraftWithLines(r.Context(), scope, repository.UpdateDraftSOInput{
		ID:           id,
		RowVersion:   ver,
		CustomerID:   custID,
		OrderDate:    orderDate,
		CurrencyCode: currency,
		TotalAmount:  totalNum,
		Lines:        lines,
		UpdatedBy:    user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, detail.Header.RowVersion, toSalesOrderResp(detail))
}

func (h *Handler) handleDeleteDraftSO(w http.ResponseWriter, r *http.Request) {
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
	if err := httpx.RequirePermission(r, "sales:write"); err != nil {
		httpx.WriteError(w, r, err)
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
	if err := h.store.DeleteDraft(r.Context(), scope, id, ver, user); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleConfirmSO(w http.ResponseWriter, r *http.Request) {
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
	if err := httpx.RequirePermission(r, "sales:write"); err != nil {
		httpx.WriteError(w, r, err)
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
		WarehouseID string `json:"warehouseId"`
	}
	if err := httpx.DecodeOptionalJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var whID *uuid.UUID
	if strings.TrimSpace(req.WarehouseID) != "" {
		parsed, err := httpx.ParseUUIDParam("warehouseId", req.WarehouseID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		whID = &parsed
	}
	detail, err := h.store.ConfirmAndEmit(r.Context(), scope, id, ver, whID, user, httpx.CorrelationUUID(r))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, detail.Header.RowVersion, toSalesOrderResp(detail))
}

func (h *Handler) handleCreateDelivery(w http.ResponseWriter, r *http.Request) {
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
	if err := httpx.RequirePermission(r, "sales:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	soID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
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
		DeliveryNo   string `json:"deliveryNo"`
		WarehouseID  string `json:"warehouseId"`
		DeliveryDate string `json:"deliveryDate"`
	}
	if err := httpx.DecodeOptionalJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var whID uuid.UUID
	if strings.TrimSpace(req.WarehouseID) != "" {
		whID, err = httpx.ParseUUIDParam("warehouseId", req.WarehouseID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
	}
	var delDate pgtype.Date
	if strings.TrimSpace(req.DeliveryDate) != "" {
		delDate, err = platform.ParseDate(req.DeliveryDate)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("deliveryDate harus YYYY-MM-DD", httpx.Detail{Field: "deliveryDate", Issue: err.Error()}))
			return
		}
	}
	detail, err := h.store.CreateDelivery(r.Context(), scope, repository.CreateDeliveryInput{
		SalesOrderID:  soID,
		RowVersion:    ver,
		DeliveryNo:    strings.TrimSpace(req.DeliveryNo),
		WarehouseID:   whID,
		DeliveryDate:  delDate,
		CorrelationID: httpx.CorrelationUUID(r),
		UpdatedBy:     user,
	})
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusCreated, detail.Header.RowVersion, toDeliveryResp(detail))
}

func (h *Handler) handleGetDelivery(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "sales:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	detail, err := h.store.GetDelivery(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, detail.Header.RowVersion, toDeliveryResp(detail))
}

func (h *Handler) handleListDeliveries(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "sales:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := h.store.ListDeliveriesWithCount(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"id":             row.ID.String(),
			"organizationId": row.OrganizationID.String(),
			"deliveryNo":     row.DeliveryNo,
			"salesOrderId":   row.SalesOrderID.String(),
			"warehouseId":    row.WarehouseID.String(),
			"deliveryDate":   platform.FormatDate(row.DeliveryDate),
			"status":         row.Status,
			"rowVersion":     row.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}
