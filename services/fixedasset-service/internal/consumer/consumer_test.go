package consumer

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/fixedasset-service/internal/fadb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type stubStore struct {
	called bool
	seen   map[uuid.UUID]bool
	asset  fadb.CreateFixedAssetParams
}

func (s *stubStore) ConsumeCapExAsset(_ context.Context, _ []uuid.UUID, mark fadb.MarkEventProcessedParams, asset fadb.CreateFixedAssetParams) (bool, error) {
	if s.seen == nil {
		s.seen = make(map[uuid.UUID]bool)
	}
	if s.seen[mark.EventID] {
		return false, nil
	}
	s.seen[mark.EventID] = true
	s.called = true
	s.asset = asset
	return true, nil
}

func TestHandleCapExCreatesDraftAssetIdempotently(t *testing.T) {
	store := &stubStore{}
	c := NewHandler(store)
	orgID := uuid.New()
	evID := uuid.New()
	catID := uuid.New()

	// 1. Non-CapEx AP invoice is cleanly ignored
	err := c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "ap.invoice.posted",
		OrganizationID: orgID.String(),
		Data:           []byte(`{"invoiceNo":"AP-OPEX-1","totalAmount":"500000.0000","lines":[{"accountCode":"6100","amount":"500000.0000"}]}`),
	})
	if err != nil {
		t.Fatalf("opex event returned error: %v", err)
	}
	if store.called {
		t.Fatal("opex event must not create a fixed asset")
	}

	// 2. CapEx AP invoice creates a Draft fixed asset
	capexEnv := eventbus.Envelope{
		EventID:        evID.String(),
		EventType:      "ap.invoice.posted",
		OrganizationID: orgID.String(),
		Data:           []byte(`{"invoiceNo":"AP-CAPEX-01","invoiceDate":"2026-11-20","lines":[{"accountCode":"1500","assetCategoryId":"` + catID.String() + `","description":"Server Rack","amount":"25000000.0000"}]}`),
	}
	if err := c.Handle(context.Background(), capexEnv); err != nil {
		t.Fatalf("capex event failed: %v", err)
	}
	if !store.called || store.asset.Status != "Draft" || store.asset.AssetNo != "FA-AP-CAPEX-01" || store.asset.AssetCategoryID != catID {
		t.Fatalf("unexpected draft asset: %+v", store.asset)
	}
	if got := platform.FormatNumeric(store.asset.AcquisitionCost, 4); got != "25000000.0000" {
		t.Fatalf("expected acquisitionCost 25000000.0000, got %s", got)
	}

	// 3. Duplicate delivery is idempotent
	if err := c.Handle(context.Background(), capexEnv); err != nil {
		t.Fatalf("duplicate event failed: %v", err)
	}

	// 4. Poison eventId returns ErrPoison
	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        "bad-uuid",
		EventType:      "procurement.goods.received",
		OrganizationID: orgID.String(),
	})
	var poison eventbus.ErrPoison
	if !errors.As(err, &poison) {
		t.Fatalf("expected ErrPoison, got %v", err)
	}
}
