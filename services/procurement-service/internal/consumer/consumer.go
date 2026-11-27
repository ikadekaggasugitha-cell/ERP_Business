// Package consumer implements procurement-service's Kafka ingest for
// workflow.approval.completed to auto-transition approved PR/PO documents.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/procurement-service/internal/procurementdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

// Topics lists workflow topics consumed by procurement-service.
var Topics = []string{
	eventbus.TopicFor("workflow.approval.completed"),
}

var sysActor = uuid.MustParse("00000000-0000-0000-0000-0000000000f1")

// Store abstracts procurement repository operations for workflow approval events.
type Store interface {
	ConsumeWorkflowApproved(ctx context.Context, scope []uuid.UUID, mark procurementdb.MarkEventProcessedParams, docType, docRef string, actor, corr uuid.UUID) (bool, error)
}

// Consumer subscribes to workflow.approval.completed for procurement-service.
type Consumer struct {
	repo Store
	c    *eventbus.Consumer
}

// New constructs a procurement workflow Consumer.
func New(repo Store, brokers []string) *Consumer {
	return &Consumer{
		repo: repo,
		c:    eventbus.NewConsumer(brokers, "procurement-service-wf", Topics),
	}
}

// NewHandler constructs a Consumer without Kafka readers for unit tests.
func NewHandler(repo Store) *Consumer { return &Consumer{repo: repo} }

// Run starts consuming events until ctx is cancelled.
func (x *Consumer) Run(ctx context.Context) error { return x.c.Run(ctx, x.Handle) }

// Close releases underlying Kafka reader resources.
func (x *Consumer) Close() error {
	if x.c != nil {
		return x.c.Close()
	}
	return nil
}

// Handle processes a single inbound envelope.
func (x *Consumer) Handle(ctx context.Context, env eventbus.Envelope) error {
	if env.EventType != "workflow.approval.completed" {
		return nil
	}
	orgID, err := uuid.Parse(strings.TrimSpace(env.OrganizationID))
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid organizationId %q: %w", env.OrganizationID, err)}
	}
	evID, err := uuid.Parse(strings.TrimSpace(env.EventID))
	if err != nil {
		return &eventbus.ErrPoison{Err: fmt.Errorf("invalid eventId %q: %w", env.EventID, err)}
	}
	corr := evID
	if c, err := uuid.Parse(strings.TrimSpace(env.CorrelationID)); err == nil && c != uuid.Nil {
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
			return &eventbus.ErrPoison{Err: fmt.Errorf("invalid workflow.approval.completed payload: %w", err)}
		}
	}

	// Only auto-approve when final workflow decision/status is Approved.
	decision := strings.ToLower(strings.TrimSpace(p.Decision))
	status := strings.ToLower(strings.TrimSpace(p.Status))
	instStatus := strings.ToLower(strings.TrimSpace(p.InstanceStatus))
	approved := decision == "approve" || decision == "approved" || status == "approved" || instStatus == "approved"
	if !approved {
		return nil
	}

	docRef := strings.TrimSpace(p.DocumentRef)
	if docRef == "" {
		return nil
	}
	docType := normalizeDocType(p.DocumentType, docRef)
	if docType == "" {
		return nil
	}

	_, err = x.repo.ConsumeWorkflowApproved(
		ctx,
		[]uuid.UUID{orgID},
		procurementdb.MarkEventProcessedParams{EventID: evID, EventType: env.EventType},
		docType,
		docRef,
		sysActor,
		corr,
	)
	return err
}

func normalizeDocType(rawType, docRef string) string {
	t := strings.ToUpper(strings.TrimSpace(rawType))
	switch t {
	case "PURCHASEORDER", "PURCHASE_ORDER", "PO":
		return "PurchaseOrder"
	case "PURCHASEREQUISITION", "PURCHASE_REQUISITION", "PR":
		return "PurchaseRequisition"
	}
	refUp := strings.ToUpper(docRef)
	if strings.HasPrefix(refUp, "PO-") {
		return "PurchaseOrder"
	}
	if strings.HasPrefix(refUp, "PR-") {
		return "PurchaseRequisition"
	}
	return ""
}
