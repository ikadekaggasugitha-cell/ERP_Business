package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/sinergiinvestasiproperti/erp/services/procurement-service/internal/procurementdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

type fakeWFStore struct {
	called  bool
	docType string
	docRef  string
}

func (f *fakeWFStore) ConsumeWorkflowApproved(_ context.Context, _ []uuid.UUID, _ procurementdb.MarkEventProcessedParams, docType, docRef string, _, _ uuid.UUID) (bool, error) {
	f.called = true
	f.docType = docType
	f.docRef = docRef
	return true, nil
}

func TestProcurementWorkflowConsumer(t *testing.T) {
	st := &fakeWFStore{}
	c := NewHandler(st)

	// Poison: invalid organizationId
	err := c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "workflow.approval.completed",
		OrganizationID: "bad-org",
	})
	var poison *eventbus.ErrPoison
	if !errors.As(err, &poison) {
		t.Fatalf("expected ErrPoison for bad orgId, got %v", err)
	}

	// Rejected workflow must be ignored
	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "workflow.approval.completed",
		OrganizationID: uuid.NewString(),
		Data:           json.RawMessage(`{"documentType":"PurchaseOrder","documentRef":"PO-2026-01","decision":"Reject","status":"Rejected"}`),
	})
	if err != nil || st.called {
		t.Fatalf("expected rejected workflow to be ignored, err=%v called=%v", err, st.called)
	}

	// Approved PurchaseOrder transitions PO
	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "workflow.approval.completed",
		OrganizationID: uuid.NewString(),
		Data:           json.RawMessage(`{"documentType":"PurchaseOrder","documentRef":"PO-2026-01","decision":"Approve","instanceStatus":"Approved"}`),
	})
	if err != nil || !st.called || st.docType != "PurchaseOrder" || st.docRef != "PO-2026-01" {
		t.Fatalf("expected PO auto-approval, err=%v st=%+v", err, st)
	}

	// Approved PR inferred by prefix
	st.called = false
	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID:        uuid.NewString(),
		EventType:      "workflow.approval.completed",
		OrganizationID: uuid.NewString(),
		Data:           json.RawMessage(`{"documentRef":"PR-2026-09","status":"Approved"}`),
	})
	if err != nil || !st.called || st.docType != "PurchaseRequisition" || st.docRef != "PR-2026-09" {
		t.Fatalf("expected PR auto-approval, err=%v st=%+v", err, st)
	}
}
