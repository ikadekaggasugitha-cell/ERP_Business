// Package httpapi wires REST endpoints for treasury-cashbank-service
// (/bank-accounts, /payments, /bank-statement-lines, /reconcile).
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

type Store interface {
	CreateBankAccount(ctx context.Context, scope []uuid.UUID, arg treasurydb.CreateBankAccountParams) (treasurydb.CreateBankAccountRow, error)
	GetBankAccount(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (treasurydb.TrsBankAccount, error)
	ListBanksWithCount(ctx context.Context, scope []uuid.UUID, arg treasurydb.ListBankAccountsParams) ([]treasurydb.ListBankAccountsRow, int64, error)
	CreatePayment(ctx context.Context, scope []uuid.UUID, arg treasurydb.CreatePaymentParams, allocs []treasurydb.CreatePaymentAllocationParams) (treasurydb.CreatePaymentRow, error)
	GetWithAllocations(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (treasurydb.TrsPayment, []treasurydb.ListPaymentAllocationsRow, error)
	ListWithCount(ctx context.Context, scope []uuid.UUID, arg treasurydb.ListPaymentsParams) ([]treasurydb.ListPaymentsRow, int64, error)
	UpdateDraft(ctx context.Context, scope []uuid.UUID, arg treasurydb.UpdateDraftPaymentParams) (int32, error)
	DeleteDraft(ctx context.Context, scope []uuid.UUID, arg treasurydb.DeleteDraftPaymentParams) (int64, error)
	SettleAndEmit(ctx context.Context, scope []uuid.UUID, settle treasurydb.SettlePaymentParams, events ...treasurydb.InsertOutboxParams) (int32, error)
	CreateStatementLine(ctx context.Context, scope []uuid.UUID, arg treasurydb.CreateBankStatementLineParams) (treasurydb.CreateBankStatementLineRow, error)
	ListStatementLinesWithCount(ctx context.Context, scope []uuid.UUID, arg treasurydb.ListBankStatementLinesParams) ([]treasurydb.ListBankStatementLinesRow, int64, error)
	ReconcileBankAccountAndEmit(ctx context.Context, scope []uuid.UUID, bankAccountID, actorID, correlationID uuid.UUID) ([]repository.ReconcileMatch, error)
}

type Server struct {
	repo Store
	idem *httpx.IdemStore
}

func NewServer(repo Store) *Server {
	return &Server{repo: repo, idem: httpx.NewIdemStore()}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /bank-accounts", s.createBankAccount)
	mux.HandleFunc("GET /bank-accounts", s.listBankAccounts)
	mux.HandleFunc("GET /bank-accounts/{id}", s.getBankAccount)
	mux.HandleFunc("POST /bank-accounts/{id}/reconcile", s.reconcileBankAccount)

	mux.HandleFunc("POST /payments", s.createPayment)
	mux.HandleFunc("GET /payments", s.listPayments)
	mux.HandleFunc("GET /payments/{id}", s.getPayment)
	mux.HandleFunc("PUT /payments/{id}", s.updateDraftPayment)
	mux.HandleFunc("DELETE /payments/{id}", s.deleteDraftPayment)
	mux.HandleFunc("POST /payments/{id}/settle", s.settlePayment)

	mux.HandleFunc("POST /bank-statement-lines", s.createStatementLine)
	mux.HandleFunc("GET /bank-statement-lines", s.listStatementLines)
}

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

type allocReq struct {
	TargetDocumentRef string `json:"targetDocumentRef"`
	AllocatedAmount   string `json:"allocatedAmount"`
}

type paymentReq struct {
	OrganizationID uuid.UUID  `json:"organizationId"`
	PaymentNo      string     `json:"paymentNo"`
	PaymentType    string     `json:"paymentType"`
	BankAccountID  uuid.UUID  `json:"bankAccountId"`
	PartyID        *uuid.UUID `json:"partyId"`
	Amount         string     `json:"amount"`
	PaymentDate    string     `json:"paymentDate"`
	ExternalRef    *string    `json:"externalRef"`
	Allocations    []allocReq `json:"allocations"`
}

func (s *Server) createPayment(w http.ResponseWriter, r *http.Request) {
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
	var req paymentReq
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
	payAmt, payRat, payDate, allocParams, err := validatePaymentFields(req, true)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	_ = payRat
	res, err := s.repo.CreatePayment(r.Context(), scope, treasurydb.CreatePaymentParams{
		OrganizationID: org,
		PaymentNo:      strings.TrimSpace(req.PaymentNo),
		PaymentType:    req.PaymentType,
		BankAccountID:  req.BankAccountID,
		PartyID:        req.PartyID,
		Amount:         payAmt,
		PaymentDate:    payDate,
		ExternalRef:    req.ExternalRef,
		CorrelationID:  httpx.CorrelationUUID(r),
		CreatedBy:      actor,
	}, allocParams)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("rekening bank tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, "nomor pembayaran sudah dipakai"))
		return
	}
	httpx.SaveIDResponse(w, s.idem, scope, key, hash, http.StatusCreated, res.RowVersion, map[string]any{
		"id":         res.ID,
		"paymentNo":  strings.TrimSpace(req.PaymentNo),
		"status":     res.Status,
		"rowVersion": res.RowVersion,
	})
}

func (s *Server) listPayments(w http.ResponseWriter, r *http.Request) {
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
	var statusPtr *string
	if st := strings.TrimSpace(r.URL.Query().Get("status")); st != "" {
		statusPtr = &st
	}
	var bankIDPtr *uuid.UUID
	if rawBank := strings.TrimSpace(r.URL.Query().Get("bankAccountId")); rawBank != "" {
		bid, err := httpx.ParseUUIDParam("bankAccountId", rawBank)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		bankIDPtr = &bid
	}
	rows, total, err := s.repo.ListWithCount(r.Context(), scope, treasurydb.ListPaymentsParams{
		Limit:         limit,
		Offset:        offset,
		Status:        statusPtr,
		BankAccountID: bankIDPtr,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, p := range rows {
		out = append(out, map[string]any{
			"id":             p.ID,
			"organizationId": p.OrganizationID,
			"paymentNo":      p.PaymentNo,
			"paymentType":    p.PaymentType,
			"bankAccountId":  p.BankAccountID,
			"partyId":        p.PartyID,
			"amount":         platform.FormatNumeric(p.Amount, 4),
			"paymentDate":    platform.FormatDate(p.PaymentDate),
			"status":         p.Status,
			"externalRef":    p.ExternalRef,
			"rowVersion":     p.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) getPayment(w http.ResponseWriter, r *http.Request) {
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
	p, allocs, err := s.repo.GetWithAllocations(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("pembayaran tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	allocOut := make([]map[string]any, 0, len(allocs))
	for _, a := range allocs {
		allocOut = append(allocOut, map[string]any{
			"id":                a.ID,
			"targetDocumentRef": a.TargetDocumentRef,
			"allocatedAmount":   platform.FormatNumeric(a.AllocatedAmount, 4),
			"rowVersion":        a.RowVersion,
		})
	}
	httpx.WriteIDResponse(w, http.StatusOK, p.RowVersion, map[string]any{
		"id":             p.ID,
		"organizationId": p.OrganizationID,
		"paymentNo":      p.PaymentNo,
		"paymentType":    p.PaymentType,
		"bankAccountId":  p.BankAccountID,
		"partyId":        p.PartyID,
		"amount":         platform.FormatNumeric(p.Amount, 4),
		"paymentDate":    platform.FormatDate(p.PaymentDate),
		"status":         p.Status,
		"externalRef":    p.ExternalRef,
		"correlationId":  p.CorrelationID,
		"rowVersion":     p.RowVersion,
		"allocations":    allocOut,
	})
}

func (s *Server) updateDraftPayment(w http.ResponseWriter, r *http.Request) {
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
	var req paymentReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	payAmt, _, payDate, _, err := validatePaymentFields(req, false)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	existing, _, err := s.repo.GetWithAllocations(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("pembayaran tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if existing.Status != "Draft" || existing.RowVersion != ver {
		httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "pembayaran bukan Draft atau versi bentrok"))
		return
	}
	newVer, err := s.repo.UpdateDraft(r.Context(), scope, treasurydb.UpdateDraftPaymentParams{
		ID:            id,
		RowVersion:    ver,
		PaymentType:   req.PaymentType,
		BankAccountID: req.BankAccountID,
		PartyID:       req.PartyID,
		Amount:        payAmt,
		PaymentDate:   payDate,
		ExternalRef:   req.ExternalRef,
		UpdatedBy:     &actor,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "pembayaran berubah secara bersamaan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, newVer, map[string]any{
		"id":         id,
		"status":     "Draft",
		"rowVersion": newVer,
	})
}

func (s *Server) deleteDraftPayment(w http.ResponseWriter, r *http.Request) {
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
	existing, _, err := s.repo.GetWithAllocations(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("pembayaran tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if existing.Status != "Draft" || existing.RowVersion != ver {
		httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "hanya pembayaran Draft dengan versi cocok yang dapat dihapus"))
		return
	}
	n, err := s.repo.DeleteDraft(r.Context(), scope, treasurydb.DeleteDraftPaymentParams{
		ID:         id,
		RowVersion: ver,
		UpdatedBy:  &actor,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if n == 0 {
		httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "pembayaran berubah secara bersamaan"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) settlePayment(w http.ResponseWriter, r *http.Request) {
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
	p, _, err := s.repo.GetWithAllocations(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("pembayaran tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if p.Status != "Draft" || p.RowVersion != ver {
		httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "pembayaran bukan Draft atau row_version sudah berubah"))
		return
	}
	amtStr := platform.FormatNumeric(p.Amount, 4)
	payload, _ := json.Marshal(map[string]any{
		"paymentId":     p.ID,
		"paymentNo":     p.PaymentNo,
		"paymentType":   p.PaymentType,
		"bankAccountId": p.BankAccountID,
		"partyId":       p.PartyID,
		"amount":        amtStr,
		"paymentDate":   platform.FormatDate(p.PaymentDate),
		"externalRef":   p.ExternalRef,
	})
	corr := httpx.CorrelationUUID(r)
	evTreasury := treasurydb.InsertOutboxParams{
		AggregateType:  "payment",
		AggregateID:    p.ID,
		EventType:      "treasury.payment.settled",
		Payload:        payload,
		OrganizationID: p.OrganizationID,
		CorrelationID:  corr,
	}
	evCashbank := treasurydb.InsertOutboxParams{
		AggregateType:  "payment",
		AggregateID:    p.ID,
		EventType:      "cashbank.payment.settled",
		Payload:        payload,
		OrganizationID: p.OrganizationID,
		CorrelationID:  corr,
	}
	newVer, err := s.repo.SettleAndEmit(r.Context(), scope, treasurydb.SettlePaymentParams{
		ID:         id,
		RowVersion: ver,
		UpdatedBy:  &actor,
	}, evTreasury, evCashbank)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "pembayaran berubah secara bersamaan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, newVer, map[string]any{
		"id":         p.ID,
		"paymentNo":  p.PaymentNo,
		"status":     "Settled",
		"rowVersion": newVer,
	})
}

type companionStatementLineItem struct {
	BankAccountID *uuid.UUID `json:"bankAccountId,omitempty"`
	StatementDate string     `json:"statementDate,omitempty"`
	Date          string     `json:"date,omitempty"`
	Description   string     `json:"description"`
	Amount        any        `json:"amount,omitempty"`
	Debit         any        `json:"debit,omitempty"`
	Credit        any        `json:"credit,omitempty"`
	Balance       any        `json:"balance,omitempty"`
	ExternalRef   *string    `json:"externalRef,omitempty"`
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

		createdIDs := make([]uuid.UUID, 0, len(parsedLines))
		var lastVer int32 = 1
		for _, pl := range parsedLines {
			row, cerr := s.repo.CreateStatementLine(r.Context(), scope, treasurydb.CreateBankStatementLineParams{
				OrganizationID: org,
				BankAccountID:  pl.bankID,
				StatementDate:  pl.date,
				Description:    pl.desc,
				Amount:         pl.amt,
				ExternalRef:    pl.extRef,
				CreatedBy:      actor,
			})
			if cerr != nil {
				if errors.Is(cerr, pgx.ErrNoRows) {
					httpx.WriteError(w, r, httpx.NotFound("rekening bank tidak ditemukan"))
					return
				}
				httpx.WriteError(w, r, httpx.MapDBError(cerr, ""))
				return
			}
			createdIDs = append(createdIDs, row.ID)
			lastVer = row.RowVersion
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

func anyToDecimalString(v any) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(val)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case json.Number:
		return val.String()
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", val))
	}
}

func resolveCompanionLineAmount(item companionStatementLineItem) (pgtype.Numeric, error) {
	if raw := anyToDecimalString(item.Amount); raw != "" {
		n, err := platform.ParseNumeric(raw)
		if err != nil {
			return pgtype.Numeric{}, err
		}
		if rVal := platform.NumericToRat(n); rVal != nil && rVal.Sign() != 0 {
			return n, nil
		}
	}
	cr := new(big.Rat)
	dr := new(big.Rat)
	if rawCr := anyToDecimalString(item.Credit); rawCr != "" {
		n, err := platform.ParseNumeric(rawCr)
		if err != nil {
			return pgtype.Numeric{}, err
		}
		if rVal := platform.NumericToRat(n); rVal != nil {
			cr.Set(rVal)
		}
	}
	if rawDr := anyToDecimalString(item.Debit); rawDr != "" {
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

func validatePaymentFields(req paymentReq, requireNo bool) (pgtype.Numeric, *big.Rat, pgtype.Date, []treasurydb.CreatePaymentAllocationParams, error) {
	if requireNo && strings.TrimSpace(req.PaymentNo) == "" {
		return pgtype.Numeric{}, nil, pgtype.Date{}, nil, httpx.BadRequest("paymentNo wajib diisi")
	}
	if req.PaymentType != "Outgoing" && req.PaymentType != "Incoming" {
		return pgtype.Numeric{}, nil, pgtype.Date{}, nil, httpx.BadRequest("paymentType harus Outgoing atau Incoming")
	}
	if req.BankAccountID == uuid.Nil {
		return pgtype.Numeric{}, nil, pgtype.Date{}, nil, httpx.BadRequest("bankAccountId wajib diisi")
	}
	amt, err := platform.ParseNumeric(req.Amount)
	if err != nil {
		return pgtype.Numeric{}, nil, pgtype.Date{}, nil, httpx.BadRequest("amount tidak valid")
	}
	payRat := platform.NumericToRat(amt)
	if payRat == nil || payRat.Sign() <= 0 {
		return pgtype.Numeric{}, nil, pgtype.Date{}, nil, httpx.BadRequest("amount harus > 0")
	}
	payDate := pgtype.Date{Time: time.Now().UTC(), Valid: true}
	if strings.TrimSpace(req.PaymentDate) != "" {
		d, err := platform.ParseDate(req.PaymentDate)
		if err != nil {
			return pgtype.Numeric{}, nil, pgtype.Date{}, nil, httpx.BadRequest("paymentDate harus YYYY-MM-DD")
		}
		payDate = d
	}
	allocParams := make([]treasurydb.CreatePaymentAllocationParams, 0, len(req.Allocations))
	allocTotal := new(big.Rat)
	for _, a := range req.Allocations {
		if strings.TrimSpace(a.TargetDocumentRef) == "" {
			return pgtype.Numeric{}, nil, pgtype.Date{}, nil, httpx.BadRequest("targetDocumentRef alokasi wajib diisi")
		}
		aNum, err := platform.ParseNumeric(a.AllocatedAmount)
		if err != nil {
			return pgtype.Numeric{}, nil, pgtype.Date{}, nil, httpx.BadRequest("allocatedAmount tidak valid")
		}
		aRat := platform.NumericToRat(aNum)
		if aRat == nil || aRat.Sign() <= 0 {
			return pgtype.Numeric{}, nil, pgtype.Date{}, nil, httpx.BadRequest("allocatedAmount harus > 0")
		}
		allocTotal.Add(allocTotal, aRat)
		allocParams = append(allocParams, treasurydb.CreatePaymentAllocationParams{
			TargetDocumentRef: strings.TrimSpace(a.TargetDocumentRef),
			AllocatedAmount:   aNum,
		})
	}
	if allocTotal.Cmp(payRat) > 0 {
		return pgtype.Numeric{}, nil, pgtype.Date{}, nil, httpx.BadRequest("total alokasi melebihi amount pembayaran")
	}
	return amt, payRat, payDate, allocParams, nil
}

func pickOrg(candidate uuid.UUID, scope []uuid.UUID) (uuid.UUID, error) {
	if len(scope) == 0 || scope[0] == uuid.Nil {
		return uuid.Nil, httpx.Forbidden("system organization cannot be used as active tenant")
	}
	if candidate == uuid.Nil {
		return scope[0], nil
	}
	for _, s := range scope {
		if s == candidate && s != uuid.Nil {
			return candidate, nil
		}
	}
	return uuid.Nil, httpx.Forbidden("organizationId di luar cakupan akses")
}
