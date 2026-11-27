package httpapi

import (
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
	"github.com/sinergiinvestasiproperti/erp/services/treasury-cashbank-service/internal/treasurydb"
)

type allocReq struct {
	TargetDocumentRef string `json:"targetDocumentRef"`
	InvoiceID         string `json:"invoiceId,omitempty"`
	InvoiceType       string `json:"invoiceType,omitempty"`
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
	payAmt, payRat, payDate, allocParams, err := validatePaymentFields(req, false)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	existing, existingAllocs, err := s.repo.GetWithAllocations(r.Context(), scope, id)
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
	var updatedAllocs []treasurydb.CreatePaymentAllocationParams
	if req.Allocations != nil {
		updatedAllocs = allocParams
	} else if len(existingAllocs) > 0 {
		existingSum := new(big.Rat)
		for _, a := range existingAllocs {
			if rVal := platform.NumericToRat(a.AllocatedAmount); rVal != nil {
				existingSum.Add(existingSum, rVal)
			}
		}
		if existingSum.Cmp(payRat) > 0 {
			httpx.WriteError(w, r, httpx.BadRequest("total alokasi yang ada melebihi amount pembayaran baru"))
			return
		}
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
	}, updatedAllocs)
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

func requireTreasuryApprove(r *http.Request) error {
	if err := httpx.RequirePermission(r, "treasury.approve"); err != nil {
		if httpx.RequirePermission(r, "treasury:approve") == nil {
			return nil
		}
		return err
	}
	return nil
}

func (s *Server) settlePayment(w http.ResponseWriter, r *http.Request) {
	if err := requireTreasuryApprove(r); err != nil {
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
	p, allocs, err := s.repo.GetWithAllocations(r.Context(), scope, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, r, httpx.NotFound("pembayaran tidak ditemukan"))
			return
		}
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if p.CreatedBy != uuid.Nil && p.CreatedBy == actor && !httpx.HasRole(r.Context(), "SYSTEM_ADMIN") {
		httpx.WriteError(w, r, httpx.Forbidden("maker-checker separation of duties: creator cannot settle own payment"))
		return
	}
	if p.Status != "Draft" || p.RowVersion != ver {
		httpx.WriteError(w, r, httpx.Conflict("VERSION_CONFLICT", "pembayaran bukan Draft atau row_version sudah berubah"))
		return
	}
	invType := "AP"
	if strings.EqualFold(p.PaymentType, "Incoming") {
		invType = "AR"
	}
	amtStr := platform.FormatNumeric(p.Amount, 4)
	allocPayloads := make([]map[string]any, 0, len(allocs))
	for _, a := range allocs {
		allocPayloads = append(allocPayloads, map[string]any{
			"id":                a.ID,
			"invoiceId":         a.TargetDocumentRef,
			"targetDocumentRef": a.TargetDocumentRef,
			"invoiceType":       invType,
			"allocatedAmount":   platform.FormatNumeric(a.AllocatedAmount, 4),
		})
	}
	if len(allocPayloads) == 0 && p.ExternalRef != nil && strings.TrimSpace(*p.ExternalRef) != "" {
		ref := strings.TrimSpace(*p.ExternalRef)
		allocPayloads = append(allocPayloads, map[string]any{
			"invoiceId":         ref,
			"targetDocumentRef": ref,
			"invoiceType":       invType,
			"allocatedAmount":   amtStr,
		})
	}
	payloadMap := map[string]any{
		"paymentId":     p.ID,
		"paymentNo":     p.PaymentNo,
		"paymentType":   p.PaymentType,
		"bankAccountId": p.BankAccountID,
		"partyId":       p.PartyID,
		"amount":        amtStr,
		"paymentDate":   platform.FormatDate(p.PaymentDate),
		"externalRef":   p.ExternalRef,
		"allocations":   allocPayloads,
	}
	if bank, berr := s.repo.GetBankAccount(r.Context(), scope, p.BankAccountID); berr == nil {
		if bank.CurrencyCode != "" {
			payloadMap["currencyCode"] = bank.CurrencyCode
		}
		if bank.GlAccountID != uuid.Nil {
			payloadMap["glAccountId"] = bank.GlAccountID
		}
	}
	payload, _ := json.Marshal(payloadMap)
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
		targetRef := strings.TrimSpace(a.TargetDocumentRef)
		if targetRef == "" {
			targetRef = strings.TrimSpace(a.InvoiceID)
		}
		if targetRef == "" {
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
			TargetDocumentRef: targetRef,
			AllocatedAmount:   aNum,
		})
	}
	if allocTotal.Cmp(payRat) > 0 {
		return pgtype.Numeric{}, nil, pgtype.Date{}, nil, httpx.BadRequest("total alokasi melebihi amount pembayaran")
	}
	return amt, payRat, payDate, allocParams, nil
}

