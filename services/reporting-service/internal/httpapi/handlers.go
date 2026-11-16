package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/reporting-service/internal/reportingdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type Store interface {
	ListGLWithCount(ctx context.Context, scope []uuid.UUID, arg reportingdb.ListGLBalanceSnapshotsParams) ([]reportingdb.ListGLBalanceSnapshotsRow, int64, error)
	ListSalesWithCount(ctx context.Context, scope []uuid.UUID, arg reportingdb.ListSalesSummariesParams) ([]reportingdb.RptSalesSummary, int64, error)
	ListInventoryWithCount(ctx context.Context, scope []uuid.UUID, arg reportingdb.ListInventoryPositionsParams) ([]reportingdb.RptInventoryPosition, int64, error)
}

type Server struct{ repo Store }

func NewServer(repo Store) *Server { return &Server{repo: repo} }

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /gl-balance-snapshots", s.listGL)
	mux.HandleFunc("GET /sales-summaries", s.listSales)
	mux.HandleFunc("GET /inventory-positions", s.listInventory)
}

type snapshotDTO struct {
	ID             string `json:"id"`
	FiscalPeriodID string `json:"fiscalPeriodId"`
	AccountCode    string `json:"accountCode"`
	ClosingBalance string `json:"closingBalance"`
	RefreshedAt    string `json:"refreshedAt"`
}

func (s *Server) listGL(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "reporting.read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListGLWithCount(r.Context(), scope, reportingdb.ListGLBalanceSnapshotsParams{Limit: limit, Offset: offset})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]snapshotDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, snapshotDTO{
			ID:             row.ID.String(),
			FiscalPeriodID: row.FiscalPeriodID.String(),
			AccountCode:    row.AccountCode,
			ClosingBalance: platform.FormatNumeric(row.ClosingBalance, 4),
			RefreshedAt:    tsStr(row.RefreshedAt),
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) listSales(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "reporting.read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListSalesWithCount(r.Context(), scope, reportingdb.ListSalesSummariesParams{Limit: limit, Offset: offset})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		cust := ""
		if row.CustomerID != nil {
			cust = row.CustomerID.String()
		}
		out = append(out, map[string]any{
			"id":               row.ID.String(),
			"organizationId":   row.OrganizationID.String(),
			"periodYear":       row.PeriodYear,
			"periodMonth":      row.PeriodMonth,
			"customerId":       cust,
			"totalSalesAmount": platform.FormatNumeric(row.TotalSalesAmount, 4),
			"orderCount":       row.OrderCount,
			"refreshedAt":      tsStr(row.RefreshedAt),
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) listInventory(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "reporting.read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListInventoryWithCount(r.Context(), scope, reportingdb.ListInventoryPositionsParams{Limit: limit, Offset: offset})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]any{
			"id":             row.ID.String(),
			"organizationId": row.OrganizationID.String(),
			"warehouseCode":  row.WarehouseCode,
			"itemCode":       row.ItemCode,
			"onHandQty":      platform.FormatNumeric(row.OnHandQty, 4),
			"availableQty":   platform.FormatNumeric(row.AvailableQty, 4),
			"refreshedAt":    tsStr(row.RefreshedAt),
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func tsStr(t pgtype.Timestamptz) string {
	if !t.Valid {
		return ""
	}
	return t.Time.UTC().Format(time.RFC3339)
}
