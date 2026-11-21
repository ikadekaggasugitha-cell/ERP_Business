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
	invs   []reportingdb.UpsertInventoryPositionParams
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
func (f *fakeProjector) ConsumeInventoryPosition(_ context.Context, _ []uuid.UUID, _ reportingdb.MarkEventProcessedParams, ups ...reportingdb.UpsertInventoryPositionParams) (bool, error) {
	f.invs = append(f.invs, ups...)
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

func TestHandleInventoryReceiptsReleasesAndMultiLineIssues(t *testing.T) {
	fp := &fakeProjector{}
	c := &Consumer{repo: fp}
	org := uuid.New()

	// 1. procurement.goods.received increments on_hand and available
	if err := c.handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "procurement.goods.received",
		AggregateID:    uuid.NewString(),
		OrganizationID: org.String(),
		Data:           json.RawMessage(`{"warehouseCode":"WH-1","itemCode":"ITEM-A","receivedQty":"25.0000"}`),
	}); err != nil {
		t.Fatalf("procurement.goods.received failed: %v", err)
	}
	// 2. inventory.stock.released increments available only
	if err := c.handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "inventory.stock.released",
		AggregateID:    uuid.NewString(),
		OrganizationID: org.String(),
		Data:           json.RawMessage(`{"warehouseCode":"WH-1","itemCode":"ITEM-A","releasedQty":"5.0000"}`),
	}); err != nil {
		t.Fatalf("inventory.stock.released failed: %v", err)
	}
	// 3. inventory.goods.issued with lines array decrements on_hand per line
	if err := c.handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "inventory.goods.issued",
		AggregateID:    uuid.NewString(),
		OrganizationID: org.String(),
		Data:           json.RawMessage(`{"warehouseCode":"WH-1","lines":[{"itemCode":"ITEM-A","issuedQty":"4.0000"},{"itemCode":"ITEM-B","qty":"2.0000"}]}`),
	}); err != nil {
		t.Fatalf("inventory.goods.issued lines failed: %v", err)
	}
	if len(fp.invs) != 4 {
		t.Fatalf("expected 4 inventory position upserts, got %d", len(fp.invs))
	}
	if platform.FormatNumeric(fp.invs[0].OnHandQty, 4) != "25.0000" || platform.FormatNumeric(fp.invs[0].AvailableQty, 4) != "25.0000" {
		t.Fatalf("unexpected receipt delta: %+v", fp.invs[0])
	}
	if platform.FormatNumeric(fp.invs[1].OnHandQty, 4) != "0.0000" || platform.FormatNumeric(fp.invs[1].AvailableQty, 4) != "5.0000" {
		t.Fatalf("unexpected release delta: %+v", fp.invs[1])
	}
	if fp.invs[2].ItemCode != "ITEM-A" || platform.FormatNumeric(fp.invs[2].OnHandQty, 4) != "-4.0000" {
		t.Fatalf("unexpected issue line 1 delta: %+v", fp.invs[2])
	}
	if fp.invs[3].ItemCode != "ITEM-B" || platform.FormatNumeric(fp.invs[3].OnHandQty, 4) != "-2.0000" {
		t.Fatalf("unexpected issue line 2 delta: %+v", fp.invs[3])
	}
}
