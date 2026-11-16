package consumer

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/reporting-service/internal/reportingdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type fakeProjector struct {
	summary reportingdb.UpsertGLBalanceSnapshotParams
	deltas  []reportingdb.AccumulateGLBalanceSnapshotParams
	sales   reportingdb.UpsertSalesSummaryParams
}

func (f *fakeProjector) ConsumeGL(_ context.Context, _ []uuid.UUID, _ reportingdb.MarkEventProcessedParams, s reportingdb.UpsertGLBalanceSnapshotParams, d []reportingdb.AccumulateGLBalanceSnapshotParams) (bool, error) {
	f.summary = s
	f.deltas = d
	return true, nil
}
func (f *fakeProjector) ConsumeSalesSummary(_ context.Context, _ []uuid.UUID, _ reportingdb.MarkEventProcessedParams, u reportingdb.UpsertSalesSummaryParams) (bool, error) {
	f.sales = u
	return true, nil
}
func (f *fakeProjector) ConsumeInventoryPosition(context.Context, []uuid.UUID, reportingdb.MarkEventProcessedParams, reportingdb.UpsertInventoryPositionParams) (bool, error) {
	return true, nil
}

func TestHandleGLProjectsNonZeroBalanceAndPerAccountDeltas(t *testing.T) {
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
			{"accountId": acct1.String(), "debitAmount": "1250.5000", "creditAmount": "0"},
			{"accountId": acct2.String(), "debitAmount": "0", "creditAmount": "1250.5000"},
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
	if fp.summary.AccountCode != "JV-100" || platform.FormatNumeric(fp.summary.ClosingBalance, 4) != "1250.5000" {
		t.Fatalf("unexpected summary: %+v bal=%s", fp.summary, platform.FormatNumeric(fp.summary.ClosingBalance, 4))
	}
	if fp.summary.FiscalPeriodID != period || len(fp.deltas) != 2 {
		t.Fatalf("unexpected period=%v deltas=%d", fp.summary.FiscalPeriodID, len(fp.deltas))
	}
	if platform.FormatNumeric(fp.deltas[1].ClosingBalance, 4) != "-1250.5000" {
		t.Fatalf("expected credit delta -1250.5000, got %s", platform.FormatNumeric(fp.deltas[1].ClosingBalance, 4))
	}
}

