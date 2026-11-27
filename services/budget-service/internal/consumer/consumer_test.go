package consumer

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/sinergiinvestasiproperti/erp/services/budget-service/internal/budgetdb"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
	"github.com/sinergiinvestasiproperti/erp/services/shared/platform"
)

type stubStore struct {
	docRef          string
	amount          pgtype.Numeric
	consumedRef     string
	approvedBudRef  string
	approvedCalled  bool
}

func (s *stubStore) ConsumePOApproved(_ context.Context, _ []uuid.UUID, _ budgetdb.MarkEventProcessedParams, _, _, _ uuid.UUID, docRef string, amount pgtype.Numeric) (bool, error) {
	s.docRef = docRef
	s.amount = amount
	return true, nil
}

func (s *stubStore) ConsumeAPInvoiceRecorded(_ context.Context, _ []uuid.UUID, _ budgetdb.MarkEventProcessedParams, _, _ uuid.UUID, sourceRef string) (bool, error) {
	s.consumedRef = sourceRef
	return true, nil
}

func (s *stubStore) ConsumeWorkflowApproved(_ context.Context, _ []uuid.UUID, _ budgetdb.MarkEventProcessedParams, _, _, _ uuid.UUID, docRef string) (bool, error) {
	s.approvedCalled = true
	s.approvedBudRef = docRef
	return true, nil
}

func TestHandlePOApprovedCommitsBudget(t *testing.T) {
	st := &stubStore{}
	c := NewHandler(st)
	err := c.Handle(context.Background(), eventbus.Envelope{
		EventID: uuid.NewString(), EventType: "procurement.po.approved",
		OrganizationID: uuid.NewString(), AggregateID: uuid.NewString(),
		Data: []byte(`{"poNo":"PO-2026-01","totalAmount":"2500000.0000"}`),
	})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if st.docRef != "PO-2026-01" || platform.FormatNumeric(st.amount, 4) != "2500000.0000" {
		t.Fatalf("unexpected commitment: ref=%q amt=%s", st.docRef, platform.FormatNumeric(st.amount, 4))
	}

	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID: uuid.NewString(), EventType: "ap.invoice.recorded",
		OrganizationID: uuid.NewString(), AggregateID: uuid.NewString(),
		Data: []byte(`{"poNo":"PO-2026-01","totalAmount":"2500000.0000"}`),
	})
	if err != nil {
		t.Fatalf("handle ap.invoice.recorded: %v", err)
	}
	if st.consumedRef != "PO-2026-01" {
		t.Fatalf("expected consumedRef PO-2026-01, got %q", st.consumedRef)
	}

	// Workflow approval for Budget auto-approves budget
	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID: uuid.NewString(), EventType: "workflow.approval.completed",
		OrganizationID: uuid.NewString(), AggregateID: uuid.NewString(),
		Data: []byte(`{"documentType":"Budget","documentRef":"BUD-2026-OPEX","decision":"Approve","instanceStatus":"Approved"}`),
	})
	if err != nil || !st.approvedCalled || st.approvedBudRef != "BUD-2026-OPEX" {
		t.Fatalf("expected budget auto-approval, err=%v called=%v ref=%q", err, st.approvedCalled, st.approvedBudRef)
	}

	// Rejected or non-budget workflow is ignored
	st.approvedCalled = false
	err = c.Handle(context.Background(), eventbus.Envelope{
		EventID: uuid.NewString(), EventType: "workflow.approval.completed",
		OrganizationID: uuid.NewString(), AggregateID: uuid.NewString(),
		Data: []byte(`{"documentType":"Budget","documentRef":"BUD-2026-OPEX","decision":"Reject","instanceStatus":"Rejected"}`),
	})
	if err != nil || st.approvedCalled {
		t.Fatalf("expected rejected budget workflow to be ignored, err=%v called=%v", err, st.approvedCalled)
	}
}
