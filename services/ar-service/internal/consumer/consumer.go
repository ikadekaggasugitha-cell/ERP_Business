// Package consumer reacts to inventory.goods.issued and project.billing.approved
// by issuing an AR customer invoice, and reacts to treasury.payment.settled by
// recording receipt allocations and updating AR invoice status.
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
	"github.com/sinergiinvestasiproperti/erp/services/ar-service/internal/repository"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

// Topics consumed by ar-service (Saga INT-001 + Project Billing + Treasury Settlement).
var Topics = []string{
	eventbus.TopicFor("inventory.goods.issued"),
	eventbus.TopicFor("project.billing.approved"),
	eventbus.TopicFor("treasury.payment.settled"),
}

var (
	sysActor   = uuid.MustParse("00000000-0000-0000-0000-0000000000b1")
	sysCust    = uuid.MustParse("10000000-0000-0000-0000-0000000000d1")
	revAccount = uuid.MustParse("10000000-0000-0000-0000-0000000000e1")
)

type inboundPayload struct {
	Reference       string `json:"reference"`
	ArInvoiceRef    string `json:"arInvoiceRef"`
	ProjectCode     string `json:"projectCode"`
	CustomerID      string `json:"customerId"`
	CurrencyCode    string `json:"currencyCode"`
	Amount          string `json:"amount"`
	TotalAmount     string `json:"totalAmount"`
	SourceEventType string `json:"sourceEventType"`
}

type paymentSettledPayload struct {
	PaymentID   string `json:"paymentId"`
	PaymentNo   string `json:"paymentNo"`
	PaymentType string `json:"paymentType"`
	Amount      string `json:"amount"`
	ExternalRef string `json:"externalRef"`
	Allocations []struct {
		InvoiceID         string `json:"invoiceId"`
		TargetDocumentRef string `json:"targetDocumentRef"`
		InvoiceType       string `json:"invoiceType"`
		AllocatedAmount   string `json:"allocatedAmount"`
	} `json:"allocations"`
}

// Store abstracts AR repository calls for unit testing.
type Store interface {
	ConsumeAndCreateInvoice(ctx context.Context, scope []uuid.UUID, mark ardb.MarkEventProcessedParams, invoice ardb.CreateInvoiceParams, lines []ardb.CreateInvoiceLineParams, ev ardb.InsertOutboxParams) (bool, error)
	ConsumePaymentSettled(ctx context.Context, scope []uuid.UUID, mark ardb.MarkEventProcessedParams, orgID, receiptID, actor, corrID uuid.UUID, allocs []repository.SettlementAllocation) (bool, error)
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

// Handle creates+emits an AR invoice for inventory.goods.issued and project.billing.approved,
// and applies treasury.payment.settled allocations; returns ErrPoison on malformed envelopes.
func (x *Consumer) Handle(ctx context.Context, env eventbus.Envelope) error {
	if env.EventType != "inventory.goods.issued" && env.EventType != "project.billing.approved" && env.EventType != "treasury.payment.settled" {
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

	if env.EventType == "treasury.payment.settled" {
		return x.handlePaymentSettled(ctx, org, eventID, corr, env)
	}

	var p inboundPayload
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &p); err != nil {
			return eventbus.ErrPoison{Err: fmt.Errorf("invalid payload: %w", err)}
		}
	}
	// Never issue customer AR invoices for internal maintenance spare-part issues.
	if env.EventType == "inventory.goods.issued" && strings.EqualFold(strings.TrimSpace(p.SourceEventType), "maintenance.workorder.parts_requested") {
		return nil
	}

	ref := strings.TrimSpace(p.ArInvoiceRef)
	if ref == "" {
		ref = strings.TrimSpace(p.Reference)
	}
	if ref == "" {
		ref = strings.TrimSpace(p.ProjectCode)
	}
	if ref == "" {
		ref = strings.TrimSpace(env.AggregateID)
	}
	if ref == "" {
		return eventbus.ErrPoison{Err: fmt.Errorf("missing reference/aggregateId")}
	}
	amtRaw := strings.TrimSpace(p.TotalAmount)
	if amtRaw == "" {
		amtRaw = strings.TrimSpace(p.Amount)
	}
	total, err := platform.ParseOptionalNumeric(amtRaw)
	if err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid amount: %w", err)}
	}
	ccy := strings.ToUpper(strings.TrimSpace(p.CurrencyCode))
	if len(ccy) != 3 {
		ccy = "IDR"
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
	if env.EventType == "project.billing.approved" && strings.HasPrefix(ref, "PB-") {
		invNo = ref
	}
	if len(invNo) > 50 {
		invNo = invNo[:50]
	}

	invoice := ardb.CreateInvoiceParams{
		OrganizationID:    org,
		InvoiceNo:         invNo,
		CustomerID:        custID,
		InvoiceDate:       dt,
		DueDate:           dt,
		CurrencyCode:      ccy,
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
		"currencyCode":      ccy,
		"totalAmount":       platform.FormatNumeric(total, 4),
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

func (x *Consumer) handlePaymentSettled(ctx context.Context, org, eventID, corr uuid.UUID, env eventbus.Envelope) error {
	var p paymentSettledPayload
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &p); err != nil {
			return eventbus.ErrPoison{Err: fmt.Errorf("invalid treasury.payment.settled payload: %w", err)}
		}
	}
	if strings.EqualFold(strings.TrimSpace(p.PaymentType), "Outgoing") {
		return nil
	}
	receiptID, err := uuid.Parse(strings.TrimSpace(p.PaymentID))
	if err != nil || receiptID == uuid.Nil {
		receiptID, err = uuid.Parse(strings.TrimSpace(env.AggregateID))
		if err != nil || receiptID == uuid.Nil {
			receiptID = eventID
		}
	}
	var allocs []repository.SettlementAllocation
	for _, a := range p.Allocations {
		if a.InvoiceType != "" && !strings.EqualFold(strings.TrimSpace(a.InvoiceType), "AR") {
			continue
		}
		ref := strings.TrimSpace(a.InvoiceID)
		if ref == "" {
			ref = strings.TrimSpace(a.TargetDocumentRef)
		}
		if ref == "" {
			continue
		}
		amt, err := platform.ParseNumeric(a.AllocatedAmount)
		if err != nil {
			return eventbus.ErrPoison{Err: fmt.Errorf("invalid allocatedAmount %q: %w", a.AllocatedAmount, err)}
		}
		if rVal := platform.NumericToRat(amt); rVal == nil || rVal.Sign() <= 0 {
			continue
		}
		allocs = append(allocs, repository.SettlementAllocation{
			InvoiceRef:      ref,
			AllocatedAmount: amt,
		})
	}
	if len(allocs) == 0 && strings.TrimSpace(p.ExternalRef) != "" && strings.TrimSpace(p.Amount) != "" {
		amt, err := platform.ParseNumeric(p.Amount)
		if err != nil {
			return eventbus.ErrPoison{Err: fmt.Errorf("invalid payment amount %q: %w", p.Amount, err)}
		}
		if rVal := platform.NumericToRat(amt); rVal != nil && rVal.Sign() > 0 {
			allocs = append(allocs, repository.SettlementAllocation{
				InvoiceRef:      strings.TrimSpace(p.ExternalRef),
				AllocatedAmount: amt,
			})
		}
	}
	if len(allocs) == 0 {
		return nil
	}
	_, err = x.repo.ConsumePaymentSettled(ctx, []uuid.UUID{org},
		ardb.MarkEventProcessedParams{EventID: eventID, EventType: env.EventType},
		org, receiptID, sysActor, corr, allocs)
	return err
}
