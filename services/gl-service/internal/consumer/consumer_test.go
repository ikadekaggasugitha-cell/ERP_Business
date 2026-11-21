package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

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
	} {
		gotDr, gotCr, gotOK := resolveAccounts(tc.ev)
		if !gotOK || gotDr != tc.dr || gotCr != tc.cr {
			t.Fatalf("%s: got (%v,%v,%v), want (%v,%v,true)", tc.ev, gotDr, gotCr, gotOK, tc.dr, tc.cr)
		}
	}

	c := &Consumer{}
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
}

