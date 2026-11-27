// Package httpapi implements tax-service REST handlers using exact *big.Rat VAT
// computation, e-Faktur reporting transitions, idempotency hash checks, and RBAC.
package httpapi

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
	"github.com/sinergiinvestasiproperti/erp/services/tax-service/internal/taxdb"
)

// Store abstracts tax repository operations for handler unit tests.
type Store interface {
	CreateTaxCode(ctx context.Context, scope []uuid.UUID, arg taxdb.CreateTaxCodeParams) (taxdb.CreateTaxCodeRow, error)
	GetTaxCode(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (taxdb.TaxTaxCode, error)
	ListTaxCodesWithCount(ctx context.Context, scope []uuid.UUID, limit, offset int32) ([]taxdb.ListTaxCodesRow, int64, error)
	CreateTransactionAndEmit(ctx context.Context, scope []uuid.UUID, txn taxdb.CreateTaxTransactionParams, ev taxdb.InsertOutboxParams) (taxdb.CreateTaxTransactionRow, error)
	GetByID(ctx context.Context, scope []uuid.UUID, id uuid.UUID) (taxdb.TaxTaxTransaction, error)
	ListTxnWithCount(ctx context.Context, scope []uuid.UUID, status string, limit, offset int32) ([]taxdb.ListTaxTransactionsRow, int64, error)
	ReportEfakturAndEmit(ctx context.Context, scope []uuid.UUID, arg taxdb.ReportTaxTransactionParams, ev taxdb.InsertOutboxParams) (int32, error)
}

type Server struct {
	repo Store
	idem *httpx.IdemStore
}

func NewServer(repo Store) *Server {
	return &Server{repo: repo, idem: httpx.NewIdemStore()}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /tax-codes", s.createCode)
	mux.HandleFunc("GET /tax-codes", s.listCodes)
	mux.HandleFunc("POST /tax-invoices", s.createTxn)
	mux.HandleFunc("GET /tax-invoices", s.listTxn)
	mux.HandleFunc("GET /tax-invoices/export/efaktur", s.exportEfakturCSV)
	mux.HandleFunc("GET /tax-invoices/spt-ppn", s.sptPPN)
	mux.HandleFunc("GET /tax-invoices/{id}", s.getTxn)
	mux.HandleFunc("POST /tax-invoices/{id}/report", s.reportEfaktur)
	mux.HandleFunc("POST /withholding-tax/calculate", s.calculateWithholdingTax)
}

type taxCodeReq struct {
	Code        string `json:"code"`
	Rate        string `json:"rate"`
	TaxType     string `json:"taxType"`
	GLAccountID string `json:"glAccountId"`
}

type taxCodeDTO struct {
	ID          string `json:"id"`
	Code        string `json:"code,omitempty"`
	Rate        string `json:"rate,omitempty"`
	TaxType     string `json:"taxType,omitempty"`
	GLAccountID string `json:"glAccountId,omitempty"`
	RowVersion  int32  `json:"rowVersion"`
}

type taxInvoiceReq struct {
	SourceDocumentRef string `json:"sourceDocumentRef"`
	TaxCodeID         string `json:"taxCodeId"`
	BaseAmount        string `json:"baseAmount"`
	TaxAmount         string `json:"taxAmount"`
	EfakturNo         string `json:"efakturNo"`
}

type reportReq struct {
	EfakturNo string `json:"efakturNo"`
	NTTE      string `json:"ntte,omitempty"`
}

type taxInvoiceDTO struct {
	ID                string `json:"id"`
	TaxCodeID         string `json:"taxCodeId,omitempty"`
	SourceDocumentRef string `json:"sourceDocumentRef,omitempty"`
	BaseAmount        string `json:"baseAmount,omitempty"`
	TaxAmount         string `json:"taxAmount,omitempty"`
	EfakturNo         string `json:"efakturNo,omitempty"`
	EfakturStatus     string `json:"efakturStatus,omitempty"`
	RowVersion        int32  `json:"rowVersion"`
}

func (s *Server) createCode(w http.ResponseWriter, r *http.Request) {
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
	if err := httpx.RequirePermission(r, "tax:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req taxCodeReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.TaxType) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("code dan taxType wajib diisi"))
		return
	}
	rateNum, err := platform.ParseNumeric(req.Rate)
	if err != nil || platform.NumericToRat(rateNum).Sign() < 0 {
		httpx.WriteError(w, r, httpx.BadRequest("rate tidak valid"))
		return
	}
	var glPtr *uuid.UUID
	if strings.TrimSpace(req.GLAccountID) != "" {
		gl, err := httpx.ParseUUIDParam("glAccountId", req.GLAccountID)
		if err != nil {
			httpx.WriteError(w, r, err)
			return
		}
		glPtr = &gl
	}
	created, err := s.repo.CreateTaxCode(r.Context(), scope, taxdb.CreateTaxCodeParams{
		OrganizationID: scope[0],
		Code:           strings.TrimSpace(req.Code),
		Rate:           rateNum,
		TaxType:        strings.TrimSpace(req.TaxType),
		GlAccountID:    glPtr,
		CreatedBy:      user,
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "kode pajak sudah dipakai"))
		return
	}
	dto := taxCodeDTO{
		ID:         created.ID.String(),
		Code:       strings.TrimSpace(req.Code),
		Rate:       platform.FormatNumeric(rateNum, 4),
		TaxType:    strings.TrimSpace(req.TaxType),
		RowVersion: created.RowVersion,
	}
	if glPtr != nil {
		dto.GLAccountID = glPtr.String()
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, dto)
}

func (s *Server) listCodes(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "tax:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	rows, total, err := s.repo.ListTaxCodesWithCount(r.Context(), scope, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]taxCodeDTO, 0, len(rows))
	for _, c := range rows {
		dto := taxCodeDTO{
			ID:         c.ID.String(),
			Code:       c.Code,
			Rate:       platform.FormatNumeric(c.Rate, 4),
			TaxType:    c.TaxType,
			RowVersion: c.RowVersion,
		}
		if c.GlAccountID != nil {
			dto.GLAccountID = c.GlAccountID.String()
		}
		out = append(out, dto)
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func (s *Server) createTxn(w http.ResponseWriter, r *http.Request) {
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
	if err := httpx.RequirePermission(r, "tax:write"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req taxInvoiceReq
	idemKey, reqHash, release, done := httpx.BeginIdempotent(w, r, s.idem, scope, &req)
	if done {
		return
	}
	defer release()

	if strings.TrimSpace(req.SourceDocumentRef) == "" {
		httpx.WriteError(w, r, httpx.BadRequest("sourceDocumentRef wajib diisi"))
		return
	}
	codeID, err := httpx.ParseUUIDParam("taxCodeId", req.TaxCodeID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	baseNum, err := platform.ParseNumeric(req.BaseAmount)
	if err != nil || platform.NumericToRat(baseNum).Sign() < 0 {
		httpx.WriteError(w, r, httpx.BadRequest("baseAmount tidak valid"))
		return
	}

	var taxNum pgtype.Numeric
	if strings.TrimSpace(req.TaxAmount) == "" {
		tc, err := s.repo.GetTaxCode(r.Context(), scope, codeID)
		if err != nil {
			httpx.WriteError(w, r, httpx.MapDBError(err, ""))
			return
		}
		taxNum, err = computeTaxFromRate(baseNum, tc.Rate)
		if err != nil {
			httpx.WriteError(w, r, httpx.Internal("tax calculation failed"))
			return
		}
	} else {
		taxNum, err = platform.ParseNumeric(req.TaxAmount)
		if err != nil || platform.NumericToRat(taxNum).Sign() < 0 {
			httpx.WriteError(w, r, httpx.BadRequest("taxAmount tidak valid"))
			return
		}
	}

	var efaktur *string
	if e := strings.TrimSpace(req.EfakturNo); e != "" {
		efaktur = &e
	}
	draft := "Draft"
	txn := taxdb.CreateTaxTransactionParams{
		OrganizationID:    scope[0],
		SourceDocumentRef: strings.TrimSpace(req.SourceDocumentRef),
		TaxCodeID:         codeID,
		BaseAmount:        baseNum,
		TaxAmount:         taxNum,
		EfakturNo:         efaktur,
		EfakturStatus:     &draft,
		CreatedBy:         user,
	}
	baseStr := platform.FormatNumeric(baseNum, 4)
	taxStr := platform.FormatNumeric(taxNum, 4)
	payload, _ := json.Marshal(map[string]any{
		"sourceDocumentRef": txn.SourceDocumentRef,
		"taxCodeId":         codeID.String(),
		"efakturNo":         strings.TrimSpace(req.EfakturNo),
		"baseAmount":        baseStr,
		"taxAmount":         taxStr,
		"status":            "Draft",
	})
	ev := taxdb.InsertOutboxParams{
		AggregateType:  "TaxTransaction",
		EventType:      "tax.invoice.issued",
		Payload:        payload,
		OrganizationID: scope[0],
		CorrelationID:  httpx.CorrelationUUID(r),
	}
	created, err := s.repo.CreateTransactionAndEmit(r.Context(), scope, txn, ev)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, "dokumen pajak duplikat"))
		return
	}
	dto := taxInvoiceDTO{
		ID:                created.ID.String(),
		TaxCodeID:         codeID.String(),
		SourceDocumentRef: txn.SourceDocumentRef,
		BaseAmount:        baseStr,
		TaxAmount:         taxStr,
		EfakturNo:         strings.TrimSpace(req.EfakturNo),
		EfakturStatus:     draft,
		RowVersion:        created.RowVersion,
	}
	httpx.SaveIDResponse(w, s.idem, scope, idemKey, reqHash, http.StatusCreated, created.RowVersion, dto)
}

func (s *Server) getTxn(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "tax:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	id, err := httpx.ParseUUIDParam("id", r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	t, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, t.RowVersion, taxInvoiceDTO{
		ID:                t.ID.String(),
		TaxCodeID:         t.TaxCodeID.String(),
		SourceDocumentRef: t.SourceDocumentRef,
		BaseAmount:        platform.FormatNumeric(t.BaseAmount, 4),
		TaxAmount:         platform.FormatNumeric(t.TaxAmount, 4),
		EfakturNo:         deref(t.EfakturNo),
		EfakturStatus:     deref(t.EfakturStatus),
		RowVersion:        t.RowVersion,
	})
}

func (s *Server) listTxn(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "tax:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	page, size, limit, offset := httpx.PaginateParams(r)
	statusFilter := strings.TrimSpace(r.URL.Query().Get("efakturStatus"))
	rows, total, err := s.repo.ListTxnWithCount(r.Context(), scope, statusFilter, limit, offset)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	out := make([]taxInvoiceDTO, 0, len(rows))
	for _, t := range rows {
		out = append(out, taxInvoiceDTO{
			ID:                t.ID.String(),
			TaxCodeID:         t.TaxCodeID.String(),
			SourceDocumentRef: t.SourceDocumentRef,
			BaseAmount:        platform.FormatNumeric(t.BaseAmount, 4),
			TaxAmount:         platform.FormatNumeric(t.TaxAmount, 4),
			EfakturNo:         deref(t.EfakturNo),
			EfakturStatus:     deref(t.EfakturStatus),
			RowVersion:        t.RowVersion,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, httpx.ListEnvelope{
		Data: out,
		Page: httpx.NewPageMeta(int(page), int(size), int(total)),
	})
}

func requireTaxApprove(r *http.Request) error {
	if err := httpx.RequirePermission(r, "tax.approve"); err != nil {
		if httpx.RequirePermission(r, "tax:approve") == nil {
			return nil
		}
		return err
	}
	return nil
}

func (s *Server) reportEfaktur(w http.ResponseWriter, r *http.Request) {
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
	if err := requireTaxApprove(r); err != nil {
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
	var req reportReq
	if err := httpx.DecodeOptionalJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	t, err := s.repo.GetByID(r.Context(), scope, id)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	if t.CreatedBy != uuid.Nil && t.CreatedBy == user && !httpx.HasRole(r.Context(), "SYSTEM_ADMIN") {
		httpx.WriteError(w, r, httpx.Forbidden("maker-checker separation of duties: creator cannot report own tax invoice"))
		return
	}
	if strings.EqualFold(deref(t.EfakturStatus), "Reported") {
		httpx.WriteError(w, r, httpx.Conflict("ALREADY_REPORTED", "faktur pajak sudah pernah dilaporkan"))
		return
	}
	efakturNo := strings.TrimSpace(req.EfakturNo)
	if efakturNo == "" {
		efakturNo = deref(t.EfakturNo)
	}
	if efakturNo == "" {
		httpx.WriteError(w, r, httpx.BadRequest("efakturNo wajib diisi untuk pelaporan e-Faktur"))
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"sourceDocumentRef": t.SourceDocumentRef,
		"efakturNo":         efakturNo,
		"ntte":              strings.TrimSpace(req.NTTE),
		"baseAmount":        platform.FormatNumeric(t.BaseAmount, 4),
		"taxAmount":         platform.FormatNumeric(t.TaxAmount, 4),
		"efakturStatus":     "Reported",
	})
	newVer, err := s.repo.ReportEfakturAndEmit(r.Context(), scope, taxdb.ReportTaxTransactionParams{
		ID:         id,
		RowVersion: rowVer,
		UpdatedBy:  &user,
		EfakturNo:  strings.TrimSpace(req.EfakturNo),
	}, taxdb.InsertOutboxParams{
		AggregateType:  "TaxTransaction",
		AggregateID:    id,
		EventType:      "tax.efaktur.reported",
		Payload:        payload,
		OrganizationID: t.OrganizationID,
		CorrelationID:  httpx.CorrelationUUID(r),
	})
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	httpx.WriteIDResponse(w, http.StatusOK, newVer, taxInvoiceDTO{
		ID:                t.ID.String(),
		TaxCodeID:         t.TaxCodeID.String(),
		SourceDocumentRef: t.SourceDocumentRef,
		BaseAmount:        platform.FormatNumeric(t.BaseAmount, 4),
		TaxAmount:         platform.FormatNumeric(t.TaxAmount, 4),
		EfakturNo:         efakturNo,
		EfakturStatus:     "Reported",
		RowVersion:        newVer,
	})
}

func computeTaxFromRate(base, ratePct pgtype.Numeric) (pgtype.Numeric, error) {
	b := platform.NumericToRat(base)
	r := platform.NumericToRat(ratePct)
	hundred := big.NewRat(100, 1)
	tax := new(big.Rat).Mul(b, r)
	tax.Quo(tax, hundred)
	return platform.RatToNumeric(tax, 4)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func sanitizeCSVCell(v string) string {
	if v == "" {
		return v
	}
	if _, ok := new(big.Rat).SetString(v); ok {
		return v
	}
	switch v[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + v
	}
	return v
}

// exportEfakturCSV exports DJP e-Faktur CSV (FK header + OF item lines) safely neutralized against CSV formula injection.
func (s *Server) exportEfakturCSV(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "tax:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	statusFilter := strings.TrimSpace(r.URL.Query().Get("efakturStatus"))
	rows, _, err := s.repo.ListTxnWithCount(r.Context(), scope, statusFilter, 10000, 0)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}

	now := time.Now().UTC()
	var buf bytes.Buffer
	cw := csv.NewWriter(&buf)
	_ = cw.Write([]string{"FK", "KD_JENIS_TRANSAKSI", "FG_PENGGANTI", "NOMOR_FAKTUR", "MASA_PAJAK", "TAHUN_PAJAK", "TANGGAL_FAKTUR", "REFERENSI", "JUMLAH_DPP", "JUMLAH_PPN", "STATUS_EFAKTUR"})
	_ = cw.Write([]string{"OF", "KODE_OBJEK", "NAMA", "HARGA_SATUAN", "JUMLAH_BARANG", "HARGA_TOTAL", "DISKON", "DPP", "PPN", "TARIF_PPNBM", "PPNBM"})

	for _, t := range rows {
		dpp := platform.FormatNumeric(t.BaseAmount, 4)
		ppn := platform.FormatNumeric(t.TaxAmount, 4)
		efNo := deref(t.EfakturNo)
		efStatus := deref(t.EfakturStatus)
		_ = cw.Write([]string{
			"FK",
			"01",
			"0",
			sanitizeCSVCell(efNo),
			now.Format("01"),
			now.Format("2006"),
			now.Format("02/01/2006"),
			sanitizeCSVCell(t.SourceDocumentRef),
			dpp,
			ppn,
			sanitizeCSVCell(efStatus),
		})
		_ = cw.Write([]string{
			"OF",
			sanitizeCSVCell(t.TaxCodeID.String()),
			sanitizeCSVCell("Supply " + t.SourceDocumentRef),
			dpp,
			"1",
			dpp,
			"0",
			dpp,
			ppn,
			"0",
			"0",
		})
	}
	cw.Flush()

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="efaktur-`+now.Format("2006-01")+`.csv"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

// sptPPN aggregates SPT Masa PPN (Pajak Keluaran, Pajak Masukan, Kurang/Lebih Bayar) using *big.Rat.
func (s *Server) sptPPN(w http.ResponseWriter, r *http.Request) {
	scope, err := httpx.RequireScope(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "tax:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	codes, _, err := s.repo.ListTaxCodesWithCount(r.Context(), scope, 1000, 0)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}
	codeTypeByID := make(map[uuid.UUID]string, len(codes))
	for _, c := range codes {
		codeTypeByID[c.ID] = strings.ToUpper(strings.TrimSpace(c.TaxType + " " + c.Code))
	}

	rows, _, err := s.repo.ListTxnWithCount(r.Context(), scope, "", 10000, 0)
	if err != nil {
		httpx.WriteError(w, r, httpx.MapDBError(err, ""))
		return
	}

	keluaran := new(big.Rat)
	masukan := new(big.Rat)
	for _, t := range rows {
		amt := platform.NumericToRat(t.TaxAmount)
		if amt == nil {
			continue
		}
		meta := codeTypeByID[t.TaxCodeID]
		refUpper := strings.ToUpper(strings.TrimSpace(t.SourceDocumentRef))
		isInput := strings.Contains(meta, "INPUT") || strings.Contains(meta, "MASUKAN") || strings.Contains(meta, "VAT_IN") ||
			strings.HasPrefix(refUpper, "AP-") || strings.HasPrefix(refUpper, "PI-")
		if isInput {
			masukan.Add(masukan, amt)
		} else {
			keluaran.Add(keluaran, amt)
		}
	}
	netPayable := new(big.Rat).Sub(keluaran, masukan)

	period := strings.TrimSpace(r.URL.Query().Get("period"))
	if period == "" {
		period = time.Now().UTC().Format("2006-01")
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"period":          period,
		"pajakKeluaran":   keluaran.FloatString(4),
		"pajakMasukan":    masukan.FloatString(4),
		"kurangLebihBayar": netPayable.FloatString(4),
		"invoiceCount":    len(rows),
	})
}

type withholdingCalcReq struct {
	TaxType     string `json:"taxType"`               // PPH21, PPH23, PPH26, PPH4_2
	Category    string `json:"category,omitempty"`    // e.g. SERVICES, DIVIDEND, RENT, CONSTRUCTION, UMKM
	GrossAmount string `json:"grossAmount"`
	HasNPWP     *bool  `json:"hasNpwp,omitempty"`     // default true; false triggers statutory non-NPWP surcharge
	CustomRate  string `json:"customRate,omitempty"`  // optional percentage override (e.g. "2.00")
}

// calculateWithholdingTax computes Indonesian PPh 21 / 23 / 26 / 4(2) withholding via *big.Rat.
func (s *Server) calculateWithholdingTax(w http.ResponseWriter, r *http.Request) {
	if _, err := httpx.RequireScope(r); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := httpx.RequirePermission(r, "tax:read"); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var req withholdingCalcReq
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	grossNum, err := platform.ParseNumeric(req.GrossAmount)
	if err != nil {
		httpx.WriteError(w, r, httpx.BadRequest("grossAmount tidak valid"))
		return
	}
	gross := platform.NumericToRat(grossNum)
	if gross == nil || gross.Sign() <= 0 {
		httpx.WriteError(w, r, httpx.Unprocessable("INVALID_AMOUNT", "grossAmount harus > 0"))
		return
	}

	hasNPWP := true
	if req.HasNPWP != nil {
		hasNPWP = *req.HasNPWP
	}

	taxType := strings.ToUpper(strings.TrimSpace(req.TaxType))
	cat := strings.ToUpper(strings.TrimSpace(req.Category))
	var baseRatePct *big.Rat
	surchargeMultiplier := big.NewRat(1, 1)

	if strings.TrimSpace(req.CustomRate) != "" {
		crNum, err := platform.ParseNumeric(req.CustomRate)
		if err != nil {
			httpx.WriteError(w, r, httpx.BadRequest("customRate tidak valid"))
			return
		}
		baseRatePct = platform.NumericToRat(crNum)
		if baseRatePct == nil || baseRatePct.Sign() < 0 || baseRatePct.Cmp(big.NewRat(100, 1)) > 0 {
			httpx.WriteError(w, r, httpx.Unprocessable("INVALID_RATE", "customRate harus antara 0 dan 100"))
			return
		}
	} else {
		switch taxType {
		case "PPH21", "PPH_21":
			taxType = "PPH21"
			baseRatePct = big.NewRat(5, 1) // Tier 1 statutory 5%
			if !hasNPWP {
				surchargeMultiplier = big.NewRat(120, 100) // +20% non-NPWP surcharge
			}
		case "PPH23", "PPH_23":
			taxType = "PPH23"
			switch cat {
			case "DIVIDEND", "ROYALTY", "INTEREST", "PRIZE":
				baseRatePct = big.NewRat(15, 1)
			default:
				baseRatePct = big.NewRat(2, 1) // Services / rent of equipment
			}
			if !hasNPWP {
				surchargeMultiplier = big.NewRat(2, 1) // +100% non-NPWP surcharge
			}
		case "PPH26", "PPH_26":
			taxType = "PPH26"
			baseRatePct = big.NewRat(20, 1) // Non-resident 20%
		case "PPH4_2", "PPH4(2)", "PPH_4_2":
			taxType = "PPH4_2"
			switch cat {
			case "CONSTRUCTION":
				baseRatePct = big.NewRat(2, 1)
			case "UMKM":
				baseRatePct = big.NewRat(1, 2) // 0.5%
			default:
				baseRatePct = big.NewRat(10, 1) // Property rent 10%
			}
		default:
			httpx.WriteError(w, r, httpx.BadRequest("taxType harus salah satu dari PPH21, PPH23, PPH26, PPH4_2"))
			return
		}
	}

	effectiveRatePct := new(big.Rat).Mul(baseRatePct, surchargeMultiplier)
	withheld := new(big.Rat).Mul(gross, effectiveRatePct)
	withheld.Quo(withheld, big.NewRat(100, 1))
	netAmount := new(big.Rat).Sub(gross, withheld)

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"taxType":             taxType,
		"category":            cat,
		"hasNpwp":             hasNPWP,
		"grossAmount":         gross.FloatString(4),
		"baseRatePercent":     baseRatePct.FloatString(4),
		"effectiveRatePercent": effectiveRatePct.FloatString(4),
		"withholdingAmount":   withheld.FloatString(4),
		"netAmount":           netAmount.FloatString(4),
	})
}
