package httpapi

import (
	"encoding/csv"
	"fmt"
	"math/big"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
	"github.com/sinergiinvestasiproperti/erp/services/treasury-cashbank-service/internal/treasurydb"
)

type parsedStatementLine struct {
	StatementDate pgtype.Date
	Amount        pgtype.Numeric
	AmountStr     string
	ExternalRef   string
	Description   string
}

var mt940LineRe = regexp.MustCompile(`^:61:(\d{6})(?:\d{4})?(R?[CD])([0-9,\.]+)([A-Z0-9]{4})([^\r\n]*)`)

// normalizeBankDecimal parses US (1,250,000.50), Indonesian/EU (1.250.000,50),
// or plain decimal strings into canonical decimal format without float64.
func normalizeBankDecimal(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "Rp")
	s = strings.TrimPrefix(s, "IDR")
	s = strings.TrimPrefix(s, "USD")
	s = strings.TrimSpace(s)
	if s == "" || s == "-" {
		return "0.0000", nil
	}
	neg := false
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		neg = true
		s = strings.TrimSpace(s[1 : len(s)-1])
	} else if strings.HasPrefix(s, "-") {
		neg = true
		s = strings.TrimSpace(s[1:])
	} else if strings.HasPrefix(s, "+") {
		s = strings.TrimSpace(s[1:])
	}

	lastDot := strings.LastIndex(s, ".")
	lastComma := strings.LastIndex(s, ",")
	switch {
	case lastComma > lastDot:
		// Indonesian/EU: dots are thousands separators, comma is decimal separator
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	case lastDot > lastComma:
		// US/ISO: commas are thousands separators, dot is decimal separator
		s = strings.ReplaceAll(s, ",", "")
	case lastComma >= 0 && lastDot < 0:
		// Single comma: if 3 digits follow it could be thousands, else decimal
		parts := strings.Split(s, ",")
		if len(parts) == 2 && len(parts[1]) != 3 {
			s = parts[0] + "." + parts[1]
		} else {
			s = strings.ReplaceAll(s, ",", "")
		}
	}
	if neg {
		s = "-" + s
	}
	num, err := platform.ParseNumeric(s)
	if err != nil {
		return "", err
	}
	return platform.FormatNumeric(num, 4), nil
}

func parseFlexBankDate(raw string) (pgtype.Date, error) {
	s := strings.TrimSpace(raw)
	for _, layout := range []string{"2006-01-02", "02/01/2006", "02-01-2006", "2006/01/02", "02 Jan 2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			return pgtype.Date{Time: t.UTC(), Valid: true}, nil
		}
	}
	return pgtype.Date{}, fmt.Errorf("invalid date %q", raw)
}

func parseMT940(content string) ([]parsedStatementLine, error) {
	rawLines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var out []parsedStatementLine
	for i := 0; i < len(rawLines); i++ {
		line := strings.TrimSpace(rawLines[i])
		if !strings.HasPrefix(line, ":61:") {
			continue
		}
		m := mt940LineRe.FindStringSubmatch(line)
		if m == nil {
			return nil, fmt.Errorf("invalid MT940 :61: line %q", line)
		}
		t, err := time.Parse("060102", m[1])
		if err != nil {
			return nil, fmt.Errorf("invalid MT940 date %q: %w", m[1], err)
		}
		dc := m[2]
		amtStr, err := normalizeBankDecimal(m[3])
		if err != nil {
			return nil, fmt.Errorf("invalid MT940 amount %q: %w", m[3], err)
		}
		amtNum, err := platform.ParseNumeric(amtStr)
		if err != nil {
			return nil, err
		}
		if dc == "D" || dc == "RC" {
			rat := platform.NumericToRat(amtNum)
			amtNum, _ = platform.RatToNumeric(new(big.Rat).Neg(rat), 4)
			amtStr = platform.FormatNumeric(amtNum, 4)
		}
		ref := strings.TrimSpace(m[5])
		desc := ""
		if i+1 < len(rawLines) && strings.HasPrefix(strings.TrimSpace(rawLines[i+1]), ":86:") {
			desc = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(rawLines[i+1]), ":86:"))
			i++
		}
		out = append(out, parsedStatementLine{
			StatementDate: pgtype.Date{Time: t.UTC(), Valid: true},
			Amount:        amtNum,
			AmountStr:     amtStr,
			ExternalRef:   ref,
			Description:   desc,
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no :61: statement lines found in MT940 payload")
	}
	return out, nil
}

func parseBankCSV(content string) ([]parsedStatementLine, error) {
	r := csv.NewReader(strings.NewReader(strings.TrimSpace(content)))
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("invalid CSV: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("CSV must include header row and at least one data row")
	}
	idxDate, idxDesc, idxAmt, idxDr, idxCr, idxRef, idxType := -1, -1, -1, -1, -1, -1, -1
	for i, h := range records[0] {
		k := strings.ToLower(strings.TrimSpace(h))
		switch {
		case strings.Contains(k, "date") || strings.Contains(k, "tanggal") || k == "tgl":
			idxDate = i
		case strings.Contains(k, "desc") || strings.Contains(k, "keterangan") || strings.Contains(k, "memo") || strings.Contains(k, "narasi"):
			idxDesc = i
		case k == "debit" || k == "db" || strings.Contains(k, "keluar"):
			idxDr = i
		case k == "credit" || k == "cr" || k == "kredit" || strings.Contains(k, "masuk"):
			idxCr = i
		case strings.Contains(k, "amount") || strings.Contains(k, "jumlah") || strings.Contains(k, "mutasi") || strings.Contains(k, "nominal"):
			idxAmt = i
		case strings.Contains(k, "ref") || strings.Contains(k, "no"):
			idxRef = i
		case k == "type" || k == "dc" || k == "db/cr" || k == "tipe":
			idxType = i
		}
	}
	if idxDate < 0 || (idxAmt < 0 && idxDr < 0 && idxCr < 0) {
		return nil, fmt.Errorf("CSV header must contain date and amount (or debit/credit) columns")
	}

	var out []parsedStatementLine
	for rowIdx, rec := range records[1:] {
		if len(rec) == 0 || (len(rec) == 1 && strings.TrimSpace(rec[0]) == "") {
			continue
		}
		cell := func(idx int) string {
			if idx < 0 || idx >= len(rec) {
				return ""
			}
			return strings.TrimSpace(rec[idx])
		}
		dt, err := parseFlexBankDate(cell(idxDate))
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", rowIdx+2, err)
		}
		var amtRat *big.Rat
		if idxAmt >= 0 && cell(idxAmt) != "" {
			norm, err := normalizeBankDecimal(cell(idxAmt))
			if err != nil {
				return nil, fmt.Errorf("row %d amount: %w", rowIdx+2, err)
			}
			n, _ := platform.ParseNumeric(norm)
			amtRat = platform.NumericToRat(n)
			tp := strings.ToUpper(cell(idxType))
			if (tp == "DB" || tp == "D" || tp == "DEBIT") && amtRat.Sign() > 0 {
				amtRat = new(big.Rat).Neg(amtRat)
			}
		} else {
			drStr, err := normalizeBankDecimal(cell(idxDr))
			if err != nil {
				return nil, fmt.Errorf("row %d debit: %w", rowIdx+2, err)
			}
			crStr, err := normalizeBankDecimal(cell(idxCr))
			if err != nil {
				return nil, fmt.Errorf("row %d credit: %w", rowIdx+2, err)
			}
			drN, _ := platform.ParseNumeric(drStr)
			crN, _ := platform.ParseNumeric(crStr)
			amtRat = new(big.Rat).Sub(platform.NumericToRat(crN), platform.NumericToRat(drN))
		}
		if amtRat == nil || amtRat.Sign() == 0 {
			continue
		}
		amtNum, _ := platform.RatToNumeric(amtRat, 4)
		out = append(out, parsedStatementLine{
			StatementDate: dt,
			Amount:        amtNum,
			AmountStr:     platform.FormatNumeric(amtNum, 4),
			ExternalRef:   cell(idxRef),
			Description:   cell(idxDesc),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no non-zero statement lines found in CSV")
	}
	return out, nil
}

func (s *Server) importBankStatementCSV(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "treasury:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	bankID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		Format  string `json:"format,omitempty"` // csv | mt940 | auto
		Content string `json:"content"`
	}
	key, hash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	if strings.TrimSpace(req.Content) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("content mutasi bank wajib diisi"))
		return
	}
	acct, err := s.repo.GetBankAccount(r.Context(), scope, bankID)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}

	format := strings.ToLower(strings.TrimSpace(req.Format))
	if format == "" || format == "auto" {
		if strings.Contains(req.Content, ":61:") {
			format = "mt940"
		} else {
			format = "csv"
		}
	}
	var parsed []parsedStatementLine
	switch format {
	case "mt940":
		parsed, err = parseMT940(req.Content)
	case "csv":
		parsed, err = parseBankCSV(req.Content)
	default:
		httpx.WriteError(w, r, httpx.BadRequest("format harus csv, mt940, atau auto"))
		return
	}
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest(err.Error()))
		return
	}

	orgID := acct.OrganizationID
	if orgID == uuid.Nil {
		orgID = scope[0]
	}
	params := make([]treasurydb.CreateBankStatementLineParams, 0, len(parsed))
	preview := make([]map[string]any, 0, len(parsed))
	for _, p := range parsed {
		var extRef *string
		if p.ExternalRef != "" {
			v := p.ExternalRef
			extRef = &v
		}
		params = append(params, treasurydb.CreateBankStatementLineParams{
			OrganizationID: orgID,
			BankAccountID:  bankID,
			StatementDate:  p.StatementDate,
			ExternalRef:    extRef,
			Description:    p.Description,
			Amount:         p.Amount,
			CreatedBy:      actor,
		})
		preview = append(preview, map[string]any{
			"statementDate": platform.FormatDate(p.StatementDate),
			"amount":        p.AmountStr,
			"externalRef":   p.ExternalRef,
			"description":   p.Description,
		})
	}
	ids, ver, err := s.repo.CreateStatementLinesBatch(r.Context(), scope, params)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	idStrs := make([]string, 0, len(ids))
	for _, id := range ids {
		idStrs = append(idStrs, id.String())
	}
	httpx.SaveIDResponse(w, s.idem, scope, key, hash, http.StatusCreated, ver, map[string]any{
		"bankAccountId": bankID.String(),
		"format":        format,
		"importedCount": len(params),
		"ids":           idStrs,
		"lines":         preview,
	})
}

func (s *Server) manualReconcileBankAccount(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "treasury:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	actor, err := httpx.RequireUser(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	bankID, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req struct {
		Action          string `json:"action"` // match | unmatch
		StatementLineID string `json:"statementLineId"`
		PaymentID       string `json:"paymentId"`
		Reason          string `json:"reason,omitempty"`
	}
	key, hash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action == "" {
		action = "match"
	}
	if action != "match" && action != "unmatch" {
		httpx.WriteError(w, r, httpx.BadRequest("action harus match atau unmatch"))
		return
	}
	lineID, err := httpx.ParseUUIDParam("statementLineId", req.StatementLineID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	payID, err := httpx.ParseUUIDParam("paymentId", req.PaymentID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := s.repo.ManualReconcileAndEmit(
		r.Context(), scope, bankID, lineID, payID,
		action, strings.TrimSpace(req.Reason), actor, httpx.CorrelationUUID(r),
	); err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, key, hash, http.StatusOK, 0, map[string]any{
		"bankAccountId":   bankID.String(),
		"statementLineId": lineID.String(),
		"paymentId":       payID.String(),
		"action":          action,
		"reason":          strings.TrimSpace(req.Reason),
		"status":          "applied",
	})
}

func (s *Server) cashPosition(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "treasury:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	banks, _, err := s.repo.ListBanksWithCount(r.Context(), scope, treasurydb.ListBankAccountsParams{Limit: 500, Offset: 0})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	payments, _, err := s.repo.ListWithCount(r.Context(), scope, treasurydb.ListPaymentsParams{Limit: 1000, Offset: 0})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}

	bankCcy := make(map[uuid.UUID]string, len(banks))
	type ccyAgg struct {
		settledIn  *big.Rat
		settledOut *big.Rat
		pendingIn  *big.Rat
		pendingOut *big.Rat
		in30       *big.Rat
		out30      *big.Rat
		in60       *big.Rat
		out60      *big.Rat
		in90       *big.Rat
		out90      *big.Rat
	}
	newAgg := func() *ccyAgg {
		return &ccyAgg{
			settledIn: new(big.Rat), settledOut: new(big.Rat),
			pendingIn: new(big.Rat), pendingOut: new(big.Rat),
			in30: new(big.Rat), out30: new(big.Rat),
			in60: new(big.Rat), out60: new(big.Rat),
			in90: new(big.Rat), out90: new(big.Rat),
		}
	}
	byCcy := make(map[string]*ccyAgg)
	for _, b := range banks {
		ccy := strings.ToUpper(strings.TrimSpace(b.CurrencyCode))
		if ccy == "" {
			ccy = "IDR"
		}
		bankCcy[b.ID] = ccy
		if _, ok := byCcy[ccy]; !ok {
			byCcy[ccy] = newAgg()
		}
	}

	now := time.Now().UTC().Truncate(24 * time.Hour)
	for _, p := range payments {
		ccy, ok := bankCcy[p.BankAccountID]
		if !ok {
			ccy = "IDR"
		}
		agg, ok := byCcy[ccy]
		if !ok {
			agg = newAgg()
			byCcy[ccy] = agg
		}
		amt := platform.NumericToRat(p.Amount)
		if amt == nil {
			continue
		}
		absAmt := new(big.Rat).Abs(amt)
		isIn := strings.EqualFold(p.PaymentType, "Incoming")
		if p.Status == "Settled" || p.Status == "Reconciled" {
			if isIn {
				agg.settledIn.Add(agg.settledIn, absAmt)
			} else {
				agg.settledOut.Add(agg.settledOut, absAmt)
			}
		} else if p.Status == "Draft" || p.Status == "Approved" {
			if isIn {
				agg.pendingIn.Add(agg.pendingIn, absAmt)
			} else {
				agg.pendingOut.Add(agg.pendingOut, absAmt)
			}
			days := 0
			if p.PaymentDate.Valid {
				days = int(p.PaymentDate.Time.UTC().Truncate(24*time.Hour).Sub(now).Hours() / 24)
			}
			switch {
			case days <= 30:
				if isIn {
					agg.in30.Add(agg.in30, absAmt)
				} else {
					agg.out30.Add(agg.out30, absAmt)
				}
			case days <= 60:
				if isIn {
					agg.in60.Add(agg.in60, absAmt)
				} else {
					agg.out60.Add(agg.out60, absAmt)
				}
			default:
				if isIn {
					agg.in90.Add(agg.in90, absAmt)
				} else {
					agg.out90.Add(agg.out90, absAmt)
				}
			}
		}
	}

	fmtRat := func(r *big.Rat) string {
		n, _ := platform.RatToNumeric(r, 4)
		return platform.FormatNumeric(n, 4)
	}
	ccys := make([]string, 0, len(byCcy))
	for c := range byCcy {
		ccys = append(ccys, c)
	}
	sort.Strings(ccys)

	positions := make([]map[string]any, 0, len(ccys))
	for _, ccy := range ccys {
		a := byCcy[ccy]
		netCash := new(big.Rat).Sub(a.settledIn, a.settledOut)
		net30 := new(big.Rat).Sub(a.in30, a.out30)
		net60 := new(big.Rat).Sub(a.in60, a.out60)
		net90 := new(big.Rat).Sub(a.in90, a.out90)
		proj30 := new(big.Rat).Add(netCash, net30)
		proj60 := new(big.Rat).Add(proj30, net60)
		proj90 := new(big.Rat).Add(proj60, net90)

		positions = append(positions, map[string]any{
			"currencyCode":   ccy,
			"settledInflow":  fmtRat(a.settledIn),
			"settledOutflow": fmtRat(a.settledOut),
			"netBalance":     fmtRat(netCash),
			"pendingInflow":  fmtRat(a.pendingIn),
			"pendingOutflow": fmtRat(a.pendingOut),
			"forecast": map[string]any{
				"days30": map[string]any{"inflow": fmtRat(a.in30), "outflow": fmtRat(a.out30), "net": fmtRat(net30), "projectedBalance": fmtRat(proj30)},
				"days60": map[string]any{"inflow": fmtRat(a.in60), "outflow": fmtRat(a.out60), "net": fmtRat(net60), "projectedBalance": fmtRat(proj60)},
				"days90": map[string]any{"inflow": fmtRat(a.in90), "outflow": fmtRat(a.out90), "net": fmtRat(net90), "projectedBalance": fmtRat(proj90)},
			},
		})
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"asOfDate":    now.Format("2006-01-02"),
		"bankCount":   len(banks),
		"currencies":  positions,
	})
}
