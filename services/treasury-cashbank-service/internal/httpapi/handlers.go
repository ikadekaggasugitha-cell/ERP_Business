// Package httpapi wires REST endpoints for treasury-cashbank-service
// (/bank-accounts, /payments, /bank-statement-lines, /reconcile, /cash-position).
package httpapi

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/shared/httpx"
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
	UpdateDraft(ctx context.Context, scope []uuid.UUID, arg treasurydb.UpdateDraftPaymentParams, allocs []treasurydb.CreatePaymentAllocationParams) (int32, error)
	DeleteDraft(ctx context.Context, scope []uuid.UUID, arg treasurydb.DeleteDraftPaymentParams) (int64, error)
	SettleAndEmit(ctx context.Context, scope []uuid.UUID, settle treasurydb.SettlePaymentParams, events ...treasurydb.InsertOutboxParams) (int32, error)
	CreateStatementLine(ctx context.Context, scope []uuid.UUID, arg treasurydb.CreateBankStatementLineParams) (treasurydb.CreateBankStatementLineRow, error)
	CreateStatementLinesBatch(ctx context.Context, scope []uuid.UUID, lines []treasurydb.CreateBankStatementLineParams) ([]uuid.UUID, int32, error)
	ListStatementLinesWithCount(ctx context.Context, scope []uuid.UUID, arg treasurydb.ListBankStatementLinesParams) ([]treasurydb.ListBankStatementLinesRow, int64, error)
	ReconcileBankAccountAndEmit(ctx context.Context, scope []uuid.UUID, bankAccountID, actorID, correlationID uuid.UUID) ([]repository.ReconcileMatch, error)
	ManualReconcileAndEmit(ctx context.Context, scope []uuid.UUID, bankAccountID, statementLineID, paymentID uuid.UUID, action, reason string, actorID, correlationID uuid.UUID) error
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
	mux.HandleFunc("POST /bank-accounts/{id}/reconcile/manual", s.manualReconcileBankAccount)
	mux.HandleFunc("POST /bank-accounts/{id}/statements/import-csv", s.importBankStatementCSV)
	mux.HandleFunc("GET /cash-position", s.cashPosition)

	mux.HandleFunc("POST /payments", s.createPayment)
	mux.HandleFunc("GET /payments", s.listPayments)
	mux.HandleFunc("GET /payments/export/disbursement", s.exportDisbursement)
	mux.HandleFunc("GET /payments/{id}", s.getPayment)
	mux.HandleFunc("PUT /payments/{id}", s.updateDraftPayment)
	mux.HandleFunc("DELETE /payments/{id}", s.deleteDraftPayment)
	mux.HandleFunc("POST /payments/{id}/settle", s.settlePayment)

	mux.HandleFunc("POST /bank-statement-lines", s.createStatementLine)
	mux.HandleFunc("GET /bank-statement-lines", s.listStatementLines)
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
