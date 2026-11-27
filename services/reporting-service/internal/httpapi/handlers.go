package httpapi

import (
	"bytes"
	"context"
	"encoding/csv"
	"math/big"
	"net/http"
	"sort"
	"strings"
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
	mux.HandleFunc("GET /financial-statements/income-statement", s.incomeStatement)
	mux.HandleFunc("GET /financial-statements/balance-sheet", s.balanceSheet)
	mux.HandleFunc("GET /financial-statements/cash-flow", s.cashFlow)
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

// -----------------------------------------------------------------------------
// Structured Financial Statements (P&L, Balance Sheet, Cash Flow)
// -----------------------------------------------------------------------------

type statementItem struct {
	AccountCode string `json:"accountCode"`
	AccountName string `json:"accountName"`
	Amount      string `json:"amount"`
}

type statementSection struct {
	Code     string          `json:"code"`
	Name     string          `json:"name"`
	Items    []statementItem `json:"items"`
	Subtotal string          `json:"subtotal"`
}

type financialStatementResp struct {
	StatementType string             `json:"statementType"`
	Currency      string             `json:"currency"`
	AsOfDate      string             `json:"asOfDate"`
	Period        string             `json:"period"`
	Sections      []statementSection `json:"sections"`
	Totals        map[string]string  `json:"totals"`
	IsBalanced    *bool              `json:"isBalanced,omitempty"`
}

func (s *Server) loadStatementSnapshots(r *http.Request) ([]reportingdb.ListGLBalanceSnapshotsRow, string, string, string, error) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		return nil, "", "", "", err
	}
	if err := httpx.RequirePermission(r, "reporting.read"); err != nil {
		return nil, "", "", "", err
	}
	rows, _, err := s.repo.ListGLWithCount(r.Context(), scope, reportingdb.ListGLBalanceSnapshotsParams{Limit: 10000, Offset: 0})
	if err != nil {
		return nil, "", "", "", httpx.MapDBError(err, "")
	}
	if fpRaw := strings.TrimSpace(r.URL.Query().Get("fiscalPeriodId")); fpRaw != "" {
		fpID, perr := uuid.Parse(fpRaw)
		if perr != nil {
			return nil, "", "", "", httpx.BadRequest("fiscalPeriodId must be a valid UUID")
		}
		filtered := make([]reportingdb.ListGLBalanceSnapshotsRow, 0, len(rows))
		for _, row := range rows {
			if row.FiscalPeriodID == fpID {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}
	currency := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("currency")))
	if currency == "" {
		currency = "IDR"
	}
	asOfDate := strings.TrimSpace(r.URL.Query().Get("asOfDate"))
	if asOfDate == "" {
		asOfDate = time.Now().UTC().Format("2006-01-02")
	}
	period := strings.TrimSpace(r.URL.Query().Get("period"))
	if period == "" {
		period = asOfDate[:7]
	}
	return rows, currency, asOfDate, period, nil
}

// aggregateByAccount sums snapshot balances per accountCode using exact *big.Rat.
func aggregateByAccount(rows []reportingdb.ListGLBalanceSnapshotsRow) ([]string, map[string]*big.Rat) {
	sums := make(map[string]*big.Rat, len(rows))
	for _, row := range rows {
		code := strings.TrimSpace(row.AccountCode)
		if code == "" {
			continue
		}
		r := platform.NumericToRat(row.ClosingBalance)
		if r == nil {
			r = new(big.Rat)
		}
		if cur, ok := sums[code]; ok {
			cur.Add(cur, r)
		} else {
			sums[code] = new(big.Rat).Set(r)
		}
	}
	codes := make([]string, 0, len(sums))
	for k := range sums {
		codes = append(codes, k)
	}
	sort.Strings(codes)
	return codes, sums
}

func (s *Server) incomeStatement(w http.ResponseWriter, r *http.Request) {
	rows, currency, asOfDate, period, err := s.loadStatementSnapshots(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	codes, sums := aggregateByAccount(rows)

	revItems := make([]statementItem, 0)
	cogsItems := make([]statementItem, 0)
	opexItems := make([]statementItem, 0)
	revTotal := new(big.Rat)
	cogsTotal := new(big.Rat)
	opexTotal := new(big.Rat)

	for _, code := range codes {
		bal := sums[code]
		switch {
		case strings.HasPrefix(code, "4"):
			// Credit-normal revenue: negate debit-positive closing_balance.
			amt := new(big.Rat).Neg(bal)
			revItems = append(revItems, statementItem{AccountCode: code, AccountName: "Account " + code, Amount: amt.FloatString(4)})
			revTotal.Add(revTotal, amt)
		case strings.HasPrefix(code, "5"):
			amt := new(big.Rat).Set(bal)
			cogsItems = append(cogsItems, statementItem{AccountCode: code, AccountName: "Account " + code, Amount: amt.FloatString(4)})
			cogsTotal.Add(cogsTotal, amt)
		case strings.HasPrefix(code, "6") || strings.HasPrefix(code, "7") || strings.HasPrefix(code, "8") || strings.HasPrefix(code, "9"):
			amt := new(big.Rat).Set(bal)
			opexItems = append(opexItems, statementItem{AccountCode: code, AccountName: "Account " + code, Amount: amt.FloatString(4)})
			opexTotal.Add(opexTotal, amt)
		}
	}

	grossProfit := new(big.Rat).Sub(revTotal, cogsTotal)
	netIncome := new(big.Rat).Sub(grossProfit, opexTotal)

	resp := financialStatementResp{
		StatementType: "INCOME_STATEMENT",
		Currency:      currency,
		AsOfDate:      asOfDate,
		Period:        period,
		Sections: []statementSection{
			{Code: "REVENUE", Name: "Revenue", Items: revItems, Subtotal: revTotal.FloatString(4)},
			{Code: "COGS", Name: "Cost of Goods Sold", Items: cogsItems, Subtotal: cogsTotal.FloatString(4)},
			{Code: "OPERATING_EXPENSES", Name: "Operating Expenses", Items: opexItems, Subtotal: opexTotal.FloatString(4)},
		},
		Totals: map[string]string{
			"revenue":           revTotal.FloatString(4),
			"cogs":              cogsTotal.FloatString(4),
			"grossProfit":       grossProfit.FloatString(4),
			"operatingExpenses": opexTotal.FloatString(4),
			"netIncome":         netIncome.FloatString(4),
		},
	}
	writeStatementResponse(w, r, resp, "income-statement")
}

func (s *Server) balanceSheet(w http.ResponseWriter, r *http.Request) {
	rows, currency, asOfDate, period, err := s.loadStatementSnapshots(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	codes, sums := aggregateByAccount(rows)

	assetItems := make([]statementItem, 0)
	liabItems := make([]statementItem, 0)
	eqItems := make([]statementItem, 0)
	assetTotal := new(big.Rat)
	liabTotal := new(big.Rat)
	eqTotal := new(big.Rat)
	currentNetIncome := new(big.Rat)

	for _, code := range codes {
		bal := sums[code]
		switch {
		case strings.HasPrefix(code, "1"):
			amt := new(big.Rat).Set(bal)
			assetItems = append(assetItems, statementItem{AccountCode: code, AccountName: "Account " + code, Amount: amt.FloatString(4)})
			assetTotal.Add(assetTotal, amt)
		case strings.HasPrefix(code, "2"):
			amt := new(big.Rat).Neg(bal)
			liabItems = append(liabItems, statementItem{AccountCode: code, AccountName: "Account " + code, Amount: amt.FloatString(4)})
			liabTotal.Add(liabTotal, amt)
		case strings.HasPrefix(code, "3"):
			amt := new(big.Rat).Neg(bal)
			eqItems = append(eqItems, statementItem{AccountCode: code, AccountName: "Account " + code, Amount: amt.FloatString(4)})
			eqTotal.Add(eqTotal, amt)
		case strings.HasPrefix(code, "4") || strings.HasPrefix(code, "5") || strings.HasPrefix(code, "6") || strings.HasPrefix(code, "7") || strings.HasPrefix(code, "8") || strings.HasPrefix(code, "9"):
			// P&L accounts roll into current-period retained earnings (-bal because credit-normal income).
			currentNetIncome.Sub(currentNetIncome, bal)
		}
	}
	if currentNetIncome.Sign() != 0 {
		eqItems = append(eqItems, statementItem{
			AccountCode: "3999",
			AccountName: "Current Period Net Income",
			Amount:      currentNetIncome.FloatString(4),
		})
		eqTotal.Add(eqTotal, currentNetIncome)
	}

	liabAndEq := new(big.Rat).Add(liabTotal, eqTotal)
	balanced := assetTotal.Cmp(liabAndEq) == 0

	resp := financialStatementResp{
		StatementType: "BALANCE_SHEET",
		Currency:      currency,
		AsOfDate:      asOfDate,
		Period:        period,
		Sections: []statementSection{
			{Code: "ASSETS", Name: "Assets", Items: assetItems, Subtotal: assetTotal.FloatString(4)},
			{Code: "LIABILITIES", Name: "Liabilities", Items: liabItems, Subtotal: liabTotal.FloatString(4)},
			{Code: "EQUITY", Name: "Equity", Items: eqItems, Subtotal: eqTotal.FloatString(4)},
		},
		Totals: map[string]string{
			"totalAssets":               assetTotal.FloatString(4),
			"totalLiabilities":          liabTotal.FloatString(4),
			"totalEquity":               eqTotal.FloatString(4),
			"totalLiabilitiesAndEquity": liabAndEq.FloatString(4),
		},
		IsBalanced: &balanced,
	}
	writeStatementResponse(w, r, resp, "balance-sheet")
}

func (s *Server) cashFlow(w http.ResponseWriter, r *http.Request) {
	rows, currency, asOfDate, period, err := s.loadStatementSnapshots(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	codes, sums := aggregateByAccount(rows)

	opItems := make([]statementItem, 0)
	invItems := make([]statementItem, 0)
	finItems := make([]statementItem, 0)
	opTotal := new(big.Rat)
	invTotal := new(big.Rat)
	finTotal := new(big.Rat)
	netIncome := new(big.Rat)

	for _, code := range codes {
		bal := sums[code]
		switch {
		case strings.HasPrefix(code, "4") || strings.HasPrefix(code, "5") || strings.HasPrefix(code, "6") || strings.HasPrefix(code, "7") || strings.HasPrefix(code, "8") || strings.HasPrefix(code, "9"):
			netIncome.Sub(netIncome, bal)
		case strings.HasPrefix(code, "15") || strings.HasPrefix(code, "16") || strings.HasPrefix(code, "17"):
			// Fixed/non-current asset additions are cash outflows (-bal).
			amt := new(big.Rat).Neg(bal)
			invItems = append(invItems, statementItem{AccountCode: code, AccountName: "Investing - " + code, Amount: amt.FloatString(4)})
			invTotal.Add(invTotal, amt)
		case strings.HasPrefix(code, "22") || strings.HasPrefix(code, "23") || strings.HasPrefix(code, "24") || strings.HasPrefix(code, "3"):
			// Long-term debt & equity inflows (-bal).
			amt := new(big.Rat).Neg(bal)
			finItems = append(finItems, statementItem{AccountCode: code, AccountName: "Financing - " + code, Amount: amt.FloatString(4)})
			finTotal.Add(finTotal, amt)
		case strings.HasPrefix(code, "12") || strings.HasPrefix(code, "13") || strings.HasPrefix(code, "14") || strings.HasPrefix(code, "21"):
			// Working capital changes (AR/Inventory/AP): -bal is cash impact.
			amt := new(big.Rat).Neg(bal)
			opItems = append(opItems, statementItem{AccountCode: code, AccountName: "Working Capital - " + code, Amount: amt.FloatString(4)})
			opTotal.Add(opTotal, amt)
		}
	}
	if netIncome.Sign() != 0 {
		opItems = append([]statementItem{{
			AccountCode: "NET_INCOME",
			AccountName: "Net Income",
			Amount:      netIncome.FloatString(4),
		}}, opItems...)
		opTotal.Add(opTotal, netIncome)
	}

	netCash := new(big.Rat).Add(opTotal, invTotal)
	netCash.Add(netCash, finTotal)

	resp := financialStatementResp{
		StatementType: "CASH_FLOW",
		Currency:      currency,
		AsOfDate:      asOfDate,
		Period:        period,
		Sections: []statementSection{
			{Code: "OPERATING", Name: "Operating Activities", Items: opItems, Subtotal: opTotal.FloatString(4)},
			{Code: "INVESTING", Name: "Investing Activities", Items: invItems, Subtotal: invTotal.FloatString(4)},
			{Code: "FINANCING", Name: "Financing Activities", Items: finItems, Subtotal: finTotal.FloatString(4)},
		},
		Totals: map[string]string{
			"operatingCashFlow": opTotal.FloatString(4),
			"investingCashFlow": invTotal.FloatString(4),
			"financingCashFlow": finTotal.FloatString(4),
			"netCashChange":     netCash.FloatString(4),
		},
	}
	writeStatementResponse(w, r, resp, "cash-flow")
}

func writeStatementResponse(w http.ResponseWriter, r *http.Request, resp financialStatementResp, filename string) {
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("format")), "csv") {
		var buf bytes.Buffer
		cw := csv.NewWriter(&buf)
		_ = cw.Write([]string{"Section", "AccountCode", "AccountName", "Amount", "Currency"})
		for _, sec := range resp.Sections {
			for _, item := range sec.Items {
				_ = cw.Write([]string{
					httpx.SanitizeCSVCell(sec.Name),
					httpx.SanitizeCSVCell(item.AccountCode),
					httpx.SanitizeCSVCell(item.AccountName),
					httpx.SanitizeCSVCell(item.Amount),
					httpx.SanitizeCSVCell(resp.Currency),
				})
			}
			_ = cw.Write([]string{
				httpx.SanitizeCSVCell(sec.Name + " Subtotal"),
				"",
				"",
				httpx.SanitizeCSVCell(sec.Subtotal),
				httpx.SanitizeCSVCell(resp.Currency),
			})
		}
		totalKeys := make([]string, 0, len(resp.Totals))
		for k := range resp.Totals {
			totalKeys = append(totalKeys, k)
		}
		sort.Strings(totalKeys)
		for _, k := range totalKeys {
			_ = cw.Write([]string{"TOTAL", httpx.SanitizeCSVCell(k), "", httpx.SanitizeCSVCell(resp.Totals[k]), httpx.SanitizeCSVCell(resp.Currency)})
		}
		cw.Flush()
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`-`+resp.Period+`.csv"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}
