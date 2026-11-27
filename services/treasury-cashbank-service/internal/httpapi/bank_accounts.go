package httpapi

import (
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
	"github.com/sinergiinvestasiproperti/erp/services/treasury-cashbank-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/treasury-cashbank-service/internal/treasurydb"
)

type createBankAccountReq struct {
	OrganizationID uuid.UUID `json:"organizationId"`
	Code           string    `json:"code"`
	BankName       string    `json:"bankName"`
	AccountNo      string    `json:"accountNo"`
	CurrencyCode   string    `json:"currencyCode"`
	GlAccountID    uuid.UUID `json:"glAccountId"`
}

func (s *Server) createBankAccount(w http.ResponseWriter, r *http.Request) {
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
	var req createBankAccountReq
	key, hash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	org, err := pickOrg(req.OrganizationID, scope)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.BankName) == "" ||
		strings.TrimSpace(req.AccountNo) == "" || req.GlAccountID == uuid.Nil {
		httpx.WriteError(w, r, httpx.BadRequest("code, bankName, accountNo, glAccountId wajib diisi"))
		return
	}
	ccy := strings.ToUpper(strings.TrimSpace(req.CurrencyCode))
	if ccy == "" {
		ccy = "IDR"
	}
	if len(ccy) != 3 {
		httpx.WriteError(w, r, httpx.BadRequest("currencyCode harus 3 karakter ISO-4217"))
		return
	}
	res, err := s.repo.CreateBankAccount(r.Context(), scope, treasurydb.CreateBankAccountParams{
		OrganizationID: org,
		Code:           strings.TrimSpace(req.Code),
		BankName:       strings.TrimSpace(req.BankName),
		AccountNo:      strings.TrimSpace(req.AccountNo),
		CurrencyCode:   ccy,
		GlAccountID:    req.GlAccountID,
		CreatedBy:      actor,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "kode rekening bank sudah dipakai"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, key, hash, http.StatusCreated, res.RowVersion, map[string]any{
		"id":         res.ID,
		"code":       strings.TrimSpace(req.Code),
		"rowVersion": res.RowVersion,
	})
}

func (s *Server) listBankAccounts(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "treasury:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var ccy *string
	if v := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("currencyCode"))); v != "" {
		ccy = &v
	}
	rows, total, err := s.repo.ListBanksWithCount(r.Context(), scope, treasurydb.ListBankAccountsParams{
		Limit:        limit,
		Offset:       offset,
		CurrencyCode: ccy,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, b := range rows {
		out = append(out, map[string]any{
			"id":             b.ID,
			"organizationId": b.OrganizationID,
			"code":           b.Code,
			"bankName":       b.BankName,
			"accountNo":      b.AccountNo,
			"currencyCode":   b.CurrencyCode,
			"glAccountId":    b.GlAccountID,
			"rowVersion":     b.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) getBankAccount(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "treasury:read"); err != nil {
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
	b, err := s.repo.GetBankAccount(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("rekening bank tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, b.RowVersion, map[string]any{
		"id":             b.ID,
		"organizationId": b.OrganizationID,
		"code":           b.Code,
		"bankName":       b.BankName,
		"accountNo":      b.AccountNo,
		"currencyCode":   b.CurrencyCode,
		"glAccountId":    b.GlAccountID,
		"rowVersion":     b.RowVersion,
	})
}


type companionStatementLineItem struct {
	BankAccountID *uuid.UUID      `json:"bankAccountId,omitempty"`
	StatementDate string          `json:"statementDate,omitempty"`
	Date          string          `json:"date,omitempty"`
	Description   string          `json:"description"`
	Amount        json.RawMessage `json:"amount,omitempty"`
	Debit         json.RawMessage `json:"debit,omitempty"`
	Credit        json.RawMessage `json:"credit,omitempty"`
	Balance       json.RawMessage `json:"balance,omitempty"`
	ExternalRef   *string         `json:"externalRef,omitempty"`
}

type createStatementLineReq struct {
	OrganizationID   uuid.UUID                    `json:"organizationId"`
	BankAccountID    uuid.UUID                    `json:"bankAccountId"`
	StatementDate    string                       `json:"statementDate"`
	Description      string                       `json:"description"`
	Amount           string                       `json:"amount"`
	ExternalRef      *string                      `json:"externalRef"`
	Lines            []companionStatementLineItem `json:"lines,omitempty"`
	BankName         string                       `json:"bankName,omitempty"`
	AccountNumber    string                       `json:"accountNumber,omitempty"`
	Origin           string                       `json:"origin,omitempty"`
	ExtensionVersion string                       `json:"extension_version,omitempty"`
}

func (s *Server) createStatementLine(w http.ResponseWriter, r *http.Request) {
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
	var req createStatementLineReq
	key, hash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	org, err := pickOrg(req.OrganizationID, scope)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	if len(req.Lines) > 0 {
		bankAccountID := req.BankAccountID
		acctNo := strings.TrimSpace(req.AccountNumber)
		bankName := strings.TrimSpace(req.BankName)
		if bankAccountID == uuid.Nil {
			if banks, _, berr := s.repo.ListBanksWithCount(r.Context(), scope, treasurydb.ListBankAccountsParams{
				Limit:  100,
				Offset: 0,
			}); berr == nil && len(banks) > 0 {
				for _, b := range banks {
					if acctNo != "" && strings.EqualFold(strings.TrimSpace(b.AccountNo), acctNo) {
						bankAccountID = b.ID
						break
					}
				}
				if bankAccountID == uuid.Nil && bankName != "" {
					for _, b := range banks {
						if strings.EqualFold(strings.TrimSpace(b.BankName), bankName) ||
							strings.Contains(strings.ToUpper(b.BankName), strings.ToUpper(bankName)) {
							bankAccountID = b.ID
							break
						}
					}
				}
				if bankAccountID == uuid.Nil && acctNo == "" && bankName == "" && len(banks) == 1 {
					bankAccountID = banks[0].ID
				}
			}
		}

		type parsedLine struct {
			bankID uuid.UUID
			desc   string
			date   pgtype.Date
			amt    pgtype.Numeric
			extRef *string
		}
		parsedLines := make([]parsedLine, 0, len(req.Lines))
		for _, item := range req.Lines {
			lineBankID := bankAccountID
			if item.BankAccountID != nil && *item.BankAccountID != uuid.Nil {
				lineBankID = *item.BankAccountID
			}
			if lineBankID == uuid.Nil {
				httpx.WriteError(w, r, httpx.BadRequest("bankAccountId wajib diisi atau rekening bank yang sesuai harus terdaftar"))
				return
			}
			amt, aerr := resolveCompanionLineAmount(item)
			if aerr != nil {
				httpx.WriteError(w, r, httpx.BadRequest("amount pada lines tidak valid"))
				return
			}
			if rVal := platform.NumericToRat(amt); rVal == nil || rVal.Sign() == 0 {
				continue
			}
			desc := strings.TrimSpace(item.Description)
			if desc == "" {
				desc = "Bank Statement Line"
			}
			parsedLines = append(parsedLines, parsedLine{
				bankID: lineBankID,
				desc:   desc,
				date:   parseCompanionStatementDate(item.StatementDate, item.Date),
				amt:    amt,
				extRef: item.ExternalRef,
			})
		}

		batchParams := make([]treasurydb.CreateBankStatementLineParams, 0, len(parsedLines))
		for _, pl := range parsedLines {
			batchParams = append(batchParams, treasurydb.CreateBankStatementLineParams{
				OrganizationID: org,
				BankAccountID:  pl.bankID,
				StatementDate:  pl.date,
				Description:    pl.desc,
				Amount:         pl.amt,
				ExternalRef:    pl.extRef,
				CreatedBy:      actor,
			})
		}
		createdIDs, lastVer, cerr := s.repo.CreateStatementLinesBatch(r.Context(), scope, batchParams)
		if cerr != nil {
			if errors.Is(cerr, pgx.ErrNoRows) {
				httpx.WriteError(w, r, httpx.NotFound("rekening bank tidak ditemukan"))
				return
			}
			httpx.WriteError(w, r, httpx.MapDBError(cerr, ""))
			return
		}
		httpx.SaveIDResponse(w, s.idem, scope, key, hash, http.StatusCreated, lastVer, map[string]any{
			"importedCount": len(createdIDs),
			"ids":           createdIDs,
			"bankAccountId": bankAccountID,
			"bankName":      req.BankName,
			"origin":        req.Origin,
		})
		return
	}

	if req.BankAccountID == uuid.Nil || strings.TrimSpace(req.Description) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("bankAccountId dan description wajib diisi"))
		return
	}
	stDate, err := platform.ParseDate(req.StatementDate)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("statementDate harus YYYY-MM-DD"))
		return
	}
	amt, err := platform.ParseNumeric(req.Amount)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("amount tidak valid"))
		return
	}
	if rVal := platform.NumericToRat(amt); rVal == nil || rVal.Sign() == 0 {
		httpx.WriteError(w, r, httpx.BadRequest("amount tidak boleh nol"))
		return
	}
	row, err := s.repo.CreateStatementLine(r.Context(), scope, treasurydb.CreateBankStatementLineParams{
		OrganizationID: org,
		BankAccountID:  req.BankAccountID,
		StatementDate:  stDate,
		Description:    strings.TrimSpace(req.Description),
		Amount:         amt,
		ExternalRef:    req.ExternalRef,
		CreatedBy:      actor,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("rekening bank tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, key, hash, http.StatusCreated, row.RowVersion, map[string]any{
		"id":            row.ID,
		"bankAccountId": req.BankAccountID,
		"isReconciled":  row.IsReconciled,
		"rowVersion":    row.RowVersion,
	})
}

func parseCompanionStatementDate(primary, fallback string) pgtype.Date {
	for _, cand := range []string{primary, fallback} {
		raw := strings.TrimSpace(cand)
		if raw == "" {
			continue
		}
		if d, err := platform.ParseDate(raw); err == nil {
			return d
		}
		for _, layout := range []string{"02/01/2006", "02-01-2006", "2006/01/02"} {
			if t, err := time.Parse(layout, raw); err == nil {
				return pgtype.Date{Time: t.UTC(), Valid: true}
			}
		}
	}
	return pgtype.Date{Time: time.Now().UTC(), Valid: true}
}

func rawJSONToDecimalString(raw json.RawMessage) (string, error) {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return "", nil
	}
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		var unquoted string
		if err := json.Unmarshal(raw, &unquoted); err != nil {
			return "", err
		}
		return strings.TrimSpace(unquoted), nil
	}
	return s, nil
}

func resolveCompanionLineAmount(item companionStatementLineItem) (pgtype.Numeric, error) {
	rawAmt, err := rawJSONToDecimalString(item.Amount)
	if err != nil {
		return pgtype.Numeric{}, err
	}
	if rawAmt != "" {
		n, err := platform.ParseNumeric(rawAmt)
		if err != nil {
			return pgtype.Numeric{}, err
		}
		if rVal := platform.NumericToRat(n); rVal != nil && rVal.Sign() != 0 {
			return n, nil
		}
	}
	cr := new(big.Rat)
	dr := new(big.Rat)
	rawCr, err := rawJSONToDecimalString(item.Credit)
	if err != nil {
		return pgtype.Numeric{}, err
	}
	if rawCr != "" {
		n, err := platform.ParseNumeric(rawCr)
		if err != nil {
			return pgtype.Numeric{}, err
		}
		if rVal := platform.NumericToRat(n); rVal != nil {
			cr.Set(rVal)
		}
	}
	rawDr, err := rawJSONToDecimalString(item.Debit)
	if err != nil {
		return pgtype.Numeric{}, err
	}
	if rawDr != "" {
		n, err := platform.ParseNumeric(rawDr)
		if err != nil {
			return pgtype.Numeric{}, err
		}
		if rVal := platform.NumericToRat(n); rVal != nil {
			dr.Set(rVal)
		}
	}
	net := new(big.Rat).Sub(cr, dr)
	return platform.ParseNumeric(net.FloatString(4))
}

func (s *Server) listStatementLines(w http.ResponseWriter, r *http.Request) {
	if err := httpx.RequirePermission(r, "treasury:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	var bankIDPtr *uuid.UUID
	if rawBank := strings.TrimSpace(r.URL.Query().Get("bankAccountId")); rawBank != "" {
		bid, err := httpx.ParseUUIDParam("bankAccountId", rawBank)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		bankIDPtr = &bid
	}
	var reconciledPtr *bool
	if rawRec := strings.TrimSpace(r.URL.Query().Get("isReconciled")); rawRec != "" {
		b, err := strconv.ParseBool(rawRec)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("isReconciled harus true/false"))
			return
		}
		reconciledPtr = &b
	}
	rows, total, err := s.repo.ListStatementLinesWithCount(r.Context(), scope, treasurydb.ListBankStatementLinesParams{
		Limit:         limit,
		Offset:        offset,
		BankAccountID: bankIDPtr,
		IsReconciled:  reconciledPtr,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, l := range rows {
		out = append(out, map[string]any{
			"id":               l.ID,
			"organizationId":   l.OrganizationID,
			"bankAccountId":    l.BankAccountID,
			"statementDate":    platform.FormatDate(l.StatementDate),
			"description":      l.Description,
			"amount":           platform.FormatNumeric(l.Amount, 4),
			"externalRef":      l.ExternalRef,
			"matchedPaymentId": l.MatchedPaymentID,
			"isReconciled":     l.IsReconciled,
			"rowVersion":       l.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) reconcileBankAccount(w http.ResponseWriter, r *http.Request) {
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
	matches, err := s.repo.ReconcileBankAccountAndEmit(r.Context(), scope, bankID, actor, httpx.CorrelationUUID(r))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("rekening bank tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if matches == nil {
		matches = []repository.ReconcileMatch{}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"bankAccountId": bankID,
		"matchedCount":  len(matches),
		"matches":       matches,
	})
}

