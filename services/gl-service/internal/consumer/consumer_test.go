package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/gl-service/internal/gldb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type fakeConsumerStore struct {
	lastEntry  gldb.CreateJournalEntryParams
	lastLines  []gldb.CreateJournalLineParams
	lastPeriod gldb.UpsertFiscalPeriodStatusParams
	postCalls  int
}

func (f *fakeConsumerStore) ConsumeAndPost(_ context.Context, _ []uuid.UUID, _ gldb.MarkEventProcessedParams, entry gldb.CreateJournalEntryParams, lines []gldb.CreateJournalLineParams, _, _ string) (bool, error) {
	f.lastEntry = entry
	f.lastLines = lines
	f.postCalls++
	return true, nil
}

func (f *fakeConsumerStore) ConsumePeriodStatus(_ context.Context, _ []uuid.UUID, _ gldb.MarkEventProcessedParams, up gldb.UpsertFiscalPeriodStatusParams) (bool, error) {
	f.lastPeriod = up
	return true, nil
}

func TestResolveAccountsAndPoisonDetection(t *testing.T) {
	dr, cr, ok := resolveAccounts("ar.invoice.issued")
	if !ok || dr != acctARControl || cr != acctRevenue {
		t.Fatalf("unexpected AR accounts: %v %v %v", dr, cr, ok)
	}
	for _, tc := range []struct {
		ev     string
		dr, cr uuid.UUID
	}{
		{"fixedasset.asset.acquired", acctFixedAsset, acctCashClear},
		{"fixedasset.depreciation.posted", acctDepExpense, acctAccumDep},
		{"fixedasset.asset.disposed", acctCashClear, acctFixedAsset},
		{"treasury.payment.settled", acctAPControl, acctCashClear},
	} {
		gotDr, gotCr, gotOK := resolveAccounts(tc.ev)
		if !gotOK || gotDr != tc.dr || gotCr != tc.cr {
			t.Fatalf("%s: got (%v,%v,%v), want (%v,%v,true)", tc.ev, gotDr, gotCr, gotOK, tc.dr, tc.cr)
		}
	}

	st := &fakeConsumerStore{}
	c := NewHandler(st)
	err := c.handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "ar.invoice.issued",
		OrganizationID: uuid.NewString(),
		Data:           json.RawMessage(`{"totalAmount":"bad-amount"}`),
	})
	var poison eventbus.ErrPoison
	if !errors.As(err, &poison) {
		t.Fatalf("expected ErrPoison for invalid amount, got %v", err)
	}

	// 1. Period status sync (financialclose.period.closed & coa.period.updated)
	orgID := uuid.New()
	periodID := uuid.New()
	if err := c.handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "financialclose.period.closed",
		OrganizationID: orgID.String(),
		Data:           json.RawMessage(`{"fiscalPeriodId":"` + periodID.String() + `"}`),
	}); err != nil {
		t.Fatalf("period closed sync failed: %v", err)
	}
	if st.lastPeriod.PeriodID != periodID || st.lastPeriod.Status != "Closed" {
		t.Fatalf("unexpected period status: %+v", st.lastPeriod)
	}

	// 2. inventory.goods.issued prefers cogsAmount over totalAmount, and skips 0.0000 cleanly
	if err := c.handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "inventory.goods.issued",
		OrganizationID: orgID.String(),
		Data:           json.RawMessage(`{"reference":"SO-1","totalAmount":"1200.0000","cogsAmount":"800.0000"}`),
	}); err != nil {
		t.Fatalf("goods issued failed: %v", err)
	}
	if got := platform.FormatNumeric(st.lastLines[0].DebitAmount, 4); got != "800.0000" {
		t.Fatalf("expected COGS debit=800.0000, got %s", got)
	}
	prevCalls := st.postCalls
	if err := c.handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "inventory.goods.issued",
		OrganizationID: orgID.String(),
		Data:           json.RawMessage(`{"reference":"MNT-1","cogsAmount":"0.0000"}`),
	}); err != nil {
		t.Fatalf("zero cogs goods issued should not error: %v", err)
	}
	if st.postCalls != prevCalls {
		t.Fatalf("expected zero-cost goods issue to skip posting")
	}

	// 3. fixedasset.asset.disposed with proceedsAmount=0 uses netBookValue
	if err := c.handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "fixedasset.asset.disposed",
		OrganizationID: orgID.String(),
		Data:           json.RawMessage(`{"assetNo":"FA-1","proceedsAmount":"0.0000","netBookValue":"2500.0000"}`),
	}); err != nil {
		t.Fatalf("disposal with 0 proceeds failed: %v", err)
	}
	if got := platform.FormatNumeric(st.lastLines[0].DebitAmount, 4); got != "2500.0000" {
		t.Fatalf("expected disposal amount=2500.0000, got %s", got)
	}

	// 4. treasury.payment.settled (Incoming + exchangeRate)
	bankGL := uuid.New()
	if err := c.handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "treasury.payment.settled",
		OrganizationID: orgID.String(),
		Data:           json.RawMessage(`{"paymentNo":"PAY-AR-1","paymentType":"Incoming","amount":"100.0000","currencyCode":"USD","exchangeRate":"15850.00000000","glAccountId":"` + bankGL.String() + `"}`),
	}); err != nil {
		t.Fatalf("payment settled failed: %v", err)
	}
	if st.lastLines[0].AccountID != bankGL || st.lastLines[1].AccountID != acctARControl {
		t.Fatalf("unexpected incoming settlement accounts: dr=%s cr=%s", st.lastLines[0].AccountID, st.lastLines[1].AccountID)
	}
	if got := platform.FormatNumeric(st.lastLines[0].FunctionalAmount, 4); got != "1585000.0000" {
		t.Fatalf("expected functionalAmount=1585000.0000, got %s", got)
	}
}
