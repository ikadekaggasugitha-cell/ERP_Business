package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/inventory-service/internal/inventorydb"
	"github.com/sinergiinvestasiproperti/erp/services/inventory-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

type fakeEventStore struct {
	soCalled   bool
	soLines    []repository.StockLineRequest
	grCalled   bool
	grLines    []repository.StockLineRequest
	emittedOK  inventorydb.InsertOutboxParams
}

func (f *fakeEventStore) FulfillSalesOrder(_ context.Context, _, _, _ uuid.UUID, _ pgtype.Date, _ string, _ *uuid.UUID, lines []repository.StockLineRequest, _ inventorydb.MarkEventProcessedParams, evOK, _ inventorydb.InsertOutboxParams) (bool, error) {
	f.soCalled = true
	f.soLines = lines
	f.emittedOK = evOK
	return true, nil
}

func (f *fakeEventStore) ReceiveProcurementGoods(_ context.Context, _, _, _ uuid.UUID, _ pgtype.Date, _ string, _ *uuid.UUID, lines []repository.StockLineRequest, _ inventorydb.MarkEventProcessedParams) (bool, error) {
	f.grCalled = true
	f.grLines = lines
	return true, nil
}

func TestInventoryConsumerHandlersAndPoison(t *testing.T) {
	store := &fakeEventStore{}
	c := &Consumer{repo: store}

	// Poison: invalid organizationId
	err := c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "sales.order.confirmed",
		AggregateID:    uuid.NewString(),
		OrganizationID: "bad-uuid",
	})
	var poison *eventbus.ErrPoison
	if !errors.As(err, &poison) {
		t.Fatalf("expected ErrPoison for bad org UUID, got %v", err)
	}

	// Poison: bad line itemId
	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "sales.order.confirmed",
		AggregateID:    uuid.NewString(),
		OrganizationID: uuid.NewString(),
		Data:           json.RawMessage(`{"soNo":"SO-1","totalAmount":"100","lines":[{"itemId":"bad","orderQty":"2"}]}`),
	})
	if !errors.As(err, &poison) {
		t.Fatalf("expected ErrPoison for bad item UUID, got %v", err)
	}

	// Valid sales.order.confirmed delegates lines to FulfillSalesOrder and propagates customerId/currencyCode
	itemID := uuid.NewString()
	custID := uuid.NewString()
	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "sales.order.confirmed",
		AggregateID:    uuid.NewString(),
		OrganizationID: uuid.NewString(),
		Data:           json.RawMessage(`{"soNo":"SO-1","customerId":"` + custID + `","currencyCode":"IDR","totalAmount":"1200.0000","lines":[{"itemId":"` + itemID + `","orderQty":"4","unitPrice":"300"}]}`),
	})
	if err != nil || !store.soCalled || len(store.soLines) != 1 {
		t.Fatalf("expected valid SO fulfillment call, err=%v called=%v lines=%d", err, store.soCalled, len(store.soLines))
	}
	var okMap map[string]any
	_ = json.Unmarshal(store.emittedOK.Payload, &okMap)
	if okMap["customerId"] != custID || okMap["currencyCode"] != "IDR" {
		t.Fatalf("expected customerId & currencyCode in inventory.goods.issued, got %+v", okMap)
	}

	// Valid procurement.goods.received delegates lines to ReceiveProcurementGoods
	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "procurement.goods.received",
		AggregateID:    uuid.NewString(),
		OrganizationID: uuid.NewString(),
		Data:           json.RawMessage(`{"poNo":"PO-1","grNo":"GR-1","lines":[{"itemId":"` + itemID + `","qty":"3","unitPrice":"300"}]}`),
	})
	if err != nil || !store.grCalled || len(store.grLines) != 1 {
		t.Fatalf("expected valid GR receipt call, err=%v called=%v lines=%d", err, store.grCalled, len(store.grLines))
	}

	// Valid manufacturing.workorder.completed receives finished goods + deducts BOM lines
	store.grCalled = false
	store.grLines = nil
	compID := uuid.NewString()
	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "manufacturing.workorder.completed",
		AggregateID:    uuid.NewString(),
		OrganizationID: uuid.NewString(),
		Data:           json.RawMessage(`{"woNo":"WO-100","productItemId":"` + itemID + `","producedQty":"10.0000","unitCost":"25000.0000","bomLines":[{"componentItemId":"` + compID + `","quantity":"20.0000"}]}`),
	})
	if err != nil || !store.grCalled || len(store.grLines) != 2 {
		t.Fatalf("expected finished goods receipt + 1 BOM deduction on WO completion, err=%v called=%v lines=%d", err, store.grCalled, len(store.grLines))
	}

	// Valid maintenance.workorder.parts_requested deducts spare parts stock with computed cost
	store.soCalled = false
	store.soLines = nil
	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "maintenance.workorder.parts_requested",
		AggregateID:    uuid.NewString(),
		OrganizationID: uuid.NewString(),
		Data:           json.RawMessage(`{"woNo":"MWO-9","itemId":"` + itemID + `","quantity":"2","unitCost":"50000"}`),
	})
	if err != nil || !store.soCalled || len(store.soLines) != 1 {
		t.Fatalf("expected spare parts issue on maintenance request, err=%v called=%v lines=%d", err, store.soCalled, len(store.soLines))
	}
	var mntMap map[string]any
	_ = json.Unmarshal(store.emittedOK.Payload, &mntMap)
	if mntMap["amount"] != "100000.0000" || mntMap["sourceEventType"] != "maintenance.workorder.parts_requested" {
		t.Fatalf("expected amount=100000.0000 and sourceEventType on maintenance issue, got %+v", mntMap)
	}
}

