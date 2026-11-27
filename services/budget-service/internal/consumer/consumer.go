package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/budget-service/internal/budgetdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

var Topics = []string{
	eventbus.TopicFor("procurement.po.approved"),
	eventbus.TopicFor("ap.invoice.recorded"),
	eventbus.TopicFor("workflow.approval.completed"),
}
var sysActor = uuid.MustParse("00000000-0000-0000-0000-0000000000b2")

type Store interface {
	ConsumePOApproved(ctx context.Context, scope []uuid.UUID, mark budgetdb.MarkEventProcessedParams, orgID, actor, corr uuid.UUID, docRef string, amount pgtype.Numeric) (bool, error)
	ConsumeAPInvoiceRecorded(ctx context.Context, scope []uuid.UUID, mark budgetdb.MarkEventProcessedParams, orgID, actor uuid.UUID, sourceRef string) (bool, error)
	ConsumeWorkflowApproved(ctx context.Context, scope []uuid.UUID, mark budgetdb.MarkEventProcessedParams, orgID, actor, corr uuid.UUID, docRef string) (bool, error)
}

type Consumer struct {
	repo Store
	c    *eventbus.Consumer
}

func New(repo Store, brokers []string) *Consumer {
	return &Consumer{repo: repo, c: eventbus.NewConsumer(brokers, "budget-service", Topics)}
}
func NewHandler(repo Store) *Consumer             { return &Consumer{repo: repo} }
func (x *Consumer) Run(ctx context.Context) error { return x.c.Run(ctx, x.Handle) }
func (x *Consumer) Close() error {
	if x.c != nil {
		return x.c.Close()
	}
	return nil
}

func (x *Consumer) Handle(ctx context.Context, env eventbus.Envelope) error {
	switch env.EventType {
	case "procurement.po.approved":
		return x.handlePOApproved(ctx, env)
	case "ap.invoice.recorded":
		return x.handleAPInvoiceRecorded(ctx, env)
	case "workflow.approval.completed":
		return x.handleWorkflowApproved(ctx, env)
	default:
		return nil
	}
}

func (x *Consumer) handlePOApproved(ctx context.Context, env eventbus.Envelope) error {
	org, err := uuid.Parse(strings.TrimSpace(env.OrganizationID))
	if err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid organizationId: %w", err)}
	}
	evID, err := uuid.Parse(strings.TrimSpace(env.EventID))
	if err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid eventId: %w", err)}
	}
	corr := evID
	if c, e := uuid.Parse(strings.TrimSpace(env.CorrelationID)); e == nil {
		corr = c
	}
	var p struct {
		PoNo        string `json:"poNo"`
		TotalAmount string `json:"totalAmount"`
	}
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &p); err != nil {
			return eventbus.ErrPoison{Err: fmt.Errorf("invalid po payload: %w", err)}
		}
	}
	amt, err := platform.ParseOptionalNumeric(p.TotalAmount)
	if err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid totalAmount: %w", err)}
	}
	if platform.NumericToRat(amt).Sign() <= 0 {
		return nil
	}
	ref := strings.TrimSpace(p.PoNo)
	if ref == "" {
		ref = env.AggregateID
	}
	_, err = x.repo.ConsumePOApproved(ctx, []uuid.UUID{org}, budgetdb.MarkEventProcessedParams{EventID: evID, EventType: env.EventType}, org, sysActor, corr, ref, amt)
	return err
}

func (x *Consumer) handleAPInvoiceRecorded(ctx context.Context, env eventbus.Envelope) error {
	org, err := uuid.Parse(strings.TrimSpace(env.OrganizationID))
	if err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid organizationId: %w", err)}
	}
	evID, err := uuid.Parse(strings.TrimSpace(env.EventID))
	if err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid eventId: %w", err)}
	}
	var p struct {
		PoNo              string `json:"poNo"`
		SourceDocumentRef string `json:"sourceDocumentRef"`
		Reference         string `json:"reference"`
	}
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &p); err != nil {
			return eventbus.ErrPoison{Err: fmt.Errorf("invalid ap invoice payload: %w", err)}
		}
	}
	ref := strings.TrimSpace(p.PoNo)
	if ref == "" {
		ref = strings.TrimSpace(p.SourceDocumentRef)
	}
	if ref == "" {
		ref = strings.TrimSpace(p.Reference)
	}
	_, err = x.repo.ConsumeAPInvoiceRecorded(ctx, []uuid.UUID{org}, budgetdb.MarkEventProcessedParams{EventID: evID, EventType: env.EventType}, org, sysActor, ref)
	return err
}

func (x *Consumer) handleWorkflowApproved(ctx context.Context, env eventbus.Envelope) error {
	org, err := uuid.Parse(strings.TrimSpace(env.OrganizationID))
	if err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid organizationId: %w", err)}
	}
	evID, err := uuid.Parse(strings.TrimSpace(env.EventID))
	if err != nil {
		return eventbus.ErrPoison{Err: fmt.Errorf("invalid eventId: %w", err)}
	}
	corr := evID
	if c, e := uuid.Parse(strings.TrimSpace(env.CorrelationID)); e == nil && c != uuid.Nil {
		corr = c
	}
	var p struct {
		DocumentType   string `json:"documentType"`
		DocumentRef    string `json:"documentRef"`
		Decision       string `json:"decision"`
		Status         string `json:"status"`
		InstanceStatus string `json:"instanceStatus"`
	}
	if len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, &p); err != nil {
			return eventbus.ErrPoison{Err: fmt.Errorf("invalid workflow.approval.completed payload: %w", err)}
		}
	}
	dec := strings.ToLower(strings.TrimSpace(p.Decision))
	st := strings.ToLower(strings.TrimSpace(p.Status))
	ist := strings.ToLower(strings.TrimSpace(p.InstanceStatus))
	if dec != "approve" && dec != "approved" && st != "approved" && ist != "approved" {
		return nil
	}
	docRef := strings.TrimSpace(p.DocumentRef)
	if docRef == "" {
		return nil
	}
	dt := strings.ToUpper(strings.TrimSpace(p.DocumentType))
	refUp := strings.ToUpper(docRef)
	if dt != "BUDGET" && !strings.HasPrefix(refUp, "BUD-") && !strings.HasPrefix(refUp, "BDG-") {
		return nil
	}
	_, err = x.repo.ConsumeWorkflowApproved(
		ctx,
		[]uuid.UUID{org},
		budgetdb.MarkEventProcessedParams{EventID: evID, EventType: env.EventType},
		org,
		sysActor,
		corr,
		docRef,
	)
	return err
}
