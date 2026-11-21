// Package consumer is gl-service's Kafka ingest (SDD BAB 6.5): the "muara -> GL"
// convergence (SAD 5.4.1 / Saga INT-001..003). It auto-posts balanced double-entry
// GL journals idempotently using strict decimal parsing.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/gl-service/internal/gldb"
	"github.com/sinergiinvestasiproperti/erp/services/gl-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

var Topics = []string{
	eventbus.TopicFor("ar.invoice.issued"),             // INT-001
	eventbus.TopicFor("ap.invoice.recorded"),           // INT-002
	eventbus.TopicFor("inventory.goods.issued"),        // INT-001 (COGS)
	eventbus.TopicFor("payroll.run.completed"),         // INT-003
	eventbus.TopicFor("fixedasset.asset.acquired"),     // Fixed Asset Capitalization
	eventbus.TopicFor("fixedasset.depreciation.posted"), // Fixed Asset Depreciation
	eventbus.TopicFor("fixedasset.asset.disposed"),     // Fixed Asset Disposal
}

func envUUID(key string, fallback uuid.UUID) uuid.UUID {
	if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
		if parsed, err := uuid.Parse(raw); err == nil && parsed != uuid.Nil {
			return parsed
		}
	}
	return fallback
}

var (
	sysActor         = uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	glFinOrg         = envUUID("GL_AUTO_FIN_ORG_ID", uuid.MustParse("10000000-0000-0000-0000-000000000001"))
	glLedger         = envUUID("GL_AUTO_LEDGER_ID", uuid.MustParse("10000000-0000-0000-0000-000000000002"))
	glPeriod         = envUUID("GL_AUTO_PERIOD_ID", uuid.MustParse("10000000-0000-0000-0000-000000000003"))
	acctARControl    = envUUID("GL_ACCT_AR_CONTROL", uuid.MustParse("10000000-0000-0000-0000-0000000000f1"))
	acctRevenue      = envUUID("GL_ACCT_REVENUE", uuid.MustParse("10000000-0000-0000-0000-0000000000f2"))
	acctExpense      = envUUID("GL_ACCT_EXPENSE", uuid.MustParse("10000000-0000-0000-0000-0000000000f3"))
	acctAPControl    = envUUID("GL_ACCT_AP_CONTROL", uuid.MustParse("10000000-0000-0000-0000-0000000000f4"))
	acctCOGS         = envUUID("GL_ACCT_COGS", uuid.MustParse("10000000-0000-0000-0000-0000000000f5"))
	acctInventory    = envUUID("GL_ACCT_INVENTORY", uuid.MustParse("10000000-0000-0000-0000-0000000000f6"))
	acctPayrollExp   = envUUID("GL_ACCT_PAYROLL_EXP", uuid.MustParse("10000000-0000-0000-0000-0000000000f7"))
	acctPayrollPayab = envUUID("GL_ACCT_PAYROLL_PAYABLE", uuid.MustParse("10000000-0000-0000-0000-0000000000f8"))
	acctFixedAsset   = envUUID("GL_ACCT_FIXED_ASSET", uuid.MustParse("10000000-0000-0000-0000-0000000000f9"))
	acctAccumDep     = envUUID("GL_ACCT_ACCUM_DEP", uuid.MustParse("10000000-0000-0000-0000-0000000000fa"))
	acctDepExpense   = envUUID("GL_ACCT_DEP_EXPENSE", uuid.MustParse("10000000-0000-0000-0000-0000000000fb"))
	acctCashClear    = envUUID("GL_ACCT_CASH_CLEAR", uuid.MustParse("10000000-0000-0000-0000-0000000000fc"))
)

type srcPayload struct {
	InvoiceNo        string `json:"invoiceNo"`
	RunNo            string `json:"runNo"`
	AssetNo          string `json:"assetNo"`
	Reference        string `json:"reference"`
	InvoiceDate      string `json:"invoiceDate"`
	CurrencyCode     string `json:"currencyCode"`
	TotalAmount      string `json:"totalAmount"`
	Amount           string `json:"amount"`
	NetBookValue     string `json:"netBookValue"`
	AssetAccountID   string `json:"assetAccountId"`
	ExpenseAccountID string `json:"expenseAccountId"`
	AccumAccountID   string `json:"accumAccountId"`
	FiscalPeriodID   string `json:"fiscalPeriodId"`
}

type Consumer struct {
	repo *repository.JournalRepository
	c    *eventbus.Consumer
}

func New(repo *repository.JournalRepository, brokers []string) *Consumer {
	return &Consumer{repo: repo, c: eventbus.NewConsumer(brokers, "gl-service", Topics)}
}

func (x *Consumer) Run(ctx context.Context) error { return x.c.Run(ctx, x.handle) }
func (x *Consumer) Close() error                  { return x.c.Close() }

func (x *Consumer) handle(ctx context.Context, env eventbus.Envelope) error {
	org, err := uuid.Parse(env.OrganizationID)
	if err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid organizationId %q: %w", env.OrganizationID, err)}
	}
	eventID, err := uuid.Parse(env.EventID)
	if err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid eventId %q: %w", env.EventID, err)}
	}
	corr, err := uuid.Parse(env.CorrelationID)
	if err != nil {
		corr = eventID
	}

	debitAcct, creditAcct, ok := resolveAccounts(env.EventType)
	if !ok {
		return nil
	}

	var p srcPayload
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &p); err != nil {
			return eventbus.ErrPoison{Err: fmt.Errorf("malformed data payload: %w", err)}
		}
	}
	pd := parseDate(p.InvoiceDate)
	if !pd.Valid {
		t := env.OccurredAt
		if t.IsZero() {
			t = time.Now().UTC()
		}
		pd = pgtype.Date{Time: t, Valid: true}
	}
	amtRaw := firstNonEmpty(p.TotalAmount, p.Amount, p.NetBookValue, "0")
	amtNum, err := platform.ParseNumeric(amtRaw)
	if err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid subledger amount %q: %w", amtRaw, err)}
	}
	if amtRat := platform.NumericToRat(amtNum); amtRat == nil || amtRat.Sign() <= 0 {
		return eventbus.ErrPoison{Err: fmt.Errorf("subledger amount must be > 0, got %q", amtRaw)}
	}
	// Honor asset-specific COA accounts when provided by fixedasset-service events.
	switch env.EventType {
	case "fixedasset.asset.acquired":
		if u, e := uuid.Parse(strings.TrimSpace(p.AssetAccountID)); e == nil && u != uuid.Nil {
			debitAcct = u
		}
	case "fixedasset.depreciation.posted":
		if u, e := uuid.Parse(strings.TrimSpace(p.ExpenseAccountID)); e == nil && u != uuid.Nil {
			debitAcct = u
		}
		if u, e := uuid.Parse(strings.TrimSpace(p.AccumAccountID)); e == nil && u != uuid.Nil {
			creditAcct = u
		}
	case "fixedasset.asset.disposed":
		if u, e := uuid.Parse(strings.TrimSpace(p.AssetAccountID)); e == nil && u != uuid.Nil {
			creditAcct = u
		}
	}
	periodID := glPeriod
	if u, e := uuid.Parse(strings.TrimSpace(p.FiscalPeriodID)); e == nil && u != uuid.Nil {
		periodID = u
	}
	zeroNum, _ := platform.ParseNumeric("0")
	exRate, _ := platform.ParseNumeric("1")

	src := firstNonEmpty(p.InvoiceNo, p.RunNo, p.Reference, p.AssetNo, env.AggregateID)
	journalNo := "AUTO-" + src
	if len(journalNo) > 50 {
		journalNo = journalNo[:50]
	}

	srcModule := strings.ToUpper(strings.SplitN(env.EventType, ".", 2)[0])
	entry := gldb.CreateJournalEntryParams{
		OrganizationID:          org,
		JournalNo:               journalNo,
		FinancialOrganizationID: glFinOrg,
		LedgerID:                glLedger,
		FiscalPeriodID:          periodID,
		PostingDate:             pd,
		JournalSource:           "SubLedger",
		SourceModule:            &srcModule,
		SourceDocumentRef:       &src,
		CurrencyCode:            strings.ToUpper(firstNonEmpty(p.CurrencyCode, "IDR")),
		ExchangeRate:            exRate,
		CorrelationID:           corr,
		CreatedBy:               sysActor,
	}
	lines := []gldb.CreateJournalLineParams{
		{OrganizationID: org, LineNo: 1, AccountID: debitAcct, DebitAmount: amtNum, CreditAmount: zeroNum, FunctionalAmount: amtNum, CreatedBy: sysActor},
		{OrganizationID: org, LineNo: 2, AccountID: creditAcct, DebitAmount: zeroNum, CreditAmount: amtNum, FunctionalAmount: amtNum, CreatedBy: sysActor},
	}
	_, err = x.repo.ConsumeAndPost(ctx, []uuid.UUID{org},
		gldb.MarkEventProcessedParams{EventID: eventID, EventType: env.EventType}, entry, lines, env.EventType, src)
	return err
}

func resolveAccounts(eventType string) (debit, credit uuid.UUID, ok bool) {
	switch eventType {
	case "ar.invoice.issued":
		return acctARControl, acctRevenue, true
	case "ap.invoice.recorded":
		return acctExpense, acctAPControl, true
	case "inventory.goods.issued":
		return acctCOGS, acctInventory, true
	case "payroll.run.completed":
		return acctPayrollExp, acctPayrollPayab, true
	case "fixedasset.asset.acquired":
		return acctFixedAsset, acctCashClear, true
	case "fixedasset.depreciation.posted":
		return acctDepExpense, acctAccumDep, true
	case "fixedasset.asset.disposed":
		return acctCashClear, acctFixedAsset, true
	default:
		return uuid.Nil, uuid.Nil, false
	}
}

func parseDate(s string) pgtype.Date {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(s))
	if err != nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: t, Valid: true}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
