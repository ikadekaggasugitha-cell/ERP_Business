package consumer

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/reporting-service/internal/reportingdb"
	"github.com/sinergiinvestasiproperti/erp/services/reporting-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type fakeProjector struct {
	deltas []repository.GLAccountDelta
	coa    reportingdb.UpsertCOACacheParams
	sales  reportingdb.UpsertSalesSummaryParams
}

func (f *fakeProjector) ConsumeGL(_ context.Context, _ []uuid.UUID, _ reportingdb.MarkEventProcessedParams, d []repository.GLAccountDelta) (bool, error) {
	f.deltas = d
	return true, nil
}
func (f *fakeProjector) ConsumeCOAAccount(_ context.Context, _ []uuid.UUID, _ reportingdb.MarkEventProcessedParams, u reportingdb.UpsertCOACacheParams) (bool, error) {
	f.coa = u
	return true, nil
}
func (f *fakeProjector) ConsumeSalesSummary(_ context.Context, _ []uuid.UUID, _ reportingdb.MarkEventProcessedParams, u reportingdb.UpsertSalesSummaryParams) (bool, error) {
	f.sales = u
	return true, nil
}
func (f *fakeProjector) ConsumeInventoryPosition(context.Context, []uuid.UUID, reportingdb.MarkEventProcessedParams, reportingdb.UpsertInventoryPositionParams) (bool, error) {
	return true, nil
}

func TestHandleGLProjectsOnlyPerAccountDeltasNoDoubleCounting(t *testing.T) {
	fp := &fakeProjector{}
	c := &Consumer{repo: fp}
	org := uuid.New()
	period := uuid.New()
	acct1 := uuid.New()
	acct2 := uuid.New()
	payload, _ := json.Marshal(map[string]any{
		"journalNo":      "JV-100",
		"fiscalPeriodId": period.String(),
		"totalDebit":     "1250.5000",
		"lines": []map[string]any{
			{"accountId": acct1.String(), "accountCode": "1120", "debitAmount": "1250.5000", "creditAmount": "0"},
			{"accountId": acct2.String(), "accountCode": "4100", "debitAmount": "0", "creditAmount": "1250.5000"},
		},
	})

	err := c.handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "gl.journal.posted",
		OrganizationID: org.String(),
		OccurredAt:     time.Now().UTC(),
		Data:           payload,
	})
	if err != nil {
		t.Fatalf("handle gl: %v", err)
	}
	if len(fp.deltas) != 2 {
		t.Fatalf("expected exactly 2 per-account deltas (no journal summary row), got %d", len(fp.deltas))
	}
	if fp.deltas[0].FiscalPeriodID != period || fp.deltas[0].AccountCode != "1120" || platform.FormatNumeric(fp.deltas[0].ClosingBalance, 4) != "1250.5000" {
		t.Fatalf("unexpected debit delta: %+v bal=%s", fp.deltas[0], platform.FormatNumeric(fp.deltas[0].ClosingBalance, 4))
	}
	if fp.deltas[1].AccountCode != "4100" || platform.FormatNumeric(fp.deltas[1].ClosingBalance, 4) != "-1250.5000" {
		t.Fatalf("expected credit delta 4100=-1250.5000, got code=%s bal=%s", fp.deltas[1].AccountCode, platform.FormatNumeric(fp.deltas[1].ClosingBalance, 4))
	}
}

func TestHandleCOAAccountUpsertsCache(t *testing.T) {
	fp := &fakeProjector{}
	c := &Consumer{repo: fp}
	org := uuid.New()
	acct := uuid.New()
	payload, _ := json.Marshal(map[string]any{
		"code":        "1101",
		"name":        "Kas Operasional",
		"accountType": "Asset",
	})
	err := c.handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "coa.account.upserted",
		AggregateID:    acct.String(),
		OrganizationID: org.String(),
		OccurredAt:     time.Now().UTC(),
		Data:           payload,
	})
	if err != nil {
		t.Fatalf("handle coa: %v", err)
	}
	if fp.coa.AccountID != acct || fp.coa.AccountCode != "1101" || fp.coa.AccountType != "Asset" {
		t.Fatalf("unexpected coa cache upsert: %+v", fp.coa)
	}
}
