// Package consumer implements inventory-service's Kafka ingest:
//   - sales.order.confirmed -> atomic stock_balance check/deduction -> inventory.goods.issued or inventory.reservation.failed
//   - procurement.goods.received -> atomic stock_balance receipt increment & Receipt stock_movement
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/inventory-service/internal/inventorydb"
	"github.com/sinergiinvestasiproperti/erp/services/inventory-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// Topics lists operational events consumed by inventory-service.
var Topics = []string{
	eventbus.TopicFor("sales.order.confirmed"),
	eventbus.TopicFor("procurement.goods.received"),
	eventbus.TopicFor("manufacturing.workorder.completed"),
	eventbus.TopicFor("maintenance.workorder.parts_requested"),
}

var sysActor = uuid.MustParse("00000000-0000-0000-0000-0000000000c1")

// EventStore abstracts inventory repository methods invoked by the Kafka consumer.
type EventStore interface {
	FulfillSalesOrder(ctx context.Context, org, corr, actor uuid.UUID, moveDate pgtype.Date, ref string, warehouseID *uuid.UUID, lines []repository.StockLineRequest, mark inventorydb.MarkEventProcessedParams, evOK, evFail inventorydb.InsertOutboxParams) (bool, error)
	ReceiveProcurementGoods(ctx context.Context, org, corr, actor uuid.UUID, moveDate pgtype.Date, ref string, warehouseID *uuid.UUID, lines []repository.StockLineRequest, mark inventorydb.MarkEventProcessedParams) (bool, error)
}

type soLinePayload struct {
	ItemID    string `json:"itemId"`
	OrderQty  string `json:"orderQty"`
	Qty       string `json:"qty"`
	UnitPrice string `json:"unitPrice"`
}

type soPayload struct {
	SoNo         string          `json:"soNo"`
	CustomerID   string          `json:"customerId,omitempty"`
	CurrencyCode string          `json:"currencyCode,omitempty"`
	TotalAmount  string          `json:"totalAmount"`
	WarehouseID  string          `json:"warehouseId,omitempty"`
	Lines        []soLinePayload `json:"lines"`
}

type grLinePayload struct {
	ItemID    string `json:"itemId"`
	Qty       string `json:"qty"`
	UnitPrice string `json:"unitPrice"`
}

type grPayload struct {
	PoNo        string          `json:"poNo"`
	GrNo        string          `json:"grNo"`
	WarehouseID string          `json:"warehouseId"`
	Lines       []grLinePayload `json:"lines"`
}

// Consumer subscribes to Saga and goods-received events for inventory-service.
type Consumer struct {
	repo EventStore
	c    *eventbus.Consumer
}

// New constructs an inventory event Consumer.
func New(repo EventStore, brokers []string) *Consumer {
	return &Consumer{repo: repo, c: eventbus.NewConsumer(brokers, "inventory-service", Topics)}
}

// Run starts consuming events until ctx is cancelled.
func (x *Consumer) Run(ctx context.Context) error { return x.c.Run(ctx, x.Handle) }

// Close releases underlying Kafka reader resources.
func (x *Consumer) Close() error { return x.c.Close() }

// Handle processes a single inbound envelope, routing poison messages to DLQ.
func (x *Consumer) Handle(ctx context.Context, env eventbus.Envelope) error {
	switch env.EventType {
	case "sales.order.confirmed":
		return x.handleSalesOrderConfirmed(ctx, env)
	case "procurement.goods.received":
		return x.handleGoodsReceived(ctx, env)
	case "manufacturing.workorder.completed":
		return x.handleWorkOrderCompleted(ctx, env)
	case "maintenance.workorder.parts_requested":
		return x.handleMaintenancePartsRequested(ctx, env)
	default:
		return nil
	}
}

func (x *Consumer) handleSalesOrderConfirmed(ctx context.Context, env eventbus.Envelope) error {
	org, err := uuid.Parse(env.OrganizationID)
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid organizationId %q: %w", env.OrganizationID, err)}
	}
	eventID, err := uuid.Parse(env.EventID)
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid eventId %q: %w", env.EventID, err)}
	}
	corr, err := uuid.Parse(env.CorrelationID)
	if err != nil {
		corr = eventID
	}
	soID, err := uuid.Parse(env.AggregateID)
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid aggregateId %q: %w", env.AggregateID, err)}
	}

	var p soPayload
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &p); err != nil {
			return &eventbus.ErrPoison{Err: fmt.Errorf("invalid sales.order.confirmed payload: %w", err)}
		}
	}
	ref := strings.TrimSpace(p.SoNo)
	if ref == "" {
		ref = env.AggregateID
	}
	amount := strings.TrimSpace(p.TotalAmount)
	if amount == "" {
		amount = "0"
	}

	var whPtr *uuid.UUID
	if strings.TrimSpace(p.WarehouseID) != "" {
		wh, err := uuid.Parse(strings.TrimSpace(p.WarehouseID))
		if err != nil {
			return &eventbus.ErrPoison{Err: fmt.Errorf("invalid warehouseId %q: %w", p.WarehouseID, err)}
		}
		whPtr = &wh
	}

	lines := make([]repository.StockLineRequest, 0, len(p.Lines))
	for _, l := range p.Lines {
		itemID, err := uuid.Parse(l.ItemID)
		if err != nil {
			return &eventbus.ErrPoison{Err: fmt.Errorf("invalid line itemId %q: %w", l.ItemID, err)}
		}
		rawQty := strings.TrimSpace(l.OrderQty)
		if rawQty == "" {
			rawQty = strings.TrimSpace(l.Qty)
		}
		qtyNum, err := platform.ParseNumeric(rawQty)
		if err != nil {
			return &eventbus.ErrPoison{Err: fmt.Errorf("invalid line qty %q: %w", rawQty, err)}
		}
		priceNum, _ := platform.ParseOptionalNumeric(l.UnitPrice)
		lines = append(lines, repository.StockLineRequest{
			ItemID:    itemID,
			Qty:       qtyNum,
			UnitPrice: priceNum,
		})
	}

	okMap := map[string]any{
		"reference":       ref,
		"amount":          amount,
		"totalAmount":     amount,
		"sourceEventType": env.EventType,
	}
	if cid := strings.TrimSpace(p.CustomerID); cid != "" {
		okMap["customerId"] = cid
	}
	if cc := strings.TrimSpace(p.CurrencyCode); cc != "" {
		okMap["currencyCode"] = cc
	}
	okPayload, _ := json.Marshal(okMap)
	failPayload, _ := json.Marshal(map[string]any{
		"reference":       ref,
		"amount":          amount,
		"sourceEventType": env.EventType,
		"reason":          "insufficient stock",
	})
	evOK := inventorydb.InsertOutboxParams{
		AggregateType:  "StockReservation",
		AggregateID:    soID,
		EventType:      "inventory.goods.issued",
		Payload:        okPayload,
		OrganizationID: org,
		CorrelationID:  corr,
	}
	evFail := inventorydb.InsertOutboxParams{
		AggregateType:  "StockReservation",
		AggregateID:    soID,
		EventType:      "inventory.reservation.failed",
		Payload:        failPayload,
		OrganizationID: org,
		CorrelationID:  corr,
	}
	ts := env.OccurredAt
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	moveDate := pgtype.Date{Time: ts.UTC(), Valid: true}
	_, err = x.repo.FulfillSalesOrder(ctx, org, corr, sysActor, moveDate, ref, whPtr, lines,
		inventorydb.MarkEventProcessedParams{EventID: eventID, EventType: env.EventType}, evOK, evFail)
	return err
}

func (x *Consumer) handleGoodsReceived(ctx context.Context, env eventbus.Envelope) error {
	org, err := uuid.Parse(env.OrganizationID)
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid organizationId %q: %w", env.OrganizationID, err)}
	}
	eventID, err := uuid.Parse(env.EventID)
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid eventId %q: %w", env.EventID, err)}
	}
	corr, err := uuid.Parse(env.CorrelationID)
	if err != nil {
		corr = eventID
	}
	var p grPayload
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &p); err != nil {
			return &eventbus.ErrPoison{Err: fmt.Errorf("invalid procurement.goods.received payload: %w", err)}
		}
	}
	var whPtr *uuid.UUID
	if strings.TrimSpace(p.WarehouseID) != "" {
		if wh, err := uuid.Parse(strings.TrimSpace(p.WarehouseID)); err == nil {
			whPtr = &wh
		}
	}
	lines := make([]repository.StockLineRequest, 0, len(p.Lines))
	for _, l := range p.Lines {
		itemID, err := uuid.Parse(l.ItemID)
		if err != nil {
			return &eventbus.ErrPoison{Err: fmt.Errorf("invalid gr line itemId %q: %w", l.ItemID, err)}
		}
		qtyNum, err := platform.ParseNumeric(l.Qty)
		if err != nil {
			return &eventbus.ErrPoison{Err: fmt.Errorf("invalid gr line qty %q: %w", l.Qty, err)}
		}
		priceNum, _ := platform.ParseOptionalNumeric(l.UnitPrice)
		lines = append(lines, repository.StockLineRequest{
			ItemID:    itemID,
			Qty:       qtyNum,
			UnitPrice: priceNum,
		})
	}
	ref := strings.TrimSpace(p.GrNo)
	if ref == "" {
		ref = strings.TrimSpace(p.PoNo)
	}
	if ref == "" {
		ref = env.AggregateID
	}
	ts := env.OccurredAt
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	moveDate := pgtype.Date{Time: ts.UTC(), Valid: true}
	_, err = x.repo.ReceiveProcurementGoods(ctx, org, corr, sysActor, moveDate, ref, whPtr, lines,
		inventorydb.MarkEventProcessedParams{EventID: eventID, EventType: env.EventType})
	return err
}

func (x *Consumer) handleWorkOrderCompleted(ctx context.Context, env eventbus.Envelope) error {
	org, err := uuid.Parse(env.OrganizationID)
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid organizationId %q: %w", env.OrganizationID, err)}
	}
	eventID, err := uuid.Parse(env.EventID)
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid eventId %q: %w", env.EventID, err)}
	}
	corr, err := uuid.Parse(env.CorrelationID)
	if err != nil {
		corr = eventID
	}
	var p struct {
		WoNo          string `json:"woNo"`
		ProductItemID string `json:"productItemId"`
		ProducedQty   string `json:"producedQty"`
		PlannedQty    string `json:"plannedQty"`
		WarehouseID   string `json:"warehouseId"`
		UnitCost      string `json:"unitCost"`
		TotalCost     string `json:"totalCost"`
		BomLines      []struct {
			ComponentItemID string `json:"componentItemId"`
			ItemID          string `json:"itemId"`
			Quantity        string `json:"quantity"`
		} `json:"bomLines"`
	}
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &p); err != nil {
			return &eventbus.ErrPoison{Err: fmt.Errorf("invalid manufacturing.workorder.completed payload: %w", err)}
		}
	}
	itemID, err := uuid.Parse(strings.TrimSpace(p.ProductItemID))
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid productItemId %q: %w", p.ProductItemID, err)}
	}
	rawQty := strings.TrimSpace(p.ProducedQty)
	if rawQty == "" {
		rawQty = strings.TrimSpace(p.PlannedQty)
	}
	qtyNum, err := platform.ParseNumeric(rawQty)
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid producedQty %q: %w", rawQty, err)}
	}
	qtyRat := platform.NumericToRat(qtyNum)
	if qtyRat == nil || qtyRat.Sign() <= 0 {
		return &eventbus.ErrPoison{Err: fmt.Errorf("producedQty must be > 0, got %q", rawQty)}
	}
	var whPtr *uuid.UUID
	if strings.TrimSpace(p.WarehouseID) != "" {
		if wh, err := uuid.Parse(strings.TrimSpace(p.WarehouseID)); err == nil {
			whPtr = &wh
		}
	}
	unitCostNum, _ := platform.ParseNumeric("0")
	if strings.TrimSpace(p.UnitCost) != "" {
		if uc, err := platform.ParseNumeric(strings.TrimSpace(p.UnitCost)); err == nil {
			if r := platform.NumericToRat(uc); r != nil && r.Sign() >= 0 {
				unitCostNum = uc
			}
		}
	} else if strings.TrimSpace(p.TotalCost) != "" {
		if tc, err := platform.ParseNumeric(strings.TrimSpace(p.TotalCost)); err == nil {
			if r := platform.NumericToRat(tc); r != nil && r.Sign() > 0 {
				perUnit := new(big.Rat).Quo(r, qtyRat)
				if uc, err := platform.RatToNumeric(perUnit, 4); err == nil {
					unitCostNum = uc
				}
			}
		}
	}
	lines := make([]repository.StockLineRequest, 0, 1+len(p.BomLines))
	lines = append(lines, repository.StockLineRequest{
		ItemID:    itemID,
		Qty:       qtyNum,
		UnitPrice: unitCostNum,
	})
	for _, bl := range p.BomLines {
		cidStr := strings.TrimSpace(bl.ComponentItemID)
		if cidStr == "" {
			cidStr = strings.TrimSpace(bl.ItemID)
		}
		cid, err := uuid.Parse(cidStr)
		if err != nil {
			continue
		}
		bq, err := platform.ParseNumeric(strings.TrimSpace(bl.Quantity))
		if err != nil {
			continue
		}
		if bqr := platform.NumericToRat(bq); bqr != nil && bqr.Sign() > 0 {
			if negBQ, err := platform.RatToNumeric(new(big.Rat).Neg(bqr), 4); err == nil {
				lines = append(lines, repository.StockLineRequest{
					ItemID: cid,
					Qty:    negBQ,
				})
			}
		}
	}
	ref := strings.TrimSpace(p.WoNo)
	if ref == "" {
		ref = env.AggregateID
	}
	ts := env.OccurredAt
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	moveDate := pgtype.Date{Time: ts.UTC(), Valid: true}
	_, err = x.repo.ReceiveProcurementGoods(ctx, org, corr, sysActor, moveDate, "WO-"+ref, whPtr, lines,
		inventorydb.MarkEventProcessedParams{EventID: eventID, EventType: env.EventType})
	return err
}

func (x *Consumer) handleMaintenancePartsRequested(ctx context.Context, env eventbus.Envelope) error {
	org, err := uuid.Parse(env.OrganizationID)
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid organizationId %q: %w", env.OrganizationID, err)}
	}
	eventID, err := uuid.Parse(env.EventID)
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid eventId %q: %w", env.EventID, err)}
	}
	corr, err := uuid.Parse(env.CorrelationID)
	if err != nil {
		corr = eventID
	}
	aggID, err := uuid.Parse(env.AggregateID)
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid aggregateId %q: %w", env.AggregateID, err)}
	}
	var p struct {
		WoNo        string `json:"woNo"`
		ItemID      string `json:"itemId"`
		Quantity    string `json:"quantity"`
		UnitCost    string `json:"unitCost"`
		WarehouseID string `json:"warehouseId"`
	}
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &p); err != nil {
			return &eventbus.ErrPoison{Err: fmt.Errorf("invalid maintenance.workorder.parts_requested payload: %w", err)}
		}
	}
	itemID, err := uuid.Parse(strings.TrimSpace(p.ItemID))
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid itemId %q: %w", p.ItemID, err)}
	}
	qtyNum, err := platform.ParseNumeric(strings.TrimSpace(p.Quantity))
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid quantity %q: %w", p.Quantity, err)}
	}
	qtyRat := platform.NumericToRat(qtyNum)
	if qtyRat == nil || qtyRat.Sign() <= 0 {
		return &eventbus.ErrPoison{Err: fmt.Errorf("quantity must be > 0, got %q", p.Quantity)}
	}
	costNum, _ := platform.ParseOptionalNumeric(p.UnitCost)
	totalCostStr := "0.0000"
	if cr := platform.NumericToRat(costNum); cr != nil && cr.Sign() > 0 {
		if totNum, err := platform.RatToNumeric(new(big.Rat).Mul(qtyRat, cr), 4); err == nil {
			totalCostStr = platform.FormatNumeric(totNum, 4)
		}
	}
	var whPtr *uuid.UUID
	if strings.TrimSpace(p.WarehouseID) != "" {
		if wh, err := uuid.Parse(strings.TrimSpace(p.WarehouseID)); err == nil {
			whPtr = &wh
		}
	}
	ref := "MNT-" + strings.TrimSpace(p.WoNo)
	if ref == "MNT-" {
		ref = "MNT-" + env.AggregateID
	}
	qtyStr := platform.FormatNumeric(qtyNum, 4)
	okPayload, _ := json.Marshal(map[string]any{
		"reference":       ref,
		"sourceEventType": env.EventType,
		"itemId":          itemID.String(),
		"qty":             qtyStr,
		"issuedQty":       qtyStr,
		"amount":          totalCostStr,
		"cogsAmount":      totalCostStr,
	})
	failPayload, _ := json.Marshal(map[string]any{
		"reference":       ref,
		"sourceEventType": env.EventType,
		"itemId":          itemID.String(),
		"reason":          "insufficient spare parts stock",
	})
	evOK := inventorydb.InsertOutboxParams{
		AggregateType: "StockReservation", AggregateID: aggID, EventType: "inventory.goods.issued",
		Payload: okPayload, OrganizationID: org, CorrelationID: corr,
	}
	evFail := inventorydb.InsertOutboxParams{
		AggregateType: "StockReservation", AggregateID: aggID, EventType: "inventory.reservation.failed",
		Payload: failPayload, OrganizationID: org, CorrelationID: corr,
	}
	ts := env.OccurredAt
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	moveDate := pgtype.Date{Time: ts.UTC(), Valid: true}
	lines := []repository.StockLineRequest{{ItemID: itemID, Qty: qtyNum, UnitPrice: costNum}}
	_, err = x.repo.FulfillSalesOrder(ctx, org, corr, sysActor, moveDate, ref, whPtr, lines,
		inventorydb.MarkEventProcessedParams{EventID: eventID, EventType: env.EventType}, evOK, evFail)
	return err
}
