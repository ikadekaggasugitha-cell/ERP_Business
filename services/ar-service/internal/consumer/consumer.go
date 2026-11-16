// Package consumer reacts to inventory.goods.issued by issuing an AR customer
// invoice and emitting ar.invoice.issued idempotently and atomically.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/ar-service/internal/ardb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// Topics consumed by ar-service (Saga INT-001).
var Topics = []string{eventbus.TopicFor("inventory.goods.issued")}

var (
	sysActor   = uuid.MustParse("00000000-0000-0000-0000-0000000000b1")
	sysCust    = uuid.MustParse("10000000-0000-0000-0000-0000000000d1")
	revAccount = uuid.MustParse("10000000-0000-0000-0000-0000000000e1")
)

type goodsIssuedPayload struct {
	Reference  string `json:"reference"`
	CustomerID string `json:"customerId"`
	Amount     string `json:"amount"`
}

// Store abstracts AR repository calls for unit testing.
type Store interface {
	ConsumeAndCreateInvoice(ctx context.Context, scope []uuid.UUID, mark ardb.MarkEventProcessedParams, invoice ardb.CreateInvoiceParams, lines []ardb.CreateInvoiceLineParams, ev ardb.InsertOutboxParams) (bool, error)
}

type Consumer struct {
	repo Store
	c    *eventbus.Consumer
}

func New(repo Store, brokers []string) *Consumer {
	return &Consumer{repo: repo, c: eventbus.NewConsumer(brokers, "ar-service", Topics)}
}

func NewHandler(repo Store) *Consumer {
	return &Consumer{repo: repo}
}

func (x *Consumer) Run(ctx context.Context) error { return x.c.Run(ctx, x.Handle) }

func (x *Consumer) Close() error {
	if x == nil || x.c == nil {
		return nil
	}
	return x.c.Close()
}

// Handle creates+emits an AR invoice for inventory.goods.issued; returns ErrPoison on malformed envelopes.
func (x *Consumer) Handle(ctx context.Context, env eventbus.Envelope) error {
	if env.EventType != "inventory.goods.issued" {
		return nil
	}
	org, err := uuid.Parse(strings.TrimSpace(env.OrganizationID))
	if err != nil || org == uuid.Nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid organizationId")}
	}
	eventID, err := uuid.Parse(strings.TrimSpace(env.EventID))
	if err != nil || eventID == uuid.Nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid eventId")}
	}
	corr := eventID
	if c, err := uuid.Parse(strings.TrimSpace(env.CorrelationID)); err == nil && c != uuid.Nil {
		corr = c
	}

	var p goodsIssuedPayload
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &p); err != nil {
			return eventbus.ErrPoison{Err: fmt.Errorf("invalid payload: %w", err)}
		}
	}
	ref := strings.TrimSpace(p.Reference)
	if ref == "" {
		ref = strings.TrimSpace(env.AggregateID)
	}
	if ref == "" {
		return eventbus.ErrPoison{Err: fmt.Errorf("missing reference/aggregateId")}
	}
	total, err := platform.ParseOptionalNumeric(p.Amount)
	if err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid amount: %w", err)}
	}
	custID := sysCust
	if strings.TrimSpace(p.CustomerID) != "" {
		if c, err := uuid.Parse(strings.TrimSpace(p.CustomerID)); err == nil && c != uuid.Nil {
			custID = c
		}
	}

	occurred := env.OccurredAt
	if occurred.IsZero() {
		occurred = time.Now().UTC()
	}
	dt := pgtype.Date{Time: occurred.UTC(), Valid: true}
	invNo := "SI-" + ref

	invoice := ardb.CreateInvoiceParams{
		OrganizationID:    org,
		InvoiceNo:         invNo,
		CustomerID:        custID,
		InvoiceDate:       dt,
		DueDate:           dt,
		CurrencyCode:      "IDR",
		TotalAmount:       total,
		OutstandingAmount: total,
		CreatedBy:         sysActor,
	}
	lines := []ardb.CreateInvoiceLineParams{
		{OrganizationID: org, LineNo: 1, AccountID: revAccount, Amount: total, CreatedBy: sysActor},
	}
	payload, _ := json.Marshal(map[string]any{
		"invoiceNo":         invNo,
		"customerId":        custID.String(),
		"invoiceDate":       platform.FormatDate(dt),
		"dueDate":           platform.FormatDate(dt),
		"currencyCode":      "IDR",
		"totalAmount":       platform.FormatNumeric(total, 4),
		"taxAmount":         "0.0000",
		"status":            "Issued",
		"sourceEventType":   env.EventType,
		"sourceDocumentRef": ref,
		"lines": []map[string]any{{
			"lineNo":    1,
			"accountId": revAccount.String(),
			"amount":    platform.FormatNumeric(total, 4),
		}},
	})
	ev := ardb.InsertOutboxParams{
		AggregateType:  "ARInvoice",
		EventType:      "ar.invoice.issued",
		Payload:        payload,
		OrganizationID: org,
		CorrelationID:  corr,
	}
	_, err = x.repo.ConsumeAndCreateInvoice(ctx, []uuid.UUID{org},
		ardb.MarkEventProcessedParams{EventID: eventID, EventType: env.EventType}, invoice, lines, ev)
	return err
}
