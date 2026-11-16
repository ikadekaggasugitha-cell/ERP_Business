// Package consumer projects GL, Sales, and Inventory domain events into
// reporting-service CQRS read models idempotently with exact decimal calculation.
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
	eventbus.TopicFor("sales.order.confirmed"),
	eventbus.TopicFor("inventory.stock.reserved"),
	eventbus.TopicFor("inventory.goods.issued"),
}

var fallbackPeriod = uuid.MustParse("20260000-0000-0000-0000-000000000001")

type Projector interface {
	ConsumeGL(ctx context.Context, scope []uuid.UUID, mark reportingdb.MarkEventProcessedParams, summary reportingdb.UpsertGLBalanceSnapshotParams, accountDeltas []reportingdb.AccumulateGLBalanceSnapshotParams) (bool, error)
	ConsumeSalesSummary(ctx context.Context, scope []uuid.UUID, mark reportingdb.MarkEventProcessedParams, up reportingdb.UpsertSalesSummaryParams) (bool, error)
	ConsumeInventoryPosition(ctx context.Context, scope []uuid.UUID, mark reportingdb.MarkEventProcessedParams, up reportingdb.UpsertInventoryPositionParams) (bool, error)
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
	case "sales.order.confirmed":
		return x.handleSales(ctx, org, mark, env)
	case "inventory.stock.reserved", "inventory.goods.issued":
		return x.handleInventory(ctx, org, mark, env)
	default:
		return nil
	}
}

func (x *Consumer) handleGL(ctx context.Context, org uuid.UUID, mark reportingdb.MarkEventProcessedParams, env eventbus.Envelope) error {
	var d struct {
		JournalNo      string `json:"journalNo"`
		FiscalPeriodID string `json:"fiscalPeriodId"`
		TotalDebit     string `json:"totalDebit"`
		Lines          []struct {
			AccountID    string `json:"accountId"`
			DebitAmount  string `json:"debitAmount"`
			CreditAmount string `json:"creditAmount"`
		} `json:"lines"`
	}
	if err := json.Unmarshal(env.Data, &d); err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("malformed gl.journal.posted payload: %w", err)}
	}
	acctCode := strings.TrimSpace(d.JournalNo)
	if acctCode == "" {
		acctCode = env.AggregateID
	}
	periodID := fallbackPeriod
	if u, err := uuid.Parse(strings.TrimSpace(d.FiscalPeriodID)); err == nil {
		periodID = u
	}

	totRat := new(big.Rat)
	var deltas []reportingdb.AccumulateGLBalanceSnapshotParams
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
		totRat.Add(totRat, drRat)
		net := new(big.Rat).Sub(drRat, crRat)
		netNum, _ := platform.RatToNumeric(net, 4)
		if strings.TrimSpace(l.AccountID) != "" {
			code := "ACCT:" + l.AccountID
			if len(code) > 50 {
				code = code[:50]
			}
			deltas = append(deltas, reportingdb.AccumulateGLBalanceSnapshotParams{
				OrganizationID: org,
				FiscalPeriodID: periodID,
				AccountCode:    code,
				ClosingBalance: netNum,
			})
		}
	}
	if strings.TrimSpace(d.TotalDebit) != "" {
		n, err := platform.ParseNumeric(d.TotalDebit)
		if err != nil {
			return eventbus.ErrPoison{Err: err}
		}
		totRat = platform.NumericToRat(n)
	}
	balNum, _ := platform.RatToNumeric(totRat, 4)
	summary := reportingdb.UpsertGLBalanceSnapshotParams{
		OrganizationID: org,
		FiscalPeriodID: periodID,
		AccountCode:    acctCode,
		ClosingBalance: balNum,
	}
	_, err := x.repo.ConsumeGL(ctx, []uuid.UUID{org}, mark, summary, deltas)
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
		Qty           string `json:"qty"`
		ReservedQty   string `json:"reservedQty"`
	}
	if err := json.Unmarshal(env.Data, &d); err != nil {
		return eventbus.ErrPoison{Err: err}
	}
	wh := firstNonEmpty(d.WarehouseCode, d.WarehouseID, "DEFAULT")
	if len(wh) > 50 {
		wh = wh[:50]
	}
	item := firstNonEmpty(d.ItemCode, d.ItemID, env.AggregateID)
	if len(item) > 50 {
		item = item[:50]
	}
	qtyStr := firstNonEmpty(d.ReservedQty, d.Qty, "0")
	qtyNum, err := platform.ParseNumeric(qtyStr)
	if err != nil {
		return eventbus.ErrPoison{Err: err}
	}
	negRat := new(big.Rat).Neg(platform.NumericToRat(qtyNum))
	negNum, _ := platform.RatToNumeric(negRat, 4)
	zeroNum, _ := platform.ParseNumeric("0")

	onHandDelta := zeroNum
	availDelta := negNum
	if env.EventType == "inventory.goods.issued" {
		onHandDelta = negNum
		availDelta = zeroNum
	}
	up := reportingdb.UpsertInventoryPositionParams{
		OrganizationID: org,
		WarehouseCode:  wh,
		ItemCode:       item,
		OnHandQty:      onHandDelta,
		AvailableQty:   availDelta,
	}
	_, err = x.repo.ConsumeInventoryPosition(ctx, []uuid.UUID{org}, mark, up)
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
