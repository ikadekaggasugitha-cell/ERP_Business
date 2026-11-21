// Package consumer projects GL, COA, Sales, Procurement, Manufacturing, and Inventory
// domain events into reporting-service CQRS read models idempotently with exact decimal calculation.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/reporting-service/internal/reportingdb"
	"github.com/sinergiinvestasiproperti/erp/services/reporting-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

var Topics = []string{
	eventbus.TopicFor("gl.journal.posted"),
	eventbus.TopicFor("coa.account.upserted"),
	eventbus.TopicFor("coa.account.created"),
	eventbus.TopicFor("sales.order.confirmed"),
	eventbus.TopicFor("inventory.stock.reserved"),
	eventbus.TopicFor("inventory.stock.released"),
	eventbus.TopicFor("inventory.goods.issued"),
	eventbus.TopicFor("procurement.goods.received"),
	eventbus.TopicFor("manufacturing.workorder.completed"),
}

var fallbackPeriod = uuid.MustParse("20260000-0000-0000-0000-000000000001")

type Projector interface {
	ConsumeGL(ctx context.Context, scope []uuid.UUID, mark reportingdb.MarkEventProcessedParams, accountDeltas []repository.GLAccountDelta) (bool, error)
	ConsumeCOAAccount(ctx context.Context, scope []uuid.UUID, mark reportingdb.MarkEventProcessedParams, up reportingdb.UpsertCOACacheParams) (bool, error)
	ConsumeSalesSummary(ctx context.Context, scope []uuid.UUID, mark reportingdb.MarkEventProcessedParams, up reportingdb.UpsertSalesSummaryParams) (bool, error)
	ConsumeInventoryPosition(ctx context.Context, scope []uuid.UUID, mark reportingdb.MarkEventProcessedParams, ups ...reportingdb.UpsertInventoryPositionParams) (bool, error)
}

var _ Projector = (*repository.SnapshotRepository)(nil)

type Consumer struct {
	repo Projector
	c    *eventbus.Consumer
}

func New(repo Projector, brokers []string) *Consumer {
	return &Consumer{repo: repo, c: eventbus.NewConsumer(brokers, "reporting-service", Topics)}
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
	mark := reportingdb.MarkEventProcessedParams{EventID: eventID, EventType: env.EventType}

	switch env.EventType {
	case "gl.journal.posted":
		return x.handleGL(ctx, org, mark, env)
	case "coa.account.upserted", "coa.account.created":
		return x.handleCOAAccount(ctx, org, mark, env)
	case "sales.order.confirmed":
		return x.handleSales(ctx, org, mark, env)
	case "inventory.stock.reserved", "inventory.stock.released", "inventory.goods.issued",
		"procurement.goods.received", "manufacturing.workorder.completed":
		return x.handleInventory(ctx, org, mark, env)
	default:
		return nil
	}
}

func (x *Consumer) handleCOAAccount(ctx context.Context, org uuid.UUID, mark reportingdb.MarkEventProcessedParams, env eventbus.Envelope) error {
	accountID, err := uuid.Parse(strings.TrimSpace(env.AggregateID))
	if err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid coa aggregateId %q: %w", env.AggregateID, err)}
	}
	var d struct {
		Code        string `json:"code"`
		Name        string `json:"name"`
		AccountType string `json:"accountType"`
	}
	if err := json.Unmarshal(env.Data, &d); err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("malformed coa account payload: %w", err)}
	}
	code := strings.TrimSpace(d.Code)
	if code == "" {
		return nil
	}
	if len(code) > 50 {
		code = code[:50]
	}
	_, err = x.repo.ConsumeCOAAccount(ctx, []uuid.UUID{org}, mark, reportingdb.UpsertCOACacheParams{
		OrganizationID: org,
		AccountID:      accountID,
		AccountCode:    code,
		AccountName:    strings.TrimSpace(d.Name),
		AccountType:    strings.TrimSpace(d.AccountType),
	})
	return err
}

func (x *Consumer) handleGL(ctx context.Context, org uuid.UUID, mark reportingdb.MarkEventProcessedParams, env eventbus.Envelope) error {
	var d struct {
		JournalNo      string `json:"journalNo"`
		FiscalPeriodID string `json:"fiscalPeriodId"`
		TotalDebit     string `json:"totalDebit"`
		Lines          []struct {
			AccountID        string `json:"accountId"`
			AccountCode      string `json:"accountCode"`
			DebitAmount      string `json:"debitAmount"`
			CreditAmount     string `json:"creditAmount"`
			FunctionalAmount string `json:"functionalAmount"`
		} `json:"lines"`
	}
	if err := json.Unmarshal(env.Data, &d); err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("malformed gl.journal.posted payload: %w", err)}
	}
	periodID := fallbackPeriod
	if u, err := uuid.Parse(strings.TrimSpace(d.FiscalPeriodID)); err == nil {
		periodID = u
	}

	var deltas []repository.GLAccountDelta
	for _, l := range d.Lines {
		drNum, err := platform.ParseOptionalNumeric(l.DebitAmount)
		if err != nil {
			return eventbus.ErrPoison{Err: err}
		}
		crNum, err := platform.ParseOptionalNumeric(l.CreditAmount)
		if err != nil {
			return eventbus.ErrPoison{Err: err}
		}
		drRat := platform.NumericToRat(drNum)
		crRat := platform.NumericToRat(crNum)
		net := new(big.Rat).Sub(drRat, crRat)
		if strings.TrimSpace(l.FunctionalAmount) != "" {
			if fnNum, ferr := platform.ParseOptionalNumeric(l.FunctionalAmount); ferr == nil {
				fnRat := platform.NumericToRat(fnNum)
				if fnRat != nil && fnRat.Sign() > 0 {
					if drRat.Sign() > 0 && crRat.Sign() == 0 {
						net = fnRat
					} else if crRat.Sign() > 0 && drRat.Sign() == 0 {
						net = new(big.Rat).Neg(fnRat)
					}
				}
			}
		}
		netNum, _ := platform.RatToNumeric(net, 4)

		var acctID *uuid.UUID
		if rawID := strings.TrimSpace(l.AccountID); rawID != "" {
			if parsed, perr := uuid.Parse(rawID); perr == nil {
				acctID = &parsed
			}
		}
		code := strings.TrimSpace(l.AccountCode)
		if code == "" && acctID == nil {
			continue
		}
		deltas = append(deltas, repository.GLAccountDelta{
			OrganizationID: org,
			FiscalPeriodID: periodID,
			AccountID:      acctID,
			AccountCode:    code,
			ClosingBalance: netNum,
		})
	}
	_, err := x.repo.ConsumeGL(ctx, []uuid.UUID{org}, mark, deltas)
	return err
}

func (x *Consumer) handleSales(ctx context.Context, org uuid.UUID, mark reportingdb.MarkEventProcessedParams, env eventbus.Envelope) error {
	var d struct {
		CustomerID  string `json:"customerId"`
		OrderDate   string `json:"orderDate"`
		TotalAmount string `json:"totalAmount"`
	}
	if err := json.Unmarshal(env.Data, &d); err != nil {
		return eventbus.ErrPoison{Err: err}
	}
	amt, err := platform.ParseOptionalNumeric(d.TotalAmount)
	if err != nil {
		return eventbus.ErrPoison{Err: err}
	}
	when := env.OccurredAt
	if t, err := time.Parse("2006-01-02", strings.TrimSpace(d.OrderDate)); err == nil {
		when = t
	}
	if when.IsZero() {
		when = time.Now().UTC()
	}
	var custID *uuid.UUID
	if u, err := uuid.Parse(strings.TrimSpace(d.CustomerID)); err == nil {
		custID = &u
	}
	up := reportingdb.UpsertSalesSummaryParams{
		OrganizationID:   org,
		PeriodYear:       int32(when.Year()),
		PeriodMonth:      int32(when.Month()),
		CustomerID:       custID,
		TotalSalesAmount: amt,
		OrderCount:       1,
	}
	_, err = x.repo.ConsumeSalesSummary(ctx, []uuid.UUID{org}, mark, up)
	return err
}

func (x *Consumer) handleInventory(ctx context.Context, org uuid.UUID, mark reportingdb.MarkEventProcessedParams, env eventbus.Envelope) error {
	var d struct {
		WarehouseCode string `json:"warehouseCode"`
		WarehouseID   string `json:"warehouseId"`
		ItemCode      string `json:"itemCode"`
		ItemID        string `json:"itemId"`
		ProductItemID string `json:"productItemId"`
		Qty           string `json:"qty"`
		Quantity      string `json:"quantity"`
		IssuedQty     string `json:"issuedQty"`
		ReservedQty   string `json:"reservedQty"`
		ReleasedQty   string `json:"releasedQty"`
		ReceivedQty   string `json:"receivedQty"`
		CompletedQty  string `json:"completedQty"`
		PlannedQty    string `json:"plannedQty"`
		Lines         []struct {
			WarehouseCode string `json:"warehouseCode"`
			WarehouseID   string `json:"warehouseId"`
			ItemCode      string `json:"itemCode"`
			ItemID        string `json:"itemId"`
			Qty           string `json:"qty"`
			Quantity      string `json:"quantity"`
			IssuedQty     string `json:"issuedQty"`
			ReceivedQty   string `json:"receivedQty"`
		} `json:"lines"`
	}
	if err := json.Unmarshal(env.Data, &d); err != nil {
		return eventbus.ErrPoison{Err: err}
	}

	defaultWh := firstNonEmpty(d.WarehouseCode, d.WarehouseID, "DEFAULT")
	if len(defaultWh) > 50 {
		defaultWh = defaultWh[:50]
	}
	zeroNum, _ := platform.ParseNumeric("0")

	buildUpsert := func(wh, item, qtyStr string) (reportingdb.UpsertInventoryPositionParams, error) {
		if len(wh) > 50 {
			wh = wh[:50]
		}
		if len(item) > 50 {
			item = item[:50]
		}
		qtyNum, err := platform.ParseNumeric(qtyStr)
		if err != nil {
			return reportingdb.UpsertInventoryPositionParams{}, eventbus.ErrPoison{Err: err}
		}
		posRat := platform.NumericToRat(qtyNum)
		posNum, _ := platform.RatToNumeric(posRat, 4)
		negNum, _ := platform.RatToNumeric(new(big.Rat).Neg(posRat), 4)

		onHandDelta := zeroNum
		availDelta := zeroNum
		switch env.EventType {
		case "inventory.stock.reserved":
			availDelta = negNum
		case "inventory.stock.released":
			availDelta = posNum
		case "inventory.goods.issued":
			onHandDelta = negNum
		case "procurement.goods.received", "manufacturing.workorder.completed":
			onHandDelta = posNum
			availDelta = posNum
		}
		return reportingdb.UpsertInventoryPositionParams{
			OrganizationID: org,
			WarehouseCode:  wh,
			ItemCode:       item,
			OnHandQty:      onHandDelta,
			AvailableQty:   availDelta,
		}, nil
	}

	var ups []reportingdb.UpsertInventoryPositionParams
	if len(d.Lines) > 0 {
		for _, l := range d.Lines {
			wh := firstNonEmpty(l.WarehouseCode, l.WarehouseID, defaultWh)
			item := firstNonEmpty(l.ItemCode, l.ItemID, d.ItemCode, d.ItemID, env.AggregateID)
			qtyStr := firstNonEmpty(l.IssuedQty, l.ReceivedQty, l.Qty, l.Quantity, "0")
			up, err := buildUpsert(wh, item, qtyStr)
			if err != nil {
				return err
			}
			ups = append(ups, up)
		}
	} else {
		item := firstNonEmpty(d.ItemCode, d.ItemID, d.ProductItemID, env.AggregateID)
		qtyStr := firstNonEmpty(d.IssuedQty, d.ReleasedQty, d.ReceivedQty, d.CompletedQty, d.ReservedQty, d.Qty, d.Quantity, d.PlannedQty, "0")
		up, err := buildUpsert(defaultWh, item, qtyStr)
		if err != nil {
			return err
		}
		ups = append(ups, up)
	}

	_, err := x.repo.ConsumeInventoryPosition(ctx, []uuid.UUID{org}, mark, ups...)
	return err
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
